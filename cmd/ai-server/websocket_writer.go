package main

import (
	"context"
	"log"
	"math"
	"sync"
	"time"

	"ai-server/internal/protocol"
	"ai-server/internal/world"
	"github.com/gorilla/websocket"
)

type websocketWriter struct {
	conn                      *websocket.Conn
	mu                        sync.Mutex
	nextServerSeq             uint64
	sourceMonsterReplayMu     sync.Mutex
	sourceMonsterReplayCancel context.CancelFunc
}

const (
	classicTownSourceMonsterReplayInitialDelay = 250 * time.Millisecond
	classicTownSourceMonsterWalkSpeed          = 130.0
	classicTownSourceMonsterRunMultiplier      = 2.0
	classicTownSourceMonsterReplayMinDelay     = 150 * time.Millisecond
	classicTownSourceMonsterReplayMaxDelay     = 8 * time.Second
)

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
	writer.mu.Lock()
	defer writer.mu.Unlock()

	seq := writer.nextServerSeq
	writer.nextServerSeq += 1
	response := protocol.Encode(protocol.Packet{
		Cmd:         cmd,
		Seq:         seq,
		Payload:     payload,
		TimestampMs: uint64(time.Now().UnixMilli()),
	})
	if writer.conn == nil {
		return nil
	}
	return writer.conn.WriteMessage(websocket.BinaryMessage, response)
}

func (writer *websocketWriter) writeClassicTownBootstrap(snapshot world.TownBootstrapSnapshot) {
	writer.stopClassicTownSourceMonsterMoveReplay()

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
	writer.startClassicTownSourceMonsterMoveReplay(snapshot)
}

func (writer *websocketWriter) startClassicTownSourceMonsterMoveReplay(snapshot world.TownBootstrapSnapshot) {
	if writer.conn == nil {
		return
	}

	steps := world.CapturedSourceMonsterMoveReplayForBootstrap(snapshot)
	if len(steps) == 0 {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	writer.sourceMonsterReplayMu.Lock()
	writer.sourceMonsterReplayCancel = cancel
	writer.sourceMonsterReplayMu.Unlock()

	go writer.replayClassicTownSourceMonsterMoves(ctx, steps)
}

func (writer *websocketWriter) stopClassicTownSourceMonsterMoveReplay() {
	writer.sourceMonsterReplayMu.Lock()
	cancel := writer.sourceMonsterReplayCancel
	writer.sourceMonsterReplayCancel = nil
	writer.sourceMonsterReplayMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (writer *websocketWriter) replayClassicTownSourceMonsterMoves(ctx context.Context, steps []world.RoleMovePush) {
	if !sleepWithContext(ctx, classicTownSourceMonsterReplayInitialDelay) {
		return
	}

	stepsByHandle := make(map[string][]world.RoleMovePush)
	handleOrder := make([]string, 0)
	for _, step := range steps {
		if _, ok := stepsByHandle[step.Handle]; !ok {
			handleOrder = append(handleOrder, step.Handle)
		}
		stepsByHandle[step.Handle] = append(stepsByHandle[step.Handle], step)
	}

	for _, handle := range handleOrder {
		handleSteps := append([]world.RoleMovePush{}, stepsByHandle[handle]...)
		go writer.replayClassicTownSourceMonsterHandleMoves(ctx, handleSteps)
	}
}

func (writer *websocketWriter) replayClassicTownSourceMonsterHandleMoves(ctx context.Context, steps []world.RoleMovePush) {
	for _, step := range steps {
		if err := writer.writePush(cmdClassicTownMoveRolePush, encodePayload(step)); err != nil {
			log.Printf("[ai-server] write classic source monster moveRole replay failed: %v", err)
			return
		}
		if !sleepWithContext(ctx, classicTownSourceMonsterReplayDelay(step)) {
			return
		}
	}
}

func classicTownSourceMonsterReplayDelay(step world.RoleMovePush) time.Duration {
	if step.Type == "Flash" {
		return classicTownSourceMonsterReplayMinDelay
	}

	speed := classicTownSourceMonsterWalkSpeed
	if step.Type == "Run" {
		speed *= classicTownSourceMonsterRunMultiplier
	}
	if speed <= 0 {
		return classicTownSourceMonsterReplayMinDelay
	}

	dx := float64(step.TX - step.X)
	dy := float64(step.TY - step.Y)
	distance := math.Sqrt(dx*dx + dy*dy)
	delay := time.Duration(distance/speed*float64(time.Second)) + classicTownSourceMonsterReplayMinDelay
	if delay < classicTownSourceMonsterReplayMinDelay {
		return classicTownSourceMonsterReplayMinDelay
	}
	if delay > classicTownSourceMonsterReplayMaxDelay {
		return classicTownSourceMonsterReplayMaxDelay
	}
	return delay
}

func sleepWithContext(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
