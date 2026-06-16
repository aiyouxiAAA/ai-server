package main

import (
	"strconv"
	"strings"

	"ai-server/internal/session"
)

const (
	sourceSkillTeacherHandle    = "1000542608713897"
	guangqingSkillTeacherHandle = "2220542612946566"
	classicBattleLootType       = "战斗"
	classicBattleLootCap        = 18
)

type classicTownRoleInteractionRequest struct {
	Handle string `json:"handle"`
	RoleID string `json:"roleId"`
	Kind   string `json:"kind"`
	MapID  string `json:"mapId"`
}

type classicTownAnswerRequest struct {
	Handle       string `json:"handle"`
	MsgHandle    string `json:"msgHandle"`
	AnswerHandle string `json:"answerHandle"`
}

type classicTownTransferRequest struct {
	MapID string `json:"mapId"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
}

type classicTownMoveRoleRequest struct {
	Handle string `json:"handle"`
	Type   string `json:"type"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Z      int    `json:"z"`
	TX     int    `json:"tx"`
	TY     int    `json:"ty"`
	TZ     int    `json:"tz"`
	MapID  string `json:"mapId"`
}

type classicTownBuySkillRequest struct {
	ShopID  string `json:"shopId"`
	SkillID int    `json:"skillId"`
}

type classicTownContainerRequest struct {
	Type string `json:"type"`
}

type classicTownContainerMoveRequest struct {
	SourceType  string   `json:"sourceType"`
	TargetType  string   `json:"targetType"`
	SourceIndex *int     `json:"sourceIndex,omitempty"`
	TargetIndex *int     `json:"targetIndex,omitempty"`
	Count       *int     `json:"count,omitempty"`
	Names       []string `json:"names,omitempty"`
}

type classicTownEquipItemRequest struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
	Count int    `json:"count"`
}

type classicTownActiveItemRequest struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
}

type classicTownDestroyItemRequest struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
	Count int    `json:"count"`
}

type classicTownSaleItemRequest struct {
	ShopID string `json:"shopId"`
	Type   string `json:"type"`
	Index  int    `json:"index"`
	Count  int    `json:"count"`
}

type classicTownChatSendRequest struct {
	Channel    string `json:"channel"`
	Msg        string `json:"msg"`
	TargetName string `json:"targetName,omitempty"`
}

type classicTownChatMessagePush struct {
	Channel    string `json:"channel"`
	Handle     string `json:"handle,omitempty"`
	Name       string `json:"name,omitempty"`
	TargetName string `json:"targetName,omitempty"`
	Msg        string `json:"msg"`
	VIP        int    `json:"vip,omitempty"`
	Outgoing   bool   `json:"outgoing,omitempty"`
	Color      string `json:"color,omitempty"`
	Bold       bool   `json:"bold,omitempty"`
}

type classicTownChatBroadcast struct {
	Recipients []string
	Message    classicTownChatMessagePush
}

type classicTownSkillInfoPush struct {
	Handle      string `json:"handle"`
	Name        string `json:"name"`
	Level       int    `json:"level"`
	Type        string `json:"type"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
}

type classicTownClearSkillInfoPush struct {
	Handle string `json:"handle"`
	Name   string `json:"name"`
	Level  int    `json:"level,omitempty"`
}

type classicTownRemoveSkillRequest struct {
	Name string `json:"name"`
}

type classicTownSkillCapPush struct {
	Count int `json:"count"`
}

type classicTownFastPanelEntry struct {
	Index int    `json:"index"`
	Type  string `json:"type"`
	Name  string `json:"name"`
}

type classicTownSetFastPanelRequest struct {
	Index int    `json:"index"`
	Type  string `json:"type"`
	Name  string `json:"name"`
}

type classicTownFastPanelPush struct {
	Entries []classicTownFastPanelEntry `json:"entries"`
}

type classicTownCurrencyPush struct {
	Handle     string                 `json:"handle"`
	Currencies session.RoleCurrencies `json:"currencies"`
}

type classicTownBuySkillResultPush struct {
	Success      bool                   `json:"success"`
	ShopID       string                 `json:"shopId"`
	SkillID      int                    `json:"skillId"`
	Currencies   session.RoleCurrencies `json:"currencies,omitempty"`
	ErrorCode    string                 `json:"errorCode,omitempty"`
	ErrorMessage string                 `json:"errorMessage,omitempty"`
}

type classicTownContainerCapacityPush struct {
	Handle   string `json:"handle"`
	Type     string `json:"type"`
	Capacity int    `json:"capacity"`
	OpenType string `json:"openType,omitempty"`
}

type classicTownItemInfoPush struct {
	Handle      string `json:"handle"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	ItemType    string `json:"itemType"`
	Display     string `json:"display"`
	Description string `json:"description"`
	Count       int    `json:"count"`
	Index       int    `json:"index"`
	Level       int    `json:"level"`
	EndTime     int    `json:"endTime"`
	Owner       string `json:"owner"`
	ItemLevel   int    `json:"itemLevel"`
}

