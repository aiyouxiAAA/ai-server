package main

import (
	"strconv"
	"strings"

	"ai-server/internal/session"
)

type sourceItemShopRoute struct {
	handle       string
	answerHandle string
	title        string
	vocation     string
	rows         string
}

type sourceItemShopRow struct {
	id           int
	name         string
	targetType   string
	icon         string
	description  string
	count        int
	level        int
	requirements []classicTownSkillShopRequirement
}

func buildClassicTownItemShopResult(request classicTownAnswerRequest) (packetResult, bool) {
	route, ok := sourceGuangqingItemShopRoutes[request.Handle]
	if !ok || request.AnswerHandle != route.answerHandle {
		return packetResult{}, false
	}
	entries := sourceItemShopEntries(route.vocation, route.rows)
	shop := classicTownSkillShopPush{
		Handle:   route.handle,
		ShopID:   "item:" + route.handle,
		Title:    route.title,
		Vocation: route.vocation,
		SkillCap: len(entries),
		Skills:   entries,
	}
	return packetResult{
		skillShop: &shop,
		handled:   true,
	}, true
}

func sourceItemShopEntries(vocation string, rows string) []classicTownSkillShopEntry {
	lines := strings.Split(strings.TrimSpace(rows), "\n")
	entries := make([]classicTownSkillShopEntry, 0, len(lines))
	for _, line := range lines {
		row, ok := parseSourceItemShopRow(line)
		if !ok {
			continue
		}
		entries = append(entries, classicTownSkillShopEntry{
			ID:           row.id,
			Name:         row.name,
			Owned:        false,
			Icon:         row.icon,
			Category:     sourceItemCategory(row.description),
			Vocation:     vocation,
			TargetType:   row.targetType,
			Description:  row.description,
			Requirements: row.requirements,
		})
	}
	return entries
}

func findSourceItemShopRow(shopID string, itemID int) (sourceItemShopRow, bool) {
	handle, ok := strings.CutPrefix(strings.TrimSpace(shopID), "item:")
	if !ok {
		return sourceItemShopRow{}, false
	}
	route, ok := sourceGuangqingItemShopRoutes[handle]
	if !ok {
		return sourceItemShopRow{}, false
	}
	for _, line := range strings.Split(strings.TrimSpace(route.rows), "\n") {
		row, ok := parseSourceItemShopRow(line)
		if ok && row.id == itemID {
			return row, true
		}
	}
	return sourceItemShopRow{}, false
}

func sourceItemShopRowToRoleItem(row sourceItemShopRow) session.RoleItem {
	return session.RoleItem{
		Type:        "背包",
		Name:        row.name,
		ItemType:    row.targetType,
		Display:     row.icon,
		Description: row.description,
		Count:       row.count,
		Index:       -1,
		ItemLevel:   row.level,
	}
}

func sourceItemShopRequirementsToRoleItemRequirements(requirements []classicTownSkillShopRequirement) []session.RoleItemRequirement {
	result := make([]session.RoleItemRequirement, 0, len(requirements))
	for _, requirement := range requirements {
		name := strings.TrimSpace(requirement.Name)
		if name == "" || requirement.Count <= 0 {
			continue
		}
		result = append(result, session.RoleItemRequirement{
			Name:  name,
			Count: requirement.Count,
		})
	}
	return result
}

func parseSourceItemShopRow(line string) (sourceItemShopRow, bool) {
	columns := strings.Split(line, "|")
	if len(columns) < 7 {
		return sourceItemShopRow{}, false
	}
	id, err := strconv.Atoi(columns[0])
	if err != nil {
		return sourceItemShopRow{}, false
	}
	count, err := strconv.Atoi(columns[5])
	if err != nil {
		return sourceItemShopRow{}, false
	}
	level, err := strconv.Atoi(columns[6])
	if err != nil {
		return sourceItemShopRow{}, false
	}
	row := sourceItemShopRow{
		id:          id,
		name:        columns[1],
		targetType:  columns[2],
		icon:        columns[3],
		description: columns[4],
		count:       count,
		level:       level,
	}
	for index := 7; index+2 < len(columns); index += 3 {
		reqCount, err := strconv.Atoi(columns[index+2])
		if err != nil {
			continue
		}
		row.requirements = append(row.requirements, classicTownSkillShopRequirement{
			Name:  columns[index],
			Icon:  columns[index+1],
			Count: reqCount,
		})
	}
	return row, true
}

