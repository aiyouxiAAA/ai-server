package main

import (
	"log"
	"sort"
	"strings"

	"ai-server/internal/session"
	"ai-server/internal/world"
)

const classicSocialFriendSeedCapture = "tmp/capture-timeline-feature-gap-audit.json#GetFrienInfos(132)+c_friendInfo(50028):恐龙抗狼1|45|29|战士"

type classicSocialEntryBase struct {
	RoleID   string `json:"roleId"`
	RoleName string `json:"roleName"`
	Level    int    `json:"level"`
	MapName  string `json:"mapName,omitempty"`
	Online   bool   `json:"online"`
}

type classicSocialFriendEntry struct {
	classicSocialEntryBase
	Relation string `json:"relation"`
}

type classicSocialBlackEntry struct {
	classicSocialEntryBase
	Relation string `json:"relation"`
}

type classicSocialEnemyEntry struct {
	classicSocialEntryBase
	Relation string `json:"relation"`
}

type classicSocialClearEntry struct {
	RoleID   string `json:"roleId,omitempty"`
	RoleName string `json:"roleName,omitempty"`
}

type classicSocialMutateRequest struct {
	RoleID   string `json:"roleId,omitempty"`
	RoleName string `json:"roleName"`
}

type classicSocialTradeRequest struct {
	Handle   string `json:"handle"`
	RoleID   string `json:"roleId,omitempty"`
	RoleName string `json:"roleName"`
}

const (
	classicSocialTradeTemporarilyClosed = "交易临时关闭"
	classicSocialTradeClosedCapture     = "20260606_200103_131_session_00324/20260606_200109_321_conn_0002#TradeRequest(108)+c_Error(49999)"
)

func buildClassicSocialAddFriendResult(store *session.Store, socketSession *packetSession, request classicSocialMutateRequest) packetResult {
	if !hasSelectedSocialRole(socketSession) {
		log.Printf("[ai-server] classic social AddFriend ignored without selected role roleName=%s", request.RoleName)
		return packetResult{handled: true}
	}
	entry, ok := normalizeFriendEntry(store, request)
	if !ok {
		return packetResult{handled: true}
	}
	if socketSession.friends == nil {
		socketSession.friends = make(map[string]classicSocialFriendEntry)
	}
	// 统一以 roleId 为主键；若历史以名字存过，先清掉避免双份。
	pruneSocialFriendDuplicates(socketSession.friends, entry)
	socketSession.friends[socialStableKey(entry.RoleID, entry.RoleName)] = entry
	return packetResult{
		friendInfos: []classicSocialFriendEntry{entry},
		handled:     true,
	}
}

func buildClassicSocialRemoveFriendResult(socketSession *packetSession, request classicSocialMutateRequest) packetResult {
	if !hasSelectedSocialRole(socketSession) {
		log.Printf("[ai-server] classic social RemoveFriend ignored without selected role roleName=%s", request.RoleName)
		return packetResult{handled: true}
	}
	clear, removed := removeFriendEntry(socketSession.friends, request)
	if !removed {
		return packetResult{handled: true}
	}
	return packetResult{
		friendClears: []classicSocialClearEntry{clear},
		handled:      true,
	}
}

func buildClassicSocialGetFriendListResult(store *session.Store, socketSession *packetSession) packetResult {
	if !hasSelectedSocialRole(socketSession) {
		log.Printf("[ai-server] classic social GetFrienInfos ignored without selected role")
		return packetResult{handled: true}
	}
	ensureClassicSocialCapturedFriendSeed(socketSession)
	entries := make([]classicSocialFriendEntry, 0, len(socketSession.friends))
	for key, entry := range socketSession.friends {
		// 列表拉取时按在线 hub / store 刷新等级、地图与在线态，对齐源码 friendMsg 后按 mapId 段判定在线。
		refreshed := refreshFriendEntry(store, entry)
		socketSession.friends[key] = refreshed
		entries = append(entries, refreshed)
	}
	sort.Slice(entries, func(left int, right int) bool {
		if entries[left].RoleName != entries[right].RoleName {
			return entries[left].RoleName < entries[right].RoleName
		}
		return entries[left].RoleID < entries[right].RoleID
	})
	return packetResult{
		friendInfos: entries,
		handled:     true,
	}
}

