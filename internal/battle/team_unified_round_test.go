package battle

import (
	"testing"

	"ai-server/internal/session"
)

func TestTeamQiYuStatusTicksEvenWhenCasterIsNotNextActor(t *testing.T) {
	runtime := &Runtime{
		BattleID:             "battle-team-qiyu",
		Round:                1,
		Phase:                PhaseCommand,
		nextSequence:         10,
		ConsumedSequence:     map[int]bool{},
		PendingTeamActions:   map[string]bool{"p1": true, "p2": true, "p3": true, "p4": true},
		PendingTeamSequences: map[string]int{"p1": 10, "p2": 11, "p3": 12, "p4": 13},
		StatusEffects:        map[string]BattleStatusEffects{},
		DefendingHandles:     map[string]bool{},
		StoredPower:          map[string]int{},
		Cells: []CellInfoPush{
			{BattleID: "battle-team-qiyu", Handle: "p1", Camp: CampTeam, HP: 500, MaxHP: 2100, MP: 100, MaxMP: 100, Name: "p1"},
			{BattleID: "battle-team-qiyu", Handle: "p2", Camp: CampTeam, HP: 500, MaxHP: 2100, MP: 100, MaxMP: 100, Name: "p2"},
			{BattleID: "battle-team-qiyu", Handle: "p3", Camp: CampTeam, HP: 500, MaxHP: 2100, MP: 100, MaxMP: 100, Name: "p3"},
			{BattleID: "battle-team-qiyu", Handle: "p4", Camp: CampTeam, HP: 500, MaxHP: 2100, MP: 100, MaxMP: 100, Name: "p4"},
			// Keep the enemy from killing anyone so the team phase can reopen.
			{BattleID: "battle-team-qiyu", Handle: "e1", Camp: CampEnemy, HP: 1000, MaxHP: 1000, Attack: 1, Hit: 1, Name: "e1"},
		},
	}

	p1 := runtime.cellByHandle("p1")
	buff := runtime.applyQiYuStatusEffect(p1)
	if buff.Name != "气疗" || buff.Round != 3 || p1.HP != 500 {
		t.Fatalf("expected 气愈式 to apply 气疗 without immediate heal, buff=%+v hp=%d", buff, p1.HP)
	}

	// Teammates act first; p1 submits last so nextActor after the enemy phase is p2.
	for _, handle := range []string{"p2", "p3", "p4"} {
		actor := runtime.cellByHandle(handle)
		result := runtime.resolveEnemyTurnAndNextCommand(actor, []ActionPush{
			runtime.resolveSelfAction(actor, CommandDefense, "防御", "def"),
		})
		if result.ErrorCode != "" {
			t.Fatalf("expected teammate %s action to resolve, got %+v", handle, result)
		}
		for _, action := range result.Actions {
			if action.ActionName == "气疗" {
				t.Fatalf("expected 气疗 only after the last team command, got early action from %s: %+v", handle, action)
			}
		}
	}

	before := p1.HP
	result := runtime.resolveEnemyTurnAndNextCommand(p1, []ActionPush{
		runtime.resolveSelfAction(p1, CommandDefense, "防御", "def"),
	})
	qiLiaoCount := 0
	for _, action := range result.Actions {
		if action.ActionName == "气疗" && action.ActorHandle == "p1" {
			qiLiaoCount += 1
		}
	}
	if qiLiaoCount != 1 {
		t.Fatalf("expected one 气疗 tick for the caster even when nextActor is a teammate, actions=%+v", result.Actions)
	}
	if p1.HP <= before {
		t.Fatalf("expected 气疗 to heal the caster, hp %d -> %d", before, p1.HP)
	}
	effect, ok := runtime.StatusEffects["p1"].Effects["气疗"]
	if !ok || effect.Rounds != 2 {
		t.Fatalf("expected 气疗 remaining rounds to drop from 3 to 2, ok=%v effect=%+v", ok, effect)
	}
	if len(runtime.PendingTeamActions) != 4 || len(runtime.PendingStarts) != 4 {
		t.Fatalf("expected the next concurrent team command windows to reopen, pending=%+v starts=%+v", runtime.PendingTeamActions, runtime.PendingStarts)
	}
}

