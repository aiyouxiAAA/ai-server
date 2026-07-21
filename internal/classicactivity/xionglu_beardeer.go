package classicactivity

import (
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	XiongluBeardeerName        = "熊鹿"
	XiongluBeardeerSourceQuery = "monstermap/beardeer.swf"
	XiongluBeardeerSpriteName  = "beardeer"
	XiongluBeardeerMapID       = 203
	XiongluBeardeerMapName     = "雷兽神坛"
	XiongluBeardeerHandle      = "4264636384163425"
	// Quest copy: 雷兽每1个小时就会出现一次. Warning lead mirrors bainian/capture 10-minute form.
	// Exact free-running interval is PARTIAL (single-session battle window only).
	XiongluBeardeerCycle       = 3600 * time.Second
	XiongluBeardeerWarningLead = 10 * time.Minute
)

type XiongluBeardeerPhase string

const (
	XiongluBeardeerPhaseIdle    XiongluBeardeerPhase = "idle"
	XiongluBeardeerPhaseWarning XiongluBeardeerPhase = "warning"
	XiongluBeardeerPhaseSpawned XiongluBeardeerPhase = "spawned"
)

type XiongluBeardeerCompanionSpawn struct {
	Handle   string
	Vocation string
	X        int
	Y        int
}

type XiongluBeardeerSpawn struct {
	MapID       int
	Handle      string
	Source      Point
	Companions  []XiongluBeardeerCompanionSpawn
}

// Capture-backed map entity samples from 20260720_225539_439_auto_move_session.
var xiongluBeardeerBossSpawn = Point{X: 1058, Y: 569}

var xiongluBeardeerCompanions = []XiongluBeardeerCompanionSpawn{
	{Handle: "4266636384163669", Vocation: "术士+", X: 348, Y: 632},
	{Handle: "4268636384164547", Vocation: "术士+", X: 334, Y: 492},
	{Handle: "4270636384165790", Vocation: "术士+", X: 1046, Y: 568},
}

var (
	xiongluBeardeerMu           sync.Mutex
	xiongluBeardeerKilledBucket int64
	xiongluBeardeerKilled       bool
	xiongluBeardeerForcedBucket int64
	xiongluBeardeerForced       bool
)

func xiongluBeardeerCycleSeconds() int64 {
	return int64(XiongluBeardeerCycle / time.Second)
}

// XiongluBeardeerCycleStart returns the start of the free-running cycle that contains now.
func XiongluBeardeerCycleStart(now time.Time) time.Time {
	sec := xiongluBeardeerCycleSeconds()
	if sec <= 0 {
		return now
	}
	unix := now.Unix()
	start := unix - (unix % sec)
	return time.Unix(start, 0).In(now.Location())
}

func XiongluBeardeerPhaseAt(now time.Time) XiongluBeardeerPhase {
	start := XiongluBeardeerCycleStart(now)
	elapsed := now.Sub(start)
	if elapsed < 0 {
		elapsed = 0
	}
	if elapsed < XiongluBeardeerWarningLead {
		return XiongluBeardeerPhaseWarning
	}
	return XiongluBeardeerPhaseSpawned
}

func XiongluBeardeerIsActive(now time.Time) bool {
	return XiongluBeardeerPhaseAt(now) == XiongluBeardeerPhaseSpawned
}

func XiongluBeardeerIsWarning(now time.Time) bool {
	return XiongluBeardeerPhaseAt(now) == XiongluBeardeerPhaseWarning
}

func XiongluBeardeerIsAlive(now time.Time) bool {
	xiongluBeardeerMu.Lock()
	defer xiongluBeardeerMu.Unlock()
	if xiongluBeardeerIsForcedLocked(now) {
		return true
	}
	if !XiongluBeardeerIsActive(now) {
		return false
	}
	if !xiongluBeardeerKilled {
		return true
	}
	return XiongluBeardeerCycleBucket(now) != xiongluBeardeerKilledBucket
}

