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
	if role.DisplayName != "555" || role.Voc != "术士" || role.Level != 50 || role.Exp != 8269210 || role.MapID != 202 {
		t.Fatalf("unexpected migrated role summary: %+v", role)
	}
	if role.Currencies["铜钱"] != 968 || role.Currencies["银元宝"] != 30 {
		t.Fatalf("unexpected captured currencies: %+v", role.Currencies)
	}
	if role.ContainerCapacities["背包"] != 42 || role.ContainerCapacities["仓库"] != 38 || role.ContainerCapacities["装备"] != 20 || role.ContainerCapacities["战斗"] != 18 {
		t.Fatalf("unexpected captured container capacities: %+v", role.ContainerCapacities)
	}
	if playerBase.RoleState == nil || playerBase.RoleState.HP != 1177 || playerBase.RoleState.MP != 3479 || playerBase.RoleState.Lv != 50 || playerBase.RoleState.Exp != 8269210 || playerBase.RoleState.Speed != 145 {
		t.Fatalf("unexpected captured role state: %+v", playerBase.RoleState)
	}
	if playerBase.RolePhysique == nil || playerBase.RolePhysique.AGI != 91 || playerBase.RolePhysique.INT != 200 || playerBase.RolePhysique.MaxHP != 1205 || playerBase.RolePhysique.MaxMP != 3649 || playerBase.RolePhysique.MgcAtk != 398 || playerBase.RolePhysique.MgcDef != 328 {
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
	if !ok || skillCap != 16 || len(skills) != 16 {
		t.Fatalf("unexpected captured skills cap=%d skills=%+v", skillCap, skills)
	}
	expectedSkills := []struct {
		name  string
		level int
	}{
		{name: "普通攻击", level: 1},
		{name: "赤焰魔咒", level: 5},
		{name: "炎狩术", level: 5},
		{name: "愈气术", level: 5},
		{name: "御气经", level: 5},
		{name: "精元经", level: 5},
		{name: "苦心经", level: 5},
		{name: "魔障术", level: 5},
		{name: "雷爆咒", level: 5},
		{name: "圣光诀", level: 1},
		{name: "回伤术", level: 1},
		{name: "还魂术", level: 5},
		{name: "雷击", level: 5},
		{name: "火神咒", level: 5},
		{name: "石雨术", level: 5},
		{name: "雷龙强袭", level: 1},
	}
	for index, expected := range expectedSkills {
		if skills[index].Name != expected.name || skills[index].Level != expected.level {
			t.Fatalf("unexpected captured skill[%d], want=%+v got=%+v all=%+v", index, expected, skills[index], skills)
		}
	}
	fastPanel, ok := reopened.GetRoleFastPanel(login.PlayerID, created.Role.RoleID)
	if !ok || len(fastPanel) != 10 || fastPanel[1].Name != "炎狩术" || fastPanel[3].Name != "圣光诀" || fastPanel[6].Name != "火神咒" || fastPanel[7].Index != 7 || fastPanel[7].Name != "石雨术" || fastPanel[8].Index != 8 || fastPanel[9].Name != "小瓶甘露" {
		t.Fatalf("unexpected captured fast panel: %+v", fastPanel)
	}

	bag, bagCapacity, ok := reopened.GetRoleItems(login.PlayerID, created.Role.RoleID, "背包")
	if !ok || bagCapacity != 42 || len(bag) != 35 {
		t.Fatalf("unexpected captured bag capacity=%d items=%d", bagCapacity, len(bag))
	}
	var soulStone RoleItem
	for _, item := range bag {
		if item.Name == "魂之石" {
			soulStone = item
			break
		}
	}
	if soulStone.Type != "背包" || soulStone.ItemType != "null" || soulStone.Display != "248.png" || soulStone.Count != 1 || soulStone.Index != 41 || soulStone.ItemLevel != 3 {
		t.Fatalf("unexpected captured 魂之石 bag item: %+v", soulStone)
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

func TestPersistentCapturedAChai555SkillMigrationKeepsOtherRoleState(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ai-server.db")
	store, err := NewPersistentStore(dbPath)
	if err != nil {
		t.Fatalf("create persistent store: %v", err)
	}
	defer store.Close()

	login := mustLogin(t, store, "55555555", "55555555")
	created := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "keep-current-state",
		Gender:         "male",
		RoleTemplateID: 1,
	})
	if !created.Success {
		t.Fatalf("create target role: %+v", created)
	}
	before, _, ok := store.GetRoleRuntimeData(login.PlayerID, created.Role.RoleID)
	if !ok {
		t.Fatal("expected created role runtime data")
	}
	migrated, err := store.MigrateCapturedAChaiSkills(login.PlayerID, created.Role.RoleID)
	if err != nil {
		t.Fatalf("migrate captured skills: %v", err)
	}
	if migrated.DisplayName != before.DisplayName || migrated.Level != before.Level || migrated.MapID != before.MapID || migrated.Voc != before.Voc {
		t.Fatalf("skill-only migration changed role state: before=%+v migrated=%+v", before, migrated)
	}
	if migrated.SkillCap != 16 || len(migrated.Skills) != 16 || migrated.Skills[12].Name != "雷击" || migrated.Skills[13].Name != "火神咒" || migrated.Skills[14].Name != "石雨术" || migrated.Skills[15].Name != "雷龙强袭" || len(migrated.FastPanel) != 10 || migrated.FastPanel[3].Name != "圣光诀" || migrated.FastPanel[6].Name != "火神咒" || migrated.FastPanel[7].Name != "石雨术" {
		t.Fatalf("unexpected migrated captured skills: %+v fastPanel=%+v", migrated.Skills, migrated.FastPanel)
	}
}