func TestConcurrentTeamRoundUnifiesStatusHotbloodStunAndConfusion(t *testing.T) {
	runtime := &Runtime{
		BattleID:             "battle-team-unified",
		Round:                1,
		Phase:                PhaseCommand,
		nextSequence:         10,
		ConsumedSequence:     map[int]bool{},
		PendingTeamActions:   map[string]bool{"p1": true, "p2": true, "p3": true, "p4": true},
		PendingTeamSequences: map[string]int{"p1": 10, "p2": 11, "p3": 12, "p4": 13},
		StatusEffects:        map[string]BattleStatusEffects{},
		DefendingHandles:     map[string]bool{},
		StoredPower:          map[string]int{},
		PendingConfusion:     map[string]bool{},
		PendingSkillSeal:     map[string]bool{},
		Cells: []CellInfoPush{
			{BattleID: "battle-team-unified", Handle: "p1", Camp: CampTeam, HP: 800, MaxHP: 1000, MP: 100, MaxMP: 100, Name: "p1"},
			{BattleID: "battle-team-unified", Handle: "p2", Camp: CampTeam, HP: 800, MaxHP: 1000, MP: 100, MaxMP: 100, Name: "p2"},
			{BattleID: "battle-team-unified", Handle: "p3", Camp: CampTeam, HP: 800, MaxHP: 1000, MP: 100, MaxMP: 100, Name: "p3"},
			{BattleID: "battle-team-unified", Handle: "p4", Camp: CampTeam, HP: 800, MaxHP: 1000, MP: 100, MaxMP: 100, Name: "p4"},
			{BattleID: "battle-team-unified", Handle: "e1", Camp: CampEnemy, HP: 5000, MaxHP: 5000, Attack: 1, Hit: 1, Name: "e1"},
		},
	}

	runtime.applyKuangBao("p1")
	runtime.applyKuangBao("p3")
	runtime.applyQiYuStatusEffect(runtime.cellByHandle("p1"))
	runtime.applyStatusEffect("p2", BattleStatusEffect{Name: "眩晕", Display: "9.png", Description: "眩晕无法行动", Rounds: 2, SkipTurn: true})
	runtime.applyStatusEffect("p3", BattleStatusEffect{Name: "混乱", Display: "20.png", Description: "混乱", Rounds: 2})

	for _, handle := range []string{"p1", "p2", "p3"} {
		actor := runtime.cellByHandle(handle)
		runtime.resolveEnemyTurnAndNextCommand(actor, []ActionPush{
			runtime.resolveSelfAction(actor, CommandDefense, "防御", "def"),
		})
	}
	result := runtime.resolveEnemyTurnAndNextCommand(runtime.cellByHandle("p4"), []ActionPush{
		runtime.resolveSelfAction(runtime.cellByHandle("p4"), CommandDefense, "防御", "def"),
	})

	qiLiao := 0
	stun := 0
	confusionStatus := 0
	confusionAttack := 0
	for _, action := range result.Actions {
		switch {
		case action.ActionName == "气疗" && action.ActorHandle == "p1":
			qiLiao++
		case action.ActionName == "眩晕" && action.ActorHandle == "p2":
			stun++
		case action.ActionName == "混乱" && action.ActorHandle == "p3":
			confusionStatus++
		case action.ActionName == "普通攻击" && action.ActorHandle == "p3":
			confusionAttack++
		}
	}
	if qiLiao != 1 || stun != 1 || confusionStatus != 1 || confusionAttack != 1 {
		t.Fatalf("expected unified round-start 气疗/眩晕/混乱+auto-attack, got qiLiao=%d stun=%d confusionStatus=%d confusionAttack=%d actions=%+v", qiLiao, stun, confusionStatus, confusionAttack, result.Actions)
	}
	if runtime.StatusEffects["p1"].KuangBaoRounds != 2 || runtime.StatusEffects["p3"].KuangBaoRounds != 2 {
		t.Fatalf("expected every living teammate's 热血 to advance, got p1=%d p3=%d", runtime.StatusEffects["p1"].KuangBaoRounds, runtime.StatusEffects["p3"].KuangBaoRounds)
	}
	if runtime.PendingTeamActions["p2"] || runtime.PendingTeamActions["p3"] {
		t.Fatalf("expected stunned/confused members to lose their command windows, pending=%+v", runtime.PendingTeamActions)
	}
	if !runtime.PendingTeamActions["p1"] || !runtime.PendingTeamActions["p4"] || len(runtime.PendingTeamActions) != 2 {
		t.Fatalf("expected only free members to reopen command windows, pending=%+v", runtime.PendingTeamActions)
	}
	if runtime.PendingConfusion["p3"] {
		t.Fatalf("expected confused member auto-attack to consume pending confusion, got %+v", runtime.PendingConfusion)
	}
	starts := map[string]bool{}
	for _, start := range runtime.PendingStarts {
		starts[start.ActorHandle] = true
	}
	if starts["p2"] || starts["p3"] || !starts["p1"] || !starts["p4"] {
		t.Fatalf("expected pending starts only for free members, starts=%+v", starts)
	}
}

