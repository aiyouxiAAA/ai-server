package main

import (
	_ "embed"
	"encoding/json"
	"strings"

	"ai-server/internal/session"
)

const (
	classicTownLookEquipmentType          = "装备"
	classicTownLookEquipmentSourceCapture = "9 captured GetLookDetail targets; see .planning/CLASSIC_TOWN_PLAYER_MENU_SOURCE_RESTORE_PLAN.md"
)

type classicTownOtherEquipmentRequest struct {
	RoleName string `json:"roleName"`
	RoleID   string `json:"roleId,omitempty"`
	Handle   string `json:"handle,omitempty"`
}

type classicTownOtherEquipmentPush struct {
	Handle        string                    `json:"handle"`
	RoleID        string                    `json:"roleId,omitempty"`
	RoleName      string                    `json:"roleName"`
	Type          string                    `json:"type"`
	Capacity      int                       `json:"capacity"`
	Items         []classicTownItemInfoPush `json:"items"`
	SourceCapture string                    `json:"sourceCapture,omitempty"`
	Partial       bool                      `json:"partial,omitempty"`
	ErrorCode     string                    `json:"errorCode,omitempty"`
	ErrorMessage  string                    `json:"errorMessage,omitempty"`
}

type classicTownLookEquipmentSpec struct {
	name  string
	index int
}

type classicTownLookEquipmentSnapshot struct {
	handle        string
	roleName      string
	sourceCapture string
	specs         []classicTownLookEquipmentSpec
}

type classicTownLookEquipmentCapture struct {
	Items []session.RoleItem `json:"items"`
}

//go:embed classic_town_other_equipment_capture.json
var classicTownLookEquipmentCaptureJSON []byte

var classicTownCapturedLookEquipmentItemsByHandle = mustClassicTownCapturedLookEquipmentItems()

