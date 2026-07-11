package session

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"ai-server/internal/guild"
	"ai-server/internal/mall"
	_ "modernc.org/sqlite"
)

const (
	defaultRoleVoc         = "新手"
	defaultSkillCap        = 12
	defaultBagCap          = 30
	bagCapacityStep        = 6
	defaultEquipCap        = 20
	defaultMallCap         = 30
	defaultWarehouseCap    = 40
	defaultCopper          = 5000
	defaultSilver          = 1
	defaultPetFullness     = 100
	defaultPetID           = "-1"
	rolePetEquipIndex      = 9
	roleTreasureEquipIndex = 14
	roleMountEquipIndex    = 18
	roleFashionClothIndex  = 4
	roleFashionPantsIndex  = 5
	roleFashionShoesIndex  = 12

	DungeonInstanceShuiliandong  = "shuiliandong"
	DungeonInstanceHuangfengzhai = "huangfengzhai"
	DungeonInstanceFeixiandong   = "feixiandong"
	DungeonInstanceShihuku       = "shihuku"
)

const dungeonInstanceTTL = time.Hour
const roleTownAvoidBuffDuration = 5 * time.Minute
const roleTownInitialExperienceBoostDuration = time.Hour
const roleTownAdvancedExperienceBoostDuration = 3 * time.Hour

const (
	classicTownAvoidBuffName                        = "\u907f\u602a"
	classicTownAvoidBuffItemName                    = "\u907f\u602a\u7b26"
	classicTownAvoidBuffLocalItemName               = "\u004c\u907f\u602a\u7b26"
	classicTownAvoidBuffDescription                 = "\u70b9\u51fb\u53d6\u6d88\u8be5\u72b6\u6001\uff1b5\u5206\u949f\u5185\u4e0d\u4f1a\u9047\u654c\uff1b\u660e\u602a\u65e0\u6548\u3002"
	classicTownAvoidBuffSourceCapture               = "woc-proxy-captures/20260612_211741_424_session_38832/connections/20260612_211756_199_conn_0002/raw/client-to-server-0001.bin#packetIndex=1433 RemoveBuff"
	classicTownRemoveAbateBuffCapture               = "instance2.staging/tmp/woc-proxy-captures/20260606_210926_394_session_08036/connections/20260607_013640_125_conn_0011/raw/client-to-server-0001.bin#packetIndex=368 RemoveABateBuff"
	classicTownInitialExperienceCardName            = "\u004c\u521d\u9636\u7ecf\u9a8c\u5361"
	classicTownInitialExperienceBoostName           = "\u53cc\u500d\u7ecf\u9a8c"
	classicTownInitialExperienceBoostDisplay        = "567.png"
	classicTownInitialExperienceBoostDescription    = "\u5728\u6218\u6597\u4e2d\u83b7\u5f97\u53cc\u500d\u7684\u7ecf\u9a8c"
	classicTownInitialExperienceBoostSourceCapture  = "D:/yzhgame/WOCClient/tmp/woc-proxy-captures/20260531_023839_239_session_45660/connections/20260531_023846_804_conn_0002/raw/server-to-client-0001.bin#330/#333/#334 after ActiveItemByIndex(114) bag slot 1"
	classicTownAdvancedExperienceCardName           = "\u004c\u8fdb\u9636\u7ecf\u9a8c\u5361"
	classicTownAdvancedExperienceBoostSourceCapture = "D:/yzhgame/WOCClient/tmp/woc-proxy-captures/20260612_211741_424_session_38832/connections/20260612_211756_199_conn_0002/raw/server-to-client-0001.bin#2844/#2847/#2848 after ActiveItemByIndex(114) bag slot 11"
	classicTownLevel1GiftBoxName                    = "\u0031\u7ea7\u793c\u76d2"
	classicTownLevel5GiftBoxName                    = "\u0035\u7ea7\u793c\u76d2"
	classicTownLevel5GiftBoxError                   = "\u89d2\u8272\u7b49\u7ea7\u5fc5\u987b\u5230\u8fbeLv5"
	classicTownLevel10GiftBoxName                   = "\u0031\u0030\u7ea7\u793c\u76d2"
	classicTownLevel10GiftBoxFullCode               = "level10_gift_box_bag_full"
	classicTownGiftBoxBagFullError                  = "\u80cc\u5305\u7a7a\u95f4\u4e0d\u8db3"
	classicTownBagCapacityPatchName                 = "\u004c\u80cc\u5305\u8865\u4e01"
)

var classicPetLevelToExp = []int{
	0, 200, 500, 900, 1400, 2100, 3000, 4100, 5400, 6900,
	8700, 10800, 13200, 15900, 19000, 22500, 26500, 31000, 36100, 41800,
	48200, 55300, 63200, 72000, 81800, 92700, 104800, 118200, 133000, 149400,
	167500, 187500, 209600, 234000, 261000, 290800, 323700, 360000, 400000, 444100,
	492700, 546300, 605400, 670500, 742200, 821200, 908200, 1004000, 1109500, 1225700,
}

func DungeonInstanceTTLSeconds() int64 {
	return int64(dungeonInstanceTTL / time.Second)
}

func DungeonInstanceExpiresAtUnix(state DungeonInstanceState) int64 {
	if state.CreatedAtUnix <= 0 {
		return 0
	}
	return state.CreatedAtUnix + DungeonInstanceTTLSeconds()
}

type RoleAppearance map[string]any

type RoleCurrencies map[string]int

type RoleSkill struct {
	Name        string `json:"name"`
	Level       int    `json:"level"`
	Type        string `json:"type"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
	MaxLevel    int    `json:"maxLevel,omitempty"`
}

type RoleFastPanelEntry struct {
	Index int    `json:"index"`
	Type  string `json:"type"`
	Name  string `json:"name"`
}

type RoleItem struct {
	Handle      string            `json:"handle,omitempty"`
	Type        string            `json:"type"`
	Name        string            `json:"name"`
	ItemType    string            `json:"itemType"`
	Display     string            `json:"display"`
	Description string            `json:"description"`
	Count       int               `json:"count"`
	Index       int               `json:"index"`
	Level       int               `json:"level"`
	EndTime     int               `json:"endTime"`
	Owner       string            `json:"owner"`
	ItemLevel   int               `json:"itemLevel"`
	PetState    *RolePetItemState `json:"petState,omitempty"`
}

type RolePetItemState struct {
	Level    int    `json:"level"`
	Exp      int    `json:"exp"`
	Fullness int    `json:"fullness"`
	PetID    string `json:"petId,omitempty"`
}

type RoleTownBuff struct {
	Handle        string `json:"handle"`
	Name          string `json:"name"`
	Display       string `json:"display"`
	Description   string `json:"description"`
	BattleOnly    int    `json:"battleOnly"`
	EndTime       int64  `json:"endTime"`
	SourceCapture string `json:"sourceCapture,omitempty"`
	Partial       bool   `json:"partial,omitempty"`
}

type LoginRequest struct {
	Platform string `json:"platform"`
	AuthCode string `json:"authCode,omitempty"`
	UserName string `json:"userName,omitempty"`
	Password string `json:"password,omitempty"`
}

type LoginResponse struct {
	PlayerID     string `json:"playerId"`
	SessionToken string `json:"sessionToken"`
	DisplayName  string `json:"displayName"`
	Success      bool   `json:"success"`
	ErrorCode    string `json:"errorCode,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

type RoleSummary struct {
	RoleID            string                          `json:"roleId"`
	DisplayName       string                          `json:"displayName"`
	Level             int                             `json:"level"`
	Exp               int                             `json:"exp"`
	Voc               string                          `json:"voc,omitempty"`
	AGI               int                             `json:"AGI,omitempty"`
	STR               int                             `json:"STR,omitempty"`
	INT               int                             `json:"INT,omitempty"`
	CON               int                             `json:"CON,omitempty"`
	LCK               int                             `json:"LCK,omitempty"`
	MapID             int                             `json:"mapId"`
	VisualRoleID      int                             `json:"visualRoleId"`
	PresetID          int                             `json:"presetId,omitempty"`
	SourceQuery       string                          `json:"sourceQuery,omitempty"`
	BattleSourceQuery string                          `json:"battleSourceQuery,omitempty"`
	Appearance        RoleAppearance                  `json:"appearance,omitempty"`
	Skills            []RoleSkill                     `json:"skills,omitempty"`
	FastPanel         []RoleFastPanelEntry            `json:"fastPanel,omitempty"`
	TownBuffs         []RoleTownBuff                  `json:"townBuffs,omitempty"`
	Currencies        RoleCurrencies                  `json:"currencies,omitempty"`
	Items             []RoleItem                      `json:"items,omitempty"`
	RoleState         *RoleState                      `json:"-"`
	RolePhysique      *RolePhysique                   `json:"-"`
	DungeonInstances  map[string]DungeonInstanceState `json:"-"`
}

type DungeonInstanceState struct {
	CreatedAtUnix                 int64    `json:"createdAtUnix"`
	DefeatedVisibleMonsterHandles []string `json:"defeatedVisibleMonsterHandles,omitempty"`
}

type RoleListRequest struct {
	PlayerID     string `json:"playerId"`
	SessionToken string `json:"sessionToken"`
}

type RoleListResponse struct {
	Success      bool          `json:"success"`
	ErrorCode    string        `json:"errorCode,omitempty"`
	ErrorMessage string        `json:"errorMessage,omitempty"`
	Roles        []RoleSummary `json:"roles"`
}

type RoleCreateRequest struct {
	PlayerID       string         `json:"playerId"`
	SessionToken   string         `json:"sessionToken,omitempty"`
	DisplayName    string         `json:"displayName"`
	Gender         string         `json:"gender"`
	RoleTemplateID int            `json:"roleTemplateId"`
	PresetID       int            `json:"presetId,omitempty"`
	SourceQuery    string         `json:"sourceQuery,omitempty"`
	Appearance     RoleAppearance `json:"appearance,omitempty"`
}

type RoleCreateResponse struct {
	Success      bool        `json:"success"`
	ErrorCode    string      `json:"errorCode,omitempty"`
	ErrorMessage string      `json:"errorMessage,omitempty"`
	Role         RoleSummary `json:"role"`
}

type RoleSelectRequest struct {
	PlayerID     string `json:"playerId"`
	SessionToken string `json:"sessionToken,omitempty"`
	RoleID       string `json:"roleId"`
}

type PlayerBaseData struct {
	PlayerID          string         `json:"playerId"`
	RoleID            string         `json:"roleId"`
	DisplayName       string         `json:"displayName"`
	Level             int            `json:"level"`
	Exp               int            `json:"exp"`
	Voc               string         `json:"voc,omitempty"`
	HP                int            `json:"hp,omitempty"`
	MP                int            `json:"mp,omitempty"`
	MaxHP             int            `json:"maxHp,omitempty"`
	MaxMP             int            `json:"maxMp,omitempty"`
	MapID             int            `json:"mapId"`
	VisualRoleID      int            `json:"visualRoleId"`
	PresetID          int            `json:"presetId,omitempty"`
	SourceQuery       string         `json:"sourceQuery,omitempty"`
	BattleSourceQuery string         `json:"battleSourceQuery,omitempty"`
	Appearance        RoleAppearance `json:"appearance,omitempty"`
	Currencies        RoleCurrencies `json:"currencies,omitempty"`
	RoleState         *RoleState     `json:"roleState,omitempty"`
	RolePhysique      *RolePhysique  `json:"rolePhysique,omitempty"`
	PK                int            `json:"pk,omitempty"`
	State             int            `json:"state,omitempty"`
	GuildName         string         `json:"guildName,omitempty"`
	GuildPic          string         `json:"guildPic,omitempty"`
}

type RoleState struct {
	Handle string `json:"handle,omitempty"`
	HP     int    `json:"hp"`
	MP     int    `json:"mp"`
	Exp    int    `json:"exp"`
	Lv     int    `json:"lv"`
	Speed  int    `json:"speed"`
	OutG   int    `json:"outG,omitempty"`
	InG    int    `json:"inG,omitempty"`
}

type RolePhysique struct {
	Handle    string   `json:"handle,omitempty"`
	ResPros   []string `json:"resPros,omitempty"`
	AGI       int      `json:"AGI"`
	STR       int      `json:"STR"`
	INT       int      `json:"INT"`
	CON       int      `json:"CON"`
	LCK       int      `json:"LCK"`
	MaxHP     int      `json:"maxHp"`
	MaxMP     int      `json:"maxMp"`
	PhyAtk    int      `json:"phyAtk"`
	MgcAtk    int      `json:"mgcAtk"`
	PhyDef    int      `json:"phyDef"`
	MgcDef    int      `json:"mgcDef"`
	Hit       int      `json:"hit"`
	Dog       int      `json:"dog"`
	Fat       int      `json:"fat"`
	LastPoint int      `json:"lastPoint"`
}

type RoleSelectResponse struct {
	Success      bool           `json:"success"`
	ErrorCode    string         `json:"errorCode,omitempty"`
	ErrorMessage string         `json:"errorMessage,omitempty"`
	Role         RoleSummary    `json:"role"`
	PlayerBase   PlayerBaseData `json:"playerBase"`
}

type RoleRemoveRequest struct {
	PlayerID     string `json:"playerId"`
	SessionToken string `json:"sessionToken,omitempty"`
	RoleID       string `json:"roleId"`
	Password     string `json:"password"`
}

type RoleRemoveResponse struct {
	RemovedRoleID string `json:"removedRoleId,omitempty"`
	Success       bool   `json:"success"`
	Message       string `json:"message,omitempty"`
	ErrorCode     string `json:"errorCode,omitempty"`
	ErrorMessage  string `json:"errorMessage,omitempty"`
}

type RoleSkillPurchaseResult struct {
	Skills       []RoleSkill
	SkillCap     int
	Currencies   RoleCurrencies
	Found        bool
	Learned      bool
	ErrorCode    string
	ErrorMessage string
}

type RoleSkillRemoveResult struct {
	Skills       []RoleSkill
	SkillCap     int
	RemovedSkill *RoleSkill
	Found        bool
	Removed      bool
	ErrorCode    string
	ErrorMessage string
}

type RoleItemRequirement struct {
	Name  string
	Count int
}

type RoleItemPurchaseResult struct {
	Role         RoleSummary
	PlayerBase   PlayerBaseData
	Item         RoleItem
	Consumed     []RoleItem
	ClearedItems []RoleItemClear
	Currencies   RoleCurrencies
	Found        bool
	Purchased    bool
	ErrorCode    string
	ErrorMessage string
}

type RoleItemSaleResult struct {
	Role         RoleSummary
	PlayerBase   PlayerBaseData
	Item         RoleItem
	UpdatedItem  *RoleItem
	ClearedItems []RoleItemClear
	Currencies   RoleCurrencies
	Found        bool
	Sold         bool
	Amount       int
	ErrorCode    string
	ErrorMessage string
}

type RoleQuestCompleteResult struct {
	Role         RoleSummary
	PlayerBase   PlayerBaseData
	UpdatedItems []RoleItem
	ClearedItems []RoleItemClear
	Currencies   RoleCurrencies
	Found        bool
	Completed    bool
	ErrorCode    string
	ErrorMessage string
}

type RoleItemClear struct {
	Type  string
	Index int
}

type RoleEquipItemResult struct {
	Role         RoleSummary
	PlayerBase   PlayerBaseData
	EquippedItem RoleItem
	UpdatedItems []RoleItem
	ClearedItems []RoleItemClear
	Found        bool
	Equipped     bool
	ErrorCode    string
	ErrorMessage string
}

type RoleTryEquipPreviewResult struct {
	Role         RoleSummary
	PlayerBase   PlayerBaseData
	Item         RoleItem
	SourceQuery  string
	Found        bool
	Previewed    bool
	ErrorCode    string
	ErrorMessage string
}

type RoleMoveItemResult struct {
	Role         RoleSummary
	PlayerBase   PlayerBaseData
	UpdatedItems []RoleItem
	ClearedItems []RoleItemClear
	Found        bool
	Moved        bool
	ErrorCode    string
	ErrorMessage string
}

type RoleFinishContainerResult struct {
	Role         RoleSummary
	PlayerBase   PlayerBaseData
	UpdatedItems []RoleItem
	ClearedItems []RoleItemClear
	Found        bool
	Changed      bool
	ErrorCode    string
	ErrorMessage string
}

type RoleUseItemResult struct {
	Role              RoleSummary
	PlayerBase        PlayerBaseData
	Item              RoleItem
	LearnedSkill      *RoleSkill
	TownBuff          *RoleTownBuff
	UpdatedItem       *RoleItem
	UpdatedItems      []RoleItem
	ClearedItems      []RoleItemClear
	Currencies        RoleCurrencies
	ContainerType     string
	ContainerCapacity int
	Found             bool
	Used              bool
	Equipped          bool
	RoleStateChanged  bool
	ErrorCode         string
	ErrorMessage      string
}

type RoleTownBuffRemoveResult struct {
	Role       RoleSummary
	PlayerBase PlayerBaseData
	Buff       RoleTownBuff
	Found      bool
	Removed    bool
}

type RoleTownBuffsRemoveResult struct {
	Role       RoleSummary
	PlayerBase PlayerBaseData
	Buffs      []RoleTownBuff
	Found      bool
	Removed    bool
}

type RolePetInfoResult struct {
	Role         RoleSummary
	PlayerBase   PlayerBaseData
	Item         RoleItem
	Found        bool
	HasPet       bool
	Level        int
	Exp          int
	Fullness     int
	Name         string
	PetType      string
	DisplayURL   string
	SourceX      int
	SourceY      int
	SkillHTML    string
	PetID        string
	Status       string
	ErrorCode    string
	ErrorMessage string
}

type RolePetFeedResult struct {
	RolePetInfoResult
	FeedItem     RoleItem
	UpdatedItem  *RoleItem
	ClearedItems []RoleItemClear
	Fed          bool
}

type RoleTownHealResult struct {
	Role         RoleSummary
	PlayerBase   PlayerBaseData
	RoleState    RoleState
	UpdatedItems []RoleItem
	ClearedItems []RoleItemClear
	Currencies   RoleCurrencies
	Found        bool
	Healed       bool
	NearlyFull   bool
	Cost         int
	ErrorCode    string
	ErrorMessage string
}

type RoleExpGrantResult struct {
	Role         RoleSummary
	PlayerBase   PlayerBaseData
	RoleState    RoleState
	RolePhysique RolePhysique
	Found        bool
	Granted      bool
	LevelChanged bool
}

type RolePointStats struct {
	AGI *int
	STR *int
	INT *int
	CON *int
	LCK *int
}

type RoleVocationResult struct {
	Role         RoleSummary
	PlayerBase   PlayerBaseData
	RoleState    RoleState
	RolePhysique RolePhysique
	Found        bool
	Changed      bool
	ErrorCode    string
	ErrorMessage string
}

type RoleAddPointResult struct {
	Role         RoleSummary
	PlayerBase   PlayerBaseData
	RolePhysique RolePhysique
	Found        bool
	Applied      bool
	ErrorCode    string
	ErrorMessage string
}

type Store struct {
	mu                   sync.Mutex
	rolesByPID           map[string][]RoleSummary
	teamDungeonInstances map[string]map[string]DungeonInstanceState
	nextRoleSeqByPID     map[string]int
	accountsByName       map[string]AccountRecord
	acceptedQuests       map[string]map[string]bool
	removedQuests        map[string]map[string]bool
	rolePersistenceLocks map[string]*sync.Mutex
	db                   *sql.DB
	now                  func() time.Time
	Guilds               *guild.Service
	Mall                 *mall.Service
	mallRequests         map[string]mall.PurchaseResult
}

type DevRoleSummary struct {
	PlayerID    string `json:"playerId"`
	RoleID      string `json:"roleId"`
	DisplayName string `json:"displayName"`
}

type AccountRecord struct {
	UserName     string
	Password     string
	PlayerID     string
	DisplayName  string
	SessionToken string
}

func NewStore() *Store {
	return &Store{
		rolesByPID:           make(map[string][]RoleSummary),
		teamDungeonInstances: make(map[string]map[string]DungeonInstanceState),
		nextRoleSeqByPID:     make(map[string]int),
		accountsByName: map[string]AccountRecord{
			"mockuser": {
				UserName:     "mockuser",
				Password:     "magicpwd",
				PlayerID:     "mock-player-001",
				DisplayName:  "Mock Swordswoman",
				SessionToken: "mock-session-token-001",
			},
		},
		now:                  time.Now,
		Guilds:               guild.NewMemoryService(),
		Mall:                 mall.NewService(),
		mallRequests:         make(map[string]mall.PurchaseResult),
		acceptedQuests:       make(map[string]map[string]bool),
		removedQuests:        make(map[string]map[string]bool),
		rolePersistenceLocks: make(map[string]*sync.Mutex),
	}
}

func (store *Store) rolePersistenceLock(playerID string, roleID string) *sync.Mutex {
	key := playerID + "\x00" + roleID
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.rolePersistenceLocks == nil {
		store.rolePersistenceLocks = make(map[string]*sync.Mutex)
	}
	lock := store.rolePersistenceLocks[key]
	if lock == nil {
		lock = &sync.Mutex{}
		store.rolePersistenceLocks[key] = lock
	}
	return lock
}

func (store *Store) DevRoleSummaries() []DevRoleSummary {
	store.mu.Lock()
	defer store.mu.Unlock()

	result := []DevRoleSummary{}
	for playerID, roles := range store.rolesByPID {
		for _, role := range roles {
			role = withRoleRuntimeDefaults(role)
			result = append(result, DevRoleSummary{
				PlayerID:    playerID,
				RoleID:      role.RoleID,
				DisplayName: role.DisplayName,
			})
		}
	}
	sort.SliceStable(result, func(left int, right int) bool {
		if result[left].PlayerID == result[right].PlayerID {
			return result[left].RoleID < result[right].RoleID
		}
		return result[left].PlayerID < result[right].PlayerID
	})
	return result
}

func NewPersistentStore(persistencePath string) (*Store, error) {
	store := NewStore()
	if persistencePath == "" {
		return store, nil
	}

	if err := os.MkdirAll(filepath.Dir(persistencePath), 0o755); err != nil {
		return nil, fmt.Errorf("create persistence directory: %w", err)
	}

	db, err := sql.Open("sqlite", persistencePath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := configureSQLitePersistence(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	store.db = db
	guildService, err := guild.NewService(db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize guild store: %w", err)
	}
	store.Guilds = guildService

	if err := store.initSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := store.loadFromDB(); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := store.saveLocked(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (store *Store) Login(request LoginRequest) LoginResponse {
	if request.UserName != "" || request.Password != "" {
		return store.loginByAccount(request.UserName, request.Password)
	}

	platform := request.Platform
	if platform == "" {
		platform = "guest"
	}

	playerID := fmt.Sprintf("%s-player-local", platform)
	return LoginResponse{
		PlayerID:     playerID,
		SessionToken: fmt.Sprintf("local-session-%s", playerID),
		DisplayName:  "本地女侠",
		Success:      true,
	}
}

func (store *Store) loginByAccount(userName string, password string) LoginResponse {
	store.mu.Lock()
	defer store.mu.Unlock()

	if validateUserName(userName) != "success" {
		return LoginResponse{
			Success:      false,
			ErrorCode:    "8",
			ErrorMessage: "用户名不合法。",
		}
	}

	account, ok := store.accountsByName[userName]
	if !ok {
		account = store.createAccountLocked(userName, password)
		if err := store.persistAccountLocked(account); err != nil {
			log.Printf("[session.Store] persist auto registered account failed: %v", err)
		}
		log.Printf("[session.Store] auto registered account userName=%s playerId=%s", account.UserName, account.PlayerID)
		return LoginResponse{
			PlayerID:     account.PlayerID,
			SessionToken: account.SessionToken,
			DisplayName:  account.DisplayName,
			Success:      true,
		}
	}

	if account.Password != password {
		return LoginResponse{
			Success:      false,
			ErrorCode:    "3",
			ErrorMessage: "密码错误!",
		}
	}

	return LoginResponse{
		PlayerID:     account.PlayerID,
		SessionToken: account.SessionToken,
		DisplayName:  account.DisplayName,
		Success:      true,
	}
}

func (store *Store) createAccountLocked(userName string, password string) AccountRecord {
	account := AccountRecord{
		UserName:     userName,
		Password:     password,
		PlayerID:     fmt.Sprintf("acct-%s", userName),
		DisplayName:  userName,
		SessionToken: fmt.Sprintf("session-%s", userName),
	}
	store.accountsByName[userName] = account
	return account
}

func (store *Store) ListRoles(request RoleListRequest) RoleListResponse {
	store.mu.Lock()
	defer store.mu.Unlock()

	validationFailure, ok := store.validateRoleAccessLocked(request.PlayerID, request.SessionToken)
	if !ok {
		return RoleListResponse{
			Success:      false,
			ErrorCode:    validationFailure.ErrorCode,
			ErrorMessage: validationFailure.ErrorMessage,
			Roles:        []RoleSummary{},
		}
	}

	normalized := store.normalizeRoleIDsLocked(request.PlayerID)
	if normalized {
		if err := store.persistPlayerStateLocked(request.PlayerID); err != nil {
			log.Printf("[session.Store] persist normalized role ids failed: %v", err)
		}
	}
	roles := append(make([]RoleSummary, 0), store.rolesByPID[request.PlayerID]...)
	for index := range roles {
		roles[index] = withRoleRuntimeDefaults(roles[index])
	}
	log.Printf("[session.Store] ListRoles playerId=%s roles=%s", request.PlayerID, formatRoleSummaries(roles))
	return RoleListResponse{
		Success: true,
		Roles:   roles,
	}
}

func (store *Store) CreateRole(request RoleCreateRequest) RoleCreateResponse {
	store.mu.Lock()
	defer store.mu.Unlock()

	validationFailure, ok := store.validateRoleAccessLocked(request.PlayerID, request.SessionToken)
	if !ok {
		return RoleCreateResponse{
			Success:      false,
			ErrorCode:    validationFailure.ErrorCode,
			ErrorMessage: validationFailure.ErrorMessage,
			Role:         emptyRoleSummary(),
		}
	}

	store.normalizeRoleIDsLocked(request.PlayerID)
	displayName := request.DisplayName
	if displayName == "" {
		displayName = "云隐女侠"
	}

	roles := store.rolesByPID[request.PlayerID]
	store.nextRoleSeqByPID[request.PlayerID] += 1
	roleSeq := store.nextRoleSeqByPID[request.PlayerID]
	role := RoleSummary{
		RoleID:       fmt.Sprintf("%s-role-%03d", request.PlayerID, roleSeq),
		DisplayName:  displayName,
		Level:        1,
		Voc:          defaultRoleVoc,
		MapID:        1,
		VisualRoleID: resolveVisualRoleID(request.PresetID),
		PresetID:     request.PresetID,
		SourceQuery:  request.SourceQuery,
		Appearance:   request.Appearance,
		Currencies:   defaultRoleCurrencies(),
	}
	store.rolesByPID[request.PlayerID] = append(roles, role)
	if err := store.persistPlayerStateLocked(request.PlayerID); err != nil {
		log.Printf("[session.Store] persist created role failed: %v", err)
	}

	return RoleCreateResponse{
		Success: true,
		Role:    role,
	}
}

func (store *Store) SelectRole(request RoleSelectRequest) RoleSelectResponse {
	store.mu.Lock()
	defer store.mu.Unlock()

	validationFailure, ok := store.validateRoleAccessLocked(request.PlayerID, request.SessionToken)
	if !ok {
		return RoleSelectResponse{
			Success:      false,
			ErrorCode:    validationFailure.ErrorCode,
			ErrorMessage: validationFailure.ErrorMessage,
			Role:         emptyRoleSummary(),
			PlayerBase:   emptyPlayerBaseData(request.PlayerID),
		}
	}

	normalized := store.normalizeRoleIDsLocked(request.PlayerID)
	if normalized {
		if err := store.persistPlayerStateLocked(request.PlayerID); err != nil {
			log.Printf("[session.Store] persist normalized role ids failed: %v", err)
		}
	}
	roles := store.rolesByPID[request.PlayerID]
	for index := range roles {
		if roles[index].RoleID == request.RoleID {
			if pruneExpiredDungeonInstances(&roles[index], store.now()) {
				store.rolesByPID[request.PlayerID] = roles
				if err := store.persistPlayerStateLocked(request.PlayerID); err != nil {
					log.Printf("[session.Store] persist expired dungeon instances failed: %v", err)
				}
			}
			role := withRoleRuntimeDefaults(roles[index])
			return RoleSelectResponse{
				Success:    true,
				Role:       role,
				PlayerBase: playerBaseDataFromRole(request.PlayerID, role),
			}
		}
	}

	return RoleSelectResponse{
		Success:      false,
		ErrorCode:    "5",
		ErrorMessage: "角色不存在。",
		Role:         emptyRoleSummary(),
		PlayerBase:   emptyPlayerBaseData(request.PlayerID),
	}
}

func (store *Store) UpdateRoleMap(playerID string, roleID string, mapID int) (RoleSummary, PlayerBaseData, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

	if mapID <= 0 {
		return RoleSummary{}, emptyPlayerBaseData(playerID), false
	}

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		roles[index].MapID = mapID
		store.rolesByPID[playerID] = roles
		if err := store.persistPlayerStateLocked(playerID); err != nil {
			log.Printf("[session.Store] persist role map failed: %v", err)
		}

		role := withRoleRuntimeDefaults(roles[index])
		return role, playerBaseDataFromRole(playerID, role), true
	}

	return RoleSummary{}, emptyPlayerBaseData(playerID), false
}

func (store *Store) EnsureRoleDungeonInstance(playerID string, roleID string, key string) (DungeonInstanceState, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

	key = strings.TrimSpace(key)
	if key == "" {
		return DungeonInstanceState{}, false
	}

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		now := store.now()
		if pruneExpiredDungeonInstances(&roles[index], now) {
			store.rolesByPID[playerID] = roles
		}
		if roles[index].DungeonInstances == nil {
			roles[index].DungeonInstances = map[string]DungeonInstanceState{}
		}
		state, ok := roles[index].DungeonInstances[key]
		if !ok {
			state = DungeonInstanceState{CreatedAtUnix: now.Unix()}
			roles[index].DungeonInstances[key] = state
		}
		store.rolesByPID[playerID] = roles
		if err := store.persistPlayerStateLocked(playerID); err != nil {
			log.Printf("[session.Store] persist dungeon instance failed: %v", err)
		}
		return cloneDungeonInstanceState(state), true
	}

	return DungeonInstanceState{}, false
}

func (store *Store) EnsureTeamDungeonInstance(teamID string, key string) (DungeonInstanceState, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

	teamID = strings.TrimSpace(teamID)
	key = strings.TrimSpace(key)
	if teamID == "" || key == "" {
		return DungeonInstanceState{}, false
	}
	instances := store.teamDungeonInstances[teamID]
	if instances == nil {
		instances = map[string]DungeonInstanceState{}
		store.teamDungeonInstances[teamID] = instances
	}
	pruneExpiredDungeonInstanceStates(instances, store.now())
	state, ok := instances[key]
	if !ok {
		state = DungeonInstanceState{CreatedAtUnix: store.now().Unix()}
		instances[key] = state
	}
	return cloneDungeonInstanceState(state), true
}

func (store *Store) GetTeamDungeonInstance(teamID string, key string) (DungeonInstanceState, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

	teamID = strings.TrimSpace(teamID)
	key = strings.TrimSpace(key)
	instances := store.teamDungeonInstances[teamID]
	if teamID == "" || key == "" || instances == nil {
		return DungeonInstanceState{}, false
	}
	pruneExpiredDungeonInstanceStates(instances, store.now())
	state, ok := instances[key]
	if !ok || state.CreatedAtUnix <= 0 {
		return DungeonInstanceState{}, false
	}
	return cloneDungeonInstanceState(state), true
}

func (store *Store) ResetTeamDungeonInstance(teamID string, key string) bool {
	store.mu.Lock()
	defer store.mu.Unlock()

	teamID = strings.TrimSpace(teamID)
	key = strings.TrimSpace(key)
	instances := store.teamDungeonInstances[teamID]
	if teamID == "" || key == "" || instances == nil {
		return false
	}
	delete(instances, key)
	if len(instances) == 0 {
		delete(store.teamDungeonInstances, teamID)
	}
	return true
}

func (store *Store) MarkTeamDungeonVisibleMonsterDefeated(teamID string, key string, handle string) (DungeonInstanceState, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

	teamID = strings.TrimSpace(teamID)
	key = strings.TrimSpace(key)
	handle = strings.TrimSpace(handle)
	if teamID == "" || key == "" || handle == "" {
		return DungeonInstanceState{}, false
	}
	instances := store.teamDungeonInstances[teamID]
	if instances == nil {
		instances = map[string]DungeonInstanceState{}
		store.teamDungeonInstances[teamID] = instances
	}
	pruneExpiredDungeonInstanceStates(instances, store.now())
	state := instances[key]
	if state.CreatedAtUnix == 0 {
		state.CreatedAtUnix = store.now().Unix()
	}
	if !containsString(state.DefeatedVisibleMonsterHandles, handle) {
		state.DefeatedVisibleMonsterHandles = append(state.DefeatedVisibleMonsterHandles, handle)
	}
	instances[key] = state
	return cloneDungeonInstanceState(state), true
}

func (store *Store) GetRoleDungeonInstance(playerID string, roleID string, key string) (DungeonInstanceState, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

	key = strings.TrimSpace(key)
	if key == "" {
		return DungeonInstanceState{}, false
	}

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		if pruneExpiredDungeonInstances(&roles[index], store.now()) {
			store.rolesByPID[playerID] = roles
			if err := store.persistPlayerStateLocked(playerID); err != nil {
				log.Printf("[session.Store] persist expired dungeon instances failed: %v", err)
			}
		}
		state, ok := roles[index].DungeonInstances[key]
		if !ok || state.CreatedAtUnix <= 0 {
			return DungeonInstanceState{}, false
		}
		return cloneDungeonInstanceState(state), true
	}

	return DungeonInstanceState{}, false
}

func (store *Store) ResetRoleDungeonInstance(playerID string, roleID string, key string) bool {
	store.mu.Lock()
	defer store.mu.Unlock()

	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}
		if len(roles[index].DungeonInstances) > 0 {
			delete(roles[index].DungeonInstances, key)
		}
		store.rolesByPID[playerID] = roles
		if err := store.persistPlayerStateLocked(playerID); err != nil {
			log.Printf("[session.Store] persist reset dungeon instance failed: %v", err)
		}
		return true
	}

	return false
}

