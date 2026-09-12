package main

import (
	"log"
	"strings"
)

const (
	classicAuctionContainerType = "拍卖行"
	classicAuctionSourceTotal   = 0
	classicAuctionSourceCapture = "20260612_211747_199_session_15948/20260612_211850_206_conn_0002#GetAuctionInfoList+c_auctionInfoCount+c_AuctionInfo"
	classicAuctionAddVipCapture = "20260619_190231_297_session_13232/20260619_190236_818_conn_0002#AddAuctionInfo+c_Error"
	classicAuctionVipError      = "交易需要VIP5"
)

var classicAuctionNPCHandles = map[string]bool{
	"4180542615109515": true,
	"5010542616817526": true,
}

const (
	wuliangChouSipinHandle       = "6190542618476150"
	wuliangChouSipinMsgHandle    = "1"
	wuliangChouSipinAuctionReply = "2"
)

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
	return buildClassicAuctionOpenResultForHandle(handle), true
}

func buildClassicAuctionAnswerResult(request classicTownAnswerRequest) (packetResult, bool) {
	if strings.TrimSpace(request.Handle) != wuliangChouSipinHandle ||
		strings.TrimSpace(request.MsgHandle) != wuliangChouSipinMsgHandle ||
		strings.TrimSpace(request.AnswerHandle) != wuliangChouSipinAuctionReply {
		return packetResult{}, false
	}
	return buildClassicAuctionOpenResultForHandle(wuliangChouSipinHandle), true
}

func buildClassicAuctionOpenResultForHandle(handle string) packetResult {
	return packetResult{
		auctionOpen: &classicAuctionOpenPush{
			ContainerType: classicAuctionContainerType,
			NPCHandle:     handle,
		},
		handled: true,
	}
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
		errorMessages: []classicTownErrorPush{{
			Msg:           classicAuctionVipError,
			SourceCapture: classicAuctionAddVipCapture,
			Partial:       true,
		}},
		handled: true,
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

var classicAuctionSourceRows = []classicAuctionSourceRow{}