func buildClassicSocialAddBlackResult(store *session.Store, socketSession *packetSession, request classicSocialMutateRequest) packetResult {
	if !hasSelectedSocialRole(socketSession) {
		log.Printf("[ai-server] classic social AddBlack ignored without selected role roleName=%s", request.RoleName)
		return packetResult{handled: true}
	}
	entry, ok := normalizeBlackEntry(store, request)
	if !ok {
		return packetResult{handled: true}
	}
	if socketSession.blackList == nil {
		socketSession.blackList = make(map[string]classicSocialBlackEntry)
	}
	pruneSocialBlackDuplicates(socketSession.blackList, entry)
	socketSession.blackList[socialStableKey(entry.RoleID, entry.RoleName)] = entry
	return packetResult{
		blackInfos: []classicSocialBlackEntry{entry},
		handled:    true,
	}
}

func buildClassicSocialRemoveBlackResult(socketSession *packetSession, request classicSocialMutateRequest) packetResult {
	if !hasSelectedSocialRole(socketSession) {
		log.Printf("[ai-server] classic social RemoveBlack ignored without selected role roleName=%s", request.RoleName)
		return packetResult{handled: true}
	}
	clear, removed := removeBlackEntry(socketSession.blackList, request)
	if !removed {
		return packetResult{handled: true}
	}
	return packetResult{
		blackClears: []classicSocialClearEntry{clear},
		handled:     true,
	}
}

func buildClassicSocialGetBlackListResult(store *session.Store, socketSession *packetSession) packetResult {
	if !hasSelectedSocialRole(socketSession) {
		log.Printf("[ai-server] classic social GetBlackListInfos ignored without selected role")
		return packetResult{handled: true}
	}
	entries := make([]classicSocialBlackEntry, 0, len(socketSession.blackList))
	for key, entry := range socketSession.blackList {
		refreshed := refreshBlackEntry(store, entry)
		socketSession.blackList[key] = refreshed
		entries = append(entries, refreshed)
	}
	sort.Slice(entries, func(left int, right int) bool {
		if entries[left].RoleName != entries[right].RoleName {
			return entries[left].RoleName < entries[right].RoleName
		}
		return entries[left].RoleID < entries[right].RoleID
	})
	return packetResult{
		blackInfos: entries,
		handled:    true,
	}
}

func buildClassicSocialTradeRequestResult(socketSession *packetSession, request classicSocialTradeRequest) packetResult {
	if !hasSelectedSocialRole(socketSession) {
		log.Printf("[ai-server] classic social TradeRequest ignored without selected role handle=%s roleName=%s", request.Handle, request.RoleName)
		return packetResult{handled: true}
	}
	return packetResult{
		errorMessages: []classicTownErrorPush{{
			Msg:           classicSocialTradeTemporarilyClosed,
			SourceCapture: classicSocialTradeClosedCapture,
			Partial:       true,
		}},
		handled: true,
	}
}

func ensureClassicSocialCapturedFriendSeed(socketSession *packetSession) {
	if socketSession == nil || socketSession.friends != nil {
		return
	}
	socketSession.friends = map[string]classicSocialFriendEntry{
		"social-恐龙抗狼1": {
			classicSocialEntryBase: classicSocialEntryBase{
				RoleID:   "social-恐龙抗狼1",
				RoleName: "恐龙抗狼1",
				Level:    29,
				MapName:  "广青镇_1",
				Online:   true,
			},
			Relation: "friend",
		},
	}
}

func normalizeFriendEntry(store *session.Store, request classicSocialMutateRequest) (classicSocialFriendEntry, bool) {
	base, ok := resolveSocialEntryBase(store, request)
	if !ok {
		return classicSocialFriendEntry{}, false
	}
	return classicSocialFriendEntry{
		classicSocialEntryBase: base,
		Relation:               "friend",
	}, true
}

func normalizeBlackEntry(store *session.Store, request classicSocialMutateRequest) (classicSocialBlackEntry, bool) {
	base, ok := resolveSocialEntryBase(store, request)
	if !ok {
		return classicSocialBlackEntry{}, false
	}
	return classicSocialBlackEntry{
		classicSocialEntryBase: base,
		Relation:               "black",
	}, true
}

// resolveSocialEntryBase 按 在线 hub → store → 最小离线占位 解析目标角色。
// 源码 AddFriend(roleName) 只带名字；玩家菜单可带 roleId。抓包 c_friendInfo 为 name|mapId|level|voc。
func resolveSocialEntryBase(store *session.Store, request classicSocialMutateRequest) (classicSocialEntryBase, bool) {
	roleName := strings.TrimSpace(request.RoleName)
	roleID := strings.TrimSpace(request.RoleID)
	if roleName == "" && roleID == "" {
		return classicSocialEntryBase{}, false
	}

	if base, ok := resolveSocialEntryFromOnline(roleID, roleName); ok {
		return base, true
	}
	if base, ok := resolveSocialEntryFromStore(store, roleID, roleName); ok {
		return base, true
	}
	return fallbackSocialEntryBase(roleID, roleName), true
}

