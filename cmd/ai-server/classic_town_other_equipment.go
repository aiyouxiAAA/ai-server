package main

import (
	"strings"
)

const (
	classicTownLookEquipmentType          = "装备"
	classicTownLookEquipmentSourceCapture = "9 captured GetLookDetail targets; see .planning/CLASSIC_TOWN_PLAYER_MENU_SOURCE_RESTORE_PLAN.md"
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

func normalizeClassicTownOtherEquipmentRequest(request classicTownOtherEquipmentRequest) classicTownOtherEquipmentRequest {
	request.RoleName = strings.TrimSpace(request.RoleName)
	request.RoleID = strings.TrimSpace(request.RoleID)
	request.Handle = strings.TrimSpace(request.Handle)
	return request
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
