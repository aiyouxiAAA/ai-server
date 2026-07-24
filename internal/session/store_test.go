package session

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"ai-server/internal/classicdata"
	"ai-server/internal/mall"
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
	if selectResponse.PlayerBase.Currencies["铜钱"] != 5000 || selectResponse.PlayerBase.Currencies["银元宝"] != 1 || selectResponse.PlayerBase.Currencies["玉币"] != 1 {
		t.Fatalf("expected default role currencies including 玉币, got %+v", selectResponse.PlayerBase.Currencies)
	}
}

func TestStoreDungeonInstancePersistsForOneHour(t *testing.T) {
	for _, instanceKey := range []string{DungeonInstanceShuiliandong, DungeonInstanceHuangfengzhai, DungeonInstanceFeixiandong, DungeonInstanceShihuku} {
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

func TestStoreTeamDungeonInstanceSharesProgressForOneHour(t *testing.T) {
	store := NewStore()
	now := time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }

	created, ok := store.EnsureTeamDungeonInstance("team-0001", DungeonInstanceShuiliandong)
	if !ok || created.CreatedAtUnix != now.Unix() {
		t.Fatalf("expected team dungeon instance, got ok=%v state=%+v", ok, created)
	}
	defeated, ok := store.MarkTeamDungeonVisibleMonsterDefeated("team-0001", DungeonInstanceShuiliandong, "5172206909807859")
	if !ok || len(defeated.DefeatedVisibleMonsterHandles) != 1 {
		t.Fatalf("expected team defeated monster state, got ok=%v state=%+v", ok, defeated)
	}

	now = now.Add(59 * time.Minute)
	shared, ok := store.GetTeamDungeonInstance("team-0001", DungeonInstanceShuiliandong)
	if !ok || shared.CreatedAtUnix != created.CreatedAtUnix || len(shared.DefeatedVisibleMonsterHandles) != 1 {
		t.Fatalf("expected shared team state before expiry, got ok=%v state=%+v", ok, shared)
	}

	now = now.Add(2 * time.Minute)
	if state, ok := store.GetTeamDungeonInstance("team-0001", DungeonInstanceShuiliandong); ok || state.CreatedAtUnix != 0 {
		t.Fatalf("expected expired team instance to be pruned, got ok=%v state=%+v", ok, state)
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

func TestStoreCapturedWoodcutter222UsesCapturedSkillsAndFastPanel(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "222",
		Gender:         "male",
		RoleTemplateID: 1,
	})

	skills, cap, ok := store.GetRoleSkills(login.PlayerID, createResponse.Role.RoleID)
	if !ok || cap != 12 || len(skills) != 12 {
		t.Fatalf("expected captured woodcutter skill list, ok=%v cap=%d skills=%+v", ok, cap, skills)
	}
	byName := map[string]RoleSkill{}
	for _, skill := range skills {
		byName[skill.Name] = skill
	}
	if _, ok := byName["密斩"]; ok {
		t.Fatalf("expected 222 to use captured ranger skills instead of default 密斩, got %+v", skills)
	}
	if _, ok := byName["多段刺"]; ok {
		t.Fatalf("expected final captured woodcutter state to have replaced 多段刺 with 奥义.暗杀者, got %+v", skills)
	}
	if skill := byName["奥义.暗杀者"]; skill.Level != 1 || skill.Type != "oneE" || skill.Icon != "262.png" || !strings.Contains(skill.Description, "需要3格魂元") {
		t.Fatalf("expected captured 奥义.暗杀者 skill info, got %+v", skill)
	}
	if skill := byName["强力飞镖"]; skill.Level != 3 || skill.Icon != "261.png" || !strings.Contains(skill.Description, "&2@24") {
		t.Fatalf("expected captured 强力飞镖 Lv3 info, got %+v", skill)
	}

	fastPanel, ok := store.GetRoleFastPanel(login.PlayerID, createResponse.Role.RoleID)
	if !ok || len(fastPanel) != 9 {
		t.Fatalf("expected captured fast panel, ok=%v fastPanel=%+v", ok, fastPanel)
	}
	expectedSlots := map[int]RoleFastPanelEntry{
		0: {Index: 0, Type: "skill", Name: "普通攻击"},
		1: {Index: 1, Type: "skill", Name: "强力飞镖"},
		2: {Index: 2, Type: "skill", Name: "奥义.暗杀者"},
		3: {Index: 3, Type: "skill", Name: "投毒"},
		4: {Index: 4, Type: "skill", Name: "疾风刺"},
		5: {Index: 5, Type: "skill", Name: "解毒术"},
		6: {Index: 6, Type: "skill", Name: "魔力突刺"},
		8: {Index: 8, Type: "item", Name: "馒头"},
		9: {Index: 9, Type: "item", Name: "小瓶甘露"},
	}
	for _, entry := range fastPanel {
		expected, ok := expectedSlots[entry.Index]
		if !ok || expected != entry {
			t.Fatalf("expected captured fast panel slots %+v, got %+v", expectedSlots, fastPanel)
		}
	}
}

func TestStoreCapturedWoodcutter333UsesCapturedLatestRuntime(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "33333333", "33333333")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "333",
		Gender:         "male",
		RoleTemplateID: 1,
	})
	if !createResponse.Success {
		t.Fatalf("expected 333 role create success, got %+v", createResponse)
	}

	role, playerBase, ok := store.GetRoleRuntimeData(login.PlayerID, createResponse.Role.RoleID)
	if !ok {
		t.Fatal("expected 333 captured role runtime data")
	}
	if role.DisplayName != "333" || playerBase.DisplayName != "333" {
		t.Fatalf("expected 333 account to keep local role name 333, role=%q base=%q", role.DisplayName, playerBase.DisplayName)
	}
	if role.Level != 50 || role.Exp != 8338018 || role.Voc != "游侠" ||
		role.AGI != 183 || role.STR != 102 || role.INT != 16 || role.CON != 2 || role.LCK != 10 {
		t.Fatalf("expected captured latest woodcutter summary, got %+v", role)
	}
	if role.MapID != 15 || playerBase.MapID != 15 {
		t.Fatalf("expected captured final map 15, role=%d base=%d", role.MapID, playerBase.MapID)
	}
	if playerBase.RoleState == nil || playerBase.RoleState.HP != 1463 || playerBase.RoleState.MP != 603 ||
		playerBase.RoleState.Lv != 50 || playerBase.RoleState.Exp != 8338018 || playerBase.RoleState.Speed != 150 {
		t.Fatalf("expected captured latest role state, got %+v", playerBase.RoleState)
	}
	if playerBase.RolePhysique == nil || playerBase.RolePhysique.MaxHP != 1535 || playerBase.RolePhysique.MaxMP != 739 ||
		playerBase.RolePhysique.PhyAtk != 357 || playerBase.RolePhysique.MgcAtk != 357 ||
		playerBase.RolePhysique.PhyDef != 299 || playerBase.RolePhysique.MgcDef != 77 ||
		playerBase.RolePhysique.Hit != 565 || playerBase.RolePhysique.Dog != 265 || playerBase.RolePhysique.Fat != 597 {
		t.Fatalf("expected captured latest role physique, got %+v", playerBase.RolePhysique)
	}
	for _, part := range []string{"a=29", "b=22", "c=26", "p=64", "se=19", "wr=39", "w1=55"} {
		if !strings.Contains(role.SourceQuery, part) || !strings.Contains(playerBase.BattleSourceQuery, part) {
			t.Fatalf("expected captured latest source query to include %s, role=%q base=%q", part, role.SourceQuery, playerBase.BattleSourceQuery)
		}
	}
	if strings.Contains(role.SourceQuery, "w3=49") || strings.Contains(playerBase.BattleSourceQuery, "w3=49") {
		t.Fatalf("expected captured latest source query to drop stale dagger appearance, role=%q base=%q", role.SourceQuery, playerBase.BattleSourceQuery)
	}

	skills, cap, ok := store.GetRoleSkills(login.PlayerID, createResponse.Role.RoleID)
	if !ok || cap != 12 || len(skills) != 12 {
		t.Fatalf("expected captured skills for 333, ok=%v cap=%d skills=%+v", ok, cap, skills)
	}
	expectedSkillOrder := []string{
		"普通攻击",
		"武器娴熟",
		"灵力进修",
		"精神力",
		"爆发力",
		"幻影",
		"贯甲连矢",
		"暗影箭",
		"奥义.轰雷矢",
		"毒矢",
		"魔力速射",
		"冰箭速射",
	}
	for index, expectedName := range expectedSkillOrder {
		if skills[index].Name != expectedName {
			t.Fatalf("expected 333 captured skill order %v, got %+v", expectedSkillOrder, skills)
		}
	}
	byName := map[string]RoleSkill{}
	for _, skill := range skills {
		byName[skill.Name] = skill
	}
	for _, staleName := range []string{"强射", "强力飞镖", "奥义.暗杀者", "投毒", "疾风刺", "解毒术", "魔力突刺"} {
		if _, ok := byName[staleName]; ok {
			t.Fatalf("expected 333 skillInfo to follow latest bow capture without stale dagger skill %s, got %+v", staleName, skills)
		}
	}
	if skill := byName["贯甲连矢"]; skill.Level != 5 || skill.Icon != "236.png" || !strings.Contains(skill.Description, "&2@28") {
		t.Fatalf("expected 333 captured final 贯甲连矢 Lv5, got %+v", skill)
	}
	if skill := byName["暗影箭"]; skill.Level != 1 || skill.Icon != "235.png" || !strings.Contains(skill.Description, "暗影箭x1") {
		t.Fatalf("expected 333 captured 暗影箭 Lv1, got %+v", skill)
	}
	if skill := byName["毒矢"]; skill.Level != 1 || skill.Icon != "237.png" || !strings.Contains(skill.Description, "毒箭x1") {
		t.Fatalf("expected 333 captured 毒矢 Lv1, got %+v", skill)
	}
	if skill := byName["奥义.轰雷矢"]; skill.Level != 1 || skill.Icon != "238.png" || !strings.Contains(skill.Description, "需要2格魂元") {
		t.Fatalf("expected 333 captured 奥义.轰雷矢 Lv1, got %+v", skill)
	}
	if skill := byName["魔力速射"]; skill.Level != 5 || skill.Icon != "234.png" || !strings.Contains(skill.Description, "魔箭x1") || !strings.Contains(skill.Description, "&2@34") {
		t.Fatalf("expected 333 captured 魔力速射 Lv5, got %+v", skill)
	}
	if skill := byName["冰箭速射"]; skill.Level != 5 || skill.Icon != "233.png" || !strings.Contains(skill.Description, "冰之箭x1") || !strings.Contains(skill.Description, "90%的机率") {
		t.Fatalf("expected 333 captured 冰箭速射 Lv5, got %+v", skill)
	}
	fastPanel, ok := store.GetRoleFastPanel(login.PlayerID, createResponse.Role.RoleID)
	fastPanelByIndex := map[int]RoleFastPanelEntry{}
	for _, entry := range fastPanel {
		fastPanelByIndex[entry.Index] = entry
	}
	if !ok || len(fastPanel) != 9 ||
		fastPanelByIndex[1].Name != "贯甲连矢" ||
		fastPanelByIndex[2].Name != "魔力速射" ||
		fastPanelByIndex[3].Name != "暗影箭" ||
		fastPanelByIndex[4].Name != "毒矢" ||
		fastPanelByIndex[5].Name != "奥义.轰雷矢" ||
		fastPanelByIndex[6].Name != "冰箭速射" ||
		fastPanelByIndex[8].Name != "馒头" ||
		fastPanelByIndex[9].Name != "小瓶甘露" {
		t.Fatalf("expected 333 captured fast panel, ok=%v fastPanel=%+v", ok, fastPanel)
	}
	bagItems, _, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "背包")
	if !ok {
		t.Fatal("expected 333 captured bag items")
	}
	piercingArrow, ok := testRoleItemByName(bagItems, "穿甲箭")
	if !ok || piercingArrow.Count != 1319 || piercingArrow.Display != "246.png" {
		t.Fatalf("expected 333 captured bag to include 穿甲箭 for 贯甲连矢, ok=%v item=%+v bag=%+v", ok, piercingArrow, bagItems)
	}
	darkArrow, ok := testRoleItemByName(bagItems, "暗之箭")
	if !ok || darkArrow.Count != 39 || darkArrow.Display != "245.png" {
		t.Fatalf("expected 333 captured bag to include 暗之箭 for 暗影箭, ok=%v item=%+v bag=%+v", ok, darkArrow, bagItems)
	}
	poisonArrow, ok := testRoleItemByName(bagItems, "毒箭")
	if !ok || poisonArrow.Count != 9 || poisonArrow.Display != "247.png" {
		t.Fatalf("expected 333 captured bag to include 毒箭 for 毒矢, ok=%v item=%+v bag=%+v", ok, poisonArrow, bagItems)
	}
	iceArrow, ok := testRoleItemByName(bagItems, "冰之箭")
	if !ok || iceArrow.Count != 50 || iceArrow.Display != "243.png" {
		t.Fatalf("expected 333 captured bag to include 冰之箭 for 冰箭速射, ok=%v item=%+v bag=%+v", ok, iceArrow, bagItems)
	}
	magicArrow, ok := testRoleItemByName(bagItems, "魔箭")
	if !ok || magicArrow.Count != 50 || magicArrow.Display != "244.png" {
		t.Fatalf("expected 333 captured bag to include 魔箭 for 魔力速射, ok=%v item=%+v bag=%+v", ok, magicArrow, bagItems)
	}
}

func TestStoreCapturedWoodcutter777UsesCurrentFullEquipment(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "77777777", "77777777")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "777",
		Gender:         "male",
		RoleTemplateID: 1,
	})
	if !createResponse.Success {
		t.Fatalf("expected 777 role create success, got %+v", createResponse)
	}

	role, playerBase, ok := store.GetRoleRuntimeData(login.PlayerID, createResponse.Role.RoleID)
	if !ok {
		t.Fatal("expected 777 captured role runtime data")
	}
	if role.DisplayName != "777" || playerBase.DisplayName != "777" || role.Level != 50 || role.Exp != 8378018 || role.MapID != 171 {
		t.Fatalf("expected current captured 777 runtime summary, role=%+v base=%+v", role, playerBase)
	}
	if playerBase.RoleState == nil || playerBase.RoleState.HP != 1410 || playerBase.RoleState.MP != 695 ||
		playerBase.RoleState.Exp != 8378018 || playerBase.RoleState.Lv != 50 || playerBase.RoleState.Speed != 150 {
		t.Fatalf("expected current captured 777 role state, got %+v", playerBase.RoleState)
	}
	if playerBase.RolePhysique == nil || playerBase.RolePhysique.PhyAtk != 358 || playerBase.RolePhysique.MgcAtk != 30 ||
		playerBase.RolePhysique.PhyDef != 302 || playerBase.RolePhysique.MgcDef != 80 {
		t.Fatalf("expected current captured 777 role physique, got %+v", playerBase.RolePhysique)
	}
	skills, cap, ok := store.GetRoleSkills(login.PlayerID, createResponse.Role.RoleID)
	if !ok || cap != 12 {
		t.Fatalf("expected 777 captured fist skills, ok=%v cap=%d skills=%+v", ok, cap, skills)
	}
	expectedSkills := []struct {
		name  string
		level int
	}{
		{name: "普通攻击", level: 1},
		{name: "武器娴熟", level: 5},
		{name: "灵力进修", level: 4},
		{name: "精神力", level: 5},
		{name: "爆发力", level: 5},
		{name: "幻影", level: 4},
		{name: "连击", level: 5},
		{name: "重烈", level: 5},
		{name: "气运丹田", level: 5},
		{name: "破魂打", level: 5},
		{name: "移形换影", level: 4},
		{name: "奥义.修罗幻翼拳", level: 4},
	}
	if len(skills) != len(expectedSkills) {
		t.Fatalf("expected 777 captured fist skill count %d, got %+v", len(expectedSkills), skills)
	}
	for index, expected := range expectedSkills {
		if skills[index].Name != expected.name || skills[index].Level != expected.level {
			t.Fatalf("expected 777 captured fist skill %d to be %+v, got %+v", index, expected, skills[index])
		}
	}
	for _, legacyBowSkill := range []string{"贯甲连矢", "暗影箭", "奥义.轰雷矢", "毒矢", "魔力速射", "冰箭速射"} {
		for _, skill := range skills {
			if skill.Name == legacyBowSkill {
				t.Fatalf("expected 777 fist snapshot to replace legacy bow skill %q, got %+v", legacyBowSkill, skills)
			}
		}
	}
	fastPanel, ok := store.GetRoleFastPanel(login.PlayerID, createResponse.Role.RoleID)
	if !ok || len(fastPanel) != 9 {
		t.Fatalf("expected complete 777 fist fast panel, ok=%v fastPanel=%+v", ok, fastPanel)
	}
	expectedFastPanel := map[int]string{
		0: "普通攻击", 1: "连击", 2: "气运丹田", 3: "破魂打", 4: "移形换影", 5: "重烈", 6: "奥义.修罗幻翼拳", 8: "馒头", 9: "小瓶甘露",
	}
	for _, entry := range fastPanel {
		if expectedName, found := expectedFastPanel[entry.Index]; !found || entry.Name != expectedName {
			t.Fatalf("expected 777 fist fast panel %+v, got %+v", expectedFastPanel, fastPanel)
		}
	}
	for _, part := range []string{"w5=64", "c=61", "p=63", "se=49", "hr=12"} {
		if !strings.Contains(role.SourceQuery, part) || !strings.Contains(playerBase.BattleSourceQuery, part) {
			t.Fatalf("expected 777 source query to include %s, role=%q base=%q", part, role.SourceQuery, playerBase.BattleSourceQuery)
		}
	}

	equipment, capacity, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "装备")
	if !ok || capacity != 20 || len(equipment) != 13 {
		t.Fatalf("expected full captured 777 equipment, ok=%v capacity=%d equipment=%+v", ok, capacity, equipment)
	}
	expectedByIndex := map[int]string{
		0: "黄风围巾", 1: "蚩颅王护肩", 2: "炎爆护手", 3: "武雷拳套", 4: "珍元胸甲",
		5: "机木护腿", 6: "骷髅戒指", 7: "银耳坠", 8: "翡翠项链", 9: "炎火兽",
		10: "锁纹护腰", 11: "普通礼服", 12: "炎爆之靴",
	}
	for _, item := range equipment {
		if expectedName, found := expectedByIndex[item.Index]; !found || item.Name != expectedName {
			t.Fatalf("expected full captured 777 equipment %+v, got %+v", expectedByIndex, equipment)
		}
	}
	bagItems, bagCapacity, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "背包")
	if !ok || bagCapacity != 42 || len(bagItems) != 38 {
		t.Fatalf("expected full captured 777 bag, ok=%v capacity=%d items=%+v", ok, bagCapacity, bagItems)
	}
	bagByIndex := map[int]RoleItem{}
	for _, item := range bagItems {
		bagByIndex[item.Index] = item
	}
	for _, expected := range []struct {
		index int
		name  string
		count int
	}{
		{index: 2, name: "传送晶体", count: 1},
		{index: 10, name: "小瓶甘露", count: 49},
		{index: 19, name: "小包还元散", count: 62},
		{index: 21, name: "穿甲箭", count: 6851},
		{index: 25, name: "L小喇叭", count: 1165},
		{index: 30, name: "宠物用营养水", count: 366},
		{index: 38, name: "盗贼护臂", count: 1},
	} {
		item, found := bagByIndex[expected.index]
		if !found || item.Name != expected.name || item.Count != expected.count {
			t.Fatalf("expected captured 777 bag index %d to be %s x%d, got %+v", expected.index, expected.name, expected.count, item)
		}
	}
	currencies, ok := store.GetRoleCurrencies(login.PlayerID, createResponse.Role.RoleID)
	if !ok || currencies["银元宝"] != 189 || currencies["铜钱"] != 1278 || len(currencies) != 2 {
		t.Fatalf("expected captured 777 currencies, ok=%v currencies=%+v", ok, currencies)
	}
}

