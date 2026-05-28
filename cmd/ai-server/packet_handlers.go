package main

import (
	"log"
	"strconv"
	"strings"

	"ai-server/internal/battle"
	"ai-server/internal/guild"
	"ai-server/internal/mall"
	"ai-server/internal/protocol"
	"ai-server/internal/session"
	"ai-server/internal/world"
)

type packetResult struct {
	responseCmd       uint64
	responsePayload   []byte
	townBootstrap     *world.TownBootstrapSnapshot
	answerSpeak       *world.AnswerSpeakPush
	createPlayer      *world.RolePush
	roleState         *session.RoleState
	rolePhysique      *session.RolePhysique
	skillCap          *classicTownSkillCapPush
	skillInfos        []classicTownSkillInfoPush
	skillShop         *classicTownSkillShopPush
	currencyPush      *classicTownCurrencyPush
	fastPanel         *classicTownFastPanelPush
	buySkillResult    *classicTownBuySkillResultPush
	containerCap      *classicTownContainerCapacityPush
	itemInfos         []classicTownItemInfoPush
	itemClears        []classicTownItemInfoClearPush
	questInfos        []classicQuestInfoPush
	questClears       []classicQuestClearPush
	questStates       []world.QuestStatePush
	friendInfos       []classicSocialFriendEntry
	friendClears      []classicSocialClearEntry
	blackInfos        []classicSocialBlackEntry
	blackClears       []classicSocialClearEntry
	enemyInfos        []classicSocialEnemyEntry
	enemyClears       []classicSocialClearEntry
	guildInfo         *guild.Guild
	guildMembers      []guild.Member
	guildAuth         *guild.Auth
	guildNotice       *classicGuildNoticePush
	guildResult       *classicGuildResultPush
	guildMemberClears []classicGuildMemberClearPush
	mallCategories    []mall.Category
	mallSearchCount   *mall.SearchCountPush
	mallSearchPage    *mall.SearchPagePush
	mallCurrency      *mall.CurrencyPush
	mallPurchase      *mall.PurchaseResult
	battleStart       *battle.StartPush
	battleCells       []battle.CellInfoPush
	battleCommand     *battle.StartCommandPush
	battleActions     []battle.ActionPush
	battleOver        *battle.OverPush
	handled           bool
}

type classicTownAddPointRequest struct {
	Stat string `json:"stat"`
}

type packetSession struct {
	selectedRole  *session.RoleSummary
	playerBase    *session.PlayerBaseData
	battleRuntime *battle.Runtime
	battleLoot    []session.RoleItem
	removedQuests map[string]bool
	friends       map[string]classicSocialFriendEntry
	blackList     map[string]classicSocialBlackEntry
	enemies       map[string]classicSocialEnemyEntry
}

func handlePacket(store *session.Store, packet protocol.Packet) packetResult {
	return handlePacketWithSession(store, packet, &packetSession{})
}