func TestTeamCollapseToOneFreeMemberOpensSoloWindowWithoutExtraEnemyPhase(t *testing.T) {
	runtime := &Runtime{
		BattleID:             "battle-team-collapse",
		Round:                1,
		Phase:                PhaseCommand,
		nextSequence:         10,
		ConsumedSequence:     map[int]bool{},
		PendingTeamActions:   map[string]bool{"p1": true, "p2": true},
		PendingTeamSequences: map[string]int{"p1": 10, "p2": 11},
		StatusEffects:        map[string]BattleStatusEffects{},
		DefendingHandles:     map[string]bool{},
		StoredPower:          map[string]int{},
		Cells: []CellInfoPush{
			{BattleID: "battle-team-collapse", Handle: "p1", Camp: CampTeam, HP: 1, MaxHP: 1000, MP: 100, MaxMP: 100, Name: "p1"},
			{BattleID: "battle-team-collapse", Handle: "p2", Camp: CampTeam, HP: 800, MaxHP: 1000, MP: 100, MaxMP: 100, Name: "p2"},
			{BattleID: "battle-team-collapse", Handle: "e1", Camp: CampEnemy, HP: 5000, MaxHP: 5000, Attack: 0, Hit: 0, Name: "e1"},
		},
	}

	// Lethal poison on p1 so concurrent status start kills one member and leaves p2 alone.
	runtime.applyStatusEffect("p1", BattleStatusEffect{
		Name:           "中毒",
		Display:        "8.png",
		Description:    "poison",
		Rounds:         2,
		SourceHandle:   "e1",
		SourceSkill:    "投毒",
		SourceAttack:   10000,
		TickMinPercent: 100,
		TickMaxPercent: 100,
	})

	// p1 acts first, p2 last so enemy phase starts with concurrentTeamRound=true and living>=2.
	if result := runtime.resolveEnemyTurnAndNextCommand(runtime.cellByHandle("p1"), []ActionPush{
		runtime.resolveSelfAction(runtime.cellByHandle("p1"), CommandDefense, "防御", "def"),
	}); result.ErrorCode != "" {
		t.Fatalf("expected first teammate action to resolve, got %+v", result)
	}

	beforeP2 := runtime.cellByHandle("p2").HP
	result := runtime.resolveEnemyTurnAndNextCommand(runtime.cellByHandle("p2"), []ActionPush{
		runtime.resolveSelfAction(runtime.cellByHandle("p2"), CommandDefense, "防御", "def"),
	})

	enemyAtk := 0
	poison := 0
	for _, action := range result.Actions {
		if action.ActorHandle == "e1" {
			enemyAtk++
		}
		if action.ActionName == "中毒" {
			poison++
		}
	}
	if poison != 1 {
		t.Fatalf("expected one poison tick that kills p1, poison=%d actions=%+v", poison, result.Actions)
	}
	if enemyAtk != 1 {
		t.Fatalf("expected exactly one enemy phase after team commands, enemyAtk=%d actions=%+v", enemyAtk, result.Actions)
	}
	if runtime.cellByHandle("p1").HP > 0 {
		t.Fatalf("expected p1 dead after poison, hp=%d", runtime.cellByHandle("p1").HP)
	}
	if len(runtime.livingCells(CampTeam)) != 1 {
		t.Fatalf("expected only one living teammate, living=%d", len(runtime.livingCells(CampTeam)))
	}
	// Collapse stays on the concurrent machine: N=1 PendingTeamActions + PendingStarts.
	if len(runtime.PendingTeamActions) != 1 || !runtime.PendingTeamActions["p2"] {
		t.Fatalf("expected concurrent N=1 pending action for survivor p2, pending=%+v", runtime.PendingTeamActions)
	}
	if len(runtime.PendingStarts) != 1 || runtime.PendingStarts[0].ActorHandle != "p2" {
		t.Fatalf("expected one concurrent PendingStarts entry for survivor p2, starts=%+v", runtime.PendingStarts)
	}
	if runtime.cellByHandle("p2").HP != beforeP2-1 {
		t.Fatalf("expected survivor to take only one enemy hit, hp %d -> %d", beforeP2, runtime.cellByHandle("p2").HP)
	}

	playOver := runtime.ProcessPlayOver(PlayOverRequest{BattleID: runtime.BattleID})
	if playOver.ErrorCode != "" || len(playOver.StartCommands) != 1 || playOver.StartCommands[0].ActorHandle != "p2" {
		t.Fatalf("expected ProcessPlayOver to deliver one concurrent startCommand for p2, got %+v", playOver)
	}
	// N=1 must NOT also mirror StartCommand — writers would double-push 3003.
	if playOver.StartCommand != nil {
		t.Fatalf("expected N=1 to deliver only StartCommands (no StartCommand mirror), got %+v", playOver.StartCommand)
	}
}

