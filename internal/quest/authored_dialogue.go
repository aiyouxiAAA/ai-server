package quest

import (
	_ "embed"
	"encoding/csv"
	"strings"
)

//go:embed authored_dialogue_catalog.csv
var authoredDialogueCSV string

type AuthoredDialogueLine struct{ Speaker, Text string }

func AuthoredDialogue(id, phase string) []AuthoredDialogueLine {
	rows, err := csv.NewReader(strings.NewReader(authoredDialogueCSV)).ReadAll()
	if err != nil {
		panic(err)
	}
	var result []AuthoredDialogueLine
	for _, row := range rows[1:] {
		if len(row) != 4 || (row[2] != "npc" && row[2] != "player") || row[3] == "" {
			panic("invalid authored dialogue row")
		}
		if row[0] == id && row[1] == phase {
			result = append(result, AuthoredDialogueLine{row[2], row[3]})
		}
	}
	return result
}
