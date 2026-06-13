package main

import (
	"log"
	"strconv"
	"strings"
	"time"

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
	chatMessages      []classicTownChatMessagePush
	skillCap          *classicTownSkillCapPush
	skillInfos        []classicTownSkillInfoPush
	skillClears       []classicTownClearSkillInfoPush
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
	dungeonInstance   *classicTownDungeonInstancePush
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
	battleBuffs       []battle.BuffInfoPush
	battleOver        *battle.OverPush
	removeRoleHandles []string
	handled           bool
}

type classicTownDungeonInstancePush struct {
	Key                           string   `json:"key"`
	DisplayName                   string   `json:"displayName"`
	MapID                         string   `json:"mapId"`
	Active                        bool     `json:"active"`
	CreatedAtUnix                 int64    `json:"createdAtUnix"`
	ExpiresAtUnix                 int64    `json:"expiresAtUnix"`
	DurationSeconds               int64    `json:"durationSeconds"`
	RemainingSeconds              int64    `json:"remainingSeconds"`
	DefeatedVisibleMonsterHandles []string `json:"defeatedVisibleMonsterHandles,omitempty"`
}

type classicTownAddPointRequest struct {
	Stat string `json:"stat"`
}

type dungeonEntryConsumePolicy string

const (
	dungeonEntryConsumeNone          dungeonEntryConsumePolicy = "none"
	dungeonEntryConsumeOnNewInstance dungeonEntryConsumePolicy = "on_new_instance"
	classicTownBagContainerType      string                    = "背包"
)

type dungeonEntryRule struct {
	InstanceKey   string
	TicketName    string
	TicketCount   int
	ConsumePolicy dungeonEntryConsumePolicy
}

