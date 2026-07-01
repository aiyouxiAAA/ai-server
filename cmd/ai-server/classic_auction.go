package main

import (
	"log"
	"strings"
)

const (
	classicAuctionContainerType = "拍卖行"
	classicAuctionSourceTotal   = 201
	classicAuctionSourceCapture = "20260612_211747_199_session_15948/20260612_211850_206_conn_0002#GetAuctionInfoList+c_auctionInfoCount+c_AuctionInfo"
	classicAuctionAddVipCapture = "20260619_190231_297_session_13232/20260619_190236_818_conn_0002#AddAuctionInfo+c_Error"
	classicAuctionVipError      = "交易需要VIP5"
)

var classicAuctionNPCHandles = map[string]bool{
	"4180542615109515": true,
	"5010542616817526": true,
}

type classicAuctionOpenPush struct {
	ContainerType string `json:"containerType"`
	NPCHandle     string `json:"npcHandle,omitempty"`
}

type classicAuctionListRequest struct {
	Page        int    `json:"page"`
	PageCount   int    `json:"pageCount"`
	AuctionType string `json:"auctionType,omitempty"`
	Name        string `json:"name,omitempty"`
	Seller      string `json:"seller,omitempty"`
	MinLv       int    `json:"minLv,omitempty"`
	MaxLv       int    `json:"maxLv,omitempty"`
	SortType    int    `json:"sortType,omitempty"`
}

type classicAuctionAddPriceItemRequest struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type classicAuctionAddRequest struct {
	SaleType   int                                 `json:"saleType"`
	ItemType   string                              `json:"itemType"`
	ItemIndex  int                                 `json:"itemIndex"`
	ItemCount  int                                 `json:"itemCount"`
	PriceItems []classicAuctionAddPriceItemRequest `json:"priceItems,omitempty"`
}

type classicAuctionItemPush struct {
	Name        string `json:"name"`
	ItemType    string `json:"itemType"`
	Display     string `json:"display"`
	Description string `json:"description"`
	Count       int    `json:"count"`
	ItemLevel   int    `json:"itemLevel"`
}

type classicAuctionInfoPush struct {
	Handle     string                   `json:"handle"`
	Index      int                      `json:"index"`
	Seller     string                   `json:"seller"`
	StartTime  int64                    `json:"startTime"`
	EndTime    int64                    `json:"endTime"`
	Item       classicAuctionItemPush   `json:"item"`
	PriceItems []classicAuctionItemPush `json:"priceItems"`
}

type classicAuctionListPush struct {
	Page          int                      `json:"page"`
	PageCount     int                      `json:"pageCount"`
	Total         int                      `json:"total"`
	AuctionType   string                   `json:"auctionType"`
	Name          string                   `json:"name"`
	Seller        string                   `json:"seller"`
	SortType      int                      `json:"sortType"`
	Items         []classicAuctionInfoPush `json:"items"`
	SourceCapture string                   `json:"sourceCapture"`
	Partial       bool                     `json:"partial"`
}

type classicAuctionSourceRow struct {
	Handle       string
	Index        int
	Seller       string
	StartTime    int64
	EndTime      int64
	ItemName     string
	ItemType     string
	Display      string
	Count        int
	PriceName    string
	PriceType    string
	PriceDisplay string
	PriceCount   int
}

func buildClassicAuctionOpenResult(request classicTownRoleInteractionRequest) (packetResult, bool) {
	handle := strings.TrimSpace(request.Handle)
	if !classicAuctionNPCHandles[handle] {
		return packetResult{}, false
	}
	return packetResult{
		auctionOpen: &classicAuctionOpenPush{
			ContainerType: classicAuctionContainerType,
			NPCHandle:     handle,
		},
		handled: true,
	}, true
}

func buildClassicAuctionListResult(request classicAuctionListRequest) packetResult {
	pageCount := request.PageCount
	if pageCount <= 0 || pageCount > 30 {
		pageCount = 30
	}
	page := request.Page
	if page < 0 {
		page = 0
	}
	filtered := filterClassicAuctionRows(request)
	total := len(filtered)
	if strings.TrimSpace(request.Name) == "" && strings.TrimSpace(request.Seller) == "" {
		total = classicAuctionSourceTotal
	}
	start := page * pageCount
	end := start + pageCount
	items := []classicAuctionInfoPush{}
	if start < len(filtered) {
		if end > len(filtered) {
			end = len(filtered)
		}
		items = make([]classicAuctionInfoPush, 0, end-start)
		for _, row := range filtered[start:end] {
			items = append(items, classicAuctionSourceRowToPush(row))
		}
	}
	log.Printf("[ai-server] classic auction list page=%d pageCount=%d auctionType=%s name=%s seller=%s items=%d partial=true", page, pageCount, request.AuctionType, request.Name, request.Seller, len(items))
	return packetResult{
		auctionList: &classicAuctionListPush{
			Page:          page,
			PageCount:     pageCount,
			Total:         total,
			AuctionType:   request.AuctionType,
			Name:          request.Name,
			Seller:        request.Seller,
			SortType:      request.SortType,
			Items:         items,
			SourceCapture: classicAuctionSourceCapture,
			Partial:       true,
		},
		handled: true,
	}
}

