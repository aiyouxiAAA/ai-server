package main

import (
	"strings"
	"testing"

	"ai-server/internal/protocol"
)

func TestHandlePacketClassicAutoBattleInfoStartStopUsesCapturedBridge(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)

	info := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicAutoBattleInfoReq,
		Seq:     1,
		Payload: mustJSON(t, classicAutoBattleInfoRequest{}),
	}, socketSession)
	if !info.handled || info.autoBattleInfo == nil {
		t.Fatalf("expected c_ab bridge info, got %+v", info)
	}
	if info.autoBattleInfo.RemainingPoints != 0 || info.autoBattleInfo.PointCost != 0 || info.autoBattleInfo.Running {
		t.Fatalf("expected captured c_ab point-only idle state, got %+v", info.autoBattleInfo)
	}
	if !info.autoBattleInfo.Partial || !strings.Contains(info.autoBattleInfo.SourceCapture, "c_ab(50109)") {
		t.Fatalf("expected partial source capture citation, got %+v", info.autoBattleInfo)
	}

	start := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicAutoBattleStartReq,
		Seq: 2,
		Payload: mustJSON(t, classicAutoBattleStartRequest{
			DelaySeconds: 30,
			Full:         false,
			ExpFlag:      1,
			FillHp:       false,
			FillMp:       false,
			Relive:       false,
		}),
	}, socketSession)
	if !start.handled || start.autoBattleInfo == nil || !start.autoBattleInfo.Running {
		t.Fatalf("expected AutoBattle running bridge, got %+v", start)
	}
	if start.autoBattleInfo.DelaySeconds != 30 || start.autoBattleInfo.ExpFlag != 1 || start.autoBattleInfo.Full || start.autoBattleInfo.FillHp || start.autoBattleInfo.FillMp || start.autoBattleInfo.Relive {
		t.Fatalf("expected captured AutoBattle payload bridge, got %+v", start.autoBattleInfo)
	}
	if start.autoBattleInfo.StatusText != "正在挂机中..." || !strings.Contains(start.autoBattleInfo.SourceCapture, "AutoBattle(257)") {
		t.Fatalf("expected source running status and capture citation, got %+v", start.autoBattleInfo)
	}

	stop := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicAutoBattleStopReq,
		Seq:     3,
		Payload: mustJSON(t, classicAutoBattleStopRequest{}),
	}, socketSession)
	if !stop.handled || stop.autoBattleInfo == nil || stop.autoBattleInfo.Running {
		t.Fatalf("expected StopAutoBattle idle bridge, got %+v", stop)
	}
	if !strings.Contains(stop.autoBattleInfo.SourceCapture, "StopAutoBattle(260)") {
		t.Fatalf("expected StopAutoBattle capture citation, got %+v", stop.autoBattleInfo)
	}
}
