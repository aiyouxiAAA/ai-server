package team

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const MaxMembers = 4

type Member struct {
	TeamID   string `json:"teamId,omitempty"`
	RoleID   string `json:"roleId"`
	Name     string `json:"name"`
	Level    int    `json:"level"`
	Vocation string `json:"vocation"`
	HP       int    `json:"hp"`
	MaxHP    int    `json:"maxHp"`
	MP       int    `json:"mp"`
	MaxMP    int    `json:"maxMp"`
	IsLeader bool   `json:"isLeader"`
	MapID    string `json:"mapId"`
	Online   bool   `json:"online"`
}

type InvitePush struct {
	InviteID      string `json:"inviteId"`
	FromRoleID    string `json:"fromRoleId"`
	FromName      string `json:"fromName"`
	ExpiresAtUnix int64  `json:"expiresAtUnix"`
}

type InfoPush struct {
	TeamID        string `json:"teamId"`
	LeaderRoleID  string `json:"leaderRoleId"`
	SyncChangeMap bool   `json:"syncChangeMap"`
	MaxMembers    int    `json:"maxMembers"`
}

type MemberClearPush struct {
	RoleID string `json:"roleId,omitempty"`
	Name   string `json:"name,omitempty"`
}

type ClearPush struct {
	Reason string `json:"reason,omitempty"`
}

