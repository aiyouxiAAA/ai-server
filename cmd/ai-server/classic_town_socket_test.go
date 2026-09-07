package main

import (
	"ai-server/internal/session"
	"strings"
	"testing"
)

func TestClassicTownSocketRoutesAndPushes(t *testing.T) {
	store := session.NewStore()
	connection, role := seedSelectedRoleSessionInStore(t, store, "凿孔协议测试")
	pid := connection.playerBase.PlayerID
	tool, ok := store.GrantRoleItem(pid, role.RoleID, session.RoleItem{Type: "背包", Name: "凿孔器", ItemType: "oneI", Display: "776.png", Description: "f_i_凿孔器&25@9999", Count: 1, Index: -1})
	if !ok {
		t.Fatal("tool grant")
	}
	target, ok := store.GrantRoleItem(pid, role.RoleID, session.RoleItem{Type: "装备", Name: "测试剑", ItemType: "equip", Display: "123.png", Description: "f_i_测试剑&23@凿孔上限 1 格&18@0", Count: 1, Index: 1})
	if !ok {
		t.Fatal("target grant")
	}
	index := target.Index
	goal := 2
	request := classicTownActiveItemRequest{Type: tool.Type, Index: tool.Index, TargetType: target.Type, TargetIndex: &index, TargetLevel: &goal}
	rejected := buildClassicTownActiveItemResult(store, connection, request)
	if !packetChatMessagesContain(rejected.chatMessages, "凿孔目标参数无效。") || len(rejected.itemInfos) > 0 {
		t.Fatal(rejected)
	}
	request.TargetLevel = nil
	result := buildClassicTownActiveItemResult(store, connection, request)
	if !result.handled || !packetChatMessagesContain(result.chatMessages, "凿孔成功") || len(result.itemClears) != 1 || len(result.itemInfos) != 1 {
		t.Fatal(result)
	}
	if !strings.Contains(result.itemInfos[0].Description, "&18@1") {
		t.Fatal(result.itemInfos)
	}
	again := buildClassicTownActiveItemResult(store, connection, request)
	if len(again.itemInfos) != 0 || len(again.itemClears) != 0 {
		t.Fatal("spent absent tool", again)
	}
}
