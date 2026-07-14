package main

import (
	"testing"

	"ai-server/internal/protocol"
	"ai-server/internal/world"
)

func TestWuliangQuestMenuTracksAcceptedAndRemovedQuestState(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)

	fresh := buildClassicTownQuestAwareAnswerSpeak(store, socketSession, classicWuliangHanXiongHandle)
	requireWuliangMenuAnswer(t, fresh.Answers, "6q2gs")
	requireWuliangMenuAnswer(t, fresh.Answers, "6q40gs")
	requireWuliangMenuAnswer(t, fresh.Answers, "6")
	requireWuliangMenuAnswer(t, fresh.Answers, "2")
	requireWuliangMenuAnswer(t, fresh.Answers, "1")
	requireWuliangMenuAbsent(t, fresh.Answers, "6q3gs")

	if !store.AcceptQuest(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, "侦查敌营") {
		t.Fatal("expected 侦查敌营 to be accepted")
	}
	accepted := buildClassicTownQuestAwareAnswerSpeak(store, socketSession, classicWuliangHanXiongHandle)
	requireWuliangMenuAnswer(t, accepted.Answers, "6q2os")
	requireWuliangMenuAbsent(t, accepted.Answers, "6q2gs")

	if !store.MarkQuestRemoved(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, "侦查敌营") {
		t.Fatal("expected 侦查敌营 to be removed")
	}
	advanced := buildClassicTownQuestAwareAnswerSpeak(store, socketSession, classicWuliangHanXiongHandle)
	requireWuliangMenuAnswer(t, advanced.Answers, "6q3gs")
	requireWuliangMenuAbsent(t, advanced.Answers, "6q2gs")
}

func TestWuliangQuestMenuRetainsNPCFunctionsAndTaskDialogueContracts(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)

	if !store.MarkQuestRemoved(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, "乌梁工匠") {
		t.Fatal("expected 乌梁工匠 to be removed")
	}
	yuMo := buildClassicTownQuestAwareAnswerSpeak(store, socketSession, classicWuliangYuMoHandle)
	requireWuliangMenuAnswer(t, yuMo.Answers, "6q10gs")
	requireWuliangMenuAnswer(t, yuMo.Answers, "2")
	requireWuliangMenuAnswer(t, yuMo.Answers, "3")
	activeResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownActiveRoleReq,
		Seq: 1,
		Payload: mustJSON(t, classicTownRoleInteractionRequest{
			Handle: classicWuliangYuMoHandle,
			Kind:   "npc",
			MapID:  "191",
		}),
	}, socketSession)
	if !activeResult.handled || activeResult.answerSpeak == nil {
		t.Fatalf("expected ActiveRole to open Yumo dialogue, got %+v", activeResult)
	}
	requireWuliangMenuAnswer(t, activeResult.answerSpeak.Answers, "6q10gs")

	if dialogue := world.BuildAnswerReply(classicWuliangYuMoHandle, "1", "6q10gs"); dialogue == nil || dialogue.MsgHandle != "6q10d_1" {
		t.Fatalf("expected captured 造坝截水 dialogue, got %+v", dialogue)
	}
	if route, ok := findClassicQuestAnswerRoute(classicTownAnswerRequest{
		Handle:       classicWuliangYuMoHandle,
		MsgHandle:    "6q10d_1",
		AnswerHandle: "6q10a_1_1",
	}); !ok || route.title != "造坝截水" {
		t.Fatalf("expected 造坝截水 accept route, got %+v ok=%t", route, ok)
	}
	if route, ok := findClassicQuestAnswerRoute(classicTownAnswerRequest{
		Handle:       classicWuliangYuMoHandle,
		MsgHandle:    "6q16d_2",
		AnswerHandle: "6q16a_2_1",
	}); !ok || route.title != "建造水车" {
		t.Fatalf("expected 建造水车 completion route, got %+v ok=%t", route, ok)
	}
	if route, ok := findClassicQuestAnswerRoute(classicTownAnswerRequest{
		Handle:       classicWuliangYuMoHandle,
		MsgHandle:    "6q17d_2",
		AnswerHandle: "6q17a_2_1",
	}); !ok || route.title != "精巧零件" {
		t.Fatalf("expected 精巧零件 completion route, got %+v ok=%t", route, ok)
	}
}

func TestWuliangQuestMenuUsesCapturedActiveDialogue(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	if !store.AcceptQuest(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, "机木玄文") {
		t.Fatal("expected 机木玄文 to be accepted")
	}

	answerSpeak := buildClassicTownQuestAwareAnswerSpeak(store, socketSession, classicWuliangYuMoHandle)
	requireWuliangMenuAnswer(t, answerSpeak.Answers, "6q25ns")
	requireWuliangMenuAbsent(t, answerSpeak.Answers, "6q25os")
	if dialogue := world.BuildAnswerReply(classicWuliangYuMoHandle, "1", "6q25ns"); dialogue == nil || dialogue.MsgHandle != "6q25d_3" {
		t.Fatalf("expected captured 机木玄文 active dialogue, got %+v", dialogue)
	}
}

func TestWuliangQuestMenuNeverEmitsUnresolvedDialogueOption(t *testing.T) {
	for _, entry := range classicWuliangQuestMenuEntries {
		for _, option := range []*world.AnswerOption{entry.start, entry.active, entry.complete} {
			if option == nil {
				continue
			}
			if !classicWuliangQuestMenuOptionSupported(entry.handle, option) {
				t.Fatalf("expected task menu option handle=%s title=%s answer=%s to resolve", entry.handle, entry.title, option.Handle)
			}
		}
	}
}

func requireWuliangMenuAnswer(t *testing.T, answers []world.AnswerOption, handle string) {
	t.Helper()
	for _, answer := range answers {
		if answer.Handle == handle {
			return
		}
	}
	t.Fatalf("expected menu answer %s, got %+v", handle, answers)
}

func requireWuliangMenuAbsent(t *testing.T, answers []world.AnswerOption, handle string) {
	t.Helper()
	for _, answer := range answers {
		if answer.Handle == handle {
			t.Fatalf("expected menu answer %s to be absent, got %+v", handle, answers)
		}
	}
}
