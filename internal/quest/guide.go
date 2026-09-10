package quest

import (
	_ "embed"
	"encoding/csv"
	"strconv"
	"strings"
)

//go:embed guide_catalog.csv
var guideCSV string

// Guide is presentation/navigation metadata. Acceptance and rewards stay in the quest service.
type Guide struct {
	QuestID, Phase, ObjectiveID, Action, TargetRole string
	MapID                                           int
	Text, Location, HintID                          string
	Priority                                        int
}

func Guides() []Guide {
	rows, err := csv.NewReader(strings.NewReader(guideCSV)).ReadAll()
	if err != nil {
		panic(err)
	}
	result := []Guide{}
	seen := map[string]bool{}
	for _, row := range rows[1:] {
		if len(row) != 10 {
			panic("quest guide column mismatch")
		}
		q, ok := FindAuthored(row[0])
		mapID, e1 := strconv.Atoi(row[5])
		priority, e2 := strconv.Atoi(row[9])
		key := row[0] + ":" + row[1]
		if !ok || seen[key] || e1 != nil || e2 != nil || mapID != q.MapID || row[2] == "" || row[6] == "" || row[3] != "npc_dialogue" || (row[4] != "start" && row[4] != "finish") || (row[1] != "available" && row[1] != "accepted") {
			panic("invalid quest guide: " + key)
		}
		seen[key] = true
		result = append(result, Guide{row[0], row[1], row[2], row[3], row[4], mapID, row[6], row[7], row[8], priority})
	}
	return result
}