func TestTeamAllStunnedAutoContinuesWithoutCommandWindows(t *testing.T) {
	runtime := &Runtime{
		BattleID:             "battle-team-all-stun",
		Round:                1,
		Phase:                PhaseCommand,
		nextSequence:         10,
		ConsumedSequence:     map[int]bool{},
		PendingTeamActions:   map[string]bool{"p1": true, "p2": true},
		PendingTeamSequences: map[string]int{"p1": 10, "p2": 11},
		StatusEffects:        map[string]BattleStatusEffects{},
		DefendingHandles:     map[string]bool{},
		StoredPower:          map[string]int{},
		PendingConfusion:     map[string]bool{},
		Cells: []CellInfoPush{
			{BattleID: "battle-team-all-stun", Handle: "p1", Camp: CampTeam, HP: 800, MaxHP: 1000, MP: 100, MaxMP: 100, Name: "p1"},
			{BattleID: "battle-team-all-stun", Handle: "p2", Camp: CampTeam, HP: 800, MaxHP: 1000, MP: 100, MaxMP: 100, Name: "p2"},
			{BattleID: "battle-team-all-stun", Handle: "e1", Camp: CampEnemy, HP: 5000, MaxHP: 5000, Attack: 1, Hit: 9999, Name: "e1"},
		},
	}
	runtime.applyStatusEffect("p1", BattleStatusEffect{Name: "眩晕", Display: "9.png", Rounds: 2, SkipTurn: true})
	runtime.applyStatusEffect("p2", BattleStatusEffect{Name: "眩晕", Display: "9.png", Rounds: 2, SkipTurn: true})

	if result := runtime.resolveEnemyTurnAndNextCommand(runtime.cellByHandle("p1"), []ActionPush{
		runtime.resolveSelfAction(runtime.cellByHandle("p1"), CommandDefense, "防御", "def"),
	}); result.ErrorCode != "" {
		t.Fatalf("expected first teammate action to resolve, got %+v", result)
	}
	result := runtime.resolveEnemyTurnAndNextCommand(runtime.cellByHandle("p2"), []ActionPush{
		runtime.resolveSelfAction(runtime.cellByHandle("p2"), CommandDefense, "防御", "def"),
	})

	enemyAtk := 0
	stun := 0
	for _, action := range result.Actions {
		if action.ActorHandle == "e1" {
			enemyAtk++
		}
		if action.ActionName == "眩晕" {
			stun++
		}
	}
	// Round N: both stunned and excluded -> auto continue into next enemy phase.
	// Round N+1: stun expired, reopen concurrent windows. Enemy therefore acts twice.
	if enemyAtk != 2 {
		t.Fatalf("expected one enemy phase while stunned plus one after stun ends, enemyAtk=%d actions=%+v", enemyAtk, result.Actions)
	}
	if stun < 2 {
		t.Fatalf("expected both members to emit stun status actions, stun=%d actions=%+v", stun, result.Actions)
	}
	if len(runtime.PendingTeamActions) != 2 || len(runtime.PendingStarts) != 2 {
		t.Fatalf("expected free concurrent windows after stun ends, pending=%+v starts=%+v", runtime.PendingTeamActions, runtime.PendingStarts)
	}

	playOver := runtime.ProcessPlayOver(PlayOverRequest{BattleID: runtime.BattleID})
	if playOver.ErrorCode != "" || len(playOver.StartCommands) != 2 || playOver.StartCommand != nil {
		t.Fatalf("expected ProcessPlayOver to deliver two owner windows, got %+v", playOver)
	}
	owners := map[string]bool{}
	for _, command := range playOver.StartCommands {
		owners[command.ActorHandle] = true
	}
	if !owners["p1"] || !owners["p2"] {
		t.Fatalf("expected start commands for both owners, got %+v", playOver.StartCommands)
	}
}

