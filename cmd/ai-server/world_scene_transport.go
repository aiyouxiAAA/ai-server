package main

import (
	"log"
	"sync"

	"ai-server/internal/world"
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
}

func newWorldSceneConnectionHub() *worldSceneConnectionHub {
	return &worldSceneConnectionHub{
		connections: make(map[string]worldSceneConnection),
	}
}

func (hub *worldSceneConnectionHub) register(roleID string, mapID int, writer *websocketWriter, socketSession *packetSession) {
	if roleID == "" || writer == nil {
		return
	}
	hub.mu.Lock()
	defer hub.mu.Unlock()
	hub.connections[roleID] = worldSceneConnection{
		writer:  writer,
		session: socketSession,
		mapID:   mapID,
	}
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
	for _, roleID := range hub.roleIDsForMap(mapID, exceptRoleID) {
		writer := hub.writerFor(roleID)
		if writer == nil {
			continue
		}
		if err := writer.writePush(cmdClassicTownCreateRolePush, encodePayload(push)); err != nil {
			log.Printf("[ai-server] world scene broadcast createRole failed roleId=%s: %v", roleID, err)
		}
	}
}

// broadcastRemoveRoleToMap 把一条 removeRole push 推给某 mapId 上除自己外所有在线邻居。
// payload 是裸 handle 字符串,与既有 removeRoleHandles 路径(main.go writePush)一致。
func (hub *worldSceneConnectionHub) broadcastRemoveRoleToMap(mapID int, exceptRoleID string, handle string) {
	for _, roleID := range hub.roleIDsForMap(mapID, exceptRoleID) {
		writer := hub.writerFor(roleID)
		if writer == nil {
			continue
		}
		if err := writer.writePush(cmdClassicTownRemoveRolePush, encodePayload(handle)); err != nil {
			log.Printf("[ai-server] world scene broadcast removeRole failed roleId=%s: %v", roleID, err)
		}
	}
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
			worldSceneHub.broadcastRemoveRoleToMap(clearedMapID, clearRoleID, clearRoleID)
		}
	}

	spawn := world.DefaultSpawnForMap(currentMapID)
	worldSceneHub.register(currentRoleID, currentMapID, writer, socketSession)

	// 把已在同 mapId 的邻居 createRole 推给自己(让自己看到别人)。
	neighbors := worldSceneHub.neighborsInMap(currentMapID, currentRoleID)
	for _, neighbor := range neighbors {
		push := world.BuildPlayerRolePush(*neighbor.session.selectedRole, *neighbor.session.playerBase, world.DefaultSpawnForMap(neighbor.session.playerBase.MapID))
		if err := writer.writePush(cmdClassicTownCreateRolePush, encodePayload(push)); err != nil {
			log.Printf("[ai-server] world scene self receive neighbor createRole failed neighborRoleId=%s: %v", push.RoleID, err)
		}
	}

	// 把自己推给同 mapId 所有邻居。
	selfPush := world.BuildPlayerRolePush(*socketSession.selectedRole, *socketSession.playerBase, spawn)
	worldSceneHub.broadcastCreateRoleToMap(currentMapID, currentRoleID, selfPush)

	return currentRoleID, currentMapID
}

// announceWorldSceneTransfer 在玩家传送/切图成功后,把 scene 状态从旧 mapId 迁到新 mapId:
//   - 给旧 mapId 邻居推 removeRole(离开旧图)。
//   - register 到新 mapId 并完成新图互推(同 syncWorldScenePresence 的互推部分)。
//
// 与 syncWorldScenePresence 的区别:这里 roleID 没变,只是 mapId 变了,所以不走"换角色"分支。
// 调用方:main.go 收到 townBootstrap(传送/切图)后调用;classic_team_transport.syncTransfer 给队员调用。
// 返回值:新 mapId(供调用方更新追踪变量)。
func announceWorldSceneTransfer(writer *websocketWriter, socketSession *packetSession, currentRoleID string, oldMapID int) int {
	if writer == nil || socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil || currentRoleID == "" {
		return oldMapID
	}
	newMapID := socketSession.playerBase.MapID

	// 离开旧图:先从 hub 注销(清掉旧 mapID 绑定),给旧图邻居推 removeRole。
	if oldMapID > 0 && oldMapID != newMapID {
		worldSceneHub.unregister(currentRoleID)
		worldSceneHub.broadcastRemoveRoleToMap(oldMapID, currentRoleID, currentRoleID)
	}

	spawn := world.DefaultSpawnForMap(newMapID)
	worldSceneHub.register(currentRoleID, newMapID, writer, socketSession)

	neighbors := worldSceneHub.neighborsInMap(newMapID, currentRoleID)
	for _, neighbor := range neighbors {
		push := world.BuildPlayerRolePush(*neighbor.session.selectedRole, *neighbor.session.playerBase, world.DefaultSpawnForMap(neighbor.session.playerBase.MapID))
		if err := writer.writePush(cmdClassicTownCreateRolePush, encodePayload(push)); err != nil {
			log.Printf("[ai-server] world scene transfer self receive neighbor createRole failed neighborRoleId=%s: %v", push.RoleID, err)
		}
	}

	selfPush := world.BuildPlayerRolePush(*socketSession.selectedRole, *socketSession.playerBase, spawn)
	worldSceneHub.broadcastCreateRoleToMap(newMapID, currentRoleID, selfPush)

	return newMapID
}
