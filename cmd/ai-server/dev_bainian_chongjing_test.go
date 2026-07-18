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

func TestDevRefreshBainianChongjingEndpointRestoresCurrentCycleEncounter(t *testing.T) {
	classicactivity.ResetBainianChongjingKillStateForTest()
	t.Cleanup(classicactivity.ResetBainianChongjingKillStateForTest)
	swapWorldSceneHub(t)

	mux := http.NewServeMux()
	registerDevItemHandlers(mux, session.NewStore())
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/dev/items/refresh-bainian-chongjing", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	var response devBainianChongjingRefreshResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success || response.MapID != classicactivity.BainianChongjingMapID {
		t.Fatalf("unexpected response %+v", response)
	}
	if !reflect.DeepEqual(response.Handles, classicactivity.BainianChongjingEncounterHandles()) {
		t.Fatalf("unexpected encounter handles %v", response.Handles)
	}
	if _, ok := classicactivity.BainianChongjingSpawnForMap(classicactivity.BainianChongjingMapID, time.Now()); !ok {
		t.Fatal("dev refresh should make the map bootstrap include the encounter")
	}
}

func TestDevRefreshBainianChongjingEndpointRequiresPost(t *testing.T) {
	mux := http.NewServeMux()
	registerDevItemHandlers(mux, session.NewStore())
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/dev/items/refresh-bainian-chongjing", nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}
