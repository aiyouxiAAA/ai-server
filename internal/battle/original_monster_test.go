package battle

import (
	"ai-server/internal/session"
	"reflect"
	"testing"
)

func snowSpiderTestBattle(t *testing.T, mapID string) *Runtime {
	t.Helper()
	role := session.RoleSummary{RoleID: "snow-spider-player", DisplayName: "雪蛛测试", Voc: "新手", Level: 1}
	base := session.PlayerBaseData{Level: 1, HP: 185, MaxHP: 185, MP: 44, MaxMP: 44,
		RolePhysique: &session.RolePhysique{MaxHP: 185, MaxMP: 44, PhyAtk: 10, PhyDef: 10, Hit: 100, Dog: 0}}
	runtime, _, ok := NewWildBattle(role, base, StartRequest{MapID: mapID, MapName: "雪岭"})
	if !ok {
		t.Fatalf("missing original encounter map %s", mapID)
	}
	return runtime
}

func TestSnowSpiderDistributionAndRewards(t *testing.T) {
	for _, tc := range []struct {
		mapID                       string
		roll, count, hp, level, exp int
	}{
		{"6", 0, 1, 32, 1, 8}, {"6", 99, 1, 32, 1, 8}, {"8", 0, 1, 42, 2, 12},
		{"108", 74, 1, 52, 3, 16}, {"108", 75, 2, 52, 3, 32}, {"108", 99, 2, 52, 3, 32}, {"113", 99, 1, 70, 5, 24},
	} {
		restore := useSourceEncounterRoll(func(max int) int { return tc.roll % max })
		runtime := snowSpiderTestBattle(t, tc.mapID)
		restore()
		enemies := runtime.livingCells(CampEnemy)
		if len(enemies) != tc.count {
			t.Fatalf("%s/%d count=%d", tc.mapID, tc.roll, len(enemies))
		}
		seen := map[string]bool{}
		for _, enemy := range enemies {
			if enemy.DisplayURL != "original-monster/snow-spider" || enemy.MaxHP != tc.hp || enemy.Level != tc.level || seen[enemy.Handle] {
				t.Fatalf("wrong authored cell %+v", enemy)
			}
			seen[enemy.Handle] = true
		}
		exp, items := runtime.sourceBattleRewards(CampTeam, false)
		wantItems := []string{"丝x1"}
		if tc.count == 2 {
			wantItems[0] = "丝x2"
		}
		if exp != tc.exp || !reflect.DeepEqual(items, wantItems) {
			t.Fatalf("reward %d/%v", exp, items)
		}
		for _, escaped := range []bool{false, true} {
			exp, items := runtime.sourceBattleRewards(CampEnemy, escaped)
			if exp != 0 || len(items) != 0 {
				t.Fatal("losing/escaping granted reward")
			}
		}
		if exp, items := runtime.sourceBattleRewards(CampTeam, true); exp != 0 || len(items) != 0 {
			t.Fatal("escape reward")
		}
	}
	if configs := sourceEnemyConfigsForEncounter("2", 0); len(configs) != 0 {
		t.Fatal("village must remain safe")
	}
}

func TestSnowSpiderCadenceStaggersAndSingleImpact(t *testing.T) {
	defer useSourceEncounterRoll(func(max int) int { return max - 1 })()
	runtime := snowSpiderTestBattle(t, "108")
	enemies := runtime.livingCells(CampEnemy)
	player := runtime.firstLiving(CampTeam)
	for turn := 1; turn <= 7; turn++ {
		for slot, enemy := range enemies {
			command := runtime.enemyBattleCommand(enemy, player)
			wantSilk := turn >= 3+slot && (turn-3-slot)%3 == 0
			profile := runtime.battleCommandProfile(enemy, command)
			if (profile.SourceActionLabel == "snow-spider/silk") != wantSilk || profile.StatusName != "" || profile.CanFat || profile.MPCost != 0 {
				t.Fatalf("turn=%d slot=%d profile=%+v", turn, slot, profile)
			}
			if wantSilk {
				before := player.HP
				actions := runtime.resolveEnemyCommandActions(enemy, player, command)
				if len(actions) != 1 || actions[0].Damage != before-player.HP || actions[0].SourceActionLabel != "snow-spider/silk" {
					t.Fatalf("silk must resolve once %+v", actions)
				}
				if len(runtime.PendingBuffInfos) != 0 || profile.DamageMultiplier != 1.35 {
					t.Fatal("unexpected status/multiplier")
				}
			}
		}
	}
	// Killing the first spider must not rephase the second spider's personal clock.
	enemies[0].HP = 0
	if runtime.enemyBattleCommand(enemies[1], player) != CommandEnemyAttack {
		t.Fatal("slot rephased after death")
	}
}

func TestSnowSpiderFirstBattleCanBeWonWithBasicAttacks(t *testing.T) {
	defer useSourceEncounterRoll(func(max int) int { return 0 })()
	defer useSourceBattleAttackRoll(func(max int) int { return max / 2 })()
	runtime := snowSpiderTestBattle(t, "6")
	player, enemy := runtime.firstLiving(CampTeam), runtime.firstLiving(CampEnemy)
	player.Fat, enemy.Dog, player.Dog = 0, 0, 0
	for round := 0; round < 6 && enemy.HP > 0; round++ {
		runtime.resolveAttack(player, enemy, CommandNormalAttack)
		if enemy.HP > 0 {
			runtime.resolveEnemyCommandActions(enemy, player, runtime.enemyBattleCommand(enemy, player))
		}
	}
	if enemy.HP != 0 || player.HP < 100 {
		t.Fatalf("intro balance player=%d enemy=%d", player.HP, enemy.HP)
	}
	if runtime.originalMonsterTurns[enemy.Handle] < 2 {
		t.Fatal("first monster ended before basic teaching rhythm")
	}
}