func sourceItemCategory(description string) string {
	const marker = "&24@"
	start := strings.Index(description, marker)
	if start < 0 {
		return "商品"
	}
	start += len(marker)
	end := strings.Index(description[start:], "&")
	if end < 0 {
		return description[start:]
	}
	return description[start : start+end]
}

var sourceGuangqingItemShopRoutes = map[string]sourceItemShopRoute{
	"7000542609490978": {
		handle:       "7000542609490978",
		answerHandle: "1",
		title:        "丑七品的道具商店",
		vocation:     "道具",
		rows:         sourceYunyinGroceryShopRows,
	},
	"4090542614314425": {
		handle:       "4090542614314425",
		answerHandle: "1",
		title:        "丑六品的道具商店",
		vocation:     "道具",
		rows:         sourceDafoGroceryShopRows,
	},
	"4960542616750900": {
		handle:       "4960542616750900",
		answerHandle: "1",
		title:        "介象的道具商店",
		vocation:     "道具",
		rows:         sourceJiantingGroceryShopRows,
	},
	"1780542610743555": {
		handle:       "1780542610743555",
		answerHandle: "1",
		title:        "伏天的武器商店",
		vocation:     "武器",
		rows:         sourceGuangqingWeaponShopRows,
	},
	"4000542609162635": {
		handle:       "4000542609162635",
		answerHandle: "1",
		title:        "布衣娘的护具商店",
		vocation:     "护具",
		rows:         sourceYunyinArmorShopRows,
	},
	"1820542611400955": {
		handle:       "1820542611400955",
		answerHandle: "1",
		title:        "丑五品的道具商店",
		vocation:     "道具",
		rows:         sourceGuangqingGroceryShopRows,
	},
	"1830542611405809": {
		handle:       "1830542611405809",
		answerHandle: "1",
		title:        "八卦炉合成",
		vocation:     "合成",
		rows:         sourceGuangqingCraftShopRows,
	},
	"2500542613172144": {
		handle:       "2500542613172144",
		answerHandle: "1",
		title:        "云衣娘的护具商店",
		vocation:     "护具",
		rows:         sourceGuangqingArmorShopRows,
	},
	"2520542613299551": {
		handle:       "2520542613299551",
		answerHandle: "1",
		title:        "无颜的药品商店",
		vocation:     "医疗",
		rows:         sourceGuangqingHealerShopRows,
	},
	"6360542618722932": {
		handle:       "6360542618722932",
		answerHandle: "1",
		title:        "虚中的药品商店",
		vocation:     "医疗",
		rows:         sourceGuangqingHealerShopRows,
	},
}

