package session

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func socketFixture(t *testing.T, store *Store, current, count int) (string, string, RoleItem, RoleItem) {
	t.Helper()
	pid, rid, _, _ := batchRefinementFixture(t, store, 0, 1)
	tool, ok := store.GrantRoleItem(pid, rid, RoleItem{Type: "背包", Name: "凿孔器", ItemType: "oneI", Display: "776.png", Description: "f_i_凿孔器&25@9999", Count: count, Index: -1})
	if !ok {
		t.Fatal("tool grant")
	}
	target, ok := store.GrantRoleItem(pid, rid, RoleItem{Type: "装备", Name: "凿孔测试剑", ItemType: "equip", Display: "123.png", Description: fmt.Sprintf("f_i_测试剑&1@55(+4)&23@凿孔上限 9 格&18@%d&f_i_原石&1@10&18@8", current), Count: 1, Index: 1, Level: 2})
	if !ok {
		t.Fatal("equipment grant")
	}
	return pid, rid, tool, target
}

func TestEquipmentSocketAllRatesAndCosts(t *testing.T) {
	for holes, rate := range equipmentSocketRates {
		for _, roll := range []int{0, rate*100 - 1, rate * 100} {
			if roll >= 10000 {
				continue
			}
			store := NewStore()
			pid, rid, tool, target := socketFixture(t, store, holes, 1)
			store.socketRoll = func(int) int { return roll }
			r := store.SocketRoleEquipment(pid, rid, tool.Type, tool.Index, target.Type, target.Index)
			want := roll < rate*100
			if !r.Applied || r.Succeeded != want || len(r.ClearedItems) != 1 {
				t.Fatalf("holes %d roll %d: %+v", holes, roll, r)
			}
			if _, ok := store.GetRoleItem(pid, rid, tool.Type, tool.Index); ok {
				t.Fatal("last tool not cleared")
			}
			got, _ := store.GetRoleItem(pid, rid, target.Type, target.Index)
			n, _, valid := equipmentSocketState(got)
			expected := holes
			if want {
				expected++
			}
			if !valid || n != expected || got.Level != target.Level || !strings.Contains(got.Description, "&1@55(+4)") || !strings.HasSuffix(got.Description, "f_i_原石&1@10&18@8") {
				t.Fatal("mutated other attributes", got)
			}
			if !want && got.Description != target.Description {
				t.Fatal("failure changed description")
			}
		}
	}
}

func TestEquipmentSocketRejectAndConcurrentLastTool(t *testing.T) {
	store := NewStore()
	pid, rid, tool, target := socketFixture(t, store, 9, 1)
	if r := store.SocketRoleEquipment(pid, rid, tool.Type, tool.Index, target.Type, target.Index); r.Applied || r.ErrorMessage != "已达到凿孔上限。" {
		t.Fatal(r)
	}
	for _, types := range [][2]string{{"仓库", "装备"}, {"背包", "仓库"}} {
		if store.SocketRoleEquipment(pid, rid, types[0], tool.Index, types[1], target.Index).Applied {
			t.Fatal("invalid containers")
		}
	}
	current, _ := store.GetRoleItem(pid, rid, tool.Type, tool.Index)
	if current.Count != 1 {
		t.Fatal("rejection consumed material")
	}
	for _, desc := range []string{"", "f_i_测试&18@x&23@凿孔上限 9 格", "f_i_测试&18@1&18@2&23@凿孔上限 9 格", "f_i_测试&23@凿孔上限 12 格"} {
		bad := target
		bad.Description = desc
		if _, _, ok := equipmentSocketState(bad); ok {
			t.Fatal("invalid description accepted", desc)
		}
	}
	store = NewStore()
	pid, rid, tool, target = socketFixture(t, store, 0, 1)
	var wg sync.WaitGroup
	results := make(chan RoleEquipmentSocketResult, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- store.SocketRoleEquipment(pid, rid, tool.Type, tool.Index, target.Type, target.Index)
		}()
	}
	wg.Wait()
	close(results)
	applied := 0
	for r := range results {
		if r.Applied {
			applied++
		}
	}
	if applied != 1 {
		t.Fatal("last tool spent twice", applied)
	}
}

func TestEquipmentSocketPersistenceRollback(t *testing.T) {
	db := filepath.Join(t.TempDir(), "socket.db")
	store, err := NewPersistentStore(db)
	if err != nil {
		t.Fatal(err)
	}
	pid, rid, tool, target := socketFixture(t, store, 0, 3)
	if !store.SocketRoleEquipment(pid, rid, tool.Type, tool.Index, target.Type, target.Index).Applied {
		t.Fatal("first hole failed")
	}
	store.Close()
	store, err = NewPersistentStore(db)
	if err != nil {
		t.Fatal(err)
	}
	item, _ := store.GetRoleItem(pid, rid, target.Type, target.Index)
	n, _, _ := equipmentSocketState(item)
	material, _ := store.GetRoleItem(pid, rid, tool.Type, tool.Index)
	if n != 1 || material.Count != 2 {
		t.Fatal("not persisted")
	}
	store.Close()
	store.socketRoll = func(int) int { return 0 }
	r := store.SocketRoleEquipment(pid, rid, tool.Type, tool.Index, target.Type, target.Index)
	if r.Applied || r.ErrorMessage != "凿孔结果保存失败。" {
		t.Fatal(r)
	}
	item, _ = store.GetRoleItem(pid, rid, target.Type, target.Index)
	n, _, _ = equipmentSocketState(item)
	material, _ = store.GetRoleItem(pid, rid, tool.Type, tool.Index)
	if n != 1 || material.Count != 2 {
		t.Fatal("rollback failed")
	}
}
