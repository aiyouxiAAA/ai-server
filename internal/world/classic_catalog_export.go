package world

import "sort"

type ClassicMapCatalogExportRow struct {
	ID                 int
	Name               string
	XMLURL             string
	DefaultSpawn       SpawnPoint
	SupportsTransferIn bool
}

type ClassicMapNPCatalogExportRow struct {
	MapID       int
	Handle      string
	RoleID      string
	DisplayName string
	SourceQuery string
	SpriteName  string
	Width       int
	Height      int
	SpawnFlash  SpawnPoint
	State       int
	QuestState  int
	GuildName   string
	GuildPic    string
	Kind        string
}

type ClassicMapSceneTransportCatalogExportRow struct {
	MapID        int
	Handle       string
	SourceQuery  string
	SpriteName   string
	Width        int
	Height       int
	SpawnFlash   SpawnPoint
	TargetMapID  int
	TargetSpawn  SpawnPoint
	Protocol     string
	AnswerHandle string
}

type ClassicMapCollectionCatalogExportRow struct {
	Handle           string
	MapID            int
	DisplayName      string
	SourceQuery      string
	SpriteName       string
	Width            int
	Height           int
	SpawnFlash       SpawnPoint
	QuestState       int
	RequiredItemName string
	RewardItemName   string
	QuestTitle       string
	QuestDescription string
	Protocol         string
}

func ExportClassicMapCatalogRows() []ClassicMapCatalogExportRow {
	mapIDs := sortedClassicMapIDs()
	result := make([]ClassicMapCatalogExportRow, 0, len(mapIDs))
	for _, mapID := range mapIDs {
		definition := townMapBootstrapDefinitions[mapID]
		result = append(result, ClassicMapCatalogExportRow{
			ID:                 definition.ID,
			Name:               definition.Name,
			XMLURL:             definition.XMLURL,
			DefaultSpawn:       definition.DefaultSpawn,
			SupportsTransferIn: definition.SupportsTransferIn,
		})
	}
	return result
}

func ExportClassicMapNPCatalogRows() []ClassicMapNPCatalogExportRow {
	mapIDs := sortedClassicMapIDs()
	result := []ClassicMapNPCatalogExportRow{}
	for _, mapID := range mapIDs {
		definition := townMapBootstrapDefinitions[mapID]
		for _, npc := range definition.SourceNPCs {
			if npc.Kind == "collection" || npc.IsGeneratedSourceTransport || npc.RoleID == "-3" {
				continue
			}
			result = append(result, ClassicMapNPCatalogExportRow{
				MapID:       mapID,
				Handle:      npc.Handle,
				RoleID:      npc.RoleID,
				DisplayName: npc.DisplayName,
				SourceQuery: npc.SourceQuery,
				SpriteName:  npc.SpriteName,
				Width:       npc.Width,
				Height:      npc.Height,
				SpawnFlash:  npc.SpawnFlash,
				State:       npc.State,
				QuestState:  npc.QuestState,
				GuildName:   npc.GuildName,
				GuildPic:    npc.GuildPic,
				Kind:        npc.Kind,
			})
		}
	}
	return result
}

func ExportClassicMapSceneTransportCatalogRows() []ClassicMapSceneTransportCatalogExportRow {
	mapIDs := sortedClassicMapIDs()
	result := []ClassicMapSceneTransportCatalogExportRow{}
	for _, mapID := range mapIDs {
		definition := townMapBootstrapDefinitions[mapID]
		for _, npc := range definition.SourceNPCs {
			if npc.RoleID != "-3" {
				continue
			}
			destination, ok := resolveTownTransportDestinationFromLegacyData(mapID, npc.Handle)
			if !ok {
				panic("Classic scene transport has no destination: " + itoa(mapID) + "/" + npc.Handle)
			}
			result = append(result, ClassicMapSceneTransportCatalogExportRow{
				MapID:        mapID,
				Handle:       npc.Handle,
				SourceQuery:  npc.SourceQuery,
				SpriteName:   npc.SpriteName,
				Width:        npc.Width,
				Height:       npc.Height,
				SpawnFlash:   npc.SpawnFlash,
				TargetMapID:  destination.MapID,
				TargetSpawn:  destination.Spawn,
				Protocol:     "CrossRole",
				AnswerHandle: "goto",
			})
		}
	}
	return result
}

func ExportClassicMapCollectionCatalogRows() []ClassicMapCollectionCatalogExportRow {
	handles := make([]string, 0, len(sourceCollectionPointsByHandle))
	for handle := range sourceCollectionPointsByHandle {
		handles = append(handles, handle)
	}
	sort.Strings(handles)
	result := make([]ClassicMapCollectionCatalogExportRow, 0, len(handles))
	for _, handle := range handles {
		point := sourceCollectionPointsByHandle[handle]
		result = append(result, ClassicMapCollectionCatalogExportRow{
			Handle:           point.Handle,
			MapID:            point.MapID,
			DisplayName:      point.DisplayName,
			SourceQuery:      point.SourceQuery,
			SpriteName:       point.SpriteName,
			Width:            point.Width,
			Height:           point.Height,
			SpawnFlash:       point.SpawnFlash,
			QuestState:       point.QuestState,
			RequiredItemName: point.RequiredItemName,
			RewardItemName:   point.RewardItemName,
			QuestTitle:       point.QuestTitle,
			QuestDescription: point.QuestDescription,
			Protocol:         "Collection",
		})
	}
	return result
}

func sortedClassicMapIDs() []int {
	mapIDs := make([]int, 0, len(townMapBootstrapDefinitions))
	for mapID := range townMapBootstrapDefinitions {
		mapIDs = append(mapIDs, mapID)
	}
	sort.Ints(mapIDs)
	return mapIDs
}
