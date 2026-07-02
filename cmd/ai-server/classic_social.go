package main

import (
	"log"
	"sort"
	"strings"
)

const classicSocialFriendSeedCapture = "tmp/capture-timeline-feature-gap-audit.json#GetFrienInfos(132)+c_friendInfo(50028):\u6050\u9f99\u6297\u72fc1|45|29|\u6218\u58eb"

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

func buildClassicSocialAddFriendResult(socketSession *packetSession, request classicSocialMutateRequest) packetResult {
	if !hasSelectedSocialRole(socketSession) {
		log.Printf("[ai-server] classic social AddFriend ignored without selected role roleName=%s", request.RoleName)
		return packetResult{handled: true}
	}
	entry, ok := normalizeFriendEntry(request)
	if !ok {
		return packetResult{handled: true}
	}
	if socketSession.friends == nil {
		socketSession.friends = make(map[string]classicSocialFriendEntry)
	}
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
	clear, removed := removeSocialEntry(socketSession.friends, request)
	if !removed {
		return packetResult{handled: true}
	}
	return packetResult{
		friendClears: []classicSocialClearEntry{clear},
		handled:      true,
	}
}

func buildClassicSocialGetFriendListResult(socketSession *packetSession) packetResult {
	if !hasSelectedSocialRole(socketSession) {
		log.Printf("[ai-server] classic social GetFrienInfos ignored without selected role")
		return packetResult{handled: true}
	}
	ensureClassicSocialCapturedFriendSeed(socketSession)
	entries := make([]classicSocialFriendEntry, 0, len(socketSession.friends))
	for _, entry := range socketSession.friends {
		entries = append(entries, entry)
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

func buildClassicSocialAddBlackResult(socketSession *packetSession, request classicSocialMutateRequest) packetResult {
	if !hasSelectedSocialRole(socketSession) {
		log.Printf("[ai-server] classic social AddBlack ignored without selected role roleName=%s", request.RoleName)
		return packetResult{handled: true}
	}
	entry, ok := normalizeBlackEntry(request)
	if !ok {
		return packetResult{handled: true}
	}
	if socketSession.blackList == nil {
		socketSession.blackList = make(map[string]classicSocialBlackEntry)
	}
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
	clear, removed := removeSocialEntry(socketSession.blackList, request)
	if !removed {
		return packetResult{handled: true}
	}
	return packetResult{
		blackClears: []classicSocialClearEntry{clear},
		handled:     true,
	}
}

func buildClassicSocialGetBlackListResult(socketSession *packetSession) packetResult {
	if !hasSelectedSocialRole(socketSession) {
		log.Printf("[ai-server] classic social GetBlackListInfos ignored without selected role")
		return packetResult{handled: true}
	}
	entries := make([]classicSocialBlackEntry, 0, len(socketSession.blackList))
	for _, entry := range socketSession.blackList {
		entries = append(entries, entry)
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
		"social-\u6050\u9f99\u6297\u72fc1": {
			classicSocialEntryBase: classicSocialEntryBase{
				RoleID:   "social-\u6050\u9f99\u6297\u72fc1",
				RoleName: "\u6050\u9f99\u6297\u72fc1",
				Level:    29,
				MapName:  "\u5e7f\u9752\u9547_1",
				Online:   true,
			},
			Relation: "friend",
		},
	}
}

func normalizeFriendEntry(request classicSocialMutateRequest) (classicSocialFriendEntry, bool) {
	base, ok := normalizeSocialEntryBase(request)
	if !ok {
		return classicSocialFriendEntry{}, false
	}
	return classicSocialFriendEntry{
		classicSocialEntryBase: base,
		Relation:               "friend",
	}, true
}

func normalizeBlackEntry(request classicSocialMutateRequest) (classicSocialBlackEntry, bool) {
	base, ok := normalizeSocialEntryBase(request)
	if !ok {
		return classicSocialBlackEntry{}, false
	}
	return classicSocialBlackEntry{
		classicSocialEntryBase: base,
		Relation:               "black",
	}, true
}

func normalizeSocialEntryBase(request classicSocialMutateRequest) (classicSocialEntryBase, bool) {
	roleName := strings.TrimSpace(request.RoleName)
	roleID := strings.TrimSpace(request.RoleID)
	if roleName == "" && roleID == "" {
		return classicSocialEntryBase{}, false
	}
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
	}, true
}

func removeSocialEntry[T any](entries map[string]T, request classicSocialMutateRequest) (classicSocialClearEntry, bool) {
	roleID := strings.TrimSpace(request.RoleID)
	roleName := strings.TrimSpace(request.RoleName)
	if entries == nil || (roleID == "" && roleName == "") {
		return classicSocialClearEntry{}, false
	}
	keys := []string{
		socialStableKey(roleID, roleName),
		roleName,
		"social-" + roleName,
	}
	for _, key := range keys {
		if key == "" {
			continue
		}
		if _, ok := entries[key]; !ok {
			continue
		}
		delete(entries, key)
		return classicSocialClearEntry{
			RoleID:   roleID,
			RoleName: roleName,
		}, true
	}
	return classicSocialClearEntry{}, false
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
