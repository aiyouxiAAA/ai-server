package main

import (
	"log"
	"sort"
	"strconv"
	"strings"

	"ai-server/internal/quest"
	"ai-server/internal/session"
	"ai-server/internal/world"
)

const sourceMainQuestTitle = "初入云隐"
const sourceMainQuestNpcHandle = "1000542608713897"
const classicQuestAcceptedLimit = 20
const classicQuestAcceptedNpcState = 2

type classicQuestInfoPush struct {
	QuestID     string                 `json:"questId,omitempty"`
	Title       string                 `json:"title"`
	Level       int                    `json:"level"`
	Description string                 `json:"description"`
	State       string                 `json:"state"`
	Type        string                 `json:"type,omitempty"`
	Reward      classicQuestRewardPush `json:"reward,omitempty"`
}

type classicQuestClearPush struct {
	Title   string `json:"title"`
	QuestID string `json:"questId,omitempty"`
}

type classicQuestRemoveRequest struct {
	Title    string `json:"title,omitempty"`
	QuestID  string `json:"questId,omitempty"`
	Complete bool   `json:"complete,omitempty"`
}

type classicQuestRewardPush struct {
	Experience    int                          `json:"experience,omitempty"`
	Items         []classicQuestRewardItemPush `json:"items,omitempty"`
	Skills        []string                     `json:"skills,omitempty"`
	OptionalItems []classicQuestRewardItemPush `json:"optionalItems,omitempty"`
}

type classicQuestRewardItemPush struct {
	Name    string `json:"name"`
	Count   int    `json:"count"`
	Display string `json:"display,omitempty"`
}

type classicQuestAnswerRoute struct {
	handle       string
	msgHandle    string
	answerHandle string
	title        string
}

func buildClassicQuestLogResult(store *session.Store, socketSession *packetSession) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic quest GetQuestLog ignored without selected role")
		return packetResult{handled: true}
	}

	accepted := map[string]bool{}
	if store != nil {
		accepted = store.AcceptedQuestTitles(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID)
	}
	infos := make([]classicQuestInfoPush, 0, len(accepted))
	for _, info := range quest.All() {
		if !accepted[info.Title] {
			continue
		}
		infos = append(infos, classicQuestInfoFromCatalog(info))
	}
	return packetResult{
		questInfos: infos,
		handled:    true,
	}
}

func buildClassicQuestRemoveResult(store *session.Store, socketSession *packetSession, request classicQuestRemoveRequest) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic quest RemoveQuest ignored without selected role title=%s questId=%s", request.Title, request.QuestID)
		return packetResult{handled: true}
	}

	title := strings.TrimSpace(request.Title)
	info, ok := quest.Info{}, false
	if questID := strings.TrimSpace(request.QuestID); questID != "" {
		info, ok = quest.FindByID(questID)
		if ok && title == "" {
			title = info.Title
		}
	}
	if !ok && title != "" {
		info, ok = quest.FindByTitle(title)
	}
	if !ok || title == "" {
		log.Printf("[ai-server] classic quest RemoveQuest ignored missing title=%s questId=%s", request.Title, request.QuestID)
		return packetResult{handled: true}
	}

	if request.Complete {
		return buildClassicQuestCompleteResult(store, socketSession, info, title)
	}

	if store == nil || !store.MarkQuestRemoved(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, title) {
		return packetResult{handled: true}
	}

	result := packetResult{
		questClears: []classicQuestClearPush{{
			Title:   title,
			QuestID: info.ID,
		}},
		handled: true,
	}
	if handle := questNpcHandleForTitle(title); handle != "" {
		result.questStates = []world.QuestStatePush{{
			Handle: handle,
			State:  0,
		}}
	}
	if request.Complete {
		result.chatMessages = append(result.chatMessages, classicTownSystemChatMessage("完成了任务【"+title+"】。"))
		applyClassicQuestReward(store, socketSession, info, &result)
	} else {
		result.chatMessages = append(result.chatMessages, classicTownSystemChatMessage("放弃了任务【"+title+"】。"))
	}
	return result
}

