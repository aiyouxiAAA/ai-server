package battle

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"ai-server/internal/session"
)

type Camp string

const (
	CampTeam  Camp = "team"
	CampEnemy Camp = "enemy"

	PhaseCommand  = "command"
	PhasePlaying  = "playing"
	PhaseFinished = "finished"

	CommandNormalAttack = "skill-normal-attack"
	CommandMiZhan       = "skill-mi-zhan"
	CommandEnemyAttack  = "enemy-normal-attack"
	CommandDefense      = "defense"
	CommandStore        = "battle-store"
	CommandEscape       = "battle-escape"
	CommandItem         = "battle-item"

	maxStoredPower = 5
	miZhanMPCost   = 5
)

type StartRequest struct {
	MapID       string  `json:"mapId"`
	MapName     string  `json:"mapName"`
	StageFocusX float64 `json:"stageFocusX"`
	ReturnRoute string  `json:"returnRoute,omitempty"`
}

type StartPush struct {
	BattleID        string  `json:"battleId"`
	QueueIndexTeam  int     `json:"queueIndexTeam"`
	QueueIndexEnemy int     `json:"queueIndexEnemy"`
	LastMapName     string  `json:"lastMapName"`
	SelfHandle      string  `json:"selfHandle"`
	IsArena         bool    `json:"isArena"`
	MapID           string  `json:"mapId"`
	EncounterID     string  `json:"encounterId"`
	EncounterLabel  string  `json:"encounterLabel"`
	StageFocusX     float64 `json:"stageFocusX"`
	ReturnRoute     string  `json:"returnRoute,omitempty"`
}

type CellInfoPush struct {
	BattleID     string `json:"battleId,omitempty"`
	Camp         Camp   `json:"camp"`
	Handle       string `json:"handle"`
	Name         string `json:"name"`
	DisplayURL   string `json:"displayUrl"`
	Level        int    `json:"level,omitempty"`
	XScale       int    `json:"xScale"`
	YScale       int    `json:"yScale"`
	MaxHP        int    `json:"maxHp"`
	HP           int    `json:"hp"`
	MaxMP        int    `json:"maxMp,omitempty"`
	MP           int    `json:"mp,omitempty"`
	Speed        int    `json:"speed"`
	Attack       int    `json:"attack"`
	Defense      int    `json:"defense"`
	Hit          int    `json:"hit,omitempty"`
	Dog          int    `json:"dog,omitempty"`
	Fat          int    `json:"fat,omitempty"`
	CommandLabel string `json:"commandLabel"`
}

type StartCommandPush struct {
	BattleID    string      `json:"battleId"`
	ActorHandle string      `json:"actorHandle"`
	Round       int         `json:"round"`
	Sequence    int         `json:"sequence"`
	Power       interface{} `json:"power"`
}

type ActionRequest struct {
	BattleID     string `json:"battleId"`
	ActorHandle  string `json:"actorHandle"`
	CommandID    string `json:"commandId"`
	TargetHandle string `json:"targetHandle"`
	Round        int    `json:"round"`
	Sequence     int    `json:"sequence"`
}

type ItemActionRequest struct {
	BattleID     string `json:"battleId"`
	ActorHandle  string `json:"actorHandle"`
	Type         string `json:"type"`
	Index        int    `json:"index"`
	TargetHandle string `json:"targetHandle"`
	Round        int    `json:"round"`
	Sequence     int    `json:"sequence"`
}

type PlayOverRequest struct {
	BattleID string `json:"battleId"`
}

type ItemAction struct {
	SourceType  string
	SourceIndex int
	Name        string
	ItemType    string
	Display     string
	Description string
	HealHP      int
	HealMP      int
}

