package main

import (
	"fmt"
	"strings"

	"ai-server/internal/session"
	"ai-server/internal/world"
)

func buildClassicTownCollectionResult(
	store *session.Store,
	socketSession *packetSession,
	request classicTownRoleInteractionRequest,
) (packetResult, bool) {
	point, ok := world.FindSourceCollectionPoint(request.Handle)
	if !ok {
		return packetResult{}, false
	}
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil || store == nil {
		return packetResult{handled: true}, true
	}
	if strings.TrimSpace(request.Kind) != "collection" {
		return packetResult{handled: true}, true
	}
	if strings.TrimSpace(request.MapID) != fmt.Sprintf("%d", point.MapID) {
		return packetResult{handled: true}, true
	}
	if socketSession.playerBase.MapID != point.MapID {
		return packetResult{handled: true}, true
	}
	if !roleHasCollectionRequiredItem(store, socketSession, point.RequiredItemName) {
		return packetResult{
			answerSpeak: buildCollectionAnswerSpeak(request.Handle, "collect-missing-tool", "需要携带普通采集手套才能进行采集。"),
			handled:     true,
		}, true
	}

	rewardItem, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, buildCollectionRewardItem(point))
	if !ok {
		return packetResult{handled: true}, true
	}
	rewardItem.Handle = socketSession.selectedRole.RoleID

	if selectedRole, playerBase, ok := store.GetRoleRuntimeData(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID); ok {
		socketSession.selectedRole = &selectedRole
		socketSession.playerBase = &playerBase
	}

	return packetResult{
		answerSpeak: buildCollectionAnswerSpeak(request.Handle, "collect-success", "你获得了【"+point.RewardItemName+"】。"),
		itemInfos: []classicTownItemInfoPush{
			classicTownItemInfoPushFromRoleItem(rewardItem),
		},
		chatMessages: []classicTownChatMessagePush{
			classicTownSystemChatMessage("获得了【" + point.RewardItemName + "】x1"),
			classicTownSystemChatMessage("日志更新"),
		},
		questInfos: []classicQuestInfoPush{{
			Title:       point.QuestTitle,
			Level:       1,
			Description: point.QuestDescription,
			State:       "进行中",
		}},
		questStates: []world.QuestStatePush{{
			Handle: point.Handle,
			State:  point.QuestState,
		}},
		handled: true,
	}, true
}

func roleHasCollectionRequiredItem(store *session.Store, socketSession *packetSession, itemName string) bool {
	items, _, ok := store.GetRoleItems(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, "背包")
	if !ok {
		return false
	}
	for _, item := range items {
		if strings.TrimSpace(item.Name) == itemName && item.Count > 0 {
			return true
		}
	}
	return false
}

func buildCollectionRewardItem(point world.SourceCollectionPoint) session.RoleItem {
	if item, ok := session.CapturedRoleItemTemplate(point.RewardItemName); ok {
		item.Type = "背包"
		item.Count = 1
		item.Index = -1
		return item
	}
	return session.RoleItem{
		Type:        "背包",
		Name:        point.RewardItemName,
		ItemType:    "null",
		Display:     "",
		Description: "f_i_" + point.RewardItemName + "&24@材料&25@99&20@采集获得的材料。",
		Count:       1,
		Index:       -1,
		ItemLevel:   1,
	}
}

func buildCollectionAnswerSpeak(handle string, msgHandle string, message string) *world.AnswerSpeakPush {
	return &world.AnswerSpeakPush{
		Handle:    handle,
		MsgHandle: msgHandle,
		Msg:       message,
		Answers: []world.AnswerOption{
			{Handle: "close", Msg: "关闭"},
		},
	}
}
