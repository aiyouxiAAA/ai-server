package battle

import (
	_ "embed"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
)

//go:embed config/original-monster-behavior.csv
var originalMonsterBehaviorCSV string

type originalMonsterBehavior struct {
	DisplayURL, NormalName, SkillCommand, SkillName, SkillLabel string
	OpeningAttacks, SkillInterval                               int
	DamageMultiplier                                            float64
}

var originalMonsterBehaviors = loadOriginalMonsterBehaviors()

func loadOriginalMonsterBehaviors() map[string]originalMonsterBehavior {
	rows, err := csv.NewReader(strings.NewReader(originalMonsterBehaviorCSV)).ReadAll()
	if err != nil || len(rows) < 2 {
		panic("invalid original monster behavior CSV")
	}
	header := battleConfigHeader(rows[0])
	result := map[string]originalMonsterBehavior{}
	for i, row := range rows[1:] {
		value := func(key string) string { return requiredBattleConfigString(row, header, key, i) }
		multiplier, err := strconv.ParseFloat(value("damage_multiplier"), 64)
		behavior := originalMonsterBehavior{DisplayURL: value("display_url"), NormalName: value("normal_name"),
			SkillCommand: value("skill_command"), SkillName: value("skill_name"), SkillLabel: value("skill_label"),
			OpeningAttacks: requiredBattleConfigInt(row, header, "opening_attacks", i),
			SkillInterval:  requiredBattleConfigInt(row, header, "skill_interval", i), DamageMultiplier: multiplier}
		if err != nil || multiplier <= 0 || behavior.OpeningAttacks < 1 || behavior.SkillInterval < 2 {
			panic(fmt.Sprintf("invalid original monster behavior row %d", i+2))
		}
		if _, exists := result[behavior.DisplayURL]; exists {
			panic("duplicate original monster behavior: " + behavior.DisplayURL)
		}
		result[behavior.DisplayURL] = behavior
	}
	return result
}

func originalMonsterProfile(actor *CellInfoPush, commandID string) (commandProfile, bool) {
	if actor == nil || actor.Camp != CampEnemy {
		return commandProfile{}, false
	}
	behavior, ok := originalMonsterBehaviors[actor.DisplayURL]
	if !ok {
		return commandProfile{}, false
	}
	profile := commandProfile{ActionName: behavior.NormalName, SourceType: "oneE", SourceActionLabel: "nomalAtk",
		DamageMultiplier: 1, CanDodge: true, CanFat: false, DefenseType: "physical"}
	if commandID == behavior.SkillCommand {
		profile.ActionName, profile.SourceActionLabel = behavior.SkillName, behavior.SkillLabel
		profile.DamageMultiplier = behavior.DamageMultiplier
	} else if commandID != CommandEnemyAttack {
		panic("unregistered original monster command: " + commandID)
	}
	return profile, true
}

func (runtime *Runtime) originalMonsterCommand(enemy *CellInfoPush) (string, bool) {
	if enemy == nil {
		return "", false
	}
	behavior, ok := originalMonsterBehaviors[enemy.DisplayURL]
	if !ok {
		return "", false
	}
	if runtime.originalMonsterTurns == nil {
		runtime.originalMonsterTurns = map[string]int{}
	}
	runtime.originalMonsterTurns[enemy.Handle]++
	turn := runtime.originalMonsterTurns[enemy.Handle]
	// Slot stagger is stable even after the first spider dies; skipped turns never call this method.
	slot := 0
	for _, cell := range runtime.Cells {
		if cell.Handle == enemy.Handle {
			break
		}
		if cell.Camp == CampEnemy && cell.DisplayURL == enemy.DisplayURL {
			slot++
		}
	}
	firstSkill := behavior.OpeningAttacks + 1 + slot
	if turn >= firstSkill && (turn-firstSkill)%behavior.SkillInterval == 0 {
		return behavior.SkillCommand, true
	}
	return CommandEnemyAttack, true
}

// Authored drops are intentionally distinct from captured "confirmed" rewards.
// Each actual enemy slot earns one configured reward, including same-species duplicates.
func (runtime *Runtime) originalMonsterRewards() (int, []string, bool) {
	exp, count := 0, 0
	items := []string{}
	for _, cell := range runtime.Cells {
		if cell.Camp != CampEnemy {
			continue
		}
		if _, ok := originalMonsterBehaviors[cell.DisplayURL]; !ok {
			return 0, nil, false
		}
		found := false
		for _, config := range sourceEnemyConfigsForMap(runtime.MapID) {
			if config.Cell.DisplayURL != cell.DisplayURL {
				continue
			}
			reward, ok := sourceBattleRewardConfigForExactEncounter(runtime.MapID, config.Cell.Handle)
			if !ok || reward.Status != "authored" {
				panic("missing authored monster reward: " + config.Cell.Handle)
			}
			exp += reward.ExpDelta
			items = append(items, reward.Items...)
			found = true
			break
		}
		if !found {
			panic("missing authored monster map: " + runtime.MapID)
		}
		count++
	}
	return exp, mergeSourceBattleRewardItemStacks(items), count > 0
}
