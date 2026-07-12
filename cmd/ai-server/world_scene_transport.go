package main

import (
	"context"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"ai-server/internal/world"
)

const (
	worldSceneHeroSpace                 = 450
	worldSceneScreenRoleLimit           = 20
	worldScenePositionReconcileInterval = 500 * time.Millisecond
)

// worldSceneHub 是"同 mapId 在线玩家"的注册表。
//
// 设计与 classicTeamHub 平行:team hub 是"Invite 进队 ≤4 人"语义,
// scene hub 是"按 mapId 自动聚合陌生人"语义,职责不重叠,不复用 team hub。
//
// 注册时机:玩家选角成功 / 换角色 / 切图后,在 main.go register 区调用。
// 注销时机:WebSocket 断开的 defer。
// 互推语义:
//   - 新玩家 register 到某 mapId 时,把已在同 mapId 的邻居 createRole 推给自己,
//     同时把自己的 createRole 推给所有邻居。
//   - 玩家 unregister(下线/换图)时,给原 mapId 邻居推 removeRole。
var worldSceneHub = newWorldSceneConnectionHub()

type worldSceneConnectionHub struct {
	mu          sync.Mutex
	connections map[string]worldSceneConnection
}

type worldSceneConnection struct {
	writer  *websocketWriter
	session *packetSession
	mapID   int
	spawn   world.SpawnPoint
	visible map[string]struct{}
}

type worldScenePushAction struct {
	writer       *websocketWriter
	recipientID  string
	createRole   *world.RolePush
	removeHandle string
	moveRole     *world.RoleMovePush
}

func newWorldSceneConnectionHub() *worldSceneConnectionHub {
	return &worldSceneConnectionHub{
		connections: make(map[string]worldSceneConnection),
	}
}

func (hub *worldSceneConnectionHub) register(roleID string, mapID int, writer *websocketWriter, socketSession *packetSession, spawn world.SpawnPoint) {
	if roleID == "" || writer == nil {
		return
	}
	hub.mu.Lock()
	defer hub.mu.Unlock()
	hub.connections[roleID] = worldSceneConnection{
		writer:  writer,
		session: socketSession,
		mapID:   mapID,
		spawn:   spawn,
		visible: make(map[string]struct{}),
	}
}

func (hub *worldSceneConnectionHub) updatePosition(roleID string, mapID int, spawn world.SpawnPoint) bool {
	if roleID == "" {
		return false
	}
	hub.mu.Lock()
	defer hub.mu.Unlock()
	conn, ok := hub.connections[roleID]
	if !ok || conn.mapID != mapID {
		return false
	}
	conn.spawn = spawn
	hub.connections[roleID] = conn
	return true
}

// unregister 返回该 roleID 之前的 mapID 与是否曾注册,供调用方广播 removeRole。
func (hub *worldSceneConnectionHub) unregister(roleID string) (int, bool) {
	if roleID == "" {
		return 0, false
	}
	hub.mu.Lock()
	defer hub.mu.Unlock()
	conn, ok := hub.connections[roleID]
	if !ok {
		return 0, false
	}
	delete(hub.connections, roleID)
	return conn.mapID, true
}

func (hub *worldSceneConnectionHub) writerFor(roleID string) *websocketWriter {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	return hub.connections[roleID].writer
}

func (hub *worldSceneConnectionHub) connectionFor(roleID string) (worldSceneConnection, bool) {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	conn, ok := hub.connections[roleID]
	return conn, ok
}

func (hub *worldSceneConnectionHub) connectionByDisplayName(displayName string) (worldSceneConnection, bool) {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	name := strings.TrimSpace(displayName)
	if name == "" {
		return worldSceneConnection{}, false
	}
	var found worldSceneConnection
	foundMatch := false
	for _, conn := range hub.connections {
		if conn.session == nil || conn.session.selectedRole == nil {
			continue
		}
		if strings.TrimSpace(conn.session.selectedRole.DisplayName) != name {
			continue
		}
		if foundMatch {
			return worldSceneConnection{}, false
		}
		found = conn
		foundMatch = true
	}
	return found, foundMatch
}

