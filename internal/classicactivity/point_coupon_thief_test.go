package classicactivity

import (
	"strconv"
	"testing"
	"time"
)

func TestPointCouponThiefSpawnForMapUsesOnlyCapturedActivityMaps(t *testing.T) {
	now := time.Date(2026, 6, 19, 18, 37, 0, 0, time.Local)
	for _, mapID := range []int{84, 114, 115} {
		spawn, ok := PointCouponThiefSpawnForMap(mapID, now)
		if !ok {
			t.Fatalf("expected point coupon thief spawn on map%d", mapID)
		}
		if !IsPointCouponThiefHandle(strconv.Itoa(mapID), spawn.Handle) {
			t.Fatalf("expected valid activity handle for map%d, got %+v", mapID, spawn)
		}
	}
	if _, ok := PointCouponThiefSpawnForMap(1, now); ok {
		t.Fatal("point coupon thief must not spawn in novice village map1")
	}
}

func TestPointCouponThiefHandleChangesOnHourBoundary(t *testing.T) {
	firstHour := time.Date(2026, 6, 19, 18, 1, 0, 0, time.Local)
	secondHour := time.Date(2026, 6, 19, 19, 0, 0, 0, time.Local)
	first, ok := PointCouponThiefSpawnForMap(84, firstHour)
	if !ok {
		t.Fatal("expected first hour point coupon thief")
	}
	second, ok := PointCouponThiefSpawnForMap(84, secondHour)
	if !ok {
		t.Fatal("expected second hour point coupon thief")
	}
	if first.Handle == second.Handle {
		t.Fatalf("expected hourly refresh to produce a new handle, got %s", first.Handle)
	}
}
