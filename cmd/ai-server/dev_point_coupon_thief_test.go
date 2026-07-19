package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ai-server/internal/classicactivity"
	"ai-server/internal/session"
)

func TestDevRefreshPointCouponThiefReplacesAllActivityMaps(t *testing.T) {
	classicactivity.ResetPointCouponThiefRefreshStateForTest()
	t.Cleanup(classicactivity.ResetPointCouponThiefRefreshStateForTest)
	swapWorldSceneHub(t)
	now := time.Date(2026, 6, 19, 18, 37, 0, 0, time.Local)
	recorder := httptest.NewRecorder()
	handleDevRefreshPointCouponThief(recorder, now)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	var response devPointCouponThiefRefreshResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success || len(response.Spawns) != 3 {
		t.Fatalf("expected three refreshed activity spawns, got %+v", response)
	}
	for _, spawn := range response.Spawns {
		active, ok := classicactivity.PointCouponThiefSpawnForMap(spawn.MapID, now)
		if !ok || active.Handle != spawn.Handle {
			t.Fatalf("expected map%d bootstrap handle %s, got %+v", spawn.MapID, spawn.Handle, active)
		}
	}
}

func TestDevRefreshPointCouponThiefEndpointRequiresPost(t *testing.T) {
	mux := http.NewServeMux()
	registerDevItemHandlers(mux, session.NewStore())
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/dev/items/refresh-point-coupon-thief", nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}
