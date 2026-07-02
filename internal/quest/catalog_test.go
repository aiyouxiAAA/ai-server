package quest

import "testing"

func TestCatalogParsesCapturedQuestRewards(t *testing.T) {
	info, ok := FindByID("capture-003")
	if !ok {
		t.Fatal("expected captured quest capture-003")
	}
	if info.Reward.Experience != 1000 {
		t.Fatalf("expected capture-003 exp reward 1000, got %+v", info.Reward)
	}
	if len(info.Reward.Items) != 1 || info.Reward.Items[0].Name != "铜钱" || info.Reward.Items[0].Count != 200 || info.Reward.Items[0].Display != "163.png" {
		t.Fatalf("expected capture-003 copper reward, got %+v", info.Reward.Items)
	}
}

func TestCatalogBuildsQuestRewardEntries(t *testing.T) {
	info, ok := FindByID("capture-003")
	if !ok {
		t.Fatal("expected captured quest capture-003")
	}
	expEntry, ok := findRewardEntry(info.RewardEntries, RewardEntryPhaseGrant, RewardEntryKindExperience, "经验")
	if !ok {
		t.Fatalf("expected capture-003 experience reward entry, got %+v", info.RewardEntries)
	}
	if expEntry.SourceType != RewardEntrySourceQuest || expEntry.SourceID != "capture-003" || expEntry.CountMin != 1000 || expEntry.CountMax != 1000 || expEntry.Probability != RewardEntryProbabilityCertain {
		t.Fatalf("expected deterministic quest exp entry, got %+v", expEntry)
	}
	copperEntry, ok := findRewardEntry(info.RewardEntries, RewardEntryPhaseGrant, RewardEntryKindCurrency, "铜钱")
	if !ok {
		t.Fatalf("expected capture-003 copper reward entry, got %+v", info.RewardEntries)
	}
	if copperEntry.SourceType != RewardEntrySourceQuest || copperEntry.SourceID != "capture-003" || copperEntry.CountMin != 200 || copperEntry.CountMax != 200 || copperEntry.Probability != RewardEntryProbabilityCertain || copperEntry.Display != "163.png" {
		t.Fatalf("expected deterministic quest currency entry, got %+v", copperEntry)
	}
}

func TestCatalogParsesQuestRequirements(t *testing.T) {
	info, ok := FindByID("capture-032")
	if !ok {
		t.Fatal("expected capture-032 in catalog")
	}
	if len(info.Requirements) != 1 || info.Requirements[0].Name != "肉" || info.Requirements[0].Count != 5 || info.Requirements[0].Display != "70.png" {
		t.Fatalf("expected capture-032 meat requirement, got %+v", info.Requirements)
	}
}

func TestCatalogParsesSkillAndOptionalRewards(t *testing.T) {
	skillInfo, ok := FindByID("capture-007")
	if !ok {
		t.Fatal("expected captured quest capture-007")
	}
	if skillInfo.Reward.Experience != 300 || len(skillInfo.Reward.Skills) != 1 || skillInfo.Reward.Skills[0].Name != "密斩" {
		t.Fatalf("expected capture-007 exp and skill reward, got %+v", skillInfo.Reward)
	}

	optionalInfo, ok := FindByID("capture-058")
	if !ok {
		t.Fatal("expected captured quest capture-058")
	}
	if optionalInfo.Reward.Experience != 5000 {
		t.Fatalf("expected capture-058 exp reward 5000, got %+v", optionalInfo.Reward)
	}
	if len(optionalInfo.Reward.Items) != 0 {
		t.Fatalf("expected capture-058 [g] section to avoid required-item rewards, got %+v", optionalInfo.Reward.Items)
	}
	if len(optionalInfo.Reward.OptionalItems) != 6 {
		t.Fatalf("expected capture-058 six optional pet rewards, got %+v", optionalInfo.Reward.OptionalItems)
	}
	grantItems := 0
	optionalItems := 0
	for _, entry := range optionalInfo.RewardEntries {
		if entry.Phase == RewardEntryPhaseGrant && entry.Kind == RewardEntryKindItem {
			grantItems++
		}
		if entry.Phase == RewardEntryPhaseOptional && entry.Kind == RewardEntryKindItem {
			optionalItems++
			if entry.ChooseGroup != "capture-058:optional" || entry.Probability != RewardEntryProbabilityCertain {
				t.Fatalf("expected optional reward choice entry, got %+v", entry)
			}
		}
	}
	if grantItems != 0 || optionalItems != 6 {
		t.Fatalf("expected capture-058 optional rewards to stay out of grant entries, got %+v", optionalInfo.RewardEntries)
	}
}

