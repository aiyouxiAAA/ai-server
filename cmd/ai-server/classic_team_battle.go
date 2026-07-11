package main

import (
	"log"
	"strconv"

	"ai-server/internal/battle"
	"ai-server/internal/session"
	"ai-server/internal/team"
)

func (hub *classicTeamConnectionHub) buildBattleActors(actor battle.TeamActor, members []team.Member, mapID string) ([]battle.TeamActor, []team.Member) {
	actors := []battle.TeamActor{actor}
	sharedMembers := make([]team.Member, 0, len(members))
	for _, member := range members {
		connection := hub.connectionFor(member.RoleID)
		if connection.writer == nil || connection.session == nil || connection.session.selectedRole == nil || connection.session.playerBase == nil {
			continue
		}
		memberMapID := connection.session.playerBase.MapID
		if memberMapID <= 0 {
			memberMapID = connection.session.selectedRole.MapID
		}
		if strconv.Itoa(memberMapID) != mapID {
			continue
		}
		actors = append(actors, battle.TeamActor{
			Role:       *connection.session.selectedRole,
			PlayerBase: *connection.session.playerBase,
		})
		sharedMembers = append(sharedMembers, member)
	}
	return actors, sharedMembers
}

func (hub *classicTeamConnectionHub) startSharedBattle(start classicTeamBattleStart) {
	if start.Runtime == nil || len(start.Members) == 0 {
		return
	}
	for _, member := range start.Members {
		connection := hub.connectionFor(member.RoleID)
		if connection.writer == nil || connection.session == nil || connection.session.selectedRole == nil {
			continue
		}
		connection.session.battleRuntime = start.Runtime
		startPush := start.Bundle.Start
		startPush.SelfHandle = member.RoleID
		if err := connection.writer.writePush(cmdClassicBattleStartPush, encodePayload(startPush)); err != nil {
			log.Printf("[ai-server] write classic team battle StartBattle failed roleId=%s: %v", member.RoleID, err)
			continue
		}
		for _, cell := range start.Bundle.Cells {
			if err := connection.writer.writePush(cmdClassicBattleCellInfoPush, encodePayload(cell)); err != nil {
				log.Printf("[ai-server] write classic team battle BattleCellInfo failed roleId=%s: %v", member.RoleID, err)
				break
			}
		}
		command, ok := start.Bundle.StartCommandForActor(member.RoleID)
		if !ok {
			continue
		}
		if err := connection.writer.writePush(cmdClassicBattleStartCommand, encodePayload(command)); err != nil {
			log.Printf("[ai-server] write classic team battle startCommand failed roleId=%s: %v", member.RoleID, err)
		}
	}
}

func (hub *classicTeamConnectionHub) syncSharedBattle(store *session.Store, sync classicTeamBattleSync) {
	recipients, ok := classicTeamManager.RecipientsForTeam(sync.ActorRoleID)
	if !ok {
		return
	}
	for _, roleID := range recipients {
		if roleID == sync.ActorRoleID {
			continue
		}
		connection := hub.connectionFor(roleID)
		if connection.session == nil {
			continue
		}
		beforeMember := classicTeamMemberFromSession(connection.session)
		result := sync.Result
		if connection.session.battleRuntime != nil && connection.session.battleRuntime.HasPendingTeamAction(roleID) {
			result.battleStopCommand = nil
		}
		if result.battleOver != nil {
			memberBattleOver := *result.battleOver
			if connection.session.battleRuntime != nil {
				memberBattleOver.Result = connection.session.battleRuntime.RerollBattleRewardItems(memberBattleOver.Result)
			}
			result.battleOver = &memberBattleOver
			roleState, rolePhysique := finalizeClassicBattleOver(store, connection.session, memberBattleOver.Result)
			connection.session.battleLoot = buildClassicBattleLoot(connection.session, memberBattleOver.Result)
			result.roleState = roleState
			result.rolePhysique = rolePhysique
			result.removeRoleHandles = markDefeatedVisibleMonsterFromBattle(store, connection.session, result.battleOver)
			connection.session.battleRuntime = nil
		} else {
			updatePlayerBaseRoleStateFromBattle(connection.session)
		}
		if connection.writer != nil {
			writeClassicTeamBattleResult(connection.writer, roleID, result)
		}
		hub.broadcast(classicTeamMemberSnapshotEventsIfChanged(beforeMember, connection.session))
	}
}

