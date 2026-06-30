package main

import "strings"

const (
	classicMailContainerType  = "邮件"
	classicMailSourceCapture  = "20260606_210926_394_session_08036/20260606_210959_953_conn_0002#c_mailSubjectList+GetMail+c_mailInfo"
	classicMailSendVipCapture = "20260613_160656_039_session_27124/20260613_160706_875_conn_0002#SendMail+c_Error"
	classicMailSendVipError   = "交易需要VIP5"
)

type classicMailOpenPush struct {
	ContainerType string `json:"containerType"`
	NPCHandle     string `json:"npcHandle"`
	SourceCapture string `json:"sourceCapture"`
}

type classicMailSubjectPush struct {
	Handle   string `json:"handle"`
	Subject  string `json:"subject"`
	SendTime int64  `json:"sendTime"`
	IsRead   bool   `json:"isRead"`
}

type classicMailListPush struct {
	Items         []classicMailSubjectPush `json:"items"`
	MailCost      []string                 `json:"mailCost"`
	SourceCapture string                   `json:"sourceCapture"`
	Partial       bool                     `json:"partial"`
}

type classicMailInfoRequest struct {
	Handle string `json:"handle"`
}

type classicMailAttachmentRequest struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
	Count int    `json:"count"`
}

type classicMailSendRequest struct {
	Subject  string                         `json:"subject"`
	To       string                         `json:"to"`
	Content  string                         `json:"content"`
	SendType int                            `json:"sendType"`
	Items    []classicMailAttachmentRequest `json:"items,omitempty"`
}

type classicMailInfoPush struct {
	Handle        string `json:"handle"`
	Subject       string `json:"subject"`
	From          string `json:"from"`
	Content       string `json:"content"`
	SendDate      int64  `json:"sendDate"`
	Found         bool   `json:"found"`
	ErrorMessage  string `json:"errorMessage,omitempty"`
	SourceCapture string `json:"sourceCapture"`
}

type classicMailSourceRecord struct {
	Handle   string
	Subject  string
	From     string
	Content  string
	SendDate int64
	IsRead   bool
}

var classicMailNPCHandles = map[string]struct{}{
	"5000542609232627": {}, // 卢掌柜: source answer handle 2 = 收发信件
	"6350542618650282": {}, // 汉雄: source answer handle 2 = 收发信件
	"5050542617322114": {}, // 白叟: source answer handle 2 = 收发信件
}

var classicMailCapturedCost = []string{
	"[i=f_i_铜钱^ffffff&24@材料 消耗品&25@1000&19@1000枚时双击可兑换为银元宝.&20@游戏中的货币,用于流通买卖.&27@sitem_tq&101@163.png&103@0&104@0&105@&107@&108@0]铜钱[/]x100",
	"[i=f_i_铜钱^ffffff&24@材料 消耗品&25@1000&19@1000枚时双击可兑换为银元宝.&20@游戏中的货币,用于流通买卖.&27@sitem_tq&101@163.png&103@0&104@0&105@&107@&108@0]铜钱[/]x50",
	"[i=f_i_铜钱^ffffff&24@材料 消耗品&25@1000&19@1000枚时双击可兑换为银元宝.&20@游戏中的货币,用于流通买卖.&27@sitem_tq&101@163.png&103@0&104@0&105@&107@&108@0]铜钱[/]x5",
}

var classicMailSourceRecords = []classicMailSourceRecord{
	{
		Handle:   "5544758100159914",
		Subject:  "新手任务提示",
		From:     "系统",
		Content:  "恭喜你升到15级,你可以接以下日常任务:\n1.义剑诛鬼 李老头 时间:每晚19:00-22:00\n2.杀猪除害 伏天 时间：每小时循环\n3.助商集物 丑五品 时间：每日循环\n4.生死劫   一心长态 噌痴 申公烈 夏侯武 时间:每日13:00-15:00",
		SendDate: 1780758100159,
		IsRead:   false,
	},
	{
		Handle:   "9073752143795590",
		Subject:  "转职时换下的衣服",
		From:     "系统",
		Content:  "转职时候换下的衣服",
		SendDate: 1780752143795,
		IsRead:   false,
	},
	{
		Handle:   "7328462314719446",
		Subject:  "转职时换下的衣服",
		From:     "系统",
		Content:  "转职时候换下的衣服",
		SendDate: 1779462314719,
		IsRead:   true,
	},
}

func buildClassicMailOpenResult(request classicTownAnswerRequest) (packetResult, bool) {
	if strings.TrimSpace(request.AnswerHandle) != "2" {
		return packetResult{}, false
	}
	if _, ok := classicMailNPCHandles[strings.TrimSpace(request.Handle)]; !ok {
		return packetResult{}, false
	}
	return packetResult{
		mailOpen: &classicMailOpenPush{
			ContainerType: classicMailContainerType,
			NPCHandle:     request.Handle,
			SourceCapture: classicMailSourceCapture,
		},
		mailList: buildClassicMailListPush(),
		handled:  true,
	}, true
}

func buildClassicMailInfoResult(request classicMailInfoRequest) packetResult {
	handle := strings.TrimSpace(request.Handle)
	for _, record := range classicMailSourceRecords {
		if record.Handle != handle {
			continue
		}
		return packetResult{
			mailInfo: &classicMailInfoPush{
				Handle:        record.Handle,
				Subject:       record.Subject,
				From:          record.From,
				Content:       record.Content,
				SendDate:      record.SendDate,
				Found:         true,
				SourceCapture: classicMailSourceCapture,
			},
			handled: true,
		}
	}
	return packetResult{
		mailInfo: &classicMailInfoPush{
			Handle:        handle,
			Found:         false,
			ErrorMessage:  "邮件不存在",
			SourceCapture: classicMailSourceCapture,
		},
		handled: true,
	}
}

func buildClassicMailSendResult(_ classicMailSendRequest) packetResult {
	return packetResult{
		chatMessages: []classicTownChatMessagePush{classicTownSystemWarningMessage(classicMailSendVipError)},
		handled:      true,
	}
}

func buildClassicMailListPush() *classicMailListPush {
	items := make([]classicMailSubjectPush, 0, len(classicMailSourceRecords))
	for _, record := range classicMailSourceRecords {
		items = append(items, classicMailSubjectPush{
			Handle:   record.Handle,
			Subject:  record.Subject,
			SendTime: record.SendDate,
			IsRead:   record.IsRead,
		})
	}
	return &classicMailListPush{
		Items:         items,
		MailCost:      append([]string(nil), classicMailCapturedCost...),
		SourceCapture: classicMailSourceCapture,
		Partial:       true,
	}
}