type packetSession struct {
	selectedRole            *session.RoleSummary
	playerBase              *session.PlayerBaseData
	battleRuntime           *battle.Runtime
	battleLoot              []session.RoleItem
	defeatedVisibleMonsters map[string]bool
	removedQuests           map[string]bool
	friends                 map[string]classicSocialFriendEntry
	blackList               map[string]classicSocialBlackEntry
	enemies                 map[string]classicSocialEnemyEntry
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
			result.dungeonInstance = syncDungeonInstanceState(store, socketSession, response.Role.MapID)
			bootstrap := world.BuildTownBootstrap(response.Role, response.PlayerBase)
			filterDefeatedVisibleMonsters(&bootstrap, socketSession)
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
		if destination, ok := resolveClassicTownTransportAnswer(socketSession, request.MapID, request.Handle, "goto"); ok {
			return buildClassicTownTransferResult(store, socketSession, strconv.Itoa(destination.MapID), destination.Spawn)
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
		if destination, ok := resolveClassicTownTransportAnswer(socketSession, request.MapID, request.Handle, "goto"); ok {
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
		if destination, ok := resolveClassicTownTransportAnswer(socketSession, "", request.Handle, request.AnswerHandle); ok {
			return buildClassicTownTransferResult(store, socketSession, strconv.Itoa(destination.MapID), destination.Spawn)
		}
		if result, ok := buildClassicTownHealerResult(store, socketSession, request); ok {
			return result
		}
		if result, ok := buildClassicTownItemShopResult(request); ok {
			return result
		}
		if isClassicTownSkillTeacherRequest(request) {
			if result, ok := buildClassicTownSkillShopResult(store, socketSession, request.Handle, request.AnswerHandle); ok {
				return result
			}
		}
		if result, ok := buildClassicTownVocationResult(store, socketSession, request); ok {
			return result
		}
		if result, ok := buildClassicQuestAnswerResult(store, socketSession, request); ok {
			return result
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
		return buildClassicTownFastPanelResult(store, socketSession)
	case cmdClassicTownSetFastPanelReq:
		var request classicTownSetFastPanelRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicTownSetFastPanelResult(store, socketSession, request)
	case cmdClassicTownBuySkillReq:
		var request classicTownBuySkillRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicTownBuySkillResult(store, socketSession, request)
	case cmdClassicTownRemoveSkillReq:
		var request classicTownRemoveSkillRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicTownRemoveSkillResult(store, socketSession, request)
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
	case cmdClassicTownActiveItemReq:
		var request classicTownActiveItemRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicTownActiveItemResult(store, socketSession, request)
	case cmdClassicTownDestroyItemReq:
		var request classicTownDestroyItemRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicTownDestroyItemResult(store, socketSession, request)
	case cmdClassicTownSaleItemReq:
		var request classicTownSaleItemRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicTownSaleItemResult(store, socketSession, request)
	case cmdClassicTownChatSendReq:
		var request classicTownChatSendRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicTownChatSendResult(socketSession, request)
	case cmdClassicTownGetQuestLogReq:
		return buildClassicQuestLogResult(store, socketSession)
	case cmdClassicTownRemoveQuestReq:
		var request classicQuestRemoveRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicQuestRemoveResult(store, socketSession, request)
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
		return buildClassicBattleStartResult(store, socketSession, request)
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

func isClassicTownSkillTeacherRequest(request classicTownAnswerRequest) bool {
	return request.MsgHandle == "10" && (request.Handle == sourceSkillTeacherHandle || request.Handle == guangqingSkillTeacherHandle)
}

func buildClassicBattleStartResult(store *session.Store, socketSession *packetSession, request battle.StartRequest) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic battle StartBattle ignored without selected role mapId=%s", request.MapID)
		return packetResult{handled: true}
	}
	if mapID, ok := battle.ParseMapID(request.MapID); ok {
		_ = syncDungeonInstanceState(store, socketSession, mapID)
	}
	if isDefeatedVisibleMonster(socketSession, request.SourceMonsterHandle) {
		log.Printf("[ai-server] classic battle StartBattle ignored defeated visible monster mapId=%s handle=%s", request.MapID, request.SourceMonsterHandle)
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
	var removeRoleHandles []string
	if result.Over != nil {
		roleState, rolePhysique = finalizeClassicBattleOver(store, socketSession, result.Over.Result)
		socketSession.battleLoot = buildClassicBattleLoot(socketSession, result.Over.Result)
		removeRoleHandles = append(removeRoleHandles, markDefeatedVisibleMonsterFromBattle(store, socketSession, result.Over)...)
		socketSession.battleRuntime = nil
	}
	return packetResult{
		battleActions:     result.Actions,
		battleBuffs:       result.BuffInfos,
		battleCommand:     result.StartCommand,
		battleOver:        result.Over,
		roleState:         roleState,
		rolePhysique:      rolePhysique,
		removeRoleHandles: removeRoleHandles,
		handled:           true,
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
	var removeRoleHandles []string
	if result.Over != nil {
		roleState, rolePhysique = finalizeClassicBattleOver(store, socketSession, result.Over.Result)
		removeRoleHandles = append(removeRoleHandles, markDefeatedVisibleMonsterFromBattle(store, socketSession, result.Over)...)
		socketSession.battleRuntime = nil
	}

	packet := packetResult{
		battleActions:     result.Actions,
		battleCommand:     result.StartCommand,
		battleOver:        result.Over,
		roleState:         roleState,
		rolePhysique:      rolePhysique,
		removeRoleHandles: removeRoleHandles,
		itemInfos:         make([]classicTownItemInfoPush, 0, 1),
		itemClears:        make([]classicTownItemInfoClearPush, 0, len(useResult.ClearedItems)),
		handled:           true,
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
	var removeRoleHandles []string
	if result.Over != nil {
		roleState, rolePhysique = finalizeClassicBattleOver(store, socketSession, result.Over.Result)
		socketSession.battleLoot = buildClassicBattleLoot(socketSession, result.Over.Result)
		removeRoleHandles = append(removeRoleHandles, markDefeatedVisibleMonsterFromBattle(store, socketSession, result.Over)...)
		socketSession.battleRuntime = nil
	}
	return packetResult{
		battleCommand:     result.StartCommand,
		battleOver:        result.Over,
		roleState:         roleState,
		rolePhysique:      rolePhysique,
		removeRoleHandles: removeRoleHandles,
		handled:           true,
	}
}

func visibleMonsterRemoveHandles(runtime *battle.Runtime, over *battle.OverPush) []string {
	if runtime == nil || over == nil || over.Result.Winner != battle.CampTeam || over.Result.Escaped {
		return nil
	}
	if strings.TrimSpace(runtime.SourceMonsterHandle) == "" {
		return nil
	}
	handles := make([]string, 0, len(runtime.Cells))
	seen := map[string]bool{}
	for _, cell := range runtime.Cells {
		if cell.Camp != battle.CampEnemy {
			continue
		}
		handle := strings.TrimSpace(cell.Handle)
		if handle == "" || seen[handle] {
			continue
		}
		seen[handle] = true
		handles = append(handles, handle)
	}
	if len(handles) == 0 {
		return []string{strings.TrimSpace(runtime.SourceMonsterHandle)}
	}
	return handles
}

func markDefeatedVisibleMonsterFromBattle(store *session.Store, socketSession *packetSession, over *battle.OverPush) []string {
	if socketSession == nil || socketSession.battleRuntime == nil {
		return nil
	}
	handles := visibleMonsterRemoveHandles(socketSession.battleRuntime, over)
	if len(handles) == 0 {
		return nil
	}
	mapID, ok := battle.ParseMapID(socketSession.battleRuntime.MapID)
	instanceKey := ""
	if ok {
		instanceKey, ok = world.DungeonInstanceKeyForMapID(mapID)
	}
	if !ok {
		for _, handle := range handles {
			markDefeatedVisibleMonster(socketSession, handle)
		}
		return handles
	}
	for _, handle := range handles {
		state, ok := store.MarkRoleDungeonVisibleMonsterDefeated(
			socketSession.playerBase.PlayerID,
			socketSession.selectedRole.RoleID,
			instanceKey,
			handle,
		)
		if ok {
			setDefeatedVisibleMonsterHandles(socketSession, state.DefeatedVisibleMonsterHandles)
		} else {
			markDefeatedVisibleMonster(socketSession, handle)
		}
	}
	return handles
}

func markDefeatedVisibleMonster(socketSession *packetSession, handle string) {
	handle = strings.TrimSpace(handle)
	if socketSession == nil || handle == "" {
		return
	}
	if socketSession.defeatedVisibleMonsters == nil {
		socketSession.defeatedVisibleMonsters = map[string]bool{}
	}
	socketSession.defeatedVisibleMonsters[handle] = true
}

func isDefeatedVisibleMonster(socketSession *packetSession, handle string) bool {
	handle = strings.TrimSpace(handle)
	return socketSession != nil && handle != "" && socketSession.defeatedVisibleMonsters[handle]
}

func setDefeatedVisibleMonsterHandles(socketSession *packetSession, handles []string) {
	if socketSession == nil {
		return
	}
	socketSession.defeatedVisibleMonsters = map[string]bool{}
	for _, handle := range handles {
		handle = strings.TrimSpace(handle)
		if handle == "" {
			continue
		}
		socketSession.defeatedVisibleMonsters[handle] = true
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
	for _, rawName := range result.Items {
		if len(items) >= classicBattleLootCap {
			break
		}
		name, count := parseClassicBattleLootNameAndCount(rawName)
		if name == "" {
			continue
		}
		item := session.RoleItem{
			Name:      name,
			ItemType:  "own",
			Count:     count,
			ItemLevel: 1,
		}
		if template, ok := session.CapturedRoleItemTemplate(name); ok {
			item = template
		}
		item.Type = classicBattleLootType
		item.Name = name
		item.Count = count
		item.Index = len(items)
		item.Handle = socketSession.selectedRole.RoleID
		item.Owner = socketSession.selectedRole.DisplayName
		items = append(items, item)
	}
	return items
}

func parseClassicBattleLootNameAndCount(value string) (string, int) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", 0
	}
	name := value
	count := 1
	if splitIndex := strings.LastIndex(value, "x"); splitIndex > 0 && splitIndex < len(value)-1 {
		if parsed, err := strconv.Atoi(strings.TrimSpace(value[splitIndex+1:])); err == nil && parsed > 0 {
			name = strings.TrimSpace(value[:splitIndex])
			count = parsed
		}
	}
	return name, count
}

func finalizeClassicBattleOver(store *session.Store, socketSession *packetSession, result battle.ResultPayload) (*session.RoleState, *session.RolePhysique) {
	roleState := updatePlayerBaseRoleStateFromBattle(socketSession)
	if store == nil || socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		return roleState, nil
	}
	if result.Winner != battle.CampTeam || result.Escaped || result.ExpDelta <= 0 {
		if roleState != nil {
			if role, playerBase, ok := store.UpdateRoleState(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, *roleState); ok {
				socketSession.selectedRole = &role
				socketSession.playerBase = &playerBase
				return playerBase.RoleState, playerBase.RolePhysique
			}
		}
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
	if role, playerBase, ok := store.UpdateRoleState(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, expResult.RoleState); ok {
		socketSession.selectedRole = &role
		socketSession.playerBase = &playerBase
		if playerBase.RoleState != nil && playerBase.RolePhysique != nil {
			return playerBase.RoleState, playerBase.RolePhysique
		}
	}
	return &expResult.RoleState, &expResult.RolePhysique
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
	if sourceType == classicBattleLootType && targetType == classicBattleLootType && request.SourceIndex != nil && request.TargetIndex != nil {
		return buildClassicBattleLootExchangeResult(socketSession, *request.SourceIndex, *request.TargetIndex, request.Count)
	}
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
	if sourceType == "装备" && targetType == "背包" && request.SourceIndex != nil {
		targetIndex := -1
		if request.TargetIndex != nil {
			targetIndex = *request.TargetIndex
		}
		moveResult := store.MoveRoleItem(
			socketSession.playerBase.PlayerID,
			socketSession.selectedRole.RoleID,
			sourceType,
			*request.SourceIndex,
			targetType,
			targetIndex,
			0,
		)
		if !moveResult.Found || !moveResult.Moved {
			log.Printf("[ai-server] classic town ContainerMove equipment noop roleId=%s source=%s[%v] target=%s[%v] error=%s", socketSession.selectedRole.RoleID, sourceType, request.SourceIndex, targetType, request.TargetIndex, moveResult.ErrorCode)
			return packetResult{handled: true}
		}
		socketSession.selectedRole = &moveResult.Role
		socketSession.playerBase = &moveResult.PlayerBase
		result := packetResult{
			createPlayer: buildClassicTownCreatePlayerPush(moveResult.Role, moveResult.PlayerBase),
			itemInfos:    make([]classicTownItemInfoPush, 0, len(moveResult.UpdatedItems)),
			itemClears:   make([]classicTownItemInfoClearPush, 0, len(moveResult.ClearedItems)),
			handled:      true,
		}
		for _, clear := range moveResult.ClearedItems {
			result.itemClears = append(result.itemClears, classicTownItemInfoClearPush{
				Handle: moveResult.Role.RoleID,
				Type:   clear.Type,
				Index:  clear.Index,
			})
		}
		for _, item := range moveResult.UpdatedItems {
			item.Handle = moveResult.Role.RoleID
			result.itemInfos = append(result.itemInfos, classicTownItemInfoPushFromRoleItem(item))
		}
		result.rolePhysique = moveResult.PlayerBase.RolePhysique
		return result
	}

	log.Printf("[ai-server] classic town ContainerMove ignored unsupported source=%s target=%s", sourceType, targetType)
	return packetResult{handled: true}
}

func buildClassicBattleLootExchangeResult(socketSession *packetSession, sourceIndex int, targetIndex int, count *int) packetResult {
	if sourceIndex == targetIndex {
		return packetResult{handled: true}
	}

	sourcePosition := -1
	targetPosition := -1
	for index, item := range socketSession.battleLoot {
		if item.Type != classicBattleLootType {
			continue
		}
		if item.Index == sourceIndex {
			sourcePosition = index
		}
		if item.Index == targetIndex {
			targetPosition = index
		}
	}
	if sourcePosition < 0 {
		return packetResult{handled: true}
	}

	sourceItem := socketSession.battleLoot[sourcePosition]
	moveCount := sourceItem.Count
	if count != nil && *count > 0 && *count < moveCount {
		moveCount = *count
	}
	result := packetResult{
		itemInfos:  []classicTownItemInfoPush{},
		itemClears: []classicTownItemInfoClearPush{},
		handled:    true,
	}
	pushItem := func(item session.RoleItem) {
		item.Handle = socketSession.selectedRole.RoleID
		result.itemInfos = append(result.itemInfos, classicTownItemInfoPushFromRoleItem(item))
	}
	clearSlot := func(index int) {
		result.itemClears = append(result.itemClears, classicTownItemInfoClearPush{
			Handle: socketSession.selectedRole.RoleID,
			Type:   classicBattleLootType,
			Index:  index,
		})
	}

	if targetPosition >= 0 {
		targetItem := socketSession.battleLoot[targetPosition]
		if strings.TrimSpace(sourceItem.Name) != "" && sourceItem.Name == targetItem.Name {
			targetItem.Count += moveCount
			socketSession.battleLoot[targetPosition] = targetItem
			if moveCount < sourceItem.Count {
				sourceItem.Count -= moveCount
				socketSession.battleLoot[sourcePosition] = sourceItem
				pushItem(sourceItem)
			} else {
				socketSession.battleLoot = append(socketSession.battleLoot[:sourcePosition], socketSession.battleLoot[sourcePosition+1:]...)
				clearSlot(sourceIndex)
			}
			pushItem(targetItem)
			return result
		}

		if moveCount < sourceItem.Count {
			return packetResult{handled: true}
		}
		sourceItem.Index = targetIndex
		targetItem.Index = sourceIndex
		socketSession.battleLoot[sourcePosition] = sourceItem
		socketSession.battleLoot[targetPosition] = targetItem
		clearSlot(sourceIndex)
		clearSlot(targetIndex)
		pushItem(sourceItem)
		pushItem(targetItem)
		return result
	}

	if moveCount < sourceItem.Count {
		sourceItem.Count -= moveCount
		moved := sourceItem
		moved.Count = moveCount
		moved.Index = targetIndex
		socketSession.battleLoot[sourcePosition] = sourceItem
		socketSession.battleLoot = append(socketSession.battleLoot, moved)
		pushItem(sourceItem)
		pushItem(moved)
		return result
	}

	sourceItem.Index = targetIndex
	socketSession.battleLoot[sourcePosition] = sourceItem
	clearSlot(sourceIndex)
	pushItem(sourceItem)
	return result
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

func buildClassicTownActiveItemResult(store *session.Store, socketSession *packetSession, request classicTownActiveItemRequest) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic town ActiveItem ignored without selected role type=%s index=%d", request.Type, request.Index)
		return packetResult{handled: true}
	}

	useResult := store.UseRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, request.Type, request.Index)
	if !useResult.Found {
		log.Printf("[ai-server] classic town ActiveItem ignored missing role roleId=%s type=%s index=%d", socketSession.selectedRole.RoleID, request.Type, request.Index)
		return packetResult{handled: true}
	}
	if !useResult.Used {
		log.Printf("[ai-server] classic town ActiveItem rejected roleId=%s type=%s index=%d error=%s", socketSession.selectedRole.RoleID, request.Type, request.Index, useResult.ErrorCode)
		return packetResult{
			chatMessages: []classicTownChatMessagePush{classicTownSystemWarningMessage(useResult.ErrorMessage)},
			handled:      true,
		}
	}

	socketSession.selectedRole = &useResult.Role
	socketSession.playerBase = &useResult.PlayerBase
	result := packetResult{
		itemInfos:  make([]classicTownItemInfoPush, 0, len(useResult.UpdatedItems)),
		itemClears: make([]classicTownItemInfoClearPush, 0, len(useResult.ClearedItems)),
		handled:    true,
	}
	for _, clear := range useResult.ClearedItems {
		result.itemClears = append(result.itemClears, classicTownItemInfoClearPush{
			Handle: useResult.Role.RoleID,
			Type:   clear.Type,
			Index:  clear.Index,
		})
	}
	for _, item := range useResult.UpdatedItems {
		item.Handle = useResult.Role.RoleID
		result.itemInfos = append(result.itemInfos, classicTownItemInfoPushFromRoleItem(item))
	}
	if len(useResult.Currencies) > 0 {
		result.currencyPush = buildClassicTownCurrencyPush(useResult.Role.RoleID, useResult.Currencies)
	}
	if useResult.LearnedSkill != nil {
		result.skillCap = &classicTownSkillCapPush{Count: 12}
		result.skillInfos = []classicTownSkillInfoPush{
			classicTownSkillInfoPushFromRoleSkill(useResult.Role.RoleID, *useResult.LearnedSkill),
		}
		result.chatMessages = append(result.chatMessages, classicTownSystemChatMessage(
			"习得【"+useResult.LearnedSkill.Name+"】Lv."+strconv.Itoa(useResult.LearnedSkill.Level),
		))
	}
	if useResult.Equipped {
		result.createPlayer = buildClassicTownCreatePlayerPush(useResult.Role, useResult.PlayerBase)
		result.rolePhysique = useResult.PlayerBase.RolePhysique
	}
	if useResult.RoleStateChanged {
		result.roleState = useResult.PlayerBase.RoleState
	}
	log.Printf("[ai-server] classic town ActiveItem roleId=%s type=%s index=%d item=%s", useResult.Role.RoleID, request.Type, request.Index, useResult.Item.Name)
	return result
}

func buildClassicTownDestroyItemResult(store *session.Store, socketSession *packetSession, request classicTownDestroyItemRequest) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic town DestroyItem ignored without selected role type=%s index=%d count=%d", request.Type, request.Index, request.Count)
		return packetResult{handled: true}
	}

	useResult := store.ConsumeRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, request.Type, request.Index, request.Count)
	if !useResult.Found {
		log.Printf("[ai-server] classic town DestroyItem ignored missing role roleId=%s type=%s index=%d count=%d", socketSession.selectedRole.RoleID, request.Type, request.Index, request.Count)
		return packetResult{handled: true}
	}
	if !useResult.Used {
		log.Printf("[ai-server] classic town DestroyItem rejected roleId=%s type=%s index=%d count=%d error=%s", socketSession.selectedRole.RoleID, request.Type, request.Index, request.Count, useResult.ErrorCode)
		return packetResult{handled: true}
	}

	socketSession.selectedRole = &useResult.Role
	socketSession.playerBase = &useResult.PlayerBase
	result := packetResult{
		itemInfos:  make([]classicTownItemInfoPush, 0, 1),
		itemClears: make([]classicTownItemInfoClearPush, 0, len(useResult.ClearedItems)),
		handled:    true,
	}
	for _, clear := range useResult.ClearedItems {
		result.itemClears = append(result.itemClears, classicTownItemInfoClearPush{
			Handle: useResult.Role.RoleID,
			Type:   clear.Type,
			Index:  clear.Index,
		})
	}
	if useResult.UpdatedItem != nil {
		item := *useResult.UpdatedItem
		item.Handle = useResult.Role.RoleID
		result.itemInfos = append(result.itemInfos, classicTownItemInfoPushFromRoleItem(item))
	}
	if request.Type == "装备" {
		result.createPlayer = buildClassicTownCreatePlayerPush(useResult.Role, useResult.PlayerBase)
		result.rolePhysique = useResult.PlayerBase.RolePhysique
	}
	log.Printf("[ai-server] classic town DestroyItem roleId=%s type=%s index=%d count=%d item=%s", useResult.Role.RoleID, request.Type, request.Index, request.Count, useResult.Item.Name)
	return result
}

