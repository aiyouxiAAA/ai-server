package session

import (
	"path/filepath"
	"testing"
)

func TestFreshStartRequiresRealAccountAndStartsEmpty(t *testing.T) {
	store, err := NewPersistentStore(filepath.Join(t.TempDir(), "fresh.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	for _, table := range []string{"accounts", "roles", "role_accepted_quests", "role_removed_quests", "role_quest_progress", "role_quest_objectives", "role_item_acquisitions"} {
		var count int
		if err := store.db.QueryRow("SELECT count(*) FROM " + table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("%s count=%d err=%v", table, count, err)
		}
	}
	if store.Login(LoginRequest{Platform: "guest"}).Success {
		t.Fatal("anonymous mock login accepted")
	}
	if store.CreateRole(RoleCreateRequest{PlayerID: "invented", SessionToken: "local-session-invented"}).Success {
		t.Fatal("invented local token accepted")
	}
	login := mustLogin(t, store, "realnewbie", "realpassword")
	created := store.CreateRole(RoleCreateRequest{PlayerID: login.PlayerID, SessionToken: login.SessionToken, DisplayName: "正式新人", RoleTemplateID: 1})
	if !created.Success || created.Role.Level != 1 || created.Role.Exp != 0 || len(created.Role.Items) != 0 || len(created.Role.Currencies) != 0 {
		t.Fatalf("new role contains development state: %+v", created)
	}
	if created.Role.Voc != "刃舞者" || created.Role.MapID != 2 || len(created.Role.Skills) != 7 {
		t.Fatalf("formal profession configuration missing: %+v", created.Role)
	}
	if result := store.TransactAuthoredQuest(login.PlayerID, created.Role.RoleID, "XZ-M002", false); result.Changed {
		t.Fatal("new role skipped first quest")
	}
}

func TestFreshStartRestartNeverRefillsNamedCharacters(t *testing.T) {
	path := filepath.Join(t.TempDir(), "restart.db")
	store, err := NewPersistentStore(path)
	if err != nil {
		t.Fatal(err)
	}
	type identity struct{ player, role string }
	identities := []identity{}
	for _, name := range []string{"77777777", "55555555", "1150045313"} {
		login := mustLogin(t, store, name, name)
		created := store.CreateRole(RoleCreateRequest{PlayerID: login.PlayerID, SessionToken: login.SessionToken, DisplayName: "111", RoleTemplateID: 1})
		if !created.Success {
			t.Fatal(created)
		}
		// A persisted player choice must survive startup, including an empty shortcut bar.
		if _, err := store.db.Exec(`UPDATE roles SET skills_json='[]',fast_panel_json='[]' WHERE role_id=?`, created.Role.RoleID); err != nil {
			t.Fatal(err)
		}
		identities = append(identities, identity{login.PlayerID, created.Role.RoleID})
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	for pass := 0; pass < 3; pass++ {
		store, err = NewPersistentStore(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, id := range identities {
			role, _, ok := store.GetRoleRuntimeData(id.player, id.role)
			if !ok || role.Level != 1 || role.Exp != 0 || len(role.Items) != 0 || len(role.Currencies) != 0 || len(role.Skills) != 0 || len(role.FastPanel) != 0 {
				t.Fatalf("restart %d injected %s: %+v", pass, id.role, role)
			}
		}
		var count int
		if err := store.db.QueryRow(`SELECT count(*) FROM role_snapshot_migrations`).Scan(&count); err != nil || count != 0 {
			t.Fatalf("automatic migration count=%d err=%v", count, err)
		}
		if err := store.Close(); err != nil {
			t.Fatal(err)
		}
	}
}