func (store *Store) MarkRoleDungeonVisibleMonsterDefeated(playerID string, roleID string, key string, handle string) (DungeonInstanceState, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

	key = strings.TrimSpace(key)
	handle = strings.TrimSpace(handle)
	if key == "" || handle == "" {
		return DungeonInstanceState{}, false
	}

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		now := store.now()
		pruneExpiredDungeonInstances(&roles[index], now)
		if roles[index].DungeonInstances == nil {
			roles[index].DungeonInstances = map[string]DungeonInstanceState{}
		}
		state := roles[index].DungeonInstances[key]
		if state.CreatedAtUnix == 0 {
			state.CreatedAtUnix = now.Unix()
		}
		if !containsString(state.DefeatedVisibleMonsterHandles, handle) {
			state.DefeatedVisibleMonsterHandles = append(state.DefeatedVisibleMonsterHandles, handle)
		}
		roles[index].DungeonInstances[key] = state
		store.rolesByPID[playerID] = roles
		if err := store.persistPlayerStateLocked(playerID); err != nil {
			log.Printf("[session.Store] persist defeated dungeon monster failed: %v", err)
		}
		return cloneDungeonInstanceState(state), true
	}

	return DungeonInstanceState{}, false
}

func pruneExpiredDungeonInstances(role *RoleSummary, now time.Time) bool {
	if role == nil || len(role.DungeonInstances) == 0 {
		return false
	}
	changed := false
	for key, state := range role.DungeonInstances {
		if state.CreatedAtUnix <= 0 || !now.Before(time.Unix(state.CreatedAtUnix, 0).Add(dungeonInstanceTTL)) {
			delete(role.DungeonInstances, key)
			changed = true
		}
	}
	if len(role.DungeonInstances) == 0 {
		role.DungeonInstances = nil
	}
	return changed
}

func pruneExpiredDungeonInstanceStates(instances map[string]DungeonInstanceState, now time.Time) {
	for key, state := range instances {
		if state.CreatedAtUnix <= 0 || !now.Before(time.Unix(state.CreatedAtUnix, 0).Add(dungeonInstanceTTL)) {
			delete(instances, key)
		}
	}
}

func (store *Store) GetRoleSkills(playerID string, roleID string) ([]RoleSkill, int, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

	for _, role := range store.rolesByPID[playerID] {
		if role.RoleID != roleID {
			continue
		}

		role = withRoleRuntimeDefaults(role)
		return cloneRoleSkills(role.Skills), defaultSkillCap, true
	}

	return nil, defaultSkillCap, false
}

func (store *Store) GetRoleFastPanel(playerID string, roleID string) ([]RoleFastPanelEntry, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

	for _, role := range store.rolesByPID[playerID] {
		if role.RoleID != roleID {
			continue
		}

		role = withRoleRuntimeDefaults(role)
		if isCapturedWoodcutter333LocalRole(role) {
			return cloneRoleFastPanel(filterRoleFastPanelEntries(role.FastPanel, capturedWoodcutter333ShortcutSkills())), true
		}
		return cloneRoleFastPanel(filterRoleFastPanelEntries(role.FastPanel, role.Skills)), true
	}

	return []RoleFastPanelEntry{}, false
}

func (store *Store) GetRoleTownBuffs(playerID string, roleID string) ([]RoleTownBuff, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		roles[index] = withRoleRuntimeDefaults(roles[index])
		filtered, changed := filterActiveRoleTownBuffs(roles[index].TownBuffs, time.Now().UnixMilli())
		if changed {
			roles[index].TownBuffs = filtered
			store.rolesByPID[playerID] = roles
			if err := store.persistPlayerStateLocked(playerID); err != nil {
				log.Printf("[session.Store] persist pruned town buffs failed: %v", err)
			}
		}
		return cloneRoleTownBuffs(filtered), true
	}

	return []RoleTownBuff{}, false
}

func (store *Store) RemoveRoleTownBuff(playerID string, roleID string, name string) RoleTownBuffRemoveResult {
	store.mu.Lock()
	defer store.mu.Unlock()

	name = strings.TrimSpace(name)
	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		roles[index] = withRoleRuntimeDefaults(roles[index])
		activeBuffs, _ := filterActiveRoleTownBuffs(roles[index].TownBuffs, time.Now().UnixMilli())
		updated := make([]RoleTownBuff, 0, len(activeBuffs))
		var removed RoleTownBuff
		for _, buff := range activeBuffs {
			if buff.Name == name && removed.Name == "" {
				removed = buff
				continue
			}
			updated = append(updated, buff)
		}
		roles[index].TownBuffs = normalizeRoleTownBuffs(updated)
		store.rolesByPID[playerID] = roles
		if removed.Name != "" {
			if err := store.persistPlayerStateLocked(playerID); err != nil {
				log.Printf("[session.Store] persist removed town buff failed: %v", err)
			}
		}
		role := withRoleRuntimeDefaults(roles[index])
		return RoleTownBuffRemoveResult{
			Role:       role,
			PlayerBase: playerBaseDataFromRole(playerID, role),
			Buff:       removed,
			Found:      true,
			Removed:    removed.Name != "",
		}
	}

	return RoleTownBuffRemoveResult{Found: false}
}

func (store *Store) RemoveExpiredRoleTownBuffs(playerID string, roleID string, nowMs int64) RoleTownBuffsRemoveResult {
	store.mu.Lock()
	defer store.mu.Unlock()

	if nowMs <= 0 {
		nowMs = time.Now().UnixMilli()
	}
	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		roles[index] = withRoleRuntimeDefaults(roles[index])
		normalized := normalizeRoleTownBuffs(roles[index].TownBuffs)
		updated := make([]RoleTownBuff, 0, len(normalized))
		removed := make([]RoleTownBuff, 0, len(normalized))
		for _, buff := range normalized {
			if buff.EndTime > 0 && buff.EndTime <= nowMs {
				if buff.SourceCapture == "" {
					buff.SourceCapture = classicTownRemoveAbateBuffCapture
				}
				removed = append(removed, buff)
				continue
			}
			updated = append(updated, buff)
		}
		roles[index].TownBuffs = normalizeRoleTownBuffs(updated)
		store.rolesByPID[playerID] = roles
		if len(removed) > 0 {
			if err := store.persistPlayerStateLocked(playerID); err != nil {
				log.Printf("[session.Store] persist expired town buffs failed: %v", err)
			}
		}
		role := withRoleRuntimeDefaults(roles[index])
		return RoleTownBuffsRemoveResult{
			Role:       role,
			PlayerBase: playerBaseDataFromRole(playerID, role),
			Buffs:      cloneRoleTownBuffs(removed),
			Found:      true,
			Removed:    len(removed) > 0,
		}
	}

	return RoleTownBuffsRemoveResult{Found: false}
}

func (store *Store) SetRoleFastPanelEntry(playerID string, roleID string, entry RoleFastPanelEntry) ([]RoleFastPanelEntry, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

	entry = normalizeRoleFastPanelEntry(entry)
	if entry.Index < 0 || entry.Index >= defaultRoleFastPanelSlotCount || entry.Type == "" || entry.Name == "" {
		return []RoleFastPanelEntry{}, false
	}

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		roles[index] = withRoleRuntimeDefaults(roles[index])
		if entry.Type == "item" && !roleHasBagItemNamed(roles[index].Items, entry.Name) {
			return cloneRoleFastPanel(filterRoleFastPanelEntries(roles[index].FastPanel, roles[index].Skills)), true
		}
		if !canSetRoleFastPanelEntry(entry, roles[index].Skills) {
			return cloneRoleFastPanel(filterRoleFastPanelEntries(roles[index].FastPanel, roles[index].Skills)), true
		}

		updated := make([]RoleFastPanelEntry, 0, len(roles[index].FastPanel)+1)
		for _, existing := range roles[index].FastPanel {
			if existing.Index == entry.Index {
				continue
			}
			updated = append(updated, existing)
		}
		updated = append(updated, entry)
		roles[index].FastPanel = filterRoleFastPanelEntries(normalizeRoleFastPanel(updated), roles[index].Skills)
		store.rolesByPID[playerID] = roles
		if err := store.persistPlayerStateLocked(playerID); err != nil {
			log.Printf("[session.Store] persist fast panel failed: %v", err)
		}
		return cloneRoleFastPanel(roles[index].FastPanel), true
	}

	return []RoleFastPanelEntry{}, false
}

func (store *Store) RemoveRoleFastPanelEntry(playerID string, roleID string, slotIndex int) ([]RoleFastPanelEntry, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

	if slotIndex < 0 || slotIndex >= defaultRoleFastPanelSlotCount {
		return []RoleFastPanelEntry{}, false
	}

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		roles[index] = withRoleRuntimeDefaults(roles[index])
		updated := make([]RoleFastPanelEntry, 0, len(roles[index].FastPanel))
		removed := false
		for _, existing := range roles[index].FastPanel {
			if existing.Index == slotIndex {
				removed = true
				continue
			}
			updated = append(updated, existing)
		}
		roles[index].FastPanel = filterRoleFastPanelEntries(normalizeRoleFastPanel(updated), roles[index].Skills)
		store.rolesByPID[playerID] = roles
		if removed {
			if err := store.persistPlayerStateLocked(playerID); err != nil {
				log.Printf("[session.Store] persist removed fast panel failed: %v", err)
			}
		}
		return cloneRoleFastPanel(roles[index].FastPanel), true
	}

	return []RoleFastPanelEntry{}, false
}

func roleHasBagItemNamed(items []RoleItem, name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	for _, item := range items {
		item = normalizeRoleItem(item)
		if item.Type == "背包" && item.Name == name && item.Count > 0 {
			return true
		}
	}
	return false
}

func filterActiveRoleTownBuffs(buffs []RoleTownBuff, nowMs int64) ([]RoleTownBuff, bool) {
	normalized := normalizeRoleTownBuffs(buffs)
	result := make([]RoleTownBuff, 0, len(normalized))
	changed := len(normalized) != len(buffs)
	for _, buff := range normalized {
		if buff.EndTime > 0 && buff.EndTime <= nowMs {
			changed = true
			continue
		}
		result = append(result, buff)
	}
	return result, changed || len(result) != len(normalized)
}

func (store *Store) GetRoleCurrencies(playerID string, roleID string) (RoleCurrencies, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

	for _, role := range store.rolesByPID[playerID] {
		if role.RoleID == roleID {
			return cloneRoleCurrencies(withRoleRuntimeDefaults(role).Currencies), true
		}
	}

	return RoleCurrencies{}, false
}

func (store *Store) AddRoleCurrency(playerID string, roleID string, currencyName string, amount int) (RoleCurrencies, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

	currencyName = strings.TrimSpace(currencyName)
	if currencyName == "" || amount <= 0 {
		return RoleCurrencies{}, false
	}
	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		roles[index] = withRoleRuntimeDefaults(roles[index])
		currencies := cloneRoleCurrencies(roles[index].Currencies)
		currencies[currencyName] += amount
		roles[index].Currencies = normalizeRoleCurrencies(currencies)
		store.rolesByPID[playerID] = roles
		if err := store.persistPlayerStateLocked(playerID); err != nil {
			log.Printf("[session.Store] persist added currency failed: %v", err)
		}
		return cloneRoleCurrencies(roles[index].Currencies), true
	}

	return RoleCurrencies{}, false
}

func (store *Store) GetRoleContainerCapacity(playerID string, roleID string, containerType string) (int, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

	containerType = strings.TrimSpace(containerType)
	baseCapacity, supported := roleContainerCapacity(containerType)
	if !supported {
		return 0, false
	}
	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID == roleID {
			roles[index] = withRoleRuntimeDefaults(roles[index])
			return effectiveRoleContainerCapacity(roles[index].Items, containerType, baseCapacity), true
		}
	}

	return 0, false
}

func (store *Store) GetRoleItems(playerID string, roleID string, containerType string) ([]RoleItem, int, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

	containerType = strings.TrimSpace(containerType)
	baseCapacity, supported := roleContainerCapacity(containerType)
	if !supported {
		return []RoleItem{}, 0, false
	}
	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		roles[index] = withRoleRuntimeDefaults(roles[index])
		if containerType == "背包" {
			updatedItems, _, _, changed := trimRoleCurrencyItemsToBalance(roles[index].Items, roles[index].Currencies)
			if changed {
				roles[index].Items = normalizeRoleItems(updatedItems)
				store.rolesByPID[playerID] = roles
				if err := store.persistPlayerStateLocked(playerID); err != nil {
					log.Printf("[session.Store] persist trimmed currency items failed: %v", err)
				}
			}
		}
		items := make([]RoleItem, 0, len(roles[index].Items))
		for _, item := range roles[index].Items {
			if item.Type == containerType {
				items = append(items, item)
			}
		}
		capacity := effectiveRoleContainerCapacity(roles[index].Items, containerType, baseCapacity)
		return cloneRoleItems(items), capacity, true
	}

	return []RoleItem{}, 0, false
}

