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
		DisplayName: "布衣娘",
		SourceQuery: "npc/布衣娘.swf",
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
				{Handle: "3q3gs", Msg: "<ml><m/>布衣娘的礼物"},
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

var map2SourceNPCs = []sourceNPCEntry{
	{
		Handle:      "4180542615109515",
		DisplayName: "交易行管理员",
		SourceQuery: "npc/交易行管理员.swf",
		SpriteName:  "jiaoyihang",
		Width:       46,
		Height:      106,
		SpawnFlash:  SpawnPoint{X: 2162, Y: 465},
		QuestState:  0,
	},
	{
		Handle:      "4190542615111877",
		DisplayName: "通天八卦炉<ma>",
		SourceQuery: "npc/通天八卦炉.swf",
		SpriteName:  "bagualu",
		Width:       174,
		Height:      220,
		SpawnFlash:  SpawnPoint{X: 780, Y: 465},
		QuestState:  0,
	},
	{
		Handle:      "4130542614832797",
		DisplayName: "炎衣娘",
		SourceQuery: "npc/炎衣娘.swf",
		SpriteName:  "yanyiniang",
		Width:       55,
		Height:      128,
		SpawnFlash:  SpawnPoint{X: 1026, Y: 481},
		QuestState:  2,
	},
	{
		Handle:      "4110542614676637",
		DisplayName: "无颜",
		SourceQuery: "npc/无颜.swf",
		SpriteName:  "wuyan",
		Width:       48,
		Height:      107,
		SpawnFlash:  SpawnPoint{X: 1781, Y: 463},
		QuestState:  1,
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message:   `((凡事知足心长乐，人到无求品自高。))<br>我乃药师无颜，有病治病，无病医心。头疼脑热只是小事。`,
			Answers: []AnswerOption{
				{Handle: "2q21gs", Msg: "<m/>采集草药"},
				{Handle: "4q48os", Msg: "<m/>寻找妙方"},
				{Handle: "4q69gs", Msg: "<m/>奇珍雪莲"},
				{Handle: "2", Msg: "进行治疗"},
				{Handle: "1", Msg: "查看商店"},
				{Handle: "0", Msg: "<c/>取消"},
			},
		},
	},
	{
		Handle:      "4160542615098838",
		DisplayName: "排行告示",
		SourceQuery: "npc/公告牌.swf",
		SpriteName:  "gonggaopai",
		Width:       136,
		Height:      149,
		SpawnFlash:  SpawnPoint{X: 2670, Y: 285},
		QuestState:  0,
	},
	{
		Handle:      "4090542614314425",
		DisplayName: "丑六品",
		SourceQuery: "npc/六品.swf",
		SpriteName:  "liupin",
		Width:       81,
		Height:      124,
		SpawnFlash:  SpawnPoint{X: 2868, Y: 310},
		QuestState:  2,
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message:   `本店供应各种日常货物，请随意挑选。唉~最近想起我那七弟来了，我们兄弟为了做生意终日奔波各处...`,
			Answers: []AnswerOption{
				{Handle: "4q72os", Msg: "<m/>丑七品的梦"},
				{Handle: "2q23gs", Msg: "<m/>丑家兄弟"},
				{Handle: "4q73gs", Msg: "<m/>广青镇相聚"},
				{Handle: "1", Msg: "道具商店"},
				{Handle: "0", Msg: "<c/>取消"},
			},
		},
	},
	{
		Handle:      "4120542614739860",
		DisplayName: "阿达奴",
		SourceQuery: "npc/阿达奴.swf",
		SpriteName:  "adanu",
		Width:       57,
		Height:      135,
		SpawnFlash:  SpawnPoint{X: 1413, Y: 463},
		QuestState:  2,
	},
	{
		Handle:      "4170542615108676",
		DisplayName: "妖术狐狸",
		SourceQuery: "npc/狐狸.swf",
		SpriteName:  "huli",
		Width:       48,
		Height:      48,
		SpawnFlash:  SpawnPoint{X: 2370, Y: 465},
		QuestState:  0,
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message:   `知道((广青镇))吗？它可是方圆百里最繁华的集市哦~`,
			Answers: []AnswerOption{
				{Handle: "2", Msg: "激活本地传送点"},
				{Handle: "3", Msg: "传送到【广青镇】(未激活！)"},
				{Handle: "4", Msg: "传送到【黄风寨口】(未激活！)"},
				{Handle: "help", Msg: "未激活是什么意思？"},
				{Handle: "0", Msg: "<c/>关闭（VIP可获每日免费传送次数）"},
			},
		},
	},
	{
		Handle:      "4150542615092700",
		DisplayName: "VIP大使",
		SourceQuery: "npc/other/节日大使.swf",
		SpriteName:  "jieridashi",
		Width:       117,
		Height:      151,
		SpawnFlash:  SpawnPoint{X: 306, Y: 615},
		QuestState:  0,
	},
	{
		Handle:      "4140542615070416",
		DisplayName: "噌痴",
		SourceQuery: "npc/噌痴.swf",
		SpriteName:  "cengchi",
		Width:       81,
		Height:      117,
		SpawnFlash:  SpawnPoint{X: 407, Y: 360},
		QuestState:  2,
	},
	{
		Handle:      "4100542614427315",
		DisplayName: "娴无禄",
		SourceQuery: "npc/娴无录.swf",
		SpriteName:  "xianwulu",
		Width:       57,
		Height:      134,
		SpawnFlash:  SpawnPoint{X: 2450, Y: 400},
		QuestState:  3,
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message:   `这位小侠，可是要打造两件趁手的兵器？！我可是当年云之大陆有名的铸剑高人之后，哼哼！别怪我没提醒你。`,
			Answers: []AnswerOption{
				{Handle: "2q28gs", Msg: "<m/>兵器打造"},
				{Handle: "0", Msg: "<c/>取消"},
			},
		},
	},
	{
		Handle:      "transp_6",
		RoleID:      "-3",
		DisplayName: "",
		SourceQuery: "transp/flag2.swf",
		SpriteName:  "flag2",
		Width:       158,
		Height:      258,
		SpawnFlash:  SpawnPoint{X: 3000, Y: 500},
		QuestState:  0,
		Dialogue:    &sourceTransportDialogue,
	},
}

var map3SourceNPCs = []sourceNPCEntry{
	{
		Handle:      "4970542616788530",
		DisplayName: "VIP大使",
		SourceQuery: "npc/other/节日大使.swf",
		SpriteName:  "jieridashi",
		Width:       117,
		Height:      151,
		SpawnFlash:  SpawnPoint{X: 372, Y: 595},
		QuestState:  0,
	},
	{
		Handle:      "4910542615957836",
		DisplayName: "申公烈",
		SourceQuery: "npc/申公烈.swf",
		SpriteName:  "shengonglie",
		Width:       108,
		Height:      116,
		SpawnFlash:  SpawnPoint{X: 675, Y: 545},
		QuestState:  0,
	},
	{
		Handle:      "4920542616052493",
		DisplayName: "火娥娘",
		SourceQuery: "npc/火娥娘.swf",
		SpriteName:  "huoeniang",
		Width:       57,
		Height:      135,
		SpawnFlash:  SpawnPoint{X: 1180, Y: 409},
		QuestState:  0,
	},
	{
		Handle:      "5010542616817526",
		DisplayName: "交易行管理员",
		SourceQuery: "npc/交易行管理员.swf",
		SpriteName:  "jiaoyihang",
		Width:       46,
		Height:      106,
		SpawnFlash:  SpawnPoint{X: 1612, Y: 420},
		QuestState:  0,
	},
	{
		Handle:      "4930542616250587",
		DisplayName: "铃铛",
		SourceQuery: "npc/铃铛.swf",
		SpriteName:  "lingdang",
		Width:       72,
		Height:      135,
		SpawnFlash:  SpawnPoint{X: 1829, Y: 426},
		QuestState:  0,
	},
	{
		Handle:      "4990542616803864",
		DisplayName: "通天八卦炉<ma>",
		SourceQuery: "npc/通天八卦炉.swf",
		SpriteName:  "bagualu",
		Width:       174,
		Height:      220,
		SpawnFlash:  SpawnPoint{X: 2139, Y: 430},
		QuestState:  0,
	},
	{
		Handle:      "5000542616815700",
		DisplayName: "妖术狐狸",
		SourceQuery: "npc/狐狸.swf",
		SpriteName:  "huli",
		Width:       48,
		Height:      48,
		SpawnFlash:  SpawnPoint{X: 2370, Y: 465},
		QuestState:  0,
	},
	{
		Handle:      "4940542616468969",
		DisplayName: "叶眉",
		SourceQuery: "npc/叶眉.swf",
		SpriteName:  "yemei",
		Width:       65,
		Height:      156,
		SpawnFlash:  SpawnPoint{X: 2589, Y: 465},
		QuestState:  0,
	},
	{
		Handle:      "4950542616589339",
		DisplayName: "熊猫竹生",
		SourceQuery: "npc/熊猫竹生.swf",
		SpriteName:  "xiongmaozhusheng",
		Width:       74,
		Height:      90,
		SpawnFlash:  SpawnPoint{X: 2862, Y: 426},
		QuestState:  0,
	},
	{
		Handle:      "4960542616750900",
		DisplayName: "介象",
		SourceQuery: "npc/介象.swf",
		SpriteName:  "jiexiang",
		Width:       84,
		Height:      146,
		SpawnFlash:  SpawnPoint{X: 3051, Y: 442},
		QuestState:  0,
	},
	{
		Handle:      "4980542616799322",
		DisplayName: "排行告示",
		SourceQuery: "npc/公告牌.swf",
		SpriteName:  "gonggaopai",
		Width:       136,
		Height:      149,
		SpawnFlash:  SpawnPoint{X: 3325, Y: 420},
		QuestState:  0,
	},
	{
		Handle:      "transp_10",
		RoleID:      "-3",
		DisplayName: "",
		SourceQuery: "transp/flag2.swf",
		SpriteName:  "flag2",
		Width:       158,
		Height:      258,
		SpawnFlash:  SpawnPoint{X: 3566, Y: 522},
		QuestState:  0,
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message:   "<font color='#990000'size='14'>出村后可能会涉及到强行PK环节，你确定要出村吗？</font>",
			Answers: []AnswerOption{
				{Handle: "1", Msg: "我要出村"},
				{Handle: "x", Msg: "取消"},
			},
		},
	},
}

