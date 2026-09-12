package session

import "testing"

func TestRetiredSkillsCannotBeLearnedBoughtOrConsumedFromBook(t *testing.T) {
	role := RoleSummary{RoleID: "retired-book-test", Voc: "刃舞者", SkillCap: 20,
		Skills: []RoleSkill{{Name: "普通攻击", Level: 1}}, Currencies: RoleCurrencies{"铜钱": 100}}
	store := &Store{rolesByPID: map[string][]RoleSummary{"player": {role}}}
	skill := RoleSkill{Name: "密斩", Level: 1}
	if _, _, found, learned := store.LearnRoleSkill("player", role.RoleID, skill); !found || learned {
		t.Fatal("retired learning admitted")
	}
	if result := store.PurchaseRoleSkill("player", role.RoleID, skill, RoleCurrencies{"铜钱": 10}); result.ErrorCode != "invalid_skill" || store.rolesByPID["player"][0].Currencies["铜钱"] != 100 {
		t.Fatalf("retired purchase charged money: %+v", result)
	}
	book := RoleItem{Name: "密斩", Type: "背包", Index: 0, Count: 1, ItemType: "技能"}
	store.rolesByPID["player"][0].Items = []RoleItem{book}
	result := store.useSkillItemLocked("player", store.rolesByPID["player"], 0, book, skill)
	if result.ErrorCode != "invalid_skill" || store.rolesByPID["player"][0].Items[0].Count != 1 || len(store.rolesByPID["player"][0].Skills) != 1 {
		t.Fatalf("retired skill book mutated state: %+v", result)
	}
}

func TestRetiredSkillsRemovedFromRoleAndShortcuts(t *testing.T) {
	for _, vocation := range []string{"刃舞者", "战士", "术士"} {
		role := RoleSummary{RoleID: "retired-test", Voc: vocation, Level: 30,
			Skills:    []RoleSkill{{Name: "密斩", Level: 1}, {Name: "普通攻击", Level: 1}, {Name: "奥义.雷魂斩", Level: 5}},
			FastPanel: []RoleFastPanelEntry{{Index: 0, Type: "skill", Name: "密斩"}, {Index: 1, Type: "skill", Name: "普通攻击"}, {Index: 8, Type: "item", Name: "馒头"}}}
		got := refreshOriginalProfessionSkills(role)
		if got.Voc != vocation || got.Level != 30 || got.RoleID != role.RoleID || len(got.Skills) != 1 || got.Skills[0].Name != "普通攻击" || len(got.FastPanel) != 2 || got.FastPanel[1].Name != "馒头" {
			t.Fatalf("unexpected retirement result: %+v", got)
		}
	}
}