type ActionPush struct {
	BattleID              string         `json:"battleId,omitempty"`
	ActorHandle           string         `json:"actorHandle"`
	TargetHandle          string         `json:"targetHandle"`
	CommandID             string         `json:"commandId,omitempty"`
	ActionName            string         `json:"actionName"`
	SourceActionLabel     string         `json:"sourceActionLabel,omitempty"`
	TargetActionState     string         `json:"targetActionState,omitempty"`
	TargetActionStateCode string         `json:"targetActionStateCode,omitempty"`
	Damage                int            `json:"damage"`
	TargetHP              int            `json:"targetHp"`
	TargetMP              int            `json:"targetMp,omitempty"`
	TargetDead            bool           `json:"targetDead"`
	RefreshInfos          []CellInfoPush `json:"refreshInfos,omitempty"`
	Round                 int            `json:"round,omitempty"`
	Sequence              int            `json:"sequence,omitempty"`
}

type OverPush struct {
	BattleID string        `json:"battleId,omitempty"`
	Winner   Camp          `json:"winner"`
	Rounds   int           `json:"rounds"`
	Result   ResultPayload `json:"result"`
}

type ResultPayload struct {
	Winner        Camp     `json:"winner"`
	Rounds        int      `json:"rounds"`
	ExpDelta      int      `json:"expDelta"`
	CurrencyDelta int      `json:"currencyDelta"`
	Items         []string `json:"items"`
	Escaped       bool     `json:"escaped,omitempty"`
}

type Runtime struct {
	BattleID         string
	RoleID           string
	MapID            string
	Phase            string
	Round            int
	Cells            []CellInfoPush
	ActiveHandle     string
	ConsumedSequence map[int]bool
	DefendingHandles map[string]bool
	StoredPower      map[string]int
	PendingStart     *StartCommandPush
	PendingOver      *OverPush
	nextSequence     int
}

type StartBundle struct {
	Start        StartPush
	Cells        []CellInfoPush
	StartCommand StartCommandPush
}

type ActionResult struct {
	Actions      []ActionPush
	StartCommand *StartCommandPush
	Over         *OverPush
	ErrorCode    string
}

