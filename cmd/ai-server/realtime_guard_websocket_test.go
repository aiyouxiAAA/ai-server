package main

import (
	"ai-server/internal/battle"
	"ai-server/internal/protocol"
	"ai-server/internal/session"
	"encoding/json"
	"github.com/gorilla/websocket"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRealtimeGuardWebsocketKicksInvalidMovement(t *testing.T) {
	store := session.NewStore()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { handleWebSocket(store, w, r) }))
	defer server.Close()
	connection, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	sequence := uint64(0)
	send := func(cmd uint64, payload interface{}) {
		sequence++
		if err := connection.WriteMessage(websocket.BinaryMessage, protocol.Encode(protocol.Packet{Cmd: cmd, Seq: sequence, Payload: mustJSON(t, payload), TimestampMs: 99999999999999})); err != nil {
			t.Fatal(err)
		}
	}
	receive := func(cmd uint64, target interface{}) {
		t.Helper()
		_ = connection.SetReadDeadline(time.Now().Add(5 * time.Second))
		for {
			_, data, err := connection.ReadMessage()
			if err != nil {
				t.Fatal(err)
			}
			packet, err := protocol.Decode(data)
			if err != nil {
				t.Fatal(err)
			}
			if packet.Cmd == cmd {
				if err := json.Unmarshal(packet.Payload, target); err != nil {
					t.Fatal(err)
				}
				return
			}
		}
	}
	send(cmdAuthLoginRequest, session.LoginRequest{UserName: "wireguard", Password: "guardpassword"})
	var login session.LoginResponse
	receive(cmdAuthLoginResponse, &login)
	if !login.Success {
		t.Fatal("login")
	}
	send(cmdRoleCreateRequest, session.RoleCreateRequest{PlayerID: login.PlayerID, SessionToken: login.SessionToken, DisplayName: "验异常", Gender: "female", RoleTemplateID: 1})
	var created session.RoleCreateResponse
	receive(cmdRoleCreateResponse, &created)
	if !created.Success {
		t.Fatal("create")
	}
	send(cmdRoleSelectRequest, session.RoleSelectRequest{PlayerID: login.PlayerID, SessionToken: login.SessionToken, RoleID: created.Role.RoleID})
	var selected session.RoleSelectResponse
	receive(cmdRoleSelectResponse, &selected)
	if !selected.Success {
		t.Fatal("select")
	}
	send(cmdClassicTownMoveRoleReq, classicTownMoveRoleRequest{Type: "Run", Handle: created.Role.RoleID, MapID: "2", X: 500000, Y: 600, TX: 500000, TY: 600})
	_ = connection.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		_, _, err := connection.ReadMessage()
		if err == nil {
			continue
		}
		close, ok := err.(*websocket.CloseError)
		if !ok || close.Code != 1008 || close.Text != "invalid_gameplay_request" {
			t.Fatalf("expected policy kick, got %v", err)
		}
		break
	}
}

func TestRealtimeGuardSettlementReleasedOnceAtDeadline(t *testing.T) {
	store, sockets := guardBattleFixture(t, 1)
	s := sockets[0]
	r := s.battleRuntime
	for i := range r.Cells {
		if r.Cells[i].Camp == battle.CampEnemy {
			r.Cells[i].HP = 1
			r.Cells[i].Dog = 0
		}
	}
	now := time.Now()
	guardAction(t, store, s, now)
	if r.PendingOver == nil {
		t.Fatal("expected pending win")
	}
	before, _, _ := store.GetRoleRuntimeData(s.playerBase.PlayerID, s.selectedRole.RoleID)
	if result, v := guardAck(t, store, s, r.Playback.ID, now); v != "playback_too_early" || result.battleOver != nil {
		t.Fatal("early settlement")
	}
	unchanged, _, _ := store.GetRoleRuntimeData(s.playerBase.PlayerID, s.selectedRole.RoleID)
	if string(mustJSON(t, before)) != string(mustJSON(t, unchanged)) {
		t.Fatal("early ACK changed persisted role")
	}
	result, v := guardAck(t, store, s, r.Playback.ID, r.Playback.Deadline)
	if v != "" || result.battleOver == nil || s.battleRuntime != nil {
		t.Fatalf("settlement: %s %+v", v, result)
	}
	after, _, _ := store.GetRoleRuntimeData(s.playerBase.PlayerID, s.selectedRole.RoleID)
	duplicate, v := handleRealtimePacket(store, protocol.Packet{Cmd: cmdClassicBattlePlayOverReq, Payload: mustJSON(t, battle.PlayOverRequest{BattleID: r.BattleID, PlaybackID: r.Playback.ID})}, s, r.Playback.Deadline.Add(time.Second))
	final, _, _ := store.GetRoleRuntimeData(s.playerBase.PlayerID, s.selectedRole.RoleID)
	if v != "" || duplicate.battleOver != nil || string(mustJSON(t, after)) != string(mustJSON(t, final)) {
		t.Fatal("duplicate reward")
	}
}

func TestRealtimeGuardFloodAndTransferBypass(t *testing.T) {
	now := time.Now()
	var budget realtimeRequestBudget
	for i := 0; i < 240; i++ {
		if !budget.accept(now) {
			t.Fatal("valid burst rejected")
		}
	}
	if budget.accept(now) {
		t.Fatal("flood allowed")
	}
	if !budget.accept(now.Add(time.Second)) {
		t.Fatal("budget did not replenish")
	}
	if v := validateRealtimeTransfer(mustJSON(t, classicTownTransferRequest{MapID: "6", X: 999999, Y: 600})); v != "transfer_position_spoof" {
		t.Fatal("transfer bypass")
	}
}
