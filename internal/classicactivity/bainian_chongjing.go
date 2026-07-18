package classicactivity

import (
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	BainianChongjingName        = "百年虫精"
	BainianChongjingSourceQuery = "monstermap/wocmon.swf"
	BainianChongjingSpriteName  = "wocmon"
	BainianChongjingMapID       = 171
	BainianChongjingMapName     = "沼泽_9"
	BainianChongjingHandle      = "7893833328746190"
	// Capture traffic-preview samples across many sessions:
	// - warn → spawn median ≈ 600.7s (keep 10-minute warning lead)
	// - consecutive warn intervals inlier n=35 median ≈ 7269s (~121.15m)
	// Not wall-clock half-hour aligned; use free-running Unix-epoch buckets.
	BainianChongjingCycle       = 7269 * time.Second
	BainianChongjingWarningLead = 10 * time.Minute
)

type BainianChongjingPhase string

const (
	BainianChongjingPhaseIdle    BainianChongjingPhase = "idle"
	BainianChongjingPhaseWarning BainianChongjingPhase = "warning"
	BainianChongjingPhaseSpawned BainianChongjingPhase = "spawned"
)

type BainianChongjingMilitiaSpawn struct {
	Handle   string
	Vocation string
	X        int
	Y        int
}

type BainianChongjingSpawn struct {
	MapID    int
	Handle   string
	Source   Point
	Militias []BainianChongjingMilitiaSpawn
}

// Capture-backed map entity samples from 20260712_120132_155_auto_move_session.
var bainianChongjingBossSpawn = Point{X: 1560, Y: 516}

var bainianChongjingMilitias = []BainianChongjingMilitiaSpawn{
	{Handle: "7895833328747103", Vocation: "游侠+", X: 1539, Y: 552},
	{Handle: "7897833328748728", Vocation: "战士+", X: 1848, Y: 600},
	{Handle: "7899833328749140", Vocation: "战士+", X: 1992, Y: 533},
}

var (
	bainianChongjingMu           sync.Mutex
	bainianChongjingKilledBucket int64
	bainianChongjingKilled       bool
	bainianChongjingForcedBucket int64
	bainianChongjingForced       bool
)

func bainianChongjingCycleSeconds() int64 {
	return int64(BainianChongjingCycle / time.Second)
}

// BainianChongjingCycleStart returns the start of the free-running cycle that contains now.
func BainianChongjingCycleStart(now time.Time) time.Time {
	sec := bainianChongjingCycleSeconds()
	if sec <= 0 {
		return now
	}
	unix := now.Unix()
	start := unix - (unix % sec)
	return time.Unix(start, 0).In(now.Location())
}

func BainianChongjingPhaseAt(now time.Time) BainianChongjingPhase {
	start := BainianChongjingCycleStart(now)
	elapsed := now.Sub(start)
	if elapsed < 0 {
		elapsed = 0
	}
	if elapsed < BainianChongjingWarningLead {
		return BainianChongjingPhaseWarning
	}
	return BainianChongjingPhaseSpawned
}

func BainianChongjingIsActive(now time.Time) bool {
	return BainianChongjingPhaseAt(now) == BainianChongjingPhaseSpawned
}

func BainianChongjingIsWarning(now time.Time) bool {
	return BainianChongjingPhaseAt(now) == BainianChongjingPhaseWarning
}

func BainianChongjingIsAlive(now time.Time) bool {
	bainianChongjingMu.Lock()
	defer bainianChongjingMu.Unlock()
	if bainianChongjingIsForcedLocked(now) {
		return true
	}
	if !BainianChongjingIsActive(now) {
		return false
	}
	if !bainianChongjingKilled {
		return true
	}
	return BainianChongjingCycleBucket(now) != bainianChongjingKilledBucket
}

// ForceBainianChongjingRefreshForDev makes the current natural-cycle boss visible for local dev tools.
// It does not change the captured production cycle or persist across a server restart.
func ForceBainianChongjingRefreshForDev(now time.Time) {
	bainianChongjingMu.Lock()
	defer bainianChongjingMu.Unlock()
	bainianChongjingKilled = false
	bainianChongjingKilledBucket = 0
	bainianChongjingForcedBucket = BainianChongjingCycleBucket(now)
	bainianChongjingForced = true
}

