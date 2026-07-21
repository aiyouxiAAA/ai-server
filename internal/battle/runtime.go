package battle

import (
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
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

	CommandNormalAttack    = "skill-normal-attack"
	CommandMiZhan          = "skill-mi-zhan"
	CommandDuoDuanZhan     = "skill-duo-duan-zhan"
	CommandDuoDuanCi       = "skill-duo-duan-ci"
	CommandShiXueZhan      = "skill-shi-xue-zhan"
	CommandKuangBao        = "skill-kuang-bao"
	CommandHongYueZhan     = "skill-hong-yue-zhan"
	CommandXueQie          = "skill-xue-qie"
	CommandPiShanGunFa     = "skill-pi-shan-gun-fa"
	CommandYeChaGunFa      = "skill-ye-cha-gun-fa"
	CommandLiShiGunShu     = "skill-li-shi-gun-shu"
	CommandPanLongGunFa    = "skill-pan-long-gun-fa"
	CommandQiangLiFeiBiao  = "skill-qiang-li-fei-biao"
	CommandTouDu           = "skill-tou-du"
	CommandMoLiTuCi        = "skill-mo-li-tu-ci"
	CommandJiFengCi        = "skill-ji-feng-ci"
	CommandJieDuShu        = "skill-jie-du-shu"
	CommandQiangShe        = "skill-qiang-she"
	CommandGuanJiaLianShi  = "skill-guan-jia-lian-shi"
	CommandBingJianSuShe   = "skill-bing-jian-su-she"
	CommandMoLiSuShe       = "skill-mo-li-su-she"
	CommandAnYingJian      = "skill-an-ying-jian"
	CommandDuShi           = "skill-du-shi"
	CommandYanShouShu      = "skill-yan-shou-shu"
	CommandChiYanMoZhou    = "skill-chi-yan-mo-zhou"
	CommandLeiJi           = "skill-lei-ji"
	CommandLeiBaoZhou      = "skill-lei-bao-zhou"
	CommandLeiLongQiangXi  = "skill-lei-long-qiang-xi"
	CommandMoZhangShu      = "skill-mo-zhang-shu"
	CommandLeiHunZhan      = "skill-lei-hun-zhan"
	CommandAoYiHongLeiShi  = "skill-ao-yi-hong-lei-shi"
	CommandAoYiAnShaZhe    = "skill-ao-yi-an-sha-zhe"
	CommandAoYiLiuHeGunFa  = "skill-ao-yi-liu-he-gun-fa"
	CommandTaunt           = "skill-taunt"
	CommandJuanYeShi       = "skill-juan-ye-shi"
	CommandQiangGuanShi    = "skill-qiang-guan-shi"
	CommandNingShenShi     = "skill-ning-shen-shi"
	CommandKuangWuShi      = "skill-kuang-wu-shi"
	CommandQiYuShi         = "skill-qi-yu-shi"
	CommandAoYiPiaoXue     = "skill-ao-yi-piao-xue"
	CommandEnemyAttack     = "enemy-normal-attack"
	CommandEnemySlideCut   = "enemy-slide-cut"
	CommandEnemyShadeCut   = "enemy-shade-cut"
	CommandEnemyHelixAtk   = "enemy-helix-atk"
	CommandEnemyPalsyAtk   = "enemy-palsy-atk"
	CommandEnemyRampage    = "enemy-rampage-power"
	CommandEnemyFirePower  = "enemy-fire-power"
	CommandEnemyDeadLight  = "enemy-dead-light"
	CommandEnemyDoubleHit  = "enemy-double-hit"
	CommandEnemyRollAtk    = "enemy-roll-atk"
	CommandEnemyRockRain   = "enemy-rock-rain"
	CommandEnemyDarkMoon   = "enemy-dark-moon-cut"
	CommandEnemyEarthShock = "enemy-earth-shock-atk"
	CommandEnemyDelude     = "enemy-delude"
	CommandEnemyPieceAtk   = "enemy-piece-attack"
	CommandEnemyLionRoars  = "enemy-lion-roars"
	CommandEnemyGoldHit    = "enemy-gold-hit"
	CommandEnemyRoundAtk   = "enemy-round-atk"
	CommandEnemyRulingAx   = "enemy-ruling-ax"
	CommandEnemyVacuumKill = "enemy-vacuum-killed"
	CommandEnemyRobotUp    = "enemy-robot-up"
	CommandEnemyChaosHit     = "enemy-chaos-hit"
	CommandEnemyThunderstorm = "enemy-thunderstorm"
	CommandEnemyAngleCurse   = "enemy-angle-curse"
	CommandEnemySweepSpear   = "enemy-sweep-spear"
	CommandDefense         = "defense"
	CommandStore           = "battle-store"
	CommandEscape          = "battle-escape"
	CommandItem            = "battle-item"

	maxStoredPower                = 5
	leiHunZhanRequiredPower       = 3
	leiLongQiangXiRequiredPower   = 2
	aoYiHongLeiShiRequiredPower   = 2
	aoYiAnShaZheRequiredPower     = 3
	aoYiLiuHeGunFaRequiredPower   = 3
	aoYiPiaoXueRequiredPower      = 3
	enemySlideCutMPCost           = 10
	enemySlideCutChance           = 20
	enemyShadeCutMPCost           = 40
	enemyShadeCutChance           = 30
	enemyHelixAtkMPCost           = 10
	enemyHelixAtkChance           = 23
	enemyHelixAtkDamageMultiplier = 1.32
	enemyPalsyAtkChance           = 40
	enemyChaosHitChance           = 46
	// 20260720_225539_439: thunderstorm/anglecurse AI approx; MP cost PARTIAL.
	// Thunderstorm damage: same-target HP-delta vs 法术普通攻击 ≈ 2.0x
	// (恐龙抗狼1 649/320, 桥头的樵夫 662/336). Anglecurse stays ~1.0x.
	enemyThunderstormMPCost           = 10
	enemyThunderstormChance           = 25
	enemyThunderstormDamageMultiplier = 2
	enemyAngleCurseMPCost             = 10
	enemyAngleCurseChance             = 10
	enemySweepSpearMPCost         = 30
	enemySweepSpearChance         = 28
	enemyPalsyAtkStatusChance     = 100
	enemySweepSpearDamageMultiplier     = 1.30
	enemyStunOnHitChance                = 5
	enemyRampageMaxRounds               = 50
	enemyBainianRampageMaxRounds        = 51
	enemyFirePowerMPCost                = 10
	enemyFirePowerChance                = 60
	enemyFirePowerDamageMultiplier      = 0.835
	enemyDeadLightMPCost                = 10
	enemyDeadLightChance                = 35
	enemyDeadLightDamageMultiplier      = 1.142
	enemyDoubleHitMPCost                = 10
	enemyDoubleHitChance                = 45
	enemyDoubleHitDamageMultiplier      = 1.825
	enemyRollAtkMPCost                  = 10
	enemyRollAtkChance                  = 45
	enemyRollAtkDamageMultiplier        = 1.49
	enemyRockRainMPCost                 = 10
	enemyRockRainChance                 = 10
	enemyDarkMoonMPCost                 = 10
	enemyDarkMoonChance                 = 25
	enemyEarthShockMPCost               = 10
	enemyEarthShockChance               = 10
	enemyDeludeMPCost                   = 10
	enemyDeludeChance                   = 33
	enemyDeludeStatusRounds             = 2
	enemyShihukuBlackshadowPieceChance  = 7
	enemyShihukuChilukingPieceChance    = 11
	enemyShihukuBlackshadowWoundMin     = 6
	enemyShihukuBlackshadowWoundMax     = 9
	enemyShihukuChilukingWoundMin       = 7
	enemyShihukuChilukingWoundMax       = 10
	enemyShihukuPieceWoundRounds        = 5
	enemyShihukuSkillMPCost             = 10
	enemyShihukuPieceDamageMultiplier   = 1.4
	enemyShihukuLionDamageMultiplier    = 1.26
	enemyShihukuGoldDamageMultiplier    = 1.72
	enemyShihukuLionRoarsChance         = 20
	enemyShihukuGoldHitChance           = 19
	enemyRobotawlRoundAtkChance         = 23
	enemyRobotSkillMPCost               = 10
	enemyRobotawlArmorBreakChance       = 70
	enemyRobotawlArmorBreakRounds       = 3
	enemyRobotawlArmorBreakAttackPct    = 10
	// Capture-backed approximate action ratios; they are not original-server AI constants.
	enemyRobotaxRulingAxChance         = 20
	enemyRobotaxRulingAxSlownessChance = 30
	enemyRobotaxRulingAxSlownessRounds = 2
	enemyRobotaxRulingAxSlownessPct    = 30
	enemyRobothmarshalVacuumKillChance = 20
	// robotup is capture-backed local approximation: 516 non-zero heal samples
	// are 263..338 before max-HP clipping; every cast spends 60 MP.
	enemyRobotupMPCost                = 60
	enemyRobotupHealMin               = 263
	enemyRobotupHealMax               = 338
	enemyRobotupEmergencyHPPercent    = 20
	enemyRobotupLowHPChance           = 98
	enemyRobotupGeneralChance         = 30
	defaultBattleHit                  = 100
	defaultBattleDog                  = 50
	defaultBattleFat                  = 5
	equipmentInnerInjuryDefaultChance = 1
	equipmentInnerInjuryDefaultRounds = 3
	equipmentInnerInjuryDefaultMin    = 10
	equipmentInnerInjuryDefaultMax    = 15
	equipmentSealDefaultChance        = 1
	equipmentSealDefaultRounds        = 3
	jiFengCiSlownessChance            = 92
	jiFengCiSlownessRounds            = 3
	jiFengCiSlownessPercent           = 50
	touDuPoisonChance                 = 80
	touDuPoisonRounds                 = 4
	touDuPoisonDefensePercent         = 15
	touDuPoisonTickMin                = 20
	touDuPoisonTickMax                = 25
	liShiGunShuAttackPercent          = 15
	liShiGunShuRounds                 = 5
	tauntRounds                       = 3
	qiangGuanShiArmorBreakRounds      = 3
	qiangGuanShiArmorBreakPercent     = 50
	ningShenRounds                    = 4
	ningShenHitPercent                = 70
	qiYuRounds                        = 3
	qiYuHealPercent                   = 13
	kuangWuShiStunChance              = 25
	kuangWuShiStunRounds              = 2
	// Captured single-sword Lv5 baselines above; lower levels come from
	// skill-level-table / role skill description packets.
	chiYanMoZhouCurseChance           = 85
	chiYanMoZhouCurseRounds           = 2
	leiJiSlownessChance               = 60
	leiJiSlownessRounds               = 2
	leiJiSlownessPercent              = 30
	magicObstacleRounds               = 5
	magicObstacleDamageToMPPercent    = 35
)

var sourceEncounterRoll = func(maxExclusive int) int {
	if maxExclusive <= 0 {
		return 0
	}
	return rand.Intn(maxExclusive)
}

var sourceBattleAttackRoll = func(maxExclusive int) int {
	if maxExclusive <= 0 {
		return 0
	}
	return rand.Intn(maxExclusive)
}

var sourceBattleHealRoll = func(maxExclusive int) int {
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
	Spectator       bool                `json:"spectator,omitempty"`
}

type CellInfoPush struct {
	BattleID          string `json:"battleId,omitempty"`
	Camp              Camp   `json:"camp"`
	Handle            string `json:"handle"`
	Name              string `json:"name"`
	DisplayURL        string `json:"displayUrl"`
	Level             int    `json:"level,omitempty"`
	Vocation          string `json:"vocation,omitempty"`
	XScale            int    `json:"xScale"`
	YScale            int    `json:"yScale"`
	MaxHP             int    `json:"maxHp"`
	HP                int    `json:"hp"`
	MaxMP             int    `json:"maxMp,omitempty"`
	MP                int    `json:"mp,omitempty"`
	Speed             int    `json:"speed"`
	Attack            int    `json:"attack"`
	MagicAttack       int    `json:"mgcAtk,omitempty"`
	Defense           int    `json:"defense"`
	MgcDefense        int    `json:"mgcDef,omitempty"`
	Hit               int    `json:"hit,omitempty"`
	Dog               int    `json:"dog,omitempty"`
	Fat               int    `json:"fat,omitempty"`
	CommandLabel      string `json:"commandLabel"`
	DamageDefenseType string `json:"damageDefenseType,omitempty"`
}

