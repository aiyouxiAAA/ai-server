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
	CategoryID string `json:"categoryId"`
	Name       string `json:"name"`
}

type Product struct {
	ProductID   string        `json:"productId"`
	CategoryID  string        `json:"categoryId"`
	Name        string        `json:"name"`
	Icon        string        `json:"icon"`
	Price       int           `json:"price"`
	CouponPrice int           `json:"couponPrice,omitempty"`
	Discount    float64       `json:"discount,omitempty"`
	Recommended bool          `json:"recommended,omitempty"`
	Currency    string        `json:"currency"`
	Description string        `json:"description"`
	Items       []ProductItem `json:"items,omitempty"`
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
		// Category labels follow the source rmb_shop left tree shell from original UI.
		// Full original TypesMarket tree payload remains PARTIAL without capture rows.
		categories: []Category{
			{CategoryID: "1", Name: "推荐商品"},
			{CategoryID: "all", Name: "全部"},
			{CategoryID: "consumable", Name: "消耗品"},
			{CategoryID: "exp-card", Name: "经验卡"},
			{CategoryID: "fruit-card", Name: "仙果卡"},
			{CategoryID: "mount", Name: "坐骑"},
			{CategoryID: "equip", Name: "装备"},
			{CategoryID: "gem", Name: "宝石"},
			{CategoryID: "pass", Name: "通行证"},
			{CategoryID: "fashion", Name: "时装"},
		},
		products: append(capturedRecommendedProducts(), []Product{
			{ProductID: "dev-ginseng", CategoryID: "consumable", Name: "L百年人参果", Icon: "14.png", Price: 1, Currency: SourceYubiCurrencyName, Description: "开发期商城丹药。"},
			{ProductID: "dev-peach", CategoryID: "consumable", Name: "L百年蟠桃", Icon: "15.png", Price: 1, Currency: SourceYubiCurrencyName, Description: "开发期商城丹药。"},
			{ProductID: "dev-mount", CategoryID: "mount", Name: "狰狞神骑", Icon: "23.png", Price: 2, Currency: SourceYubiCurrencyName, Description: "限制装备至【坐骑】格。"},
			{ProductID: "8019", CategoryID: "fashion", Name: "超时空要塞", Icon: "1205.png", Price: 150, Currency: SourceYubiCurrencyName, Description: "f_i_超时空要塞^00ccff&24@幻·时装&25@1&15@20&16@20&19@【注：7天时限】&20@交杂着爱与友情以及惑星之命运的超银河Love Story！！！ &27@sitem_ezhj&103@0&104@1779629952719&105@&107@&108@0", Items: []ProductItem{{Name: "超时空要塞", Display: "1205.png", Description: "f_i_超时空要塞^00ccff&24@幻·时装&25@1&15@20&16@20&19@【注：7天时限】&20@交杂着爱与友情以及惑星之命运的超银河Love Story！！！ &27@sitem_ezhj&103@0&104@1779629952719&105@&107@&108@0", Count: 1}}},
			{ProductID: "7033", CategoryID: "fashion", Name: "盛夏缤纷", Icon: "729.png", Price: 100, Currency: SourceYubiCurrencyName, Description: "f_i_盛夏缤纷^5BC46D&24@幻·时装&25@1&19@男性：黑背心时尚牛仔裤。\r女性：彩虹肩带短裙。&20@盛夏时尚服饰系列之一。&27@sitem_ezhj&103@0&104@0&105@&107@&108@0", Items: []ProductItem{{Name: "盛夏缤纷", Display: "729.png", Description: "f_i_盛夏缤纷^5BC46D&24@幻·时装&25@1&19@男性：黑背心时尚牛仔裤。\r女性：彩虹肩带短裙。&20@盛夏时尚服饰系列之一。&27@sitem_ezhj&103@0&104@0&105@&107@&108@0", Count: 1}}},
		}...),
	}
}