func NewWildBattle(role session.RoleSummary, playerBase session.PlayerBaseData, request StartRequest) (*Runtime, StartBundle, bool) {
	mapID := strings.TrimSpace(request.MapID)
	if mapID != "4" && mapID != "5" {
		return nil, StartBundle{}, false
	}
	mapName := strings.TrimSpace(request.MapName)
	if mapName == "" {
		mapName = "野外"
	}
	battleID := fmt.Sprintf("server-%s-map%s-r1", role.RoleID, mapID)
	playerLevel := maxInt(1, playerBase.Level)
	roleState := playerBase.RoleState
	rolePhysique := playerBase.RolePhysique
	if roleState != nil && roleState.Lv > 0 {
		playerLevel = roleState.Lv
	}
	playerMaxHP := maxInt(1, playerBase.MaxHP)
	playerMaxMP := maxInt(1, playerBase.MaxMP)
	if rolePhysique != nil {
		playerMaxHP = maxInt(1, rolePhysique.MaxHP)
		playerMaxMP = maxInt(1, rolePhysique.MaxMP)
	}
	playerHP := clampInt(defaultNonZero(playerBase.HP, playerMaxHP), 1, playerMaxHP)
	playerMP := clampInt(defaultNonZero(playerBase.MP, playerMaxMP), 0, playerMaxMP)
	if roleState != nil {
		playerHP = clampInt(defaultNonZero(roleState.HP, playerHP), 1, playerMaxHP)
		playerMP = clampInt(defaultNonZero(roleState.MP, playerMP), 0, playerMaxMP)
	}
	playerSpeed := 130
	playerAttack := 10
	playerDefense := 10
	playerHit := 100
	playerDog := 50
	playerFat := 5
	if roleState != nil && roleState.Speed > 0 {
		playerSpeed = roleState.Speed
	}
	if rolePhysique != nil {
		playerAttack = maxInt(1, rolePhysique.PhyAtk)
		playerDefense = maxInt(0, rolePhysique.PhyDef)
		playerHit = maxInt(0, rolePhysique.Hit)
		playerDog = maxInt(0, rolePhysique.Dog)
		playerFat = maxInt(0, rolePhysique.Fat)
	}
	enemy := sourceEnemyForMap(mapID)
	cells := []CellInfoPush{
		{
			BattleID:     battleID,
			Camp:         CampTeam,
			Handle:       role.RoleID,
			Name:         defaultString(playerBase.DisplayName, role.DisplayName),
			DisplayURL:   defaultString(playerBase.SourceQuery, defaultString(role.SourceQuery, "human/human.swf?w1=1&")),
			Level:        playerLevel,
			XScale:       100,
			YScale:       100,
			MaxHP:        playerMaxHP,
			HP:           playerHP,
			MaxMP:        playerMaxMP,
			MP:           playerMP,
			Speed:        playerSpeed,
			Attack:       playerAttack,
			Defense:      playerDefense,
			Hit:          playerHit,
			Dog:          playerDog,
			Fat:          playerFat,
			CommandLabel: "普通攻击",
		},
		enemy.withBattleID(battleID),
	}
	runtime := &Runtime{
		BattleID:         battleID,
		RoleID:           role.RoleID,
		MapID:            mapID,
		Phase:            PhaseCommand,
		Round:            1,
		Cells:            cells,
		ActiveHandle:     role.RoleID,
		ConsumedSequence: map[int]bool{},
		DefendingHandles: map[string]bool{},
		StoredPower:      map[string]int{},
		nextSequence:     1,
	}
	start := StartPush{
		BattleID:        battleID,
		QueueIndexTeam:  queueIndexForMap(mapID),
		QueueIndexEnemy: enemyQueueIndexForMap(mapID),
		LastMapName:     mapName,
		SelfHandle:      role.RoleID,
		IsArena:         false,
		MapID:           mapID,
		EncounterID:     "classic-wild-" + mapID,
		EncounterLabel:  mapName + " 暗雷",
		StageFocusX:     request.StageFocusX,
		ReturnRoute:     defaultString(request.ReturnRoute, "town-placeholder"),
	}
	return runtime, StartBundle{
		Start: start,
		Cells: cells,
		StartCommand: StartCommandPush{
			BattleID:    battleID,
			ActorHandle: role.RoleID,
			Round:       runtime.Round,
			Sequence:    runtime.nextSequence,
			Power:       runtime.powerFor(role.RoleID),
		},
	}, true
}

func (runtime *Runtime) ProcessAction(request ActionRequest) ActionResult {
	actor, validation := runtime.validateActorTurn(actionTurnRequest{
		BattleID:    request.BattleID,
		ActorHandle: request.ActorHandle,
		Round:       request.Round,
		Sequence:    request.Sequence,
	})
	if validation.ErrorCode != "" {
		return validation
	}

	commandID := strings.TrimSpace(request.CommandID)

	switch commandID {
	case CommandEscape:
		if !runtime.isSelfTarget(actor, request.TargetHandle) {
			return ActionResult{ErrorCode: "invalid_target"}
		}
		runtime.ConsumedSequence[request.Sequence] = true
		runtime.setStoredPower(actor.Handle, 0)
		runtime.Phase = PhaseFinished
		return ActionResult{
			Over: runtime.buildOver(CampEnemy, true),
		}
	case CommandDefense:
		if !runtime.isSelfTarget(actor, request.TargetHandle) {
			return ActionResult{ErrorCode: "invalid_target"}
		}
		runtime.ConsumedSequence[request.Sequence] = true
		runtime.setStoredPower(actor.Handle, 0)
		if runtime.DefendingHandles == nil {
			runtime.DefendingHandles = map[string]bool{}
		}
		runtime.DefendingHandles[actor.Handle] = true
		return runtime.resolveEnemyTurnAndNextCommand(actor, []ActionPush{
			runtime.resolveSelfAction(actor, commandID, "防御", "def"),
		})
	case CommandStore:
		if !runtime.isSelfTarget(actor, request.TargetHandle) {
			return ActionResult{ErrorCode: "invalid_target"}
		}
		runtime.ConsumedSequence[request.Sequence] = true
		runtime.setStoredPower(actor.Handle, runtime.powerFor(actor.Handle)+1)
		return runtime.resolveEnemyTurnAndNextCommand(actor, []ActionPush{
			runtime.resolveSelfAction(actor, commandID, "蓄气", "store"),
		})
	}

	target := runtime.cellByHandle(request.TargetHandle)
	if target == nil || target.Camp != CampEnemy || target.HP <= 0 {
		return ActionResult{ErrorCode: "invalid_target"}
	}

	runtime.ConsumedSequence[request.Sequence] = true
	runtime.setStoredPower(actor.Handle, 0)
	actions := []ActionPush{runtime.resolveAttack(actor, target, commandID)}
	return runtime.resolveEnemyTurnAndNextCommand(actor, actions)
}