func buildClassicAuctionAddResult(_ classicAuctionAddRequest) packetResult {
	log.Printf("[ai-server] classic auction add rejected sourceCapture=%s error=%s", classicAuctionAddVipCapture, classicAuctionVipError)
	return packetResult{
		chatMessages: []classicTownChatMessagePush{classicTownSystemWarningMessage(classicAuctionVipError)},
		handled:      true,
	}
}

func filterClassicAuctionRows(request classicAuctionListRequest) []classicAuctionSourceRow {
	name := strings.TrimSpace(request.Name)
	seller := strings.TrimSpace(request.Seller)
	rows := make([]classicAuctionSourceRow, 0, len(classicAuctionSourceRows))
	for _, row := range classicAuctionSourceRows {
		if name != "" && !strings.Contains(row.ItemName, name) {
			continue
		}
		if seller != "" && row.Seller != seller {
			continue
		}
		rows = append(rows, row)
	}
	return rows
}

func classicAuctionSourceRowToPush(row classicAuctionSourceRow) classicAuctionInfoPush {
	itemMeta := classicAuctionSourceItemMeta(row.ItemName, row.Display)
	priceMeta := classicAuctionSourceItemMeta(row.PriceName, row.PriceDisplay)
	return classicAuctionInfoPush{
		Handle:    row.Handle,
		Index:     row.Index,
		Seller:    row.Seller,
		StartTime: row.StartTime,
		EndTime:   row.EndTime,
		Item: classicAuctionItemPush{
			Name:        row.ItemName,
			ItemType:    row.ItemType,
			Display:     row.Display,
			Description: itemMeta.Description,
			Count:       row.Count,
			ItemLevel:   itemMeta.ItemLevel,
		},
		PriceItems: []classicAuctionItemPush{
			{
				Name:        row.PriceName,
				ItemType:    row.PriceType,
				Display:     row.PriceDisplay,
				Description: priceMeta.Description,
				Count:       row.PriceCount,
				ItemLevel:   priceMeta.ItemLevel,
			},
		},
	}
}

type classicAuctionItemMeta struct {
	Description string
	ItemLevel   int
}

func classicAuctionItemMetaKey(name string, display string) string {
	return name + "|" + display
}

func classicAuctionSourceItemMeta(name string, display string) classicAuctionItemMeta {
	return classicAuctionSourceItemMetas[classicAuctionItemMetaKey(name, display)]
}

