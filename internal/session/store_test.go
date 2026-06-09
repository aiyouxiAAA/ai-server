package session

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStoreLoginAccountSuccess(t *testing.T) {
	store := NewStore()

	response := store.Login(LoginRequest{
		UserName: "mockuser",
		Password: "magicpwd",
	})

	if !response.Success {
		t.Fatalf("expected login success, got failure: %+v", response)
	}
	if response.PlayerID != "mock-player-001" {
		t.Fatalf("expected player id mock-player-001, got %q", response.PlayerID)
	}
	if response.SessionToken != "mock-session-token-001" {
		t.Fatalf("expected session token mock-session-token-001, got %q", response.SessionToken)
	}
	if response.DisplayName != "Mock Swordswoman" {
		t.Fatalf("expected display name Mock Swordswoman, got %q", response.DisplayName)
	}
}

func TestStoreLoginAccountInvalidUserName(t *testing.T) {
	store := NewStore()

	response := store.Login(LoginRequest{
		UserName: "ab",
		Password: "magicpwd",
	})

	if response.Success {
		t.Fatalf("expected invalid username failure, got success: %+v", response)
	}
	if response.ErrorCode != "8" {
		t.Fatalf("expected error code 8, got %q", response.ErrorCode)
	}
	if response.ErrorMessage != "用户名不合法。" {
		t.Fatalf("expected invalid username message, got %q", response.ErrorMessage)
	}
}

func TestStoreLoginAccountAutoRegister(t *testing.T) {
	store := NewStore()

	firstLogin := store.Login(LoginRequest{
		UserName: "autouser",
		Password: "magicpwd",
	})
	secondLogin := store.Login(LoginRequest{
		UserName: "autouser",
		Password: "magicpwd",
	})

	if !firstLogin.Success || !secondLogin.Success {
		t.Fatalf("expected repeated login success, got first=%+v second=%+v", firstLogin, secondLogin)
	}
	if firstLogin.PlayerID != "acct-autouser" {
		t.Fatalf("expected auto registered player id acct-autouser, got %q", firstLogin.PlayerID)
	}
	if firstLogin.PlayerID != secondLogin.PlayerID {
		t.Fatalf("expected stable player id, got %q and %q", firstLogin.PlayerID, secondLogin.PlayerID)
	}
	if firstLogin.SessionToken != secondLogin.SessionToken {
		t.Fatalf("expected stable session token, got %q and %q", firstLogin.SessionToken, secondLogin.SessionToken)
	}
}

func TestStoreLoginAccountWrongPassword(t *testing.T) {
	store := NewStore()

	response := store.Login(LoginRequest{
		UserName: "mockuser",
		Password: "wrongpwd",
	})

	if response.Success {
		t.Fatalf("expected wrong password failure, got success: %+v", response)
	}
	if response.ErrorCode != "3" {
		t.Fatalf("expected error code 3, got %q", response.ErrorCode)
	}
	if response.ErrorMessage != "密码错误!" {
		t.Fatalf("expected wrong password message, got %q", response.ErrorMessage)
	}
}

func TestStoreLoginPlatformFallback(t *testing.T) {
	store := NewStore()

	response := store.Login(LoginRequest{
		Platform: "guest",
	})

	if !response.Success {
		t.Fatalf("expected guest login success, got failure: %+v", response)
	}
	if response.PlayerID != "guest-player-local" {
		t.Fatalf("expected guest player id, got %q", response.PlayerID)
	}
	if response.SessionToken != "local-session-guest-player-local" {
		t.Fatalf("expected guest session token, got %q", response.SessionToken)
	}
}

func TestStoreRoleLifecycle(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")

	listBefore := store.ListRoles(RoleListRequest{
		PlayerID:     login.PlayerID,
		SessionToken: login.SessionToken,
	})
	if !listBefore.Success {
		t.Fatalf("expected list roles success, got %+v", listBefore)
	}
	if len(listBefore.Roles) != 0 {
		t.Fatalf("expected no roles before create, got %d", len(listBefore.Roles))
	}

	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "测试女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	if !createResponse.Success {
		t.Fatalf("expected create role success, got %+v", createResponse)
	}
	if createResponse.Role.RoleID == "" {
		t.Fatal("expected created role id to be non-empty")
	}
	if createResponse.Role.DisplayName != "测试女侠" {
		t.Fatalf("expected created role name 测试女侠, got %q", createResponse.Role.DisplayName)
	}

	listAfter := store.ListRoles(RoleListRequest{
		PlayerID:     login.PlayerID,
		SessionToken: login.SessionToken,
	})
	if len(listAfter.Roles) != 1 {
		t.Fatalf("expected one role after create, got %d", len(listAfter.Roles))
	}
	if listAfter.Roles[0].RoleID != createResponse.Role.RoleID {
		t.Fatalf("expected listed role id %q, got %q", createResponse.Role.RoleID, listAfter.Roles[0].RoleID)
	}

	selectResponse := store.SelectRole(RoleSelectRequest{
		PlayerID:     login.PlayerID,
		SessionToken: login.SessionToken,
		RoleID:       createResponse.Role.RoleID,
	})
	if !selectResponse.Success {
		t.Fatalf("expected select role success, got %+v", selectResponse)
	}
	if selectResponse.Role.RoleID != createResponse.Role.RoleID {
		t.Fatalf("expected selected role id %q, got %q", createResponse.Role.RoleID, selectResponse.Role.RoleID)
	}
	if selectResponse.PlayerBase.PlayerID != login.PlayerID {
		t.Fatalf("expected player base player id %q, got %q", login.PlayerID, selectResponse.PlayerBase.PlayerID)
	}
	if selectResponse.PlayerBase.RoleID != createResponse.Role.RoleID {
		t.Fatalf("expected player base role id %q, got %q", createResponse.Role.RoleID, selectResponse.PlayerBase.RoleID)
	}
	if selectResponse.PlayerBase.Currencies["铜钱"] != 5000 || selectResponse.PlayerBase.Currencies["银元宝"] != 1 {
		t.Fatalf("expected default role currencies, got %+v", selectResponse.PlayerBase.Currencies)
	}
}

func TestStoreDungeonInstancePersistsForOneHour(t *testing.T) {
	for _, instanceKey := range []string{DungeonInstanceShuiliandong, DungeonInstanceHuangfengzhai, DungeonInstanceFeixiandong} {
		t.Run(instanceKey, func(t *testing.T) {
			store := NewStore()
			now := time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC)
			store.now = func() time.Time { return now }
			login := mustLogin(t, store, "mockuser", "magicpwd")
			createResponse := store.CreateRole(RoleCreateRequest{
				PlayerID:       login.PlayerID,
				SessionToken:   login.SessionToken,
				DisplayName:    "副本女侠",
				Gender:         "female",
				RoleTemplateID: 1,
			})

			state, ok := store.MarkRoleDungeonVisibleMonsterDefeated(login.PlayerID, createResponse.Role.RoleID, instanceKey, "5172206909807859")
			if !ok || state.CreatedAtUnix != now.Unix() || DungeonInstanceExpiresAtUnix(state) != now.Unix()+DungeonInstanceTTLSeconds() || len(state.DefeatedVisibleMonsterHandles) != 1 {
				t.Fatalf("expected dungeon instance defeat state, got ok=%v state=%+v", ok, state)
			}

			now = now.Add(59 * time.Minute)
			state, ok = store.EnsureRoleDungeonInstance(login.PlayerID, createResponse.Role.RoleID, instanceKey)
			if !ok || len(state.DefeatedVisibleMonsterHandles) != 1 {
				t.Fatalf("expected dungeon instance to persist before one hour, got ok=%v state=%+v", ok, state)
			}

			now = now.Add(2 * time.Minute)
			state, ok = store.EnsureRoleDungeonInstance(login.PlayerID, createResponse.Role.RoleID, instanceKey)
			if !ok || state.CreatedAtUnix != now.Unix() || len(state.DefeatedVisibleMonsterHandles) != 0 {
				t.Fatalf("expected dungeon instance to reset after one hour, got ok=%v state=%+v", ok, state)
			}
		})
	}
}