func (runtime *Runtime) ProcessItemAction(request ItemActionRequest, item ItemAction) ActionResult {
	actor, validation := runtime.validateActorTurn(actionTurnRequest{
		BattleID:    request.BattleID,
		ActorHandle: request.ActorHandle,
		Round:       request.Round,
		Sequence:    request.Sequence,
	})
	if validation.ErrorCode != "" {
		return validation
	}

	item.SourceType = strings.TrimSpace(item.SourceType)
	item.ItemType = strings.TrimSpace(item.ItemType)
	if item.SourceType != strings.TrimSpace(request.Type) || item.SourceIndex != request.Index {
		return ActionResult{ErrorCode: "item_mismatch"}
	}

	target, targetError := runtime.resolveItemTarget(actor, request.TargetHandle, item)
	if targetError != "" {
		return ActionResult{ErrorCode: targetError}
	}
	if item.HealHP <= 0 && item.HealMP <= 0 {
		return ActionResult{ErrorCode: "unsupported_item_effect"}
	}

	runtime.ConsumedSequence[request.Sequence] = true
	runtime.setStoredPower(actor.Handle, 0)
	if item.HealHP > 0 {
		target.HP = clampInt(target.HP+item.HealHP, 0, target.MaxHP)
	}
	if item.HealMP > 0 {
		target.MP = clampInt(target.MP+item.HealMP, 0, target.MaxMP)
	}
	return runtime.resolveEnemyTurnAndNextCommand(actor, []ActionPush{
		runtime.resolveItemAction(actor, target, item),
	})
}

func (runtime *Runtime) ProcessPlayOver(request PlayOverRequest) ActionResult {
	if runtime == nil {
		return ActionResult{ErrorCode: "battle_missing"}
	}
	if request.BattleID != runtime.BattleID {
		return ActionResult{ErrorCode: "battle_mismatch"}
	}
	if runtime.PendingOver != nil {
		over := runtime.PendingOver
		runtime.PendingOver = nil
		runtime.PendingStart = nil
		return ActionResult{Over: over}
	}
	if runtime.PendingStart != nil {
		start := *runtime.PendingStart
		runtime.PendingStart = nil
		runtime.Phase = PhaseCommand
		return ActionResult{StartCommand: &start}
	}
	return ActionResult{ErrorCode: "battle_play_over_empty"}
}

type actionTurnRequest struct {
	BattleID    string
	ActorHandle string
	Round       int
	Sequence    int
}

