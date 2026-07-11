package session

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"ai-server/internal/classicdata"
)

var classicRoleLevelToExp = []int{
	0, 20, 100, 280, 640, 1240, 2200, 3670, 5830, 8980, 13480, 19860,
	28740, 40960, 57620, 86800, 123860, 169760, 225050, 290250, 365850,
	452310, 550060, 659500, 781000, 914900, 1061510, 1221110, 1393950,
	1580250, 1780200, 1993960, 2221660, 2463400, 2719250, 2989250,
	3273410, 3571710, 3884100, 4210500, 4550800, 4904860, 5272510,
	5653550, 6047750, 6454850, 6874560, 7306560, 7750500, 8206000,
	8672650, 9150010, 9637610, 10134950, 10642050, 11158930, 11685610,
	12222110, 12768450, 13324650, 13890730, 14466710, 17495810, 19433350,
	21461610, 23581130, 25792450, 28096110, 30492650, 32982610, 38244950,
	43887450, 49914430, 56330210, 63139110, 73120450, 88121560, 108121405,
	138121246, 173121478, 218121478, 283121478, 368121478, 473121478,
	593121478, 733121478, 893121478, 1073121478, 1273121478, 1503121478,
	1763121478, 2053121478, 2373121478, 2723121478, 3103121478, 3523121478,
	3983121478, 4483121478, 5023121478, 5603121478,
	6603121478, 7403121478, 8403121478, 9503121478, 10703121478,
	12003121478, 13403121478, 14903121478, 16503121478, 18203121478,
	20003121478, 21903121478, 23903121478, 26003121478, 28203121478,
	30503121478, 33003121478, 35703121478, 38603121478, 41703121478,
	45203121478,
}

