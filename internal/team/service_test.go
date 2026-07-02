package team

import "testing"

func TestTeamInviteAcceptCreatesTeam(t *testing.T) {
	manager := NewManager()
	manager.UpsertOnline(testMember("role-a", "甲"))
	manager.UpsertOnline(testMember("role-b", "乙"))

	inviteEvents := manager.Invite("role-a", "role-b", "")
	if len(inviteEvents) != 2 || inviteEvents[0].Invite == nil {
		t.Fatalf("expected invite and result events, got %+v", inviteEvents)
	}

	events := manager.ReplyInvite("role-b", inviteEvents[0].Invite.InviteID, true)
	snapshot := collectMembers(events)
	if len(snapshot) != 2 {
		t.Fatalf("expected two members after accept, got %+v", events)
	}
	if !snapshot["role-a"].IsLeader || snapshot["role-b"].IsLeader {
		t.Fatalf("expected inviter to be leader, got %+v", snapshot)
	}
}

func TestTeamInviteRejectUsesCapturedSourceMessage(t *testing.T) {
	manager := NewManager()
	manager.UpsertOnline(testMember("role-a", "甲"))
	manager.UpsertOnline(testMember("role-b", "乙"))

	inviteEvents := manager.Invite("role-a", "role-b", "")
	if len(inviteEvents) != 2 || inviteEvents[0].Invite == nil {
		t.Fatalf("expected invite and result events, got %+v", inviteEvents)
	}

	events := manager.ReplyInvite("role-b", inviteEvents[0].Invite.InviteID, false)
	if len(events) != 2 || events[1].Result == nil {
		t.Fatalf("expected reply and inviter rejection result, got %+v", events)
	}
	result := events[1].Result
	if result.Success || result.Action != "inviteRejected" || result.ErrorCode != "TARGET_REJECTED" || result.ErrorMessage != "对方拒绝加入你的队伍" {
		t.Fatalf("expected captured TeamDeny rejection message, got %+v", result)
	}
}

func TestTeamRejectsFifthMember(t *testing.T) {
	manager := NewManager()
	for _, member := range []Member{
		testMember("role-a", "甲"),
		testMember("role-b", "乙"),
		testMember("role-c", "丙"),
		testMember("role-d", "丁"),
		testMember("role-e", "戊"),
	} {
		manager.UpsertOnline(member)
	}
	acceptInvite(t, manager, "role-a", "role-b")
	acceptInvite(t, manager, "role-a", "role-c")
	acceptInvite(t, manager, "role-a", "role-d")

	events := manager.Invite("role-a", "role-e", "")
	if len(events) != 1 || events[0].Result == nil || events[0].Result.ErrorCode != "TEAM_FULL" {
		t.Fatalf("expected full team rejection, got %+v", events)
	}
}

func TestTeamLeaderCanKickAndTransferLeader(t *testing.T) {
	manager := NewManager()
	manager.UpsertOnline(testMember("role-a", "甲"))
	manager.UpsertOnline(testMember("role-b", "乙"))
	manager.UpsertOnline(testMember("role-c", "丙"))
	acceptInvite(t, manager, "role-a", "role-b")
	acceptInvite(t, manager, "role-a", "role-c")

	transferEvents := manager.TransferLeader("role-a", "role-b", "")
	transferSnapshot := collectMembers(transferEvents)
	if !transferSnapshot["role-b"].IsLeader || transferSnapshot["role-a"].IsLeader {
		t.Fatalf("expected role-b to become leader, got %+v", transferSnapshot)
	}

	kickEvents := manager.Kick("role-b", "role-c", "")
	if !hasClearFor(kickEvents, "role-c") {
		t.Fatalf("expected kicked member clear event, got %+v", kickEvents)
	}
	afterKick := collectMembers(kickEvents)
	if _, ok := afterKick["role-c"]; ok {
		t.Fatalf("expected role-c removed from member snapshot, got %+v", afterKick)
	}
}

func TestTeamDisbandsWhenOnlyOneMemberWouldRemain(t *testing.T) {
	manager := NewManager()
	manager.UpsertOnline(testMember("role-a", "甲"))
	manager.UpsertOnline(testMember("role-b", "乙"))
	acceptInvite(t, manager, "role-a", "role-b")

	events := manager.Leave("role-b")
	if !hasClearFor(events, "role-b") || !hasClearFor(events, "role-a") {
		t.Fatalf("expected both two-person team members to receive clear events, got %+v", events)
	}
	if len(collectMembers(events)) != 0 {
		t.Fatalf("expected disband to avoid one-person team snapshots, got %+v", events)
	}
	if recipients, ok := manager.RecipientsForTeam("role-a"); ok || len(recipients) != 0 {
		t.Fatalf("expected remaining member to have no team after disband, got ok=%v recipients=%+v", ok, recipients)
	}
	if plan := manager.BuildBattleMemberPlan("role-a", "1"); len(plan.Members) != 0 {
		t.Fatalf("expected disbanded role to have no battle team members, got %+v", plan)
	}
}

