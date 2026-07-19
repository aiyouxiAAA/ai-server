package classicactivity

import (
	"strconv"
	"testing"
	"time"
)

func TestPointCouponThiefRefreshSelectsOneCapturedSpawnPerRegion(t *testing.T) {
	start := time.Date(2026, 7, 13, 18, 0, 0, 0, time.Local)
	expectedPaths := map[int]map[Point]bool{
		21:  {{X: 752, Y: 522}: true},
		29:  {{X: 1012, Y: 614}: true},
		84:  {{X: 1106, Y: 523}: true},
		92:  {{X: 600, Y: 529}: true},
		100: {{X: 1280, Y: 532}: true},
		114: {{X: 2550, Y: 491}: true, {X: 1415, Y: 548}: true},
		115: {{X: 5057, Y: 524}: true},
		117: {{X: 935, Y: 467}: true},
	}
	seenPaths := map[int]map[Point]bool{}

	for offset := 0; offset < 12; offset++ {
		now := start.Add(time.Duration(offset) * time.Hour)
		spawns := pointCouponThiefSpawnsAtHour(pointCouponThiefHourBucket(now))
		if len(spawns) != 3 {
			t.Fatalf("expected one spawn per activity region, got %+v", spawns)
		}
		regions := map[int]bool{}
		for _, spawn := range spawns {
			regionID, ok := pointCouponThiefRegionIDForMap(spawn.MapID)
			if !ok || regions[regionID] {
				t.Fatalf("expected distinct activity regions, got %+v", spawns)
			}
			regions[regionID] = true
			if !expectedPaths[spawn.MapID][spawn.Source] {
				t.Fatalf("unexpected captured spawn %+v", spawn)
			}
			if !IsPointCouponThiefHandle(strconv.Itoa(spawn.MapID), spawn.Handle) {
				t.Fatalf("expected valid activity handle for spawn %+v", spawn)
			}
			if _, ok := seenPaths[spawn.MapID]; !ok {
				seenPaths[spawn.MapID] = map[Point]bool{}
			}
			seenPaths[spawn.MapID][spawn.Source] = true
		}
	}

	for mapID, paths := range expectedPaths {
		for path := range paths {
			if !seenPaths[mapID][path] {
				t.Fatalf("expected captured map%d point %+v to be selectable, got %+v", mapID, path, seenPaths)
			}
		}
	}
	if _, ok := PointCouponThiefSpawnForMap(1, start); ok {
		t.Fatal("point coupon thief must not spawn in novice village map1")
	}
}

func TestPointCouponThiefHandleChangesOnHourBoundary(t *testing.T) {
	ResetPointCouponThiefRefreshStateForTest()
	t.Cleanup(ResetPointCouponThiefRefreshStateForTest)
	firstHour := time.Date(2026, 6, 19, 18, 1, 0, 0, time.Local)
	secondHour := time.Date(2026, 6, 19, 19, 0, 0, 0, time.Local)
	first := pointCouponThiefSpawnsAtHour(pointCouponThiefHourBucket(firstHour))
	second := pointCouponThiefSpawnsAtHour(pointCouponThiefHourBucket(secondHour))
	if len(first) != 3 || len(second) != 3 {
		t.Fatalf("expected three activity spawns each hour, got first=%+v second=%+v", first, second)
	}
	for _, firstSpawn := range first {
		regionID, ok := pointCouponThiefRegionIDForMap(firstSpawn.MapID)
		if !ok {
			t.Fatalf("expected known region for first spawn %+v", firstSpawn)
		}
		for _, secondSpawn := range second {
			secondRegionID, secondOK := pointCouponThiefRegionIDForMap(secondSpawn.MapID)
			if secondOK && regionID == secondRegionID && firstSpawn.Handle == secondSpawn.Handle {
				t.Fatalf("expected hourly refresh to replace region%d handle, got %s", regionID, firstSpawn.Handle)
			}
		}
	}
}

func TestForcePointCouponThiefRefreshReplacesCurrentCycleAndNaturalHourResumes(t *testing.T) {
	ResetPointCouponThiefRefreshStateForTest()
	t.Cleanup(ResetPointCouponThiefRefreshStateForTest)
	now := time.Date(2026, 6, 19, 18, 37, 0, 0, time.Local)

	refresh := ForcePointCouponThiefRefreshForDev(now)
	if len(refresh.Previous) != 3 || len(refresh.Current) != 3 {
		t.Fatalf("expected three removed and three active activity spawns, got %+v", refresh)
	}
	for _, current := range refresh.Current {
		active, ok := PointCouponThiefSpawnForMap(current.MapID, now)
		if !ok || active.Handle != current.Handle {
			t.Fatalf("expected map%d bootstrap to use refreshed handle %s, got %+v", current.MapID, current.Handle, active)
		}
	}
	secondForced := ForcePointCouponThiefRefreshForDev(now)
	for _, firstSpawn := range refresh.Current {
		firstRegionID, firstOK := pointCouponThiefRegionIDForMap(firstSpawn.MapID)
		if !firstOK {
			t.Fatalf("expected known region for forced spawn %+v", firstSpawn)
		}
		for _, secondSpawn := range secondForced.Current {
			secondRegionID, secondOK := pointCouponThiefRegionIDForMap(secondSpawn.MapID)
			if secondOK && firstRegionID == secondRegionID && firstSpawn.Source == secondSpawn.Source {
				t.Fatalf("expected consecutive forced refreshes to choose a new region%d path, got %+v", firstRegionID, firstSpawn)
			}
		}
	}

	natural := AdvancePointCouponThiefRefresh(now.Add(time.Hour))
	for _, forced := range secondForced.Current {
		if _, ok := pointCouponThiefRefreshSpawnForHandle(natural.Previous, forced.Handle); !ok {
			t.Fatalf("expected natural refresh to remove forced handle %s, got %+v", forced.Handle, natural.Previous)
		}
	}
	for _, current := range natural.Current {
		active, ok := PointCouponThiefSpawnForMap(current.MapID, now.Add(time.Hour))
		if !ok || active.Handle != current.Handle {
			t.Fatalf("expected natural map%d handle after dev refresh, got %+v", current.MapID, active)
		}
	}
}

func pointCouponThiefRefreshSpawnForHandle(spawns []PointCouponThiefSpawn, handle string) (PointCouponThiefSpawn, bool) {
	for _, spawn := range spawns {
		if spawn.Handle == handle {
			return spawn, true
		}
	}
	return PointCouponThiefSpawn{}, false
}

func pointCouponThiefRegionIDForMap(mapID int) (int, bool) {
	for _, region := range pointCouponThiefRegions {
		for _, path := range region.spawnPaths {
			if path.mapID == mapID {
				return region.regionID, true
			}
		}
	}
	return 0, false
}