func TestStoreGetRoleDungeonInstanceDoesNotCreateNewInstance(t *testing.T) {
	store := NewStore()
	now := time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "副本门票女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	if state, ok := store.GetRoleDungeonInstance(login.PlayerID, createResponse.Role.RoleID, DungeonInstanceHuangfengzhai); ok || state.CreatedAtUnix != 0 {
		t.Fatalf("expected missing instance read not to create state, got ok=%v state=%+v", ok, state)
	}
	created, ok := store.EnsureRoleDungeonInstance(login.PlayerID, createResponse.Role.RoleID, DungeonInstanceHuangfengzhai)
	if !ok || created.CreatedAtUnix != now.Unix() {
		t.Fatalf("expected ensure to create dungeon instance, got ok=%v state=%+v", ok, created)
	}
	state, ok := store.GetRoleDungeonInstance(login.PlayerID, createResponse.Role.RoleID, DungeonInstanceHuangfengzhai)
	if !ok || state.CreatedAtUnix != created.CreatedAtUnix {
		t.Fatalf("expected existing instance read, got ok=%v state=%+v", ok, state)
	}
	now = now.Add(61 * time.Minute)
	if state, ok := store.GetRoleDungeonInstance(login.PlayerID, createResponse.Role.RoleID, DungeonInstanceHuangfengzhai); ok || state.CreatedAtUnix != 0 {
		t.Fatalf("expected expired instance to be pruned without creating replacement, got ok=%v state=%+v", ok, state)
	}
}

func TestStorePurchaseRoleSkillDeductsCurrenciesAndPersistsSkill(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "技能货币女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	result := store.PurchaseRoleSkill(
		login.PlayerID,
		createResponse.Role.RoleID,
		RoleSkill{Name: "武器专精", Level: 1, Type: "被动技能", Icon: "631.png"},
		RoleCurrencies{"铜钱": 500},
	)

	if !result.Found || !result.Learned {
		t.Fatalf("expected purchase success, got %+v", result)
	}
	if result.Currencies["铜钱"] != 4500 {
		t.Fatalf("expected copper deducted to 4500, got %+v", result.Currencies)
	}
	if len(result.Skills) != 3 || result.Skills[2].Name != "武器专精" {
		t.Fatalf("expected learned skill to persist in result, got %+v", result.Skills)
	}

	skills, _, ok := store.GetRoleSkills(login.PlayerID, createResponse.Role.RoleID)
	if !ok || len(skills) != 3 || skills[2].Name != "武器专精" {
		t.Fatalf("expected persisted learned skill, got ok=%v skills=%+v", ok, skills)
	}
	currencies, ok := store.GetRoleCurrencies(login.PlayerID, createResponse.Role.RoleID)
	if !ok || currencies["铜钱"] != 4500 {
		t.Fatalf("expected persisted currency deduction, got ok=%v currencies=%+v", ok, currencies)
	}
}

func TestStoreRoleInventoryDefaultsAndCapacity(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "背包女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	capacity, ok := store.GetRoleContainerCapacity(login.PlayerID, createResponse.Role.RoleID, "背包")
	if !ok || capacity != 30 {
		t.Fatalf("expected default bag capacity 30, got ok=%v capacity=%d", ok, capacity)
	}

	items, itemCapacity, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "背包")
	if !ok || itemCapacity != 30 {
		t.Fatalf("expected default bag item capacity 30, got ok=%v capacity=%d", ok, itemCapacity)
	}
	if len(items) != 1 {
		t.Fatalf("expected default bag seed to contain only the starter axe, got %+v", items)
	}
	if items[0].Name != "铁斧" || items[0].Display != "29.png" || items[0].ItemType != "equip" || items[0].Index != 19 {
		t.Fatalf("expected starter axe at bag index 19, got %+v", items[0])
	}
}

func TestStoreRoleInventoryRemovesOldCapturedBagSeeds(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "旧背包女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	store.rolesByPID[login.PlayerID][0].Items = capturedDefaultRoleItems()

	items, _, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "背包")
	if !ok {
		t.Fatal("expected bag list for role with old captured seeds")
	}
	if len(items) != 1 || items[0].Name != "铁斧" || items[0].Index != 19 {
		t.Fatalf("expected old captured bag seeds to be filtered down to starter axe, got %+v", items)
	}
}

func TestStoreGetRoleItemsTrimsStaleCurrencyStacks(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "钱币残留女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	silver, ok := CapturedRoleItemTemplate("银元宝")
	if !ok {
		t.Fatal("expected silver template")
	}
	silver.Type = "背包"
	silver.Index = 1
	silver.Count = 499
	copper, ok := CapturedRoleItemTemplate("铜钱")
	if !ok {
		t.Fatal("expected copper template")
	}
	copper.Type = "背包"
	copper.Index = 7
	copper.Count = 1000
	secondCopper := copper
	secondCopper.Index = 8

	store.rolesByPID[login.PlayerID][0].Currencies = RoleCurrencies{"银元宝": 499}
	store.rolesByPID[login.PlayerID][0].Items = []RoleItem{silver, copper, secondCopper}

	items, _, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "背包")
	if !ok {
		t.Fatal("expected bag items")
	}
	itemsByName := map[string]int{}
	for _, item := range items {
		itemsByName[item.Name] += item.Count
	}
	if itemsByName["银元宝"] != 499 {
		t.Fatalf("expected silver stack to stay at 499, got %+v", items)
	}
	if itemsByName["铜钱"] != 0 {
		t.Fatalf("expected stale copper stacks to be trimmed, got %+v", items)
	}

	persistedItems, _, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "背包")
	if !ok {
		t.Fatal("expected persisted bag items")
	}
	for _, item := range persistedItems {
		if item.Name == "铜钱" {
			t.Fatalf("expected stale copper stacks to stay removed, got %+v", persistedItems)
		}
	}
}

func TestStoreUseRoleItemClearsSelectedStaleCurrencyStack(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "双击残留铜钱女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	copper, ok := CapturedRoleItemTemplate("铜钱")
	if !ok {
		t.Fatal("expected copper template")
	}
	copper.Type = "背包"
	copper.Index = 7
	copper.Count = 1000
	store.rolesByPID[login.PlayerID][0].Currencies = RoleCurrencies{"银元宝": 499}
	store.rolesByPID[login.PlayerID][0].Items = []RoleItem{copper}

	result := store.UseRoleItem(login.PlayerID, createResponse.Role.RoleID, "背包", 7)
	if !result.Found || !result.Used {
		t.Fatalf("expected stale currency item to be cleared as a handled use, got %+v", result)
	}
	if len(result.ClearedItems) != 1 || result.ClearedItems[0].Type != "背包" || result.ClearedItems[0].Index != 7 {
		t.Fatalf("expected selected copper slot to clear, got %+v", result.ClearedItems)
	}
	items, _, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "背包")
	if !ok {
		t.Fatal("expected bag items")
	}
	for _, item := range items {
		if item.Name == "铜钱" {
			t.Fatalf("expected stale copper to be removed from bag, got %+v", items)
		}
	}
}

