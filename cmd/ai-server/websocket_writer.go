package main

import (
	"context"
	"errors"
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
	outboundStateMu           sync.Mutex
	outboundQueue             chan websocketOutboundMessage
	outboundStop              chan struct{}
	outboundDone              chan struct{}
	outboundClosed            bool
	outboundCloseOnce         sync.Once
	sourceMonsterReplayMu     sync.Mutex
	sourceMonsterReplayCancel context.CancelFunc
	sourceMonsterStateMu      sync.Mutex
	sourceMonsterStates       map[string]classicTownSourceMonsterMoveState
	sourceMonsterLastTargets  map[string]world.SpawnPoint
}

type websocketOutboundMessage struct {
	cmd  uint64
	seq  uint64
	data []byte
}

type classicTownSourceMonsterMoveState struct {
	mapID    string
	position world.SpawnPoint
}

const (
	websocketWriterQueueCapacity               = 4096
	classicTownSourceMonsterReplayInitialDelay = 250 * time.Millisecond
	classicTownSourceMonsterWalkSpeed          = 130.0
	classicTownSourceMonsterRunMultiplier      = 2.0
	classicTownSourceMonsterReplayMinDelay     = 150 * time.Millisecond
	classicTownSourceMonsterReplayMaxDelay     = 8 * time.Second
	classicTownSourceMonsterChaseRadius        = 200.0
)

var (
	errWebsocketWriterClosed       = errors.New("websocket writer closed")
	errWebsocketWriterBackpressure = errors.New("websocket writer outbound queue full")
)

func (writer *websocketWriter) startOutbound() {
	if writer.conn == nil {
		return
	}

	writer.outboundStateMu.Lock()
	if writer.outboundQueue != nil {
		writer.outboundStateMu.Unlock()
		return
	}
	writer.outboundQueue = make(chan websocketOutboundMessage, websocketWriterQueueCapacity)
	writer.outboundStop = make(chan struct{})
	writer.outboundDone = make(chan struct{})
	writer.outboundStateMu.Unlock()

	go writer.runOutbound()
}

func (writer *websocketWriter) stopOutbound() {
	writer.closeOutboundSignal()

	writer.outboundStateMu.Lock()
	done := writer.outboundDone
	writer.outboundStateMu.Unlock()
	if done != nil {
		<-done
	}
}

func (writer *websocketWriter) closeOutboundSignal() {
	writer.outboundCloseOnce.Do(func() {
		writer.outboundStateMu.Lock()
		writer.outboundClosed = true
		stop := writer.outboundStop
		conn := writer.conn
		writer.outboundStateMu.Unlock()

		if stop != nil {
			close(stop)
		}
		if conn != nil {
			_ = conn.Close()
		}
	})
}

func (writer *websocketWriter) runOutbound() {
	writer.outboundStateMu.Lock()
	queue := writer.outboundQueue
	stop := writer.outboundStop
	done := writer.outboundDone
	conn := writer.conn
	writer.outboundStateMu.Unlock()

	defer close(done)
	for {
		select {
		case <-stop:
			return
		case message := <-queue:
			if err := conn.WriteMessage(websocket.BinaryMessage, message.data); err != nil {
				log.Printf("[ai-server] websocket outbound write failed cmd=%d seq=%d: %v", message.cmd, message.seq, err)
				writer.closeOutboundSignal()
				return
			}
		}
	}
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
	return writer.enqueueOutboundLocked(websocketOutboundMessage{cmd: cmd, seq: seq, data: response})
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
	return writer.enqueueOutboundLocked(websocketOutboundMessage{cmd: cmd, seq: seq, data: response})
}

func (writer *websocketWriter) enqueueOutboundLocked(message websocketOutboundMessage) error {
	writer.outboundStateMu.Lock()
	queue := writer.outboundQueue
	closed := writer.outboundClosed
	writer.outboundStateMu.Unlock()
	if writer.conn == nil {
		return nil
	}
	if closed {
		return errWebsocketWriterClosed
	}
	if queue == nil {
		return writer.conn.WriteMessage(websocket.BinaryMessage, message.data)
	}

	select {
	case queue <- message:
		return nil
	default:
		writer.closeOutboundSignal()
		return errWebsocketWriterBackpressure
	}
}

func (writer *websocketWriter) outboundDepth() int {
	writer.outboundStateMu.Lock()
	queue := writer.outboundQueue
	writer.outboundStateMu.Unlock()
	if queue == nil {
		return 0
	}
	return len(queue)
}

func (writer *websocketWriter) writeClassicTownBootstrap(snapshot world.TownBootstrapSnapshot) {
	writer.stopClassicTownSourceMonsterMoveReplay()
	writer.resetClassicTownSourceMonsterState(snapshot)

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
	loop := world.CapturedSourceMonsterMoveReplayLoopsForBootstrap(snapshot)

	ctx, cancel := context.WithCancel(context.Background())
	writer.sourceMonsterReplayMu.Lock()
	writer.sourceMonsterReplayCancel = cancel
	writer.sourceMonsterReplayMu.Unlock()

	go writer.replayClassicTownSourceMonsterMoves(ctx, steps, loop)
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

func (writer *websocketWriter) replayClassicTownSourceMonsterMoves(ctx context.Context, steps []world.RoleMovePush, loop bool) {
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
		go writer.replayClassicTownSourceMonsterHandleMoves(ctx, handleSteps, loop)
	}
}