var map33SourceNPCs = []sourceNPCEntry{
	{
		Handle:      "transp_31",
		RoleID:      "-3",
		DisplayName: "",
		SourceQuery: "transp/flag2.swf",
		SpriteName:  "flag2",
		Width:       158,
		Height:      258,
		SpawnFlash:  SpawnPoint{X: 57, Y: 584},
		QuestState:  0,
		Dialogue:    &sourceTransportDialogue,
	},
	{
		Handle:      "transp_34",
		RoleID:      "-3",
		DisplayName: "",
		SourceQuery: "transp/flag2.swf",
		SpriteName:  "flag2",
		Width:       158,
		Height:      258,
		SpawnFlash:  SpawnPoint{X: 2950, Y: 624},
		QuestState:  0,
		Dialogue:    &sourceTransportDialogue,
	},
	{
		Handle:      "transp_37",
		RoleID:      "-3",
		DisplayName: "",
		SourceQuery: "transp/flag2.swf",
		SpriteName:  "flag2",
		Width:       158,
		Height:      258,
		SpawnFlash:  SpawnPoint{X: 1124, Y: 750},
		QuestState:  0,
		Dialogue:    &sourceTransportDialogue,
	},
}

var map49SourceNPCs = []sourceNPCEntry{
	{
		Handle:      "transp_50",
		RoleID:      "-3",
		DisplayName: "",
		SourceQuery: "transp/flag2.swf",
		SpriteName:  "flag2",
		Width:       158,
		Height:      258,
		SpawnFlash:  SpawnPoint{X: 2950, Y: 650},
		QuestState:  0,
		Dialogue:    &sourceTransportDialogue,
	},
	{
		Handle:      "transp_48",
		RoleID:      "-3",
		DisplayName: "",
		SourceQuery: "transp/flag2.swf",
		SpriteName:  "flag2",
		Width:       158,
		Height:      258,
		SpawnFlash:  SpawnPoint{X: 180, Y: 464},
		QuestState:  0,
		Dialogue:    &sourceTransportDialogue,
	},
}

var map50SourceNPCs = []sourceNPCEntry{
	{
		Handle:      "transp_51",
		RoleID:      "-3",
		DisplayName: "",
		SourceQuery: "transp/flag2.swf",
		SpriteName:  "flag2",
		Width:       158,
		Height:      258,
		SpawnFlash:  SpawnPoint{X: 2647, Y: 424},
		QuestState:  0,
		Dialogue:    &sourceTransportDialogue,
	},
	{
		Handle:      "transp_55",
		RoleID:      "-3",
		DisplayName: "",
		SourceQuery: "transp/flag2.swf",
		SpriteName:  "flag2",
		Width:       158,
		Height:      258,
		SpawnFlash:  SpawnPoint{X: 2967, Y: 553},
		QuestState:  0,
		Dialogue:    &sourceTransportDialogue,
	},
	{
		Handle:      "transp_49",
		RoleID:      "-3",
		DisplayName: "",
		SourceQuery: "transp/flag2.swf",
		SpriteName:  "flag2",
		Width:       158,
		Height:      258,
		SpawnFlash:  SpawnPoint{X: 40, Y: 424},
		QuestState:  0,
		Dialogue:    &sourceTransportDialogue,
	},
}

var map45SourceNPCs = []sourceNPCEntry{
	{
		Handle:      "1810542611191117",
		DisplayName: "丑二品",
		SourceQuery: "npc/2品丑.swf",
		SpriteName:  "chouerpin",
		Width:       40,
		Height:      117,
		SpawnFlash:  SpawnPoint{X: 3213, Y: 310},
		QuestState:  4,
	},
	{
		Handle:      "1780542610743555",
		DisplayName: "伏天",
		SourceQuery: "npc/伏天.swf",
		SpriteName:  "futian",
		Width:       73,
		Height:      141,
		SpawnFlash:  SpawnPoint{X: 1600, Y: 390},
		QuestState:  3,
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message:   "((风云历历千余年，再无宝剑出世间。吾欲铸成第一剑，从此成名天下谈。))我伏天多年来一直潜居在这广青镇，不知为军为民铸就了多少柄剑，然这天下第一剑却一直只是妄念。岁月悠悠，转眼我已过知天命之年，可叹可叹......((15级之后到我这里可以接取杀猪除害任务，每小时循环一次))。((到商城购买装备传承符留在背包中，然后把精炼满20的绑定装备放到背包第一格，被传承的同部位绑定装备放在第二格，到我这里点击“装备传承”，就可以转移精炼属性了))。",
			Answers: []AnswerOption{
				{Handle: "day_9gs", Msg: "<party><m/>杀猪除害"},
				{Handle: "1", Msg: "购买武器"},
				{Handle: "2", Msg: "武器身份变更"},
				{Handle: "3", Msg: "装备传承"},
				{Handle: "0", Msg: "<c/>取消"},
			},
		},
	},
	{
		Handle:      "1850542611497321",
		DisplayName: "妖术狐狸",
		SourceQuery: "npc/狐狸.swf",
		SpriteName:  "huli",
		Width:       58,
		Height:      111,
		SpawnFlash:  SpawnPoint{X: 2420, Y: 335},
		QuestState:  0,
	},
	{
		Handle:      "transp_40",
		RoleID:      "-3",
		DisplayName: "",
		SourceQuery: "transp/flag2.swf",
		SpriteName:  "flag2",
		Width:       159,
		Height:      259,
		SpawnFlash:  SpawnPoint{X: 40, Y: 544},
		QuestState:  0,
		Dialogue:    &sourceTransportDialogue,
	},
	{
		Handle:      "1860542611566530",
		DisplayName: "圣诞树",
		SourceQuery: "npc/other/圣诞树.swf",
		SpriteName:  "shengdanshu",
		Width:       178,
		Height:      204,
		SpawnFlash:  SpawnPoint{X: 2765, Y: 400},
		QuestState:  2,
	},
	{
		Handle:      "1840542611483497",
		DisplayName: "锁妖鼎",
		SourceQuery: "npc/锁妖鼎.swf",
		SpriteName:  "suoyaoding",
		Width:       141,
		Height:      156,
		SpawnFlash:  SpawnPoint{X: 1266, Y: 417},
		QuestState:  0,
	},
	{
		Handle:      "1790542610850918",
		DisplayName: "广青守卫甲",
		SourceQuery: "npc/卫兵_向右.swf",
		SpriteName:  "weibing_right",
		Width:       80,
		Height:      170,
		SpawnFlash:  SpawnPoint{X: 464, Y: 419},
		QuestState:  2,
	},
	{
		Handle:      "1800542611079956",
		DisplayName: "雨娃",
		SourceQuery: "npc/雨娃.swf",
		SpriteName:  "yuwa",
		Width:       97,
		Height:      81,
		SpawnFlash:  SpawnPoint{X: 867, Y: 430},
		QuestState:  0,
	},
	{
		Handle:      "transp_46",
		RoleID:      "-3",
		DisplayName: "",
		SourceQuery: "transp/flag2.swf",
		SpriteName:  "flag2",
		Width:       159,
		Height:      259,
		SpawnFlash:  SpawnPoint{X: 3460, Y: 580},
		QuestState:  0,
		Dialogue:    &sourceTransportDialogue,
	},
	{
		Handle:      "1830542611405809",
		DisplayName: "通天八卦炉<ma>",
		SourceQuery: "npc/通天八卦炉.swf",
		SpriteName:  "bagualu",
		Width:       264,
		Height:      244,
		SpawnFlash:  SpawnPoint{X: 1782, Y: 400},
		QuestState:  0,
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message:   "八卦炉内燃烧着熊熊烈火......((如果拥有造物魔晶或特殊布料的话，便可以放入炉内进行冶炼合成稀有的道具。这些材料可以通过打开吉祥袋、如意袋或乾坤袋获得。注：在【商城】内可以购入【吉祥袋】。))",
			Answers: []AnswerOption{
				{Handle: "1", Msg: "合成稀有道具"},
				{Handle: "0", Msg: "<c/>取消"},
			},
		},
	},
	{
		Handle:      "1820542611400955",
		DisplayName: "丑五品",
		SourceQuery: "npc/6品丑.swf",
		SpriteName:  "chouwupin",
		Width:       66,
		Height:      121,
		SpawnFlash:  SpawnPoint{X: 2547, Y: 361},
		QuestState:  4,
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message:   "这位客官，小的是这广青镇上的杂货店老板丑五品。我们的商贸车队也遍布各个村镇，客官你尽管放心购买我们的东西，绝对货真价实，童叟无欺。((15级之后到我这里可以接取助商集物任务，每天循环一次))。",
			Answers: []AnswerOption{
				{Handle: "1", Msg: "道具商店"},
				{Handle: "3", Msg: "原石合成"},
				{Handle: "0", Msg: "<c/>取消"},
			},
		},
	},
}