var classicTownCapturedLookEquipmentSnapshots = []classicTownLookEquipmentSnapshot{
	{
		handle:        "player_3800",
		roleName:      "s4_妮可露露",
		sourceCapture: "D:/yzhgame/WOCClient/tmp/woc-proxy-captures/20260705_174016_470_session_33116/connections/20260705_174022_226_conn_0002/raw/client-to-server-0001.bin#29448 GetLookDetail(player_3800)",
		specs: []classicTownLookEquipmentSpec{
			{name: "御影面具", index: 0}, {name: "御影护肩", index: 1}, {name: "御影护腕", index: 2},
			{name: "赤云封日", index: 3}, {name: "御影护甲", index: 4}, {name: "御影护腿", index: 5},
			{name: "如意之戒", index: 6}, {name: "琉璃耳环", index: 7}, {name: "白玉项链", index: 8},
			{name: "死灵蓝娃", index: 9}, {name: "御影腰带", index: 10}, {name: "红色圣诞", index: 11},
			{name: "御影靴", index: 12}, {name: "如意之戒", index: 13}, {name: "筋斗云", index: 14},
			{name: "乾坤圈", index: 15}, {name: "混天绫", index: 16}, {name: "风火轮", index: 17},
			{name: "萌兔宝宝", index: 18},
		},
	},
	{
		handle:        "player_18590",
		roleName:      "幡",
		sourceCapture: "D:/yzhgame/WOCClient/tmp/woc-proxy-captures/20260713_222222_131_auto_move_session/conn_0002/client-to-server-0001.bin#268 GetLookDetail(player_18590)",
		specs: []classicTownLookEquipmentSpec{
			{name: "炎煌之冠.战", index: 0}, {name: "辟尘护肩", index: 1}, {name: "辟尘护腕", index: 2},
			{name: "灰飞烟灭", index: 3}, {name: "辟尘护甲", index: 4}, {name: "辟尘护腿", index: 5},
			{name: "如意之戒", index: 6}, {name: "琉璃耳环", index: 7}, {name: "白玉项链", index: 8},
			{name: "死灵红娃", index: 9}, {name: "辟尘腰带", index: 10}, {name: "汽车人", index: 11},
			{name: "炎煌战靴.战", index: 12}, {name: "神勇之戒", index: 13}, {name: "筋斗云", index: 14},
			{name: "乾坤圈", index: 15}, {name: "混天绫", index: 16}, {name: "风火轮", index: 17},
			{name: "狰狞神骑", index: 18},
		},
	},
	{
		handle:        "player_21437",
		roleName:      "小帅哥⁡",
		sourceCapture: "D:/yzhgame/WOCClient/tmp/woc-proxy-captures/20260713_222222_131_auto_move_session/conn_0002/client-to-server-0001.bin#274 GetLookDetail(player_21437)",
		specs: []classicTownLookEquipmentSpec{
			{name: "御影面具", index: 0}, {name: "御影护肩", index: 1}, {name: "御影护腕", index: 2},
			{name: "赤云封日", index: 3}, {name: "御影护甲", index: 4}, {name: "御影护腿", index: 5},
			{name: "炫法之戒", index: 6}, {name: "琉璃耳环", index: 7}, {name: "白玉项链", index: 8},
			{name: "死灵蓝娃", index: 9}, {name: "御影腰带", index: 10}, {name: "兔年呈祥", index: 11},
			{name: "御影靴", index: 12}, {name: "炫法之戒", index: 13}, {name: "混天绫", index: 15},
			{name: "乾坤圈", index: 16}, {name: "风火轮", index: 17}, {name: "狰狞神骑", index: 18},
		},
	},
	{
		handle:        "player_17011",
		roleName:      "風也很温柔",
		sourceCapture: "D:/yzhgame/WOCClient/tmp/woc-proxy-captures/20260713_222222_131_auto_move_session/conn_0002/client-to-server-0001.bin#277 GetLookDetail(player_17011)",
		specs: []classicTownLookEquipmentSpec{
			{name: "辟尘冠", index: 0}, {name: "辟尘护肩", index: 1}, {name: "炎煌护腕.战", index: 2},
			{name: "千疮百孔", index: 3}, {name: "辟尘护甲", index: 4}, {name: "辟尘护腿", index: 5},
			{name: "骷髅戒指", index: 6}, {name: "琉璃耳环", index: 7}, {name: "五彩项链", index: 8},
			{name: "魅惑兔奴", index: 9}, {name: "辟尘腰带", index: 10}, {name: "拳皇套装", index: 11},
			{name: "炎煌战靴.战", index: 12}, {name: "超凡之戒", index: 13}, {name: "试炼印", index: 14},
			{name: "筋斗云", index: 15}, {name: "风火轮", index: 16}, {name: "混天绫", index: 17},
			{name: "狰狞神骑", index: 18},
		},
	},
	{
		handle:        "player_20000",
		roleName:      "阿华田",
		sourceCapture: "D:/yzhgame/WOCClient/tmp/woc-proxy-captures/20260713_222222_131_auto_move_session/conn_0002/client-to-server-0001.bin#281 GetLookDetail(player_20000)",
		specs: []classicTownLookEquipmentSpec{
			{name: "御影面具", index: 0}, {name: "御影护肩", index: 1}, {name: "御影护腕", index: 2},
			{name: "赤云封日", index: 3}, {name: "御影护甲", index: 4}, {name: "御影护腿", index: 5},
			{name: "如意之戒", index: 6}, {name: "琉璃耳环", index: 7}, {name: "白玉项链", index: 8},
			{name: "死灵蓝娃", index: 9}, {name: "御影腰带", index: 10}, {name: "兔年呈祥", index: 11},
			{name: "御影靴", index: 12}, {name: "如意之戒", index: 13}, {name: "乾坤圈", index: 14},
			{name: "风火轮", index: 15}, {name: "混天绫", index: 16}, {name: "聚宝鼎", index: 17},
			{name: "狰狞神骑", index: 18},
		},
	},
	{
		handle:        "player_21424",
		roleName:      "恐龙抗狼1",
		sourceCapture: "D:/yzhgame/WOCClient/instances/instance2.staging/tmp/woc-proxy-captures/20260612_211747_199_session_15948/connections/20260612_211850_206_conn_0002/raw/client-to-server-0001.bin#5868 GetLookDetail(player_21424)",
		specs: []classicTownLookEquipmentSpec{
			{name: "无双头盔", index: 0}, {name: "无双护肩", index: 1}, {name: "无双铁腕", index: 2},
			{name: "饮血刀", index: 3}, {name: "寨夫人上衣", index: 4}, {name: "无双护腿", index: 5},
			{name: "泥戒指", index: 6}, {name: "怪木机", index: 9}, {name: "无双铁腰带", index: 10},
			{name: "蛤蟆精战靴", index: 12}, {name: "泥戒指", index: 13},
		},
	},
	{
		handle:        "player_5793",
		roleName:      "FangLv",
		sourceCapture: "D:/yzhgame/WOCClient/instances/instance3.staging/tmp/woc-proxy-captures/20260712_185248_314_auto_move_session/conn_0002/client-to-server-0001.bin#13502 GetLookDetail(player_5793)",
		specs: []classicTownLookEquipmentSpec{
			{name: "御影面具", index: 0}, {name: "御影护肩", index: 1}, {name: "御影护腕", index: 2},
			{name: "赤云封日", index: 3}, {name: "御影护甲", index: 4}, {name: "御影护腿", index: 5},
			{name: "炫法之戒", index: 6}, {name: "琉璃耳环", index: 7}, {name: "白玉项链", index: 8},
			{name: "魅惑兔奴", index: 9}, {name: "炎煌护腰.游", index: 10}, {name: "御影靴", index: 12},
			{name: "炫法之戒", index: 13}, {name: "混天绫", index: 14}, {name: "风火轮", index: 15},
			{name: "乾坤圈", index: 16}, {name: "聚宝鼎", index: 17}, {name: "狰狞神骑", index: 18},
		},
	},
	{
		handle:        "player_16984",
		roleName:      "雨田尚文文文",
		sourceCapture: "D:/yzhgame/WOCClient/instances/instance3.staging/tmp/woc-proxy-captures/20260712_185248_314_auto_move_session/conn_0002/client-to-server-0001.bin#13738 GetLookDetail(player_16984)",
		specs: []classicTownLookEquipmentSpec{
			{name: "御影面具", index: 0}, {name: "御影护肩", index: 1}, {name: "炎煌护腕.游", index: 2},
			{name: "赤云封日", index: 3}, {name: "御影护甲", index: 4}, {name: "御影护腿", index: 5},
			{name: "神勇之戒", index: 6}, {name: "琉璃耳环", index: 7}, {name: "白玉项链", index: 8},
			{name: "魅惑兔奴", index: 9}, {name: "炎煌护腰.游", index: 10}, {name: "龙娃幻化珠", index: 11},
			{name: "炎煌战靴.游", index: 12}, {name: "神勇之戒", index: 13}, {name: "试炼印", index: 14},
			{name: "风火轮", index: 15}, {name: "混天绫", index: 16}, {name: "乾坤圈", index: 17},
			{name: "狰狞神骑", index: 18},
		},
	},
	{
		handle:        "player_18485",
		roleName:      "是那晚夜",
		sourceCapture: "D:/yzhgame/WOCClient/instances/instance3.staging/tmp/woc-proxy-captures/20260712_185248_314_auto_move_session/conn_0002/client-to-server-0001.bin#13818 GetLookDetail(player_18485)",
		specs: []classicTownLookEquipmentSpec{
			{name: "聚仙法冠", index: 0}, {name: "聚仙飘带", index: 1}, {name: "炎煌护腕.术", index: 2},
			{name: "炎雷惊蝎", index: 3}, {name: "聚仙袍", index: 4}, {name: "聚仙护腿", index: 5},
			{name: "如意之戒", index: 6}, {name: "琉璃耳环", index: 7}, {name: "白玉项链", index: 8},
			{name: "魅惑兔奴", index: 9}, {name: "聚仙腰带", index: 10}, {name: "小白兔套装", index: 11},
			{name: "炎煌战靴.术", index: 12}, {name: "如意之戒", index: 13}, {name: "乾坤圈", index: 14},
			{name: "混天绫", index: 15}, {name: "风火轮", index: 16}, {name: "筋斗云", index: 17},
			{name: "狰狞神骑", index: 18},
		},
	},
}

