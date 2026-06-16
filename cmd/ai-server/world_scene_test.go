package main

import (
	"testing"

	"ai-server/internal/session"
	"ai-server/internal/team"
	"ai-server/internal/world"
)

// resetWorldSceneHub 重建一个干净的 scene hub,避免测试间互相污染。
func resetWorldSceneHub() *worldSceneConnectionHub {
	return newWorldSceneConnectionHub()
}

func resetClassicTeamManagerForWorldSceneTest(t *testing.T) {
	t.Helper()
	classicTeamManager.Reset()
	t.Cleanup(func() {
		classicTeamManager.Reset()
	})
}

func seedWorldSceneTeam(t *testing.T, leaderID string, memberID string, mapID string) {
	t.Helper()
	for _, roleID := range []string{leaderID, memberID} {
		classicTeamManager.UpsertOnline(team.Member{
			RoleID:   roleID,
			Name:     roleID,
			Level:    1,
			Vocation: "测试",
			HP:       100,
			MaxHP:    100,
			MP:       50,
			MaxMP:    50,
			MapID:    mapID,
			Online:   true,
		})
	}
	events := classicTeamManager.Invite(leaderID, memberID, "")
	inviteID := ""
	for _, event := range events {
		if event.Invite != nil {
			inviteID = event.Invite.InviteID
		}
	}
	if inviteID == "" {
		t.Fatalf("expected invite id when seeding team, got %+v", events)
	}
	classicTeamManager.ReplyInvite(memberID, inviteID, true)
}

// stubSceneSession 构造一个最小可用的 packetSession,带 selectedRole + playerBase,
// 用于 neighborsInMap 提取邻居信息构造 RolePush。
func stubSceneSession(roleID string, mapID int) *packetSession {
	return &packetSession{
		selectedRole: &session.RoleSummary{
			RoleID:      roleID,
			DisplayName: roleID,
			MapID:       mapID,
		},
		playerBase: &session.PlayerBaseData{
			PlayerID:    "player-" + roleID,
			RoleID:      roleID,
			DisplayName: roleID,
			MapID:       mapID,
			Level:       1,
		},
	}
}

func stubSceneSpawn(x int, y int) world.SpawnPoint {
	return world.SpawnPoint{X: x, Y: y}
}

// TestWorldSceneRoleIDsForMapFiltersByMapIDAndExcludesSelf 验证 hub 按 mapId 过滤在线玩家
// 的核心路由逻辑:同图返回、异图不返回、排除自己。
func TestWorldSceneRoleIDsForMapFiltersByMapIDAndExcludesSelf(t *testing.T) {
	hub := resetWorldSceneHub()
	hub.register("A", 1, &websocketWriter{}, stubSceneSession("A", 1), stubSceneSpawn(100, 200))
	hub.register("B", 1, &websocketWriter{}, stubSceneSession("B", 1), stubSceneSpawn(300, 400))
	hub.register("C", 2, &websocketWriter{}, stubSceneSession("C", 2), stubSceneSpawn(500, 600))

	// A 视角:同图(1)应返回 B,排除自己,排除异图 C。
	got := hub.roleIDsForMap(1, "A")
	if len(got) != 1 || got[0] != "B" {
		t.Fatalf("roleIDsForMap(1, A) = %v, want [B]", got)
	}

	// C 在 map2,视角应返回空(没有同图邻居)。
	got = hub.roleIDsForMap(2, "C")
	if len(got) != 0 {
		t.Fatalf("roleIDsForMap(2, C) = %v, want []", got)
	}

	// 不存在的 mapId 返回空。
	got = hub.roleIDsForMap(999, "A")
	if len(got) != 0 {
		t.Fatalf("roleIDsForMap(999, A) = %v, want []", got)
	}
}

