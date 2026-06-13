package battle

import (
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"ai-server/internal/session"
)

type Camp string

const (
	CampTeam  Camp = "team"
	CampEnemy Camp = "enemy"

	PhaseCommand  = "command"
	PhasePlaying  = "playing"
	PhaseFinished = "finished"

	CommandNormalAttack   = "skill-normal-attack"
	CommandMiZhan         = "skill-mi-zhan"
	CommandDuoDuanZhan    = "skill-duo-duan-zhan"
	CommandShiXueZhan     = "skill-shi-xue-zhan"
	CommandKuangBao       = "skill-kuang-bao"
	CommandHongYueZhan    = "skill-hong-yue-zhan"
	CommandXueQie         = "skill-xue-qie"
	CommandLeiHunZhan     = "skill-lei-hun-zhan"
	CommandEnemyAttack    = "enemy-normal-attack"
	CommandEnemySlideCut  = "enemy-slide-cut"
	CommandEnemyShadeCut  = "enemy-shade-cut"
	CommandEnemyHelixAtk  = "enemy-helix-atk"
	CommandEnemyPalsyAtk  = "enemy-palsy-atk"
	CommandEnemyRampage   = "enemy-rampage-power"
	CommandEnemyFirePower = "enemy-fire-power"
	CommandEnemyDeadLight = "enemy-dead-light"
	CommandEnemyDoubleHit = "enemy-double-hit"
	CommandDefense        = "defense"
	CommandStore          = "battle-store"
	CommandEscape         = "battle-escape"
	CommandItem           = "battle-item"

	maxStoredPower                 = 5
	leiHunZhanRequiredPower        = 3
	enemySlideCutMPCost            = 10
	enemySlideCutChance            = 20
	enemyShadeCutMPCost            = 40
	enemyShadeCutChance            = 30
	enemyHelixAtkMPCost            = 10
	enemyHelixAtkChance            = 23
	enemyHelixAtkDamageMultiplier  = 1.32
	enemyPalsyAtkChance            = 40
	enemyPalsyAtkStatusChance      = 100
	enemyStunCounterChance         = 5
	enemyRampageMPCost             = 10
	enemyRampageMaxRounds          = 50
	enemyFirePowerChance           = 60
	enemyFirePowerDamageMultiplier = 0.3
	enemyDeadLightChance           = 35
	enemyDeadLightMPDamage         = 40
	enemyDoubleHitChance           = 45
	defaultBattleHit               = 100
	defaultBattleDog               = 50
	defaultBattleFat               = 5
)

var sourceEncounterRoll = func(maxExclusive int) int {
	if maxExclusive <= 0 {
		return 0
	}
	return rand.Intn(maxExclusive)
}

var battleIDSerial uint64

type StartRequest struct {
	MapID               string  `json:"mapId"`
	MapName             string  `json:"mapName"`
	StageFocusX         float64 `json:"stageFocusX"`
	ReturnRoute         string  `json:"returnRoute,omitempty"`
	SourceMonsterHandle string  `json:"sourceMonsterHandle,omitempty"`
}

type StartPush struct {
	BattleID        string              `json:"battleId"`
	QueueIndexTeam  int                 `json:"queueIndexTeam"`
	QueueIndexEnemy int                 `json:"queueIndexEnemy"`
	LastMapName     string              `json:"lastMapName"`
	SelfHandle      string              `json:"selfHandle"`
	IsArena         bool                `json:"isArena"`
	MapID           string              `json:"mapId"`
	EncounterID     string              `json:"encounterId"`
	EncounterLabel  string              `json:"encounterLabel"`
	StageFocusX     float64             `json:"stageFocusX"`
	ReturnRoute     string              `json:"returnRoute,omitempty"`
	Commands        []CommandDefinition `json:"commands,omitempty"`
}

type CellInfoPush struct {
	BattleID          string `json:"battleId,omitempty"`
	Camp              Camp   `json:"camp"`
	Handle            string `json:"handle"`
	Name              string `json:"name"`
	DisplayURL        string `json:"displayUrl"`
	Level             int    `json:"level,omitempty"`
	XScale            int    `json:"xScale"`
	YScale            int    `json:"yScale"`
	MaxHP             int    `json:"maxHp"`
	HP                int    `json:"hp"`
	MaxMP             int    `json:"maxMp,omitempty"`
	MP                int    `json:"mp,omitempty"`
	Speed             int    `json:"speed"`
	Attack            int    `json:"attack"`
	Defense           int    `json:"defense"`
	MgcDefense        int    `json:"mgcDef,omitempty"`
	Hit               int    `json:"hit,omitempty"`
	Dog               int    `json:"dog,omitempty"`
	Fat               int    `json:"fat,omitempty"`
	CommandLabel      string `json:"commandLabel"`
	DamageDefenseType string `json:"damageDefenseType,omitempty"`
}

type StartCommandPush struct {
	BattleID    string      `json:"battleId"`
	ActorHandle string      `json:"actorHandle"`
	Round       int         `json:"round"`
	Sequence    int         `json:"sequence"`
	Power       interface{} `json:"power"`
}

type CommandDefinition struct {
	ID                string  `json:"id"`
	Kind              string  `json:"kind"`
	Label             string  `json:"label"`
	SourceType        string  `json:"sourceType,omitempty"`
	ActionName        string  `json:"actionName,omitempty"`
	SourceActionLabel string  `json:"sourceActionLabel,omitempty"`
	Target            string  `json:"target"`
	DamageMultiplier  float64 `json:"damageMultiplier,omitempty"`
	MPCost            int     `json:"mpCost,omitempty"`
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
	BattleID              string                  `json:"battleId,omitempty"`
	ActorHandle           string                  `json:"actorHandle"`
	TargetHandle          string                  `json:"targetHandle"`
	CommandID             string                  `json:"commandId,omitempty"`
	ActionName            string                  `json:"actionName"`
	SourceMode            string                  `json:"sourceMode,omitempty"`
	SourceActionLabel     string                  `json:"sourceActionLabel,omitempty"`
	TargetInDef           bool                    `json:"targetInDef,omitempty"`
	TargetActionState     string                  `json:"targetActionState,omitempty"`
	TargetActionStateCode string                  `json:"targetActionStateCode,omitempty"`
	TargetActionResults   []ActionResultCodeEntry `json:"targetActionResults,omitempty"`
	Damage                int                     `json:"damage"`
	TargetHP              int                     `json:"targetHp"`
	TargetMP              int                     `json:"targetMp,omitempty"`
	TargetDead            bool                    `json:"targetDead"`
	RefreshInfos          []CellInfoPush          `json:"refreshInfos,omitempty"`
	Round                 int                     `json:"round,omitempty"`
	Sequence              int                     `json:"sequence,omitempty"`
}

type ActionResultCodeEntry struct {
	Handle    string `json:"handle"`
	StateCode string `json:"stateCode"`
}

type BuffInfoPush struct {
	BattleID      string `json:"battleId,omitempty"`
	ReleaseHandle string `json:"releaseHandle,omitempty"`
	TargetHandle  string `json:"targetHandle"`
	Name          string `json:"name"`
	Display       string `json:"display"`
	Description   string `json:"description"`
	Round         int    `json:"round"`
	ActionHandle  string `json:"actionHandle,omitempty"`
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
	BattleID            string
	RoleID              string
	MapID               string
	SourceMonsterHandle string
	RoleSkills          []session.RoleSkill
	Phase               string
	Round               int
	Cells               []CellInfoPush
	ActiveHandle        string
	ConsumedSequence    map[int]bool
	DefendingHandles    map[string]bool
	StatusEffects       map[string]BattleStatusEffects
	StoredPower         map[string]int
	PendingBuffInfos    []BuffInfoPush
	PendingStart        *StartCommandPush
	PendingOver         *OverPush
	nextSequence        int
}

type BattleStatusEffect struct {
	Name           string
	Display        string
	Description    string
	Rounds         int
	SourceHandle   string
	SourceSkill    string
	AppliedAction  string
	SourceAttack   int
	TickMinPercent int
	TickMaxPercent int
	SkipTurn       bool
}

type BattleStatusEffects struct {
	KuangBaoRounds int
	Effects        map[string]BattleStatusEffect
}

type StartBundle struct {
	Start        StartPush
	Cells        []CellInfoPush
	StartCommand StartCommandPush
}

type ActionResult struct {
	Actions      []ActionPush
	BuffInfos    []BuffInfoPush
	StartCommand *StartCommandPush
	Over         *OverPush
	ErrorCode    string
}

type commandProfile struct {
	ActionName        string
	SourceType        string
	SourceActionLabel string
	DamageMultiplier  float64
	MPCost            int
	CanDodge          bool
	CanFat            bool
	LifeStealChance   int
	LifeStealRatio    float64
	DefenseType       string
	TargetMPDamage    int
	StatusName        string
	StatusDisplay     string
	StatusDescription string
	StatusRounds      int
	StatusChance      int
	StatusTickMin     int
	StatusTickMax     int
	SkipTurn          bool
}