func TestCatalogParsesWuliangRoutes(t *testing.T) {
	info, ok := FindByID("capture-186")
	if !ok {
		t.Fatal("expected captured Wuliang quest capture-186")
	}
	if info.Title != "侦查敌营" || info.QuestStateHandle != "6350542618650282" {
		t.Fatalf("expected Wuliang quest title and state handle, got %+v", info)
	}
	if len(info.Routes) != 2 {
		t.Fatalf("expected accept and completion routes, got %+v", info.Routes)
	}
	if info.Routes[0].Handle != "6350542618650282" || info.Routes[0].MsgHandle != "6q2d_1" || info.Routes[0].AnswerHandle != "6q2a_1_1" {
		t.Fatalf("expected Wuliang accept route, got %+v", info.Routes[0])
	}
	if info.Reward.Experience != 15000 {
		t.Fatalf("expected Wuliang exp reward 15000, got %+v", info.Reward)
	}
	if len(info.Reward.Items) != 1 || info.Reward.Items[0].Name != "银元宝" || info.Reward.Items[0].Count != 2 || info.Reward.Items[0].Display != "39.png" {
		t.Fatalf("expected Wuliang silver reward, got %+v", info.Reward.Items)
	}
}

func TestCatalogParsesCapturedDafoWoodMonsterRoute(t *testing.T) {
	info, ok := FindByID("capture-024")
	if !ok {
		t.Fatal("expected captured Dafo quest capture-024")
	}
	if info.Title != "讨厌的枯木怪" || info.QuestStateHandle != "4090542614314425" {
		t.Fatalf("expected Dafo quest title and state handle, got %+v", info)
	}
	if len(info.Routes) != 1 {
		t.Fatalf("expected captured Dafo accept route, got %+v", info.Routes)
	}
	route := info.Routes[0]
	if route.Handle != "4090542614314425" || route.MsgHandle != "2q23d_1" || route.AnswerHandle != "2q23a_1_1" {
		t.Fatalf("expected captured Dafo accept route, got %+v", route)
	}
}

func TestCatalogIncludesVisibleWuliangQuestChain(t *testing.T) {
	expected := map[string]string{
		"capture-185": "乌梁军策",
		"capture-186": "侦查敌营",
		"capture-187": "玄机之件",
		"capture-188": "拜访故人",
		"capture-189": "回赠特产",
		"capture-190": "暗力之源",
		"capture-191": "乌梁工匠",
		"capture-192": "水车图纸",
		"capture-193": "乌梁前锋营",
		"capture-194": "求签",
		"capture-195": "乌梁药师",
		"capture-196": "草药之策",
		"capture-197": "营长之令",
		"capture-198": "机木锥兵",
		"capture-199": "军令如山",
		"capture-200": "五花香肉",
		"capture-201": "受伤的部位",
		"capture-202": "汉雄的疑惑",
		"capture-203": "造坝截水",
		"capture-204": "水坝建成",
		"capture-205": "新的指令",
		"capture-206": "修复石坝",
		"capture-207": "建造水车",
		"capture-208": "精巧零件",
		"capture-209": "大义之举",
		"capture-210": "营救袁碧寰",
	}
	for id, title := range expected {
		info, ok := FindByID(id)
		if !ok {
			t.Fatalf("expected visible Wuliang quest %s %s", id, title)
		}
		if info.Title != title {
			t.Fatalf("expected %s title %s, got %s", id, title, info.Title)
		}
	}

	xuanji, _ := FindByID("capture-187")
	if len(xuanji.Routes) != 2 || xuanji.Routes[1].MsgHandle != "6q3d_2" || xuanji.Routes[1].AnswerHandle != "6q3a_2_1" {
		t.Fatalf("expected captured Xuanji completion route, got %+v", xuanji.Routes)
	}
	specialty, _ := FindByID("capture-189")
	if len(specialty.Routes) != 2 || specialty.Routes[1].Handle != "6350542618650282" || specialty.Routes[1].MsgHandle != "6q41d_2" {
		t.Fatalf("expected captured specialty delivery route, got %+v", specialty.Routes)
	}
	waterwheel, _ := FindByID("capture-207")
	if len(waterwheel.Routes) != 0 || waterwheel.QuestStateHandle != "6370542618853300" {
		t.Fatalf("expected waterwheel quest table-only gap on Yumo, got %+v", waterwheel)
	}
}

