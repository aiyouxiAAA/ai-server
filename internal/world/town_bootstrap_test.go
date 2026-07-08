package world

import (
	"encoding/json"
	"strings"
	"testing"

	"ai-server/internal/classicactivity"
	"ai-server/internal/session"
)

func hasAnswerOption(options []AnswerOption, handle string, msg string) bool {
	for _, option := range options {
		if option.Handle == handle && option.Msg == msg {
			return true
		}
	}
	return false
}

func TestTownBootstrapAppliesCapturedSourceTransportPoints(t *testing.T) {
	checked := 0
	for mapID, expectedTransports := range capturedSourceTransportsByMapID {
		definition, ok := townMapBootstrapDefinitions[mapID]
		if !ok {
			t.Fatalf("expected captured transport map%d to exist", mapID)
		}
		for _, expected := range expectedTransports {
			actual, ok := findSourceNPCInDefinition(definition, expected.Handle)
			if !ok {
				t.Fatalf("expected map%d captured transport %s to exist", mapID, expected.Handle)
			}
			if actual.SourceQuery != expected.SourceQuery {
				t.Fatalf("expected map%d %s sourceQuery %s got %s", mapID, expected.Handle, expected.SourceQuery, actual.SourceQuery)
			}
			if actual.SpawnFlash != expected.SpawnFlash {
				t.Fatalf("expected map%d %s spawn %+v got %+v", mapID, expected.Handle, expected.SpawnFlash, actual.SpawnFlash)
			}
			checked++
		}
	}
	if checked != 328 {
		t.Fatalf("expected 328 captured transport points, checked %d", checked)
	}
}

func findSourceNPCInDefinition(definition townMapBootstrapDefinition, handle string) (sourceNPCEntry, bool) {
	for _, npc := range definition.SourceNPCs {
		if npc.Handle == handle {
			return npc, true
		}
	}
	return sourceNPCEntry{}, false
}

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

func TestResolveTownTransportAnswerAcceptsCapturedConfirmHandle(t *testing.T) {
	destination, ok := ResolveTownTransportAnswer("transp_10", "1")
	if !ok {
		t.Fatal("expected captured transp_10 answer handle 1 to resolve")
	}
	if destination.MapID != 10 {
		t.Fatalf("expected mapId 10, got %d", destination.MapID)
	}
	if destination.Spawn.X != 180 || destination.Spawn.Y != 500 {
		t.Fatalf("expected captured map10 spawn 180,500 got %+v", destination.Spawn)
	}
}

func TestResolveTownTransportAnswerUsesCapturedJiantingRoadSpawn(t *testing.T) {
	destination, ok := ResolveTownTransportAnswer("transp_11", "goto")
	if !ok {
		t.Fatal("expected transp_11 goto answer to resolve")
	}
	if destination.MapID != 11 {
		t.Fatalf("expected mapId 11, got %d", destination.MapID)
	}
	if destination.Spawn.X != 200 || destination.Spawn.Y != 482 {
		t.Fatalf("expected captured map11 spawn 200,482 got %+v", destination.Spawn)
	}
}

func TestResolveTownTransportAnswerUsesCapturedReturnHandleDestination(t *testing.T) {
	destination, ok := ResolveTownTransportAnswer("transp_0", "goto")
	if !ok {
		t.Fatal("expected transp_0 goto answer to resolve")
	}
	if destination.MapID != 3 {
		t.Fatalf("expected mapId 3, got %d", destination.MapID)
	}
	if destination.Spawn.X != 825 || destination.Spawn.Y != 624 {
		t.Fatalf("expected captured map3 spawn 825,624 got %+v", destination.Spawn)
	}
}

func TestResolveTownTransportAnswerUsesShuiliandongDestination(t *testing.T) {
	destination, ok := ResolveTownTransportAnswer("transp_131", "goto")
	if !ok {
		t.Fatal("expected transp_131 goto answer to resolve")
	}
	if destination.MapID != 131 {
		t.Fatalf("expected mapId 131, got %d", destination.MapID)
	}
	if destination.Spawn.X != 1000 || destination.Spawn.Y != 600 {
		t.Fatalf("expected water cave default spawn 1000,600 got %+v", destination.Spawn)
	}

	returnDestination, ok := ResolveTownTransportAnswer("transp_127", "goto")
	if !ok {
		t.Fatal("expected transp_127 goto answer to resolve")
	}
	if returnDestination.MapID != 127 {
		t.Fatalf("expected mapId 127, got %d", returnDestination.MapID)
	}
}

func TestResolveTownTransportAnswerFromMapUsesCapturedHuangfengzhaiRouteSpawns(t *testing.T) {
	testCases := []struct {
		fromMapID int
		handle    string
		mapID     int
		spawn     SpawnPoint
	}{
		{fromMapID: 122, handle: "transp_146", mapID: 146, spawn: SpawnPoint{X: 1860, Y: 448}},
		{fromMapID: 146, handle: "transp_147", mapID: 147, spawn: SpawnPoint{X: 2850, Y: 498}},
		{fromMapID: 146, handle: "transp_152", mapID: 152, spawn: SpawnPoint{X: 2863, Y: 550}},
		{fromMapID: 147, handle: "transp_148", mapID: 148, spawn: SpawnPoint{X: 1855, Y: 468}},
		{fromMapID: 148, handle: "transp_149", mapID: 149, spawn: SpawnPoint{X: 129, Y: 544}},
		{fromMapID: 149, handle: "transp_148", mapID: 148, spawn: SpawnPoint{X: 424, Y: 401}},
		{fromMapID: 149, handle: "transp_150", mapID: 150, spawn: SpawnPoint{X: 1426, Y: 633}},
		{fromMapID: 150, handle: "transp_149", mapID: 149, spawn: SpawnPoint{X: 2280, Y: 365}},
		{fromMapID: 150, handle: "transp_153", mapID: 153, spawn: SpawnPoint{X: 2785, Y: 434}},
		{fromMapID: 151, handle: "transp_150", mapID: 150, spawn: SpawnPoint{X: 2893, Y: 469}},
		{fromMapID: 152, handle: "transp_151", mapID: 151, spawn: SpawnPoint{X: 2373, Y: 426}},
		{fromMapID: 153, handle: "transp_156", mapID: 156, spawn: SpawnPoint{X: 2382, Y: 473}},
		{fromMapID: 154, handle: "transp_155", mapID: 155, spawn: SpawnPoint{X: 1841, Y: 506}},
		{fromMapID: 155, handle: "transp_122", mapID: 122, spawn: SpawnPoint{X: 530, Y: 520}},
		{fromMapID: 156, handle: "transp_157", mapID: 157, spawn: SpawnPoint{X: 2890, Y: 494}},
		{fromMapID: 157, handle: "transp_154", mapID: 154, spawn: SpawnPoint{X: 920, Y: 159}},
	}

	for _, testCase := range testCases {
		destination, ok := ResolveTownTransportAnswerFromMap(testCase.fromMapID, testCase.handle, "goto")
		if !ok {
			t.Fatalf("expected map%d %s to resolve", testCase.fromMapID, testCase.handle)
		}
		if destination.MapID != testCase.mapID || destination.Spawn != testCase.spawn {
			t.Fatalf("expected map%d %s -> map%d spawn %+v, got map%d spawn %+v",
				testCase.fromMapID, testCase.handle, testCase.mapID, testCase.spawn, destination.MapID, destination.Spawn)
		}
	}
}

func TestResolveTownTransportAnswerFromMapCoversHuangfengzhaiSceneTransports(t *testing.T) {
	testCases := []struct {
		fromMapID int
		handle    string
		mapID     int
	}{
		{fromMapID: 122, handle: "transp_121", mapID: 121},
		{fromMapID: 122, handle: "transp_146", mapID: 146},
		{fromMapID: 146, handle: "transp_122", mapID: 122},
		{fromMapID: 146, handle: "transp_147", mapID: 147},
		{fromMapID: 146, handle: "transp_152", mapID: 152},
		{fromMapID: 147, handle: "transp_146", mapID: 146},
		{fromMapID: 147, handle: "transp_148", mapID: 148},
		{fromMapID: 148, handle: "transp_147", mapID: 147},
		{fromMapID: 148, handle: "transp_149", mapID: 149},
		{fromMapID: 149, handle: "transp_148", mapID: 148},
		{fromMapID: 149, handle: "transp_150", mapID: 150},
		{fromMapID: 150, handle: "transp_149", mapID: 149},
		{fromMapID: 150, handle: "transp_151", mapID: 151},
		{fromMapID: 150, handle: "transp_153", mapID: 153},
		{fromMapID: 151, handle: "transp_150", mapID: 150},
		{fromMapID: 151, handle: "transp_152", mapID: 152},
		{fromMapID: 152, handle: "transp_146", mapID: 146},
		{fromMapID: 152, handle: "transp_151", mapID: 151},
		{fromMapID: 153, handle: "transp_150", mapID: 150},
		{fromMapID: 153, handle: "transp_154", mapID: 154},
		{fromMapID: 153, handle: "transp_156", mapID: 156},
		{fromMapID: 154, handle: "transp_153", mapID: 153},
		{fromMapID: 154, handle: "transp_155", mapID: 155},
		{fromMapID: 154, handle: "transp_157", mapID: 157},
		{fromMapID: 155, handle: "transp_122", mapID: 122},
		{fromMapID: 155, handle: "transp_154", mapID: 154},
		{fromMapID: 156, handle: "transp_153", mapID: 153},
		{fromMapID: 156, handle: "transp_157", mapID: 157},
		{fromMapID: 157, handle: "transp_154", mapID: 154},
		{fromMapID: 157, handle: "transp_156", mapID: 156},
	}

	for _, testCase := range testCases {
		destination, ok := ResolveTownTransportAnswerFromMap(testCase.fromMapID, testCase.handle, "goto")
		if !ok {
			t.Fatalf("expected map%d %s to resolve", testCase.fromMapID, testCase.handle)
		}
		if destination.MapID != testCase.mapID {
			t.Fatalf("expected map%d %s -> map%d, got map%d", testCase.fromMapID, testCase.handle, testCase.mapID, destination.MapID)
		}
	}
}

func TestBuildTownBootstrapUsesCapturedShuiliandongVisibleMonsters(t *testing.T) {
	role := session.RoleSummary{
		RoleID:       "acct-test-role-143",
		DisplayName:  "测试女侠",
		Level:        20,
		MapID:        143,
		VisualRoleID: 1,
	}
	playerBase := session.PlayerBaseData{
		PlayerID:     "acct-test",
		RoleID:       role.RoleID,
		DisplayName:  role.DisplayName,
		Level:        role.Level,
		MapID:        143,
		VisualRoleID: role.VisualRoleID,
	}

	snapshot, ok := BuildTownTransferBootstrap(role, playerBase, 143, SpawnPoint{X: 1000, Y: 600})
	if !ok {
		t.Fatal("expected map143 transfer bootstrap to be supported")
	}
	if snapshot.LoadMap.EnemyShow {
		t.Fatal("expected shuiliandong maps to use visible monsters instead of wild enemyShow")
	}

	var foundBoss bool
	for _, rolePush := range snapshot.CreateRoles {
		if rolePush.Handle != "5176206909809579" {
			continue
		}
		foundBoss = true
		if rolePush.Kind != "monster" || rolePush.RoleID != "-2" {
			t.Fatalf("expected visible monster role, got %+v", rolePush)
		}
		if rolePush.DisplayName != "蛤蟆精" || rolePush.Level != 20 || rolePush.Vocation != "游侠++" {
			t.Fatalf("expected captured boss identity, got %+v", rolePush)
		}
		if rolePush.SpawnFlash.X != 2070 || rolePush.SpawnFlash.Y != 464 {
			t.Fatalf("expected captured boss spawn 2070,464 got %+v", rolePush.SpawnFlash)
		}
		if rolePush.Movement != nil {
			t.Fatalf("expected captured boss to stay still without a moveRole packet, got movement %+v", rolePush.Movement)
		}
		if rolePush.SourceNPCVisual == nil || rolePush.SourceNPCVisual.MovieClipIRPath != "runtime/classic-monstermap/cracktoad/cracktoad-movieclip-ir" {
			t.Fatalf("expected cracktoad map movieclip visual, got %+v", rolePush.SourceNPCVisual)
		}
	}
	if !foundBoss {
		t.Fatal("expected captured map143 frog boss visible monster")
	}

	for _, rolePush := range snapshot.CreateRoles {
		if rolePush.Kind != "monster" || rolePush.RoleID != "-2" {
			continue
		}
		if rolePush.Movement != nil {
			t.Fatalf("expected map143 visible monster %s bootstrap to omit movement, got %+v", rolePush.Handle, rolePush.Movement)
		}
	}
}

func TestBuildTownBootstrapUsesCapturedSwampWildEnemyShow(t *testing.T) {
	role := session.RoleSummary{
		RoleID:       "acct-test-role-56",
		DisplayName:  "测试女侠",
		Level:        25,
		MapID:        56,
		VisualRoleID: 1,
	}
	playerBase := session.PlayerBaseData{
		PlayerID:     "acct-test",
		RoleID:       role.RoleID,
		DisplayName:  role.DisplayName,
		Level:        role.Level,
		MapID:        56,
		VisualRoleID: role.VisualRoleID,
	}

	for _, mapID := range []int{56, 57, 58, 59, 60, 61, 62, 63, 172, 173, 174, 175, 177, 178} {
		snapshot, ok := BuildTownTransferBootstrap(role, playerBase, mapID, SpawnPoint{X: 1000, Y: 600})
		if !ok {
			t.Fatalf("expected swamp map %d transfer bootstrap to be supported", mapID)
		}
		if !snapshot.LoadMap.EnemyShow {
			t.Fatalf("expected swamp map %d to enable captured wild enemyShow", mapID)
		}
	}

	for _, mapID := range []int{171, 176} {
		snapshot, ok := BuildTownTransferBootstrap(role, playerBase, mapID, SpawnPoint{X: 1000, Y: 600})
		if !ok {
			t.Fatalf("expected swamp map %d transfer bootstrap to be supported", mapID)
		}
		if snapshot.LoadMap.EnemyShow {
			t.Fatalf("expected swamp map %d to stay out of wild enemyShow until capture evidence exists", mapID)
		}
	}
}

