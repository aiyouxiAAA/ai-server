package battle

// RewardItem records this battle's quantity, not the resulting bag stack total.
// Destination is assigned by the inventory owner after authoritative delivery.
type RewardItem struct {
	Name        string `json:"name"`
	Display     string `json:"display"`
	Count       int    `json:"count"`
	ItemLevel   int    `json:"itemLevel"`
	Destination string `json:"destination"`
}