// capturedRecommendedProducts keeps the first rmb_shop page in the exact packet order.
func capturedRecommendedProducts() []Product {
	return []Product{
		capturedRecommendedProduct("9011", "高级聚气碟.特x5", "高级聚气碟.特", "1724.png", 375, 0, 5, "f_i_高级聚气碟.特^f9e000&24@材料&25@99&20@炼制【上品纳元丹.特】(80级及以上适用)的容器，可抽取【四象镜】内灵气转化为提升修为的丹药。<br/><font color='#ffff00'>双击提取1,000,000经验值(增益50%)</font>&27@sitem_jhj&103@0&104@0&105@&107@&108@0"),
		capturedRecommendedProduct("9010", "中级聚气碟.特x5", "中级聚气碟.特", "1723.png", 225, 0, 5, "f_i_中级聚气碟.特^f9e000&24@材料&25@99&20@炼制【中品纳元丹.特】(70级及以上适用)的容器，可抽取【四象镜】内灵气转化为提升修为的丹药。<br/><font color='#ffff00'>双击提取200,000经验值(增益50%)</font>&27@sitem_jhj&103@0&104@0&105@&107@&108@0"),
		capturedRecommendedProduct("9009", "初级聚气碟.特x5", "初级聚气碟.特", "1722.png", 150, 0, 5, "f_i_初级聚气碟.特^f9e000&24@材料&25@99&20@炼制【下品纳元丹.特】(60级及以上适用)的容器，可抽取【四象镜】内灵气转化为提升修为的丹药。<br/><font color='#ffff00'>双击提取100,000经验值(增益50%)</font>&27@sitem_jhj&103@0&104@0&105@&107@&108@0"),
		capturedRecommendedProduct("5024", "精炼宝石", "精炼宝石", "435.png", 50, 60, 1, "f_i_精炼宝石^f9e000&24@宝物&25@999&19@精炼+3或以上装备成功率高，精炼失败后等级不会降至+3以下。\r<font color='#fee010'>注：只适用于精炼等级3及以上装备。\r双击使用该物品</font>&20@神奇的天然宝石，是精炼装备的宝物，用于提升武器,护具以及首饰属性。&27@sitem_zgbs&103@0&104@0&105@&107@&108@160"),
		capturedRecommendedProduct("5021", "壹级原石", "壹级原石", "771.png", 50, 0, 1, "f_i_壹级原石^f9e000&24@宝物&25@99&19@打碎石块可获得一颗壹级镶嵌宝石。\r<font color='#fee010'>双击使用该物品</font>&20@一块奇特的石头,看似很蹊跷。&27@sitem_jhj&103@0&104@0&105@&107@&108@0"),
		capturedRecommendedProduct("3001", "云生兽", "云生兽", "1590.png", 998, 0, 1, "f_i_云生兽^C156C7&23@限制装备至【坐骑】格。&24@坐骑&25@1&21@1&12@15&19@装备后不遇敌（明怪除外）。<br/>装备后可任意传送地图。<br/>永久坐骑。&20@拥有强大的驱妖之气，野外行动可使隐藏的妖物无法近身，乃冒险必备坐骑,装备后自动绑定。&27@sitem_piput&103@0&104@0&105@&107@&108@0"),
		capturedRecommendedProduct("1000", "高阶经验卡", "高阶经验卡", "578.png", 45, 0, 1, "f_i_高阶双倍经验卡^00ccff&24@特殊&25@99&19@双击使用。击杀怪物后，将获得双倍经验。效果持续时间为6小时。&20@可以获得经验加成的神奇商城道具。&27@sitem_book&103@0&104@0&105@&107@&108@0"),
		capturedRecommendedProduct("27", "VIP季卡", "VIP季卡", "1332.png", 600, 0, 1, "f_i_VIP季卡^C156C7&24@宝物&25@99&19@有效期3个月,获得以下特权:&20@打怪可以获得2倍经验;\r提升10%物理攻击和魔法攻击;\r提升10%物理和魔法防御;\r可以向VIP大使领取一次VIP高级新手套装;\r装备精炼成功率提高7%;\r装备凿孔成功率提高7%;\r每天可以向VIP大使领取180个L小喇叭;\r每天可以向VIP大使领取8个吉祥袋;\r每天可以向VIP大使领取10个宝匣;\r每天可以向VIP大使领取5个L魔匣;\r每天可向VIP大使领取8颗L初级精炼宝石;\r每天可向VIP大使领取5颗L精炼宝石;\r每天可以向VIP大使领取1个仙匣.\r免费享受找回两件不慎丢弃装备的特权.\r(找回的装备仅限无任何属性的空装备，VIP时间不可叠加，只会覆盖，谨慎使用)&27@sitem_jhj&103@0&104@0&105@&107@&108@0"),
		capturedRecommendedProduct("26", "吉祥袋", "吉祥袋", "959.png", 4, 0, 1, "f_i_吉祥袋^f9e000&24@宝物&25@999&19@双击打开，将有可能获得：\r<font color='#00ccff'>奥义秘诀、黑骏马、精炼宝石、凿孔器、经验丹（提升角色经验）、属性药水等超值道具。</font>&20@代表吉祥的神奇袋子的，传说是上人们用于存放物件所制。&103@0&104@0&105@&107@&108@0"),
	}
}

func capturedRecommendedProduct(productID string, name string, itemName string, icon string, price int, couponPrice int, count int, description string) Product {
	return Product{
		ProductID:   productID,
		CategoryID:  "1",
		Name:        name,
		Icon:        icon,
		Price:       price,
		CouponPrice: couponPrice,
		Discount:    1,
		Recommended: true,
		Currency:    SourceYubiCurrencyName,
		Description: description,
		Items: []ProductItem{{
			Name:        itemName,
			Display:     icon,
			Description: description,
			Count:       count,
		}},
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
		// The source label for selectDQMode is "是否列出点券商品". The wire field
		// keeps its legacy name, but now filters by the captured dq_price value.
		if request.DevCurrencyOnly && product.CouponPrice <= 0 {
			continue
		}
		// category "1" is the source-like 推荐商品 entry used by rmb_shop open requests.
		if categoryID != "" && categoryID != "all" {
			if categoryID == "1" {
				if product.CategoryID != "1" && product.CategoryID != "hot" {
					continue
				}
			} else if product.CategoryID != categoryID {
				continue
			}
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
	}
	return products
}