var map46SourceNPCs = []sourceNPCEntry{
	{
		Handle:      "2150542612192672",
		DisplayName: "王喜财",
		SourceQuery: "npc/王喜财.swf",
		SpriteName:  "wangxicai",
		Width:       136,
		Height:      115,
		SpawnFlash:  SpawnPoint{X: 3130, Y: 321},
		QuestState:  0,
	},
	{
		Handle:      "2210542612665747",
		DisplayName: "衙役乙",
		SourceQuery: "npc/衙役乙.swf",
		SpriteName:  "yayi_yi",
		Width:       90,
		Height:      146,
		SpawnFlash:  SpawnPoint{X: 1832, Y: 463},
		QuestState:  3,
	},
	{
		Handle:      "2170542612315146",
		DisplayName: "林公子",
		SourceQuery: "npc/林公子.swf",
		SpriteName:  "lingongzi",
		Width:       48,
		Height:      147,
		SpawnFlash:  SpawnPoint{X: 3087, Y: 629},
		QuestState:  0,
	},
	{
		Handle:      "2160542612239918",
		DisplayName: "白乞",
		SourceQuery: "npc/白乞.swf",
		SpriteName:  "baiqi",
		Width:       163,
		Height:      112,
		SpawnFlash:  SpawnPoint{X: 181, Y: 395},
		QuestState:  4,
	},
	{
		Handle:      "2230542612965723",
		DisplayName: "玄机道士",
		SourceQuery: "npc/玄机道士.swf",
		SpriteName:  "xuanjidaoshi",
		Width:       103,
		Height:      127,
		SpawnFlash:  SpawnPoint{X: 616, Y: 420},
		QuestState:  0,
	},
	{
		Handle:      "2190542612528390",
		DisplayName: "张妈",
		SourceQuery: "npc/张妈.swf",
		SpriteName:  "zhangma",
		Width:       75,
		Height:      123,
		SpawnFlash:  SpawnPoint{X: 1506, Y: 452},
		QuestState:  0,
	},
	{
		Handle:      "2140542612112200",
		DisplayName: "擂台护卫",
		SourceQuery: "npc/擂台护卫.swf",
		SpriteName:  "leitaihuwei",
		Width:       82,
		Height:      175,
		SpawnFlash:  SpawnPoint{X: 2763, Y: 454},
		QuestState:  0,
	},
	{
		Handle:      "2100542611864758",
		DisplayName: "大斗",
		SourceQuery: "npc/大斗.swf",
		SpriteName:  "dadou",
		Width:       114,
		Height:      135,
		SpawnFlash:  SpawnPoint{X: 2949, Y: 252},
		QuestState:  3,
	},
	{
		Handle:      "2180542612350162",
		DisplayName: "孙大福",
		SourceQuery: "npc/孙大福.swf",
		SpriteName:  "sundafu",
		Width:       74,
		Height:      163,
		SpawnFlash:  SpawnPoint{X: 3287, Y: 577},
		QuestState:  0,
	},
	{
		Handle:      "2200542612620422",
		DisplayName: "衙役甲",
		SourceQuery: "npc/衙役甲.swf",
		SpriteName:  "yayi_jia",
		Width:       78,
		Height:      158,
		SpawnFlash:  SpawnPoint{X: 2111, Y: 465},
		QuestState:  3,
	},
	{
		Handle:      "2130542612102200",
		DisplayName: "排行告示",
		SourceQuery: "npc/公告牌.swf",
		SpriteName:  "gonggaopai",
		Width:       136,
		Height:      149,
		SpawnFlash:  SpawnPoint{X: 1109, Y: 425},
		QuestState:  0,
	},
	{
		Handle:      "2220542612946566",
		DisplayName: "夏侯武",
		SourceQuery: "npc/夏侯武.swf",
		SpriteName:  "xiahouwu",
		Width:       96,
		Height:      154,
		SpawnFlash:  SpawnPoint{X: 846, Y: 430},
		QuestState:  4,
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message:   "((清平盛世起纷争，人间遍地哀鸿声。我自慨然抚长剑，管天下谁是英雄。))这好好的太平盛世都被这煞的魔军搅得乱七八糟，真令人愤慨。((每日13:00-15:00或者19:00-21:00到我这可以接取生死劫任务))。",
			Answers: []AnswerOption{
				{Handle: "4q35os", Msg: "<ml><m/>夏侯前辈找我有事？"},
				{Handle: "1", Msg: "学习技能"},
				{Handle: "2", Msg: "创建公会"},
				{Handle: "0", Msg: "<c/>下次再说"},
			},
		},
	},
	{
		Handle:      "2110542611994150",
		DisplayName: "小斗",
		SourceQuery: "npc/小斗.swf",
		SpriteName:  "xiaodou",
		Width:       107,
		Height:      129,
		SpawnFlash:  SpawnPoint{X: 3205, Y: 245},
		QuestState:  1,
	},
	{
		Handle:      "transp_47",
		RoleID:      "-3",
		DisplayName: "",
		SourceQuery: "transp/flag2.swf",
		SpriteName:  "flag2",
		Width:       159,
		Height:      259,
		SpawnFlash:  SpawnPoint{X: 3460, Y: 580},
		QuestState:  0,
		Dialogue:    &sourceTransportDialogue,
	},
	{
		Handle:      "transp_48",
		RoleID:      "-3",
		DisplayName: "",
		SourceQuery: "transp/flag2.swf",
		SpriteName:  "flag2",
		Width:       159,
		Height:      259,
		SpawnFlash:  SpawnPoint{X: 1720, Y: 730},
		QuestState:  0,
		Dialogue:    &sourceTransportDialogue,
	},
	{
		Handle:      "transp_45",
		RoleID:      "-3",
		DisplayName: "",
		SourceQuery: "transp/flag2.swf",
		SpriteName:  "flag2",
		Width:       159,
		Height:      259,
		SpawnFlash:  SpawnPoint{X: 40, Y: 564},
		QuestState:  0,
		Dialogue:    &sourceTransportDialogue,
	},
	{
		Handle:      "2120542612098244",
		DisplayName: "李老头",
		SourceQuery: "npc/李老头.swf",
		SpriteName:  "lilaotou",
		Width:       52,
		Height:      105,
		SpawnFlash:  SpawnPoint{X: 1392, Y: 440},
		QuestState:  3,
	},
}

var map47SourceNPCs = []sourceNPCEntry{
	{
		Handle:      "2500542613172144",
		DisplayName: "云衣娘",
		SourceQuery: "npc/云衣娘.swf",
		SpriteName:  "yunyiniang",
		Width:       58,
		Height:      162,
		SpawnFlash:  SpawnPoint{X: 1459, Y: 441},
		QuestState:  0,
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message:   "((一场寂寞凭谁诉，一片痴情君莫负。若知君心似我心，再不怕那相思苦。))小女子是这广青镇上的裁缝，专门做些衣物护具，客官如有需要，可以来找我。",
			Answers: []AnswerOption{
				{Handle: "1", Msg: "购买护具"},
				{Handle: "0", Msg: "<c/>取消"},
			},
		},
	},
	{
		Handle:      "2540542613409380",
		DisplayName: "广青守卫丙",
		SourceQuery: "npc/卫兵_向右.swf",
		SpriteName:  "weibing_right",
		Width:       80,
		Height:      170,
		SpawnFlash:  SpawnPoint{X: 3246, Y: 624},
		QuestState:  0,
	},
	{
		Handle:      "2560542613616537",
		DisplayName: "幻化使者",
		SourceQuery: "npc/lingxia/大葱哥.swf",
		SpriteName:  "dacongge",
		Width:       78,
		Height:      165,
		SpawnFlash:  SpawnPoint{X: 1957, Y: 453},
		QuestState:  0,
	},
	{
		Handle:      "2520542613299551",
		DisplayName: "伏罗",
		SourceQuery: "npc/伏罗.swf",
		SpriteName:  "fuluo",
		Width:       75,
		Height:      121,
		SpawnFlash:  SpawnPoint{X: 2408, Y: 380},
		QuestState:  0,
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message:   "((少小得道在远山，多年行医云水间。妙手疗得万般病，唯有相思治不痊。))老夫我少年师从名医，学得治疗天下疑难杂症的本事。",
			Answers: []AnswerOption{
				{Handle: "2", Msg: "进行治疗"},
				{Handle: "1", Msg: "查看商店"},
				{Handle: "0", Msg: "<c/>取消"},
			},
		},
	},
	{
		Handle:      "2550542613498646",
		DisplayName: "帚公",
		SourceQuery: "npc/帚公.swf",
		SpriteName:  "zhougong",
		Width:       120,
		Height:      135,
		SpawnFlash:  SpawnPoint{X: 876, Y: 409},
		QuestState:  1,
	},
	{
		Handle:      "2510542613193577",
		DisplayName: "木安",
		SourceQuery: "npc/木安.swf",
		SpriteName:  "muan",
		Width:       55,
		Height:      146,
		SpawnFlash:  SpawnPoint{X: 285, Y: 425},
		QuestState:  0,
	},
	{
		Handle:      "transp_52",
		RoleID:      "-3",
		DisplayName: "",
		SourceQuery: "transp/flag2.swf",
		SpriteName:  "flag2",
		Width:       159,
		Height:      259,
		SpawnFlash:  SpawnPoint{X: 3460, Y: 550},
		QuestState:  0,
		Dialogue:    &sourceTransportDialogue,
	},
	{
		Handle:      "2530542613373649",
		DisplayName: "广青守卫乙",
		SourceQuery: "npc/卫兵_向右.swf",
		SpriteName:  "weibing_right",
		Width:       80,
		Height:      170,
		SpawnFlash:  SpawnPoint{X: 3197, Y: 392},
		QuestState:  0,
	},
	{
		Handle:      "transp_46",
		RoleID:      "-3",
		DisplayName: "",
		SourceQuery: "transp/flag2.swf",
		SpriteName:  "flag2",
		Width:       159,
		Height:      259,
		SpawnFlash:  SpawnPoint{X: 40, Y: 564},
		QuestState:  0,
		Dialogue:    &sourceTransportDialogue,
	},
}

