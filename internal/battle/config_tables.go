package battle

import (
	"embed"
	"encoding/csv"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

//go:embed config/classic-wild-enemy.csv config/classic-visible-monster.csv config/classic-battle-reward.csv config/classic-battle-reward-candidate.csv
var battleConfigFiles embed.FS

type sourceWildEnemyConfig struct {
	Cell             CellInfoPush
	QueueIndexTeam   int
	QueueIndexEnemy  int
	Vocation         string
	Source           string
	EncounterHandles []string
}

type sourceBattleRewardConfig struct {
	SourceMonsterHandle string
	ExpDelta            int
	Items               []string
	DropRates           []sourceBattleRewardDropRate
	Source              string
	Status              string
}

type sourceBattleRewardCandidateConfig struct {
	MapID       string
	MonsterName string
	MaxHP       int
	ExpDelta    int
	DropRates   []sourceBattleRewardDropRate
	Source      string
	Status      string
}

type sourceBattleRewardDropRate struct {
	ItemName    string
	Quantity    int
	Numerator   int
	Denominator int
}

var (
	sourceWildEnemyConfigByMapID         = mustLoadSourceWildEnemyConfigs()
	sourceWildEnemyConfigsByMapID        = mustLoadSourceWildEnemyConfigLists()
	sourceVisibleMonsterConfigByKey      = mustLoadSourceVisibleMonsterConfigs()
	sourceBattleRewardConfigByKey        = mustLoadSourceBattleRewardConfigs()
	sourceBattleRewardCandidateByCellKey = mustLoadSourceBattleRewardCandidateConfigsByCellKey()
	sourceBattleRewardCandidateByNameHP  = mustLoadSourceBattleRewardCandidateConfigsByNameHP()
)

func sourceEnemyConfigForMap(mapID string) (sourceWildEnemyConfig, bool) {
	config, ok := sourceWildEnemyConfigByMapID[strings.TrimSpace(mapID)]
	return config, ok
}

func sourceEnemyConfigsForMap(mapID string) []sourceWildEnemyConfig {
	configs := sourceWildEnemyConfigsByMapID[strings.TrimSpace(mapID)]
	return append([]sourceWildEnemyConfig(nil), configs...)
}

func sourceVisibleMonsterConfigForHandle(mapID string, handle string) (sourceWildEnemyConfig, bool) {
	config, ok := sourceVisibleMonsterConfigByKey[visibleMonsterConfigKey(mapID, handle)]
	return config, ok
}

func sourceVisibleMonsterConfigsForHandle(mapID string, handle string) ([]sourceWildEnemyConfig, bool) {
	config, ok := sourceVisibleMonsterConfigForHandle(mapID, handle)
	if !ok {
		return nil, false
	}
	handles := config.EncounterHandles
	if len(handles) == 0 {
		handles = []string{config.Cell.Handle}
	}
	configs := make([]sourceWildEnemyConfig, 0, len(handles))
	for _, encounterHandle := range handles {
		encounterConfig, ok := sourceVisibleMonsterConfigForHandle(mapID, encounterHandle)
		if !ok {
			return nil, false
		}
		configs = append(configs, encounterConfig)
	}
	return configs, len(configs) > 0
}

func sourceBattleRewardConfigForMap(mapID string) (sourceBattleRewardConfig, bool) {
	reward, ok := sourceBattleRewardConfigByKey[battleRewardConfigKey(mapID, "")]
	return reward, ok
}

func sourceBattleRewardConfigForEncounter(mapID string, sourceMonsterHandle string) (sourceBattleRewardConfig, bool) {
	if strings.TrimSpace(sourceMonsterHandle) != "" {
		if reward, ok := sourceBattleRewardConfigByKey[battleRewardConfigKey(mapID, sourceMonsterHandle)]; ok {
			return reward, true
		}
	}
	return sourceBattleRewardConfigForMap(mapID)
}

func mustLoadSourceWildEnemyConfigs() map[string]sourceWildEnemyConfig {
	records := mustReadBattleConfigCSV("config/classic-wild-enemy.csv")
	header := battleConfigHeader(records[0])
	configs := map[string]sourceWildEnemyConfig{}
	for rowIndex, row := range records[1:] {
		mapID, config := sourceWildEnemyConfigFromRow(row, header, rowIndex)
		if _, exists := configs[mapID]; exists {
			continue
		}
		configs[mapID] = config
	}
	return configs
}

func mustLoadSourceWildEnemyConfigLists() map[string][]sourceWildEnemyConfig {
	records := mustReadBattleConfigCSV("config/classic-wild-enemy.csv")
	header := battleConfigHeader(records[0])
	configs := map[string][]sourceWildEnemyConfig{}
	for rowIndex, row := range records[1:] {
		mapID, config := sourceWildEnemyConfigFromRow(row, header, rowIndex)
		configs[mapID] = append(configs[mapID], config)
	}
	return configs
}

func mustLoadSourceVisibleMonsterConfigs() map[string]sourceWildEnemyConfig {
	records := mustReadBattleConfigCSV("config/classic-visible-monster.csv")
	header := battleConfigHeader(records[0])
	configs := map[string]sourceWildEnemyConfig{}
	for rowIndex, row := range records[1:] {
		mapID, config := sourceWildEnemyConfigFromRow(row, header, rowIndex)
		configs[visibleMonsterConfigKey(mapID, config.Cell.Handle)] = config
	}
	return configs
}

func sourceWildEnemyConfigFromRow(row []string, header map[string]int, rowIndex int) (string, sourceWildEnemyConfig) {
	mapID := requiredBattleConfigString(row, header, "map_id", rowIndex)
	maxHP := requiredBattleConfigInt(row, header, "max_hp", rowIndex)
	maxMP := requiredBattleConfigInt(row, header, "max_mp", rowIndex)
	return mapID, sourceWildEnemyConfig{
		Cell: CellInfoPush{
			Camp:              CampEnemy,
			Handle:            requiredBattleConfigString(row, header, "handle", rowIndex),
			Name:              requiredBattleConfigString(row, header, "name", rowIndex),
			DisplayURL:        requiredBattleConfigString(row, header, "display_url", rowIndex),
			Level:             requiredBattleConfigInt(row, header, "level", rowIndex),
			XScale:            100,
			YScale:            100,
			MaxHP:             maxHP,
			HP:                maxHP,
			MaxMP:             maxMP,
			MP:                maxMP,
			Speed:             requiredBattleConfigInt(row, header, "speed", rowIndex),
			Attack:            requiredBattleConfigInt(row, header, "attack", rowIndex),
			Defense:           requiredBattleConfigInt(row, header, "defense", rowIndex),
			Hit:               defaultBattleHit,
			Dog:               defaultBattleDog,
			Fat:               defaultBattleFat,
			CommandLabel:      requiredBattleConfigString(row, header, "command_label", rowIndex),
			DamageDefenseType: defaultString(optionalBattleConfigString(row, header, "damage_defense_type"), "physical"),
		},
		QueueIndexTeam:   requiredBattleConfigInt(row, header, "queue_index_team", rowIndex),
		QueueIndexEnemy:  requiredBattleConfigInt(row, header, "queue_index_enemy", rowIndex),
		Vocation:         requiredBattleConfigString(row, header, "vocation", rowIndex),
		Source:           optionalBattleConfigString(row, header, "source"),
		EncounterHandles: splitBattleConfigList(optionalBattleConfigString(row, header, "encounter_handles")),
	}
}

func visibleMonsterConfigKey(mapID string, handle string) string {
	return strings.TrimSpace(mapID) + ":" + strings.TrimSpace(handle)
}

func battleRewardConfigKey(mapID string, sourceMonsterHandle string) string {
	return strings.TrimSpace(mapID) + ":" + strings.TrimSpace(sourceMonsterHandle)
}

func mustLoadSourceBattleRewardConfigs() map[string]sourceBattleRewardConfig {
	records := mustReadBattleConfigCSV("config/classic-battle-reward.csv")
	header := battleConfigHeader(records[0])
	configs := map[string]sourceBattleRewardConfig{}
	for rowIndex, row := range records[1:] {
		mapID := requiredBattleConfigString(row, header, "map_id", rowIndex)
		sourceMonsterHandle := optionalBattleConfigString(row, header, "source_monster_handle")
		configs[battleRewardConfigKey(mapID, sourceMonsterHandle)] = sourceBattleRewardConfig{
			SourceMonsterHandle: sourceMonsterHandle,
			ExpDelta:            requiredBattleConfigInt(row, header, "exp_delta", rowIndex),
			Items:               splitBattleConfigList(optionalBattleConfigString(row, header, "items")),
			DropRates: parseSourceBattleRewardDropRates(
				optionalBattleConfigString(row, header, "item_counts"),
				optionalBattleConfigString(row, header, "item_drop_windows"),
				optionalBattleConfigString(row, header, "item_observed_rates"),
				optionalBattleConfigInt(row, header, "window_count"),
			),
			Source: optionalBattleConfigString(row, header, "source"),
			Status: requiredBattleConfigString(row, header, "status", rowIndex),
		}
	}
	return configs
}

func mustLoadSourceBattleRewardCandidateConfigsByCellKey() map[string]sourceBattleRewardCandidateConfig {
	configs := map[string]sourceBattleRewardCandidateConfig{}
	for _, config := range loadSourceBattleRewardCandidateConfigRows() {
		configs[battleRewardCandidateCellKey(config.MapID, config.MonsterName, config.MaxHP)] = config
	}
	return configs
}

func mustLoadSourceBattleRewardCandidateConfigsByNameHP() map[string]sourceBattleRewardCandidateConfig {
	configs := map[string]sourceBattleRewardCandidateConfig{}
	for _, config := range loadSourceBattleRewardCandidateConfigRows() {
		key := battleRewardCandidateNameHPKey(config.MonsterName, config.MaxHP)
		if _, exists := configs[key]; exists {
			continue
		}
		configs[key] = config
	}
	return configs
}

func loadSourceBattleRewardCandidateConfigRows() []sourceBattleRewardCandidateConfig {
	records := mustReadBattleConfigCSV("config/classic-battle-reward-candidate.csv")
	header := battleConfigHeader(records[0])
	configs := []sourceBattleRewardCandidateConfig{}
	for rowIndex, row := range records[1:] {
		mapID := requiredBattleConfigString(row, header, "map_id", rowIndex)
		monsterName := requiredBattleConfigString(row, header, "monster_name", rowIndex)
		maxHP := requiredBattleConfigInt(row, header, "max_hp", rowIndex)
		windowCount := requiredBattleConfigInt(row, header, "window_count", rowIndex)
		configs = append(configs, sourceBattleRewardCandidateConfig{
			MapID:       mapID,
			MonsterName: monsterName,
			MaxHP:       maxHP,
			ExpDelta:    dominantSourceBattleRewardExperience(optionalBattleConfigString(row, header, "experience_counts")),
			DropRates: parseSourceBattleRewardDropRates(
				optionalBattleConfigString(row, header, "item_counts"),
				optionalBattleConfigString(row, header, "item_drop_windows"),
				optionalBattleConfigString(row, header, "item_observed_rates"),
				windowCount,
			),
			Source: optionalBattleConfigString(row, header, "source"),
			Status: optionalBattleConfigString(row, header, "status"),
		})
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

func optionalBattleConfigInt(row []string, header map[string]int, name string) int {
	raw := optionalBattleConfigString(row, header, name)
	if raw == "" {
		return 0
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0
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

func battleRewardCandidateCellKey(mapID string, monsterName string, maxHP int) string {
	return strings.TrimSpace(mapID) + ":" + strings.TrimSpace(monsterName) + ":" + strconv.Itoa(maxHP)
}

func battleRewardCandidateNameHPKey(monsterName string, maxHP int) string {
	return strings.TrimSpace(monsterName) + ":" + strconv.Itoa(maxHP)
}

func sourceBattleRewardCandidateForCell(mapID string, monsterName string, maxHP int) (sourceBattleRewardCandidateConfig, bool) {
	if config, ok := sourceBattleRewardCandidateByCellKey[battleRewardCandidateCellKey(mapID, monsterName, maxHP)]; ok {
		return config, true
	}
	config, ok := sourceBattleRewardCandidateByNameHP[battleRewardCandidateNameHPKey(monsterName, maxHP)]
	return config, ok
}

func parseSourceBattleRewardDropRates(itemCounts string, itemDropWindows string, itemObservedRates string, windowCount int) []sourceBattleRewardDropRate {
	totalCounts := parseSourceBattleRewardItemCounts(itemCounts)
	dropWindows := parseSourceBattleRewardItemCounts(itemDropWindows)
	rates := []sourceBattleRewardDropRate{}
	for _, part := range splitBattleConfigList(itemObservedRates) {
		name, numerator, denominator, ok := parseSourceBattleRewardObservedRate(part)
		if !ok {
			continue
		}
		if denominator <= 0 {
			denominator = windowCount
		}
		quantity := 1
		if windows := dropWindows[name]; windows > 0 {
			quantity = maxInt(1, (totalCounts[name]+windows/2)/windows)
		}
		rates = append(rates, sourceBattleRewardDropRate{
			ItemName:    name,
			Quantity:    quantity,
			Numerator:   numerator,
			Denominator: maxInt(1, denominator),
		})
	}
	return rates
}

func parseSourceBattleRewardItemCounts(value string) map[string]int {
	counts := map[string]int{}
	for _, item := range splitBattleConfigList(value) {
		name, count := parseSourceBattleRewardItemStack(item)
		if name == "" {
			continue
		}
		counts[name] += count
	}
	return counts
}

func parseSourceBattleRewardItemStack(value string) (string, int) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", 0
	}
	match := regexp.MustCompile(`^(.+?)x(\d+)$`).FindStringSubmatch(value)
	if len(match) != 3 {
		return value, 1
	}
	count, err := strconv.Atoi(match[2])
	if err != nil || count <= 0 {
		count = 1
	}
	return strings.TrimSpace(match[1]), count
}

func parseSourceBattleRewardObservedRate(value string) (string, int, int, bool) {
	parts := strings.Split(strings.TrimSpace(value), "=")
	if len(parts) != 2 {
		return "", 0, 0, false
	}
	fraction := strings.Split(strings.TrimSpace(parts[1]), "/")
	if len(fraction) != 2 {
		return "", 0, 0, false
	}
	numerator, err := strconv.Atoi(strings.TrimSpace(fraction[0]))
	if err != nil {
		return "", 0, 0, false
	}
	denominator, err := strconv.Atoi(strings.TrimSpace(fraction[1]))
	if err != nil {
		return "", 0, 0, false
	}
	return strings.TrimSpace(parts[0]), numerator, denominator, true
}

func dominantSourceBattleRewardExperience(value string) int {
	bestExp := 0
	bestCount := -1
	for _, part := range splitBattleConfigList(value) {
		name, count := parseSourceBattleRewardItemStack(part)
		exp, err := strconv.Atoi(name)
		if err != nil {
			continue
		}
		if count > bestCount {
			bestExp = exp
			bestCount = count
		}
	}
	return bestExp
}
