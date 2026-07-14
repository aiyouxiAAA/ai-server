package mall

import (
	"sort"
	"strings"
)

const (
	PAYMENT_DISABLED      = "PAYMENT_DISABLED"
	PRODUCT_NOT_FOUND     = "PRODUCT_NOT_FOUND"
	INSUFFICIENT_CURRENCY = "INSUFFICIENT_CURRENCY"
	INVALID_QUANTITY      = "INVALID_QUANTITY"
	DUPLICATE_REQUEST     = "DUPLICATE_REQUEST"
	// DevCurrencyName is kept for compatibility with older mall seed rows and tests.
	// Source rmb_shop displays / spends 玉币 (proxy_user_money), not 银元宝.
	DevCurrencyName        = "银元宝"
	SourceYubiCurrencyName = "玉币"
	PageSize               = 9
)

type Category struct {
	CategoryID string     `json:"categoryId"`
	Name       string     `json:"name"`
	Children   []Category `json:"childs,omitempty"`
}

type Product struct {
	ProductID   string   `json:"productId"`
	CategoryID  string   `json:"categoryId"`
	CategoryIDs []string `json:"-"`
	// CategoryOrder is capture-only ordering metadata for source category pages.
	CategoryOrder map[string]int `json:"-"`
	Name          string         `json:"name"`
	Icon          string         `json:"icon"`
	Price         int            `json:"price"`
	CouponPrice   int            `json:"couponPrice,omitempty"`
	Discount      float64        `json:"discount,omitempty"`
	Recommended   bool           `json:"recommended,omitempty"`
	Currency      string         `json:"currency"`
	Description   string         `json:"description"`
	Items         []ProductItem  `json:"items,omitempty"`
}

type ProductItem struct {
	Name        string `json:"name"`
	Display     string `json:"display"`
	Description string `json:"description"`
	Count       int    `json:"count,omitempty"`
}

type SearchRequest struct {
	CategoryID      string `json:"categoryId,omitempty"`
	Keyword         string `json:"keyword,omitempty"`
	OrderType       string `json:"orderType,omitempty"`
	DevCurrencyOnly bool   `json:"devCurrencyOnly"`
	Offset          int    `json:"offset"`
	Limit           int    `json:"limit"`
}

type SearchCountPush struct {
	CategoryID string `json:"categoryId,omitempty"`
	Keyword    string `json:"keyword,omitempty"`
	Count      int    `json:"count"`
}

type SearchPagePush struct {
	CategoryID string    `json:"categoryId,omitempty"`
	Keyword    string    `json:"keyword,omitempty"`
	Offset     int       `json:"offset"`
	Limit      int       `json:"limit"`
	Products   []Product `json:"products"`
}

type CurrencyPush struct {
	CurrencyName string `json:"currencyName"`
	Balance      int    `json:"balance"`
}

type PurchaseRequest struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
	RequestID string `json:"requestId"`
}

type PurchaseResult struct {
	Success         bool      `json:"success"`
	ProductID       string    `json:"productId,omitempty"`
	Quantity        int       `json:"quantity,omitempty"`
	RequestID       string    `json:"requestId,omitempty"`
	CurrencyName    string    `json:"currencyName,omitempty"`
	CurrencyBalance int       `json:"currencyBalance,omitempty"`
	Delivery        *Delivery `json:"delivery,omitempty"`
	ErrorCode       string    `json:"errorCode,omitempty"`
	ErrorMessage    string    `json:"errorMessage,omitempty"`
}

// Delivery identifies the authoritative container slot filled by a successful
// local mall purchase. The corresponding item state travels on the existing
// ItemInfo push contract.
type Delivery struct {
	ContainerType string `json:"containerType"`
	ItemIndex     int    `json:"itemIndex"`
	ItemName      string `json:"itemName"`
	ItemCount     int    `json:"itemCount"`
}

type Service struct {
	categories []Category
	products   []Product
}

func NewService() *Service {
	categories, capturedProducts := loadCapturedCatalog()
	return &Service{
		categories: categories,
		products:   append(capturedProducts, developmentProducts()...),
	}
}

