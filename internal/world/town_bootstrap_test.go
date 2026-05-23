package world

import (
	"testing"

	"ai-server/internal/session"
)

func TestBuildTownBootstrapUsesCapturedMapOneData(t *testing.T) {
	role := session.RoleSummary{
		RoleID:       "acct-test-role-001",
		DisplayName:  "测试女侠",
		Level:        1,
		MapID:        1,
		VisualRoleID: 1,
		PresetID:     2,
		SourceQuery:  "human/human.swf?w1=1&",
	}
	playerBase := session.PlayerBaseData{
		PlayerID:     "acct-test",
		RoleID:       role.RoleID,
		DisplayName:  role.DisplayName,
		Level:        role.Level,
		MapID:        1,
		VisualRoleID: role.VisualRoleID,
		PresetID:     role.PresetID,
		SourceQuery:  role.SourceQuery,
	}

	snapshot := BuildTownBootstrap(role, playerBase)

	if snapshot.LoadMap.MapID != "1" {
		t.Fatalf("expected map id 1, got %q", snapshot.LoadMap.MapID)
	}
	if snapshot.LoadMap.MapName != "云隐村" {
		t.Fatalf("expected map name 云隐村, got %q", snapshot.LoadMap.MapName)
	}
	if snapshot.LoadMap.XMLURL != "xml/1.xml" {
		t.Fatalf("expected xml/1.xml, got %q", snapshot.LoadMap.XMLURL)
	}
	if snapshot.LoadMap.EnemyShow {
		t.Fatal("expected map1 enemyShow to stay false")
	}
	if snapshot.CreatePlayer.SpawnFlash.X != 820 || snapshot.CreatePlayer.SpawnFlash.Y != 451 {
		t.Fatalf("expected player spawn 820,451 got %+v", snapshot.CreatePlayer.SpawnFlash)
	}
	if len(snapshot.CreateRoles) != 13 {
		t.Fatalf("expected 13 source npcs, got %d", len(snapshot.CreateRoles))
	}
	if len(snapshot.QuestStates) != 13 {
		t.Fatalf("expected 13 quest states, got %d", len(snapshot.QuestStates))
	}

	var foundQipin bool
	var foundTransport bool
	for _, rolePush := range snapshot.CreateRoles {
		if rolePush.Handle == "transp_4" {
			foundTransport = true
			if rolePush.RoleID != "-3" {
				t.Fatalf("expected transport role id -3, got %q", rolePush.RoleID)
			}
			if rolePush.DisplayName != "" {
				t.Fatalf("expected source transport name to be empty, got %q", rolePush.DisplayName)
			}
			if rolePush.SourceNPCVisual == nil || rolePush.SourceNPCVisual.MovieClipIRPath != "runtime/classic-npc/movieclips/flag2/flag2-movieclip-ir" {
				t.Fatalf("expected transport flag2 visual, got %+v", rolePush.SourceNPCVisual)
			}
		}
		if rolePush.Handle != "7000542609490978" {
			continue
		}
		foundQipin = true
		if rolePush.DisplayName != "丑七品" {
			t.Fatalf("expected 丑七品, got %q", rolePush.DisplayName)
		}
		if rolePush.SpawnFlash.X != 2773 || rolePush.SpawnFlash.Y != 432 {
			t.Fatalf("expected qipin spawn 2773,432 got %+v", rolePush.SpawnFlash)
		}
		if rolePush.SourceNPCVisual == nil || rolePush.SourceNPCVisual.MovieClipIRPath == "" {
			t.Fatal("expected qipin source npc visual to be populated")
		}
	}
	if !foundQipin {
		t.Fatal("expected to find qipin in snapshot create roles")
	}
	if !foundTransport {
		t.Fatal("expected to find transport marker in snapshot create roles")
	}

	answerSpeak := BuildAnswerSpeak("1000542608713897")
	if answerSpeak.MsgHandle != "1" {
		t.Fatalf("expected captured msg handle 1, got %q", answerSpeak.MsgHandle)
	}
	if answerSpeak.Msg != `((一间茅屋从来住，心无是非万境闲。长问尘世谁有路，态意从容若神仙。))
贫道一心长态是也，久居云隐村这风轻云淡之隅，昼来农田耕种，夜来把酒赏月，想来这神仙的生活也不过老夫这般了。
((每日13:00-15:00或者19:00-21:00到我这可以接取生死劫任务))。` {
		t.Fatalf("expected npc-specific dialogue, got %q", answerSpeak.Msg)
	}
	if len(answerSpeak.Answers) != 6 || answerSpeak.Answers[0].Handle != "1q32gs" || answerSpeak.Answers[1].Msg != "研习职业" {
		t.Fatalf("expected skill mentor answers, got %+v", answerSpeak.Answers)
	}

	mengdayingSpeak := BuildAnswerSpeak("2000542608832485")
	if mengdayingSpeak.MsgHandle != "1" || len(mengdayingSpeak.Answers) != 2 || mengdayingSpeak.Answers[0].Handle != "1q28gs" {
		t.Fatalf("expected captured mengdaying quest entry, got %+v", mengdayingSpeak)
	}
	reply := BuildAnswerReply("2000542608832485", "1", "1q28gs")
	if reply == nil {
		t.Fatal("expected captured mengdaying follow-up dialogue")
	}
	if reply.MsgHandle != "1q28d_1" {
		t.Fatalf("expected follow-up msg handle 1q28d_1, got %q", reply.MsgHandle)
	}
	if len(reply.Answers) != 2 || reply.Answers[0].Handle != "1q28a_1_1" {
		t.Fatalf("expected follow-up answers, got %+v", reply.Answers)
	}

	exitSpeak := BuildAnswerSpeak("transp_4")
	if exitSpeak.Msg != "可以瞬间传送至该处。" {
		t.Fatalf("expected transport dialogue, got %q", exitSpeak.Msg)
	}
	if len(exitSpeak.Answers) != 2 || exitSpeak.Answers[0].Msg != "前往" {
		t.Fatalf("expected transport answers, got %+v", exitSpeak.Answers)
	}
}

