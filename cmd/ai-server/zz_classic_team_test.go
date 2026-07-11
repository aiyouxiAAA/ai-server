package main

import (
	"strings"
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
	if len(inviteResult.chatMessages) != 1 ||
		!strings.Contains(inviteResult.chatMessages[0].Msg, "你已经请求["+memberSession.selectedRole.DisplayName+"]加入队伍") ||
		!strings.Contains(inviteResult.chatMessages[0].Msg, "请等待对方确认") {
		t.Fatalf("expected source invite wait system message, got %+v", inviteResult.chatMessages)
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
	if len(acceptResult.chatBroadcasts) != 1 ||
		acceptResult.chatBroadcasts[0].Message.Msg != "["+memberSession.selectedRole.DisplayName+"]加入队伍" ||
		!classicTeamRecipientsContain(acceptResult.chatBroadcasts[0].Recipients, leaderSession.selectedRole.RoleID) ||
		!classicTeamRecipientsContain(acceptResult.chatBroadcasts[0].Recipients, memberSession.selectedRole.RoleID) {
		t.Fatalf("expected source join team system broadcast, got %+v", acceptResult.chatBroadcasts)
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

func TestHandlePacketClassicTeamInviteDenyUsesCapturedMessage(t *testing.T) {
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
	inviteEvent := firstClassicTeamInviteEvent(inviteResult)
	if inviteEvent == nil || inviteEvent.Invite == nil {
		t.Fatalf("expected invite event, got %+v", inviteResult.teamEvents)
	}

	denyResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTeamInviteReplyReq,
		Seq: 3,
		Payload: mustJSON(t, classicTeamInviteReplyRequest{
			InviteID: inviteEvent.Invite.InviteID,
			Accept:   false,
		}),
	}, memberSession)
	if !denyResult.handled || !classicTeamResultContains(denyResult, false, "inviteRejected", "对方拒绝加入你的队伍") {
		t.Fatalf("expected captured TeamDeny rejection result, got %+v", denyResult.teamEvents)
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
	teamID, ok := classicTeamManager.TeamIDForRole(leaderSession.selectedRole.RoleID)
	if !ok {
		t.Fatal("expected leader team id after reset")
	}
	if _, ok := store.GetTeamDungeonInstance(teamID, session.DungeonInstanceShuiliandong); ok {
		t.Fatal("expected shared shuiliandong instance to be cleared")
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

func TestHandlePacketClassicTeamDungeonSyncAllowsMemberWithoutTicket(t *testing.T) {
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

	if !result.handled || result.townBootstrap == nil || result.teamSyncTransfer == nil {
		t.Fatalf("expected leader ticket to open synced dungeon instance, got %+v", result)
	}
	if len(result.itemClears) != 1 || len(result.itemInfos) != 0 {
		t.Fatalf("expected only leader ticket to be consumed, got clears=%+v infos=%+v", result.itemClears, result.itemInfos)
	}
	memberEntry := buildClassicTownTransferResult(store, memberSession, "143", world.SpawnPoint{X: 631, Y: 450})
	if memberEntry.townBootstrap == nil || len(memberEntry.itemClears) != 0 || len(memberEntry.itemInfos) != 0 {
		t.Fatalf("expected member to reuse active team dungeon without ticket, got %+v", memberEntry)
	}
}

func TestClassicTeamDungeonSharesStateAndDefeatedMonsters(t *testing.T) {
	classicTeamManager.Reset()

	store := session.NewStore()
	leaderSession, memberSession := seedClassicTeamPair(t, store)
	acceptClassicTeamInvite(t, store, leaderSession, memberSession)
	grantRoleItemTemplateForTest(t, store, leaderSession, "水帘洞通行证", 1)

	leaderEntry := buildClassicTownTransferResult(store, leaderSession, "143", world.SpawnPoint{X: 631, Y: 450})
	memberEntry := buildClassicTownTransferResult(store, memberSession, "143", world.SpawnPoint{X: 631, Y: 450})
	if leaderEntry.dungeonInstance == nil || memberEntry.dungeonInstance == nil ||
		leaderEntry.dungeonInstance.CreatedAtUnix != memberEntry.dungeonInstance.CreatedAtUnix {
		t.Fatalf("expected teammates to enter one dungeon instance, leader=%+v member=%+v", leaderEntry.dungeonInstance, memberEntry.dungeonInstance)
	}
	if len(memberEntry.itemClears) != 0 || len(memberEntry.itemInfos) != 0 {
		t.Fatalf("expected teammate to enter shared dungeon without a ticket, got %+v", memberEntry)
	}

	const monsterHandle = "5172206909807859"
	leaderSession.battleRuntime = &battle.Runtime{MapID: "143", SourceMonsterHandle: monsterHandle}
	removed := markDefeatedVisibleMonsterFromBattle(store, leaderSession, &battle.OverPush{
		Winner: battle.CampTeam,
		Result: battle.ResultPayload{Winner: battle.CampTeam},
	})
	if len(removed) != 1 || removed[0] != monsterHandle {
		t.Fatalf("expected team dungeon monster removal, got %+v", removed)
	}

	memberState := syncDungeonInstanceState(store, memberSession, 143)
	if memberState == nil || len(memberState.DefeatedVisibleMonsterHandles) != 1 || memberState.DefeatedVisibleMonsterHandles[0] != monsterHandle {
		t.Fatalf("expected member to receive shared defeated monster state, got %+v", memberState)
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
	if result.battleCommand == nil || result.battleCommand.ActorHandle != leaderSession.selectedRole.RoleID || len(result.battleCommand.Commands) == 0 {
		t.Fatalf("expected leader startCommand to retain its own command definitions, got %+v", result.battleCommand)
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
	acceptResult := acceptClassicTeamInvite(t, store, leaderSession, memberSession)
	assertClassicCapturedTeamMemberSnapshot(t, acceptResult, memberSession.selectedRole.RoleID, capturedClassicTeamRoleBridgeWoodcutter, "45")

	moveClassicTeamSessionToMap(t, store, leaderSession, 4)
	moveClassicTeamSessionToMap(t, store, memberSession, 4)
	applyCapturedClassicTeamRoleFixture(memberSession, capturedClassicTeamRoleBridgeWoodcutter)
	classicTeamManager.UpsertOnline(classicTeamMemberFromSession(leaderSession))
	replayEvents := classicTeamManager.UpsertOnline(classicTeamMemberFromSession(memberSession))
	assertClassicCapturedTeamMemberSnapshot(t, packetResult{teamEvents: replayEvents}, memberSession.selectedRole.RoleID, capturedClassicTeamRoleBridgeWoodcutter, "4")
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
		if cell.DisplayURL != capturedClassicTeamRoleBridgeWoodcutter.BattleSourceQuery ||
			!strings.Contains(cell.DisplayURL, "w3=49") ||
			strings.Contains(cell.DisplayURL, "w3=43") {
			t.Fatalf("expected captured battle source query on team cell, got %q", cell.DisplayURL)
		}
		if cell.Level != capturedClassicTeamRoleBridgeWoodcutter.Level ||
			cell.Vocation != capturedClassicTeamRoleBridgeWoodcutter.Vocation ||
			cell.HP != capturedClassicTeamRoleBridgeWoodcutter.HP ||
			cell.MaxHP != capturedClassicTeamRoleBridgeWoodcutter.MaxHP ||
			cell.MP != capturedClassicTeamRoleBridgeWoodcutter.MP ||
			cell.MaxMP != capturedClassicTeamRoleBridgeWoodcutter.MaxMP {
			t.Fatalf("expected captured battle cell stats from packet fixture, got %+v", cell)
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

func TestHandlePacketClassicTeamSharedBattleAcceptsBothCommandWindowsAndAdvances(t *testing.T) {
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
	leaderCommand, leaderCommandOK := bundle.StartCommandForActor(leaderSession.selectedRole.RoleID)
	memberCommand, memberCommandOK := bundle.StartCommandForActor(memberSession.selectedRole.RoleID)
	if !leaderCommandOK || !memberCommandOK || leaderCommand.Sequence == memberCommand.Sequence {
		t.Fatalf("expected distinct command windows for both team members, got leader=%+v member=%+v", leaderCommand, memberCommand)
	}

	first := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattleActionReq,
		Seq: 9,
		Payload: mustJSON(t, battle.ActionRequest{
			BattleID:     bundle.Start.BattleID,
			ActorHandle:  memberSession.selectedRole.RoleID,
			CommandID:    battle.CommandNormalAttack,
			TargetHandle: target.Handle,
			Round:        memberCommand.Round,
			Sequence:     memberCommand.Sequence,
		}),
	}, memberSession)
	if !first.handled || len(first.battleActions) == 0 || first.battleCommand != nil {
		t.Fatalf("expected first received member action to resolve without waiting, got %+v", first)
	}

	second := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattleActionReq,
		Seq: 10,
		Payload: mustJSON(t, battle.ActionRequest{
			BattleID:     bundle.Start.BattleID,
			ActorHandle:  leaderSession.selectedRole.RoleID,
			CommandID:    battle.CommandNormalAttack,
			TargetHandle: target.Handle,
			Round:        leaderCommand.Round,
			Sequence:     leaderCommand.Sequence,
		}),
	}, leaderSession)
	if !second.handled || len(second.battleActions) == 0 {
		t.Fatalf("expected leader command to remain valid after member action, got %+v", second)
	}

	playOver := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattlePlayOverReq,
		Seq: 11,
		Payload: mustJSON(t, battle.PlayOverRequest{
			BattleID: bundle.Start.BattleID,
		}),
	}, leaderSession)
	if !playOver.handled || playOver.battleCommand != nil || len(playOver.battleCommands) != 2 {
		t.Fatalf("expected playOver to open personalized next-round commands, got %+v", playOver)
	}
	for _, command := range playOver.battleCommands {
		if command.Round != leaderCommand.Round+1 || (command.ActorHandle != leaderSession.selectedRole.RoleID && command.ActorHandle != memberSession.selectedRole.RoleID) {
			t.Fatalf("expected next command for a living team member, got %+v", command)
		}
	}
}

func TestClassicTeamSharedBattleMemberStateRefreshesSnapshot(t *testing.T) {
	classicTeamManager.Reset()

	store := session.NewStore()
	leaderSession, memberSession := seedClassicTeamPair(t, store)
	acceptClassicTeamInvite(t, store, leaderSession, memberSession)
	runtime, _, ok := battle.NewTeamWildBattle([]battle.TeamActor{
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
	memberSession.battleRuntime = runtime
	beforeMember := classicTeamMemberFromSession(memberSession)

	for index := range runtime.Cells {
		if runtime.Cells[index].Handle != memberSession.selectedRole.RoleID {
			continue
		}
		runtime.Cells[index].HP = 123
		runtime.Cells[index].MP = 17
		break
	}
	updatePlayerBaseRoleStateFromBattle(memberSession)
	events := classicTeamMemberSnapshotEventsIfChanged(beforeMember, memberSession)
	assertClassicTeamMemberSnapshot(t, packetResult{teamEvents: events}, memberSession.selectedRole.RoleID, func(member team.Member) bool {
		return member.HP == 123 && member.MP == 17 && member.MaxHP == beforeMember.MaxHP && member.MaxMP == beforeMember.MaxMP
	})

	unchangedEvents := classicTeamMemberSnapshotEventsIfChanged(classicTeamMemberFromSession(memberSession), memberSession)
	if len(unchangedEvents) != 0 {
		t.Fatalf("expected unchanged shared battle member state not to broadcast snapshot, got %+v", unchangedEvents)
	}
}

func TestClassicTeamSharedBattleActiveItemRefreshesActorSnapshot(t *testing.T) {
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
	for index := range runtime.Cells {
		if runtime.Cells[index].Handle != leaderSession.selectedRole.RoleID {
			continue
		}
		runtime.Cells[index].HP = 40
		break
	}
	updatePlayerBaseRoleStateFromBattle(leaderSession)
	classicTeamManager.UpsertOnline(classicTeamMemberFromSession(leaderSession))

	item, ok := store.GrantRoleItem(leaderSession.playerBase.PlayerID, leaderSession.selectedRole.RoleID, session.RoleItem{
		Type:        "背包",
		Name:        "L花卷",
		ItemType:    "own",
		Display:     "213.png",
		Description: "f_i_L花卷^ffffff&24@消耗品&25@99&7@35&20@恢复气力",
		Count:       2,
		Index:       20,
		ItemLevel:   1,
	})
	if !ok {
		t.Fatal("expected test item to be granted")
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattleActiveItemReq,
		Seq: 22,
		Payload: mustJSON(t, battle.ItemActionRequest{
			BattleID:    bundle.Start.BattleID,
			ActorHandle: leaderSession.selectedRole.RoleID,
			Type:        item.Type,
			Index:       item.Index,
			Round:       bundle.StartCommand.Round,
			Sequence:    bundle.StartCommand.Sequence,
		}),
	}, leaderSession)
	if !result.handled || result.teamBattleSync == nil {
		t.Fatalf("expected shared ActiveItem to produce team sync, got %+v", result)
	}
	leaderCell := classicTeamCellForRole(t, runtime, leaderSession.selectedRole.RoleID)
	assertClassicTeamMemberSnapshot(t, result, leaderSession.selectedRole.RoleID, func(member team.Member) bool {
		return member.HP == leaderCell.HP && member.MP == leaderCell.MP
	})
}

func TestClassicTeamSharedBattleOverGrantsRewardAndLootToEachOnlineMember(t *testing.T) {
	classicTeamManager.Reset()
	classicTeamHub = newClassicTeamConnectionHub()

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
	classicTeamHub.mu.Lock()
	classicTeamHub.connections[memberSession.selectedRole.RoleID] = classicTeamConnection{session: memberSession}
	classicTeamHub.mu.Unlock()
	runtime.PendingOver = &battle.OverPush{
		BattleID: bundle.Start.BattleID,
		Winner:   battle.CampTeam,
		Rounds:   1,
		Result: battle.ResultPayload{
			Winner:   battle.CampTeam,
			Rounds:   1,
			ExpDelta: 37,
			Items:    []string{"朽木x1"},
		},
	}

	playOver := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattlePlayOverReq,
		Seq: 21,
		Payload: mustJSON(t, battle.PlayOverRequest{
			BattleID: bundle.Start.BattleID,
		}),
	}, leaderSession)
	if !playOver.handled || playOver.battleOver == nil || playOver.teamBattleSync == nil {
		t.Fatalf("expected shared BattlePlayOver to produce team over sync, got %+v", playOver)
	}
	classicTeamHub.syncSharedBattle(store, *playOver.teamBattleSync)

	assertClassicTeamSharedBattleRewardState(t, leaderSession, "leader")
	assertClassicTeamSharedBattleRewardState(t, memberSession, "member")
	memberLootList := buildClassicTownItemListResult(store, memberSession, classicBattleLootType)
	if !memberLootList.handled || len(memberLootList.itemInfos) != 1 || memberLootList.itemInfos[0].Name != "朽木" {
		t.Fatalf("expected member battle loot container to list shared reward, got %+v", memberLootList)
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
	Level              int
	Vocation           string
	HP                 int
	MaxHP              int
	MP                 int
	MaxMP              int
	MapID              int
	SourceQuery        string
	RuntimeSourceQuery string
	BattleSourceQuery  string
}

var capturedClassicTeamRoleBridgeWoodcutter = classicCapturedTeamRoleFixture{
	UserName:       "capture21432",
	Password:       "local-test-only",
	DisplayName:    "222",
	Gender:         "male",
	RoleTemplateID: 1,
	PresetID:       1,
	// D:/yzhgame/WOCClient/tmp/woc-proxy-captures/20260619_190155_492_session_02408/connections/20260619_190201_802_conn_0002
	// raw/server-to-client-0001.bin packets #21781/#21795.
	// The local account keeps the user-facing test name 222, but carries the captured player_21432 body/state.
	Level:              36,
	Vocation:           "游侠",
	HP:                 1165,
	MaxHP:              1165,
	MP:                 489,
	MaxMP:              489,
	MapID:              45,
	SourceQuery:        "human/human.swf?a=29&b=14&c=17&e=6&sex=1&h=30&hr=12&co=5&m=0&n=0&p=16&se=12&wr=25&w3=49&",
	RuntimeSourceQuery: "human/human.swf?e=6&sex=1&hr=12&co=5&m=0&n=0&h=30&a=29&wr=25&w3=49&c=17&p=16&b=14&se=12&",
	BattleSourceQuery:  "human/human.swf?e=6&sex=1&hr=12&co=5&m=0&n=0&h=30&a=29&wr=25&w3=49&c=17&p=16&b=14&se=12&",
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
	applyCapturedClassicTeamRoleFixture(socketSession, fixture)
	socketSession.selectedRole.MapID = fixture.MapID
	socketSession.playerBase.MapID = fixture.MapID
	return socketSession
}

func applyCapturedClassicTeamRoleFixture(socketSession *packetSession, fixture classicCapturedTeamRoleFixture) {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		return
	}
	socketSession.selectedRole.Level = fixture.Level
	socketSession.selectedRole.Voc = fixture.Vocation
	if socketSession.selectedRole.MapID <= 0 {
		socketSession.selectedRole.MapID = fixture.MapID
	}
	socketSession.selectedRole.SourceQuery = fixture.RuntimeSourceQuery
	socketSession.selectedRole.BattleSourceQuery = fixture.BattleSourceQuery
	if socketSession.selectedRole.RoleState == nil {
		socketSession.selectedRole.RoleState = &session.RoleState{Handle: socketSession.selectedRole.RoleID}
	}
	socketSession.selectedRole.RoleState.HP = fixture.HP
	socketSession.selectedRole.RoleState.MP = fixture.MP
	socketSession.selectedRole.RoleState.Lv = fixture.Level
	if socketSession.selectedRole.RoleState.Speed == 0 {
		socketSession.selectedRole.RoleState.Speed = 145
	}
	if socketSession.selectedRole.RolePhysique == nil {
		socketSession.selectedRole.RolePhysique = &session.RolePhysique{Handle: socketSession.selectedRole.RoleID}
	}
	socketSession.selectedRole.RolePhysique.MaxHP = fixture.MaxHP
	socketSession.selectedRole.RolePhysique.MaxMP = fixture.MaxMP

	socketSession.playerBase.Level = fixture.Level
	socketSession.playerBase.Voc = fixture.Vocation
	socketSession.playerBase.HP = fixture.HP
	socketSession.playerBase.MaxHP = fixture.MaxHP
	socketSession.playerBase.MP = fixture.MP
	socketSession.playerBase.MaxMP = fixture.MaxMP
	if socketSession.playerBase.MapID <= 0 {
		socketSession.playerBase.MapID = fixture.MapID
	}
	socketSession.playerBase.SourceQuery = fixture.RuntimeSourceQuery
	socketSession.playerBase.BattleSourceQuery = fixture.BattleSourceQuery
	if socketSession.playerBase.RoleState == nil {
		socketSession.playerBase.RoleState = &session.RoleState{Handle: socketSession.selectedRole.RoleID}
	}
	socketSession.playerBase.RoleState.HP = fixture.HP
	socketSession.playerBase.RoleState.MP = fixture.MP
	socketSession.playerBase.RoleState.Lv = fixture.Level
	if socketSession.playerBase.RoleState.Speed == 0 {
		socketSession.playerBase.RoleState.Speed = 145
	}
	if socketSession.playerBase.RolePhysique == nil {
		socketSession.playerBase.RolePhysique = &session.RolePhysique{Handle: socketSession.selectedRole.RoleID}
	}
	socketSession.playerBase.RolePhysique.MaxHP = fixture.MaxHP
	socketSession.playerBase.RolePhysique.MaxMP = fixture.MaxMP
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

func classicTeamCellForRole(t *testing.T, runtime *battle.Runtime, roleID string) battle.CellInfoPush {
	t.Helper()
	for _, cell := range runtime.Cells {
		if cell.Handle == roleID {
			return cell
		}
	}
	t.Fatalf("expected team cell for roleId=%s, got %+v", roleID, runtime.Cells)
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

func assertClassicCapturedTeamMemberSnapshot(t *testing.T, result packetResult, roleID string, fixture classicCapturedTeamRoleFixture, mapID string) {
	t.Helper()
	assertClassicTeamMemberSnapshot(t, result, roleID, func(member team.Member) bool {
		return member.Name == fixture.DisplayName &&
			member.Level == fixture.Level &&
			member.Vocation == fixture.Vocation &&
			member.HP == fixture.HP &&
			member.MaxHP == fixture.MaxHP &&
			member.MP == fixture.MP &&
			member.MaxMP == fixture.MaxMP &&
			member.MapID == mapID &&
			member.Online
	})
}

func assertClassicTeamMemberSnapshot(t *testing.T, result packetResult, roleID string, predicate func(team.Member) bool) {
	t.Helper()
	for _, event := range result.teamEvents {
		for _, member := range event.Members {
			if member.RoleID != roleID {
				continue
			}
			if !predicate(member) {
				t.Fatalf("unexpected team member snapshot for roleId=%s: %+v", roleID, member)
			}
			return
		}
	}
	t.Fatalf("expected team member snapshot for roleId=%s, got %+v", roleID, result.teamEvents)
}

func assertClassicTeamSharedBattleRewardState(t *testing.T, socketSession *packetSession, label string) {
	t.Helper()
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		t.Fatalf("expected %s session to stay selected after shared battle", label)
	}
	if socketSession.selectedRole.Exp != 37 || socketSession.playerBase.Exp != 37 {
		t.Fatalf("expected %s shared battle exp reward 37, got role=%+v playerBase=%+v", label, socketSession.selectedRole, socketSession.playerBase)
	}
	if len(socketSession.battleLoot) != 1 || socketSession.battleLoot[0].Name != "朽木" || socketSession.battleLoot[0].Type != classicBattleLootType {
		t.Fatalf("expected %s shared battle loot in battle container, got %+v", label, socketSession.battleLoot)
	}
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
