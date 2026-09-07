package main

import "ai-server/internal/session"

func buildClassicTownEquipmentSocketResult(store *session.Store, connection *packetSession, request classicTownActiveItemRequest) packetResult {
	warning := func(message string) packetResult {
		return packetResult{handled: true, chatMessages: []classicTownChatMessagePush{classicTownSystemWarningMessage(message)}}
	}
	if request.TargetType == "" || request.TargetIndex == nil || request.TargetLevel != nil {
		return warning("凿孔目标参数无效。")
	}
	operation := store.SocketRoleEquipment(connection.playerBase.PlayerID, connection.selectedRole.RoleID, request.Type, request.Index, request.TargetType, *request.TargetIndex)
	if !operation.Applied {
		return warning(operation.ErrorMessage)
	}
	connection.selectedRole = &operation.Role
	connection.playerBase = &operation.PlayerBase
	result := packetResult{handled: true, chatMessages: []classicTownChatMessagePush{classicTownSystemChatMessage(operation.ResultMessage)}}
	for _, item := range operation.UpdatedItems {
		item.Handle = operation.Role.RoleID
		result.itemInfos = append(result.itemInfos, classicTownItemInfoPushFromRoleItem(item))
	}
	for _, item := range operation.ClearedItems {
		result.itemClears = append(result.itemClears, classicTownItemInfoClearPush{Handle: operation.Role.RoleID, Type: item.Type, Index: item.Index})
	}
	return result
}
