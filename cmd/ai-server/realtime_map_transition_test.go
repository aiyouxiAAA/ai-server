package main

import (
	"ai-server/internal/battle"
	"ai-server/internal/protocol"
	"ai-server/internal/session"
	"ai-server/internal/world"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func TestRealtimeGuardMapTransitionLateRequests(t *testing.T) {
	for _, cmd := range []uint64{cmdClassicBattleStartReq, cmdClassicTownCrossRoleReq, cmdClassicTownMoveRoleReq} {
		for _, sameMapBootstrap := range []bool{false, true} {
			t.Run(strconv.Itoa(int(cmd))+"/same-map="+strconv.FormatBool(sameMapBootstrap), func(t *testing.T) {
				store := session.NewStore()
				now := time.Now()
				s := &packetSession{selectedRole: &session.RoleSummary{RoleID: "moving"}, playerBase: &session.PlayerBaseData{MapID: 2}}
				s.movement.reset("moving", 6, world.SpawnPoint{X: 1000, Y: 600}, now)
				s.movement.reset("moving", 2, world.SpawnPoint{X: 2915, Y: 500}, now)
				if sameMapBootstrap {
					s.movement.reset("moving", 2, world.SpawnPoint{X: 2915, Y: 500}, now)
				}
				packet := protocol.Packet{Cmd: cmd, Payload: mustJSON(t, map[string]interface{}{
					"mapId": "6", "handle": "moving", "type": "Run", "x": 1000, "y": 600, "tx": 1000, "ty": 600,
				})}
				beforePosition, beforeCredit := s.movement.position, s.movement.credit
				result, violation := handleRealtimePacket(store, packet, s, now)
				if violation != "" || !reflect.DeepEqual(result, packetResult{handled: true}) {
					t.Fatalf("late request must be ignored without domain output: violation=%s result=%+v", violation, result)
				}
				if s.battleRuntime != nil || s.playerBase.MapID != 2 || s.movement.position != beforePosition || s.movement.credit != beforeCredit {
					t.Fatal("late request mutated current map/battle/movement")
				}
			})
		}
	}
}

func TestRealtimeGuardMapTransitionStillRejectsUnknownMaps(t *testing.T) {
	for _, cmd := range []uint64{cmdClassicBattleStartReq, cmdClassicTownCrossRoleReq, cmdClassicTownMoveRoleReq} {
		for _, scenario := range []string{"unvisited", "two-maps-ago", "other-role", "no-history", "current-map-mismatch"} {
			t.Run(strconv.Itoa(int(cmd))+"/"+scenario, func(t *testing.T) {
				now := time.Now()
				s := &packetSession{selectedRole: &session.RoleSummary{RoleID: "moving"}, playerBase: &session.PlayerBaseData{MapID: 2}}
				s.movement.reset("moving", 6, world.SpawnPoint{}, now)
				s.movement.reset("moving", 2, world.SpawnPoint{}, now)
				mapID := "6"
				switch scenario {
				case "unvisited":
					mapID = "999999"
				case "two-maps-ago":
					s.movement.reset("moving", 4, world.SpawnPoint{}, now)
					s.playerBase.MapID = 4
				case "other-role":
					s.selectedRole.RoleID = "new-role"
					s.movement.reset("new-role", 2, world.SpawnPoint{}, now)
				case "no-history":
					s.movement = realtimeMovementGuard{}
				case "current-map-mismatch":
					s.playerBase.MapID = 4
				}
				packet := protocol.Packet{Cmd: cmd, Payload: mustJSON(t, map[string]interface{}{"mapId": mapID, "type": "Run"})}
				result, violation := handleRealtimePacket(session.NewStore(), packet, s, now)
				if violation == "" || !reflect.DeepEqual(result, packetResult{handled: true}) {
					t.Fatalf("unknown map must still be rejected: %s %+v", violation, result)
				}
			})
		}
	}
}

func TestRealtimeGuardGuifengWithoutMPKeepsCommandWindow(t *testing.T) {
	store, sockets := guardBattleFixture(t, 1)
	s := sockets[0]
	r := s.battleRuntime
	command, _ := r.CommandWindowForActor(s.selectedRole.RoleID)
	commandID := ""
	for _, entry := range command.Commands {
		if entry.SourceActionLabel == "blade/guifeng" {
			commandID = entry.ID
		}
	}
	if commandID == "" {
		t.Fatal("missing guifeng command")
	}
	for i := range r.Cells {
		if r.Cells[i].Handle == s.selectedRole.RoleID {
			r.Cells[i].MP = 0
		}
	}
	packet := protocol.Packet{Cmd: cmdClassicBattleActionReq, Payload: mustJSON(t, battle.ActionRequest{
		BattleID: r.BattleID, ActorHandle: s.selectedRole.RoleID, CommandID: commandID,
		TargetHandle: "all", Round: command.Round, Sequence: command.Sequence,
	})}
	for i := 0; i < 3; i++ {
		result, violation := handleRealtimePacket(store, packet, s, time.Now())
		if violation != "" || result.battleCommand == nil || result.battleCommand.Sequence != command.Sequence || len(result.battleActions) != 0 || r.ConsumedSequence[command.Sequence] || r.Playback.ID != 0 {
			t.Fatalf("insufficient MP must retry without kicking or consuming turn: %s %+v", violation, result)
		}
	}
	guardAction(t, store, s, time.Now())
}
