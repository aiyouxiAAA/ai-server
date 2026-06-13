package battle

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"ai-server/internal/session"
)

func useSourceEncounterRoll(roll func(int) int) func() {
	previous := sourceEncounterRoll
	sourceEncounterRoll = roll
	return func() {
		sourceEncounterRoll = previous
	}
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
			items:    []string{"毛皮x2", "铜钱x106", "盗贼的首级x1", "黄风腰带x1", "兽骨x1", "雪莲花x1"},
		},
		{
			mapID:    "149",
			handle:   "3220685759639165",
			expDelta: 336,
			items:    []string{"毛皮x2", "铜钱x106", "盗贼的首级x1", "黄风腰带x1", "兽骨x1", "雪莲花x1"},
		},
		{
			mapID:    "155",
			handle:   "2600686416056495",
			expDelta: 720,
			items:    []string{"红方巾x2", "绸缎x3", "铜钱x110", "盗贼的首级x1", "呼啸战靴x1", "寨夫人上衣x1", "寨夫人护腕x1", "宝匣x1"},
		},
		{
			mapID:    "155",
			handle:   "2800686416057704",
			expDelta: 720,
			items:    []string{"红方巾x2", "绸缎x3", "铜钱x110", "盗贼的首级x1", "呼啸战靴x1", "寨夫人上衣x1", "寨夫人护腕x1", "宝匣x1"},
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

	if _, ok := sourceVisibleMonsterConfigForHandle("147", "6887685480585492"); ok {
		t.Fatal("expected map147 to stay without visible monster config until source packets are captured")
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
			name:       "奥义.雷魂斩",
			level:      1,
			desc:       "f_s_奥义.雷魂斩^00ccff&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@24&4@<font color='#00cc00'>特殊发动条件:需要3格魂元</font><br>提升240%的物理伤害",
			label:      "w8/thunderSoulAtk",
			mpCost:     24,
			multiplier: 3.4,
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

	shiXue := sourceBattleSkillProfile(session.RoleSkill{
		Name:        "嗜血斩",
		Level:       3,
		Type:        "oneE",
		Description: "f_s_嗜血斩^5BC46D&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@24&4@对敌人造成92%的物理伤害&0;并有82%机率将对敌人造成伤害的70%转换为气力</font>",
	})
	if shiXue.SourceActionLabel != "w8/xyz2" || shiXue.MPCost != 28 || shiXue.DamageMultiplier != 0.96 || shiXue.LifeStealChance != 86 || shiXue.LifeStealRatio != 0.7 {
		t.Fatalf("expected 嗜血斩 Lv3 captured profile to ignore stale Lv1 description, got %+v", shiXue)
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
			Name:        "奥义.雷魂斩",
			Level:       1,
			Type:        "oneE",
			Description: "f_s_奥义.雷魂斩^00ccff&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@24&4@<font color='#00cc00'>特殊发动条件:需要3格魂元</font><br>提升240%的物理伤害",
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
	if command := byID[CommandLeiHunZhan]; command.Label != "奥义.雷魂斩" || command.SourceType != "oneE" || command.Target != "enemy" || command.SourceActionLabel != "w8/thunderSoulAtk" || command.MPCost != 24 || command.DamageMultiplier != 3.4 {
		t.Fatalf("expected captured 奥义.雷魂斩 single-target command, got %+v", command)
	}
	if _, ok := byID["未抓包技能"]; ok {
		t.Fatalf("expected uncaptured skill to be omitted, got %+v", commands)
	}
	if byID[CommandStore].SourceActionLabel != "def" || byID[CommandEscape].SourceActionLabel != "escapeSuccess" {
		t.Fatalf("expected utility commands to use source labels, got %+v", commands)
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
	if len(runtime.PendingBuffInfos) != 1 || runtime.PendingBuffInfos[0].Name != "麻痹" || runtime.PendingBuffInfos[0].TargetHandle != "player_21424" || runtime.PendingBuffInfos[0].Round != 2 {
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
	if len(result.BuffInfos) != 1 || result.BuffInfos[0].Name != "麻痹" || result.BuffInfos[0].Round != 2 {
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
