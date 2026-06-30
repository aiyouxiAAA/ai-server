package main

import (
	"strings"
	"testing"

	"ai-server/internal/protocol"
)

func TestHandlePacketClassicDailyRewardInfoUsesCapturedStatusOnly(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicDailyRewardInfoReq,
		Seq:     1,
		Payload: mustJSON(t, classicDailyRewardInfoRequest{}),
	}, socketSession)
	if !result.handled || result.dailyRewardInfo == nil {
		t.Fatalf("expected daily reward info push, got %+v", result)
	}
	if result.dailyRewardInfo.RewardText != classicDailyRewardText || result.dailyRewardInfo.SourceMessage != classicDailyRewardSourceMessage {
		t.Fatalf("expected captured daily reward text, got %+v", result.dailyRewardInfo)
	}
	if result.dailyRewardInfo.LevelRequired != 30 || result.dailyRewardInfo.CanClaim || result.dailyRewardInfo.StatusText != "30级方能领取" {
		t.Fatalf("expected source level gate for low-level role, got %+v", result.dailyRewardInfo)
	}
	if !result.dailyRewardInfo.Partial || !strings.Contains(result.dailyRewardInfo.SourceCapture, "GetItemEveryDay(0,0)") {
		t.Fatalf("expected partial capture citation, got %+v", result.dailyRewardInfo)
	}
	if len(result.itemInfos) != 0 || len(result.itemClears) != 0 {
		t.Fatalf("daily reward status query must not grant unverified items, infos=%+v clears=%+v", result.itemInfos, result.itemClears)
	}

	socketSession.playerBase.Level = 30
	socketSession.selectedRole.Level = 30
	canClaimResult := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicDailyRewardInfoReq,
		Seq:     2,
		Payload: mustJSON(t, classicDailyRewardInfoRequest{}),
	}, socketSession)
	if !canClaimResult.handled || canClaimResult.dailyRewardInfo == nil || !canClaimResult.dailyRewardInfo.CanClaim {
		t.Fatalf("expected source get button to be enabled at level 30, got %+v", canClaimResult)
	}
	if canClaimResult.dailyRewardInfo.StatusText != "" || canClaimResult.dailyRewardInfo.Claimed {
		t.Fatalf("expected unclaimed level 30 source status, got %+v", canClaimResult.dailyRewardInfo)
	}
}

func TestHandlePacketClassicDailyRewardClaimKeepsUnverifiedGrantPartial(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)
	socketSession.playerBase.Level = 30
	socketSession.selectedRole.Level = 30

	result := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicDailyRewardClaimReq,
		Seq:     3,
		Payload: mustJSON(t, classicDailyRewardClaimRequest{}),
	}, socketSession)
	if !result.handled || result.dailyRewardInfo == nil {
		t.Fatalf("expected daily reward claim partial info push, got %+v", result)
	}
	if !result.dailyRewardInfo.CanClaim || result.dailyRewardInfo.Claimed {
		t.Fatalf("claim request must keep reward unclaimed until capture closes, got %+v", result.dailyRewardInfo)
	}
	if result.dailyRewardInfo.StatusText != classicDailyRewardClaimPartial {
		t.Fatalf("expected unverified claim status, got %+v", result.dailyRewardInfo)
	}
	if !result.dailyRewardInfo.Partial || !strings.Contains(result.dailyRewardInfo.SourceCapture, "GetItemEveryDay()") {
		t.Fatalf("expected empty-payload claim capture citation, got %+v", result.dailyRewardInfo)
	}
	if len(result.itemInfos) != 0 || len(result.itemClears) != 0 {
		t.Fatalf("daily reward claim must not grant unverified items, infos=%+v clears=%+v", result.itemInfos, result.itemClears)
	}
}
