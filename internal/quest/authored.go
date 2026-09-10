package quest

import (
	_ "embed"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
)

//go:embed authored_quest_catalog.csv
var authoredCSV string

type AuthoredNPC struct {
	Key, Name string
	X, Y      int
}

func (npc AuthoredNPC) Handle() string { return "authored-" + npc.Key }

type AuthoredQuest struct {
	Info                                                               Info
	MapID                                                              int
	Start, Finish                                                      AuthoredNPC
	OfferDialogue, ReminderDialogue, FinishDialogue, CompletedDialogue string
}

// Original content joins the existing quest provider; captured rows stay intact.
func Authored() []AuthoredQuest {
	rows, err := csv.NewReader(strings.NewReader(authoredCSV)).ReadAll()
	if err != nil {
		panic(err)
	}
	result := make([]AuthoredQuest, 0, len(rows)-1)
	seen := map[string]bool{}
	for _, row := range rows[1:] {
		if len(row) != 19 {
			panic("authored quest column mismatch")
		}
		number := func(i int) int {
			n, err := strconv.Atoi(row[i])
			if err != nil {
				panic(err)
			}
			return n
		}
		if row[0] == "" || seen[row[0]] {
			panic("duplicate or empty authored quest ID")
		}
		seen[row[0]] = true
		entry := AuthoredQuest{
			MapID: number(4), Start: AuthoredNPC{row[5], row[6], number(7), number(8)},
			Finish:        AuthoredNPC{row[9], row[10], number(11), number(12)},
			OfferDialogue: row[15], ReminderDialogue: row[16], FinishDialogue: row[17], CompletedDialogue: row[18],
			Info: Info{ID: row[0], Title: row[1], Level: number(2), Type: row[3],
				Description: fmt.Sprintf("<ml>%s<br/>[g]=经验%d", row[13], number(14)),
				State:       "<over>向" + row[10] + "报到", Reward: Reward{Experience: number(14)}},
		}
		entry.Info.QuestStateHandle = entry.Start.Handle()
		entry.Info.RewardEntries = BuildRewardEntries(RewardEntrySourceQuest, entry.Info.ID, entry.Info.Reward)
		if entry.MapID <= 0 || entry.Info.Level < 1 || entry.Info.Reward.Experience < 0 || entry.Start.Key == entry.Finish.Key {
			panic("invalid authored quest design")
		}
		result = append(result, entry)
	}
	return result
}

func FindAuthored(id string) (AuthoredQuest, bool) {
	for _, q := range Authored() {
		if q.Info.ID == id {
			return q, true
		}
	}
	return AuthoredQuest{}, false
}

func AuthoredForNPC(handle string) (AuthoredQuest, bool) {
	for _, q := range Authored() {
		if q.Start.Handle() == handle || q.Finish.Handle() == handle {
			return q, true
		}
	}
	return AuthoredQuest{}, false
}
