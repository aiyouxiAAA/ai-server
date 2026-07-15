package classicdata

import "testing"

func TestClassicWorldCatalogHasCompleteMapAndStaticNPCCoverage(t *testing.T) {
	maps := ClassicMaps()
	if len(maps) != 484 {
		t.Fatalf("expected 484 Classic map catalog rows, got %d", len(maps))
	}
	staticNPCs := 0
	for _, mapEntry := range maps {
		if _, ok := FindClassicMap(mapEntry.ID); !ok {
			t.Fatalf("expected map %d to resolve from its own catalog", mapEntry.ID)
		}
		for _, npc := range ClassicMapNPCSpawns(mapEntry.ID) {
			staticNPCs++
			resolved, ok := FindClassicMapNPCSpawn(mapEntry.ID, npc.Handle)
			if !ok || resolved != npc {
				t.Fatalf("expected map %d NPC %s to resolve, got %+v ok=%v", mapEntry.ID, npc.Handle, resolved, ok)
			}
		}
	}
	if staticNPCs != 81 {
		t.Fatalf("expected 81 static Classic NPC catalog rows, got %d", staticNPCs)
	}
}

func TestClassicWorldCatalogLinksSceneTransportsCollectionsAndMapPageTransfers(t *testing.T) {
	maps := ClassicMaps()
	sceneTransports := 0
	collectionPoints := 0
	for _, mapEntry := range maps {
		for _, transport := range ClassicMapSceneTransportSpawns(mapEntry.ID) {
			sceneTransports++
			resolved, ok := FindClassicMapSceneTransportSpawn(mapEntry.ID, transport.Handle)
			if !ok || resolved != transport {
				t.Fatalf("expected map %d scene transport %s to resolve, got %+v ok=%v", mapEntry.ID, transport.Handle, resolved, ok)
			}
			if _, ok := FindClassicMap(transport.TargetMapID); !ok {
				t.Fatalf("scene transport %d/%s points at missing target map %d", mapEntry.ID, transport.Handle, transport.TargetMapID)
			}
		}
		for _, collection := range ClassicMapCollectionPoints(mapEntry.ID) {
			collectionPoints++
			resolved, ok := FindClassicMapCollectionPoint(mapEntry.ID, collection.Handle)
			if !ok || resolved != collection {
				t.Fatalf("expected map %d collection %s to resolve, got %+v ok=%v", mapEntry.ID, collection.Handle, resolved, ok)
			}
		}
	}
	if sceneTransports != 375 {
		t.Fatalf("expected 375 scene transport catalog rows, got %d", sceneTransports)
	}
	if collectionPoints != 2 {
		t.Fatalf("expected 2 collection catalog rows, got %d", collectionPoints)
	}

	pageTransfers := ClassicMapPageTransfers()
	if len(pageTransfers) != 414 {
		t.Fatalf("expected 414 map-page transfer catalog rows, got %d", len(pageTransfers))
	}
	for _, transfer := range pageTransfers {
		if _, ok := FindClassicMap(transfer.MapID); !ok {
			t.Fatalf("map-page transfer %s/%s points at missing map %d", transfer.TransferType, transfer.TransferKey, transfer.MapID)
		}
	}

	transport, ok := FindClassicMapSceneTransportSpawn(3, "transp_10")
	if !ok || transport.TargetMapID != 10 || transport.TargetSpawnX != 165 || transport.TargetSpawnY != 540 || transport.Protocol != "CrossRole" {
		t.Fatalf("expected captured map3 transp_10 CrossRole row, got %+v ok=%v", transport, ok)
	}
	collection, ok := FindClassicMapCollectionPoint(89, "2810542613719308")
	if !ok || collection.RewardItemName != "金银花" || collection.Protocol != "Collection" {
		t.Fatalf("expected map89 collection relation, got %+v ok=%v", collection, ok)
	}
}

func TestWuliangMapCatalogLinksNPCSpawns(t *testing.T) {
	maps := WuliangMaps()
	if len(maps) != 2 || maps[0].ID != 190 || maps[1].ID != 191 {
		t.Fatalf("expected two captured Wuliang maps, got %+v", maps)
	}
	map190, ok := FindWuliangMap(190)
	if !ok || map190.Name != "乌梁营地_1" || map190.XMLURL != "xml/190.xml" || map190.DefaultSpawnX != 1000 || map190.DefaultSpawnY != 600 {
		t.Fatalf("expected map190 source row, got %+v ok=%v", map190, ok)
	}
	map191NPCs := WuliangMapNPCSpawns(191)
	if len(map191NPCs) != 3 {
		t.Fatalf("expected three map191 NPC spawns, got %+v", map191NPCs)
	}
	npc, ok := FindWuliangMapNPCSpawn("6370542618853300")
	if !ok || npc.DisplayName != "虞莫" || npc.MapID != 191 || npc.SpriteName != "yumo" || npc.SpawnX != 2483 || npc.SpawnY != 380 {
		t.Fatalf("expected captured Yumo spawn row, got %+v ok=%v", npc, ok)
	}
	linkedMap, ok := FindWuliangMapForNPCHandle("6190542618476150")
	if !ok || linkedMap.ID != 190 {
		t.Fatalf("expected Chou Sipin to link to map190, got %+v ok=%v", linkedMap, ok)
	}
}
