package main

import "log"

const (
	classicDailyRewardLevelRequired = 30
	classicDailyRewardText          = "招财符x1,未领取"
	classicDailyRewardSourceMessage = "今日奖励:招财符x1,未领取|50"
	classicDailyRewardSourceCapture = "D:/yzhgame/WOCClient/tmp/woc-proxy-captures/20260510_062452_186_conn_0002/client-to-server.bin#GetItemEveryDay(0,0)+server-to-client.bin#c_Speak 今日奖励"
	classicDailyRewardClaimCapture  = "D:/yzhgame/WOCClient/tmp/woc-proxy-captures/20260512_230919_229_conn_0002/client-to-server.bin#GetItemEveryDay() empty payload; no paired success item grant found in current audit"
	classicDailyRewardClaimPartial  = "未闭合"
)

type classicDailyRewardInfoRequest struct{}

type classicDailyRewardClaimRequest struct{}

type classicDailyRewardInfoPush struct {
	RewardText    string `json:"rewardText"`
	Claimed       bool   `json:"claimed"`
	CanClaim      bool   `json:"canClaim"`
	LevelRequired int    `json:"levelRequired"`
	StatusText    string `json:"statusText"`
	SourceMessage string `json:"sourceMessage"`
	SourceCapture string `json:"sourceCapture"`
	Partial       bool   `json:"partial"`
}

func buildClassicDailyRewardInfoResult(socketSession *packetSession, _ classicDailyRewardInfoRequest) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic daily reward ItemEveryDayInfo ignored without selected role")
		return packetResult{handled: true}
	}

	return packetResult{
		dailyRewardInfo: buildClassicDailyRewardInfoPush(socketSession.playerBase.Level, "", classicDailyRewardSourceCapture),
		handled:         true,
	}
}

func buildClassicDailyRewardClaimResult(socketSession *packetSession, _ classicDailyRewardClaimRequest) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic daily reward GetItemEveryDay ignored without selected role")
		return packetResult{handled: true}
	}

	statusText := classicDailyRewardClaimPartial
	if socketSession.playerBase.Level < classicDailyRewardLevelRequired {
		statusText = ""
	}
	return packetResult{
		dailyRewardInfo: buildClassicDailyRewardInfoPush(socketSession.playerBase.Level, statusText, classicDailyRewardClaimCapture),
		handled:         true,
	}
}

func buildClassicDailyRewardInfoPush(level int, statusOverride string, sourceCapture string) *classicDailyRewardInfoPush {
	canClaim := level >= classicDailyRewardLevelRequired
	statusText := statusOverride
	if statusText == "" && !canClaim {
		statusText = "30级方能领取"
	}
	return &classicDailyRewardInfoPush{
		RewardText:    classicDailyRewardText,
		Claimed:       false,
		CanClaim:      canClaim,
		LevelRequired: classicDailyRewardLevelRequired,
		StatusText:    statusText,
		SourceMessage: classicDailyRewardSourceMessage,
		SourceCapture: sourceCapture,
		Partial:       true,
	}
}