func encodeRoleAppearance(appearance RoleAppearance) (string, error) {
	if len(appearance) == 0 {
		return "", nil
	}

	data, err := json.Marshal(appearance)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func decodeRoleAppearance(raw string) (RoleAppearance, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	var appearance RoleAppearance
	if err := json.Unmarshal([]byte(raw), &appearance); err != nil {
		return nil, err
	}

	return appearance, nil
}

func encodeRoleSkills(skills []RoleSkill) (string, error) {
	if len(skills) == 0 {
		return "", nil
	}

	data, err := json.Marshal(skills)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func decodeRoleSkills(raw string) ([]RoleSkill, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	var skills []RoleSkill
	if err := json.Unmarshal([]byte(raw), &skills); err != nil {
		return nil, err
	}

	result := make([]RoleSkill, 0, len(skills))
	for _, skill := range skills {
		normalized := normalizeRoleSkill(skill)
		if normalized.Name != "" {
			result = append(result, normalized)
		}
	}
	return result, nil
}

func encodeRoleFastPanel(entries []RoleFastPanelEntry) (string, error) {
	normalized := normalizeRoleFastPanel(entries)
	if len(normalized) == 0 {
		return "", nil
	}

	data, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func decodeRoleFastPanel(raw string) ([]RoleFastPanelEntry, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	var entries []RoleFastPanelEntry
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		return nil, err
	}

	return normalizeRoleFastPanel(entries), nil
}

func encodeRoleCurrencies(currencies RoleCurrencies) (string, error) {
	normalized := normalizeRoleCurrencies(currencies)
	if len(normalized) == 0 {
		return "", nil
	}

	data, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func encodeRoleItems(items []RoleItem) (string, error) {
	normalized := cloneRoleItems(items)
	if len(normalized) == 0 {
		return "", nil
	}

	data, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func decodeRoleCurrencies(raw string) (RoleCurrencies, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	var currencies RoleCurrencies
	if err := json.Unmarshal([]byte(raw), &currencies); err != nil {
		return nil, err
	}

	return normalizeRoleCurrencies(currencies), nil
}

func decodeRoleItems(raw string) ([]RoleItem, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	var items []RoleItem
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, err
	}

	result := make([]RoleItem, 0, len(items))
	for _, item := range items {
		normalized := normalizeRoleItem(item)
		if normalized.Name != "" {
			result = append(result, normalized)
		}
	}
	return result, nil
}

func encodeRoleTownBuffs(buffs []RoleTownBuff) (string, error) {
	normalized := normalizeRoleTownBuffs(buffs)
	if len(normalized) == 0 {
		return "", nil
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func decodeRoleTownBuffs(raw string) ([]RoleTownBuff, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var buffs []RoleTownBuff
	if err := json.Unmarshal([]byte(raw), &buffs); err != nil {
		return nil, err
	}
	return normalizeRoleTownBuffs(buffs), nil
}

func encodeRoleState(roleState *RoleState) (string, error) {
	if roleState == nil {
		return "", nil
	}
	data, err := json.Marshal(roleState)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func decodeRoleState(raw string) (*RoleState, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var roleState RoleState
	if err := json.Unmarshal([]byte(raw), &roleState); err != nil {
		return nil, err
	}
	return &roleState, nil
}

func encodeRolePhysique(rolePhysique *RolePhysique) (string, error) {
	if rolePhysique == nil {
		return "", nil
	}
	data, err := json.Marshal(rolePhysique)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func decodeRolePhysique(raw string) (*RolePhysique, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var rolePhysique RolePhysique
	if err := json.Unmarshal([]byte(raw), &rolePhysique); err != nil {
		return nil, err
	}
	if rolePhysique.ResPros == nil {
		rolePhysique.ResPros = []string{}
	}
	return &rolePhysique, nil
}

func encodeDungeonInstances(instances map[string]DungeonInstanceState) (string, error) {
	if len(instances) == 0 {
		return "", nil
	}
	data, err := json.Marshal(instances)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func decodeDungeonInstances(raw string) (map[string]DungeonInstanceState, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var instances map[string]DungeonInstanceState
	if err := json.Unmarshal([]byte(raw), &instances); err != nil {
		return nil, err
	}
	result := make(map[string]DungeonInstanceState, len(instances))
	for key, state := range instances {
		normalizedKey := strings.TrimSpace(key)
		if normalizedKey == "" {
			continue
		}
		state.DefeatedVisibleMonsterHandles = normalizeStringList(state.DefeatedVisibleMonsterHandles)
		result[normalizedKey] = state
	}
	return result, nil
}

func cloneRoleSkills(skills []RoleSkill) []RoleSkill {
	if len(skills) == 0 {
		return []RoleSkill{}
	}

	result := make([]RoleSkill, len(skills))
	copy(result, skills)
	return result
}

func cloneRoleFastPanel(entries []RoleFastPanelEntry) []RoleFastPanelEntry {
	if len(entries) == 0 {
		return []RoleFastPanelEntry{}
	}

	result := make([]RoleFastPanelEntry, len(entries))
	copy(result, entries)
	return result
}

func cloneRoleTownBuffs(buffs []RoleTownBuff) []RoleTownBuff {
	if len(buffs) == 0 {
		return []RoleTownBuff{}
	}

	result := make([]RoleTownBuff, len(buffs))
	copy(result, buffs)
	return result
}

func cloneRoleCurrencies(currencies RoleCurrencies) RoleCurrencies {
	if len(currencies) == 0 {
		return RoleCurrencies{}
	}

	result := make(RoleCurrencies, len(currencies))
	for name, count := range currencies {
		result[name] = count
	}
	return result
}

func cloneRoleItems(items []RoleItem) []RoleItem {
	if len(items) == 0 {
		return []RoleItem{}
	}

	result := make([]RoleItem, len(items))
	copy(result, items)
	return result
}

func cloneDungeonInstanceState(state DungeonInstanceState) DungeonInstanceState {
	state.DefeatedVisibleMonsterHandles = cloneStringList(state.DefeatedVisibleMonsterHandles)
	return state
}

func cloneDungeonInstances(instances map[string]DungeonInstanceState) map[string]DungeonInstanceState {
	if len(instances) == 0 {
		return nil
	}
	result := make(map[string]DungeonInstanceState, len(instances))
	for key, state := range instances {
		result[key] = cloneDungeonInstanceState(state)
	}
	return result
}

func cloneStringList(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	result := make([]string, len(values))
	copy(result, values)
	return result
}

func normalizeStringList(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" || seen[normalized] {
			continue
		}
		seen[normalized] = true
		result = append(result, normalized)
	}
	return result
}

func containsString(values []string, needle string) bool {
	needle = strings.TrimSpace(needle)
	for _, value := range values {
		if strings.TrimSpace(value) == needle {
			return true
		}
	}
	return false
}

func normalizeRoleSkill(skill RoleSkill) RoleSkill {
	skill.Name = strings.TrimSpace(skill.Name)
	skill.Type = strings.TrimSpace(skill.Type)
	skill.Icon = strings.TrimSpace(skill.Icon)
	skill.Description = strings.TrimSpace(skill.Description)
	if skill.Level <= 0 {
		skill.Level = 1
	}
	if skill.Type == "" {
		skill.Type = "oneO"
	}
	skill.MaxLevel = normalizeSkillMaxLevel(skill.MaxLevel)
	return skill
}

func normalizeRoleFastPanelEntry(entry RoleFastPanelEntry) RoleFastPanelEntry {
	entry.Type = strings.TrimSpace(entry.Type)
	entry.Name = strings.TrimSpace(entry.Name)
	return entry
}

func normalizeRoleFastPanel(entries []RoleFastPanelEntry) []RoleFastPanelEntry {
	if len(entries) == 0 {
		return []RoleFastPanelEntry{}
	}

	byIndex := make(map[int]RoleFastPanelEntry, len(entries))
	for _, entry := range entries {
		normalized := normalizeRoleFastPanelEntry(entry)
		if normalized.Index < 0 || normalized.Index >= defaultRoleFastPanelSlotCount || normalized.Type == "" || normalized.Name == "" {
			continue
		}
		byIndex[normalized.Index] = normalized
	}

	result := make([]RoleFastPanelEntry, 0, len(byIndex))
	for _, entry := range byIndex {
		result = append(result, entry)
	}
	sort.SliceStable(result, func(left int, right int) bool {
		return result[left].Index < result[right].Index
	})
	return result
}

func filterRoleFastPanelEntries(entries []RoleFastPanelEntry, skills []RoleSkill) []RoleFastPanelEntry {
	result := make([]RoleFastPanelEntry, 0, len(entries))
	for _, entry := range normalizeRoleFastPanel(entries) {
		if canSetRoleFastPanelEntry(entry, skills) {
			result = append(result, entry)
		}
	}
	return result
}

func canSetRoleFastPanelEntry(entry RoleFastPanelEntry, skills []RoleSkill) bool {
	entry = normalizeRoleFastPanelEntry(entry)
	if entry.Type == "item" {
		return true
	}
	if entry.Type != "skill" {
		return false
	}
	for _, skill := range skills {
		skill = normalizeRoleSkill(skill)
		if skill.Name != entry.Name {
			continue
		}
		return skill.Type != "被动技能"
	}
	return false
}

func normalizeSkillMaxLevel(maxLevel int) int {
	if maxLevel <= 0 {
		return 5
	}
	if maxLevel < 1 {
		return 1
	}
	return maxLevel
}

func normalizeRoleItem(item RoleItem) RoleItem {
	item.Handle = strings.TrimSpace(item.Handle)
	item.Type = strings.TrimSpace(item.Type)
	item.Name = strings.TrimSpace(item.Name)
	item.ItemType = strings.TrimSpace(item.ItemType)
	item.Display = strings.TrimSpace(item.Display)
	item.Description = strings.TrimSpace(item.Description)
	item.Owner = strings.TrimSpace(item.Owner)
	if item.PetState != nil {
		state := *item.PetState
		state.PetID = strings.TrimSpace(state.PetID)
		if state.Level < 1 {
			state.Level = 1
		}
		if state.Exp < 0 {
			state.Exp = 0
		}
		if state.Fullness < 0 {
			state.Fullness = 0
		}
		if state.Fullness > 100 {
			state.Fullness = 100
		}
		item.PetState = &state
	}
	item = fillMissingRoleItemTemplateFields(item)
	if item.Count < 0 {
		item.Count = 0
	}
	if item.Level < 0 {
		item.Level = 0
	}
	if item.EndTime < 0 {
		item.EndTime = 0
	}
	if item.ItemLevel < 0 {
		item.ItemLevel = 0
	}
	return item
}

func fillMissingRoleItemTemplateFields(item RoleItem) RoleItem {
	if item.Name == "" {
		return item
	}
	template, ok := CapturedRoleItemTemplate(item.Name)
	if !ok {
		return item
	}
	usesTemplateDisplay := item.Display == "" || item.Display == template.Display
	usesTemplateDescription := item.Description == "" ||
		item.Description == genericCollectionRewardDescription(item.Name) ||
		item.Description == template.Description ||
		isStaleEquipmentTemplateDescription(item, template)
	if item.Display == "" {
		item.Display = template.Display
	}
	if item.ItemType == "" || (item.ItemType == "own" && template.ItemType != "" && template.ItemType != item.ItemType && usesTemplateDisplay && usesTemplateDescription) {
		item.ItemType = template.ItemType
	}
	if item.Description == "" || item.Description == genericCollectionRewardDescription(item.Name) || isStaleEquipmentTemplateDescription(item, template) {
		item.Description = template.Description
	}
	if item.ItemLevel <= 0 || (template.ItemLevel > item.ItemLevel && usesTemplateDisplay && usesTemplateDescription && item.ItemType == template.ItemType) {
		item.ItemLevel = template.ItemLevel
	}
	return item
}

func isStaleEquipmentTemplateDescription(item RoleItem, template RoleItem) bool {
	if template.ItemType != "equip" || !strings.HasPrefix(template.Description, "f_i_") {
		return false
	}
	if item.ItemType != "" && item.ItemType != "equip" {
		return false
	}
	description := strings.TrimSpace(item.Description)
	if description == "" || description == template.Description {
		return false
	}
	// Non-source text such as truncated "精炼潜质: [精炼+1" must be refreshed.
	if !strings.HasPrefix(description, "f_i_") {
		return true
	}
	// Keep instance-specific f_i_ payloads (pet level, remaining energy, owner fields).
	// Only treat as stale when core identity markers are missing.
	// Do not require &21@ — pets/法宝 often omit level requirements.
	if !strings.Contains(description, "&24@") {
		return true
	}
	if strings.Contains(template.Description, "&108@") && !strings.Contains(description, "&108@") {
		return true
	}
	if strings.Contains(template.Description, "&21@") && !strings.Contains(description, "&21@") {
		return true
	}
	return false
}

func genericCollectionRewardDescription(name string) string {
	return "f_i_" + name + "&24@材料&25@99&20@采集获得的材料。"
}

func normalizeRoleCurrencies(currencies RoleCurrencies) RoleCurrencies {
	result := make(RoleCurrencies)
	for name, count := range currencies {
		trimmedName := strings.TrimSpace(name)
		if trimmedName == "" || count <= 0 {
			continue
		}
		result[trimmedName] = count
	}
	return result
}

func defaultRoleCurrencies() RoleCurrencies {
	return RoleCurrencies{
		"铜钱":  defaultCopper,
		"银元宝": defaultSilver,
	}
}

func normalizeRoleVocation(vocation string) (string, bool) {
	switch strings.TrimSpace(vocation) {
	case "", defaultRoleVoc:
		return defaultRoleVoc, true
	case "战士":
		return "战士", true
	case "术士":
		return "术士", true
	case "游侠":
		return "游侠", true
	default:
		return "", false
	}
}

func defaultRoleSkills() []RoleSkill {
	return []RoleSkill{
		{
			Name:        "密斩",
			Level:       1,
			Type:        "oneE",
			Icon:        "426.png",
			Description: "f_s_密斩&9@单体·攻击&7@3&10@单刀/单斧&22@战斗&2@5&4@提升40%的物理伤害",
		},
		{
			Name:        "普通攻击",
			Level:       1,
			Type:        "oneE",
			Icon:        "7.png",
			Description: "f_s_普通攻击^ffffff&9@单体·攻击&10@通用&22@战斗&5@给予对手普通的物理攻击.",
		},
	}
}

const defaultRoleFastPanelSlotCount = 10

func defaultRoleFastPanel() []RoleFastPanelEntry {
	return []RoleFastPanelEntry{
		{Index: 0, Type: "skill", Name: "普通攻击"},
		{Index: 1, Type: "skill", Name: "密斩"},
	}
}

func capturedWoodcutterRoleSkills() []RoleSkill {
	return []RoleSkill{
		{
			Name:        "魔力突刺",
			Level:       1,
			Type:        "oneE",
			Icon:        "258.png",
			Description: "f_s_魔力突刺^5BC46D&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@20&4@造成敌人100%的物理伤害&0;并追加80%的魔法伤害",
		},
		{
			Name:        "奥义.暗杀者",
			Level:       1,
			Type:        "oneE",
			Icon:        "262.png",
			Description: "f_s_奥义.暗杀者^00ccff&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@26&4@<font color='#00cc00'>特殊发动条件:需要3格魂元</font><br>提升180%的物理伤害",
		},
		{
			Name:        "爆发力",
			Level:       5,
			Type:        "null",
			Icon:        "228.png",
			Description: "f_s_爆发力^ffffff&9@被动&8@游侠 &10@通用&20@190",
		},
		{
			Name:        "疾风刺",
			Level:       1,
			Type:        "oneE",
			Icon:        "259.png",
			Description: "f_s_疾风刺^5BC46D&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@20&4@对敌人造成40%的物理伤害&0;击中敌人时有92%的机率使对方进入迟钝状态(削减对方50%的命中和回避)3回合<br><font color='#00cc00'>叠加施放将削弱其造成迟钝的功效</font>",
		},
		{
			Name:        "强力飞镖",
			Level:       3,
			Type:        "oneE",
			Icon:        "261.png",
			Description: "f_s_强力飞镖^ffffff&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@24&4@<font color='#00cc00'>特殊发动条件:需要【飞镖x1】</font><br>进攻时提高50%（无视防御）的物理攻击力",
		},
		{
			Name:        "武器娴熟",
			Level:       5,
			Type:        "null",
			Icon:        "226.png",
			Description: "f_s_武器娴熟^ffffff&9@被动&8@游侠 &10@通用&12@20",
		},
		{
			Name:        "投毒",
			Level:       1,
			Type:        "oneE",
			Icon:        "166.png",
			Description: "f_s_投毒^5BC46D&9@单体·状态&8@游侠 &10@匕首&22@战斗&2@16&4@<font color='#00cc00'>特殊发动条件:需要【毒药x1】<br>叠加施放将削弱其造成中毒的功效</font><br>有80%的机率使敌人中毒，4回合内降低对方15%的物理防御和魔法防御&0;每回合使敌人损失气力为物理攻击的20%~25%",
		},
		{
			Name:        "解毒术",
			Level:       1,
			Type:        "own",
			Icon:        "260.png",
			Description: "f_s_解毒术^ffffff&9@单体·状态&8@游侠 &10@匕首&22@战斗&2@20&4@解除自身中毒状态",
		},
		{
			Name:        "普通攻击",
			Level:       1,
			Type:        "oneE",
			Icon:        "7.png",
			Description: "f_s_普通攻击^ffffff&9@单体·攻击&10@通用&22@战斗&5@给予对手普通的物理攻击.",
		},
		{
			Name:        "精神力",
			Level:       5,
			Type:        "null",
			Icon:        "230.png",
			Description: "f_s_精神力^ffffff&9@被动&8@游侠 &10@通用&18@12",
		},
		{
			Name:        "幻影",
			Level:       4,
			Type:        "null",
			Icon:        "229.png",
			Description: "f_s_幻影^ffffff&9@被动&8@游侠 &10@通用&19@10",
		},
		{
			Name:        "灵力进修",
			Level:       4,
			Type:        "null",
			Icon:        "227.png",
			Description: "f_s_灵力进修^ffffff&9@被动&8@游侠 &10@通用&13@90%&4@精力上限 +45",
		},
	}
}

func capturedWoodcutterFastPanel() []RoleFastPanelEntry {
	return []RoleFastPanelEntry{
		{Index: 0, Type: "skill", Name: "普通攻击"},
		{Index: 1, Type: "skill", Name: "强力飞镖"},
		{Index: 2, Type: "skill", Name: "奥义.暗杀者"},
		{Index: 3, Type: "skill", Name: "投毒"},
		{Index: 4, Type: "skill", Name: "疾风刺"},
		{Index: 5, Type: "skill", Name: "解毒术"},
		{Index: 6, Type: "skill", Name: "魔力突刺"},
		{Index: 8, Type: "item", Name: "馒头"},
		{Index: 9, Type: "item", Name: "小瓶甘露"},
	}
}

func capturedWoodcutterBaseSourceQuery() string {
	return "human/human.swf?e=6&sex=1&hr=12&co=5&m=0&n=0&"
}

func capturedWoodcutter40SourceQuery() string {
	return "human/human.swf?=&a=29&b=22&c=26&e=6&sex=1&h=30&hr=12&co=5&m=0&n=0&p=64&se=19&wr=39&w3=49&"
}

func capturedWoodcutter333LatestSourceQuery() string {
	return "human/human.swf?=&a=29&b=22&c=26&e=6&sex=1&h=30&hr=12&co=5&m=0&n=0&p=64&se=19&w1=55&wr=39&"
}

func capturedWoodcutter333RoleSkills() []RoleSkill {
	return []RoleSkill{
		{
			Name:        "普通攻击",
			Level:       1,
			Type:        "oneE",
			Icon:        "7.png",
			Description: "f_s_普通攻击^ffffff&9@单体·攻击&10@通用&22@战斗&5@给予对手普通的物理攻击.",
		},
		{
			Name:        "武器娴熟",
			Level:       5,
			Type:        "null",
			Icon:        "226.png",
			Description: "f_s_武器娴熟^ffffff&9@被动&8@游侠 &10@通用&12@20",
		},
		{
			Name:        "灵力进修",
			Level:       4,
			Type:        "null",
			Icon:        "227.png",
			Description: "f_s_灵力进修^ffffff&9@被动&8@游侠 &10@通用&13@90%&4@精力上限 +45",
		},
		{
			Name:        "精神力",
			Level:       5,
			Type:        "null",
			Icon:        "230.png",
			Description: "f_s_精神力^ffffff&9@被动&8@游侠 &10@通用&18@12",
		},
		{
			Name:        "爆发力",
			Level:       5,
			Type:        "null",
			Icon:        "228.png",
			Description: "f_s_爆发力^ffffff&9@被动&8@游侠 &10@通用&20@190",
		},
		{
			Name:        "幻影",
			Level:       4,
			Type:        "null",
			Icon:        "229.png",
			Description: "f_s_幻影^ffffff&9@被动&8@游侠 &10@通用&19@10",
		},
		{
			Name:        "贯甲连矢",
			Level:       5,
			Type:        "oneE",
			Icon:        "236.png",
			Description: "f_s_贯甲连矢^ffffff&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@28&4@<font color='#00cc00'>特殊发动条件:需要【穿甲箭x1】</font><br>提升25%的物理伤害&0;进攻时增加30%（无视防御）的物理攻击力.",
		},
		{
			Name:        "暗影箭",
			Level:       1,
			Type:        "oneE",
			Icon:        "235.png",
			Description: "f_s_暗影箭^5BC46D&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@20&4@<font color='#00cc00'>特殊发动条件:需要【暗影箭x1】</font><br>造成72%的物理伤害&0;击中敌人时有17%的机率使敌人进入混乱状态2回合",
		},
		{
			Name:        "奥义.轰雷矢",
			Level:       1,
			Type:        "oneE",
			Icon:        "238.png",
			Description: "f_s_奥义.轰雷矢^00ccff&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@26&4@<font color='#00cc00'>特殊发动条件:需要2格魂元</font><br>提升120%的魔法伤害&0;击中敌人时有20%的机率使敌人进入麻痹状态(每回合使其损失气力为魔法攻击的30%)2回合",
		},
		{
			Name:        "毒矢",
			Level:       1,
			Type:        "oneE",
			Icon:        "237.png",
			Description: "f_s_毒矢^5BC46D&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@15&4@<font color='#00cc00'>特殊发动条件:需要【毒箭x1】<br>叠加施放将削弱其造成中毒的功效</font><br>对敌人造成90%的物理伤害&0;击中敌人时有70%的机率使敌人中毒(4回合内降低对方20%的物理防御和魔法防御&0;每回合使敌人损失气力为物理攻击的5%~10%)",
		},
		{
			Name:        "魔力速射",
			Level:       5,
			Type:        "oneE",
			Icon:        "234.png",
			Description: "f_s_魔力速射^5BC46D&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@34&4@<font color='#00cc00'>特殊发动条件:需要【魔箭x1】</font><br>造成50%的物理伤害&0;并追加120%的魔法伤害(进攻时提高25%的魔法攻击力)",
		},
		{
			Name:        "冰箭速射",
			Level:       5,
			Type:        "oneE",
			Icon:        "233.png",
			Description: "f_s_冰箭速射^5BC46D&9@单体·攻击&8@游侠 &10@弓&22@战斗&2@28&4@<font color='#00cc00'>特殊发动条件:需要【冰之箭x1】</font><br><font color='#00cc00'>叠加施放将削弱其造成内伤的功效</font><br>造成70%的物理伤害&0;击中敌人时有90%的机率使敌人进入内伤状态(3回合内削弱敌人30%~35%的物理攻击和魔法攻击)",
		},
	}
}

func capturedWoodcutter333FastPanel() []RoleFastPanelEntry {
	return []RoleFastPanelEntry{
		{Index: 0, Type: "skill", Name: "普通攻击"},
		{Index: 1, Type: "skill", Name: "贯甲连矢"},
		{Index: 2, Type: "skill", Name: "魔力速射"},
		{Index: 3, Type: "skill", Name: "暗影箭"},
		{Index: 4, Type: "skill", Name: "毒矢"},
		{Index: 5, Type: "skill", Name: "奥义.轰雷矢"},
		{Index: 6, Type: "skill", Name: "冰箭速射"},
		{Index: 8, Type: "item", Name: "馒头"},
		{Index: 9, Type: "item", Name: "小瓶甘露"},
	}
}

func capturedWoodcutter333ShortcutSkills() []RoleSkill {
	skills := capturedWoodcutter333RoleSkills()
	skills = upsertCapturedRoleSkill(skills, RoleSkill{
		Name:        "投毒",
		Level:       1,
		Type:        "oneE",
		Icon:        "166.png",
		Description: "f_s_投毒^5BC46D&9@单体·状态&8@游侠 &10@匕首&22@战斗&2@16&4@<font color='#00cc00'>特殊发动条件:需要【毒药x1】<br>叠加施放将削弱其造成中毒的功效</font><br>有80%的机率使敌人中毒，4回合内降低对方15%的物理防御和魔法防御&0;每回合使敌人损失气力为物理攻击的20%~25%",
	})
	skills = upsertCapturedRoleSkill(skills, RoleSkill{
		Name:        "疾风刺",
		Level:       1,
		Type:        "oneE",
		Icon:        "259.png",
		Description: "f_s_疾风刺^5BC46D&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@20&4@对敌人造成40%的物理伤害&0;击中敌人时有92%的机率使对方进入迟钝状态(削减对方50%的命中和回避)3回合<br><font color='#00cc00'>叠加施放将削弱其造成迟钝的功效</font>",
	})
	skills = upsertCapturedRoleSkill(skills, RoleSkill{
		Name:        "解毒术",
		Level:       1,
		Type:        "own",
		Icon:        "260.png",
		Description: "f_s_解毒术^ffffff&9@单体·状态&8@游侠 &10@匕首&22@战斗&2@20&4@解除自身中毒状态",
	})
	skills = upsertCapturedRoleSkill(skills, RoleSkill{
		Name:        "魔力突刺",
		Level:       1,
		Type:        "oneE",
		Icon:        "258.png",
		Description: "f_s_魔力突刺^5BC46D&9@单体·攻击&8@游侠 &10@匕首&22@战斗&2@20&4@造成敌人100%的物理伤害&0;并追加80%的魔法伤害",
	})
	return skills
}

func upsertCapturedRoleSkill(skills []RoleSkill, skill RoleSkill) []RoleSkill {
	skill = normalizeRoleSkill(skill)
	for index := range skills {
		if skills[index].Name != skill.Name {
			continue
		}
		skills[index] = skill
		return skills
	}
	return append(skills, skill)
}

func capturedWoodcutterEquipmentItems() []RoleItem {
	return []RoleItem{
		{
			Type:     "装备",
			Name:     "黄风围巾",
			ItemType: "equip",
			Display:  "548.png",
			Description: `f_i_黄风围巾^00ccff&23@凿孔上限 9 格&24@护具·头部&25@1&21@28&3@19&4@12&5@100&13@7<$jstr>&27@sitem_ezhj&19@精炼潜质:
[精炼+1] 每升一级 物理防御+3
[精炼+1] 每升一级 魔法防御+3
[精炼+1] 每升一级 命中+10
[精炼+6] 每升一级 幸运+3
<font color='#00cc00'>特殊效果:
麻痹抗性:+5%
眩晕抗性:+5%
冰冻抗性:+8%</font>
[精炼+1] 每升一级 麻痹抗性+1%
[精炼+1] 每升一级 眩晕抗性+1%
[精炼+1] 每升一级 冰冻抗性+1%&103@0&104@0&105@桥头的樵夫&107@&108@160`,
			Count:     1,
			Index:     0,
			Owner:     "桥头的樵夫",
			ItemLevel: 3,
		},
		{
			Type:     "装备",
			Name:     "蚩颅王护肩",
			ItemType: "equip",
			Display:  "484.png",
			Description: `f_i_蚩颅王护肩^00ccff&23@凿孔上限 9 格&24@护具·肩部&25@1&21@31&3@18(+6)&4@10(+6)&14@5&15@4&17@5&27@sitem_jhj&19@精炼潜质:
[精炼+1] 每升一级 物理防御+3
[精炼+1] 每升一级 魔法防御+3
[精炼+6] 每升一级 气力上限+100
<font color='#00cc00'>特殊效果:
遭受爆击时&0;有7%的机率使敌人进入眩晕状态1回合</font>
[精炼+1] 每升一级 眩晕反射机率+1%&103@2&104@0&105@桥头的樵夫&107@&108@430`,
			Count:     1,
			Index:     1,
			Level:     2,
			Owner:     "桥头的樵夫",
			ItemLevel: 3,
		},
		{
			Type:        "装备",
			Name:        "黄风护腕",
			ItemType:    "equip",
			Display:     "549.png",
			Description: "f_i_黄风护腕^5BC46D&23@凿孔上限 9 格&24@护具·护腕&25@1&21@25&3@15&11@16&13@2&27@sitem_jhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2\n[精炼+1] 每升一级 命中+5&103@0&104@0&105@&107@&108@160",
			Count:       1,
			Index:       2,
			ItemLevel:   2,
		},
		{
			Type:        "装备",
			Name:        "绯雨匕首",
			ItemType:    "equip",
			Display:     "51.png",
			Description: "f_i_绯雨匕首&23@凿孔上限 9 格&24@武器·匕首系&25@1&21@30&22@游侠&1@70&11@30&27@sitem_jwep&19@精炼潜质:\n[精炼+1] 每升一级 物理攻击+2\n<font color='#00cc00'>特殊效果:\n击中敌人时候有1%机率使敌人进入内伤状态3回合(降低敌人10%~15%的物理攻击和魔法攻击)</font>&103@0&104@0&105@&107@&108@500",
			Count:       1,
			Index:       3,
			ItemLevel:   1,
		},
		{
			Type:        "装备",
			Name:        "神风护甲",
			ItemType:    "equip",
			Display:     "366.png",
			Description: "f_i_神风护甲^5BC46D&23@凿孔上限 9 格&24@护具·躯干&25@1&21@30&22@游侠&3@33&4@5&10@8&27@sitem_jhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2\n[精炼+1] 每升一级 魔法防御+2\n[精炼+6] 每升一级 爆击+12&103@0&104@0&105@&107@&108@120",
			Count:       1,
			Index:       4,
			ItemLevel:   2,
		},
		{
			Type:        "装备",
			Name:        "神风护腿",
			ItemType:    "equip",
			Display:     "368.png",
			Description: "f_i_神风护腿^5BC46D&23@凿孔上限 9 格&24@护具·腿&25@1&21@30&22@游侠&3@24&4@2&13@2&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2\n[精炼+1] 每升一级 魔法防御+2\n[精炼+6] 每升一级 回避+10&103@0&104@0&105@&107@&108@120",
			Count:       1,
			Index:       5,
			ItemLevel:   2,
		},
		{
			Type:        "装备",
			Name:        "炎火兽",
			ItemType:    "equip",
			Display:     "324.png",
			Description: "f_i_炎火兽^5BC46D&24@宠物&25@1&19@<font color='#00ff00'>可以提高主人的物理攻击力</font>&20@体内流淌着高温的酸性体液&0;喷射出来遇到火花就会烧起熊熊大火.生性好斗勇猛.&27@sitem_pet&103@0&104@0&105@&107@&108@0&19@喜好食物:\n宠物用营养水\n宠物成长药剂\n奇效宠物药剂\n<font color='#66ccff'>宠物等级:5\n物理攻击+14\n</font><font color='#00cc00'>成长属性:\n[等级1] 物理攻击+10 每升一级 +1</font>",
			Count:       1,
			Index:       9,
			ItemLevel:   2,
		},
		{
			Type:        "装备",
			Name:        "神风护腰",
			ItemType:    "equip",
			Display:     "369.png",
			Description: "f_i_神风护腰^5BC46D&23@凿孔上限 9 格&24@护具·腰部&25@1&21@30&22@游侠&3@15&10@3&17@5&27@sitem_jhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2\n[精炼+6] 每升一级 耐力+1&103@0&104@0&105@&107@&108@120",
			Count:       1,
			Index:       10,
			ItemLevel:   2,
		},
		{
			Type:        "装备",
			Name:        "神风战靴",
			ItemType:    "equip",
			Display:     "370.png",
			Description: "f_i_神风战靴^5BC46D&23@凿孔上限 9 格&24@护具·足部&25@1&21@30&22@游侠&3@17&12@5&15@2&27@sitem_jhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2\n[精炼+6] 每升一级 幸运+2&103@0&104@0&105@&107@&108@120",
			Count:       1,
			Index:       12,
			ItemLevel:   2,
		},
		{
			Type:        "装备",
			Name:        "L千年人参果",
			ItemType:    "equip",
			Display:     "921.png",
			Description: "f_i_L千年人参果^C156C7&23@限制装备至【宝1、宝2、宝3、宝4】格。<br/>剩余精力【8440】点&24@法宝&25@1&19@战斗结束后可自动将精力恢复至满，内置精力值用尽后消失。&20@<font color='#00ccff'>注:【内置50000精力 双击装备后生效】</font>&27@sitem_water&103@0&104@0&105@桥头的樵夫&107@&108@0",
			Count:       1,
			Index:       15,
			Owner:       "桥头的樵夫",
			ItemLevel:   5,
		},
	}
}

func syncCapturedWoodcutterEquipmentItems(items []RoleItem) []RoleItem {
	normalized := normalizeRoleItems(items)
	result := make([]RoleItem, 0, len(normalized)+len(capturedWoodcutterEquipmentItems()))
	petStates := make(map[int]RolePetItemState)
	for _, item := range normalized {
		if item.Type == "装备" && item.PetState != nil {
			petStates[item.Index] = *item.PetState
		}
		if item.Type == "装备" {
			continue
		}
		if item.Type == "背包" && item.Name == "铁斧" && item.Display == "29.png" && item.Index == 19 {
			continue
		}
		result = append(result, item)
	}
	for _, item := range capturedWoodcutterEquipmentItems() {
		if state, ok := petStates[item.Index]; ok && item.Index == 9 {
			item.PetState = &state
		}
		result = append(result, item)
	}
	return normalizeRoleItems(result)
}

func isCapturedWoodcutterLocalRole(role RoleSummary) bool {
	return strings.TrimSpace(role.DisplayName) == "222" || strings.Contains(role.RoleID, "-role-222")
}

func isCapturedWoodcutter333LocalRole(role RoleSummary) bool {
	roleID := strings.TrimSpace(role.RoleID)
	return strings.HasPrefix(roleID, "acct-333-role-") || strings.HasPrefix(roleID, "acct-33333333-role-")
}

func withCapturedWoodcutter333RuntimeDefaults(role RoleSummary) RoleSummary {
	role.Voc = "游侠"
	role.DisplayName = "333"
	if role.Level < 44 || role.Exp <= 0 {
		role.Level = 44
		role.Exp = 5768622
	}
	if role.AGI == 0 && role.STR == 0 && role.INT == 0 && role.CON == 0 && role.LCK == 0 {
		role.AGI = 175
		role.STR = 73
		role.INT = 19
		role.CON = 0
		role.LCK = 19
	}
	if role.MapID <= 1 {
		role.MapID = 15
	}
	if role.VisualRoleID <= 0 {
		role.VisualRoleID = 1
	}
	if role.RoleState == nil || role.RoleState.Lv < 44 || role.RoleState.Exp <= 0 {
		roleState := capturedWoodcutter40RoleState(role.RoleID)
		role.RoleState = &roleState
	}
	if role.RolePhysique == nil || role.RolePhysique.MaxHP <= 0 || role.RolePhysique.MaxMP <= 0 ||
		role.RolePhysique.MgcAtk != role.RolePhysique.PhyAtk {
		rolePhysique := capturedWoodcutter40RolePhysique(role.RoleID)
		role.RolePhysique = &rolePhysique
	}
	return role
}

func capturedWoodcutter40RoleState(roleID string) RoleState {
	return RoleState{
		Handle: roleID,
		HP:     1337,
		MP:     557,
		Exp:    5768622,
		Lv:     44,
		Speed:  140,
		OutG:   0,
		InG:    0,
	}
}

func capturedWoodcutter40RolePhysique(roleID string) RolePhysique {
	return RolePhysique{
		Handle:    roleID,
		ResPros:   []string{"冰冻|8", "眩晕|15", "封印|10", "混乱|10", "麻痹|5"},
		AGI:       175,
		STR:       73,
		INT:       19,
		CON:       0,
		LCK:       19,
		MaxHP:     1365,
		MaxMP:     709,
		PhyAtk:    297,
		MgcAtk:    297,
		PhyDef:    267,
		MgcDef:    87,
		Hit:       517,
		Dog:       268,
		Fat:       577,
		LastPoint: 0,
	}
}

func isCapturedWarrior444LocalRole(role RoleSummary) bool {
	roleID := strings.TrimSpace(role.RoleID)
	return strings.HasPrefix(roleID, "acct-444-role-") || strings.HasPrefix(roleID, "acct-44444444-role-")
}

func capturedWarrior44SourceQuery() string {
	return "human/human.swf?=&a=32&b=21&c=39&e=6&sex=1&h=30&hr=12&co=5&m=0&n=0&p=22&se=29&w11=53&wr=19&"
}

func capturedWoodcutter40BodySourceQuery() string {
	return clearRoleEquipmentAppearanceSourceQuery(capturedWoodcutter333LatestSourceQuery())
}

func capturedWarrior44BodySourceQuery() string {
	return clearRoleEquipmentAppearanceSourceQuery(capturedWarrior44SourceQuery())
}

type capturedRoleEquipmentSpec struct {
	name  string
	index int
}

func capturedWoodcutter333EquipmentItems() []RoleItem {
	return capturedRoleEquipmentItems([]capturedRoleEquipmentSpec{
		{name: "黄风围巾", index: 0},
		{name: "蚩颅王护肩", index: 1},
		{name: "机木护腕", index: 2},
		{name: "万相", index: 3},
		{name: "寒影锁甲", index: 4},
		{name: "机木护腿", index: 5},
		{name: "骷髅戒指", index: 6},
		{name: "银耳坠", index: 7},
		{name: "翡翠项链", index: 8},
		{name: "炎火兽", index: 9},
		{name: "寒影护腰", index: 10},
		{name: "寒影靴", index: 12},
	})
}

func syncCapturedWoodcutter333EquipmentItems(items []RoleItem) []RoleItem {
	normalized := normalizeRoleItems(items)
	if !shouldSyncCapturedWoodcutter333EquipmentItems(normalized) {
		return normalized
	}
	capturedEquipment := capturedWoodcutter333EquipmentItems()
	for _, captured := range capturedEquipment {
		replaced := false
		for index := range normalized {
			if normalized[index].Type == "装备" && normalized[index].Index == captured.Index {
				normalized[index] = captured
				replaced = true
				break
			}
		}
		if !replaced {
			normalized = append(normalized, captured)
		}
	}
	return normalizeRoleItems(normalized)
}

func shouldSyncCapturedWoodcutter333EquipmentItems(items []RoleItem) bool {
	hasWanXiang := false
	hasRing := false
	hasEarring := false
	hasNecklace := false
	for _, item := range items {
		if item.Name == "万相" {
			hasWanXiang = true
		}
		if item.Type != "装备" {
			continue
		}
		switch {
		case item.Name == "骷髅戒指" && item.Index == 6:
			hasRing = true
		case item.Name == "银耳坠" && item.Index == 7:
			hasEarring = true
		case item.Name == "翡翠项链" && item.Index == 8:
			hasNecklace = true
		}
	}
	return !(hasWanXiang && hasRing && hasEarring && hasNecklace)
}

func syncCapturedWoodcutter333BattleConsumables(items []RoleItem) []RoleItem {
	normalized := normalizeRoleItems(items)
	normalized = syncCapturedWoodcutter333BattleConsumable(normalized, "穿甲箭", 1319)
	normalized = syncCapturedWoodcutter333BattleConsumable(normalized, "暗之箭", 39)
	normalized = syncCapturedWoodcutter333BattleConsumable(normalized, "毒箭", 9)
	normalized = syncCapturedWoodcutter333BattleConsumable(normalized, "火之箭", 50)
	normalized = syncCapturedWoodcutter333BattleConsumable(normalized, "冰之箭", 50)
	normalized = syncCapturedWoodcutter333BattleConsumable(normalized, "魔箭", 50)
	return normalizeRoleItems(normalized)
}

func syncCapturedWoodcutter333BattleConsumable(items []RoleItem, name string, count int) []RoleItem {
	if totalRoleItemCountByName(items, "背包", name) > 0 {
		return items
	}
	item, ok := CapturedRoleItemTemplate(name)
	if !ok {
		return items
	}
	item.Type = "背包"
	item.Count = count
	item.Index = -1
	capacity := maxInt(effectiveRoleContainerCapacity(items, "背包", defaultBagCap), 42)
	updated, _, granted := grantRoleItemToItems(items, capacity, item)
	if !granted {
		return items
	}
	return updated
}

func capturedWarrior444EquipmentItems() []RoleItem {
	return capturedRoleEquipmentItems([]capturedRoleEquipmentSpec{
		{name: "黄风围巾", index: 0},
		{name: "狼人护肩", index: 1},
		{name: "龙颜护腕", index: 2},
		{name: "伏魔棍", index: 3},
		{name: "寨夫人上衣", index: 4},
		{name: "龙颜护腿", index: 5},
		{name: "骷髅戒指", index: 6},
		{name: "银耳坠", index: 7},
		{name: "翡翠项链", index: 8},
		{name: "怪木机", index: 9},
		{name: "龙颜护腰", index: 10},
		{name: "蛤蟆精战靴", index: 12},
		{name: "泥戒指", index: 13},
	})
}

func syncCapturedWarrior444EquipmentItems(items []RoleItem) []RoleItem {
	normalized := normalizeRoleItems(items)
	if !shouldSyncCapturedWarrior444EquipmentItems(normalized) {
		return normalized
	}
	capturedEquipment := capturedWarrior444EquipmentItems()
	for _, captured := range capturedEquipment {
		replaced := false
		for index := range normalized {
			if normalized[index].Type == "装备" && normalized[index].Index == captured.Index {
				normalized[index] = captured
				replaced = true
				break
			}
		}
		if !replaced {
			normalized = append(normalized, captured)
		}
	}
	return normalizeRoleItems(normalized)
}

func shouldSyncCapturedWarrior444EquipmentItems(items []RoleItem) bool {
	hasWolfShoulder := false
	hasRing := false
	hasEarring := false
	hasNecklace := false
	hasMudRing := false
	hasOldShoulder := false
	for _, item := range items {
		if item.Type != "装备" {
			continue
		}
		switch {
		case item.Name == "狼人护肩" && item.Index == 1:
			hasWolfShoulder = true
		case item.Name == "龙颜单肩" && item.Index == 1:
			hasOldShoulder = true
		case item.Name == "骷髅戒指" && item.Index == 6:
			hasRing = true
		case item.Name == "银耳坠" && item.Index == 7:
			hasEarring = true
		case item.Name == "翡翠项链" && item.Index == 8:
			hasNecklace = true
		case item.Name == "泥戒指" && item.Index == 13:
			hasMudRing = true
		}
	}
	return hasOldShoulder || !(hasWolfShoulder && hasRing && hasEarring && hasNecklace && hasMudRing)
}

func capturedRoleEquipmentItems(specs []capturedRoleEquipmentSpec) []RoleItem {
	items := make([]RoleItem, 0, len(specs))
	for _, spec := range specs {
		item, ok := CapturedRoleItemTemplate(spec.name)
		if !ok {
			item = RoleItem{
				Name:        spec.name,
				ItemType:    "equip",
				Description: fmt.Sprintf("f_i_%s&24@装备", spec.name),
			}
		}
		item.Type = "装备"
		item.ItemType = "equip"
		item.Count = 1
		item.Index = spec.index
		items = append(items, normalizeRoleItem(item))
	}
	return normalizeRoleItems(items)
}

func capturedWarrior444RoleSkills() []RoleSkill {
	return []RoleSkill{
		{
			Name:        "强体质",
			Level:       5,
			Type:        "null",
			Icon:        "167.png",
			Description: "f_s_强体质^ffffff&9@被动&8@战士 &10@通用&4@气力上限 +140",
		},
		{
			Name:        "奥义.雷魂斩",
			Level:       1,
			Type:        "oneE",
			Icon:        "183.png",
			Description: "f_s_奥义.雷魂斩^00ccff&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@24&4@<font color='#00cc00'>特殊发动条件:需要3格魂元</font><br>提升240%的物理伤害",
		},
		{
			Name:        "盘龙棍法",
			Level:       2,
			Type:        "all",
			Icon:        "187.png",
			Description: "f_s_盘龙棍法^ffffff&9@群体·攻击&8@战士 &10@棍&22@战斗&2@18&4@对所有敌人造成84%的物理伤害",
		},
		{
			Name:        "红月斩",
			Level:       2,
			Type:        "all",
			Icon:        "181.png",
			Description: "f_s_红月斩^ffffff&9@群体·攻击&8@战士 &10@单刀&22@战斗&2@45&4@对所有敌人造成74%的物理伤害",
		},
		{
			Name:        "力释棍术",
			Level:       1,
			Type:        "own",
			Icon:        "186.png",
			Description: "f_s_力释棍术^5BC46D&9@单体·状态&8@战士 &10@棍&22@战斗&2@10&4@5回合内提升物理攻击15%",
		},
		{
			Name:        "武器专精",
			Level:       5,
			Type:        "null",
			Icon:        "165.png",
			Description: "f_s_武器专精^ffffff&9@被动&8@战士 &10@通用&12@19",
		},
		{
			Name:        "抗击打",
			Level:       5,
			Type:        "null",
			Icon:        "168.png",
			Description: "f_s_抗击打^ffffff&9@被动&8@战士 &10@通用&14@16",
		},
		{
			Name:        "普通攻击",
			Level:       1,
			Type:        "oneE",
			Icon:        "7.png",
			Description: "f_s_普通攻击^ffffff&9@单体·攻击&10@通用&22@战斗&5@给予对手普通的物理攻击.",
		},
		{
			Name:        "嗜血斩",
			Level:       5,
			Type:        "oneE",
			Icon:        "179.png",
			Description: "f_s_嗜血斩^5BC46D&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@40&4@对敌人造成100%的物理伤害&0;并有100%机率将对敌人造成伤害的70%转换为气力</font>",
		},
		{
			Name:        "奥义.六合棍法",
			Level:       1,
			Type:        "oneE",
			Icon:        "190.png",
			Description: "f_s_奥义.六合棍法^00ccff&9@单体·攻击&8@战士 &10@棍&22@战斗&2@24&4@<font color='#00cc00'>特殊发动条件:需要3格魂元</font><br>提升210%的物理伤害&0;进攻时候增加300%的命中",
		},
		{
			Name:        "夜叉棍法",
			Level:       1,
			Type:        "oneE",
			Icon:        "188.png",
			Description: "f_s_夜叉棍法^5BC46D&9@单体·攻击&8@战士 &10@棍&22@战斗&2@15&4@提升12%的物理伤害&0;击中敌人时有90%的机率对敌人造成内伤(削减敌人32%的物理攻击和魔法攻击)3回合<br><font color='#00cc00'>叠加施放将削弱其造成内伤的功效</font>",
		},
		{
			Name:        "劈山棍法",
			Level:       5,
			Type:        "oneE",
			Icon:        "185.png",
			Description: "f_s_劈山棍法^ffffff&9@单体·攻击&8@战士 &10@棍&22@战斗&2@16&4@提升75%的物理伤害",
		},
	}
}

func capturedWarrior444FastPanel() []RoleFastPanelEntry {
	return []RoleFastPanelEntry{
		{Index: 0, Type: "skill", Name: "普通攻击"},
		{Index: 1, Type: "skill", Name: "劈山棍法"},
		{Index: 2, Type: "skill", Name: "夜叉棍法"},
		{Index: 3, Type: "skill", Name: "盘龙棍法"},
		{Index: 4, Type: "skill", Name: "力释棍术"},
		{Index: 5, Type: "skill", Name: "奥义.六合棍法"},
		{Index: 8, Type: "item", Name: "馒头"},
		{Index: 9, Type: "item", Name: "小瓶甘露"},
	}
}

func withCapturedWarrior444RuntimeDefaults(role RoleSummary) RoleSummary {
	role.Voc = "战士"
	role.DisplayName = "444"
	if role.Level < 44 || role.Exp < 5793804 {
		role.Level = 44
		role.Exp = 5793804
	}
	if (role.AGI == 0 && role.STR == 0 && role.INT == 0 && role.CON == 0 && role.LCK == 0) ||
		role.AGI < 76 || role.STR < 168 {
		role.AGI = 76
		role.STR = 168
		role.INT = 5
		role.CON = 0
		role.LCK = 0
	}
	if role.MapID <= 1 || role.MapID == 15 {
		role.MapID = 1
	}
	if role.VisualRoleID <= 0 {
		role.VisualRoleID = 1
	}
	if len(role.Currencies) == 0 || (role.Currencies["银元宝"] == 32 && role.Currencies["铜钱"] == 842) {
		role.Currencies = RoleCurrencies{"银元宝": 153, "铜钱": 806}
	}
	if role.RoleState == nil || role.RoleState.Lv < 44 || role.RoleState.Exp < 5793804 {
		roleState := capturedWarrior44RoleState(role.RoleID)
		role.RoleState = &roleState
	}
	if role.RolePhysique == nil || role.RolePhysique.MaxHP < 1775 || role.RolePhysique.MaxMP < 524 ||
		role.RolePhysique.PhyAtk < 341 {
		rolePhysique := capturedWarrior44RolePhysique(role.RoleID)
		role.RolePhysique = &rolePhysique
	}
	return role
}

func capturedWarrior44RoleState(roleID string) RoleState {
	return RoleState{
		Handle: roleID,
		HP:     1344,
		MP:     390,
		Exp:    5793804,
		Lv:     44,
		Speed:  147,
		OutG:   0,
		InG:    0,
	}
}

func capturedWarrior44RolePhysique(roleID string) RolePhysique {
	return RolePhysique{
		Handle:    roleID,
		ResPros:   []string{"冰冻|8", "眩晕|15", "封印|10", "混乱|40", "麻痹|15"},
		AGI:       76,
		STR:       168,
		INT:       5,
		CON:       0,
		LCK:       0,
		MaxHP:     1775,
		MaxMP:     524,
		PhyAtk:    341,
		MgcAtk:    5,
		PhyDef:    223,
		MgcDef:    73,
		Hit:       350,
		Dog:       156,
		Fat:       289,
		LastPoint: 0,
	}
}

func ensureDefaultRoleSkills(skills []RoleSkill) []RoleSkill {
	normalized := cloneRoleSkills(skills)
	seen := make(map[string]struct{}, len(normalized))
	for _, skill := range normalized {
		if skill.Name != "" {
			seen[skill.Name] = struct{}{}
		}
	}
	for _, skill := range defaultRoleSkills() {
		if _, ok := seen[skill.Name]; ok {
			continue
		}
		normalized = append(normalized, skill)
	}
	return normalized
}

func parseRoleSeq(playerID string, roleID string) (int, bool) {
	prefix := playerID + "-role-"
	if !strings.HasPrefix(roleID, prefix) {
		return 0, false
	}

	roleSeq, err := strconv.Atoi(roleID[len(prefix):])
	if err != nil || roleSeq <= 0 {
		return 0, false
	}

	return roleSeq, true
}

func formatRoleSummaries(roles []RoleSummary) string {
	if len(roles) == 0 {
		return "[]"
	}

	parts := make([]string, 0, len(roles))
	for _, role := range roles {
		parts = append(parts, fmt.Sprintf("{id:%s,name:%s,level:%d}", role.RoleID, role.DisplayName, role.Level))
	}

	return "[" + strings.Join(parts, ", ") + "]"
}

func emptyRoleSummary() RoleSummary {
	return RoleSummary{
		RoleID:       "",
		DisplayName:  "",
		Level:        0,
		MapID:        0,
		VisualRoleID: 0,
	}
}

func emptyPlayerBaseData(playerID string) PlayerBaseData {
	return PlayerBaseData{
		PlayerID:     playerID,
		RoleID:       "",
		DisplayName:  "",
		Level:        0,
		Exp:          0,
		MapID:        0,
		VisualRoleID: 0,
	}
}

func playerBaseDataFromRole(playerID string, role RoleSummary) PlayerBaseData {
	role = withRoleRuntimeDefaults(role)
	role.Level = ClassicRoleLevelForExp(role.Exp, role.Level)
	roleState := defaultRoleState(role.RoleID, role.Level, role.Exp)
	rolePhysique := defaultRolePhysique(role)
	if role.RoleState != nil {
		roleState = *role.RoleState
		if roleState.Handle == "" {
			roleState.Handle = role.RoleID
		}
	}
	if role.RolePhysique != nil {
		rolePhysique = *role.RolePhysique
		if rolePhysique.Handle == "" {
			rolePhysique.Handle = role.RoleID
		}
	}
	return PlayerBaseData{
		PlayerID:          playerID,
		RoleID:            role.RoleID,
		DisplayName:       role.DisplayName,
		Level:             role.Level,
		Exp:               role.Exp,
		Voc:               role.Voc,
		HP:                roleState.HP,
		MP:                roleState.MP,
		MaxHP:             rolePhysique.MaxHP,
		MaxMP:             rolePhysique.MaxMP,
		MapID:             role.MapID,
		VisualRoleID:      role.VisualRoleID,
		PresetID:          role.PresetID,
		SourceQuery:       role.SourceQuery,
		BattleSourceQuery: role.BattleSourceQuery,
		Appearance:        role.Appearance,
		Currencies:        cloneRoleCurrencies(role.Currencies),
		RoleState:         &roleState,
		RolePhysique:      &rolePhysique,
	}
}

func defaultRoleState(roleID string, level int, exp ...int) RoleState {
	roleExp := 0
	if len(exp) > 0 && exp[0] > 0 {
		roleExp = exp[0]
	}
	level = normalizeRoleLevel(level)
	maxHP := ClassicRoleMaxHP(level)
	maxMP := ClassicRoleMaxMP(level)
	return RoleState{
		Handle: roleID,
		HP:     maxHP,
		MP:     maxMP,
		Exp:    roleExp,
		Lv:     level,
		Speed:  ClassicRoleSpeed(level),
		OutG:   0,
		InG:    0,
	}
}

func defaultRolePhysique(role RoleSummary) RolePhysique {
	level := normalizeRoleLevel(role.Level)
	agi := maxInt(0, role.AGI)
	str := maxInt(0, role.STR)
	intelligence := maxInt(0, role.INT)
	con := maxInt(0, role.CON)
	lck := maxInt(0, role.LCK)
	equipment := roleEquipmentStats(role.Items)
	maxHP := ClassicRoleMaxHP(level) + con*10 + equipment.maxHP
	maxMP := ClassicRoleMaxMP(level) + intelligence*10 + equipment.maxMP
	allocated := agi + str + intelligence + con + lck
	return RolePhysique{
		Handle:    role.RoleID,
		ResPros:   []string{},
		AGI:       agi,
		STR:       str,
		INT:       intelligence,
		CON:       con,
		LCK:       lck,
		MaxHP:     maxHP,
		MaxMP:     maxMP,
		PhyAtk:    10 + str + equipment.phyAtk,
		MgcAtk:    equipment.mgcAtk,
		PhyDef:    10 + (agi+1)/2 + equipment.phyDef,
		MgcDef:    10 + equipment.mgcDef,
		Hit:       100 + agi + equipment.hit,
		Dog:       50 + (agi+1)/2 + equipment.dog,
		Fat:       5 + str/3 + (agi*3+1)/2 + equipment.fat,
		LastPoint: maxInt(0, level*5-allocated),
	}
}

func syncRoleProgressionRuntimeData(role RoleSummary) RoleSummary {
	role = withRoleRuntimeDefaults(role)
	role.Level = ClassicRoleLevelForExp(role.Exp, role.Level)
	roleState := defaultRoleState(role.RoleID, role.Level, role.Exp)
	if role.RoleState != nil {
		roleState = *role.RoleState
		if roleState.Handle == "" {
			roleState.Handle = role.RoleID
		}
		roleState.Exp = role.Exp
		roleState.Lv = role.Level
		roleState.Speed = ClassicRoleSpeed(role.Level)
	}
	rolePhysique := defaultRolePhysique(role)
	role.RoleState = &roleState
	role.RolePhysique = &rolePhysique
	return role
}

func normalizeRoleLevel(level int) int {
	if level <= 0 {
		return 1
	}
	return level
}

func ClassicRoleLevelToExp(level int) int {
	if level < 0 {
		return 0
	}
	if level >= len(classicRoleLevelToExp) {
		return classicRoleLevelToExp[len(classicRoleLevelToExp)-1]
	}
	return classicRoleLevelToExp[level]
}

func ClassicRoleMaxLevel() int {
	return len(classicRoleLevelToExp) - 1
}

func ClassicRoleLevelForExp(exp int, fallbackLevel int) int {
	if exp < 0 {
		exp = 0
	}
	level := normalizeRoleLevel(fallbackLevel)
	for index := 1; index < len(classicRoleLevelToExp); index++ {
		if exp < classicRoleLevelToExp[index] {
			if index > level {
				return index
			}
			return level
		}
	}
	return maxInt(level, len(classicRoleLevelToExp)-1)
}

func ClassicRoleMaxHP(level int) int {
	return 165 + normalizeRoleLevel(level)*20
}

func ClassicRoleMaxMP(level int) int {
	return 34 + normalizeRoleLevel(level)*10
}

func ClassicRoleSpeed(level int) int {
	if normalizeRoleLevel(level) >= 7 {
		return 140
	}
	return 130
}

type roleEquipmentStatBonus struct {
	phyAtk int
	mgcAtk int
	phyDef int
	mgcDef int
	maxHP  int
	maxMP  int
	hit    int
	dog    int
	fat    int
}

func roleEquipmentStats(items []RoleItem) roleEquipmentStatBonus {
	bonus := roleEquipmentStatBonus{}
	for _, item := range items {
		if item.Type != "装备" {
			continue
		}
		bonus.phyAtk += parseClassicDescriptionSignedInt(item.Description, "1")
		bonus.mgcAtk += parseClassicDescriptionSignedInt(item.Description, "2")
		bonus.phyDef += parseClassicDescriptionSignedInt(item.Description, "3")
		bonus.mgcDef += parseClassicDescriptionSignedInt(item.Description, "4")
		bonus.maxHP += parseClassicDescriptionSignedInt(item.Description, "5")
		bonus.maxMP += parseClassicDescriptionSignedInt(item.Description, "6")
		bonus.hit += parseClassicDescriptionSignedInt(item.Description, "9")
		bonus.dog += parseClassicDescriptionSignedInt(item.Description, "10")
		bonus.fat += parseClassicDescriptionSignedInt(item.Description, "11")
	}
	return bonus
}

func parseClassicDescriptionSignedInt(description string, key string) int {
	marker := "&" + strings.TrimSpace(key) + "@"
	value := 0
	for start := strings.Index(description, marker); start >= 0; {
		start += len(marker)
		end := start
		if end < len(description) && (description[end] == '-' || description[end] == '+') {
			end += 1
		}
		for end < len(description) {
			ch := description[end]
			if ch < '0' || ch > '9' {
				break
			}
			end += 1
		}
		if end > start && !(end == start+1 && (description[start] == '-' || description[start] == '+')) {
			if parsed, err := strconv.Atoi(description[start:end]); err == nil {
				value = parsed
			}
		}
		nextStart := strings.Index(description[end:], marker)
		if nextStart < 0 {
			break
		}
		start = end + nextStart
	}
	return value
}

func classicItemStackLimit(item RoleItem) int {
	limit := parseClassicDescriptionSignedInt(item.Description, "25")
	if limit <= 0 {
		return 1
	}
	return limit
}

func canRoleItemsStack(existing RoleItem, incoming RoleItem) bool {
	existing = normalizeRoleItem(existing)
	incoming = normalizeRoleItem(incoming)
	if classicItemStackLimit(existing) <= 1 || classicItemStackLimit(incoming) <= 1 {
		return false
	}
	return existing.Type == incoming.Type &&
		existing.Name == incoming.Name &&
		existing.ItemType == incoming.ItemType &&
		existing.Display == incoming.Display &&
		classicStackDescription(existing.Description) == classicStackDescription(incoming.Description) &&
		existing.Level == incoming.Level &&
		existing.EndTime == incoming.EndTime &&
		existing.Owner == incoming.Owner &&
		existing.ItemLevel == incoming.ItemLevel
}

func classicStackDescription(description string) string {
	description = strings.TrimSpace(description)
	for {
		start := strings.Index(description, "&101@")
		if start < 0 {
			return description
		}
		rest := description[start+len("&101@"):]
		next := strings.Index(rest, "&")
		if next < 0 {
			return strings.TrimSpace(description[:start])
		}
		description = description[:start] + rest[next:]
	}
}

func remainingRolePoint(role RoleSummary) int {
	level := ClassicRoleLevelForExp(role.Exp, role.Level)
	allocated := maxInt(0, role.AGI) + maxInt(0, role.STR) + maxInt(0, role.INT) + maxInt(0, role.CON) + maxInt(0, role.LCK)
	return maxInt(0, normalizeRoleLevel(level)*5-allocated)
}

func isClassicRolePointStat(statName string) bool {
	switch strings.ToUpper(strings.TrimSpace(statName)) {
	case "AGI", "STR", "INT", "CON", "LCK":
		return true
	default:
		return false
	}
}

func addRolePointToStat(role *RoleSummary, statName string) {
	if role == nil {
		return
	}
	switch strings.ToUpper(strings.TrimSpace(statName)) {
	case "AGI":
		role.AGI += 1
	case "STR":
		role.STR += 1
	case "INT":
		role.INT += 1
	case "CON":
		role.CON += 1
	case "LCK":
		role.LCK += 1
	}
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func withRoleRuntimeDefaults(role RoleSummary) RoleSummary {
	if vocation, ok := normalizeRoleVocation(role.Voc); ok {
		role.Voc = vocation
	} else {
		role.Voc = defaultRoleVoc
	}
	isWoodcutter222 := isCapturedWoodcutterLocalRole(role)
	isWoodcutter333 := isCapturedWoodcutter333LocalRole(role)
	isWarrior444 := isCapturedWarrior444LocalRole(role)
	if isWoodcutter333 {
		role = withCapturedWoodcutter333RuntimeDefaults(role)
	}
	if isWarrior444 {
		role = withCapturedWarrior444RuntimeDefaults(role)
	}
	if isWarrior444 {
		role.Skills = capturedWarrior444RoleSkills()
	} else if isWoodcutter333 {
		role.Skills = capturedWoodcutter333RoleSkills()
	} else if isWoodcutter222 {
		role.Voc = "游侠"
		role.Skills = capturedWoodcutterRoleSkills()
	} else if len(role.Skills) == 0 {
		role.Skills = defaultRoleSkills()
	} else {
		role.Skills = cloneRoleSkills(role.Skills)
	}
	if isWarrior444 {
		role.FastPanel = capturedWarrior444FastPanel()
	} else if isWoodcutter333 {
		role.FastPanel = capturedWoodcutter333FastPanel()
	} else if isWoodcutter222 {
		role.FastPanel = capturedWoodcutterFastPanel()
	} else if len(role.FastPanel) == 0 {
		role.FastPanel = defaultRoleFastPanel()
	} else {
		role.FastPanel = normalizeRoleFastPanel(role.FastPanel)
	}
	role.TownBuffs = normalizeRoleTownBuffs(role.TownBuffs)
	if len(role.Currencies) == 0 {
		role.Currencies = defaultRoleCurrencies()
	} else {
		role.Currencies = normalizeRoleCurrencies(role.Currencies)
	}
	if isWoodcutter222 {
		role.Items = syncCapturedWoodcutterEquipmentItems(role.Items)
	} else if len(role.Items) == 0 {
		if isWoodcutter333 {
			role.Items = syncCapturedWoodcutter333BattleConsumables(capturedWoodcutter333EquipmentItems())
		} else if isWarrior444 {
			role.Items = capturedWarrior444EquipmentItems()
		} else {
			role.Items = defaultRoleItems()
		}
	} else {
		role.Items = ensureStarterAxeItem(removeCapturedDefaultBagSeeds(normalizeRoleItems(role.Items)))
	}
	if isWoodcutter333 {
		role.Items = syncCapturedWoodcutter333EquipmentItems(role.Items)
		role.Items = syncCapturedWoodcutter333BattleConsumables(role.Items)
	}
	if isWarrior444 {
		role.Items = syncCapturedWarrior444EquipmentItems(role.Items)
	}
	role.DungeonInstances = cloneDungeonInstances(role.DungeonInstances)
	if isWoodcutter222 {
		role.SourceQuery = capturedWoodcutterBaseSourceQuery()
	} else if isWoodcutter333 {
		role.SourceQuery = capturedWoodcutter40BodySourceQuery()
	} else if isWarrior444 {
		role.SourceQuery = capturedWarrior44BodySourceQuery()
	}
	role.SourceQuery = applyRoleBodyAppearanceToSourceQuery(role.SourceQuery, role.Appearance)
	role.SourceQuery = rebuildRoleEquipmentAppearanceSourceQuery(role.SourceQuery, role.Items)
	if isWarrior444 || isWoodcutter333 || isWoodcutter222 {
		role.BattleSourceQuery = role.SourceQuery
	}
	return role
}

func normalizeRoleItems(items []RoleItem) []RoleItem {
	result := make([]RoleItem, 0, len(items))
	for _, item := range items {
		normalized := normalizeRoleItem(item)
		if normalized.Name != "" {
			result = append(result, normalized)
		}
	}
	sort.SliceStable(result, func(left int, right int) bool {
		if result[left].Type == result[right].Type {
			return result[left].Index < result[right].Index
		}
		return result[left].Type < result[right].Type
	})
	return result
}

func normalizeRoleTownBuffs(buffs []RoleTownBuff) []RoleTownBuff {
	result := make([]RoleTownBuff, 0, len(buffs))
	for _, buff := range buffs {
		normalized := normalizeRoleTownBuff(buff)
		if normalized.Name != "" {
			result = append(result, normalized)
		}
	}
	sort.SliceStable(result, func(left int, right int) bool {
		if result[left].EndTime == result[right].EndTime {
			return result[left].Name < result[right].Name
		}
		return result[left].EndTime < result[right].EndTime
	})
	return result
}

func normalizeRoleTownBuff(buff RoleTownBuff) RoleTownBuff {
	buff.Handle = strings.TrimSpace(buff.Handle)
	buff.Name = strings.TrimSpace(buff.Name)
	buff.Display = strings.TrimSpace(buff.Display)
	buff.Description = strings.TrimSpace(buff.Description)
	buff.SourceCapture = strings.TrimSpace(buff.SourceCapture)
	if buff.BattleOnly < 0 {
		buff.BattleOnly = 0
	}
	return buff
}

func defaultRoleItems() []RoleItem {
	return []RoleItem{starterAxeItem()}
}

func CapturedRoleItemTemplate(name string) (RoleItem, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return RoleItem{}, false
	}
	for _, item := range sourceStarterEquipmentItems() {
		if item.Name == name {
			return item, true
		}
	}
	for _, item := range capturedDefaultRoleItems() {
		if item.Name == name {
			return item, true
		}
	}
	for _, item := range capturedAdditionalRoleItemTemplates() {
		if item.Name == name {
			return item, true
		}
	}
	if item, ok := classicDataRoleItemTemplate(name); ok {
		return item, true
	}
	return RoleItem{}, false
}

func CapturedRoleItemTemplates() []RoleItem {
	items := []RoleItem{}
	items = append(items, sourceStarterEquipmentItems()...)
	items = append(items, capturedDefaultRoleItems()...)
	items = append(items, capturedAdditionalRoleItemTemplates()...)
	return cloneRoleItems(items)
}

func classicDataRoleItemTemplate(name string) (RoleItem, bool) {
	row, ok, err := classicdata.FindItemByName(name)
	if err != nil || !ok {
		return RoleItem{}, false
	}
	icon := strings.TrimSpace(row["icon"])
	itemType := strings.TrimSpace(row["item_type"])
	if itemType == "" {
		itemType = "own"
	}
	category := strings.TrimSpace(row["category"])
	if category == "" {
		category = itemType
	}
	maxStack := strings.TrimSpace(row["max_stack"])
	if maxStack == "" {
		maxStack = "1"
	}
	descriptionText := strings.TrimSpace(row["description"])
	assetFamily := strings.TrimSpace(row["asset_family"])
	templateType := "背包"
	itemLevel := 1
	if itemType == "equip" {
		templateType = "装备"
		itemLevel = 2
	}
	if strings.HasPrefix(descriptionText, "f_i_") {
		return RoleItem{
			Type:        templateType,
			Name:        name,
			ItemType:    itemType,
			Display:     icon,
			Description: descriptionText,
			Count:       1,
			Index:       classicDataRoleItemEquipmentSlot(name, category),
			ItemLevel:   itemLevel,
		}, true
	}
	description := fmt.Sprintf("f_i_%s^5BC46D&24@%s&25@%s", name, category, maxStack)
	if descriptionText != "" {
		description += "&20@" + descriptionText
	}
	if assetFamily != "" {
		description += "&27@" + assetFamily
	}
	if icon != "" {
		description += "&101@" + icon
	}
	description += "&103@0&104@0&105@&107@&108@0"
	return RoleItem{
		Type:        templateType,
		Name:        name,
		ItemType:    itemType,
		Display:     icon,
		Description: description,
		Count:       1,
		Index:       classicDataRoleItemEquipmentSlot(name, category),
		ItemLevel:   itemLevel,
	}, true
}

func classicDataRoleItemEquipmentSlot(name string, category string) int {
	if index, ok := capturedEquipmentSlotForName(name); ok {
		return index
	}
	switch {
	case strings.Contains(category, "宠物"):
		return rolePetEquipIndex
	case strings.Contains(category, "幻·时装"):
		return roleFashionEquipIndex
	case strings.Contains(category, "护具·头部"):
		return 0
	case strings.Contains(category, "护具·肩部"):
		return 1
	case strings.Contains(category, "护具·护腕"):
		return 2
	case strings.Contains(category, "武器"):
		return 3
	case strings.Contains(category, "护具·躯干"):
		return 4
	case strings.Contains(category, "护具·腿"):
		return 5
	case strings.Contains(category, "护具·腰部"):
		return 10
	case strings.Contains(category, "护具·足部"):
		return 12
	case strings.Contains(category, "法宝"):
		return roleTreasureEquipIndex
	case strings.Contains(category, "坐骑"):
		return roleMountEquipIndex
	default:
		return 0
	}
}

func capturedEquipmentSlotForName(name string) (int, bool) {
	switch strings.TrimSpace(name) {
	case "如意之戒", "骷髅戒指":
		return 6, true
	case "琉璃耳环", "银耳坠":
		return 7, true
	case "白玉项链", "翡翠项链":
		return 8, true
	default:
		return 0, false
	}
}

func CapturedRoleItemTemplateByID(itemID string) (RoleItem, bool) {
	itemID = strings.TrimSpace(strings.TrimSuffix(itemID, ".png"))
	if itemID == "" {
		return RoleItem{}, false
	}
	templates := CapturedRoleItemTemplates()
	for index, item := range templates {
		if strconv.Itoa(index+1) == itemID {
			return item, true
		}
		if strings.TrimSuffix(item.Display, ".png") == itemID {
			return item, true
		}
		if item.Name == itemID {
			return item, true
		}
	}
	return RoleItem{}, false
}

func capturedDefaultRoleItems() []RoleItem {
	items := []RoleItem{
		{
			Type:        "背包",
			Name:        "L小喇叭",
			ItemType:    "null",
			Display:     "194.png",
			Description: "f_i_L小喇叭^5BC46D&24@消耗品&25@9999&19@拥有世界喊话的权限&0;该物品不能交易&20@专门用来世界喊话的道具.&103@0&104@0&105@&107@&108@0",
			Count:       7,
			Index:       0,
			ItemLevel:   2,
		},
		{
			Type:        "背包",
			Name:        "L初阶经验卡",
			ItemType:    "own",
			Display:     "576.png",
			Description: "f_i_L初阶经验卡^00ccff&24@特殊&25@99&19@双击使用。击杀怪物后，将获得双倍经验，效果持续时间为1小时。该物品不能交易。&20@可以获得经验加成的神奇商城道具。&27@sitem_book&103@0&104@0&105@&107@&108@0",
			Count:       3,
			Index:       1,
			ItemLevel:   3,
		},
		{
			Type:        "背包",
			Name:        "传送晶体",
			ItemType:    "null",
			Display:     "778.png",
			Description: "f_i_传送晶体^f9e000&24@宝物&25@1&19@神秘的次元晶体可将使用者随意传送到指定的地区。\r<font color='#ffff00'>注:无法交易</font>&20@携带该宝物后&0;打开地图选择需要到达的地点可以瞬间传送至该处。\r<font color='#ff0000'>副本及据点无法传送</font>&27@sitem_bs&19@\n<font color='#59c5ca'>今日剩余次数:10/10</font>\n每日06:00点恢复\n&103@0&104@0&105@&107@&108@0",
			Count:       1,
			Index:       2,
			ItemLevel:   5,
		},
		{
			Type:        "背包",
			Name:        "L避怪符",
			ItemType:    "own",
			Display:     "574.png",
			Description: "f_i_L避怪符^00ccff&24@特殊&25@99&19@- 双击使用后&0;5分钟内不会遇敌。<br/>- 点击角色的避怪状态图标可取消状态。<br/><font color='#ffff00'>- 明怪无效。</font><br/><font color='#ffff00'>- 不可交易</font>&20@非常古老的驱魔咒符。&27@sitem_book&103@0&104@0&105@&107@&108@0",
			Count:       3,
			Index:       3,
			ItemLevel:   3,
		},
		{
			Type:        "背包",
			Name:        "黏液",
			ItemType:    "null",
			Display:     "132.png",
			Description: "f_i_黏液^ffffff&24@材料&25@99&20@怪物体内的黏液&0;有很强的粘性.&27@sitem_yaoji&103@0&104@0&105@&107@&108@15",
			Count:       3,
			Index:       4,
			ItemLevel:   1,
		},
		{
			Type:        "背包",
			Name:        "兽骨",
			ItemType:    "null",
			Display:     "72.png",
			Description: "f_i_兽骨&24@材料&25@99&20@野兽的骨&0;可制造工具&0;入药.&103@0&104@0&105@&107@&108@21",
			Count:       10,
			Index:       5,
			ItemLevel:   1,
		},
		{
			Type:        "背包",
			Name:        "兽血",
			ItemType:    "null",
			Display:     "73.png",
			Description: "f_i_兽血&24@材料&25@99&20@野兽的血液&0;用于装饰&0;也可入药.&27@sitem_water&103@0&104@0&105@&107@&108@19",
			Count:       7,
			Index:       6,
			ItemLevel:   1,
		},
		{
			Type:        "背包",
			Name:        "L花卷",
			ItemType:    "own",
			Display:     "213.png",
			Description: "f_i_L花卷^ffffff&24@消耗品&25@99&7@350&20@葱香的花卷馒头.食用后可恢复一些气力.&103@0&104@0&105@&107@&108@0",
			Count:       10,
			Index:       7,
			ItemLevel:   1,
		},
		{
			Type:        "背包",
			Name:        "肉",
			ItemType:    "own",
			Display:     "70.png",
			Description: "f_i_肉&24@材料 消耗品&25@99&7@50&20@动物的肉&0;直接食用能加些气力.&103@0&104@0&105@&107@&108@12",
			Count:       6,
			Index:       8,
			ItemLevel:   1,
		},
		{
			Type:        "背包",
			Name:        "丝",
			ItemType:    "null",
			Display:     "76.png",
			Description: "f_i_丝&24@材料&25@99&20@产自一些植物或者虫&0;缝制用基本原料.&27@sitem_ezhj&103@0&104@0&105@&107@&108@9",
			Count:       4,
			Index:       9,
			ItemLevel:   1,
		},
		{
			Type:        "背包",
			Name:        "银元宝",
			ItemType:    "own",
			Display:     "39.png",
			Description: "f_i_银元宝^C156C7&24@材料 消耗品&25@9999&19@双击可兑换为1000铜币&20@游戏中的货币&0;用于流通买卖&27@sitem_jhj&103@0&104@0&105@&107@&108@0",
			Count:       1,
			Index:       10,
			ItemLevel:   4,
		},
		{
			Type:        "背包",
			Name:        "毒囊",
			ItemType:    "null",
			Display:     "84.png",
			Description: "f_i_毒囊^ffffff&24@材料&25@99&20@怪物体内的有毒脏器。&103@0&104@0&105@&107@&108@19",
			Count:       3,
			Index:       11,
			ItemLevel:   1,
		},
		{
			Type:        "背包",
			Name:        "L回城咒",
			ItemType:    "own",
			Display:     "193.png",
			Description: "f_i_L回城咒&24@消耗品&25@99&19@双击使用直接回绑定点<br/><font color='#ffff00'>注：不可交易</font>&20@具有神秘力量的卷轴.&103@0&104@0&105@&107@&108@0",
			Count:       3,
			Index:       12,
			ItemLevel:   1,
		},
		{
			Type:        "背包",
			Name:        "15级礼盒",
			ItemType:    "own",
			Display:     "742.png",
			Description: "f_i_15级礼盒^00ccff&24@宝物&25@1&19@内置礼品：\r【20级礼盒】\r【L回魂丹x10】\r【L进阶经验卡x1】\r【L花卷x4】\r<font color='#ffff00'>注:【不能交易】【15级时双击打开】</font>&20@等级达到15级时，双击打开礼盒获取礼品。&103@0&104@0&105@&107@&108@0",
			Count:       1,
			Index:       13,
			ItemLevel:   3,
		},
		{
			Type:        "背包",
			Name:        "铜钱",
			ItemType:    "own",
			Display:     "163.png",
			Description: "f_i_铜钱^ffffff&24@材料 消耗品&25@1000&19@1000枚时双击可兑换为银元宝.&20@游戏中的货币&0;用于流通买卖.&27@sitem_tq&103@0&104@0&105@&107@&108@0",
			Count:       500,
			Index:       14,
			ItemLevel:   1,
		},
		{
			Type:        "背包",
			Name:        "毛皮",
			ItemType:    "null",
			Display:     "71.png",
			Description: "f_i_毛皮&24@材料&25@99&20@没经过加工的野兽毛皮&0;缝制用基本原料.&27@sitem_piput&103@0&104@0&105@&107@&108@24",
			Count:       6,
			Index:       15,
			ItemLevel:   1,
		},
		{
			Type:        "背包",
			Name:        "虫壳",
			ItemType:    "null",
			Display:     "83.png",
			Description: "f_i_虫壳&24@材料&25@99&20@虫的坚硬的外壳&0;用来制造防具和入药.&103@0&104@0&105@&107@&108@13",
			Count:       1,
			Index:       16,
			ItemLevel:   1,
		},
		{
			Type:        "背包",
			Name:        "普通采集手套",
			ItemType:    "null",
			Display:     "856.png",
			Description: "f_i_普通采集手套^ffffff&24@消耗品&25@999&20@平常的采集手套，带上进行采集得话可以很好的保护双手。&103@0&104@0&105@&107@&108@0",
			Count:       10,
			Index:       17,
			ItemLevel:   1,
		},
		{
			Type:        "背包",
			Name:        "朽木",
			ItemType:    "null",
			Display:     "915.png",
			Description: "f_i_朽木&24@材料&25@99&20@一块烂木头，看起来没有什么用处。&27@sitem_wood&103@0&104@0&105@&107@&108@0",
			Count:       1,
			Index:       18,
			ItemLevel:   1,
		},
	}
	return ensureStarterAxeItem(items)
}

func capturedAdditionalRoleItemTemplates() []RoleItem {
	return []RoleItem{
		{
			Type:        "背包",
			Name:        "超时空要塞",
			ItemType:    "equip",
			Display:     "1205.png",
			Description: "f_i_超时空要塞^00ccff&24@幻·时装&25@1&15@20&16@20&19@【注：7天时限】&20@交杂着爱与友情以及惑星之命运的超银河Love Story！！！ &27@sitem_ezhj&103@0&104@1779629952719&105@&107@&108@0",
			Count:       1,
			Index:       0,
			ItemLevel:   3,
		},
		{
			Type:        "背包",
			Name:        "盛夏缤纷",
			ItemType:    "equip",
			Display:     "729.png",
			Description: "f_i_盛夏缤纷^5BC46D&24@幻·时装&25@1&19@男性：黑背心时尚牛仔裤。\r女性：彩虹肩带短裙。&20@盛夏时尚服饰系列之一。&27@sitem_ezhj&103@0&104@0&105@&107@&108@0",
			Count:       1,
			Index:       0,
			ItemLevel:   2,
		},
		{
			Type:        "背包",
			Name:        "盗贼的首级",
			ItemType:    "null",
			Display:     "120.png",
			Description: "f_i_盗贼的首级^5BC46D&24@材料&25@99&20@击杀盗贼的证明.&103@0&104@0&105@&107@&108@15",
			Count:       1,
			Index:       0,
			ItemLevel:   2,
		},
		{
			Type:        "背包",
			Name:        "点券",
			ItemType:    "null",
			Display:     "659.png",
			Description: "f_i_点券^f9e000&24@特殊&25@9999&19@特殊消费或商城购物。&20@游戏中的换购券&0;用于流通买卖。&27@sitem_book&101@659.png&103@0&104@0&105@&107@&108@0",
			Count:       1,
			Index:       0,
			ItemLevel:   5,
		},
		{
			Type:        "背包",
			Name:        "雪莲花",
			ItemType:    "null",
			Display:     "935.png",
			Description: "f_i_雪莲花^5BC46D&24@材料&25@99&19@任务物品。&20@生长于高山雪原的珍贵药材。&27@sitem_book&103@0&104@0&105@&107@&108@0",
			Count:       1,
			Index:       0,
			ItemLevel:   2,
		},
		{
			Type:        "背包",
			Name:        "黄风腰带",
			ItemType:    "equip",
			Display:     "547.png",
			Description: "f_i_黄风腰带^5BC46D&24@护具&25@1&21@25&3@12&4@5&5@2%&27@sitem_jhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2\n[精炼+1] 每升一级 魔法防御+2\n[精炼+1] 每升一级 回避+3&103@0&104@0&105@&107@&108@160",
			Count:       1,
			Index:       0,
			ItemLevel:   2,
		},
		{
			Type:     "装备",
			Name:     "黄风围巾",
			ItemType: "equip",
			Display:  "548.png",
			Description: `f_i_黄风围巾^00ccff&23@凿孔上限 9 格&24@护具·头部&25@1&21@28&3@19&4@12&5@100&13@7<$jstr>&27@sitem_ezhj&19@精炼潜质:
[精炼+1] 每升一级 物理防御+3
[精炼+1] 每升一级 魔法防御+3
[精炼+1] 每升一级 命中+10
[精炼+6] 每升一级 幸运+3
<font color='#00cc00'>特殊效果:
麻痹抗性:+5%
眩晕抗性:+5%
冰冻抗性:+8%</font>
[精炼+1] 每升一级 麻痹抗性+1%
[精炼+1] 每升一级 眩晕抗性+1%
[精炼+1] 每升一级 冰冻抗性+1%&103@0&104@0&105@&107@&108@160`,
			Count:     1,
			Index:     0,
			ItemLevel: 3,
		},
		{
			Type:     "装备",
			Name:     "万相",
			ItemType: "equip",
			Display:  "58.png",
			Description: `f_i_万相^5BC46D&23@凿孔上限 9 格&24@武器·弓系&25@1&21@40&22@游侠&1@88&9@35&27@sitem_wood&19@精炼潜质:
[精炼+1] 每升一级 物理攻击+2
[精炼+6] 每升一级 敏捷+2&103@0&104@0&105@&107@&108@550`,
			Count:     1,
			Index:     3,
			ItemLevel: 2,
		},
		{
			Type:     "装备",
			Name:     "狼人护肩",
			ItemType: "equip",
			Display:  "537.png",
			Description: `f_i_狼人护肩^5BC46D&23@凿孔上限 9 格&24@护具·肩部&25@1&21@23&3@15&13@3&15@3&27@sitem_ezhj&19@精炼潜质:
[精炼+1] 每升一级 物理防御+2
[精炼+6] 每升一级 爆击+50&103@0&104@0&105@&107@&108@220`,
			Count:     1,
			Index:     1,
			ItemLevel: 2,
		},
		{
			Type:     "装备",
			Name:     "骷髅戒指",
			ItemType: "equip",
			Display:  "759.png",
			Description: `f_i_骷髅戒指^5BC46D&23@凿孔上限 9 格&24@饰品&25@1&21@40&3@1&13@3&27@sitem_bs&19@精炼潜质:
[精炼+1] 每升一级 物理防御+2
<font color='#00cc00'>特殊效果:
封印抗性:+10%</font>
[精炼+1] 每升一级 封印抗性+1%&103@0&104@0&105@&107@&108@320`,
			Count:     1,
			Index:     6,
			ItemLevel: 2,
		},
		{
			Type:     "装备",
			Name:     "银耳坠",
			ItemType: "equip",
			Display:  "762.png",
			Description: `f_i_银耳坠^5BC46D&23@凿孔上限 9 格&24@饰品&25@1&21@40&3@1&10@3&27@sitem_bs&19@精炼潜质:
[精炼+1] 每升一级 物理防御+2
<font color='#00cc00'>特殊效果:
眩晕抗性:+10%</font>
[精炼+1] 每升一级 眩晕抗性+1%&103@0&104@0&105@&107@&108@320`,
			Count:     1,
			Index:     7,
			ItemLevel: 2,
		},
		{
			Type:     "装备",
			Name:     "翡翠项链",
			ItemType: "equip",
			Display:  "760.png",
			Description: `f_i_翡翠项链^5BC46D&23@凿孔上限 9 格&24@饰品&25@1&21@40&3@1&14@3<$jintt>&27@sitem_bs&19@精炼潜质:
[精炼+1] 每升一级 物理防御+2
<font color='#00cc00'>特殊效果:
混乱抗性:+10%</font>
[精炼+1] 每升一级 混乱抗性+1%&103@0&104@0&105@&107@&108@340`,
			Count:     1,
			Index:     8,
			ItemLevel: 2,
		},
		{
			Type:        "背包",
			Name:        "红方巾",
			ItemType:    "null",
			Display:     "121.png",
			Description: "f_i_红方巾^ffffff&24@材料&25@99&20@红色绸缎制方巾.&27@sitem_ezhj&103@0&104@0&105@&107@&108@21",
			Count:       1,
			Index:       0,
			ItemLevel:   1,
		},
		{
			Type:        "背包",
			Name:        "绸缎",
			ItemType:    "null",
			Display:     "79.png",
			Description: "f_i_绸缎^ffffff&24@材料&25@99&20@代表高贵的布料.颜色光滑亮丽&0;五彩缤纷.&27@sitem_ezhj&103@0&104@0&105@&107@&108@24",
			Count:       1,
			Index:       0,
			ItemLevel:   1,
		},
		{
			Type:        "背包",
			Name:        "刺",
			ItemType:    "null",
			Display:     "134.png",
			Description: "f_i_刺^ffffff&24@材料&25@99&20@一些植物自带的锋利尖锐的刺状物.&27@sitem_piput&103@0&104@0&105@&107@&108@19",
			Count:       1,
			Index:       0,
			ItemLevel:   1,
		},
		{
			Type:        "背包",
			Name:        "红缨",
			ItemType:    "null",
			Display:     "77.png",
			Description: "f_i_红缨&24@材料&25@99&20@丝线染色后制成&0;用于武器和护具的装饰.&27@sitem_ezhj&103@0&104@0&105@&107@&108@24",
			Count:       1,
			Index:       0,
			ItemLevel:   1,
		},
		{
			Type:        "背包",
			Name:        "水帘洞通行证",
			ItemType:    "null",
			Display:     "781.png",
			Description: "f_i_水帘洞通行证^00ccff&24@消耗品&25@99&19@<font color='#00ff00'>进入水帘洞修炼的通行证.</font>&27@sitem_book&103@0&104@0&105@&107@&108@150",
			Count:       1,
			Index:       0,
			ItemLevel:   3,
		},
		{
			Type:        "背包",
			Name:        "黄风寨通行证",
			ItemType:    "null",
			Display:     "783.png",
			Description: "f_i_黄风寨通行证^00ccff&24@消耗品&25@99&19@<font color='#00ff00'>进入黄风寨修炼的通行证.</font>&27@sitem_book&103@0&104@0&105@&107@&108@150",
			Count:       1,
			Index:       0,
			ItemLevel:   3,
		},
		{
			Type:        "背包",
			Name:        "飞仙洞通行证",
			ItemType:    "null",
			Display:     "782.png",
			Description: "f_i_飞仙洞通行证^00ccff&24@消耗品&25@99&19@<font color='#00ff00'>进入飞仙洞修炼的通行证.</font>&27@sitem_book&103@0&104@0&105@&107@&108@150",
			Count:       1,
			Index:       0,
			ItemLevel:   3,
		},
		{
			Type:        "背包",
			Name:        "狮虎窟通行证",
			ItemType:    "null",
			Display:     "425.png",
			Description: "f_i_狮虎窟通行证^00ccff&24@消耗品&25@99&19@<font color='#00ff00'>进入狮虎窟修炼的通行证.</font>&27@sitem_book&101@425.png&103@0&104@0&105@&107@&108@165",
			Count:       1,
			Index:       0,
			ItemLevel:   3,
		},
		{
			Type:        "背包",
			Name:        "当归",
			ItemType:    "null",
			Display:     "92.png",
			Description: "f_i_当归^ffffff&24@材料&25@99&20@若发生气血逆乱&0;服用之后即可降逆定乱&0;使气血各有所归.&101@92.png&103@0&104@0&105@&107@&108@20",
			Count:       1,
			Index:       0,
			ItemLevel:   1,
		},
		{
			Type:        "背包",
			Name:        "黄连",
			ItemType:    "null",
			Display:     "95.png",
			Description: "f_i_黄连^ffffff&24@材料&25@99&20@清热燥湿&0;泻火解毒.用于湿热痞满&0;呕吐吞酸&0;泻痢&0;黄疸&0;高热神昏&0;心火亢盛&0;心烦不寐&0;血热吐衄&0;目赤&0;牙痛&0;消渴&0;痈肿疔疮.&101@95.png&103@0&104@0&105@&107@&108@11",
			Count:       1,
			Index:       0,
			ItemLevel:   1,
		},
		{
			Type:        "背包",
			Name:        "金银花",
			ItemType:    "null",
			Display:     "97.png",
			Description: "f_i_金银花^ffffff&24@材料&25@99&20@性寒味甘.具有清热解毒&0;凉血化淤的功效.&101@97.png&103@0&104@0&105@&107@&108@13",
			Count:       1,
			Index:       0,
			ItemLevel:   1,
		},
		{
			Type:        "背包",
			Name:        "碎铁矿",
			ItemType:    "null",
			Display:     "105.png",
			Description: "f_i_碎铁矿^ffffff&24@材料&25@99&20@制造武器和护具的基本素材&101@105.png&103@0&104@0&105@&107@&108@12",
			Count:       1,
			Index:       0,
			ItemLevel:   1,
		},
		{
			Type:        "背包",
			Name:        "废渣",
			ItemType:    "null",
			Display:     "103.png",
			Description: "f_i_废渣^ffffff&24@材料&25@99&20@一堆看似没有用的东西&0;在大多数情况下&0;完全没有任何价值&101@103.png&103@0&104@0&105@&107@&108@1",
			Count:       1,
			Index:       0,
			ItemLevel:   1,
		},
		{
			Type:        "背包",
			Name:        "石块",
			ItemType:    "null",
			Display:     "104.png",
			Description: "f_i_石块&24@材料&25@99&20@随处可见的石块&0;可造工具&0;也可用于建筑.&101@104.png&103@0&104@0&105@&107@&108@10",
			Count:       1,
			Index:       0,
			ItemLevel:   1,
		},
		{
			Type:        "背包",
			Name:        "水晶",
			ItemType:    "null",
			Display:     "111.png",
			Description: "f_i_水晶^ffffff&24@材料&25@99&20@晶莹剔透的天然水晶.是自然界的一个奇迹.&101@111.png&103@0&104@0&105@&107@&108@100",
			Count:       1,
			Index:       0,
			ItemLevel:   1,
		},
		{
			Type:        "背包",
			Name:        "碎金片",
			ItemType:    "null",
			Display:     "106.png",
			Description: "f_i_碎金片^5BC46D&24@材料&25@99&20@金色的碎片&0;看上去很值钱的样子.&101@106.png&103@0&104@0&105@&107@&108@180",
			Count:       1,
			Index:       0,
			ItemLevel:   2,
		},
		{
			Type:        "背包",
			Name:        "岩魔菱石",
			ItemType:    "null",
			Display:     "112.png",
			Description: "f_i_岩魔菱石^00ccff&24@材料&25@99&20@晶莹剔透的多角矿石&0;反射出令人惊叹的蓝色光芒.&27@sitem_rock&101@112.png&103@0&104@0&105@&107@&108@132",
			Count:       1,
			Index:       0,
			ItemLevel:   3,
		},
		{
			Type:        "背包",
			Name:        "岩魔球石",
			ItemType:    "own",
			Display:     "113.png",
			Description: "f_i_岩魔球石^5BC46D&24@材料&25@99&20@巨岩魔的球型石块&0;用于锻造武器.&27@sitem_rock&101@113.png&103@0&104@0&105@&107@&108@110",
			Count:       1,
			Index:       0,
			ItemLevel:   2,
		},
		{
			Type:        "背包",
			Name:        "巨岩魔的拳",
			ItemType:    "own",
			Display:     "114.png",
			Description: "f_i_巨岩魔的拳^5BC46D&24@材料&25@99&20@巨岩魔的拳型石块&0;用于锻造武器.&101@114.png&103@0&104@0&105@&107@&108@290",
			Count:       1,
			Index:       0,
			ItemLevel:   2,
		},
		{
			Type:        "背包",
			Name:        "巨岩魔的头",
			ItemType:    "null",
			Display:     "115.png",
			Description: "f_i_巨岩魔的头^00ccff&24@材料&25@99&20@巨岩魔的头颅&0;是用于锻造护具的材料&27@sitem_zgbs&101@115.png&103@0&104@0&105@&107@&108@280",
			Count:       1,
			Index:       0,
			ItemLevel:   3,
		},
		{
			Type:        "背包",
			Name:        "宝匣",
			ItemType:    "own",
			Display:     "596.png",
			Description: "f_i_宝匣^00ccff&24@宝物&25@99&19@双击打开后可能获得一个小惊喜。&20@看起来比较小巧的褐色木质匣子，不知道里面放着什么样的物品。&27@sitem_wood&101@596.png&103@0&104@0&105@&107@&108@0",
			Count:       1,
			Index:       0,
			ItemLevel:   3,
		},
		{
			Type:        "装备",
			Name:        "岩魔剑",
			ItemType:    "equip",
			Display:     "606.png",
			Description: "f_i_岩魔剑^5BC46D&23@凿孔上限 9 格&24@武器·单剑系&25@1&21@27&22@战士&1@65&16@7&17@5&27@sitem_jwep&19@精炼潜质:\n[精炼+1] 每升一级 物理攻击+2\n[精炼+1] 每升一级 命中+4\n[精炼+6] 每升一级 爆击+25&101@606.png&103@0&104@0&105@&107@&108@380",
			Count:       1,
			Index:       3,
			ItemLevel:   2,
		},
		{
			Type:        "装备",
			Name:        "岩化护腿",
			ItemType:    "equip",
			Display:     "598.png",
			Description: "f_i_岩化护腿^ffffff&23@凿孔上限 9 格&24@护具·腿&25@1&21@24&3@25&27@sitem_jhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2\n<font color='#00cc00'>特殊效果:\n外伤抗性:+10%</font>&101@598.png&103@0&104@0&105@&107@&108@260",
			Count:       1,
			Index:       5,
			ItemLevel:   1,
		},
		{
			Type:        "装备",
			Name:        "蓝晶护肩",
			ItemType:    "equip",
			Display:     "603.png",
			Description: "f_i_蓝晶护肩^5BC46D&23@凿孔上限 9 格&24@护具·肩部&25@1&21@18&3@8&4@12&14@4<$jintt>&15@2&27@sitem_jhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2\n[精炼+1] 每升一级 魔法防御+2\n[精炼+6] 每升一级 敏捷+2&101@603.png&103@0&104@0&105@&107@&108@210",
			Count:       1,
			Index:       1,
			ItemLevel:   2,
		},
		{
			Type:        "装备",
			Name:        "呼啸战靴",
			ItemType:    "equip",
			Display:     "502.png",
			Description: "f_i_呼啸战靴^5BC46D&23@凿孔上限 9 格&24@护具·足部&25@1&21@26&3@15&12@10&15@6&27@sitem_piput&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2&103@0&104@0&105@&107@&108@220",
			Count:       1,
			Index:       12,
			ItemLevel:   2,
		},
		{
			Type:        "装备",
			Name:        "寨夫人上衣",
			ItemType:    "equip",
			Display:     "474.png",
			Description: "f_i_寨夫人上衣^5BC46D&23@凿孔上限 9 格&24@护具·躯干&25@1&21@26&3@31&4@20&15@5&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2\n[精炼+1] 每升一级 魔法防御+2\n[精炼+1] 每升一级 回避+5\n<font color='#00cc00'>特殊效果:\n混乱抗性:+30%&103@0&104@0&105@&107@&108@250",
			Count:       1,
			Index:       4,
			ItemLevel:   2,
		},
		{
			Type:        "装备",
			Name:        "寨夫人护腕",
			ItemType:    "equip",
			Display:     "467.png",
			Description: "f_i_寨夫人护腕^5BC46D&23@凿孔上限 9 格&24@护具·护腕&25@1&21@24&3@14&4@4&9@10&11@10&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2\n[精炼+1] 每升一级 魔法防御+2\n[精炼+1] 每升一级 气力上限+10&103@0&104@0&105@&107@&108@230",
			Count:       1,
			Index:       2,
			ItemLevel:   2,
		},
		{
			Type:        "装备",
			Name:        "刀客布衣",
			ItemType:    "equip",
			Display:     "540.png",
			Description: "f_i_刀客布衣^5BC46D&23@凿孔上限 9 格&24@护具·躯干&25@1&21@16&3@20&4@6&15@4&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2\n[精炼+1] 每升一级 魔法防御+2\n[精炼+6] 每升一级 耐力+1&103@0&104@0&105@&107@&108@120",
			Count:       1,
			Index:       4,
			ItemLevel:   2,
		},
		{
			Type:        "装备",
			Name:        "剔骨刀",
			ItemType:    "equip",
			Display:     "552.png",
			Description: "f_i_剔骨刀^5BC46D&23@凿孔上限 9 格&24@武器·匕首系&25@1&21@15&22@游侠&1@38&15@2<$jagi>&17@3<$jlck>&27@sitem_jwep&19@精炼潜质:\n[精炼+1] 每升一级 物理攻击+2\n[精炼+6] 每升一级 回避+2\n<font color='#00cc00'>特殊效果:\n击中敌人时&0;有2%的机率使敌人进入外伤状态3回合(每回合损失气力15~30点)</font>&103@0&104@0&105@&107@&108@142",
			Count:       1,
			Index:       3,
			ItemLevel:   2,
		},
		{
			Type:        "背包",
			Name:        "图腾面具",
			ItemType:    "null",
			Display:     "135.png",
			Description: "f_i_图腾面具^5BC46D&24@材料&25@99&20@看似已经被下了诅咒的面具&0;有点毛骨悚然.&27@sitem_wood&103@0&104@0&105@&107@&108@34",
			Count:       1,
			Index:       0,
			ItemLevel:   2,
		},
		{
			Type:        "背包",
			Name:        "兽牙",
			ItemType:    "null",
			Display:     "68.png",
			Description: "f_i_兽牙&24@材料&25@99&20@野兽锋利的牙齿&0;看起来能做点什么.&101@68.png&103@0&104@0&105@&107@&108@15",
			Count:       1,
			Index:       0,
			ItemLevel:   1,
		},
		{
			Type:        "背包",
			Name:        "头骨",
			ItemType:    "null",
			Display:     "102.png",
			Description: "f_i_头骨^ffffff&24@材料&25@99&20@尸骨的头部&0;用于制造工具&0;装饰服饰.&27@sitem_wood&101@102.png&103@0&104@0&105@&107@&108@18",
			Count:       1,
			Index:       0,
			ItemLevel:   2,
		},
	}
}

func removeCapturedDefaultBagSeeds(items []RoleItem) []RoleItem {
	capturedSeeds := capturedDefaultRoleItems()
	if !shouldRemoveCapturedDefaultBagSeeds(items, capturedSeeds) {
		return items
	}
	result := make([]RoleItem, 0, len(items))
	for _, item := range items {
		if isCapturedDefaultBagSeed(item, capturedSeeds) {
			continue
		}
		result = append(result, item)
	}
	return result
}

func shouldRemoveCapturedDefaultBagSeeds(items []RoleItem, capturedSeeds []RoleItem) bool {
	seedCount := 0
	for _, item := range items {
		if item.Name == "铁斧" {
			continue
		}
		if isCapturedDefaultBagSeed(item, capturedSeeds) {
			seedCount += 1
			continue
		}
		return false
	}
	return seedCount > 0
}

func isCapturedDefaultBagSeed(item RoleItem, capturedSeeds []RoleItem) bool {
	for _, seed := range capturedSeeds {
		if seed.Name == "铁斧" {
			continue
		}
		if item.Type == seed.Type &&
			item.Index == seed.Index &&
			item.Name == seed.Name &&
			item.Display == seed.Display {
			return true
		}
	}
	return false
}

func ensureStarterAxeItem(items []RoleItem) []RoleItem {
	for _, item := range items {
		if item.Name == "铁斧" {
			return items
		}
	}
	if len(items) > 0 {
		return items
	}
	return append(items, starterAxeItem())
}

func starterAxeItem() RoleItem {
	item := sourceIronAxeItem()
	item.Type = "背包"
	item.Index = 19
	return item
}

func sourceStarterEquipmentItems() []RoleItem {
	return []RoleItem{
		sourceIronAxeItem(),
		sourceBlueClothItem(),
		sourceBluePantsItem(),
		sourceClothShoesItem(),
	}
}

func sourceIronAxeItem() RoleItem {
	return RoleItem{
		Type:        "装备",
		Name:        "铁斧",
		ItemType:    "equip",
		Display:     "29.png",
		Description: "f_i_铁斧&23@凿孔上限 9 格&23@凿孔上限 9 格&24@武器·单刀系&25@1&21@1&1@26&27@sitem_jhj&19@精炼潜质:\n[精炼+1] 每升一级 物理攻击+2&103@0&104@0&105@&107@&108@0",
		Count:       1,
		Index:       3,
		ItemLevel:   1,
	}
}

func sourceBlueClothItem() RoleItem {
	return RoleItem{
		Type:        "装备",
		Name:        "蓝布衣",
		ItemType:    "equip",
		Display:     "291.png",
		Description: "f_i_蓝布衣^ffffff&23@凿孔上限 9 格&24@护具·躯干&25@1&21@1&3@<$pdef><$jpdef>&27@sitem_ezhj&21@1&3@6&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2\n&103@0&104@0&105@&107@&108@0",
		Count:       1,
		Index:       4,
		ItemLevel:   1,
	}
}

func sourceBluePantsItem() RoleItem {
	return RoleItem{
		Type:        "装备",
		Name:        "蓝布裤",
		ItemType:    "equip",
		Display:     "3.png",
		Description: "f_i_蓝布裤^ffffff&23@凿孔上限 9 格&24@护具·腿&25@1&21@1&3@4&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2&103@0&104@0&105@&107@&108@0",
		Count:       1,
		Index:       5,
		ItemLevel:   1,
	}
}

func sourceClothShoesItem() RoleItem {
	return RoleItem{
		Type:        "装备",
		Name:        "布鞋",
		ItemType:    "equip",
		Display:     "274.png",
		Description: "f_i_布鞋^ffffff&23@凿孔上限 9 格&24@护具·足部&25@1&21@1&3@2&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2&103@0&104@0&105@&107@&108@0&18@2",
		Count:       1,
		Index:       12,
		ItemLevel:   1,
	}
}

func resolveVisualRoleID(presetID int) int {
	if presetID > 0 {
		return presetID
	}

	return 1
}
