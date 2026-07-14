package mall

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed captured_catalog.json
var capturedCatalogJSON []byte

type capturedCatalog struct {
	SchemaVersion int                  `json:"schemaVersion"`
	Capture       capturedCatalogMeta  `json:"capture"`
	Categories    []Category           `json:"categories"`
	Products      []capturedProductRow `json:"products"`
}

type capturedCatalogMeta struct {
	ResponseCount      int `json:"responseCount"`
	UniqueProductCount int `json:"uniqueProductCount"`
}

type capturedProductRow struct {
	ProductID     string         `json:"productId"`
	CategoryID    string         `json:"categoryId"`
	CategoryIDs   []string       `json:"categoryIds"`
	CategoryOrder map[string]int `json:"categoryOrder"`
	Name          string         `json:"name"`
	Icon          string         `json:"icon"`
	Price         int            `json:"price"`
	CouponPrice   int            `json:"couponPrice"`
	Discount      float64        `json:"discount"`
	Recommended   bool           `json:"recommended"`
	Currency      string         `json:"currency"`
	Description   string         `json:"description"`
	Items         []ProductItem  `json:"items"`
}

func (row capturedProductRow) product() Product {
	return Product{
		ProductID:     row.ProductID,
		CategoryID:    row.CategoryID,
		CategoryIDs:   append([]string{}, row.CategoryIDs...),
		CategoryOrder: row.CategoryOrder,
		Name:          row.Name,
		Icon:          row.Icon,
		Price:         row.Price,
		CouponPrice:   row.CouponPrice,
		Discount:      row.Discount,
		Recommended:   row.Recommended,
		Currency:      row.Currency,
		Description:   row.Description,
		Items:         append([]ProductItem{}, row.Items...),
	}
}

func loadCapturedCatalog() ([]Category, []Product) {
	var catalog capturedCatalog
	if err := json.Unmarshal(capturedCatalogJSON, &catalog); err != nil {
		panic(fmt.Sprintf("decode embedded captured mall catalog: %v", err))
	}
	if catalog.SchemaVersion != 1 || len(catalog.Categories) == 0 || len(catalog.Products) == 0 {
		panic("embedded captured mall catalog is incomplete")
	}
	products := make([]Product, 0, len(catalog.Products))
	for _, row := range catalog.Products {
		if row.ProductID == "" || row.Name == "" || len(row.Items) == 0 || len(row.CategoryIDs) == 0 {
			panic(fmt.Sprintf("embedded captured mall catalog has invalid product %q", row.ProductID))
		}
		products = append(products, row.product())
	}
	return append([]Category{}, catalog.Categories...), products
}
