package classicdata

import (
	"embed"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
)

//go:embed classic_map_catalog.csv classic_map_npc_catalog.csv classic_map_scene_transport_catalog.csv classic_map_collection_catalog.csv classic_map_page_transfer_catalog.csv
var classicWorldCatalogFS embed.FS

type ClassicMap struct {
	ID                 int
	Name               string
	XMLURL             string
	DefaultSpawnX      int
	DefaultSpawnY      int
	SupportsTransferIn bool
}

type ClassicMapNPCSpawn struct {
	MapID       int
	Handle      string
	RoleID      string
	DisplayName string
	SourceQuery string
	SpriteName  string
	Width       int
	Height      int
	SpawnX      int
	SpawnY      int
	State       int
	QuestState  int
	GuildName   string
	GuildPic    string
	Kind        string
}

type ClassicMapSceneTransportSpawn struct {
	MapID        int
	Handle       string
	SourceQuery  string
	SpriteName   string
	Width        int
	Height       int
	SpawnX       int
	SpawnY       int
	TargetMapID  int
	TargetSpawnX int
	TargetSpawnY int
	Protocol     string
	AnswerHandle string
}

type ClassicMapCollectionPoint struct {
	MapID            int
	Handle           string
	DisplayName      string
	SourceQuery      string
	SpriteName       string
	Width            int
	Height           int
	SpawnX           int
	SpawnY           int
	QuestState       int
	RequiredItemName string
	RewardItemName   string
	QuestTitle       string
	QuestDescription string
	Protocol         string
}

type ClassicMapPageTransfer struct {
	TransferType string
	TransferKey  string
	MapID        int
	MapName      string
	LandingX     int
	LandingY     int
	Source       string
}

type classicWorldCatalog struct {
	maps                     []ClassicMap
	mapByID                  map[int]ClassicMap
	npcSpawns                []ClassicMapNPCSpawn
	npcsByMapID              map[int][]ClassicMapNPCSpawn
	npcByMapAndID            map[string]ClassicMapNPCSpawn
	npcsByHandle             map[string][]ClassicMapNPCSpawn
	sceneTransports          []ClassicMapSceneTransportSpawn
	sceneTransportsByMapID   map[int][]ClassicMapSceneTransportSpawn
	sceneTransportByMapAndID map[string]ClassicMapSceneTransportSpawn
	collectionPoints         []ClassicMapCollectionPoint
	collectionsByMapID       map[int][]ClassicMapCollectionPoint
	collectionByMapAndID     map[string]ClassicMapCollectionPoint
	collectionsByHandle      map[string][]ClassicMapCollectionPoint
	pageTransfers            []ClassicMapPageTransfer
}

var loadedClassicWorldCatalog = loadClassicWorldCatalog()

func ClassicMaps() []ClassicMap {
	return append([]ClassicMap(nil), loadedClassicWorldCatalog.maps...)
}

func FindClassicMap(mapID int) (ClassicMap, bool) {
	entry, ok := loadedClassicWorldCatalog.mapByID[mapID]
	return entry, ok
}

func ClassicMapNPCSpawns(mapID int) []ClassicMapNPCSpawn {
	return append([]ClassicMapNPCSpawn(nil), loadedClassicWorldCatalog.npcsByMapID[mapID]...)
}

func FindClassicMapNPCSpawn(mapID int, handle string) (ClassicMapNPCSpawn, bool) {
	entry, ok := loadedClassicWorldCatalog.npcByMapAndID[classicMapNPCKey(mapID, handle)]
	return entry, ok
}

func FindClassicMapNPCSpawnsByHandle(handle string) []ClassicMapNPCSpawn {
	return append([]ClassicMapNPCSpawn(nil), loadedClassicWorldCatalog.npcsByHandle[strings.TrimSpace(handle)]...)
}

func ClassicMapSceneTransportSpawns(mapID int) []ClassicMapSceneTransportSpawn {
	return append([]ClassicMapSceneTransportSpawn(nil), loadedClassicWorldCatalog.sceneTransportsByMapID[mapID]...)
}

