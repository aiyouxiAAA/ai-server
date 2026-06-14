package main

import (
	"testing"

	"ai-server/internal/battle"
	"ai-server/internal/protocol"
	"ai-server/internal/session"
	"ai-server/internal/team"
	"ai-server/internal/world"
)

func TestHandlePacketClassicTeamInviteAcceptAndTeamChat(t *testing.T) {
	classicTeamManager.Reset()

	store := session.NewStore()
	leaderSession, memberSession := seedClassicTeamPair(t, store)

	inviteResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTeamInviteReq,
		Seq: 2,
		Payload: mustJSON(t, classicTeamInviteRequest{
			TargetName: memberSession.selectedRole.DisplayName,
		}),
	}, leaderSession)
	if !inviteResult.handled || len(inviteResult.teamEvents) != 2 {
		t.Fatalf("expected invite request and result events, got %+v", inviteResult)
	}
	inviteEvent := firstClassicTeamInviteEvent(inviteResult)
	if inviteEvent == nil || inviteEvent.Invite == nil {
		t.Fatalf("expected invite push event, got %+v", inviteResult.teamEvents)
	}
	if inviteEvent.Invite.FromRoleID != leaderSession.selectedRole.RoleID ||
		!classicTeamRecipientsContain(inviteEvent.Recipients, memberSession.selectedRole.RoleID) {
		t.Fatalf("expected invite from leader to member, got %+v", inviteEvent.Invite)
	}

	acceptResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTeamInviteReplyReq,
		Seq: 3,
		Payload: mustJSON(t, classicTeamInviteReplyRequest{
			InviteID: inviteEvent.Invite.InviteID,
			Accept:   true,
		}),
	}, memberSession)
	if !acceptResult.handled || len(acceptResult.teamEvents) != 3 {
		t.Fatalf("expected accepted team snapshots for both members, got %+v", acceptResult)
	}
	if classicTeamMemberEventCount(acceptResult) != 2 {
		t.Fatalf("expected two member snapshots after accept, got %+v", acceptResult.teamEvents)
	}

	chatResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownChatSendReq,
		Seq: 4,
		Payload: mustJSON(t, classicTownChatSendRequest{
			Channel: "team",
			Msg:     "集合",
		}),
	}, leaderSession)
	if !chatResult.handled || len(chatResult.chatMessages) != 0 || len(chatResult.chatBroadcasts) != 1 {
		t.Fatalf("expected team chat broadcast only, got %+v", chatResult)
	}
	broadcast := chatResult.chatBroadcasts[0]
	if broadcast.Message.Channel != "team" || broadcast.Message.Msg != "集合" || broadcast.Message.Handle != leaderSession.selectedRole.RoleID {
		t.Fatalf("expected team chat payload from leader, got %+v", broadcast.Message)
	}
	if !classicTeamRecipientsContain(broadcast.Recipients, leaderSession.selectedRole.RoleID) ||
		!classicTeamRecipientsContain(broadcast.Recipients, memberSession.selectedRole.RoleID) {
		t.Fatalf("expected chat recipients to include both team members, got %+v", broadcast.Recipients)
	}
}

func TestHandlePacketClassicTeamRejectsFifthMember(t *testing.T) {
	classicTeamManager.Reset()

	store := session.NewStore()
	leaderSession, memberSession := seedClassicTeamPair(t, store)
	acceptClassicTeamInvite(t, store, leaderSession, memberSession)

	for _, name := range []string{"队员三", "队员四"} {
		nextSession, _ := seedSelectedRoleSessionInStore(t, store, name)
		acceptClassicTeamInvite(t, store, leaderSession, nextSession)
	}

	fifthSession, _ := seedSelectedRoleSessionInStore(t, store, "队员五")
	classicTeamManager.UpsertOnline(classicTeamMemberFromSession(fifthSession))
	result := buildClassicTeamInviteResult(leaderSession, classicTeamInviteRequest{TargetName: fifthSession.selectedRole.DisplayName})
	if !result.handled || !classicTeamResultContains(result, false, "invite", "队伍人数已满。") {
		t.Fatalf("expected full team warning, got %+v", result)
	}
}

