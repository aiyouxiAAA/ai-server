package world

var map187SourceNPCs = []sourceNPCEntry{
	{
		Handle:      "7620542619919529",
		DisplayName: "莫坨坨",
		SourceQuery: "npc/莫坨坨.swf",
		SpriteName:  "motuotuo",
		Width:       109,
		Height:      150,
		SpawnFlash:  SpawnPoint{X: 519, Y: 299},
		GuildName:   "公会创建",
		GuildPic:    "5003",
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message: `((牧原号角声不绝，草坝何时还清闲？))
自从那些行为怪异的画皮族离开之后，在草坝一带很少能看见他们的行迹了。不过最近听说擎天山出了大事，草坝也出现了很多怪物，形势真是让人担忧啊，若画皮与怪物联手，牧原又是一场血雨腥风。`,
			Answers: []AnswerOption{
				{Handle: "7q12gs", Msg: "<m/>厨师与帅锅"},
				{Handle: "3", Msg: "创建公会"},
				{Handle: "x", Msg: "<c/>关闭"},
			},
		},
	},
	{
		Handle:      "7640542620118422",
		DisplayName: "胡澈",
		SourceQuery: "npc/胡澈.swf",
		SpriteName:  "huche",
		Width:       121,
		Height:      189,
		SpawnFlash:  SpawnPoint{X: 1130, Y: 423},
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message:   "我哥做事总是大起大落的，人都要被他折腾死。这几天不知道又会有什么大动作了。君子动口不动手，做事之前还是多多讨论为妙。",
			Answers: []AnswerOption{
				{Handle: "7q55gs", Msg: "<m/>种族起源"},
				{Handle: "0", Msg: "<c/>关闭"},
			},
		},
	},
	{
		Handle:      "7650542620201230",
		DisplayName: "骈釜子",
		SourceQuery: "npc/骈釜子.swf",
		SpriteName:  "pianfuzi",
		Width:       142,
		Height:      200,
		SpawnFlash:  SpawnPoint{X: 2200, Y: 450},
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message:   "这牧原镇也封闭了好多天了，局势紧张，都因为那些镇外的怪物，害我都不能搞到足够的食材烹饪了，万一镇长怪罪下来那可是我的失职。 ",
			Answers: []AnswerOption{
				{Handle: "7q12os", Msg: "<m/>大厨师，你好啊。"},
				{Handle: "0", Msg: "<c/>关闭"},
			},
		},
	},
	{
		Handle:      "7660542620215960",
		DisplayName: "妖术狐狸",
		SourceQuery: "npc/狐狸.swf",
		SpriteName:  "huli",
		Width:       58,
		Height:      111,
		SpawnFlash:  SpawnPoint{X: 735, Y: 400},
		GuildName:   "传送大师",
		GuildPic:    "5005",
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message:   "这((牧原镇))的居民大多是囊族人，他们擅长经商！少侠想去哪里？~",
			Answers: []AnswerOption{
				{Handle: "2", Msg: "激活本地传送点"},
				{Handle: "3", Msg: "传送到【乌梁营地】(铜钱x800)"},
				{Handle: "4", Msg: "传送到【灵霞村】(未激活！)"},
				{Handle: "help", Msg: "未激活是什么意思？"},
				{Handle: "0", Msg: "<c/>关闭（VIP可获每日免费传送次数）"},
			},
		},
	},
	{
		Handle:      "7630542620072660",
		DisplayName: "胡涞",
		SourceQuery: "npc/胡涞.swf",
		SpriteName:  "hulai",
		Width:       107,
		Height:      193,
		SpawnFlash:  SpawnPoint{X: 1881, Y: 445},
		GuildName:   "技能导师",
		GuildPic:    "5003",
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message: `光说不练假把势！
我是牧原镇内带兵总领胡涞，如若有人犯我牧原，我定将他杀得片甲不留。`,
			Answers: []AnswerOption{
				{Handle: "7q2os", Msg: "<ml><m/>胡军长，我来报道啦。"},
				{Handle: "1", Msg: "学习技能"},
				{Handle: "0", Msg: "<c/>关闭"},
			},
		},
	},
}

