package main

import (
	"log"
	"strconv"
	"sync"

	"ai-server/internal/session"
	"ai-server/internal/team"
	"ai-server/internal/world"
)

var classicTeamManager = team.NewManager()
var classicTeamHub = newClassicTeamConnectionHub()

type classicTeamConnectionHub struct {
	mu          sync.Mutex
	connections map[string]classicTeamConnection
}

type classicTeamConnection struct {
	writer  *websocketWriter
	session *packetSession
}

func newClassicTeamConnectionHub() *classicTeamConnectionHub {
	return &classicTeamConnectionHub{
		connections: make(map[string]classicTeamConnection),
	}
}

func (hub *classicTeamConnectionHub) register(roleID string, writer *websocketWriter, socketSession ...*packetSession) {
	if roleID == "" || writer == nil {
		return
	}
	var registeredSession *packetSession
	if len(socketSession) > 0 {
		registeredSession = socketSession[0]
	}
	hub.mu.Lock()
	defer hub.mu.Unlock()
	hub.connections[roleID] = classicTeamConnection{
		writer:  writer,
		session: registeredSession,
	}
}

func (hub *classicTeamConnectionHub) unregister(roleID string) {
	if roleID == "" {
		return
	}
	hub.mu.Lock()
	defer hub.mu.Unlock()
	delete(hub.connections, roleID)
}

func (hub *classicTeamConnectionHub) broadcast(events []team.Event) {
	for _, event := range events {
		for _, roleID := range event.Recipients {
			writer := hub.writerFor(roleID)
			if writer == nil {
				continue
			}
			if err := writeClassicTeamEvent(writer, event); err != nil {
				log.Printf("[ai-server] write classic team event failed roleId=%s: %v", roleID, err)
			}
		}
	}
}

func (hub *classicTeamConnectionHub) broadcastChat(recipients []string, message classicTownChatMessagePush) {
	for _, roleID := range recipients {
		writer := hub.writerFor(roleID)
		if writer == nil {
			continue
		}
		if err := writer.writePush(cmdClassicTownChatMessagePush, encodePayload(message)); err != nil {
			log.Printf("[ai-server] write classic team chat failed roleId=%s: %v", roleID, err)
		}
	}
}

func (hub *classicTeamConnectionHub) writerFor(roleID string) *websocketWriter {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	return hub.connections[roleID].writer
}

func (hub *classicTeamConnectionHub) connectionFor(roleID string) classicTeamConnection {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	return hub.connections[roleID]
}

func (hub *classicTeamConnectionHub) preflightDungeonSyncTransfer(store *session.Store, members []team.Member, targetMapID int) (string, bool) {
	if store == nil {
		return "队伍同步状态异常。", false
	}
	if _, ok := world.DungeonInstanceKeyForMapID(targetMapID); !ok {
		return "", true
	}
	for _, member := range members {
		connection := hub.connectionFor(member.RoleID)
		if connection.writer == nil || connection.session == nil {
			return "队员【" + member.Name + "】连接状态异常，队伍同步取消。", false
		}
		if warningMessage, ok := checkDungeonEntryTicketIfNeeded(store, connection.session, targetMapID); !ok {
			return warningMessage, false
		}
	}
	return "", true
}

func (hub *classicTeamConnectionHub) syncTransfer(store *session.Store, transfer classicTeamSyncTransfer) {
	if store == nil || transfer.TargetMapID <= 0 {
		return
	}
	for _, member := range transfer.Members {
		connection := hub.connectionFor(member.RoleID)
		if connection.writer == nil || connection.session == nil || connection.session.selectedRole == nil || connection.session.playerBase == nil {
			continue
		}
		entryResult, ok := consumeDungeonEntryTicketIfNeeded(store, connection.session, transfer.TargetMapID)
		if !ok {
			for _, chatMessage := range entryResult.chatMessages {
				if err := connection.writer.writePush(cmdClassicTownChatMessagePush, encodePayload(chatMessage)); err != nil {
					log.Printf("[ai-server] write classic team sync transfer warning failed roleId=%s: %v", member.RoleID, err)
				}
			}
			continue
		}
		writeClassicTeamTransferSideEffects(connection.writer, entryResult)
		role, playerBase, ok := store.UpdateRoleMap(connection.session.playerBase.PlayerID, member.RoleID, transfer.TargetMapID)
		if !ok {
			log.Printf("[ai-server] classic team sync transfer skipped missing role roleId=%s targetMapId=%d", member.RoleID, transfer.TargetMapID)
			continue
		}
		connection.session.selectedRole = &role
		connection.session.playerBase = &playerBase
		dungeonInstance := syncDungeonInstanceState(store, connection.session, transfer.TargetMapID)
		bootstrap, ok := world.BuildTownTransferBootstrap(role, playerBase, transfer.TargetMapID, transfer.Spawn)
		if !ok {
			continue
		}
		filterDefeatedVisibleMonsters(&bootstrap, connection.session)
		connection.writer.writeClassicTownBootstrap(bootstrap)
		if dungeonInstance != nil {
			if err := connection.writer.writePush(cmdClassicTownDungeonInstance, encodePayload(*dungeonInstance)); err != nil {
				log.Printf("[ai-server] write classic team sync dungeon instance failed roleId=%s: %v", member.RoleID, err)
			}
		}
		hub.broadcast(classicTeamManager.UpsertOnline(classicTeamMemberFromSession(connection.session)))
		log.Printf("[ai-server] classic team sync transfer actorRoleId=%s memberRoleId=%s fromMapId=%d targetMapId=%d", transfer.ActorRoleID, member.RoleID, transfer.FromMapID, transfer.TargetMapID)
	}
}

