package world

import (
	"ai-server/internal/profession"
	"strings"
)

// Replace only profession/skill branches of existing teachers; their quests still use the same dialogue routing.
func buildOriginalProfessionReply(handle, msgHandle, answerHandle string) *AnswerSpeakPush {
	npc := findSourceNPC(handle)
	if npc == nil || npc.Dialogue == nil {
		return nil
	}
	isTeacher := false
	for _, answer := range npc.Dialogue.Answers {
		if answer.Msg == "学习技能" {
			isTeacher = true
		}
	}
	if !isTeacher {
		return nil
	}
	if msgHandle == "1" && answerHandle == "2" && handle == "1000542608713897" {
		answers := []AnswerOption{}
		for _, definition := range profession.Definitions {
			answers = append(answers, AnswerOption{Handle: "job_" + definition.ID, Msg: "选择【" + definition.Name + "】"})
		}
		answers = append(answers, AnswerOption{Handle: "x", Msg: "<c/>关闭"})
		return &AnswerSpeakPush{Handle: handle, MsgHandle: "2", Msg: "当前开放以下职业。", Answers: answers}
	}
	if (msgHandle == "1" && (answerHandle == "1" || answerHandle == "show")) || msgHandle == "2" || msgHandle == "10" || strings.HasPrefix(msgHandle, "show") || msgHandle == "sshow" {
		names := []string{}
		for _, skill := range profession.Skills {
			names = append(names, skill.Name)
		}
		return &AnswerSpeakPush{Handle: handle, MsgHandle: "profession-skills", Msg: profession.Definitions[0].Name + "默认掌握：" + strings.Join(names, "、") + "。格挡为被动技能，招式可在战斗中使用。", Answers: []AnswerOption{{Handle: "x", Msg: "<c/>关闭"}}}
	}
	return nil
}