func buildClassicTownOtherEquipmentResult(request classicTownOtherEquipmentRequest) packetResult {
	request = normalizeClassicTownOtherEquipmentRequest(request)
	snapshot, ok := findClassicTownCapturedLookEquipmentSnapshot(request)
	if !ok {
		return packetResult{
			otherEquipment: &classicTownOtherEquipmentPush{
				Handle:        firstNonBlankString(request.Handle, request.RoleID, request.RoleName),
				RoleID:        request.RoleID,
				RoleName:      request.RoleName,
				Type:          classicTownLookEquipmentType,
				Capacity:      0,
				Items:         []classicTownItemInfoPush{},
				SourceCapture: classicTownLookEquipmentSourceCapture,
				Partial:       true,
				ErrorCode:     "look_equipment_unrestored",
				ErrorMessage:  "观察装备数据未恢复",
			},
			handled: true,
		}
	}

	items, ok := classicTownCapturedLookEquipmentItems(snapshot)
	if !ok {
		return packetResult{
			otherEquipment: &classicTownOtherEquipmentPush{
				Handle:        snapshot.handle,
				RoleID:        snapshot.handle,
				RoleName:      snapshot.roleName,
				Type:          classicTownLookEquipmentType,
				Capacity:      0,
				Items:         []classicTownItemInfoPush{},
				SourceCapture: snapshot.sourceCapture,
				Partial:       true,
				ErrorCode:     "look_equipment_template_missing",
				ErrorMessage:  "观察装备模板缺失",
			},
			handled: true,
		}
	}

	return packetResult{
		otherEquipment: &classicTownOtherEquipmentPush{
			Handle:        snapshot.handle,
			RoleID:        snapshot.handle,
			RoleName:      snapshot.roleName,
			Type:          classicTownLookEquipmentType,
			Capacity:      20,
			Items:         items,
			SourceCapture: snapshot.sourceCapture,
			Partial:       true,
		},
		handled: true,
	}
}

