package main

import (
	"strings"

	"ai-server/internal/quest"
	"ai-server/internal/session"
	"ai-server/internal/world"
)

const (
	classicWuliangHanXiongHandle = "6350542618650282"
	classicWuliangXuZhongHandle  = "6360542618722932"
	classicWuliangYuMoHandle     = "6370542618853300"
)

type classicWuliangQuestMenuEntry struct {
	handle            string
	title             string
	start             *world.AnswerOption
	active            *world.AnswerOption
	complete          *world.AnswerOption
	requiresCompleted []string
}

var classicWuliangQuestMenuEntries = []classicWuliangQuestMenuEntry{
	{
		handle: classicWuliangHanXiongHandle,
		title:  "侦查敌营",
		start:  classicWuliangQuestOption("6q2gs", "<ml><m/>侦查敌营"),
		complete: classicWuliangQuestOption(
			"6q2os",
			"<ml><m/>我已消灭了机木斧兵。",
		),
	},
	{
		handle:            classicWuliangHanXiongHandle,
		title:             "玄机之件",
		start:             classicWuliangQuestOption("6q3gs", "<ml><m/>玄机之件"),
		complete:          classicWuliangQuestOption("6q3os", "<ml><m/>我把玄机件拿来了。"),
		requiresCompleted: []string{"侦查敌营"},
	},
	{
		handle: classicWuliangHanXiongHandle,
		title:  "拜访故人",
		start:  classicWuliangQuestOption("6q40gs", "<m/>拜访故人"),
	},
	{
		handle:   classicWuliangHanXiongHandle,
		title:    "回赠特产",
		complete: classicWuliangQuestOption("6q41os", "<m/>这是叶眉赠你的特产。"),
	},
	{
		handle:            classicWuliangHanXiongHandle,
		title:             "暗力之源",
		start:             classicWuliangQuestOption("6q8gs", "<ml><m/>暗力之源"),
		complete:          classicWuliangQuestOption("6q8os", "<ml><m/>我把暗力之源拿来了。"),
		requiresCompleted: []string{"玄机之件"},
	},
	{
		handle:            classicWuliangHanXiongHandle,
		title:             "乌梁工匠",
		start:             classicWuliangQuestOption("6q9gs", "<ml><m/>乌梁工匠"),
		requiresCompleted: []string{"暗力之源"},
	},
	{
		handle: classicWuliangHanXiongHandle,
		title:  "水车图纸",
		start:  classicWuliangQuestOption("6q15gs", "<ml><m/>水车图纸"),
	},
	{
		handle: classicWuliangHanXiongHandle,
		title:  "乌梁药师",
		start:  classicWuliangQuestOption("6q4gs", "<ml><m/>乌梁药师"),
	},
	{
		handle:   classicWuliangHanXiongHandle,
		title:    "汉雄的疑惑",
		complete: classicWuliangQuestOption("6q28os", "<ml><m/>军策找我有事？"),
	},
	{
		handle:   classicWuliangHanXiongHandle,
		title:    "修复石坝",
		complete: classicWuliangQuestOption("6q30os", "<ml><m/>我来商量石坝之事。"),
	},
	{
		handle:   classicWuliangHanXiongHandle,
		title:    "竣工喜讯",
		complete: classicWuliangQuestOption("6q18os", "<ml><m/>水车已经竣工了。"),
	},
	{
		handle: classicWuliangHanXiongHandle,
		title:  "石头密语",
		start:  classicWuliangQuestOption("6q24gs", "<ml><m/>石头密语"),
	},
	{
		handle:   classicWuliangXuZhongHandle,
		title:    "乌梁药师",
		complete: classicWuliangQuestOption("6q4os", "<ml><m/>久仰独臂神医的大名啊。"),
	},
	{
		handle:            classicWuliangXuZhongHandle,
		title:             "草药之策",
		start:             classicWuliangQuestOption("6q5gs", "<ml><m/>草药之策"),
		complete:          classicWuliangQuestOption("6q5os", "<ml><m/>我把金银花拿来了。"),
		requiresCompleted: []string{"乌梁药师"},
	},
	{
		handle:            classicWuliangXuZhongHandle,
		title:             "营长之令",
		start:             classicWuliangQuestOption("6q6gs", "<ml><m/>营长之令"),
		requiresCompleted: []string{"草药之策"},
	},
	{
		handle:   classicWuliangXuZhongHandle,
		title:    "军令如山",
		complete: classicWuliangQuestOption("6q52os", "<m/>你违反了禁酒令，应杖责五十。"),
	},
	{
		handle:            classicWuliangXuZhongHandle,
		title:             "五花香肉",
		start:             classicWuliangQuestOption("6q53gs", "<m/>五花香肉"),
		complete:          classicWuliangQuestOption("6q53os", "<m/>肉拿来了。"),
		requiresCompleted: []string{"军令如山"},
	},
	{
		handle:            classicWuliangXuZhongHandle,
		title:             "受伤的部位",
		start:             classicWuliangQuestOption("6q54gs", "<m/>受伤的部位"),
		complete:          classicWuliangQuestOption("6q54os", "<m/>当归拿来了。"),
		requiresCompleted: []string{"五花香肉"},
	},
	{
		handle:   classicWuliangYuMoHandle,
		title:    "乌梁工匠",
		complete: classicWuliangQuestOption("6q9os", "<ml><m/>商量切断水源之事。"),
	},
	{
		handle:            classicWuliangYuMoHandle,
		title:             "造坝截水",
		start:             classicWuliangQuestOption("6q10gs", "<ml><m/>造坝截水"),
		complete:          classicWuliangQuestOption("6q10os", "<ml><m/>我把石块拿来了。"),
		requiresCompleted: []string{"乌梁工匠"},
	},
	{
		handle:            classicWuliangYuMoHandle,
		title:             "水坝建成",
		start:             classicWuliangQuestOption("6q11os", "<ml><m/>水坝建成"),
		requiresCompleted: []string{"造坝截水"},
	},
	{
		handle:   classicWuliangYuMoHandle,
		title:    "水车图纸",
		complete: classicWuliangQuestOption("6q15os", "<ml><m/>这是水车的图纸。"),
	},
	{
		handle:            classicWuliangYuMoHandle,
		title:             "建造水车",
		start:             classicWuliangQuestOption("6q16gs", "<ml><m/>建造水车"),
		complete:          classicWuliangQuestOption("6q16os", "<ml><m/>我把木材拿来了。"),
		requiresCompleted: []string{"水车图纸"},
	},
	{
		handle:   classicWuliangYuMoHandle,
		title:    "精巧零件",
		complete: classicWuliangQuestOption("6q17os", "<ml><m/>我把玄机精件拿来了。"),
	},
	{
		handle:            classicWuliangYuMoHandle,
		title:             "竣工喜讯",
		start:             classicWuliangQuestOption("6q18gs", "<ml><m/>竣工喜讯"),
		requiresCompleted: []string{"精巧零件"},
	},
	{
		handle:            classicWuliangYuMoHandle,
		title:             "机木玄文",
		start:             classicWuliangQuestOption("6q25gs", "<m/>机木玄文"),
		active:            classicWuliangQuestOption("6q25ns", "<mn/>机木玄文【进行中】"),
		complete:          classicWuliangQuestOption("6q25os", "<m/>我把玄文拿来了。"),
		requiresCompleted: []string{"竣工喜讯"},
	},
	{
		handle:   classicWuliangYuMoHandle,
		title:    "打造家具",
		complete: classicWuliangQuestOption("6q45os", "<m/>丑四品想让你打造家具。"),
	},
	{
		handle:            classicWuliangYuMoHandle,
		title:             "辅料木材",
		start:             classicWuliangQuestOption("6q46gs", "<m/>辅料木材"),
		complete:          classicWuliangQuestOption("6q46os", "<m/>木材拿来了。"),
		requiresCompleted: []string{"打造家具"},
	},
	{
		handle:            classicWuliangYuMoHandle,
		title:             "家具新革命",
		start:             classicWuliangQuestOption("6q47gs", "<m/>家具新革命"),
		requiresCompleted: []string{"辅料木材"},
	},
	{
		handle:            classicWuliangYuMoHandle,
		title:             "灵异事件",
		start:             classicWuliangQuestOption("6q50gs", "<m/>灵异事件"),
		requiresCompleted: []string{"竣工喜讯"},
	},
}