func TestHandlePacketClassicTeamSyncTransferPlan(t *testing.T) {
	classicTeamManager.Reset()

	store := session.NewStore()
	leaderSession, memberSession := seedClassicTeamPair(t, store)
	acceptClassicTeamInvite(t, store, leaderSession, memberSession)
	classicTeamManager.SetSyncChangeMap(leaderSession.selectedRole.RoleID, true)

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownTransferReq,
		Seq: 5,
		Payload: mustJSON(t, classicTownTransferRequest{
			MapID: "430",
			X:     777,
			Y:     555,
		}),
	}, leaderSession)

	if !result.handled || result.townBootstrap == nil {
		t.Fatalf("expected leader transfer bootstrap, got %+v", result)
	}
	if result.teamSyncTransfer == nil || len(result.teamSyncTransfer.Members) != 1 {
		t.Fatalf("expected one synced team member, got %+v", result.teamSyncTransfer)
	}
	if result.teamSyncTransfer.Members[0].RoleID != memberSession.selectedRole.RoleID ||
		result.teamSyncTransfer.TargetMapID != 430 ||
		result.teamSyncTransfer.Spawn.X != 777 ||
		result.teamSyncTransfer.Spawn.Y != 555 {
		t.Fatalf("unexpected sync transfer plan: %+v", result.teamSyncTransfer)
	}
}

func TestHandlePacketClassicTeamResetDungeonClearsLeaderInstance(t *testing.T) {
	classicTeamManager.Reset()

	store := session.NewStore()
	leaderSession, memberSession := seedClassicTeamPair(t, store)
	acceptClassicTeamInvite(t, store, leaderSession, memberSession)
	grantRoleItemTemplateForTest(t, store, leaderSession, "水帘洞通行证", 1)

	transfer := buildClassicTownTransferResult(store, leaderSession, "143", world.SpawnPoint{X: 631, Y: 450})
	if transfer.dungeonInstance == nil || !transfer.dungeonInstance.Active || transfer.dungeonInstance.Key != session.DungeonInstanceShuiliandong {
		t.Fatalf("expected leader to enter shuiliandong, got %+v", transfer.dungeonInstance)
	}
	if _, ok := store.GetRoleDungeonInstance(leaderSession.playerBase.PlayerID, leaderSession.selectedRole.RoleID, session.DungeonInstanceShuiliandong); !ok {
		t.Fatalf("expected shuiliandong instance before reset")
	}

	reset := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTeamResetDungeonReq,
		Seq: 6,
	}, leaderSession)
	if !reset.handled || reset.dungeonInstance == nil || reset.dungeonInstance.Active || reset.dungeonInstance.Key != session.DungeonInstanceShuiliandong {
		t.Fatalf("expected reset to push inactive dungeon instance, got %+v", reset)
	}
	if !classicTeamResultContains(reset, true, "resetDungeon", "") {
		t.Fatalf("expected reset success result, got %+v", reset.teamEvents)
	}
	if _, ok := store.GetRoleDungeonInstance(leaderSession.playerBase.PlayerID, leaderSession.selectedRole.RoleID, session.DungeonInstanceShuiliandong); ok {
		t.Fatalf("expected shuiliandong instance to be cleared")
	}
}

func TestHandlePacketClassicTeamResetDungeonRejectsNonLeader(t *testing.T) {
	classicTeamManager.Reset()

	store := session.NewStore()
	leaderSession, memberSession := seedClassicTeamPair(t, store)
	acceptClassicTeamInvite(t, store, leaderSession, memberSession)
	grantRoleItemTemplateForTest(t, store, memberSession, "水帘洞通行证", 1)

	transfer := buildClassicTownTransferResult(store, memberSession, "143", world.SpawnPoint{X: 631, Y: 450})
	if transfer.dungeonInstance == nil || !transfer.dungeonInstance.Active {
		t.Fatalf("expected member to enter shuiliandong for reset permission test, got %+v", transfer.dungeonInstance)
	}

	reset := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTeamResetDungeonReq,
		Seq: 12,
	}, memberSession)
	if !reset.handled || reset.dungeonInstance != nil || !classicTeamResultContains(reset, false, "resetDungeon", "只有队长可以重置副本。") {
		t.Fatalf("expected non-leader reset rejection, got %+v", reset)
	}
	if _, ok := store.GetRoleDungeonInstance(memberSession.playerBase.PlayerID, memberSession.selectedRole.RoleID, session.DungeonInstanceShuiliandong); !ok {
		t.Fatalf("expected non-leader reset rejection to keep dungeon instance")
	}
}

