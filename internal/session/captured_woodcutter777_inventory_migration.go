package session

import (
	"database/sql"
	"fmt"
)

const (
	capturedWoodcutter777InventoryMigrationKey = "captured-woodcutter-777-inventory-v1"
	capturedWoodcutter777RoleID                = "acct-777-role-001"
	capturedWoodcutter777LongRoleID            = "acct-77777777-role-001"
)

// applyCapturedWoodcutter777InventorySnapshot repairs the one local account
// that was seeded before the final captured inventory was made persistent.
func applyCapturedWoodcutter777InventorySnapshot(role RoleSummary) RoleSummary {
	items := make([]RoleItem, 0, len(capturedWoodcutter777EquipmentItems())+len(capturedWoodcutter777BagItems()))
	items = append(items, capturedWoodcutter777EquipmentItems()...)
	items = append(items, capturedWoodcutter777BagItems()...)
	role.Items = normalizeRoleItems(items)
	role.Currencies = cloneRoleCurrencies(capturedWoodcutter777Currencies())
	return role
}

func (store *Store) applyPendingCapturedWoodcutter777InventoryMigration() ([]string, error) {
	if store.db == nil {
		return nil, nil
	}

	migratedRoleIDs := []string{}
	for playerID, roles := range store.rolesByPID {
		for index := range roles {
			if !isCapturedWoodcutter777InventoryMigrationRole(roles[index]) {
				continue
			}

			var migrated int
			err := store.db.QueryRow(
				`SELECT 1 FROM role_snapshot_migrations WHERE role_id = ? AND migration_key = ?`,
				roles[index].RoleID,
				capturedWoodcutter777InventoryMigrationKey,
			).Scan(&migrated)
			if err == nil {
				continue
			}
			if err != sql.ErrNoRows {
				return nil, fmt.Errorf("check captured 777 inventory migration roleId=%s: %w", roles[index].RoleID, err)
			}

			roles[index] = applyCapturedWoodcutter777InventorySnapshot(roles[index])
			migratedRoleIDs = append(migratedRoleIDs, roles[index].RoleID)
		}
		store.rolesByPID[playerID] = roles
	}
	return migratedRoleIDs, nil
}

func isCapturedWoodcutter777InventoryMigrationRole(role RoleSummary) bool {
	return role.RoleID == capturedWoodcutter777RoleID || role.RoleID == capturedWoodcutter777LongRoleID
}

func (store *Store) recordCapturedWoodcutter777InventoryMigration(roleID string) error {
	if store.db == nil {
		return nil
	}
	if _, err := store.db.Exec(
		`INSERT INTO role_snapshot_migrations (role_id, migration_key) VALUES (?, ?)`,
		roleID,
		capturedWoodcutter777InventoryMigrationKey,
	); err != nil {
		return fmt.Errorf("record captured 777 inventory migration: %w", err)
	}
	return nil
}