func handlePacketWithSession(store *session.Store, packet protocol.Packet, socketSession *packetSession) packetResult {
	switch packet.Cmd {
	case cmdAuthLoginRequest:
		var request session.LoginRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return packetResult{
			responseCmd:     cmdAuthLoginResponse,
			responsePayload: encodePayload(store.Login(request)),
			handled:         true,
		}
	case cmdRoleListRequest:
		var request session.RoleListRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return packetResult{
			responseCmd:     cmdRoleListResponse,
			responsePayload: encodePayload(store.ListRoles(request)),
			handled:         true,
		}
	case cmdRoleCreateRequest:
		var request session.RoleCreateRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return packetResult{
			responseCmd:     cmdRoleCreateResponse,
			responsePayload: encodePayload(store.CreateRole(request)),
			handled:         true,
		}
	case cmdRoleSelectRequest:
		var request session.RoleSelectRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		response := store.SelectRole(request)
		result := packetResult{
			responseCmd:     cmdRoleSelectResponse,
			responsePayload: encodePayload(response),
			handled:         true,
		}
		if response.Success {
			socketSession.selectedRole = &response.Role
			socketSession.playerBase = &response.PlayerBase
			bootstrap := world.BuildTownBootstrap(response.Role, response.PlayerBase)
			result.townBootstrap = &bootstrap
		}
		return result
	case cmdRoleRemoveRequest:
		var request session.RoleRemoveRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return packetResult{
			responseCmd:     cmdRoleRemoveResponse,
			responsePayload: encodePayload(store.RemoveRole(request)),
			handled:         true,
		}
	case cmdClassicTownTargetRoleReq:
		var request classicTownRoleInteractionRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		log.Printf("[ai-server] classic town targetRole handle=%s roleId=%s kind=%s mapId=%s", request.Handle, request.RoleID, request.Kind, request.MapID)
		return packetResult{handled: true}
	case cmdClassicTownActiveRoleReq:
		var request classicTownRoleInteractionRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		log.Printf("[ai-server] classic town activeRole handle=%s roleId=%s kind=%s mapId=%s", request.Handle, request.RoleID, request.Kind, request.MapID)
		if result, ok := buildClassicTownCollectionResult(store, socketSession, request); ok {
			return result
		}
		answerSpeak := world.BuildAnswerSpeak(request.Handle)
		return packetResult{
			answerSpeak: &answerSpeak,
			handled:     true,
		}
	case cmdClassicTownCrossRoleReq:
		var request classicTownRoleInteractionRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		log.Printf("[ai-server] classic town CrossRole handle=%s roleId=%s kind=%s mapId=%s", request.Handle, request.RoleID, request.Kind, request.MapID)
		if destination, ok := world.ResolveTownTransportAnswer(request.Handle, "goto"); ok {
			return buildClassicTownTransferResult(store, socketSession, strconv.Itoa(destination.MapID), destination.Spawn)
		}
		return packetResult{handled: true}
	case cmdClassicTownAnswerReq:
		var request classicTownAnswerRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		log.Printf("[ai-server] classic town Answer handle=%s msgHandle=%s answerHandle=%s", request.Handle, request.MsgHandle, request.AnswerHandle)
		if result, ok := buildClassicTownSourceQuestRewardResult(store, socketSession, request); ok {
			return result
		}
		if destination, ok := world.ResolveTownTransportAnswer(request.Handle, request.AnswerHandle); ok {
			return buildClassicTownTransferResult(store, socketSession, strconv.Itoa(destination.MapID), destination.Spawn)
		}
		if request.Handle == sourceSkillTeacherHandle && request.MsgHandle == "10" {
			if result, ok := buildClassicTownSkillShopResult(store, socketSession, request.AnswerHandle); ok {
				return result
			}
		}
		answerSpeak := world.BuildAnswerReply(request.Handle, request.MsgHandle, request.AnswerHandle)
		return packetResult{
			answerSpeak: answerSpeak,
			handled:     true,
		}
	case cmdClassicTownTransferReq:
		var request classicTownTransferRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		mapID, err := strconv.Atoi(request.MapID)
		if err != nil || !world.SupportsTownTransferMap(mapID) {
			log.Printf("[ai-server] classic town transfer ignored unsupported mapId=%s x=%d y=%d", request.MapID, request.X, request.Y)
			return packetResult{handled: true}
		}
		return buildClassicTownTransferResult(store, socketSession, request.MapID, world.SpawnPoint{X: request.X, Y: request.Y})
	case cmdClassicTownGetSkillListReq:
		return buildClassicTownSkillListResult(store, socketSession)
	case cmdClassicTownGetFastPanelReq:
		return buildClassicTownFastPanelResult(socketSession)
	case cmdClassicTownBuySkillReq:
		var request classicTownBuySkillRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicTownBuySkillResult(store, socketSession, request)
	case cmdClassicTownGetCapacityReq:
		var request classicTownContainerRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicTownContainerCapacityResult(store, socketSession, request.Type)
	case cmdClassicTownGetItemListReq:
		var request classicTownContainerRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicTownItemListResult(store, socketSession, request.Type)
	case cmdClassicTownContainerMove:
		var request classicTownContainerMoveRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicTownContainerMoveResult(store, socketSession, request)
	case cmdClassicTownEquipItemReq:
		var request classicTownEquipItemRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicTownEquipItemResult(store, socketSession, request)
	case cmdClassicTownAddPointReq:
		var request classicTownAddPointRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicTownAddPointResult(store, socketSession, request)
	case cmdClassicTownGetQuestLogReq:
		return buildClassicQuestLogResult(socketSession)
	case cmdClassicTownRemoveQuestReq:
		var request classicQuestRemoveRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicQuestRemoveResult(socketSession, request)
	case cmdClassicSocialAddFriendReq:
		var request classicSocialMutateRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicSocialAddFriendResult(socketSession, request)
	case cmdClassicSocialRemoveFriend:
		var request classicSocialMutateRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicSocialRemoveFriendResult(socketSession, request)
	case cmdClassicSocialAddBlackReq:
		var request classicSocialMutateRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicSocialAddBlackResult(socketSession, request)
	case cmdClassicSocialRemoveBlack:
		var request classicSocialMutateRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicSocialRemoveBlackResult(socketSession, request)
	case cmdClassicGuildInfoReq:
		return buildClassicGuildInfoResult(store, socketSession)
	case cmdClassicGuildCreateReq:
		var request guild.CreateRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicGuildCreateResult(store, socketSession, request)
	case cmdClassicGuildLeaveReq:
		return buildClassicGuildLeaveResult(store, socketSession)
	case cmdClassicGuildKickReq:
		var request guild.KickRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicGuildKickResult(store, socketSession, request)
	case cmdClassicGuildDismissReq:
		return buildClassicGuildDismissResult(store, socketSession)
	case cmdClassicGuildNoticeUpdateReq:
		var request guild.NoticeUpdateRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicGuildNoticeUpdateResult(store, socketSession, request)
	case cmdClassicMallCategoryListReq:
		return buildClassicMallCategoryListResult(store, socketSession)
	case cmdClassicMallSearchCountReq:
		var request mall.SearchRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicMallSearchCountResult(store, socketSession, request)
	case cmdClassicMallSearchPageReq:
		var request mall.SearchRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicMallSearchPageResult(store, socketSession, request)
	case cmdClassicMallPurchaseReq:
		var request mall.PurchaseRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicMallPurchaseResult(store, socketSession, request)
	case cmdClassicBattleStartReq:
		var request battle.StartRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicBattleStartResult(socketSession, request)
	case cmdClassicBattleActionReq:
		var request battle.ActionRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicBattleActionResult(store, socketSession, request)
	case cmdClassicBattleActiveItemReq:
		var request battle.ItemActionRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicBattleItemActionResult(store, socketSession, request)
	case cmdClassicBattlePlayOverReq:
		var request battle.PlayOverRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicBattlePlayOverResult(store, socketSession, request)
	default:
		return packetResult{}
	}
}

