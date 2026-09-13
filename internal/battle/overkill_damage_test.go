package battle

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestOverkillAllTargetsPreserveIndependentDamageInJSON(t *testing.T) {
	r := bladeDancerTestRuntime(t)
	actor := *r.firstLiving(CampTeam)
	enemy := *r.firstLiving(CampEnemy)
	r.Cells = []CellInfoPush{actor}
	for i := 0; i < 4; i++ {
		target := enemy
		target.Handle = fmt.Sprintf("overkill-%d", i)
		target.HP, target.MaxHP, target.Defense = 1, 70, i*100
		r.Cells = append(r.Cells, target)
	}
	actorPtr := r.firstLiving(CampTeam)
	targets := []*CellInfoPush{&r.Cells[1], &r.Cells[2], &r.Cells[3], &r.Cells[4]}
	action := r.resolveAllTargetAttack(actorPtr, targets, "skill-blade-guifeng")
	data, err := json.Marshal(action)
	if err != nil {
		t.Fatal(err)
	}
	var decoded ActionPush
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.TargetActionResults) != 4 {
		t.Fatal(string(data))
	}
	for i, result := range decoded.TargetActionResults {
		if result.Handle != targets[i].Handle || result.Damage <= 1 || targets[i].HP != 0 {
			t.Fatalf("lost full lethal damage: %s", data)
		}
		if i > 0 && result.Damage >= decoded.TargetActionResults[i-1].Damage {
			t.Fatalf("different defenses reused first target damage: %s", data)
		}
	}
	if action.Damage != decoded.TargetActionResults[0].Damage || actorPtr.MP != actor.MP-24 {
		t.Fatalf("changed existing single damage or MP contract: %s", data)
	}
}

func TestOverkillSingleDamageRetainsShieldAndBarrierAbsorption(t *testing.T) {
	r := bladeDancerTestRuntime(t)
	a, target := r.firstLiving(CampTeam), r.firstLiving(CampEnemy)
	target.HP, target.MaxHP, target.MP, target.MaxMP = 1, 70, 100, 100
	r.applyShield(a, target, 30, 3, "test")
	r.applyStatusEffect(target.Handle, BattleStatusEffect{Name: "法术屏障", Rounds: 3, DamageToMPPercent: 35})
	action := r.resolveAttack(a, target, "skill-blade-triple")
	if action.Damage <= 31 || action.TargetHP != 0 || target.Shield != 0 || target.MP >= 100 {
		t.Fatalf("lost full damage or absorption order: %+v %+v", action, target)
	}
}