var map79SourceNPCs = []sourceNPCEntry{
	{
		Handle:      "transp_80",
		RoleID:      "-3",
		DisplayName: "",
		SourceQuery: "transp/flag2.swf",
		SpriteName:  "flag2",
		Width:       158,
		Height:      258,
		SpawnFlash:  SpawnPoint{X: 60, Y: 524},
		QuestState:  0,
		Dialogue:    &sourceTransportDialogue,
	},
}

var map80SourceNPCs = []sourceNPCEntry{
	{
		Handle:      "transp_81",
		RoleID:      "-3",
		DisplayName: "",
		SourceQuery: "transp/flag2.swf",
		SpriteName:  "flag2",
		Width:       158,
		Height:      258,
		SpawnFlash:  SpawnPoint{X: 80, Y: 534},
		QuestState:  0,
		Dialogue:    &sourceTransportDialogue,
	},
	{
		Handle:      "transp_79",
		RoleID:      "-3",
		DisplayName: "",
		SourceQuery: "transp/flag2.swf",
		SpriteName:  "flag2",
		Width:       158,
		Height:      258,
		SpawnFlash:  SpawnPoint{X: 2940, Y: 524},
		QuestState:  0,
		Dialogue:    &sourceTransportDialogue,
	},
}

var map81SourceNPCs = []sourceNPCEntry{
	{
		Handle:      "transp_82",
		RoleID:      "-3",
		DisplayName: "",
		SourceQuery: "transp/flag2.swf",
		SpriteName:  "flag2",
		Width:       158,
		Height:      258,
		SpawnFlash:  SpawnPoint{X: 75, Y: 464},
		QuestState:  0,
		Dialogue:    &sourceTransportDialogue,
	},
	{
		Handle:      "transp_80",
		RoleID:      "-3",
		DisplayName: "",
		SourceQuery: "transp/flag2.swf",
		SpriteName:  "flag2",
		Width:       158,
		Height:      258,
		SpawnFlash:  SpawnPoint{X: 2940, Y: 464},
		QuestState:  0,
		Dialogue:    &sourceTransportDialogue,
	},
}

var map82SourceNPCs = []sourceNPCEntry{
	{
		Handle:      "transp_83",
		RoleID:      "-3",
		DisplayName: "",
		SourceQuery: "transp/flag2.swf",
		SpriteName:  "flag2",
		Width:       158,
		Height:      258,
		SpawnFlash:  SpawnPoint{X: 75, Y: 464},
		QuestState:  0,
		Dialogue:    &sourceTransportDialogue,
	},
	{
		Handle:      "transp_81",
		RoleID:      "-3",
		DisplayName: "",
		SourceQuery: "transp/flag2.swf",
		SpriteName:  "flag2",
		Width:       158,
		Height:      258,
		SpawnFlash:  SpawnPoint{X: 2940, Y: 464},
		QuestState:  0,
		Dialogue:    &sourceTransportDialogue,
	},
}

var map83SourceNPCs = []sourceNPCEntry{
	{
		Handle:      "transp_0",
		RoleID:      "-3",
		DisplayName: "",
		SourceQuery: "transp/flag2.swf",
		SpriteName:  "flag2",
		Width:       158,
		Height:      258,
		SpawnFlash:  SpawnPoint{X: 341, Y: 363},
		QuestState:  0,
		Dialogue:    &sourceTransportDialogue,
	},
	{
		Handle:      "transp_82",
		RoleID:      "-3",
		DisplayName: "",
		SourceQuery: "transp/flag2.swf",
		SpriteName:  "flag2",
		Width:       158,
		Height:      258,
		SpawnFlash:  SpawnPoint{X: 2930, Y: 480},
		QuestState:  0,
		Dialogue:    &sourceTransportDialogue,
	},
}

var map127SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransport("transp_131", 1020, 300),
}

var map131SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransport("transp_132", 2950, 550),
	buildCapturedSourceTransport("transp_127", 44, 530),
}

var map131SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("8128205778897212", "武斗蛤蟆", "monstermap/kongfupanda.swf", "kongfupanda", 12, "游侠", 1626, 453),
	buildCapturedSourceMonster("8130205778898758", "武斗蛤蟆", "monstermap/kongfupanda.swf", "kongfupanda", 12, "游侠", 2357, 492),
	buildCapturedSourceMonster("8132205778899676", "武斗蛤蟆", "monstermap/kongfupanda.swf", "kongfupanda", 12, "游侠", 1550, 607),
}

var map132SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransport("transp_133", 2450, 550),
	buildCapturedSourceTransport("transp_131", 44, 530),
}

var map132SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("1656205827185847", "武斗蛤蟆", "monstermap/kongfupanda.swf", "kongfupanda", 13, "战士", 634, 471),
	buildCapturedSourceMonster("1658205827186196", "武斗蛤蟆", "monstermap/kongfupanda.swf", "kongfupanda", 12, "游侠", 455, 387),
	buildCapturedSourceMonster("1660205827187303", "武斗蛤蟆", "monstermap/kongfupanda.swf", "kongfupanda", 12, "游侠", 1500, 509),
	buildCapturedSourceMonster("1662205827187552", "武斗蛤蟆", "monstermap/kongfupanda.swf", "kongfupanda", 12, "游侠", 535, 589),
	buildCapturedSourceMonster("1664205827188995", "武斗蛤蟆", "monstermap/kongfupanda.swf", "kongfupanda", 13, "战士", 1980, 442),
	buildCapturedSourceMonster("1666205827189909", "武斗蛤蟆", "monstermap/kongfupanda.swf", "kongfupanda", 13, "战士", 1913, 564),
}

var map133SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransport("transp_132", 44, 530),
	buildCapturedSourceTransport("transp_137", 2960, 530),
}

var map133SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("8112205902790159", "武斗蛤蟆", "monstermap/kongfupanda.swf", "kongfupanda", 14, "战士", 507, 438),
	buildCapturedSourceMonster("8114205902791857", "武斗蛤蟆", "monstermap/kongfupanda.swf", "kongfupanda", 14, "战士", 1580, 503),
	buildCapturedSourceMonster("8116205902792836", "武斗蛤蟆", "monstermap/kongfupanda.swf", "kongfupanda", 14, "战士", 434, 603),
	buildCapturedSourceMonster("8118205902792830", "武斗蛤蟆", "monstermap/kongfupanda.swf", "kongfupanda", 14, "战士", 2480, 442),
	buildCapturedSourceMonster("8120205902793205", "武斗蛤蟆", "monstermap/kongfupanda.swf", "kongfupanda", 14, "战士", 2423, 507),
	buildCapturedSourceMonster("8122205902794636", "武斗蛤蟆", "monstermap/kongfupanda.swf", "kongfupanda", 14, "战士", 2457, 603),
}

var map137SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransport("transp_133", 40, 555),
	buildCapturedSourceTransport("transp_144", 2960, 570),
}

var map137SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("4889205982270617", "剑术蛤蟆", "monstermap/swordpanda.swf", "swordpanda", 15, "战士", 500, 423),
	buildCapturedSourceMonster("4891205982270480", "剑术蛤蟆", "monstermap/swordpanda.swf", "swordpanda", 15, "战士", 503, 507),
	buildCapturedSourceMonster("4893205982271973", "剑术蛤蟆", "monstermap/swordpanda.swf", "swordpanda", 15, "战士", 357, 600),
	buildCapturedSourceMonster("4895205982272135", "法术蛤蟆", "monstermap/magicpanda.swf", "magicpanda", 16, "术士", 1803, 423),
	buildCapturedSourceMonster("4897205982273653", "法术蛤蟆", "monstermap/magicpanda.swf", "magicpanda", 16, "术士", 2484, 500),
	buildCapturedSourceMonster("4899205982273477", "法术蛤蟆", "monstermap/magicpanda.swf", "magicpanda", 16, "术士", 1615, 588),
}

var map140SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransport("transp_141", 2950, 650),
	buildCapturedSourceTransport("transp_145", 40, 500),
}

var map140SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("8424206376338175", "剑术蛤蟆", "monstermap/swordpanda.swf", "swordpanda", 18, "战士", 1588, 611),
	buildCapturedSourceMonster("8426206376339691", "剑术蛤蟆", "monstermap/swordpanda.swf", "swordpanda", 18, "战士", 1676, 526),
	buildCapturedSourceMonster("8428206376340510", "剑术蛤蟆", "monstermap/swordpanda.swf", "swordpanda", 18, "战士", 1769, 453),
	buildCapturedSourceMonster("8430206376341780", "武斗蛤蟆", "monstermap/kongfupanda.swf", "kongfupanda", 17, "游侠", 1273, 530),
	buildCapturedSourceMonster("8432206376342756", "武斗蛤蟆", "monstermap/kongfupanda.swf", "kongfupanda", 17, "游侠", 1326, 615),
}

var map141SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransport("transp_140", 170, 340),
	buildCapturedSourceTransport("transp_142", 1800, 720),
}

var map141SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("4710206556985181", "剑术蛤蟆", "monstermap/swordpanda.swf", "swordpanda", 18, "战士", 520, 466),
	buildCapturedSourceMonster("4712206556985489", "剑术蛤蟆", "monstermap/swordpanda.swf", "swordpanda", 18, "战士", 771, 566),
	buildCapturedSourceMonster("4714206556986370", "剑术蛤蟆", "monstermap/swordpanda.swf", "swordpanda", 18, "战士", 815, 484),
	buildCapturedSourceMonster("4716206556987104", "剑术蛤蟆", "monstermap/swordpanda.swf", "swordpanda", 18, "战士", 812, 553),
	buildCapturedSourceMonster("4718206556987620", "剑术蛤蟆", "monstermap/swordpanda.swf", "swordpanda", 18, "战士", 1330, 425),
	buildCapturedSourceMonster("4720206556988825", "剑术蛤蟆", "monstermap/swordpanda.swf", "swordpanda", 18, "战士", 1202, 535),
}