func FindClassicMapSceneTransportSpawn(mapID int, handle string) (ClassicMapSceneTransportSpawn, bool) {
	entry, ok := loadedClassicWorldCatalog.sceneTransportByMapAndID[classicMapNPCKey(mapID, handle)]
	return entry, ok
}

func ClassicMapCollectionPoints(mapID int) []ClassicMapCollectionPoint {
	return append([]ClassicMapCollectionPoint(nil), loadedClassicWorldCatalog.collectionsByMapID[mapID]...)
}

func FindClassicMapCollectionPoint(mapID int, handle string) (ClassicMapCollectionPoint, bool) {
	entry, ok := loadedClassicWorldCatalog.collectionByMapAndID[classicMapNPCKey(mapID, handle)]
	return entry, ok
}

func FindClassicMapCollectionPointsByHandle(handle string) []ClassicMapCollectionPoint {
	return append([]ClassicMapCollectionPoint(nil), loadedClassicWorldCatalog.collectionsByHandle[strings.TrimSpace(handle)]...)
}

func ClassicMapPageTransfers() []ClassicMapPageTransfer {
	return append([]ClassicMapPageTransfer(nil), loadedClassicWorldCatalog.pageTransfers...)
}

type WuliangMap = ClassicMap
type WuliangMapNPCSpawn = ClassicMapNPCSpawn

func WuliangMaps() []WuliangMap {
	result := []WuliangMap{}
	for _, entry := range ClassicMaps() {
		if entry.ID == 190 || entry.ID == 191 {
			result = append(result, entry)
		}
	}
	return result
}

func FindWuliangMap(mapID int) (WuliangMap, bool) {
	if mapID != 190 && mapID != 191 {
		return WuliangMap{}, false
	}
	return FindClassicMap(mapID)
}

func WuliangMapNPCSpawns(mapID int) []WuliangMapNPCSpawn {
	if mapID != 190 && mapID != 191 {
		return nil
	}
	return ClassicMapNPCSpawns(mapID)
}

func FindWuliangMapNPCSpawn(handle string) (WuliangMapNPCSpawn, bool) {
	for _, entry := range FindClassicMapNPCSpawnsByHandle(handle) {
		if entry.MapID == 190 || entry.MapID == 191 {
			return entry, true
		}
	}
	return WuliangMapNPCSpawn{}, false
}

func FindWuliangMapForNPCHandle(handle string) (WuliangMap, bool) {
	npc, ok := FindWuliangMapNPCSpawn(handle)
	if !ok {
		return WuliangMap{}, false
	}
	return FindWuliangMap(npc.MapID)
}