func TestStoreGrantRoleItemStacksCompatibleBagConsumables(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "叠加女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	template, ok := CapturedRoleItemTemplate("肉")
	if !ok {
		t.Fatal("expected captured 肉 template")
	}
	template.Type = "背包"
	template.Index = -1
	template.Count = 1

	first, ok := store.GrantRoleItem(login.PlayerID, createResponse.Role.RoleID, template)
	if !ok {
		t.Fatal("expected first 肉 grant to succeed")
	}
	second, ok := store.GrantRoleItem(login.PlayerID, createResponse.Role.RoleID, template)
	if !ok {
		t.Fatal("expected second 肉 grant to succeed")
	}

	if first.Index != second.Index {
		t.Fatalf("expected second 肉 grant to stack onto index %d, got %d", first.Index, second.Index)
	}
	if second.Count != 2 {
		t.Fatalf("expected stacked 肉 count 2 after second grant, got %+v", second)
	}

	items, _, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "背包")
	if !ok {
		t.Fatal("expected bag list after stacking 肉")
	}

	stackedCount := 0
	for _, item := range items {
		if item.Name != "肉" {
			continue
		}
		stackedCount += 1
		if item.Index != first.Index || item.Count != 2 {
			t.Fatalf("expected one stacked 肉 item at index %d count 2, got %+v", first.Index, item)
		}
	}
	if stackedCount != 1 {
		t.Fatalf("expected exactly one 肉 stack after two grants, got %+v", items)
	}
}

func TestStoreGrantRoleItemFillsCapturedHerbIcon(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "herbicon", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "草药女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	item, ok := store.GrantRoleItem(login.PlayerID, createResponse.Role.RoleID, RoleItem{
		Type:        "背包",
		Name:        "金银花",
		ItemType:    "null",
		Description: genericCollectionRewardDescription("金银花"),
		Count:       1,
		Index:       -1,
		ItemLevel:   1,
	})
	if !ok {
		t.Fatal("expected to grant 金银花")
	}
	if item.Display != "97.png" || item.Description != "f_i_金银花^ffffff&24@材料&25@99&20@性寒味甘.具有清热解毒&0;凉血化淤的功效.&101@97.png&103@0&104@0&105@&107@&108@13" {
		t.Fatalf("expected captured 金银花 icon and description, got %+v", item)
	}
}

func TestStoreEquipStarterAxeMovesBagItemToWeaponSlot(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "装备女侠",
		Gender:         "female",
		RoleTemplateID: 1,
		SourceQuery:    "human/human.swf?co=5&sex=1&hr=12&e=14&m=5&",
	})

	result := store.EquipRoleItem(login.PlayerID, createResponse.Role.RoleID, "背包", 19, 1)

	if !result.Found || !result.Equipped {
		t.Fatalf("expected starter axe equip success, got %+v", result)
	}
	if result.EquippedItem.Type != "装备" || result.EquippedItem.Index != 3 || result.EquippedItem.Name != "铁斧" {
		t.Fatalf("expected axe to move to source weapon slot 3, got %+v", result.EquippedItem)
	}
	if result.Role.SourceQuery != "human/human.swf?co=5&sex=1&hr=12&e=14&m=5&w8=5&" {
		t.Fatalf("expected source query to receive captured weapon field, got %q", result.Role.SourceQuery)
	}

	bagItems, _, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "背包")
	if !ok {
		t.Fatal("expected bag list after equip")
	}
	for _, item := range bagItems {
		if item.Name == "铁斧" {
			t.Fatalf("expected axe to be removed from bag after equip, got %+v", bagItems)
		}
	}

	equipItems, _, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "装备")
	if !ok || len(equipItems) != 1 || equipItems[0].Name != "铁斧" || equipItems[0].Index != 3 {
		t.Fatalf("expected equipment list to contain axe at index 3, ok=%v items=%+v", ok, equipItems)
	}
}

func TestStoreMoveRoleItemMovesBagItemToEmptySlot(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "挪包女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	result := store.MoveRoleItem(login.PlayerID, createResponse.Role.RoleID, "背包", 19, "背包", 0, 0)
	if !result.Found || !result.Moved {
		t.Fatalf("expected starter axe move success, got %+v", result)
	}
	if len(result.UpdatedItems) != 1 || result.UpdatedItems[0].Name != "铁斧" || result.UpdatedItems[0].Index != 0 {
		t.Fatalf("expected moved axe at slot 0, got %+v", result.UpdatedItems)
	}
	bagItems, _, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "背包")
	if !ok || len(bagItems) != 1 || bagItems[0].Name != "铁斧" || bagItems[0].Index != 0 {
		t.Fatalf("expected bag to persist moved axe at slot 0, ok=%v items=%+v", ok, bagItems)
	}
}

func TestStoreMoveRoleItemSwapsOccupiedBagSlots(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "换包女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	template, ok := CapturedRoleItemTemplate("肉")
	if !ok {
		t.Fatal("expected captured 肉 template")
	}
	template.Type = "背包"
	template.Index = 0
	template.Count = 1
	if _, ok := store.GrantRoleItem(login.PlayerID, createResponse.Role.RoleID, template); !ok {
		t.Fatal("expected 肉 grant to succeed")
	}

	result := store.MoveRoleItem(login.PlayerID, createResponse.Role.RoleID, "背包", 19, "背包", 0, 0)
	if !result.Found || !result.Moved {
		t.Fatalf("expected bag slot swap success, got %+v", result)
	}
	bagItems, _, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "背包")
	if !ok {
		t.Fatal("expected bag list after swap")
	}
	itemsByIndex := map[int]RoleItem{}
	for _, item := range bagItems {
		itemsByIndex[item.Index] = item
	}
	if itemsByIndex[0].Name != "铁斧" || itemsByIndex[19].Name != "肉" {
		t.Fatalf("expected axe/meat swap between slots 19 and 0, got %+v", bagItems)
	}
}

func TestStoreMoveRoleItemStacksSameNameBagItems(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "叠包女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	template, ok := CapturedRoleItemTemplate("肉")
	if !ok {
		t.Fatal("expected captured 肉 template")
	}
	template.Type = "背包"
	template.Index = 0
	template.Count = 2
	if _, ok := store.GrantRoleItem(login.PlayerID, createResponse.Role.RoleID, template); !ok {
		t.Fatal("expected first 肉 grant to succeed")
	}
	template.Index = 1
	template.Count = 3
	if _, ok := store.GrantRoleItem(login.PlayerID, createResponse.Role.RoleID, template); !ok {
		t.Fatal("expected second 肉 grant to succeed")
	}

	result := store.MoveRoleItem(login.PlayerID, createResponse.Role.RoleID, "背包", 1, "背包", 0, 0)
	if !result.Found || !result.Moved {
		t.Fatalf("expected bag item stack success, got %+v", result)
	}
	if len(result.UpdatedItems) != 1 || result.UpdatedItems[0].Index != 0 || result.UpdatedItems[0].Count != 5 || result.UpdatedItems[0].Name != "肉" {
		t.Fatalf("expected single stacked 肉 push at slot 0 count 5, got %+v", result.UpdatedItems)
	}
	bagItems, _, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "背包")
	if !ok {
		t.Fatal("expected bag list after stacking")
	}
	if len(bagItems) != 2 {
		t.Fatalf("expected only starter axe and stacked 肉 left in bag, got %+v", bagItems)
	}
	itemsByIndex := map[int]RoleItem{}
	for _, item := range bagItems {
		itemsByIndex[item.Index] = item
	}
	if itemsByIndex[0].Name != "肉" || itemsByIndex[0].Count != 5 {
		t.Fatalf("expected 肉 count 5 at slot 0 after stacking, got %+v", itemsByIndex[0])
	}
	if _, exists := itemsByIndex[1]; exists {
		t.Fatalf("expected slot 1 to be cleared after stacking, got %+v", bagItems)
	}
}