const sourceGuangqingWeaponShopRows = `
0|蛮力钢剑|equip|40.png|f_i_蛮力钢剑&23@凿孔上限 9 格&24@武器·单剑系&25@1&21@20&22@战士&1@52&16@2&27@sitem_jwep&19@精炼潜质:[精炼+1] 每升一级 物理攻击+2&103@0&104@0&105@&107@&108@130|1|1|铜钱|163.png|500
1|逆天棍|equip|41.png|f_i_逆天棍&23@凿孔上限 9 格&24@武器·棍系&25@1&21@20&22@战士&1@48&16@2&27@sitem_jwep&19@精炼潜质:[精炼+1] 每升一级 物理攻击+2&103@0&104@0&105@&107@&108@128|1|1|铜钱|163.png|500
2|刎刀|equip|42.png|f_i_刎刀&23@凿孔上限 9 格&24@武器·单刀系&25@1&21@20&22@战士&1@58&16@2&27@sitem_jwep&19@精炼潜质:[精炼+1] 每升一级 物理攻击+2&103@0&104@0&105@&107@&108@124|1|1|铜钱|163.png|500
3|威武弓|equip|43.png|f_i_威武弓&23@凿孔上限 9 格&24@武器·弓系&25@1&21@20&22@游侠&1@50&16@2&27@sitem_jwep&19@精炼潜质:[精炼+1] 每升一级 物理攻击+2&103@0&104@0&105@&107@&108@132|1|1|铜钱|163.png|500
4|牙刺|equip|44.png|f_i_牙刺&23@凿孔上限 9 格&24@武器·暗器系&25@1&21@20&22@游侠&1@44&16@2&27@sitem_jwep&19@精炼潜质:[精炼+1] 每升一级 物理攻击+2&103@0&104@0&105@&107@&108@126|1|1|铜钱|163.png|500
5|天极拳套|equip|45.png|f_i_天极拳套&23@凿孔上限 9 格&24@武器·拳套系&25@1&21@20&22@游侠&1@46&16@2&27@sitem_jwep&19@精炼潜质:[精炼+1] 每升一级 物理攻击+2&103@0&104@0&105@&107@&108@127|1|1|铜钱|163.png|500
6|八极法杖|equip|46.png|f_i_八极法杖&23@凿孔上限 9 格&24@武器·法杖系&25@1&21@20&22@术士&2@48&16@2&27@sitem_jwep&19@精炼潜质:[精炼+1] 每升一级 魔法攻击+2&103@0&104@0&105@&107@&108@129|1|1|铜钱|163.png|500
7|铁块|null|107.png|f_i_铁块&24@材料&25@99&20@炼制铁器必备的材料.&103@0&104@0&105@&107@&108@10|1|1|铜钱|163.png|10|碎铁矿|105.png|10
8|铜块|null|108.png|f_i_铜块&24@材料&25@99&20@炼制铜器必备的材料.&103@0&104@0&105@&107@&108@10|1|2|铜钱|163.png|10|铜钱|163.png|1000
9|银锭|null|109.png|f_i_银锭&24@材料&25@99&20@炼制银器必备的材料.&103@0&104@0&105@&107@&108@10|1|2|铜钱|163.png|10|银元宝|39.png|10
`

