package battle

import (
	"strconv"
	"strings"

	"ai-server/internal/classicdata"
)

var sourceBattleSkillRowsByLabel = mustLoadSourceBattleSkillRowsByLabel()

func mustLoadSourceBattleSkillRowsByLabel() map[string]map[string]string {
	rows := classicdata.MustRows(classicdata.TableSkill)
	result := make(map[string]map[string]string, len(rows))
	for _, row := range rows {
		label := strings.TrimSpace(row["label"])
		if label == "" {
			continue
		}
		result[label] = row
	}
	return result
}

func sourceBattleSkillRowByLabel(label string) (map[string]string, bool) {
	row, ok := sourceBattleSkillRowsByLabel[strings.TrimSpace(label)]
	return row, ok
}

func sourceBattleSkillCommandIDFromConfig(label string) string {
	row, ok := sourceBattleSkillRowByLabel(label)
	if !ok {
		return ""
	}
	return strings.TrimSpace(row["command_id"])
}

func sourceBattleSkillSourceTypeFromConfig(label string) string {
	row, ok := sourceBattleSkillRowByLabel(label)
	if !ok {
		return ""
	}
	return strings.TrimSpace(row["source_type"])
}

func sourceBattleSkillActionNameFromConfig(label string) string {
	row, ok := sourceBattleSkillRowByLabel(label)
	if !ok {
		return ""
	}
	return strings.TrimSpace(row["action_name"])
}

func sourceBattleSkillActionLabelFromConfig(label string) string {
	row, ok := sourceBattleSkillRowByLabel(label)
	if !ok {
		return ""
	}
	return strings.TrimSpace(row["source_action_label"])
}

func sourceBattleSkillTargetFromConfig(label string) string {
	row, ok := sourceBattleSkillRowByLabel(label)
	if !ok {
		return ""
	}
	return strings.TrimSpace(row["target"])
}

func sourceBattleSkillProfileFromConfig(label string) (commandProfile, bool) {
	row, ok := sourceBattleSkillRowByLabel(label)
	if !ok {
		return commandProfile{}, false
	}
	profile := commandProfile{
		ActionName:        strings.TrimSpace(row["action_name"]),
		SourceType:        strings.TrimSpace(row["source_type"]),
		SourceActionLabel: strings.TrimSpace(row["source_action_label"]),
		DamageMultiplier:  classicDataFloat(row["damage_multiplier"]),
		MPCost:            classicDataInt(row["mp_cost"]),
		CanDodge:          true,
		CanFat:            true,
		LifeStealChance:   classicDataInt(row["hp_drain_chance_percent"]),
		LifeStealRatio:    classicDataFloat(row["hp_drain_multiplier"]),
		DirectAttackBonus: classicDataFloat(row["direct_attack_bonus_multiplier"]),
	}
	if profile.ActionName == "" {
		profile.ActionName = strings.TrimSpace(row["label"])
	}
	return profile, true
}

func classicDataInt(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0
	}
	return value
}

func classicDataFloat(raw string) float64 {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return 0
	}
	return value
}
