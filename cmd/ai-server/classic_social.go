package main

import (
	"log"
	"strings"
)

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
