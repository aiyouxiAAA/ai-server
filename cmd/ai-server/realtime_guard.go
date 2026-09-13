package main

import (
	"encoding/json"
	"log"
	"strconv"
	"time"

	"ai-server/internal/battle"
	"ai-server/internal/protocol"
	"ai-server/internal/session"
)

// All production WebSocket gameplay enters here, before any domain mutation.
// Existing domain handlers remain independently testable without a network clock.
func handleRealtimePacket(store *session.Store, packet protocol.Packet, socket *packetSession, now time.Time) (packetResult, string) {
	if !socket.requestBudget.accept(now) {
		return packetResult{handled: true}, "request_flood"
	}
	return handleRealtimeGameplay(store, packet, socket, now)
}

func handleRealtimeGameplay(store *session.Store, packet protocol.Packet, socket *packetSession, now time.Time) (packetResult, string) {
	runtime := socket.battleRuntime
	if runtime != nil {
		runtime.Playback.Wire.Lock()
		defer runtime.Playback.Wire.Unlock()
	}
	if packet.Cmd == cmdClassicTownMoveRoleReq {
		skip, violation := validateRealtimeMovement(socket, packet.Payload, now)
		if violation != "" || skip {
			return packetResult{handled: true}, violation
		}
	}
	switch packet.Cmd {
	case cmdClassicTownTransferReq:
		if violation := validateRealtimeTransfer(packet.Payload); violation != "" {
			return packetResult{handled: true}, violation
		}
	case cmdClassicTownCrossRoleReq:
		var request classicTownRoleInteractionRequest
		if json.Unmarshal(packet.Payload, &request) != nil {
			return packetResult{handled: true}, "malformed_cross_role"
		}
		if socket.playerBase == nil || request.MapID != strconv.Itoa(socket.playerBase.MapID) {
			return packetResult{handled: true}, "cross_role_map_spoof"
		}
	case cmdClassicBattleStartReq:
		var request battle.StartRequest
		if json.Unmarshal(packet.Payload, &request) != nil {
			return packetResult{handled: true}, "malformed_battle_start"
		}
		if socket.playerBase == nil || socket.selectedRole == nil {
			return packetResult{handled: true}, "unauthenticated_battle"
		}
		if request.MapID != strconv.Itoa(socket.playerBase.MapID) {
			return packetResult{handled: true}, "battle_map_spoof"
		}
	case cmdClassicBattleActionReq, cmdClassicBattleActiveItemReq:
		var request battle.ActionRequest
		if json.Unmarshal(packet.Payload, &request) != nil {
			return packetResult{handled: true}, "malformed_battle_action"
		}
		if socket.selectedRole == nil || request.ActorHandle != socket.selectedRole.RoleID {
			return packetResult{handled: true}, "battle_actor_spoof"
		}
		// A late action from the previous battle/window cannot mutate the current one.
		if runtime == nil || request.BattleID != runtime.BattleID {
			return packetResult{handled: true}, ""
		}
		if command, ok := runtime.CommandWindowForActor(request.ActorHandle); ok {
			if request.Round > command.Round || request.Sequence > command.Sequence {
				return packetResult{handled: true}, "battle_sequence_spoof"
			}
			if request.Round != command.Round || request.Sequence != command.Sequence {
				return packetResult{handled: true}, ""
			}
			if packet.Cmd == cmdClassicBattleActionReq {
				allowed := false
				for _, entry := range command.Commands {
					if entry.ID == request.CommandID {
						allowed = true
						break
					}
				}
				if !allowed {
					return packetResult{handled: true}, "battle_command_spoof"
				}
			}
		}
		if err := runtime.ValidatePlaybackCatalog(); err != nil {
			return playbackConfigurationError(err), ""
		}
	case cmdClassicBattlePlayOverReq:
		var request battle.PlayOverRequest
		if json.Unmarshal(packet.Payload, &request) != nil {
			return packetResult{handled: true}, "malformed_playback_ack"
		}
		if runtime == nil || request.BattleID != runtime.BattleID {
			return packetResult{handled: true}, ""
		}
		clock := &runtime.Playback
		if request.PlaybackID <= 0 {
			return packetResult{handled: true}, "playback_batch_spoof"
		}
		if request.PlaybackID <= clock.Consumed || (request.PlaybackID > 0 && request.PlaybackID < clock.ID) {
			return packetResult{handled: true}, ""
		}
		if request.PlaybackID <= 0 || request.PlaybackID != clock.ID {
			return packetResult{handled: true}, "playback_batch_spoof"
		}
		if now.Before(clock.Deadline) {
			if clock.Deadline.Sub(now) <= playbackEarlyGrace {
				return packetResult{handled: true, realtimeResumeAt: clock.Deadline}, ""
			}
			log.Printf("[ai-server] early playback battleId=%s batch=%d earlyByMs=%.3f", runtime.BattleID, request.PlaybackID, clock.Deadline.Sub(now).Seconds()*1000)
			return packetResult{handled: true}, "playback_too_early"
		}
		clock.Consumed = request.PlaybackID
	}
	result := handlePacketWithSession(store, packet, socket)
	if runtime != nil && len(result.battleActions) > 0 {
		if err := runtime.ProtectActions(result.battleActions, now); err != nil {
			return playbackConfigurationError(err), ""
		}
	}
	if result.townBootstrap != nil && socket.selectedRole != nil {
		socket.movement.reset(socket.selectedRole.RoleID, socket.playerBase.MapID, result.townBootstrap.CreatePlayer.SpawnFlash, now)
	}
	return result, ""
}

func playbackConfigurationError(err error) packetResult {
	log.Printf("[ai-server] authoritative playback configuration error: %v", err)
	return packetResult{handled: true, errorMessages: []classicTownErrorPush{{Msg: "战斗时序配置异常，请联系管理员。"}}}
}
