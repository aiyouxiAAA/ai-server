package battle

import (
	"embed"
	"encoding/csv"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"ai-server/internal/classicactivity"
	"ai-server/internal/classicdata"
)

//go:embed config/classic-wild-enemy.csv config/classic-wild-encounter.csv config/classic-visible-monster.csv config/classic-battle-reward.csv config/classic-battle-reward-candidate.csv
var battleConfigFiles embed.FS

type sourceWildEnemyConfig struct {
	Cell             CellInfoPush
	QueueIndexTeam   int
	QueueIndexEnemy  int
	Vocation         string
	Source           string
	EncounterHandles []string
}

type sourceWildEncounterConfig struct {
	MapID            string
	EncounterHandles []string
	Weight           int
	Source           string
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
	sourceWildEncounterConfigsByMapID    = mustLoadSourceWildEncounterConfigs()
	sourceVisibleMonsterConfigByKey      = mustLoadSourceVisibleMonsterConfigs()
	sourceBattleRewardConfigByKey        = mustLoadSourceBattleRewardConfigs()
	sourceBattleRewardCandidateByCellKey = mustLoadSourceBattleRewardCandidateConfigsByCellKey()
	sourceBattleRewardCandidateByNameHP  = mustLoadSourceBattleRewardCandidateConfigsByNameHP()
	sourceBattleRewardEquipmentItemNames = mustLoadSourceBattleRewardEquipmentItemNames()
	sourceBattleRewardEquipmentItemOrder = mustLoadSourceBattleRewardEquipmentItemOrder()
	sourceBattleRewardEquipmentPools     = mustLoadSourceBattleRewardEquipmentPools()
)

func sourceEnemyConfigForMap(mapID string) (sourceWildEnemyConfig, bool) {
	config, ok := sourceWildEnemyConfigByMapID[strings.TrimSpace(mapID)]
	return config, ok
}

func sourceEnemyConfigsForMap(mapID string) []sourceWildEnemyConfig {
	configs := sourceWildEnemyConfigsByMapID[strings.TrimSpace(mapID)]
	return append([]sourceWildEnemyConfig(nil), configs...)
}

func sourceWildEncounterConfigsForMap(mapID string) []sourceWildEncounterConfig {
	configs := sourceWildEncounterConfigsByMapID[strings.TrimSpace(mapID)]
	return append([]sourceWildEncounterConfig(nil), configs...)
}

