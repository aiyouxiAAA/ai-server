package main

import (
	"strings"

	"ai-server/internal/session"
	"ai-server/internal/team"
	"ai-server/internal/world"
)

type classicTeamInviteRequest struct {
	TargetRoleID string `json:"targetRoleId,omitempty"`
	TargetName   string `json:"targetName,omitempty"`
}

type classicTeamInviteReplyRequest struct {
	InviteID string `json:"inviteId"`
	Accept   bool   `json:"accept"`
}

type classicTeamMemberTargetRequest struct {
	RoleID string `json:"roleId,omitempty"`
	Name   string `json:"name,omitempty"`
}

type classicTeamSyncChangeMapRequest struct {
	Enabled bool `json:"enabled"`
}

func buildClassicTeamInviteResult(socketSession *packetSession, request classicTeamInviteRequest) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		return packetResult{handled: true}
	}
	classicTeamManager.UpsertOnline(classicTeamMemberFromSession(socketSession))
	events := classicTeamManager.Invite(socketSession.selectedRole.RoleID, request.TargetRoleID, request.TargetName)
	result := packetResult{teamEvents: events, handled: true}
	if classicTeamEventsContainResult(events, socketSession.selectedRole.RoleID, true, "invite") {
		targetName := strings.TrimSpace(request.TargetName)
		if targetName != "" {
			result.chatMessages = append(result.chatMessages, classicTownSystemChatMessage("你已经请求["+targetName+"]加入队伍,请等待对方确认"))
		}
	}
	return result
}

func buildClassicTeamInviteReplyResult(socketSession *packetSession, request classicTeamInviteReplyRequest) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		return packetResult{handled: true}
	}
	classicTeamManager.UpsertOnline(classicTeamMemberFromSession(socketSession))
	events := classicTeamManager.ReplyInvite(socketSession.selectedRole.RoleID, request.InviteID, request.Accept)
	result := packetResult{teamEvents: events, handled: true}
	if request.Accept && !classicTeamEventsContainResult(events, socketSession.selectedRole.RoleID, false, "replyInvite") {
		if recipients, ok := classicTeamManager.RecipientsForTeam(socketSession.selectedRole.RoleID); ok && len(recipients) > 0 {
			result.chatBroadcasts = append(result.chatBroadcasts, classicTownChatBroadcast{
				Recipients: recipients,
				Message:    classicTownSystemChatMessage("[" + socketSession.selectedRole.DisplayName + "]加入队伍"),
			})
		}
	}
	return result
}

func buildClassicTeamLeaveResult(socketSession *packetSession) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil {
		return packetResult{handled: true}
	}
	events := classicTeamManager.Leave(socketSession.selectedRole.RoleID)
	return packetResult{teamEvents: events, handled: true}
}

func buildClassicTeamKickResult(socketSession *packetSession, request classicTeamMemberTargetRequest) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil {
		return packetResult{handled: true}
	}
	events := classicTeamManager.Kick(socketSession.selectedRole.RoleID, request.RoleID, request.Name)
	return packetResult{teamEvents: events, handled: true}
}

func buildClassicTeamTransferLeaderResult(socketSession *packetSession, request classicTeamMemberTargetRequest) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil {
		return packetResult{handled: true}
	}
	events := classicTeamManager.TransferLeader(socketSession.selectedRole.RoleID, request.RoleID, request.Name)
	return packetResult{teamEvents: events, handled: true}
}

func buildClassicTeamSyncChangeMapResult(socketSession *packetSession, request classicTeamSyncChangeMapRequest) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil {
		return packetResult{handled: true}
	}
	events := classicTeamManager.SetSyncChangeMap(socketSession.selectedRole.RoleID, request.Enabled)
	return packetResult{teamEvents: events, handled: true}
}

func buildClassicTeamResetDungeonResult(store *session.Store, socketSession *packetSession) packetResult {
	if store == nil || socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		return packetResult{handled: true}
	}
	mapID := socketSession.playerBase.MapID
	if mapID <= 0 {
		mapID = socketSession.selectedRole.MapID
	}
	instanceKey, ok := world.DungeonInstanceKeyForMapID(mapID)
	if !ok {
		return packetResult{
			teamEvents: []team.Event{classicTeamResultEvent(socketSession.selectedRole.RoleID, false, "resetDungeon", "NO_DUNGEON_INSTANCE", "当前没有可重置的副本。")},
			handled:    true,
		}
	}
	plan := classicTeamManager.BuildDungeonResetPlan(socketSession.selectedRole.RoleID)
	if !plan.Allowed {
		return packetResult{
			teamEvents: []team.Event{classicTeamResultEvent(socketSession.selectedRole.RoleID, false, "resetDungeon", plan.ErrorCode, plan.ErrorMessage)},
			handled:    true,
		}
	}
	if !store.ResetRoleDungeonInstance(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, instanceKey) {
		return packetResult{
			teamEvents: []team.Event{classicTeamResultEvent(socketSession.selectedRole.RoleID, false, "resetDungeon", "ROLE_NOT_FOUND", "角色状态不存在。")},
			handled:    true,
		}
	}
	setDefeatedVisibleMonsterHandles(socketSession, nil)
	inactiveDungeon := inactiveDungeonInstancePush(instanceKey, mapID)
	return packetResult{
		dungeonInstance: inactiveDungeon,
		mapSpecial:      buildClassicTownMapSpecialPush(inactiveDungeon),
		teamEvents: []team.Event{
			classicTeamResultEvent(socketSession.selectedRole.RoleID, true, "resetDungeon", "", ""),
		},
		teamDungeonReset: &classicTeamDungeonReset{
			ActorRoleID: socketSession.selectedRole.RoleID,
			InstanceKey: instanceKey,
			MapID:       mapID,
			Members:     plan.Members,
		},
		handled: true,
	}
}

func classicTeamResultEvent(roleID string, success bool, action string, code string, message string) team.Event {
	return team.Event{
		Recipients: []string{strings.TrimSpace(roleID)},
		Result: &team.ResultPush{
			Success:      success,
			Action:       action,
			ErrorCode:    code,
			ErrorMessage: message,
		},
	}
}

func classicTeamEventsContainResult(events []team.Event, roleID string, success bool, action string) bool {
	roleID = strings.TrimSpace(roleID)
	for _, event := range events {
		if event.Result == nil || event.Result.Success != success || event.Result.Action != action {
			continue
		}
		for _, recipient := range event.Recipients {
			if strings.TrimSpace(recipient) == roleID {
				return true
			}
		}
	}
	return false
}

func buildClassicTeamChatEvents(socketSession *packetSession, push classicTownChatMessagePush) ([]classicTownChatMessagePush, []string, bool) {
	if socketSession == nil || socketSession.selectedRole == nil {
		return nil, nil, false
	}
	recipients, ok := classicTeamManager.RecipientsForTeam(socketSession.selectedRole.RoleID)
	if !ok || len(recipients) == 0 {
		return []classicTownChatMessagePush{classicTownSystemWarningMessage("你还没有队伍。")}, nil, false
	}
	push.Channel = "team"
	push.Msg = strings.TrimSpace(push.Msg)
	if push.Msg == "" {
		return nil, nil, false
	}
	return []classicTownChatMessagePush{push}, recipients, true
}
