package session

import (
	"strconv"
	"strings"
)

const classicEquipmentRefinementSourceCapture = "D:/yzhgame/WOCClient/tmp/woc-proxy-captures/20260619_152110_886_session_33084/connections/20260619_152118_193_conn_0002/raw/client-to-server-0001.bin#19396/#19397 + server-to-client-0001.bin#31365-#31380"

var classicEquipmentRefinementAttributeCodes = []struct {
	name string
	code string
}{
	{name: "物理攻击", code: "1"},
	{name: "魔法攻击", code: "2"},
	{name: "物理防御", code: "3"},
	{name: "魔法防御", code: "4"},
	{name: "气力上限", code: "5"},
	{name: "精力上限", code: "6"},
	{name: "命中", code: "9"},
	{name: "回避", code: "10"},
	{name: "爆击", code: "11"},
}

func (store *Store) RefineRoleEquipment(playerID string, roleID string, sourceType string, sourceIndex int, targetType string, targetIndex int) RoleEquipmentRefinementResult {
	store.mu.Lock()
	defer store.mu.Unlock()

	sourceType = strings.TrimSpace(sourceType)
	targetType = strings.TrimSpace(targetType)
	roles := store.rolesByPID[playerID]
	for roleIndex := range roles {
		if roles[roleIndex].RoleID != roleID {
			continue
		}

		roles[roleIndex] = withRoleRuntimeDefaults(roles[roleIndex])
		sourceItem, sourceFound := findRoleItem(roles[roleIndex].Items, sourceType, sourceIndex)
		targetItem, targetFound := findRoleItem(roles[roleIndex].Items, targetType, targetIndex)
		if !sourceFound || !targetFound {
			return equipmentRefinementResultForRole(playerID, roles[roleIndex], RoleEquipmentRefinementResult{
				Found:        true,
				ErrorCode:    "refinement_item_missing",
				ErrorMessage: "精炼材料或目标装备不存在。",
			})
		}
		if sourceType == targetType && sourceIndex == targetIndex {
			return equipmentRefinementResultForRole(playerID, roles[roleIndex], RoleEquipmentRefinementResult{
				Found:        true,
				ErrorCode:    "refinement_same_item",
				ErrorMessage: "不能对同一物品进行精炼。",
			})
		}
		if sourceItem.ItemType != "oneI" || targetItem.ItemType != "equip" {
			return equipmentRefinementResultForRole(playerID, roles[roleIndex], RoleEquipmentRefinementResult{
				Found:        true,
				ErrorCode:    "refinement_target_invalid",
				ErrorMessage: "精炼宝石只能用于装备。",
			})
		}

		equipmentLevel := parseClassicDescriptionSignedInt(targetItem.Description, "21")
		rule, matched := classicEquipmentRefinementRuleFor(sourceItem.Name, targetItem.Level, equipmentLevel)
		if !matched {
			return equipmentRefinementResultForRole(playerID, roles[roleIndex], RoleEquipmentRefinementResult{
				Found:        true,
				ErrorCode:    "refinement_rule_unavailable",
				ErrorMessage: "该精炼宝石不能用于当前装备精炼等级。",
			})
		}

		roll := defaultEquipmentRefinementRoll
		if store.refinementRoll != nil {
			roll = store.refinementRoll
		}
		succeeded := roll(10000) < rule.SuccessRateBps
		previousLevel := targetItem.Level
		if succeeded {
			targetItem.Level++
		} else {
			targetItem.Level = maxInt(rule.FailureFloor, targetItem.Level-1)
		}
		targetItem.Description = rewriteClassicEquipmentRefinementDescription(targetItem.Description, targetItem.Level)
		targetItem = normalizeRoleItem(targetItem)

		updatedItems := make([]RoleItem, 0, len(roles[roleIndex].Items))
		updatedResults := make([]RoleItem, 0, 2)
		clearedItems := make([]RoleItemClear, 0, 1)
		for _, item := range roles[roleIndex].Items {
			switch {
			case item.Type == sourceType && item.Index == sourceIndex:
				item.Count--
				if item.Count <= 0 {
					clearedItems = append(clearedItems, RoleItemClear{Type: sourceType, Index: sourceIndex})
					continue
				}
				item = normalizeRoleItem(item)
				updatedItems = append(updatedItems, item)
				updatedResults = append(updatedResults, item)
			case item.Type == targetType && item.Index == targetIndex:
				updatedItems = append(updatedItems, targetItem)
				updatedResults = append(updatedResults, targetItem)
			default:
				updatedItems = append(updatedItems, item)
			}
		}

		roles[roleIndex].Items = normalizeRoleItems(updatedItems)
		roles[roleIndex] = syncRoleProgressionRuntimeData(roles[roleIndex])
		store.rolesByPID[playerID] = roles
		if err := store.persistRoleStateLocked(playerID, roleID); err != nil {
			return equipmentRefinementResultForRole(playerID, roles[roleIndex], RoleEquipmentRefinementResult{
				Found:        true,
				ErrorCode:    "refinement_persist_failed",
				ErrorMessage: "精炼结果保存失败。",
			})
		}

		message := "精炼成功&0;等级上升1"
		if !succeeded && targetItem.Level < previousLevel {
			message = "精炼失败&0;等级下降1"
		} else if !succeeded {
			message = "精炼失败&0;等级不变"
		}
		return equipmentRefinementResultForRole(playerID, roles[roleIndex], RoleEquipmentRefinementResult{
			SourceItem:    sourceItem,
			TargetItem:    targetItem,
			UpdatedItems:  updatedResults,
			ClearedItems:  clearedItems,
			Found:         true,
			Refined:       true,
			Succeeded:     succeeded,
			ResultMessage: message,
			Rule:          rule,
		})
	}

	return RoleEquipmentRefinementResult{ErrorCode: "role_missing", ErrorMessage: "角色不存在。"}
}