func TestStoreMoveRoleItemSplitsBagItemToEmptySlot(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "分堆女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	template, ok := CapturedRoleItemTemplate("肉")
	if !ok {
		t.Fatal("expected captured 肉 template")
	}
	template.Type = "背包"
	template.Index = 1
	template.Count = 5
	if _, ok := store.GrantRoleItem(login.PlayerID, createResponse.Role.RoleID, template); !ok {
		t.Fatal("expected 肉 grant to succeed")
	}

	result := store.MoveRoleItem(login.PlayerID, createResponse.Role.RoleID, "背包", 1, "背包", 2, 2)
	if !result.Found || !result.Moved {
		t.Fatalf("expected bag item split success, got %+v", result)
	}
	if len(result.ClearedItems) != 1 || result.ClearedItems[0].Index != 2 {
		t.Fatalf("expected only target slot clear before split push, got %+v", result.ClearedItems)
	}
	itemsByIndex := map[int]RoleItem{}
	for _, item := range result.UpdatedItems {
		itemsByIndex[item.Index] = item
	}
	if itemsByIndex[1].Count != 3 || itemsByIndex[2].Count != 2 {
		t.Fatalf("expected source count 3 and moved stack count 2, got %+v", result.UpdatedItems)
	}
	bagItems, _, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "背包")
	if !ok {
		t.Fatal("expected bag list after split move")
	}
	persistedByIndex := map[int]RoleItem{}
	for _, item := range bagItems {
		persistedByIndex[item.Index] = item
	}
	if persistedByIndex[1].Count != 3 || persistedByIndex[2].Count != 2 {
		t.Fatalf("expected persisted split stacks at slots 1 and 2, got %+v", bagItems)
	}
}

func TestStoreMoveRoleItemUnequipsToNextBagSlotAndClearsAppearance(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "卸装女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	equipResult := store.EquipRoleItem(login.PlayerID, createResponse.Role.RoleID, "背包", 19, 1)
	if !equipResult.Found || !equipResult.Equipped || !strings.Contains(equipResult.Role.SourceQuery, "w8=5") {
		t.Fatalf("expected starter axe equip with weapon appearance, got %+v", equipResult)
	}

	result := store.MoveRoleItem(login.PlayerID, createResponse.Role.RoleID, "装备", 3, "背包", -1, 0)
	if !result.Found || !result.Moved {
		t.Fatalf("expected equipment unequip success, got %+v", result)
	}
	if strings.Contains(result.Role.SourceQuery, "w8=5") {
		t.Fatalf("expected weapon appearance to be cleared after unequip, got %q", result.Role.SourceQuery)
	}
	if len(result.UpdatedItems) != 1 || result.UpdatedItems[0].Type != "背包" || result.UpdatedItems[0].Index != 0 || result.UpdatedItems[0].Name != "铁斧" {
		t.Fatalf("expected axe moved to first empty bag slot, got %+v", result.UpdatedItems)
	}
	equipItems, _, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "装备")
	if !ok || len(equipItems) != 0 {
		t.Fatalf("expected equipment slot to be empty after unequip, ok=%v items=%+v", ok, equipItems)
	}
}

func TestStoreClassicRoleProgressionMatchesCapturedRoleStateAndPhysique(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "升级女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	first := store.GrantRoleExperience(login.PlayerID, createResponse.Role.RoleID, 80)
	if !first.Found || !first.Granted {
		t.Fatalf("expected exp grant to succeed, got %+v", first)
	}
	if first.Role.Level != 2 || first.RoleState.Lv != 2 || first.RoleState.Exp != 80 {
		t.Fatalf("expected capture-compatible lv2 exp80, got role=%+v state=%+v", first.Role, first.RoleState)
	}
	if first.PlayerBase.RolePhysique == nil || first.PlayerBase.RolePhysique.MaxHP != 205 || first.PlayerBase.RolePhysique.MaxMP != 54 || first.PlayerBase.RolePhysique.LastPoint != 10 || first.PlayerBase.RolePhysique.Fat != 5 {
		t.Fatalf("expected captured lv2 vitality and points, got %+v", first.PlayerBase.RolePhysique)
	}

	second := store.GrantRoleExperience(login.PlayerID, createResponse.Role.RoleID, 50)
	if !second.Found || !second.Granted {
		t.Fatalf("expected second exp grant to succeed, got %+v", second)
	}
	if second.Role.Level != 3 || second.RoleState.Lv != 3 || second.RoleState.Exp != 130 {
		t.Fatalf("expected capture-compatible lv3 exp130, got role=%+v state=%+v", second.Role, second.RoleState)
	}
	if second.PlayerBase.RolePhysique == nil || second.PlayerBase.RolePhysique.MaxHP != 225 || second.PlayerBase.RolePhysique.MaxMP != 64 || second.PlayerBase.RolePhysique.LastPoint != 15 || second.PlayerBase.RolePhysique.Fat != 5 {
		t.Fatalf("expected captured lv3 vitality and points, got %+v", second.PlayerBase.RolePhysique)
	}

	third := store.GrantRoleExperience(login.PlayerID, createResponse.Role.RoleID, 178)
	if !third.Found || !third.Granted {
		t.Fatalf("expected third exp grant to succeed, got %+v", third)
	}
	if third.Role.Level != 4 || third.RoleState.Lv != 4 || third.RoleState.Exp != 308 {
		t.Fatalf("expected capture-compatible lv4 exp308, got role=%+v state=%+v", third.Role, third.RoleState)
	}
	if third.PlayerBase.RolePhysique == nil || third.PlayerBase.RolePhysique.MaxHP != 245 || third.PlayerBase.RolePhysique.MaxMP != 74 || third.PlayerBase.RolePhysique.LastPoint != 20 || third.PlayerBase.RolePhysique.Fat != 5 {
		t.Fatalf("expected captured lv4 vitality and points, got %+v", third.PlayerBase.RolePhysique)
	}
}

func TestPersistentStoreLoadsCapturedRolePanelOverrides(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ai-server.db")
	store, err := NewPersistentStore(path)
	if err != nil {
		t.Fatalf("new persistent store: %v", err)
	}
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "抓包面板",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	store.mu.Lock()
	store.rolesByPID[login.PlayerID][0].Level = 24
	store.rolesByPID[login.PlayerID][0].Exp = 712600
	store.rolesByPID[login.PlayerID][0].Voc = "战士"
	store.rolesByPID[login.PlayerID][0].RoleState = &RoleState{
		HP:    1074,
		MP:    274,
		Exp:   712600,
		Lv:    24,
		Speed: 147,
	}
	store.rolesByPID[login.PlayerID][0].RolePhysique = &RolePhysique{
		AGI:       42,
		STR:       83,
		INT:       0,
		CON:       0,
		LCK:       3,
		MaxHP:     1075,
		MaxMP:     274,
		PhyAtk:    198,
		MgcAtk:    0,
		PhyDef:    160,
		MgcDef:    40,
		Hit:       224,
		Dog:       114,
		Fat:       162,
		LastPoint: 0,
	}
	if err := store.persistPlayerStateLocked(login.PlayerID); err != nil {
		store.mu.Unlock()
		t.Fatalf("persist role panel override: %v", err)
	}
	store.mu.Unlock()
	if err := store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	reopened, err := NewPersistentStore(path)
	if err != nil {
		t.Fatalf("reopen persistent store: %v", err)
	}
	defer reopened.Close()
	role, playerBase, ok := reopened.GetRoleRuntimeData(login.PlayerID, createResponse.Role.RoleID)
	if !ok {
		t.Fatal("expected role runtime data after reopen")
	}
	if role.Level != 24 || role.Exp != 712600 || role.Voc != "战士" || playerBase.Voc != "战士" || playerBase.RoleState == nil || playerBase.RoleState.Speed != 147 || playerBase.RoleState.HP != 1074 {
		t.Fatalf("expected captured role state override, role=%+v base=%+v", role, playerBase)
	}
	if playerBase.RolePhysique == nil || playerBase.RolePhysique.AGI != 42 || playerBase.RolePhysique.STR != 83 || playerBase.RolePhysique.MaxHP != 1075 || playerBase.RolePhysique.PhyAtk != 198 || playerBase.RolePhysique.Hit != 224 {
		t.Fatalf("expected captured role physique override, got %+v", playerBase.RolePhysique)
	}
}

