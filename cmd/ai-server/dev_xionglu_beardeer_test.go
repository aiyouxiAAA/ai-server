package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"ai-server/internal/classicactivity"
	"ai-server/internal/session"
)

func TestDevRefreshXiongluBeardeerEndpointRestoresCurrentCycleEncounter(t *testing.T) {
	classicactivity.ResetXiongluBeardeerKillStateForTest()
	t.Cleanup(classicactivity.ResetXiongluBeardeerKillStateForTest)
	swapWorldSceneHub(t)

	mux := http.NewServeMux()
	registerDevItemHandlers(mux, session.NewStore())
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/dev/items/refresh-xionglu-beardeer", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	var response devXiongluBeardeerRefreshResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success || response.MapID != classicactivity.XiongluBeardeerMapID {
		t.Fatalf("unexpected response %+v", response)
	}
	if !reflect.DeepEqual(response.Handles, classicactivity.XiongluBeardeerEncounterHandles()) {
		t.Fatalf("unexpected encounter handles %v", response.Handles)
	}
	if _, ok := classicactivity.XiongluBeardeerSpawnForMap(classicactivity.XiongluBeardeerMapID, time.Now()); !ok {
		t.Fatal("dev refresh should make the map bootstrap include the encounter")
	}
}

func TestDevRefreshXiongluBeardeerEndpointRequiresPost(t *testing.T) {
	mux := http.NewServeMux()
	registerDevItemHandlers(mux, session.NewStore())
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/dev/items/refresh-xionglu-beardeer", nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}
