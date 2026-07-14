package main

import (
	"testing"

	"ai-server/internal/protocol"
	"ai-server/internal/session"
)

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

func TestWuliangMap2NPCItemShopsUseCapturedSaleRows(t *testing.T) {
	testCases := []struct {
		name         string
		handle       string
		answerHandle string
		shopID       string
		title        string
		capacity     int
		firstItem    string
		lastItem     string
	}{
		{name: "Xuzhong healer", handle: "6360542618722932", answerHandle: "1", shopID: "item:6360542618722932", title: "虚中的药品商店", capacity: 9, firstItem: "馒头", lastItem: "解毒丸"},
		{name: "Yumo weapon", handle: "6370542618853300", answerHandle: "2", shopID: "item:6370542618853300:2", title: "虞莫的武器商店", capacity: 10, firstItem: "尘沙剑", lastItem: "银锭"},
		{name: "Yumo armor", handle: "6370542618853300", answerHandle: "3", shopID: "item:6370542618853300:3", title: "虞莫的护具商店", capacity: 24, firstItem: "龙颜钢盔", lastItem: "银锭"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, ok := buildClassicTownItemShopResult(classicTownAnswerRequest{
				Handle:       testCase.handle,
				MsgHandle:    "1",
				AnswerHandle: testCase.answerHandle,
			})
			if !ok || result.skillShop == nil {
				t.Fatalf("expected captured map191 item shop, got %+v ok=%v", result, ok)
			}
			shop := result.skillShop
			if shop.Handle != testCase.handle || shop.ShopID != testCase.shopID || shop.Title != testCase.title || shop.SaleCapacity != testCase.capacity || len(shop.Skills) != testCase.capacity {
				t.Fatalf("expected captured map191 shop header, got %+v", shop)
			}
			if shop.Skills[0].Name != testCase.firstItem || shop.Skills[len(shop.Skills)-1].Name != testCase.lastItem {
				t.Fatalf("expected captured map191 sale order, got %+v", shop.Skills)
			}
		})
	}

	weaponRow, ok := findSourceItemShopRow("item:6370542618853300:2", 0)
	if !ok || weaponRow.name != "尘沙剑" || len(weaponRow.requirements) != 1 || weaponRow.requirements[0].Name != "银元宝" || weaponRow.requirements[0].Count != 10 {
		t.Fatalf("expected captured Yumo weapon price, got %+v ok=%v", weaponRow, ok)
	}
	armorRow, ok := findSourceItemShopRow("item:6370542618853300:3", 1)
	if !ok || armorRow.name != "龙颜重铠" || len(armorRow.requirements) != 1 || armorRow.requirements[0].Name != "银元宝" || armorRow.requirements[0].Count != 3 {
		t.Fatalf("expected captured Yumo armor price, got %+v ok=%v", armorRow, ok)
	}
	materialRow, ok := findSourceItemShopRow("item:6370542618853300:2", 7)
	if !ok || materialRow.name != "铁块" || len(materialRow.requirements) != 2 || materialRow.requirements[0].Name != "铜钱" || materialRow.requirements[0].Count != 10 || materialRow.requirements[1].Name != "碎铁矿" || materialRow.requirements[1].Count != 10 {
		t.Fatalf("expected captured Yumo material requirements, got %+v ok=%v", materialRow, ok)
	}

	store, socketSession := seedSelectedRoleSession(t)
	if _, ok := store.AddRoleCurrency(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, "银元宝", 12); !ok {
		t.Fatal("expected test silver currency to be added")
	}
	buyResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownBuySkillReq,
		Seq: 1,
		Payload: mustJSON(t, classicTownBuySkillRequest{
			ShopID:  "item:6370542618853300:2",
			SkillID: 0,
		}),
	}, socketSession)
	if !buyResult.handled || buyResult.buySkillResult == nil || !buyResult.buySkillResult.Success || len(buyResult.itemInfos) != 1 || buyResult.itemInfos[0].Name != "尘沙剑" {
		t.Fatalf("expected Yumo weapon sale to consume captured currency and grant item, got %+v", buyResult)
	}
}

