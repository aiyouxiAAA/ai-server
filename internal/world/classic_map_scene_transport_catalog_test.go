package world

import (
	"testing"

	"ai-server/internal/classicdata"
)

func TestAuthoredSceneTransportPreservesSourceContracts(t *testing.T) {
	row, ok := classicdata.FindClassicMapSceneTransportSpawn(13, "transp_19")
	if !ok || row.Source != "nine_wilds_terrain_20260909" {
		t.Fatal("missing authorized original terrain branch")
	}
	entries := buildClassicSceneTransports(13, []classicdata.ClassicMapSceneTransportSpawn{row}, nil)
	if len(entries) != 1 || entries[0].RoleID != "-3" || entries[0].Dialogue != &sourceTransportDialogue || entries[0].SpawnFlash != (SpawnPoint{X: row.SpawnX, Y: row.SpawnY}) {
		t.Fatalf("authored exit lost source entity or dialogue contract: %+v", entries)
	}
	for _, scenario := range []string{"untagged addition", "wrong visual", "lost legacy exit"} {
		t.Run(scenario, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("invalid catalog must fail explicitly")
				}
			}()
			candidate := row
			var legacy []sourceNPCEntry
			switch scenario {
			case "untagged addition":
				candidate.Source = ""
			case "wrong visual":
				candidate.SourceQuery = "transp/unknown.swf"
			case "lost legacy exit":
				legacy = []sourceNPCEntry{{Handle: "transp_14"}}
			}
			buildClassicSceneTransports(13, []classicdata.ClassicMapSceneTransportSpawn{candidate}, legacy)
		})
	}
}

func TestSceneTransportExportKeepsCanonicalAuthoredCoordinates(t *testing.T) {
	count := 0
	for _, exported := range ExportClassicMapSceneTransportCatalogRows() {
		row, ok := classicdata.FindClassicMapSceneTransportSpawn(exported.MapID, exported.Handle)
		if !ok || exported.Source != row.Source || exported.SpawnFlash != (SpawnPoint{X: row.SpawnX, Y: row.SpawnY}) || exported.TargetMapID != row.TargetMapID || exported.TargetSpawn != (SpawnPoint{X: row.TargetSpawnX, Y: row.TargetSpawnY}) {
			t.Fatalf("export would undo canonical transport: %+v", exported)
		}
		if row.Source == "nine_wilds_terrain_20260909" {
			count++
		}
	}
	if count != 90 {
		t.Fatalf("authored export coverage = %d, want 90", count)
	}
}
