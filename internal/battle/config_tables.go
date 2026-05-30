package battle

import (
	"embed"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
)

//go:embed config/classic-wild-enemy.csv config/classic-battle-reward.csv
var battleConfigFiles embed.FS

type sourceWildEnemyConfig struct {
	Cell            CellInfoPush
	QueueIndexTeam  int
	QueueIndexEnemy int
	Source          string
}

type sourceBattleRewardConfig struct {
	ExpDelta int
	Items    []string
	Source   string
	Status   string
}

var (
	sourceWildEnemyConfigByMapID    = mustLoadSourceWildEnemyConfigs()
	sourceBattleRewardConfigByMapID = mustLoadSourceBattleRewardConfigs()
)

func sourceEnemyConfigForMap(mapID string) (sourceWildEnemyConfig, bool) {
	config, ok := sourceWildEnemyConfigByMapID[strings.TrimSpace(mapID)]
	return config, ok
}

func sourceBattleRewardConfigForMap(mapID string) (sourceBattleRewardConfig, bool) {
	reward, ok := sourceBattleRewardConfigByMapID[strings.TrimSpace(mapID)]
	return reward, ok
}

func mustLoadSourceWildEnemyConfigs() map[string]sourceWildEnemyConfig {
	records := mustReadBattleConfigCSV("config/classic-wild-enemy.csv")
	header := battleConfigHeader(records[0])
	configs := map[string]sourceWildEnemyConfig{}
	for rowIndex, row := range records[1:] {
		mapID := requiredBattleConfigString(row, header, "map_id", rowIndex)
		maxHP := requiredBattleConfigInt(row, header, "max_hp", rowIndex)
		maxMP := requiredBattleConfigInt(row, header, "max_mp", rowIndex)
		configs[mapID] = sourceWildEnemyConfig{
			Cell: CellInfoPush{
				Camp:         CampEnemy,
				Handle:       requiredBattleConfigString(row, header, "handle", rowIndex),
				Name:         requiredBattleConfigString(row, header, "name", rowIndex),
				DisplayURL:   requiredBattleConfigString(row, header, "display_url", rowIndex),
				Level:        requiredBattleConfigInt(row, header, "level", rowIndex),
				XScale:       100,
				YScale:       100,
				MaxHP:        maxHP,
				HP:           maxHP,
				MaxMP:        maxMP,
				MP:           maxMP,
				Speed:        requiredBattleConfigInt(row, header, "speed", rowIndex),
				Attack:       requiredBattleConfigInt(row, header, "attack", rowIndex),
				Defense:      requiredBattleConfigInt(row, header, "defense", rowIndex),
				Hit:          defaultBattleHit,
				Dog:          defaultBattleDog,
				Fat:          defaultBattleFat,
				CommandLabel: requiredBattleConfigString(row, header, "command_label", rowIndex),
			},
			QueueIndexTeam:  requiredBattleConfigInt(row, header, "queue_index_team", rowIndex),
			QueueIndexEnemy: requiredBattleConfigInt(row, header, "queue_index_enemy", rowIndex),
			Source:          optionalBattleConfigString(row, header, "source"),
		}
	}
	return configs
}

func mustLoadSourceBattleRewardConfigs() map[string]sourceBattleRewardConfig {
	records := mustReadBattleConfigCSV("config/classic-battle-reward.csv")
	header := battleConfigHeader(records[0])
	configs := map[string]sourceBattleRewardConfig{}
	for rowIndex, row := range records[1:] {
		mapID := requiredBattleConfigString(row, header, "map_id", rowIndex)
		configs[mapID] = sourceBattleRewardConfig{
			ExpDelta: requiredBattleConfigInt(row, header, "exp_delta", rowIndex),
			Items:    splitBattleConfigList(optionalBattleConfigString(row, header, "items")),
			Source:   optionalBattleConfigString(row, header, "source"),
			Status:   requiredBattleConfigString(row, header, "status", rowIndex),
		}
	}
	return configs
}

func mustReadBattleConfigCSV(path string) [][]string {
	data, err := battleConfigFiles.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("read battle config %s: %v", path, err))
	}
	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		panic(fmt.Sprintf("parse battle config %s: %v", path, err))
	}
	if len(records) < 2 {
		panic(fmt.Sprintf("battle config %s has no rows", path))
	}
	return records
}

func battleConfigHeader(record []string) map[string]int {
	header := map[string]int{}
	for index, name := range record {
		header[strings.TrimSpace(name)] = index
	}
	return header
}

func requiredBattleConfigString(row []string, header map[string]int, name string, rowIndex int) string {
	value := optionalBattleConfigString(row, header, name)
	if value == "" {
		panic(fmt.Sprintf("battle config row %d missing %s", rowIndex+2, name))
	}
	return value
}

func optionalBattleConfigString(row []string, header map[string]int, name string) string {
	index, ok := header[name]
	if !ok || index < 0 || index >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[index])
}

func requiredBattleConfigInt(row []string, header map[string]int, name string, rowIndex int) int {
	raw := requiredBattleConfigString(row, header, name, rowIndex)
	value, err := strconv.Atoi(raw)
	if err != nil {
		panic(fmt.Sprintf("battle config row %d invalid %s=%q", rowIndex+2, name, raw))
	}
	return value
}

func splitBattleConfigList(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return []string{}
	}
	parts := strings.Split(value, ";")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}