func loadClassicWorldCatalog() classicWorldCatalog {
	maps, err := readClassicMaps()
	if err != nil {
		panic("load Classic map catalog: " + err.Error())
	}
	mapByID := make(map[int]ClassicMap, len(maps))
	for _, entry := range maps {
		if entry.ID <= 0 || entry.XMLURL == "" {
			panic(fmt.Sprintf("invalid Classic map catalog row: %+v", entry))
		}
		if _, exists := mapByID[entry.ID]; exists {
			panic(fmt.Sprintf("duplicate Classic map id: %d", entry.ID))
		}
		mapByID[entry.ID] = entry
	}

	npcSpawns, err := readClassicMapNPCSpawns()
	if err != nil {
		panic("load Classic map NPC catalog: " + err.Error())
	}
	npcsByMapID := make(map[int][]ClassicMapNPCSpawn)
	npcByMapAndID := make(map[string]ClassicMapNPCSpawn, len(npcSpawns))
	npcsByHandle := make(map[string][]ClassicMapNPCSpawn)
	for _, entry := range npcSpawns {
		if _, exists := mapByID[entry.MapID]; !exists {
			panic(fmt.Sprintf("Classic NPC %s references unknown map %d", entry.Handle, entry.MapID))
		}
		if entry.Handle == "" || entry.DisplayName == "" || entry.SourceQuery == "" || entry.SpriteName == "" {
			panic(fmt.Sprintf("invalid Classic map NPC catalog row: %+v", entry))
		}
		key := classicMapNPCKey(entry.MapID, entry.Handle)
		if _, exists := npcByMapAndID[key]; exists {
			panic("duplicate Classic map NPC key: " + key)
		}
		npcByMapAndID[key] = entry
		npcsByMapID[entry.MapID] = append(npcsByMapID[entry.MapID], entry)
		npcsByHandle[entry.Handle] = append(npcsByHandle[entry.Handle], entry)
	}

	sceneTransports, err := readClassicMapSceneTransportSpawns()
	if err != nil {
		panic("load Classic map scene transport catalog: " + err.Error())
	}
	sceneTransportsByMapID := make(map[int][]ClassicMapSceneTransportSpawn)
	sceneTransportByMapAndID := make(map[string]ClassicMapSceneTransportSpawn, len(sceneTransports))
	for _, entry := range sceneTransports {
		if _, exists := mapByID[entry.MapID]; !exists {
			panic(fmt.Sprintf("Classic scene transport %s references unknown source map %d", entry.Handle, entry.MapID))
		}
		if _, exists := mapByID[entry.TargetMapID]; !exists {
			panic(fmt.Sprintf("Classic scene transport %s references unknown target map %d", entry.Handle, entry.TargetMapID))
		}
		if entry.Handle == "" || entry.SourceQuery == "" || entry.SpriteName == "" || entry.Protocol != "CrossRole" || entry.AnswerHandle == "" {
			panic(fmt.Sprintf("invalid Classic scene transport catalog row: %+v", entry))
		}
		key := classicMapNPCKey(entry.MapID, entry.Handle)
		if _, exists := sceneTransportByMapAndID[key]; exists {
			panic("duplicate Classic scene transport key: " + key)
		}
		sceneTransportByMapAndID[key] = entry
		sceneTransportsByMapID[entry.MapID] = append(sceneTransportsByMapID[entry.MapID], entry)
	}

	collectionPoints, err := readClassicMapCollectionPoints()
	if err != nil {
		panic("load Classic map collection catalog: " + err.Error())
	}
	collectionsByMapID := make(map[int][]ClassicMapCollectionPoint)
	collectionByMapAndID := make(map[string]ClassicMapCollectionPoint, len(collectionPoints))
	collectionsByHandle := make(map[string][]ClassicMapCollectionPoint)
	for _, entry := range collectionPoints {
		if _, exists := mapByID[entry.MapID]; !exists {
			panic(fmt.Sprintf("Classic collection %s references unknown map %d", entry.Handle, entry.MapID))
		}
		if entry.Handle == "" || entry.DisplayName == "" || entry.SourceQuery == "" || entry.SpriteName == "" || entry.Protocol != "Collection" {
			panic(fmt.Sprintf("invalid Classic collection catalog row: %+v", entry))
		}
		key := classicMapNPCKey(entry.MapID, entry.Handle)
		if _, exists := collectionByMapAndID[key]; exists {
			panic("duplicate Classic collection key: " + key)
		}
		collectionByMapAndID[key] = entry
		collectionsByMapID[entry.MapID] = append(collectionsByMapID[entry.MapID], entry)
		collectionsByHandle[entry.Handle] = append(collectionsByHandle[entry.Handle], entry)
	}

	pageTransfers, err := readClassicMapPageTransfers()
	if err != nil {
		panic("load Classic map page transfer catalog: " + err.Error())
	}
	for _, entry := range pageTransfers {
		if _, exists := mapByID[entry.MapID]; !exists {
			panic(fmt.Sprintf("Classic map page transfer %s/%s references unknown map %d", entry.TransferType, entry.TransferKey, entry.MapID))
		}
		if entry.TransferType != "mapbox" && entry.TransferType != "handle" && entry.TransferType != "spawn" && entry.TransferType != "role" {
			panic(fmt.Sprintf("invalid Classic map page transfer type: %+v", entry))
		}
		if entry.TransferKey == "" || entry.Source == "" {
			panic(fmt.Sprintf("invalid Classic map page transfer catalog row: %+v", entry))
		}
		if entry.TransferType != "role" && entry.MapName == "" {
			panic(fmt.Sprintf("Classic %s transfer %s has no map name", entry.TransferType, entry.TransferKey))
		}
	}
	return classicWorldCatalog{
		maps:                     maps,
		mapByID:                  mapByID,
		npcSpawns:                npcSpawns,
		npcsByMapID:              npcsByMapID,
		npcByMapAndID:            npcByMapAndID,
		npcsByHandle:             npcsByHandle,
		sceneTransports:          sceneTransports,
		sceneTransportsByMapID:   sceneTransportsByMapID,
		sceneTransportByMapAndID: sceneTransportByMapAndID,
		collectionPoints:         collectionPoints,
		collectionsByMapID:       collectionsByMapID,
		collectionByMapAndID:     collectionByMapAndID,
		collectionsByHandle:      collectionsByHandle,
		pageTransfers:            pageTransfers,
	}
}

