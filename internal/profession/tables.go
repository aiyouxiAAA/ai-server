// Package profession owns the current original professions, separate from historical capture data.
package profession

import (
	"embed"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
)

//go:embed config/*.csv
var files embed.FS

type Definition struct {
	ID, Name, TownManifest, BattleManifest string
	SpawnMapID                             int
}
type Skill struct {
	ProfessionID, ID, Name, Kind, SourceType, Target, ActionLabel, AnimationID string
	Level, Slot, MPCost, RequiredPower, TriggerChance                          int
	DamageMultiplier                                                           float64
	Icon, AssetStatus, CounterSkillID, Description                             string
}

var Definitions, Skills = load()

func load() ([]Definition, []Skill) {
	definitions := []Definition{}
	skills := []Skill{}
	ids := map[string]bool{}
	for _, r := range rows("player-profession.csv") {
		if r["enabled"] != "1" {
			continue
		}
		d := Definition{r["profession_id"], r["name"], r["town_manifest"], r["battle_manifest"], integer(r, "spawn_map_id")}
		if d.ID == "" || d.Name == "" || ids[d.ID] {
			panic("invalid profession identity")
		}
		ids[d.ID] = true
		definitions = append(definitions, d)
	}
	seen := map[string]bool{}
	for _, r := range rows("player-profession-skill.csv") {
		s := Skill{ProfessionID: r["profession_id"], ID: r["skill_id"], Name: r["name"], Kind: r["kind"],
			SourceType: r["source_type"], Target: r["target"], ActionLabel: r["action_label"], AnimationID: r["animation_id"],
			Level: integer(r, "default_level"), Slot: integer(r, "shortcut_slot"), MPCost: integer(r, "mp_cost"),
			RequiredPower: integer(r, "required_power"), TriggerChance: integer(r, "trigger_chance_percent"),
			DamageMultiplier: number(r, "damage_multiplier"), Icon: r["icon"], AssetStatus: r["asset_status"],
			CounterSkillID: r["counter_skill_id"], Description: r["description"]}
		if !ids[s.ProfessionID] || seen[s.ID] || s.ID == "" || s.Name == "" || s.Level < 1 || s.MPCost < 0 || s.DamageMultiplier < 0 || s.TriggerChance < 0 || s.TriggerChance > 100 {
			panic("invalid profession skill: " + s.ID)
		}
		if s.Kind != "skill" && s.Kind != "passive" {
			panic("invalid skill kind: " + s.Kind)
		}
		if s.Kind == "passive" && s.Slot != -1 {
			panic("passive must not occupy shortcut")
		}
		seen[s.ID] = true
		skills = append(skills, s)
	}
	for _, s := range skills {
		if s.CounterSkillID != "" && !seen[s.CounterSkillID] {
			panic("missing counter skill")
		}
	}
	if len(definitions) == 0 {
		panic("no enabled profession")
	}
	return definitions, skills
}
func rows(name string) []map[string]string {
	raw, err := files.ReadFile("config/" + name)
	if err != nil {
		panic(err)
	}
	records, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(raw), "\ufeff"))).ReadAll()
	if err != nil {
		panic(err)
	}
	result := []map[string]string{}
	for _, record := range records[1:] {
		row := map[string]string{}
		for i, key := range records[0] {
			row[key] = strings.TrimSpace(record[i])
		}
		result = append(result, row)
	}
	return result
}
func integer(row map[string]string, key string) int {
	n, err := strconv.Atoi(row[key])
	if err != nil {
		panic(fmt.Sprintf("%s: %v", key, err))
	}
	return n
}
func number(row map[string]string, key string) float64 {
	n, err := strconv.ParseFloat(row[key], 64)
	if err != nil {
		panic(err)
	}
	return n
}
func ByName(name string) (Definition, bool) {
	for _, d := range Definitions {
		if d.Name == strings.TrimSpace(name) {
			return d, true
		}
	}
	return Definition{}, false
}
func IDForName(name string) string { d, _ := ByName(name); return d.ID }
func BySkillID(id string) (Skill, bool) {
	for _, s := range Skills {
		if s.ID == id {
			return s, true
		}
	}
	return Skill{}, false
}
func BySkillName(name string) (Skill, bool) {
	for _, s := range Skills {
		if s.Name == strings.TrimSpace(name) {
			return s, true
		}
	}
	return Skill{}, false
}
