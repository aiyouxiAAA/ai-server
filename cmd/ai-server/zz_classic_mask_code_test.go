package main

import (
	"testing"

	"ai-server/internal/protocol"
)

func TestHandlePacketClassicMaskCodeRefreshAndSubmit(t *testing.T) {
	store, socketSession := seedSelectedRoleSession(t)

	refreshResult := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicMaskCodeRefreshReq,
		Seq:     1,
		Payload: mustJSON(t, classicMaskCodeRefreshRequest{}),
	}, socketSession)
	if !refreshResult.handled {
		t.Fatalf("mask code refresh was not handled")
	}
	if refreshResult.maskCodeChallenge == nil {
		t.Fatalf("expected mask code challenge push")
	}
	if refreshResult.maskCodeChallenge.Width != classicMaskCodeWidth || refreshResult.maskCodeChallenge.Height != classicMaskCodeHeight {
		t.Fatalf("unexpected mask code dimensions: %#v", refreshResult.maskCodeChallenge)
	}
	if len(refreshResult.maskCodeChallenge.Code) != classicMaskCodeWidth*classicMaskCodeHeight {
		t.Fatalf("unexpected mask code length: %d", len(refreshResult.maskCodeChallenge.Code))
	}
	if !refreshResult.maskCodeChallenge.Partial || refreshResult.maskCodeChallenge.SourceCapture == "" {
		t.Fatalf("mask code challenge must cite partial capture evidence: %#v", refreshResult.maskCodeChallenge)
	}

	submitResult := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicMaskCodeSubmitReq,
		Seq:     2,
		Payload: mustJSON(t, classicMaskCodeSubmitRequest{Value: "xgjr"}),
	}, socketSession)
	if !submitResult.handled {
		t.Fatalf("mask code submit was not handled")
	}
	if submitResult.maskCodeResult == nil || !submitResult.maskCodeResult.Accepted {
		t.Fatalf("expected accepted mask code result: %#v", submitResult.maskCodeResult)
	}
	if !submitResult.maskCodeResult.Partial || submitResult.maskCodeResult.SourceCapture == "" {
		t.Fatalf("mask code result must cite partial capture evidence: %#v", submitResult.maskCodeResult)
	}
}
