package session

import "testing"

func TestCapturedRoleItemQualityColorUsesGeneratedItemTable(t *testing.T) {
	tests := []struct {
		name  string
		color string
	}{
		{name: "宝匣", color: "00ccff"},
		{name: "魔匣", color: "c156c7"},
		{name: "仙匣", color: "f9e000"},
		{name: "贰级原石", color: "f9e000"},
		{name: "高级精炼宝石", color: "f9e000"},
	}
	for _, test := range tests {
		if got := CapturedRoleItemQualityColor(test.name, ""); got != test.color {
			t.Fatalf("CapturedRoleItemQualityColor(%s)=%s, want %s", test.name, got, test.color)
		}
	}
}

func TestNormalizeRoleItemRefreshesOnlyCapturedTitleColor(t *testing.T) {
	item := normalizeRoleItem(RoleItem{
		Type:        "背包",
		Name:        "宝匣",
		ItemType:    "own",
		Description: "f_i_宝匣&24@宝物&25@99&20@保留原始说明",
		Count:       1,
		Index:       3,
		ItemLevel:   1,
	})
	if item.Description != "f_i_宝匣^00ccff&24@宝物&25@99&20@保留原始说明" {
		t.Fatalf("unexpected migrated description: %q", item.Description)
	}
}
