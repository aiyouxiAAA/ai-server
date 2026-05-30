package battle

import (
	"testing"

	"ai-server/internal/session"
)

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
	if enemy.Cell.Name != "盗贼" || enemy.Cell.DisplayURL != "monstermap/robber.swf" || enemy.Cell.MaxHP != 445 || enemy.Cell.MaxMP != 194 {
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
	}{
		{mapID: "11", mapName: "涧庭道_1", enemyName: "蟾蜍", displayURL: "monstermap/toad.swf", level: 1, maxHP: 15, maxMP: 9},
		{mapID: "37", mapName: "平原_6", enemyName: "花妖", displayURL: "monstermap/huayao.swf", level: 10, maxHP: 230, maxMP: 150},
		{mapID: "49", mapName: "平原_15", enemyName: "盗贼", displayURL: "monstermap/robber.swf", level: 15, maxHP: 445, maxMP: 194},
		{mapID: "52", mapName: "平原_18", enemyName: "盗贼", displayURL: "monstermap/robber.swf", level: 18, maxHP: 470, maxMP: 220},
	}

	for _, testCase := range cases {
		_, bundle, ok := NewWildBattle(role, playerBase, StartRequest{MapID: testCase.mapID, MapName: testCase.mapName})
		if !ok || len(bundle.Cells) != 2 {
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
	if got := battleCriticalChancePercent(100, 50); got != 100 {
		t.Fatalf("expected fat100 to stay forced critical for deterministic tests, got %d", got)
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
		Handle:  "enemy_robber",
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
