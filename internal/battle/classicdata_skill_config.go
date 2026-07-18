package battle

import (
	"strconv"
	"strings"

	"ai-server/internal/classicdata"
)

var sourceBattleSkillRowsByLabel = mustLoadSourceBattleSkillRowsByLabel()
var sourceBattleSkillLevelRowsByKey = mustLoadSourceBattleSkillLevelRowsByKey()

func mustLoadSourceBattleSkillRowsByLabel() map[string]map[string]string {
	rows := classicdata.MustRows(classicdata.TableSkill)
	result := make(map[string]map[string]string, len(rows))
	for _, row := range rows {
		label := strings.TrimSpace(row["label"])
		skillID := strings.TrimSpace(row["skill_id"])
		if label == "" || skillID == "" {
			panic("classic skill table row requires skill_id and label")
		}
		if _, exists := result[label]; exists {
			panic("classic skill table has duplicate label: " + label)
		}
		result[label] = row
	}
	return result
}

func mustLoadSourceBattleSkillLevelRowsByKey() map[string]map[string]string {
	skillIDs := make(map[string]bool, len(sourceBattleSkillRowsByLabel))
	for _, row := range sourceBattleSkillRowsByLabel {
		skillIDs[strings.TrimSpace(row["skill_id"])] = true
	}
	rows := classicdata.MustRows(classicdata.TableSkillLevel)
	result := make(map[string]map[string]string, len(rows))
	for _, row := range rows {
		skillID := strings.TrimSpace(row["skill_id"])
		level := classicDataInt(row["level"])
		if skillID == "" || level <= 0 {
			panic("classic skill level table row requires skill_id and positive level")
		}
		if !skillIDs[skillID] {
			panic("classic skill level references unknown skill_id: " + skillID)
		}
		key := sourceBattleSkillLevelKey(skillID, level)
		if _, exists := result[key]; exists {
			panic("classic skill level table has duplicate skill_id and level: " + key)
		}
		result[key] = row
	}
	return result
}

func sourceBattleSkillLevelKey(skillID string, level int) string {
	return strings.TrimSpace(skillID) + ":" + strconv.Itoa(level)
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
	return strings.TrimSpace(row["skill_id"])
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

func sourceBattleSkillActionLabelFromConfig(label string, level int) string {
	row, ok := sourceBattleSkillRowByLabel(label)
	if !ok {
		return ""
	}
	levelRow, ok := sourceBattleSkillLevelRowsByKey[sourceBattleSkillLevelKey(row["skill_id"], level)]
	if !ok {
		return ""
	}
	return strings.TrimSpace(levelRow["source_action_label"])
}

func sourceBattleSkillTargetFromConfig(label string) string {
	row, ok := sourceBattleSkillRowByLabel(label)
	if !ok {
		return ""
	}
	return strings.TrimSpace(row["target"])
}

func sourceBattleSkillProfileFromConfig(label string, level int) (commandProfile, bool) {
	row, ok := sourceBattleSkillRowByLabel(label)
	if !ok {
		return commandProfile{}, false
	}
	profile := commandProfile{
		ActionName: strings.TrimSpace(row["action_name"]),
		SourceType: strings.TrimSpace(row["source_type"]),
		CanDodge:   true,
		CanFat:     true,
	}
	if levelRow, exists := sourceBattleSkillLevelRowsByKey[sourceBattleSkillLevelKey(row["skill_id"], level)]; exists {
		profile.SourceActionLabel = strings.TrimSpace(levelRow["source_action_label"])
		profile.DamageMultiplier = classicDataFloat(levelRow["damage_multiplier"])
		profile.MPCost = classicDataInt(levelRow["mp_cost"])
		profile.LifeStealChance = classicDataInt(levelRow["hp_drain_chance_percent"])
		profile.LifeStealRatio = classicDataFloat(levelRow["hp_drain_multiplier"])
		profile.DirectAttackBonus = classicDataFloat(levelRow["direct_attack_bonus_multiplier"])
	}
	if profile.ActionName == "" {
		profile.ActionName = strings.TrimSpace(row["label"])
	}
	return profile, true
}

func sourceBattleSkillLevelExists(label string, level int) bool {
	row, ok := sourceBattleSkillRowByLabel(label)
	if !ok || level <= 0 {
		return false
	}
	_, ok = sourceBattleSkillLevelRowsByKey[sourceBattleSkillLevelKey(row["skill_id"], level)]
	return ok
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
