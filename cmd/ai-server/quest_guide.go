package main

import (
	"ai-server/internal/quest"
	"ai-server/internal/session"
	"strconv"
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
	accepted, removed := store.AcceptedQuestTitles(id, role), store.RemovedQuestTitles(id, role)
	snapshot := &questGuideSnapshot{OwnerHandle: role, MapID: strconv.Itoa(socket.playerBase.MapID), Entries: []questGuideEntry{}, CompletedQuestIDs: []string{}}
	for _, q := range quest.Authored() {
		if removed[q.Info.Title] {
			snapshot.CompletedQuestIDs = append(snapshot.CompletedQuestIDs, q.Info.ID)
		}
	}
	for _, g := range quest.Guides() {
		q, _ := quest.FindAuthored(g.QuestID)
		if removed[q.Info.Title] || socket.playerBase.Level < q.Info.Level {
			continue
		}
		phase := "available"
		if accepted[q.Info.Title] {
			phase = "accepted"
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
		snapshot.Entries = append(snapshot.Entries, questGuideEntry{
			Key: g.QuestID + ":" + g.ObjectiveID, QuestID: g.QuestID, Title: q.Info.Title, Type: q.Info.Type,
			Phase: phase, ObjectiveID: g.ObjectiveID, Action: g.Action, MapID: strconv.Itoa(g.MapID),
			TargetHandle: npc.Handle(), TargetName: npc.Name, Objective: g.Text, Location: g.Location,
			HintID: g.HintID, Priority: g.Priority,
		})
	}
	return snapshot
}
