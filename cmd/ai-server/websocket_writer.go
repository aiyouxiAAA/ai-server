package main

import (
	"log"
	"sync"
	"time"

	"ai-server/internal/protocol"
	"ai-server/internal/world"
	"github.com/gorilla/websocket"
)

type websocketWriter struct {
	conn          *websocket.Conn
	mu            sync.Mutex
	nextServerSeq uint64
}

func (writer *websocketWriter) writePacket(cmd uint64, seq uint64, payload []byte) error {
	response := protocol.Encode(protocol.Packet{
		Cmd:         cmd,
		Seq:         seq,
		Payload:     payload,
		TimestampMs: uint64(time.Now().UnixMilli()),
	})

	writer.mu.Lock()
	defer writer.mu.Unlock()
	return writer.conn.WriteMessage(websocket.BinaryMessage, response)
}

func (writer *websocketWriter) writePush(cmd uint64, payload []byte) error {
	seq := writer.nextServerSeq
	writer.nextServerSeq += 1
	return writer.writePacket(cmd, seq, payload)
}

func (writer *websocketWriter) writeClassicTownBootstrap(snapshot world.TownBootstrapSnapshot) {
	if err := writer.writePush(cmdClassicTownLoadMapPush, encodePayload(snapshot.LoadMap)); err != nil {
		log.Printf("[ai-server] write classic town loadMap push failed: %v", err)
		return
	}
	if err := writer.writePush(cmdClassicTownCreatePlayerPush, encodePayload(snapshot.CreatePlayer)); err != nil {
		log.Printf("[ai-server] write classic town createPlayer push failed: %v", err)
		return
	}
	if snapshot.RolePhysique != nil {
		if err := writer.writePush(cmdClassicTownRolePhysiquePush, encodePayload(*snapshot.RolePhysique)); err != nil {
			log.Printf("[ai-server] write classic town rolePhysique push failed: %v", err)
			return
		}
	}
	if snapshot.RoleState != nil {
		if err := writer.writePush(cmdClassicTownRoleStatePush, encodePayload(*snapshot.RoleState)); err != nil {
			log.Printf("[ai-server] write classic town roleState push failed: %v", err)
			return
		}
	}
	for _, role := range snapshot.CreateRoles {
		if err := writer.writePush(cmdClassicTownCreateRolePush, encodePayload(role)); err != nil {
			log.Printf("[ai-server] write classic town createRole push failed: %v", err)
			return
		}
	}
	for _, questState := range snapshot.QuestStates {
		if err := writer.writePush(cmdClassicTownQuestStatePush, encodePayload(questState)); err != nil {
			log.Printf("[ai-server] write classic town questState push failed: %v", err)
			return
		}
	}
}
