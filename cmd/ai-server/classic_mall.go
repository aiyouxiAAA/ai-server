package main

import (
	"log"

	"ai-server/internal/mall"
	"ai-server/internal/session"
)

func buildClassicMallCategoryListResult(store *session.Store, socketSession *packetSession) packetResult {
	if !hasSelectedMallRole(socketSession) || store == nil || store.Mall == nil {
		log.Printf("[ai-server] classic mall CategoryList ignored without selected role")
		return packetResult{handled: true}
	}
	return packetResult{
		mallCategories: store.Mall.Categories(),
		mallCurrency:   buildClassicMallCurrencyPush(store, socketSession),
		handled:        true,
	}
}

func buildClassicMallSearchCountResult(store *session.Store, socketSession *packetSession, request mall.SearchRequest) packetResult {
	if !hasSelectedMallRole(socketSession) || store == nil || store.Mall == nil {
		log.Printf("[ai-server] classic mall SearchCount ignored without selected role")
		return packetResult{handled: true}
	}
	count := store.Mall.SearchCount(request)
	return packetResult{
		mallSearchCount: &count,
		handled:         true,
	}
}

func buildClassicMallSearchPageResult(store *session.Store, socketSession *packetSession, request mall.SearchRequest) packetResult {
	if !hasSelectedMallRole(socketSession) || store == nil || store.Mall == nil {
		log.Printf("[ai-server] classic mall SearchPage ignored without selected role")
		return packetResult{handled: true}
	}
	page := store.Mall.SearchPage(request)
	return packetResult{
		mallSearchPage: &page,
		mallCurrency:   buildClassicMallCurrencyPush(store, socketSession),
		handled:        true,
	}
}

func buildClassicMallPurchaseResult(store *session.Store, socketSession *packetSession, request mall.PurchaseRequest) packetResult {
	if !hasSelectedMallRole(socketSession) || store == nil || store.Mall == nil {
		log.Printf("[ai-server] classic mall Purchase ignored without selected role productId=%s", request.ProductID)
		return packetResult{handled: true}
	}
	product, ok := store.Mall.FindProduct(request.ProductID)
	if !ok {
		result := mall.PurchaseResult{
			Success:      false,
			ProductID:    request.ProductID,
			Quantity:     request.Quantity,
			RequestID:    request.RequestID,
			ErrorCode:    mall.PRODUCT_NOT_FOUND,
			ErrorMessage: "商品不存在。",
		}
		return packetResult{mallPurchase: &result, handled: true}
	}
	result := store.PurchaseMallProduct(
		socketSession.playerBase.PlayerID,
		socketSession.selectedRole.RoleID,
		product,
		request.Quantity,
		request.RequestID,
	)
	if result.Success {
		currencies, ok := store.GetRoleCurrencies(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID)
		if ok {
			socketSession.selectedRole.Currencies = currencies
			socketSession.playerBase.Currencies = currencies
		}
	}
	return packetResult{
		mallPurchase: &result,
		mallCurrency: &mall.CurrencyPush{
			CurrencyName: result.CurrencyName,
			Balance:      result.CurrencyBalance,
		},
		handled: true,
	}
}

func buildClassicMallCurrencyPush(store *session.Store, socketSession *packetSession) *mall.CurrencyPush {
	if store == nil || socketSession == nil || socketSession.playerBase == nil || socketSession.selectedRole == nil {
		return nil
	}
	currencies, ok := store.GetRoleCurrencies(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID)
	if !ok {
		return nil
	}
	// Source rmb_shop binds loginState from proxy_user_money as 玉币.
	// Prefer 玉币; fall back to the old 银元宝 seed for existing roles.
	balance := currencies[mall.SourceYubiCurrencyName]
	if balance <= 0 {
		balance = currencies[mall.DevCurrencyName]
	}
	return &mall.CurrencyPush{
		CurrencyName: mall.SourceYubiCurrencyName,
		Balance:      balance,
	}
}

func hasSelectedMallRole(socketSession *packetSession) bool {
	return socketSession != nil && socketSession.selectedRole != nil && socketSession.playerBase != nil
}
