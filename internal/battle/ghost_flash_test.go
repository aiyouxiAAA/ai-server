package battle

import (
	"fmt"
	"testing"
)

func TestGhostFlashFourTargetsAndSingleMPCost(t *testing.T) {
	r := bladeDancerTestRuntime(t)
	template := *r.firstLiving(CampEnemy)
	team := *r.firstLiving(CampTeam)
	r.Cells = []CellInfoPush{team}
	for i := 0; i < 4; i++ {
		enemy := template
		enemy.Handle = fmt.Sprintf("ghost-target-%d", i)
		r.Cells = append(r.Cells, enemy)
	}
	actor := r.firstLiving(CampTeam)
	targets := []*CellInfoPush{}
	for i := 1; i < len(r.Cells); i++ {
		targets = append(targets, &r.Cells[i])
	}
	before := actor.MP
	action := r.resolveAllTargetAttack(actor, targets, "skill-blade-ghost")
	if action.SourceActionLabel != "blade/ghost" || action.TargetHandle != "all" || len(action.TargetActionResults) != 4 || actor.MP != before-18 {
		t.Fatalf("incorrect ghost group: %+v", action)
	}
	for _, enemy := range targets {
		if enemy.HP >= 10000 || enemy.HP <= 0 {
			t.Fatalf("target not independently damaged: %+v", enemy)
		}
	}
	if r.battleCommandProfile(actor, "skill-blade-ghost").SourceType != "all" {
		t.Fatal("ghost is not group targeted")
	}
}
