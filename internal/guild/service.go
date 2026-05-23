package guild

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
)

const (
	PermissionNotice  = 1 << 0
	PermissionKick    = 1 << 1
	PermissionDismiss = 1 << 2
	LeaderPermission  = PermissionNotice | PermissionKick | PermissionDismiss
	DefaultMaxMember  = 30
)

type Guild struct {
	ID            string `json:"guildId"`
	Name          string `json:"name"`
	LogoID        string `json:"logoId"`
	Notice        string `json:"notice"`
	CreatorRoleID string `json:"creatorRoleId"`
	CreatorName   string `json:"creatorName"`
	MaxMember     int    `json:"maxMember"`
}

type Member struct {
	GuildID  string `json:"guildId,omitempty"`
	RoleID   string `json:"roleId"`
	RoleName string `json:"roleName"`
	Level    int    `json:"level"`
	Position string `json:"position"`
	Auth     int    `json:"auth"`
	Online   bool   `json:"online"`
}

type Auth struct {
	GuildID        string `json:"guildId,omitempty"`
	RoleID         string `json:"roleId"`
	PermissionMask int    `json:"permissionMask"`
}

type CreateRequest struct {
	Name   string `json:"name"`
	LogoID string `json:"logoId,omitempty"`
}

type NoticeUpdateRequest struct {
	Notice string `json:"notice"`
}

type KickRequest struct {
	RoleID string `json:"roleId"`
}

type Result struct {
	Success      bool     `json:"success"`
	ErrorCode    string   `json:"errorCode,omitempty"`
	ErrorMessage string   `json:"errorMessage,omitempty"`
	Info         *Guild   `json:"info,omitempty"`
	Members      []Member `json:"members,omitempty"`
	Auth         *Auth    `json:"auth,omitempty"`
	Notice       string   `json:"notice,omitempty"`
}

type Service struct {
	mu      sync.Mutex
	db      *sql.DB
	guilds  map[string]Guild
	members map[string]Member
	nextSeq int
}

func NewMemoryService() *Service {
	return &Service{
		guilds:  make(map[string]Guild),
		members: make(map[string]Member),
	}
}

func NewService(db *sql.DB) (*Service, error) {
	service := NewMemoryService()
	service.db = db
	if err := service.initSchema(); err != nil {
		return nil, err
	}
	if err := service.load(); err != nil {
		return nil, err
	}
	return service, nil
}

func (service *Service) Create(roleID string, roleName string, level int, request CreateRequest) Result {
	service.mu.Lock()
	defer service.mu.Unlock()

	roleID = strings.TrimSpace(roleID)
	roleName = strings.TrimSpace(roleName)
	name := strings.TrimSpace(request.Name)
	if roleID == "" || roleName == "" {
		return failure("NO_SELECTED_ROLE", "请先选择角色。")
	}
	if len([]rune(name)) < 2 || len([]rune(name)) > 12 {
		return failure("INVALID_NAME", "公会名称不合法。")
	}
	if _, ok := service.guildIDForRoleLocked(roleID); ok {
		return failure("ALREADY_IN_GUILD", "已经加入公会。")
	}
	if service.findGuildByNameLocked(name) != nil {
		return failure("DUPLICATE_NAME", "公会名称已存在。")
	}

	service.nextSeq += 1
	guild := Guild{
		ID:            fmt.Sprintf("guild-%04d", service.nextSeq),
		Name:          name,
		LogoID:        normalizeLogoID(request.LogoID),
		Notice:        "欢迎加入公会。",
		CreatorRoleID: roleID,
		CreatorName:   roleName,
		MaxMember:     DefaultMaxMember,
	}
	member := Member{
		GuildID:  guild.ID,
		RoleID:   roleID,
		RoleName: roleName,
		Level:    normalizeLevel(level),
		Position: "会长",
		Auth:     LeaderPermission,
		Online:   true,
	}
	service.guilds[guild.ID] = guild
	service.members[roleID] = member
	if err := service.persistGuildLocked(guild, []Member{member}); err != nil {
		log.Printf("[guild.Service] persist create guild failed: %v", err)
		return failure("PERSIST_FAILED", "公会保存失败。")
	}

	return service.snapshotLocked(roleID)
}

