package world

import (
	"testing"

	"ai-server/internal/session"
)

func TestBuildTownBootstrapUsesPersistedRoleCoordinates(t *testing.T) {
	role := session.RoleSummary{
		RoleID:       "acct-666-role-1",
		DisplayName:  "666",
		Level:        50,
		MapID:        213,
		MapX:         384,
		MapY:         547,
		VisualRoleID: 1,
	}
	playerBase := session.PlayerBaseData{
		PlayerID:     "acct-666",
		RoleID:       role.RoleID,
		DisplayName:  role.DisplayName,
		Level:        role.Level,
		MapID:        role.MapID,
		VisualRoleID: role.VisualRoleID,
	}

	snapshot := BuildTownBootstrap(role, playerBase)
	if snapshot.LoadMap.MapID != "213" {
		t.Fatalf("expected captured map 213, got %+v", snapshot.LoadMap)
	}
	if snapshot.CreatePlayer.SpawnFlash != (SpawnPoint{X: 384, Y: 547}) {
		t.Fatalf("expected persisted spawn 384,547, got %+v", snapshot.CreatePlayer.SpawnFlash)
	}
}
