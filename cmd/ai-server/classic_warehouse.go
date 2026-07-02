package main

import (
	"log"
	"strings"
)

const (
	classicWarehouseContainerType      = "\u4ed3\u5e93"
	classicWarehouseSourceCapture      = "20260619_190231_297_session_13232/20260619_190236_818_conn_0002#Answer(5050542617322114,1,6)+c_openContainer+GetItemList"
	classicWarehouseCareSourceCapture  = "20260619_190231_297_session_13232/20260619_190236_818_conn_0002#ChangeCarePass+c_careState+\u4ed3\u5e93\u672a\u89e3\u9501"
	classicWarehouseLockedState        = "1"
	classicWarehouseLockedStateMessage = "\u4ed3\u5e93\u672a\u89e3\u9501\uff0c\u70b9\u8fd9\u91cc\u89e3\u9501"
)

type classicTownOpenContainerPush struct {
	Handle        string `json:"handle"`
	Type          string `json:"type"`
	SourceCapture string `json:"sourceCapture,omitempty"`
}

type classicTownCareStatePush struct {
	LockState     string `json:"lockState"`
	UnlockAt      string `json:"unlockAt,omitempty"`
	Message       string `json:"message,omitempty"`
	SourceCapture string `json:"sourceCapture,omitempty"`
}

var classicWarehouseNPCHandles = map[string]struct{}{
	"5000542609232627": {},
	"6350542618650282": {},
	"5050542617322114": {},
}

func buildClassicWarehouseOpenResult(socketSession *packetSession, request classicTownAnswerRequest) (packetResult, bool) {
	if strings.TrimSpace(request.AnswerHandle) != "6" {
		return packetResult{}, false
	}
	sourceHandle := strings.TrimSpace(request.Handle)
	if _, ok := classicWarehouseNPCHandles[sourceHandle]; !ok {
		return packetResult{}, false
	}
	if socketSession == nil || socketSession.selectedRole == nil {
		return packetResult{handled: true}, true
	}
	return packetResult{
		openContainer: &classicTownOpenContainerPush{
			Handle:        sourceHandle,
			Type:          classicWarehouseContainerType,
			SourceCapture: classicWarehouseSourceCapture,
		},
		handled: true,
	}, true
}

func buildClassicWarehouseCareStateResult(socketSession *packetSession) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil {
		log.Println("[ai-server] classic warehouse care state ignored without selected role")
		return packetResult{handled: true}
	}
	return packetResult{
		careState: &classicTownCareStatePush{
			LockState:     classicWarehouseLockedState,
			Message:       classicWarehouseLockedStateMessage,
			SourceCapture: classicWarehouseCareSourceCapture,
		},
		handled: true,
	}
}
