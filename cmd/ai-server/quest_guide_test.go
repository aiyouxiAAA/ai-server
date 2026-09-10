package main

import (
	"ai-server/internal/protocol"
	"ai-server/internal/quest"
	"testing"
)

func TestQuestGuideLifecycleAndMapScope(t *testing.T) {
	store, socket := seedSelectedRoleSession(t)
	socket.playerBase.MapID = 2
	q, _ := quest.FindAuthored("XZ-M001")
	open := handlePacketWithSession(store, protocol.Packet{Cmd: cmdClassicTownActiveRoleReq, Seq: 1, Payload: mustJSON(t, classicTownRoleInteractionRequest{Handle: q.Start.Handle(), RoleID: "-1", Kind: "npc", MapID: "2", NavigationToken: "test-navigation"})}, socket)
	if open.answerSpeak == nil || open.answerSpeak.NavigationToken != "test-navigation" {
		t.Fatal("navigation response must echo the operation token")
	}
	snapshot := buildQuestGuideSnapshot(store, socket)
	if snapshot == nil || len(snapshot.Entries) != 1 || snapshot.Entries[0].Phase != "available" || snapshot.Entries[0].TargetHandle != q.Start.Handle() {
		t.Fatalf("offer %+v", snapshot)
	}
	if len(store.AcceptedQuestTitles(socket.playerBase.PlayerID, socket.selectedRole.RoleID)) != 0 {
		t.Fatal("guide must not auto accept")
	}
	if snapshot.Entries[0].Type != q.Info.Type || snapshot.Entries[0].Type != "main" {
		t.Fatal("available guidance must expose the catalog quest type before accepting")
	}
	socket.playerBase.MapID = 6
	if len(buildQuestGuideSnapshot(store, socket).Entries) != 0 {
		t.Fatal("offer must belong to village")
	}
	socket.playerBase.MapID = 2
	accepted, _ := buildAuthoredQuestAnswerResult(store, socket, classicTownAnswerRequest{Handle: q.Start.Handle(), MsgHandle: q.Info.ID, AnswerHandle: "accept"})
	if accepted.questGuide == nil || len(accepted.questGuide.Entries) != 1 || accepted.questGuide.Entries[0].TargetHandle != q.Finish.Handle() {
		t.Fatalf("accept %+v", accepted.questGuide)
	}
	socket.playerBase.MapID = 6
	if len(buildQuestGuideSnapshot(store, socket).Entries) != 1 {
		t.Fatal("accepted objective must survive map change")
	}
	socket.playerBase.MapID = 2
	completed, _ := buildAuthoredQuestAnswerResult(store, socket, classicTownAnswerRequest{Handle: q.Finish.Handle(), MsgHandle: q.Info.ID, AnswerHandle: "complete"})
	if completed.questGuide == nil || len(completed.questGuide.Entries) != 0 || len(completed.questGuide.CompletedQuestIDs) != 1 {
		t.Fatalf("completion %+v", completed.questGuide)
	}
	log := buildClassicQuestLogResult(store, socket)
	if log.questGuide == nil || len(log.questGuide.Entries) != 0 || len(log.questGuide.CompletedQuestIDs) != 1 {
		t.Fatalf("restore %+v", log.questGuide)
	}
	if buildQuestGuideSnapshot(nil, socket) != nil {
		t.Fatal("missing store cannot create false available state")
	}
}
