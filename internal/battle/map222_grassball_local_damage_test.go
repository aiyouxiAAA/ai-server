package battle

import "testing"

func TestMap222GrassballLocalHalfMagicDefensePolicy(t *testing.T) {
	defer useSourceBattleAttackRoll(func(int) int { return 2000 })()

	grassball, ok := sourceEnemyConfigForMap("222")
	if !ok {
		t.Fatal("expected map222 grassball config")
	}
	if grassball.Cell.Handle != "capture-222-glassss-lv42" || grassball.Cell.Attack != 301 {
		t.Fatalf("expected unit-roll calibrated map222 grassball attack 301, got %+v", grassball.Cell)
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
	if actor.DamageDefenseType != "magic" || action.Damage != 265 || action.TargetHP != 1740 {
		t.Fatalf("expected unmatched runtime to retain round(301 - 73 * 0.5) = 265 against 666, actor=%+v action=%+v target=%+v", actor, action, target)
	}
}

func TestMap222GrassballUsesPvpCaptureRollWithoutEnemyRange(t *testing.T) {
	grassball, ok := sourceEnemyConfigForMap("222")
	if !ok {
		t.Fatal("expected map222 grassball config")
	}
	runtime := &Runtime{
		BattleID:         "battle-map222-grassball-roll",
		MapID:            "222",
		Round:            1,
		nextSequence:     1,
		DefendingHandles: map[string]bool{},
	}
	actor := grassball.Cell.withBattleIDAndSlot(runtime.BattleID, 0)
	actor.Fat = 0

	defer useSourceBattleAttackRoll(func(int) int { return 0 })()
	minimumTarget := CellInfoPush{Handle: "acct-66666666-role-min", Camp: CampTeam, MaxHP: 2005, HP: 2005, MgcDefense: 73}
	minimum := runtime.resolveAttack(&actor, &minimumTarget, CommandEnemyAttack)
	if minimum.Damage != 234 || minimum.TargetHP != 1771 {
		t.Fatalf("expected 90%% local roll to deal round(301*0.9 - 73*0.5)=234, got %+v", minimum)
	}

	restoreMaximum := useSourceBattleAttackRoll(func(maxExclusive int) int { return maxExclusive - 1 })
	defer restoreMaximum()
	maximumTarget := CellInfoPush{Handle: "acct-66666666-role-max", Camp: CampTeam, MaxHP: 2005, HP: 2005, MgcDefense: 73}
	maximum := runtime.resolveAttack(&actor, &maximumTarget, CommandEnemyAttack)
	if maximum.Damage != 295 || maximum.TargetHP != 1710 {
		t.Fatalf("expected 110%% local roll to deal round(301*1.1 - 73*0.5)=295, got %+v", maximum)
	}
	if attack := (&Runtime{MapID: "223"}).captureBackedAttack(&actor, commandProfile{UseMagicAttack: true, SourceActionLabel: "nomalAtk"}); attack != 301 {
		t.Fatalf("expected unmatched wild map to retain fixed attack 301, got %.2f", attack)
	}
}
