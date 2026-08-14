package session

import (
	"log"
	"math/rand"
)

type classicChestRewardSpec struct {
	Name   string
	Count  int
	Weight int
}

// Each chest uses a 100-point, locally balanced table. Candidate items and
// quantities come from captured tooltips/results; original-server odds were
// not transmitted in the captures.
var classicChestRewardSpecs = map[string][]classicChestRewardSpec{
	"宝匣": {
		{Name: "初级精炼宝石", Count: 1, Weight: 14},
		{Name: "初阶经验卡", Count: 1, Weight: 9},
		{Name: "高级采集手套", Count: 20, Weight: 4},
		{Name: "千年灵芝", Count: 2, Weight: 7},
		{Name: "力量药水", Count: 1, Weight: 5},
		{Name: "智力药水", Count: 1, Weight: 5},
		{Name: "敏捷药水", Count: 1, Weight: 5},
		{Name: "耐力药水", Count: 1, Weight: 5},
		{Name: "幸运药水", Count: 1, Weight: 5},
		{Name: "装备重置符", Count: 1, Weight: 5},
		{Name: "一级经验丹", Count: 2, Weight: 7},
		{Name: "二级经验丹", Count: 2, Weight: 4},
		{Name: "刻刀", Count: 1, Weight: 4},
		{Name: "避怪符", Count: 1, Weight: 9},
		{Name: "大包还元散", Count: 5, Weight: 4},
		{Name: "小包还元散", Count: 5, Weight: 4},
		{Name: "大瓶甘露", Count: 5, Weight: 4},
	},
	"魔匣": {
		{Name: "初级精炼宝石", Count: 2, Weight: 18},
		{Name: "精炼宝石", Count: 1, Weight: 14},
		{Name: "壹级宝石还原符", Count: 1, Weight: 14},
		{Name: "凿孔器", Count: 1, Weight: 10},
		{Name: "初阶经验卡", Count: 2, Weight: 9},
		{Name: "宠物成长药剂", Count: 5, Weight: 14},
		{Name: "壹级原石", Count: 1, Weight: 10},
		{Name: "千年灵芝", Count: 3, Weight: 7},
		{Name: "奇效宠物药剂", Count: 5, Weight: 4},
	},
	"仙匣": {
		{Name: "贰级原石", Count: 1, Weight: 20},
		{Name: "高级精炼宝石", Count: 1, Weight: 10},
		{Name: "凿孔器", Count: 2, Weight: 15},
		{Name: "宠物成长药剂", Count: 5, Weight: 15},
		{Name: "壹级宝石还原符", Count: 2, Weight: 15},
		{Name: "精炼宝石", Count: 2, Weight: 20},
		{Name: "奇效宠物药剂", Count: 20, Weight: 5},
	},
}

func isClassicChest(name string) bool {
	_, ok := classicChestRewardSpecs[name]
	return ok
}

func classicChestRewardForRoll(chestName string, roll int) (classicChestRewardSpec, bool) {
	specs, ok := classicChestRewardSpecs[chestName]
	if !ok || len(specs) == 0 {
		return classicChestRewardSpec{}, false
	}
	for _, spec := range specs {
		if spec.Weight <= 0 {
			continue
		}
		if roll < spec.Weight {
			return spec, true
		}
		roll -= spec.Weight
	}
	return classicChestRewardSpec{}, false
}

func classicChestReward(chestName string) (RoleItem, bool) {
	specs, ok := classicChestRewardSpecs[chestName]
	if !ok || len(specs) == 0 {
		return RoleItem{}, false
	}
	totalWeight := 0
	for _, spec := range specs {
		if spec.Weight > 0 {
			totalWeight += spec.Weight
		}
	}
	if totalWeight <= 0 {
		return RoleItem{}, false
	}
	spec, ok := classicChestRewardForRoll(chestName, rand.Intn(totalWeight))
	if !ok {
		return RoleItem{}, false
	}
	item, ok := CapturedRoleItemTemplate(spec.Name)
	if !ok {
		return RoleItem{}, false
	}
	item.Count = spec.Count
	return item, true
}

func (store *Store) useClassicChestLocked(playerID string, roles []RoleSummary, roleIndex int, sourceItem RoleItem) RoleUseItemResult {
	reward, ok := classicChestReward(sourceItem.Name)
	if !ok {
		role := withRoleRuntimeDefaults(roles[roleIndex])
		return RoleUseItemResult{
			Role:         role,
			PlayerBase:   playerBaseDataFromRole(playerID, role),
			Item:         sourceItem,
			Found:        true,
			ErrorCode:    "chest_reward_missing",
			ErrorMessage: "宝匣奖励数据缺失。",
		}
	}

	baseCapacity, supported := roleContainerCapacity(sourceItem.Type)
	if !supported {
		role := withRoleRuntimeDefaults(roles[roleIndex])
		return RoleUseItemResult{
			Role:         role,
			PlayerBase:   playerBaseDataFromRole(playerID, role),
			Item:         sourceItem,
			Found:        true,
			ErrorCode:    "invalid_container",
			ErrorMessage: "invalid target container",
		}
	}

	capacity := roleContainerCapacityForRole(roles[roleIndex], sourceItem.Type, baseCapacity)
	remainingItems, updatedSource, clearedItems := consumeRoleItemBySlot(roles[roleIndex].Items, sourceItem.Type, sourceItem.Index, 1)
	reward.Type = sourceItem.Type
	reward.Index = -1
	// The bag update may carry a merged stack total, but the chest result and
	// announcement must report only the quantity rolled for this opening.
	openedReward := reward
	updatedItems, grantedReward, granted := grantRoleItemToItems(remainingItems, capacity, reward)
	if !granted {
		role := withRoleRuntimeDefaults(roles[roleIndex])
		return RoleUseItemResult{
			Role:         role,
			PlayerBase:   playerBaseDataFromRole(playerID, role),
			Item:         sourceItem,
			Found:        true,
			ErrorCode:    "chest_bag_full",
			ErrorMessage: "背包空间不足。",
		}
	}

	updatedResults := []RoleItem{grantedReward}
	if updatedSource != nil {
		updatedResults = append(updatedResults, *updatedSource)
	}
	roles[roleIndex].Items = normalizeRoleItems(updatedItems)
	store.rolesByPID[playerID] = roles
	acquisition := roleItemAcquisitionFromItem(
		playerID,
		roles[roleIndex].RoleID,
		openedReward,
		RoleItemAcquisitionSource{Kind: "开匣", Detail: sourceItem.Name},
	)
	acquisition.OccurredAtUnixMs = store.now().UnixMilli()
	if err := store.persistPlayerStateWithItemAcquisitionsLocked(playerID, []RoleItemAcquisition{acquisition}); err != nil {
		log.Printf("[session.Store] persist used %s failed: %v", sourceItem.Name, err)
	}

	role := withRoleRuntimeDefaults(roles[roleIndex])
	return RoleUseItemResult{
		Role:         role,
		PlayerBase:   playerBaseDataFromRole(playerID, role),
		Item:         sourceItem,
		ChestReward:  &openedReward,
		UpdatedItems: normalizeRoleItems(updatedResults),
		ClearedItems: clearedItems,
		Found:        true,
		Used:         true,
	}
}