func (runtime *Runtime) validateActorTurn(request actionTurnRequest) (*CellInfoPush, ActionResult) {
	if runtime == nil || runtime.Phase != PhaseCommand {
		return nil, ActionResult{ErrorCode: "battle_not_command"}
	}
	if request.BattleID != runtime.BattleID {
		return nil, ActionResult{ErrorCode: "battle_mismatch"}
	}
	if request.ActorHandle != runtime.ActiveHandle {
		return nil, ActionResult{ErrorCode: "actor_not_active"}
	}
	if request.Round != runtime.Round || request.Sequence != runtime.nextSequence {
		return nil, ActionResult{ErrorCode: "sequence_mismatch"}
	}
	if runtime.ConsumedSequence[request.Sequence] {
		return nil, ActionResult{ErrorCode: "sequence_consumed"}
	}
	actor := runtime.cellByHandle(request.ActorHandle)
	if actor == nil || actor.Camp != CampTeam || actor.HP <= 0 {
		return nil, ActionResult{ErrorCode: "invalid_actor"}
	}
	return actor, ActionResult{}
}

func (runtime *Runtime) resolveEnemyTurnAndNextCommand(actor *CellInfoPush, actions []ActionPush) ActionResult {
	if winner := runtime.resolveWinner(); winner != "" {
		runtime.Phase = PhaseFinished
		runtime.PendingStart = nil
		runtime.PendingOver = runtime.buildOver(winner)
		return ActionResult{
			Actions: actions,
		}
	}

	enemy := runtime.firstLiving(CampEnemy)
	team := runtime.firstLiving(CampTeam)
	if enemy != nil && team != nil {
		actions = append(actions, runtime.resolveAttack(enemy, team, CommandEnemyAttack))
	}
	if winner := runtime.resolveWinner(); winner != "" {
		runtime.Phase = PhaseFinished
		runtime.PendingStart = nil
		runtime.PendingOver = runtime.buildOver(winner)
		return ActionResult{
			Actions: actions,
		}
	}

	runtime.Round += 1
	runtime.nextSequence += 1
	runtime.ActiveHandle = actor.Handle
	runtime.Phase = PhasePlaying
	start := StartCommandPush{
		BattleID:    runtime.BattleID,
		ActorHandle: actor.Handle,
		Round:       runtime.Round,
		Sequence:    runtime.nextSequence,
		Power:       runtime.powerFor(actor.Handle),
	}
	runtime.PendingStart = &start
	runtime.PendingOver = nil
	return ActionResult{
		Actions: actions,
	}
}

func (runtime *Runtime) resolveAttack(actor *CellInfoPush, target *CellInfoPush, commandID string) ActionPush {
	multiplier := 1.0
	actionName := actor.CommandLabel
	sourceActionLabel := "nomalAtk"
	switch commandID {
	case CommandMiZhan:
		multiplier = 1.45
		actionName = "密斩"
		sourceActionLabel = "manycut"
	case CommandNormalAttack, CommandEnemyAttack:
		actionName = "普通攻击"
	}
	defense := target.Defense
	if runtime.DefendingHandles[target.Handle] {
		defense *= 2
	}
	damage := maxInt(1, int(math.Round(float64(actor.Attack)*multiplier))-defense)
	targetActionState := "normal"
	targetActionStateCode := "0"
	if runtime.resolveCriticalHit(actor, target, commandID) {
		damage *= 2
		targetActionState = "fat"
		targetActionStateCode = "2"
	}
	target.HP = maxInt(0, target.HP-damage)
	delete(runtime.DefendingHandles, target.Handle)
	if commandID == CommandMiZhan {
		actor.MP = maxInt(0, actor.MP-miZhanMPCost)
	}
	refreshInfos := []CellInfoPush{*target}
	if commandID == CommandMiZhan {
		refreshInfos = []CellInfoPush{*actor, *target}
	}
	return ActionPush{
		BattleID:              runtime.BattleID,
		ActorHandle:           actor.Handle,
		TargetHandle:          target.Handle,
		CommandID:             commandID,
		ActionName:            actionName,
		SourceActionLabel:     sourceActionLabel,
		TargetActionState:     targetActionState,
		TargetActionStateCode: targetActionStateCode,
		Damage:                damage,
		TargetHP:              target.HP,
		TargetDead:            target.HP <= 0,
		RefreshInfos:          refreshInfos,
		Round:                 runtime.Round,
		Sequence:              runtime.nextSequence,
	}
}

