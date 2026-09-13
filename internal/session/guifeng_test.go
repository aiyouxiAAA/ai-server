package session

import "testing"

func TestGuifengRefreshRetiresUltimateAndKeepsSkillCount(t *testing.T) {
	role := applyOriginalProfession(RoleSummary{RoleID: "migration", DisplayName: "测试", Voc: "刃舞者"})
	for i := range role.Skills {
		if role.Skills[i].Name == "断月·归锋" {
			role.Skills[i].Name = "奥义"
		}
	}
	for i := range role.FastPanel {
		if role.FastPanel[i].Name == "断月·归锋" {
			role.FastPanel[i].Name = "奥义"
		}
	}
	role = refreshOriginalProfessionSkills(role)
	if len(role.Skills) != 8 || len(role.FastPanel) != 7 {
		t.Fatalf("replacement increased defaults: %+v", role)
	}
	for _, s := range role.Skills {
		if s.Name == "奥义" {
			t.Fatal("retired learned skill retained")
		}
	}
	found := false
	for _, entry := range role.FastPanel {
		if entry.Index == 3 && entry.Name == "断月·归锋" {
			found = true
		}
	}
	if !found {
		t.Fatal("replacement not in prior shortcut")
	}
	again := refreshOriginalProfessionSkills(role)
	if len(again.Skills) != 8 || len(again.FastPanel) != 7 {
		t.Fatal("replacement repeated on next login")
	}
}