func buildClassicQuestAnswerResult(store *session.Store, socketSession *packetSession, request classicTownAnswerRequest) (packetResult, bool) {
	route, ok := findClassicQuestAnswerRoute(request)
	if !ok {
		return packetResult{}, false
	}
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic quest Answer ignored without selected role handle=%s answerHandle=%s", request.Handle, request.AnswerHandle)
		return packetResult{handled: true}, true
	}
	info, ok := quest.FindByTitle(route.title)
	if !ok {
		log.Printf("[ai-server] classic quest Answer ignored missing catalog title=%s handle=%s answerHandle=%s", route.title, request.Handle, request.AnswerHandle)
		return packetResult{handled: true}, true
	}
	if store == nil {
		return packetResult{handled: true}, true
	}

	accepted := store.AcceptedQuestTitles(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID)
	if accepted[info.Title] {
		if quest.IsCompletableState(info.State) {
			return buildClassicQuestCompleteResult(store, socketSession, info, info.Title), true
		}
		return packetResult{
			questInfos: []classicQuestInfoPush{classicQuestInfoFromCatalog(info)},
			chatMessages: []classicTownChatMessagePush{
				classicTownSystemChatMessage("日志更新"),
			},
			handled: true,
		}, true
	}
	if len(accepted) >= classicQuestAcceptedLimit {
		return packetResult{
			chatMessages: []classicTownChatMessagePush{
				classicTownSystemWarningMessage("任务列表已满，请先放弃部分任务。"),
			},
			handled: true,
		}, true
	}
	if !store.AcceptQuest(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, info.Title) {
		return packetResult{handled: true}, true
	}

	result := packetResult{
		questInfos: []classicQuestInfoPush{classicQuestInfoFromCatalog(info)},
		chatMessages: []classicTownChatMessagePush{
			classicTownSystemChatMessage("接受了任务【" + info.Title + "】。"),
			classicTownSystemChatMessage("日志更新"),
		},
		handled: true,
	}
	appendQuestStateForInfoOrRoute(&result, info, route, classicQuestAcceptedNpcState)
	return result, true
}

func buildClassicQuestCompleteResult(store *session.Store, socketSession *packetSession, info quest.Info, title string) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic quest CompleteQuest ignored without selected role title=%s questId=%s", title, info.ID)
		return packetResult{handled: true}
	}
	if !quest.IsCompletableState(info.State) {
		log.Printf("[ai-server] classic quest CompleteQuest rejected incomplete title=%s questId=%s state=%s", title, info.ID, info.State)
		return packetResult{handled: true}
	}
	if store == nil {
		return packetResult{handled: true}
	}

	completeResult := store.CompleteQuest(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, title, classicQuestRequirementItems(info.Requirements))
	if !completeResult.Found {
		log.Printf("[ai-server] classic quest CompleteQuest ignored missing role title=%s questId=%s", title, info.ID)
		return packetResult{handled: true}
	}
	if !completeResult.Completed {
		log.Printf("[ai-server] classic quest CompleteQuest rejected title=%s questId=%s error=%s", title, info.ID, completeResult.ErrorCode)
		if strings.TrimSpace(completeResult.ErrorMessage) == "" {
			return packetResult{handled: true}
		}
		return packetResult{
			chatMessages: []classicTownChatMessagePush{classicTownSystemWarningMessage(completeResult.ErrorMessage)},
			handled:      true,
		}
	}

	socketSession.selectedRole = &completeResult.Role
	socketSession.playerBase = &completeResult.PlayerBase
	result := packetResult{
		questClears: []classicQuestClearPush{{
			Title:   title,
			QuestID: info.ID,
		}},
		itemInfos:  make([]classicTownItemInfoPush, 0, len(completeResult.UpdatedItems)),
		itemClears: make([]classicTownItemInfoClearPush, 0, len(completeResult.ClearedItems)),
		chatMessages: []classicTownChatMessagePush{
			classicTownSystemChatMessage("完成了任务【" + title + "】。"),
		},
		handled: true,
	}
	for _, clear := range completeResult.ClearedItems {
		result.itemClears = append(result.itemClears, classicTownItemInfoClearPush{
			Handle: completeResult.Role.RoleID,
			Type:   clear.Type,
			Index:  clear.Index,
		})
	}
	for _, item := range completeResult.UpdatedItems {
		item.Handle = completeResult.Role.RoleID
		result.itemInfos = append(result.itemInfos, classicTownItemInfoPushFromRoleItem(item))
	}
	if handle := questNpcHandleForTitle(title); handle != "" {
		result.questStates = []world.QuestStatePush{{
			Handle: handle,
			State:  0,
		}}
	}
	applyClassicQuestReward(store, socketSession, info, &result)
	return result
}

