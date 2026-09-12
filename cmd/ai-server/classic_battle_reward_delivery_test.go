package main

import (
	"ai-server/internal/battle"
	"ai-server/internal/session"
	"strconv"
	"testing"
)

func rewardDeliveryPacket(items ...string) packetResult {
	return packetResult{battleOver: &battle.OverPush{BattleID: "receipt-test", Result: battle.ResultPayload{Winner: battle.CampTeam, Items: items}}}
}

func TestBattleRewardDeliveryBagReceiptAndNoRepeat(t *testing.T) {
	store, socket := seedSelectedRoleSession(t)
	packet := rewardDeliveryPacket("丝x2")
	appendClassicBattleRewardDelivery(store, socket, &packet)
	if !packet.battleOver.Result.RewardDeliveryComplete || len(packet.itemInfos) != 1 || len(socket.battleLoot) != 0 {
		t.Fatalf("expected bag delivery: %+v, loot=%+v", packet, socket.battleLoot)
	}
	r := packet.battleOver.Result.RewardItems[0]
	if r.Name != "丝" || r.Count != 2 || r.Destination != "背包" || r.Display == "" {
		t.Fatalf("bad receipt: %+v", r)
	}
	appendClassicBattleRewardDelivery(store, socket, &packet)
	if len(packet.itemInfos) != 1 {
		t.Fatal("same result awarded twice")
	}
}

func TestBattleRewardDeliveryFullBagRetainsPreviousOverflow(t *testing.T) {
	store, socket := seedSelectedRoleSession(t)
	fillRewardReceiptBag(t, store, socket)
	socket.battleLoot = []session.RoleItem{{Name: "旧奖励", Type: "战斗", Count: 1, Index: 0}}
	packet := rewardDeliveryPacket("丝x2", "肉x1")
	appendClassicBattleRewardDelivery(store, socket, &packet)
	if len(packet.itemInfos) != 0 || len(socket.battleLoot) != 3 || socket.battleLoot[0].Name != "旧奖励" {
		t.Fatalf("lost overflow: %+v", socket.battleLoot)
	}
	for i, r := range packet.battleOver.Result.RewardItems {
		if r.Destination != "战斗" || socket.battleLoot[i+1].Index != i+1 {
			t.Fatalf("bad overflow receipt: %+v", r)
		}
	}
	loss := rewardDeliveryPacket("丝x1")
	loss.battleOver.Result.Winner = battle.CampEnemy
	appendClassicBattleRewardDelivery(store, socket, &loss)
	escape := rewardDeliveryPacket("丝x1")
	escape.battleOver.Result.Escaped = true
	appendClassicBattleRewardDelivery(store, socket, &escape)
	if len(socket.battleLoot) != 3 || loss.battleOver.Result.RewardDeliveryComplete || escape.battleOver.Result.RewardDeliveryComplete {
		t.Fatal("loss/escape changed rewards")
	}
}

func TestBattleRewardDeliveryStacksBeforeOverflowAndKeepsDelta(t *testing.T) {
	store, socket := seedSelectedRoleSession(t)
	first := rewardDeliveryPacket("丝x2")
	appendClassicBattleRewardDelivery(store, socket, &first)
	fillRewardReceiptBag(t, store, socket)
	next := rewardDeliveryPacket("丝x3", "肉x1")
	appendClassicBattleRewardDelivery(store, socket, &next)
	r := next.battleOver.Result.RewardItems
	if len(r) != 2 || r[0].Destination != "背包" || r[0].Count != 3 || r[1].Destination != "战斗" {
		t.Fatalf("expected stack then overflow: %+v", r)
	}
}

func TestBattleRewardDeliveryPublishesAllChangedStacks(t *testing.T) {
	store, socket := seedSelectedRoleSession(t)
	first := rewardDeliveryPacket("丝x98")
	appendClassicBattleRewardDelivery(store, socket, &first)
	next := rewardDeliveryPacket("丝x3")
	appendClassicBattleRewardDelivery(store, socket, &next)
	if len(next.itemInfos) != 2 || next.battleOver.Result.RewardItems[0].Count != 3 {
		t.Fatalf("expected both split stacks and receipt delta, got %+v", next)
	}
}

func TestBattleRewardDeliveryRetainsMoreThanEighteen(t *testing.T) {
	store, socket := seedSelectedRoleSession(t)
	fillRewardReceiptBag(t, store, socket)
	names := make([]string, 20)
	for i := range names {
		names[i] = "丝x1"
	}
	packet := rewardDeliveryPacket(names...)
	appendClassicBattleRewardDelivery(store, socket, &packet)
	if len(socket.battleLoot) != 20 || len(packet.battleOver.Result.RewardItems) != 20 {
		t.Fatal("reward list was truncated to booty viewport capacity")
	}
	socket.battleLoot = socket.battleLoot[1:]
	promoted := packetResult{}
	promoteClassicBattleLootOverflow(socket, &promoted)
	if len(promoted.itemInfos) != 1 || promoted.itemInfos[0].Index != 0 || len(socket.battleLoot) != 19 {
		t.Fatalf("queued reward did not fill freed slot: %+v", promoted)
	}
}

func TestBattleRewardDeliveryCurrencyAndMissingMetadata(t *testing.T) {
	store, socket := seedSelectedRoleSession(t)
	bad := rewardDeliveryPacket("missing-reward-template")
	appendClassicBattleRewardDelivery(store, socket, &bad)
	if bad.battleOver.Result.RewardDeliveryComplete || len(bad.chatMessages) != 1 || len(socket.battleLoot) != 0 {
		t.Fatalf("unknown metadata must report incomplete delivery: %+v", bad)
	}
	packet := rewardDeliveryPacket()
	packet.battleOver.Result.CurrencyDelta = 36
	appendClassicBattleRewardDelivery(store, socket, &packet)
	if !packet.battleOver.Result.RewardDeliveryComplete || packet.currencyPush == nil {
		t.Fatal("missing currency delivery")
	}
}

func fillRewardReceiptBag(t *testing.T, store *session.Store, socket *packetSession) {
	t.Helper()
	pid, rid := socket.playerBase.PlayerID, socket.selectedRole.RoleID
	capacity, ok := store.GetRoleContainerCapacity(pid, rid, "背包")
	if !ok {
		t.Fatal("missing bag capacity")
	}
	for index := 0; index < capacity; index++ {
		if _, occupied := store.GetRoleItem(pid, rid, "背包", index); occupied {
			continue
		}
		if _, granted := store.GrantRoleItem(pid, rid, session.RoleItem{Type: "背包", Index: index, Name: "receipt-fill-" + strconv.Itoa(index), Count: 1, ItemType: "own"}); !granted {
			t.Fatal("failed filler")
		}
	}
}
