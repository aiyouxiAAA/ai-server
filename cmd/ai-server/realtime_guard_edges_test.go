package main

import (
	"ai-server/internal/battle"
	"ai-server/internal/protocol"
	"ai-server/internal/session"
	"sync"
	"testing"
	"time"
)

func TestRealtimeGuardGraceWaitsForRealDeadline(t *testing.T) {
	store, sockets := guardBattleFixture(t, 1)
	s := sockets[0]
	r := s.battleRuntime
	guardAction(t, store, s, time.Now())
	// Exercise the actual admission timer with a short fixture deadline, without sleeping/polling.
	r.Playback.Deadline = time.Now().Add(25 * time.Millisecond)
	deadline := r.Playback.Deadline
	result, v := handleRealtimePacketOnWire(store, protocol.Packet{Cmd: cmdClassicBattlePlayOverReq, Payload: mustJSON(t, battle.PlayOverRequest{BattleID: r.BattleID, PlaybackID: r.Playback.ID})}, s)
	if v != "" || len(result.battleCommands) == 0 || time.Now().Before(deadline) {
		t.Fatalf("grace granted early or kicked: %s", v)
	}
}

func TestRealtimeGuardAllCurrentCommandsHavePlaybackBudget(t *testing.T) {
	_, first := guardBattleFixture(t, 1)
	window, _ := first[0].battleRuntime.CommandWindowForActor(first[0].selectedRole.RoleID)
	if len(window.Commands) < 9 {
		t.Fatalf("missing current commands: %+v", window.Commands)
	}
	for _, definition := range window.Commands {
		t.Run(definition.ID, func(t *testing.T) {
			store, sockets := guardBattleFixture(t, 1)
			s := sockets[0]
			r := s.battleRuntime
			command, _ := r.CommandWindowForActor(s.selectedRole.RoleID)
			target := s.selectedRole.RoleID
			for i := range r.Cells {
				if r.Cells[i].Camp == battle.CampEnemy {
					target = r.Cells[i].Handle
					r.Cells[i].Dog = 0
				}
				if r.Cells[i].Camp == battle.CampTeam {
					r.Cells[i].MP = 10000
					r.Cells[i].MaxMP = 10000
				}
			}
			if definition.ID == "defense" || definition.ID == "battle-store" || definition.ID == "battle-escape" {
				target = s.selectedRole.RoleID
			}
			now := time.Now()
			result, violation := handleRealtimePacket(store, protocol.Packet{Cmd: cmdClassicBattleActionReq, Payload: mustJSON(t, battle.ActionRequest{BattleID: r.BattleID, ActorHandle: s.selectedRole.RoleID, CommandID: definition.ID, TargetHandle: target, Round: command.Round, Sequence: command.Sequence})}, s, now)
			if violation != "" || len(result.battleActions) == 0 || len(result.errorMessages) != 0 {
				t.Fatalf("command rejected: %s %+v", violation, result)
			}
			if r.Playback.ID != 1 || !r.Playback.Deadline.After(now) {
				t.Fatal("missing authoritative duration")
			}
			for _, action := range result.battleActions {
				if action.PlaybackID != 1 || action.PlaybackRemainingMS <= 0 {
					t.Fatal("action unprotected")
				}
			}
			if result, v := guardAck(t, store, s, 1, r.Playback.Deadline.Add(time.Hour)); v != "" || (result.battleOver == nil && len(result.battleCommands) == 0) {
				t.Fatalf("slow playback rejected: %s %+v", v, result)
			}
		})
	}
}

func TestRealtimeGuardItemAndReplay(t *testing.T) {
	store, sockets := guardBattleFixture(t, 1)
	s := sockets[0]
	r := s.battleRuntime
	item, ok := store.GrantRoleItem(s.playerBase.PlayerID, s.selectedRole.RoleID, session.RoleItem{Type: "背包", Name: "L花卷", ItemType: "own", Display: "213.png", Description: "f_i_L花卷^ffffff&24@消耗品&25@99&7@35&20@恢复气力", Count: 2, Index: 20, ItemLevel: 1})
	if !ok {
		t.Fatal("item")
	}
	command, _ := r.CommandWindowForActor(s.selectedRole.RoleID)
	q := battle.ItemActionRequest{BattleID: r.BattleID, ActorHandle: s.selectedRole.RoleID, Type: item.Type, Index: item.Index, Round: command.Round, Sequence: command.Sequence}
	packet := protocol.Packet{Cmd: cmdClassicBattleActiveItemReq, Payload: mustJSON(t, q)}
	now := time.Now()
	result, v := handleRealtimePacket(store, packet, s, now)
	if v != "" || len(result.battleActions) == 0 || result.battleActions[0].SourceActionLabel != "useItem" || r.Playback.ID != 1 {
		t.Fatalf("item not protected: %s %+v", v, result)
	}
	duplicate, v := handleRealtimePacket(store, packet, s, now)
	if v != "" || len(duplicate.battleActions) > 0 || r.Playback.ID != 1 {
		t.Fatal("item replayed")
	}
}

func TestRealtimeGuardConcurrentTeammateCommandsAndAcknowledgments(t *testing.T) {
	store, sockets := guardBattleFixture(t, 2)
	r := sockets[0].battleRuntime
	now := time.Now()
	packets := make([]protocol.Packet, 2)
	for i, s := range sockets {
		command, _ := r.CommandWindowForActor(s.selectedRole.RoleID)
		target := ""
		for _, c := range r.Cells {
			if c.Camp == battle.CampEnemy {
				target = c.Handle
				break
			}
		}
		packets[i] = protocol.Packet{Cmd: cmdClassicBattleActionReq, Payload: mustJSON(t, battle.ActionRequest{BattleID: r.BattleID, ActorHandle: s.selectedRole.RoleID, CommandID: battle.CommandNormalAttack, TargetHandle: target, Round: command.Round, Sequence: command.Sequence})}
	}
	var work sync.WaitGroup
	results := make([]packetResult, 2)
	violations := make([]string, 2)
	for i := range sockets {
		work.Add(1)
		go func(i int) {
			defer work.Done()
			results[i], violations[i] = handleRealtimePacket(store, packets[i], sockets[i], now)
		}(i)
	}
	work.Wait()
	if r.Playback.ID != 2 || violations[0] != "" || violations[1] != "" {
		t.Fatalf("concurrent commands: %v", violations)
	}
	for i := range sockets {
		packets[i] = protocol.Packet{Cmd: cmdClassicBattlePlayOverReq, Payload: mustJSON(t, battle.PlayOverRequest{BattleID: r.BattleID, PlaybackID: 2})}
	}
	deadline := r.Playback.Deadline
	for i := range sockets {
		work.Add(1)
		go func(i int) {
			defer work.Done()
			results[i], violations[i] = handleRealtimePacket(store, packets[i], sockets[i], deadline)
		}(i)
	}
	work.Wait()
	released := 0
	for i, result := range results {
		if violations[i] != "" {
			t.Fatal(violations[i])
		}
		if len(result.battleCommands) > 0 {
			released++
		}
	}
	if released != 1 {
		t.Fatalf("round released %d times", released)
	}
}