func (hub *worldSceneConnectionHub) broadcastChatToAll(message classicTownChatMessagePush) {
	type recipient struct {
		roleID string
		writer *websocketWriter
	}
	hub.mu.Lock()
	recipients := make([]recipient, 0, len(hub.connections))
	for roleID, conn := range hub.connections {
		if conn.writer == nil || conn.writer.conn == nil {
			continue
		}
		recipients = append(recipients, recipient{roleID: roleID, writer: conn.writer})
	}
	hub.mu.Unlock()

	for _, item := range recipients {
		if err := item.writer.writePush(cmdClassicTownChatMessagePush, encodePayload(message)); err != nil {
			log.Printf("[ai-server] world scene chat failed roleId=%s: %v", item.roleID, err)
		}
	}
}

func worldSceneDistanceSquared(left world.SpawnPoint, right world.SpawnPoint) int {
	dx := left.X - right.X
	dy := left.Y - right.Y
	return dx*dx + dy*dy
}

func worldSceneInHeroSpace(left world.SpawnPoint, right world.SpawnPoint) bool {
	return worldSceneDistanceSquared(left, right) < worldSceneHeroSpace*worldSceneHeroSpace
}

func worldSceneCanBuildRolePush(conn worldSceneConnection) bool {
	return conn.writer != nil && conn.session != nil && conn.session.selectedRole != nil && conn.session.playerBase != nil
}

func worldSceneBuildRolePush(conn worldSceneConnection) world.RolePush {
	return world.BuildPlayerRolePush(*conn.session.selectedRole, *conn.session.playerBase, conn.spawn)
}

func worldSceneTeamRoleIDSet(roleID string) map[string]struct{} {
	recipients, ok := classicTeamManager.RecipientsForTeam(roleID)
	if !ok {
		return nil
	}
	teamRoleIDs := make(map[string]struct{}, len(recipients))
	for _, recipientID := range recipients {
		if recipientID == "" || recipientID == roleID {
			continue
		}
		teamRoleIDs[recipientID] = struct{}{}
	}
	return teamRoleIDs
}

func (hub *worldSceneConnectionHub) syncMapVisibility(mapID int) []worldScenePushAction {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	actions := make([]worldScenePushAction, 0)
	roleIDs := make([]string, 0, len(hub.connections))
	for roleID, conn := range hub.connections {
		if conn.mapID == mapID && conn.writer != nil {
			roleIDs = append(roleIDs, roleID)
		}
	}
	sort.Strings(roleIDs)
	for _, observerID := range roleIDs {
		actions = append(actions, hub.syncObserverVisibilityLocked(observerID)...)
	}
	return actions
}

func (hub *worldSceneConnectionHub) syncObserverVisibilityLocked(observerID string) []worldScenePushAction {
	observer, ok := hub.connections[observerID]
	if !ok || observer.writer == nil {
		return nil
	}
	if observer.visible == nil {
		observer.visible = make(map[string]struct{})
	}

	type candidate struct {
		roleID   string
		distance int
		conn     worldSceneConnection
	}
	teamRoleIDs := worldSceneTeamRoleIDSet(observerID)
	forced := make(map[string]worldSceneConnection, len(teamRoleIDs))
	candidates := make([]candidate, 0, len(hub.connections))
	for roleID, conn := range hub.connections {
		if roleID == observerID || conn.mapID != observer.mapID || !worldSceneCanBuildRolePush(conn) {
			continue
		}
		if _, ok := teamRoleIDs[roleID]; ok {
			forced[roleID] = conn
			continue
		}
		if !worldSceneInHeroSpace(observer.spawn, conn.spawn) {
			continue
		}
		candidates = append(candidates, candidate{
			roleID:   roleID,
			distance: worldSceneDistanceSquared(observer.spawn, conn.spawn),
			conn:     conn,
		})
	}
	sort.Slice(candidates, func(left int, right int) bool {
		if candidates[left].distance != candidates[right].distance {
			return candidates[left].distance < candidates[right].distance
		}
		return candidates[left].roleID < candidates[right].roleID
	})

	desired := make(map[string]worldSceneConnection, len(forced)+worldSceneScreenRoleLimit)
	for roleID, conn := range forced {
		desired[roleID] = conn
	}
	for index, item := range candidates {
		if index >= worldSceneScreenRoleLimit {
			break
		}
		desired[item.roleID] = item.conn
	}

	actions := make([]worldScenePushAction, 0)
	for visibleID := range observer.visible {
		if _, ok := desired[visibleID]; ok {
			continue
		}
		delete(observer.visible, visibleID)
		actions = append(actions, worldScenePushAction{
			writer:       observer.writer,
			recipientID:  observerID,
			removeHandle: visibleID,
		})
	}
	for roleID, conn := range desired {
		if _, ok := observer.visible[roleID]; ok {
			continue
		}
		observer.visible[roleID] = struct{}{}
		push := worldSceneBuildRolePush(conn)
		actions = append(actions, worldScenePushAction{
			writer:      observer.writer,
			recipientID: observerID,
			createRole:  &push,
		})
	}
	hub.connections[observerID] = observer
	return actions
}

