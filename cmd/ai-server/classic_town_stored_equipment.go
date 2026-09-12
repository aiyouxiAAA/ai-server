package main

import "ai-server/internal/session"

func buildClassicTownStoredEquipmentResult(store *session.Store, request classicTownOtherEquipmentRequest) packetResult {
	request = normalizeClassicTownOtherEquipmentRequest(request)
	push := &classicTownOtherEquipmentPush{
		Handle: firstNonBlankString(request.Handle, request.RoleID, request.RoleName),
		RoleID: request.RoleID, RoleName: request.RoleName, Type: classicTownLookEquipmentType,
		Items: []classicTownItemInfoPush{}, ErrorCode: "role_missing", ErrorMessage: "角色不存在。",
	}
	if store == nil {
		return packetResult{otherEquipment: push, handled: true}
	}
	identity := firstNonBlankString(request.RoleID, request.Handle)
	var player string
	var role session.RoleSummary
	var found bool
	if identity != "" {
		player, role, found = store.FindRoleByID(identity)
	} else {
		player, role, found = store.FindRoleByDisplayName(request.RoleName)
	}
	if !found {
		return packetResult{otherEquipment: push, handled: true}
	}
	items, capacity, ok := store.GetRoleItems(player, role.RoleID, classicTownLookEquipmentType)
	if !ok {
		return packetResult{otherEquipment: push, handled: true}
	}
	push.Handle, push.RoleID, push.RoleName = role.RoleID, role.RoleID, role.DisplayName
	push.Capacity, push.ErrorCode, push.ErrorMessage = capacity, "", ""
	for _, item := range items {
		item.Handle = role.RoleID
		push.Items = append(push.Items, classicTownItemInfoPushFromRoleItem(item))
	}
	return packetResult{otherEquipment: push, handled: true}
}
