package battle

import (
	"fmt"
	"math"
	"regexp"
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

	CommandNormalAttack  = "skill-normal-attack"
	CommandMiZhan        = "skill-mi-zhan"
	CommandDuoDuanZhan   = "skill-duo-duan-zhan"
	CommandShiXueZhan    = "skill-shi-xue-zhan"
	CommandEnemyAttack   = "enemy-normal-attack"
	CommandEnemySlideCut = "enemy-slide-cut"
	CommandDefense       = "defense"
	CommandStore         = "battle-store"
	CommandEscape        = "battle-escape"
	CommandItem          = "battle-item"

	maxStoredPower      = 5
	enemySlideCutMPCost = 10
	enemySlideCutChance = 20
	defaultBattleHit    = 100
	defaultBattleDog    = 50
	defaultBattleFat    = 5
)

type StartRequest struct {
	MapID       string  `json:"mapId"`
	MapName     string  `json:"mapName"`
	StageFocusX float64 `json:"stageFocusX"`
	ReturnRoute string  `json:"returnRoute,omitempty"`
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
	BattleID              string         `json:"battleId,omitempty"`
	ActorHandle           string         `json:"actorHandle"`
	TargetHandle          string         `json:"targetHandle"`
	CommandID             string         `json:"commandId,omitempty"`
	ActionName            string         `json:"actionName"`
	SourceActionLabel     string         `json:"sourceActionLabel,omitempty"`
	TargetInDef           bool           `json:"targetInDef,omitempty"`
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
	RoleSkills       []session.RoleSkill
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

type commandProfile struct {
	ActionName        string
	SourceActionLabel string
	DamageMultiplier  float64
	MPCost            int
	CanDodge          bool
	CanFat            bool
	LifeStealChance   int
	LifeStealRatio    float64
}

func NewWildBattle(role session.RoleSummary, playerBase session.PlayerBaseData, request StartRequest) (*Runtime, StartBundle, bool) {
	mapID := strings.TrimSpace(request.MapID)
	enemy, ok := sourceEnemyForMap(mapID)
	if !ok {
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
	playerHit := defaultBattleHit
	playerDog := defaultBattleDog
	playerFat := defaultBattleFat
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
		RoleSkills:       cloneBattleRoleSkills(role.Skills),
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
	}

	if !runtime.isBattleCommandAllowed(commandID) {
		return ActionResult{ErrorCode: "unsupported_command"}
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
		actions = append(actions, runtime.resolveAttack(enemy, team, runtime.enemyBattleCommand(enemy, team)))
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
	profile := runtime.battleCommandProfile(actor, commandID)
	targetInDef := runtime.DefendingHandles[target.Handle]
	defense := effectiveBattleDefense(target, targetInDef)
	targetActionState := "normal"
	targetActionStateCode := "0"
	if runtime.resolveDodge(actor, target, commandID, profile) {
		if profile.MPCost > 0 {
			actor.MP = maxInt(0, actor.MP-profile.MPCost)
		}
		refreshInfos := []CellInfoPush{*target}
		if profile.MPCost > 0 {
			refreshInfos = []CellInfoPush{*actor, *target}
		}
		return ActionPush{
			BattleID:              runtime.BattleID,
			ActorHandle:           actor.Handle,
			TargetHandle:          target.Handle,
			CommandID:             commandID,
			ActionName:            profile.ActionName,
			SourceActionLabel:     profile.SourceActionLabel,
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
	damage := baseBattleDamage(actor, profile, defense)
	if runtime.resolveCriticalHit(actor, target, commandID, profile) {
		damage *= 2
		targetActionState = "fat"
		targetActionStateCode = "2"
	}
	target.HP = maxInt(0, target.HP-damage)
	delete(runtime.DefendingHandles, target.Handle)
	if profile.MPCost > 0 {
		actor.MP = maxInt(0, actor.MP-profile.MPCost)
	}
	if profile.LifeStealChance > 0 && profile.LifeStealRatio > 0 && damage > 0 && runtime.resolveLifeSteal(actor, target, commandID, profile) {
		actor.HP = clampInt(actor.HP+int(math.Floor(float64(damage)*profile.LifeStealRatio)), 0, actor.MaxHP)
	}
	refreshInfos := []CellInfoPush{*target}
	if profile.MPCost > 0 || profile.LifeStealChance > 0 {
		refreshInfos = []CellInfoPush{*actor, *target}
	}
	return ActionPush{
		BattleID:              runtime.BattleID,
		ActorHandle:           actor.Handle,
		TargetHandle:          target.Handle,
		CommandID:             commandID,
		ActionName:            profile.ActionName,
		SourceActionLabel:     profile.SourceActionLabel,
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
		SourceActionLabel: "nomalAtk",
		DamageMultiplier:  1,
		CanDodge:          true,
		CanFat:            true,
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
	case CommandEnemySlideCut:
		return commandProfile{
			ActionName:        "滑行斩",
			SourceActionLabel: "slideCut",
			DamageMultiplier:  1,
			MPCost:            enemySlideCutMPCost,
			CanDodge:          true,
			CanFat:            true,
		}
	case CommandNormalAttack, CommandEnemyAttack:
		profile.ActionName = "普通攻击"
	}
	return profile
}

func (runtime *Runtime) enemyBattleCommand(enemy *CellInfoPush, target *CellInfoPush) string {
	if sourceEnemyCanSlideCut(enemy) && enemy.MP >= enemySlideCutMPCost && runtime.resolveEnemySkillUse(enemy, target, CommandEnemySlideCut, enemySlideCutChance) {
		return CommandEnemySlideCut
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
	profile := commandProfile{
		ActionName:        name,
		SourceActionLabel: sourceBattleSkillActionLabel(name, level),
		DamageMultiplier:  sourceBattleSkillDamageMultiplier(skill.Description),
		MPCost:            sourceBattleSkillMPCost(skill.Description),
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
		profile.LifeStealChance = sourceBattleSkillLifeStealChance(skill.Description)
		profile.LifeStealRatio = sourceBattleSkillLifeStealRatio(skill.Description)
		if profile.LifeStealChance <= 0 {
			profile.LifeStealChance = fallbackShiXueLifeStealChance(level)
		}
		if profile.LifeStealRatio <= 0 {
			profile.LifeStealRatio = 0.7
		}
	}
	return profile
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
			SourceType:        defaultString(strings.TrimSpace(skill.Type), "oneE"),
			ActionName:        profile.ActionName,
			SourceActionLabel: profile.SourceActionLabel,
			Target:            sourceBattleCommandTarget(skill.Type),
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
	default:
		return ""
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

func effectiveBattleDefense(target *CellInfoPush, targetInDef bool) int {
	if target == nil {
		return 0
	}
	if targetInDef {
		return target.Defense * 2
	}
	return target.Defense
}

func baseBattleDamage(actor *CellInfoPush, profile commandProfile, defense int) int {
	if actor == nil {
		return 0
	}
	return maxInt(1, int(math.Round(float64(actor.Attack)*profile.DamageMultiplier))-defense)
}

func (runtime *Runtime) resolveCriticalHit(actor *CellInfoPush, target *CellInfoPush, commandID string, profile commandProfile) bool {
	if !profile.CanFat || actor == nil || target == nil || actor.Fat <= 0 {
		return false
	}
	if actor.Fat >= 100 {
		return true
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

func battleCriticalChancePercent(actorFat int, targetDog int) int {
	if actorFat <= 0 {
		return 0
	}
	if actorFat >= 100 {
		return 100
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

	reward, ok := sourceBattleRewardConfigForMap(mapID)
	if !ok || reward.Status != "confirmed" {
		return 0, []string{}
	}
	return reward.ExpDelta, append([]string{}, reward.Items...)
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

func sourceEnemyForMap(mapID string) (CellInfoPush, bool) {
	config, ok := sourceEnemyConfigForMap(mapID)
	if !ok {
		return CellInfoPush{}, false
	}
	return config.Cell, true
}

func (cell CellInfoPush) withBattleID(battleID string) CellInfoPush {
	cell.BattleID = battleID
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