func (hub *worldSceneConnectionHub) removeRoleFromVisibility(mapID int, handle string) []worldScenePushAction {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	actions := make([]worldScenePushAction, 0)
	for observerID, observer := range hub.connections {
		if observer.mapID != mapID || observer.writer == nil || observer.visible == nil {
			continue
		}
		if _, ok := observer.visible[handle]; !ok {
			continue
		}
		delete(observer.visible, handle)
		hub.connections[observerID] = observer
		actions = append(actions, worldScenePushAction{
			writer:       observer.writer,
			recipientID:  observerID,
			removeHandle: handle,
		})
	}
	return actions
}

func (hub *worldSceneConnectionHub) moveRoleActions(mapID int, actorRoleID string, push world.RoleMovePush) []worldScenePushAction {
	actions := hub.syncMapVisibility(mapID)
	hub.mu.Lock()
	defer hub.mu.Unlock()
	for observerID, observer := range hub.connections {
		if observerID == actorRoleID || observer.mapID != mapID || observer.writer == nil || observer.visible == nil {
			continue
		}
		if _, ok := observer.visible[actorRoleID]; !ok {
			continue
		}
		move := push
		actions = append(actions, worldScenePushAction{
			writer:      observer.writer,
			recipientID: observerID,
			moveRole:    &move,
		})
	}
	return actions
}

// positionReconcileActions 按原服 c_MoveRole(50012) 的静止 Run 节奏，
// 给每个观察者重放其当前可见玩家的最新权威坐标。
//
// 可见性先通过 syncMapVisibility 刷新，因此 AOI、screenRole 上限和同图队友
// 强制可见性仍由唯一的 visible 集合裁决；不会给自己或不可见玩家推送。
func (hub *worldSceneConnectionHub) positionReconcileActions() []worldScenePushAction {
	hub.mu.Lock()
	mapIDSet := make(map[int]struct{}, len(hub.connections))
	for _, conn := range hub.connections {
		if conn.writer != nil {
			mapIDSet[conn.mapID] = struct{}{}
		}
	}
	hub.mu.Unlock()

	mapIDs := make([]int, 0, len(mapIDSet))
	for mapID := range mapIDSet {
		mapIDs = append(mapIDs, mapID)
	}
	sort.Ints(mapIDs)

	actions := make([]worldScenePushAction, 0)
	for _, mapID := range mapIDs {
		actions = append(actions, hub.syncMapVisibility(mapID)...)
	}

	hub.mu.Lock()
	defer hub.mu.Unlock()
	observerIDs := make([]string, 0, len(hub.connections))
	for observerID := range hub.connections {
		observerIDs = append(observerIDs, observerID)
	}
	sort.Strings(observerIDs)
	for _, observerID := range observerIDs {
		observer := hub.connections[observerID]
		if observer.writer == nil || observer.visible == nil {
			continue
		}
		visibleRoleIDs := make([]string, 0, len(observer.visible))
		for roleID := range observer.visible {
			visibleRoleIDs = append(visibleRoleIDs, roleID)
		}
		sort.Strings(visibleRoleIDs)
		for _, roleID := range visibleRoleIDs {
			if roleID == observerID {
				continue
			}
			actor, ok := hub.connections[roleID]
			if !ok || actor.mapID != observer.mapID || !worldSceneCanBuildRolePush(actor) {
				continue
			}
			move := world.RoleMovePush{
				Handle: roleID,
				Type:   "Run",
				X:      actor.spawn.X,
				Y:      actor.spawn.Y,
				TX:     actor.spawn.X,
				TY:     actor.spawn.Y,
				MapID:  strconv.Itoa(actor.mapID),
			}
			actions = append(actions, worldScenePushAction{
				writer:      observer.writer,
				recipientID: observerID,
				moveRole:    &move,
			})
		}
	}
	return actions
}

