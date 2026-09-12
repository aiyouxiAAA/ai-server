package main

import (
	"log"

	"ai-server/internal/battle"
	"ai-server/internal/session"
)

// The 2026-09-12 reward popup is a receipt. Closing it never awards items.
// Use the inventory owner's stacking/capacity checks; only rejected items enter booty.
func appendClassicBattleRewardDelivery(store *session.Store, socket *packetSession, packet *packetResult) {
	if store == nil || socket == nil || socket.selectedRole == nil || socket.playerBase == nil || packet.battleOver == nil {
		return
	}
	result := &packet.battleOver.Result
	if result.Winner != battle.CampTeam || result.Escaped || result.RewardDeliveryComplete {
		return
	}
	roleID, playerID := socket.selectedRole.RoleID, socket.playerBase.PlayerID
	items := buildClassicBattleLoot(socket, *result)
	for _, item := range items {
		if item.Display == "" {
			log.Printf("[ai-server] missing battle reward item template name=%s battle=%s", item.Name, packet.battleOver.BattleID)
			packet.chatMessages = append(packet.chatMessages, classicTownSystemWarningMessage("战斗奖励配置缺失，奖励未发放。"))
			return
		}
	}
	before, _, found := store.GetRoleItems(playerID, roleID, "背包")
	if !found {
		return
	}
	beforeCounts := make(map[int]int, len(before))
	for _, item := range before {
		beforeCounts[item.Index] = item.Count
	}
	if result.CurrencyDelta > 0 {
		currencies, ok := store.AddRoleCurrency(playerID, roleID, "铜钱", result.CurrencyDelta)
		if !ok {
			log.Printf("[ai-server] battle currency delivery failed role=%s battle=%s", roleID, packet.battleOver.BattleID)
			packet.chatMessages = append(packet.chatMessages, classicTownSystemWarningMessage("战斗铜钱奖励发放失败。"))
			return
		}
		packet.currencyPush = buildClassicTownCurrencyPush(roleID, currencies)
	}
	result.RewardItems = make([]battle.RewardItem, 0, len(items))
	for _, item := range items {
		receipt := battle.RewardItem{Name: item.Name, Display: item.Display, Count: item.Count, ItemLevel: item.ItemLevel, Destination: "背包"}
		moved := item
		moved.Type, moved.Index = "背包", -1
		_, ok := store.GrantRoleItemWithSource(playerID, roleID, moved, session.RoleItemAcquisitionSource{
			Kind: "战斗奖励", Detail: "battle:" + packet.battleOver.BattleID,
		})
		if !ok {
			receipt.Destination = classicBattleLootType
			item.Index = nextClassicBattleLootIndex(socket.battleLoot)
			socket.battleLoot = append(socket.battleLoot, item)
		}
		result.RewardItems = append(result.RewardItems, receipt)
	}
	// Grant may fill several stacks; publish every changed stack, not only its first result.
	after, _, _ := store.GetRoleItems(playerID, roleID, "背包")
	for _, item := range after {
		if old, exists := beforeCounts[item.Index]; exists && old == item.Count {
			continue
		}
		item.Handle = roleID
		packet.itemInfos = append(packet.itemInfos, classicTownItemInfoPushFromRoleItem(item))
	}
	result.RewardDeliveryComplete = true
}

func nextClassicBattleLootIndex(items []session.RoleItem) int {
	used := make(map[int]bool, len(items))
	for _, item := range items {
		used[item.Index] = true
	}
	for index := 0; ; index++ {
		if !used[index] {
			return index
		}
	}
}

// Keep the existing eighteen-slot booty view, filling freed slots from retained overflow.
func promoteClassicBattleLootOverflow(socket *packetSession, packet *packetResult) {
	for i := range socket.battleLoot {
		item := &socket.battleLoot[i]
		if item.Index < classicBattleLootCap {
			continue
		}
		free := nextClassicBattleLootIndex(socket.battleLoot)
		if free >= classicBattleLootCap {
			return
		}
		packet.itemClears = append(packet.itemClears, classicTownItemInfoClearPush{Handle: socket.selectedRole.RoleID, Type: classicBattleLootType, Index: item.Index})
		item.Index = free
		item.Handle = socket.selectedRole.RoleID
		packet.itemInfos = append(packet.itemInfos, classicTownItemInfoPushFromRoleItem(*item))
	}
}