func buildClassicBattleStartResult(socketSession *packetSession, request battle.StartRequest) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic battle StartBattle ignored without selected role mapId=%s", request.MapID)
		return packetResult{handled: true}
	}

	runtime, bundle, ok := battle.NewWildBattle(*socketSession.selectedRole, *socketSession.playerBase, request)
	if !ok {
		log.Printf("[ai-server] classic battle StartBattle ignored unsupported mapId=%s", request.MapID)
		return packetResult{handled: true}
	}
	socketSession.battleRuntime = runtime
	log.Printf("[ai-server] classic battle StartBattle battleId=%s roleId=%s mapId=%s", bundle.Start.BattleID, socketSession.selectedRole.RoleID, request.MapID)
	return packetResult{
		battleStart:   &bundle.Start,
		battleCells:   bundle.Cells,
		battleCommand: &bundle.StartCommand,
		handled:       true,
	}
}

func buildClassicBattleActionResult(store *session.Store, socketSession *packetSession, request battle.ActionRequest) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic battle battleAction ignored without selected role battleId=%s", request.BattleID)
		return packetResult{handled: true}
	}
	if socketSession.battleRuntime == nil {
		log.Printf("[ai-server] classic battle battleAction ignored missing battle battleId=%s", request.BattleID)
		return packetResult{handled: true}
	}

	result := socketSession.battleRuntime.ProcessAction(request)
	if result.ErrorCode != "" {
		log.Printf("[ai-server] classic battle battleAction rejected battleId=%s actor=%s target=%s error=%s", request.BattleID, request.ActorHandle, request.TargetHandle, result.ErrorCode)
		return packetResult{handled: true}
	}
	var roleState *session.RoleState
	var rolePhysique *session.RolePhysique
	if result.Over != nil {
		roleState, rolePhysique = finalizeClassicBattleOver(store, socketSession, result.Over.Result)
		socketSession.battleLoot = buildClassicBattleLoot(socketSession, result.Over.Result)
		socketSession.battleRuntime = nil
	}
	return packetResult{
		battleActions: result.Actions,
		battleCommand: result.StartCommand,
		battleOver:    result.Over,
		roleState:     roleState,
		rolePhysique:  rolePhysique,
		handled:       true,
	}
}

func buildClassicBattleItemActionResult(store *session.Store, socketSession *packetSession, request battle.ItemActionRequest) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic battle ActiveItem ignored without selected role battleId=%s type=%s index=%d", request.BattleID, request.Type, request.Index)
		return packetResult{handled: true}
	}
	if socketSession.battleRuntime == nil {
		log.Printf("[ai-server] classic battle ActiveItem ignored missing battle battleId=%s type=%s index=%d", request.BattleID, request.Type, request.Index)
		return packetResult{handled: true}
	}

	item, ok := store.GetRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, request.Type, request.Index)
	if !ok {
		log.Printf("[ai-server] classic battle ActiveItem rejected missing item roleId=%s type=%s index=%d", socketSession.selectedRole.RoleID, request.Type, request.Index)
		return packetResult{handled: true}
	}

	result := socketSession.battleRuntime.ProcessItemAction(request, classicBattleItemActionFromRoleItem(item))
	if result.ErrorCode != "" {
		log.Printf("[ai-server] classic battle ActiveItem rejected battleId=%s actor=%s type=%s index=%d error=%s", request.BattleID, request.ActorHandle, request.Type, request.Index, result.ErrorCode)
		return packetResult{handled: true}
	}

	useResult := store.ConsumeRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, request.Type, request.Index, 1)
	if !useResult.Found || !useResult.Used {
		log.Printf("[ai-server] classic battle ActiveItem consume failed roleId=%s type=%s index=%d error=%s", socketSession.selectedRole.RoleID, request.Type, request.Index, useResult.ErrorCode)
		return packetResult{handled: true}
	}

	socketSession.selectedRole = &useResult.Role
	socketSession.playerBase = &useResult.PlayerBase
	var roleState *session.RoleState
	var rolePhysique *session.RolePhysique
	if result.Over != nil {
		roleState, rolePhysique = finalizeClassicBattleOver(store, socketSession, result.Over.Result)
		socketSession.battleRuntime = nil
	}

	packet := packetResult{
		battleActions: result.Actions,
		battleCommand: result.StartCommand,
		battleOver:    result.Over,
		roleState:     roleState,
		rolePhysique:  rolePhysique,
		itemInfos:     make([]classicTownItemInfoPush, 0, 1),
		itemClears:    make([]classicTownItemInfoClearPush, 0, len(useResult.ClearedItems)),
		handled:       true,
	}
	if useResult.UpdatedItem != nil {
		updatedItem := *useResult.UpdatedItem
		updatedItem.Handle = useResult.Role.RoleID
		packet.itemInfos = append(packet.itemInfos, classicTownItemInfoPushFromRoleItem(updatedItem))
	}
	for _, clear := range useResult.ClearedItems {
		packet.itemClears = append(packet.itemClears, classicTownItemInfoClearPush{
			Handle: useResult.Role.RoleID,
			Type:   clear.Type,
			Index:  clear.Index,
		})
	}
	return packet
}

