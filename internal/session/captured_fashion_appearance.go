package session

import (
	"strings"

	"ai-server/internal/classicdata"
)

// Captured source-query appearance fields are read from fashion-appearance-table.
// A missing sex row was not captured and must not borrow the other sex's fields.
func isCapturedFashionAppearanceItem(item RoleItem) bool {
	rows, err := classicdata.FindFashionAppearanceRowsByName(strings.TrimSpace(item.Name))
	return err == nil && len(rows) > 0
}

func capturedFashionAppearanceSourceParams(item RoleItem, sourceQuery string) ([]roleItemAppearanceSourceParamPair, bool) {
	sex, ok := sourceQueryParamValue(sourceQuery, "sex")
	if !ok {
		return nil, false
	}
	row, found, err := classicdata.FindFashionAppearanceByNameAndSex(strings.TrimSpace(item.Name), sex)
	if err != nil || !found {
		return nil, false
	}
	rawParams := row["source_params"]
	params := make([]roleItemAppearanceSourceParamPair, 0, strings.Count(rawParams, "&")+1)
	for _, rawParam := range strings.Split(rawParams, "&") {
		parts := strings.SplitN(rawParam, "=", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			continue
		}
		params = append(params, roleItemAppearanceSourceParamPair{key: parts[0], value: parts[1]})
	}
	return params, len(params) > 0
}

func sourceQueryParamValue(sourceQuery string, key string) (string, bool) {
	queryStart := strings.Index(sourceQuery, "?")
	if queryStart < 0 {
		return "", false
	}
	for _, rawParam := range strings.Split(sourceQuery[queryStart+1:], "&") {
		parts := strings.SplitN(rawParam, "=", 2)
		if len(parts) == 2 && parts[0] == key && parts[1] != "" {
			return parts[1], true
		}
	}
	return "", false
}