func sourceVisibleMonsterConfigForHandle(mapID string, handle string) (sourceWildEnemyConfig, bool) {
	if config, ok := sourcePointCouponThiefConfigForHandle(mapID, handle); ok {
		return config, true
	}
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

func sourceBattleRewardConfigForExactEncounter(mapID string, sourceMonsterHandle string) (sourceBattleRewardConfig, bool) {
	sourceMonsterHandle = strings.TrimSpace(sourceMonsterHandle)
	if sourceMonsterHandle == "" {
		return sourceBattleRewardConfig{}, false
	}
	if reward, ok := sourcePointCouponThiefRewardConfig(mapID, sourceMonsterHandle); ok {
		return reward, true
	}
	reward, ok := sourceBattleRewardConfigByKey[battleRewardConfigKey(mapID, sourceMonsterHandle)]
	return reward, ok
}

func sourceBattleRewardConfigForEncounter(mapID string, sourceMonsterHandle string) (sourceBattleRewardConfig, bool) {
	if strings.TrimSpace(sourceMonsterHandle) != "" {
		if reward, ok := sourceBattleRewardConfigForExactEncounter(mapID, sourceMonsterHandle); ok {
			return reward, true
		}
	}
	return sourceBattleRewardConfigForMap(mapID)
}

func sourcePointCouponThiefConfigForHandle(mapID string, handle string) (sourceWildEnemyConfig, bool) {
	if !classicactivity.IsPointCouponThiefHandle(mapID, handle) {
		return sourceWildEnemyConfig{}, false
	}
	return sourceWildEnemyConfig{
		Cell: CellInfoPush{
			Camp:              CampEnemy,
			Handle:            strings.TrimSpace(handle),
			Name:              classicactivity.PointCouponThiefName,
			DisplayURL:        classicactivity.PointCouponThiefSourceQuery,
			Level:             15,
			Vocation:          "游侠",
			XScale:            100,
			YScale:            100,
			MaxHP:             1200,
			HP:                1200,
			MaxMP:             300,
			MP:                300,
			Speed:             160,
			Attack:            160,
			Defense:           0,
			Hit:               defaultBattleHit,
			Dog:               defaultBattleDog,
			Fat:               defaultBattleFat,
			CommandLabel:      "普通攻击",
			DamageDefenseType: "physical",
		},
		QueueIndexTeam:  1,
		QueueIndexEnemy: 4,
		Vocation:        "游侠",
		Source:          "capture: tmp/shihuku-capture-summary-combined.json point coupon thief battle cell",
	}, true
}

func sourcePointCouponThiefRewardConfig(mapID string, handle string) (sourceBattleRewardConfig, bool) {
	if !classicactivity.IsPointCouponThiefHandle(mapID, handle) {
		return sourceBattleRewardConfig{}, false
	}
	return sourceBattleRewardConfig{
		SourceMonsterHandle: strings.TrimSpace(handle),
		ExpDelta:            50,
		Items:               []string{"点券x10"},
		Source:              "capture: tmp/shihuku-capture-summary-combined.json reward 点券 total x10 tail exp 50",
		Status:              "confirmed",
	}, true
}

func mustLoadSourceWildEnemyConfigs() map[string]sourceWildEnemyConfig {
	configs := map[string]sourceWildEnemyConfig{}
	for rowIndex, row := range classicdata.MustRows(classicdata.TableMonster) {
		if classicDataOptionalString(row, "source_kind") != "wild" {
			continue
		}
		mapID, config := sourceWildEnemyConfigFromClassicDataRow(row, rowIndex)
		if _, exists := configs[mapID]; exists {
			continue
		}
		configs[mapID] = config
	}
	return configs
}

func mustLoadSourceWildEnemyConfigLists() map[string][]sourceWildEnemyConfig {
	configs := map[string][]sourceWildEnemyConfig{}
	for rowIndex, row := range classicdata.MustRows(classicdata.TableMonster) {
		if classicDataOptionalString(row, "source_kind") != "wild" {
			continue
		}
		mapID, config := sourceWildEnemyConfigFromClassicDataRow(row, rowIndex)
		configs[mapID] = append(configs[mapID], config)
	}
	return configs
}

func mustLoadSourceWildEncounterConfigs() map[string][]sourceWildEncounterConfig {
	records := mustReadBattleConfigCSV("config/classic-wild-encounter.csv")
	header := battleConfigHeader(records[0])
	configs := map[string][]sourceWildEncounterConfig{}
	knownHandlesByMapID := map[string]map[string]bool{}
	for mapID, enemies := range sourceWildEnemyConfigsByMapID {
		knownHandles := map[string]bool{}
		for _, enemy := range enemies {
			knownHandles[enemy.Cell.Handle] = true
		}
		knownHandlesByMapID[mapID] = knownHandles
	}
	for rowIndex, row := range records[1:] {
		mapID := requiredBattleConfigString(row, header, "map_id", rowIndex)
		handles := splitBattleConfigList(requiredBattleConfigString(row, header, "encounter_handles", rowIndex))
		weight := requiredBattleConfigInt(row, header, "weight", rowIndex)
		if weight <= 0 {
			panic(fmt.Sprintf("battle config row %d must have positive encounter weight", rowIndex+2))
		}
		knownHandles := knownHandlesByMapID[mapID]
		for _, handle := range handles {
			if !knownHandles[handle] {
				panic(fmt.Sprintf("battle config row %d references unknown wild handle %s for map %s", rowIndex+2, handle, mapID))
			}
		}
		configs[mapID] = append(configs[mapID], sourceWildEncounterConfig{
			MapID:            mapID,
			EncounterHandles: handles,
			Weight:           weight,
			Source:           optionalBattleConfigString(row, header, "source"),
		})
	}
	return configs
}

func mustLoadSourceVisibleMonsterConfigs() map[string]sourceWildEnemyConfig {
	configs := map[string]sourceWildEnemyConfig{}
	for rowIndex, row := range classicdata.MustRows(classicdata.TableMonster) {
		if classicDataOptionalString(row, "source_kind") != "visible" {
			continue
		}
		mapID, config := sourceWildEnemyConfigFromClassicDataRow(row, rowIndex)
		configs[visibleMonsterConfigKey(mapID, config.Cell.Handle)] = config
	}
	return configs
}

func sourceWildEnemyConfigFromClassicDataRow(row map[string]string, rowIndex int) (string, sourceWildEnemyConfig) {
	mapID := requiredClassicDataString(row, "monster", "map_id", rowIndex)
	maxHP := requiredClassicDataInt(row, "monster", "max_hp", rowIndex)
	maxMP := requiredClassicDataInt(row, "monster", "max_mp", rowIndex)
	return mapID, sourceWildEnemyConfig{
		Cell: CellInfoPush{
			Camp:              CampEnemy,
			Handle:            requiredClassicDataString(row, "monster", "handle", rowIndex),
			Name:              requiredClassicDataString(row, "monster", "name", rowIndex),
			DisplayURL:        requiredClassicDataString(row, "monster", "display_url", rowIndex),
			Level:             requiredClassicDataInt(row, "monster", "level", rowIndex),
			Vocation:          requiredClassicDataString(row, "monster", "vocation", rowIndex),
			XScale:            100,
			YScale:            100,
			MaxHP:             maxHP,
			HP:                maxHP,
			MaxMP:             maxMP,
			MP:                maxMP,
			Speed:             requiredClassicDataInt(row, "monster", "speed", rowIndex),
			Attack:            requiredClassicDataInt(row, "monster", "attack", rowIndex),
			Defense:           requiredClassicDataInt(row, "monster", "defense", rowIndex),
			MgcDefense:        requiredClassicDataInt(row, "monster", "mgc_defense", rowIndex),
			Hit:               defaultBattleHit,
			Dog:               defaultBattleDog,
			Fat:               defaultBattleFat,
			CommandLabel:      requiredClassicDataString(row, "monster", "command_label", rowIndex),
			DamageDefenseType: defaultString(classicDataOptionalString(row, "damage_defense_type"), "physical"),
		},
		QueueIndexTeam:   requiredClassicDataInt(row, "monster", "queue_index_team", rowIndex),
		QueueIndexEnemy:  requiredClassicDataInt(row, "monster", "queue_index_enemy", rowIndex),
		Vocation:         requiredClassicDataString(row, "monster", "vocation", rowIndex),
		Source:           classicDataOptionalString(row, "source"),
		EncounterHandles: splitBattleConfigList(classicDataOptionalString(row, "encounter_handles")),
	}
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
			Vocation:          requiredBattleConfigString(row, header, "vocation", rowIndex),
			XScale:            100,
			YScale:            100,
			MaxHP:             maxHP,
			HP:                maxHP,
			MaxMP:             maxMP,
			MP:                maxMP,
			Speed:             requiredBattleConfigInt(row, header, "speed", rowIndex),
			Attack:            requiredBattleConfigInt(row, header, "attack", rowIndex),
			Defense:           requiredBattleConfigInt(row, header, "defense", rowIndex),
			MgcDefense:        requiredBattleConfigInt(row, header, "mgc_defense", rowIndex),
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
	configs := map[string]sourceBattleRewardConfig{}
	for rowIndex, row := range classicdata.MustRows(classicdata.TableDrop) {
		mapID := requiredClassicDataString(row, "drop", "map_id", rowIndex)
		sourceMonsterHandle := classicDataOptionalString(row, "source_monster_handle")
		configs[battleRewardConfigKey(mapID, sourceMonsterHandle)] = sourceBattleRewardConfig{
			SourceMonsterHandle: sourceMonsterHandle,
			ExpDelta:            requiredClassicDataInt(row, "drop", "exp_delta", rowIndex),
			Items:               splitBattleConfigList(classicDataOptionalString(row, "items")),
			DropRates: parseSourceBattleRewardDropRates(
				classicDataOptionalString(row, "item_counts"),
				classicDataOptionalString(row, "item_drop_windows"),
				classicDataOptionalString(row, "item_observed_rates"),
				classicDataOptionalInt(row, "window_count"),
			),
			Source: classicDataOptionalString(row, "source"),
			Status: requiredClassicDataString(row, "drop", "status", rowIndex),
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

func requiredClassicDataString(row map[string]string, tableName string, column string, rowIndex int) string {
	value := classicDataOptionalString(row, column)
	if value == "" {
		panic(fmt.Sprintf("classic data table %s row %d missing %s", tableName, rowIndex+2, column))
	}
	return value
}

func classicDataOptionalString(row map[string]string, column string) string {
	if row == nil {
		return ""
	}
	return strings.TrimSpace(row[column])
}

func requiredClassicDataInt(row map[string]string, tableName string, column string, rowIndex int) int {
	raw := requiredClassicDataString(row, tableName, column, rowIndex)
	value, err := strconv.Atoi(raw)
	if err != nil {
		panic(fmt.Sprintf("classic data table %s row %d invalid %s=%q", tableName, rowIndex+2, column, raw))
	}
	return value
}

func classicDataOptionalInt(row map[string]string, column string) int {
	raw := classicDataOptionalString(row, column)
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

func sourceBattleRewardItemIsEquipment(itemName string) bool {
	return sourceBattleRewardEquipmentItemNames[strings.TrimSpace(itemName)]
}

func sourceBattleRewardEquipmentFallbackPool(monsterName string, excluded map[string]bool) []string {
	pool := sourceBattleRewardEquipmentPools[strings.TrimSpace(monsterName)]
	if len(pool) == 0 {
		return []string{}
	}
	result := make([]string, 0, len(pool))
	for _, itemName := range pool {
		if excluded[strings.TrimSpace(itemName)] {
			continue
		}
		result = append(result, itemName)
	}
	return result
}

func mustLoadSourceBattleRewardEquipmentItemNames() map[string]bool {
	items := map[string]bool{}
	for _, row := range classicdata.MustRows(classicdata.TableItem) {
		if classicDataOptionalString(row, "item_type") != "equip" {
			continue
		}
		name := classicDataOptionalString(row, "name")
		if name == "" {
			continue
		}
		items[name] = true
	}
	return items
}

func mustLoadSourceBattleRewardEquipmentItemOrder() map[string]int {
	order := map[string]int{}
	for index, row := range classicdata.MustRows(classicdata.TableItem) {
		if classicDataOptionalString(row, "item_type") != "equip" {
			continue
		}
		name := classicDataOptionalString(row, "name")
		if name == "" {
			continue
		}
		order[name] = index
	}
	return order
}

func mustLoadSourceBattleRewardEquipmentPools() map[string][]string {
	sets := map[string]map[string]bool{}
	candidateRows := loadSourceBattleRewardCandidateConfigRows()
	for _, config := range candidateRows {
		monsterName := strings.TrimSpace(config.MonsterName)
		if monsterName == "" {
			continue
		}
		for _, drop := range config.DropRates {
			if !sourceBattleRewardItemIsEquipment(drop.ItemName) {
				continue
			}
			if _, ok := sets[monsterName]; !ok {
				sets[monsterName] = map[string]bool{}
			}
			sets[monsterName][strings.TrimSpace(drop.ItemName)] = true
		}
	}

	itemRows := classicdata.MustRows(classicdata.TableItem)
	for monsterName, set := range sets {
		for _, row := range itemRows {
			if classicDataOptionalString(row, "item_type") != "equip" {
				continue
			}
			itemName := classicDataOptionalString(row, "name")
			if itemName == "" || !strings.HasPrefix(itemName, monsterName) {
				continue
			}
			set[itemName] = true
		}
	}

	pools := map[string][]string{}
	for monsterName, set := range sets {
		items := make([]string, 0, len(set))
		for itemName := range set {
			items = append(items, itemName)
		}
		sort.Slice(items, func(left int, right int) bool {
			leftOrder, leftOK := sourceBattleRewardEquipmentItemOrder[items[left]]
			rightOrder, rightOK := sourceBattleRewardEquipmentItemOrder[items[right]]
			if leftOK && rightOK && leftOrder != rightOrder {
				return leftOrder < rightOrder
			}
			if leftOK != rightOK {
				return leftOK
			}
			return items[left] < items[right]
		})
		pools[monsterName] = items
	}
	return pools
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
	bestPositiveExp := 0
	bestPositiveCount := -1
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
		if exp > 0 && count > bestPositiveCount {
			bestPositiveExp = exp
			bestPositiveCount = count
		}
	}
	if bestPositiveCount >= 0 {
		return bestPositiveExp
	}
	return bestExp
}
