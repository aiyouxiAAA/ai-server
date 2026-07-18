package session

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed captured_konglongkanglang1_role.json
var capturedKonglongKanglang1RoleJSON []byte

type capturedKonglongKanglang1RoleSnapshot struct {
	SourceCapture string `json:"sourceCapture"`
	Role          struct {
		Vocation          string                  `json:"vocation"`
		Level             int                     `json:"level"`
		Exp               int                     `json:"exp"`
		MapID             int                     `json:"mapId"`
		MapX              int                     `json:"mapX"`
		MapY              int                     `json:"mapY"`
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
	Quests []capturedKonglongKanglang1QuestSnapshot `json:"quests"`
}

type capturedKonglongKanglang1QuestSnapshot struct {
	Name       string                       `json:"name"`
	Objectives []RoleQuestObjectiveProgress `json:"objectives"`
}

var capturedKonglongKanglang1Snapshot = mustLoadCapturedKonglongKanglang1Snapshot()

func mustLoadCapturedKonglongKanglang1Snapshot() capturedKonglongKanglang1RoleSnapshot {
	var snapshot capturedKonglongKanglang1RoleSnapshot
	if err := json.Unmarshal(capturedKonglongKanglang1RoleJSON, &snapshot); err != nil {
		panic(fmt.Sprintf("decode captured konglongkanglang1 role snapshot: %v", err))
	}
	return snapshot
}

func applyCapturedKonglongKanglang1Snapshot(role RoleSummary) RoleSummary {
	snapshot := capturedKonglongKanglang1Snapshot.Role
	role.DisplayName = "666"
	role.Voc = snapshot.Vocation
	role.Level = snapshot.Level
	role.Exp = snapshot.Exp
	role.MapID = snapshot.MapID
	role.MapX = snapshot.MapX
	role.MapY = snapshot.MapY
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

// MigrateCapturedKonglongKanglang1Role replaces the requested local role with
// the final player_21424 snapshot through the Store persistence contract.
func (store *Store) MigrateCapturedKonglongKanglang1Role(playerID string, roleID string) (RoleSummary, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}
		roles[index] = applyCapturedKonglongKanglang1Snapshot(roles[index])
		store.rolesByPID[playerID] = roles
		if err := store.persistPlayerStateLocked(playerID); err != nil {
			return RoleSummary{}, fmt.Errorf("persist captured konglongkanglang1 role: %w", err)
		}
		return roles[index], nil
	}
	return RoleSummary{}, fmt.Errorf("captured konglongkanglang1 migration target role %q not found for player %q", roleID, playerID)
}

// MigrateCapturedKonglongKanglang1Quests persists each captured objective.
// The legacy scalar table is populated only for single-objective quests.
func (store *Store) MigrateCapturedKonglongKanglang1Quests(playerID string, roleID string) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	if !store.hasRoleLocked(playerID, roleID) {
		return fmt.Errorf("captured konglongkanglang1 quest migration target role %q not found for player %q", roleID, playerID)
	}
	store.acceptedQuests[roleID] = make(map[string]bool, len(capturedKonglongKanglang1Snapshot.Quests))
	store.removedQuests[roleID] = make(map[string]bool)
	store.questProgress[roleID] = make(map[string]int, len(capturedKonglongKanglang1Snapshot.Quests))
	store.questObjectives[roleID] = make(map[string][]RoleQuestObjectiveProgress, len(capturedKonglongKanglang1Snapshot.Quests))
	if store.db != nil {
		for _, tableName := range []string{"role_accepted_quests", "role_removed_quests", "role_quest_progress", "role_quest_objectives"} {
			if _, err := store.db.Exec("DELETE FROM "+tableName+" WHERE role_id = ?", roleID); err != nil {
				return fmt.Errorf("clear captured konglongkanglang1 quest table %s: %w", tableName, err)
			}
		}
	}
	for _, quest := range capturedKonglongKanglang1Snapshot.Quests {
		objectives := cloneRoleQuestObjectiveProgress(quest.Objectives)
		store.acceptedQuests[roleID][quest.Name] = true
		store.questObjectives[roleID][quest.Name] = objectives
		if err := store.persistAcceptedQuestLocked(playerID, roleID, quest.Name); err != nil {
			return fmt.Errorf("persist captured konglongkanglang1 quest %q: %w", quest.Name, err)
		}
		if err := store.persistQuestObjectivesLocked(playerID, roleID, quest.Name, objectives); err != nil {
			return fmt.Errorf("persist captured konglongkanglang1 quest objectives %q: %w", quest.Name, err)
		}
		if len(objectives) != 1 {
			continue
		}
		store.questProgress[roleID][quest.Name] = objectives[0].Current
		if err := store.persistQuestProgressLocked(playerID, roleID, quest.Name, objectives[0].Current); err != nil {
			return fmt.Errorf("persist captured konglongkanglang1 quest progress %q: %w", quest.Name, err)
		}
	}
	return nil
}
