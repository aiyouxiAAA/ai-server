package session

import (
	"ai-server/internal/profession"
	"testing"
)

func TestBladeDancerCreationAndRetiredVocations(t *testing.T) {
	store := NewStore()
	login := store.Login(LoginRequest{UserName: "bladetest", Password: "bladetest"})
	if !login.Success {
		t.Fatalf("fixture login failed: %+v", login)
	}
	created := store.CreateRole(RoleCreateRequest{PlayerID: login.PlayerID, SessionToken: login.SessionToken, DisplayName: "刃舞新角", Gender: "female", RoleTemplateID: 1})
	if !created.Success || created.Role.Voc != "刃舞者" || created.Role.MapID != profession.Definitions[0].SpawnMapID {
		t.Fatalf("new role did not use profession: %+v", created)
	}
	skills, _, ok := store.GetRoleSkills(login.PlayerID, created.Role.RoleID)
	panel, _ := store.GetRoleFastPanel(login.PlayerID, created.Role.RoleID)
	if !ok || len(skills) != 7 || len(panel) != 6 {
		t.Fatalf("defaults: %+v %+v", skills, panel)
	}
	for _, item := range panel {
		if item.Name == "格挡" {
			t.Fatal("passive in shortcut")
		}
	}
	if canSetRoleFastPanelEntry(RoleFastPanelEntry{Index: 7, Type: "skill", Name: "格挡"}, skills) {
		t.Fatal("passive can be manually assigned")
	}
	for _, old := range []string{"战士", "术士", "游侠", "断岳"} {
		if result := store.SetRoleVocation(login.PlayerID, created.Role.RoleID, old); result.ErrorCode != "invalid_vocation" {
			t.Fatalf("old vocation accepted: %s", old)
		}
	}
}
