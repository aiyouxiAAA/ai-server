package session

import (
	"ai-server/internal/quest"
	"fmt"
)

type AuthoredQuestResult struct {
	Role        RoleSummary
	PlayerBase  PlayerBaseData
	GrantedItem *RoleItem
	Changed     bool
	Error       string
}

// These quests cannot be abandoned: the existing removed set means completed.
func (store *Store) AuthoredQuestReady(playerID, roleID, questID string) bool {
	q, ok := quest.FindAuthored(questID)
	if !ok {
		return false
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	for _, role := range store.rolesByPID[playerID] {
		if role.RoleID == roleID {
			return store.authoredQuestReadyLocked(playerID, role, q)
		}
	}
	return false
}

func (store *Store) authoredQuestReadyLocked(playerID string, role RoleSummary, q quest.AuthoredQuest) bool {
	if !store.acceptedQuests[role.RoleID][q.Info.Title] {
		return false
	}
	switch q.Rule.Kind {
	case "visit":
		return true
	case "equip":
		for _, item := range role.Items {
			if item.Type == "装备" && item.Index == 3 && item.Name == q.Rule.Target && item.Count > 0 {
				return true
			}
		}
	case "kill":
		return store.questProgress[role.RoleID][q.Info.Title] >= q.Rule.Count
	case "recover":
		base := playerBaseDataFromRole(playerID, withRoleRuntimeDefaults(role))
		return base.RoleState != nil && base.RolePhysique != nil && base.RoleState.HP >= base.RolePhysique.MaxHP && base.RoleState.MP >= base.RolePhysique.MaxMP
	}
	return false
}

// Validate and commit acceptance/gift or completion/experience as one operation.
// Publish in-memory changes only after the SQL transaction succeeds.
func (store *Store) TransactAuthoredQuest(playerID, roleID, questID string, complete bool) AuthoredQuestResult {
	q, exists := quest.FindAuthored(questID)
	if !exists {
		return AuthoredQuestResult{Error: "任务不存在。"}
	}
	persistLock := store.rolePersistenceLock(playerID, roleID)
	persistLock.Lock()
	defer persistLock.Unlock()
	store.mu.Lock()
	defer store.mu.Unlock()
	for index, source := range store.rolesByPID[playerID] {
		if source.RoleID != roleID {
			continue
		}
		role := withRoleRuntimeDefaults(source)
		result := AuthoredQuestResult{Role: role, PlayerBase: playerBaseDataFromRole(playerID, role)}
		fail := func(message string) AuthoredQuestResult { result.Error = message; return result }
		if store.removedQuests[roleID][q.Info.Title] {
			return fail("该任务已经完成。")
		}
		if role.Level < q.Info.Level {
			return fail("等级不足。")
		}
		if q.Rule.Previous != "" {
			previous, _ := quest.FindAuthored(q.Rule.Previous)
			if !store.removedQuests[roleID][previous.Info.Title] {
				return fail("请先完成前置任务。")
			}
		}
		if complete {
			if !store.authoredQuestReadyLocked(playerID, role, q) {
				return fail("任务目标尚未完成。")
			}
			role.Exp += q.Info.Reward.Experience
			role.Level = ClassicRoleLevelForExp(role.Exp, role.Level)
			role = syncRoleProgressionRuntimeData(role)
		} else {
			if store.acceptedQuests[roleID][q.Info.Title] {
				return fail("该任务已经领取。")
			}
			if len(store.acceptedQuests[roleID]) >= 5 {
				return fail("任务最多只能接5个")
			}
			if q.Rule.AcceptItem != "" {
				item, ok := CapturedRoleItemTemplate(q.Rule.AcceptItem)
				if !ok {
					return fail("任务物品配置缺失，请联系维护人员。")
				}
				item.Type, item.Index, item.Count = "背包", -1, 1
				baseCapacity, _ := roleContainerCapacity(item.Type)
				items, granted, ok := grantRoleItemToItems(cloneRoleItems(role.Items), roleContainerCapacityForRole(role, item.Type, baseCapacity), item)
				if !ok {
					return fail("背包已满，请空出一格后再领取任务。")
				}
				role.Items = normalizeRoleItems(items)
				result.GrantedItem = &granted
			}
		}
		if err := store.persistAuthoredQuestLocked(playerID, role, q, complete, result.GrantedItem); err != nil {
			result.GrantedItem = nil
			return fail(fmt.Sprintf("任务保存失败，请重试：%v", err))
		}
		store.rolesByPID[playerID][index] = role
		if store.acceptedQuests[roleID] == nil {
			store.acceptedQuests[roleID] = map[string]bool{}
		}
		if store.removedQuests[roleID] == nil {
			store.removedQuests[roleID] = map[string]bool{}
		}
		delete(store.questProgress[roleID], q.Info.Title)
		delete(store.questObjectives[roleID], q.Info.Title)
		if complete {
			delete(store.acceptedQuests[roleID], q.Info.Title)
			store.removedQuests[roleID][q.Info.Title] = true
		} else {
			store.acceptedQuests[roleID][q.Info.Title] = true
		}
		result.Role, result.PlayerBase, result.Changed = role, playerBaseDataFromRole(playerID, role), true
		return result
	}
	return AuthoredQuestResult{Error: "角色不存在。"}
}

func (store *Store) persistAuthoredQuestLocked(playerID string, role RoleSummary, q quest.AuthoredQuest, complete bool, item *RoleItem) error {
	if store.db == nil {
		return nil
	}
	payload, err := buildRolePersistencePayload(role)
	if err != nil {
		return err
	}
	tx, err := store.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = upsertRolePersistencePayload(tx, playerID, payload); err != nil {
		return err
	}
	title, roleID := q.Info.Title, role.RoleID
	if complete {
		if _, err = tx.Exec(`DELETE FROM role_accepted_quests WHERE role_id=? AND title=?`, roleID, title); err != nil {
			return err
		}
		_, err = tx.Exec(`INSERT INTO role_removed_quests(role_id,title,player_id) VALUES(?,?,?)`, roleID, title, playerID)
	} else {
		_, err = tx.Exec(`INSERT INTO role_accepted_quests(role_id,title,player_id) VALUES(?,?,?)`, roleID, title, playerID)
	}
	if err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM role_quest_progress WHERE role_id=? AND title=?`, roleID, title); err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM role_quest_objectives WHERE role_id=? AND title=?`, roleID, title); err != nil {
		return err
	}
	if item != nil {
		acquisition := roleItemAcquisitionFromItem(playerID, roleID, *item, RoleItemAcquisitionSource{Kind: "任务物资", Detail: q.Info.ID})
		acquisition.OccurredAtUnixMs = store.now().UnixMilli()
		if err = insertRoleItemAcquisition(tx, acquisition); err != nil {
			return err
		}
	}
	return tx.Commit()
}
