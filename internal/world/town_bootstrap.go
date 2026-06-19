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

type RoleMovement struct {
	Speed int     `json:"speed"`
	Angle float64 `json:"angle"`
	Mode  int     `json:"mode"`
}

type RoleMovePush struct {
	Handle string `json:"handle"`
	Type   string `json:"type"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Z      int    `json:"z"`
	TX     int    `json:"tx"`
	TY     int    `json:"ty"`
	TZ     int    `json:"tz"`
	MapID  string `json:"mapId,omitempty"`
}

type RolePush struct {
	Handle          string                 `json:"handle"`
	RoleID          string                 `json:"roleId"`
	DisplayName     string                 `json:"displayName"`
	Level           int                    `json:"level"`
	Vocation        string                 `json:"vocation,omitempty"`
	MapID           string                 `json:"mapId"`
	VisualRoleID    int                    `json:"visualRoleId"`
	PresetID        int                    `json:"presetId,omitempty"`
	SourceQuery     string                 `json:"sourceQuery,omitempty"`
	Appearance      session.RoleAppearance `json:"appearance,omitempty"`
	Kind            string                 `json:"kind"`
	SpawnFlash      SpawnPoint             `json:"spawnFlash"`
	SourceNPCVisual *SourceNPCVisual       `json:"sourceNpcVisual,omitempty"`
	Movement        *RoleMovement          `json:"movement,omitempty"`
	PK              int                    `json:"pk,omitempty"`
	State           int                    `json:"state,omitempty"`
	GuildName       string                 `json:"guildName,omitempty"`
	GuildPic        string                 `json:"guildPic,omitempty"`
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
	Handle                     string
	RoleID                     string
	DisplayName                string
	SourceQuery                string
	SpriteName                 string
	Width                      int
	Height                     int
	SpawnFlash                 SpawnPoint
	QuestState                 int
	Kind                       string
	IsGeneratedSourceTransport bool
	Dialogue                   *sourceNPCDialogueEntry
}

type sourceMonsterEntry struct {
	Handle      string
	DisplayName string
	SourceQuery string
	SpriteName  string
	Level       int
	Vocation    string
	Width       int
	Height      int
	SpawnFlash  SpawnPoint
	Movement    RoleMovement
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

const (
	sourceNPCMovieClipRoot     = "runtime/classic-npc/movieclips"
	sourceMonsterMovieClipRoot = "runtime/classic-monstermap"
)

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
	SourceMonsters     []sourceMonsterEntry
	SupportsTransferIn bool
}

type TownTransportDestination struct {
	MapID int
	Spawn SpawnPoint
}

type townTransportRouteKey struct {
	FromMapID int
	Handle    string
}

type sourceTransportLink struct {
	FromMapID int
	ToMapID   int
	Slot      int
}

var capturedTownTransportDestinations = map[string]TownTransportDestination{
	"transp_0":  {MapID: 3, Spawn: SpawnPoint{X: 825, Y: 624}},
	"transp_64": {MapID: 64, Spawn: SpawnPoint{X: 125, Y: 431}},
}

var capturedTownTransportRouteDestinations = map[townTransportRouteKey]TownTransportDestination{
	{FromMapID: 18, Handle: "transp_64"}: {MapID: 64, Spawn: SpawnPoint{X: 125, Y: 431}},
	{FromMapID: 64, Handle: "transp_65"}: {MapID: 65, Spawn: SpawnPoint{X: 125, Y: 437}},
	{FromMapID: 65, Handle: "transp_66"}: {MapID: 66, Spawn: SpawnPoint{X: 126, Y: 429}},
	{FromMapID: 66, Handle: "transp_67"}: {MapID: 67, Spawn: SpawnPoint{X: 145, Y: 494}},
	{FromMapID: 67, Handle: "transp_68"}: {MapID: 68, Spawn: SpawnPoint{X: 129, Y: 471}},
	{FromMapID: 68, Handle: "transp_69"}: {MapID: 69, Spawn: SpawnPoint{X: 129, Y: 497}},
	{FromMapID: 69, Handle: "transp_71"}: {MapID: 71, Spawn: SpawnPoint{X: 129, Y: 471}},
	{FromMapID: 71, Handle: "transp_73"}: {MapID: 73, Spawn: SpawnPoint{X: 126, Y: 394}},
	{FromMapID: 73, Handle: "transp_72"}: {MapID: 72, Spawn: SpawnPoint{X: 2372, Y: 443}},
	{FromMapID: 72, Handle: "transp_74"}: {MapID: 74, Spawn: SpawnPoint{X: 2809, Y: 422}},
	{FromMapID: 73, Handle: "transp_77"}: {MapID: 77, Spawn: SpawnPoint{X: 125, Y: 450}},
	{FromMapID: 77, Handle: "transp_78"}: {MapID: 78, Spawn: SpawnPoint{X: 88, Y: 570}},
	{FromMapID: 78, Handle: "transp_77"}: {MapID: 77, Spawn: SpawnPoint{X: 2607, Y: 430}},
	{FromMapID: 77, Handle: "transp_73"}: {MapID: 73, Spawn: SpawnPoint{X: 2852, Y: 529}},
	{FromMapID: 74, Handle: "transp_75"}: {MapID: 75, Spawn: SpawnPoint{X: 1332, Y: 394}},
	{FromMapID: 75, Handle: "transp_76"}: {MapID: 76, Spawn: SpawnPoint{X: 129, Y: 525}},
	{FromMapID: 76, Handle: "transp_18"}: {MapID: 18, Spawn: SpawnPoint{X: 600, Y: 300}},
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
	11: {},
	34: {},
	35: {},
	36: {},
	37: {},
	39: {},
	40: {},
	41: {},
	43: {},
	44: {},
	48: {},
	49: {},
	50: {},
	51: {},
	52: {},
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

	mapThirtyThree := definitions[33]
	mapThirtyThree.SourceNPCs = map33SourceNPCs
	definitions[33] = mapThirtyThree

	mapFortyNine := definitions[49]
	mapFortyNine.SourceNPCs = map49SourceNPCs
	definitions[49] = mapFortyNine

	mapFifty := definitions[50]
	mapFifty.SourceNPCs = map50SourceNPCs
	definitions[50] = mapFifty

	mapFortyFive := definitions[45]
	mapFortyFive.SourceNPCs = map45SourceNPCs
	definitions[45] = mapFortyFive

	mapFortySix := definitions[46]
	mapFortySix.SourceNPCs = map46SourceNPCs
	definitions[46] = mapFortySix

	mapFortySeven := definitions[47]
	mapFortySeven.SourceNPCs = map47SourceNPCs
	definitions[47] = mapFortySeven

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

	mapSixtyFour := definitions[64]
	mapSixtyFour.SourceNPCs = map64SourceNPCs
	mapSixtyFour.SourceMonsters = map64SourceMonsters
	definitions[64] = mapSixtyFour

	mapSixtyFive := definitions[65]
	mapSixtyFive.SourceNPCs = map65SourceNPCs
	mapSixtyFive.SourceMonsters = map65SourceMonsters
	definitions[65] = mapSixtyFive

	mapSixtySix := definitions[66]
	mapSixtySix.SourceNPCs = map66SourceNPCs
	mapSixtySix.SourceMonsters = map66SourceMonsters
	definitions[66] = mapSixtySix

	mapSixtySeven := definitions[67]
	mapSixtySeven.SourceNPCs = map67SourceNPCs
	mapSixtySeven.SourceMonsters = map67SourceMonsters
	definitions[67] = mapSixtySeven

	mapSixtyEight := definitions[68]
	mapSixtyEight.SourceNPCs = map68SourceNPCs
	mapSixtyEight.SourceMonsters = map68SourceMonsters
	definitions[68] = mapSixtyEight

	mapSixtyNine := definitions[69]
	mapSixtyNine.SourceNPCs = map69SourceNPCs
	mapSixtyNine.SourceMonsters = map69SourceMonsters
	definitions[69] = mapSixtyNine

	mapSeventyOne := definitions[71]
	mapSeventyOne.SourceNPCs = map71SourceNPCs
	mapSeventyOne.SourceMonsters = map71SourceMonsters
	definitions[71] = mapSeventyOne

	mapSeventyTwo := definitions[72]
	mapSeventyTwo.SourceNPCs = map72SourceNPCs
	mapSeventyTwo.SourceMonsters = map72SourceMonsters
	definitions[72] = mapSeventyTwo

	mapSeventyThree := definitions[73]
	mapSeventyThree.SourceNPCs = map73SourceNPCs
	mapSeventyThree.SourceMonsters = map73SourceMonsters
	definitions[73] = mapSeventyThree

	mapSeventyFour := definitions[74]
	mapSeventyFour.SourceNPCs = map74SourceNPCs
	mapSeventyFour.SourceMonsters = map74SourceMonsters
	definitions[74] = mapSeventyFour

	mapSeventyFive := definitions[75]
	mapSeventyFive.SourceNPCs = map75SourceNPCs
	mapSeventyFive.SourceMonsters = map75SourceMonsters
	definitions[75] = mapSeventyFive

	mapSeventySix := definitions[76]
	mapSeventySix.SourceNPCs = map76SourceNPCs
	mapSeventySix.SourceMonsters = map76SourceMonsters
	definitions[76] = mapSeventySix

	mapSeventySeven := definitions[77]
	mapSeventySeven.SourceNPCs = map77SourceNPCs
	mapSeventySeven.SourceMonsters = map77SourceMonsters
	definitions[77] = mapSeventySeven

	mapSeventyEight := definitions[78]
	mapSeventyEight.SourceNPCs = map78SourceNPCs
	mapSeventyEight.SourceMonsters = map78SourceMonsters
	definitions[78] = mapSeventyEight

	mapOneTwentySeven := definitions[127]
	mapOneTwentySeven.SourceNPCs = map127SourceNPCs
	definitions[127] = mapOneTwentySeven

	mapOneThirtyOne := definitions[131]
	mapOneThirtyOne.SourceNPCs = map131SourceNPCs
	mapOneThirtyOne.SourceMonsters = map131SourceMonsters
	definitions[131] = mapOneThirtyOne

	mapOneThirtyTwo := definitions[132]
	mapOneThirtyTwo.SourceNPCs = map132SourceNPCs
	mapOneThirtyTwo.SourceMonsters = map132SourceMonsters
	definitions[132] = mapOneThirtyTwo

	mapOneThirtyThree := definitions[133]
	mapOneThirtyThree.SourceNPCs = map133SourceNPCs
	mapOneThirtyThree.SourceMonsters = map133SourceMonsters
	definitions[133] = mapOneThirtyThree

	mapOneThirtySeven := definitions[137]
	mapOneThirtySeven.SourceNPCs = map137SourceNPCs
	mapOneThirtySeven.SourceMonsters = map137SourceMonsters
	definitions[137] = mapOneThirtySeven

	mapOneForty := definitions[140]
	mapOneForty.SourceNPCs = map140SourceNPCs
	mapOneForty.SourceMonsters = map140SourceMonsters
	definitions[140] = mapOneForty

	mapOneFortyOne := definitions[141]
	mapOneFortyOne.SourceNPCs = map141SourceNPCs
	mapOneFortyOne.SourceMonsters = map141SourceMonsters
	definitions[141] = mapOneFortyOne

	mapOneFortyTwo := definitions[142]
	mapOneFortyTwo.SourceNPCs = map142SourceNPCs
	mapOneFortyTwo.SourceMonsters = map142SourceMonsters
	definitions[142] = mapOneFortyTwo

	mapOneFortyThree := definitions[143]
	mapOneFortyThree.SourceNPCs = map143SourceNPCs
	mapOneFortyThree.SourceMonsters = map143SourceMonsters
	definitions[143] = mapOneFortyThree

	mapOneFortyFour := definitions[144]
	mapOneFortyFour.SourceNPCs = map144SourceNPCs
	mapOneFortyFour.SourceMonsters = map144SourceMonsters
	definitions[144] = mapOneFortyFour

	mapOneFortyFive := definitions[145]
	mapOneFortyFive.SourceNPCs = map145SourceNPCs
	mapOneFortyFive.SourceMonsters = map145SourceMonsters
	definitions[145] = mapOneFortyFive

	mapOneTwentyTwo := definitions[122]
	mapOneTwentyTwo.SourceNPCs = map122SourceNPCs
	definitions[122] = mapOneTwentyTwo

	mapOneFortySix := definitions[146]
	mapOneFortySix.SourceNPCs = map146SourceNPCs
	mapOneFortySix.SourceMonsters = map146SourceMonsters
	definitions[146] = mapOneFortySix

	mapOneFortyEight := definitions[148]
	mapOneFortyEight.SourceNPCs = map148SourceNPCs
	mapOneFortyEight.SourceMonsters = map148SourceMonsters
	definitions[148] = mapOneFortyEight

	mapOneFortyNine := definitions[149]
	mapOneFortyNine.SourceNPCs = map149SourceNPCs
	mapOneFortyNine.SourceMonsters = map149SourceMonsters
	definitions[149] = mapOneFortyNine

	mapOneFifty := definitions[150]
	mapOneFifty.SourceNPCs = map150SourceNPCs
	mapOneFifty.SourceMonsters = map150SourceMonsters
	definitions[150] = mapOneFifty

	mapOneFiftyOne := definitions[151]
	mapOneFiftyOne.SourceNPCs = map151SourceNPCs
	mapOneFiftyOne.SourceMonsters = map151SourceMonsters
	definitions[151] = mapOneFiftyOne

	mapOneFiftyTwo := definitions[152]
	mapOneFiftyTwo.SourceNPCs = map152SourceNPCs
	mapOneFiftyTwo.SourceMonsters = map152SourceMonsters
	definitions[152] = mapOneFiftyTwo

	mapOneFiftyThree := definitions[153]
	mapOneFiftyThree.SourceNPCs = map153SourceNPCs
	mapOneFiftyThree.SourceMonsters = map153SourceMonsters
	definitions[153] = mapOneFiftyThree

	mapOneFiftyFour := definitions[154]
	mapOneFiftyFour.SourceNPCs = map154SourceNPCs
	mapOneFiftyFour.SourceMonsters = map154SourceMonsters
	definitions[154] = mapOneFiftyFour

	mapOneFiftyFive := definitions[155]
	mapOneFiftyFive.SourceNPCs = map155SourceNPCs
	mapOneFiftyFive.SourceMonsters = map155SourceMonsters
	definitions[155] = mapOneFiftyFive

	mapOneFiftySix := definitions[156]
	mapOneFiftySix.SourceNPCs = map156SourceNPCs
	mapOneFiftySix.SourceMonsters = map156SourceMonsters
	definitions[156] = mapOneFiftySix

	mapOneFiftySeven := definitions[157]
	mapOneFiftySeven.SourceNPCs = map157SourceNPCs
	mapOneFiftySeven.SourceMonsters = map157SourceMonsters
	definitions[157] = mapOneFiftySeven

	mapOneFiftyEight := definitions[158]
	mapOneFiftyEight.SourceNPCs = map158SourceNPCs
	mapOneFiftyEight.SourceMonsters = map158SourceMonsters
	definitions[158] = mapOneFiftyEight

	mapOneFiftyNine := definitions[159]
	mapOneFiftyNine.SourceNPCs = map159SourceNPCs
	mapOneFiftyNine.SourceMonsters = map159SourceMonsters
	definitions[159] = mapOneFiftyNine

	mapOneSixty := definitions[160]
	mapOneSixty.SourceNPCs = map160SourceNPCs
	mapOneSixty.SourceMonsters = map160SourceMonsters
	definitions[160] = mapOneSixty

	mapOneSixtyOne := definitions[161]
	mapOneSixtyOne.SourceNPCs = map161SourceNPCs
	mapOneSixtyOne.SourceMonsters = map161SourceMonsters
	definitions[161] = mapOneSixtyOne

	mapOneSixtyTwo := definitions[162]
	mapOneSixtyTwo.SourceNPCs = map162SourceNPCs
	mapOneSixtyTwo.SourceMonsters = map162SourceMonsters
	definitions[162] = mapOneSixtyTwo

	mapOneSixtyThree := definitions[163]
	mapOneSixtyThree.SourceNPCs = map163SourceNPCs
	mapOneSixtyThree.SourceMonsters = map163SourceMonsters
	definitions[163] = mapOneSixtyThree

	mapOneSixtyFour := definitions[164]
	mapOneSixtyFour.SourceNPCs = map164SourceNPCs
	mapOneSixtyFour.SourceMonsters = map164SourceMonsters
	definitions[164] = mapOneSixtyFour

	mapOneSixtyFive := definitions[165]
	mapOneSixtyFive.SourceNPCs = map165SourceNPCs
	mapOneSixtyFive.SourceMonsters = map165SourceMonsters
	definitions[165] = mapOneSixtyFive

	mapOneSixtySix := definitions[166]
	mapOneSixtySix.SourceNPCs = map166SourceNPCs
	mapOneSixtySix.SourceMonsters = map166SourceMonsters
	definitions[166] = mapOneSixtySix

	mapOneSixtySeven := definitions[167]
	mapOneSixtySeven.SourceNPCs = map167SourceNPCs
	mapOneSixtySeven.SourceMonsters = map167SourceMonsters
	definitions[167] = mapOneSixtySeven

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

	applyCapturedSourceTransports(definitions)

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

func applyCapturedSourceTransports(definitions map[int]townMapBootstrapDefinition) {
	for mapID, capturedTransports := range capturedSourceTransportsByMapID {
		mapDefinition, ok := definitions[mapID]
		if !ok {
			continue
		}
		for _, captured := range capturedTransports {
			mapDefinition.SourceNPCs = upsertCapturedSourceTransport(mapDefinition.SourceNPCs, captured)
		}
		definitions[mapID] = mapDefinition
	}
}

func upsertCapturedSourceTransport(npcs []sourceNPCEntry, captured sourceNPCEntry) []sourceNPCEntry {
	captured.RoleID = "-3"
	captured.DisplayName = ""
	captured.QuestState = 0
	captured.Kind = ""
	captured.IsGeneratedSourceTransport = true
	if captured.Dialogue == nil {
		captured.Dialogue = &sourceTransportDialogue
	}

	for index := range npcs {
		if npcs[index].Handle != captured.Handle {
			continue
		}
		if npcs[index].Dialogue != nil {
			captured.Dialogue = npcs[index].Dialogue
		}
		captured.IsGeneratedSourceTransport = npcs[index].IsGeneratedSourceTransport
		npcs[index] = captured
		return npcs
	}

	return append(npcs, captured)
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
			SourceMonsters:     nil,
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

// DefaultSpawnForMap 返回某 mapId 的默认出生点;找不到返回零值。
// 用于"远程玩家互推"时给非自己的在线玩家构造 SpawnFlash:后端不持久化玩家
// 运行时坐标,最小闭环下让远程玩家落到该地图默认出生点(原版由 AOI 按视野推
// 真实坐标,本轮不还原)。
func DefaultSpawnForMap(mapID int) SpawnPoint {
	mapDefinition, ok := townMapBootstrapDefinitions[mapID]
	if !ok {
		return SpawnPoint{}
	}
	return mapDefinition.DefaultSpawn
}

// BuildPlayerRolePush 把一个在线玩家(RoleSummary + PlayerBaseData)转成 kind="player"
// 的 RolePush,用于"同 mapId 在线玩家互推 createRole(1103)"。
// 字段拼装与 buildTownBootstrap 的 CreatePlayer(town_bootstrap.go:795-808)一致,
// 唯一差异是 Kind 从 "self" 改成 "player",Handle/RoleID 用玩家自己的 RoleID
// (原版 createRole 用 handle 区分实体,本项目 handle 与 roleId 一致)。
func BuildPlayerRolePush(role session.RoleSummary, playerBase session.PlayerBaseData, spawn SpawnPoint) RolePush {
	return RolePush{
		Handle:       role.RoleID,
		RoleID:       role.RoleID,
		DisplayName:  playerBase.DisplayName,
		Level:        playerBase.Level,
		Vocation:     playerBase.Voc,
		MapID:        itoa(playerBase.MapID),
		VisualRoleID: playerBase.VisualRoleID,
		PresetID:     playerBase.PresetID,
		SourceQuery:  playerBase.SourceQuery,
		Appearance:   playerBase.Appearance,
		Kind:         "player",
		SpawnFlash:   spawn,
		PK:           playerBase.PK,
		State:        playerBase.State,
		GuildName:    playerBase.GuildName,
		GuildPic:     playerBase.GuildPic,
	}
}

func IsShuiliandongMapID(mapID int) bool {
	mapDefinition, ok := townMapBootstrapDefinitions[mapID]
	return ok && strings.HasPrefix(mapDefinition.Name, "水帘洞_")
}

func IsHuangfengzhaiMapID(mapID int) bool {
	mapDefinition, ok := townMapBootstrapDefinitions[mapID]
	return ok && strings.HasPrefix(mapDefinition.Name, "黄风寨_")
}

func IsFeixiandongMapID(mapID int) bool {
	mapDefinition, ok := townMapBootstrapDefinitions[mapID]
	return ok && strings.HasPrefix(mapDefinition.Name, "飞仙洞_")
}

func IsShihukuMapID(mapID int) bool {
	mapDefinition, ok := townMapBootstrapDefinitions[mapID]
	return ok && strings.HasPrefix(mapDefinition.Name, "狮虎窟_")
}

func DungeonInstanceKeyForMapID(mapID int) (string, bool) {
	switch {
	case IsShuiliandongMapID(mapID):
		return session.DungeonInstanceShuiliandong, true
	case IsHuangfengzhaiMapID(mapID):
		return session.DungeonInstanceHuangfengzhai, true
	case IsFeixiandongMapID(mapID):
		return session.DungeonInstanceFeixiandong, true
	case IsShihukuMapID(mapID):
		return session.DungeonInstanceShihuku, true
	default:
		return "", false
	}
}

func buildTownBootstrap(
	role session.RoleSummary,
	playerBase session.PlayerBaseData,
	mapDefinition townMapBootstrapDefinition,
	spawn SpawnPoint,
) TownBootstrapSnapshot {
	mapID := itoa(mapDefinition.ID)
	createRoles := make([]RolePush, 0, len(mapDefinition.SourceNPCs)+len(mapDefinition.SourceMonsters))
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
	for _, monster := range mapDefinition.SourceMonsters {
		createRoles = append(createRoles, RolePush{
			Handle:          monster.Handle,
			RoleID:          "-2",
			DisplayName:     monster.DisplayName,
			Level:           monster.Level,
			Vocation:        monster.Vocation,
			MapID:           mapID,
			VisualRoleID:    0,
			SourceQuery:     monster.SourceQuery,
			Kind:            "monster",
			SpawnFlash:      monster.SpawnFlash,
			SourceNPCVisual: buildMonsterVisual(monster),
			Movement:        buildMonsterMovement(monster),
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
			Vocation:     playerBase.Voc,
			MapID:        mapID,
			VisualRoleID: playerBase.VisualRoleID,
			PresetID:     playerBase.PresetID,
			SourceQuery:  playerBase.SourceQuery,
			Appearance:   playerBase.Appearance,
			Kind:         "self",
			SpawnFlash:   spawn,
			PK:           playerBase.PK,
			State:        playerBase.State,
			GuildName:    playerBase.GuildName,
			GuildPic:     playerBase.GuildPic,
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
			dialogue, ok = map46SourceNPCDialogueReplies[key]
			if !ok {
				return nil
			}
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
	return resolveTownTransportDestination(handle)
}

func ResolveTownTransportAnswerFromMap(fromMapID int, handle string, answerHandle string) (TownTransportDestination, bool) {
	if answerHandle != "goto" && !isCapturedTownTransportConfirmAnswer(handle, answerHandle) {
		return TownTransportDestination{}, false
	}
	if fromMapID > 0 {
		if destination, ok := capturedTownTransportRouteDestinations[townTransportRouteKey{FromMapID: fromMapID, Handle: handle}]; ok {
			if !SupportsTownTransferMap(destination.MapID) {
				return TownTransportDestination{}, false
			}
			return destination, true
		}
		if destination, ok := resolveDirectionalTownTransportDestination(fromMapID, handle); ok {
			return destination, true
		}
		if destination, ok := resolveCapturedTownTransportDestination(handle); ok {
			return destination, true
		}
	}
	return resolveTownTransportDestination(handle)
}

func resolveCapturedTownTransportDestination(handle string) (TownTransportDestination, bool) {
	if destination, ok := capturedTownTransportDestinations[handle]; ok {
		if !SupportsTownTransferMap(destination.MapID) {
			return TownTransportDestination{}, false
		}
		return destination, true
	}
	return TownTransportDestination{}, false
}

func resolveTownTransportDestination(handle string) (TownTransportDestination, bool) {
	if destination, ok := resolveCapturedTownTransportDestination(handle); ok {
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

func resolveDirectionalTownTransportDestination(fromMapID int, handle string) (TownTransportDestination, bool) {
	mapIDText, ok := strings.CutPrefix(handle, "transp_")
	if !ok {
		return TownTransportDestination{}, false
	}
	mapID, err := strconv.Atoi(mapIDText)
	if err != nil || !SupportsTownTransferMap(mapID) {
		return TownTransportDestination{}, false
	}

	mapDefinition := townMapBootstrapDefinitions[mapID]
	returnHandle := "transp_" + itoa(fromMapID)
	for _, npc := range mapDefinition.SourceNPCs {
		if npc.Handle == returnHandle {
			return TownTransportDestination{
				MapID: mapID,
				Spawn: inferTransportArrivalSpawn(mapDefinition, npc.SpawnFlash),
			}, true
		}
	}

	return TownTransportDestination{}, false
}

func inferTransportArrivalSpawn(mapDefinition townMapBootstrapDefinition, transportSpawn SpawnPoint) SpawnPoint {
	const (
		horizontalOffset = 85
		verticalOffset   = 80
		topEdgeY         = 360
		bottomEdgeY      = 660
		minX             = 40
		minY             = 40
	)

	mapWidth := observedSourceMapWidth(mapDefinition)
	spawn := transportSpawn
	leftEdge := mapWidth / 3
	rightEdge := mapWidth - leftEdge
	if transportSpawn.X <= leftEdge {
		spawn.X += horizontalOffset
	} else if transportSpawn.X >= rightEdge {
		spawn.X -= horizontalOffset
	}
	if transportSpawn.Y <= topEdgeY {
		spawn.Y += verticalOffset
	} else if transportSpawn.Y >= bottomEdgeY {
		spawn.Y -= verticalOffset
	}
	if spawn.X < minX {
		spawn.X = minX
	}
	if spawn.Y < minY {
		spawn.Y = minY
	}
	return spawn
}

func observedSourceMapWidth(mapDefinition townMapBootstrapDefinition) int {
	mapWidth := sourceTransportMapWidth(mapDefinition.ID)
	for _, npc := range mapDefinition.SourceNPCs {
		if npc.SpawnFlash.X+200 > mapWidth {
			mapWidth = npc.SpawnFlash.X + 200
		}
	}
	for _, monster := range mapDefinition.SourceMonsters {
		if monster.SpawnFlash.X+200 > mapWidth {
			mapWidth = monster.SpawnFlash.X + 200
		}
	}
	return mapWidth
}

func isCapturedTownTransportConfirmAnswer(handle string, answerHandle string) bool {
	return handle == "transp_10" && answerHandle == "1"
}

func buildSourceTransportNPC(link sourceTransportLink) sourceNPCEntry {
	return sourceNPCEntry{
		Handle:                     "transp_" + itoa(link.ToMapID),
		RoleID:                     "-3",
		DisplayName:                "",
		SourceQuery:                "transp/flag2.swf",
		SpriteName:                 "flag2",
		Width:                      158,
		Height:                     258,
		SpawnFlash:                 sourceTransportSlotSpawn(link.FromMapID, link.Slot),
		QuestState:                 0,
		IsGeneratedSourceTransport: true,
		Dialogue:                   &sourceTransportDialogue,
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

func buildMonsterVisual(monster sourceMonsterEntry) *SourceNPCVisual {
	return &SourceNPCVisual{
		MovieClipIRPath: sourceMonsterMovieClipRoot + "/" + monster.SpriteName + "/" + monster.SpriteName + "-movieclip-ir",
		Width:           monster.Width,
		Height:          monster.Height,
		NameY:           monster.Height + 18,
		QuestMarkerY:    monster.Height + 62,
	}
}

func buildMonsterMovement(monster sourceMonsterEntry) *RoleMovement {
	if monster.Movement.Speed <= 0 {
		return nil
	}
	movement := monster.Movement
	return &movement
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
