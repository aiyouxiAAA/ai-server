package session

import (
	"path/filepath"
	"testing"
)

func TestAuthoredQuestAtomicGiftAndPersistence(t *testing.T) {
	file := filepath.Join(t.TempDir(), "quests.db")
	store, err := NewPersistentStore(file)
	if err != nil {
		t.Fatal(err)
	}
	login := mustLogin(t, store, "mockuser", "magicpwd")
	created := store.CreateRole(RoleCreateRequest{PlayerID: login.PlayerID, SessionToken: login.SessionToken, DisplayName: "教程事务", Gender: "female", RoleTemplateID: 1})
	if !created.Success {
		t.Fatal(created)
	}
	player, id := login.PlayerID, created.Role.RoleID
	send := func(q string, complete bool) AuthoredQuestResult {
		return store.TransactAuthoredQuest(player, id, q, complete)
	}
	if send("XZ-M002", false).Changed {
		t.Fatal("skipped prerequisite")
	}
	if !send("XZ-M001", false).Changed || !send("XZ-M001", true).Changed {
		t.Fatal("first quest failed")
	}
	if _, err = store.db.Exec(`CREATE TRIGGER fail_gift BEFORE INSERT ON role_accepted_quests WHEN NEW.title='认识装备' BEGIN SELECT RAISE(ABORT,'test disk failure'); END`); err != nil {
		t.Fatal(err)
	}
	failed := send("XZ-M002", false)
	if failed.Changed || failed.GrantedItem != nil || store.AcceptedQuestTitles(player, id)["认识装备"] {
		t.Fatal("failed transaction published acceptance")
	}
	var items string
	if err = store.db.QueryRow(`SELECT items_json FROM roles WHERE role_id=?`, id).Scan(&items); err != nil {
		t.Fatal(err)
	}
	role, _, _ := store.GetRoleRuntimeData(player, id)
	for _, item := range role.Items {
		if item.Name == "雪栈铁刀" {
			t.Fatal("failed transaction leaked gift")
		}
	}
	if _, err = store.db.Exec(`DROP TRIGGER fail_gift`); err != nil {
		t.Fatal(err)
	}
	accepted := send("XZ-M002", false)
	if !accepted.Changed || accepted.GrantedItem == nil {
		t.Fatal(accepted.Error)
	}
	if send("XZ-M002", false).Changed || send("XZ-M002", true).Changed {
		t.Fatal("duplicate gift or unequipped completion")
	}
	giftIndex := accepted.GrantedItem.Index
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = NewPersistentStore(file)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if !store.AcceptedQuestTitles(player, id)["认识装备"] || send("XZ-M002", false).Changed {
		t.Fatal("restart lost grant state")
	}
	if !store.EquipRoleItem(player, id, "背包", giftIndex, 1).Equipped || !store.AuthoredQuestReady(player, id, "XZ-M002") {
		t.Fatal("equipment fact missing")
	}
	if !store.MoveRoleItem(player, id, "装备", 3, "背包", giftIndex, 1).Moved || store.AuthoredQuestReady(player, id, "XZ-M002") {
		t.Fatal("unequip must revoke readiness")
	}
	if !store.EquipRoleItem(player, id, "背包", giftIndex, 1).Equipped {
		t.Fatal("reequip")
	}
	if _, err = store.db.Exec(`CREATE TRIGGER fail_complete BEFORE INSERT ON role_removed_quests WHEN NEW.title='认识装备' BEGIN SELECT RAISE(ABORT,'test completion failure'); END`); err != nil {
		t.Fatal(err)
	}
	failed = send("XZ-M002", true)
	role, _, _ = store.GetRoleRuntimeData(player, id)
	if failed.Changed || role.Exp != 100 || !store.AcceptedQuestTitles(player, id)["认识装备"] {
		t.Fatal("failed completion changed experience/status")
	}
	if _, err = store.db.Exec(`DROP TRIGGER fail_complete`); err != nil {
		t.Fatal(err)
	}
	completed := send("XZ-M002", true)
	if !completed.Changed || completed.Role.Exp != 160 || send("XZ-M002", true).Changed {
		t.Fatal("completion not once-only")
	}
	var count int
	if err = store.db.QueryRow(`SELECT COUNT(*) FROM role_item_acquisitions WHERE role_id=? AND source_detail='XZ-M002'`, id).Scan(&count); err != nil || count != 1 {
		t.Fatalf("gift audit %d %v", count, err)
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = NewPersistentStore(file)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	role, _, _ = store.GetRoleRuntimeData(player, id)
	if role.Exp != 160 || !store.RemovedQuestTitles(player, id)["认识装备"] || send("XZ-M002", false).Changed {
		t.Fatal("completed restart")
	}
}

func TestAuthoredQuestFullBagAndRecovery(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	created := store.CreateRole(RoleCreateRequest{PlayerID: login.PlayerID, SessionToken: login.SessionToken, DisplayName: "满包教程", Gender: "female", RoleTemplateID: 1})
	player, id := login.PlayerID, created.Role.RoleID
	store.TransactAuthoredQuest(player, id, "XZ-M001", false)
	store.TransactAuthoredQuest(player, id, "XZ-M001", true)
	store.mu.Lock()
	var index int
	for i, r := range store.rolesByPID[player] {
		if r.RoleID == id {
			index = i
		}
	}
	role := store.rolesByPID[player][index]
	capacity, _ := roleContainerCapacity("背包")
	capacity = roleContainerCapacityForRole(role, "背包", capacity)
	role.Items = nil
	for i := 0; i < capacity; i++ {
		role.Items = append(role.Items, RoleItem{Type: "背包", Index: i, Name: "满格", Count: 1, ItemType: "equip"})
	}
	store.rolesByPID[player][index] = role
	store.mu.Unlock()
	result := store.TransactAuthoredQuest(player, id, "XZ-M002", false)
	if result.Changed || result.Error != "背包已满，请空出一格后再领取任务。" || store.AcceptedQuestTitles(player, id)["认识装备"] {
		t.Fatal("full bag accepted/lost sword", result)
	}
	store.mu.Lock()
	store.rolesByPID[player][index].Items = nil
	store.removedQuests[id]["认识装备"] = true
	store.removedQuests[id]["认识治疗"] = true
	store.removedQuests[id]["雪径初战"] = true
	store.mu.Unlock()
	if !store.TransactAuthoredQuest(player, id, "XZ-M005", false).Changed {
		t.Fatal("recovery accept")
	}
	_, base, _ := store.GetRoleRuntimeData(player, id)
	state := *base.RoleState
	state.HP -= 10
	state.MP -= 2
	store.UpdateRoleState(player, id, state)
	if store.AuthoredQuestReady(player, id, "XZ-M005") || store.TransactAuthoredQuest(player, id, "XZ-M005", true).Changed {
		t.Fatal("injured recovery completed")
	}
	if healed := store.HealRoleAtTown(player, id); !healed.Healed || healed.Cost != 0 {
		t.Fatal("healer failed")
	}
	if !store.AuthoredQuestReady(player, id, "XZ-M005") || !store.TransactAuthoredQuest(player, id, "XZ-M005", true).Changed {
		t.Fatal("recovery confirmation")
	}
}