func buildClassicTownSaleItemResult(store *session.Store, socketSession *packetSession, request classicTownSaleItemRequest) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic town SaleItem ignored without selected role shopId=%s type=%s index=%d count=%d", request.ShopID, request.Type, request.Index, request.Count)
		return packetResult{handled: true}
	}

	saleResult := store.SellRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, request.Type, request.Index, request.Count)
	if !saleResult.Found {
		log.Printf("[ai-server] classic town SaleItem ignored missing role roleId=%s shopId=%s type=%s index=%d count=%d", socketSession.selectedRole.RoleID, request.ShopID, request.Type, request.Index, request.Count)
		return packetResult{handled: true}
	}
	if !saleResult.Sold {
		log.Printf("[ai-server] classic town SaleItem rejected roleId=%s shopId=%s type=%s index=%d count=%d error=%s", socketSession.selectedRole.RoleID, request.ShopID, request.Type, request.Index, request.Count, saleResult.ErrorCode)
		return packetResult{
			chatMessages: []classicTownChatMessagePush{classicTownSystemWarningMessage(saleResult.ErrorMessage)},
			handled:      true,
		}
	}

	socketSession.selectedRole = &saleResult.Role
	socketSession.playerBase = &saleResult.PlayerBase
	saleCount := request.Count
	if saleCount <= 0 {
		saleCount = 1
	}
	result := packetResult{
		currencyPush: buildClassicTownCurrencyPush(saleResult.Role.RoleID, saleResult.Currencies),
		itemInfos:    make([]classicTownItemInfoPush, 0, 1),
		itemClears:   make([]classicTownItemInfoClearPush, 0, len(saleResult.ClearedItems)),
		chatMessages: []classicTownChatMessagePush{
			classicTownSystemChatMessage("出售了【" + saleResult.Item.Name + "】x" + strconv.Itoa(saleCount) + "，获得铜钱" + strconv.Itoa(saleResult.Amount) + "。"),
		},
		handled: true,
	}
	for _, clear := range saleResult.ClearedItems {
		result.itemClears = append(result.itemClears, classicTownItemInfoClearPush{
			Handle: saleResult.Role.RoleID,
			Type:   clear.Type,
			Index:  clear.Index,
		})
	}
	if saleResult.UpdatedItem != nil {
		item := *saleResult.UpdatedItem
		item.Handle = saleResult.Role.RoleID
		result.itemInfos = append(result.itemInfos, classicTownItemInfoPushFromRoleItem(item))
	}
	log.Printf("[ai-server] classic town SaleItem roleId=%s shopId=%s type=%s index=%d count=%d item=%s amount=%d", saleResult.Role.RoleID, request.ShopID, request.Type, request.Index, request.Count, saleResult.Item.Name, saleResult.Amount)
	return result
}