const sourceYunyinArmorShopRows = `
0|布帽|equip|293.png|f_i_布帽^ffffff&23@凿孔上限 9 格&24@护具·头部&25@1&21@1&3@3&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2&103@0&104@0&105@&107@&108@0|1|1|铜钱|163.png|20
1|蓝布衣|equip|291.png|f_i_蓝布衣^ffffff&23@凿孔上限 9 格&24@护具·躯干&25@1&21@1&3@<$pdef><$jpdef>&27@sitem_ezhj&21@1&3@6&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2\n&103@0&104@0&105@&107@&108@0|1|1|铜钱|163.png|30
2|布护腕|equip|294.png|f_i_布护腕&23@凿孔上限 9 格&24@护具·护腕&25@1&21@1&3@2&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2&103@0&104@0&105@&107@&108@0|1|1|铜钱|163.png|15
3|蓝布裤|equip|3.png|f_i_蓝布裤^ffffff&23@凿孔上限 9 格&24@护具·腿&25@1&21@1&3@4&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2&103@0&104@0&105@&107@&108@0|1|1|铜钱|163.png|25
4|布护腰|equip|292.png|f_i_布护腰^ffffff&23@凿孔上限 9 格&24@护具·腰部&25@1&21@1&3@3&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2&103@0&104@0&105@&107@&108@0|1|1|铜钱|163.png|20
5|布鞋|equip|274.png|f_i_布鞋^ffffff&23@凿孔上限 9 格&24@护具·足部&25@1&21@1&3@2&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2&103@0&104@0&105@&107@&108@0|1|1|铜钱|163.png|20
6|飞佐面甲|equip|309.png|f_i_飞佐面甲^ffffff&23@凿孔上限 9 格&24@护具·头部&25@1&21@10&22@战士&3@12&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2&103@0&104@0&105@&107@&108@25|1|1|铜钱|163.png|150
7|飞佐护甲|equip|311.png|f_i_飞佐护甲^ffffff&23@凿孔上限 9 格&24@护具·躯干&25@1&21@10&22@战士&3@16&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2&103@0&104@0&105@&107@&108@30|1|1|铜钱|163.png|200
8|飞佐护腰|equip|314.png|f_i_飞佐护腰&23@凿孔上限 9 格&24@护具·腰部&25@1&21@10&22@战士&3@8&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2&103@0&104@0&105@&107@&108@20|1|1|铜钱|163.png|150
9|飞佐护腿|equip|313.png|f_i_飞佐护腿^ffffff&23@凿孔上限 9 格&24@护具·腿&25@1&21@10&22@战士&3@13&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2&103@0&104@0&105@&107@&108@20|1|1|铜钱|163.png|200
10|飞佐战靴|equip|315.png|f_i_飞佐战靴^ffffff&23@凿孔上限 9 格&24@护具·足部&25@1&21@10&22@战士&3@10&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2&103@0&104@0&105@&107@&108@20|1|1|铜钱|163.png|150
11|飞佐护肩|equip|310.png|f_i_飞佐护肩^ffffff&23@凿孔上限 9 格&24@护具·肩部&25@1&21@10&22@战士&3@9&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2&103@0&104@0&105@&107@&108@20|1|1|铜钱|163.png|150
12|飞佐护腕|equip|312.png|f_i_飞佐护腕^ffffff&23@凿孔上限 9 格&24@护具·护腕&25@1&21@10&22@战士&3@9&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2&103@0&104@0&105@&107@&108@20|1|1|铜钱|163.png|150
13|圣羽面甲|equip|316.png|f_i_圣羽面甲^ffffff&23@凿孔上限 9 格&24@护具·头部&25@1&21@10&22@术士&3@8&4@8&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2\n[精炼+1] 每升一级 魔法防御+2&103@0&104@0&105@&107@&108@25|1|1|铜钱|163.png|150
14|圣羽战衣|equip|318.png|f_i_圣羽战衣^ffffff&23@凿孔上限 9 格&24@护具·躯干&25@1&21@10&22@术士&3@14&4@4&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2\n[精炼+1] 每升一级 魔法防御+2&103@0&104@0&105@&107@&108@30|1|1|铜钱|163.png|200
15|圣羽护腰|equip|322.png|f_i_圣羽护腰^ffffff&23@凿孔上限 9 格&24@护具·腰部&25@1&21@10&22@术士&3@7&4@1&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2\n[精炼+1] 每升一级 魔法防御+2&103@0&104@0&105@&107@&108@20|1|1|铜钱|163.png|150
16|圣羽护腿|equip|321.png|f_i_圣羽护腿^ffffff&23@凿孔上限 9 格&24@护具·腿&25@1&21@10&22@术士&3@10&4@3&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2\n[精炼+1] 每升一级 魔法防御+2&103@0&104@0&105@&107@&108@20|1|1|铜钱|163.png|200
17|圣羽战靴|equip|323.png|f_i_圣羽战靴^ffffff&23@凿孔上限 9 格&24@护具·足部&25@1&21@10&22@术士&3@9&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2&103@0&104@0&105@&107@&108@20|1|1|铜钱|163.png|150
18|圣羽护肩|equip|317.png|f_i_圣羽护肩^ffffff&23@凿孔上限 9 格&24@护具·肩部&25@1&21@10&22@术士&3@8&4@2&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2\n[精炼+1] 每升一级 魔法防御+2&103@0&104@0&105@&107@&108@20|1|1|铜钱|163.png|150
19|圣羽护腕|equip|319.png|f_i_圣羽护腕^ffffff&23@凿孔上限 9 格&24@护具·护腕&25@1&21@10&22@术士&3@9&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2&103@0&104@0&105@&107@&108@20|1|1|铜钱|163.png|150
20|飞艳面甲|equip|325.png|f_i_飞艳面甲^ffffff&23@凿孔上限 9 格&24@护具·头部&25@1&21@10&22@游侠&3@10&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2&103@0&104@0&105@&107@&108@25|1|1|铜钱|163.png|150
21|飞艳战衣|equip|327.png|f_i_飞艳战衣^ffffff&23@凿孔上限 9 格&24@护具·躯干&25@1&21@10&22@游侠&3@15&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2&103@0&104@0&105@&107@&108@30|1|1|铜钱|163.png|200
22|飞艳护腰|equip|330.png|f_i_飞艳护腰^ffffff&23@凿孔上限 9 格&24@护具·腰部&25@1&21@10&22@游侠&3@8&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2&103@0&104@0&105@&107@&108@20|1|1|铜钱|163.png|150
23|飞艳护腿|equip|329.png|f_i_飞艳护腿^ffffff&23@凿孔上限 9 格&24@护具·腿&25@1&21@10&22@游侠&3@12&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2&103@0&104@0&105@&107@&108@20|1|1|铜钱|163.png|200
24|飞艳战靴|equip|331.png|f_i_飞艳战靴^ffffff&23@凿孔上限 9 格&24@护具·足部&25@1&21@10&22@游侠&3@10&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2&103@0&104@0&105@&107@&108@20|1|1|铜钱|163.png|150
25|飞艳护肩|equip|326.png|f_i_飞艳护肩^ffffff&23@凿孔上限 9 格&24@护具·肩部&25@1&21@10&22@游侠&3@9&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2&103@0&104@0&105@&107@&108@20|1|1|铜钱|163.png|150
26|飞艳护腕|equip|328.png|f_i_飞艳护腕^ffffff&23@凿孔上限 9 格&24@护具·护腕&25@1&21@10&22@游侠&3@9&27@sitem_ezhj&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+2&103@0&104@0&105@&107@&108@20|1|1|铜钱|163.png|150
`

