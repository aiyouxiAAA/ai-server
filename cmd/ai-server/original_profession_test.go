package main

import "testing"

func TestBladeDancerRetiredShopAndVocationRoutes(t *testing.T) {
	for _, shop := range []string{"skill1", "skill2", "skill3"} {
		if _, ok := findSourceSkillShopEntry(shop, 1); ok {
			t.Fatal("old skill purchase available")
		}
	}
	for _, answer := range []string{"7", "8", "9"} {
		result, handled := buildClassicTownSkillShopResult(nil, nil, sourceSkillTeacherHandle, answer)
		if !handled || result.skillShop != nil || len(result.errorMessages) != 1 {
			t.Fatal("old shop did not return retirement notice")
		}
	}
	if len(classicTownVocationByAnswerHandle) != 1 || classicTownVocationByAnswerHandle["job_blade-dancer"] != "刃舞者" {
		t.Fatal("old vocation selection active")
	}
}
