package battle

import (
	"encoding/json"
	"strings"
	"testing"
)

func shieldFixture() (*Runtime, *CellInfoPush) {
	r := &Runtime{BattleID: "shield-test", Cells: []CellInfoPush{{Handle: "stone", Camp: CampEnemy, DisplayURL: tabletWard.DisplayURL, HP: 1000, MaxHP: 1000, MP: 100, MaxMP: 100}}}
	return r, &r.Cells[0]
}

func TestShieldWardSelfThirtyPercentThreeRounds(t *testing.T) {
	r, a := shieldFixture()
	actions := r.resolveEnemyCommandActionsBase(a, a, tabletWard.CommandID)
	if len(actions) != 1 || a.Shield != 300 || a.HP != 1000 || actions[0].TargetHandle != a.Handle || actions[0].Damage != 0 || actions[0].RefreshInfos[0].Shield != 300 {
		t.Fatalf("incorrect self ward: %+v %+v", a, actions)
	}
	if e := r.StatusEffects[a.Handle].Effects[shieldStatusName]; e.Rounds != 3 {
		t.Fatal(e)
	}
	if len(r.PendingBuffInfos) != 1 || r.PendingBuffInfos[0].Shield != 300 {
		t.Fatal(r.PendingBuffInfos)
	}
}

func TestShieldAbsorbThenOverflowAndClear(t *testing.T) {
	r, a := shieldFixture()
	r.applyShield(a, a, 300, 3, "test")
	if hp, mp := r.applyTargetHPDamage(a, 200); hp != 0 || mp != 0 || a.Shield != 100 || a.HP != 1000 {
		t.Fatalf("first hit %+v %d %d", a, hp, mp)
	}
	if hp, _ := r.applyTargetHPDamage(a, 180); hp != 80 || a.Shield != 0 || a.HP != 920 {
		t.Fatalf("overflow %+v %d", a, hp)
	}
	if len(r.PendingClearBuffInfos) != 1 || len(r.StatusEffects[a.Handle].Effects) != 0 {
		t.Fatalf("shield not removed %+v", r.StatusEffects)
	}
	encoded, _ := json.Marshal(*a)
	if !strings.Contains(string(encoded), `"shield":0`) {
		t.Fatal(string(encoded))
	}
}

func TestShieldExpiryReplacementAndExplicitClear(t *testing.T) {
	r, a := shieldFixture()
	r.applyShield(a, a, 300, 3, "test")
	r.applyTargetHPDamage(a, 100)
	r.applyShield(a, a, 300, 3, "test")
	if a.Shield != 300 {
		t.Fatal("must replace, not stack", a.Shield)
	}
	for i := 0; i < 2; i++ {
		r.resolveStatusStartActions(a)
		if a.Shield != 300 {
			t.Fatal("expired early")
		}
	}
	r.resolveStatusStartActions(a)
	if a.Shield != 0 {
		t.Fatal("expired shield retained")
	}
	r.applyShield(a, a, 300, 3, "test")
	r.clearStatusEffect(a.Handle, shieldStatusName)
	if a.Shield != 0 {
		t.Fatal("clear retained shield")
	}
}

func TestShieldBeforeMagicBarrierAndNoPowerFromAbsorbedHP(t *testing.T) {
	r, a := shieldFixture()
	r.applyShield(a, a, 100, 3, "test")
	r.applyStatusEffect(a.Handle, BattleStatusEffect{Name: "法术屏障", Rounds: 3, DamageToMPPercent: 35})
	hp, mp := r.applyTargetHPDamage(a, 200)
	if hp != 65 || mp != 35 || a.Shield != 0 || a.MP != 65 {
		t.Fatalf("order %d %d %+v", hp, mp, a)
	}
	r.applyShield(a, a, 100, 3, "test")
	hp, _ = r.applyTargetHPDamage(a, 100)
	if hp != 0 {
		t.Fatal("absorbed hit returned HP loss")
	}
}

func TestShieldInvalidAndZeroDamage(t *testing.T) {
	r, a := shieldFixture()
	if r.applyShield(a, a, 0, 3, "test") || r.applyShield(a, a, 10, 0, "test") {
		t.Fatal("invalid shield accepted")
	}
	r.applyShield(a, a, 100, 3, "test")
	r.applyTargetHPDamage(a, 0)
	if a.Shield != 100 {
		t.Fatal("zero damage consumed shield")
	}
	a.HP = 0
	if r.applyShield(a, a, 100, 3, "test") {
		t.Fatal("dead target accepted")
	}
}