func TestStorePersistentCapturedWoodcutter777ReplacesLegacyBowSkills(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ai-server.db")
	firstStore, err := NewPersistentStore(dbPath)
	if err != nil {
		t.Fatalf("expected persistent store: %v", err)
	}
	login := firstStore.Login(LoginRequest{UserName: "77777777", Password: "77777777"})
	created := firstStore.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "777",
		Gender:         "male",
		RoleTemplateID: 1,
	})
	if !created.Success {
		t.Fatalf("expected captured 777 role create success, got %+v", created)
	}
	legacySkillsJSON, err := encodeRoleSkills(capturedWoodcutter333RoleSkills())
	if err != nil {
		t.Fatalf("encode legacy bow skills: %v", err)
	}
	legacyFastPanelJSON, err := encodeRoleFastPanel(capturedWoodcutter333FastPanel())
	if err != nil {
		t.Fatalf("encode legacy bow fast panel: %v", err)
	}
	legacyItems := append(capturedWoodcutter777EquipmentItems(), capturedWoodcutter777BagItems()...)
	for index := range legacyItems {
		if legacyItems[index].Type == "背包" && legacyItems[index].Name == "银元宝" && legacyItems[index].Index == 0 {
			legacyItems[index].Count = defaultSilver
		}
	}
	legacyItemsJSON, err := encodeRoleItems(legacyItems)
	if err != nil {
		t.Fatalf("encode stale 777 bag: %v", err)
	}
	capturedCurrenciesJSON, err := encodeRoleCurrencies(capturedWoodcutter777Currencies())
	if err != nil {
		t.Fatalf("encode captured 777 currencies: %v", err)
	}
	if _, err := firstStore.db.Exec(
		`UPDATE roles SET skills_json = ?, fast_panel_json = ?, items_json = ?, currencies_json = ? WHERE role_id = ? AND player_id = ?`,
		legacySkillsJSON,
		legacyFastPanelJSON,
		legacyItemsJSON,
		capturedCurrenciesJSON,
		created.Role.RoleID,
		login.PlayerID,
	); err != nil {
		t.Fatalf("seed legacy 777 bow snapshot: %v", err)
	}
	if err := firstStore.Close(); err != nil {
		t.Fatalf("close first persistent store: %v", err)
	}

	reopened, err := NewPersistentStore(dbPath)
	if err != nil {
		t.Fatalf("reopen persistent store: %v", err)
	}
	defer reopened.Close()
	skills, _, ok := reopened.GetRoleSkills(login.PlayerID, created.Role.RoleID)
	if !ok || len(skills) != 12 || skills[6].Name != "连击" || skills[6].Level != 5 ||
		skills[10].Name != "移形换影" || skills[10].Level != 4 ||
		skills[11].Name != "奥义.修罗幻翼拳" || skills[11].Level != 4 {
		t.Fatalf("expected persistent 777 fist snapshot after legacy bow migration, ok=%v skills=%+v", ok, skills)
	}
	fastPanel, ok := reopened.GetRoleFastPanel(login.PlayerID, created.Role.RoleID)
	if !ok || len(fastPanel) != 9 || fastPanel[0].Name != "普通攻击" ||
		fastPanel[1].Name != "连击" || fastPanel[2].Name != "气运丹田" ||
		fastPanel[3].Name != "破魂打" || fastPanel[4].Name != "移形换影" ||
		fastPanel[5].Name != "重烈" || fastPanel[6].Name != "奥义.修罗幻翼拳" ||
		fastPanel[7].Name != "馒头" || fastPanel[8].Name != "小瓶甘露" {
		t.Fatalf("expected persistent 777 fist fast panel, ok=%v fastPanel=%+v", ok, fastPanel)
	}
	var persistedFastPanelJSON string
	if err := reopened.db.QueryRow(
		`SELECT fast_panel_json FROM roles WHERE role_id = ? AND player_id = ?`,
		created.Role.RoleID,
		login.PlayerID,
	).Scan(&persistedFastPanelJSON); err != nil {
		t.Fatalf("read persisted 777 fast panel: %v", err)
	}
	persistedFastPanel, err := decodeRoleFastPanel(persistedFastPanelJSON)
	if err != nil || len(persistedFastPanel) != 9 ||
		persistedFastPanel[0].Name != "普通攻击" ||
		persistedFastPanel[1].Name != "连击" ||
		persistedFastPanel[6].Name != "奥义.修罗幻翼拳" ||
		persistedFastPanel[7].Name != "馒头" ||
		persistedFastPanel[8].Name != "小瓶甘露" {
		t.Fatalf("expected SQLite to persist 777 fist fast panel, err=%v fastPanel=%+v", err, persistedFastPanel)
	}
	bagItems, _, ok := reopened.GetRoleItems(login.PlayerID, created.Role.RoleID, "背包")
	if !ok || len(bagItems) != 38 {
		t.Fatalf("expected persistent captured 777 bag replacement, ok=%v items=%+v", ok, bagItems)
	}
	for _, legacyArrowName := range []string{"暗之箭", "毒箭", "火之箭", "冰之箭", "魔箭"} {
		if _, found := testRoleItemByName(bagItems, legacyArrowName); found {
			t.Fatalf("expected persistent 777 bag to remove legacy %s, items=%+v", legacyArrowName, bagItems)
		}
	}
	teleportCrystal, found := testRoleItemByName(bagItems, "传送晶体")
	if !found || teleportCrystal.Index != 2 || teleportCrystal.Count != 1 {
		t.Fatalf("expected persistent 777 captured teleport crystal, found=%t item=%+v", found, teleportCrystal)
	}
	silver, found := testRoleItemByName(bagItems, "银元宝")
	if !found || silver.Index != 0 || silver.Count != 189 {
		t.Fatalf("expected persistent 777 captured silver stack, found=%t item=%+v", found, silver)
	}
	currencies, ok := reopened.GetRoleCurrencies(login.PlayerID, created.Role.RoleID)
	if !ok || currencies["银元宝"] != 189 || currencies["铜钱"] != 1278 || len(currencies) != 2 {
		t.Fatalf("expected persistent 777 captured currencies, ok=%v currencies=%+v", ok, currencies)
	}
}

func TestCapturedWoodcutter333RuntimeDefaultsBackfillBowMaterialsAndEquipment(t *testing.T) {
	role := withRoleRuntimeDefaults(RoleSummary{
		RoleID:      "acct-333-role-legacy",
		DisplayName: "333",
		Items: []RoleItem{{
			Type:        "背包",
			Name:        "飞镖",
			ItemType:    "null",
			Display:     "241.png",
			Description: "f_i_飞镖^ffffff&24@材料 消耗品&25@9999&20@铁制三刃飞镖&0;具有杀伤力&0;一般配合技能使用.&27@sitem_jwep&103@0&104@0&105@&107@&108@0",
			Count:       7892,
			Index:       22,
			ItemLevel:   1,
		}},
	})
	if _, ok := testRoleItemByName(role.Items, "飞镖"); !ok {
		t.Fatalf("expected existing 333 bag items to be preserved, got %+v", role.Items)
	}
	if role.RolePhysique == nil || role.RolePhysique.PhyAtk != 357 || role.RolePhysique.MgcAtk != role.RolePhysique.PhyAtk {
		t.Fatalf("expected 333 runtime defaults to mirror magic attack to physical attack, got %+v", role.RolePhysique)
	}
	piercingArrow, ok := testRoleItemByName(role.Items, "穿甲箭")
	if !ok || piercingArrow.Type != "背包" || piercingArrow.Count != 1319 || piercingArrow.Display != "246.png" {
		t.Fatalf("expected legacy 333 bag to backfill 穿甲箭 for 贯甲连矢, ok=%v item=%+v items=%+v", ok, piercingArrow, role.Items)
	}
	darkArrow, ok := testRoleItemByName(role.Items, "暗之箭")
	if !ok || darkArrow.Type != "背包" || darkArrow.Count != 39 || darkArrow.Display != "245.png" {
		t.Fatalf("expected legacy 333 bag to backfill 暗之箭 for 暗影箭, ok=%v item=%+v items=%+v", ok, darkArrow, role.Items)
	}
	poisonArrow, ok := testRoleItemByName(role.Items, "毒箭")
	if !ok || poisonArrow.Type != "背包" || poisonArrow.Count != 9 || poisonArrow.Display != "247.png" {
		t.Fatalf("expected legacy 333 bag to backfill 毒箭 for 毒矢, ok=%v item=%+v items=%+v", ok, poisonArrow, role.Items)
	}
	byName := map[string]RoleItem{}
	for _, item := range role.Items {
		if item.Type == "装备" {
			byName[item.Name] = item
		}
	}
	for name, expected := range map[string]struct {
		index   int
		display string
	}{
		"万相":   {index: 3, display: "58.png"},
		"骷髅戒指": {index: 6, display: "759.png"},
		"银耳坠":  {index: 7, display: "762.png"},
		"翡翠项链": {index: 8, display: "760.png"},
	} {
		item := byName[name]
		if item.Name == "" || item.Index != expected.index || item.Display != expected.display {
			t.Fatalf("expected legacy 333 equipment %s at %d display %s, got %+v", name, expected.index, expected.display, item)
		}
	}
}

func TestCapturedWoodcutter333LatestEquipmentTemplates(t *testing.T) {
	for name, expected := range map[string]struct {
		index           int
		display         string
		descriptionPart string
	}{
		"万相":   {index: 3, display: "58.png", descriptionPart: "武器·弓系"},
		"骷髅戒指": {index: 6, display: "759.png", descriptionPart: "封印抗性:+10%"},
		"银耳坠":  {index: 7, display: "762.png", descriptionPart: "眩晕抗性:+10%"},
		"翡翠项链": {index: 8, display: "760.png", descriptionPart: "混乱抗性:+10%"},
	} {
		template, ok := CapturedRoleItemTemplate(name)
		if !ok {
			t.Fatalf("expected captured 333 equipment template for %s", name)
		}
		if template.Type != "装备" || template.ItemType != "equip" || template.Index != expected.index || template.Display != expected.display {
			t.Fatalf("expected %s template at equipment index %d display %s, got %+v", name, expected.index, expected.display, template)
		}
		if !strings.Contains(template.Description, expected.descriptionPart) {
			t.Fatalf("expected %s template description to include %q, got %q", name, expected.descriptionPart, template.Description)
		}
	}
}

func TestStoreCapturedWarrior444UsesCapturedLevel44Runtime(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "44444444", "44444444")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "444",
		Gender:         "male",
		RoleTemplateID: 1,
	})
	if !createResponse.Success {
		t.Fatalf("expected 444 role create success, got %+v", createResponse)
	}

	role, playerBase, ok := store.GetRoleRuntimeData(login.PlayerID, createResponse.Role.RoleID)
	if !ok {
		t.Fatal("expected 444 captured warrior runtime data")
	}
	if role.DisplayName != "444" || playerBase.DisplayName != "444" {
		t.Fatalf("expected 444 account to keep local role name 444, role=%q base=%q", role.DisplayName, playerBase.DisplayName)
	}
	if role.Level != 44 || role.Exp != 5793804 || role.Voc != "战士" ||
		role.AGI != 76 || role.STR != 168 || role.INT != 5 || role.CON != 0 || role.LCK != 0 {
		t.Fatalf("expected captured level 44 warrior summary, got %+v", role)
	}
	if role.MapID != 1 || playerBase.MapID != 1 {
		t.Fatalf("expected captured final map 1, role=%d base=%d", role.MapID, playerBase.MapID)
	}
	if playerBase.RoleState == nil || playerBase.RoleState.HP != 1344 || playerBase.RoleState.MP != 390 ||
		playerBase.RoleState.Lv != 44 || playerBase.RoleState.Exp != 5793804 || playerBase.RoleState.Speed != 147 {
		t.Fatalf("expected captured warrior role state, got %+v", playerBase.RoleState)
	}
	if playerBase.RolePhysique == nil || playerBase.RolePhysique.MaxHP != 1775 || playerBase.RolePhysique.MaxMP != 524 ||
		playerBase.RolePhysique.PhyAtk != 341 || playerBase.RolePhysique.MgcAtk != 5 ||
		playerBase.RolePhysique.PhyDef != 223 || playerBase.RolePhysique.MgcDef != 73 ||
		playerBase.RolePhysique.Hit != 350 || playerBase.RolePhysique.Dog != 156 || playerBase.RolePhysique.Fat != 289 {
		t.Fatalf("expected captured warrior physique, got %+v", playerBase.RolePhysique)
	}
	for _, part := range []string{"a=32", "b=21", "c=39", "p=22", "se=29", "w11=53", "wr=19"} {
		if !strings.Contains(role.SourceQuery, part) || !strings.Contains(playerBase.BattleSourceQuery, part) {
			t.Fatalf("expected captured warrior source query to include %s, role=%q base=%q", part, role.SourceQuery, playerBase.BattleSourceQuery)
		}
	}
	equipment, equipmentCapacity, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "装备")
	if !ok || equipmentCapacity != 20 || len(equipment) != 13 {
		t.Fatalf("expected captured level 44 warrior equipment, ok=%v capacity=%d equipment=%+v", ok, equipmentCapacity, equipment)
	}
	warriorEquipmentByName := map[string]RoleItem{}
	for _, item := range equipment {
		warriorEquipmentByName[item.Name] = item
	}
	for name, expected := range map[string]struct {
		index   int
		display string
	}{
		"狼人护肩": {index: 1, display: "537.png"},
		"骷髅戒指": {index: 6, display: "759.png"},
		"银耳坠":  {index: 7, display: "762.png"},
		"翡翠项链": {index: 8, display: "760.png"},
		"泥戒指":  {index: 13, display: "431.png"},
	} {
		item := warriorEquipmentByName[name]
		if item.Name == "" || item.Index != expected.index || item.Display != expected.display || item.Type != "装备" || item.ItemType != "equip" {
			t.Fatalf("expected captured warrior equipment %s at index %d display %s, got %+v", name, expected.index, expected.display, item)
		}
	}
	if _, ok := warriorEquipmentByName["龙颜单肩"]; ok {
		t.Fatalf("expected captured level 44 equipment to replace old 龙颜单肩, got %+v", equipment)
	}

	skills, cap, ok := store.GetRoleSkills(login.PlayerID, createResponse.Role.RoleID)
	if !ok || cap != 12 {
		t.Fatalf("expected captured skills for 444, ok=%v cap=%d skills=%+v", ok, cap, skills)
	}
	byName := map[string]RoleSkill{}
	for _, skill := range skills {
		byName[skill.Name] = skill
	}
	if skill := byName["劈山棍法"]; skill.Level != 5 || skill.Icon != "185.png" || !strings.Contains(skill.Description, "&2@16") {
		t.Fatalf("expected 444 captured 劈山棍法 Lv5, got %+v", skill)
	}
	if skill := byName["盘龙棍法"]; skill.Level != 2 || skill.Type != "all" || skill.Icon != "187.png" {
		t.Fatalf("expected 444 captured 盘龙棍法 Lv2, got %+v", skill)
	}
	if skill := byName["奥义.六合棍法"]; skill.Level != 1 || skill.Icon != "190.png" {
		t.Fatalf("expected 444 captured 奥义.六合棍法, got %+v", skill)
	}
	fastPanel, ok := store.GetRoleFastPanel(login.PlayerID, createResponse.Role.RoleID)
	if !ok || len(fastPanel) != 8 || fastPanel[1].Name != "劈山棍法" || fastPanel[5].Name != "奥义.六合棍法" || fastPanel[7].Name != "小瓶甘露" {
		t.Fatalf("expected 444 captured fast panel, ok=%v fastPanel=%+v", ok, fastPanel)
	}
}

func TestStoreCapturedWoodcutter222UsesCapturedEquipmentAndAppearance(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "222",
		Gender:         "male",
		RoleTemplateID: 1,
	})

	equipment, capacity, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "装备")
	if !ok || capacity != 20 || len(equipment) != 10 {
		t.Fatalf("expected captured woodcutter equipment, ok=%v capacity=%d equipment=%+v", ok, capacity, equipment)
	}
	byName := map[string]RoleItem{}
	for _, item := range equipment {
		byName[item.Name] = item
	}
	expectedEquipment := map[string]struct {
		index   int
		display string
	}{
		"黄风围巾":   {index: 0, display: "548.png"},
		"蚩颅王护肩":  {index: 1, display: "484.png"},
		"黄风护腕":   {index: 2, display: "549.png"},
		"绯雨匕首":   {index: 3, display: "51.png"},
		"神风护甲":   {index: 4, display: "366.png"},
		"神风护腿":   {index: 5, display: "368.png"},
		"炎火兽":    {index: 9, display: "324.png"},
		"神风护腰":   {index: 10, display: "369.png"},
		"神风战靴":   {index: 12, display: "370.png"},
		"L千年人参果": {index: 15, display: "921.png"},
	}
	for name, expected := range expectedEquipment {
		item := byName[name]
		if item.Name == "" || item.Index != expected.index || item.Display != expected.display || item.Type != "装备" || item.ItemType != "equip" {
			t.Fatalf("expected captured equipment %s at index %d display %s, got %+v", name, expected.index, expected.display, item)
		}
	}
	if !strings.Contains(byName["绯雨匕首"].Description, "内伤状态3回合") {
		t.Fatalf("expected captured dagger description, got %+v", byName["绯雨匕首"])
	}
	if !strings.Contains(byName["L千年人参果"].Description, "剩余精力【8440】") || byName["L千年人参果"].Owner != "桥头的樵夫" {
		t.Fatalf("expected captured treasure equipment data, got %+v", byName["L千年人参果"])
	}

	role, playerBase, ok := store.GetRoleRuntimeData(login.PlayerID, createResponse.Role.RoleID)
	if !ok {
		t.Fatal("expected captured role runtime data")
	}
	for _, part := range []string{"h=30", "a=29", "wr=25", "w3=49", "c=17", "p=16", "b=14", "se=12", "sex=1", "hr=12"} {
		if !strings.Contains(role.SourceQuery, part) || !strings.Contains(playerBase.SourceQuery, part) {
			t.Fatalf("expected captured woodcutter source query to include %s, role=%q base=%q", part, role.SourceQuery, playerBase.SourceQuery)
		}
	}
	if strings.Contains(role.SourceQuery, "w3=43") || strings.Contains(playerBase.SourceQuery, "w3=43") {
		t.Fatalf("expected captured final equipment to replace stale w3=43, role=%q base=%q", role.SourceQuery, playerBase.SourceQuery)
	}
	if role.BattleSourceQuery != role.SourceQuery || playerBase.BattleSourceQuery != playerBase.SourceQuery {
		t.Fatalf("expected battle source query to sync captured appearance, role=%q/%q base=%q/%q", role.SourceQuery, role.BattleSourceQuery, playerBase.SourceQuery, playerBase.BattleSourceQuery)
	}
}

func TestStorePersistentCapturedWoodcutter222KeepsCapturedSkills(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ai-server.db")
	firstStore, err := NewPersistentStore(dbPath)
	if err != nil {
		t.Fatalf("expected persistent store: %v", err)
	}
	login := firstStore.Login(LoginRequest{
		UserName: "cap1366655383",
		Password: "local-test-only",
	})
	createResponse := firstStore.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "222",
		Gender:         "male",
		RoleTemplateID: 1,
	})
	if !createResponse.Success {
		t.Fatalf("expected captured role create success, got %+v", createResponse)
	}
	if err := firstStore.Close(); err != nil {
		t.Fatalf("expected close persistent store: %v", err)
	}

	reopened, err := NewPersistentStore(dbPath)
	if err != nil {
		t.Fatalf("expected reopen persistent store: %v", err)
	}
	defer reopened.Close()
	skills, _, ok := reopened.GetRoleSkills(login.PlayerID, createResponse.Role.RoleID)
	if !ok {
		t.Fatal("expected reopened captured role skills")
	}
	byName := map[string]RoleSkill{}
	for _, skill := range skills {
		byName[skill.Name] = skill
	}
	if _, ok := byName["奥义.暗杀者"]; !ok {
		t.Fatalf("expected reopened 222 to keep captured 奥义.暗杀者, got %+v", skills)
	}
	if _, ok := byName["密斩"]; ok {
		t.Fatalf("expected reopened 222 not to fall back to default skills, got %+v", skills)
	}
	equipment, _, ok := reopened.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "装备")
	if !ok || len(equipment) != 10 {
		t.Fatalf("expected reopened 222 to keep captured equipment, ok=%v equipment=%+v", ok, equipment)
	}
	role, playerBase, ok := reopened.GetRoleRuntimeData(login.PlayerID, createResponse.Role.RoleID)
	if !ok || !strings.Contains(role.SourceQuery, "w3=49") || !strings.Contains(playerBase.BattleSourceQuery, "w3=49") {
		t.Fatalf("expected reopened 222 to keep captured appearance, ok=%v role=%+v base=%+v", ok, role, playerBase)
	}
}

