package main

import (
	"log"

	"ai-server/internal/session"
)

type classicPetInfoRequest struct {
	Handle string `json:"handle,omitempty"`
}

type classicPetFeedRequest struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
	Count int    `json:"count"`
}

type classicPetInfoPush struct {
	Handle       string `json:"handle"`
	HasPet       bool   `json:"hasPet"`
	Level        int    `json:"level,omitempty"`
	Exp          int    `json:"exp,omitempty"`
	Fullness     int    `json:"fullness,omitempty"`
	Name         string `json:"name,omitempty"`
	PetType      string `json:"petType,omitempty"`
	DisplayURL   string `json:"displayUrl,omitempty"`
	Display      string `json:"display,omitempty"`
	SourceX      int    `json:"sourceX,omitempty"`
	SourceY      int    `json:"sourceY,omitempty"`
	SkillHTML    string `json:"skillHtml,omitempty"`
	PetID        string `json:"petId,omitempty"`
	Status       string `json:"status,omitempty"`
	ErrorCode    string `json:"errorCode,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

type classicPetFeedResultPush struct {
	Success      bool   `json:"success"`
	ItemType     string `json:"itemType,omitempty"`
	ItemIndex    int    `json:"itemIndex,omitempty"`
	Count        int    `json:"count,omitempty"`
	ErrorCode    string `json:"errorCode,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

func buildClassicPetInfoResult(store *session.Store, socketSession *packetSession, request classicPetInfoRequest) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic pet GetPetInfo ignored without selected role handle=%s", request.Handle)
		return packetResult{handled: true}
	}

	info := store.GetRolePetInfo(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID)
	if !info.Found {
		log.Printf("[ai-server] classic pet GetPetInfo ignored missing role roleId=%s", socketSession.selectedRole.RoleID)
		return packetResult{handled: true}
	}
	socketSession.selectedRole = &info.Role
	socketSession.playerBase = &info.PlayerBase
	return packetResult{
		petInfo: classicPetInfoPushFromRolePetInfo(info),
		handled: true,
	}
}

func buildClassicPetFeedResult(store *session.Store, socketSession *packetSession, request classicPetFeedRequest) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic pet PetFeed ignored without selected role type=%s index=%d", request.Type, request.Index)
		return packetResult{handled: true}
	}

	feedResult := store.FeedRolePet(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, request.Type, request.Index, request.Count)
	if !feedResult.Found {
		log.Printf("[ai-server] classic pet PetFeed ignored missing role roleId=%s", socketSession.selectedRole.RoleID)
		return packetResult{handled: true}
	}
	socketSession.selectedRole = &feedResult.Role
	socketSession.playerBase = &feedResult.PlayerBase
	result := packetResult{
		petInfo:       classicPetInfoPushFromRolePetInfo(feedResult.RolePetInfoResult),
		petFeedResult: classicPetFeedResultPushFromRolePetFeedResult(feedResult),
		itemClears:    make([]classicTownItemInfoClearPush, 0, len(feedResult.ClearedItems)),
		handled:       true,
	}
	if feedResult.UpdatedItem != nil {
		updatedItem := *feedResult.UpdatedItem
		updatedItem.Handle = feedResult.Role.RoleID
		result.itemInfos = append(result.itemInfos, classicTownItemInfoPushFromRoleItem(updatedItem))
	}
	for _, clear := range feedResult.ClearedItems {
		result.itemClears = append(result.itemClears, classicTownItemInfoClearPush{
			Handle: feedResult.Role.RoleID,
			Type:   clear.Type,
			Index:  clear.Index,
		})
	}
	return result
}

func classicPetInfoPushFromRolePetInfo(info session.RolePetInfoResult) *classicPetInfoPush {
	push := &classicPetInfoPush{
		Handle:       info.Role.RoleID,
		HasPet:       info.HasPet,
		Level:        info.Level,
		Exp:          info.Exp,
		Fullness:     info.Fullness,
		Name:         info.Name,
		PetType:      info.PetType,
		DisplayURL:   info.DisplayURL,
		Display:      info.Item.Display,
		SourceX:      info.SourceX,
		SourceY:      info.SourceY,
		SkillHTML:    info.SkillHTML,
		PetID:        info.PetID,
		Status:       info.Status,
		ErrorCode:    info.ErrorCode,
		ErrorMessage: info.ErrorMessage,
	}
	if !info.HasPet {
		push.Level = 0
		push.Exp = 0
		push.Fullness = 0
	}
	return push
}

func classicPetFeedResultPushFromRolePetFeedResult(result session.RolePetFeedResult) *classicPetFeedResultPush {
	return &classicPetFeedResultPush{
		Success:      result.Fed,
		ItemType:     result.FeedItem.Type,
		ItemIndex:    result.FeedItem.Index,
		Count:        result.FeedItem.Count,
		ErrorCode:    result.ErrorCode,
		ErrorMessage: result.ErrorMessage,
	}
}
