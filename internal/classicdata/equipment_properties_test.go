package classicdata

import (
	"encoding/json"
	"testing"
)

func TestEquipmentMetadataPreservesStructuredUnitsAndRuntimeBoundary(t *testing.T) {
	metadataOnly, runtime := 0, 0
	for _, row := range MustRows(TableItem) {
		if row["item_type"] != "equip" {
			continue
		}
		if ItemIsMetadataOnly(row) {
			metadataOnly++
			if row["captured_metadata"] == "" {
				t.Fatalf("metadata-only equipment %s lacks provenance", row["name"])
			}
		} else {
			runtime++
		}
		for _, key := range []string{"equipment_attributes", "refinement_steps"} {
			var values []map[string]interface{}
			if err := json.Unmarshal([]byte(row[key]), &values); err != nil || values == nil {
				t.Fatalf("equipment %s has invalid %s: %v", row["name"], key, err)
			}
		}
	}
	if metadataOnly != 183 || runtime != 380 {
		t.Fatalf("equipment metadata/runtime = %d/%d, want 183/380", metadataOnly, runtime)
	}
	for _, name := range []string{"雷兽法杖", "珍元面甲", "真.炎爆面甲", "智慧戒指", "竹斗笠"} {
		row, found, err := FindItemByName(name)
		if err != nil || !found || !ItemIsMetadataOnly(row) || row["description"] == "" {
			t.Fatalf("missing captured equipment metadata %s: found=%t err=%v", name, found, err)
		}
	}
	row, _, _ := FindItemByName("狰狞神骑")
	var steps []struct {
		Level   int `json:"level"`
		Effects []struct {
			Code   string  `json:"code"`
			Amount float64 `json:"amount"`
			Unit   string  `json:"unit"`
		} `json:"effects"`
	}
	if err := json.Unmarshal([]byte(row["refinement_steps"]), &steps); err != nil {
		t.Fatal(err)
	}
	for _, step := range steps {
		for _, effect := range step.Effects {
			if step.Level == 20 && effect.Code == "9" && effect.Amount == 20 && effect.Unit == "percent" {
				return
			}
		}
	}
	t.Fatal("captured hit +20% must retain its unit")
}
