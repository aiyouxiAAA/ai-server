package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ai-server/internal/protocol"
	"ai-server/internal/world"
	"github.com/gorilla/websocket"
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

func TestClassicTownSourceMonsterChaseDoesNotContinueWhenPlayerLeavesCaptureRadius(t *testing.T) {
	writer := &websocketWriter{}
	writer.resetClassicTownSourceMonsterState(world.TownBootstrapSnapshot{
		CreateRoles: []world.RolePush{
			{
				Handle:     "monster-a",
				RoleID:     "-2",
				Kind:       "monster",
				MapID:      "131",
				SpawnFlash: world.SpawnPoint{X: 1057, Y: 545},
			},
		},
	})

	firstMoves := writer.classicTownSourceMonsterChaseMoves(world.RoleMovePush{
		Handle: "player-a",
		Type:   "Run",
		X:      1024,
		Y:      580,
		TX:     1198,
		TY:     580,
		MapID:  "131",
	})
	if len(firstMoves) != 1 {
		t.Fatalf("expected initial chase move, got %+v", firstMoves)
	}

	farMoves := writer.classicTownSourceMonsterChaseMoves(world.RoleMovePush{
		Handle: "player-a",
		Type:   "Run",
		X:      1600,
		Y:      580,
		TX:     1700,
		TY:     580,
		MapID:  "131",
	})
	if len(farMoves) != 0 {
		t.Fatalf("expected no continued chase push outside capture radius, got %+v", farMoves)
	}
}

func TestClassicTownSourceMonsterChaseRetargetsToLatestNearbyPlayerMove(t *testing.T) {
	writer := &websocketWriter{}
	writer.resetClassicTownSourceMonsterState(world.TownBootstrapSnapshot{
		CreateRoles: []world.RolePush{
			{
				Handle:     "monster-a",
				RoleID:     "-2",
				Kind:       "monster",
				MapID:      "131",
				SpawnFlash: world.SpawnPoint{X: 1057, Y: 545},
			},
		},
	})

	firstMoves := writer.classicTownSourceMonsterChaseMoves(world.RoleMovePush{
		Handle: "player-a",
		Type:   "Run",
		X:      1024,
		Y:      580,
		TX:     1198,
		TY:     580,
		MapID:  "131",
	})
	if len(firstMoves) != 1 || firstMoves[0].TX != 1198 || firstMoves[0].TY != 580 {
		t.Fatalf("expected first player chase target, got %+v", firstMoves)
	}

	secondMoves := writer.classicTownSourceMonsterChaseMoves(world.RoleMovePush{
		Handle: "player-b",
		Type:   "Run",
		X:      1240,
		Y:      575,
		TX:     1300,
		TY:     575,
		MapID:  "131",
	})
	if len(secondMoves) != 1 {
		t.Fatalf("expected retarget chase move for second nearby player move, got %+v", secondMoves)
	}
	if secondMoves[0].Handle != "monster-a" || secondMoves[0].X != 1198 || secondMoves[0].Y != 580 || secondMoves[0].TX != 1300 || secondMoves[0].TY != 575 {
		t.Fatalf("unexpected retarget chase move: %+v", secondMoves[0])
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

func TestClassicTownSourceMonsterReplayKeepsMap171PatrolContinuousAcrossLoops(t *testing.T) {
	writer := &websocketWriter{}
	writer.resetClassicTownSourceMonsterState(world.TownBootstrapSnapshot{
		CreateRoles: []world.RolePush{
			{
				Handle:     "7893833328746190",
				RoleID:     "-2",
				Kind:       "monster",
				MapID:      "171",
				SpawnFlash: world.SpawnPoint{X: 1560, Y: 516},
			},
		},
	})

	toLeft := world.RoleMovePush{Handle: "7893833328746190", Type: "Move", X: 1112, Y: 516, TX: 942, TY: 516, MapID: "171"}
	toRight := world.RoleMovePush{Handle: "7893833328746190", Type: "Move", X: 1096, Y: 516, TX: 2027, TY: 516, MapID: "171"}

	first := writer.prepareClassicTownSourceMonsterReplayMove(toLeft)
	if first.X != 1560 || first.Y != 516 || first.TX != 942 || first.TY != 516 {
		t.Fatalf("expected first patrol to start at bootstrap position, got %+v", first)
	}
	writer.setClassicTownSourceMonsterPosition(first.Handle, first.MapID, world.SpawnPoint{X: first.TX, Y: first.TY})

	second := writer.prepareClassicTownSourceMonsterReplayMove(toRight)
	if second.X != 942 || second.Y != 516 || second.TX != 2027 || second.TY != 516 {
		t.Fatalf("expected next patrol to start at previous target, got %+v", second)
	}
	writer.setClassicTownSourceMonsterPosition(second.Handle, second.MapID, world.SpawnPoint{X: second.TX, Y: second.TY})

	nextLoop := writer.prepareClassicTownSourceMonsterReplayMove(toLeft)
	if nextLoop.X != 2027 || nextLoop.Y != 516 || nextLoop.TX != 942 || nextLoop.TY != 516 {
		t.Fatalf("expected loop boundary to continue from previous target, got %+v", nextLoop)
	}
}

func TestWebsocketWriterOutboundQueuePreservesOrderAndServerSeq(t *testing.T) {
	releaseServer := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		conn, err := upgrader.Upgrade(response, request, nil)
		if err != nil {
			t.Errorf("upgrade failed: %v", err)
			return
		}
		socketWriter := &websocketWriter{conn: conn, nextServerSeq: 1000}
		socketWriter.startOutbound()
		defer socketWriter.stopOutbound()

		if err := socketWriter.writePush(2001, []byte(`{"kind":"push-a"}`)); err != nil {
			t.Errorf("write first push failed: %v", err)
			return
		}
		if err := socketWriter.writePacket(2002, 77, []byte(`{"kind":"response"}`)); err != nil {
			t.Errorf("write response failed: %v", err)
			return
		}
		if err := socketWriter.writePush(2003, []byte(`{"kind":"push-b"}`)); err != nil {
			t.Errorf("write second push failed: %v", err)
			return
		}
		<-releaseServer
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	defer close(releaseServer)

	got := make([]protocol.Packet, 0, 3)
	for len(got) < 3 {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read message failed after %d packets: %v", len(got), err)
		}
		if messageType != websocket.BinaryMessage {
			t.Fatalf("expected binary websocket message, got %d", messageType)
		}
		packet, err := protocol.Decode(data)
		if err != nil {
			t.Fatalf("decode packet failed: %v", err)
		}
		got = append(got, packet)
	}

	if got[0].Cmd != 2001 || got[0].Seq != 1000 {
		t.Fatalf("unexpected first push: %+v", got[0])
	}
	if got[1].Cmd != 2002 || got[1].Seq != 77 {
		t.Fatalf("unexpected response packet: %+v", got[1])
	}
	if got[2].Cmd != 2003 || got[2].Seq != 1001 {
		t.Fatalf("unexpected second push: %+v", got[2])
	}
}
