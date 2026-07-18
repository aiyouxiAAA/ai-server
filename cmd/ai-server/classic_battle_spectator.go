package main

import (
	"strings"
	"sync"

	"ai-server/internal/battle"
)

type classicBattleSpectateRequest struct {
	TargetHandle string `json:"targetHandle"`
}

type classicBattleStopSpectateRequest struct {
	BattleID string `json:"battleId,omitempty"`
}

type classicBattleLookCountPush struct {
	BattleID string `json:"battleId,omitempty"`
	Count    int    `json:"count"`
}

type classicBattleSpectatorStart struct {
	ObserverRoleID string
	Start          battle.StartPush
	Cells          []battle.CellInfoPush
}

type classicBattleSpectatorStop struct {
	ObserverRoleID string
	BattleID       string
}

type classicBattleSpectatorHub struct {
	mu                sync.Mutex
	battleByObserver  map[string]string
	observersByBattle map[string]map[string]struct{}
}

var classicBattleSpectators = newClassicBattleSpectatorHub()

func newClassicBattleSpectatorHub() *classicBattleSpectatorHub {
	return &classicBattleSpectatorHub{
		battleByObserver:  make(map[string]string),
		observersByBattle: make(map[string]map[string]struct{}),
	}
}

func buildClassicBattleSpectateResult(socketSession *packetSession, request classicBattleSpectateRequest) packetResult {
	result := packetResult{handled: true}
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.battleRuntime != nil {
		return result
	}
	observerRoleID := strings.TrimSpace(socketSession.selectedRole.RoleID)
	targetHandle := strings.TrimSpace(request.TargetHandle)
	if observerRoleID == "" || targetHandle == "" || observerRoleID == targetHandle || classicBattleSpectators.isObserving(observerRoleID) {
		return result
	}
	target := classicTeamHub.connectionFor(targetHandle)
	if target.session == nil || target.session.battleRuntime == nil {
		return result
	}
	start, cells, ok := target.session.battleRuntime.SpectatorSnapshot()
	if !ok {
		return result
	}
	result.battleSpectatorStart = &classicBattleSpectatorStart{
		ObserverRoleID: observerRoleID,
		Start:          start,
		Cells:          cells,
	}
	return result
}

func buildClassicBattleStopSpectateResult(socketSession *packetSession, request classicBattleStopSpectateRequest) packetResult {
	result := packetResult{handled: true}
	if socketSession == nil || socketSession.selectedRole == nil {
		return result
	}
	observerRoleID := strings.TrimSpace(socketSession.selectedRole.RoleID)
	battleID := classicBattleSpectators.battleForObserver(observerRoleID)
	if battleID == "" || (strings.TrimSpace(request.BattleID) != "" && strings.TrimSpace(request.BattleID) != battleID) {
		return result
	}
	result.battleSpectatorStop = &classicBattleSpectatorStop{
		ObserverRoleID: observerRoleID,
		BattleID:       battleID,
	}
	return result
}

func (hub *classicBattleSpectatorHub) activate(observerRoleID string, battleID string) bool {
	observerRoleID = strings.TrimSpace(observerRoleID)
	battleID = strings.TrimSpace(battleID)
	if observerRoleID == "" || battleID == "" {
		return false
	}
	hub.mu.Lock()
	defer hub.mu.Unlock()
	if _, exists := hub.battleByObserver[observerRoleID]; exists {
		return false
	}
	observers := hub.observersByBattle[battleID]
	if observers == nil {
		observers = make(map[string]struct{})
		hub.observersByBattle[battleID] = observers
	}
	observers[observerRoleID] = struct{}{}
	hub.battleByObserver[observerRoleID] = battleID
	return true
}

func (hub *classicBattleSpectatorHub) remove(observerRoleID string) string {
	observerRoleID = strings.TrimSpace(observerRoleID)
	if observerRoleID == "" {
		return ""
	}
	hub.mu.Lock()
	defer hub.mu.Unlock()
	battleID := hub.battleByObserver[observerRoleID]
	if battleID == "" {
		return ""
	}
	delete(hub.battleByObserver, observerRoleID)
	if observers := hub.observersByBattle[battleID]; observers != nil {
		delete(observers, observerRoleID)
		if len(observers) == 0 {
			delete(hub.observersByBattle, battleID)
		}
	}
	return battleID
}

func (hub *classicBattleSpectatorHub) isObserving(observerRoleID string) bool {
	return hub.battleForObserver(observerRoleID) != ""
}

func (hub *classicBattleSpectatorHub) battleForObserver(observerRoleID string) string {
	observerRoleID = strings.TrimSpace(observerRoleID)
	hub.mu.Lock()
	defer hub.mu.Unlock()
	return hub.battleByObserver[observerRoleID]
}