func (runtime *Runtime) resolveCriticalHit(actor *CellInfoPush, target *CellInfoPush, commandID string) bool {
	if actor == nil || target == nil || actor.Fat <= 0 {
		return false
	}
	if commandID != CommandNormalAttack && commandID != CommandEnemyAttack {
		return false
	}
	if actor.Fat >= 100 {
		return true
	}
	score := 0
	seed := fmt.Sprintf("%s:%d:%d:%s:%s:%s", runtime.BattleID, runtime.Round, runtime.nextSequence, actor.Handle, target.Handle, commandID)
	for _, char := range seed {
		score = (score*31 + int(char)) % 100
	}
	return score < actor.Fat
}

func (runtime *Runtime) resolveSelfAction(
	actor *CellInfoPush,
	commandID string,
	actionName string,
	sourceActionLabel string,
) ActionPush {
	return ActionPush{
		BattleID:          runtime.BattleID,
		ActorHandle:       actor.Handle,
		TargetHandle:      actor.Handle,
		CommandID:         commandID,
		ActionName:        actionName,
		SourceActionLabel: sourceActionLabel,
		Damage:            0,
		TargetHP:          actor.HP,
		TargetMP:          actor.MP,
		TargetDead:        actor.HP <= 0,
		RefreshInfos:      []CellInfoPush{*actor},
		Round:             runtime.Round,
		Sequence:          runtime.nextSequence,
	}
}

func (runtime *Runtime) resolveItemTarget(
	actor *CellInfoPush,
	targetHandle string,
	item ItemAction,
) (*CellInfoPush, string) {
	normalizedType := strings.TrimSpace(item.ItemType)
	switch normalizedType {
	case "own", "all":
		if !runtime.isSelfTarget(actor, targetHandle) {
			return nil, "invalid_target"
		}
		return actor, ""
	case "oneO":
		target := runtime.cellByHandle(targetHandle)
		if target == nil || target.Camp != CampTeam {
			return nil, "invalid_target"
		}
		if target.HP <= 0 && item.HealHP <= 0 {
			return nil, "invalid_target"
		}
		return target, ""
	case "oneE":
		target := runtime.cellByHandle(targetHandle)
		if target == nil || target.Camp != CampEnemy || target.HP <= 0 {
			return nil, "invalid_target"
		}
		return target, ""
	default:
		return nil, "unsupported_item_type"
	}
}

func (runtime *Runtime) resolveItemAction(actor *CellInfoPush, target *CellInfoPush, item ItemAction) ActionPush {
	actionName := strings.TrimSpace(item.Name)
	if actionName == "" {
		actionName = "useItem"
	}
	return ActionPush{
		BattleID:          runtime.BattleID,
		ActorHandle:       actor.Handle,
		TargetHandle:      target.Handle,
		CommandID:         CommandItem,
		ActionName:        actionName,
		SourceActionLabel: "useItem",
		Damage:            0,
		TargetHP:          target.HP,
		TargetMP:          target.MP,
		TargetDead:        target.HP <= 0,
		RefreshInfos:      []CellInfoPush{*target},
		Round:             runtime.Round,
		Sequence:          runtime.nextSequence,
	}
}

func (runtime *Runtime) buildOver(winner Camp, escaped ...bool) *OverPush {
	isEscaped := len(escaped) > 0 && escaped[0]
	expDelta, items := sourceBattleRewardsForMap(runtime.MapID, winner, isEscaped)
	result := ResultPayload{
		Winner:        winner,
		Rounds:        runtime.Round,
		ExpDelta:      expDelta,
		CurrencyDelta: 0,
		Items:         items,
		Escaped:       isEscaped,
	}
	return &OverPush{
		BattleID: runtime.BattleID,
		Winner:   winner,
		Rounds:   runtime.Round,
		Result:   result,
	}
}