const sourceGuangqingGroceryShopRows = `
0|普通采集手套|null|856.png|f_i_普通采集手套^ffffff&24@消耗品&25@999&20@平常的采集手套，带上进行采集得话可以很好的保护双手。&103@0&104@0&105@&107@&108@0|1|1|铜钱|163.png|8
1|宠物用营养水|null|210.png|f_i_宠物用营养水^ffffff&23@每次喂食 成长 +5 饱食 +10&24@消耗品&25@999&19@培养宠物的营养水。&20@一般在道具商人处可以购得。&27@sitem_water&103@0&104@0&105@&107@&108@0|1|1|铜钱|163.png|8
2|回城咒|own|193.png|f_i_回城咒^ffffff&24@消耗品&25@99&19@双击后可传送回城镇&20@拥有神秘传送力量的咒符。&27@sitem_scroll&103@0&104@0&105@&107@&108@0|1|1|铜钱|163.png|500
3|毒药|null|240.png|f_i_毒药^ffffff&24@消耗品&25@999&20@涂在武器上的毒药。&103@0&104@0&105@&107@&108@0|10|1|铜钱|163.png|20
4|飞镖|null|241.png|f_i_飞镖^ffffff&24@消耗品&25@999&20@暗器系武器需要消耗的弹药。&103@0&104@0&105@&107@&108@0|50|1|铜钱|163.png|50
5|火之箭|null|242.png|f_i_火之箭^ffffff&24@消耗品&25@999&20@弓系武器需要消耗的箭矢。&103@0&104@0&105@&107@&108@0|50|1|铜钱|163.png|50
6|冰之箭|null|243.png|f_i_冰之箭^ffffff&24@消耗品&25@999&20@弓系武器需要消耗的箭矢。&103@0&104@0&105@&107@&108@0|50|1|铜钱|163.png|50
7|穿甲箭|null|246.png|f_i_穿甲箭^ffffff&24@消耗品&25@999&20@弓系武器需要消耗的箭矢。&103@0&104@0&105@&107@&108@0|50|2|铜钱|163.png|80
8|暗之箭|null|245.png|f_i_暗之箭^ffffff&24@消耗品&25@999&20@弓系武器需要消耗的箭矢。&103@0&104@0&105@&107@&108@0|50|1|铜钱|163.png|80
9|魔箭|null|244.png|f_i_魔箭^ffffff&24@消耗品&25@999&20@弓系武器需要消耗的箭矢。&103@0&104@0&105@&107@&108@0|50|1|铜钱|163.png|100
10|毒箭|null|247.png|f_i_毒箭^ffffff&24@消耗品&25@999&20@弓系武器需要消耗的箭矢。&103@0&104@0&105@&107@&108@0|10|2|铜钱|163.png|100
`

const sourceYunyinGroceryShopRows = sourceGuangqingGroceryShopRows + `
11|飞仙洞通行证|null|782.png|f_i_飞仙洞通行证^00ccff&24@消耗品&25@99&19@<font color='#00ff00'>进入飞仙洞修炼的通行证.</font>&27@sitem_book&103@0&104@0&105@&107@&108@150|1|3|铜钱|163.png|150
`

