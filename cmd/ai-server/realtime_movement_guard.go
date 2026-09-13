package main

import (
	"ai-server/internal/world"
	"encoding/json"
	"math"
	"strconv"
	"sync"
	"time"
)

// Source: ClassicTownConfig BASE_SPEED=130 /25*30, run multiplier=2.
// This bounded credit absorbs network bunching; it is spent cumulatively, never per packet.
const movementUnitsPerSecond = 312.0
const movementBurstSeconds = 1.5

type realtimeMovementGuard struct {
	mu                   sync.Mutex
	role                 string
	mapID, previousMapID int
	position             world.SpawnPoint
	last                 time.Time
	credit               float64
}

func (g *realtimeMovementGuard) reset(role string, mapID int, position world.SpawnPoint, now time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.role != role {
		g.previousMapID = 0
	} else if g.mapID != mapID {
		// Same-map bootstraps must not erase the map we actually just left.
		g.previousMapID = g.mapID
	}
	g.role, g.mapID, g.position, g.last = role, mapID, position, now
	g.credit = movementUnitsPerSecond * movementBurstSeconds
}

func (g *realtimeMovementGuard) isPreviousMap(role string, currentMapID int, requestedMapID string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.isPreviousMapLocked(role, currentMapID, requestedMapID)
}

func (g *realtimeMovementGuard) isPreviousMapLocked(role string, currentMapID int, requestedMapID string) bool {
	return role != "" && g.role == role && g.mapID == currentMapID && !g.last.IsZero() &&
		g.previousMapID > 0 && g.previousMapID != currentMapID && requestedMapID == strconv.Itoa(g.previousMapID)
}

func validateRealtimeMovement(socket *packetSession, payload []byte, now time.Time) (bool, string) {
	var request classicTownMoveRoleRequest
	if json.Unmarshal(payload, &request) != nil {
		return false, "malformed_movement"
	}
	if socket.selectedRole == nil || socket.playerBase == nil {
		return false, "unauthenticated_movement"
	}
	role := socket.selectedRole.RoleID
	if request.Handle != "" && request.Handle != role {
		return false, "movement_actor_spoof"
	}
	if request.Type != "Run" && request.Type != "Move" {
		return false, "movement_type_spoof"
	}
	for _, n := range []int{request.X, request.Y, request.TX, request.TY} {
		if n < -1000000 || n > 1000000 {
			return false, "movement_coordinate_range"
		}
	}
	if request.Z != 0 || request.TZ != 0 {
		return false, "movement_height_spoof"
	}
	g := &socket.movement
	g.mu.Lock()
	defer g.mu.Unlock()
	mapID, err := strconv.Atoi(request.MapID)
	if err != nil {
		return false, "movement_map_spoof"
	}
	if mapID != socket.playerBase.MapID {
		if g.isPreviousMapLocked(role, socket.playerBase.MapID, request.MapID) {
			return true, ""
		}
		return false, "movement_map_spoof"
	}
	if g.role != role || g.mapID != mapID || g.last.IsZero() {
		return true, ""
	}
	elapsed := now.Sub(g.last).Seconds()
	if elapsed < 0 {
		return true, ""
	}
	credit := math.Min(movementUnitsPerSecond*movementBurstSeconds, g.credit+elapsed*movementUnitsPerSecond)
	// Match ClassicTownConfig Y_AXIS_SCALE=0.5, including diagonal paths.
	distance := math.Hypot(float64(request.X-g.position.X), 2*float64(request.Y-g.position.Y))
	if distance > credit+3 {
		return false, "movement_speed_exceeded"
	}
	g.credit = credit - distance
	g.last, g.position = now, world.SpawnPoint{X: request.X, Y: request.Y}
	return false, ""
}
