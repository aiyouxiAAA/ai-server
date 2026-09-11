package session

import (
	"crypto/rand"
	_ "embed"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	mathrand "math/rand"
	"strconv"
	"strings"
)

// Original, user-approved progression. Values are tunable first-pass balancing.
//
//go:embed config/equipment-star.csv
var equipmentStarCSV string

type starRule struct {
	Code, Name string
	Base, Step int
}

var starRules = loadStarRules()
var starCosts = []int{1, 1, 2, 2, 3}

func loadStarRules() []starRule {
	rows, err := csv.NewReader(strings.NewReader(equipmentStarCSV)).ReadAll()
	if err != nil {
		panic(err)
	}
	result := []starRule{}
	for _, row := range rows[1:] {
		base, e := strconv.Atoi(row[2])
		if e != nil {
			panic(e)
		}
		step, e := strconv.Atoi(row[3])
		if e != nil {
			panic(e)
		}
		result = append(result, starRule{row[0], row[1], base, step})
	}
	return result
}

type EquipmentStarAttribute struct {
	Code  string `json:"code"`
	Name  string `json:"name"`
	Value int    `json:"value"`
	Step  int    `json:"step"`
}
type EquipmentStarState struct {
	ID         string                   `json:"id"`
	Level      int                      `json:"level"`
	Attributes []EquipmentStarAttribute `json:"attributes"`
}
type EquipmentStarRequest struct {
	Action       string   `json:"action"`
	Type         string   `json:"type"`
	Index        int      `json:"index"`
	ID           string   `json:"id"`
	ExpectedStar int      `json:"expectedStar"`
	Materials    []string `json:"materials"`
	Choice       string   `json:"choice"`
}
type EquipmentStarResult struct {
	Success      bool
	Message      string
	Role         RoleSummary
	PlayerBase   PlayerBaseData
	UpdatedItems []RoleItem
	ClearedItems []RoleItemClear
}

func cloneEquipmentStar(s *EquipmentStarState) *EquipmentStarState {
	if s == nil {
		return nil
	}
	next := *s
	next.Attributes = append([]EquipmentStarAttribute{}, s.Attributes...)
	return &next
}
func newEquipmentStar(item RoleItem) *EquipmentStarState {
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		panic(err)
	}
	s := &EquipmentStarState{ID: hex.EncodeToString(id), Attributes: []EquipmentStarAttribute{}}
	addStarAttribute(s, item)
	addStarAttribute(s, item)
	return s
}
func addStarAttribute(s *EquipmentStarState, item RoleItem) {
	pool := []starRule{}
	for _, r := range starRules {
		found := false
		for _, a := range s.Attributes {
			if a.Code == r.Code {
				found = true
			}
		}
		if !found {
			pool = append(pool, r)
		}
	}
	r := pool[mathrand.Intn(len(pool))]
	step := r.Base + maxInt(0, parseClassicDescriptionSignedInt(item.Description, "21"))/20*r.Step
	s.Attributes = append(s.Attributes, EquipmentStarAttribute{r.Code, r.Name, step, step})
}
func EquipmentStarMaterialEligible(target, item RoleItem) bool {
	return item.ItemType == "equip" && item.Type == "背包" && item.Count == 1 && item.Name == target.Name && item.Display == target.Display && item.ItemLevel == target.ItemLevel && item.Star != nil && target.Star != nil && item.Star.ID != target.Star.ID && item.Star.Level == 0 && item.Level == 0 && !item.Locked && item.EndTime == 0 && (item.Owner == "" || item.Owner == target.Owner) && strings.Count(item.Description, "f_i_") <= 1
}