func (store *Store) GetRoleBagItemByName(playerID string, roleID string, name string) (RoleItem, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

	name = strings.TrimSpace(name)
	if name == "" {
		return RoleItem{}, false
	}
	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		for _, item := range roles[index].Items {
			if item.Type == "背包" && strings.TrimSpace(item.Name) == name && item.Count > 0 {
				return normalizeRoleItem(item), true
			}
		}
		return RoleItem{}, false
	}

	return RoleItem{}, false
}

func (store *Store) GetRoleRuntimeData(playerID string, roleID string) (RoleSummary, PlayerBaseData, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

	for _, role := range store.rolesByPID[playerID] {
		if role.RoleID != roleID {
			continue
		}

		role = withRoleRuntimeDefaults(role)
		return role, playerBaseDataFromRole(playerID, role), true
	}

	return RoleSummary{}, emptyPlayerBaseData(playerID), false
}

func (store *Store) RemovedQuestTitles(playerID string, roleID string) map[string]bool {
	store.mu.Lock()
	defer store.mu.Unlock()

	if !store.hasRoleLocked(playerID, roleID) {
		return map[string]bool{}
	}
	return cloneBoolMap(store.removedQuests[roleID])
}

func (store *Store) AcceptedQuestTitles(playerID string, roleID string) map[string]bool {
	store.mu.Lock()
	defer store.mu.Unlock()

	if !store.hasRoleLocked(playerID, roleID) {
		return map[string]bool{}
	}
	return cloneBoolMap(store.acceptedQuests[roleID])
}

func (store *Store) AcceptQuest(playerID string, roleID string, title string) bool {
	title = strings.TrimSpace(title)
	if title == "" {
		return false
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	if !store.hasRoleLocked(playerID, roleID) {
		return false
	}
	if store.acceptedQuests[roleID] == nil {
		store.acceptedQuests[roleID] = make(map[string]bool)
	}
	if store.acceptedQuests[roleID][title] {
		return false
	}
	store.acceptedQuests[roleID][title] = true
	if store.removedQuests[roleID] != nil {
		delete(store.removedQuests[roleID], title)
	}
	if err := store.persistAcceptedQuestLocked(playerID, roleID, title); err != nil {
		log.Printf("[session.Store] persist accepted quest failed roleId=%s title=%s: %v", roleID, title, err)
	}
	if err := store.deleteRemovedQuestLocked(roleID, title); err != nil {
		log.Printf("[session.Store] delete removed accepted quest failed roleId=%s title=%s: %v", roleID, title, err)
	}
	return true
}

func (store *Store) MarkQuestRemoved(playerID string, roleID string, title string) bool {
	title = strings.TrimSpace(title)
	if title == "" {
		return false
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	if !store.hasRoleLocked(playerID, roleID) {
		return false
	}
	if store.removedQuests[roleID] == nil {
		store.removedQuests[roleID] = make(map[string]bool)
	}
	if store.acceptedQuests[roleID] != nil {
		delete(store.acceptedQuests[roleID], title)
	}
	if store.removedQuests[roleID][title] {
		return false
	}
	store.removedQuests[roleID][title] = true
	if err := store.deleteAcceptedQuestLocked(roleID, title); err != nil {
		log.Printf("[session.Store] delete accepted removed quest failed roleId=%s title=%s: %v", roleID, title, err)
	}
	if err := store.persistRemovedQuestLocked(playerID, roleID, title); err != nil {
		log.Printf("[session.Store] persist removed quest failed roleId=%s title=%s: %v", roleID, title, err)
	}
	return true
}

func (store *Store) RestoreQuest(playerID string, roleID string, title string) bool {
	title = strings.TrimSpace(title)
	if title == "" {
		return false
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	if !store.hasRoleLocked(playerID, roleID) || store.removedQuests[roleID] == nil || !store.removedQuests[roleID][title] {
		return false
	}
	delete(store.removedQuests[roleID], title)
	if err := store.deleteRemovedQuestLocked(roleID, title); err != nil {
		log.Printf("[session.Store] delete removed quest failed roleId=%s title=%s: %v", roleID, title, err)
	}
	return true
}

func (store *Store) CompleteQuest(playerID string, roleID string, title string, requirements []RoleItemRequirement) RoleQuestCompleteResult {
	title = strings.TrimSpace(title)
	if title == "" {
		return RoleQuestCompleteResult{
			ErrorCode:    "invalid_quest",
			ErrorMessage: "任务不存在。",
		}
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}
		if store.acceptedQuests[roleID] == nil || !store.acceptedQuests[roleID][title] {
			role := withRoleRuntimeDefaults(roles[index])
			return RoleQuestCompleteResult{
				Role:         role,
				PlayerBase:   playerBaseDataFromRole(playerID, role),
				Currencies:   cloneRoleCurrencies(role.Currencies),
				Found:        true,
				ErrorCode:    "quest_not_accepted",
				ErrorMessage: "尚未接受该任务。",
			}
		}

		currentRole := withRoleRuntimeDefaults(roles[index])
		currentBase := playerBaseDataFromRole(playerID, currentRole)
		currentCurrencies := cloneRoleCurrencies(currentRole.Currencies)
		updatedItems := cloneRoleItems(currentRole.Items)
		updatedRequirementItems := []RoleItem{}
		clearedItems := []RoleItemClear{}
		var trimmedCurrencyItems []RoleItem
		var trimmedClearedItems []RoleItemClear
		var changed bool
		updatedItems, trimmedCurrencyItems, trimmedClearedItems, changed = trimRoleCurrencyItemsToBalance(updatedItems, currentCurrencies)
		updatedRequirementItems = append(updatedRequirementItems, trimmedCurrencyItems...)
		clearedItems = append(clearedItems, trimmedClearedItems...)

		normalizedRequirements := normalizeRoleItemRequirements(requirements)
		for _, requirement := range normalizedRequirements {
			if requirement.Count <= 0 {
				continue
			}
			if isRoleCurrencyName(requirement.Name) {
				if currentCurrencies[requirement.Name] < requirement.Count {
					return RoleQuestCompleteResult{
						Role:         currentRole,
						PlayerBase:   currentBase,
						Currencies:   currentCurrencies,
						Found:        true,
						ErrorCode:    "not_enough_currency",
						ErrorMessage: fmt.Sprintf("%s不足。", requirement.Name),
					}
				}
				continue
			}
			if totalRoleItemCountByName(updatedItems, "背包", requirement.Name) < requirement.Count {
				return RoleQuestCompleteResult{
					Role:         currentRole,
					PlayerBase:   currentBase,
					Currencies:   currentCurrencies,
					Found:        true,
					ErrorCode:    "item_not_enough",
					ErrorMessage: fmt.Sprintf("%s不足。", requirement.Name),
				}
			}
		}

		for _, requirement := range normalizedRequirements {
			if requirement.Count <= 0 {
				continue
			}
			if isRoleCurrencyName(requirement.Name) {
				currentCurrencies[requirement.Name] -= requirement.Count
				var updated []RoleItem
				var cleared []RoleItemClear
				updatedItems, updated, cleared = consumeRoleItemsByName(updatedItems, "背包", requirement.Name, requirement.Count)
				updatedRequirementItems = append(updatedRequirementItems, updated...)
				clearedItems = append(clearedItems, cleared...)
				changed = true
				continue
			}
			var updated []RoleItem
			var cleared []RoleItemClear
			updatedItems, updated, cleared = consumeRoleItemsByName(updatedItems, "背包", requirement.Name, requirement.Count)
			updatedRequirementItems = append(updatedRequirementItems, updated...)
			clearedItems = append(clearedItems, cleared...)
			changed = true
		}

		delete(store.acceptedQuests[roleID], title)
		if store.removedQuests[roleID] == nil {
			store.removedQuests[roleID] = make(map[string]bool)
		}
		store.removedQuests[roleID][title] = true
		currentRole.Items = normalizeRoleItems(updatedItems)
		currentRole.Currencies = normalizeRoleCurrencies(currentCurrencies)
		roles[index] = currentRole
		store.rolesByPID[playerID] = roles
		if changed {
			if err := store.persistPlayerStateLocked(playerID); err != nil {
				log.Printf("[session.Store] persist completed quest item requirements failed roleId=%s title=%s: %v", roleID, title, err)
			}
		}
		if err := store.deleteAcceptedQuestLocked(roleID, title); err != nil {
			log.Printf("[session.Store] delete accepted completed quest failed roleId=%s title=%s: %v", roleID, title, err)
		}
		if err := store.persistRemovedQuestLocked(playerID, roleID, title); err != nil {
			log.Printf("[session.Store] persist completed quest failed roleId=%s title=%s: %v", roleID, title, err)
		}

		completedRole := withRoleRuntimeDefaults(roles[index])
		return RoleQuestCompleteResult{
			Role:         completedRole,
			PlayerBase:   playerBaseDataFromRole(playerID, completedRole),
			UpdatedItems: normalizeRoleItems(updatedRequirementItems),
			ClearedItems: clearedItems,
			Currencies:   cloneRoleCurrencies(completedRole.Currencies),
			Found:        true,
			Completed:    true,
		}
	}

	return RoleQuestCompleteResult{
		ErrorCode:    "role_not_found",
		ErrorMessage: "角色不存在。",
	}
}

func (store *Store) GetRoleItem(playerID string, roleID string, containerType string, itemIndex int) (RoleItem, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

	containerType = strings.TrimSpace(containerType)
	for _, role := range store.rolesByPID[playerID] {
		if role.RoleID != roleID {
			continue
		}
		role = withRoleRuntimeDefaults(role)
		item, ok := findRoleItem(role.Items, containerType, itemIndex)
		if !ok {
			return RoleItem{}, false
		}
		return normalizeRoleItem(item), true
	}
	return RoleItem{}, false
}

func (store *Store) GrantRoleItem(playerID string, roleID string, item RoleItem) (granted RoleItem, ok bool) {
	item = normalizeRoleItem(item)
	if item.Type == "" {
		item.Type = "背包"
	}
	if item.Count <= 0 {
		item.Count = 1
	}
	if item.Name == "" {
		return RoleItem{}, false
	}

	baseCapacity, supported := roleContainerCapacity(item.Type)
	if !supported {
		return RoleItem{}, false
	}

	rolePersistLock := store.rolePersistenceLock(playerID, roleID)
	rolePersistLock.Lock()
	defer rolePersistLock.Unlock()

	store.mu.Lock()
	var roleSnapshot RoleSummary
	shouldPersist := false
	defer func() {
		store.mu.Unlock()
		if shouldPersist {
			if err := store.persistRoleStateSnapshot(playerID, roleID, roleSnapshot); err != nil {
				log.Printf("[session.Store] persist granted item failed: %v", err)
			}
		}
	}()

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		roles[index] = withRoleRuntimeDefaults(roles[index])
		capacity := effectiveRoleContainerCapacity(roles[index].Items, item.Type, baseCapacity)
		updatedItems, grantedItem, ok := grantRoleItemToItems(roles[index].Items, capacity, item)
		if !ok {
			return RoleItem{}, false
		}
		roles[index].Items = normalizeRoleItems(updatedItems)
		if item.Type == "装备" {
			roles[index] = syncRoleProgressionRuntimeData(roles[index])
		}
		store.rolesByPID[playerID] = roles
		roleSnapshot = roles[index]
		shouldPersist = true
		return grantedItem, true
	}
	return RoleItem{}, false
}

func (store *Store) PurchaseRoleItem(playerID string, roleID string, item RoleItem, requirements []RoleItemRequirement) (result RoleItemPurchaseResult) {
	item = normalizeRoleItem(item)
	if item.Type == "" {
		item.Type = "背包"
	}
	if item.Index < 0 {
		item.Index = -1
	}
	if item.Count <= 0 {
		item.Count = 1
	}
	if item.Name == "" {
		return RoleItemPurchaseResult{
			ErrorCode:    "invalid_item",
			ErrorMessage: "物品不存在。",
		}
	}
	baseCapacity, supported := roleContainerCapacity(item.Type)
	if !supported {
		return RoleItemPurchaseResult{
			ErrorCode:    "invalid_container",
			ErrorMessage: "目标容器无效。",
		}
	}

	rolePersistLock := store.rolePersistenceLock(playerID, roleID)
	rolePersistLock.Lock()
	defer rolePersistLock.Unlock()

	store.mu.Lock()
	var roleSnapshot RoleSummary
	shouldPersist := false
	defer func() {
		store.mu.Unlock()
		if shouldPersist {
			if err := store.persistRoleStateSnapshot(playerID, roleID, roleSnapshot); err != nil {
				log.Printf("[session.Store] persist purchased item failed: %v", err)
			}
		}
	}()

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		currentRole := withRoleRuntimeDefaults(roles[index])
		capacity := effectiveRoleContainerCapacity(currentRole.Items, item.Type, baseCapacity)
		currentBase := playerBaseDataFromRole(playerID, currentRole)
		currentCurrencies := cloneRoleCurrencies(currentRole.Currencies)
		originalCurrencies := cloneRoleCurrencies(currentCurrencies)
		updatedItems := cloneRoleItems(currentRole.Items)
		var trimmedCurrencyItems []RoleItem
		var preClearedItems []RoleItemClear
		updatedItems, trimmedCurrencyItems, preClearedItems, _ = trimRoleCurrencyItemsToBalance(updatedItems, currentCurrencies)
		normalizedRequirements := normalizeRoleItemRequirements(requirements)
		for _, requirement := range normalizedRequirements {
			if requirement.Count <= 0 {
				continue
			}
			if isRoleCurrencyName(requirement.Name) {
				if currentCurrencies[requirement.Name] < requirement.Count {
					return RoleItemPurchaseResult{
						Role:         currentRole,
						PlayerBase:   currentBase,
						Currencies:   currentCurrencies,
						Found:        true,
						ErrorCode:    "not_enough_currency",
						ErrorMessage: fmt.Sprintf("%s不足。", requirement.Name),
					}
				}
				continue
			}
			if totalRoleItemCountByName(updatedItems, "背包", requirement.Name) < requirement.Count {
				return RoleItemPurchaseResult{
					Role:         currentRole,
					PlayerBase:   currentBase,
					Currencies:   currentCurrencies,
					Found:        true,
					ErrorCode:    "item_not_enough",
					ErrorMessage: fmt.Sprintf("%s不足。", requirement.Name),
				}
			}
		}

		consumedItems := append([]RoleItem{}, trimmedCurrencyItems...)
		clearedItems := append([]RoleItemClear{}, preClearedItems...)
		for _, requirement := range normalizedRequirements {
			if requirement.Count <= 0 {
				continue
			}
			if isRoleCurrencyName(requirement.Name) {
				currentCurrencies[requirement.Name] -= requirement.Count
				var consumed []RoleItem
				var cleared []RoleItemClear
				updatedItems, consumed, cleared = consumeRoleItemsByName(updatedItems, "背包", requirement.Name, requirement.Count)
				consumedItems = append(consumedItems, consumed...)
				clearedItems = append(clearedItems, cleared...)
				continue
			}
			var consumed []RoleItem
			var cleared []RoleItemClear
			updatedItems, consumed, cleared = consumeRoleItemsByName(updatedItems, "背包", requirement.Name, requirement.Count)
			consumedItems = append(consumedItems, consumed...)
			clearedItems = append(clearedItems, cleared...)
		}

		var purchasedItem RoleItem
		var purchased bool
		updatedItems, purchasedItem, purchased = grantRoleItemToItems(updatedItems, capacity, item)
		if !purchased {
			return RoleItemPurchaseResult{
				Role:         currentRole,
				PlayerBase:   currentBase,
				Currencies:   originalCurrencies,
				Found:        true,
				ErrorCode:    "container_full",
				ErrorMessage: "背包已满。",
			}
		}

		currentRole.Items = normalizeRoleItems(updatedItems)
		currentRole.Currencies = cloneRoleCurrencies(currentCurrencies)
		roles[index] = currentRole
		store.rolesByPID[playerID] = roles
		roleSnapshot = roles[index]
		shouldPersist = true

		role := withRoleRuntimeDefaults(roles[index])
		return RoleItemPurchaseResult{
			Role:         role,
			PlayerBase:   playerBaseDataFromRole(playerID, role),
			Item:         purchasedItem,
			Consumed:     consumedItems,
			ClearedItems: clearedItems,
			Currencies:   cloneRoleCurrencies(role.Currencies),
			Found:        true,
			Purchased:    true,
		}
	}

	return RoleItemPurchaseResult{
		ErrorCode:    "role_missing",
		ErrorMessage: "角色不存在。",
	}
}

func (store *Store) GrantRoleExperience(playerID string, roleID string, expDelta int) (result RoleExpGrantResult) {
	if expDelta <= 0 {
		return RoleExpGrantResult{}
	}

	rolePersistLock := store.rolePersistenceLock(playerID, roleID)
	rolePersistLock.Lock()
	defer rolePersistLock.Unlock()

	store.mu.Lock()
	var roleSnapshot RoleSummary
	shouldPersist := false
	defer func() {
		store.mu.Unlock()
		if shouldPersist {
			if err := store.persistRoleStateSnapshot(playerID, roleID, roleSnapshot); err != nil {
				log.Printf("[session.Store] persist granted experience failed: %v", err)
			}
		}
	}()

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		roles[index] = withRoleRuntimeDefaults(roles[index])
		previousLevel := roles[index].Level
		roles[index].Exp += expDelta
		roles[index].Level = ClassicRoleLevelForExp(roles[index].Exp, roles[index].Level)
		roles[index] = syncRoleProgressionRuntimeData(roles[index])
		store.rolesByPID[playerID] = roles
		roleSnapshot = roles[index]
		shouldPersist = true

		role := withRoleRuntimeDefaults(roles[index])
		playerBase := playerBaseDataFromRole(playerID, role)
		roleState := *playerBase.RoleState
		rolePhysique := *playerBase.RolePhysique
		return RoleExpGrantResult{
			Role:         role,
			PlayerBase:   playerBase,
			RoleState:    roleState,
			RolePhysique: rolePhysique,
			Found:        true,
			Granted:      true,
			LevelChanged: role.Level != previousLevel,
		}
	}
	return RoleExpGrantResult{}
}

func (store *Store) UpdateRoleState(playerID string, roleID string, roleState RoleState) (RoleSummary, PlayerBaseData, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		role := withRoleRuntimeDefaults(roles[index])
		role.Level = ClassicRoleLevelForExp(role.Exp, role.Level)
		rolePhysique := defaultRolePhysique(role)
		if role.RolePhysique != nil {
			rolePhysique = *role.RolePhysique
			if rolePhysique.Handle == "" {
				rolePhysique.Handle = role.RoleID
			}
		}
		if roleState.Handle == "" {
			roleState.Handle = role.RoleID
		}
		roleState.HP = clampRoleRuntimeValue(roleState.HP, 0, rolePhysique.MaxHP)
		roleState.MP = clampRoleRuntimeValue(roleState.MP, 0, rolePhysique.MaxMP)
		roleState.Exp = role.Exp
		roleState.Lv = role.Level
		roleState.Speed = ClassicRoleSpeed(role.Level)
		role.RoleState = &roleState
		role.RolePhysique = &rolePhysique
		roles[index] = role
		store.rolesByPID[playerID] = roles
		if err := store.persistPlayerStateLocked(playerID); err != nil {
			log.Printf("[session.Store] persist role state failed: %v", err)
		}

		role = withRoleRuntimeDefaults(roles[index])
		return role, playerBaseDataFromRole(playerID, role), true
	}
	return RoleSummary{}, emptyPlayerBaseData(playerID), false
}

func (store *Store) SetRoleLevel(playerID string, roleID string, level int) RoleExpGrantResult {
	store.mu.Lock()
	defer store.mu.Unlock()

	if level <= 0 {
		return RoleExpGrantResult{}
	}
	if level > ClassicRoleMaxLevel() {
		level = ClassicRoleMaxLevel()
	}

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		roles[index] = withRoleRuntimeDefaults(roles[index])
		previousLevel := roles[index].Level
		roles[index].Level = level
		if level <= 1 {
			roles[index].Exp = 0
		} else {
			roles[index].Exp = ClassicRoleLevelToExp(level - 1)
		}
		roles[index] = syncRoleProgressionRuntimeData(roles[index])
		store.rolesByPID[playerID] = roles
		if err := store.persistPlayerStateLocked(playerID); err != nil {
			log.Printf("[session.Store] persist set role level failed: %v", err)
		}

		role := withRoleRuntimeDefaults(roles[index])
		playerBase := playerBaseDataFromRole(playerID, role)
		roleState := *playerBase.RoleState
		rolePhysique := *playerBase.RolePhysique
		return RoleExpGrantResult{
			Role:         role,
			PlayerBase:   playerBase,
			RoleState:    roleState,
			RolePhysique: rolePhysique,
			Found:        true,
			Granted:      true,
			LevelChanged: role.Level != previousLevel,
		}
	}
	return RoleExpGrantResult{}
}

func (store *Store) SetRolePointStats(playerID string, roleID string, stats RolePointStats) RoleExpGrantResult {
	store.mu.Lock()
	defer store.mu.Unlock()

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		roles[index] = withRoleRuntimeDefaults(roles[index])
		if stats.AGI != nil {
			roles[index].AGI = maxInt(0, *stats.AGI)
		}
		if stats.STR != nil {
			roles[index].STR = maxInt(0, *stats.STR)
		}
		if stats.INT != nil {
			roles[index].INT = maxInt(0, *stats.INT)
		}
		if stats.CON != nil {
			roles[index].CON = maxInt(0, *stats.CON)
		}
		if stats.LCK != nil {
			roles[index].LCK = maxInt(0, *stats.LCK)
		}
		roles[index] = syncRoleProgressionRuntimeData(roles[index])
		store.rolesByPID[playerID] = roles
		if err := store.persistPlayerStateLocked(playerID); err != nil {
			log.Printf("[session.Store] persist set role point stats failed: %v", err)
		}

		role := withRoleRuntimeDefaults(roles[index])
		playerBase := playerBaseDataFromRole(playerID, role)
		roleState := *playerBase.RoleState
		rolePhysique := *playerBase.RolePhysique
		return RoleExpGrantResult{
			Role:         role,
			PlayerBase:   playerBase,
			RoleState:    roleState,
			RolePhysique: rolePhysique,
			Found:        true,
			Granted:      true,
		}
	}
	return RoleExpGrantResult{}
}

func (store *Store) SetRoleVocation(playerID string, roleID string, vocation string) RoleVocationResult {
	store.mu.Lock()
	defer store.mu.Unlock()

	vocation, ok := normalizeRoleVocation(vocation)
	if !ok {
		return RoleVocationResult{
			ErrorCode:    "invalid_vocation",
			ErrorMessage: "职业不存在。",
		}
	}

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		roles[index] = withRoleRuntimeDefaults(roles[index])
		changed := roles[index].Voc != vocation
		roles[index].Voc = vocation
		store.rolesByPID[playerID] = roles
		if changed {
			if err := store.persistPlayerStateLocked(playerID); err != nil {
				log.Printf("[session.Store] persist role vocation failed: %v", err)
			}
		}

		role := withRoleRuntimeDefaults(roles[index])
		playerBase := playerBaseDataFromRole(playerID, role)
		roleState := *playerBase.RoleState
		rolePhysique := *playerBase.RolePhysique
		return RoleVocationResult{
			Role:         role,
			PlayerBase:   playerBase,
			RoleState:    roleState,
			RolePhysique: rolePhysique,
			Found:        true,
			Changed:      changed,
		}
	}

	return RoleVocationResult{
		ErrorCode:    "role_missing",
		ErrorMessage: "角色不存在。",
	}
}

