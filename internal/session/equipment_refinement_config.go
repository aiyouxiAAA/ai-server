package session

import (
	_ "embed"
	"encoding/csv"
	"strconv"
	"strings"

	"math/rand"
)

//go:embed config/classic-equipment-refinement.csv
var classicEquipmentRefinementCSV string

// ClassicEquipmentRefinementRule is a local balancing rule, not recovered original-server data.
type ClassicEquipmentRefinementRule struct {
	ItemName          string
	MinRefineLevel    int
	MaxRefineLevel    int
	MinEquipmentLevel int
	SuccessRateBps    int
	FailureFloor      int
	DesignStatus      string
	Note              string
}

var classicEquipmentRefinementRules = mustLoadClassicEquipmentRefinementRules(classicEquipmentRefinementCSV)

func defaultEquipmentRefinementRoll(max int) int {
	return rand.Intn(max)
}

func classicEquipmentRefinementRuleFor(itemName string, refineLevel int, equipmentLevel int) (ClassicEquipmentRefinementRule, bool) {
	for _, rule := range classicEquipmentRefinementRules {
		if rule.ItemName != strings.TrimSpace(itemName) {
			continue
		}
		if refineLevel < rule.MinRefineLevel || refineLevel > rule.MaxRefineLevel || equipmentLevel < rule.MinEquipmentLevel {
			continue
		}
		return rule, true
	}
	return ClassicEquipmentRefinementRule{}, false
}

func mustLoadClassicEquipmentRefinementRules(source string) []ClassicEquipmentRefinementRule {
	reader := csv.NewReader(strings.NewReader(source))
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		panic("read classic equipment refinement config: " + err.Error())
	}
	if len(records) < 2 {
		panic("classic equipment refinement config has no rules")
	}

	rules := make([]ClassicEquipmentRefinementRule, 0, len(records)-1)
	for rowIndex, record := range records[1:] {
		if len(record) != 8 {
			panic("classic equipment refinement config invalid column count at row " + strconv.Itoa(rowIndex+2))
		}
		parseInt := func(column int) int {
			value, parseErr := strconv.Atoi(strings.TrimSpace(record[column]))
			if parseErr != nil {
				panic("classic equipment refinement config invalid integer at row " + strconv.Itoa(rowIndex+2))
			}
			return value
		}
		rule := ClassicEquipmentRefinementRule{
			ItemName:          strings.TrimSpace(record[0]),
			MinRefineLevel:    parseInt(1),
			MaxRefineLevel:    parseInt(2),
			MinEquipmentLevel: parseInt(3),
			SuccessRateBps:    parseInt(4),
			FailureFloor:      parseInt(5),
			DesignStatus:      strings.TrimSpace(record[6]),
			Note:              strings.TrimSpace(record[7]),
		}
		if rule.ItemName == "" || rule.MinRefineLevel < 0 || rule.MaxRefineLevel < rule.MinRefineLevel ||
			rule.MinEquipmentLevel < 0 || rule.SuccessRateBps <= 0 || rule.SuccessRateBps > 10000 ||
			rule.FailureFloor < 0 || rule.FailureFloor > rule.MinRefineLevel || rule.DesignStatus != "temporary_non_original" {
			panic("classic equipment refinement config invalid rule at row " + strconv.Itoa(rowIndex+2))
		}
		rules = append(rules, rule)
	}
	return rules
}