var map142SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransport("transp_141", 2780, 350),
	buildCapturedSourceTransport("transp_143", 1909, 720),
}

var map142SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("9556206735436383", "剑术蛤蟆", "monstermap/swordpanda.swf", "swordpanda", 18, "战士", 630, 434),
	buildCapturedSourceMonster("9558206735437433", "剑术蛤蟆", "monstermap/swordpanda.swf", "swordpanda", 18, "战士", 600, 557),
	buildCapturedSourceMonster("9560206735437619", "法术蛤蟆", "monstermap/magicpanda.swf", "magicpanda", 17, "术士", 803, 430),
	buildCapturedSourceMonster("9562206735438199", "法术蛤蟆", "monstermap/magicpanda.swf", "magicpanda", 17, "术士", 742, 557),
	buildCapturedSourceMonster("9564206735439303", "法术蛤蟆", "monstermap/magicpanda.swf", "magicpanda", 17, "术士", 2403, 515),
	buildCapturedSourceMonster("9566206735440658", "剑术蛤蟆", "monstermap/swordpanda.swf", "swordpanda", 18, "战士", 2327, 438),
	buildCapturedSourceMonster("9568206735440182", "剑术蛤蟆", "monstermap/swordpanda.swf", "swordpanda", 18, "战士", 2400, 592),
}

var map143SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransport("transp_142", 220, 350),
	buildCapturedSourceTransport("transp_127", 2120, 440),
}

var map143SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("5166206909805441", "法术蛤蟆", "monstermap/magicpanda.swf", "magicpanda", 17, "术士", 1237, 551),
	buildCapturedSourceMonster("5168206909805631", "法术蛤蟆", "monstermap/magicpanda.swf", "magicpanda", 17, "术士", 2080, 506),
	buildCapturedSourceMonster("5170206909806155", "法术蛤蟆", "monstermap/magicpanda.swf", "magicpanda", 17, "术士", 1951, 471),
	buildCapturedSourceMonster("5172206909807859", "剑术蛤蟆", "monstermap/swordpanda.swf", "swordpanda", 18, "战士", 631, 480),
	buildCapturedSourceMonster("5174206909807286", "剑术蛤蟆", "monstermap/swordpanda.swf", "swordpanda", 18, "战士", 487, 615),
	buildCapturedSourceMonster("5176206909809579", "蛤蟆精", "monstermap/cracktoad.swf", "cracktoad", 20, "游侠++", 2070, 464),
}

var map144SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransport("transp_145", 2800, 720),
	buildCapturedSourceTransport("transp_137", 44, 530),
}

var map144SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("2762206074545916", "武斗蛤蟆", "monstermap/kongfupanda.swf", "kongfupanda", 16, "游侠", 650, 457),
	buildCapturedSourceMonster("2764206074546810", "武斗蛤蟆", "monstermap/kongfupanda.swf", "kongfupanda", 16, "游侠", 680, 569),
	buildCapturedSourceMonster("2766206074547838", "武斗蛤蟆", "monstermap/kongfupanda.swf", "kongfupanda", 16, "游侠", 2177, 419),
	buildCapturedSourceMonster("2768206074548639", "剑术蛤蟆", "monstermap/swordpanda.swf", "swordpanda", 17, "战士", 1369, 496),
	buildCapturedSourceMonster("2770206074548423", "剑术蛤蟆", "monstermap/swordpanda.swf", "swordpanda", 17, "战士", 1480, 561),
}

var map145SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransport("transp_140", 2460, 550),
	buildCapturedSourceTransport("transp_144", 200, 410),
}

var map145SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("2890206197338884", "剑术蛤蟆", "monstermap/swordpanda.swf", "swordpanda", 18, "战士", 541, 474),
	buildCapturedSourceMonster("2892206197339825", "剑术蛤蟆", "monstermap/swordpanda.swf", "swordpanda", 18, "战士", 352, 596),
	buildCapturedSourceMonster("2894206197340572", "法术蛤蟆", "monstermap/magicpanda.swf", "magicpanda", 17, "术士", 1028, 532),
	buildCapturedSourceMonster("2896206197341225", "法术蛤蟆", "monstermap/magicpanda.swf", "magicpanda", 17, "术士", 1189, 592),
	buildCapturedSourceMonster("2898206197342379", "剑术蛤蟆", "monstermap/swordpanda.swf", "swordpanda", 18, "战士", 1689, 483),
	buildCapturedSourceMonster("2900206197343711", "剑术蛤蟆", "monstermap/swordpanda.swf", "swordpanda", 18, "战士", 1721, 602),
}

var map122SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransport("transp_121", 1460, 520),
	buildCapturedSourceTransportMovieClip("transp_146", "transp/fl.swf", "fl", 139, 147, 329, 480),
}

var map146SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransport("transp_122", 1950, 488),
	buildCapturedSourceTransport("transp_147", 55, 507),
	buildCapturedSourceTransport("transp_152", 409, 185),
}

var map148SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransport("transp_147", 1969, 505),
	buildCapturedSourceTransport("transp_149", 424, 379),
}

var map149SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransport("transp_148", 30, 550),
	buildCapturedSourceTransport("transp_150", 2280, 351),
}

var map150SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransport("transp_149", 1413, 750),
	buildCapturedSourceTransport("transp_151", 2969, 437),
	buildCapturedSourceTransport("transp_153", 151, 273),
}

var map151SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransport("transp_150", 37, 688),
	buildCapturedSourceTransport("transp_152", 2463, 442),
}

var map152SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransport("transp_146", 2963, 600),
	buildCapturedSourceTransport("transp_151", 37, 588),
}

var map153SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransport("transp_150", 2869, 444),
	buildCapturedSourceTransport("transp_154", 1401, 706),
	buildCapturedSourceTransport("transp_156", 32, 492),
}

var map154SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransport("transp_153", 2966, 230),
	buildCapturedSourceTransport("transp_155", 37, 513),
	buildCapturedSourceTransport("transp_157", 956, 114),
}

var map155SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransportMovieClip("transp_122", "transp/fl.swf", "fl", 139, 147, 158, 455),
	buildCapturedSourceTransport("transp_154", 1941, 556),
}

var map156SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransport("transp_153", 2466, 494),
	buildCapturedSourceTransport("transp_157", 31, 509),
}

var map157SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransport("transp_154", 185, 720),
	buildCapturedSourceTransport("transp_156", 2963, 515),
}

var map146SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("6887685480585492", "蛮族刀客", "monstermap/barbarianweapons.swf", "barbarianweapons", 12, "战士", 784, 325),
	buildCapturedSourceMonster("6889685480586263", "蛮族刀客", "monstermap/barbarianweapons.swf", "barbarianweapons", 12, "战士", 212, 279),
	buildCapturedSourceMonster("6891685480586720", "蛮族刀客", "monstermap/barbarianweapons.swf", "barbarianweapons", 12, "战士", 348, 392),
	buildCapturedSourceMonster("6893685480587444", "蛮族刀客", "monstermap/barbarianweapons.swf", "barbarianweapons", 12, "战士", 282, 546),
}

var map148SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("1060685893848234", "蛮族刀客", "monstermap/barbarianweapons.swf", "barbarianweapons", 14, "战士", 264, 407),
	buildCapturedSourceMonster("1062685893848980", "蛮族刀客", "monstermap/barbarianweapons.swf", "barbarianweapons", 14, "战士", 471, 412),
	buildCapturedSourceMonster("1064685893849658", "蛮族刀客", "monstermap/barbarianweapons.swf", "barbarianweapons", 13, "战士", 548, 484),
	buildCapturedSourceMonster("1066685893850755", "蛮族刀客", "monstermap/barbarianweapons.swf", "barbarianweapons", 13, "战士", 571, 620),
	buildCapturedSourceMonster("1068685893850410", "蛮族刀客", "monstermap/barbarianweapons.swf", "barbarianweapons", 14, "战士", 1443, 446),
	buildCapturedSourceMonster("1070685893851339", "蛮族刀客", "monstermap/barbarianweapons.swf", "barbarianweapons", 14, "战士", 1402, 576),
}

var map149SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("3206685759634939", "蛮族刀客", "monstermap/barbarianweapons.swf", "barbarianweapons", 14, "战士", 695, 413),
	buildCapturedSourceMonster("3208685759635236", "蛮族刀客", "monstermap/barbarianweapons.swf", "barbarianweapons", 14, "战士", 698, 541),
	buildCapturedSourceMonster("3210685759635176", "蛮族刀客", "monstermap/barbarianweapons.swf", "barbarianweapons", 14, "战士", 1939, 455),
	buildCapturedSourceMonster("3212685759636247", "蛮族刀客", "monstermap/barbarianweapons.swf", "barbarianweapons", 14, "战士", 1977, 570),
	buildCapturedSourceMonster("3214685759637760", "蛮族弓手", "monstermap/barbarianbowman.swf", "barbarianbowman", 15, "游侠", 1285, 400),
	buildCapturedSourceMonster("3216685759637116", "蛮族弓手", "monstermap/barbarianbowman.swf", "barbarianbowman", 15, "游侠", 1394, 442),
	buildCapturedSourceMonster("3218685759638239", "黄风二寨主", "monstermap/hfscastellan.swf", "hfscastellan", 19, "战士++", 1451, 403),
	buildCapturedSourceMonster("3220685759639165", "蛮族弓手", "monstermap/barbarianbowman.swf", "barbarianbowman", 15, "游侠", 1605, 442),
}

