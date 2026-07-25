package session

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed captured_a_chai_role.json
var capturedAChaiRoleJSON []byte

type capturedAChaiRoleSnapshot struct {
	SourceCapture string `json:"sourceCapture"`
	Level50Stats  struct {
		SourceCapture      string       `json:"sourceCapture"`
		RoleInfoPacket     int          `json:"roleInfoPacket"`
		RoleStatePacket    int          `json:"roleStatePacket"`
		RolePhysiquePacket int          `json:"rolePhysiquePacket"`
		Level              int          `json:"level"`
		Exp                int          `json:"exp"`
		State              RoleState    `json:"state"`
		Physique           RolePhysique `json:"physique"`
	} `json:"level50Stats"`
	Role struct {
		Vocation          string                  `json:"vocation"`
		Level             int                     `json:"level"`
		Exp               int                     `json:"exp"`
		MapID             int                     `json:"mapId"`
		VisualRoleID      int                     `json:"visualRoleId"`
		SourceQuery       string                  `json:"sourceQuery"`
		BattleSourceQuery string                  `json:"battleSourceQuery"`
		Physique          RolePhysique            `json:"physique"`
		State             RoleState               `json:"state"`
		Capacities        RoleContainerCapacities `json:"capacities"`
		Currencies        RoleCurrencies          `json:"currencies"`
		Items             []RoleItem              `json:"items"`
		SkillCap          int                     `json:"skillCap"`
		Skills            []RoleSkill             `json:"skills"`
		FastPanel         []RoleFastPanelEntry    `json:"fastPanel"`
		TownBuffs         []RoleTownBuff          `json:"townBuffs"`
	} `json:"role"`
	Quests []capturedAChaiQuestSnapshot `json:"quests"`
}

type capturedAChaiQuestSnapshot struct {
	Name     string `json:"name"`
	Progress int    `json:"progress"`
	Target   int    `json:"target"`
}

var capturedAChaiSnapshot = mustLoadCapturedAChaiSnapshot()

const (
	capturedAChaiLevel50StatsAndSkillsMigrationKey = "captured-a-chai-555-level50-stats-skills-v1"
	capturedAChaiLevel50StatsAndSkillsRoleID       = "acct-55555555-role-001"
)

func capturedAChaiRoleItemTemplate(name string) (RoleItem, bool) {
	for _, item := range capturedAChaiSnapshot.Role.Items {
		if item.Name == name {
			return item, true
		}
	}
	return RoleItem{}, false
}

func mustLoadCapturedAChaiSnapshot() capturedAChaiRoleSnapshot {
	var snapshot capturedAChaiRoleSnapshot
	if err := json.Unmarshal(capturedAChaiRoleJSON, &snapshot); err != nil {
		panic(fmt.Sprintf("decode captured a chai role snapshot: %v", err))
	}
	return snapshot
}

func applyCapturedAChaiSnapshot(role RoleSummary) RoleSummary {
	snapshot := capturedAChaiSnapshot.Role
	role.DisplayName = "555"
	role.Voc = snapshot.Vocation
	role.Level = snapshot.Level
	role.Exp = snapshot.Exp
	role.MapID = snapshot.MapID
	role.VisualRoleID = snapshot.VisualRoleID
	role.SourceQuery = snapshot.SourceQuery
	role.BattleSourceQuery = snapshot.BattleSourceQuery
	role.Currencies = cloneRoleCurrencies(snapshot.Currencies)
	role.ContainerCapacities = cloneRoleContainerCapacities(snapshot.Capacities)
	role.Items = cloneRoleItems(snapshot.Items)
	role.SkillCap = snapshot.SkillCap
	role.Skills = cloneRoleSkills(snapshot.Skills)
	role.FastPanel = cloneRoleFastPanel(snapshot.FastPanel)
	role.TownBuffs = cloneRoleTownBuffs(snapshot.TownBuffs)
	roleState := snapshot.State
	roleState.Handle = role.RoleID
	role.RoleState = &roleState
	rolePhysique := snapshot.Physique
	rolePhysique.Handle = role.RoleID
	role.RolePhysique = &rolePhysique
	return role
}

// applyCapturedAChaiLevel50StatsAndSkills updates only the independently
// captured level-50 state and the already captured two-page skill list.
func applyCapturedAChaiLevel50StatsAndSkills(role RoleSummary) RoleSummary {
	stats := capturedAChaiSnapshot.Level50Stats
	role.Level = stats.Level
	role.Exp = stats.Exp
	role.AGI = stats.Physique.AGI
	role.STR = stats.Physique.STR
	role.INT = stats.Physique.INT
	role.CON = stats.Physique.CON
	role.LCK = stats.Physique.LCK
	role.SkillCap = capturedAChaiSnapshot.Role.SkillCap
	role.Skills = cloneRoleSkills(capturedAChaiSnapshot.Role.Skills)

	roleState := stats.State
	roleState.Handle = role.RoleID
	role.RoleState = &roleState
	rolePhysique := stats.Physique
	rolePhysique.Handle = role.RoleID
	role.RolePhysique = &rolePhysique
	return role
}