// startWorldScenePositionReconcileLoop 启动服务端权威位置校正循环。
// 调用者持有返回的 stop，服务退出时必须取消，避免后台循环脱离生命周期。
func startWorldScenePositionReconcileLoop(hub *worldSceneConnectionHub) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())
	if hub == nil {
		return cancel
	}
	ticker := time.NewTicker(worldScenePositionReconcileInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				writeWorldSceneActions(hub.positionReconcileActions())
			}
		}
	}()
	return cancel
}

func writeWorldSceneActions(actions []worldScenePushAction) {
	for _, action := range actions {
		if action.writer == nil {
			continue
		}
		if action.createRole != nil {
			if err := action.writer.writePush(cmdClassicTownCreateRolePush, encodePayload(*action.createRole)); err != nil {
				log.Printf("[ai-server] world scene createRole failed roleId=%s: %v", action.recipientID, err)
			}
		}
		if action.removeHandle != "" {
			if err := action.writer.writePush(cmdClassicTownRemoveRolePush, encodePayload(action.removeHandle)); err != nil {
				log.Printf("[ai-server] world scene removeRole failed roleId=%s: %v", action.recipientID, err)
			}
		}
		if action.moveRole != nil {
			if err := action.writer.writePush(cmdClassicTownMoveRolePush, encodePayload(*action.moveRole)); err != nil {
				log.Printf("[ai-server] world scene moveRole failed roleId=%s: %v", action.recipientID, err)
			}
		}
	}
}

// roleIDsForMap 返回当前在某 mapId 上的所有在线玩家 roleID(排除 exceptRoleID)。
// 调用方持有锁外做后续 writePush,避免长时间持锁。
func (hub *worldSceneConnectionHub) roleIDsForMap(mapID int, exceptRoleID string) []string {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	roleIDs := make([]string, 0, len(hub.connections))
	for roleID, conn := range hub.connections {
		if roleID == exceptRoleID {
			continue
		}
		if conn.mapID != mapID {
			continue
		}
		roleIDs = append(roleIDs, roleID)
	}
	return roleIDs
}

// neighborsInMap 返回当前在某 mapId 上的所有邻居连接快照(排除 exceptRoleID)。
// 用于"进图时把邻居 createRole 推给自己":需要邻居的 session 来构造 RolePush。
func (hub *worldSceneConnectionHub) neighborsInMap(mapID int, exceptRoleID string) []worldSceneConnection {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	neighbors := make([]worldSceneConnection, 0, len(hub.connections))
	for roleID, conn := range hub.connections {
		if roleID == exceptRoleID {
			continue
		}
		if conn.mapID != mapID {
			continue
		}
		if conn.session == nil || conn.session.selectedRole == nil || conn.session.playerBase == nil {
			continue
		}
		neighbors = append(neighbors, conn)
	}
	return neighbors
}