func (hub *classicTeamConnectionHub) resetDungeonInstances(store *session.Store, reset classicTeamDungeonReset) {
	if store == nil || reset.InstanceKey == "" || reset.MapID <= 0 {
		return
	}
	for _, member := range reset.Members {
		connection := hub.connectionFor(member.RoleID)
		if connection.writer == nil || connection.session == nil || connection.session.selectedRole == nil || connection.session.playerBase == nil {
			continue
		}
		currentMapID := connection.session.playerBase.MapID
		if currentMapID <= 0 {
			currentMapID = connection.session.selectedRole.MapID
		}
		currentKey, ok := world.DungeonInstanceKeyForMapID(currentMapID)
		if !ok || currentKey != reset.InstanceKey {
			continue
		}
		if !store.ResetRoleDungeonInstance(connection.session.playerBase.PlayerID, member.RoleID, reset.InstanceKey) {
			continue
		}
		setDefeatedVisibleMonsterHandles(connection.session, nil)
		if err := connection.writer.writePush(cmdClassicTownDungeonInstance, encodePayload(*inactiveDungeonInstancePush(reset.InstanceKey, reset.MapID))); err != nil {
			log.Printf("[ai-server] write classic team reset dungeon failed roleId=%s: %v", member.RoleID, err)
		}
		log.Printf("[ai-server] classic team reset dungeon actorRoleId=%s memberRoleId=%s key=%s", reset.ActorRoleID, member.RoleID, reset.InstanceKey)
	}
}

func writeClassicTeamTransferSideEffects(writer *websocketWriter, result packetResult) {
	if writer == nil {
		return
	}
	for _, itemClear := range result.itemClears {
		if err := writer.writePush(cmdClassicTownItemInfoClear, encodePayload(itemClear)); err != nil {
			log.Printf("[ai-server] write classic team sync item clear failed: %v", err)
		}
	}
	for _, itemInfo := range result.itemInfos {
		if err := writer.writePush(cmdClassicTownItemInfoPush, encodePayload(itemInfo)); err != nil {
			log.Printf("[ai-server] write classic team sync item info failed: %v", err)
		}
	}
}

func writeClassicTeamEvents(writer *websocketWriter, events []team.Event) error {
	for _, event := range events {
		if err := writeClassicTeamEvent(writer, event); err != nil {
			return err
		}
	}
	return nil
}

func writeClassicTeamEvent(writer *websocketWriter, event team.Event) error {
	if event.Invite != nil {
		if err := writer.writePush(cmdClassicTeamInvitePush, encodePayload(*event.Invite)); err != nil {
			return err
		}
	}
	if event.Info != nil {
		if err := writer.writePush(cmdClassicTeamInfoPush, encodePayload(*event.Info)); err != nil {
			return err
		}
	}
	for _, member := range event.Members {
		if err := writer.writePush(cmdClassicTeamMemberPush, encodePayload(member)); err != nil {
			return err
		}
	}
	if event.MemberClear != nil {
		if err := writer.writePush(cmdClassicTeamMemberClearPush, encodePayload(*event.MemberClear)); err != nil {
			return err
		}
	}
	if event.Clear != nil {
		if err := writer.writePush(cmdClassicTeamClearPush, encodePayload(*event.Clear)); err != nil {
			return err
		}
	}
	if event.Result != nil {
		if err := writer.writePush(cmdClassicTeamResultPush, encodePayload(*event.Result)); err != nil {
			return err
		}
	}
	return nil
}

func classicTeamMemberFromSession(socketSession *packetSession) team.Member {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		return team.Member{}
	}
	role := socketSession.selectedRole
	playerBase := socketSession.playerBase
	hp := playerBase.HP
	mp := playerBase.MP
	maxHP := playerBase.MaxHP
	maxMP := playerBase.MaxMP
	if playerBase.RoleState != nil {
		if hp <= 0 {
			hp = playerBase.RoleState.HP
		}
		if mp <= 0 {
			mp = playerBase.RoleState.MP
		}
	}
	if playerBase.RolePhysique != nil {
		if maxHP <= 0 {
			maxHP = playerBase.RolePhysique.MaxHP
		}
		if maxMP <= 0 {
			maxMP = playerBase.RolePhysique.MaxMP
		}
	}
	if hp <= 0 && role.RoleState != nil {
		hp = role.RoleState.HP
	}
	if mp <= 0 && role.RoleState != nil {
		mp = role.RoleState.MP
	}
	if maxHP <= 0 && role.RolePhysique != nil {
		maxHP = role.RolePhysique.MaxHP
	}
	if maxMP <= 0 && role.RolePhysique != nil {
		maxMP = role.RolePhysique.MaxMP
	}
	return team.Member{
		RoleID:   role.RoleID,
		Name:     role.DisplayName,
		Level:    role.Level,
		Vocation: role.Voc,
		HP:       hp,
		MaxHP:    maxHP,
		MP:       mp,
		MaxMP:    maxMP,
		MapID:    strconv.Itoa(playerBase.MapID),
		Online:   true,
	}
}
