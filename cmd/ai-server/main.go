package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"ai-server/internal/protocol"
	"ai-server/internal/session"
	"github.com/gorilla/websocket"
)

const (
	cmdAuthLoginRequest   = 1001
	cmdAuthLoginResponse  = 1002
	cmdRoleListRequest    = 1011
	cmdRoleListResponse   = 1012
	cmdRoleCreateRequest  = 1013
	cmdRoleCreateResponse = 1014
	cmdRoleSelectRequest  = 1015
	cmdRoleSelectResponse = 1016
	cmdRoleRemoveRequest  = 1017
	cmdRoleRemoveResponse = 1018
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(_ *http.Request) bool {
		return true
	},
}

func main() {
	store := session.NewStore()

	apiMux := http.NewServeMux()
	wsMux := http.NewServeMux()

	healthHandler := func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"ok":true}`))
	}

	apiMux.HandleFunc("/healthz", healthHandler)
	wsMux.HandleFunc("/healthz", healthHandler)
	wsMux.HandleFunc("/ws", func(writer http.ResponseWriter, request *http.Request) {
		handleWebSocket(store, writer, request)
	})

	go func() {
		log.Println("[ai-server] http api listening on http://127.0.0.1:18080")
		if err := http.ListenAndServe("127.0.0.1:18080", apiMux); err != nil {
			log.Fatalf("[ai-server] http api failed: %v", err)
		}
	}()

	log.Println("[ai-server] websocket listening on ws://127.0.0.1:18443/ws")
	log.Fatal(http.ListenAndServe("127.0.0.1:18443", wsMux))
}

func handleWebSocket(store *session.Store, writer http.ResponseWriter, request *http.Request) {
	conn, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		log.Printf("[ai-server] websocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[ai-server] websocket closed: %v", err)
			return
		}
		if messageType != websocket.BinaryMessage {
			continue
		}

		packet, err := protocol.Decode(data)
		if err != nil {
			log.Printf("[ai-server] decode packet failed: %v", err)
			continue
		}

		responseCmd, responsePayload, ok := handlePacket(store, packet)
		if !ok {
			log.Printf("[ai-server] unsupported command: %d", packet.Cmd)
			continue
		}

		response := protocol.Encode(protocol.Packet{
			Cmd:         responseCmd,
			Seq:         packet.Seq,
			Payload:     responsePayload,
			TimestampMs: uint64(time.Now().UnixMilli()),
		})
		if err := conn.WriteMessage(websocket.BinaryMessage, response); err != nil {
			log.Printf("[ai-server] write response failed: %v", err)
			return
		}
	}
}

func handlePacket(store *session.Store, packet protocol.Packet) (uint64, []byte, bool) {
	switch packet.Cmd {
	case cmdAuthLoginRequest:
		var request session.LoginRequest
		if !decodePayload(packet.Payload, &request) {
			return 0, nil, false
		}
		return cmdAuthLoginResponse, encodePayload(store.Login(request)), true
	case cmdRoleListRequest:
		var request session.RoleListRequest
		if !decodePayload(packet.Payload, &request) {
			return 0, nil, false
		}
		return cmdRoleListResponse, encodePayload(store.ListRoles(request.PlayerID)), true
	case cmdRoleCreateRequest:
		var request session.RoleCreateRequest
		if !decodePayload(packet.Payload, &request) {
			return 0, nil, false
		}
		return cmdRoleCreateResponse, encodePayload(store.CreateRole(request)), true
	case cmdRoleSelectRequest:
		var request session.RoleSelectRequest
		if !decodePayload(packet.Payload, &request) {
			return 0, nil, false
		}
		response, ok := store.SelectRole(request.PlayerID, request.RoleID)
		if !ok {
			return 0, nil, false
		}
		return cmdRoleSelectResponse, encodePayload(response), true
	case cmdRoleRemoveRequest:
		var request session.RoleRemoveRequest
		if !decodePayload(packet.Payload, &request) {
			return 0, nil, false
		}
		return cmdRoleRemoveResponse, encodePayload(store.RemoveRole(request)), true
	default:
		return 0, nil, false
	}
}

func decodePayload(payload []byte, target any) bool {
	if err := json.Unmarshal(payload, target); err != nil {
		log.Printf("[ai-server] decode json payload failed: %v", err)
		return false
	}
	return true
}

func encodePayload(value any) []byte {
	payload, err := json.Marshal(value)
	if err != nil {
		log.Printf("[ai-server] encode json payload failed: %v", err)
		return []byte(`{}`)
	}
	return payload
}