// broadcastCreateRoleToMap 把一条 createRole push 推给某 mapId 上除自己外所有在线邻居。
func (hub *worldSceneConnectionHub) broadcastCreateRoleToMap(mapID int, exceptRoleID string, push world.RolePush) {
	writeWorldSceneActions(hub.syncMapVisibility(mapID))
}

// refreshRoleActions 给“已经看见 actor”的同图观察者重推 createRole。
// 用于地图头顶 stateICO 等已在场玩家字段刷新；不新增/删除可见集合，
// 避免把仅状态变化做成 removeRole+createRole 闪烁。
// 原版对应 c_RoleInfo(50002) → roleInfo → U_Cell1.1.Refresh → changeStateICO；
// 本项目复用 createRole(1103) upsert 承载同一 RolePush.state 字段。
func (hub *worldSceneConnectionHub) refreshRoleActions(actorRoleID string) []worldScenePushAction {
	if actorRoleID == "" {
		return nil
	}
	hub.mu.Lock()
	defer hub.mu.Unlock()
	actor, ok := hub.connections[actorRoleID]
	if !ok || !worldSceneCanBuildRolePush(actor) {
		return nil
	}
	push := worldSceneBuildRolePush(actor)
	actions := make([]worldScenePushAction, 0)
	observerIDs := make([]string, 0, len(hub.connections))
	for observerID := range hub.connections {
		observerIDs = append(observerIDs, observerID)
	}
	sort.Strings(observerIDs)
	for _, observerID := range observerIDs {
		if observerID == actorRoleID {
			continue
		}
		observer := hub.connections[observerID]
		if observer.mapID != actor.mapID || observer.writer == nil || observer.visible == nil {
			continue
		}
		if _, visible := observer.visible[actorRoleID]; !visible {
			continue
		}
		rolePush := push
		actions = append(actions, worldScenePushAction{
			writer:      observer.writer,
			recipientID: observerID,
			createRole:  &rolePush,
		})
	}
	return actions
}

func broadcastWorldSceneRoleRefresh(roleID string) {
	if roleID == "" {
		return
	}
	writeWorldSceneActions(worldSceneHub.refreshRoleActions(roleID))
}

// classic map overhead stateICO units digit (source U_Cell1.1 changeStateICO):
// 0=none, 1=fight, 2=die, 4=look, 5=battle. Tens digit chair / higher VIP remain intact.
const classicMapRoleStateFightUnits = 1

func withClassicMapRoleStateUnits(state int, units int) int {
	if state < 0 {
		state = 0
	}
	if units < 0 {
		units = 0
	}
	units %= 10
	return state - state%10 + units
}

func applyClassicMapFightState(socketSession *packetSession, inFight bool) bool {
	if socketSession == nil || socketSession.playerBase == nil {
		return false
	}
	nextUnits := 0
	if inFight {
		nextUnits = classicMapRoleStateFightUnits
	}
	next := withClassicMapRoleStateUnits(socketSession.playerBase.State, nextUnits)
	if next == socketSession.playerBase.State {
		return false
	}
	socketSession.playerBase.State = next
	return true
}

// announceClassicMapFightState 设置/清除玩家地图头顶“战斗中”状态，并刷新已看见该玩家的邻居 createRole。
func announceClassicMapFightState(socketSession *packetSession, inFight bool) {
	if !applyClassicMapFightState(socketSession, inFight) {
		return
	}
	if socketSession == nil || socketSession.selectedRole == nil {
		return
	}
	broadcastWorldSceneRoleRefresh(socketSession.selectedRole.RoleID)
}

// broadcastRemoveRoleToMap 把一条 removeRole push 推给某 mapId 上除自己外所有在线邻居。
// payload 是裸 handle 字符串,与既有 removeRoleHandles 路径(main.go writePush)一致。
func (hub *worldSceneConnectionHub) broadcastRemoveRoleToMap(mapID int, exceptRoleID string, handle string) {
	writeWorldSceneActions(hub.removeRoleFromVisibility(mapID, handle))
}