type classicTownItemInfoClearPush struct {
	Handle string `json:"handle"`
	Type   string `json:"type"`
	Index  int    `json:"index"`
}

type classicTownSkillShopPush struct {
	Handle   string                      `json:"handle"`
	ShopID   string                      `json:"shopId"`
	Title    string                      `json:"title"`
	Vocation string                      `json:"vocation"`
	SkillCap int                         `json:"skillCap"`
	Skills   []classicTownSkillShopEntry `json:"skills"`
}

type classicTownSkillShopEntry struct {
	ID           int                               `json:"id"`
	Name         string                            `json:"name"`
	Owned        bool                              `json:"owned"`
	Icon         string                            `json:"icon"`
	Category     string                            `json:"category"`
	Vocation     string                            `json:"vocation"`
	TargetType   string                            `json:"targetType"`
	Description  string                            `json:"description"`
	Requirements []classicTownSkillShopRequirement `json:"requirements"`
	MaxLevel     int                               `json:"maxLevel,omitempty"`
}

type classicTownSkillShopRequirement struct {
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Count int    `json:"count"`
}

func buildClassicTownSkillShopResult(store *session.Store, socketSession *packetSession, sourceHandle string, answerHandle string) (packetResult, bool) {
	shop, ok := sourceSkillTeacherShops[answerHandle]
	if !ok {
		return packetResult{}, false
	}
	shop = cloneSourceSkillShop(shop)
	shop.Handle = sourceHandle
	if socketSession != nil && socketSession.selectedRole != nil && socketSession.playerBase != nil {
		skills, _, found := store.GetRoleSkills(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID)
		if found {
			applyOwnedSkillsToShop(&shop, skills)
		}
	}

	return packetResult{
		skillShop: &shop,
		handled:   true,
	}, true
}

func buildClassicTownFastPanelResult(store *session.Store, socketSession *packetSession) packetResult {
	if store == nil || socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		return packetResult{handled: true}
	}

	entries, ok := store.GetRoleFastPanel(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID)
	if !ok {
		return packetResult{handled: true}
	}

	return classicTownFastPanelPacketResult(entries)
}

func buildClassicTownSetFastPanelResult(store *session.Store, socketSession *packetSession, request classicTownSetFastPanelRequest) packetResult {
	if store == nil || socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		return packetResult{handled: true}
	}

	entries, ok := store.SetRoleFastPanelEntry(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, session.RoleFastPanelEntry{
		Index: request.Index,
		Type:  strings.TrimSpace(request.Type),
		Name:  strings.TrimSpace(request.Name),
	})
	if !ok {
		return packetResult{handled: true}
	}

	return classicTownFastPanelPacketResult(entries)
}

func classicTownFastPanelPacketResult(entries []session.RoleFastPanelEntry) packetResult {
	pushEntries := make([]classicTownFastPanelEntry, 0, len(entries))
	for _, entry := range entries {
		pushEntries = append(pushEntries, classicTownFastPanelEntry{
			Index: entry.Index,
			Type:  entry.Type,
			Name:  entry.Name,
		})
	}
	return packetResult{
		fastPanel: &classicTownFastPanelPush{
			Entries: pushEntries,
		},
		handled: true,
	}
}