func buildClassicBattlePlayOverResult(store *session.Store, socketSession *packetSession, request battle.PlayOverRequest) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic battle BattlePlayOver ignored without selected role battleId=%s", request.BattleID)
		return packetResult{handled: true}
	}
	if socketSession.battleRuntime == nil {
		log.Printf("[ai-server] classic battle BattlePlayOver ignored missing battle battleId=%s", request.BattleID)
		return packetResult{handled: true}
	}

	result := socketSession.battleRuntime.ProcessPlayOver(request)
	if result.ErrorCode != "" {
		log.Printf("[ai-server] classic battle BattlePlayOver rejected battleId=%s error=%s", request.BattleID, result.ErrorCode)
		return packetResult{handled: true}
	}

	var roleState *session.RoleState
	var rolePhysique *session.RolePhysique
	if result.Over != nil {
		roleState, rolePhysique = finalizeClassicBattleOver(store, socketSession, result.Over.Result)
		socketSession.battleLoot = buildClassicBattleLoot(socketSession, result.Over.Result)
		socketSession.battleRuntime = nil
	}
	return packetResult{
		battleCommand: result.StartCommand,
		battleOver:    result.Over,
		roleState:     roleState,
		rolePhysique:  rolePhysique,
		handled:       true,
	}
}

func buildClassicBattleLoot(socketSession *packetSession, result battle.ResultPayload) []session.RoleItem {
	if socketSession == nil || result.Winner != battle.CampTeam || result.Escaped || len(result.Items) == 0 {
		return nil
	}

	capacity := len(result.Items)
	if capacity > classicBattleLootCap {
		capacity = classicBattleLootCap
	}
	items := make([]session.RoleItem, 0, capacity)
	for index, name := range result.Items {
		if index >= classicBattleLootCap {
			break
		}
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		item := session.RoleItem{
			Name:      name,
			ItemType:  "own",
			Count:     1,
			ItemLevel: 1,
		}
		if template, ok := session.CapturedRoleItemTemplate(name); ok {
			item = template
		}
		item.Type = classicBattleLootType
		item.Name = name
		item.Count = 1
		item.Index = index
		item.Handle = socketSession.selectedRole.RoleID
		item.Owner = socketSession.selectedRole.DisplayName
		items = append(items, item)
	}
	return items
}

func finalizeClassicBattleOver(store *session.Store, socketSession *packetSession, result battle.ResultPayload) (*session.RoleState, *session.RolePhysique) {
	roleState := updatePlayerBaseRoleStateFromBattle(socketSession)
	if store == nil || socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		return roleState, nil
	}
	if result.Winner != battle.CampTeam || result.Escaped || result.ExpDelta <= 0 {
		return roleState, nil
	}

	expResult := store.GrantRoleExperience(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, result.ExpDelta)
	if !expResult.Found || !expResult.Granted {
		return roleState, nil
	}

	socketSession.selectedRole = &expResult.Role
	socketSession.playerBase = &expResult.PlayerBase
	if roleState != nil {
		expResult.RoleState.HP = roleState.HP
		expResult.RoleState.MP = roleState.MP
	}
	socketSession.playerBase.HP = expResult.RoleState.HP
	socketSession.playerBase.MP = expResult.RoleState.MP
	socketSession.playerBase.MaxHP = expResult.PlayerBase.MaxHP
	socketSession.playerBase.MaxMP = expResult.PlayerBase.MaxMP
	socketSession.playerBase.Level = expResult.Role.Level
	socketSession.playerBase.Exp = expResult.Role.Exp
	if socketSession.playerBase.RoleState != nil {
		socketSession.playerBase.RoleState.HP = expResult.RoleState.HP
		socketSession.playerBase.RoleState.MP = expResult.RoleState.MP
		socketSession.playerBase.RoleState.Exp = expResult.RoleState.Exp
		socketSession.playerBase.RoleState.Lv = expResult.RoleState.Lv
		socketSession.playerBase.RoleState.Speed = expResult.RoleState.Speed
	}
	return &expResult.RoleState, socketSession.playerBase.RolePhysique
}