// staticCreateRoleActions 构造给某 map 上所有在线玩家的静态明怪 createRole actions。
// 与玩家可见性集合无关：怪物 handle 不进入 player visible 集合。
func (hub *worldSceneConnectionHub) staticCreateRoleActions(mapID int, roles []world.RolePush) []worldScenePushAction {
	if hub == nil || len(roles) == 0 {
		return nil
	}
	hub.mu.Lock()
	defer hub.mu.Unlock()
	actions := make([]worldScenePushAction, 0, len(hub.connections)*len(roles))
	for roleID, conn := range hub.connections {
		if conn.mapID != mapID || conn.writer == nil {
			continue
		}
		for index := range roles {
			roleCopy := roles[index]
			rolePtr := new(world.RolePush)
			*rolePtr = roleCopy
			actions = append(actions, worldScenePushAction{
				writer:      conn.writer,
				recipientID: roleID,
				createRole:  rolePtr,
			})
		}
	}
	return actions
}

// broadcastStaticCreateRolesToMap 给某 map 上所有在线玩家推送静态明怪 createRole。
func (hub *worldSceneConnectionHub) broadcastStaticCreateRolesToMap(mapID int, roles []world.RolePush) {
	writeWorldSceneActions(hub.staticCreateRoleActions(mapID, roles))
}

// staticRemoveHandleActions 构造给某 map 上所有在线玩家的静态明怪 removeRole actions。
func (hub *worldSceneConnectionHub) staticRemoveHandleActions(mapID int, handles []string) []worldScenePushAction {
	if hub == nil || len(handles) == 0 {
		return nil
	}
	hub.mu.Lock()
	defer hub.mu.Unlock()
	actions := make([]worldScenePushAction, 0, len(hub.connections)*len(handles))
	for roleID, conn := range hub.connections {
		if conn.mapID != mapID || conn.writer == nil {
			continue
		}
		for _, handle := range handles {
			handle = strings.TrimSpace(handle)
			if handle == "" {
				continue
			}
			actions = append(actions, worldScenePushAction{
				writer:       conn.writer,
				recipientID:  roleID,
				removeHandle: handle,
			})
		}
	}
	return actions
}

// broadcastStaticRemoveHandlesToMap 给某 map 上所有在线玩家推送静态明怪 removeRole。
func (hub *worldSceneConnectionHub) broadcastStaticRemoveHandlesToMap(mapID int, handles []string) {
	writeWorldSceneActions(hub.staticRemoveHandleActions(mapID, handles))
}


// broadcastMoveRoleToMap 按原版 c_MoveRole(50012) 的 moveRole 语义,
// 把某玩家当前位置/目标点推给同 mapId 上除自己外的在线邻居。
func (hub *worldSceneConnectionHub) broadcastMoveRoleToMap(mapID int, exceptRoleID string, push world.RoleMovePush) {
	writeWorldSceneActions(hub.moveRoleActions(mapID, exceptRoleID, push))
}