func TestResolveTownTransportAnswerUsesHandleDestination(t *testing.T) {
	destination, ok := ResolveTownTransportAnswer("transp_4", "goto")
	if !ok {
		t.Fatal("expected transp_4 goto answer to resolve")
	}
	if destination.MapID != 4 {
		t.Fatalf("expected mapId 4, got %d", destination.MapID)
	}
	if destination.Spawn.X != 1000 || destination.Spawn.Y != 600 {
		t.Fatalf("expected default source spawn 1000,600 got %+v", destination.Spawn)
	}

	if _, ok := ResolveTownTransportAnswer("transp_4", "leave"); ok {
		t.Fatal("expected leave answer not to transfer")
	}

	returnDestination, ok := ResolveTownTransportAnswer("transp_1", "goto")
	if !ok {
		t.Fatal("expected transp_1 goto answer to resolve")
	}
	if returnDestination.MapID != 1 {
		t.Fatalf("expected mapId 1, got %d", returnDestination.MapID)
	}
}

func TestResolveTownTransportAnswerUsesSourceMapXYSpawn(t *testing.T) {
	destination, ok := ResolveTownTransportAnswer("transp_5", "goto")
	if !ok {
		t.Fatal("expected transp_5 goto answer to resolve")
	}
	if destination.MapID != 5 {
		t.Fatalf("expected mapId 5, got %d", destination.MapID)
	}
	if destination.Spawn.X != 1000 || destination.Spawn.Y != 450 {
		t.Fatalf("expected source mapxy spawn 1000,450 got %+v", destination.Spawn)
	}
}

func TestBuildTownTransferBootstrapUsesMapFourScene(t *testing.T) {
	role := session.RoleSummary{
		RoleID:       "acct-test-role-001",
		DisplayName:  "测试女侠",
		Level:        8,
		MapID:        4,
		VisualRoleID: 1,
	}
	playerBase := session.PlayerBaseData{
		PlayerID:     "acct-test",
		RoleID:       role.RoleID,
		DisplayName:  role.DisplayName,
		Level:        role.Level,
		MapID:        4,
		VisualRoleID: role.VisualRoleID,
	}

	snapshot, ok := BuildTownTransferBootstrap(role, playerBase, 4, SpawnPoint{X: 1000, Y: 600})
	if !ok {
		t.Fatal("expected map4 transfer bootstrap to be supported")
	}
	if snapshot.LoadMap.MapID != "4" || snapshot.LoadMap.MapName != "云隐村口" || snapshot.LoadMap.XMLURL != "xml/4.xml" {
		t.Fatalf("expected map4 loadMap, got %+v", snapshot.LoadMap)
	}
	if !snapshot.LoadMap.EnemyShow {
		t.Fatal("expected map4 enemyShow to be true for wild battle maps")
	}
	if snapshot.CreatePlayer.SpawnFlash.X != 1000 || snapshot.CreatePlayer.SpawnFlash.Y != 600 {
		t.Fatalf("expected transfer spawn 1000,600 got %+v", snapshot.CreatePlayer.SpawnFlash)
	}
	if len(snapshot.CreateRoles) != 2 || len(snapshot.QuestStates) != 2 {
		t.Fatalf("expected map4 source transport roles, got roles=%d quests=%d", len(snapshot.CreateRoles), len(snapshot.QuestStates))
	}
	if snapshot.CreateRoles[0].Handle != "transp_1" || snapshot.CreateRoles[1].Handle != "transp_5" {
		t.Fatalf("expected map4 transports to map1 and map5, got %+v", snapshot.CreateRoles)
	}
	if snapshot.CreateRoles[0].SpawnFlash.X != 290 || snapshot.CreateRoles[0].SpawnFlash.Y != 520 {
		t.Fatalf("expected map4 left transport at 290,520 got %+v", snapshot.CreateRoles[0].SpawnFlash)
	}
	if snapshot.CreateRoles[1].SpawnFlash.X != 2710 || snapshot.CreateRoles[1].SpawnFlash.Y != 520 {
		t.Fatalf("expected map4 right transport at 2710,520 got %+v", snapshot.CreateRoles[1].SpawnFlash)
	}
	for _, rolePush := range snapshot.CreateRoles {
		if rolePush.RoleID != "-3" || rolePush.SourceNPCVisual == nil {
			t.Fatalf("expected source transport visual role, got %+v", rolePush)
		}
		if rolePush.DisplayName != "" {
			t.Fatalf("expected source transport name to be empty, got %q", rolePush.DisplayName)
		}
	}
}