type StartCommandPush struct {
	BattleID    string              `json:"battleId"`
	ActorHandle string              `json:"actorHandle"`
	Round       int                 `json:"round"`
	Sequence    int                 `json:"sequence"`
	Power       interface{}         `json:"power"`
	Commands    []CommandDefinition `json:"commands,omitempty"`
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

type ClearBuffInfoPush struct {
	BattleID     string `json:"battleId,omitempty"`
	TargetHandle string `json:"targetHandle"`
	Name         string `json:"name"`
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

type TeamActor struct {
	Role       session.RoleSummary
	PlayerBase session.PlayerBaseData
}

type Runtime struct {
	BattleID              string
	RoleID                string
	MapID                 string
	LastMapName           string
	EncounterID           string
	EncounterLabel        string
	StageFocusX           float64
	ReturnRoute           string
	QueueIndexTeam        int
	QueueIndexEnemy       int
	SourceMonsterHandle   string
	RoleSkills            []session.RoleSkill
	RoleSkillsByHandle    map[string][]session.RoleSkill
	RoleItems             []session.RoleItem
	RoleItemsByHandle     map[string][]session.RoleItem
	Phase                 string
	Round                 int
	Cells                 []CellInfoPush
	ActiveHandle          string
	ConsumedSequence      map[int]bool
	PendingTeamActions    map[string]bool
	PendingTeamSequences  map[string]int
	DefendingHandles      map[string]bool
	EnemyAttackRanges     map[string]battleAttackRange
	StatusEffects         map[string]BattleStatusEffects
	PendingConfusion      map[string]bool
	StoredPower           map[string]int
	PendingBuffInfos      []BuffInfoPush
	PendingClearBuffInfos []ClearBuffInfoPush
	PendingSkillSeal      map[string]bool
	PendingStarts         []StartCommandPush
	PendingOver           *OverPush
	nextSequence          int
	actionSequence        int
	mu                    sync.Mutex
}

type battleAttackRange struct {
	Min int
	Max int
}

type BattleStatusEffect struct {
	Name                     string
	Display                  string
	Description              string
	Rounds                   int
	SourceHandle             string
	SourceSkill              string
	AppliedAction            string
	SourceAttack             int
	DefenseReductionPercent  int
	HitDodgeReductionPercent int
	TickMinPercent           int
	TickMaxPercent           int
	AttackIncrease           int
	AttackReduction          int
	StatusAttackMin          int
	StatusAttackMax          int
	MagicAttackReduction     int
	DefenseReduction         int
	MagicDefenseReduction    int
	HitReduction             int
	DodgeReduction           int
	HitIncrease              int
	FatIncrease              int
	HealPercent              int
	DamageToMPPercent        int
	VisualOnly               bool
	SkipTurn                 bool
}

type BattleStatusEffects struct {
	KuangBaoRounds int
	Effects        map[string]BattleStatusEffect
}

type StartBundle struct {
	Start             StartPush
	Cells             []CellInfoPush
	StartCommand      StartCommandPush
	TeamStartCommands []StartCommandPush
}

type ActionResult struct {
	Actions        []ActionPush
	BuffInfos      []BuffInfoPush
	ClearBuffInfos []ClearBuffInfoPush
	StartCommand   *StartCommandPush
	StartCommands  []StartCommandPush
	Over           *OverPush
	ErrorCode      string
}

type commandProfile struct {
	ActionName            string
	SourceType            string
	SourceActionLabel     string
	DamageMultiplier      float64
	MPCost                int
	CanDodge              bool
	CanFat                bool
	LifeStealChance       int
	LifeStealRatio        float64
	DirectAttackBonus     float64
	AdditionalMagicBonus  float64
	MagicAttackBoost      float64
	DefenseType           string
	UseMagicAttack        bool
	HitMultiplier         float64
	TargetMPDamage        int
	StatusName            string
	StatusDisplay         string
	StatusDescription     string
	StatusRounds          int
	StatusChance          int
	StatusDefensePercent  int
	StatusHitDodgePercent int
	StatusAttackMin       int
	StatusAttackMax       int
	StatusTickMin         int
	StatusTickMax         int
	StatusVisualOnly      bool
	SkipTurn              bool
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
	playerMagicAttack := 0
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
		playerMagicAttack = maxInt(0, rolePhysique.MgcAtk)
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
			DisplayURL:   battleRoleDisplayURL(role, playerBase),
			Level:        playerLevel,
			Vocation:     defaultString(playerBase.Voc, role.Voc),
			XScale:       100,
			YScale:       100,
			MaxHP:        playerMaxHP,
			HP:           playerHP,
			MaxMP:        playerMaxMP,
			MP:           playerMP,
			Speed:        playerSpeed,
			Attack:       playerAttack,
			MagicAttack:  playerMagicAttack,
			Defense:      playerDefense,
			MgcDefense:   playerMgcDefense,
			Hit:          playerHit,
			Dog:          playerDog,
			Fat:          playerFat,
			CommandLabel: "普通攻击",
		},
	}
	enemyAttackRanges := map[string]battleAttackRange{}
	for index, enemyConfig := range enemyConfigs {
		var cell CellInfoPush
		if sourceMonsterHandle != "" {
			cell = enemyConfig.Cell.withBattleID(battleID)
		} else {
			cell = enemyConfig.Cell.withBattleIDAndSlot(battleID, index)
		}
		cells = append(cells, cell)
		if enemyConfig.AttackMin > 0 && enemyConfig.AttackMax >= enemyConfig.AttackMin {
			enemyAttackRanges[cell.Handle] = battleAttackRange{Min: enemyConfig.AttackMin, Max: enemyConfig.AttackMax}
		}
	}
	runtime := &Runtime{
		BattleID:            battleID,
		RoleID:              role.RoleID,
		MapID:               mapID,
		LastMapName:         mapName,
		EncounterID:         "classic-wild-" + mapID,
		EncounterLabel:      mapName + " " + encounterKind,
		StageFocusX:         request.StageFocusX,
		ReturnRoute:         defaultString(request.ReturnRoute, "town-placeholder"),
		QueueIndexTeam:      firstEnemyConfig.QueueIndexTeam,
		QueueIndexEnemy:     firstEnemyConfig.QueueIndexEnemy,
		SourceMonsterHandle: sourceMonsterHandle,
		RoleSkills:          cloneBattleRoleSkills(role.Skills),
		RoleSkillsByHandle:  map[string][]session.RoleSkill{role.RoleID: cloneBattleRoleSkills(role.Skills)},
		RoleItems:           cloneBattleRoleItems(role.Items),
		RoleItemsByHandle:   map[string][]session.RoleItem{role.RoleID: cloneBattleRoleItems(role.Items)},
		Phase:               PhaseCommand,
		Round:               1,
		Cells:               cells,
		ActiveHandle:        role.RoleID,
		ConsumedSequence:    map[int]bool{},
		DefendingHandles:    map[string]bool{},
		EnemyAttackRanges:   enemyAttackRanges,
		StatusEffects:       map[string]BattleStatusEffects{},
		PendingConfusion:    map[string]bool{},
		PendingSkillSeal:    map[string]bool{},
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
	// Solo is concurrent N=1: same PendingTeamActions machine as multi-member teams.
	runtime.resetPendingTeamActionsExcluding(nil)
	teamStarts := runtime.pendingTeamStartCommands()
	startCommand := StartCommandPush{}
	if len(teamStarts) > 0 {
		startCommand = teamStarts[0]
	}
	return runtime, StartBundle{
		Start:             start,
		Cells:             cells,
		StartCommand:      startCommand,
		TeamStartCommands: teamStarts,
	}, true
}

// SpectatorSnapshot returns a read-only battle entry payload from the current
// authoritative cells. Spectators never receive a command window.
func (runtime *Runtime) SpectatorSnapshot() (StartPush, []CellInfoPush, bool) {
	if runtime == nil {
		return StartPush{}, nil, false
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	if runtime.Phase == PhaseFinished || runtime.BattleID == "" || len(runtime.Cells) == 0 {
		return StartPush{}, nil, false
	}
	return StartPush{
		BattleID:        runtime.BattleID,
		QueueIndexTeam:  runtime.QueueIndexTeam,
		QueueIndexEnemy: runtime.QueueIndexEnemy,
		LastMapName:     runtime.LastMapName,
		SelfHandle:      "",
		MapID:           runtime.MapID,
		EncounterID:     runtime.EncounterID,
		EncounterLabel:  runtime.EncounterLabel,
		StageFocusX:     runtime.StageFocusX,
		ReturnRoute:     runtime.ReturnRoute,
		Spectator:       true,
	}, append([]CellInfoPush(nil), runtime.Cells...), true
}

func NewTeamWildBattle(actors []TeamActor, request StartRequest) (*Runtime, StartBundle, bool) {
	if len(actors) == 0 {
		return nil, StartBundle{}, false
	}
	runtime, bundle, ok := NewWildBattle(actors[0].Role, actors[0].PlayerBase, request)
	if !ok {
		return nil, StartBundle{}, false
	}
	// NewWildBattle opens an N=1 window for standalone battles. A team is still
	// being constructed here, so discard that unpublished window before assigning
	// the real concurrent team windows from sequence 1.
	runtime.PendingTeamActions = nil
	runtime.PendingTeamSequences = nil
	runtime.nextSequence = 1
	if runtime.RoleSkillsByHandle == nil {
		runtime.RoleSkillsByHandle = map[string][]session.RoleSkill{}
	}
	if runtime.RoleItemsByHandle == nil {
		runtime.RoleItemsByHandle = map[string][]session.RoleItem{}
	}
	for index := 1; index < len(actors); index++ {
		actor := actors[index]
		roleID := strings.TrimSpace(actor.Role.RoleID)
		if roleID == "" || runtime.cellByHandle(roleID) != nil {
			continue
		}
		cell := buildTeamActorCell(runtime.BattleID, actor.Role, actor.PlayerBase)
		runtime.Cells = append(runtime.Cells, cell)
		runtime.RoleSkillsByHandle[roleID] = cloneBattleRoleSkills(actor.Role.Skills)
		runtime.RoleItemsByHandle[roleID] = cloneBattleRoleItems(actor.Role.Items)
	}
	runtime.resetPendingTeamActions()
	bundle.Cells = append([]CellInfoPush(nil), runtime.Cells...)
	bundle.TeamStartCommands = runtime.pendingTeamStartCommands()
	if start, ok := bundle.StartCommandForActor(actors[0].Role.RoleID); ok {
		bundle.StartCommand = start
	}
	return runtime, bundle, true
}

func (bundle StartBundle) StartCommandForActor(handle string) (StartCommandPush, bool) {
	handle = strings.TrimSpace(handle)
	for _, command := range bundle.TeamStartCommands {
		if command.ActorHandle == handle {
			return command, true
		}
	}
	if bundle.StartCommand.ActorHandle == handle {
		return bundle.StartCommand, true
	}
	return StartCommandPush{}, false
}

func buildTeamActorCell(battleID string, role session.RoleSummary, playerBase session.PlayerBaseData) CellInfoPush {
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
	playerMagicAttack := 0
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
		playerMagicAttack = maxInt(0, rolePhysique.MgcAtk)
		playerDefense = maxInt(0, rolePhysique.PhyDef)
		playerMgcDefense = maxInt(0, rolePhysique.MgcDef)
		playerHit = maxInt(0, rolePhysique.Hit)
		playerDog = maxInt(0, rolePhysique.Dog)
		playerFat = maxInt(0, rolePhysique.Fat)
	}
	return CellInfoPush{
		BattleID:     battleID,
		Camp:         CampTeam,
		Handle:       role.RoleID,
		Name:         defaultString(playerBase.DisplayName, role.DisplayName),
		DisplayURL:   battleRoleDisplayURL(role, playerBase),
		Level:        playerLevel,
		Vocation:     defaultString(playerBase.Voc, role.Voc),
		XScale:       100,
		YScale:       100,
		MaxHP:        playerMaxHP,
		HP:           playerHP,
		MaxMP:        playerMaxMP,
		MP:           playerMP,
		Speed:        playerSpeed,
		Attack:       playerAttack,
		MagicAttack:  playerMagicAttack,
		Defense:      playerDefense,
		MgcDefense:   playerMgcDefense,
		Hit:          playerHit,
		Dog:          playerDog,
		Fat:          playerFat,
		CommandLabel: "普通攻击",
	}
}

func battleRoleDisplayURL(role session.RoleSummary, playerBase session.PlayerBaseData) string {
	return defaultString(
		playerBase.BattleSourceQuery,
		defaultString(
			role.BattleSourceQuery,
			defaultString(playerBase.SourceQuery, defaultString(role.SourceQuery, "human/human.swf?w1=1&")),
		),
	)
}

func (runtime *Runtime) ProcessAction(request ActionRequest) ActionResult {
	if runtime == nil {
		return ActionResult{ErrorCode: "battle_missing"}
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
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
	runtime.PendingClearBuffInfos = nil
	commandID := strings.TrimSpace(request.CommandID)
	if runtime.consumePendingConfusion(actor.Handle) {
		runtime.consumePendingSkillSeal(actor.Handle)
		runtime.ConsumedSequence[request.Sequence] = true
		runtime.setStoredPower(actor.Handle, 0)
		action := runtime.resolveConfusionNormalAttackAction(actor)
		actions := []ActionPush{}
		if action != nil {
			actions = append(actions, *action)
		}
		return runtime.resolveEnemyTurnAndNextCommand(actor, actions)
	}
	normalizedCommandID := normalizeBattleCommandID(commandID)
	if runtime.hasPendingSkillSeal(actor.Handle) && isBattleSkillCommandBlockedBySeal(normalizedCommandID) {
		return ActionResult{ErrorCode: "sealed_skill"}
	}

	switch normalizedCommandID {
	case CommandEscape:
		if !runtime.isSelfTarget(actor, request.TargetHandle) {
			return ActionResult{ErrorCode: "invalid_target"}
		}
		runtime.consumePendingSkillSeal(actor.Handle)
		runtime.ConsumedSequence[request.Sequence] = true
		runtime.setStoredPower(actor.Handle, 0)
		runtime.Phase = PhasePlaying
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
		runtime.consumePendingSkillSeal(actor.Handle)
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
		runtime.consumePendingSkillSeal(actor.Handle)
		runtime.ConsumedSequence[request.Sequence] = true
		runtime.setStoredPower(actor.Handle, runtime.powerFor(actor.Handle)+1)
		return runtime.resolveEnemyTurnAndNextCommand(actor, []ActionPush{
			runtime.resolveSelfAction(actor, commandID, "蓄力", "def"),
		})
	case CommandKuangBao:
		if !runtime.isBattleCommandAllowedForActor(actor.Handle, commandID) {
			return ActionResult{ErrorCode: "unsupported_command"}
		}
		if !runtime.isSelfTarget(actor, request.TargetHandle) {
			return ActionResult{ErrorCode: "invalid_target"}
		}
		runtime.ConsumedSequence[request.Sequence] = true
		runtime.setStoredPower(actor.Handle, 0)
		profile := runtime.sourceSkillProfileForActor(actor.Handle, "狂爆", 1)
		if profile.MPCost > 0 {
			actor.MP = maxInt(0, actor.MP-profile.MPCost)
		}
		runtime.applyKuangBao(actor.Handle)
		result := runtime.resolveEnemyTurnAndNextCommand(actor, []ActionPush{
			runtime.resolveSelfAction(actor, commandID, "狂爆", "w8/kb"),
		})
		result.BuffInfos = append(result.BuffInfos, runtime.resolveKuangBaoBuffInfo(actor))
		return result
	case CommandJieDuShu:
		if !runtime.isBattleCommandAllowedForActor(actor.Handle, commandID) {
			return ActionResult{ErrorCode: "unsupported_command"}
		}
		if !runtime.isSelfTarget(actor, request.TargetHandle) {
			return ActionResult{ErrorCode: "invalid_target"}
		}
		runtime.ConsumedSequence[request.Sequence] = true
		runtime.setStoredPower(actor.Handle, 0)
		profile := runtime.sourceSkillProfileForActor(actor.Handle, "解毒术", 1)
		if profile.MPCost > 0 {
			actor.MP = maxInt(0, actor.MP-profile.MPCost)
		}
		runtime.clearStatusEffect(actor.Handle, "中毒")
		return runtime.resolveEnemyTurnAndNextCommand(actor, []ActionPush{
			runtime.resolveSelfAction(actor, commandID, "解毒术", "w3/releaseDrug"),
		})
	case CommandLiShiGunShu:
		if !runtime.isBattleCommandAllowedForActor(actor.Handle, commandID) {
			return ActionResult{ErrorCode: "unsupported_command"}
		}
		if !runtime.isSelfTarget(actor, request.TargetHandle) {
			return ActionResult{ErrorCode: "invalid_target"}
		}
		runtime.ConsumedSequence[request.Sequence] = true
		runtime.setStoredPower(actor.Handle, 0)
		profile := runtime.sourceSkillProfileForActor(actor.Handle, "力释棍术", 1)
		if profile.MPCost > 0 {
			actor.MP = maxInt(0, actor.MP-profile.MPCost)
		}
		buffInfo := runtime.applyFightingSpiritStatusEffect(actor)
		result := runtime.resolveEnemyTurnAndNextCommand(actor, []ActionPush{
			runtime.resolveSelfAction(actor, commandID, "力释棍术", "w11/releasePower"),
		})
		result.BuffInfos = append(result.BuffInfos, buffInfo)
		return result
	case CommandTaunt:
		if !runtime.isBattleCommandAllowedForActor(actor.Handle, commandID) {
			return ActionResult{ErrorCode: "unsupported_command"}
		}
		if !runtime.isSelfTarget(actor, request.TargetHandle) {
			return ActionResult{ErrorCode: "invalid_target"}
		}
		runtime.ConsumedSequence[request.Sequence] = true
		runtime.setStoredPower(actor.Handle, 0)
		profile := runtime.battleCommandProfile(actor, commandID)
		if profile.MPCost > 0 {
			actor.MP = maxInt(0, actor.MP-profile.MPCost)
		}
		buffInfo := runtime.applyTauntStatusEffect(actor)
		result := runtime.resolveEnemyTurnAndNextCommand(actor, []ActionPush{
			runtime.resolveCapturedTauntAction(actor, commandID, profile),
		})
		result.BuffInfos = append(result.BuffInfos, buffInfo)
		return result
	case CommandNingShenShi:
		if !runtime.isBattleCommandAllowedForActor(actor.Handle, commandID) {
			return ActionResult{ErrorCode: "unsupported_command"}
		}
		if !runtime.isSelfTarget(actor, request.TargetHandle) {
			return ActionResult{ErrorCode: "invalid_target"}
		}
		runtime.ConsumedSequence[request.Sequence] = true
		runtime.setStoredPower(actor.Handle, 0)
		profile := runtime.battleCommandProfile(actor, commandID)
		if profile.MPCost > 0 {
			actor.MP = maxInt(0, actor.MP-profile.MPCost)
		}
		buffInfos := runtime.applyNingShenStatusEffects(actor)
		result := runtime.resolveEnemyTurnAndNextCommand(actor, []ActionPush{
			runtime.resolveProfileSelfAction(actor, commandID, profile),
		})
		result.BuffInfos = append(result.BuffInfos, buffInfos...)
		return result
	case CommandQiYuShi:
		if !runtime.isBattleCommandAllowedForActor(actor.Handle, commandID) {
			return ActionResult{ErrorCode: "unsupported_command"}
		}
		if !runtime.isSelfTarget(actor, request.TargetHandle) {
			return ActionResult{ErrorCode: "invalid_target"}
		}
		runtime.ConsumedSequence[request.Sequence] = true
		runtime.setStoredPower(actor.Handle, 0)
		profile := runtime.battleCommandProfile(actor, commandID)
		if profile.MPCost > 0 {
			actor.MP = maxInt(0, actor.MP-profile.MPCost)
		}
		buffInfo := runtime.applyQiYuStatusEffect(actor)
		result := runtime.resolveEnemyTurnAndNextCommand(actor, []ActionPush{
			runtime.resolveProfileSelfAction(actor, commandID, profile),
		})
		result.BuffInfos = append(result.BuffInfos, buffInfo)
		return result
	case CommandMoZhangShu:
		if !runtime.isBattleCommandAllowedForActor(actor.Handle, commandID) {
			return ActionResult{ErrorCode: "unsupported_command"}
		}
		if !runtime.isSelfTarget(actor, request.TargetHandle) {
			return ActionResult{ErrorCode: "invalid_target"}
		}
		runtime.ConsumedSequence[request.Sequence] = true
		runtime.setStoredPower(actor.Handle, 0)
		profile := runtime.battleCommandProfile(actor, commandID)
		if profile.MPCost > 0 {
			actor.MP = maxInt(0, actor.MP-profile.MPCost)
		}
		buffInfo := runtime.applyMagicBarrierStatusEffect(actor)
		result := runtime.resolveEnemyTurnAndNextCommand(actor, []ActionPush{
			runtime.resolveProfileSelfAction(actor, commandID, profile),
		})
		result.BuffInfos = append(result.BuffInfos, buffInfo)
		return result
	}

	if !runtime.isBattleCommandAllowedForActor(actor.Handle, commandID) {
		return ActionResult{ErrorCode: "unsupported_command"}
	}

	if runtime.battleCommandProfile(actor, commandID).SourceType == "all" {
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
	if normalizeBattleCommandID(commandID) == CommandLeiLongQiangXi && runtime.powerFor(actor.Handle) < leiLongQiangXiRequiredPower {
		return ActionResult{ErrorCode: "insufficient_power"}
	}
	if normalizeBattleCommandID(commandID) == CommandAoYiHongLeiShi && runtime.powerFor(actor.Handle) < aoYiHongLeiShiRequiredPower {
		return ActionResult{ErrorCode: "insufficient_power"}
	}
	if normalizeBattleCommandID(commandID) == CommandAoYiAnShaZhe && runtime.powerFor(actor.Handle) < aoYiAnShaZheRequiredPower {
		return ActionResult{ErrorCode: "insufficient_power"}
	}
	if normalizeBattleCommandID(commandID) == CommandAoYiLiuHeGunFa && runtime.powerFor(actor.Handle) < aoYiLiuHeGunFaRequiredPower {
		return ActionResult{ErrorCode: "insufficient_power"}
	}
	if normalizeBattleCommandID(commandID) == CommandAoYiPiaoXue && runtime.powerFor(actor.Handle) < aoYiPiaoXueRequiredPower {
		return ActionResult{ErrorCode: "insufficient_power"}
	}

	target := runtime.cellByHandle(request.TargetHandle)
	if target == nil || target.Camp != CampEnemy || target.HP <= 0 {
		return ActionResult{ErrorCode: "invalid_target"}
	}

	runtime.ConsumedSequence[request.Sequence] = true
	runtime.consumePendingSkillSeal(actor.Handle)
	actions := []ActionPush{runtime.resolveAttack(actor, target, commandID)}
	runtime.setStoredPower(actor.Handle, 0)
	return runtime.resolveEnemyTurnAndNextCommand(actor, actions)
}

func (runtime *Runtime) ProcessItemAction(request ItemActionRequest, item ItemAction) ActionResult {
	if runtime == nil {
		return ActionResult{ErrorCode: "battle_missing"}
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	actor, validation := runtime.validateActorTurn(actionTurnRequest{
		BattleID:    request.BattleID,
		ActorHandle: request.ActorHandle,
		Round:       request.Round,
		Sequence:    request.Sequence,
	})
	if validation.ErrorCode != "" {
		return validation
	}

	if runtime.consumePendingConfusion(actor.Handle) {
		runtime.consumePendingSkillSeal(actor.Handle)
		runtime.ConsumedSequence[request.Sequence] = true
		runtime.setStoredPower(actor.Handle, 0)
		action := runtime.resolveConfusionNormalAttackAction(actor)
		actions := []ActionPush{}
		if action != nil {
			actions = append(actions, *action)
		}
		return runtime.resolveEnemyTurnAndNextCommand(actor, actions)
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
	runtime.consumePendingSkillSeal(actor.Handle)
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
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	if request.BattleID != runtime.BattleID {
		return ActionResult{ErrorCode: "battle_mismatch"}
	}
	if runtime.PendingOver != nil {
		over := runtime.PendingOver
		runtime.PendingOver = nil
		runtime.PendingStarts = nil
		return ActionResult{Over: over}
	}
	if len(runtime.PendingStarts) > 0 {
		starts := append([]StartCommandPush(nil), runtime.PendingStarts...)
		runtime.PendingStarts = nil
		runtime.Phase = PhaseCommand
		// Always deliver via StartCommands only. Writers must filter by recipient
		// and must not also push a mirrored StartCommand (that doubles N=1 windows).
		return ActionResult{StartCommands: starts}
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
	command, ok := runtime.commandWindowForActor(request.ActorHandle)
	if !ok {
		return nil, ActionResult{ErrorCode: "actor_not_active"}
	}
	if request.Round != command.Round || request.Sequence != command.Sequence {
		return nil, ActionResult{ErrorCode: "sequence_mismatch"}
	}
	if runtime.ConsumedSequence[request.Sequence] {
		return nil, ActionResult{ErrorCode: "sequence_consumed"}
	}
	actor := runtime.cellByHandle(request.ActorHandle)
	if actor == nil || actor.Camp != CampTeam || actor.HP <= 0 {
		return nil, ActionResult{ErrorCode: "invalid_actor"}
	}
	runtime.actionSequence = request.Sequence
	return actor, ActionResult{}
}

func (runtime *Runtime) resolveEnemyTurnAndNextCommand(actor *CellInfoPush, actions []ActionPush) ActionResult {
	teamCommandRound := runtime.PendingTeamActions != nil
	if teamCommandRound {
		delete(runtime.PendingTeamActions, actor.Handle)
		delete(runtime.PendingTeamSequences, actor.Handle)
		runtime.prunePendingTeamActions()
		if winner := runtime.resolveWinner(); winner != "" {
			runtime.Phase = PhaseFinished
			runtime.PendingStarts = nil
			runtime.PendingOver = runtime.buildOver(winner)
			return ActionResult{
				Actions:        actions,
				BuffInfos:      runtime.consumePendingBuffInfos(),
				ClearBuffInfos: runtime.consumePendingClearBuffInfos(),
			}
		}
		if len(runtime.PendingTeamActions) > 0 {
			runtime.Phase = PhaseCommand
			runtime.PendingStarts = nil
			runtime.PendingOver = nil
			return ActionResult{
				Actions:        actions,
				BuffInfos:      runtime.consumePendingBuffInfos(),
				ClearBuffInfos: runtime.consumePendingClearBuffInfos(),
			}
		}
		runtime.PendingTeamActions = nil
		runtime.PendingTeamSequences = nil
	}

	teamHPBeforeEnemyTurn := map[string]int{}
	for _, cell := range runtime.Cells {
		if cell.Camp == CampTeam {
			teamHPBeforeEnemyTurn[cell.Handle] = cell.HP
		}
	}
	for {
		if winner := runtime.resolveWinner(); winner != "" {
			runtime.Phase = PhaseFinished
			runtime.PendingOver = runtime.buildOver(winner)
			return ActionResult{
				Actions:        actions,
				BuffInfos:      runtime.consumePendingBuffInfos(),
				ClearBuffInfos: runtime.consumePendingClearBuffInfos(),
			}
		}

		for _, enemy := range runtime.livingCells(CampEnemy) {
			if runtime.firstLiving(CampTeam) == nil {
				break
			}
			statusActions, skipTurn := runtime.resolveStatusStartActions(enemy)
			actions = append(actions, statusActions...)
			if runtime.resolveWinner() != "" {
				break
			}
			if enemy.HP <= 0 || skipTurn {
				continue
			}
			if runtime.consumePendingConfusion(enemy.Handle) {
				if action := runtime.resolveConfusionNormalAttackAction(enemy); action != nil {
					actions = append(actions, *action)
				}
				runtime.setStoredPower(enemy.Handle, 0)
				if runtime.resolveWinner() != "" {
					break
				}
				continue
			}
			// Pick a living team target per enemy action so multi-player battles
			// do not always hit the first living team cell.
			team := runtime.resolveEnemyTeamTarget(enemy)
			if team == nil {
				break
			}
			actions = append(actions, runtime.resolveEnemyRampageActions(enemy)...)
			targetHandle := team.Handle
			commandID := runtime.enemyBattleCommand(enemy, team)
			if runtime.consumePendingSkillSeal(enemy.Handle) {
				commandID = CommandEnemyAttack
			}
			actions = append(actions, runtime.resolveEnemyCommandActions(enemy, team, commandID)...)
			runtime.setStoredPower(enemy.Handle, 0)
			if _, ok := teamHPBeforeEnemyTurn[targetHandle]; !ok {
				teamHPBeforeEnemyTurn[targetHandle] = team.HP
			}
			if runtime.resolveWinner() != "" {
				break
			}
		}
		if winner := runtime.resolveWinner(); winner != "" {
			runtime.Phase = PhaseFinished
			runtime.PendingOver = runtime.buildOver(winner)
			return ActionResult{
				Actions:        actions,
				BuffInfos:      runtime.consumePendingBuffInfos(),
				ClearBuffInfos: runtime.consumePendingClearBuffInfos(),
			}
		}

		nextActor := runtime.nextLivingTeamActorAfter(actor.Handle)
		if nextActor == nil {
			runtime.Phase = PhaseFinished
			runtime.PendingOver = runtime.buildOver(CampEnemy)
			return ActionResult{
				Actions:        actions,
				BuffInfos:      runtime.consumePendingBuffInfos(),
				ClearBuffInfos: runtime.consumePendingClearBuffInfos(),
			}
		}
		// Solo and team share one machine. After the enemy phase, settle every
		// living teammate, then reopen windows only for free actors. Solo is N=1.
		if !teamCommandRound {
			// Any battle that reached an enemy phase without concurrent windows
			// is lifted onto the same machine before reopening.
			runtime.resetPendingTeamActionsExcluding(nil)
			teamCommandRound = runtime.PendingTeamActions != nil
		}
		statusActions, _, excludeWindows := runtime.resolveConcurrentTeamRoundStart(nextActor, teamHPBeforeEnemyTurn)
		actions = append(actions, statusActions...)
		if winner := runtime.resolveWinner(); winner != "" {
			runtime.Phase = PhaseFinished
			runtime.PendingStarts = nil
			runtime.PendingOver = runtime.buildOver(winner)
			return ActionResult{
				Actions:        actions,
				BuffInfos:      runtime.consumePendingBuffInfos(),
				ClearBuffInfos: runtime.consumePendingClearBuffInfos(),
			}
		}

		runtime.Round += 1
		// Sequence is allocated only inside resetPendingTeamActionsExcluding
		// (one nextSequence per free-actor window). Do not pre-increment here —
		// that produced 1,3,5 windows and shifted combat roll seeds.
		runtime.Phase = PhasePlaying
		freeActors := runtime.livingTeamActorsExcluding(excludeWindows)
		if len(freeActors) == 0 {
			// Everyone stunned/confused-auto-resolved: continue without clicks.
			if living := runtime.firstLiving(CampTeam); living != nil {
				actor = living
			} else {
				actor = nextActor
			}
			continue
		}
		runtime.resetPendingTeamActionsExcluding(excludeWindows)
		if len(runtime.PendingTeamActions) == 0 {
			if living := runtime.firstLiving(CampTeam); living != nil {
				actor = living
			} else {
				actor = nextActor
			}
			continue
		}
		runtime.ActiveHandle = freeActors[0].Handle
		runtime.PendingStarts = runtime.pendingTeamStartCommands()
		runtime.PendingOver = nil
		return ActionResult{
			Actions:        actions,
			BuffInfos:      runtime.consumePendingBuffInfos(),
			ClearBuffInfos: runtime.consumePendingClearBuffInfos(),
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
	defense := runtime.effectiveBattleDefense(actor, target, targetInDef, profile.DefenseType)
	sourceActionLabel := profile.SourceActionLabel
	targetActionState := "normal"
	targetActionStateCode := "0"
	if runtime.resolveDodge(actor, target, commandID, profile) {
		if consumeMP && profile.MPCost > 0 {
			actor.MP = maxInt(0, actor.MP-profile.MPCost)
		}
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
			Sequence:              runtime.currentActionSequence(),
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
			Sequence:              runtime.currentActionSequence(),
		}
	}
	damage := runtime.baseBattleDamage(actor, profile, defense)
	if runtime.resolveCriticalHit(actor, target, commandID, profile) {
		damage *= 2
		targetActionState = "fat"
		targetActionStateCode = "2"
	}
	actualHPLoss, _ := runtime.applyTargetHPDamage(target, damage)
	delete(runtime.DefendingHandles, target.Handle)
	runtime.applyStoredPowerFromSingleHPLoss(target, actualHPLoss)
	if consumeMP && profile.MPCost > 0 {
		actor.MP = maxInt(0, actor.MP-profile.MPCost)
	}
	if profile.LifeStealChance > 0 && profile.LifeStealRatio > 0 && damage > 0 && runtime.resolveLifeSteal(actor, target, commandID, profile) {
		actor.HP = clampInt(actor.HP+int(math.Floor(float64(damage)*profile.LifeStealRatio)), 0, actor.MaxHP)
	}
	if target.HP > 0 && runtime.resolveStatusApply(actor, target, commandID, profile) {
		effect := BattleStatusEffect{
			Name:                     profile.StatusName,
			Display:                  profile.StatusDisplay,
			Description:              profile.StatusDescription,
			Rounds:                   profile.StatusRounds,
			SourceHandle:             actor.Handle,
			SourceSkill:              profile.ActionName,
			AppliedAction:            sourceActionLabel,
			SourceAttack:             sourceBattleStatusAttack(actor, profile),
			DefenseReductionPercent:  profile.StatusDefensePercent,
			HitDodgeReductionPercent: profile.StatusHitDodgePercent,
			StatusAttackMin:          profile.StatusAttackMin,
			StatusAttackMax:          profile.StatusAttackMax,
			TickMinPercent:           profile.StatusTickMin,
			TickMaxPercent:           profile.StatusTickMax,
			VisualOnly:               profile.StatusVisualOnly,
			SkipTurn:                 profile.SkipTurn,
		}
		if strings.TrimSpace(effect.Name) == "迟钝" {
			runtime.applySlownessStatusEffect(actor, target, effect)
		} else if strings.TrimSpace(effect.Name) == "中毒" {
			runtime.applyPoisonStatusEffect(actor, target, effect)
		} else if strings.TrimSpace(effect.Name) == "内伤" {
			runtime.applyInnerInjuryStatusEffect(actor, target, effect)
		} else if strings.TrimSpace(effect.Name) == "卸甲" {
			runtime.applyArmorBreakStatusEffect(actor, target, effect)
		} else if strings.TrimSpace(effect.Name) == "麻痹" {
			runtime.applyPalsyStatusEffect(actor, target, effect)
		} else {
			runtime.applyStatusEffect(target.Handle, effect)
			runtime.PendingBuffInfos = append(runtime.PendingBuffInfos, runtime.resolveStatusBuffInfo(actor, target, effect))
		}
	}
	runtime.applyCapturedStunOnHit(actor, target, commandID)
	if target.HP > 0 {
		runtime.applyEquipmentInnerInjuryOnHit(actor, target, commandID)
	}
	if target.HP > 0 {
		runtime.applyEquipmentSealOnHit(actor, target, commandID)
	}
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
		Sequence:              runtime.currentActionSequence(),
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
		return runtime.sourceSkillProfileForActor(actor.Handle, "密斩", 1)
	case CommandDuoDuanZhan:
		return runtime.sourceSkillProfileForActor(actor.Handle, "多段斩", 1)
	case CommandDuoDuanCi:
		return runtime.sourceSkillProfileForActor(actor.Handle, "多段刺", 5)
	case CommandShiXueZhan:
		return runtime.sourceSkillProfileForActor(actor.Handle, "嗜血斩", 1)
	case CommandKuangBao:
		return runtime.sourceSkillProfileForActor(actor.Handle, "狂爆", 1)
	case CommandHongYueZhan:
		return runtime.sourceSkillProfileForActor(actor.Handle, "红月斩", 1)
	case CommandXueQie:
		return runtime.sourceSkillProfileForActor(actor.Handle, "血切", 1)
	case CommandTaunt:
		return runtime.sourceSkillProfileForActor(actor.Handle, "挑衅", 1)
	case CommandJuanYeShi:
		return runtime.sourceSkillProfileForActor(actor.Handle, "卷叶式", 5)
	case CommandQiangGuanShi:
		return runtime.sourceSkillProfileForActor(actor.Handle, "强贯式", 5)
	case CommandNingShenShi:
		return runtime.sourceSkillProfileForActor(actor.Handle, "凝神式", 5)
	case CommandKuangWuShi:
		return runtime.sourceSkillProfileForActor(actor.Handle, "狂舞式", 5)
	case CommandQiYuShi:
		return runtime.sourceSkillProfileForActor(actor.Handle, "气愈式", 5)
	case CommandAoYiPiaoXue:
		return runtime.sourceSkillProfileForActor(actor.Handle, "奥义.飘血", 4)
	case CommandPiShanGunFa:
		return runtime.sourceSkillProfileForActor(actor.Handle, "劈山棍法", 5)
	case CommandYeChaGunFa:
		return runtime.sourceSkillProfileForActor(actor.Handle, "夜叉棍法", 1)
	case CommandLiShiGunShu:
		return runtime.sourceSkillProfileForActor(actor.Handle, "力释棍术", 1)
	case CommandPanLongGunFa:
		return runtime.sourceSkillProfileForActor(actor.Handle, "盘龙棍法", 1)
	case CommandQiangLiFeiBiao:
		return runtime.sourceSkillProfileForActor(actor.Handle, "强力飞镖", 2)
	case CommandTouDu:
		return runtime.sourceSkillProfileForActor(actor.Handle, "投毒", 1)
	case CommandMoLiTuCi:
		return runtime.sourceSkillProfileForActor(actor.Handle, "魔力突刺", 1)
	case CommandJiFengCi:
		return runtime.sourceSkillProfileForActor(actor.Handle, "疾风刺", 1)
	case CommandJieDuShu:
		return runtime.sourceSkillProfileForActor(actor.Handle, "解毒术", 1)
	case CommandQiangShe:
		return runtime.sourceSkillProfileForActor(actor.Handle, "强射", 5)
	case CommandGuanJiaLianShi:
		return runtime.sourceSkillProfileForActor(actor.Handle, "贯甲连矢", 2)
	case CommandBingJianSuShe:
		return runtime.sourceSkillProfileForActor(actor.Handle, "冰箭速射", 5)
	case CommandMoLiSuShe:
		return runtime.sourceSkillProfileForActor(actor.Handle, "魔力速射", 5)
	case CommandAnYingJian:
		return runtime.sourceSkillProfileForActor(actor.Handle, "暗影箭", 1)
	case CommandDuShi:
		return runtime.sourceSkillProfileForActor(actor.Handle, "毒矢", 1)
	case CommandYanShouShu:
		return runtime.sourceSkillProfileForActor(actor.Handle, "炎狩术", 5)
	case CommandChiYanMoZhou:
		return runtime.sourceSkillProfileForActor(actor.Handle, "赤焰魔咒", 2)
	case CommandLeiJi:
		return runtime.sourceSkillProfileForActor(actor.Handle, "雷击", 3)
	case CommandLeiBaoZhou:
		return runtime.sourceSkillProfileForActor(actor.Handle, "雷爆咒", 1)
	case CommandLeiLongQiangXi:
		return runtime.sourceSkillProfileForActor(actor.Handle, "雷龙强袭", 1)
	case CommandMoZhangShu:
		return runtime.sourceSkillProfileForActor(actor.Handle, "魔障术", 2)
	case CommandLeiHunZhan:
		return runtime.sourceSkillProfileForActor(actor.Handle, "奥义.雷魂斩", 1)
	case CommandAoYiHongLeiShi:
		return runtime.sourceSkillProfileForActor(actor.Handle, "奥义.轰雷矢", 1)
	case CommandAoYiAnShaZhe:
		return runtime.sourceSkillProfileForActor(actor.Handle, "奥义.暗杀者", 1)
	case CommandAoYiLiuHeGunFa:
		return runtime.sourceSkillProfileForActor(actor.Handle, "奥义.六合棍法", 1)
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
			StatusDisplay:     "17.png",
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
			MPCost:            enemyFirePowerMPCost,
			CanDodge:          true,
			CanFat:            true,
			DefenseType:       "direct",
		}
	case CommandEnemyDeadLight:
		return commandProfile{
			ActionName:        "死亡射线",
			SourceType:        "all",
			SourceActionLabel: "deadLight",
			DamageMultiplier:  enemyDeadLightDamageMultiplier,
			MPCost:            enemyDeadLightMPCost,
			CanDodge:          true,
			CanFat:            false,
			DefenseType:       "direct",
		}
	case CommandEnemyDoubleHit:
		return commandProfile{
			ActionName:        "双锤打",
			SourceType:        "oneE",
			SourceActionLabel: "doubleHit",
			DamageMultiplier:  enemyDoubleHitDamageMultiplier,
			MPCost:            enemyDoubleHitMPCost,
			CanDodge:          true,
			CanFat:            true,
		}
	case CommandEnemyRollAtk:
		return commandProfile{
			ActionName:        "滑行连击",
			SourceType:        "oneE",
			SourceActionLabel: "rollAttack",
			DamageMultiplier:  enemyRollAtkDamageMultiplier,
			MPCost:            enemyRollAtkMPCost,
			CanDodge:          true,
			CanFat:            true,
		}
	case CommandEnemyRockRain:
		return commandProfile{
			ActionName:        "落石",
			SourceType:        "all",
			SourceActionLabel: "rockRain",
			DamageMultiplier:  1,
			MPCost:            enemyRockRainMPCost,
			CanDodge:          true,
			CanFat:            true,
			DefenseType:       "magic",
		}
	case CommandEnemyDarkMoon:
		return commandProfile{
			ActionName:        "暗月斩",
			SourceType:        "oneE",
			SourceActionLabel: "darkMoonCut",
			DamageMultiplier:  1,
			MPCost:            enemyDarkMoonMPCost,
			CanDodge:          true,
			CanFat:            true,
		}
	case CommandEnemyEarthShock:
		return commandProfile{
			ActionName:        "裂震击",
			SourceType:        "all",
			SourceActionLabel: "earthShockAtk",
			DamageMultiplier:  1,
			MPCost:            enemyEarthShockMPCost,
			CanDodge:          true,
			CanFat:            true,
		}
	case CommandEnemyDelude:
		return commandProfile{
			ActionName:        "魅惑术",
			SourceType:        "oneE",
			SourceActionLabel: "delude",
			MPCost:            enemyDeludeMPCost,
		}
	case CommandEnemyPieceAtk:
		profile := commandProfile{
			ActionName:        "撕裂",
			SourceType:        "oneE",
			SourceActionLabel: "pieceAttack",
			DamageMultiplier:  enemyShihukuPieceDamageMultiplier,
			MPCost:            enemyShihukuSkillMPCost,
			CanDodge:          true,
			CanFat:            true,
		}
		if minPercent, maxPercent, ok := sourceEnemyShihukuPieceAttackWoundPercents(actor); ok {
			profile.StatusName = "外伤"
			profile.StatusDisplay = "25.png"
			profile.StatusRounds = enemyShihukuPieceWoundRounds
			profile.StatusChance = 100
			profile.StatusTickMin = minPercent
			profile.StatusTickMax = maxPercent
			profile.StatusDescription = fmt.Sprintf("每回合损失气力为角色物理攻击的%d%%~%d%%", minPercent, maxPercent)
		}
		return profile
	case CommandEnemyLionRoars:
		return commandProfile{
			ActionName:        "狮吼",
			SourceType:        "oneE",
			SourceActionLabel: "lionroars",
			DamageMultiplier:  enemyShihukuLionDamageMultiplier,
			MPCost:            enemyShihukuSkillMPCost,
			CanDodge:          true,
			CanFat:            true,
		}
	case CommandEnemyGoldHit:
		return commandProfile{
			ActionName:        "黄金穿刺",
			SourceType:        "all",
			SourceActionLabel: "goldhit",
			DamageMultiplier:  enemyShihukuGoldDamageMultiplier,
			MPCost:            enemyShihukuSkillMPCost,
			CanDodge:          true,
			CanFat:            true,
		}
	case CommandEnemyRoundAtk:
		return commandProfile{
			ActionName:        "轮转刺伤",
			SourceType:        "oneE",
			SourceActionLabel: "roundatk",
			DamageMultiplier:  1,
			MPCost:            enemyRobotSkillMPCost,
			CanDodge:          true,
			CanFat:            true,
			StatusName:        "卸甲",
			StatusDisplay:     "10.png",
			StatusRounds:      enemyRobotawlArmorBreakRounds,
			StatusChance:      enemyRobotawlArmorBreakChance,
			StatusDescription: "降低对象物理防御力",
		}
	case CommandEnemyRulingAx:
		return commandProfile{
			ActionName:            "裁决之斧",
			SourceType:            "all",
			SourceActionLabel:     "rulingax",
			DamageMultiplier:      1,
			MPCost:                enemyRobotSkillMPCost,
			CanDodge:              true,
			CanFat:                true,
			StatusName:            "迟钝",
			StatusDisplay:         "16.png",
			StatusDescription:     "降低对象命中和回避",
			StatusRounds:          enemyRobotaxRulingAxSlownessRounds,
			StatusChance:          enemyRobotaxRulingAxSlownessChance,
			StatusHitDodgePercent: enemyRobotaxRulingAxSlownessPct,
		}
	case CommandEnemyVacuumKill:
		return commandProfile{
			ActionName:        "真空猎杀",
			SourceType:        "all",
			SourceActionLabel: "vacuumkilled",
			DamageMultiplier:  1,
			MPCost:            enemyRobotSkillMPCost,
			CanDodge:          true,
			CanFat:            true,
		}
	case CommandEnemyRobotUp:
		return commandProfile{
			ActionName:        "机木修复",
			SourceType:        "oneO",
			SourceActionLabel: "robotup",
			MPCost:            enemyRobotupMPCost,
		}
	case CommandEnemyChaosHit:
		// Capture: 混沌击 keeps nomalAtk animation and magic normal-attack damage path; only broadcast name changes.
		profile.ActionName = "混沌击"
		profile.SourceType = "oneE"
		profile.SourceActionLabel = "nomalAtk"
		profile.DamageMultiplier = 1
		profile.CanDodge = true
		profile.CanFat = true
		if actor != nil && strings.TrimSpace(actor.DamageDefenseType) != "" {
			profile.DefenseType = strings.TrimSpace(actor.DamageDefenseType)
		} else {
			profile.DefenseType = "magic"
		}
	case CommandEnemyThunderstorm:
		return commandProfile{
			ActionName:        "雷鸣怒吼",
			SourceType:        "all",
			SourceActionLabel: "thunderstorm",
			DamageMultiplier:  enemyThunderstormDamageMultiplier,
			MPCost:            enemyThunderstormMPCost,
			CanDodge:          true,
			CanFat:            true,
			DefenseType:       "magic",
		}
	case CommandEnemyAngleCurse:
		return commandProfile{
			ActionName:        "角念",
			SourceType:        "oneE",
			SourceActionLabel: "anglecurse",
			DamageMultiplier:  1,
			MPCost:            enemyAngleCurseMPCost,
			CanDodge:          true,
			CanFat:            true,
			DefenseType:       "magic",
		}
	case CommandEnemySweepSpear:
		return commandProfile{
			ActionName:        "单枪横扫",
			SourceType:        "all",
			SourceActionLabel: "sweepspear",
			DamageMultiplier:  enemySweepSpearDamageMultiplier,
			MPCost:            enemySweepSpearMPCost,
			CanDodge:          true,
			CanFat:            true,
			DefenseType:       "physical",
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
	if sourceEnemyCanRobothyunRobotUp(enemy) && enemy.MP >= enemyRobotupMPCost {
		if repairTarget := runtime.lowestLivingRobothyun(enemy.Camp); repairTarget != nil && runtime.resolveRobotupUse(enemy, repairTarget) {
			return CommandEnemyRobotUp
		}
	}
	if sourceEnemyCanRobothmarshalVacuumKill(enemy) && enemy.MP >= enemyRobotSkillMPCost && runtime.resolveEnemySkillUse(enemy, target, CommandEnemyVacuumKill, enemyRobothmarshalVacuumKillChance) {
		return CommandEnemyVacuumKill
	}
	if sourceEnemyCanRobotaxRulingAx(enemy) && enemy.MP >= enemyRobotSkillMPCost && runtime.resolveEnemySkillUse(enemy, target, CommandEnemyRulingAx, enemyRobotaxRulingAxChance) {
		return CommandEnemyRulingAx
	}
	if sourceEnemyCanShihukuGoldHit(enemy) && enemy.MP >= enemyShihukuSkillMPCost && runtime.resolveEnemySkillUse(enemy, target, CommandEnemyGoldHit, enemyShihukuGoldHitChance) {
		return CommandEnemyGoldHit
	}
	if sourceEnemyCanRobotawlRoundAtk(enemy) && enemy.MP >= enemyRobotSkillMPCost && runtime.resolveEnemySkillUse(enemy, target, CommandEnemyRoundAtk, enemyRobotawlRoundAtkChance) {
		return CommandEnemyRoundAtk
	}
	if sourceEnemyCanShihukuLionRoars(enemy) && enemy.MP >= enemyShihukuSkillMPCost && runtime.resolveEnemySkillUse(enemy, target, CommandEnemyLionRoars, enemyShihukuLionRoarsChance) {
		return CommandEnemyLionRoars
	}
	if pieceChance := sourceEnemyShihukuPieceAttackChance(enemy); pieceChance > 0 && enemy.MP >= enemyShihukuSkillMPCost && runtime.resolveEnemySkillUse(enemy, target, CommandEnemyPieceAtk, pieceChance) {
		return CommandEnemyPieceAtk
	}
	if sourceEnemyCanFirePower(enemy) && enemy.MP >= enemyFirePowerMPCost && runtime.resolveEnemySkillUse(enemy, target, CommandEnemyFirePower, enemyFirePowerChance) {
		return CommandEnemyFirePower
	}
	if sourceEnemyCanRollAtk(enemy) && enemy.MP >= enemyRollAtkMPCost && runtime.resolveEnemySkillUse(enemy, target, CommandEnemyRollAtk, enemyRollAtkChance) {
		return CommandEnemyRollAtk
	}
	if sourceEnemyCanEarthShock(enemy) && enemy.MP >= enemyEarthShockMPCost && runtime.resolveEnemySkillUse(enemy, target, CommandEnemyEarthShock, enemyEarthShockChance) {
		return CommandEnemyEarthShock
	}
	if sourceEnemyCanDelude(enemy) && enemy.MP >= enemyDeludeMPCost && runtime.resolveEnemySkillUse(enemy, target, CommandEnemyDelude, enemyDeludeChance) {
		return CommandEnemyDelude
	}
	if sourceEnemyCanDeadLight(enemy) && enemy.MP >= enemyDeadLightMPCost && runtime.resolveEnemySkillUse(enemy, target, CommandEnemyDeadLight, enemyDeadLightChance) {
		return CommandEnemyDeadLight
	}
	if sourceEnemyCanDoubleHit(enemy) && enemy.MP >= enemyDoubleHitMPCost && runtime.resolveEnemySkillUse(enemy, target, CommandEnemyDoubleHit, enemyDoubleHitChance) {
		return CommandEnemyDoubleHit
	}
	if sourceEnemyCanRockRain(enemy) && enemy.MP >= enemyRockRainMPCost && runtime.resolveEnemySkillUse(enemy, target, CommandEnemyRockRain, enemyRockRainChance) {
		return CommandEnemyRockRain
	}
	if sourceEnemyCanDarkMoon(enemy) && enemy.MP >= enemyDarkMoonMPCost && runtime.resolveEnemySkillUse(enemy, target, CommandEnemyDarkMoon, enemyDarkMoonChance) {
		return CommandEnemyDarkMoon
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
	if sourceEnemyCanChaosHit(enemy) && runtime.resolveEnemySkillUse(enemy, target, CommandEnemyChaosHit, enemyChaosHitChance) {
		return CommandEnemyChaosHit
	}
	if sourceEnemyCanThunderstorm(enemy) && enemy.MP >= enemyThunderstormMPCost && runtime.resolveEnemySkillUse(enemy, target, CommandEnemyThunderstorm, enemyThunderstormChance) {
		return CommandEnemyThunderstorm
	}
	if sourceEnemyCanAngleCurse(enemy) && enemy.MP >= enemyAngleCurseMPCost && runtime.resolveEnemySkillUse(enemy, target, CommandEnemyAngleCurse, enemyAngleCurseChance) {
		return CommandEnemyAngleCurse
	}
	if sourceEnemyCanMilitiaSweepSpear(enemy) && enemy.MP >= enemySweepSpearMPCost && runtime.resolveEnemySkillUse(enemy, target, CommandEnemySweepSpear, enemySweepSpearChance) {
		return CommandEnemySweepSpear
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

func (runtime *Runtime) resolveRobotupUse(actor *CellInfoPush, target *CellInfoPush) bool {
	if runtime == nil || actor == nil || target == nil || target.HP <= 0 || target.MaxHP <= 0 || actor.MP < enemyRobotupMPCost {
		return false
	}
	missingHPPercent := (target.MaxHP - target.HP) * 100 / target.MaxHP
	if missingHPPercent <= 0 {
		return false
	}
	chance := enemyRobotupGeneralChance
	if target.HP*100 < target.MaxHP*enemyRobotupEmergencyHPPercent {
		chance = enemyRobotupLowHPChance
	}
	return runtime.hashBattleRollWithSalt(actor, target, CommandEnemyRobotUp, "enemy-robotup") < chance
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

func sourceEnemyCanChaosHit(enemy *CellInfoPush) bool {
	if enemy == nil {
		return false
	}
	normalizedDisplay := strings.ToLower(strings.TrimSpace(enemy.DisplayURL))
	return strings.TrimSpace(enemy.Name) == "百年虫精" || strings.Contains(normalizedDisplay, "monstermap/wocmon.swf")
}

func sourceEnemyCanThunderstorm(enemy *CellInfoPush) bool {
	if enemy == nil {
		return false
	}
	normalizedDisplay := strings.ToLower(strings.TrimSpace(enemy.DisplayURL))
	return strings.TrimSpace(enemy.Name) == "熊鹿" || strings.Contains(normalizedDisplay, "monstermap/beardeer.swf")
}

func sourceEnemyCanAngleCurse(enemy *CellInfoPush) bool {
	return sourceEnemyCanThunderstorm(enemy)
}

func sourceEnemyCanMilitiaSweepSpear(enemy *CellInfoPush) bool {
	if enemy == nil {
		return false
	}
	return strings.TrimSpace(enemy.Name) == "被控制的民兵" && strings.Contains(strings.ToLower(strings.TrimSpace(enemy.DisplayURL)), "monstermap/militia.swf")
}

func sourceEnemyCanRobotawlRoundAtk(enemy *CellInfoPush) bool {
	if enemy == nil {
		return false
	}
	normalizedDisplay := strings.ToLower(strings.TrimSpace(enemy.DisplayURL))
	return strings.TrimSpace(enemy.Name) == "机木锥兵" || strings.Contains(normalizedDisplay, "monstermap/robotawl.swf")
}

func sourceEnemyCanRobotaxRulingAx(enemy *CellInfoPush) bool {
	if enemy == nil {
		return false
	}
	normalizedDisplay := strings.ToLower(strings.TrimSpace(enemy.DisplayURL))
	return strings.TrimSpace(enemy.Name) == "机木斧兵" || strings.Contains(normalizedDisplay, "monstermap/robotax.swf")
}

func sourceEnemyCanRobothmarshalVacuumKill(enemy *CellInfoPush) bool {
	if enemy == nil {
		return false
	}
	normalizedDisplay := strings.ToLower(strings.TrimSpace(enemy.DisplayURL))
	return strings.TrimSpace(enemy.Name) == "机木妖帅" || strings.Contains(normalizedDisplay, "monstermap/robothmarshal.swf")
}

func sourceEnemyCanRobothyunRobotUp(enemy *CellInfoPush) bool {
	if enemy == nil {
		return false
	}
	normalizedDisplay := strings.ToLower(strings.TrimSpace(enemy.DisplayURL))
	return strings.TrimSpace(enemy.Name) == "机木玄师" || strings.Contains(normalizedDisplay, "monstermap/robothyun.swf")
}

func sourceEnemyCanRockRain(enemy *CellInfoPush) bool {
	if enemy == nil {
		return false
	}
	normalizedDisplay := strings.ToLower(strings.TrimSpace(enemy.DisplayURL))
	return strings.TrimSpace(enemy.Name) == "咒巫师" || strings.Contains(normalizedDisplay, "monstermap/incantationshaman.swf")
}

func sourceEnemyCanDarkMoon(enemy *CellInfoPush) bool {
	if enemy == nil {
		return false
	}
	normalizedDisplay := strings.ToLower(strings.TrimSpace(enemy.DisplayURL))
	return strings.TrimSpace(enemy.Name) == "黄风二寨主" || strings.Contains(normalizedDisplay, "monstermap/hfscastellan.swf")
}

func sourceEnemyCanEarthShock(enemy *CellInfoPush) bool {
	if enemy == nil {
		return false
	}
	normalizedDisplay := strings.ToLower(strings.TrimSpace(enemy.DisplayURL))
	return strings.TrimSpace(enemy.Name) == "黄风大寨主" || strings.Contains(normalizedDisplay, "monstermap/hfcastellan.swf")
}

func sourceEnemyCanDelude(enemy *CellInfoPush) bool {
	if enemy == nil {
		return false
	}
	normalizedDisplay := strings.ToLower(strings.TrimSpace(enemy.DisplayURL))
	return strings.TrimSpace(enemy.Name) == "黄风寨夫人" || strings.Contains(normalizedDisplay, "monstermap/hflady.swf")
}

func sourceEnemyShihukuPieceAttackChance(enemy *CellInfoPush) int {
	if enemy == nil {
		return 0
	}
	normalizedName := strings.TrimSpace(enemy.Name)
	normalizedDisplay := strings.ToLower(strings.TrimSpace(enemy.DisplayURL))
	if normalizedName == "蚩颅王" || strings.Contains(normalizedDisplay, "monstermap/chiluking.swf") {
		return enemyShihukuChilukingPieceChance
	}
	if normalizedName == "黑影" || normalizedName == "黑影队长" || strings.Contains(normalizedDisplay, "monstermap/blackshadow.swf") {
		return enemyShihukuBlackshadowPieceChance
	}
	return 0
}

func sourceEnemyShihukuPieceAttackWoundPercents(enemy *CellInfoPush) (int, int, bool) {
	if enemy == nil {
		return 0, 0, false
	}
	normalizedName := strings.TrimSpace(enemy.Name)
	normalizedDisplay := strings.ToLower(strings.TrimSpace(enemy.DisplayURL))
	if normalizedName == "蚩颅王" || strings.Contains(normalizedDisplay, "monstermap/chiluking.swf") {
		return enemyShihukuChilukingWoundMin, enemyShihukuChilukingWoundMax, true
	}
	if normalizedName == "黑影" || normalizedName == "黑影队长" || strings.Contains(normalizedDisplay, "monstermap/blackshadow.swf") {
		return enemyShihukuBlackshadowWoundMin, enemyShihukuBlackshadowWoundMax, true
	}
	return 0, 0, false
}

func sourceEnemyCanShihukuLionRoars(enemy *CellInfoPush) bool {
	if enemy == nil {
		return false
	}
	normalizedName := strings.TrimSpace(enemy.Name)
	normalizedDisplay := strings.ToLower(strings.TrimSpace(enemy.DisplayURL))
	return normalizedName == "盘狮怪" || normalizedName == "盘狮队长" || strings.Contains(normalizedDisplay, "monstermap/whorllion.swf")
}

func sourceEnemyCanShihukuGoldHit(enemy *CellInfoPush) bool {
	if enemy == nil {
		return false
	}
	normalizedDisplay := strings.ToLower(strings.TrimSpace(enemy.DisplayURL))
	return strings.TrimSpace(enemy.Name) == "蚩颅王" || strings.Contains(normalizedDisplay, "monstermap/chiluking.swf")
}

func (runtime *Runtime) resolveEnemyCommandActions(enemy *CellInfoPush, target *CellInfoPush, commandID string) []ActionPush {
	if runtime == nil || enemy == nil || target == nil {
		return nil
	}
	if normalizeBattleCommandID(commandID) == CommandEnemyRobotUp {
		repairTarget := runtime.lowestLivingRobothyun(enemy.Camp)
		action := runtime.resolveEnemyRobotupAction(enemy, repairTarget)
		if strings.TrimSpace(action.ActionName) == "" {
			return nil
		}
		return []ActionPush{action}
	}
	if normalizeBattleCommandID(commandID) == CommandEnemyDelude {
		action := runtime.resolveEnemyDeludeAction(enemy, target)
		if strings.TrimSpace(action.ActionName) == "" {
			return nil
		}
		return []ActionPush{action}
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

func (runtime *Runtime) lowestLivingRobothyun(camp Camp) *CellInfoPush {
	if runtime == nil {
		return nil
	}
	var result *CellInfoPush
	for _, cell := range runtime.livingCells(camp) {
		if !sourceEnemyCanRobothyunRobotUp(cell) {
			continue
		}
		if result == nil || cell.HP*result.MaxHP < result.HP*cell.MaxHP || (cell.HP*result.MaxHP == result.HP*cell.MaxHP && cell.Handle < result.Handle) {
			result = cell
		}
	}
	return result
}

func (runtime *Runtime) resolveEnemyRobotupAction(actor *CellInfoPush, target *CellInfoPush) ActionPush {
	if runtime == nil || actor == nil || target == nil || !sourceEnemyCanRobothyunRobotUp(actor) || target.HP <= 0 || target.MaxHP <= 0 || actor.MP < enemyRobotupMPCost {
		return ActionPush{}
	}
	heal := enemyRobotupHealMin + sourceBattleHealRoll(enemyRobotupHealMax-enemyRobotupHealMin+1)
	target.HP = clampInt(target.HP+heal, 0, target.MaxHP)
	actor.MP = maxInt(0, actor.MP-enemyRobotupMPCost)
	refreshInfos := []CellInfoPush{*actor}
	if target.Handle != actor.Handle {
		refreshInfos = []CellInfoPush{*target, *actor}
	}
	profile := runtime.battleCommandProfile(actor, CommandEnemyRobotUp)
	return ActionPush{
		BattleID:          runtime.BattleID,
		ActorHandle:       actor.Handle,
		TargetHandle:      target.Handle,
		CommandID:         CommandEnemyRobotUp,
		ActionName:        profile.ActionName,
		SourceMode:        "0",
		SourceActionLabel: profile.SourceActionLabel,
		Damage:            0,
		TargetHP:          target.HP,
		TargetMP:          target.MP,
		TargetDead:        false,
		RefreshInfos:      refreshInfos,
		Round:             runtime.Round,
		Sequence:          runtime.currentActionSequence(),
	}
}

func (runtime *Runtime) resolveEnemyDeludeAction(enemy *CellInfoPush, target *CellInfoPush) ActionPush {
	if runtime == nil || enemy == nil || target == nil || target.HP <= 0 {
		return ActionPush{}
	}
	profile := runtime.battleCommandProfile(enemy, CommandEnemyDelude)
	if profile.MPCost > 0 {
		enemy.MP = maxInt(0, enemy.MP-profile.MPCost)
	}
	effect := BattleStatusEffect{
		Name:          "混乱",
		Display:       "20.png",
		Description:   "这个状态让人失去理智&0;胡乱攻击甚至自己人。",
		Rounds:        enemyDeludeStatusRounds,
		SourceHandle:  enemy.Handle,
		SourceSkill:   profile.ActionName,
		AppliedAction: profile.SourceActionLabel,
	}
	runtime.applyStatusEffect(target.Handle, effect)
	runtime.PendingBuffInfos = append(runtime.PendingBuffInfos, runtime.resolveStatusBuffInfo(enemy, target, effect))
	return ActionPush{
		BattleID:              runtime.BattleID,
		ActorHandle:           enemy.Handle,
		TargetHandle:          target.Handle,
		CommandID:             CommandEnemyDelude,
		ActionName:            profile.ActionName,
		SourceMode:            sourceBattleActionMode(profile.SourceType),
		SourceActionLabel:     profile.SourceActionLabel,
		TargetActionState:     "none",
		TargetActionStateCode: "3",
		Damage:                0,
		TargetHP:              target.HP,
		TargetMP:              target.MP,
		TargetDead:            target.HP <= 0,
		RefreshInfos:          []CellInfoPush{*enemy, *target},
		Round:                 runtime.Round,
		Sequence:              runtime.currentActionSequence(),
	}
}

func (runtime *Runtime) resolveEnemyRampageActions(enemy *CellInfoPush) []ActionPush {
	if runtime == nil || enemy == nil || enemy.HP <= 0 || !sourceEnemyCanRampage(enemy) {
		return nil
	}
	elapsed := maxInt(0, runtime.Round-1)
	remaining := maxInt(1, sourceEnemyRampageMaxRounds(enemy)-elapsed)
	runtime.PendingBuffInfos = append(runtime.PendingBuffInfos, BuffInfoPush{
		BattleID:      runtime.BattleID,
		ReleaseHandle: enemy.Handle,
		TargetHandle:  enemy.Handle,
		Name:          "暴走之力",
		Display:       "1595.png",
		Description:   fmt.Sprintf("命中：+0<br/><font color='#FF00FF'>还有 %d 回合暴走!</font>", remaining),
		Round:         remaining,
	})
	return []ActionPush{runtime.resolveSelfAction(enemy, CommandEnemyRampage, "暴走之力", "battleStand")}
}

func sourceEnemyCanRampage(enemy *CellInfoPush) bool {
	if enemy == nil {
		return false
	}
	normalizedDisplay := strings.ToLower(strings.TrimSpace(enemy.DisplayURL))
	switch strings.TrimSpace(enemy.Name) {
	case "巨岩魔", "岩化魔人", "黄风二寨主", "黄风大寨主", "黄风寨夫人", "蚩颅王", "点券盗贼", "百年虫精", "熊鹿", "龙娃":
		return true
	default:
		return strings.Contains(normalizedDisplay, "monstermap/largerock.swf") ||
			strings.Contains(normalizedDisplay, "monstermap/magicrockman.swf") ||
			strings.Contains(normalizedDisplay, "monstermap/hfscastellan.swf") ||
			strings.Contains(normalizedDisplay, "monstermap/hfcastellan.swf") ||
			strings.Contains(normalizedDisplay, "monstermap/hflady.swf") ||
			strings.Contains(normalizedDisplay, "monstermap/chiluking.swf") ||
			strings.Contains(normalizedDisplay, "monstermap/wocmon.swf") ||
			strings.Contains(normalizedDisplay, "monstermap/beardeer.swf") ||
			strings.Contains(normalizedDisplay, "monstermap/dragonson.swf")
	}
}

func sourceEnemyRampageMaxRounds(enemy *CellInfoPush) int {
	if enemy == nil {
		return enemyRampageMaxRounds
	}
	name := strings.TrimSpace(enemy.Name)
	display := strings.ToLower(strings.TrimSpace(enemy.DisplayURL))
	if name == "百年虫精" || strings.Contains(display, "monstermap/wocmon.swf") ||
		name == "熊鹿" || strings.Contains(display, "monstermap/beardeer.swf") {
		return enemyBainianRampageMaxRounds
	}
	return enemyRampageMaxRounds
}

func sourceEnemyCanFirePower(enemy *CellInfoPush) bool {
	if enemy == nil {
		return false
	}
	normalizedDisplay := strings.ToLower(strings.TrimSpace(enemy.DisplayURL))
	return strings.TrimSpace(enemy.Name) == "巨岩魔" || strings.Contains(normalizedDisplay, "monstermap/largerock.swf")
}

func sourceEnemyCanRollAtk(enemy *CellInfoPush) bool {
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

func sourceEnemyCanStunOnHit(enemy *CellInfoPush) bool {
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
	return runtime.isBattleCommandAllowedForActor("", commandID)
}

func (runtime *Runtime) isBattleCommandAllowedForActor(handle string, commandID string) bool {
	switch normalizeBattleCommandID(commandID) {
	case CommandNormalAttack:
		return true
	case CommandMiZhan:
		if len(runtime.skillsForHandle(handle)) == 0 {
			return true
		}
		return runtime.hasRoleSkillForActor(handle, "密斩")
	case CommandDuoDuanZhan:
		return runtime.hasRoleSkillForActor(handle, "多段斩")
	case CommandDuoDuanCi:
		return runtime.hasRoleSkillForActor(handle, "多段刺")
	case CommandShiXueZhan:
		return runtime.hasRoleSkillForActor(handle, "嗜血斩")
	case CommandKuangBao:
		return runtime.hasRoleSkillForActor(handle, "狂爆")
	case CommandHongYueZhan:
		return runtime.hasRoleSkillForActor(handle, "红月斩")
	case CommandXueQie:
		return runtime.hasRoleSkillForActor(handle, "血切")
	case CommandTaunt:
		return runtime.hasCapturedRoleSkillForActor(handle, "挑衅")
	case CommandJuanYeShi:
		return runtime.hasCapturedRoleSkillForActor(handle, "卷叶式")
	case CommandQiangGuanShi:
		return runtime.hasCapturedRoleSkillForActor(handle, "强贯式")
	case CommandNingShenShi:
		return runtime.hasCapturedRoleSkillForActor(handle, "凝神式")
	case CommandKuangWuShi:
		return runtime.hasCapturedRoleSkillForActor(handle, "狂舞式")
	case CommandQiYuShi:
		return runtime.hasCapturedRoleSkillForActor(handle, "气愈式")
	case CommandAoYiPiaoXue:
		return runtime.hasCapturedRoleSkillForActor(handle, "奥义.飘血")
	case CommandPiShanGunFa:
		return runtime.hasRoleSkillForActor(handle, "劈山棍法")
	case CommandYeChaGunFa:
		return runtime.hasRoleSkillForActor(handle, "夜叉棍法")
	case CommandLiShiGunShu:
		return runtime.hasRoleSkillForActor(handle, "力释棍术")
	case CommandPanLongGunFa:
		return runtime.hasRoleSkillForActor(handle, "盘龙棍法")
	case CommandQiangLiFeiBiao:
		return runtime.hasRoleSkillForActor(handle, "强力飞镖")
	case CommandTouDu:
		return runtime.hasRoleSkillForActor(handle, "投毒")
	case CommandMoLiTuCi:
		return runtime.hasRoleSkillForActor(handle, "魔力突刺")
	case CommandJiFengCi:
		return runtime.hasRoleSkillForActor(handle, "疾风刺")
	case CommandJieDuShu:
		return runtime.hasRoleSkillForActor(handle, "解毒术")
	case CommandQiangShe:
		return runtime.hasRoleSkillForActor(handle, "强射")
	case CommandGuanJiaLianShi:
		return runtime.hasRoleSkillForActor(handle, "贯甲连矢")
	case CommandBingJianSuShe:
		return runtime.hasRoleSkillForActor(handle, "冰箭速射")
	case CommandMoLiSuShe:
		return runtime.hasRoleSkillForActor(handle, "魔力速射")
	case CommandAnYingJian:
		return runtime.hasRoleSkillForActor(handle, "暗影箭")
	case CommandDuShi:
		return runtime.hasRoleSkillForActor(handle, "毒矢")
	case CommandYanShouShu:
		return runtime.hasCapturedRoleSkillForActor(handle, "炎狩术")
	case CommandChiYanMoZhou:
		return runtime.hasCapturedRoleSkillForActor(handle, "赤焰魔咒")
	case CommandLeiJi:
		return runtime.hasCapturedRoleSkillForActor(handle, "雷击")
	case CommandLeiBaoZhou:
		return runtime.hasCapturedRoleSkillForActor(handle, "雷爆咒")
	case CommandLeiLongQiangXi:
		return runtime.hasCapturedRoleSkillForActor(handle, "雷龙强袭")
	case CommandMoZhangShu:
		return runtime.hasCapturedRoleSkillForActor(handle, "魔障术")
	case CommandLeiHunZhan:
		return runtime.hasRoleSkillForActor(handle, "奥义.雷魂斩")
	case CommandAoYiHongLeiShi:
		return runtime.hasRoleSkillForActor(handle, "奥义.轰雷矢")
	case CommandAoYiAnShaZhe:
		return runtime.hasRoleSkillForActor(handle, "奥义.暗杀者")
	case CommandAoYiLiuHeGunFa:
		return runtime.hasRoleSkillForActor(handle, "奥义.六合棍法")
	default:
		return false
	}
}

func (runtime *Runtime) sourceSkillProfile(name string, fallbackLevel int) commandProfile {
	return runtime.sourceSkillProfileForActor("", name, fallbackLevel)
}

func (runtime *Runtime) sourceSkillProfileForActor(handle string, name string, fallbackLevel int) commandProfile {
	skill, ok := runtime.roleSkillByNameForActor(handle, name)
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
	return runtime.hasRoleSkillForActor("", name)
}

func (runtime *Runtime) hasRoleSkillForActor(handle string, name string) bool {
	_, ok := runtime.roleSkillByNameForActor(handle, name)
	return ok
}

func (runtime *Runtime) hasCapturedRoleSkillForActor(handle string, name string) bool {
	skill, ok := runtime.roleSkillByNameForActor(handle, name)
	return ok && isSourceBattleSkillLevelCaptured(name, skill.Level)
}

func isSourceBattleSkillLevelCaptured(name string, level int) bool {
	switch strings.TrimSpace(name) {
	case "炎狩术", "赤焰魔咒", "雷击", "雷爆咒", "雷龙强袭", "魔障术":
		return sourceBattleSkillLevelExists(name, level)
	default:
		return true
	}
}

func (runtime *Runtime) roleSkillByName(name string) (session.RoleSkill, bool) {
	return runtime.roleSkillByNameForActor("", name)
}

func (runtime *Runtime) roleSkillByNameForActor(handle string, name string) (session.RoleSkill, bool) {
	if runtime == nil {
		return session.RoleSkill{}, false
	}
	normalizedName := strings.TrimSpace(name)
	for _, skill := range runtime.skillsForHandle(handle) {
		if strings.TrimSpace(skill.Name) == normalizedName {
			return skill, true
		}
	}
	return session.RoleSkill{}, false
}

func (runtime *Runtime) skillsForHandle(handle string) []session.RoleSkill {
	if runtime == nil {
		return nil
	}
	handle = strings.TrimSpace(handle)
	if handle != "" && runtime.RoleSkillsByHandle != nil {
		if skills, ok := runtime.RoleSkillsByHandle[handle]; ok {
			return skills
		}
	}
	return runtime.RoleSkills
}

func (runtime *Runtime) commandDefinitionsForActor(handle string) []CommandDefinition {
	return sourceBattleCommandDefinitions(runtime.skillsForHandle(handle))
}

func (runtime *Runtime) itemsForHandle(handle string) []session.RoleItem {
	if runtime == nil {
		return nil
	}
	handle = strings.TrimSpace(handle)
	if handle != "" && runtime.RoleItemsByHandle != nil {
		if items, ok := runtime.RoleItemsByHandle[handle]; ok {
			return items
		}
	}
	return runtime.RoleItems
}

func sourceBattleSkillProfile(skill session.RoleSkill) commandProfile {
	name := strings.TrimSpace(skill.Name)
	level := skill.Level
	if level <= 0 {
		level = 1
	}
	description := sourceBattleSkillProfileDescription(name, level, skill.Description)
	tableProfile, hasTableProfile := sourceBattleSkillProfileFromConfig(name, level)
	sourceType := sourceBattleSkillSourceType(name, skill.Type)
	if hasTableProfile && strings.TrimSpace(tableProfile.SourceType) != "" {
		sourceType = strings.TrimSpace(tableProfile.SourceType)
	}
	actionName := name
	if hasTableProfile && strings.TrimSpace(tableProfile.ActionName) != "" {
		actionName = strings.TrimSpace(tableProfile.ActionName)
	}
	actionLabel := ""
	if hasTableProfile {
		actionLabel = strings.TrimSpace(tableProfile.SourceActionLabel)
	}
	if actionLabel == "" {
		actionLabel = sourceBattleSkillActionLabel(name, level)
	}
	profile := commandProfile{
		ActionName:        actionName,
		SourceType:        sourceType,
		SourceActionLabel: actionLabel,
		DamageMultiplier:  sourceBattleSkillDamageMultiplier(description),
		MPCost:            sourceBattleSkillMPCost(description),
		CanDodge:          true,
		CanFat:            true,
	}
	if hasTableProfile {
		profile.DirectAttackBonus = tableProfile.DirectAttackBonus
	}
	if directAttackBonus := sourceBattleSkillDirectAttackBonus(description); directAttackBonus > 0 && profile.DirectAttackBonus <= 0 {
		profile.DirectAttackBonus = directAttackBonus
	}
	if hasTableProfile && tableProfile.DamageMultiplier > 0 {
		profile.DamageMultiplier = tableProfile.DamageMultiplier
	}
	if profile.DamageMultiplier <= 0 && sourceType != "own" && name != "挑衅" {
		profile.DamageMultiplier = fallbackSourceBattleSkillMultiplier(name, level)
	}
	if hasTableProfile && tableProfile.MPCost > 0 {
		profile.MPCost = tableProfile.MPCost
	}
	if profile.MPCost <= 0 {
		profile.MPCost = fallbackSourceBattleSkillMPCost(name, level)
	}
	if name == "嗜血斩" {
		profile.LifeStealChance = sourceBattleSkillLifeStealChance(description)
		profile.LifeStealRatio = sourceBattleSkillLifeStealRatio(description)
		if hasTableProfile && tableProfile.LifeStealChance > 0 {
			profile.LifeStealChance = tableProfile.LifeStealChance
		}
		if hasTableProfile && tableProfile.LifeStealRatio > 0 {
			profile.LifeStealRatio = tableProfile.LifeStealRatio
		}
		if profile.LifeStealChance <= 0 {
			profile.LifeStealChance = fallbackShiXueLifeStealChance(level)
		}
		if profile.LifeStealRatio <= 0 {
			profile.LifeStealRatio = 0.7
		}
	}
	if name == "血切" {
		profile.StatusName = "外伤"
		profile.StatusDisplay = "25.png"
		profile.StatusRounds = 4
		profile.StatusChance = fallbackXueQieWoundChance(level)
		profile.StatusDescription = "每回合损失气力为角色物理攻击的25%~30%"
		profile.StatusTickMin = 25
		profile.StatusTickMax = 30
	}
	if name == "疾风刺" {
		profile.StatusName = "迟钝"
		profile.StatusDisplay = "16.png"
		profile.StatusRounds = jiFengCiSlownessRounds
		profile.StatusChance = jiFengCiSlownessChance
		profile.StatusDescription = "降低对象50%命中和回避"
	}
	if name == "赤焰魔咒" {
		profile.StatusName = "诅咒"
		profile.StatusDisplay = "780.png"
		profile.StatusRounds = chiYanMoZhouCurseRounds
		profile.StatusChance = chiYanMoZhouCurseChance
		profile.StatusDescription = "作用时间内无法增加魂元。"
	}
	if name == "雷击" {
		profile.StatusName = "迟钝"
		profile.StatusDisplay = "16.png"
		profile.StatusRounds = leiJiSlownessRounds
		profile.StatusChance = leiJiSlownessChance
		profile.StatusDescription = "降低对象30%命中和回避"
		profile.StatusHitDodgePercent = leiJiSlownessPercent
		profile.HitMultiplier = 1.5
	}
	if name == "强贯式" {
		armorBreakPercent := sourceQiangGuanShiArmorBreakPercent(level)
		profile.StatusName = "卸甲"
		profile.StatusDisplay = "10.png"
		profile.StatusRounds = qiangGuanShiArmorBreakRounds
		profile.StatusChance = 100
		profile.StatusDescription = fmt.Sprintf("降低对象%d%%物理防御力", armorBreakPercent)
		profile.StatusDefensePercent = armorBreakPercent
	}
	if name == "狂舞式" {
		stunChance := sourceKuangWuShiStunChance(level)
		profile.StatusName = "眩晕"
		profile.StatusDisplay = "9.png"
		profile.StatusRounds = kuangWuShiStunRounds
		profile.StatusChance = stunChance
		profile.StatusDescription = "眩晕无法行动"
		profile.SkipTurn = true
	}
	if name == "投毒" {
		profile.StatusName = "中毒"
		profile.StatusDisplay = "8.png"
		profile.StatusRounds = touDuPoisonRounds
		profile.StatusChance = touDuPoisonChance
		profile.StatusDescription = "降低对象15%魔防和物防，每回合内减少对象20%~25%气力"
		profile.StatusDefensePercent = touDuPoisonDefensePercent
		profile.StatusTickMin = touDuPoisonTickMin
		profile.StatusTickMax = touDuPoisonTickMax
	}
	if name == "暗影箭" {
		profile.StatusName = "混乱"
		profile.StatusDisplay = "20.png"
		profile.StatusRounds = 2
		profile.StatusChance = 17
		profile.StatusDescription = "这个状态让人失去理智&0;胡乱攻击甚至自己人。"
	}
	if name == "毒矢" {
		profile.StatusName = "中毒"
		profile.StatusDisplay = "8.png"
		profile.StatusRounds = 4
		profile.StatusChance = 70
		profile.StatusDescription = "降低对象20%魔防和物防，每回合内减少对象5%~10%气力"
		profile.StatusDefensePercent = 20
		profile.StatusTickMin = 5
		profile.StatusTickMax = 10
	}
	if name == "冰箭速射" {
		profile.StatusName = "内伤"
		profile.StatusDisplay = "26.png"
		profile.StatusRounds = 3
		profile.StatusChance = 90
		profile.StatusDescription = "削弱敌人物理攻击和魔法攻击"
		profile.StatusAttackMin = 30
		profile.StatusAttackMax = 35
	}
	if name == "夜叉棍法" {
		profile.StatusName = "内伤"
		profile.StatusDisplay = "26.png"
		profile.StatusRounds = 3
		profile.StatusChance = 90
		profile.StatusDescription = "削弱敌人物理攻击和魔法攻击"
		profile.StatusAttackMin = 32
		profile.StatusAttackMax = 32
	}
	if name == "魔力速射" {
		profile.AdditionalMagicBonus = 1.2
		profile.MagicAttackBoost = 0.25
	}
	if name == "奥义.轰雷矢" {
		profile.DefenseType = "magic"
		profile.UseMagicAttack = true
		profile.StatusName = "麻痹"
		profile.StatusDisplay = "17.png"
		profile.StatusRounds = 2
		profile.StatusChance = 20
		profile.StatusDescription = "眩晕&0;并每回合损失气力"
		profile.StatusTickMin = 30
		profile.StatusTickMax = 30
		profile.SkipTurn = true
	}
	if name == "炎狩术" || name == "赤焰魔咒" || name == "雷击" || name == "雷爆咒" || name == "雷龙强袭" {
		profile.DefenseType = "magic"
		profile.UseMagicAttack = true
	}
	if name == "雷爆咒" {
		profile.HitMultiplier = 1.5
	}
	if name == "雷龙强袭" {
		profile.HitMultiplier = 2
	}
	if name == "力释棍术" {
		profile.CanDodge = false
		profile.CanFat = false
	}
	if name == "挑衅" || name == "凝神式" || name == "气愈式" {
		profile.CanDodge = false
		profile.CanFat = false
	}
	if name == "奥义.六合棍法" {
		profile.HitMultiplier = 4
	}
	if name == "奥义.飘血" {
		profile.HitMultiplier = sourceAoYiPiaoXueHitMultiplier(level)
	}
	if name == "强力飞镖" {
		profile.DefenseType = "direct"
	}
	if name == "解毒术" {
		profile.CanDodge = false
		profile.CanFat = false
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
		sourceBattleCommandDefinitionFromSkill("普通攻击", sourceBattleSkillProfile(session.RoleSkill{Name: "普通攻击", Level: 1, Type: "oneE"})),
	}
	seen := map[string]bool{"普通攻击": true}
	for _, skill := range skills {
		normalizedName := strings.TrimSpace(skill.Name)
		if seen[normalizedName] {
			continue
		}
		if !isSourceBattleSkillLevelCaptured(normalizedName, skill.Level) {
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
		commands = append(commands, sourceBattleCommandDefinitionFromSkill(normalizedName, profile))
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

func CommandDefinitionsForSkills(skills []session.RoleSkill) []CommandDefinition {
	return sourceBattleCommandDefinitions(skills)
}

func sourceBattleCommandDefinitionFromSkill(label string, profile commandProfile) CommandDefinition {
	commandID := sourceBattleSkillCommandID(label)
	if commandID == "" && strings.TrimSpace(label) == "普通攻击" {
		commandID = CommandNormalAttack
	}
	target := sourceBattleSkillTargetFromConfig(label)
	if target == "" {
		target = sourceBattleCommandTarget(profile.SourceType)
	}
	return CommandDefinition{
		ID:                commandID,
		Kind:              "skill",
		Label:             strings.TrimSpace(label),
		SourceType:        profile.SourceType,
		ActionName:        profile.ActionName,
		SourceActionLabel: profile.SourceActionLabel,
		Target:            target,
		DamageMultiplier:  profile.DamageMultiplier,
		MPCost:            profile.MPCost,
	}
}

func sourceBattleSkillCommandID(name string) string {
	if commandID := sourceBattleSkillCommandIDFromConfig(name); commandID != "" {
		return commandID
	}
	switch strings.TrimSpace(name) {
	case "密斩":
		return CommandMiZhan
	case "多段斩":
		return CommandDuoDuanZhan
	case "多段刺":
		return CommandDuoDuanCi
	case "嗜血斩":
		return CommandShiXueZhan
	case "狂爆":
		return CommandKuangBao
	case "红月斩":
		return CommandHongYueZhan
	case "血切":
		return CommandXueQie
	case "劈山棍法":
		return CommandPiShanGunFa
	case "夜叉棍法":
		return CommandYeChaGunFa
	case "力释棍术":
		return CommandLiShiGunShu
	case "盘龙棍法":
		return CommandPanLongGunFa
	case "强力飞镖":
		return CommandQiangLiFeiBiao
	case "投毒":
		return CommandTouDu
	case "魔力突刺":
		return CommandMoLiTuCi
	case "疾风刺":
		return CommandJiFengCi
	case "解毒术":
		return CommandJieDuShu
	case "强射":
		return CommandQiangShe
	case "贯甲连矢":
		return CommandGuanJiaLianShi
	case "冰箭速射":
		return CommandBingJianSuShe
	case "魔力速射":
		return CommandMoLiSuShe
	case "暗影箭":
		return CommandAnYingJian
	case "毒矢":
		return CommandDuShi
	case "炎狩术":
		return CommandYanShouShu
	case "雷爆咒":
		return CommandLeiBaoZhou
	case "雷龙强袭":
		return CommandLeiLongQiangXi
	case "奥义.雷魂斩":
		return CommandLeiHunZhan
	case "奥义.轰雷矢":
		return CommandAoYiHongLeiShi
	case "奥义.暗杀者":
		return CommandAoYiAnShaZhe
	case "奥义.六合棍法":
		return CommandAoYiLiuHeGunFa
	default:
		return ""
	}
}

func sourceBattleSkillSourceType(name string, fallbackType string) string {
	if sourceType := sourceBattleSkillSourceTypeFromConfig(name); sourceType != "" {
		return sourceType
	}
	switch strings.TrimSpace(name) {
	case "密斩", "多段斩", "多段刺", "嗜血斩", "血切", "劈山棍法", "夜叉棍法", "强力飞镖", "投毒", "魔力突刺", "疾风刺", "强射", "贯甲连矢", "冰箭速射", "魔力速射", "暗影箭", "毒矢", "炎狩术", "雷龙强袭", "奥义.雷魂斩", "奥义.轰雷矢", "奥义.暗杀者", "奥义.六合棍法":
		return "oneE"
	case "狂爆", "解毒术", "力释棍术":
		return "own"
	case "红月斩", "盘龙棍法", "雷爆咒":
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
	sourceMPCostPattern             = regexp.MustCompile(`&2@(\d+)`)
	sourceImproveDamagePattern      = regexp.MustCompile(`提升(\d+)%的物理伤害`)
	sourceImproveMagicDamagePattern = regexp.MustCompile(`提升(\d+)%的魔法伤害`)
	sourcePercentDamagePattern      = regexp.MustCompile(`造成(?:敌人)?(\d+)%的物理伤害`)
	sourceDirectAttackBonusPattern  = regexp.MustCompile(`增加(\d+)%（无视防御）的物理攻击力`)
	sourceLifeStealChancePattern    = regexp.MustCompile(`有(\d+)%机率`)
	sourceLifeStealRatioPattern     = regexp.MustCompile(`伤害的(\d+)%`)
)

func normalizeBattleCommandID(commandID string) string {
	if mapped := sourceBattleSkillCommandID(commandID); mapped != "" {
		return mapped
	}
	switch strings.TrimSpace(commandID) {
	case "密斩":
		return CommandMiZhan
	case "多段斩":
		return CommandDuoDuanZhan
	case "多段刺":
		return CommandDuoDuanCi
	case "嗜血斩":
		return CommandShiXueZhan
	case "狂爆":
		return CommandKuangBao
	case "红月斩":
		return CommandHongYueZhan
	case "血切":
		return CommandXueQie
	case "劈山棍法":
		return CommandPiShanGunFa
	case "夜叉棍法":
		return CommandYeChaGunFa
	case "力释棍术":
		return CommandLiShiGunShu
	case "盘龙棍法":
		return CommandPanLongGunFa
	case "强力飞镖":
		return CommandQiangLiFeiBiao
	case "投毒":
		return CommandTouDu
	case "魔力突刺":
		return CommandMoLiTuCi
	case "疾风刺":
		return CommandJiFengCi
	case "解毒术":
		return CommandJieDuShu
	case "强射":
		return CommandQiangShe
	case "贯甲连矢":
		return CommandGuanJiaLianShi
	case "冰箭速射":
		return CommandBingJianSuShe
	case "魔力速射":
		return CommandMoLiSuShe
	case "暗影箭":
		return CommandAnYingJian
	case "毒矢":
		return CommandDuShi
	case "炎狩术":
		return CommandYanShouShu
	case "雷爆咒":
		return CommandLeiBaoZhou
	case "雷龙强袭":
		return CommandLeiLongQiangXi
	case "奥义.雷魂斩":
		return CommandLeiHunZhan
	case "奥义.轰雷矢":
		return CommandAoYiHongLeiShi
	case "奥义.暗杀者":
		return CommandAoYiAnShaZhe
	case "奥义.六合棍法":
		return CommandAoYiLiuHeGunFa
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
	case "多段刺":
		return session.RoleSkill{
			Name:        "多段刺",
			Level:       level,
			Type:        "oneE",
			Icon:        "257.png",
			Description: fallbackDuoDuanCiDescription(level),
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
	case "劈山棍法":
		return session.RoleSkill{
			Name:        "劈山棍法",
			Level:       level,
			Type:        "oneE",
			Icon:        "185.png",
			Description: fallbackPiShanGunFaDescription(level),
		}
	case "夜叉棍法":
		if level != 1 {
			return session.RoleSkill{}
		}
		return session.RoleSkill{
			Name:        "夜叉棍法",
			Level:       1,
			Type:        "oneE",
			Icon:        "188.png",
			Description: fallbackYeChaGunFaDescription(level),
		}
	case "力释棍术":
		if level != 1 {
			return session.RoleSkill{}
		}
		return session.RoleSkill{
			Name:        "力释棍术",
			Level:       1,
			Type:        "own",
			Icon:        "186.png",
			Description: "f_s_力释棍术^5BC46D&9@单体·状态&8@战士 &10@棍&22@战斗&2@10&4@5回合内提升物理攻击15%",
		}
	case "盘龙棍法":
		if level != 1 {
			return session.RoleSkill{}
		}
		return session.RoleSkill{
			Name:        "盘龙棍法",
			Level:       1,
			Type:        "all",
			Icon:        "187.png",
			Description: "f_s_盘龙棍法^ffffff&9@群体·攻击&8@战士 &10@棍&22@战斗&2@14&4@对所有敌人造成82%的物理伤害",
		}
	case "强力飞镖":
		return session.RoleSkill{
			Name:        "强力飞镖",
			Level:       level,
			Type:        "oneE",
			Icon:        "261.png",
			Description: fallbackQiangLiFeiBiaoDescription(level),
		}
	case "投毒":
		return session.RoleSkill{
			Name:        "投毒",
			Level:       level,
			Type:        "oneE",
			Icon:        "166.png",
			Description: fallbackTouDuDescription(level),
		}
	case "魔力突刺":
		return session.RoleSkill{
			Name:        "魔力突刺",
			Level:       level,
			Type:        "oneE",
			Icon:        "258.png",
			Description: fallbackMoLiTuCiDescription(level),
		}
	case "疾风刺":
		return session.RoleSkill{
			Name:        "疾风刺",
			Level:       level,
			Type:        "oneE",
			Icon:        "259.png",
			Description: fallbackJiFengCiDescription(level),
		}
	case "解毒术":
		return session.RoleSkill{
			Name:        "解毒术",
			Level:       level,
			Type:        "own",
			Icon:        "260.png",
			Description: fallbackJieDuShuDescription(level),
		}
	case "强射":
		return session.RoleSkill{
			Name:        "强射",
			Level:       level,
			Type:        "oneE",
			Icon:        "231.png",
			Description: fallbackQiangSheDescription(level),
		}
	case "贯甲连矢":
		return session.RoleSkill{
			Name:        "贯甲连矢",
			Level:       level,
			Type:        "oneE",
			Icon:        "236.png",
			Description: fallbackGuanJiaLianShiDescription(level),
		}
	case "冰箭速射":
		return session.RoleSkill{
			Name:        "冰箭速射",
			Level:       level,
			Type:        "oneE",
			Icon:        "233.png",
			Description: fallbackBingJianSuSheDescription(level),
		}
	case "魔力速射":
		return session.RoleSkill{
			Name:        "魔力速射",
			Level:       level,
			Type:        "oneE",
			Icon:        "234.png",
			Description: fallbackMoLiSuSheDescription(level),
		}
	case "暗影箭":
		return session.RoleSkill{
			Name:        "暗影箭",
			Level:       level,
			Type:        "oneE",
			Icon:        "235.png",
			Description: fallbackAnYingJianDescription(level),
		}
	case "毒矢":
		return session.RoleSkill{
			Name:        "毒矢",
			Level:       level,
			Type:        "oneE",
			Icon:        "237.png",
			Description: fallbackDuShiDescription(level),
		}
	case "炎狩术":
		if level != 5 {
			return session.RoleSkill{}
		}
		return session.RoleSkill{
			Name:        "炎狩术",
			Level:       5,
			Type:        "oneE",
			Icon:        "702.png",
			Description: "f_s_炎狩术^ffffff&9@单体·攻击&8@术士 &10@法杖&22@战斗&2@80&4@提升75%的魔法伤害",
		}
	case "雷爆咒":
		if level < 1 || level > 4 {
			return session.RoleSkill{}
		}
		return session.RoleSkill{
			Name:        "雷爆咒",
			Level:       level,
			Type:        "all",
			Icon:        "706.png",
			Description: fallbackLeiBaoZhouDescription(level),
		}
	case "雷龙强袭":
		if level != 1 {
			return session.RoleSkill{}
		}
		return session.RoleSkill{
			Name:        "雷龙强袭",
			Level:       1,
			Type:        "oneE",
			Icon:        "707.png",
			Description: "f_s_雷龙强袭^00ccff&9@单体·攻击&8@术士 &10@法杖&22@战斗&2@125&4@<font color='#00cc00'>特殊发动条件:需要2格魂元</font><br>提升180%的魔法伤害&0;进攻时候增加100%的命中",
		}
	case "奥义.雷魂斩":
		return session.RoleSkill{
			Name:        "奥义.雷魂斩",
			Level:       level,
			Type:        "oneE",
			Icon:        "183.png",
			Description: fallbackLeiHunZhanDescription(level),
		}
	case "奥义.轰雷矢":
		return session.RoleSkill{
			Name:        "奥义.轰雷矢",
			Level:       level,
			Type:        "oneE",
			Icon:        "238.png",
			Description: fallbackAoYiHongLeiShiDescription(level),
		}
	case "奥义.暗杀者":
		return session.RoleSkill{
			Name:        "奥义.暗杀者",
			Level:       level,
			Type:        "oneE",
			Icon:        "262.png",
			Description: fallbackAoYiAnShaZheDescription(level),
		}
	case "奥义.六合棍法":
		if level != 1 {
			return session.RoleSkill{}
		}
		return session.RoleSkill{
			Name:        "奥义.六合棍法",
			Level:       1,
			Type:        "oneE",
			Icon:        "190.png",
			Description: "f_s_奥义.六合棍法^00ccff&9@单体·攻击&8@战士 &10@棍&22@战斗&2@24&4@<font color='#00cc00'>特殊发动条件:需要3格魂元</font><br>提升210%的物理伤害&0;进攻时候增加300%的命中",
		}

	case "挑衅":
		return session.RoleSkill{
			Name:        "挑衅",
			Level:       1,
			Type:        "all",
			Icon:        "168.png",
			Description: "f_s_挑衅^5BC46D&9@群体·状态&8@战士 &10@通用&22@战斗&2@20&4@技能发动后&0;3回合内使自己成为敌人攻击的首要目标.",
		}
	case "卷叶式":
		return session.RoleSkill{
			Name:        "卷叶式",
			Level:       level,
			Type:        "oneE",
			Icon:        "170.png",
			Description: fallbackJuanYeShiDescription(level),
		}
	case "强贯式":
		return session.RoleSkill{
			Name:        "强贯式",
			Level:       level,
			Type:        "oneE",
			Icon:        "171.png",
			Description: fallbackQiangGuanShiDescription(level),
		}
	case "凝神式":
		return session.RoleSkill{
			Name:        "凝神式",
			Level:       level,
			Type:        "own",
			Icon:        "172.png",
			Description: fallbackNingShenShiDescription(level),
		}
	case "狂舞式":
		return session.RoleSkill{
			Name:        "狂舞式",
			Level:       level,
			Type:        "oneE",
			Icon:        "174.png",
			Description: fallbackKuangWuShiDescription(level),
		}
	case "气愈式":
		return session.RoleSkill{
			Name:        "气愈式",
			Level:       level,
			Type:        "own",
			Icon:        "173.png",
			Description: fallbackQiYuShiDescription(level),
		}
	case "奥义.飘血":
		return session.RoleSkill{
			Name:        "奥义.飘血",
			Level:       level,
			Type:        "oneE",
			Icon:        "175.png",
			Description: fallbackAoYiPiaoXueDescription(level),
		}
	default:
		return session.RoleSkill{}
	}
}


func (runtime *Runtime) roleSkillLevelForActor(handle string, name string, fallbackLevel int) int {
	skill, ok := runtime.roleSkillByNameForActor(handle, name)
	if !ok || skill.Level <= 0 {
		if fallbackLevel <= 0 {
			return 1
		}
		return fallbackLevel
	}
	return skill.Level
}

func sourceQiangGuanShiArmorBreakPercent(level int) int {
	switch level {
	case 1:
		return 30
	case 2:
		return 35
	case 3:
		return 40
	case 4:
		return 45
	default:
		return qiangGuanShiArmorBreakPercent
	}
}

func sourceKuangWuShiStunChance(level int) int {
	switch level {
	case 1:
		return 21
	case 2:
		return 22
	case 3:
		return 23
	case 4:
		return 24
	default:
		return kuangWuShiStunChance
	}
}

func sourceNingShenHitPercent(level int) int {
	switch level {
	case 1:
		return 50
	case 2:
		return 55
	case 3:
		return 60
	case 4:
		return 65
	default:
		return ningShenHitPercent
	}
}

func sourceQiYuHealPercent(level int) int {
	switch level {
	case 1:
		return 9
	case 2:
		return 10
	case 3:
		return 11
	case 4:
		return 12
	default:
		return qiYuHealPercent
	}
}

func sourceAoYiPiaoXueHitMultiplier(level int) float64 {
	switch level {
	case 1:
		return 1.70
	case 2:
		return 1.75
	case 3:
		return 1.80
	default:
		return 1.85
	}
}


func fallbackJuanYeShiDescription(level int) string {
	switch level {
	case 2:
		return "f_s_卷叶式^ffffff&9@单体·攻击&8@战士 &10@单剑&22@战斗&2@10&4@提升60%的物理伤害"
	case 3:
		return "f_s_卷叶式^ffffff&9@单体·攻击&8@战士 &10@单剑&22@战斗&2@12&4@提升65%的物理伤害"
	case 4:
		return "f_s_卷叶式^ffffff&9@单体·攻击&8@战士 &10@单剑&22@战斗&2@14&4@提升70%的物理伤害"
	case 5:
		return "f_s_卷叶式^ffffff&9@单体·攻击&8@战士 &10@单剑&22@战斗&2@16&4@提升75%的物理伤害"
	default:
		return "f_s_卷叶式^ffffff&9@单体·攻击&8@战士 &10@单剑&22@战斗&2@8&4@提升55%的物理伤害"
	}
}

func fallbackQiangGuanShiDescription(level int) string {
	switch level {
	case 1:
		return "f_s_强贯式^5BC46D&9@单体·攻击&8@战士 &10@单剑&22@战斗&2@12&4@对敌人造成151%的物理伤害&0;并在3回合内使敌人进入卸甲状态(降低其30%的物理防御)<br><font color='#00cc00'>叠加施放将削弱其造成卸甲的功效</font>"
	case 2:
		return "f_s_强贯式^5BC46D&9@单体·攻击&8@战士 &10@单剑&22@战斗&2@13&4@对敌人造成152%的物理伤害&0;并在3回合内使敌人进入卸甲状态(降低其35%的物理防御)<br><font color='#00cc00'>叠加施放将削弱其造成卸甲的功效</font>"
	case 3:
		return "f_s_强贯式^5BC46D&9@单体·攻击&8@战士 &10@单剑&22@战斗&2@15&4@对敌人造成153%的物理伤害&0;并在3回合内使敌人进入卸甲状态(降低其40%的物理防御)<br><font color='#00cc00'>叠加施放将削弱其造成卸甲的功效</font>"
	case 4:
		return "f_s_强贯式^5BC46D&9@单体·攻击&8@战士 &10@单剑&22@战斗&2@17&4@对敌人造成154%的物理伤害&0;并在3回合内使敌人进入卸甲状态(降低其45%的物理防御)<br><font color='#00cc00'>叠加施放将削弱其造成卸甲的功效</font>"
	default:
		return "f_s_强贯式^5BC46D&9@单体·攻击&8@战士 &10@单剑&22@战斗&2@20&4@对敌人造成155%的物理伤害&0;并在3回合内使敌人进入卸甲状态(降低其50%的物理防御)<br><font color='#00cc00'>叠加施放将削弱其造成卸甲的功效</font>"
	}
}

func fallbackNingShenShiDescription(level int) string {
	switch level {
	case 1:
		return "f_s_凝神式^5BC46D&9@单体·状态&8@战士 &10@单剑&22@战斗&2@12&4@4回合内命中提升50%&0;爆击翻倍"
	case 2:
		return "f_s_凝神式^5BC46D&9@单体·状态&8@战士 &10@单剑&22@战斗&2@14&4@4回合内命中提升55%&0;爆击翻倍"
	case 3:
		return "f_s_凝神式^5BC46D&9@单体·状态&8@战士 &10@单剑&22@战斗&2@16&4@4回合内命中提升60%&0;爆击翻倍"
	case 4:
		return "f_s_凝神式^5BC46D&9@单体·状态&8@战士 &10@单剑&22@战斗&2@18&4@4回合内命中提升65%&0;爆击翻倍"
	default:
		return "f_s_凝神式^5BC46D&9@单体·状态&8@战士 &10@单剑&22@战斗&2@22&4@4回合内命中提升70%&0;爆击翻倍"
	}
}

func fallbackKuangWuShiDescription(level int) string {
	switch level {
	case 1:
		return "f_s_狂舞式^ffffff&9@单体·攻击&8@战士 &10@单剑&22@战斗&2@22&4@造成80%的物理伤害&0;击中敌人时有21%的机率使敌人眩晕2回合"
	case 2:
		return "f_s_狂舞式^ffffff&9@单体·攻击&8@战士 &10@单剑&22@战斗&2@24&4@造成85%的物理伤害&0;击中敌人时有22%的机率使敌人眩晕2回合"
	case 3:
		return "f_s_狂舞式^ffffff&9@单体·攻击&8@战士 &10@单剑&22@战斗&2@26&4@造成90%的物理伤害&0;击中敌人时有23%的机率使敌人眩晕2回合"
	case 4:
		return "f_s_狂舞式^ffffff&9@单体·攻击&8@战士 &10@单剑&22@战斗&2@28&4@造成95%的物理伤害&0;击中敌人时有24%的机率使敌人眩晕2回合"
	default:
		return "f_s_狂舞式^ffffff&9@单体·攻击&8@战士 &10@单剑&22@战斗&2@30&4@造成100%的物理伤害&0;击中敌人时有25%的机率使敌人眩晕2回合"
	}
}

func fallbackQiYuShiDescription(level int) string {
	switch level {
	case 1:
		return "f_s_气愈式^5BC46D&9@单体·状态&8@战士 &10@单剑&22@战斗&2@15&4@每回合提升9%的气力&0;持续3回合"
	case 2:
		return "f_s_气愈式^5BC46D&9@单体·状态&8@战士 &10@单剑&22@战斗&2@19&4@每回合提升10%的气力&0;持续3回合"
	case 3:
		return "f_s_气愈式^5BC46D&9@单体·状态&8@战士 &10@单剑&22@战斗&2@23&4@每回合提升11%的气力&0;持续3回合"
	case 4:
		return "f_s_气愈式^5BC46D&9@单体·状态&8@战士 &10@单剑&22@战斗&2@27&4@每回合提升12%的气力&0;持续3回合"
	default:
		return "f_s_气愈式^5BC46D&9@单体·状态&8@战士 &10@单剑&22@战斗&2@31&4@每回合提升13%的气力&0;持续3回合"
	}
}

func fallbackAoYiPiaoXueDescription(level int) string {
	switch level {
	case 1:
		return "f_s_奥义.飘血^00ccff&9@单体·攻击&8@战士 &10@单剑&22@战斗&2@26&4@<font color='#00cc00'>特殊发动条件:需要3格魂元</font><br>提升200%的物理伤害&0;进攻时增加70%的命中"
	case 2:
		return "f_s_奥义.飘血^00ccff&9@单体·攻击&8@战士 &10@单剑&22@战斗&2@30&4@<font color='#00cc00'>特殊发动条件:需要3格魂元</font><br>提升210%的物理伤害&0;进攻时增加75%的命中"
	case 3:
		return "f_s_奥义.飘血^00ccff&9@单体·攻击&8@战士 &10@单剑&22@战斗&2@34&4@<font color='#00cc00'>特殊发动条件:需要3格魂元</font><br>提升220%的物理伤害&0;进攻时增加80%的命中"
	default:
		return "f_s_奥义.飘血^00ccff&9@单体·攻击&8@战士 &10@单剑&22@战斗&2@38&4@<font color='#00cc00'>特殊发动条件:需要3格魂元</font><br>提升230%的物理伤害&0;进攻时增加85%的命中"
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

func fallbackDuoDuanCiDescription(level int) string {
	switch level {
	case 5:
		return "f_s_多段刺^ffffff&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@18&4@提升45%的物理伤害"
	default:
		return ""
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

func fallbackPiShanGunFaDescription(level int) string {
	switch level {
	case 2:
		return "f_s_劈山棍法^ffffff&9@单体·攻击&8@战士 &10@棍&22@战斗&2@10&4@提升60%的物理伤害"
	case 3:
		return "f_s_劈山棍法^ffffff&9@单体·攻击&8@战士 &10@棍&22@战斗&2@12&4@提升65%的物理伤害"
	case 4:
		return "f_s_劈山棍法^ffffff&9@单体·攻击&8@战士 &10@棍&22@战斗&2@14&4@提升70%的物理伤害"
	case 5:
		return "f_s_劈山棍法^ffffff&9@单体·攻击&8@战士 &10@棍&22@战斗&2@16&4@提升75%的物理伤害"
	default:
		return "f_s_劈山棍法^ffffff&9@单体·攻击&8@战士 &10@棍&22@战斗&2@8&4@提升55%的物理伤害"
	}
}

func fallbackYeChaGunFaDescription(level int) string {
	if level != 1 {
		return ""
	}
	return "f_s_夜叉棍法^5BC46D&9@单体·攻击&8@战士 &10@棍&22@战斗&2@15&4@提升12%的物理伤害&0;击中敌人时有90%的机率对敌人造成内伤(削减敌人32%的物理攻击和魔法攻击)3回合<br><font color='#00cc00'>叠加施放将削弱其造成内伤的功效</font>"
}

func fallbackQiangLiFeiBiaoDescription(level int) string {
	switch level {
	case 2:
		return "f_s_强力飞镖^ffffff&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@20&4@<font color='#00cc00'>特殊发动条件:需要【飞镖x1】</font><br>进攻时提高48%（无视防御）的物理攻击力"
	case 3:
		return "f_s_强力飞镖^ffffff&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@24&4@<font color='#00cc00'>特殊发动条件:需要【飞镖x1】</font><br>进攻时提高50%（无视防御）的物理攻击力"
	default:
		return ""
	}
}

func fallbackTouDuDescription(level int) string {
	switch level {
	case 1:
		return "f_s_投毒^5BC46D&9@单体·状态&8@游侠 &10@匕首&22@战斗&2@16&4@<font color='#00cc00'>特殊发动条件:需要【毒药x1】<br>叠加施放将削弱其造成中毒的功效</font><br>有80%的机率使敌人中毒，4回合内降低对方15%的物理防御和魔法防御&0;每回合使敌人损失气力为物理攻击的20%~25%"
	default:
		return ""
	}
}

func fallbackMoLiTuCiDescription(level int) string {
	switch level {
	case 1:
		return "f_s_魔力突刺^5BC46D&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@20&4@造成敌人100%的物理伤害&0;并追加80%的魔法伤害"
	default:
		return ""
	}
}

func fallbackJiFengCiDescription(level int) string {
	switch level {
	case 1:
		return "f_s_疾风刺^5BC46D&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@20&4@对敌人造成40%的物理伤害&0;击中敌人时有92%的机率使对方进入迟钝状态(削减对方50%的命中和回避)3回合<br><font color='#00cc00'>叠加施放将削弱其造成迟钝的功效</font>"
	default:
		return ""
	}
}

func fallbackJieDuShuDescription(level int) string {
	switch level {
	case 1:
		return "f_s_解毒术^ffffff&9@单体·状态&8@游侠 &10@匕首&22@战斗&2@20&4@解除自身中毒状态"
	default:
		return ""
	}
}

func fallbackQiangSheDescription(level int) string {
	switch level {
	case 5:
		return "f_s_强射^ffffff&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@18&4@提升45%的物理伤害"
	default:
		return ""
	}
}

func fallbackGuanJiaLianShiDescription(level int) string {
	switch level {
	case 5:
		return "f_s_贯甲连矢^ffffff&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@28&4@<font color='#00cc00'>特殊发动条件:需要【穿甲箭x1】</font><br>提升25%的物理伤害&0;进攻时增加30%（无视防御）的物理攻击力."
	case 2:
		return "f_s_贯甲连矢^ffffff&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@25&4@<font color='#00cc00'>特殊发动条件:需要【穿甲箭x1】</font><br>提升10%的物理伤害&0;进攻时增加15%（无视防御）的物理攻击力."
	default:
		return ""
	}
}

func fallbackBingJianSuSheDescription(level int) string {
	switch level {
	case 5:
		return "f_s_冰箭速射^5BC46D&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@28&4@<font color='#00cc00'>特殊发动条件:需要【冰之箭x1】</font><br><font color='#00cc00'>叠加施放将削弱其造成内伤的功效</font><br>造成70%的物理伤害&0;击中敌人时有90%的机率使敌人进入内伤状态(3回合内削弱敌人30%~35%的物理攻击和魔法攻击)"
	default:
		return ""
	}
}

func fallbackMoLiSuSheDescription(level int) string {
	switch level {
	case 5:
		return "f_s_魔力速射^5BC46D&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@34&4@<font color='#00cc00'>特殊发动条件:需要【魔箭x1】</font><br>造成50%的物理伤害&0;并追加120%的魔法伤害(进攻时提高25%的魔法攻击力)"
	default:
		return ""
	}
}

func fallbackAnYingJianDescription(level int) string {
	switch level {
	case 1:
		return "f_s_暗影箭^5BC46D&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@20&4@<font color='#00cc00'>特殊发动条件:需要【暗之箭x1】</font><br>造成72%的物理伤害&0;击中敌人时有17%的机率使敌人进入混乱状态2回合"
	default:
		return ""
	}
}

func fallbackDuShiDescription(level int) string {
	switch level {
	case 1:
		return "f_s_毒矢^5BC46D&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@15&4@<font color='#00cc00'>特殊发动条件:需要【毒箭x1】<br>叠加施放将削弱其造成中毒的功效</font><br>对敌人造成90%的物理伤害&0;击中敌人时有70%的机率使敌人中毒(4回合内降低对方20%的物理防御和魔法防御&0;每回合使敌人损失气力为物理攻击的5%~10%)"
	default:
		return ""
	}
}

func fallbackLeiHunZhanDescription(level int) string {
	switch level {
	default:
		return "f_s_奥义.雷魂斩^00ccff&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@24&4@<font color='#00cc00'>特殊发动条件:需要3格魂元</font><br>提升240%的物理伤害"
	}
}

func fallbackAoYiHongLeiShiDescription(level int) string {
	switch level {
	case 1:
		return "f_s_奥义.轰雷矢^00ccff&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@26&4@<font color='#00cc00'>特殊发动条件:需要2格魂元</font><br>提升120%的魔法伤害&0;击中敌人时有20%的机率使敌人进入麻痹状态(每回合使其损失气力为魔法攻击的30%)2回合"
	default:
		return ""
	}
}

func fallbackAoYiAnShaZheDescription(level int) string {
	switch level {
	default:
		return "f_s_奥义.暗杀者^00ccff&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@26&4@<font color='#00cc00'>特殊发动条件:需要3格魂元</font><br>提升180%的物理伤害"
	}
}

func fallbackLeiBaoZhouDescription(level int) string {
	switch level {
	case 1:
		return "f_s_雷爆咒^ffffff&9@群体·攻击&8@术士 &10@法杖&22@战斗&2@70&4@提升82%的魔法伤害&0;击中敌人时有12%的机率使敌人进入麻痹状态(每回合使其损失气力为魔法攻击的5%~7%)"
	case 2:
		return "f_s_雷爆咒^ffffff&9@群体·攻击&8@术士 &10@法杖&22@战斗&2@80&4@提升84%的魔法伤害&0;击中敌人时有14%的机率使敌人进入麻痹状态(每回合使其损失气力为魔法攻击的7%~9%)"
	case 3:
		return "f_s_雷爆咒^ffffff&9@群体·攻击&8@术士 &10@法杖&22@战斗&2@90&4@提升86%的魔法伤害&0;击中敌人时有16%的机率使敌人进入麻痹状态(每回合使其损失气力为魔法攻击的9%~11%)"
	case 4:
		return "f_s_雷爆咒^ffffff&9@群体·攻击&8@术士 &10@法杖&22@战斗&2@100&4@提升88%的魔法伤害&0;击中敌人时有18%的机率使敌人进入麻痹状态(每回合使其损失气力为魔法攻击的11%~13%)"
	default:
		return ""
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
	case "多段刺":
		return "w3/ddCut"
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
	case "劈山棍法":
		return "w11/cutHill2"
	case "夜叉棍法":
		return "w11/yaksa"
	case "力释棍术":
		return "w11/releasePower"
	case "盘龙棍法":
		return "w11/circleDargon"
	case "强力飞镖":
		return "w3/powerDart"
	case "投毒":
		return "w3/drugAtk"
	case "魔力突刺":
		return "w3/magicCut"
	case "疾风刺":
		return "w3/windCut"
	case "解毒术":
		return "w3/releaseDrug"
	case "强射":
		return "w1/powerShoot"
	case "贯甲连矢":
		return "w1/breakArmorShoot2"
	case "冰箭速射":
		return "w1/iceShoot"
	case "魔力速射":
		return "w1/magicShoot"
	case "暗影箭":
		return "w1/darkShoot"
	case "毒矢":
		return "w1/drugShoot"
	case "炎狩术":
		return "w10/fire2"
	case "雷爆咒":
		return "w10/thunderBombs"
	case "雷龙强袭":
		return "w10/thunderDrongAtk"
	case "奥义.雷魂斩":
		return "w8/thunderSoulAtk"
	case "奥义.轰雷矢":
		return "w1/bombThunderShoot"
	case "奥义.暗杀者":
		return "w3/assassinate"
	case "奥义.六合棍法":
		return "w11/liuhe"
	case "普通攻击":
		return "nomalAtk"
	default:
		return sourceBattleSkillActionLabelFromConfig(name, level)
	}
}

func sourceBattleSkillMPCost(description string) int {
	return firstSourcePercentInt(sourceMPCostPattern, description)
}

func sourceBattleSkillDamageMultiplier(description string) float64 {
	if value := firstSourcePercentInt(sourceImproveDamagePattern, description); value > 0 {
		return 1 + float64(value)/100
	}
	if value := firstSourcePercentInt(sourceImproveMagicDamagePattern, description); value > 0 {
		return 1 + float64(value)/100
	}
	if value := firstSourcePercentInt(sourcePercentDamagePattern, description); value > 0 {
		return float64(value) / 100
	}
	return 0
}

func sourceBattleSkillDirectAttackBonus(description string) float64 {
	if value := firstSourcePercentInt(sourceDirectAttackBonusPattern, description); value > 0 {
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
	case "多段刺":
		if level == 5 {
			return 1.45
		}
		return 1
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
	case "劈山棍法":
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
	case "夜叉棍法":
		if level == 1 {
			return 1.12
		}
		return 1
	case "力释棍术":
		if level == 1 {
			return 0
		}
		return 0
	case "盘龙棍法":
		if level == 1 {
			return 0.82
		}
		return 0
	case "强力飞镖":
		switch level {
		case 2:
			return 1.48
		case 3:
			return 1.5
		}
		return 1
	case "投毒":
		return 0
	case "魔力突刺":
		return 1
	case "疾风刺":
		return 0.4
	case "解毒术":
		return 0
	case "强射":
		if level == 5 {
			return 1.45
		}
		return 1
	case "贯甲连矢":
		if level == 5 {
			return 1.25
		}
		if level == 2 {
			return 1.1
		}
		return 1
	case "冰箭速射":
		if level == 5 {
			return 0.7
		}
		return 1
	case "魔力速射":
		if level == 5 {
			return 0.5
		}
		return 1
	case "暗影箭":
		if level == 1 {
			return 0.72
		}
		return 1
	case "毒矢":
		if level == 1 {
			return 0.9
		}
		return 1
	case "炎狩术":
		if level == 5 {
			return 1.75
		}
		return 0
	case "雷爆咒":
		switch level {
		case 1:
			return 1.82
		case 2:
			return 1.84
		case 3:
			return 1.86
		case 4:
			return 1.88
		default:
			return 0
		}
	case "雷龙强袭":
		if level == 1 {
			return 2.8
		}
		return 0
	case "奥义.雷魂斩":
		return 3.4
	case "奥义.轰雷矢":
		if level == 1 {
			return 2.2
		}
		return 0
	case "奥义.暗杀者":
		return 2.8
	case "奥义.六合棍法":
		if level == 1 {
			return 3.1
		}
		return 0

	case "卷叶式":
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
	case "强贯式":
		switch level {
		case 1:
			return 1.51
		case 2:
			return 1.52
		case 3:
			return 1.53
		case 4:
			return 1.54
		default:
			return 1.55
		}
	case "狂舞式":
		switch level {
		case 1:
			return 0.8
		case 2:
			return 0.85
		case 3:
			return 0.9
		case 4:
			return 0.95
		default:
			return 1
		}
	case "奥义.飘血":
		switch level {
		case 1:
			return 3
		case 2:
			return 3.1
		case 3:
			return 3.2
		default:
			return 3.3
		}
	case "挑衅", "凝神式", "气愈式":
		return 0
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
	case "多段刺":
		if level == 5 {
			return 18
		}
		return 0
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
	case "劈山棍法":
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
	case "夜叉棍法":
		if level == 1 {
			return 15
		}
		return 0
	case "力释棍术":
		if level == 1 {
			return 10
		}
		return 0
	case "盘龙棍法":
		if level == 1 {
			return 14
		}
		return 0
	case "强力飞镖":
		switch level {
		case 2:
			return 20
		case 3:
			return 24
		}
		return 0
	case "投毒":
		return 16
	case "魔力突刺":
		return 20
	case "疾风刺":
		return 20
	case "解毒术":
		return 20
	case "强射":
		if level == 5 {
			return 18
		}
		return 0
	case "贯甲连矢":
		if level == 5 {
			return 28
		}
		if level == 2 {
			return 25
		}
		return 0
	case "冰箭速射":
		if level == 5 {
			return 28
		}
		return 0
	case "魔力速射":
		if level == 5 {
			return 34
		}
		return 0
	case "暗影箭":
		if level == 1 {
			return 20
		}
		return 0
	case "毒矢":
		if level == 1 {
			return 15
		}
		return 0
	case "炎狩术":
		if level == 5 {
			return 80
		}
		return 0
	case "雷爆咒":
		switch level {
		case 1:
			return 70
		case 2:
			return 80
		case 3:
			return 90
		case 4:
			return 100
		default:
			return 0
		}
	case "雷龙强袭":
		if level == 1 {
			return 125
		}
		return 0
	case "奥义.雷魂斩":
		return 24
	case "奥义.轰雷矢":
		if level == 1 {
			return 26
		}
		return 0
	case "奥义.暗杀者":
		return 26
	case "奥义.六合棍法":
		if level == 1 {
			return 24
		}
		return 0

	case "挑衅":
		return 20
	case "卷叶式":
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
	case "强贯式":
		switch level {
		case 1:
			return 12
		case 2:
			return 13
		case 3:
			return 15
		case 4:
			return 17
		default:
			return 20
		}
	case "凝神式":
		switch level {
		case 1:
			return 12
		case 2:
			return 14
		case 3:
			return 16
		case 4:
			return 18
		default:
			return 22
		}
	case "狂舞式":
		switch level {
		case 1:
			return 22
		case 2:
			return 24
		case 3:
			return 26
		case 4:
			return 28
		default:
			return 30
		}
	case "气愈式":
		switch level {
		case 1:
			return 15
		case 2:
			return 19
		case 3:
			return 23
		case 4:
			return 27
		default:
			return 31
		}
	case "奥义.飘血":
		switch level {
		case 1:
			return 26
		case 2:
			return 30
		case 3:
			return 34
		default:
			return 38
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

func fallbackXueQieWoundChance(level int) int {
	switch level {
	case 2:
		return 85
	default:
		return 80
	}
}

func (runtime *Runtime) effectiveBattleDefense(actor *CellInfoPush, target *CellInfoPush, targetInDef bool, defenseType string) int {
	if target == nil {
		return 0
	}
	if strings.EqualFold(strings.TrimSpace(defenseType), "direct") {
		return 0
	}
	if runtime.hasKuangBao(target.Handle) {
		return 0
	}
	normalizedDefenseType := strings.ToLower(strings.TrimSpace(defenseType))
	baseDefense := target.Defense
	if normalizedDefenseType == "magic" {
		baseDefense = target.MgcDefense
	}
	baseDefense = int(math.Round(float64(baseDefense) * 0.5))
	if targetInDef {
		return baseDefense * 2
	}
	return baseDefense
}

func (runtime *Runtime) hasCaptureBackedEnemyAttackRange(actor *CellInfoPush) bool {
	if runtime == nil || actor == nil || actor.Camp != CampEnemy {
		return false
	}
	attackRange, ok := runtime.EnemyAttackRanges[actor.Handle]
	return ok && attackRange.Min > 0 && attackRange.Max >= attackRange.Min
}

func (runtime *Runtime) baseBattleDamage(actor *CellInfoPush, profile commandProfile, defense int) int {
	if actor == nil {
		return 0
	}
	attack := runtime.captureBackedEnemyAttack(actor, profile)
	if profile.UseMagicAttack && actor.MagicAttack > 0 {
		attack = actor.MagicAttack
	}
	if runtime.hasKuangBao(actor.Handle) {
		attack *= 2
	}
	damage := maxInt(1, int(math.Round(float64(attack)*profile.DamageMultiplier))-defense)
	if profile.AdditionalMagicBonus > 0 && actor.MagicAttack > 0 {
		magicAttack := float64(actor.MagicAttack)
		if profile.MagicAttackBoost > 0 {
			magicAttack *= 1 + profile.MagicAttackBoost
		}
		damage += maxInt(1, int(math.Round(magicAttack*profile.AdditionalMagicBonus)))
	}
	if profile.DirectAttackBonus > 0 {
		if bonus := int(math.Round(float64(attack) * profile.DirectAttackBonus)); bonus > 0 {
			damage += bonus
		}
	}
	return damage
}

func (runtime *Runtime) applyTargetHPDamage(target *CellInfoPush, damage int) (int, int) {
	if target == nil || damage <= 0 {
		return 0, 0
	}
	beforeHP := target.HP
	mpDamage := 0
	if runtime != nil && runtime.StatusEffects != nil {
		effects := runtime.StatusEffects[target.Handle]
		if effect, ok := effects.Effects["法术屏障"]; ok && effect.Rounds > 0 && effect.DamageToMPPercent > 0 {
			mpDamage = damage * effect.DamageToMPPercent / 100
			if mpDamage > target.MP {
				mpDamage = maxInt(0, target.MP)
			}
			target.MP = maxInt(0, target.MP-mpDamage)
		}
	}
	target.HP = maxInt(0, target.HP-(damage-mpDamage))
	return beforeHP - target.HP, mpDamage
}

func (runtime *Runtime) captureBackedEnemyAttack(actor *CellInfoPush, profile commandProfile) int {
	if actor == nil {
		return 0
	}
	attack := actor.Attack
	if !runtime.hasCaptureBackedEnemyAttackRange(actor) || (profile.SourceActionLabel != "nomalAtk" && profile.SourceActionLabel != "sweepspear") {
		return attack
	}
	attackRange := runtime.EnemyAttackRanges[actor.Handle]
	return attackRange.Min + sourceBattleAttackRoll(attackRange.Max-attackRange.Min+1)
}

func sourceBattleStatusAttack(actor *CellInfoPush, profile commandProfile) int {
	if actor == nil {
		return 0
	}
	if profile.UseMagicAttack && actor.MagicAttack > 0 {
		return actor.MagicAttack
	}
	return actor.Attack
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
	if runtime.DefendingHandles[target.Handle] || runtime.StoredPower[target.Handle] > 0 {
		return false
	}
	if actor.Hit <= 0 {
		return true
	}
	actorHit := actor.Hit
	if profile.HitMultiplier > 0 {
		actorHit = int(math.Round(float64(actorHit) * profile.HitMultiplier))
	}
	chance := battleDodgeChancePercent(actorHit, target.Dog)
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

func (runtime *Runtime) applySlownessStatusEffect(actor *CellInfoPush, target *CellInfoPush, effect BattleStatusEffect) bool {
	if runtime == nil || target == nil || strings.TrimSpace(effect.Name) == "" || effect.Rounds <= 0 {
		return false
	}
	runtime.restoreExistingStatusEffect(target.Handle, "迟钝")
	hitReduction := 0
	dodgeReduction := 0
	if !effect.VisualOnly {
		percent := effect.HitDodgeReductionPercent
		if percent <= 0 {
			percent = jiFengCiSlownessPercent
		}
		hitReduction = percentReduction(target.Hit, percent)
		dodgeReduction = percentReduction(target.Dog, percent)
	}
	if hitReduction > 0 {
		target.Hit = maxInt(0, target.Hit-hitReduction)
	}
	if dodgeReduction > 0 {
		target.Dog = maxInt(0, target.Dog-dodgeReduction)
	}
	effect.Name = "迟钝"
	if strings.TrimSpace(effect.Display) == "" {
		effect.Display = "16.png"
	}
	if effect.VisualOnly {
		if strings.TrimSpace(effect.Description) == "" {
			effect.Description = "降低对象命中和回避"
		}
	} else {
		effect.Description = fmt.Sprintf("降低对象%d点命中和%d点回避", hitReduction, dodgeReduction)
	}
	effect.HitReduction = hitReduction
	effect.DodgeReduction = dodgeReduction
	runtime.applyStatusEffect(target.Handle, effect)
	runtime.PendingBuffInfos = append(runtime.PendingBuffInfos, runtime.resolveStatusBuffInfo(actor, target, effect))
	return true
}

func (runtime *Runtime) applyPoisonStatusEffect(actor *CellInfoPush, target *CellInfoPush, effect BattleStatusEffect) bool {
	if runtime == nil || target == nil || strings.TrimSpace(effect.Name) == "" || effect.Rounds <= 0 {
		return false
	}
	runtime.restoreExistingStatusEffect(target.Handle, "中毒")
	defensePercent := effect.DefenseReductionPercent
	if defensePercent <= 0 {
		defensePercent = touDuPoisonDefensePercent
	}
	magicDefenseReduction := percentReduction(target.MgcDefense, defensePercent)
	defenseReduction := percentReduction(target.Defense, defensePercent)
	if magicDefenseReduction > 0 {
		target.MgcDefense = maxInt(0, target.MgcDefense-magicDefenseReduction)
	}
	if defenseReduction > 0 {
		target.Defense = maxInt(0, target.Defense-defenseReduction)
	}
	sourceAttack := effect.SourceAttack
	if sourceAttack <= 0 && actor != nil {
		sourceAttack = actor.Attack
	}
	tickMinPercent := effect.TickMinPercent
	if tickMinPercent <= 0 {
		tickMinPercent = touDuPoisonTickMin
	}
	tickMaxPercent := effect.TickMaxPercent
	if tickMaxPercent < tickMinPercent {
		tickMaxPercent = tickMinPercent
	}
	tickMin := maxInt(1, int(math.Round(float64(sourceAttack)*float64(tickMinPercent)/100)))
	tickMax := maxInt(tickMin, int(math.Round(float64(sourceAttack)*float64(tickMaxPercent)/100)))
	effect.Name = "中毒"
	if strings.TrimSpace(effect.Display) == "" {
		effect.Display = "8.png"
	}
	effect.Description = fmt.Sprintf("降低对象%d点魔防和%d点物防，每回合内减少对象%d~%d点气力", magicDefenseReduction, defenseReduction, tickMin, tickMax)
	effect.DefenseReduction = defenseReduction
	effect.MagicDefenseReduction = magicDefenseReduction
	effect.SourceAttack = sourceAttack
	effect.DefenseReductionPercent = defensePercent
	effect.TickMinPercent = tickMinPercent
	effect.TickMaxPercent = tickMaxPercent
	runtime.applyStatusEffect(target.Handle, effect)
	runtime.PendingBuffInfos = append(runtime.PendingBuffInfos, runtime.resolveStatusBuffInfo(actor, target, effect))
	return true
}

func (runtime *Runtime) applyInnerInjuryStatusEffect(actor *CellInfoPush, target *CellInfoPush, effect BattleStatusEffect) bool {
	if runtime == nil || target == nil || strings.TrimSpace(effect.Name) == "" || effect.Rounds <= 0 {
		return false
	}
	runtime.restoreExistingStatusEffect(target.Handle, "内伤")
	minPercent := effect.StatusAttackMin
	if minPercent <= 0 {
		minPercent = equipmentInnerInjuryDefaultMin
	}
	maxPercent := effect.StatusAttackMax
	if maxPercent < minPercent {
		maxPercent = minPercent
	}
	percent := minPercent
	if maxPercent > minPercent && actor != nil {
		roll := runtime.hashBattleRollWithSalt(actor, target, effect.AppliedAction, "status:内伤:percent")
		percent += roll % (maxPercent - minPercent + 1)
	}
	attackReduction := percentReduction(target.Attack, percent)
	magicReduction := percentReduction(target.MagicAttack, percent)
	if attackReduction > 0 {
		target.Attack = maxInt(0, target.Attack-attackReduction)
	}
	if magicReduction > 0 {
		target.MagicAttack = maxInt(0, target.MagicAttack-magicReduction)
	}
	effect.Name = "内伤"
	if strings.TrimSpace(effect.Display) == "" {
		effect.Display = "26.png"
	}
	effect.Description = fmt.Sprintf("降低对象%d点物理攻击和%d点魔法攻击力", attackReduction, magicReduction)
	effect.AttackReduction = attackReduction
	effect.MagicAttackReduction = magicReduction
	runtime.applyStatusEffect(target.Handle, effect)
	runtime.PendingBuffInfos = append(runtime.PendingBuffInfos, runtime.resolveStatusBuffInfo(actor, target, effect))
	return true
}

func (runtime *Runtime) applyPalsyStatusEffect(actor *CellInfoPush, target *CellInfoPush, effect BattleStatusEffect) bool {
	if runtime == nil || target == nil || strings.TrimSpace(effect.Name) == "" || effect.Rounds <= 0 {
		return false
	}
	sourceAttack := effect.SourceAttack
	if sourceAttack <= 0 && actor != nil {
		sourceAttack = actor.MagicAttack
		if sourceAttack <= 0 {
			sourceAttack = actor.Attack
		}
	}
	tickDamage := runtime.resolveStatusTickDamage(target, effect, sourceAttack)
	effect.Name = "麻痹"
	if strings.TrimSpace(effect.Display) == "" {
		effect.Display = "17.png"
	}
	if tickDamage > 0 {
		effect.Description = fmt.Sprintf("眩晕&0;并在每回合造成%d点伤害", tickDamage)
	}
	effect.SourceAttack = sourceAttack
	runtime.applyStatusEffect(target.Handle, effect)
	runtime.PendingBuffInfos = append(runtime.PendingBuffInfos, runtime.resolveStatusBuffInfo(actor, target, effect))
	return true
}

func (runtime *Runtime) applyArmorBreakStatusEffect(actor *CellInfoPush, target *CellInfoPush, effect BattleStatusEffect) bool {
	if runtime == nil || target == nil || strings.TrimSpace(effect.Name) == "" || effect.Rounds <= 0 {
		return false
	}
	runtime.restoreExistingStatusEffect(target.Handle, "卸甲")
	sourceAttack := effect.SourceAttack
	if sourceAttack <= 0 && actor != nil {
		sourceAttack = actor.Attack
	}
	defenseReduction := 0
	if effect.DefenseReductionPercent > 0 {
		defenseReduction = percentReduction(target.Defense, effect.DefenseReductionPercent)
	} else if sourceAttack > 0 && enemyRobotawlArmorBreakAttackPct > 0 {
		defenseReduction = maxInt(1, int(math.Floor(float64(sourceAttack)*float64(enemyRobotawlArmorBreakAttackPct)/100)))
		if target.Defense < defenseReduction {
			defenseReduction = maxInt(0, target.Defense)
		}
	}
	if defenseReduction > 0 {
		target.Defense = maxInt(0, target.Defense-defenseReduction)
	}
	effect.Name = "卸甲"
	if strings.TrimSpace(effect.Display) == "" {
		effect.Display = "10.png"
	}
	effect.Description = fmt.Sprintf("降低对象%d点物理防御力", defenseReduction)
	effect.DefenseReduction = defenseReduction
	effect.SourceAttack = sourceAttack
	runtime.applyStatusEffect(target.Handle, effect)
	runtime.PendingBuffInfos = append(runtime.PendingBuffInfos, runtime.resolveStatusBuffInfo(actor, target, effect))
	return true
}

func percentReduction(value int, percent int) int {
	if value <= 0 || percent <= 0 {
		return 0
	}
	return maxInt(1, int(math.Round(float64(value)*float64(percent)/100)))
}

type equipmentInnerInjuryEffect struct {
	Chance     int
	Rounds     int
	MinPercent int
	MaxPercent int
}

type equipmentSealEffect struct {
	Chance     int
	Rounds     int
	SourceName string
}

var innerInjuryEquipmentPattern = regexp.MustCompile(`有(\d+)%机率.*进入内伤状态(\d+)回合.*降低敌人(\d+)%~(\d+)%`)
var sealEquipmentPattern = regexp.MustCompile(`(?s)有(\d+)%机率.*造成封印(\d+)回合`)

func (runtime *Runtime) applyEquipmentInnerInjuryOnHit(actor *CellInfoPush, target *CellInfoPush, commandID string) bool {
	if runtime == nil || actor == nil || target == nil || target.HP <= 0 {
		return false
	}
	effectConfig, ok := sourceEquipmentInnerInjuryEffect(runtime.itemsForHandle(actor.Handle))
	if !ok {
		return false
	}
	chance := clampInt(effectConfig.Chance, 0, 100)
	if chance <= 0 {
		return false
	}
	if chance < 100 && runtime.hashBattleRollWithSalt(actor, target, commandID, "equipment:内伤") >= chance {
		return false
	}
	percent := effectConfig.MinPercent
	if effectConfig.MaxPercent > effectConfig.MinPercent {
		roll := runtime.hashBattleRollWithSalt(actor, target, commandID, "equipment:内伤:percent")
		percent += roll % (effectConfig.MaxPercent - effectConfig.MinPercent + 1)
	}
	effect := BattleStatusEffect{
		Name:            "内伤",
		Display:         "26.png",
		Rounds:          effectConfig.Rounds,
		SourceHandle:    actor.Handle,
		SourceSkill:     "绯雨匕首",
		AppliedAction:   "equipment:inner-injury",
		StatusAttackMin: percent,
		StatusAttackMax: percent,
	}
	return runtime.applyInnerInjuryStatusEffect(actor, target, effect)
}

func sourceEquipmentInnerInjuryEffect(items []session.RoleItem) (equipmentInnerInjuryEffect, bool) {
	for _, item := range items {
		if strings.TrimSpace(item.Type) != "装备" {
			continue
		}
		description := strings.TrimSpace(item.Description)
		if !strings.Contains(description, "进入内伤状态") {
			continue
		}
		effect := equipmentInnerInjuryEffect{
			Chance:     equipmentInnerInjuryDefaultChance,
			Rounds:     equipmentInnerInjuryDefaultRounds,
			MinPercent: equipmentInnerInjuryDefaultMin,
			MaxPercent: equipmentInnerInjuryDefaultMax,
		}
		if matches := innerInjuryEquipmentPattern.FindStringSubmatch(description); len(matches) == 5 {
			effect.Chance = parsePositiveIntOrDefault(matches[1], effect.Chance)
			effect.Rounds = parsePositiveIntOrDefault(matches[2], effect.Rounds)
			effect.MinPercent = parsePositiveIntOrDefault(matches[3], effect.MinPercent)
			effect.MaxPercent = parsePositiveIntOrDefault(matches[4], effect.MaxPercent)
		}
		if effect.MaxPercent < effect.MinPercent {
			effect.MaxPercent = effect.MinPercent
		}
		return effect, true
	}
	return equipmentInnerInjuryEffect{}, false
}

func (runtime *Runtime) applyEquipmentSealOnHit(actor *CellInfoPush, target *CellInfoPush, commandID string) bool {
	if runtime == nil || actor == nil || target == nil || target.HP <= 0 {
		return false
	}
	effectConfig, ok := sourceEquipmentSealEffect(runtime.itemsForHandle(actor.Handle))
	if !ok {
		return false
	}
	chance := clampInt(effectConfig.Chance, 0, 100)
	if chance <= 0 {
		return false
	}
	if chance < 100 && runtime.hashBattleRollWithSalt(actor, target, commandID, "equipment:封印") >= chance {
		return false
	}
	effect := BattleStatusEffect{
		Name:          "封印",
		Display:       "19.png",
		Description:   "作用时间内对象无法使用技能",
		Rounds:        effectConfig.Rounds,
		SourceHandle:  actor.Handle,
		SourceSkill:   effectConfig.SourceName,
		AppliedAction: "equipment:seal",
	}
	runtime.applyStatusEffect(target.Handle, effect)
	runtime.PendingBuffInfos = append(runtime.PendingBuffInfos, runtime.resolveStatusBuffInfo(actor, target, effect))
	return true
}

func sourceEquipmentSealEffect(items []session.RoleItem) (equipmentSealEffect, bool) {
	for _, item := range items {
		if strings.TrimSpace(item.Type) != "装备" {
			continue
		}
		description := strings.TrimSpace(item.Description)
		if !strings.Contains(description, "造成封印") {
			continue
		}
		effect := equipmentSealEffect{
			Chance:     equipmentSealDefaultChance,
			Rounds:     equipmentSealDefaultRounds,
			SourceName: strings.TrimSpace(item.Name),
		}
		if effect.SourceName == "" {
			effect.SourceName = "装备封印"
		}
		if matches := sealEquipmentPattern.FindStringSubmatch(description); len(matches) == 3 {
			effect.Chance = parsePositiveIntOrDefault(matches[1], effect.Chance)
			effect.Rounds = parsePositiveIntOrDefault(matches[2], effect.Rounds)
		}
		return effect, true
	}
	return equipmentSealEffect{}, false
}

func parsePositiveIntOrDefault(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func (runtime *Runtime) restoreExistingStatusEffect(handle string, name string) {
	if runtime == nil || runtime.StatusEffects == nil {
		return
	}
	handle = strings.TrimSpace(handle)
	name = strings.TrimSpace(name)
	if handle == "" || name == "" {
		return
	}
	effects := runtime.StatusEffects[handle]
	if effects.Effects == nil {
		return
	}
	effect, ok := effects.Effects[name]
	if !ok {
		return
	}
	runtime.restoreStatusEffect(runtime.cellByHandle(handle), effect)
}

func (runtime *Runtime) clearStatusEffect(handle string, name string) bool {
	if runtime == nil || runtime.StatusEffects == nil {
		return false
	}
	handle = strings.TrimSpace(handle)
	name = strings.TrimSpace(name)
	if handle == "" || name == "" {
		return false
	}
	effects := runtime.StatusEffects[handle]
	if effects.Effects == nil {
		return false
	}
	effect, ok := effects.Effects[name]
	if !ok {
		return false
	}
	runtime.restoreStatusEffect(runtime.cellByHandle(handle), effect)
	delete(effects.Effects, name)
	if len(effects.Effects) == 0 && effects.KuangBaoRounds <= 0 {
		delete(runtime.StatusEffects, handle)
	} else {
		runtime.StatusEffects[handle] = effects
	}
	runtime.queueClearBuffInfo(handle, effect.Name)
	return true
}

func (runtime *Runtime) restoreStatusEffect(target *CellInfoPush, effect BattleStatusEffect) {
	if target == nil {
		return
	}
	if effect.AttackReduction > 0 {
		target.Attack += effect.AttackReduction
	}
	if effect.MagicAttackReduction > 0 {
		target.MagicAttack += effect.MagicAttackReduction
	}
	if effect.AttackIncrease > 0 {
		target.Attack = maxInt(0, target.Attack-effect.AttackIncrease)
	}
	if effect.DefenseReduction > 0 {
		target.Defense += effect.DefenseReduction
	}
	if effect.MagicDefenseReduction > 0 {
		target.MgcDefense += effect.MagicDefenseReduction
	}
	if effect.HitReduction > 0 {
		target.Hit += effect.HitReduction
	}
	if effect.DodgeReduction > 0 {
		target.Dog += effect.DodgeReduction
	}
	if effect.HitIncrease > 0 {
		target.Hit = maxInt(0, target.Hit-effect.HitIncrease)
	}
	if effect.FatIncrease > 0 {
		target.Fat = maxInt(0, target.Fat-effect.FatIncrease)
	}
}

func (runtime *Runtime) applyCapturedStunOnHit(actor *CellInfoPush, target *CellInfoPush, commandID string) bool {
	if runtime == nil || actor == nil || target == nil || actor.Camp != CampEnemy || target.Camp != CampTeam || target.HP <= 0 {
		return false
	}
	if !sourceEnemyCanStunOnHit(actor) {
		return false
	}
	if runtime.hashBattleRollWithSalt(actor, target, commandID, "status:眩晕") >= enemyStunOnHitChance {
		return false
	}
	effect := BattleStatusEffect{
		Name:          "眩晕",
		Display:       "9.png",
		Description:   "眩晕无法行动",
		Rounds:        2,
		SourceHandle:  actor.Handle,
		SourceSkill:   "眩晕",
		AppliedAction: "yun",
		SkipTurn:      true,
	}
	runtime.applyStatusEffect(target.Handle, effect)
	runtime.PendingBuffInfos = append(runtime.PendingBuffInfos, runtime.resolveStatusBuffInfo(actor, target, effect))
	return true
}

func (runtime *Runtime) resolveConcurrentTeamRoundStart(nextActor *CellInfoPush, teamHPBeforeEnemyTurn map[string]int) ([]ActionPush, bool, map[string]bool) {
	excludeWindows := map[string]bool{}
	if runtime == nil {
		return nil, false, excludeWindows
	}

	actions := make([]ActionPush, 0)
	nextActorSkip := false
	for _, member := range runtime.livingCells(CampTeam) {
		if member == nil || member.HP <= 0 {
			continue
		}
		memberActions, memberSkip := runtime.resolveStatusStartActions(member)
		actions = append(actions, memberActions...)
		runtime.advanceKuangBaoRound(member.Handle)
		if nextActor != nil && member.Handle == nextActor.Handle {
			nextActorSkip = memberSkip
		}
		if member.HP <= 0 {
			excludeWindows[member.Handle] = true
			if runtime.resolveWinner() != "" {
				break
			}
			continue
		}
		if memberSkip && runtime.hasActiveAutoContinueSkipStatus(member.Handle) {
			// 眩晕 still active after the status tick: no command window, auto-continue.
			excludeWindows[member.Handle] = true
			if runtime.resolveWinner() != "" {
				break
			}
			continue
		}

		hpBefore, ok := teamHPBeforeEnemyTurn[member.Handle]
		if !ok {
			hpBefore = member.HP
		}
		if !memberSkip {
			runtime.setStoredPower(member.Handle, maxInt(
				runtime.powerFor(member.Handle),
				storedPowerFromSingleHPLoss(maxInt(0, hpBefore-member.HP), member.MaxHP),
			))
		}

		// Confusion resolves at round-start for concurrent teams too, instead of
		// waiting until the confused member clicks a command window.
		if !memberSkip && runtime.consumePendingConfusion(member.Handle) {
			if action := runtime.resolveConfusionNormalAttackAction(member); action != nil {
				actions = append(actions, *action)
			}
			runtime.setStoredPower(member.Handle, 0)
			excludeWindows[member.Handle] = true
		}
		if runtime.resolveWinner() != "" {
			break
		}
	}
	return actions, nextActorSkip, excludeWindows
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
			runtime.restoreStatusEffect(actor, effect)
			delete(effects.Effects, name)
			runtime.queueClearBuffInfo(actor.Handle, effect.Name)
			continue
		}
		switch strings.TrimSpace(effect.Name) {
		case "挑衅":
			actions = append(actions, runtime.resolveSelfAction(actor, "", effect.Name, "battleStand"))
		case "法术屏障":
			actions = append(actions, runtime.resolveSelfAction(actor, "", effect.Name, "battleStand"))
		case "气疗":
			action := runtime.resolveQiLiaoStatusAction(actor, effect)
			if action != nil {
				actions = append(actions, *action)
			}
		case "外伤":
			action := runtime.resolveWoundStatusAction(actor, effect)
			if action != nil {
				actions = append(actions, *action)
			}
		case "中毒":
			action := runtime.resolvePoisonStatusAction(actor, effect)
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
		case "混乱":
			action := runtime.resolveSkipTurnStatusAction(actor, effect)
			if action != nil {
				actions = append(actions, *action)
				runtime.markPendingConfusion(actor.Handle)
			}
		case "封印":
			action := runtime.resolveSkipTurnStatusAction(actor, effect)
			if action != nil {
				actions = append(actions, *action)
				runtime.markPendingSkillSeal(actor.Handle)
			}
		}
		effect.Rounds -= 1
		if strings.TrimSpace(effect.Name) == "法术屏障" {
			runtime.PendingBuffInfos = append(runtime.PendingBuffInfos, runtime.resolveStatusBuffInfo(actor, actor, effect))
		}
		if effect.Rounds <= 0 {
			runtime.restoreStatusEffect(actor, effect)
			delete(effects.Effects, name)
			runtime.queueClearBuffInfo(actor.Handle, effect.Name)
		} else {
			effects.Effects[name] = effect
		}
	}
	if len(effects.Effects) == 0 && effects.KuangBaoRounds <= 0 {
		delete(runtime.StatusEffects, actor.Handle)
	} else {
		runtime.StatusEffects[actor.Handle] = effects
	}
	if skipTurn {
		runtime.consumePendingConfusion(actor.Handle)
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

func (runtime *Runtime) markPendingConfusion(handle string) {
	if runtime == nil || strings.TrimSpace(handle) == "" {
		return
	}
	if runtime.PendingConfusion == nil {
		runtime.PendingConfusion = map[string]bool{}
	}
	runtime.PendingConfusion[handle] = true
}

func (runtime *Runtime) consumePendingConfusion(handle string) bool {
	if runtime == nil || runtime.PendingConfusion == nil {
		return false
	}
	if !runtime.PendingConfusion[handle] {
		return false
	}
	delete(runtime.PendingConfusion, handle)
	return true
}

func (runtime *Runtime) markPendingSkillSeal(handle string) {
	if runtime == nil || strings.TrimSpace(handle) == "" {
		return
	}
	if runtime.PendingSkillSeal == nil {
		runtime.PendingSkillSeal = map[string]bool{}
	}
	runtime.PendingSkillSeal[handle] = true
}

func (runtime *Runtime) hasPendingSkillSeal(handle string) bool {
	if runtime == nil || runtime.PendingSkillSeal == nil {
		return false
	}
	return runtime.PendingSkillSeal[strings.TrimSpace(handle)]
}

func (runtime *Runtime) consumePendingSkillSeal(handle string) bool {
	if runtime == nil || runtime.PendingSkillSeal == nil {
		return false
	}
	handle = strings.TrimSpace(handle)
	if !runtime.PendingSkillSeal[handle] {
		return false
	}
	delete(runtime.PendingSkillSeal, handle)
	return true
}

func isBattleSkillCommandBlockedBySeal(commandID string) bool {
	switch strings.TrimSpace(commandID) {
	case "", CommandNormalAttack, CommandDefense, CommandStore, CommandEscape, CommandItem:
		return false
	default:
		return true
	}
}

func (runtime *Runtime) resolveConfusionNormalAttackAction(actor *CellInfoPush) *ActionPush {
	target := runtime.resolveConfusionTarget(actor)
	if target == nil {
		return nil
	}
	commandID := CommandNormalAttack
	if actor.Camp == CampEnemy {
		commandID = CommandEnemyAttack
	}
	action := runtime.resolveAttack(actor, target, commandID)
	return &action
}

func (runtime *Runtime) resolveConfusionTarget(actor *CellInfoPush) *CellInfoPush {
	if runtime == nil || actor == nil || actor.HP <= 0 {
		return nil
	}
	targets := make([]*CellInfoPush, 0, len(runtime.Cells)-1)
	for index := range runtime.Cells {
		cell := &runtime.Cells[index]
		if cell.HP <= 0 || strings.TrimSpace(cell.Handle) == "" || cell.Handle == actor.Handle {
			continue
		}
		targets = append(targets, cell)
	}
	if len(targets) == 0 {
		return nil
	}
	roll := runtime.hashBattleRollWithSalt(actor, actor, CommandNormalAttack, "status:混乱:target")
	return targets[roll%len(targets)]
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
	runtime.applyTargetHPDamage(target, damage)
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
		Sequence:              runtime.currentActionSequence(),
	}
}

func (runtime *Runtime) resolvePoisonStatusAction(target *CellInfoPush, effect BattleStatusEffect) *ActionPush {
	action := runtime.resolveWoundStatusAction(target, effect)
	if action == nil {
		return nil
	}
	action.ActionName = "中毒"
	return action
}

func (runtime *Runtime) resolveQiLiaoStatusAction(target *CellInfoPush, effect BattleStatusEffect) *ActionPush {
	if runtime == nil || target == nil || target.HP <= 0 {
		return nil
	}
	healPercent := effect.HealPercent
	if healPercent <= 0 {
		healPercent = qiYuHealPercent
	}
	healAmount := int(math.Round(float64(target.MaxHP) * float64(healPercent) / 100))
	target.HP = clampInt(target.HP+healAmount, 0, target.MaxHP)
	action := runtime.resolveSelfAction(target, "", "气疗", "battleStand")
	return &action
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
	damage := runtime.resolveStatusTickDamage(target, effect, effect.SourceAttack)
	if damage > 0 {
		runtime.applyTargetHPDamage(target, damage)
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
		Damage:                damage,
		TargetHP:              target.HP,
		TargetMP:              target.MP,
		TargetDead:            target.HP <= 0,
		RefreshInfos:          []CellInfoPush{*target},
		Round:                 runtime.Round,
		Sequence:              runtime.currentActionSequence(),
	}
}

func (runtime *Runtime) resolveStatusTickDamage(target *CellInfoPush, effect BattleStatusEffect, sourceAttack int) int {
	if sourceAttack <= 0 || effect.TickMinPercent <= 0 {
		return 0
	}
	minDamage := maxInt(1, int(math.Round(float64(sourceAttack)*float64(effect.TickMinPercent)/100)))
	maxPercent := effect.TickMaxPercent
	if maxPercent < effect.TickMinPercent {
		maxPercent = effect.TickMinPercent
	}
	maxDamage := maxInt(minDamage, int(math.Round(float64(sourceAttack)*float64(maxPercent)/100)))
	if minDamage == maxDamage || target == nil || runtime == nil {
		return minDamage
	}
	roll := runtime.hashBattleRollWithSalt(target, target, "status-tick", effect.SourceHandle+":"+effect.SourceSkill+":"+effect.Name)
	return minDamage + roll%(maxDamage-minDamage+1)
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

func (runtime *Runtime) queueClearBuffInfo(targetHandle string, name string) {
	targetHandle = strings.TrimSpace(targetHandle)
	name = strings.TrimSpace(name)
	if runtime == nil || targetHandle == "" || name == "" {
		return
	}
	for _, existing := range runtime.PendingClearBuffInfos {
		if existing.TargetHandle == targetHandle && existing.Name == name {
			return
		}
	}
	runtime.PendingClearBuffInfos = append(runtime.PendingClearBuffInfos, ClearBuffInfoPush{
		BattleID:     runtime.BattleID,
		TargetHandle: targetHandle,
		Name:         name,
	})
}

func (runtime *Runtime) consumePendingClearBuffInfos() []ClearBuffInfoPush {
	if len(runtime.PendingClearBuffInfos) == 0 {
		return nil
	}
	clearBuffInfos := append([]ClearBuffInfoPush(nil), runtime.PendingClearBuffInfos...)
	runtime.PendingClearBuffInfos = nil
	return clearBuffInfos
}

func (runtime *Runtime) applyTauntStatusEffect(actor *CellInfoPush) BuffInfoPush {
	if runtime == nil || actor == nil {
		return BuffInfoPush{}
	}
	runtime.restoreExistingStatusEffect(actor.Handle, "挑衅")
	effect := BattleStatusEffect{
		Name:          "挑衅",
		Display:       "27.png",
		Description:   "敌人每次必攻击该对象",
		Rounds:        tauntRounds,
		SourceHandle:  actor.Handle,
		SourceSkill:   "挑衅",
		AppliedAction: "com/tx",
	}
	runtime.applyStatusEffect(actor.Handle, effect)
	return runtime.resolveStatusBuffInfo(actor, actor, effect)
}

func (runtime *Runtime) applyNingShenStatusEffects(actor *CellInfoPush) []BuffInfoPush {
	if runtime == nil || actor == nil {
		return nil
	}
	runtime.restoreExistingStatusEffect(actor.Handle, "集中")
	runtime.restoreExistingStatusEffect(actor.Handle, "爆击提升")
	hitPercent := sourceNingShenHitPercent(runtime.roleSkillLevelForActor(actor.Handle, "凝神式", 5))
	hitIncrease := int(math.Round(float64(actor.Hit) * float64(hitPercent) / 100))
	fatIncrease := actor.Fat
	actor.Hit += hitIncrease
	actor.Fat += fatIncrease
	concentration := BattleStatusEffect{
		Name:          "集中",
		Display:       "15.png",
		Description:   fmt.Sprintf("提高对象%d命中", hitIncrease),
		Rounds:        ningShenRounds,
		SourceHandle:  actor.Handle,
		SourceSkill:   "凝神式",
		AppliedAction: "w7/nss",
		HitIncrease:   hitIncrease,
	}
	criticalBoost := BattleStatusEffect{
		Name:          "爆击提升",
		Display:       "653.png",
		Description:   fmt.Sprintf("提高对象%d爆击", fatIncrease),
		Rounds:        ningShenRounds,
		SourceHandle:  actor.Handle,
		SourceSkill:   "凝神式",
		AppliedAction: "w7/nss",
		FatIncrease:   fatIncrease,
	}
	runtime.applyStatusEffect(actor.Handle, concentration)
	runtime.applyStatusEffect(actor.Handle, criticalBoost)
	return []BuffInfoPush{
		runtime.resolveStatusBuffInfo(actor, actor, concentration),
		runtime.resolveStatusBuffInfo(actor, actor, criticalBoost),
	}
}

func (runtime *Runtime) applyQiYuStatusEffect(actor *CellInfoPush) BuffInfoPush {
	if runtime == nil || actor == nil {
		return BuffInfoPush{}
	}
	runtime.restoreExistingStatusEffect(actor.Handle, "气疗")
	healPercent := sourceQiYuHealPercent(runtime.roleSkillLevelForActor(actor.Handle, "气愈式", 5))
	healAmount := int(math.Round(float64(actor.MaxHP) * float64(healPercent) / 100))
	effect := BattleStatusEffect{
		Name:          "气疗",
		Display:       "21.png",
		Description:   fmt.Sprintf("每回合对象恢复%d气力", healAmount),
		Rounds:        qiYuRounds,
		SourceHandle:  actor.Handle,
		SourceSkill:   "气愈式",
		AppliedAction: "w7/qys",
		HealPercent:   healPercent,
	}
	runtime.applyStatusEffect(actor.Handle, effect)
	return runtime.resolveStatusBuffInfo(actor, actor, effect)
}

func (runtime *Runtime) applyMagicBarrierStatusEffect(actor *CellInfoPush) BuffInfoPush {
	if runtime == nil || actor == nil {
		return BuffInfoPush{}
	}
	runtime.restoreExistingStatusEffect(actor.Handle, "法术屏障")
	effect := BattleStatusEffect{
		Name:              "法术屏障",
		Display:           "28.png",
		Description:       "作用时间内气力损失量的35%以精力来代替",
		Rounds:            magicObstacleRounds,
		SourceHandle:      actor.Handle,
		SourceSkill:       "魔障术",
		AppliedAction:     "w10/magicObstacle",
		DamageToMPPercent: magicObstacleDamageToMPPercent,
	}
	runtime.applyStatusEffect(actor.Handle, effect)
	return runtime.resolveStatusBuffInfo(actor, actor, effect)
}

func (runtime *Runtime) applyFightingSpiritStatusEffect(actor *CellInfoPush) BuffInfoPush {
	if runtime == nil || actor == nil {
		return BuffInfoPush{}
	}
	runtime.restoreExistingStatusEffect(actor.Handle, "斗志")
	increase := maxInt(1, int(math.Round(float64(actor.Attack)*float64(liShiGunShuAttackPercent)/100)))
	actor.Attack += increase
	description := fmt.Sprintf("提升对象%d点物理攻击", increase)
	effect := BattleStatusEffect{
		Name:           "斗志",
		Display:        "23.png",
		Description:    description,
		Rounds:         liShiGunShuRounds,
		SourceHandle:   actor.Handle,
		SourceSkill:    "力释棍术",
		AppliedAction:  "w11/releasePower",
		AttackIncrease: increase,
	}
	runtime.applyStatusEffect(actor.Handle, effect)
	return BuffInfoPush{
		BattleID:      runtime.BattleID,
		ReleaseHandle: actor.Handle,
		TargetHandle:  actor.Handle,
		Name:          "斗志",
		Display:       "23.png",
		Description:   description,
		Round:         liShiGunShuRounds,
	}
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
	// Seed with the active command-window sequence (request.Sequence via actionSequence),
	// not the allocator cursor nextSequence. Opening a window advances nextSequence so the
	// next free slot can be assigned; using that cursor would shift first-turn rolls from
	// seed 1 to seed 2 after NewWildBattle/N=1 concurrent open.
	seed := fmt.Sprintf("%s:%d:%d:%s:%s:%s:%s", runtime.BattleID, runtime.Round, runtime.currentActionSequence(), actor.Handle, target.Handle, commandID, salt)
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
		Sequence:          runtime.currentActionSequence(),
	}
}

func (runtime *Runtime) resolveProfileSelfAction(actor *CellInfoPush, commandID string, profile commandProfile) ActionPush {
	action := runtime.resolveSelfAction(actor, commandID, profile.ActionName, profile.SourceActionLabel)
	action.SourceMode = sourceBattleActionMode(profile.SourceType)
	return action
}

// resolveCapturedTauntAction matches capture shape:
// 挑衅,<actionId>,<actor>,all,1,com/tx
// sourceMode=1 drives the original Attack() approach to stage center; target=all is protocol
// field only and does not mean multi-hit damage.
func (runtime *Runtime) resolveCapturedTauntAction(actor *CellInfoPush, commandID string, profile commandProfile) ActionPush {
	action := runtime.resolveSelfAction(actor, commandID, profile.ActionName, profile.SourceActionLabel)
	action.TargetHandle = "all"
	action.SourceMode = "1"
	if strings.TrimSpace(action.SourceActionLabel) == "" {
		action.SourceActionLabel = "com/tx"
	}
	return action
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
		Sequence:          runtime.currentActionSequence(),
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

	if reward, ok := runtime.sourceBattleRewardConfig(); ok && reward.Status == "confirmed" {
		items := rollSourceBattleRewardItems(reward.Items, reward.DropRates)
		return reward.ExpDelta, runtime.appendSourceBattleRewardEquipmentDrops(items, reward.DropRates)
	}

	if reward, ok := runtime.sourceBattleRewardCandidate(); ok {
		items := rollSourceBattleRewardItems(nil, reward.DropRates)
		return reward.ExpDelta, runtime.appendSourceBattleRewardEquipmentDrops(items, reward.DropRates)
	}
	return 0, []string{}
}

func (runtime *Runtime) RerollBattleRewardItems(result ResultPayload) ResultPayload {
	if runtime == nil || result.Winner != CampTeam || result.Escaped || !runtime.hasSourceBattleRewardSource() {
		return result
	}
	_, items := runtime.sourceBattleRewards(result.Winner, result.Escaped)
	reroll := result
	reroll.Items = items
	return reroll
}

func (runtime *Runtime) hasSourceBattleRewardSource() bool {
	if runtime == nil {
		return false
	}
	if reward, ok := runtime.sourceBattleRewardConfig(); ok && reward.Status == "confirmed" {
		return true
	}
	_, ok := runtime.sourceBattleRewardCandidate()
	return ok
}

func (runtime *Runtime) sourceBattleRewardConfig() (sourceBattleRewardConfig, bool) {
	if runtime == nil {
		return sourceBattleRewardConfig{}, false
	}
	if strings.TrimSpace(runtime.SourceMonsterHandle) != "" {
		return sourceBattleRewardConfigForEncounter(runtime.MapID, runtime.SourceMonsterHandle)
	}
	for _, cell := range runtime.Cells {
		if cell.Camp != CampEnemy {
			continue
		}
		if reward, ok := sourceBattleRewardConfigForExactEncounter(runtime.MapID, cell.Handle); ok {
			return reward, true
		}
	}
	return sourceBattleRewardConfigForMap(runtime.MapID)
}

func (runtime *Runtime) sourceBattleRewardCandidate() (sourceBattleRewardCandidateConfig, bool) {
	if runtime == nil {
		return sourceBattleRewardCandidateConfig{}, false
	}
	for _, cell := range runtime.sourceBattleRewardEnemyCells() {
		if reward, ok := sourceBattleRewardCandidateForCell(runtime.MapID, cell.Name, cell.MaxHP); ok {
			return reward, true
		}
	}
	return sourceBattleRewardCandidateConfig{}, false
}

func (runtime *Runtime) appendSourceBattleRewardEquipmentDrops(items []string, baseDropRates []sourceBattleRewardDropRate) []string {
	if runtime == nil {
		return items
	}
	cell, ok := runtime.sourceBattleRewardEnemyCell()
	if !ok {
		return items
	}

	stattedItems := sourceBattleRewardDropRateItemSet(baseDropRates)
	if candidate, ok := sourceBattleRewardCandidateForCell(runtime.MapID, cell.Name, cell.MaxHP); ok {
		supplementalRates := sourceBattleRewardSupplementalEquipmentDropRates(candidate.DropRates, stattedItems)
		items = append(items, rollSourceBattleRewardItems(nil, supplementalRates)...)
	}

	fallbackPool := sourceBattleRewardEquipmentFallbackPool(cell.Name, stattedItems)
	if len(fallbackPool) == 0 {
		return items
	}
	if sourceEncounterRoll(5) >= 1 {
		return items
	}
	items = append(items, formatSourceBattleRewardItemStack(fallbackPool[sourceEncounterRoll(len(fallbackPool))], 1))
	return items
}

func (runtime *Runtime) sourceBattleRewardEnemyCell() (CellInfoPush, bool) {
	cells := runtime.sourceBattleRewardEnemyCells()
	if len(cells) == 0 {
		return CellInfoPush{}, false
	}
	return cells[0], true
}

func (runtime *Runtime) sourceBattleRewardEnemyCells() []CellInfoPush {
	if runtime == nil {
		return []CellInfoPush{}
	}
	cells := make([]CellInfoPush, 0, len(runtime.Cells))
	for _, cell := range runtime.Cells {
		if cell.Camp != CampEnemy {
			continue
		}
		cells = append(cells, cell)
	}
	return cells
}

func sourceBattleRewardDropRateItemSet(dropRates []sourceBattleRewardDropRate) map[string]bool {
	items := map[string]bool{}
	for _, drop := range dropRates {
		itemName := strings.TrimSpace(drop.ItemName)
		if itemName == "" {
			continue
		}
		items[itemName] = true
	}
	return items
}

func sourceBattleRewardSupplementalEquipmentDropRates(dropRates []sourceBattleRewardDropRate, stattedItems map[string]bool) []sourceBattleRewardDropRate {
	result := make([]sourceBattleRewardDropRate, 0)
	for _, drop := range dropRates {
		itemName := strings.TrimSpace(drop.ItemName)
		if itemName == "" || stattedItems[itemName] || !sourceBattleRewardItemIsEquipment(itemName) {
			continue
		}
		stattedItems[itemName] = true
		drop.ItemName = itemName
		result = append(result, drop)
	}
	return result
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

func (runtime *Runtime) resolveEnemyTeamTarget(enemy *CellInfoPush) *CellInfoPush {
	if runtime == nil || enemy == nil {
		return nil
	}
	targets := runtime.livingCells(CampTeam)
	for _, target := range targets {
		if runtime.hasTaunt(target.Handle) {
			return target
		}
	}
	switch len(targets) {
	case 0:
		return nil
	case 1:
		return targets[0]
	default:
		// Deterministic roll among living team members so multi-player battles
		// do not always hit firstLiving(CampTeam).
		roll := runtime.hashBattleRollWithSalt(enemy, enemy, CommandEnemyAttack, "enemy-target")
		return targets[roll%len(targets)]
	}
}

func (runtime *Runtime) hasTaunt(handle string) bool {
	if runtime == nil || runtime.StatusEffects == nil {
		return false
	}
	effect, ok := runtime.StatusEffects[strings.TrimSpace(handle)].Effects["挑衅"]
	return ok && effect.Rounds > 0
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

func (runtime *Runtime) livingTeamActorsExcluding(exclude map[string]bool) []*CellInfoPush {
	if runtime == nil {
		return nil
	}
	actors := make([]*CellInfoPush, 0)
	for _, cell := range runtime.livingCells(CampTeam) {
		if cell == nil || cell.HP <= 0 {
			continue
		}
		if exclude != nil && exclude[cell.Handle] {
			continue
		}
		actors = append(actors, cell)
	}
	return actors
}

func (runtime *Runtime) nextLivingTeamActorAfter(handle string) *CellInfoPush {
	if runtime == nil {
		return nil
	}
	teamIndexes := make([]int, 0, len(runtime.Cells))
	currentTeamIndex := -1
	for index := range runtime.Cells {
		if runtime.Cells[index].Camp != CampTeam {
			continue
		}
		if runtime.Cells[index].Handle == handle {
			currentTeamIndex = len(teamIndexes)
		}
		teamIndexes = append(teamIndexes, index)
	}
	if len(teamIndexes) == 0 {
		return nil
	}
	start := 0
	if currentTeamIndex >= 0 {
		start = (currentTeamIndex + 1) % len(teamIndexes)
	}
	for offset := 0; offset < len(teamIndexes); offset++ {
		index := teamIndexes[(start+offset)%len(teamIndexes)]
		if runtime.Cells[index].HP > 0 {
			return &runtime.Cells[index]
		}
	}
	return nil
}

func (runtime *Runtime) resetPendingTeamActions() {
	runtime.resetPendingTeamActionsExcluding(nil)
}

// resetPendingTeamActionsExcluding opens concurrent windows for every living free
// teammate. Solo is just N=1 on the same machine.
func (runtime *Runtime) resetPendingTeamActionsExcluding(exclude map[string]bool) {
	if runtime == nil {
		return
	}
	if len(runtime.livingCells(CampTeam)) < 1 {
		runtime.PendingTeamActions = nil
		runtime.PendingTeamSequences = nil
		return
	}
	runtime.PendingTeamActions = map[string]bool{}
	runtime.PendingTeamSequences = map[string]int{}
	for _, cell := range runtime.livingCells(CampTeam) {
		if exclude != nil && exclude[cell.Handle] {
			continue
		}
		for runtime.ConsumedSequence != nil && runtime.ConsumedSequence[runtime.nextSequence] {
			runtime.nextSequence += 1
		}
		runtime.PendingTeamActions[cell.Handle] = true
		runtime.PendingTeamSequences[cell.Handle] = runtime.nextSequence
		runtime.nextSequence += 1
	}
	if len(runtime.PendingTeamActions) == 0 {
		runtime.PendingTeamActions = nil
		runtime.PendingTeamSequences = nil
	}
}

func (runtime *Runtime) pendingTeamStartCommands() []StartCommandPush {
	if runtime == nil || len(runtime.PendingTeamActions) == 0 {
		return nil
	}
	commands := make([]StartCommandPush, 0, len(runtime.PendingTeamActions))
	for _, cell := range runtime.Cells {
		sequence, ok := runtime.PendingTeamSequences[cell.Handle]
		if !ok || cell.Camp != CampTeam || cell.HP <= 0 || !runtime.PendingTeamActions[cell.Handle] {
			continue
		}
		commands = append(commands, StartCommandPush{
			BattleID:    runtime.BattleID,
			ActorHandle: cell.Handle,
			Round:       runtime.Round,
			Sequence:    sequence,
			Power:       runtime.powerFor(cell.Handle),
			Commands:    runtime.commandDefinitionsForActor(cell.Handle),
		})
	}
	return commands
}

func (runtime *Runtime) prunePendingTeamActions() {
	if runtime == nil {
		return
	}
	for handle := range runtime.PendingTeamActions {
		cell := runtime.cellByHandle(handle)
		if cell != nil && cell.Camp == CampTeam && cell.HP > 0 {
			continue
		}
		delete(runtime.PendingTeamActions, handle)
		delete(runtime.PendingTeamSequences, handle)
	}
}

func (runtime *Runtime) ensureConcurrentCommandWindows() {
	if runtime == nil || runtime.PendingTeamActions != nil || runtime.Phase != PhaseCommand {
		return
	}
	// Hand-built runtimes/tests may only set ActiveHandle. Lift them onto concurrent windows.
	active := strings.TrimSpace(runtime.ActiveHandle)
	if active != "" {
		actor := runtime.cellByHandle(active)
		if actor != nil && actor.Camp == CampTeam && actor.HP > 0 {
			runtime.PendingTeamActions = map[string]bool{active: true}
			runtime.PendingTeamSequences = map[string]int{active: runtime.nextSequence}
			return
		}
	}
	runtime.resetPendingTeamActionsExcluding(nil)
}

func (runtime *Runtime) commandWindowForActor(handle string) (StartCommandPush, bool) {
	if runtime == nil {
		return StartCommandPush{}, false
	}
	runtime.ensureConcurrentCommandWindows()
	handle = strings.TrimSpace(handle)
	actor := runtime.cellByHandle(handle)
	if actor == nil || actor.Camp != CampTeam || actor.HP <= 0 {
		return StartCommandPush{}, false
	}
	sequence, ok := runtime.PendingTeamSequences[handle]
	if !ok || runtime.PendingTeamActions == nil || !runtime.PendingTeamActions[handle] {
		return StartCommandPush{}, false
	}
	return StartCommandPush{
		BattleID:    runtime.BattleID,
		ActorHandle: handle,
		Round:       runtime.Round,
		Sequence:    sequence,
		Power:       runtime.powerFor(handle),
		Commands:    runtime.commandDefinitionsForActor(handle),
	}, true
}

func (runtime *Runtime) CommandWindowForActor(handle string) (StartCommandPush, bool) {
	if runtime == nil {
		return StartCommandPush{}, false
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	if runtime.Phase != PhaseCommand {
		return StartCommandPush{}, false
	}
	return runtime.commandWindowForActor(handle)
}

func (runtime *Runtime) HasPendingTeamAction(handle string) bool {
	if runtime == nil {
		return false
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	if runtime.Phase != PhaseCommand || runtime.PendingTeamActions == nil {
		return false
	}
	_, ok := runtime.commandWindowForActor(handle)
	return ok
}

func (runtime *Runtime) currentActionSequence() int {
	if runtime == nil || runtime.actionSequence <= 0 {
		return runtime.nextSequence
	}
	return runtime.actionSequence
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
	if runtime.hasActiveStatusEffect(handle, "诅咒") && value > runtime.StoredPower[handle] {
		return
	}
	runtime.StoredPower[handle] = clampInt(value, 0, maxStoredPower)
}

func (runtime *Runtime) hasActiveStatusEffect(handle string, name string) bool {
	if runtime == nil || runtime.StatusEffects == nil {
		return false
	}
	effect, ok := runtime.StatusEffects[strings.TrimSpace(handle)].Effects[strings.TrimSpace(name)]
	return ok && effect.Rounds > 0
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

func sourceEnemyConfigsForEncounter(mapID string, _ float64) []sourceWildEnemyConfig {
	encounters := sourceWildEncounterConfigsForMap(mapID)
	if len(encounters) <= 0 {
		return nil
	}
	totalWeight := 0
	for _, encounter := range encounters {
		totalWeight += encounter.Weight
	}
	roll := sourceEncounterRoll(totalWeight)
	selected := encounters[len(encounters)-1]
	for _, encounter := range encounters {
		if roll < encounter.Weight {
			selected = encounter
			break
		}
		roll -= encounter.Weight
	}
	configsByHandle := map[string]sourceWildEnemyConfig{}
	for _, config := range sourceEnemyConfigsForMap(mapID) {
		configsByHandle[config.Cell.Handle] = config
	}
	configs := make([]sourceWildEnemyConfig, 0, len(selected.EncounterHandles))
	for _, handle := range selected.EncounterHandles {
		config, ok := configsByHandle[handle]
		if !ok {
			return nil
		}
		configs = append(configs, config)
	}
	return configs
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

func cloneBattleRoleItems(items []session.RoleItem) []session.RoleItem {
	if len(items) == 0 {
		return nil
	}
	cloned := make([]session.RoleItem, len(items))
	copy(cloned, items)
	return cloned
}

func ParseMapID(mapID string) (int, bool) {
	value, err := strconv.Atoi(strings.TrimSpace(mapID))
	return value, err == nil
}