func TestStorePersistentCapturedWoodcutter333KeepsLatestSnapshot(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ai-server.db")
	firstStore, err := NewPersistentStore(dbPath)
	if err != nil {
		t.Fatalf("expected persistent store: %v", err)
	}
	login := firstStore.Login(LoginRequest{
		UserName: "33333333",
		Password: "33333333",
	})
	createResponse := firstStore.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "333",
		Gender:         "male",
		RoleTemplateID: 1,
	})
	if !createResponse.Success {
		t.Fatalf("expected captured 333 role create success, got %+v", createResponse)
	}
	if err := firstStore.Close(); err != nil {
		t.Fatalf("expected close persistent store: %v", err)
	}

	reopened, err := NewPersistentStore(dbPath)
	if err != nil {
		t.Fatalf("expected reopen persistent store: %v", err)
	}
	defer reopened.Close()
	role, playerBase, ok := reopened.GetRoleRuntimeData(login.PlayerID, createResponse.Role.RoleID)
	if !ok {
		t.Fatal("expected reopened 333 captured role")
	}
	if role.DisplayName != "333" || role.Level != 50 || role.Exp != 8338018 || role.Voc != "游侠" {
		t.Fatalf("expected reopened 333 to keep captured latest role, got %+v", role)
	}
	if playerBase.RoleState == nil || playerBase.RoleState.HP != 1463 || playerBase.RoleState.MP != 603 || playerBase.RoleState.Lv != 50 {
		t.Fatalf("expected reopened 333 to keep captured role state, got %+v", playerBase.RoleState)
	}
	if playerBase.RolePhysique == nil || playerBase.RolePhysique.MaxHP != 1535 || playerBase.RolePhysique.MaxMP != 739 ||
		playerBase.RolePhysique.PhyAtk != 357 || playerBase.RolePhysique.MgcAtk != 357 {
		t.Fatalf("expected reopened 333 to keep captured role physique, got %+v", playerBase.RolePhysique)
	}
	if !strings.Contains(role.SourceQuery, "p=64") || !strings.Contains(role.SourceQuery, "w1=55") || strings.Contains(role.SourceQuery, "w3=49") || role.BattleSourceQuery != role.SourceQuery {
		t.Fatalf("expected reopened 333 to keep captured appearance, role=%q battle=%q", role.SourceQuery, role.BattleSourceQuery)
	}
	skills, cap, ok := reopened.GetRoleSkills(login.PlayerID, createResponse.Role.RoleID)
	if !ok || cap != 12 || len(skills) != 12 || skills[6].Name != "贯甲连矢" || skills[6].Level != 5 || skills[7].Name != "暗影箭" || skills[8].Name != "奥义.轰雷矢" || skills[9].Name != "毒矢" || skills[10].Name != "魔力速射" || skills[11].Name != "冰箭速射" {
		t.Fatalf("expected reopened 333 to keep captured final skills, ok=%v cap=%d skills=%+v", ok, cap, skills)
	}
	fastPanel, ok := reopened.GetRoleFastPanel(login.PlayerID, createResponse.Role.RoleID)
	fastPanelByIndex := map[int]RoleFastPanelEntry{}
	for _, entry := range fastPanel {
		fastPanelByIndex[entry.Index] = entry
	}
	if !ok || fastPanelByIndex[1].Name != "贯甲连矢" || fastPanelByIndex[2].Name != "魔力速射" ||
		fastPanelByIndex[3].Name != "暗影箭" || fastPanelByIndex[4].Name != "毒矢" || fastPanelByIndex[5].Name != "奥义.轰雷矢" || fastPanelByIndex[6].Name != "冰箭速射" {
		t.Fatalf("expected reopened 333 to keep captured final fast panel, ok=%v fastPanel=%+v", ok, fastPanel)
	}
}

func TestStorePersistentCapturedWarrior444KeepsLevel44Snapshot(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ai-server.db")
	firstStore, err := NewPersistentStore(dbPath)
	if err != nil {
		t.Fatalf("expected persistent store: %v", err)
	}
	login := firstStore.Login(LoginRequest{
		UserName: "44444444",
		Password: "44444444",
	})
	createResponse := firstStore.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "444",
		Gender:         "male",
		RoleTemplateID: 1,
	})
	if !createResponse.Success {
		t.Fatalf("expected captured 444 role create success, got %+v", createResponse)
	}
	if err := firstStore.Close(); err != nil {
		t.Fatalf("expected close persistent store: %v", err)
	}

	reopened, err := NewPersistentStore(dbPath)
	if err != nil {
		t.Fatalf("expected reopen persistent store: %v", err)
	}
	defer reopened.Close()
	role, playerBase, ok := reopened.GetRoleRuntimeData(login.PlayerID, createResponse.Role.RoleID)
	if !ok {
		t.Fatal("expected reopened 444 captured role")
	}
	if role.DisplayName != "444" || role.Level != 44 || role.Exp != 5793804 || role.Voc != "战士" {
		t.Fatalf("expected reopened 444 to keep captured level 44 role, got %+v", role)
	}
	if playerBase.RoleState == nil || playerBase.RoleState.HP != 1344 || playerBase.RoleState.MP != 390 || playerBase.RoleState.Lv != 44 {
		t.Fatalf("expected reopened 444 to keep captured role state, got %+v", playerBase.RoleState)
	}
	if playerBase.RolePhysique == nil || playerBase.RolePhysique.MaxHP != 1775 || playerBase.RolePhysique.MaxMP != 524 || playerBase.RolePhysique.PhyAtk != 341 {
		t.Fatalf("expected reopened 444 to keep captured role physique, got %+v", playerBase.RolePhysique)
	}
	if !strings.Contains(role.SourceQuery, "a=32") || !strings.Contains(role.SourceQuery, "w11=53") || role.BattleSourceQuery != role.SourceQuery {
		t.Fatalf("expected reopened 444 to keep captured appearance, role=%q battle=%q", role.SourceQuery, role.BattleSourceQuery)
	}
	equipment, _, ok := reopened.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "装备")
	if !ok || len(equipment) != 13 {
		t.Fatalf("expected reopened 444 to keep captured level 44 equipment, ok=%v equipment=%+v", ok, equipment)
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

	warehouseCapacity, ok := store.GetRoleContainerCapacity(login.PlayerID, createResponse.Role.RoleID, "\u4ed3\u5e93")
	if !ok || warehouseCapacity != 40 {
		t.Fatalf("expected default warehouse capacity 40, got ok=%v capacity=%d", ok, warehouseCapacity)
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

func TestStorePurchaseCapturedMallFashionProductRequiresYubi(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "商城女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	product, ok := store.Mall.FindProduct("8019")
	if !ok {
		t.Fatal("expected captured mall fashion product")
	}
	result := store.PurchaseMallProduct(login.PlayerID, createResponse.Role.RoleID, product, 1, "captured-fashion")
	// New roles seed a small development balance, but captured fashion rows
	// remain far above that seed so purchase must still reject.
	if result.Success || result.ErrorCode != "INSUFFICIENT_CURRENCY" || result.CurrencyName != "玉币" || result.CurrencyBalance >= product.Price {
		t.Fatalf("expected captured mall fashion purchase to fail on insufficient 玉币 balance, got %+v", result)
	}

	items, _, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "商城")
	if !ok || len(items) != 0 {
		t.Fatalf("expected failed captured fashion purchase not to write 商城 items, ok=%v items=%+v", ok, items)
	}

	store.rolesByPID[login.PlayerID][0].Currencies["玉币"] = product.Price
	result = store.PurchaseMallProduct(login.PlayerID, createResponse.Role.RoleID, product, 1, "captured-fashion-success")
	if !result.Success {
		t.Fatalf("expected captured fashion purchase success, got %+v", result)
	}
	items, _, ok = store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "商城")
	if !ok || len(items) != 1 {
		t.Fatalf("expected successful captured fashion purchase in 商城, ok=%v items=%+v", ok, items)
	}
	item := items[0]
	if item.Name != "超时空要塞" || item.Type != "商城" || item.ItemType != "equip" || item.ItemLevel != 3 || item.Owner != "" {
		t.Fatalf("expected captured fashion template semantics in 商城, got %+v", item)
	}
}

func TestStorePurchaseCapturedMallBundleUsesInnerItem(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "商城礼包女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	product, ok := store.Mall.FindProduct("9011")
	if !ok {
		t.Fatal("expected captured 高级聚气碟.特x5 product")
	}
	store.rolesByPID[login.PlayerID][0].Currencies["玉币"] = product.Price
	result := store.PurchaseMallProduct(login.PlayerID, createResponse.Role.RoleID, product, 1, "captured-bundle")
	if !result.Success {
		t.Fatalf("expected captured bundle purchase success, got %+v", result)
	}

	items, _, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "商城")
	if !ok || len(items) != 1 {
		t.Fatalf("expected one purchased bundle item, ok=%v items=%+v", ok, items)
	}
	item := items[0]
	if item.Name != "高级聚气碟.特" || item.Display != "1724.png" || item.Count != 5 || item.Description != product.Items[0].Description {
		t.Fatalf("expected product inner item to enter 商城 container, got %+v", item)
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

func TestStoreGetRolePetInfoUsesEquippedCapturedPet(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "222",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	info := store.GetRolePetInfo(login.PlayerID, createResponse.Role.RoleID)
	if !info.Found || !info.HasPet {
		t.Fatalf("expected equipped pet info, got %+v", info)
	}
	if info.Name != "炎火兽" || info.PetType != "炎火兽" || info.Level != 5 || info.Fullness != 100 {
		t.Fatalf("expected captured fire pet level/fullness, got %+v", info)
	}
	if info.DisplayURL != "petmap/yhs1.swf" || !strings.Contains(info.SkillHTML, "喜好食物") {
		t.Fatalf("expected captured pet display/skill info, got %+v", info)
	}
}

func TestStoreFeedRolePetConsumesNutritionWaterAndRefreshesPetInfo(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "222",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	water, ok := CapturedRoleItemTemplate("宠物用营养水")
	if !ok {
		t.Fatal("expected captured nutrition water template")
	}
	water.Type = "背包"
	water.Index = 0
	water.Count = 2
	store.rolesByPID[login.PlayerID][0].Items = append(store.rolesByPID[login.PlayerID][0].Items, water)
	equipment, _, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "装备")
	if !ok || len(equipment) == 0 {
		t.Fatalf("expected default captured equipment, got ok=%v equipment=%+v", ok, equipment)
	}

	before := store.GetRolePetInfo(login.PlayerID, createResponse.Role.RoleID)
	result := store.FeedRolePet(login.PlayerID, createResponse.Role.RoleID, "背包", 0, 1)
	if !result.Found || !result.Fed {
		t.Fatalf("expected pet feed success, got %+v", result)
	}
	if result.Level != before.Level || result.Exp != before.Exp+5 || result.Fullness != 100 {
		t.Fatalf("expected water to add growth 5 and clamp fullness, before=%+v after=%+v", before, result.RolePetInfoResult)
	}
	if result.UpdatedItem == nil || result.UpdatedItem.Name != "宠物用营养水" || result.UpdatedItem.Count != 1 {
		t.Fatalf("expected remaining nutrition water stack, got %+v", result.UpdatedItem)
	}
	items, _, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "背包")
	if !ok {
		t.Fatal("expected bag items after feed")
	}
	remaining, ok := findRoleItem(items, "背包", 0)
	if !ok || remaining.Count != 1 {
		t.Fatalf("expected persisted water count 1, ok=%v item=%+v items=%+v", ok, remaining, items)
	}
}

func TestStoreRoleInventoryPreservesCapturedFullInventoryDefaultLikeSlots(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "抓包背包女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	seedItems := capturedDefaultRoleItems()
	var defaultLike RoleItem
	for _, item := range seedItems {
		if item.Name == "L避怪符" {
			defaultLike = item
			break
		}
	}
	if defaultLike.Name == "" {
		t.Fatal("expected L避怪符 seed item")
	}
	fullInventoryItem := RoleItem{
		Type:        "背包",
		Name:        "王花花蕾",
		ItemType:    "null",
		Display:     "143.png",
		Description: "f_i_王花花蕾^ffffff&24@材料&25@99&20@花蕾&103@0&104@0&105@&107@&108@0",
		Count:       1,
		Index:       36,
		ItemLevel:   1,
	}
	store.rolesByPID[login.PlayerID][0].Items = []RoleItem{defaultLike, fullInventoryItem}

	items, capacity, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "背包")
	if !ok {
		t.Fatal("expected bag items")
	}
	if capacity != 42 {
		t.Fatalf("expected captured index 36 to expand visible bag capacity to 42, got %d", capacity)
	}
	if _, ok := findRoleItem(items, "背包", defaultLike.Index); !ok {
		t.Fatalf("expected default-like item in captured full inventory to remain, got %+v", items)
	}
	if item, ok := findRoleItem(items, "背包", fullInventoryItem.Index); !ok || item.Name != fullInventoryItem.Name {
		t.Fatalf("expected full inventory item to remain, ok=%v item=%+v", ok, item)
	}
	for _, item := range items {
		if item.Name == "铁斧" {
			t.Fatalf("expected captured full inventory without starter axe pollution, got %+v", items)
		}
	}
}

func TestStoreRemoveExpiredTownBuffsClearsAvoidBuff(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "town-buff-expiry",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	item, ok := CapturedRoleItemTemplate("\u004c\u907f\u602a\u7b26")
	if !ok {
		t.Fatal("expected captured avoid buff item template")
	}
	item.Type = "\u80cc\u5305"
	item.Index = -1
	item.Count = 1
	granted, ok := store.GrantRoleItem(login.PlayerID, createResponse.Role.RoleID, item)
	if !ok {
		t.Fatal("expected avoid buff item grant")
	}

	useResult := store.UseRoleItem(login.PlayerID, createResponse.Role.RoleID, granted.Type, granted.Index)
	if !useResult.Found || !useResult.Used || useResult.TownBuff == nil {
		t.Fatalf("expected avoid buff to be applied, got %+v", useResult)
	}

	removeResult := store.RemoveExpiredRoleTownBuffs(
		login.PlayerID,
		createResponse.Role.RoleID,
		useResult.TownBuff.EndTime+1,
	)
	if !removeResult.Found || !removeResult.Removed || len(removeResult.Buffs) != 1 {
		t.Fatalf("expected expired town buff removal, got %+v", removeResult)
	}
	if removeResult.Buffs[0].Name != "\u907f\u602a" || removeResult.Buffs[0].Display != "574.png" {
		t.Fatalf("expected expired avoid buff details, got %+v", removeResult.Buffs[0])
	}

	buffs, ok := store.GetRoleTownBuffs(login.PlayerID, createResponse.Role.RoleID)
	if !ok || len(buffs) != 0 {
		t.Fatalf("expected expired town buff to be removed, ok=%v buffs=%+v", ok, buffs)
	}
}

func TestStoreUseRoleItemAppliesCapturedInitialExperienceBuff(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "initial-exp-card",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	item, ok := CapturedRoleItemTemplate("\u004c\u521d\u9636\u7ecf\u9a8c\u5361")
	if !ok {
		t.Fatal("expected captured initial experience card template")
	}
	item.Type = "\u80cc\u5305"
	item.Index = -1
	item.Count = 5
	granted, ok := store.GrantRoleItem(login.PlayerID, createResponse.Role.RoleID, item)
	if !ok {
		t.Fatal("expected initial experience card grant")
	}

	before := time.Now().UnixMilli()
	result := store.UseRoleItem(login.PlayerID, createResponse.Role.RoleID, granted.Type, granted.Index)
	after := time.Now().UnixMilli()
	if !result.Found || !result.Used || result.TownBuff == nil {
		t.Fatalf("expected initial experience card to apply a town buff, got %+v", result)
	}
	buff := result.TownBuff
	if buff.Name != "\u53cc\u500d\u7ecf\u9a8c" || buff.Display != "567.png" || buff.Description != "\u5728\u6218\u6597\u4e2d\u83b7\u5f97\u53cc\u500d\u7684\u7ecf\u9a8c" {
		t.Fatalf("expected captured double-exp buff fields, got %+v", buff)
	}
	if buff.BattleOnly != 0 || !buff.Partial || !strings.Contains(buff.SourceCapture, "ActiveItemByIndex(114)") {
		t.Fatalf("expected captured partial double-exp buff evidence, got %+v", buff)
	}
	minEnd := before + int64(time.Hour/time.Millisecond) - 1000
	maxEnd := after + int64(time.Hour/time.Millisecond) + 1000
	if buff.EndTime < minEnd || buff.EndTime > maxEnd {
		t.Fatalf("expected captured one-hour double-exp buff, got end=%d min=%d max=%d", buff.EndTime, minEnd, maxEnd)
	}
	if len(result.UpdatedItems) != 1 || result.UpdatedItems[0].Name != "\u004c\u521d\u9636\u7ecf\u9a8c\u5361" || result.UpdatedItems[0].Count != 4 {
		t.Fatalf("expected initial experience card count refresh to 4, got %+v", result.UpdatedItems)
	}
	buffs, ok := store.GetRoleTownBuffs(login.PlayerID, createResponse.Role.RoleID)
	if !ok || len(buffs) != 1 || buffs[0].Name != "\u53cc\u500d\u7ecf\u9a8c" {
		t.Fatalf("expected persisted double-exp town buff, ok=%v buffs=%+v", ok, buffs)
	}
}

func TestStoreUseRoleItemAppliesCapturedAdvancedExperienceBuff(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "advanced-exp-card",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	item, ok := CapturedRoleItemTemplate("\u004c\u8fdb\u9636\u7ecf\u9a8c\u5361")
	if !ok {
		t.Fatal("expected captured advanced experience card template")
	}
	item.Type = "\u80cc\u5305"
	item.Index = -1
	item.Count = 4
	granted, ok := store.GrantRoleItem(login.PlayerID, createResponse.Role.RoleID, item)
	if !ok {
		t.Fatal("expected advanced experience card grant")
	}

	before := time.Now().UnixMilli()
	result := store.UseRoleItem(login.PlayerID, createResponse.Role.RoleID, granted.Type, granted.Index)
	after := time.Now().UnixMilli()
	if !result.Found || !result.Used || result.TownBuff == nil {
		t.Fatalf("expected advanced experience card to apply a town buff, got %+v", result)
	}
	buff := result.TownBuff
	if buff.Name != "\u53cc\u500d\u7ecf\u9a8c" || buff.Display != "567.png" || buff.Description != "\u5728\u6218\u6597\u4e2d\u83b7\u5f97\u53cc\u500d\u7684\u7ecf\u9a8c" {
		t.Fatalf("expected captured double-exp buff fields, got %+v", buff)
	}
	if buff.BattleOnly != 0 || !buff.Partial || !strings.Contains(buff.SourceCapture, "#2844/#2847/#2848") {
		t.Fatalf("expected captured partial advanced double-exp buff evidence, got %+v", buff)
	}
	minEnd := before + int64((3*time.Hour)/time.Millisecond) - 1000
	maxEnd := after + int64((3*time.Hour)/time.Millisecond) + 1000
	if buff.EndTime < minEnd || buff.EndTime > maxEnd {
		t.Fatalf("expected captured three-hour double-exp buff, got end=%d min=%d max=%d", buff.EndTime, minEnd, maxEnd)
	}
	if len(result.UpdatedItems) != 1 || result.UpdatedItems[0].Name != "\u004c\u8fdb\u9636\u7ecf\u9a8c\u5361" || result.UpdatedItems[0].Count != 3 {
		t.Fatalf("expected advanced experience card count refresh to 3, got %+v", result.UpdatedItems)
	}
	buffs, ok := store.GetRoleTownBuffs(login.PlayerID, createResponse.Role.RoleID)
	if !ok || len(buffs) != 1 || buffs[0].Name != "\u53cc\u500d\u7ecf\u9a8c" {
		t.Fatalf("expected persisted double-exp town buff, ok=%v buffs=%+v", ok, buffs)
	}
}