func developmentProducts() []Product {
	return []Product{
		{ProductID: "dev-ginseng", CategoryID: "3", Name: "L百年人参果", Icon: "14.png", Price: 1, Currency: SourceYubiCurrencyName, Description: "开发期商城丹药。"},
		{ProductID: "dev-peach", CategoryID: "3", Name: "L百年蟠桃", Icon: "15.png", Price: 1, Currency: SourceYubiCurrencyName, Description: "开发期商城丹药。"},
		{ProductID: "dev-mount", CategoryID: "11", Name: "狰狞神骑", Icon: "23.png", Price: 2, Currency: SourceYubiCurrencyName, Description: "限制装备至【坐骑】格。"},
	}
}

func (service *Service) Categories() []Category {
	return append([]Category{}, service.categories...)
}

func (service *Service) SearchCount(request SearchRequest) SearchCountPush {
	products := service.search(request)
	return SearchCountPush{
		CategoryID: strings.TrimSpace(request.CategoryID),
		Keyword:    strings.TrimSpace(request.Keyword),
		Count:      len(products),
	}
}

func (service *Service) SearchPage(request SearchRequest) SearchPagePush {
	products := service.search(request)
	offset := request.Offset
	if offset < 0 {
		offset = 0
	}
	limit := request.Limit
	if limit <= 0 || limit > PageSize {
		limit = PageSize
	}
	if offset > len(products) {
		offset = len(products)
	}
	end := offset + limit
	if end > len(products) {
		end = len(products)
	}
	return SearchPagePush{
		CategoryID: strings.TrimSpace(request.CategoryID),
		Keyword:    strings.TrimSpace(request.Keyword),
		Offset:     offset,
		Limit:      limit,
		Products:   append([]Product{}, products[offset:end]...),
	}
}

func (service *Service) FindProduct(productID string) (Product, bool) {
	productID = strings.TrimSpace(productID)
	for _, product := range service.products {
		if product.ProductID == productID {
			return product, true
		}
	}
	return Product{}, false
}

func (service *Service) PaymentDisabled(requestID string) PurchaseResult {
	return PurchaseResult{
		Success:      false,
		RequestID:    strings.TrimSpace(requestID),
		ErrorCode:    PAYMENT_DISABLED,
		ErrorMessage: "真实支付暂未开放。",
	}
}

func (service *Service) search(request SearchRequest) []Product {
	categoryID := strings.TrimSpace(request.CategoryID)
	if categoryID == "all" {
		categoryID = "2"
	}
	keyword := strings.ToLower(strings.TrimSpace(request.Keyword))
	products := make([]Product, 0, len(service.products))
	for _, product := range service.products {
		// The source label for selectDQMode is "是否列出点券商品". The wire field
		// keeps its legacy name, but now filters by the captured dq_price value.
		if request.DevCurrencyOnly && product.CouponPrice <= 0 {
			continue
		}
		// category "1" is the source-like 推荐商品 entry used by rmb_shop open requests.
		if categoryID != "" && !matchesCapturedCategory(categoryID, product) {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(product.Name), keyword) {
			continue
		}
		products = append(products, product)
	}
	if request.OrderType == "price_asc" || request.OrderType == "price_desc" {
		sort.SliceStable(products, func(left int, right int) bool {
			if products[left].Price == products[right].Price {
				return false
			}
			if request.OrderType == "price_desc" {
				return products[left].Price > products[right].Price
			}
			return products[left].Price < products[right].Price
		})
	} else if categoryID != "2" {
		sort.SliceStable(products, func(left int, right int) bool {
			leftOrder, leftFound := products[left].CategoryOrder[categoryID]
			rightOrder, rightFound := products[right].CategoryOrder[categoryID]
			if leftFound && rightFound {
				return leftOrder < rightOrder
			}
			return leftFound && !rightFound
		})
	}
	return products
}

func matchesCapturedCategory(selectedCategoryID string, product Product) bool {
	if selectedCategoryID == "2" {
		return true
	}
	categoryIDs := product.CategoryIDs
	if len(categoryIDs) == 0 {
		categoryIDs = []string{product.CategoryID}
	}
	if selectedCategoryID == "3" {
		return containsCategoryID(categoryIDs, "3", "4", "5", "6", "7", "8")
	}
	if selectedCategoryID == "15" {
		return containsCategoryID(categoryIDs, "15", "16", "17")
	}
	return containsCategoryID(categoryIDs, selectedCategoryID)
}

func containsCategoryID(categoryIDs []string, expected ...string) bool {
	for _, categoryID := range categoryIDs {
		for _, candidate := range expected {
			if categoryID == candidate {
				return true
			}
		}
	}
	return false
}
