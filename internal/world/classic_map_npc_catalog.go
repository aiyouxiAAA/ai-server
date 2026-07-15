package world

import (
	"fmt"

	"ai-server/internal/classicdata"
)

func applyClassicMapNPCatalog(definitions map[int]townMapBootstrapDefinition) {
	for mapID, definition := range definitions {
		rows := classicdata.ClassicMapNPCSpawns(mapID)
		legacyCatalogEntries, preservedEntries := splitClassicCatalogNPCEntries(definition.SourceNPCs)
		if len(legacyCatalogEntries) == 0 {
			if len(rows) != 0 {
				panic(fmt.Sprintf("Classic map %d NPC catalog has rows but legacy data is empty", mapID))
			}
			continue
		}
		definition.SourceNPCs = append(buildClassicSourceNPCs(mapID, rows, legacyCatalogEntries), preservedEntries...)
		definitions[mapID] = definition
	}
}

func splitClassicCatalogNPCEntries(entries []sourceNPCEntry) ([]sourceNPCEntry, []sourceNPCEntry) {
	catalogEntries := make([]sourceNPCEntry, 0, len(entries))
	preservedEntries := make([]sourceNPCEntry, 0)
	for _, entry := range entries {
		if entry.RoleID == "-3" || entry.Kind == "collection" || entry.IsGeneratedSourceTransport {
			preservedEntries = append(preservedEntries, entry)
			continue
		}
		catalogEntries = append(catalogEntries, entry)
	}
	return catalogEntries, preservedEntries
}

func buildClassicSourceNPCs(mapID int, rows []classicdata.ClassicMapNPCSpawn, legacyEntries []sourceNPCEntry) []sourceNPCEntry {
	if len(rows) != len(legacyEntries) {
		panic(fmt.Sprintf("Classic map %d NPC catalog coverage mismatch: table=%d legacy=%d", mapID, len(rows), len(legacyEntries)))
	}
	legacyByHandle := make(map[string]sourceNPCEntry, len(legacyEntries))
	for _, entry := range legacyEntries {
		if _, exists := legacyByHandle[entry.Handle]; exists {
			panic(fmt.Sprintf("Classic map %d has duplicate legacy NPC handle %s", mapID, entry.Handle))
		}
		legacyByHandle[entry.Handle] = entry
	}

	result := make([]sourceNPCEntry, 0, len(rows))
	for _, row := range rows {
		legacy, ok := legacyByHandle[row.Handle]
		if !ok {
			panic(fmt.Sprintf("Classic map %d NPC catalog row %s has no legacy dialogue entry", mapID, row.Handle))
		}
		result = append(result, sourceNPCEntry{
			Handle:                     row.Handle,
			RoleID:                     row.RoleID,
			DisplayName:                row.DisplayName,
			SourceQuery:                row.SourceQuery,
			SpriteName:                 row.SpriteName,
			Width:                      row.Width,
			Height:                     row.Height,
			SpawnFlash:                 SpawnPoint{X: row.SpawnX, Y: row.SpawnY},
			State:                      row.State,
			QuestState:                 row.QuestState,
			GuildName:                  row.GuildName,
			GuildPic:                   row.GuildPic,
			Kind:                       row.Kind,
			IsGeneratedSourceTransport: legacy.IsGeneratedSourceTransport,
			Dialogue:                   legacy.Dialogue,
		})
	}
	return result
}
