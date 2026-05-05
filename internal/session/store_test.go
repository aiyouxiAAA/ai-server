package session

import "testing"

func TestStoreLoginAccountSuccess(t *testing.T) {
	store := NewStore()

	response := store.Login(LoginRequest{
		UserName: "mockuser",
		Password: "magicpwd",
	})

	if !response.Success {
		t.Fatalf("expected login success, got failure: %+v", response)
	}
	if response.PlayerID != "mock-player-001" {
		t.Fatalf("expected player id mock-player-001, got %q", response.PlayerID)
	}
	if response.SessionToken != "mock-session-token-001" {
		t.Fatalf("expected session token mock-session-token-001, got %q", response.SessionToken)
	}
	if response.DisplayName != "Mock Swordswoman" {
		t.Fatalf("expected display name Mock Swordswoman, got %q", response.DisplayName)
	}
}

func TestStoreLoginAccountInvalidUserName(t *testing.T) {
	store := NewStore()

	response := store.Login(LoginRequest{
		UserName: "ab",
		Password: "magicpwd",
	})

	if response.Success {
		t.Fatalf("expected invalid username failure, got success: %+v", response)
	}
	if response.ErrorCode != "8" {
		t.Fatalf("expected error code 8, got %q", response.ErrorCode)
	}
	if response.ErrorMessage != "用户名不合法。" {
		t.Fatalf("expected invalid username message, got %q", response.ErrorMessage)
	}
}

func TestStoreLoginAccountNotFound(t *testing.T) {
	store := NewStore()

	response := store.Login(LoginRequest{
		UserName: "unknownuser",
		Password: "magicpwd",
	})

	if !response.Success {
		t.Fatalf("expected auto register login success, got failure: %+v", response)
	}
	if response.PlayerID != "acct-unknownuser" {
		t.Fatalf("expected player id acct-unknownuser, got %q", response.PlayerID)
	}
	if response.DisplayName != "unknownuser" {
		t.Fatalf("expected display name unknownuser, got %q", response.DisplayName)
	}
}

func TestStoreLoginAutoRegisteredAccountCanLoginAgain(t *testing.T) {
	store := NewStore()

	firstLogin := store.Login(LoginRequest{
		UserName: "autouser",
		Password: "magicpwd",
	})
	secondLogin := store.Login(LoginRequest{
		UserName: "autouser",
		Password: "magicpwd",
	})

	if !firstLogin.Success || !secondLogin.Success {
		t.Fatalf("expected repeated login success, got first=%+v second=%+v", firstLogin, secondLogin)
	}
	if firstLogin.PlayerID != secondLogin.PlayerID {
		t.Fatalf("expected stable player id, got %q and %q", firstLogin.PlayerID, secondLogin.PlayerID)
	}
}

func TestStoreLoginAccountWrongPassword(t *testing.T) {
	store := NewStore()

	response := store.Login(LoginRequest{
		UserName: "mockuser",
		Password: "wrongpwd",
	})

	if response.Success {
		t.Fatalf("expected wrong password failure, got success: %+v", response)
	}
	if response.ErrorCode != "3" {
		t.Fatalf("expected error code 3, got %q", response.ErrorCode)
	}
	if response.ErrorMessage != "密码错误!" {
		t.Fatalf("expected wrong password message, got %q", response.ErrorMessage)
	}
}

func TestStoreLoginPlatformFallback(t *testing.T) {
	store := NewStore()

	response := store.Login(LoginRequest{
		Platform: "guest",
	})

	if !response.Success {
		t.Fatalf("expected guest login success, got failure: %+v", response)
	}
	if response.PlayerID != "guest-player-local" {
		t.Fatalf("expected guest player id, got %q", response.PlayerID)
	}
	if response.SessionToken != "local-session-guest-player-local" {
		t.Fatalf("expected guest session token, got %q", response.SessionToken)
	}
}

