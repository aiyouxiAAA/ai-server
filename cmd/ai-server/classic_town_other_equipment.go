package main

import (
	"strings"

	"ai-server/internal/session"
)

const (
	classicTownLookEquipmentType          = "装备"
	classicTownLookEquipmentCapturedRole  = "player_21424"
	classicTownLookEquipmentCapturedName  = "恐龙抗狼1"
	classicTownLookEquipmentSourceCapture = "instance2.staging/tmp/woc-proxy-captures/20260612_211747_199_session_15948/connections/20260612_211850_206_conn_0002/raw/client-to-server-0001.bin#5866 GetConrainerCapacity(恐龙抗狼1,装备); #5867 GetLookDetail(player_21424); raw/server-to-client-0001.bin#4438 c_ContainerCapacity(player_21424,装备,20); #4439+ c_ItemInfo"
)

type classicTownOtherEquipmentRequest struct {
	RoleName string `json:"roleName"`
	RoleID   string `json:"roleId,omitempty"`
	Handle   string `json:"handle,omitempty"`
}

type classicTownOtherEquipmentPush struct {
	Handle        string                    `json:"handle"`
	RoleID        string                    `json:"roleId,omitempty"`
	RoleName      string                    `json:"roleName"`
	Type          string                    `json:"type"`
	Capacity      int                       `json:"capacity"`
	Items         []classicTownItemInfoPush `json:"items"`
	SourceCapture string                    `json:"sourceCapture,omitempty"`
	Partial       bool                      `json:"partial,omitempty"`
	ErrorCode     string                    `json:"errorCode,omitempty"`
	ErrorMessage  string                    `json:"errorMessage,omitempty"`
}

type classicTownLookEquipmentSpec struct {
	name  string
	index int
}

func buildClassicTownOtherEquipmentResult(request classicTownOtherEquipmentRequest) packetResult {
	request = normalizeClassicTownOtherEquipmentRequest(request)
	if !isClassicTownCapturedLookEquipmentRequest(request) {
		return packetResult{
			otherEquipment: &classicTownOtherEquipmentPush{
				Handle:        firstNonBlankString(request.Handle, request.RoleID, request.RoleName),
				RoleID:        request.RoleID,
				RoleName:      request.RoleName,
				Type:          classicTownLookEquipmentType,
				Capacity:      0,
				Items:         []classicTownItemInfoPush{},
				SourceCapture: classicTownLookEquipmentSourceCapture,
				Partial:       true,
				ErrorCode:     "look_equipment_unrestored",
				ErrorMessage:  "观察装备数据未恢复",
			},
			handled: true,
		}
	}

	items, ok := classicTownCapturedLookEquipmentItems()
	if !ok {
		return packetResult{
			otherEquipment: &classicTownOtherEquipmentPush{
				Handle:        classicTownLookEquipmentCapturedRole,
				RoleID:        classicTownLookEquipmentCapturedRole,
				RoleName:      classicTownLookEquipmentCapturedName,
				Type:          classicTownLookEquipmentType,
				Capacity:      0,
				Items:         []classicTownItemInfoPush{},
				SourceCapture: classicTownLookEquipmentSourceCapture,
				Partial:       true,
				ErrorCode:     "look_equipment_template_missing",
				ErrorMessage:  "观察装备模板缺失",
			},
			handled: true,
		}
	}

	return packetResult{
		otherEquipment: &classicTownOtherEquipmentPush{
			Handle:        classicTownLookEquipmentCapturedRole,
			RoleID:        classicTownLookEquipmentCapturedRole,
			RoleName:      classicTownLookEquipmentCapturedName,
			Type:          classicTownLookEquipmentType,
			Capacity:      20,
			Items:         items,
			SourceCapture: classicTownLookEquipmentSourceCapture,
			Partial:       true,
		},
		handled: true,
	}
}

func normalizeClassicTownOtherEquipmentRequest(request classicTownOtherEquipmentRequest) classicTownOtherEquipmentRequest {
	request.RoleName = strings.TrimSpace(request.RoleName)
	request.RoleID = strings.TrimSpace(request.RoleID)
	request.Handle = strings.TrimSpace(request.Handle)
	return request
}

func isClassicTownCapturedLookEquipmentRequest(request classicTownOtherEquipmentRequest) bool {
	return request.Handle == classicTownLookEquipmentCapturedRole ||
		request.RoleID == classicTownLookEquipmentCapturedRole ||
		request.RoleName == classicTownLookEquipmentCapturedName
}

func classicTownCapturedLookEquipmentItems() ([]classicTownItemInfoPush, bool) {
	specs := []classicTownLookEquipmentSpec{
		{name: "无双头盔", index: 0},
		{name: "无双护肩", index: 1},
		{name: "无双铁腕", index: 2},
		{name: "饮血刀", index: 3},
		{name: "寨夫人上衣", index: 4},
		{name: "无双护腿", index: 5},
		{name: "泥戒指", index: 6},
		{name: "怪木机", index: 9},
		{name: "无双铁腰带", index: 10},
		{name: "蛤蟆精战靴", index: 12},
	}
	items := make([]classicTownItemInfoPush, 0, len(specs))
	for _, spec := range specs {
		item, ok := session.CapturedRoleItemTemplate(spec.name)
		if !ok {
			return nil, false
		}
		item.Handle = classicTownLookEquipmentCapturedRole
		item.Type = classicTownLookEquipmentType
		item.ItemType = "equip"
		item.Count = 1
		item.Index = spec.index
		items = append(items, classicTownItemInfoPushFromRoleItem(item))
	}
	return items, true
}

func firstNonBlankString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
