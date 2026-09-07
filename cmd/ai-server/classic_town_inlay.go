package main

import "ai-server/internal/session"

type classicTownInlayResponse struct {
	Success      bool                           `json:"success"`
	Message      string                         `json:"message"`
	UpdatedItems []classicTownItemInfoPush      `json:"updatedItems"`
	ClearedItems []classicTownItemInfoClearPush `json:"clearedItems"`
}

func buildClassicTownInlayResult(store *session.Store, connection *packetSession, request session.EquipmentInlayRequest) packetResult {
	response := classicTownInlayResponse{Message: "请先登录并选择角色。", UpdatedItems: []classicTownItemInfoPush{}, ClearedItems: []classicTownItemInfoClearPush{}}
	if connection != nil && connection.playerBase != nil && connection.selectedRole != nil {
		operation := store.InlayRoleEquipment(connection.playerBase.PlayerID, connection.selectedRole.RoleID, request)
		response.Success = operation.Applied
		response.Message = operation.Message
		if operation.Applied {
			connection.selectedRole = &operation.Role
			connection.playerBase = &operation.PlayerBase
			for _, item := range operation.UpdatedItems {
				item.Handle = operation.Role.RoleID
				response.UpdatedItems = append(response.UpdatedItems, classicTownItemInfoPushFromRoleItem(item))
			}
			for _, item := range operation.ClearedItems {
				response.ClearedItems = append(response.ClearedItems, classicTownItemInfoClearPush{Handle: operation.Role.RoleID, Type: item.Type, Index: item.Index})
			}
		}
	}
	return packetResult{handled: true, responseCmd: cmdClassicTownInlayResponse, responsePayload: encodePayload(response)}
}