func sourceMainQuestInfo() classicQuestInfoPush {
	return classicQuestInfoPush{
		Title:       sourceMainQuestTitle,
		Level:       1,
		Description: "<ml>拜访一心长态<br/>[g]=前往云隐村，和一心长态交谈，熟悉江湖任务委托。",
		State:       "进行中",
	}
}

func classicQuestInfoFromCatalog(info quest.Info) classicQuestInfoPush {
	return classicQuestInfoPush{
		QuestID:     info.ID,
		Title:       info.Title,
		Level:       info.Level,
		Description: info.Description,
		State:       info.State,
		Type:        info.Type,
		Reward:      classicQuestRewardPushFromReward(info.Reward),
	}
}

func questNpcHandleForTitle(title string) string {
	if strings.TrimSpace(title) == sourceMainQuestTitle {
		return sourceMainQuestNpcHandle
	}
	if info, ok := quest.FindByTitle(title); ok {
		return questNpcHandleForInfo(info)
	}
	return ""
}

func questNpcHandleForInfo(info quest.Info) string {
	if strings.TrimSpace(info.QuestStateHandle) != "" {
		return strings.TrimSpace(info.QuestStateHandle)
	}
	if len(info.Routes) > 0 {
		return info.Routes[0].Handle
	}
	return ""
}

func appendQuestStateForInfo(result *packetResult, info quest.Info, state int) {
	appendQuestStateForHandle(result, questNpcHandleForInfo(info), state)
}

func appendQuestStateForInfoOrRoute(result *packetResult, info quest.Info, route classicQuestAnswerRoute, state int) {
	handle := questNpcHandleForInfo(info)
	if handle == "" {
		handle = strings.TrimSpace(route.handle)
	}
	appendQuestStateForHandle(result, handle, state)
}

func appendQuestStateForHandle(result *packetResult, handle string, state int) {
	if result == nil {
		return
	}
	if handle = strings.TrimSpace(handle); handle != "" {
		result.questStates = append(result.questStates, world.QuestStatePush{
			Handle: handle,
			State:  state,
		})
	}
}

func applyAcceptedQuestStatesToBootstrap(snapshot *world.TownBootstrapSnapshot, store *session.Store, socketSession *packetSession) {
	applyQuestStateOverrides(snapshot, acceptedQuestStatePushes(store, socketSession))
}

func acceptedQuestStatePushes(store *session.Store, socketSession *packetSession) []world.QuestStatePush {
	if store == nil || socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		return nil
	}
	accepted := store.AcceptedQuestTitles(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID)
	if len(accepted) == 0 {
		return nil
	}

	stateByHandle := map[string]int{}
	for _, info := range quest.All() {
		if !accepted[info.Title] {
			continue
		}
		handle := questNpcHandleForInfo(info)
		if handle == "" {
			continue
		}
		stateByHandle[handle] = classicQuestAcceptedNpcState
	}
	if len(stateByHandle) == 0 {
		return nil
	}

	handles := make([]string, 0, len(stateByHandle))
	for handle := range stateByHandle {
		handles = append(handles, handle)
	}
	sort.Strings(handles)

	pushes := make([]world.QuestStatePush, 0, len(handles))
	for _, handle := range handles {
		pushes = append(pushes, world.QuestStatePush{
			Handle: handle,
			State:  stateByHandle[handle],
		})
	}
	return pushes
}

func applyQuestStateOverrides(snapshot *world.TownBootstrapSnapshot, overrides []world.QuestStatePush) {
	if snapshot == nil || len(overrides) == 0 {
		return
	}

	questStateIndexByHandle := make(map[string]int, len(snapshot.QuestStates))
	for index, questState := range snapshot.QuestStates {
		questStateIndexByHandle[questState.Handle] = index
	}
	createRoleHandles := make(map[string]bool, len(snapshot.CreateRoles))
	for _, role := range snapshot.CreateRoles {
		createRoleHandles[role.Handle] = true
	}

	for _, override := range overrides {
		if strings.TrimSpace(override.Handle) == "" {
			continue
		}
		if index, ok := questStateIndexByHandle[override.Handle]; ok {
			if snapshot.QuestStates[index].State == 0 {
				snapshot.QuestStates[index].State = override.State
			}
			continue
		}
		if createRoleHandles[override.Handle] {
			snapshot.QuestStates = append(snapshot.QuestStates, override)
			questStateIndexByHandle[override.Handle] = len(snapshot.QuestStates) - 1
		}
	}
}