func TestStoreSetRoleVocationPersistsSelectedVocation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ai-server.db")
	store, err := NewPersistentStore(path)
	if err != nil {
		t.Fatalf("new persistent store: %v", err)
	}
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "转职女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	result := store.SetRoleVocation(login.PlayerID, createResponse.Role.RoleID, "游侠")
	if !result.Found || !result.Changed || result.Role.Voc != "游侠" || result.PlayerBase.Voc != "游侠" {
		t.Fatalf("expected role vocation 游侠, got %+v", result)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	reopened, err := NewPersistentStore(path)
	if err != nil {
		t.Fatalf("reopen persistent store: %v", err)
	}
	defer reopened.Close()
	role, playerBase, ok := reopened.GetRoleRuntimeData(login.PlayerID, createResponse.Role.RoleID)
	if !ok || role.Voc != "游侠" || playerBase.Voc != "游侠" {
		t.Fatalf("expected reopened vocation 游侠, ok=%v role=%+v base=%+v", ok, role, playerBase)
	}
}

func TestStoreAddRolePointMatchesCapturedRolePhysiqueDeltas(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "加点女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	store.rolesByPID[login.PlayerID][0].Level = 2
	store.rolesByPID[login.PlayerID][0].Exp = 80

	for index := 0; index < 8; index += 1 {
		result := store.AddRolePoint(login.PlayerID, createResponse.Role.RoleID, "STR")
		if !result.Found || !result.Applied {
			t.Fatalf("expected STR AddPoint %d to succeed, got %+v", index, result)
		}
	}
	afterStrength := store.AddRolePoint(login.PlayerID, createResponse.Role.RoleID, "AGI")
	if !afterStrength.Found || !afterStrength.Applied {
		t.Fatalf("expected AGI AddPoint to succeed, got %+v", afterStrength)
	}
	if afterStrength.RolePhysique.STR != 8 || afterStrength.RolePhysique.AGI != 1 ||
		afterStrength.RolePhysique.PhyAtk != 18 || afterStrength.RolePhysique.PhyDef != 11 ||
		afterStrength.RolePhysique.Hit != 101 || afterStrength.RolePhysique.Dog != 51 ||
		afterStrength.RolePhysique.Fat != 9 || afterStrength.RolePhysique.LastPoint != 1 {
		t.Fatalf("expected captured STR8/AGI1 physique, got %+v", afterStrength.RolePhysique)
	}

	afterSecondAgility := store.AddRolePoint(login.PlayerID, createResponse.Role.RoleID, "AGI")
	if !afterSecondAgility.Found || !afterSecondAgility.Applied {
		t.Fatalf("expected second AGI AddPoint to succeed, got %+v", afterSecondAgility)
	}
	if afterSecondAgility.RolePhysique.STR != 8 || afterSecondAgility.RolePhysique.AGI != 2 ||
		afterSecondAgility.RolePhysique.PhyAtk != 18 || afterSecondAgility.RolePhysique.PhyDef != 11 ||
		afterSecondAgility.RolePhysique.Hit != 102 || afterSecondAgility.RolePhysique.Dog != 51 ||
		afterSecondAgility.RolePhysique.Fat != 10 || afterSecondAgility.RolePhysique.LastPoint != 0 {
		t.Fatalf("expected captured STR8/AGI2 physique, got %+v", afterSecondAgility.RolePhysique)
	}

	noPoint := store.AddRolePoint(login.PlayerID, createResponse.Role.RoleID, "STR")
	if !noPoint.Found || noPoint.Applied || noPoint.ErrorCode != "no_point" {
		t.Fatalf("expected AddPoint to reject when no points remain, got %+v", noPoint)
	}
}

func TestStoreClassicRolePhysiqueIncludesEquippedSourceStats(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "装备属性女侠",
		Gender:         "female",
		RoleTemplateID: 1,
		SourceQuery:    "human/human.swf?co=5&sex=1&hr=12&e=14&m=5&",
	})

	result := store.EquipRoleItem(login.PlayerID, createResponse.Role.RoleID, "背包", 19, 1)

	if !result.Found || !result.Equipped {
		t.Fatalf("expected starter axe equip success, got %+v", result)
	}
	if result.PlayerBase.RolePhysique == nil || result.PlayerBase.RolePhysique.PhyAtk != 36 {
		t.Fatalf("expected equipped &1@26 axe to raise source phyAtk to 36, got %+v", result.PlayerBase.RolePhysique)
	}
}

func TestStoreCapturedStarterArmorTemplatesMatchSourceAndEquipStats(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "布衣装备女侠",
		Gender:         "female",
		RoleTemplateID: 1,
		SourceQuery:    "human/human.swf?co=5&sex=1&hr=12&e=14&m=5&",
	})

	cases := []struct {
		name            string
		display         string
		slot            int
		sourceQueryPart string
		descriptionPart string
	}{
		{name: "蓝布衣", display: "291.png", slot: 4, sourceQueryPart: "c=1", descriptionPart: "&24@护具·躯干"},
		{name: "蓝布裤", display: "3.png", slot: 5, sourceQueryPart: "p=1", descriptionPart: "&24@护具·腿"},
		{name: "布鞋", display: "274.png", slot: 12, sourceQueryPart: "se=1", descriptionPart: "&18@2"},
	}

	for _, tc := range cases {
		template, ok := CapturedRoleItemTemplate(tc.name)
		if !ok {
			t.Fatalf("expected captured source template for %s", tc.name)
		}
		if template.Type != "装备" || template.ItemType != "equip" || template.Display != tc.display || template.Index != tc.slot {
			t.Fatalf("expected %s source equipment template display=%s slot=%d, got %+v", tc.name, tc.display, tc.slot, template)
		}
		if !strings.Contains(template.Description, tc.descriptionPart) {
			t.Fatalf("expected %s description to include %q, got %q", tc.name, tc.descriptionPart, template.Description)
		}

		bagItem := template
		bagItem.Type = "背包"
		bagItem.Index = -1
		granted, ok := store.GrantRoleItem(login.PlayerID, createResponse.Role.RoleID, bagItem)
		if !ok {
			t.Fatalf("expected granting %s to bag to succeed", tc.name)
		}
		result := store.EquipRoleItem(login.PlayerID, createResponse.Role.RoleID, granted.Type, granted.Index, 1)
		if !result.Found || !result.Equipped {
			t.Fatalf("expected equipping %s to succeed, got %+v", tc.name, result)
		}
		if result.EquippedItem.Type != "装备" || result.EquippedItem.Index != tc.slot || result.EquippedItem.Name != tc.name {
			t.Fatalf("expected %s to equip into source slot %d, got %+v", tc.name, tc.slot, result.EquippedItem)
		}
		if !strings.Contains(result.Role.SourceQuery, tc.sourceQueryPart) {
			t.Fatalf("expected %s equip to add %s to source query, got %q", tc.name, tc.sourceQueryPart, result.Role.SourceQuery)
		}
	}

	role, playerBase, ok := store.GetRoleRuntimeData(login.PlayerID, createResponse.Role.RoleID)
	if !ok {
		t.Fatal("expected role runtime data after equipping starter armor")
	}
	for _, part := range []string{"c=1", "p=1", "se=1"} {
		if !strings.Contains(role.SourceQuery, part) {
			t.Fatalf("expected final source query to contain %s, got %q", part, role.SourceQuery)
		}
	}
	if playerBase.RolePhysique == nil || playerBase.RolePhysique.PhyDef != 22 {
		t.Fatalf("expected armor &3@6 + &3@4 + &3@2 to raise source phyDef to 22, got %+v", playerBase.RolePhysique)
	}
	if got := parseClassicDescriptionSignedInt(sourceBlueClothItem().Description, "3"); got != 6 {
		t.Fatalf("expected repeated blue-cloth &3@ placeholder to resolve to final numeric value 6, got %d", got)
	}
}

