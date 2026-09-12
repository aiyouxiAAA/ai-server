package quest

import (
	_ "embed"
	"encoding/csv"
	"strconv"
	"strings"
)

//go:embed authored_quest_rules.csv
var authoredRulesCSV string

// One objective per non-abandonable quest. Gifts are granted on acceptance.
type AuthoredRule struct {
	Previous, Kind, Target, AcceptItem string
	Count, TargetMap                   int
}

func authoredRules() map[string]AuthoredRule {
	rows, err := csv.NewReader(strings.NewReader(authoredRulesCSV)).ReadAll()
	if err != nil {
		panic(err)
	}
	result := map[string]AuthoredRule{}
	for _, r := range rows[1:] {
		if len(r) != 7 {
			panic("authored rule column mismatch")
		}
		count, e1 := strconv.Atoi(r[4])
		mapID, e2 := strconv.Atoi(r[5])
		if r[0] == "" || result[r[0]].Kind != "" || e1 != nil || e2 != nil || count < 0 || mapID < 0 {
			panic("invalid authored rule: " + r[0])
		}
		switch r[2] {
		case "visit", "recover":
			if count != 0 || r[3] != "" || mapID != 0 {
				panic("invalid visit/recovery objective")
			}
		case "equip":
			if r[3] == "" || count != 1 || mapID != 0 {
				panic("invalid equipment objective")
			}
		case "kill":
			if r[3] == "" || count < 1 || mapID < 1 {
				panic("invalid kill objective")
			}
		default:
			panic("unsupported authored objective")
		}
		result[r[0]] = AuthoredRule{Previous: r[1], Kind: r[2], Target: r[3], Count: count, TargetMap: mapID, AcceptItem: r[6]}
	}
	return result
}

func validateAuthoredRules(quests []AuthoredQuest, rules map[string]AuthoredRule) {
	byID := map[string]AuthoredQuest{}
	titles := map[string]bool{}
	npcs := map[string]AuthoredNPC{}
	for _, q := range quests {
		if titles[q.Info.Title] {
			panic("duplicate authored title")
		}
		titles[q.Info.Title] = true
		byID[q.Info.ID] = q
		for _, npc := range []AuthoredNPC{q.Start, q.Finish} {
			if old, exists := npcs[npc.Handle()]; exists && old != npc {
				panic("inconsistent authored NPC")
			}
			npcs[npc.Handle()] = npc
		}
	}
	if len(rules) != len(quests) {
		panic("orphan authored rule")
	}
	for _, q := range quests {
		seen := map[string]bool{q.Info.ID: true}
		for id := q.Rule.Previous; id != ""; {
			previous, ok := byID[id]
			if !ok || seen[id] {
				panic("missing or cyclic authored prerequisite")
			}
			seen[id] = true
			id = previous.Rule.Previous
		}
	}
}