var map150SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("7626685662869779", "蛮族弓手", "monstermap/barbarianbowman.swf", "barbarianbowman", 15, "游侠", 550, 476),
	buildCapturedSourceMonster("7628685662869126", "蛮族弓手", "monstermap/barbarianbowman.swf", "barbarianbowman", 15, "游侠", 519, 569),
	buildCapturedSourceMonster("7630685662870286", "蛮族战士", "monstermap/barbarianfighter.swf", "barbarianfighter", 16, "战士", 396, 400),
	buildCapturedSourceMonster("7632685662871580", "蛮族战士", "monstermap/barbarianfighter.swf", "barbarianfighter", 16, "战士", 303, 500),
	buildCapturedSourceMonster("7634685662872533", "蛮族战士", "monstermap/barbarianfighter.swf", "barbarianfighter", 16, "战士", 2423, 500),
	buildCapturedSourceMonster("7636685662872653", "蛮族战士", "monstermap/barbarianfighter.swf", "barbarianfighter", 16, "战士", 2446, 607),
}

var map151SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("3479685591196972", "蛮族刀客", "monstermap/barbarianweapons.swf", "barbarianweapons", 14, "战士", 1961, 362),
	buildCapturedSourceMonster("3481685591197274", "蛮族刀客", "monstermap/barbarianweapons.swf", "barbarianweapons", 14, "战士", 2006, 442),
	buildCapturedSourceMonster("3483685591198699", "蛮族弓手", "monstermap/barbarianbowman.swf", "barbarianbowman", 15, "游侠", 708, 381),
	buildCapturedSourceMonster("3485685591198493", "蛮族弓手", "monstermap/barbarianbowman.swf", "barbarianbowman", 15, "游侠", 480, 451),
}

var map152SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("2200685534047114", "蛮族刀客", "monstermap/barbarianweapons.swf", "barbarianweapons", 14, "战士", 1311, 476),
	buildCapturedSourceMonster("2400685534048391", "蛮族刀客", "monstermap/barbarianweapons.swf", "barbarianweapons", 14, "战士", 2496, 561),
	buildCapturedSourceMonster("2600685534049446", "蛮族刀客", "monstermap/barbarianweapons.swf", "barbarianweapons", 14, "战士", 1953, 534),
	buildCapturedSourceMonster("2800685534050729", "蛮族弓手", "monstermap/barbarianbowman.swf", "barbarianbowman", 15, "游侠", 361, 488),
	buildCapturedSourceMonster("3000685534050820", "蛮族弓手", "monstermap/barbarianbowman.swf", "barbarianbowman", 15, "游侠", 342, 573),
}

var map153SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("7486686002236449", "蛮族弓手", "monstermap/barbarianbowman.swf", "barbarianbowman", 16, "游侠", 1955, 390),
	buildCapturedSourceMonster("7488686002237979", "蛮族弓手", "monstermap/barbarianbowman.swf", "barbarianbowman", 16, "游侠", 1918, 475),
	buildCapturedSourceMonster("7490686002238200", "蛮族弓手", "monstermap/barbarianbowman.swf", "barbarianbowman", 16, "游侠", 784, 420),
	buildCapturedSourceMonster("7492686002238537", "蛮族弓手", "monstermap/barbarianbowman.swf", "barbarianbowman", 16, "游侠", 910, 494),
	buildCapturedSourceMonster("7494686002239485", "咒巫师", "monstermap/incantationshaman.swf", "incantationshaman", 16, "术士", 1282, 453),
	buildCapturedSourceMonster("7496686002240421", "咒巫师", "monstermap/incantationshaman.swf", "incantationshaman", 16, "术士", 1587, 435),
}

var map154SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("4401686309555513", "蛮族战士", "monstermap/barbarianfighter.swf", "barbarianfighter", 18, "战士", 1796, 373),
	buildCapturedSourceMonster("4403686309556739", "蛮族战士", "monstermap/barbarianfighter.swf", "barbarianfighter", 18, "战士", 1811, 492),
	buildCapturedSourceMonster("4405686309557261", "蛮族战士", "monstermap/barbarianfighter.swf", "barbarianfighter", 18, "战士", 407, 411),
	buildCapturedSourceMonster("4407686309558215", "蛮族战士", "monstermap/barbarianfighter.swf", "barbarianfighter", 18, "战士", 392, 569),
	buildCapturedSourceMonster("4409686309558350", "蛮族弓手", "monstermap/barbarianbowman.swf", "barbarianbowman", 17, "游侠", 1326, 396),
	buildCapturedSourceMonster("4411686309559260", "蛮族弓手", "monstermap/barbarianbowman.swf", "barbarianbowman", 17, "游侠", 1334, 565),
}

var map155SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("1800686416053680", "蛮族战士", "monstermap/barbarianfighter.swf", "barbarianfighter", 18, "战士", 1271, 451),
	buildCapturedSourceMonster("2000686416054570", "蛮族战士", "monstermap/barbarianfighter.swf", "barbarianfighter", 18, "战士", 1143, 617),
	buildCapturedSourceMonster("2200686416054601", "蛮族战士", "monstermap/barbarianfighter.swf", "barbarianfighter", 18, "战士", 710, 464),
	buildCapturedSourceMonster("2400686416055777", "蛮族战士", "monstermap/barbarianfighter.swf", "barbarianfighter", 18, "战士", 653, 589),
	buildCapturedSourceMonster("2600686416056495", "黄风大寨主", "monstermap/hfcastellan.swf", "hfcastellan", 20, "战士++", 292, 476),
	buildCapturedSourceMonster("2800686416057704", "黄风寨夫人", "monstermap/hflady.swf", "hflady", 20, "游侠++", 300, 553),
}

var map156SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("1810686076568601", "蛮族弓手", "monstermap/barbarianbowman.swf", "barbarianbowman", 17, "游侠", 266, 432),
	buildCapturedSourceMonster("1812686076568276", "蛮族弓手", "monstermap/barbarianbowman.swf", "barbarianbowman", 17, "游侠", 243, 519),
	buildCapturedSourceMonster("1814686076569307", "蛮族弓手", "monstermap/barbarianbowman.swf", "barbarianbowman", 17, "游侠", 2135, 420),
	buildCapturedSourceMonster("1816686076570599", "蛮族弓手", "monstermap/barbarianbowman.swf", "barbarianbowman", 17, "游侠", 2090, 509),
	buildCapturedSourceMonster("1818686076571461", "咒巫师", "monstermap/incantationshaman.swf", "incantationshaman", 17, "术士", 961, 461),
	buildCapturedSourceMonster("1820686076572431", "咒巫师", "monstermap/incantationshaman.swf", "incantationshaman", 17, "术士", 1090, 522),
	buildCapturedSourceMonster("1822686076573189", "咒巫师", "monstermap/incantationshaman.swf", "incantationshaman", 17, "术士", 1282, 453),
}

var map157SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("7597686175728336", "蛮族弓手", "monstermap/barbarianbowman.swf", "barbarianbowman", 17, "游侠", 1207, 453),
	buildCapturedSourceMonster("7599686175728881", "蛮族弓手", "monstermap/barbarianbowman.swf", "barbarianbowman", 17, "游侠", 1161, 542),
	buildCapturedSourceMonster("7601686175729343", "咒巫师", "monstermap/incantationshaman.swf", "incantationshaman", 17, "术士", 469, 496),
	buildCapturedSourceMonster("7603686175731981", "咒巫师", "monstermap/incantationshaman.swf", "incantationshaman", 17, "术士", 792, 576),
	buildCapturedSourceMonster("7605686175731943", "蛮族战士", "monstermap/barbarianfighter.swf", "barbarianfighter", 18, "战士", 2527, 438),
	buildCapturedSourceMonster("7607686175732966", "蛮族战士", "monstermap/barbarianfighter.swf", "barbarianfighter", 18, "战士", 2477, 534),
}

func buildCapturedSourceTransport(handle string, x int, y int) sourceNPCEntry {
	return buildCapturedSourceTransportMovieClip(handle, "transp/flag2.swf", "flag2", 158, 258, x, y)
}

func buildCapturedSourceTransportMovieClip(handle string, sourceQuery string, spriteName string, width int, height int, x int, y int) sourceNPCEntry {
	return sourceNPCEntry{
		Handle:      handle,
		RoleID:      "-3",
		DisplayName: "",
		SourceQuery: sourceQuery,
		SpriteName:  spriteName,
		Width:       width,
		Height:      height,
		SpawnFlash:  SpawnPoint{X: x, Y: y},
		QuestState:  0,
		Dialogue:    &sourceTransportDialogue,
	}
}

func buildCapturedSourceMonster(handle string, displayName string, sourceQuery string, spriteName string, level int, vocation string, x int, y int) sourceMonsterEntry {
	return sourceMonsterEntry{
		Handle:      handle,
		DisplayName: displayName,
		SourceQuery: sourceQuery,
		SpriteName:  spriteName,
		Level:       level,
		Vocation:    vocation,
		Width:       140,
		Height:      140,
		SpawnFlash:  SpawnPoint{X: x, Y: y},
		Movement:    capturedSourceMonsterMovements[handle],
	}
}

var map64SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransportMovieClip("transp_18", "transp/flag2.swf", "flag2", 158, 258, 40, 431),
	buildCapturedSourceTransportMovieClip("transp_65", "transp/flag2.swf", "flag2", 158, 258, 950, 430),
}

var map65SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransportMovieClip("transp_64", "transp/flag2.swf", "flag2", 158, 258, 45, 454),
	buildCapturedSourceTransportMovieClip("transp_66", "transp/flag2.swf", "flag2", 158, 258, 1250, 500),
}

var map66SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransportMovieClip("transp_65", "transp/flag2.swf", "flag2", 158, 258, 45, 454),
	buildCapturedSourceTransportMovieClip("transp_67", "transp/flag2.swf", "flag2", 158, 258, 1450, 500),
}

var map67SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransportMovieClip("transp_66", "transp/flag2.swf", "flag2", 158, 258, 45, 494),
	buildCapturedSourceTransportMovieClip("transp_68", "transp/flag2.swf", "flag2", 158, 258, 1550, 470),
}

