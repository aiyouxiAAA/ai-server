package battle

import (
	"embed"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed config/playback-timing.csv
var playbackFiles embed.FS

type playbackTiming struct{ duration, travel, reaction time.Duration }

var playbackTimings = loadPlaybackTimings()

func loadPlaybackTimings() map[string]playbackTiming {
	data, err := playbackFiles.ReadFile("config/playback-timing.csv")
	if err != nil {
		panic(err)
	}
	rows, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
	if err != nil {
		panic(err)
	}
	result := map[string]playbackTiming{}
	for _, row := range rows[1:] {
		if len(row) != 6 {
			panic("invalid playback timing row")
		}
		var ms [3]time.Duration
		for i := range ms {
			n, err := strconv.ParseInt(row[i+2], 10, 64)
			if err != nil || n < 0 {
				panic("invalid playback duration")
			}
			ms[i] = time.Duration(n) * time.Millisecond
		}
		key := row[0] + ":" + row[1]
		if _, ok := result[key]; ok || ms[0] <= 0 {
			panic("duplicate/empty playback timing: " + key)
		}
		result[key] = playbackTiming{ms[0], ms[1], ms[2]}
	}
	return result
}

// Wire serializes the complete authenticated request, including consumption and settlement,
// across teammates sharing this runtime. Domain formula tests do not simulate transport time.
type PlaybackClock struct {
	Wire     sync.Mutex
	ID       int
	Consumed int
	Deadline time.Time
}

func playbackActorKey(cell CellInfoPush) string {
	if cell.ProfessionID != "" {
		return "profession/" + cell.ProfessionID
	}
	// Match ClassicBattleMonsterProvider's source wrapper mapping.
	return strings.Replace(cell.DisplayURL, "monstermap/", "monsterbox/", 1)
}

func (runtime *Runtime) ValidatePlaybackCatalog() error {
	for _, cell := range runtime.Cells {
		if _, ok := playbackTimings[playbackActorKey(cell)+":nomalAtk"]; !ok {
			return fmt.Errorf("playback timing missing: %s", playbackActorKey(cell))
		}
	}
	return nil
}

// ProtectActions assigns a server batch and one cumulative timeline. Concurrent teammate
// actions cannot each spend the same elapsed second. Client timestamps are never read.
func (runtime *Runtime) ProtectActions(actions []ActionPush, now time.Time) error {
	if len(actions) == 0 {
		return nil
	}
	var duration time.Duration
	for _, action := range actions {
		actor := runtime.cellByHandle(action.ActorHandle)
		if actor == nil {
			return fmt.Errorf("playback actor missing")
		}
		timing, ok := playbackTimings[playbackActorKey(*actor)+":"+action.SourceActionLabel]
		if !ok {
			return fmt.Errorf("playback timing missing: %s/%s", playbackActorKey(*actor), action.SourceActionLabel)
		}
		duration += timing.duration
		if action.ActorHandle != action.TargetHandle {
			duration += timing.travel
			targets := []string{action.TargetHandle}
			for _, entry := range action.TargetActionResults {
				targets = append(targets, entry.Handle)
			}
			// Client target reactions run concurrently, so the longest target governs the budget.
			var reaction time.Duration
			for _, handle := range targets {
				target := runtime.cellByHandle(handle)
				if target == nil || target.Handle == actor.Handle {
					continue
				}
				targetTiming, ok := playbackTimings[playbackActorKey(*target)+":nomalAtk"]
				if !ok {
					return fmt.Errorf("playback target missing")
				}
				if targetTiming.reaction > reaction {
					reaction = targetTiming.reaction
				}
			}
			duration += reaction
		}
	}
	clock := &runtime.Playback
	if clock.Deadline.Before(now) {
		clock.Deadline = now
	}
	clock.Deadline = clock.Deadline.Add(duration)
	clock.ID++
	remaining := (clock.Deadline.Sub(now) + time.Millisecond - 1) / time.Millisecond
	for i := range actions {
		actions[i].PlaybackID = clock.ID
		actions[i].PlaybackRemainingMS = int64(remaining)
	}
	return nil
}