func classicTeamMemberSnapshotEventsIfChanged(before team.Member, socketSession *packetSession) []team.Event {
	after := classicTeamMemberFromSession(socketSession)
	if after.RoleID == "" || after.Name == "" {
		return nil
	}
	if before.RoleID == after.RoleID &&
		before.Name == after.Name &&
		before.Level == after.Level &&
		before.Vocation == after.Vocation &&
		before.HP == after.HP &&
		before.MaxHP == after.MaxHP &&
		before.MP == after.MP &&
		before.MaxMP == after.MaxMP &&
		before.MapID == after.MapID &&
		before.Online == after.Online {
		return nil
	}
	return classicTeamManager.UpsertOnline(after)
}

func writeClassicTeamBattleResult(writer *websocketWriter, recipientRoleID string, result packetResult) {
	if writer == nil {
		return
	}
	for _, battleBuff := range result.battleBuffs {
		if err := writer.writePush(cmdClassicBattleBuffInfoPush, encodePayload(battleBuff)); err != nil {
			log.Printf("[ai-server] write classic team battle BuffInfo failed: %v", err)
			return
		}
	}
	for _, battleAction := range result.battleActions {
		if err := writer.writePush(cmdClassicBattleActionPush, encodePayload(battleAction)); err != nil {
			log.Printf("[ai-server] write classic team battle battleAction failed: %v", err)
			return
		}
		for _, clearCell := range result.battleClearCells {
			if clearCell.BattleID != battleAction.BattleID || clearCell.Handle != battleAction.ActorHandle {
				continue
			}
			if err := writer.writePush(cmdClassicBattleClearCellInfo, encodePayload(clearCell)); err != nil {
				log.Printf("[ai-server] write classic team battle clearBattleCellInfo failed: %v", err)
				return
			}
		}
		if result.battleStopCommand != nil {
			if err := writer.writePush(cmdClassicBattleStopCommand, encodePayload(*result.battleStopCommand)); err != nil {
				log.Printf("[ai-server] write classic team battle stopCommand failed: %v", err)
				return
			}
		}
	}
	for _, clearBuff := range result.battleClearBuffs {
		if err := writer.writePush(cmdClassicBattleClearBuffInfo, encodePayload(clearBuff)); err != nil {
			log.Printf("[ai-server] write classic team battle clearBuffInfo failed: %v", err)
			return
		}
	}
	if result.battleOver != nil {
		if err := writer.writePush(cmdClassicBattleOverPush, encodePayload(*result.battleOver)); err != nil {
			log.Printf("[ai-server] write classic team battle OverBattle failed: %v", err)
			return
		}
	}
	if result.battleRelive != nil {
		if err := writer.writePush(cmdClassicBattleRelivePush, encodePayload(*result.battleRelive)); err != nil {
			log.Printf("[ai-server] write classic team battle DoRelive failed: %v", err)
			return
		}
	}
	if result.rolePhysique != nil {
		if err := writer.writePush(cmdClassicTownRolePhysiquePush, encodePayload(*result.rolePhysique)); err != nil {
			log.Printf("[ai-server] write classic team battle rolePhysique failed: %v", err)
			return
		}
	}
	if result.roleState != nil {
		if err := writer.writePush(cmdClassicTownRoleStatePush, encodePayload(*result.roleState)); err != nil {
			log.Printf("[ai-server] write classic team battle roleState failed: %v", err)
			return
		}
	}
	for _, handle := range result.removeRoleHandles {
		if err := writer.writePush(cmdClassicTownRemoveRolePush, encodePayload(handle)); err != nil {
			log.Printf("[ai-server] write classic team battle removeRole failed: %v", err)
			return
		}
	}
	if result.battleCommand != nil {
		if err := writer.writePush(cmdClassicBattleStartCommand, encodePayload(*result.battleCommand)); err != nil {
			log.Printf("[ai-server] write classic team battle startCommand failed: %v", err)
			return
		}
	}
	for _, command := range result.battleCommands {
		if command.ActorHandle != recipientRoleID {
			continue
		}
		if err := writer.writePush(cmdClassicBattleStartCommand, encodePayload(command)); err != nil {
			log.Printf("[ai-server] write classic team battle personalized startCommand failed roleId=%s: %v", recipientRoleID, err)
		}
		return
	}
}