func readClassicMaps() ([]ClassicMap, error) {
	rows, err := readClassicCatalogRows("classic_map_catalog.csv")
	if err != nil {
		return nil, err
	}
	result := make([]ClassicMap, 0, len(rows))
	for _, row := range rows {
		result = append(result, ClassicMap{
			ID:                 readClassicInt(row, "map_id"),
			Name:               readClassicField(row, "name"),
			XMLURL:             readClassicField(row, "xml_url"),
			DefaultSpawnX:      readClassicInt(row, "default_spawn_x"),
			DefaultSpawnY:      readClassicInt(row, "default_spawn_y"),
			SupportsTransferIn: readClassicBool(row, "supports_transfer_in"),
		})
	}
	return result, nil
}

func readClassicMapNPCSpawns() ([]ClassicMapNPCSpawn, error) {
	rows, err := readClassicCatalogRows("classic_map_npc_catalog.csv")
	if err != nil {
		return nil, err
	}
	result := make([]ClassicMapNPCSpawn, 0, len(rows))
	for _, row := range rows {
		result = append(result, ClassicMapNPCSpawn{
			MapID:       readClassicInt(row, "map_id"),
			Handle:      readClassicField(row, "handle"),
			RoleID:      readClassicField(row, "role_id"),
			DisplayName: readClassicField(row, "display_name"),
			SourceQuery: readClassicField(row, "source_query"),
			SpriteName:  readClassicField(row, "sprite_name"),
			Width:       readClassicInt(row, "width"),
			Height:      readClassicInt(row, "height"),
			SpawnX:      readClassicInt(row, "spawn_x"),
			SpawnY:      readClassicInt(row, "spawn_y"),
			State:       readClassicInt(row, "state"),
			QuestState:  readClassicInt(row, "quest_state"),
			GuildName:   readClassicField(row, "guild_name"),
			GuildPic:    readClassicField(row, "guild_pic"),
			Kind:        readClassicField(row, "kind"),
		})
	}
	return result, nil
}

func readClassicMapSceneTransportSpawns() ([]ClassicMapSceneTransportSpawn, error) {
	rows, err := readClassicCatalogRows("classic_map_scene_transport_catalog.csv")
	if err != nil {
		return nil, err
	}
	result := make([]ClassicMapSceneTransportSpawn, 0, len(rows))
	for _, row := range rows {
		result = append(result, ClassicMapSceneTransportSpawn{
			MapID:        readClassicInt(row, "map_id"),
			Handle:       readClassicField(row, "handle"),
			SourceQuery:  readClassicField(row, "source_query"),
			SpriteName:   readClassicField(row, "sprite_name"),
			Width:        readClassicInt(row, "width"),
			Height:       readClassicInt(row, "height"),
			SpawnX:       readClassicInt(row, "spawn_x"),
			SpawnY:       readClassicInt(row, "spawn_y"),
			TargetMapID:  readClassicInt(row, "target_map_id"),
			TargetSpawnX: readClassicInt(row, "target_spawn_x"),
			TargetSpawnY: readClassicInt(row, "target_spawn_y"),
			Protocol:     readClassicField(row, "protocol"),
			AnswerHandle: readClassicField(row, "answer_handle"),
		})
	}
	return result, nil
}

