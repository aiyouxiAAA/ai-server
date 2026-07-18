package main

import (
	"testing"

	"ai-server/internal/battle"
	"ai-server/internal/protocol"
	"ai-server/internal/session"
)

func TestClassicBattleSpectatorBuildsReadOnlySnapshotAndStopsByBattleID(t *testing.T) {
	previousTeamHub := classicTeamHub
	previousSpectators := classicBattleSpectators
	classicTeamHub = newClassicTeamConnectionHub()
	classicBattleSpectators = newClassicBattleSpectatorHub()
	t.Cleanup(func() {
		classicTeamHub = previousTeamHub
		classicBattleSpectators = previousSpectators
	})

	store := session.NewStore()
	targetSession, _ := seedSelectedRoleSessionInStore(t, store, "被观战玩家")
	observerSession, _ := seedSelectedRoleSessionInStore(t, store, "观战玩家")
	runtime, _, ok := battle.NewWildBattle(*targetSession.selectedRole, *targetSession.playerBase, battle.StartRequest{
		MapID:       "4",
		MapName:     "云隐村口",
		StageFocusX: 120,
		ReturnRoute: "town-placeholder",
	})
	if !ok {
		t.Fatal("expected target battle runtime")
	}
	targetSession.battleRuntime = runtime
	classicTeamHub.register(targetSession.selectedRole.RoleID, &websocketWriter{}, targetSession)
	classicTeamHub.register(observerSession.selectedRole.RoleID, &websocketWriter{}, observerSession)

	join := handlePacketWithSession(store, protocol.Packet{
		Cmd: cmdClassicBattleSpectateReq,
		Payload: mustJSON(t, classicBattleSpectateRequest{
			TargetHandle: targetSession.selectedRole.RoleID,
		}),
	}, observerSession)
	if !join.handled || join.battleSpectatorStart == nil {
		t.Fatalf("expected spectator snapshot, got %+v", join)
	}
	if !join.battleSpectatorStart.Start.Spectator || join.battleSpectatorStart.Start.SelfHandle != "" {
		t.Fatalf("expected read-only source battle start, got %+v", join.battleSpectatorStart.Start)
	}
	if join.battleSpectatorStart.Start.BattleID != runtime.BattleID || len(join.battleSpectatorStart.Cells) != len(runtime.Cells) {
		t.Fatalf("expected current battle cells, got start=%+v cells=%+v", join.battleSpectatorStart.Start, join.battleSpectatorStart.Cells)
	}
	if !classicBattleSpectators.activate(observerSession.selectedRole.RoleID, runtime.BattleID) {
		t.Fatal("expected observer activation")
	}

	stop := handlePacketWithSession(store, protocol.Packet{
		Cmd:     cmdClassicBattleStopSpectate,
		Payload: mustJSON(t, classicBattleStopSpectateRequest{BattleID: runtime.BattleID}),
	}, observerSession)
	if !stop.handled || stop.battleSpectatorStop == nil || stop.battleSpectatorStop.BattleID != runtime.BattleID {
		t.Fatalf("expected spectator stop for the current battle, got %+v", stop)
	}
	if removed := classicBattleSpectators.remove(observerSession.selectedRole.RoleID); removed != runtime.BattleID {
		t.Fatalf("expected observer removal for %s, got %q", runtime.BattleID, removed)
	}
	if targetSession.battleRuntime != runtime {
		t.Fatal("spectator exit must not mutate the watched runtime")
	}
}