func TestHandlePacketClassicTeamDungeonSyncRequiresMemberTicket(t *testing.T) {
	classicTeamManager.Reset()
	classicTeamHub = newClassicTeamConnectionHub()

	store := session.NewStore()
	leaderSession, memberSession := seedClassicTeamPair(t, store)
	acceptClassicTeamInvite(t, store, leaderSession, memberSession)
	classicTeamManager.SetSyncChangeMap(leaderSession.selectedRole.RoleID, true)
	classicTeamHub.register(memberSession.selectedRole.RoleID, &websocketWriter{}, memberSession)
	grantRoleItemTemplateForTest(t, store, leaderSession, "水帘洞通行证", 1)

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownTransferReq,
		Seq: 7,
		Payload: mustJSON(t, classicTownTransferRequest{
			MapID: "143",
			X:     631,
			Y:     450,
		}),
	}, leaderSession)

	if !result.handled || result.townBootstrap != nil || result.teamSyncTransfer != nil {
		t.Fatalf("expected dungeon sync to be cancelled before leader transfer, got %+v", result)
	}
	if len(result.chatMessages) != 1 || result.chatMessages[0].Msg == "" {
		t.Fatalf("expected missing member ticket warning, got %+v", result.chatMessages)
	}
	if leaderSession.selectedRole.MapID == 143 {
		t.Fatalf("expected leader to stay outside dungeon when team preflight fails")
	}
}

func TestHandlePacketClassicTeamSharedBattleStartIncludesMember(t *testing.T) {
	classicTeamManager.Reset()
	classicTeamHub = newClassicTeamConnectionHub()

	store := session.NewStore()
	leaderSession, memberSession := seedClassicTeamPair(t, store)
	acceptClassicTeamInvite(t, store, leaderSession, memberSession)
	moveClassicTeamSessionToMap(t, store, leaderSession, 4)
	moveClassicTeamSessionToMap(t, store, memberSession, 4)
	classicTeamManager.UpsertOnline(classicTeamMemberFromSession(leaderSession))
	classicTeamManager.UpsertOnline(classicTeamMemberFromSession(memberSession))
	classicTeamHub.register(memberSession.selectedRole.RoleID, &websocketWriter{}, memberSession)

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattleStartReq,
		Seq: 8,
		Payload: mustJSON(t, battle.StartRequest{
			MapID:       "4",
			MapName:     "云隐村口",
			StageFocusX: 120,
			ReturnRoute: "town-placeholder",
		}),
	}, leaderSession)

	if !result.handled || result.battleStart == nil || result.teamBattleStart == nil {
		t.Fatalf("expected shared team battle start, got %+v", result)
	}
	if len(result.teamBattleStart.Members) != 1 || result.teamBattleStart.Members[0].RoleID != memberSession.selectedRole.RoleID {
		t.Fatalf("expected member in shared battle start, got %+v", result.teamBattleStart.Members)
	}
	teamCells := 0
	for _, cell := range result.battleCells {
		if cell.Camp == battle.CampTeam {
			teamCells += 1
		}
	}
	if teamCells != 2 {
		t.Fatalf("expected two team cells in shared battle, got cells=%+v", result.battleCells)
	}
}