func NewWildBattle(role session.RoleSummary, playerBase session.PlayerBaseData, request StartRequest) (*Runtime, StartBundle, bool) {
	mapID := strings.TrimSpace(request.MapID)
	enemyConfigs := sourceEnemyConfigsForEncounter(mapID, request.StageFocusX)
	encounterKind := "暗雷"
	sourceMonsterHandle := strings.TrimSpace(request.SourceMonsterHandle)
	if sourceMonsterHandle != "" {
		visibleMonsters, ok := sourceVisibleMonsterConfigsForHandle(mapID, sourceMonsterHandle)
		if !ok {
			return nil, StartBundle{}, false
		}
		enemyConfigs = visibleMonsters
		encounterKind = "明怪"
	}
	if len(enemyConfigs) <= 0 {
		return nil, StartBundle{}, false
	}
	firstEnemyConfig := enemyConfigs[0]
	mapName := strings.TrimSpace(request.MapName)
	if mapName == "" {
		mapName = "野外"
	}
	if sourceEncounterHasBoss(enemyConfigs) {
		encounterKind = "首领"
	}
	battleID := fmt.Sprintf("server-%s-map%s-%d-%d", role.RoleID, mapID, time.Now().UnixNano(), atomic.AddUint64(&battleIDSerial, 1))
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
	playerMgcDefense := 10
	playerHit := defaultBattleHit
	playerDog := defaultBattleDog
	playerFat := defaultBattleFat
	if roleState != nil && roleState.Speed > 0 {
		playerSpeed = roleState.Speed
	}
	if rolePhysique != nil {
		playerAttack = maxInt(1, rolePhysique.PhyAtk)
		playerDefense = maxInt(0, rolePhysique.PhyDef)
		playerMgcDefense = maxInt(0, rolePhysique.MgcDef)
		playerHit = maxInt(0, rolePhysique.Hit)
		playerDog = maxInt(0, rolePhysique.Dog)
		playerFat = maxInt(0, rolePhysique.Fat)
	}
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
			MgcDefense:   playerMgcDefense,
			Hit:          playerHit,
			Dog:          playerDog,
			Fat:          playerFat,
			CommandLabel: "普通攻击",
		},
	}
	for index, enemyConfig := range enemyConfigs {
		if sourceMonsterHandle != "" {
			cells = append(cells, enemyConfig.Cell.withBattleID(battleID))
			continue
		}
		cells = append(cells, enemyConfig.Cell.withBattleIDAndSlot(battleID, index))
	}
	runtime := &Runtime{
		BattleID:            battleID,
		RoleID:              role.RoleID,
		MapID:               mapID,
		SourceMonsterHandle: sourceMonsterHandle,
		RoleSkills:          cloneBattleRoleSkills(role.Skills),
		Phase:               PhaseCommand,
		Round:               1,
		Cells:               cells,
		ActiveHandle:        role.RoleID,
		ConsumedSequence:    map[int]bool{},
		DefendingHandles:    map[string]bool{},
		StatusEffects:       map[string]BattleStatusEffects{},
		StoredPower:         map[string]int{},
		nextSequence:        1,
	}
	start := StartPush{
		BattleID:        battleID,
		QueueIndexTeam:  firstEnemyConfig.QueueIndexTeam,
		QueueIndexEnemy: firstEnemyConfig.QueueIndexEnemy,
		LastMapName:     mapName,
		SelfHandle:      role.RoleID,
		IsArena:         false,
		MapID:           mapID,
		EncounterID:     "classic-wild-" + mapID,
		EncounterLabel:  mapName + " " + encounterKind,
		StageFocusX:     request.StageFocusX,
		ReturnRoute:     defaultString(request.ReturnRoute, "town-placeholder"),
		Commands:        sourceBattleCommandDefinitions(role.Skills),
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

	runtime.PendingBuffInfos = nil
	commandID := strings.TrimSpace(request.CommandID)

	switch commandID {
	case CommandEscape:
		if !runtime.isSelfTarget(actor, request.TargetHandle) {
			return ActionResult{ErrorCode: "invalid_target"}
		}
		runtime.ConsumedSequence[request.Sequence] = true
		runtime.setStoredPower(actor.Handle, 0)
		runtime.Phase = PhasePlaying
		runtime.PendingStart = nil
		runtime.PendingOver = runtime.buildOver(CampEnemy, true)
		return ActionResult{
			Actions: []ActionPush{
				runtime.resolveSelfAction(actor, commandID, "逃跑", "escapeSuccess"),
			},
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
			runtime.resolveSelfAction(actor, commandID, "蓄力", "def"),
		})
	case CommandKuangBao:
		if !runtime.isBattleCommandAllowed(commandID) {
			return ActionResult{ErrorCode: "unsupported_command"}
		}
		if !runtime.isSelfTarget(actor, request.TargetHandle) {
			return ActionResult{ErrorCode: "invalid_target"}
		}
		runtime.ConsumedSequence[request.Sequence] = true
		runtime.setStoredPower(actor.Handle, 0)
		profile := runtime.sourceSkillProfile("狂爆", 1)
		if profile.MPCost > 0 {
			actor.MP = maxInt(0, actor.MP-profile.MPCost)
		}
		runtime.applyKuangBao(actor.Handle)
		result := runtime.resolveEnemyTurnAndNextCommand(actor, []ActionPush{
			runtime.resolveSelfAction(actor, commandID, "狂爆", "w8/kb"),
		})
		result.BuffInfos = append(result.BuffInfos, runtime.resolveKuangBaoBuffInfo(actor))
		return result
	}

	if !runtime.isBattleCommandAllowed(commandID) {
		return ActionResult{ErrorCode: "unsupported_command"}
	}

	if normalizeBattleCommandID(commandID) == CommandHongYueZhan {
		targets := runtime.livingCells(CampEnemy)
		if len(targets) == 0 {
			return ActionResult{ErrorCode: "invalid_target"}
		}
		runtime.ConsumedSequence[request.Sequence] = true
		action := runtime.resolveAllTargetAttack(actor, targets, commandID)
		runtime.setStoredPower(actor.Handle, 0)
		return runtime.resolveEnemyTurnAndNextCommand(actor, []ActionPush{action})
	}
	if normalizeBattleCommandID(commandID) == CommandLeiHunZhan && runtime.powerFor(actor.Handle) < leiHunZhanRequiredPower {
		return ActionResult{ErrorCode: "insufficient_power"}
	}

	target := runtime.cellByHandle(request.TargetHandle)
	if target == nil || target.Camp != CampEnemy || target.HP <= 0 {
		return ActionResult{ErrorCode: "invalid_target"}
	}

	runtime.ConsumedSequence[request.Sequence] = true
	actions := []ActionPush{runtime.resolveAttack(actor, target, commandID)}
	runtime.setStoredPower(actor.Handle, 0)
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
	maxActorSingleHPLoss := 0
	for {
		if winner := runtime.resolveWinner(); winner != "" {
			runtime.Phase = PhaseFinished
			runtime.PendingStart = nil
			runtime.PendingOver = runtime.buildOver(winner)
			return ActionResult{
				Actions:   actions,
				BuffInfos: runtime.consumePendingBuffInfos(),
			}
		}

		team := runtime.firstLiving(CampTeam)
		for _, enemy := range runtime.livingCells(CampEnemy) {
			if team == nil {
				break
			}
			statusActions, skipTurn := runtime.resolveStatusStartActions(enemy)
			actions = append(actions, statusActions...)
			if runtime.resolveWinner() != "" {
				break
			}
			if enemy.HP <= 0 || skipTurn {
				team = runtime.firstLiving(CampTeam)
				continue
			}
			actions = append(actions, runtime.resolveEnemyRampageActions(enemy)...)
			beforeActorHP := actor.HP
			beforeHP := team.HP
			targetHandle := team.Handle
			commandID := runtime.enemyBattleCommand(enemy, team)
			actions = append(actions, runtime.resolveEnemyCommandActions(enemy, team, commandID)...)
			runtime.setStoredPower(enemy.Handle, 0)
			if targetHandle == actor.Handle {
				maxActorSingleHPLoss = maxInt(maxActorSingleHPLoss, beforeHP-team.HP)
			}
			if commandID == CommandEnemyFirePower || commandID == CommandEnemyDeadLight {
				maxActorSingleHPLoss = maxInt(maxActorSingleHPLoss, beforeActorHP-actor.HP)
			}
			team = runtime.firstLiving(CampTeam)
			if runtime.resolveWinner() != "" {
				break
			}
		}
		if winner := runtime.resolveWinner(); winner != "" {
			runtime.Phase = PhaseFinished
			runtime.PendingStart = nil
			runtime.PendingOver = runtime.buildOver(winner)
			return ActionResult{
				Actions:   actions,
				BuffInfos: runtime.consumePendingBuffInfos(),
			}
		}

		statusActions, skipTurn := runtime.resolveStatusStartActions(actor)
		actions = append(actions, statusActions...)
		if winner := runtime.resolveWinner(); winner != "" {
			runtime.Phase = PhaseFinished
			runtime.PendingStart = nil
			runtime.PendingOver = runtime.buildOver(winner)
			return ActionResult{
				Actions:   actions,
				BuffInfos: runtime.consumePendingBuffInfos(),
			}
		}

		runtime.Round += 1
		runtime.nextSequence += 1
		runtime.ActiveHandle = actor.Handle
		runtime.Phase = PhasePlaying
		runtime.advanceKuangBaoRound(actor.Handle)
		if skipTurn && runtime.hasActiveAutoContinueSkipStatus(actor.Handle) {
			continue
		}
		if !skipTurn {
			runtime.setStoredPower(actor.Handle, maxInt(
				runtime.powerFor(actor.Handle),
				storedPowerFromSingleHPLoss(maxActorSingleHPLoss, actor.MaxHP),
			))
		}
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
			Actions:   actions,
			BuffInfos: runtime.consumePendingBuffInfos(),
		}
	}
}

func (runtime *Runtime) resolveAttack(actor *CellInfoPush, target *CellInfoPush, commandID string) ActionPush {
	return runtime.resolveAttackWithMPCost(actor, target, commandID, true)
}

