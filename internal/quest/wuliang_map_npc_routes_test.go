package quest

import "testing"

func TestWuliangMapNPCQuestRoutesLinkCatalogRows(t *testing.T) {
	routes := WuliangMapNPCQuestRoutes()
	find := func(questID string, handle string, msgHandle string) (WuliangMapNPCQuestRoute, bool) {
		for _, route := range routes {
			if route.QuestID == questID && route.NPCHandle == handle && route.MsgHandle == msgHandle {
				return route, true
			}
		}
		return WuliangMapNPCQuestRoute{}, false
	}

	scout, ok := find("capture-186", "6350542618650282", "6q2d_1")
	if !ok || scout.MapID != 191 || scout.MapName != "乌梁营地_2" || scout.NPCName != "汉雄" || scout.AnswerHandle != "6q2a_1_1" {
		t.Fatalf("expected scout route to link Hanxiong on map191, got %+v ok=%v", scout, ok)
	}
	furnitureStart, ok := find("capture-232", "6190542618476150", "6q45d_1")
	if !ok || furnitureStart.MapID != 190 || furnitureStart.NPCName != "丑四品" {
		t.Fatalf("expected furniture start route to link Chou Sipin on map190, got %+v ok=%v", furnitureStart, ok)
	}
	furnitureComplete, ok := find("capture-232", "6370542618853300", "6q45d_2")
	if !ok || furnitureComplete.MapID != 191 || furnitureComplete.NPCName != "虞莫" {
		t.Fatalf("expected furniture completion route to link Yumo on map191, got %+v ok=%v", furnitureComplete, ok)
	}
}

func TestClassicMapNPCQuestRoutesResolveCatalogRelations(t *testing.T) {
	routes := ClassicMapNPCQuestRoutes()
	if len(routes) == 0 {
		t.Fatal("expected captured quest routes to link to the Classic map/NPC catalog")
	}
	foundDafoRoute := false
	for _, route := range routes {
		if route.MapID <= 0 || route.NPCHandle == "" || route.NPCName == "" {
			t.Fatalf("expected complete map/NPC relation, got %+v", route)
		}
		if route.QuestID == "capture-024" && route.MapID == 2 && route.NPCHandle == "4090542614314425" && route.NPCName == "丑六品" {
			foundDafoRoute = true
		}
	}
	if !foundDafoRoute {
		t.Fatal("expected Dafo village quest route to link map2 NPC Chou Liupin")
	}
}
