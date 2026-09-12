package main

import (
	"ai-server/internal/quest"
	"ai-server/internal/session"
	"ai-server/internal/world"
	"strings"
)

const authoredQuestMenuID = "village-quests"

func finishOrStart(start, finish int) int {
	priority := map[int]int{0: 0, 4: 1, 2: 2, 1: 3}
	if priority[finish] > priority[start] {
		return finish
	}
	return start
}

func allAuthoredQuestStates(store *session.Store, socket *packetSession) []world.QuestStatePush {
	result := []world.QuestStatePush{}
	indices := map[string]int{}
	if socket == nil || socket.playerBase == nil {
		return result
	}
	for _, q := range quest.Authored() {
		if socket.playerBase.MapID != q.MapID {
			continue
		}
		for _, state := range authoredQuestStates(store, socket, q) {
			if i, ok := indices[state.Handle]; ok {
				result[i].State = finishOrStart(result[i].State, state.State)
			} else {
				indices[state.Handle] = len(result)
				result = append(result, state)
			}
		}
	}
	return result
}

func authoredNPCQuests(store *session.Store, socket *packetSession, handle string) []quest.AuthoredQuest {
	result := []quest.AuthoredQuest{}
	for _, q := range quest.Authored() {
		status := authoredQuestStatus(store, socket, q)
		if (status == "available" && q.Start.Handle() == handle) || (status == "accepted" && (q.Start.Handle() == handle || q.Finish.Handle() == handle)) {
			result = append(result, q)
		}
	}
	return result
}

func appendAuthoredServiceOptions(socket *packetSession, dialogue *world.AnswerSpeakPush) {
	npc, ok := world.FindAuthoredServiceNPC(dialogue.Handle)
	if !ok || authoredServiceError(socket, npc) != "" {
		return
	}
	service := authoredServiceDialogue(socket, npc)
	for _, option := range service.Answers {
		option.Handle = "service:" + option.Handle
		dialogue.Answers = append(dialogue.Answers, option)
	}
}

func buildAuthoredNPCMenu(store *session.Store, socket *packetSession, handle string) (packetResult, bool) {
	first, known := quest.AuthoredForNPC(handle)
	if !known {
		return packetResult{}, false
	}
	if !authoredQuestRoleReady(socket, first) {
		return packetResult{handled: true, errorMessages: []classicTownErrorPush{{Msg: "请在村内与该人物交谈。"}}}, true
	}
	candidates := authoredNPCQuests(store, socket, handle)
	if len(candidates) == 0 {
		if _, service := world.FindAuthoredServiceNPC(handle); service {
			return packetResult{}, false
		}
	}
	dialogue := world.AnswerSpeakPush{Handle: handle, MsgHandle: authoredQuestMenuID, Msg: "有什么需要我帮忙的？", Answers: []world.AnswerOption{}}
	if len(candidates) <= 1 {
		selected := first
		if len(candidates) == 1 {
			selected = candidates[0]
		} else {
			for _, q := range quest.Authored() {
				if (q.Start.Handle() == handle || q.Finish.Handle() == handle) && authoredQuestStatus(store, socket, q) == "completed" {
					selected = q
				}
			}
		}
		dialogue = authoredQuestDialogue(store, socket, selected, handle)
	} else {
		for _, q := range candidates {
			dialogue.Answers = append(dialogue.Answers, world.AnswerOption{Handle: "quest:" + q.Info.ID, Msg: "<ml><m/>" + q.Info.Title})
		}
	}
	appendAuthoredServiceOptions(socket, &dialogue)
	return packetResult{handled: true, answerSpeak: &dialogue, questStates: allAuthoredQuestStates(store, socket)}, true
}

func selectAuthoredNPCQuest(store *session.Store, socket *packetSession, request classicTownAnswerRequest) (packetResult, bool) {
	if strings.HasPrefix(request.AnswerHandle, "service:") {
		request.MsgHandle = authoredServiceMessageID
		request.AnswerHandle = strings.TrimPrefix(request.AnswerHandle, "service:")
		return buildAuthoredServiceAnswerResult(store, socket, request)
	}
	for _, q := range authoredNPCQuests(store, socket, request.Handle) {
		if request.AnswerHandle == "quest:"+q.Info.ID && authoredQuestRoleReady(socket, q) {
			dialogue := authoredQuestDialogue(store, socket, q, request.Handle)
			appendAuthoredServiceOptions(socket, &dialogue)
			return packetResult{handled: true, answerSpeak: &dialogue}, true
		}
	}
	return packetResult{handled: true, errorMessages: []classicTownErrorPush{{Msg: "任务对话已失效，请重新交谈。"}}}, true
}
