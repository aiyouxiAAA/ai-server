package session

import (
	"path/filepath"
	"testing"
)

func TestCapturedWoodcutter777RuntimeDefaultsOnlyTargetSnapshotRoles(t *testing.T) {
	for _, roleID := range []string{capturedWoodcutter777RoleID, capturedWoodcutter777LongRoleID} {
		if !isCapturedWoodcutter777LocalRole(RoleSummary{RoleID: roleID}) {
			t.Fatalf("expected captured 777 snapshot role %q", roleID)
		}
	}
	if isCapturedWoodcutter777LocalRole(RoleSummary{RoleID: "acct-777-role-002"}) {
		t.Fatal("expected later 777 account roles to keep their own runtime defaults")
	}
}

func TestPersistentCapturedWoodcutter777InventoryMigrationRunsOnce(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ai-server.db")
	firstStore, err := NewPersistentStore(dbPath)
	if err != nil {
		t.Fatalf("create persistent store: %v", err)
	}
	firstStoreClosed := false
	defer func() {
		if !firstStoreClosed {
			_ = firstStore.Close()
		}
	}()
	login := firstStore.Login(LoginRequest{UserName: "77777777", Password: "77777777"})
	created := firstStore.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "777",
		Gender:         "male",
		RoleTemplateID: 1,
	})
	if !created.Success {
		t.Fatalf("create 777 role: %+v", created)
	}

	legacyItems := append(capturedWoodcutter777EquipmentItems(), capturedWoodcutter777BagItems()...)
	filteredItems := make([]RoleItem, 0, len(legacyItems))
	for _, item := range legacyItems {
		if item.Type == "背包" && item.Index == 2 || item.Type == "装备" && item.Index == 3 {
			continue
		}
		filteredItems = append(filteredItems, item)
	}
	legacyArrow, ok := CapturedRoleItemTemplate("冰之箭")
	if !ok {
		t.Fatal("expected legacy bow material template")
	}
	legacyArrow.Type = "背包"
	legacyArrow.Index = 36
	legacyArrow.Count = 50
	filteredItems = append(filteredItems, legacyArrow)
	legacyItemsJSON, err := encodeRoleItems(filteredItems)
	if err != nil {
		t.Fatalf("encode legacy 777 items: %v", err)
	}
	legacyCurrenciesJSON, err := encodeRoleCurrencies(RoleCurrencies{"铜钱": 278, "银元宝": 190})
	if err != nil {
		t.Fatalf("encode legacy 777 currencies: %v", err)
	}
	if _, err := firstStore.db.Exec(
		`UPDATE roles SET items_json = ?, currencies_json = ? WHERE player_id = ? AND role_id = ?`,
		legacyItemsJSON,
		legacyCurrenciesJSON,
		login.PlayerID,
		created.Role.RoleID,
	); err != nil {
		t.Fatalf("seed stale 777 inventory: %v", err)
	}
	if err := firstStore.Close(); err != nil {
		t.Fatalf("close initial store: %v", err)
	}
	firstStoreClosed = true

	migratedStore, err := NewPersistentStore(dbPath)
	if err != nil {
		t.Fatalf("reopen migrated store: %v", err)
	}
	migratedStoreClosed := false
	defer func() {
		if !migratedStoreClosed {
			_ = migratedStore.Close()
		}
	}()
	bag, capacity, ok := migratedStore.GetRoleItems(login.PlayerID, created.Role.RoleID, "背包")
	if !ok || capacity != 42 || len(bag) != 38 {
		t.Fatalf("expected captured 777 bag after migration, ok=%v capacity=%d items=%+v", ok, capacity, bag)
	}
	teleportCrystal, ok := findRoleItem(bag, "背包", 2)
	if !ok || teleportCrystal.Name != "传送晶体" || teleportCrystal.Count != 1 {
		t.Fatalf("expected captured teleport crystal after migration, ok=%v item=%+v", ok, teleportCrystal)
	}
	equipment, _, ok := migratedStore.GetRoleItems(login.PlayerID, created.Role.RoleID, "装备")
	weapon, hasWeapon := findRoleItem(equipment, "装备", 3)
	if !ok || !hasWeapon || weapon.Name != "武雷拳套" {
		t.Fatalf("expected captured fist weapon after migration, ok=%v item=%+v", ok, weapon)
	}
	currencies, ok := migratedStore.GetRoleCurrencies(login.PlayerID, created.Role.RoleID)
	if !ok || currencies["铜钱"] != 1278 || currencies["银元宝"] != 189 {
		t.Fatalf("expected captured currencies after migration, ok=%v currencies=%+v", ok, currencies)
	}
	var migrationRows int
	if err := migratedStore.db.QueryRow(
		`SELECT COUNT(*) FROM role_snapshot_migrations WHERE role_id = ? AND migration_key = ?`,
		created.Role.RoleID,
		capturedWoodcutter777InventoryMigrationKey,
	).Scan(&migrationRows); err != nil || migrationRows != 1 {
		t.Fatalf("expected one recorded 777 migration, rows=%d err=%v", migrationRows, err)
	}

	movedCrystal := migratedStore.MoveRoleItem(login.PlayerID, created.Role.RoleID, "背包", 2, "背包", 36, 1)
	if !movedCrystal.Found || !movedCrystal.Moved || movedCrystal.ErrorCode != "" {
		t.Fatalf("move captured teleport crystal: %+v", movedCrystal)
	}
	movedWeapon := migratedStore.MoveRoleItem(login.PlayerID, created.Role.RoleID, "装备", 3, "背包", 40, 1)
	if !movedWeapon.Found || !movedWeapon.Moved || movedWeapon.ErrorCode != "" {
		t.Fatalf("unequip captured fist weapon: %+v", movedWeapon)
	}
	usedFood := migratedStore.ConsumeRoleItemMutationOnly(login.PlayerID, created.Role.RoleID, "背包", 18, 1)
	if !usedFood.Found || !usedFood.Used || usedFood.ErrorCode != "" {
		t.Fatalf("consume captured food: %+v", usedFood)
	}
	if err := migratedStore.Close(); err != nil {
		t.Fatalf("close migrated store: %v", err)
	}
	migratedStoreClosed = true

	reopened, err := NewPersistentStore(dbPath)
	if err != nil {
		t.Fatalf("reopen after player mutations: %v", err)
	}
	defer reopened.Close()
	bag, _, ok = reopened.GetRoleItems(login.PlayerID, created.Role.RoleID, "背包")
	if !ok {
		t.Fatal("expected persisted 777 bag after player mutations")
	}
	teleportCrystal, ok = findRoleItem(bag, "背包", 36)
	if !ok || teleportCrystal.Name != "传送晶体" || teleportCrystal.Count != 1 {
		t.Fatalf("expected moved teleport crystal to survive restart, ok=%v item=%+v", ok, teleportCrystal)
	}
	if _, exists := findRoleItem(bag, "背包", 2); exists {
		t.Fatalf("expected empty original teleport slot to survive restart, items=%+v", bag)
	}
	weapon, hasWeapon = findRoleItem(bag, "背包", 40)
	if !hasWeapon || weapon.Name != "武雷拳套" {
		t.Fatalf("expected unequipped fist weapon to survive restart, item=%+v", weapon)
	}
	food, ok := findRoleItem(bag, "背包", 18)
	if !ok || food.Name != "馒头" || food.Count != 92 {
		t.Fatalf("expected consumed food count to survive restart, ok=%v item=%+v", ok, food)
	}
	equipment, _, ok = reopened.GetRoleItems(login.PlayerID, created.Role.RoleID, "装备")
	if !ok {
		t.Fatal("expected persisted 777 equipment")
	}
	if _, exists := findRoleItem(equipment, "装备", 3); exists {
		t.Fatalf("expected empty weapon equipment slot to survive restart, equipment=%+v", equipment)
	}
}