func (runtime *Runtime) resolveAllTargetAttack(actor *CellInfoPush, targets []*CellInfoPush, commandID string) ActionPush {
	actions := make([]ActionPush, 0, len(targets))
	refreshInfos := make([]CellInfoPush, 0, len(targets)+1)
	results := make([]ActionResultCodeEntry, 0, len(targets))
	actorRefreshAdded := false
	targetCamp := CampEnemy
	if actor != nil && actor.Camp == CampEnemy {
		targetCamp = CampTeam
	}
	for index := range targets {
		target := runtime.cellByHandle(targets[index].Handle)
		if target == nil || target.Camp != targetCamp || target.HP <= 0 {
			continue
		}
		action := runtime.resolveAttackWithMPCost(actor, target, commandID, len(actions) == 0)
		actions = append(actions, action)
		results = append(results, ActionResultCodeEntry{
			Handle:    target.Handle,
			StateCode: action.TargetActionStateCode,
		})
		for _, refresh := range action.RefreshInfos {
			if refresh.Handle == actor.Handle {
				if actorRefreshAdded {
					continue
				}
				actorRefreshAdded = true
			}
			refreshInfos = append(refreshInfos, refresh)
		}
	}
	if len(actions) == 0 {
		return ActionPush{}
	}
	action := actions[0]
	action.TargetHandle = "all"
	action.TargetActionResults = results
	action.RefreshInfos = refreshInfos
	return action
}

