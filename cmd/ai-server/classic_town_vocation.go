package main

import (
	"log"
	"strings"

	"ai-server/internal/profession"
	"ai-server/internal/session"
	"ai-server/internal/world"
)

var classicTownVocationByAnswerHandle = mustLoadClassicTownVocationsByAnswerHandle()

func mustLoadClassicTownVocationsByAnswerHandle() map[string]string {
	result := make(map[string]string, len(profession.Definitions))
	for _, definition := range profession.Definitions {
		result["job_"+definition.ID] = definition.Name
	}
	return result
}

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
		chatMessages: []classicTownChatMessagePush{classicTownSystemChatMessage("成功转职为【" + vocation + "】。")},
		handled:      true,
	}, true
}

func classicTownAnswerVocation(request classicTownAnswerRequest) (string, bool) {
	if strings.TrimSpace(request.Handle) != sourceSkillTeacherHandle || strings.TrimSpace(request.MsgHandle) != "2" {
		return "", false
	}
	vocation, ok := classicTownVocationByAnswerHandle[strings.TrimSpace(request.AnswerHandle)]
	return vocation, ok
}
