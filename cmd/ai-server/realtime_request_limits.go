package main

import (
	"ai-server/internal/classicdata"
	"encoding/json"
	"strconv"
	"time"
)

// Engineering abuse limit, independent of game frames: 120 requests/s, 240 burst.
// A login's preload requests and buffered network delivery fit in the burst.
type realtimeRequestBudget struct {
	last   time.Time
	credit float64
}

func (b *realtimeRequestBudget) accept(now time.Time) bool {
	if b.last.IsZero() {
		b.last = now
		b.credit = 240
	}
	elapsed := now.Sub(b.last).Seconds()
	if elapsed < 0 {
		return false
	}
	b.credit += elapsed * 120
	if b.credit > 240 {
		b.credit = 240
	}
	b.last = now
	if b.credit < 1 {
		return false
	}
	b.credit--
	return true
}
func validateRealtimeTransfer(payload []byte) string {
	var q classicTownTransferRequest
	if json.Unmarshal(payload, &q) != nil {
		return "malformed_transfer"
	}
	id, err := strconv.Atoi(q.MapID)
	if err != nil {
		return "transfer_map_spoof"
	}
	// A free-map request cannot select an arbitrary landing point and reset its move budget.
	// Match ClassicTownMapTransfer's explicit source mapbox/default contract.
	x, y := 1000, 600
	for _, row := range classicdata.ClassicMapPageTransfers() {
		if row.TransferType == "mapbox" && row.MapID == id {
			x, y = row.LandingX, row.LandingY
			break
		}
	}
	if q.X != x || q.Y != y {
		return "transfer_position_spoof"
	}
	return ""
}
