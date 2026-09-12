package battle

import (
	"ai-server/internal/profession"
	"math/rand"
)

var professionGuardRoll = rand.Intn

type guardCounter struct{ attacker, defender, skillID string }

func originalSkillProfile(skill profession.Skill) commandProfile {
	return commandProfile{ActionName: skill.Name, SourceType: skill.SourceType, SourceActionLabel: skill.ActionLabel,
		MPCost: skill.MPCost, DamageMultiplier: skill.DamageMultiplier, DefenseType: "physical", CanDodge: true, CanFat: true}
}

func (runtime *Runtime) tryProfessionGuard(actor, target *CellInfoPush, commandID string, consumeMP bool, profile commandProfile) (ActionPush, bool) {
	if actor.Camp != CampEnemy || target.Camp != CampTeam || target.HP <= 0 || target.ProfessionID == "" || profile.DamageMultiplier <= 0 {
		return ActionPush{}, false
	}
	for _, passive := range profession.Skills {
		if passive.ProfessionID != target.ProfessionID || passive.Kind != "passive" || passive.CounterSkillID == "" || !runtime.hasRoleSkillForActor(target.Handle, passive.Name) {
			continue
		}
		if professionGuardRoll(100) >= passive.TriggerChance {
			return ActionPush{}, false
		}
		if consumeMP && profile.MPCost > 0 {
			actor.MP = maxInt(0, actor.MP-profile.MPCost)
		}
		runtime.guardCounters = append(runtime.guardCounters, guardCounter{actor.Handle, target.Handle, passive.CounterSkillID})
		return ActionPush{BattleID: runtime.BattleID, ActorHandle: actor.Handle, TargetHandle: target.Handle, CommandID: commandID,
			ActionName: profile.ActionName, SourceMode: sourceBattleActionMode(profile.SourceType), SourceActionLabel: profile.SourceActionLabel,
			TargetInDef: true, TargetActionState: "normal", TargetActionStateCode: "0", Damage: 0, TargetHP: target.HP, TargetMP: target.MP,
			RefreshInfos: []CellInfoPush{*actor, *target}, Round: runtime.Round, Sequence: runtime.currentActionSequence()}, true
	}
	return ActionPush{}, false
}

func (runtime *Runtime) resolveEnemyCommandActions(enemy, target *CellInfoPush, commandID string) []ActionPush {
	runtime.guardCounters = nil
	actions := runtime.resolveEnemyCommandActionsBase(enemy, target, commandID)
	counters := runtime.guardCounters
	runtime.guardCounters = nil
	for _, counter := range counters {
		defender := runtime.cellByHandle(counter.defender)
		attacker := runtime.cellByHandle(counter.attacker)
		if defender == nil || attacker == nil || defender.HP <= 0 || attacker.HP <= 0 {
			continue
		}
		// Free retaliation is part of this enemy action, never a player turn or another guard trigger.
		actions = append(actions, runtime.resolveAttackWithMPCost(defender, attacker, counter.skillID, false))
	}
	return actions
}
