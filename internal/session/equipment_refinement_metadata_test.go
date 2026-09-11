package session

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

const legacyMaskDescription = "f_i_蛮力面甲^ffffff&24@护具·头部&25@1&21@20&22@战士&3@17&5@30&27@sitem_piput&103@0&104@0&105@&107@&108@100"

func legacyMask() RoleItem {
	return RoleItem{Name: "蛮力面甲", ItemType: "equip", Display: "334.png", Description: legacyMaskDescription, Type: "装备", Index: 0, Count: 1, ItemLevel: 1}
}

func TestMissingEquipmentRefinementMetadata(t *testing.T) {
	i := legacyMask()
	repaired, ok := missingEquipmentRefinementDescription(i)
	if !ok || !strings.HasPrefix(repaired, i.Description) || !strings.Contains(repaired, "&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2") {
		t.Fatal(repaired, ok)
	}
	i.Description = repaired
	if _, changed := missingEquipmentRefinementDescription(i); changed {
		t.Fatal("non-idempotent repair")
	}
	if got := rewriteClassicEquipmentRefinementDescription(repaired, 1); !strings.Contains(got, "&3@17(+2)") || !strings.Contains(got, "&5@30") {
		t.Fatal(got)
	}
	for _, change := range []func(*RoleItem){func(i *RoleItem) { i.Display = "other.png" }, func(i *RoleItem) { i.ItemType = "own" }, func(i *RoleItem) { i.Description = strings.Replace(i.Description, "&3@17", "&3@18", 1) }, func(i *RoleItem) { i.Description += "&19@自定义潜质" }} {
		i := legacyMask()
		change(&i)
		if _, changed := missingEquipmentRefinementDescription(i); changed {
			t.Fatal("changed a nonmatching instance", i)
		}
	}
}

func TestPersistentStoreRepairsMissingEquipmentRefinementMetadata(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	s, err := NewPersistentStore(path)
	if err != nil {
		t.Fatal(err)
	}
	pid, rid, gem, _ := batchRefinementFixture(t, s, 0, 2)
	mask := legacyMask()
	mask.Owner = "保留归属"
	s.rolesByPID[pid][0].Items = []RoleItem{mask, gem}
	if err = s.persistRoleItemsLocked(pid, rid); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = NewPersistentStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	item, ok := s.GetRoleItem(pid, rid, "装备", 0)
	if !ok || !strings.Contains(item.Description, "&19@") || item.Owner != mask.Owner || item.ItemLevel != 1 || item.Level != 0 {
		t.Fatal(item)
	}
	var raw string
	if err = s.db.QueryRow("SELECT items_json FROM roles WHERE role_id = ?", rid).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var items []RoleItem
	if err = json.Unmarshal([]byte(raw), &items); err != nil {
		t.Fatal(err)
	}
	persisted, found := findRoleItem(items, "装备", 0)
	if !found || !strings.Contains(persisted.Description, "&19@") {
		t.Fatal("repair was not persisted")
	}
	s.refinementRoll = func(int) int { return 0 }
	r := s.RefineRoleEquipment(pid, rid, gem.Type, gem.Index, "装备", 0)
	if !r.Succeeded || r.TargetItem.Level != 1 || !strings.Contains(r.TargetItem.Description, "&3@17(+2)") || !strings.Contains(r.TargetItem.Description, "&5@30") {
		t.Fatalf("%+v", r)
	}
}
