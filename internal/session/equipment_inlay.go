package session

import (
	"regexp"
	"strconv"
	"strings"
)

type EquipmentInlayRequest struct {
	TargetType                string `json:"targetType"`
	TargetIndex               *int   `json:"targetIndex"`
	SourceType                string `json:"sourceType"`
	SourceIndex               *int   `json:"sourceIndex"`
	SocketIndex               *int   `json:"socketIndex"`
	ExpectedTargetDescription string `json:"expectedTargetDescription"`
	ExpectedGemDescription    string `json:"expectedGemDescription"`
	ExpectedGemCount          int    `json:"expectedGemCount"`
	Replace                   bool   `json:"replace"`
}

type RoleEquipmentInlayResult struct {
	Applied      bool
	Message      string
	Role         RoleSummary
	PlayerBase   PlayerBaseData
	UpdatedItems []RoleItem
	ClearedItems []RoleItemClear
}

var inlayFieldPattern = regexp.MustCompile(`&([0-9]+)@`)

// Attribute values may contain escaped commas (&0;), so plain '&' splitting is incorrect.
func inlayField(body, key string) string {
	matches := inlayFieldPattern.FindAllStringSubmatchIndex(body, -1)
	for i, m := range matches {
		if body[m[2]:m[3]] != key {
			continue
		}
		end := len(body)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		return strings.TrimSuffix(body[m[1]:end], "&")
	}
	return ""
}

func inlaySocketBodies(description string, opened int) (map[int]string, bool) {
	bodies := strings.Split(description, "f_i_")
	if len(bodies) < 2 || bodies[0] != "" {
		return nil, false
	}
	result := map[int]string{}
	for _, body := range bodies[2:] {
		index, err := strconv.Atoi(inlayField(body, "102"))
		if err != nil || index < 0 || index >= opened || inlayField(body, "24") != "镶嵌物" {
			return nil, false
		}
		if _, exists := result[index]; exists {
			return nil, false
		}
		result[index] = body
	}
	return result, true
}

// One accepted insertion consumes one gem; no inferred random failure, currency cost or gem return.
func (store *Store) InlayRoleEquipment(pid, rid string, request EquipmentInlayRequest) RoleEquipmentInlayResult {
	store.mu.Lock()
	defer store.mu.Unlock()
	reject := func(message string) RoleEquipmentInlayResult { return RoleEquipmentInlayResult{Message: message} }
	if request.TargetIndex == nil || request.SourceIndex == nil || request.SocketIndex == nil ||
		*request.TargetIndex < 0 || *request.SourceIndex < 0 || *request.SocketIndex < 0 ||
		request.SourceType != "背包" || (request.TargetType != "装备" && request.TargetType != "背包") {
		return reject("镶嵌参数无效。")
	}
	roles := store.rolesByPID[pid]
	for i, role := range roles {
		if role.RoleID != rid {
			continue
		}
		target, ok := findRoleItem(role.Items, request.TargetType, *request.TargetIndex)
		gem, found := findRoleItem(role.Items, request.SourceType, *request.SourceIndex)
		if !ok || !found || target.ItemType != "equip" || target.Count != 1 || gem.Count <= 0 {
			return reject("装备或宝石不存在。")
		}
		if target.Description != request.ExpectedTargetDescription || gem.Description != request.ExpectedGemDescription || gem.Count != request.ExpectedGemCount {
			return reject("装备或宝石已变化，请重新选择。")
		}
		parts := strings.Split(gem.Description, "f_i_")
		if len(parts) != 2 || parts[0] != "" || inlayField(parts[1], "24") != "镶嵌物" || gem.ItemType == "equip" || gem.Display == "" {
			return reject("该物品不是镶嵌物。")
		}
		if inlayField(parts[1], "102") != "" {
			return reject("宝石数据无效。")
		}
		opened, _, valid := equipmentSocketState(target)
		if !valid || *request.SocketIndex >= opened {
			return reject("目标孔位尚未开启。")
		}
		sockets, valid := inlaySocketBodies(target.Description, opened)
		if !valid {
			return reject("装备孔位数据无效。")
		}
		if _, occupied := sockets[*request.SocketIndex]; occupied && !request.Replace {
			return reject("此孔已有宝石，请确认覆盖。")
		}
		body := strings.TrimSuffix(parts[1], "&")
		if icon := inlayField(body, "101"); icon != "" && icon != gem.Display {
			return reject("宝石图标数据不一致。")
		}
		if inlayField(body, "101") == "" {
			body += "&101@" + gem.Display
		}
		sockets[*request.SocketIndex] = body + "&102@" + strconv.Itoa(*request.SocketIndex)
		// Equipment body stays byte-identical; never add gem percentages to refinement/base text.
		target.Description = "f_i_" + strings.Split(target.Description, "f_i_")[1]
		for socket := 0; socket < opened; socket++ {
			if body, ok := sockets[socket]; ok {
				target.Description += "f_i_" + body
			}
		}
		result := RoleEquipmentInlayResult{Applied: true, Message: "镶嵌成功"}
		next := role
		next.Items = make([]RoleItem, 0, len(role.Items))
		for _, item := range role.Items {
			if item.Type == gem.Type && item.Index == gem.Index {
				item.Count--
				if item.Count == 0 {
					result.ClearedItems = append(result.ClearedItems, RoleItemClear{Type: item.Type, Index: item.Index})
					continue
				}
				result.UpdatedItems = append(result.UpdatedItems, item)
			} else if item.Type == target.Type && item.Index == target.Index {
				item = target
				result.UpdatedItems = append(result.UpdatedItems, item)
			}
			next.Items = append(next.Items, item)
		}
		roles[i] = next
		store.rolesByPID[pid] = roles
		if err := store.persistRoleStateLocked(pid, rid); err != nil {
			roles[i] = role
			return reject("镶嵌结果保存失败。")
		}
		result.Role = next
		result.PlayerBase = playerBaseDataFromRole(pid, next)
		return result
	}
	return reject("镶嵌角色不存在。")
}