var sourceSkillTeacherShops = map[string]classicTownSkillShopPush{
	"7": {
		Handle:   sourceSkillTeacherHandle,
		ShopID:   "skill1",
		Title:    "战士技能",
		Vocation: "战士",
		SkillCap: 22,
		Skills:   sourceSkillShopEntries("skill1", "战士", sourceWarriorSkillShopRows),
	},
	"8": {
		Handle:   sourceSkillTeacherHandle,
		ShopID:   "skill2",
		Title:    "术士技能",
		Vocation: "术士",
		SkillCap: 24,
		Skills:   sourceSkillShopEntries("skill2", "术士", sourceMageSkillShopRows),
	},
	"9": {
		Handle:   sourceSkillTeacherHandle,
		ShopID:   "skill3",
		Title:    "游侠技能",
		Vocation: "游侠",
		SkillCap: 26,
		Skills:   sourceSkillShopEntries("skill3", "游侠", sourceRangerSkillShopRows),
	},
}

func sourceSkillShopEntries(shopID string, vocation string, rows string) []classicTownSkillShopEntry {
	lines := strings.Split(strings.TrimSpace(rows), "\n")
	entries := make([]classicTownSkillShopEntry, 0, len(lines))
	for _, line := range lines {
		columns := strings.Split(line, "|")
		if len(columns) != 6 {
			continue
		}
		id, err := strconv.Atoi(columns[0])
		if err != nil {
			continue
		}
		entries = append(entries, classicTownSkillShopEntry{
			ID:           id,
			Name:         columns[1],
			Owned:        false,
			Icon:         columns[2],
			Category:     columns[3],
			Vocation:     vocation,
			TargetType:   columns[4],
			Description:  columns[5],
			Requirements: sourceSkillShopRequirements(shopID, id),
			MaxLevel:     sourceSkillShopMaxLevel(shopID, id),
		})
	}
	return entries
}

func sourceSkillShopRequirements(shopID string, skillID int) []classicTownSkillShopRequirement {
	requirements := sourceSkillShopRequirementByKey[sourceSkillShopKey(shopID, skillID)]
	return append([]classicTownSkillShopRequirement(nil), requirements...)
}

func sourceSkillShopMaxLevel(shopID string, skillID int) int {
	if sourceSkillShopSingleLevelByKey[sourceSkillShopKey(shopID, skillID)] {
		return 1
	}
	return 5
}

func sourceSkillShopKey(shopID string, skillID int) string {
	return shopID + ":" + strconv.Itoa(skillID)
}

func parseSourceSkillShopRequirementRows(rows string) map[string][]classicTownSkillShopRequirement {
	result := map[string][]classicTownSkillShopRequirement{}
	for _, line := range strings.Split(strings.TrimSpace(rows), "\n") {
		columns := strings.Split(line, "|")
		if len(columns) != 5 {
			continue
		}
		skillID, err := strconv.Atoi(columns[1])
		if err != nil {
			continue
		}
		count, err := strconv.Atoi(columns[4])
		if err != nil {
			continue
		}
		key := sourceSkillShopKey(columns[0], skillID)
		result[key] = append(result[key], classicTownSkillShopRequirement{
			Name:  columns[2],
			Icon:  columns[3],
			Count: count,
		})
	}
	return result
}

func cloneSourceSkillShop(shop classicTownSkillShopPush) classicTownSkillShopPush {
	shop.Skills = append([]classicTownSkillShopEntry(nil), shop.Skills...)
	for index := range shop.Skills {
		shop.Skills[index].Requirements = append([]classicTownSkillShopRequirement(nil), shop.Skills[index].Requirements...)
	}
	return shop
}

func applyOwnedSkillsToShop(shop *classicTownSkillShopPush, skills []session.RoleSkill) {
	owned := make(map[string]struct{}, len(skills))
	for _, skill := range skills {
		owned[skill.Name] = struct{}{}
	}
	for index := range shop.Skills {
		_, shop.Skills[index].Owned = owned[shop.Skills[index].Name]
	}
}