func TestHandlePacketClassicTeamCapturedSecondAccountSharesBattleAppearance(t *testing.T) {
	classicTeamManager.Reset()
	classicTeamHub = newClassicTeamConnectionHub()

	store := session.NewStore()
	leaderSession, _ := seedSelectedRoleSessionInStore(t, store, "恐龙抗狼1")
	memberSession := seedCapturedClassicTeamRoleSessionInStore(t, store, capturedClassicTeamRoleBridgeWoodcutter)
	if leaderSession.playerBase.PlayerID == memberSession.playerBase.PlayerID {
		t.Fatalf("expected captured member to use a separate local account")
	}
	classicTeamManager.UpsertOnline(classicTeamMemberFromSession(leaderSession))
	classicTeamManager.UpsertOnline(classicTeamMemberFromSession(memberSession))
	acceptClassicTeamInvite(t, store, leaderSession, memberSession)

	moveClassicTeamSessionToMap(t, store, leaderSession, 4)
	moveClassicTeamSessionToMap(t, store, memberSession, 4)
	classicTeamManager.UpsertOnline(classicTeamMemberFromSession(leaderSession))
	classicTeamManager.UpsertOnline(classicTeamMemberFromSession(memberSession))
	classicTeamHub.register(memberSession.selectedRole.RoleID, &websocketWriter{}, memberSession)

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattleStartReq,
		Seq: 14,
		Payload: mustJSON(t, battle.StartRequest{
			MapID:       "4",
			MapName:     "云隐村口",
			StageFocusX: 120,
			ReturnRoute: "town-placeholder",
		}),
	}, leaderSession)

	if !result.handled || result.battleStart == nil || result.teamBattleStart == nil {
		t.Fatalf("expected captured second account to join shared battle, got %+v", result)
	}
	capturedCellFound := false
	for _, cell := range result.battleCells {
		if cell.Camp != battle.CampTeam || cell.Name != capturedClassicTeamRoleBridgeWoodcutter.DisplayName {
			continue
		}
		capturedCellFound = true
		if cell.DisplayURL != capturedClassicTeamRoleBridgeWoodcutter.RuntimeSourceQuery {
			t.Fatalf("expected captured source query on team cell, got %q", cell.DisplayURL)
		}
	}
	if !capturedCellFound {
		t.Fatalf("expected shared battle cells to include captured role %s, got %+v", capturedClassicTeamRoleBridgeWoodcutter.DisplayName, result.battleCells)
	}
}

func TestHandlePacketClassicTeamSharedBattleExcludesDifferentMapMember(t *testing.T) {
	classicTeamManager.Reset()
	classicTeamHub = newClassicTeamConnectionHub()

	store := session.NewStore()
	leaderSession, memberSession := seedClassicTeamPair(t, store)
	acceptClassicTeamInvite(t, store, leaderSession, memberSession)
	moveClassicTeamSessionToMap(t, store, leaderSession, 4)
	moveClassicTeamSessionToMap(t, store, memberSession, 5)
	classicTeamManager.UpsertOnline(classicTeamMemberFromSession(leaderSession))
	classicTeamManager.UpsertOnline(classicTeamMemberFromSession(memberSession))
	classicTeamHub.register(memberSession.selectedRole.RoleID, &websocketWriter{}, memberSession)

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattleStartReq,
		Seq: 13,
		Payload: mustJSON(t, battle.StartRequest{
			MapID:       "4",
			MapName:     "云隐村口",
			StageFocusX: 120,
			ReturnRoute: "town-placeholder",
		}),
	}, leaderSession)

	if !result.handled || result.battleStart == nil {
		t.Fatalf("expected leader battle start, got %+v", result)
	}
	if result.teamBattleStart == nil || len(result.teamBattleStart.Members) != 0 {
		t.Fatalf("expected different-map member excluded from shared battle, got %+v", result.teamBattleStart)
	}
	teamCells := 0
	for _, cell := range result.battleCells {
		if cell.Camp == battle.CampTeam {
			teamCells += 1
		}
	}
	if teamCells != 1 {
		t.Fatalf("expected only leader team cell when member is on different map, got cells=%+v", result.battleCells)
	}
}

