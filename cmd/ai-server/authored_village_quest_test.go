package main

import (
	"ai-server/internal/protocol"
	"ai-server/internal/quest"
	"ai-server/internal/world"
	"testing"
)

func TestAuthoredVillageQuestLifecycle(t *testing.T) {
	store, socket := seedSelectedRoleSession(t)
	socket.playerBase.MapID = 2
	q, ok := quest.FindAuthored("XZ-M001")
	if !ok {
		t.Fatal("missing authored table row")
	}
	send := func(handle, answer string) packetResult {
		return handlePacketWithSession(store, protocol.Packet{Cmd: cmdClassicTownAnswerReq, Seq: 10, Payload: mustJSON(t, classicTownAnswerRequest{Handle: handle, MsgHandle: q.Info.ID, AnswerHandle: answer})}, socket)
	}
	check := func(start, finish int) {
		t.Helper()
		states := authoredQuestStates(store, socket, q)
		if states[0].State != start || states[1].State != finish {
			t.Fatalf("states %+v", states)
		}
	}
	check(2, 0)
	offer := authoredQuestDialogue(store, socket, q, q.Start.Handle())
	if len(offer.DialogueLines) != 4 || offer.DialogueLines[1].Speaker != "player" || len(offer.Answers) != 1 || offer.Answers[0].Handle != "accept" {
		t.Fatalf("authored offer presentation %+v", offer)
	}
	check(2, 0) // Opening and reading must not accept the task.
	wrong := send(q.Finish.Handle(), "accept")
	if len(wrong.questInfos) != 0 {
		t.Fatal("wrong NPC accepted")
	}
	premature := send(q.Finish.Handle(), "complete")
	if len(premature.questClears) != 0 {
		t.Fatal("completed before accept")
	}
	accepted := send(q.Start.Handle(), "accept")
	if len(accepted.questInfos) != 1 || accepted.questInfos[0].QuestID != q.Info.ID {
		t.Fatalf("accept %+v", accepted)
	}
	check(4, 1)
	report := authoredQuestDialogue(store, socket, q, q.Finish.Handle())
	if len(report.DialogueLines) != 3 || report.DialogueLines[1].Speaker != "player" || report.Answers[0].Handle != "complete" {
		t.Fatalf("authored report presentation %+v", report)
	}
	if accepted.answerSpeak != nil {
		t.Fatal("accept must not reopen dialogue")
	}
	again := send(q.Start.Handle(), "accept")
	if len(again.questInfos) != 0 || len(again.questClears) != 0 {
		t.Fatal("duplicate accept mutated quest")
	}
	wrong = send(q.Start.Handle(), "complete")
	if len(wrong.questClears) != 0 {
		t.Fatal("wrong NPC completed")
	}
	for _, complete := range []bool{false, true} {
		r := buildClassicQuestRemoveResult(store, socket, classicQuestRemoveRequest{QuestID: q.Info.ID, Complete: complete})
		if len(r.questClears) != 0 {
			t.Fatal("direct removal allowed")
		}
	}
	log := buildClassicQuestLogResult(store, socket)
	found := false
	for _, info := range log.questInfos {
		if info.QuestID == q.Info.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("missing task log")
	}
	snapshot := world.TownBootstrapSnapshot{}
	snapshot.LoadMap.MapID = "2"
	applyAuthoredVillageQuestBootstrap(&snapshot, store, socket)
	if len(snapshot.CreateRoles) != 2 || snapshot.QuestStates[0].State != 4 || snapshot.QuestStates[1].State != 1 {
		t.Fatalf("accepted restore %+v", snapshot)
	}
	completed := send(q.Finish.Handle(), "complete")
	if len(completed.questClears) != 1 || completed.roleState == nil || completed.roleState.Exp != 100 {
		t.Fatalf("completion reward %+v", completed)
	}
	check(0, 0)
	if completed.answerSpeak != nil {
		t.Fatal("complete must not reopen dialogue")
	}
	twice := send(q.Finish.Handle(), "complete")
	if twice.roleState != nil || len(twice.questClears) != 0 {
		t.Fatal("duplicate reward")
	}
	again = send(q.Start.Handle(), "accept")
	if len(again.questInfos) != 0 {
		t.Fatal("completed quest reaccepted")
	}
	snapshot = world.TownBootstrapSnapshot{}
	snapshot.LoadMap.MapID = "2"
	applyAuthoredVillageQuestBootstrap(&snapshot, store, socket)
	if snapshot.QuestStates[0].State != 0 || snapshot.QuestStates[1].State != 0 {
		t.Fatal("completion did not persist into bootstrap")
	}
	other, _ := seedSelectedRoleSessionInStore(t, store, "独立进度")
	other.playerBase.MapID = 2
	if authoredQuestStatus(store, other, q) != "available" {
		t.Fatal("other role inherited progress")
	}
}

func TestAuthoredVillageQuestGuards(t *testing.T) {
	store, socket := seedSelectedRoleSession(t)
	q, _ := quest.FindAuthored("XZ-M001")
	request := classicTownAnswerRequest{Handle: q.Start.Handle(), MsgHandle: q.Info.ID, AnswerHandle: "accept"}
	socket.playerBase.MapID = 3
	buildAuthoredQuestAnswerResult(store, socket, request)
	socket.playerBase.MapID = 2
	socket.playerBase.Level = 0
	buildAuthoredQuestAnswerResult(store, socket, request)
	socket.playerBase.Level = 1
	bad := request
	bad.MsgHandle = "wrong"
	buildAuthoredQuestAnswerResult(store, socket, bad)
	if authoredQuestStatus(store, socket, q) != "available" {
		t.Fatal("invalid request changed progress")
	}
	for _, title := range []string{"a", "b", "c", "d", "e"} {
		store.AcceptQuest(socket.playerBase.PlayerID, socket.selectedRole.RoleID, title)
	}
	result, _ := buildAuthoredQuestAnswerResult(store, socket, request)
	if len(result.errorMessages) != 1 || authoredQuestStatus(store, socket, q) != "available" {
		t.Fatal("capacity guard")
	}
}
