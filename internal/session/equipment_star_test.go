package session

import (
	"path/filepath"
	"reflect"
	"testing"
)

func starFixture(t *testing.T, store *Store) (string, string, RoleItem) {
	t.Helper()
	pid, rid, _, target := batchRefinementFixture(t, store, 1, 1)
	for i := 0; i < 10; i++ {
		mat := target
		mat.Type = "背包"
		mat.Index = -1
		mat.Level = 0
		if _, ok := store.GrantRoleItem(pid, rid, mat); !ok {
			t.Fatal("grant material")
		}
	}
	q := EquipmentStarRequest{Action: "prepare", Type: target.Type, Index: target.Index}
	r := store.EquipmentStar(pid, rid, q)
	if !r.Success {
		t.Fatal(r.Message)
	}
	target, _ = store.GetRoleItem(pid, rid, target.Type, target.Index)
	return pid, rid, target
}
func TestEquipmentStarProgressionAndReplay(t *testing.T) {
	store := NewStore()
	pid, rid, target := starFixture(t, store)
	originalDescription := target.Description
	initial := cloneEquipmentStar(target.Star)
	for level := 0; level < 5; level++ {
		items, _, _ := store.GetRoleItems(pid, rid, "背包")
		ids := []string{}
		for _, item := range items {
			if EquipmentStarMaterialEligible(target, item) && len(ids) < starCosts[level] {
				ids = append(ids, item.Star.ID)
			}
		}
		q := EquipmentStarRequest{Action: "upgrade", Type: target.Type, Index: target.Index, ID: target.Star.ID, ExpectedStar: level, Materials: ids}
		if level == 4 {
			q.Choice = target.Star.Attributes[0].Code
		}
		r := store.EquipmentStar(pid, rid, q)
		if !r.Success {
			t.Fatal(level, r.Message)
		}
		if len(r.ClearedItems) != starCosts[level] {
			t.Fatal("wrong consumption", r)
		}
		target, _ = store.GetRoleItem(pid, rid, target.Type, target.Index)
		if target.Star.Level != level+1 || len(target.Star.Attributes) != min(4, level+3) || target.Level != 1 || target.Description != originalDescription {
			t.Fatal("state corruption", target)
		}
		if store.EquipmentStar(pid, rid, q).Success {
			t.Fatal("replay accepted")
		}
	}
	if target.Star.ID != initial.ID {
		t.Fatal("identity changed")
	}
	before := cloneEquipmentStar(target.Star)
	r := store.EquipmentStar(pid, rid, EquipmentStarRequest{Action: "prepare", Type: target.Type, Index: target.Index})
	if !r.Success {
		t.Fatal(r.Message)
	}
	target, _ = store.GetRoleItem(pid, rid, target.Type, target.Index)
	if !reflect.DeepEqual(before, target.Star) {
		t.Fatal("prepare rerolled")
	}
	with := roleEquipmentStats([]RoleItem{target})
	base := target
	base.Star = nil
	without := roleEquipmentStats([]RoleItem{base})
	if reflect.DeepEqual(with, without) {
		t.Fatal("star stats not applied")
	}
	target.Type = "背包"
	if roleEquipmentStats([]RoleItem{target}) != (roleEquipmentStatBonus{}) {
		t.Fatal("unequipped contribution")
	}
}
func TestEquipmentStarPersistenceAndProtection(t *testing.T) {
	db := filepath.Join(t.TempDir(), "star.db")
	store, err := NewPersistentStore(db)
	if err != nil {
		t.Fatal(err)
	}
	pid, rid, target := starFixture(t, store)
	before := cloneEquipmentStar(target.Star)
	items, _, _ := store.GetRoleItems(pid, rid, "背包")
	var material RoleItem
	for _, item := range items {
		if EquipmentStarMaterialEligible(target, item) {
			material = item
			break
		}
	}
	for _, mutate := range []func(*RoleItem){func(i *RoleItem) { i.Locked = true }, func(i *RoleItem) { i.Level = 1 }, func(i *RoleItem) { i.Type = "装备" }, func(i *RoleItem) { i.Description += "f_i_gem" }, func(i *RoleItem) { i.ItemLevel++ }, func(i *RoleItem) { i.EndTime = 123 }, func(i *RoleItem) { i.Owner = "other" }} {
		copy := material
		mutate(&copy)
		if EquipmentStarMaterialEligible(target, copy) {
			t.Fatal("protected material accepted", copy)
		}
	}
	store.Close()
	store, err = NewPersistentStore(db)
	if err != nil {
		t.Fatal(err)
	}
	target, _ = store.GetRoleItem(pid, rid, target.Type, target.Index)
	if !reflect.DeepEqual(before, target.Star) {
		t.Fatal("restart changed attributes")
	}
	// Closed database forces persist error; no materials, state, or random results may leak.
	store.Close()
	q := EquipmentStarRequest{Action: "upgrade", Type: target.Type, Index: target.Index, ID: target.Star.ID, Materials: []string{material.Star.ID}}
	r := store.EquipmentStar(pid, rid, q)
	if r.Success {
		t.Fatal("closed DB accepted save")
	}
	after, _ := store.GetRoleItem(pid, rid, target.Type, target.Index)
	if !reflect.DeepEqual(before, after.Star) {
		t.Fatal("rollback changed star")
	}
	kept, ok := store.GetRoleItem(pid, rid, material.Type, material.Index)
	if !ok || kept.Star.ID != material.Star.ID {
		t.Fatal("rollback lost material")
	}
}
