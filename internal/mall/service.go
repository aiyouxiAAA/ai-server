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
	DevCurrencyName       = "银元宝"
	PageSize              = 9
)

type Category struct {
	CategoryID string `json:"categoryId"`
	Name       string `json:"name"`
}

type Product struct {
	ProductID   string `json:"productId"`
	CategoryID  string `json:"categoryId"`
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	Price       int    `json:"price"`
	Currency    string `json:"currency"`
	Description string `json:"description"`
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
	Success         bool   `json:"success"`
	ProductID       string `json:"productId,omitempty"`
	Quantity        int    `json:"quantity,omitempty"`
	RequestID       string `json:"requestId,omitempty"`
	CurrencyName    string `json:"currencyName,omitempty"`
	CurrencyBalance int    `json:"currencyBalance,omitempty"`
	ErrorCode       string `json:"errorCode,omitempty"`
	ErrorMessage    string `json:"errorMessage,omitempty"`
}

type Service struct {
	categories []Category
	products   []Product
}

func NewService() *Service {
	return &Service{
		categories: []Category{
			{CategoryID: "hot", Name: "热卖"},
			{CategoryID: "medicine", Name: "丹药"},
			{CategoryID: "mount", Name: "坐骑"},
		},
		products: []Product{
			{ProductID: "dev-ginseng", CategoryID: "medicine", Name: "L百年人参果", Icon: "14.png", Price: 1, Currency: DevCurrencyName, Description: "开发期商城丹药。"},
			{ProductID: "dev-peach", CategoryID: "medicine", Name: "L百年蟠桃", Icon: "15.png", Price: 1, Currency: DevCurrencyName, Description: "开发期商城丹药。"},
			{ProductID: "dev-mount", CategoryID: "mount", Name: "狰狞神骑", Icon: "23.png", Price: 2, Currency: DevCurrencyName, Description: "限制装备至【坐骑】格。"},
			{ProductID: "dev-pack-01", CategoryID: "hot", Name: "行囊补给", Icon: "39.png", Price: 1, Currency: DevCurrencyName, Description: "开发期补给道具。"},
			{ProductID: "dev-pack-02", CategoryID: "hot", Name: "修行补给", Icon: "163.png", Price: 1, Currency: DevCurrencyName, Description: "开发期修行道具。"},
			{ProductID: "dev-pack-03", CategoryID: "hot", Name: "江湖补给", Icon: "426.png", Price: 1, Currency: DevCurrencyName, Description: "开发期江湖道具。"},
			{ProductID: "dev-pack-04", CategoryID: "hot", Name: "云隐补给", Icon: "7.png", Price: 1, Currency: DevCurrencyName, Description: "开发期云隐道具。"},
			{ProductID: "dev-pack-05", CategoryID: "hot", Name: "侠客补给", Icon: "29.png", Price: 1, Currency: DevCurrencyName, Description: "开发期侠客道具。"},
			{ProductID: "dev-pack-06", CategoryID: "hot", Name: "奇珍补给", Icon: "30.png", Price: 1, Currency: DevCurrencyName, Description: "开发期奇珍道具。"},
			{ProductID: "dev-pack-07", CategoryID: "hot", Name: "秘宝补给", Icon: "31.png", Price: 1, Currency: DevCurrencyName, Description: "开发期秘宝道具。"},
		},
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
	keyword := strings.ToLower(strings.TrimSpace(request.Keyword))
	products := make([]Product, 0, len(service.products))
	for _, product := range service.products {
		if request.DevCurrencyOnly && product.Currency != DevCurrencyName {
			continue
		}
		if categoryID != "" && categoryID != "all" && product.CategoryID != categoryID {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(product.Name), keyword) {
			continue
		}
		products = append(products, product)
	}
	sort.Slice(products, func(left int, right int) bool {
		if request.OrderType == "price_desc" {
			return products[left].Price > products[right].Price
		}
		if products[left].Price != products[right].Price {
			return products[left].Price < products[right].Price
		}
		return products[left].Name < products[right].Name
	})
	return products
}
