package battle

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"ai-server/internal/classicactivity"
	"ai-server/internal/classicdata"
	"ai-server/internal/session"
)

func pendingStartActor(runtime *Runtime) *StartCommandPush {
	if runtime == nil || len(runtime.PendingStarts) == 0 {
		return nil
	}
	start := runtime.PendingStarts[0]
	return &start
}

func pendingStartFor(runtime *Runtime, handle string) *StartCommandPush {
	if runtime == nil {
		return nil
	}
	for index := range runtime.PendingStarts {
		if runtime.PendingStarts[index].ActorHandle == handle {
			start := runtime.PendingStarts[index]
			return &start
		}
	}
	return nil
}

func useSourceEncounterRoll(roll func(int) int) func() {
	previous := sourceEncounterRoll
	sourceEncounterRoll = roll
	return func() {
		sourceEncounterRoll = previous
	}
}

func useSourceBattleAttackRoll(roll func(int) int) func() {
	previous := sourceBattleAttackRoll
	sourceBattleAttackRoll = roll
	return func() {
		sourceBattleAttackRoll = previous
	}
}

func useSourceBattleHealRoll(roll func(int) int) func() {
	previous := sourceBattleHealRoll
	sourceBattleHealRoll = roll
	return func() {
		sourceBattleHealRoll = previous
	}
}

func requireSourceBattleRewardDropRate(t *testing.T, rates []sourceBattleRewardDropRate, itemName string) sourceBattleRewardDropRate {
	t.Helper()
	for _, rate := range rates {
		if rate.ItemName == itemName {
			return rate
		}
	}
	t.Fatalf("expected drop rate for %s in %+v", itemName, rates)
	return sourceBattleRewardDropRate{}
}

func containsSourceBattleRewardItem(items []string, expected string) bool {
	for _, item := range items {
		if item == expected {
			return true
		}
	}
	return false
}

func TestBuildOverUsesCapturedMap5BattleRewards(t *testing.T) {
	defer useSourceEncounterRoll(func(maxExclusive int) int { return 0 })()
	runtime := &Runtime{
		BattleID: "battle-map5",
		MapID:    "5",
		Round:    2,
	}

	over := runtime.buildOver(CampTeam)
	if over == nil {
		t.Fatal("expected OverBattle push")
	}
	if over.Result.ExpDelta != 0 {
		t.Fatalf("expected captured map5 exp 0, got %d", over.Result.ExpDelta)
	}
	expectedItems := []string{"肉x1", "兽牙x1"}
	if !reflect.DeepEqual(over.Result.Items, expectedItems) {
		t.Fatalf("expected captured map5 low-roll reward %+v, got %+v", expectedItems, over.Result.Items)
	}
}

func TestRerollBattleRewardItemsUsesSameBattleSource(t *testing.T) {
	rolls := []int{0, 99}
	rollIndex := 0
	defer useSourceEncounterRoll(func(maxExclusive int) int {
		value := rolls[rollIndex]
		if rollIndex < len(rolls)-1 {
			rollIndex++
		}
		if value >= maxExclusive {
			return maxExclusive - 1
		}
		return value
	})()

	runtime := &Runtime{
		BattleID: "battle-map5",
		MapID:    "5",
		Round:    2,
	}

	leaderResult := runtime.buildOver(CampTeam).Result
	if !reflect.DeepEqual(leaderResult.Items, []string{"肉x1", "兽牙x1"}) {
		t.Fatalf("expected first participant low-roll rewards, got %+v", leaderResult.Items)
	}

	memberResult := runtime.RerollBattleRewardItems(leaderResult)
	if !reflect.DeepEqual(memberResult.Items, []string{"肉x1"}) {
		t.Fatalf("expected second participant reroll to keep guaranteed item only, got %+v", memberResult.Items)
	}
	if memberResult.ExpDelta != leaderResult.ExpDelta || memberResult.Winner != leaderResult.Winner || memberResult.Rounds != leaderResult.Rounds {
		t.Fatalf("expected reroll to preserve non-item result fields, got leader=%+v member=%+v", leaderResult, memberResult)
	}
}

func TestRerollBattleRewardItemsKeepsManualResultWithoutSourceReward(t *testing.T) {
	runtime := &Runtime{
		BattleID: "battle-map4",
		MapID:    "4",
		Round:    1,
	}
	result := ResultPayload{
		Winner:   CampTeam,
		Rounds:   1,
		ExpDelta: 37,
		Items:    []string{"朽木x1"},
	}

	reroll := runtime.RerollBattleRewardItems(result)
	if !reflect.DeepEqual(reroll.Items, result.Items) {
		t.Fatalf("expected manual result without source reward config to stay unchanged, got %+v", reroll.Items)
	}
}

func TestBuildOverDoesNotInventUncapturedOrEscapedRewards(t *testing.T) {
	map4 := (&Runtime{BattleID: "battle-map4", MapID: "4", Round: 1}).buildOver(CampTeam)
	if map4.Result.ExpDelta != 0 || len(map4.Result.Items) != 0 {
		t.Fatalf("expected uncaptured map4 reward to stay empty, got %+v", map4.Result)
	}

	escaped := (&Runtime{BattleID: "battle-escaped", MapID: "5", Round: 1}).buildOver(CampEnemy, true)
	if escaped.Result.ExpDelta != 0 || len(escaped.Result.Items) != 0 {
		t.Fatalf("expected escaped battle reward to stay empty, got %+v", escaped.Result)
	}
}

func TestTeamWildBattleAdvancesToNextPlayerActor(t *testing.T) {
	defer useSourceEncounterRoll(func(maxExclusive int) int { return 0 })()

	leader := TeamActor{
		Role: session.RoleSummary{
			RoleID:      "role_leader",
			DisplayName: "队长",
			Level:       12,
			Skills: []session.RoleSkill{
				{Name: "贯甲连矢", Level: 5, Type: "oneE", Description: "f_s_贯甲连矢&2@28&4@提升25%的物理伤害&0;增加30%（无视防御）的物理攻击力"},
			},
		},
		PlayerBase: session.PlayerBaseData{
			PlayerID:    "player_leader",
			DisplayName: "队长",
			Level:       12,
			MapID:       4,
			HP:          300,
			MaxHP:       300,
			MP:          80,
			MaxMP:       80,
		},
	}
	member := TeamActor{
		Role: session.RoleSummary{
			RoleID:      "role_member",
			DisplayName: "队员",
			Level:       12,
			Skills: []session.RoleSkill{
				{Name: "盘龙棍法", Level: 2, Type: "all", Description: "f_s_盘龙棍法&2@14&4@对所有敌人造成82%的物理伤害"},
			},
		},
		PlayerBase: session.PlayerBaseData{
			PlayerID:    "player_member",
			DisplayName: "队员",
			Level:       12,
			MapID:       4,
			HP:          280,
			MaxHP:       280,
			MP:          70,
			MaxMP:       70,
		},
	}
	runtime, bundle, ok := NewTeamWildBattle([]TeamActor{leader, member}, StartRequest{
		MapID:       "4",
		MapName:     "云隐村口",
		StageFocusX: 120,
	})
	if !ok {
		t.Fatal("expected team battle to start")
	}
	leaderCommand, leaderCommandOK := bundle.StartCommandForActor(leader.Role.RoleID)
	memberCommand, memberCommandOK := bundle.StartCommandForActor(member.Role.RoleID)
	if len(runtime.livingCells(CampTeam)) != 2 || !leaderCommandOK || !memberCommandOK {
		t.Fatalf("expected a command window for each living team member, got cells=%+v starts=%+v", runtime.Cells, bundle.TeamStartCommands)
	}
	if leaderCommand.Round != memberCommand.Round || leaderCommand.Sequence == memberCommand.Sequence {
		t.Fatalf("expected concurrent same-round commands with distinct sequences, got leader=%+v member=%+v", leaderCommand, memberCommand)
	}
	if !commandDefinitionsContain(leaderCommand.Commands, CommandGuanJiaLianShi) || commandDefinitionsContain(leaderCommand.Commands, CommandPanLongGunFa) {
		t.Fatalf("expected leader command window to use leader commands, got %+v", leaderCommand.Commands)
	}
	if !commandDefinitionsContain(memberCommand.Commands, CommandPanLongGunFa) || commandDefinitionsContain(memberCommand.Commands, CommandGuanJiaLianShi) {
		t.Fatalf("expected member command window to use member commands, got %+v", memberCommand.Commands)
	}
	target := runtime.firstLiving(CampEnemy)
	if target == nil {
		t.Fatal("expected enemy target")
	}

	memberResult := runtime.ProcessAction(ActionRequest{
		BattleID:     bundle.Start.BattleID,
		ActorHandle:  member.Role.RoleID,
		CommandID:    CommandNormalAttack,
		TargetHandle: target.Handle,
		Round:        memberCommand.Round,
		Sequence:     memberCommand.Sequence,
	})
	if memberResult.ErrorCode != "" || len(memberResult.Actions) == 0 {
		t.Fatalf("expected first received member action to resolve immediately, got %+v", memberResult)
	}
	if runtime.Phase != PhaseCommand || !runtime.HasPendingTeamAction(leader.Role.RoleID) || len(runtime.PendingStarts) != 0 {
		t.Fatalf("expected leader command window to remain open after member action, got phase=%s pending=%+v starts=%+v", runtime.Phase, runtime.PendingTeamActions, runtime.PendingStarts)
	}
	if replay := runtime.ProcessPlayOver(PlayOverRequest{BattleID: bundle.Start.BattleID}); replay.ErrorCode != "battle_play_over_empty" {
		t.Fatalf("expected first action playback acknowledgement not to advance the remaining team command, got %+v", replay)
	}

	leaderResult := runtime.ProcessAction(ActionRequest{
		BattleID:     bundle.Start.BattleID,
		ActorHandle:  leader.Role.RoleID,
		CommandID:    CommandNormalAttack,
		TargetHandle: target.Handle,
		Round:        leaderCommand.Round,
		Sequence:     leaderCommand.Sequence,
	})
	if leaderResult.ErrorCode != "" || len(leaderResult.Actions) == 0 {
		t.Fatalf("expected remaining leader command to stay valid after member action, got %+v", leaderResult)
	}
	if len(runtime.PendingTeamActions) != 2 || len(runtime.PendingStarts) != 2 {
		t.Fatalf("expected enemy turn only after the last team command and then a fresh pair of windows, got pending=%+v starts=%+v", runtime.PendingTeamActions, runtime.PendingStarts)
	}
	playOver := runtime.ProcessPlayOver(PlayOverRequest{BattleID: bundle.Start.BattleID})
	if playOver.ErrorCode != "" || len(playOver.StartCommands) != 2 || playOver.StartCommand != nil {
		t.Fatalf("expected playback completion to open both next-round command windows, got %+v", playOver)
	}
	nextLeader := StartCommandPush{}
	nextMember := StartCommandPush{}
	for _, command := range playOver.StartCommands {
		if command.ActorHandle == leader.Role.RoleID {
			nextLeader = command
		}
		if command.ActorHandle == member.Role.RoleID {
			nextMember = command
		}
	}
	if nextLeader.Round != leaderCommand.Round+1 || nextMember.Round != leaderCommand.Round+1 || nextLeader.Sequence == nextMember.Sequence {
		t.Fatalf("expected distinct next-round command windows, got leader=%+v member=%+v", nextLeader, nextMember)
	}

	finalRuntime, finalBundle, ok := NewTeamWildBattle([]TeamActor{leader, member}, StartRequest{
		MapID:       "4",
		MapName:     "云隐村口",
		StageFocusX: 120,
	})
	if !ok {
		t.Fatal("expected final-target team battle to start")
	}
	finalTarget := finalRuntime.firstLiving(CampEnemy)
	if finalTarget == nil {
		t.Fatal("expected final target")
	}
	finalTarget.HP = 1
	finalTarget.Dog = 0
	finalResult := finalRuntime.ProcessAction(ActionRequest{
		BattleID:     finalBundle.Start.BattleID,
		ActorHandle:  leader.Role.RoleID,
		CommandID:    CommandNormalAttack,
		TargetHandle: finalTarget.Handle,
		Round:        finalBundle.StartCommand.Round,
		Sequence:     finalBundle.StartCommand.Sequence,
	})
	if finalResult.ErrorCode != "" {
		t.Fatalf("expected final target action to succeed, got %s", finalResult.ErrorCode)
	}
	if len(finalRuntime.PendingStarts) != 0 || finalRuntime.PendingOver == nil {
		t.Fatalf("expected final target kill to finish before the remaining team command, got starts=%+v over=%+v", finalRuntime.PendingStarts, finalRuntime.PendingOver)
	}
	finalPlayOver := finalRuntime.ProcessPlayOver(PlayOverRequest{BattleID: finalBundle.Start.BattleID})
	if finalPlayOver.Over == nil || finalPlayOver.StartCommand != nil {
		t.Fatalf("expected final target kill to return OverBattle instead of a member command, got %+v", finalPlayOver)
	}
}

func TestTeamWildBattleCreatesFourMemberCommandWindows(t *testing.T) {
	defer useSourceEncounterRoll(func(maxExclusive int) int { return 0 })()

	actors := make([]TeamActor, 0, 4)
	for index := 1; index <= 4; index++ {
		roleID := fmt.Sprintf("role-%d", index)
		actors = append(actors, TeamActor{
			Role: session.RoleSummary{
				RoleID:      roleID,
				DisplayName: fmt.Sprintf("队员%d", index),
				Level:       12,
			},
			PlayerBase: session.PlayerBaseData{
				PlayerID:    fmt.Sprintf("player-%d", index),
				DisplayName: fmt.Sprintf("队员%d", index),
				Level:       12,
				MapID:       4,
				HP:          300,
				MaxHP:       300,
				MP:          80,
				MaxMP:       80,
			},
		})
	}

	runtime, bundle, ok := NewTeamWildBattle(actors, StartRequest{
		MapID:       "4",
		MapName:     "云隐村口",
		StageFocusX: 120,
	})
	if !ok {
		t.Fatal("expected four-member team battle to start")
	}
	if len(runtime.livingCells(CampTeam)) != 4 || len(bundle.TeamStartCommands) != 4 {
		t.Fatalf("expected four team cells and command windows, got cells=%+v commands=%+v", runtime.Cells, bundle.TeamStartCommands)
	}

	sequences := map[int]bool{}
	for _, actor := range actors {
		command, commandOK := bundle.StartCommandForActor(actor.Role.RoleID)
		if !commandOK || command.ActorHandle != actor.Role.RoleID || sequences[command.Sequence] {
			t.Fatalf("expected unique command window for %s, got %+v", actor.Role.RoleID, command)
		}
		sequences[command.Sequence] = true
	}
	for index := range runtime.Cells {
		if runtime.Cells[index].Camp == CampEnemy {
			runtime.Cells[index].HP = 100000
			runtime.Cells[index].MaxHP = 100000
		}
	}
	target := runtime.firstLiving(CampEnemy)
	if target == nil {
		t.Fatal("expected enemy target for four-member team battle")
	}

	for _, actor := range actors {
		command, _ := bundle.StartCommandForActor(actor.Role.RoleID)
		result := runtime.ProcessAction(ActionRequest{
			BattleID:     bundle.Start.BattleID,
			ActorHandle:  actor.Role.RoleID,
			CommandID:    CommandNormalAttack,
			TargetHandle: target.Handle,
			Round:        command.Round,
			Sequence:     command.Sequence,
		})
		if result.ErrorCode != "" {
			t.Fatalf("expected %s normal attack to resolve, got %+v", actor.Role.RoleID, result)
		}
	}
	if len(runtime.PendingTeamActions) != 4 || len(runtime.PendingStarts) != 4 {
		t.Fatalf("expected the next round to prepare four command windows, got pending=%+v starts=%+v", runtime.PendingTeamActions, runtime.PendingStarts)
	}
	playOver := runtime.ProcessPlayOver(PlayOverRequest{BattleID: bundle.Start.BattleID})
	if playOver.ErrorCode != "" || len(playOver.StartCommands) != 4 {
		t.Fatalf("expected next round to reopen four command windows, got %+v", playOver)
	}
}

func commandDefinitionsContain(commands []CommandDefinition, commandID string) bool {
	for _, command := range commands {
		if command.ID == commandID {
			return true
		}
	}
	return false
}

func TestTeamWildBattleDeadMemberHasNoCommandWindow(t *testing.T) {
	runtime := &Runtime{
		BattleID:             "battle-team-dead-member",
		Phase:                PhaseCommand,
		Round:                3,
		Cells:                []CellInfoPush{{Handle: "leader", Camp: CampTeam, HP: 100, MaxHP: 100}, {Handle: "member", Camp: CampTeam, HP: 0, MaxHP: 100}},
		PendingTeamActions:   map[string]bool{"leader": true, "member": true},
		PendingTeamSequences: map[string]int{"leader": 7, "member": 8},
	}

	if _, ok := runtime.CommandWindowForActor("member"); ok {
		t.Fatal("expected HP <= 0 teammate to have no command window")
	}
	runtime.prunePendingTeamActions()
	if runtime.HasPendingTeamAction("member") || len(runtime.PendingTeamActions) != 1 || !runtime.PendingTeamActions["leader"] {
		t.Fatalf("expected dead member to be removed from the pending team phase, got %+v", runtime.PendingTeamActions)
	}
}

func TestBuildOverUsesCapturedMap49RobberRewardWithExperience(t *testing.T) {
	defer useSourceEncounterRoll(func(maxExclusive int) int { return 0 })()
	over := (&Runtime{BattleID: "battle-map49", MapID: "49", Round: 1}).buildOver(CampTeam)
	if over == nil {
		t.Fatal("expected OverBattle push")
	}
	if over.Result.ExpDelta != 610 {
		t.Fatalf("expected captured map49 robber reward exp 610, got %d", over.Result.ExpDelta)
	}
	expectedItems := []string{"铜钱x8", "盗贼的首级x1", "L小喇叭x1", "红方巾x1", "毛皮x1"}
	if !reflect.DeepEqual(over.Result.Items, expectedItems) {
		t.Fatalf("expected captured map49 low-roll reward %+v, got %+v", expectedItems, over.Result.Items)
	}
}

func TestBuildOverPrefersCapturedMap49RobberHandleReward(t *testing.T) {
	defer useSourceEncounterRoll(func(maxExclusive int) int { return 0 })()
	over := (&Runtime{
		BattleID: "battle-map49-robber-452",
		MapID:    "49",
		Round:    1,
		Cells: []CellInfoPush{
			{
				Handle: "3784555592010429",
				Camp:   CampEnemy,
				Name:   "盗贼",
				MaxHP:  452,
				Level:  16,
			},
		},
	}).buildOver(CampTeam)
	if over == nil {
		t.Fatal("expected OverBattle push")
	}
	if over.Result.ExpDelta != 610 {
		t.Fatalf("expected captured map49 452HP robber reward exp 610, got %d", over.Result.ExpDelta)
	}
	expectedItems := []string{"L小喇叭x1", "铜钱x6", "剔骨刀x1", "红方巾x1", "盗贼的首级x1", "毛皮x1", "盗贼布衣x1", "盗贼护臂x1", "盗贼的鞋x1"}
	if !reflect.DeepEqual(over.Result.Items, expectedItems) {
		t.Fatalf("expected captured map49 452HP robber low-roll reward %+v, got %+v", expectedItems, over.Result.Items)
	}
}

func TestBuildOverUsesTwentyPercentFallbackForUnstattedRobberEquipment(t *testing.T) {
	defer useSourceEncounterRoll(func(maxExclusive int) int {
		switch maxExclusive {
		case 5:
			return 0
		case 3:
			return 2
		default:
			return maxExclusive - 1
		}
	})()
	over := (&Runtime{
		BattleID: "battle-map49-robber-452-equipment-fallback",
		MapID:    "49",
		Round:    1,
		Cells: []CellInfoPush{
			{
				Handle: "3784555592010429",
				Camp:   CampEnemy,
				Name:   "盗贼",
				MaxHP:  452,
				Level:  16,
			},
		},
	}).buildOver(CampTeam)
	if over == nil {
		t.Fatal("expected OverBattle push")
	}
	expectedItems := []string{"盗贼腰带x1"}
	if !reflect.DeepEqual(over.Result.Items, expectedItems) {
		t.Fatalf("expected only 20%% unstatted robber equipment fallback %+v, got %+v", expectedItems, over.Result.Items)
	}
}

func TestBuildOverSuppressesExperienceWhenPlayerOutlevelsMonsterByMoreThanSeven(t *testing.T) {
	defer useSourceEncounterRoll(func(maxExclusive int) int { return 0 })()
	over := (&Runtime{
		BattleID: "battle-map49-overlevel",
		MapID:    "49",
		Round:    1,
		Cells: []CellInfoPush{
			{Handle: "player_21424", Camp: CampTeam, Level: 23},
			{Handle: "enemy_1", Camp: CampEnemy, Level: 15},
		},
	}).buildOver(CampTeam)
	if over.Result.ExpDelta != 0 {
		t.Fatalf("expected player level 23 versus monster level 15 to suppress exp, got %d", over.Result.ExpDelta)
	}
	expectedItems := []string{"铜钱x8", "盗贼的首级x1", "L小喇叭x1", "红方巾x1", "毛皮x1"}
	if !reflect.DeepEqual(over.Result.Items, expectedItems) {
		t.Fatalf("expected overlevel battle to keep probability-rolled item rewards %+v, got %+v", expectedItems, over.Result.Items)
	}
}

func TestBuildOverKeepsExperienceWhenPlayerIsOnlySevenLevelsAboveMonster(t *testing.T) {
	over := (&Runtime{
		BattleID: "battle-map49-boundary",
		MapID:    "49",
		Round:    1,
		Cells: []CellInfoPush{
			{Handle: "player_21424", Camp: CampTeam, Level: 22},
			{Handle: "enemy_1", Camp: CampEnemy, Level: 15},
		},
	}).buildOver(CampTeam)
	if over.Result.ExpDelta != 610 {
		t.Fatalf("expected player level 22 versus monster level 15 to keep exp 610, got %d", over.Result.ExpDelta)
	}
}

func TestBuildOverUsesCapturedCracktoadVisibleBossReward(t *testing.T) {
	defer useSourceEncounterRoll(func(maxExclusive int) int { return 0 })()
	plainMap143 := (&Runtime{BattleID: "battle-map143", MapID: "143", Round: 1}).buildOver(CampTeam)
	if plainMap143.Result.ExpDelta != 0 || len(plainMap143.Result.Items) != 0 {
		t.Fatalf("expected map143 without visible boss handle to stay empty, got %+v", plainMap143.Result)
	}

	over := (&Runtime{
		BattleID:            "battle-cracktoad",
		MapID:               "143",
		SourceMonsterHandle: "5176206909809579",
		Round:               1,
	}).buildOver(CampTeam)
	if over == nil {
		t.Fatal("expected OverBattle push")
	}
	expectedItems := []string{"毒囊x5", "肉x1", "黏液x1", "蛤蟆精战靴x1", "铜钱x50"}
	if over.Result.ExpDelta != 0 {
		t.Fatalf("expected captured cracktoad reward exp 0, got %d", over.Result.ExpDelta)
	}
	if len(over.Result.Items) != len(expectedItems) {
		t.Fatalf("expected cracktoad reward items %+v, got %+v", expectedItems, over.Result.Items)
	}
	for index, item := range expectedItems {
		if over.Result.Items[index] != item {
			t.Fatalf("expected cracktoad reward item %d to be %q, got %+v", index, item, over.Result.Items)
		}
	}
}

func TestBuildOverUsesCapturedShuiliandongVisibleMonsterBaseRewards(t *testing.T) {
	defer useSourceEncounterRoll(func(maxExclusive int) int { return 0 })()
	testCases := []struct {
		mapID  string
		handle string
		items  []string
	}{
		{mapID: "131", handle: "8128205778897212", items: []string{"毒囊x1", "黏液x1", "竹x1"}},
		{mapID: "143", handle: "5166206909805441", items: []string{"L小喇叭x1", "黏液x1", "甘露x1", "毒囊x1"}},
		{mapID: "145", handle: "2890206197338884", items: []string{"毒囊x2", "翠带护腰x1"}},
	}

	for _, testCase := range testCases {
		over := (&Runtime{
			BattleID:            "battle-shuiliandong-visible",
			MapID:               testCase.mapID,
			SourceMonsterHandle: testCase.handle,
			Round:               1,
		}).buildOver(CampTeam)
		if over == nil {
			t.Fatalf("expected OverBattle push for %+v", testCase)
		}
		if !reflect.DeepEqual(over.Result.Items, testCase.items) {
			t.Fatalf("expected shuiliandong visible monster %+v low-roll rewards, got %+v", testCase, over.Result.Items)
		}
	}
}

func TestPointCouponThiefVisibleBattleConfigAndReward(t *testing.T) {
	handle := classicactivity.PointCouponThiefHandle(114, 1781888400, 0)
	config, ok := sourceVisibleMonsterConfigForHandle("114", handle)
	if !ok {
		t.Fatal("expected point coupon thief dynamic visible monster config")
	}
	if config.Cell.Name != "点券盗贼" || config.Cell.DisplayURL != "monstermap/militia.swf" || config.Cell.Level != 15 || config.Cell.MaxHP != 1200 || config.Cell.MaxMP != 300 {
		t.Fatalf("expected captured point coupon thief battle cell, got %+v", config.Cell)
	}
	if !sourceEnemyCanRampage(&config.Cell) {
		t.Fatalf("expected point coupon thief to cast 暴走之力, got %+v", config.Cell)
	}

	over := (&Runtime{
		BattleID:            "battle-point-coupon-thief",
		MapID:               "114",
		SourceMonsterHandle: handle,
		Round:               1,
	}).buildOver(CampTeam)
	if over == nil || over.Result.ExpDelta != 50 || !reflect.DeepEqual(over.Result.Items, []string{"点券x10"}) {
		t.Fatalf("expected point coupon thief 点券x10 reward with exp 50, got %+v", over)
	}
}

func TestBuildOverUsesCapturedHuangfengzhaiBossRewards(t *testing.T) {
	defer useSourceEncounterRoll(func(maxExclusive int) int { return 0 })()
	testCases := []struct {
		mapID    string
		handle   string
		expDelta int
		items    []string
	}{
		{
			mapID:    "149",
			handle:   "3218685759638239",
			expDelta: 336,
			items:    []string{"毛皮x2", "铜钱x131", "盗贼的首级x1", "黄风腰带x1", "兽骨x1", "黄风围巾x1", "头骨x1", "雪莲花x1"},
		},
		{
			mapID:    "149",
			handle:   "3220685759639165",
			expDelta: 336,
			items:    []string{"毛皮x2", "铜钱x131", "盗贼的首级x1", "黄风腰带x1", "兽骨x1", "黄风围巾x1", "头骨x1", "雪莲花x1"},
		},
		{
			mapID:    "155",
			handle:   "2600686416056495",
			expDelta: 720,
			items:    []string{"红方巾x2", "绸缎x3", "铜钱x280", "盗贼的首级x1", "呼啸战靴x1", "寨夫人上衣x1", "寨夫人护腕x1", "宝匣x1"},
		},
		{
			mapID:    "155",
			handle:   "2800686416057704",
			expDelta: 720,
			items:    []string{"红方巾x2", "绸缎x3", "铜钱x280", "盗贼的首级x1", "呼啸战靴x1", "寨夫人上衣x1", "寨夫人护腕x1", "宝匣x1"},
		},
	}

	for _, testCase := range testCases {
		over := (&Runtime{
			BattleID:            "battle-huangfengzhai-boss",
			MapID:               testCase.mapID,
			SourceMonsterHandle: testCase.handle,
			Round:               1,
		}).buildOver(CampTeam)
		if over == nil {
			t.Fatalf("expected OverBattle push for %+v", testCase)
		}
		if over.Result.ExpDelta != testCase.expDelta {
			t.Fatalf("expected huangfengzhai boss reward exp %d for %+v, got %d", testCase.expDelta, testCase, over.Result.ExpDelta)
		}
		if len(over.Result.Items) != len(testCase.items) {
			t.Fatalf("expected huangfengzhai boss reward items %+v for %+v, got %+v", testCase.items, testCase, over.Result.Items)
		}
		for index, item := range testCase.items {
			if over.Result.Items[index] != item {
				t.Fatalf("expected huangfengzhai boss reward item %d to be %q for %+v, got %+v", index, item, testCase, over.Result.Items)
			}
		}
	}
}

func TestBuildOverUsesCapturedHuangfengzhaiBossObservedDropRates(t *testing.T) {
	defer useSourceEncounterRoll(func(maxExclusive int) int { return maxExclusive - 1 })()

	secondChief := (&Runtime{
		BattleID:            "battle-huangfengzhai-second-chief-high-roll",
		MapID:               "149",
		SourceMonsterHandle: "3218685759638239",
		Round:               1,
	}).buildOver(CampTeam)
	expectedSecondChiefItems := []string{"毛皮x2", "铜钱x131", "盗贼的首级x1"}
	if secondChief == nil || secondChief.Result.ExpDelta != 336 || !reflect.DeepEqual(secondChief.Result.Items, expectedSecondChiefItems) {
		t.Fatalf("expected high-roll second-chief drops to keep only 2/2 items %+v, got %+v", expectedSecondChiefItems, secondChief)
	}

	chief := (&Runtime{
		BattleID:            "battle-huangfengzhai-chief-high-roll",
		MapID:               "155",
		SourceMonsterHandle: "2600686416056495",
		Round:               1,
	}).buildOver(CampTeam)
	expectedChiefItems := []string{"红方巾x2", "绸缎x3", "铜钱x280", "盗贼的首级x1"}
	if chief == nil || chief.Result.ExpDelta != 720 || !reflect.DeepEqual(chief.Result.Items, expectedChiefItems) {
		t.Fatalf("expected high-roll chief drops to keep only 2/2 items %+v, got %+v", expectedChiefItems, chief)
	}
}

func TestBuildOverUsesCapturedShihukuRewardCandidates(t *testing.T) {
	defer useSourceEncounterRoll(func(maxExclusive int) int { return 0 })()

	bossOver := (&Runtime{
		BattleID: "battle-shihuku-boss-reward-candidate",
		MapID:    "167",
		Round:    1,
		Cells: []CellInfoPush{
			{Camp: CampTeam, Handle: "player_1", Name: "玩家", Level: 30},
			{Camp: CampEnemy, Handle: "enemy_chiluking", Name: "蚩颅王", Level: 30, MaxHP: 6000, HP: 6000},
		},
	}).buildOver(CampTeam)
	if bossOver == nil {
		t.Fatal("expected shihuku boss OverBattle push")
	}
	if bossOver.Result.ExpDelta != 558 {
		t.Fatalf("expected shihuku boss captured candidate exp 558, got %d", bossOver.Result.ExpDelta)
	}
	expectedBossItems := []string{"兽血x1", "L小喇叭x1", "兽骨x2", "石块x1", "碎金片x1", "铁块x1", "蚩颅王的角x1", "蚩颅王的头x1", "蚩颅王护肩x1", "兽牙x1", "宝匣x1"}
	if !reflect.DeepEqual(bossOver.Result.Items, expectedBossItems) {
		t.Fatalf("expected shihuku boss low-roll reward %+v, got %+v", expectedBossItems, bossOver.Result.Items)
	}

	blackshadowOver := (&Runtime{
		BattleID: "battle-shihuku-blackshadow-reward-candidate",
		MapID:    "162",
		Round:    1,
		Cells: []CellInfoPush{
			{Camp: CampEnemy, Handle: "enemy_blackshadow", Name: "黑影", Level: 26, MaxHP: 2300, HP: 2300},
		},
	}).buildOver(CampTeam)
	expectedBlackshadowItems := []string{"肉x1", "兽牙x1", "兽血x1", "兽骨x1", "L小喇叭x1"}
	if !reflect.DeepEqual(blackshadowOver.Result.Items, expectedBlackshadowItems) {
		t.Fatalf("expected shihuku blackshadow low-roll reward %+v, got %+v", expectedBlackshadowItems, blackshadowOver.Result.Items)
	}
}

func TestBuildOverUsesCapturedHuangfengzhaiMap147RewardCandidate(t *testing.T) {
	defer useSourceEncounterRoll(func(maxExclusive int) int { return 0 })()

	over := (&Runtime{
		BattleID: "battle-huangfengzhai-map147-reward-candidate",
		MapID:    "147",
		Round:    1,
		Cells: []CellInfoPush{
			{Camp: CampEnemy, Handle: "7633284548716137", Name: "蛮族刀客", Level: 12, MaxHP: 520, HP: 520},
		},
	}).buildOver(CampTeam)
	if over == nil {
		t.Fatal("expected huangfengzhai map147 OverBattle push")
	}
	if over.Result.ExpDelta != 0 {
		t.Fatalf("expected huangfengzhai map147 captured candidate exp 0, got %d", over.Result.ExpDelta)
	}
	expectedItems := []string{"铜钱x5", "刀客布衣x1"}
	if !reflect.DeepEqual(over.Result.Items, expectedItems) {
		t.Fatalf("expected huangfengzhai map147 low-roll reward %+v, got %+v", expectedItems, over.Result.Items)
	}
}

func TestHuangfengzhaiRewardCandidatesUseFullCaptureStatistics(t *testing.T) {
	testCases := []struct {
		mapID       string
		monsterName string
		maxHP       int
		itemName    string
		quantity    int
		numerator   int
		denominator int
	}{
		{mapID: "122", monsterName: "秃鹫", maxHP: 215, itemName: "刺", quantity: 1, numerator: 2, denominator: 2},
		{mapID: "146", monsterName: "蛮族刀客", maxHP: 520, itemName: "红缨", quantity: 1, numerator: 7, denominator: 9},
		{mapID: "150", monsterName: "蛮族弓手", maxHP: 530, itemName: "铜钱", quantity: 8, numerator: 2, denominator: 2},
		{mapID: "153", monsterName: "咒巫师", maxHP: 500, itemName: "图腾面具", quantity: 1, numerator: 3, denominator: 8},
		{mapID: "156", monsterName: "咒巫师", maxHP: 550, itemName: "兽骨", quantity: 1, numerator: 10, denominator: 11},
	}

	for _, testCase := range testCases {
		config, ok := sourceBattleRewardCandidateForCell(testCase.mapID, testCase.monsterName, testCase.maxHP)
		if !ok {
			t.Fatalf("expected Huangfeng reward candidate %+v", testCase)
		}
		rate := requireSourceBattleRewardDropRate(t, config.DropRates, testCase.itemName)
		if rate.Quantity != testCase.quantity || rate.Numerator != testCase.numerator || rate.Denominator != testCase.denominator {
			t.Fatalf("expected Huangfeng reward candidate drop %+v, got %+v", testCase, rate)
		}
	}
}

func TestBaiyuanYaozhisenRewardCandidatesIncludeLateMapChain(t *testing.T) {
	testCases := []struct {
		mapID       string
		monsterName string
		maxHP       int
		itemName    string
		quantity    int
		numerator   int
		denominator int
	}{
		{mapID: "201", monsterName: "机木玄师", maxHP: 1189, itemName: "木材", quantity: 1, numerator: 169, denominator: 282},
		{mapID: "201", monsterName: "机木玄师", maxHP: 1189, itemName: "机木护腰", quantity: 1, numerator: 10, denominator: 282},
		{mapID: "202", monsterName: "机木锥兵", maxHP: 1330, itemName: "木材", quantity: 1, numerator: 38, denominator: 50},
		{mapID: "202", monsterName: "机木锥兵", maxHP: 1330, itemName: "神奇朽木", quantity: 1, numerator: 2, denominator: 50},
	}

	for _, testCase := range testCases {
		config, ok := sourceBattleRewardCandidateForCell(testCase.mapID, testCase.monsterName, testCase.maxHP)
		if !ok {
			t.Fatalf("expected Baiyuan/Yaozhisen reward candidate %+v", testCase)
		}
		rate := requireSourceBattleRewardDropRate(t, config.DropRates, testCase.itemName)
		if rate.Quantity != testCase.quantity || rate.Numerator != testCase.numerator || rate.Denominator != testCase.denominator {
			t.Fatalf("expected Baiyuan/Yaozhisen reward candidate drop %+v, got %+v", testCase, rate)
		}
	}
}

func TestGrasslandRewardCandidatesUseCapturedAutoMoveStatistics(t *testing.T) {
	testCases := []struct {
		mapID       string
		monsterName string
		maxHP       int
		itemName    string
		quantity    int
		numerator   int
		denominator int
	}{
		{mapID: "210", monsterName: "草刺槐", maxHP: 1530, itemName: "韧苇叶", quantity: 1, numerator: 38, denominator: 48},
		{mapID: "210", monsterName: "草须芒", maxHP: 2000, itemName: "斗草叶", quantity: 1, numerator: 21, denominator: 32},
		{mapID: "216", monsterName: "草球蔓", maxHP: 1357, itemName: "斗草叶", quantity: 1, numerator: 126, denominator: 262},
		{mapID: "222", monsterName: "草球蔓", maxHP: 1450, itemName: "斗草叶", quantity: 1, numerator: 64, denominator: 124},
	}

	for _, testCase := range testCases {
		config, ok := sourceBattleRewardCandidateForCell(testCase.mapID, testCase.monsterName, testCase.maxHP)
		if !ok || config.Status != "candidate" {
			t.Fatalf("expected grassland capture candidate %+v, got %+v", testCase, config)
		}
		rate := requireSourceBattleRewardDropRate(t, config.DropRates, testCase.itemName)
		if rate.Quantity != testCase.quantity || rate.Numerator != testCase.numerator || rate.Denominator != testCase.denominator {
			t.Fatalf("expected grassland reward candidate drop %+v, got %+v", testCase, rate)
		}
	}
}

func TestBuildOverUsesCaptureBackedWuliangYaozhisenCandidateRewards(t *testing.T) {
	defer useSourceEncounterRoll(func(maxExclusive int) int { return 0 })()

	over := (&Runtime{
		BattleID: "battle-yaozhisen-candidate-boundary",
		MapID:    "201",
		Round:    1,
		Cells: []CellInfoPush{
			{Camp: CampTeam, Handle: "player_1", Name: "玩家", Level: 34},
			{Camp: CampEnemy, Handle: "enemy_robothyun", Name: "机木玄师", Level: 34, MaxHP: 1189, HP: 1189},
		},
	}).buildOver(CampTeam)
	if over == nil {
		t.Fatal("expected Yaozhisen OverBattle push")
	}
	if over.Result.ExpDelta != 2576 {
		t.Fatalf("expected map201 captured positive-exp mode 2576, got %+v", over.Result)
	}
	for _, item := range []string{"木材x1", "暗力之源x1", "机木护腰x1"} {
		if !containsSourceBattleRewardItem(over.Result.Items, item) {
			t.Fatalf("expected map201 captured candidate drop %s, got %+v", item, over.Result.Items)
		}
	}
}

func TestBuildOverWuliangYaozhisenCandidateRewardDoesNotDuplicateForDoubleMystics(t *testing.T) {
	defer useSourceEncounterRoll(func(maxExclusive int) int { return 0 })()

	over := (&Runtime{
		BattleID: "battle-yaozhisen-double-mystic-reward",
		MapID:    "201",
		Round:    1,
		Cells: []CellInfoPush{
			{Camp: CampTeam, Handle: "player_1", Name: "玩家", Level: 34},
			{Camp: CampEnemy, Handle: "enemy_robothyun_1", Name: "机木玄师", Level: 34, MaxHP: 1189, HP: 0},
			{Camp: CampEnemy, Handle: "enemy_robothyun_2", Name: "机木玄师", Level: 34, MaxHP: 1189, HP: 0},
		},
	}).buildOver(CampTeam)
	if over == nil || over.Result.ExpDelta != 2576 {
		t.Fatalf("expected one map201 candidate reward, got %+v", over)
	}
	if containsSourceBattleRewardItem(over.Result.Items, "木材x2") {
		t.Fatalf("expected one encounter reward roll, not combined double-mystic stacks, got %+v", over.Result.Items)
	}
}

func TestBuildOverWuliangYaozhisenCandidateRewardDoesNotApplyOnEscape(t *testing.T) {
	defer useSourceEncounterRoll(func(maxExclusive int) int { return 0 })()

	over := (&Runtime{
		BattleID: "battle-yaozhisen-escape-reward",
		MapID:    "202",
		Round:    1,
		Cells: []CellInfoPush{
			{Camp: CampTeam, Handle: "player_1", Name: "玩家", Level: 35},
			{Camp: CampEnemy, Handle: "enemy_robotawl", Name: "机木锥兵", Level: 35, MaxHP: 1330, HP: 1330},
		},
	}).buildOver(CampTeam, true)
	if over == nil || over.Result.ExpDelta != 0 || len(over.Result.Items) != 0 || !over.Result.Escaped {
		t.Fatalf("expected escaped map202 battle to have no candidate reward, got %+v", over)
	}
}

func TestBuildOverRollsCapturedObservedDropRates(t *testing.T) {
	defer useSourceEncounterRoll(func(maxExclusive int) int { return maxExclusive - 1 })()
	over := (&Runtime{BattleID: "battle-map5-high-roll", MapID: "5", Round: 1}).buildOver(CampTeam)
	expectedItems := []string{"肉x1"}
	if !reflect.DeepEqual(over.Result.Items, expectedItems) {
		t.Fatalf("expected 100%% meat to survive high roll and 50%% tooth to drop out, got %+v", over.Result.Items)
	}
}

func TestBuildOverUsesFeixiandongBossRewardDropRates(t *testing.T) {
	defer useSourceEncounterRoll(func(maxExclusive int) int { return 0 })()

	largerockLowRoll := (&Runtime{
		BattleID:            "battle-feixiandong-largerock-reward",
		MapID:               "76",
		SourceMonsterHandle: "1048675671977626",
		Round:               1,
	}).buildOver(CampTeam)
	if largerockLowRoll == nil || largerockLowRoll.Result.ExpDelta != 682 {
		t.Fatalf("expected captured largerock reward exp, got %+v", largerockLowRoll)
	}
	expectedLargerockLowRollItems := []string{"废渣x1", "石块x2", "水晶x1", "岩魔菱石x1", "岩魔球石x1", "巨岩魔的拳x1", "巨岩魔的头x1", "岩化护腿x1", "蓝晶护肩x1", "岩魔剑x1", "宝匣x1"}
	if !reflect.DeepEqual(largerockLowRoll.Result.Items, expectedLargerockLowRollItems) {
		t.Fatalf("expected low-roll largerock drops to include latest capture items, got %+v", largerockLowRoll.Result.Items)
	}

	magicrockmanLowRoll := (&Runtime{
		BattleID:            "battle-feixiandong-magicrockman-reward",
		MapID:               "78",
		SourceMonsterHandle: "1681675260686878",
		Round:               1,
	}).buildOver(CampTeam)
	expectedMagicrockmanLowRollItems := []string{"废渣x1", "碎铁矿x1", "石块x2", "岩化护腿x1", "碎金片x1", "岩魔菱石x1"}
	if magicrockmanLowRoll == nil || magicrockmanLowRoll.Result.ExpDelta != 658 || !reflect.DeepEqual(magicrockmanLowRoll.Result.Items, expectedMagicrockmanLowRollItems) {
		t.Fatalf("expected low-roll magicrockman drops to include latest capture items, got %+v", magicrockmanLowRoll)
	}

	defer useSourceEncounterRoll(func(maxExclusive int) int { return maxExclusive - 1 })()
	largerockHighRoll := (&Runtime{
		BattleID:            "battle-feixiandong-largerock-reward-high",
		MapID:               "76",
		SourceMonsterHandle: "1048675671977626",
		Round:               1,
	}).buildOver(CampTeam)
	expectedLargerockHighRollItems := []string{"废渣x1", "石块x2", "水晶x1", "岩魔菱石x1", "岩魔球石x1", "巨岩魔的拳x1"}
	if largerockHighRoll == nil || !reflect.DeepEqual(largerockHighRoll.Result.Items, expectedLargerockHighRollItems) {
		t.Fatalf("expected high-roll largerock drops to exclude 20%% equipment and 1/2 items, got %+v", largerockHighRoll)
	}

	magicrockmanHighRoll := (&Runtime{
		BattleID:            "battle-feixiandong-magicrockman-reward-high",
		MapID:               "78",
		SourceMonsterHandle: "1681675260686878",
		Round:               1,
	}).buildOver(CampTeam)
	expectedMagicrockmanHighRollItems := []string{"石块x2"}
	if magicrockmanHighRoll == nil || !reflect.DeepEqual(magicrockmanHighRoll.Result.Items, expectedMagicrockmanHighRollItems) {
		t.Fatalf("expected high-roll magicrockman drops to keep only observed 2/2 material, got %+v", magicrockmanHighRoll)
	}
}

func TestSourceBattleConfigTablesLoadCapturedRows(t *testing.T) {
	enemy, ok := sourceEnemyConfigForMap("49")
	if !ok {
		t.Fatal("expected map49 wild enemy config")
	}
	if enemy.Cell.Name != "盗贼" || enemy.Cell.DisplayURL != "monstermap/robber.swf" || enemy.Cell.MaxHP != 445 || enemy.Cell.MaxMP != 194 || enemy.Cell.Attack != 107 {
		t.Fatalf("expected captured map49 robber config, got %+v", enemy)
	}
	if enemy.QueueIndexTeam != 1 || enemy.QueueIndexEnemy != 4 {
		t.Fatalf("expected captured map49 queue indexes 1/4, got %+v", enemy)
	}

	map4, ok := sourceEnemyConfigForMap("4")
	if !ok {
		t.Fatal("expected map4 wild enemy config")
	}
	if map4.QueueIndexTeam != 0 || map4.QueueIndexEnemy != 0 {
		t.Fatalf("expected captured map4 queue indexes 0/0, got %+v", map4)
	}

	visibleBoss, ok := sourceVisibleMonsterConfigForHandle("143", "5176206909809579")
	if !ok {
		t.Fatal("expected captured shuiliandong visible boss config")
	}
	if visibleBoss.Cell.Name != "蛤蟆精" || visibleBoss.Cell.DisplayURL != "monstermap/cracktoad.swf" {
		t.Fatalf("expected cracktoad visible boss config, got %+v", visibleBoss.Cell)
	}
	if visibleBoss.Cell.Level != 20 || visibleBoss.Cell.MaxHP != 1500 || visibleBoss.Cell.MaxMP != 564 {
		t.Fatalf("expected captured cracktoad level/hp/mp, got %+v", visibleBoss.Cell)
	}
	if visibleBoss.QueueIndexTeam != 1 || visibleBoss.QueueIndexEnemy != 4 {
		t.Fatalf("expected visible monster queue indexes 1/4, got %+v", visibleBoss)
	}

	visibleAttackCases := []struct {
		mapID  string
		handle string
		name   string
		level  int
		label  string
	}{
		{mapID: "131", handle: "8128205778897212", name: "武斗蛤蟆", level: 12, label: "普通攻击"},
		{mapID: "132", handle: "1656205827185847", name: "武斗蛤蟆", level: 13, label: "普通攻击"},
		{mapID: "133", handle: "8112205902790159", name: "武斗蛤蟆", level: 14, label: "普通攻击"},
		{mapID: "144", handle: "2762206074545916", name: "武斗蛤蟆", level: 16, label: "普通攻击"},
		{mapID: "140", handle: "8430206376341780", name: "武斗蛤蟆", level: 17, label: "普通攻击"},
		{mapID: "137", handle: "4889205982270617", name: "剑术蛤蟆", level: 15, label: "普通攻击"},
		{mapID: "144", handle: "2768206074548639", name: "剑术蛤蟆", level: 17, label: "普通攻击"},
		{mapID: "145", handle: "2890206197338884", name: "剑术蛤蟆", level: 18, label: "普通攻击"},
		{mapID: "143", handle: "5172206909807859", name: "剑术蛤蟆", level: 18, label: "普通攻击"},
		{mapID: "137", handle: "4895205982272135", name: "法术蛤蟆", level: 16, label: "法术普通攻击"},
		{mapID: "143", handle: "5166206909805441", name: "法术蛤蟆", level: 17, label: "法术普通攻击"},
	}
	for _, testCase := range visibleAttackCases {
		config, ok := sourceVisibleMonsterConfigForHandle(testCase.mapID, testCase.handle)
		if !ok {
			t.Fatalf("expected captured shuiliandong visible monster config for %s/%s", testCase.mapID, testCase.handle)
		}
		if config.Cell.Name != testCase.name || config.Cell.Level != testCase.level || config.Cell.CommandLabel != testCase.label {
			t.Fatalf("expected captured visible monster %+v, got %+v", testCase, config.Cell)
		}
	}

	reward, ok := sourceBattleRewardConfigForMap("5")
	if !ok {
		t.Fatal("expected map5 battle reward config")
	}
	if reward.Status != "confirmed" || reward.ExpDelta != 0 || len(reward.Items) != 2 || len(reward.DropRates) != 2 {
		t.Fatalf("expected confirmed captured map5 reward config, got %+v", reward)
	}

	map49Reward, ok := sourceBattleRewardConfigForMap("49")
	if !ok {
		t.Fatal("expected map49 battle reward config")
	}
	if map49Reward.Status != "confirmed" || map49Reward.ExpDelta != 610 || len(map49Reward.Items) != 5 || len(map49Reward.DropRates) != 5 {
		t.Fatalf("expected confirmed captured map49 reward config, got %+v", map49Reward)
	}

	cracktoadReward, ok := sourceBattleRewardConfigForEncounter("143", "5176206909809579")
	if !ok {
		t.Fatal("expected cracktoad visible boss reward config")
	}
	if cracktoadReward.SourceMonsterHandle != "5176206909809579" || cracktoadReward.ExpDelta != 0 || len(cracktoadReward.Items) != 5 || len(cracktoadReward.DropRates) != 5 {
		t.Fatalf("expected captured cracktoad visible boss reward config, got %+v", cracktoadReward)
	}

	map50Configs := sourceEnemyConfigsForMap("50")
	if len(map50Configs) != 2 {
		t.Fatalf("expected captured map50 normal enemy attacks 203/216, got %+v", map50Configs)
	}
	for _, config := range map50Configs {
		if config.Cell.Name == "魔族射手" {
			t.Fatalf("expected map50 special-event 魔族射手 to stay out of normal wild encounters, got %+v", map50Configs)
		}
	}

	swampCases := []struct {
		mapID  string
		names  []string
		maxHPs []int
	}{
		{mapID: "56", names: []string{"小恶鬼", "瘴气泥巴"}, maxHPs: []int{600, 510}},
		{mapID: "57", names: []string{"瘴气泥巴", "瘴气泥巴"}, maxHPs: []int{520, 525}},
		{mapID: "58", names: []string{"瘴气泥巴", "瘴气泥巴"}, maxHPs: []int{520, 525}},
		{mapID: "59", names: []string{"王花", "王花"}, maxHPs: []int{750, 760}},
		{mapID: "60", names: []string{"毒蜂", "王花"}, maxHPs: []int{680, 750}},
		{mapID: "61", names: []string{"王花", "王花"}, maxHPs: []int{750, 760}},
		{mapID: "62", names: []string{"毒蜂"}, maxHPs: []int{680}},
		{mapID: "63", names: []string{"金斑鳄", "金斑鳄"}, maxHPs: []int{980, 1020}},
		{mapID: "172", names: []string{"玄龟兽", "玄龟兽"}, maxHPs: []int{1221, 1260}},
		{mapID: "173", names: []string{"玄龟兽"}, maxHPs: []int{1221}},
		{mapID: "174", names: []string{"玄龟兽", "毒蜂"}, maxHPs: []int{1260, 850}},
		{mapID: "175", names: []string{"赤蛰子", "毒蜂", "玄龟兽"}, maxHPs: []int{3100, 850, 1260}},
		{mapID: "177", names: []string{"赤蛰子", "毒蜂"}, maxHPs: []int{3100, 850}},
		{mapID: "178", names: []string{"毒蜂"}, maxHPs: []int{850}},
	}
	for _, testCase := range swampCases {
		configs := sourceEnemyConfigsForMap(testCase.mapID)
		if len(configs) != len(testCase.names) {
			t.Fatalf("expected swamp map %s to have %d captured wild configs, got %+v", testCase.mapID, len(testCase.names), configs)
		}
		for index, expectedName := range testCase.names {
			config := configs[index]
			if config.Cell.Name != expectedName || config.Cell.MaxHP != testCase.maxHPs[index] {
				t.Fatalf("expected swamp map %s enemy %d %s hp=%d, got %+v", testCase.mapID, index, expectedName, testCase.maxHPs[index], config.Cell)
			}
		}
	}
	if _, ok := sourceEnemyConfigForMap("171"); ok {
		t.Fatal("map171 must stay out of wild encounters; 百年虫精 is a visible special-event encounter")
	}

	// 百年虫精 is a capture-backed visible special event on map171, not a wild encounter.
	bainian, ok := sourceVisibleMonsterConfigForHandle("171", "7893833328746190")
	if !ok {
		t.Fatal("expected map171 visible 百年虫精 config")
	}
	if bainian.Cell.Name != "百年虫精" || bainian.Cell.DisplayURL != "monstermap/wocmon.swf" || bainian.Cell.Level != 30 || bainian.Cell.MaxHP != 8000 || bainian.Cell.MaxMP != 1500 || bainian.Cell.Attack != 202 || bainian.Cell.CommandLabel != "法术普通攻击" || bainian.Cell.DamageDefenseType != "magic" {
		t.Fatalf("unexpected 百年虫精 config: %+v", bainian.Cell)
	}
	group, ok := sourceVisibleMonsterConfigsForHandle("171", "7893833328746190")
	if !ok || len(group) != 4 {
		t.Fatalf("expected 百年虫精 encounter group size 4, got %+v", group)
	}
	rangerMilitia := group[1].Cell
	if rangerMilitia.Handle != "7895833328747103" || rangerMilitia.Vocation != "游侠+" || rangerMilitia.MaxHP != 1800 || rangerMilitia.MaxMP != 600 || rangerMilitia.CommandLabel != "普通攻击" || rangerMilitia.DamageDefenseType != "physical" {
		t.Fatalf("unexpected captured ranger militia config: %+v", rangerMilitia)
	}
	reward, ok = sourceBattleRewardConfigForExactEncounter("171", "7893833328746190")
	if !ok || reward.ExpDelta != 0 {
		t.Fatalf("expected 百年虫精 reward exp 0, got %+v", reward)
	}
	wantItems := []string{"宠物成长药剂x6", "铜钱x500", "魔匣x2", "阴阳结x1", "回魂丹x1"}
	if len(reward.Items) != len(wantItems) {
		t.Fatalf("expected 百年虫精 reward items %+v, got %+v", wantItems, reward.Items)
	}
	for i, item := range wantItems {
		if reward.Items[i] != item {
			t.Fatalf("expected 百年虫精 reward item %d %q, got %+v", i, item, reward.Items)
		}
	}
	if _, ok := sourceEnemyConfigForMap("176"); ok {
		t.Fatal("map176 must stay out of wild encounters until battle/reward capture evidence exists")
	}

	for mapID, configs := range sourceWildEnemyConfigsByMapID {
		for _, config := range configs {
			if config.Cell.Attack <= 0 {
				t.Fatalf("expected captured enemy attack to be filled for map %s, got %+v", mapID, config.Cell)
			}
		}
	}
}

func TestCaptureBackedYaozhisenNormalAttackUsesConfiguredRange(t *testing.T) {
	defer useSourceBattleAttackRoll(func(maxExclusive int) int { return maxExclusive - 1 })()
	defer useSourceEncounterRoll(func(int) int { return 227 })()

	enemy, ok := sourceEnemyConfigForMap("201")
	if !ok || enemy.Cell.Attack != 193 || enemy.AttackMin != 186 || enemy.AttackMax != 374 {
		t.Fatalf("expected map201 half-defense calibrated attack 193 and range 186..374, got %+v", enemy)
	}
	runtime, bundle, started := NewWildBattle(
		session.RoleSummary{RoleID: "player_yaozhisen_range", DisplayName: "测试女侠", Level: 34},
		session.PlayerBaseData{RoleID: "player_yaozhisen_range", DisplayName: "测试女侠", Level: 34},
		StartRequest{MapID: "201", MapName: "妖之森_10"},
	)
	if !started || len(bundle.Cells) != 2 {
		t.Fatalf("expected map201 capture-backed encounter, got started=%v bundle=%+v", started, bundle)
	}
	actor := runtime.cellByHandle(bundle.Cells[1].Handle)
	if actor == nil || runtime.EnemyAttackRanges[actor.Handle] != (battleAttackRange{Min: 186, Max: 374}) {
		t.Fatalf("expected map201 range to reach runtime actor, got actor=%+v ranges=%+v", actor, runtime.EnemyAttackRanges)
	}
	if damage := runtime.baseBattleDamage(actor, commandProfile{SourceActionLabel: "nomalAtk", DamageMultiplier: 1}, 20); damage != 354 {
		t.Fatalf("expected upper bound 374 minus defense 20, got %d", damage)
	}
	if damage := (&Runtime{}).baseBattleDamage(actor, commandProfile{SourceActionLabel: "nomalAtk", DamageMultiplier: 1}, 20); damage != 173 {
		t.Fatalf("expected missing range to retain fixed attack proxy 193 minus defense 20, got %d", damage)
	}
}

func TestEnemyRobotupUsesCapturedTargetMPAndHealingRange(t *testing.T) {
	defer useSourceBattleHealRoll(func(maxExclusive int) int { return maxExclusive - 1 })()

	runtime := &Runtime{
		BattleID: "battle-robotup",
		Round:    3,
		Cells: []CellInfoPush{
			{Handle: "player_1", Camp: CampTeam, HP: 900, MaxHP: 900},
			{Handle: "robot_full", Name: "机木玄师", DisplayURL: "monstermap/robothyun.swf", Camp: CampEnemy, HP: 1189, MaxHP: 1189, MP: 910, MaxMP: 910},
			{Handle: "robot_low", Name: "机木玄师", DisplayURL: "monstermap/robothyun.swf", Camp: CampEnemy, HP: 100, MaxHP: 1189, MP: 910, MaxMP: 910},
		},
	}
	actor := runtime.cellByHandle("robot_full")
	target := runtime.cellByHandle("robot_low")
	if command := runtime.enemyBattleCommand(actor, runtime.cellByHandle("player_1")); command != CommandEnemyRobotUp {
		t.Fatalf("expected capture-backed robotup selection for lowest living mystic, got %q", command)
	}
	target.HP = 900
	action := runtime.resolveEnemyRobotupAction(actor, target)
	if action.ActionName != "机木修复" || action.CommandID != CommandEnemyRobotUp || action.SourceMode != "0" || action.SourceActionLabel != "robotup" {
		t.Fatalf("expected captured robotup action fields, got %+v", action)
	}
	if action.TargetHandle != target.Handle || target.HP != 1189 || actor.MP != 850 {
		t.Fatalf("expected robotup to cap target at max HP and spend 60 MP, actor=%+v target=%+v action=%+v", actor, target, action)
	}
	if len(action.RefreshInfos) != 2 || action.RefreshInfos[0].Handle != target.Handle || action.RefreshInfos[1].Handle != actor.Handle {
		t.Fatalf("expected ally robotup refresh order target then actor, got %+v", action.RefreshInfos)
	}
}

func TestEnemyRobotupSelfUsesCapturedHealingMinimum(t *testing.T) {
	defer useSourceBattleHealRoll(func(int) int { return 0 })()

	runtime := &Runtime{BattleID: "battle-robotup-self", Round: 2}
	actor := &CellInfoPush{Handle: "robot_self", Name: "机木玄师", DisplayURL: "monstermap/robothyun.swf", Camp: CampEnemy, HP: 500, MaxHP: 1189, MP: 910, MaxMP: 910}
	action := runtime.resolveEnemyRobotupAction(actor, actor)
	if actor.HP != 763 || actor.MP != 850 || action.TargetHP != 763 {
		t.Fatalf("expected robotup lower captured healing bound 263 with MP=-60, got actor=%+v action=%+v", actor, action)
	}
	if len(action.RefreshInfos) != 1 || action.RefreshInfos[0].Handle != actor.Handle {
		t.Fatalf("expected self robotup to push one combined actor refresh, got %+v", action.RefreshInfos)
	}
}

func TestSourceBattleRuntimeConfigUsesClassicDataTables(t *testing.T) {
	monsterRows, err := classicdata.FindRows(classicdata.TableMonster, "handle", "5176206909809579")
	if err != nil {
		t.Fatalf("FindRows monster error = %v", err)
	}
	if len(monsterRows) != 1 || monsterRows[0]["source_kind"] != "visible" {
		t.Fatalf("expected classicdata visible monster row, got %+v", monsterRows)
	}
	monsterConfig, ok := sourceVisibleMonsterConfigForHandle("143", "5176206909809579")
	if !ok {
		t.Fatal("expected visible monster config from classicdata")
	}
	if monsterConfig.Cell.Name != monsterRows[0]["name"] || strconv.Itoa(monsterConfig.Cell.Attack) != monsterRows[0]["attack"] {
		t.Fatalf("expected visible monster config to match classicdata row %+v, got %+v", monsterRows[0], monsterConfig.Cell)
	}

	dropRows, err := classicdata.FindDropRowsByMapID("49")
	if err != nil {
		t.Fatalf("FindDropRowsByMapID error = %v", err)
	}
	if len(dropRows) == 0 {
		t.Fatal("expected classicdata drop rows for map49")
	}
	reward, ok := sourceBattleRewardConfigForMap("49")
	if !ok {
		t.Fatal("expected map49 reward from classicdata")
	}
	if strconv.Itoa(reward.ExpDelta) != dropRows[0]["exp_delta"] || reward.Status != dropRows[0]["status"] {
		t.Fatalf("expected reward config to match classicdata row %+v, got %+v", dropRows[0], reward)
	}
}

func TestVisibleMonsterEncounterGroupsUseCapturedBattleCells(t *testing.T) {
	cases := []struct {
		mapID   string
		handle  string
		handles []string
		names   []string
	}{
		{
			mapID:   "144",
			handle:  "2766206074547838",
			handles: []string{"2762206074545916", "2764206074546810", "2766206074547838"},
			names:   []string{"武斗蛤蟆", "武斗蛤蟆", "武斗蛤蟆"},
		},
		{
			mapID:   "141",
			handle:  "4716206556987104",
			handles: []string{"4714206556986370", "4716206556987104"},
			names:   []string{"剑术蛤蟆", "剑术蛤蟆"},
		},
		{
			mapID:   "143",
			handle:  "5176206909809579",
			handles: []string{"5168206909805631", "5170206909806155", "5176206909809579"},
			names:   []string{"法术蛤蟆", "法术蛤蟆", "蛤蟆精"},
		},
		{
			mapID:   "149",
			handle:  "3218685759638239",
			handles: []string{"3220685759639165", "3218685759638239"},
			names:   []string{"蛮族弓手", "黄风二寨主"},
		},
		{
			mapID:   "155",
			handle:  "2600686416056495",
			handles: []string{"2800686416057704", "2600686416056495"},
			names:   []string{"黄风寨夫人", "黄风大寨主"},
		},
		{
			mapID:   "152",
			handle:  "2600685534049446",
			handles: []string{"2600685534049446", "3000685534050820", "2800685534050729"},
			names:   []string{"蛮族刀客", "蛮族弓手", "蛮族弓手"},
		},
		{
			mapID:   "69",
			handle:  "4261674575785819",
			handles: []string{"4261674575785819", "4255674575781235"},
			names:   []string{"白咒石怪", "白咒石怪"},
		},
		{
			mapID:   "72",
			handle:  "5321674881103452",
			handles: []string{"5319674881102907", "5317674881101168", "5321674881103452"},
			names:   []string{"晶石怪", "晶石怪", "晶石怪"},
		},
		{
			mapID:   "74",
			handle:  "9041674933916861",
			handles: []string{"9039674933915532", "9041674933916861"},
			names:   []string{"晶石怪", "晶石怪"},
		},
		{
			mapID:   "74",
			handle:  "9033674933911913",
			handles: []string{"9031674933909671", "9033674933911913"},
			names:   []string{"白咒石怪", "白咒石怪"},
		},
		{
			mapID:   "77",
			handle:  "3030675136603700",
			handles: []string{"3028675136602500", "3030675136603700"},
			names:   []string{"晶石怪", "晶石怪"},
		},
		{
			mapID:   "77",
			handle:  "3036675136607240",
			handles: []string{"3034675136606406", "3036675136607240"},
			names:   []string{"晶石怪", "晶石怪"},
		},
		{
			mapID:   "78",
			handle:  "1681675260686878",
			handles: []string{"1679675260685862", "1681675260686878"},
			names:   []string{"晶石怪", "岩化魔人"},
		},
		{
			mapID:   "78",
			handle:  "1677675260684828",
			handles: []string{"1675675260682596", "1677675260684828"},
			names:   []string{"晶石怪", "晶石怪"},
		},
		{
			mapID:   "75",
			handle:  "8130675527791587",
			handles: []string{"8110675527789273", "8130675527791587"},
			names:   []string{"晶石怪", "晶石怪"},
		},
		{
			mapID:   "75",
			handle:  "8190675527795636",
			handles: []string{"8150675527792799", "8170675527794372", "8190675527795636"},
			names:   []string{"晶石怪", "晶石怪", "晶石怪"},
		},
		{
			mapID:   "76",
			handle:  "1046675671975970",
			handles: []string{"1044675671974869", "1046675671975970"},
			names:   []string{"晶石怪", "晶石怪"},
		},
		{
			mapID:   "76",
			handle:  "1048675671977626",
			handles: []string{"1042675671973672", "1048675671977626"},
			names:   []string{"晶石怪", "巨岩魔"},
		},
		{
			mapID:   "76",
			handle:  "1040675671971889",
			handles: []string{"1038675671970511", "1040675671971889"},
			names:   []string{"晶石怪", "晶石怪"},
		},
	}

	for _, testCase := range cases {
		configs, ok := sourceVisibleMonsterConfigsForHandle(testCase.mapID, testCase.handle)
		if !ok {
			t.Fatalf("expected captured visible encounter group for %+v", testCase)
		}
		if len(configs) != len(testCase.handles) {
			t.Fatalf("expected captured visible encounter group %+v, got %+v", testCase.handles, configs)
		}
		for index, expectedHandle := range testCase.handles {
			if configs[index].Cell.Handle != expectedHandle || configs[index].Cell.Name != testCase.names[index] {
				t.Fatalf("expected captured visible encounter member %d %s/%s, got %+v", index, expectedHandle, testCase.names[index], configs[index].Cell)
			}
		}
	}
}

func TestHuangfengzhaiVisibleMonsterConfigsUseCapturedBattleCells(t *testing.T) {
	cases := []struct {
		mapID  string
		handle string
		name   string
		level  int
		maxHP  int
		maxMP  int
		label  string
	}{
		{mapID: "146", handle: "6887685480585492", name: "蛮族刀客", level: 12, maxHP: 520, maxMP: 214, label: "普通攻击"},
		{mapID: "147", handle: "7633284548716137", name: "蛮族刀客", level: 12, maxHP: 520, maxMP: 214, label: "普通攻击"},
		{mapID: "149", handle: "3218685759638239", name: "黄风二寨主", level: 19, maxHP: 1200, maxMP: 564, label: "普通攻击"},
		{mapID: "153", handle: "7494686002239485", name: "咒巫师", level: 16, maxHP: 500, maxMP: 550, label: "法术普通攻击"},
		{mapID: "155", handle: "2600686416056495", name: "黄风大寨主", level: 20, maxHP: 1500, maxMP: 564, label: "普通攻击"},
		{mapID: "155", handle: "2800686416057704", name: "黄风寨夫人", level: 20, maxHP: 1200, maxMP: 704, label: "普通攻击"},
	}

	for _, testCase := range cases {
		config, ok := sourceVisibleMonsterConfigForHandle(testCase.mapID, testCase.handle)
		if !ok {
			t.Fatalf("expected captured huangfengzhai visible monster config for %s/%s", testCase.mapID, testCase.handle)
		}
		if config.Cell.Name != testCase.name || config.Cell.Level != testCase.level || config.Cell.MaxHP != testCase.maxHP || config.Cell.MaxMP != testCase.maxMP || config.Cell.CommandLabel != testCase.label {
			t.Fatalf("expected captured huangfengzhai visible monster %+v, got %+v", testCase, config.Cell)
		}
		if config.QueueIndexTeam != 1 || config.QueueIndexEnemy != 4 {
			t.Fatalf("expected huangfengzhai visible monster queue indexes 1/4, got %+v", config)
		}
	}
}

func TestShihukuVisibleMonsterConfigsUseCapturedBattleCells(t *testing.T) {
	cases := []struct {
		mapID  string
		handle string
		name   string
		level  int
		maxHP  int
		maxMP  int
		label  string
	}{
		{mapID: "158", handle: "5837621591158929", name: "黑影", level: 25, maxHP: 2300, maxMP: 724, label: "普通攻击"},
		{mapID: "160", handle: "3824621817512450", name: "蛮虎怪", level: 25, maxHP: 3000, maxMP: 584, label: "普通攻击"},
		{mapID: "161", handle: "9333621886743795", name: "蛮虎队长", level: 28, maxHP: 3500, maxMP: 600, label: "普通攻击"},
		{mapID: "163", handle: "8094622782649492", name: "盘狮队长", level: 26, maxHP: 3300, maxMP: 440, label: "普通攻击"},
		{mapID: "165", handle: "3856623006745359", name: "黑影队长", level: 26, maxHP: 3000, maxMP: 760, label: "普通攻击"},
		{mapID: "160", handle: "3826621817513876", name: "盘狮怪", level: 25, maxHP: 2600, maxMP: 384, label: "普通攻击"},
		{mapID: "163", handle: "8088622782646450", name: "盘狮怪", level: 26, maxHP: 2800, maxMP: 400, label: "普通攻击"},
		{mapID: "167", handle: "7548622260837633", name: "盘狮怪", level: 27, maxHP: 3000, maxMP: 384, label: "普通攻击"},
		{mapID: "167", handle: "7546622260836700", name: "蛮虎怪", level: 27, maxHP: 3200, maxMP: 600, label: "普通攻击"},
		{mapID: "167", handle: "7550622260838906", name: "蚩颅王", level: 30, maxHP: 6000, maxMP: 600, label: "普通攻击"},
	}

	for _, testCase := range cases {
		config, ok := sourceVisibleMonsterConfigForHandle(testCase.mapID, testCase.handle)
		if !ok {
			t.Fatalf("expected captured shihuku visible monster config for %s/%s", testCase.mapID, testCase.handle)
		}
		if config.Cell.Name != testCase.name || config.Cell.Level != testCase.level || config.Cell.MaxHP != testCase.maxHP || config.Cell.MaxMP != testCase.maxMP || config.Cell.CommandLabel != testCase.label {
			t.Fatalf("expected captured shihuku visible monster %+v, got %+v", testCase, config.Cell)
		}
		if config.QueueIndexTeam != 1 || config.QueueIndexEnemy != 4 {
			t.Fatalf("expected shihuku visible monster queue indexes 1/4, got %+v", config)
		}
	}
}

func TestShihukuVisibleMonsterEncounterGroupsUseCapturedBattleCells(t *testing.T) {
	cases := []struct {
		mapID   string
		handle  string
		handles []string
		names   []string
	}{
		{
			mapID:   "161",
			handle:  "9329621886741609",
			handles: []string{"9329621886741609", "9331621886742375"},
			names:   []string{"蛮虎怪", "蛮虎怪"},
		},
		{
			mapID:   "161",
			handle:  "9325621886740100",
			handles: []string{"9325621886740100", "9327621886741599"},
			names:   []string{"蛮虎怪", "蛮虎怪"},
		},
		{
			mapID:   "163",
			handle:  "8088622782646450",
			handles: []string{"8088622782646450", "8090622782647529"},
			names:   []string{"盘狮怪", "盘狮怪"},
		},
		{
			mapID:   "166",
			handle:  "1959622577208401",
			handles: []string{"1959622577208401", "1961622577209280"},
			names:   []string{"盘狮怪", "盘狮怪"},
		},
		{
			mapID:   "167",
			handle:  "7542622260835182",
			handles: []string{"7542622260835182", "7544622260836750"},
			names:   []string{"盘狮怪", "盘狮怪"},
		},
		{
			// 20260616_215712_071_session_41212 packet 5712..5714:
			// map167 boss window is 蛮虎怪 + 盘狮怪 + 蚩颅王, not boss alone.
			mapID:   "167",
			handle:  "7550622260838906",
			handles: []string{"7546622260836700", "7548622260837633", "7550622260838906"},
			names:   []string{"蛮虎怪", "盘狮怪", "蚩颅王"},
		},
		{
			mapID:   "167",
			handle:  "7546622260836700",
			handles: []string{"7546622260836700", "7548622260837633", "7550622260838906"},
			names:   []string{"蛮虎怪", "盘狮怪", "蚩颅王"},
		},
	}

	for _, testCase := range cases {
		configs, ok := sourceVisibleMonsterConfigsForHandle(testCase.mapID, testCase.handle)
		if !ok {
			t.Fatalf("expected captured shihuku visible encounter group for %+v", testCase)
		}
		if len(configs) != len(testCase.handles) {
			t.Fatalf("expected captured shihuku encounter group %+v, got %+v", testCase.handles, configs)
		}
		for index, expectedHandle := range testCase.handles {
			if configs[index].Cell.Handle != expectedHandle || configs[index].Cell.Name != testCase.names[index] {
				t.Fatalf("expected captured shihuku encounter member %d %s/%s, got %+v", index, expectedHandle, testCase.names[index], configs[index].Cell)
			}
		}
	}
}

func TestNewWildBattleUsesCapturedShihukuBossEncounterGroup(t *testing.T) {
	role := session.RoleSummary{
		RoleID:      "player_21424",
		DisplayName: "恐龙抗狼1",
		Level:       35,
		SourceQuery: "human/human.swf?a=14&w8=47&",
	}
	playerBase := session.PlayerBaseData{
		PlayerID:    "player",
		RoleID:      role.RoleID,
		DisplayName: role.DisplayName,
		Level:       35,
		HP:          1555,
		MP:          424,
		MaxHP:       1555,
		MaxMP:       424,
		SourceQuery: role.SourceQuery,
		RolePhysique: &session.RolePhysique{
			Handle: role.RoleID,
			MaxHP:  1555,
			MaxMP:  424,
			PhyAtk: 500,
			PhyDef: 200,
			MgcDef: 30,
			Hit:    201,
			Dog:    115,
			Fat:    156,
		},
	}

	runtime, bundle, ok := NewWildBattle(role, playerBase, StartRequest{
		MapID:               "167",
		MapName:             "狮虎窟_10",
		SourceMonsterHandle: "7550622260838906",
	})
	if !ok || runtime == nil {
		t.Fatalf("expected shihuku boss encounter runtime, got ok=%v runtime=%+v", ok, runtime)
	}
	if len(bundle.Cells) != 4 {
		t.Fatalf("expected player plus captured boss trio, got %+v", bundle.Cells)
	}
	expected := []struct {
		handle string
		name   string
		maxHP  int
	}{
		{"7546622260836700", "蛮虎怪", 3200},
		{"7548622260837633", "盘狮怪", 3000},
		{"7550622260838906", "蚩颅王", 6000},
	}
	for index, want := range expected {
		cell := bundle.Cells[index+1]
		if cell.Handle != want.handle || cell.Name != want.name || cell.MaxHP != want.maxHP || cell.Camp != CampEnemy {
			t.Fatalf("expected shihuku boss trio member %d %+v, got %+v", index, want, cell)
		}
	}
}

func TestNewWildBattleUsesCapturedVisibleMonsterEncounterGroup(t *testing.T) {
	role := session.RoleSummary{
		RoleID:      "player_21424",
		DisplayName: "恐龙抗狼1",
		Level:       23,
		SourceQuery: "human/human.swf?a=4&w8=42&",
	}
	playerBase := session.PlayerBaseData{
		PlayerID:    "player",
		RoleID:      role.RoleID,
		DisplayName: role.DisplayName,
		Level:       23,
		HP:          1045,
		MP:          264,
		MaxHP:       1045,
		MaxMP:       264,
		SourceQuery: role.SourceQuery,
		RolePhysique: &session.RolePhysique{
			Handle: role.RoleID,
			MaxHP:  1045,
			MaxMP:  264,
			PhyAtk: 145,
			PhyDef: 158,
			MgcDef: 30,
			Hit:    201,
			Dog:    115,
			Fat:    156,
		},
	}

	runtime, bundle, ok := NewWildBattle(role, playerBase, StartRequest{
		MapID:               "143",
		MapName:             "水帘洞_13",
		SourceMonsterHandle: "5176206909809579",
	})

	if !ok || runtime == nil {
		t.Fatalf("expected visible monster group battle runtime, got ok=%v runtime=%+v", ok, runtime)
	}
	if len(bundle.Cells) != 4 {
		t.Fatalf("expected player plus captured boss encounter group, got %+v", bundle.Cells)
	}
	expectedEnemyHandles := []string{"5168206909805631", "5170206909806155", "5176206909809579"}
	for index, expectedHandle := range expectedEnemyHandles {
		cell := bundle.Cells[index+1]
		if cell.Handle != expectedHandle {
			t.Fatalf("expected captured enemy handle %s without synthetic suffix, got %+v", expectedHandle, cell)
		}
		if cell.Camp != CampEnemy {
			t.Fatalf("expected captured enemy camp for %s, got %+v", expectedHandle, cell)
		}
	}
	if bundle.Cells[1].Name != "法术蛤蟆" || bundle.Cells[3].Name != "蛤蟆精" || bundle.Cells[3].MaxHP != 1500 {
		t.Fatalf("expected captured boss group cell stats, got %+v", bundle.Cells)
	}
	if bundle.Start.EncounterLabel != "水帘洞_13 首领" {
		t.Fatalf("expected visible boss group to keep boss encounter label, got %+v", bundle.Start)
	}
}

func TestNewWildBattleUsesCapturedHuangfengzhaiVisibleMonsterEncounterGroup(t *testing.T) {
	role := session.RoleSummary{
		RoleID:      "player_huangfeng",
		DisplayName: "测试女侠",
		Level:       23,
		SourceQuery: "human/human.swf?a=4&w8=42&",
	}
	playerBase := session.PlayerBaseData{
		PlayerID:    "player",
		RoleID:      role.RoleID,
		DisplayName: role.DisplayName,
		Level:       23,
		HP:          1045,
		MP:          264,
		MaxHP:       1045,
		MaxMP:       264,
		SourceQuery: role.SourceQuery,
		RolePhysique: &session.RolePhysique{
			Handle: role.RoleID,
			MaxHP:  1045,
			MaxMP:  264,
			PhyAtk: 145,
			PhyDef: 158,
			MgcDef: 30,
			Hit:    201,
			Dog:    115,
			Fat:    156,
		},
	}

	runtime, bundle, ok := NewWildBattle(role, playerBase, StartRequest{
		MapID:               "155",
		MapName:             "黄风寨_10",
		SourceMonsterHandle: "2600686416056495",
	})

	if !ok || runtime == nil {
		t.Fatalf("expected huangfengzhai visible monster group battle runtime, got ok=%v runtime=%+v", ok, runtime)
	}
	if runtime.SourceMonsterHandle != "2600686416056495" {
		t.Fatalf("expected runtime to retain huangfengzhai source monster handle, got %+v", runtime)
	}
	if len(bundle.Cells) != 3 {
		t.Fatalf("expected player plus captured huangfengzhai boss pair, got %+v", bundle.Cells)
	}
	expectedEnemies := []struct {
		handle string
		name   string
		maxHP  int
	}{
		{handle: "2800686416057704", name: "黄风寨夫人", maxHP: 1200},
		{handle: "2600686416056495", name: "黄风大寨主", maxHP: 1500},
	}
	for index, expected := range expectedEnemies {
		cell := bundle.Cells[index+1]
		if cell.Handle != expected.handle || cell.Name != expected.name || cell.MaxHP != expected.maxHP {
			t.Fatalf("expected captured huangfengzhai enemy %+v at index %d, got %+v", expected, index, cell)
		}
		if cell.Camp != CampEnemy {
			t.Fatalf("expected huangfengzhai enemy camp for %s, got %+v", expected.handle, cell)
		}
	}
	if bundle.Start.EncounterLabel != "黄风寨_10 首领" {
		t.Fatalf("expected huangfengzhai visible boss group to keep boss encounter label, got %+v", bundle.Start)
	}
}

func TestProcessEscapeQueuesSourceActionBeforeOver(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-escape",
		Round:            1,
		Phase:            PhaseCommand,
		ActiveHandle:     "player_21424",
		nextSequence:     3,
		ConsumedSequence: map[int]bool{},
		DefendingHandles: map[string]bool{},
		StoredPower:      map[string]int{"player_21424": 2},
		Cells: []CellInfoPush{
			{
				BattleID: "battle-escape",
				Handle:   "player_21424",
				Camp:     CampTeam,
				HP:       225,
				MaxHP:    225,
				MP:       64,
				MaxMP:    64,
			},
			{
				BattleID: "battle-escape",
				Handle:   "enemy_1",
				Camp:     CampEnemy,
				HP:       120,
				MaxHP:    120,
			},
		},
	}

	result := runtime.ProcessAction(ActionRequest{
		BattleID:     "battle-escape",
		ActorHandle:  "player_21424",
		CommandID:    CommandEscape,
		TargetHandle: "player_21424",
		Round:        1,
		Sequence:     3,
	})

	if result.Over != nil || len(result.Actions) != 1 {
		t.Fatalf("expected one escape action before over, got %+v", result)
	}
	action := result.Actions[0]
	if action.CommandID != CommandEscape || action.SourceActionLabel != "escapeSuccess" || action.Damage != 0 {
		t.Fatalf("expected escapeSuccess self action, got %+v", action)
	}
	if runtime.PendingOver == nil || runtime.Phase != PhasePlaying {
		t.Fatalf("expected pending over while escape animation plays, runtime=%+v", runtime)
	}
	if runtime.StoredPower["player_21424"] != 0 {
		t.Fatalf("expected escape to clear stored power, got %d", runtime.StoredPower["player_21424"])
	}

	playOver := runtime.ProcessPlayOver(PlayOverRequest{BattleID: "battle-escape"})
	if playOver.Over == nil || playOver.Over.Winner != CampEnemy || !playOver.Over.Result.Escaped {
		t.Fatalf("expected escaped over after play over, got %+v", playOver)
	}
}

func TestEnemyHitSetsStoredPowerFromSingleHPLossPercent(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-hit-power",
		Round:            1,
		Phase:            PhaseCommand,
		ActiveHandle:     "player_21424",
		nextSequence:     1,
		ConsumedSequence: map[int]bool{},
		DefendingHandles: map[string]bool{},
		StoredPower:      map[string]int{},
		Cells: []CellInfoPush{
			{
				BattleID: "battle-hit-power",
				Handle:   "player_21424",
				Camp:     CampTeam,
				HP:       1000,
				MaxHP:    1000,
				Attack:   1,
				Hit:      100,
			},
			{
				BattleID: "battle-hit-power",
				Handle:   "enemy_1",
				Camp:     CampEnemy,
				HP:       120,
				MaxHP:    120,
				Attack:   250,
				Hit:      100,
			},
		},
	}

	result := runtime.ProcessAction(ActionRequest{
		BattleID:     "battle-hit-power",
		ActorHandle:  "player_21424",
		CommandID:    CommandNormalAttack,
		TargetHandle: "enemy_1",
		Round:        1,
		Sequence:     1,
	})

	if result.ErrorCode != "" {
		t.Fatalf("expected normal attack to resolve, got %+v", result)
	}
	start := pendingStartFor(runtime, "player_21424")
	if start == nil || start.Power != 2 || runtime.powerFor("player_21424") != 2 {
		t.Fatalf("expected 250/1000 HP loss to set stored power 2 without damage bonus, pending=%+v stored=%d", start, runtime.powerFor("player_21424"))
	}
}

func TestEnemyHitByPlayerStoresPowerAndConsumesItOnEnemyTurn(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-enemy-hit-power",
		Round:            1,
		Phase:            PhaseCommand,
		ActiveHandle:     "player_21424",
		nextSequence:     1,
		ConsumedSequence: map[int]bool{},
		DefendingHandles: map[string]bool{},
		StoredPower:      map[string]int{},
		Cells: []CellInfoPush{
			{
				BattleID: "battle-enemy-hit-power",
				Handle:   "player_21424",
				Camp:     CampTeam,
				HP:       1000,
				MaxHP:    1000,
				Attack:   250,
				Hit:      100,
			},
			{
				BattleID: "battle-enemy-hit-power",
				Handle:   "enemy_1",
				Camp:     CampEnemy,
				HP:       1000,
				MaxHP:    1000,
				Attack:   100,
				Hit:      100,
			},
		},
	}

	result := runtime.ProcessAction(ActionRequest{
		BattleID:     "battle-enemy-hit-power",
		ActorHandle:  "player_21424",
		CommandID:    CommandNormalAttack,
		TargetHandle: "enemy_1",
		Round:        1,
		Sequence:     1,
	})

	if result.ErrorCode != "" || len(result.Actions) < 2 {
		t.Fatalf("expected player action followed by enemy action, got %+v", result)
	}
	if result.Actions[0].Damage != 250 {
		t.Fatalf("expected stored power not to raise player damage, got %+v", result.Actions[0])
	}
	if result.Actions[1].Damage != 100 {
		t.Fatalf("expected enemy stored power not to raise enemy damage, got %+v", result.Actions[1])
	}
	if runtime.StoredPower["enemy_1"] != 0 {
		t.Fatalf("expected enemy stored power to clear after attacking, got %d", runtime.StoredPower["enemy_1"])
	}
}

func TestStorePowerSurvivesLighterEnemyHit(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-store-hit-power",
		Round:            1,
		Phase:            PhaseCommand,
		ActiveHandle:     "player_21424",
		nextSequence:     1,
		ConsumedSequence: map[int]bool{},
		DefendingHandles: map[string]bool{},
		StoredPower:      map[string]int{"player_21424": 1},
		Cells: []CellInfoPush{
			{
				BattleID: "battle-store-hit-power",
				Handle:   "player_21424",
				Camp:     CampTeam,
				HP:       1000,
				MaxHP:    1000,
				Attack:   1,
				Hit:      100,
			},
			{
				BattleID: "battle-store-hit-power",
				Handle:   "enemy_1",
				Camp:     CampEnemy,
				HP:       120,
				MaxHP:    120,
				Attack:   5,
				Hit:      100,
			},
		},
	}

	result := runtime.ProcessAction(ActionRequest{
		BattleID:     "battle-store-hit-power",
		ActorHandle:  "player_21424",
		CommandID:    CommandStore,
		TargetHandle: "player_21424",
		Round:        1,
		Sequence:     1,
	})

	if result.ErrorCode != "" {
		t.Fatalf("expected store to resolve, got %+v", result)
	}
	start := pendingStartFor(runtime, "player_21424")
	if start == nil || start.Power != 2 || runtime.powerFor("player_21424") != 2 {
		t.Fatalf("expected stored power 2 to survive lighter enemy hit, pending=%+v stored=%d", start, runtime.powerFor("player_21424"))
	}
}

func TestNewWildBattleUsesRoleStateAndPhysiqueForTeamCell(t *testing.T) {
	role := session.RoleSummary{
		RoleID:      "player_21424",
		DisplayName: "恐龙抗狼1",
		Level:       3,
		Exp:         130,
		SourceQuery: "human/human.swf?p=1&w8=5&",
	}
	playerBase := session.PlayerBaseData{
		PlayerID:    "player",
		RoleID:      role.RoleID,
		DisplayName: role.DisplayName,
		Level:       3,
		Exp:         130,
		HP:          225,
		MP:          64,
		MaxHP:       225,
		MaxMP:       64,
		SourceQuery: role.SourceQuery,
		RoleState: &session.RoleState{
			Handle: role.RoleID,
			HP:     225,
			MP:     64,
			Exp:    130,
			Lv:     3,
			Speed:  130,
		},
		RolePhysique: &session.RolePhysique{
			Handle: role.RoleID,
			MaxHP:  225,
			MaxMP:  64,
			PhyAtk: 47,
			PhyDef: 24,
			Hit:    104,
			Dog:    52,
			Fat:    14,
		},
	}

	runtime, bundle, ok := NewWildBattle(role, playerBase, StartRequest{MapID: "4", MapName: "涧庭村口"})

	if !ok || runtime == nil {
		t.Fatalf("expected battle runtime, got ok=%v runtime=%+v", ok, runtime)
	}
	if len(bundle.Cells) == 0 {
		t.Fatalf("expected start cells, got %+v", bundle.Cells)
	}
	if bundle.StartCommand.Power != 1 {
		t.Fatalf("expected first player command to start with one stored power, got %+v", bundle.StartCommand)
	}
	team := bundle.Cells[0]
	if team.MaxHP != 225 || team.MaxMP != 64 || team.Attack != 47 || team.Defense != 24 || team.Fat != 14 || team.Speed != 130 {
		t.Fatalf("expected team cell to mirror roleState/rolePhysique, got %+v", team)
	}
}

func TestNewWildBattlePrefersCapturedBattleSourceQueryForTeamCell(t *testing.T) {
	role := session.RoleSummary{
		RoleID:            "player_21432",
		DisplayName:       "222",
		Level:             26,
		Voc:               "游侠",
		SourceQuery:       "human/human.swf?e=6&sex=1&hr=12&co=5&m=0&n=0&",
		BattleSourceQuery: "human/human.swf?a=34&b=31&c=35&e=6&sex=1&h=12&hr=12&co=5&m=0&n=0&p=13&se=27&wr=11&w3=43&",
	}
	playerBase := session.PlayerBaseData{
		PlayerID:          "acct-cap1366655383",
		RoleID:            role.RoleID,
		DisplayName:       role.DisplayName,
		Level:             role.Level,
		Voc:               role.Voc,
		HP:                815,
		MP:                394,
		MaxHP:             815,
		MaxMP:             394,
		SourceQuery:       role.SourceQuery,
		BattleSourceQuery: role.BattleSourceQuery,
	}

	_, bundle, ok := NewWildBattle(role, playerBase, StartRequest{MapID: "4", MapName: "涧庭村口"})

	if !ok || len(bundle.Cells) == 0 {
		t.Fatalf("expected battle cells, ok=%v bundle=%+v", ok, bundle)
	}
	if bundle.Cells[0].DisplayURL != role.BattleSourceQuery || !strings.Contains(bundle.Cells[0].DisplayURL, "w3=43") {
		t.Fatalf("expected captured battle display url, got %+v", bundle.Cells[0])
	}
}

func TestNewWildBattleKeepsCapturedBattleSourceQueryForWarriorLeader(t *testing.T) {
	role := session.RoleSummary{
		RoleID:            "player_21424",
		DisplayName:       "111",
		Level:             35,
		Voc:               "战士",
		SourceQuery:       "human/human.swf?n=0&sex=1&co=5&hr=32&e=14&m=5&h=8&a=4&wr=7&w8=42&c=10&p=8&b=5&se=4&",
		BattleSourceQuery: "human/human.swf?=&a=14&w8=47&b=16&c=39&e=6&sex=1&h=30&hr=12&co=5&m=0&n=0&p=18&se=29&wr=15&",
	}
	playerBase := session.PlayerBaseData{
		PlayerID:          "acct-1150045313",
		RoleID:            role.RoleID,
		DisplayName:       role.DisplayName,
		Level:             role.Level,
		Voc:               role.Voc,
		HP:                1425,
		MP:                296,
		MaxHP:             1555,
		MaxMP:             424,
		SourceQuery:       role.SourceQuery,
		BattleSourceQuery: role.BattleSourceQuery,
	}

	_, bundle, ok := NewWildBattle(role, playerBase, StartRequest{MapID: "4", MapName: "涧庭村口"})

	if !ok || len(bundle.Cells) == 0 {
		t.Fatalf("expected battle cells, ok=%v bundle=%+v", ok, bundle)
	}
	if bundle.Cells[0].DisplayURL != role.BattleSourceQuery || !strings.Contains(bundle.Cells[0].DisplayURL, "w8=47") {
		t.Fatalf("expected captured 111 battle display url, got %+v", bundle.Cells[0])
	}
}

func TestNewWildBattleSupportsCapturedBambooMaps(t *testing.T) {
	role := session.RoleSummary{
		RoleID:      "player_21424",
		DisplayName: "恐龙抗狼1",
		Level:       3,
		Exp:         130,
	}
	playerBase := session.PlayerBaseData{
		PlayerID:    "player",
		RoleID:      role.RoleID,
		DisplayName: role.DisplayName,
		Level:       3,
		Exp:         130,
	}

	cases := []struct {
		mapID      string
		mapName    string
		enemyName  string
		displayURL string
		queueTeam  int
		queueEnemy int
		roll       int
	}{
		{mapID: "84", mapName: "竹林_1", enemyName: "绿甲螳螂", displayURL: "monstermap/greenmantis.swf", queueTeam: 1, queueEnemy: 4},
		{mapID: "85", mapName: "竹林_2", enemyName: "绿甲螳螂", displayURL: "monstermap/greenmantis.swf", queueTeam: 1, queueEnemy: 4},
		{mapID: "86", mapName: "竹林_3", enemyName: "小竹妖", displayURL: "monstermap/bambooboy.swf", queueTeam: 1, queueEnemy: 4},
		{mapID: "87", mapName: "竹林_4", enemyName: "跳跳竹", displayURL: "monstermap/jumpboo.swf", queueTeam: 1, queueEnemy: 4},
		{mapID: "88", mapName: "竹林_5", enemyName: "刀手螳螂", displayURL: "monstermap/kinfemantis.swf", queueTeam: 1, queueEnemy: 4},
		{mapID: "90", mapName: "竹林_7", enemyName: "小竹妖", displayURL: "monstermap/bambooboy.swf", queueTeam: 1, queueEnemy: 4},
		{mapID: "97", mapName: "竹林_10", enemyName: "小竹妖", displayURL: "monstermap/bambooboy.swf", queueTeam: 1, queueEnemy: 4, roll: 5},
	}

	for _, testCase := range cases {
		restore := useSourceEncounterRoll(func(int) int { return testCase.roll })
		runtime, bundle, ok := NewWildBattle(role, playerBase, StartRequest{MapID: testCase.mapID, MapName: testCase.mapName})
		restore()
		if !ok || runtime == nil {
			t.Fatalf("expected captured bamboo map %s to start battle, got ok=%v runtime=%+v", testCase.mapID, ok, runtime)
		}
		if bundle.Start.MapID != testCase.mapID || bundle.Start.QueueIndexTeam != testCase.queueTeam || bundle.Start.QueueIndexEnemy != testCase.queueEnemy {
			t.Fatalf("expected bamboo map %s queue indexes %d/%d, got %+v", testCase.mapID, testCase.queueTeam, testCase.queueEnemy, bundle.Start)
		}
		if len(bundle.Cells) != 2 {
			t.Fatalf("expected player and enemy cells for map %s, got %+v", testCase.mapID, bundle.Cells)
		}
		enemy := bundle.Cells[1]
		if enemy.Name != testCase.enemyName || enemy.DisplayURL != testCase.displayURL {
			t.Fatalf("expected bamboo map %s enemy %s/%s, got %+v", testCase.mapID, testCase.enemyName, testCase.displayURL, enemy)
		}
	}
}

func TestNewWildBattleSupportsCaptureBackedYaozhisenMaps(t *testing.T) {
	role := session.RoleSummary{
		RoleID:      "player_yaozhisen",
		DisplayName: "测试女侠",
		Level:       35,
		Exp:         10000,
	}
	playerBase := session.PlayerBaseData{
		PlayerID:    "player",
		RoleID:      role.RoleID,
		DisplayName: role.DisplayName,
		Level:       role.Level,
		Exp:         role.Exp,
	}

	cases := []struct {
		mapID      string
		mapName    string
		enemyName  string
		displayURL string
		cellCount  int
		roll       int
	}{
		{mapID: "192", mapName: "妖之森_1", enemyName: "机木斧兵", displayURL: "monstermap/robotax.swf", cellCount: 2, roll: 0},
		{mapID: "193", mapName: "妖之森_2", enemyName: "机木斧兵", displayURL: "monstermap/robotax.swf", cellCount: 2, roll: 0},
		{mapID: "196", mapName: "妖之森_5", enemyName: "机木斧兵", displayURL: "monstermap/robotax.swf", cellCount: 2, roll: 0},
		{mapID: "198", mapName: "妖之森_7", enemyName: "机木锥兵", displayURL: "monstermap/robotawl.swf", cellCount: 2, roll: 0},
		{mapID: "199", mapName: "妖之森_8", enemyName: "机木玄师", displayURL: "monstermap/robothyun.swf", cellCount: 3, roll: 0},
		{mapID: "200", mapName: "妖之森_9", enemyName: "机木锥兵", displayURL: "monstermap/robotawl.swf", cellCount: 2, roll: 6},
		{mapID: "201", mapName: "妖之森_10", enemyName: "机木玄师", displayURL: "monstermap/robothyun.swf", cellCount: 2, roll: 227},
		{mapID: "202", mapName: "妖之森_11", enemyName: "机木锥兵", displayURL: "monstermap/robotawl.swf", cellCount: 3, roll: 0},
		{mapID: "204", mapName: "妖之森_13", enemyName: "机木斧兵", displayURL: "monstermap/robotax.swf", cellCount: 3, roll: 0},
		{mapID: "205", mapName: "妖之森_14", enemyName: "机木斧兵", displayURL: "monstermap/robotax.swf", cellCount: 4, roll: 0},
	}

	for _, testCase := range cases {
		restore := useSourceEncounterRoll(func(int) int { return testCase.roll })
		runtime, bundle, ok := NewWildBattle(role, playerBase, StartRequest{
			MapID:   testCase.mapID,
			MapName: testCase.mapName,
		})
		restore()
		if !ok || runtime == nil || len(bundle.Cells) != testCase.cellCount {
			t.Fatalf("expected capture-backed map %s to start with %d cells, ok=%v runtime=%+v bundle=%+v", testCase.mapID, testCase.cellCount, ok, runtime, bundle)
		}
		enemy := bundle.Cells[1]
		if enemy.Name != testCase.enemyName || enemy.DisplayURL != testCase.displayURL {
			t.Fatalf("expected map %s enemy %s/%s, got %+v", testCase.mapID, testCase.enemyName, testCase.displayURL, enemy)
		}
	}

	restore := useSourceEncounterRoll(func(int) int { return 6 })
	_, map196BossBundle, map196BossOK := NewWildBattle(role, playerBase, StartRequest{MapID: "196", MapName: "妖之森_5"})
	restore()
	if !map196BossOK || len(map196BossBundle.Cells) != 5 || map196BossBundle.Cells[1].Name != "机木玄师" || map196BossBundle.Cells[3].Name != "机木妖帅" || map196BossBundle.Cells[4].Name != "机木锥兵" {
		t.Fatalf("expected captured map196 boss composition, got %+v", map196BossBundle.Cells)
	}

	restore = useSourceEncounterRoll(func(int) int { return 0 })
	_, map200MixedBundle, map200MixedOK := NewWildBattle(role, playerBase, StartRequest{MapID: "200", MapName: "妖之森_9"})
	restore()
	if !map200MixedOK || len(map200MixedBundle.Cells) != 3 || map200MixedBundle.Cells[1].Name != "机木斧兵" || map200MixedBundle.Cells[2].Name != "机木锥兵" {
		t.Fatalf("expected captured map200 mixed composition, got %+v", map200MixedBundle.Cells)
	}

	restore = useSourceEncounterRoll(func(int) int { return 0 })
	_, map201DoubleBundle, map201DoubleOK := NewWildBattle(role, playerBase, StartRequest{MapID: "201", MapName: "妖之森_10"})
	restore()
	if !map201DoubleOK || len(map201DoubleBundle.Cells) != 3 {
		t.Fatalf("expected captured map201 double mystic candidate, got %+v", map201DoubleBundle.Cells)
	}
}

func TestNewWildBattleUsesCapturedPlainEnemyStats(t *testing.T) {
	rolls := []int{0, 0, 31, 426}
	rollIndex := 0
	defer useSourceEncounterRoll(func(int) int {
		roll := rolls[rollIndex]
		rollIndex++
		return roll
	})()

	role := session.RoleSummary{
		RoleID:      "player_21424",
		DisplayName: "恐龙抗狼1",
		Level:       16,
		Exp:         1000,
	}
	playerBase := session.PlayerBaseData{
		PlayerID:    "player",
		RoleID:      role.RoleID,
		DisplayName: role.DisplayName,
		Level:       16,
		Exp:         1000,
	}

	cases := []struct {
		mapID      string
		mapName    string
		enemyName  string
		displayURL string
		level      int
		maxHP      int
		maxMP      int
		cellCount  int
	}{
		{mapID: "11", mapName: "涧庭道_1", enemyName: "蟾蜍", displayURL: "monstermap/toad.swf", level: 1, maxHP: 15, maxMP: 9, cellCount: 2},
		{mapID: "37", mapName: "平原_6", enemyName: "花妖", displayURL: "monstermap/huayao.swf", level: 10, maxHP: 230, maxMP: 150, cellCount: 2},
		{mapID: "49", mapName: "平原_15", enemyName: "盗贼", displayURL: "monstermap/robber.swf", level: 15, maxHP: 445, maxMP: 194, cellCount: 3},
		{mapID: "52", mapName: "平原_18", enemyName: "盗贼", displayURL: "monstermap/robber.swf", level: 18, maxHP: 470, maxMP: 220, cellCount: 2},
	}

	for _, testCase := range cases {
		_, bundle, ok := NewWildBattle(role, playerBase, StartRequest{MapID: testCase.mapID, MapName: testCase.mapName})
		if !ok || len(bundle.Cells) != testCase.cellCount {
			t.Fatalf("expected captured plain map %s to start battle, got ok=%v bundle=%+v", testCase.mapID, ok, bundle)
		}
		if bundle.Start.QueueIndexTeam != 1 || bundle.Start.QueueIndexEnemy != 4 {
			t.Fatalf("expected captured map %s queue indexes 1/4, got %+v", testCase.mapID, bundle.Start)
		}
		enemy := bundle.Cells[1]
		if enemy.Name != testCase.enemyName || enemy.DisplayURL != testCase.displayURL || enemy.Level != testCase.level || enemy.MaxHP != testCase.maxHP || enemy.HP != testCase.maxHP || enemy.MaxMP != testCase.maxMP || enemy.MP != testCase.maxMP {
			t.Fatalf("expected captured enemy stats for map %s, got %+v", testCase.mapID, enemy)
		}
	}
}

func TestNewWildBattleUsesExpandedCapturedPlainEnemyStats(t *testing.T) {
	defer useSourceEncounterRoll(func(int) int { return 0 })()

	role := session.RoleSummary{
		RoleID:      "player_21424",
		DisplayName: "恐龙抗狼1",
		Level:       16,
		Exp:         1000,
	}
	playerBase := session.PlayerBaseData{
		PlayerID:    "player",
		RoleID:      role.RoleID,
		DisplayName: role.DisplayName,
		Level:       16,
		Exp:         1000,
	}

	cases := []struct {
		mapID      string
		mapName    string
		enemyName  string
		displayURL string
		level      int
		maxHP      int
		maxMP      int
		cellCount  int
	}{
		{mapID: "34", mapName: "平原_3", enemyName: "爆骨猪", displayURL: "monstermap/bomepig.swf", level: 12, maxHP: 260, maxMP: 155, cellCount: 2},
		{mapID: "35", mapName: "平原_4", enemyName: "爆骨猪", displayURL: "monstermap/bomepig.swf", level: 13, maxHP: 295, maxMP: 162, cellCount: 2},
		{mapID: "36", mapName: "平原_5", enemyName: "爆骨猪", displayURL: "monstermap/bomepig.swf", level: 13, maxHP: 280, maxMP: 145, cellCount: 2},
		{mapID: "39", mapName: "平原_8", enemyName: "尖刀暴牙", displayURL: "monstermap/jdby.swf", level: 12, maxHP: 280, maxMP: 130, cellCount: 2},
		{mapID: "40", mapName: "平原_9", enemyName: "尖刀暴牙", displayURL: "monstermap/jdby.swf", level: 12, maxHP: 280, maxMP: 130, cellCount: 2},
		{mapID: "41", mapName: "平原_10", enemyName: "牙菇", displayURL: "monstermap/yagu.swf", level: 10, maxHP: 250, maxMP: 190, cellCount: 2},
		{mapID: "43", mapName: "平原_12", enemyName: "刺鸟", displayURL: "monstermap/swordbird.swf", level: 11, maxHP: 230, maxMP: 160, cellCount: 2},
		{mapID: "44", mapName: "平原_13", enemyName: "尖刀暴牙", displayURL: "monstermap/jdby.swf", level: 12, maxHP: 250, maxMP: 130, cellCount: 2},
		{mapID: "48", mapName: "平原_14", enemyName: "盗贼", displayURL: "monstermap/robber.swf", level: 14, maxHP: 330, maxMP: 210, cellCount: 2},
		{mapID: "50", mapName: "平原_16", enemyName: "巡路小鬼", displayURL: "monstermap/lilghost.swf", level: 17, maxHP: 545, maxMP: 184, cellCount: 2},
		{mapID: "51", mapName: "平原_17", enemyName: "盗贼", displayURL: "monstermap/robber.swf", level: 18, maxHP: 470, maxMP: 220, cellCount: 2},
	}

	for _, testCase := range cases {
		_, bundle, ok := NewWildBattle(role, playerBase, StartRequest{MapID: testCase.mapID, MapName: testCase.mapName})
		if !ok || len(bundle.Cells) != testCase.cellCount {
			t.Fatalf("expected captured plain map %s to start battle, got ok=%v bundle=%+v", testCase.mapID, ok, bundle)
		}
		if bundle.Start.QueueIndexTeam != 1 || bundle.Start.QueueIndexEnemy != 4 {
			t.Fatalf("expected captured map %s queue indexes 1/4, got %+v", testCase.mapID, bundle.Start)
		}
		enemy := bundle.Cells[1]
		if enemy.Name != testCase.enemyName || enemy.DisplayURL != testCase.displayURL || enemy.Level != testCase.level || enemy.MaxHP != testCase.maxHP || enemy.HP != testCase.maxHP || enemy.MaxMP != testCase.maxMP || enemy.MP != testCase.maxMP {
			t.Fatalf("expected captured enemy stats for map %s, got %+v", testCase.mapID, enemy)
		}
	}
}

func TestNewWildBattleSupportsCaptureBackedGrasslandMaps(t *testing.T) {
	defer useSourceEncounterRoll(func(int) int { return 0 })()

	role := session.RoleSummary{
		RoleID:      "player_grassland",
		DisplayName: "测试女侠",
		Level:       42,
		Exp:         10000,
	}
	playerBase := session.PlayerBaseData{
		PlayerID:    "player",
		RoleID:      role.RoleID,
		DisplayName: role.DisplayName,
		Level:       role.Level,
		Exp:         role.Exp,
	}

	testCases := []struct {
		mapID      string
		mapName    string
		enemyName  string
		displayURL string
		level      int
		maxHP      int
		maxMP      int
		attack     int
	}{
		{mapID: "210", mapName: "草坝_4", enemyName: "草刺槐", displayURL: "monstermap/glassyx.swf", level: 38, maxHP: 1530, maxMP: 1034, attack: 264},
		{mapID: "216", mapName: "草坝_11", enemyName: "草球蔓", displayURL: "monstermap/glassss.swf", level: 40, maxHP: 1357, maxMP: 1500, attack: 281},
		{mapID: "222", mapName: "草坝_16", enemyName: "草球蔓", displayURL: "monstermap/glassss.swf", level: 42, maxHP: 1450, maxMP: 1518, attack: 290},
	}

	for _, testCase := range testCases {
		runtime, bundle, ok := NewWildBattle(role, playerBase, StartRequest{MapID: testCase.mapID, MapName: testCase.mapName})
		if !ok || runtime == nil || len(bundle.Cells) < 2 {
			t.Fatalf("expected capture-backed grassland map %s to start battle, got ok=%v runtime=%+v bundle=%+v", testCase.mapID, ok, runtime, bundle)
		}
		enemy := bundle.Cells[1]
		if enemy.Name != testCase.enemyName || enemy.DisplayURL != testCase.displayURL || enemy.Level != testCase.level || enemy.MaxHP != testCase.maxHP || enemy.HP != testCase.maxHP || enemy.MaxMP != testCase.maxMP || enemy.MP != testCase.maxMP || enemy.Attack != testCase.attack {
			t.Fatalf("expected capture-backed grassland enemy stats for map %s, got %+v", testCase.mapID, enemy)
		}
	}
}

func TestNewWildBattleSelectsCapturedEncounterWeightBoundaries(t *testing.T) {
	role := session.RoleSummary{
		RoleID:      "player_21424",
		DisplayName: "恐龙抗狼1",
		Level:       16,
		Exp:         1000,
	}
	playerBase := session.PlayerBaseData{
		PlayerID:    "player",
		RoleID:      role.RoleID,
		DisplayName: role.DisplayName,
		Level:       16,
		Exp:         1000,
	}

	cases := []struct {
		mapID          string
		mapName        string
		roll           int
		handle         string
		enemyName      string
		level          int
		maxHP          int
		maxMP          int
		encounterLabel string
		enemyCount     int
	}{
		{mapID: "36", mapName: "平原_5", roll: 123, handle: "6093598288385864", enemyName: "爆骨猪", level: 13, maxHP: 265, maxMP: 162, encounterLabel: "平原_5 暗雷", enemyCount: 1},
		{mapID: "40", mapName: "平原_9", roll: 4392, handle: "6132760366794317", enemyName: "刺鸟", level: 11, maxHP: 230, maxMP: 160, encounterLabel: "平原_9 暗雷", enemyCount: 1},
		{mapID: "50", mapName: "平原_16", roll: 50, enemyName: "巡路小鬼", level: 16, maxHP: 520, maxMP: 160, encounterLabel: "平原_16 暗雷", enemyCount: 2},
		{mapID: "52", mapName: "平原_18", roll: 0, handle: "7014979944725157", enemyName: "巡路小鬼", level: 18, maxHP: 545, maxMP: 184, encounterLabel: "平原_18 暗雷", enemyCount: 1},
		{mapID: "52", mapName: "平原_18", roll: 836, handle: "1478600550966619", enemyName: "单刀狼人", level: 21, maxHP: 2500, maxMP: 334, encounterLabel: "平原_18 首领", enemyCount: 4},
		{mapID: "173", mapName: "沼泽_11", roll: 304, handle: "2500770180184588", enemyName: "玄龟兽", level: 28, maxHP: 1221, maxMP: 200, encounterLabel: "沼泽_11 暗雷", enemyCount: 3},
		{mapID: "201", mapName: "妖之森_10", roll: 227, handle: "capture-201-robothyun-lv34", enemyName: "机木玄师", level: 34, maxHP: 1189, maxMP: 910, encounterLabel: "妖之森_10 暗雷", enemyCount: 1},
	}

	for _, testCase := range cases {
		restore := useSourceEncounterRoll(func(int) int { return testCase.roll })
		_, bundle, ok := NewWildBattle(role, playerBase, StartRequest{
			MapID:   testCase.mapID,
			MapName: testCase.mapName,
		})
		restore()
		if !ok || len(bundle.Cells) != testCase.enemyCount+1 {
			t.Fatalf("expected weighted captured candidate for map %s roll %d, got ok=%v bundle=%+v", testCase.mapID, testCase.roll, ok, bundle)
		}
		if bundle.Start.EncounterLabel != testCase.encounterLabel {
			t.Fatalf("expected encounter label %s, got %+v", testCase.encounterLabel, bundle.Start)
		}
		enemy := bundle.Cells[1]
		if enemy.Name != testCase.enemyName || enemy.Level != testCase.level || enemy.MaxHP != testCase.maxHP || enemy.MaxMP != testCase.maxMP {
			t.Fatalf("expected weighted candidate enemy for map %s roll %d, got %+v", testCase.mapID, testCase.roll, enemy)
		}
		if testCase.handle != "" && enemy.Handle != testCase.handle {
			t.Fatalf("expected source handle %s for map %s roll %d, got %+v", testCase.handle, testCase.mapID, testCase.roll, enemy)
		}
		if testCase.mapID == "52" && testCase.roll == 836 {
			names := []string{bundle.Cells[1].Name, bundle.Cells[2].Name, bundle.Cells[3].Name, bundle.Cells[4].Name}
			if names[0] != "单刀狼人" || names[1] != "盗贼" || names[2] != "盗贼" || names[3] != "盗贼" {
				t.Fatalf("expected captured 平原_18 boss encounter 单刀狼人 + 3 盗贼, got %+v", names)
			}
		}
	}
}

func TestNewWildBattleRandomizesCapturedNormalEnemyCountButKeepsBossFixed(t *testing.T) {
	role := session.RoleSummary{
		RoleID:      "player_21424",
		DisplayName: "恐龙抗狼1",
		Level:       16,
		Exp:         1000,
	}
	playerBase := session.PlayerBaseData{
		PlayerID:    "player",
		RoleID:      role.RoleID,
		DisplayName: role.DisplayName,
		Level:       16,
		Exp:         1000,
	}

	restore := useSourceEncounterRoll(func(maxExclusive int) int {
		return 0
	})
	_, singleBundle, singleOK := NewWildBattle(role, playerBase, StartRequest{MapID: "50", MapName: "平原_16", StageFocusX: 0})
	restore()
	if !singleOK || len(singleBundle.Cells) != 2 {
		t.Fatalf("expected random low roll to create one normal enemy plus player, got ok=%v bundle=%+v", singleOK, singleBundle)
	}

	restore = useSourceEncounterRoll(func(maxExclusive int) int {
		return maxExclusive - 1
	})
	_, doubleBundle, doubleOK := NewWildBattle(role, playerBase, StartRequest{MapID: "50", MapName: "平原_16", StageFocusX: 0})
	restore()
	if !doubleOK || len(doubleBundle.Cells) != 3 {
		t.Fatalf("expected random high roll to create two normal enemies plus player, got ok=%v bundle=%+v", doubleOK, doubleBundle)
	}

	restore = useSourceEncounterRoll(func(maxExclusive int) int {
		return maxExclusive - 1
	})
	_, bossBundle, bossOK := NewWildBattle(role, playerBase, StartRequest{MapID: "52", MapName: "平原_18", StageFocusX: 1600})
	restore()
	if !bossOK || len(bossBundle.Cells) != 5 {
		t.Fatalf("expected boss encounter count to stay fixed at four enemies plus player, got ok=%v bundle=%+v", bossOK, bossBundle)
	}
}

func TestNewWildBattleUsesCapturedVisibleMonsterHandle(t *testing.T) {
	role := session.RoleSummary{
		RoleID:      "player_21424",
		DisplayName: "恐龙抗狼1",
		Level:       23,
		MapID:       143,
		SourceQuery: "human/human.swf?w1=1&",
	}
	playerBase := session.PlayerBaseData{
		RoleID:       role.RoleID,
		DisplayName:  role.DisplayName,
		Level:        role.Level,
		MapID:        143,
		SourceQuery:  role.SourceQuery,
		MaxHP:        1045,
		HP:           1045,
		MaxMP:        264,
		MP:           264,
		RolePhysique: &session.RolePhysique{PhyAtk: 145, PhyDef: 122},
	}

	runtime, bundle, ok := NewWildBattle(role, playerBase, StartRequest{
		MapID:               "143",
		MapName:             "水帘洞_10",
		StageFocusX:         2070,
		SourceMonsterHandle: "5176206909809579",
	})
	if !ok || runtime == nil {
		t.Fatal("expected captured visible monster battle to start")
	}
	if runtime.SourceMonsterHandle != "5176206909809579" {
		t.Fatalf("expected runtime to retain visible monster handle, got %+v", runtime)
	}
	if bundle.Start.EncounterLabel != "水帘洞_10 首领" {
		t.Fatalf("expected visible boss encounter label, got %q", bundle.Start.EncounterLabel)
	}
	if len(bundle.Cells) != 4 {
		t.Fatalf("expected player plus captured visible boss group cells, got %+v", bundle.Cells)
	}
	if bundle.Cells[1].Handle != "5168206909805631" || bundle.Cells[2].Handle != "5170206909806155" {
		t.Fatalf("expected captured cracktoad helper cells, got %+v", bundle.Cells)
	}
	enemy := bundle.Cells[3]
	if enemy.Handle != "5176206909809579" || enemy.Name != "蛤蟆精" {
		t.Fatalf("expected captured cracktoad enemy, got %+v", enemy)
	}
	if enemy.MaxHP != 1500 || enemy.MaxMP != 564 || enemy.DisplayURL != "monstermap/cracktoad.swf" {
		t.Fatalf("expected captured cracktoad stats, got %+v", enemy)
	}
}

func TestNewWildBattleBattleIDChangesBetweenStarts(t *testing.T) {
	role := session.RoleSummary{
		RoleID:      "player_21424",
		DisplayName: "恐龙抗狼1",
		Level:       23,
		MapID:       144,
		SourceQuery: "human/human.swf?w1=1&",
	}
	playerBase := session.PlayerBaseData{
		RoleID:       role.RoleID,
		DisplayName:  role.DisplayName,
		Level:        role.Level,
		MapID:        144,
		SourceQuery:  role.SourceQuery,
		MaxHP:        1045,
		HP:           1045,
		MaxMP:        264,
		MP:           264,
		RolePhysique: &session.RolePhysique{PhyAtk: 145, PhyDef: 122, Hit: 100},
	}
	request := StartRequest{
		MapID:               "144",
		MapName:             "水帘洞_5",
		StageFocusX:         2070,
		SourceMonsterHandle: "2762206074545916",
	}

	_, firstBundle, firstOK := NewWildBattle(role, playerBase, request)
	_, secondBundle, secondOK := NewWildBattle(role, playerBase, request)
	if !firstOK || !secondOK {
		t.Fatalf("expected both captured visible monster battles to start, first=%v second=%v", firstOK, secondOK)
	}
	if firstBundle.Start.BattleID == secondBundle.Start.BattleID {
		t.Fatalf("expected battle id to vary between repeated visible monster starts, got %q", firstBundle.Start.BattleID)
	}
	if firstBundle.Cells[1].Handle != secondBundle.Cells[1].Handle {
		t.Fatalf("expected captured visible monster group to stay source-backed while battle id changes, first=%+v second=%+v", firstBundle.Cells, secondBundle.Cells)
	}
}

func TestProcessActionQueuesAllLivingCapturedEnemies(t *testing.T) {
	defer useSourceEncounterRoll(func(maxExclusive int) int {
		return maxExclusive - 1
	})()

	role := session.RoleSummary{
		RoleID:      "player_21424",
		DisplayName: "恐龙抗狼1",
		Level:       20,
		Exp:         1000,
	}
	playerBase := session.PlayerBaseData{
		PlayerID:    "player",
		RoleID:      role.RoleID,
		DisplayName: role.DisplayName,
		Level:       20,
		Exp:         1000,
		HP:          2000,
		MP:          300,
		MaxHP:       2000,
		MaxMP:       300,
		RoleState: &session.RoleState{
			Handle: role.RoleID,
			HP:     2000,
			MP:     300,
			Lv:     20,
			Speed:  130,
		},
		RolePhysique: &session.RolePhysique{
			Handle: role.RoleID,
			MaxHP:  2000,
			MaxMP:  300,
			PhyAtk: 100,
			PhyDef: 0,
			Hit:    100,
			Dog:    0,
			Fat:    0,
		},
	}

	runtime, bundle, ok := NewWildBattle(role, playerBase, StartRequest{MapID: "52", MapName: "平原_18", StageFocusX: 1600})
	if !ok || runtime == nil || len(bundle.Cells) != 5 {
		t.Fatalf("expected captured four-enemy 平原_18 battle, got ok=%v bundle=%+v", ok, bundle)
	}

	result := runtime.ProcessAction(ActionRequest{
		BattleID:     bundle.Start.BattleID,
		ActorHandle:  role.RoleID,
		CommandID:    CommandNormalAttack,
		TargetHandle: bundle.Cells[1].Handle,
		Round:        1,
		Sequence:     1,
	})
	if result.ErrorCode != "" {
		t.Fatalf("expected player action to be accepted, got %+v", result)
	}
	if len(result.Actions) != 5 {
		t.Fatalf("expected player action plus four living enemy actions, got %+v", result.Actions)
	}
	for index, action := range result.Actions[1:] {
		if action.ActorHandle != bundle.Cells[index+1].Handle || action.TargetHandle != role.RoleID {
			t.Fatalf("expected enemy action %d from captured enemy slot, got %+v", index, action)
		}
	}
}

func TestNewWildBattleRejectsUncapturedWildMaps(t *testing.T) {
	role := session.RoleSummary{
		RoleID:      "player_21424",
		DisplayName: "恐龙抗狼1",
		Level:       3,
		Exp:         130,
	}
	playerBase := session.PlayerBaseData{
		PlayerID:    "player",
		RoleID:      role.RoleID,
		DisplayName: role.DisplayName,
		Level:       3,
		Exp:         130,
	}

	cases := []StartRequest{
		{MapID: "9", MapName: "云隐山道_5"},
		{MapID: "89", MapName: "竹林_6"},
		{MapID: "91", MapName: "竹林_8"},
		{MapID: "194", MapName: "妖之森_3"},
		{MapID: "195", MapName: "妖之森_4"},
		{MapID: "197", MapName: "妖之森_6"},
		{MapID: "203", MapName: "雷兽神坛"},
	}

	for _, request := range cases {
		runtime, bundle, ok := NewWildBattle(role, playerBase, request)
		if ok {
			t.Fatalf("expected map %s to reject wild battle, got bundle=%+v runtime=%+v", request.MapID, bundle, runtime)
		}
		if runtime != nil {
			t.Fatalf("expected runtime to stay nil for map %s, got %+v", request.MapID, runtime)
		}
	}
}

func TestResolveAttackCriticalFatDoublesDamageAndSendsSourceStateCode(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-critical",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
	}
	actor := &CellInfoPush{
		Handle:       "player_21424",
		Camp:         CampTeam,
		Attack:       47,
		Fat:          100,
		CommandLabel: "普通攻击",
	}
	target := &CellInfoPush{
		Handle:  "2429459987213640",
		Camp:    CampEnemy,
		MaxHP:   120,
		HP:      120,
		Defense: 0,
	}

	action := runtime.resolveAttack(actor, target, CommandNormalAttack)

	if action.Damage != 94 || action.TargetHP != 26 {
		t.Fatalf("expected critical to double base damage 47 without stored power damage bonus, got %+v", action)
	}
	if action.TargetActionState != "fat" || action.TargetActionStateCode != "2" {
		t.Fatalf("expected source fat target action state code 2, got %+v", action)
	}
}

func TestResolveAttackNormalUsesSourceStateCodeZero(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-normal",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
	}
	actor := &CellInfoPush{
		Handle:       "player_21424",
		Camp:         CampTeam,
		Attack:       47,
		Fat:          0,
		CommandLabel: "普通攻击",
	}
	target := &CellInfoPush{
		Handle:  "2429459987213640",
		Camp:    CampEnemy,
		MaxHP:   120,
		HP:      120,
		Defense: 0,
	}

	action := runtime.resolveAttack(actor, target, CommandNormalAttack)

	if action.Damage != 47 || action.TargetHP != 73 {
		t.Fatalf("expected normal damage to use base damage 47 without stored power damage bonus, got %+v", action)
	}
	if action.TargetActionState != "normal" || action.TargetActionStateCode != "0" {
		t.Fatalf("expected source normal target action state code 0, got %+v", action)
	}
}

func TestResolveAttackDoesNotApplyStoredPowerDamageBonus(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-power-damage",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
		StoredPower:      map[string]int{"player_21424": 2},
	}
	actor := &CellInfoPush{
		Handle: "player_21424",
		Camp:   CampTeam,
		Attack: 47,
	}
	target := &CellInfoPush{
		Handle: "enemy_1",
		Camp:   CampEnemy,
		MaxHP:  120,
		HP:     120,
	}

	action := runtime.resolveAttack(actor, target, CommandNormalAttack)

	if action.Damage != 47 || action.TargetHP != 73 {
		t.Fatalf("expected stored power to leave normal damage unchanged, got %+v", action)
	}
}

func TestResolveAttackDodgePreventsDamageAndFat(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-dodge",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
	}
	actor := &CellInfoPush{
		Handle:       "player_21424",
		Camp:         CampTeam,
		Attack:       47,
		Hit:          0,
		Fat:          100,
		CommandLabel: "普通攻击",
	}
	target := &CellInfoPush{
		Handle:  "2429459987213640",
		Camp:    CampEnemy,
		MaxHP:   120,
		HP:      120,
		Defense: 0,
		Dog:     1,
	}

	action := runtime.resolveAttack(actor, target, CommandNormalAttack)

	if action.Damage != 0 || action.TargetHP != 120 || target.HP != 120 {
		t.Fatalf("expected dodge to prevent damage before fat, got action=%+v target=%+v", action, target)
	}
	if action.TargetActionState != "dog" || action.TargetActionStateCode != "1" {
		t.Fatalf("expected source dog state code 1, got %+v", action)
	}
}

func TestBattleChanceFormulaMatchesCapturedBaselineScale(t *testing.T) {
	if got := battleDodgeChancePercent(100, 50); got != 8 {
		t.Fatalf("expected baseline hit100/dog50 dodge chance 8%%, got %d", got)
	}
	if got := battleCriticalChancePercent(31, 50); got != 16 {
		t.Fatalf("expected captured STR41/AGI9 fat31 critical chance 16%%, got %d", got)
	}
	if got := battleCriticalChancePercent(7, 50); got != 5 {
		t.Fatalf("expected low fat7 critical chance near captured low-role scale, got %d", got)
	}
	if got := battleDodgeChancePercent(0, 1); got != 100 {
		t.Fatalf("expected zero hit to stay forced dodge for deterministic tests, got %d", got)
	}
	if got := battleCriticalChancePercent(100, 50); got != 50 {
		t.Fatalf("expected fat100 to use 50%% test critical chance, got %d", got)
	}
}

func TestResolveMiZhanConsumesSourceMpCostAndSendsActorRefresh(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-mizhan",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
	}
	actor := &CellInfoPush{
		Handle:       "player_21424",
		Camp:         CampTeam,
		Attack:       47,
		MaxMP:        64,
		MP:           64,
		Fat:          0,
		CommandLabel: "普通攻击",
	}
	target := &CellInfoPush{
		Handle:  "2429459987213640",
		Camp:    CampEnemy,
		MaxHP:   120,
		HP:      120,
		Defense: 0,
	}

	action := runtime.resolveAttack(actor, target, CommandMiZhan)

	if action.TargetActionState != "normal" || action.TargetActionStateCode != "0" {
		t.Fatalf("expected 密斩 normal state code 0, got %+v", action)
	}
	if action.ActionName != "密斩" || action.SourceActionLabel != "w8/manycut" {
		t.Fatalf("expected captured w8/manycut action for 密斩, got %+v", action)
	}
	if action.Damage != 66 || action.TargetHP != 54 {
		t.Fatalf("expected captured 密斩 +40%% damage without stored power bonus, got %+v", action)
	}
	if actor.MP != 59 {
		t.Fatalf("expected 密斩 to consume source 精力 cost 5, got actor MP=%d action=%+v", actor.MP, action)
	}
	if len(action.RefreshInfos) != 2 {
		t.Fatalf("expected actor MP and target HP refreshInfos, got %+v", action.RefreshInfos)
	}
	if action.RefreshInfos[0].Handle != actor.Handle || action.RefreshInfos[0].MP != 59 {
		t.Fatalf("expected first refreshInfo to update actor MP to 59, got %+v", action.RefreshInfos[0])
	}
	if action.RefreshInfos[1].Handle != target.Handle || action.RefreshInfos[1].HP != action.TargetHP {
		t.Fatalf("expected second refreshInfo to update target HP, got %+v action=%+v", action.RefreshInfos[1], action)
	}
}

func TestProcessActionRejectsUnlearnedCapturedSkill(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-command-skill-gate",
		Round:            1,
		Phase:            PhaseCommand,
		ActiveHandle:     "player_21424",
		nextSequence:     1,
		ConsumedSequence: map[int]bool{},
		DefendingHandles: map[string]bool{},
		RoleSkills: []session.RoleSkill{
			{
				Name:        "密斩",
				Level:       1,
				Type:        "oneE",
				Description: "f_s_密斩&9@单体·攻击&7@3&10@单刀/单斧&22@战斗&2@5&4@提升40%的物理伤害",
			},
		},
		Cells: []CellInfoPush{
			{
				BattleID: "battle-command-skill-gate",
				Handle:   "player_21424",
				Camp:     CampTeam,
				HP:       225,
				MaxHP:    225,
				MP:       64,
				MaxMP:    64,
				Attack:   47,
			},
			{
				BattleID: "battle-command-skill-gate",
				Handle:   "enemy_1",
				Camp:     CampEnemy,
				HP:       120,
				MaxHP:    120,
			},
		},
	}

	result := runtime.ProcessAction(ActionRequest{
		BattleID:     "battle-command-skill-gate",
		ActorHandle:  "player_21424",
		CommandID:    CommandDuoDuanZhan,
		TargetHandle: "enemy_1",
		Round:        1,
		Sequence:     1,
	})

	if result.ErrorCode != "unsupported_command" {
		t.Fatalf("expected unlearned 多段斩 to be rejected, got %+v", result)
	}
	if runtime.ConsumedSequence[1] {
		t.Fatalf("expected rejected command not to consume sequence")
	}
	if runtime.Cells[1].HP != 120 || runtime.Cells[0].MP != 64 {
		t.Fatalf("expected rejected command not to mutate cells, got %+v", runtime.Cells)
	}
}

func TestProcessActionAllowsLearnedCapturedSkill(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-command-skill-learned",
		Round:            1,
		Phase:            PhaseCommand,
		ActiveHandle:     "player_21424",
		nextSequence:     1,
		ConsumedSequence: map[int]bool{},
		DefendingHandles: map[string]bool{},
		StoredPower:      map[string]int{},
		RoleSkills: []session.RoleSkill{
			{
				Name:        "多段斩",
				Level:       2,
				Type:        "oneE",
				Description: "f_s_多段斩^ffffff&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@10&4@提升60%的物理伤害",
			},
		},
		Cells: []CellInfoPush{
			{
				BattleID: "battle-command-skill-learned",
				Handle:   "player_21424",
				Camp:     CampTeam,
				HP:       225,
				MaxHP:    225,
				MP:       64,
				MaxMP:    64,
				Attack:   47,
			},
			{
				BattleID: "battle-command-skill-learned",
				Handle:   "enemy_1",
				Camp:     CampEnemy,
				HP:       120,
				MaxHP:    120,
			},
		},
	}

	result := runtime.ProcessAction(ActionRequest{
		BattleID:     "battle-command-skill-learned",
		ActorHandle:  "player_21424",
		CommandID:    CommandDuoDuanZhan,
		TargetHandle: "enemy_1",
		Round:        1,
		Sequence:     1,
	})

	if result.ErrorCode != "" || len(result.Actions) == 0 {
		t.Fatalf("expected learned 多段斩 action, got %+v", result)
	}
	action := result.Actions[0]
	if action.ActionName != "多段斩" || action.SourceActionLabel != "w8/ddz1" {
		t.Fatalf("expected captured 多段斩 Lv2 action, got %+v", action)
	}
	if action.Damage != 75 || action.TargetHP != 45 || action.RefreshInfos[0].MP != 54 {
		t.Fatalf("expected learned 多段斩 Lv2 to damage and consume MP, got %+v", action)
	}
}

func TestProcessActionAllowsCapturedDaggerSkills(t *testing.T) {
	cases := []struct {
		name           string
		level          int
		description    string
		commandID      string
		label          string
		expectedDamage int
		expectedMP     int
	}{
		{
			name:           "多段刺",
			level:          5,
			description:    "f_s_多段刺^ffffff&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@18&4@提升45%的物理伤害",
			commandID:      CommandDuoDuanCi,
			label:          "w3/ddCut",
			expectedDamage: 125,
			expectedMP:     82,
		},
		{
			name:           "强力飞镖",
			level:          2,
			description:    "f_s_强力飞镖^ffffff&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@20&4@<font color='#00cc00'>特殊发动条件:需要【飞镖x1】</font><br>进攻时提高48%（无视防御）的物理攻击力",
			commandID:      CommandQiangLiFeiBiao,
			label:          "w3/powerDart",
			expectedDamage: 148,
			expectedMP:     80,
		},
		{
			name:           "强力飞镖",
			level:          3,
			description:    "f_s_强力飞镖^ffffff&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@24&4@<font color='#00cc00'>特殊发动条件:需要【飞镖x1】</font><br>进攻时提高50%（无视防御）的物理攻击力",
			commandID:      CommandQiangLiFeiBiao,
			label:          "w3/powerDart",
			expectedDamage: 150,
			expectedMP:     76,
		},
		{
			name:           "投毒",
			level:          1,
			description:    "f_s_投毒^5BC46D&9@单体·状态&8@游侠 &10@匕首&22@战斗&2@16&4@<font color='#00cc00'>特殊发动条件:需要【毒药x1】<br>叠加施放将削弱其造成中毒的功效</font><br>有80%的机率使敌人中毒，4回合内降低对方15%的物理防御和魔法防御&0;每回合使敌人损失气力为物理攻击的20%~25%",
			commandID:      CommandTouDu,
			label:          "w3/drugAtk",
			expectedDamage: 1,
			expectedMP:     84,
		},
		{
			name:           "魔力突刺",
			level:          1,
			description:    "f_s_魔力突刺^5BC46D&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@20&4@造成敌人100%的物理伤害&0;并追加80%的魔法伤害",
			commandID:      CommandMoLiTuCi,
			label:          "w3/magicCut",
			expectedDamage: 80,
			expectedMP:     80,
		},
		{
			name:           "疾风刺",
			level:          1,
			description:    "f_s_疾风刺^5BC46D&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@20&4@对敌人造成40%的物理伤害&0;击中敌人时有92%的机率使对方进入迟钝状态(削减对方50%的命中和回避)3回合<br><font color='#00cc00'>叠加施放将削弱其造成迟钝的功效</font>",
			commandID:      CommandJiFengCi,
			label:          "w3/windCut",
			expectedDamage: 20,
			expectedMP:     80,
		},
	}

	for _, testCase := range cases {
		runtime := &Runtime{
			BattleID:         "battle-command-dagger-" + testCase.name,
			Round:            1,
			Phase:            PhaseCommand,
			ActiveHandle:     "player_21432",
			nextSequence:     1,
			ConsumedSequence: map[int]bool{},
			DefendingHandles: map[string]bool{},
			StoredPower:      map[string]int{},
			RoleSkills: []session.RoleSkill{
				{
					Name:        testCase.name,
					Level:       testCase.level,
					Type:        "oneE",
					Description: testCase.description,
				},
			},
			Cells: []CellInfoPush{
				{
					BattleID: "battle-command-dagger-" + testCase.name,
					Handle:   "player_21432",
					Camp:     CampTeam,
					HP:       300,
					MaxHP:    300,
					MP:       100,
					MaxMP:    100,
					Attack:   100,
				},
				{
					BattleID: "battle-command-dagger-" + testCase.name,
					Handle:   "enemy_1",
					Camp:     CampEnemy,
					HP:       220,
					MaxHP:    220,
					Defense:  40,
				},
			},
		}

		result := runtime.ProcessAction(ActionRequest{
			BattleID:     runtime.BattleID,
			ActorHandle:  "player_21432",
			CommandID:    testCase.commandID,
			TargetHandle: "enemy_1",
			Round:        1,
			Sequence:     1,
		})

		if result.ErrorCode != "" || len(result.Actions) == 0 {
			t.Fatalf("expected learned %s action, got %+v", testCase.name, result)
		}
		action := result.Actions[0]
		if action.ActionName != testCase.name || action.SourceActionLabel != testCase.label {
			t.Fatalf("expected captured %s action label %s, got %+v", testCase.name, testCase.label, action)
		}
		if action.Damage != testCase.expectedDamage || action.TargetHP != 220-testCase.expectedDamage || action.RefreshInfos[0].MP != testCase.expectedMP {
			t.Fatalf("expected captured %s damage/mp, got %+v", testCase.name, action)
		}
	}
}

func TestProcessActionAllowsCapturedRangerBowSkills(t *testing.T) {
	cases := []struct {
		name           string
		level          int
		description    string
		commandID      string
		label          string
		expectedDamage int
		expectedMP     int
		storedPower    int
		magicAttack    int
	}{
		{
			name:           "强射",
			level:          5,
			description:    "f_s_强射^ffffff&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@18&4@提升45%的物理伤害",
			commandID:      CommandQiangShe,
			label:          "w1/powerShoot",
			expectedDamage: 125,
			expectedMP:     82,
		},
		{
			name:           "贯甲连矢",
			level:          5,
			description:    "f_s_贯甲连矢^ffffff&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@28&4@<font color='#00cc00'>特殊发动条件:需要【穿甲箭x1】</font><br>提升25%的物理伤害&0;进攻时增加30%（无视防御）的物理攻击力.",
			commandID:      CommandGuanJiaLianShi,
			label:          "w1/breakArmorShoot2",
			expectedDamage: 135,
			expectedMP:     72,
		},
		{
			name:           "魔力速射",
			level:          5,
			description:    "f_s_魔力速射^5BC46D&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@34&4@<font color='#00cc00'>特殊发动条件:需要【魔箭x1】</font><br>造成50%的物理伤害&0;并追加120%的魔法伤害(进攻时提高25%的魔法攻击力)",
			commandID:      CommandMoLiSuShe,
			label:          "w1/magicShoot",
			expectedDamage: 84,
			expectedMP:     66,
			magicAttack:    36,
		},
		{
			name:           "冰箭速射",
			level:          5,
			description:    "f_s_冰箭速射^5BC46D&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@28&4@<font color='#00cc00'>特殊发动条件:需要【冰之箭x1】</font><br><font color='#00cc00'>叠加施放将削弱其造成内伤的功效</font><br>造成70%的物理伤害&0;击中敌人时有90%的机率使敌人进入内伤状态(3回合内削弱敌人30%~35%的物理攻击和魔法攻击)",
			commandID:      CommandBingJianSuShe,
			label:          "w1/iceShoot",
			expectedDamage: 50,
			expectedMP:     72,
		},
		{
			name:           "暗影箭",
			level:          1,
			description:    "f_s_暗影箭^5BC46D&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@20&4@<font color='#00cc00'>特殊发动条件:需要【暗之箭x1】</font><br>造成72%的物理伤害&0;击中敌人时有17%的机率使敌人进入混乱状态2回合",
			commandID:      CommandAnYingJian,
			label:          "w1/darkShoot",
			expectedDamage: 52,
			expectedMP:     80,
		},
		{
			name:           "毒矢",
			level:          1,
			description:    "f_s_毒矢^5BC46D&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@15&4@<font color='#00cc00'>特殊发动条件:需要【毒箭x1】<br>叠加施放将削弱其造成中毒的功效</font><br>对敌人造成90%的物理伤害&0;击中敌人时有70%的机率使敌人中毒(4回合内降低对方20%的物理防御和魔法防御&0;每回合使敌人损失气力为物理攻击的5%~10%)",
			commandID:      CommandDuShi,
			label:          "w1/drugShoot",
			expectedDamage: 70,
			expectedMP:     85,
		},
		{
			name:           "奥义.轰雷矢",
			level:          1,
			description:    "f_s_奥义.轰雷矢^00ccff&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@26&4@<font color='#00cc00'>特殊发动条件:需要2格魂元</font><br>提升120%的魔法伤害&0;击中敌人时有20%的机率使敌人进入麻痹状态(每回合使其损失气力为魔法攻击的30%)2回合",
			commandID:      CommandAoYiHongLeiShi,
			label:          "w1/bombThunderShoot",
			expectedDamage: 74,
			expectedMP:     74,
			storedPower:    2,
			magicAttack:    36,
		},
	}

	for _, testCase := range cases {
		runtime := &Runtime{
			BattleID:         "battle-command-bow-" + testCase.name,
			Round:            1,
			Phase:            PhaseCommand,
			ActiveHandle:     "player_21432",
			nextSequence:     1,
			ConsumedSequence: map[int]bool{},
			DefendingHandles: map[string]bool{},
			StoredPower:      map[string]int{"player_21432": testCase.storedPower},
			RoleSkills: []session.RoleSkill{
				{
					Name:        testCase.name,
					Level:       testCase.level,
					Type:        "oneE",
					Description: testCase.description,
				},
			},
			Cells: []CellInfoPush{
				{
					BattleID:    "battle-command-bow-" + testCase.name,
					Handle:      "player_21432",
					Camp:        CampTeam,
					HP:          300,
					MaxHP:       300,
					MP:          100,
					MaxMP:       100,
					Attack:      100,
					MagicAttack: testCase.magicAttack,
				},
				{
					BattleID:   "battle-command-bow-" + testCase.name,
					Handle:     "enemy_1",
					Camp:       CampEnemy,
					HP:         220,
					MaxHP:      220,
					Defense:    40,
					MgcDefense: 10,
				},
			},
		}

		result := runtime.ProcessAction(ActionRequest{
			BattleID:     runtime.BattleID,
			ActorHandle:  "player_21432",
			CommandID:    testCase.commandID,
			TargetHandle: "enemy_1",
			Round:        1,
			Sequence:     1,
		})

		if result.ErrorCode != "" || len(result.Actions) == 0 {
			t.Fatalf("expected learned %s action, got %+v", testCase.name, result)
		}
		action := result.Actions[0]
		if action.ActionName != testCase.name || action.SourceActionLabel != testCase.label {
			t.Fatalf("expected captured %s action label %s, got %+v", testCase.name, testCase.label, action)
		}
		if action.Damage != testCase.expectedDamage || action.TargetHP != 220-testCase.expectedDamage || action.RefreshInfos[0].MP != testCase.expectedMP {
			t.Fatalf("expected captured %s damage/mp, got %+v", testCase.name, action)
		}
	}
}

func TestProcessActionAllowsCapturedRangerSelfStatusSkill(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-command-ranger-self-status",
		Round:            1,
		Phase:            PhaseCommand,
		ActiveHandle:     "player_21432",
		nextSequence:     1,
		ConsumedSequence: map[int]bool{},
		DefendingHandles: map[string]bool{},
		StoredPower:      map[string]int{},
		StatusEffects: map[string]BattleStatusEffects{
			"player_21432": {
				Effects: map[string]BattleStatusEffect{
					"中毒": {
						Name:                  "中毒",
						Rounds:                3,
						SourceHandle:          "enemy_1",
						SourceSkill:           "投毒",
						DefenseReduction:      9,
						MagicDefenseReduction: 5,
					},
				},
			},
		},
		RoleSkills: []session.RoleSkill{
			{
				Name:        "解毒术",
				Level:       1,
				Type:        "own",
				Description: "f_s_解毒术^ffffff&9@单体·状态&8@游侠 &10@匕首&22@战斗&2@20&4@解除自身中毒状态",
			},
		},
		Cells: []CellInfoPush{
			{
				BattleID:   "battle-command-ranger-self-status",
				Handle:     "player_21432",
				Camp:       CampTeam,
				HP:         300,
				MaxHP:      300,
				MP:         100,
				MaxMP:      100,
				Attack:     100,
				Defense:    51,
				MgcDefense: 31,
			},
			{
				BattleID: "battle-command-ranger-self-status",
				Handle:   "enemy_1",
				Camp:     CampEnemy,
				HP:       220,
				MaxHP:    220,
				Defense:  40,
			},
		},
	}
	initialActor := *runtime.cellByHandle("player_21432")
	initialEnemy := *runtime.cellByHandle("enemy_1")
	initialPoisonBuff := BuffInfoPush{
		BattleID:      runtime.BattleID,
		ReleaseHandle: "enemy_1",
		TargetHandle:  "player_21432",
		Name:          "中毒",
		Display:       "8.png",
		Description:   "降低对象5点魔防和9点物防，每回合内减少对象20~25点气力",
		Round:         3,
	}

	result := runtime.ProcessAction(ActionRequest{
		BattleID:     runtime.BattleID,
		ActorHandle:  "player_21432",
		CommandID:    CommandJieDuShu,
		TargetHandle: "player_21432",
		Round:        1,
		Sequence:     1,
	})

	if result.ErrorCode != "" || len(result.Actions) == 0 {
		t.Fatalf("expected learned 解毒术 action, got %+v", result)
	}
	action := result.Actions[0]
	if action.ActionName != "解毒术" || action.SourceActionLabel != "w3/releaseDrug" || action.SourceMode != "0" {
		t.Fatalf("expected captured 解毒术 self action, got %+v", action)
	}
	if action.Damage != 0 || action.TargetHP != 300 || action.TargetMP != 80 || action.RefreshInfos[0].MP != 80 {
		t.Fatalf("expected 解毒术 to consume MP without damage, got %+v", action)
	}
	if action.RefreshInfos[0].Defense != 60 || action.RefreshInfos[0].MgcDefense != 36 {
		t.Fatalf("expected 解毒术 to restore poison defense reductions, got %+v", action.RefreshInfos[0])
	}
	if _, ok := runtime.StatusEffects["player_21432"]; ok {
		t.Fatalf("expected 解毒术 to clear 中毒 status, got %+v", runtime.StatusEffects)
	}
	if len(result.ClearBuffInfos) != 1 || result.ClearBuffInfos[0].TargetHandle != "player_21432" || result.ClearBuffInfos[0].Name != "中毒" {
		t.Fatalf("expected 解毒术 to push clearBuffInfo for 中毒, got %+v", result.ClearBuffInfos)
	}
	writeJieDuRealFlowFixture(t, initialActor, initialEnemy, initialPoisonBuff, result)
}

func TestBattleSkillProfileFromCapturedDescriptions(t *testing.T) {
	cases := []struct {
		name       string
		level      int
		desc       string
		label      string
		mpCost     int
		multiplier float64
		chance     int
		ratio      float64
	}{
		{
			name:       "密斩",
			level:      1,
			desc:       "f_s_密斩&9@单体·攻击&7@3&10@单刀/单斧&22@战斗&2@5&4@提升40%的物理伤害",
			label:      "w8/manycut",
			mpCost:     5,
			multiplier: 1.4,
		},
		{
			name:       "多段斩",
			level:      2,
			desc:       "f_s_多段斩^ffffff&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@10&4@提升60%的物理伤害",
			label:      "w8/ddz1",
			mpCost:     10,
			multiplier: 1.6,
		},
		{
			name:       "多段斩",
			level:      3,
			desc:       "f_s_多段斩^ffffff&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@12&4@提升65%的物理伤害",
			label:      "w8/ddz2",
			mpCost:     12,
			multiplier: 1.65,
		},
		{
			name:       "多段斩",
			level:      5,
			desc:       "f_s_多段斩^ffffff&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@16&4@提升75%的物理伤害",
			label:      "w8/ddz2",
			mpCost:     16,
			multiplier: 1.75,
		},
		{
			name:       "多段刺",
			level:      5,
			desc:       "f_s_多段刺^ffffff&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@18&4@提升45%的物理伤害",
			label:      "w3/ddCut",
			mpCost:     18,
			multiplier: 1.45,
		},
		{
			name:       "嗜血斩",
			level:      3,
			desc:       "f_s_嗜血斩^5BC46D&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@28&4@对敌人造成96%的物理伤害&0;并有86%机率将对敌人造成伤害的70%转换为气力</font>",
			label:      "w8/xyz2",
			mpCost:     28,
			multiplier: 0.96,
			chance:     86,
			ratio:      0.7,
		},
		{
			name:       "红月斩",
			level:      1,
			desc:       "f_s_红月斩^ffffff&9@群体·攻击&8@战士 &10@单刀&22@战斗&2@40&4@对所有敌人造成72%的物理伤害",
			label:      "w8/redMoonAtk",
			mpCost:     40,
			multiplier: 0.72,
		},
		{
			name:       "血切",
			level:      1,
			desc:       "f_s_血切^5BC46D&9@单体·状态&8@战士 &10@单刀&22@战斗&2@19&4@对敌人造成30%的物理伤害&0;击中敌人时有80%的机率使对方进入外伤状态4回合<br>(每回合损失气力为角色物理攻击的25%~30%)",
			label:      "w8/cutBlood",
			mpCost:     19,
			multiplier: 0.3,
		},
		{
			name:       "力释棍术",
			level:      1,
			desc:       "f_s_力释棍术^5BC46D&9@单体·状态&8@战士 &10@棍&22@战斗&2@10&4@5回合内提升物理攻击15%",
			label:      "w11/releasePower",
			mpCost:     10,
			multiplier: 0,
		},
		{
			name:       "盘龙棍法",
			level:      1,
			desc:       "f_s_盘龙棍法^ffffff&9@群体·攻击&8@战士 &10@棍&22@战斗&2@14&4@对所有敌人造成82%的物理伤害",
			label:      "w11/circleDargon",
			mpCost:     14,
			multiplier: 0.82,
		},
		{
			name:       "夜叉棍法",
			level:      1,
			desc:       "f_s_夜叉棍法^5BC46D&9@单体·攻击&8@战士 &10@棍&22@战斗&2@15&4@提升12%的物理伤害&0;击中敌人时有90%的机率对敌人造成内伤(削减敌人32%的物理攻击和魔法攻击)3回合<br><font color='#00cc00'>叠加施放将削弱其造成内伤的功效</font>",
			label:      "w11/yaksa",
			mpCost:     15,
			multiplier: 1.12,
		},
		{
			name:       "强力飞镖",
			level:      2,
			desc:       "f_s_强力飞镖^ffffff&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@20&4@<font color='#00cc00'>特殊发动条件:需要【飞镖x1】</font><br>进攻时提高48%（无视防御）的物理攻击力",
			label:      "w3/powerDart",
			mpCost:     20,
			multiplier: 1.48,
		},
		{
			name:       "强力飞镖",
			level:      3,
			desc:       "f_s_强力飞镖^ffffff&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@24&4@<font color='#00cc00'>特殊发动条件:需要【飞镖x1】</font><br>进攻时提高50%（无视防御）的物理攻击力",
			label:      "w3/powerDart",
			mpCost:     24,
			multiplier: 1.5,
		},
		{
			name:       "投毒",
			level:      1,
			desc:       "f_s_投毒^5BC46D&9@单体·状态&8@游侠 &10@匕首&22@战斗&2@16&4@<font color='#00cc00'>特殊发动条件:需要【毒药x1】<br>叠加施放将削弱其造成中毒的功效</font><br>有80%的机率使敌人中毒，4回合内降低对方15%的物理防御和魔法防御&0;每回合使敌人损失气力为物理攻击的20%~25%",
			label:      "w3/drugAtk",
			mpCost:     16,
			multiplier: 0,
		},
		{
			name:       "魔力突刺",
			level:      1,
			desc:       "f_s_魔力突刺^5BC46D&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@20&4@造成敌人100%的物理伤害&0;并追加80%的魔法伤害",
			label:      "w3/magicCut",
			mpCost:     20,
			multiplier: 1,
		},
		{
			name:       "疾风刺",
			level:      1,
			desc:       "f_s_疾风刺^5BC46D&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@20&4@对敌人造成40%的物理伤害&0;击中敌人时有92%的机率使对方进入迟钝状态(削减对方50%的命中和回避)3回合<br><font color='#00cc00'>叠加施放将削弱其造成迟钝的功效</font>",
			label:      "w3/windCut",
			mpCost:     20,
			multiplier: 0.4,
		},
		{
			name:       "解毒术",
			level:      1,
			desc:       "f_s_解毒术^ffffff&9@单体·状态&8@游侠 &10@匕首&22@战斗&2@20&4@解除自身中毒状态",
			label:      "w3/releaseDrug",
			mpCost:     20,
			multiplier: 0,
		},
		{
			name:       "奥义.雷魂斩",
			level:      1,
			desc:       "f_s_奥义.雷魂斩^00ccff&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@24&4@<font color='#00cc00'>特殊发动条件:需要3格魂元</font><br>提升240%的物理伤害",
			label:      "w8/thunderSoulAtk",
			mpCost:     24,
			multiplier: 3.4,
		},
		{
			name:       "魔力速射",
			level:      5,
			desc:       "f_s_魔力速射^5BC46D&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@34&4@<font color='#00cc00'>特殊发动条件:需要【魔箭x1】</font><br>造成50%的物理伤害&0;并追加120%的魔法伤害(进攻时提高25%的魔法攻击力)",
			label:      "w1/magicShoot",
			mpCost:     34,
			multiplier: 0.5,
		},
		{
			name:       "冰箭速射",
			level:      5,
			desc:       "f_s_冰箭速射^5BC46D&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@28&4@<font color='#00cc00'>特殊发动条件:需要【冰之箭x1】</font><br><font color='#00cc00'>叠加施放将削弱其造成内伤的功效</font><br>造成70%的物理伤害&0;击中敌人时有90%的机率使敌人进入内伤状态(3回合内削弱敌人30%~35%的物理攻击和魔法攻击)",
			label:      "w1/iceShoot",
			mpCost:     28,
			multiplier: 0.7,
		},
		{
			name:       "暗影箭",
			level:      1,
			desc:       "f_s_暗影箭^5BC46D&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@20&4@<font color='#00cc00'>特殊发动条件:需要【暗之箭x1】</font><br>造成72%的物理伤害&0;击中敌人时有17%的机率使敌人进入混乱状态2回合",
			label:      "w1/darkShoot",
			mpCost:     20,
			multiplier: 0.72,
		},
		{
			name:       "毒矢",
			level:      1,
			desc:       "f_s_毒矢^5BC46D&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@15&4@<font color='#00cc00'>特殊发动条件:需要【毒箭x1】<br>叠加施放将削弱其造成中毒的功效</font><br>对敌人造成90%的物理伤害&0;击中敌人时有70%的机率使敌人中毒(4回合内降低对方20%的物理防御和魔法防御&0;每回合使敌人损失气力为物理攻击的5%~10%)",
			label:      "w1/drugShoot",
			mpCost:     15,
			multiplier: 0.9,
		},
		{
			name:       "奥义.轰雷矢",
			level:      1,
			desc:       "f_s_奥义.轰雷矢^00ccff&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@26&4@<font color='#00cc00'>特殊发动条件:需要2格魂元</font><br>提升120%的魔法伤害&0;击中敌人时有20%的机率使敌人进入麻痹状态(每回合使其损失气力为魔法攻击的30%)2回合",
			label:      "w1/bombThunderShoot",
			mpCost:     26,
			multiplier: 2.2,
		},
		{
			name:       "奥义.暗杀者",
			level:      1,
			desc:       "f_s_奥义.暗杀者^00ccff&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@26&4@<font color='#00cc00'>特殊发动条件:需要3格魂元</font><br>提升180%的物理伤害",
			label:      "w3/assassinate",
			mpCost:     26,
			multiplier: 2.8,
		},
		{
			name:       "奥义.六合棍法",
			level:      1,
			desc:       "f_s_奥义.六合棍法^00ccff&9@单体·攻击&8@战士 &10@棍&22@战斗&2@24&4@<font color='#00cc00'>特殊发动条件:需要3格魂元</font><br>提升210%的物理伤害&0;进攻时候增加300%的命中",
			label:      "w11/liuhe",
			mpCost:     24,
			multiplier: 3.1,
		},
	}

	for _, testCase := range cases {
		profile := sourceBattleSkillProfile(session.RoleSkill{
			Name:        testCase.name,
			Level:       testCase.level,
			Type:        "oneE",
			Description: testCase.desc,
		})
		if profile.ActionName != testCase.name || profile.SourceActionLabel != testCase.label || profile.MPCost != testCase.mpCost {
			t.Fatalf("expected captured profile for %s Lv%d, got %+v", testCase.name, testCase.level, profile)
		}
		if profile.DamageMultiplier != testCase.multiplier {
			t.Fatalf("expected %s Lv%d multiplier %v, got %+v", testCase.name, testCase.level, testCase.multiplier, profile)
		}
		if profile.LifeStealChance != testCase.chance || profile.LifeStealRatio != testCase.ratio {
			t.Fatalf("expected %s Lv%d lifesteal %d/%v, got %+v", testCase.name, testCase.level, testCase.chance, testCase.ratio, profile)
		}
		if testCase.name == "强力飞镖" && profile.DefenseType != "direct" {
			t.Fatalf("expected 强力飞镖 to ignore defense from captured description, got %+v", profile)
		}
		if testCase.name == "暗影箭" && (profile.StatusName != "混乱" || profile.StatusChance != 17 || profile.StatusRounds != 2) {
			t.Fatalf("expected 暗影箭 captured confusion status, got %+v", profile)
		}
		if testCase.name == "毒矢" && (profile.StatusName != "中毒" || profile.StatusChance != 70 || profile.StatusRounds != 4 || profile.StatusDefensePercent != 20 || profile.StatusTickMin != 5 || profile.StatusTickMax != 10) {
			t.Fatalf("expected 毒矢 captured poison status, got %+v", profile)
		}
		if testCase.name == "魔力速射" && (profile.AdditionalMagicBonus != 1.2 || profile.MagicAttackBoost != 0.25) {
			t.Fatalf("expected 魔力速射 captured physical+magic profile, got %+v", profile)
		}
		if testCase.name == "冰箭速射" && (profile.StatusName != "内伤" || profile.StatusChance != 90 || profile.StatusRounds != 3 || profile.StatusAttackMin != 30 || profile.StatusAttackMax != 35) {
			t.Fatalf("expected 冰箭速射 captured inner-injury profile, got %+v", profile)
		}
		if testCase.name == "夜叉棍法" && (profile.StatusName != "内伤" || profile.StatusChance != 90 || profile.StatusRounds != 3 || profile.StatusAttackMin != 32 || profile.StatusAttackMax != 32) {
			t.Fatalf("expected 夜叉棍法 captured inner-injury profile, got %+v", profile)
		}
		if testCase.name == "奥义.轰雷矢" && (profile.DefenseType != "magic" || !profile.UseMagicAttack || profile.StatusName != "麻痹" || profile.StatusChance != 20 || profile.StatusTickMin != 30 || !profile.SkipTurn) {
			t.Fatalf("expected 奥义.轰雷矢 captured magic/palsy profile, got %+v", profile)
		}
	}
}

func TestSourceBattleSkillProfileUsesClassicDataSkillTable(t *testing.T) {
	row, ok, err := classicdata.FindSkillByLabel("投毒")
	if err != nil {
		t.Fatalf("FindSkillByLabel error = %v", err)
	}
	if !ok {
		t.Fatal("expected classicdata skill row for 投毒")
	}
	levelRows, err := classicdata.FindSkillLevelRowsBySkillID(row["skill_id"])
	if err != nil {
		t.Fatalf("FindSkillLevelRowsBySkillID error = %v", err)
	}
	if len(levelRows) != 1 || levelRows[0]["level"] != "1" || levelRows[0]["source_action_label"] != "w3/drugAtk" {
		t.Fatalf("expected 投毒 Lv1 skill-level row, got %+v", levelRows)
	}
	if commandID := sourceBattleSkillCommandID("投毒"); commandID != row["skill_id"] {
		t.Fatalf("expected 投毒 command id from skill master %q, got %q", row["skill_id"], commandID)
	}
	profile := sourceBattleSkillProfile(session.RoleSkill{
		Name:  "投毒",
		Level: 1,
		Type:  "oneE",
	})
	if profile.ActionName != row["action_name"] || profile.SourceType != row["source_type"] || profile.SourceActionLabel != levelRows[0]["source_action_label"] {
		t.Fatalf("expected 投毒 profile to link classicdata skill and level rows master=%+v levels=%+v got=%+v", row, levelRows, profile)
	}
	commands := sourceBattleCommandDefinitions([]session.RoleSkill{{Name: "投毒", Level: 1, Type: "oneE"}})
	var touDu CommandDefinition
	for _, command := range commands {
		if command.ID == row["skill_id"] {
			touDu = command
			break
		}
	}
	if touDu.ID == "" || touDu.Target != row["target"] {
		t.Fatalf("expected 投毒 command definition to use classicdata target row=%+v commands=%+v", row, commands)
	}
}

func TestBattleSkillProfileUsesCapturedLevelWhenStoredDescriptionIsStale(t *testing.T) {
	duoDuan := sourceBattleSkillProfile(session.RoleSkill{
		Name:        "多段斩",
		Level:       3,
		Type:        "oneE",
		Description: "提升对敌人造成的物理伤害",
	})
	if duoDuan.SourceActionLabel != "w8/ddz2" || duoDuan.MPCost != 12 || duoDuan.DamageMultiplier != 1.65 {
		t.Fatalf("expected 多段斩 Lv3 captured profile to ignore stale description, got %+v", duoDuan)
	}

	duoDuanLv5 := sourceBattleSkillProfile(session.RoleSkill{
		Name:        "多段斩",
		Level:       5,
		Type:        "oneE",
		Description: "f_s_多段斩^ffffff&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@8&4@提升55%的物理伤害",
	})
	if duoDuanLv5.SourceActionLabel != "w8/ddz2" || duoDuanLv5.MPCost != 16 || duoDuanLv5.DamageMultiplier != 1.75 {
		t.Fatalf("expected 多段斩 Lv5 captured profile to ignore stale Lv1 description, got %+v", duoDuanLv5)
	}

	duoDuanCi := sourceBattleSkillProfile(session.RoleSkill{
		Name:        "多段刺",
		Level:       5,
		Type:        "oneE",
		Description: "提高对敌人造成的物理伤害",
	})
	if duoDuanCi.SourceActionLabel != "w3/ddCut" || duoDuanCi.MPCost != 18 || duoDuanCi.DamageMultiplier != 1.45 {
		t.Fatalf("expected 多段刺 Lv5 captured profile to ignore stale description, got %+v", duoDuanCi)
	}

	shiXue := sourceBattleSkillProfile(session.RoleSkill{
		Name:        "嗜血斩",
		Level:       3,
		Type:        "oneE",
		Description: "f_s_嗜血斩^5BC46D&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@24&4@对敌人造成92%的物理伤害&0;并有82%机率将对敌人造成伤害的70%转换为气力</font>",
	})
	if shiXue.SourceActionLabel != "w8/xyz2" || shiXue.MPCost != 28 || shiXue.DamageMultiplier != 0.96 || shiXue.LifeStealChance != 86 || shiXue.LifeStealRatio != 0.7 {
		t.Fatalf("expected 嗜血斩 Lv3 captured profile to ignore stale Lv1 description, got %+v", shiXue)
	}

	qiangLiFeiBiao := sourceBattleSkillProfile(session.RoleSkill{
		Name:        "强力飞镖",
		Level:       2,
		Type:        "oneE",
		Description: "对敌人造成物理伤害 / 进攻时候提升一定的物理攻击力",
	})
	if qiangLiFeiBiao.SourceActionLabel != "w3/powerDart" || qiangLiFeiBiao.MPCost != 20 || qiangLiFeiBiao.DamageMultiplier != 1.48 || qiangLiFeiBiao.DefenseType != "direct" {
		t.Fatalf("expected 强力飞镖 Lv2 captured profile to ignore stale description, got %+v", qiangLiFeiBiao)
	}

	qiangLiFeiBiaoLv3 := sourceBattleSkillProfile(session.RoleSkill{
		Name:        "强力飞镖",
		Level:       3,
		Type:        "oneE",
		Description: "对敌人造成物理伤害 / 进攻时候提升一定的物理攻击力",
	})
	if qiangLiFeiBiaoLv3.SourceActionLabel != "w3/powerDart" || qiangLiFeiBiaoLv3.MPCost != 24 || qiangLiFeiBiaoLv3.DamageMultiplier != 1.5 || qiangLiFeiBiaoLv3.DefenseType != "direct" {
		t.Fatalf("expected 强力飞镖 Lv3 captured profile to ignore stale description, got %+v", qiangLiFeiBiaoLv3)
	}

	touDu := sourceBattleSkillProfile(session.RoleSkill{
		Name:        "投毒",
		Level:       1,
		Type:        "oneE",
		Description: "有机率使敌人中毒",
	})
	if touDu.SourceActionLabel != "w3/drugAtk" || touDu.MPCost != 16 || touDu.DamageMultiplier != 0 || touDu.StatusName != "中毒" || touDu.StatusChance != 80 || touDu.StatusRounds != 4 {
		t.Fatalf("expected 投毒 Lv1 captured profile to ignore stale description, got %+v", touDu)
	}

	moLiTuCi := sourceBattleSkillProfile(session.RoleSkill{
		Name:        "魔力突刺",
		Level:       1,
		Type:        "oneE",
		Description: "提升对敌人造成的物理伤害 / 并附加魔法伤害",
	})
	if moLiTuCi.SourceActionLabel != "w3/magicCut" || moLiTuCi.MPCost != 20 || moLiTuCi.DamageMultiplier != 1 {
		t.Fatalf("expected 魔力突刺 Lv1 captured profile to ignore stale description, got %+v", moLiTuCi)
	}

	jiFengCi := sourceBattleSkillProfile(session.RoleSkill{
		Name:        "疾风刺",
		Level:       1,
		Type:        "oneE",
		Description: "对敌人造成物理伤害 / 击中敌人时有机率使其进入迟钝状态",
	})
	if jiFengCi.SourceActionLabel != "w3/windCut" || jiFengCi.MPCost != 20 || jiFengCi.DamageMultiplier != 0.4 {
		t.Fatalf("expected 疾风刺 Lv1 captured profile to ignore stale description, got %+v", jiFengCi)
	}

	jieDuShu := sourceBattleSkillProfile(session.RoleSkill{
		Name:        "解毒术",
		Level:       1,
		Type:        "own",
		Description: "解除自身的中毒状态",
	})
	if jieDuShu.SourceActionLabel != "w3/releaseDrug" || jieDuShu.MPCost != 20 || jieDuShu.DamageMultiplier != 0 || jieDuShu.SourceType != "own" {
		t.Fatalf("expected 解毒术 Lv1 captured profile to ignore stale description, got %+v", jieDuShu)
	}

	aoYiAnShaZhe := sourceBattleSkillProfile(session.RoleSkill{
		Name:        "奥义.暗杀者",
		Level:       1,
		Type:        "oneE",
		Description: "特殊发动条件:3格魂元 / 大幅提升对敌人造成的物理伤害",
	})
	if aoYiAnShaZhe.SourceActionLabel != "w3/assassinate" || aoYiAnShaZhe.MPCost != 26 || aoYiAnShaZhe.DamageMultiplier != 2.8 {
		t.Fatalf("expected 奥义.暗杀者 Lv1 captured profile to ignore stale description, got %+v", aoYiAnShaZhe)
	}

	liShiGunShu := sourceBattleSkillProfile(session.RoleSkill{
		Name:        "力释棍术",
		Level:       1,
		Type:        "oneE",
		Description: "提升自身物理攻击",
	})
	if liShiGunShu.SourceActionLabel != "w11/releasePower" || liShiGunShu.MPCost != 10 || liShiGunShu.DamageMultiplier != 0 || liShiGunShu.SourceType != "own" {
		t.Fatalf("expected 力释棍术 Lv1 captured profile to ignore stale description, got %+v", liShiGunShu)
	}

	panLongGunFa := sourceBattleSkillProfile(session.RoleSkill{
		Name:        "盘龙棍法",
		Level:       1,
		Type:        "oneE",
		Description: "对敌人造成群体物理伤害",
	})
	if panLongGunFa.SourceActionLabel != "w11/circleDargon" || panLongGunFa.MPCost != 14 || panLongGunFa.DamageMultiplier != 0.82 || panLongGunFa.SourceType != "all" {
		t.Fatalf("expected 盘龙棍法 Lv1 captured profile to ignore stale description, got %+v", panLongGunFa)
	}

	piShanGunFa := sourceBattleSkillProfile(session.RoleSkill{
		Name:        "劈山棍法",
		Level:       5,
		Type:        "oneE",
		Description: "f_s_劈山棍法^ffffff&9@单体·攻击&8@战士 &10@棍&22@战斗&2@16&4@提升75%的物理伤害",
	})
	if piShanGunFa.SourceActionLabel != "w11/cutHill2" || piShanGunFa.MPCost != 16 || piShanGunFa.DamageMultiplier != 1.75 || piShanGunFa.SourceType != "oneE" {
		t.Fatalf("expected 劈山棍法 Lv5 captured profile, got %+v", piShanGunFa)
	}

	yeChaGunFa := sourceBattleSkillProfile(session.RoleSkill{
		Name:        "夜叉棍法",
		Level:       1,
		Type:        "oneE",
		Description: "提升物理伤害并造成内伤",
	})
	if yeChaGunFa.SourceActionLabel != "w11/yaksa" || yeChaGunFa.MPCost != 15 || yeChaGunFa.DamageMultiplier != 1.12 || yeChaGunFa.SourceType != "oneE" || yeChaGunFa.StatusName != "内伤" || yeChaGunFa.StatusAttackMin != 32 || yeChaGunFa.StatusAttackMax != 32 {
		t.Fatalf("expected 夜叉棍法 Lv1 captured profile to ignore stale description, got %+v", yeChaGunFa)
	}

	aoYiLiuHeGunFa := sourceBattleSkillProfile(session.RoleSkill{
		Name:        "奥义.六合棍法",
		Level:       1,
		Type:        "oneE",
		Description: "特殊发动条件:3格魂元 / 大幅提升伤害和命中",
	})
	if aoYiLiuHeGunFa.SourceActionLabel != "w11/liuhe" || aoYiLiuHeGunFa.MPCost != 24 || aoYiLiuHeGunFa.DamageMultiplier != 3.1 || aoYiLiuHeGunFa.HitMultiplier != 4 {
		t.Fatalf("expected 奥义.六合棍法 Lv1 captured profile to ignore stale description, got %+v", aoYiLiuHeGunFa)
	}
}

func TestSourceBattleCommandDefinitionsUseCapturedSkillProfiles(t *testing.T) {
	commands := sourceBattleCommandDefinitions([]session.RoleSkill{
		{
			Name:        "普通攻击",
			Level:       1,
			Type:        "oneE",
			Description: "f_s_普通攻击^ffffff&9@单体·攻击&10@通用&22@战斗&5@给予对手普通的物理攻击.",
		},
		{
			Name:        "密斩",
			Level:       1,
			Type:        "oneE",
			Description: "f_s_密斩&9@单体·攻击&7@3&10@单刀/单斧&22@战斗&2@5&4@提升40%的物理伤害",
		},
		{
			Name:        "多段斩",
			Level:       3,
			Type:        "oneE",
			Description: "f_s_多段斩^ffffff&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@12&4@提升65%的物理伤害",
		},
		{
			Name:        "多段刺",
			Level:       5,
			Type:        "oneE",
			Description: "f_s_多段刺^ffffff&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@18&4@提升45%的物理伤害",
		},
		{
			Name:        "嗜血斩",
			Level:       2,
			Type:        "oneE",
			Description: "f_s_嗜血斩^5BC46D&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@26&4@对敌人造成94%的物理伤害&0;并有84%机率将对敌人造成伤害的70%转换为气力</font>",
		},
		{
			Name:        "狂爆",
			Level:       1,
			Type:        "技能·单刀系",
			Description: "f_s_狂爆^5BC46D&9@单体·状态&8@战士 &10@单刀&22@战斗&2@15&4@3回合内物理攻击力翻倍&0;并降低100%的物理防御",
		},
		{
			Name:        "红月斩",
			Level:       1,
			Type:        "all",
			Description: "f_s_红月斩^ffffff&9@群体·攻击&8@战士 &10@单刀&22@战斗&2@40&4@对所有敌人造成72%的物理伤害",
		},
		{
			Name:        "血切",
			Level:       1,
			Type:        "oneE",
			Description: "f_s_血切^5BC46D&9@单体·状态&8@战士 &10@单刀&22@战斗&2@19&4@对敌人造成30%的物理伤害&0;击中敌人时有80%的机率使对方进入外伤状态4回合<br>(每回合损失气力为角色物理攻击的25%~30%)",
		},
		{
			Name:        "力释棍术",
			Level:       1,
			Type:        "own",
			Description: "f_s_力释棍术^5BC46D&9@单体·状态&8@战士 &10@棍&22@战斗&2@10&4@5回合内提升物理攻击15%",
		},
		{
			Name:        "盘龙棍法",
			Level:       1,
			Type:        "all",
			Description: "f_s_盘龙棍法^ffffff&9@群体·攻击&8@战士 &10@棍&22@战斗&2@14&4@对所有敌人造成82%的物理伤害",
		},
		{
			Name:        "劈山棍法",
			Level:       5,
			Type:        "oneE",
			Description: "f_s_劈山棍法^ffffff&9@单体·攻击&8@战士 &10@棍&22@战斗&2@16&4@提升75%的物理伤害",
		},
		{
			Name:        "夜叉棍法",
			Level:       1,
			Type:        "oneE",
			Description: "f_s_夜叉棍法^5BC46D&9@单体·攻击&8@战士 &10@棍&22@战斗&2@15&4@提升12%的物理伤害&0;击中敌人时有90%的机率对敌人造成内伤(削减敌人32%的物理攻击和魔法攻击)3回合<br><font color='#00cc00'>叠加施放将削弱其造成内伤的功效</font>",
		},
		{
			Name:        "强力飞镖",
			Level:       2,
			Type:        "oneE",
			Description: "f_s_强力飞镖^ffffff&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@20&4@<font color='#00cc00'>特殊发动条件:需要【飞镖x1】</font><br>进攻时提高48%（无视防御）的物理攻击力",
		},
		{
			Name:        "投毒",
			Level:       1,
			Type:        "oneE",
			Description: "f_s_投毒^5BC46D&9@单体·状态&8@游侠 &10@匕首&22@战斗&2@16&4@<font color='#00cc00'>特殊发动条件:需要【毒药x1】<br>叠加施放将削弱其造成中毒的功效</font><br>有80%的机率使敌人中毒，4回合内降低对方15%的物理防御和魔法防御&0;每回合使敌人损失气力为物理攻击的20%~25%",
		},
		{
			Name:        "魔力突刺",
			Level:       1,
			Type:        "oneE",
			Description: "f_s_魔力突刺^5BC46D&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@20&4@造成敌人100%的物理伤害&0;并追加80%的魔法伤害",
		},
		{
			Name:        "疾风刺",
			Level:       1,
			Type:        "oneE",
			Description: "f_s_疾风刺^5BC46D&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@20&4@对敌人造成40%的物理伤害&0;击中敌人时有92%的机率使对方进入迟钝状态(削减对方50%的命中和回避)3回合<br><font color='#00cc00'>叠加施放将削弱其造成迟钝的功效</font>",
		},
		{
			Name:        "解毒术",
			Level:       1,
			Type:        "own",
			Description: "f_s_解毒术^ffffff&9@单体·状态&8@游侠 &10@匕首&22@战斗&2@20&4@解除自身中毒状态",
		},
		{
			Name:        "强射",
			Level:       5,
			Type:        "oneE",
			Description: "f_s_强射^ffffff&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@18&4@提升45%的物理伤害",
		},
		{
			Name:        "贯甲连矢",
			Level:       2,
			Type:        "oneE",
			Description: "f_s_贯甲连矢^ffffff&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@25&4@<font color='#00cc00'>特殊发动条件:需要【穿甲箭x1】</font><br>提升10%的物理伤害&0;进攻时增加15%（无视防御）的物理攻击力.",
		},
		{
			Name:        "魔力速射",
			Level:       5,
			Type:        "oneE",
			Description: "f_s_魔力速射^5BC46D&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@34&4@<font color='#00cc00'>特殊发动条件:需要【魔箭x1】</font><br>造成50%的物理伤害&0;并追加120%的魔法伤害(进攻时提高25%的魔法攻击力)",
		},
		{
			Name:        "冰箭速射",
			Level:       5,
			Type:        "oneE",
			Description: "f_s_冰箭速射^5BC46D&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@28&4@<font color='#00cc00'>特殊发动条件:需要【冰之箭x1】</font><br><font color='#00cc00'>叠加施放将削弱其造成内伤的功效</font><br>造成70%的物理伤害&0;击中敌人时有90%的机率使敌人进入内伤状态(3回合内削弱敌人30%~35%的物理攻击和魔法攻击)",
		},
		{
			Name:        "暗影箭",
			Level:       1,
			Type:        "oneE",
			Description: "f_s_暗影箭^5BC46D&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@20&4@<font color='#00cc00'>特殊发动条件:需要【暗之箭x1】</font><br>造成72%的物理伤害&0;击中敌人时有17%的机率使敌人进入混乱状态2回合",
		},
		{
			Name:        "毒矢",
			Level:       1,
			Type:        "oneE",
			Description: "f_s_毒矢^5BC46D&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@15&4@<font color='#00cc00'>特殊发动条件:需要【毒箭x1】<br>叠加施放将削弱其造成中毒的功效</font><br>对敌人造成90%的物理伤害&0;击中敌人时有70%的机率使敌人中毒(4回合内降低对方20%的物理防御和魔法防御&0;每回合使敌人损失气力为物理攻击的5%~10%)",
		},
		{
			Name:        "奥义.轰雷矢",
			Level:       1,
			Type:        "oneE",
			Description: "f_s_奥义.轰雷矢^00ccff&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@26&4@<font color='#00cc00'>特殊发动条件:需要2格魂元</font><br>提升120%的魔法伤害&0;击中敌人时有20%的机率使敌人进入麻痹状态(每回合使其损失气力为魔法攻击的30%)2回合",
		},
		{
			Name:        "奥义.雷魂斩",
			Level:       1,
			Type:        "oneE",
			Description: "f_s_奥义.雷魂斩^00ccff&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@24&4@<font color='#00cc00'>特殊发动条件:需要3格魂元</font><br>提升240%的物理伤害",
		},
		{
			Name:        "奥义.暗杀者",
			Level:       1,
			Type:        "oneE",
			Description: "f_s_奥义.暗杀者^00ccff&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@26&4@<font color='#00cc00'>特殊发动条件:需要3格魂元</font><br>提升180%的物理伤害",
		},
		{
			Name:        "奥义.六合棍法",
			Level:       1,
			Type:        "oneE",
			Description: "f_s_奥义.六合棍法^00ccff&9@单体·攻击&8@战士 &10@棍&22@战斗&2@24&4@<font color='#00cc00'>特殊发动条件:需要3格魂元</font><br>提升210%的物理伤害&0;进攻时候增加300%的命中",
		},
		{
			Name: "未抓包技能",
			Type: "oneE",
		},
	})

	byID := map[string]CommandDefinition{}
	for _, command := range commands {
		byID[command.ID] = command
	}
	if byID[CommandNormalAttack].SourceActionLabel != "nomalAtk" {
		t.Fatalf("expected normal attack command, got %+v", commands)
	}
	if command := byID[CommandMiZhan]; command.Label != "密斩" || command.SourceActionLabel != "w8/manycut" || command.MPCost != 5 || command.DamageMultiplier != 1.4 {
		t.Fatalf("expected captured 密斩 command, got %+v", command)
	}
	if command := byID[CommandDuoDuanZhan]; command.Label != "多段斩" || command.SourceActionLabel != "w8/ddz2" || command.MPCost != 12 || command.DamageMultiplier != 1.65 {
		t.Fatalf("expected captured 多段斩 Lv3 command, got %+v", command)
	}
	if command := byID[CommandDuoDuanCi]; command.Label != "多段刺" || command.SourceActionLabel != "w3/ddCut" || command.MPCost != 18 || command.DamageMultiplier != 1.45 {
		t.Fatalf("expected captured 多段刺 Lv5 command, got %+v", command)
	}
	if command := byID[CommandShiXueZhan]; command.Label != "嗜血斩" || command.SourceActionLabel != "w8/xyz1" || command.MPCost != 26 || command.DamageMultiplier != 0.94 {
		t.Fatalf("expected captured 嗜血斩 Lv2 command, got %+v", command)
	}
	if command := byID[CommandKuangBao]; command.Label != "狂爆" || command.SourceType != "own" || command.Target != "self" || command.SourceActionLabel != "w8/kb" || command.MPCost != 15 || command.DamageMultiplier != 0 {
		t.Fatalf("expected captured 狂爆 self status command, got %+v", command)
	}
	if command := byID[CommandHongYueZhan]; command.Label != "红月斩" || command.SourceType != "all" || command.Target != "enemy" || command.SourceActionLabel != "w8/redMoonAtk" || command.MPCost != 40 || command.DamageMultiplier != 0.72 {
		t.Fatalf("expected captured 红月斩 all-target command, got %+v", command)
	}
	if command := byID[CommandXueQie]; command.Label != "血切" || command.SourceType != "oneE" || command.Target != "enemy" || command.SourceActionLabel != "w8/cutBlood" || command.MPCost != 19 || command.DamageMultiplier != 0.3 {
		t.Fatalf("expected captured 血切 single-target command, got %+v", command)
	}
	if command := byID[CommandLiShiGunShu]; command.Label != "力释棍术" || command.SourceType != "own" || command.Target != "self" || command.SourceActionLabel != "w11/releasePower" || command.MPCost != 10 || command.DamageMultiplier != 0 {
		t.Fatalf("expected captured 力释棍术 self command, got %+v", command)
	}
	if command := byID[CommandPanLongGunFa]; command.Label != "盘龙棍法" || command.SourceType != "all" || command.Target != "enemy" || command.SourceActionLabel != "w11/circleDargon" || command.MPCost != 14 || command.DamageMultiplier != 0.82 {
		t.Fatalf("expected captured 盘龙棍法 all-target command, got %+v", command)
	}
	if command := byID[CommandPiShanGunFa]; command.Label != "劈山棍法" || command.SourceType != "oneE" || command.Target != "enemy" || command.SourceActionLabel != "w11/cutHill2" || command.MPCost != 16 || command.DamageMultiplier != 1.75 {
		t.Fatalf("expected captured 劈山棍法 Lv5 command, got %+v", command)
	}
	if command := byID[CommandYeChaGunFa]; command.Label != "夜叉棍法" || command.SourceType != "oneE" || command.Target != "enemy" || command.SourceActionLabel != "w11/yaksa" || command.MPCost != 15 || command.DamageMultiplier != 1.12 {
		t.Fatalf("expected captured 夜叉棍法 Lv1 command, got %+v", command)
	}
	if command := byID[CommandQiangLiFeiBiao]; command.Label != "强力飞镖" || command.SourceType != "oneE" || command.Target != "enemy" || command.SourceActionLabel != "w3/powerDart" || command.MPCost != 20 || command.DamageMultiplier != 1.48 {
		t.Fatalf("expected captured 强力飞镖 Lv2 command, got %+v", command)
	}
	if command := byID[CommandTouDu]; command.Label != "投毒" || command.SourceType != "oneE" || command.Target != "enemy" || command.SourceActionLabel != "w3/drugAtk" || command.MPCost != 16 || command.DamageMultiplier != 0 {
		t.Fatalf("expected captured 投毒 status command, got %+v", command)
	}
	if command := byID[CommandMoLiTuCi]; command.Label != "魔力突刺" || command.SourceType != "oneE" || command.Target != "enemy" || command.SourceActionLabel != "w3/magicCut" || command.MPCost != 20 || command.DamageMultiplier != 1 {
		t.Fatalf("expected captured 魔力突刺 command, got %+v", command)
	}
	if command := byID[CommandJiFengCi]; command.Label != "疾风刺" || command.SourceType != "oneE" || command.Target != "enemy" || command.SourceActionLabel != "w3/windCut" || command.MPCost != 20 || command.DamageMultiplier != 0.4 {
		t.Fatalf("expected captured 疾风刺 command, got %+v", command)
	}
	if command := byID[CommandJieDuShu]; command.Label != "解毒术" || command.SourceType != "own" || command.Target != "self" || command.SourceActionLabel != "w3/releaseDrug" || command.MPCost != 20 || command.DamageMultiplier != 0 {
		t.Fatalf("expected captured 解毒术 self command, got %+v", command)
	}
	if command := byID[CommandQiangShe]; command.Label != "强射" || command.SourceType != "oneE" || command.Target != "enemy" || command.SourceActionLabel != "w1/powerShoot" || command.MPCost != 18 || command.DamageMultiplier != 1.45 {
		t.Fatalf("expected captured 强射 Lv5 command, got %+v", command)
	}
	if command := byID[CommandGuanJiaLianShi]; command.Label != "贯甲连矢" || command.SourceType != "oneE" || command.Target != "enemy" || command.SourceActionLabel != "w1/breakArmorShoot2" || command.MPCost != 25 || command.DamageMultiplier != 1.1 {
		t.Fatalf("expected captured 贯甲连矢 Lv2 command, got %+v", command)
	}
	if command := byID[CommandMoLiSuShe]; command.Label != "魔力速射" || command.SourceType != "oneE" || command.Target != "enemy" || command.SourceActionLabel != "w1/magicShoot" || command.MPCost != 34 || command.DamageMultiplier != 0.5 {
		t.Fatalf("expected captured 魔力速射 Lv5 command, got %+v", command)
	}
	if command := byID[CommandBingJianSuShe]; command.Label != "冰箭速射" || command.SourceType != "oneE" || command.Target != "enemy" || command.SourceActionLabel != "w1/iceShoot" || command.MPCost != 28 || command.DamageMultiplier != 0.7 {
		t.Fatalf("expected captured 冰箭速射 Lv5 command, got %+v", command)
	}
	if command := byID[CommandAnYingJian]; command.Label != "暗影箭" || command.SourceType != "oneE" || command.Target != "enemy" || command.SourceActionLabel != "w1/darkShoot" || command.MPCost != 20 || command.DamageMultiplier != 0.72 {
		t.Fatalf("expected captured 暗影箭 Lv1 command, got %+v", command)
	}
	if command := byID[CommandDuShi]; command.Label != "毒矢" || command.SourceType != "oneE" || command.Target != "enemy" || command.SourceActionLabel != "w1/drugShoot" || command.MPCost != 15 || command.DamageMultiplier != 0.9 {
		t.Fatalf("expected captured 毒矢 Lv1 command, got %+v", command)
	}
	if command := byID[CommandLeiHunZhan]; command.Label != "奥义.雷魂斩" || command.SourceType != "oneE" || command.Target != "enemy" || command.SourceActionLabel != "w8/thunderSoulAtk" || command.MPCost != 24 || command.DamageMultiplier != 3.4 {
		t.Fatalf("expected captured 奥义.雷魂斩 single-target command, got %+v", command)
	}
	if command := byID[CommandAoYiHongLeiShi]; command.Label != "奥义.轰雷矢" || command.SourceType != "oneE" || command.Target != "enemy" || command.SourceActionLabel != "w1/bombThunderShoot" || command.MPCost != 26 || command.DamageMultiplier != 2.2 {
		t.Fatalf("expected captured 奥义.轰雷矢 single-target command, got %+v", command)
	}
	if command := byID[CommandAoYiAnShaZhe]; command.Label != "奥义.暗杀者" || command.SourceType != "oneE" || command.Target != "enemy" || command.SourceActionLabel != "w3/assassinate" || command.MPCost != 26 || command.DamageMultiplier != 2.8 {
		t.Fatalf("expected captured 奥义.暗杀者 single-target command, got %+v", command)
	}
	if command := byID[CommandAoYiLiuHeGunFa]; command.Label != "奥义.六合棍法" || command.SourceType != "oneE" || command.Target != "enemy" || command.SourceActionLabel != "w11/liuhe" || command.MPCost != 24 || command.DamageMultiplier != 3.1 {
		t.Fatalf("expected captured 奥义.六合棍法 single-target command, got %+v", command)
	}
	if _, ok := byID["未抓包技能"]; ok {
		t.Fatalf("expected uncaptured skill to be omitted, got %+v", commands)
	}
	if byID[CommandStore].SourceActionLabel != "def" || byID[CommandEscape].SourceActionLabel != "escapeSuccess" {
		t.Fatalf("expected utility commands to use source labels, got %+v", commands)
	}
}

func TestGuanJiaLianShiUsesCapturedDirectAttackBonus(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-ranger-bow",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
		RoleSkills: []session.RoleSkill{
			{
				Name:        "贯甲连矢",
				Level:       2,
				Type:        "oneE",
				Description: "f_s_贯甲连矢^ffffff&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@25&4@<font color='#00cc00'>特殊发动条件:需要【穿甲箭x1】</font><br>提升10%的物理伤害&0;进攻时增加15%（无视防御）的物理攻击力.",
			},
		},
	}
	actor := &CellInfoPush{
		Handle: "player_21432",
		Camp:   CampTeam,
		Attack: 100,
		Hit:    1000,
		MaxMP:  100,
		MP:     100,
		Fat:    0,
	}
	target := &CellInfoPush{
		Handle:  "enemy_bow_target",
		Camp:    CampEnemy,
		MaxHP:   500,
		HP:      500,
		Defense: 50,
		Dog:     0,
	}

	action := runtime.resolveAttack(actor, target, CommandGuanJiaLianShi)

	if action.ActionName != "贯甲连矢" || action.SourceActionLabel != "w1/breakArmorShoot2" || action.TargetActionStateCode != "0" {
		t.Fatalf("expected captured 贯甲连矢 hit action, got %+v", action)
	}
	if action.Damage != 100 || action.TargetHP != 400 || target.HP != 400 {
		t.Fatalf("expected round(100*1.1)-round(50*0.5) plus round(100*0.15)=100 damage, got action=%+v target=%+v", action, target)
	}
	if actor.MP != 75 || len(action.RefreshInfos) != 2 || action.RefreshInfos[0].MP != 75 {
		t.Fatalf("expected captured 贯甲连矢 MP cost 25, got actor=%+v action=%+v", actor, action)
	}
}

func TestResolveDuoDuanZhanUsesCapturedLevelProfileAndCanFat(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-duoduan",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
		RoleSkills: []session.RoleSkill{
			{
				Name:        "多段斩",
				Level:       3,
				Type:        "oneE",
				Description: "f_s_多段斩^ffffff&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@12&4@提升65%的物理伤害",
			},
		},
	}
	actor := &CellInfoPush{
		Handle: "player_21424",
		Camp:   CampTeam,
		Attack: 100,
		MaxMP:  194,
		MP:     194,
		Fat:    100,
	}
	target := &CellInfoPush{
		Handle:  "2429459987213640",
		Camp:    CampEnemy,
		MaxHP:   452,
		HP:      452,
		Defense: 0,
	}

	action := runtime.resolveAttack(actor, target, CommandDuoDuanZhan)

	if action.ActionName != "多段斩" || action.SourceActionLabel != "w8/ddz2" {
		t.Fatalf("expected captured 多段斩 Lv3 label, got %+v", action)
	}
	if action.Damage != 330 || action.TargetHP != 122 || action.TargetActionStateCode != "2" {
		t.Fatalf("expected Lv3 多段斩 damage without stored power bonus before fat, got %+v", action)
	}
	if actor.MP != 182 || action.RefreshInfos[0].MP != 182 {
		t.Fatalf("expected 多段斩 Lv3 to consume MP 12, got actor=%+v action=%+v", actor, action)
	}
}

func TestHongYueZhanHitsAllLivingEnemiesAndConsumesMPOnce(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-hongyue",
		Round:            1,
		Phase:            PhaseCommand,
		ActiveHandle:     "player_21424",
		nextSequence:     1,
		ConsumedSequence: map[int]bool{},
		DefendingHandles: map[string]bool{},
		RoleSkills: []session.RoleSkill{
			{
				Name:        "红月斩",
				Level:       1,
				Type:        "all",
				Description: "f_s_红月斩^ffffff&9@群体·攻击&8@战士 &10@单刀&22@战斗&2@40&4@对所有敌人造成72%的物理伤害",
			},
		},
		Cells: []CellInfoPush{
			{
				BattleID: "battle-hongyue",
				Handle:   "player_21424",
				Camp:     CampTeam,
				HP:       500,
				MaxHP:    500,
				MP:       100,
				MaxMP:    100,
				Attack:   100,
				Defense:  0,
				Hit:      100,
				Fat:      0,
			},
			{
				BattleID: "battle-hongyue",
				Handle:   "enemy_1",
				Camp:     CampEnemy,
				HP:       500,
				MaxHP:    500,
				Attack:   1,
				Defense:  0,
				Dog:      0,
			},
			{
				BattleID: "battle-hongyue",
				Handle:   "enemy_2",
				Camp:     CampEnemy,
				HP:       500,
				MaxHP:    500,
				Attack:   1,
				Defense:  0,
				Dog:      0,
			},
		},
	}

	result := runtime.ProcessAction(ActionRequest{
		BattleID:     "battle-hongyue",
		ActorHandle:  "player_21424",
		CommandID:    CommandHongYueZhan,
		TargetHandle: "enemy_1",
		Round:        1,
		Sequence:     1,
	})

	if result.ErrorCode != "" {
		t.Fatalf("expected 红月斩 to be accepted, got %+v", result)
	}
	if len(result.Actions) < 1 {
		t.Fatalf("expected 红月斩 to emit one all-target action, got %+v", result.Actions)
	}
	action := result.Actions[0]
	if action.ActionName != "红月斩" || action.SourceMode != "1" || action.SourceActionLabel != "w8/redMoonAtk" || action.TargetHandle != "all" {
		t.Fatalf("expected 红月斩 source all-target action, got %+v", action)
	}
	if len(action.TargetActionResults) != 2 || action.TargetActionResults[0].Handle != "enemy_1" || action.TargetActionResults[1].Handle != "enemy_2" {
		t.Fatalf("expected 红月斩 to carry target result pairs, got %+v", action.TargetActionResults)
	}
	for _, expectedHandle := range []string{"enemy_1", "enemy_2"} {
		target := runtime.cellByHandle(expectedHandle)
		if target == nil || target.HP != 428 {
			t.Fatalf("expected 红月斩 Lv1 captured damage against %s, got target=%+v action=%+v", expectedHandle, target, action)
		}
	}
	if len(action.RefreshInfos) != 3 || action.RefreshInfos[0].Handle != "player_21424" || action.RefreshInfos[1].Handle != "enemy_1" || action.RefreshInfos[2].Handle != "enemy_2" {
		t.Fatalf("expected 红月斩 to refresh actor once and both targets, got %+v", action.RefreshInfos)
	}
	actor := runtime.cellByHandle("player_21424")
	if actor == nil || actor.MP != 60 {
		t.Fatalf("expected 红月斩 to consume MP 40 once, got actor=%+v", actor)
	}
}

func TestPanLongGunFaHitsAllLivingEnemiesAndConsumesCapturedMPOnce(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-panlong",
		Round:            1,
		Phase:            PhaseCommand,
		ActiveHandle:     "player_21424",
		nextSequence:     1,
		ConsumedSequence: map[int]bool{},
		DefendingHandles: map[string]bool{},
		RoleSkills: []session.RoleSkill{
			{
				Name:        "盘龙棍法",
				Level:       1,
				Type:        "all",
				Description: "f_s_盘龙棍法^ffffff&9@群体·攻击&8@战士 &10@棍&22@战斗&2@14&4@对所有敌人造成82%的物理伤害",
			},
		},
		Cells: []CellInfoPush{
			{
				BattleID: "battle-panlong",
				Handle:   "player_21424",
				Camp:     CampTeam,
				HP:       500,
				MaxHP:    500,
				MP:       100,
				MaxMP:    100,
				Attack:   100,
				Defense:  0,
				Hit:      100,
				Fat:      0,
			},
			{
				BattleID: "battle-panlong",
				Handle:   "enemy_1",
				Camp:     CampEnemy,
				HP:       500,
				MaxHP:    500,
				Attack:   1,
				Defense:  0,
				Dog:      0,
			},
			{
				BattleID: "battle-panlong",
				Handle:   "enemy_2",
				Camp:     CampEnemy,
				HP:       500,
				MaxHP:    500,
				Attack:   1,
				Defense:  0,
				Dog:      0,
			},
		},
	}

	result := runtime.ProcessAction(ActionRequest{
		BattleID:     "battle-panlong",
		ActorHandle:  "player_21424",
		CommandID:    CommandPanLongGunFa,
		TargetHandle: "enemy_1",
		Round:        1,
		Sequence:     1,
	})

	if result.ErrorCode != "" {
		t.Fatalf("expected 盘龙棍法 to be accepted, got %+v", result)
	}
	if len(result.Actions) < 1 {
		t.Fatalf("expected 盘龙棍法 to emit one all-target action, got %+v", result.Actions)
	}
	action := result.Actions[0]
	if action.ActionName != "盘龙棍法" || action.SourceMode != "1" || action.SourceActionLabel != "w11/circleDargon" || action.TargetHandle != "all" {
		t.Fatalf("expected 盘龙棍法 source all-target action, got %+v", action)
	}
	if len(action.TargetActionResults) != 2 || action.TargetActionResults[0].Handle != "enemy_1" || action.TargetActionResults[1].Handle != "enemy_2" {
		t.Fatalf("expected 盘龙棍法 to carry target result pairs, got %+v", action.TargetActionResults)
	}
	for _, expectedHandle := range []string{"enemy_1", "enemy_2"} {
		target := runtime.cellByHandle(expectedHandle)
		if target == nil || target.HP != 418 {
			t.Fatalf("expected 盘龙棍法 Lv1 captured damage against %s, got target=%+v action=%+v", expectedHandle, target, action)
		}
	}
	actor := runtime.cellByHandle("player_21424")
	if actor == nil || actor.MP != 86 {
		t.Fatalf("expected 盘龙棍法 to consume MP 14 once, got actor=%+v", actor)
	}
}

func TestPiShanGunFaUsesCapturedLv5DamageAndCutHill2(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-pishan",
		Round:            1,
		Phase:            PhaseCommand,
		ActiveHandle:     "player_21424",
		nextSequence:     1,
		ConsumedSequence: map[int]bool{},
		DefendingHandles: map[string]bool{},
		RoleSkills: []session.RoleSkill{
			{
				Name:        "劈山棍法",
				Level:       5,
				Type:        "oneE",
				Description: "f_s_劈山棍法^ffffff&9@单体·攻击&8@战士 &10@棍&22@战斗&2@16&4@提升75%的物理伤害",
			},
		},
		Cells: []CellInfoPush{
			{
				BattleID: "battle-pishan",
				Handle:   "player_21424",
				Camp:     CampTeam,
				HP:       500,
				MaxHP:    500,
				MP:       100,
				MaxMP:    100,
				Attack:   100,
				Defense:  0,
				Hit:      100,
				Fat:      0,
			},
			{
				BattleID: "battle-pishan",
				Handle:   "enemy_1",
				Camp:     CampEnemy,
				HP:       500,
				MaxHP:    500,
				Attack:   1,
				Defense:  0,
				Dog:      0,
			},
		},
	}

	result := runtime.ProcessAction(ActionRequest{
		BattleID:     "battle-pishan",
		ActorHandle:  "player_21424",
		CommandID:    CommandPiShanGunFa,
		TargetHandle: "enemy_1",
		Round:        1,
		Sequence:     1,
	})

	if result.ErrorCode != "" {
		t.Fatalf("expected 劈山棍法 to be accepted, got %+v", result)
	}
	if len(result.Actions) < 1 {
		t.Fatalf("expected 劈山棍法 action, got %+v", result.Actions)
	}
	action := result.Actions[0]
	if action.ActionName != "劈山棍法" || action.SourceMode != "1" || action.SourceActionLabel != "w11/cutHill2" || action.TargetHandle != "enemy_1" {
		t.Fatalf("expected captured 劈山棍法 single-target action, got %+v", action)
	}
	if action.Damage != 175 || action.TargetHP != 325 || action.TargetActionStateCode != "0" {
		t.Fatalf("expected 劈山棍法 Lv5 captured damage, got %+v", action)
	}
	actor := runtime.cellByHandle("player_21424")
	target := runtime.cellByHandle("enemy_1")
	if actor == nil || actor.MP != 84 || target == nil || target.HP != 325 {
		t.Fatalf("expected 劈山棍法 to consume MP 16 and damage target, actor=%+v target=%+v", actor, target)
	}
	if len(action.RefreshInfos) != 2 || action.RefreshInfos[0].Handle != "player_21424" || action.RefreshInfos[0].MP != 84 || action.RefreshInfos[1].Handle != "enemy_1" || action.RefreshInfos[1].HP != 325 {
		t.Fatalf("expected 劈山棍法 to refresh actor MP and target HP, got %+v", action.RefreshInfos)
	}
}

func TestYeChaGunFaUsesCapturedLv1DamageAndInnerInjury(t *testing.T) {
	var runtime *Runtime
	var actor *CellInfoPush
	var target *CellInfoPush
	for index := 0; index < 500; index += 1 {
		actorHandle := fmt.Sprintf("player_yaksa_%d", index)
		candidate := &Runtime{
			BattleID:         "battle-yaksa",
			Round:            1,
			Phase:            PhaseCommand,
			ActiveHandle:     actorHandle,
			nextSequence:     1,
			ConsumedSequence: map[int]bool{},
			DefendingHandles: map[string]bool{},
			RoleSkills: []session.RoleSkill{
				{
					Name:        "夜叉棍法",
					Level:       1,
					Type:        "oneE",
					Description: "f_s_夜叉棍法^5BC46D&9@单体·攻击&8@战士 &10@棍&22@战斗&2@15&4@提升12%的物理伤害&0;击中敌人时有90%的机率对敌人造成内伤(削减敌人32%的物理攻击和魔法攻击)3回合<br><font color='#00cc00'>叠加施放将削弱其造成内伤的功效</font>",
				},
			},
			Cells: []CellInfoPush{
				{
					BattleID: "battle-yaksa",
					Handle:   actorHandle,
					Camp:     CampTeam,
					HP:       500,
					MaxHP:    500,
					MP:       100,
					MaxMP:    100,
					Attack:   100,
					Defense:  0,
					Hit:      100,
					Fat:      0,
				},
				{
					BattleID:    "battle-yaksa",
					Handle:      "enemy_yaksa",
					Camp:        CampEnemy,
					HP:          500,
					MaxHP:       500,
					Attack:      180,
					MagicAttack: 10,
					Defense:     0,
					Dog:         0,
				},
			},
		}
		candidateActor := candidate.cellByHandle(actorHandle)
		candidateTarget := candidate.cellByHandle("enemy_yaksa")
		if candidate.hashBattleRollWithSalt(candidateActor, candidateTarget, CommandYeChaGunFa, "status:内伤") < 90 {
			runtime = candidate
			actor = candidateActor
			target = candidateTarget
			break
		}
	}
	if runtime == nil {
		t.Fatal("expected to find deterministic 夜叉棍法 inner-injury roll below 90")
	}

	result := runtime.ProcessAction(ActionRequest{
		BattleID:     "battle-yaksa",
		ActorHandle:  actor.Handle,
		CommandID:    CommandYeChaGunFa,
		TargetHandle: target.Handle,
		Round:        1,
		Sequence:     1,
	})

	if result.ErrorCode != "" {
		t.Fatalf("expected 夜叉棍法 to be accepted, got %+v", result)
	}
	if len(result.Actions) < 1 {
		t.Fatalf("expected 夜叉棍法 action, got %+v", result.Actions)
	}
	action := result.Actions[0]
	if action.ActionName != "夜叉棍法" || action.SourceMode != "1" || action.SourceActionLabel != "w11/yaksa" || action.TargetHandle != "enemy_yaksa" {
		t.Fatalf("expected captured 夜叉棍法 single-target action, got %+v", action)
	}
	if action.Damage != 112 || action.TargetHP != 388 || action.TargetActionStateCode != "0" {
		t.Fatalf("expected 夜叉棍法 Lv1 captured damage, got %+v", action)
	}
	if actor.MP != 85 || target.HP != 388 || target.Attack != 122 || target.MagicAttack != 7 {
		t.Fatalf("expected 夜叉棍法 to consume MP and apply 32%% inner injury, actor=%+v target=%+v", actor, target)
	}
	if len(result.BuffInfos) != 1 {
		t.Fatalf("expected one 夜叉棍法 内伤 BuffInfo, got %+v", result.BuffInfos)
	}
	buff := result.BuffInfos[0]
	if buff.Name != "内伤" || buff.Display != "26.png" || buff.Round != 3 || buff.ReleaseHandle != actor.Handle || buff.TargetHandle != target.Handle {
		t.Fatalf("expected captured 夜叉棍法 内伤 BuffInfo metadata, got %+v", buff)
	}
	if !strings.Contains(buff.Description, "降低对象58点物理攻击") || !strings.Contains(buff.Description, "3点魔法攻击力") {
		t.Fatalf("expected captured 夜叉棍法 内伤 reduction description, got %q", buff.Description)
	}
	effect := runtime.StatusEffects[target.Handle].Effects["内伤"]
	if effect.Name != "内伤" || effect.Display != "26.png" || effect.Rounds != 2 || effect.SourceHandle != actor.Handle || effect.SourceSkill != "夜叉棍法" || effect.AttackReduction != 58 || effect.MagicAttackReduction != 3 || effect.StatusAttackMin != 32 || effect.StatusAttackMax != 32 {
		t.Fatalf("expected runtime 夜叉棍法 内伤 status to be recorded, got %+v", runtime.StatusEffects)
	}
}

func TestYeChaGunFaDoesNotApplyInnerInjuryOnDodge(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-yaksa-dodge",
		Round:            1,
		Phase:            PhaseCommand,
		ActiveHandle:     "player_yaksa_dodge",
		nextSequence:     1,
		ConsumedSequence: map[int]bool{},
		DefendingHandles: map[string]bool{},
		RoleSkills: []session.RoleSkill{
			{
				Name:        "夜叉棍法",
				Level:       1,
				Type:        "oneE",
				Description: "f_s_夜叉棍法^5BC46D&9@单体·攻击&8@战士 &10@棍&22@战斗&2@15&4@提升12%的物理伤害&0;击中敌人时有90%的机率对敌人造成内伤(削减敌人32%的物理攻击和魔法攻击)3回合<br><font color='#00cc00'>叠加施放将削弱其造成内伤的功效</font>",
			},
		},
		Cells: []CellInfoPush{
			{
				BattleID: "battle-yaksa-dodge",
				Handle:   "player_yaksa_dodge",
				Camp:     CampTeam,
				HP:       500,
				MaxHP:    500,
				MP:       100,
				MaxMP:    100,
				Attack:   100,
				Hit:      0,
			},
			{
				BattleID:    "battle-yaksa-dodge",
				Handle:      "enemy_yaksa_dodge",
				Camp:        CampEnemy,
				HP:          500,
				MaxHP:       500,
				Attack:      180,
				MagicAttack: 10,
				Defense:     0,
				Dog:         1,
			},
		},
	}

	result := runtime.ProcessAction(ActionRequest{
		BattleID:     "battle-yaksa-dodge",
		ActorHandle:  "player_yaksa_dodge",
		CommandID:    CommandYeChaGunFa,
		TargetHandle: "enemy_yaksa_dodge",
		Round:        1,
		Sequence:     1,
	})

	if result.ErrorCode != "" {
		t.Fatalf("expected 夜叉棍法 dodge action to be accepted, got %+v", result)
	}
	if len(result.Actions) < 1 || result.Actions[0].TargetActionStateCode != "1" || result.Actions[0].Damage != 0 {
		t.Fatalf("expected 夜叉棍法 dodge to use captured state code 1, got %+v", result.Actions)
	}
	if len(result.BuffInfos) != 0 || len(runtime.StatusEffects) != 0 {
		t.Fatalf("expected dodge to suppress 夜叉棍法 内伤 status, buffs=%+v effects=%+v", result.BuffInfos, runtime.StatusEffects)
	}
}

func TestLiShiGunShuAppliesCapturedFightingSpirit(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-lishi",
		Round:            1,
		Phase:            PhaseCommand,
		ActiveHandle:     "player_21424",
		nextSequence:     1,
		ConsumedSequence: map[int]bool{},
		DefendingHandles: map[string]bool{},
		RoleSkills: []session.RoleSkill{
			{
				Name:        "力释棍术",
				Level:       1,
				Type:        "own",
				Description: "f_s_力释棍术^5BC46D&9@单体·状态&8@战士 &10@棍&22@战斗&2@10&4@5回合内提升物理攻击15%",
			},
		},
		Cells: []CellInfoPush{
			{
				BattleID: "battle-lishi",
				Handle:   "player_21424",
				Camp:     CampTeam,
				HP:       500,
				MaxHP:    500,
				MP:       50,
				MaxMP:    50,
				Attack:   100,
				Defense:  0,
				Hit:      100,
			},
			{
				BattleID: "battle-lishi",
				Handle:   "enemy_1",
				Camp:     CampEnemy,
				HP:       500,
				MaxHP:    500,
				Attack:   0,
				Defense:  0,
			},
		},
	}

	result := runtime.ProcessAction(ActionRequest{
		BattleID:     "battle-lishi",
		ActorHandle:  "player_21424",
		CommandID:    CommandLiShiGunShu,
		TargetHandle: "player_21424",
		Round:        1,
		Sequence:     1,
	})

	if result.ErrorCode != "" {
		t.Fatalf("expected 力释棍术 to be accepted, got %+v", result)
	}
	if len(result.Actions) < 1 {
		t.Fatalf("expected 力释棍术 action, got %+v", result.Actions)
	}
	action := result.Actions[0]
	if action.ActionName != "力释棍术" || action.SourceActionLabel != "w11/releasePower" || action.TargetHandle != "player_21424" {
		t.Fatalf("expected captured 力释棍术 self action, got %+v", action)
	}
	actor := runtime.cellByHandle("player_21424")
	if actor == nil || actor.Attack != 115 || actor.MP != 40 {
		t.Fatalf("expected 力释棍术 to raise attack by 15 and consume MP 10, got actor=%+v", actor)
	}
	if len(result.BuffInfos) != 1 || result.BuffInfos[0].Name != "斗志" || result.BuffInfos[0].Display != "23.png" || result.BuffInfos[0].Round != 5 || !strings.Contains(result.BuffInfos[0].Description, "15点物理攻击") {
		t.Fatalf("expected captured 斗志 BuffInfo, got %+v", result.BuffInfos)
	}
	effect := runtime.StatusEffects["player_21424"].Effects["斗志"]
	if effect.AttackIncrease != 15 || effect.AppliedAction != "w11/releasePower" {
		t.Fatalf("expected 力释棍术 to persist reversible attack increase, got %+v", effect)
	}

	secondBuff := runtime.applyFightingSpiritStatusEffect(actor)
	if actor.Attack != 115 {
		t.Fatalf("expected reapplying 斗志 to restore the old increase before applying the new one, got actor=%+v secondBuff=%+v", actor, secondBuff)
	}
	effect = runtime.StatusEffects["player_21424"].Effects["斗志"]
	if effect.AttackIncrease != 15 || effect.Rounds != liShiGunShuRounds || !strings.Contains(secondBuff.Description, "15点物理攻击") {
		t.Fatalf("expected overwritten 斗志 to keep a single reversible attack increase, effect=%+v secondBuff=%+v", effect, secondBuff)
	}
	for turn := 1; turn < liShiGunShuRounds; turn += 1 {
		actions, skipTurn := runtime.resolveStatusStartActions(actor)
		if len(actions) != 0 || skipTurn {
			t.Fatalf("expected 斗志 round %d to tick without action or skip, actions=%+v skip=%v", turn, actions, skipTurn)
		}
		if actor.Attack != 115 {
			t.Fatalf("expected 斗志 round %d to keep boosted attack before expiry, got actor=%+v", turn, actor)
		}
		effect = runtime.StatusEffects["player_21424"].Effects["斗志"]
		if effect.Rounds != liShiGunShuRounds-turn {
			t.Fatalf("expected 斗志 round %d to decrement to %d, got %+v", turn, liShiGunShuRounds-turn, effect)
		}
	}
	actions, skipTurn := runtime.resolveStatusStartActions(actor)
	if len(actions) != 0 || skipTurn {
		t.Fatalf("expected expiring 斗志 to tick without action or skip, actions=%+v skip=%v", actions, skipTurn)
	}
	if actor.Attack != 100 {
		t.Fatalf("expected expired 斗志 to restore original attack, got actor=%+v", actor)
	}
	if _, ok := runtime.StatusEffects["player_21424"]; ok {
		t.Fatalf("expected expired 斗志 status to clear, got %+v", runtime.StatusEffects)
	}
	if len(runtime.PendingClearBuffInfos) != 1 || runtime.PendingClearBuffInfos[0].Name != "斗志" || runtime.PendingClearBuffInfos[0].TargetHandle != "player_21424" {
		t.Fatalf("expected 斗志 clear buff info, got %+v", runtime.PendingClearBuffInfos)
	}
}

func TestCapturedMageSupportSkillsUseCapturedActionsAndFixedHealing(t *testing.T) {
	newRuntime := func(skill session.RoleSkill) *Runtime {
		return &Runtime{
			BattleID:         "battle-mage-support-" + skill.Name,
			Round:            1,
			Phase:            PhaseCommand,
			ActiveHandle:     "player_mage",
			nextSequence:     1,
			ConsumedSequence: map[int]bool{},
			DefendingHandles: map[string]bool{},
			StatusEffects:    map[string]BattleStatusEffects{},
			RoleSkills:       []session.RoleSkill{skill},
			Cells: []CellInfoPush{
				{BattleID: "battle-mage-support", Handle: "player_mage", Camp: CampTeam, HP: 700, MaxHP: 1105, MP: 500, MaxMP: 3273, MagicAttack: 367},
				{BattleID: "battle-mage-support", Handle: "player_ally", Camp: CampTeam, HP: 383, MaxHP: 2000, MP: 100, MaxMP: 200, Attack: 100, MagicAttack: 80},
				{BattleID: "battle-mage-support", Handle: "enemy_1", Camp: CampEnemy, HP: 1000, MaxHP: 1000, Attack: 1},
			},
		}
	}

	yuQi := newRuntime(session.RoleSkill{Name: "愈气术", Level: 5, Type: "oneO"})
	yuQiResult := yuQi.ProcessAction(ActionRequest{BattleID: yuQi.BattleID, ActorHandle: "player_mage", CommandID: CommandYuQiShu, TargetHandle: "player_ally", Round: 1, Sequence: 1})
	if yuQiResult.ErrorCode != "" || len(yuQiResult.Actions) == 0 {
		t.Fatalf("expected captured 愈气术 action, got %+v", yuQiResult)
	}
	yuQiAction := yuQiResult.Actions[0]
	if yuQiAction.ActionName != "愈气术" || yuQiAction.SourceActionLabel != "w10/hpup" || yuQiAction.SourceMode != "0" || yuQiAction.TargetHandle != "player_ally" || yuQiAction.TargetHP != 1148 || yuQi.cellByHandle("player_mage").MP != 395 {
		t.Fatalf("expected captured 愈气术 +765 and MP105 action, got action=%+v actor=%+v", yuQiAction, yuQi.cellByHandle("player_mage"))
	}
	if len(yuQiAction.RefreshInfos) != 2 || yuQiAction.RefreshInfos[0].Handle != "player_ally" || yuQiAction.RefreshInfos[0].HP != 1148 || yuQiAction.RefreshInfos[1].Handle != "player_mage" || yuQiAction.RefreshInfos[1].MP != 395 {
		t.Fatalf("expected 愈气术 target/actor refresh order, got %+v", yuQiAction.RefreshInfos)
	}

	huiShang := newRuntime(session.RoleSkill{Name: "回伤术", Level: 1, Type: "oneO"})
	huiTarget := huiShang.cellByHandle("player_ally")
	huiTarget.Attack = 70
	huiTarget.MagicAttack = 56
	huiShang.StatusEffects[huiTarget.Handle] = BattleStatusEffects{Effects: map[string]BattleStatusEffect{
		"内伤": {Name: "内伤", Rounds: 2, AttackReduction: 30, MagicAttackReduction: 24},
		"外伤": {Name: "外伤", Rounds: 2},
	}}
	huiShangResult := huiShang.ProcessAction(ActionRequest{BattleID: huiShang.BattleID, ActorHandle: "player_mage", CommandID: CommandHuiShangShu, TargetHandle: huiTarget.Handle, Round: 1, Sequence: 1})
	if huiShangResult.ErrorCode != "" || len(huiShangResult.Actions) == 0 {
		t.Fatalf("expected captured 回伤术 action, got %+v", huiShangResult)
	}
	huiShangAction := huiShangResult.Actions[0]
	if huiShangAction.ActionName != "回伤术" || huiShangAction.SourceActionLabel != "w10/backInjury" || huiShangAction.TargetHandle != huiTarget.Handle || huiShang.cellByHandle("player_mage").MP != 440 {
		t.Fatalf("expected captured 回伤术 action/MP, got action=%+v actor=%+v", huiShangAction, huiShang.cellByHandle("player_mage"))
	}
	if huiTarget.Attack != 100 || huiTarget.MagicAttack != 80 {
		t.Fatalf("expected 回伤术 to restore inner injury reductions, target=%+v", huiTarget)
	}
	if len(huiShangResult.BuffInfos) != 1 || huiShangResult.BuffInfos[0].Name != "气疗" || huiShangResult.BuffInfos[0].Display != "21.png" || huiShangResult.BuffInfos[0].Description != "每回合对象恢复130气力" || huiShangResult.BuffInfos[0].Round != 3 {
		t.Fatalf("expected captured 回伤术 气疗 BuffInfo, got %+v", huiShangResult.BuffInfos)
	}
	if len(huiShangResult.ClearBuffInfos) != 2 || huiShangResult.ClearBuffInfos[0].Name != "内伤" || huiShangResult.ClearBuffInfos[1].Name != "外伤" {
		t.Fatalf("expected 回伤术 to clear 内伤/外伤, got %+v", huiShangResult.ClearBuffInfos)
	}
	qiLiaoEffect := huiShang.StatusEffects[huiTarget.Handle].Effects["气疗"]
	if qiLiaoEffect.HealAmount != 130 || qiLiaoEffect.Rounds != 2 {
		t.Fatalf("expected 回伤术 气疗 to tick once during the progressed turn, got %+v", qiLiaoEffect)
	}
	qiLiaoProbe := newRuntime(session.RoleSkill{Name: "回伤术", Level: 1, Type: "oneO"})
	qiLiaoProbeTarget := qiLiaoProbe.cellByHandle("player_ally")
	qiLiaoProbe.applyHuiShangShuQiLiaoStatusEffect(qiLiaoProbe.cellByHandle("player_mage"), qiLiaoProbeTarget)
	qiLiaoEffect = qiLiaoProbe.StatusEffects[qiLiaoProbeTarget.Handle].Effects["气疗"]
	beforeQiLiaoTick := qiLiaoProbeTarget.HP
	qiLiaoActions, qiLiaoSkip := qiLiaoProbe.resolveStatusStartActions(qiLiaoProbeTarget)
	if qiLiaoEffect.HealAmount != 130 || qiLiaoEffect.Rounds != 3 || qiLiaoSkip || len(qiLiaoActions) != 1 || qiLiaoActions[0].ActionName != "气疗" || qiLiaoProbeTarget.HP != beforeQiLiaoTick+130 {
		t.Fatalf("expected 回伤术 initial 气疗 fixed +130 tick, before=%d target=%+v effect=%+v actions=%+v skip=%t", beforeQiLiaoTick, qiLiaoProbeTarget, qiLiaoEffect, qiLiaoActions, qiLiaoSkip)
	}

	shengGuang := newRuntime(session.RoleSkill{Name: "圣光诀", Level: 1, Type: "oneO"})
	shengTarget := shengGuang.cellByHandle("player_ally")
	shengGuang.StatusEffects[shengTarget.Handle] = BattleStatusEffects{Effects: map[string]BattleStatusEffect{
		"混乱": {Name: "混乱", Rounds: 2},
		"眩晕": {Name: "眩晕", Rounds: 2, SkipTurn: true},
		"冰冻": {Name: "冰冻", Rounds: 2},
		"麻痹": {Name: "麻痹", Rounds: 2, SkipTurn: true},
	}}
	shengGuangResult := shengGuang.ProcessAction(ActionRequest{BattleID: shengGuang.BattleID, ActorHandle: "player_mage", CommandID: CommandShengGuangJue, TargetHandle: shengTarget.Handle, Round: 1, Sequence: 1})
	if shengGuangResult.ErrorCode != "" || len(shengGuangResult.Actions) == 0 {
		t.Fatalf("expected 圣光诀 action, got %+v", shengGuangResult)
	}
	shengGuangAction := shengGuangResult.Actions[0]
	if shengGuangAction.ActionName != "圣光诀" || shengGuangAction.SourceActionLabel != "w10/holyLight" || shengGuangAction.TargetHandle != shengTarget.Handle || shengGuang.cellByHandle("player_mage").MP != 440 {
		t.Fatalf("expected 圣光诀 action/MP, got action=%+v actor=%+v", shengGuangAction, shengGuang.cellByHandle("player_mage"))
	}
	if len(shengGuang.StatusEffects[shengTarget.Handle].Effects) != 0 || len(shengGuangResult.ClearBuffInfos) != 4 {
		t.Fatalf("expected 圣光诀 partial cleanse set to clear all configured statuses, effects=%+v clears=%+v", shengGuang.StatusEffects, shengGuangResult.ClearBuffInfos)
	}

	huanHun := newRuntime(session.RoleSkill{Name: "还魂术", Level: 5, Type: "oneO"})
	huanHunTarget := huanHun.cellByHandle("player_ally")
	huanHunTarget.HP = -420
	huanHunTarget.MaxHP = 2101
	huanHunResult := huanHun.ProcessAction(ActionRequest{BattleID: huanHun.BattleID, ActorHandle: "player_mage", CommandID: CommandHuanHunShu, TargetHandle: huanHunTarget.Handle, Round: 1, Sequence: 1})
	if huanHunResult.ErrorCode != "" || len(huanHunResult.Actions) == 0 {
		t.Fatalf("expected captured 还魂术 action, got %+v", huanHunResult)
	}
	huanHunAction := huanHunResult.Actions[0]
	if huanHunAction.ActionName != "还魂术" || huanHunAction.SourceActionLabel != "w10/relive" || huanHunAction.SourceMode != "1" || huanHunAction.TargetHP != 1051 || huanHunAction.TargetDead || huanHun.cellByHandle("player_mage").MP != 370 {
		t.Fatalf("expected captured 还魂术 revive/MP/source action, got action=%+v actor=%+v", huanHunAction, huanHun.cellByHandle("player_mage"))
	}
	if len(huanHunAction.RefreshInfos) != 2 || huanHunAction.RefreshInfos[0].Handle != huanHunTarget.Handle || huanHunAction.RefreshInfos[0].HP != 1051 || huanHunAction.RefreshInfos[1].Handle != "player_mage" || huanHunAction.RefreshInfos[1].MP != 370 {
		t.Fatalf("expected 还魂术 target/actor refresh order, got %+v", huanHunAction.RefreshInfos)
	}

	livingTarget := newRuntime(session.RoleSkill{Name: "还魂术", Level: 5, Type: "oneO"})
	livingResult := livingTarget.ProcessAction(ActionRequest{BattleID: livingTarget.BattleID, ActorHandle: "player_mage", CommandID: CommandHuanHunShu, TargetHandle: "player_ally", Round: 1, Sequence: 1})
	if livingResult.ErrorCode != "invalid_target" || livingTarget.cellByHandle("player_mage").MP != 500 {
		t.Fatalf("expected 还魂术 to reject living allies without MP mutation, got result=%+v actor=%+v", livingResult, livingTarget.cellByHandle("player_mage"))
	}

	enemyTarget := newRuntime(session.RoleSkill{Name: "还魂术", Level: 5, Type: "oneO"})
	enemyTarget.cellByHandle("enemy_1").HP = -420
	enemyResult := enemyTarget.ProcessAction(ActionRequest{BattleID: enemyTarget.BattleID, ActorHandle: "player_mage", CommandID: CommandHuanHunShu, TargetHandle: "enemy_1", Round: 1, Sequence: 1})
	if enemyResult.ErrorCode != "invalid_target" || enemyTarget.cellByHandle("player_mage").MP != 500 {
		t.Fatalf("expected 还魂术 to reject enemies without MP mutation, got result=%+v actor=%+v", enemyResult, enemyTarget.cellByHandle("player_mage"))
	}

	invalidTarget := newRuntime(session.RoleSkill{Name: "愈气术", Level: 5, Type: "oneO"})
	invalidResult := invalidTarget.ProcessAction(ActionRequest{BattleID: invalidTarget.BattleID, ActorHandle: "player_mage", CommandID: CommandYuQiShu, TargetHandle: "enemy_1", Round: 1, Sequence: 1})
	if invalidResult.ErrorCode != "invalid_target" || invalidTarget.cellByHandle("player_mage").MP != 500 {
		t.Fatalf("expected friendly support skills to reject enemy targets without MP mutation, got result=%+v actor=%+v", invalidResult, invalidTarget.cellByHandle("player_mage"))
	}
}

func TestCapturedMageSkillProfilesAndLevelGates(t *testing.T) {
	cases := []struct {
		name          string
		level         int
		sourceType    string
		label         string
		mpCost        int
		multiplier    float64
		hitMultiplier float64
	}{
		{name: "炎狩术", level: 5, sourceType: "oneE", label: "w10/fire2", mpCost: 80, multiplier: 1.75},
		{name: "雷爆咒", level: 1, sourceType: "all", label: "w10/thunderBombs", mpCost: 70, multiplier: 0.82, hitMultiplier: 1.5},
		{name: "雷爆咒", level: 2, sourceType: "all", label: "w10/thunderBombs", mpCost: 80, multiplier: 0.84, hitMultiplier: 1.5},
		{name: "雷爆咒", level: 3, sourceType: "all", label: "w10/thunderBombs", mpCost: 90, multiplier: 0.86, hitMultiplier: 1.5},
		{name: "雷爆咒", level: 4, sourceType: "all", label: "w10/thunderBombs", mpCost: 100, multiplier: 0.88, hitMultiplier: 1.5},
		{name: "雷爆咒", level: 5, sourceType: "all", label: "w10/thunderBombs", mpCost: 110, multiplier: 0.9, hitMultiplier: 1.5},
		{name: "火神咒", level: 5, sourceType: "all", label: "w10/fireFiend", mpCost: 260, multiplier: 1.2},
		{name: "石雨术", level: 5, sourceType: "all", label: "w10/rockRain", mpCost: 95, multiplier: 0.9},
		{name: "雷龙强袭", level: 1, sourceType: "oneE", label: "w10/thunderDrongAtk", mpCost: 125, multiplier: 2.8, hitMultiplier: 2},
	}

	for _, testCase := range cases {
		profile := sourceBattleSkillProfile(session.RoleSkill{
			Name:  testCase.name,
			Level: testCase.level,
			Type:  testCase.sourceType,
		})
		multiplierMatches := profile.DamageMultiplier >= testCase.multiplier-0.000001 && profile.DamageMultiplier <= testCase.multiplier+0.000001
		if profile.SourceType != testCase.sourceType || profile.SourceActionLabel != testCase.label || profile.MPCost != testCase.mpCost || !multiplierMatches {
			t.Fatalf("expected captured %s Lv%d profile, got %+v", testCase.name, testCase.level, profile)
		}
		if profile.DefenseType != "magic" || !profile.UseMagicAttack || profile.HitMultiplier != testCase.hitMultiplier {
			t.Fatalf("expected captured %s Lv%d magic/hit profile, got %+v", testCase.name, testCase.level, profile)
		}
	}
	if profile := sourceBattleSkillProfile(session.RoleSkill{Name: "雷爆咒", Level: 5, Type: "all"}); profile.StatusName != "麻痹" || profile.StatusRounds != 2 || profile.StatusChance != 20 || profile.StatusTickMin != 13 || profile.StatusTickMax != 15 || !profile.SkipTurn {
		t.Fatalf("expected captured 雷爆咒 Lv5 palsy profile, got %+v", profile)
	}
	if profile := sourceBattleSkillProfile(session.RoleSkill{Name: "火神咒", Level: 5, Type: "all"}); profile.StatusName != "外伤" || profile.StatusRounds != 3 || profile.StatusChance != 90 || profile.StatusTickMin != 20 || profile.StatusTickMax != 25 {
		t.Fatalf("expected captured 火神咒 Lv5 wound profile, got %+v", profile)
	}
	if profile := sourceBattleSkillProfile(session.RoleSkill{Name: "石雨术", Level: 5, Type: "all"}); profile.StatusName != "眩晕" || profile.StatusRounds != 2 || profile.StatusChance != 20 || !profile.SkipTurn {
		t.Fatalf("expected captured 石雨术 Lv5 stun profile, got %+v", profile)
	}

	for _, skill := range []session.RoleSkill{
		{Name: "炎狩术", Level: 4, Type: "oneE"},
		{Name: "雷爆咒", Level: 6, Type: "all"},
		{Name: "火神咒", Level: 6, Type: "all"},
		{Name: "石雨术", Level: 6, Type: "all"},
		{Name: "雷龙强袭", Level: 2, Type: "oneE"},
	} {
		if isSourceBattleSkillLevelCaptured(skill.Name, skill.Level) {
			t.Fatalf("expected uncaptured %s Lv%d to stay unavailable", skill.Name, skill.Level)
		}
		for _, command := range sourceBattleCommandDefinitions([]session.RoleSkill{skill}) {
			if command.Label == skill.Name {
				t.Fatalf("expected uncaptured %s Lv%d to be omitted from commands, got %+v", skill.Name, skill.Level, command)
			}
		}
	}
}

func TestSkillLevelTableLinksMasterSkillToCapturedMageLevel(t *testing.T) {
	profile, ok := sourceBattleSkillProfileFromConfig("雷爆咒", 4)
	if !ok {
		t.Fatal("expected 雷爆咒 master skill row")
	}
	if sourceBattleSkillCommandIDFromConfig("雷爆咒") != CommandLeiBaoZhou {
		t.Fatalf("expected 雷爆咒 to keep its command skill id")
	}
	if profile.SourceType != "all" || profile.SourceActionLabel != "w10/thunderBombs" || profile.MPCost != 100 || profile.DamageMultiplier != 0.88 {
		t.Fatalf("expected 雷爆咒 Lv4 to resolve linked skill-level values, got %+v", profile)
	}
	if !sourceBattleSkillLevelExists("雷爆咒", 4) {
		t.Fatal("expected 雷爆咒 Lv4 skill-level row")
	}
	lv5, ok := sourceBattleSkillProfileFromConfig("雷爆咒", 5)
	if !ok {
		t.Fatal("expected 雷爆咒 Lv5 skill-level row from 20260724 skillInfo capture")
	}
	if lv5.SourceType != "all" || lv5.SourceActionLabel != "w10/thunderBombs" || lv5.MPCost != 110 || lv5.DamageMultiplier != 0.9 {
		t.Fatalf("expected 雷爆咒 Lv5 to resolve linked skill-level values, got %+v", lv5)
	}
	if !sourceBattleSkillLevelExists("雷爆咒", 5) {
		t.Fatal("expected 雷爆咒 Lv5 skill-level row")
	}
	if sourceBattleSkillLevelExists("雷爆咒", 6) {
		t.Fatal("unexpected unconfirmed 雷爆咒 Lv6 skill-level row")
	}
	for _, testCase := range []struct {
		name       string
		label      string
		mpCost     int
		multiplier float64
	}{
		{name: "火神咒", label: "w10/fireFiend", mpCost: 260, multiplier: 1.2},
		{name: "石雨术", label: "w10/rockRain", mpCost: 95, multiplier: 0.9},
	} {
		profile, ok := sourceBattleSkillProfileFromConfig(testCase.name, 5)
		if !ok || profile.SourceType != "all" || profile.SourceActionLabel != testCase.label || profile.MPCost != testCase.mpCost || profile.DamageMultiplier != testCase.multiplier {
			t.Fatalf("expected %s Lv5 source table row, got %+v found=%t", testCase.name, profile, ok)
		}
		if !sourceBattleSkillLevelExists(testCase.name, 5) || sourceBattleSkillLevelExists(testCase.name, 6) {
			t.Fatalf("expected %s captured levels 1-5 only", testCase.name)
		}
	}
}

func TestProcessActionUsesCapturedMageSkills(t *testing.T) {
	newRuntime := func(skill session.RoleSkill, enemies ...CellInfoPush) *Runtime {
		cells := []CellInfoPush{
			{
				BattleID:    "battle-mage-" + skill.Name,
				Handle:      "player_mage",
				Camp:        CampTeam,
				HP:          600,
				MaxHP:       600,
				MP:          200,
				MaxMP:       200,
				Attack:      10,
				MagicAttack: 100,
				Hit:         100,
			},
		}
		cells = append(cells, enemies...)
		return &Runtime{
			BattleID:         "battle-mage-" + skill.Name,
			Round:            1,
			Phase:            PhaseCommand,
			ActiveHandle:     "player_mage",
			nextSequence:     1,
			ConsumedSequence: map[int]bool{},
			DefendingHandles: map[string]bool{},
			StoredPower:      map[string]int{},
			RoleSkills:       []session.RoleSkill{skill},
			Cells:            cells,
		}
	}
	newEnemy := func(handle string, magicDefense int) CellInfoPush {
		return CellInfoPush{
			BattleID:   "battle-mage",
			Handle:     handle,
			Camp:       CampEnemy,
			HP:         1000,
			MaxHP:      1000,
			Attack:     1,
			MgcDefense: magicDefense,
		}
	}

	flame := newRuntime(session.RoleSkill{Name: "炎狩术", Level: 5, Type: "oneE"}, newEnemy("enemy_fire", 20))
	flameResult := flame.ProcessAction(ActionRequest{BattleID: flame.BattleID, ActorHandle: "player_mage", CommandID: CommandYanShouShu, TargetHandle: "enemy_fire", Round: 1, Sequence: 1})
	if flameResult.ErrorCode != "" || len(flameResult.Actions) == 0 {
		t.Fatalf("expected 炎狩术 action, got %+v", flameResult)
	}
	flameAction := flameResult.Actions[0]
	if flameAction.SourceActionLabel != "w10/fire2" || flameAction.Damage != 165 || flameAction.TargetHP != 835 || flame.cellByHandle("player_mage").MP != 120 {
		t.Fatalf("expected captured 炎狩术 magic action, got %+v", flameAction)
	}

	thunder := newRuntime(session.RoleSkill{Name: "雷爆咒", Level: 2, Type: "all"}, newEnemy("enemy_thunder_1", 20), newEnemy("enemy_thunder_2", 30))
	thunderResult := thunder.ProcessAction(ActionRequest{BattleID: thunder.BattleID, ActorHandle: "player_mage", CommandID: CommandLeiBaoZhou, TargetHandle: "enemy_thunder_1", Round: 1, Sequence: 1})
	if thunderResult.ErrorCode != "" || len(thunderResult.Actions) == 0 {
		t.Fatalf("expected 雷爆咒 action, got %+v", thunderResult)
	}
	thunderAction := thunderResult.Actions[0]
	if thunderAction.TargetHandle != "all" || thunderAction.SourceActionLabel != "w10/thunderBombs" || len(thunderAction.TargetActionResults) != 2 || thunderAction.Damage != 74 || thunder.cellByHandle("enemy_thunder_1").HP != 926 || thunder.cellByHandle("enemy_thunder_2").HP != 931 || thunder.cellByHandle("player_mage").MP != 120 {
		t.Fatalf("expected captured 雷爆咒 all-target action and one MP deduction, got %+v", thunderAction)
	}
	dragon := newRuntime(session.RoleSkill{Name: "雷龙强袭", Level: 1, Type: "oneE"}, newEnemy("enemy_dragon", 20))
	insufficient := dragon.ProcessAction(ActionRequest{BattleID: dragon.BattleID, ActorHandle: "player_mage", CommandID: CommandLeiLongQiangXi, TargetHandle: "enemy_dragon", Round: 1, Sequence: 1})
	if insufficient.ErrorCode != "insufficient_power" {
		t.Fatalf("expected 雷龙强袭 to require 2 soul power, got %+v", insufficient)
	}
	dragon.StoredPower["player_mage"] = 2
	dragonResult := dragon.ProcessAction(ActionRequest{BattleID: dragon.BattleID, ActorHandle: "player_mage", CommandID: CommandLeiLongQiangXi, TargetHandle: "enemy_dragon", Round: 1, Sequence: 1})
	if dragonResult.ErrorCode != "" || len(dragonResult.Actions) == 0 {
		t.Fatalf("expected 雷龙强袭 action, got %+v", dragonResult)
	}
	dragonAction := dragonResult.Actions[0]
	if dragonAction.SourceActionLabel != "w10/thunderDrongAtk" || dragonAction.Damage != 270 || dragonAction.TargetHP != 730 || dragon.cellByHandle("player_mage").MP != 75 {
		t.Fatalf("expected captured 雷龙强袭 magic action, got %+v", dragonAction)
	}

	unsupported := newRuntime(session.RoleSkill{Name: "炎狩术", Level: 4, Type: "oneE"}, newEnemy("enemy_invalid", 20))
	unsupportedResult := unsupported.ProcessAction(ActionRequest{BattleID: unsupported.BattleID, ActorHandle: "player_mage", CommandID: CommandYanShouShu, TargetHandle: "enemy_invalid", Round: 1, Sequence: 1})
	if unsupportedResult.ErrorCode != "unsupported_command" {
		t.Fatalf("expected uncaptured 炎狩术 Lv4 to be rejected, got %+v", unsupportedResult)
	}
}

func TestCapturedMageAllTargetStatusEffects(t *testing.T) {
	newRuntime := func(battleID string, skill session.RoleSkill) (*Runtime, *CellInfoPush, *CellInfoPush) {
		runtime := &Runtime{
			BattleID:         battleID,
			Round:            1,
			DefendingHandles: map[string]bool{},
			StoredPower:      map[string]int{},
			RoleSkills:       []session.RoleSkill{skill},
			Cells: []CellInfoPush{
				{BattleID: battleID, Handle: "player_mage", Camp: CampTeam, HP: 600, MaxHP: 600, MP: 500, MaxMP: 500, MagicAttack: 100, Hit: 100},
				{BattleID: battleID, Handle: "enemy_mage", Camp: CampEnemy, HP: 1000, MaxHP: 1000, MgcDefense: 20, Hit: 100},
			},
		}
		return runtime, runtime.cellByHandle("player_mage"), runtime.cellByHandle("enemy_mage")
	}
	findAppliedStatus := func(skill session.RoleSkill, commandID string, status string) (*Runtime, *CellInfoPush) {
		for index := 0; index < 300; index += 1 {
			runtime, actor, target := newRuntime(fmt.Sprintf("battle-mage-%s-%d", skill.Name, index), skill)
			runtime.resolveAllTargetAttack(actor, []*CellInfoPush{target}, commandID)
			if effect, ok := runtime.StatusEffects[target.Handle].Effects[status]; ok {
				if effect.SourceHandle != actor.Handle || effect.SourceSkill != skill.Name {
					t.Fatalf("expected %s source ownership, got %+v", skill.Name, effect)
				}
				return runtime, target
			}
		}
		return nil, nil
	}

	fireRuntime, fireTarget := findAppliedStatus(session.RoleSkill{Name: "火神咒", Level: 5, Type: "all"}, CommandHuoShenZhou, "外伤")
	if fireRuntime == nil {
		t.Fatal("expected deterministic 火神咒 external-injury application")
	}
	fireEffect := fireRuntime.StatusEffects[fireTarget.Handle].Effects["外伤"]
	if fireEffect.Rounds != 3 || fireEffect.SourceAttack != 100 || fireEffect.TickMinPercent != 20 || fireEffect.TickMaxPercent != 25 || fireEffect.SkipTurn {
		t.Fatalf("expected captured 火神咒 wound values, got %+v", fireEffect)
	}
	if len(fireRuntime.PendingBuffInfos) != 1 || fireRuntime.PendingBuffInfos[0].Description != "每回合减少对象20~25气力" {
		t.Fatalf("expected 火神咒 BuffInfo damage range, got %+v", fireRuntime.PendingBuffInfos)
	}
	fireTicks, fireSkipped := fireRuntime.resolveStatusStartActions(fireTarget)
	if fireSkipped || len(fireTicks) != 1 || fireTicks[0].ActionName != "外伤" || fireTicks[0].Damage < 20 || fireTicks[0].Damage > 25 {
		t.Fatalf("expected 火神咒 magic external-injury tick, actions=%+v skip=%t", fireTicks, fireSkipped)
	}

	stoneRuntime, stoneTarget := findAppliedStatus(session.RoleSkill{Name: "石雨术", Level: 5, Type: "all"}, CommandShiYuShu, "眩晕")
	if stoneRuntime == nil {
		t.Fatal("expected deterministic 石雨术 stun application")
	}
	stoneTicks, stoneSkipped := stoneRuntime.resolveStatusStartActions(stoneTarget)
	if !stoneSkipped || len(stoneTicks) != 1 || stoneTicks[0].ActionName != "眩晕" || stoneTicks[0].SourceActionLabel != "yun" {
		t.Fatalf("expected 石雨术 captured stun skip action, actions=%+v skip=%t", stoneTicks, stoneSkipped)
	}

	thunderRuntime, thunderTarget := findAppliedStatus(session.RoleSkill{Name: "雷爆咒", Level: 5, Type: "all"}, CommandLeiBaoZhou, "麻痹")
	if thunderRuntime == nil {
		t.Fatal("expected deterministic 雷爆咒 palsy application")
	}
	thunderEffect := thunderRuntime.StatusEffects[thunderTarget.Handle].Effects["麻痹"]
	if thunderEffect.Rounds != 2 || thunderEffect.SourceAttack != 100 || thunderEffect.TickMinPercent != 13 || thunderEffect.TickMaxPercent != 15 || !thunderEffect.SkipTurn {
		t.Fatalf("expected captured 雷爆咒 palsy values, got %+v", thunderEffect)
	}
	if len(thunderRuntime.PendingBuffInfos) != 1 || thunderRuntime.PendingBuffInfos[0].Description != "眩晕&0;并在每回合造成13~15点伤害" {
		t.Fatalf("expected 雷爆咒 BuffInfo damage range, got %+v", thunderRuntime.PendingBuffInfos)
	}
	thunderTicks, thunderSkipped := thunderRuntime.resolveStatusStartActions(thunderTarget)
	if !thunderSkipped || len(thunderTicks) != 1 || thunderTicks[0].ActionName != "麻痹" || thunderTicks[0].Damage < 13 || thunderTicks[0].Damage > 15 {
		t.Fatalf("expected 雷爆咒 magic palsy tick and skip action, actions=%+v skip=%t", thunderTicks, thunderSkipped)
	}

	powerRuntime, powerActor, powerTarget := newRuntime("battle-mage-fire-power", session.RoleSkill{Name: "火神咒", Level: 5, Type: "all"})
	powerRuntime.Phase = PhaseCommand
	powerRuntime.ActiveHandle = powerActor.Handle
	powerRuntime.nextSequence = 1
	powerRuntime.ConsumedSequence = map[int]bool{}
	if result := powerRuntime.ProcessAction(ActionRequest{BattleID: powerRuntime.BattleID, ActorHandle: powerActor.Handle, CommandID: CommandHuoShenZhou, TargetHandle: powerTarget.Handle, Round: 1, Sequence: 1}); result.ErrorCode != "insufficient_power" {
		t.Fatalf("expected 火神咒 to require two stored power before all-target attack, got %+v", result)
	}
}

func TestCapturedMageAdditionalStaffSkillProfiles(t *testing.T) {
	cases := []struct {
		name          string
		level         int
		commandID     string
		sourceType    string
		actionLabel   string
		mpCost        int
		multiplier    float64
		hitMultiplier float64
	}{
		{name: "赤焰魔咒", level: 2, commandID: CommandChiYanMoZhou, sourceType: "oneE", actionLabel: "w10/bombFire", mpCost: 65, multiplier: 0.85},
		{name: "赤焰魔咒", level: 3, commandID: CommandChiYanMoZhou, sourceType: "oneE", actionLabel: "w10/bombFire", mpCost: 75, multiplier: 0.9},
		{name: "赤焰魔咒", level: 4, commandID: CommandChiYanMoZhou, sourceType: "oneE", actionLabel: "w10/bombFire", mpCost: 85, multiplier: 0.95},
		{name: "赤焰魔咒", level: 5, commandID: CommandChiYanMoZhou, sourceType: "oneE", actionLabel: "w10/bombFire", mpCost: 95, multiplier: 1},
		{name: "雷击", level: 3, commandID: CommandLeiJi, sourceType: "oneE", actionLabel: "w10/thunderMagicAtk", mpCost: 75, multiplier: 1.2, hitMultiplier: 1.5},
		{name: "雷击", level: 4, commandID: CommandLeiJi, sourceType: "oneE", actionLabel: "w10/thunderMagicAtk", mpCost: 85, multiplier: 1.25, hitMultiplier: 1.5},
		{name: "雷击", level: 5, commandID: CommandLeiJi, sourceType: "oneE", actionLabel: "w10/thunderMagicAtk", mpCost: 95, multiplier: 1.3, hitMultiplier: 1.5},
		{name: "魔障术", level: 2, commandID: CommandMoZhangShu, sourceType: "own", actionLabel: "w10/magicObstacle", mpCost: 90, multiplier: 0},
		{name: "魔障术", level: 3, commandID: CommandMoZhangShu, sourceType: "own", actionLabel: "w10/magicObstacle", mpCost: 110, multiplier: 0},
		{name: "魔障术", level: 4, commandID: CommandMoZhangShu, sourceType: "own", actionLabel: "w10/magicObstacle", mpCost: 130, multiplier: 0},
		{name: "魔障术", level: 5, commandID: CommandMoZhangShu, sourceType: "own", actionLabel: "w10/magicObstacle", mpCost: 150, multiplier: 0},
	}

	for _, testCase := range cases {
		profile := sourceBattleSkillProfile(session.RoleSkill{Name: testCase.name, Level: testCase.level, Type: testCase.sourceType})
		if sourceBattleSkillCommandIDFromConfig(testCase.name) != testCase.commandID {
			t.Fatalf("expected %s to retain configured command id", testCase.name)
		}
		if profile.SourceType != testCase.sourceType || profile.SourceActionLabel != testCase.actionLabel || profile.MPCost != testCase.mpCost || profile.DamageMultiplier != testCase.multiplier || profile.HitMultiplier != testCase.hitMultiplier {
			t.Fatalf("expected captured %s Lv%d profile, got %+v", testCase.name, testCase.level, profile)
		}
	}

	if profile := sourceBattleSkillProfile(session.RoleSkill{Name: "赤焰魔咒", Level: 2, Type: "oneE"}); profile.StatusName != "诅咒" || profile.StatusDisplay != "780.png" || profile.StatusRounds != 2 || profile.StatusChance != 85 {
		t.Fatalf("expected captured 赤焰魔咒 curse profile, got %+v", profile)
	}
	if profile := sourceBattleSkillProfile(session.RoleSkill{Name: "赤焰魔咒", Level: 5, Type: "oneE"}); profile.StatusChance != 100 {
		t.Fatalf("expected 赤焰魔咒 Lv5 curse chance 100, got %+v", profile)
	}
	if profile := sourceBattleSkillProfile(session.RoleSkill{Name: "雷击", Level: 3, Type: "oneE"}); profile.StatusName != "迟钝" || profile.StatusRounds != 2 || profile.StatusChance != 60 || profile.StatusHitDodgePercent != 30 {
		t.Fatalf("expected captured 雷击 slowness profile, got %+v", profile)
	}
	if profile := sourceBattleSkillProfile(session.RoleSkill{Name: "雷击", Level: 5, Type: "oneE"}); profile.StatusChance != 70 || profile.StatusHitDodgePercent != 40 {
		t.Fatalf("expected 雷击 Lv5 slowness chance/percent, got %+v", profile)
	}
	if got := sourceMoZhangShuDamageToMPPercent(2); got != 35 {
		t.Fatalf("expected 魔障术 Lv2 barrier percent 35, got %d", got)
	}
	if got := sourceMoZhangShuDamageToMPPercent(5); got != 50 {
		t.Fatalf("expected 魔障术 Lv5 barrier percent 50, got %d", got)
	}
}

func TestCapturedMageAdditionalStaffStatuses(t *testing.T) {
	newRuntime := func(battleID string, skill session.RoleSkill) (*Runtime, *CellInfoPush, *CellInfoPush) {
		runtime := &Runtime{
			BattleID:         battleID,
			Round:            1,
			nextSequence:     1,
			DefendingHandles: map[string]bool{},
			StoredPower:      map[string]int{},
			RoleSkills:       []session.RoleSkill{skill},
			Cells: []CellInfoPush{
				{BattleID: battleID, Handle: "player_mage", Camp: CampTeam, HP: 600, MaxHP: 600, MP: 200, MaxMP: 200, MagicAttack: 100, Hit: 100},
				{BattleID: battleID, Handle: "enemy_mage", Camp: CampEnemy, HP: 1000, MaxHP: 1000, MgcDefense: 20, Hit: 100, Dog: 50},
			},
		}
		return runtime, runtime.cellByHandle("player_mage"), runtime.cellByHandle("enemy_mage")
	}

	var curseRuntime *Runtime
	for index := 0; index < 200; index += 1 {
		runtime, actor, target := newRuntime(fmt.Sprintf("battle-chi-yan-%d", index), session.RoleSkill{Name: "赤焰魔咒", Level: 2, Type: "oneE"})
		runtime.setStoredPower(target.Handle, 2)
		action := runtime.resolveAttack(actor, target, CommandChiYanMoZhou)
		if len(runtime.PendingBuffInfos) != 1 {
			continue
		}
		if action.SourceActionLabel != "w10/bombFire" || action.Damage != 75 || target.HP != 925 || actor.MP != 135 {
			t.Fatalf("expected captured 赤焰魔咒 action, got %+v", action)
		}
		curse := runtime.PendingBuffInfos[0]
		if curse.Name != "诅咒" || curse.Display != "780.png" || curse.Round != 2 || curse.Description != "作用时间内无法增加魂元。" {
			t.Fatalf("expected captured curse BuffInfo, got %+v", curse)
		}
		curseRuntime = runtime
		break
	}
	if curseRuntime == nil {
		t.Fatal("expected to find deterministic 赤焰魔咒 curse roll")
	}
	curseRuntime.setStoredPower("enemy_mage", 2)
	curseRuntime.setStoredPower("enemy_mage", 3)
	if curseRuntime.StoredPower["enemy_mage"] != 2 {
		t.Fatalf("expected curse to block soul gain, got %+v", curseRuntime.StoredPower)
	}
	curseRuntime.setStoredPower("enemy_mage", 0)
	if curseRuntime.StoredPower["enemy_mage"] != 0 {
		t.Fatalf("expected curse to allow soul consumption, got %+v", curseRuntime.StoredPower)
	}

	var slowRuntime *Runtime
	for index := 0; index < 200; index += 1 {
		runtime, actor, target := newRuntime(fmt.Sprintf("battle-lei-ji-%d", index), session.RoleSkill{Name: "雷击", Level: 3, Type: "oneE"})
		action := runtime.resolveAttack(actor, target, CommandLeiJi)
		if len(runtime.PendingBuffInfos) != 1 {
			continue
		}
		if action.SourceActionLabel != "w10/thunderMagicAtk" || action.Damage != 110 || target.HP != 890 || actor.MP != 125 {
			t.Fatalf("expected captured 雷击 action, got %+v", action)
		}
		slow := runtime.PendingBuffInfos[0]
		if slow.Name != "迟钝" || slow.Display != "16.png" || slow.Round != 2 || slow.Description != "降低对象30点命中和15点回避" || target.Hit != 70 || target.Dog != 35 {
			t.Fatalf("expected captured 雷击 slowness BuffInfo, got buff=%+v target=%+v", slow, target)
		}
		slowRuntime = runtime
		break
	}
	if slowRuntime == nil {
		t.Fatal("expected to find deterministic 雷击 slowness roll")
	}
}

func TestMagicObstacleUsesCapturedBarrierDamageAndCountdown(t *testing.T) {
	actor := &CellInfoPush{Handle: "player_mage", Camp: CampTeam, HP: 1000, MaxHP: 1000, MP: 2000, MaxMP: 2000}
	runtime := &Runtime{
		BattleID:         "battle-magic-obstacle",
		Round:            1,
		nextSequence:     1,
		StatusEffects:    map[string]BattleStatusEffects{},
		StoredPower:      map[string]int{},
		DefendingHandles: map[string]bool{},
		Cells:            []CellInfoPush{*actor},
	}
	actor = runtime.cellByHandle("player_mage")
	buff := runtime.applyMagicBarrierStatusEffect(actor)
	if buff.Name != "法术屏障" || buff.Display != "28.png" || buff.Round != 5 || buff.Description != "作用时间内气力损失量的35%以精力来代替" {
		t.Fatalf("expected captured magic barrier BuffInfo, got %+v", buff)
	}
	if damage, mpDamage := runtime.applyTargetHPDamage(actor, 132); damage != 86 || mpDamage != 46 || actor.HP != 914 || actor.MP != 1954 {
		t.Fatalf("expected captured 132 damage split to HP86/MP46, got damage=%d mpDamage=%d actor=%+v", damage, mpDamage, actor)
	}
	actor.HP = 1000
	actor.MP = 20
	if damage, mpDamage := runtime.applyTargetHPDamage(actor, 132); damage != 112 || mpDamage != 20 || actor.HP != 888 || actor.MP != 0 {
		t.Fatalf("expected insufficient MP remainder to stay HP damage, got damage=%d mpDamage=%d actor=%+v", damage, mpDamage, actor)
	}

	runtime.applyMagicBarrierStatusEffect(actor)
	runtime.PendingBuffInfos = nil
	for expectedRound := 4; expectedRound >= 0; expectedRound -= 1 {
		actions, skipTurn := runtime.resolveStatusStartActions(actor)
		if skipTurn || len(actions) != 1 || actions[0].ActionName != "法术屏障" || actions[0].SourceActionLabel != "battleStand" {
			t.Fatalf("expected magic barrier battleStand countdown action, actions=%+v skip=%v", actions, skipTurn)
		}
		buffs := runtime.consumePendingBuffInfos()
		if len(buffs) != 1 || buffs[0].Name != "法术屏障" || buffs[0].Round != expectedRound {
			t.Fatalf("expected magic barrier round %d update, got %+v", expectedRound, buffs)
		}
	}
	clears := runtime.consumePendingClearBuffInfos()
	if len(clears) != 1 || clears[0].TargetHandle != actor.Handle || clears[0].Name != "法术屏障" {
		t.Fatalf("expected magic barrier clear on round zero, got %+v", clears)
	}
}

func TestLeiHunZhanRequiresCapturedSoulPowerAndUsesThunderSoulAtk(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-leihun",
		Round:            1,
		Phase:            PhaseCommand,
		ActiveHandle:     "player_21424",
		nextSequence:     1,
		ConsumedSequence: map[int]bool{},
		DefendingHandles: map[string]bool{},
		StoredPower:      map[string]int{},
		RoleSkills: []session.RoleSkill{
			{
				Name:        "奥义.雷魂斩",
				Level:       1,
				Type:        "oneE",
				Description: "f_s_奥义.雷魂斩^00ccff&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@24&4@<font color='#00cc00'>特殊发动条件:需要3格魂元</font><br>提升240%的物理伤害",
			},
		},
		Cells: []CellInfoPush{
			{
				BattleID: "battle-leihun",
				Handle:   "player_21424",
				Camp:     CampTeam,
				HP:       500,
				MaxHP:    500,
				MP:       100,
				MaxMP:    100,
				Attack:   100,
				Defense:  0,
				Hit:      100,
				Fat:      0,
			},
			{
				BattleID: "battle-leihun",
				Handle:   "enemy_1",
				Camp:     CampEnemy,
				HP:       1000,
				MaxHP:    1000,
				Attack:   1,
				Defense:  0,
				Dog:      0,
			},
		},
	}

	insufficient := runtime.ProcessAction(ActionRequest{
		BattleID:     "battle-leihun",
		ActorHandle:  "player_21424",
		CommandID:    CommandLeiHunZhan,
		TargetHandle: "enemy_1",
		Round:        1,
		Sequence:     1,
	})
	if insufficient.ErrorCode != "insufficient_power" {
		t.Fatalf("expected 奥义.雷魂斩 to require 3 soul power, got %+v", insufficient)
	}

	runtime.StoredPower["player_21424"] = 3
	result := runtime.ProcessAction(ActionRequest{
		BattleID:     "battle-leihun",
		ActorHandle:  "player_21424",
		CommandID:    CommandLeiHunZhan,
		TargetHandle: "enemy_1",
		Round:        1,
		Sequence:     1,
	})
	if result.ErrorCode != "" {
		t.Fatalf("expected 奥义.雷魂斩 to be accepted with 3 soul power, got %+v", result)
	}
	if len(result.Actions) < 1 {
		t.Fatalf("expected 奥义.雷魂斩 action, got %+v", result.Actions)
	}
	action := result.Actions[0]
	if action.ActionName != "奥义.雷魂斩" || action.SourceMode != "1" || action.SourceActionLabel != "w8/thunderSoulAtk" {
		t.Fatalf("expected captured 奥义.雷魂斩 action label, got %+v", action)
	}
	if action.Damage != 340 || action.TargetHP != 660 {
		t.Fatalf("expected 奥义.雷魂斩 Lv1 captured multiplier without soul power damage bonus, got %+v", action)
	}
	actor := runtime.cellByHandle("player_21424")
	if actor == nil || actor.MP != 76 {
		t.Fatalf("expected 奥义.雷魂斩 to consume MP 24, got actor=%+v", actor)
	}
}

func TestAoYiLiuHeGunFaRequiresCapturedSoulPowerAndUsesLiuhe(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-liuhe",
		Round:            1,
		Phase:            PhaseCommand,
		ActiveHandle:     "player_21424",
		nextSequence:     1,
		ConsumedSequence: map[int]bool{},
		DefendingHandles: map[string]bool{},
		StoredPower:      map[string]int{},
		RoleSkills: []session.RoleSkill{
			{
				Name:        "奥义.六合棍法",
				Level:       1,
				Type:        "oneE",
				Description: "f_s_奥义.六合棍法^00ccff&9@单体·攻击&8@战士 &10@棍&22@战斗&2@24&4@<font color='#00cc00'>特殊发动条件:需要3格魂元</font><br>提升210%的物理伤害&0;进攻时候增加300%的命中",
			},
		},
		Cells: []CellInfoPush{
			{
				BattleID: "battle-liuhe",
				Handle:   "player_21424",
				Camp:     CampTeam,
				HP:       500,
				MaxHP:    500,
				MP:       100,
				MaxMP:    100,
				Attack:   100,
				Defense:  0,
				Hit:      100,
				Fat:      0,
			},
			{
				BattleID: "battle-liuhe",
				Handle:   "enemy_1",
				Camp:     CampEnemy,
				HP:       1000,
				MaxHP:    1000,
				Attack:   1,
				Defense:  0,
				Dog:      0,
			},
		},
	}

	insufficient := runtime.ProcessAction(ActionRequest{
		BattleID:     "battle-liuhe",
		ActorHandle:  "player_21424",
		CommandID:    CommandAoYiLiuHeGunFa,
		TargetHandle: "enemy_1",
		Round:        1,
		Sequence:     1,
	})
	if insufficient.ErrorCode != "insufficient_power" {
		t.Fatalf("expected 奥义.六合棍法 to require 3 soul power, got %+v", insufficient)
	}

	runtime.StoredPower["player_21424"] = 3
	result := runtime.ProcessAction(ActionRequest{
		BattleID:     "battle-liuhe",
		ActorHandle:  "player_21424",
		CommandID:    CommandAoYiLiuHeGunFa,
		TargetHandle: "enemy_1",
		Round:        1,
		Sequence:     1,
	})
	if result.ErrorCode != "" {
		t.Fatalf("expected 奥义.六合棍法 to be accepted with 3 soul power, got %+v", result)
	}
	if len(result.Actions) < 1 {
		t.Fatalf("expected 奥义.六合棍法 action, got %+v", result.Actions)
	}
	action := result.Actions[0]
	if action.ActionName != "奥义.六合棍法" || action.SourceMode != "1" || action.SourceActionLabel != "w11/liuhe" {
		t.Fatalf("expected captured 奥义.六合棍法 action label, got %+v", action)
	}
	if action.Damage != 310 || action.TargetHP != 690 {
		t.Fatalf("expected 奥义.六合棍法 Lv1 captured multiplier without soul power damage bonus, got %+v", action)
	}
	actor := runtime.cellByHandle("player_21424")
	if actor == nil || actor.MP != 76 {
		t.Fatalf("expected 奥义.六合棍法 to consume MP 24, got actor=%+v", actor)
	}
}

func TestAoYiAnShaZheRequiresCapturedSoulPowerAndUsesAssassinate(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-assassinate",
		Round:            1,
		Phase:            PhaseCommand,
		ActiveHandle:     "player_21432",
		nextSequence:     1,
		ConsumedSequence: map[int]bool{},
		DefendingHandles: map[string]bool{},
		StoredPower:      map[string]int{},
		RoleSkills: []session.RoleSkill{
			{
				Name:        "奥义.暗杀者",
				Level:       1,
				Type:        "oneE",
				Description: "f_s_奥义.暗杀者^00ccff&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@26&4@<font color='#00cc00'>特殊发动条件:需要3格魂元</font><br>提升180%的物理伤害",
			},
		},
		Cells: []CellInfoPush{
			{
				BattleID: "battle-assassinate",
				Handle:   "player_21432",
				Camp:     CampTeam,
				HP:       500,
				MaxHP:    500,
				MP:       100,
				MaxMP:    100,
				Attack:   100,
				Defense:  0,
				Hit:      100,
				Fat:      0,
			},
			{
				BattleID: "battle-assassinate",
				Handle:   "enemy_1",
				Camp:     CampEnemy,
				HP:       1000,
				MaxHP:    1000,
				Attack:   1,
				Defense:  0,
				Dog:      0,
			},
		},
	}

	insufficient := runtime.ProcessAction(ActionRequest{
		BattleID:     "battle-assassinate",
		ActorHandle:  "player_21432",
		CommandID:    CommandAoYiAnShaZhe,
		TargetHandle: "enemy_1",
		Round:        1,
		Sequence:     1,
	})
	if insufficient.ErrorCode != "insufficient_power" {
		t.Fatalf("expected 奥义.暗杀者 to require 3 soul power, got %+v", insufficient)
	}

	runtime.StoredPower["player_21432"] = 3
	result := runtime.ProcessAction(ActionRequest{
		BattleID:     "battle-assassinate",
		ActorHandle:  "player_21432",
		CommandID:    CommandAoYiAnShaZhe,
		TargetHandle: "enemy_1",
		Round:        1,
		Sequence:     1,
	})
	if result.ErrorCode != "" {
		t.Fatalf("expected 奥义.暗杀者 to be accepted with 3 soul power, got %+v", result)
	}
	if len(result.Actions) < 1 {
		t.Fatalf("expected 奥义.暗杀者 action, got %+v", result.Actions)
	}
	action := result.Actions[0]
	if action.ActionName != "奥义.暗杀者" || action.SourceMode != "1" || action.SourceActionLabel != "w3/assassinate" {
		t.Fatalf("expected captured 奥义.暗杀者 action label, got %+v", action)
	}
	if action.Damage != 280 || action.TargetHP != 720 {
		t.Fatalf("expected 奥义.暗杀者 Lv1 captured multiplier without soul power damage bonus, got %+v", action)
	}
	actor := runtime.cellByHandle("player_21432")
	if actor == nil || actor.MP != 74 {
		t.Fatalf("expected 奥义.暗杀者 to consume MP 26, got actor=%+v", actor)
	}
}

func TestKuangBaoUsesCapturedSelfBuffFormula(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-kuangbao",
		Round:            1,
		Phase:            PhaseCommand,
		ActiveHandle:     "player_21424",
		nextSequence:     1,
		ConsumedSequence: map[int]bool{},
		DefendingHandles: map[string]bool{},
		StatusEffects:    map[string]BattleStatusEffects{},
		RoleSkills: []session.RoleSkill{
			{
				Name:        "狂爆",
				Level:       1,
				Type:        "技能·单刀系",
				Description: "f_s_狂爆^5BC46D&9@单体·状态&8@战士 &10@单刀&22@战斗&2@15&4@3回合内物理攻击力翻倍&0;并降低100%的物理防御",
			},
		},
		Cells: []CellInfoPush{
			{
				BattleID: "battle-kuangbao",
				Handle:   "player_21424",
				Camp:     CampTeam,
				HP:       225,
				MaxHP:    225,
				MP:       64,
				MaxMP:    64,
				Attack:   50,
				Defense:  20,
				Hit:      100,
			},
			{
				BattleID: "battle-kuangbao",
				Handle:   "enemy_1",
				Camp:     CampEnemy,
				HP:       500,
				MaxHP:    500,
				Attack:   30,
				Defense:  0,
				Hit:      100,
			},
		},
	}

	result := runtime.ProcessAction(ActionRequest{
		BattleID:     "battle-kuangbao",
		ActorHandle:  "player_21424",
		CommandID:    CommandKuangBao,
		TargetHandle: "player_21424",
		Round:        1,
		Sequence:     1,
	})
	if result.ErrorCode != "" {
		t.Fatalf("expected 狂爆 to be accepted, got %+v", result)
	}
	actor := runtime.cellByHandle("player_21424")
	if actor == nil || actor.MP != 49 {
		t.Fatalf("expected 狂爆 to consume source MP 15, got actor=%+v", actor)
	}
	if !runtime.hasKuangBao("player_21424") || runtime.StatusEffects["player_21424"].KuangBaoRounds != 2 {
		t.Fatalf("expected 狂爆 status to advance to 2 rounds for the next startCommand, got %+v", runtime.StatusEffects)
	}
	if len(result.Actions) < 1 || result.Actions[0].ActionName != "狂爆" || result.Actions[0].SourceActionLabel != "w8/kb" {
		t.Fatalf("expected captured 狂爆 self action, got %+v", result.Actions)
	}
	if len(result.BuffInfos) != 1 || result.BuffInfos[0].Name != "热血" || result.BuffInfos[0].Display != "24.png" || result.BuffInfos[0].TargetHandle != "player_21424" {
		t.Fatalf("expected captured 热血 BuffInfo for actor overlay, got %+v", result.BuffInfos)
	}

	target := runtime.cellByHandle("enemy_1")
	target.HP = 500
	attack := runtime.resolveAttack(actor, target, CommandNormalAttack)
	if attack.Damage != 100 || attack.SourceActionLabel != "nomalAtk" || runtime.StatusEffects["player_21424"].KuangBaoRounds != 2 {
		t.Fatalf("expected 狂爆 to double attack without consuming rounds on hit, got action=%+v effects=%+v", attack, runtime.StatusEffects)
	}

	defenseProbe := runtime.resolveAttack(target, actor, CommandEnemyAttack)
	if defenseProbe.Damage != 30 {
		t.Fatalf("expected enemy damage after 狂爆 defense reduction without stored power bonus, got %+v", defenseProbe)
	}

	runtime.advanceKuangBaoRound("player_21424")
	runtime.advanceKuangBaoRound("player_21424")
	if runtime.hasKuangBao("player_21424") {
		t.Fatalf("expected 狂爆 to expire on startCommand round advance, got %+v", runtime.StatusEffects)
	}
	target.HP = 500
	expiredAttack := runtime.resolveAttack(actor, target, CommandNormalAttack)
	if expiredAttack.Damage != 50 {
		t.Fatalf("expected expired 狂爆 to stop modifying attack damage, got %+v", expiredAttack)
	}
}

func TestXueQieAppliesWoundBuffInfoOnHit(t *testing.T) {
	var runtime *Runtime
	for index := 0; index < 200; index += 1 {
		candidate := &Runtime{
			BattleID:         fmt.Sprintf("battle-xueqie-%d", index),
			Round:            1,
			Phase:            PhaseCommand,
			ActiveHandle:     "player_21424",
			nextSequence:     1,
			ConsumedSequence: map[int]bool{},
			DefendingHandles: map[string]bool{},
			StatusEffects:    map[string]BattleStatusEffects{},
			RoleSkills: []session.RoleSkill{
				{
					Name:        "血切",
					Level:       1,
					Type:        "oneE",
					Description: "f_s_血切^5BC46D&9@单体·状态&8@战士 &10@单刀&22@战斗&2@19&4@对敌人造成30%的物理伤害&0;击中敌人时有80%的机率使对方进入外伤状态4回合<br>(每回合损失气力为角色物理攻击的25%~30%)",
				},
			},
			Cells: []CellInfoPush{
				{
					BattleID: "battle-xueqie",
					Handle:   "player_21424",
					Camp:     CampTeam,
					HP:       500,
					MaxHP:    500,
					MP:       80,
					MaxMP:    80,
					Attack:   100,
					Defense:  0,
					Hit:      100,
				},
				{
					BattleID: "battle-xueqie",
					Handle:   "enemy_1",
					Camp:     CampEnemy,
					HP:       300,
					MaxHP:    300,
					Attack:   0,
					Defense:  0,
					Hit:      100,
					Dog:      0,
					Fat:      0,
				},
			},
		}
		actor := candidate.cellByHandle("player_21424")
		target := candidate.cellByHandle("enemy_1")
		if candidate.hashBattleRollWithSalt(actor, target, CommandXueQie, "status:外伤") < 80 {
			runtime = candidate
			break
		}
	}
	if runtime == nil {
		t.Fatal("expected to find deterministic 血切 status roll below 80")
	}

	result := runtime.ProcessAction(ActionRequest{
		BattleID:     runtime.BattleID,
		ActorHandle:  "player_21424",
		CommandID:    CommandXueQie,
		TargetHandle: "enemy_1",
		Round:        1,
		Sequence:     1,
	})
	if result.ErrorCode != "" {
		t.Fatalf("expected 血切 to be accepted, got %+v", result)
	}
	if len(result.BuffInfos) != 1 {
		t.Fatalf("expected 血切 hit to push one 外伤 BuffInfo, got %+v", result.BuffInfos)
	}
	buff := result.BuffInfos[0]
	if buff.Name != "外伤" || buff.Display != "25.png" || buff.TargetHandle != "enemy_1" || buff.ReleaseHandle != "player_21424" || buff.Round != 4 {
		t.Fatalf("expected source 外伤 buff metadata, got %+v", buff)
	}
	effect := runtime.StatusEffects["enemy_1"].Effects["外伤"]
	if effect.Name != "外伤" || effect.Rounds != 3 || effect.SourceHandle != "player_21424" || effect.SourceSkill != "血切" || effect.SourceAttack != 100 || effect.TickMinPercent != 25 || effect.TickMaxPercent != 30 {
		t.Fatalf("expected runtime 外伤 status to be recorded, got %+v", runtime.StatusEffects)
	}
	if len(result.Actions) < 2 || result.Actions[0].ActionName != "血切" || result.Actions[0].SourceActionLabel != "w8/cutBlood" {
		t.Fatalf("expected 血切 action to keep captured source label, got %+v", result.Actions)
	}
	woundAction := result.Actions[1]
	if woundAction.ActionName != "外伤" || woundAction.ActorHandle != "enemy_1" || woundAction.TargetHandle != "enemy_1" || woundAction.SourceMode != "0" || woundAction.SourceActionLabel != "battleStand" || woundAction.TargetActionStateCode != "3" {
		t.Fatalf("expected captured 外伤 battleStand self action before enemy attack, got %+v", result.Actions)
	}
	if woundAction.Damage < 25 || woundAction.Damage > 30 || woundAction.TargetHP != runtime.cellByHandle("enemy_1").HP {
		t.Fatalf("expected 外伤 tick to use 25%%-30%% source attack damage, got action=%+v enemy=%+v", woundAction, runtime.cellByHandle("enemy_1"))
	}
}

func TestWoundTickCanKillEnemyBeforeEnemyAttack(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-wound-kill",
		Round:            1,
		Phase:            PhaseCommand,
		ActiveHandle:     "player_21424",
		nextSequence:     1,
		ConsumedSequence: map[int]bool{},
		DefendingHandles: map[string]bool{},
		StatusEffects: map[string]BattleStatusEffects{
			"enemy_1": {
				Effects: map[string]BattleStatusEffect{
					"外伤": {
						Name:           "外伤",
						Rounds:         1,
						SourceHandle:   "player_21424",
						SourceSkill:    "血切",
						SourceAttack:   100,
						TickMinPercent: 25,
						TickMaxPercent: 30,
					},
				},
			},
		},
		Cells: []CellInfoPush{
			{
				BattleID: "battle-wound-kill",
				Handle:   "player_21424",
				Camp:     CampTeam,
				HP:       500,
				MaxHP:    500,
				Attack:   10,
				Defense:  0,
			},
			{
				BattleID: "battle-wound-kill",
				Handle:   "enemy_1",
				Camp:     CampEnemy,
				HP:       20,
				MaxHP:    300,
				Attack:   999,
				Defense:  0,
				Hit:      100,
			},
		},
	}

	result := runtime.ProcessAction(ActionRequest{
		BattleID:     "battle-wound-kill",
		ActorHandle:  "player_21424",
		CommandID:    CommandDefense,
		TargetHandle: "player_21424",
		Round:        1,
		Sequence:     1,
	})

	if result.ErrorCode != "" {
		t.Fatalf("expected defense to be accepted, got %+v", result)
	}
	if len(result.Actions) != 2 || result.Actions[0].ActionName != "防御" || result.Actions[1].ActionName != "外伤" {
		t.Fatalf("expected 外伤 tick to replace killed enemy attack, got %+v", result.Actions)
	}
	if result.Actions[1].Damage < 25 || !result.Actions[1].TargetDead {
		t.Fatalf("expected 外伤 tick to kill enemy, got %+v", result.Actions[1])
	}
	if runtime.cellByHandle("enemy_1").HP != 0 || runtime.StatusEffects["enemy_1"].Effects != nil {
		t.Fatalf("expected killed enemy wound status to expire, got enemy=%+v effects=%+v", runtime.cellByHandle("enemy_1"), runtime.StatusEffects)
	}
	if runtime.PendingOver == nil || runtime.PendingOver.Winner != CampTeam {
		t.Fatalf("expected wound kill to queue battle over for team, got result=%+v pendingOver=%+v", result, runtime.PendingOver)
	}
}

func TestXueQieDoesNotApplyWoundOnDodge(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-xueqie-dodge",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
		StatusEffects:    map[string]BattleStatusEffects{},
	}
	actor := &CellInfoPush{
		Handle: "player_21424",
		Camp:   CampTeam,
		Attack: 100,
		MP:     80,
		MaxMP:  80,
		Hit:    0,
	}
	target := &CellInfoPush{
		Handle: "enemy_1",
		Camp:   CampEnemy,
		HP:     300,
		MaxHP:  300,
		Dog:    1,
	}

	action := runtime.resolveAttack(actor, target, CommandXueQie)
	if action.TargetActionStateCode != "1" || action.Damage != 0 {
		t.Fatalf("expected 血切 dodge action, got %+v", action)
	}
	if len(runtime.PendingBuffInfos) != 0 {
		t.Fatalf("expected dodge to suppress 外伤 BuffInfo, got %+v", runtime.PendingBuffInfos)
	}
	if len(runtime.StatusEffects) != 0 {
		t.Fatalf("expected dodge to leave status effects empty, got %+v", runtime.StatusEffects)
	}
}

func TestEquipmentInnerInjuryAppliesCapturedBuffInfoOnHit(t *testing.T) {
	var runtime *Runtime
	var actor *CellInfoPush
	var target *CellInfoPush
	for index := 0; index < 500; index += 1 {
		candidate := newInnerInjuryTestRuntime(fmt.Sprintf("player_inner_%d", index), "enemy_inner")
		candidateActor := candidate.cellByHandle(fmt.Sprintf("player_inner_%d", index))
		candidateTarget := candidate.cellByHandle("enemy_inner")
		if candidate.hashBattleRollWithSalt(candidateActor, candidateTarget, CommandNormalAttack, "equipment:内伤") < 1 {
			runtime = candidate
			actor = candidateActor
			target = candidateTarget
			break
		}
	}
	if runtime == nil {
		t.Fatal("expected to find deterministic 绯雨匕首 inner-injury roll below 1")
	}

	action := runtime.resolveAttack(actor, target, CommandNormalAttack)

	if action.TargetActionStateCode != "0" || action.Damage <= 0 {
		t.Fatalf("expected inner-injury source hit to land, got %+v", action)
	}
	if len(runtime.PendingBuffInfos) != 1 {
		t.Fatalf("expected one 内伤 BuffInfo, got %+v", runtime.PendingBuffInfos)
	}
	buff := runtime.PendingBuffInfos[0]
	if buff.Name != "内伤" || buff.Display != "26.png" || buff.Round != 3 || buff.ReleaseHandle != actor.Handle || buff.TargetHandle != target.Handle {
		t.Fatalf("expected captured 内伤 BuffInfo metadata, got %+v", buff)
	}
	if !strings.Contains(buff.Description, "降低对象") || !strings.Contains(buff.Description, "点物理攻击") || !strings.Contains(buff.Description, "点魔法攻击力") {
		t.Fatalf("expected captured 内伤 description shape, got %q", buff.Description)
	}
	effect := runtime.StatusEffects[target.Handle].Effects["内伤"]
	if effect.Name != "内伤" || effect.Display != "26.png" || effect.Rounds != 3 || effect.SourceHandle != actor.Handle || effect.SourceSkill != "绯雨匕首" {
		t.Fatalf("expected runtime 内伤 status to be recorded, got %+v", runtime.StatusEffects)
	}
	if effect.AttackReduction < 18 || effect.AttackReduction > 30 {
		t.Fatalf("expected 内伤 to reduce 10%%-15%% of post-hit attack 180, got %+v", effect)
	}
	if target.Attack != 180-effect.AttackReduction {
		t.Fatalf("expected target attack to be reduced by inner injury, target=%+v effect=%+v", target, effect)
	}
}

func TestEquipmentInnerInjuryDoesNotApplyOnDodge(t *testing.T) {
	runtime := newInnerInjuryTestRuntime("player_inner_dodge", "enemy_inner_dodge")
	actor := runtime.cellByHandle("player_inner_dodge")
	target := runtime.cellByHandle("enemy_inner_dodge")
	actor.Hit = 0
	target.Dog = 1

	action := runtime.resolveAttack(actor, target, CommandNormalAttack)

	if action.TargetActionStateCode != "1" || action.Damage != 0 {
		t.Fatalf("expected dodge action, got %+v", action)
	}
	if len(runtime.PendingBuffInfos) != 0 || len(runtime.StatusEffects) != 0 {
		t.Fatalf("expected dodge to suppress 内伤 status, buffs=%+v effects=%+v", runtime.PendingBuffInfos, runtime.StatusEffects)
	}
}

func TestEquipmentInnerInjuryRestoresAttackWhenExpired(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-inner-injury-expire",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
		StatusEffects: map[string]BattleStatusEffects{
			"enemy_inner": {
				Effects: map[string]BattleStatusEffect{
					"内伤": {
						Name:            "内伤",
						Rounds:          1,
						AttackReduction: 24,
					},
				},
			},
		},
	}
	target := &CellInfoPush{
		BattleID: "battle-inner-injury-expire",
		Handle:   "enemy_inner",
		Camp:     CampEnemy,
		HP:       500,
		MaxHP:    500,
		Attack:   156,
	}

	actions, skipTurn := runtime.resolveStatusStartActions(target)

	if len(actions) != 0 || skipTurn {
		t.Fatalf("expected 内伤 to tick without action or skip, actions=%+v skip=%v", actions, skipTurn)
	}
	if target.Attack != 180 {
		t.Fatalf("expected 内伤 expiration to restore attack, got %+v", target)
	}
	if _, ok := runtime.StatusEffects["enemy_inner"]; ok {
		t.Fatalf("expected expired 内伤 status to clear, got %+v", runtime.StatusEffects)
	}
	if len(runtime.PendingClearBuffInfos) != 1 || runtime.PendingClearBuffInfos[0].Name != "内伤" || runtime.PendingClearBuffInfos[0].TargetHandle != "enemy_inner" {
		t.Fatalf("expected 内伤 clear buff info, got %+v", runtime.PendingClearBuffInfos)
	}
}

func TestEquipmentSealAppliesCapturedBuffInfoOnHit(t *testing.T) {
	var runtime *Runtime
	var actor *CellInfoPush
	var target *CellInfoPush
	for index := 0; index < 1000; index += 1 {
		candidate := newSealTestRuntime(fmt.Sprintf("player_seal_%d", index), "enemy_seal")
		candidateActor := candidate.cellByHandle(fmt.Sprintf("player_seal_%d", index))
		candidateTarget := candidate.cellByHandle("enemy_seal")
		if candidate.hashBattleRollWithSalt(candidateActor, candidateTarget, CommandNormalAttack, "equipment:封印") < 1 {
			runtime = candidate
			actor = candidateActor
			target = candidateTarget
			break
		}
	}
	if runtime == nil {
		t.Fatal("expected to find deterministic 伏魔棍 seal roll below 1")
	}

	action := runtime.resolveAttack(actor, target, CommandNormalAttack)

	if action.TargetActionStateCode != "0" || action.Damage <= 0 {
		t.Fatalf("expected seal source hit to land, got %+v", action)
	}
	if len(runtime.PendingBuffInfos) != 1 {
		t.Fatalf("expected one 封印 BuffInfo, got %+v", runtime.PendingBuffInfos)
	}
	buff := runtime.PendingBuffInfos[0]
	if buff.Name != "封印" || buff.Display != "19.png" || buff.Description != "作用时间内对象无法使用技能" || buff.Round != 3 || buff.ReleaseHandle != actor.Handle || buff.TargetHandle != target.Handle {
		t.Fatalf("expected captured 封印 BuffInfo metadata, got %+v", buff)
	}
	effect := runtime.StatusEffects[target.Handle].Effects["封印"]
	if effect.Name != "封印" || effect.Display != "19.png" || effect.Rounds != 3 || effect.SourceHandle != actor.Handle || effect.SourceSkill != "伏魔棍" || effect.SkipTurn {
		t.Fatalf("expected runtime 封印 status without skip-turn, got %+v", runtime.StatusEffects)
	}
}

func TestEquipmentSealDoesNotApplyOnDodge(t *testing.T) {
	runtime := newSealTestRuntime("player_seal_dodge", "enemy_seal_dodge")
	actor := runtime.cellByHandle("player_seal_dodge")
	target := runtime.cellByHandle("enemy_seal_dodge")
	actor.Hit = 0
	target.Dog = 1

	action := runtime.resolveAttack(actor, target, CommandNormalAttack)

	if action.TargetActionStateCode != "1" || action.Damage != 0 {
		t.Fatalf("expected dodge action, got %+v", action)
	}
	if len(runtime.PendingBuffInfos) != 0 || len(runtime.StatusEffects) != 0 {
		t.Fatalf("expected dodge to suppress 封印 status, buffs=%+v effects=%+v", runtime.PendingBuffInfos, runtime.StatusEffects)
	}
}

func TestEquipmentSealIgnoresResistanceOnlyDescription(t *testing.T) {
	runtime := newSealResistanceOnlyTestRuntime("player_seal_resist", "enemy_seal_resist")
	actor := runtime.cellByHandle("player_seal_resist")
	target := runtime.cellByHandle("enemy_seal_resist")

	action := runtime.resolveAttack(actor, target, CommandNormalAttack)

	if action.TargetActionStateCode != "0" || action.Damage <= 0 {
		t.Fatalf("expected normal hit to land, got %+v", action)
	}
	if len(runtime.PendingBuffInfos) != 0 || len(runtime.StatusEffects) != 0 {
		t.Fatalf("expected 封印抗性 equipment text not to create 封印 status, buffs=%+v effects=%+v", runtime.PendingBuffInfos, runtime.StatusEffects)
	}
}

func TestSealStatusActionForcesEnemyNormalAttackWithoutSkippingTurn(t *testing.T) {
	var runtime *Runtime
	for index := 0; index < 1000; index += 1 {
		candidate := &Runtime{
			BattleID:         fmt.Sprintf("battle-seal-flow-%d", index),
			Round:            1,
			Phase:            PhaseCommand,
			ActiveHandle:     "player_21424",
			nextSequence:     1,
			ConsumedSequence: map[int]bool{},
			DefendingHandles: map[string]bool{},
			StatusEffects: map[string]BattleStatusEffects{
				"enemy_shaman": {
					Effects: map[string]BattleStatusEffect{
						"封印": {
							Name:         "封印",
							Display:      "19.png",
							Description:  "作用时间内对象无法使用技能",
							Rounds:       1,
							SourceHandle: "player_21424",
							SourceSkill:  "伏魔棍",
						},
					},
				},
			},
			PendingSkillSeal: map[string]bool{},
			Cells: []CellInfoPush{
				{
					BattleID: "battle-seal-flow",
					Handle:   "player_21424",
					Camp:     CampTeam,
					HP:       500,
					MaxHP:    500,
					Attack:   10,
					Defense:  0,
					Dog:      0,
				},
				{
					BattleID:   "battle-seal-flow",
					Handle:     "enemy_shaman",
					Name:       "咒巫师",
					DisplayURL: "monstermap/incantationshaman.swf",
					Camp:       CampEnemy,
					HP:         300,
					MaxHP:      300,
					MP:         100,
					MaxMP:      100,
					Attack:     50,
					Defense:    0,
					Hit:        100,
				},
			},
		}
		enemy := candidate.cellByHandle("enemy_shaman")
		target := candidate.cellByHandle("player_21424")
		if candidate.resolveEnemySkillUse(enemy, target, CommandEnemyRockRain, enemyRockRainChance) {
			runtime = candidate
			break
		}
	}
	if runtime == nil {
		t.Fatal("expected to find deterministic 咒巫师落石 roll")
	}

	result := runtime.ProcessAction(ActionRequest{
		BattleID:     runtime.BattleID,
		ActorHandle:  "player_21424",
		CommandID:    CommandDefense,
		TargetHandle: "player_21424",
		Round:        1,
		Sequence:     1,
	})

	if result.ErrorCode != "" {
		t.Fatalf("expected defense to be accepted, got %+v", result)
	}
	if len(result.Actions) < 3 || result.Actions[0].ActionName != "防御" || result.Actions[1].ActionName != "封印" || result.Actions[2].ActionName != "普通攻击" {
		t.Fatalf("expected 防御 -> 封印 -> 普通攻击 sequence, got %+v", result.Actions)
	}
	if result.Actions[1].SourceActionLabel != "battleStand" || result.Actions[1].ActorHandle != "enemy_shaman" || result.Actions[1].TargetHandle != "enemy_shaman" || result.Actions[1].Damage != 0 {
		t.Fatalf("expected 封印 battleStand self action, got %+v", result.Actions[1])
	}
	for _, action := range result.Actions {
		if action.ActionName == "落石" {
			t.Fatalf("expected sealed enemy to avoid skill 落石, got actions=%+v", result.Actions)
		}
	}
	if len(result.ClearBuffInfos) != 1 || result.ClearBuffInfos[0].Name != "封印" || result.ClearBuffInfos[0].TargetHandle != "enemy_shaman" {
		t.Fatalf("expected final seal round to clear buff info, got %+v", result.ClearBuffInfos)
	}
	if runtime.PendingSkillSeal["enemy_shaman"] {
		t.Fatalf("expected enemy seal marker to be consumed after forced normal attack, got %+v", runtime.PendingSkillSeal)
	}
}

func TestSealRejectsPlayerSkillButAllowsNormalAttack(t *testing.T) {
	runtime := newSealPlayerCommandRuntime()
	skillResult := runtime.ProcessAction(ActionRequest{
		BattleID:     runtime.BattleID,
		ActorHandle:  "player_21424",
		CommandID:    CommandDuoDuanZhan,
		TargetHandle: "enemy_seal_command",
		Round:        1,
		Sequence:     1,
	})

	if skillResult.ErrorCode != "sealed_skill" {
		t.Fatalf("expected sealed player active skill to be rejected, got %+v", skillResult)
	}
	if !runtime.PendingSkillSeal["player_21424"] || runtime.ConsumedSequence[1] {
		t.Fatalf("expected rejected sealed skill not to consume command, pendingSeal=%+v consumed=%+v", runtime.PendingSkillSeal, runtime.ConsumedSequence)
	}

	normalResult := runtime.ProcessAction(ActionRequest{
		BattleID:     runtime.BattleID,
		ActorHandle:  "player_21424",
		CommandID:    CommandNormalAttack,
		TargetHandle: "enemy_seal_command",
		Round:        1,
		Sequence:     1,
	})

	if normalResult.ErrorCode != "" {
		t.Fatalf("expected sealed player normal attack to be accepted, got %+v", normalResult)
	}
	if runtime.PendingSkillSeal["player_21424"] {
		t.Fatalf("expected normal attack to consume player seal marker, got %+v", runtime.PendingSkillSeal)
	}
}

func TestJiFengCiAppliesSlownessBuffInfoOnHit(t *testing.T) {
	var runtime *Runtime
	var actor *CellInfoPush
	var target *CellInfoPush
	for index := 0; index < 200; index += 1 {
		candidate := newSlownessTestRuntime(fmt.Sprintf("player_slow_%d", index), "enemy_slow")
		candidateActor := candidate.cellByHandle(fmt.Sprintf("player_slow_%d", index))
		candidateTarget := candidate.cellByHandle("enemy_slow")
		if candidate.hashBattleRollWithSalt(candidateActor, candidateTarget, CommandJiFengCi, "status:迟钝") < jiFengCiSlownessChance {
			runtime = candidate
			actor = candidateActor
			target = candidateTarget
			break
		}
	}
	if runtime == nil {
		t.Fatal("expected to find deterministic 疾风刺迟钝 roll below 92")
	}

	action := runtime.resolveAttack(actor, target, CommandJiFengCi)

	if action.ActionName != "疾风刺" || action.SourceActionLabel != "w3/windCut" || action.TargetActionStateCode != "0" {
		t.Fatalf("expected captured 疾风刺 hit action, got %+v", action)
	}
	if len(runtime.PendingBuffInfos) != 1 {
		t.Fatalf("expected one 迟钝 BuffInfo, got %+v", runtime.PendingBuffInfos)
	}
	buff := runtime.PendingBuffInfos[0]
	if buff.Name != "迟钝" || buff.Display != "16.png" || buff.Round != 3 || buff.ReleaseHandle != actor.Handle || buff.TargetHandle != target.Handle {
		t.Fatalf("expected captured 迟钝 BuffInfo metadata, got %+v", buff)
	}
	if buff.Description != "降低对象100点命中和30点回避" {
		t.Fatalf("expected captured 迟钝 description shape, got %q", buff.Description)
	}
	effect := runtime.StatusEffects[target.Handle].Effects["迟钝"]
	if effect.Name != "迟钝" || effect.HitReduction != 100 || effect.DodgeReduction != 30 || effect.Rounds != 3 {
		t.Fatalf("expected runtime 迟钝 status to be recorded, got %+v", runtime.StatusEffects)
	}
	if target.Hit != 100 || target.Dog != 30 {
		t.Fatalf("expected target hit/dodge to be reduced by 50%%, got %+v", target)
	}
}

func TestTouDuAppliesPoisonBuffInfoOnHit(t *testing.T) {
	var runtime *Runtime
	var actor *CellInfoPush
	var target *CellInfoPush
	for index := 0; index < 200; index += 1 {
		candidate := newPoisonTestRuntime(fmt.Sprintf("player_poison_%d", index), "enemy_poison")
		candidateActor := candidate.cellByHandle(fmt.Sprintf("player_poison_%d", index))
		candidateTarget := candidate.cellByHandle("enemy_poison")
		if candidate.hashBattleRollWithSalt(candidateActor, candidateTarget, CommandTouDu, "status:中毒") < touDuPoisonChance {
			runtime = candidate
			actor = candidateActor
			target = candidateTarget
			break
		}
	}
	if runtime == nil {
		t.Fatal("expected to find deterministic 投毒中毒 roll below 80")
	}

	action := runtime.resolveAttack(actor, target, CommandTouDu)

	if action.ActionName != "投毒" || action.SourceActionLabel != "w3/drugAtk" || action.TargetActionStateCode != "0" {
		t.Fatalf("expected captured 投毒 hit action, got %+v", action)
	}
	if len(runtime.PendingBuffInfos) != 1 {
		t.Fatalf("expected one 中毒 BuffInfo, got %+v", runtime.PendingBuffInfos)
	}
	buff := runtime.PendingBuffInfos[0]
	if buff.Name != "中毒" || buff.Display != "8.png" || buff.Round != 4 || buff.ReleaseHandle != actor.Handle || buff.TargetHandle != target.Handle {
		t.Fatalf("expected captured 中毒 BuffInfo metadata, got %+v", buff)
	}
	if buff.Description != "降低对象5点魔防和9点物防，每回合内减少对象48~60点气力" {
		t.Fatalf("expected captured 中毒 description shape, got %q", buff.Description)
	}
	effect := runtime.StatusEffects[target.Handle].Effects["中毒"]
	if effect.Name != "中毒" || effect.DefenseReduction != 9 || effect.MagicDefenseReduction != 5 || effect.SourceAttack != 240 || effect.TickMinPercent != 20 || effect.TickMaxPercent != 25 || effect.Rounds != 4 {
		t.Fatalf("expected runtime 中毒 status to be recorded, got %+v", runtime.StatusEffects)
	}
	if target.Defense != 51 || target.MgcDefense != 31 {
		t.Fatalf("expected target defense/magic defense to be reduced by 15%%, got %+v", target)
	}
}

func TestTouDuProcessActionAppliesPoisonAndTicksInEnemyTurn(t *testing.T) {
	var runtime *Runtime
	var actor *CellInfoPush
	var target *CellInfoPush
	for index := 0; index < 200; index += 1 {
		actorHandle := fmt.Sprintf("player_poison_process_%d", index)
		candidate := newPoisonTestRuntime(actorHandle, "enemy_poison_process")
		candidate.BattleID = fmt.Sprintf("battle-poison-process-%d", index)
		candidate.Phase = PhaseCommand
		candidate.ActiveHandle = actorHandle
		candidate.ConsumedSequence = map[int]bool{}
		candidate.StoredPower = map[string]int{}
		for cellIndex := range candidate.Cells {
			candidate.Cells[cellIndex].BattleID = candidate.BattleID
		}
		candidateActor := candidate.cellByHandle(actorHandle)
		candidateTarget := candidate.cellByHandle("enemy_poison_process")
		if candidate.hashBattleRollWithSalt(candidateActor, candidateTarget, CommandTouDu, "status:中毒") < touDuPoisonChance {
			runtime = candidate
			actor = candidateActor
			target = candidateTarget
			break
		}
	}
	if runtime == nil {
		t.Fatal("expected to find deterministic ProcessAction 投毒中毒 roll below 80")
	}

	initialActor := *actor
	initialTarget := *target
	result := runtime.ProcessAction(ActionRequest{
		BattleID:     runtime.BattleID,
		ActorHandle:  actor.Handle,
		CommandID:    CommandTouDu,
		TargetHandle: target.Handle,
		Round:        1,
		Sequence:     1,
	})

	if result.ErrorCode != "" {
		t.Fatalf("expected ProcessAction 投毒 to be accepted, got %+v", result)
	}
	if len(result.Actions) < 2 {
		t.Fatalf("expected ProcessAction 投毒 to emit skill and poison tick actions, got %+v", result.Actions)
	}
	t.Logf("real ProcessAction poison actions=%+v buffs=%+v clears=%+v", result.Actions, result.BuffInfos, result.ClearBuffInfos)
	touDuAction := result.Actions[0]
	if touDuAction.ActionName != "投毒" || touDuAction.CommandID != CommandTouDu || touDuAction.SourceActionLabel != "w3/drugAtk" || touDuAction.ActorHandle != actor.Handle || touDuAction.TargetHandle != target.Handle {
		t.Fatalf("expected real ProcessAction first action to be captured 投毒, got %+v", touDuAction)
	}
	if touDuAction.Damage != 1 || touDuAction.TargetHP != 999 || touDuAction.TargetActionStateCode != "0" {
		t.Fatalf("expected 投毒 ProcessAction to apply captured minimum hit damage before poison status, got %+v", touDuAction)
	}
	if len(result.BuffInfos) != 1 {
		t.Fatalf("expected real ProcessAction to push one 中毒 BuffInfo, got %+v", result.BuffInfos)
	}
	buff := result.BuffInfos[0]
	if buff.Name != "中毒" || buff.Display != "8.png" || buff.Round != 4 || buff.ReleaseHandle != actor.Handle || buff.TargetHandle != target.Handle {
		t.Fatalf("expected real ProcessAction captured 中毒 BuffInfo metadata, got %+v", buff)
	}
	var poisonTick *ActionPush
	for index := range result.Actions {
		if result.Actions[index].ActionName == "中毒" && result.Actions[index].ActorHandle == target.Handle {
			poisonTick = &result.Actions[index]
			break
		}
	}
	if poisonTick == nil {
		t.Fatalf("expected real ProcessAction enemy turn to emit 中毒/battleStand tick, got %+v", result.Actions)
	}
	if poisonTick.SourceMode != "0" || poisonTick.SourceActionLabel != "battleStand" || poisonTick.TargetHandle != target.Handle || poisonTick.TargetActionStateCode != "3" {
		t.Fatalf("expected real ProcessAction 中毒 tick self state action, got %+v", poisonTick)
	}
	if poisonTick.Damage < 48 || poisonTick.Damage > 60 {
		t.Fatalf("expected real ProcessAction 中毒 tick damage from 20%%-25%% source attack, got %+v", poisonTick)
	}
	if actor.MP != 84 {
		t.Fatalf("expected real ProcessAction 投毒 to consume MP 16, actor=%+v", actor)
	}
	expectedTargetHP := touDuAction.TargetHP - poisonTick.Damage
	if target.Defense != 51 || target.MgcDefense != 31 || target.HP != expectedTargetHP || poisonTick.TargetHP != expectedTargetHP {
		t.Fatalf("expected real ProcessAction to apply poison reductions and tick HP, target=%+v tick=%+v", target, poisonTick)
	}
	if runtime.StatusEffects[target.Handle].Effects["中毒"].Rounds != 3 {
		t.Fatalf("expected real ProcessAction enemy-turn tick to reduce remaining poison rounds to 3, got %+v", runtime.StatusEffects)
	}
	writePoisonRealFlowFixture(t, initialActor, initialTarget, result)
}

func writePoisonRealFlowFixture(t *testing.T, actor CellInfoPush, target CellInfoPush, result ActionResult) {
	t.Helper()
	outputPath := strings.TrimSpace(os.Getenv("BATTLE_POISON_REAL_FLOW_OUT"))
	if outputPath == "" {
		return
	}
	payload := struct {
		BattleID       string              `json:"battleId"`
		Actor          CellInfoPush        `json:"actor"`
		Target         CellInfoPush        `json:"target"`
		Actions        []ActionPush        `json:"actions"`
		BuffInfos      []BuffInfoPush      `json:"buffInfos"`
		ClearBuffInfos []ClearBuffInfoPush `json:"clearBuffInfos"`
	}{
		BattleID:       result.Actions[0].BattleID,
		Actor:          actor,
		Target:         target,
		Actions:        result.Actions,
		BuffInfos:      result.BuffInfos,
		ClearBuffInfos: result.ClearBuffInfos,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal poison real flow fixture: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		t.Fatalf("create poison real flow fixture dir: %v", err)
	}
	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		t.Fatalf("write poison real flow fixture: %v", err)
	}
}

func writeJieDuRealFlowFixture(t *testing.T, actor CellInfoPush, enemy CellInfoPush, poisonBuff BuffInfoPush, result ActionResult) {
	t.Helper()
	outputPath := strings.TrimSpace(os.Getenv("BATTLE_JIEDU_REAL_FLOW_OUT"))
	if outputPath == "" {
		return
	}
	payload := struct {
		BattleID       string              `json:"battleId"`
		Actor          CellInfoPush        `json:"actor"`
		Enemy          CellInfoPush        `json:"enemy"`
		BuffInfos      []BuffInfoPush      `json:"buffInfos"`
		Actions        []ActionPush        `json:"actions"`
		ClearBuffInfos []ClearBuffInfoPush `json:"clearBuffInfos"`
	}{
		BattleID:       actor.BattleID,
		Actor:          actor,
		Enemy:          enemy,
		BuffInfos:      []BuffInfoPush{poisonBuff},
		Actions:        result.Actions,
		ClearBuffInfos: result.ClearBuffInfos,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal jiedu real flow fixture: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		t.Fatalf("create jiedu real flow fixture dir: %v", err)
	}
	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		t.Fatalf("write jiedu real flow fixture: %v", err)
	}
}

func TestTouDuDoesNotApplyPoisonOnDodge(t *testing.T) {
	runtime := newPoisonTestRuntime("player_poison_dodge", "enemy_poison_dodge")
	actor := runtime.cellByHandle("player_poison_dodge")
	target := runtime.cellByHandle("enemy_poison_dodge")
	actor.Hit = 0
	target.Dog = 1

	action := runtime.resolveAttack(actor, target, CommandTouDu)

	if action.TargetActionStateCode != "1" || action.Damage != 0 {
		t.Fatalf("expected 投毒 dodge action, got %+v", action)
	}
	if len(runtime.PendingBuffInfos) != 0 || len(runtime.StatusEffects) != 0 {
		t.Fatalf("expected dodge to suppress 中毒 status, buffs=%+v effects=%+v", runtime.PendingBuffInfos, runtime.StatusEffects)
	}
}

func TestPoisonTickRestoresDefenseWhenExpired(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-poison-expire",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
		StatusEffects: map[string]BattleStatusEffects{
			"enemy_poison": {
				Effects: map[string]BattleStatusEffect{
					"中毒": {
						Name:                  "中毒",
						Rounds:                1,
						SourceHandle:          "player_poison",
						SourceSkill:           "投毒",
						SourceAttack:          240,
						TickMinPercent:        20,
						TickMaxPercent:        25,
						DefenseReduction:      9,
						MagicDefenseReduction: 5,
					},
				},
			},
		},
	}
	target := &CellInfoPush{
		BattleID:   "battle-poison-expire",
		Handle:     "enemy_poison",
		Camp:       CampEnemy,
		HP:         500,
		MaxHP:      500,
		Defense:    51,
		MgcDefense: 31,
	}

	actions, skipTurn := runtime.resolveStatusStartActions(target)

	if len(actions) != 1 || skipTurn {
		t.Fatalf("expected one 中毒 tick without skip, actions=%+v skip=%v", actions, skipTurn)
	}
	action := actions[0]
	if action.ActionName != "中毒" || action.ActorHandle != "enemy_poison" || action.TargetHandle != "enemy_poison" || action.SourceMode != "0" || action.SourceActionLabel != "battleStand" || action.TargetActionStateCode != "3" {
		t.Fatalf("expected captured 中毒 battleStand self action, got %+v", action)
	}
	if action.Damage < 48 || action.Damage > 60 || action.TargetHP != target.HP {
		t.Fatalf("expected 中毒 tick to use 20%%-25%% source attack damage, action=%+v target=%+v", action, target)
	}
	if target.Defense != 60 || target.MgcDefense != 36 {
		t.Fatalf("expected 中毒 expiration to restore defense/magic defense, got %+v", target)
	}
	if _, ok := runtime.StatusEffects["enemy_poison"]; ok {
		t.Fatalf("expected expired 中毒 status to clear, got %+v", runtime.StatusEffects)
	}
	if len(runtime.PendingClearBuffInfos) != 1 || runtime.PendingClearBuffInfos[0].Name != "中毒" || runtime.PendingClearBuffInfos[0].TargetHandle != "enemy_poison" {
		t.Fatalf("expected 中毒 clear buff info, got %+v", runtime.PendingClearBuffInfos)
	}
}

func TestPoisonTickCanKillAndMarksTargetDead(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-poison-death",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
		StatusEffects: map[string]BattleStatusEffects{
			"enemy_poison": {
				Effects: map[string]BattleStatusEffect{
					"中毒": {
						Name:           "中毒",
						Rounds:         2,
						SourceHandle:   "player_poison",
						SourceSkill:    "投毒",
						SourceAttack:   240,
						TickMinPercent: 20,
						TickMaxPercent: 25,
					},
				},
			},
		},
	}
	target := &CellInfoPush{
		BattleID: "battle-poison-death",
		Handle:   "enemy_poison",
		Camp:     CampEnemy,
		HP:       30,
		MaxHP:    500,
	}

	actions, skipTurn := runtime.resolveStatusStartActions(target)

	if skipTurn || len(actions) != 1 {
		t.Fatalf("expected one killing 中毒 tick without skip, actions=%+v skip=%v", actions, skipTurn)
	}
	action := actions[0]
	if action.ActionName != "中毒" || action.TargetHP != 0 || !action.TargetDead || target.HP != 0 {
		t.Fatalf("expected 中毒 tick to kill and mark target dead, action=%+v target=%+v", action, target)
	}
	if len(action.RefreshInfos) != 1 || action.RefreshInfos[0].Handle != target.Handle || action.RefreshInfos[0].HP != 0 {
		t.Fatalf("expected killing 中毒 tick to include hp=0 refresh info, got %+v", action.RefreshInfos)
	}
}

func TestPoisonTickDeathDoesNotQueueStartCommandForDeadActor(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-poison-next-actor-death",
		Round:            1,
		Phase:            PhasePlaying,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
		StatusEffects: map[string]BattleStatusEffects{
			"player_next": {
				Effects: map[string]BattleStatusEffect{
					"中毒": {
						Name:           "中毒",
						Rounds:         2,
						SourceHandle:   "player_leader",
						SourceSkill:    "投毒",
						SourceAttack:   240,
						TickMinPercent: 20,
						TickMaxPercent: 25,
					},
				},
			},
		},
		Cells: []CellInfoPush{
			{
				Handle: "player_leader",
				Camp:   CampTeam,
				HP:     500,
				MaxHP:  500,
				Attack: 100,
				Hit:    100,
			},
			{
				Handle: "player_next",
				Camp:   CampTeam,
				HP:     30,
				MaxHP:  500,
				Attack: 100,
				Hit:    100,
			},
			{
				Handle:  "enemy_dummy",
				Camp:    CampEnemy,
				HP:      1000,
				MaxHP:   1000,
				Attack:  1,
				Defense: 0,
				Hit:     100,
				Dog:     0,
			},
		},
	}

	result := runtime.resolveEnemyTurnAndNextCommand(runtime.cellByHandle("player_leader"), nil)

	if result.ErrorCode != "" {
		t.Fatalf("expected enemy turn flow to resolve, got %+v", result)
	}
	var poisonDeath *ActionPush
	for index := range result.Actions {
		action := result.Actions[index]
		if action.ActionName == "中毒" && action.ActorHandle == "player_next" {
			poisonDeath = &result.Actions[index]
			break
		}
	}
	if poisonDeath == nil || !poisonDeath.TargetDead || poisonDeath.TargetHP != 0 {
		t.Fatalf("expected player_next poison tick death action, got action=%+v actions=%+v", poisonDeath, result.Actions)
	}
	start := pendingStartFor(runtime, "player_leader")
	if start == nil || pendingStartFor(runtime, "player_next") != nil {
		t.Fatalf("expected next startCommand to skip dead poison actor, pendingStarts=%+v", runtime.PendingStarts)
	}
}

func TestPoisonTickDeathEndsBattleWhenLastEnemyDies(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-poison-last-enemy-death",
		Round:            1,
		Phase:            PhasePlaying,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
		StatusEffects: map[string]BattleStatusEffects{
			"enemy_poison": {
				Effects: map[string]BattleStatusEffect{
					"中毒": {
						Name:           "中毒",
						Rounds:         2,
						SourceHandle:   "player_poison",
						SourceSkill:    "投毒",
						SourceAttack:   240,
						TickMinPercent: 20,
						TickMaxPercent: 25,
					},
				},
			},
		},
		Cells: []CellInfoPush{
			{
				Handle: "player_poison",
				Camp:   CampTeam,
				HP:     500,
				MaxHP:  500,
				Attack: 100,
				Hit:    100,
			},
			{
				Handle: "enemy_poison",
				Camp:   CampEnemy,
				HP:     30,
				MaxHP:  500,
				Attack: 120,
				Hit:    100,
			},
		},
	}

	result := runtime.resolveEnemyTurnAndNextCommand(runtime.cellByHandle("player_poison"), nil)

	if result.ErrorCode != "" {
		t.Fatalf("expected enemy turn flow to resolve, got %+v", result)
	}
	if len(result.Actions) != 1 || result.Actions[0].ActionName != "中毒" || !result.Actions[0].TargetDead || result.Actions[0].TargetHP != 0 {
		t.Fatalf("expected only poison death action for last enemy, got %+v", result.Actions)
	}
	if runtime.Phase != PhaseFinished || runtime.PendingOver == nil || runtime.PendingOver.Winner != CampTeam {
		t.Fatalf("expected poison death of last enemy to finish battle with team win, phase=%s pendingOver=%+v", runtime.Phase, runtime.PendingOver)
	}
	if pendingStartActor(runtime) != nil {
		t.Fatalf("expected no next startCommand after last enemy poison death, got starts=%+v", runtime.PendingStarts)
	}
}

func TestJiFengCiDoesNotApplySlownessOnDodge(t *testing.T) {
	runtime := newSlownessTestRuntime("player_slow_dodge", "enemy_slow_dodge")
	actor := runtime.cellByHandle("player_slow_dodge")
	target := runtime.cellByHandle("enemy_slow_dodge")
	actor.Hit = 0
	target.Dog = 1

	action := runtime.resolveAttack(actor, target, CommandJiFengCi)

	if action.TargetActionStateCode != "1" || action.Damage != 0 {
		t.Fatalf("expected 疾风刺 dodge action, got %+v", action)
	}
	if len(runtime.PendingBuffInfos) != 0 || len(runtime.StatusEffects) != 0 {
		t.Fatalf("expected dodge to suppress 迟钝 status, buffs=%+v effects=%+v", runtime.PendingBuffInfos, runtime.StatusEffects)
	}
}

func TestJiFengCiSlownessRestoresHitAndDodgeWhenExpired(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-slowness-expire",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
		StatusEffects: map[string]BattleStatusEffects{
			"enemy_slow": {
				Effects: map[string]BattleStatusEffect{
					"迟钝": {
						Name:           "迟钝",
						Rounds:         1,
						HitReduction:   100,
						DodgeReduction: 30,
					},
				},
			},
		},
	}
	target := &CellInfoPush{
		BattleID: "battle-slowness-expire",
		Handle:   "enemy_slow",
		Camp:     CampEnemy,
		HP:       500,
		MaxHP:    500,
		Hit:      100,
		Dog:      30,
	}

	actions, skipTurn := runtime.resolveStatusStartActions(target)

	if len(actions) != 0 || skipTurn {
		t.Fatalf("expected 迟钝 to tick without action or skip, actions=%+v skip=%v", actions, skipTurn)
	}
	if target.Hit != 200 || target.Dog != 60 {
		t.Fatalf("expected 迟钝 expiration to restore hit/dodge, got %+v", target)
	}
	if _, ok := runtime.StatusEffects["enemy_slow"]; ok {
		t.Fatalf("expected expired 迟钝 status to clear, got %+v", runtime.StatusEffects)
	}
	if len(runtime.PendingClearBuffInfos) != 1 || runtime.PendingClearBuffInfos[0].Name != "迟钝" || runtime.PendingClearBuffInfos[0].TargetHandle != "enemy_slow" {
		t.Fatalf("expected 迟钝 clear buff info, got %+v", runtime.PendingClearBuffInfos)
	}
}

func newSlownessTestRuntime(actorHandle string, targetHandle string) *Runtime {
	return &Runtime{
		BattleID:         "battle-slowness",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
		StatusEffects:    map[string]BattleStatusEffects{},
		Cells: []CellInfoPush{
			{
				BattleID: "battle-slowness",
				Handle:   actorHandle,
				Camp:     CampTeam,
				HP:       500,
				MaxHP:    500,
				MP:       100,
				MaxMP:    100,
				Attack:   100,
				Hit:      100,
			},
			{
				BattleID: "battle-slowness",
				Handle:   targetHandle,
				Camp:     CampEnemy,
				HP:       1000,
				MaxHP:    1000,
				Attack:   120,
				Defense:  0,
				Hit:      200,
				Dog:      60,
			},
		},
	}
}

func newPoisonTestRuntime(actorHandle string, targetHandle string) *Runtime {
	return &Runtime{
		BattleID:         "battle-poison",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
		StatusEffects:    map[string]BattleStatusEffects{},
		RoleSkills: []session.RoleSkill{
			{
				Name:        "投毒",
				Level:       1,
				Type:        "oneE",
				Description: "f_s_投毒^5BC46D&9@单体·状态&8@游侠 &10@匕首&22@战斗&2@16&4@<font color='#00cc00'>特殊发动条件:需要【毒药x1】<br>叠加施放将削弱其造成中毒的功效</font><br>有80%的机率使敌人中毒，4回合内降低对方15%的物理防御和魔法防御&0;每回合使敌人损失气力为物理攻击的20%~25%",
			},
		},
		Cells: []CellInfoPush{
			{
				BattleID: "battle-poison",
				Handle:   actorHandle,
				Camp:     CampTeam,
				HP:       500,
				MaxHP:    500,
				MP:       100,
				MaxMP:    100,
				Attack:   240,
				Hit:      100,
			},
			{
				BattleID:   "battle-poison",
				Handle:     targetHandle,
				Camp:       CampEnemy,
				HP:         1000,
				MaxHP:      1000,
				Attack:     120,
				Defense:    60,
				MgcDefense: 36,
				Hit:        200,
				Dog:        0,
			},
		},
	}
}

func newInnerInjuryTestRuntime(actorHandle string, targetHandle string) *Runtime {
	return &Runtime{
		BattleID:         "battle-inner-injury",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
		StatusEffects:    map[string]BattleStatusEffects{},
		RoleItemsByHandle: map[string][]session.RoleItem{
			actorHandle: {
				{
					Type:        "装备",
					Name:        "绯雨匕首",
					Display:     "51.png",
					Description: "f_i_绯雨匕首<br>特殊效果: 击中敌人时候有1%机率使敌人进入内伤状态3回合(降低敌人10%~15%的物理攻击和魔法攻击)",
				},
			},
		},
		Cells: []CellInfoPush{
			{
				BattleID: "battle-inner-injury",
				Handle:   actorHandle,
				Camp:     CampTeam,
				HP:       500,
				MaxHP:    500,
				Attack:   100,
				Hit:      100,
			},
			{
				BattleID:     "battle-inner-injury",
				Handle:       targetHandle,
				Camp:         CampEnemy,
				HP:           1000,
				MaxHP:        1000,
				Attack:       180,
				Defense:      0,
				MgcDefense:   10,
				Dog:          0,
				CommandLabel: "普通攻击",
			},
		},
	}
}

func newSealTestRuntime(actorHandle string, targetHandle string) *Runtime {
	return &Runtime{
		BattleID:         "battle-seal",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
		StatusEffects:    map[string]BattleStatusEffects{},
		RoleItemsByHandle: map[string][]session.RoleItem{
			actorHandle: {
				{
					Type:        "装备",
					Name:        "伏魔棍",
					Display:     "56.png",
					Description: "f_i_伏魔棍<br>特殊效果: 击中敌人时有1%的机率对敌人造成封印3回合",
				},
			},
		},
		Cells: []CellInfoPush{
			{
				BattleID: "battle-seal",
				Handle:   actorHandle,
				Camp:     CampTeam,
				HP:       500,
				MaxHP:    500,
				Attack:   100,
				Hit:      100,
			},
			{
				BattleID:     "battle-seal",
				Handle:       targetHandle,
				Camp:         CampEnemy,
				HP:           1000,
				MaxHP:        1000,
				Attack:       180,
				Defense:      0,
				MgcDefense:   10,
				Dog:          0,
				CommandLabel: "普通攻击",
			},
		},
	}
}

func newSealResistanceOnlyTestRuntime(actorHandle string, targetHandle string) *Runtime {
	runtime := newSealTestRuntime(actorHandle, targetHandle)
	runtime.RoleItemsByHandle[actorHandle] = []session.RoleItem{
		{
			Type:        "装备",
			Name:        "骷髅戒指",
			Display:     "759.png",
			Description: "f_i_骷髅戒指<br>特殊效果: 封印抗性:+10%<br>[精炼+1] 每升一级 封印抗性+1%",
		},
	}
	return runtime
}

func newSealPlayerCommandRuntime() *Runtime {
	return &Runtime{
		BattleID:         "battle-seal-command",
		Round:            1,
		Phase:            PhaseCommand,
		ActiveHandle:     "player_21424",
		nextSequence:     1,
		ConsumedSequence: map[int]bool{},
		DefendingHandles: map[string]bool{},
		StatusEffects:    map[string]BattleStatusEffects{},
		PendingSkillSeal: map[string]bool{"player_21424": true},
		RoleSkillsByHandle: map[string][]session.RoleSkill{
			"player_21424": {
				{Name: "多段斩", Level: 1, Type: "oneE", Description: fallbackDuoDuanDescription(1)},
			},
		},
		Cells: []CellInfoPush{
			{
				BattleID: "battle-seal-command",
				Handle:   "player_21424",
				Camp:     CampTeam,
				HP:       500,
				MaxHP:    500,
				MP:       100,
				MaxMP:    100,
				Attack:   100,
				Hit:      100,
			},
			{
				BattleID: "battle-seal-command",
				Handle:   "enemy_seal_command",
				Camp:     CampEnemy,
				HP:       1000,
				MaxHP:    1000,
				Attack:   1,
				Defense:  0,
				Hit:      100,
				Dog:      0,
			},
		},
	}
}

func TestEnemyPalsyAtkAppliesParalysisOnHit(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-palsy-hit",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
		StatusEffects:    map[string]BattleStatusEffects{},
	}
	enemy := &CellInfoPush{
		Handle: "enemy_bee",
		Name:   "毒蜂",
		Camp:   CampEnemy,
		Attack: 80,
		Hit:    100,
	}
	target := &CellInfoPush{
		Handle: "player_21424",
		Camp:   CampTeam,
		HP:     500,
		MaxHP:  500,
		Dog:    0,
	}

	action := runtime.resolveAttack(enemy, target, CommandEnemyPalsyAtk)

	if action.ActionName != "蜂刺" || action.SourceActionLabel != "palsyAtk" || action.TargetActionStateCode != "0" {
		t.Fatalf("expected captured 毒蜂 palsy action, got %+v", action)
	}
	if len(runtime.PendingBuffInfos) != 1 || runtime.PendingBuffInfos[0].Name != "麻痹" || runtime.PendingBuffInfos[0].Display != "17.png" || runtime.PendingBuffInfos[0].TargetHandle != "player_21424" || runtime.PendingBuffInfos[0].Round != 2 {
		t.Fatalf("expected 蜂刺 hit to push 麻痹 BuffInfo, got %+v", runtime.PendingBuffInfos)
	}
	effect := runtime.StatusEffects["player_21424"].Effects["麻痹"]
	if effect.Name != "麻痹" || effect.Rounds != 2 || effect.SourceHandle != "enemy_bee" || effect.SourceSkill != "蜂刺" || !effect.SkipTurn {
		t.Fatalf("expected 麻痹 skip-turn status, got %+v", runtime.StatusEffects)
	}
}

func TestEnemyPalsyAtkDoesNotApplyParalysisOnDodge(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-palsy-dodge",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
		StatusEffects:    map[string]BattleStatusEffects{},
	}
	enemy := &CellInfoPush{
		Handle: "enemy_bee",
		Name:   "毒蜂",
		Camp:   CampEnemy,
		Attack: 80,
		Hit:    0,
	}
	target := &CellInfoPush{
		Handle: "player_21424",
		Camp:   CampTeam,
		HP:     500,
		MaxHP:  500,
		Dog:    1,
	}

	action := runtime.resolveAttack(enemy, target, CommandEnemyPalsyAtk)

	if action.TargetActionStateCode != "1" || action.Damage != 0 {
		t.Fatalf("expected 蜂刺 dodge action, got %+v", action)
	}
	if len(runtime.PendingBuffInfos) != 0 || len(runtime.StatusEffects) != 0 {
		t.Fatalf("expected dodge to suppress 麻痹 status, buffs=%+v effects=%+v", runtime.PendingBuffInfos, runtime.StatusEffects)
	}
}

func TestRobotawlRoundAtkAppliesArmorBreakOnHit(t *testing.T) {
	var runtime *Runtime
	var enemy *CellInfoPush
	var target *CellInfoPush
	for index := 0; index < 300; index += 1 {
		candidate := &Runtime{
			BattleID:         fmt.Sprintf("battle-robotawl-xiejia-%d", index),
			Round:            1,
			nextSequence:     1,
			DefendingHandles: map[string]bool{},
			StatusEffects:    map[string]BattleStatusEffects{},
		}
		candidateEnemy := &CellInfoPush{
			Handle:     "enemy_robotawl",
			Name:       "机木锥兵",
			DisplayURL: "monstermap/robotawl.swf",
			Camp:       CampEnemy,
			HP:         1330,
			MaxHP:      1330,
			MP:         20,
			MaxMP:      20,
			Attack:     624,
			Hit:        100,
		}
		candidateTarget := &CellInfoPush{
			Handle:  "player_21432",
			Camp:    CampTeam,
			HP:      1215,
			MaxHP:   1215,
			Defense: 240,
			Dog:     0,
		}
		if candidate.hashBattleRollWithSalt(candidateEnemy, candidateTarget, CommandEnemyRoundAtk, "status:卸甲") < enemyRobotawlArmorBreakChance {
			runtime = candidate
			enemy = candidateEnemy
			target = candidateTarget
			break
		}
	}
	if runtime == nil {
		t.Fatal("expected to find deterministic 轮转刺伤卸甲 roll below 70")
	}

	action := runtime.resolveAttack(enemy, target, CommandEnemyRoundAtk)

	if action.ActionName != "轮转刺伤" || action.SourceActionLabel != "roundatk" || action.TargetActionStateCode == "1" {
		t.Fatalf("expected captured 机木锥兵 轮转刺伤 hit action, got %+v", action)
	}
	if len(runtime.PendingBuffInfos) != 1 {
		t.Fatalf("expected one 卸甲 BuffInfo, got %+v", runtime.PendingBuffInfos)
	}
	buff := runtime.PendingBuffInfos[0]
	if buff.Name != "卸甲" || buff.Display != "10.png" || buff.Round != 3 || buff.ReleaseHandle != enemy.Handle || buff.TargetHandle != target.Handle {
		t.Fatalf("expected captured 卸甲 BuffInfo metadata, got %+v", buff)
	}
	if buff.Description != "降低对象62点物理防御力" {
		t.Fatalf("expected captured 卸甲 description shape, got %q", buff.Description)
	}
	effect := runtime.StatusEffects[target.Handle].Effects["卸甲"]
	if effect.Name != "卸甲" || effect.DefenseReduction != 62 || effect.SourceAttack != 624 || effect.Rounds != 3 || effect.SourceSkill != "轮转刺伤" || effect.AppliedAction != "roundatk" {
		t.Fatalf("expected runtime 卸甲 status to be recorded, got %+v", runtime.StatusEffects)
	}
	if target.Defense != 178 {
		t.Fatalf("expected target physical defense to be reduced by captured source attack 10%%, got %+v", target)
	}
	if enemy.MP != 10 {
		t.Fatalf("expected captured 轮转刺伤 to consume 10 MP, got %+v", enemy)
	}
}

func TestRobotawlRoundAtkDoesNotApplyArmorBreakOnDodge(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-robotawl-xiejia-dodge",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
		StatusEffects:    map[string]BattleStatusEffects{},
	}
	enemy := &CellInfoPush{
		Handle:     "enemy_robotawl",
		Name:       "机木锥兵",
		DisplayURL: "monstermap/robotawl.swf",
		Camp:       CampEnemy,
		MP:         20,
		MaxMP:      20,
		Attack:     624,
		Hit:        0,
	}
	target := &CellInfoPush{
		Handle:  "player_21432",
		Camp:    CampTeam,
		HP:      1215,
		MaxHP:   1215,
		Defense: 240,
		Dog:     1,
	}

	action := runtime.resolveAttack(enemy, target, CommandEnemyRoundAtk)

	if action.TargetActionStateCode != "1" || action.Damage != 0 {
		t.Fatalf("expected 轮转刺伤 dodge action, got %+v", action)
	}
	if len(runtime.PendingBuffInfos) != 0 || len(runtime.StatusEffects) != 0 {
		t.Fatalf("expected dodge to suppress 卸甲 status, buffs=%+v effects=%+v", runtime.PendingBuffInfos, runtime.StatusEffects)
	}
	if target.Defense != 240 {
		t.Fatalf("expected dodge to keep target defense unchanged, got %+v", target)
	}
}

func TestArmorBreakRestoresDefenseWhenExpired(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-xiejia-expire",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
		StatusEffects: map[string]BattleStatusEffects{
			"player_21432": {
				Effects: map[string]BattleStatusEffect{
					"卸甲": {
						Name:             "卸甲",
						Rounds:           1,
						DefenseReduction: 62,
					},
				},
			},
		},
	}
	target := &CellInfoPush{
		BattleID: "battle-xiejia-expire",
		Handle:   "player_21432",
		Camp:     CampTeam,
		HP:       1000,
		MaxHP:    1000,
		Defense:  178,
	}

	actions, skipTurn := runtime.resolveStatusStartActions(target)

	if len(actions) != 0 || skipTurn {
		t.Fatalf("expected 卸甲 to tick without action or skip, actions=%+v skip=%v", actions, skipTurn)
	}
	if target.Defense != 240 {
		t.Fatalf("expected 卸甲 expiration to restore defense, got %+v", target)
	}
	if _, ok := runtime.StatusEffects["player_21432"]; ok {
		t.Fatalf("expected expired 卸甲 status to clear, got %+v", runtime.StatusEffects)
	}
	if len(runtime.PendingClearBuffInfos) != 1 || runtime.PendingClearBuffInfos[0].Name != "卸甲" || runtime.PendingClearBuffInfos[0].TargetHandle != "player_21432" {
		t.Fatalf("expected 卸甲 clear buff info, got %+v", runtime.PendingClearBuffInfos)
	}
}

func TestEnemyPalsyAtkSkipsPlayerNextCommand(t *testing.T) {
	var runtime *Runtime
	for index := 0; index < 300; index += 1 {
		candidate := &Runtime{
			BattleID:         fmt.Sprintf("battle-palsy-flow-%d", index),
			Round:            1,
			Phase:            PhaseCommand,
			ActiveHandle:     "player_21424",
			nextSequence:     1,
			ConsumedSequence: map[int]bool{},
			DefendingHandles: map[string]bool{},
			StatusEffects:    map[string]BattleStatusEffects{},
			Cells: []CellInfoPush{
				{
					BattleID: "battle-palsy-flow",
					Handle:   "player_21424",
					Camp:     CampTeam,
					HP:       500,
					MaxHP:    500,
					Attack:   10,
					Defense:  0,
					Dog:      0,
				},
				{
					BattleID:   "battle-palsy-flow",
					Handle:     "enemy_bee",
					Name:       "毒蜂",
					DisplayURL: "monstermap/drughornets.swf",
					Camp:       CampEnemy,
					HP:         300,
					MaxHP:      300,
					Attack:     50,
					Defense:    0,
					Hit:        100,
				},
			},
		}
		actor := candidate.cellByHandle("enemy_bee")
		target := candidate.cellByHandle("player_21424")
		if candidate.resolveEnemySkillUse(actor, target, CommandEnemyPalsyAtk, enemyPalsyAtkChance) {
			runtime = candidate
			break
		}
	}
	if runtime == nil {
		t.Fatal("expected to find deterministic 毒蜂 palsy skill roll")
	}

	result := runtime.ProcessAction(ActionRequest{
		BattleID:     runtime.BattleID,
		ActorHandle:  "player_21424",
		CommandID:    CommandDefense,
		TargetHandle: "player_21424",
		Round:        1,
		Sequence:     1,
	})

	if result.ErrorCode != "" {
		t.Fatalf("expected defense turn to be accepted, got %+v", result)
	}
	if len(result.Actions) != 3 || result.Actions[0].ActionName != "防御" || result.Actions[1].ActionName != "蜂刺" || result.Actions[2].ActionName != "麻痹" {
		t.Fatalf("expected 防御 -> 蜂刺 -> 麻痹 skip sequence, got %+v", result.Actions)
	}
	if result.Actions[2].ActorHandle != "player_21424" || result.Actions[2].TargetHandle != "player_21424" || result.Actions[2].SourceMode != "0" || result.Actions[2].SourceActionLabel != "battleStand" || result.Actions[2].Damage != 0 {
		t.Fatalf("expected 麻痹 battleStand self action, got %+v", result.Actions[2])
	}
	if len(result.BuffInfos) != 1 || result.BuffInfos[0].Name != "麻痹" || result.BuffInfos[0].Display != "17.png" || result.BuffInfos[0].Round != 2 {
		t.Fatalf("expected 麻痹 BuffInfo to be pushed with source round, got %+v", result.BuffInfos)
	}
	effect := runtime.StatusEffects["player_21424"].Effects["麻痹"]
	if effect.Rounds != 1 || !effect.SkipTurn {
		t.Fatalf("expected first skipped command to consume one 麻痹 round, got %+v", runtime.StatusEffects)
	}
	start := pendingStartFor(runtime, "player_21424")
	if start == nil || start.Round != 2 || start.Sequence != 2 {
		t.Fatalf("expected skipped turn to queue the next player startCommand, got %+v starts=%+v", start, runtime.PendingStarts)
	}
}

func TestCapturedStunOnHitAppliesBuffInfoDuringEnemyAttack(t *testing.T) {
	var runtime *Runtime
	for index := 0; index < 300; index += 1 {
		candidate := &Runtime{
			BattleID:         fmt.Sprintf("battle-stun-on-hit-%d", index),
			Round:            1,
			nextSequence:     1,
			DefendingHandles: map[string]bool{},
			StatusEffects:    map[string]BattleStatusEffects{},
		}
		actor := &CellInfoPush{
			Handle:     "enemy_bomepig",
			Name:       "爆骨猪",
			DisplayURL: "monstermap/bomepig.swf",
			Camp:       CampEnemy,
			Attack:     30,
			Hit:        100,
			HP:         300,
			MaxHP:      300,
		}
		target := &CellInfoPush{
			Handle: "player_21424",
			Camp:   CampTeam,
			HP:     500,
			MaxHP:  500,
			Dog:    0,
		}
		if candidate.hashBattleRollWithSalt(actor, target, CommandEnemyAttack, "status:眩晕") < enemyStunOnHitChance {
			runtime = candidate
			break
		}
	}
	if runtime == nil {
		t.Fatal("expected to find deterministic 眩晕 on-hit roll")
	}

	player := &CellInfoPush{
		Handle: "player_21424",
		Camp:   CampTeam,
		Attack: 20,
		Hit:    100,
		HP:     500,
		MaxHP:  500,
	}
	enemy := &CellInfoPush{
		Handle:     "enemy_bomepig",
		Name:       "爆骨猪",
		DisplayURL: "monstermap/bomepig.swf",
		Camp:       CampEnemy,
		Attack:     30,
		Hit:        100,
		HP:         300,
		MaxHP:      300,
		Dog:        0,
	}

	playerAction := runtime.resolveAttack(player, enemy, CommandNormalAttack)

	if playerAction.TargetActionStateCode != "0" || enemy.HP >= enemy.MaxHP {
		t.Fatalf("expected player attack to hit captured stun source, got action=%+v enemy=%+v", playerAction, enemy)
	}
	if len(runtime.PendingBuffInfos) != 0 || len(runtime.StatusEffects) != 0 {
		t.Fatalf("expected player attack not to apply 眩晕, buffs=%+v effects=%+v", runtime.PendingBuffInfos, runtime.StatusEffects)
	}

	action := runtime.resolveAttack(enemy, player, CommandEnemyAttack)

	if action.TargetActionStateCode != "0" || player.HP >= player.MaxHP {
		t.Fatalf("expected enemy attack to hit player, got action=%+v player=%+v", action, player)
	}
	if len(runtime.PendingBuffInfos) != 1 {
		t.Fatalf("expected captured 眩晕 BuffInfo, got %+v", runtime.PendingBuffInfos)
	}
	buff := runtime.PendingBuffInfos[0]
	if buff.Name != "眩晕" || buff.Display != "9.png" || buff.Description != "眩晕无法行动" || buff.Round != 2 || buff.ReleaseHandle != "enemy_bomepig" || buff.TargetHandle != "player_21424" {
		t.Fatalf("expected captured 眩晕 BuffInfo metadata, got %+v", buff)
	}
	effect := runtime.StatusEffects["player_21424"].Effects["眩晕"]
	if effect.Name != "眩晕" || effect.Rounds != 2 || effect.SourceHandle != "enemy_bomepig" || effect.AppliedAction != "yun" || !effect.SkipTurn {
		t.Fatalf("expected 眩晕 skip-turn status, got %+v", runtime.StatusEffects)
	}
}

func TestCapturedStunOnHitSkipsPlayerWithYunAction(t *testing.T) {
	var runtime *Runtime
	for index := 0; index < 300; index += 1 {
		candidate := &Runtime{
			BattleID:         fmt.Sprintf("battle-stun-flow-%d", index),
			Round:            1,
			Phase:            PhaseCommand,
			ActiveHandle:     "player_21424",
			nextSequence:     1,
			ConsumedSequence: map[int]bool{},
			DefendingHandles: map[string]bool{},
			StatusEffects:    map[string]BattleStatusEffects{},
			Cells: []CellInfoPush{
				{
					BattleID: "battle-stun-flow",
					Handle:   "player_21424",
					Camp:     CampTeam,
					HP:       500,
					MaxHP:    500,
					Attack:   20,
					Defense:  0,
					Hit:      100,
					Dog:      0,
				},
				{
					BattleID:   "battle-stun-flow",
					Handle:     "enemy_crystalrock",
					Name:       "晶石怪",
					DisplayURL: "monstermap/crystalrock.swf",
					Camp:       CampEnemy,
					HP:         300,
					MaxHP:      300,
					Attack:     10,
					Defense:    0,
					Hit:        100,
					Dog:        0,
				},
			},
		}
		actor := candidate.cellByHandle("enemy_crystalrock")
		target := candidate.cellByHandle("player_21424")
		if candidate.hashBattleRollWithSalt(actor, target, CommandEnemyAttack, "status:眩晕") < enemyStunOnHitChance {
			runtime = candidate
			break
		}
	}
	if runtime == nil {
		t.Fatal("expected to find deterministic 眩晕 flow roll")
	}

	result := runtime.ProcessAction(ActionRequest{
		BattleID:     runtime.BattleID,
		ActorHandle:  "player_21424",
		CommandID:    CommandNormalAttack,
		TargetHandle: "enemy_crystalrock",
		Round:        1,
		Sequence:     1,
	})

	if result.ErrorCode != "" {
		t.Fatalf("expected player attack to be accepted, got %+v", result)
	}
	if len(result.Actions) != 5 ||
		result.Actions[0].ActionName != "普通攻击" ||
		result.Actions[1].ActorHandle != "enemy_crystalrock" ||
		result.Actions[2].ActionName != "眩晕" ||
		result.Actions[3].ActorHandle != "enemy_crystalrock" ||
		result.Actions[4].ActionName != "眩晕" {
		t.Fatalf("expected player attack -> enemy action -> 眩晕 -> enemy action -> 眩晕 sequence, got %+v", result.Actions)
	}
	stunAction := result.Actions[2]
	if stunAction.ActorHandle != "player_21424" || stunAction.TargetHandle != "player_21424" || stunAction.SourceMode != "0" || stunAction.SourceActionLabel != "yun" || stunAction.Damage != 0 {
		t.Fatalf("expected captured 眩晕 yun self action, got %+v", stunAction)
	}
	secondStunAction := result.Actions[4]
	if secondStunAction.ActorHandle != "player_21424" || secondStunAction.TargetHandle != "player_21424" || secondStunAction.SourceMode != "0" || secondStunAction.SourceActionLabel != "yun" || secondStunAction.Damage != 0 {
		t.Fatalf("expected second captured 眩晕 yun self action, got %+v", secondStunAction)
	}
	if len(result.BuffInfos) != 1 || result.BuffInfos[0].Name != "眩晕" || result.BuffInfos[0].Display != "9.png" || result.BuffInfos[0].Round != 2 {
		t.Fatalf("expected captured 眩晕 BuffInfo to be pushed, got %+v", result.BuffInfos)
	}
	if len(result.ClearBuffInfos) != 1 || result.ClearBuffInfos[0].TargetHandle != "player_21424" || result.ClearBuffInfos[0].Name != "眩晕" {
		t.Fatalf("expected consumed 眩晕 to push clearBuffInfo before next command, got %+v", result.ClearBuffInfos)
	}
	if runtime.hasActiveAutoContinueSkipStatus("player_21424") {
		t.Fatalf("expected chained 眩晕 skips to consume stun before next command, got %+v", runtime.StatusEffects)
	}
	start := pendingStartFor(runtime, "player_21424")
	// Round advances twice through auto-continue; sequence only advances when a free window opens.
	if start == nil || start.Round != 3 || start.Sequence != 2 {
		t.Fatalf("expected 眩晕 skip chain to queue player startCommand after monster continuation, got %+v starts=%+v", start, runtime.PendingStarts)
	}
}

func TestResolveShiXueZhanLifeStealUsesActualDamage(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-shixue",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
		RoleSkills: []session.RoleSkill{
			{
				Name:        "嗜血斩",
				Level:       1,
				Type:        "oneE",
				Description: "f_s_嗜血斩^5BC46D&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@24&4@对敌人造成92%的物理伤害&0;并有100%机率将对敌人造成伤害的70%转换为气力</font>",
			},
		},
	}
	actor := &CellInfoPush{
		Handle: "player_21424",
		Camp:   CampTeam,
		Attack: 135,
		MaxHP:  725,
		HP:     409,
		MaxMP:  194,
		MP:     170,
		Fat:    0,
	}
	target := &CellInfoPush{
		Handle:  "enemy_robber",
		Camp:    CampEnemy,
		MaxHP:   445,
		HP:      445,
		Defense: 0,
	}

	action := runtime.resolveAttack(actor, target, CommandShiXueZhan)

	if action.ActionName != "嗜血斩" || action.SourceActionLabel != "w8/xyz1" {
		t.Fatalf("expected captured 嗜血斩 Lv1 label, got %+v", action)
	}
	if action.Damage != 124 || action.TargetHP != 321 {
		t.Fatalf("expected 92%% physical damage without stored power bonus from 135 attack, got %+v", action)
	}
	if actor.MP != 146 {
		t.Fatalf("expected 嗜血斩 Lv1 to consume MP 24, got actor=%+v", actor)
	}
	if actor.HP != 495 {
		t.Fatalf("expected lifesteal floor(124*0.7)=86 to restore HP 409->495, got actor=%+v action=%+v", actor, action)
	}
	if action.RefreshInfos[0].Handle != actor.Handle || action.RefreshInfos[0].HP != 495 || action.RefreshInfos[0].MP != 146 {
		t.Fatalf("expected actor HP/MP refresh from 嗜血斩, got %+v", action.RefreshInfos)
	}
}

func TestCapturedSkillDodgeStillConsumesMP(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-skill-dodge",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
		RoleSkills: []session.RoleSkill{
			{
				Name:        "多段斩",
				Level:       2,
				Type:        "oneE",
				Description: "f_s_多段斩^ffffff&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@10&4@提升60%的物理伤害",
			},
		},
	}
	actor := &CellInfoPush{
		Handle: "player_21424",
		Camp:   CampTeam,
		Attack: 100,
		Hit:    0,
		MaxMP:  154,
		MP:     154,
		Fat:    100,
	}
	target := &CellInfoPush{
		Handle: "enemy_flower",
		Camp:   CampEnemy,
		MaxHP:  230,
		HP:     230,
		Dog:    1,
	}

	action := runtime.resolveAttack(actor, target, CommandDuoDuanZhan)

	if action.TargetActionStateCode != "1" || action.Damage != 0 || target.HP != 230 {
		t.Fatalf("expected skill dodge to prevent damage, got action=%+v target=%+v", action, target)
	}
	if actor.MP != 144 || len(action.RefreshInfos) != 2 || action.RefreshInfos[0].MP != 144 {
		t.Fatalf("expected captured skill dodge to still consume MP 10, got actor=%+v action=%+v", actor, action)
	}
}

func TestBattleEnemySlideCutUsesCapturedLabelAndMPCost(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-enemy-slidecut",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
	}
	actor := &CellInfoPush{
		Handle:     "enemy_robber",
		Camp:       CampEnemy,
		Name:       "盗贼",
		DisplayURL: "monstermap/robber.swf",
		Attack:     47,
		MaxMP:      194,
		MP:         194,
	}
	target := &CellInfoPush{
		Handle: "player_21424",
		Camp:   CampTeam,
		MaxHP:  725,
		HP:     725,
	}

	action := runtime.resolveAttack(actor, target, CommandEnemySlideCut)

	if action.ActionName != "滑行斩" || action.SourceActionLabel != "slideCut" {
		t.Fatalf("expected captured enemy slideCut action, got %+v", action)
	}
	if action.Damage != 47 || action.TargetHP != 678 {
		t.Fatalf("expected slideCut to use current physical damage path without stored power bonus, got %+v", action)
	}
	if actor.MP != 184 || len(action.RefreshInfos) != 2 || action.RefreshInfos[0].MP != 184 {
		t.Fatalf("expected slideCut to consume captured MP cost 10, got actor=%+v action=%+v", actor, action)
	}
}

func TestBattleEnemyShadeCutUsesCapturedLabelAndMPCost(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-enemy-shadecut",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
	}
	actor := &CellInfoPush{
		Handle:     "enemy_wolf",
		Camp:       CampEnemy,
		Name:       "单刀狼人",
		DisplayURL: "monstermap/bigswordwolf.swf",
		Attack:     260,
		MaxMP:      334,
		MP:         334,
	}
	target := &CellInfoPush{
		Handle: "player_21424",
		Camp:   CampTeam,
		MaxHP:  725,
		HP:     725,
	}

	action := runtime.resolveAttack(actor, target, CommandEnemyShadeCut)

	if action.ActionName != "影刃" || action.SourceActionLabel != "shadeCut" {
		t.Fatalf("expected captured enemy shadeCut action, got %+v", action)
	}
	if action.Damage != 260 || action.TargetHP != 465 {
		t.Fatalf("expected shadeCut to use current physical damage path without stored power bonus, got %+v", action)
	}
	if actor.MP != 294 || len(action.RefreshInfos) != 2 || action.RefreshInfos[0].MP != 294 {
		t.Fatalf("expected shadeCut to consume captured MP cost 40, got actor=%+v action=%+v", actor, action)
	}
}

func TestBattleEnemyHelixAtkUsesCapturedLabelAndMPCost(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-enemy-helix",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
	}
	actor := &CellInfoPush{
		Handle:     "5176206909809579",
		Camp:       CampEnemy,
		Name:       "蛤蟆精",
		DisplayURL: "monstermap/cracktoad.swf",
		Attack:     240,
		MaxMP:      564,
		MP:         564,
	}
	target := &CellInfoPush{
		Handle:  "player_21424",
		Camp:    CampTeam,
		MaxHP:   1045,
		HP:      1045,
		Defense: 158,
	}

	action := runtime.resolveAttack(actor, target, CommandEnemyHelixAtk)

	if action.ActionName != "螺旋锤杀" || action.SourceActionLabel != "helixAtk" {
		t.Fatalf("expected captured cracktoad helixAtk action, got %+v", action)
	}
	if action.Damage != 238 || action.TargetHP != 807 {
		t.Fatalf("expected helixAtk to use the half physical defense path, got %+v", action)
	}
	if actor.MP != 554 || len(action.RefreshInfos) != 2 || action.RefreshInfos[0].MP != 554 {
		t.Fatalf("expected helixAtk to consume captured MP cost 10, got actor=%+v action=%+v", actor, action)
	}
}

func TestFeixiandongBossRampagePowerUsesCapturedBuffWithoutMPCost(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-feixiandong-rampage",
		Round:            3,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
	}
	actor := &CellInfoPush{
		Handle:     "2654338502092795",
		Camp:       CampEnemy,
		Name:       "巨岩魔",
		DisplayURL: "monstermap/largerock.swf",
		HP:         1500,
		MaxHP:      1500,
		MP:         564,
		MaxMP:      564,
	}

	actions := runtime.resolveEnemyRampageActions(actor)

	if len(actions) != 1 || actions[0].ActionName != "暴走之力" || actions[0].SourceActionLabel != "battleStand" || actions[0].TargetHandle != actor.Handle {
		t.Fatalf("expected captured boss 暴走之力 self action, got %+v", actions)
	}
	if actor.MP != 564 || len(actions[0].RefreshInfos) != 1 || actions[0].RefreshInfos[0].MP != 564 {
		t.Fatalf("expected 暴走之力 to keep captured MP unchanged, actor=%+v action=%+v", actor, actions[0])
	}
	if len(runtime.PendingBuffInfos) != 1 || runtime.PendingBuffInfos[0].Name != "暴走之力" || runtime.PendingBuffInfos[0].Display != "1595.png" || runtime.PendingBuffInfos[0].Round != 48 || !strings.Contains(runtime.PendingBuffInfos[0].Description, "还有 48 回合暴走") {
		t.Fatalf("expected captured 暴走之力 BuffInfo, got %+v", runtime.PendingBuffInfos)
	}
}

func TestHuangfengBossesUseCapturedRampagePower(t *testing.T) {
	for _, tc := range []struct {
		name       string
		displayURL string
	}{
		{name: "黄风二寨主", displayURL: "monstermap/hfscastellan.swf"},
		{name: "黄风大寨主", displayURL: "monstermap/hfcastellan.swf"},
		{name: "黄风寨夫人", displayURL: "monstermap/hflady.swf"},
	} {
		runtime := &Runtime{
			BattleID:         "battle-huangfeng-rampage",
			Round:            1,
			nextSequence:     1,
			DefendingHandles: map[string]bool{},
		}
		actor := &CellInfoPush{
			Handle:     tc.name,
			Camp:       CampEnemy,
			Name:       tc.name,
			DisplayURL: tc.displayURL,
			HP:         1200,
			MaxHP:      1200,
			MP:         564,
			MaxMP:      564,
		}

		actions := runtime.resolveEnemyRampageActions(actor)

		if len(actions) != 1 || actions[0].ActionName != "暴走之力" || actions[0].SourceActionLabel != "battleStand" || actions[0].TargetHandle != actor.Handle {
			t.Fatalf("expected %s to use captured 暴走之力 battleStand, got %+v", tc.name, actions)
		}
		if actor.MP != 564 || len(actions[0].RefreshInfos) != 1 || actions[0].RefreshInfos[0].MP != 564 {
			t.Fatalf("expected %s 暴走之力 to keep MP unchanged, actor=%+v action=%+v", tc.name, actor, actions[0])
		}
	}
}

func TestHuangfengIncantationShamanRockRainUsesCapturedAllTargetSkill(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-huangfeng-rockrain",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
		Cells: []CellInfoPush{
			{
				Handle:     "player_21424",
				Camp:       CampTeam,
				HP:         1085,
				MaxHP:      1085,
				MgcDefense: 40,
			},
			{
				Handle:     "7496686002240421",
				Camp:       CampEnemy,
				Name:       "咒巫师",
				DisplayURL: "monstermap/incantationshaman.swf",
				HP:         500,
				MaxHP:      500,
				MP:         550,
				MaxMP:      550,
				Attack:     202,
			},
		},
	}
	actor := runtime.cellByHandle("7496686002240421")

	action := runtime.resolveEnemyCommandActions(actor, runtime.cellByHandle("player_21424"), CommandEnemyRockRain)[0]

	if action.ActionName != "落石" || action.SourceActionLabel != "rockRain" || action.TargetHandle != "all" || action.SourceMode != "1" {
		t.Fatalf("expected captured 落石 all-target action, got %+v", action)
	}
	if action.Damage != 182 || runtime.Cells[0].HP != 903 || runtime.Cells[1].MP != 540 {
		t.Fatalf("expected 落石 to use captured MP cost 10 and magic damage path, action=%+v cells=%+v", action, runtime.Cells)
	}
}

func TestHuangfengSecondCastellanDarkMoonCutUsesCapturedLabel(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-huangfeng-darkmoon",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
	}
	actor := &CellInfoPush{
		Handle:     "3218685759638239",
		Camp:       CampEnemy,
		Name:       "黄风二寨主",
		DisplayURL: "monstermap/hfscastellan.swf",
		HP:         1200,
		MaxHP:      1200,
		MP:         564,
		MaxMP:      564,
		Attack:     240,
	}
	target := &CellInfoPush{
		Handle:  "player_21424",
		Camp:    CampTeam,
		HP:      1085,
		MaxHP:   1085,
		Defense: 167,
	}

	action := runtime.resolveAttack(actor, target, CommandEnemyDarkMoon)

	if action.ActionName != "暗月斩" || action.SourceActionLabel != "darkMoonCut" || action.SourceMode != "1" {
		t.Fatalf("expected captured 暗月斩 action, got %+v", action)
	}
	if action.Damage != 157 || action.TargetHP != 928 || actor.MP != 554 {
		t.Fatalf("expected 暗月斩 to use captured MP cost 10 and half physical defense, action=%+v actor=%+v", action, actor)
	}
}

func TestHuangfengFirstCastellanEarthShockUsesCapturedAllTargetSkill(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-huangfeng-earthshock",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
		Cells: []CellInfoPush{
			{
				Handle:  "player_21424",
				Camp:    CampTeam,
				HP:      1085,
				MaxHP:   1085,
				Defense: 160,
			},
			{
				Handle:     "2600686416056495",
				Camp:       CampEnemy,
				Name:       "黄风大寨主",
				DisplayURL: "monstermap/hfcastellan.swf",
				HP:         1500,
				MaxHP:      1500,
				MP:         564,
				MaxMP:      564,
				Attack:     260,
			},
		},
	}
	actor := runtime.cellByHandle("2600686416056495")

	action := runtime.resolveEnemyCommandActions(actor, runtime.cellByHandle("player_21424"), CommandEnemyEarthShock)[0]

	if action.ActionName != "裂震击" || action.SourceActionLabel != "earthShockAtk" || action.TargetHandle != "all" || action.SourceMode != "1" {
		t.Fatalf("expected captured 裂震击 all-target action, got %+v", action)
	}
	if action.Damage != 180 || runtime.Cells[0].HP != 905 || runtime.Cells[1].MP != 554 {
		t.Fatalf("expected 裂震击 to use captured MP cost 10 and half physical defense, action=%+v cells=%+v", action, runtime.Cells)
	}
}

func TestHuangfengLadyDeludeAppliesCapturedConfusionBuff(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-huangfeng-delude",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
		StatusEffects:    map[string]BattleStatusEffects{},
	}
	actor := &CellInfoPush{
		Handle:     "5431285114036433",
		Camp:       CampEnemy,
		Name:       "黄风寨夫人",
		DisplayURL: "monstermap/hflady.swf",
		HP:         111,
		MaxHP:      1200,
		MP:         704,
		MaxMP:      704,
	}
	target := &CellInfoPush{
		Handle: "player_21424",
		Camp:   CampTeam,
		HP:     1555,
		MaxHP:  1555,
		MP:     424,
		MaxMP:  424,
	}

	actions := runtime.resolveEnemyCommandActions(actor, target, CommandEnemyDelude)

	if len(actions) != 1 {
		t.Fatalf("expected one captured 魅惑术 action, got %+v", actions)
	}
	action := actions[0]
	if action.ActionName != "魅惑术" || action.SourceActionLabel != "delude" || action.SourceMode != "1" || action.TargetHandle != target.Handle {
		t.Fatalf("expected captured 魅惑术/delude one-target action, got %+v", action)
	}
	if action.Damage != 0 || action.TargetHP != target.HP || len(action.TargetActionResults) != 0 {
		t.Fatalf("expected 魅惑术 to be status-only without target result rows, got %+v", action)
	}
	if action.TargetActionState != "none" || action.TargetActionStateCode != "3" {
		t.Fatalf("expected 魅惑术 to play as status-only target none, got %+v", action)
	}
	if actor.MP != 694 || len(action.RefreshInfos) != 2 || action.RefreshInfos[0].MP != 694 {
		t.Fatalf("expected 魅惑术 to consume captured MP cost 10, actor=%+v action=%+v", actor, action)
	}
	if len(runtime.PendingBuffInfos) != 1 {
		t.Fatalf("expected 魅惑术 to push one 混乱 BuffInfo, got %+v", runtime.PendingBuffInfos)
	}
	buff := runtime.PendingBuffInfos[0]
	if buff.Name != "混乱" || buff.Display != "20.png" || buff.Description != "这个状态让人失去理智&0;胡乱攻击甚至自己人。" || buff.Round != 2 || buff.ReleaseHandle != actor.Handle || buff.TargetHandle != target.Handle {
		t.Fatalf("expected captured 混乱 BuffInfo metadata, got %+v", buff)
	}
	effect := runtime.StatusEffects[target.Handle].Effects["混乱"]
	if effect.Name != "混乱" || effect.Rounds != 2 || effect.SourceHandle != actor.Handle || effect.SourceSkill != "魅惑术" || effect.AppliedAction != "delude" || effect.SkipTurn {
		t.Fatalf("expected captured 混乱 status without skip-turn, got %+v", runtime.StatusEffects)
	}
}

func TestHuangfengLadyDeludeUsesCapturedChanceAndStatusAction(t *testing.T) {
	if enemyDeludeChance != 33 || enemyDeludeMPCost != 10 || enemyDeludeStatusRounds != 2 {
		t.Fatalf("expected captured delude chance/cost/rounds to stay 33/10/2, got chance=%d cost=%d rounds=%d", enemyDeludeChance, enemyDeludeMPCost, enemyDeludeStatusRounds)
	}
	if !sourceEnemyCanDelude(&CellInfoPush{Name: "黄风寨夫人"}) || !sourceEnemyCanDelude(&CellInfoPush{DisplayURL: "monstermap/hflady.swf"}) {
		t.Fatal("expected 黄风寨夫人/hflady to unlock captured 魅惑术")
	}
	if sourceEnemyCanDelude(&CellInfoPush{Name: "黄风大寨主", DisplayURL: "monstermap/hfcastellan.swf"}) {
		t.Fatal("expected non-hflady enemies not to unlock 魅惑术")
	}
	runtime := &Runtime{
		BattleID:      "battle-huangfeng-confusion",
		Round:         2,
		nextSequence:  1,
		StatusEffects: map[string]BattleStatusEffects{},
	}
	target := &CellInfoPush{
		Handle: "player_21424",
		Camp:   CampTeam,
		HP:     1555,
		MaxHP:  1555,
		MP:     424,
		MaxMP:  424,
	}
	runtime.applyStatusEffect(target.Handle, BattleStatusEffect{
		Name:          "混乱",
		Display:       "20.png",
		Description:   "这个状态让人失去理智&0;胡乱攻击甚至自己人。",
		Rounds:        2,
		SourceHandle:  "5431285114036433",
		SourceSkill:   "魅惑术",
		AppliedAction: "delude",
	})

	statusActions, skipTurn := runtime.resolveStatusStartActions(target)

	if skipTurn || len(statusActions) != 1 {
		t.Fatalf("expected 混乱 to emit status action without skipping turn, skip=%v actions=%+v", skipTurn, statusActions)
	}
	action := statusActions[0]
	if action.ActionName != "混乱" || action.ActorHandle != target.Handle || action.TargetHandle != target.Handle || action.SourceMode != "0" || action.SourceActionLabel != "battleStand" {
		t.Fatalf("expected captured 混乱 battleStand self action, got %+v", action)
	}
	if runtime.StatusEffects[target.Handle].Effects["混乱"].Rounds != 1 {
		t.Fatalf("expected first 混乱 status action to consume one round, got %+v", runtime.StatusEffects)
	}
	if !runtime.PendingConfusion[target.Handle] {
		t.Fatalf("expected 混乱 status action to force the next command into random normal attack, got %+v", runtime.PendingConfusion)
	}
}

func TestConfusionForcesCapturedPlayerNormalAttackAgainstTeammate(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-confusion-teammate",
		Round:            4,
		Phase:            PhaseCommand,
		ActiveHandle:     "player_21424",
		nextSequence:     1,
		ConsumedSequence: map[int]bool{},
		DefendingHandles: map[string]bool{},
		PendingConfusion: map[string]bool{"player_21424": true},
		Cells: []CellInfoPush{
			{
				Handle:  "player_21424",
				Camp:    CampTeam,
				HP:      1050,
				MaxHP:   1525,
				MP:      414,
				MaxMP:   414,
				Attack:  200,
				Defense: 0,
				Hit:     100,
			},
			{
				Handle:  "player_21432",
				Camp:    CampTeam,
				HP:      715,
				MaxHP:   715,
				Defense: 0,
				Dog:     0,
			},
		},
	}

	result := runtime.ProcessAction(ActionRequest{
		BattleID:     runtime.BattleID,
		ActorHandle:  "player_21424",
		CommandID:    CommandDuoDuanZhan,
		TargetHandle: "5431285114036433",
		Round:        4,
		Sequence:     1,
	})

	if result.ErrorCode != "" {
		t.Fatalf("expected confused command to be accepted as forced normal attack, got %+v", result)
	}
	if len(result.Actions) == 0 {
		t.Fatalf("expected forced normal attack action, got %+v", result)
	}
	action := result.Actions[0]
	if action.ActionName != "普通攻击" || action.CommandID != CommandNormalAttack || action.ActorHandle != "player_21424" || action.TargetHandle != "player_21432" || action.SourceActionLabel != "nomalAtk" {
		t.Fatalf("expected captured confused teammate normal attack, got %+v", action)
	}
	if action.Damage <= 0 || runtime.cellByHandle("player_21432").HP >= 715 {
		t.Fatalf("expected confused normal attack to damage teammate, action=%+v target=%+v", action, runtime.cellByHandle("player_21432"))
	}
	if runtime.cellByHandle("player_21424").MP != 414 {
		t.Fatalf("expected forced normal attack not to spend requested skill MP, actor=%+v", runtime.cellByHandle("player_21424"))
	}
	if runtime.PendingConfusion["player_21424"] {
		t.Fatalf("expected pending confusion command to be consumed, got %+v", runtime.PendingConfusion)
	}
}

func TestConfusionAutoExecutesPlayerNormalAttackBeforeStartCommand(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-confusion-teammate",
		Round:            4,
		Phase:            PhasePlaying,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
		StatusEffects:    map[string]BattleStatusEffects{},
		PendingConfusion: map[string]bool{},
		Cells: []CellInfoPush{
			{
				Handle:  "player_21424",
				Camp:    CampTeam,
				HP:      1050,
				MaxHP:   1525,
				MP:      414,
				MaxMP:   414,
				Attack:  200,
				Defense: 0,
				Hit:     100,
			},
			{
				Handle:  "player_21432",
				Camp:    CampTeam,
				HP:      715,
				MaxHP:   715,
				Defense: 0,
				Dog:     0,
			},
			{
				Handle:  "enemy_dummy",
				Camp:    CampEnemy,
				Name:    "黄风寨喽啰",
				HP:      1200,
				MaxHP:   1200,
				Attack:  1,
				Defense: 0,
				Hit:     100,
			},
		},
	}
	runtime.applyStatusEffect("player_21424", BattleStatusEffect{
		Name:          "混乱",
		Display:       "20.png",
		Description:   "这个状态让人失去理智&0;胡乱攻击甚至自己人。",
		Rounds:        1,
		SourceHandle:  "5431285114036433",
		SourceSkill:   "魅惑术",
		AppliedAction: "delude",
	})

	result := runtime.resolveEnemyTurnAndNextCommand(&CellInfoPush{Handle: "not-in-team-order"}, nil)

	if result.ErrorCode != "" {
		t.Fatalf("expected confused turn to auto-resolve, got %+v", result)
	}
	statusIndex := -1
	for index := range result.Actions {
		action := result.Actions[index]
		if action.ActionName == "混乱" && action.ActorHandle == "player_21424" && action.SourceActionLabel == "battleStand" {
			statusIndex = index
			break
		}
	}
	if statusIndex < 0 {
		t.Fatalf("expected 混乱 status action before auto attack, got %+v", result.Actions)
	}
	if statusIndex+1 >= len(result.Actions) {
		t.Fatalf("expected auto normal attack after 混乱 status action, got %+v", result.Actions)
	}
	action := result.Actions[statusIndex+1]
	if action.ActionName != "普通攻击" || action.CommandID != CommandNormalAttack || action.ActorHandle != "player_21424" || action.TargetHandle != "player_21432" || action.SourceActionLabel != "nomalAtk" {
		t.Fatalf("expected captured 混乱 to auto normal-attack teammate before startCommand, got %+v actions=%+v", action, result.Actions)
	}
	if action.Damage <= 0 || runtime.cellByHandle("player_21432").HP >= 715 {
		t.Fatalf("expected auto confused attack to damage teammate, action=%+v target=%+v", action, runtime.cellByHandle("player_21432"))
	}
	if runtime.PendingConfusion["player_21424"] {
		t.Fatalf("expected auto confused attack to consume pending confusion, got %+v", runtime.PendingConfusion)
	}
	if pendingStartFor(runtime, "player_21424") != nil {
		t.Fatalf("expected no startCommand to be queued for confused actor, pendingStarts=%+v", runtime.PendingStarts)
	}
	if len(result.ClearBuffInfos) != 1 || result.ClearBuffInfos[0].TargetHandle != "player_21424" || result.ClearBuffInfos[0].Name != "混乱" {
		t.Fatalf("expected expired 混乱 to clear during auto flow, got %+v", result.ClearBuffInfos)
	}
}

func TestConfusionProcessActionCanSelectEnemyOrTeammate(t *testing.T) {
	found := map[string]bool{}
	for index := 0; index < 300; index += 1 {
		runtime := &Runtime{
			BattleID:         fmt.Sprintf("battle-confusion-target-%d", index),
			Round:            4,
			Phase:            PhaseCommand,
			ActiveHandle:     "player_21424",
			nextSequence:     1,
			ConsumedSequence: map[int]bool{},
			DefendingHandles: map[string]bool{},
			PendingConfusion: map[string]bool{"player_21424": true},
			Cells: []CellInfoPush{
				{Handle: "player_21424", Camp: CampTeam, HP: 1000, MaxHP: 1000, MP: 414, MaxMP: 414, Attack: 100, Hit: 100},
				{Handle: "player_21432", Camp: CampTeam, Name: "桥头的樵夫", HP: 800, MaxHP: 800, Defense: 0, Dog: 0},
				{Handle: "5431285114036433", Camp: CampEnemy, Name: "黄风寨夫人", HP: 1200, MaxHP: 1200, Attack: 1, Defense: 0, Hit: 100, Dog: 0},
			},
		}
		result := runtime.ProcessAction(ActionRequest{
			BattleID:     runtime.BattleID,
			ActorHandle:  "player_21424",
			CommandID:    CommandDuoDuanZhan,
			TargetHandle: "5431285114036433",
			Round:        4,
			Sequence:     1,
		})
		if result.ErrorCode != "" {
			t.Fatalf("expected confused command to be accepted as forced normal attack, got %+v", result)
		}
		if len(result.Actions) == 0 {
			t.Fatalf("expected confused ProcessAction to emit a normal attack, got %+v", result)
		}
		action := result.Actions[0]
		if action.ActionName != "普通攻击" || action.CommandID != CommandNormalAttack || action.ActorHandle != "player_21424" || action.SourceActionLabel != "nomalAtk" {
			t.Fatalf("expected ProcessAction to force captured confused normal attack, got %+v", action)
		}
		if action.TargetHandle != "player_21432" && action.TargetHandle != "5431285114036433" {
			t.Fatalf("expected confused target to be teammate or enemy, got %+v", action)
		}
		if action.Damage <= 0 {
			t.Fatalf("expected confused normal attack to damage the chosen target, got %+v", action)
		}
		if runtime.cellByHandle("player_21424").MP != 414 {
			t.Fatalf("expected forced normal attack not to spend requested skill MP, actor=%+v", runtime.cellByHandle("player_21424"))
		}
		if runtime.PendingConfusion["player_21424"] {
			t.Fatalf("expected ProcessAction to consume pending confusion, got %+v", runtime.PendingConfusion)
		}
		if !found[action.TargetHandle] {
			t.Logf("confusion ProcessAction rolled target %s in %s", action.TargetHandle, runtime.BattleID)
		}
		found[action.TargetHandle] = true
		if found["player_21432"] && found["5431285114036433"] {
			return
		}
	}
	t.Fatalf("expected 混乱 target roll to be able to select teammate and enemy, got %+v", found)
}

func TestFeixiandongLargerockFirePowerUsesCapturedAllTargetAction(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-feixiandong-fire-power",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
		Cells: []CellInfoPush{
			{
				Handle: "player_21424",
				Camp:   CampTeam,
				HP:     1555,
				MaxHP:  1555,
				MP:     275,
				MaxMP:  424,
			},
			{
				Handle:     "2654338502092795",
				Camp:       CampEnemy,
				Name:       "巨岩魔",
				DisplayURL: "monstermap/largerock.swf",
				HP:         1500,
				MaxHP:      1500,
				MP:         554,
				MaxMP:      564,
				Attack:     260,
			},
		},
	}
	actor := runtime.cellByHandle("2654338502092795")

	action := runtime.resolveAllTargetAttack(actor, runtime.livingCells(CampTeam), CommandEnemyFirePower)

	if action.ActionName != "赤焰击" || action.SourceActionLabel != "firePower" || action.TargetHandle != "all" || action.SourceMode != "1" {
		t.Fatalf("expected captured 赤焰击 all-target action, got %+v", action)
	}
	if action.Damage != 217 || action.TargetActionResults[0].Handle != "player_21424" || runtime.Cells[0].HP != 1338 || runtime.Cells[1].MP != 544 {
		t.Fatalf("expected 赤焰击 to use captured direct damage and MP cost, action=%+v cells=%+v", action, runtime.Cells)
	}
}

func TestFeixiandongMagicrockmanDeadLightUsesCapturedHPDamage(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-feixiandong-dead-light",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
		Cells: []CellInfoPush{
			{
				Handle: "player_21424",
				Camp:   CampTeam,
				HP:     1555,
				MaxHP:  1555,
				MP:     304,
				MaxMP:  424,
			},
			{
				Handle:     "9141338375384737",
				Camp:       CampEnemy,
				Name:       "岩化魔人",
				DisplayURL: "monstermap/magicrockman.swf",
				HP:         1500,
				MaxHP:      1500,
				MP:         554,
				MaxMP:      564,
				Attack:     240,
			},
		},
	}
	actor := runtime.cellByHandle("9141338375384737")

	action := runtime.resolveAllTargetAttack(actor, runtime.livingCells(CampTeam), CommandEnemyDeadLight)

	if action.ActionName != "死亡射线" || action.SourceActionLabel != "deadLight" || action.TargetHandle != "all" || action.SourceMode != "1" {
		t.Fatalf("expected captured 死亡射线 all-target action, got %+v", action)
	}
	if action.Damage != 274 || action.TargetMP != 0 || runtime.Cells[0].HP != 1281 || runtime.Cells[0].MP != 304 || runtime.Cells[1].MP != 544 {
		t.Fatalf("expected 死亡射线 to use captured HP damage without target MP drain, action=%+v cells=%+v", action, runtime.Cells)
	}
}

func TestFeixiandongMagicrockmanDoubleHitUsesCapturedLabel(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-feixiandong-double-hit",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
	}
	actor := &CellInfoPush{
		Handle:     "9141338375384737",
		Camp:       CampEnemy,
		Name:       "岩化魔人",
		DisplayURL: "monstermap/magicrockman.swf",
		Attack:     240,
		MP:         544,
		MaxMP:      564,
	}
	target := &CellInfoPush{
		Handle:  "player_21424",
		Camp:    CampTeam,
		HP:      1020,
		MaxHP:   1075,
		Defense: 0,
	}

	action := runtime.resolveAttack(actor, target, CommandEnemyDoubleHit)

	if action.ActionName != "双锤打" || action.SourceActionLabel != "doubleHit" {
		t.Fatalf("expected captured 双锤打 action, got %+v", action)
	}
	if action.Damage != 438 || action.TargetHP != 582 || actor.MP != 534 {
		t.Fatalf("expected 双锤打 to use captured physical damage path, got action=%+v target=%+v", action, target)
	}
}

func TestFeixiandongLargerockRollAttackUsesCapturedDamage(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-feixiandong-roll-attack",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
	}
	actor := &CellInfoPush{
		Handle:     "2654338502092795",
		Camp:       CampEnemy,
		Name:       "巨岩魔",
		DisplayURL: "monstermap/largerock.swf",
		Attack:     260,
		MP:         554,
		MaxMP:      564,
	}
	target := &CellInfoPush{
		Handle:  "player_21424",
		Camp:    CampTeam,
		HP:      773,
		MaxHP:   1075,
		Defense: 169,
	}

	action := runtime.resolveAttack(actor, target, CommandEnemyRollAtk)

	if action.ActionName != "滑行连击" || action.SourceActionLabel != "rollAttack" {
		t.Fatalf("expected captured 滑行连击 action, got %+v", action)
	}
	if action.Damage != 303 || action.TargetHP != 470 || actor.MP != 544 {
		t.Fatalf("expected 滑行连击 to use half physical defense and MP cost, got action=%+v target=%+v", action, target)
	}
}

func TestShihukuEnemySkillProfilesUseCapturedActionLabels(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-shihuku-skill-profiles",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
		Cells: []CellInfoPush{
			{Handle: "player_21424", Camp: CampTeam, HP: 1085, MaxHP: 1085, Defense: 120},
			{Handle: "player_21432", Camp: CampTeam, HP: 1010, MaxHP: 1010, Defense: 100},
			{Handle: "enemy_whorllion", Camp: CampEnemy, Name: "盘狮怪", DisplayURL: "monstermap/whorllion.swf", HP: 3000, MaxHP: 3000, MP: 384, MaxMP: 384, Attack: 328},
			{Handle: "enemy_chiluking", Camp: CampEnemy, Name: "蚩颅王", DisplayURL: "monstermap/chiluking.swf", HP: 6000, MaxHP: 6000, MP: 600, MaxMP: 600, Attack: 342},
		},
	}

	lion := runtime.resolveAttack(runtime.cellByHandle("enemy_whorllion"), runtime.cellByHandle("player_21424"), CommandEnemyLionRoars)
	if lion.ActionName != "狮吼" || lion.SourceActionLabel != "lionroars" || lion.SourceMode != "1" {
		t.Fatalf("expected shihuku whorllion 狮吼/lionroars action, got %+v", lion)
	}
	if lion.Damage != 353 {
		t.Fatalf("expected 狮吼 to use 1.26 damage multiplier and half physical defense, got %+v", lion)
	}
	if runtime.cellByHandle("enemy_whorllion").MP != 374 {
		t.Fatalf("expected 狮吼 to use captured 10 MP cost, got %+v", runtime.cellByHandle("enemy_whorllion"))
	}

	piece := runtime.resolveAttack(runtime.cellByHandle("enemy_chiluking"), runtime.cellByHandle("player_21424"), CommandEnemyPieceAtk)
	if piece.ActionName != "撕裂" || piece.SourceActionLabel != "pieceAttack" || piece.SourceMode != "1" {
		t.Fatalf("expected shihuku 撕裂/pieceAttack action, got %+v", piece)
	}
	if piece.Damage != 419 {
		t.Fatalf("expected shihuku 撕裂 to use 1.4 damage multiplier and half physical defense, got %+v", piece)
	}
	if runtime.cellByHandle("enemy_chiluking").MP != 590 {
		t.Fatalf("expected shihuku 撕裂 to use captured 10 MP cost, got %+v", runtime.cellByHandle("enemy_chiluking"))
	}
	if len(runtime.PendingBuffInfos) != 1 {
		t.Fatalf("expected shihuku 撕裂 hit to push 外伤 BuffInfo, got %+v", runtime.PendingBuffInfos)
	}
	buff := runtime.PendingBuffInfos[0]
	if buff.Name != "外伤" || buff.Display != "25.png" || buff.ReleaseHandle != "enemy_chiluking" || buff.TargetHandle != "player_21424" || buff.Round != 5 {
		t.Fatalf("expected shihuku 撕裂 BuffInfo metadata, got %+v", buff)
	}
	effect := runtime.StatusEffects["player_21424"].Effects["外伤"]
	if effect.SourceHandle != "enemy_chiluking" || effect.SourceSkill != "撕裂" || effect.SourceAttack != 342 || effect.TickMinPercent != 7 || effect.TickMaxPercent != 10 || effect.Rounds != 5 {
		t.Fatalf("expected shihuku 蚩颅王 外伤 to use source attack 7%%~10%%, got %+v", effect)
	}

	gold := runtime.resolveEnemyCommandActions(runtime.cellByHandle("enemy_chiluking"), runtime.cellByHandle("player_21424"), CommandEnemyGoldHit)[0]
	if gold.ActionName != "黄金穿刺" || gold.SourceActionLabel != "goldhit" || gold.TargetHandle != "all" || gold.SourceMode != "1" {
		t.Fatalf("expected shihuku 蚩颅王 黄金穿刺/goldhit all-target action, got %+v", gold)
	}
	if gold.Damage != 528 {
		t.Fatalf("expected shihuku 黄金穿刺 to use 1.72 damage multiplier and half physical defense, got %+v", gold)
	}
	if runtime.cellByHandle("enemy_chiluking").MP != 580 {
		t.Fatalf("expected shihuku 黄金穿刺 to use captured 10 MP cost once, got %+v", runtime.cellByHandle("enemy_chiluking"))
	}
}

func TestShihukuBlackshadowPieceAttackAppliesCapturedWoundPercent(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-shihuku-blackshadow-wound",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
		StatusEffects:    map[string]BattleStatusEffects{},
	}
	actor := &CellInfoPush{
		Handle:     "enemy_blackshadow",
		Camp:       CampEnemy,
		Name:       "黑影",
		DisplayURL: "monstermap/blackshadow.swf",
		HP:         2300,
		MaxHP:      2300,
		Attack:     300,
		Hit:        100,
	}
	target := &CellInfoPush{
		Handle:  "player_21424",
		Camp:    CampTeam,
		HP:      1000,
		MaxHP:   1555,
		Attack:  999,
		Defense: 0,
		Dog:     0,
	}

	action := runtime.resolveAttack(actor, target, CommandEnemyPieceAtk)

	if action.ActionName != "撕裂" || action.TargetActionStateCode == "1" {
		t.Fatalf("expected blackshadow 撕裂 hit action, got %+v", action)
	}
	if len(runtime.PendingBuffInfos) != 1 {
		t.Fatalf("expected blackshadow 撕裂 hit to push 外伤 BuffInfo, got %+v", runtime.PendingBuffInfos)
	}
	buff := runtime.PendingBuffInfos[0]
	if buff.Name != "外伤" || buff.Display != "25.png" || buff.ReleaseHandle != actor.Handle || buff.TargetHandle != target.Handle || buff.Round != 5 {
		t.Fatalf("expected blackshadow 外伤 BuffInfo metadata, got %+v", buff)
	}
	effect := runtime.StatusEffects[target.Handle].Effects["外伤"]
	if effect.SourceHandle != actor.Handle || effect.SourceSkill != "撕裂" || effect.SourceAttack != 300 || effect.TickMinPercent != 6 || effect.TickMaxPercent != 9 || effect.Rounds != 5 {
		t.Fatalf("expected blackshadow 外伤 to use captured source attack 6%%~9%%, got %+v", effect)
	}

	hpBeforeTick := target.HP
	statusActions, skipTurn := runtime.resolveStatusStartActions(target)
	if skipTurn || len(statusActions) != 1 || statusActions[0].ActionName != "外伤" {
		t.Fatalf("expected one blackshadow 外伤 tick action without skip, actions=%+v skip=%v", statusActions, skipTurn)
	}
	if statusActions[0].Damage < 18 || statusActions[0].Damage > 27 {
		t.Fatalf("expected blackshadow 外伤 tick to use source attack 300 * 6%%~9%%, got %+v", statusActions[0])
	}
	if target.HP != hpBeforeTick-statusActions[0].Damage {
		t.Fatalf("expected target HP to follow 外伤 tick damage, target=%+v action=%+v", target, statusActions[0])
	}
}

func TestShihukuPieceAttackDodgeSuppressesWound(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-shihuku-piece-dodge",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
		StatusEffects:    map[string]BattleStatusEffects{},
	}
	actor := &CellInfoPush{
		Handle:     "enemy_chiluking",
		Camp:       CampEnemy,
		Name:       "蚩颅王",
		DisplayURL: "monstermap/chiluking.swf",
		HP:         6000,
		MaxHP:      6000,
		Attack:     342,
		Hit:        0,
	}
	target := &CellInfoPush{
		Handle: "player_21424",
		Camp:   CampTeam,
		HP:     1000,
		MaxHP:  1555,
		Dog:    1,
	}

	action := runtime.resolveAttack(actor, target, CommandEnemyPieceAtk)

	if action.TargetActionStateCode != "1" || action.Damage != 0 {
		t.Fatalf("expected shihuku 撕裂 dodge action, got %+v", action)
	}
	if len(runtime.PendingBuffInfos) != 0 {
		t.Fatalf("expected shihuku 撕裂 dodge to suppress 外伤 BuffInfo, got %+v", runtime.PendingBuffInfos)
	}
	if len(runtime.StatusEffects) != 0 {
		t.Fatalf("expected shihuku 撕裂 dodge to leave status effects empty, got %+v", runtime.StatusEffects)
	}
}

func TestShihukuEnemyBattleCommandUsesCapturedActionRatios(t *testing.T) {
	target := &CellInfoPush{Handle: "player_21424", Camp: CampTeam, HP: 1085, MaxHP: 1085}
	testCases := []struct {
		name        string
		displayURL  string
		expected    string
		chanceLabel string
	}{
		{name: "黑影", displayURL: "monstermap/blackshadow.swf", expected: CommandEnemyPieceAtk, chanceLabel: "pieceAttack 7/100"},
		{name: "盘狮怪", displayURL: "monstermap/whorllion.swf", expected: CommandEnemyLionRoars, chanceLabel: "lionroars 20/100"},
		{name: "蚩颅王", displayURL: "monstermap/chiluking.swf", expected: CommandEnemyGoldHit, chanceLabel: "goldhit 19/100"},
		{name: "蚩颅王", displayURL: "monstermap/chiluking.swf", expected: CommandEnemyPieceAtk, chanceLabel: "pieceAttack 11/100 after goldhit misses"},
	}

	for _, testCase := range testCases {
		enemy := &CellInfoPush{
			Handle:     "enemy_" + testCase.expected,
			Camp:       CampEnemy,
			Name:       testCase.name,
			DisplayURL: testCase.displayURL,
			HP:         3000,
			MaxHP:      3000,
			MP:         600,
			MaxMP:      600,
		}
		battleID, ok := findBattleIDForEnemyCommand(testCase.expected, enemy, target)
		if !ok {
			t.Fatalf("expected to find deterministic battle id for %s %s", testCase.name, testCase.chanceLabel)
		}
		runtime := &Runtime{BattleID: battleID, Round: 1, nextSequence: 1}
		if command := runtime.enemyBattleCommand(enemy, target); command != testCase.expected {
			t.Fatalf("expected %s to choose %s from captured %s, got %s with battle id %s", testCase.name, testCase.expected, testCase.chanceLabel, command, battleID)
		}
	}

	powerTiger := &CellInfoPush{Handle: "enemy_powertiger", Camp: CampEnemy, Name: "蛮虎怪", DisplayURL: "monstermap/powertiger.swf", HP: 3200, MaxHP: 3200, MP: 600}
	if command := (&Runtime{BattleID: "battle-shihuku-powertiger", Round: 1, nextSequence: 1}).enemyBattleCommand(powerTiger, target); command != CommandEnemyAttack {
		t.Fatalf("expected shihuku powertiger to keep normal attack only, got %s", command)
	}
}

func TestRobotawlEnemyBattleCommandCanUseCapturedRoundAtk(t *testing.T) {
	target := &CellInfoPush{Handle: "player_21424", Camp: CampTeam, HP: 1085, MaxHP: 1085}
	enemy := &CellInfoPush{
		Handle:     "enemy_robotawl",
		Camp:       CampEnemy,
		Name:       "机木锥兵",
		DisplayURL: "monstermap/robotawl.swf",
		HP:         1330,
		MaxHP:      1330,
		MP:         20,
		MaxMP:      20,
		Attack:     624,
	}
	battleID, ok := findBattleIDForEnemyCommand(CommandEnemyRoundAtk, enemy, target)
	if !ok {
		t.Fatal("expected to find deterministic battle id for 机木锥兵 轮转刺伤 23/100 capture-backed local approximation")
	}

	runtime := &Runtime{BattleID: battleID, Round: 1, nextSequence: 1}
	if command := runtime.enemyBattleCommand(enemy, target); command != CommandEnemyRoundAtk {
		t.Fatalf("expected 机木锥兵 to choose captured 轮转刺伤/roundatk, got %s with battle id %s", command, battleID)
	}
}

func TestYaozhisenEnemyCommandsUseCapturedAllTargetLabels(t *testing.T) {
	target := &CellInfoPush{Handle: "player_21424", Camp: CampTeam, HP: 1085, MaxHP: 1085, Hit: 400, Dog: 200}
	testCases := []struct {
		name        string
		displayURL  string
		expected    string
		actionName  string
		actionLabel string
	}{
		{name: "机木斧兵", displayURL: "monstermap/robotax.swf", expected: CommandEnemyRulingAx, actionName: "裁决之斧", actionLabel: "rulingax"},
		{name: "机木妖帅", displayURL: "monstermap/robothmarshal.swf", expected: CommandEnemyVacuumKill, actionName: "真空猎杀", actionLabel: "vacuumkilled"},
	}

	for _, testCase := range testCases {
		enemy := &CellInfoPush{
			Handle:     "enemy_" + testCase.expected,
			Camp:       CampEnemy,
			Name:       testCase.name,
			DisplayURL: testCase.displayURL,
			HP:         6500,
			MaxHP:      6500,
			MP:         20,
			MaxMP:      20,
			Attack:     360,
		}
		battleID, ok := findBattleIDForEnemyCommand(testCase.expected, enemy, target)
		if !ok {
			t.Fatalf("expected deterministic battle id for %s", testCase.actionName)
		}
		runtime := &Runtime{
			BattleID:         battleID,
			Round:            1,
			nextSequence:     1,
			DefendingHandles: map[string]bool{},
			StatusEffects:    map[string]BattleStatusEffects{},
			Cells: []CellInfoPush{
				*enemy,
				*target,
				{Handle: "player_21432", Camp: CampTeam, HP: 1000, MaxHP: 1000, Hit: 300, Dog: 150},
			},
		}
		actor := runtime.cellByHandle(enemy.Handle)
		if command := runtime.enemyBattleCommand(actor, runtime.cellByHandle(target.Handle)); command != testCase.expected {
			t.Fatalf("expected %s to choose %s, got %s", testCase.name, testCase.expected, command)
		}
		action := runtime.resolveAllTargetAttack(actor, runtime.livingCells(CampTeam), testCase.expected)
		if action.ActionName != testCase.actionName || action.SourceActionLabel != testCase.actionLabel || action.TargetHandle != "all" || len(action.TargetActionResults) != 2 {
			t.Fatalf("expected captured all-target action for %s, got %+v", testCase.name, action)
		}
		if actor.MP != 10 {
			t.Fatalf("expected captured %s to consume 10 MP, got %+v", testCase.actionName, actor)
		}
	}
}

func TestRulingAxSlowStatusReducesHitAndDodgeByCapturedThirtyPercent(t *testing.T) {
	runtime := &Runtime{StatusEffects: map[string]BattleStatusEffects{}}
	target := &CellInfoPush{Handle: "player_21424", Camp: CampTeam, HP: 1000, MaxHP: 1000, Hit: 400, Dog: 200}
	actor := &CellInfoPush{Handle: "enemy_robotax", Camp: CampEnemy, Name: "机木斧兵"}

	if !runtime.applySlownessStatusEffect(actor, target, BattleStatusEffect{
		Name:                     "迟钝",
		Display:                  "16.png",
		Description:              "降低对象命中和回避",
		Rounds:                   enemyRobotaxRulingAxSlownessRounds,
		HitDodgeReductionPercent: enemyRobotaxRulingAxSlownessPct,
	}) {
		t.Fatal("expected ruling ax slow status to apply")
	}
	if target.Hit != 280 || target.Dog != 140 {
		t.Fatalf("expected captured ruling ax slow to reduce hit/dodge by 30%%, got %+v", target)
	}
	buff := runtime.StatusEffects[target.Handle].Effects["迟钝"]
	if buff.Rounds != 2 || buff.Display != "16.png" || buff.Description != "降低对象120点命中和60点回避" || buff.HitDodgeReductionPercent != 30 || buff.VisualOnly {
		t.Fatalf("expected capture-backed 30%% ruling ax slow metadata, got %+v", buff)
	}
}

func TestRulingAxAppliesCapturedSlowAndMPCostThroughCommandProfile(t *testing.T) {
	var runtime *Runtime
	var enemy *CellInfoPush
	var target *CellInfoPush
	for index := 0; index < 400; index += 1 {
		candidate := &Runtime{
			BattleID:         fmt.Sprintf("battle-rulingax-slow-%d", index),
			Round:            1,
			nextSequence:     1,
			DefendingHandles: map[string]bool{},
			StatusEffects:    map[string]BattleStatusEffects{},
		}
		candidateEnemy := &CellInfoPush{
			Handle:     "enemy_robotax",
			Camp:       CampEnemy,
			Name:       "机木斧兵",
			DisplayURL: "monstermap/robotax.swf",
			HP:         1755,
			MaxHP:      1755,
			MP:         20,
			MaxMP:      20,
			Attack:     360,
			Hit:        100,
		}
		candidateTarget := &CellInfoPush{
			Handle:  "player_21424",
			Camp:    CampTeam,
			HP:      1215,
			MaxHP:   1215,
			Hit:     280,
			Dog:     138,
			Defense: 0,
		}
		if candidate.hashBattleRollWithSalt(candidateEnemy, candidateTarget, CommandEnemyRulingAx, "status:迟钝") < enemyRobotaxRulingAxSlownessChance {
			runtime = candidate
			enemy = candidateEnemy
			target = candidateTarget
			break
		}
	}
	if runtime == nil {
		t.Fatal("expected deterministic 裁决之斧迟钝 roll below candidate status chance")
	}

	action := runtime.resolveAttack(enemy, target, CommandEnemyRulingAx)

	if action.ActionName != "裁决之斧" || action.SourceActionLabel != "rulingax" || action.TargetActionStateCode == "1" {
		t.Fatalf("expected captured 裁决之斧 hit action, got %+v", action)
	}
	if enemy.MP != 10 {
		t.Fatalf("expected captured 裁决之斧 to consume 10 MP, got %+v", enemy)
	}
	if target.Hit != 196 || target.Dog != 97 {
		t.Fatalf("expected captured 裁决之斧 to reduce target hit/dodge by 30%%, got %+v", target)
	}
	if len(runtime.PendingBuffInfos) != 1 {
		t.Fatalf("expected one 裁决之斧迟钝 BuffInfo, got %+v", runtime.PendingBuffInfos)
	}
	buff := runtime.PendingBuffInfos[0]
	if buff.Name != "迟钝" || buff.Display != "16.png" || buff.Round != 2 || buff.Description != "降低对象84点命中和41点回避" || buff.ReleaseHandle != enemy.Handle || buff.TargetHandle != target.Handle {
		t.Fatalf("expected captured 裁决之斧迟钝 BuffInfo, got %+v", buff)
	}
}

func TestYaozhisenRobotSkillsRequireCapturedTenMP(t *testing.T) {
	target := &CellInfoPush{Handle: "player_21424", Camp: CampTeam, HP: 1085, MaxHP: 1085}
	testCases := []struct {
		name       string
		displayURL string
		commandID  string
	}{
		{name: "机木锥兵", displayURL: "monstermap/robotawl.swf", commandID: CommandEnemyRoundAtk},
		{name: "机木斧兵", displayURL: "monstermap/robotax.swf", commandID: CommandEnemyRulingAx},
		{name: "机木妖帅", displayURL: "monstermap/robothmarshal.swf", commandID: CommandEnemyVacuumKill},
	}

	for _, testCase := range testCases {
		enemy := &CellInfoPush{
			Handle:     "enemy_mp_gate_" + testCase.commandID,
			Camp:       CampEnemy,
			Name:       testCase.name,
			DisplayURL: testCase.displayURL,
			HP:         2000,
			MaxHP:      2000,
			MP:         enemyRobotSkillMPCost,
			MaxMP:      enemyRobotSkillMPCost,
			Attack:     360,
		}
		battleID, ok := findBattleIDForEnemyCommand(testCase.commandID, enemy, target)
		if !ok {
			t.Fatalf("expected deterministic skill roll for %s", testCase.name)
		}
		withoutMP := *enemy
		withoutMP.MP = enemyRobotSkillMPCost - 1
		runtime := &Runtime{BattleID: battleID, Round: 1, nextSequence: 1}
		if command := runtime.enemyBattleCommand(&withoutMP, target); command != CommandEnemyAttack {
			t.Fatalf("expected %s to fall back to normal attack below 10 MP, got %s", testCase.name, command)
		}
	}
}

func TestBainianChongjingUsesCapturedRampageAndChaosHit(t *testing.T) {
	bainian, ok := sourceVisibleMonsterConfigForHandle("171", "7893833328746190")
	if !ok {
		t.Fatal("expected map171 visible 百年虫精 config")
	}
	if !sourceEnemyCanRampage(&bainian.Cell) {
		t.Fatalf("expected 百年虫精 to use captured 暴走之力, cell=%+v", bainian.Cell)
	}
	if !sourceEnemyCanChaosHit(&bainian.Cell) {
		t.Fatalf("expected 百年虫精 to use captured 混沌击, cell=%+v", bainian.Cell)
	}
	if enemyChaosHitChance != 46 {
		t.Fatalf("expected capture-backed 百年虫精混沌击 chance 46, got %d", enemyChaosHitChance)
	}
	candidate, ok := sourceBattleRewardCandidateForCell("171", "百年虫精", 8000)
	if !ok || candidate.ExpDelta != 0 {
		t.Fatalf("expected zero-exp 百年虫精 reward candidate, got %+v", candidate)
	}
	for _, expected := range []sourceBattleRewardDropRate{
		{ItemName: "宠物成长药剂", Quantity: 6, Numerator: 4, Denominator: 4},
		{ItemName: "铜钱", Quantity: 500, Numerator: 4, Denominator: 4},
		{ItemName: "魔匣", Quantity: 2, Numerator: 3, Denominator: 4},
		{ItemName: "阴阳结", Quantity: 1, Numerator: 3, Denominator: 4},
		{ItemName: "回魂丹", Quantity: 1, Numerator: 3, Denominator: 4},
	} {
		if actual := requireSourceBattleRewardDropRate(t, candidate.DropRates, expected.ItemName); actual != expected {
			t.Fatalf("expected 百年虫精 %s rate %+v, got %+v", expected.ItemName, expected, actual)
		}
	}
	if huihun, ok := session.CapturedRoleItemTemplate("回魂丹"); !ok || huihun.Display != "225.png" || !strings.Contains(huihun.Description, "f_i_回魂丹") {
		t.Fatalf("expected captured 回魂丹 item template, got %+v, found=%t", huihun, ok)
	}
	group, ok := sourceVisibleMonsterConfigsForHandle("171", "7893833328746190")
	if !ok || len(group) != 4 {
		t.Fatalf("expected four-enemy 百年虫精 encounter, got %+v", group)
	}
	expectedAttackRanges := map[string]battleAttackRange{
		"7893833328746190": {Min: 197, Max: 208},
		"7895833328747103": {Min: 222, Max: 245},
		"7897833328748728": {Min: 216, Max: 321},
		"7899833328749140": {Min: 216, Max: 321},
	}
	expectedSweepSpearRanges := map[string]battleAttackRange{
		"7895833328747103": {Min: 209, Max: 209},
		"7897833328748728": {Min: 207, Max: 314},
		"7899833328749140": {Min: 207, Max: 314},
	}
	for _, enemy := range group {
		if actual := (battleAttackRange{Min: enemy.AttackMin, Max: enemy.AttackMax}); actual != expectedAttackRanges[enemy.Cell.Handle] {
			t.Fatalf("expected capture-backed 百年虫精 attack range for %s, got %+v", enemy.Cell.Handle, actual)
		}
		if actual := (battleAttackRange{Min: enemy.SweepSpearAttackMin, Max: enemy.SweepSpearAttackMax}); enemy.Cell.Name == "被控制的民兵" && actual != expectedSweepSpearRanges[enemy.Cell.Handle] {
			t.Fatalf("expected capture-backed 百年虫精 sweepspear range for %s, got %+v", enemy.Cell.Handle, actual)
		}
	}

	runtime := &Runtime{
		BattleID:              "battle-bainian-chongjing",
		Round:                 1,
		nextSequence:          1,
		DefendingHandles:      map[string]bool{},
		EnemyAttackRanges:     expectedAttackRanges,
		EnemySweepSpearRanges: expectedSweepSpearRanges,
	}
	actor := bainian.Cell.withBattleIDAndSlot(runtime.BattleID, 0)
	rampage := runtime.resolveEnemyRampageActions(&actor)
	if len(rampage) != 1 || rampage[0].ActionName != "暴走之力" || rampage[0].SourceActionLabel != "battleStand" || rampage[0].TargetHandle != actor.Handle {
		t.Fatalf("expected 百年虫精 rampage battleStand self action, got %+v", rampage)
	}
	if len(runtime.PendingBuffInfos) != 1 || runtime.PendingBuffInfos[0].Round != 51 || !strings.Contains(runtime.PendingBuffInfos[0].Description, "还有 51 回合暴走") {
		t.Fatalf("expected captured 百年虫精 first rampage countdown 51, got %+v", runtime.PendingBuffInfos)
	}
	for _, militia := range group[1:] {
		if sourceEnemyCanRampage(&militia.Cell) || len(runtime.resolveEnemyRampageActions(&militia.Cell)) != 0 {
			t.Fatalf("expected militia to stay out of captured rampage, got %+v", militia.Cell)
		}
		if !sourceEnemyCanMilitiaSweepSpear(&militia.Cell) {
			t.Fatalf("expected controlled militia sweepspear eligibility, got %+v", militia.Cell)
		}
	}
	if sourceEnemyCanMilitiaSweepSpear(&CellInfoPush{Name: "点券盗贼", DisplayURL: "monstermap/militia.swf"}) {
		t.Fatal("point-coupon thief must not inherit controlled militia sweepspear")
	}
	if actor.MP != bainian.Cell.MaxMP {
		t.Fatalf("expected rampage to keep MP, actor=%+v", actor)
	}

	target := &CellInfoPush{
		Handle:     "player_21424",
		Camp:       CampTeam,
		Name:       "恐龙抗狼1",
		MaxHP:      1895,
		HP:         1895,
		Defense:    100,
		MgcDefense: 80,
	}
	// Force chaos command path.
	action := runtime.resolveAttack(&actor, target, CommandEnemyChaosHit)
	if action.ActionName != "混沌击" || action.SourceActionLabel != "nomalAtk" {
		t.Fatalf("expected 混沌击 broadcast with nomalAtk animation, got %+v", action)
	}
	if bainian.Cell.DamageDefenseType != "magic" {
		t.Fatalf("expected magic defense type for 百年虫精, got %+v", bainian.Cell)
	}

	var ranger *CellInfoPush
	for _, militia := range group[1:] {
		if militia.Cell.Vocation == "游侠+" {
			cell := militia.Cell.withBattleIDAndSlot(runtime.BattleID, 1)
			ranger = &cell
			break
		}
	}
	if ranger == nil {
		t.Fatal("expected captured ranger militia")
	}
	targetOne := CellInfoPush{Handle: "player_sweep_1", Camp: CampTeam, HP: 1000, MaxHP: 1000, Defense: 290}
	targetTwo := CellInfoPush{Handle: "player_sweep_2", Camp: CampTeam, HP: 1000, MaxHP: 1000, Defense: 299}
	sweepSpearSelected := false
	for index := 0; index < 100; index += 1 {
		candidate := &Runtime{BattleID: fmt.Sprintf("battle-bainian-sweepspear-%d", index), Round: 1, nextSequence: 1}
		if candidate.enemyBattleCommand(ranger, &targetOne) == CommandEnemySweepSpear {
			sweepSpearSelected = true
			break
		}
	}
	if !sweepSpearSelected {
		t.Fatal("expected capture-backed militia sweepspear to be selectable by enemy AI")
	}
	runtime.Cells = []CellInfoPush{targetOne, targetTwo, *ranger}
	ranger = runtime.cellByHandle(ranger.Handle)
	runtime.EnemyAttackRanges[ranger.Handle] = expectedAttackRanges["7895833328747103"]
	runtime.EnemySweepSpearRanges[ranger.Handle] = expectedSweepSpearRanges["7895833328747103"]
	ranger.Fat = 0
	defer useSourceBattleAttackRoll(func(int) int { return 0 })()
	normalAction := runtime.resolveAttack(ranger, runtime.cellByHandle(targetOne.Handle), CommandEnemyAttack)
	if normalAction.Damage != 77 || normalAction.TargetHP != 923 {
		t.Fatalf("expected militia normal attack to use its 222 minimum instead of sweepspear range, action=%+v", normalAction)
	}
	runtime.cellByHandle(targetOne.Handle).HP = 1000
	sweepActions := runtime.resolveEnemyCommandActions(ranger, runtime.cellByHandle(targetOne.Handle), CommandEnemySweepSpear)
	if len(sweepActions) != 1 || sweepActions[0].ActionName != "单枪横扫" || sweepActions[0].TargetHandle != "all" || sweepActions[0].SourceActionLabel != "sweepspear" || len(sweepActions[0].TargetActionResults) != 2 || runtime.cellByHandle(ranger.Handle).MP != 570 || runtime.cellByHandle(targetOne.Handle).HP != 873 || runtime.cellByHandle(targetTwo.Handle).HP != 878 {
		t.Fatalf("expected militia sweepspear all-target MP and half-defense behavior, actions=%+v cells=%+v", sweepActions, runtime.Cells)
	}

	defer useSourceEncounterRoll(func(maxExclusive int) int { return 0 })()
	rewards := (&Runtime{MapID: "171", Cells: []CellInfoPush{bainian.Cell}}).buildOver(CampTeam).Result
	wantRewards := []string{"宠物成长药剂x6", "铜钱x500", "魔匣x2", "阴阳结x1", "回魂丹x1"}
	if !reflect.DeepEqual(rewards.Items, wantRewards) {
		t.Fatalf("expected low-roll 百年虫精 rewards %+v, got %+v", wantRewards, rewards.Items)
	}
}

func TestXiongluBeardeerUsesCapturedRampageAndSkills(t *testing.T) {
	beardeer, ok := sourceVisibleMonsterConfigForHandle("203", "4264636384163425")
	if !ok {
		t.Fatal("expected map203 visible 熊鹿 config")
	}
	if beardeer.Cell.Name != "熊鹿" || beardeer.Cell.DisplayURL != "monstermap/beardeer.swf" || beardeer.Cell.Level != 40 || beardeer.Cell.MaxHP != 20000 || beardeer.Cell.MaxMP != 3102 || beardeer.Cell.CommandLabel != "法术普通攻击" || beardeer.Cell.DamageDefenseType != "magic" {
		t.Fatalf("unexpected 熊鹿 config: %+v", beardeer.Cell)
	}
	if !sourceEnemyCanRampage(&beardeer.Cell) {
		t.Fatalf("expected 熊鹿 to use captured 暴走之力, cell=%+v", beardeer.Cell)
	}
	if !sourceEnemyCanThunderstorm(&beardeer.Cell) {
		t.Fatalf("expected 熊鹿 to use captured 雷鸣怒吼, cell=%+v", beardeer.Cell)
	}
	if !sourceEnemyCanAngleCurse(&beardeer.Cell) {
		t.Fatalf("expected 熊鹿 to use captured 角念, cell=%+v", beardeer.Cell)
	}
	if sourceEnemyRampageMaxRounds(&beardeer.Cell) != enemyBainianRampageMaxRounds {
		t.Fatalf("expected 熊鹿 rampage max rounds %d, got %d", enemyBainianRampageMaxRounds, sourceEnemyRampageMaxRounds(&beardeer.Cell))
	}

	group, ok := sourceVisibleMonsterConfigsForHandle("203", "4264636384163425")
	if !ok || len(group) != 4 {
		t.Fatalf("expected four-enemy 熊鹿 encounter, got %+v", group)
	}
	if group[0].Cell.Handle != "4264636384163425" || group[0].Cell.Name != "熊鹿" {
		t.Fatalf("expected boss first in 熊鹿 encounter, got %+v", group[0].Cell)
	}
	for _, companion := range group[1:] {
		if companion.Cell.Name != "机木玄师" || companion.Cell.DisplayURL != "monstermap/robothyun.swf" || companion.Cell.MaxHP != 2200 || companion.Cell.MaxMP != 860 {
			t.Fatalf("unexpected 机木玄师 companion: %+v", companion.Cell)
		}
		if sourceEnemyCanRampage(&companion.Cell) || sourceEnemyCanThunderstorm(&companion.Cell) {
			t.Fatalf("companions must not inherit 熊鹿 rampage/thunderstorm, got %+v", companion.Cell)
		}
		if !sourceEnemyCanRobothyunRobotUp(&companion.Cell) {
			t.Fatalf("expected 机木玄师 robotup eligibility, got %+v", companion.Cell)
		}
	}

	runtime := &Runtime{
		BattleID:         "battle-xionglu-beardeer",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
	}
	actor := beardeer.Cell.withBattleIDAndSlot(runtime.BattleID, 0)
	rampage := runtime.resolveEnemyRampageActions(&actor)
	if len(rampage) != 1 || rampage[0].ActionName != "暴走之力" || rampage[0].SourceActionLabel != "battleStand" || rampage[0].TargetHandle != actor.Handle {
		t.Fatalf("expected 熊鹿 rampage battleStand self action, got %+v", rampage)
	}
	if len(runtime.PendingBuffInfos) != 1 || runtime.PendingBuffInfos[0].Round != 51 || !strings.Contains(runtime.PendingBuffInfos[0].Description, "还有 51 回合暴走") {
		t.Fatalf("expected captured 熊鹿 first rampage countdown 51, got %+v", runtime.PendingBuffInfos)
	}

	target := &CellInfoPush{
		Handle:     "player_xionglu_1",
		Camp:       CampTeam,
		Name:       "恐龙抗狼1",
		MaxHP:      1895,
		HP:         1895,
		Defense:    100,
		MgcDefense: 80,
	}
	storm := runtime.resolveAttack(&actor, target, CommandEnemyThunderstorm)
	if storm.ActionName != "雷鸣怒吼" || storm.SourceActionLabel != "thunderstorm" {
		t.Fatalf("expected 雷鸣怒吼 thunderstorm action, got %+v", storm)
	}
	profileStorm := runtime.battleCommandProfile(&actor, CommandEnemyThunderstorm)
	profileNormal := runtime.battleCommandProfile(&actor, CommandEnemyAttack)
	profileCurse := runtime.battleCommandProfile(&actor, CommandEnemyAngleCurse)
	if profileStorm.DamageMultiplier != enemyThunderstormDamageMultiplier {
		t.Fatalf("expected thunderstorm damage multiplier %v, got %v", enemyThunderstormDamageMultiplier, profileStorm.DamageMultiplier)
	}
	if profileNormal.DamageMultiplier != 1 || profileCurse.DamageMultiplier != 1 {
		t.Fatalf("expected nomalAtk/anglecurse multiplier 1, got normal=%v curse=%v", profileNormal.DamageMultiplier, profileCurse.DamageMultiplier)
	}
	defense := runtime.effectiveBattleDefense(&actor, target, false, profileStorm.DefenseType)
	stormDamage := runtime.baseBattleDamage(&actor, profileStorm, defense)
	normalDamage := runtime.baseBattleDamage(&actor, profileNormal, defense)
	expectedStorm := maxInt(1, int(float64(actor.Attack)*enemyThunderstormDamageMultiplier+0.5)-defense)
	expectedNormal := maxInt(1, int(float64(actor.Attack)*1+0.5)-defense)
	if stormDamage != expectedStorm || normalDamage != expectedNormal {
		t.Fatalf("expected thunderstorm %d and normal %d under defense %d attack %d, got storm=%d normal=%d", expectedStorm, expectedNormal, defense, actor.Attack, stormDamage, normalDamage)
	}
	if stormDamage < normalDamage*2-1 {
		t.Fatalf("expected capture-backed thunderstorm ~2x normal damage, storm=%d normal=%d", stormDamage, normalDamage)
	}
	curse := runtime.resolveAttack(&actor, target, CommandEnemyAngleCurse)
	if curse.ActionName != "角念" || curse.SourceActionLabel != "anglecurse" {
		t.Fatalf("expected 角念 anglecurse action, got %+v", curse)
	}
	if profileCurse.StatusName != "封印" || profileCurse.StatusDisplay != "19.png" || profileCurse.StatusChance != enemyAngleCurseSealChance || profileCurse.StatusRounds != enemyAngleCurseSealRounds || profileCurse.SkipTurn {
		t.Fatalf("expected 角念 to carry 封印 status 20%%/3r without skip-turn, got %+v", profileCurse)
	}

	targetOne := CellInfoPush{Handle: "player_xionglu_a", Camp: CampTeam, HP: 2000, MaxHP: 2000, MgcDefense: 100}
	targetTwo := CellInfoPush{Handle: "player_xionglu_b", Camp: CampTeam, HP: 2000, MaxHP: 2000, MgcDefense: 100}
	actor.MP = beardeer.Cell.MaxMP
	runtime.Cells = []CellInfoPush{targetOne, targetTwo, actor}
	actorPtr := runtime.cellByHandle(actor.Handle)
	defer useSourceBattleAttackRoll(func(int) int { return 0 })()
	allActions := runtime.resolveEnemyCommandActions(actorPtr, runtime.cellByHandle(targetOne.Handle), CommandEnemyThunderstorm)
	if len(allActions) != 1 || allActions[0].ActionName != "雷鸣怒吼" || allActions[0].TargetHandle != "all" || allActions[0].SourceActionLabel != "thunderstorm" || len(allActions[0].TargetActionResults) != 2 {
		t.Fatalf("expected 雷鸣怒吼 all-target behavior, actions=%+v cells=%+v", allActions, runtime.Cells)
	}
	if runtime.cellByHandle(actor.Handle).MP != beardeer.Cell.MaxMP-enemyThunderstormMPCost {
		t.Fatalf("expected thunderstorm MP cost %d, actor=%+v", enemyThunderstormMPCost, runtime.cellByHandle(actor.Handle))
	}

	stormSelected := false
	curseSelected := false
	for index := 0; index < 200; index += 1 {
		candidate := &Runtime{BattleID: fmt.Sprintf("battle-xionglu-ai-%d", index), Round: 1, nextSequence: 1}
		probe := beardeer.Cell.withBattleIDAndSlot(candidate.BattleID, 0)
		probe.MP = beardeer.Cell.MaxMP
		command := candidate.enemyBattleCommand(&probe, target)
		if command == CommandEnemyThunderstorm {
			stormSelected = true
		}
		if command == CommandEnemyAngleCurse {
			curseSelected = true
		}
		if stormSelected && curseSelected {
			break
		}
	}
	if !stormSelected {
		t.Fatal("expected capture-backed thunderstorm to be selectable by enemy AI")
	}
	if !curseSelected {
		t.Fatal("expected capture-backed anglecurse to be selectable by enemy AI")
	}

	candidate, ok := sourceBattleRewardCandidateForCell("203", "熊鹿", 20000)
	if !ok {
		t.Fatalf("expected 熊鹿 reward candidate for map203 HP20000")
	}
	if candidate.ExpDelta != 0 {
		t.Fatalf("expected zero-exp 熊鹿 reward candidate sample, got %+v", candidate)
	}
}

func TestXiongluBeardeerAngleCurseCanApplySeal(t *testing.T) {
	// Deterministic seed: battleId/handles chosen so status:封印 roll < 20.
	// Seal reuses equipment 封印 contract (19.png, block skills, no skip-turn).
	// Seal rate 20% is design default, not capture-confirmed.
	runtime := &Runtime{
		BattleID:         "battle-xionglu-anglecurse-seal",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
		StatusEffects:    map[string]BattleStatusEffects{},
		PendingBuffInfos: nil,
	}
	actor := &CellInfoPush{
		Handle:            "4264636384163425",
		Camp:              CampEnemy,
		Name:              "熊鹿",
		DisplayURL:        "monstermap/beardeer.swf",
		Level:             40,
		MaxHP:             20000,
		HP:                20000,
		MaxMP:             3102,
		MP:                3102,
		Attack:            400,
		CommandLabel:      "法术普通攻击",
		DamageDefenseType: "magic",
	}
	// Probe rolls until we find a target handle that seals and one that misses,
	// so the test does not hardcode a fragile hash seed.
	var sealedTarget *CellInfoPush
	var missedTarget *CellInfoPush
	for index := 0; index < 400; index += 1 {
		handle := fmt.Sprintf("player_anglecurse_seal_%d", index)
		target := &CellInfoPush{
			Handle:     handle,
			Camp:       CampTeam,
			Name:       "测试目标",
			MaxHP:      5000,
			HP:         5000,
			MgcDefense: 0,
			Defense:    0,
		}
		probe := &Runtime{
			BattleID:         runtime.BattleID,
			Round:            1,
			nextSequence:     1,
			DefendingHandles: map[string]bool{},
		}
		roll := probe.hashBattleRollWithSalt(actor, target, CommandEnemyAngleCurse, "status:封印")
		if sealedTarget == nil && roll < enemyAngleCurseSealChance {
			sealedTarget = target
		}
		if missedTarget == nil && roll >= enemyAngleCurseSealChance {
			missedTarget = target
		}
		if sealedTarget != nil && missedTarget != nil {
			break
		}
	}
	if sealedTarget == nil || missedTarget == nil {
		t.Fatalf("failed to locate deterministic seal hit/miss seeds under 20%% chance")
	}

	// Force hit path: high hit / zero dog so dodge cannot suppress status.
	actor.Hit = 999
	sealedTarget.Dog = 0
	missedTarget.Dog = 0
	defer useSourceBattleAttackRoll(func(int) int { return 0 })()

	runtime.PendingBuffInfos = nil
	runtime.StatusEffects = map[string]BattleStatusEffects{}
	action := runtime.resolveAttack(actor, sealedTarget, CommandEnemyAngleCurse)
	if action.ActionName != "角念" || action.SourceActionLabel != "anglecurse" || action.Damage <= 0 {
		t.Fatalf("expected damaging 角念 action on seal seed, got %+v", action)
	}
	if len(runtime.PendingBuffInfos) != 1 {
		t.Fatalf("expected one 封印 BuffInfo from 角念, got %+v", runtime.PendingBuffInfos)
	}
	buff := runtime.PendingBuffInfos[0]
	if buff.Name != "封印" || buff.Display != "19.png" || buff.Description != "作用时间内对象无法使用技能" || buff.Round != enemyAngleCurseSealRounds || buff.ReleaseHandle != actor.Handle || buff.TargetHandle != sealedTarget.Handle {
		t.Fatalf("expected captured 封印 BuffInfo metadata from 角念, got %+v", buff)
	}
	effect := runtime.StatusEffects[sealedTarget.Handle].Effects["封印"]
	if effect.Name != "封印" || effect.Display != "19.png" || effect.Rounds != enemyAngleCurseSealRounds || effect.SourceHandle != actor.Handle || effect.SourceSkill != "角念" || effect.SkipTurn {
		t.Fatalf("expected runtime 封印 status without skip-turn from 角念, got %+v", runtime.StatusEffects)
	}

	// Miss seed must not apply seal.
	runtime.PendingBuffInfos = nil
	runtime.StatusEffects = map[string]BattleStatusEffects{}
	miss := runtime.resolveAttack(actor, missedTarget, CommandEnemyAngleCurse)
	if miss.ActionName != "角念" || miss.Damage <= 0 {
		t.Fatalf("expected damaging 角念 miss-seed action, got %+v", miss)
	}
	if len(runtime.PendingBuffInfos) != 0 {
		t.Fatalf("expected no 封印 BuffInfo on miss seed, got %+v", runtime.PendingBuffInfos)
	}
	if _, ok := runtime.StatusEffects[missedTarget.Handle]; ok {
		t.Fatalf("expected no status on miss seed, got %+v", runtime.StatusEffects)
	}
}

func TestAllDamageUsesUnifiedHalfDefense(t *testing.T) {
	runtime := &Runtime{EnemyAttackRanges: map[string]battleAttackRange{
		"enemy_wocmon": {Min: 389, Max: 399},
	}}
	enemy := &CellInfoPush{Handle: "enemy_wocmon", Camp: CampEnemy, Attack: 184}
	target := &CellInfoPush{Handle: "player_21499", Camp: CampTeam, MgcDefense: 300}
	profile := commandProfile{SourceActionLabel: "nomalAtk", DamageMultiplier: 1, DefenseType: "magic"}

	defense := runtime.effectiveBattleDefense(enemy, target, false, profile.DefenseType)
	if defense != 150 {
		t.Fatalf("expected half 阿柴 magic defense to mitigate enemy magic damage, got %d", defense)
	}
	restoreMinimumRoll := useSourceBattleAttackRoll(func(int) int { return 0 })
	if damage := runtime.baseBattleDamage(enemy, profile, defense); damage != 239 {
		t.Fatalf("expected a 389 attack to deal 239 against half magic defense 150, got %d", damage)
	}
	restoreMinimumRoll()
	restoreMaximumRoll := useSourceBattleAttackRoll(func(maxExclusive int) int { return maxExclusive - 1 })
	defer restoreMaximumRoll()
	if damage := runtime.baseBattleDamage(enemy, profile, defense); damage != 249 {
		t.Fatalf("expected a 399 attack to deal 249 against half magic defense 150, got %d", damage)
	}

	player := &CellInfoPush{Handle: "player_mage", Camp: CampTeam, Attack: 184}
	if defense := runtime.effectiveBattleDefense(player, target, false, profile.DefenseType); defense != 150 {
		t.Fatalf("expected player magic damage to use half magic defense, got %d", defense)
	}

	physicalEnemy := &CellInfoPush{Handle: "7895833328747103", Camp: CampEnemy, Name: "被控制的民兵", DisplayURL: "monstermap/militia.swf", Attack: 200}
	runtime.EnemyAttackRanges[physicalEnemy.Handle] = battleAttackRange{Min: 209, Max: 245}
	physicalTarget := &CellInfoPush{Handle: "player_21432", Camp: CampTeam, Defense: 300}
	physicalProfile := commandProfile{SourceActionLabel: "nomalAtk", DamageMultiplier: 1, DefenseType: "physical"}
	if defense := runtime.effectiveBattleDefense(physicalEnemy, physicalTarget, false, physicalProfile.DefenseType); defense != 150 {
		t.Fatalf("expected enemy physical damage to use half defense, got %d", defense)
	}
	otherPhysicalEnemy := &CellInfoPush{Handle: "enemy_with_range_but_no_cross_target_evidence", Camp: CampEnemy, Attack: 200}
	runtime.EnemyAttackRanges[otherPhysicalEnemy.Handle] = battleAttackRange{Min: 153, Max: 247}
	if defense := runtime.effectiveBattleDefense(otherPhysicalEnemy, physicalTarget, false, physicalProfile.DefenseType); defense != 150 {
		t.Fatalf("expected every enemy physical attack to use half defense, got %d", defense)
	}
	playerPhysical := &CellInfoPush{Handle: "player_physical", Camp: CampTeam, Attack: 200}
	if defense := runtime.effectiveBattleDefense(playerPhysical, physicalTarget, false, physicalProfile.DefenseType); defense != 150 {
		t.Fatalf("expected player physical damage to use half defense, got %d", defense)
	}
	if defense := runtime.effectiveBattleDefense(enemy, target, false, "direct"); defense != 0 {
		t.Fatalf("expected direct damage to bypass half-defense policy, got %d", defense)
	}
}

func TestMilitiaDamageAgainst777InfluxUsesFractionalHalfDefense(t *testing.T) {
	runtime := &Runtime{
		EnemyAttackRanges: map[string]battleAttackRange{
			"militia-ranger":  {Min: 222, Max: 245},
			"militia-warrior": {Min: 216, Max: 321},
		},
		EnemySweepSpearRanges: map[string]battleAttackRange{
			"militia-ranger":  {Min: 209, Max: 209},
			"militia-warrior": {Min: 207, Max: 314},
		},
	}
	target := &CellInfoPush{Handle: "acct-777-role-001", Camp: CampTeam, Defense: 509}
	ranger := &CellInfoPush{Handle: "militia-ranger", Camp: CampEnemy, Attack: 234}
	warrior := &CellInfoPush{Handle: "militia-warrior", Camp: CampEnemy, Attack: 269}
	normalProfile := commandProfile{SourceActionLabel: "nomalAtk", DamageMultiplier: 1, DefenseType: "physical"}
	sweepProfile := commandProfile{SourceActionLabel: "sweepspear", DamageMultiplier: enemySweepSpearDamageMultiplier, DefenseType: "physical"}
	defense := runtime.effectiveBattleDefenseValue(ranger, target, false, normalProfile.DefenseType)
	if defense != 254.5 {
		t.Fatalf("expected 777 influx defense 509 to retain fractional half-defense 254.5, got %v", defense)
	}

	restoreMinimumRoll := useSourceBattleAttackRoll(func(int) int { return 0 })
	if damage := maxInt(1, int(math.Round(runtime.baseBattleDamageValue(ranger, normalProfile, defense)))); damage != 1 {
		t.Fatalf("expected ranger militia minimum normal damage 1 against influx 777, got %d", damage)
	}
	if damage := maxInt(1, int(math.Round(runtime.baseBattleDamageValue(ranger, sweepProfile, defense)))); damage != 17 {
		t.Fatalf("expected ranger militia sweepspear damage 17 against influx 777, got %d", damage)
	}
	if damage := maxInt(1, int(math.Round(runtime.baseBattleDamageValue(warrior, normalProfile, defense)))); damage != 1 {
		t.Fatalf("expected warrior militia minimum normal damage 1 against influx 777, got %d", damage)
	}
	if damage := maxInt(1, int(math.Round(runtime.baseBattleDamageValue(warrior, sweepProfile, defense)))); damage != 15 {
		t.Fatalf("expected warrior militia minimum sweepspear damage 15 against influx 777, got %d", damage)
	}
	restoreMinimumRoll()

	restoreMaximumRoll := useSourceBattleAttackRoll(func(maxExclusive int) int { return maxExclusive - 1 })
	defer restoreMaximumRoll()
	if damage := maxInt(1, int(math.Round(runtime.baseBattleDamageValue(ranger, normalProfile, defense)))); damage != 1 {
		t.Fatalf("expected ranger militia maximum normal damage 1 against influx 777, got %d", damage)
	}
	if damage := maxInt(1, int(math.Round(runtime.baseBattleDamageValue(warrior, normalProfile, defense)))); damage != 67 {
		t.Fatalf("expected warrior militia maximum normal damage 67 against influx 777, got %d", damage)
	}
	if damage := maxInt(1, int(math.Round(runtime.baseBattleDamageValue(warrior, sweepProfile, defense)))); damage != 154 {
		t.Fatalf("expected warrior militia maximum sweepspear damage 154 against influx 777, got %d", damage)
	}
}

func TestLocalHalfDefensePolicyDoublesDefenseUntilOneLandedHit(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-local-defense-policy",
		nextSequence:     1,
		DefendingHandles: map[string]bool{"player_defending": true},
	}
	enemy := &CellInfoPush{
		Handle:       "enemy_local_policy",
		Camp:         CampEnemy,
		Attack:       300,
		Hit:          100,
		CommandLabel: "普通攻击",
	}
	target := &CellInfoPush{
		Handle:     "player_defending",
		Camp:       CampTeam,
		HP:         1000,
		MaxHP:      1000,
		Defense:    90,
		MgcDefense: 80,
	}

	if defense := runtime.effectiveBattleDefense(enemy, target, true, "physical"); defense != 90 {
		t.Fatalf("expected local Defense to double half physical mitigation to 90, got %d", defense)
	}
	if defense := runtime.effectiveBattleDefense(enemy, target, true, "magic"); defense != 80 {
		t.Fatalf("expected local Defense to double half magic mitigation to 80, got %d", defense)
	}

	action := runtime.resolveAttack(enemy, target, CommandNormalAttack)
	if !action.TargetInDef || action.Damage != 210 || target.HP != 790 {
		t.Fatalf("expected the defended enemy hit to use the local 90 effective defense, got action=%+v target=%+v", action, target)
	}
	if runtime.DefendingHandles[target.Handle] {
		t.Fatalf("expected a landed target outcome to consume the local Defense stance")
	}

	runtime.DefendingHandles[target.Handle] = true
	target.Dog = 1
	enemy.Hit = 0
	defendingHit := runtime.resolveAttack(enemy, target, CommandNormalAttack)
	if defendingHit.TargetActionStateCode != "0" || defendingHit.Damage != 210 || target.HP != 580 || runtime.DefendingHandles[target.Handle] {
		t.Fatalf("expected a defending target to be unable to dodge and consume Defense on hit, got action=%+v defending=%+v", defendingHit, runtime.DefendingHandles)
	}
}

func TestStoredPowerTargetCannotDodge(t *testing.T) {
	runtime := &Runtime{StoredPower: map[string]int{"player_storing": 1}}
	actor := &CellInfoPush{Handle: "enemy", Camp: CampEnemy, Attack: 100, Hit: 0}
	target := &CellInfoPush{Handle: "player_storing", Camp: CampTeam, HP: 500, MaxHP: 500, Dog: 1}

	action := runtime.resolveAttack(actor, target, CommandNormalAttack)

	if action.TargetActionStateCode != "0" || action.Damage != 100 || target.HP != 400 {
		t.Fatalf("expected a storing target to be unable to dodge, got action=%+v target=%+v", action, target)
	}
}

func TestShihukuChilukingUsesCapturedRampageDisplayAction(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-shihuku-chiluking-rampage",
		Round:            3,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
	}
	actor := &CellInfoPush{
		Handle:     "enemy_chiluking",
		Camp:       CampEnemy,
		Name:       "蚩颅王",
		DisplayURL: "monstermap/chiluking.swf",
		HP:         6000,
		MaxHP:      6000,
		MP:         600,
		MaxMP:      600,
	}

	actions := runtime.resolveEnemyRampageActions(actor)

	if len(actions) != 1 || actions[0].ActionName != "暴走之力" || actions[0].SourceActionLabel != "battleStand" || actions[0].TargetHandle != actor.Handle {
		t.Fatalf("expected shihuku chiluking captured 暴走之力 battleStand self action, got %+v", actions)
	}
	if actor.MP != 600 || len(runtime.PendingBuffInfos) != 1 || runtime.PendingBuffInfos[0].Name != "暴走之力" {
		t.Fatalf("expected shihuku chiluking rampage to keep MP and emit buff info, actor=%+v buffs=%+v", actor, runtime.PendingBuffInfos)
	}
}

func TestZanglongtanDragonsonUsesCapturedRampageDisplayAction(t *testing.T) {
	visibleMonster, ok := sourceVisibleMonsterConfigForHandle("186", "9611310624908840")
	if !ok {
		t.Fatal("expected captured map186 龙娃 visible monster config")
	}
	if !sourceEnemyCanRampage(&visibleMonster.Cell) {
		t.Fatalf("expected captured 龙娃 to use shared 暴走之力 contract, cell=%+v", visibleMonster.Cell)
	}
	runtime := &Runtime{
		BattleID:         "battle-zanglongtan-dragonson-rampage",
		Round:            19,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
	}
	actor := visibleMonster.Cell.withBattleIDAndSlot(runtime.BattleID, 0)
	actions := runtime.resolveEnemyRampageActions(&actor)
	if len(actions) != 1 || actions[0].ActionName != "暴走之力" || actions[0].SourceActionLabel != "battleStand" || actions[0].TargetHandle != actor.Handle {
		t.Fatalf("expected captured 龙娃 暴走之力 battleStand self action, got %+v", actions)
	}
	if actor.MP != actor.MaxMP || len(actions[0].RefreshInfos) != 1 || actions[0].RefreshInfos[0].MP != actor.MaxMP {
		t.Fatalf("expected 龙娃 暴走之力 to keep MP unchanged, actor=%+v action=%+v", actor, actions[0])
	}
	if len(runtime.PendingBuffInfos) != 1 || runtime.PendingBuffInfos[0].Name != "暴走之力" || runtime.PendingBuffInfos[0].Display != "1595.png" || runtime.PendingBuffInfos[0].TargetHandle != actor.Handle {
		t.Fatalf("expected 龙娃 captured 暴走之力 BuffInfo, got %+v", runtime.PendingBuffInfos)
	}
}

func findBattleIDForEnemyCommand(expected string, enemy *CellInfoPush, target *CellInfoPush) (string, bool) {
	for index := 0; index < 10000; index++ {
		battleID := fmt.Sprintf("battle-shihuku-ai-%s-%d", expected, index)
		runtime := &Runtime{BattleID: battleID, Round: 1, nextSequence: 1}
		if runtime.enemyBattleCommand(enemy, target) == expected {
			return battleID, true
		}
	}
	return "", false
}

func TestMagicpandaEnemyNormalAttackUsesCapturedBroadcastName(t *testing.T) {
	visibleMonster, ok := sourceVisibleMonsterConfigForHandle("143", "5166206909805441")
	if !ok {
		t.Fatal("expected captured magicpanda visible monster config")
	}
	runtime := &Runtime{
		BattleID:         "battle-magicpanda-normal",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
	}
	actor := visibleMonster.Cell.withBattleIDAndSlot(runtime.BattleID, 0)
	target := &CellInfoPush{
		Handle:     "player_21424",
		Camp:       CampTeam,
		MaxHP:      1045,
		HP:         1045,
		Defense:    158,
		MgcDefense: 30,
	}

	action := runtime.resolveAttack(&actor, target, CommandEnemyAttack)

	if action.ActionName != "法术普通攻击" || action.SourceActionLabel != "nomalAtk" {
		t.Fatalf("expected magicpanda normal attack to use CSV command_label while keeping source animation, config=%+v action=%+v", visibleMonster.Cell, action)
	}
	if visibleMonster.Cell.DamageDefenseType != "magic" || action.Damage != 128 || action.TargetHP != 917 {
		t.Fatalf("expected magicpanda normal attack to use the half magic-defense path without stored power bonus, config=%+v action=%+v target=%+v", visibleMonster.Cell, action, target)
	}
}

func TestMap143SwordpandaNormalAttackUsesCapturedDirectDamage(t *testing.T) {
	visibleMonster, ok := sourceVisibleMonsterConfigForHandle("143", "5172206909807859")
	if !ok {
		t.Fatal("expected captured map143 swordpanda visible monster config")
	}
	runtime := &Runtime{
		BattleID:         "battle-swordpanda-direct",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
	}
	actor := visibleMonster.Cell.withBattleIDAndSlot(runtime.BattleID, 0)
	target := &CellInfoPush{
		Handle:  "player_21424",
		Camp:    CampTeam,
		MaxHP:   1045,
		HP:      1045,
		Defense: 158,
	}

	action := runtime.resolveAttack(&actor, target, CommandEnemyAttack)

	if visibleMonster.Cell.DamageDefenseType != "direct" || action.Damage != 121 || action.TargetHP != 924 {
		t.Fatalf("expected map143 swordpanda captured HP delta without stored power bonus or second defense subtraction, config=%+v action=%+v target=%+v", visibleMonster.Cell, action, target)
	}
}

func TestEnemyTurnSelectsAmongLivingTeamTargets(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-enemy-target-roll",
		Round:            1,
		Phase:            PhaseCommand,
		ActiveHandle:     "player_leader",
		nextSequence:     1,
		ConsumedSequence: map[int]bool{},
		DefendingHandles: map[string]bool{},
		StoredPower:      map[string]int{},
		Cells: []CellInfoPush{
			{
				BattleID: "battle-enemy-target-roll",
				Handle:   "player_leader",
				Camp:     CampTeam,
				Name:     "队长",
				HP:       1000,
				MaxHP:    1000,
				Attack:   1,
			},
			{
				BattleID: "battle-enemy-target-roll",
				Handle:   "player_member",
				Camp:     CampTeam,
				Name:     "队员",
				HP:       1000,
				MaxHP:    1000,
				Attack:   1,
			},
			{
				BattleID:   "battle-enemy-target-roll",
				Handle:     "enemy_blackshadow",
				Camp:       CampEnemy,
				Name:       "黑影",
				DisplayURL: "monstermap/blackshadow.swf",
				HP:         2300,
				MaxHP:      2300,
				Attack:     50,
			},
		},
	}

	found := map[string]bool{}
	for battleIndex := 0; battleIndex < 40; battleIndex++ {
		candidate := *runtime
		candidate.BattleID = fmt.Sprintf("battle-enemy-target-roll-%d", battleIndex)
		candidate.Cells = append([]CellInfoPush(nil), runtime.Cells...)
		for index := range candidate.Cells {
			candidate.Cells[index].BattleID = candidate.BattleID
		}
		candidate.ConsumedSequence = map[int]bool{}
		candidate.DefendingHandles = map[string]bool{}
		candidate.StoredPower = map[string]int{}
		enemy := candidate.cellByHandle("enemy_blackshadow")
		target := candidate.resolveEnemyTeamTarget(enemy)
		if target == nil || target.Camp != CampTeam {
			t.Fatalf("expected living team target, got %+v", target)
		}
		found[target.Handle] = true
		if len(found) == 2 {
			break
		}
	}
	if !found["player_leader"] || !found["player_member"] {
		t.Fatalf("expected enemy team target roll to cover both living team members, got %+v", found)
	}

	// Full enemy-turn path must not hard-lock to firstLiving(CampTeam).
	resultFound := map[string]bool{}
	for battleIndex := 0; battleIndex < 40; battleIndex++ {
		candidate := &Runtime{
			BattleID:         fmt.Sprintf("battle-enemy-target-turn-%d", battleIndex),
			Round:            1,
			Phase:            PhaseCommand,
			ActiveHandle:     "player_leader",
			nextSequence:     1,
			ConsumedSequence: map[int]bool{},
			DefendingHandles: map[string]bool{},
			StoredPower:      map[string]int{},
			Cells: []CellInfoPush{
				{
					Handle: "player_leader",
					Camp:   CampTeam,
					Name:   "队长",
					HP:     1000,
					MaxHP:  1000,
					Attack: 1,
				},
				{
					Handle: "player_member",
					Camp:   CampTeam,
					Name:   "队员",
					HP:     1000,
					MaxHP:  1000,
					Attack: 1,
				},
				{
					Handle:     "enemy_blackshadow",
					Camp:       CampEnemy,
					Name:       "黑影",
					DisplayURL: "monstermap/blackshadow.swf",
					HP:         2300,
					MaxHP:      2300,
					Attack:     50,
				},
			},
		}
		result := candidate.ProcessAction(ActionRequest{
			BattleID:     candidate.BattleID,
			ActorHandle:  "player_leader",
			CommandID:    CommandNormalAttack,
			TargetHandle: "enemy_blackshadow",
			Round:        1,
			Sequence:     1,
		})
		if result.ErrorCode != "" || len(result.Actions) < 2 {
			t.Fatalf("expected player action followed by enemy action, got %+v", result)
		}
		enemyAction := result.Actions[len(result.Actions)-1]
		if enemyAction.ActorHandle != "enemy_blackshadow" {
			t.Fatalf("expected last action to be enemy turn, got %+v", enemyAction)
		}
		if enemyAction.TargetHandle == "all" {
			// AOE path is not under test here.
			continue
		}
		resultFound[enemyAction.TargetHandle] = true
		if len(resultFound) == 2 {
			break
		}
	}
	if !resultFound["player_leader"] || !resultFound["player_member"] {
		t.Fatalf("expected enemy turn actions to hit both living team members over rolls, got %+v", resultFound)
	}
}

func TestRobberEnemyTurnCanUseCapturedSlideCut(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-slide-2",
		Round:            1,
		Phase:            PhaseCommand,
		ActiveHandle:     "player_21424",
		nextSequence:     1,
		ConsumedSequence: map[int]bool{},
		DefendingHandles: map[string]bool{},
		StoredPower:      map[string]int{},
		Cells: []CellInfoPush{
			{
				BattleID: "battle-slide-2",
				Handle:   "player_21424",
				Camp:     CampTeam,
				HP:       725,
				MaxHP:    725,
				Attack:   1,
			},
			{
				BattleID:   "battle-slide-2",
				Handle:     "enemy_robber",
				Camp:       CampEnemy,
				Name:       "盗贼",
				DisplayURL: "monstermap/robber.swf",
				HP:         445,
				MaxHP:      445,
				MP:         194,
				MaxMP:      194,
				Attack:     47,
			},
		},
	}

	result := runtime.ProcessAction(ActionRequest{
		BattleID:     "battle-slide-2",
		ActorHandle:  "player_21424",
		CommandID:    CommandNormalAttack,
		TargetHandle: "enemy_robber",
		Round:        1,
		Sequence:     1,
	})

	if result.ErrorCode != "" || len(result.Actions) < 2 {
		t.Fatalf("expected player action followed by enemy action, got %+v", result)
	}
	enemyAction := result.Actions[1]
	if enemyAction.CommandID != CommandEnemySlideCut || enemyAction.ActionName != "滑行斩" || enemyAction.SourceActionLabel != "slideCut" {
		t.Fatalf("expected robber enemy turn to use captured slideCut, got %+v", enemyAction)
	}
	if runtime.Cells[1].MP != 184 || enemyAction.RefreshInfos[0].Handle != "enemy_robber" || enemyAction.RefreshInfos[0].MP != 184 {
		t.Fatalf("expected enemy slideCut turn to consume MP 10, got cells=%+v action=%+v", runtime.Cells, enemyAction)
	}
}

func TestBigSwordWolfEnemyTurnCanUseCapturedShadeCut(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-shade-2",
		Round:            1,
		Phase:            PhaseCommand,
		ActiveHandle:     "player_21424",
		nextSequence:     1,
		ConsumedSequence: map[int]bool{},
		DefendingHandles: map[string]bool{},
		StoredPower:      map[string]int{},
		Cells: []CellInfoPush{
			{
				BattleID: "battle-shade-2",
				Handle:   "player_21424",
				Camp:     CampTeam,
				HP:       725,
				MaxHP:    725,
				Attack:   1,
			},
			{
				BattleID:   "battle-shade-2",
				Handle:     "enemy_wolf",
				Camp:       CampEnemy,
				Name:       "单刀狼人",
				DisplayURL: "monstermap/bigswordwolf.swf",
				HP:         2500,
				MaxHP:      2500,
				MP:         334,
				MaxMP:      334,
				Attack:     260,
			},
		},
	}

	result := runtime.ProcessAction(ActionRequest{
		BattleID:     "battle-shade-2",
		ActorHandle:  "player_21424",
		CommandID:    CommandNormalAttack,
		TargetHandle: "enemy_wolf",
		Round:        1,
		Sequence:     1,
	})

	if result.ErrorCode != "" || len(result.Actions) < 2 {
		t.Fatalf("expected player action followed by enemy action, got %+v", result)
	}
	enemyAction := result.Actions[1]
	if enemyAction.CommandID != CommandEnemyShadeCut || enemyAction.ActionName != "影刃" || enemyAction.SourceActionLabel != "shadeCut" {
		t.Fatalf("expected wolf enemy turn to use captured shadeCut, got %+v", enemyAction)
	}
	if runtime.Cells[1].MP != 294 || enemyAction.RefreshInfos[0].Handle != "enemy_wolf" || enemyAction.RefreshInfos[0].MP != 294 {
		t.Fatalf("expected enemy shadeCut turn to consume MP 40, got cells=%+v action=%+v", runtime.Cells, enemyAction)
	}
}

func TestCracktoadEnemyTurnCanUseCapturedHelixAtk(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-helix-roll-4",
		Round:            1,
		Phase:            PhaseCommand,
		ActiveHandle:     "player_21424",
		nextSequence:     1,
		ConsumedSequence: map[int]bool{},
		DefendingHandles: map[string]bool{},
		StoredPower:      map[string]int{},
		Cells: []CellInfoPush{
			{
				BattleID: "battle-helix-2",
				Handle:   "player_21424",
				Camp:     CampTeam,
				HP:       1045,
				MaxHP:    1045,
				Attack:   1,
				Defense:  158,
			},
			{
				BattleID:   "battle-helix-roll-4",
				Handle:     "5176206909809579",
				Camp:       CampEnemy,
				Name:       "蛤蟆精",
				DisplayURL: "monstermap/cracktoad.swf",
				HP:         1500,
				MaxHP:      1500,
				MP:         564,
				MaxMP:      564,
				Attack:     240,
			},
		},
	}

	result := runtime.ProcessAction(ActionRequest{
		BattleID:     "battle-helix-roll-4",
		ActorHandle:  "player_21424",
		CommandID:    CommandNormalAttack,
		TargetHandle: "5176206909809579",
		Round:        1,
		Sequence:     1,
	})

	if result.ErrorCode != "" || len(result.Actions) < 2 {
		t.Fatalf("expected player action followed by enemy action, got %+v", result)
	}
	enemyAction := result.Actions[1]
	if enemyAction.CommandID != CommandEnemyHelixAtk || enemyAction.ActionName != "螺旋锤杀" || enemyAction.SourceActionLabel != "helixAtk" {
		t.Fatalf("expected cracktoad enemy turn to use captured helixAtk, got %+v", enemyAction)
	}
	if enemyAction.Damage != 238 || enemyAction.TargetHP != 807 {
		t.Fatalf("expected cracktoad enemy turn to apply half physical defense with helixAtk multiplier, got %+v", enemyAction)
	}
	if runtime.Cells[1].MP != 554 || enemyAction.RefreshInfos[0].Handle != "5176206909809579" || enemyAction.RefreshInfos[0].MP != 554 {
		t.Fatalf("expected enemy helixAtk turn to consume MP 10, got cells=%+v action=%+v", runtime.Cells, enemyAction)
	}
}

func TestCracktoadEnemyTurnCanUseCapturedNormalAttack(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-helix-roll-0",
		Round:            1,
		Phase:            PhaseCommand,
		ActiveHandle:     "player_21424",
		nextSequence:     1,
		ConsumedSequence: map[int]bool{},
		DefendingHandles: map[string]bool{},
		StoredPower:      map[string]int{},
		Cells: []CellInfoPush{
			{
				BattleID: "battle-helix-roll-0",
				Handle:   "player_21424",
				Camp:     CampTeam,
				HP:       1045,
				MaxHP:    1045,
				Attack:   1,
				Defense:  158,
			},
			{
				BattleID:     "battle-helix-roll-0",
				Handle:       "5176206909809579",
				Camp:         CampEnemy,
				Name:         "蛤蟆精",
				DisplayURL:   "monstermap/cracktoad.swf",
				HP:           1500,
				MaxHP:        1500,
				MP:           564,
				MaxMP:        564,
				Attack:       240,
				CommandLabel: "普通攻击",
			},
		},
	}

	result := runtime.ProcessAction(ActionRequest{
		BattleID:     "battle-helix-roll-0",
		ActorHandle:  "player_21424",
		CommandID:    CommandNormalAttack,
		TargetHandle: "5176206909809579",
		Round:        1,
		Sequence:     1,
	})

	if result.ErrorCode != "" || len(result.Actions) < 2 {
		t.Fatalf("expected player action followed by enemy action, got %+v", result)
	}
	enemyAction := result.Actions[1]
	if enemyAction.CommandID != CommandEnemyAttack || enemyAction.ActionName != "普通攻击" || enemyAction.SourceActionLabel != "nomalAtk" {
		t.Fatalf("expected cracktoad enemy turn to sometimes use captured normal attack, got %+v", enemyAction)
	}
	if enemyAction.Damage != 161 || enemyAction.TargetHP != 884 {
		t.Fatalf("expected cracktoad normal attack to apply half physical defense, got %+v", enemyAction)
	}
	if runtime.Cells[1].MP != 564 || len(enemyAction.RefreshInfos) != 1 {
		t.Fatalf("expected cracktoad normal attack to keep MP unchanged, got cells=%+v action=%+v", runtime.Cells, enemyAction)
	}
}

func TestCapturedSingleSwordSkillProfilesAndCommands(t *testing.T) {
	cases := []struct {
		name          string
		level         int
		sourceType    string
		label         string
		mpCost        int
		multiplier    float64
		target        string
		armorBreakPct int
		stunChance    int
		hitMultiplier float64
	}{
		{name: "挑衅", level: 1, sourceType: "all", label: "com/tx", mpCost: 20, multiplier: 0, target: "self"},
		{name: "卷叶式", level: 1, sourceType: "oneE", label: "w7/jys2", mpCost: 8, multiplier: 1.55, target: "enemy"},
		{name: "卷叶式", level: 5, sourceType: "oneE", label: "w7/jys2", mpCost: 16, multiplier: 1.75, target: "enemy"},
		{name: "强贯式", level: 1, sourceType: "oneE", label: "w7/qgs1", mpCost: 12, multiplier: 1.51, target: "enemy", armorBreakPct: 30},
		{name: "强贯式", level: 3, sourceType: "oneE", label: "w7/qgs1", mpCost: 15, multiplier: 1.53, target: "enemy", armorBreakPct: 40},
		{name: "强贯式", level: 5, sourceType: "oneE", label: "w7/qgs1", mpCost: 20, multiplier: 1.55, target: "enemy", armorBreakPct: 50},
		{name: "凝神式", level: 1, sourceType: "own", label: "w7/nss", mpCost: 12, multiplier: 0, target: "self"},
		{name: "凝神式", level: 5, sourceType: "own", label: "w7/nss", mpCost: 22, multiplier: 0, target: "self"},
		{name: "狂舞式", level: 1, sourceType: "oneE", label: "w7/kws", mpCost: 22, multiplier: 0.8, target: "enemy", stunChance: 21},
		{name: "狂舞式", level: 5, sourceType: "oneE", label: "w7/kws", mpCost: 30, multiplier: 1, target: "enemy", stunChance: 25},
		{name: "气愈式", level: 1, sourceType: "own", label: "w7/qys", mpCost: 15, multiplier: 0, target: "self"},
		{name: "气愈式", level: 5, sourceType: "own", label: "w7/qys", mpCost: 31, multiplier: 0, target: "self"},
		{name: "奥义.飘血", level: 1, sourceType: "oneE", label: "w7/aypx", mpCost: 26, multiplier: 3, target: "enemy", hitMultiplier: 1.70},
		{name: "奥义.飘血", level: 4, sourceType: "oneE", label: "w7/aypx", mpCost: 38, multiplier: 3.3, target: "enemy", hitMultiplier: 1.85},
	}
	commandCases := []struct {
		name       string
		level      int
		sourceType string
		label      string
		mpCost     int
		multiplier float64
		target     string
	}{
		{name: "挑衅", level: 1, sourceType: "all", label: "com/tx", mpCost: 20, multiplier: 0, target: "self"},
		{name: "卷叶式", level: 5, sourceType: "oneE", label: "w7/jys2", mpCost: 16, multiplier: 1.75, target: "enemy"},
		{name: "强贯式", level: 5, sourceType: "oneE", label: "w7/qgs1", mpCost: 20, multiplier: 1.55, target: "enemy"},
		{name: "凝神式", level: 5, sourceType: "own", label: "w7/nss", mpCost: 22, multiplier: 0, target: "self"},
		{name: "狂舞式", level: 5, sourceType: "oneE", label: "w7/kws", mpCost: 30, multiplier: 1, target: "enemy"},
		{name: "气愈式", level: 5, sourceType: "own", label: "w7/qys", mpCost: 31, multiplier: 0, target: "self"},
		{name: "奥义.飘血", level: 4, sourceType: "oneE", label: "w7/aypx", mpCost: 38, multiplier: 3.3, target: "enemy"},
	}
	for _, testCase := range cases {
		skill := session.RoleSkill{Name: testCase.name, Level: testCase.level, Type: testCase.sourceType}
		profile := sourceBattleSkillProfile(skill)
		if profile.SourceType != testCase.sourceType || profile.SourceActionLabel != testCase.label || profile.MPCost != testCase.mpCost || profile.DamageMultiplier != testCase.multiplier {
			t.Fatalf("expected captured %s Lv%d profile, got %+v", testCase.name, testCase.level, profile)
		}
		if testCase.armorBreakPct > 0 && (profile.StatusName != "卸甲" || profile.StatusDefensePercent != testCase.armorBreakPct || profile.StatusRounds != qiangGuanShiArmorBreakRounds || profile.StatusChance != 100) {
			t.Fatalf("expected captured %s Lv%d armor break %d%%, got %+v", testCase.name, testCase.level, testCase.armorBreakPct, profile)
		}
		if testCase.stunChance > 0 && (profile.StatusName != "眩晕" || profile.StatusChance != testCase.stunChance || profile.StatusRounds != kuangWuShiStunRounds || !profile.SkipTurn) {
			t.Fatalf("expected captured %s Lv%d stun chance %d, got %+v", testCase.name, testCase.level, testCase.stunChance, profile)
		}
		if testCase.hitMultiplier > 0 && profile.HitMultiplier != testCase.hitMultiplier {
			t.Fatalf("expected captured %s Lv%d hit multiplier %.2f, got %+v", testCase.name, testCase.level, testCase.hitMultiplier, profile)
		}
	}
	skills := make([]session.RoleSkill, 0, len(commandCases))
	for _, testCase := range commandCases {
		skills = append(skills, session.RoleSkill{Name: testCase.name, Level: testCase.level, Type: testCase.sourceType})
	}
	commands := sourceBattleCommandDefinitions(skills)
	byLabel := map[string]CommandDefinition{}
	for _, command := range commands {
		byLabel[command.Label] = command
	}
	for _, testCase := range commandCases {
		command, ok := byLabel[testCase.name]
		if !ok || command.SourceType != testCase.sourceType || command.SourceActionLabel != testCase.label || command.MPCost != testCase.mpCost || command.DamageMultiplier != testCase.multiplier || command.Target != testCase.target {
			t.Fatalf("expected captured %s command definition, got %+v", testCase.name, command)
		}
	}
}

func TestCapturedSingleSwordLowerLevelStatusAndHeal(t *testing.T) {
	if got := sourceQiangGuanShiArmorBreakPercent(1); got != 30 {
		t.Fatalf("expected 强贯式 Lv1 armor break 30, got %d", got)
	}
	if got := sourceKuangWuShiStunChance(1); got != 21 {
		t.Fatalf("expected 狂舞式 Lv1 stun chance 21, got %d", got)
	}
	if got := sourceNingShenHitPercent(1); got != 50 {
		t.Fatalf("expected 凝神式 Lv1 hit percent 50, got %d", got)
	}
	if got := sourceQiYuHealPercent(1); got != 9 {
		t.Fatalf("expected 气愈式 Lv1 heal percent 9, got %d", got)
	}
	if got := sourceAoYiPiaoXueHitMultiplier(1); got != 1.70 {
		t.Fatalf("expected 奥义.飘血 Lv1 hit multiplier 1.70, got %v", got)
	}

	strongRuntime := &Runtime{
		BattleID:         "battle-single-sword-strong-pierce-lv1",
		StatusEffects:    map[string]BattleStatusEffects{},
		DefendingHandles: map[string]bool{},
		RoleSkills:       []session.RoleSkill{{Name: "强贯式", Level: 1, Type: "oneE"}},
		Cells:            []CellInfoPush{{Handle: "player_sword", Camp: CampTeam, HP: 1000, MaxHP: 1000, MP: 100, Attack: 100}, {Handle: "enemy_1", Camp: CampEnemy, HP: 1000, MaxHP: 1000, Defense: 100}},
	}
	strongAction := strongRuntime.resolveAttack(strongRuntime.cellByHandle("player_sword"), strongRuntime.cellByHandle("enemy_1"), CommandQiangGuanShi)
	strongBuffs := strongRuntime.consumePendingBuffInfos()
	if strongAction.SourceActionLabel != "w7/qgs1" || len(strongBuffs) != 1 || strongBuffs[0].Name != "卸甲" || strongRuntime.cellByHandle("enemy_1").Defense != 70 {
		t.Fatalf("expected 强贯式 Lv1 30%% armor break, action=%+v buffs=%+v enemy=%+v", strongAction, strongBuffs, strongRuntime.cellByHandle("enemy_1"))
	}

	ningRuntime := &Runtime{
		BattleID:      "battle-single-sword-ning-lv1",
		StatusEffects: map[string]BattleStatusEffects{},
		RoleSkills:    []session.RoleSkill{{Name: "凝神式", Level: 1, Type: "own"}},
		Cells:         []CellInfoPush{{Handle: "player_sword", Camp: CampTeam, HP: 1000, MaxHP: 1000, Hit: 400, Fat: 200}},
	}
	actor := ningRuntime.cellByHandle("player_sword")
	buffInfos := ningRuntime.applyNingShenStatusEffects(actor)
	// 50% of 400 hit = 200
	if actor.Hit != 600 || actor.Fat != 400 || len(buffInfos) != 2 || buffInfos[0].Description != "提高对象200命中" {
		t.Fatalf("expected 凝神式 Lv1 hit boost, actor=%+v buffs=%+v", actor, buffInfos)
	}

	healRuntime := &Runtime{
		BattleID:      "battle-single-sword-heal-lv1",
		StatusEffects: map[string]BattleStatusEffects{},
		RoleSkills:    []session.RoleSkill{{Name: "气愈式", Level: 1, Type: "own"}},
		Cells:         []CellInfoPush{{Handle: "player_sword", Camp: CampTeam, HP: 500, MaxHP: 2100}},
	}
	healer := healRuntime.cellByHandle("player_sword")
	qiLiao := healRuntime.applyQiYuStatusEffect(healer)
	statusActions, _ := healRuntime.resolveStatusStartActions(healer)
	// 9% of 2100 = 189
	if qiLiao.Name != "气疗" || qiLiao.Description != "每回合对象恢复189气力" || healer.HP != 689 || len(statusActions) != 1 || statusActions[0].ActionName != "气疗" {
		t.Fatalf("expected 气愈式 Lv1 heal, buff=%+v actor=%+v actions=%+v", qiLiao, healer, statusActions)
	}
}

func TestCapturedSingleSwordStrongPierceAndWildDanceStatusEffects(t *testing.T) {
	strongRuntime := &Runtime{
		BattleID:         "battle-single-sword-strong-pierce",
		StatusEffects:    map[string]BattleStatusEffects{},
		DefendingHandles: map[string]bool{},
		RoleSkills:       []session.RoleSkill{{Name: "强贯式", Level: 5, Type: "oneE"}},
		Cells:            []CellInfoPush{{Handle: "player_sword", Camp: CampTeam, HP: 1000, MaxHP: 1000, MP: 100, Attack: 100}, {Handle: "enemy_1", Camp: CampEnemy, HP: 1000, MaxHP: 1000, Defense: 90}},
	}
	if !strongRuntime.isBattleCommandAllowedForActor("player_sword", CommandQiangGuanShi) {
		t.Fatal("expected captured 强贯式 command to be allowed for the learned role")
	}
	strongAction := strongRuntime.resolveAttack(strongRuntime.cellByHandle("player_sword"), strongRuntime.cellByHandle("enemy_1"), CommandQiangGuanShi)
	strongBuffs := strongRuntime.consumePendingBuffInfos()
	if strongAction.SourceActionLabel != "w7/qgs1" || len(strongBuffs) != 1 || strongBuffs[0].Name != "卸甲" || strongBuffs[0].Display != "10.png" || strongBuffs[0].Description != "降低对象45点物理防御力" || strongBuffs[0].Round != 3 || strongRuntime.cellByHandle("enemy_1").Defense != 45 {
		t.Fatalf("expected captured 强贯式 armor break, action=%+v buffs=%+v enemy=%+v", strongAction, strongBuffs, strongRuntime.cellByHandle("enemy_1"))
	}
	for index := 0; index < qiangGuanShiArmorBreakRounds; index += 1 {
		strongRuntime.resolveStatusStartActions(strongRuntime.cellByHandle("enemy_1"))
	}
	if strongRuntime.cellByHandle("enemy_1").Defense != 90 {
		t.Fatalf("expected 强贯式 armor break to restore defense on expiry, enemy=%+v", strongRuntime.cellByHandle("enemy_1"))
	}

	var wildDanceRuntime *Runtime
	for index := 0; index < 300; index += 1 {
		candidate := &Runtime{
			BattleID:         fmt.Sprintf("battle-single-sword-wild-dance-%d", index),
			StatusEffects:    map[string]BattleStatusEffects{},
			DefendingHandles: map[string]bool{},
			RoleSkills:       []session.RoleSkill{{Name: "狂舞式", Level: 5, Type: "oneE"}},
			Cells:            []CellInfoPush{{Handle: "player_sword", Camp: CampTeam, HP: 1000, MaxHP: 1000, MP: 100, Attack: 100}, {Handle: "enemy_1", Camp: CampEnemy, HP: 1000, MaxHP: 1000}},
		}
		if candidate.hashBattleRollWithSalt(candidate.cellByHandle("player_sword"), candidate.cellByHandle("enemy_1"), CommandKuangWuShi, "status:眩晕") < kuangWuShiStunChance {
			wildDanceRuntime = candidate
			break
		}
	}
	if wildDanceRuntime == nil {
		t.Fatal("expected deterministic 狂舞式 stun roll")
	}
	wildDanceAction := wildDanceRuntime.resolveAttack(wildDanceRuntime.cellByHandle("player_sword"), wildDanceRuntime.cellByHandle("enemy_1"), CommandKuangWuShi)
	wildDanceBuffs := wildDanceRuntime.consumePendingBuffInfos()
	if wildDanceAction.SourceActionLabel != "w7/kws" || len(wildDanceBuffs) != 1 || wildDanceBuffs[0].Name != "眩晕" || wildDanceBuffs[0].Display != "9.png" || wildDanceBuffs[0].Round != 2 {
		t.Fatalf("expected captured 狂舞式 stun BuffInfo, action=%+v buffs=%+v", wildDanceAction, wildDanceBuffs)
	}
	statusActions, skipped := wildDanceRuntime.resolveStatusStartActions(wildDanceRuntime.cellByHandle("enemy_1"))
	if !skipped || len(statusActions) != 1 || statusActions[0].ActionName != "眩晕" || statusActions[0].SourceActionLabel != "yun" {
		t.Fatalf("expected 狂舞式 stun to use captured yun skip action, actions=%+v skipped=%t", statusActions, skipped)
	}
}

func TestCapturedTauntCastUsesAllTargetAndApproachSourceMode(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-single-sword-taunt-cast",
		Round:            1,
		Phase:            PhaseCommand,
		ActiveHandle:     "player_sword",
		nextSequence:     1,
		ConsumedSequence: map[int]bool{},
		DefendingHandles: map[string]bool{},
		StatusEffects:    map[string]BattleStatusEffects{},
		RoleSkills:       []session.RoleSkill{{Name: "挑衅", Level: 1, Type: "all"}},
		Cells: []CellInfoPush{
			{BattleID: "battle-single-sword-taunt-cast", Handle: "player_sword", Camp: CampTeam, HP: 1000, MaxHP: 1000, MP: 100, MaxMP: 100},
			{BattleID: "battle-single-sword-taunt-cast", Handle: "enemy_1", Camp: CampEnemy, HP: 1000, MaxHP: 1000},
		},
	}
	result := runtime.ProcessAction(ActionRequest{
		BattleID:     runtime.BattleID,
		ActorHandle:  "player_sword",
		CommandID:    CommandTaunt,
		TargetHandle: "player_sword",
		Round:        1,
		Sequence:     1,
	})
	if result.ErrorCode != "" || len(result.Actions) == 0 {
		t.Fatalf("expected captured 挑衅 cast action, got %+v", result)
	}
	action := result.Actions[0]
	if action.ActionName != "挑衅" || action.SourceActionLabel != "com/tx" || action.TargetHandle != "all" || action.SourceMode != "1" || action.Damage != 0 {
		t.Fatalf("expected captured 挑衅 action shape all/1/com/tx, got %+v", action)
	}
	if len(result.BuffInfos) != 1 || result.BuffInfos[0].Name != "挑衅" || result.BuffInfos[0].TargetHandle != "player_sword" {
		t.Fatalf("expected self 挑衅 BuffInfo alongside cast action, got %+v", result.BuffInfos)
	}
	if runtime.cellByHandle("player_sword").MP != 80 {
		t.Fatalf("expected 挑衅 MP cost 20, actor=%+v", runtime.cellByHandle("player_sword"))
	}
}

func TestCapturedSingleSwordStatusEffectsUseCapturedValues(t *testing.T) {
	runtime := &Runtime{
		BattleID:      "battle-single-sword-status",
		StatusEffects: map[string]BattleStatusEffects{},
		Cells: []CellInfoPush{
			{BattleID: "battle-single-sword-status", Handle: "player_sword", Camp: CampTeam, HP: 1000, MaxHP: 2100, MP: 100, MaxMP: 614, Hit: 414, Fat: 342},
			{BattleID: "battle-single-sword-status", Handle: "player_ally", Camp: CampTeam, HP: 1000, MaxHP: 1000},
			{BattleID: "battle-single-sword-status", Handle: "enemy_1", Camp: CampEnemy, HP: 1000, MaxHP: 1000},
			{BattleID: "battle-single-sword-status", Handle: "enemy_2", Camp: CampEnemy, HP: 1000, MaxHP: 1000},
			{BattleID: "battle-single-sword-status", Handle: "enemy_3", Camp: CampEnemy, HP: 1000, MaxHP: 1000},
		},
	}
	actor := runtime.cellByHandle("player_sword")
	taunt := runtime.applyTauntStatusEffect(actor)
	if taunt.Name != "挑衅" || taunt.Display != "27.png" || taunt.Description != "敌人每次必攻击该对象" || taunt.Round != 3 {
		t.Fatalf("expected captured taunt buff, got %+v", taunt)
	}
	for _, enemyHandle := range []string{"enemy_1", "enemy_2", "enemy_3"} {
		target := runtime.resolveEnemyTeamTarget(runtime.cellByHandle(enemyHandle))
		if target == nil || target.Handle != actor.Handle {
			t.Fatalf("expected %s to target taunting player, got %+v", enemyHandle, target)
		}
	}
	statusActions, _ := runtime.resolveStatusStartActions(actor)
	if len(statusActions) != 1 || statusActions[0].ActionName != "挑衅" || statusActions[0].SourceActionLabel != "battleStand" {
		t.Fatalf("expected captured taunt countdown action, got %+v", statusActions)
	}

	buffInfos := runtime.applyNingShenStatusEffects(actor)
	if actor.Hit != 704 || actor.Fat != 684 || len(buffInfos) != 2 || buffInfos[0].Name != "集中" || buffInfos[0].Description != "提高对象290命中" || buffInfos[1].Name != "爆击提升" || buffInfos[1].Description != "提高对象342爆击" {
		t.Fatalf("expected captured concentration and critical boosts, actor=%+v buffs=%+v", actor, buffInfos)
	}
	for index := 0; index < ningShenRounds; index += 1 {
		runtime.resolveStatusStartActions(actor)
	}
	if actor.Hit != 414 || actor.Fat != 342 {
		t.Fatalf("expected 凝神式 values to restore after expiry, actor=%+v", actor)
	}

	healRuntime := &Runtime{
		BattleID:      "battle-single-sword-heal",
		StatusEffects: map[string]BattleStatusEffects{},
		Cells:         []CellInfoPush{{BattleID: "battle-single-sword-heal", Handle: "player_sword", Camp: CampTeam, HP: 500, MaxHP: 2100}},
	}
	healer := healRuntime.cellByHandle("player_sword")
	qiLiao := healRuntime.applyQiYuStatusEffect(healer)
	statusActions, _ = healRuntime.resolveStatusStartActions(healer)
	if qiLiao.Name != "气疗" || qiLiao.Display != "21.png" || qiLiao.Description != "每回合对象恢复273气力" || qiLiao.Round != 3 || healer.HP != 773 || len(statusActions) != 1 || statusActions[0].ActionName != "气疗" || statusActions[0].SourceActionLabel != "battleStand" {
		t.Fatalf("expected captured qiyu recovery status, buff=%+v actor=%+v actions=%+v", qiLiao, healer, statusActions)
	}
}

func TestAoYiPiaoXueRequiresCapturedSoulPowerAndUsesCapturedAction(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-piao-xue",
		Round:            1,
		Phase:            PhaseCommand,
		ActiveHandle:     "player_sword",
		nextSequence:     1,
		ConsumedSequence: map[int]bool{},
		DefendingHandles: map[string]bool{},
		StoredPower:      map[string]int{},
		RoleSkills:       []session.RoleSkill{{Name: "奥义.飘血", Level: 4, Type: "oneE"}},
		Cells: []CellInfoPush{
			{BattleID: "battle-piao-xue", Handle: "player_sword", Camp: CampTeam, HP: 500, MaxHP: 500, MP: 100, MaxMP: 100, Attack: 100, Hit: 100},
			{BattleID: "battle-piao-xue", Handle: "enemy_1", Camp: CampEnemy, HP: 1000, MaxHP: 1000},
		},
	}
	request := ActionRequest{BattleID: runtime.BattleID, ActorHandle: "player_sword", CommandID: CommandAoYiPiaoXue, TargetHandle: "enemy_1", Round: 1, Sequence: 1}
	if result := runtime.ProcessAction(request); result.ErrorCode != "insufficient_power" {
		t.Fatalf("expected 奥义.飘血 to require three stored power, got %+v", result)
	}
	runtime.StoredPower["player_sword"] = aoYiPiaoXueRequiredPower
	result := runtime.ProcessAction(request)
	if result.ErrorCode != "" || len(result.Actions) == 0 {
		t.Fatalf("expected 奥义.飘血 captured action, got %+v", result)
	}
	action := result.Actions[0]
	actor := runtime.cellByHandle("player_sword")
	if action.SourceActionLabel != "w7/aypx" || action.Damage != 330 || action.TargetHP != 670 || actor.MP != 62 {
		t.Fatalf("expected captured 奥义.飘血 action and MP cost, action=%+v actor=%+v", action, actor)
	}
}

func TestWoodcutterFistSkillProfilesUseCapturedRows(t *testing.T) {
	cases := []struct {
		name        string
		level       int
		commandID   string
		sourceType  string
		actionLabel string
		mpCost      int
		multiplier  float64
		directBonus float64
		target      string
	}{
		{name: "连击", level: 5, commandID: CommandFistDoubleAtk, sourceType: "oneE", actionLabel: "w5/doubleAtk", mpCost: 18, multiplier: 1.45, target: "enemy"},
		{name: "重烈", level: 5, commandID: CommandFistPowHit, sourceType: "oneE", actionLabel: "w5/powHit", mpCost: 28, multiplier: 1, target: "enemy"},
		{name: "气运丹田", level: 5, commandID: CommandFistInfluxGas, sourceType: "own", actionLabel: "w5/influxGas", mpCost: 32, multiplier: 0, target: "self"},
		{name: "破魂打", level: 5, commandID: CommandFistBreakSoul, sourceType: "oneE", actionLabel: "w5/breakSoul", mpCost: 36, multiplier: 1.5, directBonus: 0.3, target: "enemy"},
		{name: "移形换影", level: 4, commandID: CommandFistMoveShadow, sourceType: "own", actionLabel: "w5/moveShadow", mpCost: 36, multiplier: 0, target: "self"},
		{name: "奥义.修罗幻翼拳", level: 4, commandID: CommandFistPowerAxeWing, sourceType: "oneE", actionLabel: "w5/PowerAxeWing", mpCost: 38, multiplier: 2.7, target: "enemy"},
	}

	skills := make([]session.RoleSkill, 0, len(cases))
	for _, testCase := range cases {
		skill := session.RoleSkill{Name: testCase.name, Level: testCase.level, Type: testCase.sourceType, Description: "stale description"}
		skills = append(skills, skill)
		profile := sourceBattleSkillProfile(skill)
		if sourceBattleSkillCommandID(testCase.name) != testCase.commandID || profile.SourceType != testCase.sourceType || profile.SourceActionLabel != testCase.actionLabel || profile.MPCost != testCase.mpCost || profile.DamageMultiplier != testCase.multiplier || profile.DirectAttackBonus != testCase.directBonus {
			t.Fatalf("expected captured %s profile, got %+v", testCase.name, profile)
		}
	}

	commandsByID := map[string]CommandDefinition{}
	for _, command := range sourceBattleCommandDefinitions(skills) {
		commandsByID[command.ID] = command
	}
	for _, testCase := range cases {
		command, ok := commandsByID[testCase.commandID]
		if !ok || command.SourceType != testCase.sourceType || command.SourceActionLabel != testCase.actionLabel || command.MPCost != testCase.mpCost || command.DamageMultiplier != testCase.multiplier || command.Target != testCase.target {
			t.Fatalf("expected captured %s command definition, got %+v", testCase.name, command)
		}
	}

	limitedRuntime := &Runtime{RoleSkills: []session.RoleSkill{{Name: "连击", Level: 5, Type: "oneE"}}}
	if !limitedRuntime.isBattleCommandAllowedForActor("", CommandFistDoubleAtk) || limitedRuntime.isBattleCommandAllowedForActor("", CommandFistPowHit) {
		t.Fatalf("expected fist commands to require the learned captured skill")
	}
}

func TestWoodcutterFistDamagingSkillsUseCapturedProfiles(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-fist-damage",
		Round:            1,
		DefendingHandles: map[string]bool{},
		StatusEffects:    map[string]BattleStatusEffects{},
		RoleSkills: []session.RoleSkill{
			{Name: "连击", Level: 5, Type: "oneE"},
			{Name: "破魂打", Level: 5, Type: "oneE"},
		},
		Cells: []CellInfoPush{
			{BattleID: "battle-fist-damage", Handle: "player_fist", Camp: CampTeam, HP: 1000, MaxHP: 1000, MP: 100, MaxMP: 100, Attack: 100, Hit: 100},
			{BattleID: "battle-fist-damage", Handle: "enemy_1", Camp: CampEnemy, HP: 1000, MaxHP: 1000, Defense: 0},
		},
	}
	actor := runtime.cellByHandle("player_fist")
	target := runtime.cellByHandle("enemy_1")
	doubleAttack := runtime.resolveAttack(actor, target, CommandFistDoubleAtk)
	if doubleAttack.ActionName != "连击" || doubleAttack.SourceActionLabel != "w5/doubleAtk" || doubleAttack.Damage != 145 || actor.MP != 82 {
		t.Fatalf("expected 连击 captured action, damage, and MP cost, action=%+v actor=%+v", doubleAttack, actor)
	}

	actor.MP = 100
	target.HP = target.MaxHP
	breakSoul := runtime.resolveAttack(actor, target, CommandFistBreakSoul)
	if breakSoul.ActionName != "破魂打" || breakSoul.SourceActionLabel != "w5/breakSoul" || breakSoul.Damage != 180 || actor.MP != 64 {
		t.Fatalf("expected 破魂打 captured damage/direct bonus and MP cost, action=%+v actor=%+v", breakSoul, actor)
	}
}

func TestWoodcutterFistInfluxGasBuffsRestoreOnReplaceAndExpiry(t *testing.T) {
	runtime := &Runtime{
		BattleID:      "battle-fist-influx",
		StatusEffects: map[string]BattleStatusEffects{},
		RoleSkills:    []session.RoleSkill{{Name: "气运丹田", Level: 1, Type: "own"}},
		Cells: []CellInfoPush{
			{BattleID: "battle-fist-influx", Handle: "player_fist", Camp: CampTeam, HP: 1000, MaxHP: 1000, Attack: 360, Defense: 312, MgcDefense: 80},
		},
	}
	actor := runtime.cellByHandle("player_fist")
	buffs := runtime.applyFistInfluxGasStatusEffects(actor)
	if actor.Defense != 437 || actor.MgcDefense != 144 || actor.Attack != 396 || len(buffs) != 3 || buffs[0].Name != "护甲" || buffs[0].Display != "11.png" || buffs[0].Description != "提升对象125点物理防御力" || buffs[1].Name != "御法" || buffs[1].Display != "13.png" || buffs[1].Description != "提升对象64法术防御力" || buffs[2].Name != "斗志" || buffs[2].Display != "23.png" || buffs[2].Description != "提升对象36点物理攻击" {
		t.Fatalf("expected 气运丹田 captured three-buff values, actor=%+v buffs=%+v", actor, buffs)
	}

	runtime.applyFistInfluxGasStatusEffects(actor)
	if actor.Defense != 437 || actor.MgcDefense != 144 || actor.Attack != 396 {
		t.Fatalf("expected 气运丹田 recast to replace rather than stack buffs, actor=%+v", actor)
	}
	for index := 0; index < fistInfluxGasRounds; index += 1 {
		runtime.resolveStatusStartActions(actor)
	}
	if actor.Defense != 312 || actor.MgcDefense != 80 || actor.Attack != 360 {
		t.Fatalf("expected 气运丹田 buffs to restore all attributes on expiry, actor=%+v", actor)
	}
	clears := runtime.consumePendingClearBuffInfos()
	cleared := map[string]bool{}
	for _, clear := range clears {
		cleared[clear.Name] = clear.TargetHandle == actor.Handle
	}
	if !cleared["护甲"] || !cleared["御法"] || !cleared["斗志"] {
		t.Fatalf("expected 气运丹田 expiry to clear all three buff icons, got %+v", clears)
	}
}

func TestWoodcutterFistFinalLevelsUseCapturedStatusValues(t *testing.T) {
	runtime := &Runtime{
		BattleID:      "battle-fist-final-status-levels",
		StatusEffects: map[string]BattleStatusEffects{},
		RoleSkills: []session.RoleSkill{
			{Name: "气运丹田", Level: 5, Type: "own"},
			{Name: "移形换影", Level: 4, Type: "own"},
		},
		Cells: []CellInfoPush{
			{BattleID: "battle-fist-final-status-levels", Handle: "player_fist", Camp: CampTeam, HP: 1000, MaxHP: 1000, Attack: 360, Defense: 312, MgcDefense: 80, Dog: 267},
		},
	}
	actor := runtime.cellByHandle("player_fist")
	buffs := runtime.applyFistInfluxGasStatusEffects(actor)
	if actor.Defense != 499 || actor.MgcDefense != 160 || actor.Attack != 468 || len(buffs) != 3 ||
		buffs[0].Description != "提升对象187点物理防御力" || buffs[1].Description != "提升对象80法术防御力" || buffs[2].Description != "提升对象108点物理攻击" {
		t.Fatalf("expected final-level 气运丹田 values, actor=%+v buffs=%+v", actor, buffs)
	}
	shadowBuff := runtime.applyFistMoveShadowStatusEffect(actor)
	if actor.Dog != 387 || shadowBuff.Description != "提高对象120点回避" {
		t.Fatalf("expected final-level 移形换影 values, actor=%+v buff=%+v", actor, shadowBuff)
	}
}

func TestWoodcutterFistMoveShadowAndPowHitStatusLifecycle(t *testing.T) {
	shadowRuntime := &Runtime{
		BattleID:      "battle-fist-shadow",
		StatusEffects: map[string]BattleStatusEffects{},
		RoleSkills:    []session.RoleSkill{{Name: "移形换影", Level: 1, Type: "own"}},
		Cells: []CellInfoPush{
			{BattleID: "battle-fist-shadow", Handle: "player_fist", Camp: CampTeam, HP: 1000, MaxHP: 1000, Dog: 267},
		},
	}
	shadowActor := shadowRuntime.cellByHandle("player_fist")
	shadowBuff := shadowRuntime.applyFistMoveShadowStatusEffect(shadowActor)
	if shadowActor.Dog != 347 || shadowBuff.Name != "回避提升" || shadowBuff.Display != "14.png" || shadowBuff.Description != "提高对象80点回避" || shadowBuff.Round != 3 {
		t.Fatalf("expected 移形换影 captured dodge buff, actor=%+v buff=%+v", shadowActor, shadowBuff)
	}
	shadowRuntime.applyFistMoveShadowStatusEffect(shadowActor)
	if shadowActor.Dog != 347 {
		t.Fatalf("expected 移形换影 recast to replace rather than stack dodge, actor=%+v", shadowActor)
	}
	if !shadowRuntime.clearStatusEffect(shadowActor.Handle, "回避提升") || shadowActor.Dog != 267 {
		t.Fatalf("expected 移形换影 clear to restore dodge, actor=%+v effects=%+v", shadowActor, shadowRuntime.StatusEffects)
	}

	var stunRuntime *Runtime
	for index := 0; index < 200; index += 1 {
		candidate := &Runtime{
			BattleID:         fmt.Sprintf("battle-fist-pow-hit-%d", index),
			Round:            1,
			DefendingHandles: map[string]bool{},
			StatusEffects:    map[string]BattleStatusEffects{},
			RoleSkills:       []session.RoleSkill{{Name: "重烈", Level: 5, Type: "oneE"}},
			Cells: []CellInfoPush{
				{Handle: "player_fist", Camp: CampTeam, HP: 1000, MaxHP: 1000, MP: 100, Attack: 100, Hit: 100},
				{Handle: "enemy_1", Camp: CampEnemy, HP: 1000, MaxHP: 1000, Dog: 0},
			},
		}
		if candidate.hashBattleRollWithSalt(candidate.cellByHandle("player_fist"), candidate.cellByHandle("enemy_1"), CommandFistPowHit, "status:眩晕") < fistPowHitStunChance {
			stunRuntime = candidate
			break
		}
	}
	if stunRuntime == nil {
		t.Fatal("expected deterministic 重烈 stun roll")
	}
	stunAction := stunRuntime.resolveAttack(stunRuntime.cellByHandle("player_fist"), stunRuntime.cellByHandle("enemy_1"), CommandFistPowHit)
	stunBuffs := stunRuntime.consumePendingBuffInfos()
	if stunAction.TargetActionStateCode != "0" || stunAction.SourceActionLabel != "w5/powHit" || len(stunBuffs) != 1 || stunBuffs[0].Name != "眩晕" || stunBuffs[0].Display != "9.png" || stunBuffs[0].Round != 2 {
		t.Fatalf("expected 重烈 hit to emit captured stun buff, action=%+v buffs=%+v", stunAction, stunBuffs)
	}
	for index := 0; index < fistPowHitStunRounds; index += 1 {
		actions, skipped := stunRuntime.resolveStatusStartActions(stunRuntime.cellByHandle("enemy_1"))
		if !skipped || len(actions) != 1 || actions[0].SourceActionLabel != "yun" {
			t.Fatalf("expected 重烈 stun to use yun skip-turn action, actions=%+v skipped=%t", actions, skipped)
		}
	}
	if clears := stunRuntime.consumePendingClearBuffInfos(); len(clears) != 1 || clears[0].Name != "眩晕" {
		t.Fatalf("expected 重烈 stun to clear after two rounds, got %+v", clears)
	}

	dodgeRuntime := &Runtime{
		BattleID:         "battle-fist-pow-hit-dodge",
		DefendingHandles: map[string]bool{},
		StatusEffects:    map[string]BattleStatusEffects{},
		RoleSkills:       []session.RoleSkill{{Name: "重烈", Level: 5, Type: "oneE"}},
	}
	dodgeAction := dodgeRuntime.resolveAttack(&CellInfoPush{Handle: "player_fist", Camp: CampTeam, Attack: 100, Hit: 0}, &CellInfoPush{Handle: "enemy_1", Camp: CampEnemy, HP: 1000, MaxHP: 1000, Dog: 1}, CommandFistPowHit)
	if dodgeAction.TargetActionStateCode != "1" || len(dodgeRuntime.PendingBuffInfos) != 0 || len(dodgeRuntime.StatusEffects) != 0 {
		t.Fatalf("expected 重烈 dodge to suppress stun, action=%+v effects=%+v", dodgeAction, dodgeRuntime.StatusEffects)
	}
}

func TestWoodcutterFistUltimateRequiresSoulPower(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-fist-ultimate",
		Round:            1,
		Phase:            PhaseCommand,
		ActiveHandle:     "player_fist",
		nextSequence:     1,
		ConsumedSequence: map[int]bool{},
		DefendingHandles: map[string]bool{},
		StoredPower:      map[string]int{},
		RoleSkills:       []session.RoleSkill{{Name: "奥义.修罗幻翼拳", Level: 4, Type: "oneE"}},
		Cells: []CellInfoPush{
			{BattleID: "battle-fist-ultimate", Handle: "player_fist", Camp: CampTeam, HP: 1000, MaxHP: 1000, MP: 100, MaxMP: 100, Attack: 100, Hit: 100},
			{BattleID: "battle-fist-ultimate", Handle: "enemy_1", Camp: CampEnemy, HP: 1000, MaxHP: 1000, Defense: 0},
		},
	}
	request := ActionRequest{BattleID: runtime.BattleID, ActorHandle: "player_fist", CommandID: CommandFistPowerAxeWing, TargetHandle: "enemy_1", Round: 1, Sequence: 1}
	if result := runtime.ProcessAction(request); result.ErrorCode != "insufficient_power" {
		t.Fatalf("expected 奥义.修罗幻翼拳 to require three stored power, got %+v", result)
	}
	runtime.StoredPower["player_fist"] = fistPowerAxeWingRequiredPower
	result := runtime.ProcessAction(request)
	if result.ErrorCode != "" || len(result.Actions) == 0 {
		t.Fatalf("expected 奥义.修罗幻翼拳 action with three stored power, got %+v", result)
	}
	action := result.Actions[0]
	actor := runtime.cellByHandle("player_fist")
	if action.SourceActionLabel != "w5/PowerAxeWing" || action.Damage != 810 || action.TargetHP != 190 || actor.MP != 62 {
		t.Fatalf("expected 奥义.修罗幻翼拳 captured action and MP cost, action=%+v actor=%+v", action, actor)
	}
}

func TestWoodcutter777SharedPhysicalFormulaReplaysCapturedDamage(t *testing.T) {
	previousRoll := sourceBattleAttackRoll
	defer func() { sourceBattleAttackRoll = previousRoll }()

	runtime := &Runtime{StoredPower: map[string]int{"acct-777-role-001": fistPowerAxeWingRequiredPower}}
	actor := &CellInfoPush{Handle: "acct-777-role-001", Camp: CampTeam, Attack: 361}
	map52Robber := &CellInfoPush{Handle: "map52_robber", Camp: CampEnemy, Defense: 321}

	// 113.96% is the captured p50 attack roll for Lv5 破魂打 against map52 盗贼.
	sourceBattleAttackRoll = func(int) int { return 1896 }
	breakSoul := runtime.baseBattleDamageValue(actor, commandProfile{DamageMultiplier: 1.5, DirectAttackBonus: 0.3}, runtime.effectiveBattleDefenseValue(actor, map52Robber, false, "physical"))
	if got := int(math.Round(breakSoul)); got != 580 {
		t.Fatalf("expected shared formula to replay 777 Lv5 破魂打 p50=580, got %.4f -> %d", breakSoul, got)
	}

	actor.Attack = 397
	map48Robber := &CellInfoPush{Handle: "map48_robber", Camp: CampEnemy, Defense: 121}
	// 96.98% with three stored soul power gives the captured 5423 critical exactly.
	sourceBattleAttackRoll = func(int) int { return 198 }
	ultimate := runtime.baseBattleDamageValue(actor, commandProfile{DamageMultiplier: 2.4, DamageUsesStoredPower: true}, runtime.effectiveBattleDefenseValue(actor, map48Robber, false, "physical"))
	if got := int(math.Round(ultimate * 2)); got != 5423 {
		t.Fatalf("expected shared formula to defer rounding until after 777 ultimate critical, got %.4f -> %d", ultimate, got)
	}
}