var map188SourceNPCs = []sourceNPCEntry{
	{
		Handle:      "7720542620445842",
		DisplayName: "郭菩",
		SourceQuery: "npc/郭菩.swf",
		SpriteName:  "guopu",
		Width:       101,
		Height:      158,
		SpawnFlash:  SpawnPoint{X: 1388, Y: 433},
		GuildName:   "医疗师",
		GuildPic:    "5002",
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message:   "我是牧原镇的药师郭菩，你若是受了伤，可以找我治疗。",
			Answers: []AnswerOption{
				{Handle: "day_11gs", Msg: "<party><m/>提神香丸"},
				{Handle: "2", Msg: "进行治疗"},
				{Handle: "1", Msg: "查看商店"},
				{Handle: "0", Msg: "<c/>取消"},
			},
		},
	},
	{
		Handle:      "7710542620292513",
		DisplayName: "介仕",
		SourceQuery: "npc/介仕.swf",
		SpriteName:  "jieshi",
		Width:       86,
		Height:      169,
		SpawnFlash:  SpawnPoint{X: 826, Y: 384},
		GuildName:   "道具商",
		GuildPic:    "5001",
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message:   "囊族与画皮的冲突历来已久，看似画皮与魔军联手了。你听说了没？镇外现在很危险，旅行中的同胞们已经带回了确凿的消息。",
			Answers: []AnswerOption{
				{Handle: "1", Msg: "道具商店"},
				{Handle: "3", Msg: "原石合成"},
				{Handle: "0", Msg: "<c/>取消"},
			},
		},
	},
	{
		Handle:      "7730542620459606",
		DisplayName: "矿点",
		SourceQuery: "npc/采矿点.swf",
		SpriteName:  "caikuangdian",
		Width:       73,
		Height:      60,
		SpawnFlash:  SpawnPoint{X: 1171, Y: 385},
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message:   "一个小小的采矿点，看起来好像刚被其他人开采过......",
			Answers: []AnswerOption{
				{Handle: "st", Msg: "【操作】采集矿石"},
				{Handle: "0", Msg: "<c/>取消"},
			},
		},
	},
}

var map189SourceNPCs = []sourceNPCEntry{
	{
		Handle:      "7480542619653453",
		DisplayName: "杜可筠",
		SourceQuery: "npc/杜可筠.swf",
		SpriteName:  "dukeyun",
		Width:       104,
		Height:      182,
		SpawnFlash:  SpawnPoint{X: 390, Y: 410},
		GuildName:   "交易行",
		GuildPic:    "5001",
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message:   "鄙人杜可筠，外号((万事通))。我开这二手交易行几十年了，什么宝贝没有见过？你有什么宝物没，速速给我过目，鉴定宝物可不是一件简单的事情。",
			Answers: []AnswerOption{
				{Handle: "1", Msg: "打开交易行"},
				{Handle: "0", Msg: "<c/>取消"},
			},
		},
	},
	{
		Handle:      "7470542619589250",
		DisplayName: "魁星泰斗",
		SourceQuery: "npc/魁星泰斗.swf",
		SpriteName:  "kuixingtai",
		Width:       176,
		Height:      221,
		SpawnFlash:  SpawnPoint{X: 2685, Y: 365},
		GuildName:   "锻造师",
		GuildPic:    "5000",
	},
	{
		Handle:      "7490542619656404",
		DisplayName: "通天八卦炉<ma>",
		SourceQuery: "npc/通天八卦炉.swf",
		SpriteName:  "bagualu",
		Width:       263,
		Height:      244,
		SpawnFlash:  SpawnPoint{X: 2105, Y: 380},
		GuildName:   "合成道具",
		GuildPic:    "5001",
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
		Handle:      "7460542619494385",
		DisplayName: "祝掌柜",
		SourceQuery: "npc/祝掌柜.swf",
		SpriteName:  "zhuzhanggui",
		Width:       141,
		Height:      183,
		SpawnFlash:  SpawnPoint{X: 1626, Y: 407},
		GuildName:   "仓库管理",
		GuildPic:    "5004",
		Dialogue: &sourceNPCDialogueEntry{
			MsgHandle: "1",
			Message:   "镇外兵荒马乱的，害我这小小的客栈生意都做不下去了，这位客官请问有什么需要帮忙的么？",
			Answers: []AnswerOption{
				{Handle: "7q48gs", Msg: "<m/>心有牵挂"},
				{Handle: "6", Msg: "使用仓库"},
				{Handle: "1", Msg: "住店绑定"},
				{Handle: "2", Msg: "收发信件"},
				{Handle: "0", Msg: "<c/>取消"},
			},
		},
	},
}