type ResultPush struct {
	Success      bool   `json:"success"`
	Action       string `json:"action"`
	ErrorCode    string `json:"errorCode,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

type SyncTransferPlan struct {
	Enabled      bool
	Members      []Member
	ErrorCode    string
	ErrorMessage string
}

type MemberPlan struct {
	Allowed      bool
	Members      []Member
	ErrorCode    string
	ErrorMessage string
}

type Event struct {
	Recipients  []string
	Invite      *InvitePush
	Info        *InfoPush
	Members     []Member
	MemberClear *MemberClearPush
	Clear       *ClearPush
	Result      *ResultPush
}

type Team struct {
	ID            string
	LeaderRoleID  string
	SyncChangeMap bool
	Members       []string
}

type invite struct {
	ID         string
	FromRoleID string
	ToRoleID   string
	ExpiresAt  time.Time
}

type Manager struct {
	mu            sync.Mutex
	teams         map[string]*Team
	teamByRoleID  map[string]string
	members       map[string]Member
	onlineByName  map[string]string
	invites       map[string]invite
	nextTeamSeq   int
	nextInviteSeq int
	now           func() time.Time
}

func NewManager() *Manager {
	return &Manager{
		teams:        make(map[string]*Team),
		teamByRoleID: make(map[string]string),
		members:      make(map[string]Member),
		onlineByName: make(map[string]string),
		invites:      make(map[string]invite),
		now:          time.Now,
	}
}

func (manager *Manager) Reset() {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	manager.teams = make(map[string]*Team)
	manager.teamByRoleID = make(map[string]string)
	manager.members = make(map[string]Member)
	manager.onlineByName = make(map[string]string)
	manager.invites = make(map[string]invite)
	manager.nextTeamSeq = 0
	manager.nextInviteSeq = 0
}

func (manager *Manager) UpsertOnline(member Member) []Event {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	member.RoleID = strings.TrimSpace(member.RoleID)
	member.Name = strings.TrimSpace(member.Name)
	if member.RoleID == "" || member.Name == "" {
		return nil
	}
	member.Online = true
	if member.MaxHP <= 0 {
		member.MaxHP = member.HP
	}
	if member.MaxMP <= 0 {
		member.MaxMP = member.MP
	}
	previous, existed := manager.members[member.RoleID]
	manager.members[member.RoleID] = member
	manager.onlineByName[member.Name] = member.RoleID
	if teamID := manager.teamByRoleID[member.RoleID]; teamID != "" {
		// main.go 在每个业务包后都会 UpsertOnline。字段未变时再广播整队快照会让客户端
		// 反复销毁/重建队员条 SWF，移动时叠加 "The asset has been destroyed!" 与掉帧。
		if existed && memberSnapshotEqual(previous, member) {
			return nil
		}
		return manager.snapshotEventsLocked(teamID)
	}
	return nil
}

func memberSnapshotEqual(left Member, right Member) bool {
	return left.RoleID == right.RoleID &&
		left.Name == right.Name &&
		left.Level == right.Level &&
		left.Vocation == right.Vocation &&
		left.HP == right.HP &&
		left.MaxHP == right.MaxHP &&
		left.MP == right.MP &&
		left.MaxMP == right.MaxMP &&
		left.MapID == right.MapID &&
		left.Online == right.Online
}

func (manager *Manager) SetOffline(roleID string) []Event {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	roleID = strings.TrimSpace(roleID)
	member, ok := manager.members[roleID]
	if !ok {
		return nil
	}
	delete(manager.onlineByName, member.Name)
	member.Online = false
	manager.members[roleID] = member
	if teamID := manager.teamByRoleID[roleID]; teamID != "" {
		return manager.snapshotEventsLocked(teamID)
	}
	return nil
}

func (manager *Manager) Invite(fromRoleID string, targetRoleID string, targetName string) []Event {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	fromRoleID = strings.TrimSpace(fromRoleID)
	targetRoleID = strings.TrimSpace(targetRoleID)
	targetName = strings.TrimSpace(targetName)
	from, ok := manager.members[fromRoleID]
	if !ok || !from.Online {
		return []Event{manager.resultLocked(fromRoleID, false, "invite", "NO_SELECTED_ROLE", "请先选择角色。")}
	}
	if targetRoleID == "" && targetName != "" {
		targetRoleID = manager.onlineByName[targetName]
	}
	target, ok := manager.members[targetRoleID]
	if !ok || !target.Online {
		return []Event{manager.resultLocked(fromRoleID, false, "invite", "TARGET_OFFLINE", "目标玩家不在线。")}
	}
	if target.RoleID == fromRoleID {
		return []Event{manager.resultLocked(fromRoleID, false, "invite", "SELF_INVITE", "不能邀请自己。")}
	}
	if _, ok := manager.teamByRoleID[target.RoleID]; ok {
		return []Event{manager.resultLocked(fromRoleID, false, "invite", "TARGET_IN_TEAM", "目标玩家已有队伍。")}
	}
	if teamID := manager.teamByRoleID[fromRoleID]; teamID != "" {
		current := manager.teams[teamID]
		if current == nil || current.LeaderRoleID != fromRoleID {
			return []Event{manager.resultLocked(fromRoleID, false, "invite", "NOT_LEADER", "只有队长可以邀请队员。")}
		}
		if len(current.Members) >= MaxMembers {
			return []Event{manager.resultLocked(fromRoleID, false, "invite", "TEAM_FULL", "队伍人数已满。")}
		}
	}
	manager.nextInviteSeq += 1
	inviteID := fmt.Sprintf("team-invite-%04d", manager.nextInviteSeq)
	expiresAt := manager.now().Add(20 * time.Second)
	manager.invites[inviteID] = invite{
		ID:         inviteID,
		FromRoleID: fromRoleID,
		ToRoleID:   target.RoleID,
		ExpiresAt:  expiresAt,
	}
	return []Event{
		{
			Recipients: []string{target.RoleID},
			Invite: &InvitePush{
				InviteID:      inviteID,
				FromRoleID:    fromRoleID,
				FromName:      from.Name,
				ExpiresAtUnix: expiresAt.Unix(),
			},
		},
		manager.resultLocked(fromRoleID, true, "invite", "", ""),
	}
}

func (manager *Manager) ReplyInvite(toRoleID string, inviteID string, accept bool) []Event {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	toRoleID = strings.TrimSpace(toRoleID)
	inviteID = strings.TrimSpace(inviteID)
	pending, ok := manager.invites[inviteID]
	if !ok || pending.ToRoleID != toRoleID {
		return []Event{manager.resultLocked(toRoleID, false, "replyInvite", "INVITE_NOT_FOUND", "组队邀请已失效。")}
	}
	delete(manager.invites, inviteID)
	if manager.now().After(pending.ExpiresAt) {
		return []Event{manager.resultLocked(toRoleID, false, "replyInvite", "INVITE_EXPIRED", "组队邀请已超时。")}
	}
	if !accept {
		return []Event{
			manager.resultLocked(toRoleID, true, "replyInvite", "", ""),
			manager.resultLocked(pending.FromRoleID, false, "inviteRejected", "TARGET_REJECTED", "对方拒绝加入你的队伍"),
		}
	}
	if _, ok := manager.teamByRoleID[toRoleID]; ok {
		return []Event{manager.resultLocked(toRoleID, false, "replyInvite", "ALREADY_IN_TEAM", "你已经在队伍中。")}
	}
	from, fromOK := manager.members[pending.FromRoleID]
	to, toOK := manager.members[toRoleID]
	if !fromOK || !from.Online || !toOK || !to.Online {
		return []Event{manager.resultLocked(toRoleID, false, "replyInvite", "INVITER_OFFLINE", "邀请者已离线。")}
	}
	teamID := manager.teamByRoleID[pending.FromRoleID]
	if teamID == "" {
		manager.nextTeamSeq += 1
		teamID = fmt.Sprintf("team-%04d", manager.nextTeamSeq)
		manager.teams[teamID] = &Team{ID: teamID, LeaderRoleID: pending.FromRoleID, Members: []string{pending.FromRoleID}}
		manager.teamByRoleID[pending.FromRoleID] = teamID
	} else if team := manager.teams[teamID]; team == nil || team.LeaderRoleID != pending.FromRoleID {
		return []Event{manager.resultLocked(toRoleID, false, "replyInvite", "INVITER_NOT_LEADER", "邀请者已不是队长。")}
	}
	team := manager.teams[teamID]
	if len(team.Members) >= MaxMembers {
		return []Event{manager.resultLocked(toRoleID, false, "replyInvite", "TEAM_FULL", "队伍人数已满。")}
	}
	team.Members = append(team.Members, toRoleID)
	manager.teamByRoleID[toRoleID] = teamID
	return manager.snapshotEventsLocked(teamID)
}

func (manager *Manager) Leave(roleID string) []Event {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	roleID = strings.TrimSpace(roleID)
	teamID := manager.teamByRoleID[roleID]
	if teamID == "" {
		return []Event{manager.resultLocked(roleID, false, "leave", "NO_TEAM", "你还没有队伍。")}
	}
	return manager.removeMemberLocked(teamID, roleID, "leave")
}

func (manager *Manager) Kick(actorRoleID string, targetRoleID string, targetName string) []Event {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	actorRoleID = strings.TrimSpace(actorRoleID)
	targetRoleID = strings.TrimSpace(targetRoleID)
	targetName = strings.TrimSpace(targetName)
	teamID := manager.teamByRoleID[actorRoleID]
	team := manager.teams[teamID]
	if team == nil {
		return []Event{manager.resultLocked(actorRoleID, false, "kick", "NO_TEAM", "你还没有队伍。")}
	}
	if team.LeaderRoleID != actorRoleID {
		return []Event{manager.resultLocked(actorRoleID, false, "kick", "NOT_LEADER", "只有队长可以逐出队员。")}
	}
	if targetRoleID == "" && targetName != "" {
		targetRoleID = manager.roleIDByNameInTeamLocked(team, targetName)
	}
	if targetRoleID == "" || targetRoleID == actorRoleID || manager.teamByRoleID[targetRoleID] != teamID {
		return []Event{manager.resultLocked(actorRoleID, false, "kick", "TARGET_NOT_IN_TEAM", "队员不存在。")}
	}
	return manager.removeMemberLocked(teamID, targetRoleID, "kick")
}

func (manager *Manager) TransferLeader(actorRoleID string, targetRoleID string, targetName string) []Event {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	actorRoleID = strings.TrimSpace(actorRoleID)
	targetRoleID = strings.TrimSpace(targetRoleID)
	targetName = strings.TrimSpace(targetName)
	teamID := manager.teamByRoleID[actorRoleID]
	team := manager.teams[teamID]
	if team == nil {
		return []Event{manager.resultLocked(actorRoleID, false, "transferLeader", "NO_TEAM", "你还没有队伍。")}
	}
	if team.LeaderRoleID != actorRoleID {
		return []Event{manager.resultLocked(actorRoleID, false, "transferLeader", "NOT_LEADER", "只有队长可以委任队长。")}
	}
	if targetRoleID == "" && targetName != "" {
		targetRoleID = manager.roleIDByNameInTeamLocked(team, targetName)
	}
	if targetRoleID == "" || targetRoleID == actorRoleID || manager.teamByRoleID[targetRoleID] != teamID {
		return []Event{manager.resultLocked(actorRoleID, false, "transferLeader", "TARGET_NOT_IN_TEAM", "队员不存在。")}
	}
	team.LeaderRoleID = targetRoleID
	return manager.snapshotEventsLocked(teamID)
}

func (manager *Manager) SetSyncChangeMap(actorRoleID string, enabled bool) []Event {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	teamID := manager.teamByRoleID[strings.TrimSpace(actorRoleID)]
	team := manager.teams[teamID]
	if team == nil {
		return []Event{manager.resultLocked(actorRoleID, false, "syncChangeMap", "NO_TEAM", "你还没有队伍。")}
	}
	if team.LeaderRoleID != actorRoleID {
		return []Event{manager.resultLocked(actorRoleID, false, "syncChangeMap", "NOT_LEADER", "只有队长可以切换队伍同步。")}
	}
	team.SyncChangeMap = enabled
	return manager.snapshotEventsLocked(teamID)
}

func (manager *Manager) RecipientsForTeam(roleID string) ([]string, bool) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	teamID := manager.teamByRoleID[strings.TrimSpace(roleID)]
	team := manager.teams[teamID]
	if team == nil {
		return nil, false
	}
	return append([]string{}, team.Members...), true
}

func (manager *Manager) TeamIDForRole(roleID string) (string, bool) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	teamID := manager.teamByRoleID[strings.TrimSpace(roleID)]
	team := manager.teams[teamID]
	if team == nil || len(team.Members) < 2 {
		return "", false
	}
	return teamID, true
}

func (manager *Manager) BuildSyncTransferPlan(actorRoleID string, fromMapID string) SyncTransferPlan {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	actorRoleID = strings.TrimSpace(actorRoleID)
	fromMapID = strings.TrimSpace(fromMapID)
	teamID := manager.teamByRoleID[actorRoleID]
	team := manager.teams[teamID]
	if team == nil || !team.SyncChangeMap {
		return SyncTransferPlan{}
	}
	if team.LeaderRoleID != actorRoleID {
		return SyncTransferPlan{
			Enabled:      true,
			ErrorCode:    "NOT_LEADER",
			ErrorMessage: "只有队长可以同步队伍切图。",
		}
	}
	members := make([]Member, 0, len(team.Members)-1)
	for _, roleID := range team.Members {
		if roleID == actorRoleID {
			continue
		}
		member := manager.members[roleID]
		if !member.Online {
			return SyncTransferPlan{
				Enabled:      true,
				ErrorCode:    "MEMBER_OFFLINE",
				ErrorMessage: "队员【" + member.Name + "】不在线，队伍同步取消。",
			}
		}
		if member.MapID != fromMapID {
			return SyncTransferPlan{
				Enabled:      true,
				ErrorCode:    "MEMBER_MAP_MISMATCH",
				ErrorMessage: "队员【" + member.Name + "】不在同一地图，队伍同步取消。",
			}
		}
		members = append(members, member)
	}
	return SyncTransferPlan{
		Enabled: true,
		Members: members,
	}
}

func (manager *Manager) BuildDungeonResetPlan(actorRoleID string) MemberPlan {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	actorRoleID = strings.TrimSpace(actorRoleID)
	teamID := manager.teamByRoleID[actorRoleID]
	team := manager.teams[teamID]
	if team == nil {
		return MemberPlan{Allowed: true}
	}
	if team.LeaderRoleID != actorRoleID {
		return MemberPlan{
			Allowed:      false,
			ErrorCode:    "NOT_LEADER",
			ErrorMessage: "只有队长可以重置副本。",
		}
	}
	members := make([]Member, 0, len(team.Members)-1)
	for _, roleID := range team.Members {
		if roleID == actorRoleID {
			continue
		}
		member := manager.members[roleID]
		if member.Online {
			members = append(members, member)
		}
	}
	return MemberPlan{
		Allowed: true,
		Members: members,
	}
}

func (manager *Manager) BuildBattleMemberPlan(actorRoleID string, mapID string) MemberPlan {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	actorRoleID = strings.TrimSpace(actorRoleID)
	mapID = strings.TrimSpace(mapID)
	teamID := manager.teamByRoleID[actorRoleID]
	team := manager.teams[teamID]
	if team == nil {
		return MemberPlan{Allowed: true}
	}
	members := make([]Member, 0, len(team.Members)-1)
	for _, roleID := range team.Members {
		if roleID == actorRoleID {
			continue
		}
		member := manager.members[roleID]
		if !member.Online || member.MapID != mapID {
			continue
		}
		members = append(members, member)
	}
	return MemberPlan{
		Allowed: true,
		Members: members,
	}
}

func (manager *Manager) removeMemberLocked(teamID string, roleID string, action string) []Event {
	team := manager.teams[teamID]
	if team == nil {
		return nil
	}
	removed, ok := manager.members[roleID]
	delete(manager.teamByRoleID, roleID)
	nextMembers := make([]string, 0, len(team.Members))
	for _, memberRoleID := range team.Members {
		if memberRoleID != roleID {
			nextMembers = append(nextMembers, memberRoleID)
		}
	}
	team.Members = nextMembers
	events := []Event{{
		Recipients: []string{roleID},
		Clear:      &ClearPush{Reason: action},
	}}
	if len(team.Members) <= 1 {
		for _, remainingRoleID := range team.Members {
			delete(manager.teamByRoleID, remainingRoleID)
		}
		if len(team.Members) > 0 {
			events = append(events, Event{
				Recipients: append([]string{}, team.Members...),
				Clear:      &ClearPush{Reason: "disband"},
			})
		}
		delete(manager.teams, teamID)
		return events
	}
	if ok {
		events = append(events, Event{
			Recipients:  append([]string{}, team.Members...),
			MemberClear: &MemberClearPush{RoleID: removed.RoleID, Name: removed.Name},
		})
	}
	if team.LeaderRoleID == roleID {
		team.LeaderRoleID = team.Members[0]
	}
	return append(events, manager.snapshotEventsLocked(teamID)...)
}

func (manager *Manager) snapshotEventsLocked(teamID string) []Event {
	team := manager.teams[teamID]
	if team == nil {
		return nil
	}
	recipients := append([]string{}, team.Members...)
	events := []Event{{
		Recipients: recipients,
		Info: &InfoPush{
			TeamID:        team.ID,
			LeaderRoleID:  team.LeaderRoleID,
			SyncChangeMap: team.SyncChangeMap,
			MaxMembers:    MaxMembers,
		},
	}}
	members := make([]Member, 0, len(team.Members))
	for _, roleID := range team.Members {
		member := manager.members[roleID]
		member.TeamID = team.ID
		member.IsLeader = roleID == team.LeaderRoleID
		members = append(members, member)
	}
	sort.SliceStable(members, func(left, right int) bool {
		if members[left].IsLeader != members[right].IsLeader {
			return members[left].IsLeader
		}
		return members[left].Name < members[right].Name
	})
	for _, member := range members {
		events = append(events, Event{
			Recipients: recipients,
			Members:    []Member{member},
		})
	}
	return events
}

func (manager *Manager) roleIDByNameInTeamLocked(team *Team, name string) string {
	for _, roleID := range team.Members {
		if manager.members[roleID].Name == name {
			return roleID
		}
	}
	return ""
}

func (manager *Manager) resultLocked(roleID string, success bool, action string, code string, message string) Event {
	return Event{
		Recipients: []string{strings.TrimSpace(roleID)},
		Result: &ResultPush{
			Success:      success,
			Action:       action,
			ErrorCode:    code,
			ErrorMessage: message,
		},
	}
}