var map68SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransportMovieClip("transp_67", "transp/flag2.swf", "flag2", 158, 258, 45, 494),
	buildCapturedSourceTransportMovieClip("transp_69", "transp/flag2.swf", "flag2", 158, 258, 1380, 354),
	buildCapturedSourceTransportMovieClip("transp_70", "transp/flag2.swf", "flag2", 158, 258, 1950, 580),
}

var map69SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransportMovieClip("transp_68", "transp/flag2.swf", "flag2", 158, 258, 45, 514),
	buildCapturedSourceTransportMovieClip("transp_71", "transp/flag2.swf", "flag2", 158, 258, 2450, 440),
}

var map71SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransportMovieClip("transp_69", "transp/flag2.swf", "flag2", 158, 258, 45, 424),
	buildCapturedSourceTransportMovieClip("transp_73", "transp/flag2.swf", "flag2", 158, 258, 2750, 600),
}

var map72SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransportMovieClip("transp_70", "transp/flag2.swf", "flag2", 158, 258, 45, 584),
	buildCapturedSourceTransportMovieClip("transp_73", "transp/flag2.swf", "flag2", 158, 258, 2405, 423),
	buildCapturedSourceTransportMovieClip("transp_74", "transp/flag2.swf", "flag2", 158, 258, 2798, 580),
}

var map73SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransportMovieClip("transp_71", "transp/flag2.swf", "flag2", 158, 258, 45, 424),
	buildCapturedSourceTransportMovieClip("transp_72", "transp/flag2.swf", "flag2", 158, 258, 2350, 720),
	buildCapturedSourceTransportMovieClip("transp_77", "transp/flag2.swf", "flag2", 158, 258, 2950, 550),
}

var map74SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransportMovieClip("transp_72", "transp/flag2.swf", "flag2", 158, 258, 2819, 340),
	buildCapturedSourceTransportMovieClip("transp_75", "transp/flag2.swf", "flag2", 158, 258, 95, 720),
}

var map75SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransportMovieClip("transp_74", "transp/flag2.swf", "flag2", 158, 258, 1338, 380),
	buildCapturedSourceTransportMovieClip("transp_76", "transp/flag2.swf", "flag2", 158, 258, 2838, 350),
}

var map76SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransportMovieClip("transp_18", "transp/fl.swf", "fl", 158, 258, 1590, 541),
	buildCapturedSourceTransportMovieClip("transp_75", "transp/flag2.swf", "flag2", 158, 258, 25, 544),
}

var map77SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransportMovieClip("transp_73", "transp/flag2.swf", "flag2", 158, 258, 40, 431),
	buildCapturedSourceTransportMovieClip("transp_78", "transp/flag2.swf", "flag2", 158, 258, 2611, 400),
}

var map78SourceNPCs = []sourceNPCEntry{
	buildCapturedSourceTransportMovieClip("transp_77", "transp/flag2.swf", "flag2", 158, 258, 40, 550),
}

var map64SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("8216674186649650", "蓝咒石怪", "monstermap/bluemagicrock.swf", "bluemagicrock", 12, "战士", 334, 377),
	buildCapturedSourceMonster("8218674186650741", "蓝咒石怪", "monstermap/bluemagicrock.swf", "bluemagicrock", 12, "战士", 756, 466),
}

var map65SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("4710674219611654", "蓝咒石怪", "monstermap/bluemagicrock.swf", "bluemagicrock", 12, "战士", 383, 421),
	buildCapturedSourceMonster("4730674219612608", "蓝咒石怪", "monstermap/bluemagicrock.swf", "bluemagicrock", 12, "战士", 971, 526),
}

var map66SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("2579674249044327", "蓝咒石怪", "monstermap/bluemagicrock.swf", "bluemagicrock", 13, "战士", 398, 434),
	buildCapturedSourceMonster("2581674249044878", "蓝咒石怪", "monstermap/bluemagicrock.swf", "bluemagicrock", 13, "战士", 1142, 490),
	buildCapturedSourceMonster("2583674249045842", "蓝咒石怪", "monstermap/bluemagicrock.swf", "bluemagicrock", 14, "战士", 430, 507),
}

var map67SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("5375674287331644", "白咒石怪", "monstermap/baimagicrock.swf", "baimagicrock", 15, "战士", 1237, 527),
	buildCapturedSourceMonster("5377674287332736", "蓝咒石怪", "monstermap/bluemagicrock.swf", "bluemagicrock", 14, "战士", 334, 459),
	buildCapturedSourceMonster("5379674287333653", "蓝咒石怪", "monstermap/bluemagicrock.swf", "bluemagicrock", 14, "战士", 313, 556),
}

var map68SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("1509674533823464", "白咒石怪", "monstermap/baimagicrock.swf", "baimagicrock", 15, "战士", 515, 412),
	buildCapturedSourceMonster("1511674533825915", "白咒石怪", "monstermap/baimagicrock.swf", "baimagicrock", 15, "战士", 559, 569),
	buildCapturedSourceMonster("1513674533826981", "白咒石怪", "monstermap/baimagicrock.swf", "baimagicrock", 15, "战士", 1484, 512),
	buildCapturedSourceMonster("1515674533827497", "白咒石怪", "monstermap/baimagicrock.swf", "baimagicrock", 15, "战士", 1502, 600),
}

var map69SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("4255674575781235", "白咒石怪", "monstermap/baimagicrock.swf", "baimagicrock", 16, "游侠", 461, 442),
	buildCapturedSourceMonster("4257674575782978", "白咒石怪", "monstermap/baimagicrock.swf", "baimagicrock", 16, "游侠", 429, 605),
	buildCapturedSourceMonster("4259674575784252", "白咒石怪", "monstermap/baimagicrock.swf", "baimagicrock", 16, "战士", 2041, 419),
	buildCapturedSourceMonster("4261674575785819", "白咒石怪", "monstermap/baimagicrock.swf", "baimagicrock", 16, "战士", 2083, 519),
}

var map71SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("1183674677329555", "白咒石怪", "monstermap/baimagicrock.swf", "baimagicrock", 16, "游侠", 301, 391),
	buildCapturedSourceMonster("1185674677330655", "白咒石怪", "monstermap/baimagicrock.swf", "baimagicrock", 16, "游侠", 308, 473),
	buildCapturedSourceMonster("1187674677331194", "白咒石怪", "monstermap/baimagicrock.swf", "baimagicrock", 16, "战士", 2376, 509),
	buildCapturedSourceMonster("1189674677333369", "白咒石怪", "monstermap/baimagicrock.swf", "baimagicrock", 16, "战士", 2412, 624),
	buildCapturedSourceMonster("1191674677334956", "白咒石怪", "monstermap/baimagicrock.swf", "baimagicrock", 16, "游侠", 714, 455),
	buildCapturedSourceMonster("1193674677335390", "白咒石怪", "monstermap/baimagicrock.swf", "baimagicrock", 16, "游侠", 717, 556),
}

var map72SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("5313674881098281", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 17, "战士", 563, 513),
	buildCapturedSourceMonster("5315674881099944", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 17, "游侠", 1093, 608),
	buildCapturedSourceMonster("5317674881101168", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 17, "战士", 1403, 496),
	buildCapturedSourceMonster("5319674881102907", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 17, "战士", 1561, 576),
	buildCapturedSourceMonster("5321674881103452", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 17, "游侠", 2069, 500),
}

var map73SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("5579674739334441", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 17, "战士", 380, 373),
	buildCapturedSourceMonster("5581674739336439", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 17, "战士", 296, 446),
	buildCapturedSourceMonster("5583674739338294", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 17, "战士", 1092, 419),
	buildCapturedSourceMonster("5585674739340680", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 17, "战士", 1053, 507),
	buildCapturedSourceMonster("5587674739342989", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 17, "游侠", 2667, 502),
	buildCapturedSourceMonster("5589674739343164", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 17, "游侠", 2288, 547),
}

var map74SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("9031674933909671", "白咒石怪", "monstermap/baimagicrock.swf", "baimagicrock", 16, "游侠", 2284, 496),
	buildCapturedSourceMonster("9033674933911913", "白咒石怪", "monstermap/baimagicrock.swf", "baimagicrock", 16, "游侠", 2392, 576),
	buildCapturedSourceMonster("9035674933912540", "白咒石怪", "monstermap/baimagicrock.swf", "baimagicrock", 16, "战士", 573, 515),
	buildCapturedSourceMonster("9037674933914197", "白咒石怪", "monstermap/baimagicrock.swf", "baimagicrock", 16, "战士", 657, 603),
	buildCapturedSourceMonster("9039674933915532", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 17, "战士", 1480, 526),
	buildCapturedSourceMonster("9041674933916861", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 17, "战士", 1384, 603),
}

var map75SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("8110675527789273", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 18, "战士", 323, 476),
	buildCapturedSourceMonster("8130675527791587", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 18, "战士", 350, 596),
	buildCapturedSourceMonster("8150675527792799", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 17, "游侠", 511, 546),
	buildCapturedSourceMonster("8170675527794372", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 18, "战士", 1715, 492),
	buildCapturedSourceMonster("8190675527795636", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 18, "战士", 1676, 611),
	buildCapturedSourceMonster("8210675527797680", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 17, "游侠", 1811, 550),
}

var map76SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("1038675671970511", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 18, "战士", 1417, 541),
	buildCapturedSourceMonster("1040675671971889", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 18, "战士", 1571, 592),
	buildCapturedSourceMonster("1042675671973672", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 18, "战士", 1761, 535),
	buildCapturedSourceMonster("1044675671974869", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 17, "游侠", 817, 528),
	buildCapturedSourceMonster("1046675671975970", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 17, "游侠", 1020, 615),
	buildCapturedSourceMonster("1048675671977626", "巨岩魔", "monstermap/largerock.swf", "largerock", 20, "战士++", 1576, 515),
}

