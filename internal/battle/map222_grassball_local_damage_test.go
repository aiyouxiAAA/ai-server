package battle

import "testing"

func TestMap222GrassballLocalFullMagicDefenseCalibration(t *testing.T) {
	grassball, ok := sourceEnemyConfigForMap("222")
	if !ok {
		t.Fatal("expected map222 grassball config")
	}
	if grassball.Cell.Handle != "capture-222-glassss-lv42" || grassball.Cell.Attack != 344 {
		t.Fatalf("expected calibrated map222 grassball attack 344, got %+v", grassball.Cell)
	}

	runtime := &Runtime{
		BattleID:         "battle-map222-grassball-local",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
	}
	actor := grassball.Cell.withBattleIDAndSlot(runtime.BattleID, 0)
	target := &CellInfoPush{
		Handle:     "acct-66666666-role-001",
		Camp:       CampTeam,
		MaxHP:      2005,
		HP:         2005,
		MgcDefense: 73,
	}

	action := runtime.resolveAttack(&actor, target, CommandEnemyAttack)
	if action.ActionName != "法术普通攻击" || action.SourceActionLabel != "nomalAtk" {
		t.Fatalf("expected grassball normal magic attack, got %+v", action)
	}
	if actor.DamageDefenseType != "magic" || action.Damage != 271 || action.TargetHP != 1734 {
		t.Fatalf("expected 344 - 73 = 271 against 666, actor=%+v action=%+v target=%+v", actor, action, target)
	}
}
