package main

import "ai-server/internal/session"

const cmdEquipmentStarRequest = 1260
const cmdEquipmentStarResponse = 1261

func buildEquipmentStarResult(store *session.Store, connection *packetSession, request session.EquipmentStarRequest) packetResult {
	response := classicTownInlayResponse{Message: "请先选择角色。", UpdatedItems: []classicTownItemInfoPush{}, ClearedItems: []classicTownItemInfoClearPush{}}
	result := packetResult{handled: true, responseCmd: cmdEquipmentStarResponse}
	if connection != nil && connection.playerBase != nil && connection.selectedRole != nil {
		operation := store.EquipmentStar(connection.playerBase.PlayerID, connection.selectedRole.RoleID, request)
		response.Success = operation.Success
		response.Message = operation.Message
		if operation.Success {
			connection.selectedRole = &operation.Role
			connection.playerBase = &operation.PlayerBase
			for _, item := range operation.UpdatedItems {
				item.Handle = operation.Role.RoleID
				response.UpdatedItems = append(response.UpdatedItems, classicTownItemInfoPushFromRoleItem(item))
			}
			for _, item := range operation.ClearedItems {
				response.ClearedItems = append(response.ClearedItems, classicTownItemInfoClearPush{Handle: operation.Role.RoleID, Type: item.Type, Index: item.Index})
			}
			result.rolePhysique = operation.PlayerBase.RolePhysique
		}
	}
	result.responsePayload = encodePayload(response)
	return result
}
