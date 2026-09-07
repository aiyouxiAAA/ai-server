package session

import "testing"

func TestMetadataOnlyEquipmentDoesNotBecomeDefaultGrant(t *testing.T) {
	for _, name := range []string{"雷兽法杖", "智慧戒指", "珍元面甲"} {
		if _, found := classicDataRoleItemTemplate(name); found {
			t.Fatalf("captured instance %s must not become a new default equipment grant", name)
		}
	}
	if _, found := classicDataRoleItemTemplate("御影护甲"); !found {
		t.Fatal("existing runtime equipment must remain available")
	}
}
