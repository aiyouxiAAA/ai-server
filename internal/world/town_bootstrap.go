package world

import (
	_ "embed"
	"encoding/json"
	"strconv"
	"strings"

	"ai-server/internal/session"
)

type LoadMapPush struct {
	MapID     string `json:"mapId"`
	MapName   string `json:"mapName"`
	XMLURL    string `json:"xmlUrl"`
	EnemyShow bool   `json:"enemyShow"`
}

type SpawnPoint struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type SourceNPCVisual struct {
	MovieClipIRPath string `json:"movieClipIrPath"`
	Width           int    `json:"width"`
	Height          int    `json:"height"`
	NameY           int    `json:"nameY"`
	QuestMarkerY    int    `json:"questMarkerY"`
}

type RolePush struct {
	Handle          string                 `json:"handle"`
	RoleID          string                 `json:"roleId"`
	DisplayName     string                 `json:"displayName"`
	Level           int                    `json:"level"`
	MapID           string                 `json:"mapId"`
	VisualRoleID    int                    `json:"visualRoleId"`
	PresetID        int                    `json:"presetId,omitempty"`
	SourceQuery     string                 `json:"sourceQuery,omitempty"`
	Appearance      session.RoleAppearance `json:"appearance,omitempty"`
	Kind            string                 `json:"kind"`
	SpawnFlash      SpawnPoint             `json:"spawnFlash"`
	SourceNPCVisual *SourceNPCVisual       `json:"sourceNpcVisual,omitempty"`
}

type QuestStatePush struct {
	Handle string `json:"handle"`
	State  int    `json:"state"`
}

type AnswerOption struct {
	Handle string `json:"handle"`
	Msg    string `json:"msg"`
}

type AnswerSpeakPush struct {
	Handle    string         `json:"handle"`
	MsgHandle string         `json:"msgHandle"`
	Msg       string         `json:"msg"`
	Answers   []AnswerOption `json:"answers"`
}

type TownBootstrapSnapshot struct {
	LoadMap      LoadMapPush           `json:"loadMap"`
	CreatePlayer RolePush              `json:"createPlayer"`
	CreateRoles  []RolePush            `json:"createRoles"`
	QuestStates  []QuestStatePush      `json:"questStates"`
	RoleState    *session.RoleState    `json:"roleState,omitempty"`
	RolePhysique *session.RolePhysique `json:"rolePhysique,omitempty"`
}

type sourceNPCEntry struct {
	Handle      string
	RoleID      string
	DisplayName string
	SourceQuery string
	SpriteName  string
	Width       int
	Height      int
	SpawnFlash  SpawnPoint
	QuestState  int
	Kind        string
	Dialogue    *sourceNPCDialogueEntry
}

type SourceCollectionPoint struct {
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
}

type sourceNPCDialogueEntry struct {
	MsgHandle string
	Message   string
	Answers   []AnswerOption
}

const sourceNPCMovieClipRoot = "runtime/classic-npc/movieclips"

type sourceNPCDialogueReplyKey struct {
	Handle       string
	MsgHandle    string
	AnswerHandle string
}

type townMapBootstrapDefinition struct {
	ID                 int
	Name               string
	XMLURL             string
	DefaultSpawn       SpawnPoint
	SourceNPCs         []sourceNPCEntry
	SupportsTransferIn bool
}

type TownTransportDestination struct {
	MapID int
	Spawn SpawnPoint
}

type sourceTransportLink struct {
	FromMapID int
	ToMapID   int
	Slot      int
}

var capturedTownTransportDestinations = map[string]TownTransportDestination{
	"transp_0": {MapID: 3, Spawn: SpawnPoint{X: 825, Y: 624}},
}

type townMapsIndexFile struct {
	Maps []townMapsIndexEntry `json:"maps"`
}

type townMapsIndexEntry struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SourceXML string `json:"sourceXml"`
}

var sourceWildBattleMapIDs = map[int]struct{}{
	4:  {},
	5:  {},
	84: {},
	85: {},
	86: {},
	87: {},
	88: {},
	90: {},
	97: {},
}