func TestSoloConcurrentWindowSequencesAreContiguous(t *testing.T) {
	role := session.RoleSummary{
		RoleID:      "player_seq",
		DisplayName: "序列",
		Level:       20,
		Skills: []session.RoleSkill{
			{Name: "贯甲连矢", Level: 1, Type: "oneE", Description: "f_s_贯甲连矢&2@10"},
		},
	}
	playerBase := session.PlayerBaseData{
		PlayerID:    "player_seq",
		DisplayName: "序列",
		Level:       20,
		MapID:       4,
		HP:          500,
		MaxHP:       500,
		MP:          100,
		MaxMP:       100,
	}
	runtime, bundle, ok := NewWildBattle(role, playerBase, StartRequest{MapID: "4", MapName: "涧庭村口"})
	if !ok {
		t.Fatal("expected solo wild battle to start")
	}
	if len(bundle.TeamStartCommands) != 1 {
		t.Fatalf("expected concurrent N=1 start commands, got %+v", bundle.TeamStartCommands)
	}
	first := bundle.TeamStartCommands[0]
	if first.Sequence != 1 || first.Round != 1 {
		t.Fatalf("expected first window sequence=1 round=1, got %+v", first)
	}
	if runtime.nextSequence != 2 {
		t.Fatalf("expected allocator to leave nextSequence=2 after first window, got %d", runtime.nextSequence)
	}

	target := runtime.firstLiving(CampEnemy)
	if target == nil {
		t.Fatal("expected enemy target")
	}
	// Soft enemy so the player can finish a full command -> enemy -> next window cycle.
	for index := range runtime.Cells {
		if runtime.Cells[index].Camp == CampEnemy {
			runtime.Cells[index].Attack = 0
			runtime.Cells[index].Hit = 0
		}
	}

	result := runtime.ProcessAction(ActionRequest{
		BattleID:     bundle.Start.BattleID,
		ActorHandle:  role.RoleID,
		CommandID:    CommandDefense,
		TargetHandle: role.RoleID,
		Round:        first.Round,
		Sequence:     first.Sequence,
	})
	if result.ErrorCode != "" {
		t.Fatalf("expected defense to resolve, got %+v", result)
	}
	if len(runtime.PendingStarts) != 1 {
		t.Fatalf("expected one pending start after enemy phase, starts=%+v", runtime.PendingStarts)
	}
	secondPending := runtime.PendingStarts[0]
	if secondPending.Sequence != 2 || secondPending.Round != 2 {
		t.Fatalf("expected contiguous second window sequence=2 round=2, got %+v", secondPending)
	}

	playOver := runtime.ProcessPlayOver(PlayOverRequest{BattleID: runtime.BattleID})
	if playOver.ErrorCode != "" || len(playOver.StartCommands) != 1 || playOver.StartCommand != nil {
		t.Fatalf("expected ProcessPlayOver to deliver only StartCommands for N=1, got %+v", playOver)
	}
	second := playOver.StartCommands[0]
	if second.Sequence != 2 || second.Round != 2 {
		t.Fatalf("expected delivered second window sequence=2 round=2, got %+v", second)
	}

	// Third window must continue 3, not skip to 4/5 from double-advance.
	result = runtime.ProcessAction(ActionRequest{
		BattleID:     runtime.BattleID,
		ActorHandle:  role.RoleID,
		CommandID:    CommandDefense,
		TargetHandle: role.RoleID,
		Round:        second.Round,
		Sequence:     second.Sequence,
	})
	if result.ErrorCode != "" {
		t.Fatalf("expected second defense to resolve, got %+v", result)
	}
	playOver = runtime.ProcessPlayOver(PlayOverRequest{BattleID: runtime.BattleID})
	if playOver.ErrorCode != "" || len(playOver.StartCommands) != 1 {
		t.Fatalf("expected third window delivery, got %+v", playOver)
	}
	third := playOver.StartCommands[0]
	if third.Sequence != 3 || third.Round != 3 {
		t.Fatalf("expected contiguous third window sequence=3 round=3, got %+v", third)
	}
}