func TestStoreRoleInventoryCapacityFollowsCapturedExpandedIndexes(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "扩格背包女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	items := make([]RoleItem, 0, 31)
	for index := 0; index < defaultBagCap; index += 1 {
		items = append(items, RoleItem{
			Type:      "背包",
			Name:      "占位物",
			ItemType:  "own",
			Count:     1,
			Index:     index,
			ItemLevel: 1,
		})
	}
	items = append(items, RoleItem{
		Type:      "背包",
		Name:      "红缨",
		ItemType:  "null",
		Display:   "77.png",
		Count:     1,
		Index:     39,
		ItemLevel: 1,
	})
	store.rolesByPID[login.PlayerID][0].Items = items

	capacity, ok := store.GetRoleContainerCapacity(login.PlayerID, createResponse.Role.RoleID, "背包")
	if !ok || capacity != 42 {
		t.Fatalf("expected expanded bag capacity 42 from captured index 39, ok=%v capacity=%d", ok, capacity)
	}
	granted, ok := store.GrantRoleItem(login.PlayerID, createResponse.Role.RoleID, RoleItem{
		Type:      "背包",
		Name:      "新材料",
		ItemType:  "own",
		Count:     1,
		Index:     -1,
		ItemLevel: 1,
	})
	if !ok || granted.Index != 30 {
		t.Fatalf("expected grant to use first expanded empty slot 30, ok=%v item=%+v", ok, granted)
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

func TestStoreUseRoleItemRejectsLevel5GiftBoxBelowCapturedLevelGate(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "gift-box-level-gate",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	giftBox, ok := CapturedRoleItemTemplate("\u0035\u7ea7\u793c\u76d2")
	if !ok {
		t.Fatal("expected captured level 5 gift box template")
	}
	giftBox.Type = "\u80cc\u5305"
	giftBox.Index = -1
	giftBox.Count = 1
	granted, ok := store.GrantRoleItem(login.PlayerID, createResponse.Role.RoleID, giftBox)
	if !ok {
		t.Fatal("expected gift box grant")
	}

	result := store.UseRoleItem(login.PlayerID, createResponse.Role.RoleID, granted.Type, granted.Index)
	if !result.Found || result.Used || result.ErrorCode != "item_level_too_low" || result.ErrorMessage != "\u89d2\u8272\u7b49\u7ea7\u5fc5\u987b\u5230\u8fbeLv5" {
		t.Fatalf("expected captured level gate rejection, got %+v", result)
	}
	items, _, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "\u80cc\u5305")
	if !ok {
		t.Fatal("expected bag items")
	}
	for _, item := range items {
		if item.Name == "\u0035\u7ea7\u793c\u76d2" && item.Index == granted.Index && item.Count == 1 {
			return
		}
	}
	t.Fatalf("expected rejected gift box not to be consumed, got %+v", items)
}

func TestStoreRefineRoleEquipmentUsesTemporaryRuleAndPersists(t *testing.T) {
	persistencePath := filepath.Join(t.TempDir(), "equipment-refinement.db")
	store, err := NewPersistentStore(persistencePath)
	if err != nil {
		t.Fatalf("create persistent store: %v", err)
	}
	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Fatalf("close persistent store: %v", closeErr)
		}
	}()

	login := mustLogin(t, store, "mockuser", "magicpwd")
	created := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "精炼持久化测试",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	gem, ok := store.GrantRoleItem(login.PlayerID, created.Role.RoleID, RoleItem{
		Type:        "背包",
		Name:        "初级精炼宝石",
		ItemType:    "oneI",
		Display:     "616.png",
		Description: "f_i_初级精炼宝石&24@宝物&25@999",
		Count:       2,
		Index:       -1,
	})
	if !ok {
		t.Fatal("expected refinement gem grant")
	}
	target, ok := store.GrantRoleItem(login.PlayerID, created.Role.RoleID, RoleItem{
		Type:        "装备",
		Name:        "精炼测试护肩",
		ItemType:    "equip",
		Display:     "603.png",
		Description: "f_i_精炼测试护肩&24@护具·肩部&25@1&21@10&3@10(+3)&4@5(+2)&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+3\n[精炼+1] 每升一级 魔法防御+2",
		Count:       1,
		Index:       19,
		Level:       1,
		ItemLevel:   1,
	})
	if !ok {
		t.Fatal("expected refinement target grant")
	}

	result := store.RefineRoleEquipment(login.PlayerID, created.Role.RoleID, gem.Type, gem.Index, target.Type, target.Index)
	if !result.Found || !result.Refined || !result.Succeeded || result.ResultMessage != "精炼成功&0;等级上升1" {
		t.Fatalf("expected temporary refinement success, got %+v", result)
	}
	if result.TargetItem.Level != 2 || !strings.Contains(result.TargetItem.Description, "&3@10(+6)") || !strings.Contains(result.TargetItem.Description, "&4@5(+4)") {
		t.Fatalf("expected level two source-style stat contributions, got %+v", result.TargetItem)
	}
	if result.PlayerBase.RolePhysique == nil || roleEquipmentStats([]RoleItem{result.TargetItem}).phyDef != 16 || roleEquipmentStats([]RoleItem{result.TargetItem}).mgcDef != 9 {
		t.Fatalf("expected refinement contributions in role physique calculation, got %+v", result.PlayerBase.RolePhysique)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("close before reload: %v", err)
	}
	store, err = NewPersistentStore(persistencePath)
	if err != nil {
		t.Fatalf("reload persistent store: %v", err)
	}
	restoredEquipItems, _, ok := store.GetRoleItems(login.PlayerID, created.Role.RoleID, "装备")
	restoredTargetLevel := -1
	for _, item := range restoredEquipItems {
		if item.Name == "精炼测试护肩" {
			restoredTargetLevel = item.Level
		}
	}
	if !ok || restoredTargetLevel != 2 {
		t.Fatalf("expected persisted refined equipment, got %+v", restoredEquipItems)
	}
	restoredBagItems, _, ok := store.GetRoleItems(login.PlayerID, created.Role.RoleID, "背包")
	restoredGemCount := -1
	for _, item := range restoredBagItems {
		if item.Name == "初级精炼宝石" {
			restoredGemCount = item.Count
		}
	}
	if !ok || restoredGemCount != 1 {
		t.Fatalf("expected persisted consumed refinement gem, got %+v", restoredBagItems)
	}
}

func TestStoreRefineRoleEquipmentUsesTemporaryFailureRule(t *testing.T) {
	store := NewStore()
	store.refinementRoll = func(int) int { return 9999 }
	login := mustLogin(t, store, "mockuser", "magicpwd")
	created := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "精炼失败测试",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	gem, ok := store.GrantRoleItem(login.PlayerID, created.Role.RoleID, RoleItem{
		Type:        "背包",
		Name:        "初级精炼宝石",
		ItemType:    "oneI",
		Display:     "616.png",
		Description: "f_i_初级精炼宝石&24@宝物&25@999",
		Count:       1,
		Index:       -1,
	})
	if !ok {
		t.Fatal("expected refinement gem grant")
	}
	target, ok := store.GrantRoleItem(login.PlayerID, created.Role.RoleID, RoleItem{
		Type:        "装备",
		Name:        "精炼失败测试护肩",
		ItemType:    "equip",
		Display:     "603.png",
		Description: "f_i_精炼失败测试护肩&24@护具·肩部&25@1&21@10&3@10(+6)&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+3",
		Count:       1,
		Index:       19,
		Level:       2,
		ItemLevel:   1,
	})
	if !ok {
		t.Fatal("expected refinement target grant")
	}

	result := store.RefineRoleEquipment(login.PlayerID, created.Role.RoleID, gem.Type, gem.Index, target.Type, target.Index)
	if !result.Found || !result.Refined || result.Succeeded || result.ResultMessage != "精炼失败&0;等级下降1" {
		t.Fatalf("expected temporary refinement failure, got %+v", result)
	}
	if result.TargetItem.Level != 1 || !strings.Contains(result.TargetItem.Description, "&3@10(+3)") {
		t.Fatalf("expected failure to reduce temporary level and contribution, got %+v", result.TargetItem)
	}
	if len(result.ClearedItems) != 1 || result.ClearedItems[0].Type != gem.Type || result.ClearedItems[0].Index != gem.Index {
		t.Fatalf("expected failed attempt to consume its gem, got %+v", result.ClearedItems)
	}
}

func TestStoreCapturedWoodcutter222PreservesRefinementResult(t *testing.T) {
	store := NewStore()
	store.refinementRoll = func(int) int { return 0 }
	login := mustLogin(t, store, "mockuser", "magicpwd")
	created := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "222",
		Gender:         "male",
		RoleTemplateID: 1,
	})
	gem, ok := store.GrantRoleItem(login.PlayerID, created.Role.RoleID, RoleItem{
		Type:        "背包",
		Name:        "初级精炼宝石",
		ItemType:    "oneI",
		Display:     "616.png",
		Description: "f_i_初级精炼宝石&24@宝物&25@999",
		Count:       1,
		Index:       -1,
	})
	if !ok {
		t.Fatal("expected refinement gem grant")
	}

	result := store.RefineRoleEquipment(login.PlayerID, created.Role.RoleID, gem.Type, gem.Index, "装备", 1)
	if !result.Found || !result.Refined || !result.Succeeded || result.TargetItem.Level != 3 {
		t.Fatalf("expected captured 222 shoulder refinement success, got %+v", result)
	}
	equipment, _, ok := store.GetRoleItems(login.PlayerID, created.Role.RoleID, "装备")
	if !ok {
		t.Fatal("expected captured 222 equipment")
	}
	for _, item := range equipment {
		if item.Index == 1 {
			if item.Name != "蚩颅王护肩" || item.Level != 3 || !strings.Contains(item.Description, "&3@18(+9)") {
				t.Fatalf("expected refined captured shoulder to survive runtime defaults, got %+v", item)
			}
			return
		}
	}
	t.Fatalf("expected captured shoulder in equipment %+v", equipment)
}

func TestStoreUseRoleItemLearnsCapturedWeaponFamiliarityPresentation(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "weapon-familiarity-learn",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	skillBook, ok := CapturedRoleItemTemplate("\u6b66\u5668\u5a34\u719f")
	if !ok {
		t.Fatal("expected captured weapon familiarity skill book template")
	}
	skillBook.Type = "\u80cc\u5305"
	skillBook.Index = -1
	skillBook.Count = 1
	granted, ok := store.GrantRoleItem(login.PlayerID, createResponse.Role.RoleID, skillBook)
	if !ok {
		t.Fatal("expected skill book grant")
	}

	result := store.UseRoleItem(login.PlayerID, createResponse.Role.RoleID, granted.Type, granted.Index)
	if !result.Found || !result.Used || result.LearnedSkill == nil {
		t.Fatalf("expected captured skill book to learn a skill, got %+v", result)
	}
	if result.LearnedSkill.Name != "\u6b66\u5668\u5a34\u719f" || result.LearnedSkill.Level != 1 || result.LearnedSkill.Type != "null" || result.LearnedSkill.Icon != "226.png" {
		t.Fatalf("expected captured learned skill presentation, got %+v", result.LearnedSkill)
	}
	if result.LearnedSkill.Description != "f_s_\u6b66\u5668\u5a34\u719f^ffffff&9@\u88ab\u52a8&8@\u6e38\u4fa0 &10@\u901a\u7528&12@8" {
		t.Fatalf("expected captured learned skill description, got %q", result.LearnedSkill.Description)
	}
	if len(result.ClearedItems) != 1 || result.ClearedItems[0].Index != granted.Index {
		t.Fatalf("expected captured skill book to be consumed, got %+v", result.ClearedItems)
	}
}

func TestStoreUseRoleItemConsumesCapturedBagCapacityPatch(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "bag-capacity-patch",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	patch, ok := CapturedRoleItemTemplate("\u004c\u80cc\u5305\u8865\u4e01")
	if !ok {
		t.Fatal("expected captured bag capacity patch template")
	}
	patch.Type = "\u80cc\u5305"
	patch.Index = 7
	patch.Count = 1
	if _, ok := store.GrantRoleItem(login.PlayerID, createResponse.Role.RoleID, patch); !ok {
		t.Fatal("expected bag capacity patch grant")
	}

	result := store.UseRoleItem(login.PlayerID, createResponse.Role.RoleID, "\u80cc\u5305", 7)
	if !result.Found || !result.Used || result.ErrorCode != "" {
		t.Fatalf("expected captured bag capacity patch use, got %+v", result)
	}
	if result.ContainerType != "\u80cc\u5305" || result.ContainerCapacity != 30 {
		t.Fatalf("expected captured bag capacity result 30, got type=%s capacity=%d", result.ContainerType, result.ContainerCapacity)
	}
	if len(result.ClearedItems) != 1 || result.ClearedItems[0].Type != "\u80cc\u5305" || result.ClearedItems[0].Index != 7 {
		t.Fatalf("expected captured patch slot 7 clear, got %+v", result.ClearedItems)
	}
	items, capacity, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "\u80cc\u5305")
	if !ok || capacity != 30 {
		t.Fatalf("expected persisted bag capacity 30 after patch, ok=%v capacity=%d", ok, capacity)
	}
	for _, item := range items {
		if item.Name == "\u004c\u80cc\u5305\u8865\u4e01" {
			t.Fatalf("expected bag capacity patch to be consumed, got %+v", items)
		}
	}
}

func TestStoreUseRoleItemOpensCapturedLevel1GiftBox(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "level1-gift-open",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	store.rolesByPID[login.PlayerID][0].Items = testLevel1GiftBoxBagItems(t)

	result := store.UseRoleItem(login.PlayerID, createResponse.Role.RoleID, "\u80cc\u5305", 1)
	if !result.Found || !result.Used || result.ErrorCode != "" {
		t.Fatalf("expected captured level 1 gift box to open, got %+v", result)
	}
	if len(result.ClearedItems) != 1 || result.ClearedItems[0].Type != "\u80cc\u5305" || result.ClearedItems[0].Index != 1 {
		t.Fatalf("expected source gift box slot 1 clear, got %+v", result.ClearedItems)
	}
	updated := testRoleItemsByName(result.UpdatedItems)
	if updated["\u0035\u7ea7\u793c\u76d2"].Count != 1 || updated["\u0035\u7ea7\u793c\u76d2"].Index != 1 {
		t.Fatalf("expected level 5 gift box reward at freed slot 1, got %+v", result.UpdatedItems)
	}
	if updated["\u004c\u907f\u602a\u7b26"].Count != 3 || updated["\u004c\u907f\u602a\u7b26"].Index != 3 {
		t.Fatalf("expected avoid buff item reward at slot 3, got %+v", result.UpdatedItems)
	}
	if updated["\u004c\u767e\u5e74\u4eba\u53c2\u679c"].Count != 1 || updated["\u004c\u767e\u5e74\u4eba\u53c2\u679c"].Index != 4 {
		t.Fatalf("expected ginseng reward at slot 4, got %+v", result.UpdatedItems)
	}
	if updated["\u004c\u767e\u5e74\u87e0\u6843"].Count != 1 || updated["\u004c\u767e\u5e74\u87e0\u6843"].Index != 5 {
		t.Fatalf("expected peach reward at slot 5, got %+v", result.UpdatedItems)
	}
	items, _, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "\u80cc\u5305")
	if !ok {
		t.Fatal("expected bag items")
	}
	if _, ok := testRoleItemByName(items, "\u0031\u7ea7\u793c\u76d2"); ok {
		t.Fatalf("expected level 1 gift box to be consumed, got %+v", items)
	}
}

func TestStoreUseRoleItemOpensCapturedLevel5GiftBox(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "level5-gift-open",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	store.rolesByPID[login.PlayerID][0].Level = 5
	store.rolesByPID[login.PlayerID][0].Items = testLevel5GiftBoxBagItems(t)

	result := store.UseRoleItem(login.PlayerID, createResponse.Role.RoleID, "\u80cc\u5305", 1)
	if !result.Found || !result.Used || result.ErrorCode != "" {
		t.Fatalf("expected captured level 5 gift box to open, got %+v", result)
	}
	if len(result.ClearedItems) != 1 || result.ClearedItems[0].Type != "\u80cc\u5305" || result.ClearedItems[0].Index != 1 {
		t.Fatalf("expected source gift box slot 1 clear, got %+v", result.ClearedItems)
	}
	updated := testRoleItemsByName(result.UpdatedItems)
	if updated["\u004c\u521d\u9636\u7ecf\u9a8c\u5361"].Count != 1 || updated["\u004c\u521d\u9636\u7ecf\u9a8c\u5361"].Index != 1 {
		t.Fatalf("expected exp card reward at freed slot 1, got %+v", result.UpdatedItems)
	}
	if updated["\u004c\u82b1\u5377"].Count != 7 || updated["\u004c\u82b1\u5377"].Index != 8 {
		t.Fatalf("expected flower roll stack to count 7 at slot 8, got %+v", result.UpdatedItems)
	}
	if updated["\u004c\u56de\u57ce\u5492"].Count != 3 || updated["\u004c\u56de\u57ce\u5492"].Index != 6 {
		t.Fatalf("expected home scroll reward at slot 6, got %+v", result.UpdatedItems)
	}
	if updated["\u0031\u0030\u7ea7\u793c\u76d2"].Count != 1 || updated["\u0031\u0030\u7ea7\u793c\u76d2"].Index != 7 {
		t.Fatalf("expected level 10 gift box reward at slot 7, got %+v", result.UpdatedItems)
	}
	items, _, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "\u80cc\u5305")
	if !ok {
		t.Fatal("expected bag items")
	}
	if _, ok := testRoleItemByName(items, "\u0035\u7ea7\u793c\u76d2"); ok {
		t.Fatalf("expected level 5 gift box to be consumed, got %+v", items)
	}
}

func TestStoreUseRoleItemRejectsLevel10GiftBoxWhenCapturedBagFull(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "level10-gift-full",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	store.rolesByPID[login.PlayerID][0].Level = 12
	store.rolesByPID[login.PlayerID][0].Items = testLevel10GiftBoxBagItems(t, nil)

	result := store.UseRoleItem(login.PlayerID, createResponse.Role.RoleID, "\u80cc\u5305", 7)
	if !result.Found || result.Used || result.ErrorCode != "level10_gift_box_bag_full" || result.ErrorMessage != "\u80cc\u5305\u7a7a\u95f4\u4e0d\u8db3" {
		t.Fatalf("expected captured level 10 gift box full-bag rejection, got %+v", result)
	}
	if len(result.UpdatedItems) != 1 || result.UpdatedItems[0].Name != "\u0031\u0030\u7ea7\u793c\u76d2" || result.UpdatedItems[0].Index != 7 {
		t.Fatalf("expected rejected gift box refresh at slot 7, got %+v", result.UpdatedItems)
	}
	items, _, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "\u80cc\u5305")
	if !ok {
		t.Fatal("expected bag items")
	}
	item, ok := testRoleItemAt(items, "\u80cc\u5305", 7)
	if !ok || item.Name != "\u0031\u0030\u7ea7\u793c\u76d2" || item.Count != 1 {
		t.Fatalf("expected rejected gift box not to be consumed, got %+v", items)
	}
}

func TestStoreUseRoleItemOpensCapturedLevel10GiftBox(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "level10-gift-open",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	store.rolesByPID[login.PlayerID][0].Level = 12
	store.rolesByPID[login.PlayerID][0].Items = testLevel10GiftBoxBagItems(t, map[int]bool{15: true})

	result := store.UseRoleItem(login.PlayerID, createResponse.Role.RoleID, "\u80cc\u5305", 7)
	if !result.Found || !result.Used || result.ErrorCode != "" {
		t.Fatalf("expected captured level 10 gift box to open, got %+v", result)
	}
	if len(result.ClearedItems) != 1 || result.ClearedItems[0].Type != "\u80cc\u5305" || result.ClearedItems[0].Index != 7 {
		t.Fatalf("expected source gift box slot 7 clear, got %+v", result.ClearedItems)
	}
	updated := testRoleItemsByName(result.UpdatedItems)
	if updated["\u004c\u521d\u9636\u7ecf\u9a8c\u5361"].Count != 4 || updated["\u004c\u521d\u9636\u7ecf\u9a8c\u5361"].Index != 1 {
		t.Fatalf("expected exp card stack to count 4 at slot 1, got %+v", result.UpdatedItems)
	}
	if updated["\u004c\u82b1\u5377"].Count != 12 || updated["\u004c\u82b1\u5377"].Index != 8 {
		t.Fatalf("expected flower roll stack to count 12 at slot 8, got %+v", result.UpdatedItems)
	}
	if updated["\u004c\u80cc\u5305\u8865\u4e01"].Count != 1 || updated["\u004c\u80cc\u5305\u8865\u4e01"].Index != 7 || updated["\u004c\u80cc\u5305\u8865\u4e01"].Display != "560.png" {
		t.Fatalf("expected bag patch reward at freed slot 7, got %+v", result.UpdatedItems)
	}
	if updated["\u0031\u0035\u7ea7\u793c\u76d2"].Count != 1 || updated["\u0031\u0035\u7ea7\u793c\u76d2"].Index != 15 || updated["\u0031\u0035\u7ea7\u793c\u76d2"].Display != "742.png" {
		t.Fatalf("expected level 15 gift box reward at slot 15, got %+v", result.UpdatedItems)
	}
	items, _, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "\u80cc\u5305")
	if !ok {
		t.Fatal("expected bag items")
	}
	if item, ok := testRoleItemAt(items, "\u80cc\u5305", 7); !ok || item.Name != "\u004c\u80cc\u5305\u8865\u4e01" {
		t.Fatalf("expected slot 7 to contain bag patch after open, got %+v", items)
	}
	if _, ok := testRoleItemByName(items, "\u0031\u0030\u7ea7\u793c\u76d2"); ok {
		t.Fatalf("expected level 10 gift box to be consumed, got %+v", items)
	}
}

func testLevel10GiftBoxBagItems(t *testing.T, freeSlots map[int]bool) []RoleItem {
	t.Helper()

	items := make([]RoleItem, 0, 30)
	for index := 0; index < 30; index += 1 {
		if freeSlots != nil && freeSlots[index] {
			continue
		}
		switch index {
		case 1:
			items = append(items, testCapturedBagItem(t, "\u004c\u521d\u9636\u7ecf\u9a8c\u5361", index, 3))
		case 7:
			items = append(items, testCapturedBagItem(t, "\u0031\u0030\u7ea7\u793c\u76d2", index, 1))
		case 8:
			items = append(items, testCapturedBagItem(t, "\u004c\u82b1\u5377", index, 9))
		default:
			items = append(items, RoleItem{
				Type:        "\u80cc\u5305",
				Name:        fmt.Sprintf("filler-%02d", index),
				ItemType:    "null",
				Display:     fmt.Sprintf("filler-%02d.png", index),
				Description: fmt.Sprintf("f_i_filler-%02d&24@material&25@1&20@test filler&103@0&104@0&105@&107@&108@0", index),
				Count:       1,
				Index:       index,
				ItemLevel:   1,
			})
		}
	}
	return normalizeRoleItems(items)
}

func testLevel1GiftBoxBagItems(t *testing.T) []RoleItem {
	t.Helper()

	items := make([]RoleItem, 0, 3)
	for index := 0; index <= 5; index += 1 {
		switch index {
		case 1:
			items = append(items, testCapturedBagItem(t, "\u0031\u7ea7\u793c\u76d2", index, 1))
		case 3, 4, 5:
			continue
		default:
			items = append(items, RoleItem{
				Type:        "\u80cc\u5305",
				Name:        fmt.Sprintf("level1-filler-%02d", index),
				ItemType:    "null",
				Display:     fmt.Sprintf("level1-filler-%02d.png", index),
				Description: fmt.Sprintf("f_i_level1-filler-%02d&24@material&25@1&20@test filler&103@0&104@0&105@&107@&108@0", index),
				Count:       1,
				Index:       index,
				ItemLevel:   1,
			})
		}
	}
	return normalizeRoleItems(items)
}

