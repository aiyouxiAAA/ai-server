package classicdata

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

//go:embed generated/*.json
var generatedTables embed.FS

const (
	TableDrop         = "drop"
	TableItem         = "item"
	TableSkill        = "skill"
	TableMonsterSkill = "monster-skill"
	TableProfession   = "profession"
	TableBuff         = "buff"
	TableMonster      = "monster"
	TableAttribute    = "attribute"
	TableEffect       = "effect"
	TableEffectSource = "effect-source"
)

var KnownTables = []string{
	TableDrop,
	TableItem,
	TableSkill,
	TableMonsterSkill,
	TableProfession,
	TableBuff,
	TableMonster,
	TableAttribute,
	TableEffect,
	TableEffectSource,
}

type Table struct {
	SchemaVersion int                 `json:"schemaVersion"`
	Name          string              `json:"name"`
	Source        string              `json:"source"`
	RowCount      int                 `json:"rowCount"`
	Rows          []map[string]string `json:"rows"`
}

var tableCache = struct {
	sync.RWMutex
	tables map[string]Table
}{
	tables: make(map[string]Table),
}

func LoadTable(name string) (Table, error) {
	table, err := loadTableCached(name)
	if err != nil {
		return Table{}, err
	}
	return cloneTable(table), nil
}

func loadTableCached(name string) (Table, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Table{}, fmt.Errorf("classic data table name is empty")
	}

	tableCache.RLock()
	if table, ok := tableCache.tables[name]; ok {
		tableCache.RUnlock()
		return table, nil
	}
	tableCache.RUnlock()

	tableCache.Lock()
	defer tableCache.Unlock()
	if table, ok := tableCache.tables[name]; ok {
		return table, nil
	}

	path := fmt.Sprintf("generated/%s-table.json", name)
	data, err := generatedTables.ReadFile(path)
	if err != nil {
		return Table{}, fmt.Errorf("read classic data table %s: %w", name, err)
	}

	var table Table
	if err := json.Unmarshal(data, &table); err != nil {
		return Table{}, fmt.Errorf("decode classic data table %s: %w", name, err)
	}
	if table.SchemaVersion != 1 {
		return Table{}, fmt.Errorf("classic data table %s has unsupported schema version %d", name, table.SchemaVersion)
	}
	if table.Name != name {
		return Table{}, fmt.Errorf("classic data table %s has mismatched name %q", name, table.Name)
	}
	if table.RowCount != len(table.Rows) {
		return Table{}, fmt.Errorf("classic data table %s rowCount=%d, rows=%d", name, table.RowCount, len(table.Rows))
	}
	tableCache.tables[name] = table
	return table, nil
}

func cloneTable(table Table) Table {
	table.Rows = cloneRows(table.Rows)
	return table
}

func MustLoadTable(name string) Table {
	table, err := LoadTable(name)
	if err != nil {
		panic(err)
	}
	return table
}

func LoadAllTables() (map[string]Table, error) {
	tables := make(map[string]Table, len(KnownTables))
	for _, name := range KnownTables {
		table, err := LoadTable(name)
		if err != nil {
			return nil, err
		}
		tables[name] = table
	}
	return tables, nil
}