func (writer *websocketWriter) replayClassicTownSourceMonsterHandleMoves(ctx context.Context, steps []world.RoleMovePush, loop bool) {
	for {
		replayed := false
		for _, step := range steps {
			if !classicTownSourceMonsterReplayableStep(step) {
				continue
			}
			replayed = true
			replayStep := writer.prepareClassicTownSourceMonsterReplayMove(step)
			if err := writer.writePush(cmdClassicTownMoveRolePush, encodePayload(replayStep)); err != nil {
				log.Printf("[ai-server] write classic source monster moveRole replay failed: %v", err)
				return
			}
			if !sleepWithContext(ctx, classicTownSourceMonsterReplayDelay(replayStep)) {
				return
			}
			writer.setClassicTownSourceMonsterPosition(replayStep.Handle, replayStep.MapID, world.SpawnPoint{X: replayStep.TX, Y: replayStep.TY})
		}
		if !loop || !replayed {
			return
		}
	}
}

func classicTownSourceMonsterReplayableStep(step world.RoleMovePush) bool {
	return step.Type != "Run"
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

func (writer *websocketWriter) resetClassicTownSourceMonsterState(snapshot world.TownBootstrapSnapshot) {
	states := make(map[string]classicTownSourceMonsterMoveState)
	for _, role := range snapshot.CreateRoles {
		if role.Kind != "monster" || role.RoleID != "-2" || role.Handle == "" {
			continue
		}
		states[role.Handle] = classicTownSourceMonsterMoveState{
			mapID:    role.MapID,
			position: role.SpawnFlash,
		}
	}

	writer.sourceMonsterStateMu.Lock()
	defer writer.sourceMonsterStateMu.Unlock()
	writer.sourceMonsterStates = states
	writer.sourceMonsterLastTargets = make(map[string]world.SpawnPoint)
}

func (writer *websocketWriter) setClassicTownSourceMonsterPosition(handle string, mapID string, position world.SpawnPoint) {
	if handle == "" {
		return
	}

	writer.sourceMonsterStateMu.Lock()
	defer writer.sourceMonsterStateMu.Unlock()
	if writer.sourceMonsterStates == nil {
		writer.sourceMonsterStates = make(map[string]classicTownSourceMonsterMoveState)
	}
	state := writer.sourceMonsterStates[handle]
	state.mapID = mapID
	state.position = position
	writer.sourceMonsterStates[handle] = state
}

func (writer *websocketWriter) prepareClassicTownSourceMonsterReplayMove(step world.RoleMovePush) world.RoleMovePush {
	if step.Handle == "" {
		return step
	}

	writer.sourceMonsterStateMu.Lock()
	defer writer.sourceMonsterStateMu.Unlock()
	if writer.sourceMonsterStates == nil {
		writer.sourceMonsterStates = make(map[string]classicTownSourceMonsterMoveState)
	}

	state, exists := writer.sourceMonsterStates[step.Handle]
	if exists {
		step.X = state.position.X
		step.Y = state.position.Y
	}
	state.mapID = step.MapID
	state.position = world.SpawnPoint{X: step.X, Y: step.Y}
	writer.sourceMonsterStates[step.Handle] = state
	return step
}

func (writer *websocketWriter) removeClassicTownSourceMonsterStates(handles []string) {
	if len(handles) == 0 {
		return
	}

	writer.sourceMonsterStateMu.Lock()
	defer writer.sourceMonsterStateMu.Unlock()
	for _, handle := range handles {
		delete(writer.sourceMonsterStates, handle)
		delete(writer.sourceMonsterLastTargets, handle)
	}
}

func (writer *websocketWriter) classicTownSourceMonsterChaseMoves(playerMove world.RoleMovePush) []world.RoleMovePush {
	if playerMove.MapID == "" {
		return nil
	}

	playerCurrent := world.SpawnPoint{X: playerMove.X, Y: playerMove.Y}
	playerTarget := world.SpawnPoint{X: playerMove.TX, Y: playerMove.TY}

	writer.sourceMonsterStateMu.Lock()
	defer writer.sourceMonsterStateMu.Unlock()
	if len(writer.sourceMonsterStates) == 0 {
		return nil
	}
	if writer.sourceMonsterLastTargets == nil {
		writer.sourceMonsterLastTargets = make(map[string]world.SpawnPoint)
	}

	moves := make([]world.RoleMovePush, 0)
	for handle, state := range writer.sourceMonsterStates {
		if state.mapID != playerMove.MapID {
			continue
		}
		target, ok := classicTownSourceMonsterChaseTarget(state.position, playerCurrent, playerTarget)
		if !ok {
			continue
		}
		if lastTarget, ok := writer.sourceMonsterLastTargets[handle]; ok && lastTarget == target {
			continue
		}
		move := world.RoleMovePush{
			Handle: handle,
			Type:   "Run",
			X:      state.position.X,
			Y:      state.position.Y,
			Z:      0,
			TX:     target.X,
			TY:     target.Y,
			TZ:     0,
			MapID:  state.mapID,
		}
		moves = append(moves, move)
		state.position = target
		writer.sourceMonsterStates[handle] = state
		writer.sourceMonsterLastTargets[handle] = target
	}
	return moves
}

func classicTownSourceMonsterChaseTarget(source world.SpawnPoint, playerCurrent world.SpawnPoint, playerTarget world.SpawnPoint) (world.SpawnPoint, bool) {
	if isClassicTownSourceMonsterWithinChaseRadius(source, playerTarget) {
		return playerTarget, true
	}
	if isClassicTownSourceMonsterWithinChaseRadius(source, playerCurrent) {
		return playerCurrent, true
	}
	return world.SpawnPoint{}, false
}

func isClassicTownSourceMonsterWithinChaseRadius(source world.SpawnPoint, target world.SpawnPoint) bool {
	dx := float64(target.X - source.X)
	dy := float64(target.Y - source.Y)
	return math.Sqrt(dx*dx+dy*dy) <= classicTownSourceMonsterChaseRadius
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