func TestStoreCapturedWarriorEquipmentSlotsAndAppearance(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "蛮力装备女侠",
		Gender:         "female",
		RoleTemplateID: 1,
		SourceQuery:    "human/human.swf?co=5&sex=1&hr=12&e=14&m=5&",
	})

	cases := []struct {
		name            string
		display         string
		descriptionPart string
		slot            int
		sourceQueryPart string
	}{
		{name: "刎刀", display: "42.png", descriptionPart: "&24@武器·单刀系", slot: 3, sourceQueryPart: "w8=42"},
		{name: "蛮力面甲", display: "334.png", descriptionPart: "&24@护具·头部", slot: 0, sourceQueryPart: "h=8"},
		{name: "蛮力护甲", display: "336.png", descriptionPart: "&24@护具·身体", slot: 4, sourceQueryPart: "c=10"},
		{name: "蛮力护腰", display: "339.png", descriptionPart: "&24@护具·腰部", slot: 10, sourceQueryPart: "b=5"},
		{name: "蛮力护腿", display: "338.png", descriptionPart: "&24@护具·腿部", slot: 5, sourceQueryPart: "p=8"},
		{name: "蛮力战靴", display: "340.png", descriptionPart: "&24@护具·脚部", slot: 12, sourceQueryPart: "se=4"},
		{name: "蛤蟆精战靴", display: "503.png", descriptionPart: "&24@护具·足部", slot: 12, sourceQueryPart: "se=29"},
		{name: "蛮力肩甲", display: "335.png", descriptionPart: "&24@护具·肩部", slot: 1, sourceQueryPart: "a=4"},
		{name: "蛮力护腕", display: "337.png", descriptionPart: "&24@护具·腕部", slot: 2, sourceQueryPart: "wr=7"},
	}

	for _, tc := range cases {
		item := RoleItem{
			Type:        "背包",
			Name:        tc.name,
			ItemType:    "equip",
			Display:     tc.display,
			Description: "f_i_" + tc.name + "^ffffff" + tc.descriptionPart + "&25@1&21@20&22@战士&3@17&5@30",
			Count:       1,
			Index:       -1,
			ItemLevel:   1,
		}
		granted, ok := store.GrantRoleItem(login.PlayerID, createResponse.Role.RoleID, item)
		if !ok {
			t.Fatalf("expected granting %s to bag to succeed", tc.name)
		}
		result := store.EquipRoleItem(login.PlayerID, createResponse.Role.RoleID, granted.Type, granted.Index, 1)
		if !result.Found || !result.Equipped {
			t.Fatalf("expected equipping %s to succeed, got %+v", tc.name, result)
		}
		if result.EquippedItem.Type != "装备" || result.EquippedItem.Index != tc.slot || result.EquippedItem.Name != tc.name {
			t.Fatalf("expected %s to equip into slot %d, got %+v", tc.name, tc.slot, result.EquippedItem)
		}
		if !strings.Contains(result.Role.SourceQuery, tc.sourceQueryPart) {
			t.Fatalf("expected %s equip to add %s to source query, got %q", tc.name, tc.sourceQueryPart, result.Role.SourceQuery)
		}
	}
}

func TestStoreRoleSourceQueryReflectsOnlyEquippedItems(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "清理外观女侠",
		Gender:         "female",
		RoleTemplateID: 1,
		SourceQuery:    "human/human.swf?a=4&w8=42&b=5&c=10&e=14&sex=1&hr=12&co=5&m=5&p=8&se=29&wr=7&",
	})

	store.rolesByPID[login.PlayerID][0].Items = normalizeRoleItems([]RoleItem{
		{
			Type:        "装备",
			Name:        "蛮力护腿",
			ItemType:    "equip",
			Display:     "338.png",
			Description: "f_i_蛮力护腿^ffffff&24@护具·腿部&25@1&21@20&22@战士",
			Count:       1,
			Index:       5,
		},
		{
			Type:        "背包",
			Name:        "刎刀",
			ItemType:    "equip",
			Display:     "42.png",
			Description: "f_i_刎刀^ffffff&24@武器·单刀系&25@1&21@20&22@战士",
			Count:       1,
			Index:       0,
		},
	})

	role, playerBase, ok := store.GetRoleRuntimeData(login.PlayerID, createResponse.Role.RoleID)
	if !ok {
		t.Fatal("expected role runtime data")
	}
	for _, stalePart := range []string{"w8=42", "a=4", "b=5", "c=10", "se=29", "wr=7"} {
		if strings.Contains(role.SourceQuery, stalePart) || strings.Contains(playerBase.SourceQuery, stalePart) {
			t.Fatalf("expected stale %s to be removed from source query, role=%q base=%q", stalePart, role.SourceQuery, playerBase.SourceQuery)
		}
	}
	if !strings.Contains(role.SourceQuery, "p=8") || !strings.Contains(playerBase.SourceQuery, "p=8") {
		t.Fatalf("expected equipped pants to remain in source query, role=%q base=%q", role.SourceQuery, playerBase.SourceQuery)
	}
}

func TestStoreRoleSourceQueryUsesCapturedBodyAppearance(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "白发女侠",
		Gender:         "female",
		RoleTemplateID: 1,
		SourceQuery:    "human/human.swf?e=6&sex=1&hr=12&co=5&m=0&w8=5&",
		Appearance: RoleAppearance{
			"body": map[string]any{
				"sex":       1,
				"skinColor": 5,
				"hair":      32,
				"eyes":      14,
				"mouth":     5,
			},
		},
	})

	role, playerBase, ok := store.GetRoleRuntimeData(login.PlayerID, createResponse.Role.RoleID)
	if !ok {
		t.Fatal("expected role runtime data")
	}
	for _, expectedPart := range []string{"hr=32", "e=14", "m=5", "sex=1", "co=5"} {
		if !strings.Contains(role.SourceQuery, expectedPart) || !strings.Contains(playerBase.SourceQuery, expectedPart) {
			t.Fatalf("expected captured body %s in source query, role=%q base=%q", expectedPart, role.SourceQuery, playerBase.SourceQuery)
		}
	}
	if strings.Contains(role.SourceQuery, "hr=12") || strings.Contains(role.SourceQuery, "e=6") || strings.Contains(role.SourceQuery, "m=0") {
		t.Fatalf("expected stale body params to be replaced, got %q", role.SourceQuery)
	}
}

func TestStorePurchaseRoleSkillRejectsInsufficientCurrencyWithoutMutation(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "余额不足女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	result := store.PurchaseRoleSkill(
		login.PlayerID,
		createResponse.Role.RoleID,
		RoleSkill{Name: "昂贵技能", Level: 1, Type: "被动技能", Icon: "999.png"},
		RoleCurrencies{"铜钱": 999999},
	)

	if !result.Found || result.Learned || result.ErrorCode != "not_enough_currency" {
		t.Fatalf("expected insufficient currency rejection, got %+v", result)
	}
	if result.Currencies["铜钱"] != 5000 {
		t.Fatalf("expected copper to remain 5000, got %+v", result.Currencies)
	}
	skills, _, ok := store.GetRoleSkills(login.PlayerID, createResponse.Role.RoleID)
	if !ok || len(skills) != 2 || skills[0].Name != "密斩" || skills[1].Name != "普通攻击" {
		t.Fatalf("expected default skill list to remain unchanged, got ok=%v skills=%+v", ok, skills)
	}
}

