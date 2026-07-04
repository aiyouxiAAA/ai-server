package battle

import (
	"encoding/json"
	"fmt"
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

func useSourceEncounterRoll(roll func(int) int) func() {
	previous := sourceEncounterRoll
	sourceEncounterRoll = roll
	return func() {
		sourceEncounterRoll = previous
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
	if len(runtime.livingCells(CampTeam)) != 2 || bundle.StartCommand.ActorHandle != leader.Role.RoleID {
		t.Fatalf("expected leader to act first in two-player team battle, got cells=%+v start=%+v", runtime.Cells, bundle.StartCommand)
	}
	target := runtime.firstLiving(CampEnemy)
	if target == nil {
		t.Fatal("expected enemy target")
	}

	result := runtime.ProcessAction(ActionRequest{
		BattleID:     bundle.Start.BattleID,
		ActorHandle:  leader.Role.RoleID,
		CommandID:    CommandNormalAttack,
		TargetHandle: target.Handle,
		Round:        bundle.StartCommand.Round,
		Sequence:     bundle.StartCommand.Sequence,
	})
	if result.ErrorCode != "" {
		t.Fatalf("expected leader action to succeed, got %s", result.ErrorCode)
	}
	if result.StartCommand != nil || runtime.PendingStart == nil {
		t.Fatalf("expected action to queue pending startCommand, got result=%+v pending=%+v", result.StartCommand, runtime.PendingStart)
	}
	playOver := runtime.ProcessPlayOver(PlayOverRequest{BattleID: bundle.Start.BattleID})
	if playOver.ErrorCode != "" {
		t.Fatalf("expected playOver to succeed, got %s", playOver.ErrorCode)
	}
	if playOver.StartCommand == nil || playOver.StartCommand.ActorHandle != member.Role.RoleID {
		t.Fatalf("expected next startCommand for member, got %+v", playOver.StartCommand)
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
	if enemy.Cell.Name != "盗贼" || enemy.Cell.DisplayURL != "monstermap/robber.swf" || enemy.Cell.MaxHP != 445 || enemy.Cell.MaxMP != 194 || enemy.Cell.Attack != 164 {
		t.Fatalf("expected captured map49 robber config, got %+v", enemy)
	}
	if damage := (&Runtime{}).baseBattleDamage(&enemy.Cell, commandProfile{DamageMultiplier: 1}, 122); damage != 42 {
		t.Fatalf("expected captured map49 robber attack 164 against captured player defense 122 to deal 42, got %d", damage)
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
	if visibleBoss.Cell.Attack != 240 {
		t.Fatalf("expected captured cracktoad base attack 240 from normal-attack HP deltas, got %+v", visibleBoss.Cell)
	}
	if visibleBoss.QueueIndexTeam != 1 || visibleBoss.QueueIndexEnemy != 4 {
		t.Fatalf("expected visible monster queue indexes 1/4, got %+v", visibleBoss)
	}

	visibleAttackCases := []struct {
		mapID  string
		handle string
		name   string
		level  int
		attack int
		label  string
	}{
		{mapID: "131", handle: "8128205778897212", name: "武斗蛤蟆", level: 12, attack: 208, label: "普通攻击"},
		{mapID: "132", handle: "1656205827185847", name: "武斗蛤蟆", level: 13, attack: 230, label: "普通攻击"},
		{mapID: "133", handle: "8112205902790159", name: "武斗蛤蟆", level: 14, attack: 234, label: "普通攻击"},
		{mapID: "144", handle: "2762206074545916", name: "武斗蛤蟆", level: 16, attack: 205, label: "普通攻击"},
		{mapID: "140", handle: "8430206376341780", name: "武斗蛤蟆", level: 17, attack: 211, label: "普通攻击"},
		{mapID: "137", handle: "4889205982270617", name: "剑术蛤蟆", level: 15, attack: 246, label: "普通攻击"},
		{mapID: "144", handle: "2768206074548639", name: "剑术蛤蟆", level: 17, attack: 244, label: "普通攻击"},
		{mapID: "145", handle: "2890206197338884", name: "剑术蛤蟆", level: 18, attack: 220, label: "普通攻击"},
		{mapID: "143", handle: "5172206909807859", name: "剑术蛤蟆", level: 18, attack: 121, label: "普通攻击"},
		{mapID: "137", handle: "4895205982272135", name: "法术蛤蟆", level: 16, attack: 202, label: "法术普通攻击"},
		{mapID: "143", handle: "5166206909805441", name: "法术蛤蟆", level: 17, attack: 143, label: "法术普通攻击"},
	}
	for _, testCase := range visibleAttackCases {
		config, ok := sourceVisibleMonsterConfigForHandle(testCase.mapID, testCase.handle)
		if !ok {
			t.Fatalf("expected captured shuiliandong visible monster config for %s/%s", testCase.mapID, testCase.handle)
		}
		if config.Cell.Name != testCase.name || config.Cell.Level != testCase.level || config.Cell.Attack != testCase.attack || config.Cell.CommandLabel != testCase.label {
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
	if len(map50Configs) != 2 || map50Configs[0].Cell.Attack != 203 || map50Configs[1].Cell.Attack != 216 {
		t.Fatalf("expected captured map50 normal enemy attacks 203/216, got %+v", map50Configs)
	}
	for _, config := range map50Configs {
		if config.Cell.Name == "魔族射手" {
			t.Fatalf("expected map50 special-event 魔族射手 to stay out of normal wild encounters, got %+v", map50Configs)
		}
	}

	for mapID, configs := range sourceWildEnemyConfigsByMapID {
		for _, config := range configs {
			if config.Cell.Attack <= 0 {
				t.Fatalf("expected captured enemy attack to be filled for map %s, got %+v", mapID, config.Cell)
			}
		}
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
		attack int
		label  string
	}{
		{mapID: "146", handle: "6887685480585492", name: "蛮族刀客", level: 12, maxHP: 520, maxMP: 214, attack: 208, label: "普通攻击"},
		{mapID: "147", handle: "7633284548716137", name: "蛮族刀客", level: 12, maxHP: 520, maxMP: 214, attack: 208, label: "普通攻击"},
		{mapID: "149", handle: "3218685759638239", name: "黄风二寨主", level: 19, maxHP: 1200, maxMP: 564, attack: 240, label: "普通攻击"},
		{mapID: "153", handle: "7494686002239485", name: "咒巫师", level: 16, maxHP: 500, maxMP: 550, attack: 202, label: "法术普通攻击"},
		{mapID: "155", handle: "2600686416056495", name: "黄风大寨主", level: 20, maxHP: 1500, maxMP: 564, attack: 260, label: "普通攻击"},
		{mapID: "155", handle: "2800686416057704", name: "黄风寨夫人", level: 20, maxHP: 1200, maxMP: 704, attack: 240, label: "普通攻击"},
	}

	for _, testCase := range cases {
		config, ok := sourceVisibleMonsterConfigForHandle(testCase.mapID, testCase.handle)
		if !ok {
			t.Fatalf("expected captured huangfengzhai visible monster config for %s/%s", testCase.mapID, testCase.handle)
		}
		if config.Cell.Name != testCase.name || config.Cell.Level != testCase.level || config.Cell.MaxHP != testCase.maxHP || config.Cell.MaxMP != testCase.maxMP || config.Cell.Attack != testCase.attack || config.Cell.CommandLabel != testCase.label {
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
		attack int
		label  string
	}{
		{mapID: "158", handle: "5837621591158929", name: "黑影", level: 25, maxHP: 2300, maxMP: 724, attack: 402, label: "普通攻击"},
		{mapID: "160", handle: "3824621817512450", name: "蛮虎怪", level: 25, maxHP: 3000, maxMP: 584, attack: 303, label: "普通攻击"},
		{mapID: "161", handle: "9333621886743795", name: "蛮虎队长", level: 28, maxHP: 3500, maxMP: 600, attack: 322, label: "普通攻击"},
		{mapID: "163", handle: "8094622782649492", name: "盘狮队长", level: 26, maxHP: 3300, maxMP: 440, attack: 228, label: "普通攻击"},
		{mapID: "165", handle: "3856623006745359", name: "黑影队长", level: 26, maxHP: 3000, maxMP: 760, attack: 276, label: "普通攻击"},
		{mapID: "167", handle: "7550622260838906", name: "蚩颅王", level: 30, maxHP: 6000, maxMP: 600, attack: 236, label: "普通攻击"},
	}

	for _, testCase := range cases {
		config, ok := sourceVisibleMonsterConfigForHandle(testCase.mapID, testCase.handle)
		if !ok {
			t.Fatalf("expected captured shihuku visible monster config for %s/%s", testCase.mapID, testCase.handle)
		}
		if config.Cell.Name != testCase.name || config.Cell.Level != testCase.level || config.Cell.MaxHP != testCase.maxHP || config.Cell.MaxMP != testCase.maxMP || config.Cell.Attack != testCase.attack || config.Cell.CommandLabel != testCase.label {
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
	if runtime.PendingStart == nil || runtime.PendingStart.Power != 2 || runtime.powerFor("player_21424") != 2 {
		t.Fatalf("expected 250/1000 HP loss to set stored power 2 without damage bonus, pending=%+v stored=%d", runtime.PendingStart, runtime.powerFor("player_21424"))
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
	if runtime.PendingStart == nil || runtime.PendingStart.Power != 2 || runtime.powerFor("player_21424") != 2 {
		t.Fatalf("expected stored power 2 to survive lighter enemy hit, pending=%+v stored=%d", runtime.PendingStart, runtime.powerFor("player_21424"))
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
	}{
		{mapID: "84", mapName: "竹林_1", enemyName: "绿甲螳螂", displayURL: "monstermap/greenmantis.swf", queueTeam: 1, queueEnemy: 4},
		{mapID: "85", mapName: "竹林_2", enemyName: "绿甲螳螂", displayURL: "monstermap/greenmantis.swf", queueTeam: 1, queueEnemy: 4},
		{mapID: "86", mapName: "竹林_3", enemyName: "小竹妖", displayURL: "monstermap/bambooboy.swf", queueTeam: 1, queueEnemy: 4},
		{mapID: "87", mapName: "竹林_4", enemyName: "跳跳竹", displayURL: "monstermap/jumpboo.swf", queueTeam: 1, queueEnemy: 4},
		{mapID: "88", mapName: "竹林_5", enemyName: "刀手螳螂", displayURL: "monstermap/kinfemantis.swf", queueTeam: 1, queueEnemy: 4},
		{mapID: "90", mapName: "竹林_7", enemyName: "竹炮", displayURL: "monstermap/boobomb.swf", queueTeam: 1, queueEnemy: 4},
		{mapID: "97", mapName: "竹林_10", enemyName: "小竹妖", displayURL: "monstermap/bambooboy.swf", queueTeam: 1, queueEnemy: 4},
	}

	for _, testCase := range cases {
		runtime, bundle, ok := NewWildBattle(role, playerBase, StartRequest{MapID: testCase.mapID, MapName: testCase.mapName})
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

func TestNewWildBattleUsesCapturedPlainEnemyStats(t *testing.T) {
	defer useSourceEncounterRoll(func(maxExclusive int) int {
		return maxExclusive - 1
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
		attack     int
		cellCount  int
	}{
		{mapID: "11", mapName: "涧庭道_1", enemyName: "蟾蜍", displayURL: "monstermap/toad.swf", level: 1, maxHP: 15, maxMP: 9, attack: 28, cellCount: 2},
		{mapID: "37", mapName: "平原_6", enemyName: "花妖", displayURL: "monstermap/huayao.swf", level: 10, maxHP: 230, maxMP: 150, attack: 162, cellCount: 2},
		{mapID: "49", mapName: "平原_15", enemyName: "盗贼", displayURL: "monstermap/robber.swf", level: 15, maxHP: 445, maxMP: 194, attack: 164, cellCount: 3},
		{mapID: "52", mapName: "平原_18", enemyName: "盗贼", displayURL: "monstermap/robber.swf", level: 18, maxHP: 470, maxMP: 220, attack: 202, cellCount: 2},
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
		if enemy.Attack != testCase.attack {
			t.Fatalf("expected captured enemy attack for map %s to be %d, got %+v", testCase.mapID, testCase.attack, enemy)
		}
	}
}

func TestNewWildBattleUsesExpandedCapturedPlainEnemyStats(t *testing.T) {
	defer useSourceEncounterRoll(func(maxExclusive int) int {
		return maxExclusive - 1
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
		attack     int
		cellCount  int
	}{
		{mapID: "34", mapName: "平原_3", enemyName: "爆骨猪", displayURL: "monstermap/bomepig.swf", level: 12, maxHP: 260, maxMP: 155, attack: 193, cellCount: 2},
		{mapID: "35", mapName: "平原_4", enemyName: "爆骨猪", displayURL: "monstermap/bomepig.swf", level: 13, maxHP: 295, maxMP: 162, attack: 211, cellCount: 2},
		{mapID: "36", mapName: "平原_5", enemyName: "爆骨猪", displayURL: "monstermap/bomepig.swf", level: 13, maxHP: 280, maxMP: 145, attack: 164, cellCount: 3},
		{mapID: "39", mapName: "平原_8", enemyName: "尖刀暴牙", displayURL: "monstermap/jdby.swf", level: 12, maxHP: 280, maxMP: 130, attack: 200, cellCount: 2},
		{mapID: "40", mapName: "平原_9", enemyName: "尖刀暴牙", displayURL: "monstermap/jdby.swf", level: 12, maxHP: 280, maxMP: 130, attack: 200, cellCount: 2},
		{mapID: "41", mapName: "平原_10", enemyName: "牙菇", displayURL: "monstermap/yagu.swf", level: 10, maxHP: 250, maxMP: 190, attack: 229, cellCount: 2},
		{mapID: "43", mapName: "平原_12", enemyName: "刺鸟", displayURL: "monstermap/swordbird.swf", level: 11, maxHP: 230, maxMP: 160, attack: 175, cellCount: 2},
		{mapID: "44", mapName: "平原_13", enemyName: "尖刀暴牙", displayURL: "monstermap/jdby.swf", level: 12, maxHP: 250, maxMP: 130, attack: 187, cellCount: 2},
		{mapID: "48", mapName: "平原_14", enemyName: "盗贼", displayURL: "monstermap/robber.swf", level: 14, maxHP: 330, maxMP: 210, attack: 171, cellCount: 2},
		{mapID: "50", mapName: "平原_16", enemyName: "巡路小鬼", displayURL: "monstermap/lilghost.swf", level: 16, maxHP: 520, maxMP: 160, attack: 203, cellCount: 3},
		{mapID: "51", mapName: "平原_17", enemyName: "盗贼", displayURL: "monstermap/robber.swf", level: 18, maxHP: 470, maxMP: 220, attack: 202, cellCount: 3},
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
		if enemy.Attack != testCase.attack {
			t.Fatalf("expected captured enemy attack for map %s to be %d, got %+v", testCase.mapID, testCase.attack, enemy)
		}
	}
}

func TestNewWildBattleSelectsCapturedMapCandidatesByStageFocusX(t *testing.T) {
	defer useSourceEncounterRoll(func(maxExclusive int) int {
		return maxExclusive - 1
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
		mapID          string
		mapName        string
		stageFocusX    float64
		enemyName      string
		level          int
		maxHP          int
		maxMP          int
		attack         int
		encounterLabel string
		enemyCount     int
	}{
		{mapID: "50", mapName: "平原_16", stageFocusX: 0, enemyName: "巡路小鬼", level: 16, maxHP: 520, maxMP: 160, attack: 203, encounterLabel: "平原_16 暗雷", enemyCount: 2},
		{mapID: "50", mapName: "平原_16", stageFocusX: 800, enemyName: "巡路小鬼", level: 17, maxHP: 545, maxMP: 184, attack: 216, encounterLabel: "平原_16 暗雷", enemyCount: 2},
		{mapID: "50", mapName: "平原_16", stageFocusX: 1600, enemyName: "巡路小鬼", level: 16, maxHP: 520, maxMP: 160, attack: 203, encounterLabel: "平原_16 暗雷", enemyCount: 2},
		{mapID: "52", mapName: "平原_18", stageFocusX: 800, enemyName: "单刀狼人", level: 21, maxHP: 2500, maxMP: 334, attack: 260, encounterLabel: "平原_18 首领", enemyCount: 4},
	}

	for _, testCase := range cases {
		_, bundle, ok := NewWildBattle(role, playerBase, StartRequest{
			MapID:       testCase.mapID,
			MapName:     testCase.mapName,
			StageFocusX: testCase.stageFocusX,
		})
		if !ok || len(bundle.Cells) != testCase.enemyCount+1 {
			t.Fatalf("expected captured candidate for map %s at %.0f, got ok=%v bundle=%+v", testCase.mapID, testCase.stageFocusX, ok, bundle)
		}
		if bundle.Start.EncounterLabel != testCase.encounterLabel {
			t.Fatalf("expected encounter label %s, got %+v", testCase.encounterLabel, bundle.Start)
		}
		enemy := bundle.Cells[1]
		if enemy.Name != testCase.enemyName || enemy.Level != testCase.level || enemy.MaxHP != testCase.maxHP || enemy.MaxMP != testCase.maxMP || enemy.Attack != testCase.attack {
			t.Fatalf("expected candidate enemy for map %s at %.0f, got %+v", testCase.mapID, testCase.stageFocusX, enemy)
		}
		if testCase.mapID == "52" && testCase.stageFocusX == 800 {
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
		return 0
	})
	_, bossBundle, bossOK := NewWildBattle(role, playerBase, StartRequest{MapID: "52", MapName: "平原_18", StageFocusX: 800})
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

	runtime, bundle, ok := NewWildBattle(role, playerBase, StartRequest{MapID: "52", MapName: "平原_18", StageFocusX: 800})
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
			expectedDamage: 105,
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
			expectedDamage: 60,
			expectedMP:     80,
		},
		{
			name:           "疾风刺",
			level:          1,
			description:    "f_s_疾风刺^5BC46D&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@20&4@对敌人造成40%的物理伤害&0;击中敌人时有92%的机率使对方进入迟钝状态(削减对方50%的命中和回避)3回合<br><font color='#00cc00'>叠加施放将削弱其造成迟钝的功效</font>",
			commandID:      CommandJiFengCi,
			label:          "w3/windCut",
			expectedDamage: 1,
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
			expectedDamage: 105,
			expectedMP:     82,
		},
		{
			name:           "贯甲连矢",
			level:          5,
			description:    "f_s_贯甲连矢^ffffff&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@28&4@<font color='#00cc00'>特殊发动条件:需要【穿甲箭x1】</font><br>提升25%的物理伤害&0;进攻时增加30%（无视防御）的物理攻击力.",
			commandID:      CommandGuanJiaLianShi,
			label:          "w1/breakArmorShoot2",
			expectedDamage: 115,
			expectedMP:     72,
		},
		{
			name:           "暗影箭",
			level:          1,
			description:    "f_s_暗影箭^5BC46D&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@20&4@<font color='#00cc00'>特殊发动条件:需要【暗影箭x1】</font><br>造成72%的物理伤害&0;击中敌人时有17%的机率使敌人进入混乱状态2回合",
			commandID:      CommandAnYingJian,
			label:          "w1/darkShoot",
			expectedDamage: 32,
			expectedMP:     80,
		},
		{
			name:           "毒矢",
			level:          1,
			description:    "f_s_毒矢^5BC46D&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@15&4@<font color='#00cc00'>特殊发动条件:需要【毒箭x1】<br>叠加施放将削弱其造成中毒的功效</font><br>对敌人造成90%的物理伤害&0;击中敌人时有70%的机率使敌人中毒(4回合内降低对方20%的物理防御和魔法防御&0;每回合使敌人损失气力为物理攻击的5%~10%)",
			commandID:      CommandDuShi,
			label:          "w1/drugShoot",
			expectedDamage: 50,
			expectedMP:     85,
		},
		{
			name:           "奥义.轰雷矢",
			level:          1,
			description:    "f_s_奥义.轰雷矢^00ccff&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@26&4@<font color='#00cc00'>特殊发动条件:需要2格魂元</font><br>提升120%的魔法伤害&0;击中敌人时有20%的机率使敌人进入麻痹状态(每回合使其损失气力为魔法攻击的30%)2回合",
			commandID:      CommandAoYiHongLeiShi,
			label:          "w1/bombThunderShoot",
			expectedDamage: 69,
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
			name:       "暗影箭",
			level:      1,
			desc:       "f_s_暗影箭^5BC46D&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@20&4@<font color='#00cc00'>特殊发动条件:需要【暗影箭x1】</font><br>造成72%的物理伤害&0;击中敌人时有17%的机率使敌人进入混乱状态2回合",
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
	if commandID := sourceBattleSkillCommandID("投毒"); commandID != row["command_id"] {
		t.Fatalf("expected 投毒 command id from classicdata %q, got %q", row["command_id"], commandID)
	}
	profile := sourceBattleSkillProfile(session.RoleSkill{
		Name:  "投毒",
		Level: 1,
		Type:  "oneE",
	})
	if profile.ActionName != row["action_name"] || profile.SourceType != row["source_type"] || profile.SourceActionLabel != row["source_action_label"] {
		t.Fatalf("expected 投毒 profile to use classicdata row %+v, got %+v", row, profile)
	}
	commands := sourceBattleCommandDefinitions([]session.RoleSkill{{Name: "投毒", Level: 1, Type: "oneE"}})
	var touDu CommandDefinition
	for _, command := range commands {
		if command.ID == row["command_id"] {
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
			Name:        "暗影箭",
			Level:       1,
			Type:        "oneE",
			Description: "f_s_暗影箭^5BC46D&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@20&4@<font color='#00cc00'>特殊发动条件:需要【暗影箭x1】</font><br>造成72%的物理伤害&0;击中敌人时有17%的机率使敌人进入混乱状态2回合",
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
	if action.Damage != 75 || action.TargetHP != 425 || target.HP != 425 {
		t.Fatalf("expected round(100*1.1)-50 plus round(100*0.15)=75 damage, got action=%+v target=%+v", action, target)
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
	if runtime.PendingStart == nil || runtime.PendingStart.ActorHandle != "player_leader" {
		t.Fatalf("expected next startCommand to skip dead poison actor, pending=%+v", runtime.PendingStart)
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
	if runtime.PendingStart != nil {
		t.Fatalf("expected no next startCommand after last enemy poison death, got %+v", runtime.PendingStart)
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
	if runtime.PendingStart == nil || runtime.PendingStart.ActorHandle != "player_21424" || runtime.PendingStart.Round != 2 || runtime.PendingStart.Sequence != 2 {
		t.Fatalf("expected skipped turn to queue the next player startCommand, got %+v", runtime.PendingStart)
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
	if runtime.PendingStart == nil || runtime.PendingStart.ActorHandle != "player_21424" || runtime.PendingStart.Round != 3 || runtime.PendingStart.Sequence != 3 {
		t.Fatalf("expected 眩晕 skip chain to queue player startCommand after monster continuation, got %+v", runtime.PendingStart)
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
	if action.Damage != 159 || action.TargetHP != 886 {
		t.Fatalf("expected helixAtk to use captured 1.32x damage path without stored power bonus, got %+v", action)
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
	if action.Damage != 162 || runtime.Cells[0].HP != 923 || runtime.Cells[1].MP != 540 {
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
	if action.Damage != 73 || action.TargetHP != 1012 || actor.MP != 554 {
		t.Fatalf("expected 暗月斩 to use captured MP cost 10 and physical damage path, action=%+v actor=%+v", action, actor)
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
	if action.Damage != 100 || runtime.Cells[0].HP != 985 || runtime.Cells[1].MP != 554 {
		t.Fatalf("expected 裂震击 to use captured MP cost 10 and physical damage path, action=%+v cells=%+v", action, runtime.Cells)
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
	if runtime.PendingStart == nil || runtime.PendingStart.ActorHandle == "player_21424" {
		t.Fatalf("expected no startCommand to be queued for confused actor, pendingStart=%+v", runtime.PendingStart)
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
	if action.Damage != 218 || action.TargetHP != 555 || actor.MP != 544 {
		t.Fatalf("expected 滑行连击 to use captured damage and MP cost, got action=%+v target=%+v", action, target)
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
			{Handle: "enemy_whorllion", Camp: CampEnemy, Name: "盘狮怪", DisplayURL: "monstermap/whorllion.swf", HP: 3000, MaxHP: 3000, MP: 384, MaxMP: 384, Attack: 228},
			{Handle: "enemy_chiluking", Camp: CampEnemy, Name: "蚩颅王", DisplayURL: "monstermap/chiluking.swf", HP: 6000, MaxHP: 6000, MP: 600, MaxMP: 600, Attack: 236},
		},
	}

	lion := runtime.resolveAttack(runtime.cellByHandle("enemy_whorllion"), runtime.cellByHandle("player_21424"), CommandEnemyLionRoars)
	if lion.ActionName != "狮吼" || lion.SourceActionLabel != "lionroars" || lion.SourceMode != "1" {
		t.Fatalf("expected shihuku whorllion 狮吼/lionroars action, got %+v", lion)
	}
	if lion.Damage != 167 {
		t.Fatalf("expected 狮吼 to use capture-backed 1.26 damage multiplier, got %+v", lion)
	}
	if runtime.cellByHandle("enemy_whorllion").MP != 374 {
		t.Fatalf("expected 狮吼 to use captured 10 MP cost, got %+v", runtime.cellByHandle("enemy_whorllion"))
	}

	piece := runtime.resolveAttack(runtime.cellByHandle("enemy_chiluking"), runtime.cellByHandle("player_21424"), CommandEnemyPieceAtk)
	if piece.ActionName != "撕裂" || piece.SourceActionLabel != "pieceAttack" || piece.SourceMode != "1" {
		t.Fatalf("expected shihuku 撕裂/pieceAttack action, got %+v", piece)
	}
	if piece.Damage != 210 {
		t.Fatalf("expected shihuku 撕裂 to use capture-backed 1.4 damage multiplier, got %+v", piece)
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
	if effect.SourceHandle != "enemy_chiluking" || effect.SourceSkill != "撕裂" || effect.SourceAttack != 236 || effect.TickMinPercent != 10 || effect.TickMaxPercent != 15 || effect.Rounds != 5 {
		t.Fatalf("expected shihuku 蚩颅王 外伤 to use source attack 10%%~15%%, got %+v", effect)
	}

	gold := runtime.resolveEnemyCommandActions(runtime.cellByHandle("enemy_chiluking"), runtime.cellByHandle("player_21424"), CommandEnemyGoldHit)[0]
	if gold.ActionName != "黄金穿刺" || gold.SourceActionLabel != "goldhit" || gold.TargetHandle != "all" || gold.SourceMode != "1" {
		t.Fatalf("expected shihuku 蚩颅王 黄金穿刺/goldhit all-target action, got %+v", gold)
	}
	if gold.Damage != 286 {
		t.Fatalf("expected shihuku 黄金穿刺 to use capture-backed 1.72 damage multiplier, got %+v", gold)
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
		Attack:     236,
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
		Attack:     624,
	}
	battleID, ok := findBattleIDForEnemyCommand(CommandEnemyRoundAtk, enemy, target)
	if !ok {
		t.Fatal("expected to find deterministic battle id for 机木锥兵 轮转刺伤 5/100")
	}

	runtime := &Runtime{BattleID: battleID, Round: 1, nextSequence: 1}
	if command := runtime.enemyBattleCommand(enemy, target); command != CommandEnemyRoundAtk {
		t.Fatalf("expected 机木锥兵 to choose captured 轮转刺伤/roundatk, got %s with battle id %s", command, battleID)
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
	if visibleMonster.Cell.DamageDefenseType != "magic" || action.Damage != 113 || action.TargetHP != 932 {
		t.Fatalf("expected magicpanda normal attack to use captured magic-defense damage path without stored power bonus, config=%+v action=%+v target=%+v", visibleMonster.Cell, action, target)
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
	if enemyAction.Damage != 159 || enemyAction.TargetHP != 886 {
		t.Fatalf("expected cracktoad enemy turn to apply captured helixAtk multiplier without stored power bonus, got %+v", enemyAction)
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
	if enemyAction.Damage != 82 || enemyAction.TargetHP != 963 {
		t.Fatalf("expected cracktoad normal attack to apply captured base damage without stored power bonus, got %+v", enemyAction)
	}
	if runtime.Cells[1].MP != 564 || len(enemyAction.RefreshInfos) != 1 {
		t.Fatalf("expected cracktoad normal attack to keep MP unchanged, got cells=%+v action=%+v", runtime.Cells, enemyAction)
	}
}
