package main

import (
	"ai-server/internal/protocol"
	"ai-server/internal/quest"
	"ai-server/internal/world"
	"strings"
	"testing"
)

func TestAuthoredTutorialChainPackets(t *testing.T) {
	store, socket := seedVillageServiceRole(t, 1, 0, 0)
	checkMarkers := func(states []world.QuestStatePush, expected map[string]int) {
		t.Helper()
		actual := map[string]int{}
		for _, state := range states {
			actual[state.Handle] = state.State
		}
		for handle, want := range expected {
			if got, ok := actual[handle]; !ok || got != want {
				t.Fatalf("marker %s = %d (present=%v), want %d; all=%+v", handle, got, ok, want, states)
			}
		}
	}
	send := func(id, handle, answer string) packetResult {
		return handlePacketWithSession(store, protocol.Packet{Cmd: cmdClassicTownAnswerReq, Seq: 1, Payload: mustJSON(t, classicTownAnswerRequest{Handle: handle, MsgHandle: id, AnswerHandle: answer})}, socket)
	}
	step := func(id string, complete bool) packetResult {
		t.Helper()
		q, _ := quest.FindAuthored(id)
		handle, answer := q.Start.Handle(), "accept"
		if complete {
			handle, answer = q.Finish.Handle(), "complete"
		}
		r := send(id, handle, answer)
		if len(r.errorMessages) != 0 || complete && len(r.questClears) != 1 || !complete && len(r.questInfos) != 1 {
			t.Fatalf("step %s %v: %+v", id, complete, r)
		}
		return r
	}
	first := step("XZ-M001", false)
	checkMarkers(first.questStates, map[string]int{"authored-lu-xiaoman": 0, "authored-guan-boheng": 1})
	step("XZ-M001", true)
	for _, q := range quest.Authored()[1:] {
		// Inspect each plain offer without mutating another quest's progress.
		plain := q
		plain.Rule.Previous = ""
		offer := authoredQuestDialogue(store, socket, plain, plain.Start.Handle())
		if len(offer.DialogueLines) != 1 || len(offer.Answers) != 1 || offer.Answers[0].Handle != "accept" || strings.Count(offer.DialogueLines[0].Text, q.Info.Description) != 1 {
			t.Fatalf("plain quest must offer directly after one non-duplicated description: %+v", offer)
		}
	}
	gift := step("XZ-M002", false)
	checkMarkers(gift.questStates, map[string]int{"authored-shi-li": 4, "authored-lu-xiaoman": 0, "authored-guan-boheng": 0})
	if len(gift.itemInfos) != 1 || gift.itemInfos[0].Name != "雪栈铁刀" || gift.questGuide.Entries[0].Action != "open_page" {
		t.Fatalf("gift %+v", gift)
	}
	blocked := send("XZ-M002", "authored-shi-li", "complete")
	if len(blocked.questClears) != 0 {
		t.Fatal("unworn sword completed")
	}
	menu, ok := buildAuthoredQuestOpenResult(store, socket, "authored-shi-li")
	if !ok || menu.answerSpeak == nil {
		t.Fatal("missing smith menu")
	}
	shop := send(menu.answerSpeak.MsgHandle, "authored-shi-li", "service:weapon")
	if shop.skillShop == nil {
		t.Fatal("quest hid weapon shop")
	}
	equip := handlePacketWithSession(store, protocol.Packet{Cmd: cmdClassicTownEquipItemReq, Seq: 1, Payload: mustJSON(t, classicTownEquipItemRequest{Type: "背包", Index: gift.itemInfos[0].Index, Count: 1})}, socket)
	if equip.questGuide == nil || equip.questGuide.Entries[0].TargetHandle != "authored-shi-li" {
		t.Fatalf("equip guide %+v", equip.questGuide)
	}
	checkMarkers(equip.questStates, map[string]int{"authored-shi-li": 1})
	step("XZ-M002", true)
	lesson := step("XZ-M003", false)
	checkMarkers(lesson.questStates, map[string]int{"authored-shi-li": 0, "authored-ye-zhidong": 1})
	medical, _ := buildAuthoredQuestOpenResult(store, socket, "authored-ye-zhidong")
	if medical.answerSpeak == nil || len(medical.answerSpeak.Answers) != 3 {
		t.Fatal("healing lesson must retain heal/shop options")
	}
	step("XZ-M003", true)
	battle := step("XZ-M004", false)
	checkMarkers(battle.questStates, map[string]int{"authored-huo-changying": 4, "authored-ye-zhidong": 0})
	if len(send("XZ-M004", "authored-huo-changying", "complete").questClears) != 0 {
		t.Fatal("unwon battle completed")
	}
	socket.playerBase.MapID = 8
	if len(advanceClassicQuestProgressForTargets(store, socket, quest.ObjectiveKindKill, []string{"雪蛛"})) != 0 {
		t.Fatal("wrong field counted")
	}
	socket.playerBase.MapID = 6
	if len(advanceClassicQuestProgressForTargets(store, socket, quest.ObjectiveKindKill, []string{"其他怪物"})) != 0 {
		t.Fatal("wrong monster counted")
	}
	if len(advanceClassicQuestProgressForTargets(store, socket, quest.ObjectiveKindKill, []string{"雪蛛"})) != 1 {
		t.Fatal("snow path kill not counted")
	}
	if len(send("XZ-M004", "authored-huo-changying", "complete").questClears) != 0 {
		t.Fatal("field turn-in allowed")
	}
	socket.playerBase.MapID = 2
	step("XZ-M004", true)
	step("XZ-M005", false)
	// Full state needs no fabricated injury or redundant healer transaction.
	full := store.HealRoleAtTown(socket.playerBase.PlayerID, socket.selectedRole.RoleID)
	socket.selectedRole, socket.playerBase = &full.Role, &full.PlayerBase
	if !store.AuthoredQuestReady(socket.playerBase.PlayerID, socket.selectedRole.RoleID, "XZ-M005") {
		t.Fatal("full role blocked")
	}
	state := *socket.playerBase.RoleState
	state.HP -= 11
	role, base, _ := store.UpdateRoleState(socket.playerBase.PlayerID, socket.selectedRole.RoleID, state)
	socket.selectedRole, socket.playerBase = &role, &base
	checkMarkers(allAuthoredQuestStates(store, socket), map[string]int{"authored-huo-changying": 0, "authored-ye-zhidong": 4})
	if len(send("XZ-M005", "authored-ye-zhidong", "complete").questClears) != 0 {
		t.Fatal("injured turn-in allowed")
	}
	healed := send("XZ-M005", "authored-ye-zhidong", "service:heal")
	if healed.roleState == nil || healed.questGuide == nil || healed.questGuide.Entries[0].ObjectiveID != "confirm-health" {
		t.Fatal("healing didn't refresh guide")
	}
	checkMarkers(healed.questStates, map[string]int{"authored-huo-changying": 0, "authored-ye-zhidong": 1})
	step("XZ-M005", true)
	returning := step("XZ-M006", false)
	checkMarkers(returning.questStates, map[string]int{"authored-ye-zhidong": 0, "authored-guan-boheng": 1})
	final := step("XZ-M006", true)
	if len(final.questGuide.Entries) != 0 || len(final.questGuide.CompletedQuestIDs) != 6 {
		t.Fatal("chain end")
	}
	for _, q := range quest.Authored() {
		if len(send(q.Info.ID, q.Start.Handle(), "accept").questInfos) != 0 {
			t.Fatal("reaccepted", q.Info.ID)
		}
	}
	snapshot := world.TownBootstrapSnapshot{}
	snapshot.LoadMap.MapID = "2"
	applyAcceptedQuestStatesToBootstrap(&snapshot, store, socket)
	seen := map[string]bool{}
	for _, npc := range snapshot.CreateRoles {
		if seen[npc.Handle] {
			t.Fatal("duplicate NPC", npc.Handle)
		}
		seen[npc.Handle] = true
	}
	for _, state := range snapshot.QuestStates {
		if state.State != 0 {
			t.Fatal("completed marker remained", state)
		}
	}
}