const sourceDafoGroceryShopRows = sourceGuangqingGroceryShopRows + `
11|黄风寨通行证|null|783.png|f_i_黄风寨通行证^00ccff&24@消耗品&25@99&19@<font color='#00ff00'>进入黄风寨修炼的通行证.</font>&27@sitem_book&103@0&104@0&105@&107@&108@150|1|3|铜钱|163.png|150
`

const sourceJiantingGroceryShopRows = sourceGuangqingGroceryShopRows + `
11|水帘洞通行证|null|781.png|f_i_水帘洞通行证^00ccff&24@消耗品&25@99&19@<font color='#00ff00'>进入水帘洞修炼的通行证.</font>&27@sitem_book&103@0&104@0&105@&107@&108@150|1|3|铜钱|163.png|150
`

const sourceGuangqingArmorShopRows = `
0|蛮力面甲|equip|334.png|f_i_蛮力面甲^ffffff&23@凿孔上限 9 格&24@护具·头部&25@1&21@20&22@战士&3@17&5@30&27@sitem_piput&19@精炼潜质:[精炼+1] 每升一级 物理防御+2&103@0&104@0&105@&107@&108@100|1|1|铜钱|163.png|400
1|蛮力护甲|equip|336.png|f_i_蛮力护甲^ffffff&24@护具·身体&25@1&21@20&22@战士&3@20&5@35&27@sitem_piput&103@0&104@0&105@&107@&108@100|1|1|铜钱|163.png|500
2|蛮力护腰|equip|339.png|f_i_蛮力护腰^ffffff&24@护具·腰部&25@1&21@20&22@战士&3@17&5@30&27@sitem_piput&103@0&104@0&105@&107@&108@100|1|1|铜钱|163.png|400
3|蛮力护腿|equip|338.png|f_i_蛮力护腿^ffffff&24@护具·腿部&25@1&21@20&22@战士&3@20&5@35&27@sitem_piput&103@0&104@0&105@&107@&108@100|1|1|铜钱|163.png|500
4|蛮力战靴|equip|340.png|f_i_蛮力战靴^ffffff&24@护具·脚部&25@1&21@20&22@战士&3@17&5@30&27@sitem_piput&103@0&104@0&105@&107@&108@100|1|1|铜钱|163.png|400
5|蛮力肩甲|equip|335.png|f_i_蛮力肩甲^ffffff&24@护具·肩部&25@1&21@20&22@战士&3@17&5@30&27@sitem_piput&103@0&104@0&105@&107@&108@100|1|1|铜钱|163.png|400
6|蛮力护腕|equip|337.png|f_i_蛮力护腕^ffffff&24@护具·腕部&25@1&21@20&22@战士&3@17&5@30&27@sitem_piput&103@0&104@0&105@&107@&108@100|1|1|铜钱|163.png|400
7|八极法冠|equip|350.png|f_i_八极法冠^ffffff&24@护具·头部&25@1&21@20&22@术士&4@17&6@30&27@sitem_piput&103@0&104@0&105@&107@&108@100|1|1|铜钱|163.png|400
8|八极法衣|equip|352.png|f_i_八极法衣^ffffff&24@护具·身体&25@1&21@20&22@术士&4@20&6@35&27@sitem_piput&103@0&104@0&105@&107@&108@100|1|1|铜钱|163.png|500
9|八极护腰|equip|355.png|f_i_八极护腰^ffffff&24@护具·腰部&25@1&21@20&22@术士&4@17&6@30&27@sitem_piput&103@0&104@0&105@&107@&108@100|1|1|铜钱|163.png|400
10|八极护腿|equip|354.png|f_i_八极护腿^ffffff&24@护具·腿部&25@1&21@20&22@术士&4@20&6@35&27@sitem_piput&103@0&104@0&105@&107@&108@100|1|1|铜钱|163.png|500
11|八极法靴|equip|356.png|f_i_八极法靴^ffffff&24@护具·脚部&25@1&21@20&22@术士&4@17&6@30&27@sitem_piput&103@0&104@0&105@&107@&108@100|1|1|铜钱|163.png|400
12|八极护肩|equip|351.png|f_i_八极护肩^ffffff&24@护具·肩部&25@1&21@20&22@术士&4@17&6@30&27@sitem_piput&103@0&104@0&105@&107@&108@100|1|1|铜钱|163.png|400
13|八极护腕|equip|353.png|f_i_八极护腕^ffffff&24@护具·腕部&25@1&21@20&22@术士&4@17&6@30&27@sitem_piput&103@0&104@0&105@&107@&108@100|1|1|铜钱|163.png|400
14|威武面甲|equip|341.png|f_i_威武面甲^ffffff&24@护具·头部&25@1&21@20&22@游侠&3@17&5@30&27@sitem_piput&103@0&104@0&105@&107@&108@100|1|1|铜钱|163.png|400
15|威武皮甲|equip|343.png|f_i_威武皮甲^ffffff&24@护具·身体&25@1&21@20&22@游侠&3@20&5@35&27@sitem_piput&103@0&104@0&105@&107@&108@100|1|1|铜钱|163.png|500
16|威武腰带|equip|347.png|f_i_威武腰带^ffffff&24@护具·腰部&25@1&21@20&22@游侠&3@17&5@30&27@sitem_piput&103@0&104@0&105@&107@&108@100|1|1|铜钱|163.png|400
17|威武护腿|equip|346.png|f_i_威武护腿^ffffff&24@护具·腿部&25@1&21@20&22@游侠&3@20&5@35&27@sitem_piput&103@0&104@0&105@&107@&108@100|1|1|铜钱|163.png|500
18|威武皮靴|equip|348.png|f_i_威武皮靴^ffffff&24@护具·脚部&25@1&21@20&22@游侠&3@17&5@30&27@sitem_piput&103@0&104@0&105@&107@&108@100|1|1|铜钱|163.png|400
19|威武护肩|equip|342.png|f_i_威武护肩^ffffff&24@护具·肩部&25@1&21@20&22@游侠&3@17&5@30&27@sitem_piput&103@0&104@0&105@&107@&108@100|1|1|铜钱|163.png|400
20|威武护腕|equip|344.png|f_i_威武护腕^ffffff&24@护具·腕部&25@1&21@20&22@游侠&3@17&5@30&27@sitem_piput&103@0&104@0&105@&107@&108@100|1|1|铜钱|163.png|400
21|麻布|null|78.png|f_i_麻布&24@材料&25@99&20@最普通的布料.&103@0&104@0&105@&107@&108@9|1|1|铜钱|163.png|10
22|红缨|null|77.png|f_i_红缨&24@材料&25@99&20@红色的丝缨.&103@0&104@0&105@&107@&108@0|1|1|铜钱|163.png|10|丝|76.png|5|兽血|73.png|1
23|绸缎|null|79.png|f_i_绸缎&24@材料&25@99&20@柔滑的高级布料.&103@0&104@0&105@&107@&108@0|1|1|铜钱|163.png|10|丝|76.png|10
24|皮革|null|80.png|f_i_皮革&24@材料&25@99&20@常用的制甲材料.&103@0&104@0&105@&107@&108@0|1|1|铜钱|163.png|10|毛皮|71.png|2
25|精制皮革|null|81.png|f_i_精制皮革&24@材料&25@99&20@精制后的皮革.&103@0&104@0&105@&107@&108@0|1|2|铜钱|163.png|30|皮革|80.png|5
`

