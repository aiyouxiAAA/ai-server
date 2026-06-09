package session

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"ai-server/internal/guild"
	"ai-server/internal/mall"
	_ "modernc.org/sqlite"
)

const (
	defaultRoleVoc  = "新手"
	defaultSkillCap = 12
	defaultBagCap   = 30
	defaultEquipCap = 20
	defaultMallCap  = 30
	defaultCopper   = 5000
	defaultSilver   = 1

	DungeonInstanceShuiliandong  = "shuiliandong"
	DungeonInstanceHuangfengzhai = "huangfengzhai"
	DungeonInstanceFeixiandong   = "feixiandong"
)

const dungeonInstanceTTL = time.Hour

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
	Handle      string `json:"handle,omitempty"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	ItemType    string `json:"itemType"`
	Display     string `json:"display"`
	Description string `json:"description"`
	Count       int    `json:"count"`
	Index       int    `json:"index"`
	Level       int    `json:"level"`
	EndTime     int    `json:"endTime"`
	Owner       string `json:"owner"`
	ItemLevel   int    `json:"itemLevel"`
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
	RoleID           string                          `json:"roleId"`
	DisplayName      string                          `json:"displayName"`
	Level            int                             `json:"level"`
	Exp              int                             `json:"exp"`
	Voc              string                          `json:"voc,omitempty"`
	AGI              int                             `json:"AGI,omitempty"`
	STR              int                             `json:"STR,omitempty"`
	INT              int                             `json:"INT,omitempty"`
	CON              int                             `json:"CON,omitempty"`
	LCK              int                             `json:"LCK,omitempty"`
	MapID            int                             `json:"mapId"`
	VisualRoleID     int                             `json:"visualRoleId"`
	PresetID         int                             `json:"presetId,omitempty"`
	SourceQuery      string                          `json:"sourceQuery,omitempty"`
	Appearance       RoleAppearance                  `json:"appearance,omitempty"`
	Skills           []RoleSkill                     `json:"skills,omitempty"`
	FastPanel        []RoleFastPanelEntry            `json:"fastPanel,omitempty"`
	Currencies       RoleCurrencies                  `json:"currencies,omitempty"`
	Items            []RoleItem                      `json:"items,omitempty"`
	RoleState        *RoleState                      `json:"-"`
	RolePhysique     *RolePhysique                   `json:"-"`
	DungeonInstances map[string]DungeonInstanceState `json:"-"`
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
	PlayerID     string         `json:"playerId"`
	RoleID       string         `json:"roleId"`
	DisplayName  string         `json:"displayName"`
	Level        int            `json:"level"`
	Exp          int            `json:"exp"`
	Voc          string         `json:"voc,omitempty"`
	HP           int            `json:"hp,omitempty"`
	MP           int            `json:"mp,omitempty"`
	MaxHP        int            `json:"maxHp,omitempty"`
	MaxMP        int            `json:"maxMp,omitempty"`
	MapID        int            `json:"mapId"`
	VisualRoleID int            `json:"visualRoleId"`
	PresetID     int            `json:"presetId,omitempty"`
	SourceQuery  string         `json:"sourceQuery,omitempty"`
	Appearance   RoleAppearance `json:"appearance,omitempty"`
	Currencies   RoleCurrencies `json:"currencies,omitempty"`
	RoleState    *RoleState     `json:"roleState,omitempty"`
	RolePhysique *RolePhysique  `json:"rolePhysique,omitempty"`
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

type RoleItemClear struct {
	Type  string
	Index int
}

type RoleEquipItemResult struct {
	Role         RoleSummary
	PlayerBase   PlayerBaseData
	EquippedItem RoleItem
	ClearedItems []RoleItemClear
	Found        bool
	Equipped     bool
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

type RoleUseItemResult struct {
	Role         RoleSummary
	PlayerBase   PlayerBaseData
	Item         RoleItem
	LearnedSkill *RoleSkill
	UpdatedItem  *RoleItem
	UpdatedItems []RoleItem
	ClearedItems []RoleItemClear
	Currencies   RoleCurrencies
	Found        bool
	Used         bool
	Equipped     bool
	ErrorCode    string
	ErrorMessage string
}

type RoleExpGrantResult struct {
	Role       RoleSummary
	PlayerBase PlayerBaseData
	RoleState  RoleState
	Found      bool
	Granted    bool
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
	mu               sync.Mutex
	rolesByPID       map[string][]RoleSummary
	nextRoleSeqByPID map[string]int
	accountsByName   map[string]AccountRecord
	db               *sql.DB
	now              func() time.Time
	Guilds           *guild.Service
	Mall             *mall.Service
	mallRequests     map[string]mall.PurchaseResult
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
		rolesByPID:       make(map[string][]RoleSummary),
		nextRoleSeqByPID: make(map[string]int),
		accountsByName: map[string]AccountRecord{
			"mockuser": {
				UserName:     "mockuser",
				Password:     "magicpwd",
				PlayerID:     "mock-player-001",
				DisplayName:  "Mock Swordswoman",
				SessionToken: "mock-session-token-001",
			},
		},
		now:          time.Now,
		Guilds:       guild.NewMemoryService(),
		Mall:         mall.NewService(),
		mallRequests: make(map[string]mall.PurchaseResult),
	}
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
		return cloneRoleFastPanel(filterRoleFastPanelEntries(role.FastPanel, role.Skills)), true
	}

	return []RoleFastPanelEntry{}, false
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
	capacity, supported := roleContainerCapacity(containerType)
	if !supported {
		return 0, false
	}
	for _, role := range store.rolesByPID[playerID] {
		if role.RoleID == roleID {
			return capacity, true
		}
	}

	return 0, false
}

func (store *Store) GetRoleItems(playerID string, roleID string, containerType string) ([]RoleItem, int, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

	containerType = strings.TrimSpace(containerType)
	capacity, supported := roleContainerCapacity(containerType)
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
		return cloneRoleItems(items), capacity, true
	}

	return []RoleItem{}, 0, false
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

func (store *Store) GrantRoleItem(playerID string, roleID string, item RoleItem) (RoleItem, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

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

	capacity, supported := roleContainerCapacity(item.Type)
	if !supported {
		return RoleItem{}, false
	}

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		roles[index] = withRoleRuntimeDefaults(roles[index])
		updatedItems, grantedItem, ok := grantRoleItemToItems(roles[index].Items, capacity, item)
		if !ok {
			return RoleItem{}, false
		}
		roles[index].Items = normalizeRoleItems(updatedItems)
		store.rolesByPID[playerID] = roles
		if err := store.persistPlayerStateLocked(playerID); err != nil {
			log.Printf("[session.Store] persist granted item failed: %v", err)
		}
		return grantedItem, true
	}
	return RoleItem{}, false
}

func (store *Store) PurchaseRoleItem(playerID string, roleID string, item RoleItem, requirements []RoleItemRequirement) RoleItemPurchaseResult {
	store.mu.Lock()
	defer store.mu.Unlock()

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
	capacity, supported := roleContainerCapacity(item.Type)
	if !supported {
		return RoleItemPurchaseResult{
			ErrorCode:    "invalid_container",
			ErrorMessage: "目标容器无效。",
		}
	}

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		currentRole := withRoleRuntimeDefaults(roles[index])
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
		if err := store.persistPlayerStateLocked(playerID); err != nil {
			log.Printf("[session.Store] persist purchased item failed: %v", err)
		}

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

func (store *Store) GrantRoleExperience(playerID string, roleID string, expDelta int) RoleExpGrantResult {
	store.mu.Lock()
	defer store.mu.Unlock()

	if expDelta <= 0 {
		return RoleExpGrantResult{}
	}

	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		roles[index] = withRoleRuntimeDefaults(roles[index])
		roles[index].Exp += expDelta
		roles[index].Level = ClassicRoleLevelForExp(roles[index].Exp, roles[index].Level)
		store.rolesByPID[playerID] = roles
		if err := store.persistPlayerStateLocked(playerID); err != nil {
			log.Printf("[session.Store] persist granted experience failed: %v", err)
		}

		role := withRoleRuntimeDefaults(roles[index])
		playerBase := playerBaseDataFromRole(playerID, role)
		roleState := *playerBase.RoleState
		return RoleExpGrantResult{
			Role:       role,
			PlayerBase: playerBase,
			RoleState:  roleState,
			Found:      true,
			Granted:    true,
		}
	}
	return RoleExpGrantResult{}
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
		roles[index].Level = level
		if level <= 1 {
			roles[index].Exp = 0
		} else {
			roles[index].Exp = ClassicRoleLevelToExp(level - 1)
		}
		store.rolesByPID[playerID] = roles
		if err := store.persistPlayerStateLocked(playerID); err != nil {
			log.Printf("[session.Store] persist set role level failed: %v", err)
		}

		role := withRoleRuntimeDefaults(roles[index])
		playerBase := playerBaseDataFromRole(playerID, role)
		roleState := *playerBase.RoleState
		return RoleExpGrantResult{
			Role:       role,
			PlayerBase: playerBase,
			RoleState:  roleState,
			Found:      true,
			Granted:    true,
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
		store.rolesByPID[playerID] = roles
		if err := store.persistPlayerStateLocked(playerID); err != nil {
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

		switch sourceItem.Name {
		case "银元宝":
			return store.useCurrencyExchangeItemLocked(playerID, roles, index, sourceItem, 1, "铜钱", 1000)
		case "铜钱":
			return store.useCurrencyExchangeItemLocked(playerID, roles, index, sourceItem, 1000, "银元宝", 1)
		default:
			if skill, ok := roleSkillFromItem(sourceItem); ok {
				return store.useSkillItemLocked(playerID, roles, index, sourceItem, skill)
			}
			targetIndex, ok := roleEquipTargetIndex(sourceItem)
			if ok {
				return store.useEquipmentItemLocked(playerID, roles, index, sourceItem, sourceType, sourceIndex, targetIndex)
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

func roleSkillFromItem(item RoleItem) (RoleSkill, bool) {
	itemType := strings.TrimSpace(item.ItemType)
	if itemType != "被动技能" && !strings.HasPrefix(itemType, "技能") {
		return RoleSkill{}, false
	}
	name := strings.TrimSpace(item.Name)
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
		updatedItems = append(updatedItems, normalizeRoleItem(*replacedItem))
	}

	equippedItem := sourceItem
	equippedItem.Type = "装备"
	equippedItem.Index = targetIndex
	updatedItems = append(updatedItems, normalizeRoleItem(equippedItem))
	roles[roleIndex].Items = normalizeRoleItems(updatedItems)
	roles[roleIndex].SourceQuery = rebuildRoleEquipmentAppearanceSourceQuery(roles[roleIndex].SourceQuery, roles[roleIndex].Items)
	store.rolesByPID[playerID] = roles
	if err := store.persistPlayerStateLocked(playerID); err != nil {
		log.Printf("[session.Store] persist active equipped item failed: %v", err)
	}

	role := withRoleRuntimeDefaults(roles[roleIndex])
	return RoleUseItemResult{
		Role:         role,
		PlayerBase:   playerBaseDataFromRole(playerID, role),
		Item:         sourceItem,
		UpdatedItems: []RoleItem{normalizeRoleItem(equippedItem)},
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
			updatedItems = append(updatedItems, normalizeRoleItem(*replacedItem))
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
		store.rolesByPID[playerID] = roles
		if err := store.persistPlayerStateLocked(playerID); err != nil {
			log.Printf("[session.Store] persist equipped item failed: %v", err)
		}

		role := withRoleRuntimeDefaults(roles[index])
		return RoleEquipItemResult{
			Role:         role,
			PlayerBase:   playerBaseDataFromRole(playerID, role),
			EquippedItem: equippedItem,
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

func (store *Store) MoveRoleItem(playerID string, roleID string, sourceType string, sourceIndex int, targetType string, targetIndex int, count int) RoleMoveItemResult {
	store.mu.Lock()
	defer store.mu.Unlock()

	sourceType = strings.TrimSpace(sourceType)
	targetType = strings.TrimSpace(targetType)
	roles := store.rolesByPID[playerID]
	for index := range roles {
		if roles[index].RoleID != roleID {
			continue
		}

		roles[index] = withRoleRuntimeDefaults(roles[index])
		capacity, supported := roleContainerCapacity(targetType)
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
		canSplitMove := sourceType == targetType && targetType == "背包" && moveCount < sourceItem.Count && !hasTarget
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
		}
		store.rolesByPID[playerID] = roles
		if err := store.persistPlayerStateLocked(playerID); err != nil {
			log.Printf("[session.Store] persist moved item failed: %v", err)
		}

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

func roleContainerCapacity(containerType string) (int, bool) {
	switch containerType {
	case "背包":
		return defaultBagCap, true
	case "装备":
		return defaultEquipCap, true
	case "商城":
		return defaultMallCap, true
	default:
		return 0, false
	}
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
	return 0, false
}

func applyRoleItemAppearanceToSourceQuery(sourceQuery string, item RoleItem) string {
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

func roleItemAppearanceSourceParam(item RoleItem) (string, string, bool) {
	switch item.Name {
	case "铁斧":
		return "w8", "5", true
	case "蛮力钢剑":
		return "w8", "40", true
	case "刎刀":
		return "w8", "42", true
	case "蓝布衣":
		return "c", "1", true
	case "蛮力护甲":
		return "c", "10", true
	case "蓝布裤":
		return "p", "1", true
	case "蛮力护腿":
		return "p", "8", true
	case "布鞋":
		return "se", "1", true
	case "蛮力战靴":
		return "se", "4", true
	case "蛤蟆精战靴":
		return "se", "29", true
	case "蛮力面甲":
		return "h", "8", true
	case "蛮力护腰":
		return "b", "5", true
	case "蛮力肩甲":
		return "a", "4", true
	case "蛮力护腕":
		return "wr", "7", true
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