var classicAuctionSourceItemMetas = map[string]classicAuctionItemMeta{
	classicAuctionItemMetaKey("大包还元散", "700.png"):   {Description: "f_i_大包还元散^00ccff&24@消耗品&25@999&7@2000&20@由高人炼制的小包药物散剂,使用之后能回复气力.&27@sitem_book&103@0&104@0&105@&107@&108@0", ItemLevel: 3},
	classicAuctionItemMetaKey("银元宝", "39.png"):      {Description: "f_i_银元宝^C156C7&24@材料 消耗品&25@9999&19@双击可兑换为1000铜币&20@游戏中的货币,用于流通买卖&27@sitem_jhj&103@0&104@0&105@&107@&108@0", ItemLevel: 4},
	classicAuctionItemMetaKey("装备重置符", "777.png"):   {Description: "f_i_装备重置符^f9e000&24@宝物&25@99&19@可将已经获得的装备重新生成，有可能出现带孔装备同时可以解除装备绑定。\r<font color='#ff0000'>装备重置后精炼等级和装备孔数，以及已镶嵌的宝石都会消失。</font>\r<font color='#fee010'>双击使用该物品</font>&20@用于武器装备的古老咒符。&27@sitem_book&103@0&104@0&105@&107@&108@0", ItemLevel: 5},
	classicAuctionItemMetaKey("粉莲宝座", "1137.png"):   {Description: "f_i_粉莲宝座^f9e000&23@限制装备至【坐骑】格。&24@坐骑&25@1&21@35&4@300&12@20&27@sitem_pet&19@精炼潜质:\n[精炼+1] 每升一级 魔法防御+3\n[精炼+3] 每升一级 魔法防御+5\n[精炼+8] 每升一级 魔法防御+10\n[精炼+14] 每升一级 魔法防御+15&103@0&104@0&105@&107@&108@0", ItemLevel: 3},
	classicAuctionItemMetaKey("神马驼驼", "1136.png"):   {Description: "f_i_神马驼驼^f9e000&23@限制装备至【坐骑】格。&24@坐骑&25@1&21@35&3@300<$jpdef>&12@20&27@sitem_pet&19@精炼潜质:\n[精炼+1] 每升一级 物理防御+3\n[精炼+3] 每升一级 物理防御+5\n[精炼+8] 每升一级 物理防御+10\n[精炼+14] 每升一级 物理防御+15&103@0&104@0&105@&107@&108@0", ItemLevel: 3},
	classicAuctionItemMetaKey("绝影飞禽", "1196.png"):   {Description: "f_i_绝影飞禽^f9e000&23@限制装备至【坐骑】格。&24@坐骑&25@1&21@40&5@300&12@20&27@sitem_pet&19@精炼潜质:\n[精炼+1] 每升一级 气力上限+5\n[精炼+3] 每升一级 气力上限+10\n[精炼+8] 每升一级 气力上限+30\n[精炼+14] 每升一级 气力上限+40&103@0&104@0&105@&107@&108@0", ItemLevel: 3},
	classicAuctionItemMetaKey("奇效宠物药剂", "932.png"):  {Description: "f_i_奇效宠物药剂^5BC46D&23@每次喂食 成长 +10 饱食 +2&24@消耗品&25@500&19@促进宠物急速成长的特效药剂。&20@一般在商城内可以购得。&27@sitem_piput&103@0&104@0&105@&107@&108@0", ItemLevel: 1},
	classicAuctionItemMetaKey("萌兔宝宝", "1188.png"):   {Description: "f_i_萌兔宝宝^f9e000&23@限制装备至【坐骑】格。&24@坐骑&25@1&21@40&2@300&12@20&27@sitem_pet&19@精炼潜质:\n[精炼+1] 每升一级 魔法攻击+3\n[精炼+3] 每升一级 魔法攻击+5\n[精炼+8] 每升一级 魔法攻击+10\n[精炼+14] 每升一级 魔法攻击+15&103@0&104@0&105@&107@&108@0", ItemLevel: 3},
	classicAuctionItemMetaKey("特级精炼宝石", "1145.png"): {Description: "f_i_特级精炼宝石^f9e000&24@宝物&25@99&19@精炼+14或以上装备成功率高，精炼失败后等级不会降至+14以下。\r注：适用于精炼等级14及以上装备。\r双击使用该物品</font>&20@罕见的天然宝石，精炼高级装备有奇效。&27@sitem_zgbs&103@0&104@0&105@&107@&108@0", ItemLevel: 5},
	classicAuctionItemMetaKey("魁拔幻化珠", "1340.png"):  {Description: "f_i_魁拔幻化珠^C156C7&24@幻·幻化珠&25@1&19@装备后幻化为猛将魁魃&20@拥有该类宝物便可随意变换为各种形态。&27@sitem_wood&103@0&104@0&105@&107@&108@0", ItemLevel: 4},
	classicAuctionItemMetaKey("筋斗云", "968.png"):     {Description: "f_i_筋斗云^00ccff&23@限制装备至【宝1、宝2、宝3、宝4】格。&24@法宝&25@1&12@10&15@10&19@装备后产生坐骑效果。&20@传说是孙悟空所骑乘之云，有日行千里的神奇效果。&27@sitem_ezhj&103@0&104@0&105@&107@&108@0", ItemLevel: 3},
	classicAuctionItemMetaKey("四象镜", "1710.png"):    {Description: "f_i_四象镜^f9e000&23@限制装备至【宝1、宝2、宝3、宝4】格。&24@法宝&25@1&21@1&5@&6@&20@内藏一方乾坤，能收纳灵气，佩戴后修炼有事半功倍之效。&27@sitem_jhj&19@\n<font color='#59c5ca'>法宝效果:转化经验\n装备后，将时间和金钱转化为经验。</font>\n已存储经验:0。\n&103@0&104@0&105@&107@&108@0", ItemLevel: 5},
	classicAuctionItemMetaKey("千年灵芝", "588.png"):    {Description: "f_i_千年灵芝^f9e000&24@特殊&25@999&19@如果放在背包里,在副本内死亡后可立即原地复活。&20@灵芝自古以来就被认为是吉祥,富贵,美好,长寿的象征,有 仙草 瑞草之称.民间传说灵芝有起死回生,长生不老之功效.&103@0&104@0&105@&107@&108@0", ItemLevel: 5},
	classicAuctionItemMetaKey("仙宝葫芦", "1138.png"):   {Description: "f_i_仙宝葫芦^f9e000&23@限制装备至【坐骑】格。&24@坐骑&25@1&21@40&1@300&12@20&27@sitem_pet&19@精炼潜质:\n[精炼+1] 每升一级 物理攻击+3\n[精炼+3] 每升一级 物理攻击+5\n[精炼+8] 每升一级 物理攻击+10\n[精炼+14] 每升一级 物理攻击+15&103@0&104@0&105@&107@&108@0", ItemLevel: 3},
	classicAuctionItemMetaKey("木葫芦", "192.png"):     {Description: "f_i_木葫芦^5BC46D&24@特殊&25@1&19@有3格容积的木葫芦,双击打开.&20@普通的木葫芦,可以放些物品在里面.&103@0&104@0&105@&107@&108@0", ItemLevel: 1},
}

