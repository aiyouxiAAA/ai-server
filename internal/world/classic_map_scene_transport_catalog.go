package world

import (
	"fmt"

	"ai-server/internal/classicdata"
)

func applyClassicMapSceneTransportCatalog(definitions map[int]townMapBootstrapDefinition) {
	for mapID, definition := range definitions {
		rows := classicdata.ClassicMapSceneTransportSpawns(mapID)
		legacyEntries, preservedEntries := splitClassicSceneTransportEntries(definition.SourceNPCs)
		if len(legacyEntries) == 0 {
			if len(rows) != 0 {
				panic(fmt.Sprintf("Classic map %d scene transport catalog has rows but legacy data is empty", mapID))
			}
			continue
		}
		definition.SourceNPCs = append(preservedEntries, buildClassicSceneTransports(mapID, rows, legacyEntries)...)
		definitions[mapID] = definition
	}
}

func splitClassicSceneTransportEntries(entries []sourceNPCEntry) ([]sourceNPCEntry, []sourceNPCEntry) {
	transports := make([]sourceNPCEntry, 0, len(entries))
	preserved := make([]sourceNPCEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.RoleID == "-3" {
			transports = append(transports, entry)
			continue
		}
		preserved = append(preserved, entry)
	}
	return transports, preserved
}

func buildClassicSceneTransports(mapID int, rows []classicdata.ClassicMapSceneTransportSpawn, legacyEntries []sourceNPCEntry) []sourceNPCEntry {
	if len(rows) != len(legacyEntries) {
		panic(fmt.Sprintf("Classic map %d scene transport coverage mismatch: table=%d legacy=%d", mapID, len(rows), len(legacyEntries)))
	}
	legacyByHandle := make(map[string]sourceNPCEntry, len(legacyEntries))
	for _, entry := range legacyEntries {
		if _, exists := legacyByHandle[entry.Handle]; exists {
			panic(fmt.Sprintf("Classic map %d has duplicate legacy scene transport handle %s", mapID, entry.Handle))
		}
		legacyByHandle[entry.Handle] = entry
	}

	result := make([]sourceNPCEntry, 0, len(rows))
	for _, row := range rows {
		legacy, ok := legacyByHandle[row.Handle]
		if !ok {
			panic(fmt.Sprintf("Classic map %d scene transport row %s has no legacy dialogue entry", mapID, row.Handle))
		}
		if legacy.SourceQuery != row.SourceQuery || legacy.SpriteName != row.SpriteName || legacy.Width != row.Width || legacy.Height != row.Height || legacy.SpawnFlash != (SpawnPoint{X: row.SpawnX, Y: row.SpawnY}) {
			panic(fmt.Sprintf("Classic map %d scene transport row %s disagrees with legacy entity data", mapID, row.Handle))
		}
		result = append(result, sourceNPCEntry{
			Handle:                     row.Handle,
			RoleID:                     "-3",
			SourceQuery:                row.SourceQuery,
			SpriteName:                 row.SpriteName,
			Width:                      row.Width,
			Height:                     row.Height,
			SpawnFlash:                 SpawnPoint{X: row.SpawnX, Y: row.SpawnY},
			IsGeneratedSourceTransport: legacy.IsGeneratedSourceTransport,
			Dialogue:                   legacy.Dialogue,
		})
	}
	return result
}

func resolveClassicMapSceneTransportDestination(fromMapID int, handle string) (TownTransportDestination, bool) {
	row, ok := classicdata.FindClassicMapSceneTransportSpawn(fromMapID, handle)
	if !ok || !SupportsTownTransferMap(row.TargetMapID) {
		return TownTransportDestination{}, false
	}
	return TownTransportDestination{
		MapID: row.TargetMapID,
		Spawn: SpawnPoint{X: row.TargetSpawnX, Y: row.TargetSpawnY},
	}, true
}

func resolveTownTransportDestinationFromLegacyData(fromMapID int, handle string) (TownTransportDestination, bool) {
	return resolveLegacyTownTransportDestinationFromMap(fromMapID, handle)
}

func init() {
	for _, mapEntry := range classicdata.ClassicMaps() {
		for _, row := range classicdata.ClassicMapSceneTransportSpawns(mapEntry.ID) {
			legacyDestination, ok := resolveLegacyTownTransportDestinationFromMap(row.MapID, row.Handle)
			if !ok || legacyDestination.MapID != row.TargetMapID || legacyDestination.Spawn != (SpawnPoint{X: row.TargetSpawnX, Y: row.TargetSpawnY}) {
				panic(fmt.Sprintf("Classic map %d scene transport row %s disagrees with legacy destination", row.MapID, row.Handle))
			}
		}
	}
}
