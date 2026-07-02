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

func TestClassicQuestRouteIncludesCapturedDafoWoodMonsterQuest(t *testing.T) {
	route, ok := findClassicQuestAnswerRoute(classicTownAnswerRequest{
		Handle:       "4090542614314425",
		MsgHandle:    "2q23d_1",
		AnswerHandle: "2q23a_1_1",
	})
	if !ok {
		t.Fatal("expected captured Dafo quest route")
	}
	if route.title != "讨厌的枯木怪" {
		t.Fatalf("expected captured Dafo route title, got %+v", route)
	}
	if handle := questNpcHandleForTitle("讨厌的枯木怪"); handle != "4090542614314425" {
		t.Fatalf("expected captured Dafo quest marker handle, got %s", handle)
	}
}