func TestHandlePacketClassicTeamSharedBattleRejectsNonActiveActorAndAdvances(t *testing.T) {
	classicTeamManager.Reset()

	store := session.NewStore()
	leaderSession, memberSession := seedClassicTeamPair(t, store)
	acceptClassicTeamInvite(t, store, leaderSession, memberSession)
	runtime, bundle, ok := battle.NewTeamWildBattle([]battle.TeamActor{
		{Role: *leaderSession.selectedRole, PlayerBase: *leaderSession.playerBase},
		{Role: *memberSession.selectedRole, PlayerBase: *memberSession.playerBase},
	}, battle.StartRequest{
		MapID:       "4",
		MapName:     "云隐村口",
		StageFocusX: 120,
		ReturnRoute: "town-placeholder",
	})
	if !ok {
		t.Fatal("expected shared battle runtime")
	}
	leaderSession.battleRuntime = runtime
	memberSession.battleRuntime = runtime
	target := firstEnemyCellForTest(t, runtime)

	rejected := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattleActionReq,
		Seq: 9,
		Payload: mustJSON(t, battle.ActionRequest{
			BattleID:     bundle.Start.BattleID,
			ActorHandle:  memberSession.selectedRole.RoleID,
			CommandID:    battle.CommandNormalAttack,
			TargetHandle: target.Handle,
			Round:        bundle.StartCommand.Round,
			Sequence:     bundle.StartCommand.Sequence,
		}),
	}, memberSession)
	if !rejected.handled || len(rejected.battleActions) != 0 || rejected.battleCommand != nil {
		t.Fatalf("expected non-active member action to be rejected without pushes, got %+v", rejected)
	}

	accepted := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattleActionReq,
		Seq: 10,
		Payload: mustJSON(t, battle.ActionRequest{
			BattleID:     bundle.Start.BattleID,
			ActorHandle:  leaderSession.selectedRole.RoleID,
			CommandID:    battle.CommandNormalAttack,
			TargetHandle: target.Handle,
			Round:        bundle.StartCommand.Round,
			Sequence:     bundle.StartCommand.Sequence,
		}),
	}, leaderSession)
	if !accepted.handled || len(accepted.battleActions) == 0 {
		t.Fatalf("expected active leader action to produce battle actions, got %+v", accepted)
	}

	playOver := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattlePlayOverReq,
		Seq: 11,
		Payload: mustJSON(t, battle.PlayOverRequest{
			BattleID: bundle.Start.BattleID,
		}),
	}, leaderSession)
	if !playOver.handled || playOver.battleCommand == nil || playOver.battleCommand.ActorHandle != memberSession.selectedRole.RoleID {
		t.Fatalf("expected playOver to advance command to member, got %+v", playOver)
	}
}

func seedClassicTeamPair(t *testing.T, store *session.Store) (*packetSession, *packetSession) {
	t.Helper()

	leaderSession, _ := seedSelectedRoleSessionInStore(t, store, "队长测试")
	memberSession, _ := seedSelectedRoleSessionInStore(t, store, "队员测试")
	classicTeamManager.UpsertOnline(classicTeamMemberFromSession(leaderSession))
	classicTeamManager.UpsertOnline(classicTeamMemberFromSession(memberSession))
	return leaderSession, memberSession
}

type classicCapturedTeamRoleFixture struct {
	UserName           string
	Password           string
	DisplayName        string
	Gender             string
	RoleTemplateID     int
	PresetID           int
	SourceQuery        string
	RuntimeSourceQuery string
}

var capturedClassicTeamRoleBridgeWoodcutter = classicCapturedTeamRoleFixture{
	UserName:       "capture21432",
	Password:       "local-test-only",
	DisplayName:    "222",
	Gender:         "male",
	RoleTemplateID: 1,
	PresetID:       1,
	// D:/yzhgame/WOCClient/tmp/woc-proxy-captures/20260611_222123_073_session_33544/connections/20260611_222129_098_conn_0002/derived/traffic-preview-0001.log:6
	// createRole player_21432, name=222, vocation=游侠, level=18.
	SourceQuery:        "human/human.swf?a=5&b=7&c=9&e=6&sex=1&h=7&hr=12&co=5&m=0&n=0&p=30&se=6&wr=6&w3=25&",
	RuntimeSourceQuery: "human/human.swf?e=6&sex=1&hr=12&co=5&m=0&n=0&",
}