// ForceXiongluBeardeerRefreshForDev makes the current natural-cycle boss visible for local dev tools.
// It does not change the production cycle or persist across a server restart.
func ForceXiongluBeardeerRefreshForDev(now time.Time) {
	xiongluBeardeerMu.Lock()
	defer xiongluBeardeerMu.Unlock()
	xiongluBeardeerKilled = false
	xiongluBeardeerKilledBucket = 0
	xiongluBeardeerForcedBucket = XiongluBeardeerCycleBucket(now)
	xiongluBeardeerForced = true
}

// XiongluBeardeerIsForcedForDev reports whether the active cycle has already been force-refreshed.
func XiongluBeardeerIsForcedForDev(now time.Time) bool {
	xiongluBeardeerMu.Lock()
	defer xiongluBeardeerMu.Unlock()
	return xiongluBeardeerIsForcedLocked(now)
}

func MarkXiongluBeardeerKilled(now time.Time) {
	xiongluBeardeerMu.Lock()
	defer xiongluBeardeerMu.Unlock()
	xiongluBeardeerKilled = true
	xiongluBeardeerKilledBucket = XiongluBeardeerCycleBucket(now)
	xiongluBeardeerForced = false
	xiongluBeardeerForcedBucket = 0
}

func ResetXiongluBeardeerKillStateForTest() {
	xiongluBeardeerMu.Lock()
	defer xiongluBeardeerMu.Unlock()
	xiongluBeardeerKilled = false
	xiongluBeardeerKilledBucket = 0
	xiongluBeardeerForced = false
	xiongluBeardeerForcedBucket = 0
}

func xiongluBeardeerIsForcedLocked(now time.Time) bool {
	if !xiongluBeardeerForced {
		return false
	}
	if xiongluBeardeerForcedBucket == XiongluBeardeerCycleBucket(now) {
		return true
	}
	xiongluBeardeerForced = false
	xiongluBeardeerForcedBucket = 0
	return false
}

func XiongluBeardeerSpawnForMap(mapID int, now time.Time) (XiongluBeardeerSpawn, bool) {
	if mapID != XiongluBeardeerMapID || !XiongluBeardeerIsAlive(now) {
		return XiongluBeardeerSpawn{}, false
	}
	return XiongluBeardeerSpawn{
		MapID:      XiongluBeardeerMapID,
		Handle:     XiongluBeardeerHandle,
		Source:     xiongluBeardeerBossSpawn,
		Companions: append([]XiongluBeardeerCompanionSpawn(nil), xiongluBeardeerCompanions...),
	}, true
}

func IsXiongluBeardeerHandle(mapID string, handle string) bool {
	handle = strings.TrimSpace(handle)
	if handle == "" || handle != XiongluBeardeerHandle {
		return false
	}
	if strings.TrimSpace(mapID) == "" {
		return true
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(mapID))
	if err != nil {
		return false
	}
	return parsed == XiongluBeardeerMapID
}

func IsXiongluBeardeerHandleAnyMap(handle string) bool {
	return IsXiongluBeardeerHandle(strconv.Itoa(XiongluBeardeerMapID), handle)
}

func XiongluBeardeerEncounterHandles() []string {
	handles := []string{XiongluBeardeerHandle}
	for _, companion := range xiongluBeardeerCompanions {
		handles = append(handles, companion.Handle)
	}
	return handles
}

func IsXiongluBeardeerEncounterHandle(handle string) bool {
	handle = strings.TrimSpace(handle)
	for _, candidate := range XiongluBeardeerEncounterHandles() {
		if handle == candidate {
			return true
		}
	}
	return false
}

// Warning/spawn copy is PARTIAL: battle session has kill notice only; form mirrors bainian + map name.
func XiongluBeardeerWarningNoticeText() string {
	return "十分钟后【熊鹿】将在雷兽神坛出现!"
}

func XiongluBeardeerSpawnNoticeText() string {
	return "【熊鹿】已经出现在雷兽神坛!"
}

func XiongluBeardeerKillNoticeText(killerNames ...string) string {
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
	return strings.Join(parts, "") + "在[雷兽神坛]消灭[熊鹿]。"
}

func XiongluBeardeerCycleBucket(now time.Time) int64 {
	return XiongluBeardeerCycleStart(now).Unix()
}
