package main

import "testing"

// Historical packet fixtures are explicitly installed by capture-shape tests only.
var classicTownSourceBuyBackEntries = []classicTownSourceBuyBackEntry{
	{Index: 0, Name: "藤条", ItemType: "null", Display: "90.png", Description: "f_i_藤条&24@材料&25@99&20@密实坚固又轻巧坚韧的天然材料&0;具有不怕挤&0;不怕压&0;柔韧有弹性的特性.&103@0&104@0&105@&107@&108@31", Count: 5, ItemLevel: 1, Price: 155, SourceCapture: "D:/yzhgame/WOCClient/instances/instance2.staging/tmp/woc-proxy-captures/20260606_210926_394_session_08036/connections/20260606_215548_514_conn_0005/raw/server-to-client-0001.bin#2250"},
	{Index: 1, Name: "花瓣", ItemType: "null", Display: "89.png", Description: "f_i_花瓣&24@材料&25@99&20@花瓣具有显著的斑纹&0;或有蜜腺可以分泌蜜汁&0;产生含糖的花蜜&0;吸引昆虫.&103@0&104@0&105@&107@&108@33", Count: 6, ItemLevel: 1, Price: 198, SourceCapture: "D:/yzhgame/WOCClient/instances/instance2.staging/tmp/woc-proxy-captures/20260606_210926_394_session_08036/connections/20260606_215548_514_conn_0005/raw/server-to-client-0001.bin#2254"},
}

func TestClassicTownBuyBackNewSessionHasNoCapturedInventory(t *testing.T) {
	store, socket := seedSelectedRoleSession(t)
	if rows := buildClassicTownBuyBackListResult(socket).buyBackInfos; len(rows) != 0 {
		t.Fatalf("new session must have an empty buyback list: %+v", rows)
	}
	before := roleCurrenciesOrEmpty(store, socket.playerBase.PlayerID, socket.selectedRole.RoleID)["铜钱"]
	result := buildClassicTownBuyBackResult(store, socket, classicTownBuyBackRequest{Index: 0})
	if result.currencyPush != nil || len(result.itemInfos) != 0 {
		t.Fatalf("historical row cannot be purchased without a real sale: %+v", result)
	}
	if after := roleCurrenciesOrEmpty(store, socket.playerBase.PlayerID, socket.selectedRole.RoleID)["铜钱"]; after != before {
		t.Fatal("rejected historical row changed copper")
	}
}