func classicBattleItemActionFromRoleItem(item session.RoleItem) battle.ItemAction {
	return battle.ItemAction{
		SourceType:  item.Type,
		SourceIndex: item.Index,
		Name:        item.Name,
		ItemType:    item.ItemType,
		Display:     item.Display,
		Description: item.Description,
		HealHP:      parseClassicItemDescriptionInt(item.Description, "7"),
		HealMP:      parseClassicItemDescriptionInt(item.Description, "8"),
	}
}

func parseClassicItemDescriptionInt(description string, key string) int {
	marker := "&" + strings.TrimSpace(key) + "@"
	start := strings.Index(description, marker)
	if start < 0 {
		return 0
	}
	start += len(marker)
	end := start
	for end < len(description) {
		ch := description[end]
		if ch < '0' || ch > '9' {
			break
		}
		end += 1
	}
	if end == start {
		return 0
	}
	value, err := strconv.Atoi(description[start:end])
	if err != nil {
		return 0
	}
	return value
}

func updatePlayerBaseRoleStateFromBattle(socketSession *packetSession) *session.RoleState {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil || socketSession.battleRuntime == nil {
		return nil
	}

	for _, cell := range socketSession.battleRuntime.Cells {
		if cell.Handle != socketSession.selectedRole.RoleID {
			continue
		}

		socketSession.playerBase.HP = cell.HP
		socketSession.playerBase.MP = cell.MP
		if socketSession.playerBase.RoleState == nil {
			socketSession.playerBase.RoleState = &session.RoleState{
				Handle: socketSession.selectedRole.RoleID,
				Exp:    socketSession.playerBase.Exp,
				Lv:     socketSession.playerBase.Level,
				Speed:  130,
			}
		}
		socketSession.playerBase.RoleState.Handle = socketSession.selectedRole.RoleID
		socketSession.playerBase.RoleState.HP = cell.HP
		socketSession.playerBase.RoleState.MP = cell.MP
		socketSession.playerBase.RoleState.Exp = socketSession.playerBase.Exp
		socketSession.playerBase.RoleState.Lv = socketSession.playerBase.Level
		if socketSession.playerBase.RoleState.Speed == 0 {
			socketSession.playerBase.RoleState.Speed = 130
		}
		return socketSession.playerBase.RoleState
	}

	return nil
}

func buildClassicTownContainerCapacityResult(store *session.Store, socketSession *packetSession, containerType string) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic town GetContainerCapacity ignored without selected role type=%s", containerType)
		return packetResult{handled: true}
	}

	containerType = strings.TrimSpace(containerType)
	if containerType == classicBattleLootType {
		return packetResult{
			containerCap: &classicTownContainerCapacityPush{
				Handle:   socketSession.selectedRole.RoleID,
				Type:     classicBattleLootType,
				Capacity: classicBattleLootCap,
				OpenType: "",
			},
			handled: true,
		}
	}

	capacity, ok := store.GetRoleContainerCapacity(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, containerType)
	if !ok {
		log.Printf("[ai-server] classic town GetContainerCapacity ignored unsupported roleId=%s type=%s", socketSession.selectedRole.RoleID, containerType)
		return packetResult{handled: true}
	}

	return packetResult{
		containerCap: &classicTownContainerCapacityPush{
			Handle:   socketSession.selectedRole.RoleID,
			Type:     containerType,
			Capacity: capacity,
			OpenType: "",
		},
		handled: true,
	}
}

func buildClassicTownItemListResult(store *session.Store, socketSession *packetSession, containerType string) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic town GetItemList ignored without selected role type=%s", containerType)
		return packetResult{handled: true}
	}

	containerType = strings.TrimSpace(containerType)
	if containerType == classicBattleLootType {
		result := packetResult{
			containerCap: &classicTownContainerCapacityPush{
				Handle:   socketSession.selectedRole.RoleID,
				Type:     classicBattleLootType,
				Capacity: classicBattleLootCap,
				OpenType: "",
			},
			itemInfos: make([]classicTownItemInfoPush, 0, len(socketSession.battleLoot)),
			handled:   true,
		}
		for _, item := range socketSession.battleLoot {
			if item.Type != classicBattleLootType {
				continue
			}
			item.Handle = socketSession.selectedRole.RoleID
			result.itemInfos = append(result.itemInfos, classicTownItemInfoPushFromRoleItem(item))
		}
		return result
	}

	items, capacity, ok := store.GetRoleItems(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, containerType)
	if !ok {
		log.Printf("[ai-server] classic town GetItemList ignored unsupported roleId=%s type=%s", socketSession.selectedRole.RoleID, containerType)
		return packetResult{handled: true}
	}

	result := packetResult{
		containerCap: &classicTownContainerCapacityPush{
			Handle:   socketSession.selectedRole.RoleID,
			Type:     containerType,
			Capacity: capacity,
			OpenType: "",
		},
		itemInfos: make([]classicTownItemInfoPush, 0, len(items)),
		handled:   true,
	}
	for _, item := range items {
		if item.Type != containerType {
			continue
		}
		item.Handle = socketSession.selectedRole.RoleID
		result.itemInfos = append(result.itemInfos, classicTownItemInfoPushFromRoleItem(item))
	}
	return result
}