func TestBuildTownBootstrapOmitsCapturedFeixiandongVisibleMonsterBootstrapMovement(t *testing.T) {
	testCases := []struct {
		mapID   int
		handles map[string]struct{}
	}{
		{
			mapID: 64,
			handles: map[string]struct{}{
				"8216674186649650": {},
				"8218674186650741": {},
			},
		},
		{
			mapID: 76,
			handles: map[string]struct{}{
				"1044675671974869": {},
				"1048675671977626": {},
			},
		},
		{
			mapID: 78,
			handles: map[string]struct{}{
				"1679675260685862": {},
				"1681675260686878": {},
			},
		},
	}

	for _, testCase := range testCases {
		role := session.RoleSummary{
			RoleID:       "acct-test-role-feixiandong-visible",
			DisplayName:  "测试女侠",
			Level:        20,
			MapID:        testCase.mapID,
			VisualRoleID: 1,
		}
		playerBase := session.PlayerBaseData{
			PlayerID:     "acct-test",
			RoleID:       role.RoleID,
			DisplayName:  role.DisplayName,
			Level:        role.Level,
			MapID:        testCase.mapID,
			VisualRoleID: role.VisualRoleID,
		}

		snapshot, ok := BuildTownTransferBootstrap(role, playerBase, testCase.mapID, SpawnPoint{X: 1000, Y: 600})
		if !ok {
			t.Fatalf("expected map%d transfer bootstrap to be supported", testCase.mapID)
		}
		for _, rolePush := range snapshot.CreateRoles {
			_, ok := testCase.handles[rolePush.Handle]
			if !ok {
				continue
			}
			if rolePush.Movement != nil {
				t.Fatalf("expected map%d visible monster %s bootstrap to omit movement, got %+v", testCase.mapID, rolePush.Handle, rolePush.Movement)
			}
			delete(testCase.handles, rolePush.Handle)
		}
		if len(testCase.handles) != 0 {
			t.Fatalf("missing captured feixiandong movement checks for map%d: %+v", testCase.mapID, testCase.handles)
		}
	}
}

func TestBuildTownBootstrapOmitsCapturedShihukuVisibleMonsterBootstrapMovement(t *testing.T) {
	testCases := []struct {
		mapID   int
		handles map[string]struct{}
	}{
		{
			mapID: 158,
			handles: map[string]struct{}{
				"5835621591157688": {},
				"5837621591158929": {},
				"5839621591159706": {},
			},
		},
		{
			mapID: 163,
			handles: map[string]struct{}{
				"8088622782646450": {},
				"8094622782649492": {},
			},
		},
		{
			mapID: 167,
			handles: map[string]struct{}{
				"7546622260836700": {},
				"7550622260838906": {},
			},
		},
	}

	for _, testCase := range testCases {
		role := session.RoleSummary{
			RoleID:       "acct-test-role-shihuku-visible",
			DisplayName:  "测试女侠",
			Level:        30,
			MapID:        testCase.mapID,
			VisualRoleID: 1,
		}
		playerBase := session.PlayerBaseData{
			PlayerID:     "acct-test",
			RoleID:       role.RoleID,
			DisplayName:  role.DisplayName,
			Level:        role.Level,
			MapID:        testCase.mapID,
			VisualRoleID: role.VisualRoleID,
		}

		snapshot, ok := BuildTownTransferBootstrap(role, playerBase, testCase.mapID, SpawnPoint{X: 1000, Y: 600})
		if !ok {
			t.Fatalf("expected map%d transfer bootstrap to be supported", testCase.mapID)
		}
		for _, rolePush := range snapshot.CreateRoles {
			_, ok := testCase.handles[rolePush.Handle]
			if !ok {
				continue
			}
			if rolePush.Movement != nil {
				t.Fatalf("expected map%d visible monster %s bootstrap to omit movement, got %+v", testCase.mapID, rolePush.Handle, rolePush.Movement)
			}
			delete(testCase.handles, rolePush.Handle)
		}
		if len(testCase.handles) != 0 {
			t.Fatalf("missing captured shihuku movement checks for map%d: %+v", testCase.mapID, testCase.handles)
		}
	}
}

func TestCapturedSourceMonsterMoveReplayUsesCapturedMoveRoleTargets(t *testing.T) {
	testCases := []struct {
		name      string
		mapID     int
		wantCount int
		want      RoleMovePush
	}{
		{
			name:      "feixiandong",
			mapID:     64,
			wantCount: 5,
			want:      RoleMovePush{Handle: "8216674186649650", Type: "Move", X: 332, Y: 417, Z: 0, TX: 328, TY: 486, TZ: 0, MapID: "64"},
		},
		{
			name:      "shuiliandong",
			mapID:     143,
			wantCount: 7,
			want:      RoleMovePush{Handle: "5166206909805441", Type: "Move", X: 1315, Y: 553, Z: 0, TX: 1862, TY: 570, TZ: 0, MapID: "143"},
		},
		{
			name:      "huangfengzhai",
			mapID:     147,
			wantCount: 26,
			want:      RoleMovePush{Handle: "7633284548716137", Type: "Move", X: 2161, Y: 342, Z: 0, TX: 2338, TY: 326, TZ: 0, MapID: "147"},
		},
		{
			name:      "shihuku",
			mapID:     158,
			wantCount: 50,
			want:      RoleMovePush{Handle: "5839621591159706", Type: "Move", X: 2196, Y: 493, Z: 0, TX: 2557, TY: 484, TZ: 0, MapID: "158"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			steps := CapturedSourceMonsterMoveReplayForMap(testCase.mapID)
			if len(steps) != testCase.wantCount {
				t.Fatalf("expected map%d replay step count %d, got %d", testCase.mapID, testCase.wantCount, len(steps))
			}
			if !hasMoveReplayStep(steps, testCase.want) {
				t.Fatalf("expected map%d replay to include captured moveRole %+v", testCase.mapID, testCase.want)
			}
		})
	}
}

func TestCapturedSourceMonsterMoveReplayForBootstrapFiltersVisibleHandles(t *testing.T) {
	snapshot := TownBootstrapSnapshot{
		LoadMap: LoadMapPush{MapID: "143"},
		CreateRoles: []RolePush{
			{Handle: "5166206909805441", RoleID: "-2", Kind: "monster"},
			{Handle: "5172206909807859", RoleID: "-2", Kind: "monster"},
			{Handle: "transp_127", RoleID: "-3", Kind: "npc"},
		},
	}

	steps := CapturedSourceMonsterMoveReplayForBootstrap(snapshot)
	if len(steps) == 0 {
		t.Fatal("expected map143 bootstrap replay steps")
	}
	for _, step := range steps {
		if step.Handle != "5166206909805441" && step.Handle != "5172206909807859" {
			t.Fatalf("expected replay to filter to visible monster createRoles, got %+v", step)
		}
		if step.MapID != "143" {
			t.Fatalf("expected replay step mapId 143, got %+v", step)
		}
	}
	if !hasMoveReplayStep(steps, RoleMovePush{Handle: "5166206909805441", Type: "Move", X: 1315, Y: 553, Z: 0, TX: 1862, TY: 570, TZ: 0, MapID: "143"}) {
		t.Fatalf("expected filtered replay to keep captured map143 caster route, got %+v", steps)
	}
}

func hasMoveReplayStep(steps []RoleMovePush, want RoleMovePush) bool {
	for _, step := range steps {
		if step == want {
			return true
		}
	}
	return false
}