func testLevel5GiftBoxBagItems(t *testing.T) []RoleItem {
	t.Helper()

	items := make([]RoleItem, 0, 7)
	for index := 0; index <= 8; index += 1 {
		switch index {
		case 1:
			items = append(items, testCapturedBagItem(t, "\u0035\u7ea7\u793c\u76d2", index, 1))
		case 6, 7:
			continue
		case 8:
			items = append(items, testCapturedBagItem(t, "\u004c\u82b1\u5377", index, 5))
		default:
			items = append(items, RoleItem{
				Type:        "\u80cc\u5305",
				Name:        fmt.Sprintf("level5-filler-%02d", index),
				ItemType:    "null",
				Display:     fmt.Sprintf("level5-filler-%02d.png", index),
				Description: fmt.Sprintf("f_i_level5-filler-%02d&24@material&25@1&20@test filler&103@0&104@0&105@&107@&108@0", index),
				Count:       1,
				Index:       index,
				ItemLevel:   1,
			})
		}
	}
	return normalizeRoleItems(items)
}

func testCapturedBagItem(t *testing.T, name string, index int, count int) RoleItem {
	t.Helper()

	item, ok := CapturedRoleItemTemplate(name)
	if !ok {
		t.Fatalf("expected captured item template %s", name)
	}
	item.Type = "\u80cc\u5305"
	item.Index = index
	item.Count = count
	return item
}

func testRoleItemsByName(items []RoleItem) map[string]RoleItem {
	result := make(map[string]RoleItem, len(items))
	for _, item := range items {
		result[item.Name] = item
	}
	return result
}

func testRoleItemByName(items []RoleItem, name string) (RoleItem, bool) {
	for _, item := range items {
		if item.Name == name {
			return item, true
		}
	}
	return RoleItem{}, false
}

func testRoleItemAt(items []RoleItem, itemType string, index int) (RoleItem, bool) {
	for _, item := range items {
		if item.Type == itemType && item.Index == index {
			return item, true
		}
	}
	return RoleItem{}, false
}

func TestStoreUseRoleItemRestoresTownHPFromClassicDescription(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "主城回血女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	store.rolesByPID[login.PlayerID][0].RoleState = &RoleState{
		Handle: createResponse.Role.RoleID,
		HP:     50,
		MP:     20,
		Exp:    0,
		Lv:     1,
		Speed:  130,
	}
	granted, ok := store.GrantRoleItem(login.PlayerID, createResponse.Role.RoleID, RoleItem{
		Type:        "背包",
		Name:        "测试包子",
		ItemType:    "消耗品",
		Display:     "212.png",
		Description: "f_i_测试包子&24@消耗品&25@99&7@60&20@恢复气力",
		Count:       2,
		Index:       -1,
		ItemLevel:   1,
	})
	if !ok {
		t.Fatal("expected recovery item grant")
	}

	result := store.UseRoleItem(login.PlayerID, createResponse.Role.RoleID, granted.Type, granted.Index)
	if !result.Found || !result.Used || !result.RoleStateChanged {
		t.Fatalf("expected town recovery item to be used, got %+v", result)
	}
	if result.PlayerBase.RoleState == nil || result.PlayerBase.RoleState.HP != 110 || result.PlayerBase.RoleState.MP != 20 {
		t.Fatalf("expected HP to recover to 110 and MP to stay 20, got %+v", result.PlayerBase.RoleState)
	}
	if len(result.UpdatedItems) != 1 || result.UpdatedItems[0].Count != 1 {
		t.Fatalf("expected stack count to decrease to 1, got %+v", result.UpdatedItems)
	}
	_, playerBase, ok := store.GetRoleRuntimeData(login.PlayerID, createResponse.Role.RoleID)
	if !ok || playerBase.RoleState == nil || playerBase.RoleState.HP != 110 {
		t.Fatalf("expected recovered HP to persist, got ok=%v playerBase=%+v", ok, playerBase)
	}
}

func TestStoreUseRoleItemRestoresSourceOwnMedicineFromDescription(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "快捷包子女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	store.rolesByPID[login.PlayerID][0].RoleState = &RoleState{
		Handle: createResponse.Role.RoleID,
		HP:     50,
		MP:     20,
		Exp:    0,
		Lv:     1,
		Speed:  130,
	}
	granted, ok := store.GrantRoleItem(login.PlayerID, createResponse.Role.RoleID, RoleItem{
		Type:        "背包",
		Name:        "包子",
		ItemType:    "own",
		Display:     "212.png",
		Description: "f_i_包子&24@消耗品&25@99&7@600&20@食用后可恢复些气力.",
		Count:       1,
		Index:       -1,
		ItemLevel:   1,
	})
	if !ok {
		t.Fatal("expected source medicine grant")
	}

	result := store.UseRoleItem(login.PlayerID, createResponse.Role.RoleID, granted.Type, granted.Index)
	if !result.Found || !result.Used || result.ErrorCode != "" {
		t.Fatalf("expected source own medicine to be usable, got %+v", result)
	}
	if result.PlayerBase.RoleState == nil || result.PlayerBase.RoleState.HP != result.PlayerBase.RolePhysique.MaxHP {
		t.Fatalf("expected source own medicine to recover HP to max, got state=%+v physique=%+v", result.PlayerBase.RoleState, result.PlayerBase.RolePhysique)
	}
}

func TestStoreUseRoleItemRejectsRecoveryItemWhenStateIsFull(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "满状态女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	granted, ok := store.GrantRoleItem(login.PlayerID, createResponse.Role.RoleID, RoleItem{
		Type:        "背包",
		Name:        "包子",
		ItemType:    "own",
		Display:     "212.png",
		Description: "f_i_包子&24@消耗品&25@99&7@600&20@食用后可恢复些气力.",
		Count:       1,
		Index:       -1,
		ItemLevel:   1,
	})
	if !ok {
		t.Fatal("expected source medicine grant")
	}

	result := store.UseRoleItem(login.PlayerID, createResponse.Role.RoleID, granted.Type, granted.Index)
	if !result.Found || result.Used || result.ErrorCode != "role_state_full" {
		t.Fatalf("expected full-state medicine use to be rejected, got %+v", result)
	}
	items, _, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "背包")
	if !ok {
		t.Fatal("expected bag items")
	}
	for _, item := range items {
		if item.Name == "包子" && item.Index == granted.Index && item.Count == 1 {
			return
		}
	}
	t.Fatalf("expected rejected medicine not to be consumed, got %+v", items)
}

func TestStoreUseRoleItemRestoresTownMPAndClampsToMax(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "主城回蓝女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	store.rolesByPID[login.PlayerID][0].RoleState = &RoleState{
		Handle: createResponse.Role.RoleID,
		HP:     80,
		MP:     30,
		Exp:    0,
		Lv:     1,
		Speed:  130,
	}
	granted, ok := store.GrantRoleItem(login.PlayerID, createResponse.Role.RoleID, RoleItem{
		Type:        "背包",
		Name:        "测试甘露",
		ItemType:    "材料 消耗品",
		Display:     "214.png",
		Description: "f_i_测试甘露&24@材料 消耗品&25@99&8@100&20@恢复精力",
		Count:       1,
		Index:       -1,
		ItemLevel:   1,
	})
	if !ok {
		t.Fatal("expected recovery item grant")
	}

	result := store.UseRoleItem(login.PlayerID, createResponse.Role.RoleID, granted.Type, granted.Index)
	if !result.Found || !result.Used || !result.RoleStateChanged {
		t.Fatalf("expected town recovery item to be used, got %+v", result)
	}
	if result.PlayerBase.RoleState == nil || result.PlayerBase.RoleState.HP != 80 || result.PlayerBase.RoleState.MP != result.PlayerBase.RolePhysique.MaxMP {
		t.Fatalf("expected MP to clamp to max and HP to stay 80, got state=%+v physique=%+v", result.PlayerBase.RoleState, result.PlayerBase.RolePhysique)
	}
	if len(result.ClearedItems) != 1 || result.ClearedItems[0].Index != granted.Index {
		t.Fatalf("expected single MP item to clear, got %+v", result.ClearedItems)
	}
}

func TestStoreHealRoleAtTownRestoresHPAndMPForFreeBeforeLevel15(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "医疗女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	store.rolesByPID[login.PlayerID][0].RoleState = &RoleState{
		Handle: createResponse.Role.RoleID,
		HP:     50,
		MP:     10,
		Exp:    0,
		Lv:     1,
		Speed:  130,
	}

	result := store.HealRoleAtTown(login.PlayerID, createResponse.Role.RoleID)
	if !result.Found || !result.Healed || result.Cost != 0 {
		t.Fatalf("expected free town heal, got %+v", result)
	}
	if result.RoleState.HP != result.PlayerBase.RolePhysique.MaxHP || result.RoleState.MP != result.PlayerBase.RolePhysique.MaxMP {
		t.Fatalf("expected HP/MP to restore to max, state=%+v physique=%+v", result.RoleState, result.PlayerBase.RolePhysique)
	}
	_, playerBase, ok := store.GetRoleRuntimeData(login.PlayerID, createResponse.Role.RoleID)
	if !ok || playerBase.RoleState == nil || playerBase.RoleState.HP != playerBase.RolePhysique.MaxHP || playerBase.RoleState.MP != playerBase.RolePhysique.MaxMP {
		t.Fatalf("expected town heal to persist, ok=%v playerBase=%+v", ok, playerBase)
	}
}

func TestStoreHealRoleAtTownChargesCopperAfterLevel15(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "收费医疗女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	levelResult := store.SetRoleLevel(login.PlayerID, createResponse.Role.RoleID, 19)
	if !levelResult.Found || !levelResult.Granted {
		t.Fatalf("expected level setup, got %+v", levelResult)
	}
	role := &store.rolesByPID[login.PlayerID][0]
	role.Currencies["铜钱"] = 289
	role.RoleState.HP = role.RolePhysique.MaxHP - 203
	role.RoleState.MP = role.RolePhysique.MaxMP

	result := store.HealRoleAtTown(login.PlayerID, createResponse.Role.RoleID)
	if !result.Found || !result.Healed || result.Cost != 45 {
		t.Fatalf("expected captured level 19 heal cost 45, got %+v", result)
	}
	if result.Currencies["铜钱"] != 244 {
		t.Fatalf("expected copper 289 -> 244, got %+v", result.Currencies)
	}
	if result.RoleState.HP != result.PlayerBase.RolePhysique.MaxHP || result.RoleState.MP != result.PlayerBase.RolePhysique.MaxMP {
		t.Fatalf("expected paid heal to restore HP/MP, state=%+v physique=%+v", result.RoleState, result.PlayerBase.RolePhysique)
	}
}

func TestClassicTownHealerCostMatchesCapturedSamples(t *testing.T) {
	if got := ClassicTownHealerCost(19, 203, 0); got != 45 {
		t.Fatalf("expected captured level 19 healer cost 45, got %d", got)
	}
	if got := ClassicTownHealerCost(34, 263, 248); got != 120 {
		t.Fatalf("expected captured level 34 healer cost 120, got %d", got)
	}
	if got := ClassicTownHealerCost(14, 999, 999); got != 0 {
		t.Fatalf("expected healer to be free before level 15, got %d", got)
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

func TestStoreEquipPromotedContainerEquipmentUsesCapturedSlots(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "装备槽验证",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	for _, spec := range []struct {
		name          string
		expectedIndex int
	}{
		{name: "如意之戒", expectedIndex: 6},
		{name: "琉璃耳环", expectedIndex: 7},
		{name: "白玉项链", expectedIndex: 8},
		{name: "死灵蓝娃", expectedIndex: 9},
		{name: "红色圣诞", expectedIndex: 11},
	} {
		template, ok := CapturedRoleItemTemplate(spec.name)
		if !ok {
			t.Fatalf("expected promoted template for %s", spec.name)
		}
		template.Type = "背包"
		template.Index = -1
		granted, ok := store.GrantRoleItem(login.PlayerID, createResponse.Role.RoleID, template)
		if !ok {
			t.Fatalf("expected grant for %s", spec.name)
		}
		result := store.EquipRoleItem(login.PlayerID, createResponse.Role.RoleID, granted.Type, granted.Index, 1)
		if !result.Found || !result.Equipped || result.EquippedItem.Index != spec.expectedIndex {
			t.Fatalf("expected %s to equip to captured slot %d, got %+v", spec.name, spec.expectedIndex, result)
		}
	}
}

func TestNormalizeRoleItemKeepsCapturedPetInstanceDescription(t *testing.T) {
	item := normalizeRoleItem(RoleItem{
		Type:        "装备",
		Name:        "死灵蓝娃",
		ItemType:    "equip",
		Display:     "970.png",
		Description: "f_i_死灵蓝娃^00ccff&24@宠物&25@1&19@喜好食物&103@0&104@0&105@s4_妮可露露&107@&108@0&19@宠物等级:45",
		Count:       1,
		Index:       rolePetEquipIndex,
		ItemLevel:   2,
	})
	if !strings.Contains(item.Description, "宠物等级:45") || !strings.Contains(item.Description, "s4_妮可露露") {
		t.Fatalf("expected captured pet instance description to remain intact, got %q", item.Description)
	}
}

func TestStoreTryEquipSupportsCapturedTreasureAndMountSlots(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "法宝试穿",
		Gender:         "female",
		RoleTemplateID: 1,
		SourceQuery:    "human/human.swf?co=5&sex=1&hr=12&e=14&m=5&",
	})

	preview := store.PreviewTryEquip(login.PlayerID, createResponse.Role.RoleID, "筋斗云")
	if !preview.Found || !preview.Previewed || preview.Item.Name != "筋斗云" {
		t.Fatalf("expected captured treasure TryEquip preview, got %+v", preview)
	}

	for _, spec := range []struct {
		name          string
		expectedIndex int
		expectedQuery string
	}{
		{name: "筋斗云", expectedIndex: 14},
		{name: "狰狞神骑", expectedIndex: 18, expectedQuery: "r=zn"},
	} {
		template, ok := CapturedRoleItemTemplate(spec.name)
		if !ok {
			t.Fatalf("expected template for %s", spec.name)
		}
		granted, ok := store.GrantRoleItem(login.PlayerID, createResponse.Role.RoleID, template)
		if !ok {
			t.Fatalf("expected grant for %s", spec.name)
		}
		result := store.EquipRoleItem(login.PlayerID, createResponse.Role.RoleID, granted.Type, granted.Index, 1)
		if !result.Found || !result.Equipped || result.EquippedItem.Index != spec.expectedIndex {
			t.Fatalf("expected %s to equip to source slot %d, got %+v", spec.name, spec.expectedIndex, result)
		}
		if spec.expectedQuery != "" && !strings.Contains(result.Role.SourceQuery, spec.expectedQuery) {
			t.Fatalf("expected %s to rebuild appearance with %s, got %q", spec.name, spec.expectedQuery, result.Role.SourceQuery)
		}
	}
}

func TestStoreTryEquipSupportsCapturedFashionSetPreview(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "超时空试穿",
		Gender:         "female",
		RoleTemplateID: 1,
		SourceQuery:    "human/human.swf?e=8&sex=1&hr=46&co=6&m=3&n=11&c=1&p=1&se=1&",
	})

	equipResult := store.EquipRoleItem(login.PlayerID, createResponse.Role.RoleID, "背包", 19, 1)
	if !equipResult.Found || !equipResult.Equipped || !strings.Contains(equipResult.Role.SourceQuery, "w8=5") {
		t.Fatalf("expected starter axe equip before fashion TryEquip, got %+v", equipResult)
	}
	mountTemplate, ok := CapturedRoleItemTemplate("狰狞神骑")
	if !ok {
		t.Fatal("expected captured mount template before fashion TryEquip")
	}
	mountItem, ok := store.GrantRoleItem(login.PlayerID, createResponse.Role.RoleID, mountTemplate)
	if !ok {
		t.Fatal("expected to grant captured mount before fashion TryEquip")
	}
	mountResult := store.EquipRoleItem(login.PlayerID, createResponse.Role.RoleID, mountItem.Type, mountItem.Index, 1)
	if !mountResult.Found || !mountResult.Equipped || !strings.Contains(mountResult.Role.SourceQuery, "r=zn") {
		t.Fatalf("expected captured mount before fashion TryEquip, got %+v", mountResult)
	}

	preview := store.PreviewTryEquip(login.PlayerID, createResponse.Role.RoleID, "超时空要塞")
	if !preview.Found || !preview.Previewed || preview.Item.Name != "超时空要塞" {
		t.Fatalf("expected captured fashion TryEquip preview, got %+v", preview)
	}
	for _, part := range []string{"w8=5", "c=88", "p=91", "se=79", "e=8", "sex=1", "hr=46", "co=6", "m=3", "n=11"} {
		if !strings.Contains(preview.SourceQuery, part) {
			t.Fatalf("expected captured fashion preview source query to include %s, got %q", part, preview.SourceQuery)
		}
	}
	for _, stalePart := range []string{"c=1", "p=1", "se=1", "r=zn"} {
		if strings.Contains(preview.SourceQuery, stalePart) {
			t.Fatalf("expected captured fashion preview to replace stale %s, got %q", stalePart, preview.SourceQuery)
		}
	}

	summerPreview := store.PreviewTryEquip(login.PlayerID, createResponse.Role.RoleID, "盛夏缤纷")
	if !summerPreview.Found || !summerPreview.Previewed || summerPreview.Item.Name != "盛夏缤纷" {
		t.Fatalf("expected captured summer fashion TryEquip preview, got %+v", summerPreview)
	}
	for _, part := range []string{"w8=5", "c=52", "p=55", "se=41", "hr=19", "e=8", "sex=1", "co=6", "m=3", "n=11"} {
		if !strings.Contains(summerPreview.SourceQuery, part) {
			t.Fatalf("expected captured summer fashion preview source query to include %s, got %q", part, summerPreview.SourceQuery)
		}
	}
	for _, stalePart := range []string{"c=1", "p=1", "se=1", "hr=46", "r=zn"} {
		if strings.Contains(summerPreview.SourceQuery, stalePart) {
			t.Fatalf("expected captured summer fashion preview to replace stale %s, got %q", stalePart, summerPreview.SourceQuery)
		}
	}

	role, _, ok := store.GetRoleRuntimeData(login.PlayerID, createResponse.Role.RoleID)
	if !ok || role.SourceQuery != mountResult.Role.SourceQuery {
		t.Fatalf("TryEquip must not mutate role source query, ok=%v role=%+v before=%q", ok, role, mountResult.Role.SourceQuery)
	}
}

func TestStoreEquipCapturedSpaceFortressAppliesAndRestoresFashionHair(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	created := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "超时空穿戴",
		Gender:         "female",
		RoleTemplateID: 1,
		SourceQuery:    "human/human.swf?e=8&sex=1&hr=12&co=6&m=3&n=11&",
		Appearance: RoleAppearance{
			"body": map[string]any{
				"sex":       1,
				"skinColor": 6,
				"hair":      12,
				"eyes":      8,
				"mouth":     3,
				"nose":      11,
			},
		},
	})
	if !created.Success {
		t.Fatalf("expected role create success, got %+v", created)
	}

	for _, name := range []string{"蛮力面甲", "蛮力护甲", "蛮力护腿", "蛮力战靴", "狰狞神骑"} {
		item, ok := CapturedRoleItemTemplate(name)
		if !ok {
			t.Fatalf("expected captured %s template", name)
		}
		item, ok = store.GrantRoleItem(login.PlayerID, created.Role.RoleID, item)
		if !ok {
			t.Fatalf("expected captured %s grant", name)
		}
		result := store.EquipRoleItem(login.PlayerID, created.Role.RoleID, item.Type, item.Index, 1)
		if !result.Found || !result.Equipped {
			t.Fatalf("expected %s equip, got %+v", name, result)
		}
	}

	fashion, ok := CapturedRoleItemTemplate("超时空要塞")
	if !ok {
		t.Fatal("expected captured 超时空要塞 template")
	}
	fashion, ok = store.GrantRoleItem(login.PlayerID, created.Role.RoleID, fashion)
	if !ok {
		t.Fatal("expected captured 超时空要塞 grant")
	}
	equipped := store.EquipRoleItem(login.PlayerID, created.Role.RoleID, fashion.Type, fashion.Index, 1)
	if !equipped.Found || !equipped.Equipped {
		t.Fatalf("expected 超时空要塞 equip, got %+v", equipped)
	}
	for _, part := range []string{"c=88", "p=91", "se=79", "hr=46"} {
		if !strings.Contains(equipped.Role.SourceQuery, part) {
			t.Fatalf("expected captured fashion equip query to include %s, got %q", part, equipped.Role.SourceQuery)
		}
	}
	if equipped.Role.BattleSourceQuery != equipped.Role.SourceQuery || equipped.PlayerBase.BattleSourceQuery != equipped.Role.SourceQuery {
		t.Fatalf("expected fashion battle source query to follow source query, source=%q roleBattle=%q baseBattle=%q", equipped.Role.SourceQuery, equipped.Role.BattleSourceQuery, equipped.PlayerBase.BattleSourceQuery)
	}
	if strings.Contains(equipped.Role.SourceQuery, "h=") {
		t.Fatalf("captured fashion has no hat field, got %q", equipped.Role.SourceQuery)
	}
	if strings.Contains(equipped.Role.SourceQuery, "&r=") {
		t.Fatalf("captured fashion has no ride field, got %q", equipped.Role.SourceQuery)
	}
	equippedFashion, ok := findRoleItem(equipped.Role.Items, "装备", roleFashionEquipIndex)
	if !ok || equippedFashion.Name != "超时空要塞" {
		t.Fatalf("expected captured fashion in standalone fashion slot %d, got %+v", roleFashionEquipIndex, equipped.Role.Items)
	}
	for _, name := range []string{"蛮力面甲", "蛮力护甲", "蛮力护腿", "蛮力战靴", "狰狞神骑"} {
		found := false
		for _, item := range equipped.Role.Items {
			if item.Type == "装备" && item.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected underlying %s to remain equipped, got %+v", name, equipped.Role.Items)
		}
	}

	unequipped := store.MoveRoleItem(login.PlayerID, created.Role.RoleID, "装备", roleFashionEquipIndex, "背包", -1, 1)
	if !unequipped.Found || !unequipped.Moved {
		t.Fatalf("expected 超时空要塞 unequip, got %+v", unequipped)
	}
	if !strings.Contains(unequipped.Role.SourceQuery, "hr=12") || strings.Contains(unequipped.Role.SourceQuery, "hr=46") {
		t.Fatalf("expected unequipped fashion to restore role hair, got %q", unequipped.Role.SourceQuery)
	}
	for _, part := range []string{"h=8", "c=10", "p=8", "se=4", "r=zn"} {
		if !strings.Contains(unequipped.Role.SourceQuery, part) {
			t.Fatalf("expected unequipped fashion to restore %s, got %q", part, unequipped.Role.SourceQuery)
		}
	}
}