func equipmentRefinementResultForRole(playerID string, role RoleSummary, result RoleEquipmentRefinementResult) RoleEquipmentRefinementResult {
	role = withRoleRuntimeDefaults(role)
	result.Role = role
	result.PlayerBase = playerBaseDataFromRole(playerID, role)
	return result
}

func rewriteClassicEquipmentRefinementDescription(description string, refinementLevel int) string {
	for _, attribute := range classicEquipmentRefinementAttributeCodes {
		bonus := classicEquipmentRefinementBonusForAttribute(description, refinementLevel, attribute.name)
		description = rewriteClassicEquipmentRefinementContribution(description, attribute.code, bonus)
	}
	return description
}

func classicEquipmentRefinementBonusForAttribute(description string, refinementLevel int, attributeName string) int {
	bonus := 0
	remaining := description
	for {
		start := strings.Index(remaining, "[精炼+")
		if start < 0 {
			return bonus
		}
		remaining = remaining[start+len("[精炼+"):]
		thresholdEnd := strings.Index(remaining, "]")
		if thresholdEnd < 0 {
			return bonus
		}
		threshold, err := strconv.Atoi(remaining[:thresholdEnd])
		if err != nil {
			return bonus
		}
		remaining = remaining[thresholdEnd+1:]
		lineEnd := strings.IndexAny(remaining, "\r\n")
		line := remaining
		if lineEnd >= 0 {
			line = remaining[:lineEnd]
			remaining = remaining[lineEnd+1:]
		} else {
			remaining = ""
		}
		prefix := "每升一级 " + attributeName + "+"
		if !strings.HasPrefix(strings.TrimSpace(line), prefix) || refinementLevel < threshold {
			continue
		}
		amountText := strings.TrimPrefix(strings.TrimSpace(line), prefix)
		amountEnd := 0
		for amountEnd < len(amountText) && amountText[amountEnd] >= '0' && amountText[amountEnd] <= '9' {
			amountEnd++
		}
		amount, parseErr := strconv.Atoi(amountText[:amountEnd])
		if parseErr != nil || amount <= 0 {
			continue
		}
		bonus += (refinementLevel - threshold + 1) * amount
	}
}

func rewriteClassicEquipmentRefinementContribution(description string, key string, bonus int) string {
	marker := "&" + key + "@"
	start := strings.Index(description, marker)
	if start < 0 {
		return description
	}
	valueStart := start + len(marker)
	valueEnd := valueStart
	if valueEnd < len(description) && (description[valueEnd] == '+' || description[valueEnd] == '-') {
		valueEnd++
	}
	for valueEnd < len(description) && description[valueEnd] >= '0' && description[valueEnd] <= '9' {
		valueEnd++
	}
	if valueEnd == valueStart || (valueEnd == valueStart+1 && (description[valueStart] == '+' || description[valueStart] == '-')) {
		return description
	}
	contributionEnd := valueEnd
	if contributionEnd < len(description) && description[contributionEnd] == '(' {
		if close := strings.IndexByte(description[contributionEnd:], ')'); close >= 0 {
			contributionEnd += close + 1
		}
	}
	contribution := ""
	if bonus > 0 {
		contribution = "(+" + strconv.Itoa(bonus) + ")"
	}
	return description[:valueEnd] + contribution + description[contributionEnd:]
}