func TestCatalogIncludesExtendedBaiyuanQuestChain(t *testing.T) {
	expected := map[string]string{
		"capture-211": "赤红之器",
		"capture-212": "有名的媒婆",
		"capture-213": "千里姻缘",
		"capture-214": "雷兽熊鹿",
		"capture-215": "收集石块",
		"capture-216": "剿木行动",
		"capture-217": "再返白源",
		"capture-218": "被困的少女",
		"capture-219": "袁大小姐",
		"capture-220": "机木玄师",
		"capture-221": "竣工喜讯",
		"capture-222": "护送回家",
		"capture-223": "丑四品的遗憾",
		"capture-224": "灵异事件",
		"capture-225": "五子棋大战",
		"capture-226": "石头密语",
		"capture-227": "离家出游",
		"capture-228": "赚钱的买卖",
		"capture-229": "机木玄文",
		"capture-230": "胜利的曙光",
		"capture-231": "机木妖帅",
		"capture-232": "打造家具",
		"capture-233": "辅料木材",
		"capture-234": "家具新革命",
		"capture-235": "投资工厂",
		"capture-236": "难做的买卖",
		"capture-237": "猎杀蚩颅王",
		"capture-238": "狮虎三魁",
	}
	for id, title := range expected {
		info, ok := FindByID(id)
		if !ok {
			t.Fatalf("expected extended captured quest %s %s", id, title)
		}
		if info.Title != title {
			t.Fatalf("expected %s title %s, got %s", id, title, info.Title)
		}
		if info.Reward.Experience <= 0 && len(info.Reward.Items) == 0 && len(info.Reward.Skills) == 0 {
			t.Fatalf("expected %s %s captured reward, got %+v", id, title, info.Reward)
		}
	}

	redArtifact, _ := FindByID("capture-211")
	if len(redArtifact.Routes) != 0 || redArtifact.QuestStateHandle != "" {
		t.Fatalf("expected red artifact to stay QuestInfo-only until c_Speak evidence appears, got %+v", redArtifact)
	}
	rescue, _ := FindByID("capture-218")
	if len(rescue.Routes) != 2 || rescue.Routes[0].AnswerHandle != "6q34gs" || rescue.Routes[1].MsgHandle != "6q34d_2" {
		t.Fatalf("expected captured Yuan rescue routes, got %+v", rescue.Routes)
	}
	furniture, _ := FindByID("capture-232")
	if len(furniture.Routes) != 2 || furniture.Routes[0].Handle != "6190542618476150" || furniture.Routes[1].Handle != "6370542618853300" {
		t.Fatalf("expected captured furniture routes, got %+v", furniture.Routes)
	}
	shihu, _ := FindByID("capture-238")
	if len(shihu.Routes) != 2 || shihu.Routes[0].AnswerHandle != "4q51a_1_1" || shihu.Routes[1].AnswerHandle != "4q51a_2_1" {
		t.Fatalf("expected captured Shihuku quest routes, got %+v", shihu.Routes)
	}
}

