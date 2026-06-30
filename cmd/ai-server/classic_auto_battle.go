package main

import "log"

const (
	classicAutoBattleSourceCapture      = "tmp/capture-timeline-feature-gap-audit.json#AutoBattle(257)+GetAB(259)+c_ab(50109)+StopAutoBattle(260)"
	classicAutoBattleDefaultDelay       = 30
	classicAutoBattleDefaultExpFlag     = 1
	classicAutoBattleDefaultPointCost   = 0
	classicAutoBattleDefaultPointRemain = 0
)

type classicAutoBattleInfoRequest struct{}

type classicAutoBattleStartRequest struct {
	DelaySeconds int  `json:"delaySeconds,omitempty"`
	Full         bool `json:"full,omitempty"`
	ExpFlag      int  `json:"expFlag,omitempty"`
	FillHp       bool `json:"fillHp,omitempty"`
	FillMp       bool `json:"fillMp,omitempty"`
	Relive       bool `json:"relive,omitempty"`
}

type classicAutoBattleStopRequest struct{}

type classicAutoBattleInfoPush struct {
	PointCost       int    `json:"pointCost"`
	RemainingPoints int    `json:"remainingPoints"`
	DelaySeconds    int    `json:"delaySeconds"`
	Full            bool   `json:"full"`
	ExpFlag         int    `json:"expFlag"`
	FillHp          bool   `json:"fillHp"`
	FillMp          bool   `json:"fillMp"`
	Relive          bool   `json:"relive"`
	Running         bool   `json:"running"`
	StatusText      string `json:"statusText"`
	SourceCapture   string `json:"sourceCapture"`
	Partial         bool   `json:"partial"`
}

func buildClassicAutoBattleInfoResult(socketSession *packetSession, _ classicAutoBattleInfoRequest) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic auto battle GetAB ignored without selected role")
		return packetResult{handled: true}
	}
	return packetResult{
		autoBattleInfo: buildClassicAutoBattleInfoPush(false, classicAutoBattleStartRequest{}, ""),
		handled:        true,
	}
}

func buildClassicAutoBattleStartResult(socketSession *packetSession, request classicAutoBattleStartRequest) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic auto battle AutoBattle ignored without selected role")
		return packetResult{handled: true}
	}
	return packetResult{
		autoBattleInfo: buildClassicAutoBattleInfoPush(true, request, "正在挂机中..."),
		handled:        true,
	}
}

func buildClassicAutoBattleStopResult(socketSession *packetSession, _ classicAutoBattleStopRequest) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic auto battle StopAutoBattle ignored without selected role")
		return packetResult{handled: true}
	}
	return packetResult{
		autoBattleInfo: buildClassicAutoBattleInfoPush(false, classicAutoBattleStartRequest{}, ""),
		handled:        true,
	}
}

func buildClassicAutoBattleInfoPush(running bool, request classicAutoBattleStartRequest, statusText string) *classicAutoBattleInfoPush {
	delaySeconds := request.DelaySeconds
	if delaySeconds < 0 {
		delaySeconds = 0
	}
	if delaySeconds == 0 && !running {
		delaySeconds = classicAutoBattleDefaultDelay
	}
	expFlag := request.ExpFlag
	if expFlag <= 0 {
		expFlag = classicAutoBattleDefaultExpFlag
	}
	return &classicAutoBattleInfoPush{
		PointCost:       classicAutoBattleDefaultPointCost,
		RemainingPoints: classicAutoBattleDefaultPointRemain,
		DelaySeconds:    delaySeconds,
		Full:            request.Full,
		ExpFlag:         expFlag,
		FillHp:          request.FillHp,
		FillMp:          request.FillMp,
		Relive:          request.Relive,
		Running:         running,
		StatusText:      statusText,
		SourceCapture:   classicAutoBattleSourceCapture,
		Partial:         true,
	}
}
