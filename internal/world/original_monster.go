package world

import (
	"ai-server/internal/classicdata"
	"strconv"
	"strings"
)

var originalWildBattleMapIDs = func() map[int]bool {
	maps := map[int]bool{}
	for _, row := range classicdata.MustRows(classicdata.TableMonster) {
		if row["source_kind"] != "wild" || !strings.HasPrefix(row["monster_id"], "original-") {
			continue
		}
		id, err := strconv.Atoi(row["map_id"])
		if err != nil || id <= 0 {
			panic("invalid authored monster map")
		}
		maps[id] = true
	}
	return maps
}()