func buildClassicTownChatSendResult(socketSession *packetSession, request classicTownChatSendRequest) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil {
		log.Printf("[ai-server] classic town Speak ignored without selected role channel=%s", request.Channel)
		return packetResult{handled: true}
	}

	message := strings.TrimSpace(request.Msg)
	if message == "" {
		return packetResult{handled: true}
	}

	channel := normalizeClassicTownChatChannel(request.Channel)
	role := socketSession.selectedRole
	push := classicTownChatMessagePush{
		Channel: channel,
		Handle:  role.RoleID,
		Name:    role.DisplayName,
		Msg:     message,
	}
	if channel == "whisper" {
		targetName := strings.TrimSpace(request.TargetName)
		if targetName == "" {
			return packetResult{handled: true}
		}
		push.TargetName = targetName
		push.Outgoing = true
	}

	log.Printf("[ai-server] classic town Speak roleId=%s channel=%s msg=%s", role.RoleID, channel, message)
	return packetResult{
		chatMessages: []classicTownChatMessagePush{push},
		handled:      true,
	}
}

func classicTownSystemChatMessage(message string) classicTownChatMessagePush {
	return classicTownChatMessagePush{
		Channel: "system",
		Msg:     message,
	}
}

func classicTownSystemWarningMessage(message string) classicTownChatMessagePush {
	push := classicTownSystemChatMessage(message)
	push.Color = "#ff0000"
	push.Bold = true
	return push
}