func (runtime *Runtime) resolveAttackWithMPCost(actor *CellInfoPush, target *CellInfoPush, commandID string, consumeMP bool) ActionPush {
	profile := runtime.battleCommandProfile(actor, commandID)
	targetInDef := runtime.DefendingHandles[target.Handle]
	defense := runtime.effectiveBattleDefense(target, targetInDef, profile.DefenseType)
	sourceActionLabel := profile.SourceActionLabel
	targetActionState := "normal"
	targetActionStateCode := "0"
	if runtime.resolveDodge(actor, target, commandID, profile) {
		if consumeMP && profile.MPCost > 0 {
			actor.MP = maxInt(0, actor.MP-profile.MPCost)
		}
		runtime.applyCapturedStunCounter(actor, target, commandID)
		refreshInfos := []CellInfoPush{*target}
		if consumeMP && profile.MPCost > 0 {
			refreshInfos = []CellInfoPush{*actor, *target}
		}
		return ActionPush{
			BattleID:              runtime.BattleID,
			ActorHandle:           actor.Handle,
			TargetHandle:          target.Handle,
			CommandID:             commandID,
			ActionName:            profile.ActionName,
			SourceMode:            sourceBattleActionMode(profile.SourceType),
			SourceActionLabel:     sourceActionLabel,
			TargetInDef:           targetInDef,
			TargetActionState:     "dog",
			TargetActionStateCode: "1",
			Damage:                0,
			TargetHP:              target.HP,
			TargetDead:            target.HP <= 0,
			RefreshInfos:          refreshInfos,
			Round:                 runtime.Round,
			Sequence:              runtime.nextSequence,
		}
	}
	if profile.TargetMPDamage > 0 {
		beforeTargetMP := target.MP
		target.MP = maxInt(0, target.MP-profile.TargetMPDamage)
		delete(runtime.DefendingHandles, target.Handle)
		if consumeMP && profile.MPCost > 0 {
			actor.MP = maxInt(0, actor.MP-profile.MPCost)
		}
		refreshInfos := []CellInfoPush{*target}
		if (consumeMP && profile.MPCost > 0) || beforeTargetMP != target.MP {
			refreshInfos = []CellInfoPush{*target}
			if consumeMP && profile.MPCost > 0 {
				refreshInfos = []CellInfoPush{*actor, *target}
			}
		}
		return ActionPush{
			BattleID:              runtime.BattleID,
			ActorHandle:           actor.Handle,
			TargetHandle:          target.Handle,
			CommandID:             commandID,
			ActionName:            profile.ActionName,
			SourceMode:            sourceBattleActionMode(profile.SourceType),
			SourceActionLabel:     sourceActionLabel,
			TargetInDef:           targetInDef,
			TargetActionState:     targetActionState,
			TargetActionStateCode: targetActionStateCode,
			Damage:                0,
			TargetHP:              target.HP,
			TargetMP:              target.MP,
			TargetDead:            target.HP <= 0,
			RefreshInfos:          refreshInfos,
			Round:                 runtime.Round,
			Sequence:              runtime.nextSequence,
		}
	}
	damage := runtime.baseBattleDamage(actor, profile, defense)
	if runtime.resolveCriticalHit(actor, target, commandID, profile) {
		damage *= 2
		targetActionState = "fat"
		targetActionStateCode = "2"
	}
	beforeTargetHP := target.HP
	target.HP = maxInt(0, target.HP-damage)
	delete(runtime.DefendingHandles, target.Handle)
	runtime.applyStoredPowerFromSingleHPLoss(target, beforeTargetHP-target.HP)
	if consumeMP && profile.MPCost > 0 {
		actor.MP = maxInt(0, actor.MP-profile.MPCost)
	}
	if profile.LifeStealChance > 0 && profile.LifeStealRatio > 0 && damage > 0 && runtime.resolveLifeSteal(actor, target, commandID, profile) {
		actor.HP = clampInt(actor.HP+int(math.Floor(float64(damage)*profile.LifeStealRatio)), 0, actor.MaxHP)
	}
	if target.HP > 0 && runtime.resolveStatusApply(actor, target, commandID, profile) {
		effect := BattleStatusEffect{
			Name:           profile.StatusName,
			Display:        profile.StatusDisplay,
			Description:    profile.StatusDescription,
			Rounds:         profile.StatusRounds,
			SourceHandle:   actor.Handle,
			SourceSkill:    profile.ActionName,
			AppliedAction:  sourceActionLabel,
			SourceAttack:   actor.Attack,
			TickMinPercent: profile.StatusTickMin,
			TickMaxPercent: profile.StatusTickMax,
			SkipTurn:       profile.SkipTurn,
		}
		runtime.applyStatusEffect(target.Handle, effect)
		runtime.PendingBuffInfos = append(runtime.PendingBuffInfos, runtime.resolveStatusBuffInfo(actor, target, effect))
	}
	runtime.applyCapturedStunCounter(actor, target, commandID)
	refreshInfos := []CellInfoPush{*target}
	if (consumeMP && profile.MPCost > 0) || profile.LifeStealChance > 0 {
		refreshInfos = []CellInfoPush{*actor, *target}
	}
	return ActionPush{
		BattleID:              runtime.BattleID,
		ActorHandle:           actor.Handle,
		TargetHandle:          target.Handle,
		CommandID:             commandID,
		ActionName:            profile.ActionName,
		SourceMode:            sourceBattleActionMode(profile.SourceType),
		SourceActionLabel:     sourceActionLabel,
		TargetInDef:           targetInDef,
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

func (runtime *Runtime) battleCommandProfile(actor *CellInfoPush, commandID string) commandProfile {
	profile := commandProfile{
		ActionName:        "普通攻击",
		SourceType:        "oneE",
		SourceActionLabel: "nomalAtk",
		DamageMultiplier:  1,
		CanDodge:          true,
		CanFat:            true,
		DefenseType:       "physical",
	}
	if actor != nil && strings.TrimSpace(actor.CommandLabel) != "" {
		profile.ActionName = actor.CommandLabel
	}
	switch normalizeBattleCommandID(commandID) {
	case CommandMiZhan:
		return runtime.sourceSkillProfile("密斩", 1)
	case CommandDuoDuanZhan:
		return runtime.sourceSkillProfile("多段斩", 1)
	case CommandShiXueZhan:
		return runtime.sourceSkillProfile("嗜血斩", 1)
	case CommandKuangBao:
		return runtime.sourceSkillProfile("狂爆", 1)
	case CommandHongYueZhan:
		return runtime.sourceSkillProfile("红月斩", 1)
	case CommandXueQie:
		return runtime.sourceSkillProfile("血切", 1)
	case CommandLeiHunZhan:
		return runtime.sourceSkillProfile("奥义.雷魂斩", 1)
	case CommandEnemySlideCut:
		return commandProfile{
			ActionName:        "滑行斩",
			SourceType:        "oneE",
			SourceActionLabel: "slideCut",
			DamageMultiplier:  1,
			MPCost:            enemySlideCutMPCost,
			CanDodge:          true,
			CanFat:            true,
		}
	case CommandEnemyShadeCut:
		return commandProfile{
			ActionName:        "影刃",
			SourceType:        "oneE",
			SourceActionLabel: "shadeCut",
			DamageMultiplier:  1,
			MPCost:            enemyShadeCutMPCost,
			CanDodge:          true,
			CanFat:            true,
		}
	case CommandEnemyHelixAtk:
		return commandProfile{
			ActionName:        "螺旋锤杀",
			SourceType:        "oneE",
			SourceActionLabel: "helixAtk",
			DamageMultiplier:  enemyHelixAtkDamageMultiplier,
			MPCost:            enemyHelixAtkMPCost,
			CanDodge:          true,
			CanFat:            true,
		}
	case CommandEnemyPalsyAtk:
		return commandProfile{
			ActionName:        "蜂刺",
			SourceType:        "oneE",
			SourceActionLabel: "palsyAtk",
			DamageMultiplier:  1,
			CanDodge:          true,
			CanFat:            true,
			StatusName:        "麻痹",
			StatusRounds:      2,
			StatusChance:      enemyPalsyAtkStatusChance,
			StatusDescription: "眩晕并每回合损失气力",
			SkipTurn:          true,
		}
	case CommandEnemyFirePower:
		return commandProfile{
			ActionName:        "赤焰击",
			SourceType:        "all",
			SourceActionLabel: "firePower",
			DamageMultiplier:  enemyFirePowerDamageMultiplier,
			CanDodge:          true,
			CanFat:            false,
			DefenseType:       "direct",
		}
	case CommandEnemyDeadLight:
		return commandProfile{
			ActionName:        "死亡射线",
			SourceType:        "all",
			SourceActionLabel: "deadLight",
			DamageMultiplier:  0,
			CanDodge:          true,
			CanFat:            false,
			DefenseType:       "direct",
			TargetMPDamage:    enemyDeadLightMPDamage,
		}
	case CommandEnemyDoubleHit:
		return commandProfile{
			ActionName:        "双锤打",
			SourceType:        "oneE",
			SourceActionLabel: "doubleHit",
			DamageMultiplier:  1,
			CanDodge:          true,
			CanFat:            true,
		}
	case CommandNormalAttack, CommandEnemyAttack:
		if actor == nil || strings.TrimSpace(actor.CommandLabel) == "" {
			profile.ActionName = "普通攻击"
		}
		if actor != nil && strings.TrimSpace(actor.DamageDefenseType) != "" {
			profile.DefenseType = strings.TrimSpace(actor.DamageDefenseType)
		}
	}
	return profile
}

func (runtime *Runtime) enemyBattleCommand(enemy *CellInfoPush, target *CellInfoPush) string {
	if sourceEnemyCanFirePower(enemy) && runtime.resolveEnemySkillUse(enemy, target, CommandEnemyFirePower, enemyFirePowerChance) {
		return CommandEnemyFirePower
	}
	if sourceEnemyCanDeadLight(enemy) && runtime.resolveEnemySkillUse(enemy, target, CommandEnemyDeadLight, enemyDeadLightChance) {
		return CommandEnemyDeadLight
	}
	if sourceEnemyCanDoubleHit(enemy) && runtime.resolveEnemySkillUse(enemy, target, CommandEnemyDoubleHit, enemyDoubleHitChance) {
		return CommandEnemyDoubleHit
	}
	if sourceEnemyCanHelixAtk(enemy) && enemy.MP >= enemyHelixAtkMPCost && runtime.resolveEnemySkillUse(enemy, target, CommandEnemyHelixAtk, enemyHelixAtkChance) {
		return CommandEnemyHelixAtk
	}
	if sourceEnemyCanShadeCut(enemy) && enemy.MP >= enemyShadeCutMPCost && runtime.resolveEnemySkillUse(enemy, target, CommandEnemyShadeCut, enemyShadeCutChance) {
		return CommandEnemyShadeCut
	}
	if sourceEnemyCanSlideCut(enemy) && enemy.MP >= enemySlideCutMPCost && runtime.resolveEnemySkillUse(enemy, target, CommandEnemySlideCut, enemySlideCutChance) {
		return CommandEnemySlideCut
	}
	if sourceEnemyCanPalsyAtk(enemy) && runtime.resolveEnemySkillUse(enemy, target, CommandEnemyPalsyAtk, enemyPalsyAtkChance) {
		return CommandEnemyPalsyAtk
	}
	return CommandEnemyAttack
}

func (runtime *Runtime) resolveEnemySkillUse(actor *CellInfoPush, target *CellInfoPush, commandID string, chance int) bool {
	if chance <= 0 || runtime == nil || actor == nil || target == nil {
		return false
	}
	if chance >= 100 {
		return true
	}
	return runtime.hashBattleRollWithSalt(actor, target, commandID, "enemy-skill") < chance
}

func sourceEnemyCanSlideCut(enemy *CellInfoPush) bool {
	if enemy == nil {
		return false
	}
	return strings.TrimSpace(enemy.Name) == "盗贼" || strings.Contains(strings.ToLower(strings.TrimSpace(enemy.DisplayURL)), "monstermap/robber.swf")
}

func sourceEnemyCanShadeCut(enemy *CellInfoPush) bool {
	if enemy == nil {
		return false
	}
	return strings.TrimSpace(enemy.Name) == "单刀狼人" || strings.Contains(strings.ToLower(strings.TrimSpace(enemy.DisplayURL)), "monstermap/bigswordwolf.swf")
}

func sourceEnemyCanHelixAtk(enemy *CellInfoPush) bool {
	if enemy == nil {
		return false
	}
	return strings.TrimSpace(enemy.Name) == "蛤蟆精" || strings.Contains(strings.ToLower(strings.TrimSpace(enemy.DisplayURL)), "monstermap/cracktoad.swf")
}

func sourceEnemyCanPalsyAtk(enemy *CellInfoPush) bool {
	if enemy == nil {
		return false
	}
	normalizedDisplay := strings.ToLower(strings.TrimSpace(enemy.DisplayURL))
	return strings.TrimSpace(enemy.Name) == "毒蜂" || strings.Contains(normalizedDisplay, "monstermap/drughornets.swf")
}

func (runtime *Runtime) resolveEnemyCommandActions(enemy *CellInfoPush, target *CellInfoPush, commandID string) []ActionPush {
	if runtime == nil || enemy == nil || target == nil {
		return nil
	}
	profile := runtime.battleCommandProfile(enemy, commandID)
	if strings.TrimSpace(profile.SourceType) == "all" {
		action := runtime.resolveAllTargetAttack(enemy, runtime.livingCells(CampTeam), commandID)
		if strings.TrimSpace(action.ActionName) == "" {
			return nil
		}
		return []ActionPush{action}
	}
	return []ActionPush{runtime.resolveAttack(enemy, target, commandID)}
}

func (runtime *Runtime) resolveEnemyRampageActions(enemy *CellInfoPush) []ActionPush {
	if runtime == nil || enemy == nil || enemy.HP <= 0 || !sourceEnemyCanRampage(enemy) || enemy.MP < enemyRampageMPCost {
		return nil
	}
	elapsed := 0
	if enemy.MaxMP > 0 && enemy.MP <= enemy.MaxMP {
		elapsed = (enemy.MaxMP - enemy.MP) / enemyRampageMPCost
	}
	remaining := maxInt(1, enemyRampageMaxRounds-elapsed)
	roundField := maxInt(1, 9998-elapsed)
	runtime.PendingBuffInfos = append(runtime.PendingBuffInfos, BuffInfoPush{
		BattleID:      runtime.BattleID,
		ReleaseHandle: enemy.Handle,
		TargetHandle:  enemy.Handle,
		Name:          "暴走之力",
		Display:       "1595.png",
		Description:   fmt.Sprintf("命中：+0<br/><font color='#FF00FF'>还有 %d 回合暴走!</font>", remaining),
		Round:         roundField,
	})
	enemy.MP = maxInt(0, enemy.MP-enemyRampageMPCost)
	return []ActionPush{runtime.resolveSelfAction(enemy, CommandEnemyRampage, "暴走之力", "battleStand")}
}

func sourceEnemyCanRampage(enemy *CellInfoPush) bool {
	if enemy == nil {
		return false
	}
	normalizedDisplay := strings.ToLower(strings.TrimSpace(enemy.DisplayURL))
	switch strings.TrimSpace(enemy.Name) {
	case "巨岩魔", "岩化魔人":
		return true
	default:
		return strings.Contains(normalizedDisplay, "monstermap/largerock.swf") || strings.Contains(normalizedDisplay, "monstermap/magicrockman.swf")
	}
}

func sourceEnemyCanFirePower(enemy *CellInfoPush) bool {
	if enemy == nil {
		return false
	}
	normalizedDisplay := strings.ToLower(strings.TrimSpace(enemy.DisplayURL))
	return strings.TrimSpace(enemy.Name) == "巨岩魔" || strings.Contains(normalizedDisplay, "monstermap/largerock.swf")
}

func sourceEnemyCanDeadLight(enemy *CellInfoPush) bool {
	if enemy == nil {
		return false
	}
	normalizedDisplay := strings.ToLower(strings.TrimSpace(enemy.DisplayURL))
	return strings.TrimSpace(enemy.Name) == "岩化魔人" || strings.Contains(normalizedDisplay, "monstermap/magicrockman.swf")
}

func sourceEnemyCanDoubleHit(enemy *CellInfoPush) bool {
	if enemy == nil {
		return false
	}
	normalizedDisplay := strings.ToLower(strings.TrimSpace(enemy.DisplayURL))
	return strings.TrimSpace(enemy.Name) == "岩化魔人" || strings.Contains(normalizedDisplay, "monstermap/magicrockman.swf")
}

func sourceEnemyCanStunCounter(enemy *CellInfoPush) bool {
	if enemy == nil {
		return false
	}
	normalizedDisplay := strings.ToLower(strings.TrimSpace(enemy.DisplayURL))
	switch strings.TrimSpace(enemy.Name) {
	case "爆骨猪", "晶石怪":
		return true
	default:
		return strings.Contains(normalizedDisplay, "monstermap/bomepig.swf") || strings.Contains(normalizedDisplay, "monstermap/crystalrock.swf")
	}
}

func (runtime *Runtime) isBattleCommandAllowed(commandID string) bool {
	switch normalizeBattleCommandID(commandID) {
	case CommandNormalAttack:
		return true
	case CommandMiZhan:
		if len(runtime.RoleSkills) == 0 {
			return true
		}
		return runtime.hasRoleSkill("密斩")
	case CommandDuoDuanZhan:
		return runtime.hasRoleSkill("多段斩")
	case CommandShiXueZhan:
		return runtime.hasRoleSkill("嗜血斩")
	case CommandKuangBao:
		return runtime.hasRoleSkill("狂爆")
	case CommandHongYueZhan:
		return runtime.hasRoleSkill("红月斩")
	case CommandXueQie:
		return runtime.hasRoleSkill("血切")
	case CommandLeiHunZhan:
		return runtime.hasRoleSkill("奥义.雷魂斩")
	default:
		return false
	}
}

func (runtime *Runtime) sourceSkillProfile(name string, fallbackLevel int) commandProfile {
	skill, ok := runtime.roleSkillByName(name)
	if !ok {
		skill = fallbackSourceBattleSkill(name, fallbackLevel)
	}
	profile := sourceBattleSkillProfile(skill)
	if strings.TrimSpace(profile.ActionName) == "" {
		return commandProfile{
			ActionName:        "普通攻击",
			SourceActionLabel: "nomalAtk",
			DamageMultiplier:  1,
			CanDodge:          true,
			CanFat:            true,
		}
	}
	return profile
}

func (runtime *Runtime) hasRoleSkill(name string) bool {
	_, ok := runtime.roleSkillByName(name)
	return ok
}

func (runtime *Runtime) roleSkillByName(name string) (session.RoleSkill, bool) {
	if runtime == nil {
		return session.RoleSkill{}, false
	}
	normalizedName := strings.TrimSpace(name)
	for _, skill := range runtime.RoleSkills {
		if strings.TrimSpace(skill.Name) == normalizedName {
			return skill, true
		}
	}
	return session.RoleSkill{}, false
}

func sourceBattleSkillProfile(skill session.RoleSkill) commandProfile {
	name := strings.TrimSpace(skill.Name)
	level := skill.Level
	if level <= 0 {
		level = 1
	}
	description := sourceBattleSkillProfileDescription(name, level, skill.Description)
	profile := commandProfile{
		ActionName:        name,
		SourceType:        sourceBattleSkillSourceType(name, skill.Type),
		SourceActionLabel: sourceBattleSkillActionLabel(name, level),
		DamageMultiplier:  sourceBattleSkillDamageMultiplier(description),
		MPCost:            sourceBattleSkillMPCost(description),
		CanDodge:          true,
		CanFat:            true,
	}
	if profile.DamageMultiplier <= 0 {
		profile.DamageMultiplier = fallbackSourceBattleSkillMultiplier(name, level)
	}
	if profile.MPCost <= 0 {
		profile.MPCost = fallbackSourceBattleSkillMPCost(name, level)
	}
	if name == "嗜血斩" {
		profile.LifeStealChance = sourceBattleSkillLifeStealChance(description)
		profile.LifeStealRatio = sourceBattleSkillLifeStealRatio(description)
		if profile.LifeStealChance <= 0 {
			profile.LifeStealChance = fallbackShiXueLifeStealChance(level)
		}
		if profile.LifeStealRatio <= 0 {
			profile.LifeStealRatio = 0.7
		}
	}
	if name == "血切" {
		profile.StatusName = "外伤"
		profile.StatusRounds = 4
		profile.StatusChance = fallbackXueQieWoundChance(level)
		profile.StatusDescription = "每回合损失气力为角色物理攻击的25%~30%"
		profile.StatusTickMin = 25
		profile.StatusTickMax = 30
	}
	return profile
}

func sourceBattleSkillProfileDescription(name string, level int, description string) string {
	if captured := fallbackSourceBattleSkill(name, level); strings.TrimSpace(captured.Description) != "" {
		return captured.Description
	}
	return description
}

func sourceBattleCommandDefinitions(skills []session.RoleSkill) []CommandDefinition {
	commands := []CommandDefinition{
		{
			ID:                CommandNormalAttack,
			Kind:              "skill",
			Label:             "普通攻击",
			SourceType:        "oneE",
			ActionName:        "普通攻击",
			SourceActionLabel: "nomalAtk",
			Target:            "enemy",
			DamageMultiplier:  1,
		},
	}
	seen := map[string]bool{"普通攻击": true}
	for _, skill := range skills {
		normalizedName := strings.TrimSpace(skill.Name)
		if seen[normalizedName] {
			continue
		}
		commandID := sourceBattleSkillCommandID(normalizedName)
		if commandID == "" {
			continue
		}
		profile := sourceBattleSkillProfile(skill)
		if strings.TrimSpace(profile.ActionName) == "" {
			continue
		}
		commands = append(commands, CommandDefinition{
			ID:                commandID,
			Kind:              "skill",
			Label:             normalizedName,
			SourceType:        sourceBattleSkillSourceType(normalizedName, skill.Type),
			ActionName:        profile.ActionName,
			SourceActionLabel: profile.SourceActionLabel,
			Target:            sourceBattleCommandTarget(sourceBattleSkillSourceType(normalizedName, skill.Type)),
			DamageMultiplier:  profile.DamageMultiplier,
			MPCost:            profile.MPCost,
		})
		seen[normalizedName] = true
	}
	commands = append(commands,
		CommandDefinition{
			ID:                CommandStore,
			Kind:              "store",
			Label:             "蓄魂",
			ActionName:        "蓄魂",
			SourceActionLabel: "def",
			Target:            "self",
		},
		CommandDefinition{
			ID:                CommandDefense,
			Kind:              "defense",
			Label:             "防御",
			ActionName:        "防御",
			SourceActionLabel: "def",
			Target:            "self",
		},
		CommandDefinition{
			ID:                CommandEscape,
			Kind:              "escape",
			Label:             "逃跑",
			ActionName:        "逃跑",
			SourceActionLabel: "escapeSuccess",
			Target:            "self",
		},
	)
	return commands
}

func sourceBattleSkillCommandID(name string) string {
	switch strings.TrimSpace(name) {
	case "密斩":
		return CommandMiZhan
	case "多段斩":
		return CommandDuoDuanZhan
	case "嗜血斩":
		return CommandShiXueZhan
	case "狂爆":
		return CommandKuangBao
	case "红月斩":
		return CommandHongYueZhan
	case "血切":
		return CommandXueQie
	case "奥义.雷魂斩":
		return CommandLeiHunZhan
	default:
		return ""
	}
}

func sourceBattleSkillSourceType(name string, fallbackType string) string {
	switch strings.TrimSpace(name) {
	case "密斩", "多段斩", "嗜血斩", "血切", "奥义.雷魂斩":
		return "oneE"
	case "狂爆":
		return "own"
	case "红月斩":
		return "all"
	default:
		return defaultString(strings.TrimSpace(fallbackType), "oneE")
	}
}

func sourceBattleActionMode(sourceType string) string {
	switch strings.TrimSpace(sourceType) {
	case "oneE", "all":
		return "1"
	default:
		return "0"
	}
}

func sourceBattleCommandTarget(sourceType string) string {
	switch strings.TrimSpace(sourceType) {
	case "own":
		return "self"
	default:
		return "enemy"
	}
}

var (
	sourceMPCostPattern          = regexp.MustCompile(`&2@(\d+)`)
	sourceImproveDamagePattern   = regexp.MustCompile(`提升(\d+)%的物理伤害`)
	sourcePercentDamagePattern   = regexp.MustCompile(`造成(\d+)%的物理伤害`)
	sourceLifeStealChancePattern = regexp.MustCompile(`有(\d+)%机率`)
	sourceLifeStealRatioPattern  = regexp.MustCompile(`伤害的(\d+)%`)
)

func normalizeBattleCommandID(commandID string) string {
	switch strings.TrimSpace(commandID) {
	case "密斩":
		return CommandMiZhan
	case "多段斩":
		return CommandDuoDuanZhan
	case "嗜血斩":
		return CommandShiXueZhan
	case "狂爆":
		return CommandKuangBao
	case "红月斩":
		return CommandHongYueZhan
	case "血切":
		return CommandXueQie
	case "奥义.雷魂斩":
		return CommandLeiHunZhan
	default:
		return strings.TrimSpace(commandID)
	}
}

func fallbackSourceBattleSkill(name string, level int) session.RoleSkill {
	if level <= 0 {
		level = 1
	}
	switch strings.TrimSpace(name) {
	case "密斩":
		return session.RoleSkill{
			Name:        "密斩",
			Level:       1,
			Type:        "oneE",
			Icon:        "426.png",
			Description: "f_s_密斩&9@单体·攻击&7@3&10@单刀/单斧&22@战斗&2@5&4@提升40%的物理伤害",
		}
	case "多段斩":
		return session.RoleSkill{
			Name:        "多段斩",
			Level:       level,
			Type:        "oneE",
			Icon:        "178.png",
			Description: fallbackDuoDuanDescription(level),
		}
	case "嗜血斩":
		return session.RoleSkill{
			Name:        "嗜血斩",
			Level:       level,
			Type:        "oneE",
			Icon:        "179.png",
			Description: fallbackShiXueDescription(level),
		}
	case "狂爆":
		return session.RoleSkill{
			Name:        "狂爆",
			Level:       level,
			Type:        "own",
			Icon:        "646.png",
			Description: "f_s_狂爆^5BC46D&9@单体·状态&8@战士 &10@单刀&22@战斗&2@15&4@3回合内物理攻击力翻倍&0;并降低100%的物理防御",
		}
	case "红月斩":
		return session.RoleSkill{
			Name:        "红月斩",
			Level:       level,
			Type:        "all",
			Icon:        "181.png",
			Description: fallbackHongYueDescription(level),
		}
	case "血切":
		return session.RoleSkill{
			Name:        "血切",
			Level:       level,
			Type:        "oneE",
			Icon:        "182.png",
			Description: fallbackXueQieDescription(level),
		}
	case "奥义.雷魂斩":
		return session.RoleSkill{
			Name:        "奥义.雷魂斩",
			Level:       level,
			Type:        "oneE",
			Icon:        "183.png",
			Description: fallbackLeiHunZhanDescription(level),
		}
	default:
		return session.RoleSkill{}
	}
}

func fallbackDuoDuanDescription(level int) string {
	switch level {
	case 2:
		return "f_s_多段斩^ffffff&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@10&4@提升60%的物理伤害"
	case 3:
		return "f_s_多段斩^ffffff&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@12&4@提升65%的物理伤害"
	case 4:
		return "f_s_多段斩^ffffff&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@14&4@提升70%的物理伤害"
	case 5:
		return "f_s_多段斩^ffffff&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@16&4@提升75%的物理伤害"
	default:
		return "f_s_多段斩^ffffff&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@8&4@提升55%的物理伤害"
	}
}

func fallbackShiXueDescription(level int) string {
	switch level {
	case 2:
		return "f_s_嗜血斩^5BC46D&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@26&4@对敌人造成94%的物理伤害&0;并有84%机率将对敌人造成伤害的70%转换为气力</font>"
	case 3:
		return "f_s_嗜血斩^5BC46D&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@28&4@对敌人造成96%的物理伤害&0;并有86%机率将对敌人造成伤害的70%转换为气力</font>"
	default:
		return "f_s_嗜血斩^5BC46D&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@24&4@对敌人造成92%的物理伤害&0;并有82%机率将对敌人造成伤害的70%转换为气力</font>"
	}
}

func fallbackHongYueDescription(level int) string {
	switch level {
	default:
		return "f_s_红月斩^ffffff&9@群体·攻击&8@战士 &10@单刀&22@战斗&2@40&4@对所有敌人造成72%的物理伤害"
	}
}

func fallbackXueQieDescription(level int) string {
	switch level {
	default:
		return "f_s_血切^5BC46D&9@单体·状态&8@战士 &10@单刀&22@战斗&2@19&4@对敌人造成30%的物理伤害&0;击中敌人时有80%的机率使对方进入外伤状态4回合<br>(每回合损失气力为角色物理攻击的25%~30%)"
	}
}

func fallbackLeiHunZhanDescription(level int) string {
	switch level {
	default:
		return "f_s_奥义.雷魂斩^00ccff&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@24&4@<font color='#00cc00'>特殊发动条件:需要3格魂元</font><br>提升240%的物理伤害"
	}
}

func sourceBattleSkillActionLabel(name string, level int) string {
	switch strings.TrimSpace(name) {
	case "密斩":
		return "w8/manycut"
	case "多段斩":
		if level >= 3 {
			return "w8/ddz2"
		}
		return "w8/ddz1"
	case "嗜血斩":
		if level >= 3 {
			return "w8/xyz2"
		}
		return "w8/xyz1"
	case "狂爆":
		return "w8/kb"
	case "红月斩":
		return "w8/redMoonAtk"
	case "血切":
		return "w8/cutBlood"
	case "奥义.雷魂斩":
		return "w8/thunderSoulAtk"
	case "普通攻击":
		return "nomalAtk"
	default:
		return ""
	}
}

func sourceBattleSkillMPCost(description string) int {
	return firstSourcePercentInt(sourceMPCostPattern, description)
}

func sourceBattleSkillDamageMultiplier(description string) float64 {
	if value := firstSourcePercentInt(sourceImproveDamagePattern, description); value > 0 {
		return 1 + float64(value)/100
	}
	if value := firstSourcePercentInt(sourcePercentDamagePattern, description); value > 0 {
		return float64(value) / 100
	}
	return 0
}

func sourceBattleSkillLifeStealChance(description string) int {
	return firstSourcePercentInt(sourceLifeStealChancePattern, description)
}

func sourceBattleSkillLifeStealRatio(description string) float64 {
	value := firstSourcePercentInt(sourceLifeStealRatioPattern, description)
	if value <= 0 {
		return 0
	}
	return float64(value) / 100
}

func firstSourcePercentInt(pattern *regexp.Regexp, text string) int {
	match := pattern.FindStringSubmatch(text)
	if len(match) < 2 {
		return 0
	}
	value, err := strconv.Atoi(match[1])
	if err != nil {
		return 0
	}
	return value
}

func fallbackSourceBattleSkillMultiplier(name string, level int) float64 {
	switch strings.TrimSpace(name) {
	case "密斩":
		return 1.4
	case "多段斩":
		switch level {
		case 2:
			return 1.6
		case 3:
			return 1.65
		case 4:
			return 1.7
		case 5:
			return 1.75
		default:
			return 1.55
		}
	case "嗜血斩":
		switch level {
		case 2:
			return 0.94
		case 3:
			return 0.96
		default:
			return 0.92
		}
	case "狂爆":
		return 0
	case "红月斩":
		return 0.72
	case "血切":
		return 0.3
	case "奥义.雷魂斩":
		return 3.4
	default:
		return 1
	}
}

func fallbackSourceBattleSkillMPCost(name string, level int) int {
	switch strings.TrimSpace(name) {
	case "密斩":
		return 5
	case "多段斩":
		switch level {
		case 2:
			return 10
		case 3:
			return 12
		case 4:
			return 14
		case 5:
			return 16
		default:
			return 8
		}
	case "嗜血斩":
		switch level {
		case 2:
			return 26
		case 3:
			return 28
		default:
			return 24
		}
	case "狂爆":
		return 15
	case "红月斩":
		return 40
	case "血切":
		return 19
	case "奥义.雷魂斩":
		return 24
	default:
		return 0
	}
}

func fallbackShiXueLifeStealChance(level int) int {
	switch level {
	case 2:
		return 84
	case 3:
		return 86
	default:
		return 82
	}
}

func fallbackXueQieWoundChance(level int) int {
	switch level {
	case 2:
		return 85
	default:
		return 80
	}
}

func (runtime *Runtime) effectiveBattleDefense(target *CellInfoPush, targetInDef bool, defenseType string) int {
	if target == nil {
		return 0
	}
	if strings.EqualFold(strings.TrimSpace(defenseType), "direct") {
		return 0
	}
	if runtime.hasKuangBao(target.Handle) {
		return 0
	}
	baseDefense := target.Defense
	if strings.EqualFold(strings.TrimSpace(defenseType), "magic") {
		baseDefense = target.MgcDefense
	}
	if targetInDef {
		return baseDefense * 2
	}
	return baseDefense
}

func (runtime *Runtime) baseBattleDamage(actor *CellInfoPush, profile commandProfile, defense int) int {
	if actor == nil {
		return 0
	}
	attack := actor.Attack
	if runtime.hasKuangBao(actor.Handle) {
		attack *= 2
	}
	return maxInt(1, int(math.Round(float64(attack)*profile.DamageMultiplier))-defense)
}

func (runtime *Runtime) applyStoredPowerFromSingleHPLoss(target *CellInfoPush, hpLoss int) {
	if runtime == nil || target == nil || hpLoss <= 0 {
		return
	}
	runtime.setStoredPower(target.Handle, maxInt(
		runtime.powerFor(target.Handle),
		storedPowerFromSingleHPLoss(hpLoss, target.MaxHP),
	))
}

func (runtime *Runtime) resolveCriticalHit(actor *CellInfoPush, target *CellInfoPush, commandID string, profile commandProfile) bool {
	if !profile.CanFat || actor == nil || target == nil || actor.Fat <= 0 {
		return false
	}
	chance := battleCriticalChancePercent(actor.Fat, target.Dog)
	return runtime.hashBattleRollWithSalt(actor, target, commandID, "fat") < chance
}

func (runtime *Runtime) resolveLifeSteal(actor *CellInfoPush, target *CellInfoPush, commandID string, profile commandProfile) bool {
	if profile.LifeStealChance <= 0 || actor == nil || target == nil {
		return false
	}
	if profile.LifeStealChance >= 100 {
		return true
	}
	return runtime.hashBattleRollWithSalt(actor, target, commandID, "lifesteal") < profile.LifeStealChance
}

func (runtime *Runtime) resolveStatusApply(actor *CellInfoPush, target *CellInfoPush, commandID string, profile commandProfile) bool {
	if strings.TrimSpace(profile.StatusName) == "" || profile.StatusRounds <= 0 || profile.StatusChance <= 0 || actor == nil || target == nil {
		return false
	}
	if profile.StatusChance >= 100 {
		return true
	}
	return runtime.hashBattleRollWithSalt(actor, target, commandID, "status:"+strings.TrimSpace(profile.StatusName)) < profile.StatusChance
}

func (runtime *Runtime) resolveDodge(actor *CellInfoPush, target *CellInfoPush, commandID string, profile commandProfile) bool {
	if !profile.CanDodge || actor == nil || target == nil || target.Dog <= 0 {
		return false
	}
	if actor.Hit <= 0 {
		return true
	}
	chance := battleDodgeChancePercent(actor.Hit, target.Dog)
	return runtime.hashBattleRollWithSalt(actor, target, commandID, "dog") < chance
}

func (runtime *Runtime) applyStatusEffect(handle string, effect BattleStatusEffect) {
	if runtime.StatusEffects == nil {
		runtime.StatusEffects = map[string]BattleStatusEffects{}
	}
	effect.Name = strings.TrimSpace(effect.Name)
	if handle == "" || effect.Name == "" || effect.Rounds <= 0 {
		return
	}
	effects := runtime.StatusEffects[handle]
	if effects.Effects == nil {
		effects.Effects = map[string]BattleStatusEffect{}
	}
	effects.Effects[effect.Name] = effect
	runtime.StatusEffects[handle] = effects
}

func (runtime *Runtime) applyCapturedStunCounter(actor *CellInfoPush, target *CellInfoPush, commandID string) bool {
	if runtime == nil || actor == nil || target == nil || actor.Camp != CampTeam || target.Camp != CampEnemy || target.HP <= 0 {
		return false
	}
	if !sourceEnemyCanStunCounter(target) {
		return false
	}
	if runtime.hashBattleRollWithSalt(target, actor, commandID, "status:眩晕") >= enemyStunCounterChance {
		return false
	}
	effect := BattleStatusEffect{
		Name:          "眩晕",
		Display:       "9.png",
		Description:   "眩晕无法行动",
		Rounds:        2,
		SourceHandle:  target.Handle,
		SourceSkill:   "眩晕",
		AppliedAction: "yun",
		SkipTurn:      true,
	}
	runtime.applyStatusEffect(actor.Handle, effect)
	runtime.PendingBuffInfos = append(runtime.PendingBuffInfos, runtime.resolveStatusBuffInfo(target, actor, effect))
	return true
}

func (runtime *Runtime) resolveStatusStartActions(actor *CellInfoPush) ([]ActionPush, bool) {
	if runtime == nil || actor == nil || actor.HP <= 0 || runtime.StatusEffects == nil {
		return nil, false
	}
	effects := runtime.StatusEffects[actor.Handle]
	if len(effects.Effects) == 0 {
		return nil, false
	}

	actions := make([]ActionPush, 0, len(effects.Effects))
	skipTurn := false
	for _, name := range sortedStatusEffectNames(effects.Effects) {
		if actor.HP <= 0 {
			break
		}
		effect := effects.Effects[name]
		if effect.Rounds <= 0 {
			delete(effects.Effects, name)
			continue
		}
		switch strings.TrimSpace(effect.Name) {
		case "外伤":
			action := runtime.resolveWoundStatusAction(actor, effect)
			if action != nil {
				actions = append(actions, *action)
			}
		case "麻痹":
			action := runtime.resolveSkipTurnStatusAction(actor, effect)
			if action != nil {
				actions = append(actions, *action)
				if effect.SkipTurn {
					skipTurn = true
				}
			}
		case "眩晕":
			action := runtime.resolveSkipTurnStatusAction(actor, effect)
			if action != nil {
				actions = append(actions, *action)
				if effect.SkipTurn {
					skipTurn = true
				}
			}
		}
		effect.Rounds -= 1
		if effect.Rounds <= 0 {
			delete(effects.Effects, name)
		} else {
			effects.Effects[name] = effect
		}
	}
	if len(effects.Effects) == 0 && effects.KuangBaoRounds <= 0 {
		delete(runtime.StatusEffects, actor.Handle)
	} else {
		runtime.StatusEffects[actor.Handle] = effects
	}
	return actions, skipTurn
}

func (runtime *Runtime) hasActiveAutoContinueSkipStatus(handle string) bool {
	if runtime == nil || runtime.StatusEffects == nil {
		return false
	}
	effects := runtime.StatusEffects[handle]
	for _, effect := range effects.Effects {
		if effect.SkipTurn && effect.Rounds > 0 && strings.TrimSpace(effect.Name) == "眩晕" {
			return true
		}
	}
	return false
}

func (runtime *Runtime) resolveWoundStatusAction(target *CellInfoPush, effect BattleStatusEffect) *ActionPush {
	if runtime == nil || target == nil || target.HP <= 0 {
		return nil
	}
	sourceAttack := effect.SourceAttack
	if sourceAttack <= 0 {
		sourceAttack = target.Attack
	}
	minPercent := effect.TickMinPercent
	maxPercent := effect.TickMaxPercent
	if minPercent <= 0 {
		minPercent = 25
	}
	if maxPercent < minPercent {
		maxPercent = minPercent
	}
	percent := minPercent
	if maxPercent > minPercent {
		roll := runtime.hashBattleRollWithSalt(target, target, "status-wound", effect.SourceHandle+":"+effect.SourceSkill)
		percent += roll % (maxPercent - minPercent + 1)
	}
	damage := maxInt(1, int(math.Round(float64(sourceAttack)*float64(percent)/100)))
	target.HP = maxInt(0, target.HP-damage)
	delete(runtime.DefendingHandles, target.Handle)
	return &ActionPush{
		BattleID:              runtime.BattleID,
		ActorHandle:           target.Handle,
		TargetHandle:          target.Handle,
		ActionName:            "外伤",
		SourceMode:            "0",
		SourceActionLabel:     "battleStand",
		TargetActionState:     "none",
		TargetActionStateCode: "3",
		Damage:                damage,
		TargetHP:              target.HP,
		TargetDead:            target.HP <= 0,
		RefreshInfos:          []CellInfoPush{*target},
		Round:                 runtime.Round,
		Sequence:              runtime.nextSequence,
	}
}

func (runtime *Runtime) resolveSkipTurnStatusAction(target *CellInfoPush, effect BattleStatusEffect) *ActionPush {
	if runtime == nil || target == nil || target.HP <= 0 {
		return nil
	}
	name := strings.TrimSpace(effect.Name)
	if name == "" {
		return nil
	}
	sourceActionLabel := "battleStand"
	if name == "眩晕" {
		sourceActionLabel = "yun"
	}
	return &ActionPush{
		BattleID:              runtime.BattleID,
		ActorHandle:           target.Handle,
		TargetHandle:          target.Handle,
		ActionName:            name,
		SourceMode:            "0",
		SourceActionLabel:     sourceActionLabel,
		TargetActionState:     "none",
		TargetActionStateCode: "3",
		Damage:                0,
		TargetHP:              target.HP,
		TargetMP:              target.MP,
		TargetDead:            target.HP <= 0,
		RefreshInfos:          []CellInfoPush{*target},
		Round:                 runtime.Round,
		Sequence:              runtime.nextSequence,
	}
}

func sortedStatusEffectNames(effects map[string]BattleStatusEffect) []string {
	names := make([]string, 0, len(effects))
	for name := range effects {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (runtime *Runtime) resolveStatusBuffInfo(actor *CellInfoPush, target *CellInfoPush, effect BattleStatusEffect) BuffInfoPush {
	buff := BuffInfoPush{
		BattleID:      runtime.BattleID,
		ReleaseHandle: effect.SourceHandle,
		TargetHandle:  target.Handle,
		Name:          effect.Name,
		Display:       effect.Display,
		Description:   effect.Description,
		Round:         effect.Rounds,
	}
	if strings.TrimSpace(buff.ReleaseHandle) == "" && actor != nil {
		buff.ReleaseHandle = actor.Handle
	}
	return buff
}

func (runtime *Runtime) consumePendingBuffInfos() []BuffInfoPush {
	if len(runtime.PendingBuffInfos) == 0 {
		return nil
	}
	buffInfos := append([]BuffInfoPush(nil), runtime.PendingBuffInfos...)
	runtime.PendingBuffInfos = nil
	return buffInfos
}

func (runtime *Runtime) applyKuangBao(handle string) {
	if runtime.StatusEffects == nil {
		runtime.StatusEffects = map[string]BattleStatusEffects{}
	}
	effects := runtime.StatusEffects[handle]
	effects.KuangBaoRounds = 3
	if effects.Effects == nil {
		effects.Effects = map[string]BattleStatusEffect{}
	}
	effects.Effects["热血"] = BattleStatusEffect{
		Name:         "热血",
		Display:      "24.png",
		Description:  "提升物理攻击&0;同时降低物理防御",
		Rounds:       3,
		SourceHandle: handle,
		SourceSkill:  "狂爆",
	}
	runtime.StatusEffects[handle] = effects
}

func (runtime *Runtime) resolveKuangBaoBuffInfo(actor *CellInfoPush) BuffInfoPush {
	return BuffInfoPush{
		BattleID:      runtime.BattleID,
		ReleaseHandle: actor.Handle,
		TargetHandle:  actor.Handle,
		Name:          "热血",
		Display:       "24.png",
		Description:   "提升物理攻击&0;同时降低物理防御",
		Round:         3,
	}
}

func (runtime *Runtime) hasKuangBao(handle string) bool {
	if runtime == nil || runtime.StatusEffects == nil {
		return false
	}
	return runtime.StatusEffects[handle].KuangBaoRounds > 0
}

func (runtime *Runtime) advanceKuangBaoRound(handle string) {
	if runtime == nil || runtime.StatusEffects == nil {
		return
	}
	effects := runtime.StatusEffects[handle]
	if effects.KuangBaoRounds <= 0 {
		return
	}
	effects.KuangBaoRounds -= 1
	if hotBlood, ok := effects.Effects["热血"]; ok {
		hotBlood.Rounds = effects.KuangBaoRounds
		if hotBlood.Rounds <= 0 {
			delete(effects.Effects, "热血")
		} else {
			effects.Effects["热血"] = hotBlood
		}
	}
	if effects.KuangBaoRounds <= 0 {
		effects.KuangBaoRounds = 0
		if len(effects.Effects) == 0 {
			delete(runtime.StatusEffects, handle)
			return
		}
		runtime.StatusEffects[handle] = effects
		return
	}
	runtime.StatusEffects[handle] = effects
}

func battleCriticalChancePercent(actorFat int, targetDog int) int {
	if actorFat <= 0 {
		return 0
	}
	if actorFat >= 100 {
		return 50
	}
	chance := int(math.Round(float64(actorFat)*0.45 - float64(maxInt(0, targetDog))/25 + 4))
	return clampInt(chance, 1, 80)
}

func battleDodgeChancePercent(actorHit int, targetDog int) int {
	if targetDog <= 0 {
		return 0
	}
	if actorHit <= 0 {
		return 100
	}
	chance := int(math.Round(8 + (float64(targetDog)-float64(actorHit)/2)/10))
	return clampInt(chance, 1, 80)
}

func (runtime *Runtime) hashBattleRoll(actor *CellInfoPush, target *CellInfoPush, commandID string) int {
	return runtime.hashBattleRollWithSalt(actor, target, commandID, "")
}

func (runtime *Runtime) hashBattleRollWithSalt(actor *CellInfoPush, target *CellInfoPush, commandID string, salt string) int {
	score := 0
	seed := fmt.Sprintf("%s:%d:%d:%s:%s:%s:%s", runtime.BattleID, runtime.Round, runtime.nextSequence, actor.Handle, target.Handle, commandID, salt)
	for _, char := range seed {
		score = (score*31 + int(char)) % 100
	}
	return score
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
		SourceMode:        "0",
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
		if strings.TrimSpace(targetHandle) != "" && !runtime.isSelfTarget(actor, targetHandle) {
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
		SourceMode:        "0",
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
	expDelta, items := runtime.sourceBattleRewards(winner, isEscaped)
	expDelta = runtime.applyExperienceLevelSuppression(expDelta, winner, isEscaped)
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
	return sourceBattleRewardsForEncounter(mapID, "", winner, escaped)
}

func sourceBattleRewardsForEncounter(mapID string, sourceMonsterHandle string, winner Camp, escaped bool) (int, []string) {
	if winner != CampTeam || escaped {
		return 0, []string{}
	}

	reward, ok := sourceBattleRewardConfigForEncounter(mapID, sourceMonsterHandle)
	if !ok || reward.Status != "confirmed" {
		return 0, []string{}
	}
	return reward.ExpDelta, rollSourceBattleRewardItems(reward.Items, reward.DropRates)
}

func (runtime *Runtime) sourceBattleRewards(winner Camp, escaped bool) (int, []string) {
	if runtime == nil {
		return 0, []string{}
	}
	if winner != CampTeam || escaped {
		return 0, []string{}
	}

	if reward, ok := sourceBattleRewardConfigForEncounter(runtime.MapID, runtime.SourceMonsterHandle); ok && reward.Status == "confirmed" {
		return reward.ExpDelta, rollSourceBattleRewardItems(reward.Items, reward.DropRates)
	}

	if reward, ok := runtime.sourceBattleRewardCandidate(); ok {
		return reward.ExpDelta, rollSourceBattleRewardItems(nil, reward.DropRates)
	}
	return 0, []string{}
}

func (runtime *Runtime) sourceBattleRewardCandidate() (sourceBattleRewardCandidateConfig, bool) {
	if runtime == nil {
		return sourceBattleRewardCandidateConfig{}, false
	}
	for _, cell := range runtime.Cells {
		if cell.Camp != CampEnemy {
			continue
		}
		if reward, ok := sourceBattleRewardCandidateForCell(runtime.MapID, cell.Name, cell.MaxHP); ok {
			return reward, true
		}
	}
	return sourceBattleRewardCandidateConfig{}, false
}

func rollSourceBattleRewardItems(fallbackItems []string, dropRates []sourceBattleRewardDropRate) []string {
	if len(dropRates) == 0 {
		return append([]string{}, fallbackItems...)
	}
	items := make([]string, 0, len(dropRates))
	for _, drop := range dropRates {
		if drop.ItemName == "" || drop.Numerator <= 0 || drop.Denominator <= 0 {
			continue
		}
		if drop.Numerator < drop.Denominator && sourceEncounterRoll(drop.Denominator) >= drop.Numerator {
			continue
		}
		items = append(items, formatSourceBattleRewardItemStack(drop.ItemName, drop.Quantity))
	}
	return items
}

func formatSourceBattleRewardItemStack(name string, quantity int) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if quantity <= 1 {
		return name + "x1"
	}
	return fmt.Sprintf("%sx%d", name, quantity)
}

func (runtime *Runtime) applyExperienceLevelSuppression(expDelta int, winner Camp, escaped bool) int {
	if runtime == nil || expDelta <= 0 || winner != CampTeam || escaped {
		return expDelta
	}
	playerLevel, ok := runtime.highestCampLevel(CampTeam)
	if !ok {
		return expDelta
	}
	monsterLevel, ok := runtime.highestCampLevel(CampEnemy)
	if !ok {
		return expDelta
	}
	if playerLevel > monsterLevel+7 {
		return 0
	}
	return expDelta
}

func (runtime *Runtime) highestCampLevel(camp Camp) (int, bool) {
	if runtime == nil {
		return 0, false
	}
	highest := 0
	for _, cell := range runtime.Cells {
		if cell.Camp != camp || cell.Level <= 0 {
			continue
		}
		highest = maxInt(highest, cell.Level)
	}
	return highest, highest > 0
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

func (runtime *Runtime) livingCells(camp Camp) []*CellInfoPush {
	cells := []*CellInfoPush{}
	for index := range runtime.Cells {
		if runtime.Cells[index].Camp == camp && runtime.Cells[index].HP > 0 {
			cells = append(cells, &runtime.Cells[index])
		}
	}
	return cells
}

func (runtime *Runtime) isSelfTarget(actor *CellInfoPush, targetHandle string) bool {
	return actor != nil && strings.TrimSpace(targetHandle) == actor.Handle
}

func (runtime *Runtime) powerFor(handle string) int {
	if runtime == nil {
		return 0
	}
	return clampInt(maxInt(1, runtime.StoredPower[handle]), 1, maxStoredPower)
}

func (runtime *Runtime) setStoredPower(handle string, value int) {
	if runtime.StoredPower == nil {
		runtime.StoredPower = map[string]int{}
	}
	runtime.StoredPower[handle] = clampInt(value, 0, maxStoredPower)
}

func storedPowerFromSingleHPLoss(hpLoss int, maxHP int) int {
	if maxHP <= 0 {
		return 0
	}
	return clampInt(int(math.Floor(float64(maxInt(0, hpLoss))*10/float64(maxHP))), 1, maxStoredPower)
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

func sourceEnemyForMap(mapID string) (CellInfoPush, bool) {
	config, ok := sourceEnemyConfigForMap(mapID)
	if !ok {
		return CellInfoPush{}, false
	}
	return config.Cell, true
}

func sourceEnemyConfigForEncounter(mapID string, stageFocusX float64) (sourceWildEnemyConfig, bool) {
	configs := sourceEnemyConfigsForEncounter(mapID, stageFocusX)
	if len(configs) <= 0 {
		return sourceWildEnemyConfig{}, false
	}
	return configs[0], true
}

func sourceEnemyConfigsForEncounter(mapID string, stageFocusX float64) []sourceWildEnemyConfig {
	configs := sourceEnemyConfigsForMap(mapID)
	if len(configs) <= 0 {
		return nil
	}
	selectedIndex := sourceEnemyCandidateIndex(stageFocusX, len(configs))
	selected := configs[selectedIndex]

	if strings.TrimSpace(mapID) == "52" && isSourceBossEnemyConfig(selected) {
		normal := selected
		for _, config := range configs {
			if !isSourceBossEnemyConfig(config) {
				normal = config
				break
			}
		}
		return []sourceWildEnemyConfig{selected, normal, normal, normal}
	}

	count := capturedSourceEncounterEnemyCount(mapID)
	if count <= 1 {
		return []sourceWildEnemyConfig{selected}
	}
	result := make([]sourceWildEnemyConfig, 0, count)
	for offset := 0; offset < count; offset++ {
		result = append(result, configs[(selectedIndex+offset)%len(configs)])
	}
	return result
}

func sourceEnemyCandidateIndex(stageFocusX float64, candidateCount int) int {
	if candidateCount <= 1 {
		return 0
	}
	sourceZone := int(math.Floor(math.Abs(math.Round(stageFocusX)) / 800))
	return sourceZone % candidateCount
}

func isSourceBossEnemyConfig(config sourceWildEnemyConfig) bool {
	return strings.HasSuffix(strings.TrimSpace(config.Vocation), "+")
}

func sourceEncounterHasBoss(configs []sourceWildEnemyConfig) bool {
	for _, config := range configs {
		if isSourceBossEnemyConfig(config) {
			return true
		}
	}
	return false
}

func capturedSourceEncounterEnemyCount(mapID string) int {
	switch strings.TrimSpace(mapID) {
	case "36", "49", "50", "51":
		return 1 + sourceEncounterRoll(2)
	default:
		return 1
	}
}

func (cell CellInfoPush) withBattleID(battleID string) CellInfoPush {
	cell.BattleID = battleID
	return cell
}

func (cell CellInfoPush) withBattleIDAndSlot(battleID string, slot int) CellInfoPush {
	cell.BattleID = battleID
	if slot > 0 {
		cell.Handle = fmt.Sprintf("%s_%d", cell.Handle, slot)
	}
	return cell
}

func queueIndexForMap(mapID string) int {
	config, ok := sourceEnemyConfigForMap(mapID)
	if !ok {
		return 0
	}
	return config.QueueIndexTeam
}

func enemyQueueIndexForMap(mapID string) int {
	config, ok := sourceEnemyConfigForMap(mapID)
	if !ok {
		return 0
	}
	return config.QueueIndexEnemy
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

func cloneBattleRoleSkills(skills []session.RoleSkill) []session.RoleSkill {
	if len(skills) == 0 {
		return nil
	}
	cloned := make([]session.RoleSkill, len(skills))
	copy(cloned, skills)
	return cloned
}

func ParseMapID(mapID string) (int, bool) {
	value, err := strconv.Atoi(strings.TrimSpace(mapID))
	return value, err == nil
}
