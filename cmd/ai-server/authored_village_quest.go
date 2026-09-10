package main

import (
	"ai-server/internal/quest"
	"ai-server/internal/session"
	"ai-server/internal/world"
	"strconv"
)

func authoredQuestRoleReady(socket *packetSession, q quest.AuthoredQuest) bool {
	return socket != nil && socket.selectedRole != nil && socket.playerBase != nil && socket.playerBase.MapID == q.MapID
}

func authoredQuestStatus(store *session.Store, socket *packetSession, q quest.AuthoredQuest) string {
	if store == nil || !authoredQuestRoleReady(socket, q) {
		return "locked"
	}
	id, role := socket.playerBase.PlayerID, socket.selectedRole.RoleID
	if store.RemovedQuestTitles(id, role)[q.Info.Title] {
		return "completed"
	}
	if store.AcceptedQuestTitles(id, role)[q.Info.Title] {
		return "accepted"
	}
	if socket.playerBase.Level < q.Info.Level {
		return "locked"
	}
	return "available"
}

func authoredQuestStates(store *session.Store, socket *packetSession, q quest.AuthoredQuest) []world.QuestStatePush {
	start, finish := 0, 0
	switch authoredQuestStatus(store, socket, q) {
	case "available":
		start = 2
	case "accepted":
		start = 4
		finish = 1
	}
	return []world.QuestStatePush{{Handle: q.Start.Handle(), State: start}, {Handle: q.Finish.Handle(), State: finish}}
}

// Reuse createRole/QuestState for authored identities while keeping captured
// world definitions and their strict source catalog unchanged.
func applyAuthoredVillageQuestBootstrap(snapshot *world.TownBootstrapSnapshot, store *session.Store, socket *packetSession) {
	if snapshot == nil {
		return
	}
	for _, q := range quest.Authored() {
		if snapshot.LoadMap.MapID != strconv.Itoa(q.MapID) {
			continue
		}
		for _, npc := range []quest.AuthoredNPC{q.Start, q.Finish} {
			snapshot.CreateRoles = append(snapshot.CreateRoles, world.RolePush{
				Handle: npc.Handle(), RoleID: "-1", DisplayName: npc.Name, Level: 1,
				MapID: strconv.Itoa(q.MapID), SourceQuery: "authored/" + npc.Key, Kind: "npc",
				SpawnFlash: world.SpawnPoint{X: npc.X, Y: npc.Y},
			})
		}
		snapshot.QuestStates = append(snapshot.QuestStates, authoredQuestStates(store, socket, q)...)
	}
}

func authoredQuestDialogue(store *session.Store, socket *packetSession, q quest.AuthoredQuest, handle string) world.AnswerSpeakPush {
	msg := world.AnswerSpeakPush{Handle: handle, MsgHandle: q.Info.ID, Answers: []world.AnswerOption{}}
	status := authoredQuestStatus(store, socket, q)
	phase := ""
	switch {
	case status == "completed":
		msg.Msg = q.CompletedDialogue
	case status == "available" && handle == q.Start.Handle():
		phase = "offer"
		msg.Msg = q.OfferDialogue + "<br/>" + q.Info.Description
		msg.Answers = append(msg.Answers, world.AnswerOption{Handle: "accept", Msg: "<ml><m/>领取：" + q.Info.Title})
	case status == "accepted" && handle == q.Finish.Handle():
		phase = "finish"
		msg.Msg = q.FinishDialogue
		msg.Answers = append(msg.Answers, world.AnswerOption{Handle: "complete", Msg: "<ml><m/>完成：" + q.Info.Title})
	case status == "accepted":
		msg.Msg = q.ReminderDialogue
	default:
		msg.Msg = "先去和陆小满打个招呼吧。"
	}
	if phase != "" {
		for _, line := range quest.AuthoredDialogue(q.Info.ID, phase) {
			msg.DialogueLines = append(msg.DialogueLines, world.DialogueLine{Speaker: line.Speaker, Text: line.Text})
		}
		msg.DialogueLines = append(msg.DialogueLines, world.DialogueLine{Speaker: "npc", Text: "【" + q.Info.Title + "】<br/>" + q.Info.Description})
	}
	return msg
}

func buildAuthoredQuestOpenResult(store *session.Store, socket *packetSession, handle string) (packetResult, bool) {
	q, ok := quest.AuthoredForNPC(handle)
	if !ok {
		return packetResult{}, false
	}
	if !authoredQuestRoleReady(socket, q) {
		return packetResult{handled: true}, true
	}
	dialogue := authoredQuestDialogue(store, socket, q, handle)
	return packetResult{answerSpeak: &dialogue, questStates: authoredQuestStates(store, socket, q), handled: true}, true
}

func buildAuthoredQuestAnswerResult(store *session.Store, socket *packetSession, request classicTownAnswerRequest) (packetResult, bool) {
	q, ok := quest.AuthoredForNPC(request.Handle)
	if !ok {
		return packetResult{}, false
	}
	result := packetResult{handled: true}
	if store == nil || !authoredQuestRoleReady(socket, q) || request.MsgHandle != q.Info.ID {
		return result, true
	}
	status := authoredQuestStatus(store, socket, q)
	switch {
	case request.AnswerHandle == "accept" && request.Handle == q.Start.Handle() && status == "available":
		if len(store.AcceptedQuestTitles(socket.playerBase.PlayerID, socket.selectedRole.RoleID)) >= classicQuestAcceptedLimit {
			result.errorMessages = []classicTownErrorPush{{Msg: classicQuestFullError}}
			return result, true
		}
		if !store.AcceptQuest(socket.playerBase.PlayerID, socket.selectedRole.RoleID, q.Info.Title) {
			return result, true
		}
		result.questInfos = []classicQuestInfoPush{classicQuestInfoForRole(store, socket, q.Info)}
		result.chatMessages = []classicTownChatMessagePush{classicTownSystemChatMessage("接受了任务【" + q.Info.Title + "】。")}
	case request.AnswerHandle == "complete" && request.Handle == q.Finish.Handle() && status == "accepted":
		result = buildClassicQuestCompleteResult(store, socket, q.Info, q.Info.Title)
	}
	result.questStates = authoredQuestStates(store, socket, q)
	result.questGuide = buildQuestGuideSnapshot(store, socket)
	// The client closes its answer page on submit. Successful transactions only
	// push state; a fresh AnswerSpeak would reopen an unsolicited reminder page.
	if len(result.questInfos) == 0 && len(result.questClears) == 0 {
		dialogue := authoredQuestDialogue(store, socket, q, request.Handle)
		result.answerSpeak = &dialogue
	}
	return result, true
}
