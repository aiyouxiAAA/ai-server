package battle

import (
	_ "embed"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
)

const shieldStatusName = "护盾"

//go:embed config/original-shield-skill.csv
var originalShieldSkillCSV string

type shieldSkillConfig struct {
	CommandID, Name, DisplayURL, SourceActionLabel string
	MaxHPPercent, Rounds                           int
}

var tabletWard = loadShieldSkillConfig()

func loadShieldSkillConfig() shieldSkillConfig {
	rows, err := csv.NewReader(strings.NewReader(originalShieldSkillCSV)).ReadAll()
	if err != nil || len(rows) != 2 || len(rows[1]) != 6 {
		panic("invalid original shield skill config")
	}
	r := rows[1]
	percent, e1 := strconv.Atoi(r[4])
	rounds, e2 := strconv.Atoi(r[5])
	if e1 != nil || e2 != nil || percent <= 0 || rounds <= 0 {
		panic("invalid original shield skill values")
	}
	return shieldSkillConfig{r[0], r[1], r[2], r[3], percent, rounds}
}

// Called within the existing Runtime action lock. Reapplication replaces capacity
// and refreshes duration; one shield pool cannot accumulate without a bound.
func (runtime *Runtime) applyShield(source, target *CellInfoPush, amount, rounds int, skill string) bool {
	if runtime == nil || source == nil || target == nil || target.HP <= 0 || amount <= 0 || rounds <= 0 {
		return false
	}
	target.Shield = amount
	effect := BattleStatusEffect{Name: shieldStatusName, Display: "original-shield", Description: fmt.Sprintf("优先吸收伤害，当前护盾%d点。", amount), Rounds: rounds, SourceHandle: source.Handle, SourceSkill: skill}
	runtime.applyStatusEffect(target.Handle, effect)
	runtime.PendingBuffInfos = append(runtime.PendingBuffInfos, runtime.resolveStatusBuffInfo(source, target, effect))
	return true
}

func (runtime *Runtime) absorbShieldDamage(target *CellInfoPush, damage int) int {
	if target == nil || target.HP <= 0 || damage <= 0 || target.Shield <= 0 {
		return 0
	}
	effect, exists := runtime.StatusEffects[target.Handle].Effects[shieldStatusName]
	if !exists || effect.Rounds <= 0 {
		target.Shield = 0
		return 0
	}
	absorbed := damage
	if target.Shield < absorbed {
		absorbed = target.Shield
	}
	target.Shield -= absorbed
	if target.Shield == 0 {
		runtime.clearStatusEffect(target.Handle, shieldStatusName)
	}
	return absorbed
}

func (runtime *Runtime) resolveTabletWardAction(actor *CellInfoPush) []ActionPush {
	if actor == nil || actor.DisplayURL != tabletWard.DisplayURL || actor.HP <= 0 {
		return nil
	}
	amount := maxInt(1, actor.MaxHP*tabletWard.MaxHPPercent/100)
	if !runtime.applyShield(actor, actor, amount, tabletWard.Rounds, tabletWard.Name) {
		return nil
	}
	action := runtime.resolveSelfAction(actor, tabletWard.CommandID, tabletWard.Name, tabletWard.SourceActionLabel)
	action.TargetActionState = "none"
	action.TargetActionStateCode = "3"
	return []ActionPush{action}
}
