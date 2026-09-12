package main

import (
	"ai-server/internal/quest"
	"ai-server/internal/world"
	"strings"
)

// Apply at the packet boundary so stored source dialogues cannot re-offer removed tasks.
func removeRetiredQuestContent(result *packetResult) {
	if result.townBootstrap != nil {
		for i := range result.townBootstrap.QuestStates {
			if !strings.HasPrefix(result.townBootstrap.QuestStates[i].Handle, "authored-") {
				result.townBootstrap.QuestStates[i].State = 0
			}
		}
	}
	for i := range result.questStates {
		if !strings.HasPrefix(result.questStates[i].Handle, "authored-") {
			result.questStates[i].State = 0
		}
	}
	dialogue := result.answerSpeak
	if dialogue == nil || strings.HasPrefix(dialogue.Handle, "authored-") {
		return
	}
	options := make([]world.AnswerOption, 0, len(dialogue.Answers))
	for _, option := range dialogue.Answers {
		if !quest.IsRetiredAnswer(option.Handle, option.Msg) {
			options = append(options, option)
		}
	}
	dialogue.Answers = options
}