// BainianChongjingIsForcedForDev reports whether the active cycle has already been force-refreshed.
// The announcement loop uses it to avoid repeating a natural spawn notification after a manual refresh.
func BainianChongjingIsForcedForDev(now time.Time) bool {
	bainianChongjingMu.Lock()
	defer bainianChongjingMu.Unlock()
	return bainianChongjingIsForcedLocked(now)
}

func MarkBainianChongjingKilled(now time.Time) {
	bainianChongjingMu.Lock()
	defer bainianChongjingMu.Unlock()
	bainianChongjingKilled = true
	bainianChongjingKilledBucket = BainianChongjingCycleBucket(now)
	bainianChongjingForced = false
	bainianChongjingForcedBucket = 0
}

func ResetBainianChongjingKillStateForTest() {
	bainianChongjingMu.Lock()
	defer bainianChongjingMu.Unlock()
	bainianChongjingKilled = false
	bainianChongjingKilledBucket = 0
	bainianChongjingForced = false
	bainianChongjingForcedBucket = 0
}

func bainianChongjingIsForcedLocked(now time.Time) bool {
	if !bainianChongjingForced {
		return false
	}
	if bainianChongjingForcedBucket == BainianChongjingCycleBucket(now) {
		return true
	}
	bainianChongjingForced = false
	bainianChongjingForcedBucket = 0
	return false
}

func BainianChongjingSpawnForMap(mapID int, now time.Time) (BainianChongjingSpawn, bool) {
	if mapID != BainianChongjingMapID || !BainianChongjingIsAlive(now) {
		return BainianChongjingSpawn{}, false
	}
	return BainianChongjingSpawn{
		MapID:    BainianChongjingMapID,
		Handle:   BainianChongjingHandle,
		Source:   bainianChongjingBossSpawn,
		Militias: append([]BainianChongjingMilitiaSpawn(nil), bainianChongjingMilitias...),
	}, true
}

func IsBainianChongjingHandle(mapID string, handle string) bool {
	handle = strings.TrimSpace(handle)
	if handle == "" || handle != BainianChongjingHandle {
		return false
	}
	if strings.TrimSpace(mapID) == "" {
		return true
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(mapID))
	if err != nil {
		return false
	}
	return parsed == BainianChongjingMapID
}

func IsBainianChongjingHandleAnyMap(handle string) bool {
	return IsBainianChongjingHandle(strconv.Itoa(BainianChongjingMapID), handle)
}

func BainianChongjingEncounterHandles() []string {
	handles := []string{BainianChongjingHandle}
	for _, militia := range bainianChongjingMilitias {
		handles = append(handles, militia.Handle)
	}
	return handles
}

func IsBainianChongjingEncounterHandle(handle string) bool {
	handle = strings.TrimSpace(handle)
	for _, candidate := range BainianChongjingEncounterHandles() {
		if handle == candidate {
			return true
		}
	}
	return false
}

func BainianChongjingWarningNoticeText() string {
	return "十分钟后【百年虫精】将在沼泽9出现!"
}

func BainianChongjingSpawnNoticeText() string {
	return "【百年虫精】已经出现在沼泽9!"
}

func BainianChongjingKillNoticeText(killerNames ...string) string {
	names := make([]string, 0, len(killerNames))
	for _, name := range killerNames {
		name = strings.TrimSpace(name)
		if name != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		names = []string{"冒险者"}
	}
	parts := make([]string, 0, len(names))
	for _, name := range names {
		parts = append(parts, "["+name+"]")
	}
	return strings.Join(parts, "") + "在[沼泽_9]消灭[百年虫精]。"
}

// BainianChongjingCycleBucket returns a stable key for the current free-running cycle.
func BainianChongjingCycleBucket(now time.Time) int64 {
	return BainianChongjingCycleStart(now).Unix()
}