// TestWorldSceneUnregisterReturnsOldMapID 验证注销时返回旧 mapId,
// 供调用方广播 removeRole 给原 mapId 邻居。
func TestWorldSceneUnregisterReturnsOldMapID(t *testing.T) {
	hub := resetWorldSceneHub()
	hub.register("A", 1, &websocketWriter{}, stubSceneSession("A", 1), stubSceneSpawn(100, 200))
	hub.register("B", 1, &websocketWriter{}, stubSceneSession("B", 1), stubSceneSpawn(300, 400))

	oldMapID, ok := hub.unregister("A")
	if !ok || oldMapID != 1 {
		t.Fatalf("unregister(A) = (%d, %v), want (1, true)", oldMapID, ok)
	}

	// A 注销后,B 视角下 map1 应无邻居。
	got := hub.roleIDsForMap(1, "B")
	if len(got) != 0 {
		t.Fatalf("after unregister A, roleIDsForMap(1, B) = %v, want []", got)
	}

	// 重复注销返回 false。
	_, ok = hub.unregister("A")
	if ok {
		t.Fatalf("unregister(A) twice should return ok=false")
	}
}

// TestWorldSceneNeighborsInMapReturnsSessionsForRolePush 验证 neighborsInMap
// 能拿到邻居的 session(含 selectedRole + playerBase),用于构造 RolePush。
func TestWorldSceneNeighborsInMapReturnsSessionsForRolePush(t *testing.T) {
	hub := resetWorldSceneHub()
	hub.register("A", 1, &websocketWriter{}, stubSceneSession("A", 1), stubSceneSpawn(100, 200))
	hub.register("B", 1, &websocketWriter{}, stubSceneSession("B", 1), stubSceneSpawn(300, 400))
	hub.register("C", 2, &websocketWriter{}, stubSceneSession("C", 2), stubSceneSpawn(500, 600))

	// A 进图时,应拿到同图(1)的 B,排除自己,排除异图 C。
	neighbors := hub.neighborsInMap(1, "A")
	if len(neighbors) != 1 {
		t.Fatalf("neighborsInMap(1, A) returned %d neighbors, want 1", len(neighbors))
	}
	neighbor := neighbors[0]
	if neighbor.session == nil || neighbor.session.selectedRole == nil || neighbor.session.playerBase == nil {
		t.Fatalf("neighbor session/selectedRole/playerBase must not be nil")
	}
	if neighbor.session.selectedRole.RoleID != "B" {
		t.Fatalf("neighbor roleID = %s, want B", neighbor.session.selectedRole.RoleID)
	}

	// 用邻居 session 构造 RolePush,验证字段能正确映射。
	push := world.BuildPlayerRolePush(*neighbor.session.selectedRole, *neighbor.session.playerBase, world.DefaultSpawnForMap(neighbor.session.playerBase.MapID))
	if push.Kind != "player" {
		t.Fatalf("BuildPlayerRolePush kind = %s, want player", push.Kind)
	}
	if push.RoleID != "B" || push.Handle != "B" {
		t.Fatalf("BuildPlayerRolePush RoleID/Handle = %s/%s, want B/B", push.RoleID, push.Handle)
	}
	if push.DisplayName != "B" {
		t.Fatalf("BuildPlayerRolePush DisplayName = %s, want B", push.DisplayName)
	}
}

// TestWorldSceneNeighborsInMapSkipsIncompleteSessions 验证 neighborsInMap
// 跳过 session 不完整的连接(防止构造 RolePush 时 nil 解引用)。
func TestWorldSceneNeighborsInMapSkipsIncompleteSessions(t *testing.T) {
	hub := resetWorldSceneHub()
	hub.register("A", 1, &websocketWriter{}, stubSceneSession("A", 1), stubSceneSpawn(100, 200))
	// B 的 session 缺 playerBase(模拟异常状态)。
	hub.register("B", 1, &websocketWriter{}, &packetSession{
		selectedRole: &session.RoleSummary{RoleID: "B", MapID: 1},
	}, stubSceneSpawn(300, 400))

	neighbors := hub.neighborsInMap(1, "A")
	// B 因 session 不完整应被跳过,只剩 0 个有效邻居。
	if len(neighbors) != 0 {
		t.Fatalf("neighborsInMap(1, A) = %d neighbors, want 0 (B has incomplete session)", len(neighbors))
	}
}

