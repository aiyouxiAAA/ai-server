package classicdata

import (
	"strings"
	"testing"
)

func TestGeneratedClassicTablesLoad(t *testing.T) {
	tables, err := LoadAllTables()
	if err != nil {
		t.Fatalf("LoadAllTables() error = %v", err)
	}
	for _, name := range KnownTables {
		table, ok := tables[name]
		if !ok {
			t.Fatalf("missing table %s", name)
		}
		if table.RowCount <= 0 || len(table.Rows) != table.RowCount {
			t.Fatalf("table %s row count mismatch: rowCount=%d rows=%d", name, table.RowCount, len(table.Rows))
		}
	}
}

func TestGeneratedClassicTablesContainKnownRows(t *testing.T) {
	itemTable := MustLoadTable(TableItem)
	assertHasRow(t, itemTable, "name", "馒头")
	assertHasRow(t, itemTable, "name", "铜钱")

	skillTable := MustLoadTable(TableSkill)
	assertHasRow(t, skillTable, "label", "投毒")
	assertHasRow(t, skillTable, "label", "强力飞镖")

	monsterSkillTable := MustLoadTable(TableMonsterSkill)
	assertHasRow(t, monsterSkillTable, "monster_skill_id", "monster-blackshadow-piece-attack")
	assertHasRow(t, monsterSkillTable, "source_action_label", "goldhit")

	professionTable := MustLoadTable(TableProfession)
	assertHasRow(t, professionTable, "name", "战士")
	assertHasRow(t, professionTable, "name", "术士")
	assertHasRow(t, professionTable, "name", "游侠")

	buffTable := MustLoadTable(TableBuff)
	assertHasRow(t, buffTable, "name", "中毒")
	assertHasRow(t, buffTable, "name", "眩晕")
	assertBuffDescription(t, buffTable, "seal", "作用时间内对象无法使用技能", "fixed")
	assertBuffDescription(t, buffTable, "hotblood", "提升物理攻击&0;同时降低物理防御", "fixed")
	assertBuffDescription(t, buffTable, "slow", "降低对象X点命中和Y点回避", "dynamic")
	assertBuffDescription(t, buffTable, "internal-injury", "降低对象X点物理攻击和Y点魔法攻击力", "dynamic")
	assertBuffDescription(t, buffTable, "freeze", "", "unknown")

	monsterTable := MustLoadTable(TableMonster)
	assertHasRow(t, monsterTable, "monster_id", "wild-49-1914555684754474")
	assertHasRow(t, monsterTable, "name", "蛤蟆精")

	attributeTable := MustLoadTable(TableAttribute)
	assertHasRow(t, attributeTable, "attribute_id", "phy_atk")
	assertHasRow(t, attributeTable, "attribute_id", "dodge")

	effectTable := MustLoadTable(TableEffect)
	assertHasRow(t, effectTable, "effect_id", "poison-tick")
	assertHasRow(t, effectTable, "effect_id", "fighting-spirit-phy-atk")

	effectSourceTable := MustLoadTable(TableEffectSource)
	assertHasRow(t, effectSourceTable, "source_link_id", "skill-tou-du-poison")
	assertHasRow(t, effectSourceTable, "source_link_id", "equipment-feiyu-dagger-internal-injury")

	dropTable := MustLoadTable(TableDrop)
	assertHasRow(t, dropTable, "map_id", "49")
}

func TestGeneratedClassicItemEquipmentRowsKeepSourceDescriptions(t *testing.T) {
	itemTable := MustLoadTable(TableItem)
	equipmentRows := 0
	sourceDescriptionRows := 0
	for _, row := range itemTable.Rows {
		if row["item_type"] != "equip" {
			continue
		}
		equipmentRows++
		if row["description"] == "精炼潜质: [精炼+1" {
			t.Fatalf("equipment row %s still has truncated source description", row["name"])
		}
		if strings.HasPrefix(row["description"], "f_i_") {
			sourceDescriptionRows++
		}
	}
	if equipmentRows == 0 || sourceDescriptionRows != equipmentRows {
		t.Fatalf("equipment source description coverage = %d/%d, want all equipment rows", sourceDescriptionRows, equipmentRows)
	}
}

func TestGeneratedClassicItemTablePromotesTrialSealFromCapturedEquipment(t *testing.T) {
	item, ok, err := FindItemByName("试炼印")
	if err != nil {
		t.Fatalf("FindItemByName(试炼印) error = %v", err)
	}
	if !ok {
		t.Fatal("expected captured 试炼印 item row")
	}
	if item["item_type"] != "equip" || item["icon"] != "1605.png" || !strings.HasPrefix(item["description"], "f_i_试炼印^f9e000") {
		t.Fatalf("expected captured trial-seal equipment metadata, got %+v", item)
	}
}