func resolveSocialEntryFromOnline(roleID string, roleName string) (classicSocialEntryBase, bool) {
	if roleID != "" {
		if conn, ok := worldSceneHub.connectionFor(roleID); ok {
			if base, ok := socialEntryFromOnlineConnection(conn); ok {
				return base, true
			}
		}
		return classicSocialEntryBase{}, false
	}
	if roleName != "" {
		if conn, ok := worldSceneHub.connectionByDisplayName(roleName); ok {
			if base, ok := socialEntryFromOnlineConnection(conn); ok {
				return base, true
			}
		}
	}
	return classicSocialEntryBase{}, false
}

func socialEntryFromOnlineConnection(conn worldSceneConnection) (classicSocialEntryBase, bool) {
	if conn.session == nil || conn.session.selectedRole == nil {
		return classicSocialEntryBase{}, false
	}
	role := conn.session.selectedRole
	displayName := strings.TrimSpace(role.DisplayName)
	if displayName == "" && conn.session.playerBase != nil {
		displayName = strings.TrimSpace(conn.session.playerBase.DisplayName)
	}
	if displayName == "" {
		displayName = strings.TrimSpace(role.RoleID)
	}
	level := role.Level
	if level <= 0 && conn.session.playerBase != nil {
		level = conn.session.playerBase.Level
	}
	if level <= 0 {
		level = 1
	}
	return classicSocialEntryBase{
		RoleID:   strings.TrimSpace(role.RoleID),
		RoleName: displayName,
		Level:    level,
		MapName:  world.MapNameForMapID(conn.mapID),
		Online:   true,
	}, true
}

func resolveSocialEntryFromStore(store *session.Store, roleID string, roleName string) (classicSocialEntryBase, bool) {
	if store == nil {
		return classicSocialEntryBase{}, false
	}
	if roleID != "" {
		if _, role, ok := store.FindRoleByID(roleID); ok {
			return socialEntryFromStoredRole(role, false), true
		}
		return classicSocialEntryBase{}, false
	}
	if roleName != "" {
		if _, role, ok := store.FindRoleByDisplayName(roleName); ok {
			return socialEntryFromStoredRole(role, false), true
		}
	}
	return classicSocialEntryBase{}, false
}

func socialEntryFromStoredRole(role session.RoleSummary, online bool) classicSocialEntryBase {
	displayName := strings.TrimSpace(role.DisplayName)
	if displayName == "" {
		displayName = strings.TrimSpace(role.RoleID)
	}
	level := role.Level
	if level <= 0 {
		level = 1
	}
	mapName := ""
	if online || role.MapID > 0 {
		mapName = world.MapNameForMapID(role.MapID)
	}
	return classicSocialEntryBase{
		RoleID:   strings.TrimSpace(role.RoleID),
		RoleName: displayName,
		Level:    level,
		MapName:  mapName,
		Online:   online,
	}
}

func fallbackSocialEntryBase(roleID string, roleName string) classicSocialEntryBase {
	if roleName == "" {
		roleName = roleID
	}
	if roleID == "" {
		roleID = "social-" + roleName
	}
	return classicSocialEntryBase{
		RoleID:   roleID,
		RoleName: roleName,
		Level:    1,
		MapName:  "云隐村",
		Online:   false,
	}
}

func refreshFriendEntry(store *session.Store, entry classicSocialFriendEntry) classicSocialFriendEntry {
	base := refreshSocialEntryBase(store, entry.classicSocialEntryBase)
	entry.classicSocialEntryBase = base
	if entry.Relation == "" {
		entry.Relation = "friend"
	}
	return entry
}

func refreshBlackEntry(store *session.Store, entry classicSocialBlackEntry) classicSocialBlackEntry {
	base := refreshSocialEntryBase(store, entry.classicSocialEntryBase)
	entry.classicSocialEntryBase = base
	if entry.Relation == "" {
		entry.Relation = "black"
	}
	return entry
}

