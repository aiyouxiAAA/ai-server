package session

import (
	"fmt"
	"strings"
)

func (store *Store) Close() error {
	if store.db == nil {
		return nil
	}

	return store.db.Close()
}

func (store *Store) initSchema() error {
	if store.db == nil {
		return nil
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS accounts (
			user_name TEXT PRIMARY KEY,
			password TEXT NOT NULL,
			player_id TEXT NOT NULL UNIQUE,
			display_name TEXT NOT NULL,
			session_token TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS role_sequences (
			player_id TEXT PRIMARY KEY,
			next_role_seq INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS roles (
			role_id TEXT PRIMARY KEY,
			player_id TEXT NOT NULL,
			display_name TEXT NOT NULL,
			level INTEGER NOT NULL,
			exp INTEGER NOT NULL DEFAULT 0,
			voc TEXT NOT NULL DEFAULT '',
			agi INTEGER NOT NULL DEFAULT 0,
			str INTEGER NOT NULL DEFAULT 0,
			intelligence INTEGER NOT NULL DEFAULT 0,
			con INTEGER NOT NULL DEFAULT 0,
			lck INTEGER NOT NULL DEFAULT 0,
			map_id INTEGER NOT NULL,
			visual_role_id INTEGER NOT NULL,
			preset_id INTEGER NOT NULL DEFAULT 0,
			source_query TEXT NOT NULL DEFAULT '',
			battle_source_query TEXT NOT NULL DEFAULT '',
			appearance_json TEXT NOT NULL DEFAULT '',
			skills_json TEXT NOT NULL DEFAULT '',
			fast_panel_json TEXT NOT NULL DEFAULT '',
			currencies_json TEXT NOT NULL DEFAULT '',
			items_json TEXT NOT NULL DEFAULT '',
			role_state_json TEXT NOT NULL DEFAULT '',
			role_physique_json TEXT NOT NULL DEFAULT '',
			dungeon_instances_json TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS role_removed_quests (
			role_id TEXT NOT NULL,
			title TEXT NOT NULL,
			player_id TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (role_id, title)
		)`,
		`CREATE TABLE IF NOT EXISTS role_accepted_quests (
			role_id TEXT NOT NULL,
			title TEXT NOT NULL,
			player_id TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (role_id, title)
		)`,
	}

	for _, query := range queries {
		if _, err := store.db.Exec(query); err != nil {
			return fmt.Errorf("initialize sqlite schema: %w", err)
		}
	}
	if _, err := store.db.Exec(`ALTER TABLE roles ADD COLUMN skills_json TEXT NOT NULL DEFAULT ''`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			return fmt.Errorf("migrate roles skills_json column: %w", err)
		}
	}
	if _, err := store.db.Exec(`ALTER TABLE roles ADD COLUMN fast_panel_json TEXT NOT NULL DEFAULT ''`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			return fmt.Errorf("migrate roles fast_panel_json column: %w", err)
		}
	}
	if _, err := store.db.Exec(`ALTER TABLE roles ADD COLUMN currencies_json TEXT NOT NULL DEFAULT ''`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			return fmt.Errorf("migrate roles currencies_json column: %w", err)
		}
	}
	if _, err := store.db.Exec(`ALTER TABLE roles ADD COLUMN items_json TEXT NOT NULL DEFAULT ''`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			return fmt.Errorf("migrate roles items_json column: %w", err)
		}
	}
	if _, err := store.db.Exec(`ALTER TABLE roles ADD COLUMN role_state_json TEXT NOT NULL DEFAULT ''`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			return fmt.Errorf("migrate roles role_state_json column: %w", err)
		}
	}
	if _, err := store.db.Exec(`ALTER TABLE roles ADD COLUMN role_physique_json TEXT NOT NULL DEFAULT ''`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			return fmt.Errorf("migrate roles role_physique_json column: %w", err)
		}
	}
	if _, err := store.db.Exec(`ALTER TABLE roles ADD COLUMN dungeon_instances_json TEXT NOT NULL DEFAULT ''`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			return fmt.Errorf("migrate roles dungeon_instances_json column: %w", err)
		}
	}
	if _, err := store.db.Exec(`ALTER TABLE roles ADD COLUMN battle_source_query TEXT NOT NULL DEFAULT ''`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			return fmt.Errorf("migrate roles battle_source_query column: %w", err)
		}
	}
	if _, err := store.db.Exec(`ALTER TABLE roles ADD COLUMN exp INTEGER NOT NULL DEFAULT 0`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			return fmt.Errorf("migrate roles exp column: %w", err)
		}
	}
	if _, err := store.db.Exec(`ALTER TABLE roles ADD COLUMN voc TEXT NOT NULL DEFAULT ''`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			return fmt.Errorf("migrate roles voc column: %w", err)
		}
	}
	for _, migration := range []struct {
		column string
		query  string
	}{
		{column: "agi", query: `ALTER TABLE roles ADD COLUMN agi INTEGER NOT NULL DEFAULT 0`},
		{column: "str", query: `ALTER TABLE roles ADD COLUMN str INTEGER NOT NULL DEFAULT 0`},
		{column: "intelligence", query: `ALTER TABLE roles ADD COLUMN intelligence INTEGER NOT NULL DEFAULT 0`},
		{column: "con", query: `ALTER TABLE roles ADD COLUMN con INTEGER NOT NULL DEFAULT 0`},
		{column: "lck", query: `ALTER TABLE roles ADD COLUMN lck INTEGER NOT NULL DEFAULT 0`},
	} {
		if _, err := store.db.Exec(migration.query); err != nil {
			if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
				return fmt.Errorf("migrate roles %s column: %w", migration.column, err)
			}
		}
	}

	return nil
}

func (store *Store) loadFromDB() error {
	if store.db == nil {
		return nil
	}

	accountRows, err := store.db.Query(`
		SELECT user_name, password, player_id, display_name, session_token
		FROM accounts
	`)
	if err != nil {
		return fmt.Errorf("query accounts: %w", err)
	}
	defer accountRows.Close()

	for accountRows.Next() {
		var account AccountRecord
		if err := accountRows.Scan(
			&account.UserName,
			&account.Password,
			&account.PlayerID,
			&account.DisplayName,
			&account.SessionToken,
		); err != nil {
			return fmt.Errorf("scan account: %w", err)
		}
		store.accountsByName[account.UserName] = account
	}
	if err := accountRows.Err(); err != nil {
		return fmt.Errorf("iterate accounts: %w", err)
	}

	roleSeqRows, err := store.db.Query(`
		SELECT player_id, next_role_seq
		FROM role_sequences
	`)
	if err != nil {
		return fmt.Errorf("query role sequences: %w", err)
	}
	defer roleSeqRows.Close()

	for roleSeqRows.Next() {
		var playerID string
		var nextRoleSeq int
		if err := roleSeqRows.Scan(&playerID, &nextRoleSeq); err != nil {
			return fmt.Errorf("scan role sequence: %w", err)
		}
		store.nextRoleSeqByPID[playerID] = nextRoleSeq
	}
	if err := roleSeqRows.Err(); err != nil {
		return fmt.Errorf("iterate role sequences: %w", err)
	}

	roleRows, err := store.db.Query(`
		SELECT role_id, player_id, display_name, level, exp, voc, agi, str, intelligence, con, lck, map_id, visual_role_id, preset_id, source_query, battle_source_query, appearance_json, skills_json, fast_panel_json, currencies_json, items_json, role_state_json, role_physique_json, dungeon_instances_json
		FROM roles
		ORDER BY player_id ASC, role_id ASC
	`)
	if err != nil {
		return fmt.Errorf("query roles: %w", err)
	}
	defer roleRows.Close()

	for roleRows.Next() {
		var role RoleSummary
		var playerID string
		var appearanceJSON string
		var skillsJSON string
		var fastPanelJSON string
		var currenciesJSON string
		var itemsJSON string
		var roleStateJSON string
		var rolePhysiqueJSON string
		var dungeonInstancesJSON string
		if err := roleRows.Scan(
			&role.RoleID,
			&playerID,
			&role.DisplayName,
			&role.Level,
			&role.Exp,
			&role.Voc,
			&role.AGI,
			&role.STR,
			&role.INT,
			&role.CON,
			&role.LCK,
			&role.MapID,
			&role.VisualRoleID,
			&role.PresetID,
			&role.SourceQuery,
			&role.BattleSourceQuery,
			&appearanceJSON,
			&skillsJSON,
			&fastPanelJSON,
			&currenciesJSON,
			&itemsJSON,
			&roleStateJSON,
			&rolePhysiqueJSON,
			&dungeonInstancesJSON,
		); err != nil {
			return fmt.Errorf("scan role: %w", err)
		}
		appearance, err := decodeRoleAppearance(appearanceJSON)
		if err != nil {
			return fmt.Errorf("decode role appearance for %s: %w", role.RoleID, err)
		}
		role.Appearance = appearance
		skills, err := decodeRoleSkills(skillsJSON)
		if err != nil {
			return fmt.Errorf("decode role skills for %s: %w", role.RoleID, err)
		}
		role.Skills = skills
		fastPanel, err := decodeRoleFastPanel(fastPanelJSON)
		if err != nil {
			return fmt.Errorf("decode role fast panel for %s: %w", role.RoleID, err)
		}
		role.FastPanel = fastPanel
		currencies, err := decodeRoleCurrencies(currenciesJSON)
		if err != nil {
			return fmt.Errorf("decode role currencies for %s: %w", role.RoleID, err)
		}
		role.Currencies = currencies
		items, err := decodeRoleItems(itemsJSON)
		if err != nil {
			return fmt.Errorf("decode role items for %s: %w", role.RoleID, err)
		}
		role.Items = items
		roleState, err := decodeRoleState(roleStateJSON)
		if err != nil {
			return fmt.Errorf("decode role state for %s: %w", role.RoleID, err)
		}
		role.RoleState = roleState
		rolePhysique, err := decodeRolePhysique(rolePhysiqueJSON)
		if err != nil {
			return fmt.Errorf("decode role physique for %s: %w", role.RoleID, err)
		}
		role.RolePhysique = rolePhysique
		dungeonInstances, err := decodeDungeonInstances(dungeonInstancesJSON)
		if err != nil {
			return fmt.Errorf("decode dungeon instances for %s: %w", role.RoleID, err)
		}
		role.DungeonInstances = dungeonInstances
		store.rolesByPID[playerID] = append(store.rolesByPID[playerID], role)
	}
	if err := roleRows.Err(); err != nil {
		return fmt.Errorf("iterate roles: %w", err)
	}

	removedQuestRows, err := store.db.Query(`
		SELECT role_id, title
		FROM role_removed_quests
	`)
	if err != nil {
		return fmt.Errorf("query role removed quests: %w", err)
	}
	defer removedQuestRows.Close()

	for removedQuestRows.Next() {
		var roleID string
		var title string
		if err := removedQuestRows.Scan(&roleID, &title); err != nil {
			return fmt.Errorf("scan role removed quest: %w", err)
		}
		roleID = strings.TrimSpace(roleID)
		title = strings.TrimSpace(title)
		if roleID == "" || title == "" {
			continue
		}
		if store.removedQuests[roleID] == nil {
			store.removedQuests[roleID] = make(map[string]bool)
		}
		store.removedQuests[roleID][title] = true
	}
	if err := removedQuestRows.Err(); err != nil {
		return fmt.Errorf("iterate role removed quests: %w", err)
	}

	acceptedQuestRows, err := store.db.Query(`
		SELECT role_id, title
		FROM role_accepted_quests
	`)
	if err != nil {
		return fmt.Errorf("query role accepted quests: %w", err)
	}
	defer acceptedQuestRows.Close()

	for acceptedQuestRows.Next() {
		var roleID string
		var title string
		if err := acceptedQuestRows.Scan(&roleID, &title); err != nil {
			return fmt.Errorf("scan role accepted quest: %w", err)
		}
		roleID = strings.TrimSpace(roleID)
		title = strings.TrimSpace(title)
		if roleID == "" || title == "" {
			continue
		}
		if store.acceptedQuests[roleID] == nil {
			store.acceptedQuests[roleID] = make(map[string]bool)
		}
		store.acceptedQuests[roleID][title] = true
	}
	if err := acceptedQuestRows.Err(); err != nil {
		return fmt.Errorf("iterate role accepted quests: %w", err)
	}

	return nil
}

func (store *Store) persistAcceptedQuestLocked(playerID string, roleID string, title string) error {
	if store.db == nil {
		return nil
	}

	_, err := store.db.Exec(
		`INSERT INTO role_accepted_quests (role_id, title, player_id)
		 VALUES (?, ?, ?)
		 ON CONFLICT(role_id, title) DO UPDATE SET player_id = excluded.player_id`,
		roleID,
		title,
		playerID,
	)
	if err != nil {
		return fmt.Errorf("upsert accepted quest roleId=%s title=%s: %w", roleID, title, err)
	}
	return nil
}

func (store *Store) deleteAcceptedQuestLocked(roleID string, title string) error {
	if store.db == nil {
		return nil
	}

	if _, err := store.db.Exec(`DELETE FROM role_accepted_quests WHERE role_id = ? AND title = ?`, roleID, title); err != nil {
		return fmt.Errorf("delete accepted quest roleId=%s title=%s: %w", roleID, title, err)
	}
	return nil
}

func (store *Store) persistAccountLocked(account AccountRecord) error {
	if store.db == nil {
		return nil
	}

	_, err := store.db.Exec(
		`INSERT INTO accounts (user_name, password, player_id, display_name, session_token)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(user_name) DO UPDATE SET
		   password = excluded.password,
		   player_id = excluded.player_id,
		   display_name = excluded.display_name,
		   session_token = excluded.session_token`,
		account.UserName,
		account.Password,
		account.PlayerID,
		account.DisplayName,
		account.SessionToken,
	)
	if err != nil {
		return fmt.Errorf("upsert account %s: %w", account.UserName, err)
	}

	return nil
}

func (store *Store) persistRemovedQuestLocked(playerID string, roleID string, title string) error {
	if store.db == nil {
		return nil
	}

	_, err := store.db.Exec(
		`INSERT INTO role_removed_quests (role_id, title, player_id)
		 VALUES (?, ?, ?)
		 ON CONFLICT(role_id, title) DO UPDATE SET player_id = excluded.player_id`,
		roleID,
		title,
		playerID,
	)
	if err != nil {
		return fmt.Errorf("upsert removed quest roleId=%s title=%s: %w", roleID, title, err)
	}
	return nil
}

func (store *Store) deleteRemovedQuestLocked(roleID string, title string) error {
	if store.db == nil {
		return nil
	}

	if _, err := store.db.Exec(`DELETE FROM role_removed_quests WHERE role_id = ? AND title = ?`, roleID, title); err != nil {
		return fmt.Errorf("delete removed quest roleId=%s title=%s: %w", roleID, title, err)
	}
	return nil
}

func (store *Store) persistPlayerStateLocked(playerID string) error {
	if store.db == nil {
		return nil
	}

	tx, err := store.db.Begin()
	if err != nil {
		return fmt.Errorf("begin sqlite transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	nextRoleSeq := store.nextRoleSeqByPID[playerID]
	if _, err = tx.Exec(
		`INSERT INTO role_sequences (player_id, next_role_seq)
		 VALUES (?, ?)
		 ON CONFLICT(player_id) DO UPDATE SET next_role_seq = excluded.next_role_seq`,
		playerID,
		nextRoleSeq,
	); err != nil {
		return fmt.Errorf("upsert role sequence for %s: %w", playerID, err)
	}

	if _, err = tx.Exec(`DELETE FROM roles WHERE player_id = ?`, playerID); err != nil {
		return fmt.Errorf("reset roles for %s: %w", playerID, err)
	}

	for _, role := range store.rolesByPID[playerID] {
		runtimeRole := withRoleRuntimeDefaults(role)
		appearanceJSON, encodeErr := encodeRoleAppearance(runtimeRole.Appearance)
		if encodeErr != nil {
			return fmt.Errorf("encode role appearance for %s: %w", role.RoleID, encodeErr)
		}
		skillsJSON, encodeErr := encodeRoleSkills(runtimeRole.Skills)
		if encodeErr != nil {
			return fmt.Errorf("encode role skills for %s: %w", role.RoleID, encodeErr)
		}
		fastPanelJSON, encodeErr := encodeRoleFastPanel(runtimeRole.FastPanel)
		if encodeErr != nil {
			return fmt.Errorf("encode role fast panel for %s: %w", role.RoleID, encodeErr)
		}
		currenciesJSON, encodeErr := encodeRoleCurrencies(runtimeRole.Currencies)
		if encodeErr != nil {
			return fmt.Errorf("encode role currencies for %s: %w", role.RoleID, encodeErr)
		}
		itemsJSON, encodeErr := encodeRoleItems(runtimeRole.Items)
		if encodeErr != nil {
			return fmt.Errorf("encode role items for %s: %w", role.RoleID, encodeErr)
		}
		roleStateJSON, encodeErr := encodeRoleState(runtimeRole.RoleState)
		if encodeErr != nil {
			return fmt.Errorf("encode role state for %s: %w", role.RoleID, encodeErr)
		}
		rolePhysiqueJSON, encodeErr := encodeRolePhysique(runtimeRole.RolePhysique)
		if encodeErr != nil {
			return fmt.Errorf("encode role physique for %s: %w", role.RoleID, encodeErr)
		}
		dungeonInstancesJSON, encodeErr := encodeDungeonInstances(runtimeRole.DungeonInstances)
		if encodeErr != nil {
			return fmt.Errorf("encode dungeon instances for %s: %w", role.RoleID, encodeErr)
		}
		if _, err = tx.Exec(
			`INSERT INTO roles (role_id, player_id, display_name, level, exp, voc, agi, str, intelligence, con, lck, map_id, visual_role_id, preset_id, source_query, battle_source_query, appearance_json, skills_json, fast_panel_json, currencies_json, items_json, role_state_json, role_physique_json, dungeon_instances_json)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			runtimeRole.RoleID,
			playerID,
			runtimeRole.DisplayName,
			runtimeRole.Level,
			runtimeRole.Exp,
			runtimeRole.Voc,
			runtimeRole.AGI,
			runtimeRole.STR,
			runtimeRole.INT,
			runtimeRole.CON,
			runtimeRole.LCK,
			runtimeRole.MapID,
			runtimeRole.VisualRoleID,
			runtimeRole.PresetID,
			runtimeRole.SourceQuery,
			runtimeRole.BattleSourceQuery,
			appearanceJSON,
			skillsJSON,
			fastPanelJSON,
			currenciesJSON,
			itemsJSON,
			roleStateJSON,
			rolePhysiqueJSON,
			dungeonInstancesJSON,
		); err != nil {
			return fmt.Errorf("insert role %s: %w", role.RoleID, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit player state transaction: %w", err)
	}

	return nil
}

func (store *Store) saveLocked() error {
	if store.db == nil {
		return nil
	}

	tx, err := store.db.Begin()
	if err != nil {
		return fmt.Errorf("begin sqlite transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.Exec(`DELETE FROM accounts`); err != nil {
		return fmt.Errorf("reset accounts table: %w", err)
	}
	for _, account := range store.accountsByName {
		if _, err = tx.Exec(
			`INSERT INTO accounts (user_name, password, player_id, display_name, session_token) VALUES (?, ?, ?, ?, ?)`,
			account.UserName,
			account.Password,
			account.PlayerID,
			account.DisplayName,
			account.SessionToken,
		); err != nil {
			return fmt.Errorf("insert account %s: %w", account.UserName, err)
		}
	}

	if _, err = tx.Exec(`DELETE FROM role_sequences`); err != nil {
		return fmt.Errorf("reset role_sequences table: %w", err)
	}
	for playerID, nextRoleSeq := range store.nextRoleSeqByPID {
		if _, err = tx.Exec(
			`INSERT INTO role_sequences (player_id, next_role_seq) VALUES (?, ?)`,
			playerID,
			nextRoleSeq,
		); err != nil {
			return fmt.Errorf("insert role sequence for %s: %w", playerID, err)
		}
	}

	if _, err = tx.Exec(`DELETE FROM roles`); err != nil {
		return fmt.Errorf("reset roles table: %w", err)
	}
	for playerID, roles := range store.rolesByPID {
		for _, role := range roles {
			runtimeRole := withRoleRuntimeDefaults(role)
			appearanceJSON, encodeErr := encodeRoleAppearance(runtimeRole.Appearance)
			if encodeErr != nil {
				return fmt.Errorf("encode role appearance for %s: %w", role.RoleID, encodeErr)
			}
			skillsJSON, encodeErr := encodeRoleSkills(runtimeRole.Skills)
			if encodeErr != nil {
				return fmt.Errorf("encode role skills for %s: %w", role.RoleID, encodeErr)
			}
			fastPanelJSON, encodeErr := encodeRoleFastPanel(runtimeRole.FastPanel)
			if encodeErr != nil {
				return fmt.Errorf("encode role fast panel for %s: %w", role.RoleID, encodeErr)
			}
			currenciesJSON, encodeErr := encodeRoleCurrencies(runtimeRole.Currencies)
			if encodeErr != nil {
				return fmt.Errorf("encode role currencies for %s: %w", role.RoleID, encodeErr)
			}
			itemsJSON, encodeErr := encodeRoleItems(runtimeRole.Items)
			if encodeErr != nil {
				return fmt.Errorf("encode role items for %s: %w", role.RoleID, encodeErr)
			}
			roleStateJSON, encodeErr := encodeRoleState(runtimeRole.RoleState)
			if encodeErr != nil {
				return fmt.Errorf("encode role state for %s: %w", role.RoleID, encodeErr)
			}
			rolePhysiqueJSON, encodeErr := encodeRolePhysique(runtimeRole.RolePhysique)
			if encodeErr != nil {
				return fmt.Errorf("encode role physique for %s: %w", role.RoleID, encodeErr)
			}
			dungeonInstancesJSON, encodeErr := encodeDungeonInstances(runtimeRole.DungeonInstances)
			if encodeErr != nil {
				return fmt.Errorf("encode dungeon instances for %s: %w", role.RoleID, encodeErr)
			}
			if _, err = tx.Exec(
				`INSERT INTO roles (role_id, player_id, display_name, level, exp, voc, agi, str, intelligence, con, lck, map_id, visual_role_id, preset_id, source_query, battle_source_query, appearance_json, skills_json, fast_panel_json, currencies_json, items_json, role_state_json, role_physique_json, dungeon_instances_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				runtimeRole.RoleID,
				playerID,
				runtimeRole.DisplayName,
				runtimeRole.Level,
				runtimeRole.Exp,
				runtimeRole.Voc,
				runtimeRole.AGI,
				runtimeRole.STR,
				runtimeRole.INT,
				runtimeRole.CON,
				runtimeRole.LCK,
				runtimeRole.MapID,
				runtimeRole.VisualRoleID,
				runtimeRole.PresetID,
				runtimeRole.SourceQuery,
				runtimeRole.BattleSourceQuery,
				appearanceJSON,
				skillsJSON,
				fastPanelJSON,
				currenciesJSON,
				itemsJSON,
				roleStateJSON,
				rolePhysiqueJSON,
				dungeonInstancesJSON,
			); err != nil {
				return fmt.Errorf("insert role %s: %w", role.RoleID, err)
			}
		}
	}

	if _, err = tx.Exec(`DELETE FROM role_accepted_quests`); err != nil {
		return fmt.Errorf("reset role_accepted_quests table: %w", err)
	}
	if _, err = tx.Exec(`DELETE FROM role_removed_quests`); err != nil {
		return fmt.Errorf("reset role_removed_quests table: %w", err)
	}
	for playerID, roles := range store.rolesByPID {
		for _, role := range roles {
			for title, accepted := range store.acceptedQuests[role.RoleID] {
				if !accepted {
					continue
				}
				if _, err = tx.Exec(
					`INSERT INTO role_accepted_quests (role_id, title, player_id) VALUES (?, ?, ?)`,
					role.RoleID,
					title,
					playerID,
				); err != nil {
					return fmt.Errorf("insert accepted quest roleId=%s title=%s: %w", role.RoleID, title, err)
				}
			}
			for title, removed := range store.removedQuests[role.RoleID] {
				if !removed {
					continue
				}
				if _, err = tx.Exec(
					`INSERT INTO role_removed_quests (role_id, title, player_id) VALUES (?, ?, ?)`,
					role.RoleID,
					title,
					playerID,
				); err != nil {
					return fmt.Errorf("insert removed quest roleId=%s title=%s: %w", role.RoleID, title, err)
				}
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit sqlite transaction: %w", err)
	}

	return nil
}
