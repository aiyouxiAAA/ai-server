package main

import (
	"ai-server/internal/battle"
	"ai-server/internal/protocol"
	"ai-server/internal/session"
	"ai-server/internal/world"
	"fmt"
	"math"
	"testing"
	"time"
)

func guardBattleFixture(t *testing.T, count int) (*session.Store, []*packetSession) {
	t.Helper()
	store := session.NewStore()
	sockets := []*packetSession{}
	actors := []battle.TeamActor{}
	for i := 0; i < count; i++ {
		login := store.Login(session.LoginRequest{UserName: fmt.Sprintf("guarduser%d", i), Password: "guardpassword"})
		if !login.Success {
			t.Fatalf("login: %+v", login)
		}
		created := store.CreateRole(session.RoleCreateRequest{PlayerID: login.PlayerID, SessionToken: login.SessionToken, DisplayName: fmt.Sprintf("验时%d", i), Gender: "female", RoleTemplateID: 1})
		if !created.Success {
			t.Fatalf("create: %+v", created)
		}
		role, base, ok := store.UpdateRoleMap(login.PlayerID, created.Role.RoleID, 4)
		if !ok {
			t.Fatal("map")
		}
		base.HP, base.MaxHP = 100000, 100000
		if base.RolePhysique != nil {
			base.RolePhysique.MaxHP = 100000
		}
		socket := &packetSession{selectedRole: &role, playerBase: &base}
		sockets = append(sockets, socket)
		actors = append(actors, battle.TeamActor{Role: role, PlayerBase: base})
	}
	runtime, _, ok := battle.NewTeamWildBattle(actors, battle.StartRequest{MapID: "4", MapName: "test"})
	if !ok {
		t.Fatal("battle")
	}
	for i := range runtime.Cells {
		runtime.Cells[i].HP = 100000
		runtime.Cells[i].MaxHP = 100000
	}
	for _, s := range sockets {
		s.battleRuntime = runtime
	}
	return store, sockets
}
func guardAction(t *testing.T, store *session.Store, s *packetSession, now time.Time) packetResult {
	t.Helper()
	r := s.battleRuntime
	command, ok := r.CommandWindowForActor(s.selectedRole.RoleID)
	if !ok {
		t.Fatal("no command")
	}
	target := ""
	for _, c := range r.Cells {
		if c.Camp == battle.CampEnemy {
			target = c.Handle
			break
		}
	}
	result, violation := handleRealtimePacket(store, protocol.Packet{Cmd: cmdClassicBattleActionReq, Payload: mustJSON(t, battle.ActionRequest{BattleID: r.BattleID, ActorHandle: s.selectedRole.RoleID, CommandID: battle.CommandNormalAttack, TargetHandle: target, Round: command.Round, Sequence: command.Sequence})}, s, now)
	if violation != "" || len(result.battleActions) == 0 {
		t.Fatalf("action rejected: %s %+v", violation, result)
	}
	return result
}
func guardAck(t *testing.T, store *session.Store, s *packetSession, batch int, now time.Time) (packetResult, string) {
	t.Helper()
	return handleRealtimePacket(store, protocol.Packet{Cmd: cmdClassicBattlePlayOverReq, Payload: mustJSON(t, battle.PlayOverRequest{BattleID: s.battleRuntime.BattleID, PlaybackID: batch})}, s, now)
}
func TestRealtimeGuardPlaybackEarlyAndDuplicate(t *testing.T) {
	store, sockets := guardBattleFixture(t, 1)
	s := sockets[0]
	now := time.Now()
	r := s.battleRuntime
	actions := guardAction(t, store, s, now)
	if actions.battleActions[0].PlaybackID != 1 || actions.battleActions[0].PlaybackRemainingMS <= 0 {
		t.Fatal("missing timing")
	}
	starts := len(r.PendingStarts)
	phase := r.Phase
	for _, at := range []time.Time{now, r.Playback.Deadline.Add(-playbackEarlyGrace - time.Nanosecond)} {
		result, violation := guardAck(t, store, s, 1, at)
		if violation != "playback_too_early" || result.battleOver != nil || len(r.PendingStarts) != starts || r.Phase != phase || r.Playback.Consumed != 0 {
			t.Fatalf("early ACK mutated state: %s", violation)
		}
	}
	for _, early := range []time.Duration{playbackEarlyGrace, time.Millisecond, time.Nanosecond} {
		result, violation := guardAck(t, store, s, 1, r.Playback.Deadline.Add(-early))
		if violation != "" || !result.realtimeResumeAt.Equal(r.Playback.Deadline) || result.battleOver != nil || len(result.battleCommands) != 0 || r.Playback.Consumed != 0 {
			t.Fatal("grace must defer without kicking or advancing gameplay")
		}
	}
	for _, id := range []int{0, -1, 2, 10000} {
		_, violation := guardAck(t, store, s, id, r.Playback.Deadline)
		if violation != "playback_batch_spoof" {
			t.Fatalf("forged id %d: %s", id, violation)
		}
	}
	result, violation := guardAck(t, store, s, 1, r.Playback.Deadline)
	if violation != "" || len(result.battleCommands) == 0 {
		t.Fatalf("deadline rejected: %s %+v", violation, result)
	}
	result, violation = guardAck(t, store, s, 1, r.Playback.Deadline.Add(time.Second))
	if violation != "" || result.battleOver != nil || len(result.battleCommands) != 0 {
		t.Fatal("duplicate replayed mutation")
	}
}
func TestRealtimeGuardTeamBatchesShareCumulativeTime(t *testing.T) {
	store, s := guardBattleFixture(t, 2)
	now := time.Now()
	r := s[0].battleRuntime
	guardAction(t, store, s[0], now)
	first := r.Playback.Deadline
	guardAction(t, store, s[1], now)
	if r.Playback.ID != 2 || !r.Playback.Deadline.After(first) {
		t.Fatal("team actions reused time")
	}
	if _, v := guardAck(t, store, s[0], 1, now); v != "" {
		t.Fatal("old teammate ACK misclassified")
	}
	if _, v := guardAck(t, store, s[1], 2, first); v != "playback_too_early" {
		t.Fatal("second batch advanced early")
	}
	result, v := guardAck(t, store, s[1], 2, r.Playback.Deadline)
	if v != "" || len(result.battleCommands) != 2 {
		t.Fatalf("team deadline: %s %+v", v, result)
	}
}
func TestRealtimeGuardRejectsSpoofBeforeDamage(t *testing.T) {
	store, s := guardBattleFixture(t, 1)
	r := s[0].battleRuntime
	command, _ := r.CommandWindowForActor(s[0].selectedRole.RoleID)
	for _, mutate := range []func(*battle.ActionRequest){func(q *battle.ActionRequest) { q.ActorHandle = "someone-else" }, func(q *battle.ActionRequest) { q.Sequence += 999 }, func(q *battle.ActionRequest) { q.CommandID = "skill-forged" }} {
		q := battle.ActionRequest{BattleID: r.BattleID, ActorHandle: s[0].selectedRole.RoleID, CommandID: battle.CommandNormalAttack, Round: command.Round, Sequence: command.Sequence}
		mutate(&q)
		_, v := handleRealtimePacket(store, protocol.Packet{Cmd: cmdClassicBattleActionReq, Payload: mustJSON(t, q)}, s[0], time.Now())
		if v == "" || r.ConsumedSequence[command.Sequence] || r.Playback.ID != 0 {
			t.Fatal("spoof mutated action")
		}
	}
}
func TestRealtimeGuardMovementRatesAndJitter(t *testing.T) {
	for _, delivery := range []struct{ rate, group int }{{4, 1}, {20, 1}, {20, 5}} {
		for _, speed := range []float64{1, 1.5, 2, 5} {
			t.Run(fmt.Sprintf("%gx/%dHz/group%d", speed, delivery.rate, delivery.group), func(t *testing.T) {
				now := time.Now()
				s := &packetSession{selectedRole: &session.RoleSummary{RoleID: "move"}, playerBase: &session.PlayerBaseData{MapID: 4}}
				s.movement.reset("move", 4, world.SpawnPoint{}, now)
				rejected := false
				for i := 1; i <= delivery.rate*30; i++ {
					// Cover the client's usual 4Hz movement sync, frequent reports, and grouped delivery.
					elapsed := float64((i+delivery.group-1)/delivery.group*delivery.group) / float64(delivery.rate)
					x := int(math.Round(float64(i) * 312 * speed / float64(delivery.rate)))
					q := classicTownMoveRoleRequest{Handle: "move", Type: "Run", MapID: "4", X: x, TX: x}
					before := s.movement.position
					beforeCredit, beforeTime := s.movement.credit, s.movement.last
					_, v := validateRealtimeMovement(s, mustJSON(t, q), now.Add(time.Duration(elapsed*float64(time.Second))))
					if v != "" {
						if v != "movement_speed_exceeded" || s.movement.position != before || s.movement.credit != beforeCredit || s.movement.last != beforeTime {
							t.Fatal("invalid move committed")
						}
						if speed == 1.5 && elapsed > 3.25 {
							t.Fatalf("1.5x sustained speed exceeded the expected bounded detection time: %.3fs", elapsed)
						}
						t.Logf("speed=%gx rejectedAt=%.3fs lastAcceptedX=%d rejectedX=%d reason=%s", speed, elapsed, before.X, x, v)
						rejected = true
						break
					}
				}
				if rejected != (speed > 1) {
					t.Fatalf("speed %.1f rejected=%v", speed, rejected)
				}
			})
		}
	}
}
func TestRealtimeGuardMovementVerticalIdleAndTransfer(t *testing.T) {
	now := time.Now()
	s := &packetSession{selectedRole: &session.RoleSummary{RoleID: "move"}, playerBase: &session.PlayerBaseData{MapID: 4}}
	s.movement.reset("move", 4, world.SpawnPoint{}, now)
	q := classicTownMoveRoleRequest{Type: "Run", MapID: "4", Y: 10000, TY: 10000}
	if _, v := validateRealtimeMovement(s, mustJSON(t, q), now.Add(time.Hour)); v != "movement_speed_exceeded" {
		t.Fatal("idle accumulated unlimited teleport credit")
	}
	q.Y, q.TY = 300, 300
	if _, v := validateRealtimeMovement(s, mustJSON(t, q), now); v != "movement_speed_exceeded" {
		t.Fatal("vertical speed bypass")
	}
	s.playerBase.MapID = 6
	s.movement.reset("move", 6, world.SpawnPoint{X: 1000, Y: 600}, now)
	if skip, v := validateRealtimeMovement(s, mustJSON(t, q), now); !skip || v != "" {
		t.Fatal("old map packet kicked")
	}
	q.MapID = "6"
	q.X, q.TX, q.Y, q.TY = 1000, 1000, 600, 600
	if skip, v := validateRealtimeMovement(s, mustJSON(t, q), now); skip || v != "" {
		t.Fatalf("legitimate transfer: %s", v)
	}
}
