package main

import (
	"ai-server/internal/quest"
	"ai-server/internal/session"
	"strconv"
	"strings"
)

const cmdQuestGuideSnapshotPush = 1252 // Authored protocol extension; not a captured Classic command.

type questGuideEntry struct {
	Key          string `json:"key"`
	QuestID      string `json:"questId"`
	Title        string `json:"title"`
	Type         string `json:"type"`
	Phase        string `json:"phase"`
	ObjectiveID  string `json:"objectiveId"`
	Action       string `json:"action"`
	MapID        string `json:"mapId"`
	TargetHandle string `json:"targetHandle"`
	TargetName   string `json:"targetName"`
	Objective    string `json:"objective"`
	Location     string `json:"location"`
	HintID       string `json:"hintId"`
	Priority     int    `json:"priority"`
}
type questGuideSnapshot struct {
	OwnerHandle       string            `json:"ownerHandle"`
	MapID             string            `json:"mapId"`
	Entries           []questGuideEntry `json:"entries"`
	CompletedQuestIDs []string          `json:"completedQuestIds"`
}

func buildQuestGuideSnapshot(store *session.Store, socket *packetSession) *questGuideSnapshot {
	if store == nil || socket == nil || socket.selectedRole == nil || socket.playerBase == nil {
		return nil
	}
	id, role := socket.playerBase.PlayerID, socket.selectedRole.RoleID
	removed := store.RemovedQuestTitles(id, role)
	snapshot := &questGuideSnapshot{OwnerHandle: role, MapID: strconv.Itoa(socket.playerBase.MapID), Entries: []questGuideEntry{}, CompletedQuestIDs: []string{}}
	for _, q := range quest.Authored() {
		if removed[q.Info.Title] {
			snapshot.CompletedQuestIDs = append(snapshot.CompletedQuestIDs, q.Info.ID)
		}
	}
	for _, g := range quest.Guides() {
		q, _ := quest.FindAuthored(g.QuestID)
		phase := authoredQuestStatus(store, socket, q)
		if phase == "locked" || phase == "completed" {
			continue
		}
		if phase == "accepted" && q.Rule.Kind != "visit" && store.AuthoredQuestReady(id, role, q.Info.ID) {
			phase = "ready"
		}
		if phase != g.Phase {
			continue
		}
		// Offer guidance belongs to the current village; accepted objectives can point back from another map.
		if phase == "available" && socket.playerBase.MapID != q.MapID {
			continue
		}
		npc := q.Start
		if g.TargetRole == "finish" {
			npc = q.Finish
		}
		targetHandle, targetName := npc.Handle(), npc.Name
		if g.Action == "open_page" {
			targetHandle, targetName = g.TargetRole, g.Location
		}
		progress, _ := store.QuestProgress(id, role, q.Info.Title)
		text := strings.ReplaceAll(g.Text, "{progress}", strconv.Itoa(progress))
		// 'ready' chooses a guide row; the client still sees the accepted quest phase.
		if phase == "ready" {
			phase = "accepted"
		}
		snapshot.Entries = append(snapshot.Entries, questGuideEntry{
			Key: g.QuestID + ":" + g.ObjectiveID, QuestID: g.QuestID, Title: q.Info.Title, Type: q.Info.Type,
			Phase: phase, ObjectiveID: g.ObjectiveID, Action: g.Action, MapID: strconv.Itoa(g.MapID),
			TargetHandle: targetHandle, TargetName: targetName, Objective: text, Location: g.Location,
			HintID: g.HintID, Priority: g.Priority,
		})
	}
	return snapshot
}
