package session

import (
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func batchRefinementFixture(t *testing.T, store *Store, level, count int) (string, string, RoleItem, RoleItem) {
	t.Helper()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	created := store.CreateRole(RoleCreateRequest{PlayerID: login.PlayerID, SessionToken: login.SessionToken, DisplayName: "批量精炼测试", Gender: "female", RoleTemplateID: 1})
	gem, ok := store.GrantRoleItem(login.PlayerID, created.Role.RoleID, RoleItem{Type: "背包", Name: "初级精炼宝石", ItemType: "oneI", Display: "616.png", Description: "f_i_初级精炼宝石&24@宝物&25@999", Count: count, Index: -1})
	if !ok {
		t.Fatal("grant gem")
	}
	target, ok := store.GrantRoleItem(login.PlayerID, created.Role.RoleID, RoleItem{Type: "装备", Name: "批量测试护肩", ItemType: "equip", Display: "603.png", Description: "f_i_批量测试护肩&24@护具·肩部&25@1&21@10&3@10&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+3", Count: 1, Index: 19, Level: level, ItemLevel: 1})
	if !ok {
		t.Fatal("grant target")
	}
	return login.PlayerID, created.Role.RoleID, gem, target
}

func TestBatchRefinementGoalAndFailures(t *testing.T) {
	for _, tc := range []struct {
		name                                          string
		level, count, goal, roll, wantLevel, consumed int
		reached                                       bool
	}{
		{"goal", 0, 5, 2, 0, 2, 2, true}, {"cross rate segment", 0, 5, 3, 0, 3, 3, true}, {"exhausted", 0, 1, 2, 0, 1, 1, false},
		{"failure then retries", 2, 3, 3, 9999, 1, 3, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := NewStore()
			store.refinementRoll = func(int) int { return tc.roll }
			pid, rid, gem, target := batchRefinementFixture(t, store, tc.level, tc.count)
			r := store.RefineRoleEquipment(pid, rid, gem.Type, gem.Index, target.Type, target.Index, tc.goal)
			if !r.Refined || r.Succeeded != tc.reached || r.TargetItem.Level != tc.wantLevel {
				t.Fatalf("unexpected result %+v", r)
			}
			items, _, _ := store.GetRoleItems(pid, rid, "背包")
			remaining := 0
			for _, item := range items {
				if item.Name == gem.Name {
					remaining += item.Count
				}
			}
			if remaining != tc.count-tc.consumed {
				t.Fatalf("remaining %d", remaining)
			}
			stop := "宝石耗尽"
			if tc.reached {
				stop = "已达目标"
			}
			if !strings.Contains(r.ResultMessage, stop) {
				t.Fatal(r.ResultMessage)
			}
		})
	}
}

func TestBatchRefinementInvalidGoalsDoNotConsume(t *testing.T) {
	for _, goal := range []int{-1, 0, 4, 999999} {
		store := NewStore()
		pid, rid, gem, target := batchRefinementFixture(t, store, 0, 5)
		store.refinementRoll = func(int) int { t.Fatal("invalid goal rolled"); return 0 }
		r := store.RefineRoleEquipment(pid, rid, gem.Type, gem.Index, target.Type, target.Index, goal)
		if r.Refined || r.ErrorCode != "refinement_goal_invalid" {
			t.Fatalf("%d %+v", goal, r)
		}
		items, _, _ := store.GetRoleItems(pid, rid, "背包")
		found, _ := findRoleItem(items, gem.Type, gem.Index)
		if found.Count != 5 {
			t.Fatal("invalid goal consumed gems")
		}
	}
}

func TestBatchRefinementMultipleStacksAndDuplicateGoal(t *testing.T) {
	store := NewStore()
	pid, rid, gem, target := batchRefinementFixture(t, store, 0, 1)
	store.refinementRoll = func(int) int { return 0 }
	roles := store.rolesByPID[pid]
	for i := range roles {
		if roles[i].RoleID == rid {
			extra := gem
			extra.Index = 80
			extra.Count = 3
			roles[i].Items = append(roles[i].Items, extra)
			other := gem
			other.Index = 81
			other.Name = "精炼宝石"
			other.Display = "435.png"
			other.Count = 6
			roles[i].Items = append(roles[i].Items, other)
		}
	}
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			store.RefineRoleEquipment(pid, rid, gem.Type, gem.Index, target.Type, target.Index, 2)
		}()
	}
	wg.Wait()
	items, _, _ := store.GetRoleItems(pid, rid, "背包")
	extra, _ := findRoleItem(items, "背包", 80)
	other, _ := findRoleItem(items, "背包", 81)
	if extra.Count != 2 || other.Count != 6 {
		t.Fatalf("wrong batch cost %d / other %d", extra.Count, other.Count)
	}
	equipment, _, _ := store.GetRoleItems(pid, rid, "装备")
	updated, _ := findRoleItem(equipment, target.Type, target.Index)
	if updated.Level != 2 {
		t.Fatal(updated.Level)
	}
}

func TestBatchRefinementPersistenceAndRollback(t *testing.T) {
	db := filepath.Join(t.TempDir(), "batch.db")
	store, err := NewPersistentStore(db)
	if err != nil {
		t.Fatal(err)
	}
	pid, rid, gem, target := batchRefinementFixture(t, store, 0, 5)
	store.refinementRoll = func(int) int { return 0 }
	r := store.RefineRoleEquipment(pid, rid, gem.Type, gem.Index, target.Type, target.Index, 2)
	if !r.Refined || !strings.Contains(r.TargetItem.Description, "&3@10(+6)") {
		t.Fatal(r)
	}
	store.Close()
	store, err = NewPersistentStore(db)
	if err != nil {
		t.Fatal(err)
	}
	items, _, _ := store.GetRoleItems(pid, rid, "背包")
	remaining, _ := findRoleItem(items, gem.Type, gem.Index)
	if remaining.Count != 3 {
		t.Fatal(remaining)
	}
	equipment, _, _ := store.GetRoleItems(pid, rid, "装备")
	updated, _ := findRoleItem(equipment, target.Type, target.Index)
	if updated.Level != 2 {
		t.Fatal(updated)
	}
	store.Close()
	store.refinementRoll = func(int) int { return 0 }
	r = store.RefineRoleEquipment(pid, rid, gem.Type, gem.Index, target.Type, target.Index, 3)
	if r.Refined || r.ErrorCode != "refinement_persist_failed" {
		t.Fatalf("closed persistence %+v", r)
	}
	items, _, _ = store.GetRoleItems(pid, rid, "背包")
	remaining, _ = findRoleItem(items, gem.Type, gem.Index)
	equipment, _, _ = store.GetRoleItems(pid, rid, "装备")
	updated, _ = findRoleItem(equipment, target.Type, target.Index)
	if remaining.Count != 3 || updated.Level != 2 {
		t.Fatal("failed persistence mutated memory")
	}
}