func (store *Store) AddRolePoint(playerID string, roleID string, statName string) RoleAddPointResult {
	store.mu.Lock()
	defer store.mu.Unlock()

	statName = strings.ToUpper(strings.TrimSpace(statName))
	if !isClassicRolePointStat(statName) {
		return RoleAddPointResult{
			ErrorCode:    "invalid_stat",
			ErrorMessage: "属性不存在。",
		}
	}

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		roles[index] = withRoleRuntimeDefaults(roles[index])
		roles[index].Level = ClassicRoleLevelForExp(roles[index].Exp, roles[index].Level)
		if remainingRolePoint(roles[index]) <= 0 {
			return RoleAddPointResult{
				Role:         withRoleRuntimeDefaults(roles[index]),
				PlayerBase:   playerBaseDataFromRole(playerID, roles[index]),
				Found:        true,
				ErrorCode:    "no_point",
				ErrorMessage: "没有可分配的能力点。",
			}
		}
		addRolePointToStat(&roles[index], statName)
		roles[index] = syncRoleProgressionRuntimeData(roles[index])
		store.rolesByPID[playerID] = roles
		if err := store.persistPlayerStateLocked(playerID); err != nil {
			log.Printf("[session.Store] persist added role point failed: %v", err)
		}

		role := withRoleRuntimeDefaults(roles[index])
		playerBase := playerBaseDataFromRole(playerID, role)
		rolePhysique := *playerBase.RolePhysique
		return RoleAddPointResult{
			Role:         role,
			PlayerBase:   playerBase,
			RolePhysique: rolePhysique,
			Found:        true,
			Applied:      true,
		}
	}
	return RoleAddPointResult{
		ErrorCode:    "role_missing",
		ErrorMessage: "角色不存在。",
	}
}

func (store *Store) ConsumeRoleItem(playerID string, roleID string, sourceType string, sourceIndex int, count int) RoleUseItemResult {
	store.mu.Lock()
	defer store.mu.Unlock()

	sourceType = strings.TrimSpace(sourceType)
	if count <= 0 {
		count = 1
	}

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		roles[index] = withRoleRuntimeDefaults(roles[index])
		sourceItem, ok := findRoleItem(roles[index].Items, sourceType, sourceIndex)
		if !ok {
			role := withRoleRuntimeDefaults(roles[index])
			return RoleUseItemResult{
				Role:         role,
				PlayerBase:   playerBaseDataFromRole(playerID, role),
				Found:        true,
				ErrorCode:    "item_missing",
				ErrorMessage: "物品不存在。",
			}
		}
		if sourceItem.Count < count {
			role := withRoleRuntimeDefaults(roles[index])
			return RoleUseItemResult{
				Role:         role,
				PlayerBase:   playerBaseDataFromRole(playerID, role),
				Item:         sourceItem,
				Found:        true,
				ErrorCode:    "item_not_enough",
				ErrorMessage: "物品数量不足。",
			}
		}

		updatedItems := make([]RoleItem, 0, len(roles[index].Items))
		for _, item := range roles[index].Items {
			if item.Type == sourceType && item.Index == sourceIndex {
				continue
			}
			updatedItems = append(updatedItems, item)
		}

		result := RoleUseItemResult{
			Item:  normalizeRoleItem(sourceItem),
			Found: true,
			Used:  true,
		}
		if sourceItem.Count > count {
			updatedItem := sourceItem
			updatedItem.Count -= count
			updatedItems = append(updatedItems, normalizeRoleItem(updatedItem))
			normalizedUpdated := normalizeRoleItem(updatedItem)
			result.UpdatedItem = &normalizedUpdated
		} else {
			result.ClearedItems = []RoleItemClear{{
				Type:  sourceType,
				Index: sourceIndex,
			}}
		}

		roles[index].Items = normalizeRoleItems(updatedItems)
		if sourceType == "装备" {
			roles[index] = syncRoleProgressionRuntimeData(roles[index])
		}
		store.rolesByPID[playerID] = roles
		if err := store.persistRoleStateLocked(playerID, roleID); err != nil {
			log.Printf("[session.Store] persist consumed item failed: %v", err)
		}

		role := withRoleRuntimeDefaults(roles[index])
		result.Role = role
		result.PlayerBase = playerBaseDataFromRole(playerID, role)
		return result
	}

	return RoleUseItemResult{
		Found:        false,
		ErrorCode:    "role_missing",
		ErrorMessage: "角色不存在。",
	}
}

func (store *Store) ConsumeRoleItemMutationOnly(playerID string, roleID string, sourceType string, sourceIndex int, count int) RoleUseItemResult {
	sourceType = strings.TrimSpace(sourceType)
	if count <= 0 {
		count = 1
	}

	rolePersistLock := store.rolePersistenceLock(playerID, roleID)
	rolePersistLock.Lock()
	defer rolePersistLock.Unlock()

	store.mu.Lock()
	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		sourceItem, ok := findRoleItem(roles[index].Items, sourceType, sourceIndex)
		if !ok {
			roles[index] = withRoleRuntimeDefaults(roles[index])
			sourceItem, ok = findRoleItem(roles[index].Items, sourceType, sourceIndex)
			if !ok {
				store.mu.Unlock()
				return RoleUseItemResult{
					Role:         RoleSummary{RoleID: roleID},
					PlayerBase:   PlayerBaseData{PlayerID: playerID, RoleID: roleID},
					Found:        true,
					ErrorCode:    "item_missing",
					ErrorMessage: "物品不存在。",
				}
			}
		}
		if sourceItem.Count < count {
			store.mu.Unlock()
			return RoleUseItemResult{
				Role:         RoleSummary{RoleID: roleID},
				PlayerBase:   PlayerBaseData{PlayerID: playerID, RoleID: roleID},
				Item:         normalizeRoleItem(sourceItem),
				Found:        true,
				ErrorCode:    "item_not_enough",
				ErrorMessage: "物品数量不足。",
			}
		}

		updatedItems := make([]RoleItem, 0, len(roles[index].Items))
		for _, item := range roles[index].Items {
			if item.Type == sourceType && item.Index == sourceIndex {
				continue
			}
			updatedItems = append(updatedItems, item)
		}

		result := RoleUseItemResult{
			Role:       RoleSummary{RoleID: roleID},
			PlayerBase: PlayerBaseData{PlayerID: playerID, RoleID: roleID},
			Item:       normalizeRoleItem(sourceItem),
			Found:      true,
			Used:       true,
		}
		if sourceItem.Count > count {
			updatedItem := sourceItem
			updatedItem.Count -= count
			updatedItems = append(updatedItems, normalizeRoleItem(updatedItem))
			normalizedUpdated := normalizeRoleItem(updatedItem)
			result.UpdatedItem = &normalizedUpdated
		} else {
			result.ClearedItems = []RoleItemClear{{
				Type:  sourceType,
				Index: sourceIndex,
			}}
		}

		roles[index].Items = normalizeRoleItems(updatedItems)
		if sourceType == "装备" {
			roles[index] = syncRoleProgressionRuntimeData(roles[index])
			result.Role = withRoleRuntimeDefaults(roles[index])
			result.PlayerBase = playerBaseDataFromRole(playerID, result.Role)
			store.rolesByPID[playerID] = roles
			if err := store.persistRoleStateLocked(playerID, roleID); err != nil {
				log.Printf("[session.Store] persist mutation-only consumed equipment failed: %v", err)
			}
			store.mu.Unlock()
			return result
		}

		store.rolesByPID[playerID] = roles
		itemsJSON, err := encodeRoleItems(roles[index].Items)
		roleSnapshot := roles[index]
		store.mu.Unlock()
		if err != nil {
			log.Printf("[session.Store] encode mutation-only consumed item failed: %v", err)
			return result
		}
		if err := store.persistRoleItemsSnapshot(playerID, roleID, itemsJSON, roleSnapshot); err != nil {
			log.Printf("[session.Store] persist mutation-only consumed item failed: %v", err)
		}
		return result
	}
	store.mu.Unlock()

	return RoleUseItemResult{
		Found:        false,
		ErrorCode:    "role_missing",
		ErrorMessage: "角色不存在。",
	}
}

func (store *Store) UseRoleRecoveryItemMutationOnly(playerID string, roleID string, sourceType string, sourceIndex int, healHP int, healMP int) RoleUseItemResult {
	sourceType = strings.TrimSpace(sourceType)
	if healHP <= 0 && healMP <= 0 {
		return RoleUseItemResult{
			Role:         RoleSummary{RoleID: roleID},
			PlayerBase:   PlayerBaseData{PlayerID: playerID, RoleID: roleID},
			Found:        true,
			ErrorCode:    "item_not_usable",
			ErrorMessage: "该物品不能使用。",
		}
	}

	rolePersistLock := store.rolePersistenceLock(playerID, roleID)
	rolePersistLock.Lock()
	defer rolePersistLock.Unlock()

	store.mu.Lock()
	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		roles[index] = withRoleRuntimeDefaults(roles[index])
		sourceItem, ok := findRoleItem(roles[index].Items, sourceType, sourceIndex)
		if !ok {
			role := withRoleRuntimeDefaults(roles[index])
			store.mu.Unlock()
			return RoleUseItemResult{
				Role:         role,
				PlayerBase:   playerBaseDataFromRole(playerID, role),
				Found:        true,
				ErrorCode:    "item_missing",
				ErrorMessage: "物品不存在。",
			}
		}
		if sourceItem.Count <= 0 {
			role := withRoleRuntimeDefaults(roles[index])
			store.mu.Unlock()
			return RoleUseItemResult{
				Role:         role,
				PlayerBase:   playerBaseDataFromRole(playerID, role),
				Item:         normalizeRoleItem(sourceItem),
				Found:        true,
				ErrorCode:    "item_not_enough",
				ErrorMessage: "物品数量不足。",
			}
		}

		roleState := defaultRoleState(roles[index].RoleID, roles[index].Level, roles[index].Exp)
		if roles[index].RoleState != nil {
			roleState = *roles[index].RoleState
			if roleState.Handle == "" {
				roleState.Handle = roles[index].RoleID
			}
		}
		roleState.Exp = roles[index].Exp
		roleState.Lv = roles[index].Level
		roleState.Speed = ClassicRoleSpeed(roles[index].Level)

		rolePhysique := defaultRolePhysique(roles[index])
		if roles[index].RolePhysique != nil {
			rolePhysique = *roles[index].RolePhysique
			if rolePhysique.Handle == "" {
				rolePhysique.Handle = roles[index].RoleID
			}
		}
		canRecoverHP := healHP > 0 && roleState.HP < rolePhysique.MaxHP
		canRecoverMP := healMP > 0 && roleState.MP < rolePhysique.MaxMP
		if !canRecoverHP && !canRecoverMP {
			role := withRoleRuntimeDefaults(roles[index])
			store.mu.Unlock()
			return RoleUseItemResult{
				Role:         role,
				PlayerBase:   playerBaseDataFromRole(playerID, role),
				Item:         normalizeRoleItem(sourceItem),
				Found:        true,
				ErrorCode:    "role_state_full",
				ErrorMessage: "当前状态不需要使用该物品。",
			}
		}
		if healHP > 0 {
			roleState.HP = clampRoleRuntimeValue(roleState.HP+healHP, 0, rolePhysique.MaxHP)
		}
		if healMP > 0 {
			roleState.MP = clampRoleRuntimeValue(roleState.MP+healMP, 0, rolePhysique.MaxMP)
		}

		updatedItems, updatedSource, clearedItems := consumeRoleItemBySlot(roles[index].Items, sourceItem.Type, sourceItem.Index, 1)
		roles[index].Items = normalizeRoleItems(updatedItems)
		roles[index].RoleState = &roleState
		roles[index].RolePhysique = &rolePhysique
		store.rolesByPID[playerID] = roles

		role := withRoleRuntimeDefaults(roles[index])
		result := RoleUseItemResult{
			Role:             role,
			PlayerBase:       playerBaseDataFromRole(playerID, role),
			Item:             normalizeRoleItem(sourceItem),
			ClearedItems:     clearedItems,
			Found:            true,
			Used:             true,
			RoleStateChanged: true,
		}
		if updatedSource != nil {
			updatedItem := *updatedSource
			result.UpdatedItem = &updatedItem
			result.UpdatedItems = []RoleItem{updatedItem}
		}
		roleSnapshot := roles[index]
		store.mu.Unlock()
		if err := store.persistRoleStateSnapshot(playerID, roleID, roleSnapshot); err != nil {
			log.Printf("[session.Store] persist mutation-only recovery item failed: %v", err)
		}
		return result
	}
	store.mu.Unlock()

	return RoleUseItemResult{
		Found:        false,
		ErrorCode:    "role_missing",
		ErrorMessage: "角色不存在。",
	}
}

func (store *Store) GetRolePetInfo(playerID string, roleID string) RolePetInfoResult {
	store.mu.Lock()
	defer store.mu.Unlock()

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		roles[index] = withRoleRuntimeDefaults(roles[index])
		role := roles[index]
		petItem, _, ok := findEquippedRolePetItem(role.Items)
		if !ok {
			return RolePetInfoResult{
				Role:         role,
				PlayerBase:   playerBaseDataFromRole(playerID, role),
				Found:        true,
				ErrorCode:    "pet_missing",
				ErrorMessage: "你没有装备宠物。",
			}
		}
		return buildRolePetInfoResult(playerID, role, petItem)
	}

	return RolePetInfoResult{
		Found:        false,
		ErrorCode:    "role_missing",
		ErrorMessage: "角色不存在。",
	}
}

func (store *Store) FeedRolePet(playerID string, roleID string, sourceType string, sourceIndex int, count int) RolePetFeedResult {
	store.mu.Lock()
	defer store.mu.Unlock()

	sourceType = strings.TrimSpace(sourceType)
	if sourceType == "" {
		sourceType = "背包"
	}
	if count <= 0 {
		count = 1
	}

	roles := store.rolesByPID[playerID]
	for roleIndex := range roles {
		if roles[roleIndex].RoleID != roleID {
			continue
		}

		roles[roleIndex] = withRoleRuntimeDefaults(roles[roleIndex])
		petItem, petItemIndex, ok := findEquippedRolePetItem(roles[roleIndex].Items)
		if !ok {
			role := withRoleRuntimeDefaults(roles[roleIndex])
			return RolePetFeedResult{
				RolePetInfoResult: RolePetInfoResult{
					Role:         role,
					PlayerBase:   playerBaseDataFromRole(playerID, role),
					Found:        true,
					ErrorCode:    "pet_missing",
					ErrorMessage: "你没有装备宠物。",
				},
			}
		}

		feedItem, feedItemIndex, ok := findRolePetFeedItem(roles[roleIndex].Items, sourceType, sourceIndex)
		if !ok {
			role := withRoleRuntimeDefaults(roles[roleIndex])
			info := buildRolePetInfoResult(playerID, role, petItem)
			return RolePetFeedResult{
				RolePetInfoResult: withRolePetInfoError(info, "feed_missing", "背包内没有可喂食的宠物食品。"),
			}
		}
		if feedItem.Count < count {
			role := withRoleRuntimeDefaults(roles[roleIndex])
			info := buildRolePetInfoResult(playerID, role, petItem)
			return RolePetFeedResult{
				RolePetInfoResult: withRolePetInfoError(info, "feed_not_enough", "宠物食品数量不足。"),
				FeedItem:          normalizeRoleItem(feedItem),
			}
		}

		growthGain, fullnessGain := rolePetFeedGains(feedItem)
		petItem = applyRolePetFeedState(petItem, growthGain, fullnessGain)
		updatedItems := cloneRoleItems(roles[roleIndex].Items)
		updatedItems[petItemIndex] = normalizeRoleItem(petItem)
		result := RolePetFeedResult{
			FeedItem: normalizeRoleItem(feedItem),
			Fed:      true,
		}
		if feedItem.Count > count {
			feedItem.Count -= count
			updatedItems[feedItemIndex] = normalizeRoleItem(feedItem)
			updatedItem := normalizeRoleItem(feedItem)
			result.UpdatedItem = &updatedItem
		} else {
			updatedItems = append(updatedItems[:feedItemIndex], updatedItems[feedItemIndex+1:]...)
			result.ClearedItems = []RoleItemClear{{
				Type:  feedItem.Type,
				Index: feedItem.Index,
			}}
		}

		roles[roleIndex].Items = normalizeRoleItems(updatedItems)
		store.rolesByPID[playerID] = roles
		if err := store.persistPlayerStateLocked(playerID); err != nil {
			log.Printf("[session.Store] persist pet feed failed: %v", err)
		}

		role := withRoleRuntimeDefaults(roles[roleIndex])
		persistedPetItem, _, _ := findEquippedRolePetItem(role.Items)
		result.RolePetInfoResult = buildRolePetInfoResult(playerID, role, persistedPetItem)
		return result
	}

	return RolePetFeedResult{
		RolePetInfoResult: RolePetInfoResult{
			Found:        false,
			ErrorCode:    "role_missing",
			ErrorMessage: "角色不存在。",
		},
	}
}

func (store *Store) SellRoleItem(playerID string, roleID string, sourceType string, sourceIndex int, count int) (result RoleItemSaleResult) {
	sourceType = strings.TrimSpace(sourceType)
	if count <= 0 {
		count = 1
	}

	rolePersistLock := store.rolePersistenceLock(playerID, roleID)
	rolePersistLock.Lock()
	defer rolePersistLock.Unlock()

	store.mu.Lock()
	var roleSnapshot RoleSummary
	shouldPersist := false
	defer func() {
		store.mu.Unlock()
		if shouldPersist {
			if err := store.persistRoleStateSnapshot(playerID, roleID, roleSnapshot); err != nil {
				log.Printf("[session.Store] persist sold item failed: %v", err)
			}
		}
	}()

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		roles[index] = withRoleRuntimeDefaults(roles[index])
		sourceItem, ok := findRoleItem(roles[index].Items, sourceType, sourceIndex)
		if !ok {
			role := withRoleRuntimeDefaults(roles[index])
			return RoleItemSaleResult{
				Role:         role,
				PlayerBase:   playerBaseDataFromRole(playerID, role),
				Found:        true,
				ErrorCode:    "item_missing",
				ErrorMessage: "物品不存在。",
			}
		}
		if sourceItem.Count < count {
			role := withRoleRuntimeDefaults(roles[index])
			return RoleItemSaleResult{
				Role:         role,
				PlayerBase:   playerBaseDataFromRole(playerID, role),
				Item:         normalizeRoleItem(sourceItem),
				Found:        true,
				ErrorCode:    "item_not_enough",
				ErrorMessage: "物品数量不足。",
			}
		}
		price := parseClassicDescriptionSignedInt(sourceItem.Description, "108")
		if price <= 0 {
			role := withRoleRuntimeDefaults(roles[index])
			return RoleItemSaleResult{
				Role:         role,
				PlayerBase:   playerBaseDataFromRole(playerID, role),
				Item:         normalizeRoleItem(sourceItem),
				Found:        true,
				ErrorCode:    "item_no_sale",
				ErrorMessage: "该物品无法出售。",
			}
		}

		updatedItems, updatedItem, clearedItems := consumeRoleItemBySlot(roles[index].Items, sourceType, sourceIndex, count)
		currencies := cloneRoleCurrencies(roles[index].Currencies)
		amount := price * count
		currencies["铜钱"] += amount
		roles[index].Items = normalizeRoleItems(updatedItems)
		roles[index].Currencies = normalizeRoleCurrencies(currencies)
		if sourceType == "装备" {
			roles[index] = syncRoleProgressionRuntimeData(roles[index])
		}
		store.rolesByPID[playerID] = roles
		roleSnapshot = roles[index]
		shouldPersist = true

		role := withRoleRuntimeDefaults(roles[index])
		return RoleItemSaleResult{
			Role:         role,
			PlayerBase:   playerBaseDataFromRole(playerID, role),
			Item:         normalizeRoleItem(sourceItem),
			UpdatedItem:  updatedItem,
			ClearedItems: clearedItems,
			Currencies:   cloneRoleCurrencies(role.Currencies),
			Found:        true,
			Sold:         true,
			Amount:       amount,
		}
	}

	return RoleItemSaleResult{
		Found:        false,
		ErrorCode:    "role_missing",
		ErrorMessage: "角色不存在。",
	}
}

func (store *Store) HealRoleAtTown(playerID string, roleID string) RoleTownHealResult {
	store.mu.Lock()
	defer store.mu.Unlock()

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		currentRole := withRoleRuntimeDefaults(roles[index])
		currentBase := playerBaseDataFromRole(playerID, currentRole)
		if currentBase.RoleState == nil || currentBase.RolePhysique == nil {
			return RoleTownHealResult{
				Role:         currentRole,
				PlayerBase:   currentBase,
				Currencies:   cloneRoleCurrencies(currentRole.Currencies),
				Found:        true,
				ErrorCode:    "role_state_missing",
				ErrorMessage: "角色状态不存在。",
			}
		}

		roleState := *currentBase.RoleState
		rolePhysique := *currentBase.RolePhysique
		missingHP := maxInt(0, rolePhysique.MaxHP-roleState.HP)
		missingMP := maxInt(0, rolePhysique.MaxMP-roleState.MP)
		if missingHP == 0 && missingMP == 0 {
			return RoleTownHealResult{
				Role:       currentRole,
				PlayerBase: currentBase,
				RoleState:  roleState,
				Currencies: cloneRoleCurrencies(currentRole.Currencies),
				Found:      true,
				NearlyFull: true,
			}
		}

		cost := ClassicTownHealerCost(currentRole.Level, missingHP, missingMP)
		currentCurrencies := cloneRoleCurrencies(currentRole.Currencies)
		updatedItems := cloneRoleItems(currentRole.Items)
		var updatedCurrencyItems []RoleItem
		var clearedItems []RoleItemClear
		updatedItems, updatedCurrencyItems, clearedItems, _ = trimRoleCurrencyItemsToBalance(updatedItems, currentCurrencies)
		if cost > 0 && currentCurrencies["铜钱"] < cost {
			return RoleTownHealResult{
				Role:         currentRole,
				PlayerBase:   currentBase,
				RoleState:    roleState,
				Currencies:   currentCurrencies,
				Found:        true,
				Cost:         cost,
				ErrorCode:    "not_enough_currency",
				ErrorMessage: "铜钱不足。",
			}
		}

		if cost > 0 {
			currentCurrencies["铜钱"] -= cost
			var consumedCurrencyItems []RoleItem
			var clearedCurrencyItems []RoleItemClear
			updatedItems, consumedCurrencyItems, clearedCurrencyItems = consumeRoleItemsByName(updatedItems, "背包", "铜钱", cost)
			updatedCurrencyItems = append(updatedCurrencyItems, consumedCurrencyItems...)
			clearedItems = append(clearedItems, clearedCurrencyItems...)
		}

		roleState.HP = rolePhysique.MaxHP
		roleState.MP = rolePhysique.MaxMP
		roleState.Exp = currentRole.Exp
		roleState.Lv = currentRole.Level
		roleState.Speed = ClassicRoleSpeed(currentRole.Level)

		currentRole.Items = normalizeRoleItems(updatedItems)
		currentRole.Currencies = normalizeRoleCurrencies(currentCurrencies)
		currentRole.RoleState = &roleState
		currentRole.RolePhysique = &rolePhysique
		roles[index] = currentRole
		store.rolesByPID[playerID] = roles
		if err := store.persistPlayerStateLocked(playerID); err != nil {
			log.Printf("[session.Store] persist town heal failed: %v", err)
		}

		role := withRoleRuntimeDefaults(roles[index])
		return RoleTownHealResult{
			Role:         role,
			PlayerBase:   playerBaseDataFromRole(playerID, role),
			RoleState:    roleState,
			UpdatedItems: updatedCurrencyItems,
			ClearedItems: clearedItems,
			Currencies:   cloneRoleCurrencies(role.Currencies),
			Found:        true,
			Healed:       true,
			Cost:         cost,
		}
	}

	return RoleTownHealResult{
		ErrorCode:    "role_missing",
		ErrorMessage: "角色不存在。",
	}
}

