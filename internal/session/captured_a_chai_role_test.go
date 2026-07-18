package session

import (
	"path/filepath"
	"testing"
)

func TestPersistentCapturedAChai555MigrationKeepsFinalSnapshot(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ai-server.db")
	store, err := NewPersistentStore(dbPath)
	if err != nil {
		t.Fatalf("create persistent store: %v", err)
	}

	login := mustLogin(t, store, "55555555", "55555555")
	created := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "555",
		Gender:         "male",
		RoleTemplateID: 1,
	})
	if !created.Success {
		t.Fatalf("create target role: %+v", created)
	}
	if _, err := store.MigrateCapturedAChaiRole(login.PlayerID, created.Role.RoleID); err != nil {
		t.Fatalf("migrate captured role: %v", err)
	}
	if err := store.MigrateCapturedAChaiQuests(login.PlayerID, created.Role.RoleID); err != nil {
		t.Fatalf("migrate captured quests: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close migrated store: %v", err)
	}

	reopened, err := NewPersistentStore(dbPath)
	if err != nil {
		t.Fatalf("reopen persistent store: %v", err)
	}
	defer reopened.Close()

	role, playerBase, ok := reopened.GetRoleRuntimeData(login.PlayerID, created.Role.RoleID)
	if !ok {
		t.Fatal("expected migrated role runtime data")
	}
	if role.DisplayName != "555" || role.Voc != "术士" || role.Level != 45 || role.Exp != 6299164 || role.MapID != 202 {
		t.Fatalf("unexpected migrated role summary: %+v", role)
	}
	if role.Currencies["铜钱"] != 968 || role.Currencies["银元宝"] != 30 {
		t.Fatalf("unexpected captured currencies: %+v", role.Currencies)
	}
	if role.ContainerCapacities["背包"] != 42 || role.ContainerCapacities["仓库"] != 38 || role.ContainerCapacities["装备"] != 20 || role.ContainerCapacities["战斗"] != 18 {
		t.Fatalf("unexpected captured container capacities: %+v", role.ContainerCapacities)
	}
	if playerBase.RoleState == nil || playerBase.RoleState.HP != 1068 || playerBase.RoleState.MP != 3077 || playerBase.RoleState.Lv != 45 || playerBase.RoleState.Exp != 6299164 || playerBase.RoleState.Speed != 145 {
		t.Fatalf("unexpected captured role state: %+v", playerBase.RoleState)
	}
	if playerBase.RolePhysique == nil || playerBase.RolePhysique.AGI != 80 || playerBase.RolePhysique.INT != 177 || playerBase.RolePhysique.MaxHP != 1105 || playerBase.RolePhysique.MaxMP != 3273 || playerBase.RolePhysique.MgcAtk != 367 {
		t.Fatalf("unexpected captured role physique: %+v", playerBase.RolePhysique)
	}
	for _, expected := range []struct {
		key   string
		value string
	}{
		{key: "a", value: "16"},
		{key: "b", value: "31"},
		{key: "c", value: "34"},
		{key: "h", value: "57"},
		{key: "p", value: "64"},
		{key: "se", value: "27"},
		{key: "w10", value: "58"},
		{key: "wr", value: "26"},
	} {
		roleValue, roleOK := sourceQueryParamValue(role.SourceQuery, expected.key)
		battleValue, battleOK := sourceQueryParamValue(role.BattleSourceQuery, expected.key)
		if !roleOK || !battleOK || roleValue != expected.value || battleValue != expected.value {
			t.Fatalf("expected captured appearance %s=%s, role=%q battle=%q", expected.key, expected.value, role.SourceQuery, role.BattleSourceQuery)
		}
	}

	skills, skillCap, ok := reopened.GetRoleSkills(login.PlayerID, created.Role.RoleID)
	if !ok || skillCap != 12 || len(skills) != 11 {
		t.Fatalf("unexpected captured skills cap=%d skills=%+v", skillCap, skills)
	}
	if skills[1].Name != "赤焰魔咒" || skills[1].Level != 2 || skills[9].Name != "雷爆咒" || skills[9].Level != 4 || skills[10].Name != "雷龙强袭" {
		t.Fatalf("unexpected captured skill order: %+v", skills)
	}
	fastPanel, ok := reopened.GetRoleFastPanel(login.PlayerID, created.Role.RoleID)
	if !ok || len(fastPanel) != 9 || fastPanel[1].Name != "炎狩术" || fastPanel[6].Name != "雷龙强袭" || fastPanel[7].Index != 8 || fastPanel[8].Name != "小瓶甘露" {
		t.Fatalf("unexpected captured fast panel: %+v", fastPanel)
	}

	bag, bagCapacity, ok := reopened.GetRoleItems(login.PlayerID, created.Role.RoleID, "背包")
	if !ok || bagCapacity != 42 || len(bag) != 34 {
		t.Fatalf("unexpected captured bag capacity=%d items=%d", bagCapacity, len(bag))
	}
	warehouse, warehouseCapacity, ok := reopened.GetRoleItems(login.PlayerID, created.Role.RoleID, "仓库")
	if !ok || warehouseCapacity != 38 || len(warehouse) != 21 {
		t.Fatalf("unexpected captured warehouse capacity=%d items=%d", warehouseCapacity, len(warehouse))
	}
	equipment, equipmentCapacity, ok := reopened.GetRoleItems(login.PlayerID, created.Role.RoleID, "装备")
	if !ok || equipmentCapacity != 20 || len(equipment) != 13 || equipment[3].Name != "流云法杖" || equipment[9].Name != "霹雳兽" {
		t.Fatalf("unexpected captured equipment capacity=%d items=%+v", equipmentCapacity, equipment)
	}

	accepted := reopened.AcceptedQuestTitles(login.PlayerID, created.Role.RoleID)
	if len(accepted) != 5 || !accepted["幻化白骨"] || !accepted["青铜密盒"] {
		t.Fatalf("unexpected captured quest state: %+v", accepted)
	}
	progress, ok := reopened.QuestProgress(login.PlayerID, created.Role.RoleID, "幻化白骨")
	if !ok || progress != 11 {
		t.Fatalf("unexpected captured quest progress=%d found=%t", progress, ok)
	}
}