func findSourceSkillShopEntry(shopID string, skillID int) (classicTownSkillShopEntry, bool) {
	for _, shop := range sourceSkillTeacherShops {
		if shop.ShopID != shopID {
			continue
		}
		for _, entry := range shop.Skills {
			if entry.ID == skillID {
				return entry, true
			}
		}
	}
	return classicTownSkillShopEntry{}, false
}

func sourceSkillEntryToRoleSkill(entry classicTownSkillShopEntry) session.RoleSkill {
	return session.RoleSkill{
		Name:        entry.Name,
		Level:       1,
		Type:        entry.Category,
		Icon:        entry.Icon,
		Description: sourceRoleSkillDescription(entry.Name, 1, entry.Description),
		MaxLevel:    entry.MaxLevel,
	}
}

func sourceSkillEntryToRoleItem(entry classicTownSkillShopEntry) session.RoleItem {
	return session.RoleItem{
		Type:        "背包",
		Name:        entry.Name,
		ItemType:    entry.Category,
		Display:     entry.Icon,
		Description: sourceRoleSkillDescription(entry.Name, 1, entry.Description),
		Count:       1,
		Index:       -1,
		ItemLevel:   entry.MaxLevel,
	}
}

func sourceRoleSkillDescription(name string, level int, fallback string) string {
	if level <= 0 {
		level = 1
	}
	switch strings.TrimSpace(name) {
	case "普通攻击":
		return "f_s_普通攻击^ffffff&9@单体·攻击&10@通用&22@战斗&5@给予对手普通的物理攻击."
	case "密斩":
		return "f_s_密斩&9@单体·攻击&7@3&10@单刀/单斧&22@战斗&2@5&4@提升40%的物理伤害"
	case "多段斩":
		switch level {
		case 2:
			return "f_s_多段斩^ffffff&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@10&4@提升60%的物理伤害"
		case 3:
			return "f_s_多段斩^ffffff&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@12&4@提升65%的物理伤害"
		case 4:
			return "f_s_多段斩^ffffff&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@14&4@提升70%的物理伤害"
		case 5:
			return "f_s_多段斩^ffffff&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@16&4@提升75%的物理伤害"
		default:
			return "f_s_多段斩^ffffff&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@8&4@提升55%的物理伤害"
		}
	case "嗜血斩":
		switch level {
		case 2:
			return "f_s_嗜血斩^5BC46D&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@26&4@对敌人造成94%的物理伤害&0;并有84%机率将对敌人造成伤害的70%转换为气力</font>"
		case 3:
			return "f_s_嗜血斩^5BC46D&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@28&4@对敌人造成96%的物理伤害&0;并有86%机率将对敌人造成伤害的70%转换为气力</font>"
		default:
			return "f_s_嗜血斩^5BC46D&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@24&4@对敌人造成92%的物理伤害&0;并有82%机率将对敌人造成伤害的70%转换为气力</font>"
		}
	case "狂爆":
		return "f_s_狂爆^5BC46D&9@单体·状态&8@战士 &10@单刀&22@战斗&2@15&4@3回合内物理攻击力翻倍&0;并降低100%的物理防御"
	case "红月斩":
		return "f_s_红月斩^ffffff&9@群体·攻击&8@战士 &10@单刀&22@战斗&2@40&4@对所有敌人造成72%的物理伤害"
	case "血切":
		return "f_s_血切^5BC46D&9@单体·状态&8@战士 &10@单刀&22@战斗&2@19&4@对敌人造成30%的物理伤害&0;击中敌人时有80%的机率使对方进入外伤状态4回合<br>(每回合损失气力为角色物理攻击的25%~30%)"
	case "奥义.雷魂斩":
		return "f_s_奥义.雷魂斩^00ccff&9@单体·攻击&8@战士 &10@单刀&22@战斗&2@24&4@<font color='#00cc00'>特殊发动条件:需要3格魂元</font><br>提升240%的物理伤害"
	default:
		return fallback
	}
}

func sourceSkillRequirementsToCurrencies(requirements []classicTownSkillShopRequirement) session.RoleCurrencies {
	currencies := session.RoleCurrencies{}
	for _, requirement := range requirements {
		currencies[requirement.Name] += requirement.Count
	}
	return currencies
}

