package session

import (
	"ai-server/internal/profession"
	"fmt"
)

func applyOriginalProfession(role RoleSummary) RoleSummary {
	definition := profession.Definitions[0]
	role.Voc = definition.Name
	role.Skills = nil
	role.FastPanel = nil
	for _, skill := range profession.Skills {
		if skill.ProfessionID != definition.ID {
			continue
		}
		role.Skills = append(role.Skills, originalRoleSkill(definition, skill, skill.Level))
		if skill.Kind == "skill" && skill.Slot >= 0 {
			role.FastPanel = append(role.FastPanel, RoleFastPanelEntry{Index: skill.Slot, Type: "skill", Name: skill.Name})
		}
	}
	role.SkillCap = maxInt(defaultSkillCap, len(role.Skills))
	return role
}

func originalRoleSkill(definition profession.Definition, skill profession.Skill, level int) RoleSkill {
	category := "单体·攻击"
	if skill.Kind == "passive" {
		category = "被动"
	}
	text := fmt.Sprintf("f_s_%s&9@%s&8@%s&22@战斗&2@%d&4@%s", skill.Name, category, definition.Name, skill.MPCost, skill.Description)
	return RoleSkill{Name: skill.Name, Level: level, Type: skill.SourceType, Icon: skill.Icon, Description: text, MaxLevel: 1}
}

// Only the current catalog can remain learned or occupy a skill shortcut.
func refreshOriginalProfessionSkills(role RoleSummary) RoleSummary {
	definition, ok := profession.ByName(role.Voc)
	current := []RoleSkill{}
	for _, owned := range role.Skills {
		if skill, exists := profession.BySkillName(owned.Name); exists {
			if ok && skill.ProfessionID == definition.ID {
				current = append(current, originalRoleSkill(definition, skill, owned.Level))
			} else if skill.ID == "skill-normal-attack" {
				current = append(current, originalRoleSkill(profession.Definitions[0], skill, owned.Level))
			}
		}
	}
	role.Skills = current
	role.FastPanel = filterRoleFastPanelEntries(role.FastPanel, role.Skills)
	return role
}
