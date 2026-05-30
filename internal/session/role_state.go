package session

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
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

func cloneRoleSkills(skills []RoleSkill) []RoleSkill {
	if len(skills) == 0 {
		return []RoleSkill{}
	}

	result := make([]RoleSkill, len(skills))
	copy(result, skills)
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
	if item.Display == "" {
		item.Display = template.Display
	}
	if item.ItemType == "" {
		item.ItemType = template.ItemType
	}
	if item.Description == "" || item.Description == genericCollectionRewardDescription(item.Name) {
		item.Description = template.Description
	}
	if item.ItemLevel <= 0 {
		item.ItemLevel = template.ItemLevel
	}
	return item
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
	return PlayerBaseData{
		PlayerID:     playerID,
		RoleID:       role.RoleID,
		DisplayName:  role.DisplayName,
		Level:        role.Level,
		Exp:          role.Exp,
		Voc:          role.Voc,
		HP:           roleState.HP,
		MP:           roleState.MP,
		MaxHP:        rolePhysique.MaxHP,
		MaxMP:        rolePhysique.MaxMP,
		MapID:        role.MapID,
		VisualRoleID: role.VisualRoleID,
		PresetID:     role.PresetID,
		SourceQuery:  role.SourceQuery,
		Appearance:   role.Appearance,
		Currencies:   cloneRoleCurrencies(role.Currencies),
		RoleState:    &roleState,
		RolePhysique: &rolePhysique,
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
		existing.Description == incoming.Description &&
		existing.Level == incoming.Level &&
		existing.EndTime == incoming.EndTime &&
		existing.Owner == incoming.Owner &&
		existing.ItemLevel == incoming.ItemLevel
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
	if role.Voc == "" {
		role.Voc = defaultRoleVoc
	}
	role.Skills = ensureDefaultRoleSkills(role.Skills)
	if len(role.Currencies) == 0 {
		role.Currencies = defaultRoleCurrencies()
	} else {
		role.Currencies = normalizeRoleCurrencies(role.Currencies)
	}
	if len(role.Items) == 0 {
		role.Items = defaultRoleItems()
	} else {
		role.Items = ensureStarterAxeItem(removeCapturedDefaultBagSeeds(normalizeRoleItems(role.Items)))
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
	return RoleItem{}, false
}

func CapturedRoleItemTemplates() []RoleItem {
	items := []RoleItem{}
	items = append(items, sourceStarterEquipmentItems()...)
	items = append(items, capturedDefaultRoleItems()...)
	items = append(items, capturedAdditionalRoleItemTemplates()...)
	return cloneRoleItems(items)
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
	}
}

func removeCapturedDefaultBagSeeds(items []RoleItem) []RoleItem {
	capturedSeeds := capturedDefaultRoleItems()
	result := make([]RoleItem, 0, len(items))
	for _, item := range items {
		if isCapturedDefaultBagSeed(item, capturedSeeds) {
			continue
		}
		result = append(result, item)
	}
	return result
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