// TestWorldSceneBuildPlayerRolePushFields 验证 BuildPlayerRolePush 输出的 RolePush
// 字段与前端 ClassicTownRolePushMessage 对齐,且 Kind="player"。
func TestWorldSceneBuildPlayerRolePushFields(t *testing.T) {
	role := session.RoleSummary{
		RoleID:       "role-123",
		DisplayName:  "阿华田",
		Level:        18,
		Voc:          "游侠",
		MapID:        5,
		VisualRoleID: 7,
		PresetID:     2,
		SourceQuery:  "human/human.swf?a=5&b=7&",
	}
	playerBase := session.PlayerBaseData{
		PlayerID:     "player-123",
		RoleID:       "role-123",
		DisplayName:  "阿华田",
		Level:        18,
		Voc:          "游侠",
		MapID:        5,
		VisualRoleID: 7,
		PresetID:     2,
		SourceQuery:  "human/human.swf?a=5&b=7&",
	}
	spawn := world.SpawnPoint{X: 100, Y: 200}

	push := world.BuildPlayerRolePush(role, playerBase, spawn)

	if push.Kind != "player" {
		t.Errorf("Kind = %s, want player", push.Kind)
	}
	if push.Handle != "role-123" || push.RoleID != "role-123" {
		t.Errorf("Handle/RoleID = %s/%s, want role-123/role-123", push.Handle, push.RoleID)
	}
	if push.DisplayName != "阿华田" {
		t.Errorf("DisplayName = %s, want 阿华田", push.DisplayName)
	}
	if push.Level != 18 {
		t.Errorf("Level = %d, want 18", push.Level)
	}
	if push.Vocation != "游侠" {
		t.Errorf("Vocation = %s, want 游侠", push.Vocation)
	}
	if push.MapID != "5" {
		t.Errorf("MapID = %s, want 5", push.MapID)
	}
	if push.VisualRoleID != 7 {
		t.Errorf("VisualRoleID = %d, want 7", push.VisualRoleID)
	}
	if push.PresetID != 2 {
		t.Errorf("PresetID = %d, want 2", push.PresetID)
	}
	if push.SourceQuery != "human/human.swf?a=5&b=7&" {
		t.Errorf("SourceQuery = %s, want human/human.swf?a=5&b=7&", push.SourceQuery)
	}
	if push.SpawnFlash.X != 100 || push.SpawnFlash.Y != 200 {
		t.Errorf("SpawnFlash = %+v, want {100 200}", push.SpawnFlash)
	}
	// 默认零值断言:BuildPlayerRolePush 必须透传 PlayerBaseData 的 PK/State/GuildName/GuildPic,
	// 当前业务系统未填值,前端按字段渲染缺失优雅隐藏。详见 ClassicTownEntityPresenter.attachNameLabel。
	if push.PK != 0 {
		t.Errorf("PK = %d, want 0 (default)", push.PK)
	}
	if push.State != 0 {
		t.Errorf("State = %d, want 0 (default)", push.State)
	}
	if push.GuildName != "" {
		t.Errorf("GuildName = %s, want empty (default)", push.GuildName)
	}
	if push.GuildPic != "" {
		t.Errorf("GuildPic = %s, want empty (default)", push.GuildPic)
	}
}

// TestWorldSceneDefaultSpawnForMap 验证 DefaultSpawnForMap 对已知地图返回非零出生点,
// 未知地图返回零值(不 panic)。
func TestWorldSceneDefaultSpawnForMap(t *testing.T) {
	// mapId=1 是主城,有 DefaultSpawn{820, 451}(见 town_bootstrap.go:275)。
	spawn := world.DefaultSpawnForMap(1)
	if spawn.X == 0 && spawn.Y == 0 {
		t.Errorf("DefaultSpawnForMap(1) = %+v, want non-zero (main town has a spawn)", spawn)
	}

	// 未知地图返回零值,不 panic。
	unknown := world.DefaultSpawnForMap(99999)
	if unknown.X != 0 || unknown.Y != 0 {
		t.Errorf("DefaultSpawnForMap(99999) = %+v, want {0 0}", unknown)
	}
}

