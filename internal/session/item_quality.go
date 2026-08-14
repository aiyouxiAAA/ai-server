package session

import (
	"regexp"
	"strings"

	"ai-server/internal/classicdata"
)

var (
	classicItemQualityColorPattern      = regexp.MustCompile(`^f_i_[^^&]+\^([0-9A-Fa-f]{6})`)
	classicItemQualityTitlePattern      = regexp.MustCompile(`^(f_i_[^^&]+)(?:\^([0-9A-Fa-f]{6}))?`)
	classicItemQualityColorValuePattern = regexp.MustCompile(`^[0-9A-Fa-f]{6}$`)
)

// CapturedRoleItemQualityColor returns the captured title color stored in the
// canonical item table. Source descriptions remain the fallback for templates
// that predate the table; Flash defaults titles without a color suffix to white.
func CapturedRoleItemQualityColor(name string, description string) string {
	if color, ok := capturedClassicDataItemQualityColor(name); ok {
		return color
	}
	if matches := classicItemQualityColorPattern.FindStringSubmatch(strings.TrimSpace(description)); len(matches) == 2 {
		return strings.ToLower(matches[1])
	}
	return "ffffff"
}

func applyCapturedRoleItemQualityColor(item RoleItem) RoleItem {
	color, ok := capturedClassicDataItemQualityColor(item.Name)
	if !ok || !strings.HasPrefix(item.Description, "f_i_"+item.Name) {
		return item
	}
	matches := classicItemQualityTitlePattern.FindStringSubmatch(item.Description)
	if len(matches) != 3 || strings.EqualFold(matches[2], color) {
		return item
	}
	item.Description = matches[1] + "^" + color + item.Description[len(matches[0]):]
	return item
}

func capturedClassicDataItemQualityColor(name string) (string, bool) {
	row, ok, err := classicdata.FindItemByName(strings.TrimSpace(name))
	if err != nil || !ok {
		return "", false
	}
	color := normalizeClassicItemQualityColor(row["quality_color"])
	return color, color != ""
}

func normalizeClassicItemQualityColor(color string) string {
	color = strings.TrimPrefix(strings.TrimSpace(color), "#")
	if !classicItemQualityColorValuePattern.MatchString(color) {
		return ""
	}
	return strings.ToLower(color)
}
