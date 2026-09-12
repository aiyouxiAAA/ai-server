package main

import (
	"ai-server/internal/battle"
	"ai-server/internal/protocol"
	"ai-server/internal/session"
	"ai-server/internal/world"
	"fmt"
	"strings"
	"testing"
)

func seedVillageServiceRole(t *testing.T, level, missingHP, missingMP int) (*session.Store, *packetSession) {
	t.Helper()
	store, socket := seedSelectedRoleSession(t)
	pid, rid := socket.playerBase.PlayerID, socket.selectedRole.RoleID
	store.SetRoleLevel(pid, rid, level)
	_, base, ok := store.UpdateRoleMap(pid, rid, 2)
	if !ok {
		t.Fatal("map setup")
	}
	state := *base.RoleState
	state.HP = base.RolePhysique.MaxHP - missingHP
	state.MP = base.RolePhysique.MaxMP - missingMP
	role, base, ok := store.UpdateRoleState(pid, rid, state)
	if !ok {
		t.Fatal("injury setup")
	}
	socket.selectedRole, socket.playerBase = &role, &base
	return store, socket
}

func villageAnswer(t *testing.T, store *session.Store, socket *packetSession, handle, answer string) packetResult {
	t.Helper()
	return handlePacketWithSession(store, protocol.Packet{Cmd: cmdClassicTownAnswerReq, Seq: 1, Payload: mustJSON(t, classicTownAnswerRequest{Handle: handle, MsgHandle: authoredServiceMessageID, AnswerHandle: answer})}, socket)
}

func TestAuthoredVillageServicesBootstrapAndMenus(t *testing.T) {
	store, socket := seedVillageServiceRole(t, 1, 50, 10)
	snapshot := world.BuildTownBootstrap(*socket.selectedRole, *socket.playerBase)
	applyAcceptedQuestStatesToBootstrap(&snapshot, store, socket)
	for _, npc := range world.AuthoredServiceNPCs() {
		count := 0
		for _, role := range snapshot.CreateRoles {
			if role.Handle == npc.Handle() {
				count++
				if role.SpawnFlash != npc.Spawn || role.DisplayName != npc.Name || role.Kind != "npc" {
					t.Fatalf("bad identity: %+v", role)
				}
			}
		}
		if count != 1 {
			t.Fatalf("NPC %s count %d", npc.Handle(), count)
		}
		opened := handlePacketWithSession(store, protocol.Packet{Cmd: cmdClassicTownActiveRoleReq, Seq: 1, Payload: mustJSON(t, classicTownRoleInteractionRequest{Handle: npc.Handle(), MapID: "2", Kind: "npc", NavigationToken: "service-nav"})}, socket)
		if opened.answerSpeak == nil || opened.answerSpeak.NavigationToken != "service-nav" || len(opened.answerSpeak.Answers) != len(npc.Services) {
			t.Fatalf("menu %+v", opened)
		}
	}
	snapshot.LoadMap.MapID = "6"
	snapshot.CreateRoles = nil
	world.AppendAuthoredServiceNPCs(&snapshot)
	if len(snapshot.CreateRoles) != 0 {
		t.Fatal("service NPCs leaked to field")
	}
}

func TestAuthoredVillageHealing(t *testing.T) {
	for _, tc := range []struct{ level, hp, mp, cost int }{{14, 50, 10, 0}, {19, 203, 0, 45}, {34, 263, 248, 120}} {
		store, socket := seedVillageServiceRole(t, tc.level, tc.hp, tc.mp)
		fundVillageServiceTest(t, store, socket)
		before := socket.playerBase.Currencies["铜钱"]
		opened, _ := buildAuthoredServiceOpenResult(socket, "authored-ye-zhidong")
		if opened.answerSpeak == nil || !strings.Contains(opened.answerSpeak.Answers[0].Msg, "疗伤") {
			t.Fatal("healing quote missing")
		}
		healed := villageAnswer(t, store, socket, "authored-ye-zhidong", "heal")
		if healed.roleState == nil || healed.roleState.HP != socket.playerBase.RolePhysique.MaxHP || healed.roleState.MP != socket.playerBase.RolePhysique.MaxMP || healed.currencyPush.Currencies["铜钱"] != before-tc.cost {
			t.Fatalf("healing %+v", healed)
		}
		again := villageAnswer(t, store, socket, "authored-ye-zhidong", "heal")
		if again.roleState != nil || again.answerSpeak == nil || again.answerSpeak.Answers[0].Handle != "heal" || again.currencyPush.Currencies["铜钱"] != before-tc.cost {
			t.Fatalf("full/repeat %+v", again)
		}
		_, base, ok := store.GetRoleRuntimeData(socket.playerBase.PlayerID, socket.selectedRole.RoleID)
		if !ok || base.RoleState.HP != base.RolePhysique.MaxHP || base.Currencies["铜钱"] != before-tc.cost {
			t.Fatal("healing not stored")
		}
	}
}

