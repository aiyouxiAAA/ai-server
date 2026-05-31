package main

import (
	"log"
	"strings"

	"ai-server/internal/session"
	"ai-server/internal/world"
)

func buildClassicTownVocationResult(store *session.Store, socketSession *packetSession, request classicTownAnswerRequest) (packetResult, bool) {
	vocation, ok := classicTownAnswerVocation(request)
	if !ok {
		return packetResult{}, false
	}
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		return packetResult{handled: true}, true
	}

	result := store.SetRoleVocation(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, vocation)
	if !result.Found {
		log.Printf("[ai-server] classic town vocation rejected roleId=%s vocation=%s error=%s", socketSession.selectedRole.RoleID, vocation, result.ErrorCode)
		return packetResult{handled: true}, true
	}

	socketSession.selectedRole = &result.Role
	socketSession.playerBase = &result.PlayerBase
	answerSpeak := world.AnswerSpeakPush{
		Handle:    request.Handle,
		MsgHandle: request.AnswerHandle,
		Msg:       "你已经成为<font color='#990000'><b>" + vocation + "</b></font>。",
		Answers: []world.AnswerOption{
			{Handle: "x", Msg: "<c/>关闭"},
		},
	}
	log.Printf("[ai-server] classic town vocation roleId=%s vocation=%s changed=%v", result.Role.RoleID, vocation, result.Changed)
	return packetResult{
		answerSpeak:  &answerSpeak,
		createPlayer: buildClassicTownCreatePlayerPush(result.Role, result.PlayerBase),
		roleState:    &result.RoleState,
		rolePhysique: &result.RolePhysique,
		chatMessages: []classicTownChatMessagePush{classicTownSystemChatMessage("已转职为【" + vocation + "】。")},
		handled:      true,
	}, true
}

func classicTownAnswerVocation(request classicTownAnswerRequest) (string, bool) {
	if strings.TrimSpace(request.Handle) != sourceSkillTeacherHandle || strings.TrimSpace(request.MsgHandle) != "2" {
		return "", false
	}
	switch strings.TrimSpace(request.AnswerHandle) {
	case "job_warrior":
		return "战士", true
	case "job_sorcerer":
		return "术士", true
	case "job_ranger":
		return "游侠", true
	default:
		return "", false
	}
}
