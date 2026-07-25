package main

import (
	"strings"
	"testing"

	"ai-server/internal/protocol"
	"ai-server/internal/session"
)

func TestClassicTownNPCShopPurchaseSweep(t *testing.T) {
	t.Run("item shops", func(t *testing.T) {
		for _, routes := range sourceGuangqingItemShopRoutes {
			for _, route := range routes {
				route := route
				t.Run(route.title, func(t *testing.T) {
					store, socketSession := seedSelectedRoleSession(t)
					openResult := handlePacketWithSession(store, protocol.Packet{
						Cmd: cmdClassicTownAnswerReq,
						Seq: 1,
						Payload: mustJSON(t, classicTownAnswerRequest{
							Handle:       route.handle,
							MsgHandle:    "1",
							AnswerHandle: route.answerHandle,
						}),
					}, socketSession)
					if !openResult.handled || openResult.skillShop == nil {
						t.Fatalf("expected NPC item shop to open, got %+v", openResult)
					}

					expectedRows := sourceItemShopEntries(route.vocation, route.rows)
					if len(expectedRows) == 0 || openResult.skillShop.ShopID != sourceItemShopRouteID(route) || len(openResult.skillShop.Skills) != len(expectedRows) {
						t.Fatalf("expected complete shop rows for %s, got %+v", route.title, openResult.skillShop)
					}

					for _, entry := range openResult.skillShop.Skills {
						entry := entry
						t.Run(entry.Name, func(t *testing.T) {
							purchaseStore, purchaseSession := seedSelectedRoleSession(t)
							row, ok := findSourceItemShopRow(openResult.skillShop.ShopID, entry.ID)
							if !ok {
								t.Fatalf("expected route row for %s (%d)", entry.Name, entry.ID)
							}
							fundClassicTownShopRequirements(t, purchaseStore, purchaseSession, row.requirements)
							assertClassicTownShopPurchase(t, purchaseStore, purchaseSession, openResult.skillShop.ShopID, entry.ID, row.name, row.count)
						})
					}
				})
			}
		}
	})

	t.Run("skill teacher entrances", func(t *testing.T) {
		for _, handle := range []string{
			sourceSkillTeacherHandle,
			guangqingSkillTeacherHandle,
			baiyuanSkillTeacherHandle,
			"6180542618405797",
		} {
			handle := handle
			t.Run(handle, func(t *testing.T) {
				for answerHandle, expectedShop := range sourceSkillTeacherShops {
					answerHandle, expectedShop := answerHandle, expectedShop
					t.Run(expectedShop.Title, func(t *testing.T) {
						store, socketSession := seedSelectedRoleSession(t)
						openResult := handlePacketWithSession(store, protocol.Packet{
							Cmd: cmdClassicTownAnswerReq,
							Seq: 1,
							Payload: mustJSON(t, classicTownAnswerRequest{
								Handle:       handle,
								MsgHandle:    "10",
								AnswerHandle: answerHandle,
							}),
						}, socketSession)
						if !openResult.handled || openResult.skillShop == nil || openResult.skillShop.ShopID != expectedShop.ShopID || len(openResult.skillShop.Skills) != len(expectedShop.Skills) {
							t.Fatalf("expected %s to open %s, got %+v", handle, expectedShop.Title, openResult)
						}
					})
				}
			})
		}
	})

	t.Run("skill shop goods", func(t *testing.T) {
		for _, shop := range sourceSkillTeacherShops {
			shop := shop
			t.Run(shop.Title, func(t *testing.T) {
				if len(shop.Skills) == 0 {
					t.Fatalf("expected captured skill rows for %s", shop.Title)
				}
				for _, entry := range shop.Skills {
					entry := entry
					t.Run(entry.Name, func(t *testing.T) {
						store, socketSession := seedSelectedRoleSession(t)
						fundClassicTownShopRequirements(t, store, socketSession, entry.Requirements)
						assertClassicTownShopPurchase(t, store, socketSession, shop.ShopID, entry.ID, entry.Name, 1)
					})
				}
			})
		}
	})
}

func fundClassicTownShopRequirements(
	t *testing.T,
	store *session.Store,
	socketSession *packetSession,
	requirements []classicTownSkillShopRequirement,
) {
	t.Helper()

	for _, requirement := range requirements {
		name := strings.TrimSpace(requirement.Name)
		if name == "" || requirement.Count <= 0 {
			continue
		}
		switch name {
		case "铜钱", "银元宝", "玉币":
			if _, ok := store.AddRoleCurrency(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, name, requirement.Count); !ok {
				t.Fatalf("expected currency %s to be funded", name)
			}
		default:
			item, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, session.RoleItem{
				Type:        "背包",
				Name:        name,
				ItemType:    requirement.ItemType,
				Display:     requirement.Icon,
				Description: requirement.Description,
				Count:       requirement.Count,
				Index:       -1,
				ItemLevel:   requirement.ItemLevel,
			})
			if !ok || item.Name != name {
				t.Fatalf("expected requirement %s to be funded, got %+v", name, item)
			}
		}
	}
}

func assertClassicTownShopPurchase(
	t *testing.T,
	store *session.Store,
	socketSession *packetSession,
	shopID string,
	entryID int,
	itemName string,
	itemCount int,
) {
	t.Helper()

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicTownBuySkillReq,
		Seq: 2,
		Payload: mustJSON(t, classicTownBuySkillRequest{
			ShopID:  shopID,
			SkillID: entryID,
		}),
	}, socketSession)
	if !result.handled || result.buySkillResult == nil || !result.buySkillResult.Success {
		t.Fatalf("expected purchase %s from %s to succeed, got %+v", itemName, shopID, result)
	}
	if result.buySkillResult.ShopID != shopID || result.buySkillResult.SkillID != entryID || !packetChatMessagesContain(result.chatMessages, "购买了【"+itemName+"】") {
		t.Fatalf("expected successful purchase pushes for %s from %s, got %+v", itemName, shopID, result)
	}
	if !classicTownItemInfosContain(result.itemInfos, itemName, itemCount) {
		t.Fatalf("expected purchased item push %s x%d, got %+v", itemName, itemCount, result.itemInfos)
	}

	items, _, found := store.GetRoleItems(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, "背包")
	if !found || !classicTownRoleItemsContain(items, itemName, itemCount) {
		t.Fatalf("expected purchased item %s x%d to persist in bag, got %+v", itemName, itemCount, items)
	}
}

func classicTownItemInfosContain(items []classicTownItemInfoPush, name string, count int) bool {
	for _, item := range items {
		if item.Name == name && item.Count >= count {
			return true
		}
	}
	return false
}

func classicTownRoleItemsContain(items []session.RoleItem, name string, count int) bool {
	for _, item := range items {
		if item.Name == name && item.Count >= count {
			return true
		}
	}
	return false
}