func TestAuthoredVillageServicesRejectInvalidState(t *testing.T) {
	store, socket := seedVillageServiceRole(t, 19, 203, 0)
	before := *socket.playerBase.RoleState
	bad := villageAnswer(t, store, socket, "authored-shi-li", "heal")
	if len(bad.errorMessages) == 0 || bad.roleState != nil {
		t.Fatal("wrong NPC healed")
	}
	bad, _ = buildAuthoredServiceAnswerResult(store, socket, classicTownAnswerRequest{Handle: "authored-ye-zhidong", MsgHandle: "stale", AnswerHandle: "heal"})
	if len(bad.errorMessages) == 0 {
		t.Fatal("stale dialogue accepted")
	}
	for _, state := range []string{"other-map", "battle"} {
		socket.playerBase.MapID = 2
		socket.battleRuntime = nil
		if state == "other-map" {
			socket.playerBase.MapID = 6
		} else {
			socket.battleRuntime = &battle.Runtime{}
		}
		bad = villageAnswer(t, store, socket, "authored-ye-zhidong", "heal")
		if len(bad.errorMessages) == 0 || bad.roleState != nil {
			t.Fatal("invalid healing state", state)
		}
		buy := buildClassicTownBuySkillResult(store, socket, classicTownBuySkillRequest{ShopID: "item:authored-ye-zhidong:medicine", SkillID: 0})
		if buy.buySkillResult == nil || buy.buySkillResult.Success || buy.buySkillResult.ErrorCode != "shop_unavailable" {
			t.Fatal("invalid purchase state", state)
		}
	}
	if *socket.playerBase.RoleState != before {
		t.Fatal("rejection changed health")
	}
}

func TestAuthoredVillageShopPurchaseSaleAndBuyback(t *testing.T) {
	store, socket := seedVillageServiceRole(t, 1, 0, 0)
	fundVillageServiceTest(t, store, socket)
	for _, tc := range []struct {
		handle, answer, name string
		capacity, price      int
	}{
		{"authored-ye-zhidong", "medicine", "馒头", 9, 10},
		{"authored-shi-li", "weapon", "蛮力钢剑", 10, 500},
		{"authored-shi-li", "armor", "布帽", 27, 20},
		{"authored-luo-qiancheng", "grocery", "普通采集手套", 12, 8},
	} {
		shop := villageAnswer(t, store, socket, tc.handle, tc.answer).skillShop
		if shop == nil || shop.Handle != tc.handle || shop.RoleName == "" || shop.SaleCapacity != tc.capacity || shop.Skills[0].Name != tc.name {
			t.Fatalf("shop %+v", shop)
		}
		before := socket.playerBase.Currencies["铜钱"]
		buy := buildClassicTownBuySkillResult(store, socket, classicTownBuySkillRequest{ShopID: shop.ShopID, SkillID: 0})
		if buy.buySkillResult == nil || !buy.buySkillResult.Success || buy.currencyPush.Currencies["铜钱"] != before-tc.price {
			t.Fatalf("buy %+v", buy)
		}
		if itemInfosByName(buy.itemInfos)[tc.name].Count != 1 {
			t.Fatalf("missing item %s", tc.name)
		}
	}
	loot := grantRoleItemTemplateForTest(t, store, socket, "肉", 2)
	before := socket.playerBase.Currencies["铜钱"]
	sale := buildClassicTownSaleItemResult(store, socket, classicTownSaleItemRequest{ShopID: "item:authored-luo-qiancheng:grocery", Type: "背包", Index: loot.Index, Count: 2})
	if sale.currencyPush == nil || sale.currencyPush.Currencies["铜钱"] <= before || len(socket.buyBackSoldEntries) == 0 {
		t.Fatalf("sale %+v", sale)
	}
	entry := socket.buyBackSoldEntries[len(socket.buyBackSoldEntries)-1]
	back := buildClassicTownBuyBackResult(store, socket, classicTownBuyBackRequest{Index: entry.Index})
	if itemInfosByName(back.itemInfos)["肉"].Count != 2 {
		t.Fatalf("buyback %+v", back)
	}
	invalid := buildClassicTownBuySkillResult(store, socket, classicTownBuySkillRequest{ShopID: "item:authored-shi-li:armor", SkillID: 999})
	if invalid.buySkillResult == nil || invalid.buySkillResult.ErrorCode != "item_missing" {
		t.Fatal("unknown item accepted")
	}
}