func buildClassicTownQuestAwareAnswerSpeak(store *session.Store, socketSession *packetSession, handle string) world.AnswerSpeakPush {
	answerSpeak := world.BuildAnswerSpeak(handle)
	questAnswers := buildClassicWuliangQuestMenuOptions(store, socketSession, handle)
	if len(questAnswers) == 0 {
		return answerSpeak
	}

	taskHandles := classicWuliangQuestMenuOptionHandles(handle)
	answers := make([]world.AnswerOption, 0, len(questAnswers)+len(answerSpeak.Answers))
	answers = append(answers, questAnswers...)
	for _, answer := range answerSpeak.Answers {
		if !taskHandles[answer.Handle] {
			answers = append(answers, answer)
		}
	}
	answerSpeak.Answers = answers
	return answerSpeak
}

func buildClassicWuliangQuestMenuOptions(store *session.Store, socketSession *packetSession, handle string) []world.AnswerOption {
	if !isClassicWuliangQuestNPC(handle) || store == nil || socketSession == nil || socketSession.playerBase == nil || socketSession.selectedRole == nil {
		return nil
	}

	accepted := store.AcceptedQuestTitles(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID)
	removed := store.RemovedQuestTitles(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID)
	options := []world.AnswerOption{}
	for _, entry := range classicWuliangQuestMenuEntries {
		if entry.handle != handle {
			continue
		}
		if accepted[entry.title] {
			option := entry.complete
			if !classicWuliangQuestRequirementsReady(store, socketSession, entry.title) && entry.active != nil {
				option = entry.active
			}
			if classicWuliangQuestMenuOptionSupported(handle, option) {
				options = append(options, *option)
			}
			continue
		}
		if removed[entry.title] || entry.start == nil || !classicWuliangQuestPrerequisitesMet(entry.requiresCompleted, removed) {
			continue
		}
		if classicWuliangQuestMenuOptionSupported(handle, entry.start) {
			options = append(options, *entry.start)
		}
	}
	return options
}

