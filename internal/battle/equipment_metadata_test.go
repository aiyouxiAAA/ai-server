package battle

import "testing"

func TestMetadataOnlyEquipmentDoesNotExpandRewardPools(t *testing.T) {
	names := mustLoadSourceBattleRewardEquipmentItemNames()
	if len(names) != 380 {
		t.Fatalf("runtime equipment names = %d, want original 380", len(names))
	}
	for _, pool := range mustLoadSourceBattleRewardEquipmentPools() {
		for _, name := range pool {
			if name == "蛤蟆精护腿" || name == "蚩颅王面甲" || name == "机木胸甲" {
				t.Fatalf("metadata import changed runtime reward pool: %s", name)
			}
		}
	}
}