func buildClassicTownHealerResult(
	store *session.Store,
	socketSession *packetSession,
	request classicTownAnswerRequest,
) (packetResult, bool) {
	if !isClassicTownHealerAnswer(request.Handle, request.AnswerHandle) {
		return packetResult{}, false
	}
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic town Healer ignored without selected role handle=%s", request.Handle)
		return packetResult{handled: true}, true
	}

	healResult := store.HealRoleAtTown(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID)
	if !healResult.Found {
		log.Printf("[ai-server] classic town Healer ignored missing role roleId=%s handle=%s", socketSession.selectedRole.RoleID, request.Handle)
		return packetResult{handled: true}, true
	}

	socketSession.selectedRole = &healResult.Role
	socketSession.playerBase = &healResult.PlayerBase
	result := packetResult{
		currencyPush: buildClassicTownCurrencyPush(healResult.Role.RoleID, healResult.Currencies),
		itemInfos:    make([]classicTownItemInfoPush, 0, len(healResult.UpdatedItems)),
		itemClears:   make([]classicTownItemInfoClearPush, 0, len(healResult.ClearedItems)),
		handled:      true,
	}
	for _, updatedItem := range healResult.UpdatedItems {
		updatedItem.Handle = healResult.Role.RoleID
		result.itemInfos = append(result.itemInfos, classicTownItemInfoPushFromRoleItem(updatedItem))
	}
	for _, clear := range healResult.ClearedItems {
		result.itemClears = append(result.itemClears, classicTownItemInfoClearPush{
			Handle: healResult.Role.RoleID,
			Type:   clear.Type,
			Index:  clear.Index,
		})
	}
	switch {
	case healResult.Healed:
		result.roleState = &healResult.RoleState
	case healResult.NearlyFull:
		answerSpeak := world.BuildAnswerSpeak(request.Handle)
		result.answerSpeak = &world.AnswerSpeakPush{
			Handle:    request.Handle,
			MsgHandle: request.MsgHandle,
			Msg:       "你几乎不需要治疗了",
			Answers:   answerSpeak.Answers,
		}
	default:
		result.chatMessages = append(result.chatMessages, classicTownSystemWarningMessage(healResult.ErrorMessage))
	}
	log.Printf("[ai-server] classic town Healer roleId=%s handle=%s healed=%v cost=%d error=%s", healResult.Role.RoleID, request.Handle, healResult.Healed, healResult.Cost, healResult.ErrorCode)
	return result, true
}