func sourceSkillRequirementsToRoleItemRequirements(requirements []classicTownSkillShopRequirement) []session.RoleItemRequirement {
	result := make([]session.RoleItemRequirement, 0, len(requirements))
	for _, requirement := range requirements {
		result = append(result, session.RoleItemRequirement{
			Name:  requirement.Name,
			Count: requirement.Count,
		})
	}
	return result
}

var sourceSkillShopRequirementByKey = parseSourceSkillShopRequirementRows(sourceSkillShopRequirementRows)

var sourceSkillShopSingleLevelByKey = map[string]bool{
	sourceSkillShopKey("skill1", 3): true,
	sourceSkillShopKey("skill3", 9): true,
}

const sourceSkillShopRequirementRows = `
skill1|0|铜钱|163.png|500
skill1|1|铜钱|163.png|500
skill1|2|铜钱|163.png|500
skill1|3|铜钱|163.png|500
skill1|4|铜钱|163.png|500
skill1|5|银元宝|39.png|1
skill1|6|银元宝|39.png|3
skill1|7|银元宝|39.png|3
skill1|8|银元宝|39.png|5
skill1|9|银元宝|39.png|10
skill1|10|铜钱|163.png|500
skill1|11|银元宝|39.png|1
skill1|12|银元宝|39.png|5
skill1|13|银元宝|39.png|3
skill1|14|银元宝|39.png|4
skill1|15|银元宝|39.png|10
skill1|16|铜钱|163.png|500
skill1|17|银元宝|39.png|2
skill1|18|银元宝|39.png|4
skill1|19|银元宝|39.png|4
skill1|20|银元宝|39.png|3
skill1|21|银元宝|39.png|10
skill2|0|铜钱|163.png|500
skill2|1|铜钱|163.png|500
skill2|2|铜钱|163.png|500
skill2|3|铜钱|163.png|500
skill2|4|银元宝|39.png|2
skill2|5|银元宝|39.png|4
skill2|6|铜钱|163.png|500
skill2|7|银元宝|39.png|2
skill2|8|银元宝|39.png|4
skill2|9|银元宝|39.png|3
skill2|10|银元宝|39.png|2
skill2|11|银元宝|39.png|1
skill2|12|银元宝|39.png|4
skill2|13|银元宝|39.png|1
skill2|14|银元宝|39.png|2
skill2|15|银元宝|39.png|3
skill2|16|银元宝|39.png|5
skill2|17|银元宝|39.png|3
skill2|18|银元宝|39.png|7
skill2|19|银元宝|39.png|6
skill2|20|银元宝|39.png|6
skill2|21|银元宝|39.png|3
skill2|22|银元宝|39.png|5
skill2|23|银元宝|39.png|4
skill3|0|铜钱|163.png|500
skill3|1|铜钱|163.png|500
skill3|2|铜钱|163.png|500
skill3|3|铜钱|163.png|500
skill3|4|铜钱|163.png|500
skill3|5|铜钱|163.png|500
skill3|6|铜钱|163.png|500
skill3|7|银元宝|39.png|3
skill3|8|银元宝|39.png|2
skill3|9|银元宝|39.png|1
skill3|10|银元宝|39.png|2
skill3|11|银元宝|39.png|10
skill3|12|铜钱|163.png|500
skill3|13|银元宝|39.png|1
skill3|14|银元宝|39.png|1
skill3|15|银元宝|39.png|1
skill3|16|银元宝|39.png|1
skill3|17|银元宝|39.png|4
skill3|18|银元宝|39.png|3
skill3|19|银元宝|39.png|10
skill3|20|铜钱|163.png|500
skill3|21|银元宝|39.png|2
skill3|22|银元宝|39.png|5
skill3|23|银元宝|39.png|3
skill3|24|银元宝|39.png|5
skill3|25|银元宝|39.png|10
`

