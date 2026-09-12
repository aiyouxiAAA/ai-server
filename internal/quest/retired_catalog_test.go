package quest

import (
	"os"
	"testing"
)

// Historical parser evidence must not repopulate the production quest provider.
func retiredCatalog() []Info {
	file, err := os.Open("testdata/classic_quest_catalog.retired.csv")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	rows, err := parseCatalog(file)
	if err != nil {
		panic(err)
	}
	return rows
}
func retiredFindByID(id string) (Info, bool) {
	for _, q := range retiredCatalog() {
		if q.ID == id {
			return q, true
		}
	}
	return Info{}, false
}
func retiredMapRoutes() []ClassicMapNPCQuestRoute {
	return classicMapNPCQuestRoutesFor(retiredCatalog())
}
func retiredWuliangRoutes() []WuliangMapNPCQuestRoute {
	result := []WuliangMapNPCQuestRoute{}
	for _, r := range retiredMapRoutes() {
		if r.MapID == 190 || r.MapID == 191 {
			result = append(result, r)
		}
	}
	return result
}
func TestAuthoredCatalogReplacesClassic(t *testing.T) {
	if len(All()) != 6 || len(retiredCatalog()) != 353 || len(ClassicMapNPCQuestRoutes()) != 0 {
		t.Fatal("old quests remain active or archive incomplete")
	}
	for _, q := range All() {
		if _, ok := FindAuthored(q.ID); !ok {
			t.Fatal("non-authored quest", q.ID)
		}
	}
	if _, ok := FindByID("capture-001"); ok {
		t.Fatal("retired quest still found")
	}
	for _, g := range Guides() {
		if _, ok := FindAuthored(g.QuestID); !ok {
			t.Fatal("orphan guide")
		}
	}
}
