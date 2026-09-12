package session

import (
	"strings"
	"testing"

	"ai-server/internal/classicdata"
)

func TestOriginalXuezhanTachiGrantEquipAndRemove(t *testing.T) {
	row, found, err := classicdata.FindItemByName("雪栈铁刀")
	if err != nil || !found || row["status"] != "authored" || row["required_level"] != "1" || row["phy_atk"] != "6" || row["quality_color"] != "ffffff" {
		t.Fatalf("invalid original starter weapon: %v %+v", err, row)
	}
	template, ok := CapturedRoleItemTemplate("雪栈铁刀")
	if !ok || template.ItemType != "equip" || template.Display != "original-equipment/xuezhan-iron-tachi.png" || !strings.Contains(template.Description, "&21@1&1@6") {
		t.Fatalf("original item must resolve through shared provider: %+v", template)
	}
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	created := store.CreateRole(RoleCreateRequest{PlayerID: login.PlayerID, SessionToken: login.SessionToken, DisplayName: "赠刀验证", Gender: "female", RoleTemplateID: 1})
	if !created.Success || created.Role.Level != 1 {
		t.Fatalf("expected level-one role: %+v", created)
	}
	for _, item := range created.Role.Items {
		if item.Name == template.Name {
			t.Fatal("table registration must not silently grant the future quest reward")
		}
	}
	template.Type, template.Index = "背包", -1
	granted, ok := store.GrantRoleItem(login.PlayerID, created.Role.RoleID, template)
	if !ok {
		t.Fatal("explicit test grant failed")
	}
	if bonus := roleEquipmentStats([]RoleItem{granted}); bonus.phyAtk != 0 {
		t.Fatal("bag weapon must not contribute attack")
	}
	equipped := store.EquipRoleItem(login.PlayerID, created.Role.RoleID, "背包", granted.Index, 1)
	if !equipped.Equipped || equipped.EquippedItem.Index != 3 || roleEquipmentStats(equipped.Role.Items).phyAtk != 6 {
		t.Fatalf("weapon must equip into slot 3 and add exactly six attack: %+v", equipped)
	}
	missing := store.UseRoleItem(login.PlayerID, created.Role.RoleID, "背包", granted.Index)
	if missing.ErrorCode != "item_missing" {
		t.Fatalf("consumed bag slot must not equip twice: %s", missing.ErrorCode)
	}
	removed := store.MoveRoleItem(login.PlayerID, created.Role.RoleID, "装备", 3, "背包", granted.Index, 1)
	if !removed.Moved || roleEquipmentStats(removed.Role.Items).phyAtk != 0 {
		t.Fatalf("unequipping must remove the attack contribution: %+v", removed)
	}
}
