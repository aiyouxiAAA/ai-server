package world

var map1SourceNPCs = []sourceNPCEntry{
	{
		Handle:      "1000542608713897",
		DisplayName: "一心长态",
		SourceQuery: "npc/一心长态.swf",
		SpriteName:  "yixinchangtai",
		Width:       81,
		Height:      117,
		SpawnFlash:  SpawnPoint{X: 320, Y: 534},
		QuestState:  2,
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message: `((一间茅屋从来住，心无是非万境闲。长问尘世谁有路，态意从容若神仙。))
贫道一心长态是也，久居云隐村这风轻云淡之隅，昼来农田耕种，夜来把酒赏月，想来这神仙的生活也不过老夫这般了。
((每日13:00-15:00或者19:00-21:00到我这可以接取生死劫任务))。`,
			Answers: []AnswerOption{
				{Handle: "1q32gs", Msg: "<m/>飞仙洞弑炼"},
				{Handle: "2", Msg: "研习职业"},
				{Handle: "1", Msg: "学习技能"},
				{Handle: "show", Msg: "技能演示"},
				{Handle: "3", Msg: "创建公会"},
				{Handle: "x", Msg: "<c/>关闭"},
			},
		},
	},
	{
		Handle:      "9000542609558425",
		DisplayName: "通天八卦炉<ma>",
		SourceQuery: "npc/通天八卦炉.swf",
		SpriteName:  "bagualu",
		Width:       174,
		Height:      220,
		SpawnFlash:  SpawnPoint{X: 625, Y: 440},
		QuestState:  0,
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message: `八卦炉内燃烧着熊熊烈火......
((如果拥有造物魔晶或特殊布料的话，便可以放入炉内进行冶炼合成稀有的道具。这些材料可以通过打开吉祥袋、如意袋或乾坤袋获得。
注：在【商城】内可以购入【吉祥袋】。))`,
			Answers: []AnswerOption{
				{Handle: "1", Msg: "合成稀有道具"},
				{Handle: "0", Msg: "<c/>取消"},
			},
		},
	},
	{
		Handle:      "2000542608832485",
		DisplayName: "蒙达英",
		SourceQuery: "npc/蒙达英[兄].swf",
		SpriteName:  "mengdaying",
		Width:       57,
		Height:      134,
		SpawnFlash:  SpawnPoint{X: 848, Y: 430},
		QuestState:  3,
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message: `((渺渺云陆春复秋，青山碧水空自流。谁怜我辈英雄志，披肝沥胆万古愁！))
为了照顾弟弟，我放弃了小时候要进入天宫做一番大事的心愿，转而拜姬前辈学打铁。不知道弟弟这火爆的脾气，什么时候能改改呢？`,
			Answers: []AnswerOption{
				{Handle: "1q28gs", Msg: "<m/>防御工事"},
				{Handle: "0", Msg: "<c/>取消"},
			},
		},
	},
	{
		Handle:      "3000542609015823",
		DisplayName: "蒙达悟",
		SourceQuery: "npc/蒙达悟[弟].swf",
		SpriteName:  "mengdawu",
		Width:       57,
		Height:      127,
		SpawnFlash:  SpawnPoint{X: 1092, Y: 410},
		QuestState:  2,
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message: `师父总说：((天外有天，人外有人))，但他心里一直自以为是天下第一铸剑高手。
一心长态村长却说：((云水后浪推前浪，一代新人胜旧人))，我一定要铸一把天下第一剑给大家看看......`,
			Answers: []AnswerOption{
				{Handle: "1q19gs", Msg: "<m/>准备柴火"},
				{Handle: "1", Msg: "购买武器"},
				{Handle: "0", Msg: "<c/>取消"},
			},
		},
	},
	{
		Handle:      "4000542609162635",
		DisplayName: "叶眉",
		SourceQuery: "npc/叶眉.swf",
		SpriteName:  "buyiniang",
		Width:       55,
		Height:      128,
		SpawnFlash:  SpawnPoint{X: 1408, Y: 432},
		QuestState:  0,
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message: `((风吹竹叶飒飒响，眉娘堂前织布忙。衣衫冷暖惟自知，只愿天下无寒霜。))
好想出门见识一下外面的世界啊！若不是身边有爹爹需要供养，我早就像那些男人一样地去闯荡天下了。`,
			Answers: []AnswerOption{
				{Handle: "3q3gs", Msg: "<ml><m/>叶眉的礼物"},
				{Handle: "1", Msg: "购买护具"},
				{Handle: "0", Msg: "<c/>取消"},
			},
		},
	},
	{
		Handle:      "1100542609593728",
		DisplayName: "交易行管理员",
		SourceQuery: "npc/交易行管理员.swf",
		SpriteName:  "jiaoyihang",
		Width:       46,
		Height:      106,
		SpawnFlash:  SpawnPoint{X: 1759, Y: 450},
		QuestState:  0,
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message: `客官来啦？我们((交易行可以为你寄售物品))，有什么宝物需要寄售的吗？
另外你((也可以看看我们这有没有你想要的宝贝))，淘几件回去也不错呀。`,
			Answers: []AnswerOption{
				{Handle: "1", Msg: "查看交易行"},
				{Handle: "0", Msg: "<c/>取消"},
			},
		},
	},
	{
		Handle:      "5000542609232627",
		DisplayName: "卢掌柜",
		SourceQuery: "npc/卢掌柜.swf",
		SpriteName:  "luzhanggui",
		Width:       71,
		Height:      120,
		SpawnFlash:  SpawnPoint{X: 1916, Y: 450},
		QuestState:  2,
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message: `((桂花饼，香气浓，糯米赛玉，红豆如珠。))
欢迎光临云隐客栈，我是卢掌柜。本客栈马上要推出祖传的桂花糯米糕，别忘记了到时候来尝尝。`,
			Answers: []AnswerOption{
				{Handle: "1q22gs", Msg: "<m/>消灭刺鸟"},
				{Handle: "6", Msg: "使用仓库"},
				{Handle: "1", Msg: "住店绑定"},
				{Handle: "2", Msg: "收发信件"},
				{Handle: "0", Msg: "<c/>取消"},
			},
		},
	},
	{
		Handle:      "6000542609425103",
		DisplayName: "南风先生",
		SourceQuery: "npc/南风先生.swf",
		SpriteName:  "nanfeng",
		Width:       48,
		Height:      107,
		SpawnFlash:  SpawnPoint{X: 2228, Y: 423},
		QuestState:  2,
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message: `((一生长命又何求，半生颠沛半生酒。我欲乘风向云去，奈何业已入白首。))
我南风先生颠沛半生，天为被，地为床，酒为食，药为生，不知遇到多少人与事，都如过眼烟云，只觉人生如梦。((这位侠客，15级以前我可以为你免费治疗。))`,
			Answers: []AnswerOption{
				{Handle: "1q21gs", Msg: "<m/>采集草药"},
				{Handle: "2", Msg: "进行治疗"},
				{Handle: "1", Msg: "查看商店"},
				{Handle: "0", Msg: "<c/>取消"},
			},
		},
	},
	{
		Handle:      "1200542609639689",
		DisplayName: "妖术狐狸",
		SourceQuery: "npc/狐狸.swf",
		SpriteName:  "huli",
		Width:       48,
		Height:      48,
		SpawnFlash:  SpawnPoint{X: 2517, Y: 451},
		QuestState:  0,
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message:   `知道((广青镇))吗？它可是方圆百里最繁华的集市哦~`,
			Answers: []AnswerOption{
				{Handle: "2", Msg: "激活本地传送点"},
				{Handle: "3", Msg: "传送到【广青镇】(未激活！)"},
				{Handle: "4", Msg: "传送到【飞仙崖】(未激活！)"},
				{Handle: "help", Msg: "未激活是什么意思？"},
				{Handle: "0", Msg: "<c/>关闭（VIP可获每日免费传送次数）"},
			},
		},
	},
	{
		Handle:      "7000542609490978",
		DisplayName: "丑七品",
		SourceQuery: "npc/七品.swf",
		SpriteName:  "qipin",
		Width:       81,
		Height:      124,
		SpawnFlash:  SpawnPoint{X: 2773, Y: 432},
		QuestState:  2,
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message: `((丑字当头照，利字心头绕。娇娘梦里抱，梦醒数元宝。))
我们丑氏家族世世代代在云之大陆的各个村镇经商。客官，来看看我这里有没有你需要的东西。`,
			Answers: []AnswerOption{
				{Handle: "1q23gs", Msg: "<m/>丑七品的梦"},
				{Handle: "1", Msg: "道具商店"},
				{Handle: "3", Msg: "原石合成"},
				{Handle: "2", Msg: "查看交易行"},
				{Handle: "0", Msg: "<c/>取消"},
			},
		},
	},
	{
		Handle:      "1000542609582165",
		DisplayName: "排行告示",
		SourceQuery: "npc/公告牌.swf",
		SpriteName:  "gonggaopai",
		Width:       136,
		Height:      149,
		SpawnFlash:  SpawnPoint{X: 3000, Y: 400},
		QuestState:  0,
		Dialogue: &sourceNPCDialogueEntry{
			Message: "这里张贴着村里的公告。",
			Answers: []AnswerOption{
				{Handle: "view", Msg: "查看公告"},
				{Handle: "leave", Msg: "离开"},
			},
		},
	},
	{
		Handle:      "8000542609527450",
		DisplayName: "VIP大使",
		SourceQuery: "npc/other/节日大使.swf",
		SpriteName:  "jieridashi",
		Width:       117,
		Height:      151,
		SpawnFlash:  SpawnPoint{X: 2421, Y: 468},
		QuestState:  0,
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message: `到商城购买VIP卡，享受VIP尊贵特权，您还在等什么？心动不如行动！
((当前已经是VIP的，VIP时间不可叠加，更换其他VIP的时候会直接覆盖。VIP提升的经验倍率可以和其他双倍经验卡叠加使用。VIP高级新手套装每个角色只能领取一次！))`,
			Answers: []AnswerOption{
				{Handle: "2", Msg: "领取VIP每日奖品"},
				{Handle: "3", Msg: "领取VIP高级新手套装"},
				{Handle: "4", Msg: "黄牛服务"},
				{Handle: "1", Msg: "充值奖励兑换"},
				{Handle: "0", Msg: "<c/>关闭"},
			},
		},
	},
	{
		Handle:      "transp_4",
		RoleID:      "-3",
		DisplayName: "",
		SourceQuery: "transp/flag2.swf",
		SpriteName:  "flag2",
		Width:       158,
		Height:      258,
		SpawnFlash:  SpawnPoint{X: 3210, Y: 530},
		QuestState:  0,
		Dialogue: &sourceNPCDialogueEntry{
			Message: "可以瞬间传送至该处。",
			Answers: []AnswerOption{
				{Handle: "goto", Msg: "前往"},
				{Handle: "leave", Msg: "离开"},
			},
		},
	},
}