func (store *Store) UseRoleItem(playerID string, roleID string, sourceType string, sourceIndex int) RoleUseItemResult {
	store.mu.Lock()
	defer store.mu.Unlock()

	sourceType = strings.TrimSpace(sourceType)
	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		roles[index] = withRoleRuntimeDefaults(roles[index])
		sourceItem, ok := findRoleItem(roles[index].Items, sourceType, sourceIndex)
		if !ok {
			role := withRoleRuntimeDefaults(roles[index])
			return RoleUseItemResult{
				Role:         role,
				PlayerBase:   playerBaseDataFromRole(playerID, role),
				Found:        true,
				ErrorCode:    "item_missing",
				ErrorMessage: "物品不存在。",
			}
		}
		updatedItems, updatedCurrencyItems, clearedCurrencyItems, trimmedCurrencyItems := trimRoleCurrencyItemsToBalance(roles[index].Items, roles[index].Currencies)
		if trimmedCurrencyItems {
			roles[index].Items = normalizeRoleItems(updatedItems)
			store.rolesByPID[playerID] = roles
			if err := store.persistPlayerStateLocked(playerID); err != nil {
				log.Printf("[session.Store] persist trimmed currency items failed: %v", err)
			}
			if isRoleCurrencyName(sourceItem.Name) {
				if _, stillExists := findRoleItem(roles[index].Items, sourceType, sourceIndex); !stillExists {
					role := withRoleRuntimeDefaults(roles[index])
					return RoleUseItemResult{
						Role:         role,
						PlayerBase:   playerBaseDataFromRole(playerID, role),
						Item:         sourceItem,
						UpdatedItems: updatedCurrencyItems,
						ClearedItems: clearedCurrencyItems,
						Currencies:   cloneRoleCurrencies(role.Currencies),
						Found:        true,
						Used:         true,
					}
				}
				sourceItem, _ = findRoleItem(roles[index].Items, sourceType, sourceIndex)
			}
		}

		if sourceItem.Name == classicTownLevel1GiftBoxName && roles[index].Level >= 1 {
			return store.useLevel1GiftBoxLocked(playerID, roles, index, sourceItem)
		}
		if sourceItem.Name == classicTownLevel5GiftBoxName && roles[index].Level < 5 {
			role := withRoleRuntimeDefaults(roles[index])
			return RoleUseItemResult{
				Role:         role,
				PlayerBase:   playerBaseDataFromRole(playerID, role),
				Item:         sourceItem,
				Found:        true,
				ErrorCode:    "item_level_too_low",
				ErrorMessage: classicTownLevel5GiftBoxError,
			}
		}
		if sourceItem.Name == classicTownLevel5GiftBoxName && roles[index].Level >= 5 {
			return store.useLevel5GiftBoxLocked(playerID, roles, index, sourceItem)
		}
		if sourceItem.Name == classicTownLevel10GiftBoxName && roles[index].Level >= 10 {
			return store.useLevel10GiftBoxLocked(playerID, roles, index, sourceItem)
		}
		if sourceItem.Name == classicTownBagCapacityPatchName {
			return store.useBagCapacityPatchLocked(playerID, roles, index, sourceItem)
		}
		if sourceItem.Name == classicTownInitialExperienceCardName {
			return store.useInitialExperienceCardLocked(playerID, roles, index, sourceItem)
		}
		if sourceItem.Name == classicTownAdvancedExperienceCardName {
			return store.useAdvancedExperienceCardLocked(playerID, roles, index, sourceItem)
		}

		switch sourceItem.Name {
		case "银元宝":
			return store.useCurrencyExchangeItemLocked(playerID, roles, index, sourceItem, 1, "铜钱", 1000)
		case "铜钱":
			return store.useCurrencyExchangeItemLocked(playerID, roles, index, sourceItem, 1000, "银元宝", 1)
		default:
			if isAvoidMonsterBuffItem(sourceItem) {
				return store.useAvoidMonsterBuffItemLocked(playerID, roles, index, sourceItem)
			}
			if skill, ok := roleSkillFromItem(sourceItem); ok {
				return store.useSkillItemLocked(playerID, roles, index, sourceItem, skill)
			}
			targetIndex, ok := roleEquipTargetIndex(sourceItem)
			if ok {
				return store.useEquipmentItemLocked(playerID, roles, index, sourceItem, sourceType, sourceIndex, targetIndex)
			}
			if healHP, healMP, ok := roleItemRecoveryAmounts(sourceItem); ok {
				return store.useRecoveryItemLocked(playerID, roles, index, sourceItem, healHP, healMP)
			}
			role := withRoleRuntimeDefaults(roles[index])
			return RoleUseItemResult{
				Role:         role,
				PlayerBase:   playerBaseDataFromRole(playerID, role),
				Item:         sourceItem,
				Found:        true,
				ErrorCode:    "item_not_usable",
				ErrorMessage: "该物品不能使用。",
			}
		}
	}

	return RoleUseItemResult{
		Found:        false,
		ErrorCode:    "role_missing",
		ErrorMessage: "角色不存在。",
	}
}

func (store *Store) useBagCapacityPatchLocked(playerID string, roles []RoleSummary, roleIndex int, sourceItem RoleItem) RoleUseItemResult {
	baseCapacity, supported := roleContainerCapacity(sourceItem.Type)
	if !supported {
		role := withRoleRuntimeDefaults(roles[roleIndex])
		return RoleUseItemResult{
			Role:         role,
			PlayerBase:   playerBaseDataFromRole(playerID, role),
			Item:         sourceItem,
			Found:        true,
			ErrorCode:    "invalid_container",
			ErrorMessage: "invalid target container",
		}
	}

	capacity := effectiveRoleContainerCapacity(roles[roleIndex].Items, sourceItem.Type, baseCapacity)
	updatedItems, updatedSource, clearedItems := consumeRoleItemBySlot(roles[roleIndex].Items, sourceItem.Type, sourceItem.Index, 1)
	updatedResults := []RoleItem{}
	if updatedSource != nil {
		updatedResults = append(updatedResults, *updatedSource)
	}

	roles[roleIndex].Items = normalizeRoleItems(updatedItems)
	store.rolesByPID[playerID] = roles
	if err := store.persistPlayerStateLocked(playerID); err != nil {
		log.Printf("[session.Store] persist used bag capacity patch failed: %v", err)
	}

	role := withRoleRuntimeDefaults(roles[roleIndex])
	return RoleUseItemResult{
		Role:              role,
		PlayerBase:        playerBaseDataFromRole(playerID, role),
		Item:              sourceItem,
		UpdatedItems:      normalizeRoleItems(updatedResults),
		ClearedItems:      clearedItems,
		ContainerType:     sourceItem.Type,
		ContainerCapacity: capacity,
		Found:             true,
		Used:              true,
	}
}

func (store *Store) useLevel1GiftBoxLocked(playerID string, roles []RoleSummary, roleIndex int, sourceItem RoleItem) RoleUseItemResult {
	rewardItems, ok := classicTownLevel1GiftBoxRewards()
	if !ok {
		role := withRoleRuntimeDefaults(roles[roleIndex])
		return RoleUseItemResult{
			Role:         role,
			PlayerBase:   playerBaseDataFromRole(playerID, role),
			Item:         sourceItem,
			Found:        true,
			ErrorCode:    "target_item_missing",
			ErrorMessage: "gift box reward item missing",
		}
	}

	baseCapacity, supported := roleContainerCapacity(sourceItem.Type)
	if !supported {
		role := withRoleRuntimeDefaults(roles[roleIndex])
		return RoleUseItemResult{
			Role:         role,
			PlayerBase:   playerBaseDataFromRole(playerID, role),
			Item:         sourceItem,
			Found:        true,
			ErrorCode:    "invalid_container",
			ErrorMessage: "invalid target container",
		}
	}

	capacity := effectiveRoleContainerCapacity(roles[roleIndex].Items, sourceItem.Type, baseCapacity)
	updatedItems, updatedSource, clearedItems := consumeRoleItemBySlot(roles[roleIndex].Items, sourceItem.Type, sourceItem.Index, 1)
	updatedResults := []RoleItem{}
	if updatedSource != nil {
		updatedResults = append(updatedResults, *updatedSource)
	}
	for _, reward := range rewardItems {
		reward.Type = sourceItem.Type
		reward.Index = -1
		var granted RoleItem
		updatedItems, granted, ok = grantRoleItemToItems(updatedItems, capacity, reward)
		if !ok {
			role := withRoleRuntimeDefaults(roles[roleIndex])
			return RoleUseItemResult{
				Role:         role,
				PlayerBase:   playerBaseDataFromRole(playerID, role),
				Item:         sourceItem,
				Found:        true,
				ErrorCode:    "bag_full",
				ErrorMessage: "\u80cc\u5305\u5df2\u6ee1\u3002",
			}
		}
		updatedResults = append(updatedResults, granted)
	}

	roles[roleIndex].Items = normalizeRoleItems(updatedItems)
	store.rolesByPID[playerID] = roles
	if err := store.persistPlayerStateLocked(playerID); err != nil {
		log.Printf("[session.Store] persist used level 1 gift box failed: %v", err)
	}

	role := withRoleRuntimeDefaults(roles[roleIndex])
	return RoleUseItemResult{
		Role:         role,
		PlayerBase:   playerBaseDataFromRole(playerID, role),
		Item:         sourceItem,
		UpdatedItems: normalizeRoleItems(updatedResults),
		ClearedItems: clearedItems,
		Found:        true,
		Used:         true,
	}
}

func classicTownLevel1GiftBoxRewards() ([]RoleItem, bool) {
	rewardSpecs := []struct {
		name  string
		count int
	}{
		{name: "\u0035\u7ea7\u793c\u76d2", count: 1},
		{name: "\u004c\u907f\u602a\u7b26", count: 3},
		{name: "\u004c\u767e\u5e74\u4eba\u53c2\u679c", count: 1},
		{name: "\u004c\u767e\u5e74\u87e0\u6843", count: 1},
	}
	rewards := make([]RoleItem, 0, len(rewardSpecs))
	for _, spec := range rewardSpecs {
		item, ok := CapturedRoleItemTemplate(spec.name)
		if !ok {
			return nil, false
		}
		item.Type = "\u80cc\u5305"
		item.Index = -1
		item.Count = spec.count
		rewards = append(rewards, item)
	}
	return rewards, true
}

func (store *Store) useLevel5GiftBoxLocked(playerID string, roles []RoleSummary, roleIndex int, sourceItem RoleItem) RoleUseItemResult {
	rewardItems, ok := classicTownLevel5GiftBoxRewards()
	if !ok {
		role := withRoleRuntimeDefaults(roles[roleIndex])
		return RoleUseItemResult{
			Role:         role,
			PlayerBase:   playerBaseDataFromRole(playerID, role),
			Item:         sourceItem,
			Found:        true,
			ErrorCode:    "target_item_missing",
			ErrorMessage: "gift box reward item missing",
		}
	}

	baseCapacity, supported := roleContainerCapacity(sourceItem.Type)
	if !supported {
		role := withRoleRuntimeDefaults(roles[roleIndex])
		return RoleUseItemResult{
			Role:         role,
			PlayerBase:   playerBaseDataFromRole(playerID, role),
			Item:         sourceItem,
			Found:        true,
			ErrorCode:    "invalid_container",
			ErrorMessage: "invalid target container",
		}
	}

	capacity := effectiveRoleContainerCapacity(roles[roleIndex].Items, sourceItem.Type, baseCapacity)
	updatedItems, updatedSource, clearedItems := consumeRoleItemBySlot(roles[roleIndex].Items, sourceItem.Type, sourceItem.Index, 1)
	updatedResults := []RoleItem{}
	if updatedSource != nil {
		updatedResults = append(updatedResults, *updatedSource)
	}
	for _, reward := range rewardItems {
		reward.Type = sourceItem.Type
		reward.Index = -1
		var granted RoleItem
		updatedItems, granted, ok = grantRoleItemToItems(updatedItems, capacity, reward)
		if !ok {
			role := withRoleRuntimeDefaults(roles[roleIndex])
			return RoleUseItemResult{
				Role:         role,
				PlayerBase:   playerBaseDataFromRole(playerID, role),
				Item:         sourceItem,
				Found:        true,
				ErrorCode:    "bag_full",
				ErrorMessage: "背包已满。",
			}
		}
		updatedResults = append(updatedResults, granted)
	}

	roles[roleIndex].Items = normalizeRoleItems(updatedItems)
	store.rolesByPID[playerID] = roles
	if err := store.persistPlayerStateLocked(playerID); err != nil {
		log.Printf("[session.Store] persist used level 5 gift box failed: %v", err)
	}

	role := withRoleRuntimeDefaults(roles[roleIndex])
	return RoleUseItemResult{
		Role:         role,
		PlayerBase:   playerBaseDataFromRole(playerID, role),
		Item:         sourceItem,
		UpdatedItems: normalizeRoleItems(updatedResults),
		ClearedItems: clearedItems,
		Found:        true,
		Used:         true,
	}
}

func classicTownLevel5GiftBoxRewards() ([]RoleItem, bool) {
	rewardSpecs := []struct {
		name  string
		count int
	}{
		{name: "\u004c\u521d\u9636\u7ecf\u9a8c\u5361", count: 1},
		{name: "\u004c\u82b1\u5377", count: 2},
		{name: "\u004c\u56de\u57ce\u5492", count: 3},
		{name: "\u0031\u0030\u7ea7\u793c\u76d2", count: 1},
	}
	rewards := make([]RoleItem, 0, len(rewardSpecs))
	for _, spec := range rewardSpecs {
		item, ok := CapturedRoleItemTemplate(spec.name)
		if !ok {
			return nil, false
		}
		item.Type = "\u80cc\u5305"
		item.Index = -1
		item.Count = spec.count
		rewards = append(rewards, item)
	}
	return rewards, true
}

func (store *Store) useLevel10GiftBoxLocked(playerID string, roles []RoleSummary, roleIndex int, sourceItem RoleItem) RoleUseItemResult {
	rewardItems, ok := classicTownLevel10GiftBoxRewards()
	if !ok {
		role := withRoleRuntimeDefaults(roles[roleIndex])
		return RoleUseItemResult{
			Role:         role,
			PlayerBase:   playerBaseDataFromRole(playerID, role),
			Item:         sourceItem,
			Found:        true,
			ErrorCode:    "target_item_missing",
			ErrorMessage: "gift box reward item missing",
		}
	}

	baseCapacity, supported := roleContainerCapacity(sourceItem.Type)
	if !supported {
		role := withRoleRuntimeDefaults(roles[roleIndex])
		return RoleUseItemResult{
			Role:         role,
			PlayerBase:   playerBaseDataFromRole(playerID, role),
			Item:         sourceItem,
			Found:        true,
			ErrorCode:    "invalid_container",
			ErrorMessage: "invalid target container",
		}
	}

	capacity := effectiveRoleContainerCapacity(roles[roleIndex].Items, sourceItem.Type, baseCapacity)
	updatedItems, updatedSource, clearedItems := consumeRoleItemBySlot(roles[roleIndex].Items, sourceItem.Type, sourceItem.Index, 1)
	updatedResults := []RoleItem{}
	if updatedSource != nil {
		updatedResults = append(updatedResults, *updatedSource)
	}
	for _, reward := range rewardItems {
		reward.Type = sourceItem.Type
		reward.Index = -1
		var granted RoleItem
		updatedItems, granted, ok = grantRoleItemToItems(updatedItems, capacity, reward)
		if !ok {
			role := withRoleRuntimeDefaults(roles[roleIndex])
			return RoleUseItemResult{
				Role:         role,
				PlayerBase:   playerBaseDataFromRole(playerID, role),
				Item:         sourceItem,
				UpdatedItems: []RoleItem{normalizeRoleItem(sourceItem)},
				Found:        true,
				ErrorCode:    classicTownLevel10GiftBoxFullCode,
				ErrorMessage: classicTownGiftBoxBagFullError,
			}
		}
		updatedResults = append(updatedResults, granted)
	}

	roles[roleIndex].Items = normalizeRoleItems(updatedItems)
	store.rolesByPID[playerID] = roles
	if err := store.persistPlayerStateLocked(playerID); err != nil {
		log.Printf("[session.Store] persist used level 10 gift box failed: %v", err)
	}

	role := withRoleRuntimeDefaults(roles[roleIndex])
	return RoleUseItemResult{
		Role:         role,
		PlayerBase:   playerBaseDataFromRole(playerID, role),
		Item:         sourceItem,
		UpdatedItems: normalizeRoleItems(updatedResults),
		ClearedItems: clearedItems,
		Found:        true,
		Used:         true,
	}
}

func classicTownLevel10GiftBoxRewards() ([]RoleItem, bool) {
	rewardSpecs := []struct {
		name  string
		count int
	}{
		{name: "\u004c\u521d\u9636\u7ecf\u9a8c\u5361", count: 1},
		{name: "\u004c\u82b1\u5377", count: 3},
		{name: "\u004c\u80cc\u5305\u8865\u4e01", count: 1},
		{name: "\u0031\u0035\u7ea7\u793c\u76d2", count: 1},
	}
	rewards := make([]RoleItem, 0, len(rewardSpecs))
	for _, spec := range rewardSpecs {
		item, ok := CapturedRoleItemTemplate(spec.name)
		if !ok {
			return nil, false
		}
		item.Type = "\u80cc\u5305"
		item.Index = -1
		item.Count = spec.count
		if item.Name == "\u004c\u80cc\u5305\u8865\u4e01" {
			item.Display = "560.png"
			item.Description = "f_i_L\u80cc\u5305\u8865\u4e01^00ccff&24@\u7279\u6b8a&25@99&19@\u4f7f\u7528\u540e\u589e\u52a06\u683c\u80cc\u5305\u5bb9\u79ef,\u6269\u5927\u4e0a\u9650\u4e3a90\u683c&20@\u7f1d\u5236\u548c\u6269\u5c55\u80cc\u5305\u7684\u4e13\u7528\u8865\u4e01.&27@sitem_ezhj&103@0&104@0&105@&107@&108@0"
			item.ItemType = "own"
			item.ItemLevel = 3
		}
		rewards = append(rewards, item)
	}
	return rewards, true
}

func isAvoidMonsterBuffItem(item RoleItem) bool {
	name := strings.TrimSpace(item.Name)
	return name == classicTownAvoidBuffItemName || name == classicTownAvoidBuffLocalItemName
}

func (store *Store) useInitialExperienceCardLocked(
	playerID string,
	roles []RoleSummary,
	roleIndex int,
	sourceItem RoleItem,
) RoleUseItemResult {
	return store.useExperienceBoostCardLocked(
		playerID,
		roles,
		roleIndex,
		sourceItem,
		roleTownInitialExperienceBoostDuration,
		classicTownInitialExperienceBoostSourceCapture,
	)
}

func (store *Store) useAdvancedExperienceCardLocked(
	playerID string,
	roles []RoleSummary,
	roleIndex int,
	sourceItem RoleItem,
) RoleUseItemResult {
	return store.useExperienceBoostCardLocked(
		playerID,
		roles,
		roleIndex,
		sourceItem,
		roleTownAdvancedExperienceBoostDuration,
		classicTownAdvancedExperienceBoostSourceCapture,
	)
}

func (store *Store) useExperienceBoostCardLocked(
	playerID string,
	roles []RoleSummary,
	roleIndex int,
	sourceItem RoleItem,
	duration time.Duration,
	sourceCapture string,
) RoleUseItemResult {
	updatedItems, updatedSource, clearedItems := consumeRoleItemBySlot(roles[roleIndex].Items, sourceItem.Type, sourceItem.Index, 1)
	nowMs := time.Now().UnixMilli()
	buff := RoleTownBuff{
		Handle:        roles[roleIndex].RoleID,
		Name:          classicTownInitialExperienceBoostName,
		Display:       classicTownInitialExperienceBoostDisplay,
		Description:   classicTownInitialExperienceBoostDescription,
		BattleOnly:    0,
		EndTime:       nowMs + int64(duration/time.Millisecond),
		SourceCapture: sourceCapture,
		Partial:       true,
	}
	activeBuffs, _ := filterActiveRoleTownBuffs(roles[roleIndex].TownBuffs, nowMs)
	nextBuffs := make([]RoleTownBuff, 0, len(activeBuffs)+1)
	for _, existing := range activeBuffs {
		if existing.Name != buff.Name {
			nextBuffs = append(nextBuffs, existing)
		}
	}
	nextBuffs = append(nextBuffs, buff)

	roles[roleIndex].Items = normalizeRoleItems(updatedItems)
	roles[roleIndex].TownBuffs = normalizeRoleTownBuffs(nextBuffs)
	store.rolesByPID[playerID] = roles
	if err := store.persistPlayerStateLocked(playerID); err != nil {
		log.Printf("[session.Store] persist used experience boost card failed: %v", err)
	}

	role := withRoleRuntimeDefaults(roles[roleIndex])
	result := RoleUseItemResult{
		Role:         role,
		PlayerBase:   playerBaseDataFromRole(playerID, role),
		Item:         sourceItem,
		TownBuff:     &buff,
		ClearedItems: clearedItems,
		Found:        true,
		Used:         true,
	}
	if updatedSource != nil {
		result.UpdatedItems = []RoleItem{*updatedSource}
	}
	return result
}

func (store *Store) useAvoidMonsterBuffItemLocked(
	playerID string,
	roles []RoleSummary,
	roleIndex int,
	sourceItem RoleItem,
) RoleUseItemResult {
	updatedItems, updatedSource, clearedItems := consumeRoleItemBySlot(roles[roleIndex].Items, sourceItem.Type, sourceItem.Index, 1)
	buff := RoleTownBuff{
		Handle:        roles[roleIndex].RoleID,
		Name:          classicTownAvoidBuffName,
		Display:       strings.TrimSpace(sourceItem.Display),
		Description:   classicTownAvoidBuffDescription,
		BattleOnly:    0,
		EndTime:       time.Now().Add(roleTownAvoidBuffDuration).UnixMilli(),
		SourceCapture: classicTownAvoidBuffSourceCapture,
		Partial:       true,
	}
	if buff.Display == "" {
		buff.Display = "574.png"
	}
	activeBuffs, _ := filterActiveRoleTownBuffs(roles[roleIndex].TownBuffs, time.Now().UnixMilli())
	nextBuffs := make([]RoleTownBuff, 0, len(activeBuffs)+1)
	for _, existing := range activeBuffs {
		if existing.Name != buff.Name {
			nextBuffs = append(nextBuffs, existing)
		}
	}
	nextBuffs = append(nextBuffs, buff)

	roles[roleIndex].Items = normalizeRoleItems(updatedItems)
	roles[roleIndex].TownBuffs = normalizeRoleTownBuffs(nextBuffs)
	store.rolesByPID[playerID] = roles
	if err := store.persistPlayerStateLocked(playerID); err != nil {
		log.Printf("[session.Store] persist used avoid buff item failed: %v", err)
	}

	role := withRoleRuntimeDefaults(roles[roleIndex])
	result := RoleUseItemResult{
		Role:         role,
		PlayerBase:   playerBaseDataFromRole(playerID, role),
		Item:         sourceItem,
		TownBuff:     &buff,
		ClearedItems: clearedItems,
		Found:        true,
		Used:         true,
	}
	if updatedSource != nil {
		result.UpdatedItems = []RoleItem{*updatedSource}
	}
	return result
}

