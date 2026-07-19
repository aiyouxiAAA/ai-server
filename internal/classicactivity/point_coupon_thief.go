package classicactivity

import (
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	PointCouponThiefName         = "点券盗贼"
	PointCouponThiefSourceQuery  = "monstermap/militia.swf"
	PointCouponThiefSpriteName   = "militia"
	PointCouponThiefHandlePrefix = "point_coupon_thief_"
)

type Point struct {
	X int
	Y int
}

type PointCouponThiefSpawn struct {
	MapID     int
	Handle    string
	Source    Point
	MoveSpeed int
	MoveAngle float64
	MoveMode  int
}

// PointCouponThiefRefresh describes one complete activity replacement. Previous
// entries must be removed from online maps before Current entries are pushed.
type PointCouponThiefRefresh struct {
	Previous []PointCouponThiefSpawn
	Current  []PointCouponThiefSpawn
}

type pointCouponThiefSpawnPath struct {
	mapID     int
	source    Point
	moveSpeed int
	moveAngle float64
	moveMode  int
}

type pointCouponThiefRegionConfig struct {
	regionID   int
	spawnPaths []pointCouponThiefSpawnPath
}

// Each region selects one captured spawn path on every refresh. The activity
// therefore creates exactly three thieves: one in Tree Sea, Bamboo Forest,
// and Wofogu respectively.
var pointCouponThiefRegions = []pointCouponThiefRegionConfig{
	{
		regionID: 21,
		spawnPaths: []pointCouponThiefSpawnPath{
			{mapID: 21, source: Point{X: 752, Y: 522}},
			{mapID: 29, source: Point{X: 1012, Y: 614}},
		},
	},
	{
		regionID: 84,
		spawnPaths: []pointCouponThiefSpawnPath{
			{mapID: 84, source: Point{X: 1106, Y: 523}, moveSpeed: 215, moveAngle: 215.90004964021622, moveMode: 2},
			{mapID: 92, source: Point{X: 600, Y: 529}},
			{mapID: 100, source: Point{X: 1280, Y: 532}},
		},
	},
	{
		regionID: 114,
		spawnPaths: []pointCouponThiefSpawnPath{
			{mapID: 114, source: Point{X: 2550, Y: 491}, moveSpeed: 184, moveAngle: 184.3987053549955, moveMode: 2},
			{mapID: 114, source: Point{X: 1415, Y: 548}},
			{mapID: 115, source: Point{X: 5057, Y: 524}},
			{mapID: 117, source: Point{X: 935, Y: 467}},
		},
	},
}

var (
	pointCouponThiefRefreshMu        sync.Mutex
	pointCouponThiefRefreshBucket    int64
	pointCouponThiefRefreshSpawnHour int64
	pointCouponThiefDevRefreshSerial int64
)

func PointCouponThiefSpawnForMap(mapID int, now time.Time) (PointCouponThiefSpawn, bool) {
	spawnHour := pointCouponThiefCurrentSpawnHour(now)
	return pointCouponThiefSpawnForMapAtHour(mapID, spawnHour)
}

// AdvancePointCouponThiefRefresh moves the hourly event to its natural current
// cycle and returns both the stale and current source roles for map push.
func AdvancePointCouponThiefRefresh(now time.Time) PointCouponThiefRefresh {
	bucket := pointCouponThiefHourBucket(now)
	previousBucket := bucket - int64(time.Hour/time.Second)

	pointCouponThiefRefreshMu.Lock()
	previousSpawnHour := previousBucket
	if pointCouponThiefRefreshBucket == previousBucket && pointCouponThiefRefreshSpawnHour != 0 {
		previousSpawnHour = pointCouponThiefRefreshSpawnHour
	}
	pointCouponThiefRefreshBucket = bucket
	pointCouponThiefRefreshSpawnHour = bucket
	pointCouponThiefRefreshMu.Unlock()

	return PointCouponThiefRefresh{
		Previous: pointCouponThiefSpawnsAtHour(previousSpawnHour),
		Current:  pointCouponThiefSpawnsAtHour(bucket),
	}
}

// ForcePointCouponThiefRefreshForDev creates a distinct current-cycle source
// handle for the local dev tool. The original hourly cycle remains unchanged
// and resumes automatically at the next natural hour boundary.
func ForcePointCouponThiefRefreshForDev(now time.Time) PointCouponThiefRefresh {
	bucket := pointCouponThiefHourBucket(now)

	pointCouponThiefRefreshMu.Lock()
	previousSpawnHour := bucket
	if pointCouponThiefRefreshBucket == bucket && pointCouponThiefRefreshSpawnHour != 0 {
		previousSpawnHour = pointCouponThiefRefreshSpawnHour
	}
	pointCouponThiefDevRefreshSerial++
	forcedSpawnHour := int64(4102444800) + pointCouponThiefDevRefreshSerial
	pointCouponThiefRefreshBucket = bucket
	pointCouponThiefRefreshSpawnHour = forcedSpawnHour
	pointCouponThiefRefreshMu.Unlock()

	return PointCouponThiefRefresh{
		Previous: pointCouponThiefSpawnsAtHour(previousSpawnHour),
		Current:  pointCouponThiefSpawnsAtHour(forcedSpawnHour),
	}
}

// ResetPointCouponThiefRefreshStateForTest clears the process-local dev
// refresh state so tests do not leak a forced cycle into another contract.
func ResetPointCouponThiefRefreshStateForTest() {
	pointCouponThiefRefreshMu.Lock()
	defer pointCouponThiefRefreshMu.Unlock()
	pointCouponThiefRefreshBucket = 0
	pointCouponThiefRefreshSpawnHour = 0
	pointCouponThiefDevRefreshSerial = 0
}

