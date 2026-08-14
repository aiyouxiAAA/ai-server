package main

import (
	"strconv"
	"strings"

	"ai-server/internal/session"
)

var classicChestWorldAnnouncementRewardNames = map[string]struct{}{
	"凿孔器":    {},
	"精炼宝石":   {},
	"贰级原石":   {},
	"高级精炼宝石": {},
}

func classicChestWorldAnnouncementMessage(
	roleName string,
	chestName string,
	reward session.RoleItem,
) (classicTownChatMessagePush, bool) {
	chestName = strings.TrimSpace(chestName)
	rewardName := strings.TrimSpace(reward.Name)
	if _, ok := classicChestWorldAnnouncementRewardNames[rewardName]; !ok {
		return classicTownChatMessagePush{}, false
	}

	return classicTownChatMessagePush{
		Channel: "system",
		Msg:     "<w>[" + strings.TrimSpace(roleName) + "]打开" + classicChestWorldAnnouncementColoredItem(chestName, session.CapturedRoleItemQualityColor(chestName, "")) + "获得了" + classicChestWorldAnnouncementColoredItem(rewardName, session.CapturedRoleItemQualityColor(rewardName, reward.Description)) + "x" + strconv.Itoa(reward.Count),
		Bold:    true,
	}, true
}

func classicChestWorldAnnouncementColoredItem(name string, color string) string {
	itemName := "[" + strings.TrimSpace(name) + "]"
	if strings.TrimSpace(color) == "" {
		return itemName
	}
	return "<font color='#" + strings.ToLower(strings.TrimSpace(color)) + "'>" + itemName + "</font>"
}