func TestStorePurchaseRoleSkillUpgradesDuplicateAndDeductsAgain(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "重复技能女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	skill := RoleSkill{Name: "武器专精", Level: 1, Type: "被动技能", Icon: "631.png"}

	first := store.PurchaseRoleSkill(login.PlayerID, createResponse.Role.RoleID, skill, RoleCurrencies{"铜钱": 500})
	second := store.PurchaseRoleSkill(login.PlayerID, createResponse.Role.RoleID, skill, RoleCurrencies{"铜钱": 500})

	if !first.Learned {
		t.Fatalf("expected first purchase success, got %+v", first)
	}
	if !second.Learned || second.ErrorCode != "" {
		t.Fatalf("expected duplicate purchase to upgrade, got %+v", second)
	}
	if second.Currencies["铜钱"] != 4000 {
		t.Fatalf("expected upgrade purchase to deduct again, got %+v", second.Currencies)
	}
	if len(second.Skills) != 3 || second.Skills[2].Name != "武器专精" || second.Skills[2].Level != 2 {
		t.Fatalf("expected duplicate purchase to raise skill to level 2, got %+v", second.Skills)
	}
}

func TestStorePurchaseRoleSkillRejectsAtMaxLevelWithoutDeduction(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "满级技能女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	skill := RoleSkill{Name: "挑衅", Level: 1, Type: "技能·通用", Icon: "634.png", MaxLevel: 1}

	first := store.PurchaseRoleSkill(login.PlayerID, createResponse.Role.RoleID, skill, RoleCurrencies{"铜钱": 500})
	second := store.PurchaseRoleSkill(login.PlayerID, createResponse.Role.RoleID, skill, RoleCurrencies{"铜钱": 500})

	if !first.Learned {
		t.Fatalf("expected first purchase success, got %+v", first)
	}
	if second.Learned || second.ErrorCode != "skill_level_max" {
		t.Fatalf("expected max-level purchase rejection, got %+v", second)
	}
	if second.Currencies["铜钱"] != 4500 {
		t.Fatalf("expected max-level rejection not to deduct again, got %+v", second.Currencies)
	}
}

func TestStoreRoleOperationsRejectInvalidSession(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	invalidSessionToken := login.SessionToken + "-invalid"

	listResponse := store.ListRoles(RoleListRequest{
		PlayerID:     login.PlayerID,
		SessionToken: invalidSessionToken,
	})
	assertInvalidSessionFailure(t, listResponse.Success, listResponse.ErrorCode, listResponse.ErrorMessage)

	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   invalidSessionToken,
		DisplayName:    "失败女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	assertInvalidSessionFailure(t, createResponse.Success, createResponse.ErrorCode, createResponse.ErrorMessage)
	if createResponse.Role.RoleID != "" {
		t.Fatalf("expected empty role on invalid create session, got %+v", createResponse.Role)
	}

	selectResponse := store.SelectRole(RoleSelectRequest{
		PlayerID:     login.PlayerID,
		SessionToken: invalidSessionToken,
		RoleID:       "mock-player-001-role-001",
	})
	assertInvalidSessionFailure(t, selectResponse.Success, selectResponse.ErrorCode, selectResponse.ErrorMessage)
	if selectResponse.PlayerBase.PlayerID != login.PlayerID {
		t.Fatalf("expected player base to preserve player id, got %+v", selectResponse.PlayerBase)
	}

	removeResponse := store.RemoveRole(RoleRemoveRequest{
		PlayerID:     login.PlayerID,
		SessionToken: invalidSessionToken,
		RoleID:       "mock-player-001-role-001",
		Password:     "magicpwd",
	})
	assertInvalidSessionFailure(t, removeResponse.Success, removeResponse.ErrorCode, removeResponse.ErrorMessage)
}

func TestPersistentStoreKeepsRolesAfterRestart(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ai-server.db")

	firstStore, err := NewPersistentStore(dbPath)
	if err != nil {
		t.Fatalf("expected persistent store init success, got error: %v", err)
	}

	loginResponse := firstStore.Login(LoginRequest{
		UserName: "persistuser",
		Password: "magicpwd",
	})
	if !loginResponse.Success {
		t.Fatalf("expected persistent login success, got %+v", loginResponse)
	}

	createResponse := firstStore.CreateRole(RoleCreateRequest{
		PlayerID:       loginResponse.PlayerID,
		SessionToken:   loginResponse.SessionToken,
		DisplayName:    "持久女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	if !createResponse.Success {
		t.Fatalf("expected create role success, got %+v", createResponse)
	}
	if err := firstStore.Close(); err != nil {
		t.Fatalf("expected first store close success, got error: %v", err)
	}

	secondStore, err := NewPersistentStore(dbPath)
	if err != nil {
		t.Fatalf("expected persistent store reopen success, got error: %v", err)
	}
	defer func() {
		if closeErr := secondStore.Close(); closeErr != nil {
			t.Fatalf("expected second store close success, got error: %v", closeErr)
		}
	}()

	secondLogin := secondStore.Login(LoginRequest{
		UserName: "persistuser",
		Password: "magicpwd",
	})
	if !secondLogin.Success {
		t.Fatalf("expected persistent relogin success, got %+v", secondLogin)
	}

	listResponse := secondStore.ListRoles(RoleListRequest{
		PlayerID:     loginResponse.PlayerID,
		SessionToken: secondLogin.SessionToken,
	})
	if len(listResponse.Roles) != 1 {
		t.Fatalf("expected one role after restart, got %d", len(listResponse.Roles))
	}
	if listResponse.Roles[0].RoleID != createResponse.Role.RoleID {
		t.Fatalf("expected role id %q after restart, got %q", createResponse.Role.RoleID, listResponse.Roles[0].RoleID)
	}
	if listResponse.Roles[0].DisplayName != "持久女侠" {
		t.Fatalf("expected persistent role name 持久女侠, got %q", listResponse.Roles[0].DisplayName)
	}
	items, capacity, ok := secondStore.GetRoleItems(loginResponse.PlayerID, createResponse.Role.RoleID, "背包")
	if !ok || capacity != 30 || len(items) != 1 || items[0].Name != "铁斧" || items[0].Index != 19 {
		t.Fatalf("expected persistent starter axe inventory after restart, ok=%v capacity=%d items=%+v", ok, capacity, items)
	}
}

func TestPersistentStoreUpdatingOnePlayerDoesNotEraseAnotherPlayer(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ai-server.db")

	store, err := NewPersistentStore(dbPath)
	if err != nil {
		t.Fatalf("expected persistent store init success, got error: %v", err)
	}

	firstLogin := store.Login(LoginRequest{
		UserName: "playerone",
		Password: "magicpwd",
	})
	secondLogin := store.Login(LoginRequest{
		UserName: "playertwo",
		Password: "magicpwd",
	})
	if !firstLogin.Success || !secondLogin.Success {
		t.Fatalf("expected both persistent logins success, got first=%+v second=%+v", firstLogin, secondLogin)
	}

	firstRole := store.CreateRole(RoleCreateRequest{
		PlayerID:       firstLogin.PlayerID,
		SessionToken:   firstLogin.SessionToken,
		DisplayName:    "一号女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	secondRole := store.CreateRole(RoleCreateRequest{
		PlayerID:       secondLogin.PlayerID,
		SessionToken:   secondLogin.SessionToken,
		DisplayName:    "二号女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	removeResponse := store.RemoveRole(RoleRemoveRequest{
		PlayerID:     secondLogin.PlayerID,
		SessionToken: secondLogin.SessionToken,
		RoleID:       secondRole.Role.RoleID,
		Password:     "magicpwd",
	})
	if !removeResponse.Success {
		t.Fatalf("expected second player remove success, got %+v", removeResponse)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("expected store close success, got error: %v", err)
	}

	reopenedStore, err := NewPersistentStore(dbPath)
	if err != nil {
		t.Fatalf("expected persistent store reopen success, got error: %v", err)
	}
	defer func() {
		if closeErr := reopenedStore.Close(); closeErr != nil {
			t.Fatalf("expected reopened store close success, got error: %v", closeErr)
		}
	}()

	reopenedFirstLogin := reopenedStore.Login(LoginRequest{
		UserName: "playerone",
		Password: "magicpwd",
	})
	reopenedSecondLogin := reopenedStore.Login(LoginRequest{
		UserName: "playertwo",
		Password: "magicpwd",
	})

	firstList := reopenedStore.ListRoles(RoleListRequest{
		PlayerID:     reopenedFirstLogin.PlayerID,
		SessionToken: reopenedFirstLogin.SessionToken,
	})
	if len(firstList.Roles) != 1 {
		t.Fatalf("expected first player role to survive restart, got %d roles", len(firstList.Roles))
	}
	if firstList.Roles[0].RoleID != firstRole.Role.RoleID {
		t.Fatalf("expected first player role id %q, got %q", firstRole.Role.RoleID, firstList.Roles[0].RoleID)
	}

	secondList := reopenedStore.ListRoles(RoleListRequest{
		PlayerID:     reopenedSecondLogin.PlayerID,
		SessionToken: reopenedSecondLogin.SessionToken,
	})
	if len(secondList.Roles) != 0 {
		t.Fatalf("expected second player roles to stay deleted, got %d roles", len(secondList.Roles))
	}
}

func TestStoreRemoveRoleSuccess(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")

	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "待删除女侠",
		Gender:         "female",
		RoleTemplateID: 1,
		PresetID:       4,
		SourceQuery:    "hair=4",
		Appearance: RoleAppearance{
			"body": map[string]any{
				"hair": "4",
			},
		},
	})

	removeResponse := store.RemoveRole(RoleRemoveRequest{
		PlayerID:     login.PlayerID,
		SessionToken: login.SessionToken,
		RoleID:       createResponse.Role.RoleID,
		Password:     "magicpwd",
	})

	if !removeResponse.Success {
		t.Fatalf("expected remove role success, got failure: %+v", removeResponse)
	}
	if removeResponse.RemovedRoleID != createResponse.Role.RoleID {
		t.Fatalf("expected removed role id %q, got %q", createResponse.Role.RoleID, removeResponse.RemovedRoleID)
	}
	if removeResponse.Message != "删除成功！" {
		t.Fatalf("expected remove role success message 删除成功！, got %q", removeResponse.Message)
	}

	listResponse := store.ListRoles(RoleListRequest{
		PlayerID:     login.PlayerID,
		SessionToken: login.SessionToken,
	})
	if len(listResponse.Roles) != 0 {
		t.Fatalf("expected no roles after delete, got %d", len(listResponse.Roles))
	}
}