func normalizeClassicTownOtherEquipmentRequest(request classicTownOtherEquipmentRequest) classicTownOtherEquipmentRequest {
	request.RoleName = strings.TrimSpace(request.RoleName)
	request.RoleID = strings.TrimSpace(request.RoleID)
	request.Handle = strings.TrimSpace(request.Handle)
	return request
}

func findClassicTownCapturedLookEquipmentSnapshot(request classicTownOtherEquipmentRequest) (classicTownLookEquipmentSnapshot, bool) {
	if request.Handle != "" || request.RoleID != "" {
		for _, identity := range []string{request.Handle, request.RoleID} {
			if identity == "" {
				continue
			}
			for _, snapshot := range classicTownCapturedLookEquipmentSnapshots {
				if snapshot.handle == identity {
					return snapshot, true
				}
			}
		}
		return classicTownLookEquipmentSnapshot{}, false
	}

	var matched *classicTownLookEquipmentSnapshot
	for index := range classicTownCapturedLookEquipmentSnapshots {
		snapshot := &classicTownCapturedLookEquipmentSnapshots[index]
		if snapshot.roleName != request.RoleName {
			continue
		}
		if matched != nil {
			return classicTownLookEquipmentSnapshot{}, false
		}
		matched = snapshot
	}
	if matched == nil {
		return classicTownLookEquipmentSnapshot{}, false
	}
	return *matched, true
}

func classicTownCapturedLookEquipmentItems(snapshot classicTownLookEquipmentSnapshot) ([]classicTownItemInfoPush, bool) {
	capturedItems, ok := classicTownCapturedLookEquipmentItemsByHandle[snapshot.handle]
	if !ok || len(capturedItems) != len(snapshot.specs) {
		return nil, false
	}
	items := make([]classicTownItemInfoPush, 0, len(capturedItems))
	for index, item := range capturedItems {
		spec := snapshot.specs[index]
		if item.Handle != snapshot.handle || item.Type != classicTownLookEquipmentType || item.Name != spec.name || item.Index != spec.index {
			return nil, false
		}
		items = append(items, classicTownItemInfoPushFromRoleItem(item))
	}
	return items, true
}

func mustClassicTownCapturedLookEquipmentItems() map[string][]session.RoleItem {
	var capture classicTownLookEquipmentCapture
	if err := json.Unmarshal(classicTownLookEquipmentCaptureJSON, &capture); err != nil {
		panic("decode captured other-equipment item snapshots: " + err.Error())
	}
	itemsByHandle := make(map[string][]session.RoleItem)
	for _, item := range capture.Items {
		if item.Handle == "" || item.Type != classicTownLookEquipmentType || item.ItemType != "equip" || item.Name == "" || item.Display == "" || item.Description == "" || item.Index < 0 {
			panic("invalid captured other-equipment item snapshot")
		}
		itemsByHandle[item.Handle] = append(itemsByHandle[item.Handle], item)
	}
	return itemsByHandle
}

func firstNonBlankString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
