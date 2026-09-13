package battle

import (
	"fmt"
	"testing"
)

func TestGuifengReplacesUltimateWithFourTargetAttack(t *testing.T) {
	r := bladeDancerTestRuntime(t)
	template := *r.firstLiving(CampEnemy)
	team := *r.firstLiving(CampTeam)
	r.Cells = []CellInfoPush{team}
	for i := 0; i < 4; i++ {
		enemy := template
		enemy.Handle = fmt.Sprintf("guifeng-%d", i)
		r.Cells = append(r.Cells, enemy)
	}
	actor := r.firstLiving(CampTeam)
	if r.isBattleCommandAllowedForActor(actor.Handle, "skill-blade-ultimate") {
		t.Fatal("retired ultimate still accepted")
	}
	profile := r.battleCommandProfile(actor, "skill-blade-guifeng")
	if profile.SourceType != "all" || profile.ActionName != "断月·归锋" || profile.MPCost != 24 || profile.DamageMultiplier != 3.4 {
		t.Fatalf("wrong replacement: %+v", profile)
	}
	start := r.pendingTeamStartCommands()[0]
	before := actor.MP
	result := r.ProcessAction(ActionRequest{BattleID: r.BattleID, ActorHandle: actor.Handle, TargetHandle: r.Cells[1].Handle, CommandID: "skill-blade-guifeng", Round: start.Round, Sequence: start.Sequence})
	if result.ErrorCode != "" {
		t.Fatalf("group command failed: %+v", result)
	}
	if actor.MP != before-24 {
		t.Fatalf("MP charged per target: %d", actor.MP)
	}
	for i := 1; i < len(r.Cells); i++ {
		if r.Cells[i].HP >= 10000 {
			t.Fatalf("target missed: %+v", r.Cells[i])
		}
	}
}

func TestGuifengInsufficientMPDoesNotConsumeTurn(t *testing.T) {
	r := bladeDancerTestRuntime(t)
	actor := r.firstLiving(CampTeam)
	enemy := r.firstLiving(CampEnemy)
	actor.MP = 23
	start := r.pendingTeamStartCommands()[0]
	result := r.ProcessAction(ActionRequest{BattleID: r.BattleID, ActorHandle: actor.Handle, TargetHandle: enemy.Handle, CommandID: "skill-blade-guifeng", Round: start.Round, Sequence: start.Sequence})
	if result.ErrorCode != "insufficient_mp" || actor.MP != 23 || enemy.HP != 10000 || r.ConsumedSequence[start.Sequence] {
		t.Fatalf("failure mutated battle: %+v", result)
	}
}