const sourceGuangqingHealerShopRows = `
0|馒头|own|0.png|f_i_馒头&24@消耗品&25@99&7@200&20@又白又香的馒头&0;饥饿的时候用来充饥.&103@0&104@0&105@&107@&108@0|1|1|铜钱|163.png|10
1|花卷|own|213.png|f_i_花卷&24@消耗品&25@99&7@500&20@又白又香的花卷&0;饥饿的时候用来充饥.&103@0&104@0&105@&107@&108@0|1|1|铜钱|163.png|50
2|包子|own|212.png|f_i_包子&24@消耗品&25@99&7@600&20@带馅的包子&0;看起来非常可口&0;食用后可恢复些气力.&103@0&104@0&105@&107@&108@0|1|1|铜钱|163.png|120
3|小包还元散|own|695.png|f_i_小包还元散&24@消耗品&25@99&7@1500&20@恢复气力的药散.&103@0&104@0&105@&107@&108@0|1|1|铜钱|163.png|350
4|甘露|own|214.png|f_i_甘露&24@消耗品&25@99&8@50&20@恢复精力的甘露.&103@0&104@0&105@&107@&108@0|1|1|铜钱|163.png|10
5|小瓶甘露|own|696.png|f_i_小瓶甘露&24@消耗品&25@99&8@100&20@恢复精力的甘露.&103@0&104@0&105@&107@&108@0|1|1|铜钱|163.png|25
6|中瓶甘露|own|697.png|f_i_中瓶甘露&24@消耗品&25@99&8@200&20@恢复精力的甘露.&103@0&104@0&105@&107@&108@0|1|2|铜钱|163.png|60
7|大瓶甘露|own|698.png|f_i_大瓶甘露&24@消耗品&25@99&8@400&20@恢复精力的甘露.&103@0&104@0&105@&107@&108@0|1|3|铜钱|163.png|150
8|解毒丸|own|218.png|f_i_解毒丸&24@消耗品&25@99&20@解除中毒状态的药丸.&103@0&104@0&105@&107@&108@0|1|1|铜钱|163.png|10
`

