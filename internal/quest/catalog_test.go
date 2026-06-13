package quest

import "testing"

func TestCatalogParsesCapturedQuestRewards(t *testing.T) {
	info, ok := FindByID("capture-003")
	if !ok {
		t.Fatal("expected captured quest capture-003")
	}
	if info.Reward.Experience != 1000 {
		t.Fatalf("expected capture-003 exp reward 1000, got %+v", info.Reward)
	}
	if len(info.Reward.Items) != 1 || info.Reward.Items[0].Name != "铜钱" || info.Reward.Items[0].Count != 200 || info.Reward.Items[0].Display != "163.png" {
		t.Fatalf("expected capture-003 copper reward, got %+v", info.Reward.Items)
	}
}

func TestCatalogParsesQuestRequirements(t *testing.T) {
	info, ok := FindByID("capture-032")
	if !ok {
		t.Fatal("expected capture-032 in catalog")
	}
	if len(info.Requirements) != 1 || info.Requirements[0].Name != "肉" || info.Requirements[0].Count != 5 || info.Requirements[0].Display != "70.png" {
		t.Fatalf("expected capture-032 meat requirement, got %+v", info.Requirements)
	}
}

func TestCatalogParsesSkillAndOptionalRewards(t *testing.T) {
	skillInfo, ok := FindByID("capture-007")
	if !ok {
		t.Fatal("expected captured quest capture-007")
	}
	if skillInfo.Reward.Experience != 300 || len(skillInfo.Reward.Skills) != 1 || skillInfo.Reward.Skills[0].Name != "密斩" {
		t.Fatalf("expected capture-007 exp and skill reward, got %+v", skillInfo.Reward)
	}

	optionalInfo, ok := FindByID("capture-058")
	if !ok {
		t.Fatal("expected captured quest capture-058")
	}
	if optionalInfo.Reward.Experience != 5000 {
		t.Fatalf("expected capture-058 exp reward 5000, got %+v", optionalInfo.Reward)
	}
	if len(optionalInfo.Reward.Items) != 0 {
		t.Fatalf("expected capture-058 [g] section to avoid required-item rewards, got %+v", optionalInfo.Reward.Items)
	}
	if len(optionalInfo.Reward.OptionalItems) != 6 {
		t.Fatalf("expected capture-058 six optional pet rewards, got %+v", optionalInfo.Reward.OptionalItems)
	}
}

func TestAllCatalogRowsHaveGrantRewardMarker(t *testing.T) {
	for _, info := range All() {
		if info.Reward.Experience <= 0 && len(info.Reward.Items) == 0 && len(info.Reward.Skills) == 0 {
			t.Fatalf("expected parsed grant reward for %s %s, got %+v", info.ID, info.Title, info.Reward)
		}
	}
}