func seedCapturedClassicTeamRoleSessionInStore(t *testing.T, store *session.Store, fixture classicCapturedTeamRoleFixture) *packetSession {
	t.Helper()

	login := store.Login(session.LoginRequest{
		UserName: fixture.UserName,
		Password: fixture.Password,
	})
	if !login.Success {
		t.Fatalf("expected captured local account login success, got %+v", login)
	}
	create := store.CreateRole(session.RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    fixture.DisplayName,
		Gender:         fixture.Gender,
		RoleTemplateID: fixture.RoleTemplateID,
		PresetID:       fixture.PresetID,
		SourceQuery:    fixture.SourceQuery,
	})
	if !create.Success || create.Role.DisplayName != fixture.DisplayName || create.Role.SourceQuery != fixture.SourceQuery {
		t.Fatalf("expected captured role fixture to create selected role, got %+v", create)
	}

	socketSession := &packetSession{}
	selectResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdRoleSelectRequest,
		Seq: 1,
		Payload: mustJSON(t, session.RoleSelectRequest{
			PlayerID:     login.PlayerID,
			SessionToken: login.SessionToken,
			RoleID:       create.Role.RoleID,
		}),
	}, socketSession)
	if !selectResult.handled || selectResult.townBootstrap == nil || socketSession.selectedRole == nil {
		t.Fatalf("expected captured role select to seed town bootstrap, got %+v", selectResult)
	}
	if socketSession.playerBase.SourceQuery != fixture.RuntimeSourceQuery {
		t.Fatalf("expected captured source query on player base, got %q", socketSession.playerBase.SourceQuery)
	}
	return socketSession
}

func firstEnemyCellForTest(t *testing.T, runtime *battle.Runtime) battle.CellInfoPush {
	t.Helper()
	for _, cell := range runtime.Cells {
		if cell.Camp == battle.CampEnemy {
			return cell
		}
	}
	t.Fatalf("expected enemy cell, got %+v", runtime.Cells)
	return battle.CellInfoPush{}
}

func moveClassicTeamSessionToMap(t *testing.T, store *session.Store, socketSession *packetSession, mapID int) {
	t.Helper()
	role, playerBase, ok := store.UpdateRoleMap(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, mapID)
	if !ok {
		t.Fatalf("expected update role map to %d for %s", mapID, socketSession.selectedRole.RoleID)
	}
	socketSession.selectedRole = &role
	socketSession.playerBase = &playerBase
}

func acceptClassicTeamInvite(t *testing.T, store *session.Store, leaderSession *packetSession, memberSession *packetSession) packetResult {
	t.Helper()

	classicTeamManager.UpsertOnline(classicTeamMemberFromSession(leaderSession))
	classicTeamManager.UpsertOnline(classicTeamMemberFromSession(memberSession))
	inviteResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTeamInviteReq,
		Seq: 2,
		Payload: mustJSON(t, classicTeamInviteRequest{
			TargetName: memberSession.selectedRole.DisplayName,
		}),
	}, leaderSession)
	inviteEvent := firstClassicTeamInviteEvent(inviteResult)
	if inviteEvent == nil || inviteEvent.Invite == nil {
		t.Fatalf("expected invite event, got %+v", inviteResult.teamEvents)
	}
	acceptResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTeamInviteReplyReq,
		Seq: 3,
		Payload: mustJSON(t, classicTeamInviteReplyRequest{
			InviteID: inviteEvent.Invite.InviteID,
			Accept:   true,
		}),
	}, memberSession)
	if !acceptResult.handled {
		t.Fatalf("expected accept handled, got %+v", acceptResult)
	}
	return acceptResult
}

func classicTeamMemberEventCount(result packetResult) int {
	count := 0
	for _, event := range result.teamEvents {
		count += len(event.Members)
	}
	return count
}

func firstClassicTeamInviteEvent(result packetResult) *team.Event {
	for _, event := range result.teamEvents {
		if event.Invite != nil {
			return &event
		}
	}
	return nil
}

func classicTeamResultContains(result packetResult, success bool, action string, errorMessage string) bool {
	for _, event := range result.teamEvents {
		if event.Result != nil &&
			event.Result.Success == success &&
			event.Result.Action == action &&
			event.Result.ErrorMessage == errorMessage {
			return true
		}
	}
	return false
}

func classicTeamRecipientsContain(recipients []string, roleID string) bool {
	for _, recipient := range recipients {
		if recipient == roleID {
			return true
		}
	}
	return false
}