func TestStoreRoleLifecycle(t *testing.T) {
	store := NewStore()
	playerID := "mock-player-001"

	listBefore := store.ListRoles(playerID)
	if len(listBefore.Roles) != 0 {
		t.Fatalf("expected no roles before create, got %d", len(listBefore.Roles))
	}

	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       playerID,
		DisplayName:    "测试女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	if createResponse.Role.RoleID == "" {
		t.Fatal("expected created role id to be non-empty")
	}
	if createResponse.Role.DisplayName != "测试女侠" {
		t.Fatalf("expected created role name 测试女侠, got %q", createResponse.Role.DisplayName)
	}

	listAfter := store.ListRoles(playerID)
	if len(listAfter.Roles) != 1 {
		t.Fatalf("expected one role after create, got %d", len(listAfter.Roles))
	}
	if listAfter.Roles[0].RoleID != createResponse.Role.RoleID {
		t.Fatalf("expected listed role id %q, got %q", createResponse.Role.RoleID, listAfter.Roles[0].RoleID)
	}

	selectResponse, ok := store.SelectRole(playerID, createResponse.Role.RoleID)
	if !ok {
		t.Fatal("expected select role success")
	}
	if selectResponse.Role.RoleID != createResponse.Role.RoleID {
		t.Fatalf("expected selected role id %q, got %q", createResponse.Role.RoleID, selectResponse.Role.RoleID)
	}
	if selectResponse.PlayerBase.PlayerID != playerID {
		t.Fatalf("expected player base player id %q, got %q", playerID, selectResponse.PlayerBase.PlayerID)
	}
	if selectResponse.PlayerBase.RoleID != createResponse.Role.RoleID {
		t.Fatalf("expected player base role id %q, got %q", createResponse.Role.RoleID, selectResponse.PlayerBase.RoleID)
	}
}

func TestStoreRemoveRoleSuccess(t *testing.T) {
	store := NewStore()
	playerID := "mock-player-001"

	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       playerID,
		DisplayName:    "待删除女侠",
		Gender:         "female",
		RoleTemplateID: 1,
		PresetID:       4,
		SourceQuery:    "hair=4",
		Appearance: RoleAppearance{
			"body": map[string]any{
				"hair": "4",
			},
		},
	})

	removeResponse := store.RemoveRole(RoleRemoveRequest{
		PlayerID: playerID,
		RoleID:   createResponse.Role.RoleID,
		Password: "magicpwd",
	})

	if !removeResponse.Success {
		t.Fatalf("expected remove role success, got failure: %+v", removeResponse)
	}
	if removeResponse.RemovedRoleID != createResponse.Role.RoleID {
		t.Fatalf("expected removed role id %q, got %q", createResponse.Role.RoleID, removeResponse.RemovedRoleID)
	}
	if removeResponse.Message != "删除成功！" {
		t.Fatalf("expected remove role success message 删除成功！, got %q", removeResponse.Message)
	}

	listResponse := store.ListRoles(playerID)
	if len(listResponse.Roles) != 0 {
		t.Fatalf("expected no roles after delete, got %d", len(listResponse.Roles))
	}
}