func TestClassicDataLookupsReturnDetachedRows(t *testing.T) {
	item, ok, err := FindItemByName("馒头")
	if err != nil {
		t.Fatalf("FindItemByName() error = %v", err)
	}
	if !ok {
		t.Fatalf("FindItemByName() did not find 馒头")
	}
	if item["icon"] != "0.png" {
		t.Fatalf("FindItemByName() icon = %q, want 0.png", item["icon"])
	}

	item["icon"] = "changed.png"
	itemAgain, ok, err := FindItemByName("馒头")
	if err != nil {
		t.Fatalf("FindItemByName() second error = %v", err)
	}
	if !ok || itemAgain["icon"] != "0.png" {
		t.Fatalf("FindItemByName() returned shared row: ok=%v icon=%q", ok, itemAgain["icon"])
	}

	skill, ok, err := FindSkillByLabel("投毒")
	if err != nil {
		t.Fatalf("FindSkillByLabel() error = %v", err)
	}
	if !ok || skill["source_action_label"] != "w3/drugAtk" {
		t.Fatalf("FindSkillByLabel() = %v %v, want 投毒 w3/drugAtk", ok, skill)
	}

	monsterSkill, ok, err := FindMonsterSkillByID("monster-chiluking-goldhit")
	if err != nil {
		t.Fatalf("FindMonsterSkillByID() error = %v", err)
	}
	if !ok || monsterSkill["source_action_label"] != "goldhit" || monsterSkill["target"] != "enemy" {
		t.Fatalf("FindMonsterSkillByID() = %v %v, want chiluking goldhit enemy skill", ok, monsterSkill)
	}

	monsterSkills, err := FindMonsterSkillRowsByDisplayURL("monstermap/chiluking.swf")
	if err != nil {
		t.Fatalf("FindMonsterSkillRowsByDisplayURL() error = %v", err)
	}
	if len(monsterSkills) < 2 {
		t.Fatalf("FindMonsterSkillRowsByDisplayURL(chiluking) returned %d rows, want at least 2", len(monsterSkills))
	}

	profession, ok, err := FindProfessionByID("warrior")
	if err != nil {
		t.Fatalf("FindProfessionByID() error = %v", err)
	}
	if !ok || profession["name"] != "战士" {
		t.Fatalf("FindProfessionByID() = %v %v, want 战士", ok, profession)
	}

	buff, ok, err := FindBuffByID("poison")
	if err != nil {
		t.Fatalf("FindBuffByID() error = %v", err)
	}
	if !ok || buff["name"] != "中毒" {
		t.Fatalf("FindBuffByID() = %v %v, want 中毒", ok, buff)
	}

	monster, ok, err := FindMonsterByID("wild-49-1914555684754474")
	if err != nil {
		t.Fatalf("FindMonsterByID() error = %v", err)
	}
	if !ok || monster["name"] != "盗贼" {
		t.Fatalf("FindMonsterByID() = %v %v, want 盗贼", ok, monster)
	}

	grasslandMonster, found, err := FindMonsterByID("wild-210-caoba-glassyx-lv38")
	if err != nil {
		t.Fatalf("FindMonsterByID(grassland) error = %v", err)
	}
	if !found || grasslandMonster["name"] != "草刺槐" || grasslandMonster["map_id"] != "210" || grasslandMonster["max_hp"] != "1530" || grasslandMonster["max_mp"] != "1034" || grasslandMonster["attack"] != "391" {
		t.Fatalf("FindMonsterByID(grassland) = %v %v, want 草刺槐 map210 HP1530 MP1034 attack391", found, grasslandMonster)
	}

	drops, err := FindDropRowsByMapID("49")
	if err != nil {
		t.Fatalf("FindDropRowsByMapID() error = %v", err)
	}
	if len(drops) == 0 {
		t.Fatalf("FindDropRowsByMapID() returned no rows")
	}

	attribute, ok, err := FindAttributeByID("phy_atk")
	if err != nil {
		t.Fatalf("FindAttributeByID() error = %v", err)
	}
	if !ok || attribute["battle_cell_field"] != "attack" {
		t.Fatalf("FindAttributeByID() = %v %v, want battle attack field", ok, attribute)
	}

	effects, err := FindEffectRowsByBuffID("poison")
	if err != nil {
		t.Fatalf("FindEffectRowsByBuffID() error = %v", err)
	}
	if len(effects) < 3 {
		t.Fatalf("FindEffectRowsByBuffID(poison) returned %d rows, want at least 3", len(effects))
	}

	sources, err := FindEffectSourceRowsBySourceID("skill-tou-du")
	if err != nil {
		t.Fatalf("FindEffectSourceRowsBySourceID() error = %v", err)
	}
	if len(sources) != 1 || sources[0]["buff_id"] != "poison" {
		t.Fatalf("FindEffectSourceRowsBySourceID(skill-tou-du) = %+v, want poison source", sources)
	}
}

func assertHasRow(t *testing.T, table Table, key string, value string) {
	t.Helper()
	for _, row := range table.Rows {
		if row[key] == value {
			return
		}
	}
	t.Fatalf("table %s missing row with %s=%s", table.Name, key, value)
}

func assertBuffDescription(t *testing.T, table Table, buffID string, description string, kind string) {
	t.Helper()
	for _, row := range table.Rows {
		if row["buff_id"] != buffID {
			continue
		}
		if row["description"] != description {
			t.Fatalf("buff %s description = %q, want %q", buffID, row["description"], description)
		}
		if row["description_kind"] != kind {
			t.Fatalf("buff %s description_kind = %q, want %q", buffID, row["description_kind"], kind)
		}
		return
	}
	t.Fatalf("table %s missing buff_id=%s", table.Name, buffID)
}