func TestTeamKickDisbandsTwoPersonTeam(t *testing.T) {
	manager := NewManager()
	manager.UpsertOnline(testMember("role-a", "甲"))
	manager.UpsertOnline(testMember("role-b", "乙"))
	acceptInvite(t, manager, "role-a", "role-b")

	events := manager.Kick("role-a", "role-b", "")
	if !hasClearFor(events, "role-a") || !hasClearFor(events, "role-b") {
		t.Fatalf("expected leader and kicked member to receive clear events, got %+v", events)
	}
	if len(collectMembers(events)) != 0 {
		t.Fatalf("expected kick disband to avoid one-person team snapshots, got %+v", events)
	}
	if recipients, ok := manager.RecipientsForTeam("role-a"); ok || len(recipients) != 0 {
		t.Fatalf("expected leader to have no team after two-person kick, got ok=%v recipients=%+v", ok, recipients)
	}
}

func TestTeamUpsertOnlineRefreshesMemberSnapshotFields(t *testing.T) {
	manager := NewManager()
	manager.UpsertOnline(testMember("role-a", "甲"))
	manager.UpsertOnline(testMember("role-b", "乙"))
	acceptInvite(t, manager, "role-a", "role-b")

	events := manager.UpsertOnline(Member{
		RoleID:   "role-b",
		Name:     "乙",
		Level:    21,
		Vocation: "游侠",
		HP:       42,
		MaxHP:    120,
		MP:       7,
		MaxMP:    80,
		MapID:    "2",
	})

	snapshot := collectMembers(events)
	refreshed := snapshot["role-b"]
	if refreshed.Level != 21 || refreshed.Vocation != "游侠" || refreshed.HP != 42 || refreshed.MaxHP != 120 || refreshed.MP != 7 || refreshed.MaxMP != 80 || refreshed.MapID != "2" {
		t.Fatalf("expected refreshed role-b snapshot fields, got %+v", refreshed)
	}
	if !refreshed.Online || refreshed.IsLeader {
		t.Fatalf("expected refreshed member to stay online and non-leader, got %+v", refreshed)
	}
	if leader := snapshot["role-a"]; !leader.IsLeader || !leader.Online || leader.MapID != "1" {
		t.Fatalf("expected leader snapshot to stay intact, got %+v", leader)
	}
}

func TestTeamNonLeaderCannotKick(t *testing.T) {
	manager := NewManager()
	manager.UpsertOnline(testMember("role-a", "甲"))
	manager.UpsertOnline(testMember("role-b", "乙"))
	manager.UpsertOnline(testMember("role-c", "丙"))
	acceptInvite(t, manager, "role-a", "role-b")
	acceptInvite(t, manager, "role-a", "role-c")

	events := manager.Kick("role-b", "role-c", "")
	if len(events) != 1 || events[0].Result == nil || events[0].Result.ErrorCode != "NOT_LEADER" {
		t.Fatalf("expected non-leader kick rejection, got %+v", events)
	}
}

func TestTeamLeavePromotesNextLeader(t *testing.T) {
	manager := NewManager()
	manager.UpsertOnline(testMember("role-a", "甲"))
	manager.UpsertOnline(testMember("role-b", "乙"))
	manager.UpsertOnline(testMember("role-c", "丙"))
	acceptInvite(t, manager, "role-a", "role-b")
	acceptInvite(t, manager, "role-a", "role-c")

	events := manager.Leave("role-a")
	if !hasClearFor(events, "role-a") {
		t.Fatalf("expected leaving leader clear event, got %+v", events)
	}
	snapshot := collectMembers(events)
	if !snapshot["role-b"].IsLeader {
		t.Fatalf("expected next member to be promoted, got %+v", snapshot)
	}
}

