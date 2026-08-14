package session

import "strings"

// RoleItemAcquisitionSource identifies the gameplay or operational path that
// caused an item to enter a role's inventory.
type RoleItemAcquisitionSource struct {
	Kind   string
	Detail string
}

// RoleItemAcquisition is an append-only audit record. Quantity is the amount
// granted by that event, never the post-merge stack total.
type RoleItemAcquisition struct {
	ID               int64  `json:"id"`
	OccurredAtUnixMs int64  `json:"occurredAtUnixMs"`
	PlayerID         string `json:"playerId"`
	RoleID           string `json:"roleId"`
	ItemName         string `json:"itemName"`
	ItemType         string `json:"itemType"`
	ContainerType    string `json:"containerType"`
	ItemDisplay      string `json:"itemDisplay"`
	ItemDescription  string `json:"itemDescription,omitempty"`
	ItemLevel        int    `json:"itemLevel"`
	Quantity         int    `json:"quantity"`
	Source           string `json:"source"`
	SourceDetail     string `json:"sourceDetail,omitempty"`
}

func normalizeRoleItemAcquisitionSource(source RoleItemAcquisitionSource) RoleItemAcquisitionSource {
	source.Kind = strings.TrimSpace(source.Kind)
	source.Detail = strings.TrimSpace(source.Detail)
	if source.Kind == "" {
		source.Kind = "系统发放"
	}
	return source
}

func roleItemAcquisitionFromItem(playerID string, roleID string, item RoleItem, source RoleItemAcquisitionSource) RoleItemAcquisition {
	item = normalizeRoleItem(item)
	if item.Count <= 0 {
		item.Count = 1
	}
	source = normalizeRoleItemAcquisitionSource(source)
	return RoleItemAcquisition{
		PlayerID:        playerID,
		RoleID:          roleID,
		ItemName:        item.Name,
		ItemType:        item.ItemType,
		ContainerType:   item.Type,
		ItemDisplay:     item.Display,
		ItemDescription: item.Description,
		ItemLevel:       item.ItemLevel,
		Quantity:        item.Count,
		Source:          source.Kind,
		SourceDetail:    source.Detail,
	}
}

func roleItemAcquisitionsForRewards(
	playerID string,
	roleID string,
	rewards []RoleItem,
	source RoleItemAcquisitionSource,
	occurredAtUnixMs int64,
) []RoleItemAcquisition {
	acquisitions := make([]RoleItemAcquisition, 0, len(rewards))
	for _, reward := range rewards {
		acquisition := roleItemAcquisitionFromItem(playerID, roleID, reward, source)
		acquisition.OccurredAtUnixMs = occurredAtUnixMs
		acquisitions = append(acquisitions, acquisition)
	}
	return acquisitions
}
