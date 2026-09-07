package session

import "testing"

func TestRefinementStoneRepairsLegacyShopType(t *testing.T) {
	for _, name := range []string{"初级精炼宝石", "精炼宝石", "高级精炼宝石"} {
		t.Run(name, func(t *testing.T) {
			template, found := CapturedRoleItemTemplate(name)
			if !found || template.ItemType != "oneI" {
				t.Fatalf("missing captured oneI template for %s: %+v", name, template)
			}
			item := template
			item.ItemType = "null"
			item.Description = "f_i_" + name + "&24@材料&20@装备精炼所需的宝石."
			item.Count = 7
			item.Index = 12
			fixed := fillMissingRoleItemTemplateFields(item)
			if fixed.ItemType != "oneI" || fixed.Count != 7 || fixed.Index != 12 || fixed.Description != item.Description {
				t.Fatalf("repair must change only the stale interaction type: %+v", fixed)
			}
			item.Display = "different.png"
			if normalizeRoleItem(item).ItemType != "null" {
				t.Fatal("must not repair a mismatched icon")
			}
		})
	}
	item := RoleItem{Name: "其他物品", ItemType: "null", Display: "435.png"}
	if isStaleRefinementStoneType(item, RoleItem{ItemType: "oneI", Display: "435.png"}) {
		t.Fatal("must not repair unrelated item names")
	}
}
