package battle

import (
	"ai-server/internal/profession"
	"ai-server/internal/session"
	"testing"
)

func bladeDancerTestRuntime(t *testing.T) *Runtime {
	t.Helper()
	skills := []session.RoleSkill{}
	for _, s := range profession.Skills {
		skills = append(skills, session.RoleSkill{Name: s.Name, Level: s.Level, Type: s.SourceType})
	}
	role := session.RoleSummary{RoleID: "blade-test", DisplayName: "刃舞者测试", Voc: "刃舞者", Level: 3, Skills: skills, SourceQuery: "human/human.swf?w8=42&sex=1&"}
	base := session.PlayerBaseData{PlayerID: "test", RoleID: role.RoleID, DisplayName: role.DisplayName, Voc: role.Voc, HP: 10000, MaxHP: 10000, MP: 72, MaxMP: 72,
		RolePhysique: &session.RolePhysique{Handle: role.RoleID, MaxHP: 10000, MaxMP: 72, PhyAtk: 100, PhyDef: 0, Hit: 100, Dog: 0, Fat: 0}}
	runtime, _, ok := NewWildBattle(role, base, StartRequest{MapID: "4", MapName: "test"})
	if !ok {
		t.Fatal("battle not created")
	}
	for i := range runtime.Cells {
		runtime.Cells[i].HP = 10000
		runtime.Cells[i].MaxHP = 10000
		runtime.Cells[i].Dog = 0
		runtime.Cells[i].Fat = 0
	}
	return runtime
}

func TestBladeDancerConfiguredDefaultsAndCast(t *testing.T) {
	for _, skill := range profession.Skills {
		if skill.Kind != "skill" {
			continue
		}
		t.Run(skill.Name, func(t *testing.T) {
			runtime := bladeDancerTestRuntime(t)
			actor := runtime.firstLiving(CampTeam)
			target := runtime.firstLiving(CampEnemy)
			if actor.ProfessionID != "blade-dancer" {
				t.Fatal("missing profession identity")
			}
			if !runtime.isBattleCommandAllowedForActor(actor.Handle, skill.ID) {
				t.Fatal("configured command rejected")
			}
			if runtime.isBattleCommandAllowedForActor(actor.Handle, CommandMiZhan) {
				t.Fatal("legacy command admitted")
			}
			before := actor.MP
			action := runtime.resolveAttack(actor, target, skill.ID)
			if action.SourceActionLabel != skill.ActionLabel || before-actor.MP != skill.MPCost || action.Damage <= 0 {
				t.Fatalf("wrong cast: %+v", action)
			}
		})
	}
	runtime := bladeDancerTestRuntime(t)
	for _, cmd := range runtime.commandDefinitionsForActor(runtime.RoleID) {
		if cmd.Label == "格挡" {
			t.Fatal("passive exposed as command")
		}
	}
}

func TestBladeDancerGuardChanceAndFreeCounter(t *testing.T) {
	previous := professionGuardRoll
	defer func() { professionGuardRoll = previous }()
	for _, roll := range []int{0, 49, 50, 99} {
		runtime := bladeDancerTestRuntime(t)
		actor := runtime.firstLiving(CampTeam)
		enemy := runtime.firstLiving(CampEnemy)
		actor.MP = 0
		professionGuardRoll = func(int) int { return roll }
		beforeHP, enemyHP := actor.HP, enemy.HP
		actions := runtime.resolveEnemyCommandActions(enemy, actor, CommandEnemyAttack)
		if roll < 50 {
			if len(actions) != 2 || !actions[0].TargetInDef || actions[0].Damage != 0 || actor.HP != beforeHP || actions[1].SourceActionLabel != "blade/execution" || enemy.HP >= enemyHP || actor.MP != 0 {
				t.Fatalf("guard/counter failed at %d: %+v", roll, actions)
			}
		} else if len(actions) != 1 || actions[0].TargetInDef || actor.HP >= beforeHP {
			t.Fatalf("non-guard failed at %d: %+v", roll, actions)
		}
	}
}

func TestBladeDancerInsufficientMPAndUnlearnedDoNotConsumeTurn(t *testing.T) {
	runtime := bladeDancerTestRuntime(t)
	actor := runtime.firstLiving(CampTeam)
	enemy := runtime.firstLiving(CampEnemy)
	actor.MP = 0
	start := runtime.pendingTeamStartCommands()[0]
	request := ActionRequest{BattleID: runtime.BattleID, ActorHandle: actor.Handle, TargetHandle: enemy.Handle, CommandID: "skill-blade-triple", Round: start.Round, Sequence: start.Sequence}
	result := runtime.ProcessAction(request)
	if result.ErrorCode != "insufficient_mp" || runtime.ConsumedSequence[start.Sequence] || enemy.HP != 10000 {
		t.Fatalf("MP failure mutated battle: %+v", result)
	}
	request.CommandID = "passive-blade-guard"
	result = runtime.ProcessAction(request)
	if result.ErrorCode != "unsupported_command" || runtime.ConsumedSequence[start.Sequence] {
		t.Fatal("passive cast accepted")
	}
	actor.MP = 72
	runtime.RoleSkillsByHandle[actor.Handle] = []session.RoleSkill{{Name: "普通攻击", Level: 1}}
	request.CommandID = "skill-blade-triple"
	if result := runtime.ProcessAction(request); result.ErrorCode != "unsupported_command" || runtime.ConsumedSequence[start.Sequence] {
		t.Fatal("unlearned skill accepted")
	}
}

func TestBladeDancerGuardAllTargetsKeepIndependentResults(t *testing.T) {
	previous := professionGuardRoll
	defer func() { professionGuardRoll = previous }()
	runtime := bladeDancerTestRuntime(t)
	first := runtime.firstLiving(CampTeam)
	second := *first
	second.Handle = "second-blade"
	runtime.Cells = append(runtime.Cells, second)
	runtime.RoleSkillsByHandle[second.Handle] = runtime.RoleSkills
	first = runtime.cellByHandle(first.Handle)
	index := 0
	professionGuardRoll = func(int) int {
		index++
		if index == 1 {
			return 99
		}
		return 0
	}
	action := runtime.resolveAllTargetAttack(runtime.firstLiving(CampEnemy), []*CellInfoPush{first, runtime.cellByHandle(second.Handle)}, CommandEnemyAttack)
	if len(action.TargetActionResults) != 2 || action.TargetActionResults[0].TargetInDef || !action.TargetActionResults[1].TargetInDef || len(runtime.guardCounters) != 1 {
		t.Fatalf("wrong grouped guard results: %+v", action)
	}
}