var classicAuctionSourceRows = []classicAuctionSourceRow{
	{Handle: "8040284297490233", Index: 0, Seller: "风雨雷", StartTime: 1781284297490, EndTime: 1781543497490, ItemName: "大包还元散", ItemType: "own", Display: "700.png", Count: 999, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 700},
	{Handle: "6458284231251562", Index: 1, Seller: "风雨雷", StartTime: 1781284231251, EndTime: 1781543431251, ItemName: "装备重置符", ItemType: "oneI", Display: "777.png", Count: 99, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 200},
	{Handle: "4684284204612484", Index: 2, Seller: "风雨雷", StartTime: 1781284204612, EndTime: 1781543404612, ItemName: "粉莲宝座", ItemType: "equip", Display: "1137.png", Count: 1, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 300},
	{Handle: "4106284196486634", Index: 3, Seller: "风雨雷", StartTime: 1781284196486, EndTime: 1781543396486, ItemName: "粉莲宝座", ItemType: "equip", Display: "1137.png", Count: 1, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 300},
	{Handle: "2129284166802841", Index: 4, Seller: "风雨雷", StartTime: 1781284166802, EndTime: 1781543366802, ItemName: "神马驼驼", ItemType: "equip", Display: "1136.png", Count: 1, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 300},
	{Handle: "5670284144109835", Index: 5, Seller: "风雨雷", StartTime: 1781284144109, EndTime: 1781543344109, ItemName: "绝影飞禽", ItemType: "equip", Display: "1196.png", Count: 1, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 300},
	{Handle: "9968284135748198", Index: 6, Seller: "风雨雷", StartTime: 1781284135748, EndTime: 1781543335748, ItemName: "粉莲宝座", ItemType: "equip", Display: "1137.png", Count: 1, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 300},
	{Handle: "9656284130508745", Index: 7, Seller: "风雨雷", StartTime: 1781284130508, EndTime: 1781543330508, ItemName: "神马驼驼", ItemType: "equip", Display: "1136.png", Count: 1, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 300},
	{Handle: "8239284109936264", Index: 8, Seller: "风雨雷", StartTime: 1781284109936, EndTime: 1781543309936, ItemName: "神马驼驼", ItemType: "equip", Display: "1136.png", Count: 1, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 300},
	{Handle: "7878283959784843", Index: 9, Seller: "风雨雷", StartTime: 1781283959784, EndTime: 1781543159784, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "7525283954634793", Index: 10, Seller: "风雨雷", StartTime: 1781283954634, EndTime: 1781543154634, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "7084283948337430", Index: 11, Seller: "风雨雷", StartTime: 1781283948337, EndTime: 1781543148337, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "6656283942594736", Index: 12, Seller: "风雨雷", StartTime: 1781283942594, EndTime: 1781543142594, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "6098283935060572", Index: 13, Seller: "风雨雷", StartTime: 1781283935060, EndTime: 1781543135060, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "5677283929630192", Index: 14, Seller: "风雨雷", StartTime: 1781283929630, EndTime: 1781543129630, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "5280283924173516", Index: 15, Seller: "风雨雷", StartTime: 1781283924173, EndTime: 1781543124173, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "4425283912409892", Index: 16, Seller: "风雨雷", StartTime: 1781283912409, EndTime: 1781543112409, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "3259283895630290", Index: 17, Seller: "风雨雷", StartTime: 1781283895630, EndTime: 1781543095630, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "8380283862068683", Index: 18, Seller: "风雨雷", StartTime: 1781283862068, EndTime: 1781543062068, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "3100283854277323", Index: 19, Seller: "风雨雷", StartTime: 1781283854277, EndTime: 1781543054277, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "9343283841580428", Index: 20, Seller: "风雨雷", StartTime: 1781283841580, EndTime: 1781543041580, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "8993283835874421", Index: 21, Seller: "风雨雷", StartTime: 1781283835874, EndTime: 1781543035874, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "8542283830099323", Index: 22, Seller: "风雨雷", StartTime: 1781283830099, EndTime: 1781543030099, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "8066283824073200", Index: 23, Seller: "风雨雷", StartTime: 1781283824073, EndTime: 1781543024073, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "7532283818191713", Index: 24, Seller: "风雨雷", StartTime: 1781283818191, EndTime: 1781543018191, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "6191283800118838", Index: 25, Seller: "风雨雷", StartTime: 1781283800118, EndTime: 1781543000118, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "5701283793293879", Index: 26, Seller: "风雨雷", StartTime: 1781283793293, EndTime: 1781542993293, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 490, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "3594283766471897", Index: 27, Seller: "风雨雷", StartTime: 1781283766471, EndTime: 1781542966471, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "2990283758576346", Index: 28, Seller: "风雨雷", StartTime: 1781283758576, EndTime: 1781542958576, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "2576283752673593", Index: 29, Seller: "风雨雷", StartTime: 1781283752673, EndTime: 1781542952673, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "2050283744696459", Index: 0, Seller: "风雨雷", StartTime: 1781283744696, EndTime: 1781542944696, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 499, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "1477283737083122", Index: 1, Seller: "风雨雷", StartTime: 1781283737083, EndTime: 1781542937083, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "3450283719365936", Index: 2, Seller: "风雨雷", StartTime: 1781283719365, EndTime: 1781542919365, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "9934283713624299", Index: 3, Seller: "风雨雷", StartTime: 1781283713624, EndTime: 1781542913624, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "9563283707675808", Index: 4, Seller: "风雨雷", StartTime: 1781283707675, EndTime: 1781542907675, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "8147283688567317", Index: 5, Seller: "风雨雷", StartTime: 1781283688567, EndTime: 1781542888567, ItemName: "装备重置符", ItemType: "oneI", Display: "777.png", Count: 99, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 200},
	{Handle: "5204283649410584", Index: 6, Seller: "风雨雷", StartTime: 1781283649410, EndTime: 1781542849410, ItemName: "萌兔宝宝", ItemType: "equip", Display: "1188.png", Count: 1, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 9999},
	{Handle: "3949283631760121", Index: 7, Seller: "风雨雷", StartTime: 1781283631760, EndTime: 1781542831760, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "3365283624356676", Index: 8, Seller: "风雨雷", StartTime: 1781283624356, EndTime: 1781542824356, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "7210283586814233", Index: 9, Seller: "风雨雷", StartTime: 1781283586814, EndTime: 1781542786814, ItemName: "神马驼驼", ItemType: "equip", Display: "1136.png", Count: 1, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 300},
	{Handle: "2232283467131727", Index: 10, Seller: "风雨雷", StartTime: 1781283467131, EndTime: 1781542667131, ItemName: "粉莲宝座", ItemType: "equip", Display: "1137.png", Count: 1, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 300},
	{Handle: "1876283462130118", Index: 11, Seller: "风雨雷", StartTime: 1781283462130, EndTime: 1781542662130, ItemName: "粉莲宝座", ItemType: "equip", Display: "1137.png", Count: 1, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 300},
	{Handle: "2930259882688613", Index: 12, Seller: "虎王", StartTime: 1781259882688, EndTime: 1781519082688, ItemName: "特级精炼宝石", ItemType: "oneI", Display: "1145.png", Count: 5, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 2500},
	{Handle: "9262196268878987", Index: 13, Seller: "风雨雷", StartTime: 1781196268878, EndTime: 1781455468878, ItemName: "装备重置符", ItemType: "oneI", Display: "777.png", Count: 99, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 200},
	{Handle: "8680196262571240", Index: 14, Seller: "风雨雷", StartTime: 1781196262571, EndTime: 1781455462571, ItemName: "装备重置符", ItemType: "oneI", Display: "777.png", Count: 99, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 200},
	{Handle: "8290196256789788", Index: 15, Seller: "风雨雷", StartTime: 1781196256789, EndTime: 1781455456789, ItemName: "装备重置符", ItemType: "oneI", Display: "777.png", Count: 99, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 200},
	{Handle: "2310193040981729", Index: 16, Seller: "风雨雷", StartTime: 1781193040981, EndTime: 1781452240981, ItemName: "魁拔幻化珠", ItemType: "equip", Display: "1340.png", Count: 1, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 9999},
	{Handle: "8855192835012924", Index: 17, Seller: "风雨雷", StartTime: 1781192835012, EndTime: 1781452035012, ItemName: "装备重置符", ItemType: "oneI", Display: "777.png", Count: 99, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 200},
	{Handle: "8171192823957787", Index: 18, Seller: "风雨雷", StartTime: 1781192823957, EndTime: 1781452023957, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "7721192816757868", Index: 19, Seller: "风雨雷", StartTime: 1781192816757, EndTime: 1781452016757, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "7313192810846315", Index: 20, Seller: "风雨雷", StartTime: 1781192810846, EndTime: 1781452010846, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "6996192805650397", Index: 21, Seller: "风雨雷", StartTime: 1781192805650, EndTime: 1781452005650, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "6187192793560802", Index: 22, Seller: "风雨雷", StartTime: 1781192793560, EndTime: 1781451993560, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "5743192787453994", Index: 23, Seller: "风雨雷", StartTime: 1781192787453, EndTime: 1781451987453, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "5246192778741594", Index: 24, Seller: "风雨雷", StartTime: 1781192778741, EndTime: 1781451978741, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "4939192772817296", Index: 25, Seller: "风雨雷", StartTime: 1781192772817, EndTime: 1781451972817, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "4534192765395518", Index: 26, Seller: "风雨雷", StartTime: 1781192765395, EndTime: 1781451965395, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "4211192759368683", Index: 27, Seller: "风雨雷", StartTime: 1781192759368, EndTime: 1781451959368, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "3784192753000450", Index: 28, Seller: "风雨雷", StartTime: 1781192753000, EndTime: 1781451953000, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "3284192745543526", Index: 29, Seller: "风雨雷", StartTime: 1781192745543, EndTime: 1781451945543, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "2804192736274688", Index: 0, Seller: "风雨雷", StartTime: 1781192736274, EndTime: 1781451936274, ItemName: "筋斗云", ItemType: "equip", Display: "968.png", Count: 1, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 800},
	{Handle: "6950192701755600", Index: 1, Seller: "风雨雷", StartTime: 1781192701755, EndTime: 1781451901755, ItemName: "装备重置符", ItemType: "oneI", Display: "777.png", Count: 99, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 200},
	{Handle: "3520192695345229", Index: 2, Seller: "风雨雷", StartTime: 1781192695345, EndTime: 1781451895345, ItemName: "装备重置符", ItemType: "oneI", Display: "777.png", Count: 99, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 200},
	{Handle: "8260192660836819", Index: 3, Seller: "风雨雷", StartTime: 1781192660836, EndTime: 1781451860836, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "7683192651796438", Index: 4, Seller: "风雨雷", StartTime: 1781192651796, EndTime: 1781451851796, ItemName: "四象镜", ItemType: "equip", Display: "1710.png", Count: 1, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 5000},
	{Handle: "7293192645520999", Index: 5, Seller: "风雨雷", StartTime: 1781192645520, EndTime: 1781451845520, ItemName: "四象镜", ItemType: "equip", Display: "1710.png", Count: 1, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 5000},
	{Handle: "3196192582568459", Index: 6, Seller: "风雨雷", StartTime: 1781192582568, EndTime: 1781451782568, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "2864192577040896", Index: 7, Seller: "风雨雷", StartTime: 1781192577040, EndTime: 1781451777040, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "2567192572017514", Index: 8, Seller: "风雨雷", StartTime: 1781192572017, EndTime: 1781451772017, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "2160192563800900", Index: 9, Seller: "风雨雷", StartTime: 1781192563800, EndTime: 1781451763800, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "1697192556766911", Index: 10, Seller: "风雨雷", StartTime: 1781192556766, EndTime: 1781451756766, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "1275192548887761", Index: 11, Seller: "风雨雷", StartTime: 1781192548887, EndTime: 1781451748887, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "1710192533630304", Index: 12, Seller: "风雨雷", StartTime: 1781192533630, EndTime: 1781451733630, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "9371192522399472", Index: 13, Seller: "风雨雷", StartTime: 1781192522399, EndTime: 1781451722399, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "9068192517629760", Index: 14, Seller: "风雨雷", StartTime: 1781192517629, EndTime: 1781451717629, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "8659192511099270", Index: 15, Seller: "风雨雷", StartTime: 1781192511099, EndTime: 1781451711099, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "7058191298329418", Index: 16, Seller: "风雨雷", StartTime: 1781191298329, EndTime: 1781450498329, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "6487191290118107", Index: 17, Seller: "风雨雷", StartTime: 1781191290118, EndTime: 1781450490118, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "5972191282958612", Index: 18, Seller: "风雨雷", StartTime: 1781191282958, EndTime: 1781450482958, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "7015191143319492", Index: 19, Seller: "风雨雷", StartTime: 1781191143319, EndTime: 1781450343319, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "6331191131769979", Index: 20, Seller: "风雨雷", StartTime: 1781191131769, EndTime: 1781450331769, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "5913191124166292", Index: 21, Seller: "风雨雷", StartTime: 1781191124166, EndTime: 1781450324166, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "9147191021216976", Index: 22, Seller: "风雨雷", StartTime: 1781191021216, EndTime: 1781450221216, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "8747191016114112", Index: 23, Seller: "风雨雷", StartTime: 1781191016114, EndTime: 1781450216114, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "8458191010209143", Index: 24, Seller: "风雨雷", StartTime: 1781191010209, EndTime: 1781450210209, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "8032191003508864", Index: 25, Seller: "风雨雷", StartTime: 1781191003508, EndTime: 1781450203508, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "7586190996833474", Index: 26, Seller: "风雨雷", StartTime: 1781190996833, EndTime: 1781450196833, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "7071190989922239", Index: 27, Seller: "风雨雷", StartTime: 1781190989922, EndTime: 1781450189922, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "6240190978117905", Index: 28, Seller: "风雨雷", StartTime: 1781190978117, EndTime: 1781450178117, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "5489190521292337", Index: 29, Seller: "风雨雷", StartTime: 1781190521292, EndTime: 1781449721292, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "4939190512616951", Index: 0, Seller: "风雨雷", StartTime: 1781190512616, EndTime: 1781449712616, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "4251190502061389", Index: 1, Seller: "风雨雷", StartTime: 1781190502061, EndTime: 1781449702061, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "3850190496405728", Index: 2, Seller: "风雨雷", StartTime: 1781190496405, EndTime: 1781449696405, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "3507190491105383", Index: 3, Seller: "风雨雷", StartTime: 1781190491105, EndTime: 1781449691105, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "2900190482503619", Index: 4, Seller: "风雨雷", StartTime: 1781190482503, EndTime: 1781449682503, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "1594190462622492", Index: 5, Seller: "风雨雷", StartTime: 1781190462622, EndTime: 1781449662622, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "1118190456031481", Index: 6, Seller: "风雨雷", StartTime: 1781190456031, EndTime: 1781449656031, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "7010190449857380", Index: 7, Seller: "风雨雷", StartTime: 1781190449857, EndTime: 1781449649857, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "1550190440306509", Index: 8, Seller: "风雨雷", StartTime: 1781190440306, EndTime: 1781449640306, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "9875190435034140", Index: 9, Seller: "风雨雷", StartTime: 1781190435034, EndTime: 1781449635034, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "9491190427724723", Index: 10, Seller: "风雨雷", StartTime: 1781190427724, EndTime: 1781449627724, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "9066190421048850", Index: 11, Seller: "风雨雷", StartTime: 1781190421048, EndTime: 1781449621048, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "8681190414241263", Index: 12, Seller: "风雨雷", StartTime: 1781190414241, EndTime: 1781449614241, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "8213190407535684", Index: 13, Seller: "风雨雷", StartTime: 1781190407535, EndTime: 1781449607535, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "7844190401124800", Index: 14, Seller: "风雨雷", StartTime: 1781190401124, EndTime: 1781449601124, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "7475190394474458", Index: 15, Seller: "风雨雷", StartTime: 1781190394474, EndTime: 1781449594474, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "6955190383795273", Index: 16, Seller: "风雨雷", StartTime: 1781190383795, EndTime: 1781449583795, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "6389190375030888", Index: 17, Seller: "风雨雷", StartTime: 1781190375030, EndTime: 1781449575030, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "5758189744907593", Index: 18, Seller: "风雨雷", StartTime: 1781189744907, EndTime: 1781448944907, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "3654189715253196", Index: 19, Seller: "风雨雷", StartTime: 1781189715253, EndTime: 1781448915253, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "3339189710187347", Index: 20, Seller: "风雨雷", StartTime: 1781189710187, EndTime: 1781448910187, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "9830189676299523", Index: 21, Seller: "风雨雷", StartTime: 1781189676299, EndTime: 1781448876299, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "4250189669022199", Index: 22, Seller: "风雨雷", StartTime: 1781189669022, EndTime: 1781448869022, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "9019189648350274", Index: 23, Seller: "风雨雷", StartTime: 1781189648350, EndTime: 1781448848350, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "8963189183330867", Index: 24, Seller: "风雨雷", StartTime: 1781189183330, EndTime: 1781448383330, ItemName: "千年灵芝", ItemType: "null", Display: "588.png", Count: 999, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 3600},
	{Handle: "7935189167823812", Index: 25, Seller: "风雨雷", StartTime: 1781189167823, EndTime: 1781448367823, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "7634189162544300", Index: 26, Seller: "风雨雷", StartTime: 1781189162544, EndTime: 1781448362544, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "7140189155452323", Index: 27, Seller: "风雨雷", StartTime: 1781189155452, EndTime: 1781448355452, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "6645189147292203", Index: 28, Seller: "风雨雷", StartTime: 1781189147292, EndTime: 1781448347292, ItemName: "筋斗云", ItemType: "equip", Display: "968.png", Count: 1, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 800},
	{Handle: "5809189134353394", Index: 29, Seller: "风雨雷", StartTime: 1781189134353, EndTime: 1781448334353, ItemName: "仙宝葫芦", ItemType: "equip", Display: "1138.png", Count: 1, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 9999},
	{Handle: "3591189098296766", Index: 0, Seller: "风雨雷", StartTime: 1781189098296, EndTime: 1781448298296, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "3165189091012328", Index: 1, Seller: "风雨雷", StartTime: 1781189091012, EndTime: 1781448291012, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "2786189084629415", Index: 2, Seller: "风雨雷", StartTime: 1781189084629, EndTime: 1781448284629, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "2318189076595481", Index: 3, Seller: "风雨雷", StartTime: 1781189076595, EndTime: 1781448276595, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "2008189070087975", Index: 4, Seller: "风雨雷", StartTime: 1781189070087, EndTime: 1781448270087, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "1430189060392714", Index: 5, Seller: "风雨雷", StartTime: 1781189060392, EndTime: 1781448260392, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "1090189053659480", Index: 6, Seller: "风雨雷", StartTime: 1781189053659, EndTime: 1781448253659, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "7030189047858956", Index: 7, Seller: "风雨雷", StartTime: 1781189047858, EndTime: 1781448247858, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "2990189041245726", Index: 8, Seller: "风雨雷", StartTime: 1781189041245, EndTime: 1781448241245, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "9658189030898270", Index: 9, Seller: "风雨雷", StartTime: 1781189030898, EndTime: 1781448230898, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "9388189025488980", Index: 10, Seller: "风雨雷", StartTime: 1781189025488, EndTime: 1781448225488, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "8794189015949341", Index: 11, Seller: "风雨雷", StartTime: 1781189015949, EndTime: 1781448215949, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "8422189009789938", Index: 12, Seller: "风雨雷", StartTime: 1781189009789, EndTime: 1781448209789, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "8025189002633720", Index: 13, Seller: "风雨雷", StartTime: 1781189002633, EndTime: 1781448202633, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "7350188991303985", Index: 14, Seller: "风雨雷", StartTime: 1781188991303, EndTime: 1781448191303, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "6935188984511379", Index: 15, Seller: "风雨雷", StartTime: 1781188984511, EndTime: 1781448184511, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "6622188977985430", Index: 16, Seller: "风雨雷", StartTime: 1781188977985, EndTime: 1781448177985, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "6300188971934429", Index: 17, Seller: "风雨雷", StartTime: 1781188971934, EndTime: 1781448171934, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "5968188966845821", Index: 18, Seller: "风雨雷", StartTime: 1781188966845, EndTime: 1781448166845, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "5430188956424541", Index: 19, Seller: "风雨雷", StartTime: 1781188956424, EndTime: 1781448156424, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "5009188949754305", Index: 20, Seller: "风雨雷", StartTime: 1781188949754, EndTime: 1781448149754, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "4711188944433940", Index: 21, Seller: "风雨雷", StartTime: 1781188944433, EndTime: 1781448144433, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "4280188937025297", Index: 22, Seller: "风雨雷", StartTime: 1781188937025, EndTime: 1781448137025, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "8910188841358630", Index: 23, Seller: "风雨雷", StartTime: 1781188841358, EndTime: 1781448041358, ItemName: "装备重置符", ItemType: "oneI", Display: "777.png", Count: 99, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 200},
	{Handle: "8454188833627463", Index: 24, Seller: "风雨雷", StartTime: 1781188833627, EndTime: 1781448033627, ItemName: "装备重置符", ItemType: "oneI", Display: "777.png", Count: 99, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 200},
	{Handle: "8043188827937457", Index: 25, Seller: "风雨雷", StartTime: 1781188827937, EndTime: 1781448027937, ItemName: "装备重置符", ItemType: "oneI", Display: "777.png", Count: 99, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 200},
	{Handle: "7509188820124100", Index: 26, Seller: "风雨雷", StartTime: 1781188820124, EndTime: 1781448020124, ItemName: "装备重置符", ItemType: "oneI", Display: "777.png", Count: 99, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 200},
	{Handle: "7055188812291205", Index: 27, Seller: "风雨雷", StartTime: 1781188812291, EndTime: 1781448012291, ItemName: "奇效宠物药剂", ItemType: "null", Display: "932.png", Count: 500, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 150},
	{Handle: "3972188762608830", Index: 28, Seller: "风雨雷", StartTime: 1781188762608, EndTime: 1781447962608, ItemName: "木葫芦", ItemType: "own", Display: "192.png", Count: 1, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 1000},
	{Handle: "3180183301204950", Index: 29, Seller: "风雨雷", StartTime: 1781183301204, EndTime: 1781442501204, ItemName: "装备重置符", ItemType: "oneI", Display: "777.png", Count: 99, PriceName: "银元宝", PriceType: "own", PriceDisplay: "39.png", PriceCount: 200},
}