func buildClassicTownContainerMoveResult(store *session.Store, socketSession *packetSession, request classicTownContainerMoveRequest) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic town ContainerMove ignored without selected role source=%s target=%s", request.SourceType, request.TargetType)
		return packetResult{handled: true}
	}

	sourceType := strings.TrimSpace(request.SourceType)
	targetType := strings.TrimSpace(request.TargetType)
	if sourceType == classicBattleLootType && targetType == "背包" {
		nameFilter := map[string]bool{}
		for _, name := range request.Names {
			name = strings.TrimSpace(name)
			if name != "" {
				nameFilter[name] = true
			}
		}

		remaining := make([]session.RoleItem, 0, len(socketSession.battleLoot))
		itemInfos := []classicTownItemInfoPush{}
		itemClears := []classicTownItemInfoClearPush{}
		for _, item := range socketSession.battleLoot {
			if item.Type != classicBattleLootType || (len(nameFilter) > 0 && !nameFilter[item.Name]) {
				remaining = append(remaining, item)
				continue
			}

			moved := item
			moved.Type = "背包"
			moved.Index = -1
			granted, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, moved)
			if !ok {
				remaining = append(remaining, item)
				continue
			}

			granted.Handle = socketSession.selectedRole.RoleID
			itemInfos = append(itemInfos, classicTownItemInfoPushFromRoleItem(granted))
			itemClears = append(itemClears, classicTownItemInfoClearPush{
				Handle: socketSession.selectedRole.RoleID,
				Type:   classicBattleLootType,
				Index:  item.Index,
			})
		}
		socketSession.battleLoot = remaining

		return packetResult{
			containerCap: &classicTownContainerCapacityPush{
				Handle:   socketSession.selectedRole.RoleID,
				Type:     classicBattleLootType,
				Capacity: classicBattleLootCap,
				OpenType: "",
			},
			itemInfos:  itemInfos,
			itemClears: itemClears,
			handled:    true,
		}
	}
	if sourceType == "背包" && targetType == "背包" && request.SourceIndex != nil && request.TargetIndex != nil {
		moveCount := 0
		if request.Count != nil {
			moveCount = *request.Count
		}
		moveResult := store.MoveRoleItem(
			socketSession.playerBase.PlayerID,
			socketSession.selectedRole.RoleID,
			sourceType,
			*request.SourceIndex,
			targetType,
			*request.TargetIndex,
			moveCount,
		)
		if !moveResult.Found || !moveResult.Moved {
			log.Printf("[ai-server] classic town ContainerMove bag noop roleId=%s source=%s[%v] target=%s[%v] error=%s", socketSession.selectedRole.RoleID, sourceType, request.SourceIndex, targetType, request.TargetIndex, moveResult.ErrorCode)
			return packetResult{handled: true}
		}
		result := packetResult{
			itemInfos:  make([]classicTownItemInfoPush, 0, len(moveResult.UpdatedItems)),
			itemClears: make([]classicTownItemInfoClearPush, 0, len(moveResult.ClearedItems)),
			handled:    true,
		}
		for _, clear := range moveResult.ClearedItems {
			result.itemClears = append(result.itemClears, classicTownItemInfoClearPush{
				Handle: socketSession.selectedRole.RoleID,
				Type:   clear.Type,
				Index:  clear.Index,
			})
		}
		for _, item := range moveResult.UpdatedItems {
			item.Handle = socketSession.selectedRole.RoleID
			result.itemInfos = append(result.itemInfos, classicTownItemInfoPushFromRoleItem(item))
		}
		return result
	}

	log.Printf("[ai-server] classic town ContainerMove ignored unsupported source=%s target=%s", sourceType, targetType)
	return packetResult{handled: true}
}

