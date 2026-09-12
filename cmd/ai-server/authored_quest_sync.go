package main

import (
	"ai-server/internal/protocol"
	"ai-server/internal/quest"
	"ai-server/internal/session"
)

// Observe real operation results. Guide refresh never grants objective progress.
func handlePacketWithSession(store *session.Store, packet protocol.Packet, socket *packetSession) packetResult {
	result := handlePacketWithSessionCore(store, packet, socket)
	removeRetiredQuestContent(&result)
	refreshAuthoredQuestResult(store, socket, &result)
	return result
}

func refreshAuthoredQuestResult(store *session.Store, socket *packetSession, result *packetResult) {
	if store == nil || socket == nil || socket.playerBase == nil || socket.selectedRole == nil {
		return
	}
	if len(result.itemInfos) == 0 && len(result.itemClears) == 0 && result.roleState == nil && len(result.questInfos) == 0 && len(result.questClears) == 0 {
		return
	}
	result.questGuide = buildQuestGuideSnapshot(store, socket)
	states := allAuthoredQuestStates(store, socket)
	for _, state := range states {
		found := false
		for i := range result.questStates {
			if result.questStates[i].Handle == state.Handle {
				result.questStates[i] = state
				found = true
				break
			}
		}
		if !found {
			result.questStates = append(result.questStates, state)
		}
	}
	accepted := store.AcceptedQuestTitles(socket.playerBase.PlayerID, socket.selectedRole.RoleID)
	for _, q := range quest.Authored() {
		if !accepted[q.Info.Title] {
			continue
		}
		found := false
		for _, info := range result.questInfos {
			if info.QuestID == q.Info.ID {
				found = true
				break
			}
		}
		if !found {
			result.questInfos = append(result.questInfos, classicQuestInfoForRole(store, socket, q.Info))
		}
	}
}
