package battle

import (
	"testing"

	"ai-server/internal/session"
)

func Test666JuanYeShiAppliesMultiplierBeforeTargetDefense(t *testing.T) {
	runtime := &Runtime{
		BattleID:         "battle-666-juan-ye-shi",
		DefendingHandles: map[string]bool{},
		RoleSkills:       []session.RoleSkill{{Name: "卷叶式", Level: 5, Type: "oneE"}},
		Cells: []CellInfoPush{
			{Handle: "acct-66666666-role-001", Camp: CampTeam, HP: 2005, MaxHP: 2005, MP: 455, Attack: 392, Hit: 391},
			{Handle: "target", Camp: CampEnemy, HP: 1000, MaxHP: 1000, Defense: 14},
		},
	}

	action := runtime.resolveAttack(
		runtime.cellByHandle("acct-66666666-role-001"),
		runtime.cellByHandle("target"),
		CommandJuanYeShi,
	)

	if action.SourceActionLabel != "w7/jys2" || action.Damage != 672 || action.TargetHP != 328 {
		t.Fatalf("expected round(392 * 1.75) - 14 = 672, got %+v", action)
	}
	if actor := runtime.cellByHandle("acct-66666666-role-001"); actor.MP != 439 {
		t.Fatalf("expected Lv5 卷叶式 to consume 16 MP, actor=%+v", actor)
	}
}
