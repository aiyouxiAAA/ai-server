package main

import (
	"ai-server/internal/protocol"
	"ai-server/internal/quest"
	"ai-server/internal/world"
	"testing"
)

func TestAuthoredOnlyNoLegacyQuestEntrances(t *testing.T) {
	store, socket := seedVillageServiceRole(t, 1, 0, 0)
	snapshot := world.BuildTownBootstrap(*socket.selectedRole, *socket.playerBase)
	result := packetResult{townBootstrap: &snapshot}
	removeRetiredQuestContent(&result)
	for _, state := range result.townBootstrap.QuestStates {
		if state.State != 0 {
			t.Fatal("legacy marker remained")
		}
	}
	for _, handle := range []string{"4090542614314425", "4110542614676637", "1000542608713897"} {
		opened := handlePacketWithSession(store, protocol.Packet{Cmd: cmdClassicTownActiveRoleReq, Seq: 1, Payload: mustJSON(t, classicTownRoleInteractionRequest{Handle: handle, MapID: "2", Kind: "npc"})}, socket)
		if opened.answerSpeak != nil {
			for _, answer := range opened.answerSpeak.Answers {
				if quest.IsRetiredAnswer(answer.Handle, answer.Msg) {
					t.Fatal("legacy option remained", answer)
				}
			}
		}
	}
	for _, answer := range []string{"2q23a_1_1", "1q32gs", "day_11gs"} {
		r := handlePacketWithSession(store, protocol.Packet{Cmd: cmdClassicTownAnswerReq, Seq: 1, Payload: mustJSON(t, classicTownAnswerRequest{Handle: "4090542614314425", MsgHandle: "1", AnswerHandle: answer})}, socket)
		if len(r.errorMessages) != 1 || len(r.questInfos) != 0 || len(r.itemInfos) != 0 {
			t.Fatal("retired request mutated state")
		}
	}
	if len(store.AcceptedQuestTitles(socket.playerBase.PlayerID, socket.selectedRole.RoleID)) != 0 {
		t.Fatal("legacy request accepted quest")
	}
}