var sourceTransportDialogue = sourceNPCDialogueEntry{
	Message: "可以瞬间传送至该处。",
	Answers: []AnswerOption{
		{Handle: "goto", Msg: "前往"},
		{Handle: "leave", Msg: "离开"},
	},
}

var yunyinSourceTransportLinks = []sourceTransportLink{
	{FromMapID: 4, ToMapID: 1, Slot: 0},
	{FromMapID: 4, ToMapID: 5, Slot: 1},
	{FromMapID: 5, ToMapID: 4, Slot: 0},
	{FromMapID: 5, ToMapID: 9, Slot: 1},
	{FromMapID: 9, ToMapID: 5, Slot: 0},
	{FromMapID: 9, ToMapID: 13, Slot: 1},
	{FromMapID: 13, ToMapID: 9, Slot: 0},
	{FromMapID: 13, ToMapID: 14, Slot: 1},
	{FromMapID: 13, ToMapID: 19, Slot: 2},
	{FromMapID: 14, ToMapID: 13, Slot: 0},
	{FromMapID: 14, ToMapID: 15, Slot: 1},
	{FromMapID: 15, ToMapID: 14, Slot: 0},
	{FromMapID: 15, ToMapID: 16, Slot: 1},
	{FromMapID: 16, ToMapID: 15, Slot: 0},
	{FromMapID: 16, ToMapID: 17, Slot: 1},
	{FromMapID: 17, ToMapID: 16, Slot: 0},
	{FromMapID: 17, ToMapID: 18, Slot: 1},
	{FromMapID: 18, ToMapID: 17, Slot: 0},
	{FromMapID: 19, ToMapID: 13, Slot: 0},
	{FromMapID: 19, ToMapID: 20, Slot: 1},
	{FromMapID: 20, ToMapID: 19, Slot: 0},
	{FromMapID: 20, ToMapID: 21, Slot: 1},
	{FromMapID: 21, ToMapID: 20, Slot: 0},
	{FromMapID: 21, ToMapID: 22, Slot: 1},
	{FromMapID: 21, ToMapID: 25, Slot: 2},
	{FromMapID: 22, ToMapID: 21, Slot: 0},
	{FromMapID: 22, ToMapID: 23, Slot: 1},
	{FromMapID: 23, ToMapID: 22, Slot: 0},
	{FromMapID: 23, ToMapID: 24, Slot: 1},
	{FromMapID: 24, ToMapID: 23, Slot: 0},
	{FromMapID: 24, ToMapID: 27, Slot: 1},
	{FromMapID: 25, ToMapID: 21, Slot: 0},
	{FromMapID: 25, ToMapID: 30, Slot: 1},
	{FromMapID: 26, ToMapID: 30, Slot: 0},
	{FromMapID: 26, ToMapID: 28, Slot: 1},
	{FromMapID: 27, ToMapID: 24, Slot: 0},
	{FromMapID: 27, ToMapID: 29, Slot: 1},
	{FromMapID: 28, ToMapID: 26, Slot: 0},
	{FromMapID: 28, ToMapID: 32, Slot: 1},
	{FromMapID: 29, ToMapID: 27, Slot: 0},
	{FromMapID: 30, ToMapID: 25, Slot: 0},
	{FromMapID: 30, ToMapID: 26, Slot: 1},
	{FromMapID: 32, ToMapID: 28, Slot: 0},
}

