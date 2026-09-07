package main

import (
	"ai-server/internal/protocol"
	"ai-server/internal/session"
	"strings"
	"testing"
)

func TestHandlePacketBatchRefinementReturnsOnlyFinalState(t *testing.T) {
	store := session.NewStore()
	socket, role := seedSelectedRoleSessionInStore(t, store, "一键协议测试")
	gem, ok := store.GrantRoleItem(socket.playerBase.PlayerID, role.RoleID, session.RoleItem{Type: "背包", Name: "初级精炼宝石", ItemType: "oneI", Display: "616.png", Description: "f_i_初级精炼宝石&24@宝物&25@999", Count: 5, Index: -1})
	if !ok {
		t.Fatal("gem")
	}
	target, ok := store.GrantRoleItem(socket.playerBase.PlayerID, role.RoleID, session.RoleItem{Type: "装备", Name: "一键测试护肩", ItemType: "equip", Display: "603.png", Description: "f_i_一键测试护肩&24@护具·肩部&25@1&21@10&3@10&19@[精炼+1] 每升一级 物理防御+3", Count: 1, Index: 19})
	if !ok {
		t.Fatal("target")
	}
	index, goal := target.Index, 2
	request := classicTownActiveItemRequest{Type: gem.Type, Index: gem.Index, TargetType: target.Type, TargetIndex: &index, TargetLevel: &goal}
	result := handlePacketWithSession(store, protocol.Packet{Cmd: cmdClassicTownActiveItemReq, Seq: 29, Payload: mustJSON(t, request)}, socket)
	if !result.handled || result.rolePhysique == nil || len(result.chatMessages) != 1 || len(result.itemInfos) != 2 {
		t.Fatalf("unexpected output %+v", result)
	}
	if !packetChatMessagesContain(result.chatMessages, "一键精炼完成") || !packetChatMessagesContain(result.chatMessages, "消耗2颗") {
		t.Fatal(result.chatMessages)
	}
	items := itemInfosByName(result.itemInfos)
	if items[gem.Name].Count != 3 || items[target.Name].Level != 2 || !strings.Contains(items[target.Name].Description, "&3@10(+6)") {
		t.Fatal(items)
	}
	// Repeating the now-satisfied goal must not charge a second batch.
	result = handlePacketWithSession(store, protocol.Packet{Cmd: cmdClassicTownActiveItemReq, Seq: 30, Payload: mustJSON(t, request)}, socket)
	if len(result.itemInfos) != 0 || !packetChatMessagesContain(result.chatMessages, "目标精炼等级无效") {
		t.Fatal(result)
	}
}
