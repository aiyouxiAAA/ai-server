package classicactivity

import (
	"strconv"
	"testing"
	"time"
)

func testBainianCycleStart(t *testing.T) time.Time {
	t.Helper()
	// Anchor to a known local wall time, then snap to free-running cycle boundary.
	base := time.Date(2026, 7, 12, 12, 0, 0, 0, time.Local)
	return BainianChongjingCycleStart(base)
}

func TestBainianChongjingPhaseUsesTenMinuteWarningThenSpawn(t *testing.T) {
	start := testBainianCycleStart(t)
	if phase := BainianChongjingPhaseAt(start); phase != BainianChongjingPhaseWarning {
		t.Fatalf("expected warning at cycle start, got %s", phase)
	}
	if BainianChongjingIsActive(start) {
		t.Fatal("boss should not be active during warning window")
	}
	spawnAt := start.Add(BainianChongjingWarningLead)
	if phase := BainianChongjingPhaseAt(spawnAt); phase != BainianChongjingPhaseSpawned {
		t.Fatalf("expected spawned phase after warning lead, got %s", phase)
	}
	if !BainianChongjingIsActive(spawnAt) {
		t.Fatal("expected boss active after warning lead")
	}
	nearEnd := start.Add(BainianChongjingCycle - time.Second)
	if phase := BainianChongjingPhaseAt(nearEnd); phase != BainianChongjingPhaseSpawned {
		t.Fatalf("expected still spawned near cycle end, got %s", phase)
	}
	nextStart := start.Add(BainianChongjingCycle)
	if phase := BainianChongjingPhaseAt(nextStart); phase != BainianChongjingPhaseWarning {
		t.Fatalf("expected next-cycle warning at +cycle, got %s", phase)
	}
	if BainianChongjingCycleBucket(start) == BainianChongjingCycleBucket(nextStart) {
		t.Fatal("expected distinct cycle buckets across cycle boundary")
	}
}

func TestBainianChongjingSpawnForMapOnlyOnActiveMap171(t *testing.T) {
	start := testBainianCycleStart(t)
	active := start.Add(BainianChongjingWarningLead + time.Minute)
	spawn, ok := BainianChongjingSpawnForMap(BainianChongjingMapID, active)
	if !ok {
		t.Fatal("expected map171 spawn while active")
	}
	if spawn.Handle != BainianChongjingHandle || spawn.Source.X != 1560 || spawn.Source.Y != 516 {
		t.Fatalf("unexpected boss spawn %+v", spawn)
	}
	if len(spawn.Militias) != 3 {
		t.Fatalf("expected 3 militia companions, got %+v", spawn.Militias)
	}
	if spawn.Militias[0].Handle != "7895833328747103" || spawn.Militias[0].Vocation != "游侠+" || spawn.Militias[0].X != 1539 || spawn.Militias[0].Y != 552 {
		t.Fatalf("expected captured ranger militia spawn, got %+v", spawn.Militias[0])
	}
	if _, ok := BainianChongjingSpawnForMap(BainianChongjingMapID, start.Add(5*time.Minute)); ok {
		t.Fatal("expected no spawn during warning window")
	}
	if _, ok := BainianChongjingSpawnForMap(170, active); ok {
		t.Fatal("expected no spawn on non-171 maps")
	}
}

func TestBainianChongjingHandleHelpers(t *testing.T) {
	if !IsBainianChongjingHandle(strconv.Itoa(BainianChongjingMapID), BainianChongjingHandle) {
		t.Fatal("expected boss handle recognition")
	}
	if !IsBainianChongjingEncounterHandle("7897833328748728") {
		t.Fatal("expected militia encounter handle recognition")
	}
	if IsBainianChongjingHandle("170", BainianChongjingHandle) {
		t.Fatal("boss handle should not match wrong map")
	}
	if got := BainianChongjingKillNoticeText("恐龙抗狼1", "桥头的樵夫"); got != "[恐龙抗狼1][桥头的樵夫]在[沼泽_9]消灭[百年虫精]。" {
		t.Fatalf("unexpected kill notice %q", got)
	}
}

func TestBainianChongjingKillHidesUntilNextCycle(t *testing.T) {
	ResetBainianChongjingKillStateForTest()
	t.Cleanup(ResetBainianChongjingKillStateForTest)

	start := testBainianCycleStart(t)
	active := start.Add(BainianChongjingWarningLead + time.Minute)
	if !BainianChongjingIsAlive(active) {
		t.Fatal("expected alive before kill in active window")
	}
	MarkBainianChongjingKilled(active)
	if BainianChongjingIsAlive(active) {
		t.Fatal("expected dead after kill in same cycle")
	}
	if _, ok := BainianChongjingSpawnForMap(BainianChongjingMapID, active); ok {
		t.Fatal("expected no spawn after kill in same cycle")
	}
	nextCycle := start.Add(BainianChongjingCycle + BainianChongjingWarningLead + time.Minute)
	if !BainianChongjingIsAlive(nextCycle) {
		t.Fatal("expected alive again in next cycle after kill")
	}
}

func TestForceBainianChongjingRefreshForDevOverridesWarningOnlyForCurrentCycle(t *testing.T) {
	ResetBainianChongjingKillStateForTest()
	t.Cleanup(ResetBainianChongjingKillStateForTest)

	start := testBainianCycleStart(t)
	warning := start.Add(time.Minute)
	if BainianChongjingIsAlive(warning) {
		t.Fatal("boss should not be alive during the normal warning window")
	}

	ForceBainianChongjingRefreshForDev(warning)
	if !BainianChongjingIsForcedForDev(warning) || !BainianChongjingIsAlive(warning) {
		t.Fatal("dev refresh should make the boss alive during the warning window")
	}
	if _, ok := BainianChongjingSpawnForMap(BainianChongjingMapID, warning); !ok {
		t.Fatal("dev refresh should restore the map171 encounter spawn")
	}

	MarkBainianChongjingKilled(warning)
	if BainianChongjingIsAlive(warning) {
		t.Fatal("killing a dev-refreshed boss should remove it")
	}

	nextWarning := start.Add(BainianChongjingCycle + time.Minute)
	if BainianChongjingIsForcedForDev(nextWarning) {
		t.Fatal("dev refresh must not cross a natural cycle boundary")
	}
	if BainianChongjingIsAlive(nextWarning) {
		t.Fatal("next cycle should return to its normal warning phase")
	}
}

func TestBainianChongjingCycleMatchesCaptureMedianInterval(t *testing.T) {
	// Capture inlier consecutive warn intervals median ≈ 7269s; warning lead stays 10 minutes.
	if BainianChongjingCycle != 7269*time.Second {
		t.Fatalf("expected capture-backed cycle 7269s, got %s", BainianChongjingCycle)
	}
	if BainianChongjingWarningLead != 10*time.Minute {
		t.Fatalf("expected 10-minute warning lead, got %s", BainianChongjingWarningLead)
	}
}
