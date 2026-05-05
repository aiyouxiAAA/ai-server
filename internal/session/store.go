package session

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
)

type RoleAppearance map[string]any

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
	RoleID       string         `json:"roleId"`
	DisplayName  string         `json:"displayName"`
	Level        int            `json:"level"`
	MapID        int            `json:"mapId"`
	VisualRoleID int            `json:"visualRoleId"`
	PresetID     int            `json:"presetId,omitempty"`
	SourceQuery  string         `json:"sourceQuery,omitempty"`
	Appearance   RoleAppearance `json:"appearance,omitempty"`
}

type RoleListRequest struct {
	PlayerID     string `json:"playerId"`
	SessionToken string `json:"sessionToken"`
}

type RoleListResponse struct {
	Roles []RoleSummary `json:"roles"`
}

type RoleCreateRequest struct {
	PlayerID       string         `json:"playerId"`
	DisplayName    string         `json:"displayName"`
	Gender         string         `json:"gender"`
	RoleTemplateID int            `json:"roleTemplateId"`
	PresetID       int            `json:"presetId,omitempty"`
	SourceQuery    string         `json:"sourceQuery,omitempty"`
	Appearance     RoleAppearance `json:"appearance,omitempty"`
}

type RoleCreateResponse struct {
	Role RoleSummary `json:"role"`
}

type RoleSelectRequest struct {
	PlayerID string `json:"playerId"`
	RoleID   string `json:"roleId"`
}

type PlayerBaseData struct {
	PlayerID     string         `json:"playerId"`
	RoleID       string         `json:"roleId"`
	DisplayName  string         `json:"displayName"`
	Level        int            `json:"level"`
	Exp          int            `json:"exp"`
	MapID        int            `json:"mapId"`
	VisualRoleID int            `json:"visualRoleId"`
	PresetID     int            `json:"presetId,omitempty"`
	SourceQuery  string         `json:"sourceQuery,omitempty"`
	Appearance   RoleAppearance `json:"appearance,omitempty"`
}

type RoleSelectResponse struct {
	Role       RoleSummary    `json:"role"`
	PlayerBase PlayerBaseData `json:"playerBase"`
}

type RoleRemoveRequest struct {
	PlayerID string `json:"playerId"`
	RoleID   string `json:"roleId"`
	Password string `json:"password"`
}

type RoleRemoveResponse struct {
	RemovedRoleID string `json:"removedRoleId,omitempty"`
	Success       bool   `json:"success"`
	Message       string `json:"message,omitempty"`
	ErrorCode     string `json:"errorCode,omitempty"`
	ErrorMessage  string `json:"errorMessage,omitempty"`
}

type Store struct {
	mu               sync.Mutex
	rolesByPID       map[string][]RoleSummary
	nextRoleSeqByPID map[string]int
	accountsByName   map[string]AccountRecord
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
	}
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

func (store *Store) ListRoles(playerID string) RoleListResponse {
	store.mu.Lock()
	defer store.mu.Unlock()

	store.normalizeRoleIDsLocked(playerID)
	roles := append(make([]RoleSummary, 0), store.rolesByPID[playerID]...)
	log.Printf("[session.Store] ListRoles playerId=%s roles=%s", playerID, formatRoleSummaries(roles))
	return RoleListResponse{Roles: roles}
}

func (store *Store) CreateRole(request RoleCreateRequest) RoleCreateResponse {
	store.mu.Lock()
	defer store.mu.Unlock()

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
		MapID:        1,
		VisualRoleID: resolveVisualRoleID(request.PresetID),
		PresetID:     request.PresetID,
		SourceQuery:  request.SourceQuery,
		Appearance:   request.Appearance,
	}
	store.rolesByPID[request.PlayerID] = append(roles, role)

	return RoleCreateResponse{Role: role}
}

func (store *Store) SelectRole(playerID string, roleID string) (RoleSelectResponse, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

	store.normalizeRoleIDsLocked(playerID)
	for _, role := range store.rolesByPID[playerID] {
		if role.RoleID == roleID {
			return RoleSelectResponse{
				Role: role,
				PlayerBase: PlayerBaseData{
					PlayerID:     playerID,
					RoleID:       role.RoleID,
					DisplayName:  role.DisplayName,
					Level:        role.Level,
					Exp:          0,
					MapID:        role.MapID,
					VisualRoleID: role.VisualRoleID,
					PresetID:     role.PresetID,
					SourceQuery:  role.SourceQuery,
					Appearance:   role.Appearance,
				},
			}, true
		}
	}

	return RoleSelectResponse{}, false
}

func (store *Store) RemoveRole(request RoleRemoveRequest) RoleRemoveResponse {
	store.mu.Lock()
	defer store.mu.Unlock()

	account, ok := store.findAccountByPlayerIDLocked(request.PlayerID)
	if !ok {
		return RoleRemoveResponse{
			Success:      false,
			ErrorCode:    "2",
			ErrorMessage: "帐号不存在!",
		}
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

func (store *Store) findAccountByPlayerIDLocked(playerID string) (AccountRecord, bool) {
	for _, account := range store.accountsByName {
		if account.PlayerID == playerID {
			return account, true
		}
	}

	return AccountRecord{}, false
}

func (store *Store) normalizeRoleIDsLocked(playerID string) {
	roles := store.rolesByPID[playerID]
	if len(roles) == 0 {
		return
	}

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
	}

	store.rolesByPID[playerID] = roles
	store.nextRoleSeqByPID[playerID] = maxSeq
}

func parseRoleSeq(playerID string, roleID string) (int, bool) {
	prefix := playerID + "-role-"
	if !strings.HasPrefix(roleID, prefix) {
		return 0, false
	}

	roleSeq, err := strconv.Atoi(roleID[len(prefix):])
	if err != nil || roleSeq <= 0 {
		return 0, false
	}

	return roleSeq, true
}

func formatRoleSummaries(roles []RoleSummary) string {
	if len(roles) == 0 {
		return "[]"
	}

	parts := make([]string, 0, len(roles))
	for _, role := range roles {
		parts = append(parts, fmt.Sprintf("{id:%s,name:%s,level:%d}", role.RoleID, role.DisplayName, role.Level))
	}

	return "[" + strings.Join(parts, ", ") + "]"
}

func resolveVisualRoleID(presetID int) int {
	if presetID > 0 {
		return presetID
	}

	return 1
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