func (service *Service) Info(roleID string) Result {
	service.mu.Lock()
	defer service.mu.Unlock()

	roleID = strings.TrimSpace(roleID)
	if roleID == "" {
		return failure("NO_SELECTED_ROLE", "请先选择角色。")
	}
	if _, ok := service.guildIDForRoleLocked(roleID); !ok {
		return failure("NO_GUILD", "你还没有公会")
	}
	return service.snapshotLocked(roleID)
}

func (service *Service) AddMember(guildID string, roleID string, roleName string, level int, auth int) Result {
	service.mu.Lock()
	defer service.mu.Unlock()

	guildID = strings.TrimSpace(guildID)
	roleID = strings.TrimSpace(roleID)
	roleName = strings.TrimSpace(roleName)
	guild, ok := service.guilds[guildID]
	if !ok {
		return failure("GUILD_NOT_FOUND", "公会不存在。")
	}
	if roleID == "" || roleName == "" {
		return failure("INVALID_MEMBER", "成员不存在。")
	}
	if _, ok := service.guildIDForRoleLocked(roleID); ok {
		return failure("ALREADY_IN_GUILD", "已经加入公会。")
	}
	members := service.membersForGuildLocked(guildID)
	if len(members) >= guild.MaxMember {
		return failure("MEMBER_FULL", "公会人数已满。")
	}
	service.members[roleID] = Member{
		GuildID:  guildID,
		RoleID:   roleID,
		RoleName: roleName,
		Level:    normalizeLevel(level),
		Position: "成员",
		Auth:     auth,
		Online:   false,
	}
	if err := service.persistGuildLocked(guild, service.membersForGuildLocked(guildID)); err != nil {
		log.Printf("[guild.Service] persist add member failed: %v", err)
		return failure("PERSIST_FAILED", "公会保存失败。")
	}
	return service.snapshotLocked(roleID)
}

func (service *Service) UpdateNotice(roleID string, notice string) Result {
	service.mu.Lock()
	defer service.mu.Unlock()

	member, guildID, ok := service.memberAndGuildLocked(roleID)
	if !ok {
		return failure("NO_GUILD", "你还没有公会")
	}
	if member.Auth&PermissionNotice == 0 {
		return failure("NO_AUTH", "没有公会权限。")
	}
	guild := service.guilds[guildID]
	guild.Notice = strings.TrimSpace(notice)
	service.guilds[guildID] = guild
	if err := service.persistGuildLocked(guild, service.membersForGuildLocked(guildID)); err != nil {
		log.Printf("[guild.Service] persist notice failed: %v", err)
		return failure("PERSIST_FAILED", "公会保存失败。")
	}
	return service.snapshotLocked(roleID)
}

func (service *Service) Leave(roleID string) Result {
	service.mu.Lock()
	defer service.mu.Unlock()

	member, guildID, ok := service.memberAndGuildLocked(roleID)
	if !ok {
		return failure("NO_GUILD", "你还没有公会")
	}
	if member.Auth&PermissionDismiss != 0 {
		return failure("LEADER_CANNOT_LEAVE", "会长不能直接离开公会。")
	}
	delete(service.members, strings.TrimSpace(roleID))
	if err := service.persistGuildLocked(service.guilds[guildID], service.membersForGuildLocked(guildID)); err != nil {
		log.Printf("[guild.Service] persist leave failed: %v", err)
		return failure("PERSIST_FAILED", "公会保存失败。")
	}
	return Result{Success: true, Auth: &Auth{RoleID: roleID}}
}