func TestNewTeamWildBattleUnifiedRoundLifecycle(t *testing.T) {
	defer useSourceEncounterRoll(func(int) int { return 0 })()

	actors := make([]TeamActor, 0, 4)
	for index := 1; index <= 4; index++ {
		roleID := "team-flow-" + string(rune('0'+index))
		actors = append(actors, TeamActor{
			Role: session.RoleSummary{
				RoleID:      roleID,
				DisplayName: roleID,
				Level:       20,
			},
			PlayerBase: session.PlayerBaseData{
				PlayerID:    "player-" + roleID,
				DisplayName: roleID,
				Level:       20,
				MapID:       4,
				HP:          1000,
				MaxHP:       1000,
				MP:          100,
				MaxMP:       100,
			},
		})
	}

	runtime, bundle, ok := NewTeamWildBattle(actors, StartRequest{MapID: "4", MapName: "云隐村口"})
	if !ok {
		t.Fatal("expected four-member team battle to start")
	}
	for index, actor := range actors {
		command, found := bundle.StartCommandForActor(actor.Role.RoleID)
		if !found || command.Round != 1 || command.Sequence != index+1 {
			t.Fatalf("expected initial team window %d for %s, got %+v", index+1, actor.Role.RoleID, command)
		}
	}
	for index := range runtime.Cells {
		if runtime.Cells[index].Camp != CampEnemy {
			continue
		}
		runtime.Cells[index].HP = 100000
		runtime.Cells[index].MaxHP = 100000
		runtime.Cells[index].Attack = 1
		runtime.Cells[index].Hit = 1
	}

	first := runtime.cellByHandle(actors[0].Role.RoleID)
	stunned := runtime.cellByHandle(actors[1].Role.RoleID)
	confused := runtime.cellByHandle(actors[2].Role.RoleID)
	lastFree := runtime.cellByHandle(actors[3].Role.RoleID)
	if first == nil || stunned == nil || confused == nil || lastFree == nil {
		t.Fatal("expected every team actor to have a battle cell")
	}
	runtime.applyQiYuStatusEffect(first)
	runtime.applyKuangBao(first.Handle)
	runtime.applyKuangBao(lastFree.Handle)
	runtime.applyStatusEffect(stunned.Handle, BattleStatusEffect{Name: "眩晕", Display: "9.png", Rounds: 2, SkipTurn: true})
	runtime.applyStatusEffect(confused.Handle, BattleStatusEffect{Name: "混乱", Display: "20.png", Rounds: 2})

	var final ActionResult
	for _, actor := range actors {
		command, found := bundle.StartCommandForActor(actor.Role.RoleID)
		if !found {
			t.Fatalf("missing command for %s", actor.Role.RoleID)
		}
		final = runtime.ProcessAction(ActionRequest{
			BattleID:     bundle.Start.BattleID,
			ActorHandle:  actor.Role.RoleID,
			CommandID:    CommandDefense,
			TargetHandle: actor.Role.RoleID,
			Round:        command.Round,
			Sequence:     command.Sequence,
		})
		if final.ErrorCode != "" {
			t.Fatalf("expected %s defense to resolve, got %+v", actor.Role.RoleID, final)
		}
	}

	counts := map[string]int{}
	for _, action := range final.Actions {
		if action.ActorHandle == first.Handle && action.ActionName == "气疗" {
			counts["heal"]++
		}
		if action.ActorHandle == stunned.Handle && action.ActionName == "眩晕" {
			counts["stun"]++
		}
		if action.ActorHandle == confused.Handle && action.ActionName == "混乱" {
			counts["confusion"]++
		}
		if action.ActorHandle == confused.Handle && action.ActionName == "普通攻击" {
			counts["confusionAttack"]++
		}
	}
	if counts["heal"] != 1 || counts["stun"] != 1 || counts["confusion"] != 1 || counts["confusionAttack"] != 1 {
		t.Fatalf("expected complete unified status cycle, got counts=%+v actions=%+v", counts, final.Actions)
	}
	if runtime.StatusEffects[first.Handle].KuangBaoRounds != 2 || runtime.StatusEffects[lastFree.Handle].KuangBaoRounds != 2 {
		t.Fatalf("expected all hotblood rounds to advance, got first=%+v last=%+v", runtime.StatusEffects[first.Handle], runtime.StatusEffects[lastFree.Handle])
	}
	if len(runtime.PendingTeamActions) != 2 || !runtime.PendingTeamActions[first.Handle] || !runtime.PendingTeamActions[lastFree.Handle] || runtime.PendingTeamActions[stunned.Handle] || runtime.PendingTeamActions[confused.Handle] {
		t.Fatalf("expected windows only for free actors, got %+v", runtime.PendingTeamActions)
	}
	if len(runtime.PendingStarts) != 2 {
		t.Fatalf("expected two pending starts for free actors, got %+v", runtime.PendingStarts)
	}

	playOver := runtime.ProcessPlayOver(PlayOverRequest{BattleID: runtime.BattleID})
	if playOver.ErrorCode != "" || playOver.StartCommand != nil || len(playOver.StartCommands) != 2 {
		t.Fatalf("expected only two personalized start commands after playback, got %+v", playOver)
	}
	nextSequences := map[string]int{}
	for _, command := range playOver.StartCommands {
		nextSequences[command.ActorHandle] = command.Sequence
	}
	if nextSequences[first.Handle] != 5 || nextSequences[lastFree.Handle] != 6 || len(nextSequences) != 2 {
		t.Fatalf("expected contiguous next windows 5/6 for free actors, got %+v", nextSequences)
	}
}

