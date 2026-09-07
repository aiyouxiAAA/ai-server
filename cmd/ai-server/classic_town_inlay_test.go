package main

import (
	"ai-server/internal/protocol"
	"ai-server/internal/session"
	"encoding/json"
	"testing"
)

func TestClassicTownInlayMalformedAndUnauthenticated(t *testing.T) {
	store := session.NewStore()
	for _, payload := range [][]byte{[]byte(`{`), []byte(`{}`), []byte(`{"socketIndex":1.5}`)} {
		got := handlePacketWithSession(store, protocol.Packet{Cmd: cmdClassicTownInlayRequest, Seq: 19, Payload: payload}, &packetSession{})
		var response classicTownInlayResponse
		if got.responseCmd != cmdClassicTownInlayResponse || json.Unmarshal(got.responsePayload, &response) != nil || response.Success || response.Message == "" {
			t.Fatal(got)
		}
	}
}