// swapWorldSceneHub 临时把全局 worldSceneHub 替换成干净实例,测试结束自动恢复。
// syncWorldScenePresence / announceWorldSceneTransfer 操作的是全局 hub,
// 必须替换全局才能隔离测试,避免污染其它测试或被其它测试污染。
func swapWorldSceneHub(t *testing.T) {
	t.Helper()
	original := worldSceneHub
	fresh := newWorldSceneConnectionHub()
	worldSceneHub = fresh
	t.Cleanup(func() { worldSceneHub = original })
}

// TestWorldSceneSyncIsIdempotentForSameRoleAndMap 验证幂等门禁:
// 同 roleID + 同 mapID 第二次调用 syncWorldScenePresence 时,函数早退,
// 不重新 register、不重复互推。这是防止"每个后续包重复互推 createRole"
// 的核心保障(问题1修复)。
func TestWorldSceneSyncIsIdempotentForSameRoleAndMap(t *testing.T) {
	swapWorldSceneHub(t)
	sess := stubSceneSession("A", 1)
	writer := &websocketWriter{}

	// 首次注册(无邻居,writePush 不会被调用)。
	roleID, mapID := syncWorldScenePresence(writer, sess, "", 0)
	if roleID != "A" || mapID != 1 {
		t.Fatalf("first sync returned (%s,%d), want (A,1)", roleID, mapID)
	}
	// 首次注册后 A 应在 hub 里。
	if _, ok := worldSceneHub.connectionFor("A"); !ok {
		t.Fatal("after first sync, A should be registered in hub")
	}

	// 第二次:同 roleID 同 mapID,应幂等早退。
	roleID2, mapID2 := syncWorldScenePresence(writer, sess, "A", 1)
	if roleID2 != "A" || mapID2 != 1 {
		t.Fatalf("idempotent sync returned (%s,%d), want (A,1)", roleID2, mapID2)
	}
	// hub 注册状态不变(还是 A 在 map1)。
	conn, ok := worldSceneHub.connectionFor("A")
	if !ok {
		t.Fatal("after idempotent sync, A should still be registered")
	}
	if conn.mapID != 1 {
		t.Errorf("after idempotent sync, A mapID = %d, want 1", conn.mapID)
	}
}

// TestWorldSceneSyncMigratesSameRoleToNewMap 验证同角色换图:
// A 已在 map1,session 改 map2 后调 sync(传 oldMapID=1),
// hub 里 A 的 mapID 应更新为 2,旧 map1 邻居(若有)会收到 removeRole。
// 本测试无邻居,避免触发 writePush 到 nil conn。
func TestWorldSceneSyncMigratesSameRoleToNewMap(t *testing.T) {
	swapWorldSceneHub(t)
	sess := stubSceneSession("A", 1)
	writer := &websocketWriter{}

	// 先注册到 map1。
	syncWorldScenePresence(writer, sess, "", 0)

	// 模拟切图:session 的 mapID 改成 2。
	sess.playerBase.MapID = 2
	sess.selectedRole.MapID = 2

	// 同角色换图,oldRoleID="A" oldMapID=1。
	roleID, mapID := syncWorldScenePresence(writer, sess, "A", 1)
	if roleID != "A" || mapID != 2 {
		t.Fatalf("map-change sync returned (%s,%d), want (A,2)", roleID, mapID)
	}
	conn, ok := worldSceneHub.connectionFor("A")
	if !ok {
		t.Fatal("after map change, A should still be registered")
	}
	if conn.mapID != 2 {
		t.Errorf("after map change, A mapID = %d, want 2", conn.mapID)
	}
	// 旧 map1 应已无 A。
	if got := worldSceneHub.roleIDsForMap(1, ""); len(got) != 0 {
		t.Errorf("after A moved to map2, roleIDsForMap(1) = %v, want []", got)
	}
	// 新 map2 应有 A。
	if got := worldSceneHub.roleIDsForMap(2, ""); len(got) != 1 || got[0] != "A" {
		t.Errorf("after A moved to map2, roleIDsForMap(2) = %v, want [A]", got)
	}
}

