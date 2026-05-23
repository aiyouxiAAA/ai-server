package main

import (
	"log"
	"strings"

	"ai-server/internal/world"
)

const sourceMainQuestTitle = "初入云隐"
const sourceMainQuestNpcHandle = "1000542608713897"

type classicQuestInfoPush struct {
	Title       string `json:"title"`
	Level       int    `json:"level"`
	Description string `json:"description"`
	State       string `json:"state"`
}

type classicQuestClearPush struct {
	Title   string `json:"title"`
	QuestID string `json:"questId,omitempty"`
}

type classicQuestRemoveRequest struct {
	Title   string `json:"title,omitempty"`
	QuestID string `json:"questId,omitempty"`
}

func buildClassicQuestLogResult(socketSession *packetSession) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic quest GetQuestLog ignored without selected role")
		return packetResult{handled: true}
	}

	if socketSession.removedQuests != nil && socketSession.removedQuests[sourceMainQuestTitle] {
		return packetResult{handled: true}
	}

	return packetResult{
		questInfos: []classicQuestInfoPush{sourceMainQuestInfo()},
		handled:    true,
	}
}

func buildClassicQuestRemoveResult(socketSession *packetSession, request classicQuestRemoveRequest) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic quest RemoveQuest ignored without selected role title=%s questId=%s", request.Title, request.QuestID)
		return packetResult{handled: true}
	}

	title := strings.TrimSpace(request.Title)
	if title == "" && strings.TrimSpace(request.QuestID) == "main-001" {
		title = sourceMainQuestTitle
	}
	if title != sourceMainQuestTitle {
		log.Printf("[ai-server] classic quest RemoveQuest ignored missing title=%s questId=%s", request.Title, request.QuestID)
		return packetResult{handled: true}
	}

	if socketSession.removedQuests == nil {
		socketSession.removedQuests = make(map[string]bool)
	}
	if socketSession.removedQuests[title] {
		return packetResult{handled: true}
	}
	socketSession.removedQuests[title] = true

	return packetResult{
		questClears: []classicQuestClearPush{{
			Title:   title,
			QuestID: "main-001",
		}},
		questStates: []world.QuestStatePush{{
			Handle: sourceMainQuestNpcHandle,
			State:  0,
		}},
		handled: true,
	}
}

func sourceMainQuestInfo() classicQuestInfoPush {
	return classicQuestInfoPush{
		Title:       sourceMainQuestTitle,
		Level:       1,
		Description: "<ml>拜访一心长态<br/>[g]=前往云隐村，和一心长态交谈，熟悉江湖任务委托。",
		State:       "进行中",
	}
}
