package main

import (
	"log"

	"ai-server/internal/guild"
	"ai-server/internal/session"
)

type classicGuildNoticePush struct {
	GuildID string `json:"guildId,omitempty"`
	Notice  string `json:"notice"`
}

type classicGuildResultPush struct {
	Success      bool   `json:"success"`
	ErrorCode    string `json:"errorCode,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	GuildID      string `json:"guildId,omitempty"`
}

type classicGuildMemberClearPush struct {
	RoleID string `json:"roleId"`
}

func buildClassicGuildInfoResult(store *session.Store, socketSession *packetSession) packetResult {
	if !hasSelectedGuildRole(socketSession) || store == nil || store.Guilds == nil {
		log.Printf("[ai-server] classic guild GetGuidInfo ignored without selected role")
		return packetResult{handled: true}
	}
	result := store.Guilds.Info(socketSession.selectedRole.RoleID)
	return classicGuildPacketResult(result)
}

func buildClassicGuildCreateResult(store *session.Store, socketSession *packetSession, request guild.CreateRequest) packetResult {
	if !hasSelectedGuildRole(socketSession) || store == nil || store.Guilds == nil {
		log.Printf("[ai-server] classic guild CreateGuild ignored without selected role name=%s", request.Name)
		return packetResult{handled: true}
	}
	result := store.Guilds.Create(
		socketSession.selectedRole.RoleID,
		socketSession.selectedRole.DisplayName,
		socketSession.selectedRole.Level,
		request,
	)
	return classicGuildPacketResult(result)
}

func buildClassicGuildNoticeUpdateResult(store *session.Store, socketSession *packetSession, request guild.NoticeUpdateRequest) packetResult {
	if !hasSelectedGuildRole(socketSession) || store == nil || store.Guilds == nil {
		log.Printf("[ai-server] classic guild UpdateNotice ignored without selected role")
		return packetResult{handled: true}
	}
	result := store.Guilds.UpdateNotice(socketSession.selectedRole.RoleID, request.Notice)
	return classicGuildPacketResult(result)
}

func buildClassicGuildLeaveResult(store *session.Store, socketSession *packetSession) packetResult {
	if !hasSelectedGuildRole(socketSession) || store == nil || store.Guilds == nil {
		log.Printf("[ai-server] classic guild LeaveGuild ignored without selected role")
		return packetResult{handled: true}
	}
	result := store.Guilds.Leave(socketSession.selectedRole.RoleID)
	packetResult := classicGuildPacketResult(result)
	if result.Success {
		packetResult.guildMemberClears = []classicGuildMemberClearPush{{RoleID: socketSession.selectedRole.RoleID}}
	}
	return packetResult
}

func buildClassicGuildKickResult(store *session.Store, socketSession *packetSession, request guild.KickRequest) packetResult {
	if !hasSelectedGuildRole(socketSession) || store == nil || store.Guilds == nil {
		log.Printf("[ai-server] classic guild KickGuildMember ignored without selected role roleId=%s", request.RoleID)
		return packetResult{handled: true}
	}
	result := store.Guilds.Kick(socketSession.selectedRole.RoleID, request.RoleID)
	packetResult := classicGuildPacketResult(result)
	if result.Success {
		packetResult.guildMemberClears = []classicGuildMemberClearPush{{RoleID: request.RoleID}}
	}
	return packetResult
}

func buildClassicGuildDismissResult(store *session.Store, socketSession *packetSession) packetResult {
	if !hasSelectedGuildRole(socketSession) || store == nil || store.Guilds == nil {
		log.Printf("[ai-server] classic guild DismissGuild ignored without selected role")
		return packetResult{handled: true}
	}
	result := store.Guilds.Dismiss(socketSession.selectedRole.RoleID)
	return classicGuildPacketResult(result)
}

func classicGuildPacketResult(result guild.Result) packetResult {
	packetResult := packetResult{
		guildResult: &classicGuildResultPush{
			Success:      result.Success,
			ErrorCode:    result.ErrorCode,
			ErrorMessage: result.ErrorMessage,
		},
		handled: true,
	}
	if result.Info != nil {
		packetResult.guildInfo = result.Info
		packetResult.guildResult.GuildID = result.Info.ID
	}
	if len(result.Members) > 0 {
		packetResult.guildMembers = result.Members
	}
	if result.Auth != nil {
		packetResult.guildAuth = result.Auth
	}
	if result.Notice != "" || result.Info != nil {
		packetResult.guildNotice = &classicGuildNoticePush{
			Notice: result.Notice,
		}
		if result.Info != nil {
			packetResult.guildNotice.GuildID = result.Info.ID
		}
	}
	return packetResult
}

func hasSelectedGuildRole(socketSession *packetSession) bool {
	return socketSession != nil && socketSession.selectedRole != nil && socketSession.playerBase != nil
}