var sourceCollectionPointsByHandle = map[string]SourceCollectionPoint{
	"2810542613719308": {
		Handle:           "2810542613719308",
		MapID:            89,
		DisplayName:      "金银花采集点",
		SourceQuery:      "transp/flag2.swf",
		SpriteName:       "flag2",
		Width:            158,
		Height:           258,
		SpawnFlash:       SpawnPoint{X: 1242, Y: 451},
		QuestState:       2,
		RequiredItemName: "普通采集手套",
		RewardItemName:   "金银花",
		QuestTitle:       "采集金银花",
		QuestDescription: "<ml>采集金银花<br/>[g]=前往【竹林_6】采集点，携带普通采集手套采集金银花。",
	},
	"2050542611677774": {
		Handle:           "2050542611677774",
		MapID:            91,
		DisplayName:      "黄连采集点",
		SourceQuery:      "transp/flag2.swf",
		SpriteName:       "flag2",
		Width:            158,
		Height:           258,
		SpawnFlash:       SpawnPoint{X: 1755, Y: 452},
		QuestState:       2,
		RequiredItemName: "普通采集手套",
		RewardItemName:   "黄连",
		QuestTitle:       "采集黄连",
		QuestDescription: "<ml>采集黄连<br/>[g]=前往【竹林_8】采集点，携带普通采集手套采集黄连。",
	},
}

//go:embed town_maps_index.json
var townMapsIndexJSON []byte

var townMapBootstrapDefinitions = buildTownMapBootstrapDefinitions()

func buildTownMapBootstrapDefinitions() map[int]townMapBootstrapDefinition {
	definitions := loadTownMapIndexDefinitions()

	mapOne := definitions[1]
	mapOne.ID = 1
	mapOne.Name = "云隐村"
	mapOne.XMLURL = "xml/1.xml"
	mapOne.DefaultSpawn = SpawnPoint{X: 820, Y: 451}
	mapOne.SourceNPCs = map1SourceNPCs
	mapOne.SupportsTransferIn = true
	definitions[1] = mapOne

	mapTwo := definitions[2]
	mapTwo.ID = 2
	mapTwo.Name = "大佛村"
	mapTwo.XMLURL = "xml/2.xml"
	mapTwo.DefaultSpawn = SpawnPoint{X: 1000, Y: 600}
	mapTwo.SourceNPCs = map2SourceNPCs
	mapTwo.SupportsTransferIn = true
	definitions[2] = mapTwo

	mapThree := definitions[3]
	mapThree.ID = 3
	mapThree.Name = "涧庭村"
	mapThree.XMLURL = "xml/3.xml"
	mapThree.DefaultSpawn = SpawnPoint{X: 825, Y: 624}
	mapThree.SourceNPCs = map3SourceNPCs
	mapThree.SupportsTransferIn = true
	definitions[3] = mapThree

	mapFortyNine := definitions[49]
	mapFortyNine.SourceNPCs = map49SourceNPCs
	definitions[49] = mapFortyNine

	mapFifty := definitions[50]
	mapFifty.SourceNPCs = map50SourceNPCs
	definitions[50] = mapFifty

	mapSeventyNine := definitions[79]
	mapSeventyNine.SourceNPCs = map79SourceNPCs
	definitions[79] = mapSeventyNine

	mapEighty := definitions[80]
	mapEighty.SourceNPCs = map80SourceNPCs
	definitions[80] = mapEighty

	mapEightyOne := definitions[81]
	mapEightyOne.SourceNPCs = map81SourceNPCs
	definitions[81] = mapEightyOne

	mapEightyTwo := definitions[82]
	mapEightyTwo.SourceNPCs = map82SourceNPCs
	definitions[82] = mapEightyTwo

	mapEightyThree := definitions[83]
	mapEightyThree.SourceNPCs = map83SourceNPCs
	definitions[83] = mapEightyThree

	for _, point := range sourceCollectionPointsByHandle {
		mapDefinition, ok := definitions[point.MapID]
		if !ok {
			continue
		}
		if !hasSourceNPCHandle(mapDefinition.SourceNPCs, point.Handle) {
			mapDefinition.SourceNPCs = append(mapDefinition.SourceNPCs, sourceNPCEntry{
				Handle:      point.Handle,
				RoleID:      point.Handle,
				DisplayName: point.DisplayName,
				SourceQuery: point.SourceQuery,
				SpriteName:  point.SpriteName,
				Width:       point.Width,
				Height:      point.Height,
				SpawnFlash:  point.SpawnFlash,
				QuestState:  point.QuestState,
				Kind:        "collection",
			})
		}
		definitions[point.MapID] = mapDefinition
	}

	for _, link := range sourceTransportLinks {
		mapDefinition, ok := definitions[link.FromMapID]
		if !ok {
			continue
		}
		if _, ok := definitions[link.ToMapID]; !ok {
			continue
		}
		if !hasSourceNPCHandle(mapDefinition.SourceNPCs, "transp_"+itoa(link.ToMapID)) {
			mapDefinition.SourceNPCs = append(mapDefinition.SourceNPCs, buildSourceTransportNPC(link))
		}
		definitions[link.FromMapID] = mapDefinition
	}

	return definitions
}