// Prepare initializes owned inventory once and commits it before exposing any random results.
// Upgrade uses instance ID + expected star as a compare-and-swap guard against stale/replayed requests.
func (store *Store) EquipmentStar(pid, rid string, q EquipmentStarRequest) EquipmentStarResult {
	store.mu.Lock()
	defer store.mu.Unlock()
	reject := func(s string) EquipmentStarResult { return EquipmentStarResult{Message: s} }
	if q.Action != "prepare" && q.Action != "upgrade" {
		return reject("升星操作无效。")
	}
	if q.Type != "背包" && q.Type != "装备" {
		return reject("只能操作自己的背包或已穿戴装备。")
	}
	roles := store.rolesByPID[pid]
	for ri, original := range roles {
		if original.RoleID != rid {
			continue
		}
		next := original
		next.Items = cloneRoleItems(original.Items)
		ti := -1
		for i, item := range next.Items {
			if item.Type == q.Type && item.Index == q.Index {
				ti = i
				break
			}
		}
		if ti < 0 || next.Items[ti].ItemType != "equip" || next.Items[ti].Count != 1 {
			return reject("装备不存在或不是独立装备。")
		}
		result := EquipmentStarResult{Success: true, Message: "装备属性已就绪"}
		if q.Action == "prepare" {
			for i, item := range next.Items {
				if item.ItemType == "equip" && item.Count == 1 && (item.Type == "背包" || item.Type == "装备") && item.Star == nil {
					item.Star = newEquipmentStar(item)
					next.Items[i] = item
					result.UpdatedItems = append(result.UpdatedItems, item)
				}
			}
		} else {
			target := next.Items[ti]
			s := target.Star
			if s == nil || q.ID != s.ID || q.ExpectedStar != s.Level {
				return reject("装备状态已变化，请重新选择。")
			}
			if s.Level < 0 || s.Level >= 5 {
				return reject("装备已满星。")
			}
			if target.EndTime != 0 {
				return reject("限时装备不参与升星。")
			}
			if len(q.Materials) != starCosts[s.Level] {
				return reject("同款材料数量不足或不正确。")
			}
			consumed := map[string]bool{}
			for _, id := range q.Materials {
				if consumed[id] {
					return reject("不能重复选择同一件材料。")
				}
				found := false
				for _, item := range next.Items {
					if item.Star != nil && item.Star.ID == id && EquipmentStarMaterialEligible(target, item) {
						found = true
						break
					}
				}
				if !found {
					return reject("材料已变化、被保护或不是可用同款。")
				}
				consumed[id] = true
			}
			s = cloneEquipmentStar(s)
			if s.Level < 2 {
				addStarAttribute(s, target)
			} else {
				index := -1
				if s.Level == 4 {
					for i, a := range s.Attributes {
						if a.Code == q.Choice {
							index = i
						}
					}
					if index < 0 {
						return reject("请选择满星要强化的属性。")
					}
				} else {
					index = mathrand.Intn(len(s.Attributes))
				}
				s.Attributes[index].Value += s.Attributes[index].Step
			}
			s.Level++
			target.Star = s
			next.Items[ti] = target
			filtered := []RoleItem{}
			for _, item := range next.Items {
				if item.Star != nil && consumed[item.Star.ID] {
					result.ClearedItems = append(result.ClearedItems, RoleItemClear{Type: item.Type, Index: item.Index})
				} else {
					filtered = append(filtered, item)
				}
			}
			next.Items = filtered
			result.UpdatedItems = []RoleItem{target}
			result.Message = fmt.Sprintf("升星成功，当前%d星", s.Level)
		}
		next = syncRoleProgressionRuntimeData(next)
		roles[ri] = next
		store.rolesByPID[pid] = roles
		if err := store.persistRoleStateLocked(pid, rid); err != nil {
			roles[ri] = original
			return reject("保存失败，装备和材料均未改变。")
		}
		result.Role = next
		result.PlayerBase = playerBaseDataFromRole(pid, next)
		// Full owned inventory supplies initialized material IDs, even when prepare had nothing to change.
		if q.Action == "prepare" {
			result.UpdatedItems = cloneRoleItems(next.Items)
		}
		return result
	}
	return reject("角色不存在。")
}