func findClassicQuestAnswerRoute(request classicTownAnswerRequest) (classicQuestAnswerRoute, bool) {
	handle := strings.TrimSpace(request.Handle)
	msgHandle := strings.TrimSpace(request.MsgHandle)
	answerHandle := strings.TrimSpace(request.AnswerHandle)
	for _, info := range quest.All() {
		for _, route := range info.Routes {
			if route.Handle == handle && route.MsgHandle == msgHandle && route.AnswerHandle == answerHandle {
				return classicQuestAnswerRoute{
					handle:       route.Handle,
					msgHandle:    route.MsgHandle,
					answerHandle: route.AnswerHandle,
					title:        info.Title,
				}, true
			}
		}
	}
	for _, route := range classicQuestAnswerRoutes {
		if route.handle == handle && route.msgHandle == msgHandle && route.answerHandle == answerHandle {
			return route, true
		}
	}
	return classicQuestAnswerRoute{}, false
}

var classicQuestAnswerRoutes = []classicQuestAnswerRoute{
	{handle: "1000542608713897", msgHandle: "1", answerHandle: "1q32gs", title: "飞仙洞弑炼"},
	{handle: "3000542609015823", msgHandle: "1", answerHandle: "1q19gs", title: "准备柴火"},
	{handle: "5000542609232627", msgHandle: "1", answerHandle: "1q22gs", title: "消灭刺鸟"},
	{handle: "6000542609425103", msgHandle: "1", answerHandle: "1q21gs", title: "采集草药"},
	{handle: "7000542609490978", msgHandle: "1", answerHandle: "1q23gs", title: "丑七品的梦"},
	{handle: "2000542608832485", msgHandle: "1q28d_1", answerHandle: "1q28a_1_1", title: "全民锻造"},
	{handle: "4110542614676637", msgHandle: "2q21d_1", answerHandle: "2q21a_1_1", title: "山谷采药"},
	{handle: "4110542614676637", msgHandle: "4q69d_2", answerHandle: "4q69a_2_1", title: "奇珍雪莲"},
}

func classicQuestRewardPushFromReward(reward quest.Reward) classicQuestRewardPush {
	result := classicQuestRewardPush{
		Experience:    reward.Experience,
		Items:         classicQuestRewardItemPushes(reward.Items),
		OptionalItems: classicQuestRewardItemPushes(reward.OptionalItems),
	}
	for _, skill := range reward.Skills {
		if strings.TrimSpace(skill.Name) != "" {
			result.Skills = append(result.Skills, strings.TrimSpace(skill.Name))
		}
	}
	return result
}

func classicQuestRewardItemPushes(items []quest.RewardItem) []classicQuestRewardItemPush {
	result := make([]classicQuestRewardItemPush, 0, len(items))
	for _, item := range items {
		result = append(result, classicQuestRewardItemPush{
			Name:    item.Name,
			Count:   item.Count,
			Display: item.Display,
		})
	}
	return result
}

func classicQuestRequirementItems(items []quest.RewardItem) []session.RoleItemRequirement {
	result := make([]session.RoleItemRequirement, 0, len(items))
	for _, item := range items {
		count := item.Count
		if count <= 0 {
			count = 1
		}
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		result = append(result, session.RoleItemRequirement{
			Name:  name,
			Count: count,
		})
	}
	return result
}