func TestStoreRemoveRoleWrongPassword(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")

	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "保留女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	removeResponse := store.RemoveRole(RoleRemoveRequest{
		PlayerID:     login.PlayerID,
		SessionToken: login.SessionToken,
		RoleID:       createResponse.Role.RoleID,
		Password:     "wrongpwd",
	})

	if removeResponse.Success {
		t.Fatalf("expected remove role failure, got success: %+v", removeResponse)
	}
	if removeResponse.ErrorCode != "3" {
		t.Fatalf("expected wrong password error code 3, got %q", removeResponse.ErrorCode)
	}

	listResponse := store.ListRoles(RoleListRequest{
		PlayerID:     login.PlayerID,
		SessionToken: login.SessionToken,
	})
	if len(listResponse.Roles) != 1 {
		t.Fatalf("expected role to remain after failed delete, got %d", len(listResponse.Roles))
	}
}

func TestStoreCreateRoleIDDoesNotReuseAfterDelete(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")

	first := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "甲",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	second := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "乙",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	third := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "丙",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	if first.Role.RoleID == second.Role.RoleID || second.Role.RoleID == third.Role.RoleID {
		t.Fatalf("expected created roles to have unique ids, got %q %q %q", first.Role.RoleID, second.Role.RoleID, third.Role.RoleID)
	}

	removeResponse := store.RemoveRole(RoleRemoveRequest{
		PlayerID:     login.PlayerID,
		SessionToken: login.SessionToken,
		RoleID:       second.Role.RoleID,
		Password:     "magicpwd",
	})
	if !removeResponse.Success {
		t.Fatalf("expected remove role success, got failure: %+v", removeResponse)
	}
	if removeResponse.Message != "删除成功！" {
		t.Fatalf("expected remove role success message 删除成功！, got %q", removeResponse.Message)
	}

	fourth := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "丁",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	if fourth.Role.RoleID == first.Role.RoleID ||
		fourth.Role.RoleID == second.Role.RoleID ||
		fourth.Role.RoleID == third.Role.RoleID {
		t.Fatalf("expected newly created role id to stay unique after delete, got reused id %q", fourth.Role.RoleID)
	}

	listResponse := store.ListRoles(RoleListRequest{
		PlayerID:     login.PlayerID,
		SessionToken: login.SessionToken,
	})
	if len(listResponse.Roles) != 3 {
		t.Fatalf("expected three roles after delete and recreate, got %d", len(listResponse.Roles))
	}
}

func TestStoreListRolesNormalizesDuplicatedRoleIDs(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")

	store.rolesByPID[login.PlayerID] = []RoleSummary{
		{
			RoleID:       "mock-player-001-role-001",
			DisplayName:  "甲",
			Level:        1,
			MapID:        1,
			VisualRoleID: 1,
		},
		{
			RoleID:       "mock-player-001-role-001",
			DisplayName:  "乙",
			Level:        1,
			MapID:        1,
			VisualRoleID: 1,
		},
	}

	listResponse := store.ListRoles(RoleListRequest{
		PlayerID:     login.PlayerID,
		SessionToken: login.SessionToken,
	})
	if len(listResponse.Roles) != 2 {
		t.Fatalf("expected two roles after normalization, got %d", len(listResponse.Roles))
	}
	if listResponse.Roles[0].RoleID == listResponse.Roles[1].RoleID {
		t.Fatalf("expected duplicate role ids to be repaired, got %q and %q", listResponse.Roles[0].RoleID, listResponse.Roles[1].RoleID)
	}
}

func TestStoreRemoveRoleRemovesAllDuplicatedMatches(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")

	store.rolesByPID[login.PlayerID] = []RoleSummary{
		{
			RoleID:       "mock-player-001-role-001",
			DisplayName:  "甲",
			Level:        1,
			MapID:        1,
			VisualRoleID: 1,
		},
		{
			RoleID:       "mock-player-001-role-001",
			DisplayName:  "乙",
			Level:        1,
			MapID:        1,
			VisualRoleID: 1,
		},
	}

	removeResponse := store.RemoveRole(RoleRemoveRequest{
		PlayerID:     login.PlayerID,
		SessionToken: login.SessionToken,
		RoleID:       "mock-player-001-role-001",
		Password:     "magicpwd",
	})

	if !removeResponse.Success {
		t.Fatalf("expected remove role success, got failure: %+v", removeResponse)
	}

	listResponse := store.ListRoles(RoleListRequest{
		PlayerID:     login.PlayerID,
		SessionToken: login.SessionToken,
	})
	if len(listResponse.Roles) != 0 {
		t.Fatalf("expected duplicated roles to be fully removed, got %d roles", len(listResponse.Roles))
	}
}

func mustLogin(t *testing.T, store *Store, userName string, password string) LoginResponse {
	t.Helper()

	response := store.Login(LoginRequest{
		UserName: userName,
		Password: password,
	})
	if !response.Success {
		t.Fatalf("expected login success, got %+v", response)
	}

	return response
}

func assertInvalidSessionFailure(t *testing.T, success bool, errorCode string, errorMessage string) {
	t.Helper()

	if success {
		t.Fatalf("expected invalid session failure, got success")
	}
	if errorCode != "6" {
		t.Fatalf("expected invalid session error code 6, got %q", errorCode)
	}
	if errorMessage != "登录状态已失效，请重新登录。" {
		t.Fatalf("expected invalid session message, got %q", errorMessage)
	}
}