func TestStoreCapturedWoodcutter333EquipDaggerAndBowRebuildsAppearance(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "33333333", "33333333")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "333",
		Gender:         "male",
		RoleTemplateID: 1,
	})
	if !createResponse.Success {
		t.Fatalf("expected 333 role create success, got %+v", createResponse)
	}

	grantedDagger, ok := store.GrantRoleItem(login.PlayerID, createResponse.Role.RoleID, RoleItem{
		Type:        "背包",
		Name:        "绯雨匕首",
		ItemType:    "equip",
		Display:     "51.png",
		Description: "f_i_绯雨匕首&24@武器·匕首系&25@1&22@游侠",
		Count:       1,
		Index:       20,
		ItemLevel:   2,
	})
	if !ok {
		t.Fatal("expected to grant captured dagger")
	}

	result := store.EquipRoleItem(login.PlayerID, createResponse.Role.RoleID, grantedDagger.Type, grantedDagger.Index, 1)

	if !result.Found || !result.Equipped {
		t.Fatalf("expected captured dagger equip success, got %+v", result)
	}
	if result.EquippedItem.Type != "装备" || result.EquippedItem.Index != 3 || result.EquippedItem.Name != "绯雨匕首" {
		t.Fatalf("expected dagger to equip into weapon slot 3, got %+v", result.EquippedItem)
	}
	if len(result.UpdatedItems) != 1 || result.UpdatedItems[0].Type != "背包" || result.UpdatedItems[0].Index != grantedDagger.Index || result.UpdatedItems[0].Name != "万相" {
		t.Fatalf("expected replaced bow to return to source bag slot, got %+v", result.UpdatedItems)
	}
	if len(result.ClearedItems) != 1 || result.ClearedItems[0].Type != grantedDagger.Type || result.ClearedItems[0].Index != grantedDagger.Index {
		t.Fatalf("expected dagger source bag slot clear, got %+v", result.ClearedItems)
	}
	if !strings.Contains(result.Role.SourceQuery, "w3=49") || strings.Contains(result.Role.SourceQuery, "w1=55") {
		t.Fatalf("expected dagger appearance w3=49 and no stale bow w1=55, got %q", result.Role.SourceQuery)
	}
	if result.Role.BattleSourceQuery != result.Role.SourceQuery {
		t.Fatalf("expected battle source query to follow dynamic equipment appearance, role=%q battle=%q", result.Role.SourceQuery, result.Role.BattleSourceQuery)
	}

	bagItems, _, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "背包")
	if !ok {
		t.Fatal("expected bag list after dagger equip")
	}
	foundBow := false
	for _, item := range bagItems {
		if item.Index == grantedDagger.Index && item.Name == "万相" {
			foundBow = true
		}
		if item.Name == "绯雨匕首" {
			t.Fatalf("expected dagger to leave bag after equip, got %+v", bagItems)
		}
	}
	if !foundBow {
		t.Fatalf("expected bow to occupy dagger source bag slot after equip, got %+v", bagItems)
	}

	revertResult := store.UseRoleItem(login.PlayerID, createResponse.Role.RoleID, "背包", grantedDagger.Index)
	if !revertResult.Found || !revertResult.Used || !revertResult.Equipped {
		t.Fatalf("expected backpack bow activeItem to equip back, got %+v", revertResult)
	}
	if len(revertResult.UpdatedItems) != 2 {
		t.Fatalf("expected activeItem equip to push replaced dagger and equipped bow, got %+v", revertResult.UpdatedItems)
	}
	if revertResult.UpdatedItems[0].Type != "背包" || revertResult.UpdatedItems[0].Index != grantedDagger.Index || revertResult.UpdatedItems[0].Name != "绯雨匕首" {
		t.Fatalf("expected replaced dagger to return to bow source bag slot, got %+v", revertResult.UpdatedItems)
	}
	if revertResult.UpdatedItems[1].Type != "装备" || revertResult.UpdatedItems[1].Index != 3 || revertResult.UpdatedItems[1].Name != "万相" {
		t.Fatalf("expected bow equipment push after activeItem equip, got %+v", revertResult.UpdatedItems)
	}
	if len(revertResult.ClearedItems) != 1 || revertResult.ClearedItems[0].Type != "背包" || revertResult.ClearedItems[0].Index != grantedDagger.Index {
		t.Fatalf("expected bow source bag slot clear before replacement item push, got %+v", revertResult.ClearedItems)
	}
	if !strings.Contains(revertResult.Role.SourceQuery, "w1=55") || strings.Contains(revertResult.Role.SourceQuery, "w3=49") {
		t.Fatalf("expected bow appearance w1=55 and no stale dagger w3=49, got %q", revertResult.Role.SourceQuery)
	}
	if revertResult.Role.BattleSourceQuery != revertResult.Role.SourceQuery {
		t.Fatalf("expected battle source query to follow reverted dagger appearance, role=%q battle=%q", revertResult.Role.SourceQuery, revertResult.Role.BattleSourceQuery)
	}

	bagItems, _, ok = store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "背包")
	if !ok {
		t.Fatal("expected bag list after bow re-equip")
	}
	foundDagger := false
	for _, item := range bagItems {
		if item.Index == grantedDagger.Index && item.Name == "绯雨匕首" {
			foundDagger = true
		}
		if item.Name == "万相" {
			t.Fatalf("expected bow to leave bag after re-equip, got %+v", bagItems)
		}
	}
	if !foundDagger {
		t.Fatalf("expected dagger to occupy bow source bag slot after re-equip, got %+v", bagItems)
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

func TestStoreMoveRoleItemStacksSameNameWarehouseItems(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "warehouse stack test",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	template, ok := CapturedRoleItemTemplate("\u8089")
	if !ok {
		t.Fatal("expected captured meat template")
	}
	template.Type = "\u4ed3\u5e93"
	template.Index = 0
	template.Count = 2
	if _, ok := store.GrantRoleItem(login.PlayerID, createResponse.Role.RoleID, template); !ok {
		t.Fatal("expected first warehouse meat grant to succeed")
	}
	template.Index = 1
	template.Count = 3
	if _, ok := store.GrantRoleItem(login.PlayerID, createResponse.Role.RoleID, template); !ok {
		t.Fatal("expected second warehouse meat grant to succeed")
	}

	result := store.MoveRoleItem(login.PlayerID, createResponse.Role.RoleID, "\u4ed3\u5e93", 1, "\u4ed3\u5e93", 0, 0)
	if !result.Found || !result.Moved {
		t.Fatalf("expected warehouse item stack success, got %+v", result)
	}
	if len(result.UpdatedItems) != 1 || result.UpdatedItems[0].Type != "\u4ed3\u5e93" || result.UpdatedItems[0].Index != 0 || result.UpdatedItems[0].Count != 5 || result.UpdatedItems[0].Name != "\u8089" {
		t.Fatalf("expected single stacked warehouse meat push at slot 0 count 5, got %+v", result.UpdatedItems)
	}
	warehouseItems, _, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "\u4ed3\u5e93")
	if !ok {
		t.Fatal("expected warehouse list after stacking")
	}
	itemsByIndex := map[int]RoleItem{}
	for _, item := range warehouseItems {
		itemsByIndex[item.Index] = item
	}
	if itemsByIndex[0].Name != "\u8089" || itemsByIndex[0].Count != 5 {
		t.Fatalf("expected warehouse meat count 5 at slot 0 after stacking, got %+v", itemsByIndex[0])
	}
	if _, exists := itemsByIndex[1]; exists {
		t.Fatalf("expected warehouse slot 1 to be cleared after stacking, got %+v", warehouseItems)
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

func TestStoreFinishRoleContainerStacksCompatibleBagItems(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "整包女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	meat, ok := CapturedRoleItemTemplate("肉")
	if !ok {
		t.Fatal("expected captured 肉 template")
	}
	meat.Type = "背包"
	meat.Index = 7
	meat.Count = 2
	if _, ok := store.GrantRoleItem(login.PlayerID, createResponse.Role.RoleID, meat); !ok {
		t.Fatal("expected first 肉 grant to succeed")
	}
	meat.Index = 22
	meat.Count = 3
	if _, ok := store.GrantRoleItem(login.PlayerID, createResponse.Role.RoleID, meat); !ok {
		t.Fatal("expected second 肉 grant to succeed")
	}

	skull, ok := CapturedRoleItemTemplate("头骨")
	if !ok {
		t.Fatal("expected captured 头骨 template")
	}
	skull.Type = "背包"
	skull.Index = 25
	skull.Count = 1
	if _, ok := store.GrantRoleItem(login.PlayerID, createResponse.Role.RoleID, skull); !ok {
		t.Fatal("expected 头骨 grant to succeed")
	}

	result := store.FinishRoleContainer(login.PlayerID, createResponse.Role.RoleID, "背包")
	if !result.Found || !result.Changed {
		t.Fatalf("expected bag finish success, got %+v", result)
	}
	expectedClears := []int{7, 19, 22, 25}
	if len(result.ClearedItems) != len(expectedClears) {
		t.Fatalf("expected compact clears %+v, got %+v", expectedClears, result.ClearedItems)
	}
	for index, expected := range expectedClears {
		if result.ClearedItems[index].Type != "背包" || result.ClearedItems[index].Index != expected {
			t.Fatalf("expected compact clear slot %d at position %d, got %+v", expected, index, result.ClearedItems)
		}
	}
	updatedByIndex := map[int]RoleItem{}
	for _, item := range result.UpdatedItems {
		updatedByIndex[item.Index] = item
	}
	if updatedByIndex[0].Name != "肉" || updatedByIndex[0].Count != 5 ||
		updatedByIndex[1].Name != "铁斧" || updatedByIndex[2].Name != "头骨" {
		t.Fatalf("expected compact stacked item pushes at slots 0..2, got %+v", result.UpdatedItems)
	}

	bagItems, _, ok := store.GetRoleItems(login.PlayerID, createResponse.Role.RoleID, "背包")
	if !ok {
		t.Fatal("expected bag list after finishing")
	}
	if len(bagItems) != 3 {
		t.Fatalf("expected three persisted bag items, got %+v", bagItems)
	}
	itemsByIndex := map[int]RoleItem{}
	for _, item := range bagItems {
		itemsByIndex[item.Index] = item
	}
	if itemsByIndex[0].Name != "肉" || itemsByIndex[0].Count != 5 {
		t.Fatalf("expected stacked 肉 to compact to slot 0, got %+v", bagItems)
	}
	if itemsByIndex[1].Name != "铁斧" || itemsByIndex[2].Name != "头骨" {
		t.Fatalf("expected unrelated items to compact in stable order, got %+v", bagItems)
	}
	for _, emptyIndex := range []int{7, 19, 22, 25} {
		if _, exists := itemsByIndex[emptyIndex]; exists {
			t.Fatalf("expected old slot %d to be empty after compact finish, got %+v", emptyIndex, bagItems)
		}
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

func TestPersistentStoreLoadsTownBuffs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ai-server.db")
	store, err := NewPersistentStore(path)
	if err != nil {
		t.Fatalf("new persistent store: %v", err)
	}
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "避怪测试",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	endTime := time.Now().Add(5 * time.Minute).UnixMilli()
	store.mu.Lock()
	store.rolesByPID[login.PlayerID][0].TownBuffs = []RoleTownBuff{{
		Handle:        createResponse.Role.RoleID,
		Name:          classicTownAvoidBuffName,
		Display:       "574.png",
		Description:   classicTownAvoidBuffDescription,
		EndTime:       endTime,
		BattleOnly:    0,
		SourceCapture: classicTownAvoidBuffSourceCapture,
	}}
	if err := store.persistPlayerStateLocked(login.PlayerID); err != nil {
		store.mu.Unlock()
		t.Fatalf("persist town buff: %v", err)
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
	buffs, ok := reopened.GetRoleTownBuffs(login.PlayerID, createResponse.Role.RoleID)
	if !ok || len(buffs) != 1 {
		t.Fatalf("expected persisted town buff after reopen, ok=%v buffs=%+v", ok, buffs)
	}
	if buffs[0].Name != classicTownAvoidBuffName || buffs[0].Display != "574.png" || buffs[0].EndTime != endTime {
		t.Fatalf("expected persisted avoid buff, got %+v", buffs[0])
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

func TestStorePersistsCapturedBattleSourceQuery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ai-server.db")
	store, err := NewPersistentStore(path)
	if err != nil {
		t.Fatalf("new persistent store: %v", err)
	}
	login := mustLogin(t, store, "mockuser", "magicpwd")
	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "抓包游侠",
		Gender:         "female",
		RoleTemplateID: 1,
		SourceQuery:    "human/human.swf?e=6&sex=1&hr=12&co=5&m=0&n=0&",
	})
	if !createResponse.Success {
		t.Fatalf("expected role create success, got %+v", createResponse)
	}

	// Stale battle_source_query must not win over the rebuilt town SourceQuery.
	// Account 111 regressed this way: town used fashion/current gear while battle
	// still preferred an older BattleSourceQuery snapshot.
	staleBattleSourceQuery := "human/human.swf?a=34&b=31&c=35&e=6&sex=1&h=12&hr=12&co=5&m=0&n=0&p=13&se=27&wr=11&w3=43&"
	store.mu.Lock()
	store.rolesByPID[login.PlayerID][0].BattleSourceQuery = staleBattleSourceQuery
	if err := store.persistPlayerStateLocked(login.PlayerID); err != nil {
		store.mu.Unlock()
		t.Fatalf("persist battle source query: %v", err)
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
		t.Fatal("expected role runtime data")
	}
	if role.BattleSourceQuery != role.SourceQuery {
		t.Fatalf("expected runtime battle source query to follow rebuilt source query, role=%q battle=%q", role.SourceQuery, role.BattleSourceQuery)
	}
	if playerBase.BattleSourceQuery != playerBase.SourceQuery || playerBase.SourceQuery != role.SourceQuery {
		t.Fatalf("expected player base battle/source query to match rebuilt role, role=%q baseSource=%q baseBattle=%q", role.SourceQuery, playerBase.SourceQuery, playerBase.BattleSourceQuery)
	}
	if strings.Contains(playerBase.BattleSourceQuery, "w3=43") {
		t.Fatalf("expected stale battle source query w3=43 to be discarded, got %q", playerBase.BattleSourceQuery)
	}
}

func TestStoreOrdinaryRoleBattleSourceQueryFollowsFashionEquip(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	created := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "111外观同步",
		Gender:         "male",
		RoleTemplateID: 1,
		SourceQuery:    "human/human.swf?e=14&sex=1&hr=12&co=5&m=5&n=0&",
		Appearance: RoleAppearance{
			"body": map[string]any{
				"sex":       1,
				"skinColor": 5,
				"hair":      12,
				"eyes":      14,
				"mouth":     5,
				"nose":      0,
			},
		},
	})
	if !created.Success {
		t.Fatalf("expected role create success, got %+v", created)
	}

	for _, name := range []string{"刎刀", "蛮力面甲", "蛮力肩甲", "蛮力护腕", "蛮力护腿", "蛮力护腰", "蛮力战靴", "狰狞神骑"} {
		item, ok := CapturedRoleItemTemplate(name)
		if !ok {
			t.Fatalf("expected captured %s template", name)
		}
		item, ok = store.GrantRoleItem(login.PlayerID, created.Role.RoleID, item)
		if !ok {
			t.Fatalf("expected grant %s", name)
		}
		result := store.EquipRoleItem(login.PlayerID, created.Role.RoleID, item.Type, item.Index, 1)
		if !result.Found || !result.Equipped {
			t.Fatalf("expected equip %s, got %+v", name, result)
		}
	}

	fashion, ok := CapturedRoleItemTemplate("超时空要塞")
	if !ok {
		t.Fatal("expected captured 超时空要塞 template")
	}
	fashion, ok = store.GrantRoleItem(login.PlayerID, created.Role.RoleID, fashion)
	if !ok {
		t.Fatal("expected grant 超时空要塞")
	}
	equipped := store.EquipRoleItem(login.PlayerID, created.Role.RoleID, fashion.Type, fashion.Index, 1)
	if !equipped.Found || !equipped.Equipped {
		t.Fatalf("expected fashion equip, got %+v", equipped)
	}
	for _, part := range []string{"w8=42", "c=88", "p=91", "se=79", "hr=46"} {
		if !strings.Contains(equipped.Role.SourceQuery, part) {
			t.Fatalf("expected fashion source query to include %s, got %q", part, equipped.Role.SourceQuery)
		}
	}
	if equipped.Role.BattleSourceQuery != equipped.Role.SourceQuery {
		t.Fatalf("expected battle source query to follow fashion equip, source=%q battle=%q", equipped.Role.SourceQuery, equipped.Role.BattleSourceQuery)
	}
	if equipped.PlayerBase.BattleSourceQuery != equipped.Role.SourceQuery {
		t.Fatalf("expected player base battle source query to follow fashion equip, source=%q battle=%q", equipped.Role.SourceQuery, equipped.PlayerBase.BattleSourceQuery)
	}
}

func TestPersistentStoreConsumeRoleItemUpdatesSingleRoleWithoutDeletingSiblings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ai-server.db")
	store, err := NewPersistentStore(path)
	if err != nil {
		t.Fatalf("new persistent store: %v", err)
	}
	login := mustLogin(t, store, "mockuser", "magicpwd")
	firstRole := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "扣箭游侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	secondRole := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "同账号备用角色",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	arrow, ok := CapturedRoleItemTemplate("穿甲箭")
	if !ok {
		t.Fatal("expected captured piercing arrow template")
	}
	arrow.Type = "背包"
	arrow.Index = 0
	arrow.Count = 3
	if _, ok := store.GrantRoleItem(login.PlayerID, firstRole.Role.RoleID, arrow); !ok {
		t.Fatal("expected piercing arrow grant")
	}

	result := store.ConsumeRoleItem(login.PlayerID, firstRole.Role.RoleID, "背包", 0, 1)
	if !result.Found || !result.Used || result.UpdatedItem == nil || result.UpdatedItem.Count != 2 {
		t.Fatalf("expected piercing arrow consume to leave count 2, got %+v", result)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	reopened, err := NewPersistentStore(path)
	if err != nil {
		t.Fatalf("reopen persistent store: %v", err)
	}
	defer reopened.Close()
	items, _, ok := reopened.GetRoleItems(login.PlayerID, firstRole.Role.RoleID, "背包")
	if !ok {
		t.Fatal("expected first role items after reopen")
	}
	remaining, ok := findRoleItem(items, "背包", 0)
	if !ok || remaining.Name != "穿甲箭" || remaining.Count != 2 {
		t.Fatalf("expected persisted piercing arrow count 2, ok=%v item=%+v items=%+v", ok, remaining, items)
	}
	sibling, _, ok := reopened.GetRoleRuntimeData(login.PlayerID, secondRole.Role.RoleID)
	if !ok || sibling.DisplayName != "同账号备用角色" {
		t.Fatalf("expected sibling role to survive single-role item consume persist, ok=%v role=%+v", ok, sibling)
	}
}

func TestPersistentStoreConfiguresSQLiteForFrequentItemWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ai-server.db")
	store, err := NewPersistentStore(path)
	if err != nil {
		t.Fatalf("new persistent store: %v", err)
	}
	defer store.Close()

	journalMode := ""
	if err := store.db.QueryRow(`PRAGMA journal_mode`).Scan(&journalMode); err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}
	if !strings.EqualFold(journalMode, "wal") {
		t.Fatalf("expected sqlite WAL journal mode for frequent item writes, got %q", journalMode)
	}
	synchronous := -1
	if err := store.db.QueryRow(`PRAGMA synchronous`).Scan(&synchronous); err != nil {
		t.Fatalf("query synchronous: %v", err)
	}
	if synchronous != 1 {
		t.Fatalf("expected sqlite synchronous=NORMAL(1), got %d", synchronous)
	}
}