func buildClassicTownSourceQuestRewardResult(store *session.Store, socketSession *packetSession, request classicTownAnswerRequest) (packetResult, bool) {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		return packetResult{}, false
	}
	if strings.TrimSpace(request.Handle) != "4000542609162635" ||
		strings.TrimSpace(request.MsgHandle) != "3q3d_1" ||
		strings.TrimSpace(request.AnswerHandle) != "3q3a_1_1" {
		return packetResult{}, false
	}

	rewardItems := sourceYeMeiGiftRewardItems()
	itemInfos := make([]classicTownItemInfoPush, 0, len(rewardItems))
	for _, item := range rewardItems {
		granted, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, item)
		if !ok {
			continue
		}
		granted.Handle = socketSession.selectedRole.RoleID
		itemInfos = append(itemInfos, classicTownItemInfoPushFromRoleItem(granted))
	}

	if selectedRole, playerBase, ok := store.GetRoleRuntimeData(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID); ok {
		socketSession.selectedRole = &selectedRole
		socketSession.playerBase = &playerBase
	}

	return packetResult{
		answerSpeak: &world.AnswerSpeakPush{
			Handle:    request.Handle,
			MsgHandle: "3q3a_1_1",
			Msg:       "谢谢您啦。",
			Answers: []world.AnswerOption{
				{Handle: "3q3a_1_2", Msg: "<c/>关闭"},
			},
		},
		itemInfos: itemInfos,
		handled:   true,
	}, true
}

func sourceYeMeiGiftRewardItems() []session.RoleItem {
	names := []string{"蓝布衣", "蓝布裤", "布鞋", "L花卷"}
	counts := map[string]int{"L花卷": 5}
	items := make([]session.RoleItem, 0, len(names))
	for _, name := range names {
		item := session.RoleItem{
			Type:      "背包",
			Name:      name,
			Count:     1,
			Index:     -1,
			ItemLevel: 1,
		}
		if template, ok := session.CapturedRoleItemTemplate(name); ok {
			item = template
		}
		item.Type = "背包"
		item.Index = -1
		if count, ok := counts[name]; ok {
			item.Count = count
		} else {
			item.Count = 1
		}
		items = append(items, item)
	}
	return items
}

func buildClassicTownEquipItemResult(store *session.Store, socketSession *packetSession, request classicTownEquipItemRequest) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic town EquipItem ignored without selected role type=%s index=%d", request.Type, request.Index)
		return packetResult{handled: true}
	}

	equipResult := store.EquipRoleItem(
		socketSession.playerBase.PlayerID,
		socketSession.selectedRole.RoleID,
		request.Type,
		request.Index,
		request.Count,
	)
	if !equipResult.Found {
		log.Printf("[ai-server] classic town EquipItem ignored missing role roleId=%s type=%s index=%d", socketSession.selectedRole.RoleID, request.Type, request.Index)
		return packetResult{handled: true}
	}
	if !equipResult.Equipped {
		log.Printf("[ai-server] classic town EquipItem rejected roleId=%s type=%s index=%d error=%s", socketSession.selectedRole.RoleID, request.Type, request.Index, equipResult.ErrorCode)
		return packetResult{handled: true}
	}

	socketSession.selectedRole = &equipResult.Role
	socketSession.playerBase = &equipResult.PlayerBase
	equippedItem := equipResult.EquippedItem
	equippedItem.Handle = equipResult.Role.RoleID
	result := packetResult{
		createPlayer: buildClassicTownCreatePlayerPush(equipResult.Role, equipResult.PlayerBase),
		itemInfos: []classicTownItemInfoPush{
			classicTownItemInfoPushFromRoleItem(equippedItem),
		},
		itemClears: make([]classicTownItemInfoClearPush, 0, len(equipResult.ClearedItems)),
		handled:    true,
	}
	for _, clear := range equipResult.ClearedItems {
		result.itemClears = append(result.itemClears, classicTownItemInfoClearPush{
			Handle: equipResult.Role.RoleID,
			Type:   clear.Type,
			Index:  clear.Index,
		})
	}
	result.rolePhysique = equipResult.PlayerBase.RolePhysique
	log.Printf("[ai-server] classic town EquipItem roleId=%s sourceType=%s sourceIndex=%d equipIndex=%d item=%s", equipResult.Role.RoleID, request.Type, request.Index, equippedItem.Index, equippedItem.Name)
	return result
}

func buildClassicTownCreatePlayerPush(role session.RoleSummary, playerBase session.PlayerBaseData) *world.RolePush {
	bootstrap := world.BuildTownBootstrap(role, playerBase)
	return &bootstrap.CreatePlayer
}

func buildClassicTownAddPointResult(
	store *session.Store,
	socketSession *packetSession,
	request classicTownAddPointRequest,
) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic town AddPoint ignored without selected role stat=%s", request.Stat)
		return packetResult{handled: true}
	}

	result := store.AddRolePoint(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, request.Stat)
	if !result.Found || !result.Applied {
		log.Printf("[ai-server] classic town AddPoint rejected roleId=%s stat=%s error=%s", socketSession.selectedRole.RoleID, request.Stat, result.ErrorCode)
		return packetResult{handled: true}
	}

	socketSession.selectedRole = &result.Role
	socketSession.playerBase = &result.PlayerBase
	log.Printf("[ai-server] classic town AddPoint roleId=%s stat=%s", result.Role.RoleID, strings.ToUpper(strings.TrimSpace(request.Stat)))
	return packetResult{
		rolePhysique: &result.RolePhysique,
		handled:      true,
	}
}

