package main

import (
	"ai-server/internal/quest"
	"ai-server/internal/session"
	"ai-server/internal/world"
	"strconv"
	"strings"
)

func authoredQuestRoleReady(socket *packetSession, q quest.AuthoredQuest) bool {
	return socket != nil && socket.selectedRole != nil && socket.playerBase != nil && socket.playerBase.MapID == q.MapID && socket.battleRuntime == nil
}

func authoredQuestStatus(store *session.Store, socket *packetSession, q quest.AuthoredQuest) string {
	if store == nil || socket == nil || socket.selectedRole == nil || socket.playerBase == nil {
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
	if q.Rule.Previous != "" {
		previous, _ := quest.FindAuthored(q.Rule.Previous)
		if !store.RemovedQuestTitles(id, role)[previous.Info.Title] {
			return "locked"
		}
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
		finish = 4
		if store.AuthoredQuestReady(socket.playerBase.PlayerID, socket.selectedRole.RoleID, q.Info.ID) {
			finish = 1
		}
	}
	if q.Start.Handle() == q.Finish.Handle() {
		return []world.QuestStatePush{{Handle: q.Start.Handle(), State: finishOrStart(start, finish)}}
	}
	return []world.QuestStatePush{{Handle: q.Start.Handle(), State: start}, {Handle: q.Finish.Handle(), State: finish}}
}

// Reuse createRole/QuestState for authored identities while keeping captured
// world definitions and their strict source catalog unchanged.
func applyAuthoredVillageQuestBootstrap(snapshot *world.TownBootstrapSnapshot, store *session.Store, socket *packetSession) {
	if snapshot == nil {
		return
	}
	seen := map[string]bool{}
	for _, role := range snapshot.CreateRoles {
		seen[role.Handle] = true
	}
	for _, q := range quest.Authored() {
		if snapshot.LoadMap.MapID != strconv.Itoa(q.MapID) {
			continue
		}
		for _, npc := range []quest.AuthoredNPC{q.Start, q.Finish} {
			if seen[npc.Handle()] {
				continue
			}
			seen[npc.Handle()] = true
			snapshot.CreateRoles = append(snapshot.CreateRoles, world.RolePush{
				Handle: npc.Handle(), RoleID: "-1", DisplayName: npc.Name, Level: 1,
				MapID: strconv.Itoa(q.MapID), SourceQuery: "authored/" + npc.Key, Kind: "npc",
				SpawnFlash: world.SpawnPoint{X: npc.X, Y: npc.Y},
			})
		}
	}
	snapshot.QuestStates = append(snapshot.QuestStates, allAuthoredQuestStates(store, socket)...)
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
	case status == "accepted" && handle == q.Finish.Handle() && store.AuthoredQuestReady(socket.playerBase.PlayerID, socket.selectedRole.RoleID, q.Info.ID):
		phase = "finish"
		msg.Msg = q.FinishDialogue
		msg.Answers = append(msg.Answers, world.AnswerOption{Handle: "complete", Msg: "<ml><m/>完成：" + q.Info.Title})
	case status == "accepted":
		msg.Msg = q.ReminderDialogue
	default:
		msg.Msg = "先完成当前的委托，再来找我吧。"
	}
	if phase != "" {
		for _, line := range quest.AuthoredDialogue(q.Info.ID, phase) {
			msg.DialogueLines = append(msg.DialogueLines, world.DialogueLine{Speaker: line.Speaker, Text: line.Text})
		}
		if len(msg.DialogueLines) == 0 {
			msg.DialogueLines = append(msg.DialogueLines, world.DialogueLine{Speaker: "npc", Text: msg.Msg})
		}
		msg.DialogueLines = append(msg.DialogueLines, world.DialogueLine{Speaker: "npc", Text: "【" + q.Info.Title + "】<br/>" + q.Info.Description})
	}
	return msg
}

func buildAuthoredQuestOpenResult(store *session.Store, socket *packetSession, handle string) (packetResult, bool) {
	return buildAuthoredNPCMenu(store, socket, handle)
}

func buildAuthoredQuestAnswerResult(store *session.Store, socket *packetSession, request classicTownAnswerRequest) (packetResult, bool) {
	if request.MsgHandle == authoredQuestMenuID {
		return selectAuthoredNPCQuest(store, socket, request)
	}
	q, ok := quest.FindAuthored(request.MsgHandle)
	if !ok {
		return packetResult{}, false
	}
	result := packetResult{handled: true}
	if store == nil || !authoredQuestRoleReady(socket, q) || (request.Handle != q.Start.Handle() && request.Handle != q.Finish.Handle()) {
		result.errorMessages = []classicTownErrorPush{{Msg: "请在任务所在村庄与指定人物交谈。"}}
		return result, true
	}
	if strings.HasPrefix(request.AnswerHandle, "service:") {
		request.MsgHandle = authoredServiceMessageID
		request.AnswerHandle = strings.TrimPrefix(request.AnswerHandle, "service:")
		return buildAuthoredServiceAnswerResult(store, socket, request)
	}
	status := authoredQuestStatus(store, socket, q)
	valid := request.AnswerHandle == "accept" && request.Handle == q.Start.Handle() && status == "available" || request.AnswerHandle == "complete" && request.Handle == q.Finish.Handle() && status == "accepted"
	if !valid {
		dialogue := authoredQuestDialogue(store, socket, q, request.Handle)
		appendAuthoredServiceOptions(socket, &dialogue)
		result.answerSpeak = &dialogue
		return result, true
	}
	change := store.TransactAuthoredQuest(socket.playerBase.PlayerID, socket.selectedRole.RoleID, q.Info.ID, request.AnswerHandle == "complete")
	if !change.Changed {
		result.errorMessages = []classicTownErrorPush{{Msg: change.Error}}
		return result, true
	}
	socket.selectedRole, socket.playerBase = &change.Role, &change.PlayerBase
	switch {
	case request.AnswerHandle == "accept":
		result.questInfos = []classicQuestInfoPush{classicQuestInfoForRole(store, socket, q.Info)}
		result.chatMessages = []classicTownChatMessagePush{classicTownSystemChatMessage("接受了任务【" + q.Info.Title + "】。")}
		if change.GrantedItem != nil {
			item := *change.GrantedItem
			item.Handle = change.Role.RoleID
			result.itemInfos = append(result.itemInfos, classicTownItemInfoPushFromRoleItem(item))
			result.chatMessages = append(result.chatMessages, classicTownSystemChatMessage("获得了【"+item.Name+"】x1"))
		}
	case request.AnswerHandle == "complete":
		result.questClears = []classicQuestClearPush{{QuestID: q.Info.ID, Title: q.Info.Title}}
		result.roleState, result.rolePhysique = change.PlayerBase.RoleState, change.PlayerBase.RolePhysique
		result.chatMessages = []classicTownChatMessagePush{classicTownSystemChatMessage("完成了任务【" + q.Info.Title + "】。"), classicTownSystemChatMessage("获得经验:" + strconv.Itoa(q.Info.Reward.Experience))}
	}
	result.questStates = allAuthoredQuestStates(store, socket)
	result.questGuide = buildQuestGuideSnapshot(store, socket)
	// The client closes its answer page on submit. Successful transactions only
	// push state; a fresh AnswerSpeak would reopen an unsolicited reminder page.
	if len(result.questInfos) == 0 && len(result.questClears) == 0 {
		dialogue := authoredQuestDialogue(store, socket, q, request.Handle)
		result.answerSpeak = &dialogue
	}
	return result, true
}