func applyClassicQuestReward(store *session.Store, socketSession *packetSession, info quest.Info, result *packetResult) {
	if store == nil || socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil || result == nil {
		return
	}
	playerID := socketSession.playerBase.PlayerID
	roleID := socketSession.selectedRole.RoleID
	entries := info.RewardEntries
	if len(entries) == 0 {
		entries = quest.BuildRewardEntries(quest.RewardEntrySourceQuest, info.ID, info.Reward)
	}

	for _, entry := range entries {
		if entry.Phase != quest.RewardEntryPhaseGrant {
			continue
		}
		if entry.Probability != quest.RewardEntryProbabilityCertain {
			log.Printf("[ai-server] classic quest reward entry skipped non-deterministic questId=%s kind=%s name=%s probability=%d", info.ID, entry.Kind, entry.Name, entry.Probability)
			continue
		}
		count := classicQuestRewardEntryCount(entry)
		switch entry.Kind {
		case quest.RewardEntryKindExperience:
			expResult := store.GrantRoleExperience(playerID, roleID, count)
			if expResult.Granted {
				socketSession.selectedRole = &expResult.Role
				socketSession.playerBase = &expResult.PlayerBase
				result.roleState = &expResult.RoleState
				result.chatMessages = append(result.chatMessages, classicTownSystemChatMessage("获得经验:"+strconv.Itoa(count)))
				if expResult.LevelChanged {
					result.rolePhysique = &expResult.RolePhysique
				}
			}
		case quest.RewardEntryKindCurrency:
			if currencies, ok := store.AddRoleCurrency(playerID, roleID, entry.Name, count); ok {
				result.currencyPush = buildClassicTownCurrencyPush(roleID, currencies)
				result.chatMessages = append(result.chatMessages, classicTownSystemChatMessage(classicQuestRewardItemSystemMessage(classicQuestRewardItemFromEntry(entry))))
			}
		case quest.RewardEntryKindItem:
			item := classicQuestRewardItemFromEntry(entry)
			rewardItem := classicQuestRewardRoleItem(item)
			granted, ok := store.GrantRoleItem(playerID, roleID, rewardItem)
			if !ok {
				log.Printf("[ai-server] classic quest reward item grant failed roleId=%s questId=%s item=%s count=%d", roleID, info.ID, item.Name, item.Count)
				continue
			}
			granted.Handle = roleID
			result.itemInfos = append(result.itemInfos, classicTownItemInfoPushFromRoleItem(granted))
			result.chatMessages = append(result.chatMessages, classicTownSystemChatMessage(classicQuestRewardItemSystemMessage(item)))
		case quest.RewardEntryKindSkill:
			roleSkill, ok := classicQuestRewardRoleSkill(quest.RewardSkill{Name: entry.Name})
			if !ok {
				log.Printf("[ai-server] classic quest reward skill missing template questId=%s skill=%s", info.ID, entry.Name)
				continue
			}
			_, skillCap, found, learned := store.LearnRoleSkill(playerID, roleID, roleSkill)
			if !found || !learned {
				continue
			}
			result.skillCap = &classicTownSkillCapPush{Count: skillCap}
			result.skillInfos = append(result.skillInfos, classicTownSkillInfoPushFromRoleSkill(roleID, roleSkill))
			result.chatMessages = append(result.chatMessages, classicTownSystemChatMessage("习得【"+roleSkill.Name+"】Lv."+strconv.Itoa(roleSkill.Level)))
		}
	}

	if selectedRole, playerBase, ok := store.GetRoleRuntimeData(playerID, roleID); ok {
		socketSession.selectedRole = &selectedRole
		socketSession.playerBase = &playerBase
	}
}

func classicQuestRewardEntryCount(entry quest.RewardEntry) int {
	if entry.CountMin > 0 {
		return entry.CountMin
	}
	if entry.CountMax > 0 {
		return entry.CountMax
	}
	return 1
}

func classicQuestRewardItemFromEntry(entry quest.RewardEntry) quest.RewardItem {
	return quest.RewardItem{
		Name:        entry.Name,
		Count:       classicQuestRewardEntryCount(entry),
		Display:     entry.Display,
		Description: entry.Description,
		SourceMeta:  entry.SourceMeta,
	}
}

func classicQuestRewardItemSystemMessage(item quest.RewardItem) string {
	count := item.Count
	if count <= 0 {
		count = 1
	}
	return "获得了【" + item.Name + "】x" + strconv.Itoa(count)
}

func classicQuestRewardRoleItem(item quest.RewardItem) session.RoleItem {
	count := item.Count
	if count <= 0 {
		count = 1
	}
	if template, ok := session.CapturedRoleItemTemplate(item.Name); ok {
		template.Type = classicTownBagContainerType
		template.Index = -1
		template.Count = count
		return template
	}
	return session.RoleItem{
		Type:        classicTownBagContainerType,
		Name:        item.Name,
		ItemType:    "null",
		Display:     item.Display,
		Description: item.SourceMeta,
		Count:       count,
		Index:       -1,
		ItemLevel:   1,
	}
}

func classicQuestRewardRoleSkill(skill quest.RewardSkill) (session.RoleSkill, bool) {
	switch strings.TrimSpace(skill.Name) {
	case "密斩":
		return session.RoleSkill{
			Name:        "密斩",
			Level:       1,
			Type:        "oneE",
			Icon:        "426.png",
			Description: "f_s_密斩&9@单体·攻击&7@3&10@单刀/单斧&22@战斗&2@5&4@提升40%的物理伤害",
		}, true
	default:
		return session.RoleSkill{}, false
	}
}