func (hub *classicBattleSpectatorHub) observerRoleIDs(battleID string) []string {
	battleID = strings.TrimSpace(battleID)
	hub.mu.Lock()
	defer hub.mu.Unlock()
	observers := hub.observersByBattle[battleID]
	roleIDs := make([]string, 0, len(observers))
	for roleID := range observers {
		roleIDs = append(roleIDs, roleID)
	}
	return roleIDs
}

func (hub *classicBattleSpectatorHub) observerCount(battleID string) int {
	battleID = strings.TrimSpace(battleID)
	hub.mu.Lock()
	defer hub.mu.Unlock()
	return len(hub.observersByBattle[battleID])
}

func (hub *classicBattleSpectatorHub) broadcastLookCount(battleID string) {
	battleID = strings.TrimSpace(battleID)
	if battleID == "" {
		return
	}
	count := hub.observerCount(battleID)
	seen := make(map[*websocketWriter]struct{})
	for _, connection := range classicTeamHub.battleConnections(battleID) {
		if connection.writer == nil {
			continue
		}
		seen[connection.writer] = struct{}{}
	}
	for _, observerRoleID := range hub.observerRoleIDs(battleID) {
		if writer := classicTeamHub.writerFor(observerRoleID); writer != nil {
			seen[writer] = struct{}{}
		}
	}
	for writer := range seen {
		_ = writer.writePush(cmdClassicBattleLookCountPush, encodePayload(classicBattleLookCountPush{
			BattleID: battleID,
			Count:    count,
		}))
	}
}

func (hub *classicBattleSpectatorHub) broadcastResult(result packetResult) {
	battleID := battleIDForSpectatorResult(result)
	if battleID == "" {
		return
	}
	for _, observerRoleID := range hub.observerRoleIDs(battleID) {
		writer := classicTeamHub.writerFor(observerRoleID)
		if writer == nil || writeClassicBattleSpectatorResult(writer, result) != nil {
			hub.remove(observerRoleID)
		}
	}
	if result.battleOver != nil {
		for _, observerRoleID := range hub.observerRoleIDs(battleID) {
			hub.remove(observerRoleID)
		}
	}
}

func battleIDForSpectatorResult(result packetResult) string {
	if result.battleOver != nil {
		return strings.TrimSpace(result.battleOver.BattleID)
	}
	if len(result.battleActions) > 0 {
		return strings.TrimSpace(result.battleActions[0].BattleID)
	}
	if result.battleCommand != nil {
		return strings.TrimSpace(result.battleCommand.BattleID)
	}
	return ""
}

func writeClassicBattleSpectatorStart(writer *websocketWriter, start classicBattleSpectatorStart) error {
	if err := writer.writePush(cmdClassicBattleStartPush, encodePayload(start.Start)); err != nil {
		return err
	}
	if err := writer.writePush(cmdClassicBattleCellCountPush, encodePayload(buildClassicBattleCellCountPush(start.Start.BattleID, len(start.Cells), false))); err != nil {
		return err
	}
	for _, cell := range start.Cells {
		if err := writer.writePush(cmdClassicBattleCellInfoPush, encodePayload(cell)); err != nil {
			return err
		}
	}
	return nil
}

func writeClassicBattleSpectatorStop(writer *websocketWriter, stop classicBattleSpectatorStop) error {
	return writer.writePush(cmdClassicBattleOverPush, encodePayload(battle.OverPush{
		BattleID: stop.BattleID,
		Winner:   battle.CampEnemy,
		Result: battle.ResultPayload{
			Winner:  battle.CampEnemy,
			Escaped: true,
		},
	}))
}

func writeClassicBattleSpectatorResult(writer *websocketWriter, result packetResult) error {
	for _, buff := range result.battleBuffs {
		if err := writer.writePush(cmdClassicBattleBuffInfoPush, encodePayload(buff)); err != nil {
			return err
		}
	}
	for _, action := range result.battleActions {
		if err := writer.writePush(cmdClassicBattleActionPush, encodePayload(action)); err != nil {
			return err
		}
		for _, clearCell := range result.battleClearCells {
			if clearCell.BattleID == action.BattleID && clearCell.Handle == action.ActorHandle {
				if err := writer.writePush(cmdClassicBattleClearCellInfo, encodePayload(clearCell)); err != nil {
					return err
				}
			}
		}
	}
	for _, clearBuff := range result.battleClearBuffs {
		if err := writer.writePush(cmdClassicBattleClearBuffInfo, encodePayload(clearBuff)); err != nil {
			return err
		}
	}
	if result.battleOver != nil {
		return writer.writePush(cmdClassicBattleOverPush, encodePayload(*result.battleOver))
	}
	return nil
}