func readClassicMapCollectionPoints() ([]ClassicMapCollectionPoint, error) {
	rows, err := readClassicCatalogRows("classic_map_collection_catalog.csv")
	if err != nil {
		return nil, err
	}
	result := make([]ClassicMapCollectionPoint, 0, len(rows))
	for _, row := range rows {
		result = append(result, ClassicMapCollectionPoint{
			MapID:            readClassicInt(row, "map_id"),
			Handle:           readClassicField(row, "handle"),
			DisplayName:      readClassicField(row, "display_name"),
			SourceQuery:      readClassicField(row, "source_query"),
			SpriteName:       readClassicField(row, "sprite_name"),
			Width:            readClassicInt(row, "width"),
			Height:           readClassicInt(row, "height"),
			SpawnX:           readClassicInt(row, "spawn_x"),
			SpawnY:           readClassicInt(row, "spawn_y"),
			QuestState:       readClassicInt(row, "quest_state"),
			RequiredItemName: readClassicField(row, "required_item_name"),
			RewardItemName:   readClassicField(row, "reward_item_name"),
			QuestTitle:       readClassicField(row, "quest_title"),
			QuestDescription: readClassicField(row, "quest_description"),
			Protocol:         readClassicField(row, "protocol"),
		})
	}
	return result, nil
}

func readClassicMapPageTransfers() ([]ClassicMapPageTransfer, error) {
	rows, err := readClassicCatalogRows("classic_map_page_transfer_catalog.csv")
	if err != nil {
		return nil, err
	}
	result := make([]ClassicMapPageTransfer, 0, len(rows))
	for _, row := range rows {
		result = append(result, ClassicMapPageTransfer{
			TransferType: readClassicField(row, "transfer_type"),
			TransferKey:  readClassicField(row, "transfer_key"),
			MapID:        readClassicInt(row, "map_id"),
			MapName:      readClassicField(row, "map_name"),
			LandingX:     readClassicInt(row, "landing_x"),
			LandingY:     readClassicInt(row, "landing_y"),
			Source:       readClassicField(row, "source"),
		})
	}
	return result, nil
}

func readClassicCatalogRows(name string) ([]map[string]string, error) {
	file, err := classicWorldCatalogFS.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	headers, err := reader.Read()
	if err != nil {
		return nil, err
	}
	normalizedHeaders := make([]string, len(headers))
	for index, header := range headers {
		normalizedHeaders[index] = strings.TrimSpace(header)
	}
	rows := []map[string]string{}
	for {
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, readErr
		}
		row := make(map[string]string, len(normalizedHeaders))
		for index, header := range normalizedHeaders {
			if index < len(record) {
				row[header] = strings.TrimSpace(record[index])
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func readClassicField(row map[string]string, field string) string {
	return strings.TrimSpace(row[field])
}

func readClassicInt(row map[string]string, field string) int {
	value := readClassicField(row, field)
	if value == "" {
		return 0
	}
	result, err := strconv.Atoi(value)
	if err != nil {
		panic(fmt.Sprintf("invalid Classic catalog %s=%q", field, value))
	}
	return result
}

func readClassicBool(row map[string]string, field string) bool {
	value := readClassicField(row, field)
	result, err := strconv.ParseBool(value)
	if err != nil {
		panic(fmt.Sprintf("invalid Classic catalog %s=%q", field, value))
	}
	return result
}

func classicMapNPCKey(mapID int, handle string) string {
	return strconv.Itoa(mapID) + ":" + strings.TrimSpace(handle)
}