func TestNewWildBattleFirstProcessActionUsesWindowSequenceForCombatRolls(t *testing.T) {
	role := session.RoleSummary{
		RoleID:      "player_roll_seed",
		DisplayName: "种子",
		Level:       20,
		Skills: []session.RoleSkill{
			{Name: "贯甲连矢", Level: 1, Type: "oneE", Description: "f_s_贯甲连矢&2@10"},
		},
	}
	playerBase := session.PlayerBaseData{
		PlayerID:    "player_roll_seed",
		DisplayName: "种子",
		Level:       20,
		MapID:       4,
		HP:          500,
		MaxHP:       500,
		MP:          100,
		MaxMP:       100,
	}
	runtime, bundle, ok := NewWildBattle(role, playerBase, StartRequest{MapID: "4", MapName: "涧庭村口"})
	if !ok {
		t.Fatal("expected solo wild battle to start")
	}
	if len(bundle.TeamStartCommands) != 1 || bundle.TeamStartCommands[0].Sequence != 1 {
		t.Fatalf("expected first window sequence=1, got %+v", bundle.TeamStartCommands)
	}
	// Allocator cursor is already past the open window.
	if runtime.nextSequence != 2 {
		t.Fatalf("expected nextSequence=2 after opening first window, got %d", runtime.nextSequence)
	}

	actor := runtime.cellByHandle(role.RoleID)
	target := runtime.firstLiving(CampEnemy)
	if actor == nil || target == nil {
		t.Fatal("expected actor and enemy target")
	}
	actor.Fat = 10000
	target.Dog = 0
	target.HP = 100000
	target.MaxHP = 100000
	window := bundle.TeamStartCommands[0]

	// Explicit seed-1 baseline must match the real first ProcessAction roll.
	baseline := &Runtime{
		BattleID:       runtime.BattleID,
		Round:          1,
		actionSequence: window.Sequence,
		nextSequence:   99, // deliberately wrong allocator cursor; must be ignored while actionSequence is set
		Cells:          append([]CellInfoPush(nil), runtime.Cells...),
	}
	baselineActor := baseline.cellByHandle(role.RoleID)
	baselineTarget := baseline.firstLiving(CampEnemy)
	profile := baseline.battleCommandProfile(baselineActor, CommandNormalAttack)
	wantCritical := baseline.resolveCriticalHit(baselineActor, baselineTarget, CommandNormalAttack, profile)

	// Cursor-only seed (nextSequence=2, no actionSequence) must differ from the window seed.
	cursorOnly := &Runtime{
		BattleID:     runtime.BattleID,
		Round:        1,
		nextSequence: 2,
		Cells:        append([]CellInfoPush(nil), runtime.Cells...),
	}
	baselineRoll := baseline.hashBattleRollWithSalt(baselineActor, baselineTarget, CommandNormalAttack, "fat")
	cursorRoll := cursorOnly.hashBattleRollWithSalt(cursorOnly.cellByHandle(role.RoleID), cursorOnly.firstLiving(CampEnemy), CommandNormalAttack, "fat")
	if cursorRoll == baselineRoll {
		t.Fatal("test fixture must distinguish window seed 1 from allocator cursor seed 2")
	}

	result := runtime.ProcessAction(ActionRequest{
		BattleID:     runtime.BattleID,
		ActorHandle:  role.RoleID,
		CommandID:    CommandNormalAttack,
		TargetHandle: target.Handle,
		Round:        window.Round,
		Sequence:     window.Sequence,
	})
	if result.ErrorCode != "" || len(result.Actions) == 0 {
		t.Fatalf("expected first normal action to resolve, got %+v", result)
	}
	firstAction := result.Actions[0]
	wantState := "0"
	if wantCritical {
		wantState = "2"
	}
	if firstAction.ActorHandle != role.RoleID || firstAction.Sequence != window.Sequence || firstAction.TargetActionStateCode != wantState {
		t.Fatalf("expected real first action to use window-seeded critical result %s, got %+v", wantState, firstAction)
	}
}
