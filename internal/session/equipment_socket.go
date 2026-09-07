package session

import (
	_ "embed"
	"encoding/csv"
	"regexp"
	"strconv"
	"strings"

	"ai-server/internal/classicdata"
)

//go:embed config/classic-equipment-socket.csv
var equipmentSocketCSV string
var equipmentSocketRates = loadEquipmentSocketRates(equipmentSocketCSV)
var socketCountPattern = regexp.MustCompile(`&18@([^&]*)`)
var socketMaximumPattern = regexp.MustCompile(`凿孔上限\s*(\d+)\s*格`)

func loadEquipmentSocketRates(text string) []int {
	rows, err := csv.NewReader(strings.NewReader(text)).ReadAll()
	if err != nil || len(rows) != 10 || strings.Join(rows[0], ",") != "target_holes,success_percent,basis" {
		panic("invalid equipment socket config")
	}
	rates := make([]int, 9)
	for i, row := range rows[1:] {
		hole, e1 := strconv.Atoi(row[0])
		rate, e2 := strconv.Atoi(row[1])
		if e1 != nil || e2 != nil || hole != i+1 || rate <= 0 || rate > 100 || (i > 0 && rate >= rates[i-1]) || row[2] == "" {
			panic("invalid equipment socket rate")
		}
		rates[i] = rate
	}
	if rates[8] != 1 {
		panic("ninth socket rate must be 1 percent")
	}
	return rates
}

type RoleEquipmentSocketResult struct {
	Role                        RoleSummary
	PlayerBase                  PlayerBaseData
	UpdatedItems                []RoleItem
	ClearedItems                []RoleItemClear
	Applied, Succeeded          bool
	ResultMessage, ErrorMessage string
}

func equipmentSocketState(item RoleItem) (int, int, bool) {
	if !strings.HasPrefix(item.Description, "f_i_") {
		return 0, 0, false
	}
	body := strings.Split(item.Description, "f_i_")[1]
	current := 0
	if fields := socketCountPattern.FindAllStringSubmatch(body, -1); len(fields) > 0 {
		if len(fields) != 1 {
			return 0, 0, false
		}
		value, err := strconv.Atoi(strings.TrimSpace(fields[0][1]))
		if err != nil || value < 0 {
			return 0, 0, false
		}
		current = value
	}
	maximum := 0
	if match := socketMaximumPattern.FindStringSubmatch(body); match != nil {
		maximum, _ = strconv.Atoi(match[1])
	} else {
		table, err := classicdata.LoadTable(classicdata.TableItem)
		if err != nil {
			return 0, 0, false
		}
		for _, row := range table.Rows {
			if row["name"] == item.Name && row["icon"] == item.Display && row["item_type"] == "equip" {
				maximum, _ = strconv.Atoi(row["max_socket_count"])
				break
			}
		}
	}
	return current, maximum, maximum > 0 && maximum <= len(equipmentSocketRates) && current <= maximum
}

func rewriteEquipmentSockets(description string, count int) string {
	parts := strings.SplitN(description, "f_i_", 3)
	body := parts[1]
	marker := "&18@" + strconv.Itoa(count)
	if socketCountPattern.MatchString(body) {
		body = socketCountPattern.ReplaceAllString(body, marker)
	} else {
		body = strings.TrimSuffix(body, "&") + marker + "&"
	}
	result := "f_i_" + body
	if len(parts) == 3 {
		result += "f_i_" + parts[2]
	}
	return result
}

// Captured primitive: one tool, +1 socket on success, unchanged equipment on failure.
// Rates are the user-designed local curve, without an inferred VIP multiplier.
func (store *Store) SocketRoleEquipment(pid, rid, sourceType string, sourceIndex int, targetType string, targetIndex int) RoleEquipmentSocketResult {
	store.mu.Lock()
	defer store.mu.Unlock()
	reject := func(msg string) RoleEquipmentSocketResult { return RoleEquipmentSocketResult{ErrorMessage: msg} }
	if sourceType != "背包" || (targetType != "背包" && targetType != "装备") {
		return reject("凿孔材料或目标容器无效。")
	}
	roles := store.rolesByPID[pid]
	for i, role := range roles {
		if role.RoleID != rid {
			continue
		}
		tool, found := findRoleItem(role.Items, sourceType, sourceIndex)
		target, targetFound := findRoleItem(role.Items, targetType, targetIndex)
		if !found || !targetFound || tool.Count < 1 {
			return reject("凿孔材料或目标装备不存在。")
		}
		if tool.Name != "凿孔器" || tool.Display != "776.png" || tool.ItemType != "oneI" || target.ItemType != "equip" || target.Count != 1 {
			return reject("凿孔器只能用于装备。")
		}
		current, maximum, valid := equipmentSocketState(target)
		if !valid {
			return reject("装备凿孔数据无效。")
		}
		if current >= maximum {
			return reject("已达到凿孔上限。")
		}
		roll := defaultEquipmentRefinementRoll
		if store.socketRoll != nil {
			roll = store.socketRoll
		}
		succeeded := roll(10000) < equipmentSocketRates[current]*100
		if succeeded {
			target.Description = rewriteEquipmentSockets(target.Description, current+1)
		}
		result := RoleEquipmentSocketResult{Applied: true, Succeeded: succeeded, ResultMessage: "凿孔失败"}
		if succeeded {
			result.ResultMessage = "凿孔成功"
		}
		updated := make([]RoleItem, 0, len(role.Items))
		for _, item := range role.Items {
			if item.Type == sourceType && item.Index == sourceIndex {
				item.Count--
				if item.Count == 0 {
					result.ClearedItems = append(result.ClearedItems, RoleItemClear{Type: item.Type, Index: item.Index})
					continue
				}
				result.UpdatedItems = append(result.UpdatedItems, item)
			} else if item.Type == targetType && item.Index == targetIndex {
				item = target
				result.UpdatedItems = append(result.UpdatedItems, item)
			}
			updated = append(updated, item)
		}
		next := role
		next.Items = updated
		roles[i] = next
		store.rolesByPID[pid] = roles
		if err := store.persistRoleStateLocked(pid, rid); err != nil {
			roles[i] = role
			return reject("凿孔结果保存失败。")
		}
		result.Role = next
		result.PlayerBase = playerBaseDataFromRole(pid, next)
		return result
	}
	return reject("凿孔角色不存在。")
}