func isClassicTownHealerAnswer(handle string, answerHandle string) bool {
	if strings.TrimSpace(answerHandle) != "2" {
		return false
	}
	switch strings.TrimSpace(handle) {
	case "6000542609425103",
		"4110542614676637",
		"2520542613299551",
		"4950542616589339",
		"4710542615621525":
		return true
	default:
		return false
	}
}

func normalizeClassicTownChatChannel(channel string) string {
	switch strings.ToLower(strings.TrimSpace(channel)) {
	case "1", "world", "public", "p":
		return "world"
	case "2", "team", "t":
		return "team"
	case "3", "whisper", "w":
		return "whisper"
	case "5", "guild", "guid", "g":
		return "guild"
	case "50", "system":
		return "system"
	case "0", "region", "local", "say":
		return "region"
	default:
		return "region"
	}
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

func buildClassicTownRemoveSkillResult(
	store *session.Store,
	socketSession *packetSession,
	request classicTownRemoveSkillRequest,
) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic town RemoveSkill ignored without selected role skill=%s", request.Name)
		return packetResult{handled: true}
	}

	removeResult := store.RemoveRoleSkill(
		socketSession.playerBase.PlayerID,
		socketSession.selectedRole.RoleID,
		request.Name,
	)
	if !removeResult.Found {
		log.Printf("[ai-server] classic town RemoveSkill ignored missing role roleId=%s skill=%s", socketSession.selectedRole.RoleID, request.Name)
		return packetResult{handled: true}
	}
	if !removeResult.Removed || removeResult.RemovedSkill == nil {
		log.Printf("[ai-server] classic town RemoveSkill rejected roleId=%s skill=%s error=%s", socketSession.selectedRole.RoleID, request.Name, removeResult.ErrorCode)
		return packetResult{handled: true}
	}

	log.Printf("[ai-server] classic town RemoveSkill removed roleId=%s skill=%s", socketSession.selectedRole.RoleID, removeResult.RemovedSkill.Name)
	return packetResult{
		skillClears: []classicTownClearSkillInfoPush{{
			Handle: socketSession.selectedRole.RoleID,
			Name:   removeResult.RemovedSkill.Name,
			Level:  removeResult.RemovedSkill.Level,
		}},
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

	if strings.HasPrefix(strings.TrimSpace(request.ShopID), "item:") {
		return buildClassicTownBuyItemResult(store, socketSession, request)
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

	purchase := store.PurchaseRoleItem(
		socketSession.playerBase.PlayerID,
		socketSession.selectedRole.RoleID,
		sourceSkillEntryToRoleItem(entry),
		sourceSkillRequirementsToRoleItemRequirements(entry.Requirements),
	)
	if !purchase.Found {
		log.Printf("[ai-server] classic town BuySkill ignored missing role roleId=%s shopId=%s skillId=%d", socketSession.selectedRole.RoleID, request.ShopID, request.SkillID)
		return packetResult{handled: true}
	}

	socketSession.selectedRole = &purchase.Role
	socketSession.playerBase = &purchase.PlayerBase
	result := packetResult{
		currencyPush: buildClassicTownCurrencyPush(
			purchase.Role.RoleID,
			purchase.Currencies,
		),
		buySkillResult: &classicTownBuySkillResultPush{
			Success:      purchase.Purchased,
			ShopID:       request.ShopID,
			SkillID:      request.SkillID,
			Currencies:   purchase.Currencies,
			ErrorCode:    purchase.ErrorCode,
			ErrorMessage: purchase.ErrorMessage,
		},
		itemInfos:  make([]classicTownItemInfoPush, 0, 1+len(purchase.Consumed)),
		itemClears: make([]classicTownItemInfoClearPush, 0, len(purchase.ClearedItems)),
		handled:    true,
	}
	if purchase.Purchased {
		grantedItem := purchase.Item
		grantedItem.Handle = purchase.Role.RoleID
		result.itemInfos = append(result.itemInfos, classicTownItemInfoPushFromRoleItem(grantedItem))
		result.chatMessages = append(result.chatMessages, classicTownSystemChatMessage("购买了【"+entry.Name+"】x1。"))
		for _, consumedItem := range purchase.Consumed {
			consumedItem.Handle = purchase.Role.RoleID
			result.itemInfos = append(result.itemInfos, classicTownItemInfoPushFromRoleItem(consumedItem))
		}
		for _, clear := range purchase.ClearedItems {
			result.itemClears = append(result.itemClears, classicTownItemInfoClearPush{
				Handle: purchase.Role.RoleID,
				Type:   clear.Type,
				Index:  clear.Index,
			})
		}
		log.Printf("[ai-server] classic town BuySkill purchased skill item roleId=%s shopId=%s skillId=%d skill=%s", purchase.Role.RoleID, request.ShopID, request.SkillID, entry.Name)
	} else {
		log.Printf("[ai-server] classic town BuySkill rejected roleId=%s shopId=%s skillId=%d error=%s", purchase.Role.RoleID, request.ShopID, request.SkillID, purchase.ErrorCode)
	}
	return result
}

func buildClassicTownBuyItemResult(
	store *session.Store,
	socketSession *packetSession,
	request classicTownBuySkillRequest,
) packetResult {
	row, ok := findSourceItemShopRow(request.ShopID, request.SkillID)
	if !ok {
		return packetResult{
			buySkillResult: &classicTownBuySkillResultPush{
				Success:      false,
				ShopID:       request.ShopID,
				SkillID:      request.SkillID,
				ErrorCode:    "item_missing",
				ErrorMessage: "商品不存在。",
			},
			handled: true,
		}
	}

	purchase := store.PurchaseRoleItem(
		socketSession.playerBase.PlayerID,
		socketSession.selectedRole.RoleID,
		sourceItemShopRowToRoleItem(row),
		sourceItemShopRequirementsToRoleItemRequirements(row.requirements),
	)
	if !purchase.Found {
		log.Printf("[ai-server] classic town BuyItem ignored missing role roleId=%s shopId=%s itemId=%d", socketSession.selectedRole.RoleID, request.ShopID, request.SkillID)
		return packetResult{handled: true}
	}

	socketSession.selectedRole = &purchase.Role
	socketSession.playerBase = &purchase.PlayerBase
	result := packetResult{
		currencyPush: buildClassicTownCurrencyPush(
			purchase.Role.RoleID,
			purchase.Currencies,
		),
		buySkillResult: &classicTownBuySkillResultPush{
			Success:      purchase.Purchased,
			ShopID:       request.ShopID,
			SkillID:      request.SkillID,
			Currencies:   purchase.Currencies,
			ErrorCode:    purchase.ErrorCode,
			ErrorMessage: purchase.ErrorMessage,
		},
		itemInfos:  make([]classicTownItemInfoPush, 0, 1+len(purchase.Consumed)),
		itemClears: make([]classicTownItemInfoClearPush, 0, len(purchase.ClearedItems)),
		handled:    true,
	}
	if purchase.Purchased {
		grantedItem := purchase.Item
		grantedItem.Handle = purchase.Role.RoleID
		result.itemInfos = append(result.itemInfos, classicTownItemInfoPushFromRoleItem(grantedItem))
		result.chatMessages = append(result.chatMessages, classicTownSystemChatMessage("购买了【"+row.name+"】x"+strconv.Itoa(row.count)+"。"))
		for _, consumedItem := range purchase.Consumed {
			consumedItem.Handle = purchase.Role.RoleID
			result.itemInfos = append(result.itemInfos, classicTownItemInfoPushFromRoleItem(consumedItem))
		}
		for _, clear := range purchase.ClearedItems {
			result.itemClears = append(result.itemClears, classicTownItemInfoClearPush{
				Handle: purchase.Role.RoleID,
				Type:   clear.Type,
				Index:  clear.Index,
			})
		}
		log.Printf("[ai-server] classic town BuyItem purchased roleId=%s shopId=%s itemId=%d item=%s count=%d", purchase.Role.RoleID, request.ShopID, request.SkillID, row.name, row.count)
	} else {
		log.Printf("[ai-server] classic town BuyItem rejected roleId=%s shopId=%s itemId=%d error=%s", purchase.Role.RoleID, request.ShopID, request.SkillID, purchase.ErrorCode)
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
	entryResult, ok := consumeDungeonEntryTicketIfNeeded(store, socketSession, mapID)
	if !ok {
		return entryResult
	}
	role, playerBase, ok := store.UpdateRoleMap(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, mapID)
	if !ok {
		log.Printf("[ai-server] classic town transfer ignored missing role roleId=%s mapId=%s", socketSession.selectedRole.RoleID, mapIDText)
		return packetResult{handled: true}
	}
	socketSession.selectedRole = &role
	socketSession.playerBase = &playerBase
	dungeonInstance := syncDungeonInstanceState(store, socketSession, mapID)
	bootstrap, ok := world.BuildTownTransferBootstrap(role, playerBase, mapID, spawn)
	if !ok {
		return packetResult{handled: true}
	}
	filterDefeatedVisibleMonsters(&bootstrap, socketSession)
	log.Printf("[ai-server] classic town transfer mapId=%s x=%d y=%d roleId=%s", mapIDText, spawn.X, spawn.Y, role.RoleID)
	return packetResult{
		townBootstrap:   &bootstrap,
		dungeonInstance: dungeonInstance,
		itemInfos:       entryResult.itemInfos,
		itemClears:      entryResult.itemClears,
		chatMessages:    entryResult.chatMessages,
		handled:         true,
	}
}

func resolveClassicTownTransportAnswer(
	socketSession *packetSession,
	requestMapID string,
	handle string,
	answerHandle string,
) (world.TownTransportDestination, bool) {
	fromMapID := 0
	if requestMapID != "" {
		if mapID, err := strconv.Atoi(requestMapID); err == nil {
			fromMapID = mapID
		}
	}
	if fromMapID == 0 && socketSession != nil && socketSession.selectedRole != nil {
		fromMapID = socketSession.selectedRole.MapID
	}
	if fromMapID > 0 {
		return world.ResolveTownTransportAnswerFromMap(fromMapID, handle, answerHandle)
	}
	return world.ResolveTownTransportAnswer(handle, answerHandle)
}

func consumeDungeonEntryTicketIfNeeded(store *session.Store, socketSession *packetSession, targetMapID int) (packetResult, bool) {
	if store == nil || socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		return packetResult{handled: true}, false
	}
	targetInstanceKey, ok := world.DungeonInstanceKeyForMapID(targetMapID)
	if !ok {
		return packetResult{handled: true}, true
	}
	rule, ok := dungeonEntryRuleForInstance(targetInstanceKey)
	if !ok || rule.ConsumePolicy == dungeonEntryConsumeNone || rule.TicketName == "" {
		return packetResult{handled: true}, true
	}
	currentMapID := socketSession.selectedRole.MapID
	currentInstanceKey, currentInDungeon := world.DungeonInstanceKeyForMapID(currentMapID)
	if currentInDungeon && currentInstanceKey == targetInstanceKey {
		return packetResult{handled: true}, true
	}
	if _, ok := store.GetRoleDungeonInstance(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, targetInstanceKey); ok {
		return packetResult{handled: true}, true
	}
	ticket, ok := findRoleTicketItem(store, socketSession, rule.TicketName)
	if !ok || ticket.Count < rule.TicketCount {
		message := "进入" + dungeonInstanceDisplayName(targetInstanceKey) + "需要" + rule.TicketName + "x" + strconv.Itoa(rule.TicketCount) + "。"
		log.Printf("[ai-server] classic town transfer rejected missing dungeon ticket roleId=%s mapId=%d ticket=%s", socketSession.selectedRole.RoleID, targetMapID, rule.TicketName)
		return packetResult{
			chatMessages: []classicTownChatMessagePush{classicTownSystemWarningMessage(message)},
			handled:      true,
		}, false
	}
	useResult := store.ConsumeRoleItem(
		socketSession.playerBase.PlayerID,
		socketSession.selectedRole.RoleID,
		ticket.Type,
		ticket.Index,
		rule.TicketCount,
	)
	if !useResult.Found || !useResult.Used {
		log.Printf("[ai-server] classic town transfer rejected consume dungeon ticket roleId=%s mapId=%d ticket=%s error=%s", socketSession.selectedRole.RoleID, targetMapID, rule.TicketName, useResult.ErrorCode)
		return packetResult{
			chatMessages: []classicTownChatMessagePush{classicTownSystemWarningMessage(useResult.ErrorMessage)},
			handled:      true,
		}, false
	}

	socketSession.selectedRole = &useResult.Role
	socketSession.playerBase = &useResult.PlayerBase
	result := packetResult{
		itemInfos:  make([]classicTownItemInfoPush, 0, len(useResult.UpdatedItems)+1),
		itemClears: make([]classicTownItemInfoClearPush, 0, len(useResult.ClearedItems)),
		handled:    true,
	}
	if useResult.UpdatedItem != nil {
		item := *useResult.UpdatedItem
		item.Handle = useResult.Role.RoleID
		result.itemInfos = append(result.itemInfos, classicTownItemInfoPushFromRoleItem(item))
	}
	for _, item := range useResult.UpdatedItems {
		item.Handle = useResult.Role.RoleID
		result.itemInfos = append(result.itemInfos, classicTownItemInfoPushFromRoleItem(item))
	}
	for _, clear := range useResult.ClearedItems {
		result.itemClears = append(result.itemClears, classicTownItemInfoClearPush{
			Handle: useResult.Role.RoleID,
			Type:   clear.Type,
			Index:  clear.Index,
		})
	}
	return result, true
}

func dungeonEntryRuleForInstance(instanceKey string) (dungeonEntryRule, bool) {
	switch instanceKey {
	case session.DungeonInstanceShuiliandong:
		return dungeonEntryRule{
			InstanceKey:   instanceKey,
			TicketName:    "水帘洞通行证",
			TicketCount:   1,
			ConsumePolicy: dungeonEntryConsumeOnNewInstance,
		}, true
	case session.DungeonInstanceHuangfengzhai:
		return dungeonEntryRule{
			InstanceKey:   instanceKey,
			TicketName:    "黄风寨通行证",
			TicketCount:   1,
			ConsumePolicy: dungeonEntryConsumeOnNewInstance,
		}, true
	case session.DungeonInstanceFeixiandong:
		return dungeonEntryRule{
			InstanceKey:   instanceKey,
			TicketName:    "飞仙洞通行证",
			TicketCount:   1,
			ConsumePolicy: dungeonEntryConsumeOnNewInstance,
		}, true
	default:
		return dungeonEntryRule{}, false
	}
}

func findRoleTicketItem(store *session.Store, socketSession *packetSession, ticketName string) (session.RoleItem, bool) {
	items, _, ok := store.GetRoleItems(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, classicTownBagContainerType)
	if !ok {
		return session.RoleItem{}, false
	}
	for _, item := range items {
		if item.Name == ticketName && item.Count > 0 {
			return item, true
		}
	}
	return session.RoleItem{}, false
}

func filterDefeatedVisibleMonsters(snapshot *world.TownBootstrapSnapshot, socketSession *packetSession) {
	if snapshot == nil || socketSession == nil || len(socketSession.defeatedVisibleMonsters) == 0 {
		return
	}
	createRoles := snapshot.CreateRoles[:0]
	for _, rolePush := range snapshot.CreateRoles {
		if rolePush.Kind == "monster" && socketSession.defeatedVisibleMonsters[rolePush.Handle] {
			continue
		}
		createRoles = append(createRoles, rolePush)
	}
	snapshot.CreateRoles = createRoles
}

func syncDungeonInstanceState(store *session.Store, socketSession *packetSession, mapID int) *classicTownDungeonInstancePush {
	if store == nil || socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		return nil
	}
	instanceKey, ok := world.DungeonInstanceKeyForMapID(mapID)
	if !ok {
		setDefeatedVisibleMonsterHandles(socketSession, nil)
		return &classicTownDungeonInstancePush{
			MapID:           strconv.Itoa(mapID),
			Active:          false,
			DurationSeconds: session.DungeonInstanceTTLSeconds(),
		}
	}
	state, ok := store.EnsureRoleDungeonInstance(
		socketSession.playerBase.PlayerID,
		socketSession.selectedRole.RoleID,
		instanceKey,
	)
	if !ok {
		return nil
	}
	setDefeatedVisibleMonsterHandles(socketSession, state.DefeatedVisibleMonsterHandles)
	expiresAtUnix := session.DungeonInstanceExpiresAtUnix(state)
	remainingSeconds := expiresAtUnix - time.Now().Unix()
	if remainingSeconds < 0 {
		remainingSeconds = 0
	}
	return &classicTownDungeonInstancePush{
		Key:                           instanceKey,
		DisplayName:                   dungeonInstanceDisplayName(instanceKey),
		MapID:                         strconv.Itoa(mapID),
		Active:                        true,
		CreatedAtUnix:                 state.CreatedAtUnix,
		ExpiresAtUnix:                 expiresAtUnix,
		DurationSeconds:               session.DungeonInstanceTTLSeconds(),
		RemainingSeconds:              remainingSeconds,
		DefeatedVisibleMonsterHandles: append([]string{}, state.DefeatedVisibleMonsterHandles...),
	}
}

func dungeonInstanceDisplayName(key string) string {
	switch key {
	case session.DungeonInstanceShuiliandong:
		return "水帘洞"
	case session.DungeonInstanceHuangfengzhai:
		return "黄风寨"
	case session.DungeonInstanceFeixiandong:
		return "飞仙洞"
	default:
		return "副本"
	}
}
