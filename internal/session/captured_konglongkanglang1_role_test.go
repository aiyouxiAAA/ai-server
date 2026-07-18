package session

import (
	"path/filepath"
	"testing"
)

func TestPersistentCapturedKonglongKanglang1MigrationKeepsFinalSnapshot(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ai-server.db")
	store, err := NewPersistentStore(dbPath)
	if err != nil {
		t.Fatalf("create persistent store: %v", err)
	}

	login := mustLogin(t, store, "66666666", "66666666")
	created := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "666",
		Gender:         "male",
		RoleTemplateID: 1,
	})
	if !created.Success {
		t.Fatalf("create target role: %+v", created)
	}
	if _, err := store.MigrateCapturedKonglongKanglang1Role(login.PlayerID, created.Role.RoleID); err != nil {
		t.Fatalf("migrate captured role: %v", err)
	}
	if err := store.MigrateCapturedKonglongKanglang1Quests(login.PlayerID, created.Role.RoleID); err != nil {
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
	if role.DisplayName != "666" || role.Voc != "战士" || role.Level != 50 || role.Exp != 8338724 || role.MapID != 213 || role.MapX != 384 || role.MapY != 547 {
		t.Fatalf("unexpected migrated role summary: %+v", role)
	}
	if role.Currencies["铜钱"] != 417 || role.Currencies["银元宝"] != 155 {
		t.Fatalf("unexpected captured currencies: %+v", role.Currencies)
	}
	if role.ContainerCapacities["背包"] != 42 || role.ContainerCapacities["仓库"] != 38 || role.ContainerCapacities["装备"] != 20 || role.ContainerCapacities["战斗"] != 18 {
		t.Fatalf("unexpected captured container capacities: %+v", role.ContainerCapacities)
	}
	if playerBase.RoleState == nil || playerBase.RoleState.HP != 1974 || playerBase.RoleState.MP != 424 || playerBase.RoleState.Lv != 50 || playerBase.RoleState.Exp != 8338724 || playerBase.RoleState.Speed != 147 {
		t.Fatalf("unexpected captured role state: %+v", playerBase.RoleState)
	}
	if playerBase.RolePhysique == nil || playerBase.RolePhysique.AGI != 87 || playerBase.RolePhysique.STR != 192 || playerBase.RolePhysique.MaxHP != 2005 || playerBase.RolePhysique.MaxMP != 584 || playerBase.RolePhysique.PhyAtk != 392 || playerBase.RolePhysique.PhyDef != 230 {
		t.Fatalf("unexpected captured role physique: %+v", playerBase.RolePhysique)
	}
	for _, query := range []string{role.SourceQuery, role.BattleSourceQuery} {
		for _, expected := range []struct {
			key   string
			value string
		}{
			{key: "c", value: "61"},
			{key: "p", value: "63"},
			{key: "se", value: "49"},
			{key: "hr", value: "12"},
			{key: "w7", value: "59"},
		} {
			value, found := sourceQueryParamValue(query, expected.key)
			if !found || value != expected.value {
				t.Fatalf("expected captured appearance %s=%s, query=%q", expected.key, expected.value, query)
			}
		}
		for _, staleKey := range []string{"h", "a", "wr", "b"} {
			if _, found := sourceQueryParamValue(query, staleKey); found {
				t.Fatalf("captured full fashion must remove stale %s, query=%q", staleKey, query)
			}
		}
	}

	skills, skillCap, ok := reopened.GetRoleSkills(login.PlayerID, created.Role.RoleID)
	expectedSkills := []struct {
		name  string
		level int
	}{
		{name: "普通攻击", level: 1},
		{name: "武器专精", level: 5},
		{name: "强体质", level: 5},
		{name: "抗击打", level: 5},
		{name: "奥义.六合棍法", level: 4},
		{name: "挑衅", level: 1},
		{name: "卷叶式", level: 5},
		{name: "强贯式", level: 5},
		{name: "凝神式", level: 5},
		{name: "奥义.飘血", level: 4},
		{name: "气愈式", level: 5},
		{name: "狂舞式", level: 5},
	}
	if !ok || skillCap != 12 || len(skills) != len(expectedSkills) {
		t.Fatalf("unexpected captured skills cap=%d skills=%+v", skillCap, skills)
	}
	for index, expected := range expectedSkills {
		if skills[index].Name != expected.name || skills[index].Level != expected.level {
			t.Fatalf("unexpected captured skill at index %d: got=%+v want=%+v", index, skills[index], expected)
		}
	}
	fastPanel, ok := reopened.GetRoleFastPanel(login.PlayerID, created.Role.RoleID)
	if !ok || len(fastPanel) != 9 || fastPanel[1].Name != "卷叶式" || fastPanel[2].Name != "挑衅" || fastPanel[6].Name != "奥义.飘血" || fastPanel[7].Index != 8 || fastPanel[8].Name != "小瓶甘露" {
		t.Fatalf("unexpected captured fast panel: %+v", fastPanel)
	}
	for _, container := range []struct {
		name  string
		count int
	}{
		{name: "背包", count: 42},
		{name: "仓库", count: 38},
		{name: "装备", count: 14},
	} {
		items, _, found := reopened.GetRoleItems(login.PlayerID, created.Role.RoleID, container.name)
		if !found || len(items) != container.count {
			t.Fatalf("unexpected captured %s item count=%d items=%+v", container.name, len(items), items)
		}
	}
	battleItemCount := 0
	for _, item := range role.Items {
		if item.Handle == "player_21424" {
			t.Fatalf("captured item retained original role handle: %+v", item)
		}
		if item.Type == "战斗" {
			battleItemCount += 1
		}
	}
	if battleItemCount != 1 {
		t.Fatalf("expected one captured battle-container item, got %d in %+v", battleItemCount, role.Items)
	}
	if len(role.TownBuffs) != 0 {
		t.Fatalf("temporary capture buffs must not migrate into town state: %+v", role.TownBuffs)
	}

	accepted := reopened.AcceptedQuestTitles(login.PlayerID, created.Role.RoleID)
	if len(accepted) != 4 || !accepted["无缘情份"] || !accepted["幻化白骨"] {
		t.Fatalf("unexpected captured quest state: %+v", accepted)
	}
	objectives, ok := reopened.QuestObjectiveProgress(login.PlayerID, created.Role.RoleID, "无缘情份")
	if !ok || len(objectives) != 3 || objectives[0] != (RoleQuestObjectiveProgress{Name: "毛皮", Current: 20, Target: 20}) || objectives[1] != (RoleQuestObjectiveProgress{Name: "裘皮", Current: 0, Target: 1}) || objectives[2] != (RoleQuestObjectiveProgress{Name: "碎金片", Current: 1, Target: 5}) {
		t.Fatalf("unexpected captured multi-objective quest state: %+v", objectives)
	}
}
