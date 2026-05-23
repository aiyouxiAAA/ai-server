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
	Dialogue    *sourceNPCDialogueEntry
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
	9:  {},
	13: {},
	14: {},
	15: {},
	16: {},
	17: {},
	18: {},
	19: {},
	20: {},
	21: {},
	22: {},
	23: {},
	24: {},
	25: {},
	26: {},
	27: {},
	28: {},
	29: {},
	30: {},
	32: {},
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

	for _, link := range yunyinSourceTransportLinks {
		mapDefinition, ok := definitions[link.FromMapID]
		if !ok {
			continue
		}
		if _, ok := definitions[link.ToMapID]; !ok {
			continue
		}
		mapDefinition.SourceNPCs = append(mapDefinition.SourceNPCs, buildSourceTransportNPC(link))
		definitions[link.FromMapID] = mapDefinition
	}

	return definitions
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
	case "云隐村口_1", "云隐村口_2":
		return SpawnPoint{X: 1000, Y: 450}
	case "云隐山道_3", "云隐山道_4":
		return SpawnPoint{X: 1000, Y: 300}
	case "云隐山道_5":
		return SpawnPoint{X: 500, Y: 400}
	case "树洞":
		return SpawnPoint{X: 500, Y: 550}
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
		createRoles = append(createRoles, RolePush{
			Handle:          npc.Handle,
			RoleID:          roleID,
			DisplayName:     npc.DisplayName,
			Level:           1,
			MapID:           mapID,
			VisualRoleID:    0,
			SourceQuery:     npc.SourceQuery,
			Kind:            "npc",
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

func BuildAnswerReply(handle string, msgHandle string, answerHandle string) *AnswerSpeakPush {
	dialogue, ok := map1SourceNPCDialogueReplies[sourceNPCDialogueReplyKey{
		Handle:       handle,
		MsgHandle:    msgHandle,
		AnswerHandle: answerHandle,
	}]
	if !ok {
		return nil
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
	if answerHandle != "goto" {
		return TownTransportDestination{}, false
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
