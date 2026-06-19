package classicactivity

import (
	"strconv"
	"strings"
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

type pointCouponThiefMapConfig struct {
	mapID      int
	spawnPaths []pointCouponThiefSpawnPath
}

type pointCouponThiefSpawnPath struct {
	source    Point
	moveSpeed int
	moveAngle float64
	moveMode  int
}

var pointCouponThiefMaps = []pointCouponThiefMapConfig{
	{
		mapID: 84,
		spawnPaths: []pointCouponThiefSpawnPath{
			{source: Point{X: 1106, Y: 523}, moveSpeed: 215, moveAngle: 215.90004964021622, moveMode: 2},
		},
	},
	{
		mapID: 114,
		spawnPaths: []pointCouponThiefSpawnPath{
			{source: Point{X: 2550, Y: 491}, moveSpeed: 184, moveAngle: 184.3987053549955, moveMode: 2},
		},
	},
	{
		mapID: 115,
		spawnPaths: []pointCouponThiefSpawnPath{
			{source: Point{X: 5057, Y: 524}},
		},
	},
}

func PointCouponThiefSpawnForMap(mapID int, now time.Time) (PointCouponThiefSpawn, bool) {
	config, ok := pointCouponThiefConfigForMap(mapID)
	if !ok || len(config.spawnPaths) == 0 {
		return PointCouponThiefSpawn{}, false
	}
	hour := now.Local().Truncate(time.Hour).Unix()
	pathIndex := pointCouponThiefPathIndex(mapID, hour, len(config.spawnPaths))
	path := config.spawnPaths[pathIndex]
	return PointCouponThiefSpawn{
		MapID:     mapID,
		Handle:    PointCouponThiefHandle(mapID, hour, pathIndex),
		Source:    path.source,
		MoveSpeed: path.moveSpeed,
		MoveAngle: path.moveAngle,
		MoveMode:  path.moveMode,
	}, true
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
	if _, ok := pointCouponThiefConfigForMap(parsedMapID); !ok {
		return false
	}
	if _, err := strconv.ParseInt(parts[4], 10, 64); err != nil {
		return false
	}
	pathIndex, err := strconv.Atoi(parts[5])
	if err != nil || pathIndex < 0 {
		return false
	}
	config, _ := pointCouponThiefConfigForMap(parsedMapID)
	return pathIndex < len(config.spawnPaths)
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

func pointCouponThiefConfigForMap(mapID int) (pointCouponThiefMapConfig, bool) {
	for _, config := range pointCouponThiefMaps {
		if config.mapID == mapID {
			return config, true
		}
	}
	return pointCouponThiefMapConfig{}, false
}

func pointCouponThiefPathIndex(mapID int, hourUnix int64, pathCount int) int {
	if pathCount <= 1 {
		return 0
	}
	seed := int64(mapID*1103515245) + hourUnix/3600
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
