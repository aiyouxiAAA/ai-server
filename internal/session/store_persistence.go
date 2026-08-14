package session

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type rolePersistenceExecer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

type rolePersistencePayload struct {
	runtimeRole             RoleSummary
	appearanceJSON          string
	skillsJSON              string
	fastPanelJSON           string
	currenciesJSON          string
	containerCapacitiesJSON string
	itemsJSON               string
	townBuffsJSON           string
	roleStateJSON           string
	rolePhysiqueJSON        string
	dungeonInstancesJSON    string
}

type persistedRoleItemRepair struct {
	playerID string
	roleID   string
	items    []RoleItem
}

func configureSQLitePersistence(db *sql.DB) error {
	if db == nil {
		return nil
	}
	if _, err := db.Exec(`PRAGMA busy_timeout = 5000`); err != nil {
		return fmt.Errorf("configure sqlite busy_timeout: %w", err)
	}
	journalMode := ""
	if err := db.QueryRow(`PRAGMA journal_mode = WAL`).Scan(&journalMode); err != nil {
		return fmt.Errorf("configure sqlite journal_mode: %w", err)
	}
	if !strings.EqualFold(journalMode, "wal") {
		return fmt.Errorf("configure sqlite journal_mode: expected wal, got %s", journalMode)
	}
	if _, err := db.Exec(`PRAGMA synchronous = NORMAL`); err != nil {
		return fmt.Errorf("configure sqlite synchronous: %w", err)
	}
	if _, err := db.Exec(`PRAGMA wal_autocheckpoint = 1000`); err != nil {
		return fmt.Errorf("configure sqlite wal_autocheckpoint: %w", err)
	}
	return nil
}

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
			map_x INTEGER NOT NULL DEFAULT 0,
			map_y INTEGER NOT NULL DEFAULT 0,
			visual_role_id INTEGER NOT NULL,
			preset_id INTEGER NOT NULL DEFAULT 0,
			source_query TEXT NOT NULL DEFAULT '',
			battle_source_query TEXT NOT NULL DEFAULT '',
			appearance_json TEXT NOT NULL DEFAULT '',
			skills_json TEXT NOT NULL DEFAULT '',
			skill_cap INTEGER NOT NULL DEFAULT 12,
			fast_panel_json TEXT NOT NULL DEFAULT '',
			currencies_json TEXT NOT NULL DEFAULT '',
			container_capacities_json TEXT NOT NULL DEFAULT '',
			items_json TEXT NOT NULL DEFAULT '',
			town_buffs_json TEXT NOT NULL DEFAULT '',
			role_state_json TEXT NOT NULL DEFAULT '',
			role_physique_json TEXT NOT NULL DEFAULT '',
			dungeon_instances_json TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS role_snapshot_migrations (
			role_id TEXT NOT NULL,
			migration_key TEXT NOT NULL,
			PRIMARY KEY (role_id, migration_key)
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
		`CREATE TABLE IF NOT EXISTS role_quest_progress (
			role_id TEXT NOT NULL,
			title TEXT NOT NULL,
			player_id TEXT NOT NULL DEFAULT '',
			current INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (role_id, title)
		)`,
		`CREATE TABLE IF NOT EXISTS role_quest_objectives (
			role_id TEXT NOT NULL,
			title TEXT NOT NULL,
			player_id TEXT NOT NULL DEFAULT '',
			objectives_json TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (role_id, title)
		)`,
		`CREATE TABLE IF NOT EXISTS role_item_acquisitions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			occurred_at_unix_ms INTEGER NOT NULL,
			player_id TEXT NOT NULL,
			role_id TEXT NOT NULL,
			item_name TEXT NOT NULL,
			item_type TEXT NOT NULL DEFAULT '',
			container_type TEXT NOT NULL DEFAULT '',
			item_display TEXT NOT NULL DEFAULT '',
			item_description TEXT NOT NULL DEFAULT '',
			item_level INTEGER NOT NULL DEFAULT 0,
			quantity INTEGER NOT NULL,
			source TEXT NOT NULL,
			source_detail TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE INDEX IF NOT EXISTS idx_role_item_acquisitions_role_time
			ON role_item_acquisitions (player_id, role_id, occurred_at_unix_ms DESC, id DESC)`,
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
	if _, err := store.db.Exec(`ALTER TABLE roles ADD COLUMN skill_cap INTEGER NOT NULL DEFAULT 12`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			return fmt.Errorf("migrate roles skill_cap column: %w", err)
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
	if _, err := store.db.Exec(`ALTER TABLE roles ADD COLUMN container_capacities_json TEXT NOT NULL DEFAULT ''`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			return fmt.Errorf("migrate roles container_capacities_json column: %w", err)
		}
	}
	if _, err := store.db.Exec(`ALTER TABLE roles ADD COLUMN items_json TEXT NOT NULL DEFAULT ''`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			return fmt.Errorf("migrate roles items_json column: %w", err)
		}
	}
	if _, err := store.db.Exec(`ALTER TABLE roles ADD COLUMN town_buffs_json TEXT NOT NULL DEFAULT ''`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			return fmt.Errorf("migrate roles town_buffs_json column: %w", err)
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
		{column: "map_x", query: `ALTER TABLE roles ADD COLUMN map_x INTEGER NOT NULL DEFAULT 0`},
		{column: "map_y", query: `ALTER TABLE roles ADD COLUMN map_y INTEGER NOT NULL DEFAULT 0`},
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
		SELECT role_id, player_id, display_name, level, exp, voc, agi, str, intelligence, con, lck, map_id, map_x, map_y, visual_role_id, preset_id, source_query, battle_source_query, appearance_json, skills_json, skill_cap, fast_panel_json, currencies_json, container_capacities_json, items_json, town_buffs_json, role_state_json, role_physique_json, dungeon_instances_json
		FROM roles
		ORDER BY player_id ASC, role_id ASC
	`)
	if err != nil {
		return fmt.Errorf("query roles: %w", err)
	}
	defer roleRows.Close()

	itemRepairs := []persistedRoleItemRepair{}
	for roleRows.Next() {
		var role RoleSummary
		var playerID string
		var appearanceJSON string
		var skillsJSON string
		var fastPanelJSON string
		var currenciesJSON string
		var containerCapacitiesJSON string
		var itemsJSON string
		var townBuffsJSON string
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
			&role.MapX,
			&role.MapY,
			&role.VisualRoleID,
			&role.PresetID,
			&role.SourceQuery,
			&role.BattleSourceQuery,
			&appearanceJSON,
			&skillsJSON,
			&role.SkillCap,
			&fastPanelJSON,
			&currenciesJSON,
			&containerCapacitiesJSON,
			&itemsJSON,
			&townBuffsJSON,
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
		containerCapacities, err := decodeRoleContainerCapacities(containerCapacitiesJSON)
		if err != nil {
			return fmt.Errorf("decode role container capacities for %s: %w", role.RoleID, err)
		}
		role.ContainerCapacities = containerCapacities
		var persistedItems []RoleItem
		if strings.TrimSpace(itemsJSON) != "" {
			if err := json.Unmarshal([]byte(itemsJSON), &persistedItems); err != nil {
				return fmt.Errorf("decode persisted role items for %s: %w", role.RoleID, err)
			}
		}
		itemsNeedRepair := roleItemsNeedEquipmentTemplateRefinementLineBreakRepair(persistedItems)
		items, err := decodeRoleItems(itemsJSON)
		if err != nil {
			return fmt.Errorf("decode role items for %s: %w", role.RoleID, err)
		}
		role.Items = items
		if itemsNeedRepair {
			itemRepairs = append(itemRepairs, persistedRoleItemRepair{
				playerID: playerID,
				roleID:   role.RoleID,
				items:    items,
			})
		}
		townBuffs, err := decodeRoleTownBuffs(townBuffsJSON)
		if err != nil {
			return fmt.Errorf("decode role town buffs for %s: %w", role.RoleID, err)
		}
		role.TownBuffs = townBuffs
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
	if err := roleRows.Close(); err != nil {
		return fmt.Errorf("close roles query: %w", err)
	}
	for _, repair := range itemRepairs {
		itemsJSON, err := encodeRoleItems(repair.items)
		if err != nil {
			return fmt.Errorf("encode repaired role items for %s: %w", repair.roleID, err)
		}
		if _, err := store.db.Exec(`UPDATE roles SET items_json = ? WHERE role_id = ? AND player_id = ?`, itemsJSON, repair.roleID, repair.playerID); err != nil {
			return fmt.Errorf("persist repaired role items for %s: %w", repair.roleID, err)
		}
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

	questProgressRows, err := store.db.Query(`
		SELECT role_id, title, current
		FROM role_quest_progress
	`)
	if err != nil {
		return fmt.Errorf("query role quest progress: %w", err)
	}
	defer questProgressRows.Close()

	for questProgressRows.Next() {
		var roleID string
		var title string
		var current int
		if err := questProgressRows.Scan(&roleID, &title, &current); err != nil {
			return fmt.Errorf("scan role quest progress: %w", err)
		}
		roleID = strings.TrimSpace(roleID)
		title = strings.TrimSpace(title)
		if roleID == "" || title == "" || current < 0 {
			continue
		}
		if store.questProgress[roleID] == nil {
			store.questProgress[roleID] = make(map[string]int)
		}
		store.questProgress[roleID][title] = current
	}
	if err := questProgressRows.Err(); err != nil {
		return fmt.Errorf("iterate role quest progress: %w", err)
	}

	questObjectiveRows, err := store.db.Query(`
		SELECT role_id, title, objectives_json
		FROM role_quest_objectives
	`)
	if err != nil {
		return fmt.Errorf("query role quest objectives: %w", err)
	}
	defer questObjectiveRows.Close()

	for questObjectiveRows.Next() {
		var roleID string
		var title string
		var objectivesJSON string
		if err := questObjectiveRows.Scan(&roleID, &title, &objectivesJSON); err != nil {
			return fmt.Errorf("scan role quest objectives: %w", err)
		}
		roleID = strings.TrimSpace(roleID)
		title = strings.TrimSpace(title)
		if roleID == "" || title == "" || strings.TrimSpace(objectivesJSON) == "" {
			continue
		}
		var objectives []RoleQuestObjectiveProgress
		if err := json.Unmarshal([]byte(objectivesJSON), &objectives); err != nil {
			return fmt.Errorf("decode role quest objectives for %s/%s: %w", roleID, title, err)
		}
		if store.questObjectives[roleID] == nil {
			store.questObjectives[roleID] = make(map[string][]RoleQuestObjectiveProgress)
		}
		store.questObjectives[roleID][title] = cloneRoleQuestObjectiveProgress(objectives)
	}
	if err := questObjectiveRows.Err(); err != nil {
		return fmt.Errorf("iterate role quest objectives: %w", err)
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

func (store *Store) persistQuestProgressLocked(playerID string, roleID string, title string, current int) error {
	if store.db == nil {
		return nil
	}

	_, err := store.db.Exec(
		`INSERT INTO role_quest_progress (role_id, title, player_id, current)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(role_id, title) DO UPDATE SET
		   player_id = excluded.player_id,
		   current = excluded.current`,
		roleID,
		title,
		playerID,
		current,
	)
	if err != nil {
		return fmt.Errorf("upsert quest progress roleId=%s title=%s: %w", roleID, title, err)
	}
	return nil
}

func (store *Store) deleteQuestProgressLocked(roleID string, title string) error {
	if store.db == nil {
		return nil
	}

	if _, err := store.db.Exec(`DELETE FROM role_quest_progress WHERE role_id = ? AND title = ?`, roleID, title); err != nil {
		return fmt.Errorf("delete quest progress roleId=%s title=%s: %w", roleID, title, err)
	}
	return nil
}

func (store *Store) persistQuestObjectivesLocked(playerID string, roleID string, title string, objectives []RoleQuestObjectiveProgress) error {
	if store.db == nil {
		return nil
	}
	objectivesJSON, err := json.Marshal(cloneRoleQuestObjectiveProgress(objectives))
	if err != nil {
		return fmt.Errorf("encode quest objectives roleId=%s title=%s: %w", roleID, title, err)
	}
	_, err = store.db.Exec(
		`INSERT INTO role_quest_objectives (role_id, title, player_id, objectives_json)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(role_id, title) DO UPDATE SET
		   player_id = excluded.player_id,
		   objectives_json = excluded.objectives_json`,
		roleID,
		title,
		playerID,
		string(objectivesJSON),
	)
	if err != nil {
		return fmt.Errorf("upsert quest objectives roleId=%s title=%s: %w", roleID, title, err)
	}
	return nil
}

func (store *Store) deleteQuestObjectivesLocked(roleID string, title string) error {
	if store.db == nil {
		return nil
	}
	if _, err := store.db.Exec(`DELETE FROM role_quest_objectives WHERE role_id = ? AND title = ?`, roleID, title); err != nil {
		return fmt.Errorf("delete quest objectives roleId=%s title=%s: %w", roleID, title, err)
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
		payload, encodeErr := buildRolePersistencePayload(role)
		if encodeErr != nil {
			return encodeErr
		}
		if err = upsertRolePersistencePayload(tx, playerID, payload); err != nil {
			return fmt.Errorf("insert role %s: %w", role.RoleID, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit player state transaction: %w", err)
	}

	return nil
}

func (store *Store) persistPlayerStateWithItemAcquisitionsLocked(playerID string, acquisitions []RoleItemAcquisition) error {
	if store.db == nil {
		return nil
	}

	tx, err := store.db.Begin()
	if err != nil {
		return fmt.Errorf("begin player acquisition transaction: %w", err)
	}
	defer tx.Rollback()

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
		payload, encodeErr := buildRolePersistencePayload(role)
		if encodeErr != nil {
			return encodeErr
		}
		if err = upsertRolePersistencePayload(tx, playerID, payload); err != nil {
			return fmt.Errorf("insert role %s: %w", role.RoleID, err)
		}
	}
	for _, acquisition := range acquisitions {
		if err = insertRoleItemAcquisition(tx, acquisition); err != nil {
			return err
		}
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit player acquisition transaction: %w", err)
	}
	return nil
}

func (store *Store) persistRoleStateLocked(playerID string, roleID string) error {
	if store.db == nil {
		return nil
	}

	for _, role := range store.rolesByPID[playerID] {
		if role.RoleID != roleID {
			continue
		}
		return store.persistRoleStateSnapshot(playerID, roleID, role)
	}
	return fmt.Errorf("role %s missing for player %s", roleID, playerID)
}

func (store *Store) persistRoleStateSnapshot(playerID string, roleID string, role RoleSummary) error {
	if store.db == nil {
		return nil
	}
	if role.RoleID != roleID {
		return fmt.Errorf("role snapshot mismatch for player %s: got %s want %s", playerID, role.RoleID, roleID)
	}
	payload, err := buildRolePersistencePayload(role)
	if err != nil {
		return err
	}
	if err := upsertRolePersistencePayload(store.db, playerID, payload); err != nil {
		return fmt.Errorf("upsert role %s: %w", role.RoleID, err)
	}
	return nil
}

func (store *Store) persistRoleStateSnapshotWithItemAcquisitions(
	playerID string,
	roleID string,
	role RoleSummary,
	acquisitions []RoleItemAcquisition,
) error {
	if store.db == nil {
		return nil
	}
	if role.RoleID != roleID {
		return fmt.Errorf("role snapshot mismatch for player %s: got %s want %s", playerID, role.RoleID, roleID)
	}
	payload, err := buildRolePersistencePayload(role)
	if err != nil {
		return err
	}
	tx, err := store.db.Begin()
	if err != nil {
		return fmt.Errorf("begin role acquisition transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if err = upsertRolePersistencePayload(tx, playerID, payload); err != nil {
		return fmt.Errorf("upsert role %s: %w", roleID, err)
	}
	for _, acquisition := range acquisitions {
		if err = insertRoleItemAcquisition(tx, acquisition); err != nil {
			return err
		}
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit role acquisition transaction: %w", err)
	}
	return nil
}

func insertRoleItemAcquisition(execer rolePersistenceExecer, acquisition RoleItemAcquisition) error {
	if acquisition.Quantity <= 0 || strings.TrimSpace(acquisition.ItemName) == "" {
		return fmt.Errorf("invalid role item acquisition")
	}
	acquisition.Source = strings.TrimSpace(acquisition.Source)
	if acquisition.Source == "" {
		acquisition.Source = "系统发放"
	}
	if acquisition.OccurredAtUnixMs <= 0 {
		acquisition.OccurredAtUnixMs = time.Now().UnixMilli()
	}
	_, err := execer.Exec(
		`INSERT INTO role_item_acquisitions (
			occurred_at_unix_ms, player_id, role_id, item_name, item_type,
			container_type, item_display, item_description, item_level, quantity,
			source, source_detail
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		acquisition.OccurredAtUnixMs,
		acquisition.PlayerID,
		acquisition.RoleID,
		acquisition.ItemName,
		acquisition.ItemType,
		acquisition.ContainerType,
		acquisition.ItemDisplay,
		acquisition.ItemDescription,
		acquisition.ItemLevel,
		acquisition.Quantity,
		acquisition.Source,
		acquisition.SourceDetail,
	)
	if err != nil {
		return fmt.Errorf("insert role item acquisition roleId=%s item=%s: %w", acquisition.RoleID, acquisition.ItemName, err)
	}
	return nil
}