func hasSourceNPCHandle(npcs []sourceNPCEntry, handle string) bool {
	for _, npc := range npcs {
		if npc.Handle == handle {
			return true
		}
	}
	return false
}

func loadTownMapIndexDefinitions() map[int]townMapBootstrapDefinition {
	var index townMapsIndexFile
	if err := json.Unmarshal(townMapsIndexJSON, &index); err != nil {
		panic("parse embedded town maps index: " + err.Error())
	}
	if len(index.Maps) == 0 {
		panic("embedded town maps index is empty")
	}

	definitions := make(map[int]townMapBootstrapDefinition, len(index.Maps))
	for _, entry := range index.Maps {
		mapID, err := strconv.Atoi(entry.ID)
		if err != nil || mapID <= 0 {
			panic("invalid embedded town map id: " + entry.ID)
		}

		definitions[mapID] = townMapBootstrapDefinition{
			ID:                 mapID,
			Name:               entry.Name,
			XMLURL:             normalizeTownMapXMLURL(entry.SourceXML, mapID),
			DefaultSpawn:       resolveSourceMapDefaultSpawn(entry.Name),
			SourceNPCs:         nil,
			SupportsTransferIn: true,
		}
	}

	return definitions
}

func resolveSourceMapDefaultSpawn(mapName string) SpawnPoint {
	switch mapName {
	case "平原_15":
		return SpawnPoint{X: 2862, Y: 586}
	case "平原_16":
		return SpawnPoint{X: 131, Y: 423}
	case "云隐村口_1", "云隐村口_2":
		return SpawnPoint{X: 1000, Y: 450}
	case "云隐山道_3", "云隐山道_4":
		return SpawnPoint{X: 1000, Y: 300}
	case "云隐山道_5":
		return SpawnPoint{X: 500, Y: 400}
	case "树洞":
		return SpawnPoint{X: 500, Y: 550}
	case "涧庭村口":
		return SpawnPoint{X: 180, Y: 500}
	case "涧庭道_1":
		return SpawnPoint{X: 200, Y: 482}
	case "黄泉路_1":
		return SpawnPoint{X: 2037, Y: 460}
	case "黄泉路_2":
		return SpawnPoint{X: 2800, Y: 486}
	case "黄泉路_3":
		return SpawnPoint{X: 2828, Y: 429}
	case "奈何桥":
		return SpawnPoint{X: 2809, Y: 433}
	case "重生台":
		return SpawnPoint{X: 2788, Y: 456}
	default:
		return SpawnPoint{X: 1000, Y: 600}
	}
}

func normalizeTownMapXMLURL(sourceXML string, mapID int) string {
	normalized := strings.ReplaceAll(sourceXML, "\\", "/")
	normalized = strings.TrimPrefix(normalized, "mapdefined/")
	if strings.HasPrefix(normalized, "xml/") {
		return normalized
	}
	return "xml/" + itoa(mapID) + ".xml"
}

func BuildTownBootstrap(role session.RoleSummary, playerBase session.PlayerBaseData) TownBootstrapSnapshot {
	mapID := "1"
	if playerBase.MapID > 0 {
		mapID = itoa(playerBase.MapID)
	}

	mapDefinition := resolveTownMapBootstrapDefinition(mapID)
	return buildTownBootstrap(role, playerBase, mapDefinition, mapDefinition.DefaultSpawn)
}

func BuildTownTransferBootstrap(role session.RoleSummary, playerBase session.PlayerBaseData, mapID int, spawn SpawnPoint) (TownBootstrapSnapshot, bool) {
	mapDefinition, ok := townMapBootstrapDefinitions[mapID]
	if !ok || !mapDefinition.SupportsTransferIn {
		return TownBootstrapSnapshot{}, false
	}
	return buildTownBootstrap(role, playerBase, mapDefinition, spawn), true
}

func SupportsTownTransferMap(mapID int) bool {
	mapDefinition, ok := townMapBootstrapDefinitions[mapID]
	return ok && mapDefinition.SupportsTransferIn
}