// applyPendingCapturedAChaiLevel50StatsAndSkillsMigration runs only for the
// explicitly synchronized local role. The role update and completion marker
// share one SQLite transaction so a completed migration cannot replay on login.
func (store *Store) applyPendingCapturedAChaiLevel50StatsAndSkillsMigration() (bool, error) {
	if store.db == nil {
		return false, nil
	}

	for playerID, roles := range store.rolesByPID {
		for index := range roles {
			if roles[index].RoleID != capturedAChaiLevel50StatsAndSkillsRoleID {
				continue
			}

			var migrated int
			err := store.db.QueryRow(
				`SELECT 1 FROM role_snapshot_migrations WHERE role_id = ? AND migration_key = ?`,
				roles[index].RoleID,
				capturedAChaiLevel50StatsAndSkillsMigrationKey,
			).Scan(&migrated)
			if err == nil {
				return false, nil
			}
			if err != sql.ErrNoRows {
				return false, fmt.Errorf("check captured a chai level 50 migration roleId=%s: %w", roles[index].RoleID, err)
			}

			migratedRole := applyCapturedAChaiLevel50StatsAndSkills(roles[index])
			payload, err := buildRolePersistencePayload(migratedRole)
			if err != nil {
				return false, fmt.Errorf("build captured a chai level 50 migration payload: %w", err)
			}
			tx, err := store.db.Begin()
			if err != nil {
				return false, fmt.Errorf("begin captured a chai level 50 migration: %w", err)
			}
			if err := upsertRolePersistencePayload(tx, playerID, payload); err != nil {
				_ = tx.Rollback()
				return false, fmt.Errorf("persist captured a chai level 50 migration: %w", err)
			}
			if _, err := tx.Exec(
				`INSERT INTO role_snapshot_migrations (role_id, migration_key) VALUES (?, ?)`,
				migratedRole.RoleID,
				capturedAChaiLevel50StatsAndSkillsMigrationKey,
			); err != nil {
				_ = tx.Rollback()
				return false, fmt.Errorf("record captured a chai level 50 migration: %w", err)
			}
			if err := tx.Commit(); err != nil {
				return false, fmt.Errorf("commit captured a chai level 50 migration: %w", err)
			}

			roles[index] = migratedRole
			store.rolesByPID[playerID] = roles
			return true, nil
		}
	}
	return false, nil
}

// MigrateCapturedAChaiRole replaces the requested local role with the final
// captured state for player_21499, using the Store persistence contract.
func (store *Store) MigrateCapturedAChaiRole(playerID string, roleID string) (RoleSummary, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}
		roles[index] = applyCapturedAChaiSnapshot(roles[index])
		store.rolesByPID[playerID] = roles
		if err := store.persistPlayerStateLocked(playerID); err != nil {
			return RoleSummary{}, fmt.Errorf("persist captured a chai role: %w", err)
		}
		return roles[index], nil
	}
	return RoleSummary{}, fmt.Errorf("captured a chai migration target role %q not found for player %q", roleID, playerID)
}

// MigrateCapturedAChaiSkills replaces only the captured role's skills and
// fast panel, preserving its current role state, inventory, and quest data.
func (store *Store) MigrateCapturedAChaiSkills(playerID string, roleID string) (RoleSummary, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}
		roles[index].Skills = cloneRoleSkills(capturedAChaiSnapshot.Role.Skills)
		roles[index].SkillCap = capturedAChaiSnapshot.Role.SkillCap
		roles[index].FastPanel = cloneRoleFastPanel(capturedAChaiSnapshot.Role.FastPanel)
		store.rolesByPID[playerID] = roles
		if err := store.persistPlayerStateLocked(playerID); err != nil {
			return RoleSummary{}, fmt.Errorf("persist captured a chai skills: %w", err)
		}
		return roles[index], nil
	}
	return RoleSummary{}, fmt.Errorf("captured a chai skill migration target role %q not found for player %q", roleID, playerID)
}

// MigrateCapturedAChaiQuests replaces the target role's active quest state
// with the final captured quest snapshot after the role row is persisted.
func (store *Store) MigrateCapturedAChaiQuests(playerID string, roleID string) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	if !store.hasRoleLocked(playerID, roleID) {
		return fmt.Errorf("captured a chai quest migration target role %q not found for player %q", roleID, playerID)
	}
	store.acceptedQuests[roleID] = make(map[string]bool, len(capturedAChaiSnapshot.Quests))
	store.removedQuests[roleID] = make(map[string]bool)
	store.questProgress[roleID] = make(map[string]int, len(capturedAChaiSnapshot.Quests))
	if store.db != nil {
		for _, tableName := range []string{"role_accepted_quests", "role_removed_quests", "role_quest_progress"} {
			if _, err := store.db.Exec("DELETE FROM "+tableName+" WHERE role_id = ?", roleID); err != nil {
				return fmt.Errorf("clear captured a chai quest table %s: %w", tableName, err)
			}
		}
	}
	for _, quest := range capturedAChaiSnapshot.Quests {
		store.acceptedQuests[roleID][quest.Name] = true
		store.questProgress[roleID][quest.Name] = quest.Progress
		if err := store.persistAcceptedQuestLocked(playerID, roleID, quest.Name); err != nil {
			return fmt.Errorf("persist captured a chai quest %q: %w", quest.Name, err)
		}
		if quest.Progress > 0 {
			if err := store.persistQuestProgressLocked(playerID, roleID, quest.Name, quest.Progress); err != nil {
				return fmt.Errorf("persist captured a chai quest progress %q: %w", quest.Name, err)
			}
		}
	}
	return nil
}
