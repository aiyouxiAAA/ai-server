package classicactivity

import (
	"strconv"
	"testing"
	"time"
)

func testXiongluCycleStart(t *testing.T) time.Time {
	t.Helper()
	base := time.Date(2026, 7, 20, 22, 0, 0, 0, time.Local)
	return XiongluBeardeerCycleStart(base)
}

func TestXiongluBeardeerPhaseUsesTenMinuteWarningThenSpawn(t *testing.T) {
	start := testXiongluCycleStart(t)
	if phase := XiongluBeardeerPhaseAt(start); phase != XiongluBeardeerPhaseWarning {
		t.Fatalf("expected warning at cycle start, got %s", phase)
	}
	if XiongluBeardeerIsActive(start) {
		t.Fatal("boss should not be active during warning window")
	}
	spawnAt := start.Add(XiongluBeardeerWarningLead)
	if phase := XiongluBeardeerPhaseAt(spawnAt); phase != XiongluBeardeerPhaseSpawned {
		t.Fatalf("expected spawned phase after warning lead, got %s", phase)
	}
	if !XiongluBeardeerIsActive(spawnAt) {
		t.Fatal("expected boss active after warning lead")
	}
	nearEnd := start.Add(XiongluBeardeerCycle - time.Second)
	if phase := XiongluBeardeerPhaseAt(nearEnd); phase != XiongluBeardeerPhaseSpawned {
		t.Fatalf("expected still spawned near cycle end, got %s", phase)
	}
	nextStart := start.Add(XiongluBeardeerCycle)
	if phase := XiongluBeardeerPhaseAt(nextStart); phase != XiongluBeardeerPhaseWarning {
		t.Fatalf("expected next-cycle warning at +cycle, got %s", phase)
	}
	if XiongluBeardeerCycleBucket(start) == XiongluBeardeerCycleBucket(nextStart) {
		t.Fatal("expected distinct cycle buckets across cycle boundary")
	}
}

func TestXiongluBeardeerSpawnForMapOnlyOnActiveMap203(t *testing.T) {
	start := testXiongluCycleStart(t)
	active := start.Add(XiongluBeardeerWarningLead + time.Minute)
	spawn, ok := XiongluBeardeerSpawnForMap(XiongluBeardeerMapID, active)
	if !ok {
		t.Fatal("expected map203 spawn while active")
	}
	if spawn.Handle != XiongluBeardeerHandle || spawn.Source.X != 1058 || spawn.Source.Y != 569 {
		t.Fatalf("unexpected boss spawn %+v", spawn)
	}
	if len(spawn.Companions) != 3 {
		t.Fatalf("expected 3 robothyun companions, got %+v", spawn.Companions)
	}
	if spawn.Companions[0].Handle != "4266636384163669" || spawn.Companions[0].X != 348 || spawn.Companions[0].Y != 632 {
		t.Fatalf("expected captured companion spawn, got %+v", spawn.Companions[0])
	}
	if _, ok := XiongluBeardeerSpawnForMap(XiongluBeardeerMapID, start.Add(5*time.Minute)); ok {
		t.Fatal("expected no spawn during warning window")
	}
	if _, ok := XiongluBeardeerSpawnForMap(202, active); ok {
		t.Fatal("expected no spawn on non-203 maps")
	}
}

func TestXiongluBeardeerHandleHelpers(t *testing.T) {
	if !IsXiongluBeardeerHandle(strconv.Itoa(XiongluBeardeerMapID), XiongluBeardeerHandle) {
		t.Fatal("expected boss handle recognition")
	}
	if !IsXiongluBeardeerEncounterHandle("4268636384164547") {
		t.Fatal("expected companion encounter handle recognition")
	}
	if IsXiongluBeardeerHandle("202", XiongluBeardeerHandle) {
		t.Fatal("boss handle should not match wrong map")
	}
	if got := XiongluBeardeerKillNoticeText("恐龙抗狼1", "桥头的樵夫", "阿柴"); got != "[恐龙抗狼1][桥头的樵夫][阿柴]在[雷兽神坛]消灭[熊鹿]。" {
		t.Fatalf("unexpected kill notice %q", got)
	}
	if len(XiongluBeardeerEncounterHandles()) != 4 {
		t.Fatalf("expected 4 encounter handles, got %v", XiongluBeardeerEncounterHandles())
	}
}

func TestXiongluBeardeerKillHidesUntilNextCycle(t *testing.T) {
	ResetXiongluBeardeerKillStateForTest()
	t.Cleanup(ResetXiongluBeardeerKillStateForTest)

	start := testXiongluCycleStart(t)
	active := start.Add(XiongluBeardeerWarningLead + time.Minute)
	if !XiongluBeardeerIsAlive(active) {
		t.Fatal("expected alive before kill in active window")
	}
	MarkXiongluBeardeerKilled(active)
	if XiongluBeardeerIsAlive(active) {
		t.Fatal("expected dead after kill in same cycle")
	}
	if _, ok := XiongluBeardeerSpawnForMap(XiongluBeardeerMapID, active); ok {
		t.Fatal("expected no spawn after kill in same cycle")
	}
	nextCycle := start.Add(XiongluBeardeerCycle + XiongluBeardeerWarningLead + time.Minute)
	if !XiongluBeardeerIsAlive(nextCycle) {
		t.Fatal("expected alive again in next cycle after kill")
	}
}

func TestForceXiongluBeardeerRefreshForDevOverridesWarningOnlyForCurrentCycle(t *testing.T) {
	ResetXiongluBeardeerKillStateForTest()
	t.Cleanup(ResetXiongluBeardeerKillStateForTest)

	start := testXiongluCycleStart(t)
	warning := start.Add(time.Minute)
	if XiongluBeardeerIsAlive(warning) {
		t.Fatal("boss should not be alive during the normal warning window")
	}

	ForceXiongluBeardeerRefreshForDev(warning)
	if !XiongluBeardeerIsForcedForDev(warning) || !XiongluBeardeerIsAlive(warning) {
		t.Fatal("dev refresh should make the boss alive during the warning window")
	}
	if _, ok := XiongluBeardeerSpawnForMap(XiongluBeardeerMapID, warning); !ok {
		t.Fatal("dev refresh should restore the map203 encounter spawn")
	}

	MarkXiongluBeardeerKilled(warning)
	if XiongluBeardeerIsAlive(warning) {
		t.Fatal("killing a dev-refreshed boss should remove it")
	}

	nextWarning := start.Add(XiongluBeardeerCycle + time.Minute)
	if XiongluBeardeerIsForcedForDev(nextWarning) {
		t.Fatal("dev refresh must not cross a natural cycle boundary")
	}
	if XiongluBeardeerIsAlive(nextWarning) {
		t.Fatal("next cycle should return to its normal warning phase")
	}
}

func TestXiongluBeardeerCycleMatchesHourlyQuestCopy(t *testing.T) {
	if XiongluBeardeerCycle != 3600*time.Second {
		t.Fatalf("expected quest-backed hourly cycle 3600s, got %s", XiongluBeardeerCycle)
	}
	if XiongluBeardeerWarningLead != 10*time.Minute {
		t.Fatalf("expected 10-minute warning lead, got %s", XiongluBeardeerWarningLead)
	}
}
