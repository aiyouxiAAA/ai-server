package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"ai-server/internal/battle"
	"ai-server/internal/classicactivity"
	"ai-server/internal/protocol"
	"ai-server/internal/quest"
	"ai-server/internal/session"
	"ai-server/internal/world"
)

func TestHandlePacketLoginSuccess(t *testing.T) {
	store := session.NewStore()
	request := session.LoginRequest{
		UserName: "mockuser",
		Password: "magicpwd",
	}

	result := handlePacket(store, protocol.Packet{
		Cmd:     cmdAuthLoginRequest,
		Seq:     1,
		Payload: mustJSON(t, request),
	})

	if !result.handled {
		t.Fatal("expected login packet to be handled")
	}
	if result.responseCmd != cmdAuthLoginResponse {
		t.Fatalf("expected response cmd %d, got %d", cmdAuthLoginResponse, result.responseCmd)
	}

	var response session.LoginResponse
	decodeJSON(t, result.responsePayload, &response)
	if !response.Success {
		t.Fatalf("expected login success, got failure: %+v", response)
	}
	if response.PlayerID != "mock-player-001" {
		t.Fatalf("expected player id mock-player-001, got %q", response.PlayerID)
	}
}

func TestHandlePacketHeartbeatIsOneWayKeepAlive(t *testing.T) {
	store := session.NewStore()

	result := handlePacket(store, protocol.Packet{
		Cmd: cmdHeartbeat,
		Seq: 1,
	})

	if !result.handled {
		t.Fatal("expected heartbeat packet to be handled")
	}
	if result.responseCmd != 0 {
		t.Fatalf("expected heartbeat to be one-way without response, got response cmd %d", result.responseCmd)
	}
	if len(result.responsePayload) != 0 {
		t.Fatalf("expected heartbeat response payload to be empty, got %d bytes", len(result.responsePayload))
	}
}

func TestHandlePacketRoleSelectPushesTownRoleStats(t *testing.T) {
	store := session.NewStore()
	login := store.Login(session.LoginRequest{
		UserName: "mockuser",
		Password: "magicpwd",
	})
	create := store.CreateRole(session.RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "状态女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	socketSession := &packetSession{}
	beforeSelect := time.Now().Add(-time.Second).UnixMilli()

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdRoleSelectRequest,
		Seq: 2,
		Payload: mustJSON(t, session.RoleSelectRequest{
			PlayerID:     login.PlayerID,
			SessionToken: login.SessionToken,
			RoleID:       create.Role.RoleID,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected role select to be handled")
	}
	if result.responseCmd != cmdRoleSelectResponse {
		t.Fatalf("expected role select response cmd %d, got %d", cmdRoleSelectResponse, result.responseCmd)
	}

	var response session.RoleSelectResponse
	decodeJSON(t, result.responsePayload, &response)
	if !response.Success {
		t.Fatalf("expected role select success, got %+v", response)
	}
	if response.PlayerBase.RoleState == nil {
		t.Fatalf("expected playerBase roleState, got %+v", response.PlayerBase)
	}
	if response.Role.Voc != "新手" || response.PlayerBase.Voc != "新手" {
		t.Fatalf("expected default vocation 新手, got role=%q playerBase=%q", response.Role.Voc, response.PlayerBase.Voc)
	}
	if response.PlayerBase.RolePhysique == nil {
		t.Fatalf("expected playerBase rolePhysique, got %+v", response.PlayerBase)
	}
	if response.PlayerBase.RoleState.Handle != create.Role.RoleID ||
		response.PlayerBase.RoleState.HP != 185 ||
		response.PlayerBase.RoleState.MP != 44 ||
		response.PlayerBase.RoleState.Lv != 1 ||
		response.PlayerBase.RoleState.Speed != 130 {
		t.Fatalf("expected source-style roleState defaults, got %+v", response.PlayerBase.RoleState)
	}
	if response.PlayerBase.RolePhysique.Handle != create.Role.RoleID ||
		response.PlayerBase.RolePhysique.MaxHP != 185 ||
		response.PlayerBase.RolePhysique.MaxMP != 44 ||
		response.PlayerBase.RolePhysique.PhyAtk != 10 ||
		response.PlayerBase.RolePhysique.Hit != 100 {
		t.Fatalf("expected source-style rolePhysique defaults, got %+v", response.PlayerBase.RolePhysique)
	}
	if result.townBootstrap == nil {
		t.Fatal("expected role select to produce town bootstrap")
	}
	if result.townBootstrap.RoleState == nil || result.townBootstrap.RoleState.Handle != create.Role.RoleID {
		t.Fatalf("expected town bootstrap roleState for selected role, got %+v", result.townBootstrap.RoleState)
	}
	if result.townBootstrap.RolePhysique == nil || result.townBootstrap.RolePhysique.Handle != create.Role.RoleID {
		t.Fatalf("expected town bootstrap rolePhysique for selected role, got %+v", result.townBootstrap.RolePhysique)
	}
	if result.serverTime == nil {
		t.Fatal("expected role select to push classic town serverTime")
	}
	afterSelect := time.Now().Add(time.Second).UnixMilli()
	if result.serverTime.ServerTime < beforeSelect || result.serverTime.ServerTime > afterSelect {
		t.Fatalf("expected serverTime near role select time, got %+v", result.serverTime)
	}
	if !strings.Contains(result.serverTime.SourceCapture, "c_serverTime") {
		t.Fatalf("expected serverTime source capture to cite c_serverTime, got %+v", result.serverTime)
	}
	if len(result.gameTips) != 1 {
		t.Fatalf("expected role select on map1 to push captured npc gameTip, got %+v", result.gameTips)
	}
	gameTip := result.gameTips[0]
	if gameTip.TipID != "zzshow" || gameTip.Kind != "npc" || gameTip.TargetName != "一心长态" {
		t.Fatalf("expected captured npc gameTip identity, got %+v", gameTip)
	}
	if !strings.Contains(gameTip.HTMLText, "研习职业") || !strings.Contains(gameTip.SourceCapture, "c_gameTip") {
		t.Fatalf("expected captured npc gameTip source payload, got %+v", gameTip)
	}
	if len(gameTip.Fields) != 4 || gameTip.Fields[0] != gameTip.TipID || gameTip.Fields[1] != gameTip.Kind || gameTip.Fields[2] != gameTip.TargetName || gameTip.Fields[3] != gameTip.HTMLText {
		t.Fatalf("expected raw c_gameTip fields to mirror structured payload, got %+v", gameTip.Fields)
	}
	if tips := buildClassicTownRoleSelectGameTips(2, *result.townBootstrap); len(tips) != 0 {
		t.Fatalf("expected non-map1 role select to skip map1 npc gameTip, got %+v", tips)
	}
	bootstrapWithoutTipTarget := *result.townBootstrap
	bootstrapWithoutTipTarget.CreateRoles = nil
	if tips := buildClassicTownRoleSelectGameTips(1, bootstrapWithoutTipTarget); len(tips) != 0 {
		t.Fatalf("expected missing target npc to skip gameTip, got %+v", tips)
	}
	if socketSession.playerBase == nil || socketSession.playerBase.RoleState == nil || socketSession.playerBase.RolePhysique == nil {
		t.Fatalf("expected socket session to retain role stats, got %+v", socketSession.playerBase)
	}
}

func TestHandlePacketClassicTownVocationAnswerChangesRoleVocation(t *testing.T) {
	store := session.NewStore()
	login := store.Login(session.LoginRequest{
		UserName: "mockuser",
		Password: "magicpwd",
	})
	create := store.CreateRole(session.RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "转职女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	socketSession := &packetSession{}
	handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdRoleSelectRequest,
		Seq: 2,
		Payload: mustJSON(t, session.RoleSelectRequest{
			PlayerID:     login.PlayerID,
			SessionToken: login.SessionToken,
			RoleID:       create.Role.RoleID,
		}),
	}, socketSession)

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownAnswerReq,
		Seq: 3,
		Payload: mustJSON(t, classicTownAnswerRequest{
			Handle:       sourceSkillTeacherHandle,
			MsgHandle:    "2",
			AnswerHandle: "job_warrior",
		}),
	}, socketSession)

	if !result.handled || result.answerSpeak == nil || result.answerSpeak.MsgHandle != "job_warrior" {
		t.Fatalf("expected vocation answer result, got %+v", result.answerSpeak)
	}
	if socketSession.selectedRole == nil || socketSession.selectedRole.Voc != "战士" || socketSession.playerBase == nil || socketSession.playerBase.Voc != "战士" {
		t.Fatalf("expected socket session vocation 战士, role=%+v playerBase=%+v", socketSession.selectedRole, socketSession.playerBase)
	}
	if result.createPlayer == nil || result.createPlayer.Vocation != "战士" {
		t.Fatalf("expected createPlayer vocation 战士, got %+v", result.createPlayer)
	}
	if result.roleState == nil || result.rolePhysique == nil {
		t.Fatalf("expected vocation result to push role state and physique, got state=%+v physique=%+v", result.roleState, result.rolePhysique)
	}
	if len(result.chatMessages) != 1 || result.chatMessages[0].Channel != "system" || !strings.Contains(result.chatMessages[0].Msg, "成功转职为【战士】") {
		t.Fatalf("expected source sayS vocation message, got %+v", result.chatMessages)
	}
	role, playerBase, ok := store.GetRoleRuntimeData(login.PlayerID, create.Role.RoleID)
	if !ok || role.Voc != "战士" || playerBase.Voc != "战士" {
		t.Fatalf("expected persisted runtime vocation 战士, ok=%v role=%+v base=%+v", ok, role, playerBase)
	}
}

func TestHandlePacketClassicTownTargetRoleNoResponse(t *testing.T) {
	store := session.NewStore()
	result := handlePacket(store, protocol.Packet{
		Cmd: cmdClassicTownTargetRoleReq,
		Seq: 1,
		Payload: mustJSON(t, classicTownRoleInteractionRequest{
			Handle: "1000542608713897",
			RoleID: "-1",
			Kind:   "npc",
			MapID:  "1",
		}),
	})

	if !result.handled {
		t.Fatal("expected targetRole to be handled")
	}
	if result.responseCmd != 0 || result.answerSpeak != nil {
		t.Fatalf("expected targetRole to be fire-and-forget, got response cmd %d answerSpeak=%+v", result.responseCmd, result.answerSpeak)
	}
}

func TestHandlePacketClassicTownActiveRolePushesAnswerSpeak(t *testing.T) {
	store := session.NewStore()
	result := handlePacket(store, protocol.Packet{
		Cmd: cmdClassicTownActiveRoleReq,
		Seq: 1,
		Payload: mustJSON(t, classicTownRoleInteractionRequest{
			Handle: "1000542608713897",
			RoleID: "-1",
			Kind:   "npc",
			MapID:  "1",
		}),
	})

	if !result.handled {
		t.Fatal("expected activeRole to be handled")
	}
	if result.responseCmd != 0 {
		t.Fatalf("expected activeRole to have no direct response, got cmd %d", result.responseCmd)
	}
	if result.answerSpeak == nil {
		t.Fatal("expected activeRole to produce answerSpeak push")
	}
	if result.answerSpeak.Handle != "1000542608713897" {
		t.Fatalf("expected answerSpeak handle to match, got %q", result.answerSpeak.Handle)
	}
	if result.answerSpeak.MsgHandle != "1" {
		t.Fatalf("expected captured msg handle 1, got %q", result.answerSpeak.MsgHandle)
	}
	if result.answerSpeak.Msg != `((一间茅屋从来住，心无是非万境闲。长问尘世谁有路，态意从容若神仙。))
贫道一心长态是也，久居云隐村这风轻云淡之隅，昼来农田耕种，夜来把酒赏月，想来这神仙的生活也不过老夫这般了。
((每日13:00-15:00或者19:00-21:00到我这可以接取生死劫任务))。` {
		t.Fatalf("expected source-specific answerSpeak msg, got %q", result.answerSpeak.Msg)
	}
	if len(result.answerSpeak.Answers) != 6 {
		t.Fatalf("expected captured answerSpeak options, got %+v", result.answerSpeak)
	}
}

func TestHandlePacketClassicTownActiveRolePlayerDoesNotOpenAnswerSpeak(t *testing.T) {
	store := session.NewStore()
	result := handlePacket(store, protocol.Packet{
		Cmd: cmdClassicTownActiveRoleReq,
		Seq: 1,
		Payload: mustJSON(t, classicTownRoleInteractionRequest{
			Handle: "acct-cap1366655383-role-222",
			RoleID: "acct-cap1366655383-role-222",
			Kind:   "player",
			MapID:  "1",
		}),
	})

	if !result.handled {
		t.Fatal("expected player activeRole to be handled")
	}
	if result.responseCmd != 0 || result.answerSpeak != nil {
		t.Fatalf("expected player activeRole to stay fire-and-forget, got response cmd %d answerSpeak=%+v", result.responseCmd, result.answerSpeak)
	}
}

func TestHandlePacketClassicTownActiveCollectionStartsSubGameThenCollectionGrantsRewardAndQuest(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	gloves, ok := session.CapturedRoleItemTemplate("普通采集手套")
	if !ok {
		t.Fatal("expected captured 普通采集手套 template")
	}
	gloves.Type = "背包"
	gloves.Index = -1
	gloves.Count = 1
	if _, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, gloves); !ok {
		t.Fatal("expected to grant 普通采集手套 for collection test")
	}
	if selectedRole, playerBase, ok := store.GetRoleRuntimeData(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID); ok {
		socketSession.selectedRole = &selectedRole
		socketSession.playerBase = &playerBase
	}
	transferResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownTransferReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownTransferRequest{
			MapID: "89",
			X:     1242,
			Y:     451,
		}),
	}, socketSession)
	if !transferResult.handled || transferResult.townBootstrap == nil {
		t.Fatalf("expected transfer to map89 bootstrap, got %+v", transferResult)
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownActiveRoleReq,
		Seq: 3,
		Payload: mustJSON(t, classicTownRoleInteractionRequest{
			Handle: "2810542613719308",
			RoleID: "2810542613719308",
			Kind:   "collection",
			MapID:  "89",
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected collection activeRole to be handled")
	}
	if result.answerSpeak != nil || result.subGame == nil {
		t.Fatalf("expected collection activeRole to open subGame only, got answerSpeak=%+v subGame=%+v", result.answerSpeak, result.subGame)
	}
	if result.subGame.Handle != "2810542613719308" || result.subGame.MapID != "89" || result.subGame.DurationSeconds != 5 || result.subGame.SourcePayload != "pro:5|" {
		t.Fatalf("expected source c_subGame progress push, got %+v", result.subGame)
	}
	if len(result.itemInfos) != 0 || len(result.questInfos) != 0 || len(result.chatMessages) != 0 {
		t.Fatalf("activeRole must not grant before Collection, got items=%+v quests=%+v chats=%+v", result.itemInfos, result.questInfos, result.chatMessages)
	}

	rewardResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownCollectionReq,
		Seq: 4,
		Payload: mustJSON(t, classicTownCollectionRequest{
			Handle: "2810542613719308",
			MapID:  "89",
			Score:  0,
			Code:   "",
			Time:   123,
		}),
	}, socketSession)
	if !rewardResult.handled {
		t.Fatal("expected Collection request to be handled")
	}
	if rewardResult.answerSpeak != nil || rewardResult.collectionComplete == nil || rewardResult.collectionComplete.Msg != "你获得了【金银花】。" {
		t.Fatalf("expected collection complete subgame push, got answerSpeak=%+v complete=%+v", rewardResult.answerSpeak, rewardResult.collectionComplete)
	}
	if len(rewardResult.itemInfos) != 1 || rewardResult.itemInfos[0].Type != "背包" || rewardResult.itemInfos[0].Name != "金银花" || rewardResult.itemInfos[0].Count != 1 || rewardResult.itemInfos[0].Display != "97.png" {
		t.Fatalf("expected 金银花 item push, got %+v", rewardResult.itemInfos)
	}
	if len(rewardResult.questInfos) != 1 || rewardResult.questInfos[0].Title != "采集金银花" {
		t.Fatalf("expected collection quest info, got %+v", rewardResult.questInfos)
	}
	if len(rewardResult.questStates) != 1 || rewardResult.questStates[0].Handle != "2810542613719308" || rewardResult.questStates[0].State != 2 {
		t.Fatalf("expected collection quest state refresh, got %+v", rewardResult.questStates)
	}
	if len(rewardResult.chatMessages) != 2 || !strings.Contains(rewardResult.chatMessages[0].Msg, "获得了【金银花】x1") || rewardResult.chatMessages[1].Msg != "日志更新" {
		t.Fatalf("expected collection source system notifications, got %+v", rewardResult.chatMessages)
	}
}

func TestHandlePacketClassicTownCollectionGrantsHuanglianOnMap91(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	gloves, ok := session.CapturedRoleItemTemplate("普通采集手套")
	if !ok {
		t.Fatal("expected captured 普通采集手套 template")
	}
	gloves.Type = "背包"
	gloves.Index = -1
	gloves.Count = 1
	if _, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, gloves); !ok {
		t.Fatal("expected to grant 普通采集手套 for collection test")
	}
	if selectedRole, playerBase, ok := store.GetRoleRuntimeData(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID); ok {
		socketSession.selectedRole = &selectedRole
		socketSession.playerBase = &playerBase
	}
	transferResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownTransferReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownTransferRequest{
			MapID: "91",
			X:     1755,
			Y:     452,
		}),
	}, socketSession)
	if !transferResult.handled || transferResult.townBootstrap == nil {
		t.Fatalf("expected transfer to map91 bootstrap, got %+v", transferResult)
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownActiveRoleReq,
		Seq: 3,
		Payload: mustJSON(t, classicTownRoleInteractionRequest{
			Handle: "2050542611677774",
			RoleID: "2050542611677774",
			Kind:   "collection",
			MapID:  "91",
		}),
	}, socketSession)
	if !result.handled || result.subGame == nil || result.subGame.Handle != "2050542611677774" {
		t.Fatalf("expected map91 collection activeRole to open subGame, got %+v", result)
	}

	rewardResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownCollectionReq,
		Seq: 4,
		Payload: mustJSON(t, classicTownCollectionRequest{
			Handle: "2050542611677774",
			MapID:  "91",
			Score:  0,
			Code:   "",
			Time:   123,
		}),
	}, socketSession)
	if !rewardResult.handled {
		t.Fatal("expected map91 Collection request to be handled")
	}
	if rewardResult.answerSpeak != nil || rewardResult.collectionComplete == nil || rewardResult.collectionComplete.Msg != "你获得了【黄连】。" {
		t.Fatalf("expected map91 collection complete push, got answerSpeak=%+v complete=%+v", rewardResult.answerSpeak, rewardResult.collectionComplete)
	}
	if len(rewardResult.itemInfos) != 1 || rewardResult.itemInfos[0].Type != "背包" || rewardResult.itemInfos[0].Name != "黄连" || rewardResult.itemInfos[0].Count != 1 || rewardResult.itemInfos[0].Display != "95.png" {
		t.Fatalf("expected 黄连 item push, got %+v", rewardResult.itemInfos)
	}
	if len(rewardResult.questInfos) != 1 || rewardResult.questInfos[0].Title != "采集黄连" {
		t.Fatalf("expected collection quest info, got %+v", rewardResult.questInfos)
	}
}

func TestHandlePacketClassicTownCollectionWarnsWhenBagFull(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	gloves, ok := session.CapturedRoleItemTemplate("普通采集手套")
	if !ok {
		t.Fatal("expected captured 普通采集手套 template")
	}
	gloves.Type = "背包"
	gloves.Index = -1
	gloves.Count = 1
	if _, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, gloves); !ok {
		t.Fatal("expected to grant 普通采集手套 for collection test")
	}
	fillBagSlotsForTest(t, store, socketSession, 1)
	if selectedRole, playerBase, ok := store.GetRoleRuntimeData(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID); ok {
		socketSession.selectedRole = &selectedRole
		socketSession.playerBase = &playerBase
	}
	transferResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownTransferReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownTransferRequest{
			MapID: "91",
			X:     1755,
			Y:     452,
		}),
	}, socketSession)
	if !transferResult.handled || transferResult.townBootstrap == nil {
		t.Fatalf("expected transfer to map91 bootstrap, got %+v", transferResult)
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownActiveRoleReq,
		Seq: 3,
		Payload: mustJSON(t, classicTownRoleInteractionRequest{
			Handle: "2050542611677774",
			RoleID: "2050542611677774",
			Kind:   "collection",
			MapID:  "91",
		}),
	}, socketSession)
	if !result.handled || result.subGame == nil {
		t.Fatalf("expected full-bag collection activeRole to still open subGame, got %+v", result)
	}

	rewardResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownCollectionReq,
		Seq: 4,
		Payload: mustJSON(t, classicTownCollectionRequest{
			Handle: "2050542611677774",
			MapID:  "91",
			Score:  0,
			Code:   "",
			Time:   123,
		}),
	}, socketSession)
	if !rewardResult.handled {
		t.Fatal("expected full-bag Collection request to be handled")
	}
	if rewardResult.collectionComplete == nil || !strings.Contains(rewardResult.collectionComplete.Msg, "背包空间不足") {
		t.Fatalf("expected full-bag collection completion message, got %+v", rewardResult.collectionComplete)
	}
	if len(rewardResult.itemInfos) != 0 || len(rewardResult.questInfos) != 0 {
		t.Fatalf("full-bag collection must not grant item or quest progress, got items=%+v quests=%+v", rewardResult.itemInfos, rewardResult.questInfos)
	}
	if len(rewardResult.chatMessages) != 1 || !strings.Contains(rewardResult.chatMessages[0].Msg, "背包空间不足") || !rewardResult.chatMessages[0].Bold {
		t.Fatalf("expected full-bag warning message, got %+v", rewardResult.chatMessages)
	}
}

func TestHandlePacketClassicTownActiveCollectionRequiresTool(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	transferResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownTransferReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownTransferRequest{
			MapID: "91",
			X:     1755,
			Y:     452,
		}),
	}, socketSession)
	if !transferResult.handled || transferResult.townBootstrap == nil {
		t.Fatalf("expected transfer to map91 bootstrap, got %+v", transferResult)
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownActiveRoleReq,
		Seq: 3,
		Payload: mustJSON(t, classicTownRoleInteractionRequest{
			Handle: "2050542611677774",
			RoleID: "2050542611677774",
			Kind:   "collection",
			MapID:  "91",
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected collection activeRole without tool to be handled")
	}
	if result.answerSpeak == nil || result.answerSpeak.Msg != "需要携带普通采集手套才能进行采集。" {
		t.Fatalf("expected missing-tool answerSpeak, got %+v", result.answerSpeak)
	}
	if len(result.itemInfos) != 0 || len(result.questInfos) != 0 {
		t.Fatalf("expected no reward without tool, got items=%+v quests=%+v", result.itemInfos, result.questInfos)
	}
}

func TestHandlePacketClassicTownAnswerNoResponse(t *testing.T) {
	store := session.NewStore()
	result := handlePacket(store, protocol.Packet{
		Cmd: cmdClassicTownAnswerReq,
		Seq: 1,
		Payload: mustJSON(t, classicTownAnswerRequest{
			Handle:       "1000542608713897",
			MsgHandle:    "msg-1000542608713897-001",
			AnswerHandle: "talk",
		}),
	})

	if !result.handled {
		t.Fatal("expected Answer to be handled")
	}
	if result.responseCmd != 0 || result.answerSpeak != nil {
		t.Fatalf("expected Answer to be fire-and-forget, got response cmd %d answerSpeak=%+v", result.responseCmd, result.answerSpeak)
	}
}

func TestHandlePacketClassicTownAnswerCanPushFollowUpDialogue(t *testing.T) {
	store := session.NewStore()
	result := handlePacket(store, protocol.Packet{
		Cmd: cmdClassicTownAnswerReq,
		Seq: 1,
		Payload: mustJSON(t, classicTownAnswerRequest{
			Handle:       "2000542608832485",
			MsgHandle:    "1",
			AnswerHandle: "1q28gs",
		}),
	})

	if !result.handled {
		t.Fatal("expected Answer to be handled")
	}
	if result.responseCmd != 0 {
		t.Fatalf("expected Answer follow-up to be pushed, got response cmd %d", result.responseCmd)
	}
	if result.answerSpeak == nil {
		t.Fatal("expected Answer follow-up dialogue")
	}
	if result.answerSpeak.MsgHandle != "1q28d_1" {
		t.Fatalf("expected follow-up msg handle 1q28d_1, got %q", result.answerSpeak.MsgHandle)
	}
	if len(result.answerSpeak.Answers) != 2 || result.answerSpeak.Answers[0].Handle != "1q28a_1_1" {
		t.Fatalf("expected follow-up answers, got %+v", result.answerSpeak.Answers)
	}
}

func TestHandlePacketClassicTownAnswerInactiveTransportPushesCapturedError(t *testing.T) {
	result := handlePacket(session.NewStore(), protocol.Packet{
		Cmd: cmdClassicTownAnswerReq,
		Seq: 1,
		Payload: mustJSON(t, classicTownAnswerRequest{
			Handle:       "4170542615108676",
			MsgHandle:    "1",
			AnswerHandle: "3",
		}),
	})

	if !result.handled || result.answerSpeak != nil || len(result.chatMessages) != 0 || len(result.errorMessages) != 1 {
		t.Fatalf("expected inactive transport Answer to push captured c_Error only, got %+v", result)
	}
	errorMessage := result.errorMessages[0]
	if errorMessage.Msg != "传送点未激活！" || !strings.Contains(errorMessage.SourceCapture, "Speak(101)") || !strings.Contains(errorMessage.SourceCapture, "c_Error") {
		t.Fatalf("expected captured inactive transport c_Error, got %+v", errorMessage)
	}
}

func TestHandlePacketClassicTownWarehouseAnswerOpensContainer(t *testing.T) {
	_, socketSession := seedSelectedRoleSession(t)
	result := handlePacketWithSession(session.NewStore(), protocol.Packet{
		Cmd: cmdClassicTownAnswerReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownAnswerRequest{
			Handle:       "5050542617322114",
			MsgHandle:    "1",
			AnswerHandle: "6",
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected warehouse Answer to be handled")
	}
	if result.answerSpeak != nil {
		t.Fatalf("expected warehouse Answer to open container instead of reply dialogue, got %+v", result.answerSpeak)
	}
	if result.openContainer == nil {
		t.Fatal("expected warehouse Answer to push openContainer")
	}
	if result.openContainer.Handle != "5050542617322114" || result.openContainer.Type != classicWarehouseContainerType {
		t.Fatalf("expected warehouse openContainer for source NPC, got %+v", result.openContainer)
	}
	if !strings.Contains(result.openContainer.SourceCapture, "c_openContainer") {
		t.Fatalf("expected source capture to cite c_openContainer, got %+v", result.openContainer)
	}
}

func TestHandlePacketClassicTownWarehouseCareState(t *testing.T) {
	_, socketSession := seedSelectedRoleSession(t)
	result := handlePacketWithSession(session.NewStore(), protocol.Packet{
		Cmd:     cmdClassicTownCareStateReq,
		Seq:     2,
		Payload: mustJSON(t, map[string]any{"source": "warehouse"}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected warehouse care state request to be handled")
	}
	if result.responseCmd != 0 {
		t.Fatalf("expected care state to be push-only, got response cmd %d", result.responseCmd)
	}
	if result.careState == nil {
		t.Fatal("expected warehouse care state push")
	}
	if result.careState.LockState != classicWarehouseLockedState {
		t.Fatalf("expected captured locked state %q, got %+v", classicWarehouseLockedState, result.careState)
	}
	if result.careState.Message != classicWarehouseLockedStateMessage {
		t.Fatalf("expected source locked message, got %+v", result.careState)
	}
	if !strings.Contains(result.careState.SourceCapture, "c_careState") {
		t.Fatalf("expected source capture to cite c_careState, got %+v", result.careState)
	}
}

func TestHandlePacketClassicTownGetSkillListPushesSourceShape(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	result := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetSkillListReq,
		Seq:     2,
		Payload: mustJSON(t, map[string]any{}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected GetSkillList to be handled")
	}
	if result.responseCmd != 0 || result.answerSpeak != nil {
		t.Fatalf("expected GetSkillList to be push-only, got response cmd %d answerSpeak=%+v", result.responseCmd, result.answerSpeak)
	}
	if result.skillCap == nil || result.skillCap.Count != 12 {
		t.Fatalf("expected source skill capacity 12, got %+v", result.skillCap)
	}
	if len(result.skillInfos) != 2 {
		t.Fatalf("expected new role to have two source default skills, got %+v", result.skillInfos)
	}
	if result.skillInfos[0].Name != "密斩" || result.skillInfos[0].Icon != "426.png" || result.skillInfos[0].Type != "oneE" {
		t.Fatalf("expected captured mizhan skill info, got %+v", result.skillInfos[0])
	}
	if result.skillInfos[1].Name != "普通攻击" || result.skillInfos[1].Icon != "7.png" || result.skillInfos[1].Type != "oneE" {
		t.Fatalf("expected captured normal attack skill info, got %+v", result.skillInfos[1])
	}
	if result.currencyPush == nil || result.currencyPush.Currencies["铜钱"] != 5000 || result.currencyPush.Currencies["银元宝"] != 1 {
		t.Fatalf("expected skill list to push default currencies, got %+v", result.currencyPush)
	}
}

func TestHandlePacketClassicTownGetAbilityCountPushesSourceShape(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	result := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetAbilityCountReq,
		Seq:     2,
		Payload: mustJSON(t, map[string]any{}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected GetAbilityCount to be handled")
	}
	if result.responseCmd != 0 || result.answerSpeak != nil {
		t.Fatalf("expected GetAbilityCount to be push-only, got response cmd %d answerSpeak=%+v", result.responseCmd, result.answerSpeak)
	}
	if result.abilityCount == nil || result.abilityCount.Handle != socketSession.selectedRole.RoleID || result.abilityCount.Count != 12 {
		t.Fatalf("expected source abilityCount handle/count, got %+v", result.abilityCount)
	}
	if result.skillCap != nil || len(result.skillInfos) != 0 {
		t.Fatalf("expected GetAbilityCount not to push skill list directly, got cap=%+v skills=%+v", result.skillCap, result.skillInfos)
	}
}

func TestHandlePacketClassicTownRemoveSkillPushesClearSkillInfo(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	_, _, found, learned := store.LearnRoleSkill(
		socketSession.playerBase.PlayerID,
		socketSession.selectedRole.RoleID,
		session.RoleSkill{Name: "武器专精", Level: 1, Type: "被动技能", Icon: "631.png"},
	)
	if !found || !learned {
		t.Fatal("expected test skill to be learned")
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownRemoveSkillReq,
		Seq: 3,
		Payload: mustJSON(t, classicTownRemoveSkillRequest{
			Name: "武器专精",
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected RemoveSkill to be handled")
	}
	if len(result.skillClears) != 1 || result.skillClears[0].Handle != socketSession.selectedRole.RoleID || result.skillClears[0].Name != "武器专精" {
		t.Fatalf("expected clearSkillInfo push for 武器专精, got %+v", result.skillClears)
	}
	listResult := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetSkillListReq,
		Seq:     4,
		Payload: mustJSON(t, map[string]any{}),
	}, socketSession)
	for _, skill := range listResult.skillInfos {
		if skill.Name == "武器专精" {
			t.Fatalf("expected removed skill to stay removed, got %+v", listResult.skillInfos)
		}
	}
}

func TestHandlePacketClassicTownRemoveSkillRejectsNormalAttack(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownRemoveSkillReq,
		Seq: 3,
		Payload: mustJSON(t, classicTownRemoveSkillRequest{
			Name: "普通攻击",
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected RemoveSkill to be handled")
	}
	if len(result.skillClears) != 0 {
		t.Fatalf("expected normal attack removal to be rejected without clear push, got %+v", result.skillClears)
	}
}

func TestHandlePacketClassicTownRemoveDefaultSkillDoesNotResurrect(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownRemoveSkillReq,
		Seq: 3,
		Payload: mustJSON(t, classicTownRemoveSkillRequest{
			Name: "密斩",
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected RemoveSkill to be handled")
	}
	if len(result.skillClears) != 1 || result.skillClears[0].Name != "密斩" {
		t.Fatalf("expected clearSkillInfo push for 密斩, got %+v", result.skillClears)
	}

	listResult := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetSkillListReq,
		Seq:     4,
		Payload: mustJSON(t, map[string]any{}),
	}, socketSession)
	for _, skill := range listResult.skillInfos {
		if skill.Name == "密斩" {
			t.Fatalf("expected removed default skill not to be re-added, got %+v", listResult.skillInfos)
		}
	}

	fastPanelResult := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetFastPanelReq,
		Seq:     5,
		Payload: mustJSON(t, map[string]any{}),
	}, socketSession)
	for _, entry := range fastPanelResult.fastPanel.Entries {
		if entry.Name == "密斩" {
			t.Fatalf("expected removed default skill not to appear on fast panel, got %+v", fastPanelResult.fastPanel.Entries)
		}
	}
}

func TestHandlePacketClassicTownGetContainerCapacityPushesBagCapacity(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownGetCapacityReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownContainerRequest{
			Type: "背包",
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected GetContainerCapacity to be handled")
	}
	if result.responseCmd != 0 {
		t.Fatalf("expected GetContainerCapacity to be push-only, got response cmd %d", result.responseCmd)
	}
	if result.containerCap == nil || result.containerCap.Type != "背包" || result.containerCap.Capacity != 30 {
		t.Fatalf("expected source bag capacity push, got %+v", result.containerCap)
	}
}

func TestHandlePacketClassicTownBattleBootyContainerAndMove(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	socketSession.battleLoot = []session.RoleItem{{
		Type:        "战斗",
		Name:        "朽木",
		ItemType:    "own",
		Display:     "915.png",
		Description: "f_i_朽木&24@材料&25@99&20@一块烂木头，看起来没有什么用处。",
		Count:       2,
		Index:       0,
		ItemLevel:   1,
	}}

	listResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownGetItemListReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownContainerRequest{
			Type: "战斗",
		}),
	}, socketSession)
	if !listResult.handled {
		t.Fatal("expected battle booty GetItemList to be handled")
	}
	if listResult.containerCap == nil || listResult.containerCap.Type != "战斗" || listResult.containerCap.Capacity != 18 {
		t.Fatalf("expected source battle booty capacity 18, got %+v", listResult.containerCap)
	}
	if len(listResult.itemInfos) != 1 || listResult.itemInfos[0].Name != "朽木" || listResult.itemInfos[0].Index != 0 {
		t.Fatalf("expected battle loot item push, got %+v", listResult.itemInfos)
	}

	moveResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownContainerMove,
		Seq: 3,
		Payload: mustJSON(t, classicTownContainerMoveRequest{
			SourceType: "战斗",
			TargetType: "背包",
		}),
	}, socketSession)
	if !moveResult.handled {
		t.Fatal("expected battle booty ContainerMove to be handled")
	}
	if len(moveResult.itemClears) != 1 || moveResult.itemClears[0].Type != "战斗" || moveResult.itemClears[0].Index != 0 {
		t.Fatalf("expected battle slot clear after move, got %+v", moveResult.itemClears)
	}
	if len(moveResult.itemInfos) != 1 || moveResult.itemInfos[0].Type != "背包" || moveResult.itemInfos[0].Name != "朽木" {
		t.Fatalf("expected moved loot to be pushed into bag, got %+v", moveResult.itemInfos)
	}
	if len(socketSession.battleLoot) != 0 {
		t.Fatalf("expected battle loot session container to be empty, got %+v", socketSession.battleLoot)
	}
}

func TestHandlePacketClassicTownBattleBootyContainerMoveFiltersNames(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	socketSession.battleLoot = []session.RoleItem{{
		Type:      "战斗",
		Name:      "朽木",
		ItemType:  "own",
		Display:   "915.png",
		Count:     2,
		Index:     0,
		ItemLevel: 1,
	}, {
		Type:      "战斗",
		Name:      "兽牙",
		ItemType:  "own",
		Display:   "913.png",
		Count:     1,
		Index:     1,
		ItemLevel: 2,
	}}

	moveResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownContainerMove,
		Seq: 3,
		Payload: mustJSON(t, classicTownContainerMoveRequest{
			SourceType: "战斗",
			TargetType: "背包",
			Names:      []string{"兽牙"},
		}),
	}, socketSession)
	if !moveResult.handled {
		t.Fatal("expected battle booty filtered ContainerMove to be handled")
	}
	if len(moveResult.itemClears) != 1 || moveResult.itemClears[0].Index != 1 {
		t.Fatalf("expected only filtered battle slot to clear, got %+v", moveResult.itemClears)
	}
	if len(moveResult.itemInfos) != 1 || moveResult.itemInfos[0].Type != "背包" || moveResult.itemInfos[0].Name != "兽牙" {
		t.Fatalf("expected only filtered loot to move to bag, got %+v", moveResult.itemInfos)
	}
	if len(socketSession.battleLoot) != 1 || socketSession.battleLoot[0].Name != "朽木" || socketSession.battleLoot[0].Index != 0 {
		t.Fatalf("expected unfiltered battle loot to remain, got %+v", socketSession.battleLoot)
	}
}

func TestHandlePacketClassicTownBattleBootyMoveStacksCapturedLootIntoFullBag(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	template, ok := session.CapturedRoleItemTemplate("兽骨")
	if !ok {
		t.Fatal("expected captured 兽骨 template")
	}
	template.Type = "背包"
	template.Index = 0
	template.Count = 38
	template.Owner = ""
	if _, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, template); !ok {
		t.Fatal("expected existing 兽骨 stack to be granted")
	}
	fillBagSlotsForTest(t, store, socketSession, 1)
	socketSession.battleLoot = buildClassicBattleLoot(socketSession, battle.ResultPayload{
		Winner: battle.CampTeam,
		Items:  []string{"兽骨x1"},
	})
	if len(socketSession.battleLoot) != 1 || socketSession.battleLoot[0].Owner != "" {
		t.Fatalf("expected unbound 兽骨 battle loot, got %+v", socketSession.battleLoot)
	}

	moveResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownContainerMove,
		Seq: 3,
		Payload: mustJSON(t, classicTownContainerMoveRequest{
			SourceType: "战斗",
			TargetType: "背包",
		}),
	}, socketSession)
	if !moveResult.handled {
		t.Fatal("expected battle booty ContainerMove to be handled")
	}
	if len(moveResult.itemClears) != 1 || moveResult.itemClears[0].Type != "战斗" || moveResult.itemClears[0].Index != 0 {
		t.Fatalf("expected battle slot to clear after stacking, got %+v", moveResult.itemClears)
	}
	if len(moveResult.itemInfos) != 1 || moveResult.itemInfos[0].Type != "背包" || moveResult.itemInfos[0].Name != "兽骨" || moveResult.itemInfos[0].Count != 39 || moveResult.itemInfos[0].Index != 0 {
		t.Fatalf("expected loot to stack into existing 兽骨 slot, got %+v", moveResult.itemInfos)
	}
	if len(moveResult.chatMessages) != 0 {
		t.Fatalf("expected no full-bag warning when stacking succeeds, got %+v", moveResult.chatMessages)
	}
	if len(socketSession.battleLoot) != 0 {
		t.Fatalf("expected battle loot session container to be empty, got %+v", socketSession.battleLoot)
	}
}

func TestHandlePacketClassicTownBattleBootyMoveStacksHeadboneWithIconDescriptionDrift(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	template, ok := session.CapturedRoleItemTemplate("头骨")
	if !ok {
		t.Fatal("expected captured 头骨 template")
	}
	template.Type = "背包"
	template.Index = 0
	template.Count = 16
	template.Owner = ""
	template.Description = strings.Replace(template.Description, "&101@102.png", "", 1)
	if _, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, template); !ok {
		t.Fatal("expected existing 头骨 stack to be granted")
	}
	fillBagSlotsForTest(t, store, socketSession, 1)
	socketSession.battleLoot = buildClassicBattleLoot(socketSession, battle.ResultPayload{
		Winner: battle.CampTeam,
		Items:  []string{"头骨x1"},
	})

	moveResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownContainerMove,
		Seq: 3,
		Payload: mustJSON(t, classicTownContainerMoveRequest{
			SourceType: "战斗",
			TargetType: "背包",
		}),
	}, socketSession)
	if !moveResult.handled {
		t.Fatal("expected battle booty ContainerMove to be handled")
	}
	if len(moveResult.itemClears) != 1 || moveResult.itemClears[0].Type != "战斗" || moveResult.itemClears[0].Index != 0 {
		t.Fatalf("expected battle headbone slot to clear after stacking, got %+v", moveResult.itemClears)
	}
	if len(moveResult.itemInfos) != 1 || moveResult.itemInfos[0].Type != "背包" || moveResult.itemInfos[0].Name != "头骨" || moveResult.itemInfos[0].Count != 17 || moveResult.itemInfos[0].Index != 0 {
		t.Fatalf("expected headbone loot to stack into existing slot, got %+v", moveResult.itemInfos)
	}
	if len(moveResult.chatMessages) != 0 {
		t.Fatalf("expected no full-bag warning when headbone stacks, got %+v", moveResult.chatMessages)
	}
	if len(socketSession.battleLoot) != 0 {
		t.Fatalf("expected battle loot session container to be empty, got %+v", socketSession.battleLoot)
	}
}

func TestHandlePacketClassicTownBattleBootyMoveWarnsWhenBagFull(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	fillBagSlotsForTest(t, store, socketSession, 0)
	socketSession.battleLoot = []session.RoleItem{{
		Type:      "战斗",
		Name:      "满背包测试物",
		ItemType:  "own",
		Count:     1,
		Index:     0,
		ItemLevel: 1,
	}}

	moveResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownContainerMove,
		Seq: 3,
		Payload: mustJSON(t, classicTownContainerMoveRequest{
			SourceType: "战斗",
			TargetType: "背包",
		}),
	}, socketSession)
	if !moveResult.handled {
		t.Fatal("expected battle booty ContainerMove to be handled")
	}
	if len(moveResult.itemClears) != 0 || len(moveResult.itemInfos) != 0 {
		t.Fatalf("expected no item changes when full bag rejects loot, got infos=%+v clears=%+v", moveResult.itemInfos, moveResult.itemClears)
	}
	if len(moveResult.chatMessages) != 1 || !strings.Contains(moveResult.chatMessages[0].Msg, "背包空间不足") || !moveResult.chatMessages[0].Bold {
		t.Fatalf("expected full-bag warning message, got %+v", moveResult.chatMessages)
	}
	if len(socketSession.battleLoot) != 1 || socketSession.battleLoot[0].Name != "满背包测试物" {
		t.Fatalf("expected rejected loot to remain in battle container, got %+v", socketSession.battleLoot)
	}
}

func TestHandlePacketClassicTownBattleBootyContainerMoveExchangesBattleSlots(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	sourceIndex := 0
	targetIndex := 1
	socketSession.battleLoot = []session.RoleItem{{
		Type:      "战斗",
		Name:      "朽木",
		ItemType:  "own",
		Display:   "915.png",
		Count:     2,
		Index:     0,
		ItemLevel: 1,
	}, {
		Type:      "战斗",
		Name:      "兽牙",
		ItemType:  "own",
		Display:   "913.png",
		Count:     1,
		Index:     1,
		ItemLevel: 2,
	}}

	moveResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownContainerMove,
		Seq: 3,
		Payload: mustJSON(t, classicTownContainerMoveRequest{
			SourceType:  "战斗",
			TargetType:  "战斗",
			SourceIndex: &sourceIndex,
			TargetIndex: &targetIndex,
		}),
	}, socketSession)
	if !moveResult.handled {
		t.Fatal("expected battle booty slot exchange to be handled")
	}
	if len(moveResult.itemClears) != 2 {
		t.Fatalf("expected both battle slots to clear before exchange pushes, got %+v", moveResult.itemClears)
	}
	if len(moveResult.itemInfos) != 2 {
		t.Fatalf("expected both exchanged battle items to be pushed, got %+v", moveResult.itemInfos)
	}
	if socketSession.battleLoot[0].Name != "朽木" || socketSession.battleLoot[0].Index != 1 {
		t.Fatalf("expected source item to move to target battle slot, got %+v", socketSession.battleLoot)
	}
	if socketSession.battleLoot[1].Name != "兽牙" || socketSession.battleLoot[1].Index != 0 {
		t.Fatalf("expected target item to swap into source battle slot, got %+v", socketSession.battleLoot)
	}
}

func TestHandlePacketClassicBattleOverAppliesSourceResultRewards(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	startResult := startClassicBattleForTest(t, socketSession)
	socketSession.battleRuntime.PendingOver = &battle.OverPush{
		BattleID: startResult.battleStart.BattleID,
		Winner:   battle.CampTeam,
		Rounds:   1,
		Result: battle.ResultPayload{
			Winner:   battle.CampTeam,
			Rounds:   1,
			ExpDelta: 37,
			Items:    []string{"朽木"},
		},
	}

	playOver := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattlePlayOverReq,
		Seq: 4,
		Payload: mustJSON(t, battle.PlayOverRequest{
			BattleID: startResult.battleStart.BattleID,
		}),
	}, socketSession)

	if !playOver.handled || playOver.battleOver == nil {
		t.Fatalf("expected BattlePlayOver to return source OverBattle, got %+v", playOver)
	}
	if playOver.roleState == nil || playOver.roleState.Exp != 37 {
		t.Fatalf("expected roleState exp to come from OverBattle result expDelta, got %+v", playOver.roleState)
	}
	if playOver.rolePhysique == nil || playOver.rolePhysique.MaxHP != 205 || playOver.rolePhysique.MaxMP != 54 {
		t.Fatalf("expected level-up rolePhysique after OverBattle exp reward, got %+v", playOver.rolePhysique)
	}
	if socketSession.playerBase == nil || socketSession.playerBase.Exp != 37 || socketSession.selectedRole.Exp != 37 {
		t.Fatalf("expected selected role/playerBase exp to persist result expDelta, got role=%+v playerBase=%+v", socketSession.selectedRole, socketSession.playerBase)
	}
	if socketSession.playerBase.Level != 2 || socketSession.selectedRole.Level != 2 {
		t.Fatalf("expected source RoleConvert exp table to level role to 2, got role=%+v playerBase=%+v", socketSession.selectedRole, socketSession.playerBase)
	}
	if len(socketSession.battleLoot) != 1 || socketSession.battleLoot[0].Name != "朽木" || socketSession.battleLoot[0].Type != "战斗" {
		t.Fatalf("expected result items to enter battle loot container, got %+v", socketSession.battleLoot)
	}
}

func TestHandlePacketClassicBattleOverRemovesDefeatedVisibleMonster(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	const visibleMonsterHandle = "5172206909807859"
	expectedRemovedHandles := []string{"5172206909807859", "5174206909807286"}

	startResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattleStartReq,
		Seq: 2,
		Payload: mustJSON(t, battle.StartRequest{
			MapID:               "143",
			MapName:             "水帘洞_13",
			StageFocusX:         631,
			ReturnRoute:         "town-placeholder",
			SourceMonsterHandle: visibleMonsterHandle,
		}),
	}, socketSession)
	if startResult.battleStart == nil || socketSession.battleRuntime == nil || socketSession.battleRuntime.SourceMonsterHandle != visibleMonsterHandle {
		t.Fatalf("expected visible monster battle to start, got result=%+v runtime=%+v", startResult, socketSession.battleRuntime)
	}
	socketSession.battleRuntime.PendingOver = &battle.OverPush{
		BattleID: startResult.battleStart.BattleID,
		Winner:   battle.CampTeam,
		Rounds:   1,
		Result: battle.ResultPayload{
			Winner: battle.CampTeam,
			Rounds: 1,
		},
	}

	playOver := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattlePlayOverReq,
		Seq: 3,
		Payload: mustJSON(t, battle.PlayOverRequest{
			BattleID: startResult.battleStart.BattleID,
		}),
	}, socketSession)
	if playOver.battleOver == nil || len(playOver.removeRoleHandles) != len(expectedRemovedHandles) {
		t.Fatalf("expected visible monster group removeRole after BattlePlayOver, got %+v", playOver)
	}
	for index, expectedHandle := range expectedRemovedHandles {
		if playOver.removeRoleHandles[index] != expectedHandle {
			t.Fatalf("expected visible monster removeRole %d to be %s, got %+v", index, expectedHandle, playOver.removeRoleHandles)
		}
		if !socketSession.defeatedVisibleMonsters[expectedHandle] {
			t.Fatalf("expected session to retain defeated visible monster handle %s, got %+v", expectedHandle, socketSession.defeatedVisibleMonsters)
		}
	}

	transfer := buildClassicTownTransferResult(store, socketSession, "143", world.SpawnPoint{X: 1000, Y: 600})
	if transfer.townBootstrap == nil {
		t.Fatalf("expected map143 transfer bootstrap, got %+v", transfer)
	}
	if transfer.dungeonInstance == nil || !transfer.dungeonInstance.Active || transfer.dungeonInstance.Key != session.DungeonInstanceShuiliandong || transfer.dungeonInstance.DisplayName != "水帘洞" {
		t.Fatalf("expected map143 transfer to sync active shuiliandong instance, got %+v", transfer.dungeonInstance)
	}
	if transfer.dungeonInstance.DurationSeconds != session.DungeonInstanceTTLSeconds() || transfer.dungeonInstance.ExpiresAtUnix <= transfer.dungeonInstance.CreatedAtUnix || transfer.dungeonInstance.RemainingSeconds <= 0 {
		t.Fatalf("expected map143 transfer to include countdown fields, got %+v", transfer.dungeonInstance)
	}
	for _, rolePush := range transfer.townBootstrap.CreateRoles {
		for _, expectedHandle := range expectedRemovedHandles {
			if rolePush.Handle == expectedHandle {
				t.Fatalf("expected defeated visible monster group to stay removed from map143 bootstrap, got %+v", rolePush)
			}
		}
	}

	newSocketSession := &packetSession{}
	selectAgain := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdRoleSelectRequest,
		Seq: 4,
		Payload: mustJSON(t, session.RoleSelectRequest{
			PlayerID:     socketSession.playerBase.PlayerID,
			SessionToken: "mock-session-token-001",
			RoleID:       socketSession.selectedRole.RoleID,
		}),
	}, newSocketSession)
	if selectAgain.townBootstrap == nil {
		t.Fatalf("expected reselect bootstrap to restore dungeon instance state, got %+v", selectAgain)
	}
	if selectAgain.dungeonInstance == nil || !selectAgain.dungeonInstance.Active || selectAgain.dungeonInstance.Key != session.DungeonInstanceShuiliandong {
		t.Fatalf("expected reselect to push active shuiliandong instance, got %+v", selectAgain.dungeonInstance)
	}
	for _, expectedHandle := range expectedRemovedHandles {
		if !newSocketSession.defeatedVisibleMonsters[expectedHandle] {
			t.Fatalf("expected new socket session to restore defeated visible monster %s, got %+v", expectedHandle, newSocketSession.defeatedVisibleMonsters)
		}
	}
	for _, rolePush := range selectAgain.townBootstrap.CreateRoles {
		for _, expectedHandle := range expectedRemovedHandles {
			if rolePush.Handle == expectedHandle {
				t.Fatalf("expected defeated visible monster group to stay removed after reselect, got %+v", rolePush)
			}
		}
	}

	duplicateStart := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattleStartReq,
		Seq: 5,
		Payload: mustJSON(t, battle.StartRequest{
			MapID:               "143",
			MapName:             "水帘洞_13",
			StageFocusX:         631,
			SourceMonsterHandle: visibleMonsterHandle,
		}),
	}, socketSession)
	if duplicateStart.battleStart != nil || socketSession.battleRuntime != nil {
		t.Fatalf("expected defeated visible monster battle to be rejected, got result=%+v runtime=%+v", duplicateStart, socketSession.battleRuntime)
	}

	groupMateStart := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattleStartReq,
		Seq: 6,
		Payload: mustJSON(t, battle.StartRequest{
			MapID:               "143",
			MapName:             "水帘洞_13",
			StageFocusX:         487,
			SourceMonsterHandle: "5174206909807286",
		}),
	}, socketSession)
	if groupMateStart.battleStart != nil || socketSession.battleRuntime != nil {
		t.Fatalf("expected defeated visible monster group mate battle to be rejected, got result=%+v runtime=%+v", groupMateStart, socketSession.battleRuntime)
	}

	exitTransfer := buildClassicTownTransferResult(store, socketSession, "122", world.SpawnPoint{X: 1000, Y: 600})
	if exitTransfer.dungeonInstance == nil || exitTransfer.dungeonInstance.Active || exitTransfer.dungeonInstance.Key != "" {
		t.Fatalf("expected non-dungeon transfer to hide dungeon countdown, got %+v", exitTransfer.dungeonInstance)
	}
}

func TestClassicTownMapSpecialBuildsCapturedLastTimePayload(t *testing.T) {
	push := buildClassicTownMapSpecialPush(&classicTownDungeonInstancePush{
		Active:        true,
		ExpiresAtUnix: 1781288111,
	})
	if push == nil || push.LastTime != 1781288111000 {
		t.Fatalf("expected lastTime to be expiresAtUnix milliseconds, got %+v", push)
	}
	if len(push.Entries) != 1 || push.Entries[0] != "lastTime:1781288111000" {
		t.Fatalf("expected source-style lastTime entry, got %+v", push)
	}
	if !strings.Contains(push.SourceCapture, "c_mapSpecial") || !strings.Contains(push.SourceCapture, "lastTime") {
		t.Fatalf("expected capture source pointer for c_mapSpecial lastTime, got %+v", push)
	}

	clear := buildClassicTownMapSpecialPush(&classicTownDungeonInstancePush{Active: false})
	if clear == nil || clear.LastTime != 0 || len(clear.Entries) != 0 {
		t.Fatalf("expected inactive dungeon to clear lastTime, got %+v", clear)
	}
}

func TestClassicTownGetMapSpecialUsesDungeonExpiry(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	role, playerBase, ok := store.UpdateRoleMap(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, 143)
	if !ok {
		t.Fatal("expected test role to move to map143")
	}
	socketSession.selectedRole = &role
	socketSession.playerBase = &playerBase

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownGetMapSpecialReq,
		Seq: 41,
		Payload: mustJSON(t, classicTownMapSpecialRequest{
			MapID: "143",
		}),
	}, socketSession)
	if !result.handled || result.mapSpecial == nil {
		t.Fatalf("expected GetMapSpecial to push MapSpecial, got %+v", result)
	}
	if result.mapSpecial.LastTime <= time.Now().UnixMilli() {
		t.Fatalf("expected future lastTime from dungeon expiry, got %+v", result.mapSpecial)
	}
	if len(result.mapSpecial.Entries) != 1 || !strings.HasPrefix(result.mapSpecial.Entries[0], "lastTime:") {
		t.Fatalf("expected source-style lastTime entry, got %+v", result.mapSpecial)
	}
}

func TestHandlePacketClassicBattleOverRemovesHuangfengzhaiVisibleMonster(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	grantRoleItemTemplateForTest(t, store, socketSession, "黄风寨通行证", 1)
	const visibleMonsterHandle = "3218685759638239"
	expectedRemovedHandles := []string{"3220685759639165", "3218685759638239"}

	transfer := buildClassicTownTransferResult(store, socketSession, "149", world.SpawnPoint{X: 1451, Y: 403})
	if transfer.townBootstrap == nil {
		t.Fatalf("expected map149 transfer bootstrap, got %+v", transfer)
	}
	if transfer.dungeonInstance == nil || !transfer.dungeonInstance.Active || transfer.dungeonInstance.Key != session.DungeonInstanceHuangfengzhai || transfer.dungeonInstance.DisplayName != "黄风寨" {
		t.Fatalf("expected map149 transfer to sync active huangfengzhai instance, got %+v", transfer.dungeonInstance)
	}
	if transfer.dungeonInstance.DurationSeconds != session.DungeonInstanceTTLSeconds() || transfer.dungeonInstance.ExpiresAtUnix <= transfer.dungeonInstance.CreatedAtUnix || transfer.dungeonInstance.RemainingSeconds <= 0 {
		t.Fatalf("expected map149 transfer to include countdown fields, got %+v", transfer.dungeonInstance)
	}

	startResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattleStartReq,
		Seq: 2,
		Payload: mustJSON(t, battle.StartRequest{
			MapID:               "149",
			MapName:             "黄风寨_4",
			StageFocusX:         1451,
			ReturnRoute:         "town-placeholder",
			SourceMonsterHandle: visibleMonsterHandle,
		}),
	}, socketSession)
	if startResult.battleStart == nil || socketSession.battleRuntime == nil || socketSession.battleRuntime.SourceMonsterHandle != visibleMonsterHandle {
		t.Fatalf("expected huangfengzhai visible monster battle to start, got result=%+v runtime=%+v", startResult, socketSession.battleRuntime)
	}
	socketSession.battleRuntime.PendingOver = &battle.OverPush{
		BattleID: startResult.battleStart.BattleID,
		Winner:   battle.CampTeam,
		Rounds:   1,
		Result: battle.ResultPayload{
			Winner: battle.CampTeam,
			Rounds: 1,
		},
	}

	playOver := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattlePlayOverReq,
		Seq: 3,
		Payload: mustJSON(t, battle.PlayOverRequest{
			BattleID: startResult.battleStart.BattleID,
		}),
	}, socketSession)
	if playOver.battleOver == nil || len(playOver.removeRoleHandles) != len(expectedRemovedHandles) {
		t.Fatalf("expected huangfengzhai visible monster group removeRole after BattlePlayOver, got %+v", playOver)
	}
	for index, expectedHandle := range expectedRemovedHandles {
		if playOver.removeRoleHandles[index] != expectedHandle {
			t.Fatalf("expected huangfengzhai visible monster removeRole %d to be %s, got %+v", index, expectedHandle, playOver.removeRoleHandles)
		}
		if !socketSession.defeatedVisibleMonsters[expectedHandle] {
			t.Fatalf("expected session to retain huangfengzhai defeated visible monster handle %s, got %+v", expectedHandle, socketSession.defeatedVisibleMonsters)
		}
	}

	reenter := buildClassicTownTransferResult(store, socketSession, "149", world.SpawnPoint{X: 1000, Y: 600})
	if reenter.townBootstrap == nil || reenter.dungeonInstance == nil || !reenter.dungeonInstance.Active || reenter.dungeonInstance.Key != session.DungeonInstanceHuangfengzhai {
		t.Fatalf("expected reenter map149 to keep huangfengzhai instance active, got %+v", reenter)
	}
	for _, rolePush := range reenter.townBootstrap.CreateRoles {
		for _, expectedHandle := range expectedRemovedHandles {
			if rolePush.Handle == expectedHandle {
				t.Fatalf("expected defeated huangfengzhai visible monster group to stay removed from map149 bootstrap, got %+v", rolePush)
			}
		}
	}

	newSocketSession := &packetSession{}
	selectAgain := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdRoleSelectRequest,
		Seq: 4,
		Payload: mustJSON(t, session.RoleSelectRequest{
			PlayerID:     socketSession.playerBase.PlayerID,
			SessionToken: "mock-session-token-001",
			RoleID:       socketSession.selectedRole.RoleID,
		}),
	}, newSocketSession)
	if selectAgain.townBootstrap == nil || selectAgain.dungeonInstance == nil || !selectAgain.dungeonInstance.Active || selectAgain.dungeonInstance.Key != session.DungeonInstanceHuangfengzhai {
		t.Fatalf("expected reselect to restore huangfengzhai instance state, got %+v", selectAgain)
	}
	for _, expectedHandle := range expectedRemovedHandles {
		if !newSocketSession.defeatedVisibleMonsters[expectedHandle] {
			t.Fatalf("expected new socket session to restore huangfengzhai defeated visible monster %s, got %+v", expectedHandle, newSocketSession.defeatedVisibleMonsters)
		}
	}
}

func TestHandlePacketClassicBattleOverRemovesFeixiandongVisibleMonster(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	role, playerBase, ok := store.UpdateRoleMap(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, 18)
	if !ok {
		t.Fatal("expected test role map update to feixiandong entrance")
	}
	socketSession.selectedRole = &role
	socketSession.playerBase = &playerBase
	grantRoleItemTemplateForTest(t, store, socketSession, "飞仙洞通行证", 1)

	const visibleMonsterHandle = "1048675671977626"
	expectedRemovedHandles := []string{"1042675671973672", "1048675671977626"}

	transfer := buildClassicTownTransferResult(store, socketSession, "76", world.SpawnPoint{X: 1576, Y: 515})
	if transfer.townBootstrap == nil {
		t.Fatalf("expected map76 transfer bootstrap, got %+v", transfer)
	}
	if transfer.dungeonInstance == nil || !transfer.dungeonInstance.Active || transfer.dungeonInstance.Key != session.DungeonInstanceFeixiandong || transfer.dungeonInstance.DisplayName != "飞仙洞" {
		t.Fatalf("expected map76 transfer to sync active feixiandong instance, got %+v", transfer.dungeonInstance)
	}

	startResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattleStartReq,
		Seq: 2,
		Payload: mustJSON(t, battle.StartRequest{
			MapID:               "76",
			MapName:             "飞仙洞_13",
			StageFocusX:         1576,
			ReturnRoute:         "town-placeholder",
			SourceMonsterHandle: visibleMonsterHandle,
		}),
	}, socketSession)
	if startResult.battleStart == nil || socketSession.battleRuntime == nil || socketSession.battleRuntime.SourceMonsterHandle != visibleMonsterHandle {
		t.Fatalf("expected feixiandong visible monster battle to start, got result=%+v runtime=%+v", startResult, socketSession.battleRuntime)
	}
	socketSession.battleRuntime.PendingOver = &battle.OverPush{
		BattleID: startResult.battleStart.BattleID,
		Winner:   battle.CampTeam,
		Rounds:   1,
		Result: battle.ResultPayload{
			Winner: battle.CampTeam,
			Rounds: 1,
		},
	}

	playOver := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattlePlayOverReq,
		Seq: 3,
		Payload: mustJSON(t, battle.PlayOverRequest{
			BattleID: startResult.battleStart.BattleID,
		}),
	}, socketSession)
	if playOver.battleOver == nil || len(playOver.removeRoleHandles) != len(expectedRemovedHandles) {
		t.Fatalf("expected feixiandong visible monster group removeRole after BattlePlayOver, got %+v", playOver)
	}
	for index, expectedHandle := range expectedRemovedHandles {
		if playOver.removeRoleHandles[index] != expectedHandle {
			t.Fatalf("expected feixiandong visible monster removeRole %d to be %s, got %+v", index, expectedHandle, playOver.removeRoleHandles)
		}
		if !socketSession.defeatedVisibleMonsters[expectedHandle] {
			t.Fatalf("expected session to retain feixiandong defeated visible monster handle %s, got %+v", expectedHandle, socketSession.defeatedVisibleMonsters)
		}
	}

	reenter := buildClassicTownTransferResult(store, socketSession, "76", world.SpawnPoint{X: 1000, Y: 600})
	if reenter.townBootstrap == nil || reenter.dungeonInstance == nil || !reenter.dungeonInstance.Active || reenter.dungeonInstance.Key != session.DungeonInstanceFeixiandong {
		t.Fatalf("expected reenter map76 to keep feixiandong instance active, got %+v", reenter)
	}
	for _, rolePush := range reenter.townBootstrap.CreateRoles {
		for _, expectedHandle := range expectedRemovedHandles {
			if rolePush.Handle == expectedHandle {
				t.Fatalf("expected defeated feixiandong visible monster group to stay removed from map76 bootstrap, got %+v", rolePush)
			}
		}
	}

	newSocketSession := &packetSession{}
	selectAgain := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdRoleSelectRequest,
		Seq: 4,
		Payload: mustJSON(t, session.RoleSelectRequest{
			PlayerID:     socketSession.playerBase.PlayerID,
			SessionToken: "mock-session-token-001",
			RoleID:       socketSession.selectedRole.RoleID,
		}),
	}, newSocketSession)
	if selectAgain.townBootstrap == nil || selectAgain.dungeonInstance == nil || !selectAgain.dungeonInstance.Active || selectAgain.dungeonInstance.Key != session.DungeonInstanceFeixiandong {
		t.Fatalf("expected reselect to restore feixiandong instance state, got %+v", selectAgain)
	}
	for _, expectedHandle := range expectedRemovedHandles {
		if !newSocketSession.defeatedVisibleMonsters[expectedHandle] {
			t.Fatalf("expected new socket session to restore feixiandong defeated visible monster %s, got %+v", expectedHandle, newSocketSession.defeatedVisibleMonsters)
		}
	}
	for _, rolePush := range selectAgain.townBootstrap.CreateRoles {
		for _, expectedHandle := range expectedRemovedHandles {
			if rolePush.Handle == expectedHandle {
				t.Fatalf("expected defeated feixiandong visible monster group to stay removed after reselect, got %+v", rolePush)
			}
		}
	}

	groupMateStart := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattleStartReq,
		Seq: 5,
		Payload: mustJSON(t, battle.StartRequest{
			MapID:               "76",
			MapName:             "飞仙洞_13",
			StageFocusX:         1761,
			SourceMonsterHandle: "1042675671973672",
		}),
	}, socketSession)
	if groupMateStart.battleStart != nil || socketSession.battleRuntime != nil {
		t.Fatalf("expected defeated feixiandong visible monster group mate battle to be rejected, got result=%+v runtime=%+v", groupMateStart, socketSession.battleRuntime)
	}
}

func TestClassicTownDungeonEntryRequiresAndConsumesTicketOncePerInstance(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	role, playerBase, ok := store.UpdateRoleMap(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, 122)
	if !ok {
		t.Fatal("expected test role map update to huangfengzhai entrance")
	}
	socketSession.selectedRole = &role
	socketSession.playerBase = &playerBase

	rejected := buildClassicTownTransferResult(store, socketSession, "146", world.SpawnPoint{X: 329, Y: 480})
	if rejected.townBootstrap != nil || rejected.dungeonInstance != nil {
		t.Fatalf("expected dungeon entry without ticket to be rejected, got %+v", rejected)
	}
	if len(rejected.chatMessages) != 1 || !strings.Contains(rejected.chatMessages[0].Msg, "黄风寨通行证x1") {
		t.Fatalf("expected missing ticket system message, got %+v", rejected.chatMessages)
	}
	role, _, ok = store.GetRoleRuntimeData(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID)
	if !ok || role.MapID != 122 {
		t.Fatalf("expected rejected entry to keep role on map122, got ok=%v role=%+v", ok, role)
	}

	granted := grantRoleItemTemplateForTest(t, store, socketSession, "黄风寨通行证", 2)
	firstEntry := buildClassicTownTransferResult(store, socketSession, "146", world.SpawnPoint{X: 329, Y: 480})
	if firstEntry.townBootstrap == nil || firstEntry.dungeonInstance == nil || !firstEntry.dungeonInstance.Active || firstEntry.dungeonInstance.Key != session.DungeonInstanceHuangfengzhai {
		t.Fatalf("expected ticketed entry to enter huangfengzhai, got %+v", firstEntry)
	}
	if len(firstEntry.itemInfos) != 1 || firstEntry.itemInfos[0].Index != granted.Index || firstEntry.itemInfos[0].Count != 1 {
		t.Fatalf("expected first entry to decrement ticket stack, got infos=%+v granted=%+v", firstEntry.itemInfos, granted)
	}
	if len(firstEntry.itemClears) != 0 {
		t.Fatalf("expected stacked ticket not to clear slot, got %+v", firstEntry.itemClears)
	}

	exit := buildClassicTownTransferResult(store, socketSession, "122", world.SpawnPoint{X: 1000, Y: 600})
	if exit.townBootstrap == nil || exit.dungeonInstance == nil || exit.dungeonInstance.Active {
		t.Fatalf("expected exit to ordinary map to hide dungeon timer, got %+v", exit)
	}
	reentry := buildClassicTownTransferResult(store, socketSession, "146", world.SpawnPoint{X: 329, Y: 480})
	if reentry.townBootstrap == nil || reentry.dungeonInstance == nil || !reentry.dungeonInstance.Active {
		t.Fatalf("expected active instance reentry without extra ticket consume, got %+v", reentry)
	}
	if len(reentry.itemInfos) != 0 || len(reentry.itemClears) != 0 {
		t.Fatalf("expected reentry during active instance not to consume another ticket, got infos=%+v clears=%+v", reentry.itemInfos, reentry.itemClears)
	}
	ticket, ok := store.GetRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, "背包", granted.Index)
	if !ok || ticket.Name != "黄风寨通行证" || ticket.Count != 1 {
		t.Fatalf("expected one ticket to remain after reentry, got ok=%v ticket=%+v", ok, ticket)
	}
}

func TestClassicTownShuiliandongEntryRequiresTicketAndClearsSingleTicketSlot(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	role, playerBase, ok := store.UpdateRoleMap(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, 127)
	if !ok {
		t.Fatal("expected test role map update to shuiliandong entrance")
	}
	socketSession.selectedRole = &role
	socketSession.playerBase = &playerBase

	rejected := buildClassicTownTransferResult(store, socketSession, "131", world.SpawnPoint{X: 1020, Y: 300})
	if rejected.townBootstrap != nil || len(rejected.chatMessages) != 1 || !strings.Contains(rejected.chatMessages[0].Msg, "水帘洞通行证x1") {
		t.Fatalf("expected shuiliandong entry without ticket to be rejected, got %+v", rejected)
	}

	granted := grantRoleItemTemplateForTest(t, store, socketSession, "水帘洞通行证", 1)
	entry := buildClassicTownTransferResult(store, socketSession, "131", world.SpawnPoint{X: 1020, Y: 300})
	if entry.townBootstrap == nil || entry.dungeonInstance == nil || !entry.dungeonInstance.Active || entry.dungeonInstance.Key != session.DungeonInstanceShuiliandong {
		t.Fatalf("expected ticketed shuiliandong entry, got %+v", entry)
	}
	if len(entry.itemClears) != 1 || entry.itemClears[0].Type != "背包" || entry.itemClears[0].Index != granted.Index {
		t.Fatalf("expected single shuiliandong ticket slot clear, got clears=%+v granted=%+v", entry.itemClears, granted)
	}
}

func TestClassicTownFeixiandongEntryRequiresTicketAndClearsSingleTicketSlot(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	role, playerBase, ok := store.UpdateRoleMap(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, 18)
	if !ok {
		t.Fatal("expected test role map update to feixiandong entrance")
	}
	socketSession.selectedRole = &role
	socketSession.playerBase = &playerBase

	rejected := buildClassicTownTransferResult(store, socketSession, "64", world.SpawnPoint{X: 1000, Y: 600})
	if rejected.townBootstrap != nil || len(rejected.chatMessages) != 1 || !strings.Contains(rejected.chatMessages[0].Msg, "飞仙洞通行证x1") {
		t.Fatalf("expected feixiandong entry without ticket to be rejected, got %+v", rejected)
	}

	granted := grantRoleItemTemplateForTest(t, store, socketSession, "飞仙洞通行证", 1)
	entry := buildClassicTownTransferResult(store, socketSession, "64", world.SpawnPoint{X: 1000, Y: 600})
	if entry.townBootstrap == nil || entry.dungeonInstance == nil || !entry.dungeonInstance.Active || entry.dungeonInstance.Key != session.DungeonInstanceFeixiandong || entry.dungeonInstance.DisplayName != "飞仙洞" {
		t.Fatalf("expected ticketed feixiandong entry, got %+v", entry)
	}
	if len(entry.itemClears) != 1 || entry.itemClears[0].Type != "背包" || entry.itemClears[0].Index != granted.Index {
		t.Fatalf("expected single feixiandong ticket slot clear, got clears=%+v granted=%+v", entry.itemClears, granted)
	}
}

func TestClassicTownShihukuEntryRequiresTicketAndClearsActualTicketSlot(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	role, playerBase, ok := store.UpdateRoleMap(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, 36)
	if !ok {
		t.Fatal("expected test role map update to shihuku entrance")
	}
	socketSession.selectedRole = &role
	socketSession.playerBase = &playerBase

	rejected := buildClassicTownTransferResult(store, socketSession, "158", world.SpawnPoint{X: 120, Y: 481})
	if rejected.townBootstrap != nil || len(rejected.chatMessages) != 1 || !strings.Contains(rejected.chatMessages[0].Msg, "狮虎窟通行证x1") {
		t.Fatalf("expected shihuku entry without ticket to be rejected, got %+v", rejected)
	}

	granted := grantRoleItemTemplateForTest(t, store, socketSession, "狮虎窟通行证", 1)
	entry := buildClassicTownTransferResult(store, socketSession, "158", world.SpawnPoint{X: 120, Y: 481})
	if entry.townBootstrap == nil || entry.dungeonInstance == nil || !entry.dungeonInstance.Active || entry.dungeonInstance.Key != session.DungeonInstanceShihuku || entry.dungeonInstance.DisplayName != "狮虎窟" {
		t.Fatalf("expected ticketed shihuku entry, got %+v", entry)
	}
	if len(entry.itemClears) != 1 || entry.itemClears[0].Type != "背包" || entry.itemClears[0].Index != granted.Index {
		t.Fatalf("expected single shihuku ticket actual slot clear, got clears=%+v granted=%+v", entry.itemClears, granted)
	}
}

func TestHandlePacketClassicTownAddPointPushesSourcePhysique(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	socketSession.selectedRole.Level = 2
	socketSession.selectedRole.Exp = 80
	socketSession.playerBase.Level = 2
	socketSession.playerBase.Exp = 80
	storeState := store.GrantRoleExperience(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, 80)
	if !storeState.Found || !storeState.Granted {
		t.Fatalf("expected test role to reach lv2, got %+v", storeState)
	}
	socketSession.selectedRole = &storeState.Role
	socketSession.playerBase = &storeState.PlayerBase

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownAddPointReq,
		Seq: 5,
		Payload: mustJSON(t, classicTownAddPointRequest{
			Stat: "STR",
		}),
	}, socketSession)

	if !result.handled || result.rolePhysique == nil {
		t.Fatalf("expected AddPoint to push rolePhysique, got %+v", result)
	}
	if result.rolePhysique.STR != 1 || result.rolePhysique.PhyAtk != 11 || result.rolePhysique.LastPoint != 9 {
		t.Fatalf("expected source STR AddPoint physique delta, got %+v", result.rolePhysique)
	}
	if socketSession.playerBase == nil || socketSession.playerBase.RolePhysique == nil || socketSession.playerBase.RolePhysique.STR != 1 {
		t.Fatalf("expected socket session playerBase to refresh after AddPoint, got %+v", socketSession.playerBase)
	}

	for index := 0; index < 9; index += 1 {
		consumePoint := handlePacketWithSession(store, protocol.Packet{
			Cmd: cmdClassicTownAddPointReq,
			Seq: uint64(6 + index),
			Payload: mustJSON(t, classicTownAddPointRequest{
				Stat: "STR",
			}),
		}, socketSession)
		if !consumePoint.handled || consumePoint.rolePhysique == nil {
			t.Fatalf("expected AddPoint %d to consume remaining point, got %+v", index, consumePoint)
		}
	}

	noPoint := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownAddPointReq,
		Seq: 20,
		Payload: mustJSON(t, classicTownAddPointRequest{
			Stat: "STR",
		}),
	}, socketSession)
	if !noPoint.handled || noPoint.rolePhysique != nil || len(noPoint.chatMessages) != 0 || len(noPoint.errorMessages) != 1 {
		t.Fatalf("expected no-point AddPoint to return captured c_Error only, got %+v", noPoint)
	}
	noPointError := noPoint.errorMessages[0]
	if noPointError.Msg != "你已经没有剩余点数了" || !strings.Contains(noPointError.SourceCapture, "AddPoint(170)") || !strings.Contains(noPointError.SourceCapture, "c_Error") {
		t.Fatalf("expected captured AddPoint no-point c_Error, got %+v", noPointError)
	}
}

func TestBuildClassicBattleLootUsesCapturedSourceItemMetadata(t *testing.T) {
	_, socketSession := seedSelectedRoleSession(t)
	loot := buildClassicBattleLoot(socketSession, battle.ResultPayload{
		Winner: battle.CampTeam,
		Items:  []string{"肉"},
	})

	if len(loot) != 1 {
		t.Fatalf("expected one source battle loot item, got %+v", loot)
	}
	item := loot[0]
	if item.Type != "战斗" || item.Name != "肉" || item.Count != 1 || item.Index != 0 {
		t.Fatalf("expected captured meat reward in battle container slot 0, got %+v", item)
	}
	if item.Display != "70.png" || item.ItemType != "own" || !strings.Contains(item.Description, "f_i_肉") {
		t.Fatalf("expected captured meat display metadata, got %+v", item)
	}
	if item.Handle != socketSession.selectedRole.RoleID || item.Owner != "" {
		t.Fatalf("expected loot handle without role-bound owner metadata, got %+v", item)
	}
}

func TestBuildClassicBattleLootParsesCapturedItemCounts(t *testing.T) {
	_, socketSession := seedSelectedRoleSession(t)
	loot := buildClassicBattleLoot(socketSession, battle.ResultPayload{
		Winner: battle.CampTeam,
		Items:  []string{"铜钱x5", "盗贼的首级x1", "雪莲花x1"},
	})

	if len(loot) != 3 {
		t.Fatalf("expected three captured reward stacks, got %+v", loot)
	}
	if loot[0].Name != "铜钱" || loot[0].Count != 5 || loot[0].Index != 0 || loot[0].Display != "163.png" {
		t.Fatalf("expected captured 铜钱x5 reward stack, got %+v", loot[0])
	}
	if loot[1].Name != "盗贼的首级" || loot[1].Count != 1 || loot[1].Index != 1 || loot[1].Display == "" {
		t.Fatalf("expected captured 盗贼的首级x1 reward stack, got %+v", loot[1])
	}
	var snowLotus *session.RoleItem
	for index := range loot {
		if loot[index].Name == "雪莲花" {
			snowLotus = &loot[index]
			break
		}
	}
	if snowLotus == nil || snowLotus.Count != 1 || snowLotus.Display != "935.png" || !strings.Contains(snowLotus.Description, "任务物品") {
		t.Fatalf("expected captured 雪莲花x1 quest item stack, got %+v", loot)
	}
}

func TestBuildClassicBattleLootUsesPointCouponMetadata(t *testing.T) {
	_, socketSession := seedSelectedRoleSession(t)
	loot := buildClassicBattleLoot(socketSession, battle.ResultPayload{
		Winner: battle.CampTeam,
		Items:  []string{"点券x10"},
	})

	if len(loot) != 1 {
		t.Fatalf("expected one point coupon loot stack, got %+v", loot)
	}
	if loot[0].Name != "点券" || loot[0].Count != 10 || loot[0].Display != "659.png" || loot[0].ItemLevel != 5 {
		t.Fatalf("expected captured 点券 metadata, got %+v", loot[0])
	}
	if !strings.Contains(loot[0].Description, "特殊消费或商城购物") {
		t.Fatalf("expected captured 点券 description, got %+v", loot[0])
	}
}

func TestBuildClassicBattleLootUsesHuangfengEquipmentMetadata(t *testing.T) {
	_, socketSession := seedSelectedRoleSession(t)
	loot := buildClassicBattleLoot(socketSession, battle.ResultPayload{
		Winner: battle.CampTeam,
		Items:  []string{"黄风腰带x1"},
	})

	if len(loot) != 1 {
		t.Fatalf("expected one 黄风腰带 loot stack, got %+v", loot)
	}
	item := loot[0]
	if item.Type != classicBattleLootType || item.Name != "黄风腰带" || item.Count != 1 || item.Index != 0 {
		t.Fatalf("expected 黄风腰带 reward in battle container slot 0, got %+v", item)
	}
	if item.Display != "547.png" || item.ItemType != "equip" || item.ItemLevel != 2 || !strings.Contains(item.Description, "护具") {
		t.Fatalf("expected captured 黄风腰带 equipment metadata, got %+v", item)
	}
	if item.Handle != socketSession.selectedRole.RoleID || item.Owner != "" {
		t.Fatalf("expected loot handle without role-bound owner metadata, got %+v", item)
	}
}

func TestBuildClassicBattleLootUsesHuangfengChiefRewardMetadata(t *testing.T) {
	_, socketSession := seedSelectedRoleSession(t)
	loot := buildClassicBattleLoot(socketSession, battle.ResultPayload{
		Winner: battle.CampTeam,
		Items: []string{
			"红方巾x2",
			"绸缎x3",
			"铜钱x110",
			"盗贼的首级x1",
			"呼啸战靴x1",
			"寨夫人上衣x1",
			"寨夫人护腕x1",
			"宝匣x1",
		},
	})

	if len(loot) != 8 {
		t.Fatalf("expected eight Huangfeng chief loot stacks, got %+v", loot)
	}
	expected := map[string]struct {
		count     int
		display   string
		itemType  string
		itemLevel int
		token     string
	}{
		"红方巾":   {count: 2, display: "121.png", itemType: "null", itemLevel: 1, token: "红色绸缎制方巾"},
		"绸缎":    {count: 3, display: "79.png", itemType: "null", itemLevel: 1, token: "代表高贵的布料"},
		"铜钱":    {count: 110, display: "163.png", itemType: "own", itemLevel: 1, token: "游戏中的货币"},
		"盗贼的首级": {count: 1, display: "120.png", itemType: "null", itemLevel: 2, token: "击杀盗贼的证明"},
		"呼啸战靴":  {count: 1, display: "502.png", itemType: "equip", itemLevel: 2, token: "护具·足部"},
		"寨夫人上衣": {count: 1, display: "474.png", itemType: "equip", itemLevel: 2, token: "护具·躯干"},
		"寨夫人护腕": {count: 1, display: "467.png", itemType: "equip", itemLevel: 2, token: "护具·护腕"},
		"宝匣":    {count: 1, display: "596.png", itemType: "own", itemLevel: 3, token: "双击打开"},
	}
	for index, item := range loot {
		want, ok := expected[item.Name]
		if !ok {
			t.Fatalf("unexpected Huangfeng chief loot item %+v in %+v", item, loot)
		}
		if item.Type != classicBattleLootType || item.Count != want.count || item.Index != index || item.Display != want.display || item.ItemType != want.itemType || item.ItemLevel != want.itemLevel {
			t.Fatalf("expected %s loot metadata count=%d display=%s itemType=%s itemLevel=%d index=%d, got %+v", item.Name, want.count, want.display, want.itemType, want.itemLevel, index, item)
		}
		if !strings.Contains(item.Description, want.token) || item.Handle != socketSession.selectedRole.RoleID || item.Owner != "" {
			t.Fatalf("expected %s source description and unowned battle loot handle, got %+v", item.Name, item)
		}
	}
}

func TestBuildClassicBattleLootUsesHuangfengCandidateRewardMetadata(t *testing.T) {
	_, socketSession := seedSelectedRoleSession(t)
	loot := buildClassicBattleLoot(socketSession, battle.ResultPayload{
		Winner: battle.CampTeam,
		Items: []string{
			"刺x1",
			"红缨x2",
			"刀客布衣x1",
			"剔骨刀x1",
			"图腾面具x1",
			"黄风围巾x1",
		},
	})

	if len(loot) != 6 {
		t.Fatalf("expected six Huangfeng candidate loot stacks, got %+v", loot)
	}
	expected := map[string]struct {
		count     int
		display   string
		itemType  string
		itemLevel int
		token     string
	}{
		"刺":    {count: 1, display: "134.png", itemType: "null", itemLevel: 1, token: "锋利尖锐"},
		"红缨":   {count: 2, display: "77.png", itemType: "null", itemLevel: 1, token: "丝线染色后制成"},
		"刀客布衣": {count: 1, display: "540.png", itemType: "equip", itemLevel: 2, token: "护具·躯干"},
		"剔骨刀":  {count: 1, display: "552.png", itemType: "equip", itemLevel: 2, token: "武器·匕首系"},
		"图腾面具": {count: 1, display: "135.png", itemType: "null", itemLevel: 2, token: "诅咒的面具"},
		"黄风围巾": {count: 1, display: "548.png", itemType: "equip", itemLevel: 3, token: "护具·头部"},
	}
	for index, item := range loot {
		want, ok := expected[item.Name]
		if !ok {
			t.Fatalf("unexpected Huangfeng candidate loot item %+v in %+v", item, loot)
		}
		if item.Type != classicBattleLootType || item.Count != want.count || item.Index != index || item.Display != want.display || item.ItemType != want.itemType || item.ItemLevel != want.itemLevel {
			t.Fatalf("expected %s loot metadata count=%d display=%s itemType=%s itemLevel=%d index=%d, got %+v", item.Name, want.count, want.display, want.itemType, want.itemLevel, index, item)
		}
		if !strings.Contains(item.Description, want.token) || item.Handle != socketSession.selectedRole.RoleID || item.Owner != "" {
			t.Fatalf("expected %s source description and unowned battle loot handle, got %+v", item.Name, item)
		}
	}
}

func TestPointCouponThiefBattleOverPushesWorldNotice(t *testing.T) {
	store := session.NewStore()
	socketSession, role := seedSelectedRoleSessionInStore(t, store, "恐龙抗狼1")
	handle := classicactivity.PointCouponThiefHandle(114, 1718193600, 0)
	startResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattleStartReq,
		Seq: 2,
		Payload: mustJSON(t, battle.StartRequest{
			MapID:               "114",
			MapName:             "卧佛谷_10",
			StageFocusX:         2329,
			ReturnRoute:         "town-placeholder",
			SourceMonsterHandle: handle,
		}),
	}, socketSession)
	if startResult.battleStart == nil || socketSession.battleRuntime == nil {
		t.Fatalf("expected point coupon thief battle start, got %+v", startResult)
	}
	socketSession.battleRuntime.PendingOver = &battle.OverPush{
		BattleID: startResult.battleStart.BattleID,
		Winner:   battle.CampTeam,
		Rounds:   1,
		Result: battle.ResultPayload{
			Winner:   battle.CampTeam,
			Rounds:   1,
			ExpDelta: 50,
			Items:    []string{"点券x10"},
		},
	}

	playOver := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattlePlayOverReq,
		Seq: 3,
		Payload: mustJSON(t, battle.PlayOverRequest{
			BattleID: startResult.battleStart.BattleID,
		}),
	}, socketSession)

	if !playOver.handled || playOver.battleOver == nil {
		t.Fatalf("expected point coupon thief BattlePlayOver, got %+v", playOver)
	}
	if len(playOver.chatMessages) != 1 {
		t.Fatalf("expected one point coupon thief world notice, got %+v", playOver.chatMessages)
	}
	notice := playOver.chatMessages[0]
	if notice.Channel != "system" || notice.Msg != "<w>["+role.DisplayName+"]消灭了[点券盗贼]，幸运的获得点券奖励。" {
		t.Fatalf("expected captured point coupon thief world notice, got %+v", notice)
	}
}

func TestPointCouponThiefSpawnAnnouncementUsesCapturedWorldNotice(t *testing.T) {
	notice := classicPointCouponThiefSpawnAnnouncementMessage()

	if notice.Channel != "system" {
		t.Fatalf("expected point coupon thief spawn notice to use system channel, got %+v", notice)
	}
	if notice.Msg != "<w>[<font color='#00ccff'>点券盗贼</font>]突然出现在树海、卧佛谷、竹林地区。" {
		t.Fatalf("expected captured point coupon thief spawn notice, got %+v", notice)
	}
}

func TestBuildClassicBattleLootUsesFeixiandongBossItemMetadata(t *testing.T) {
	_, socketSession := seedSelectedRoleSession(t)
	loot := buildClassicBattleLoot(socketSession, battle.ResultPayload{
		Winner: battle.CampTeam,
		Items:  []string{"岩魔剑x1", "宝匣x1", "岩魔菱石x2"},
	})

	if len(loot) != 3 {
		t.Fatalf("expected three feixiandong boss loot stacks, got %+v", loot)
	}
	if loot[0].Name != "岩魔剑" || loot[0].ItemType != "equip" || loot[0].Display != "606.png" || !strings.Contains(loot[0].Description, "武器·单剑系") {
		t.Fatalf("expected captured 岩魔剑 equipment metadata, got %+v", loot[0])
	}
	if loot[1].Name != "宝匣" || loot[1].ItemType != "own" || loot[1].Display != "596.png" || !strings.Contains(loot[1].Description, "双击打开") {
		t.Fatalf("expected captured 宝匣 metadata, got %+v", loot[1])
	}
	if loot[2].Name != "岩魔菱石" || loot[2].Count != 2 || loot[2].Display != "112.png" || loot[2].ItemLevel != 3 {
		t.Fatalf("expected captured 岩魔菱石 material metadata, got %+v", loot[2])
	}
}

func TestHandlePacketClassicTownGetItemListPushesBagItems(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownGetItemListReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownContainerRequest{
			Type: "背包",
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected GetItemList to be handled")
	}
	if result.containerCap == nil || result.containerCap.Capacity != 30 {
		t.Fatalf("expected item list to include bag capacity, got %+v", result.containerCap)
	}
	if len(result.itemInfos) != 1 {
		t.Fatalf("expected default bag seed to push only the starter axe, got %+v", result.itemInfos)
	}
	if result.itemInfos[0].Name != "铁斧" || result.itemInfos[0].Display != "29.png" || result.itemInfos[0].ItemType != "equip" || result.itemInfos[0].Index != 19 {
		t.Fatalf("expected starter axe at bag index 19, got %+v", result.itemInfos[0])
	}
}

func TestHandlePacketClassicTownGetItemListSupportsEquipContainer(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownGetItemListReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownContainerRequest{
			Type: "装备",
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected GetItemList for equipment to be handled")
	}
	if result.containerCap == nil || result.containerCap.Type != "装备" || result.containerCap.Capacity != 20 {
		t.Fatalf("expected source equipment capacity 20, got %+v", result.containerCap)
	}
	if len(result.itemInfos) != 0 {
		t.Fatalf("expected current local seed to have no equipment items, got %+v", result.itemInfos)
	}
}

func TestHandlePacketClassicTownWarehouseCapacityAndList(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	capacityResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownGetCapacityReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownContainerRequest{
			Type: classicWarehouseContainerType,
		}),
	}, socketSession)

	if !capacityResult.handled || capacityResult.containerCap == nil {
		t.Fatalf("expected warehouse capacity push, got %+v", capacityResult)
	}
	if capacityResult.containerCap.Type != classicWarehouseContainerType || capacityResult.containerCap.Capacity != 40 {
		t.Fatalf("expected warehouse capacity 40, got %+v", capacityResult.containerCap)
	}

	listResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownGetItemListReq,
		Seq: 3,
		Payload: mustJSON(t, classicTownContainerRequest{
			Type: classicWarehouseContainerType,
		}),
	}, socketSession)

	if !listResult.handled || listResult.containerCap == nil {
		t.Fatalf("expected warehouse item list to be handled, got %+v", listResult)
	}
	if listResult.containerCap.Type != classicWarehouseContainerType || listResult.containerCap.Capacity != 40 {
		t.Fatalf("expected warehouse item list capacity 40, got %+v", listResult.containerCap)
	}
	for _, item := range listResult.itemInfos {
		if item.Type != classicWarehouseContainerType {
			t.Fatalf("expected only warehouse item infos, got %+v", listResult.itemInfos)
		}
	}
}

func TestHandlePacketClassicTownContainerMoveMovesWarehouseItemWithinWarehouseBySlot(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	warehouseItem := session.RoleItem{
		Type:      classicWarehouseContainerType,
		Name:      "warehouse drag test item",
		ItemType:  "own",
		Display:   "1.png",
		Count:     1,
		Index:     0,
		ItemLevel: 1,
	}
	if _, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, warehouseItem); !ok {
		t.Fatal("expected warehouse item grant to succeed")
	}

	sourceIndex := 0
	targetIndex := 1
	count := 1
	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownContainerMove,
		Seq: 2,
		Payload: mustJSON(t, classicTownContainerMoveRequest{
			SourceType:  classicWarehouseContainerType,
			TargetType:  classicWarehouseContainerType,
			SourceIndex: &sourceIndex,
			TargetIndex: &targetIndex,
			Count:       &count,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected warehouse ContainerMove to be handled")
	}
	if len(result.itemClears) != 2 || result.itemClears[0].Type != classicWarehouseContainerType || result.itemClears[0].Index != 0 || result.itemClears[1].Type != classicWarehouseContainerType || result.itemClears[1].Index != 1 {
		t.Fatalf("expected warehouse source/target clears, got %+v", result.itemClears)
	}
	if len(result.itemInfos) != 1 || result.itemInfos[0].Type != classicWarehouseContainerType || result.itemInfos[0].Name != "warehouse drag test item" || result.itemInfos[0].Index != 1 {
		t.Fatalf("expected moved warehouse item push at warehouse slot 1, got %+v", result.itemInfos)
	}

	listResult := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetItemListReq,
		Seq:     3,
		Payload: mustJSON(t, classicTownContainerRequest{Type: classicWarehouseContainerType}),
	}, socketSession)
	itemsByIndex := map[int]classicTownItemInfoPush{}
	for _, item := range listResult.itemInfos {
		itemsByIndex[item.Index] = item
	}
	if itemsByIndex[1].Name != "warehouse drag test item" {
		t.Fatalf("expected moved warehouse item to persist in slot 1, got %+v", listResult.itemInfos)
	}
}

func TestHandlePacketClassicTownContainerMoveMovesBagItemToWarehouseBySlot(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	sourceIndex := 19
	targetIndex := 0
	count := 1
	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownContainerMove,
		Seq: 2,
		Payload: mustJSON(t, classicTownContainerMoveRequest{
			SourceType:  classicTownBagContainerType,
			TargetType:  classicWarehouseContainerType,
			SourceIndex: &sourceIndex,
			TargetIndex: &targetIndex,
			Count:       &count,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected bag-to-warehouse ContainerMove to be handled")
	}
	if len(result.itemClears) != 2 || result.itemClears[0].Type != classicTownBagContainerType || result.itemClears[0].Index != 19 || result.itemClears[1].Type != classicWarehouseContainerType || result.itemClears[1].Index != 0 {
		t.Fatalf("expected bag source and warehouse target clears, got %+v", result.itemClears)
	}
	if len(result.itemInfos) != 1 || result.itemInfos[0].Type != classicWarehouseContainerType || result.itemInfos[0].Name != "铁斧" || result.itemInfos[0].Index != 0 {
		t.Fatalf("expected moved bag item in warehouse slot 0, got %+v", result.itemInfos)
	}

	warehouseList := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetItemListReq,
		Seq:     3,
		Payload: mustJSON(t, classicTownContainerRequest{Type: classicWarehouseContainerType}),
	}, socketSession)
	if len(warehouseList.itemInfos) != 1 || warehouseList.itemInfos[0].Name != "铁斧" || warehouseList.itemInfos[0].Index != 0 {
		t.Fatalf("expected moved bag item to persist in warehouse slot 0, got %+v", warehouseList.itemInfos)
	}
}

func TestHandlePacketClassicTownContainerMoveMovesWarehouseItemToBagBySlot(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	warehouseItem := session.RoleItem{
		Type:      classicWarehouseContainerType,
		Name:      "warehouse to bag test item",
		ItemType:  "own",
		Display:   "1.png",
		Count:     1,
		Index:     0,
		ItemLevel: 1,
	}
	if _, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, warehouseItem); !ok {
		t.Fatal("expected warehouse item grant to succeed")
	}

	sourceIndex := 0
	targetIndex := 0
	count := 1
	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownContainerMove,
		Seq: 2,
		Payload: mustJSON(t, classicTownContainerMoveRequest{
			SourceType:  classicWarehouseContainerType,
			TargetType:  classicTownBagContainerType,
			SourceIndex: &sourceIndex,
			TargetIndex: &targetIndex,
			Count:       &count,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected warehouse-to-bag ContainerMove to be handled")
	}
	if len(result.itemClears) != 2 || result.itemClears[0].Type != classicWarehouseContainerType || result.itemClears[0].Index != 0 || result.itemClears[1].Type != classicTownBagContainerType || result.itemClears[1].Index != 0 {
		t.Fatalf("expected warehouse source and bag target clears, got %+v", result.itemClears)
	}
	if len(result.itemInfos) != 1 || result.itemInfos[0].Type != classicTownBagContainerType || result.itemInfos[0].Name != "warehouse to bag test item" || result.itemInfos[0].Index != 0 {
		t.Fatalf("expected moved warehouse item in bag slot 0, got %+v", result.itemInfos)
	}

	bagList := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetItemListReq,
		Seq:     3,
		Payload: mustJSON(t, classicTownContainerRequest{Type: classicTownBagContainerType}),
	}, socketSession)
	itemsByIndex := map[int]classicTownItemInfoPush{}
	for _, item := range bagList.itemInfos {
		itemsByIndex[item.Index] = item
	}
	if itemsByIndex[0].Name != "warehouse to bag test item" {
		t.Fatalf("expected moved warehouse item to persist in bag slot 0, got %+v", bagList.itemInfos)
	}
}

func TestHandlePacketClassicTownContainerMoveSplitsBagItemToWarehouseByCount(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	stackItem := session.RoleItem{
		Type:      classicTownBagContainerType,
		Name:      "warehouse split test item",
		ItemType:  "own",
		Display:   "1.png",
		Count:     5,
		Index:     1,
		ItemLevel: 1,
	}
	if _, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, stackItem); !ok {
		t.Fatal("expected bag stack item grant to succeed")
	}

	sourceIndex := 1
	targetIndex := 2
	count := 2
	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownContainerMove,
		Seq: 2,
		Payload: mustJSON(t, classicTownContainerMoveRequest{
			SourceType:  classicTownBagContainerType,
			TargetType:  classicWarehouseContainerType,
			SourceIndex: &sourceIndex,
			TargetIndex: &targetIndex,
			Count:       &count,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected split bag-to-warehouse ContainerMove to be handled")
	}
	if len(result.itemClears) != 1 || result.itemClears[0].Type != classicWarehouseContainerType || result.itemClears[0].Index != 2 {
		t.Fatalf("expected only warehouse target clear for split move, got %+v", result.itemClears)
	}
	itemsByTypeIndex := map[string]classicTownItemInfoPush{}
	for _, item := range result.itemInfos {
		itemsByTypeIndex[item.Type+":"+strconv.Itoa(item.Index)] = item
	}
	if itemsByTypeIndex[classicTownBagContainerType+":1"].Count != 3 {
		t.Fatalf("expected source bag slot to keep count 3, got %+v", result.itemInfos)
	}
	if itemsByTypeIndex[classicWarehouseContainerType+":2"].Count != 2 {
		t.Fatalf("expected warehouse target slot to receive count 2, got %+v", result.itemInfos)
	}
}

func TestHandlePacketClassicTownEquipItemMovesAxeAndPushesAppearance(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownEquipItemReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownEquipItemRequest{
			Type:  "背包",
			Index: 19,
			Count: 1,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected EquipItem to be handled")
	}
	if result.createPlayer == nil || result.createPlayer.SourceQuery == "" {
		t.Fatalf("expected EquipItem to push updated createPlayer, got %+v", result.createPlayer)
	}
	if !strings.Contains(result.createPlayer.SourceQuery, "w8=5") {
		t.Fatalf("expected updated source query to include captured axe weapon field, got %q", result.createPlayer.SourceQuery)
	}
	if len(result.itemClears) != 1 || result.itemClears[0].Type != "背包" || result.itemClears[0].Index != 19 {
		t.Fatalf("expected bag axe slot to clear, got %+v", result.itemClears)
	}
	if len(result.itemInfos) != 1 || result.itemInfos[0].Type != "装备" || result.itemInfos[0].Index != 3 || result.itemInfos[0].Name != "铁斧" {
		t.Fatalf("expected equipment axe push at source slot 3, got %+v", result.itemInfos)
	}
	if socketSession.selectedRole == nil || !strings.Contains(socketSession.selectedRole.SourceQuery, "w8=5") {
		t.Fatalf("expected socket session role source query to update, got %+v", socketSession.selectedRole)
	}

	listResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownGetItemListReq,
		Seq: 3,
		Payload: mustJSON(t, classicTownContainerRequest{
			Type: "装备",
		}),
	}, socketSession)
	if len(listResult.itemInfos) != 1 || listResult.itemInfos[0].Name != "铁斧" || listResult.itemInfos[0].Index != 3 {
		t.Fatalf("expected equipped axe to persist in equipment list, got %+v", listResult.itemInfos)
	}
}

func TestHandlePacketClassicTownTryEquipPreviewsAppearanceWithoutEquipping(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	originalSourceQuery := socketSession.selectedRole.SourceQuery
	bagList := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownGetItemListReq,
		Seq: 1,
		Payload: mustJSON(t, classicTownContainerRequest{
			Type: classicTownBagContainerType,
		}),
	}, socketSession)
	if len(bagList.itemInfos) != 1 {
		t.Fatalf("expected starter bag item before TryEquip, got %+v", bagList.itemInfos)
	}
	starterName := bagList.itemInfos[0].Name

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownTryEquipReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownTryEquipRequest{
			Name: starterName,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected TryEquip to be handled")
	}
	if result.tryEquip == nil || result.tryEquip.ItemName != starterName {
		t.Fatalf("expected TryEquip preview push for starter axe, got %+v", result.tryEquip)
	}
	if !strings.Contains(result.tryEquip.SourceQuery, "w8=5") || result.tryEquip.SourceCapture != "TryEquip(231)->c_tryEquip(50092)" {
		t.Fatalf("expected TryEquip source query/capture to match c_tryEquip preview, got %+v", result.tryEquip)
	}
	if result.createPlayer != nil || len(result.itemClears) != 0 || len(result.itemInfos) != 0 {
		t.Fatalf("TryEquip must not mutate role or item containers, got create=%+v clears=%+v infos=%+v", result.createPlayer, result.itemClears, result.itemInfos)
	}
	if socketSession.selectedRole == nil || socketSession.selectedRole.SourceQuery != originalSourceQuery {
		t.Fatalf("TryEquip must not update selected role source query, before=%q after=%+v", originalSourceQuery, socketSession.selectedRole)
	}

	bagListAfter := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownGetItemListReq,
		Seq: 3,
		Payload: mustJSON(t, classicTownContainerRequest{
			Type: classicTownBagContainerType,
		}),
	}, socketSession)
	if len(bagListAfter.itemInfos) != 1 || bagListAfter.itemInfos[0].Name != starterName {
		t.Fatalf("expected TryEquip not to remove the starter bag item, got %+v", bagListAfter.itemInfos)
	}
}

func TestHandlePacketClassicTownTryEquipCapturedFashionSetPreview(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	equipResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownEquipItemReq,
		Seq: 1,
		Payload: mustJSON(t, classicTownEquipItemRequest{
			Type:  "背包",
			Index: 19,
			Count: 1,
		}),
	}, socketSession)
	if !equipResult.handled || equipResult.createPlayer == nil || !strings.Contains(equipResult.createPlayer.SourceQuery, "w8=5") {
		t.Fatalf("expected starter axe equip before fashion TryEquip, got %+v", equipResult)
	}
	originalSourceQuery := socketSession.selectedRole.SourceQuery

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownTryEquipReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownTryEquipRequest{
			Name: "超时空要塞",
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected TryEquip to be handled")
	}
	if result.tryEquip == nil || result.tryEquip.ItemName != "超时空要塞" {
		t.Fatalf("expected TryEquip preview push for captured fashion, got %+v", result.tryEquip)
	}
	for _, part := range []string{"w8=5", "c=88", "p=91", "se=79"} {
		if !strings.Contains(result.tryEquip.SourceQuery, part) {
			t.Fatalf("expected captured fashion TryEquip source query to include %s, got %+v", part, result.tryEquip)
		}
	}
	if result.tryEquip.SourceCapture != "TryEquip(231)->c_tryEquip(50092)" {
		t.Fatalf("expected captured TryEquip source label, got %+v", result.tryEquip)
	}
	if result.createPlayer != nil || len(result.itemClears) != 0 || len(result.itemInfos) != 0 {
		t.Fatalf("TryEquip must not mutate role or item containers, got create=%+v clears=%+v infos=%+v", result.createPlayer, result.itemClears, result.itemInfos)
	}
	if socketSession.selectedRole == nil || socketSession.selectedRole.SourceQuery != originalSourceQuery {
		t.Fatalf("TryEquip must not update selected role source query, before=%q after=%+v", originalSourceQuery, socketSession.selectedRole)
	}

	summerResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownTryEquipReq,
		Seq: 3,
		Payload: mustJSON(t, classicTownTryEquipRequest{
			Name: "盛夏缤纷",
		}),
	}, socketSession)
	if !summerResult.handled || summerResult.tryEquip == nil || summerResult.tryEquip.ItemName != "盛夏缤纷" {
		t.Fatalf("expected TryEquip preview push for captured summer fashion, got %+v", summerResult)
	}
	for _, part := range []string{"w8=5", "c=52", "p=55", "se=41", "hr=19"} {
		if !strings.Contains(summerResult.tryEquip.SourceQuery, part) {
			t.Fatalf("expected captured summer fashion TryEquip source query to include %s, got %+v", part, summerResult.tryEquip)
		}
	}
	if socketSession.selectedRole == nil || socketSession.selectedRole.SourceQuery != originalSourceQuery {
		t.Fatalf("summer TryEquip must not update selected role source query, before=%q after=%+v", originalSourceQuery, socketSession.selectedRole)
	}
}

func TestHandlePacketClassicTownEquipItemPushesReplacedBowFor333Dagger(t *testing.T) {
	store := session.NewStore()
	login := store.Login(session.LoginRequest{
		UserName: "33333333",
		Password: "33333333",
	})
	create := store.CreateRole(session.RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "333",
		Gender:         "male",
		RoleTemplateID: 1,
	})
	if !create.Success {
		t.Fatalf("expected 333 role create success, got %+v", create)
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
	if !selectResult.handled || selectResult.townBootstrap == nil {
		t.Fatalf("expected 333 role select to seed town bootstrap, got %+v", selectResult)
	}
	grantedDagger, ok := store.GrantRoleItem(login.PlayerID, create.Role.RoleID, session.RoleItem{
		Type:        "背包",
		Name:        "绯雨匕首",
		ItemType:    "equip",
		Display:     "51.png",
		Description: "f_i_绯雨匕首&24@武器·匕首系&25@1&22@游侠",
		Count:       1,
		Index:       20,
		ItemLevel:   2,
	})
	if !ok {
		t.Fatal("expected to grant captured dagger")
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownEquipItemReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownEquipItemRequest{
			Type:  grantedDagger.Type,
			Index: grantedDagger.Index,
			Count: 1,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected EquipItem to be handled")
	}
	if result.createPlayer == nil || !strings.Contains(result.createPlayer.SourceQuery, "w3=49") || strings.Contains(result.createPlayer.SourceQuery, "w1=55") {
		t.Fatalf("expected dagger source query and no stale bow appearance, got %+v", result.createPlayer)
	}
	if len(result.itemClears) != 1 || result.itemClears[0].Type != "背包" || result.itemClears[0].Index != grantedDagger.Index {
		t.Fatalf("expected dagger source bag slot to clear, got %+v", result.itemClears)
	}
	byTypeIndex := map[string]classicTownItemInfoPush{}
	for _, item := range result.itemInfos {
		byTypeIndex[item.Type+":"+strconv.Itoa(item.Index)] = item
	}
	if item := byTypeIndex["装备:3"]; item.Name != "绯雨匕首" || item.Display != "51.png" {
		t.Fatalf("expected dagger equipment itemInfo, got %+v from %+v", item, result.itemInfos)
	}
	if item := byTypeIndex["背包:"+strconv.Itoa(grantedDagger.Index)]; item.Name != "万相" {
		t.Fatalf("expected replaced bow bag itemInfo, got %+v from %+v", item, result.itemInfos)
	}
	if socketSession.selectedRole == nil || !strings.Contains(socketSession.selectedRole.SourceQuery, "w3=49") || strings.Contains(socketSession.selectedRole.SourceQuery, "w1=55") {
		t.Fatalf("expected socket selected role to keep dagger appearance, got %+v", socketSession.selectedRole)
	}
}

func TestHandlePacketClassicTownActiveItemExchangesSilverToCopper(t *testing.T) {
	store := session.NewStore()
	socketSession, role := seedSelectedRoleSessionInStore(t, store, "元宝兑换测试")
	silver, ok := session.CapturedRoleItemTemplate("银元宝")
	if !ok {
		t.Fatal("expected silver template")
	}
	silver.Type = "背包"
	silver.Index = -1
	silver.Count = 2
	granted, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, role.RoleID, silver)
	if !ok {
		t.Fatal("expected silver grant")
	}
	if _, ok := store.AddRoleCurrency(socketSession.playerBase.PlayerID, role.RoleID, "银元宝", 1); !ok {
		t.Fatal("expected silver currency add")
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownActiveItemReq,
		Seq: 9,
		Payload: mustJSON(t, classicTownActiveItemRequest{
			Type:  granted.Type,
			Index: granted.Index,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected ActiveItem to be handled")
	}
	if result.currencyPush == nil || result.currencyPush.Currencies["银元宝"] != 1 || result.currencyPush.Currencies["铜钱"] != 6000 {
		t.Fatalf("expected silver->copper currencies, got %+v", result.currencyPush)
	}
	if !packetChatMessagesContain(result.chatMessages, "\u83b7\u5f97\u7269\u54c1:[\u94dc\u94b1]x1000") {
		t.Fatalf("expected captured silver exchange c_Speak, got %+v", result.chatMessages)
	}
	items := itemInfosByName(result.itemInfos)
	if items["银元宝"].Count != 1 || items["铜钱"].Count != 1000 {
		t.Fatalf("expected silver item down to 1 and copper x1000 push, got %+v", result.itemInfos)
	}
}

func TestHandlePacketClassicTownActiveItemEquipsSourceArmor(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	reward := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownAnswerReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownAnswerRequest{
			Handle:       "4000542609162635",
			MsgHandle:    "3q3d_1",
			AnswerHandle: "3q3a_1_1",
		}),
	}, socketSession)
	rewardItems := itemInfosByName(reward.itemInfos)
	bagItem := rewardItems["蓝布衣"]
	if bagItem.Name == "" {
		t.Fatalf("expected reward armor item in %+v", reward.itemInfos)
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownActiveItemReq,
		Seq: 3,
		Payload: mustJSON(t, classicTownActiveItemRequest{
			Type:  bagItem.Type,
			Index: bagItem.Index,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected ActiveItem equip to be handled")
	}
	if len(result.itemInfos) != 1 || result.itemInfos[0].Type != "装备" || result.itemInfos[0].Index != 4 || result.itemInfos[0].Name != "蓝布衣" {
		t.Fatalf("expected active item to equip source armor into equipment slot 4, got %+v", result.itemInfos)
	}
	if len(result.itemClears) != 1 || result.itemClears[0].Type != bagItem.Type || result.itemClears[0].Index != bagItem.Index {
		t.Fatalf("expected active item to clear source bag slot, got %+v", result.itemClears)
	}
	if result.createPlayer == nil || !strings.Contains(result.createPlayer.SourceQuery, "c=1") {
		t.Fatalf("expected active equip to push createPlayer source query c=1, got %+v", result.createPlayer)
	}
	if result.rolePhysique == nil {
		t.Fatalf("expected active equip to push rolePhysique")
	}
	if result.currencyPush != nil {
		t.Fatalf("expected active equip not to push currency, got %+v", result.currencyPush)
	}
}

func TestHandlePacketClassicTownActiveItemExchangesCopperToSilver(t *testing.T) {
	store := session.NewStore()
	socketSession, role := seedSelectedRoleSessionInStore(t, store, "铜钱兑换测试")
	copper, ok := session.CapturedRoleItemTemplate("铜钱")
	if !ok {
		t.Fatal("expected copper template")
	}
	copper.Type = "背包"
	copper.Index = -1
	copper.Count = 1000
	granted, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, role.RoleID, copper)
	if !ok {
		t.Fatal("expected copper grant")
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownActiveItemReq,
		Seq: 10,
		Payload: mustJSON(t, classicTownActiveItemRequest{
			Type:  granted.Type,
			Index: granted.Index,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected ActiveItem to be handled")
	}
	if result.currencyPush == nil || result.currencyPush.Currencies["银元宝"] != 2 || result.currencyPush.Currencies["铜钱"] != 4000 {
		t.Fatalf("expected copper->silver currencies, got %+v", result.currencyPush)
	}
	items := itemInfosByName(result.itemInfos)
	if items["银元宝"].Count != 1 {
		t.Fatalf("expected silver item x1 push, got %+v", result.itemInfos)
	}
	if len(result.itemClears) != 1 || result.itemClears[0].Type != granted.Type || result.itemClears[0].Index != granted.Index {
		t.Fatalf("expected copper item clear, got %+v", result.itemClears)
	}
	if len(result.chatMessages) != 1 || !strings.Contains(result.chatMessages[0].Msg, "\u83b7\u5f97\u7269\u54c1:[\u94f6\u5143\u5b9d]x1") {
		t.Fatalf("expected captured copper exchange c_Speak message, got %+v", result.chatMessages)
	}
}

func TestHandlePacketClassicTownActiveItemRejectsRecoveryItemWhenStateIsFull(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	granted, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, session.RoleItem{
		Type:        "背包",
		Name:        "测试包子",
		ItemType:    "消耗品",
		Display:     "212.png",
		Description: "f_i_测试包子&24@消耗品&25@99&7@60&20@恢复气力",
		Count:       1,
		Index:       -1,
		ItemLevel:   1,
	})
	if !ok {
		t.Fatal("expected recovery item grant")
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownActiveItemReq,
		Seq: 11,
		Payload: mustJSON(t, classicTownActiveItemRequest{
			Type:  granted.Type,
			Index: granted.Index,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected ActiveItem full-state recovery item to be handled")
	}
	if result.roleState != nil || len(result.itemClears) != 0 {
		t.Fatalf("expected full-state recovery item not to mutate state/items, got state=%+v clears=%+v", result.roleState, result.itemClears)
	}
	if len(result.chatMessages) != 1 || !strings.Contains(result.chatMessages[0].Msg, "不需要使用") {
		t.Fatalf("expected full-state recovery item warning, got %+v", result.chatMessages)
	}
}

func TestHandlePacketClassicTownActiveItemLevel5GiftBoxPushesCapturedError(t *testing.T) {
	store := session.NewStore()
	socketSession, role := seedSelectedRoleSessionInStore(t, store, "gift-box-level-gate")
	giftBox, ok := session.CapturedRoleItemTemplate("\u0035\u7ea7\u793c\u76d2")
	if !ok {
		t.Fatal("expected captured level 5 gift box template")
	}
	giftBox.Type = classicTownBagContainerType
	giftBox.Index = -1
	giftBox.Count = 1
	granted, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, role.RoleID, giftBox)
	if !ok {
		t.Fatal("expected gift box grant")
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownActiveItemReq,
		Seq: 13,
		Payload: mustJSON(t, classicTownActiveItemRequest{
			Type:  granted.Type,
			Index: granted.Index,
		}),
	}, socketSession)

	if !result.handled || len(result.errorMessages) != 1 || len(result.chatMessages) != 0 || len(result.itemClears) != 0 || len(result.itemInfos) != 0 {
		t.Fatalf("expected captured level-gate c_Error only, got %+v", result)
	}
	errorMessage := result.errorMessages[0]
	if errorMessage.Msg != "\u89d2\u8272\u7b49\u7ea7\u5fc5\u987b\u5230\u8fbeLv5" ||
		!strings.Contains(errorMessage.SourceCapture, "ActiveItemByIndex(114)") ||
		!strings.Contains(errorMessage.SourceCapture, "c_Error") {
		t.Fatalf("expected captured ActiveItem level-gate c_Error, got %+v", errorMessage)
	}
}

func TestHandlePacketClassicTownActiveItemLevel1GiftBoxPushesCapturedRewardSpeaks(t *testing.T) {
	store := session.NewStore()
	socketSession, role := seedSelectedRoleSessionInStore(t, store, "level1-gift-speaks")
	giftBox, ok := session.CapturedRoleItemTemplate("\u0031\u7ea7\u793c\u76d2")
	if !ok {
		t.Fatal("expected captured level 1 gift box template")
	}
	giftBox.Type = classicTownBagContainerType
	giftBox.Index = -1
	giftBox.Count = 1
	granted, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, role.RoleID, giftBox)
	if !ok {
		t.Fatal("expected gift box grant")
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownActiveItemReq,
		Seq: 17,
		Payload: mustJSON(t, classicTownActiveItemRequest{
			Type:  granted.Type,
			Index: granted.Index,
		}),
	}, socketSession)

	if !result.handled || len(result.errorMessages) != 0 || len(result.itemClears) != 1 {
		t.Fatalf("expected captured level 1 gift success, got %+v", result)
	}
	for _, snippet := range []string{
		"\u83b7\u5f97\u7269\u54c1:[\u0035\u7ea7\u793c\u76d2]x1",
		"\u83b7\u5f97\u7269\u54c1:[\u004c\u907f\u602a\u7b26]x3",
		"\u83b7\u5f97\u7269\u54c1:[\u004c\u767e\u5e74\u4eba\u53c2\u679c]x1",
		"\u83b7\u5f97\u7269\u54c1:[\u004c\u767e\u5e74\u87e0\u6843]x1",
	} {
		if !packetChatMessagesContain(result.chatMessages, snippet) {
			t.Fatalf("expected captured reward message %q in %+v", snippet, result.chatMessages)
		}
	}
	items := itemInfosByName(result.itemInfos)
	for _, name := range []string{
		"\u0035\u7ea7\u793c\u76d2",
		"\u004c\u907f\u602a\u7b26",
		"\u004c\u767e\u5e74\u4eba\u53c2\u679c",
		"\u004c\u767e\u5e74\u87e0\u6843",
	} {
		if _, ok := items[name]; !ok {
			t.Fatalf("expected captured level 1 gift reward %s, got %+v", name, result.itemInfos)
		}
	}
}

func TestHandlePacketClassicTownActiveItemLevel5GiftBoxPushesCapturedRewardSpeaks(t *testing.T) {
	store := session.NewStore()
	socketSession, role := seedSelectedRoleSessionInStore(t, store, "level5-gift-speaks")
	store.SetRoleLevel(socketSession.playerBase.PlayerID, role.RoleID, 5)
	giftBox, ok := session.CapturedRoleItemTemplate("\u0035\u7ea7\u793c\u76d2")
	if !ok {
		t.Fatal("expected captured level 5 gift box template")
	}
	giftBox.Type = classicTownBagContainerType
	giftBox.Index = -1
	giftBox.Count = 1
	granted, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, role.RoleID, giftBox)
	if !ok {
		t.Fatal("expected gift box grant")
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownActiveItemReq,
		Seq: 16,
		Payload: mustJSON(t, classicTownActiveItemRequest{
			Type:  granted.Type,
			Index: granted.Index,
		}),
	}, socketSession)

	if !result.handled || len(result.errorMessages) != 0 || len(result.itemClears) != 1 {
		t.Fatalf("expected captured level 5 gift success, got %+v", result)
	}
	for _, snippet := range []string{
		"\u83b7\u5f97\u7269\u54c1:[\u004c\u521d\u9636\u7ecf\u9a8c\u5361]x1",
		"\u83b7\u5f97\u7269\u54c1:[\u004c\u82b1\u5377]x2",
		"\u83b7\u5f97\u7269\u54c1:[\u004c\u56de\u57ce\u5492]x3",
		"\u83b7\u5f97\u7269\u54c1:[\u0031\u0030\u7ea7\u793c\u76d2]x1",
	} {
		if !packetChatMessagesContain(result.chatMessages, snippet) {
			t.Fatalf("expected captured reward message %q in %+v", snippet, result.chatMessages)
		}
	}
	items := itemInfosByName(result.itemInfos)
	for _, name := range []string{
		"\u004c\u521d\u9636\u7ecf\u9a8c\u5361",
		"\u004c\u82b1\u5377",
		"\u004c\u56de\u57ce\u5492",
		"\u0031\u0030\u7ea7\u793c\u76d2",
	} {
		if _, ok := items[name]; !ok {
			t.Fatalf("expected captured level 5 gift reward %s, got %+v", name, result.itemInfos)
		}
	}
}

func TestHandlePacketClassicTownActiveItemLevel10GiftBoxPushesCapturedFullBagError(t *testing.T) {
	store := session.NewStore()
	socketSession, role := seedSelectedRoleSessionInStore(t, store, "level10-gift-full")
	store.SetRoleLevel(socketSession.playerBase.PlayerID, role.RoleID, 12)
	var ok bool
	for index := 0; index < 30; index += 1 {
		item := session.RoleItem{
			Type:        classicTownBagContainerType,
			Name:        fmt.Sprintf("level10-filler-%02d", index),
			ItemType:    "null",
			Display:     fmt.Sprintf("level10-filler-%02d.png", index),
			Description: fmt.Sprintf("f_i_level10-filler-%02d&24@material&25@1&20@test filler&103@0&104@0&105@&107@&108@0", index),
			Count:       1,
			Index:       index,
			ItemLevel:   1,
		}
		if index == 1 {
			item, ok = session.CapturedRoleItemTemplate("\u004c\u521d\u9636\u7ecf\u9a8c\u5361")
			if !ok {
				t.Fatal("expected captured exp card template")
			}
			item.Type = classicTownBagContainerType
			item.Index = index
			item.Count = 3
		}
		if index == 7 {
			item, ok = session.CapturedRoleItemTemplate("\u0031\u0030\u7ea7\u793c\u76d2")
			if !ok {
				t.Fatal("expected captured level 10 gift box template")
			}
			item.Type = classicTownBagContainerType
			item.Index = index
			item.Count = 1
		}
		if index == 8 {
			item, ok = session.CapturedRoleItemTemplate("\u004c\u82b1\u5377")
			if !ok {
				t.Fatal("expected captured flower roll template")
			}
			item.Type = classicTownBagContainerType
			item.Index = index
			item.Count = 9
		}
		if _, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, role.RoleID, item); !ok {
			t.Fatalf("expected bag filler grant at index %d", index)
		}
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownActiveItemReq,
		Seq: 14,
		Payload: mustJSON(t, classicTownActiveItemRequest{
			Type:  classicTownBagContainerType,
			Index: 7,
		}),
	}, socketSession)

	if !result.handled || len(result.errorMessages) != 1 || len(result.chatMessages) != 0 || len(result.itemClears) != 0 || len(result.itemInfos) != 1 {
		t.Fatalf("expected captured full-bag c_Error and gift refresh only, got %+v", result)
	}
	errorMessage := result.errorMessages[0]
	if errorMessage.Msg != "\u80cc\u5305\u7a7a\u95f4\u4e0d\u8db3" ||
		!strings.Contains(errorMessage.SourceCapture, "#3298") ||
		!strings.Contains(errorMessage.SourceCapture, "ActiveItemByIndex(114)") {
		t.Fatalf("expected captured level 10 gift full-bag c_Error, got %+v", errorMessage)
	}
	if result.itemInfos[0].Name != "\u0031\u0030\u7ea7\u793c\u76d2" || result.itemInfos[0].Index != 7 {
		t.Fatalf("expected level 10 gift box refresh at slot 7, got %+v", result.itemInfos)
	}
}

func TestHandlePacketClassicTownActiveItemLevel10GiftBoxPushesCapturedRewardSpeaks(t *testing.T) {
	store := session.NewStore()
	socketSession, role := seedSelectedRoleSessionInStore(t, store, "level10-gift-speaks")
	store.SetRoleLevel(socketSession.playerBase.PlayerID, role.RoleID, 12)
	grants := []struct {
		name  string
		index int
		count int
	}{
		{name: "\u004c\u521d\u9636\u7ecf\u9a8c\u5361", index: 1, count: 3},
		{name: "\u0031\u0030\u7ea7\u793c\u76d2", index: 7, count: 1},
		{name: "\u004c\u82b1\u5377", index: 8, count: 9},
	}
	for _, grant := range grants {
		item, ok := session.CapturedRoleItemTemplate(grant.name)
		if !ok {
			t.Fatalf("expected captured template %s", grant.name)
		}
		item.Type = classicTownBagContainerType
		item.Index = grant.index
		item.Count = grant.count
		if _, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, role.RoleID, item); !ok {
			t.Fatalf("expected captured %s grant", grant.name)
		}
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownActiveItemReq,
		Seq: 15,
		Payload: mustJSON(t, classicTownActiveItemRequest{
			Type:  classicTownBagContainerType,
			Index: 7,
		}),
	}, socketSession)

	if !result.handled || len(result.errorMessages) != 0 || len(result.itemClears) != 1 {
		t.Fatalf("expected captured level 10 gift success, got %+v", result)
	}
	if len(result.chatMessages) != 4 {
		t.Fatalf("expected four captured reward c_Speak messages, got %+v", result.chatMessages)
	}
	for _, snippet := range []string{
		"\u83b7\u5f97\u7269\u54c1:[\u004c\u521d\u9636\u7ecf\u9a8c\u5361]x1",
		"\u83b7\u5f97\u7269\u54c1:[\u004c\u82b1\u5377]x3",
		"\u83b7\u5f97\u7269\u54c1:[\u004c\u80cc\u5305\u8865\u4e01]x1",
		"\u83b7\u5f97\u7269\u54c1:[\u0031\u0035\u7ea7\u793c\u76d2]x1",
	} {
		if !packetChatMessagesContain(result.chatMessages, snippet) {
			t.Fatalf("expected captured reward message %q in %+v", snippet, result.chatMessages)
		}
	}
	items := itemInfosByName(result.itemInfos)
	for _, name := range []string{
		"\u004c\u521d\u9636\u7ecf\u9a8c\u5361",
		"\u004c\u82b1\u5377",
		"\u004c\u80cc\u5305\u8865\u4e01",
		"\u0031\u0035\u7ea7\u793c\u76d2",
	} {
		if _, ok := items[name]; !ok {
			t.Fatalf("expected captured level 10 gift reward %s, got %+v", name, result.itemInfos)
		}
	}
}

func TestHandlePacketClassicTownActiveItemBagPatchPushesCapturedCapacity(t *testing.T) {
	store := session.NewStore()
	socketSession, role := seedSelectedRoleSessionInStore(t, store, "bag-capacity-patch")
	patch, ok := session.CapturedRoleItemTemplate("\u004c\u80cc\u5305\u8865\u4e01")
	if !ok {
		t.Fatal("expected captured bag capacity patch template")
	}
	patch.Type = classicTownBagContainerType
	patch.Index = 7
	patch.Count = 1
	if _, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, role.RoleID, patch); !ok {
		t.Fatal("expected captured bag capacity patch grant")
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownActiveItemReq,
		Seq: 16,
		Payload: mustJSON(t, classicTownActiveItemRequest{
			Type:  classicTownBagContainerType,
			Index: 7,
		}),
	}, socketSession)

	if !result.handled || len(result.errorMessages) != 0 || len(result.itemInfos) != 0 {
		t.Fatalf("expected captured bag capacity patch success, got %+v", result)
	}
	if len(result.itemClears) != 1 || result.itemClears[0].Type != classicTownBagContainerType || result.itemClears[0].Index != 7 {
		t.Fatalf("expected captured patch slot 7 clear, got %+v", result.itemClears)
	}
	if result.containerCap == nil || result.containerCap.Type != classicTownBagContainerType || result.containerCap.Capacity != 30 {
		t.Fatalf("expected captured bag capacity 30 push, got %+v", result.containerCap)
	}
	if len(result.chatMessages) != 1 || !packetChatMessagesContain(result.chatMessages, "\u4f60\u7684\u80cc\u5305\u6269\u5927\u81f330\u683c") {
		t.Fatalf("expected captured bag expansion c_Speak, got %+v", result.chatMessages)
	}
}

func TestHandlePacketClassicTownActiveItemAppliesAvoidBuffAndRemoveBuff(t *testing.T) {
	store := session.NewStore()
	socketSession, role := seedSelectedRoleSessionInStore(t, store, "avoid-buff")
	item, ok := session.CapturedRoleItemTemplate("\u004c\u907f\u602a\u7b26")
	if !ok {
		t.Fatal("expected captured avoid buff item template")
	}
	item.Type = classicTownBagContainerType
	item.Index = -1
	item.Count = 1
	granted, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, role.RoleID, item)
	if !ok {
		t.Fatal("expected avoid buff item grant")
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownActiveItemReq,
		Seq: 12,
		Payload: mustJSON(t, classicTownActiveItemRequest{
			Type:  granted.Type,
			Index: granted.Index,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected ActiveItem avoid buff to be handled")
	}
	if len(result.townBuffs) != 1 {
		t.Fatalf("expected one BuffInfo push, got %+v", result.townBuffs)
	}
	buff := result.townBuffs[0]
	if buff.Handle != role.RoleID || buff.Name != "\u907f\u602a" || buff.Display != "574.png" {
		t.Fatalf("expected avoid buff push for role, got %+v", buff)
	}
	if buff.BattleOnly != 0 || buff.EndTime <= time.Now().UnixMilli() {
		t.Fatalf("expected active timed town buff, got %+v", buff)
	}
	if !strings.Contains(buff.Description, "\u70b9\u51fb\u53d6\u6d88\u8be5\u72b6\u6001") || !buff.Partial || !strings.Contains(buff.SourceCapture, "RemoveBuff") {
		t.Fatalf("expected source-clickable partial avoid buff evidence, got %+v", buff)
	}
	if len(result.itemClears) != 1 || result.itemClears[0].Type != granted.Type || result.itemClears[0].Index != granted.Index {
		t.Fatalf("expected consumed avoid buff item clear, got %+v", result.itemClears)
	}
	if len(result.chatMessages) != 1 || !strings.Contains(result.chatMessages[0].Msg, "\u4f7f\u7528\u4e86\u907f\u602a\u7b26") {
		t.Fatalf("expected source avoid-buff chat message, got %+v", result.chatMessages)
	}
	storedBuffs, ok := store.GetRoleTownBuffs(socketSession.playerBase.PlayerID, role.RoleID)
	if !ok || len(storedBuffs) != 1 || storedBuffs[0].Name != "\u907f\u602a" {
		t.Fatalf("expected persisted avoid buff, ok=%v buffs=%+v", ok, storedBuffs)
	}

	notExpiredClear := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownRemoveABateBuff,
		Seq:     14,
		Payload: mustJSON(t, map[string]any{}),
	}, socketSession)

	if !notExpiredClear.handled {
		t.Fatal("expected RemoveABateBuff to be handled")
	}
	if len(notExpiredClear.townBuffClears) != 0 {
		t.Fatalf("expected RemoveABateBuff not to clear unexpired buff, got %+v", notExpiredClear.townBuffClears)
	}
	storedBuffs, ok = store.GetRoleTownBuffs(socketSession.playerBase.PlayerID, role.RoleID)
	if !ok || len(storedBuffs) != 1 || storedBuffs[0].Name != "\u907f\u602a" {
		t.Fatalf("expected unexpired avoid buff to remain, ok=%v buffs=%+v", ok, storedBuffs)
	}

	clear := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownRemoveBuffReq,
		Seq: 15,
		Payload: mustJSON(t, classicTownRemoveBuffRequest{
			Name: "\u907f\u602a",
		}),
	}, socketSession)

	if !clear.handled {
		t.Fatal("expected RemoveBuff to be handled")
	}
	if len(clear.townBuffClears) != 1 || clear.townBuffClears[0].Handle != role.RoleID || clear.townBuffClears[0].Name != "\u907f\u602a" {
		t.Fatalf("expected clearBuffInfo push, got %+v", clear.townBuffClears)
	}
	storedBuffs, ok = store.GetRoleTownBuffs(socketSession.playerBase.PlayerID, role.RoleID)
	if !ok || len(storedBuffs) != 0 {
		t.Fatalf("expected avoid buff removed from store, ok=%v buffs=%+v", ok, storedBuffs)
	}
}

func TestHandlePacketClassicTownActiveItemAppliesCapturedInitialExperienceBuff(t *testing.T) {
	store := session.NewStore()
	socketSession, role := seedSelectedRoleSessionInStore(t, store, "initial-exp-card")
	item, ok := session.CapturedRoleItemTemplate("\u004c\u521d\u9636\u7ecf\u9a8c\u5361")
	if !ok {
		t.Fatal("expected captured initial experience card template")
	}
	item.Type = classicTownBagContainerType
	item.Index = -1
	item.Count = 5
	granted, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, role.RoleID, item)
	if !ok {
		t.Fatal("expected initial experience card grant")
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownActiveItemReq,
		Seq: 12,
		Payload: mustJSON(t, classicTownActiveItemRequest{
			Type:  granted.Type,
			Index: granted.Index,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected ActiveItem initial experience card to be handled")
	}
	if len(result.townBuffs) != 1 {
		t.Fatalf("expected one double-exp BuffInfo push, got %+v", result.townBuffs)
	}
	buff := result.townBuffs[0]
	if buff.Handle != role.RoleID || buff.Name != "\u53cc\u500d\u7ecf\u9a8c" || buff.Display != "567.png" {
		t.Fatalf("expected captured double-exp buff push, got %+v", buff)
	}
	if buff.BattleOnly != 0 || buff.EndTime <= time.Now().UnixMilli() || !buff.Partial || !strings.Contains(buff.SourceCapture, "ActiveItemByIndex(114)") {
		t.Fatalf("expected source-backed partial double-exp buff, got %+v", buff)
	}
	if len(result.itemInfos) != 1 || result.itemInfos[0].Name != "\u004c\u521d\u9636\u7ecf\u9a8c\u5361" || result.itemInfos[0].Count != 4 {
		t.Fatalf("expected initial experience card count refresh to 4, got %+v", result.itemInfos)
	}
	if len(result.chatMessages) != 1 || !strings.Contains(result.chatMessages[0].Msg, "\u83b7\u5f97\u53cc\u500d\u7ecf\u9a8c\u65f6\u95f41\u5c0f\u65f6") {
		t.Fatalf("expected captured double-exp c_Speak message, got %+v", result.chatMessages)
	}
	storedBuffs, ok := store.GetRoleTownBuffs(socketSession.playerBase.PlayerID, role.RoleID)
	if !ok || len(storedBuffs) != 1 || storedBuffs[0].Name != "\u53cc\u500d\u7ecf\u9a8c" {
		t.Fatalf("expected persisted double-exp buff, ok=%v buffs=%+v", ok, storedBuffs)
	}
}

func TestHandlePacketClassicTownActiveItemAppliesCapturedAdvancedExperienceBuff(t *testing.T) {
	store := session.NewStore()
	socketSession, role := seedSelectedRoleSessionInStore(t, store, "advanced-exp-card")
	item, ok := session.CapturedRoleItemTemplate("\u004c\u8fdb\u9636\u7ecf\u9a8c\u5361")
	if !ok {
		t.Fatal("expected captured advanced experience card template")
	}
	item.Type = classicTownBagContainerType
	item.Index = -1
	item.Count = 4
	granted, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, role.RoleID, item)
	if !ok {
		t.Fatal("expected advanced experience card grant")
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownActiveItemReq,
		Seq: 12,
		Payload: mustJSON(t, classicTownActiveItemRequest{
			Type:  granted.Type,
			Index: granted.Index,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected ActiveItem advanced experience card to be handled")
	}
	if len(result.townBuffs) != 1 {
		t.Fatalf("expected one double-exp BuffInfo push, got %+v", result.townBuffs)
	}
	buff := result.townBuffs[0]
	if buff.Handle != role.RoleID || buff.Name != "\u53cc\u500d\u7ecf\u9a8c" || buff.Display != "567.png" {
		t.Fatalf("expected captured double-exp buff push, got %+v", buff)
	}
	if buff.BattleOnly != 0 || buff.EndTime <= time.Now().UnixMilli() || !buff.Partial || !strings.Contains(buff.SourceCapture, "#2844/#2847/#2848") {
		t.Fatalf("expected source-backed partial advanced double-exp buff, got %+v", buff)
	}
	if len(result.itemInfos) != 1 || result.itemInfos[0].Name != "\u004c\u8fdb\u9636\u7ecf\u9a8c\u5361" || result.itemInfos[0].Count != 3 {
		t.Fatalf("expected advanced experience card count refresh to 3, got %+v", result.itemInfos)
	}
	if len(result.chatMessages) != 1 || !strings.Contains(result.chatMessages[0].Msg, "\u83b7\u5f97\u53cc\u500d\u7ecf\u9a8c\u65f6\u95f43\u5c0f\u65f6") {
		t.Fatalf("expected captured advanced double-exp c_Speak message, got %+v", result.chatMessages)
	}
	storedBuffs, ok := store.GetRoleTownBuffs(socketSession.playerBase.PlayerID, role.RoleID)
	if !ok || len(storedBuffs) != 1 || storedBuffs[0].Name != "\u53cc\u500d\u7ecf\u9a8c" {
		t.Fatalf("expected persisted double-exp buff, ok=%v buffs=%+v", ok, storedBuffs)
	}
}

func TestHandlePacketClassicTownBuyBackListAndTakeDeductsCopper(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	beforeCopper := roleCurrenciesOrEmpty(
		store,
		socketSession.playerBase.PlayerID,
		socketSession.selectedRole.RoleID,
	)["铜钱"]

	list := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetBuyBackListReq,
		Seq:     2,
		Payload: mustJSON(t, map[string]any{}),
	}, socketSession)
	if !list.handled || list.buyBackRefresh == nil || !list.buyBackRefresh.Partial {
		t.Fatalf("expected captured partial BuyBack list refresh, got %+v", list)
	}
	if len(list.buyBackInfos) != 2 {
		t.Fatalf("expected two captured BuyBack rows, got %+v", list.buyBackInfos)
	}
	first := list.buyBackInfos[0]
	if first.Index != 0 || first.Item.Name != "藤条" || first.Item.Display != "90.png" || first.Item.Count != 5 || !first.Partial {
		t.Fatalf("expected first captured BuyBack row for 藤条 x5, got %+v", first)
	}
	if len(first.EquivalentInfos) != 1 || first.EquivalentInfos[0].Name != "铜钱" || first.EquivalentInfos[0].Count != 155 {
		t.Fatalf("expected first BuyBack row to cost 铜钱 x155, got %+v", first.EquivalentInfos)
	}
	second := list.buyBackInfos[1]
	if second.Index != 1 || second.Item.Name != "花瓣" || second.Item.Display != "89.png" || second.Item.Count != 6 {
		t.Fatalf("expected second captured BuyBack row for 花瓣 x6, got %+v", second)
	}

	take := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownBuyBackReq,
		Seq: 3,
		Payload: mustJSON(t, classicTownBuyBackRequest{
			Index: 0,
		}),
	}, socketSession)
	if !take.handled || take.currencyPush == nil || take.buyBackRefresh == nil {
		t.Fatalf("expected BuyBack to mutate currency and refresh list, got %+v", take)
	}
	if take.currencyPush.Currencies["铜钱"] != beforeCopper-155 {
		t.Fatalf("expected BuyBack to deduct 铜钱 155, before=%d got %+v", beforeCopper, take.currencyPush)
	}
	items := itemInfosByName(take.itemInfos)
	if items["藤条"].Count != 5 || items["藤条"].Display != "90.png" {
		t.Fatalf("expected BuyBack to grant 藤条 x5, got %+v", take.itemInfos)
	}
	if !packetChatMessagesContain(take.chatMessages, "回购了【藤条】x5") {
		t.Fatalf("expected BuyBack success system message, got %+v", take.chatMessages)
	}
	if len(take.buyBackInfos) != 1 || take.buyBackInfos[0].Index != 1 {
		t.Fatalf("expected BuyBack refresh to remove taken row 0, got %+v", take.buyBackInfos)
	}

	listAgain := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetBuyBackListReq,
		Seq:     4,
		Payload: mustJSON(t, map[string]any{}),
	}, socketSession)
	if len(listAgain.buyBackInfos) != 1 || listAgain.buyBackInfos[0].Index != 1 {
		t.Fatalf("expected subsequent BuyBack list to keep only row 1, got %+v", listAgain.buyBackInfos)
	}
}

func TestHandlePacketClassicTownHealerAnswerReportsNearlyFull(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownAnswerReq,
		Seq: 11,
		Payload: mustJSON(t, classicTownAnswerRequest{
			Handle:       "2520542613299551",
			MsgHandle:    "1",
			AnswerHandle: "2",
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected healer Answer to be handled")
	}
	if result.answerSpeak == nil || !strings.Contains(result.answerSpeak.Msg, "你几乎不需要治疗了") {
		t.Fatalf("expected source healer nearly-full reply, got %+v", result.answerSpeak)
	}
	if result.roleState != nil {
		t.Fatalf("expected nearly-full healer not to push roleState, got %+v", result.roleState)
	}
}

func TestHandlePacketClassicTownChatSendFormatsSourceChannels(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)

	worldResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownChatSendReq,
		Seq: 11,
		Payload: mustJSON(t, classicTownChatSendRequest{
			Channel: "world",
			Msg:     "大家好",
		}),
	}, socketSession)
	if !worldResult.handled || len(worldResult.chatMessages) != 1 {
		t.Fatalf("expected world chat push, got %+v", worldResult)
	}
	if worldResult.chatMessages[0].Channel != "world" || worldResult.chatMessages[0].Name != "技能测试" || worldResult.chatMessages[0].Msg != "大家好" {
		t.Fatalf("expected source sayP-style world message, got %+v", worldResult.chatMessages[0])
	}

	whisperResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownChatSendReq,
		Seq: 12,
		Payload: mustJSON(t, classicTownChatSendRequest{
			Channel:    "3",
			TargetName: "小明",
			Msg:        "在吗",
		}),
	}, socketSession)
	if len(whisperResult.chatMessages) != 1 || whisperResult.chatMessages[0].Channel != "whisper" || !whisperResult.chatMessages[0].Outgoing || whisperResult.chatMessages[0].TargetName != "小明" {
		t.Fatalf("expected source sayW-style outgoing whisper, got %+v", whisperResult.chatMessages)
	}
}

func TestHandlePacketClassicTownContainerMoveMovesBagItemBySlot(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	sourceIndex := 19
	targetIndex := 0
	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownContainerMove,
		Seq: 2,
		Payload: mustJSON(t, classicTownContainerMoveRequest{
			SourceType:  "背包",
			TargetType:  "背包",
			SourceIndex: &sourceIndex,
			TargetIndex: &targetIndex,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected bag ContainerMove to be handled")
	}
	if len(result.itemClears) != 2 {
		t.Fatalf("expected source/target bag clears, got %+v", result.itemClears)
	}
	if len(result.itemInfos) != 1 || result.itemInfos[0].Type != "背包" || result.itemInfos[0].Name != "铁斧" || result.itemInfos[0].Index != 0 {
		t.Fatalf("expected moved axe push at bag slot 0, got %+v", result.itemInfos)
	}

	listResult := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetItemListReq,
		Seq:     3,
		Payload: mustJSON(t, classicTownContainerRequest{Type: "背包"}),
	}, socketSession)
	if len(listResult.itemInfos) != 1 || listResult.itemInfos[0].Name != "铁斧" || listResult.itemInfos[0].Index != 0 {
		t.Fatalf("expected moved axe to persist in bag slot 0, got %+v", listResult.itemInfos)
	}
}

func TestHandlePacketClassicTownContainerMoveMovesMallItemToBagBySlot(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	mallItem := session.RoleItem{
		Type:      "商城",
		Name:      "商城测试物品",
		ItemType:  "own",
		Display:   "1.png",
		Count:     2,
		Index:     0,
		ItemLevel: 1,
	}
	if _, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, mallItem); !ok {
		t.Fatal("expected mall item grant to succeed")
	}

	sourceIndex := 0
	targetIndex := 0
	count := 2
	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownContainerMove,
		Seq: 2,
		Payload: mustJSON(t, classicTownContainerMoveRequest{
			SourceType:  "商城",
			TargetType:  "背包",
			SourceIndex: &sourceIndex,
			TargetIndex: &targetIndex,
			Count:       &count,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected mall-to-bag ContainerMove to be handled")
	}
	if len(result.itemClears) != 2 {
		t.Fatalf("expected mall source and bag target clears, got %+v", result.itemClears)
	}
	if result.itemClears[0].Type != "商城" || result.itemClears[0].Index != 0 {
		t.Fatalf("expected mall source slot 0 to be cleared first, got %+v", result.itemClears)
	}
	if len(result.itemInfos) != 1 || result.itemInfos[0].Type != "背包" || result.itemInfos[0].Name != "商城测试物品" || result.itemInfos[0].Index != 0 || result.itemInfos[0].Count != 2 {
		t.Fatalf("expected moved mall item push at bag slot 0, got %+v", result.itemInfos)
	}

	mallList := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetItemListReq,
		Seq:     3,
		Payload: mustJSON(t, classicTownContainerRequest{Type: "商城"}),
	}, socketSession)
	if len(mallList.itemInfos) != 0 {
		t.Fatalf("expected mall container to be empty after move, got %+v", mallList.itemInfos)
	}
	bagList := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetItemListReq,
		Seq:     4,
		Payload: mustJSON(t, classicTownContainerRequest{Type: "背包"}),
	}, socketSession)
	itemsByIndex := map[int]classicTownItemInfoPush{}
	for _, item := range bagList.itemInfos {
		itemsByIndex[item.Index] = item
	}
	if itemsByIndex[0].Name != "商城测试物品" || itemsByIndex[0].Count != 2 {
		t.Fatalf("expected moved mall item to persist in bag slot 0, got %+v", bagList.itemInfos)
	}
}

func TestHandlePacketClassicTownContainerMoveMovesMallItemWithinMallBySlot(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	mallItem := session.RoleItem{
		Type:      "商城",
		Name:      "商城换格测试物品",
		ItemType:  "own",
		Display:   "1.png",
		Count:     1,
		Index:     0,
		ItemLevel: 1,
	}
	if _, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, mallItem); !ok {
		t.Fatal("expected mall item grant to succeed")
	}

	sourceIndex := 0
	targetIndex := 1
	count := 1
	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownContainerMove,
		Seq: 2,
		Payload: mustJSON(t, classicTownContainerMoveRequest{
			SourceType:  "商城",
			TargetType:  "商城",
			SourceIndex: &sourceIndex,
			TargetIndex: &targetIndex,
			Count:       &count,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected mall ContainerMove to be handled")
	}
	if len(result.itemClears) != 2 || result.itemClears[0].Type != "商城" || result.itemClears[0].Index != 0 || result.itemClears[1].Type != "商城" || result.itemClears[1].Index != 1 {
		t.Fatalf("expected mall source/target clears, got %+v", result.itemClears)
	}
	if len(result.itemInfos) != 1 || result.itemInfos[0].Type != "商城" || result.itemInfos[0].Name != "商城换格测试物品" || result.itemInfos[0].Index != 1 {
		t.Fatalf("expected moved mall item push at mall slot 1, got %+v", result.itemInfos)
	}

	mallList := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetItemListReq,
		Seq:     3,
		Payload: mustJSON(t, classicTownContainerRequest{Type: "商城"}),
	}, socketSession)
	if len(mallList.itemInfos) != 1 || mallList.itemInfos[0].Name != "商城换格测试物品" || mallList.itemInfos[0].Index != 1 {
		t.Fatalf("expected moved mall item to persist at mall slot 1, got %+v", mallList.itemInfos)
	}
}

func TestHandlePacketClassicTownFinishingContainerStacksBagItems(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	roleID := socketSession.selectedRole.RoleID
	playerID := socketSession.playerBase.PlayerID

	meat, ok := session.CapturedRoleItemTemplate("肉")
	if !ok {
		t.Fatal("expected captured 肉 template")
	}
	meat.Type = "背包"
	meat.Index = 7
	meat.Count = 2
	if _, ok := store.GrantRoleItem(playerID, roleID, meat); !ok {
		t.Fatal("expected first 肉 grant to succeed")
	}
	meat.Index = 22
	meat.Count = 3
	if _, ok := store.GrantRoleItem(playerID, roleID, meat); !ok {
		t.Fatal("expected second 肉 grant to succeed")
	}

	skull, ok := session.CapturedRoleItemTemplate("头骨")
	if !ok {
		t.Fatal("expected captured 头骨 template")
	}
	skull.Type = "背包"
	skull.Index = 25
	skull.Count = 1
	if _, ok := store.GrantRoleItem(playerID, roleID, skull); !ok {
		t.Fatal("expected 头骨 grant to succeed")
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownFinishingContainerReq,
		Seq:     2,
		Payload: mustJSON(t, classicTownContainerRequest{Type: "背包"}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected FinishingContainer to be handled")
	}
	expectedClears := []int{7, 19, 22, 25}
	if len(result.itemClears) != len(expectedClears) {
		t.Fatalf("expected compact clears %+v, got %+v", expectedClears, result.itemClears)
	}
	for index, expected := range expectedClears {
		if result.itemClears[index].Type != "背包" || result.itemClears[index].Index != expected {
			t.Fatalf("expected compact clear slot %d at position %d, got %+v", expected, index, result.itemClears)
		}
	}
	resultItemsByIndex := map[int]classicTownItemInfoPush{}
	for _, item := range result.itemInfos {
		resultItemsByIndex[item.Index] = item
	}
	if resultItemsByIndex[0].Name != "肉" || resultItemsByIndex[0].Count != 5 ||
		resultItemsByIndex[1].Name != "铁斧" || resultItemsByIndex[2].Name != "头骨" {
		t.Fatalf("expected compact item pushes at slots 0..2, got %+v", result.itemInfos)
	}

	listResult := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetItemListReq,
		Seq:     3,
		Payload: mustJSON(t, classicTownContainerRequest{Type: "背包"}),
	}, socketSession)
	itemsByIndex := map[int]classicTownItemInfoPush{}
	for _, item := range listResult.itemInfos {
		itemsByIndex[item.Index] = item
	}
	if itemsByIndex[0].Name != "肉" || itemsByIndex[0].Count != 5 ||
		itemsByIndex[1].Name != "铁斧" || itemsByIndex[2].Name != "头骨" {
		t.Fatalf("expected compact stacked bag to persist, got %+v", listResult.itemInfos)
	}
	for _, emptyIndex := range expectedClears {
		if _, exists := itemsByIndex[emptyIndex]; exists {
			t.Fatalf("expected old slot %d to be empty after compact finish, got %+v", emptyIndex, listResult.itemInfos)
		}
	}
}

func TestHandlePacketClassicTownDestroyItemConsumesRequestedCount(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	roleID := socketSession.selectedRole.RoleID
	playerID := socketSession.playerBase.PlayerID
	template, ok := session.CapturedRoleItemTemplate("肉")
	if !ok {
		t.Fatal("expected captured 肉 template")
	}
	template.Type = "背包"
	template.Index = 0
	template.Count = 5
	if _, ok := store.GrantRoleItem(playerID, roleID, template); !ok {
		t.Fatal("expected 肉 grant to succeed")
	}

	sourceIndex := 0
	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownDestroyItemReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownDestroyItemRequest{
			Type:  "背包",
			Index: sourceIndex,
			Count: 2,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected DestroyItem to be handled")
	}
	if len(result.itemClears) != 0 {
		t.Fatalf("expected partial destroy not to clear slot, got %+v", result.itemClears)
	}
	if len(result.itemInfos) != 1 || result.itemInfos[0].Name != "肉" || result.itemInfos[0].Index != sourceIndex || result.itemInfos[0].Count != 3 {
		t.Fatalf("expected remaining 肉 count 3 in same slot, got %+v", result.itemInfos)
	}

	listResult := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetItemListReq,
		Seq:     3,
		Payload: mustJSON(t, classicTownContainerRequest{Type: "背包"}),
	}, socketSession)
	items := itemInfosByName(listResult.itemInfos)
	if items["肉"].Index != sourceIndex || items["肉"].Count != 3 {
		t.Fatalf("expected remaining 肉 to persist at count 3, got %+v", listResult.itemInfos)
	}
}

func TestHandlePacketClassicTownDestroyItemClearsBagSlot(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	roleID := socketSession.selectedRole.RoleID
	playerID := socketSession.playerBase.PlayerID
	template, ok := session.CapturedRoleItemTemplate("肉")
	if !ok {
		t.Fatal("expected captured 肉 template")
	}
	template.Type = "背包"
	template.Index = 0
	template.Count = 2
	if _, ok := store.GrantRoleItem(playerID, roleID, template); !ok {
		t.Fatal("expected 肉 grant to succeed")
	}

	sourceIndex := 0
	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownDestroyItemReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownDestroyItemRequest{
			Type:  "背包",
			Index: sourceIndex,
			Count: 2,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected DestroyItem to be handled")
	}
	if len(result.itemClears) != 1 || result.itemClears[0].Type != "背包" || result.itemClears[0].Index != sourceIndex {
		t.Fatalf("expected bag source slot clear, got %+v", result.itemClears)
	}
	if len(result.itemInfos) != 0 {
		t.Fatalf("expected no item info push after full destroy, got %+v", result.itemInfos)
	}

	listResult := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetItemListReq,
		Seq:     3,
		Payload: mustJSON(t, classicTownContainerRequest{Type: "背包"}),
	}, socketSession)
	for _, item := range listResult.itemInfos {
		if item.Name == "肉" && item.Index == sourceIndex {
			t.Fatalf("expected destroyed slot to stay empty, got %+v", item)
		}
	}
}

func TestHandlePacketClassicTownSaleItemAddsCopperAndConsumesCount(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	roleID := socketSession.selectedRole.RoleID
	playerID := socketSession.playerBase.PlayerID
	template, ok := session.CapturedRoleItemTemplate("肉")
	if !ok {
		t.Fatal("expected captured 肉 template")
	}
	template.Type = "背包"
	template.Index = 0
	template.Count = 5
	if _, ok := store.GrantRoleItem(playerID, roleID, template); !ok {
		t.Fatal("expected 肉 grant to succeed")
	}

	sourceIndex := 0
	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownSaleItemReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownSaleItemRequest{
			ShopID: "item:7000542609490978",
			Type:   "背包",
			Index:  sourceIndex,
			Count:  2,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected SaleItem to be handled")
	}
	if result.currencyPush == nil || result.currencyPush.Currencies["铜钱"] != 5024 {
		t.Fatalf("expected copper +24 after selling 肉 x2, got %+v", result.currencyPush)
	}
	if len(result.itemClears) != 0 {
		t.Fatalf("expected partial sale not to clear slot, got %+v", result.itemClears)
	}
	if len(result.itemInfos) != 1 || result.itemInfos[0].Name != "肉" || result.itemInfos[0].Index != sourceIndex || result.itemInfos[0].Count != 3 {
		t.Fatalf("expected remaining 肉 count 3 in same slot, got %+v", result.itemInfos)
	}
	if result.buyBackRefresh == nil || !result.buyBackRefresh.Partial {
		t.Fatalf("expected SaleItem to refresh buyback list, got %+v", result.buyBackRefresh)
	}
	soldBuyBackIndex := -1
	for _, entry := range result.buyBackInfos {
		if entry.Item.Name == template.Name &&
			entry.Item.Count == 2 &&
			len(entry.EquivalentInfos) == 1 &&
			entry.EquivalentInfos[0].Name == "铜钱" &&
			entry.EquivalentInfos[0].Count == 24 {
			soldBuyBackIndex = entry.Index
			break
		}
	}
	if soldBuyBackIndex < len(classicTownSourceBuyBackEntries) {
		t.Fatalf("expected dynamic sold buyback row after static captures, index=%d rows=%+v", soldBuyBackIndex, result.buyBackInfos)
	}

	takeSold := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownBuyBackReq,
		Seq: 3,
		Payload: mustJSON(t, classicTownBuyBackRequest{
			Index: soldBuyBackIndex,
		}),
	}, socketSession)
	if !takeSold.handled || takeSold.currencyPush == nil || takeSold.buyBackRefresh == nil {
		t.Fatalf("expected dynamic BuyBack to mutate currency and refresh list, got %+v", takeSold)
	}
	if takeSold.currencyPush.Currencies["铜钱"] != result.currencyPush.Currencies["铜钱"]-24 {
		t.Fatalf("expected dynamic BuyBack to deduct sale price 24, got %+v", takeSold.currencyPush)
	}
	restoredItem, ok := store.GetRoleItem(playerID, roleID, "背包", sourceIndex)
	if !ok || restoredItem.Name != template.Name || restoredItem.Count != 5 {
		t.Fatalf("expected dynamic BuyBack to restore sold stack, ok=%v item=%+v", ok, restoredItem)
	}
	for _, entry := range takeSold.buyBackInfos {
		if entry.Index == soldBuyBackIndex {
			t.Fatalf("expected dynamic BuyBack row to disappear after take, got %+v", takeSold.buyBackInfos)
		}
	}
}

func TestHandlePacketClassicTownSaleItemRejectsNoSaleItem(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	roleID := socketSession.selectedRole.RoleID
	playerID := socketSession.playerBase.PlayerID
	item := session.RoleItem{
		Type:        "背包",
		Name:        "测试不可售物",
		ItemType:    "null",
		Display:     "1.png",
		Description: "f_i_测试不可售物&24@材料&25@99&20@测试&108@0",
		Count:       2,
		Index:       0,
		ItemLevel:   1,
	}
	if _, ok := store.GrantRoleItem(playerID, roleID, item); !ok {
		t.Fatal("expected no-sale item grant to succeed")
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownSaleItemReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownSaleItemRequest{
			ShopID: "item:7000542609490978",
			Type:   "背包",
			Index:  0,
			Count:  1,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected SaleItem reject to be handled")
	}
	if result.currencyPush != nil || len(result.itemClears) != 0 || len(result.itemInfos) != 0 {
		t.Fatalf("expected no mutation for no-sale item, got currency=%+v clears=%+v infos=%+v", result.currencyPush, result.itemClears, result.itemInfos)
	}
	if len(result.chatMessages) != 1 || result.chatMessages[0].Msg != "该物品无法出售。" {
		t.Fatalf("expected no-sale warning chat, got %+v", result.chatMessages)
	}
}

func TestHandlePacketClassicTownContainerMoveUnequipsEquipmentToBag(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	equipResult := store.EquipRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, "背包", 19, 1)
	if !equipResult.Found || !equipResult.Equipped {
		t.Fatalf("expected starter axe equip success, got %+v", equipResult)
	}
	socketSession.selectedRole = &equipResult.Role
	socketSession.playerBase = &equipResult.PlayerBase

	sourceIndex := 3
	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownContainerMove,
		Seq: 2,
		Payload: mustJSON(t, classicTownContainerMoveRequest{
			SourceType:  "装备",
			TargetType:  "背包",
			SourceIndex: &sourceIndex,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected equipment ContainerMove to be handled")
	}
	if result.createPlayer == nil || strings.Contains(result.createPlayer.SourceQuery, "w8=5") {
		t.Fatalf("expected unequip to push createPlayer without weapon appearance, got %+v", result.createPlayer)
	}
	if result.rolePhysique == nil {
		t.Fatal("expected unequip to push rolePhysique")
	}
	if len(result.itemClears) != 2 {
		t.Fatalf("expected equipment and target bag clears, got %+v", result.itemClears)
	}
	if len(result.itemInfos) != 1 || result.itemInfos[0].Type != "背包" || result.itemInfos[0].Name != "铁斧" || result.itemInfos[0].Index != 0 {
		t.Fatalf("expected unequipped axe at first empty bag slot, got %+v", result.itemInfos)
	}
}

func TestHandlePacketClassicTownContainerMoveStacksSameBagItems(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	roleID := socketSession.selectedRole.RoleID
	playerID := socketSession.playerBase.PlayerID
	template, ok := session.CapturedRoleItemTemplate("肉")
	if !ok {
		t.Fatal("expected captured 肉 template")
	}
	template.Type = "背包"
	template.Index = 0
	template.Count = 2
	if _, ok := store.GrantRoleItem(playerID, roleID, template); !ok {
		t.Fatal("expected first 肉 grant to succeed")
	}
	template.Index = 1
	template.Count = 3
	if _, ok := store.GrantRoleItem(playerID, roleID, template); !ok {
		t.Fatal("expected second 肉 grant to succeed")
	}

	sourceIndex := 1
	targetIndex := 0
	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownContainerMove,
		Seq: 2,
		Payload: mustJSON(t, classicTownContainerMoveRequest{
			SourceType:  "背包",
			TargetType:  "背包",
			SourceIndex: &sourceIndex,
			TargetIndex: &targetIndex,
		}),
	}, socketSession)
	if !result.handled {
		t.Fatal("expected bag stack ContainerMove to be handled")
	}
	if len(result.itemClears) != 2 {
		t.Fatalf("expected source/target bag clears for stack, got %+v", result.itemClears)
	}
	if len(result.itemInfos) != 1 || result.itemInfos[0].Name != "肉" || result.itemInfos[0].Index != 0 || result.itemInfos[0].Count != 5 {
		t.Fatalf("expected stacked 肉 push at slot 0 count 5, got %+v", result.itemInfos)
	}

	listResult := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetItemListReq,
		Seq:     3,
		Payload: mustJSON(t, classicTownContainerRequest{Type: "背包"}),
	}, socketSession)
	items := itemInfosByName(listResult.itemInfos)
	if items["肉"].Index != 0 || items["肉"].Count != 5 {
		t.Fatalf("expected stacked 肉 to persist at slot 0 count 5, got %+v", listResult.itemInfos)
	}
}

func TestHandlePacketClassicTownContainerMoveStacksSameWarehouseItems(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	roleID := socketSession.selectedRole.RoleID
	playerID := socketSession.playerBase.PlayerID
	template, ok := session.CapturedRoleItemTemplate("\u8089")
	if !ok {
		t.Fatal("expected captured meat template")
	}
	template.Type = classicWarehouseContainerType
	template.Index = 0
	template.Count = 2
	if _, ok := store.GrantRoleItem(playerID, roleID, template); !ok {
		t.Fatal("expected first warehouse meat grant to succeed")
	}
	template.Index = 1
	template.Count = 3
	if _, ok := store.GrantRoleItem(playerID, roleID, template); !ok {
		t.Fatal("expected second warehouse meat grant to succeed")
	}

	sourceIndex := 1
	targetIndex := 0
	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownContainerMove,
		Seq: 2,
		Payload: mustJSON(t, classicTownContainerMoveRequest{
			SourceType:  classicWarehouseContainerType,
			TargetType:  classicWarehouseContainerType,
			SourceIndex: &sourceIndex,
			TargetIndex: &targetIndex,
		}),
	}, socketSession)
	if !result.handled {
		t.Fatal("expected warehouse stack ContainerMove to be handled")
	}
	if len(result.itemClears) != 2 {
		t.Fatalf("expected source/target warehouse clears for stack, got %+v", result.itemClears)
	}
	if len(result.itemInfos) != 1 || result.itemInfos[0].Type != classicWarehouseContainerType || result.itemInfos[0].Name != "\u8089" || result.itemInfos[0].Index != 0 || result.itemInfos[0].Count != 5 {
		t.Fatalf("expected stacked warehouse meat push at slot 0 count 5, got %+v", result.itemInfos)
	}

	listResult := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetItemListReq,
		Seq:     3,
		Payload: mustJSON(t, classicTownContainerRequest{Type: classicWarehouseContainerType}),
	}, socketSession)
	itemsByIndex := map[int]classicTownItemInfoPush{}
	for _, item := range listResult.itemInfos {
		itemsByIndex[item.Index] = item
	}
	if itemsByIndex[0].Name != "\u8089" || itemsByIndex[0].Count != 5 {
		t.Fatalf("expected stacked warehouse meat to persist at slot 0 count 5, got %+v", listResult.itemInfos)
	}
	if _, exists := itemsByIndex[1]; exists {
		t.Fatalf("expected warehouse slot 1 to be cleared after stacking, got %+v", listResult.itemInfos)
	}
}

func TestHandlePacketClassicTownContainerMoveSplitsBagItemByCount(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	roleID := socketSession.selectedRole.RoleID
	playerID := socketSession.playerBase.PlayerID
	template, ok := session.CapturedRoleItemTemplate("肉")
	if !ok {
		t.Fatal("expected captured 肉 template")
	}
	template.Type = "背包"
	template.Index = 1
	template.Count = 5
	if _, ok := store.GrantRoleItem(playerID, roleID, template); !ok {
		t.Fatal("expected 肉 grant to succeed")
	}

	sourceIndex := 1
	targetIndex := 2
	moveCount := 2
	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownContainerMove,
		Seq: 2,
		Payload: mustJSON(t, classicTownContainerMoveRequest{
			SourceType:  "背包",
			TargetType:  "背包",
			SourceIndex: &sourceIndex,
			TargetIndex: &targetIndex,
			Count:       &moveCount,
		}),
	}, socketSession)
	if !result.handled {
		t.Fatal("expected bag split ContainerMove to be handled")
	}
	if len(result.itemClears) != 1 || result.itemClears[0].Index != 2 {
		t.Fatalf("expected only target clear for split move, got %+v", result.itemClears)
	}
	if len(result.itemInfos) != 2 {
		t.Fatalf("expected source+target item infos for split move, got %+v", result.itemInfos)
	}

	listResult := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetItemListReq,
		Seq:     3,
		Payload: mustJSON(t, classicTownContainerRequest{Type: "背包"}),
	}, socketSession)
	itemsByIndex := map[int]classicTownItemInfoPush{}
	for _, item := range listResult.itemInfos {
		itemsByIndex[item.Index] = item
	}
	if itemsByIndex[1].Count != 3 || itemsByIndex[2].Count != 2 {
		t.Fatalf("expected split stacks count 3/2 at slots 1/2, got %+v", listResult.itemInfos)
	}
}

func TestHandlePacketClassicTownYeMeiGiftGrantsSourceArmorToBag(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	dialogue := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownAnswerReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownAnswerRequest{
			Handle:       "4000542609162635",
			MsgHandle:    "1",
			AnswerHandle: "3q3gs",
		}),
	}, socketSession)

	if !dialogue.handled || dialogue.answerSpeak == nil {
		t.Fatalf("expected YeMei gift dialogue, got %+v", dialogue)
	}
	if dialogue.answerSpeak.MsgHandle != "3q3d_1" || len(dialogue.answerSpeak.Answers) != 2 || dialogue.answerSpeak.Answers[0].Handle != "3q3a_1_1" {
		t.Fatalf("expected captured YeMei gift dialogue shape, got %+v", dialogue.answerSpeak)
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownAnswerReq,
		Seq: 3,
		Payload: mustJSON(t, classicTownAnswerRequest{
			Handle:       "4000542609162635",
			MsgHandle:    "3q3d_1",
			AnswerHandle: "3q3a_1_1",
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected YeMei reward answer to be handled")
	}
	if result.answerSpeak == nil || result.answerSpeak.MsgHandle != "3q3a_1_1" {
		t.Fatalf("expected YeMei reward close dialogue, got %+v", result.answerSpeak)
	}
	if len(result.itemInfos) != 4 {
		t.Fatalf("expected YeMei reward to push 4 source items, got %+v", result.itemInfos)
	}
	items := itemInfosByName(result.itemInfos)
	assertItemInfo(t, items["蓝布衣"], "背包", "equip", "291.png", 1)
	assertItemInfo(t, items["蓝布裤"], "背包", "equip", "3.png", 1)
	assertItemInfo(t, items["布鞋"], "背包", "equip", "274.png", 1)
	assertItemInfo(t, items["L花卷"], "背包", "own", "213.png", 5)
	if !strings.Contains(items["布鞋"].Description, "&18@2") {
		t.Fatalf("expected cloth shoes reward to keep captured socket marker &18@2, got %q", items["布鞋"].Description)
	}

	bag := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownGetItemListReq,
		Seq: 4,
		Payload: mustJSON(t, classicTownContainerRequest{
			Type: "背包",
		}),
	}, socketSession)
	bagItems := itemInfosByName(bag.itemInfos)
	for _, name := range []string{"铁斧", "蓝布衣", "蓝布裤", "布鞋", "L花卷"} {
		if bagItems[name].Name == "" {
			t.Fatalf("expected bag to contain %s after YeMei reward, got %+v", name, bag.itemInfos)
		}
	}
}

func TestHandlePacketClassicTownEquipSourceArmorUpdatesSlotsAppearanceAndPhysique(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	reward := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownAnswerReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownAnswerRequest{
			Handle:       "4000542609162635",
			MsgHandle:    "3q3d_1",
			AnswerHandle: "3q3a_1_1",
		}),
	}, socketSession)

	rewardItems := itemInfosByName(reward.itemInfos)
	cases := []struct {
		name            string
		slot            int
		sourceQueryPart string
	}{
		{name: "蓝布衣", slot: 4, sourceQueryPart: "c=1"},
		{name: "蓝布裤", slot: 5, sourceQueryPart: "p=1"},
		{name: "布鞋", slot: 12, sourceQueryPart: "se=1"},
	}
	for _, tc := range cases {
		bagItem := rewardItems[tc.name]
		if bagItem.Name == "" {
			t.Fatalf("expected reward item %s in %+v", tc.name, reward.itemInfos)
		}

		result := handlePacketWithSession(store, protocol.Packet{
			Cmd: cmdClassicTownEquipItemReq,
			Seq: 3,
			Payload: mustJSON(t, classicTownEquipItemRequest{
				Type:  bagItem.Type,
				Index: bagItem.Index,
				Count: 1,
			}),
		}, socketSession)

		if !result.handled {
			t.Fatalf("expected EquipItem for %s to be handled", tc.name)
		}
		if len(result.itemInfos) != 1 || result.itemInfos[0].Type != "装备" || result.itemInfos[0].Index != tc.slot || result.itemInfos[0].Name != tc.name {
			t.Fatalf("expected %s equipment push at source slot %d, got %+v", tc.name, tc.slot, result.itemInfos)
		}
		if len(result.itemClears) != 1 || result.itemClears[0].Type != bagItem.Type || result.itemClears[0].Index != bagItem.Index {
			t.Fatalf("expected %s source bag slot to clear, got %+v", tc.name, result.itemClears)
		}
		if result.createPlayer == nil || !strings.Contains(result.createPlayer.SourceQuery, tc.sourceQueryPart) {
			t.Fatalf("expected %s equip to push createPlayer sourceQuery %s, got %+v", tc.name, tc.sourceQueryPart, result.createPlayer)
		}
		if result.rolePhysique == nil {
			t.Fatalf("expected %s equip to push rolePhysique", tc.name)
		}
	}

	if socketSession.playerBase == nil || socketSession.playerBase.RolePhysique == nil || socketSession.playerBase.RolePhysique.PhyDef != 22 {
		t.Fatalf("expected equipped starter armor to raise source phyDef to 22, got %+v", socketSession.playerBase)
	}
	for _, part := range []string{"c=1", "p=1", "se=1"} {
		if socketSession.selectedRole == nil || !strings.Contains(socketSession.selectedRole.SourceQuery, part) {
			t.Fatalf("expected socket selected role source query to contain %s, got %+v", part, socketSession.selectedRole)
		}
	}
}

func TestHandlePacketClassicTownLearnSkillAnswerPushesSourceCategoryDialogue(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownAnswerReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownAnswerRequest{
			Handle:       "1000542608713897",
			MsgHandle:    "1",
			AnswerHandle: "1",
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected learn skill answer to be handled")
	}
	if result.answerSpeak == nil {
		t.Fatal("expected learn skill answer to push source category dialogue")
	}
	if result.answerSpeak.MsgHandle != "10" {
		t.Fatalf("expected source skill category msgHandle 10, got %q", result.answerSpeak.MsgHandle)
	}
	if len(result.answerSpeak.Answers) != 4 || result.answerSpeak.Answers[0].Handle != "7" || result.answerSpeak.Answers[1].Handle != "8" || result.answerSpeak.Answers[2].Handle != "9" {
		t.Fatalf("expected warrior/mage/ranger skill category answers, got %+v", result.answerSpeak.Answers)
	}
	if result.skillCap != nil || len(result.skillInfos) != 0 {
		t.Fatalf("expected category answer not to mutate learned skills, got cap=%+v skills=%+v", result.skillCap, result.skillInfos)
	}

	listResult := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetSkillListReq,
		Seq:     3,
		Payload: mustJSON(t, map[string]any{}),
	}, socketSession)
	if len(listResult.skillInfos) != 2 || listResult.skillInfos[0].Name != "密斩" || listResult.skillInfos[1].Name != "普通攻击" {
		t.Fatalf("expected only default source skills after category dialogue, got %+v", listResult.skillInfos)
	}
}

func TestHandlePacketClassicTownFastPanelPushesCapturedDefaultSlots(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	result := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetFastPanelReq,
		Seq:     2,
		Payload: mustJSON(t, map[string]any{}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected GetFastPanel to be handled")
	}
	if result.fastPanel == nil || len(result.fastPanel.Entries) != 2 {
		t.Fatalf("expected two default fast panel entries, got %+v", result.fastPanel)
	}
	if result.fastPanel.Entries[0].Index != 0 || result.fastPanel.Entries[0].Type != "skill" || result.fastPanel.Entries[0].Name != "普通攻击" {
		t.Fatalf("expected slot 0 normal attack, got %+v", result.fastPanel.Entries[0])
	}
	if result.fastPanel.Entries[1].Index != 1 || result.fastPanel.Entries[1].Type != "skill" || result.fastPanel.Entries[1].Name != "密斩" {
		t.Fatalf("expected slot 1 mizhan, got %+v", result.fastPanel.Entries[1])
	}
}

func TestHandlePacketClassicTownFastPanelDoesNotAutoAddLearnedSkills(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	_, _, found, learned := store.LearnRoleSkill(
		socketSession.playerBase.PlayerID,
		socketSession.selectedRole.RoleID,
		session.RoleSkill{Name: "武器专精", Level: 1, Type: "被动技能", Icon: "631.png"},
	)
	if !found || !learned {
		t.Fatal("expected passive test skill to be learned")
	}
	_, _, found, learned = store.LearnRoleSkill(
		socketSession.playerBase.PlayerID,
		socketSession.selectedRole.RoleID,
		session.RoleSkill{Name: "挑衅", Level: 1, Type: "技能·通用", Icon: "634.png"},
	)
	if !found || !learned {
		t.Fatal("expected active test skill to be learned")
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetFastPanelReq,
		Seq:     2,
		Payload: mustJSON(t, map[string]any{}),
	}, socketSession)

	if result.fastPanel == nil {
		t.Fatal("expected fast panel push")
	}
	for _, entry := range result.fastPanel.Entries {
		if entry.Name == "武器专精" {
			t.Fatalf("expected passive skill not to appear on fast panel, got %+v", result.fastPanel.Entries)
		}
	}
	for _, entry := range result.fastPanel.Entries {
		if entry.Name == "挑衅" {
			t.Fatalf("expected newly learned active skill not to auto-appear on fast panel, got %+v", result.fastPanel.Entries)
		}
	}
	if len(result.fastPanel.Entries) != 2 || result.fastPanel.Entries[0].Name != "普通攻击" || result.fastPanel.Entries[1].Name != "密斩" {
		t.Fatalf("expected only source default fast panel slots, got %+v", result.fastPanel.Entries)
	}
}

func TestHandlePacketClassicTownCapturedWoodcutter222PushesCapturedSkillsAndFastPanel(t *testing.T) {
	store := session.NewStore()
	socketSession, _ := seedSelectedRoleSessionInStore(t, store, "222")
	skillResult := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetSkillListReq,
		Seq:     2,
		Payload: mustJSON(t, map[string]any{}),
	}, socketSession)

	if !skillResult.handled || skillResult.skillCap == nil || skillResult.skillCap.Count != 12 || len(skillResult.skillInfos) != 12 {
		t.Fatalf("expected captured 222 skill list, got %+v", skillResult)
	}
	skillByName := map[string]classicTownSkillInfoPush{}
	for _, skill := range skillResult.skillInfos {
		skillByName[skill.Name] = skill
	}
	if _, ok := skillByName["密斩"]; ok {
		t.Fatalf("expected captured 222 not to receive default 密斩, got %+v", skillResult.skillInfos)
	}
	if _, ok := skillByName["多段刺"]; ok {
		t.Fatalf("expected captured final 222 state to replace 多段刺 with 奥义.暗杀者, got %+v", skillResult.skillInfos)
	}
	if skill := skillByName["奥义.暗杀者"]; skill.Level != 1 || skill.Type != "oneE" || skill.Icon != "262.png" || !strings.Contains(skill.Description, "提升180%的物理伤害") {
		t.Fatalf("expected captured 奥义.暗杀者 info, got %+v", skill)
	}

	fastPanelResult := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetFastPanelReq,
		Seq:     3,
		Payload: mustJSON(t, map[string]any{}),
	}, socketSession)
	if !fastPanelResult.handled || fastPanelResult.fastPanel == nil {
		t.Fatalf("expected captured 222 fast panel push, got %+v", fastPanelResult)
	}
	for _, expected := range []struct {
		index     int
		entryType string
		name      string
	}{
		{0, "skill", "普通攻击"},
		{1, "skill", "强力飞镖"},
		{2, "skill", "奥义.暗杀者"},
		{3, "skill", "投毒"},
		{4, "skill", "疾风刺"},
		{5, "skill", "解毒术"},
		{6, "skill", "魔力突刺"},
		{8, "item", "馒头"},
		{9, "item", "小瓶甘露"},
	} {
		if !fastPanelContains(fastPanelResult.fastPanel.Entries, expected.index, expected.entryType, expected.name) {
			t.Fatalf("expected captured fast panel slot %+v, got %+v", expected, fastPanelResult.fastPanel.Entries)
		}
	}
}

func TestHandlePacketClassicTownCapturedWoodcutter333PushesCapturedFinalSkillsAndFastPanel(t *testing.T) {
	store := session.NewStore()
	login := store.Login(session.LoginRequest{
		UserName: "33333333",
		Password: "33333333",
	})
	create := store.CreateRole(session.RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "333",
		Gender:         "male",
		RoleTemplateID: 1,
	})
	if !create.Success {
		t.Fatalf("expected 333 role create success, got %+v", create)
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
	if !selectResult.handled || selectResult.townBootstrap == nil {
		t.Fatalf("expected 333 role select to seed town bootstrap, got %+v", selectResult)
	}

	skillResult := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetSkillListReq,
		Seq:     2,
		Payload: mustJSON(t, map[string]any{}),
	}, socketSession)
	if !skillResult.handled || skillResult.skillCap == nil || skillResult.skillCap.Count != 12 || len(skillResult.skillInfos) != 12 {
		t.Fatalf("expected captured final 333 skill list, got %+v", skillResult)
	}
	expectedSkillOrder := []string{"普通攻击", "武器娴熟", "灵力进修", "精神力", "爆发力", "幻影", "贯甲连矢", "暗影箭", "奥义.轰雷矢", "毒矢", "魔力速射", "冰箭速射"}
	for index, expectedName := range expectedSkillOrder {
		if skillResult.skillInfos[index].Name != expectedName {
			t.Fatalf("expected captured 333 skill order %v, got %+v", expectedSkillOrder, skillResult.skillInfos)
		}
	}
	skillByName := map[string]classicTownSkillInfoPush{}
	for _, skill := range skillResult.skillInfos {
		skillByName[skill.Name] = skill
	}
	if skillByName["贯甲连矢"].Level != 5 || !strings.Contains(skillByName["贯甲连矢"].Description, "&2@28") {
		t.Fatalf("expected captured final 333 贯甲连矢 Lv5, got %+v", skillByName["贯甲连矢"])
	}
	if skillByName["暗影箭"].Level != 1 || !strings.Contains(skillByName["暗影箭"].Description, "暗影箭x1") {
		t.Fatalf("expected captured final 333 暗影箭 Lv1, got %+v", skillByName["暗影箭"])
	}
	if skillByName["毒矢"].Level != 1 || !strings.Contains(skillByName["毒矢"].Description, "毒箭x1") {
		t.Fatalf("expected captured final 333 毒矢 Lv1, got %+v", skillByName["毒矢"])
	}
	if skillByName["奥义.轰雷矢"].Level != 1 || !strings.Contains(skillByName["奥义.轰雷矢"].Description, "需要2格魂元") {
		t.Fatalf("expected captured final 333 奥义.轰雷矢 Lv1, got %+v", skillByName["奥义.轰雷矢"])
	}
	if skillByName["魔力速射"].Level != 5 || !strings.Contains(skillByName["魔力速射"].Description, "魔箭x1") {
		t.Fatalf("expected captured final 333 魔力速射 Lv5, got %+v", skillByName["魔力速射"])
	}
	if skillByName["冰箭速射"].Level != 5 || !strings.Contains(skillByName["冰箭速射"].Description, "冰之箭x1") {
		t.Fatalf("expected captured final 333 冰箭速射 Lv5, got %+v", skillByName["冰箭速射"])
	}
	if _, ok := skillByName["奥义.暗杀者"]; ok {
		t.Fatalf("expected captured final 333 skillInfo not to reuse stale 222 dagger list, got %+v", skillResult.skillInfos)
	}

	fastPanelResult := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetFastPanelReq,
		Seq:     3,
		Payload: mustJSON(t, map[string]any{}),
	}, socketSession)
	if !fastPanelResult.handled || fastPanelResult.fastPanel == nil {
		t.Fatalf("expected captured final 333 fast panel push, got %+v", fastPanelResult)
	}
	for _, expected := range []struct {
		index     int
		entryType string
		name      string
	}{
		{0, "skill", "普通攻击"},
		{1, "skill", "贯甲连矢"},
		{2, "skill", "魔力速射"},
		{3, "skill", "暗影箭"},
		{4, "skill", "毒矢"},
		{5, "skill", "奥义.轰雷矢"},
		{6, "skill", "冰箭速射"},
		{8, "item", "馒头"},
		{9, "item", "小瓶甘露"},
	} {
		if !fastPanelContains(fastPanelResult.fastPanel.Entries, expected.index, expected.entryType, expected.name) {
			t.Fatalf("expected captured final 333 fast panel slot %+v, got %+v", expected, fastPanelResult.fastPanel.Entries)
		}
	}
}

func TestHandlePacketClassicTownCapturedWoodcutter222PushesCapturedEquipment(t *testing.T) {
	store := session.NewStore()
	socketSession, _ := seedSelectedRoleSessionInStore(t, store, "222")

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownGetItemListReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownContainerRequest{
			Type: "装备",
		}),
	}, socketSession)

	if !result.handled || result.containerCap == nil || result.containerCap.Type != "装备" || result.containerCap.Capacity != 20 {
		t.Fatalf("expected captured equipment list response, got %+v", result)
	}
	if len(result.itemInfos) != 10 {
		t.Fatalf("expected captured woodcutter equipment rows, got %+v", result.itemInfos)
	}
	itemsByName := itemInfosByName(result.itemInfos)
	expectedEquipment := map[string]struct {
		index   int
		display string
	}{
		"黄风围巾":   {index: 0, display: "548.png"},
		"蚩颅王护肩":  {index: 1, display: "484.png"},
		"黄风护腕":   {index: 2, display: "549.png"},
		"绯雨匕首":   {index: 3, display: "51.png"},
		"神风护甲":   {index: 4, display: "366.png"},
		"神风护腿":   {index: 5, display: "368.png"},
		"炎火兽":    {index: 9, display: "324.png"},
		"神风护腰":   {index: 10, display: "369.png"},
		"神风战靴":   {index: 12, display: "370.png"},
		"L千年人参果": {index: 15, display: "921.png"},
	}
	for name, expected := range expectedEquipment {
		item := itemsByName[name]
		if item.Name == "" || item.Index != expected.index || item.Display != expected.display || item.Type != "装备" || item.ItemType != "equip" {
			t.Fatalf("expected captured equipment %s at index %d display %s, got %+v", name, expected.index, expected.display, item)
		}
	}
	if socketSession.selectedRole == nil || !strings.Contains(socketSession.selectedRole.SourceQuery, "w3=49") || strings.Contains(socketSession.selectedRole.SourceQuery, "w3=43") {
		t.Fatalf("expected selected role to use captured final appearance, got %+v", socketSession.selectedRole)
	}
	if socketSession.playerBase == nil || !strings.Contains(socketSession.playerBase.BattleSourceQuery, "w3=49") {
		t.Fatalf("expected player base battle source query to use captured final appearance, got %+v", socketSession.playerBase)
	}
}

func TestHandlePacketClassicTownOtherEquipmentUsesCapturedLookDetailFixture(t *testing.T) {
	store := session.NewStore()
	socketSession, _ := seedSelectedRoleSessionInStore(t, store, "look-equipment")

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownOtherEquipmentReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownOtherEquipmentRequest{
			RoleName: "恐龙抗狼1",
			RoleID:   "player_21424",
			Handle:   "player_21424",
		}),
	}, socketSession)

	if !result.handled || result.otherEquipment == nil {
		t.Fatalf("expected captured other equipment response, got %+v", result)
	}
	equipment := result.otherEquipment
	if equipment.Handle != "player_21424" || equipment.RoleID != "player_21424" || equipment.RoleName != "恐龙抗狼1" {
		t.Fatalf("expected captured target identity, got %+v", equipment)
	}
	if equipment.Type != "装备" || equipment.Capacity != 20 || !equipment.Partial {
		t.Fatalf("expected partial captured equipment container, got %+v", equipment)
	}
	if !strings.Contains(equipment.SourceCapture, "GetLookDetail(player_21424)") {
		t.Fatalf("expected source capture pointer to GetLookDetail, got %q", equipment.SourceCapture)
	}
	if len(equipment.Items) != 10 {
		t.Fatalf("expected captured other equipment rows, got %+v", equipment.Items)
	}

	itemsByName := itemInfosByName(equipment.Items)
	expectedEquipment := map[string]struct {
		index   int
		display string
	}{
		"无双头盔":  {index: 0, display: "357.png"},
		"无双护肩":  {index: 1, display: "358.png"},
		"无双铁腕":  {index: 2, display: "360.png"},
		"饮血刀":   {index: 3, display: "48.png"},
		"寨夫人上衣": {index: 4, display: "474.png"},
		"无双护腿":  {index: 5, display: "361.png"},
		"泥戒指":   {index: 6, display: "431.png"},
		"怪木机":   {index: 9, display: "333.png"},
		"无双铁腰带": {index: 10, display: "362.png"},
		"蛤蟆精战靴": {index: 12, display: "503.png"},
	}
	for name, expected := range expectedEquipment {
		item := itemsByName[name]
		if item.Name == "" || item.Index != expected.index || item.Display != expected.display || item.Type != "装备" || item.ItemType != "equip" {
			t.Fatalf("expected captured other equipment %s at index %d display %s, got %+v", name, expected.index, expected.display, item)
		}
	}

	missingResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownOtherEquipmentReq,
		Seq: 3,
		Payload: mustJSON(t, classicTownOtherEquipmentRequest{
			RoleName: "未恢复目标",
			RoleID:   "player_unrestored",
			Handle:   "player_unrestored",
		}),
	}, socketSession)
	if !missingResult.handled || missingResult.otherEquipment == nil {
		t.Fatalf("expected explicit unrestored-target response, got %+v", missingResult)
	}
	if missingResult.otherEquipment.ErrorCode != "look_equipment_unrestored" || len(missingResult.otherEquipment.Items) != 0 {
		t.Fatalf("expected unrestored-target error without guessed items, got %+v", missingResult.otherEquipment)
	}
}

func TestHandlePacketClassicTownSetFastPanelStoresActiveSkill(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	_, _, found, learned := store.LearnRoleSkill(
		socketSession.playerBase.PlayerID,
		socketSession.selectedRole.RoleID,
		session.RoleSkill{Name: "挑衅", Level: 1, Type: "技能·通用", Icon: "634.png"},
	)
	if !found || !learned {
		t.Fatal("expected active test skill to be learned")
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownSetFastPanelReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownSetFastPanelRequest{
			Index: 4,
			Type:  "skill",
			Name:  "挑衅",
		}),
	}, socketSession)

	if !result.handled || result.fastPanel == nil {
		t.Fatalf("expected SetFastPanel to push fast panel, got %+v", result)
	}
	if !fastPanelContains(result.fastPanel.Entries, 4, "skill", "挑衅") {
		t.Fatalf("expected slot 4 taunt after SetFastPanel, got %+v", result.fastPanel.Entries)
	}

	getResult := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetFastPanelReq,
		Seq:     3,
		Payload: mustJSON(t, map[string]any{}),
	}, socketSession)
	if getResult.fastPanel == nil || !fastPanelContains(getResult.fastPanel.Entries, 4, "skill", "挑衅") {
		t.Fatalf("expected persisted slot 4 taunt after GetFastPanel, got %+v", getResult.fastPanel)
	}
}

func TestHandlePacketClassicTownRemoveFastPanelClearsSlot(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	_, _, found, learned := store.LearnRoleSkill(
		socketSession.playerBase.PlayerID,
		socketSession.selectedRole.RoleID,
		session.RoleSkill{Name: "鎸戣", Level: 1, Type: "鎶€鑳铰烽€氱敤", Icon: "634.png"},
	)
	if !found || !learned {
		t.Fatal("expected active test skill to be learned")
	}

	setResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownSetFastPanelReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownSetFastPanelRequest{
			Index: 4,
			Type:  "skill",
			Name:  "鎸戣",
		}),
	}, socketSession)
	if !setResult.handled || setResult.fastPanel == nil || !fastPanelContains(setResult.fastPanel.Entries, 4, "skill", "鎸戣") {
		t.Fatalf("expected slot 4 taunt after SetFastPanel, got %+v", setResult.fastPanel)
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownRemoveFastPanelReq,
		Seq: 3,
		Payload: mustJSON(t, classicTownRemoveFastPanelRequest{
			Index: 4,
		}),
	}, socketSession)
	if !result.handled || result.fastPanel == nil {
		t.Fatalf("expected RemoveFastPanel to push fast panel, got %+v", result)
	}
	if fastPanelContains(result.fastPanel.Entries, 4, "skill", "鎸戣") {
		t.Fatalf("expected slot 4 taunt to be removed, got %+v", result.fastPanel.Entries)
	}

	getResult := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetFastPanelReq,
		Seq:     4,
		Payload: mustJSON(t, map[string]any{}),
	}, socketSession)
	if getResult.fastPanel == nil || fastPanelContains(getResult.fastPanel.Entries, 4, "skill", "鎸戣") {
		t.Fatalf("expected persisted slot 4 removal after GetFastPanel, got %+v", getResult.fastPanel)
	}
}

func TestHandlePacketClassicTownSetFastPanelStoresBagItem(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	if _, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, session.RoleItem{
		Type:        "背包",
		Name:        "包子",
		ItemType:    "own",
		Display:     "212.png",
		Description: "f_i_包子&24@消耗品&25@99&7@600&20@带馅的包子&0;看起来非常可口&0;食用后可恢复些气力.",
		Count:       3,
		Index:       -1,
		Level:       1,
		ItemLevel:   1,
	}); !ok {
		t.Fatal("expected test bun to be granted")
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownSetFastPanelReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownSetFastPanelRequest{
			Index: 5,
			Type:  "item",
			Name:  "包子",
		}),
	}, socketSession)

	if !result.handled || result.fastPanel == nil {
		t.Fatalf("expected item SetFastPanel to push fast panel, got %+v", result)
	}
	if !fastPanelContains(result.fastPanel.Entries, 5, "item", "包子") {
		t.Fatalf("expected slot 5 bun after SetFastPanel, got %+v", result.fastPanel.Entries)
	}

	getResult := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetFastPanelReq,
		Seq:     3,
		Payload: mustJSON(t, map[string]any{}),
	}, socketSession)
	if getResult.fastPanel == nil || !fastPanelContains(getResult.fastPanel.Entries, 5, "item", "包子") {
		t.Fatalf("expected persisted slot 5 bun after GetFastPanel, got %+v", getResult.fastPanel)
	}
}

func TestHandlePacketClassicTownSetFastPanelRejectsMissingBagItem(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownSetFastPanelReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownSetFastPanelRequest{
			Index: 5,
			Type:  "item",
			Name:  "不存在的蓝药",
		}),
	}, socketSession)

	if !result.handled || result.fastPanel == nil {
		t.Fatalf("expected missing item SetFastPanel to be handled with unchanged fast panel, got %+v", result)
	}
	if fastPanelContains(result.fastPanel.Entries, 5, "item", "不存在的蓝药") {
		t.Fatalf("expected missing bag item not to enter fast panel, got %+v", result.fastPanel.Entries)
	}
}

func TestHandlePacketClassicTownSetFastPanelRejectsPassiveSkill(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	_, _, found, learned := store.LearnRoleSkill(
		socketSession.playerBase.PlayerID,
		socketSession.selectedRole.RoleID,
		session.RoleSkill{Name: "武器专精", Level: 1, Type: "被动技能", Icon: "631.png"},
	)
	if !found || !learned {
		t.Fatal("expected passive test skill to be learned")
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownSetFastPanelReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownSetFastPanelRequest{
			Index: 4,
			Type:  "skill",
			Name:  "武器专精",
		}),
	}, socketSession)

	if !result.handled || result.fastPanel == nil {
		t.Fatalf("expected passive SetFastPanel to be handled with unchanged fast panel, got %+v", result)
	}
	if fastPanelContains(result.fastPanel.Entries, 4, "skill", "武器专精") {
		t.Fatalf("expected passive skill not to enter fast panel, got %+v", result.fastPanel.Entries)
	}
}

func TestHandlePacketClassicTownBuySkillDeductsCurrencyAndPushesSkillItem(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownBuySkillReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownBuySkillRequest{
			ShopID:  "skill1",
			SkillID: 0,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected BuySkill to be handled")
	}
	if result.buySkillResult == nil || !result.buySkillResult.Success {
		t.Fatalf("expected successful buySkill result, got %+v", result.buySkillResult)
	}
	if result.currencyPush == nil || result.currencyPush.Currencies["铜钱"] != 4500 {
		t.Fatalf("expected copper deduction push to 4500, got %+v", result.currencyPush)
	}
	if len(result.skillInfos) != 0 {
		t.Fatalf("expected buy skill to avoid direct skill push, got %+v", result.skillInfos)
	}
	if len(result.itemInfos) != 1 || result.itemInfos[0].Name != "武器专精" || result.itemInfos[0].Display != "631.png" {
		t.Fatalf("expected purchased skill item push, got %+v", result.itemInfos)
	}
	if len(result.chatMessages) != 1 || !strings.Contains(result.chatMessages[0].Msg, "购买了【武器专精】x1") {
		t.Fatalf("expected buy skill source system message, got %+v", result.chatMessages)
	}

	listResult := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetSkillListReq,
		Seq:     3,
		Payload: mustJSON(t, map[string]any{}),
	}, socketSession)
	if len(listResult.skillInfos) != 2 {
		t.Fatalf("expected purchased skill item not to persist as learned skill, got %+v", listResult.skillInfos)
	}
	if listResult.currencyPush == nil || listResult.currencyPush.Currencies["铜钱"] != 4500 {
		t.Fatalf("expected purchased currency state to persist, got %+v", listResult.currencyPush)
	}

	learnResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownActiveItemReq,
		Seq: 4,
		Payload: mustJSON(t, classicTownActiveItemRequest{
			Type:  result.itemInfos[0].Type,
			Index: result.itemInfos[0].Index,
		}),
	}, socketSession)
	if len(learnResult.skillInfos) != 1 || learnResult.skillInfos[0].Name != "武器专精" || learnResult.skillInfos[0].Icon != "631.png" {
		t.Fatalf("expected active skill item to push learned skill, got %+v", learnResult.skillInfos)
	}
	if len(learnResult.itemClears) != 1 || learnResult.itemClears[0].Index != result.itemInfos[0].Index {
		t.Fatalf("expected active skill item to clear consumed item, got %+v", learnResult.itemClears)
	}
	if len(learnResult.chatMessages) != 1 || learnResult.chatMessages[0].Channel != "system" || !strings.Contains(learnResult.chatMessages[0].Msg, "习得【武器专精】Lv.1") {
		t.Fatalf("expected active skill item to push source sayS learn message, got %+v", learnResult.chatMessages)
	}
}

func TestHandlePacketClassicTownActiveItemLearnsCapturedWeaponFamiliarityMessage(t *testing.T) {
	store := session.NewStore()
	socketSession, role := seedSelectedRoleSessionInStore(t, store, "weapon-familiarity-message")
	skillBook, ok := session.CapturedRoleItemTemplate("\u6b66\u5668\u5a34\u719f")
	if !ok {
		t.Fatal("expected captured weapon familiarity skill book template")
	}
	skillBook.Type = classicTownBagContainerType
	skillBook.Index = -1
	skillBook.Count = 1
	granted, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, role.RoleID, skillBook)
	if !ok {
		t.Fatal("expected skill book grant")
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownActiveItemReq,
		Seq: 18,
		Payload: mustJSON(t, classicTownActiveItemRequest{
			Type:  granted.Type,
			Index: granted.Index,
		}),
	}, socketSession)

	if !result.handled || len(result.skillInfos) != 1 || len(result.itemClears) != 1 {
		t.Fatalf("expected captured weapon familiarity learn result, got %+v", result)
	}
	skillInfo := result.skillInfos[0]
	if skillInfo.Name != "\u6b66\u5668\u5a34\u719f" || skillInfo.Level != 1 || skillInfo.Type != "null" || skillInfo.Icon != "226.png" {
		t.Fatalf("expected captured skillInfo presentation, got %+v", skillInfo)
	}
	if !strings.Contains(skillInfo.Description, "f_s_\u6b66\u5668\u5a34\u719f") || !strings.Contains(skillInfo.Description, "&12@8") {
		t.Fatalf("expected captured skillInfo description, got %+v", skillInfo)
	}
	if len(result.chatMessages) != 1 || result.chatMessages[0].Msg != "\u4e60\u5f97Lv1[\u6b66\u5668\u5a34\u719f]" {
		t.Fatalf("expected captured learned-skill c_Speak message, got %+v", result.chatMessages)
	}
}

func TestSourceSkillItemsAndSkillInfoUseCapturedLevelDescriptions(t *testing.T) {
	entry, ok := findSourceSkillShopEntry("skill1", 10)
	if !ok {
		t.Fatal("expected captured 多段斩 shop entry")
	}
	item := sourceSkillEntryToRoleItem(entry)
	if item.Description != "f_s_多段斩^ffffff&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@8&4@提升55%的物理伤害" {
		t.Fatalf("expected 多段斩 skill item to use captured Lv1 description, got %+v", item)
	}

	skillInfo := classicTownSkillInfoPushFromRoleSkill("role_1", session.RoleSkill{
		Name:        "嗜血斩",
		Level:       3,
		Type:        "oneE",
		Icon:        "645.png",
		Description: "f_s_嗜血斩^5BC46D&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@24&4@对敌人造成92%的物理伤害&0;并有82%机率将对敌人造成伤害的70%转换为气力</font>",
	})
	if !strings.Contains(skillInfo.Description, "&2@28") || !strings.Contains(skillInfo.Description, "造成96%的物理伤害") || !strings.Contains(skillInfo.Description, "86%机率") {
		t.Fatalf("expected skillInfo to use captured 嗜血斩 Lv3 description, got %+v", skillInfo)
	}

	redMoonInfo := classicTownSkillInfoPushFromRoleSkill("role_1", session.RoleSkill{
		Name:        "红月斩",
		Level:       1,
		Type:        "all",
		Icon:        "647.png",
		Description: "对所有敌人造成一定的物理伤害",
	})
	if !strings.Contains(redMoonInfo.Description, "&2@40") || !strings.Contains(redMoonInfo.Description, "对所有敌人造成72%的物理伤害") {
		t.Fatalf("expected skillInfo to use captured 红月斩 Lv1 description, got %+v", redMoonInfo)
	}

	xueQieInfo := classicTownSkillInfoPushFromRoleSkill("role_1", session.RoleSkill{
		Name:        "血切",
		Level:       1,
		Type:        "oneE",
		Icon:        "648.png",
		Description: "对敌人造成一定的物理伤害",
	})
	if !strings.Contains(xueQieInfo.Description, "&2@19") || !strings.Contains(xueQieInfo.Description, "造成30%的物理伤害") || !strings.Contains(xueQieInfo.Description, "80%的机率") {
		t.Fatalf("expected skillInfo to use captured 血切 Lv1 description, got %+v", xueQieInfo)
	}

	duoDuanCiInfo := classicTownSkillInfoPushFromRoleSkill("role_1", session.RoleSkill{
		Name:        "多段刺",
		Level:       5,
		Type:        "oneE",
		Icon:        "257.png",
		Description: "提高对敌人造成的物理伤害",
	})
	if !strings.Contains(duoDuanCiInfo.Description, "&2@18") || !strings.Contains(duoDuanCiInfo.Description, "提升45%的物理伤害") {
		t.Fatalf("expected skillInfo to use captured 多段刺 Lv5 description, got %+v", duoDuanCiInfo)
	}

	qiangLiFeiBiaoInfo := classicTownSkillInfoPushFromRoleSkill("role_1", session.RoleSkill{
		Name:        "强力飞镖",
		Level:       2,
		Type:        "oneE",
		Icon:        "261.png",
		Description: "对敌人造成物理伤害 / 进攻时候提升一定的物理攻击力",
	})
	if !strings.Contains(qiangLiFeiBiaoInfo.Description, "&2@20") || !strings.Contains(qiangLiFeiBiaoInfo.Description, "需要【飞镖x1】") || !strings.Contains(qiangLiFeiBiaoInfo.Description, "提高48%") || !strings.Contains(qiangLiFeiBiaoInfo.Description, "无视防御") {
		t.Fatalf("expected skillInfo to use captured 强力飞镖 Lv2 description, got %+v", qiangLiFeiBiaoInfo)
	}

	qiangLiFeiBiaoLv3Info := classicTownSkillInfoPushFromRoleSkill("role_1", session.RoleSkill{
		Name:        "强力飞镖",
		Level:       3,
		Type:        "oneE",
		Icon:        "261.png",
		Description: "对敌人造成物理伤害 / 进攻时候提升一定的物理攻击力",
	})
	if !strings.Contains(qiangLiFeiBiaoLv3Info.Description, "&2@24") || !strings.Contains(qiangLiFeiBiaoLv3Info.Description, "提高50%") || !strings.Contains(qiangLiFeiBiaoLv3Info.Description, "无视防御") {
		t.Fatalf("expected skillInfo to use captured 强力飞镖 Lv3 description, got %+v", qiangLiFeiBiaoLv3Info)
	}

	moLiTuCiInfo := classicTownSkillInfoPushFromRoleSkill("role_1", session.RoleSkill{
		Name:        "魔力突刺",
		Level:       1,
		Type:        "oneE",
		Icon:        "258.png",
		Description: "提升对敌人造成的物理伤害 / 并附加魔法伤害",
	})
	if !strings.Contains(moLiTuCiInfo.Description, "&2@20") || !strings.Contains(moLiTuCiInfo.Description, "造成敌人100%的物理伤害") || !strings.Contains(moLiTuCiInfo.Description, "追加80%的魔法伤害") {
		t.Fatalf("expected skillInfo to use captured 魔力突刺 Lv1 description, got %+v", moLiTuCiInfo)
	}

	jiFengCiInfo := classicTownSkillInfoPushFromRoleSkill("role_1", session.RoleSkill{
		Name:        "疾风刺",
		Level:       1,
		Type:        "oneE",
		Icon:        "259.png",
		Description: "对敌人造成物理伤害 / 击中敌人时有机率使其进入迟钝状态",
	})
	if !strings.Contains(jiFengCiInfo.Description, "&2@20") || !strings.Contains(jiFengCiInfo.Description, "造成40%的物理伤害") || !strings.Contains(jiFengCiInfo.Description, "92%的机率") {
		t.Fatalf("expected skillInfo to use captured 疾风刺 Lv1 description, got %+v", jiFengCiInfo)
	}

	touDuInfo := classicTownSkillInfoPushFromRoleSkill("role_1", session.RoleSkill{
		Name:        "投毒",
		Level:       1,
		Type:        "oneE",
		Icon:        "166.png",
		Description: "有机率使敌人中毒",
	})
	if !strings.Contains(touDuInfo.Description, "&2@16") || !strings.Contains(touDuInfo.Description, "需要【毒药x1】") || !strings.Contains(touDuInfo.Description, "有80%的机率") {
		t.Fatalf("expected skillInfo to use captured 投毒 Lv1 description, got %+v", touDuInfo)
	}

	jieDuShuInfo := classicTownSkillInfoPushFromRoleSkill("role_1", session.RoleSkill{
		Name:        "解毒术",
		Level:       1,
		Type:        "own",
		Icon:        "260.png",
		Description: "解除自身的中毒状态",
	})
	if !strings.Contains(jieDuShuInfo.Description, "&2@20") || !strings.Contains(jieDuShuInfo.Description, "解除自身中毒状态") {
		t.Fatalf("expected skillInfo to use captured 解毒术 Lv1 description, got %+v", jieDuShuInfo)
	}

	leiHunEntry, ok := findSourceSkillShopEntry("skill1", 15)
	if !ok {
		t.Fatal("expected captured 奥义.雷魂斩 shop entry")
	}
	if !strings.Contains(leiHunEntry.Description, "需要3格魂元") || !strings.Contains(leiHunEntry.Description, "提升240%的物理伤害") {
		t.Fatalf("expected 奥义.雷魂斩 shop entry to use captured description, got %+v", leiHunEntry)
	}

	leiHunInfo := classicTownSkillInfoPushFromRoleSkill("role_1", session.RoleSkill{
		Name:        "奥义.雷魂斩",
		Level:       1,
		Type:        "oneE",
		Icon:        "649.png",
		Description: "特殊发动条件:3格魂元 / 大幅提升对敌人造成的物理伤害",
	})
	if !strings.Contains(leiHunInfo.Description, "&2@24") || !strings.Contains(leiHunInfo.Description, "特殊发动条件:需要3格魂元") || !strings.Contains(leiHunInfo.Description, "提升240%的物理伤害") {
		t.Fatalf("expected skillInfo to use captured 奥义.雷魂斩 Lv1 description, got %+v", leiHunInfo)
	}

	anShaEntry, ok := findSourceSkillShopEntry("skill3", 11)
	if !ok {
		t.Fatal("expected captured 奥义.暗杀者 shop entry")
	}
	if !strings.Contains(anShaEntry.Description, "3格魂元") || !strings.Contains(anShaEntry.Description, "大幅提升对敌人造成的物理伤害") {
		t.Fatalf("expected 奥义.暗杀者 shop entry to use captured shop summary, got %+v", anShaEntry)
	}

	anShaInfo := classicTownSkillInfoPushFromRoleSkill("role_1", session.RoleSkill{
		Name:        "奥义.暗杀者",
		Level:       1,
		Type:        "oneE",
		Icon:        "687.png",
		Description: "特殊发动条件:3格魂元 / 大幅提升对敌人造成的物理伤害",
	})
	if !strings.Contains(anShaInfo.Description, "&2@26") || !strings.Contains(anShaInfo.Description, "特殊发动条件:需要3格魂元") || !strings.Contains(anShaInfo.Description, "提升180%的物理伤害") {
		t.Fatalf("expected skillInfo to use captured 奥义.暗杀者 Lv1 description, got %+v", anShaInfo)
	}
}

func TestHandlePacketClassicTownSkillItemUpgradesDuplicateAfterUse(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	request := classicTownBuySkillRequest{
		ShopID:  "skill1",
		SkillID: 0,
	}
	first := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownBuySkillReq,
		Seq:     2,
		Payload: mustJSON(t, request),
	}, socketSession)
	second := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownBuySkillReq,
		Seq:     3,
		Payload: mustJSON(t, request),
	}, socketSession)

	if first.buySkillResult == nil || !first.buySkillResult.Success {
		t.Fatalf("expected first buy success, got %+v", first.buySkillResult)
	}
	if second.buySkillResult == nil || !second.buySkillResult.Success {
		t.Fatalf("expected duplicate buy to create another skill item, got %+v", second.buySkillResult)
	}
	if second.currencyPush == nil || second.currencyPush.Currencies["铜钱"] != 4000 {
		t.Fatalf("expected duplicate buy to deduct again, got %+v", second.currencyPush)
	}
	if len(first.itemInfos) != 1 || len(second.itemInfos) != 1 {
		t.Fatalf("expected both purchases to push skill items, first=%+v second=%+v", first.itemInfos, second.itemInfos)
	}
	firstUse := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownActiveItemReq,
		Seq: 4,
		Payload: mustJSON(t, classicTownActiveItemRequest{
			Type:  first.itemInfos[0].Type,
			Index: first.itemInfos[0].Index,
		}),
	}, socketSession)
	secondUse := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownActiveItemReq,
		Seq: 5,
		Payload: mustJSON(t, classicTownActiveItemRequest{
			Type:  second.itemInfos[0].Type,
			Index: second.itemInfos[0].Index,
		}),
	}, socketSession)
	if len(firstUse.skillInfos) != 1 || firstUse.skillInfos[0].Name != "武器专精" || firstUse.skillInfos[0].Level != 1 {
		t.Fatalf("expected first skill item to learn level 1, got %+v", firstUse.skillInfos)
	}
	if len(secondUse.skillInfos) != 1 || secondUse.skillInfos[0].Name != "武器专精" || secondUse.skillInfos[0].Level != 2 {
		t.Fatalf("expected second skill item to upgrade level 2, got %+v", secondUse.skillInfos)
	}
}

func TestHandlePacketClassicTownBuySkillRejectsMissingSkill(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownBuySkillReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownBuySkillRequest{
			ShopID:  "skill1",
			SkillID: 999,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected missing BuySkill to be handled")
	}
	if result.buySkillResult == nil || result.buySkillResult.Success || result.buySkillResult.ErrorCode != "skill_missing" {
		t.Fatalf("expected missing skill rejection, got %+v", result.buySkillResult)
	}
	if result.currencyPush != nil || len(result.skillInfos) != 0 {
		t.Fatalf("expected missing skill not to mutate pushes, got currency=%+v skills=%+v", result.currencyPush, result.skillInfos)
	}
}

func TestHandlePacketClassicTownBuyItemDeductsCurrencyAndPushesBagItem(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	copper, ok := session.CapturedRoleItemTemplate("铜钱")
	if !ok {
		t.Fatal("expected copper template")
	}
	copper.Type = "背包"
	copper.Index = -1
	copper.Count = 1000
	grantedCopper, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, copper)
	if !ok {
		t.Fatal("expected copper grant")
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownBuySkillReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownBuySkillRequest{
			ShopID:  "item:1820542611400955",
			SkillID: 0,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected BuyItem to be handled")
	}
	if result.buySkillResult == nil || !result.buySkillResult.Success {
		t.Fatalf("expected successful buy item result, got %+v", result.buySkillResult)
	}
	if result.currencyPush == nil || result.currencyPush.Currencies["铜钱"] != 4992 {
		t.Fatalf("expected copper deduction push to 4992, got %+v", result.currencyPush)
	}
	itemsByName := itemInfosByName(result.itemInfos)
	if itemsByName["普通采集手套"].Count != 1 || itemsByName["普通采集手套"].Display != "856.png" {
		t.Fatalf("expected purchased 普通采集手套 item push, got %+v", result.itemInfos)
	}
	if itemsByName["铜钱"].Index != grantedCopper.Index || itemsByName["铜钱"].Count != 992 {
		t.Fatalf("expected copper stack push to 992 at slot %d, got %+v", grantedCopper.Index, result.itemInfos)
	}
	if socketSession.playerBase == nil || socketSession.playerBase.Currencies["铜钱"] != 4992 {
		t.Fatalf("expected socket currency state to update, got %+v", socketSession.playerBase)
	}
	if len(result.chatMessages) != 1 || !strings.Contains(result.chatMessages[0].Msg, "购买了【普通采集手套】x1") {
		t.Fatalf("expected buy item source system message, got %+v", result.chatMessages)
	}

	listResult := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetItemListReq,
		Seq:     3,
		Payload: mustJSON(t, classicTownContainerRequest{Type: "背包"}),
	}, socketSession)
	items := itemInfosByName(listResult.itemInfos)
	if items["普通采集手套"].Count != 1 {
		t.Fatalf("expected purchased item to persist in bag, got %+v", listResult.itemInfos)
	}
	if items["铜钱"].Index != grantedCopper.Index || items["铜钱"].Count != 992 {
		t.Fatalf("expected persisted copper stack to 992 at slot %d, got %+v", grantedCopper.Index, listResult.itemInfos)
	}
}

func TestHandlePacketClassicTownBuyItemSellsLocalDungeonTickets(t *testing.T) {
	cases := []struct {
		name       string
		shopID     string
		ticketName string
		icon       string
	}{
		{name: "yunyin feixiandong ticket", shopID: "item:7000542609490978", ticketName: "飞仙洞通行证", icon: "782.png"},
		{name: "dafo huangfengzhai ticket", shopID: "item:4090542614314425", ticketName: "黄风寨通行证", icon: "783.png"},
		{name: "jianting shuiliandong ticket", shopID: "item:4960542616750900", ticketName: "水帘洞通行证", icon: "781.png"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			store, socketSession := seedSelectedRoleSession(t)
			copper, ok := session.CapturedRoleItemTemplate("铜钱")
			if !ok {
				t.Fatal("expected copper template")
			}
			copper.Type = "背包"
			copper.Index = -1
			copper.Count = 1000
			if _, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, copper); !ok {
				t.Fatal("expected copper grant")
			}

			result := handlePacketWithSession(store, protocol.Packet{
				Cmd: cmdClassicTownBuySkillReq,
				Seq: 2,
				Payload: mustJSON(t, classicTownBuySkillRequest{
					ShopID:  tt.shopID,
					SkillID: 11,
				}),
			}, socketSession)

			if !result.handled {
				t.Fatal("expected BuyItem to be handled")
			}
			if result.buySkillResult == nil || !result.buySkillResult.Success {
				t.Fatalf("expected successful ticket purchase, got %+v", result.buySkillResult)
			}
			itemsByName := itemInfosByName(result.itemInfos)
			ticket := itemsByName[tt.ticketName]
			if ticket.Name != tt.ticketName || ticket.Display != tt.icon || ticket.Count != 1 {
				t.Fatalf("expected purchased ticket %s/%s, got %+v", tt.ticketName, tt.icon, result.itemInfos)
			}
			if result.currencyPush == nil || result.currencyPush.Currencies["铜钱"] != 4850 {
				t.Fatalf("expected ticket purchase to deduct 150 copper, got %+v", result.currencyPush)
			}
			if !packetChatMessagesContain(result.chatMessages, "购买了【"+tt.ticketName+"】x1") {
				t.Fatalf("expected ticket purchase system message, got %+v", result.chatMessages)
			}
		})
	}
}

func TestHandlePacketClassicTownBuyItemRejectsMissingMaterialWithoutMutation(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownBuySkillReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownBuySkillRequest{
			ShopID:  "item:1830542611405809",
			SkillID: 1,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected BuyItem to be handled")
	}
	if result.buySkillResult == nil || result.buySkillResult.Success || result.buySkillResult.ErrorCode != "item_not_enough" {
		t.Fatalf("expected missing material rejection, got %+v", result.buySkillResult)
	}
	if len(result.itemInfos) != 0 || len(result.itemClears) != 0 {
		t.Fatalf("expected missing material not to mutate bag pushes, got infos=%+v clears=%+v", result.itemInfos, result.itemClears)
	}
	if result.currencyPush == nil || result.currencyPush.Currencies["铜钱"] != 5000 {
		t.Fatalf("expected currency state unchanged, got %+v", result.currencyPush)
	}

	listResult := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetItemListReq,
		Seq:     3,
		Payload: mustJSON(t, classicTownContainerRequest{Type: "背包"}),
	}, socketSession)
	if _, ok := itemInfosByName(listResult.itemInfos)["精炼宝石"]; ok {
		t.Fatalf("expected rejected craft item not to appear in bag, got %+v", listResult.itemInfos)
	}
}

func TestHandlePacketClassicTownSkillCategoryPushesSkillShop(t *testing.T) {
	cases := []struct {
		answerHandle string
		shopID       string
		vocation     string
		skillCap     int
		firstName    string
		firstIcon    string
		lastName     string
		lastIcon     string
	}{
		{
			answerHandle: "7",
			shopID:       "skill1",
			vocation:     "战士",
			skillCap:     22,
			firstName:    "武器专精",
			firstIcon:    "631.png",
			lastName:     "奥义.六合棍法",
			lastIcon:     "657.png",
		},
		{
			answerHandle: "8",
			shopID:       "skill2",
			vocation:     "术士",
			skillCap:     24,
			firstName:    "苦心经",
			firstIcon:    "692.png",
			lastName:     "意志打击",
			lastIcon:     "722.png",
		},
		{
			answerHandle: "9",
			shopID:       "skill3",
			vocation:     "游侠",
			skillCap:     26,
			firstName:    "武器娴熟",
			firstIcon:    "661.png",
			lastName:     "奥义.修罗幻翼拳",
			lastIcon:     "680.png",
		},
	}

	for _, tt := range cases {
		t.Run(tt.vocation, func(t *testing.T) {
			store, socketSession := seedSelectedRoleSession(t)
			result := handlePacketWithSession(store, protocol.Packet{
				Cmd: cmdClassicTownAnswerReq,
				Seq: 2,
				Payload: mustJSON(t, classicTownAnswerRequest{
					Handle:       "1000542608713897",
					MsgHandle:    "10",
					AnswerHandle: tt.answerHandle,
				}),
			}, socketSession)

			if !result.handled {
				t.Fatal("expected skill category answer to be handled")
			}
			if result.answerSpeak != nil {
				t.Fatalf("expected skill category not to push dialogue, got %+v", result.answerSpeak)
			}
			if result.skillShop == nil {
				t.Fatal("expected skill category to push skill shop")
			}
			if result.skillShop.ShopID != tt.shopID || result.skillShop.Vocation != tt.vocation || result.skillShop.SkillCap != tt.skillCap {
				t.Fatalf("expected captured %s shop metadata, got %+v", tt.shopID, result.skillShop)
			}
			if len(result.skillShop.Skills) != tt.skillCap {
				t.Fatalf("expected captured %s skill list count %d, got %d", tt.vocation, tt.skillCap, len(result.skillShop.Skills))
			}
			if result.skillShop.SaleCapacity != tt.skillCap || result.skillShop.BuyCapacity != 0 || result.skillShop.MadeCapacity != 0 {
				t.Fatalf("expected structured source shop capacities sale=%d buy=0 made=0, got %+v", tt.skillCap, result.skillShop)
			}
			for _, skill := range result.skillShop.Skills {
				if len(skill.Requirements) == 0 {
					t.Fatalf("expected captured %s skill %s to have source requirements", tt.vocation, skill.Name)
				}
			}
			if result.skillShop.Skills[0].Name != tt.firstName || result.skillShop.Skills[0].Icon != tt.firstIcon {
				t.Fatalf("expected captured %s first skill entry, got %+v", tt.vocation, result.skillShop.Skills[0])
			}
			if len(result.skillShop.Skills[0].Requirements) != 1 || result.skillShop.Skills[0].Requirements[0].Icon != "163.png" || result.skillShop.Skills[0].Requirements[0].Count != 500 {
				t.Fatalf("expected captured %s first skill copper requirement, got %+v", tt.vocation, result.skillShop.Skills[0].Requirements)
			}
			firstRequirement := result.skillShop.Skills[0].Requirements[0]
			if firstRequirement.Description == "" || firstRequirement.ItemType == "" || firstRequirement.ItemLevel == 0 {
				t.Fatalf("expected captured %s first requirement to carry source item tooltip fields, got %+v", tt.vocation, firstRequirement)
			}
			lastSkill := result.skillShop.Skills[len(result.skillShop.Skills)-1]
			if lastSkill.Name != tt.lastName || lastSkill.Icon != tt.lastIcon {
				t.Fatalf("expected captured %s last skill entry, got %+v", tt.vocation, lastSkill)
			}
			if tt.answerHandle == "7" {
				silverSkill := result.skillShop.Skills[5]
				if len(silverSkill.Requirements) != 1 || silverSkill.Requirements[0].Icon != "39.png" || silverSkill.Requirements[0].Count != 1 {
					t.Fatalf("expected captured warrior silver requirement, got %+v", silverSkill.Requirements)
				}
			}
		})
	}
}

func TestHandlePacketClassicTownGuangqingSkillTeacherPushesCapturedSkillShop(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)

	categoryResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownAnswerReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownAnswerRequest{
			Handle:       "2220542612946566",
			MsgHandle:    "1",
			AnswerHandle: "1",
		}),
	}, socketSession)
	if categoryResult.answerSpeak == nil || categoryResult.answerSpeak.MsgHandle != "10" {
		t.Fatalf("expected captured Guangqing skill category dialogue, got %+v", categoryResult.answerSpeak)
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownAnswerReq,
		Seq: 3,
		Payload: mustJSON(t, classicTownAnswerRequest{
			Handle:       "2220542612946566",
			MsgHandle:    "10",
			AnswerHandle: "7",
		}),
	}, socketSession)
	if result.skillShop == nil {
		t.Fatal("expected Guangqing skill teacher to push skill shop")
	}
	if result.skillShop.Handle != "2220542612946566" || result.skillShop.ShopID != "skill1" || result.skillShop.Title != "战士技能" {
		t.Fatalf("expected captured Guangqing warrior skill shop, got %+v", result.skillShop)
	}
	if result.skillShop.RoleName != "夏侯武" || result.skillShop.SourceRoleName != "夏侯武" {
		t.Fatalf("expected source NPC role name for Guangqing skill shop, got %+v", result.skillShop)
	}
	if len(result.skillShop.Skills) != 22 || result.skillShop.Skills[0].Name != "武器专精" {
		t.Fatalf("expected captured Guangqing warrior skill rows, got %+v", result.skillShop.Skills)
	}
	if result.skillShop.SaleCapacity != 22 || result.skillShop.BuyCapacity != 0 || result.skillShop.MadeCapacity != 0 {
		t.Fatalf("expected structured Guangqing shop capacities sale=22 buy=0 made=0, got %+v", result.skillShop)
	}
}

func TestHandlePacketClassicTownBaiyuanSkillTeacherPushesCapturedSkillShop(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)

	categoryResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownAnswerReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownAnswerRequest{
			Handle:       baiyuanSkillTeacherHandle,
			MsgHandle:    "1",
			AnswerHandle: "1",
		}),
	}, socketSession)
	if categoryResult.answerSpeak == nil || categoryResult.answerSpeak.MsgHandle != "10" {
		t.Fatalf("expected captured Baiyuan skill category dialogue, got %+v", categoryResult.answerSpeak)
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownAnswerReq,
		Seq: 3,
		Payload: mustJSON(t, classicTownAnswerRequest{
			Handle:       baiyuanSkillTeacherHandle,
			MsgHandle:    "10",
			AnswerHandle: "7",
		}),
	}, socketSession)
	if result.skillShop == nil {
		t.Fatal("expected Baiyuan skill teacher to push skill shop")
	}
	if result.skillShop.Handle != baiyuanSkillTeacherHandle || result.skillShop.ShopID != "skill1" {
		t.Fatalf("expected captured Baiyuan warrior skill shop, got %+v", result.skillShop)
	}
	if len(result.skillShop.Skills) != 22 || result.skillShop.Skills[0].Name != "武器专精" {
		t.Fatalf("expected captured warrior skill rows, got %+v", result.skillShop.Skills)
	}
	if result.skillShop.SaleCapacity != 22 || result.skillShop.BuyCapacity != 0 || result.skillShop.MadeCapacity != 0 {
		t.Fatalf("expected structured Baiyuan shop capacities sale=22 buy=0 made=0, got %+v", result.skillShop)
	}
}

func TestHandlePacketClassicTownGuangqingCapturedItemShopPushesSaleRows(t *testing.T) {
	cases := []struct {
		name           string
		handle         string
		answerHandle   string
		shopID         string
		title          string
		count          int
		firstName      string
		firstIcon      string
		firstReqName   string
		firstReqCost   int
		multiReqID     int
		multiReqName   string
		multiReqNames  []string
		multiReqCounts []int
		roleName       string
		sourceRoleName string
	}{
		{
			name:         "yunyin grocery",
			handle:       "7000542609490978",
			answerHandle: "1",
			shopID:       "item:7000542609490978",
			title:        "丑七品的道具商店",
			count:        12,
			firstName:    "普通采集手套",
			firstIcon:    "856.png",
			firstReqName: "铜钱",
			firstReqCost: 8,
			multiReqID:   11,
			multiReqName: "飞仙洞通行证",
			multiReqNames: []string{
				"铜钱",
			},
			multiReqCounts: []int{150},
			roleName:       "丑七品",
			sourceRoleName: "丑七品",
		},
		{
			name:         "dafo grocery",
			handle:       "4090542614314425",
			answerHandle: "1",
			shopID:       "item:4090542614314425",
			title:        "丑六品的道具商店",
			count:        12,
			firstName:    "普通采集手套",
			firstIcon:    "856.png",
			firstReqName: "铜钱",
			firstReqCost: 8,
			multiReqID:   11,
			multiReqName: "黄风寨通行证",
			multiReqNames: []string{
				"铜钱",
			},
			multiReqCounts: []int{150},
		},
		{
			name:         "jianting grocery",
			handle:       "4960542616750900",
			answerHandle: "1",
			shopID:       "item:4960542616750900",
			title:        "介象的道具商店",
			count:        12,
			firstName:    "普通采集手套",
			firstIcon:    "856.png",
			firstReqName: "铜钱",
			firstReqCost: 8,
			multiReqID:   11,
			multiReqName: "水帘洞通行证",
			multiReqNames: []string{
				"铜钱",
			},
			multiReqCounts: []int{150},
		},
		{
			name:         "weapon",
			handle:       "1780542610743555",
			answerHandle: "1",
			shopID:       "item:1780542610743555",
			title:        "伏天的武器商店",
			count:        10,
			firstName:    "蛮力钢剑",
			firstIcon:    "40.png",
			firstReqName: "铜钱",
			firstReqCost: 500,
			multiReqID:   9,
			multiReqName: "银锭",
			multiReqNames: []string{
				"铜钱",
				"银元宝",
			},
			multiReqCounts: []int{10, 10},
		},
		{
			name:         "grocery",
			handle:       "1820542611400955",
			answerHandle: "1",
			shopID:       "item:1820542611400955",
			title:        "丑五品的道具商店",
			count:        11,
			firstName:    "普通采集手套",
			firstIcon:    "856.png",
			firstReqName: "铜钱",
			firstReqCost: 8,
			multiReqID:   0,
			multiReqNames: []string{
				"铜钱",
			},
			multiReqCounts: []int{8},
		},
		{
			name:         "yunyin armor",
			handle:       "4000542609162635",
			answerHandle: "1",
			shopID:       "item:4000542609162635",
			title:        "布衣娘的护具商店",
			count:        27,
			firstName:    "布帽",
			firstIcon:    "293.png",
			firstReqName: "铜钱",
			firstReqCost: 20,
			multiReqID:   26,
			multiReqName: "飞艳护腕",
			multiReqNames: []string{
				"铜钱",
			},
			multiReqCounts: []int{150},
		},
		{
			name:         "craft",
			handle:       "1830542611405809",
			answerHandle: "1",
			shopID:       "item:1830542611405809",
			title:        "八卦炉合成",
			count:        10,
			firstName:    "狰狞神骑",
			firstIcon:    "130.png",
			firstReqName: "狰狞的头",
			firstReqCost: 99,
			multiReqID:   0,
			multiReqName: "狰狞神骑",
			multiReqNames: []string{
				"狰狞的头",
				"狰狞的皮",
				"狰狞的尾",
				"狰狞的爪",
				"狰狞精魄",
			},
			multiReqCounts: []int{99, 99, 99, 99, 100},
			roleName:       "通天八卦炉",
			sourceRoleName: "通天八卦炉<ma>",
		},
		{
			name:         "armor",
			handle:       "2500542613172144",
			answerHandle: "1",
			shopID:       "item:2500542613172144",
			title:        "云衣娘的护具商店",
			count:        26,
			firstName:    "蛮力面甲",
			firstIcon:    "334.png",
			firstReqName: "铜钱",
			firstReqCost: 400,
			multiReqID:   22,
			multiReqName: "红缨",
			multiReqNames: []string{
				"铜钱",
				"丝",
				"兽血",
			},
			multiReqCounts: []int{10, 5, 1},
		},
		{
			name:         "healer",
			handle:       "2520542613299551",
			answerHandle: "1",
			shopID:       "item:2520542613299551",
			title:        "无颜的药品商店",
			count:        9,
			firstName:    "馒头",
			firstIcon:    "0.png",
			firstReqName: "铜钱",
			firstReqCost: 10,
			multiReqID:   0,
			multiReqName: "馒头",
			multiReqNames: []string{
				"铜钱",
			},
			multiReqCounts: []int{10},
		},
		{
			name:         "baiyuan healer",
			handle:       "4710542615621525",
			answerHandle: "1",
			shopID:       "item:4710542615621525",
			title:        "向隐的药品商店",
			count:        9,
			firstName:    "馒头",
			firstIcon:    "0.png",
			firstReqName: "铜钱",
			firstReqCost: 10,
			multiReqID:   3,
			multiReqName: "小包还元散",
			multiReqNames: []string{
				"铜钱",
			},
			multiReqCounts: []int{350},
		},
		{
			name:         "baiyuan weapon",
			handle:       "5300542617580783",
			answerHandle: "2",
			shopID:       "item:5300542617580783:2",
			title:        "苦虚无的武器商店",
			count:        10,
			firstName:    "真虹剑",
			firstIcon:    "49.png",
			firstReqName: "银元宝",
			firstReqCost: 5,
			multiReqID:   8,
			multiReqName: "铜块",
			multiReqNames: []string{
				"铜钱",
				"铜钱",
			},
			multiReqCounts: []int{10, 1000},
		},
		{
			name:         "baiyuan armor",
			handle:       "5300542617580783",
			answerHandle: "3",
			shopID:       "item:5300542617580783:3",
			title:        "苦虚无的护具商店",
			count:        24,
			firstName:    "无双头盔",
			firstIcon:    "357.png",
			firstReqName: "铜钱",
			firstReqCost: 800,
			multiReqID:   21,
			multiReqName: "铁块",
			multiReqNames: []string{
				"铜钱",
				"碎铁矿",
			},
			multiReqCounts: []int{10, 10},
		},
		{
			name:         "baiyuan grocery",
			handle:       "5310542617702520",
			answerHandle: "1",
			shopID:       "item:5310542617702520",
			title:        "游氏子的道具商店",
			count:        11,
			firstName:    "普通采集手套",
			firstIcon:    "856.png",
			firstReqName: "铜钱",
			firstReqCost: 8,
			multiReqID:   7,
			multiReqName: "穿甲箭",
			multiReqNames: []string{
				"铜钱",
			},
			multiReqCounts: []int{80},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			store, socketSession := seedSelectedRoleSession(t)
			result := handlePacketWithSession(store, protocol.Packet{
				Cmd: cmdClassicTownAnswerReq,
				Seq: 2,
				Payload: mustJSON(t, classicTownAnswerRequest{
					Handle:       tt.handle,
					MsgHandle:    "1",
					AnswerHandle: tt.answerHandle,
				}),
			}, socketSession)

			if !result.handled {
				t.Fatal("expected captured item shop answer to be handled")
			}
			if result.skillShop == nil {
				t.Fatal("expected captured item shop to reuse shop push")
			}
			if result.skillShop.ShopID != tt.shopID || result.skillShop.Title != tt.title || result.skillShop.SkillCap != tt.count {
				t.Fatalf("expected captured item shop metadata, got %+v", result.skillShop)
			}
			if tt.roleName != "" && (result.skillShop.RoleName != tt.roleName || result.skillShop.SourceRoleName != tt.sourceRoleName) {
				t.Fatalf("expected source NPC role names %s/%s, got %+v", tt.roleName, tt.sourceRoleName, result.skillShop)
			}
			if len(result.skillShop.Skills) != tt.count {
				t.Fatalf("expected captured item count %d, got %d", tt.count, len(result.skillShop.Skills))
			}
			if result.skillShop.SaleCapacity != tt.count || result.skillShop.BuyCapacity != 0 || result.skillShop.MadeCapacity != 0 {
				t.Fatalf("expected structured item shop capacities sale=%d buy=0 made=0, got %+v", tt.count, result.skillShop)
			}
			first := result.skillShop.Skills[0]
			if first.Name != tt.firstName || first.Icon != tt.firstIcon {
				t.Fatalf("expected captured first item, got %+v", first)
			}
			if len(first.Requirements) == 0 || first.Requirements[0].Name != tt.firstReqName || first.Requirements[0].Count != tt.firstReqCost {
				t.Fatalf("expected captured first item price, got %+v", first.Requirements)
			}
			var multiReq classicTownSkillShopEntry
			for _, entry := range result.skillShop.Skills {
				if entry.ID == tt.multiReqID {
					multiReq = entry
					break
				}
			}
			if len(multiReq.Requirements) != len(tt.multiReqNames) {
				t.Fatalf("expected captured multi requirement count %d, got %+v", len(tt.multiReqNames), multiReq.Requirements)
			}
			if tt.multiReqName != "" && multiReq.Name != tt.multiReqName {
				t.Fatalf("expected captured item %s at id %d, got %+v", tt.multiReqName, tt.multiReqID, multiReq)
			}
			for index, expectedName := range tt.multiReqNames {
				requirement := multiReq.Requirements[index]
				if requirement.Name != expectedName || requirement.Count != tt.multiReqCounts[index] {
					t.Fatalf("expected captured requirement %d %s x%d, got %+v", index, expectedName, tt.multiReqCounts[index], requirement)
				}
			}
		})
	}
}

func TestHandlePacketClassicTownTransportAnswerPushesMapBootstrap(t *testing.T) {
	store := session.NewStore()
	login := store.Login(session.LoginRequest{
		UserName: "mockuser",
		Password: "magicpwd",
	})
	create := store.CreateRole(session.RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "场景传送测试",
		Gender:         "female",
		RoleTemplateID: 1,
	})
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
	if !selectResult.handled || selectResult.townBootstrap == nil {
		t.Fatalf("expected role select to seed town bootstrap, got %+v", selectResult)
	}

	transportResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownAnswerReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownAnswerRequest{
			Handle:       "transp_4",
			MsgHandle:    "1",
			AnswerHandle: "goto",
		}),
	}, socketSession)

	if !transportResult.handled {
		t.Fatal("expected transport answer to be handled")
	}
	if transportResult.answerSpeak != nil {
		t.Fatalf("expected transport answer not to push dialogue, got %+v", transportResult.answerSpeak)
	}
	if transportResult.townBootstrap == nil {
		t.Fatal("expected transport answer to produce town bootstrap")
	}
	if transportResult.townBootstrap.LoadMap.MapID != "4" || transportResult.townBootstrap.LoadMap.MapName != "云隐村口" {
		t.Fatalf("expected map4 bootstrap, got %+v", transportResult.townBootstrap.LoadMap)
	}
	if transportResult.townBootstrap.CreatePlayer.SpawnFlash.X != 175 || transportResult.townBootstrap.CreatePlayer.SpawnFlash.Y != 380 {
		t.Fatalf("expected transport spawn near map4 return point 175,380 got %+v", transportResult.townBootstrap.CreatePlayer.SpawnFlash)
	}
}

func TestHandlePacketClassicTownCrossRolePushesMapBootstrap(t *testing.T) {
	store := session.NewStore()
	login := store.Login(session.LoginRequest{
		UserName: "mockuser",
		Password: "magicpwd",
	})
	create := store.CreateRole(session.RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "触碰传送测试",
		Gender:         "female",
		RoleTemplateID: 1,
	})
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
	if !selectResult.handled || selectResult.townBootstrap == nil {
		t.Fatalf("expected role select to seed town bootstrap, got %+v", selectResult)
	}

	crossResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownCrossRoleReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownRoleInteractionRequest{
			Handle: "transp_4",
			RoleID: "-3",
			Kind:   "npc",
			MapID:  "1",
		}),
	}, socketSession)

	if !crossResult.handled {
		t.Fatal("expected CrossRole to be handled")
	}
	if crossResult.answerSpeak != nil {
		t.Fatalf("expected CrossRole not to push dialogue, got %+v", crossResult.answerSpeak)
	}
	if crossResult.townBootstrap == nil {
		t.Fatal("expected CrossRole to produce town bootstrap")
	}
	if crossResult.townBootstrap.LoadMap.MapID != "4" || crossResult.townBootstrap.LoadMap.MapName != "云隐村口" {
		t.Fatalf("expected map4 bootstrap, got %+v", crossResult.townBootstrap.LoadMap)
	}
	if crossResult.townBootstrap.CreatePlayer.SpawnFlash.X != 175 || crossResult.townBootstrap.CreatePlayer.SpawnFlash.Y != 380 {
		t.Fatalf("expected CrossRole spawn near map4 return point 175,380 got %+v", crossResult.townBootstrap.CreatePlayer.SpawnFlash)
	}
}

func TestHandlePacketClassicTownCrossRoleStartsVisibleMonsterBattle(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	role, playerBase, ok := store.UpdateRoleMap(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, 76)
	if !ok {
		t.Fatal("expected test role map update to feixiandong map76")
	}
	socketSession.selectedRole = &role
	socketSession.playerBase = &playerBase

	const visibleMonsterHandle = "1048675671977626"
	crossResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownCrossRoleReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownRoleInteractionRequest{
			Handle: visibleMonsterHandle,
			RoleID: "-2",
			Kind:   "monster",
			MapID:  "76",
		}),
	}, socketSession)

	if !crossResult.handled || crossResult.battleStart == nil || crossResult.battleCommand == nil {
		t.Fatalf("expected monster CrossRole to start visible battle, got %+v", crossResult)
	}
	if socketSession.battleRuntime == nil || socketSession.battleRuntime.SourceMonsterHandle != visibleMonsterHandle {
		t.Fatalf("expected CrossRole runtime source monster handle %s, got %+v", visibleMonsterHandle, socketSession.battleRuntime)
	}
	enemyNames := make([]string, 0, len(crossResult.battleCells))
	for _, cell := range crossResult.battleCells {
		if cell.Camp == battle.CampEnemy {
			enemyNames = append(enemyNames, cell.Name)
		}
	}
	if strings.Join(enemyNames, ",") != "晶石怪,巨岩魔" {
		t.Fatalf("expected Feixiandong map76 boss group from CrossRole, got %v", enemyNames)
	}
}

func TestHandlePacketClassicTownTransportActiveRolePushesMapBootstrap(t *testing.T) {
	store := session.NewStore()
	login := store.Login(session.LoginRequest{
		UserName: "mockuser",
		Password: "magicpwd",
	})
	create := store.CreateRole(session.RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "传送双击测试",
		Gender:         "female",
		RoleTemplateID: 1,
	})
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
	if !selectResult.handled || selectResult.townBootstrap == nil {
		t.Fatalf("expected role select to seed town bootstrap, got %+v", selectResult)
	}

	activeResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownActiveRoleReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownRoleInteractionRequest{
			Handle: "transp_4",
			RoleID: "-3",
			Kind:   "npc",
			MapID:  "1",
		}),
	}, socketSession)

	if !activeResult.handled {
		t.Fatal("expected transport ActiveRole to be handled")
	}
	if activeResult.answerSpeak != nil {
		t.Fatalf("expected transport ActiveRole not to push dialogue, got %+v", activeResult.answerSpeak)
	}
	if activeResult.townBootstrap == nil {
		t.Fatal("expected transport ActiveRole to produce town bootstrap")
	}
	if activeResult.townBootstrap.LoadMap.MapID != "4" || activeResult.townBootstrap.LoadMap.MapName != "云隐村口" {
		t.Fatalf("expected map4 bootstrap, got %+v", activeResult.townBootstrap.LoadMap)
	}
	if activeResult.townBootstrap.CreatePlayer.SpawnFlash.X != 175 || activeResult.townBootstrap.CreatePlayer.SpawnFlash.Y != 380 {
		t.Fatalf("expected ActiveRole spawn near map4 return point 175,380 got %+v", activeResult.townBootstrap.CreatePlayer.SpawnFlash)
	}
}

func TestHandlePacketClassicTownTransferPushesMapBootstrap(t *testing.T) {
	store := session.NewStore()
	login := store.Login(session.LoginRequest{
		UserName: "mockuser",
		Password: "magicpwd",
	})
	create := store.CreateRole(session.RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "传送测试",
		Gender:         "female",
		RoleTemplateID: 1,
	})
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
	if !selectResult.handled || selectResult.townBootstrap == nil {
		t.Fatalf("expected role select to seed town bootstrap, got %+v", selectResult)
	}

	transferResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownTransferReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownTransferRequest{
			MapID: "430",
			X:     777,
			Y:     555,
		}),
	}, socketSession)

	if !transferResult.handled {
		t.Fatal("expected transfer to be handled")
	}
	if transferResult.townBootstrap == nil {
		t.Fatal("expected transfer to produce town bootstrap")
	}
	if transferResult.townBootstrap.LoadMap.MapID != "430" || transferResult.townBootstrap.LoadMap.XMLURL != "xml/430.xml" {
		t.Fatalf("expected map430 bootstrap, got %+v", transferResult.townBootstrap.LoadMap)
	}
	if transferResult.townBootstrap.CreatePlayer.SpawnFlash.X != 777 || transferResult.townBootstrap.CreatePlayer.SpawnFlash.Y != 555 {
		t.Fatalf("expected source transfer spawn 777,555 got %+v", transferResult.townBootstrap.CreatePlayer.SpawnFlash)
	}
	if len(transferResult.townBootstrap.CreateRoles) != 0 {
		t.Fatalf("expected generated map source roles to stay empty until captured, got %d", len(transferResult.townBootstrap.CreateRoles))
	}
}

func TestHandlePacketLoginInvalidPayload(t *testing.T) {
	store := session.NewStore()

	result := handlePacket(store, protocol.Packet{
		Cmd:     cmdAuthLoginRequest,
		Seq:     1,
		Payload: []byte("{invalid-json"),
	})

	if result.handled {
		t.Fatal("expected invalid payload to be rejected")
	}
}

func TestHandlePacketRoleFlow(t *testing.T) {
	store := session.NewStore()
	login := store.Login(session.LoginRequest{
		UserName: "mockuser",
		Password: "magicpwd",
	})
	if !login.Success {
		t.Fatalf("expected login success, got %+v", login)
	}

	createResult := handlePacket(store, protocol.Packet{
		Cmd: cmdRoleCreateRequest,
		Seq: 2,
		Payload: mustJSON(t, session.RoleCreateRequest{
			PlayerID:       login.PlayerID,
			SessionToken:   login.SessionToken,
			DisplayName:    "测试女侠",
			Gender:         "female",
			RoleTemplateID: 1,
		}),
	})
	if !createResult.handled {
		t.Fatal("expected create role packet to be handled")
	}
	if createResult.responseCmd != cmdRoleCreateResponse {
		t.Fatalf("expected response cmd %d, got %d", cmdRoleCreateResponse, createResult.responseCmd)
	}

	var createResponse session.RoleCreateResponse
	decodeJSON(t, createResult.responsePayload, &createResponse)
	if !createResponse.Success || createResponse.Role.RoleID == "" {
		t.Fatalf("expected created role id to be non-empty, got %+v", createResponse)
	}

	listResult := handlePacket(store, protocol.Packet{
		Cmd: cmdRoleListRequest,
		Seq: 3,
		Payload: mustJSON(t, session.RoleListRequest{
			PlayerID:     login.PlayerID,
			SessionToken: login.SessionToken,
		}),
	})
	if !listResult.handled {
		t.Fatal("expected list role packet to be handled")
	}
	if listResult.responseCmd != cmdRoleListResponse {
		t.Fatalf("expected response cmd %d, got %d", cmdRoleListResponse, listResult.responseCmd)
	}

	var listResponse session.RoleListResponse
	decodeJSON(t, listResult.responsePayload, &listResponse)
	if !listResponse.Success || len(listResponse.Roles) != 1 {
		t.Fatalf("expected exactly one role, got %+v", listResponse)
	}

	selectResult := handlePacket(store, protocol.Packet{
		Cmd: cmdRoleSelectRequest,
		Seq: 4,
		Payload: mustJSON(t, session.RoleSelectRequest{
			PlayerID:     login.PlayerID,
			SessionToken: login.SessionToken,
			RoleID:       createResponse.Role.RoleID,
		}),
	})
	if !selectResult.handled {
		t.Fatal("expected select role packet to be handled")
	}
	if selectResult.responseCmd != cmdRoleSelectResponse {
		t.Fatalf("expected response cmd %d, got %d", cmdRoleSelectResponse, selectResult.responseCmd)
	}

	var selectResponse session.RoleSelectResponse
	decodeJSON(t, selectResult.responsePayload, &selectResponse)
	if !selectResponse.Success {
		t.Fatalf("expected select role success, got %+v", selectResponse)
	}
	if selectResponse.Role.RoleID != createResponse.Role.RoleID {
		t.Fatalf("expected selected role id %q, got %q", createResponse.Role.RoleID, selectResponse.Role.RoleID)
	}
	if selectResponse.PlayerBase.PlayerID != login.PlayerID {
		t.Fatalf("expected player base player id %q, got %q", login.PlayerID, selectResponse.PlayerBase.PlayerID)
	}
	if selectResult.townBootstrap == nil {
		t.Fatal("expected classic town bootstrap on successful role select")
	}
	if selectResult.townBootstrap.CreatePlayer.SpawnFlash.X != 820 || selectResult.townBootstrap.CreatePlayer.SpawnFlash.Y != 451 {
		t.Fatalf("expected createPlayer spawn 820,451 got %+v", selectResult.townBootstrap.CreatePlayer.SpawnFlash)
	}
	if len(selectResult.townBootstrap.CreateRoles) != 13 {
		t.Fatalf("expected 13 world createRole pushes, got %d", len(selectResult.townBootstrap.CreateRoles))
	}
	if len(selectResult.townBootstrap.QuestStates) != 13 {
		t.Fatalf("expected 13 quest state pushes, got %d", len(selectResult.townBootstrap.QuestStates))
	}

	removeResult := handlePacket(store, protocol.Packet{
		Cmd: cmdRoleRemoveRequest,
		Seq: 5,
		Payload: mustJSON(t, session.RoleRemoveRequest{
			PlayerID:     login.PlayerID,
			SessionToken: login.SessionToken,
			RoleID:       createResponse.Role.RoleID,
			Password:     "magicpwd",
		}),
	})
	if !removeResult.handled {
		t.Fatal("expected remove role packet to be handled")
	}
	if removeResult.responseCmd != cmdRoleRemoveResponse {
		t.Fatalf("expected response cmd %d, got %d", cmdRoleRemoveResponse, removeResult.responseCmd)
	}

	var removeResponse session.RoleRemoveResponse
	decodeJSON(t, removeResult.responsePayload, &removeResponse)
	if !removeResponse.Success {
		t.Fatalf("expected remove role success, got failure: %+v", removeResponse)
	}
	if removeResponse.RemovedRoleID != createResponse.Role.RoleID {
		t.Fatalf("expected removed role id %q, got %q", createResponse.Role.RoleID, removeResponse.RemovedRoleID)
	}

	listAfterDeleteResult := handlePacket(store, protocol.Packet{
		Cmd: cmdRoleListRequest,
		Seq: 6,
		Payload: mustJSON(t, session.RoleListRequest{
			PlayerID:     login.PlayerID,
			SessionToken: login.SessionToken,
		}),
	})
	if !listAfterDeleteResult.handled {
		t.Fatal("expected list role packet after delete to be handled")
	}
	if listAfterDeleteResult.responseCmd != cmdRoleListResponse {
		t.Fatalf("expected response cmd %d, got %d", cmdRoleListResponse, listAfterDeleteResult.responseCmd)
	}

	var listResponseAfterDelete session.RoleListResponse
	decodeJSON(t, listAfterDeleteResult.responsePayload, &listResponseAfterDelete)
	if len(listResponseAfterDelete.Roles) != 0 {
		t.Fatalf("expected no roles after delete, got %d", len(listResponseAfterDelete.Roles))
	}
}

func TestHandlePacketRoleListInvalidSession(t *testing.T) {
	store := session.NewStore()

	result := handlePacket(store, protocol.Packet{
		Cmd: cmdRoleListRequest,
		Seq: 7,
		Payload: mustJSON(t, session.RoleListRequest{
			PlayerID:     "mock-player-001",
			SessionToken: "invalid-session-token",
		}),
	})

	if !result.handled {
		t.Fatal("expected invalid-session role list packet to still be handled")
	}
	if result.responseCmd != cmdRoleListResponse {
		t.Fatalf("expected response cmd %d, got %d", cmdRoleListResponse, result.responseCmd)
	}

	var response session.RoleListResponse
	decodeJSON(t, result.responsePayload, &response)
	if response.Success {
		t.Fatalf("expected invalid session failure, got %+v", response)
	}
	if response.ErrorCode != "6" {
		t.Fatalf("expected invalid session error code 6, got %q", response.ErrorCode)
	}
	if response.ErrorMessage != "登录状态已失效，请重新登录。" {
		t.Fatalf("expected invalid session message, got %q", response.ErrorMessage)
	}
}

func TestHandlePacketClassicBattleStartPushesServerOwnedBattle(t *testing.T) {
	_, socketSession := seedSelectedRoleSession(t)

	result := handlePacketWithSession(session.NewStore(), protocol.Packet{
		Cmd: cmdClassicBattleStartReq,
		Seq: 2,
		Payload: mustJSON(t, battle.StartRequest{
			MapID:       "5",
			MapName:     "云隐村口_1",
			StageFocusX: 320,
			ReturnRoute: "town-placeholder",
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected StartBattle request to be handled")
	}
	if result.battleStart == nil || !strings.HasPrefix(result.battleStart.BattleID, "server-") {
		t.Fatalf("expected server battle start push, got %+v", result.battleStart)
	}
	if len(result.battleCells) != 2 {
		t.Fatalf("expected player and enemy BattleCellInfo pushes, got %+v", result.battleCells)
	}
	if result.battleCommand == nil || result.battleCommand.ActorHandle != socketSession.selectedRole.RoleID {
		t.Fatalf("expected player startCommand, got %+v", result.battleCommand)
	}
	if socketSession.battleRuntime == nil || socketSession.battleRuntime.BattleID != result.battleStart.BattleID {
		t.Fatalf("expected socket battle runtime to be retained, got %+v", socketSession.battleRuntime)
	}
}

func TestHandlePacketClassicBattleStartSupportsCapturedBambooMaps(t *testing.T) {
	cases := []struct {
		mapID      string
		mapName    string
		enemyName  string
		displayURL string
	}{
		{mapID: "84", mapName: "竹林_1", enemyName: "绿甲螳螂", displayURL: "monstermap/greenmantis.swf"},
		{mapID: "86", mapName: "竹林_3", enemyName: "小竹妖", displayURL: "monstermap/bambooboy.swf"},
		{mapID: "90", mapName: "竹林_7", enemyName: "竹炮", displayURL: "monstermap/boobomb.swf"},
		{mapID: "97", mapName: "竹林_10", enemyName: "小竹妖", displayURL: "monstermap/bambooboy.swf"},
	}

	for _, testCase := range cases {
		_, socketSession := seedSelectedRoleSession(t)
		result := handlePacketWithSession(session.NewStore(), protocol.Packet{
			Cmd: cmdClassicBattleStartReq,
			Seq: 2,
			Payload: mustJSON(t, battle.StartRequest{
				MapID:       testCase.mapID,
				MapName:     testCase.mapName,
				StageFocusX: 320,
				ReturnRoute: "town-placeholder",
			}),
		}, socketSession)

		if !result.handled || result.battleStart == nil || result.battleCommand == nil {
			t.Fatalf("expected bamboo StartBattle %s to be handled, got %+v", testCase.mapID, result)
		}
		if len(result.battleCells) != 2 {
			t.Fatalf("expected bamboo map %s to push two battle cells, got %+v", testCase.mapID, result.battleCells)
		}
		enemy := result.battleCells[1]
		if enemy.Name != testCase.enemyName || enemy.DisplayURL != testCase.displayURL {
			t.Fatalf("expected bamboo map %s enemy %s/%s, got %+v", testCase.mapID, testCase.enemyName, testCase.displayURL, enemy)
		}
	}
}

func TestHandlePacketClassicBattleActionResolvesAndRejectsConsumedSequence(t *testing.T) {
	_, socketSession := seedSelectedRoleSession(t)
	startResult := handlePacketWithSession(session.NewStore(), protocol.Packet{
		Cmd: cmdClassicBattleStartReq,
		Seq: 2,
		Payload: mustJSON(t, battle.StartRequest{
			MapID:       "4",
			MapName:     "云隐村口",
			StageFocusX: 120,
			ReturnRoute: "town-placeholder",
		}),
	}, socketSession)
	if startResult.battleStart == nil || startResult.battleCommand == nil {
		t.Fatalf("expected battle start result, got %+v", startResult)
	}

	result := handlePacketWithSession(session.NewStore(), protocol.Packet{
		Cmd: cmdClassicBattleActionReq,
		Seq: 3,
		Payload: mustJSON(t, battle.ActionRequest{
			BattleID:     startResult.battleStart.BattleID,
			ActorHandle:  socketSession.selectedRole.RoleID,
			CommandID:    "skill-mi-zhan",
			TargetHandle: startResult.battleCells[1].Handle,
			Round:        startResult.battleCommand.Round,
			Sequence:     startResult.battleCommand.Sequence,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected battle action to be handled")
	}
	if len(result.battleActions) == 0 {
		t.Fatalf("expected at least one battleAction push, got %+v", result)
	}
	if result.battleStopCommand == nil ||
		result.battleStopCommand.BattleID != startResult.battleStart.BattleID ||
		!strings.Contains(result.battleStopCommand.SourceCapture, "c_stopListenBattleCommand") {
		t.Fatalf("expected source-backed stopCommand after battleAction, got %+v", result.battleStopCommand)
	}
	if result.battleActions[0].BattleID != startResult.battleStart.BattleID || result.battleActions[0].CommandID != "skill-mi-zhan" || result.battleActions[0].SourceActionLabel != "w8/manycut" {
		t.Fatalf("expected structured 密斩 action, got %+v", result.battleActions[0])
	}
	if result.battleOver != nil || result.battleCommand != nil {
		t.Fatalf("expected battleAction only before BattlePlayOver, got %+v", result)
	}

	playOver := handlePacketWithSession(session.NewStore(), protocol.Packet{
		Cmd: cmdClassicBattlePlayOverReq,
		Seq: 4,
		Payload: mustJSON(t, battle.PlayOverRequest{
			BattleID: startResult.battleStart.BattleID,
		}),
	}, socketSession)
	if !playOver.handled {
		t.Fatal("expected BattlePlayOver to be handled")
	}
	if playOver.battleOver == nil && playOver.battleCommand == nil {
		t.Fatalf("expected over or next startCommand after BattlePlayOver, got %+v", playOver)
	}

	duplicate := handlePacketWithSession(session.NewStore(), protocol.Packet{
		Cmd: cmdClassicBattleActionReq,
		Seq: 5,
		Payload: mustJSON(t, battle.ActionRequest{
			BattleID:     startResult.battleStart.BattleID,
			ActorHandle:  socketSession.selectedRole.RoleID,
			CommandID:    "skill-mi-zhan",
			TargetHandle: startResult.battleCells[1].Handle,
			Round:        startResult.battleCommand.Round,
			Sequence:     startResult.battleCommand.Sequence,
		}),
	}, socketSession)
	if !duplicate.handled {
		t.Fatal("expected duplicate action to be handled")
	}
	if len(duplicate.battleActions) != 0 || duplicate.battleCommand != nil || duplicate.battleOver != nil {
		t.Fatalf("expected consumed sequence to be rejected without pushes, got %+v", duplicate)
	}
}

func TestHandlePacketClassicBattleQiangLiFeiBiaoConsumesDart(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	startResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattleStartReq,
		Seq: 2,
		Payload: mustJSON(t, battle.StartRequest{
			MapID:       "4",
			MapName:     "云隐村口",
			StageFocusX: 120,
			ReturnRoute: "town-placeholder",
		}),
	}, socketSession)
	if startResult.battleStart == nil || startResult.battleCommand == nil {
		t.Fatalf("expected battle start result, got %+v", startResult)
	}

	dart := session.RoleItem{
		Type:        "背包",
		Name:        "飞镖",
		ItemType:    "null",
		Display:     "241.png",
		Description: "f_i_飞镖^ffffff&24@材料 消耗品&25@9999&20@铁制三刃飞镖&0;具有杀伤力&0;一般配合技能使用.&27@sitem_jwep&103@0&104@0&105@&107@&108@0",
		Count:       2,
		Index:       -1,
		ItemLevel:   1,
	}
	grantedDart, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, dart)
	if !ok {
		t.Fatal("expected grant 飞镖")
	}

	qiangLiFeiBiaoSkill := []session.RoleSkill{
		{
			Name:        "强力飞镖",
			Level:       2,
			Type:        "oneE",
			Description: "f_s_强力飞镖^ffffff&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@20&4@<font color='#00cc00'>特殊发动条件:需要【飞镖x1】</font><br>进攻时提高48%（无视防御）的物理攻击力",
		},
	}
	socketSession.battleRuntime.RoleSkills = qiangLiFeiBiaoSkill
	socketSession.battleRuntime.RoleSkillsByHandle[socketSession.selectedRole.RoleID] = qiangLiFeiBiaoSkill
	socketSession.battleRuntime.Cells[0].Attack = 100
	socketSession.battleRuntime.Cells[0].MP = 100
	socketSession.battleRuntime.Cells[0].MaxMP = 100
	socketSession.battleRuntime.Cells[0].Fat = 0
	socketSession.battleRuntime.Cells[1].HP = 300
	socketSession.battleRuntime.Cells[1].Defense = 40
	socketSession.battleRuntime.Cells[1].Dog = 0

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattleActionReq,
		Seq: 3,
		Payload: mustJSON(t, battle.ActionRequest{
			BattleID:     startResult.battleStart.BattleID,
			ActorHandle:  socketSession.selectedRole.RoleID,
			CommandID:    battle.CommandQiangLiFeiBiao,
			TargetHandle: socketSession.battleRuntime.Cells[1].Handle,
			Round:        startResult.battleCommand.Round,
			Sequence:     startResult.battleCommand.Sequence,
		}),
	}, socketSession)

	if !result.handled || len(result.battleActions) == 0 {
		t.Fatalf("expected 强力飞镖 battleAction, got %+v", result)
	}
	action := result.battleActions[0]
	if action.ActionName != "强力飞镖" || action.SourceActionLabel != "w3/powerDart" || action.Damage != 148 || action.RefreshInfos[0].MP != 80 {
		t.Fatalf("expected captured 强力飞镖 action and MP cost, got %+v", action)
	}
	if len(result.itemInfos) != 1 || result.itemInfos[0].Name != "飞镖" || result.itemInfos[0].Index != grantedDart.Index || result.itemInfos[0].Count != 1 {
		t.Fatalf("expected 飞镖 itemInfo count 1 after skill use, got %+v", result.itemInfos)
	}
	persisted, ok := store.GetRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, grantedDart.Type, grantedDart.Index)
	if !ok || persisted.Count != 1 {
		t.Fatalf("expected persisted 飞镖 count 1, ok=%v item=%+v", ok, persisted)
	}
}

func TestHandlePacketClassicBattleQiangLiFeiBiaoRejectsMissingDart(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	startResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattleStartReq,
		Seq: 2,
		Payload: mustJSON(t, battle.StartRequest{
			MapID:       "4",
			MapName:     "云隐村口",
			StageFocusX: 120,
			ReturnRoute: "town-placeholder",
		}),
	}, socketSession)
	if startResult.battleStart == nil || startResult.battleCommand == nil {
		t.Fatalf("expected battle start result, got %+v", startResult)
	}

	qiangLiFeiBiaoSkill := []session.RoleSkill{
		{
			Name:        "强力飞镖",
			Level:       2,
			Type:        "oneE",
			Description: "f_s_强力飞镖^ffffff&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@20&4@<font color='#00cc00'>特殊发动条件:需要【飞镖x1】</font><br>进攻时提高48%（无视防御）的物理攻击力",
		},
	}
	socketSession.battleRuntime.RoleSkills = qiangLiFeiBiaoSkill
	socketSession.battleRuntime.RoleSkillsByHandle[socketSession.selectedRole.RoleID] = qiangLiFeiBiaoSkill
	enemyHP := socketSession.battleRuntime.Cells[1].HP

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattleActionReq,
		Seq: 3,
		Payload: mustJSON(t, battle.ActionRequest{
			BattleID:     startResult.battleStart.BattleID,
			ActorHandle:  socketSession.selectedRole.RoleID,
			CommandID:    battle.CommandQiangLiFeiBiao,
			TargetHandle: socketSession.battleRuntime.Cells[1].Handle,
			Round:        startResult.battleCommand.Round,
			Sequence:     startResult.battleCommand.Sequence,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected missing 飞镖 request to be handled")
	}
	if len(result.battleActions) != 0 || len(result.itemInfos) != 0 || len(result.itemClears) != 0 || result.battleCommand == nil {
		t.Fatalf("expected missing 飞镖 to reject with retry startCommand and without mutation pushes, got %+v", result)
	}
	if result.battleCommand.ActorHandle != socketSession.selectedRole.RoleID || result.battleCommand.Sequence != startResult.battleCommand.Sequence {
		t.Fatalf("expected missing 飞镖 retry command to keep current actor/sequence, got %+v", result.battleCommand)
	}
	if len(result.chatMessages) != 0 || len(result.errorMessages) != 1 || result.errorMessages[0].Msg != "物品不足" || !strings.Contains(result.errorMessages[0].SourceCapture, "c_Error 物品不足 after ActiveSkill(122)") {
		t.Fatalf("expected missing 飞镖 source error push only, chats=%+v errors=%+v", result.chatMessages, result.errorMessages)
	}
	if socketSession.battleRuntime.Cells[1].HP != enemyHP || socketSession.battleRuntime.ConsumedSequence[startResult.battleCommand.Sequence] {
		t.Fatalf("expected missing 飞镖 to avoid battle mutation, hp=%d/%d consumed=%v", socketSession.battleRuntime.Cells[1].HP, enemyHP, socketSession.battleRuntime.ConsumedSequence)
	}
}

func TestClassicBattleActionRequiredItemNameIncludesCapturedRangerBowArrows(t *testing.T) {
	if got := classicBattleActionRequiredItemName(battle.CommandGuanJiaLianShi); got != "穿甲箭" {
		t.Fatalf("expected 贯甲连矢 to require 穿甲箭, got %q", got)
	}
	if got := classicBattleActionRequiredItemName(battle.CommandQiangShe); got != "" {
		t.Fatalf("expected 强射 to have no item requirement, got %q", got)
	}
	if got := classicBattleActionRequiredItemName(battle.CommandBingJianSuShe); got != "冰之箭" {
		t.Fatalf("expected 冰箭速射 to require 冰之箭, got %q", got)
	}
	if got := classicBattleActionRequiredItemName(battle.CommandMoLiSuShe); got != "魔箭" {
		t.Fatalf("expected 魔力速射 to require 魔箭, got %q", got)
	}
	if got := classicBattleActionRequiredItemName(battle.CommandAnYingJian); got != "暗之箭" {
		t.Fatalf("expected 暗影箭 to require 暗之箭, got %q", got)
	}
	if got := classicBattleActionRequiredItemName(battle.CommandDuShi); got != "毒箭" {
		t.Fatalf("expected 毒矢 to require 毒箭, got %q", got)
	}
	if got := classicBattleActionRequiredItemName(battle.CommandAoYiHongLeiShi); got != "" {
		t.Fatalf("expected 奥义.轰雷矢 to have no item requirement, got %q", got)
	}
}

func TestHandlePacketClassicBattleTouDuConsumesPoison(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	startResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattleStartReq,
		Seq: 2,
		Payload: mustJSON(t, battle.StartRequest{
			MapID:       "4",
			MapName:     "云隐村口",
			StageFocusX: 120,
			ReturnRoute: "town-placeholder",
		}),
	}, socketSession)
	if startResult.battleStart == nil || startResult.battleCommand == nil {
		t.Fatalf("expected battle start result, got %+v", startResult)
	}

	poison := session.RoleItem{
		Type:        "背包",
		Name:        "毒药",
		ItemType:    "null",
		Display:     "240.png",
		Description: "f_i_毒药^ffffff&24@消耗品&25@999&20@涂在武器上的毒药。&103@0&104@0&105@&107@&108@0",
		Count:       2,
		Index:       -1,
		ItemLevel:   1,
	}
	grantedPoison, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, poison)
	if !ok {
		t.Fatal("expected grant 毒药")
	}

	touDuSkill := []session.RoleSkill{
		{
			Name:        "投毒",
			Level:       1,
			Type:        "oneE",
			Description: "f_s_投毒^5BC46D&9@单体·状态&8@游侠 &10@匕首&22@战斗&2@16&4@<font color='#00cc00'>特殊发动条件:需要【毒药x1】<br>叠加施放将削弱其造成中毒的功效</font><br>有80%的机率使敌人中毒，4回合内降低对方15%的物理防御和魔法防御&0;每回合使敌人损失气力为物理攻击的20%~25%",
		},
	}
	socketSession.battleRuntime.RoleSkills = touDuSkill
	socketSession.battleRuntime.RoleSkillsByHandle[socketSession.selectedRole.RoleID] = touDuSkill
	socketSession.battleRuntime.Cells[0].Attack = 240
	socketSession.battleRuntime.Cells[0].MP = 100
	socketSession.battleRuntime.Cells[0].MaxMP = 100
	socketSession.battleRuntime.Cells[0].Fat = 0
	socketSession.battleRuntime.Cells[1].HP = 300
	socketSession.battleRuntime.Cells[1].Defense = 60
	socketSession.battleRuntime.Cells[1].MgcDefense = 36
	socketSession.battleRuntime.Cells[1].Dog = 0

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattleActionReq,
		Seq: 3,
		Payload: mustJSON(t, battle.ActionRequest{
			BattleID:     startResult.battleStart.BattleID,
			ActorHandle:  socketSession.selectedRole.RoleID,
			CommandID:    battle.CommandTouDu,
			TargetHandle: socketSession.battleRuntime.Cells[1].Handle,
			Round:        startResult.battleCommand.Round,
			Sequence:     startResult.battleCommand.Sequence,
		}),
	}, socketSession)

	if !result.handled || len(result.battleActions) == 0 {
		t.Fatalf("expected 投毒 battleAction, got %+v", result)
	}
	action := result.battleActions[0]
	if action.ActionName != "投毒" || action.SourceActionLabel != "w3/drugAtk" || action.Damage != 1 || action.RefreshInfos[0].MP != 84 {
		t.Fatalf("expected captured 投毒 action and MP cost, got %+v", action)
	}
	if len(result.itemInfos) != 1 || result.itemInfos[0].Name != "毒药" || result.itemInfos[0].Index != grantedPoison.Index || result.itemInfos[0].Count != 1 {
		t.Fatalf("expected 毒药 itemInfo count 1 after skill use, got %+v", result.itemInfos)
	}
	persisted, ok := store.GetRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, grantedPoison.Type, grantedPoison.Index)
	if !ok || persisted.Count != 1 {
		t.Fatalf("expected persisted 毒药 count 1, ok=%v item=%+v", ok, persisted)
	}
}

func TestHandlePacketClassicBattleTouDuRejectsMissingPoison(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	startResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattleStartReq,
		Seq: 2,
		Payload: mustJSON(t, battle.StartRequest{
			MapID:       "4",
			MapName:     "云隐村口",
			StageFocusX: 120,
			ReturnRoute: "town-placeholder",
		}),
	}, socketSession)
	if startResult.battleStart == nil || startResult.battleCommand == nil {
		t.Fatalf("expected battle start result, got %+v", startResult)
	}

	touDuSkill := []session.RoleSkill{
		{
			Name:        "投毒",
			Level:       1,
			Type:        "oneE",
			Description: "f_s_投毒^5BC46D&9@单体·状态&8@游侠 &10@匕首&22@战斗&2@16&4@<font color='#00cc00'>特殊发动条件:需要【毒药x1】<br>叠加施放将削弱其造成中毒的功效</font><br>有80%的机率使敌人中毒，4回合内降低对方15%的物理防御和魔法防御&0;每回合使敌人损失气力为物理攻击的20%~25%",
		},
	}
	socketSession.battleRuntime.RoleSkills = touDuSkill
	socketSession.battleRuntime.RoleSkillsByHandle[socketSession.selectedRole.RoleID] = touDuSkill
	enemyHP := socketSession.battleRuntime.Cells[1].HP

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattleActionReq,
		Seq: 3,
		Payload: mustJSON(t, battle.ActionRequest{
			BattleID:     startResult.battleStart.BattleID,
			ActorHandle:  socketSession.selectedRole.RoleID,
			CommandID:    battle.CommandTouDu,
			TargetHandle: socketSession.battleRuntime.Cells[1].Handle,
			Round:        startResult.battleCommand.Round,
			Sequence:     startResult.battleCommand.Sequence,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected missing 毒药 request to be handled")
	}
	if len(result.battleActions) != 0 || len(result.itemInfos) != 0 || len(result.itemClears) != 0 || result.battleCommand == nil {
		t.Fatalf("expected missing 毒药 to reject with retry startCommand and without mutation pushes, got %+v", result)
	}
	if result.battleCommand.ActorHandle != socketSession.selectedRole.RoleID || result.battleCommand.Sequence != startResult.battleCommand.Sequence {
		t.Fatalf("expected missing 毒药 retry command to keep current actor/sequence, got %+v", result.battleCommand)
	}
	if len(result.chatMessages) != 0 || len(result.errorMessages) != 1 || result.errorMessages[0].Msg != "物品不足" || !strings.Contains(result.errorMessages[0].SourceCapture, "c_Error 物品不足 after ActiveSkill(122)") {
		t.Fatalf("expected missing 毒药 source error push only, chats=%+v errors=%+v", result.chatMessages, result.errorMessages)
	}
	if socketSession.battleRuntime.Cells[1].HP != enemyHP || socketSession.battleRuntime.ConsumedSequence[startResult.battleCommand.Sequence] {
		t.Fatalf("expected missing 毒药 to avoid battle mutation, hp=%d/%d consumed=%v", socketSession.battleRuntime.Cells[1].HP, enemyHP, socketSession.battleRuntime.ConsumedSequence)
	}
}

func TestHandlePacketClassicBattleUtilityCommands(t *testing.T) {
	t.Run("defense reduces the next enemy hit", func(t *testing.T) {
		_, socketSession := seedSelectedRoleSession(t)
		startResult := startClassicBattleForTest(t, socketSession)
		socketSession.battleRuntime.Cells[0].Defense = 10
		socketSession.battleRuntime.Cells[0].Dog = 0
		socketSession.battleRuntime.Cells[1].Attack = 40

		result := handlePacketWithSession(session.NewStore(), protocol.Packet{
			Cmd: cmdClassicBattleActionReq,
			Seq: 3,
			Payload: mustJSON(t, battle.ActionRequest{
				BattleID:     startResult.battleStart.BattleID,
				ActorHandle:  socketSession.selectedRole.RoleID,
				CommandID:    battle.CommandDefense,
				TargetHandle: socketSession.selectedRole.RoleID,
				Round:        startResult.battleCommand.Round,
				Sequence:     startResult.battleCommand.Sequence,
			}),
		}, socketSession)

		if len(result.battleActions) < 2 {
			t.Fatalf("expected defense action and enemy response, got %+v", result.battleActions)
		}
		defenseAction := result.battleActions[0]
		if defenseAction.SourceActionLabel != "def" || defenseAction.Damage != 0 || defenseAction.TargetHandle != socketSession.selectedRole.RoleID {
			t.Fatalf("expected self defense action, got %+v", defenseAction)
		}
		enemyAction := result.battleActions[1]
		if enemyAction.Damage != 20 {
			t.Fatalf("expected doubled defense to reduce enemy damage without stored power bonus, got %+v", enemyAction)
		}
	})

	t.Run("store keeps power for the next player command", func(t *testing.T) {
		_, socketSession := seedSelectedRoleSession(t)
		startResult := startClassicBattleForTest(t, socketSession)

		result := handlePacketWithSession(session.NewStore(), protocol.Packet{
			Cmd: cmdClassicBattleActionReq,
			Seq: 3,
			Payload: mustJSON(t, battle.ActionRequest{
				BattleID:     startResult.battleStart.BattleID,
				ActorHandle:  socketSession.selectedRole.RoleID,
				CommandID:    battle.CommandStore,
				TargetHandle: socketSession.selectedRole.RoleID,
				Round:        startResult.battleCommand.Round,
				Sequence:     startResult.battleCommand.Sequence,
			}),
		}, socketSession)

		if len(result.battleActions) != 2 {
			t.Fatalf("expected store pose then enemy action, got %+v", result.battleActions)
		}
		storeAction := result.battleActions[0]
		if storeAction.CommandID != battle.CommandStore || storeAction.ActionName != "蓄力" || storeAction.SourceActionLabel != "def" {
			t.Fatalf("expected captured store def pose action, got %+v", storeAction)
		}
		if result.battleActions[1].CommandID != battle.CommandEnemyAttack {
			t.Fatalf("expected enemy action after store pose, got %+v", result.battleActions)
		}
		if result.battleCommand != nil || result.battleOver != nil {
			t.Fatalf("expected store enemy action to wait for BattlePlayOver before next command, got %+v", result)
		}

		playOver := handlePacketWithSession(session.NewStore(), protocol.Packet{
			Cmd: cmdClassicBattlePlayOverReq,
			Seq: 4,
			Payload: mustJSON(t, battle.PlayOverRequest{
				BattleID: startResult.battleStart.BattleID,
			}),
		}, socketSession)
		if playOver.battleCommand == nil || playOver.battleCommand.Power != 2 {
			t.Fatalf("expected next startCommand power 2 after BattlePlayOver, got %+v", playOver.battleCommand)
		}
	})

	t.Run("escape plays source action before battle over", func(t *testing.T) {
		_, socketSession := seedSelectedRoleSession(t)
		startResult := startClassicBattleForTest(t, socketSession)

		result := handlePacketWithSession(session.NewStore(), protocol.Packet{
			Cmd: cmdClassicBattleActionReq,
			Seq: 3,
			Payload: mustJSON(t, battle.ActionRequest{
				BattleID:     startResult.battleStart.BattleID,
				ActorHandle:  socketSession.selectedRole.RoleID,
				CommandID:    battle.CommandEscape,
				TargetHandle: socketSession.selectedRole.RoleID,
				Round:        startResult.battleCommand.Round,
				Sequence:     startResult.battleCommand.Sequence,
			}),
		}, socketSession)

		if len(result.battleActions) != 1 || result.battleOver != nil || result.battleCommand != nil {
			t.Fatalf("expected escape source action before over, got %+v", result)
		}
		action := result.battleActions[0]
		if action.CommandID != battle.CommandEscape || action.SourceActionLabel != "escapeSuccess" {
			t.Fatalf("expected escapeSuccess action, got %+v", action)
		}
		if len(result.battleClearCells) != 1 ||
			result.battleClearCells[0].Handle != socketSession.selectedRole.RoleID ||
			result.battleClearCells[0].BattleID != startResult.battleStart.BattleID ||
			!strings.Contains(result.battleClearCells[0].SourceCapture, "c_clearBattleCellInfo") {
			t.Fatalf("expected source clearBattleCellInfo for escaped actor, got %+v", result.battleClearCells)
		}
		if socketSession.battleRuntime == nil {
			t.Fatal("expected battle runtime to wait for BattlePlayOver after escape action")
		}

		playOver := handlePacketWithSession(session.NewStore(), protocol.Packet{
			Cmd: cmdClassicBattlePlayOverReq,
			Seq: 4,
			Payload: mustJSON(t, battle.PlayOverRequest{
				BattleID: startResult.battleStart.BattleID,
			}),
		}, socketSession)

		if playOver.battleOver == nil || playOver.battleOver.Winner != battle.CampEnemy || !playOver.battleOver.Result.Escaped {
			t.Fatalf("expected escaped enemy over result after BattlePlayOver, got %+v", playOver.battleOver)
		}
		if socketSession.battleRuntime != nil {
			t.Fatalf("expected battle runtime to be cleared after escape, got %+v", socketSession.battleRuntime)
		}
	})
}

func TestHandlePacketClassicBattlePersistsRemainingHPMPIntoNextBattle(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	firstStart := startClassicBattleForTest(t, socketSession)
	socketSession.battleRuntime.Cells[0].HP = 123
	socketSession.battleRuntime.Cells[0].MP = 22
	socketSession.battleRuntime.Cells[0].Attack = 80
	socketSession.battleRuntime.Cells[0].Fat = 0
	socketSession.battleRuntime.Cells[1].HP = 10
	socketSession.battleRuntime.Cells[1].Defense = 0
	socketSession.battleRuntime.Cells[1].Dog = 0

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattleActionReq,
		Seq: 3,
		Payload: mustJSON(t, battle.ActionRequest{
			BattleID:     firstStart.battleStart.BattleID,
			ActorHandle:  socketSession.selectedRole.RoleID,
			CommandID:    battle.CommandMiZhan,
			TargetHandle: firstStart.battleCells[1].Handle,
			Round:        firstStart.battleCommand.Round,
			Sequence:     firstStart.battleCommand.Sequence,
		}),
	}, socketSession)

	if len(result.battleActions) != 1 || result.battleOver != nil {
		t.Fatalf("expected winning action to wait for BattlePlayOver, got %+v", result)
	}
	if result.battleActions[0].TargetDead != true || result.battleActions[0].RefreshInfos[0].MP != 17 {
		t.Fatalf("expected 密斩 to kill enemy and reduce MP to 17, got %+v", result.battleActions[0])
	}

	playOver := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattlePlayOverReq,
		Seq: 4,
		Payload: mustJSON(t, battle.PlayOverRequest{
			BattleID: firstStart.battleStart.BattleID,
		}),
	}, socketSession)
	if playOver.battleOver == nil || playOver.battleOver.Winner != battle.CampTeam {
		t.Fatalf("expected team over after first battle, got %+v", playOver)
	}
	if socketSession.playerBase.HP != 123 || socketSession.playerBase.MP != 17 || socketSession.playerBase.RoleState.HP != 123 || socketSession.playerBase.RoleState.MP != 17 {
		t.Fatalf("expected first battle to persist HP/MP 123/17, got playerBase=%+v roleState=%+v", socketSession.playerBase, socketSession.playerBase.RoleState)
	}
	_, persistedBase, ok := store.GetRoleRuntimeData(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID)
	if !ok || persistedBase.RoleState == nil || persistedBase.RoleState.HP != 123 || persistedBase.RoleState.MP != 17 {
		t.Fatalf("expected store to persist battle over HP/MP 123/17, ok=%v playerBase=%+v", ok, persistedBase)
	}

	granted, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, session.RoleItem{
		Type:        "背包",
		Name:        "测试包子",
		ItemType:    "own",
		Display:     "212.png",
		Description: "f_i_测试包子&24@消耗品&25@99&7@60&20@恢复气力",
		Count:       1,
		Index:       -1,
		ItemLevel:   1,
	})
	if !ok {
		t.Fatal("expected HP medicine grant")
	}
	useItem := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownActiveItemReq,
		Seq: 5,
		Payload: mustJSON(t, classicTownActiveItemRequest{
			Type:  granted.Type,
			Index: granted.Index,
		}),
	}, socketSession)
	if useItem.roleState == nil || useItem.roleState.MP != 17 {
		t.Fatalf("expected HP medicine to preserve battle MP 17, got %+v", useItem.roleState)
	}

	secondStart := startClassicBattleForTest(t, socketSession)
	team := secondStart.battleCells[0]
	if team.HP != useItem.roleState.HP || team.MP != 17 {
		t.Fatalf("expected second battle to start from item-updated HP and persisted MP 17, got %+v roleState=%+v", team, useItem.roleState)
	}
}

func TestHandlePacketClassicBattleActiveItemConsumesAndRefreshes(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	startResult := startClassicBattleForTest(t, socketSession)
	socketSession.battleRuntime.Cells[0].HP = 40

	item, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, session.RoleItem{
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
		Seq: 3,
		Payload: mustJSON(t, battle.ItemActionRequest{
			BattleID:    startResult.battleStart.BattleID,
			ActorHandle: socketSession.selectedRole.RoleID,
			Type:        item.Type,
			Index:       item.Index,
			Round:       startResult.battleCommand.Round,
			Sequence:    startResult.battleCommand.Sequence,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected ActiveItem request to be handled")
	}
	if len(result.battleActions) < 2 {
		t.Fatalf("expected useItem action and enemy response, got %+v", result.battleActions)
	}
	itemAction := result.battleActions[0]
	if itemAction.CommandID != battle.CommandItem || itemAction.SourceActionLabel != "useItem" || itemAction.Damage != 0 {
		t.Fatalf("expected useItem battle action, got %+v", itemAction)
	}
	if itemAction.TargetHandle != socketSession.selectedRole.RoleID || itemAction.TargetHP != 75 {
		t.Fatalf("expected item to restore player HP to 75, got %+v", itemAction)
	}
	if len(result.itemInfos) != 1 || result.itemInfos[0].Index != item.Index || result.itemInfos[0].Count != 1 {
		t.Fatalf("expected consumed item count push, got %+v", result.itemInfos)
	}
	if len(result.itemClears) != 0 {
		t.Fatalf("expected stack to remain after consuming one item, got clears %+v", result.itemClears)
	}
	if result.battleCommand != nil || result.battleOver != nil {
		t.Fatalf("expected item action to wait for BattlePlayOver before next command, got %+v", result)
	}
}

func TestHandlePacketClassicBattleActiveItemRestoresMP(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	startResult := startClassicBattleForTest(t, socketSession)
	socketSession.battleRuntime.Cells[0].MP = 30
	socketSession.battleRuntime.Cells[0].MaxMP = 100

	item, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, session.RoleItem{
		Type:        "背包",
		Name:        "小还丹",
		ItemType:    "own",
		Display:     "214.png",
		Description: "f_i_小还丹^ffffff&24@消耗品&25@99&8@25&20@恢复内力",
		Count:       1,
		Index:       21,
		ItemLevel:   1,
	})
	if !ok {
		t.Fatal("expected test item to be granted")
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattleActiveItemReq,
		Seq: 3,
		Payload: mustJSON(t, battle.ItemActionRequest{
			BattleID:    startResult.battleStart.BattleID,
			ActorHandle: socketSession.selectedRole.RoleID,
			Type:        item.Type,
			Index:       item.Index,
			Round:       startResult.battleCommand.Round,
			Sequence:    startResult.battleCommand.Sequence,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected ActiveItem request to be handled")
	}
	if len(result.battleActions) < 2 {
		t.Fatalf("expected useItem action and enemy response, got %+v", result.battleActions)
	}
	itemAction := result.battleActions[0]
	if itemAction.CommandID != battle.CommandItem || itemAction.SourceActionLabel != "useItem" || itemAction.TargetMP != 55 {
		t.Fatalf("expected useItem battle action to restore MP to 55, got %+v", itemAction)
	}
	if len(result.itemClears) != 1 || result.itemClears[0].Index != item.Index {
		t.Fatalf("expected single-use MP item to be cleared, got %+v", result.itemClears)
	}
	if result.battleCommand != nil || result.battleOver != nil {
		t.Fatalf("expected item action to wait for BattlePlayOver before next command, got %+v", result)
	}
}

func TestHandlePacketClassicBattleActiveItemCanReviveTeamTarget(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	startResult := startClassicBattleForTest(t, socketSession)
	socketSession.battleRuntime.Cells = append(socketSession.battleRuntime.Cells, battle.CellInfoPush{
		BattleID:     startResult.battleStart.BattleID,
		Camp:         battle.CampTeam,
		Handle:       "dead-ally",
		Name:         "倒地队友",
		DisplayURL:   socketSession.battleRuntime.Cells[0].DisplayURL,
		Level:        1,
		XScale:       100,
		YScale:       100,
		MaxHP:        100,
		HP:           0,
		MaxMP:        100,
		MP:           10,
		Speed:        90,
		Attack:       8,
		Defense:      2,
		CommandLabel: "普通攻击",
	})

	item, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, session.RoleItem{
		Type:        "背包",
		Name:        "回魂散",
		ItemType:    "oneO",
		Display:     "215.png",
		Description: "f_i_回魂散^ffffff&24@消耗品&25@99&7@30&20@恢复气血",
		Count:       1,
		Index:       22,
		ItemLevel:   1,
	})
	if !ok {
		t.Fatal("expected revive item to be granted")
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattleActiveItemReq,
		Seq: 3,
		Payload: mustJSON(t, battle.ItemActionRequest{
			BattleID:     startResult.battleStart.BattleID,
			ActorHandle:  socketSession.selectedRole.RoleID,
			Type:         item.Type,
			Index:        item.Index,
			TargetHandle: "dead-ally",
			Round:        startResult.battleCommand.Round,
			Sequence:     startResult.battleCommand.Sequence,
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected ActiveItem request to be handled")
	}
	if len(result.battleActions) < 2 {
		t.Fatalf("expected useItem action and enemy response, got %+v", result.battleActions)
	}
	itemAction := result.battleActions[0]
	if itemAction.TargetHandle != "dead-ally" || itemAction.TargetHP != 30 || itemAction.TargetDead {
		t.Fatalf("expected oneO HP item to revive dead ally to 30 HP, got %+v", itemAction)
	}
	if result.battleCommand != nil || result.battleOver != nil {
		t.Fatalf("expected item action to wait for BattlePlayOver before next command, got %+v", result)
	}
}

func TestClassicBattleRelivePushUsesCapturedDoRelivePayload(t *testing.T) {
	over := &battle.OverPush{
		BattleID: "battle-relive-capture",
		Winner:   battle.CampEnemy,
		Rounds:   3,
		Result: battle.ResultPayload{
			Winner: battle.CampEnemy,
			Rounds: 3,
		},
	}
	relive := buildClassicBattleRelivePush(over)
	if relive == nil {
		t.Fatal("expected enemy-win battle over to produce DoRelive push")
	}
	if relive.BattleID != over.BattleID || relive.Ltim <= time.Now().UnixMilli() {
		t.Fatalf("expected relive push to keep battleId and future ltim, got %+v", relive)
	}
	if !strings.Contains(relive.NeedItem, "千年灵芝") || !strings.Contains(relive.NeedItem, "立即原地复活") {
		t.Fatalf("expected captured 千年灵芝 needItem markup, got %s", relive.NeedItem)
	}
	if !strings.Contains(relive.SourceCapture, "c_doRelive") {
		t.Fatalf("expected c_doRelive source capture pointer, got %s", relive.SourceCapture)
	}
	if escaped := buildClassicBattleRelivePush(&battle.OverPush{
		BattleID: "battle-escape",
		Winner:   battle.CampEnemy,
		Result: battle.ResultPayload{
			Winner:  battle.CampEnemy,
			Escaped: true,
		},
	}); escaped != nil {
		t.Fatalf("escape battle over must not open death relive question, got %+v", escaped)
	}
}

func TestClassicBattleLoadProgressUsesCapturedPayload(t *testing.T) {
	progress := buildClassicBattleLoadProgressPush(nil, "battle-load-capture", 142, "")
	if progress == nil {
		t.Fatal("expected battle load progress push")
	}
	if progress.BattleID != "battle-load-capture" || progress.Name != "battleLoad" || progress.Progress != 100 {
		t.Fatalf("expected clamped captured battleLoad progress, got %+v", progress)
	}
	if !strings.Contains(progress.SourceCapture, "c_battleLoadPro") {
		t.Fatalf("expected c_battleLoadPro source capture pointer, got %s", progress.SourceCapture)
	}
	if negative := buildClassicBattleLoadProgressPush(nil, "", -3, "capture"); negative.Progress != 0 {
		t.Fatalf("expected negative progress to clamp to 0, got %+v", negative)
	}
}

func TestClassicBattleCellCountUsesCapturedPKPayload(t *testing.T) {
	cellCount := buildClassicBattleCellCountPush("battle-cell-count-capture", 2, true)
	if cellCount == nil {
		t.Fatal("expected battle cell count push")
	}
	if cellCount.BattleID != "battle-cell-count-capture" || cellCount.Count != 2 || !cellCount.PKWarning {
		t.Fatalf("expected captured battleCellCount shape, got %+v", cellCount)
	}
	if !strings.Contains(cellCount.SourceCapture, "c_battleCellCount") {
		t.Fatalf("expected c_battleCellCount source capture pointer, got %s", cellCount.SourceCapture)
	}
	if negative := buildClassicBattleCellCountPush("", -1, false); negative.Count != 0 || negative.PKWarning {
		t.Fatalf("expected negative cell count to clamp and pk false to stay false, got %+v", negative)
	}
}

func TestHandlePacketClassicBattleLoadProtocol(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	progressResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattleLoadProReq,
		Seq: 4,
		Payload: mustJSON(t, classicBattleLoadProgressRequest{
			BattleID: "battle-load-capture",
			Progress: 87,
		}),
	}, socketSession)

	if !progressResult.handled || progressResult.battleLoadProgress == nil {
		t.Fatalf("expected BattleLoadPro request to return progress push, got %+v", progressResult)
	}
	if progressResult.battleLoadProgress.BattleID != "battle-load-capture" || progressResult.battleLoadProgress.Progress != 87 {
		t.Fatalf("expected BattleLoadPro to echo battleId/progress, got %+v", progressResult.battleLoadProgress)
	}
	if progressResult.teamBattleLoadProgress == nil || progressResult.teamBattleLoadProgress.Progress != 87 {
		t.Fatalf("expected BattleLoadPro to mark team broadcast progress, got %+v", progressResult.teamBattleLoadProgress)
	}
	if !strings.Contains(progressResult.battleLoadProgress.SourceCapture, "BattleLoadPro") {
		t.Fatalf("expected BattleLoadPro capture pointer, got %s", progressResult.battleLoadProgress.SourceCapture)
	}

	readyResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattleRoleReadyReq,
		Seq: 5,
		Payload: mustJSON(t, classicBattleRoleReadyRequest{
			BattleID: "battle-load-capture",
		}),
	}, socketSession)
	if !readyResult.handled || readyResult.battleLoadProgress == nil {
		t.Fatalf("expected BattleRoleReady request to return progress push, got %+v", readyResult)
	}
	if readyResult.battleLoadProgress.Progress != 100 {
		t.Fatalf("expected BattleRoleReady to mark progress 100, got %+v", readyResult.battleLoadProgress)
	}
	if readyResult.teamBattleLoadProgress == nil || readyResult.teamBattleLoadProgress.Progress != 100 {
		t.Fatalf("expected BattleRoleReady to mark team broadcast progress 100, got %+v", readyResult.teamBattleLoadProgress)
	}
	// RoleReady remains report-only; it only fans out progress and never releases commands.
	if len(readyResult.battleCommands) != 0 {
		t.Fatalf("expected RoleReady not to release startCommand, got %+v", readyResult)
	}
	if !strings.Contains(readyResult.battleLoadProgress.SourceCapture, "BattleRoleReady") {
		t.Fatalf("expected BattleRoleReady capture pointer, got %s", readyResult.battleLoadProgress.SourceCapture)
	}
}

func TestHandlePacketClassicBattleDoReliveRejectsMissingItem(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattleDoReliveReq,
		Seq: 4,
		Payload: mustJSON(t, classicBattleDoReliveRequest{
			BattleID: "battle-relive-capture",
		}),
	}, socketSession)

	if !result.handled {
		t.Fatal("expected DoRelive request to be handled")
	}
	if len(result.chatMessages) != 0 || len(result.errorMessages) != 1 {
		t.Fatalf("expected captured missing relive item c_Error push only, got %+v", result)
	}
	reliveError := result.errorMessages[0]
	if reliveError.Msg != classicBattleReliveMissingItemError ||
		!reliveError.Partial ||
		!strings.Contains(reliveError.SourceCapture, "c_Error") {
		t.Fatalf("expected captured missing relive item c_Error push, got %+v", reliveError)
	}
}

func TestHandlePacketClassicQuestLogAndRemove(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	const questTitle = "小试牛刀"
	const questID = "capture-001"

	emptyLog := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetQuestLogReq,
		Seq:     2,
		Payload: mustJSON(t, map[string]any{}),
	}, socketSession)
	if !emptyLog.handled {
		t.Fatal("expected GetQuestLog to be handled")
	}
	if len(emptyLog.questInfos) != 0 {
		t.Fatalf("expected empty quest log before accepting quests, got %+v", emptyLog.questInfos)
	}

	if !store.AcceptQuest(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, questTitle) {
		t.Fatalf("expected to accept quest %s", questTitle)
	}
	logResult := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetQuestLogReq,
		Seq:     3,
		Payload: mustJSON(t, map[string]any{}),
	}, socketSession)
	if !logResult.handled {
		t.Fatal("expected GetQuestLog after accept to be handled")
	}
	if len(logResult.questInfos) != 1 {
		t.Fatalf("expected one accepted quest in log, got %+v", logResult.questInfos)
	}
	var found classicQuestInfoPush
	for _, info := range logResult.questInfos {
		if info.QuestID == questID {
			found = info
			break
		}
	}
	if found.Title != questTitle || found.Level != 1 || found.Type != "main" {
		t.Fatalf("expected catalog quest %s/%s, got %+v", questID, questTitle, found)
	}
	if !strings.Contains(found.Description, "<ml>") || !strings.Contains(found.State, "<over>") {
		t.Fatalf("expected source tags to remain in QuestInfo, got description=%q state=%q", found.Description, found.State)
	}

	removeResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownRemoveQuestReq,
		Seq: 4,
		Payload: mustJSON(t, classicQuestRemoveRequest{
			QuestID: questID,
		}),
	}, socketSession)
	if !removeResult.handled {
		t.Fatal("expected RemoveQuest to be handled")
	}
	if len(removeResult.questClears) != 1 || removeResult.questClears[0].Title != questTitle || removeResult.questClears[0].QuestID != questID {
		t.Fatalf("expected ClearQuestInfo push, got %+v", removeResult.questClears)
	}
	if len(removeResult.questStates) != 0 {
		t.Fatalf("expected no NPC QuestState without a mapped source handle, got %+v", removeResult.questStates)
	}

	duplicate := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownRemoveQuestReq,
		Seq: 5,
		Payload: mustJSON(t, classicQuestRemoveRequest{
			Title: questTitle,
		}),
	}, socketSession)
	if !duplicate.handled {
		t.Fatal("expected duplicate RemoveQuest to be handled")
	}
	if len(duplicate.questClears) != 0 || len(duplicate.questStates) != 0 {
		t.Fatalf("expected duplicate RemoveQuest to produce no pushes, got %+v", duplicate)
	}

	logAfterRemove := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetQuestLogReq,
		Seq:     6,
		Payload: mustJSON(t, map[string]any{}),
	}, socketSession)
	for _, info := range logAfterRemove.questInfos {
		if info.Title == questTitle {
			t.Fatalf("expected removed quest to stay filtered from log, got %+v", logAfterRemove.questInfos)
		}
	}
}

func TestHandlePacketClassicQuestAnswerAcceptsQuest(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownAnswerReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownAnswerRequest{
			Handle:       "7000542609490978",
			MsgHandle:    "1",
			AnswerHandle: "1q23gs",
		}),
	}, socketSession)
	if !result.handled {
		t.Fatal("expected quest Answer to be handled")
	}
	if len(result.questInfos) != 1 || result.questInfos[0].Title != "丑七品的梦" {
		t.Fatalf("expected accepted quest info push, got %+v", result.questInfos)
	}
	if len(result.questStates) != 1 || result.questStates[0].Handle != "7000542609490978" || result.questStates[0].State != 2 {
		t.Fatalf("expected accepted quest NPC state push, got %+v", result.questStates)
	}
	if !packetChatMessagesContain(result.chatMessages, "接受了任务【丑七品的梦】") ||
		!packetChatMessagesContain(result.chatMessages, "日志更新") {
		t.Fatalf("expected accept and quest update chat feedback, got %+v", result.chatMessages)
	}

	logResult := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetQuestLogReq,
		Seq:     3,
		Payload: mustJSON(t, map[string]any{}),
	}, socketSession)
	if len(logResult.questInfos) != 1 || logResult.questInfos[0].Title != "丑七品的梦" {
		t.Fatalf("expected accepted quest to persist in quest log, got %+v", logResult.questInfos)
	}
}

func TestHandlePacketClassicQuestAnswerPushesBaiyuanQuestStateAndTransferBootstrapKeepsIt(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)

	transferToBaiyuan3 := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownTransferReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownTransferRequest{
			MapID: "170",
			X:     1173,
			Y:     430,
		}),
	}, socketSession)
	if !transferToBaiyuan3.handled || transferToBaiyuan3.townBootstrap == nil {
		t.Fatalf("expected transfer to Baiyuan 3 bootstrap, got %+v", transferToBaiyuan3)
	}

	acceptResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownAnswerReq,
		Seq: 3,
		Payload: mustJSON(t, classicTownAnswerRequest{
			Handle:       "5300542617580783",
			MsgHandle:    "1",
			AnswerHandle: "5q22gs",
		}),
	}, socketSession)
	if !acceptResult.handled {
		t.Fatal("expected Baiyuan quest Answer to be handled")
	}
	if len(acceptResult.questInfos) != 1 || acceptResult.questInfos[0].Title != "最后的心愿" {
		t.Fatalf("expected 最后的心愿 QuestInfo, got %+v", acceptResult.questInfos)
	}
	if len(acceptResult.questStates) != 1 || acceptResult.questStates[0].Handle != "4710542615621525" || acceptResult.questStates[0].State != 2 {
		t.Fatalf("expected Baiyuan target NPC quest state push, got %+v", acceptResult.questStates)
	}

	transferToBaiyuan1 := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownTransferReq,
		Seq: 4,
		Payload: mustJSON(t, classicTownTransferRequest{
			MapID: "168",
			X:     1170,
			Y:     434,
		}),
	}, socketSession)
	if !transferToBaiyuan1.handled || transferToBaiyuan1.townBootstrap == nil {
		t.Fatalf("expected transfer to Baiyuan 1 bootstrap, got %+v", transferToBaiyuan1)
	}
	state, ok := questStateForHandle(transferToBaiyuan1.townBootstrap.QuestStates, "4710542615621525")
	if !ok || state != 2 {
		t.Fatalf("expected accepted Baiyuan quest to restore Xiangyin QuestState=2 on transfer bootstrap, got state=%d ok=%v states=%+v", state, ok, transferToBaiyuan1.townBootstrap.QuestStates)
	}
}

func TestHandlePacketRoleSelectOverlaysAcceptedBaiyuanQuestState(t *testing.T) {
	store := session.NewStore()
	login := store.Login(session.LoginRequest{
		UserName: "mockuser",
		Password: "magicpwd",
	})
	create := store.CreateRole(session.RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "白源任务测试",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	if !store.AcceptQuest(login.PlayerID, create.Role.RoleID, "最后的心愿") {
		t.Fatal("expected to seed accepted Baiyuan quest")
	}
	if _, _, ok := store.UpdateRoleMap(login.PlayerID, create.Role.RoleID, 168); !ok {
		t.Fatal("expected to move role to Baiyuan 1")
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdRoleSelectRequest,
		Seq: 2,
		Payload: mustJSON(t, session.RoleSelectRequest{
			PlayerID:     login.PlayerID,
			SessionToken: login.SessionToken,
			RoleID:       create.Role.RoleID,
		}),
	}, &packetSession{})
	if !result.handled || result.townBootstrap == nil {
		t.Fatalf("expected role select bootstrap, got %+v", result)
	}
	state, ok := questStateForHandle(result.townBootstrap.QuestStates, "4710542615621525")
	if !ok || state != 2 {
		t.Fatalf("expected role select bootstrap to show accepted Baiyuan QuestState=2, got state=%d ok=%v states=%+v", state, ok, result.townBootstrap.QuestStates)
	}
}

func TestHandlePacketClassicQuestAnswerPushesCapturedErrorWhenQuestLogFull(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	accepted := 0
	for _, info := range quest.All() {
		if info.Title == "讨厌的枯木怪" {
			continue
		}
		if store.AcceptQuest(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, info.Title) {
			accepted++
		}
		if accepted == classicQuestAcceptedLimit {
			break
		}
	}
	if accepted != classicQuestAcceptedLimit {
		t.Fatalf("expected to seed %d accepted quests, got %d", classicQuestAcceptedLimit, accepted)
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownAnswerReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownAnswerRequest{
			Handle:       "4090542614314425",
			MsgHandle:    "2q23d_1",
			AnswerHandle: "2q23a_1_1",
		}),
	}, socketSession)
	if !result.handled {
		t.Fatal("expected full quest Answer to be handled")
	}
	if len(result.questInfos) != 0 {
		t.Fatalf("expected no QuestInfo when quest log is full, got %+v", result.questInfos)
	}
	if len(result.chatMessages) != 0 || len(result.errorMessages) != 1 {
		t.Fatalf("expected captured quest-full c_Error only, got %+v", result)
	}
	errorMessage := result.errorMessages[0]
	if errorMessage.Msg != "任务最多只能接5个" ||
		!strings.Contains(errorMessage.SourceCapture, "Speak(101)") ||
		!strings.Contains(errorMessage.SourceCapture, "c_Error") ||
		!strings.Contains(errorMessage.SourceCapture, "2q23d_1|2q23a_1_1") {
		t.Fatalf("expected captured quest-full c_Error, got %+v", errorMessage)
	}
}

func TestHandlePacketClassicQuestCompleteGrantsParsedRewards(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	if !store.AcceptQuest(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, "蟾蜍之患") {
		t.Fatal("expected to seed accepted capture-003 quest")
	}
	if !store.AcceptQuest(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, "天象异常") {
		t.Fatal("expected to seed accepted capture-009 quest")
	}

	expAndCurrency := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownRemoveQuestReq,
		Seq: 2,
		Payload: mustJSON(t, classicQuestRemoveRequest{
			QuestID:  "capture-003",
			Complete: true,
		}),
	}, socketSession)
	if !expAndCurrency.handled {
		t.Fatal("expected completed quest request to be handled")
	}
	if len(expAndCurrency.questClears) != 1 || expAndCurrency.questClears[0].QuestID != "capture-003" {
		t.Fatalf("expected completed quest to clear capture-003, got %+v", expAndCurrency.questClears)
	}
	if expAndCurrency.roleState == nil || expAndCurrency.roleState.Exp != 1000 {
		t.Fatalf("expected capture-003 to grant 1000 experience, got %+v", expAndCurrency.roleState)
	}
	if expAndCurrency.roleState.Lv != 5 {
		t.Fatalf("expected capture-003 exp reward to level role to 5, got %+v", expAndCurrency.roleState)
	}
	if expAndCurrency.rolePhysique == nil || expAndCurrency.rolePhysique.MaxHP != 265 || expAndCurrency.rolePhysique.MaxMP != 84 || expAndCurrency.rolePhysique.LastPoint != 25 {
		t.Fatalf("expected capture-003 level-up rolePhysique push, got %+v", expAndCurrency.rolePhysique)
	}
	if expAndCurrency.currencyPush == nil || expAndCurrency.currencyPush.Currencies["铜钱"] != 5200 {
		t.Fatalf("expected capture-003 to grant 200 copper, got %+v", expAndCurrency.currencyPush)
	}
	if !packetChatMessagesContain(expAndCurrency.chatMessages, "完成了任务【蟾蜍之患】") ||
		!packetChatMessagesContain(expAndCurrency.chatMessages, "获得经验:1000") ||
		!packetChatMessagesContain(expAndCurrency.chatMessages, "获得了【铜钱】x200") {
		t.Fatalf("expected capture-003 source system reward messages, got %+v", expAndCurrency.chatMessages)
	}
	if len(expAndCurrency.itemInfos) != 0 {
		t.Fatalf("expected currency reward to avoid bag item push, got %+v", expAndCurrency.itemInfos)
	}

	itemReward := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownRemoveQuestReq,
		Seq: 3,
		Payload: mustJSON(t, classicQuestRemoveRequest{
			QuestID:  "capture-009",
			Complete: true,
		}),
	}, socketSession)
	if itemReward.roleState == nil || itemReward.roleState.Exp != 1300 {
		t.Fatalf("expected capture-009 to add 300 experience after previous reward, got %+v", itemReward.roleState)
	}
	if itemReward.roleState.Lv != 6 {
		t.Fatalf("expected capture-009 exp reward to level role to 6, got %+v", itemReward.roleState)
	}
	if itemReward.rolePhysique == nil || itemReward.rolePhysique.MaxHP != 285 || itemReward.rolePhysique.MaxMP != 94 || itemReward.rolePhysique.LastPoint != 30 {
		t.Fatalf("expected capture-009 level-up rolePhysique push, got %+v", itemReward.rolePhysique)
	}
	if len(itemReward.itemInfos) != 1 || itemReward.itemInfos[0].Name != "L初阶经验卡" || itemReward.itemInfos[0].Count != 1 || itemReward.itemInfos[0].Display != "576.png" {
		t.Fatalf("expected capture-009 item reward push, got %+v", itemReward.itemInfos)
	}
	if !packetChatMessagesContain(itemReward.chatMessages, "完成了任务【天象异常】") ||
		!packetChatMessagesContain(itemReward.chatMessages, "获得经验:300") ||
		!packetChatMessagesContain(itemReward.chatMessages, "获得了【L初阶经验卡】x1") {
		t.Fatalf("expected capture-009 source system reward messages, got %+v", itemReward.chatMessages)
	}
}

func TestHandlePacketClassicQuestCompleteConsumesRequirements(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	const questTitle = "全民锻造"
	const questID = "capture-028"
	if !store.AcceptQuest(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, questTitle) {
		t.Fatalf("expected to seed accepted %s quest", questTitle)
	}
	beforeItems, _, ok := store.GetRoleItems(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, "背包")
	if !ok {
		t.Fatal("expected bag items")
	}
	beforeOreCount := 0
	for _, item := range beforeItems {
		if item.Name == "碎铁矿" {
			beforeOreCount += item.Count
		}
	}
	ore := session.RoleItem{
		Type:        "背包",
		Name:        "碎铁矿",
		ItemType:    "null",
		Display:     "105.png",
		Description: "f_i_碎铁矿^ffffff&24@材料&25@99&20@制造武器和护具的基本素材&101@105.png&103@0&104@0&105@&107@&108@12",
		Count:       20,
		Index:       -1,
		ItemLevel:   1,
	}
	if _, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, ore); !ok {
		t.Fatal("expected 碎铁矿 grant to succeed")
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownRemoveQuestReq,
		Seq: 2,
		Payload: mustJSON(t, classicQuestRemoveRequest{
			QuestID:  questID,
			Complete: true,
		}),
	}, socketSession)
	if !result.handled {
		t.Fatal("expected completed quest request to be handled")
	}
	if len(result.questClears) != 1 || result.questClears[0].QuestID != questID {
		t.Fatalf("expected completed quest to clear %s, got %+v", questID, result.questClears)
	}
	if len(result.itemInfos) == 0 && len(result.itemClears) == 0 {
		t.Fatalf("expected requirement item mutation push, got infos=%+v clears=%+v", result.itemInfos, result.itemClears)
	}
	if result.currencyPush == nil || result.currencyPush.Currencies["银元宝"] != 2 {
		t.Fatalf("expected reward silver ingot currency push, got %+v", result.currencyPush)
	}
	if !packetChatMessagesContain(result.chatMessages, "完成了任务【全民锻造】") ||
		!packetChatMessagesContain(result.chatMessages, "获得了【银元宝】x1") {
		t.Fatalf("expected completion and reward chat messages, got %+v", result.chatMessages)
	}

	afterItems, _, ok := store.GetRoleItems(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, "背包")
	if !ok {
		t.Fatal("expected bag items after completion")
	}
	afterOreCount := 0
	for _, item := range afterItems {
		if item.Name == "碎铁矿" {
			afterOreCount += item.Count
		}
	}
	if afterOreCount != beforeOreCount {
		t.Fatalf("expected quest to consume exactly 20 granted 碎铁矿, before=%d after=%d", beforeOreCount, afterOreCount)
	}
}

func TestHandlePacketClassicQuestAnswerCompletesAcceptedQuest(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	const questTitle = "全民锻造"
	if !store.AcceptQuest(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, questTitle) {
		t.Fatalf("expected to seed accepted %s quest", questTitle)
	}
	ore := session.RoleItem{
		Type:        "背包",
		Name:        "碎铁矿",
		ItemType:    "null",
		Display:     "105.png",
		Description: "f_i_碎铁矿^ffffff&24@材料&25@99&20@制造武器和护具的基本素材&101@105.png&103@0&104@0&105@&107@&108@12",
		Count:       20,
		Index:       -1,
		ItemLevel:   1,
	}
	if _, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, ore); !ok {
		t.Fatal("expected 碎铁矿 grant to succeed")
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownAnswerReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownAnswerRequest{
			Handle:       "2000542608832485",
			MsgHandle:    "1q28d_1",
			AnswerHandle: "1q28a_1_1",
		}),
	}, socketSession)
	if !result.handled {
		t.Fatal("expected quest Answer to be handled")
	}
	if len(result.questClears) != 1 || result.questClears[0].Title != questTitle {
		t.Fatalf("expected accepted quest answer to complete and clear quest, got %+v", result.questClears)
	}
	if result.currencyPush == nil || result.currencyPush.Currencies["银元宝"] != 2 {
		t.Fatalf("expected reward silver ingot currency push, got %+v", result.currencyPush)
	}
	if !packetChatMessagesContain(result.chatMessages, "完成了任务【全民锻造】") {
		t.Fatalf("expected completion chat message, got %+v", result.chatMessages)
	}
}

func TestHandlePacketClassicQuestCompleteRejectsMissingRequirements(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	const questTitle = "全民锻造"
	if !store.AcceptQuest(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, questTitle) {
		t.Fatalf("expected to seed accepted %s quest", questTitle)
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownRemoveQuestReq,
		Seq: 2,
		Payload: mustJSON(t, classicQuestRemoveRequest{
			QuestID:  "capture-028",
			Complete: true,
		}),
	}, socketSession)
	if !result.handled {
		t.Fatal("expected completed quest request to be handled")
	}
	if len(result.questClears) != 0 || result.roleState != nil || result.currencyPush != nil || len(result.itemInfos) != 0 || len(result.itemClears) != 0 {
		t.Fatalf("expected missing requirement to avoid completion mutation pushes, got %+v", result)
	}
	if !packetChatMessagesContain(result.chatMessages, "碎铁矿不足") {
		t.Fatalf("expected missing 碎铁矿 warning, got %+v", result.chatMessages)
	}

	logResult := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetQuestLogReq,
		Seq:     3,
		Payload: mustJSON(t, map[string]any{}),
	}, socketSession)
	if len(logResult.questInfos) != 1 || logResult.questInfos[0].Title != questTitle {
		t.Fatalf("expected quest to remain accepted after rejected completion, got %+v", logResult.questInfos)
	}
}

func TestHandlePacketClassicQuestCompleteRejectsIncompleteState(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	if !store.AcceptQuest(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, "收集朽木") {
		t.Fatal("expected to seed accepted incomplete quest")
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownRemoveQuestReq,
		Seq: 2,
		Payload: mustJSON(t, classicQuestRemoveRequest{
			QuestID:  "capture-008",
			Complete: true,
		}),
	}, socketSession)
	if !result.handled {
		t.Fatal("expected incomplete completed quest request to be handled")
	}
	if len(result.questClears) != 0 || result.roleState != nil || len(result.itemInfos) != 0 || result.currencyPush != nil {
		t.Fatalf("expected incomplete quest completion to avoid mutation pushes, got %+v", result)
	}

	logResult := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicTownGetQuestLogReq,
		Seq:     3,
		Payload: mustJSON(t, map[string]any{}),
	}, socketSession)
	found := false
	for _, info := range logResult.questInfos {
		if info.QuestID == "capture-008" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected rejected incomplete quest to remain in log, got %+v", logResult.questInfos)
	}
}

func TestHandlePacketClassicQuestRequiresSelectedRole(t *testing.T) {
	result := handlePacketWithSession(session.NewStore(), protocol.Packet{
		Cmd:     cmdClassicTownGetQuestLogReq,
		Seq:     2,
		Payload: mustJSON(t, map[string]any{}),
	}, &packetSession{})
	if !result.handled {
		t.Fatal("expected GetQuestLog without selected role to be handled")
	}
	if len(result.questInfos) != 0 {
		t.Fatalf("expected no QuestInfo without selected role, got %+v", result.questInfos)
	}
}

func TestHandlePacketClassicSocialAddRemoveFriendAndBlackList(t *testing.T) {
	_, socketSession := seedSelectedRoleSession(t)

	addFriend := handlePacketWithSession(session.NewStore(), protocol.Packet{
		Cmd: cmdClassicSocialAddFriendReq,
		Seq: 2,
		Payload: mustJSON(t, classicSocialMutateRequest{
			RoleName: "江湖好友",
		}),
	}, socketSession)
	if !addFriend.handled || len(addFriend.friendInfos) != 1 {
		t.Fatalf("expected FriendInfo push, got %+v", addFriend)
	}
	duplicateFriend := handlePacketWithSession(session.NewStore(), protocol.Packet{
		Cmd: cmdClassicSocialAddFriendReq,
		Seq: 3,
		Payload: mustJSON(t, classicSocialMutateRequest{
			RoleName: "江湖好友",
		}),
	}, socketSession)
	if !duplicateFriend.handled || len(duplicateFriend.friendInfos) != 1 {
		t.Fatalf("expected duplicate add to normalize to one FriendInfo push, got %+v", duplicateFriend)
	}
	if len(socketSession.friends) != 1 {
		t.Fatalf("expected normalized friend map size 1, got %d", len(socketSession.friends))
	}

	removeFriend := handlePacketWithSession(session.NewStore(), protocol.Packet{
		Cmd: cmdClassicSocialRemoveFriend,
		Seq: 4,
		Payload: mustJSON(t, classicSocialMutateRequest{
			RoleName: "江湖好友",
		}),
	}, socketSession)
	if !removeFriend.handled || len(removeFriend.friendClears) != 1 {
		t.Fatalf("expected ClearFriendInfo push, got %+v", removeFriend)
	}
	removeMissingFriend := handlePacketWithSession(session.NewStore(), protocol.Packet{
		Cmd: cmdClassicSocialRemoveFriend,
		Seq: 5,
		Payload: mustJSON(t, classicSocialMutateRequest{
			RoleName: "江湖好友",
		}),
	}, socketSession)
	if len(removeMissingFriend.friendClears) != 0 {
		t.Fatalf("expected missing friend removal to produce no clear push, got %+v", removeMissingFriend)
	}

	addBlack := handlePacketWithSession(session.NewStore(), protocol.Packet{
		Cmd: cmdClassicSocialAddBlackReq,
		Seq: 6,
		Payload: mustJSON(t, classicSocialMutateRequest{
			RoleName: "黑名目标",
		}),
	}, socketSession)
	if !addBlack.handled || len(addBlack.blackInfos) != 1 {
		t.Fatalf("expected BlackListInfo push, got %+v", addBlack)
	}
	removeBlack := handlePacketWithSession(session.NewStore(), protocol.Packet{
		Cmd: cmdClassicSocialRemoveBlack,
		Seq: 7,
		Payload: mustJSON(t, classicSocialMutateRequest{
			RoleName: "黑名目标",
		}),
	}, socketSession)
	if !removeBlack.handled || len(removeBlack.blackClears) != 1 {
		t.Fatalf("expected ClearBlackListInfo push, got %+v", removeBlack)
	}
}

func TestHandlePacketClassicSocialGetFriendListReturnsCurrentSnapshot(t *testing.T) {
	_, socketSession := seedSelectedRoleSession(t)

	for index, roleName := range []string{"zeta-friend", "alpha-friend"} {
		result := handlePacketWithSession(session.NewStore(), protocol.Packet{
			Cmd: cmdClassicSocialAddFriendReq,
			Seq: uint64(index + 2),
			Payload: mustJSON(t, classicSocialMutateRequest{
				RoleName: roleName,
			}),
		}, socketSession)
		if !result.handled || len(result.friendInfos) != 1 {
			t.Fatalf("expected add friend push for %s, got %+v", roleName, result)
		}
	}

	result := handlePacketWithSession(session.NewStore(), protocol.Packet{
		Cmd: cmdClassicSocialGetFriendListReq,
		Seq: 4,
	}, socketSession)
	if !result.handled || len(result.friendInfos) != 2 {
		t.Fatalf("expected two FriendInfo pushes from current session, got %+v", result)
	}
	if result.friendInfos[0].RoleName != "alpha-friend" || result.friendInfos[1].RoleName != "zeta-friend" {
		t.Fatalf("expected stable friend-list order, got %+v", result.friendInfos)
	}

	withoutRole := handlePacketWithSession(session.NewStore(), protocol.Packet{
		Cmd: cmdClassicSocialGetFriendListReq,
		Seq: 5,
	}, &packetSession{})
	if !withoutRole.handled || len(withoutRole.friendInfos) != 0 {
		t.Fatalf("expected GetFrienInfos without selected role to return no entries, got %+v", withoutRole)
	}
}

func TestHandlePacketClassicSocialGetFriendListSeedsCapturedFriendInfo(t *testing.T) {
	_, socketSession := seedSelectedRoleSession(t)

	result := handlePacketWithSession(session.NewStore(), protocol.Packet{
		Cmd: cmdClassicSocialGetFriendListReq,
		Seq: 2,
	}, socketSession)
	if !result.handled || len(result.friendInfos) != 1 {
		t.Fatalf("expected captured FriendInfo seed, got %+v", result)
	}
	friend := result.friendInfos[0]
	if friend.RoleName != "恐龙抗狼1" || friend.Level != 29 || friend.MapName != "广青镇_1" || !friend.Online || friend.Relation != "friend" {
		t.Fatalf("expected captured 恐龙抗狼1|45|29|战士 friend mapping, got %+v", friend)
	}

	replay := handlePacketWithSession(session.NewStore(), protocol.Packet{
		Cmd: cmdClassicSocialGetFriendListReq,
		Seq: 3,
	}, socketSession)
	if !replay.handled || len(replay.friendInfos) != 1 || replay.friendInfos[0].RoleName != "恐龙抗狼1" {
		t.Fatalf("expected captured friend seed to persist in current session snapshot, got %+v", replay)
	}
}

func TestHandlePacketClassicSocialAddFriendResolvesStoredRoleAndClearUsesStoredIdentity(t *testing.T) {
	store := session.NewStore()
	ownerSession, _ := seedSelectedRoleSessionInStore(t, store, "好友主人")
	_, targetRole := seedSelectedRoleSessionInStore(t, store, "目标侠客")
	if _, foundRole, ok := store.FindRoleByDisplayName("目标侠客"); !ok || foundRole.RoleID != targetRole.RoleID {
		t.Fatalf("expected target role in store, got ok=%v role=%+v", ok, foundRole)
	}

	addFriend := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicSocialAddFriendReq,
		Seq: 2,
		Payload: mustJSON(t, classicSocialMutateRequest{
			RoleName: "目标侠客",
		}),
	}, ownerSession)
	if !addFriend.handled || len(addFriend.friendInfos) != 1 {
		t.Fatalf("expected FriendInfo push for stored role, got %+v", addFriend)
	}
	friend := addFriend.friendInfos[0]
	if friend.RoleName != "目标侠客" {
		t.Fatalf("expected resolved role name 目标侠客, got %+v", friend)
	}
	if friend.RoleID != targetRole.RoleID || strings.HasPrefix(friend.RoleID, "social-") {
		t.Fatalf("expected real roleId %s from store, got %+v", targetRole.RoleID, friend)
	}
	if friend.Level < 1 {
		t.Fatalf("expected positive level from store role, got %+v", friend)
	}
	if friend.Online {
		t.Fatalf("expected offline stored role to report Online=false, got %+v", friend)
	}
	if friend.Relation != "friend" {
		t.Fatalf("expected relation friend, got %+v", friend)
	}

	removeFriend := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicSocialRemoveFriend,
		Seq: 3,
		Payload: mustJSON(t, classicSocialMutateRequest{
			RoleName: "目标侠客",
		}),
	}, ownerSession)
	if !removeFriend.handled || len(removeFriend.friendClears) != 1 {
		t.Fatalf("expected ClearFriendInfo push, got %+v", removeFriend)
	}
	clear := removeFriend.friendClears[0]
	if clear.RoleID != friend.RoleID || clear.RoleName != "目标侠客" {
		t.Fatalf("expected clear payload to use stored identity roleId=%s roleName=目标侠客, got %+v", friend.RoleID, clear)
	}
	if len(ownerSession.friends) != 0 {
		t.Fatalf("expected friend map empty after remove, got %+v", ownerSession.friends)
	}
}

func TestHandlePacketClassicSocialAddFriendResolvesOnlineHubRole(t *testing.T) {
	swapWorldSceneHub(t)

	store := session.NewStore()
	ownerSession, _ := seedSelectedRoleSessionInStore(t, store, "在线主人")
	targetSession, targetRole := seedSelectedRoleSessionInStore(t, store, "在线目标")
	targetSession.selectedRole.Level = 44
	targetSession.playerBase.Level = 44
	targetSession.selectedRole.MapID = 45
	targetSession.playerBase.MapID = 45

	worldSceneHub.register(targetRole.RoleID, 45, &websocketWriter{}, targetSession, world.SpawnPoint{X: 100, Y: 200})

	addFriend := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicSocialAddFriendReq,
		Seq: 2,
		Payload: mustJSON(t, classicSocialMutateRequest{
			RoleName: "在线目标",
		}),
	}, ownerSession)
	if !addFriend.handled || len(addFriend.friendInfos) != 1 {
		t.Fatalf("expected FriendInfo push for online role, got %+v", addFriend)
	}
	friend := addFriend.friendInfos[0]
	if friend.RoleID != targetRole.RoleID || friend.RoleName != "在线目标" {
		t.Fatalf("expected online hub identity, got %+v", friend)
	}
	if !friend.Online || friend.Level != 44 {
		t.Fatalf("expected online level 44, got %+v", friend)
	}
	if friend.MapName != "广青镇_1" {
		t.Fatalf("expected mapName 广青镇_1 for map 45, got %+v", friend)
	}

	list := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicSocialGetFriendListReq,
		Seq: 3,
	}, ownerSession)
	if !list.handled || len(list.friendInfos) != 1 || !list.friendInfos[0].Online || list.friendInfos[0].MapName != "广青镇_1" {
		t.Fatalf("expected get list to refresh online map/level, got %+v", list)
	}
}

func TestHandlePacketClassicSocialSameNameRoleIDsRemainIndependent(t *testing.T) {
	store := session.NewStore()
	ownerSession, _ := seedSelectedRoleSessionInStore(t, store, "同名好友主人")
	_, firstRole := seedSelectedRoleSessionInStore(t, store, "同名目标")
	_, secondRole := seedSelectedRoleSessionInStore(t, store, "同名目标")

	for index, role := range []session.RoleSummary{firstRole, secondRole} {
		result := handlePacketWithSession(store, protocol.Packet{
			Cmd: cmdClassicSocialAddFriendReq,
			Seq: uint64(index + 2),
			Payload: mustJSON(t, classicSocialMutateRequest{
				RoleID:   role.RoleID,
				RoleName: role.DisplayName,
			}),
		}, ownerSession)
		if !result.handled || len(result.friendInfos) != 1 || result.friendInfos[0].RoleID != role.RoleID {
			t.Fatalf("expected exact same-name roleId %s, got %+v", role.RoleID, result)
		}
	}
	if len(ownerSession.friends) != 2 {
		t.Fatalf("expected two same-name friend roleIds to remain, got %+v", ownerSession.friends)
	}

	remove := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicSocialRemoveFriend,
		Seq: 4,
		Payload: mustJSON(t, classicSocialMutateRequest{
			RoleID:   firstRole.RoleID,
			RoleName: firstRole.DisplayName,
		}),
	}, ownerSession)
	if !remove.handled || len(remove.friendClears) != 1 || remove.friendClears[0].RoleID != firstRole.RoleID {
		t.Fatalf("expected only first same-name role to clear, got %+v", remove)
	}
	if len(ownerSession.friends) != 1 || ownerSession.friends[secondRole.RoleID].RoleID != secondRole.RoleID {
		t.Fatalf("expected second same-name role to remain, got %+v", ownerSession.friends)
	}
}

func TestHandlePacketClassicSocialNameOnlyDoesNotResolveAmbiguousStoredRole(t *testing.T) {
	store := session.NewStore()
	ownerSession, _ := seedSelectedRoleSessionInStore(t, store, "重名查询主人")
	seedSelectedRoleSessionInStore(t, store, "重名查询目标")
	seedSelectedRoleSessionInStore(t, store, "重名查询目标")

	if _, _, ok := store.FindRoleByDisplayName("重名查询目标"); ok {
		t.Fatal("expected duplicate display-name lookup to refuse an arbitrary role")
	}
	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicSocialAddFriendReq,
		Seq: 2,
		Payload: mustJSON(t, classicSocialMutateRequest{
			RoleName: "重名查询目标",
		}),
	}, ownerSession)
	if !result.handled || len(result.friendInfos) != 1 || result.friendInfos[0].RoleID != "social-重名查询目标" || result.friendInfos[0].Online {
		t.Fatalf("expected ambiguous name to remain an offline legacy placeholder, got %+v", result)
	}
}

func TestHandlePacketClassicSocialRoleIDDoesNotFallBackToName(t *testing.T) {
	store := session.NewStore()
	ownerSession, _ := seedSelectedRoleSessionInStore(t, store, "身份优先主人")
	_, targetRole := seedSelectedRoleSessionInStore(t, store, "身份优先目标")
	request := classicSocialMutateRequest{
		RoleID:   "missing-role-id",
		RoleName: targetRole.DisplayName,
	}

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicSocialAddFriendReq,
		Seq:     2,
		Payload: mustJSON(t, request),
	}, ownerSession)
	if !result.handled || len(result.friendInfos) != 1 {
		t.Fatalf("expected fallback friend push, got %+v", result)
	}
	friend := result.friendInfos[0]
	if friend.RoleID != request.RoleID || friend.RoleID == targetRole.RoleID || friend.Online {
		t.Fatalf("expected unresolved roleId to stay authoritative instead of resolving %s by name, got %+v", targetRole.RoleID, friend)
	}
}

func TestHandlePacketClassicSocialGetBlackListReturnsCurrentSnapshot(t *testing.T) {
	_, socketSession := seedSelectedRoleSession(t)

	for index, roleName := range []string{"zeta-black", "alpha-black"} {
		result := handlePacketWithSession(session.NewStore(), protocol.Packet{
			Cmd: cmdClassicSocialAddBlackReq,
			Seq: uint64(index + 2),
			Payload: mustJSON(t, classicSocialMutateRequest{
				RoleName: roleName,
			}),
		}, socketSession)
		if !result.handled || len(result.blackInfos) != 1 {
			t.Fatalf("expected add black-list push for %s, got %+v", roleName, result)
		}
	}

	result := handlePacketWithSession(session.NewStore(), protocol.Packet{
		Cmd: cmdClassicSocialGetBlackListReq,
		Seq: 4,
	}, socketSession)
	if !result.handled || len(result.blackInfos) != 2 {
		t.Fatalf("expected two BlackListInfo pushes from current session, got %+v", result)
	}
	if result.blackInfos[0].RoleName != "alpha-black" || result.blackInfos[1].RoleName != "zeta-black" {
		t.Fatalf("expected stable black-list order, got %+v", result.blackInfos)
	}

	withoutRole := handlePacketWithSession(session.NewStore(), protocol.Packet{
		Cmd: cmdClassicSocialGetBlackListReq,
		Seq: 5,
	}, &packetSession{})
	if !withoutRole.handled || len(withoutRole.blackInfos) != 0 {
		t.Fatalf("expected GetBlackListInfos without selected role to return no entries, got %+v", withoutRole)
	}
}

func TestHandlePacketClassicSocialRequiresSelectedRole(t *testing.T) {
	result := handlePacketWithSession(session.NewStore(), protocol.Packet{
		Cmd: cmdClassicSocialAddFriendReq,
		Seq: 2,
		Payload: mustJSON(t, classicSocialMutateRequest{
			RoleName: "未登录好友",
		}),
	}, &packetSession{})
	if !result.handled {
		t.Fatal("expected AddFriend without selected role to be handled")
	}
	if len(result.friendInfos) != 0 {
		t.Fatalf("expected no FriendInfo without selected role, got %+v", result.friendInfos)
	}
}

func TestHandlePacketClassicSocialTradeRequestReturnsCapturedClosedErrorPush(t *testing.T) {
	_, socketSession := seedSelectedRoleSession(t)

	result := handlePacketWithSession(session.NewStore(), protocol.Packet{
		Cmd: cmdClassicSocialTradeReq,
		Seq: 2,
		Payload: mustJSON(t, classicSocialTradeRequest{
			Handle:   "player_21432",
			RoleName: "桥头的樵夫",
		}),
	}, socketSession)
	if !result.handled || len(result.chatMessages) != 0 || len(result.errorMessages) != 1 {
		t.Fatalf("expected captured trade-closed c_Error push only, got %+v", result)
	}
	tradeError := result.errorMessages[0]
	if tradeError.Msg != classicSocialTradeTemporarilyClosed ||
		!tradeError.Partial ||
		!strings.Contains(tradeError.SourceCapture, "TradeRequest(108)+c_Error(49999)") {
		t.Fatalf("expected source-shaped trade c_Error push, got %+v", tradeError)
	}

	withoutRole := handlePacketWithSession(session.NewStore(), protocol.Packet{
		Cmd: cmdClassicSocialTradeReq,
		Seq: 3,
		Payload: mustJSON(t, classicSocialTradeRequest{
			Handle:   "player_21432",
			RoleName: "桥头的樵夫",
		}),
	}, &packetSession{})
	if !withoutRole.handled || len(withoutRole.chatMessages) != 0 || len(withoutRole.errorMessages) != 0 {
		t.Fatalf("expected trade request without selected role to be handled without warning/error, got %+v", withoutRole)
	}
}

func TestHandlePacketClassicGuildCreateDuplicateNoticeAuthAndPersistence(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ai-server.db")
	store, err := session.NewPersistentStore(dbPath)
	if err != nil {
		t.Fatalf("failed to open persistent store: %v", err)
	}

	leaderSession, leaderRole := seedSelectedRoleSessionInStore(t, store, "公会会长")
	invalidName := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicGuildCreateReq,
		Seq: 2,
		Payload: mustJSON(t, map[string]any{
			"name": "A",
		}),
	}, leaderSession)
	if !invalidName.handled || invalidName.guildResult == nil || invalidName.guildResult.ErrorCode != "INVALID_NAME" {
		t.Fatalf("expected invalid guild name rejection, got %+v", invalidName)
	}

	create := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicGuildCreateReq,
		Seq: 3,
		Payload: mustJSON(t, map[string]any{
			"name":   "快意盟",
			"logoId": "2",
		}),
	}, leaderSession)
	if !create.handled || create.guildInfo == nil || create.guildInfo.Name != "快意盟" {
		t.Fatalf("expected guild info after create, got %+v", create)
	}
	if len(create.guildMembers) != 1 || create.guildMembers[0].RoleID != leaderRole.RoleID {
		t.Fatalf("expected creator member after create, got %+v", create.guildMembers)
	}
	if create.guildAuth == nil || create.guildAuth.PermissionMask == 0 {
		t.Fatalf("expected creator auth after create, got %+v", create.guildAuth)
	}

	memberSession, memberRole := seedSelectedRoleSessionInStore(t, store, "公会成员")
	duplicate := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicGuildCreateReq,
		Seq: 4,
		Payload: mustJSON(t, map[string]any{
			"name": "快意盟",
		}),
	}, memberSession)
	if !duplicate.handled || duplicate.guildResult == nil || duplicate.guildResult.ErrorCode != "DUPLICATE_NAME" {
		t.Fatalf("expected duplicate guild name rejection, got %+v", duplicate)
	}

	addMember := store.Guilds.AddMember(create.guildInfo.ID, memberRole.RoleID, memberRole.DisplayName, memberRole.Level, 0)
	if !addMember.Success {
		t.Fatalf("failed to seed guild member: %+v", addMember)
	}
	noticeWithoutAuth := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicGuildNoticeUpdateReq,
		Seq: 5,
		Payload: mustJSON(t, map[string]any{
			"notice": "普通成员不能改公告",
		}),
	}, memberSession)
	if !noticeWithoutAuth.handled || noticeWithoutAuth.guildResult == nil || noticeWithoutAuth.guildResult.ErrorCode != "NO_AUTH" {
		t.Fatalf("expected notice update without auth rejection, got %+v", noticeWithoutAuth)
	}

	leave := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicGuildLeaveReq,
		Seq:     6,
		Payload: mustJSON(t, map[string]any{}),
	}, memberSession)
	if !leave.handled || leave.guildResult == nil || !leave.guildResult.Success || len(leave.guildMemberClears) != 1 {
		t.Fatalf("expected member leave success, got %+v", leave)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("failed to close store: %v", err)
	}

	reopened, err := session.NewPersistentStore(dbPath)
	if err != nil {
		t.Fatalf("failed to reopen persistent store: %v", err)
	}
	defer reopened.Close()
	reopenedLeaderSession := &packetSession{}
	selectResult := handlePacketWithSession(reopened, protocol.Packet{
		Cmd: cmdRoleSelectRequest,
		Seq: 7,
		Payload: mustJSON(t, session.RoleSelectRequest{
			PlayerID:     "mock-player-001",
			SessionToken: "mock-session-token-001",
			RoleID:       leaderRole.RoleID,
		}),
	}, reopenedLeaderSession)
	if !selectResult.handled || selectResult.townBootstrap == nil {
		t.Fatalf("expected reopened role select success, got %+v", selectResult)
	}
	info := handlePacketWithSession(reopened, protocol.Packet{
		Cmd:     cmdClassicGuildInfoReq,
		Seq:     8,
		Payload: mustJSON(t, map[string]any{}),
	}, reopenedLeaderSession)
	if !info.handled || info.guildInfo == nil || info.guildInfo.Name != "快意盟" || len(info.guildMembers) != 1 {
		t.Fatalf("expected persisted guild after restart, got %+v", info)
	}
}

func TestHandlePacketClassicGuildRequiresSelectedRole(t *testing.T) {
	result := handlePacketWithSession(session.NewStore(), protocol.Packet{
		Cmd:     cmdClassicGuildInfoReq,
		Seq:     2,
		Payload: mustJSON(t, map[string]any{}),
	}, &packetSession{})
	if !result.handled {
		t.Fatal("expected guild info without selected role to be handled")
	}
	if result.guildInfo != nil || len(result.guildMembers) != 0 {
		t.Fatalf("expected no guild pushes without selected role, got %+v", result)
	}
}

func TestHandlePacketClassicMallSearchPurchaseAndErrors(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)

	categories := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicMallCategoryListReq,
		Seq:     2,
		Payload: mustJSON(t, map[string]any{}),
	}, socketSession)
	if !categories.handled || len(categories.mallCategories) == 0 || categories.mallCurrency == nil {
		t.Fatalf("expected mall categories and currency, got %+v", categories)
	}

	count := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicMallSearchCountReq,
		Seq: 3,
		Payload: mustJSON(t, map[string]any{
			"categoryId":      "all",
			"devCurrencyOnly": true,
		}),
	}, socketSession)
	if !count.handled || count.mallSearchCount == nil || count.mallSearchCount.Count < 9 {
		t.Fatalf("expected mall search count, got %+v", count)
	}

	page := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicMallSearchPageReq,
		Seq: 4,
		Payload: mustJSON(t, map[string]any{
			"categoryId":      "all",
			"devCurrencyOnly": true,
			"offset":          0,
			"limit":           99,
		}),
	}, socketSession)
	if !page.handled || page.mallSearchPage == nil || page.mallSearchPage.Limit != 9 || len(page.mallSearchPage.Products) != 9 {
		t.Fatalf("expected 9-limit mall page with hot products, got %+v", page.mallSearchPage)
	}

	purchase := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicMallPurchaseReq,
		Seq: 5,
		Payload: mustJSON(t, map[string]any{
			"productId": "dev-ginseng",
			"quantity":  1,
			"requestId": "mall-req-1",
		}),
	}, socketSession)
	if !purchase.handled || purchase.mallPurchase == nil || !purchase.mallPurchase.Success {
		t.Fatalf("expected mall purchase success, got %+v", purchase)
	}
	items, _, ok := store.GetRoleItems(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, "商城")
	if !ok || len(items) != 1 || items[0].Name != "L百年人参果" {
		t.Fatalf("expected purchased item in 商城 container, got ok=%v items=%+v", ok, items)
	}

	duplicate := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicMallPurchaseReq,
		Seq: 6,
		Payload: mustJSON(t, map[string]any{
			"productId": "dev-ginseng",
			"quantity":  1,
			"requestId": "mall-req-1",
		}),
	}, socketSession)
	if duplicate.mallPurchase == nil || duplicate.mallPurchase.ErrorCode != "DUPLICATE_REQUEST" {
		t.Fatalf("expected duplicate request rejection, got %+v", duplicate)
	}

	insufficient := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicMallPurchaseReq,
		Seq: 7,
		Payload: mustJSON(t, map[string]any{
			"productId": "dev-mount",
			"quantity":  1,
			"requestId": "mall-req-2",
		}),
	}, socketSession)
	if insufficient.mallPurchase == nil || insufficient.mallPurchase.ErrorCode != "INSUFFICIENT_CURRENCY" {
		t.Fatalf("expected insufficient currency, got %+v", insufficient)
	}

	notFound := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicMallPurchaseReq,
		Seq: 8,
		Payload: mustJSON(t, map[string]any{
			"productId": "missing",
			"quantity":  1,
			"requestId": "mall-req-3",
		}),
	}, socketSession)
	if notFound.mallPurchase == nil || notFound.mallPurchase.ErrorCode != "PRODUCT_NOT_FOUND" {
		t.Fatalf("expected product not found, got %+v", notFound)
	}
}

func TestHandlePacketClassicPetInfoAndFeed(t *testing.T) {
	store := session.NewStore()
	socketSession, _ := seedSelectedRoleSessionInStore(t, store, "222")
	water, ok := session.CapturedRoleItemTemplate("宠物用营养水")
	if !ok {
		t.Fatal("expected captured nutrition water template")
	}
	water.Type = "背包"
	water.Index = -1
	water.Count = 2
	granted, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, water)
	if !ok {
		t.Fatal("expected nutrition water to be granted")
	}

	infoResult := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicPetInfoReq,
		Seq:     2,
		Payload: mustJSON(t, classicPetInfoRequest{}),
	}, socketSession)
	if !infoResult.handled || infoResult.petInfo == nil || !infoResult.petInfo.HasPet {
		t.Fatalf("expected pet info push, got %+v", infoResult)
	}
	if infoResult.petInfo.Name != "炎火兽" || infoResult.petInfo.DisplayURL != "petmap/yhs1.swf" {
		t.Fatalf("expected captured fire pet info, got %+v", infoResult.petInfo)
	}

	feedResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicPetFeedReq,
		Seq: 3,
		Payload: mustJSON(t, classicPetFeedRequest{
			Type:  granted.Type,
			Index: granted.Index,
			Count: 1,
		}),
	}, socketSession)
	if !feedResult.handled || feedResult.petFeedResult == nil || !feedResult.petFeedResult.Success || feedResult.petInfo == nil {
		t.Fatalf("expected pet feed result and refreshed info, got %+v", feedResult)
	}
	if feedResult.petInfo.Exp != infoResult.petInfo.Exp+5 || feedResult.petInfo.Fullness != 100 {
		t.Fatalf("expected pet exp/fullness refresh, before=%+v after=%+v", infoResult.petInfo, feedResult.petInfo)
	}
	if len(feedResult.itemInfos) != 1 || feedResult.itemInfos[0].Name != "宠物用营养水" || feedResult.itemInfos[0].Count != 1 {
		t.Fatalf("expected remaining water item push, got infos=%+v clears=%+v", feedResult.itemInfos, feedResult.itemClears)
	}
}

func TestHandlePacketClassicAuctionOpenAndListUsesCapturedPages(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)

	openResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownActiveRoleReq,
		Seq: 1,
		Payload: mustJSON(t, classicTownRoleInteractionRequest{
			Handle: "5010542616817526",
			Kind:   "npc",
			MapID:  "3",
		}),
	}, socketSession)
	if !openResult.handled || openResult.auctionOpen == nil {
		t.Fatalf("expected auction NPC to push open message, got %+v", openResult)
	}
	if openResult.auctionOpen.ContainerType != classicAuctionContainerType || openResult.auctionOpen.NPCHandle != "5010542616817526" {
		t.Fatalf("expected captured auction container open, got %+v", openResult.auctionOpen)
	}

	listResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicAuctionListReq,
		Seq: 2,
		Payload: mustJSON(t, classicAuctionListRequest{
			Page:        0,
			PageCount:   30,
			AuctionType: "道具",
			SortType:    0,
		}),
	}, socketSession)
	if !listResult.handled || listResult.auctionList == nil {
		t.Fatalf("expected auction list push, got %+v", listResult)
	}
	if listResult.auctionList.Total != 201 || len(listResult.auctionList.Items) != 30 || !listResult.auctionList.Partial {
		t.Fatalf("expected captured partial first page total=201 count=30, got %+v", listResult.auctionList)
	}
	first := listResult.auctionList.Items[0]
	if first.Handle != "8040284297490233" || first.Item.Name != "大包还元散" || first.Item.Display != "700.png" {
		t.Fatalf("expected first captured auction item, got %+v", first)
	}
	if first.Item.ItemLevel != 3 || !strings.Contains(first.Item.Description, "大包还元散") || !strings.Contains(first.Item.Description, "&7@2000") {
		t.Fatalf("expected first captured auction item source description, got %+v", first.Item)
	}
	if len(first.PriceItems) != 1 || first.PriceItems[0].Name != "银元宝" || first.PriceItems[0].Count != 700 {
		t.Fatalf("expected first captured auction price, got %+v", first.PriceItems)
	}
	if first.PriceItems[0].ItemLevel != 4 || !strings.Contains(first.PriceItems[0].Description, "银元宝") || !strings.Contains(first.PriceItems[0].Description, "&25@9999") {
		t.Fatalf("expected first captured auction price source description, got %+v", first.PriceItems[0])
	}

	pageTwoResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicAuctionListReq,
		Seq: 3,
		Payload: mustJSON(t, classicAuctionListRequest{
			Page:        2,
			PageCount:   30,
			AuctionType: "道具",
			SortType:    0,
		}),
	}, socketSession)
	if !pageTwoResult.handled || pageTwoResult.auctionList == nil {
		t.Fatalf("expected auction page 2 push, got %+v", pageTwoResult)
	}
	if pageTwoResult.auctionList.Page != 2 || pageTwoResult.auctionList.Total != 201 || len(pageTwoResult.auctionList.Items) != 30 {
		t.Fatalf("expected captured page 2 total=201 count=30, got %+v", pageTwoResult.auctionList)
	}
	pageTwoFirst := pageTwoResult.auctionList.Items[0]
	if pageTwoFirst.Handle != "2804192736274688" || pageTwoFirst.Item.Name != "筋斗云" || pageTwoFirst.Item.Display != "968.png" {
		t.Fatalf("expected captured page 2 first auction item, got %+v", pageTwoFirst)
	}
	if pageTwoFirst.Item.ItemLevel != 3 || !strings.Contains(pageTwoFirst.Item.Description, "筋斗云") || !strings.Contains(pageTwoFirst.Item.Description, "&24@法宝") {
		t.Fatalf("expected captured page 2 first auction item source description, got %+v", pageTwoFirst.Item)
	}
	if len(pageTwoFirst.PriceItems) != 1 || pageTwoFirst.PriceItems[0].Name != "银元宝" || pageTwoFirst.PriceItems[0].Count != 800 {
		t.Fatalf("expected captured page 2 first auction price, got %+v", pageTwoFirst.PriceItems)
	}
	for _, row := range classicAuctionSourceRows {
		itemMeta := classicAuctionSourceItemMeta(row.ItemName, row.Display)
		if itemMeta.Description == "" || itemMeta.ItemLevel == 0 {
			t.Fatalf("expected captured auction item metadata for %+v, got %+v", row, itemMeta)
		}
		priceMeta := classicAuctionSourceItemMeta(row.PriceName, row.PriceDisplay)
		if priceMeta.Description == "" || priceMeta.ItemLevel == 0 {
			t.Fatalf("expected captured auction price metadata for %+v, got %+v", row, priceMeta)
		}
	}
}

func TestHandlePacketClassicMailOpenAndReadUsesCapturedMessages(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)

	openResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownAnswerReq,
		Seq: 1,
		Payload: mustJSON(t, classicTownAnswerRequest{
			Handle:       "5050542617322114",
			MsgHandle:    "1",
			AnswerHandle: "2",
		}),
	}, socketSession)
	if !openResult.handled || openResult.mailOpen == nil || openResult.mailList == nil {
		t.Fatalf("expected mail answer to push open/list, got %+v", openResult)
	}
	if openResult.mailOpen.ContainerType != classicMailContainerType || openResult.mailOpen.NPCHandle != "5050542617322114" {
		t.Fatalf("expected captured mail container open, got %+v", openResult.mailOpen)
	}
	if len(openResult.mailList.Items) != 3 || !openResult.mailList.Partial {
		t.Fatalf("expected captured partial mail list, got %+v", openResult.mailList)
	}
	if len(openResult.mailList.MailCost) != 3 || !strings.Contains(openResult.mailList.MailCost[1], "铜钱[/]x50") {
		t.Fatalf("expected captured mail postage costs, got %+v", openResult.mailList.MailCost)
	}
	if openResult.mailList.Items[0].Handle != "5544758100159914" || openResult.mailList.Items[0].Subject != "新手任务提示" {
		t.Fatalf("expected first captured mail subject, got %+v", openResult.mailList.Items[0])
	}

	infoResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicMailInfoReq,
		Seq: 2,
		Payload: mustJSON(t, classicMailInfoRequest{
			Handle: "5544758100159914",
		}),
	}, socketSession)
	if !infoResult.handled || infoResult.mailInfo == nil || !infoResult.mailInfo.Found {
		t.Fatalf("expected captured mail info, got %+v", infoResult)
	}
	if infoResult.mailInfo.Subject != "新手任务提示" || infoResult.mailInfo.From != "系统" || !strings.Contains(infoResult.mailInfo.Content, "恭喜你升到15级") {
		t.Fatalf("expected captured mail detail, got %+v", infoResult.mailInfo)
	}
}

func TestHandlePacketClassicMailGetAllMovesCapturedAttachmentsToBag(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)

	infoResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicMailInfoReq,
		Seq: 1,
		Payload: mustJSON(t, classicMailInfoRequest{
			Handle: "5544758100159914",
		}),
	}, socketSession)
	if !infoResult.handled || infoResult.mailInfo == nil || !infoResult.mailInfo.Found {
		t.Fatalf("expected captured mail info, got %+v", infoResult)
	}

	listResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownGetItemListReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownContainerRequest{
			Type: classicMailContainerType,
		}),
	}, socketSession)
	if !listResult.handled || listResult.containerCap == nil || listResult.containerCap.Type != classicMailContainerType || listResult.containerCap.Capacity != classicMailCapacity {
		t.Fatalf("expected mail attachment container list, got %+v", listResult)
	}
	if len(listResult.itemInfos) != 2 || listResult.itemInfos[0].Name != "宝匣" || listResult.itemInfos[1].Name != "宠物月饼" {
		t.Fatalf("expected captured mail attachments, got %+v", listResult.itemInfos)
	}

	moveResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownContainerMove,
		Seq: 3,
		Payload: mustJSON(t, classicTownContainerMoveRequest{
			SourceType: classicMailContainerType,
			TargetType: classicTownBagContainerType,
		}),
	}, socketSession)
	if !moveResult.handled {
		t.Fatal("expected mail ContainerMove to be handled")
	}
	if len(moveResult.itemClears) != 2 || moveResult.itemClears[0].Type != classicMailContainerType || moveResult.itemClears[0].Index != 0 || moveResult.itemClears[1].Index != 1 {
		t.Fatalf("expected mail attachment clears for slots 0/1, got %+v", moveResult.itemClears)
	}
	if len(moveResult.itemInfos) != 2 || moveResult.itemInfos[0].Type != classicTownBagContainerType || moveResult.itemInfos[0].Name != "宝匣" || moveResult.itemInfos[1].Name != "宠物月饼" {
		t.Fatalf("expected captured attachments to be pushed into bag, got %+v", moveResult.itemInfos)
	}

	listAgain := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownGetItemListReq,
		Seq: 4,
		Payload: mustJSON(t, classicTownContainerRequest{
			Type: classicMailContainerType,
		}),
	}, socketSession)
	if len(listAgain.itemInfos) != 0 {
		t.Fatalf("expected mail attachment list to be empty after get-all, got %+v", listAgain.itemInfos)
	}
}

func TestHandlePacketClassicMailSendAndAuctionAddUseCapturedVipErrorPush(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)

	mailResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicMailSendReq,
		Seq: 1,
		Payload: mustJSON(t, classicMailSendRequest{
			Subject: "1",
			To:      "桥头的樵夫",
			Items: []classicMailAttachmentRequest{
				{Type: "背包", Index: 38, Count: 1},
			},
		}),
	}, socketSession)
	if !mailResult.handled || len(mailResult.chatMessages) != 0 || len(mailResult.errorMessages) != 1 {
		t.Fatalf("expected SendMail captured VIP c_Error push only, got %+v", mailResult)
	}
	mailError := mailResult.errorMessages[0]
	if mailError.Msg != classicMailSendVipError ||
		!mailError.Partial ||
		!strings.Contains(mailError.SourceCapture, "SendMail+c_Error") {
		t.Fatalf("expected SendMail captured VIP c_Error push, got %+v", mailError)
	}

	auctionResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicAuctionAddReq,
		Seq: 2,
		Payload: mustJSON(t, classicAuctionAddRequest{
			SaleType:  0,
			ItemType:  "背包",
			ItemIndex: 37,
			ItemCount: 3,
			PriceItems: []classicAuctionAddPriceItemRequest{
				{Name: "银元宝", Count: 1},
			},
		}),
	}, socketSession)
	if !auctionResult.handled || len(auctionResult.chatMessages) != 0 || len(auctionResult.errorMessages) != 1 {
		t.Fatalf("expected AddAuctionInfo captured VIP c_Error push only, got %+v", auctionResult)
	}
	auctionError := auctionResult.errorMessages[0]
	if auctionError.Msg != classicAuctionVipError ||
		!auctionError.Partial ||
		!strings.Contains(auctionError.SourceCapture, "AddAuctionInfo+c_Error") {
		t.Fatalf("expected AddAuctionInfo captured VIP c_Error push, got %+v", auctionError)
	}
}

func seedSelectedRoleSession(t *testing.T) (*session.Store, *packetSession) {
	t.Helper()

	store := session.NewStore()
	socketSession, _ := seedSelectedRoleSessionInStore(t, store, "技能测试")
	return store, socketSession
}

func grantRoleItemTemplateForTest(t *testing.T, store *session.Store, socketSession *packetSession, name string, count int) session.RoleItem {
	t.Helper()

	template, ok := session.CapturedRoleItemTemplate(name)
	if !ok {
		t.Fatalf("expected captured item template %s", name)
	}
	template.Count = count
	granted, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, template)
	if !ok {
		t.Fatalf("expected grant item %s", name)
	}
	return granted
}

func fillBagSlotsForTest(t *testing.T, store *session.Store, socketSession *packetSession, startIndex int) {
	t.Helper()

	for index := startIndex; index < 30; index += 1 {
		item := session.RoleItem{
			Type:      "背包",
			Name:      "test-slot-" + strconv.Itoa(index),
			ItemType:  "own",
			Count:     1,
			Index:     index,
			ItemLevel: 1,
		}
		if _, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, item); !ok {
			t.Fatalf("expected filler item for bag slot %d to be granted", index)
		}
	}
}

func startClassicBattleForTest(t *testing.T, socketSession *packetSession) packetResult {
	t.Helper()

	result := handlePacketWithSession(session.NewStore(), protocol.Packet{
		Cmd: cmdClassicBattleStartReq,
		Seq: 2,
		Payload: mustJSON(t, battle.StartRequest{
			MapID:       "4",
			MapName:     "云隐村口",
			StageFocusX: 120,
			ReturnRoute: "town-placeholder",
		}),
	}, socketSession)
	if result.battleStart == nil || result.battleCommand == nil || socketSession.battleRuntime == nil {
		t.Fatalf("expected battle start result, got %+v", result)
	}
	return result
}

func seedSelectedRoleSessionInStore(t *testing.T, store *session.Store, displayName string) (*packetSession, session.RoleSummary) {
	t.Helper()

	login := store.Login(session.LoginRequest{
		UserName: "mockuser",
		Password: "magicpwd",
	})
	create := store.CreateRole(session.RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    displayName,
		Gender:         "female",
		RoleTemplateID: 1,
	})
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
	if !selectResult.handled || selectResult.townBootstrap == nil {
		t.Fatalf("expected role select to seed town bootstrap, got %+v", selectResult)
	}
	return socketSession, create.Role
}

func fastPanelContains(entries []classicTownFastPanelEntry, index int, entryType string, name string) bool {
	for _, entry := range entries {
		if entry.Index == index && entry.Type == entryType && entry.Name == name {
			return true
		}
	}
	return false
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()

	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("failed to marshal json: %v", err)
	}
	return payload
}

func decodeJSON(t *testing.T, payload []byte, target any) {
	t.Helper()

	if err := json.Unmarshal(payload, target); err != nil {
		t.Fatalf("failed to unmarshal json: %v", err)
	}
}

func itemInfosByName(items []classicTownItemInfoPush) map[string]classicTownItemInfoPush {
	result := make(map[string]classicTownItemInfoPush, len(items))
	for _, item := range items {
		result[item.Name] = item
	}
	return result
}

func packetChatMessagesContain(messages []classicTownChatMessagePush, snippet string) bool {
	for _, message := range messages {
		if message.Channel == "system" && strings.Contains(message.Msg, snippet) {
			return true
		}
	}
	return false
}

func questStateForHandle(states []world.QuestStatePush, handle string) (int, bool) {
	for _, state := range states {
		if state.Handle == handle {
			return state.State, true
		}
	}
	return 0, false
}

func assertItemInfo(t *testing.T, item classicTownItemInfoPush, itemType string, classicItemType string, display string, count int) {
	t.Helper()

	if item.Name == "" {
		t.Fatalf("expected item %s/%s display=%s count=%d, got empty item", itemType, classicItemType, display, count)
	}
	if item.Type != itemType || item.ItemType != classicItemType || item.Display != display || item.Count != count {
		t.Fatalf("expected item type=%s itemType=%s display=%s count=%d, got %+v", itemType, classicItemType, display, count, item)
	}
}

func TestDevItemsStateListsRolesAndTemplates(t *testing.T) {
	store := session.NewStore()
	login := store.Login(session.LoginRequest{UserName: "mockuser", Password: "magicpwd"})
	create := store.CreateRole(session.RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "发物品测试",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	state := buildDevItemsState(store)
	if len(state.Roles) != 1 {
		t.Fatalf("expected one role, got %+v", state.Roles)
	}
	if state.Roles[0].PlayerID != login.PlayerID || state.Roles[0].RoleID != create.Role.RoleID || state.Roles[0].DisplayName != "发物品测试" {
		t.Fatalf("expected created role in dev state, got %+v", state.Roles[0])
	}
	if len(state.Templates) == 0 {
		t.Fatal("expected item templates")
	}
	foundMeat := false
	foundWeaponShopItem := false
	foundGroceryShopItem := false
	foundCraftShopItem := false
	foundSkillBook := false
	for _, item := range state.Templates {
		if item.Name == "肉" && item.Display == "70.png" {
			foundMeat = true
		}
		if item.Name == "蛮力钢剑" && item.Display == "40.png" {
			foundWeaponShopItem = true
		}
		if item.Name == "宠物用营养水" && item.Display == "210.png" {
			foundGroceryShopItem = true
		}
		if item.Name == "奥义秘诀" && item.Display == "2.png" {
			foundCraftShopItem = true
		}
		if item.Name == "武器专精" && item.Display == "631.png" {
			foundSkillBook = true
		}
	}
	if !foundMeat {
		t.Fatalf("expected meat template, got %+v", state.Templates)
	}
	if !foundWeaponShopItem || !foundGroceryShopItem || !foundCraftShopItem || !foundSkillBook {
		t.Fatalf("expected captured shop templates weapon=%v grocery=%v craft=%v skill=%v", foundWeaponShopItem, foundGroceryShopItem, foundCraftShopItem, foundSkillBook)
	}
}

func TestDevAddItemHandlerAddsByDisplayID(t *testing.T) {
	store := session.NewStore()
	login := store.Login(session.LoginRequest{UserName: "mockuser", Password: "magicpwd"})
	create := store.CreateRole(session.RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "发肉测试",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	payload := devAddItemRequest{
		PlayerID: login.PlayerID,
		RoleID:   create.Role.RoleID,
		ItemID:   "70",
		Count:    2,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/dev/items/add", bytes.NewReader(body))
	recorder := httptest.NewRecorder()
	handleDevAddItem(recorder, request, store)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	var response devAddItemResponse
	decodeJSON(t, recorder.Body.Bytes(), &response)
	if !response.Success || response.Item.Name != "肉" || response.Item.Count != 2 {
		t.Fatalf("expected meat count 2, got %+v", response)
	}
	items, _, ok := store.GetRoleItems(login.PlayerID, create.Role.RoleID, "背包")
	if !ok {
		t.Fatal("expected role items")
	}
	var meatCount int
	for _, item := range items {
		if item.Name == "肉" {
			meatCount = item.Count
		}
	}
	if meatCount != 2 {
		t.Fatalf("expected persisted meat count 2, got %d", meatCount)
	}
}

func TestDevAddItemHandlerAddsCapturedShopItemByDisplayID(t *testing.T) {
	store := session.NewStore()
	login := store.Login(session.LoginRequest{UserName: "mockuser", Password: "magicpwd"})
	create := store.CreateRole(session.RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "发商店道具测试",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	payload := devAddItemRequest{
		PlayerID: login.PlayerID,
		RoleID:   create.Role.RoleID,
		ItemID:   "210",
		Count:    3,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/dev/items/add", bytes.NewReader(body))
	recorder := httptest.NewRecorder()
	handleDevAddItem(recorder, request, store)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	var response devAddItemResponse
	decodeJSON(t, recorder.Body.Bytes(), &response)
	if !response.Success || response.Item.Name != "宠物用营养水" || response.Item.Display != "210.png" || response.Item.Count != 3 {
		t.Fatalf("expected captured grocery item count 3, got %+v", response)
	}
}

func TestDevSetLevelHandlerSetsRoleLevel(t *testing.T) {
	store := session.NewStore()
	login := store.Login(session.LoginRequest{UserName: "mockuser", Password: "magicpwd"})
	create := store.CreateRole(session.RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "调等级测试",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	str := 30
	payload := devSetLevelRequest{
		PlayerID: login.PlayerID,
		RoleID:   create.Role.RoleID,
		Level:    20,
		STR:      &str,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/dev/items/set-level", bytes.NewReader(body))
	recorder := httptest.NewRecorder()
	handleDevSetLevel(recorder, request, store)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	var response devSetLevelResponse
	decodeJSON(t, recorder.Body.Bytes(), &response)
	if !response.Success || response.Role == nil || response.Role.Level != 20 || response.Role.Exp != session.ClassicRoleLevelToExp(19) {
		t.Fatalf("expected level 20 response, got %+v", response)
	}
	role, playerBase, ok := store.GetRoleRuntimeData(login.PlayerID, create.Role.RoleID)
	if !ok || role.Level != 20 || playerBase.Level != 20 || playerBase.RoleState == nil || playerBase.RoleState.Lv != 20 {
		t.Fatalf("expected persisted level 20 runtime data, ok=%v role=%+v base=%+v", ok, role, playerBase)
	}
	if playerBase.RolePhysique == nil || playerBase.RolePhysique.STR != 30 || playerBase.RolePhysique.PhyAtk != 40 {
		t.Fatalf("expected set-level dev helper to set STR runtime data, base=%+v", playerBase)
	}
}

func TestDevAddCurrencyHandlerAddsSilverCurrency(t *testing.T) {
	store := session.NewStore()
	login := store.Login(session.LoginRequest{UserName: "mockuser", Password: "magicpwd"})
	create := store.CreateRole(session.RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "加元宝测试",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	payload := devAddCurrencyRequest{
		PlayerID: login.PlayerID,
		RoleID:   create.Role.RoleID,
		Amount:   500,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/dev/items/add-currency", bytes.NewReader(body))
	recorder := httptest.NewRecorder()
	handleDevAddCurrency(recorder, request, store)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	var response devAddCurrencyResponse
	decodeJSON(t, recorder.Body.Bytes(), &response)
	if !response.Success || response.Currencies["银元宝"] != 501 {
		t.Fatalf("expected silver currency 501, got %+v", response)
	}
	currencies, ok := store.GetRoleCurrencies(login.PlayerID, create.Role.RoleID)
	if !ok || currencies["银元宝"] != 501 {
		t.Fatalf("expected persisted silver currency 501, got ok=%v currencies=%+v", ok, currencies)
	}
	items, _, ok := store.GetRoleItems(login.PlayerID, create.Role.RoleID, "背包")
	if !ok {
		t.Fatal("expected role bag items")
	}
	var silverItemCount int
	for _, item := range items {
		if item.Name == "银元宝" {
			silverItemCount += item.Count
		}
	}
	if silverItemCount != 501 {
		t.Fatalf("expected silver bag item count 501, got %d items=%+v", silverItemCount, items)
	}
}
