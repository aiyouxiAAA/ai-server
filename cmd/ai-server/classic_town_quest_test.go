package main

import "testing"

func TestClassicQuestRouteCanComeFromCatalog(t *testing.T) {
	route, ok := findClassicQuestAnswerRoute(classicTownAnswerRequest{
		Handle:       "6350542618650282",
		MsgHandle:    "6q2d_1",
		AnswerHandle: "6q2a_1_1",
	})
	if !ok {
		t.Fatal("expected catalog-backed Wuliang quest route")
	}
	if route.title != "侦查敌营" {
		t.Fatalf("expected Wuliang route title, got %+v", route)
	}
	if handle := questNpcHandleForTitle("拜访故人"); handle != "6350542618650282" {
		t.Fatalf("expected Wuliang quest marker handle, got %s", handle)
	}
}