func sourceBattleRewardsForMap(mapID string, winner Camp, escaped bool) (int, []string) {
	if winner != CampTeam || escaped {
		return 0, []string{}
	}

	switch strings.TrimSpace(mapID) {
	case "5":
		// Captured source tail packet 20260517_043413_010_conn_0005:
		// "战斗奖励: 肉 x1" and "获得经验:0".
		return 0, []string{"肉"}
	default:
		return 0, []string{}
	}
}

func (runtime *Runtime) cellByHandle(handle string) *CellInfoPush {
	for index := range runtime.Cells {
		if runtime.Cells[index].Handle == handle {
			return &runtime.Cells[index]
		}
	}
	return nil
}

func (runtime *Runtime) firstLiving(camp Camp) *CellInfoPush {
	for index := range runtime.Cells {
		if runtime.Cells[index].Camp == camp && runtime.Cells[index].HP > 0 {
			return &runtime.Cells[index]
		}
	}
	return nil
}

func (runtime *Runtime) isSelfTarget(actor *CellInfoPush, targetHandle string) bool {
	return actor != nil && strings.TrimSpace(targetHandle) == actor.Handle
}

func (runtime *Runtime) powerFor(handle string) int {
	if runtime == nil {
		return 0
	}
	return clampInt(runtime.StoredPower[handle], 0, maxStoredPower)
}

func (runtime *Runtime) setStoredPower(handle string, value int) {
	if runtime.StoredPower == nil {
		runtime.StoredPower = map[string]int{}
	}
	runtime.StoredPower[handle] = clampInt(value, 0, maxStoredPower)
}

func (runtime *Runtime) resolveWinner() Camp {
	hasTeam := runtime.firstLiving(CampTeam) != nil
	hasEnemy := runtime.firstLiving(CampEnemy) != nil
	if !hasTeam {
		return CampEnemy
	}
	if !hasEnemy {
		return CampTeam
	}
	return ""
}

func sourceEnemyForMap(mapID string) CellInfoPush {
	switch mapID {
	case "5":
		return CellInfoPush{
			Camp:         CampEnemy,
			Handle:       "7069963681398983",
			Name:         "山林狼兽",
			DisplayURL:   "monstermap/graywolf.swf",
			Level:        1,
			XScale:       100,
			YScale:       100,
			MaxHP:        72,
			HP:           72,
			MaxMP:        9,
			MP:           9,
			Speed:        15,
			Attack:       9,
			Defense:      9,
			CommandLabel: "普通攻击",
		}
	default:
		return CellInfoPush{
			Camp:         CampEnemy,
			Handle:       "7089932810872715",
			Name:         "山林蜘蛛",
			DisplayURL:   "monstermap/forestspider.swf",
			Level:        1,
			XScale:       100,
			YScale:       100,
			MaxHP:        72,
			HP:           72,
			MaxMP:        9,
			MP:           9,
			Speed:        15,
			Attack:       9,
			Defense:      9,
			CommandLabel: "普通攻击",
		}
	}
}

func (cell CellInfoPush) withBattleID(battleID string) CellInfoPush {
	cell.BattleID = battleID
	return cell
}

func queueIndexForMap(mapID string) int {
	if mapID == "5" {
		return 1
	}
	return 0
}

func enemyQueueIndexForMap(mapID string) int {
	if mapID == "5" {
		return 4
	}
	return 0
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func defaultNonZero(value int, fallback int) int {
	if value == 0 {
		return fallback
	}
	return value
}

func clampInt(value int, minimum int, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func ParseMapID(mapID string) (int, bool) {
	value, err := strconv.Atoi(strings.TrimSpace(mapID))
	return value, err == nil
}
