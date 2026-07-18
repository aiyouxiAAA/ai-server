package classicdata

import (
	"fmt"
	"strings"
)

// Rows returns a detached copy of a generated classic data table.
func Rows(name string) ([]map[string]string, error) {
	table, err := loadTableCached(name)
	if err != nil {
		return nil, err
	}
	return cloneRows(table.Rows), nil
}

func MustRows(name string) []map[string]string {
	rows, err := Rows(name)
	if err != nil {
		panic(err)
	}
	return rows
}

func FindRow(name string, column string, value string) (map[string]string, bool, error) {
	rows, err := FindRows(name, column, value)
	if err != nil {
		return nil, false, err
	}
	if len(rows) == 0 {
		return nil, false, nil
	}
	return rows[0], true, nil
}

func FindRows(name string, column string, value string) ([]map[string]string, error) {
	column = strings.TrimSpace(column)
	if column == "" {
		return nil, fmt.Errorf("classic data lookup column is empty")
	}

	table, err := LoadTable(name)
	if err != nil {
		return nil, err
	}

	matches := make([]map[string]string, 0)
	for _, row := range table.Rows {
		if row[column] == value {
			matches = append(matches, cloneRow(row))
		}
	}
	return matches, nil
}

func FindItemByName(name string) (map[string]string, bool, error) {
	return FindRow(TableItem, "name", name)
}

func FindFashionAppearanceRowsByName(name string) ([]map[string]string, error) {
	return FindRows(TableFashionAppearance, "fashion_name", strings.TrimSpace(name))
}

func FindFashionAppearanceByNameAndSex(name string, sex string) (map[string]string, bool, error) {
	rows, err := FindFashionAppearanceRowsByName(name)
	if err != nil {
		return nil, false, err
	}
	sex = strings.TrimSpace(sex)
	for _, row := range rows {
		if row["sex"] == sex {
			return row, true, nil
		}
	}
	return nil, false, nil
}

func FindSkillByLabel(label string) (map[string]string, bool, error) {
	return FindRow(TableSkill, "label", label)
}

func FindSkillLevelRowsBySkillID(skillID string) ([]map[string]string, error) {
	return FindRows(TableSkillLevel, "skill_id", skillID)
}

func FindMonsterSkillByID(id string) (map[string]string, bool, error) {
	return FindRow(TableMonsterSkill, "monster_skill_id", id)
}

func FindMonsterSkillRowsByDisplayURL(displayURL string) ([]map[string]string, error) {
	return FindRows(TableMonsterSkill, "display_url", displayURL)
}

func FindProfessionByID(id string) (map[string]string, bool, error) {
	return FindRow(TableProfession, "profession_id", id)
}

func FindBuffByID(id string) (map[string]string, bool, error) {
	return FindRow(TableBuff, "buff_id", id)
}

func FindMonsterByID(id string) (map[string]string, bool, error) {
	return FindRow(TableMonster, "monster_id", id)
}

func FindAttributeByID(id string) (map[string]string, bool, error) {
	return FindRow(TableAttribute, "attribute_id", id)
}

func FindEffectRowsByBuffID(buffID string) ([]map[string]string, error) {
	return FindRows(TableEffect, "buff_id", buffID)
}

func FindEffectSourceRowsByBuffID(buffID string) ([]map[string]string, error) {
	return FindRows(TableEffectSource, "buff_id", buffID)
}

func FindEffectSourceRowsBySourceID(sourceID string) ([]map[string]string, error) {
	return FindRows(TableEffectSource, "source_id", sourceID)
}

func FindDropRowsByMapID(mapID string) ([]map[string]string, error) {
	return FindRows(TableDrop, "map_id", mapID)
}

func cloneRows(rows []map[string]string) []map[string]string {
	result := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		result = append(result, cloneRow(row))
	}
	return result
}

func cloneRow(row map[string]string) map[string]string {
	result := make(map[string]string, len(row))
	for key, value := range row {
		result[key] = value
	}
	return result
}