var map1SourceNPCDialogueReplies = map[sourceNPCDialogueReplyKey]sourceNPCDialogueEntry{
	{
		Handle:       "1000542608713897",
		MsgHandle:    "1",
		AnswerHandle: "2",
	}: {
		MsgHandle: "2",
		Message:   "现在有三个职业可以选择，请从职业选项中选择你希望的职业......",
		Answers: []AnswerOption{
			{Handle: "4", Msg: "了解【战士】的特点"},
			{Handle: "5", Msg: "了解【术士】的特点"},
			{Handle: "6", Msg: "了解【游侠】的特点"},
			{Handle: "0", Msg: "<c/>返回"},
		},
	},
	{
		Handle:       "1000542608713897",
		MsgHandle:    "2",
		AnswerHandle: "4",
	}: {
		MsgHandle: "13",
		Message:   "<font color='#990000'><b>战士</b></font>若有较强的气力以及物理攻击和防御。弱点在于遇到法术攻击的时候较为吃力。",
		Answers: []AnswerOption{
			{Handle: "0", Msg: "<c/>返回"},
		},
	},
	{
		Handle:       "1000542608713897",
		MsgHandle:    "2",
		AnswerHandle: "5",
	}: {
		MsgHandle: "14",
		Message:   "<font color='#990000'><b>术士</b></font>这个职业拥有强大的法术攻击和防御能力，但使用的武器类型范围较窄。",
		Answers: []AnswerOption{
			{Handle: "0", Msg: "<c/>返回"},
		},
	},
	{
		Handle:       "1000542608713897",
		MsgHandle:    "2",
		AnswerHandle: "6",
	}: {
		MsgHandle: "15",
		Message:   "<font color='#990000'><b>游侠</b></font>具有其他职业无法比拟的高敏捷特性，可研习提高物理攻击的技能。",
		Answers: []AnswerOption{
			{Handle: "0", Msg: "<c/>返回"},
		},
	},
	{
		Handle:       "1000542608713897",
		MsgHandle:    "13",
		AnswerHandle: "0",
	}: {
		MsgHandle: "1",
		Message:   "",
		Answers:   nil,
	},
	{
		Handle:       "1000542608713897",
		MsgHandle:    "14",
		AnswerHandle: "0",
	}: {
		MsgHandle: "1",
		Message:   "",
		Answers:   nil,
	},
	{
		Handle:       "1000542608713897",
		MsgHandle:    "15",
		AnswerHandle: "0",
	}: {
		MsgHandle: "1",
		Message:   "",
		Answers:   nil,
	},
	{
		Handle:       "1000542608713897",
		MsgHandle:    "2",
		AnswerHandle: "0",
	}: {
		MsgHandle: "1",
		Message:   "",
		Answers:   nil,
	},
	{
		Handle:       "1000542608713897",
		MsgHandle:    "1",
		AnswerHandle: "1",
	}: {
		MsgHandle: "10",
		Message:   "你想学习什么职业的技能？",
		Answers: []AnswerOption{
			{Handle: "7", Msg: "战士技能"},
			{Handle: "8", Msg: "术士技能"},
			{Handle: "9", Msg: "游侠技能"},
			{Handle: "0", Msg: "<c/>返回"},
		},
	},
	{
		Handle:       "1000542608713897",
		MsgHandle:    "10",
		AnswerHandle: "0",
	}: {
		MsgHandle: "1",
		Message:   "",
		Answers:   nil,
	},
	{
		Handle:       "1000542608713897",
		MsgHandle:    "1",
		AnswerHandle: "show",
	}: {
		MsgHandle: "sshow",
		Message:   "请选择一个职业技能进行预览。",
		Answers: []AnswerOption{
			{Handle: "show01", Msg: "[战士]奥义.飘血"},
			{Handle: "show02", Msg: "[术士]白猿之怒"},
			{Handle: "show03", Msg: "[游侠]奥义.修罗幻翼拳"},
			{Handle: "x", Msg: "<c/>关闭"},
		},
	},
	{
		Handle:       "1000542608713897",
		MsgHandle:    "sshow",
		AnswerHandle: "show01",
	}: {
		MsgHandle: "shows01",
		Message:   "((【战士.剑系】的【奥义.飘血】技能展示。))[img]ect/px.swf[/img]",
		Answers: []AnswerOption{
			{Handle: "0", Msg: "<c/>返回"},
		},
	},
	{
		Handle:       "1000542608713897",
		MsgHandle:    "sshow",
		AnswerHandle: "show02",
	}: {
		MsgHandle: "shows02",
		Message:   "((【战士.法杖】的【白猿之怒】技能展示。))[img]ect/byzn.swf[/img]",
		Answers: []AnswerOption{
			{Handle: "0", Msg: "<c/>返回"},
		},
	},
	{
		Handle:       "1000542608713897",
		MsgHandle:    "sshow",
		AnswerHandle: "show03",
	}: {
		MsgHandle: "shows03",
		Message:   "((【游侠.拳系】的【奥义.修罗幻翼拳】技能展示。))[img]ect/xlhy.swf[/img]",
		Answers: []AnswerOption{
			{Handle: "0", Msg: "<c/>返回"},
		},
	},
	{
		Handle:       "1000542608713897",
		MsgHandle:    "shows01",
		AnswerHandle: "0",
	}: {
		MsgHandle: "1",
		Message:   "",
		Answers:   nil,
	},
	{
		Handle:       "1000542608713897",
		MsgHandle:    "shows02",
		AnswerHandle: "0",
	}: {
		MsgHandle: "1",
		Message:   "",
		Answers:   nil,
	},
	{
		Handle:       "1000542608713897",
		MsgHandle:    "shows03",
		AnswerHandle: "0",
	}: {
		MsgHandle: "1",
		Message:   "",
		Answers:   nil,
	},
	{
		Handle:       "4000542609162635",
		MsgHandle:    "1",
		AnswerHandle: "3q3gs",
	}: {
		MsgHandle: "3q3d_1",
		Message:   "你来啦？一个人出门在外不容易，之前看你穿的单薄，特意给你做了套布衣裤，另外还有一些花卷，就算我送你的礼物吧。",
		Answers: []AnswerOption{
			{Handle: "3q3a_1_1", Msg: "<m/>谢谢您啦。"},
			{Handle: "3q3a_1_2", Msg: "<c/>关闭"},
		},
	},
	{
		Handle:       "2000542608832485",
		MsgHandle:    "1",
		AnswerHandle: "1q28gs",
	}: {
		MsgHandle: "1q28d_1",
		Message: `我刚刚听广青镇来的人说魔军已经打到白源那边了，我们正在修筑防御工事，魔军若来了也能抵御一阵。现在还需要很多木材，你((去【树海_1】的木材采集点采20根木材))过来吧。((【注：该任务2小时一周期】))

[g]=[i=f_i_银元宝^C156C7&24@材料 消耗品&25@9999&19@双击可兑换为1000铜币&20@游戏中的货币&0;用于流通买卖&27@sitem_jhj&101@39.png&103@0&104@0&105@&107@&108@0]银元宝[/]x1`,
		Answers: []AnswerOption{
			{Handle: "1q28a_1_1", Msg: "<m/>我这就去。"},
			{Handle: "1q28a_1_2", Msg: "<c/>关闭"},
		},
	},
}