func TestStoreRemoveRoleWrongPassword(t *testing.T) {
	store := NewStore()
	playerID := "mock-player-001"

	createResponse := store.CreateRole(RoleCreateRequest{
		PlayerID:       playerID,
		DisplayName:    "保留女侠",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	removeResponse := store.RemoveRole(RoleRemoveRequest{
		PlayerID: playerID,
		RoleID:   createResponse.Role.RoleID,
		Password: "wrongpwd",
	})

	if removeResponse.Success {
		t.Fatalf("expected remove role failure, got success: %+v", removeResponse)
	}
	if removeResponse.ErrorCode != "3" {
		t.Fatalf("expected wrong password error code 3, got %q", removeResponse.ErrorCode)
	}

	listResponse := store.ListRoles(playerID)
	if len(listResponse.Roles) != 1 {
		t.Fatalf("expected role to remain after failed delete, got %d", len(listResponse.Roles))
	}
}

func TestStoreCreateRoleIDDoesNotReuseAfterDelete(t *testing.T) {
	store := NewStore()
	playerID := "mock-player-001"

	first := store.CreateRole(RoleCreateRequest{
		PlayerID:       playerID,
		DisplayName:    "甲",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	second := store.CreateRole(RoleCreateRequest{
		PlayerID:       playerID,
		DisplayName:    "乙",
		Gender:         "female",
		RoleTemplateID: 1,
	})
	third := store.CreateRole(RoleCreateRequest{
		PlayerID:       playerID,
		DisplayName:    "丙",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	if first.Role.RoleID == second.Role.RoleID || second.Role.RoleID == third.Role.RoleID {
		t.Fatalf("expected created roles to have unique ids, got %q %q %q", first.Role.RoleID, second.Role.RoleID, third.Role.RoleID)
	}

	removeResponse := store.RemoveRole(RoleRemoveRequest{
		PlayerID: playerID,
		RoleID:   second.Role.RoleID,
		Password: "magicpwd",
	})
	if !removeResponse.Success {
		t.Fatalf("expected remove role success, got failure: %+v", removeResponse)
	}
	if removeResponse.Message != "删除成功！" {
		t.Fatalf("expected remove role success message 删除成功！, got %q", removeResponse.Message)
	}

	fourth := store.CreateRole(RoleCreateRequest{
		PlayerID:       playerID,
		DisplayName:    "丁",
		Gender:         "female",
		RoleTemplateID: 1,
	})

	if fourth.Role.RoleID == first.Role.RoleID ||
		fourth.Role.RoleID == second.Role.RoleID ||
		fourth.Role.RoleID == third.Role.RoleID {
		t.Fatalf("expected newly created role id to stay unique after delete, got reused id %q", fourth.Role.RoleID)
	}

	listResponse := store.ListRoles(playerID)
	if len(listResponse.Roles) != 3 {
		t.Fatalf("expected three roles after delete and recreate, got %d", len(listResponse.Roles))
	}
}

func TestStoreListRolesNormalizesDuplicatedRoleIDs(t *testing.T) {
	store := NewStore()
	playerID := "mock-player-001"

	store.rolesByPID[playerID] = []RoleSummary{
		{
			RoleID:       "mock-player-001-role-001",
			DisplayName:  "甲",
			Level:        1,
			MapID:        1,
			VisualRoleID: 1,
		},
		{
			RoleID:       "mock-player-001-role-001",
			DisplayName:  "乙",
			Level:        1,
			MapID:        1,
			VisualRoleID: 1,
		},
	}

	listResponse := store.ListRoles(playerID)
	if len(listResponse.Roles) != 2 {
		t.Fatalf("expected two roles after normalization, got %d", len(listResponse.Roles))
	}
	if listResponse.Roles[0].RoleID == listResponse.Roles[1].RoleID {
		t.Fatalf("expected duplicate role ids to be repaired, got %q and %q", listResponse.Roles[0].RoleID, listResponse.Roles[1].RoleID)
	}
}

func TestStoreRemoveRoleRemovesAllDuplicatedMatches(t *testing.T) {
	store := NewStore()
	playerID := "mock-player-001"

	store.rolesByPID[playerID] = []RoleSummary{
		{
			RoleID:       "mock-player-001-role-001",
			DisplayName:  "甲",
			Level:        1,
			MapID:        1,
			VisualRoleID: 1,
		},
		{
			RoleID:       "mock-player-001-role-001",
			DisplayName:  "乙",
			Level:        1,
			MapID:        1,
			VisualRoleID: 1,
		},
	}

	removeResponse := store.RemoveRole(RoleRemoveRequest{
		PlayerID: playerID,
		RoleID:   "mock-player-001-role-001",
		Password: "magicpwd",
	})

	if !removeResponse.Success {
		t.Fatalf("expected remove role success, got failure: %+v", removeResponse)
	}

	listResponse := store.ListRoles(playerID)
	if len(listResponse.Roles) != 0 {
		t.Fatalf("expected duplicated roles to be fully removed, got %d roles", len(listResponse.Roles))
	}
}