func buildClassicTownSkillListResult(store *session.Store, socketSession *packetSession) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic town GetSkillList ignored without selected role")
		return packetResult{handled: true}
	}

	skills, cap, ok := store.GetRoleSkills(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID)
	if !ok {
		log.Printf("[ai-server] classic town GetSkillList ignored missing role roleId=%s", socketSession.selectedRole.RoleID)
		return packetResult{handled: true}
	}

	return packetResult{
		skillCap: &classicTownSkillCapPush{Count: cap},
		skillInfos: classicTownSkillInfoPushes(
			socketSession.selectedRole.RoleID,
			skills,
		),
		currencyPush: buildClassicTownCurrencyPush(
			socketSession.selectedRole.RoleID,
			roleCurrenciesOrEmpty(store, socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID),
		),
		handled: true,
	}
}

func buildClassicTownBuySkillResult(
	store *session.Store,
	socketSession *packetSession,
	request classicTownBuySkillRequest,
) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic town BuySkill ignored without selected role shopId=%s skillId=%d", request.ShopID, request.SkillID)
		return packetResult{handled: true}
	}

	entry, ok := findSourceSkillShopEntry(request.ShopID, request.SkillID)
	if !ok {
		return packetResult{
			buySkillResult: &classicTownBuySkillResultPush{
				Success:      false,
				ShopID:       request.ShopID,
				SkillID:      request.SkillID,
				ErrorCode:    "skill_missing",
				ErrorMessage: "技能不存在。",
			},
			handled: true,
		}
	}

	purchase := store.PurchaseRoleSkill(
		socketSession.playerBase.PlayerID,
		socketSession.selectedRole.RoleID,
		sourceSkillEntryToRoleSkill(entry),
		sourceSkillRequirementsToCurrencies(entry.Requirements),
	)
	if !purchase.Found {
		log.Printf("[ai-server] classic town BuySkill ignored missing role roleId=%s shopId=%s skillId=%d", socketSession.selectedRole.RoleID, request.ShopID, request.SkillID)
		return packetResult{handled: true}
	}

	socketSession.selectedRole.Currencies = purchase.Currencies
	socketSession.selectedRole.Skills = purchase.Skills
	socketSession.playerBase.Currencies = purchase.Currencies
	result := packetResult{
		skillCap: &classicTownSkillCapPush{Count: purchase.SkillCap},
		currencyPush: buildClassicTownCurrencyPush(
			socketSession.selectedRole.RoleID,
			purchase.Currencies,
		),
		buySkillResult: &classicTownBuySkillResultPush{
			Success:      purchase.Learned,
			ShopID:       request.ShopID,
			SkillID:      request.SkillID,
			Currencies:   purchase.Currencies,
			ErrorCode:    purchase.ErrorCode,
			ErrorMessage: purchase.ErrorMessage,
		},
		handled: true,
	}
	if purchase.Learned {
		result.skillInfos = []classicTownSkillInfoPush{
			classicTownSkillInfoPushFromRoleSkill(
				socketSession.selectedRole.RoleID,
				sourceSkillEntryToRoleSkill(entry),
			),
		}
		log.Printf("[ai-server] classic town BuySkill learned roleId=%s shopId=%s skillId=%d skill=%s", socketSession.selectedRole.RoleID, request.ShopID, request.SkillID, entry.Name)
	} else {
		log.Printf("[ai-server] classic town BuySkill rejected roleId=%s shopId=%s skillId=%d error=%s", socketSession.selectedRole.RoleID, request.ShopID, request.SkillID, purchase.ErrorCode)
	}
	return result
}

func buildClassicTownTransferResult(
	store *session.Store,
	socketSession *packetSession,
	mapIDText string,
	spawn world.SpawnPoint,
) packetResult {
	mapID, err := strconv.Atoi(mapIDText)
	if err != nil || !world.SupportsTownTransferMap(mapID) {
		log.Printf("[ai-server] classic town transfer ignored unsupported mapId=%s x=%d y=%d", mapIDText, spawn.X, spawn.Y)
		return packetResult{handled: true}
	}
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic town transfer ignored without selected role mapId=%s x=%d y=%d", mapIDText, spawn.X, spawn.Y)
		return packetResult{handled: true}
	}
	role, playerBase, ok := store.UpdateRoleMap(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, mapID)
	if !ok {
		log.Printf("[ai-server] classic town transfer ignored missing role roleId=%s mapId=%s", socketSession.selectedRole.RoleID, mapIDText)
		return packetResult{handled: true}
	}
	socketSession.selectedRole = &role
	socketSession.playerBase = &playerBase
	bootstrap, ok := world.BuildTownTransferBootstrap(role, playerBase, mapID, spawn)
	if !ok {
		return packetResult{handled: true}
	}
	log.Printf("[ai-server] classic town transfer mapId=%s x=%d y=%d roleId=%s", mapIDText, spawn.X, spawn.Y, role.RoleID)
	return packetResult{
		townBootstrap: &bootstrap,
		handled:       true,
	}
}