func buildTownBootstrap(
	role session.RoleSummary,
	playerBase session.PlayerBaseData,
	mapDefinition townMapBootstrapDefinition,
	spawn SpawnPoint,
) TownBootstrapSnapshot {
	mapID := itoa(mapDefinition.ID)
	createRoles := make([]RolePush, 0, len(mapDefinition.SourceNPCs))
	questStates := make([]QuestStatePush, 0, len(mapDefinition.SourceNPCs))
	for _, npc := range mapDefinition.SourceNPCs {
		roleID := npc.RoleID
		if roleID == "" {
			roleID = npc.Handle
		}
		kind := npc.Kind
		if kind == "" {
			kind = "npc"
		}
		createRoles = append(createRoles, RolePush{
			Handle:          npc.Handle,
			RoleID:          roleID,
			DisplayName:     npc.DisplayName,
			Level:           1,
			MapID:           mapID,
			VisualRoleID:    0,
			SourceQuery:     npc.SourceQuery,
			Kind:            kind,
			SpawnFlash:      npc.SpawnFlash,
			SourceNPCVisual: buildNPCVisual(npc),
		})
		questStates = append(questStates, QuestStatePush{
			Handle: npc.Handle,
			State:  npc.QuestState,
		})
	}

	return TownBootstrapSnapshot{
		LoadMap: LoadMapPush{
			MapID:     mapID,
			MapName:   mapDefinition.Name,
			XMLURL:    mapDefinition.XMLURL,
			EnemyShow: isSourceWildBattleMap(mapDefinition.ID),
		},
		CreatePlayer: RolePush{
			Handle:       role.RoleID,
			RoleID:       role.RoleID,
			DisplayName:  playerBase.DisplayName,
			Level:        playerBase.Level,
			MapID:        mapID,
			VisualRoleID: playerBase.VisualRoleID,
			PresetID:     playerBase.PresetID,
			SourceQuery:  playerBase.SourceQuery,
			Appearance:   playerBase.Appearance,
			Kind:         "self",
			SpawnFlash:   spawn,
		},
		CreateRoles:  createRoles,
		QuestStates:  questStates,
		RoleState:    playerBase.RoleState,
		RolePhysique: playerBase.RolePhysique,
	}
}

func isSourceWildBattleMap(mapID int) bool {
	_, ok := sourceWildBattleMapIDs[mapID]
	return ok
}

func resolveTownMapBootstrapDefinition(mapID string) townMapBootstrapDefinition {
	for _, definition := range townMapBootstrapDefinitions {
		if itoa(definition.ID) == mapID {
			return definition
		}
	}
	return townMapBootstrapDefinitions[1]
}

func BuildAnswerSpeak(handle string) AnswerSpeakPush {
	if npc := findSourceNPC(handle); npc != nil && npc.Dialogue != nil {
		return AnswerSpeakPush{
			Handle:    handle,
			MsgHandle: resolveDialogueMsgHandle(npc.Dialogue),
			Msg:       npc.Dialogue.Message,
			Answers:   cloneAnswerOptions(npc.Dialogue.Answers),
		}
	}

	displayName := handle
	if npc := findSourceNPC(handle); npc != nil {
		displayName = stripSourceMarkup(npc.DisplayName)
	}

	return AnswerSpeakPush{
		Handle:    handle,
		MsgHandle: "msg-" + handle + "-001",
		Msg:       displayName + "：少侠，有什么需要帮忙？",
		Answers: []AnswerOption{
			{
				Handle: "talk",
				Msg:    "交谈",
			},
			{
				Handle: "leave",
				Msg:    "离开",
			},
		},
	}
}

func FindSourceCollectionPoint(handle string) (SourceCollectionPoint, bool) {
	point, ok := sourceCollectionPointsByHandle[strings.TrimSpace(handle)]
	return point, ok
}

func BuildAnswerReply(handle string, msgHandle string, answerHandle string) *AnswerSpeakPush {
	key := sourceNPCDialogueReplyKey{
		Handle:       handle,
		MsgHandle:    msgHandle,
		AnswerHandle: answerHandle,
	}
	dialogue, ok := map1SourceNPCDialogueReplies[key]
	if !ok {
		dialogue, ok = map2SourceNPCDialogueReplies[key]
		if !ok {
			return nil
		}
	}
	if dialogue.MsgHandle == "1" && dialogue.Message == "" && dialogue.Answers == nil {
		returnValue := BuildAnswerSpeak(handle)
		return &returnValue
	}

	return &AnswerSpeakPush{
		Handle:    handle,
		MsgHandle: resolveDialogueMsgHandle(&dialogue),
		Msg:       dialogue.Message,
		Answers:   cloneAnswerOptions(dialogue.Answers),
	}
}