const sourceGuangqingCraftShopRows = `
0|狰狞神骑|equip|130.png|f_i_狰狞神骑^f9e000&23@限制装备至【坐骑】格。&24@坐骑&25@1&21@90&13@50&14@50&15@50&16@50&17@50&27@sitem_pet&19@精炼潜质:[精炼+16] 每升一级 物理攻击+160[精炼+16] 每升一级 魔法攻击+160[精炼+16] 每升一级 移动+10[精炼+18] 每升一级 魔法防御+300[精炼+18] 每升一级 物理防御+300[精炼+20] 每升一级 命中+20%[精炼+20] 每升一级 回避+20%[精炼+20] 每升一级 气力上限+1500&103@0&104@0&105@&107@&108@0|1|5|狰狞的头|130.png|99|狰狞的皮|128.png|99|狰狞的尾|131.png|99|狰狞的爪|129.png|99|狰狞精魄|1661.png|100
1|精炼宝石|null|435.png|f_i_精炼宝石^f9e000&24@材料&25@999&20@装备精炼所需的宝石.&103@0&104@0&105@&107@&108@0|1|5|造物魔晶|857.png|13
2|宠物锦囊|null|141.png|f_i_宠物锦囊^f9e000&24@宝物&25@99&20@神秘的宠物锦囊。&103@0&104@0&105@&107@&108@0|1|4|造物魔晶|857.png|60
3|筋斗云|equip|968.png|f_i_筋斗云^f9e000&24@坐骑&25@1&20@传说中的筋斗云。&103@0&104@0&105@&107@&108@0|1|3|造物魔晶|857.png|100
4|吉祥袋|null|959.png|f_i_吉祥袋^f9e000&24@宝物&25@99&20@打开可获得珍贵材料。&103@0&104@0&105@&107@&108@0|1|5|吉祥布料|960.png|5
5|如意袋|null|966.png|f_i_如意袋^f9e000&24@宝物&25@99&20@打开可获得珍贵材料。&103@0&104@0&105@&107@&108@0|1|5|如意布料|961.png|5
6|乾坤袋|null|967.png|f_i_乾坤袋^f9e000&24@宝物&25@99&20@打开可获得珍贵材料。&103@0&104@0&105@&107@&108@0|1|5|乾坤布料|962.png|5
7|奥义秘诀|own|2.png|f_i_奥义秘诀^f9e000&24@消耗品&25@99&20@记载奥义的秘诀。&103@0&104@0&105@&107@&108@0|1|4|造物魔晶|857.png|20
8|神技天书|own|1257.png|f_i_神技天书^f9e000&24@消耗品&25@99&20@记载神技的天书。&103@0&104@0&105@&107@&108@0|1|4|造物魔晶|857.png|50
9|盖世神功|own|1230.png|f_i_盖世神功^f9e000&24@消耗品&25@99&20@记载盖世神功的秘籍。&103@0&104@0&105@&107@&108@0|1|5|造物魔晶|857.png|150
`
