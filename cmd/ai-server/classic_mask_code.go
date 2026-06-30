package main

const (
	classicMaskCodeWidth         = 80
	classicMaskCodeHeight        = 30
	classicMaskCodeSourceCapture = "D:/yzhgame/WOCClient/instances/instance2.staging/tmp/woc-proxy-captures/20260606_210926_394_session_08036/connections/20260606_215548_514_conn_0005/raw/server-to-client-0001.bin#c_maskCode + raw/client-to-server-0001.bin#UpMaskCode"
)

type classicMaskCodeRefreshRequest struct{}

type classicMaskCodeSubmitRequest struct {
	Value string `json:"value"`
}

type classicMaskCodeChallengePush struct {
	Code          string `json:"code"`
	Width         int    `json:"width"`
	Height        int    `json:"height"`
	SourceCapture string `json:"sourceCapture"`
	Partial       bool   `json:"partial"`
}

type classicMaskCodeResultPush struct {
	Accepted      bool   `json:"accepted"`
	Message       string `json:"message"`
	SourceCapture string `json:"sourceCapture"`
	Partial       bool   `json:"partial"`
}

func buildClassicMaskCodeRefreshResult(_ classicMaskCodeRefreshRequest) packetResult {
	return packetResult{
		maskCodeChallenge: &classicMaskCodeChallengePush{
			Code:          buildClassicMaskCodeSample(),
			Width:         classicMaskCodeWidth,
			Height:        classicMaskCodeHeight,
			SourceCapture: classicMaskCodeSourceCapture,
			Partial:       true,
		},
		handled: true,
	}
}

func buildClassicMaskCodeSubmitResult(request classicMaskCodeSubmitRequest) packetResult {
	accepted := request.Value != ""
	message := "验证码已提交"
	if !accepted {
		message = "验证码不能为空"
	}
	return packetResult{
		maskCodeResult: &classicMaskCodeResultPush{
			Accepted:      accepted,
			Message:       message,
			SourceCapture: classicMaskCodeSourceCapture,
			Partial:       true,
		},
		handled: true,
	}
}

func buildClassicMaskCodeSample() string {
	bytes := make([]byte, classicMaskCodeWidth*classicMaskCodeHeight)
	for y := 0; y < classicMaskCodeHeight; y++ {
		for x := 0; x < classicMaskCodeWidth; x++ {
			index := y*classicMaskCodeWidth + x
			active := (x+y)%17 == 0 || (x-y+classicMaskCodeWidth)%23 == 0
			if y > 8 && y < 21 {
				active = active || (x > 8 && x < 16 && y%3 == 0)
				active = active || (x > 25 && x < 34 && (x+y)%5 == 0)
				active = active || (x > 45 && x < 52 && y%4 == 0)
				active = active || (x > 62 && x < 70 && (x-y)%4 == 0)
			}
			if active {
				bytes[index] = '1'
			} else {
				bytes[index] = '0'
			}
		}
	}
	return string(bytes)
}