func refreshSocialEntryBase(store *session.Store, entry classicSocialEntryBase) classicSocialEntryBase {
	roleID := strings.TrimSpace(entry.RoleID)
	roleName := strings.TrimSpace(entry.RoleName)
	if base, ok := resolveSocialEntryFromOnline(roleID, roleName); ok {
		return base
	}
	if base, ok := resolveSocialEntryFromStore(store, roleID, roleName); ok {
		// 离线时保留上次 mapName（若 store 无 map），避免列表空白。
		if base.MapName == "" {
			base.MapName = entry.MapName
		}
		return base
	}
	// 抓包 seed / 未知名占位：保留上次快照（含 online/map/level），不强行改写。
	if entry.Level <= 0 {
		entry.Level = 1
	}
	return entry
}

func removeFriendEntry(entries map[string]classicSocialFriendEntry, request classicSocialMutateRequest) (classicSocialClearEntry, bool) {
	if entries == nil {
		return classicSocialClearEntry{}, false
	}
	for key, entry := range entries {
		if !socialIdentityMatches(entry.RoleID, entry.RoleName, key, request) {
			continue
		}
		delete(entries, key)
		// 再扫一遍同 identity 的残留键，避免 roleId/name 双写。
		for residualKey, residual := range entries {
			if socialIdentityMatches(residual.RoleID, residual.RoleName, residualKey, classicSocialMutateRequest{
				RoleID:   entry.RoleID,
				RoleName: entry.RoleName,
			}) {
				delete(entries, residualKey)
			}
		}
		return classicSocialClearEntry{
			RoleID:   entry.RoleID,
			RoleName: entry.RoleName,
		}, true
	}
	return classicSocialClearEntry{}, false
}

func removeBlackEntry(entries map[string]classicSocialBlackEntry, request classicSocialMutateRequest) (classicSocialClearEntry, bool) {
	if entries == nil {
		return classicSocialClearEntry{}, false
	}
	for key, entry := range entries {
		if !socialIdentityMatches(entry.RoleID, entry.RoleName, key, request) {
			continue
		}
		delete(entries, key)
		for residualKey, residual := range entries {
			if socialIdentityMatches(residual.RoleID, residual.RoleName, residualKey, classicSocialMutateRequest{
				RoleID:   entry.RoleID,
				RoleName: entry.RoleName,
			}) {
				delete(entries, residualKey)
			}
		}
		return classicSocialClearEntry{
			RoleID:   entry.RoleID,
			RoleName: entry.RoleName,
		}, true
	}
	return classicSocialClearEntry{}, false
}

func socialIdentityMatches(roleID string, roleName string, mapKey string, request classicSocialMutateRequest) bool {
	reqID := strings.TrimSpace(request.RoleID)
	reqName := strings.TrimSpace(request.RoleName)
	roleID = strings.TrimSpace(roleID)
	roleName = strings.TrimSpace(roleName)
	mapKey = strings.TrimSpace(mapKey)
	if reqID == "" && reqName == "" {
		return false
	}
	if reqID != "" {
		return roleID == reqID || mapKey == reqID || isLegacySocialIdentity(roleID, mapKey, reqName)
	}
	return roleName == reqName || mapKey == reqName || isLegacySocialIdentity(roleID, mapKey, reqName)
}

func isLegacySocialIdentity(roleID string, mapKey string, roleName string) bool {
	if roleName == "" {
		return false
	}
	legacyRoleID := "social-" + roleName
	return roleID == legacyRoleID || mapKey == legacyRoleID
}

func pruneSocialFriendDuplicates(entries map[string]classicSocialFriendEntry, entry classicSocialFriendEntry) {
	for key, existing := range entries {
		if socialIdentityMatches(existing.RoleID, existing.RoleName, key, classicSocialMutateRequest{
			RoleID:   entry.RoleID,
			RoleName: entry.RoleName,
		}) {
			delete(entries, key)
		}
	}
}

func pruneSocialBlackDuplicates(entries map[string]classicSocialBlackEntry, entry classicSocialBlackEntry) {
	for key, existing := range entries {
		if socialIdentityMatches(existing.RoleID, existing.RoleName, key, classicSocialMutateRequest{
			RoleID:   entry.RoleID,
			RoleName: entry.RoleName,
		}) {
			delete(entries, key)
		}
	}
}

func socialStableKey(roleID string, roleName string) string {
	roleID = strings.TrimSpace(roleID)
	if roleID != "" {
		return roleID
	}
	return strings.TrimSpace(roleName)
}

func hasSelectedSocialRole(socketSession *packetSession) bool {
	return socketSession != nil && socketSession.selectedRole != nil && socketSession.playerBase != nil
}
