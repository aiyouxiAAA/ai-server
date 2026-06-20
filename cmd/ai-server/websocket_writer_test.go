package main

import (
	"testing"

	"ai-server/internal/world"
)

func TestClassicTownSourceMonsterChasePrefersPlayerTargetWithinCaptureRadius(t *testing.T) {
	writer := &websocketWriter{}
	writer.resetClassicTownSourceMonsterState(world.TownBootstrapSnapshot{
		CreateRoles: []world.RolePush{
			{
				Handle:     "8122205778895755",
				RoleID:     "-2",
				Kind:       "monster",
				MapID:      "131",
				SpawnFlash: world.SpawnPoint{X: 1057, Y: 545},
			},
		},
	})

	moves := writer.classicTownSourceMonsterChaseMoves(world.RoleMovePush{
		Handle: "player_21424",
		Type:   "Run",
		X:      1024,
		Y:      580,
		TX:     1198,
		TY:     580,
		MapID:  "131",
	})

	if len(moves) != 1 {
		t.Fatalf("expected one source monster chase move, got %+v", moves)
	}
	got := moves[0]
	if got.Handle != "8122205778895755" || got.Type != "Run" || got.X != 1057 || got.Y != 545 || got.TX != 1198 || got.TY != 580 || got.MapID != "131" {
		t.Fatalf("unexpected chase move: %+v", got)
	}
	if repeat := writer.classicTownSourceMonsterChaseMoves(world.RoleMovePush{Handle: "player_21424", Type: "Run", X: 1024, Y: 580, TX: 1198, TY: 580, MapID: "131"}); len(repeat) != 0 {
		t.Fatalf("same target should not be pushed twice, got %+v", repeat)
	}
}

func TestClassicTownSourceMonsterChaseUsesCurrentWhenTargetOutsideCaptureRadius(t *testing.T) {
	writer := &websocketWriter{}
	writer.resetClassicTownSourceMonsterState(world.TownBootstrapSnapshot{
		CreateRoles: []world.RolePush{
			{
				Handle:     "8122205778895755",
				RoleID:     "-2",
				Kind:       "monster",
				MapID:      "131",
				SpawnFlash: world.SpawnPoint{X: 1161, Y: 559},
			},
		},
	})

	moves := writer.classicTownSourceMonsterChaseMoves(world.RoleMovePush{
		Handle: "player_21424",
		Type:   "Run",
		X:      1287,
		Y:      576,
		TX:     1410,
		TY:     515,
		MapID:  "131",
	})

	if len(moves) != 1 {
		t.Fatalf("expected one source monster chase move, got %+v", moves)
	}
	got := moves[0]
	if got.TX != 1287 || got.TY != 576 {
		t.Fatalf("expected chase target to use current player point inside radius, got %+v", got)
	}
}

func TestClassicTownSourceMonsterChaseSkipsOutsideCaptureRadiusAndWrongMap(t *testing.T) {
	writer := &websocketWriter{}
	writer.resetClassicTownSourceMonsterState(world.TownBootstrapSnapshot{
		CreateRoles: []world.RolePush{
			{
				Handle:     "monster-a",
				RoleID:     "-2",
				Kind:       "monster",
				MapID:      "131",
				SpawnFlash: world.SpawnPoint{X: 100, Y: 100},
			},
			{
				Handle:     "monster-b",
				RoleID:     "-2",
				Kind:       "monster",
				MapID:      "132",
				SpawnFlash: world.SpawnPoint{X: 1020, Y: 100},
			},
		},
	})

	moves := writer.classicTownSourceMonsterChaseMoves(world.RoleMovePush{
		Handle: "player_21424",
		Type:   "Run",
		X:      1000,
		Y:      100,
		TX:     1100,
		TY:     100,
		MapID:  "131",
	})

	if len(moves) != 0 {
		t.Fatalf("expected no chase outside radius or across map, got %+v", moves)
	}
}

func TestClassicTownSourceMonsterReplaySkipsCapturedRunChaseSteps(t *testing.T) {
	if classicTownSourceMonsterReplayableStep(world.RoleMovePush{Type: "Run"}) {
		t.Fatal("captured Run moveRole steps are chase samples and must not replay as static patrol")
	}
	if !classicTownSourceMonsterReplayableStep(world.RoleMovePush{Type: "Move"}) {
		t.Fatal("captured Move moveRole steps must remain replayable patrol movement")
	}
}