func TestWuliangBaguaAndRockCraftShopsUseCapturedRows(t *testing.T) {
	bagua, ok := buildClassicTownItemShopResult(classicTownAnswerRequest{
		Handle:       "6200542618485245",
		MsgHandle:    "1",
		AnswerHandle: "1",
	})
	if !ok || bagua.skillShop == nil {
		t.Fatalf("expected Wuliang Bagua craft shop, got %+v ok=%v", bagua, ok)
	}
	if bagua.skillShop.Handle != "6200542618485245" || bagua.skillShop.SaleCapacity != 10 || bagua.skillShop.Skills[0].Name != "狰狞神骑" || bagua.skillShop.Skills[9].Name != "盖世神功" {
		t.Fatalf("expected captured Bagua craft rows, got %+v", bagua.skillShop)
	}

	rock, ok := buildClassicTownItemShopResult(classicTownAnswerRequest{
		Handle:       "6190542618476150",
		MsgHandle:    "1",
		AnswerHandle: "3",
	})
	if !ok || rock.skillShop == nil {
		t.Fatalf("expected Wuliang rock craft shop, got %+v ok=%v", rock, ok)
	}
	if rock.skillShop.ShopID != "rock" || rock.skillShop.SaleCapacity != 2 || len(rock.skillShop.Skills) != 2 {
		t.Fatalf("expected captured rock shop header, got %+v", rock.skillShop)
	}
	if rock.skillShop.Skills[0].Name != "贰级原石" || rock.skillShop.Skills[1].Name != "叁级原石" || len(rock.skillShop.Skills[0].Requirements) != 5 || len(rock.skillShop.Skills[1].Requirements) != 5 {
		t.Fatalf("expected captured rock recipes, got %+v", rock.skillShop.Skills)
	}
	row, ok := findSourceItemShopRow(wuliangRockShopID, 0)
	if !ok || row.name != "贰级原石" || len(row.requirements) != 5 {
		t.Fatalf("expected rock shop id to resolve captured craft row, got %+v ok=%v", row, ok)
	}

	store, socketSession := seedSelectedRoleSession(t)
	buyResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownBuySkillReq,
		Seq: 1,
		Payload: mustJSON(t, classicTownBuySkillRequest{
			ShopID:  wuliangRockShopID,
			SkillID: 0,
		}),
	}, socketSession)
	if !buyResult.handled || buyResult.buySkillResult == nil || buyResult.buySkillResult.Success {
		t.Fatalf("expected rock recipe to use item-craft result path when materials are missing, got %+v", buyResult)
	}
	for _, requirement := range row.requirements {
		material, granted := store.GrantRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, session.RoleItem{
			Type:     "背包",
			Name:     requirement.Name,
			ItemType: requirement.ItemType,
			Display:  requirement.Icon,
			Count:    requirement.Count,
			Index:    -1,
		})
		if !granted || material.Name != requirement.Name {
			t.Fatalf("expected captured rock material %s to be granted, got %+v granted=%v", requirement.Name, material, granted)
		}
	}
	completedResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownBuySkillReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownBuySkillRequest{
			ShopID:  wuliangRockShopID,
			SkillID: 0,
		}),
	}, socketSession)
	if !completedResult.handled || completedResult.buySkillResult == nil || !completedResult.buySkillResult.Success || len(completedResult.itemInfos) != 1 || len(completedResult.itemClears) != 5 || completedResult.itemInfos[0].Name != "贰级原石" {
		t.Fatalf("expected rock craft to consume five fragments and grant 贰级原石, got %+v", completedResult)
	}
}

func TestWuliangChouSipinAuctionReplyOpensExistingAuctionPage(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownAnswerReq,
		Seq: 1,
		Payload: mustJSON(t, classicTownAnswerRequest{
			Handle:       "6190542618476150",
			MsgHandle:    "1",
			AnswerHandle: "2",
		}),
	}, socketSession)
	if result.auctionOpen == nil || result.auctionOpen.ContainerType != classicAuctionContainerType || result.auctionOpen.NPCHandle != "6190542618476150" {
		t.Fatalf("expected Chou Sipin auction reply to open existing auction page, got %+v", result)
	}
}

func TestWuliangZhenShanWeiSkillTeacherUsesCapturedCategoryAndSale(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	category := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownAnswerReq,
		Seq: 1,
		Payload: mustJSON(t, classicTownAnswerRequest{
			Handle:       "6180542618405797",
			MsgHandle:    "1",
			AnswerHandle: "1",
		}),
	}, socketSession)
	if category.answerSpeak == nil || category.answerSpeak.MsgHandle != "10" || category.answerSpeak.Msg != "你想学习什么职业的技能?" {
		t.Fatalf("expected captured Zhenshanwei skill category dialogue, got %+v", category.answerSpeak)
	}
	if len(category.answerSpeak.Answers) != 4 || category.answerSpeak.Answers[0].Handle != "7" || category.answerSpeak.Answers[1].Handle != "8" || category.answerSpeak.Answers[2].Handle != "9" {
		t.Fatalf("expected captured warrior/mage/ranger choices, got %+v", category.answerSpeak.Answers)
	}

	shopResult := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownAnswerReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownAnswerRequest{
			Handle:       "6180542618405797",
			MsgHandle:    "10",
			AnswerHandle: "9",
		}),
	}, socketSession)
	if shopResult.skillShop == nil {
		t.Fatal("expected Zhenshanwei to open the captured ranger skill shop")
	}
	if shopResult.skillShop.Handle != "6180542618405797" || shopResult.skillShop.ShopID != "skill3" || shopResult.skillShop.SkillCap != 26 {
		t.Fatalf("expected captured skill3 sale response, got %+v", shopResult.skillShop)
	}
}