func (store *Store) useSkillItemLocked(
	playerID string,
	roles []RoleSummary,
	roleIndex int,
	sourceItem RoleItem,
	skill RoleSkill,
) RoleUseItemResult {
	skill = normalizeRoleSkill(skill)
	if skill.Name == "" {
		role := withRoleRuntimeDefaults(roles[roleIndex])
		return RoleUseItemResult{
			Role:         role,
			PlayerBase:   playerBaseDataFromRole(playerID, role),
			Item:         sourceItem,
			Found:        true,
			ErrorCode:    "invalid_skill",
			ErrorMessage: "技能不存在。",
		}
	}

	currentSkills := cloneRoleSkills(roles[roleIndex].Skills)
	maxLevel := normalizeSkillMaxLevel(skill.MaxLevel)
	targetSkillIndex := -1
	for index, existing := range roles[roleIndex].Skills {
		existing = normalizeRoleSkill(existing)
		if existing.Name != skill.Name {
			continue
		}
		if existing.Level >= maxLevel {
			role := withRoleRuntimeDefaults(roles[roleIndex])
			return RoleUseItemResult{
				Role:         role,
				PlayerBase:   playerBaseDataFromRole(playerID, role),
				Item:         sourceItem,
				Found:        true,
				ErrorCode:    "skill_level_max",
				ErrorMessage: "该技能已达到当前最高等级。",
			}
		}
		targetSkillIndex = index
		skill.Level = existing.Level + 1
		break
	}
	skill = roleSkillWithCapturedActiveItemPresentation(skill)

	if targetSkillIndex < 0 && len(roles[roleIndex].Skills) >= defaultSkillCap {
		role := withRoleRuntimeDefaults(roles[roleIndex])
		return RoleUseItemResult{
			Role:         role,
			PlayerBase:   playerBaseDataFromRole(playerID, role),
			Item:         sourceItem,
			Found:        true,
			ErrorCode:    "skill_cap_full",
			ErrorMessage: "可学习技能数量已满。",
		}
	}

	updatedItems, updatedSource, clearedItems := consumeRoleItemBySlot(roles[roleIndex].Items, sourceItem.Type, sourceItem.Index, 1)
	if updatedSource != nil {
		updatedSource.Handle = roles[roleIndex].RoleID
	}
	if targetSkillIndex >= 0 {
		roles[roleIndex].Skills[targetSkillIndex] = skill
	} else {
		roles[roleIndex].Skills = append(currentSkills, skill)
	}
	roles[roleIndex].Items = normalizeRoleItems(updatedItems)
	store.rolesByPID[playerID] = roles
	if err := store.persistPlayerStateLocked(playerID); err != nil {
		log.Printf("[session.Store] persist learned skill item failed: %v", err)
	}

	role := withRoleRuntimeDefaults(roles[roleIndex])
	result := RoleUseItemResult{
		Role:         role,
		PlayerBase:   playerBaseDataFromRole(playerID, role),
		Item:         sourceItem,
		LearnedSkill: &skill,
		ClearedItems: clearedItems,
		Found:        true,
		Used:         true,
	}
	if updatedSource != nil {
		result.UpdatedItems = []RoleItem{*updatedSource}
	}
	return result
}

func roleSkillWithCapturedActiveItemPresentation(skill RoleSkill) RoleSkill {
	if skill.Name == "\u6b66\u5668\u5a34\u719f" && skill.Level == 1 {
		skill.Type = "null"
		skill.Icon = "226.png"
		skill.Description = "f_s_\u6b66\u5668\u5a34\u719f^ffffff&9@\u88ab\u52a8&8@\u6e38\u4fa0 &10@\u901a\u7528&12@8"
	}
	return skill
}

func roleSkillFromItem(item RoleItem) (RoleSkill, bool) {
	name := strings.TrimSpace(item.Name)
	itemType := strings.TrimSpace(item.ItemType)
	if name == "\u6b66\u5668\u5a34\u719f" && strings.Contains(item.Description, "&24@\u88ab\u52a8\u6280\u80fd") {
		itemType = "\u88ab\u52a8\u6280\u80fd"
	}
	if itemType != "被动技能" && !strings.HasPrefix(itemType, "技能") {
		return RoleSkill{}, false
	}
	if name == "" {
		return RoleSkill{}, false
	}
	maxLevel := item.ItemLevel
	if maxLevel <= 0 {
		maxLevel = 5
	}
	return RoleSkill{
		Name:        name,
		Level:       1,
		Type:        itemType,
		Icon:        strings.TrimSpace(item.Display),
		Description: strings.TrimSpace(item.Description),
		MaxLevel:    maxLevel,
	}, true
}

func roleItemRecoveryAmounts(item RoleItem) (int, int, bool) {
	healHP := maxInt(0, parseClassicDescriptionSignedInt(item.Description, "7"))
	healMP := maxInt(0, parseClassicDescriptionSignedInt(item.Description, "8"))
	return healHP, healMP, healHP > 0 || healMP > 0
}

func (store *Store) useRecoveryItemLocked(
	playerID string,
	roles []RoleSummary,
	roleIndex int,
	sourceItem RoleItem,
	healHP int,
	healMP int,
) RoleUseItemResult {
	roleState := defaultRoleState(roles[roleIndex].RoleID, roles[roleIndex].Level, roles[roleIndex].Exp)
	if roles[roleIndex].RoleState != nil {
		roleState = *roles[roleIndex].RoleState
		if roleState.Handle == "" {
			roleState.Handle = roles[roleIndex].RoleID
		}
	}
	roleState.Exp = roles[roleIndex].Exp
	roleState.Lv = roles[roleIndex].Level
	roleState.Speed = ClassicRoleSpeed(roles[roleIndex].Level)

	rolePhysique := defaultRolePhysique(roles[roleIndex])
	if roles[roleIndex].RolePhysique != nil {
		rolePhysique = *roles[roleIndex].RolePhysique
		if rolePhysique.Handle == "" {
			rolePhysique.Handle = roles[roleIndex].RoleID
		}
	}
	canRecoverHP := healHP > 0 && roleState.HP < rolePhysique.MaxHP
	canRecoverMP := healMP > 0 && roleState.MP < rolePhysique.MaxMP
	if !canRecoverHP && !canRecoverMP {
		role := withRoleRuntimeDefaults(roles[roleIndex])
		return RoleUseItemResult{
			Role:         role,
			PlayerBase:   playerBaseDataFromRole(playerID, role),
			Item:         sourceItem,
			Found:        true,
			ErrorCode:    "role_state_full",
			ErrorMessage: "当前状态不需要使用该物品。",
		}
	}
	if healHP > 0 {
		roleState.HP = clampRoleRuntimeValue(roleState.HP+healHP, 0, rolePhysique.MaxHP)
	}
	if healMP > 0 {
		roleState.MP = clampRoleRuntimeValue(roleState.MP+healMP, 0, rolePhysique.MaxMP)
	}

	updatedItems, updatedSource, clearedItems := consumeRoleItemBySlot(roles[roleIndex].Items, sourceItem.Type, sourceItem.Index, 1)
	roles[roleIndex].Items = normalizeRoleItems(updatedItems)
	roles[roleIndex].RoleState = &roleState
	roles[roleIndex].RolePhysique = &rolePhysique
	store.rolesByPID[playerID] = roles
	if err := store.persistRoleStateLocked(playerID, roles[roleIndex].RoleID); err != nil {
		log.Printf("[session.Store] persist used recovery item failed: %v", err)
	}

	role := withRoleRuntimeDefaults(roles[roleIndex])
	result := RoleUseItemResult{
		Role:             role,
		PlayerBase:       playerBaseDataFromRole(playerID, role),
		Item:             sourceItem,
		ClearedItems:     clearedItems,
		Found:            true,
		Used:             true,
		RoleStateChanged: true,
	}
	if updatedSource != nil {
		result.UpdatedItems = []RoleItem{*updatedSource}
	}
	return result
}

func clampRoleRuntimeValue(value int, minValue int, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func ClassicTownHealerCost(level int, missingHP int, missingMP int) int {
	if level < 15 {
		return 0
	}
	missingTotal := maxInt(0, missingHP) + maxInt(0, missingMP)
	if missingTotal <= 0 {
		return 0
	}
	rawCost := (missingTotal*23 + 50) / 100
	cost := ((rawCost + 2) / 5) * 5
	if cost < 5 {
		return 5
	}
	return cost
}

func (store *Store) useCurrencyExchangeItemLocked(
	playerID string,
	roles []RoleSummary,
	roleIndex int,
	sourceItem RoleItem,
	sourceCount int,
	targetName string,
	targetCount int,
) RoleUseItemResult {
	if sourceItem.Count < sourceCount {
		role := withRoleRuntimeDefaults(roles[roleIndex])
		return RoleUseItemResult{
			Role:         role,
			PlayerBase:   playerBaseDataFromRole(playerID, role),
			Item:         sourceItem,
			Found:        true,
			ErrorCode:    "item_not_enough",
			ErrorMessage: "物品数量不足。",
		}
	}

	currencies := cloneRoleCurrencies(roles[roleIndex].Currencies)
	if sourceItem.Name == "银元宝" {
		if currencies["银元宝"] < sourceCount {
			role := withRoleRuntimeDefaults(roles[roleIndex])
			return RoleUseItemResult{
				Role:         role,
				PlayerBase:   playerBaseDataFromRole(playerID, role),
				Item:         sourceItem,
				Found:        true,
				ErrorCode:    "currency_not_enough",
				ErrorMessage: "银元宝不足。",
			}
		}
		currencies["银元宝"] -= sourceCount
		currencies["铜钱"] += targetCount
	} else if sourceItem.Name == "铜钱" {
		if currencies["铜钱"] < sourceCount {
			role := withRoleRuntimeDefaults(roles[roleIndex])
			return RoleUseItemResult{
				Role:         role,
				PlayerBase:   playerBaseDataFromRole(playerID, role),
				Item:         sourceItem,
				Found:        true,
				ErrorCode:    "currency_not_enough",
				ErrorMessage: "铜钱不足。",
			}
		}
		currencies["铜钱"] -= sourceCount
		currencies["银元宝"] += targetCount
	}

	updatedItems, updatedSource, clearedItems := consumeRoleItemBySlot(roles[roleIndex].Items, sourceItem.Type, sourceItem.Index, sourceCount)
	updatedResults := []RoleItem{}
	if updatedSource != nil {
		updatedResults = append(updatedResults, *updatedSource)
	}
	targetTemplate, ok := CapturedRoleItemTemplate(targetName)
	if !ok {
		role := withRoleRuntimeDefaults(roles[roleIndex])
		return RoleUseItemResult{
			Role:         role,
			PlayerBase:   playerBaseDataFromRole(playerID, role),
			Item:         sourceItem,
			Found:        true,
			ErrorCode:    "target_item_missing",
			ErrorMessage: "兑换目标物品不存在。",
		}
	}
	targetTemplate.Type = sourceItem.Type
	targetTemplate.Index = -1
	targetTemplate.Count = targetCount
	updatedItems, grantedItem, ok := grantRoleItemToItems(updatedItems, defaultBagCap, targetTemplate)
	if !ok {
		role := withRoleRuntimeDefaults(roles[roleIndex])
		return RoleUseItemResult{
			Role:         role,
			PlayerBase:   playerBaseDataFromRole(playerID, role),
			Item:         sourceItem,
			Found:        true,
			ErrorCode:    "bag_full",
			ErrorMessage: "背包已满。",
		}
	}
	updatedResults = append(updatedResults, grantedItem)

	roles[roleIndex].Items = normalizeRoleItems(updatedItems)
	roles[roleIndex].Currencies = normalizeRoleCurrencies(currencies)
	store.rolesByPID[playerID] = roles
	if err := store.persistPlayerStateLocked(playerID); err != nil {
		log.Printf("[session.Store] persist used role item failed: %v", err)
	}

	role := withRoleRuntimeDefaults(roles[roleIndex])
	return RoleUseItemResult{
		Role:         role,
		PlayerBase:   playerBaseDataFromRole(playerID, role),
		Item:         sourceItem,
		UpdatedItems: normalizeRoleItems(updatedResults),
		ClearedItems: clearedItems,
		Currencies:   cloneRoleCurrencies(role.Currencies),
		Found:        true,
		Used:         true,
	}
}

func (store *Store) useEquipmentItemLocked(
	playerID string,
	roles []RoleSummary,
	roleIndex int,
	sourceItem RoleItem,
	sourceType string,
	sourceIndex int,
	targetIndex int,
) RoleUseItemResult {
	updatedItems := make([]RoleItem, 0, len(roles[roleIndex].Items)+1)
	updatedResultItems := make([]RoleItem, 0, 2)
	var replacedItem *RoleItem
	for _, item := range roles[roleIndex].Items {
		if item.Type == sourceType && item.Index == sourceIndex {
			continue
		}
		if item.Type == "装备" && item.Index == targetIndex {
			copyItem := item
			replacedItem = &copyItem
			continue
		}
		updatedItems = append(updatedItems, item)
	}

	if replacedItem != nil && sourceType != "装备" {
		replacedItem.Type = sourceType
		replacedItem.Index = sourceIndex
		normalized := normalizeRoleItem(*replacedItem)
		updatedItems = append(updatedItems, normalized)
		updatedResultItems = append(updatedResultItems, normalized)
	}

	equippedItem := sourceItem
	equippedItem.Type = "装备"
	equippedItem.Index = targetIndex
	normalizedEquippedItem := normalizeRoleItem(equippedItem)
	updatedItems = append(updatedItems, normalizedEquippedItem)
	updatedResultItems = append(updatedResultItems, normalizedEquippedItem)
	roles[roleIndex].Items = normalizeRoleItems(updatedItems)
	roles[roleIndex].SourceQuery = rebuildRoleEquipmentAppearanceSourceQuery(roles[roleIndex].SourceQuery, roles[roleIndex].Items)
	roles[roleIndex] = syncRoleProgressionRuntimeData(roles[roleIndex])
	store.rolesByPID[playerID] = roles
	if err := store.persistPlayerStateLocked(playerID); err != nil {
		log.Printf("[session.Store] persist active equipped item failed: %v", err)
	}

	role := withRoleRuntimeDefaults(roles[roleIndex])
	return RoleUseItemResult{
		Role:         role,
		PlayerBase:   playerBaseDataFromRole(playerID, role),
		Item:         sourceItem,
		UpdatedItems: updatedResultItems,
		ClearedItems: []RoleItemClear{{
			Type:  sourceType,
			Index: sourceIndex,
		}},
		Found:    true,
		Used:     true,
		Equipped: true,
	}
}

func (store *Store) PurchaseMallProduct(playerID string, roleID string, product mall.Product, quantity int, requestID string) mall.PurchaseResult {
	store.mu.Lock()
	defer store.mu.Unlock()

	requestID = strings.TrimSpace(requestID)
	if requestID != "" {
		key := roleID + ":" + requestID
		if previous, ok := store.mallRequests[key]; ok {
			previous.ErrorCode = mall.DUPLICATE_REQUEST
			previous.ErrorMessage = "重复购买请求。"
			return previous
		}
	}
	if quantity <= 0 || quantity > 99 {
		return mall.PurchaseResult{
			Success:      false,
			ProductID:    product.ProductID,
			Quantity:     quantity,
			RequestID:    requestID,
			ErrorCode:    mall.INVALID_QUANTITY,
			ErrorMessage: "购买数量不合法。",
		}
	}

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}
		roles[index] = withRoleRuntimeDefaults(roles[index])
		totalPrice := product.Price * quantity
		if roles[index].Currencies[product.Currency] < totalPrice {
			return mall.PurchaseResult{
				Success:         false,
				ProductID:       product.ProductID,
				Quantity:        quantity,
				RequestID:       requestID,
				CurrencyName:    product.Currency,
				CurrencyBalance: roles[index].Currencies[product.Currency],
				ErrorCode:       mall.INSUFFICIENT_CURRENCY,
				ErrorMessage:    product.Currency + "不足。",
			}
		}
		targetIndex, ok := nextRoleItemIndex(roles[index].Items, "商城", defaultMallCap)
		if !ok {
			return mall.PurchaseResult{
				Success:      false,
				ProductID:    product.ProductID,
				Quantity:     quantity,
				RequestID:    requestID,
				ErrorCode:    "MALL_BAG_FULL",
				ErrorMessage: "商城背包已满。",
			}
		}
		roles[index].Currencies[product.Currency] -= totalPrice
		roles[index].Items = normalizeRoleItems(append(roles[index].Items, RoleItem{
			Type:        "商城",
			Name:        product.Name,
			ItemType:    "mall",
			Display:     product.Icon,
			Description: product.Description,
			Count:       quantity,
			Index:       targetIndex,
			Level:       1,
			EndTime:     0,
			Owner:       roles[index].DisplayName,
			ItemLevel:   1,
		}))
		store.rolesByPID[playerID] = roles
		if err := store.persistPlayerStateLocked(playerID); err != nil {
			log.Printf("[session.Store] persist mall purchase failed: %v", err)
		}
		result := mall.PurchaseResult{
			Success:         true,
			ProductID:       product.ProductID,
			Quantity:        quantity,
			RequestID:       requestID,
			CurrencyName:    product.Currency,
			CurrencyBalance: roles[index].Currencies[product.Currency],
		}
		if requestID != "" {
			store.mallRequests[roleID+":"+requestID] = result
		}
		return result
	}

	return mall.PurchaseResult{
		Success:      false,
		ProductID:    product.ProductID,
		Quantity:     quantity,
		RequestID:    requestID,
		ErrorCode:    "ROLE_NOT_FOUND",
		ErrorMessage: "角色不存在。",
	}
}

func (store *Store) EquipRoleItem(playerID string, roleID string, sourceType string, sourceIndex int, count int) RoleEquipItemResult {
	store.mu.Lock()
	defer store.mu.Unlock()

	sourceType = strings.TrimSpace(sourceType)
	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		roles[index] = withRoleRuntimeDefaults(roles[index])
		sourceItem, ok := findRoleItem(roles[index].Items, sourceType, sourceIndex)
		if !ok {
			return RoleEquipItemResult{
				Role:         withRoleRuntimeDefaults(roles[index]),
				PlayerBase:   playerBaseDataFromRole(playerID, roles[index]),
				Found:        true,
				ErrorCode:    "item_missing",
				ErrorMessage: "物品不存在。",
			}
		}

		targetIndex, ok := roleEquipTargetIndex(sourceItem)
		if !ok {
			return RoleEquipItemResult{
				Role:         withRoleRuntimeDefaults(roles[index]),
				PlayerBase:   playerBaseDataFromRole(playerID, roles[index]),
				Found:        true,
				ErrorCode:    "not_equipment",
				ErrorMessage: "该物品不能装备。",
			}
		}

		updatedItems := make([]RoleItem, 0, len(roles[index].Items)+1)
		updatedResultItems := make([]RoleItem, 0, 1)
		var replacedItem *RoleItem
		for _, item := range roles[index].Items {
			if item.Type == sourceType && item.Index == sourceIndex {
				continue
			}
			if item.Type == "装备" && item.Index == targetIndex {
				copyItem := item
				replacedItem = &copyItem
				continue
			}
			updatedItems = append(updatedItems, item)
		}

		if replacedItem != nil && sourceType != "装备" {
			replacedItem.Type = sourceType
			replacedItem.Index = sourceIndex
			normalized := normalizeRoleItem(*replacedItem)
			updatedItems = append(updatedItems, normalized)
			updatedResultItems = append(updatedResultItems, normalized)
		}

		equippedItem := sourceItem
		equippedItem.Type = "装备"
		equippedItem.Index = targetIndex
		if count > 0 && count < equippedItem.Count {
			equippedItem.Count = count
		}
		updatedItems = append(updatedItems, normalizeRoleItem(equippedItem))
		roles[index].Items = normalizeRoleItems(updatedItems)
		roles[index].SourceQuery = rebuildRoleEquipmentAppearanceSourceQuery(roles[index].SourceQuery, roles[index].Items)
		roles[index] = syncRoleProgressionRuntimeData(roles[index])
		store.rolesByPID[playerID] = roles
		if err := store.persistPlayerStateLocked(playerID); err != nil {
			log.Printf("[session.Store] persist equipped item failed: %v", err)
		}

		role := withRoleRuntimeDefaults(roles[index])
		return RoleEquipItemResult{
			Role:         role,
			PlayerBase:   playerBaseDataFromRole(playerID, role),
			EquippedItem: equippedItem,
			UpdatedItems: updatedResultItems,
			ClearedItems: []RoleItemClear{{
				Type:  sourceType,
				Index: sourceIndex,
			}},
			Found:    true,
			Equipped: true,
		}
	}

	return RoleEquipItemResult{
		Found:        false,
		ErrorCode:    "role_missing",
		ErrorMessage: "角色不存在。",
	}
}

func (store *Store) PreviewTryEquip(playerID string, roleID string, itemName string) RoleTryEquipPreviewResult {
	store.mu.Lock()
	defer store.mu.Unlock()

	itemName = strings.TrimSpace(itemName)
	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		role := withRoleRuntimeDefaults(roles[index])
		sourceItem, ok := findRoleItemByName(role.Items, itemName)
		if !ok {
			sourceItem, ok = CapturedRoleItemTemplate(itemName)
		}
		if !ok {
			return RoleTryEquipPreviewResult{
				Role:         role,
				PlayerBase:   playerBaseDataFromRole(playerID, role),
				Found:        true,
				ErrorCode:    "item_missing",
				ErrorMessage: "item missing",
			}
		}

		sourceItem = normalizeRoleItem(sourceItem)
		targetIndex, ok := roleEquipTargetIndex(sourceItem)
		if !ok {
			return RoleTryEquipPreviewResult{
				Role:         role,
				PlayerBase:   playerBaseDataFromRole(playerID, role),
				Item:         sourceItem,
				Found:        true,
				ErrorCode:    "not_equipment",
				ErrorMessage: "not equipment",
			}
		}

		replacementIndices := roleTryEquipReplacementIndices(sourceItem, targetIndex)
		previewItems := make([]RoleItem, 0, len(role.Items)+1)
		for _, item := range role.Items {
			if item.Type == "装备" && roleItemIndexMatches(item.Index, replacementIndices) {
				continue
			}
			previewItems = append(previewItems, item)
		}
		previewItem := sourceItem
		previewItem.Type = "装备"
		previewItem.Index = targetIndex
		previewItems = append(previewItems, normalizeRoleItem(previewItem))

		sourceQuery := rebuildRoleEquipmentAppearanceSourceQuery(role.SourceQuery, previewItems)
		return RoleTryEquipPreviewResult{
			Role:        role,
			PlayerBase:  playerBaseDataFromRole(playerID, role),
			Item:        sourceItem,
			SourceQuery: sourceQuery,
			Found:       true,
			Previewed:   true,
		}
	}

	return RoleTryEquipPreviewResult{
		Found:        false,
		ErrorCode:    "role_missing",
		ErrorMessage: "role missing",
	}
}

