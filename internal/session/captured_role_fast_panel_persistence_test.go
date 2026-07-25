package session

import (
	"path/filepath"
	"testing"
)

func TestCapturedRoleFastPanelChangesSurviveRestart(t *testing.T) {
	testCases := []struct {
		name        string
		userName    string
		displayName string
		removeSlots []int
	}{
		{name: "222", userName: "22222222", displayName: "222"},
		{name: "333", userName: "33333333", displayName: "333"},
		{name: "444", userName: "44444444", displayName: "444"},
		{name: "555", userName: "55555555", displayName: "555"},
		{name: "666", userName: "66666666", displayName: "666"},
		{name: "777", userName: "77777777", displayName: "777", removeSlots: []int{0, 1}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			dbPath := filepath.Join(t.TempDir(), "ai-server.db")
			store, err := NewPersistentStore(dbPath)
			if err != nil {
				t.Fatalf("create store: %v", err)
			}
			login := store.Login(LoginRequest{UserName: testCase.userName, Password: "local-test-only"})
			created := store.CreateRole(RoleCreateRequest{
				PlayerID:       login.PlayerID,
				SessionToken:   login.SessionToken,
				DisplayName:    testCase.displayName,
				Gender:         "male",
				RoleTemplateID: 1,
			})
			if !created.Success {
				t.Fatalf("create role: %+v", created)
			}

			if len(testCase.removeSlots) > 0 {
				for _, slotIndex := range testCase.removeSlots {
					if _, ok := store.RemoveRoleFastPanelEntry(login.PlayerID, created.Role.RoleID, slotIndex); !ok {
						t.Fatalf("remove fast-panel entry at slot %d", slotIndex)
					}
				}
			} else {
				entries, ok := store.SetRoleFastPanelEntry(login.PlayerID, created.Role.RoleID, RoleFastPanelEntry{
					Index: 7,
					Type:  "skill",
					Name:  "普通攻击",
				})
				if !ok || !hasRoleFastPanelEntry(entries, 7, "普通攻击") {
					t.Fatalf("set fast-panel entry: ok=%v entries=%+v", ok, entries)
				}
			}
			if err := store.Close(); err != nil {
				t.Fatalf("close store: %v", err)
			}

			reopened, err := NewPersistentStore(dbPath)
			if err != nil {
				t.Fatalf("reopen store: %v", err)
			}
			defer reopened.Close()
			entries, ok := reopened.GetRoleFastPanel(login.PlayerID, created.Role.RoleID)
			if !ok {
				t.Fatal("read fast-panel entries after restart")
			}
			if len(testCase.removeSlots) > 0 {
				for _, slotIndex := range testCase.removeSlots {
					if hasRoleFastPanelIndex(entries, slotIndex) {
						t.Fatalf("removed entry at slot %d returned after restart: %+v", slotIndex, entries)
					}
				}
				return
			}
			if !hasRoleFastPanelEntry(entries, 7, "普通攻击") {
				t.Fatalf("custom entry reset after restart: %+v", entries)
			}
		})
	}
}

func hasRoleFastPanelEntry(entries []RoleFastPanelEntry, index int, name string) bool {
	for _, entry := range entries {
		if entry.Index == index && entry.Name == name {
			return true
		}
	}
	return false
}

func hasRoleFastPanelIndex(entries []RoleFastPanelEntry, index int) bool {
	for _, entry := range entries {
		if entry.Index == index {
			return true
		}
	}
	return false
}