// TestWorldSceneSyncHandlesRoleChange 验证换角色:
// 旧 roleID="A" 已注册,session 切到 roleID="B",
// sync 后 A 从 hub 移除(给旧图邻居推 removeRole),B 注册到当前 mapId。
func TestWorldSceneSyncHandlesRoleChange(t *testing.T) {
	swapWorldSceneHub(t)
	writer := &websocketWriter{}

	// 先注册 A 到 map1。
	sessA := stubSceneSession("A", 1)
	syncWorldScenePresence(writer, sessA, "", 0)

	// 换角色:session 切到 B(同 map1,无邻居所以 removeRole 广播无 recipient,安全)。
	sessB := stubSceneSession("B", 1)
	roleID, mapID := syncWorldScenePresence(writer, sessB, "A", 1)
	if roleID != "B" || mapID != 1 {
		t.Fatalf("role-change sync returned (%s,%d), want (B,1)", roleID, mapID)
	}

	// A 应已从 hub 移除。
	if _, ok := worldSceneHub.connectionFor("A"); ok {
		t.Error("after role change, old role A should be unregistered")
	}
	// B 应在 hub。
	if _, ok := worldSceneHub.connectionFor("B"); !ok {
		t.Error("after role change, new role B should be registered")
	}
}

// TestWorldSceneSyncRoleChangeDoesNotEarlyReturnWhenTargetAlreadyRegistered 验证换角色时
// 即使目标 roleID 已在 hub 的当前 mapId,也不能被幂等早退吞掉,必须先清理旧 roleID。
func TestWorldSceneSyncRoleChangeDoesNotEarlyReturnWhenTargetAlreadyRegistered(t *testing.T) {
	swapWorldSceneHub(t)
	writer := &websocketWriter{}

	sessA := stubSceneSession("A", 2)
	syncWorldScenePresence(writer, sessA, "", 0)
	sessBExisting := stubSceneSession("B", 1)
	worldSceneHub.register("B", 1, writer, sessBExisting, stubSceneSpawn(300, 400))

	sessB := stubSceneSession("B", 1)
	roleID, mapID := syncWorldScenePresence(writer, sessB, "A", 2)
	if roleID != "B" || mapID != 1 {
		t.Fatalf("role-change to existing target returned (%s,%d), want (B,1)", roleID, mapID)
	}
	if _, ok := worldSceneHub.connectionFor("A"); ok {
		t.Error("after role change to existing B, old role A should be unregistered")
	}
	if conn, ok := worldSceneHub.connectionFor("B"); !ok || conn.mapID != 1 {
		t.Errorf("after role change to existing B, B hub mapID = %d (ok=%v), want 1", conn.mapID, ok)
	}
}

// TestWorldSceneAnnounceTransferUpdatesMapID 验证 announceWorldSceneTransfer
// 把角色从 oldMapID 迁到 session 当前 mapID,返回新 mapID。
func TestWorldSceneAnnounceTransferUpdatesMapID(t *testing.T) {
	swapWorldSceneHub(t)
	sess := stubSceneSession("A", 1)
	writer := &websocketWriter{}
	syncWorldScenePresence(writer, sess, "", 0)

	// 模拟切图到 map2。
	sess.playerBase.MapID = 2
	sess.selectedRole.MapID = 2

	newMapID := announceWorldSceneTransfer(writer, sess, "A", 1, stubSceneSpawn(333, 444))
	if newMapID != 2 {
		t.Fatalf("announceWorldSceneTransfer returned %d, want 2", newMapID)
	}
	conn, ok := worldSceneHub.connectionFor("A")
	if !ok || conn.mapID != 2 {
		t.Errorf("after transfer, A registered mapID = %d (ok=%v), want 2", conn.mapID, ok)
	}
	if conn.spawn.X != 333 || conn.spawn.Y != 444 {
		t.Errorf("after transfer, A spawn = %+v, want {333 444}", conn.spawn)
	}
}

