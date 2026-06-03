package battle

import (
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
	if len(over.Result.Items) != 1 || over.Result.Items[0] != "肉" {
		t.Fatalf("expected captured map5 reward 肉 x1, got %+v", over.Result.Items)
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
	over := (&Runtime{BattleID: "battle-map49", MapID: "49", Round: 1}).buildOver(CampTeam)
	if over == nil {
		t.Fatal("expected OverBattle push")
	}
	if over.Result.ExpDelta != 610 {
		t.Fatalf("expected captured map49 robber reward exp 610, got %d", over.Result.ExpDelta)
	}
	if len(over.Result.Items) != 2 || over.Result.Items[0] != "铜钱x5" || over.Result.Items[1] != "盗贼的首级x1" {
		t.Fatalf("expected captured map49 reward 铜钱x5 and 盗贼的首级x1, got %+v", over.Result.Items)
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

	reward, ok := sourceBattleRewardConfigForMap("5")
	if !ok {
		t.Fatal("expected map5 battle reward config")
	}
	if reward.Status != "confirmed" || reward.ExpDelta != 0 || len(reward.Items) != 1 || reward.Items[0] != "肉" {
		t.Fatalf("expected confirmed captured map5 reward config, got %+v", reward)
	}

	map49Reward, ok := sourceBattleRewardConfigForMap("49")
	if !ok {
		t.Fatal("expected map49 battle reward config")
	}
	if map49Reward.Status != "confirmed" || map49Reward.ExpDelta != 610 || len(map49Reward.Items) != 2 {
		t.Fatalf("expected confirmed captured map49 reward config, got %+v", map49Reward)
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
	if runtime.powerFor("player_21424") != 0 {
		t.Fatalf("expected escape to clear stored power, got %d", runtime.powerFor("player_21424"))
	}

	playOver := runtime.ProcessPlayOver(PlayOverRequest{BattleID: "battle-escape"})
	if playOver.Over == nil || playOver.Over.Winner != CampEnemy || !playOver.Over.Result.Escaped {
		t.Fatalf("expected escaped over after play over, got %+v", playOver)
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
		t.Fatalf("expected critical to double base damage 47 to 94, got %+v", action)
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
		t.Fatalf("expected normal damage 47, got %+v", action)
	}
	if action.TargetActionState != "normal" || action.TargetActionStateCode != "0" {
		t.Fatalf("expected source normal target action state code 0, got %+v", action)
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
		t.Fatalf("expected captured 密斩 +40%% damage from 47 attack, got %+v", action)
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
			name:       "嗜血斩",
			level:      3,
			desc:       "f_s_嗜血斩^5BC46D&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@28&4@对敌人造成96%的物理伤害&0;并有86%机率将对敌人造成伤害的70%转换为气力</font>",
			label:      "w8/xyz2",
			mpCost:     28,
			multiplier: 0.96,
			chance:     86,
			ratio:      0.7,
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
		t.Fatalf("expected Lv3 多段斩 165 damage doubled by fat, got %+v", action)
	}
	if actor.MP != 182 || action.RefreshInfos[0].MP != 182 {
		t.Fatalf("expected 多段斩 Lv3 to consume MP 12, got actor=%+v action=%+v", actor, action)
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
	if !runtime.hasKuangBao("player_21424") || runtime.StatusEffects["player_21424"].KuangBaoRounds != 3 {
		t.Fatalf("expected 狂爆 3-round status, got %+v", runtime.StatusEffects)
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
		t.Fatalf("expected 狂爆 to double attack while keeping captured normal attack label, got action=%+v effects=%+v", attack, runtime.StatusEffects)
	}

	defenseProbe := runtime.resolveAttack(target, actor, CommandEnemyAttack)
	if defenseProbe.Damage != 30 {
		t.Fatalf("expected 狂爆 to reduce physical defense by 100%%, got %+v", defenseProbe)
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
		t.Fatalf("expected 92%% physical damage from 135 attack, got %+v", action)
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
		t.Fatalf("expected slideCut to use current physical damage path, got %+v", action)
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
		t.Fatalf("expected shadeCut to use current physical damage path, got %+v", action)
	}
	if actor.MP != 294 || len(action.RefreshInfos) != 2 || action.RefreshInfos[0].MP != 294 {
		t.Fatalf("expected shadeCut to consume captured MP cost 40, got actor=%+v action=%+v", actor, action)
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
