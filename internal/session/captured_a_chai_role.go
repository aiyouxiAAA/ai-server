package session

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed captured_a_chai_role.json
var capturedAChaiRoleJSON []byte

type capturedAChaiRoleSnapshot struct {
	SourceCapture string `json:"sourceCapture"`
	Role          struct {
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