// TestWorldSceneSyncEarlyReturnsWhenHubAlreadyAtCurrentMap 验证幂等门禁以 hub 实际状态为准,
// 不信任调用方传入的 stale oldMapID。这是防止"队伍同步切图后队员下一次请求重复 remove/create"
// 的核心保障:
// 场景——classic_team_transport.syncTransfer 已帮队员调 announceWorldSceneTransfer 把 hub 里的
// 队员迁到新图(当前 mapID=2),但队员自己那条 WebSocket 循环里的 registeredSceneMapID 仍是旧值(1)。
// 队员发下一个包时 main.go 用 stale oldMapID=1 调 syncWorldScenePresence,
// 若以 oldMapID 判断会误以为 mapChanged,把刚迁好的 map2 注册 unregister 再重建,
// 新图邻居视角出现 removeRole+createRole 闪烁。修复后以 hub 为准:hub 已在 map2,直接早退。
func TestWorldSceneSyncEarlyReturnsWhenHubAlreadyAtCurrentMap(t *testing.T) {
	swapWorldSceneHub(t)
	sess := stubSceneSession("A", 1)
	writer := &websocketWriter{}
	// A 首次注册到 map1。
	syncWorldScenePresence(writer, sess, "", 0)

	// 模拟 syncTransfer 帮队员迁图:session 改 map2,announceWorldSceneTransfer 把 hub 迁到 map2。
	sess.playerBase.MapID = 2
	sess.selectedRole.MapID = 2
	announceWorldSceneTransfer(writer, sess, "A", 1, stubSceneSpawn(333, 444))

	// 此时 hub 里 A 在 map2(正确)。但队员自己的 registeredSceneMapID 还是旧值 1(stale)。
	// 队员发下一包,main.go 用 stale oldMapID=1 调 sync——必须早退,不能误迁。
	roleID, mapID := syncWorldScenePresence(writer, sess, "A", 1)
	if roleID != "A" || mapID != 2 {
		t.Fatalf("stale-oldMapID sync returned (%s,%d), want (A,2) early-return", roleID, mapID)
	}
	// hub 状态应保持 map2,没有被 unregister+重建。
	conn, ok := worldSceneHub.connectionFor("A")
	if !ok || conn.mapID != 2 {
		t.Errorf("after stale-oldMapID sync, A hub mapID = %d (ok=%v), want 2 (no re-migration)", conn.mapID, ok)
	}
}

func TestWorldSceneUpdatePositionStoresLatestSpawn(t *testing.T) {
	hub := resetWorldSceneHub()
	hub.register("A", 1, &websocketWriter{}, stubSceneSession("A", 1), stubSceneSpawn(100, 200))

	if !hub.updatePosition("A", 1, stubSceneSpawn(777, 888)) {
		t.Fatal("updatePosition(A, map1) should return true")
	}

	conn, ok := hub.connectionFor("A")
	if !ok {
		t.Fatal("after updatePosition, A should still be registered")
	}
	if conn.spawn.X != 777 || conn.spawn.Y != 888 {
		t.Fatalf("after updatePosition, spawn = %+v, want {777 888}", conn.spawn)
	}
	if hub.updatePosition("A", 2, stubSceneSpawn(1, 2)) {
		t.Fatal("updatePosition with mismatched map should return false")
	}
	conn, _ = hub.connectionFor("A")
	if conn.spawn.X != 777 || conn.spawn.Y != 888 {
		t.Fatalf("mismatched map update changed spawn to %+v, want {777 888}", conn.spawn)
	}
}

func TestWorldSceneVisibilityUsesHeroSpace(t *testing.T) {
	resetClassicTeamManagerForWorldSceneTest(t)
	hub := resetWorldSceneHub()
	hub.register("A", 1, &websocketWriter{}, stubSceneSession("A", 1), stubSceneSpawn(0, 0))
	hub.register("B", 1, &websocketWriter{}, stubSceneSession("B", 1), stubSceneSpawn(449, 0))
	hub.register("C", 1, &websocketWriter{}, stubSceneSession("C", 1), stubSceneSpawn(450, 0))

	hub.syncMapVisibility(1)
	connA, ok := hub.connectionFor("A")
	if !ok {
		t.Fatal("A should remain registered")
	}
	if _, ok := connA.visible["B"]; !ok {
		t.Fatal("A should see B inside heroSpace")
	}
	if _, ok := connA.visible["C"]; ok {
		t.Fatal("A should not see C at heroSpace boundary")
	}
}

