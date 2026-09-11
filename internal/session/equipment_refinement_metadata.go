package session

import "strings"

// Repair only the exact legacy template that omitted fields 19/23, not custom
// equipment variants. Capture: 20260524_113450_799_conn_0002/server-to-client.bin,
// f_i_蛮力面甲 at byte 90386, field 19: [精炼+1] 每升一级 物理防御+2.
// Insert the rule alone; preserve every instance field and current numeric value.
func missingEquipmentRefinementDescription(item RoleItem) (string, bool) {
	if item.ItemType != "equip" || strings.Contains(item.Description, "&19@") {
		return item.Description, false
	}
	template, ok := CapturedRoleItemTemplate(item.Name)
	if !ok || template.ItemType != "equip" || item.Display != template.Display {
		return item.Description, false
	}
	start := strings.Index(template.Description, "&19@")
	if start < 0 {
		return item.Description, false
	}
	end := classicDescriptionFieldEnd(template.Description, start+4)
	rule := template.Description[start:end]
	withoutRule := template.Description[:start] + template.Description[end:]
	legacy := withoutRule
	if notice := strings.Index(legacy, "&23@"); notice >= 0 {
		legacy = legacy[:notice] + legacy[classicDescriptionFieldEnd(legacy, notice+4):]
	}
	if item.Description != withoutRule && item.Description != legacy {
		return item.Description, false
	}
	return item.Description + rule, true
}
