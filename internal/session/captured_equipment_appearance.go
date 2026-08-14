package session

import (
	"strings"

	"ai-server/internal/classicdata"
)

// Captured equipment appearance fields are read from equipment-appearance-table.
// Rows marked packet_confirmed have a same-role c_ItemInfo -> c_RoleInfo transition.
func capturedEquipmentAppearanceSourceParam(item RoleItem) (string, string, bool) {
	row, found, err := classicdata.FindEquipmentAppearanceByName(strings.TrimSpace(item.Name))
	if err != nil || !found {
		return "", "", false
	}
	key := strings.TrimSpace(row["source_key"])
	value := strings.TrimSpace(row["source_value"])
	if key == "" || value == "" {
		return "", "", false
	}
	return key, value, true
}