func (store *Store) MoveRoleItem(playerID string, roleID string, sourceType string, sourceIndex int, targetType string, targetIndex int, count int) (result RoleMoveItemResult) {
	sourceType = strings.TrimSpace(sourceType)
	targetType = strings.TrimSpace(targetType)

	rolePersistLock := store.rolePersistenceLock(playerID, roleID)
	rolePersistLock.Lock()
	defer rolePersistLock.Unlock()

	store.mu.Lock()
	var roleSnapshot RoleSummary
	shouldPersist := false
	defer func() {
		store.mu.Unlock()
		if shouldPersist {
			if err := store.persistRoleStateSnapshot(playerID, roleID, roleSnapshot); err != nil {
				log.Printf("[session.Store] persist moved item failed: %v", err)
			}
		}
	}()

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		roles[index] = withRoleRuntimeDefaults(roles[index])
		baseCapacity, supported := roleContainerCapacity(targetType)
		capacity := effectiveRoleContainerCapacity(roles[index].Items, targetType, baseCapacity)
		if supported && targetIndex < 0 {
			if nextIndex, ok := nextRoleItemIndex(roles[index].Items, targetType, capacity); ok {
				targetIndex = nextIndex
			}
		}
		if !supported || targetIndex < 0 || targetIndex >= capacity {
			return RoleMoveItemResult{
				Role:         withRoleRuntimeDefaults(roles[index]),
				PlayerBase:   playerBaseDataFromRole(playerID, roles[index]),
				Found:        true,
				ErrorCode:    "target_invalid",
				ErrorMessage: "目标位置无效。",
			}
		}

		_, ok := findRoleItem(roles[index].Items, sourceType, sourceIndex)
		if !ok {
			return RoleMoveItemResult{
				Role:         withRoleRuntimeDefaults(roles[index]),
				PlayerBase:   playerBaseDataFromRole(playerID, roles[index]),
				Found:        true,
				ErrorCode:    "item_missing",
				ErrorMessage: "物品不存在。",
			}
		}
		if sourceType == targetType && sourceIndex == targetIndex {
			return RoleMoveItemResult{
				Role:       withRoleRuntimeDefaults(roles[index]),
				PlayerBase: playerBaseDataFromRole(playerID, roles[index]),
				Found:      true,
			}
		}

		sourceItem, _ := findRoleItem(roles[index].Items, sourceType, sourceIndex)
		if count <= 0 || count > sourceItem.Count {
			count = sourceItem.Count
		}
		moveCount := count
		targetItem, hasTarget := findRoleItem(roles[index].Items, targetType, targetIndex)
		canStack := hasTarget && sourceType == targetType && strings.TrimSpace(sourceItem.Name) != "" && sourceItem.Name == targetItem.Name
		canSplitMove := moveCount < sourceItem.Count && !hasTarget
		updatedItems := make([]RoleItem, 0, len(roles[index].Items)+1)
		updatedResultItems := make([]RoleItem, 0, 3)
		for _, item := range roles[index].Items {
			switch {
			case canStack && item.Type == sourceType && item.Index == sourceIndex:
				if sourceItem.Count > moveCount {
					remaining := item
					remaining.Count = sourceItem.Count - moveCount
					updatedItems = append(updatedItems, normalizeRoleItem(remaining))
					updatedResultItems = append(updatedResultItems, normalizeRoleItem(remaining))
				}
			case canStack && item.Type == targetType && item.Index == targetIndex:
				stacked := item
				stacked.Count += moveCount
				updatedItems = append(updatedItems, normalizeRoleItem(stacked))
				updatedResultItems = append(updatedResultItems, normalizeRoleItem(stacked))
			case canSplitMove && item.Type == sourceType && item.Index == sourceIndex:
				remaining := item
				remaining.Count = sourceItem.Count - moveCount
				updatedItems = append(updatedItems, normalizeRoleItem(remaining))
				updatedResultItems = append(updatedResultItems, normalizeRoleItem(remaining))
				moved := item
				moved.Type = targetType
				moved.Index = targetIndex
				moved.Count = moveCount
				updatedItems = append(updatedItems, normalizeRoleItem(moved))
				updatedResultItems = append(updatedResultItems, normalizeRoleItem(moved))
			case item.Type == sourceType && item.Index == sourceIndex:
				moved := item
				moved.Type = targetType
				moved.Index = targetIndex
				moved.Count = moveCount
				updatedItems = append(updatedItems, normalizeRoleItem(moved))
				updatedResultItems = append(updatedResultItems, normalizeRoleItem(moved))
			case hasTarget && item.Type == targetType && item.Index == targetIndex:
				swapped := item
				swapped.Type = sourceType
				swapped.Index = sourceIndex
				updatedItems = append(updatedItems, normalizeRoleItem(swapped))
				updatedResultItems = append(updatedResultItems, normalizeRoleItem(swapped))
			default:
				updatedItems = append(updatedItems, item)
			}
		}

		roles[index].Items = normalizeRoleItems(updatedItems)
		if sourceType == "装备" || targetType == "装备" {
			roles[index].SourceQuery = rebuildRoleEquipmentAppearanceSourceQuery(roles[index].SourceQuery, roles[index].Items)
			roles[index] = syncRoleProgressionRuntimeData(roles[index])
		}
		store.rolesByPID[playerID] = roles
		roleSnapshot = roles[index]
		shouldPersist = true

		role := withRoleRuntimeDefaults(roles[index])
		cleared := make([]RoleItemClear, 0, 2)
		if !canSplitMove && !(canStack && sourceItem.Count > moveCount) {
			cleared = append(cleared, RoleItemClear{Type: sourceType, Index: sourceIndex})
		}
		if sourceType != targetType || sourceIndex != targetIndex {
			cleared = append(cleared, RoleItemClear{Type: targetType, Index: targetIndex})
		}
		if !hasTarget {
			_ = targetItem
		}
		return RoleMoveItemResult{
			Role:         role,
			PlayerBase:   playerBaseDataFromRole(playerID, role),
			UpdatedItems: updatedResultItems,
			ClearedItems: cleared,
			Found:        true,
			Moved:        true,
		}
	}

	return RoleMoveItemResult{
		Found:        false,
		ErrorCode:    "role_missing",
		ErrorMessage: "角色不存在。",
	}
}

func (store *Store) FinishRoleContainer(playerID string, roleID string, containerType string) (result RoleFinishContainerResult) {
	containerType = strings.TrimSpace(containerType)
	_, supported := roleContainerCapacity(containerType)
	if !supported || containerType != "背包" {
		return RoleFinishContainerResult{
			Found:        true,
			ErrorCode:    "container_unsupported",
			ErrorMessage: "容器类型不支持整理。",
		}
	}

	rolePersistLock := store.rolePersistenceLock(playerID, roleID)
	rolePersistLock.Lock()
	defer rolePersistLock.Unlock()

	store.mu.Lock()
	var roleSnapshot RoleSummary
	shouldPersist := false
	defer func() {
		store.mu.Unlock()
		if shouldPersist {
			if err := store.persistRoleStateSnapshot(playerID, roleID, roleSnapshot); err != nil {
				log.Printf("[session.Store] persist finished container failed: %v", err)
			}
		}
	}()

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		roles[index] = withRoleRuntimeDefaults(roles[index])
		containerItems := make([]RoleItem, 0, len(roles[index].Items))
		otherItems := make([]RoleItem, 0, len(roles[index].Items))
		for _, item := range roles[index].Items {
			item = normalizeRoleItem(item)
			if item.Type == containerType && item.Index >= 0 {
				containerItems = append(containerItems, item)
				continue
			}
			otherItems = append(otherItems, item)
		}

		sort.SliceStable(containerItems, func(left int, right int) bool {
			return containerItems[left].Index < containerItems[right].Index
		})

		originalItemsByIndex := make(map[int]RoleItem, len(containerItems))
		for _, item := range containerItems {
			originalItemsByIndex[item.Index] = item
		}

		removed := make([]bool, len(containerItems))
		stackChanged := false
		for targetIndex := range containerItems {
			if removed[targetIndex] {
				continue
			}
			stackLimit := classicItemStackLimit(containerItems[targetIndex])
			if stackLimit <= 1 || containerItems[targetIndex].Count >= stackLimit {
				continue
			}
			for sourceIndex := targetIndex + 1; sourceIndex < len(containerItems); sourceIndex += 1 {
				if removed[sourceIndex] || !canRoleItemsStack(containerItems[targetIndex], containerItems[sourceIndex]) {
					continue
				}
				space := stackLimit - containerItems[targetIndex].Count
				if space <= 0 {
					break
				}
				moveCount := containerItems[sourceIndex].Count
				if moveCount > space {
					moveCount = space
				}
				if moveCount <= 0 {
					continue
				}

				stackChanged = true
				containerItems[targetIndex].Count += moveCount
				containerItems[targetIndex] = normalizeRoleItem(containerItems[targetIndex])

				containerItems[sourceIndex].Count -= moveCount
				if containerItems[sourceIndex].Count <= 0 {
					removed[sourceIndex] = true
					continue
				}
				containerItems[sourceIndex] = normalizeRoleItem(containerItems[sourceIndex])
			}
		}

		compactedItems := make([]RoleItem, 0, len(containerItems))
		nextIndex := 0
		for itemIndex, item := range containerItems {
			if removed[itemIndex] {
				continue
			}
			item.Index = nextIndex
			item = normalizeRoleItem(item)
			compactedItems = append(compactedItems, item)
			nextIndex += 1
		}

		updatedItems := make([]RoleItem, 0, len(compactedItems))
		finalItemsByIndex := make(map[int]RoleItem, len(compactedItems))
		for _, item := range compactedItems {
			finalItemsByIndex[item.Index] = item
			originalItem, existed := originalItemsByIndex[item.Index]
			if !existed || !roleItemsEqualForContainerFinish(originalItem, item) {
				updatedItems = append(updatedItems, item)
			}
		}
		clearedItems := make([]RoleItemClear, 0, len(originalItemsByIndex))
		for originalIndex := range originalItemsByIndex {
			if _, exists := finalItemsByIndex[originalIndex]; !exists {
				clearedItems = append(clearedItems, RoleItemClear{Type: containerType, Index: originalIndex})
			}
		}
		sort.SliceStable(clearedItems, func(left int, right int) bool {
			return clearedItems[left].Index < clearedItems[right].Index
		})
		changed := stackChanged || len(clearedItems) > 0 || len(updatedItems) > 0
		if !changed {
			role := withRoleRuntimeDefaults(roles[index])
			return RoleFinishContainerResult{
				Role:       role,
				PlayerBase: playerBaseDataFromRole(playerID, role),
				Found:      true,
			}
		}

		normalizedItems := make([]RoleItem, 0, len(otherItems)+len(compactedItems))
		normalizedItems = append(normalizedItems, otherItems...)
		normalizedItems = append(normalizedItems, compactedItems...)
		roles[index].Items = normalizeRoleItems(normalizedItems)
		store.rolesByPID[playerID] = roles
		roleSnapshot = roles[index]
		shouldPersist = true

		role := withRoleRuntimeDefaults(roles[index])
		sort.SliceStable(updatedItems, func(left int, right int) bool {
			return updatedItems[left].Index < updatedItems[right].Index
		})
		return RoleFinishContainerResult{
			Role:         role,
			PlayerBase:   playerBaseDataFromRole(playerID, role),
			UpdatedItems: normalizeRoleItems(updatedItems),
			ClearedItems: clearedItems,
			Found:        true,
			Changed:      true,
		}
	}

	return RoleFinishContainerResult{
		Found:        false,
		ErrorCode:    "role_missing",
		ErrorMessage: "角色不存在。",
	}
}

func roleItemsEqualForContainerFinish(left RoleItem, right RoleItem) bool {
	left = normalizeRoleItem(left)
	right = normalizeRoleItem(right)
	return left.Type == right.Type &&
		left.Name == right.Name &&
		left.ItemType == right.ItemType &&
		left.Display == right.Display &&
		left.Description == right.Description &&
		left.Count == right.Count &&
		left.Index == right.Index &&
		left.Level == right.Level &&
		left.EndTime == right.EndTime &&
		left.Owner == right.Owner &&
		left.ItemLevel == right.ItemLevel &&
		rolePetItemStatesEqual(left.PetState, right.PetState)
}

func rolePetItemStatesEqual(left *RolePetItemState, right *RolePetItemState) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Level == right.Level &&
		left.Exp == right.Exp &&
		left.Fullness == right.Fullness &&
		left.PetID == right.PetID
}

func roleContainerCapacity(containerType string) (int, bool) {
	switch containerType {
	case "背包":
		return defaultBagCap, true
	case "装备":
		return defaultEquipCap, true
	case "商城":
		return defaultMallCap, true
	case "\u4ed3\u5e93":
		return defaultWarehouseCap, true
	default:
		return 0, false
	}
}

func effectiveRoleContainerCapacity(items []RoleItem, containerType string, baseCapacity int) int {
	if containerType != "背包" {
		return baseCapacity
	}
	capacity := baseCapacity
	for _, item := range items {
		if item.Type != containerType || item.Index < 0 {
			continue
		}
		required := item.Index + 1
		if required > capacity {
			capacity = roundUpBagCapacity(required)
		}
	}
	return capacity
}

func roundUpBagCapacity(required int) int {
	if required <= defaultBagCap {
		return defaultBagCap
	}
	remainder := required % bagCapacityStep
	if remainder == 0 {
		return required
	}
	return required + bagCapacityStep - remainder
}

func nextRoleItemIndex(items []RoleItem, containerType string, capacity int) (int, bool) {
	used := make(map[int]bool, len(items))
	for _, item := range items {
		if item.Type == containerType {
			used[item.Index] = true
		}
	}
	for index := 0; index < capacity; index += 1 {
		if !used[index] {
			return index, true
		}
	}
	return 0, false
}

func grantRoleItemToItems(items []RoleItem, capacity int, item RoleItem) ([]RoleItem, RoleItem, bool) {
	item = normalizeRoleItem(item)
	if item.Type == "" {
		item.Type = "背包"
	}
	if item.Count <= 0 {
		item.Count = 1
	}
	if item.Name == "" {
		return items, RoleItem{}, false
	}

	if item.Index < 0 {
		updatedItems := make([]RoleItem, 0, len(items)+1)
		remainingCount := item.Count
		var grantedItem RoleItem
		grantedItemSet := false
		for _, existing := range items {
			if remainingCount > 0 && canRoleItemsStack(existing, item) {
				stackLimit := classicItemStackLimit(existing)
				if existing.Count < stackLimit {
					addedCount := remainingCount
					if addedCount > stackLimit-existing.Count {
						addedCount = stackLimit - existing.Count
					}
					existing.Count += addedCount
					remainingCount -= addedCount
					existing = normalizeRoleItem(existing)
					if !grantedItemSet {
						grantedItem = existing
						grantedItemSet = true
					}
				}
			}
			updatedItems = append(updatedItems, existing)
		}
		if remainingCount > 0 {
			targetIndex, ok := nextRoleItemIndex(updatedItems, item.Type, capacity)
			if !ok {
				return items, RoleItem{}, false
			}
			item.Index = targetIndex
			item.Count = remainingCount
			normalizedItem := normalizeRoleItem(item)
			updatedItems = append(updatedItems, normalizedItem)
			if !grantedItemSet {
				grantedItem = normalizedItem
				grantedItemSet = true
			}
		}
		return updatedItems, grantedItem, grantedItemSet
	}

	updatedItems := make([]RoleItem, 0, len(items)+1)
	for _, existing := range items {
		if existing.Type == item.Type && existing.Index == item.Index {
			continue
		}
		updatedItems = append(updatedItems, existing)
	}
	updatedItems = append(updatedItems, normalizeRoleItem(item))
	return updatedItems, item, true
}

func findRoleItem(items []RoleItem, itemType string, index int) (RoleItem, bool) {
	for _, item := range items {
		if item.Type == itemType && item.Index == index {
			return item, true
		}
	}
	return RoleItem{}, false
}

func findRoleItemByName(items []RoleItem, name string) (RoleItem, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return RoleItem{}, false
	}
	for _, item := range items {
		if item.Name == name {
			return item, true
		}
	}
	return RoleItem{}, false
}

func findEquippedRolePetItem(items []RoleItem) (RoleItem, int, bool) {
	for index, item := range items {
		if item.Type == "装备" && item.Index == rolePetEquipIndex && isRolePetItem(item) {
			return normalizeRoleItem(item), index, true
		}
	}
	return RoleItem{}, -1, false
}

func isRolePetItem(item RoleItem) bool {
	return strings.Contains(item.Description, "&24@宠物") || strings.Contains(item.Description, "&27@sitem_pet")
}

func buildRolePetInfoResult(playerID string, role RoleSummary, petItem RoleItem) RolePetInfoResult {
	state := rolePetStateFromItem(petItem)
	return RolePetInfoResult{
		Role:       role,
		PlayerBase: playerBaseDataFromRole(playerID, role),
		Item:       normalizeRoleItem(petItem),
		Found:      true,
		HasPet:     true,
		Level:      state.Level,
		Exp:        state.Exp,
		Fullness:   state.Fullness,
		Name:       petItem.Name,
		PetType:    petItem.Name,
		DisplayURL: rolePetDisplayURL(petItem),
		SourceX:    100,
		SourceY:    100,
		SkillHTML:  rolePetSkillHTML(petItem),
		PetID:      state.PetID,
		Status:     rolePetStatus(state.Fullness),
	}
}

func withRolePetInfoError(info RolePetInfoResult, code string, message string) RolePetInfoResult {
	info.ErrorCode = code
	info.ErrorMessage = message
	return info
}

func findRolePetFeedItem(items []RoleItem, sourceType string, sourceIndex int) (RoleItem, int, bool) {
	for index, item := range items {
		if item.Type == sourceType && item.Index == sourceIndex && isRolePetFeedItem(item) && item.Count > 0 {
			return normalizeRoleItem(item), index, true
		}
	}

	for _, name := range []string{"奇效宠物药剂", "宠物成长药剂", "宠物用营养水"} {
		for index, item := range items {
			if item.Type == "背包" && item.Name == name && isRolePetFeedItem(item) && item.Count > 0 {
				return normalizeRoleItem(item), index, true
			}
		}
	}

	return RoleItem{}, -1, false
}

func isRolePetFeedItem(item RoleItem) bool {
	switch item.Name {
	case "奇效宠物药剂", "宠物成长药剂", "宠物用营养水":
		return true
	default:
		return strings.Contains(item.Name, "宠物") && (strings.Contains(item.Description, "饱食") || strings.Contains(item.Description, "成长"))
	}
}

func rolePetFeedGains(item RoleItem) (int, int) {
	growth := parsePositiveNumberAfterMarker(item.Description, "成长")
	fullness := parsePositiveNumberAfterMarker(item.Description, "饱食")
	if item.Name == "宠物用营养水" {
		if growth <= 0 {
			growth = 5
		}
		if fullness <= 0 {
			fullness = 10
		}
	}
	if item.Name == "奇效宠物药剂" || item.Name == "宠物成长药剂" {
		if fullness <= 0 {
			fullness = 2
		}
	}
	return growth, fullness
}

func applyRolePetFeedState(item RoleItem, growthGain int, fullnessGain int) RoleItem {
	state := rolePetStateFromItem(item)
	state.Exp += maxInt(0, growthGain)
	state.Fullness = clampInt(state.Fullness+maxInt(0, fullnessGain), 0, 100)
	state.Level = rolePetLevelForExp(state.Exp)
	item.PetState = &state
	return normalizeRoleItem(item)
}

func rolePetStateFromItem(item RoleItem) RolePetItemState {
	level := parseRolePetLevel(item.Description)
	exp := rolePetExpForLevel(level - 1)
	fullness := defaultPetFullness
	petID := defaultPetID
	if item.PetState != nil {
		if item.PetState.Level > 0 {
			level = item.PetState.Level
		}
		if item.PetState.Exp >= 0 {
			exp = item.PetState.Exp
		}
		fullness = clampInt(item.PetState.Fullness, 0, 100)
		if strings.TrimSpace(item.PetState.PetID) != "" {
			petID = strings.TrimSpace(item.PetState.PetID)
		}
	}
	if level < 1 {
		level = rolePetLevelForExp(exp)
	}
	return RolePetItemState{
		Level:    level,
		Exp:      maxInt(0, exp),
		Fullness: fullness,
		PetID:    petID,
	}
}

func parseRolePetLevel(description string) int {
	level := parsePositiveNumberAfterMarker(description, "宠物等级:")
	if level <= 0 {
		return 1
	}
	return level
}

func rolePetExpForLevel(level int) int {
	if level <= 0 {
		return 0
	}
	if level >= len(classicPetLevelToExp) {
		return classicPetLevelToExp[len(classicPetLevelToExp)-1]
	}
	return classicPetLevelToExp[level]
}

func rolePetLevelForExp(exp int) int {
	level := 1
	for index := 1; index < len(classicPetLevelToExp); index += 1 {
		if exp < classicPetLevelToExp[index] {
			break
		}
		level = index + 1
	}
	return level
}

func rolePetStatus(fullness int) string {
	if fullness <= 10 {
		return "paralyzed"
	}
	return "good"
}

func rolePetDisplayURL(item RoleItem) string {
	switch item.Name {
	case "炎火兽":
		return "petmap/yhs1.swf"
	default:
		return ""
	}
}

func rolePetSkillHTML(item RoleItem) string {
	description := item.Description
	index := strings.LastIndex(description, "&19@")
	if index < 0 {
		return description
	}
	return description[index:]
}

func parsePositiveNumberAfterMarker(text string, marker string) int {
	index := strings.Index(text, marker)
	if index < 0 {
		return 0
	}
	rest := text[index+len(marker):]
	start := -1
	end := -1
	for offset, value := range rest {
		if value >= '0' && value <= '9' {
			if start < 0 {
				start = offset
			}
			end = offset + 1
			continue
		}
		if start >= 0 {
			break
		}
	}
	if start < 0 || end <= start {
		return 0
	}
	value, err := strconv.Atoi(rest[start:end])
	if err != nil {
		return 0
	}
	return value
}