func ResolveTownTransportAnswer(handle string, answerHandle string) (TownTransportDestination, bool) {
	if answerHandle != "goto" && !isCapturedTownTransportConfirmAnswer(handle, answerHandle) {
		return TownTransportDestination{}, false
	}
	if destination, ok := capturedTownTransportDestinations[handle]; ok {
		if !SupportsTownTransferMap(destination.MapID) {
			return TownTransportDestination{}, false
		}
		return destination, true
	}
	mapIDText, ok := strings.CutPrefix(handle, "transp_")
	if !ok {
		return TownTransportDestination{}, false
	}
	mapID, err := strconv.Atoi(mapIDText)
	if err != nil || !SupportsTownTransferMap(mapID) {
		return TownTransportDestination{}, false
	}

	mapDefinition := townMapBootstrapDefinitions[mapID]
	return TownTransportDestination{
		MapID: mapID,
		Spawn: mapDefinition.DefaultSpawn,
	}, true
}

func isCapturedTownTransportConfirmAnswer(handle string, answerHandle string) bool {
	return handle == "transp_10" && answerHandle == "1"
}

func buildSourceTransportNPC(link sourceTransportLink) sourceNPCEntry {
	return sourceNPCEntry{
		Handle:      "transp_" + itoa(link.ToMapID),
		RoleID:      "-3",
		DisplayName: "",
		SourceQuery: "transp/flag2.swf",
		SpriteName:  "flag2",
		Width:       158,
		Height:      258,
		SpawnFlash:  sourceTransportSlotSpawn(link.FromMapID, link.Slot),
		QuestState:  0,
		Dialogue:    &sourceTransportDialogue,
	}
}

func sourceTransportSlotSpawn(mapID int, slot int) SpawnPoint {
	const (
		edgeInsetX = 290
		defaultY   = 520
		branchY    = 430
	)

	mapWidth := sourceTransportMapWidth(mapID)
	switch slot {
	case 0:
		return SpawnPoint{X: edgeInsetX, Y: defaultY}
	case 1:
		return SpawnPoint{X: mapWidth - edgeInsetX, Y: defaultY}
	case 2:
		return SpawnPoint{X: mapWidth / 2, Y: branchY}
	default:
		return SpawnPoint{X: mapWidth / 2, Y: defaultY}
	}
}

func sourceTransportMapWidth(mapID int) int {
	switch mapID {
	case 4, 9, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 32:
		return 3000
	case 5:
		return 2903
	case 13, 18:
		return 1500
	case 14:
		return 1400
	case 15:
		return 1300
	case 16:
		return 1200
	case 17:
		return 1100
	case 30:
		return 1000
	default:
		return 2000
	}
}

func resolveDialogueMsgHandle(dialogue *sourceNPCDialogueEntry) string {
	if dialogue != nil && dialogue.MsgHandle != "" {
		return dialogue.MsgHandle
	}
	return "1"
}

func findSourceNPC(handle string) *sourceNPCEntry {
	for _, mapDefinition := range townMapBootstrapDefinitions {
		for index := range mapDefinition.SourceNPCs {
			if mapDefinition.SourceNPCs[index].Handle == handle {
				return &mapDefinition.SourceNPCs[index]
			}
		}
	}

	return nil
}

func cloneAnswerOptions(answers []AnswerOption) []AnswerOption {
	if len(answers) == 0 {
		return nil
	}

	result := make([]AnswerOption, len(answers))
	copy(result, answers)
	return result
}

func buildNPCVisual(npc sourceNPCEntry) *SourceNPCVisual {
	return &SourceNPCVisual{
		MovieClipIRPath: sourceNPCMovieClipRoot + "/" + npc.SpriteName + "/" + npc.SpriteName + "-movieclip-ir",
		Width:           npc.Width,
		Height:          npc.Height,
		NameY:           npc.Height + 18,
		QuestMarkerY:    npc.Height + 62,
	}
}

func stripSourceMarkup(value string) string {
	result := make([]rune, 0, len(value))
	inTag := false
	for _, char := range value {
		switch char {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				result = append(result, char)
			}
		}
	}
	return string(result)
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}

	negative := value < 0
	if negative {
		value = -value
	}

	var digits [16]byte
	index := len(digits)
	for value > 0 {
		index--
		digits[index] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		index--
		digits[index] = '-'
	}
	return string(digits[index:])
}
