package world

import (
	"ai-server/internal/session"
	"testing"
)

func TestSnowSpiderWildMapsBootstrap(t *testing.T) {
	names := map[int]string{6: "村外雪径", 8: "听雪坡", 108: "冻溪桥", 113: "冰镜湖", 2: "雪栈村"}
	for _, id := range []int{6, 8, 108, 113, 2} {
		role := session.RoleSummary{RoleID: "snow-spider-world", DisplayName: "测试", Level: 1, MapID: id}
		base := session.PlayerBaseData{RoleID: role.RoleID, Level: 1, MapID: id}
		snapshot := BuildTownBootstrap(role, base)
		if snapshot.LoadMap.EnemyShow != (id != 2) {
			t.Fatalf("map=%d enemyShow=%v", id, snapshot.LoadMap.EnemyShow)
		}
		if snapshot.LoadMap.MapName != names[id] {
			t.Fatalf("map=%d name=%s", id, snapshot.LoadMap.MapName)
		}
	}
}