func (store *Store) ListRoleItemAcquisitions(playerID string, roleID string, limit int) ([]RoleItemAcquisition, bool) {
	if store.db == nil || strings.TrimSpace(playerID) == "" || strings.TrimSpace(roleID) == "" {
		return []RoleItemAcquisition{}, false
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := store.db.Query(
		`SELECT id, occurred_at_unix_ms, player_id, role_id, item_name, item_type,
			container_type, item_display, item_description, item_level, quantity,
			source, source_detail
		 FROM role_item_acquisitions
		 WHERE player_id = ? AND role_id = ?
		 ORDER BY occurred_at_unix_ms DESC, id DESC
		 LIMIT ?`,
		playerID,
		roleID,
		limit,
	)
	if err != nil {
		return []RoleItemAcquisition{}, false
	}
	defer rows.Close()
	result := []RoleItemAcquisition{}
	for rows.Next() {
		var acquisition RoleItemAcquisition
		if err := rows.Scan(
			&acquisition.ID,
			&acquisition.OccurredAtUnixMs,
			&acquisition.PlayerID,
			&acquisition.RoleID,
			&acquisition.ItemName,
			&acquisition.ItemType,
			&acquisition.ContainerType,
			&acquisition.ItemDisplay,
			&acquisition.ItemDescription,
			&acquisition.ItemLevel,
			&acquisition.Quantity,
			&acquisition.Source,
			&acquisition.SourceDetail,
		); err != nil {
			return []RoleItemAcquisition{}, false
		}
		result = append(result, acquisition)
	}
	if err := rows.Err(); err != nil {
		return []RoleItemAcquisition{}, false
	}
	return result, true
}

func (store *Store) persistRoleItemsLocked(playerID string, roleID string) error {
	if store.db == nil {
		return nil
	}

	for _, role := range store.rolesByPID[playerID] {
		if role.RoleID != roleID {
			continue
		}
		itemsJSON, err := encodeRoleItems(role.Items)
		if err != nil {
			return fmt.Errorf("encode role items for %s: %w", role.RoleID, err)
		}
		result, err := store.db.Exec(`UPDATE roles SET items_json = ? WHERE role_id = ? AND player_id = ?`, itemsJSON, roleID, playerID)
		if err != nil {
			return fmt.Errorf("update role items for %s: %w", roleID, err)
		}
		if affected, affectedErr := result.RowsAffected(); affectedErr == nil && affected == 0 {
			return store.persistRoleStateLocked(playerID, roleID)
		}
		return nil
	}
	return fmt.Errorf("role %s missing for player %s", roleID, playerID)
}

func (store *Store) persistRoleItemsSnapshot(playerID string, roleID string, itemsJSON string, fallbackRole RoleSummary) error {
	if store.db == nil {
		return nil
	}

	result, err := store.db.Exec(`UPDATE roles SET items_json = ? WHERE role_id = ? AND player_id = ?`, itemsJSON, roleID, playerID)
	if err != nil {
		return fmt.Errorf("update role items for %s: %w", roleID, err)
	}
	if affected, affectedErr := result.RowsAffected(); affectedErr == nil && affected == 0 {
		payload, payloadErr := buildRolePersistencePayload(fallbackRole)
		if payloadErr != nil {
			return payloadErr
		}
		if upsertErr := upsertRolePersistencePayload(store.db, playerID, payload); upsertErr != nil {
			return fmt.Errorf("upsert role %s after missing item row: %w", roleID, upsertErr)
		}
	}
	return nil
}

func buildRolePersistencePayload(role RoleSummary) (rolePersistencePayload, error) {
	runtimeRole := withRoleRuntimeDefaults(role)
	appearanceJSON, err := encodeRoleAppearance(runtimeRole.Appearance)
	if err != nil {
		return rolePersistencePayload{}, fmt.Errorf("encode role appearance for %s: %w", role.RoleID, err)
	}
	skillsJSON, err := encodeRoleSkills(runtimeRole.Skills)
	if err != nil {
		return rolePersistencePayload{}, fmt.Errorf("encode role skills for %s: %w", role.RoleID, err)
	}
	fastPanelJSON, err := encodeRoleFastPanel(runtimeRole.FastPanel)
	if err != nil {
		return rolePersistencePayload{}, fmt.Errorf("encode role fast panel for %s: %w", role.RoleID, err)
	}
	currenciesJSON, err := encodeRoleCurrencies(runtimeRole.Currencies)
	if err != nil {
		return rolePersistencePayload{}, fmt.Errorf("encode role currencies for %s: %w", role.RoleID, err)
	}
	containerCapacitiesJSON, err := encodeRoleContainerCapacities(runtimeRole.ContainerCapacities)
	if err != nil {
		return rolePersistencePayload{}, fmt.Errorf("encode role container capacities for %s: %w", role.RoleID, err)
	}
	itemsJSON, err := encodeRoleItems(runtimeRole.Items)
	if err != nil {
		return rolePersistencePayload{}, fmt.Errorf("encode role items for %s: %w", role.RoleID, err)
	}
	townBuffsJSON, err := encodeRoleTownBuffs(runtimeRole.TownBuffs)
	if err != nil {
		return rolePersistencePayload{}, fmt.Errorf("encode role town buffs for %s: %w", role.RoleID, err)
	}
	roleStateJSON, err := encodeRoleState(runtimeRole.RoleState)
	if err != nil {
		return rolePersistencePayload{}, fmt.Errorf("encode role state for %s: %w", role.RoleID, err)
	}
	rolePhysiqueJSON, err := encodeRolePhysique(runtimeRole.RolePhysique)
	if err != nil {
		return rolePersistencePayload{}, fmt.Errorf("encode role physique for %s: %w", role.RoleID, err)
	}
	dungeonInstancesJSON, err := encodeDungeonInstances(runtimeRole.DungeonInstances)
	if err != nil {
		return rolePersistencePayload{}, fmt.Errorf("encode dungeon instances for %s: %w", role.RoleID, err)
	}
	return rolePersistencePayload{
		runtimeRole:             runtimeRole,
		appearanceJSON:          appearanceJSON,
		skillsJSON:              skillsJSON,
		fastPanelJSON:           fastPanelJSON,
		currenciesJSON:          currenciesJSON,
		containerCapacitiesJSON: containerCapacitiesJSON,
		itemsJSON:               itemsJSON,
		townBuffsJSON:           townBuffsJSON,
		roleStateJSON:           roleStateJSON,
		rolePhysiqueJSON:        rolePhysiqueJSON,
		dungeonInstancesJSON:    dungeonInstancesJSON,
	}, nil
}

func upsertRolePersistencePayload(execer rolePersistenceExecer, playerID string, payload rolePersistencePayload) error {
	role := payload.runtimeRole
	_, err := execer.Exec(
		`INSERT INTO roles (role_id, player_id, display_name, level, exp, voc, agi, str, intelligence, con, lck, map_id, map_x, map_y, visual_role_id, preset_id, source_query, battle_source_query, appearance_json, skills_json, skill_cap, fast_panel_json, currencies_json, container_capacities_json, items_json, town_buffs_json, role_state_json, role_physique_json, dungeon_instances_json)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(role_id) DO UPDATE SET
		   player_id = excluded.player_id,
		   display_name = excluded.display_name,
		   level = excluded.level,
		   exp = excluded.exp,
		   voc = excluded.voc,
		   agi = excluded.agi,
		   str = excluded.str,
		   intelligence = excluded.intelligence,
		   con = excluded.con,
		   lck = excluded.lck,
		   map_id = excluded.map_id,
		   map_x = excluded.map_x,
		   map_y = excluded.map_y,
		   visual_role_id = excluded.visual_role_id,
		   preset_id = excluded.preset_id,
		   source_query = excluded.source_query,
		   battle_source_query = excluded.battle_source_query,
		   appearance_json = excluded.appearance_json,
		   skills_json = excluded.skills_json,
		   skill_cap = excluded.skill_cap,
		   fast_panel_json = excluded.fast_panel_json,
		   currencies_json = excluded.currencies_json,
		   container_capacities_json = excluded.container_capacities_json,
		   items_json = excluded.items_json,
		   town_buffs_json = excluded.town_buffs_json,
		   role_state_json = excluded.role_state_json,
		   role_physique_json = excluded.role_physique_json,
		   dungeon_instances_json = excluded.dungeon_instances_json`,
		role.RoleID,
		playerID,
		role.DisplayName,
		role.Level,
		role.Exp,
		role.Voc,
		role.AGI,
		role.STR,
		role.INT,
		role.CON,
		role.LCK,
		role.MapID,
		role.MapX,
		role.MapY,
		role.VisualRoleID,
		role.PresetID,
		role.SourceQuery,
		role.BattleSourceQuery,
		payload.appearanceJSON,
		payload.skillsJSON,
		role.SkillCap,
		payload.fastPanelJSON,
		payload.currenciesJSON,
		payload.containerCapacitiesJSON,
		payload.itemsJSON,
		payload.townBuffsJSON,
		payload.roleStateJSON,
		payload.rolePhysiqueJSON,
		payload.dungeonInstancesJSON,
	)
	return err
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
			containerCapacitiesJSON, encodeErr := encodeRoleContainerCapacities(runtimeRole.ContainerCapacities)
			if encodeErr != nil {
				return fmt.Errorf("encode role container capacities for %s: %w", role.RoleID, encodeErr)
			}
			itemsJSON, encodeErr := encodeRoleItems(runtimeRole.Items)
			if encodeErr != nil {
				return fmt.Errorf("encode role items for %s: %w", role.RoleID, encodeErr)
			}
			townBuffsJSON, encodeErr := encodeRoleTownBuffs(runtimeRole.TownBuffs)
			if encodeErr != nil {
				return fmt.Errorf("encode role town buffs for %s: %w", role.RoleID, encodeErr)
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
				`INSERT INTO roles (role_id, player_id, display_name, level, exp, voc, agi, str, intelligence, con, lck, map_id, map_x, map_y, visual_role_id, preset_id, source_query, battle_source_query, appearance_json, skills_json, skill_cap, fast_panel_json, currencies_json, container_capacities_json, items_json, town_buffs_json, role_state_json, role_physique_json, dungeon_instances_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
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
				runtimeRole.MapX,
				runtimeRole.MapY,
				runtimeRole.VisualRoleID,
				runtimeRole.PresetID,
				runtimeRole.SourceQuery,
				runtimeRole.BattleSourceQuery,
				appearanceJSON,
				skillsJSON,
				runtimeRole.SkillCap,
				fastPanelJSON,
				currenciesJSON,
				containerCapacitiesJSON,
				itemsJSON,
				townBuffsJSON,
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
	if _, err = tx.Exec(`DELETE FROM role_quest_progress`); err != nil {
		return fmt.Errorf("reset role_quest_progress table: %w", err)
	}
	if _, err = tx.Exec(`DELETE FROM role_quest_objectives`); err != nil {
		return fmt.Errorf("reset role_quest_objectives table: %w", err)
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
			for title, current := range store.questProgress[role.RoleID] {
				if !store.acceptedQuests[role.RoleID][title] || current < 0 {
					continue
				}
				if _, err = tx.Exec(
					`INSERT INTO role_quest_progress (role_id, title, player_id, current) VALUES (?, ?, ?, ?)`,
					role.RoleID,
					title,
					playerID,
					current,
				); err != nil {
					return fmt.Errorf("insert quest progress roleId=%s title=%s: %w", role.RoleID, title, err)
				}
			}
			for title, objectives := range store.questObjectives[role.RoleID] {
				if !store.acceptedQuests[role.RoleID][title] || len(objectives) == 0 {
					continue
				}
				objectivesJSON, encodeErr := json.Marshal(cloneRoleQuestObjectiveProgress(objectives))
				if encodeErr != nil {
					return fmt.Errorf("encode quest objectives roleId=%s title=%s: %w", role.RoleID, title, encodeErr)
				}
				if _, err = tx.Exec(
					`INSERT INTO role_quest_objectives (role_id, title, player_id, objectives_json) VALUES (?, ?, ?, ?)`,
					role.RoleID,
					title,
					playerID,
					string(objectivesJSON),
				); err != nil {
					return fmt.Errorf("insert quest objectives roleId=%s title=%s: %w", role.RoleID, title, err)
				}
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit sqlite transaction: %w", err)
	}

	return nil
}