func clampInt(value int, minValue int, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func normalizeRoleItemRequirements(requirements []RoleItemRequirement) []RoleItemRequirement {
	result := make([]RoleItemRequirement, 0, len(requirements))
	merged := make(map[string]int, len(requirements))
	order := make([]string, 0, len(requirements))
	for _, requirement := range requirements {
		name := strings.TrimSpace(requirement.Name)
		if name == "" || requirement.Count <= 0 {
			continue
		}
		if _, ok := merged[name]; !ok {
			order = append(order, name)
		}
		merged[name] += requirement.Count
	}
	for _, name := range order {
		result = append(result, RoleItemRequirement{Name: name, Count: merged[name]})
	}
	return result
}

func isRoleCurrencyName(name string) bool {
	switch strings.TrimSpace(name) {
	case "铜钱", "银元宝":
		return true
	default:
		return false
	}
}

func totalRoleItemCountByName(items []RoleItem, containerType string, name string) int {
	total := 0
	for _, item := range items {
		if item.Type == containerType && item.Name == name {
			total += item.Count
		}
	}
	return total
}

func consumeRoleItemsByName(items []RoleItem, containerType string, name string, count int) ([]RoleItem, []RoleItem, []RoleItemClear) {
	remaining := count
	updatedItems := make([]RoleItem, 0, len(items))
	consumedItems := []RoleItem{}
	clearedItems := []RoleItemClear{}
	for _, item := range items {
		if remaining <= 0 || item.Type != containerType || item.Name != name {
			updatedItems = append(updatedItems, item)
			continue
		}
		consumeCount := item.Count
		if consumeCount > remaining {
			consumeCount = remaining
		}
		remaining -= consumeCount
		if item.Count > consumeCount {
			item.Count -= consumeCount
			normalizedItem := normalizeRoleItem(item)
			updatedItems = append(updatedItems, normalizedItem)
			consumedItems = append(consumedItems, normalizedItem)
			continue
		}
		clearedItems = append(clearedItems, RoleItemClear{
			Type:  item.Type,
			Index: item.Index,
		})
	}
	return updatedItems, consumedItems, clearedItems
}

func trimRoleCurrencyItemsToBalance(items []RoleItem, currencies RoleCurrencies) ([]RoleItem, []RoleItem, []RoleItemClear, bool) {
	remainingByName := cloneRoleCurrencies(currencies)
	updatedItems := make([]RoleItem, 0, len(items))
	updatedCurrencyItems := []RoleItem{}
	clearedItems := []RoleItemClear{}
	changed := false
	for _, item := range items {
		if item.Type != "背包" || !isRoleCurrencyName(item.Name) {
			updatedItems = append(updatedItems, item)
			continue
		}
		remaining := remainingByName[item.Name]
		if remaining <= 0 {
			clearedItems = append(clearedItems, RoleItemClear{
				Type:  item.Type,
				Index: item.Index,
			})
			changed = true
			continue
		}
		if item.Count > remaining {
			item.Count = remaining
			normalizedItem := normalizeRoleItem(item)
			updatedItems = append(updatedItems, normalizedItem)
			updatedCurrencyItems = append(updatedCurrencyItems, normalizedItem)
			remainingByName[item.Name] = 0
			changed = true
			continue
		}
		remainingByName[item.Name] = remaining - item.Count
		updatedItems = append(updatedItems, item)
	}
	return updatedItems, updatedCurrencyItems, clearedItems, changed
}

func consumeRoleItemBySlot(items []RoleItem, containerType string, index int, count int) ([]RoleItem, *RoleItem, []RoleItemClear) {
	if count <= 0 {
		return items, nil, []RoleItemClear{}
	}
	updatedItems := make([]RoleItem, 0, len(items))
	clearedItems := []RoleItemClear{}
	var updatedItem *RoleItem
	for _, item := range items {
		if item.Type != containerType || item.Index != index {
			updatedItems = append(updatedItems, item)
			continue
		}
		if item.Count > count {
			remaining := item
			remaining.Count -= count
			normalized := normalizeRoleItem(remaining)
			updatedItems = append(updatedItems, normalized)
			updatedItem = &normalized
			continue
		}
		clearedItems = append(clearedItems, RoleItemClear{
			Type:  item.Type,
			Index: item.Index,
		})
	}
	return updatedItems, updatedItem, clearedItems
}

func roleEquipTargetIndex(item RoleItem) (int, bool) {
	if item.ItemType != "equip" {
		return 0, false
	}
	if _, ok := roleItemAppearanceSourceParams(item); ok {
		return roleFashionClothIndex, true
	}
	switch item.Name {
	case "铁斧":
		return 3, true
	case "蓝布衣":
		return 4, true
	case "蓝布裤":
		return 5, true
	case "布鞋":
		return 12, true
	}
	if strings.Contains(item.Description, "武器") {
		return 3, true
	}
	if strings.Contains(item.Description, "护具·头部") {
		return 0, true
	}
	if strings.Contains(item.Description, "护具·肩部") {
		return 1, true
	}
	if strings.Contains(item.Description, "护具·腕部") || strings.Contains(item.Description, "护具·护腕") {
		return 2, true
	}
	if strings.Contains(item.Description, "护具·躯干") || strings.Contains(item.Description, "护具·身体") {
		return 4, true
	}
	if strings.Contains(item.Description, "护具·腿") {
		return 5, true
	}
	if strings.Contains(item.Description, "护具·腰部") {
		return 10, true
	}
	if strings.Contains(item.Description, "护具·足部") || strings.Contains(item.Description, "护具·脚部") {
		return 12, true
	}
	if strings.Contains(item.Description, "法宝") || strings.Contains(item.Description, "宝1") || strings.Contains(item.Description, "宝2") || strings.Contains(item.Description, "宝3") || strings.Contains(item.Description, "宝4") {
		return roleTreasureEquipIndex, true
	}
	if strings.Contains(item.Description, "坐骑") {
		return roleMountEquipIndex, true
	}
	return 0, false
}

func roleTryEquipReplacementIndices(item RoleItem, targetIndex int) []int {
	if _, ok := roleItemAppearanceSourceParams(item); ok {
		return []int{roleFashionClothIndex, roleFashionPantsIndex, roleFashionShoesIndex}
	}
	return []int{targetIndex}
}

func roleItemIndexMatches(index int, candidates []int) bool {
	for _, candidate := range candidates {
		if index == candidate {
			return true
		}
	}
	return false
}

func applyRoleItemAppearanceToSourceQuery(sourceQuery string, item RoleItem) string {
	if params, ok := roleItemAppearanceSourceParams(item); ok {
		for _, param := range params {
			sourceQuery = setSourceQueryParam(sourceQuery, param.key, param.value)
		}
		return sourceQuery
	}
	if key, value, ok := roleItemAppearanceSourceParam(item); ok {
		return setSourceQueryParam(sourceQuery, key, value)
	}
	return sourceQuery
}

func applyRoleBodyAppearanceToSourceQuery(sourceQuery string, appearance RoleAppearance) string {
	body, ok := appearance["body"].(map[string]any)
	if !ok {
		return sourceQuery
	}
	for _, field := range []struct {
		bodyKey  string
		queryKey string
	}{
		{bodyKey: "sex", queryKey: "sex"},
		{bodyKey: "skinColor", queryKey: "co"},
		{bodyKey: "hair", queryKey: "hr"},
		{bodyKey: "eyes", queryKey: "e"},
		{bodyKey: "nose", queryKey: "n"},
		{bodyKey: "mouth", queryKey: "m"},
	} {
		value, ok := roleAppearanceInt(body[field.bodyKey])
		if ok {
			sourceQuery = setSourceQueryParam(sourceQuery, field.queryKey, fmt.Sprintf("%d", value))
		}
	}
	return sourceQuery
}

func roleAppearanceInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	case float32:
		return int(typed), true
	}
	return 0, false
}

func rebuildRoleEquipmentAppearanceSourceQuery(sourceQuery string, items []RoleItem) string {
	sourceQuery = clearRoleEquipmentAppearanceSourceQuery(sourceQuery)
	equipment := make([]RoleItem, 0)
	for _, item := range items {
		if item.Type == "装备" && item.ItemType == "equip" {
			equipment = append(equipment, item)
		}
	}
	sort.SliceStable(equipment, func(left int, right int) bool {
		return equipment[left].Index < equipment[right].Index
	})
	for _, item := range equipment {
		sourceQuery = applyRoleItemAppearanceToSourceQuery(sourceQuery, item)
	}
	return sourceQuery
}

func clearRoleEquipmentAppearanceSourceQuery(sourceQuery string) string {
	for _, key := range roleEquipmentAppearanceSourceKeys() {
		sourceQuery = removeSourceQueryParam(sourceQuery, key)
	}
	return sourceQuery
}

func roleEquipmentAppearanceSourceKeys() []string {
	keys := []string{"w", "h", "a", "g", "c", "b", "wr", "se", "p"}
	for index := 1; index <= 20; index += 1 {
		keys = append(keys, fmt.Sprintf("w%d", index))
	}
	return keys
}

type roleItemAppearanceSourceParamPair struct {
	key   string
	value string
}

func roleItemAppearanceSourceParams(item RoleItem) ([]roleItemAppearanceSourceParamPair, bool) {
	switch item.Name {
	case "超时空要塞":
		return []roleItemAppearanceSourceParamPair{
			{key: "c", value: "88"},
			{key: "p", value: "91"},
			{key: "se", value: "79"},
		}, true
	case "盛夏缤纷":
		return []roleItemAppearanceSourceParamPair{
			{key: "c", value: "52"},
			{key: "p", value: "55"},
			{key: "se", value: "41"},
			{key: "hr", value: "19"},
		}, true
	}
	return nil, false
}

func roleItemAppearanceSourceParam(item RoleItem) (string, string, bool) {
	switch item.Name {
	case "铁斧":
		return "w8", "5", true
	case "蛮力钢剑":
		return "w8", "40", true
	case "刎刀":
		return "w8", "42", true
	case "剔骨刀":
		return "w3", "34", true
	case "牙刺":
		return "w3", "43", true
	case "绯雨匕首":
		return "w3", "49", true
	case "万相":
		return "w1", "55", true
	case "伏魔棍":
		return "w11", "53", true
	case "蓝布衣":
		return "c", "1", true
	case "蛮力护甲":
		return "c", "10", true
	case "蛤蟆法袍":
		return "c", "35", true
	case "神风护甲":
		return "c", "17", true
	case "寒影锁甲":
		return "c", "26", true
	case "寨夫人上衣":
		return "c", "39", true
	case "蓝布裤":
		return "p", "1", true
	case "蛮力护腿":
		return "p", "8", true
	case "威武护腿":
		return "p", "13", true
	case "神风护腿":
		return "p", "16", true
	case "机木护腿":
		return "p", "64", true
	case "龙颜护腿":
		return "p", "22", true
	case "布鞋":
		return "se", "1", true
	case "蛮力战靴":
		return "se", "4", true
	case "蛤蟆精战靴":
		return "se", "29", true
	case "盗贼的鞋":
		return "se", "27", true
	case "呼啸战靴":
		return "se", "26", true
	case "神风战靴":
		return "se", "12", true
	case "寒影靴":
		return "se", "19", true
	case "蛮力面甲":
		return "h", "8", true
	case "威武面甲":
		return "h", "12", true
	case "黄风围巾":
		return "h", "30", true
	case "蛮力护腰":
		return "b", "5", true
	case "蛤蟆精护腰":
		return "b", "31", true
	case "神风护腰":
		return "b", "14", true
	case "寒影护腰":
		return "b", "22", true
	case "龙颜护腰":
		return "b", "21", true
	case "蛮力肩甲":
		return "a", "4", true
	case "威武护肩":
		return "a", "10", true
	case "蓝晶护肩":
		return "a", "34", true
	case "蚩颅王护肩":
		return "a", "29", true
	case "狼人护肩":
		return "a", "32", true
	case "龙颜单肩":
		return "a", "19", true
	case "蛮力护腕":
		return "wr", "7", true
	case "威武护腕":
		return "wr", "11", true
	case "黄风护腕":
		return "wr", "25", true
	case "机木护腕":
		return "wr", "39", true
	case "龙颜护腕":
		return "wr", "19", true
	}
	if item.ItemType == "equip" && strings.TrimSpace(item.Display) == "29.png" {
		return "w8", "5", true
	}
	return "", "", false
}

func removeSourceQueryParam(sourceQuery string, key string) string {
	sourceQuery = strings.TrimSpace(sourceQuery)
	if sourceQuery == "" {
		return sourceQuery
	}

	base := sourceQuery
	query := ""
	if separator := strings.Index(sourceQuery, "?"); separator >= 0 {
		base = sourceQuery[:separator]
		query = sourceQuery[separator+1:]
	}
	if query == "" {
		return sourceQuery
	}

	parts := make([]string, 0)
	for _, part := range strings.Split(query, "&") {
		part = strings.TrimSpace(part)
		if part == "" || strings.HasPrefix(part, key+"=") {
			continue
		}
		parts = append(parts, part)
	}
	if len(parts) == 0 {
		return base + "?"
	}
	return base + "?" + strings.Join(parts, "&") + "&"
}

func setSourceQueryParam(sourceQuery string, key string, value string) string {
	sourceQuery = strings.TrimSpace(sourceQuery)
	if sourceQuery == "" {
		sourceQuery = "human/human.swf?"
	}

	base := sourceQuery
	query := ""
	if separator := strings.Index(sourceQuery, "?"); separator >= 0 {
		base = sourceQuery[:separator]
		query = sourceQuery[separator+1:]
	}
	if base == "" {
		base = "human/human.swf"
	}

	parts := make([]string, 0)
	for _, part := range strings.Split(query, "&") {
		part = strings.TrimSpace(part)
		if part == "" || strings.HasPrefix(part, key+"=") {
			continue
		}
		parts = append(parts, part)
	}
	parts = append(parts, key+"="+value)
	return base + "?" + strings.Join(parts, "&") + "&"
}

func (store *Store) LearnRoleSkill(playerID string, roleID string, skill RoleSkill) ([]RoleSkill, int, bool, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		roles[index] = withRoleRuntimeDefaults(roles[index])
		skill = normalizeRoleSkill(skill)
		if skill.Name == "" {
			return cloneRoleSkills(roles[index].Skills), defaultSkillCap, true, false
		}

		for _, existing := range roles[index].Skills {
			if existing.Name == skill.Name {
				return cloneRoleSkills(roles[index].Skills), defaultSkillCap, true, false
			}
		}

		if len(roles[index].Skills) >= defaultSkillCap {
			return cloneRoleSkills(roles[index].Skills), defaultSkillCap, true, false
		}

		roles[index].Skills = append(roles[index].Skills, skill)
		store.rolesByPID[playerID] = roles
		if err := store.persistPlayerStateLocked(playerID); err != nil {
			log.Printf("[session.Store] persist learned skill failed: %v", err)
		}

		return cloneRoleSkills(roles[index].Skills), defaultSkillCap, true, true
	}

	return nil, defaultSkillCap, false, false
}

func (store *Store) RemoveRoleSkill(playerID string, roleID string, name string) RoleSkillRemoveResult {
	store.mu.Lock()
	defer store.mu.Unlock()

	name = strings.TrimSpace(name)
	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		roles[index] = withRoleRuntimeDefaults(roles[index])
		currentSkills := cloneRoleSkills(roles[index].Skills)
		if name == "" {
			return RoleSkillRemoveResult{
				Skills:       currentSkills,
				SkillCap:     defaultSkillCap,
				Found:        true,
				ErrorCode:    "skill_missing",
				ErrorMessage: "技能不存在。",
			}
		}
		if name == "普通攻击" {
			return RoleSkillRemoveResult{
				Skills:       currentSkills,
				SkillCap:     defaultSkillCap,
				Found:        true,
				ErrorCode:    "skill_locked",
				ErrorMessage: "普通攻击不能遗忘。",
			}
		}

		updatedSkills := make([]RoleSkill, 0, len(roles[index].Skills))
		var removedSkill *RoleSkill
		for _, skill := range roles[index].Skills {
			if skill.Name == name {
				copySkill := normalizeRoleSkill(skill)
				removedSkill = &copySkill
				continue
			}
			updatedSkills = append(updatedSkills, skill)
		}
		if removedSkill == nil {
			return RoleSkillRemoveResult{
				Skills:       currentSkills,
				SkillCap:     defaultSkillCap,
				Found:        true,
				ErrorCode:    "skill_missing",
				ErrorMessage: "技能不存在。",
			}
		}

		roles[index].Skills = cloneRoleSkills(updatedSkills)
		roles[index].FastPanel = filterRoleFastPanelEntries(roles[index].FastPanel, roles[index].Skills)
		store.rolesByPID[playerID] = roles
		if err := store.persistPlayerStateLocked(playerID); err != nil {
			log.Printf("[session.Store] persist removed skill failed: %v", err)
		}

		return RoleSkillRemoveResult{
			Skills:       cloneRoleSkills(roles[index].Skills),
			SkillCap:     defaultSkillCap,
			RemovedSkill: removedSkill,
			Found:        true,
			Removed:      true,
		}
	}

	return RoleSkillRemoveResult{
		SkillCap:     defaultSkillCap,
		Found:        false,
		ErrorCode:    "role_missing",
		ErrorMessage: "角色不存在。",
	}
}

func (store *Store) PurchaseRoleSkill(playerID string, roleID string, skill RoleSkill, cost RoleCurrencies) RoleSkillPurchaseResult {
	store.mu.Lock()
	defer store.mu.Unlock()

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		roles[index] = withRoleRuntimeDefaults(roles[index])
		skill = normalizeRoleSkill(skill)
		maxLevel := normalizeSkillMaxLevel(skill.MaxLevel)
		currentSkills := cloneRoleSkills(roles[index].Skills)
		currentCurrencies := cloneRoleCurrencies(roles[index].Currencies)
		if skill.Name == "" {
			return RoleSkillPurchaseResult{
				Skills:       currentSkills,
				SkillCap:     defaultSkillCap,
				Currencies:   currentCurrencies,
				Found:        true,
				ErrorCode:    "invalid_skill",
				ErrorMessage: "技能不存在。",
			}
		}

		normalizedCost := normalizeRoleCurrencies(cost)
		for name, count := range normalizedCost {
			if roles[index].Currencies[name] < count {
				return RoleSkillPurchaseResult{
					Skills:       currentSkills,
					SkillCap:     defaultSkillCap,
					Currencies:   currentCurrencies,
					Found:        true,
					ErrorCode:    "not_enough_currency",
					ErrorMessage: fmt.Sprintf("%s不足。", name),
				}
			}
		}

		for skillIndex, existing := range roles[index].Skills {
			if existing.Name == skill.Name {
				existing = normalizeRoleSkill(existing)
				if existing.Level >= maxLevel {
					return RoleSkillPurchaseResult{
						Skills:       currentSkills,
						SkillCap:     defaultSkillCap,
						Currencies:   currentCurrencies,
						Found:        true,
						ErrorCode:    "skill_level_max",
						ErrorMessage: "该技能已达到当前最高等级。",
					}
				}
				for name, count := range normalizedCost {
					roles[index].Currencies[name] -= count
				}
				skill.Level = existing.Level + 1
				skill.MaxLevel = maxLevel
				roles[index].Skills[skillIndex] = skill
				store.rolesByPID[playerID] = roles
				if err := store.persistPlayerStateLocked(playerID); err != nil {
					log.Printf("[session.Store] persist upgraded skill failed: %v", err)
				}
				return RoleSkillPurchaseResult{
					Skills:     cloneRoleSkills(roles[index].Skills),
					SkillCap:   defaultSkillCap,
					Currencies: cloneRoleCurrencies(roles[index].Currencies),
					Found:      true,
					Learned:    true,
				}
			}
		}

		if len(roles[index].Skills) >= defaultSkillCap {
			return RoleSkillPurchaseResult{
				Skills:       currentSkills,
				SkillCap:     defaultSkillCap,
				Currencies:   currentCurrencies,
				Found:        true,
				ErrorCode:    "skill_cap_full",
				ErrorMessage: "可学习技能数量已满。",
			}
		}

		for name, count := range normalizedCost {
			roles[index].Currencies[name] -= count
		}
		skill.MaxLevel = maxLevel
		roles[index].Skills = append(roles[index].Skills, skill)
		store.rolesByPID[playerID] = roles
		if err := store.persistPlayerStateLocked(playerID); err != nil {
			log.Printf("[session.Store] persist purchased skill failed: %v", err)
		}

		return RoleSkillPurchaseResult{
			Skills:     cloneRoleSkills(roles[index].Skills),
			SkillCap:   defaultSkillCap,
			Currencies: cloneRoleCurrencies(roles[index].Currencies),
			Found:      true,
			Learned:    true,
		}
	}

	return RoleSkillPurchaseResult{
		SkillCap:     defaultSkillCap,
		Currencies:   RoleCurrencies{},
		ErrorCode:    "role_missing",
		ErrorMessage: "角色不存在。",
	}
}

func (store *Store) RemoveRole(request RoleRemoveRequest) RoleRemoveResponse {
	store.mu.Lock()
	defer store.mu.Unlock()

	account, validationFailure, ok := store.validateRoleAccessLockedWithAccountLocked(request.PlayerID, request.SessionToken)
	if !ok {
		return validationFailure
	}
	if request.Password == "" {
		return RoleRemoveResponse{
			Success:      false,
			ErrorCode:    "4",
			ErrorMessage: "请输入密码。",
		}
	}
	if account.Password != request.Password {
		return RoleRemoveResponse{
			Success:      false,
			ErrorCode:    "3",
			ErrorMessage: "密码错误!",
		}
	}

	roles := store.rolesByPID[request.PlayerID]
	log.Printf(
		"[session.Store] RemoveRole request playerId=%s roleId=%s rolesBefore=%s",
		request.PlayerID,
		request.RoleID,
		formatRoleSummaries(roles),
	)
	updatedRoles := make([]RoleSummary, 0, len(roles))
	removed := false
	for _, role := range roles {
		if role.RoleID == request.RoleID {
			removed = true
			continue
		}
		updatedRoles = append(updatedRoles, role)
	}
	if removed {
		delete(store.removedQuests, request.RoleID)
		delete(store.acceptedQuests, request.RoleID)
		if store.db != nil {
			if _, err := store.db.Exec(`DELETE FROM role_accepted_quests WHERE role_id = ?`, request.RoleID); err != nil {
				log.Printf("[session.Store] delete removed role accepted quest state failed roleId=%s: %v", request.RoleID, err)
			}
			if _, err := store.db.Exec(`DELETE FROM role_removed_quests WHERE role_id = ?`, request.RoleID); err != nil {
				log.Printf("[session.Store] delete removed role quest state failed roleId=%s: %v", request.RoleID, err)
			}
		}
		store.rolesByPID[request.PlayerID] = updatedRoles
		store.normalizeRoleIDsLocked(request.PlayerID)
		if err := store.persistPlayerStateLocked(request.PlayerID); err != nil {
			log.Printf("[session.Store] persist removed role failed: %v", err)
		}
		log.Printf(
			"[session.Store] RemoveRole success playerId=%s roleId=%s rolesAfter=%s",
			request.PlayerID,
			request.RoleID,
			formatRoleSummaries(store.rolesByPID[request.PlayerID]),
		)
		return RoleRemoveResponse{
			RemovedRoleID: request.RoleID,
			Success:       true,
			Message:       "删除成功！",
		}
	}

	return RoleRemoveResponse{
		Success:      false,
		ErrorCode:    "5",
		ErrorMessage: "角色不存在。",
	}
}

func (store *Store) validateRoleAccessLocked(playerID string, sessionToken string) (RoleRemoveResponse, bool) {
	_, failure, ok := store.validateRoleAccessLockedWithAccountLocked(playerID, sessionToken)
	return failure, ok
}

func (store *Store) validateRoleAccessLockedWithAccountLocked(playerID string, sessionToken string) (AccountRecord, RoleRemoveResponse, bool) {
	trimmedPlayerID := strings.TrimSpace(playerID)
	trimmedSessionToken := strings.TrimSpace(sessionToken)
	if trimmedPlayerID == "" || trimmedSessionToken == "" {
		return AccountRecord{}, RoleRemoveResponse{
			Success:      false,
			ErrorCode:    "6",
			ErrorMessage: "登录状态已失效，请重新登录。",
		}, false
	}

	account, ok := store.findAccountByPlayerIDLocked(trimmedPlayerID)
	if ok {
		if account.SessionToken != trimmedSessionToken {
			return AccountRecord{}, RoleRemoveResponse{
				Success:      false,
				ErrorCode:    "6",
				ErrorMessage: "登录状态已失效，请重新登录。",
			}, false
		}
		return account, RoleRemoveResponse{}, true
	}

	localSessionToken := fmt.Sprintf("local-session-%s", trimmedPlayerID)
	if trimmedSessionToken == localSessionToken {
		return AccountRecord{
			PlayerID:     trimmedPlayerID,
			SessionToken: trimmedSessionToken,
		}, RoleRemoveResponse{}, true
	}

	return AccountRecord{}, RoleRemoveResponse{
		Success:      false,
		ErrorCode:    "6",
		ErrorMessage: "登录状态已失效，请重新登录。",
	}, false
}

func (store *Store) findAccountByPlayerIDLocked(playerID string) (AccountRecord, bool) {
	for _, account := range store.accountsByName {
		if account.PlayerID == playerID {
			return account, true
		}
	}

	return AccountRecord{}, false
}

func (store *Store) normalizeRoleIDsLocked(playerID string) bool {
	roles := store.rolesByPID[playerID]
	if len(roles) == 0 {
		return false
	}

	changed := false
	maxSeq := store.nextRoleSeqByPID[playerID]
	usedRoleIDs := make(map[string]struct{}, len(roles))
	for index := range roles {
		roleID := roles[index].RoleID
		roleSeq, ok := parseRoleSeq(playerID, roleID)
		if ok {
			if roleSeq > maxSeq {
				maxSeq = roleSeq
			}
			if _, duplicated := usedRoleIDs[roleID]; !duplicated {
				usedRoleIDs[roleID] = struct{}{}
				continue
			}
		}

		maxSeq += 1
		roles[index].RoleID = fmt.Sprintf("%s-role-%03d", playerID, maxSeq)
		usedRoleIDs[roles[index].RoleID] = struct{}{}
		changed = true
	}

	store.rolesByPID[playerID] = roles
	if store.nextRoleSeqByPID[playerID] != maxSeq {
		changed = true
	}
	store.nextRoleSeqByPID[playerID] = maxSeq
	return changed
}

func (store *Store) hasRoleLocked(playerID string, roleID string) bool {
	for _, role := range store.rolesByPID[playerID] {
		if role.RoleID == roleID {
			return true
		}
	}
	return false
}

func cloneBoolMap(values map[string]bool) map[string]bool {
	result := make(map[string]bool, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func validateUserName(userName string) string {
	if len(userName) > 16 || len(userName) < 3 {
		return "long"
	}

	for index := 0; index < len(userName); index++ {
		code := userName[index]
		isNumeric := code >= '0' && code <= '9'
		isUppercase := code >= 'A' && code <= 'Z'
		isLowercase := code >= 'a' && code <= 'z'
		if !isNumeric && !isUppercase && !isLowercase {
			return "errtype"
		}
	}

	return "success"
}
