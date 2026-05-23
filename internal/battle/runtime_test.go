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

	if action.ActionName != "密斩" || action.SourceActionLabel != "manycut" {
		t.Fatalf("expected source manycut action for 密斩, got %+v", action)
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
