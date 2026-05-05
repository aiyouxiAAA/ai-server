package main

import (
	"encoding/json"
	"testing"

	"ai-server/internal/protocol"
	"ai-server/internal/session"
)

func TestHandlePacketLoginSuccess(t *testing.T) {
	store := session.NewStore()
	request := session.LoginRequest{
		UserName: "mockuser",
		Password: "magicpwd",
	}

	responseCmd, payload, ok := handlePacket(store, protocol.Packet{
		Cmd:     cmdAuthLoginRequest,
		Seq:     1,
		Payload: mustJSON(t, request),
	})

	if !ok {
		t.Fatal("expected login packet to be handled")
	}
	if responseCmd != cmdAuthLoginResponse {
		t.Fatalf("expected response cmd %d, got %d", cmdAuthLoginResponse, responseCmd)
	}

	var response session.LoginResponse
	decodeJSON(t, payload, &response)
	if !response.Success {
		t.Fatalf("expected login success, got failure: %+v", response)
	}
	if response.PlayerID != "mock-player-001" {
		t.Fatalf("expected player id mock-player-001, got %q", response.PlayerID)
	}
}

func TestHandlePacketLoginInvalidPayload(t *testing.T) {
	store := session.NewStore()

	_, _, ok := handlePacket(store, protocol.Packet{
		Cmd:     cmdAuthLoginRequest,
		Seq:     1,
		Payload: []byte("{invalid-json"),
	})

	if ok {
		t.Fatal("expected invalid payload to be rejected")
	}
}

func TestHandlePacketRoleFlow(t *testing.T) {
	store := session.NewStore()
	playerID := "mock-player-001"

	createCmd, createPayload, ok := handlePacket(store, protocol.Packet{
		Cmd: cmdRoleCreateRequest,
		Seq: 2,
		Payload: mustJSON(t, session.RoleCreateRequest{
			PlayerID:       playerID,
			DisplayName:    "测试女侠",
			Gender:         "female",
			RoleTemplateID: 1,
		}),
	})
	if !ok {
		t.Fatal("expected create role packet to be handled")
	}
	if createCmd != cmdRoleCreateResponse {
		t.Fatalf("expected response cmd %d, got %d", cmdRoleCreateResponse, createCmd)
	}

	var createResponse session.RoleCreateResponse
	decodeJSON(t, createPayload, &createResponse)
	if createResponse.Role.RoleID == "" {
		t.Fatal("expected created role id to be non-empty")
	}

	listCmd, listPayload, ok := handlePacket(store, protocol.Packet{
		Cmd: cmdRoleListRequest,
		Seq: 3,
		Payload: mustJSON(t, session.RoleListRequest{
			PlayerID: playerID,
		}),
	})
	if !ok {
		t.Fatal("expected list role packet to be handled")
	}
	if listCmd != cmdRoleListResponse {
		t.Fatalf("expected response cmd %d, got %d", cmdRoleListResponse, listCmd)
	}

	var listResponse session.RoleListResponse
	decodeJSON(t, listPayload, &listResponse)
	if len(listResponse.Roles) != 1 {
		t.Fatalf("expected exactly one role, got %d", len(listResponse.Roles))
	}

	selectCmd, selectPayload, ok := handlePacket(store, protocol.Packet{
		Cmd: cmdRoleSelectRequest,
		Seq: 4,
		Payload: mustJSON(t, session.RoleSelectRequest{
			PlayerID: playerID,
			RoleID:   createResponse.Role.RoleID,
		}),
	})
	if !ok {
		t.Fatal("expected select role packet to be handled")
	}
	if selectCmd != cmdRoleSelectResponse {
		t.Fatalf("expected response cmd %d, got %d", cmdRoleSelectResponse, selectCmd)
	}

	var selectResponse session.RoleSelectResponse
	decodeJSON(t, selectPayload, &selectResponse)
	if selectResponse.Role.RoleID != createResponse.Role.RoleID {
		t.Fatalf("expected selected role id %q, got %q", createResponse.Role.RoleID, selectResponse.Role.RoleID)
	}
	if selectResponse.PlayerBase.PlayerID != playerID {
		t.Fatalf("expected player base player id %q, got %q", playerID, selectResponse.PlayerBase.PlayerID)
	}

	removeCmd, removePayload, ok := handlePacket(store, protocol.Packet{
		Cmd: cmdRoleRemoveRequest,
		Seq: 5,
		Payload: mustJSON(t, session.RoleRemoveRequest{
			PlayerID: playerID,
			RoleID:   createResponse.Role.RoleID,
			Password: "magicpwd",
		}),
	})
	if !ok {
		t.Fatal("expected remove role packet to be handled")
	}
	if removeCmd != cmdRoleRemoveResponse {
		t.Fatalf("expected response cmd %d, got %d", cmdRoleRemoveResponse, removeCmd)
	}

	var removeResponse session.RoleRemoveResponse
	decodeJSON(t, removePayload, &removeResponse)
	if !removeResponse.Success {
		t.Fatalf("expected remove role success, got failure: %+v", removeResponse)
	}
	if removeResponse.RemovedRoleID != createResponse.Role.RoleID {
		t.Fatalf("expected removed role id %q, got %q", createResponse.Role.RoleID, removeResponse.RemovedRoleID)
	}

	listCmdAfterDelete, listPayloadAfterDelete, ok := handlePacket(store, protocol.Packet{
		Cmd: cmdRoleListRequest,
		Seq: 6,
		Payload: mustJSON(t, session.RoleListRequest{
			PlayerID: playerID,
		}),
	})
	if !ok {
		t.Fatal("expected list role packet after delete to be handled")
	}
	if listCmdAfterDelete != cmdRoleListResponse {
		t.Fatalf("expected response cmd %d, got %d", cmdRoleListResponse, listCmdAfterDelete)
	}

	var listResponseAfterDelete session.RoleListResponse
	decodeJSON(t, listPayloadAfterDelete, &listResponseAfterDelete)
	if len(listResponseAfterDelete.Roles) != 0 {
		t.Fatalf("expected no roles after delete, got %d", len(listResponseAfterDelete.Roles))
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()

	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("failed to marshal json: %v", err)
	}
	return payload
}

func decodeJSON(t *testing.T, payload []byte, target any) {
	t.Helper()

	if err := json.Unmarshal(payload, target); err != nil {
		t.Fatalf("failed to unmarshal json: %v", err)
	}
}