const sourceWarriorSkillShopRows = `
0|武器专精|631.png|被动技能||用于提升物理攻击力
1|强体质|632.png|被动技能||用于提升气力上限
2|抗击打|633.png|被动技能||用于提升物理防御
3|挑衅|634.png|技能·通用|群体·状态|使自己成为敌人的首要攻击目标
4|卷叶式|636.png|技能·单剑系|单体·攻击|提升对敌人造成的物理伤害
5|强贯式|637.png|技能·单剑系|单体·攻击|对敌人造成一定的物理伤害 / 击中敌人的同时使其进入卸甲(削弱其物理防御)状态 / 叠加施放将削弱其造成卸甲的功效
6|凝神式|638.png|技能·单剑系|单体·状态|提升命中的同时爆击翻倍
7|狂舞式|639.png|技能·单剑系|单体·攻击|提升对敌人造成的物理伤害 / 击中敌人时有机率使其眩晕
8|气愈式|640.png|技能·单剑系|单体·状态|在限定回合内 / 每回合自动恢复一定的气力
9|奥义.飘血|641.png|技能·单剑系|单体·攻击|特殊发动条件:3格魂元 / 大幅提升对敌人造成的物理伤害 / 进攻时将提升一定的命中
10|多段斩|644.png|技能·单刀系|单体·攻击|提升对敌人造成的物理伤害
11|嗜血斩|645.png|技能·单刀系|单体·攻击|对敌人造成物理伤害 / 击中敌人的同时有一定机率将对敌人造成的伤害转换为气力。
12|狂爆|646.png|技能·单刀系|单体·状态|3回合内物理攻击力翻倍 / 并降低100%的物理防御
13|红月斩|647.png|技能·单刀系|群体·攻击|对所有敌人造成一定的物理伤害
14|血切|648.png|技能·单刀系|单体·状态|对敌人造成一定的物理伤害 / 击中后有机率使敌人进入外伤(每回合损失气力)状态
15|奥义.雷魂斩|649.png|技能·单刀系|单体·攻击|特殊发动条件:需要3格魂元 / 提升240%的物理伤害
16|劈山棍法|651.png|技能·棍系|单体·攻击|提升对敌人造成的物理伤害
17|力释棍术|652.png|技能·棍系|单体·状态|提升物理攻击力
18|盘龙棍法|654.png|技能·棍系|群体·攻击|对所有敌人造成物理伤害
19|夜叉棍法|655.png|技能·棍系|单体·攻击|提升对敌人造成的物理伤害 / 有机率对敌人造成内伤(削弱其物理攻击和魔法攻击) / 叠加施放将削弱其造成内伤的功效
20|封御棍术|656.png|技能·棍系|单体·状态|一定回合内提升物理防御和魔法防御
21|奥义.六合棍法|657.png|技能·棍系|单体·攻击|特殊发动条件:3格魂元 / 大幅提升对敌人造成的物理伤害 / 进攻时提升300%的命中
`