// syncWorldScenePresence 在玩家选角成功 / 换角色后,把 scene hub 状态对齐到当前 session,
// 并完成"同 mapId 在线玩家互推":
//   - 换角色(oldRoleID != 当前 roleID 且 oldRoleID 非空):先 unregister 旧角色,
//     给旧角色原 mapId 邻居推 removeRole。
//   - 把当前角色 register 到当前 mapId。
//   - 把已在同 mapId 的邻居 createRole 推给当前玩家(让自己看到别人)。
//   - 把当前玩家的 createRole 推给同 mapId 所有邻居(让别人看到自己)。
//
// 幂等门禁以 hub 实际状态为准,不信任调用方传入的 oldMapID(可能 stale):
// 队伍同步传送(classic_team_transport.syncTransfer)会帮队员调 announceWorldSceneTransfer
// 直接把 hub 里的队员迁到新图,但队员自己那条 WebSocket 循环里的 registeredSceneMapID 仍是旧值;
// 若用 stale oldMapID 判断 mapChanged 会误迁,把刚迁好的新图注册 unregister 再重建,
// 新图邻居视角会出现 removeRole+createRole 闪烁。
// 因此先查 hub:若 currentRoleID 已注册在 currentMapID,无论 oldMapID 说什么都早退。
// 同角色同地图的普通请求(NPC 交互/背包/移动/心跳)也走这条早退路径。
//
// 调用方:main.go register 区。返回 (newRoleID, newMapID) 供 main.go 更新追踪变量。
func syncWorldScenePresence(writer *websocketWriter, socketSession *packetSession, oldRoleID string, oldMapID int) (string, int) {
	if writer == nil || socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		return oldRoleID, oldMapID
	}
	currentRoleID := socketSession.selectedRole.RoleID
	if currentRoleID == "" {
		return oldRoleID, oldMapID
	}
	currentMapID := socketSession.playerBase.MapID
	roleChanged := oldRoleID != "" && oldRoleID != currentRoleID

	// 幂等门禁(以 hub 实际状态为准,不信任 stale oldMapID):
	// 若 currentRoleID 已在 hub 注册且 mapID == currentMapID,说明 scene 状态已正确
	// (可能是首次注册后、普通请求重复进入、或 syncTransfer 已帮队员迁好图),直接早退。
	// 但换角色时必须先清理 oldRoleID,不能被 currentRoleID 的既有注册吞掉。
	if !roleChanged {
		if existing, ok := worldSceneHub.connectionFor(currentRoleID); ok && existing.mapID == currentMapID {
			return currentRoleID, currentMapID
		}
	}

	// 换角色或换图:先清理旧注册,给旧 mapId 邻居推 removeRole。
	// - 换角色:旧 roleID 整个注销(它不再在线)。
	// - 同角色换图:注销当前 roleID 的旧 mapId 绑定(下面重新 register 到新 mapId)。
	clearRoleID := currentRoleID
	if roleChanged && oldRoleID != "" {
		clearRoleID = oldRoleID
	}
	if clearRoleID != "" {
		if clearedMapID, ok := worldSceneHub.unregister(clearRoleID); ok {
			writeWorldSceneActions(worldSceneHub.removeRoleFromVisibility(clearedMapID, clearRoleID))
		}
	}

	spawn := world.DefaultSpawnForMap(currentMapID)
	worldSceneHub.register(currentRoleID, currentMapID, writer, socketSession, spawn)

	writeWorldSceneActions(worldSceneHub.syncMapVisibility(currentMapID))

	return currentRoleID, currentMapID
}

// announceWorldSceneTransfer 在玩家传送/切图成功后,把 scene 状态从旧 mapId 迁到新 mapId:
//   - 给旧 mapId 邻居推 removeRole(离开旧图)。
//   - register 到新 mapId 并完成新图互推(同 syncWorldScenePresence 的互推部分)。
//
// 与 syncWorldScenePresence 的区别:这里 roleID 没变,只是 mapId 变了,所以不走"换角色"分支。
// 调用方:main.go 收到 townBootstrap(传送/切图)后调用;classic_team_transport.syncTransfer 给队员调用。
// 返回值:新 mapId(供调用方更新追踪变量)。
func announceWorldSceneTransfer(writer *websocketWriter, socketSession *packetSession, currentRoleID string, oldMapID int, spawn world.SpawnPoint) int {
	if writer == nil || socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil || currentRoleID == "" {
		return oldMapID
	}
	newMapID := socketSession.playerBase.MapID

	// 离开旧图:先从 hub 注销(清掉旧 mapID 绑定),给旧图邻居推 removeRole。
	if oldMapID > 0 && oldMapID != newMapID {
		worldSceneHub.unregister(currentRoleID)
		writeWorldSceneActions(worldSceneHub.removeRoleFromVisibility(oldMapID, currentRoleID))
	}

	if spawn.X == 0 && spawn.Y == 0 {
		spawn = world.DefaultSpawnForMap(newMapID)
	}
	worldSceneHub.register(currentRoleID, newMapID, writer, socketSession, spawn)

	writeWorldSceneActions(worldSceneHub.syncMapVisibility(newMapID))

	return newMapID
}
