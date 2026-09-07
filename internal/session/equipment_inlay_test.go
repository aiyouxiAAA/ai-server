package session

import (
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func inlayFixture(t *testing.T, store *Store, count int) (string, string, EquipmentInlayRequest) {
	t.Helper()
	pid, rid, _, _ := batchRefinementFixture(t, store, 0, 1)
	gem, ok := CapturedRoleItemTemplate("壹级谜影石")
	if !ok {
		t.Fatal("missing captured gem")
	}
	gem.Type = "背包"
	gem.Index = -1
	gem.Count = count
	gem, ok = store.GrantRoleItem(pid, rid, gem)
	if !ok {
		t.Fatal("grant")
	}
	target, ok := store.GrantRoleItem(pid, rid, RoleItem{Type: "装备", Index: 1, Name: "测试肩甲", ItemType: "equip", Count: 1, Display: "484.png", Level: 6, Description: "f_i_测试肩甲&3@18(+18)&23@凿孔上限 9 格&18@3"})
	if !ok {
		t.Fatal("target")
	}
	slot := 2
	return pid, rid, EquipmentInlayRequest{SourceType: gem.Type, SourceIndex: &gem.Index, TargetType: target.Type, TargetIndex: &target.Index, SocketIndex: &slot, ExpectedGemCount: count, ExpectedGemDescription: gem.Description, ExpectedTargetDescription: target.Description}
}

func TestEquipmentInlayWritesReplacementAndStaleGuard(t *testing.T) {
	s := NewStore()
	pid, rid, r := inlayFixture(t, s, 2)
	got := s.InlayRoleEquipment(pid, rid, r)
	if !got.Applied || len(got.UpdatedItems) != 2 {
		t.Fatal(got)
	}
	target, _ := s.GetRoleItem(pid, rid, r.TargetType, *r.TargetIndex)
	if !strings.HasPrefix(target.Description, r.ExpectedTargetDescription+"f_i_") || !strings.Contains(target.Description, "&101@895.png&102@2") || target.Level != 6 {
		t.Fatal(target)
	}
	if s.InlayRoleEquipment(pid, rid, r).Applied {
		t.Fatal("stale duplicate spent twice")
	}
	r.ExpectedTargetDescription = target.Description
	r.ExpectedGemCount = 1
	if s.InlayRoleEquipment(pid, rid, r).Applied {
		t.Fatal("replacement without consent")
	}
	r.Replace = true
	if got = s.InlayRoleEquipment(pid, rid, r); !got.Applied || len(got.ClearedItems) != 1 {
		t.Fatal(got)
	}
	target, _ = s.GetRoleItem(pid, rid, r.TargetType, *r.TargetIndex)
	if strings.Count(target.Description, "&102@2") != 1 {
		t.Fatal("duplicate socket")
	}
}

func TestEquipmentInlayValidationAndConcurrentLastGem(t *testing.T) {
	s := NewStore()
	pid, rid, r := inlayFixture(t, s, 1)
	for _, change := range []func(*EquipmentInlayRequest){
		func(r *EquipmentInlayRequest) { r.SocketIndex = nil }, func(r *EquipmentInlayRequest) { i := 3; r.SocketIndex = &i },
		func(r *EquipmentInlayRequest) { i := -1; r.SocketIndex = &i }, func(r *EquipmentInlayRequest) { r.SourceType = "仓库" },
		func(r *EquipmentInlayRequest) { r.ExpectedGemDescription = "forged" }, func(r *EquipmentInlayRequest) { r.ExpectedGemCount = 9 },
	} {
		bad := r
		change(&bad)
		if s.InlayRoleEquipment(pid, rid, bad).Applied {
			t.Fatal("invalid accepted")
		}
	}
	if s.InlayRoleEquipment("another-account", rid, r).Applied {
		t.Fatal("foreign role")
	}
	var wg sync.WaitGroup
	ch := make(chan bool, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); ch <- s.InlayRoleEquipment(pid, rid, r).Applied }()
	}
	wg.Wait()
	close(ch)
	applied := 0
	for ok := range ch {
		if ok {
			applied++
		}
	}
	if applied != 1 {
		t.Fatal("concurrent duplicate", applied)
	}
}

func TestEquipmentInlayPersistenceAndRollback(t *testing.T) {
	file := filepath.Join(t.TempDir(), "inlay.db")
	s, err := NewPersistentStore(file)
	if err != nil {
		t.Fatal(err)
	}
	pid, rid, r := inlayFixture(t, s, 2)
	if !s.InlayRoleEquipment(pid, rid, r).Applied {
		t.Fatal("insert")
	}
	s.Close()
	s, err = NewPersistentStore(file)
	if err != nil {
		t.Fatal(err)
	}
	target, _ := s.GetRoleItem(pid, rid, r.TargetType, *r.TargetIndex)
	gem, _ := s.GetRoleItem(pid, rid, r.SourceType, *r.SourceIndex)
	if !strings.Contains(target.Description, "&102@2") || gem.Count != 1 {
		t.Fatal("not persisted")
	}
	r.ExpectedTargetDescription = target.Description
	r.ExpectedGemCount = 1
	r.Replace = true
	s.Close()
	if got := s.InlayRoleEquipment(pid, rid, r); got.Applied || got.Message != "镶嵌结果保存失败。" {
		t.Fatal(got)
	}
	gem, _ = s.GetRoleItem(pid, rid, r.SourceType, *r.SourceIndex)
	if gem.Count != 1 {
		t.Fatal("rollback failed")
	}
}