func TestPersistentStoreConsumeRoleItemMutationOnlyPersistsItems(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ai-server.db")
	store, err := NewPersistentStore(path)
	if err != nil {
		t.Fatalf("new persistent store: %v", err)
	}
	login := mustLogin(t, store, "mockuser", "magicpwd")
	create := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "扣箭游侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	arrow, ok := CapturedRoleItemTemplate("穿甲箭")
	if !ok {
		t.Fatal("expected captured piercing arrow template")
	}
	arrow.Type = "背包"
	arrow.Index = 0
	arrow.Count = 3
	if _, ok := store.GrantRoleItem(login.PlayerID, create.Role.RoleID, arrow); !ok {
		t.Fatal("expected piercing arrow grant")
	}

	result := store.ConsumeRoleItemMutationOnly(login.PlayerID, create.Role.RoleID, "背包", 0, 1)
	if !result.Found || !result.Used || result.UpdatedItem == nil || result.UpdatedItem.Count != 2 {
		t.Fatalf("expected mutation-only piercing arrow consume to leave count 2, got %+v", result)
	}
	if result.PlayerBase.RoleState != nil || result.PlayerBase.RolePhysique != nil {
		t.Fatalf("expected mutation-only consume to avoid rebuilding player base, got %+v", result.PlayerBase)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	reopened, err := NewPersistentStore(path)
	if err != nil {
		t.Fatalf("reopen persistent store: %v", err)
	}
	defer reopened.Close()
	items, _, ok := reopened.GetRoleItems(login.PlayerID, create.Role.RoleID, "背包")
	if !ok {
		t.Fatal("expected bag after reopen")
	}
	remaining, ok := findRoleItem(items, "背包", 0)
	if !ok || remaining.Name != "穿甲箭" || remaining.Count != 2 {
		t.Fatalf("expected persisted mutation-only piercing arrow count 2, ok=%v item=%+v items=%+v", ok, remaining, items)
	}
}

func TestPersistentStoreUseRoleRecoveryItemMutationOnlyPersistsSingleRole(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ai-server.db")
	store, err := NewPersistentStore(path)
	if err != nil {
		t.Fatalf("new persistent store: %v", err)
	}
	login := mustLogin(t, store, "mockuser", "magicpwd")
	firstRole := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "主城吃药女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	secondRole := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "同账号旁观角色",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	medicine := RoleItem{
		Type:        "背包",
		Name:        "测试包子",
		ItemType:    "消耗品",
		Display:     "212.png",
		Description: "f_i_测试包子&24@消耗品&25@99&7@60&20@恢复气力",
		Count:       2,
		Index:       0,
		ItemLevel:   1,
	}
	if _, ok := store.GrantRoleItem(login.PlayerID, firstRole.Role.RoleID, medicine); !ok {
		t.Fatal("expected recovery item grant")
	}
	store.UpdateRoleState(login.PlayerID, firstRole.Role.RoleID, RoleState{
		Handle: firstRole.Role.RoleID,
		HP:     50,
		MP:     20,
		Exp:    0,
		Lv:     1,
		Speed:  130,
	})

	result := store.UseRoleRecoveryItemMutationOnly(login.PlayerID, firstRole.Role.RoleID, "背包", 0, 60, 0)
	if !result.Found || !result.Used || !result.RoleStateChanged {
		t.Fatalf("expected mutation-only recovery item use success, got %+v", result)
	}
	if result.PlayerBase.RoleState == nil || result.PlayerBase.RoleState.HP != 110 || result.PlayerBase.RoleState.MP != 20 {
		t.Fatalf("expected HP to recover to 110 and MP to stay 20, got %+v", result.PlayerBase.RoleState)
	}
	if result.UpdatedItem == nil || result.UpdatedItem.Count != 1 {
		t.Fatalf("expected recovery item stack count 1, got %+v", result)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	reopened, err := NewPersistentStore(path)
	if err != nil {
		t.Fatalf("reopen persistent store: %v", err)
	}
	defer reopened.Close()
	_, playerBase, ok := reopened.GetRoleRuntimeData(login.PlayerID, firstRole.Role.RoleID)
	if !ok || playerBase.RoleState == nil || playerBase.RoleState.HP != 110 {
		t.Fatalf("expected recovered HP after reopen, ok=%v playerBase=%+v", ok, playerBase)
	}
	items, _, ok := reopened.GetRoleItems(login.PlayerID, firstRole.Role.RoleID, "背包")
	if !ok {
		t.Fatal("expected first role items after reopen")
	}
	remaining, ok := findRoleItem(items, "背包", 0)
	if !ok || remaining.Name != "测试包子" || remaining.Count != 1 {
		t.Fatalf("expected persisted recovery item count 1, ok=%v item=%+v items=%+v", ok, remaining, items)
	}
	sibling, _, ok := reopened.GetRoleRuntimeData(login.PlayerID, secondRole.Role.RoleID)
	if !ok || sibling.DisplayName != "同账号旁观角色" {
		t.Fatalf("expected sibling role to survive recovery item persist, ok=%v role=%+v", ok, sibling)
	}
}

func TestPersistentStoreHighFrequencyRoleMutationsDoNotRewriteSiblingRows(t *testing.T) {
	type mutationSpec struct {
		name   string
		setup  func(t *testing.T, store *Store, playerID string, roleID string)
		mutate func(t *testing.T, store *Store, playerID string, roleID string)
	}
	specs := []mutationSpec{
		{
			name: "grant item",
			mutate: func(t *testing.T, store *Store, playerID string, roleID string) {
				t.Helper()
				item := RoleItem{
					Type:        "背包",
					Name:        "高频奖励物",
					Display:     "1.png",
					Description: "f_i_高频奖励物&24@材料",
					Count:       1,
					Index:       -1,
					ItemLevel:   1,
				}
				if _, ok := store.GrantRoleItem(playerID, roleID, item); !ok {
					t.Fatal("expected grant item success")
				}
			},
		},
		{
			name: "purchase item",
			mutate: func(t *testing.T, store *Store, playerID string, roleID string) {
				t.Helper()
				item := RoleItem{
					Type:        "背包",
					Name:        "高频购买物",
					Display:     "1.png",
					Description: "f_i_高频购买物&24@材料",
					Count:       1,
					Index:       -1,
					ItemLevel:   1,
				}
				result := store.PurchaseRoleItem(playerID, roleID, item, nil)
				if !result.Found || !result.Purchased {
					t.Fatalf("expected purchase item success, got %+v", result)
				}
			},
		},
		{
			name: "grant experience",
			mutate: func(t *testing.T, store *Store, playerID string, roleID string) {
				t.Helper()
				result := store.GrantRoleExperience(playerID, roleID, 80)
				if !result.Found || !result.Granted {
					t.Fatalf("expected grant experience success, got %+v", result)
				}
			},
		},
		{
			name: "move item",
			mutate: func(t *testing.T, store *Store, playerID string, roleID string) {
				t.Helper()
				result := store.MoveRoleItem(playerID, roleID, "背包", 19, "背包", 0, 0)
				if !result.Found || !result.Moved {
					t.Fatalf("expected move item success, got %+v", result)
				}
			},
		},
		{
			name: "finish container",
			setup: func(t *testing.T, store *Store, playerID string, roleID string) {
				t.Helper()
				meat, ok := CapturedRoleItemTemplate("肉")
				if !ok {
					t.Fatal("expected captured 肉 template")
				}
				meat.Type = "背包"
				meat.Index = 7
				meat.Count = 2
				if _, ok := store.GrantRoleItem(playerID, roleID, meat); !ok {
					t.Fatal("expected first 肉 grant")
				}
				meat.Index = 22
				meat.Count = 3
				if _, ok := store.GrantRoleItem(playerID, roleID, meat); !ok {
					t.Fatal("expected second 肉 grant")
				}
			},
			mutate: func(t *testing.T, store *Store, playerID string, roleID string) {
				t.Helper()
				result := store.FinishRoleContainer(playerID, roleID, "背包")
				if !result.Found || !result.Changed {
					t.Fatalf("expected finish container success, got %+v", result)
				}
			},
		},
		{
			name: "sell item",
			setup: func(t *testing.T, store *Store, playerID string, roleID string) {
				t.Helper()
				item := RoleItem{
					Type:        "背包",
					Name:        "可卖高频物",
					Display:     "1.png",
					Description: "f_i_可卖高频物&24@材料&108@3",
					Count:       2,
					Index:       0,
					ItemLevel:   1,
				}
				if _, ok := store.GrantRoleItem(playerID, roleID, item); !ok {
					t.Fatal("expected sale item grant")
				}
			},
			mutate: func(t *testing.T, store *Store, playerID string, roleID string) {
				t.Helper()
				result := store.SellRoleItem(playerID, roleID, "背包", 0, 1)
				if !result.Found || !result.Sold {
					t.Fatalf("expected sell item success, got %+v", result)
				}
			},
		},
	}

	for _, spec := range specs {
		t.Run(spec.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "ai-server.db")
			store, err := NewPersistentStore(path)
			if err != nil {
				t.Fatalf("new persistent store: %v", err)
			}
			login := mustLogin(t, store, "mockuser", "magicpwd")
			firstRole := store.CreateRole(RoleCreateRequest{
				PlayerID:       login.PlayerID,
				SessionToken:   login.SessionToken,
				DisplayName:    "高频主角色",
				Gender:         "female",
				RoleTemplateID: 1,
			})
			secondRole := store.CreateRole(RoleCreateRequest{
				PlayerID:       login.PlayerID,
				SessionToken:   login.SessionToken,
				DisplayName:    "同账号哨兵角色",
				Gender:         "female",
				RoleTemplateID: 1,
			})
			if spec.setup != nil {
				spec.setup(t, store, login.PlayerID, firstRole.Role.RoleID)
			}
			const sentinelName = "数据库哨兵角色"
			if _, err := store.db.Exec(`UPDATE roles SET display_name = ? WHERE player_id = ? AND role_id = ?`, sentinelName, login.PlayerID, secondRole.Role.RoleID); err != nil {
				t.Fatalf("mark sibling role row: %v", err)
			}

			spec.mutate(t, store, login.PlayerID, firstRole.Role.RoleID)
			if err := store.Close(); err != nil {
				t.Fatalf("close store: %v", err)
			}

			reopened, err := NewPersistentStore(path)
			if err != nil {
				t.Fatalf("reopen persistent store: %v", err)
			}
			defer reopened.Close()
			sibling, _, ok := reopened.GetRoleRuntimeData(login.PlayerID, secondRole.Role.RoleID)
			if !ok || sibling.DisplayName != sentinelName {
				t.Fatalf("expected sibling db row to stay untouched, ok=%v role=%+v", ok, sibling)
			}
		})
	}
}