func TestWorldSceneVisibilityCapsScreenRoleToTwentyNearest(t *testing.T) {
	resetClassicTeamManagerForWorldSceneTest(t)
	hub := resetWorldSceneHub()
	hub.register("observer", 1, &websocketWriter{}, stubSceneSession("observer", 1), stubSceneSpawn(0, 0))
	for index := 1; index <= 22; index += 1 {
		roleID := "R" + string(rune('A'+index-1))
		hub.register(roleID, 1, &websocketWriter{}, stubSceneSession(roleID, 1), stubSceneSpawn(index*10, 0))
	}

	hub.syncMapVisibility(1)
	observer, ok := hub.connectionFor("observer")
	if !ok {
		t.Fatal("observer should remain registered")
	}
	if len(observer.visible) != worldSceneScreenRoleLimit {
		t.Fatalf("observer visible count = %d, want %d", len(observer.visible), worldSceneScreenRoleLimit)
	}
	if _, ok := observer.visible["RU"]; ok {
		t.Fatal("observer should not see the 21st nearest role")
	}
	if _, ok := observer.visible["RV"]; ok {
		t.Fatal("observer should not see the 22nd nearest role")
	}
}

func TestWorldSceneVisibilityKeepsSameMapTeammateOutsideHeroSpace(t *testing.T) {
	resetClassicTeamManagerForWorldSceneTest(t)
	seedWorldSceneTeam(t, "observer", "teammate", "1")

	hub := resetWorldSceneHub()
	hub.register("observer", 1, &websocketWriter{}, stubSceneSession("observer", 1), stubSceneSpawn(0, 0))
	hub.register("teammate", 1, &websocketWriter{}, stubSceneSession("teammate", 1), stubSceneSpawn(1000, 0))
	hub.register("stranger", 1, &websocketWriter{}, stubSceneSession("stranger", 1), stubSceneSpawn(1000, 0))

	hub.syncMapVisibility(1)
	observer, ok := hub.connectionFor("observer")
	if !ok {
		t.Fatal("observer should remain registered")
	}
	if _, ok := observer.visible["teammate"]; !ok {
		t.Fatal("observer should see same-map teammate outside heroSpace")
	}
	if _, ok := observer.visible["stranger"]; ok {
		t.Fatal("observer should not see stranger outside heroSpace")
	}
}

func TestWorldSceneVisibilityKeepsSameMapTeammateBeyondScreenRoleLimit(t *testing.T) {
	resetClassicTeamManagerForWorldSceneTest(t)
	seedWorldSceneTeam(t, "observer", "teammate", "1")

	hub := resetWorldSceneHub()
	hub.register("observer", 1, &websocketWriter{}, stubSceneSession("observer", 1), stubSceneSpawn(0, 0))
	for index := 1; index <= 22; index += 1 {
		roleID := "R" + string(rune('A'+index-1))
		hub.register(roleID, 1, &websocketWriter{}, stubSceneSession(roleID, 1), stubSceneSpawn(index*10, 0))
	}
	hub.register("teammate", 1, &websocketWriter{}, stubSceneSession("teammate", 1), stubSceneSpawn(430, 0))

	hub.syncMapVisibility(1)
	observer, ok := hub.connectionFor("observer")
	if !ok {
		t.Fatal("observer should remain registered")
	}
	if _, ok := observer.visible["teammate"]; !ok {
		t.Fatal("observer should keep same-map teammate beyond screenRole cap")
	}
	if len(observer.visible) != worldSceneScreenRoleLimit+1 {
		t.Fatalf("observer visible count = %d, want %d", len(observer.visible), worldSceneScreenRoleLimit+1)
	}
	if _, ok := observer.visible["RU"]; ok {
		t.Fatal("observer should still cap the 21st nearest non-team role")
	}
}