const sourceMageSkillShopRows = `
0|苦心经|692.png|被动技能||用于提升魔法攻击力
1|精元经|693.png|被动技能||用于提升精力上限
2|御气经|694.png|被动技能||用于提升魔法防御
3|炎狩术|702.png|技能·法杖系|单体·攻击|提升对敌人造成的魔法伤害
4|赤焰魔咒|703.png|技能·法杖系|单体·攻击|学习条件: / 需要【炎狩术 Lv.3】 / 提升对敌人造成的魔法伤害
5|火神咒|704.png|技能·法杖系|群体·攻击|学习条件: / 需要【2格魂元】 / 需要【炎狩术 Lv.3】 / 需要【赤焰魔咒 Lv.4】 / 对所有敌人造成魔法伤害 / 有机率使用敌人进入外伤(每回合损失气力)状态
6|雷击|705.png|技能·法杖系|单体·攻击|提升对敌人造成魔法伤害 / 有机率使用敌人进入迟钝(削弱敌人命中和回避)状态 / 进攻时候提升命中
7|雷爆咒|706.png|技能·法杖系|群体·攻击|学习条件: / 需要【雷击 Lv.2】 / 对所有敌人造成魔法伤害 / 有机率使用敌人进入麻痹(眩晕并使敌人每回合损失气力)状态 / 进攻时提升命中
8|雷龙强袭|707.png|技能·法杖系|单体·攻击|学习条件: / 需要【2格魂元】 / 需要【雷击 Lv.2】 / 需要【雷爆咒 Lv.4】 / 对敌人造成强大的魔法伤害 / 进攻时提升命中
9|巫毒魔咒|708.png|技能·法杖系|单体·状态|有机率使敌人中毒(降低敌人魔法防御和物理防御并且使其每回合损失气力)
10|石雨术|709.png|技能·法杖系|群体·攻击|对所有敌人造成魔法伤害 / 击中敌人时有机率使其眩晕
11|寒冰弹|710.png|技能·法杖系|单体·攻击|对敌人造成魔法伤害 / 击中敌人时有机率使其进入内伤(削弱其物理攻击和魔法攻击)状态
12|冰封|711.png|技能·法杖系|单体·攻击|学习条件: / 需要【寒冰弹 Lv.2】 / 对敌人造成魔法伤害 / 击中敌人时有机率使其进入冰冻(眩晕并降低其物理防御和魔法防御力)状态
13|驱毒咒|723.png|技能·法杖系|单体·辅助|解除目标的中毒状态
14|回伤术|713.png|技能·法杖系|单体·辅助|解除目标的内伤和外伤状态 / 并增加一定回合的气疗状态，每回合提升一定的气力
15|圣光诀|714.png|技能·法杖系|单体·辅助|解除目标的混乱、眩晕、冰冻、麻痹状态
16|还魂术|715.png|技能·法杖系|单体·恢复|学习条件: / 需要【魂之石x1】 / 需要【圣光诀 Lv.1】 / 复活在战斗中死亡的角色 / 并恢复一定的气力
17|愈气术|716.png|技能·法杖系|单体·恢复|学习条件: / 需要【回伤术 Lv.1】 / 恢复目标一定的气力值
18|归原术|717.png|技能·法杖系|群体·恢复|学习条件: / 需要【愈气术 Lv.3】 / 恢复所有队员一定的气力值
19|魔障术|718.png|技能·法杖系|单体·状态|一定回合内将受到的部分伤害转化为精力损失
20|精甲术|719.png|技能·法杖系|单体·状态|提升目标的物理防御力
21|攻心术|721.png|技能·法杖系|单体·攻击|魔法伤害加成，并提高魔法攻击力
22|魔封术|720.png|技能·法杖系|单体·攻击|魔法伤害加成，有机率使敌人进入溃法(降低魔法防御)状态
23|意志打击|722.png|技能·法杖系|单体·状态|有机率使敌人进入迟钝(削弱敌人的命中和回避)状态
`

