package world

import (
	"fmt"

	"ai-server/internal/classicdata"
)

func applyClassicMapSceneTransportCatalog(definitions map[int]townMapBootstrapDefinition) {
	for mapID, definition := range definitions {
		rows := classicdata.ClassicMapSceneTransportSpawns(mapID)
		legacyEntries, preservedEntries := splitClassicSceneTransportEntries(definition.SourceNPCs)
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
	legacyByHandle := make(map[string]sourceNPCEntry, len(legacyEntries))
	for _, entry := range legacyEntries {
		if _, exists := legacyByHandle[entry.Handle]; exists {
			panic(fmt.Sprintf("Classic map %d has duplicate legacy scene transport handle %s", mapID, entry.Handle))
		}
		legacyByHandle[entry.Handle] = entry
	}

	result := make([]sourceNPCEntry, 0, len(rows))
	for _, row := range rows {
		authored := row.Source == "nine_wilds_terrain_20260909"
		legacy, ok := legacyByHandle[row.Handle]
		if !ok && !authored {
			panic(fmt.Sprintf("Classic map %d scene transport row %s has no legacy dialogue entry", mapID, row.Handle))
		}
		if !ok {
			legacy = sourceNPCEntry{SourceQuery: "transp/flag2.swf", SpriteName: "flag2", Width: 158, Height: 258, IsGeneratedSourceTransport: true, Dialogue: &sourceTransportDialogue}
		}
		if legacy.SourceQuery != row.SourceQuery || legacy.SpriteName != row.SpriteName || legacy.Width != row.Width || legacy.Height != row.Height || (!authored && legacy.SpawnFlash != (SpawnPoint{X: row.SpawnX, Y: row.SpawnY})) {
			panic(fmt.Sprintf("Classic map %d scene transport row %s disagrees with legacy entity data", mapID, row.Handle))
		}
		delete(legacyByHandle, row.Handle)
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
	if len(legacyByHandle) != 0 {
		panic(fmt.Sprintf("Classic map %d scene transport catalog lost %d legacy entries", mapID, len(legacyByHandle)))
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
			if row.Source == "nine_wilds_terrain_20260909" {
				continue
			}
			legacyDestination, ok := resolveLegacyTownTransportDestinationFromMap(row.MapID, row.Handle)
			if !ok || legacyDestination.MapID != row.TargetMapID || legacyDestination.Spawn != (SpawnPoint{X: row.TargetSpawnX, Y: row.TargetSpawnY}) {
				panic(fmt.Sprintf("Classic map %d scene transport row %s disagrees with legacy destination", row.MapID, row.Handle))
			}
		}
	}
}
