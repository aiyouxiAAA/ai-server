package battle

import (
	"ai-server/internal/classicdata"
	"ai-server/internal/profession"
	"ai-server/internal/session"
	"testing"
)

func TestRetiredSkillsRejectedWithoutTurnMutation(t *testing.T) {
	for _, command := range []string{CommandMiZhan, "密斩", CommandLeiHunZhan, CommandTouDu, CommandEnemyThunderstorm, "w8/thunderSoulAtk"} {
		runtime := bladeDancerTestRuntime(t)
		actor, target := runtime.firstLiving(CampTeam), runtime.firstLiving(CampEnemy)
		start := runtime.pendingTeamStartCommands()[0]
		beforeMP, beforeHP := actor.MP, target.HP
		result := runtime.ProcessAction(ActionRequest{BattleID: runtime.BattleID, ActorHandle: actor.Handle, TargetHandle: target.Handle,
			CommandID: command, Round: start.Round, Sequence: start.Sequence})
		if result.ErrorCode != "unsupported_command" || len(result.Actions) != 0 || actor.MP != beforeMP || target.HP != beforeHP || runtime.ConsumedSequence[start.Sequence] {
			t.Fatalf("retired command %q mutated battle: %+v", command, result)
		}
	}
}

func TestRetiredSkillsAbsentFromActiveCatalogAndMonsterAI(t *testing.T) {
	for _, row := range classicdata.MustRows(classicdata.TableSkill) {
		if row["kind"] != "skill" {
			continue
		}
		if skill, ok := profession.BySkillID(row["skill_id"]); !ok || skill.Kind != "skill" {
			t.Fatalf("retired player skill in compiled catalog: %+v", row)
		}
	}
	for _, row := range classicdata.MustRows(classicdata.TableMonsterSkill) {
		behavior, ok := originalMonsterBehaviors[row["display_url"]]
		if !ok || row["command_id"] != behavior.SkillCommand {
			t.Fatalf("retired monster skill in compiled catalog: %+v", row)
		}
	}
	commands := CommandDefinitionsForSkills([]session.RoleSkill{{Name: "密斩", Level: 1}, {Name: "雷击", Level: 3}, {Name: "奥义", Level: 1}})
	for _, command := range commands {
		if command.Kind == "skill" && command.ID != CommandNormalAttack && command.ID != "skill-blade-ultimate" {
			t.Fatalf("retired learned entry surfaced: %+v", command)
		}
	}
	runtime := bladeDancerTestRuntime(t)
	for _, row := range classicdata.MustRows(classicdata.TableMonster) {
		if _, authored := originalMonsterBehaviors[row["display_url"]]; authored {
			continue
		}
		enemy := &CellInfoPush{Handle: row["handle"], DisplayURL: row["display_url"], Camp: CampEnemy, HP: 1, MaxHP: 100, MP: 9999}
		for i := 0; i < 10; i++ {
			if command := runtime.enemyBattleCommand(enemy, runtime.firstLiving(CampTeam)); command != CommandEnemyAttack {
				t.Fatalf("retired AI on %s: %s", enemy.DisplayURL, command)
			}
		}
	}
}