const sourceRangerSkillShopRows = `
0|武器娴熟|661.png|被动技能||用于提升物理攻击力
1|灵力进修|662.png|被动技能||用于提升魔法攻击和精力上限
2|爆发力|663.png|被动技能||用于提升爆击
3|幻影|664.png|被动技能||用于提升回避
4|精神力|665.png|被动技能||用于提升命中
5|多段刺|682.png|技能·匕首系|单体·攻击|提高对敌人造成的物理伤害
6|魔力突刺|683.png|技能·匕首系|单体·攻击|提升对敌人造成的物理伤害 / 并附加魔法伤害
7|疾风刺|684.png|技能·匕首系|单体·攻击|对敌人造成物理伤害 / 击中敌人时有机率使其进入迟钝状态(削减其命中和回避)
8|投毒|660.png|技能·匕首系|单体·状态|有机率使敌人中毒 / (一定回合内减低对手的物理防御和魔法防御 / 并使其每回合损失气力)
9|解毒术|685.png|技能·匕首系|单体·恢复|解除自身的中毒状态
10|强力飞镖|686.png|技能·匕首系|单体·攻击|对敌人造成物理伤害 / 进攻时候提升一定的物理攻击力
11|奥义.暗杀者|687.png|技能·匕首系|单体·攻击|特殊发动条件:3格魂元 / 大幅提升对敌人造成的物理伤害
12|强射|666.png|技能·弓系|单体·攻击|提升对敌人造成的物理伤害
13|火箭速射|667.png|技能·弓系|单体·攻击|特殊发动条件:火之箭x1 / 提升对敌人造成的物理伤害 / 击中敌人时有机率使其进入外伤(每回合损失气力)状态
14|冰箭速射|668.png|技能·弓系|单体·攻击|特殊发动条件:冰之箭x1 / 对敌人造成一定的物理伤害 / 击中敌人时有机率使其进入内伤(削弱物理攻击和魔法攻击)状态
15|魔力速射|669.png|技能·弓系|单体·攻击|特殊发动条件:魔箭x1 / 对敌人造成物理伤害以及魔法伤害 / 进攻时候提升一定的魔法攻击力
16|暗影箭|670.png|技能·弓系|单体·攻击|特殊发动条件:暗之箭x1 / 对敌人造成一定的物理伤害 / 击中敌人时有机率使其进入混乱状态
17|贯甲连矢|671.png|技能·弓系|单体·攻击|特殊发动条件:穿甲箭x1 / 提升对敌人造成的物理伤害 / 进攻时增加一定的物理攻击力
18|毒矢|672.png|技能·弓系|单体·攻击|特殊发动条件:毒箭x1 / 对敌人造成一定的物理伤害 / 击中敌人时有机率使其进入中毒状态 / (降低敌人的物理防御和魔法防御 / 并使其每回合损失气力)
19|奥义.轰雷矢|673.png|技能·弓系|单体·攻击|特殊发动条件:2格魂元 / 大幅提升对敌人造成的魔法伤害 / 击中敌人时有机率使其进入麻痹(眩晕并每回合损失气力)状态
20|连击|675.png|技能·拳套系|单体·攻击|提升对敌人造成的物理伤害
21|重烈|676.png|技能·拳套系|单体·攻击|提升对敌人造成的物理伤害 / 击中敌人时有机率使其进入眩晕状态
22|气运丹田|677.png|技能·拳套系|单体·状态|增加一格魂元 / 并提升一定的魔法防御，物理防御和物理攻击力
23|破魂打|678.png|技能·拳套系|单体·攻击|进攻时增加一定的物理攻击力 / 击中敌人时有机率削弱其魂元
24|移形换影|679.png|技能·拳套系|单体·状态|提升自身回避
25|奥义.修罗幻翼拳|680.png|技能·拳套系|单体·攻击|特殊发动条件:3格魂元 / 大幅提升对敌人造成的物理伤害 / 进攻时提升一定的物理攻击力
`

func classicTownSkillInfoPushes(roleID string, skills []session.RoleSkill) []classicTownSkillInfoPush {
	result := make([]classicTownSkillInfoPush, 0, len(skills))
	for _, skill := range skills {
		result = append(result, classicTownSkillInfoPushFromRoleSkill(roleID, skill))
	}
	return result
}

func classicTownSkillInfoPushFromRoleSkill(roleID string, skill session.RoleSkill) classicTownSkillInfoPush {
	return classicTownSkillInfoPush{
		Handle:      roleID,
		Name:        skill.Name,
		Level:       skill.Level,
		Type:        skill.Type,
		Icon:        skill.Icon,
		Description: sourceRoleSkillDescription(skill.Name, skill.Level, skill.Description),
	}
}

func classicTownItemInfoPushFromRoleItem(item session.RoleItem) classicTownItemInfoPush {
	return classicTownItemInfoPush{
		Handle:      item.Handle,
		Type:        item.Type,
		Name:        item.Name,
		ItemType:    item.ItemType,
		Display:     item.Display,
		Description: item.Description,
		Count:       item.Count,
		Index:       item.Index,
		Level:       item.Level,
		EndTime:     item.EndTime,
		Owner:       item.Owner,
		ItemLevel:   item.ItemLevel,
	}
}

func buildClassicTownCurrencyPush(roleID string, currencies session.RoleCurrencies) *classicTownCurrencyPush {
	return &classicTownCurrencyPush{
		Handle:     roleID,
		Currencies: currencies,
	}
}

func roleCurrenciesOrEmpty(store *session.Store, playerID string, roleID string) session.RoleCurrencies {
	currencies, ok := store.GetRoleCurrencies(playerID, roleID)
	if !ok {
		return session.RoleCurrencies{}
	}
	return currencies
}