func TestTeamBuildSyncTransferPlanRequiresLeaderSameMapAndOnline(t *testing.T) {
	manager := NewManager()
	manager.UpsertOnline(testMember("role-a", "甲"))
	manager.UpsertOnline(testMember("role-b", "乙"))
	manager.UpsertOnline(testMember("role-c", "丙"))
	acceptInvite(t, manager, "role-a", "role-b")
	acceptInvite(t, manager, "role-a", "role-c")

	manager.SetSyncChangeMap("role-a", true)
	plan := manager.BuildSyncTransferPlan("role-a", "1")
	if !plan.Enabled || plan.ErrorCode != "" || len(plan.Members) != 2 {
		t.Fatalf("expected same-map sync plan for two members, got %+v", plan)
	}

	manager.UpsertOnline(Member{
		RoleID:   "role-c",
		Name:     "丙",
		Level:    20,
		Vocation: "战士",
		HP:       100,
		MaxHP:    100,
		MP:       50,
		MaxMP:    50,
		MapID:    "2",
	})
	plan = manager.BuildSyncTransferPlan("role-a", "1")
	if !plan.Enabled || plan.ErrorCode != "MEMBER_MAP_MISMATCH" || len(plan.Members) != 0 {
		t.Fatalf("expected map mismatch to cancel sync, got %+v", plan)
	}

	manager.SetSyncChangeMap("role-a", false)
	plan = manager.BuildSyncTransferPlan("role-b", "1")
	if plan.Enabled {
		t.Fatalf("expected non-leader transfer to stay normal when sync disabled, got %+v", plan)
	}
}

func TestTeamBuildDungeonResetPlanRequiresLeader(t *testing.T) {
	manager := NewManager()
	manager.UpsertOnline(testMember("role-a", "甲"))
	manager.UpsertOnline(testMember("role-b", "乙"))
	acceptInvite(t, manager, "role-a", "role-b")

	leaderPlan := manager.BuildDungeonResetPlan("role-a")
	if !leaderPlan.Allowed || len(leaderPlan.Members) != 1 || leaderPlan.Members[0].RoleID != "role-b" {
		t.Fatalf("expected leader reset plan to include online member, got %+v", leaderPlan)
	}

	memberPlan := manager.BuildDungeonResetPlan("role-b")
	if memberPlan.Allowed || memberPlan.ErrorCode != "NOT_LEADER" {
		t.Fatalf("expected non-leader reset to be rejected, got %+v", memberPlan)
	}
}

func TestTeamBuildBattleMemberPlanIncludesOnlyOnlineSameMap(t *testing.T) {
	manager := NewManager()
	manager.UpsertOnline(testMember("role-a", "甲"))
	manager.UpsertOnline(testMember("role-b", "乙"))
	manager.UpsertOnline(testMember("role-c", "丙"))
	acceptInvite(t, manager, "role-a", "role-b")
	acceptInvite(t, manager, "role-a", "role-c")

	manager.UpsertOnline(Member{
		RoleID:   "role-c",
		Name:     "丙",
		Level:    20,
		Vocation: "战士",
		HP:       100,
		MaxHP:    100,
		MP:       50,
		MaxMP:    50,
		MapID:    "2",
	})
	plan := manager.BuildBattleMemberPlan("role-a", "1")
	if !plan.Allowed || len(plan.Members) != 1 || plan.Members[0].RoleID != "role-b" {
		t.Fatalf("expected battle plan to include only same-map online member, got %+v", plan)
	}

	manager.SetOffline("role-b")
	plan = manager.BuildBattleMemberPlan("role-a", "1")
	if !plan.Allowed || len(plan.Members) != 0 {
		t.Fatalf("expected offline and different-map members to be excluded, got %+v", plan)
	}
}

func acceptInvite(t *testing.T, manager *Manager, fromRoleID string, targetRoleID string) {
	t.Helper()
	inviteEvents := manager.Invite(fromRoleID, targetRoleID, "")
	if len(inviteEvents) == 0 || inviteEvents[0].Invite == nil {
		t.Fatalf("expected invite from %s to %s, got %+v", fromRoleID, targetRoleID, inviteEvents)
	}
	events := manager.ReplyInvite(targetRoleID, inviteEvents[0].Invite.InviteID, true)
	if len(collectMembers(events)) == 0 {
		t.Fatalf("expected team snapshot after accept, got %+v", events)
	}
}

func testMember(roleID string, name string) Member {
	return Member{
		RoleID:   roleID,
		Name:     name,
		Level:    20,
		Vocation: "战士",
		HP:       100,
		MaxHP:    100,
		MP:       50,
		MaxMP:    50,
		MapID:    "1",
	}
}

func collectMembers(events []Event) map[string]Member {
	members := make(map[string]Member)
	for _, event := range events {
		for _, member := range event.Members {
			members[member.RoleID] = member
		}
	}
	return members
}

func hasClearFor(events []Event, roleID string) bool {
	for _, event := range events {
		if event.Clear == nil {
			continue
		}
		for _, recipient := range event.Recipients {
			if recipient == roleID {
				return true
			}
		}
	}
	return false
}