var map77SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("3028675136602500", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 18, "战士", 711, 496),
	buildCapturedSourceMonster("3030675136603700", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 18, "战士", 746, 600),
	buildCapturedSourceMonster("3032675136605872", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 17, "游侠", 1265, 538),
	buildCapturedSourceMonster("3034675136606406", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 18, "战士", 2230, 496),
	buildCapturedSourceMonster("3036675136607240", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 18, "战士", 2430, 607),
	buildCapturedSourceMonster("3038675136609295", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 17, "游侠", 2638, 573),
}

var map78SourceMonsters = []sourceMonsterEntry{
	buildCapturedSourceMonster("1675675260682596", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 17, "游侠", 2588, 584),
	buildCapturedSourceMonster("1677675260684828", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 17, "游侠", 2423, 623),
	buildCapturedSourceMonster("1679675260685862", "晶石怪", "monstermap/crystalrock.swf", "crystalrock", 17, "游侠", 2257, 576),
	buildCapturedSourceMonster("1681675260686878", "岩化魔人", "monstermap/magicrockman.swf", "magicrockman", 20, "游侠++", 2471, 576),
}

var capturedSourceMonsterMovements = map[string]RoleMovement{
	"8128205778897212": {Speed: 130, Angle: 358.9376829766436, Mode: 2},
	"8130205778898758": {Speed: 130, Angle: 177.72696898379588, Mode: 1},
	"8132205778899676": {Speed: 130, Angle: 359.6976498989142, Mode: 1},
	"1656205827185847": {Speed: 130, Angle: 0.17629357064717712, Mode: 1},
	"1658205827186196": {Speed: 130, Angle: 149.96672031129458, Mode: 1},
	"1660205827187303": {Speed: 130, Angle: 177.4266413042915, Mode: 1},
	"1662205827187552": {Speed: 130, Angle: 0.693760266366009, Mode: 1},
	"1664205827188995": {Speed: 130, Angle: 211.12246969276458, Mode: 1},
	"1666205827189909": {Speed: 130, Angle: 4.739813201914896, Mode: 1},
	"8112205902790159": {Speed: 130, Angle: 180, Mode: 1},
	"8114205902791857": {Speed: 130, Angle: 180, Mode: 2},
	"8116205902792836": {Speed: 130, Angle: 0, Mode: 1},
	"8118205902792830": {Speed: 130, Angle: 178.35402271153018, Mode: 2},
	"8120205902793205": {Speed: 130, Angle: 1.0648546391172092, Mode: 2},
	"8122205902794636": {Speed: 130, Angle: 180.52085637447797, Mode: 2},
	"4889205982270617": {Speed: 130, Angle: 2.640227681825575, Mode: 1},
	"4891205982270480": {Speed: 130, Angle: 1.2310954483496047, Mode: 1},
	"4893205982271973": {Speed: 130, Angle: 1.3236719665851033, Mode: 1},
	"4895205982272135": {Speed: 130, Angle: 270, Mode: 1},
	"4897205982273653": {Speed: 130, Angle: 0, Mode: 2},
	"4899205982273477": {Speed: 130, Angle: 3.0679239161799926, Mode: 1},
	"8424206376338175": {Speed: 130, Angle: 358.2343509018431, Mode: 2},
	"8426206376339691": {Speed: 130, Angle: 180, Mode: 2},
	"8428206376340510": {Speed: 130, Angle: 358.385123565852, Mode: 2},
	"8430206376341780": {Speed: 130, Angle: 179.6390856174283, Mode: 2},
	"8432206376342756": {Speed: 130, Angle: 180, Mode: 1},
	"4710206556985181": {Speed: 130, Angle: 2.556150004796946, Mode: 1},
	"4712206556985489": {Speed: 130, Angle: 178.26429541107162, Mode: 2},
	"4714206556986370": {Speed: 130, Angle: 179.325963102015, Mode: 2},
	"4716206556987104": {Speed: 130, Angle: 2.109942920593726, Mode: 2},
	"4718206556987620": {Speed: 130, Angle: 359.5802582233581, Mode: 1},
	"4720206556988825": {Speed: 130, Angle: 359.36901491791984, Mode: 2},
	"9556206735436383": {Speed: 130, Angle: 243.43494882292202, Mode: 1},
	"9558206735437433": {Speed: 130, Angle: 347.9885215240773, Mode: 1},
	"9560206735437619": {Speed: 130, Angle: 4.590543012598198, Mode: 1},
	"9562206735438199": {Speed: 130, Angle: 1.0681029719549366, Mode: 1},
	"9564206735439303": {Speed: 130, Angle: 180, Mode: 1},
	"9566206735440658": {Speed: 130, Angle: 181.43209618416465, Mode: 1},
	"9568206735440182": {Speed: 130, Angle: 26.07535558394876, Mode: 1},
	"5166206909805441": {Speed: 130, Angle: 1.780099783622664, Mode: 1},
	"5170206909806155": {Speed: 130, Angle: 135, Mode: 1},
	"5172206909807859": {Speed: 130, Angle: 2.984785658956835, Mode: 1},
	"5174206909807286": {Speed: 130, Angle: 0.8709838935552793, Mode: 2},
	"2764206074546810": {Speed: 130, Angle: 345.9637565320735, Mode: 1},
	"2768206074548639": {Speed: 130, Angle: 183.34936774972577, Mode: 1},
	"2770206074548423": {Speed: 130, Angle: 176.42366562500266, Mode: 1},
	"2890206197338884": {Speed: 130, Angle: 359.65963320693706, Mode: 1},
	"2892206197339825": {Speed: 130, Angle: 1.3726389921296542, Mode: 2},
	"2894206197340572": {Speed: 130, Angle: 358.0079063326333, Mode: 1},
	"2896206197341225": {Speed: 130, Angle: 177.68627791855844, Mode: 2},
	"2898206197342379": {Speed: 130, Angle: 2.6718648202670905, Mode: 1},
	"2900206197343711": {Speed: 130, Angle: 3.4873973083901544, Mode: 2},
}

var sourceTransportDialogue = sourceNPCDialogueEntry{
	Message: "可以瞬间传送至该处。",
	Answers: []AnswerOption{
		{Handle: "goto", Msg: "前往"},
		{Handle: "leave", Msg: "离开"},
	},
}

var sourceTransportLinks = []sourceTransportLink{
	{FromMapID: 3, ToMapID: 10, Slot: 1},
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
	{FromMapID: 18, ToMapID: 64, Slot: 1},
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
	// Current Feixiandong export has captured incoming links to map70, but no map70 outgoing rows.
	{FromMapID: 70, ToMapID: 68, Slot: 0},
	{FromMapID: 70, ToMapID: 72, Slot: 1},
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
			{Handle: "job_warrior", Msg: "选择【战士】"},
			{Handle: "job_sorcerer", Msg: "选择【术士】"},
			{Handle: "job_ranger", Msg: "选择【游侠】"},
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

var map2SourceNPCDialogueReplies = map[sourceNPCDialogueReplyKey]sourceNPCDialogueEntry{
	{
		Handle:       "4110542614676637",
		MsgHandle:    "1",
		AnswerHandle: "2q21gs",
	}: {
		MsgHandle: "2q21d_1",
		Message:   "人算不如天算，在这个节骨眼村西头的王大爷竟然食物中毒，偏偏那解毒丸又吃完了，在下又没有时间前去采摘，小侠若能帮我采些草药回来，在下感激不尽。",
		Answers: []AnswerOption{
			{Handle: "2q21a_1_1", Msg: "<m/>我这就去。"},
			{Handle: "2q21a_1_2", Msg: "<c/>关闭"},
		},
	},
	{
		Handle:       "4110542614676637",
		MsgHandle:    "1",
		AnswerHandle: "4q48os",
	}: {
		MsgHandle: "4q48d_2",
		Message:   "忘情药的配方？问世间情为何物，直叫人生死相许。若是世间真有忘情药，又何须叫人生死相许。",
		Answers: []AnswerOption{
			{Handle: "4q48a_2_1", Msg: "<m/>受教了。"},
			{Handle: "4q48a_2_2", Msg: "<c/>关闭"},
		},
	},
	{
		Handle:       "4110542614676637",
		MsgHandle:    "1",
		AnswerHandle: "4q69gs",
	}: {
		MsgHandle: "4q69d_1",
		Message:   "嗯，如果真不想受相思之苦的话，我倒是有一法可抵得上忘情之药，不过你得先替我完成一件事。",
		Answers: []AnswerOption{
			{Handle: "4q69a_1_1", Msg: "<m/>请讲。"},
			{Handle: "4q69a_1_2", Msg: "<c/>关闭"},
		},
	},
	{
		Handle:       "4110542614676637",
		MsgHandle:    "4q69d_1",
		AnswerHandle: "4q69a_1_1",
	}: {
		MsgHandle: "4q69d_2",
		Message:   "我前日出游寻得一珍贵的药材，名为((雪莲花))，可路途中((被那黄风二寨主夺走))，小侠若能((去【黄风寨】替我夺回雪莲花))，在下自会尽力相助。",
		Answers: []AnswerOption{
			{Handle: "4q69a_2_1", Msg: "<m/>我这就去。"},
			{Handle: "4q69a_2_2", Msg: "<c/>关闭"},
		},
	},
}

var map46SourceNPCDialogueReplies = map[sourceNPCDialogueReplyKey]sourceNPCDialogueEntry{
	{
		Handle:       "2220542612946566",
		MsgHandle:    "1",
		AnswerHandle: "1",
	}: {
		MsgHandle: "10",
		Message:   "你想学习什么职业的技能?",
		Answers: []AnswerOption{
			{Handle: "7", Msg: "战士技能"},
			{Handle: "8", Msg: "术士技能"},
			{Handle: "9", Msg: "游侠技能"},
			{Handle: "0", Msg: "<c/>下次再说"},
		},
	},
}