func TestBuildTownBootstrapOmitsMovementForCapturedStillVisibleMonsters(t *testing.T) {
	testCases := []struct {
		mapID   int
		handles map[string]string
	}{
		{
			mapID: 143,
			handles: map[string]string{
				"5168206909805631": "法术蛤蟆",
				"5176206909809579": "蛤蟆精",
			},
		},
		{
			mapID: 144,
			handles: map[string]string{
				"2762206074545916": "武斗蛤蟆",
				"2766206074547838": "武斗蛤蟆",
			},
		},
		{
			mapID: 68,
			handles: map[string]string{
				"1515674533827497": "白咒石怪",
			},
		},
		{
			mapID: 71,
			handles: map[string]string{
				"1187674677331194": "白咒石怪",
				"1189674677333369": "白咒石怪",
			},
		},
	}

	for _, testCase := range testCases {
		role := session.RoleSummary{
			RoleID:       "acct-test-role-still-visible",
			DisplayName:  "测试女侠",
			Level:        20,
			MapID:        testCase.mapID,
			VisualRoleID: 1,
		}
		playerBase := session.PlayerBaseData{
			PlayerID:     "acct-test",
			RoleID:       role.RoleID,
			DisplayName:  role.DisplayName,
			Level:        role.Level,
			MapID:        testCase.mapID,
			VisualRoleID: role.VisualRoleID,
		}

		snapshot, ok := BuildTownTransferBootstrap(role, playerBase, testCase.mapID, SpawnPoint{X: 1000, Y: 600})
		if !ok {
			t.Fatalf("expected map%d transfer bootstrap to be supported", testCase.mapID)
		}
		for _, rolePush := range snapshot.CreateRoles {
			_, ok := testCase.handles[rolePush.Handle]
			if !ok {
				continue
			}
			if rolePush.Movement != nil {
				t.Fatalf("expected captured still monster %s on map%d to omit movement, got %+v", rolePush.Handle, testCase.mapID, rolePush.Movement)
			}
			delete(testCase.handles, rolePush.Handle)
		}
		if len(testCase.handles) != 0 {
			t.Fatalf("missing captured still monster checks for map%d: %+v", testCase.mapID, testCase.handles)
		}
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
	if snapshot.CreateRoles[0].SpawnFlash.X != 90 || snapshot.CreateRoles[0].SpawnFlash.Y != 300 {
		t.Fatalf("expected map4 left transport at 90,300 got %+v", snapshot.CreateRoles[0].SpawnFlash)
	}
	if snapshot.CreateRoles[1].SpawnFlash.X != 2908 || snapshot.CreateRoles[1].SpawnFlash.Y != 570 {
		t.Fatalf("expected map4 right transport at 2908,570 got %+v", snapshot.CreateRoles[1].SpawnFlash)
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

func TestBuildTownBootstrapDoesNotGenerateStaleYunyinRoadBranch(t *testing.T) {
	cases := []struct {
		mapID   int
		mapName string
		handles []string
		spawns  []SpawnPoint
	}{
		{mapID: 13, mapName: "云隐山道_1", handles: []string{"transp_14", "transp_9"}, spawns: []SpawnPoint{{X: 1420, Y: 260}, {X: 80, Y: 560}}},
		{mapID: 19, mapName: "树海_1", handles: []string{"transp_20", "transp_9"}, spawns: []SpawnPoint{{X: 2920, Y: 530}, {X: 67, Y: 524}}},
	}

	for _, testCase := range cases {
		role := session.RoleSummary{
			RoleID:       "acct-test-role-yunyin-road",
			DisplayName:  "测试女侠",
			Level:        8,
			MapID:        testCase.mapID,
			VisualRoleID: 1,
		}
		playerBase := session.PlayerBaseData{
			PlayerID:     "acct-test",
			RoleID:       role.RoleID,
			DisplayName:  role.DisplayName,
			Level:        role.Level,
			MapID:        testCase.mapID,
			VisualRoleID: role.VisualRoleID,
		}

		snapshot := BuildTownBootstrap(role, playerBase)
		if snapshot.LoadMap.MapID != itoa(testCase.mapID) || snapshot.LoadMap.MapName != testCase.mapName {
			t.Fatalf("expected map%d %s loadMap, got %+v", testCase.mapID, testCase.mapName, snapshot.LoadMap)
		}
		transports := make([]RolePush, 0, len(testCase.handles))
		for _, rolePush := range snapshot.CreateRoles {
			if rolePush.RoleID == "-3" {
				transports = append(transports, rolePush)
			}
		}
		if len(transports) != len(testCase.handles) {
			t.Fatalf("expected map%d transport count %d got %d: %+v", testCase.mapID, len(testCase.handles), len(transports), transports)
		}
		for index, handle := range testCase.handles {
			if transports[index].Handle != handle {
				t.Fatalf("expected map%d transport %d handle %s got %s", testCase.mapID, index, handle, transports[index].Handle)
			}
			if transports[index].SpawnFlash != testCase.spawns[index] {
				t.Fatalf("expected map%d %s spawn %+v got %+v", testCase.mapID, handle, testCase.spawns[index], transports[index].SpawnFlash)
			}
		}
	}
}

func TestBuildTownBootstrapUsesCapturedMapTwoTransportData(t *testing.T) {
	role := session.RoleSummary{
		RoleID:       "acct-test-role-002",
		DisplayName:  "测试女侠",
		Level:        8,
		MapID:        2,
		VisualRoleID: 1,
	}
	playerBase := session.PlayerBaseData{
		PlayerID:     "acct-test",
		RoleID:       role.RoleID,
		DisplayName:  role.DisplayName,
		Level:        role.Level,
		MapID:        2,
		VisualRoleID: role.VisualRoleID,
	}

	snapshot := BuildTownBootstrap(role, playerBase)
	if snapshot.LoadMap.MapID != "2" || snapshot.LoadMap.MapName != "大佛村" || snapshot.LoadMap.XMLURL != "xml/2.xml" {
		t.Fatalf("expected map2 loadMap, got %+v", snapshot.LoadMap)
	}
	if snapshot.LoadMap.EnemyShow {
		t.Fatal("expected map2 enemyShow to stay false")
	}
	if snapshot.CreatePlayer.SpawnFlash.X != 1000 || snapshot.CreatePlayer.SpawnFlash.Y != 600 {
		t.Fatalf("expected captured map2 spawn 1000,600 got %+v", snapshot.CreatePlayer.SpawnFlash)
	}
	if len(snapshot.CreateRoles) != 12 || len(snapshot.QuestStates) != 12 {
		t.Fatalf("expected map2 captured source roles, got roles=%d quests=%d", len(snapshot.CreateRoles), len(snapshot.QuestStates))
	}

	assertRole := func(handle string, name string, sourceQuery string, x int, y int) {
		t.Helper()
		for _, rolePush := range snapshot.CreateRoles {
			if rolePush.Handle != handle {
				continue
			}
			if rolePush.DisplayName != name || rolePush.SourceQuery != sourceQuery {
				t.Fatalf("expected %s/%s for %s, got name=%q source=%q", name, sourceQuery, handle, rolePush.DisplayName, rolePush.SourceQuery)
			}
			if rolePush.SpawnFlash.X != x || rolePush.SpawnFlash.Y != y {
				t.Fatalf("expected %s spawn %d,%d got %+v", handle, x, y, rolePush.SpawnFlash)
			}
			if rolePush.SourceNPCVisual == nil || rolePush.SourceNPCVisual.MovieClipIRPath == "" {
				t.Fatalf("expected source visual for %s, got %+v", handle, rolePush.SourceNPCVisual)
			}
			return
		}
		t.Fatalf("expected map2 role %s", handle)
	}

	assertRole("4180542615109515", "交易行管理员", "npc/交易行管理员.swf", 2162, 465)
	assertRole("4190542615111877", "通天八卦炉<ma>", "npc/通天八卦炉.swf", 780, 465)
	assertRole("4110542614676637", "无颜", "npc/无颜.swf", 1781, 463)
	assertRole("4170542615108676", "妖术狐狸", "npc/狐狸.swf", 2370, 465)
	assertRole("4140542615070416", "噌痴", "npc/噌痴.swf", 407, 360)
	assertRole("4100542614427315", "娴无禄", "npc/娴无录.swf", 2450, 400)
	assertRole("transp_6", "", "transp/flag2.swf", 3000, 500)

	foxSpeak := BuildAnswerSpeak("4170542615108676")
	if foxSpeak.MsgHandle != "1" || foxSpeak.Msg != `知道((广青镇))吗？它可是方圆百里最繁华的集市哦~` {
		t.Fatalf("expected captured map2 fox dialogue, got %+v", foxSpeak)
	}
	if len(foxSpeak.Answers) != 5 || foxSpeak.Answers[2].Msg != "传送到【黄风寨口】(未激活！)" {
		t.Fatalf("expected captured map2 fox answers, got %+v", foxSpeak.Answers)
	}

	chouSpeak := BuildAnswerSpeak("4090542614314425")
	if chouSpeak.MsgHandle != "1" || !hasAnswerOption(chouSpeak.Answers, "2q23gs", "<m/>讨厌的枯木怪") {
		t.Fatalf("expected captured Chou Liupin quest entry dialogue, got %+v", chouSpeak)
	}
	chouReply := BuildAnswerReply("4090542614314425", "1", "2q23gs")
	if chouReply == nil {
		t.Fatal("expected captured Chou Liupin quest follow-up dialogue")
	}
	if chouReply.MsgHandle != "2q23d_1" || !hasAnswerOption(chouReply.Answers, "2q23a_1_1", "<m/>好的，我这就去。") {
		t.Fatalf("expected Chou Liupin 2q23d_1 dialogue, got %+v", chouReply)
	}

	wuyanSpeak := BuildAnswerSpeak("4110542614676637")
	if wuyanSpeak.MsgHandle != "1" || len(wuyanSpeak.Answers) != 6 || wuyanSpeak.Answers[0].Handle != "2q21gs" {
		t.Fatalf("expected captured wuyan entry dialogue, got %+v", wuyanSpeak)
	}
	wuyanReply := BuildAnswerReply("4110542614676637", "1", "4q69gs")
	if wuyanReply == nil {
		t.Fatal("expected captured wuyan follow-up dialogue")
	}
	if wuyanReply.MsgHandle != "4q69d_1" || len(wuyanReply.Answers) != 2 || wuyanReply.Answers[0].Handle != "4q69a_1_1" {
		t.Fatalf("expected wuyan 4q69d_1 dialogue, got %+v", wuyanReply)
	}
	wuyanReply2 := BuildAnswerReply("4110542614676637", "4q69d_1", "4q69a_1_1")
	if wuyanReply2 == nil {
		t.Fatal("expected captured wuyan second follow-up dialogue")
	}
	if wuyanReply2.MsgHandle != "4q69d_2" || len(wuyanReply2.Answers) != 2 || wuyanReply2.Answers[0].Handle != "4q69a_2_1" {
		t.Fatalf("expected wuyan 4q69d_2 dialogue, got %+v", wuyanReply2)
	}
}

func TestBuildTownTransferBootstrapUsesCapturedMapThirtyThreeTransportData(t *testing.T) {
	role := session.RoleSummary{
		RoleID:       "acct-test-role-033",
		DisplayName:  "测试女侠",
		Level:        8,
		MapID:        33,
		VisualRoleID: 1,
	}
	playerBase := session.PlayerBaseData{
		PlayerID:     "acct-test",
		RoleID:       role.RoleID,
		DisplayName:  role.DisplayName,
		Level:        role.Level,
		MapID:        33,
		VisualRoleID: role.VisualRoleID,
	}

	snapshot, ok := BuildTownTransferBootstrap(role, playerBase, 33, SpawnPoint{X: 1000, Y: 600})
	if !ok {
		t.Fatal("expected map33 transfer bootstrap to be supported")
	}
	if snapshot.LoadMap.MapID != "33" || snapshot.LoadMap.MapName != "平原_2" || snapshot.LoadMap.XMLURL != "xml/33.xml" {
		t.Fatalf("expected map33 loadMap, got %+v", snapshot.LoadMap)
	}
	assertTransportRole := func(handle string, x int, y int) {
		t.Helper()
		for _, rolePush := range snapshot.CreateRoles {
			if rolePush.Handle != handle {
				continue
			}
			if rolePush.DisplayName != "" || rolePush.RoleID != "-3" || rolePush.SourceQuery != "transp/flag2.swf" {
				t.Fatalf("expected %s to be empty-name transport, got %+v", handle, rolePush)
			}
			if rolePush.SpawnFlash.X != x || rolePush.SpawnFlash.Y != y {
				t.Fatalf("expected %s spawn %d,%d got %+v", handle, x, y, rolePush.SpawnFlash)
			}
			if rolePush.SourceNPCVisual == nil || rolePush.SourceNPCVisual.MovieClipIRPath != "runtime/classic-npc/movieclips/flag2/flag2-movieclip-ir" {
				t.Fatalf("expected flag2 visual for %s, got %+v", handle, rolePush.SourceNPCVisual)
			}
			return
		}
		t.Fatalf("expected map33 transport role %s", handle)
	}

	assertTransportRole("transp_31", 57, 584)
	assertTransportRole("transp_34", 2950, 624)
	assertTransportRole("transp_37", 1124, 750)
}

func TestBuildTownBootstrapUsesCapturedBaiyuanTownTransportData(t *testing.T) {
	type expectedTransport struct {
		handle      string
		spawn       SpawnPoint
		sourceQuery string
	}
	cases := []struct {
		mapID      int
		transports []expectedTransport
	}{
		{mapID: 56, transports: []expectedTransport{
			{handle: "transp_55", spawn: SpawnPoint{X: 50, Y: 544}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_57", spawn: SpawnPoint{X: 2940, Y: 562}, sourceQuery: "transp/flag2.swf"},
		}},
		{mapID: 57, transports: []expectedTransport{
			{handle: "transp_56", spawn: SpawnPoint{X: 50, Y: 544}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_58", spawn: SpawnPoint{X: 2630, Y: 722}, sourceQuery: "transp/flag2.swf"},
		}},
		{mapID: 58, transports: []expectedTransport{
			{handle: "transp_57", spawn: SpawnPoint{X: 2241, Y: 414}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_59", spawn: SpawnPoint{X: 460, Y: 720}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_61", spawn: SpawnPoint{X: 2940, Y: 562}, sourceQuery: "transp/flag2.swf"},
		}},
		{mapID: 59, transports: []expectedTransport{
			{handle: "transp_58", spawn: SpawnPoint{X: 322, Y: 440}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_60", spawn: SpawnPoint{X: 2940, Y: 562}, sourceQuery: "transp/flag2.swf"},
		}},
		{mapID: 60, transports: []expectedTransport{
			{handle: "transp_59", spawn: SpawnPoint{X: 50, Y: 544}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_62", spawn: SpawnPoint{X: 2940, Y: 562}, sourceQuery: "transp/flag2.swf"},
		}},
		{mapID: 61, transports: []expectedTransport{
			{handle: "transp_58", spawn: SpawnPoint{X: 50, Y: 544}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_63", spawn: SpawnPoint{X: 2940, Y: 602}, sourceQuery: "transp/flag2.swf"},
		}},
		{mapID: 62, transports: []expectedTransport{
			{handle: "transp_168", spawn: SpawnPoint{X: 2940, Y: 562}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_60", spawn: SpawnPoint{X: 45, Y: 504}, sourceQuery: "transp/flag2.swf"},
		}},
		{mapID: 63, transports: []expectedTransport{
			{handle: "transp_179", spawn: SpawnPoint{X: 1394, Y: 510}, sourceQuery: "transp/fl.swf"},
			{handle: "transp_61", spawn: SpawnPoint{X: 50, Y: 504}, sourceQuery: "transp/flag2.swf"},
		}},
		{mapID: 168, transports: []expectedTransport{
			{handle: "transp_62", spawn: SpawnPoint{X: 45, Y: 504}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_169", spawn: SpawnPoint{X: 2450, Y: 554}, sourceQuery: "transp/flag2.swf"},
		}},
		{mapID: 169, transports: []expectedTransport{
			{handle: "transp_168", spawn: SpawnPoint{X: 45, Y: 564}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_170", spawn: SpawnPoint{X: 2467, Y: 585}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_173", spawn: SpawnPoint{X: 1406, Y: 720}, sourceQuery: "transp/flag2.swf"},
		}},
		{mapID: 170, transports: []expectedTransport{
			{handle: "transp_169", spawn: SpawnPoint{X: 45, Y: 544}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_171", spawn: SpawnPoint{X: 2467, Y: 585}, sourceQuery: "transp/flag2.swf"},
		}},
		{mapID: 171, transports: []expectedTransport{
			{handle: "transp_170", spawn: SpawnPoint{X: 50, Y: 544}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_172", spawn: SpawnPoint{X: 2630, Y: 722}, sourceQuery: "transp/flag2.swf"},
		}},
		{mapID: 172, transports: []expectedTransport{
			{handle: "transp_171", spawn: SpawnPoint{X: 2041, Y: 500}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_173", spawn: SpawnPoint{X: 33, Y: 591}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_174", spawn: SpawnPoint{X: 955, Y: 700}, sourceQuery: "transp/flag2.swf"},
		}},
		{mapID: 173, transports: []expectedTransport{
			{handle: "transp_169", spawn: SpawnPoint{X: 1019, Y: 441}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_172", spawn: SpawnPoint{X: 1960, Y: 569}, sourceQuery: "transp/flag2.swf"},
		}},
		{mapID: 174, transports: []expectedTransport{
			{handle: "transp_172", spawn: SpawnPoint{X: 2047, Y: 451}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_175", spawn: SpawnPoint{X: 2704, Y: 720}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_176", spawn: SpawnPoint{X: 568, Y: 730}, sourceQuery: "transp/flag2.swf"},
		}},
		{mapID: 175, transports: []expectedTransport{
			{handle: "transp_174", spawn: SpawnPoint{X: 2763, Y: 437}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_177", spawn: SpawnPoint{X: 272, Y: 700}, sourceQuery: "transp/flag2.swf"},
		}},
		{mapID: 177, transports: []expectedTransport{
			{handle: "transp_175", spawn: SpawnPoint{X: 2620, Y: 430}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_176", spawn: SpawnPoint{X: 400, Y: 450}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_178", spawn: SpawnPoint{X: 85, Y: 720}, sourceQuery: "transp/flag2.swf"},
		}},
		{mapID: 178, transports: []expectedTransport{
			{handle: "transp_177", spawn: SpawnPoint{X: 2777, Y: 460}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_190", spawn: SpawnPoint{X: 513, Y: 730}, sourceQuery: "transp/flag2.swf"},
		}},
		{mapID: 190, transports: []expectedTransport{
			{handle: "transp_178", spawn: SpawnPoint{X: 2785, Y: 420}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_191", spawn: SpawnPoint{X: 40, Y: 560}, sourceQuery: "transp/flag2.swf"},
		}},
		{mapID: 191, transports: []expectedTransport{
			{handle: "transp_190", spawn: SpawnPoint{X: 2939, Y: 530}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_192", spawn: SpawnPoint{X: 40, Y: 560}, sourceQuery: "transp/flag2.swf"},
		}},
		{mapID: 192, transports: []expectedTransport{
			{handle: "transp_191", spawn: SpawnPoint{X: 2939, Y: 560}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_193", spawn: SpawnPoint{X: 40, Y: 600}, sourceQuery: "transp/flag2.swf"},
		}},
		{mapID: 193, transports: []expectedTransport{
			{handle: "transp_192", spawn: SpawnPoint{X: 2939, Y: 580}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_194", spawn: SpawnPoint{X: 1557, Y: 470}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_198", spawn: SpawnPoint{X: 40, Y: 600}, sourceQuery: "transp/flag2.swf"},
		}},
		{mapID: 196, transports: []expectedTransport{
			{handle: "transp_195", spawn: SpawnPoint{X: 2950, Y: 580}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_197", spawn: SpawnPoint{X: 40, Y: 600}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_199", spawn: SpawnPoint{X: 1517, Y: 730}, sourceQuery: "transp/flag2.swf"},
		}},
		{mapID: 198, transports: []expectedTransport{
			{handle: "transp_193", spawn: SpawnPoint{X: 2939, Y: 560}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_199", spawn: SpawnPoint{X: 40, Y: 600}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_200", spawn: SpawnPoint{X: 1130, Y: 730}, sourceQuery: "transp/flag2.swf"},
		}},
		{mapID: 199, transports: []expectedTransport{
			{handle: "transp_196", spawn: SpawnPoint{X: 258, Y: 470}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_198", spawn: SpawnPoint{X: 2450, Y: 590}, sourceQuery: "transp/flag2.swf"},
		}},
		{mapID: 200, transports: []expectedTransport{
			{handle: "transp_198", spawn: SpawnPoint{X: 2840, Y: 420}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_201", spawn: SpawnPoint{X: 40, Y: 580}, sourceQuery: "transp/flag2.swf"},
		}},
		{mapID: 201, transports: []expectedTransport{
			{handle: "transp_200", spawn: SpawnPoint{X: 2939, Y: 550}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_202", spawn: SpawnPoint{X: 40, Y: 520}, sourceQuery: "transp/flag2.swf"},
		}},
		{mapID: 202, transports: []expectedTransport{
			{handle: "transp_197", spawn: SpawnPoint{X: 1674, Y: 425}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_201", spawn: SpawnPoint{X: 2800, Y: 730}, sourceQuery: "transp/flag2.swf"},
			{handle: "transp_205", spawn: SpawnPoint{X: 40, Y: 600}, sourceQuery: "transp/flag2.swf"},
		}},
	}

	for _, testCase := range cases {
		role := session.RoleSummary{
			RoleID:       "acct-test-role-baiyuan",
			DisplayName:  "测试女侠",
			Level:        20,
			MapID:        testCase.mapID,
			VisualRoleID: 1,
		}
		playerBase := session.PlayerBaseData{
			PlayerID:     "acct-test",
			RoleID:       role.RoleID,
			DisplayName:  role.DisplayName,
			Level:        role.Level,
			MapID:        role.MapID,
			VisualRoleID: role.VisualRoleID,
		}

		snapshot := BuildTownBootstrap(role, playerBase)
		if snapshot.LoadMap.MapID != itoa(testCase.mapID) || snapshot.LoadMap.XMLURL != "xml/"+itoa(testCase.mapID)+".xml" {
			t.Fatalf("expected map%d loadMap, got %+v", testCase.mapID, snapshot.LoadMap)
		}
		_, expectedWildEnemyShow := sourceWildBattleMapIDs[testCase.mapID]
		if snapshot.LoadMap.EnemyShow != expectedWildEnemyShow {
			t.Fatalf("expected map%d enemyShow=%v, got %v", testCase.mapID, expectedWildEnemyShow, snapshot.LoadMap.EnemyShow)
		}
		transportsByHandle := map[string]RolePush{}
		for _, rolePush := range snapshot.CreateRoles {
			if rolePush.RoleID == "-3" {
				transportsByHandle[rolePush.Handle] = rolePush
			}
		}
		if len(transportsByHandle) != len(testCase.transports) {
			t.Fatalf("expected map%d transport count %d got %d", testCase.mapID, len(testCase.transports), len(transportsByHandle))
		}
		for _, expected := range testCase.transports {
			rolePush, ok := transportsByHandle[expected.handle]
			if !ok {
				t.Fatalf("expected map%d transport handle %s, got %+v", testCase.mapID, expected.handle, transportsByHandle)
			}
			if rolePush.DisplayName != "" || rolePush.SourceQuery != expected.sourceQuery {
				t.Fatalf("expected source transport role for %s, got %+v", expected.handle, rolePush)
			}
			if rolePush.SpawnFlash != expected.spawn {
				t.Fatalf("expected map%d %s spawn %+v got %+v", testCase.mapID, expected.handle, expected.spawn, rolePush.SpawnFlash)
			}
		}
	}
}

func TestBuildTownBootstrapUsesCapturedGuangqingTownData(t *testing.T) {
	cases := []struct {
		mapID     int
		mapName   string
		roleCount int
		roles     []struct {
			handle      string
			name        string
			sourceQuery string
			x           int
			y           int
			questState  int
			irPath      string
		}
	}{
		{
			mapID:     45,
			mapName:   "广青镇_1",
			roleCount: 11,
			roles: []struct {
				handle      string
				name        string
				sourceQuery string
				x           int
				y           int
				questState  int
				irPath      string
			}{
				{handle: "1810542611191117", name: "丑二品", sourceQuery: "npc/2品丑.swf", x: 3213, y: 310, questState: 4, irPath: "runtime/classic-npc/movieclips/chouerpin/chouerpin-movieclip-ir"},
				{handle: "1790542610850918", name: "广青守卫甲", sourceQuery: "npc/卫兵_向右.swf", x: 464, y: 419, questState: 2, irPath: "runtime/classic-npc/movieclips/weibing_right/weibing_right-movieclip-ir"},
				{handle: "transp_46", name: "", sourceQuery: "transp/flag2.swf", x: 3460, y: 580, questState: 0, irPath: "runtime/classic-npc/movieclips/flag2/flag2-movieclip-ir"},
			},
		},
		{
			mapID:     46,
			mapName:   "广青镇_2",
			roleCount: 17,
			roles: []struct {
				handle      string
				name        string
				sourceQuery string
				x           int
				y           int
				questState  int
				irPath      string
			}{
				{handle: "2160542612239918", name: "白乞", sourceQuery: "npc/白乞.swf", x: 181, y: 395, questState: 4, irPath: "runtime/classic-npc/movieclips/baiqi/baiqi-movieclip-ir"},
				{handle: "2220542612946566", name: "夏侯武", sourceQuery: "npc/夏侯武.swf", x: 846, y: 430, questState: 4, irPath: "runtime/classic-npc/movieclips/xiahouwu/xiahouwu-movieclip-ir"},
				{handle: "transp_48", name: "", sourceQuery: "transp/flag2.swf", x: 1720, y: 730, questState: 0, irPath: "runtime/classic-npc/movieclips/flag2/flag2-movieclip-ir"},
			},
		},
		{
			mapID:     47,
			mapName:   "广青镇_3",
			roleCount: 9,
			roles: []struct {
				handle      string
				name        string
				sourceQuery string
				x           int
				y           int
				questState  int
				irPath      string
			}{
				{handle: "2500542613172144", name: "云衣娘", sourceQuery: "npc/云衣娘.swf", x: 1459, y: 441, questState: 0, irPath: "runtime/classic-npc/movieclips/yunyiniang/yunyiniang-movieclip-ir"},
				{handle: "2550542613498646", name: "帚公", sourceQuery: "npc/帚公.swf", x: 876, y: 409, questState: 1, irPath: "runtime/classic-npc/movieclips/zhougong/zhougong-movieclip-ir"},
				{handle: "transp_52", name: "", sourceQuery: "transp/flag2.swf", x: 3460, y: 550, questState: 0, irPath: "runtime/classic-npc/movieclips/flag2/flag2-movieclip-ir"},
			},
		},
	}

	for _, testCase := range cases {
		role := session.RoleSummary{
			RoleID:       "acct-test-role-guangqing",
			DisplayName:  "测试女侠",
			Level:        18,
			MapID:        testCase.mapID,
			VisualRoleID: 1,
		}
		playerBase := session.PlayerBaseData{
			PlayerID:     "acct-test",
			RoleID:       role.RoleID,
			DisplayName:  role.DisplayName,
			Level:        role.Level,
			MapID:        testCase.mapID,
			VisualRoleID: role.VisualRoleID,
		}

		snapshot := BuildTownBootstrap(role, playerBase)
		if snapshot.LoadMap.MapID != itoa(testCase.mapID) || snapshot.LoadMap.MapName != testCase.mapName || snapshot.LoadMap.XMLURL != "xml/"+itoa(testCase.mapID)+".xml" {
			t.Fatalf("expected map%d loadMap, got %+v", testCase.mapID, snapshot.LoadMap)
		}
		if snapshot.LoadMap.EnemyShow {
			t.Fatalf("expected map%d enemyShow to stay false", testCase.mapID)
		}
		if len(snapshot.CreateRoles) != testCase.roleCount || len(snapshot.QuestStates) != testCase.roleCount {
			t.Fatalf("expected map%d captured roles=%d got roles=%d quests=%d", testCase.mapID, testCase.roleCount, len(snapshot.CreateRoles), len(snapshot.QuestStates))
		}
		for _, expected := range testCase.roles {
			assertCapturedTownRole(t, snapshot, expected.handle, expected.name, expected.sourceQuery, expected.x, expected.y, expected.questState, expected.irPath)
		}
	}
}

func assertCapturedTownRole(
	t *testing.T,
	snapshot TownBootstrapSnapshot,
	handle string,
	name string,
	sourceQuery string,
	x int,
	y int,
	questState int,
	irPath string,
) {
	t.Helper()
	for _, rolePush := range snapshot.CreateRoles {
		if rolePush.Handle != handle {
			continue
		}
		if rolePush.DisplayName != name || rolePush.SourceQuery != sourceQuery {
			t.Fatalf("expected %s/%s for %s, got name=%q source=%q", name, sourceQuery, handle, rolePush.DisplayName, rolePush.SourceQuery)
		}
		if rolePush.SpawnFlash.X != x || rolePush.SpawnFlash.Y != y {
			t.Fatalf("expected %s spawn %d,%d got %+v", handle, x, y, rolePush.SpawnFlash)
		}
		if rolePush.SourceNPCVisual == nil || rolePush.SourceNPCVisual.MovieClipIRPath != irPath {
			t.Fatalf("expected %s visual %q, got %+v", handle, irPath, rolePush.SourceNPCVisual)
		}
		for _, statePush := range snapshot.QuestStates {
			if statePush.Handle == handle {
				if statePush.State != questState {
					t.Fatalf("expected %s quest state %d got %d", handle, questState, statePush.State)
				}
				return
			}
		}
		t.Fatalf("expected quest state for %s", handle)
	}
	t.Fatalf("expected role %s", handle)
}

func TestBuildTownBootstrapIncludesBaiyuanNPCHeadTitles(t *testing.T) {
	cases := []struct {
		mapID     int
		handle    string
		guildName string
		guildPic  string
	}{
		{mapID: 168, handle: "4710542615621525", guildName: "\u533b\u7597\u5e08", guildPic: "5002"},
		{mapID: 168, handle: "4730542615655127", guildName: "\u5408\u6210\u9053\u5177", guildPic: "5001"},
		{mapID: 169, handle: "5040542617131880", guildName: "\u6280\u80fd\u5bfc\u5e08", guildPic: "5003"},
		{mapID: 169, handle: "5060542617335713", guildName: "\u4f20\u9001\u5927\u5e08", guildPic: "5005"},
		{mapID: 169, handle: "5050542617322114", guildName: "\u4ed3\u5e93\u7ba1\u7406", guildPic: "5004"},
		{mapID: 170, handle: "5310542617702520", guildName: "\u9053\u5177\u5546", guildPic: "5001"},
		{mapID: 170, handle: "5300542617580783", guildName: "\u953b\u9020\u5e08", guildPic: "5000"},
	}

	for _, testCase := range cases {
		role := session.RoleSummary{
			RoleID:       "acct-test-role-baiyuan-head-title",
			DisplayName:  "test",
			Level:        40,
			MapID:        testCase.mapID,
			VisualRoleID: 1,
		}
		playerBase := session.PlayerBaseData{
			PlayerID:     "acct-test",
			RoleID:       role.RoleID,
			DisplayName:  role.DisplayName,
			Level:        role.Level,
			MapID:        testCase.mapID,
			VisualRoleID: role.VisualRoleID,
		}
		snapshot := BuildTownBootstrap(role, playerBase)
		var found *RolePush
		for index := range snapshot.CreateRoles {
			if snapshot.CreateRoles[index].Handle == testCase.handle {
				found = &snapshot.CreateRoles[index]
				break
			}
		}
		if found == nil {
			t.Fatalf("expected Baiyuan role %s on map%d", testCase.handle, testCase.mapID)
		}
		if found.GuildName != testCase.guildName || found.GuildPic != testCase.guildPic {
			t.Fatalf("expected %s head title %q/%q got %q/%q", testCase.handle, testCase.guildName, testCase.guildPic, found.GuildName, found.GuildPic)
		}
	}
}

func TestBuildTownBootstrapEncodesEmptyCreatePlayerGuildName(t *testing.T) {
	role := session.RoleSummary{
		RoleID:       "acct-test-role-no-guild",
		DisplayName:  "test",
		Level:        40,
		MapID:        168,
		VisualRoleID: 1,
	}
	playerBase := session.PlayerBaseData{
		PlayerID:     "acct-test",
		RoleID:       role.RoleID,
		DisplayName:  role.DisplayName,
		Level:        role.Level,
		MapID:        role.MapID,
		VisualRoleID: role.VisualRoleID,
	}
	snapshot := BuildTownBootstrap(role, playerBase)
	payload, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("marshal town bootstrap: %v", err)
	}
	var decoded struct {
		CreatePlayer map[string]interface{} `json:"createPlayer"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal town bootstrap: %v", err)
	}
	guildName, ok := decoded.CreatePlayer["guildName"]
	if !ok {
		t.Fatalf("expected empty createPlayer guildName to be encoded, payload=%s", string(payload))
	}
	if guildName != "" {
		t.Fatalf("expected empty createPlayer guildName, got %#v", guildName)
	}
}

func TestBuildTownBootstrapUsesCapturedUnderworldTransportData(t *testing.T) {
	cases := []struct {
		mapID   int
		mapName string
		xmlURL  string
		handles []string
		spawns  []SpawnPoint
	}{
		{mapID: 79, mapName: "黄泉路_1", xmlURL: "xml/79.xml", handles: []string{"transp_80"}, spawns: []SpawnPoint{{X: 60, Y: 524}}},
		{mapID: 80, mapName: "黄泉路_2", xmlURL: "xml/80.xml", handles: []string{"transp_81", "transp_79"}, spawns: []SpawnPoint{{X: 80, Y: 534}, {X: 2940, Y: 524}}},
		{mapID: 81, mapName: "黄泉路_3", xmlURL: "xml/81.xml", handles: []string{"transp_82", "transp_80"}, spawns: []SpawnPoint{{X: 75, Y: 464}, {X: 2940, Y: 464}}},
		{mapID: 82, mapName: "奈何桥", xmlURL: "xml/82.xml", handles: []string{"transp_83", "transp_81"}, spawns: []SpawnPoint{{X: 75, Y: 464}, {X: 2940, Y: 464}}},
		{mapID: 83, mapName: "重生台", xmlURL: "xml/83.xml", handles: []string{"transp_0", "transp_82"}, spawns: []SpawnPoint{{X: 341, Y: 363}, {X: 2930, Y: 480}}},
	}

	for _, testCase := range cases {
		role := session.RoleSummary{
			RoleID:       "acct-test-role-underworld",
			DisplayName:  "测试女侠",
			Level:        8,
			MapID:        testCase.mapID,
			VisualRoleID: 1,
		}
		playerBase := session.PlayerBaseData{
			PlayerID:     "acct-test",
			RoleID:       role.RoleID,
			DisplayName:  role.DisplayName,
			Level:        role.Level,
			MapID:        testCase.mapID,
			VisualRoleID: role.VisualRoleID,
		}

		snapshot := BuildTownBootstrap(role, playerBase)
		if snapshot.LoadMap.MapID != itoa(testCase.mapID) || snapshot.LoadMap.MapName != testCase.mapName || snapshot.LoadMap.XMLURL != testCase.xmlURL {
			t.Fatalf("expected map%d loadMap, got %+v", testCase.mapID, snapshot.LoadMap)
		}
		transports := make([]RolePush, 0, len(testCase.handles))
		for _, rolePush := range snapshot.CreateRoles {
			if rolePush.RoleID == "-3" {
				transports = append(transports, rolePush)
			}
		}
		if len(transports) != len(testCase.handles) {
			t.Fatalf("expected map%d transport count %d got %d", testCase.mapID, len(testCase.handles), len(transports))
		}
		for index, handle := range testCase.handles {
			rolePush := transports[index]
			if rolePush.Handle != handle {
				t.Fatalf("expected map%d transport handle %s at index %d got %s", testCase.mapID, handle, index, rolePush.Handle)
			}
			if rolePush.SpawnFlash != testCase.spawns[index] {
				t.Fatalf("expected map%d %s spawn %+v got %+v", testCase.mapID, handle, testCase.spawns[index], rolePush.SpawnFlash)
			}
		}
	}
}

func TestBuildTownBootstrapUsesCapturedShuiliandongTransportData(t *testing.T) {
	cases := []struct {
		mapID   int
		mapName string
		handles []string
		spawns  []SpawnPoint
	}{
		{mapID: 127, mapName: "观瀑台", handles: []string{"transp_131", "transp_126"}, spawns: []SpawnPoint{{X: 1020, Y: 300}, {X: 1950, Y: 570}}},
		{mapID: 131, mapName: "水帘洞_1", handles: []string{"transp_132", "transp_127"}, spawns: []SpawnPoint{{X: 2950, Y: 550}, {X: 44, Y: 530}}},
		{mapID: 133, mapName: "水帘洞_3", handles: []string{"transp_132", "transp_137", "transp_134"}, spawns: []SpawnPoint{{X: 44, Y: 530}, {X: 2960, Y: 530}, {X: 1909, Y: 720}}},
		{mapID: 137, mapName: "水帘洞_7", handles: []string{"transp_133", "transp_144", "transp_138"}, spawns: []SpawnPoint{{X: 40, Y: 555}, {X: 2960, Y: 570}, {X: 1740, Y: 380}}},
		{mapID: 140, mapName: "水帘洞_10", handles: []string{"transp_141", "transp_145", "transp_139"}, spawns: []SpawnPoint{{X: 2950, Y: 650}, {X: 40, Y: 500}, {X: 1446, Y: 360}}},
		{mapID: 142, mapName: "水帘洞_12", handles: []string{"transp_141", "transp_143", "transp_136"}, spawns: []SpawnPoint{{X: 2780, Y: 350}, {X: 1909, Y: 720}, {X: 40, Y: 520}}},
		{mapID: 143, mapName: "水帘洞_13", handles: []string{"transp_142", "transp_127"}, spawns: []SpawnPoint{{X: 220, Y: 350}, {X: 2120, Y: 440}}},
		{mapID: 145, mapName: "水帘洞_15", handles: []string{"transp_140", "transp_144", "transp_136"}, spawns: []SpawnPoint{{X: 2460, Y: 550}, {X: 200, Y: 410}, {X: 1523, Y: 720}}},
	}

	for _, testCase := range cases {
		role := session.RoleSummary{
			RoleID:       "acct-test-role-shuiliandong",
			DisplayName:  "测试女侠",
			Level:        20,
			MapID:        testCase.mapID,
			VisualRoleID: 1,
		}
		playerBase := session.PlayerBaseData{
			PlayerID:     "acct-test",
			RoleID:       role.RoleID,
			DisplayName:  role.DisplayName,
			Level:        role.Level,
			MapID:        testCase.mapID,
			VisualRoleID: role.VisualRoleID,
		}

		snapshot := BuildTownBootstrap(role, playerBase)
		if snapshot.LoadMap.MapID != itoa(testCase.mapID) || snapshot.LoadMap.MapName != testCase.mapName || snapshot.LoadMap.XMLURL != "xml/"+itoa(testCase.mapID)+".xml" {
			t.Fatalf("expected map%d loadMap, got %+v", testCase.mapID, snapshot.LoadMap)
		}
		transports := make([]RolePush, 0, len(testCase.handles))
		for _, rolePush := range snapshot.CreateRoles {
			if rolePush.RoleID == "-3" {
				transports = append(transports, rolePush)
			}
		}
		if len(transports) != len(testCase.handles) {
			t.Fatalf("expected map%d transport count %d got %d", testCase.mapID, len(testCase.handles), len(transports))
		}
		transportsByHandle := map[string]RolePush{}
		for _, rolePush := range transports {
			transportsByHandle[rolePush.Handle] = rolePush
		}
		for index, handle := range testCase.handles {
			rolePush, ok := transportsByHandle[handle]
			if !ok {
				t.Fatalf("expected map%d transport handle %s, got %+v", testCase.mapID, handle, transports)
			}
			if rolePush.SpawnFlash != testCase.spawns[index] {
				t.Fatalf("expected map%d %s spawn %+v got %+v", testCase.mapID, handle, testCase.spawns[index], rolePush.SpawnFlash)
			}
			expectedSourceQuery := "transp/flag2.swf"
			if (testCase.mapID == 127 && handle == "transp_131") || (testCase.mapID == 143 && handle == "transp_127") {
				expectedSourceQuery = "transp/fl.swf"
			}
			if rolePush.RoleID != "-3" || rolePush.DisplayName != "" || rolePush.SourceQuery != expectedSourceQuery {
				t.Fatalf("expected source transport role for %s, got %+v", handle, rolePush)
			}
		}
	}
}

func TestBuildTownBootstrapUsesCapturedHuangfengzhaiTransportData(t *testing.T) {
	cases := []struct {
		mapID         int
		mapName       string
		handles       []string
		spawns        []SpawnPoint
		sourceQueries []string
	}{
		{mapID: 122, mapName: "黄风寨口", handles: []string{"transp_121", "transp_146"}, spawns: []SpawnPoint{{X: 1460, Y: 520}, {X: 329, Y: 480}}, sourceQueries: []string{"transp/flag2.swf", "transp/fl.swf"}},
		{mapID: 146, mapName: "黄风寨_1", handles: []string{"transp_122", "transp_147", "transp_152"}, spawns: []SpawnPoint{{X: 1950, Y: 488}, {X: 55, Y: 507}, {X: 409, Y: 185}}, sourceQueries: []string{"transp/flag2.swf", "transp/flag2.swf", "transp/flag2.swf"}},
		{mapID: 147, mapName: "黄风寨_2", handles: []string{"transp_146", "transp_148"}, spawns: []SpawnPoint{{X: 2964, Y: 529}, {X: 31, Y: 561}}, sourceQueries: []string{"transp/flag2.swf", "transp/flag2.swf"}},
		{mapID: 148, mapName: "黄风寨_3", handles: []string{"transp_147", "transp_149"}, spawns: []SpawnPoint{{X: 1969, Y: 505}, {X: 424, Y: 379}}, sourceQueries: []string{"transp/flag2.swf", "transp/flag2.swf"}},
		{mapID: 149, mapName: "黄风寨_4", handles: []string{"transp_148", "transp_150"}, spawns: []SpawnPoint{{X: 30, Y: 550}, {X: 2280, Y: 351}}, sourceQueries: []string{"transp/flag2.swf", "transp/flag2.swf"}},
		{mapID: 150, mapName: "黄风寨_5", handles: []string{"transp_149", "transp_151", "transp_153"}, spawns: []SpawnPoint{{X: 1413, Y: 750}, {X: 2969, Y: 437}, {X: 151, Y: 273}}, sourceQueries: []string{"transp/flag2.swf", "transp/flag2.swf", "transp/flag2.swf"}},
		{mapID: 151, mapName: "黄风寨_6", handles: []string{"transp_150", "transp_152"}, spawns: []SpawnPoint{{X: 37, Y: 688}, {X: 2463, Y: 442}}, sourceQueries: []string{"transp/flag2.swf", "transp/flag2.swf"}},
		{mapID: 152, mapName: "黄风寨_7", handles: []string{"transp_146", "transp_151"}, spawns: []SpawnPoint{{X: 2963, Y: 600}, {X: 37, Y: 588}}, sourceQueries: []string{"transp/flag2.swf", "transp/flag2.swf"}},
		{mapID: 153, mapName: "黄风寨_8", handles: []string{"transp_150", "transp_154", "transp_156"}, spawns: []SpawnPoint{{X: 2869, Y: 444}, {X: 1401, Y: 706}, {X: 32, Y: 492}}, sourceQueries: []string{"transp/flag2.swf", "transp/flag2.swf", "transp/flag2.swf"}},
		{mapID: 154, mapName: "黄风寨_9", handles: []string{"transp_153", "transp_155", "transp_157"}, spawns: []SpawnPoint{{X: 2966, Y: 230}, {X: 37, Y: 513}, {X: 956, Y: 114}}, sourceQueries: []string{"transp/flag2.swf", "transp/flag2.swf", "transp/flag2.swf"}},
		{mapID: 155, mapName: "黄风寨_10", handles: []string{"transp_122", "transp_154"}, spawns: []SpawnPoint{{X: 158, Y: 455}, {X: 1941, Y: 556}}, sourceQueries: []string{"transp/fl.swf", "transp/flag2.swf"}},
		{mapID: 156, mapName: "黄风寨_11", handles: []string{"transp_153", "transp_157"}, spawns: []SpawnPoint{{X: 2466, Y: 494}, {X: 31, Y: 509}}, sourceQueries: []string{"transp/flag2.swf", "transp/flag2.swf"}},
		{mapID: 157, mapName: "黄风寨_12", handles: []string{"transp_154", "transp_156"}, spawns: []SpawnPoint{{X: 185, Y: 720}, {X: 2963, Y: 515}}, sourceQueries: []string{"transp/flag2.swf", "transp/flag2.swf"}},
	}

	for _, testCase := range cases {
		role := session.RoleSummary{
			RoleID:       "acct-test-role-huangfengzhai",
			DisplayName:  "测试女侠",
			Level:        20,
			MapID:        testCase.mapID,
			VisualRoleID: 1,
		}
		playerBase := session.PlayerBaseData{
			PlayerID:     "acct-test",
			RoleID:       role.RoleID,
			DisplayName:  role.DisplayName,
			Level:        role.Level,
			MapID:        testCase.mapID,
			VisualRoleID: role.VisualRoleID,
		}

		snapshot := BuildTownBootstrap(role, playerBase)
		if snapshot.LoadMap.MapID != itoa(testCase.mapID) || snapshot.LoadMap.MapName != testCase.mapName || snapshot.LoadMap.XMLURL != "xml/"+itoa(testCase.mapID)+".xml" {
			t.Fatalf("expected huangfengzhai map%d loadMap, got %+v", testCase.mapID, snapshot.LoadMap)
		}
		if snapshot.LoadMap.EnemyShow {
			t.Fatalf("expected captured huangfengzhai map%d to use visible monsters instead of enemyShow", testCase.mapID)
		}
		transports := make([]RolePush, 0, len(testCase.handles))
		for _, rolePush := range snapshot.CreateRoles {
			if rolePush.RoleID == "-3" {
				transports = append(transports, rolePush)
			}
		}
		if len(transports) != len(testCase.handles) {
			t.Fatalf("expected map%d transport count %d got %d", testCase.mapID, len(testCase.handles), len(transports))
		}
		for index, handle := range testCase.handles {
			rolePush := transports[index]
			if rolePush.Handle != handle {
				t.Fatalf("expected map%d transport handle %s at index %d got %s", testCase.mapID, handle, index, rolePush.Handle)
			}
			if rolePush.SpawnFlash != testCase.spawns[index] {
				t.Fatalf("expected map%d %s spawn %+v got %+v", testCase.mapID, handle, testCase.spawns[index], rolePush.SpawnFlash)
			}
			if rolePush.SourceQuery != testCase.sourceQueries[index] {
				t.Fatalf("expected map%d %s sourceQuery %s got %s", testCase.mapID, handle, testCase.sourceQueries[index], rolePush.SourceQuery)
			}
		}
	}
}

func TestBuildTownBootstrapIncludesFeixiandongEntranceTransport(t *testing.T) {
	role := session.RoleSummary{
		RoleID:       "acct-test-role-feixiandong",
		DisplayName:  "测试女侠",
		Level:        24,
		MapID:        18,
		VisualRoleID: 1,
	}
	playerBase := session.PlayerBaseData{
		PlayerID:     "acct-test",
		RoleID:       role.RoleID,
		DisplayName:  role.DisplayName,
		Level:        role.Level,
		MapID:        18,
		VisualRoleID: role.VisualRoleID,
	}

	snapshot := BuildTownBootstrap(role, playerBase)
	found := false
	for _, rolePush := range snapshot.CreateRoles {
		if rolePush.Handle != "transp_64" {
			continue
		}
		found = true
		if rolePush.RoleID != "-3" || rolePush.SourceQuery != "transp/flag2.swf" {
			t.Fatalf("expected map18 transp_64 source transport, got %+v", rolePush)
		}
	}
	if !found {
		t.Fatalf("expected map18 to include transp_64 entrance into feixiandong, got %+v", snapshot.CreateRoles)
	}
}

func TestResolveTownTransportAnswerUsesFeixiandongEntranceSpawn(t *testing.T) {
	destination, ok := ResolveTownTransportAnswer("transp_64", "goto")
	if !ok {
		t.Fatalf("expected transp_64 to resolve")
	}
	if destination.MapID != 64 || destination.Spawn != (SpawnPoint{X: 125, Y: 431}) {
		t.Fatalf("expected transp_64 to land at map64 entrance, got %+v", destination)
	}
}

func TestResolveTownTransportAnswerFromMapKeepsCapturedHandleSpawn(t *testing.T) {
	destination, ok := ResolveTownTransportAnswerFromMap(18, "transp_64", "goto")
	if !ok {
		t.Fatalf("expected map18 transp_64 to resolve")
	}
	if destination.MapID != 64 || destination.Spawn != (SpawnPoint{X: 125, Y: 431}) {
		t.Fatalf("expected map18 transp_64 to keep captured handle spawn, got %+v", destination)
	}
}

func TestResolveTownTransportAnswerFromMapPrefersDirectionalSpawnOverGlobalHandle(t *testing.T) {
	destination, ok := ResolveTownTransportAnswerFromMap(65, "transp_64", "goto")
	if !ok {
		t.Fatalf("expected map65 transp_64 to resolve")
	}
	if destination.MapID != 64 || destination.Spawn != (SpawnPoint{X: 950, Y: 430}) {
		t.Fatalf("expected map65 transp_64 to land near map64 return transport, got %+v", destination)
	}
}

func TestResolveTownTransportAnswerFromMapUsesCapturedGeneratedReturnSpawn(t *testing.T) {
	destination, ok := ResolveTownTransportAnswerFromMap(5, "transp_9", "goto")
	if !ok {
		t.Fatalf("expected map5 transp_9 to resolve")
	}
	if destination.MapID != 9 || destination.Spawn != (SpawnPoint{X: 160, Y: 512}) {
		t.Fatalf("expected map5 transp_9 to land near captured map9 return transport, got %+v", destination)
	}
}

func TestResolveTownTransportAnswerFromMapUsesCapturedRouteSpawn(t *testing.T) {
	destination, ok := ResolveTownTransportAnswerFromMap(64, "transp_65", "goto")
	if !ok {
		t.Fatalf("expected map64 transp_65 to resolve")
	}
	if destination.MapID != 65 || destination.Spawn != (SpawnPoint{X: 125, Y: 437}) {
		t.Fatalf("expected map64 transp_65 to land at captured map65 entrance, got %+v", destination)
	}
}

func TestResolveTownTransportAnswerFromMapInfersSpawnNearReturnTransport(t *testing.T) {
	destination, ok := ResolveTownTransportAnswerFromMap(70, "transp_72", "goto")
	if !ok {
		t.Fatalf("expected map70 transp_72 to resolve")
	}
	if destination.MapID != 72 || destination.Spawn != (SpawnPoint{X: 130, Y: 584}) {
		t.Fatalf("expected map70 transp_72 to infer spawn near map72 transp_70, got %+v", destination)
	}
}

func TestBuildTownBootstrapUsesCapturedHuangfengzhaiVisibleMonsters(t *testing.T) {
	cases := []struct {
		mapID       int
		roleCount   int
		questCount  int
		bossHandles map[string]RolePush
	}{
		{
			mapID:      149,
			roleCount:  10,
			questCount: 2,
			bossHandles: map[string]RolePush{
				"3218685759638239": {
					DisplayName: "黄风二寨主",
					Level:       19,
					Vocation:    "战士++",
					SourceQuery: "monstermap/hfscastellan.swf",
					SpawnFlash:  SpawnPoint{X: 1451, Y: 403},
					SourceNPCVisual: &SourceNPCVisual{
						MovieClipIRPath: "runtime/classic-monstermap/hfscastellan/hfscastellan-movieclip-ir",
					},
				},
			},
		},
		{
			mapID:      155,
			roleCount:  8,
			questCount: 2,
			bossHandles: map[string]RolePush{
				"2600686416056495": {
					DisplayName: "黄风大寨主",
					Level:       20,
					Vocation:    "战士++",
					SourceQuery: "monstermap/hfcastellan.swf",
					SpawnFlash:  SpawnPoint{X: 292, Y: 476},
					SourceNPCVisual: &SourceNPCVisual{
						MovieClipIRPath: "runtime/classic-monstermap/hfcastellan/hfcastellan-movieclip-ir",
					},
				},
				"2800686416057704": {
					DisplayName: "黄风寨夫人",
					Level:       20,
					Vocation:    "游侠++",
					SourceQuery: "monstermap/hflady.swf",
					SpawnFlash:  SpawnPoint{X: 300, Y: 553},
					SourceNPCVisual: &SourceNPCVisual{
						MovieClipIRPath: "runtime/classic-monstermap/hflady/hflady-movieclip-ir",
					},
				},
			},
		},
	}

	for _, testCase := range cases {
		role := session.RoleSummary{
			RoleID:       "acct-test-role-huangfengzhai-monster",
			DisplayName:  "测试女侠",
			Level:        20,
			MapID:        testCase.mapID,
			VisualRoleID: 1,
		}
		playerBase := session.PlayerBaseData{
			PlayerID:     "acct-test",
			RoleID:       role.RoleID,
			DisplayName:  role.DisplayName,
			Level:        role.Level,
			MapID:        testCase.mapID,
			VisualRoleID: role.VisualRoleID,
		}

		snapshot := BuildTownBootstrap(role, playerBase)
		if snapshot.LoadMap.EnemyShow {
			t.Fatalf("expected captured huangfengzhai map%d visible monsters instead of enemyShow", testCase.mapID)
		}
		if len(snapshot.CreateRoles) != testCase.roleCount || len(snapshot.QuestStates) != testCase.questCount {
			t.Fatalf("expected map%d roles=%d quests=%d got roles=%d quests=%d", testCase.mapID, testCase.roleCount, testCase.questCount, len(snapshot.CreateRoles), len(snapshot.QuestStates))
		}

		for _, rolePush := range snapshot.CreateRoles {
			expected, ok := testCase.bossHandles[rolePush.Handle]
			if !ok {
				continue
			}
			if rolePush.RoleID != "-2" || rolePush.Kind != "monster" {
				t.Fatalf("expected huangfengzhai boss %s to be visible monster, got %+v", rolePush.Handle, rolePush)
			}
			if rolePush.DisplayName != expected.DisplayName || rolePush.Level != expected.Level || rolePush.Vocation != expected.Vocation || rolePush.SourceQuery != expected.SourceQuery {
				t.Fatalf("expected huangfengzhai boss identity %+v got %+v", expected, rolePush)
			}
			if rolePush.SpawnFlash != expected.SpawnFlash {
				t.Fatalf("expected huangfengzhai boss spawn %+v got %+v", expected.SpawnFlash, rolePush.SpawnFlash)
			}
			if rolePush.Movement != nil {
				t.Fatalf("expected huangfengzhai boss %s bootstrap to omit movement, got %+v", rolePush.Handle, rolePush.Movement)
			}
			if rolePush.SourceNPCVisual == nil || rolePush.SourceNPCVisual.MovieClipIRPath != expected.SourceNPCVisual.MovieClipIRPath {
				t.Fatalf("expected huangfengzhai boss visual %+v got %+v", expected.SourceNPCVisual, rolePush.SourceNPCVisual)
			}
			delete(testCase.bossHandles, rolePush.Handle)
		}
		if len(testCase.bossHandles) != 0 {
			t.Fatalf("missing huangfengzhai boss checks for map%d: %+v", testCase.mapID, testCase.bossHandles)
		}
	}
}

func TestShihukuDungeonMapsUseCapturedTransportsAndVisibleMonsters(t *testing.T) {
	cases := []struct {
		mapID           int
		roleCount       int
		questCount      int
		expectedHandles map[string]RolePush
	}{
		{
			mapID:      158,
			roleCount:  5,
			questCount: 2,
			expectedHandles: map[string]RolePush{
				"5835621591157688": {
					DisplayName: "黑影",
					Level:       25,
					Vocation:    "游侠+",
					SourceQuery: "monstermap/blackshadow.swf",
					SpawnFlash:  SpawnPoint{X: 1053, Y: 450},
					SourceNPCVisual: &SourceNPCVisual{
						MovieClipIRPath: "runtime/classic-monstermap/blackshadow/blackshadow-movieclip-ir",
					},
				},
			},
		},
		{
			mapID:      167,
			roleCount:  8,
			questCount: 3,
			expectedHandles: map[string]RolePush{
				"7550622260838906": {
					DisplayName: "蚩颅王",
					Level:       30,
					Vocation:    "战士++",
					SourceQuery: "monstermap/chiluking.swf",
					SpawnFlash:  SpawnPoint{X: 2189, Y: 451},
					SourceNPCVisual: &SourceNPCVisual{
						MovieClipIRPath: "runtime/classic-monstermap/chiluking/chiluking-movieclip-ir",
					},
				},
			},
		},
	}

	for _, testCase := range cases {
		role := session.RoleSummary{
			RoleID:       "acct-test-role-shihuku-monster",
			DisplayName:  "测试女侠",
			Level:        25,
			MapID:        testCase.mapID,
			VisualRoleID: 1,
		}
		playerBase := session.PlayerBaseData{
			PlayerID:     "acct-test",
			RoleID:       role.RoleID,
			DisplayName:  role.DisplayName,
			Level:        role.Level,
			MapID:        testCase.mapID,
			VisualRoleID: role.VisualRoleID,
		}

		snapshot := BuildTownBootstrap(role, playerBase)
		if snapshot.LoadMap.EnemyShow {
			t.Fatalf("expected captured shihuku map%d visible monsters instead of enemyShow", testCase.mapID)
		}
		if len(snapshot.CreateRoles) != testCase.roleCount || len(snapshot.QuestStates) != testCase.questCount {
			t.Fatalf("expected map%d roles=%d quests=%d got roles=%d quests=%d", testCase.mapID, testCase.roleCount, testCase.questCount, len(snapshot.CreateRoles), len(snapshot.QuestStates))
		}

		rolesByHandle := map[string]RolePush{}
		for _, rolePush := range snapshot.CreateRoles {
			rolesByHandle[rolePush.Handle] = rolePush
		}
		for handle, expected := range testCase.expectedHandles {
			actual, ok := rolesByHandle[handle]
			if !ok {
				t.Fatalf("expected map%d captured monster handle %s, got roles=%+v", testCase.mapID, handle, snapshot.CreateRoles)
			}
			if actual.DisplayName != expected.DisplayName || actual.Level != expected.Level || actual.Vocation != expected.Vocation || actual.SourceQuery != expected.SourceQuery || actual.SpawnFlash != expected.SpawnFlash || actual.Kind != "monster" {
				t.Fatalf("expected map%d captured monster %+v, got %+v", testCase.mapID, expected, actual)
			}
			if actual.SourceNPCVisual == nil || actual.SourceNPCVisual.MovieClipIRPath != expected.SourceNPCVisual.MovieClipIRPath {
				t.Fatalf("expected map%d captured monster visual %+v, got %+v", testCase.mapID, expected.SourceNPCVisual, actual.SourceNPCVisual)
			}
		}
	}
}

func TestShihukuDungeonInstanceKeyAndTransportRoutes(t *testing.T) {
	if key, ok := DungeonInstanceKeyForMapID(158); !ok || key != session.DungeonInstanceShihuku {
		t.Fatalf("expected map158 shihuku dungeon key, got key=%s ok=%v", key, ok)
	}
	if key, ok := DungeonInstanceKeyForMapID(167); !ok || key != session.DungeonInstanceShihuku {
		t.Fatalf("expected map167 shihuku dungeon key, got key=%s ok=%v", key, ok)
	}

	cases := []struct {
		fromMapID int
		handle    string
		mapID     int
	}{
		{fromMapID: 36, handle: "transp_158", mapID: 158},
		{fromMapID: 158, handle: "transp_159", mapID: 159},
		{fromMapID: 159, handle: "transp_162", mapID: 162},
		{fromMapID: 167, handle: "transp_36", mapID: 36},
	}
	for _, testCase := range cases {
		destination, ok := ResolveTownTransportAnswerFromMap(testCase.fromMapID, testCase.handle, "goto")
		if !ok || destination.MapID != testCase.mapID {
			t.Fatalf("expected shihuku transport %+v to resolve, got destination=%+v ok=%v", testCase, destination, ok)
		}
	}
}

func TestBuildTownBootstrapUsesCapturedHuangfengzhaiMap147Roles(t *testing.T) {
	role := session.RoleSummary{
		RoleID:       "acct-test-role-huangfengzhai-147",
		DisplayName:  "测试女侠",
		Level:        20,
		MapID:        147,
		VisualRoleID: 1,
	}
	playerBase := session.PlayerBaseData{
		PlayerID:     "acct-test",
		RoleID:       role.RoleID,
		DisplayName:  role.DisplayName,
		Level:        role.Level,
		MapID:        role.MapID,
		VisualRoleID: role.VisualRoleID,
	}

	snapshot := BuildTownBootstrap(role, playerBase)
	if snapshot.LoadMap.MapID != "147" || snapshot.LoadMap.MapName != "黄风寨_2" {
		t.Fatalf("expected map147 bootstrap, got %+v", snapshot.LoadMap)
	}
	if len(snapshot.CreateRoles) != 6 || len(snapshot.QuestStates) != 2 {
		t.Fatalf("expected map147 captured roles=6 quests=2, got roles=%d quests=%d", len(snapshot.CreateRoles), len(snapshot.QuestStates))
	}

	expectedMonsters := map[string]struct {
		spawn SpawnPoint
	}{
		"7633284548716137": {spawn: SpawnPoint{X: 2073, Y: 350}},
		"7635284548716728": {spawn: SpawnPoint{X: 1196, Y: 438}},
		"7637284548717286": {spawn: SpawnPoint{X: 980, Y: 576}},
		"7639284548717945": {spawn: SpawnPoint{X: 438, Y: 430}},
	}
	for _, rolePush := range snapshot.CreateRoles {
		expected, ok := expectedMonsters[rolePush.Handle]
		if !ok {
			continue
		}
		if rolePush.RoleID != "-2" || rolePush.Kind != "monster" || rolePush.DisplayName != "蛮族刀客" || rolePush.SourceQuery != "monstermap/barbarianweapons.swf" {
			t.Fatalf("expected map147 barbarianweapons source monster, got %+v", rolePush)
		}
		if rolePush.SpawnFlash != expected.spawn {
			t.Fatalf("expected map147 monster %s spawn %+v, got %+v", rolePush.Handle, expected.spawn, rolePush.SpawnFlash)
		}
		if rolePush.Movement != nil {
			t.Fatalf("expected map147 monster %s bootstrap to omit movement, got %+v", rolePush.Handle, rolePush.Movement)
		}
		if rolePush.SourceNPCVisual == nil || rolePush.SourceNPCVisual.MovieClipIRPath != "runtime/classic-monstermap/barbarianweapons/barbarianweapons-movieclip-ir" {
			t.Fatalf("expected map147 monster %s barbarianweapons visual, got %+v", rolePush.Handle, rolePush.SourceNPCVisual)
		}
		delete(expectedMonsters, rolePush.Handle)
	}
	if len(expectedMonsters) != 0 {
		t.Fatalf("missing map147 captured monster checks: %+v", expectedMonsters)
	}
}

func TestBuildTownBootstrapUsesCapturedMapThreeData(t *testing.T) {
	role := session.RoleSummary{
		RoleID:       "acct-test-role-003",
		DisplayName:  "测试女侠",
		Level:        8,
		MapID:        3,
		VisualRoleID: 1,
	}
	playerBase := session.PlayerBaseData{
		PlayerID:     "acct-test",
		RoleID:       role.RoleID,
		DisplayName:  role.DisplayName,
		Level:        role.Level,
		MapID:        3,
		VisualRoleID: role.VisualRoleID,
	}

	snapshot := BuildTownBootstrap(role, playerBase)
	if snapshot.LoadMap.MapID != "3" || snapshot.LoadMap.MapName != "涧庭村" || snapshot.LoadMap.XMLURL != "xml/3.xml" {
		t.Fatalf("expected map3 loadMap, got %+v", snapshot.LoadMap)
	}
	if snapshot.LoadMap.EnemyShow {
		t.Fatal("expected map3 enemyShow to stay false")
	}
	if len(snapshot.CreateRoles) != 12 || len(snapshot.QuestStates) != 12 {
		t.Fatalf("expected map3 captured source roles, got roles=%d quests=%d", len(snapshot.CreateRoles), len(snapshot.QuestStates))
	}

	assertRole := func(handle string, name string, sourceQuery string, x int, y int) {
		t.Helper()
		for _, rolePush := range snapshot.CreateRoles {
			if rolePush.Handle != handle {
				continue
			}
			if rolePush.DisplayName != name || rolePush.SourceQuery != sourceQuery {
				t.Fatalf("expected %s/%s for %s, got name=%q source=%q", name, sourceQuery, handle, rolePush.DisplayName, rolePush.SourceQuery)
			}
			if rolePush.SpawnFlash.X != x || rolePush.SpawnFlash.Y != y {
				t.Fatalf("expected %s spawn %d,%d got %+v", handle, x, y, rolePush.SpawnFlash)
			}
			if rolePush.SourceNPCVisual == nil || rolePush.SourceNPCVisual.MovieClipIRPath == "" {
				t.Fatalf("expected source visual for %s, got %+v", handle, rolePush.SourceNPCVisual)
			}
			return
		}
		t.Fatalf("expected map3 role %s", handle)
	}

	assertRole("4970542616788530", "VIP大使", "npc/other/节日大使.swf", 372, 595)
	assertRole("4910542615957836", "申公烈", "npc/申公烈.swf", 675, 545)
	assertRole("4920542616052493", "火娥娘", "npc/火娥娘.swf", 1180, 409)
	assertRole("5010542616817526", "交易行管理员", "npc/交易行管理员.swf", 1612, 420)
	assertRole("4930542616250587", "铃铛", "npc/铃铛.swf", 1829, 426)
	assertRole("4990542616803864", "通天八卦炉<ma>", "npc/通天八卦炉.swf", 2139, 430)
	assertRole("5000542616815700", "妖术狐狸", "npc/狐狸.swf", 2370, 465)
	assertRole("4940542616468969", "叶眉", "npc/叶眉.swf", 2589, 465)
	assertRole("4950542616589339", "熊猫竹生", "npc/熊猫竹生.swf", 2862, 426)
	assertRole("4960542616750900", "介象", "npc/介象.swf", 3051, 442)
	assertRole("4980542616799322", "排行告示", "npc/公告牌.swf", 3325, 420)
	assertRole("transp_10", "", "transp/flag2.swf", 3566, 522)
}

func TestBuildAnswerSpeakMap3PandaHealerIncludesTreatment(t *testing.T) {
	speak := BuildAnswerSpeak("4950542616589339")
	if speak.Handle != "4950542616589339" || !strings.Contains(speak.Msg, "气力和精力") {
		t.Fatalf("expected panda healer dialogue, got %+v", speak)
	}
	if !hasAnswerOption(speak.Answers, "2", "进行治疗") {
		t.Fatalf("expected panda healer treatment answer, got %+v", speak.Answers)
	}
}

func TestBuildTownBootstrapUsesCapturedMap191NPCs(t *testing.T) {
	role := session.RoleSummary{
		RoleID:      "acct-test-role-191",
		DisplayName: "测试游侠",
		Level:       32,
		MapID:       191,
	}
	playerBase := session.PlayerBaseData{
		PlayerID:    "acct-test",
		RoleID:      role.RoleID,
		DisplayName: role.DisplayName,
		Level:       role.Level,
		MapID:       191,
	}

	snapshot := BuildTownBootstrap(role, playerBase)
	if snapshot.LoadMap.MapID != "191" || snapshot.LoadMap.XMLURL != "xml/191.xml" {
		t.Fatalf("expected map191 loadMap, got %+v", snapshot.LoadMap)
	}

	assertRole := func(handle string, name string, sourceQuery string, spriteName string, x int, y int) {
		t.Helper()
		for _, rolePush := range snapshot.CreateRoles {
			if rolePush.Handle != handle {
				continue
			}
			if rolePush.DisplayName != name || rolePush.SourceQuery != sourceQuery {
				t.Fatalf("expected %s/%s for %s, got name=%q source=%q", name, sourceQuery, handle, rolePush.DisplayName, rolePush.SourceQuery)
			}
			if rolePush.SpawnFlash.X != x || rolePush.SpawnFlash.Y != y {
				t.Fatalf("expected %s spawn %d,%d got %+v", handle, x, y, rolePush.SpawnFlash)
			}
			if rolePush.SourceNPCVisual == nil || rolePush.SourceNPCVisual.MovieClipIRPath != "runtime/classic-npc/movieclips/"+spriteName+"/"+spriteName+"-movieclip-ir" {
				t.Fatalf("expected %s source visual, got %+v", handle, rolePush.SourceNPCVisual)
			}
			return
		}
		t.Fatalf("expected map191 role %s", handle)
	}

	assertRole("6360542618722932", "虚中", "npc/虚中.swf", "xuzhong", 1082, 450)
	assertRole("6370542618853300", "虞莫", "npc/虞莫.swf", "yumo", 2483, 380)
	assertRole("6350542618650282", "汉雄", "npc/汉雄.swf", "hanxiong", 1514, 471)

	xuzhongSpeak := BuildAnswerSpeak("6360542618722932")
	if !hasAnswerOption(xuzhongSpeak.Answers, "2", "进行治疗") || !hasAnswerOption(xuzhongSpeak.Answers, "1", "查看商店") {
		t.Fatalf("expected Xuzhong healer/shop answers, got %+v", xuzhongSpeak.Answers)
	}

	reply := BuildAnswerReply("6350542618650282", "1", "6q2gs")
	if reply == nil || reply.MsgHandle != "6q2d_1" || !hasAnswerOption(reply.Answers, "6q2a_1_1", "<m/>我这就去。") {
		t.Fatalf("expected Hanxiong scout quest reply, got %+v", reply)
	}

	xuzhongQuestReply := BuildAnswerReply("6350542618650282", "1", "6q4gs")
	if xuzhongQuestReply == nil || xuzhongQuestReply.MsgHandle != "6q4d_1" || !hasAnswerOption(xuzhongQuestReply.Answers, "6q4a_1_1", "<m/>我这就去。") {
		t.Fatalf("expected Hanxiong Xuzhong quest reply, got %+v", xuzhongQuestReply)
	}

	meatReply := BuildAnswerReply("6360542618722932", "6q53d_1", "6q53a_1_1")
	if meatReply == nil || meatReply.MsgHandle != "6q53d_2" || !hasAnswerOption(meatReply.Answers, "6q53a_2_1", "<m/>好的，我这就去。") {
		t.Fatalf("expected Xuzhong meat quest continuation, got %+v", meatReply)
	}

	specialtyReply := BuildAnswerReply("6350542618650282", "1", "6q41os")
	if specialtyReply == nil || specialtyReply.MsgHandle != "6q41d_2" || !hasAnswerOption(specialtyReply.Answers, "6q41a_2_1", "<m/>是啊。") {
		t.Fatalf("expected Hanxiong specialty completion reply, got %+v", specialtyReply)
	}
}

func TestBuildTownBootstrapUsesCapturedBaiyuanTownNPCs(t *testing.T) {
	type expectedRole struct {
		mapID       int
		handle      string
		name        string
		sourceQuery string
		spriteName  string
		x           int
		y           int
		questState  int
	}
	roles := []expectedRole{
		{mapID: 168, handle: "4720542615647172", name: "白源镇告示栏", sourceQuery: "npc/公告牌.swf", spriteName: "gonggaopai", x: 724, y: 430, questState: 0},
		{mapID: 168, handle: "4710542615621525", name: "向隐", sourceQuery: "npc/向隐.swf", spriteName: "xiangyin", x: 1170, y: 434, questState: 2},
		{mapID: 168, handle: "4730542615655127", name: "通天八卦炉<ma>", sourceQuery: "npc/通天八卦炉.swf", spriteName: "bagualu", x: 1759, y: 440, questState: 0},
		{mapID: 169, handle: "5040542617131880", name: "袁天纲", sourceQuery: "npc/袁天纲.swf", spriteName: "yuantiangang", x: 805, y: 400, questState: 2},
		{mapID: 169, handle: "5060542617335713", name: "妖术狐狸", sourceQuery: "npc/狐狸.swf", spriteName: "huli", x: 1238, y: 450, questState: 0},
		{mapID: 169, handle: "5050542617322114", name: "白叟", sourceQuery: "npc/白叟.swf", spriteName: "baisou", x: 1618, y: 468, questState: 2},
		{mapID: 169, handle: "5070542617350720", name: "排行告示", sourceQuery: "npc/公告牌.swf", spriteName: "gonggaopai", x: 2208, y: 430, questState: 0},
		{mapID: 170, handle: "5310542617702520", name: "游氏子", sourceQuery: "npc/游氏子.swf", spriteName: "youshizi", x: 745, y: 423, questState: 2},
		{mapID: 170, handle: "5300542617580783", name: "苦虚无", sourceQuery: "npc/苦虚无.swf", spriteName: "kuxuwu", x: 1173, y: 430, questState: 4},
	}

	for _, expected := range roles {
		t.Run(expected.handle, func(t *testing.T) {
			role := session.RoleSummary{RoleID: "role-baiyuan-npc", DisplayName: "测试角色", Level: 30, MapID: expected.mapID}
			playerBase := session.PlayerBaseData{PlayerID: "player-baiyuan-npc", RoleID: role.RoleID, DisplayName: role.DisplayName, Level: role.Level, MapID: expected.mapID}
			snapshot := BuildTownBootstrap(role, playerBase)
			for _, rolePush := range snapshot.CreateRoles {
				if rolePush.Handle != expected.handle {
					continue
				}
				if rolePush.DisplayName != expected.name || rolePush.SourceQuery != expected.sourceQuery {
					t.Fatalf("expected %s/%s for %s, got name=%q source=%q", expected.name, expected.sourceQuery, expected.handle, rolePush.DisplayName, rolePush.SourceQuery)
				}
				if rolePush.SpawnFlash.X != expected.x || rolePush.SpawnFlash.Y != expected.y {
					t.Fatalf("expected %s spawn %d,%d got %+v", expected.handle, expected.x, expected.y, rolePush.SpawnFlash)
				}
				expectedIRPath := "runtime/classic-npc/movieclips/" + expected.spriteName + "/" + expected.spriteName + "-movieclip-ir"
				if rolePush.SourceNPCVisual == nil || rolePush.SourceNPCVisual.MovieClipIRPath != expectedIRPath {
					t.Fatalf("expected %s source visual %s, got %+v", expected.handle, expectedIRPath, rolePush.SourceNPCVisual)
				}
				for _, statePush := range snapshot.QuestStates {
					if statePush.Handle != expected.handle {
						continue
					}
					if statePush.State != expected.questState {
						t.Fatalf("expected %s quest state %d got %d", expected.handle, expected.questState, statePush.State)
					}
					return
				}
				t.Fatalf("expected %s quest state", expected.handle)
			}
			t.Fatalf("expected map%d role %s", expected.mapID, expected.handle)
		})
	}

	xiangyinSpeak := BuildAnswerSpeak("4710542615621525")
	if !hasAnswerOption(xiangyinSpeak.Answers, "2", "进行治疗") || !hasAnswerOption(xiangyinSpeak.Answers, "1", "查看商店") || !hasAnswerOption(xiangyinSpeak.Answers, "5q5gs", "<ml><m/>三七伤药") {
		t.Fatalf("expected Xiangyin healer/shop/quest answers, got %+v", xiangyinSpeak.Answers)
	}
	yuanSpeak := BuildAnswerSpeak("5040542617131880")
	if !hasAnswerOption(yuanSpeak.Answers, "1", "学习技能") || !hasAnswerOption(yuanSpeak.Answers, "5q25gs", "<m/>魔王降世") {
		t.Fatalf("expected Yuan Tiangang skill/quest answers, got %+v", yuanSpeak.Answers)
	}
	skillReply := BuildAnswerReply("5040542617131880", "1", "1")
	if skillReply == nil || skillReply.MsgHandle != "10" || !hasAnswerOption(skillReply.Answers, "7", "战士技能") {
		t.Fatalf("expected Yuan Tiangang skill category reply, got %+v", skillReply)
	}
	baisouSpeak := BuildAnswerSpeak("5050542617322114")
	if !hasAnswerOption(baisouSpeak.Answers, "6", "使用仓库") || !hasAnswerOption(baisouSpeak.Answers, "5q60ns", "<mn/>奇怪的考验【进行中】") {
		t.Fatalf("expected Baisou warehouse/quest answers, got %+v", baisouSpeak.Answers)
	}
	kuxuwuSpeak := BuildAnswerSpeak("5300542617580783")
	if !hasAnswerOption(kuxuwuSpeak.Answers, "2", "购买武器") || !hasAnswerOption(kuxuwuSpeak.Answers, "3", "购买护具") || !hasAnswerOption(kuxuwuSpeak.Answers, "5q8gs", "<ml><m/>铁矿之需") {
		t.Fatalf("expected Kuxuwu shop/quest answers, got %+v", kuxuwuSpeak.Answers)
	}
	youshiziSpeak := BuildAnswerSpeak("5310542617702520")
	if !hasAnswerOption(youshiziSpeak.Answers, "1", "道具商店") || !hasAnswerOption(youshiziSpeak.Answers, "5q44gs", "<m/>千里送信") {
		t.Fatalf("expected Youshizi shop/quest answers, got %+v", youshiziSpeak.Answers)
	}
	foxSpeak := BuildAnswerSpeak("5060542617335713")
	if !hasAnswerOption(foxSpeak.Answers, "3", "传送到【广青镇】(铜钱x500)") || !hasAnswerOption(foxSpeak.Answers, "5", "传送到【葬龙潭】(铜钱x500)") {
		t.Fatalf("expected Baiyuan fox transport answers, got %+v", foxSpeak.Answers)
	}
}

func TestBuildTownBootstrapIncludesPointCouponThiefOnlyOnActivityMaps(t *testing.T) {
	for _, mapID := range []int{84, 114, 115} {
		role := session.RoleSummary{
			RoleID:      "role-point-coupon-thief",
			DisplayName: "测试角色",
			Level:       15,
			MapID:       mapID,
		}
		playerBase := session.PlayerBaseData{
			PlayerID:    "player-point-coupon-thief",
			RoleID:      role.RoleID,
			DisplayName: role.DisplayName,
			Level:       role.Level,
			MapID:       mapID,
		}
		snapshot := BuildTownBootstrap(role, playerBase)
		var thief *RolePush
		for index := range snapshot.CreateRoles {
			if snapshot.CreateRoles[index].DisplayName == classicactivity.PointCouponThiefName {
				thief = &snapshot.CreateRoles[index]
				break
			}
		}
		if thief == nil {
			t.Fatalf("expected point coupon thief on map%d, got roles=%+v", mapID, snapshot.CreateRoles)
		}
		if thief.Kind != "monster" || thief.RoleID != "-2" || thief.SourceQuery != classicactivity.PointCouponThiefSourceQuery || thief.Level != 15 || thief.Vocation != "游侠" {
			t.Fatalf("expected point coupon thief source role on map%d, got %+v", mapID, thief)
		}
		if !classicactivity.IsPointCouponThiefHandle(snapshot.LoadMap.MapID, thief.Handle) {
			t.Fatalf("expected point coupon thief hourly handle on map%d, got %s", mapID, thief.Handle)
		}
		if thief.SourceNPCVisual == nil || thief.SourceNPCVisual.MovieClipIRPath != "runtime/classic-monstermap/militia/militia-movieclip-ir" {
			t.Fatalf("expected militia monstermap visual on map%d, got %+v", mapID, thief.SourceNPCVisual)
		}
	}

	role := session.RoleSummary{RoleID: "role-novice", DisplayName: "测试角色", Level: 1, MapID: 1}
	playerBase := session.PlayerBaseData{PlayerID: "player-novice", RoleID: role.RoleID, DisplayName: role.DisplayName, Level: role.Level, MapID: 1}
	snapshot := BuildTownBootstrap(role, playerBase)
	for _, rolePush := range snapshot.CreateRoles {
		if rolePush.DisplayName == classicactivity.PointCouponThiefName {
			t.Fatalf("point coupon thief must not spawn in novice village, got %+v", rolePush)
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

func TestBuildTownTransferBootstrapMarksCapturedBambooEnemyShow(t *testing.T) {
	capturedMaps := []struct {
		mapID   int
		mapName string
	}{
		{mapID: 84, mapName: "竹林_1"},
		{mapID: 85, mapName: "竹林_2"},
		{mapID: 86, mapName: "竹林_3"},
		{mapID: 87, mapName: "竹林_4"},
		{mapID: 88, mapName: "竹林_5"},
		{mapID: 90, mapName: "竹林_7"},
		{mapID: 97, mapName: "竹林_10"},
	}

	for _, testCase := range capturedMaps {
		role := session.RoleSummary{
			RoleID:       "acct-test-role-bamboo",
			DisplayName:  "测试女侠",
			Level:        8,
			MapID:        testCase.mapID,
			VisualRoleID: 1,
		}
		playerBase := session.PlayerBaseData{
			PlayerID:     "acct-test",
			RoleID:       role.RoleID,
			DisplayName:  role.DisplayName,
			Level:        role.Level,
			MapID:        testCase.mapID,
			VisualRoleID: role.VisualRoleID,
		}

		snapshot, ok := BuildTownTransferBootstrap(role, playerBase, testCase.mapID, SpawnPoint{X: 1000, Y: 300})
		if !ok {
			t.Fatalf("expected map%d transfer bootstrap to be supported", testCase.mapID)
		}
		if snapshot.LoadMap.MapID != itoa(testCase.mapID) || snapshot.LoadMap.MapName != testCase.mapName || snapshot.LoadMap.XMLURL != "xml/"+itoa(testCase.mapID)+".xml" {
			t.Fatalf("expected bamboo map%d loadMap, got %+v", testCase.mapID, snapshot.LoadMap)
		}
		if !snapshot.LoadMap.EnemyShow {
			t.Fatalf("expected captured bamboo map%d enemyShow to be true", testCase.mapID)
		}
	}
}

func TestBuildTownTransferBootstrapMarksCapturedPlainEnemyShow(t *testing.T) {
	capturedMaps := []struct {
		mapID   int
		mapName string
	}{
		{mapID: 34, mapName: "平原_3"},
		{mapID: 35, mapName: "平原_4"},
		{mapID: 36, mapName: "平原_5"},
		{mapID: 37, mapName: "平原_6"},
		{mapID: 39, mapName: "平原_8"},
		{mapID: 40, mapName: "平原_9"},
		{mapID: 41, mapName: "平原_10"},
		{mapID: 43, mapName: "平原_12"},
		{mapID: 44, mapName: "平原_13"},
		{mapID: 48, mapName: "平原_14"},
		{mapID: 49, mapName: "平原_15"},
		{mapID: 50, mapName: "平原_16"},
		{mapID: 51, mapName: "平原_17"},
		{mapID: 52, mapName: "平原_18"},
	}

	for _, testCase := range capturedMaps {
		role := session.RoleSummary{
			RoleID:       "acct-test-role-plain",
			DisplayName:  "测试女侠",
			Level:        18,
			MapID:        testCase.mapID,
			VisualRoleID: 1,
		}
		playerBase := session.PlayerBaseData{
			PlayerID:     "acct-test",
			RoleID:       role.RoleID,
			DisplayName:  role.DisplayName,
			Level:        role.Level,
			MapID:        testCase.mapID,
			VisualRoleID: role.VisualRoleID,
		}

		snapshot, ok := BuildTownTransferBootstrap(role, playerBase, testCase.mapID, SpawnPoint{X: 1000, Y: 600})
		if !ok {
			t.Fatalf("expected map%d transfer bootstrap to be supported", testCase.mapID)
		}
		if snapshot.LoadMap.MapID != itoa(testCase.mapID) || snapshot.LoadMap.MapName != testCase.mapName || snapshot.LoadMap.XMLURL != "xml/"+itoa(testCase.mapID)+".xml" {
			t.Fatalf("expected plain map%d loadMap, got %+v", testCase.mapID, snapshot.LoadMap)
		}
		if !snapshot.LoadMap.EnemyShow {
			t.Fatalf("expected captured plain map%d enemyShow to be true", testCase.mapID)
		}
	}
}

func TestBuildTownTransferBootstrapMarksCapturedShuiliandongEnemyShow(t *testing.T) {
	capturedMaps := []struct {
		mapID   int
		mapName string
	}{
		{mapID: 131, mapName: "水帘洞_1"},
		{mapID: 132, mapName: "水帘洞_2"},
		{mapID: 133, mapName: "水帘洞_3"},
		{mapID: 137, mapName: "水帘洞_7"},
		{mapID: 140, mapName: "水帘洞_10"},
		{mapID: 141, mapName: "水帘洞_11"},
		{mapID: 142, mapName: "水帘洞_12"},
		{mapID: 143, mapName: "水帘洞_13"},
		{mapID: 144, mapName: "水帘洞_14"},
		{mapID: 145, mapName: "水帘洞_15"},
	}

	for _, testCase := range capturedMaps {
		role := session.RoleSummary{
			RoleID:       "acct-test-role-shuiliandong-enemy",
			DisplayName:  "测试女侠",
			Level:        20,
			MapID:        testCase.mapID,
			VisualRoleID: 1,
		}
		playerBase := session.PlayerBaseData{
			PlayerID:     "acct-test",
			RoleID:       role.RoleID,
			DisplayName:  role.DisplayName,
			Level:        role.Level,
			MapID:        testCase.mapID,
			VisualRoleID: role.VisualRoleID,
		}

		snapshot, ok := BuildTownTransferBootstrap(role, playerBase, testCase.mapID, SpawnPoint{X: 1000, Y: 600})
		if !ok {
			t.Fatalf("expected map%d transfer bootstrap to be supported", testCase.mapID)
		}
		if snapshot.LoadMap.MapID != itoa(testCase.mapID) || snapshot.LoadMap.MapName != testCase.mapName || snapshot.LoadMap.XMLURL != "xml/"+itoa(testCase.mapID)+".xml" {
			t.Fatalf("expected shuiliandong map%d loadMap, got %+v", testCase.mapID, snapshot.LoadMap)
		}
		if snapshot.LoadMap.EnemyShow {
			t.Fatalf("expected captured shuiliandong map%d to use visible monsters instead of enemyShow", testCase.mapID)
		}
	}
}

func TestBuildTownTransferBootstrapRestoresBambooCollectionPoints(t *testing.T) {
	cases := []struct {
		mapID       int
		mapName     string
		handle      string
		displayName string
		spawn       SpawnPoint
	}{
		{mapID: 89, mapName: "竹林_6", handle: "2810542613719308", displayName: "金银花采集点", spawn: SpawnPoint{X: 1242, Y: 451}},
		{mapID: 91, mapName: "竹林_8", handle: "2050542611677774", displayName: "黄连采集点", spawn: SpawnPoint{X: 1755, Y: 452}},
	}

	for _, testCase := range cases {
		role := session.RoleSummary{
			RoleID:       "acct-test-role-collection",
			DisplayName:  "测试女侠",
			Level:        8,
			MapID:        testCase.mapID,
			VisualRoleID: 1,
		}
		playerBase := session.PlayerBaseData{
			PlayerID:     "acct-test",
			RoleID:       role.RoleID,
			DisplayName:  role.DisplayName,
			Level:        role.Level,
			MapID:        testCase.mapID,
			VisualRoleID: role.VisualRoleID,
		}

		snapshot, ok := BuildTownTransferBootstrap(role, playerBase, testCase.mapID, SpawnPoint{X: 1000, Y: 300})
		if !ok {
			t.Fatalf("expected map%d transfer bootstrap to be supported", testCase.mapID)
		}
		if snapshot.LoadMap.MapID != itoa(testCase.mapID) || snapshot.LoadMap.MapName != testCase.mapName || snapshot.LoadMap.XMLURL != "xml/"+itoa(testCase.mapID)+".xml" {
			t.Fatalf("expected collection map%d loadMap, got %+v", testCase.mapID, snapshot.LoadMap)
		}
		if snapshot.LoadMap.EnemyShow {
			t.Fatalf("expected collection map%d enemyShow to stay false", testCase.mapID)
		}
		var rolePush *RolePush
		for index := range snapshot.CreateRoles {
			if snapshot.CreateRoles[index].Handle == testCase.handle {
				rolePush = &snapshot.CreateRoles[index]
				break
			}
		}
		if rolePush == nil {
			t.Fatalf("expected collection role %s on map%d, got %+v", testCase.handle, testCase.mapID, snapshot.CreateRoles)
		}
		if rolePush.Handle != testCase.handle || rolePush.Kind != "collection" || rolePush.DisplayName != testCase.displayName {
			t.Fatalf("expected collection role %+v, got %+v", testCase, *rolePush)
		}
		if rolePush.SourceNPCVisual == nil || rolePush.SourceNPCVisual.MovieClipIRPath != "runtime/classic-npc/movieclips/flag2/flag2-movieclip-ir" {
			t.Fatalf("expected collection role to reuse flag2 visual, got %+v", rolePush.SourceNPCVisual)
		}
		if rolePush.SpawnFlash != testCase.spawn {
			t.Fatalf("expected collection map%d spawn %+v, got %+v", testCase.mapID, testCase.spawn, rolePush.SpawnFlash)
		}
	}
}

func TestBuildTownTransferBootstrapDoesNotExposeUncapturedWildBattleMaps(t *testing.T) {
	role := session.RoleSummary{
		RoleID:       "acct-test-role-009",
		DisplayName:  "测试女侠",
		Level:        8,
		MapID:        9,
		VisualRoleID: 1,
	}
	playerBase := session.PlayerBaseData{
		PlayerID:     "acct-test",
		RoleID:       role.RoleID,
		DisplayName:  role.DisplayName,
		Level:        role.Level,
		MapID:        9,
		VisualRoleID: role.VisualRoleID,
	}

	snapshot, ok := BuildTownTransferBootstrap(role, playerBase, 9, SpawnPoint{X: 1000, Y: 300})
	if !ok {
		t.Fatal("expected map9 transfer bootstrap to be supported")
	}
	if snapshot.LoadMap.MapID != "9" || snapshot.LoadMap.XMLURL != "xml/9.xml" {
		t.Fatalf("expected map9 loadMap, got %+v", snapshot.LoadMap)
	}
	if snapshot.LoadMap.EnemyShow {
		t.Fatal("expected uncaptured map9 enemyShow to stay false")
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