func (service *Service) Kick(roleID string, targetRoleID string) Result {
	service.mu.Lock()
	defer service.mu.Unlock()

	member, guildID, ok := service.memberAndGuildLocked(roleID)
	if !ok {
		return failure("NO_GUILD", "你还没有公会")
	}
	if member.Auth&PermissionKick == 0 {
		return failure("NO_AUTH", "没有公会权限。")
	}
	targetRoleID = strings.TrimSpace(targetRoleID)
	if targetRoleID == "" || targetRoleID == roleID {
		return failure("INVALID_TARGET", "成员不存在。")
	}
	targetGuildID, ok := service.guildIDForRoleLocked(targetRoleID)
	if !ok || targetGuildID != guildID {
		return failure("INVALID_TARGET", "成员不存在。")
	}
	delete(service.members, targetRoleID)
	if err := service.persistGuildLocked(service.guilds[guildID], service.membersForGuildLocked(guildID)); err != nil {
		log.Printf("[guild.Service] persist kick failed: %v", err)
		return failure("PERSIST_FAILED", "公会保存失败。")
	}
	return service.snapshotLocked(roleID)
}

func (service *Service) Dismiss(roleID string) Result {
	service.mu.Lock()
	defer service.mu.Unlock()

	member, guildID, ok := service.memberAndGuildLocked(roleID)
	if !ok {
		return failure("NO_GUILD", "你还没有公会")
	}
	if member.Auth&PermissionDismiss == 0 {
		return failure("NO_AUTH", "没有公会权限。")
	}
	for memberRoleID := range service.members {
		if currentGuildID, memberOk := service.guildIDForRoleLocked(memberRoleID); memberOk && currentGuildID == guildID {
			delete(service.members, memberRoleID)
		}
	}
	delete(service.guilds, guildID)
	if err := service.deleteGuildLocked(guildID); err != nil {
		log.Printf("[guild.Service] persist dismiss failed: %v", err)
		return failure("PERSIST_FAILED", "公会保存失败。")
	}
	return Result{Success: true, Auth: &Auth{RoleID: roleID}}
}

func (service *Service) snapshotLocked(roleID string) Result {
	guildID, ok := service.guildIDForRoleLocked(roleID)
	if !ok {
		return failure("NO_GUILD", "你还没有公会")
	}
	guild := service.guilds[guildID]
	member := service.members[roleID]
	members := service.membersForGuildLocked(guildID)
	return Result{
		Success: true,
		Info:    &guild,
		Members: members,
		Auth: &Auth{
			GuildID:        guild.ID,
			RoleID:         roleID,
			PermissionMask: member.Auth,
		},
		Notice: guild.Notice,
	}
}

func (service *Service) memberAndGuildLocked(roleID string) (Member, string, bool) {
	roleID = strings.TrimSpace(roleID)
	member, ok := service.members[roleID]
	if !ok {
		return Member{}, "", false
	}
	guildID, ok := service.guildIDForRoleLocked(roleID)
	return member, guildID, ok
}

func (service *Service) guildIDForRoleLocked(roleID string) (string, bool) {
	roleID = strings.TrimSpace(roleID)
	member, ok := service.members[roleID]
	if !ok {
		return "", false
	}
	if _, ok := service.guilds[member.GuildID]; !ok {
		return "", false
	}
	return member.GuildID, true
}

func (service *Service) membersForGuildLocked(guildID string) []Member {
	if _, ok := service.guilds[guildID]; !ok {
		return []Member{}
	}
	members := make([]Member, 0)
	for _, member := range service.members {
		if member.GuildID == guildID {
			members = append(members, member)
		}
	}
	sort.Slice(members, func(left int, right int) bool {
		if members[left].Auth != members[right].Auth {
			return members[left].Auth > members[right].Auth
		}
		return members[left].RoleName < members[right].RoleName
	})
	return members
}

func (service *Service) findGuildByNameLocked(name string) *Guild {
	for _, guild := range service.guilds {
		if strings.EqualFold(guild.Name, name) {
			copyGuild := guild
			return &copyGuild
		}
	}
	return nil
}

func failure(code string, message string) Result {
	return Result{
		Success:      false,
		ErrorCode:    code,
		ErrorMessage: message,
	}
}

func normalizeLogoID(logoID string) string {
	logoID = strings.TrimSpace(logoID)
	if logoID == "" {
		return "1"
	}
	return logoID
}

func normalizeLevel(level int) int {
	if level <= 0 {
		return 1
	}
	return level
}