func fundVillageServiceTest(t *testing.T, store *session.Store, socket *packetSession) {
	t.Helper()
	pid, rid := socket.playerBase.PlayerID, socket.selectedRole.RoleID
	if _, ok := store.AddRoleCurrency(pid, rid, "铜钱", 5000); !ok {
		t.Fatal("isolated service funds fixture")
	}
	role, base, ok := store.GetRoleRuntimeData(pid, rid)
	if !ok {
		t.Fatal("isolated service role")
	}
	socket.selectedRole, socket.playerBase = &role, &base
}

func TestAuthoredVillageInsufficientFundsAndFullBag(t *testing.T) {
	store, socket := seedVillageServiceRole(t, 19, 203, 0)
	pid, rid := socket.playerBase.PlayerID, socket.selectedRole.RoleID
	item, _ := session.CapturedRoleItemTemplate("肉")
	item.Type, item.Index, item.Count = "背包", -1, 1
	drain := store.PurchaseRoleItem(pid, rid, item, []session.RoleItemRequirement{{Name: "铜钱", Count: socket.playerBase.Currencies["铜钱"]}})
	if !drain.Purchased {
		t.Fatal("funds fixture")
	}
	socket.selectedRole, socket.playerBase = &drain.Role, &drain.PlayerBase
	before := *socket.playerBase.RoleState
	rejected := villageAnswer(t, store, socket, "authored-ye-zhidong", "heal")
	if rejected.roleState != nil || !packetChatMessagesContain(rejected.chatMessages, "铜钱不足") || *socket.playerBase.RoleState != before {
		t.Fatal("insufficient healer mutated state")
	}
	buy := buildClassicTownBuySkillResult(store, socket, classicTownBuySkillRequest{ShopID: "item:authored-ye-zhidong:medicine", SkillID: 0})
	if buy.buySkillResult == nil || buy.buySkillResult.Success {
		t.Fatal("unfunded buy accepted")
	}
	store.AddRoleCurrency(pid, rid, "铜钱", 100)
	capacity, _ := store.GetRoleContainerCapacity(pid, rid, "背包")
	for i := 0; i < capacity; i++ {
		_, ok := store.GrantRoleItem(pid, rid, session.RoleItem{Type: "背包", Index: -1, Name: fmt.Sprintf("service-bag-fixture-%d", i), Count: 1, ItemType: "null", Description: "f_i_fixture&25@1&108@0"})
		if !ok {
			break
		}
	}
	role, base, _ := store.GetRoleRuntimeData(pid, rid)
	socket.selectedRole, socket.playerBase = &role, &base
	buy = buildClassicTownBuySkillResult(store, socket, classicTownBuySkillRequest{ShopID: "item:authored-ye-zhidong:medicine", SkillID: 0})
	if buy.buySkillResult == nil || buy.buySkillResult.Success || buy.currencyPush.Currencies["铜钱"] != 100 {
		t.Fatalf("full bag purchase %+v", buy.buySkillResult)
	}
}