func classicWuliangQuestOption(handle string, message string) *world.AnswerOption {
	return &world.AnswerOption{Handle: handle, Msg: message}
}

func classicWuliangQuestMenuOptionHandles(handle string) map[string]bool {
	handles := map[string]bool{}
	for _, entry := range classicWuliangQuestMenuEntries {
		if entry.handle != handle {
			continue
		}
		for _, option := range []*world.AnswerOption{entry.start, entry.active, entry.complete} {
			if option != nil {
				handles[option.Handle] = true
			}
		}
	}
	return handles
}

func classicWuliangQuestMenuOptionSupported(handle string, option *world.AnswerOption) bool {
	if option == nil {
		return false
	}
	if world.BuildAnswerReply(handle, "1", option.Handle) != nil {
		return true
	}
	_, ok := findClassicQuestAnswerRoute(classicTownAnswerRequest{
		Handle:       handle,
		MsgHandle:    "1",
		AnswerHandle: option.Handle,
	})
	return ok
}

func classicWuliangQuestPrerequisitesMet(required []string, removed map[string]bool) bool {
	for _, title := range required {
		if !removed[title] {
			return false
		}
	}
	return true
}

func classicWuliangQuestRequirementsReady(store *session.Store, socketSession *packetSession, title string) bool {
	info, ok := quest.FindByTitle(title)
	if !ok || len(info.Requirements) == 0 {
		return true
	}
	role, _, ok := store.GetRoleRuntimeData(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID)
	if !ok {
		return false
	}

	countByName := map[string]int{}
	for _, item := range role.Items {
		if strings.TrimSpace(item.Type) == "背包" && item.Count > 0 {
			countByName[strings.TrimSpace(item.Name)] += item.Count
		}
	}
	for _, requirement := range info.Requirements {
		if countByName[strings.TrimSpace(requirement.Name)] < requirement.Count {
			return false
		}
	}
	return true
}

func isClassicWuliangQuestNPC(handle string) bool {
	switch handle {
	case classicWuliangHanXiongHandle, classicWuliangXuZhongHandle, classicWuliangYuMoHandle:
		return true
	default:
		return false
	}
}