func TestCatalogIncludesDateSweepBaiyuanAndSwampQuestChain(t *testing.T) {
	expected := map[string]string{
		"capture-239": "全新护具",
		"capture-240": "加工水晶项链",
		"capture-241": "大战赤蛰子",
		"capture-242": "女儿的噩耗",
		"capture-243": "女儿的礼物",
		"capture-244": "寻找线索",
		"capture-245": "怒不可遏",
		"capture-246": "探望女儿",
		"capture-247": "最后的心愿",
		"capture-248": "注定孤苦",
		"capture-249": "王花的花蕾",
		"capture-250": "珍贵的宝贝",
		"capture-251": "缺少原料",
		"capture-252": "项链做成",
		"capture-253": "埋伏之战",
		"capture-254": "收集毒囊",
		"capture-255": "根治之法",
		"capture-256": "治疗心病",
		"capture-257": "白须老翁",
		"capture-258": "花泪解药",
		"capture-259": "药到病除",
		"capture-260": "新鲜活力",
	}
	for id, title := range expected {
		info, ok := FindByID(id)
		if !ok {
			t.Fatalf("expected date-sweep captured quest %s %s", id, title)
		}
		if info.Title != title {
			t.Fatalf("expected %s title %s, got %s", id, title, info.Title)
		}
		if info.Reward.Experience <= 0 && len(info.Reward.Items) == 0 && len(info.Reward.Skills) == 0 {
			t.Fatalf("expected %s %s captured reward, got %+v", id, title, info.Reward)
		}
	}

	clue, _ := FindByID("capture-244")
	if len(clue.Routes) != 2 || clue.Routes[0].AnswerHandle != "5q51gs" || clue.Routes[1].MsgHandle != "5q51d_2" {
		t.Fatalf("expected captured clue quest accept and completion routes, got %+v", clue.Routes)
	}
	lastWish, _ := FindByID("capture-247")
	if len(lastWish.Routes) != 2 || lastWish.Routes[0].Handle != "5300542617580783" || lastWish.Routes[1].Handle != "4710542615621525" {
		t.Fatalf("expected captured last-wish cross-NPC routes, got %+v", lastWish.Routes)
	}
	poison, _ := FindByID("capture-254")
	if len(poison.Routes) != 2 || poison.Routes[0].AnswerHandle != "aq30gs" || poison.Routes[1].AnswerHandle != "aq30os" {
		t.Fatalf("expected captured poison-sac routes, got %+v", poison.Routes)
	}
	fresh, _ := FindByID("capture-260")
	if len(fresh.Routes) != 2 || fresh.Routes[0].Handle != "1810542611191117" || fresh.Routes[1].Handle != "6190542618476150" {
		t.Fatalf("expected captured fresh-vitality routes, got %+v", fresh.Routes)
	}
}

func TestAllCatalogRowsHaveGrantRewardMarker(t *testing.T) {
	for _, info := range All() {
		if info.Reward.Experience <= 0 && len(info.Reward.Items) == 0 && len(info.Reward.Skills) == 0 {
			t.Fatalf("expected parsed grant reward for %s %s, got %+v", info.ID, info.Title, info.Reward)
		}
	}
}

func TestAllQuestRewardEntriesAreLinkedToCatalogRows(t *testing.T) {
	entries := AllRewardEntries()
	if len(entries) == 0 {
		t.Fatal("expected quest reward entry table")
	}
	for _, entry := range entries {
		if entry.SourceType != RewardEntrySourceQuest || entry.SourceID == "" {
			t.Fatalf("expected reward entry to link back to quest row, got %+v", entry)
		}
		if entry.Probability != RewardEntryProbabilityCertain {
			t.Fatalf("expected quest reward entry to be deterministic, got %+v", entry)
		}
		if entry.CountMin <= 0 || entry.CountMax < entry.CountMin {
			t.Fatalf("expected valid quest reward count range, got %+v", entry)
		}
		if entry.Phase == RewardEntryPhaseOptional && entry.ChooseGroup == "" {
			t.Fatalf("expected optional reward entry choose group, got %+v", entry)
		}
	}
}

func findRewardEntry(entries []RewardEntry, phase RewardEntryPhase, kind RewardEntryKind, name string) (RewardEntry, bool) {
	for _, entry := range entries {
		if entry.Phase == phase && entry.Kind == kind && entry.Name == name {
			return entry, true
		}
	}
	return RewardEntry{}, false
}
