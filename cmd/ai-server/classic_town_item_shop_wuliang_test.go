package main

import "testing"

func TestWuliangChouSipinItemShopUsesCapturedSaleRows(t *testing.T) {
	result, ok := buildClassicTownItemShopResult(classicTownAnswerRequest{
		Handle:       "6190542618476150",
		MsgHandle:    "1",
		AnswerHandle: "1",
	})
	if !ok || result.skillShop == nil {
		t.Fatalf("expected Wuliang Chou Sipin item shop, got %+v ok=%v", result, ok)
	}
	shop := result.skillShop
	if shop.Handle != "6190542618476150" || shop.Title != "丑四品的道具商店" || shop.SaleCapacity != 11 {
		t.Fatalf("expected captured Chou Sipin shop header, got %+v", shop)
	}
	if len(shop.Skills) != 11 || shop.Skills[0].Name != "普通采集手套" || shop.Skills[10].Name != "毒箭" {
		t.Fatalf("expected captured Chou Sipin sale rows, got %+v", shop.Skills)
	}
}