func pointCouponThiefCurrentSpawnHour(now time.Time) int64 {
	bucket := pointCouponThiefHourBucket(now)
	pointCouponThiefRefreshMu.Lock()
	defer pointCouponThiefRefreshMu.Unlock()
	if pointCouponThiefRefreshBucket == bucket && pointCouponThiefRefreshSpawnHour != 0 {
		return pointCouponThiefRefreshSpawnHour
	}
	return bucket
}

func pointCouponThiefHourBucket(now time.Time) int64 {
	return now.Local().Truncate(time.Hour).Unix()
}

func pointCouponThiefSpawnsAtHour(spawnHour int64) []PointCouponThiefSpawn {
	spawns := make([]PointCouponThiefSpawn, 0, len(pointCouponThiefRegions))
	for _, config := range pointCouponThiefRegions {
		if len(config.spawnPaths) == 0 {
			continue
		}
		pathIndex := pointCouponThiefPathIndex(config.regionID, spawnHour, len(config.spawnPaths))
		path := config.spawnPaths[pathIndex]
		spawns = append(spawns, PointCouponThiefSpawn{
			MapID:     path.mapID,
			Handle:    PointCouponThiefHandle(path.mapID, spawnHour, pointCouponThiefMapPathIndex(config.spawnPaths, pathIndex)),
			Source:    path.source,
			MoveSpeed: path.moveSpeed,
			MoveAngle: path.moveAngle,
			MoveMode:  path.moveMode,
		})
	}
	return spawns
}

func pointCouponThiefSpawnForMapAtHour(mapID int, spawnHour int64) (PointCouponThiefSpawn, bool) {
	for _, spawn := range pointCouponThiefSpawnsAtHour(spawnHour) {
		if spawn.MapID == mapID {
			return spawn, true
		}
	}
	return PointCouponThiefSpawn{}, false
}

func PointCouponThiefHandle(mapID int, hourUnix int64, pathIndex int) string {
	return PointCouponThiefHandlePrefix +
		strconv.Itoa(mapID) + "_" +
		strconv.FormatInt(hourUnix, 10) + "_" +
		strconv.Itoa(maxInt(0, pathIndex))
}

func IsPointCouponThiefHandle(mapID string, handle string) bool {
	handle = strings.TrimSpace(handle)
	if handle == "" || !strings.HasPrefix(handle, PointCouponThiefHandlePrefix) {
		return false
	}
	parts := strings.Split(handle, "_")
	if len(parts) != 6 {
		return false
	}
	handleMapID := strings.TrimSpace(parts[3])
	if handleMapID == "" || handleMapID != strings.TrimSpace(mapID) {
		return false
	}
	parsedMapID, err := strconv.Atoi(handleMapID)
	if err != nil {
		return false
	}
	if _, err := strconv.ParseInt(parts[4], 10, 64); err != nil {
		return false
	}
	pathIndex, err := strconv.Atoi(parts[5])
	if err != nil || pathIndex < 0 {
		return false
	}
	return pointCouponThiefHasMapPathIndex(parsedMapID, pathIndex)
}

func IsPointCouponThiefHandleAnyMap(handle string) bool {
	handle = strings.TrimSpace(handle)
	if handle == "" || !strings.HasPrefix(handle, PointCouponThiefHandlePrefix) {
		return false
	}
	parts := strings.Split(handle, "_")
	if len(parts) != 6 {
		return false
	}
	return IsPointCouponThiefHandle(parts[3], handle)
}

// IsCurrentPointCouponThiefHandle confirms that a syntactically valid activity
// handle is the one selected for its region in the current refresh cycle.
func IsCurrentPointCouponThiefHandle(mapID string, handle string, now time.Time) bool {
	mapID = strings.TrimSpace(mapID)
	handle = strings.TrimSpace(handle)
	if !IsPointCouponThiefHandle(mapID, handle) {
		return false
	}
	parsedMapID, err := strconv.Atoi(mapID)
	if err != nil {
		return false
	}
	spawn, ok := PointCouponThiefSpawnForMap(parsedMapID, now)
	return ok && spawn.Handle == handle
}

func pointCouponThiefMapPathIndex(paths []pointCouponThiefSpawnPath, selectedIndex int) int {
	if selectedIndex < 0 || selectedIndex >= len(paths) {
		return 0
	}
	pathIndex := 0
	for index := 0; index < selectedIndex; index++ {
		if paths[index].mapID == paths[selectedIndex].mapID {
			pathIndex++
		}
	}
	return pathIndex
}

func pointCouponThiefHasMapPathIndex(mapID int, pathIndex int) bool {
	for _, region := range pointCouponThiefRegions {
		for selectedIndex, path := range region.spawnPaths {
			if path.mapID == mapID && pointCouponThiefMapPathIndex(region.spawnPaths, selectedIndex) == pathIndex {
				return true
			}
		}
	}
	return false
}

func pointCouponThiefPathIndex(mapID int, hourUnix int64, pathCount int) int {
	if pathCount <= 1 {
		return 0
	}
	cycle := hourUnix / int64(time.Hour/time.Second)
	// Natural refreshes use an exact hour bucket. Forced local refreshes use a
	// distinct sub-hour serial so each click also selects a new path.
	cycle += hourUnix % int64(time.Hour/time.Second)
	seed := int64(mapID*1103515245) + cycle
	if seed < 0 {
		seed = -seed
	}
	return int(seed % int64(pathCount))
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}