func TestBuildTownTransferBootstrapMarksCapturedWildMapEnemyShow(t *testing.T) {
	role := session.RoleSummary{
		RoleID:       "acct-test-role-001",
		DisplayName:  "测试女侠",
		Level:        8,
		MapID:        5,
		VisualRoleID: 1,
	}
	playerBase := session.PlayerBaseData{
		PlayerID:     "acct-test",
		RoleID:       role.RoleID,
		DisplayName:  role.DisplayName,
		Level:        role.Level,
		MapID:        5,
		VisualRoleID: role.VisualRoleID,
	}

	snapshot, ok := BuildTownTransferBootstrap(role, playerBase, 5, SpawnPoint{X: 1000, Y: 450})
	if !ok {
		t.Fatal("expected map5 transfer bootstrap to be supported")
	}
	if snapshot.LoadMap.MapID != "5" || snapshot.LoadMap.MapName != "云隐村口_1" || snapshot.LoadMap.XMLURL != "xml/5.xml" {
		t.Fatalf("expected map5 loadMap, got %+v", snapshot.LoadMap)
	}
	if !snapshot.LoadMap.EnemyShow {
		t.Fatal("expected captured map5 enemyShow to be true")
	}
	if snapshot.CreatePlayer.SpawnFlash.X != 1000 || snapshot.CreatePlayer.SpawnFlash.Y != 450 {
		t.Fatalf("expected transfer spawn 1000,450 got %+v", snapshot.CreatePlayer.SpawnFlash)
	}
}

func TestBuildTownTransferBootstrapSupportsGeneratedMapsWithoutNPCs(t *testing.T) {
	role := session.RoleSummary{
		RoleID:       "acct-test-role-001",
		DisplayName:  "测试女侠",
		Level:        8,
		MapID:        430,
		VisualRoleID: 1,
	}
	playerBase := session.PlayerBaseData{
		PlayerID:     "acct-test",
		RoleID:       role.RoleID,
		DisplayName:  role.DisplayName,
		Level:        role.Level,
		MapID:        430,
		VisualRoleID: role.VisualRoleID,
	}

	snapshot, ok := BuildTownTransferBootstrap(role, playerBase, 430, SpawnPoint{X: 777, Y: 555})
	if !ok {
		t.Fatal("expected generated map430 transfer bootstrap to be supported")
	}
	if snapshot.LoadMap.MapID != "430" || snapshot.LoadMap.XMLURL != "xml/430.xml" {
		t.Fatalf("expected map430 loadMap, got %+v", snapshot.LoadMap)
	}
	if snapshot.CreatePlayer.SpawnFlash.X != 777 || snapshot.CreatePlayer.SpawnFlash.Y != 555 {
		t.Fatalf("expected requested transfer spawn 777,555 got %+v", snapshot.CreatePlayer.SpawnFlash)
	}
	if len(snapshot.CreateRoles) != 0 || len(snapshot.QuestStates) != 0 {
		t.Fatalf("expected generated maps to stay npc-free until captured, got roles=%d quests=%d", len(snapshot.CreateRoles), len(snapshot.QuestStates))
	}
}

func TestSupportsTownTransferMapRejectsMissingMap(t *testing.T) {
	if !SupportsTownTransferMap(430) {
		t.Fatal("expected generated map430 to support transfer")
	}
	if SupportsTownTransferMap(9999) {
		t.Fatal("expected missing map9999 to reject transfer")
	}
}
