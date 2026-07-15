package world

import (
	"fmt"
	"strings"

	"ai-server/internal/classicdata"
)

func applyClassicMapCollectionCatalog(definitions map[int]townMapBootstrapDefinition) {
	for mapID, definition := range definitions {
		rows := classicdata.ClassicMapCollectionPoints(mapID)
		legacyEntries, preservedEntries := splitClassicCollectionEntries(definition.SourceNPCs)
		if len(legacyEntries) == 0 {
			if len(rows) != 0 {
				panic(fmt.Sprintf("Classic map %d collection catalog has rows but legacy data is empty", mapID))
			}
			continue
		}
		definition.SourceNPCs = append(preservedEntries, buildClassicCollectionEntries(mapID, rows, legacyEntries)...)
		definitions[mapID] = definition
	}
}

func splitClassicCollectionEntries(entries []sourceNPCEntry) ([]sourceNPCEntry, []sourceNPCEntry) {
	collections := make([]sourceNPCEntry, 0, len(entries))
	preserved := make([]sourceNPCEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.Kind == "collection" {
			collections = append(collections, entry)
			continue
		}
		preserved = append(preserved, entry)
	}
	return collections, preserved
}

func buildClassicCollectionEntries(mapID int, rows []classicdata.ClassicMapCollectionPoint, legacyEntries []sourceNPCEntry) []sourceNPCEntry {
	if len(rows) != len(legacyEntries) {
		panic(fmt.Sprintf("Classic map %d collection coverage mismatch: table=%d legacy=%d", mapID, len(rows), len(legacyEntries)))
	}
	legacyByHandle := make(map[string]sourceNPCEntry, len(legacyEntries))
	for _, entry := range legacyEntries {
		if _, exists := legacyByHandle[entry.Handle]; exists {
			panic(fmt.Sprintf("Classic map %d has duplicate legacy collection handle %s", mapID, entry.Handle))
		}
		legacyByHandle[entry.Handle] = entry
	}
	result := make([]sourceNPCEntry, 0, len(rows))
	for _, row := range rows {
		if _, ok := legacyByHandle[row.Handle]; !ok {
			panic(fmt.Sprintf("Classic map %d collection row %s has no legacy entry", mapID, row.Handle))
		}
		legacyPoint, ok := sourceCollectionPointsByHandle[row.Handle]
		if !ok || legacyPoint.MapID != row.MapID || legacyPoint.DisplayName != row.DisplayName || legacyPoint.SourceQuery != row.SourceQuery || legacyPoint.SpriteName != row.SpriteName || legacyPoint.Width != row.Width || legacyPoint.Height != row.Height || legacyPoint.SpawnFlash != (SpawnPoint{X: row.SpawnX, Y: row.SpawnY}) || legacyPoint.QuestState != row.QuestState || legacyPoint.RequiredItemName != row.RequiredItemName || legacyPoint.RewardItemName != row.RewardItemName || legacyPoint.QuestTitle != row.QuestTitle || legacyPoint.QuestDescription != row.QuestDescription {
			panic(fmt.Sprintf("Classic map %d collection row %s disagrees with legacy collection data", mapID, row.Handle))
		}
		result = append(result, sourceNPCEntry{
			Handle:      row.Handle,
			RoleID:      row.Handle,
			DisplayName: row.DisplayName,
			SourceQuery: row.SourceQuery,
			SpriteName:  row.SpriteName,
			Width:       row.Width,
			Height:      row.Height,
			SpawnFlash:  SpawnPoint{X: row.SpawnX, Y: row.SpawnY},
			QuestState:  row.QuestState,
			Kind:        "collection",
		})
	}
	return result
}

func findClassicMapCollectionPoint(handle string) (SourceCollectionPoint, bool) {
	rows := classicdata.FindClassicMapCollectionPointsByHandle(strings.TrimSpace(handle))
	if len(rows) != 1 {
		return SourceCollectionPoint{}, false
	}
	row := rows[0]
	return SourceCollectionPoint{
		Handle:           row.Handle,
		MapID:            row.MapID,
		DisplayName:      row.DisplayName,
		SourceQuery:      row.SourceQuery,
		SpriteName:       row.SpriteName,
		Width:            row.Width,
		Height:           row.Height,
		SpawnFlash:       SpawnPoint{X: row.SpawnX, Y: row.SpawnY},
		QuestState:       row.QuestState,
		RequiredItemName: row.RequiredItemName,
		RewardItemName:   row.RewardItemName,
		QuestTitle:       row.QuestTitle,
		QuestDescription: row.QuestDescription,
	}, true
}