func TestWorldSceneMoveRoleRemovesWhenActorLeavesHeroSpace(t *testing.T) {
	resetClassicTeamManagerForWorldSceneTest(t)
	hub := resetWorldSceneHub()
	hub.register("A", 1, &websocketWriter{}, stubSceneSession("A", 1), stubSceneSpawn(0, 0))
	hub.register("B", 1, &websocketWriter{}, stubSceneSession("B", 1), stubSceneSpawn(100, 0))
	hub.syncMapVisibility(1)

	hub.updatePosition("B", 1, stubSceneSpawn(1000, 0))
	actions := hub.moveRoleActions(1, "B", world.RoleMovePush{Handle: "B", X: 1000, Y: 0, TX: 1000, TY: 0, MapID: "1"})
	sawRemove := false
	sawMove := false
	for _, action := range actions {
		if action.recipientID == "A" && action.removeHandle == "B" {
			sawRemove = true
		}
		if action.recipientID == "A" && action.moveRole != nil && action.moveRole.Handle == "B" {
			sawMove = true
		}
	}
	if !sawRemove {
		t.Fatal("A should receive removeRole for B after B leaves heroSpace")
	}
	if sawMove {
		t.Fatal("A should not receive moveRole for B after B leaves heroSpace")
	}
}

func TestBuildClassicTownMoveRoleResultUsesSessionRoleAndUpdatesHub(t *testing.T) {
	swapWorldSceneHub(t)
	sess := stubSceneSession("real-role", 45)
	worldSceneHub.register("real-role", 45, &websocketWriter{}, sess, stubSceneSpawn(100, 200))

	result := buildClassicTownMoveRoleResult(sess, classicTownMoveRoleRequest{
		Handle: "spoofed-role",
		Type:   "Run",
		X:      1231,
		Y:      490,
		Z:      0,
		TX:     1075,
		TY:     490,
		TZ:     0,
		MapID:  "45",
	})

	if !result.handled || result.moveRole == nil {
		t.Fatalf("moveRole request should be handled with push, got %+v", result)
	}
	if result.moveRole.Handle != "real-role" {
		t.Fatalf("moveRole handle = %s, want session role real-role", result.moveRole.Handle)
	}
	if result.moveRole.Type != "Run" || result.moveRole.X != 1231 || result.moveRole.Y != 490 || result.moveRole.TX != 1075 || result.moveRole.TY != 490 || result.moveRole.MapID != "45" {
		t.Fatalf("unexpected moveRole payload: %+v", result.moveRole)
	}
	conn, ok := worldSceneHub.connectionFor("real-role")
	if !ok {
		t.Fatal("real-role should remain registered after moveRole")
	}
	if conn.spawn.X != 1231 || conn.spawn.Y != 490 {
		t.Fatalf("hub spawn after moveRole = %+v, want {1231 490}", conn.spawn)
	}
}

func TestBuildClassicTownMoveRoleResultIgnoresMismatchedMap(t *testing.T) {
	swapWorldSceneHub(t)
	sess := stubSceneSession("A", 45)
	worldSceneHub.register("A", 45, &websocketWriter{}, sess, stubSceneSpawn(100, 200))

	result := buildClassicTownMoveRoleResult(sess, classicTownMoveRoleRequest{
		Type:  "Run",
		X:     999,
		Y:     999,
		TX:    999,
		TY:    999,
		MapID: "46",
	})

	if !result.handled {
		t.Fatal("mismatched map moveRole should still be handled")
	}
	if result.moveRole != nil {
		t.Fatalf("mismatched map moveRole should not broadcast, got %+v", result.moveRole)
	}
	conn, ok := worldSceneHub.connectionFor("A")
	if !ok {
		t.Fatal("A should remain registered after mismatched map moveRole")
	}
	if conn.spawn.X != 100 || conn.spawn.Y != 200 {
		t.Fatalf("mismatched map moveRole changed spawn to %+v, want {100 200}", conn.spawn)
	}
}