func TestPersistentStoreConsumeRoleItemMutationOnlySerializesSameRoleConcurrentWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ai-server.db")
	store, err := NewPersistentStore(path)
	if err != nil {
		t.Fatalf("new persistent store: %v", err)
	}
	login := mustLogin(t, store, "mockuser", "magicpwd")
	create := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "并发扣箭游侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	arrow, ok := CapturedRoleItemTemplate("穿甲箭")
	if !ok {
		t.Fatal("expected captured piercing arrow template")
	}
	arrow.Type = "背包"
	arrow.Index = 0
	arrow.Count = 20
	if _, ok := store.GrantRoleItem(login.PlayerID, create.Role.RoleID, arrow); !ok {
		t.Fatal("expected piercing arrow grant")
	}

	const consumeCount = 12
	start := make(chan struct{})
	errors := make(chan error, consumeCount)
	var wg sync.WaitGroup
	for index := 0; index < consumeCount; index += 1 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			result := store.ConsumeRoleItemMutationOnly(login.PlayerID, create.Role.RoleID, "背包", 0, 1)
			if !result.Found || !result.Used {
				errors <- fmt.Errorf("expected concurrent mutation-only consume success, got %+v", result)
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	items, _, ok := store.GetRoleItems(login.PlayerID, create.Role.RoleID, "背包")
	if !ok {
		t.Fatal("expected bag before reopen")
	}
	remaining, ok := findRoleItem(items, "背包", 0)
	if !ok || remaining.Name != "穿甲箭" || remaining.Count != 8 {
		t.Fatalf("expected in-memory piercing arrow count 8 after concurrent consume, ok=%v item=%+v items=%+v", ok, remaining, items)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	reopened, err := NewPersistentStore(path)
	if err != nil {
		t.Fatalf("reopen persistent store: %v", err)
	}
	defer reopened.Close()
	items, _, ok = reopened.GetRoleItems(login.PlayerID, create.Role.RoleID, "背包")
	if !ok {
		t.Fatal("expected bag after reopen")
	}
	remaining, ok = findRoleItem(items, "背包", 0)
	if !ok || remaining.Name != "穿甲箭" || remaining.Count != 8 {
		t.Fatalf("expected persisted piercing arrow count 8 after concurrent consume, ok=%v item=%+v items=%+v", ok, remaining, items)
	}
}

func TestStoreGetRoleBagItemByNameDoesNotBackfillConsumedMaterial(t *testing.T) {
	store := NewStore()
	login := mustLogin(t, store, "mockuser", "magicpwd")
	create := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "扣空材料游侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	arrow, ok := CapturedRoleItemTemplate("穿甲箭")
	if !ok {
		t.Fatal("expected captured piercing arrow template")
	}
	arrow.Type = "背包"
	arrow.Index = 0
	arrow.Count = 1
	if _, ok := store.GrantRoleItem(login.PlayerID, create.Role.RoleID, arrow); !ok {
		t.Fatal("expected piercing arrow grant")
	}

	result := store.ConsumeRoleItemMutationOnly(login.PlayerID, create.Role.RoleID, "背包", 0, 1)
	if !result.Found || !result.Used || len(result.ClearedItems) != 1 {
		t.Fatalf("expected mutation-only consume to clear final piercing arrow, got %+v", result)
	}
	if item, ok := store.GetRoleBagItemByName(login.PlayerID, create.Role.RoleID, "穿甲箭"); ok {
		t.Fatalf("expected consumed material lookup to stay missing, got %+v", item)
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

func TestStoreCapturedMaterialTemplatesFillMissingIconFields(t *testing.T) {
	cases := []struct {
		name      string
		display   string
		itemType  string
		itemLevel int
	}{
		{name: "碎铁矿", display: "105.png", itemType: "null", itemLevel: 1},
		{name: "废渣", display: "103.png", itemType: "null", itemLevel: 1},
		{name: "石块", display: "104.png", itemType: "null", itemLevel: 1},
		{name: "水晶", display: "111.png", itemType: "null", itemLevel: 1},
		{name: "碎金片", display: "106.png", itemType: "null", itemLevel: 2},
		{name: "岩魔菱石", display: "112.png", itemType: "null", itemLevel: 3},
		{name: "岩魔球石", display: "113.png", itemType: "own", itemLevel: 2},
		{name: "巨岩魔的拳", display: "114.png", itemType: "own", itemLevel: 2},
		{name: "巨岩魔的头", display: "115.png", itemType: "null", itemLevel: 3},
		{name: "宝匣", display: "596.png", itemType: "own", itemLevel: 3},
		{name: "兽牙", display: "68.png", itemType: "null", itemLevel: 1},
		{name: "头骨", display: "102.png", itemType: "null", itemLevel: 2},
		{name: "雪莲花", display: "935.png", itemType: "null", itemLevel: 2},
	}

	for _, tc := range cases {
		template, ok := CapturedRoleItemTemplate(tc.name)
		if !ok {
			t.Fatalf("expected captured source material template for %s", tc.name)
		}
		if template.Display != tc.display || template.ItemType != tc.itemType || template.ItemLevel != tc.itemLevel {
			t.Fatalf("expected %s template display=%s itemLevel=%d, got %+v", tc.name, tc.display, tc.itemLevel, template)
		}

		item := normalizeRoleItem(RoleItem{
			Type:  "背包",
			Name:  tc.name,
			Count: 1,
			Index: 6,
		})
		if item.Display != tc.display || item.ItemType != tc.itemType || item.Description == "" || item.ItemLevel != tc.itemLevel {
			t.Fatalf("expected %s missing fields to be filled from template, got %+v", tc.name, item)
		}
	}
}

func TestCapturedRoleItemTemplatesCoverConfiguredBattleRewardItems(t *testing.T) {
	rewardSources := map[string]string{}
	for _, row := range classicdata.MustLoadTable(classicdata.TableDrop).Rows {
		collectConfiguredRewardItemNames(rewardSources, row["items"], "classicdata/drop-table")
	}

	candidateFile, err := os.Open(filepath.Join("..", "battle", "config", "classic-battle-reward-candidate.csv"))
	if err != nil {
		t.Fatalf("open classic battle reward candidate table: %v", err)
	}
	defer candidateFile.Close()
	reader := csv.NewReader(candidateFile)
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("read classic battle reward candidate table: %v", err)
	}
	if len(records) < 2 {
		t.Fatal("expected classic battle reward candidate rows")
	}
	itemCountsIndex := -1
	for index, name := range records[0] {
		if name == "item_counts" {
			itemCountsIndex = index
			break
		}
	}
	if itemCountsIndex < 0 {
		t.Fatal("classic battle reward candidate table missing item_counts column")
	}
	for _, record := range records[1:] {
		if itemCountsIndex >= len(record) {
			continue
		}
		collectConfiguredRewardItemNames(rewardSources, record[itemCountsIndex], "battle/reward-candidate")
	}

	missing := []string{}
	for name, source := range rewardSources {
		template, ok := CapturedRoleItemTemplate(name)
		if !ok || strings.TrimSpace(template.Display) == "" {
			missing = append(missing, fmt.Sprintf("%s from %s", name, source))
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("configured battle reward items missing icon templates: %s", strings.Join(missing, "; "))
	}
}

func collectConfiguredRewardItemNames(out map[string]string, value string, source string) {
	for _, part := range strings.Split(value, ";") {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		if x := strings.LastIndex(name, "x"); x > 0 && x < len(name)-1 {
			allDigits := true
			for _, ch := range name[x+1:] {
				if ch < '0' || ch > '9' {
					allDigits = false
					break
				}
			}
			if allDigits {
				name = strings.TrimSpace(name[:x])
			}
		}
		if name != "" {
			out[name] = source
		}
	}
}

func TestStoreCapturedHuangfengEquipmentDropTemplateFillsMissingIconFields(t *testing.T) {
	template, ok := CapturedRoleItemTemplate("黄风腰带")
	if !ok {
		t.Fatal("expected captured source equipment template for 黄风腰带")
	}
	if template.Display != "547.png" || template.ItemType != "equip" || template.ItemLevel != 2 {
		t.Fatalf("expected 黄风腰带 template display=547.png itemType=equip itemLevel=2, got %+v", template)
	}
	if !strings.Contains(template.Description, "护具") || !strings.Contains(template.Description, "sitem_jhj") {
		t.Fatalf("expected 黄风腰带 source description metadata, got %q", template.Description)
	}

	item := normalizeRoleItem(RoleItem{
		Type:  "背包",
		Name:  "黄风腰带",
		Count: 1,
		Index: 40,
	})
	if item.Display != "547.png" || item.ItemType != "equip" || item.Description == "" || item.ItemLevel != 2 {
		t.Fatalf("expected 黄风腰带 missing fields to be filled from template, got %+v", item)
	}

	staleFallbackItem := normalizeRoleItem(RoleItem{
		Type:        "背包",
		Name:        "黄风腰带",
		ItemType:    "own",
		Display:     template.Display,
		Description: template.Description,
		Count:       1,
		Index:       40,
		ItemLevel:   1,
	})
	if staleFallbackItem.ItemType != "equip" || staleFallbackItem.ItemLevel != 2 {
		t.Fatalf("expected stale fallback 黄风腰带 metadata to be repaired from template, got %+v", staleFallbackItem)
	}
}

func TestStoreCapturedHuangfengBossRewardTemplatesFillMissingIconFields(t *testing.T) {
	cases := []struct {
		name         string
		templateType string
		display      string
		itemType     string
		slot         int
		itemLevel    int
		token        string
	}{
		{name: "红方巾", templateType: "背包", display: "121.png", itemType: "null", slot: 0, itemLevel: 1, token: "红色绸缎制方巾"},
		{name: "绸缎", templateType: "背包", display: "79.png", itemType: "null", slot: 0, itemLevel: 1, token: "代表高贵的布料"},
		{name: "呼啸战靴", templateType: "装备", display: "502.png", itemType: "equip", slot: 12, itemLevel: 2, token: "护具·足部"},
		{name: "寨夫人上衣", templateType: "装备", display: "474.png", itemType: "equip", slot: 4, itemLevel: 2, token: "护具·躯干"},
		{name: "寨夫人护腕", templateType: "装备", display: "467.png", itemType: "equip", slot: 2, itemLevel: 2, token: "护具·护腕"},
	}

	for _, tc := range cases {
		template, ok := CapturedRoleItemTemplate(tc.name)
		if !ok {
			t.Fatalf("expected captured Huangfeng boss reward template for %s", tc.name)
		}
		if template.Type != tc.templateType || template.Display != tc.display || template.ItemType != tc.itemType || template.Index != tc.slot || template.ItemLevel != tc.itemLevel {
			t.Fatalf("expected %s template type=%s display=%s itemType=%s slot=%d itemLevel=%d, got %+v", tc.name, tc.templateType, tc.display, tc.itemType, tc.slot, tc.itemLevel, template)
		}
		if !strings.Contains(template.Description, tc.token) {
			t.Fatalf("expected %s description to include %q, got %q", tc.name, tc.token, template.Description)
		}

		item := normalizeRoleItem(RoleItem{
			Type:  "背包",
			Name:  tc.name,
			Count: 1,
			Index: 6,
		})
		if item.Display != tc.display || item.ItemType != tc.itemType || item.Description == "" || item.ItemLevel != tc.itemLevel {
			t.Fatalf("expected %s missing fields to be filled from template, got %+v", tc.name, item)
		}

		staleFallbackItem := normalizeRoleItem(RoleItem{
			Type:        "背包",
			Name:        tc.name,
			ItemType:    "own",
			Display:     template.Display,
			Description: template.Description,
			Count:       1,
			Index:       6,
			ItemLevel:   1,
		})
		if staleFallbackItem.ItemType != tc.itemType || staleFallbackItem.ItemLevel != tc.itemLevel {
			t.Fatalf("expected stale fallback %s metadata to be repaired from template, got %+v", tc.name, staleFallbackItem)
		}
	}
}

func TestStoreCapturedHuangfengCandidateRewardTemplatesFillMissingIconFields(t *testing.T) {
	cases := []struct {
		name         string
		templateType string
		display      string
		itemType     string
		slot         int
		itemLevel    int
		token        string
	}{
		{name: "刺", templateType: "背包", display: "134.png", itemType: "null", slot: 0, itemLevel: 1, token: "锋利尖锐"},
		{name: "红缨", templateType: "背包", display: "77.png", itemType: "null", slot: 0, itemLevel: 1, token: "丝线染色后制成"},
		{name: "刀客布衣", templateType: "装备", display: "540.png", itemType: "equip", slot: 4, itemLevel: 2, token: "护具·躯干"},
		{name: "剔骨刀", templateType: "装备", display: "552.png", itemType: "equip", slot: 3, itemLevel: 2, token: "武器·匕首系"},
		{name: "图腾面具", templateType: "背包", display: "135.png", itemType: "null", slot: 0, itemLevel: 2, token: "诅咒的面具"},
		{name: "黄风围巾", templateType: "装备", display: "548.png", itemType: "equip", slot: 0, itemLevel: 3, token: "护具·头部"},
	}

	for _, tc := range cases {
		template, ok := CapturedRoleItemTemplate(tc.name)
		if !ok {
			t.Fatalf("expected captured Huangfeng candidate reward template for %s", tc.name)
		}
		if template.Type != tc.templateType || template.Display != tc.display || template.ItemType != tc.itemType || template.Index != tc.slot || template.ItemLevel != tc.itemLevel {
			t.Fatalf("expected %s template type=%s display=%s itemType=%s slot=%d itemLevel=%d, got %+v", tc.name, tc.templateType, tc.display, tc.itemType, tc.slot, tc.itemLevel, template)
		}
		if !strings.Contains(template.Description, tc.token) {
			t.Fatalf("expected %s description to include %q, got %q", tc.name, tc.token, template.Description)
		}

		item := normalizeRoleItem(RoleItem{
			Type:  "背包",
			Name:  tc.name,
			Count: 1,
			Index: 6,
		})
		if item.Display != tc.display || item.ItemType != tc.itemType || item.Description == "" || item.ItemLevel != tc.itemLevel {
			t.Fatalf("expected %s missing fields to be filled from template, got %+v", tc.name, item)
		}
	}
}

func TestStoreClassicDataEquipmentTemplatesFillRobberDropFallbacks(t *testing.T) {
	cases := []struct {
		name    string
		display string
		slot    int
		token   string
		stat    string
	}{
		{name: "盗贼的鞋", display: "542.png", slot: 12, token: "护具·足部", stat: "移动+2"},
		{name: "盗贼护腿", display: "543.png", slot: 5, token: "护具·腿", stat: "命中+3"},
		{name: "盗贼布衣", display: "544.png", slot: 4, token: "护具·躯干", stat: "耐力+2"},
		{name: "盗贼护臂", display: "545.png", slot: 2, token: "护具·护腕", stat: "爆击+50"},
		{name: "盗贼腰带", display: "546.png", slot: 10, token: "护具·腰部", stat: "爆击+5"},
	}

	for _, tc := range cases {
		template, ok := CapturedRoleItemTemplate(tc.name)
		if !ok {
			t.Fatalf("expected classicdata robber equipment template for %s", tc.name)
		}
		if template.Type != "装备" || template.ItemType != "equip" || template.Display != tc.display || template.Index != tc.slot || template.ItemLevel != 2 {
			t.Fatalf("expected %s robber equipment template display=%s slot=%d, got %+v", tc.name, tc.display, tc.slot, template)
		}
		if !strings.Contains(template.Description, tc.token) {
			t.Fatalf("expected %s description to include %q, got %q", tc.name, tc.token, template.Description)
		}
		if !strings.Contains(template.Description, tc.stat) {
			t.Fatalf("expected %s source stat description to include %q, got %q", tc.name, tc.stat, template.Description)
		}

		item := normalizeRoleItem(RoleItem{
			Type:  "背包",
			Name:  tc.name,
			Count: 1,
			Index: 6,
		})
		if item.Display != tc.display || item.ItemType != "equip" || item.Description == "" || item.ItemLevel != 2 {
			t.Fatalf("expected %s missing fields to be filled from classicdata template, got %+v", tc.name, item)
		}

		staleItem := normalizeRoleItem(RoleItem{
			Type:        "背包",
			Name:        tc.name,
			ItemType:    "equip",
			Display:     tc.display,
			Description: "精炼潜质: [精炼+1",
			Count:       1,
			Index:       6,
			ItemLevel:   2,
		})
		if staleItem.Index != 6 || staleItem.Type != "背包" {
			t.Fatalf("expected %s stale item refresh to preserve container/index, got %+v", tc.name, staleItem)
		}
		if !strings.Contains(staleItem.Description, tc.token) || !strings.Contains(staleItem.Description, tc.stat) || !strings.Contains(staleItem.Description, "&108@") {
			t.Fatalf("expected %s stale description to refresh from classicdata template, got %q", tc.name, staleItem.Description)
		}
	}
}

func TestStoreCapturedFeixiandongEquipmentTemplates(t *testing.T) {
	cases := []struct {
		name             string
		display          string
		slot             int
		descriptionToken string
	}{
		{name: "岩魔剑", display: "606.png", slot: 3, descriptionToken: "武器·单剑系"},
		{name: "岩化护腿", display: "598.png", slot: 5, descriptionToken: "护具·腿"},
		{name: "蓝晶护肩", display: "603.png", slot: 1, descriptionToken: "护具·肩部"},
	}

	for _, tc := range cases {
		template, ok := CapturedRoleItemTemplate(tc.name)
		if !ok {
			t.Fatalf("expected captured source equipment template for %s", tc.name)
		}
		if template.Type != "装备" || template.ItemType != "equip" || template.Display != tc.display || template.Index != tc.slot {
			t.Fatalf("expected %s equipment template display=%s slot=%d, got %+v", tc.name, tc.display, tc.slot, template)
		}
		if !strings.Contains(template.Description, tc.descriptionToken) {
			t.Fatalf("expected %s description to include %q, got %q", tc.name, tc.descriptionToken, template.Description)
		}
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

func TestStoreCapturedWoodcutterEquipmentAppearance(t *testing.T) {
	sourceQuery := "human/human.swf?sex=1&co=5&hr=12&e=6&n=0&m=0&"
	items := []RoleItem{
		{Type: "装备", Name: "黄风围巾", ItemType: "equip", Index: 0},
		{Type: "装备", Name: "蚩颅王护肩", ItemType: "equip", Index: 1},
		{Type: "装备", Name: "黄风护腕", ItemType: "equip", Index: 2},
		{Type: "装备", Name: "绯雨匕首", ItemType: "equip", Index: 3},
		{Type: "装备", Name: "神风护甲", ItemType: "equip", Index: 4},
		{Type: "装备", Name: "神风护腿", ItemType: "equip", Index: 5},
		{Type: "装备", Name: "神风护腰", ItemType: "equip", Index: 10},
		{Type: "装备", Name: "神风战靴", ItemType: "equip", Index: 12},
	}

	result := rebuildRoleEquipmentAppearanceSourceQuery(sourceQuery, items)
	for _, part := range []string{"h=30", "a=29", "wr=25", "w3=49", "c=17", "p=16", "b=14", "se=12"} {
		if !strings.Contains(result, part) {
			t.Fatalf("expected captured woodcutter source query to include %s, got %q", part, result)
		}
	}
	if strings.Contains(result, "w3=43") {
		t.Fatalf("expected captured woodcutter final source query to remove stale w3=43, got %q", result)
	}
}

func TestStorePromotedEquipmentAppearanceMappings(t *testing.T) {
	for _, spec := range []struct {
		name  string
		key   string
		value string
	}{
		{name: "天魁", key: "w1", value: "62"},
		{name: "武雷拳套", key: "w5", value: "64"},
		{name: "央月九影", key: "w8", value: "61"},
		{name: "炎爆龙鳞甲", key: "c", value: "28"},
		{name: "狼人护腿", key: "p", value: "29"},
		{name: "蛮族护腿", key: "p", value: "27"},
		{name: "炎爆之靴", key: "se", value: "21"},
		{name: "炎爆护手", key: "wr", value: "21"},
		{name: "蛤蟆精护腕", key: "wr", value: "26"},
		{name: "八极法冠", key: "h", value: "26"},
		{name: "八极法靴", key: "se", value: "3"},
		{name: "八极法衣", key: "c", value: "6"},
		{name: "八极法杖", key: "w10", value: "21"},
		{name: "八极护肩", key: "a", value: "3"},
		{name: "八极护腿", key: "p", value: "6"},
		{name: "八极护腰", key: "b", value: "4"},
		{name: "翠带护腰", key: "b", value: "32"},
		{name: "刀客布衣", key: "c", value: "30"},
		{name: "盗贼布衣", key: "c", value: "34"},
		{name: "盗贼护腿", key: "p", value: "30"},
		{name: "盗贼腰带", key: "b", value: "35"},
		{name: "蛤蟆布衣", key: "c", value: "37"},
		{name: "机木护腰", key: "b", value: "39"},
		{name: "狼牙棒", key: "w8", value: "33"},
		{name: "雷霆法杖", key: "w10", value: "51"},
		{name: "雷霆冠", key: "h", value: "17"},
		{name: "雷霆护肩", key: "a", value: "15"},
		{name: "神风护肩", key: "a", value: "13"},
		{name: "威武皮甲", key: "c", value: "14"},
		{name: "威武皮靴", key: "se", value: "10"},
		{name: "威武腰带", key: "b", value: "11"},
		{name: "无双护肩", key: "a", value: "14"},
		{name: "无双护腿", key: "p", value: "18"},
		{name: "无双铁腰带", key: "b", value: "16"},
		{name: "无双头盔", key: "h", value: "16"},
		{name: "岩化护腿", key: "p", value: "49"},
		{name: "饮血刀", key: "w8", value: "47"},
		{name: "珍元护腰", key: "b", value: "45"},
		{name: "L小白马", key: "r", value: "xmj"},
		{name: "萌兔宝宝", key: "r", value: "mengtubaobao"},
		{name: "狰狞神骑", key: "r", value: "zn"},
	} {
		template, ok := CapturedRoleItemTemplate(spec.name)
		if !ok {
			t.Fatalf("expected item-table template for %s", spec.name)
		}
		key, value, ok := roleItemAppearanceSourceParam(template)
		if !ok || key != spec.key || value != spec.value {
			t.Fatalf("expected %s appearance %s=%s, got %s=%s ok=%v", spec.name, spec.key, spec.value, key, value, ok)
		}
		query := applyRoleItemAppearanceToSourceQuery("human/human.swf?sex=1&", template)
		if !strings.Contains(query, spec.key+"="+spec.value+"&") {
			t.Fatalf("expected %s source query to include %s=%s, got %q", spec.name, spec.key, spec.value, query)
		}
	}
}

func TestStorePromotedFashionAppearanceMappings(t *testing.T) {
	template, ok := CapturedRoleItemTemplate("普通礼服")
	if !ok {
		t.Fatal("expected item-table template for 普通礼服")
	}
	params, ok := capturedFashionAppearanceSourceParams(template, "human/human.swf?sex=1&")
	if !ok {
		t.Fatal("expected 普通礼服 appearance parameters")
	}
	query := applyRoleItemAppearanceToSourceQuery("human/human.swf?sex=1&", template)
	for _, param := range params {
		if !strings.Contains(query, param.key+"="+param.value+"&") {
			t.Fatalf("expected 普通礼服 source query to include %s=%s, got %q", param.key, param.value, query)
		}
	}
}

func TestStoreCapturedFashionAppearanceDataCoversMallCatalog(t *testing.T) {
	store := NewStore()
	fashionCount := 0
	var untemplated mall.Product
	for offset := 0; ; offset += mall.PageSize {
		page := store.Mall.SearchPage(mall.SearchRequest{CategoryID: "15", Offset: offset, Limit: mall.PageSize})
		if len(page.Products) == 0 {
			break
		}
		for _, product := range page.Products {
			if !strings.Contains(product.Description, "&24@幻·时装") {
				continue
			}
			fashionCount += 1
			item := RoleItem{Name: product.Name, ItemType: "equip"}
			if !isCapturedFashionAppearanceItem(item) {
				t.Fatalf("expected captured appearance mapping for %s", product.Name)
			}
			variants, err := classicdata.FindFashionAppearanceRowsByName(product.Name)
			if err != nil {
				t.Fatalf("load captured appearance rows for %s: %v", product.Name, err)
			}
			if len(variants) == 0 {
				t.Fatalf("expected at least one captured sex variant for %s", product.Name)
			}
			for _, variant := range variants {
				sex := variant["sex"]
				rawParams := variant["source_params"]
				params, ok := capturedFashionAppearanceSourceParams(item, "human/human.swf?sex="+sex+"&")
				if !ok || len(params) == 0 {
					t.Fatalf("expected %s sex=%s appearance parameters", product.Name, sex)
				}
				query := applyRoleItemAppearanceToSourceQuery("human/human.swf?sex="+sex+"&h=1&r=zn&", item)
				for _, param := range params {
					if !strings.Contains(query, param.key+"="+param.value+"&") {
						t.Fatalf("expected %s sex=%s source query to include %s=%s, got %q", product.Name, sex, param.key, param.value, query)
					}
				}
				if rawParams == "" {
					t.Fatalf("captured fashion %s sex=%s must not use an empty mapping", product.Name, sex)
				}
			}
			if _, found := CapturedRoleItemTemplate(product.Name); !found && untemplated.Name == "" {
				untemplated = product
			}
		}
		if len(page.Products) < mall.PageSize {
			break
		}
	}
	if fashionCount != 56 {
		t.Fatalf("expected 56 captured mall fashion products, got %d", fashionCount)
	}
	if untemplated.Name == "" {
		t.Fatal("expected at least one catalog fashion without an item-table template")
	}

	login := mustLogin(t, store, "mockuser", "magicpwd")
	created := store.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "商城时装校验",
		Gender:         "male",
		RoleTemplateID: 1,
	})
	if !created.Success {
		t.Fatalf("expected role creation, got %+v", created)
	}
	store.rolesByPID[login.PlayerID][0].Currencies[untemplated.Currency] = untemplated.Price
	purchase := store.PurchaseMallProduct(login.PlayerID, created.Role.RoleID, untemplated, 1, "captured-fashion-untemplated")
	if !purchase.Success {
		t.Fatalf("expected untemplated fashion purchase, got %+v", purchase)
	}
	items, _, ok := store.GetRoleItems(login.PlayerID, created.Role.RoleID, "商城")
	if !ok || len(items) != 1 || items[0].Name != untemplated.Name || items[0].ItemType != "equip" {
		t.Fatalf("expected untemplated fashion catalog delivery as equip, ok=%v items=%+v", ok, items)
	}
	preview := store.PreviewTryEquip(login.PlayerID, created.Role.RoleID, untemplated.Name)
	if !preview.Found || !preview.Previewed {
		t.Fatalf("expected catalog fashion TryEquip preview, got %+v", preview)
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

func TestPersistentStoreKeepsQuestProgressAfterRestart(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "quest-progress.db")
	firstStore, err := NewPersistentStore(dbPath)
	if err != nil {
		t.Fatalf("expected persistent store init success, got error: %v", err)
	}

	login := mustLogin(t, firstStore, "mockuser", "magicpwd")
	created := firstStore.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "任务进度持久化",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	if !created.Success {
		t.Fatalf("expected role create success, got %+v", created)
	}
	if !firstStore.AcceptQuest(login.PlayerID, created.Role.RoleID, "收集朽木") {
		t.Fatal("expected 收集朽木 acceptance")
	}
	advanced := firstStore.AdvanceQuestProgress(login.PlayerID, created.Role.RoleID, "收集朽木", 3, 2)
	if !advanced.Found || !advanced.Updated || advanced.Completed || advanced.Current != 2 {
		t.Fatalf("expected persisted 2/3 quest progress, got %+v", advanced)
	}
	if err := firstStore.Close(); err != nil {
		t.Fatalf("expected first store close success, got error: %v", err)
	}

	secondStore, err := NewPersistentStore(dbPath)
	if err != nil {
		t.Fatalf("expected persistent store reopen success, got error: %v", err)
	}
	defer secondStore.Close()
	progress, ok := secondStore.QuestProgress(login.PlayerID, created.Role.RoleID, "收集朽木")
	if !ok || progress != 2 {
		t.Fatalf("expected restored 收集朽木 progress 2, ok=%v progress=%d", ok, progress)
	}
	completed := secondStore.AdvanceQuestProgress(login.PlayerID, created.Role.RoleID, "收集朽木", 3, 2)
	if !completed.Completed || completed.Current != 3 {
		t.Fatalf("expected progress to clamp at captured target 3, got %+v", completed)
	}
	if !secondStore.MarkQuestRemoved(login.PlayerID, created.Role.RoleID, "收集朽木") {
		t.Fatal("expected quest removal")
	}
	if _, ok := secondStore.QuestProgress(login.PlayerID, created.Role.RoleID, "收集朽木"); ok {
		t.Fatal("expected removed quest progress to be unavailable")
	}
}

func TestPersistentStoreRepairsFlattenedEquipmentRefinementDescription(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ai-server.db")
	firstStore, err := NewPersistentStore(dbPath)
	if err != nil {
		t.Fatalf("expected persistent store init success, got error: %v", err)
	}

	login := firstStore.Login(LoginRequest{UserName: "refineuser", Password: "magicpwd"})
	if !login.Success {
		t.Fatalf("expected persistent login success, got %+v", login)
	}
	created := firstStore.CreateRole(RoleCreateRequest{
		PlayerID:       login.PlayerID,
		SessionToken:   login.SessionToken,
		DisplayName:    "精炼持久女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	if !created.Success {
		t.Fatalf("expected persistent role create success, got %+v", created)
	}

	template, ok := CapturedRoleItemTemplate("狰狞神骑")
	if !ok {
		t.Fatal("expected captured 狰狞神骑 template")
	}
	flattenedDescription := strings.ReplaceAll(template.Description, "\n", "") + "&1@(+160)"
	firstStore.rolesByPID[login.PlayerID][0].Items = []RoleItem{{
		Type:        "装备",
		Name:        template.Name,
		Display:     template.Display,
		Description: flattenedDescription,
		Count:       1,
		Index:       18,
		Owner:       "持久化归属",
		Level:       7,
		ItemLevel:   9,
	}}
	if err := firstStore.persistRoleItemsLocked(login.PlayerID, created.Role.RoleID); err != nil {
		t.Fatalf("expected flattened refinement instance persistence, got error: %v", err)
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

	item, ok := secondStore.GetRoleItem(login.PlayerID, created.Role.RoleID, "装备", 18)
	if !ok {
		t.Fatal("expected repaired equipped mount")
	}
	if !strings.Contains(item.Description, "精炼潜质:\n[精炼+16]") || !strings.Contains(item.Description, "&1@(+160)") {
		t.Fatalf("expected canonical refinement line breaks with dynamic field preserved, got %q", item.Description)
	}
	if item.Owner != "持久化归属" || item.Level != 7 || item.ItemLevel != 9 || item.ItemType != "equip" {
		t.Fatalf("expected instance fields preserved while repairing refinement description, got %+v", item)
	}

	var persistedItemsJSON string
	if err := secondStore.db.QueryRow(`SELECT items_json FROM roles WHERE role_id = ? AND player_id = ?`, created.Role.RoleID, login.PlayerID).Scan(&persistedItemsJSON); err != nil {
		t.Fatalf("expected repaired items JSON query success, got error: %v", err)
	}
	var persistedItems []RoleItem
	if err := json.Unmarshal([]byte(persistedItemsJSON), &persistedItems); err != nil {
		t.Fatalf("expected repaired items JSON decode success, got error: %v", err)
	}
	if len(persistedItems) != 1 || !strings.Contains(persistedItems[0].Description, "精炼潜质:\n[精炼+16]") || !strings.Contains(persistedItems[0].Description, "&1@(+160)") {
		t.Fatalf("expected repaired line breaks and dynamic field in persisted data, got %+v", persistedItems)
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