func TestPersistentCapturedAChaiLevel50StatsAndSkillsMigrationRunsOnce(t *testing.T) {
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
	if created.Role.RoleID != capturedAChaiLevel50StatsAndSkillsRoleID {
		t.Fatalf("unexpected migration target role id: %s", created.Role.RoleID)
	}

	store.mu.Lock()
	roles := store.rolesByPID[login.PlayerID]
	roles[0].Level = 45
	roles[0].Exp = 6299164
	roles[0].SkillCap = 12
	roles[0].Skills = cloneRoleSkills(capturedAChaiSnapshot.Role.Skills[:12])
	roles[0].FastPanel = []RoleFastPanelEntry{
		{Index: 0, Type: "skill", Name: "普通攻击"},
		{Index: 7, Type: "skill", Name: "石雨术"},
	}
	store.rolesByPID[login.PlayerID] = roles
	if err := store.persistRoleStateLocked(login.PlayerID, created.Role.RoleID); err != nil {
		store.mu.Unlock()
		t.Fatalf("persist legacy a chai state: %v", err)
	}
	store.mu.Unlock()
	if err := store.Close(); err != nil {
		t.Fatalf("close legacy store: %v", err)
	}

	migrated, err := NewPersistentStore(dbPath)
	if err != nil {
		t.Fatalf("reopen migration store: %v", err)
	}
	role, playerBase, ok := migrated.GetRoleRuntimeData(login.PlayerID, created.Role.RoleID)
	if !ok || role.Level != 50 || role.Exp != 8269210 || playerBase.RoleState == nil || playerBase.RoleState.HP != 1177 || playerBase.RolePhysique == nil || playerBase.RolePhysique.INT != 200 {
		migrated.Close()
		t.Fatalf("expected captured level-50 state after migration, role=%+v base=%+v", role, playerBase)
	}
	skills, skillCap, ok := migrated.GetRoleSkills(login.PlayerID, created.Role.RoleID)
	if !ok || skillCap != 16 || len(skills) != 16 || skills[12].Name != "雷击" || skills[15].Name != "雷龙强袭" {
		migrated.Close()
		t.Fatalf("expected restored two-page skill list, cap=%d skills=%+v", skillCap, skills)
	}
	fastPanel, ok := migrated.GetRoleFastPanel(login.PlayerID, created.Role.RoleID)
	if !ok || len(fastPanel) != 2 || fastPanel[1].Index != 7 || fastPanel[1].Name != "石雨术" {
		migrated.Close()
		t.Fatalf("expected current fast panel to remain untouched, entries=%+v", fastPanel)
	}
	var migrationRows int
	if err := migrated.db.QueryRow(
		`SELECT COUNT(*) FROM role_snapshot_migrations WHERE role_id = ? AND migration_key = ?`,
		created.Role.RoleID,
		capturedAChaiLevel50StatsAndSkillsMigrationKey,
	).Scan(&migrationRows); err != nil || migrationRows != 1 {
		migrated.Close()
		t.Fatalf("expected one migration marker, rows=%d err=%v", migrationRows, err)
	}
	if removed := migrated.RemoveRoleSkill(login.PlayerID, created.Role.RoleID, "雷龙强袭"); !removed.Removed {
		migrated.Close()
		t.Fatalf("remove migrated skill: %+v", removed)
	}
	if err := migrated.Close(); err != nil {
		t.Fatalf("close migrated store: %v", err)
	}

	reopened, err := NewPersistentStore(dbPath)
	if err != nil {
		t.Fatalf("reopen completed migration store: %v", err)
	}
	defer reopened.Close()
	skills, skillCap, ok = reopened.GetRoleSkills(login.PlayerID, created.Role.RoleID)
	if !ok || skillCap != 16 || len(skills) != 15 {
		t.Fatalf("completed migration overwrote later skill mutation, cap=%d skills=%+v", skillCap, skills)
	}
	for _, skill := range skills {
		if skill.Name == "雷龙强袭" {
			t.Fatalf("completed migration restored deleted skill: %+v", skills)
		}
	}
	if err := reopened.db.QueryRow(
		`SELECT COUNT(*) FROM role_snapshot_migrations WHERE role_id = ? AND migration_key = ?`,
		created.Role.RoleID,
		capturedAChaiLevel50StatsAndSkillsMigrationKey,
	).Scan(&migrationRows); err != nil || migrationRows != 1 {
		t.Fatalf("expected one stable migration marker, rows=%d err=%v", migrationRows, err)
	}
}
