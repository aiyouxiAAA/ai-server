package main

import (
	"log"
	"strconv"
	"strings"
	"time"

	"ai-server/internal/battle"
	"ai-server/internal/classicactivity"
	"ai-server/internal/guild"
	"ai-server/internal/mall"
	"ai-server/internal/protocol"
	"ai-server/internal/session"
	"ai-server/internal/team"
	"ai-server/internal/world"
)

type packetResult struct {
	responseCmd        uint64
	responsePayload    []byte
	serverTime         *classicTownServerTimePush
	townBootstrap      *world.TownBootstrapSnapshot
	gameTips           []classicTownGameTipPush
	answerSpeak        *world.AnswerSpeakPush
	subGame            *classicTownSubGamePush
	collectionComplete *classicTownCollectionCompletePush
	openContainer      *classicTownOpenContainerPush
	careState          *classicTownCareStatePush
	createPlayer       *world.RolePush
	roleState          *session.RoleState
	rolePhysique       *session.RolePhysique
	chatMessages       []classicTownChatMessagePush
	errorMessages      []classicTownErrorPush
	chatBroadcasts     []classicTownChatBroadcast
	abilityCount       *classicTownAbilityCountPush
	skillCap           *classicTownSkillCapPush
	skillInfos         []classicTownSkillInfoPush
	skillClears        []classicTownClearSkillInfoPush
	skillShop          *classicTownSkillShopPush
	currencyPush       *classicTownCurrencyPush
	fastPanel          *classicTownFastPanelPush
	townBuffs          []classicTownBuffInfoPush
	townBuffClears     []classicTownClearBuffInfoPush
	buySkillResult     *classicTownBuySkillResultPush
	buyBackRefresh     *classicTownBuyBackRefreshPush
	buyBackInfos       []classicTownBuyBackInfoPush
	containerCap       *classicTownContainerCapacityPush
	itemInfos          []classicTownItemInfoPush
	itemClears         []classicTownItemInfoClearPush
	otherEquipment     *classicTownOtherEquipmentPush
	tryEquip           *classicTownTryEquipPush
	questInfos         []classicQuestInfoPush
	questClears        []classicQuestClearPush
	questStates        []world.QuestStatePush
	dungeonInstance    *classicTownDungeonInstancePush
	mapSpecial         *classicTownMapSpecialPush
	autoBattleInfo     *classicAutoBattleInfoPush
	friendInfos        []classicSocialFriendEntry
	friendClears       []classicSocialClearEntry
	blackInfos         []classicSocialBlackEntry
	blackClears        []classicSocialClearEntry
	enemyInfos         []classicSocialEnemyEntry
	enemyClears        []classicSocialClearEntry
	guildInfo          *guild.Guild
	guildMembers       []guild.Member
	guildAuth          *guild.Auth
	guildNotice        *classicGuildNoticePush
	guildResult        *classicGuildResultPush
	guildMemberClears  []classicGuildMemberClearPush
	mallCategories     []mall.Category
	mallSearchCount    *mall.SearchCountPush
	mallSearchPage     *mall.SearchPagePush
	mallCurrency       *mall.CurrencyPush
	mallPurchase       *mall.PurchaseResult
	petInfo            *classicPetInfoPush
	petFeedResult      *classicPetFeedResultPush
	auctionOpen        *classicAuctionOpenPush
	auctionList        *classicAuctionListPush
	mailOpen           *classicMailOpenPush
	mailList           *classicMailListPush
	mailInfo           *classicMailInfoPush
	dailyRewardInfo    *classicDailyRewardInfoPush
	maskCodeChallenge  *classicMaskCodeChallengePush
	maskCodeResult     *classicMaskCodeResultPush
	teamEvents         []team.Event
	teamSyncTransfer   *classicTeamSyncTransfer
	teamDungeonReset   *classicTeamDungeonReset
	teamBattleStart    *classicTeamBattleStart
	teamBattleSync     *classicTeamBattleSync
	battleStart        *battle.StartPush
	battleCells        []battle.CellInfoPush
	battleCellCount    *classicBattleCellCountPush
	battleCommand      *battle.StartCommandPush
	battleActions      []battle.ActionPush
	battleStopCommand  *classicBattleStopCommandPush
	battleBuffs        []battle.BuffInfoPush
	battleClearBuffs   []battle.ClearBuffInfoPush
	battleClearCells   []classicBattleClearCellInfoPush
	battleOver         *battle.OverPush
	battleLoadProgress *classicBattleLoadProgressPush
	battleRelive       *classicBattleRelivePush
	removeRoleHandles  []string
	moveRole           *world.RoleMovePush
	// sceneTransferFromMapID 标记本次结果是由"传送/切图"触发的,值为玩家传送前的旧 mapId。
	// main.go 收到 townBootstrap 后据此调用 announceWorldSceneTransfer:给旧图邻居推 removeRole,
	// 在新图重新互推。首次进图(选角)不走这里,它的互推在 register 区的 syncWorldScenePresence 完成。
	sceneTransferFromMapID int
	sceneTransferSpawn     world.SpawnPoint
	handled                bool
}

type classicTownErrorPush struct {
	Msg           string `json:"msg"`
	SourceCapture string `json:"sourceCapture,omitempty"`
	Partial       bool   `json:"partial,omitempty"`
}

type classicTownGameTipPush struct {
	TipID         string   `json:"tipId"`
	Kind          string   `json:"kind,omitempty"`
	TargetName    string   `json:"targetName,omitempty"`
	HTMLText      string   `json:"htmlText,omitempty"`
	Fields        []string `json:"fields,omitempty"`
	SourceCapture string   `json:"sourceCapture,omitempty"`
}

const (
	classicTownGameTipNpcSourceCapture = "tmp/capture-timeline-feature-gap-audit.json#command=50110+c_gameTip+npc"
	classicTownGameTipNpcTipID         = "zzshow"
	classicTownGameTipNpcKind          = "npc"
	classicTownGameTipNpcTargetName    = "一心长态"
	classicTownGameTipNpcHTMLText      = "<font color='#ffff00' size='14'><b>去找一心常态【研习职业】。</b></font><br>转职后可以学习更多炫酷技能。"
)

type classicTeamSyncTransfer struct {
	ActorRoleID string
	FromMapID   int
	TargetMapID int
	Spawn       world.SpawnPoint
	Members     []team.Member
}

type classicTeamDungeonReset struct {
	ActorRoleID string
	InstanceKey string
	MapID       int
	Members     []team.Member
}

type classicTeamBattleStart struct {
	ActorRoleID string
	Runtime     *battle.Runtime
	Bundle      battle.StartBundle
	Members     []team.Member
}

type classicTeamBattleSync struct {
	ActorRoleID string
	Result      packetResult
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

type classicTownMapSpecialRequest struct {
	ID    string `json:"id,omitempty"`
	MapID string `json:"mapId,omitempty"`
}

type classicTownMapSpecialPush struct {
	LastTime      int64    `json:"lastTime,omitempty"`
	Entries       []string `json:"entries,omitempty"`
	SourceCapture string   `json:"sourceCapture,omitempty"`
}

type classicTownServerTimePush struct {
	ServerTime    int64  `json:"serverTime"`
	SourceCapture string `json:"sourceCapture,omitempty"`
}

const classicTownServerTimeSourceCapture = "D:/yzhgame/WOCClient/instances/instance2.staging/tmp/woc-proxy-captures/20260606_210926_394_session_08036/connections/20260606_210959_953_conn_0002/raw/server-to-client-0001.bin#c_serverTime"
const classicTownMapSpecialLastTimeSourceCapture = "tmp/capture-timeline-feature-gap-audit.json#command=50084+c_mapSpecial+lastTime"

const (
	classicTownAddPointNoPointSourceCapture            = "D:/yzhgame/WOCClient/instances/instance2.staging/tmp/woc-proxy-captures/20260606_210926_394_session_08036/connections/20260606_210959_953_conn_0002/raw/server-to-client-0001.bin#1007+c_Error after AddPoint(170)"
	classicTownActiveItemLevel1GiftBoxSourceCapture    = "D:/yzhgame/WOCClient/instances/instance2.staging/tmp/woc-proxy-captures/20260606_210926_394_session_08036/connections/20260606_210959_953_conn_0002/raw/server-to-client-0001.bin#96-#108 after ActiveItemByIndex(114) bag slot 1"
	classicTownActiveItemLevel5GiftBoxSourceCapture    = "D:/yzhgame/WOCClient/instances/instance2.staging/tmp/woc-proxy-captures/20260606_210926_394_session_08036/connections/20260606_210959_953_conn_0002/raw/server-to-client-0001.bin#112+c_Error after ActiveItemByIndex(114)"
	classicTownActiveItemLevel10GiftBoxSourceCapture   = "D:/yzhgame/WOCClient/instances/instance2.staging/tmp/woc-proxy-captures/20260606_210926_394_session_08036/connections/20260606_210959_953_conn_0002/raw/server-to-client-0001.bin#3298/#3302/#3306+c_Error after ActiveItemByIndex(114) bag slot 7; success #3310-#3322"
	classicTownActiveItemBagCapacityPatchSourceCapture = "D:/yzhgame/WOCClient/instances/instance2.staging/tmp/woc-proxy-captures/20260606_210926_394_session_08036/connections/20260606_210959_953_conn_0002/raw/server-to-client-0001.bin#3326/#3327/#3331 after ActiveItemByIndex(114) bag slot 7"
	classicTownActiveItemLevel5GiftBoxError            = "\u89d2\u8272\u7b49\u7ea7\u5fc5\u987b\u5230\u8fbeLv5"
	classicTownActiveItemLevel1GiftBoxName             = "\u0031\u7ea7\u793c\u76d2"
	classicTownActiveItemLevel5GiftBoxName             = "\u0035\u7ea7\u793c\u76d2"
	classicTownActiveItemLevel10GiftBoxFullError       = "\u80cc\u5305\u7a7a\u95f4\u4e0d\u8db3"
	classicTownActiveItemLevel10GiftBoxName            = "\u0031\u0030\u7ea7\u793c\u76d2"
	classicTownActiveItemBagCapacityPatchName          = "\u004c\u80cc\u5305\u8865\u4e01"
	classicTownSilverIngotExchangeItemName             = "\u94f6\u5143\u5b9d"
	classicTownCopperExchangeItemName                  = "\u94dc\u94b1"
	classicTownInitialExperienceCardName               = "\u004c\u521d\u9636\u7ecf\u9a8c\u5361"
	classicTownAdvancedExperienceCardName              = "\u004c\u8fdb\u9636\u7ecf\u9a8c\u5361"
	classicTownInitialExperienceBoostName              = "\u53cc\u500d\u7ecf\u9a8c"
	classicTownInitialExperienceBoostMessage           = "\u83b7\u5f97\u53cc\u500d\u7ecf\u9a8c\u65f6\u95f41\u5c0f\u65f6"
	classicTownAdvancedExperienceBoostMessage          = "\u83b7\u5f97\u53cc\u500d\u7ecf\u9a8c\u65f6\u95f43\u5c0f\u65f6"
	classicTownAddPointNoPointError                    = "你已经没有剩余点数了"
)

const (
	classicTownInactiveTransportHandle        = "4170542615108676"
	classicTownInactiveTransportMsgHandle     = "1"
	classicTownInactiveTransportAnswerHandle  = "3"
	classicTownInactiveTransportSourceCapture = "D:/yzhgame/WOCClient/instances/instance2.staging/tmp/woc-proxy-captures/20260606_210926_394_session_08036/connections/20260606_210959_953_conn_0002/raw/server-to-client-0001.bin#1835+c_Error after Speak(101)"
	classicTownInactiveTransportError         = "传送点未激活！"
)

const (
	classicBattleCellCountSourceCapture     = "D:/yzhgame/WOCClient/instances/instance2.staging/tmp/woc-proxy-captures/20260606_210926_394_session_08036/connections/20260606_210959_953_conn_0002/raw/server-to-client-0001.bin#297+c_battleCellCount"
	classicBattleStopCommandSourceCapture   = "D:/yzhgame/WOCClient/instances/instance2.staging/tmp/woc-proxy-captures/20260606_210926_394_session_08036/connections/20260606_210959_953_conn_0002/raw/server-to-client-0001.bin#307+c_stopListenBattleCommand"
	classicBattleLoadProgressSourceCapture  = "D:/yzhgame/WOCClient/instances/instance2.staging/tmp/woc-proxy-captures/20260606_210926_394_session_08036/connections/20260606_210959_953_conn_0002/raw/server-to-client-0001.bin#300+c_battleLoadPro"
	classicBattleLoadProgressRequestCapture = "D:/yzhgame/WOCClient/instances/instance2.staging/tmp/woc-proxy-captures/20260606_210926_394_session_08036/connections/20260606_210959_953_conn_0002/raw/client-to-server-0001.bin#159+BattleLoadPro"
	classicBattleRoleReadyRequestCapture    = "D:/yzhgame/WOCClient/instances/instance2.staging/tmp/woc-proxy-captures/20260606_210926_394_session_08036/connections/20260606_210959_953_conn_0002/raw/client-to-server-0001.bin#160+BattleRoleReady"
	classicBattleClearCellInfoSourceCapture = "D:/yzhgame/WOCClient/instances/instance2.staging/tmp/woc-proxy-captures/20260619_152122_327_session_19024/connections/20260619_152158_516_conn_0002/raw/server-to-client-0001.bin#12481-12486+escapeSuccess+c_clearBattleCellInfo"
	classicBattleReliveSourceCapture        = "D:/yzhgame/WOCClient/instances/instance2.staging/tmp/woc-proxy-captures/20260616_215757_020_session_42684/connections/20260616_215819_960_conn_0002/raw/server-to-client-0001.bin#11601+c_doRelive"
	classicBattleReliveRequestCapture       = "D:/yzhgame/WOCClient/instances/instance2.staging/tmp/woc-proxy-captures/20260616_215757_020_session_42684/connections/20260616_215819_960_conn_0002/raw/client-to-server-0001.bin#6170+DoRelive"
	classicBattleReliveMissingItemCapture   = "D:/yzhgame/WOCClient/instances/instance2.staging/tmp/woc-proxy-captures/20260616_215757_020_session_42684/connections/20260616_215819_960_conn_0002/raw/server-to-client-0001.bin#11616+c_Error"
	classicBattleReliveItemName             = "千年灵芝"
	classicBattleReliveMissingItemError     = "用于复活的物品不足"
	classicBattleReliveNeedItemMarkup       = "[i=f_i_千年灵芝^f9e000&24@特殊&25@999&19@如果放在背包里,在副本内死亡后可立即原地复活。&20@灵芝自古以来就被认为是吉祥,富贵,美好,长寿的象征,有 仙草 瑞草之称.民间传说灵芝有起死回生,长生不老之功效.&101@588.png&103@0&104@0&105@&107@&108@0]千年灵芝[/]x1"
)

type classicBattleCellCountPush struct {
	BattleID      string `json:"battleId,omitempty"`
	Count         int    `json:"count"`
	PKWarning     bool   `json:"pkWarning,omitempty"`
	SourceCapture string `json:"sourceCapture,omitempty"`
}

type classicBattleStopCommandPush struct {
	BattleID      string `json:"battleId,omitempty"`
	SourceCapture string `json:"sourceCapture,omitempty"`
}

type classicBattleClearCellInfoPush struct {
	BattleID      string `json:"battleId,omitempty"`
	Handle        string `json:"handle"`
	SourceCapture string `json:"sourceCapture,omitempty"`
}

type classicBattleLoadProgressPush struct {
	BattleID      string `json:"battleId,omitempty"`
	Name          string `json:"name"`
	Progress      int    `json:"progress"`
	SourceCapture string `json:"sourceCapture,omitempty"`
}

type classicBattleLoadProgressRequest struct {
	BattleID string `json:"battleId,omitempty"`
	Progress int    `json:"progress"`
}

type classicBattleRoleReadyRequest struct {
	BattleID string `json:"battleId,omitempty"`
}

type classicBattleRelivePush struct {
	BattleID      string `json:"battleId,omitempty"`
	Ltim          int64  `json:"ltim"`
	NeedItem      string `json:"needItem"`
	SourceCapture string `json:"sourceCapture,omitempty"`
}

type classicBattleDoReliveRequest struct {
	BattleID string `json:"battleId,omitempty"`
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
	buyBackTaken            map[int]bool
	buyBackSoldEntries      []classicTownSourceBuyBackEntry
	currentMailHandle       string
	mailAttachmentTaken     map[string]map[int]bool
}

func handlePacket(store *session.Store, packet protocol.Packet) packetResult {
	return handlePacketWithSession(store, packet, &packetSession{})
}

func handlePacketWithSession(store *session.Store, packet protocol.Packet, socketSession *packetSession) packetResult {
	switch packet.Cmd {
	case cmdHeartbeat:
		return packetResult{handled: true}
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
			result.serverTime = buildClassicTownServerTimePush(time.Now().UnixMilli())
			result.dungeonInstance = syncDungeonInstanceState(store, socketSession, response.Role.MapID)
			result.mapSpecial = buildClassicTownMapSpecialPush(result.dungeonInstance)
			bootstrap := world.BuildTownBootstrap(response.Role, response.PlayerBase)
			filterDefeatedVisibleMonsters(&bootstrap, socketSession)
			applyAcceptedQuestStatesToBootstrap(&bootstrap, store, socketSession)
			result.gameTips = buildClassicTownRoleSelectGameTips(response.Role.MapID, bootstrap)
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
		return packetResult{handled: true}
	case cmdClassicTownActiveRoleReq:
		var request classicTownRoleInteractionRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		if strings.TrimSpace(request.Kind) == "player" {
			return packetResult{handled: true}
		}
		if result, ok := buildClassicTownCollectionResult(store, socketSession, request); ok {
			return result
		}
		if result, ok := buildClassicAuctionOpenResult(request); ok {
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
	case cmdClassicTownCollectionReq:
		var request classicTownCollectionRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicTownCollectionRewardResult(store, socketSession, request)
	case cmdClassicTownCrossRoleReq:
		var request classicTownRoleInteractionRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		log.Printf("[ai-server] classic town CrossRole handle=%s roleId=%s kind=%s mapId=%s", request.Handle, request.RoleID, request.Kind, request.MapID)
		if isClassicTownVisibleMonsterCrossRole(request) {
			return buildClassicBattleStartResult(store, socketSession, battle.StartRequest{
				MapID:               request.MapID,
				MapName:             "m_" + strings.TrimSpace(request.MapID),
				StageFocusX:         0,
				ReturnRoute:         "town-placeholder",
				SourceMonsterHandle: request.Handle,
			})
		}
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
		if isClassicTownInactiveTransportAnswer(request) {
			return packetResult{
				errorMessages: []classicTownErrorPush{{
					Msg:           classicTownInactiveTransportError,
					SourceCapture: classicTownInactiveTransportSourceCapture,
				}},
				handled: true,
			}
		}
		if result, ok := buildClassicTownHealerResult(store, socketSession, request); ok {
			return result
		}
		if result, ok := buildClassicTownItemShopResult(request); ok {
			return result
		}
		if result, ok := buildClassicMailOpenResult(request); ok {
			return result
		}
		if result, ok := buildClassicWarehouseOpenResult(socketSession, request); ok {
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
	case cmdClassicTownGetMapSpecialReq:
		var request classicTownMapSpecialRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		mapID := resolveClassicTownMapSpecialMapID(socketSession, request)
		dungeonInstance := syncDungeonInstanceState(store, socketSession, mapID)
		return packetResult{
			mapSpecial: buildClassicTownMapSpecialPush(dungeonInstance),
			handled:    true,
		}
	case cmdClassicTownMoveRoleReq:
		var request classicTownMoveRoleRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicTownMoveRoleResult(socketSession, request)
	case cmdClassicTownGetSkillListReq:
		return buildClassicTownSkillListResult(store, socketSession)
	case cmdClassicTownGetAbilityCountReq:
		return buildClassicTownAbilityCountResult(store, socketSession)
	case cmdClassicAutoBattleInfoReq:
		var request classicAutoBattleInfoRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicAutoBattleInfoResult(socketSession, request)
	case cmdClassicAutoBattleStartReq:
		var request classicAutoBattleStartRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicAutoBattleStartResult(socketSession, request)
	case cmdClassicAutoBattleStopReq:
		var request classicAutoBattleStopRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicAutoBattleStopResult(socketSession, request)
	case cmdClassicTownGetFastPanelReq:
		return buildClassicTownFastPanelResult(store, socketSession)
	case cmdClassicTownGetBuffsListReq:
		return buildClassicTownBuffListResult(store, socketSession)
	case cmdClassicTownSetFastPanelReq:
		var request classicTownSetFastPanelRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicTownSetFastPanelResult(store, socketSession, request)
	case cmdClassicTownRemoveFastPanelReq:
		var request classicTownRemoveFastPanelRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicTownRemoveFastPanelResult(store, socketSession, request)
	case cmdClassicTownBuySkillReq:
		var request classicTownBuySkillRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicTownBuySkillResult(store, socketSession, request)
	case cmdClassicTownGetBuyBackListReq:
		return buildClassicTownBuyBackListResult(socketSession)
	case cmdClassicTownBuyBackReq:
		var request classicTownBuyBackRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicTownBuyBackResult(store, socketSession, request)
	case cmdClassicTownRemoveSkillReq:
		var request classicTownRemoveSkillRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicTownRemoveSkillResult(store, socketSession, request)
	case cmdClassicTownRemoveBuffReq:
		var request classicTownRemoveBuffRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicTownRemoveBuffResult(store, socketSession, request)
	case cmdClassicTownRemoveABateBuff:
		return buildClassicTownRemoveABateBuffResult(store, socketSession)
	case cmdClassicTownCareStateReq:
		return buildClassicWarehouseCareStateResult(socketSession)
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
	case cmdClassicTownOtherEquipmentReq:
		var request classicTownOtherEquipmentRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicTownOtherEquipmentResult(request)
	case cmdClassicTownContainerMove:
		var request classicTownContainerMoveRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicTownContainerMoveResult(store, socketSession, request)
	case cmdClassicTownFinishingContainerReq:
		var request classicTownContainerRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicTownFinishingContainerResult(store, socketSession, request.Type)
	case cmdClassicTownEquipItemReq:
		var request classicTownEquipItemRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicTownEquipItemResult(store, socketSession, request)
	case cmdClassicTownTryEquipReq:
		var request classicTownTryEquipRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicTownTryEquipResult(store, socketSession, request)
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
	case cmdClassicPetInfoReq:
		var request classicPetInfoRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicPetInfoResult(store, socketSession, request)
	case cmdClassicPetFeedReq:
		var request classicPetFeedRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicPetFeedResult(store, socketSession, request)
	case cmdClassicAuctionListReq:
		var request classicAuctionListRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicAuctionListResult(request)
	case cmdClassicAuctionAddReq:
		var request classicAuctionAddRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicAuctionAddResult(request)
	case cmdClassicMailInfoReq:
		var request classicMailInfoRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicMailInfoResult(socketSession, request)
	case cmdClassicMailSendReq:
		var request classicMailSendRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicMailSendResult(request)
	case cmdClassicDailyRewardInfoReq:
		var request classicDailyRewardInfoRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicDailyRewardInfoResult(socketSession, request)
	case cmdClassicDailyRewardClaimReq:
		var request classicDailyRewardClaimRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicDailyRewardClaimResult(socketSession, request)
	case cmdClassicMaskCodeRefreshReq:
		var request classicMaskCodeRefreshRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicMaskCodeRefreshResult(request)
	case cmdClassicMaskCodeSubmitReq:
		var request classicMaskCodeSubmitRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicMaskCodeSubmitResult(request)
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
	case cmdClassicSocialGetFriendListReq:
		return buildClassicSocialGetFriendListResult(socketSession)
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
	case cmdClassicSocialGetBlackListReq:
		return buildClassicSocialGetBlackListResult(socketSession)
	case cmdClassicSocialTradeReq:
		var request classicSocialTradeRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicSocialTradeRequestResult(socketSession, request)
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
	case cmdClassicTeamInviteReq:
		var request classicTeamInviteRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicTeamInviteResult(socketSession, request)
	case cmdClassicTeamInviteReplyReq:
		var request classicTeamInviteReplyRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicTeamInviteReplyResult(socketSession, request)
	case cmdClassicTeamLeaveReq:
		return buildClassicTeamLeaveResult(socketSession)
	case cmdClassicTeamKickReq:
		var request classicTeamMemberTargetRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicTeamKickResult(socketSession, request)
	case cmdClassicTeamTransferLeaderReq:
		var request classicTeamMemberTargetRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicTeamTransferLeaderResult(socketSession, request)
	case cmdClassicTeamSyncChangeMapReq:
		var request classicTeamSyncChangeMapRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicTeamSyncChangeMapResult(socketSession, request)
	case cmdClassicTeamResetDungeonReq:
		return buildClassicTeamResetDungeonResult(store, socketSession)
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
	case cmdClassicBattleLoadProReq:
		var request classicBattleLoadProgressRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicBattleLoadProgressResult(socketSession, request)
	case cmdClassicBattleRoleReadyReq:
		var request classicBattleRoleReadyRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicBattleRoleReadyResult(socketSession, request)
	case cmdClassicBattleDoReliveReq:
		var request classicBattleDoReliveRequest
		if !decodePayload(packet.Payload, &request) {
			return packetResult{}
		}
		return buildClassicBattleDoReliveResult(store, socketSession, request)
	default:
		return packetResult{}
	}
}

func buildClassicTownRoleSelectGameTips(mapID int, bootstrap world.TownBootstrapSnapshot) []classicTownGameTipPush {
	if mapID != 1 {
		return nil
	}
	if !townBootstrapHasSourceRoleDisplayName(bootstrap, classicTownGameTipNpcTargetName) {
		return nil
	}
	return []classicTownGameTipPush{{
		TipID:      classicTownGameTipNpcTipID,
		Kind:       classicTownGameTipNpcKind,
		TargetName: classicTownGameTipNpcTargetName,
		HTMLText:   classicTownGameTipNpcHTMLText,
		Fields: []string{
			classicTownGameTipNpcTipID,
			classicTownGameTipNpcKind,
			classicTownGameTipNpcTargetName,
			classicTownGameTipNpcHTMLText,
		},
		SourceCapture: classicTownGameTipNpcSourceCapture,
	}}
}

func townBootstrapHasSourceRoleDisplayName(bootstrap world.TownBootstrapSnapshot, displayName string) bool {
	for _, rolePush := range bootstrap.CreateRoles {
		if rolePush.DisplayName == displayName {
			return true
		}
	}
	return false
}

func isClassicTownInactiveTransportAnswer(request classicTownAnswerRequest) bool {
	return strings.TrimSpace(request.Handle) == classicTownInactiveTransportHandle &&
		strings.TrimSpace(request.MsgHandle) == classicTownInactiveTransportMsgHandle &&
		strings.TrimSpace(request.AnswerHandle) == classicTownInactiveTransportAnswerHandle
}

func isClassicTownSkillTeacherRequest(request classicTownAnswerRequest) bool {
	return request.MsgHandle == "10" && (request.Handle == sourceSkillTeacherHandle || request.Handle == guangqingSkillTeacherHandle || request.Handle == baiyuanSkillTeacherHandle)
}

func isClassicTownVisibleMonsterCrossRole(request classicTownRoleInteractionRequest) bool {
	return strings.TrimSpace(request.Kind) == "monster" || strings.TrimSpace(request.RoleID) == "-2"
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

	teamPlan := classicTeamManager.BuildBattleMemberPlan(socketSession.selectedRole.RoleID, request.MapID)
	actors, sharedMembers := classicTeamHub.buildBattleActors(battle.TeamActor{
		Role:       *socketSession.selectedRole,
		PlayerBase: *socketSession.playerBase,
	}, teamPlan.Members, request.MapID)
	runtime, bundle, ok := battle.NewTeamWildBattle(actors, request)
	if !ok {
		log.Printf("[ai-server] classic battle StartBattle ignored unsupported mapId=%s", request.MapID)
		return packetResult{handled: true}
	}
	socketSession.battleRuntime = runtime
	enemyCells := make([]string, 0, len(bundle.Cells))
	for _, cell := range bundle.Cells {
		if cell.Camp == battle.CampEnemy {
			enemyCells = append(enemyCells, cell.Handle+":"+cell.Name)
		}
	}
	log.Printf(
		"[ai-server] classic battle StartBattle battleId=%s roleId=%s mapId=%s sourceMonsterHandle=%s enemies=%v",
		bundle.Start.BattleID,
		socketSession.selectedRole.RoleID,
		request.MapID,
		request.SourceMonsterHandle,
		enemyCells,
	)
	return packetResult{
		battleStart: &bundle.Start,
		battleCells: bundle.Cells,
		battleCellCount: buildClassicBattleCellCountPush(
			bundle.Start.BattleID,
			len(bundle.Cells),
			false,
		),
		battleCommand: &battle.StartCommandPush{
			BattleID:    bundle.StartCommand.BattleID,
			ActorHandle: bundle.StartCommand.ActorHandle,
			Round:       bundle.StartCommand.Round,
			Sequence:    bundle.StartCommand.Sequence,
			Power:       bundle.StartCommand.Power,
		},
		teamBattleStart: &classicTeamBattleStart{
			ActorRoleID: socketSession.selectedRole.RoleID,
			Runtime:     runtime,
			Bundle:      bundle,
			Members:     sharedMembers,
		},
		handled: true,
	}
}

func buildClassicBattleCellCountPush(battleID string, count int, pkWarning bool) *classicBattleCellCountPush {
	if count < 0 {
		count = 0
	}
	return &classicBattleCellCountPush{
		BattleID:      strings.TrimSpace(battleID),
		Count:         count,
		PKWarning:     pkWarning,
		SourceCapture: classicBattleCellCountSourceCapture,
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

	sharedBattle := shouldBroadcastTeamBattle(socketSession)
	beforeTeamMember := team.Member{}
	if sharedBattle {
		beforeTeamMember = classicTeamMemberFromSession(socketSession)
	}
	requiredItemName := classicBattleActionRequiredItemName(request.CommandID)
	requiredItem := session.RoleItem{}
	if requiredItemName != "" {
		item, ok := findClassicBattleRequiredBagItem(store, socketSession, requiredItemName)
		if !ok {
			log.Printf("[ai-server] classic battle battleAction rejected missing required item battleId=%s actor=%s command=%s item=%s", request.BattleID, request.ActorHandle, request.CommandID, requiredItemName)
			return buildClassicBattleActionRejectedRetryResult(socketSession, request, requiredItemName+"不足。")
		}
		requiredItem = item
	}
	result := socketSession.battleRuntime.ProcessAction(request)
	if result.ErrorCode != "" {
		log.Printf("[ai-server] classic battle battleAction rejected battleId=%s actor=%s target=%s error=%s", request.BattleID, request.ActorHandle, request.TargetHandle, result.ErrorCode)
		return buildClassicBattleActionRejectedRetryResult(socketSession, request, "")
	}
	var consumedRequiredItem session.RoleUseItemResult
	if requiredItemName != "" {
		consumedRequiredItem = store.ConsumeRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, requiredItem.Type, requiredItem.Index, 1)
		if !consumedRequiredItem.Found || !consumedRequiredItem.Used {
			log.Printf("[ai-server] classic battle battleAction required item consume failed roleId=%s command=%s item=%s type=%s index=%d error=%s", socketSession.selectedRole.RoleID, request.CommandID, requiredItemName, requiredItem.Type, requiredItem.Index, consumedRequiredItem.ErrorCode)
			return packetResult{handled: true}
		}
		socketSession.selectedRole = &consumedRequiredItem.Role
		socketSession.playerBase = &consumedRequiredItem.PlayerBase
	}
	var roleState *session.RoleState
	var rolePhysique *session.RolePhysique
	var removeRoleHandles []string
	var chatMessages []classicTownChatMessagePush
	if result.Over != nil {
		chatMessages = append(chatMessages, classicBattleOverChatMessages(socketSession, result.Over)...)
		roleState, rolePhysique = finalizeClassicBattleOver(store, socketSession, result.Over.Result)
		socketSession.battleLoot = buildClassicBattleLoot(socketSession, result.Over.Result)
		removeRoleHandles = append(removeRoleHandles, markDefeatedVisibleMonsterFromBattle(store, socketSession, result.Over)...)
		socketSession.battleRuntime = nil
	} else if sharedBattle {
		updatePlayerBaseRoleStateFromBattle(socketSession)
	}
	packet := packetResult{
		battleActions:     result.Actions,
		battleStopCommand: buildClassicBattleStopCommandPush(result.Actions),
		battleBuffs:       result.BuffInfos,
		battleClearBuffs:  result.ClearBuffInfos,
		battleClearCells:  buildClassicBattleClearCellInfoPushes(result.Actions),
		battleCommand:     result.StartCommand,
		battleOver:        result.Over,
		battleRelive:      buildClassicBattleRelivePush(result.Over),
		roleState:         roleState,
		rolePhysique:      rolePhysique,
		chatMessages:      chatMessages,
		removeRoleHandles: removeRoleHandles,
		itemInfos:         make([]classicTownItemInfoPush, 0, 1),
		itemClears:        make([]classicTownItemInfoClearPush, 0, 1),
		handled:           true,
	}
	appendConsumedRequiredBattleItemPushes(&packet, consumedRequiredItem)
	if sharedBattle {
		packet.teamEvents = append(packet.teamEvents, classicTeamMemberSnapshotEventsIfChanged(beforeTeamMember, socketSession)...)
		packet.teamBattleSync = &classicTeamBattleSync{
			ActorRoleID: socketSession.selectedRole.RoleID,
			Result: packetResult{
				battleActions:     packet.battleActions,
				battleStopCommand: packet.battleStopCommand,
				battleBuffs:       packet.battleBuffs,
				battleClearBuffs:  packet.battleClearBuffs,
				battleClearCells:  packet.battleClearCells,
				battleCommand:     packet.battleCommand,
				battleOver:        packet.battleOver,
				battleRelive:      packet.battleRelive,
			},
		}
	}
	return packet
}

func buildClassicBattleStopCommandPush(actions []battle.ActionPush) *classicBattleStopCommandPush {
	if len(actions) == 0 {
		return nil
	}
	return &classicBattleStopCommandPush{
		BattleID:      actions[0].BattleID,
		SourceCapture: classicBattleStopCommandSourceCapture,
	}
}

func buildClassicBattleClearCellInfoPushes(actions []battle.ActionPush) []classicBattleClearCellInfoPush {
	pushes := make([]classicBattleClearCellInfoPush, 0, len(actions))
	for _, action := range actions {
		if strings.TrimSpace(action.SourceActionLabel) != "escapeSuccess" || strings.TrimSpace(action.ActorHandle) == "" {
			continue
		}
		pushes = append(pushes, classicBattleClearCellInfoPush{
			BattleID:      action.BattleID,
			Handle:        strings.TrimSpace(action.ActorHandle),
			SourceCapture: classicBattleClearCellInfoSourceCapture,
		})
	}
	return pushes
}

func buildClassicBattleActionRejectedRetryResult(socketSession *packetSession, request battle.ActionRequest, warning string) packetResult {
	result := packetResult{handled: true}
	if socketSession == nil || socketSession.battleRuntime == nil {
		return result
	}
	runtime := socketSession.battleRuntime
	if runtime.Phase != battle.PhaseCommand ||
		request.BattleID != runtime.BattleID ||
		request.Round != runtime.Round ||
		strings.TrimSpace(request.ActorHandle) != strings.TrimSpace(runtime.ActiveHandle) {
		return result
	}
	actorHandle := strings.TrimSpace(runtime.ActiveHandle)
	if actorHandle == "" {
		actorHandle = strings.TrimSpace(request.ActorHandle)
	}
	result.battleCommand = &battle.StartCommandPush{
		BattleID:    runtime.BattleID,
		ActorHandle: actorHandle,
		Round:       runtime.Round,
		Sequence:    request.Sequence,
		Power:       classicBattleRetryPower(runtime, actorHandle),
	}
	if strings.TrimSpace(warning) != "" {
		result.chatMessages = []classicTownChatMessagePush{classicTownSystemWarningMessage(warning)}
	}
	return result
}

func classicBattleRetryPower(runtime *battle.Runtime, actorHandle string) int {
	if runtime == nil || strings.TrimSpace(actorHandle) == "" {
		return 1
	}
	power := runtime.StoredPower[actorHandle]
	if power <= 0 {
		return 1
	}
	if power > 5 {
		return 5
	}
	return power
}

func classicBattleActionRequiredItemName(commandID string) string {
	switch strings.TrimSpace(commandID) {
	case battle.CommandQiangLiFeiBiao, "强力飞镖":
		return "飞镖"
	case battle.CommandTouDu, "投毒":
		return "毒药"
	case battle.CommandGuanJiaLianShi, "贯甲连矢":
		return "穿甲箭"
	default:
		return ""
	}
}

func findClassicBattleRequiredBagItem(store *session.Store, socketSession *packetSession, name string) (session.RoleItem, bool) {
	if store == nil || socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		return session.RoleItem{}, false
	}
	items, _, ok := store.GetRoleItems(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, classicTownBagContainerType)
	if !ok {
		return session.RoleItem{}, false
	}
	for _, item := range items {
		if strings.TrimSpace(item.Name) == name && item.Count > 0 {
			return item, true
		}
	}
	return session.RoleItem{}, false
}

func appendConsumedRequiredBattleItemPushes(packet *packetResult, useResult session.RoleUseItemResult) {
	if packet == nil || !useResult.Used {
		return
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

	sharedBattle := shouldBroadcastTeamBattle(socketSession)
	beforeTeamMember := team.Member{}
	if sharedBattle {
		beforeTeamMember = classicTeamMemberFromSession(socketSession)
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
	var chatMessages []classicTownChatMessagePush
	if result.Over != nil {
		chatMessages = append(chatMessages, classicBattleOverChatMessages(socketSession, result.Over)...)
		roleState, rolePhysique = finalizeClassicBattleOver(store, socketSession, result.Over.Result)
		socketSession.battleLoot = buildClassicBattleLoot(socketSession, result.Over.Result)
		removeRoleHandles = append(removeRoleHandles, markDefeatedVisibleMonsterFromBattle(store, socketSession, result.Over)...)
		socketSession.battleRuntime = nil
	} else if sharedBattle {
		updatePlayerBaseRoleStateFromBattle(socketSession)
	}

	packet := packetResult{
		battleActions:     result.Actions,
		battleBuffs:       result.BuffInfos,
		battleClearBuffs:  result.ClearBuffInfos,
		battleClearCells:  buildClassicBattleClearCellInfoPushes(result.Actions),
		battleCommand:     result.StartCommand,
		battleOver:        result.Over,
		battleRelive:      buildClassicBattleRelivePush(result.Over),
		roleState:         roleState,
		rolePhysique:      rolePhysique,
		chatMessages:      chatMessages,
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
	if sharedBattle {
		packet.teamEvents = append(packet.teamEvents, classicTeamMemberSnapshotEventsIfChanged(beforeTeamMember, socketSession)...)
		packet.teamBattleSync = &classicTeamBattleSync{
			ActorRoleID: socketSession.selectedRole.RoleID,
			Result:      packetResult{battleActions: packet.battleActions, battleBuffs: packet.battleBuffs, battleClearBuffs: packet.battleClearBuffs, battleClearCells: packet.battleClearCells, battleCommand: packet.battleCommand, battleOver: packet.battleOver, battleRelive: packet.battleRelive},
		}
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

	sharedBattle := shouldBroadcastTeamBattle(socketSession)
	beforeTeamMember := team.Member{}
	if sharedBattle {
		beforeTeamMember = classicTeamMemberFromSession(socketSession)
	}
	result := socketSession.battleRuntime.ProcessPlayOver(request)
	if result.ErrorCode != "" {
		log.Printf("[ai-server] classic battle BattlePlayOver rejected battleId=%s error=%s", request.BattleID, result.ErrorCode)
		return packetResult{handled: true}
	}

	var roleState *session.RoleState
	var rolePhysique *session.RolePhysique
	var removeRoleHandles []string
	var chatMessages []classicTownChatMessagePush
	if result.Over != nil {
		chatMessages = append(chatMessages, classicBattleOverChatMessages(socketSession, result.Over)...)
		roleState, rolePhysique = finalizeClassicBattleOver(store, socketSession, result.Over.Result)
		socketSession.battleLoot = buildClassicBattleLoot(socketSession, result.Over.Result)
		removeRoleHandles = append(removeRoleHandles, markDefeatedVisibleMonsterFromBattle(store, socketSession, result.Over)...)
		socketSession.battleRuntime = nil
	} else if sharedBattle {
		updatePlayerBaseRoleStateFromBattle(socketSession)
	}
	packet := packetResult{
		battleCommand:     result.StartCommand,
		battleOver:        result.Over,
		battleRelive:      buildClassicBattleRelivePush(result.Over),
		roleState:         roleState,
		rolePhysique:      rolePhysique,
		chatMessages:      chatMessages,
		removeRoleHandles: removeRoleHandles,
		handled:           true,
	}
	if sharedBattle {
		packet.teamEvents = append(packet.teamEvents, classicTeamMemberSnapshotEventsIfChanged(beforeTeamMember, socketSession)...)
		packet.teamBattleSync = &classicTeamBattleSync{
			ActorRoleID: socketSession.selectedRole.RoleID,
			Result:      packetResult{battleCommand: packet.battleCommand, battleOver: packet.battleOver, battleRelive: packet.battleRelive},
		}
	}
	return packet
}

func buildClassicBattleRelivePush(over *battle.OverPush) *classicBattleRelivePush {
	if over == nil || over.Result.Winner != battle.CampEnemy || over.Result.Escaped {
		return nil
	}
	return &classicBattleRelivePush{
		BattleID:      over.BattleID,
		Ltim:          time.Now().Add(time.Minute).UnixMilli(),
		NeedItem:      classicBattleReliveNeedItemMarkup,
		SourceCapture: classicBattleReliveSourceCapture,
	}
}

func buildClassicBattleLoadProgressResult(socketSession *packetSession, request classicBattleLoadProgressRequest) packetResult {
	return packetResult{
		battleLoadProgress: buildClassicBattleLoadProgressPush(socketSession, request.BattleID, request.Progress, classicBattleLoadProgressRequestCapture),
		handled:            true,
	}
}

func buildClassicBattleRoleReadyResult(socketSession *packetSession, request classicBattleRoleReadyRequest) packetResult {
	return packetResult{
		battleLoadProgress: buildClassicBattleLoadProgressPush(socketSession, request.BattleID, 100, classicBattleRoleReadyRequestCapture),
		handled:            true,
	}
}

func buildClassicBattleLoadProgressPush(socketSession *packetSession, battleID string, progress int, sourceCapture string) *classicBattleLoadProgressPush {
	name := "battleLoad"
	if socketSession != nil && socketSession.selectedRole != nil {
		if displayName := strings.TrimSpace(socketSession.selectedRole.DisplayName); displayName != "" {
			name = displayName
		}
	}
	if sourceCapture == "" {
		sourceCapture = classicBattleLoadProgressSourceCapture
	}
	return &classicBattleLoadProgressPush{
		BattleID:      strings.TrimSpace(battleID),
		Name:          name,
		Progress:      clampClassicBattleLoadProgress(progress),
		SourceCapture: sourceCapture,
	}
}

func clampClassicBattleLoadProgress(progress int) int {
	if progress < 0 {
		return 0
	}
	if progress > 100 {
		return 100
	}
	return progress
}

func buildClassicBattleDoReliveResult(store *session.Store, socketSession *packetSession, request classicBattleDoReliveRequest) packetResult {
	result := packetResult{handled: true}
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic battle DoRelive ignored without selected role battleId=%s sourceCapture=%s", request.BattleID, classicBattleReliveRequestCapture)
		return result
	}
	if _, ok := findClassicBattleRequiredBagItem(store, socketSession, classicBattleReliveItemName); !ok {
		log.Printf("[ai-server] classic battle DoRelive rejected missing item roleId=%s battleId=%s item=%s sourceCapture=%s", socketSession.selectedRole.RoleID, request.BattleID, classicBattleReliveItemName, classicBattleReliveMissingItemCapture)
		result.errorMessages = []classicTownErrorPush{{
			Msg:           classicBattleReliveMissingItemError,
			SourceCapture: classicBattleReliveMissingItemCapture,
			Partial:       true,
		}}
		return result
	}

	log.Printf("[ai-server] classic battle DoRelive success branch blocked pending capture evidence roleId=%s battleId=%s item=%s", socketSession.selectedRole.RoleID, request.BattleID, classicBattleReliveItemName)
	return result
}

func classicBattleOverChatMessages(socketSession *packetSession, over *battle.OverPush) []classicTownChatMessagePush {
	if socketSession == nil || socketSession.battleRuntime == nil || socketSession.selectedRole == nil || over == nil {
		return nil
	}
	if over.Result.Winner != battle.CampTeam || over.Result.Escaped {
		return nil
	}
	if !classicactivity.IsPointCouponThiefHandleAnyMap(socketSession.battleRuntime.SourceMonsterHandle) {
		return nil
	}
	return []classicTownChatMessagePush{{
		Channel: "system",
		Msg:     "<w>[" + socketSession.selectedRole.DisplayName + "]消灭了[点券盗贼]，幸运的获得点券奖励。",
	}}
}

func shouldBroadcastTeamBattle(socketSession *packetSession) bool {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.battleRuntime == nil {
		return false
	}
	teamCells := 0
	for _, cell := range socketSession.battleRuntime.Cells {
		if cell.Camp == battle.CampTeam {
			teamCells += 1
		}
	}
	if teamCells <= 1 {
		return false
	}
	recipients, ok := classicTeamManager.RecipientsForTeam(socketSession.selectedRole.RoleID)
	return ok && len(recipients) > 1
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
		item.Owner = ""
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
	if containerType == classicMailContainerType {
		return buildClassicMailContainerCapacityResult(socketSession)
	}
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
	if containerType == classicMailContainerType {
		return buildClassicMailItemListResult(socketSession)
	}
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

func isClassicTownIndexedContainerMoveSupported(sourceType string, targetType string) bool {
	if sourceType == classicTownBagContainerType && targetType == classicTownBagContainerType {
		return true
	}
	if sourceType == classicTownBagContainerType && targetType == classicWarehouseContainerType {
		return true
	}
	if sourceType == "商城" && (targetType == classicTownBagContainerType || targetType == "商城") {
		return true
	}
	if sourceType == classicWarehouseContainerType && targetType == classicTownBagContainerType {
		return true
	}
	if sourceType == classicWarehouseContainerType && targetType == classicWarehouseContainerType {
		return true
	}
	return false
}

func buildClassicTownContainerMoveResult(store *session.Store, socketSession *packetSession, request classicTownContainerMoveRequest) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic town ContainerMove ignored without selected role source=%s target=%s", request.SourceType, request.TargetType)
		return packetResult{handled: true}
	}

	sourceType := strings.TrimSpace(request.SourceType)
	targetType := strings.TrimSpace(request.TargetType)
	if sourceType == classicMailContainerType && targetType == classicTownBagContainerType {
		return buildClassicMailContainerMoveResult(store, socketSession)
	}
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
		moveAttempts := 0
		moveFailures := 0
		for _, item := range socketSession.battleLoot {
			if item.Type != classicBattleLootType || (len(nameFilter) > 0 && !nameFilter[item.Name]) {
				remaining = append(remaining, item)
				continue
			}

			moveAttempts += 1
			moved := item
			moved.Type = "背包"
			moved.Index = -1
			granted, ok := store.GrantRoleItem(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, moved)
			if !ok {
				moveFailures += 1
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
		chatMessages := []classicTownChatMessagePush{}
		if moveFailures > 0 {
			message := "背包空间不足，部分战利品未能放入背包。"
			if moveFailures == moveAttempts {
				message = "背包空间不足，战利品未能放入背包。"
			}
			chatMessages = append(chatMessages, classicTownSystemWarningMessage(message))
		}

		return packetResult{
			containerCap: &classicTownContainerCapacityPush{
				Handle:   socketSession.selectedRole.RoleID,
				Type:     classicBattleLootType,
				Capacity: classicBattleLootCap,
				OpenType: "",
			},
			chatMessages: chatMessages,
			itemInfos:    itemInfos,
			itemClears:   itemClears,
			handled:      true,
		}
	}
	if request.SourceIndex != nil && request.TargetIndex != nil && isClassicTownIndexedContainerMoveSupported(sourceType, targetType) {
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
			log.Printf("[ai-server] classic town ContainerMove item noop roleId=%s source=%s[%v] target=%s[%v] error=%s", socketSession.selectedRole.RoleID, sourceType, request.SourceIndex, targetType, request.TargetIndex, moveResult.ErrorCode)
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

func buildClassicTownFinishingContainerResult(store *session.Store, socketSession *packetSession, containerType string) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic town FinishingContainer ignored without selected role type=%s", containerType)
		return packetResult{handled: true}
	}

	containerType = strings.TrimSpace(containerType)
	if containerType != classicTownBagContainerType {
		log.Printf("[ai-server] classic town FinishingContainer ignored unsupported type=%s", containerType)
		return packetResult{handled: true}
	}

	finishResult := store.FinishRoleContainer(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, containerType)
	if !finishResult.Found {
		log.Printf("[ai-server] classic town FinishingContainer role missing roleId=%s type=%s", socketSession.selectedRole.RoleID, containerType)
		return packetResult{handled: true}
	}

	result := packetResult{
		itemInfos:  make([]classicTownItemInfoPush, 0, len(finishResult.UpdatedItems)),
		itemClears: make([]classicTownItemInfoClearPush, 0, len(finishResult.ClearedItems)),
		handled:    true,
	}
	for _, clear := range finishResult.ClearedItems {
		result.itemClears = append(result.itemClears, classicTownItemInfoClearPush{
			Handle: socketSession.selectedRole.RoleID,
			Type:   clear.Type,
			Index:  clear.Index,
		})
	}
	for _, item := range finishResult.UpdatedItems {
		item.Handle = socketSession.selectedRole.RoleID
		result.itemInfos = append(result.itemInfos, classicTownItemInfoPushFromRoleItem(item))
	}
	return result
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
	itemInfos := make([]classicTownItemInfoPush, 0, 1+len(equipResult.UpdatedItems))
	itemInfos = append(itemInfos, classicTownItemInfoPushFromRoleItem(equippedItem))
	for _, updatedItem := range equipResult.UpdatedItems {
		updatedItem.Handle = equipResult.Role.RoleID
		itemInfos = append(itemInfos, classicTownItemInfoPushFromRoleItem(updatedItem))
	}
	result := packetResult{
		createPlayer: buildClassicTownCreatePlayerPush(equipResult.Role, equipResult.PlayerBase),
		itemInfos:    itemInfos,
		itemClears:   make([]classicTownItemInfoClearPush, 0, len(equipResult.ClearedItems)),
		handled:      true,
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

func buildClassicTownTryEquipResult(store *session.Store, socketSession *packetSession, request classicTownTryEquipRequest) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic town TryEquip ignored without selected role item=%s", request.Name)
		return packetResult{handled: true}
	}

	previewResult := store.PreviewTryEquip(
		socketSession.playerBase.PlayerID,
		socketSession.selectedRole.RoleID,
		request.Name,
	)
	if !previewResult.Found {
		log.Printf("[ai-server] classic town TryEquip ignored missing role roleId=%s item=%s", socketSession.selectedRole.RoleID, request.Name)
		return packetResult{handled: true}
	}
	if !previewResult.Previewed {
		log.Printf("[ai-server] classic town TryEquip rejected roleId=%s item=%s error=%s", socketSession.selectedRole.RoleID, request.Name, previewResult.ErrorCode)
		return packetResult{handled: true}
	}

	return packetResult{
		tryEquip: &classicTownTryEquipPush{
			SourceQuery:   previewResult.SourceQuery,
			ItemName:      previewResult.Item.Name,
			SourceCapture: "TryEquip(231)->c_tryEquip(50092)",
		},
		handled: true,
	}
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
		if useResult.ErrorCode == "item_level_too_low" {
			return packetResult{
				errorMessages: []classicTownErrorPush{{
					Msg:           classicTownActiveItemLevel5GiftBoxError,
					SourceCapture: classicTownActiveItemLevel5GiftBoxSourceCapture,
				}},
				handled: true,
			}
		}
		if useResult.ErrorCode == "level10_gift_box_bag_full" {
			result := packetResult{
				errorMessages: []classicTownErrorPush{{
					Msg:           classicTownActiveItemLevel10GiftBoxFullError,
					SourceCapture: classicTownActiveItemLevel10GiftBoxSourceCapture,
				}},
				itemInfos: make([]classicTownItemInfoPush, 0, len(useResult.UpdatedItems)),
				handled:   true,
			}
			for _, item := range useResult.UpdatedItems {
				item.Handle = useResult.Role.RoleID
				result.itemInfos = append(result.itemInfos, classicTownItemInfoPushFromRoleItem(item))
			}
			return result
		}
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
	if useResult.Item.Name == classicTownActiveItemLevel1GiftBoxName {
		result.chatMessages = append(result.chatMessages, classicTownLevel1GiftBoxRewardMessages()...)
	}
	if useResult.Item.Name == classicTownActiveItemLevel5GiftBoxName {
		result.chatMessages = append(result.chatMessages, classicTownLevel5GiftBoxRewardMessages()...)
	}
	if useResult.Item.Name == classicTownActiveItemLevel10GiftBoxName {
		result.chatMessages = append(result.chatMessages, classicTownLevel10GiftBoxRewardMessages()...)
	}
	if useResult.Item.Name == classicTownActiveItemBagCapacityPatchName && useResult.ContainerCapacity > 0 {
		result.chatMessages = append(result.chatMessages, classicTownBagCapacityPatchMessage(useResult.ContainerCapacity))
	}
	if useResult.ContainerType != "" && useResult.ContainerCapacity > 0 {
		result.containerCap = &classicTownContainerCapacityPush{
			Handle:   useResult.Role.RoleID,
			Type:     useResult.ContainerType,
			Capacity: useResult.ContainerCapacity,
		}
	}
	if useResult.Item.Name == classicTownSilverIngotExchangeItemName {
		result.chatMessages = append(result.chatMessages, classicTownSilverIngotExchangeRewardMessage())
	}
	if useResult.Item.Name == classicTownCopperExchangeItemName {
		result.chatMessages = append(result.chatMessages, classicTownCopperExchangeRewardMessage())
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
			classicTownLearnedSkillMessage(*useResult.LearnedSkill),
		))
	}
	if useResult.TownBuff != nil {
		result.townBuffs = []classicTownBuffInfoPush{
			classicTownBuffInfoPushFromRoleTownBuff(*useResult.TownBuff),
		}
		if message := classicTownActiveItemTownBuffMessage(*useResult.TownBuff, useResult.Item.Name); message != "" {
			result.chatMessages = append(result.chatMessages, classicTownSystemChatMessage(message))
		}
	}
	if useResult.Equipped {
		result.createPlayer = buildClassicTownCreatePlayerPush(useResult.Role, useResult.PlayerBase)
		result.rolePhysique = useResult.PlayerBase.RolePhysique
	}
	if useResult.RoleStateChanged {
		result.roleState = useResult.PlayerBase.RoleState
	}
	return result
}

func classicTownLearnedSkillMessage(skill session.RoleSkill) string {
	if skill.Name == "\u6b66\u5668\u5a34\u719f" && skill.Level == 1 {
		return "\u4e60\u5f97Lv1[\u6b66\u5668\u5a34\u719f]"
	}
	return "\u4e60\u5f97\u3010" + skill.Name + "\u3011Lv." + strconv.Itoa(skill.Level)
}

func classicTownActiveItemTownBuffMessage(buff session.RoleTownBuff, sourceItemName string) string {
	switch buff.Name {
	case "\u907f\u602a":
		return "\u4f7f\u7528\u4e86\u907f\u602a\u7b26"
	case classicTownInitialExperienceBoostName:
		switch sourceItemName {
		case classicTownInitialExperienceCardName:
			return classicTownInitialExperienceBoostMessage
		case classicTownAdvancedExperienceCardName:
			return classicTownAdvancedExperienceBoostMessage
		default:
			return ""
		}
	default:
		return ""
	}
}

func classicTownLevel1GiftBoxRewardMessages() []classicTownChatMessagePush {
	return []classicTownChatMessagePush{
		classicTownSystemChatMessage("\u83b7\u5f97\u7269\u54c1:[\u0035\u7ea7\u793c\u76d2]x1"),
		classicTownSystemChatMessage("\u83b7\u5f97\u7269\u54c1:[\u004c\u907f\u602a\u7b26]x3"),
		classicTownSystemChatMessage("\u83b7\u5f97\u7269\u54c1:[\u004c\u767e\u5e74\u4eba\u53c2\u679c]x1"),
		classicTownSystemChatMessage("\u83b7\u5f97\u7269\u54c1:[\u004c\u767e\u5e74\u87e0\u6843]x1"),
	}
}

func classicTownLevel5GiftBoxRewardMessages() []classicTownChatMessagePush {
	return []classicTownChatMessagePush{
		classicTownSystemChatMessage("\u83b7\u5f97\u7269\u54c1:[\u004c\u521d\u9636\u7ecf\u9a8c\u5361]x1"),
		classicTownSystemChatMessage("\u83b7\u5f97\u7269\u54c1:[\u004c\u82b1\u5377]x2"),
		classicTownSystemChatMessage("\u83b7\u5f97\u7269\u54c1:[\u004c\u56de\u57ce\u5492]x3"),
		classicTownSystemChatMessage("\u83b7\u5f97\u7269\u54c1:[\u0031\u0030\u7ea7\u793c\u76d2]x1"),
	}
}

func classicTownLevel10GiftBoxRewardMessages() []classicTownChatMessagePush {
	return []classicTownChatMessagePush{
		classicTownSystemChatMessage("\u83b7\u5f97\u7269\u54c1:[\u004c\u521d\u9636\u7ecf\u9a8c\u5361]x1"),
		classicTownSystemChatMessage("\u83b7\u5f97\u7269\u54c1:[\u004c\u82b1\u5377]x3"),
		classicTownSystemChatMessage("\u83b7\u5f97\u7269\u54c1:[\u004c\u80cc\u5305\u8865\u4e01]x1"),
		classicTownSystemChatMessage("\u83b7\u5f97\u7269\u54c1:[\u0031\u0035\u7ea7\u793c\u76d2]x1"),
	}
}

func classicTownBagCapacityPatchMessage(capacity int) classicTownChatMessagePush {
	return classicTownSystemChatMessage("\u4f60\u7684\u80cc\u5305\u6269\u5927\u81f3" + strconv.Itoa(capacity) + "\u683c")
}

func classicTownSilverIngotExchangeRewardMessage() classicTownChatMessagePush {
	return classicTownSystemChatMessage("\u83b7\u5f97\u7269\u54c1:[\u94dc\u94b1]x1000")
}

func classicTownCopperExchangeRewardMessage() classicTownChatMessagePush {
	return classicTownSystemChatMessage("\u83b7\u5f97\u7269\u54c1:[\u94f6\u5143\u5b9d]x1")
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
	soldBuyBackEntry := classicTownAppendSoldBuyBackEntry(socketSession, saleResult.Item, saleCount, saleResult.Amount)
	result.buyBackRefresh = &classicTownBuyBackRefreshPush{
		SourceCapture: classicTownBuyBackRefreshSourceCapture,
		Partial:       true,
	}
	result.buyBackInfos = classicTownBuyBackInfoPushes(
		saleResult.Role.RoleID,
		socketSession.buyBackTaken,
		socketSession.buyBackSoldEntries,
	)
	log.Printf("[ai-server] classic town SaleItem roleId=%s shopId=%s type=%s index=%d count=%d item=%s amount=%d", saleResult.Role.RoleID, request.ShopID, request.Type, request.Index, request.Count, saleResult.Item.Name, saleResult.Amount)
	log.Printf("[ai-server] classic town SaleItem appended buyBack roleId=%s buyBackIndex=%d item=%s count=%d price=%d", saleResult.Role.RoleID, soldBuyBackEntry.Index, soldBuyBackEntry.Name, soldBuyBackEntry.Count, soldBuyBackEntry.Price)
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
	if channel == "team" {
		chatMessages, recipients, ok := buildClassicTeamChatEvents(socketSession, push)
		if !ok {
			return packetResult{
				chatMessages: chatMessages,
				handled:      true,
			}
		}
		return packetResult{
			chatBroadcasts: []classicTownChatBroadcast{{Recipients: recipients, Message: push}},
			handled:        true,
		}
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
		"4710542615621525",
		"6360542618722932":
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

func buildClassicTownServerTimePush(serverTime int64) *classicTownServerTimePush {
	return &classicTownServerTimePush{
		ServerTime:    serverTime,
		SourceCapture: classicTownServerTimeSourceCapture,
	}
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
		if result.ErrorCode == "no_point" {
			return packetResult{
				errorMessages: []classicTownErrorPush{{
					Msg:           classicTownAddPointNoPointError,
					SourceCapture: classicTownAddPointNoPointSourceCapture,
				}},
				handled: true,
			}
		}
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

func buildClassicTownAbilityCountResult(store *session.Store, socketSession *packetSession) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic town GetAbilityCount ignored without selected role")
		return packetResult{handled: true}
	}

	_, cap, ok := store.GetRoleSkills(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID)
	if !ok {
		log.Printf("[ai-server] classic town GetAbilityCount ignored missing role roleId=%s", socketSession.selectedRole.RoleID)
		return packetResult{handled: true}
	}

	return packetResult{
		abilityCount: &classicTownAbilityCountPush{
			Handle: socketSession.selectedRole.RoleID,
			Count:  cap,
		},
		handled: true,
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

func buildClassicTownBuyBackListResult(socketSession *packetSession) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil {
		log.Printf("[ai-server] classic town GetBuyBackList ignored without selected role")
		return packetResult{handled: true}
	}

	return packetResult{
		buyBackRefresh: &classicTownBuyBackRefreshPush{
			SourceCapture: classicTownBuyBackRefreshSourceCapture,
			Partial:       true,
		},
		buyBackInfos: classicTownBuyBackInfoPushes(
			socketSession.selectedRole.RoleID,
			socketSession.buyBackTaken,
			socketSession.buyBackSoldEntries,
		),
		handled: true,
	}
}

func buildClassicTownBuyBackResult(
	store *session.Store,
	socketSession *packetSession,
	request classicTownBuyBackRequest,
) packetResult {
	if store == nil || socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		log.Printf("[ai-server] classic town BuyBack ignored without selected role index=%d", request.Index)
		return packetResult{handled: true}
	}

	entry, ok := findClassicTownSourceBuyBackEntry(request.Index, socketSession.buyBackSoldEntries)
	if !ok || (socketSession.buyBackTaken != nil && socketSession.buyBackTaken[request.Index]) {
		log.Printf("[ai-server] classic town BuyBack ignored missing source entry roleId=%s index=%d", socketSession.selectedRole.RoleID, request.Index)
		return packetResult{
			buyBackRefresh: &classicTownBuyBackRefreshPush{
				SourceCapture: classicTownBuyBackRefreshSourceCapture,
				Partial:       true,
			},
			buyBackInfos: classicTownBuyBackInfoPushes(
				socketSession.selectedRole.RoleID,
				socketSession.buyBackTaken,
				socketSession.buyBackSoldEntries,
			),
			handled: true,
		}
	}

	purchase := store.PurchaseRoleItem(
		socketSession.playerBase.PlayerID,
		socketSession.selectedRole.RoleID,
		classicTownSourceBuyBackEntryToRoleItem(entry),
		classicTownSourceBuyBackRequirements(entry),
	)
	if !purchase.Found {
		log.Printf("[ai-server] classic town BuyBack ignored missing role roleId=%s index=%d", socketSession.selectedRole.RoleID, request.Index)
		return packetResult{handled: true}
	}

	socketSession.selectedRole = &purchase.Role
	socketSession.playerBase = &purchase.PlayerBase
	result := packetResult{
		currencyPush: buildClassicTownCurrencyPush(
			purchase.Role.RoleID,
			purchase.Currencies,
		),
		buyBackRefresh: &classicTownBuyBackRefreshPush{
			SourceCapture: classicTownBuyBackRefreshSourceCapture,
			Partial:       true,
		},
		itemInfos:  make([]classicTownItemInfoPush, 0, 1+len(purchase.Consumed)),
		itemClears: make([]classicTownItemInfoClearPush, 0, len(purchase.ClearedItems)),
		handled:    true,
	}
	if purchase.Purchased {
		if socketSession.buyBackTaken == nil {
			socketSession.buyBackTaken = make(map[int]bool)
		}
		socketSession.buyBackTaken[entry.Index] = true
		grantedItem := purchase.Item
		grantedItem.Handle = purchase.Role.RoleID
		result.itemInfos = append(result.itemInfos, classicTownItemInfoPushFromRoleItem(grantedItem))
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
		result.chatMessages = append(result.chatMessages, classicTownSystemChatMessage("回购了【"+entry.Name+"】x"+strconv.Itoa(entry.Count)+"。"))
		log.Printf("[ai-server] classic town BuyBack purchased roleId=%s index=%d item=%s count=%d price=%d", purchase.Role.RoleID, entry.Index, entry.Name, entry.Count, entry.Price)
	} else {
		result.chatMessages = append(result.chatMessages, classicTownSystemWarningMessage(purchase.ErrorMessage))
		log.Printf("[ai-server] classic town BuyBack rejected roleId=%s index=%d item=%s error=%s", purchase.Role.RoleID, request.Index, entry.Name, purchase.ErrorCode)
	}
	result.buyBackInfos = classicTownBuyBackInfoPushes(
		purchase.Role.RoleID,
		socketSession.buyBackTaken,
		socketSession.buyBackSoldEntries,
	)
	return result
}

func buildClassicTownMoveRoleResult(socketSession *packetSession, request classicTownMoveRoleRequest) packetResult {
	if socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		return packetResult{handled: true}
	}
	roleID := socketSession.selectedRole.RoleID
	if roleID == "" {
		return packetResult{handled: true}
	}
	mapID := socketSession.playerBase.MapID
	if mapID <= 0 {
		mapID = socketSession.selectedRole.MapID
	}
	if request.MapID != "" {
		requestMapID, err := strconv.Atoi(strings.TrimSpace(request.MapID))
		if err != nil || requestMapID != mapID {
			return packetResult{handled: true}
		}
	}

	push := world.RoleMovePush{
		Handle: roleID,
		Type:   normalizeClassicTownMoveRoleType(request.Type),
		X:      request.X,
		Y:      request.Y,
		Z:      request.Z,
		TX:     request.TX,
		TY:     request.TY,
		TZ:     request.TZ,
		MapID:  strconv.Itoa(mapID),
	}
	worldSceneHub.updatePosition(roleID, mapID, world.SpawnPoint{X: request.X, Y: request.Y})
	return packetResult{
		moveRole: &push,
		handled:  true,
	}
}

func normalizeClassicTownMoveRoleType(value string) string {
	switch strings.TrimSpace(value) {
	case "Run":
		return "Run"
	case "Flash":
		return "Flash"
	default:
		return "Move"
	}
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
	fromMapID := socketSession.playerBase.MapID
	if fromMapID <= 0 {
		fromMapID = socketSession.selectedRole.MapID
	}
	syncPlan := classicTeamManager.BuildSyncTransferPlan(socketSession.selectedRole.RoleID, strconv.Itoa(fromMapID))
	if syncPlan.Enabled && syncPlan.ErrorMessage == "" {
		if _, isDungeonTarget := world.DungeonInstanceKeyForMapID(mapID); isDungeonTarget {
			if warningMessage, ok := classicTeamHub.preflightDungeonSyncTransfer(store, syncPlan.Members, mapID); !ok {
				return packetResult{
					chatMessages: []classicTownChatMessagePush{classicTownSystemWarningMessage(warningMessage)},
					handled:      true,
				}
			}
		}
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
	applyAcceptedQuestStatesToBootstrap(&bootstrap, store, socketSession)
	log.Printf("[ai-server] classic town transfer mapId=%s x=%d y=%d roleId=%s", mapIDText, spawn.X, spawn.Y, role.RoleID)
	var teamSyncTransfer *classicTeamSyncTransfer
	if syncPlan.Enabled {
		if syncPlan.ErrorMessage != "" {
			entryResult.chatMessages = append(entryResult.chatMessages, classicTownSystemWarningMessage(syncPlan.ErrorMessage))
		} else if len(syncPlan.Members) > 0 {
			teamSyncTransfer = &classicTeamSyncTransfer{
				ActorRoleID: role.RoleID,
				FromMapID:   fromMapID,
				TargetMapID: mapID,
				Spawn:       spawn,
				Members:     syncPlan.Members,
			}
		}
	}
	return packetResult{
		townBootstrap:          &bootstrap,
		dungeonInstance:        dungeonInstance,
		mapSpecial:             buildClassicTownMapSpecialPush(dungeonInstance),
		itemInfos:              entryResult.itemInfos,
		itemClears:             entryResult.itemClears,
		chatMessages:           entryResult.chatMessages,
		teamSyncTransfer:       teamSyncTransfer,
		sceneTransferFromMapID: fromMapID,
		sceneTransferSpawn:     spawn,
		handled:                true,
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

func checkDungeonEntryTicketIfNeeded(store *session.Store, socketSession *packetSession, targetMapID int) (string, bool) {
	if store == nil || socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		return "队员连接状态异常，队伍同步取消。", false
	}
	targetInstanceKey, ok := world.DungeonInstanceKeyForMapID(targetMapID)
	if !ok {
		return "", true
	}
	rule, ok := dungeonEntryRuleForInstance(targetInstanceKey)
	if !ok || rule.ConsumePolicy == dungeonEntryConsumeNone || rule.TicketName == "" {
		return "", true
	}
	currentMapID := socketSession.selectedRole.MapID
	currentInstanceKey, currentInDungeon := world.DungeonInstanceKeyForMapID(currentMapID)
	if currentInDungeon && currentInstanceKey == targetInstanceKey {
		return "", true
	}
	if _, ok := store.GetRoleDungeonInstance(socketSession.playerBase.PlayerID, socketSession.selectedRole.RoleID, targetInstanceKey); ok {
		return "", true
	}
	ticket, ok := findRoleTicketItem(store, socketSession, rule.TicketName)
	if !ok || ticket.Count < rule.TicketCount {
		return "队员【" + socketSession.selectedRole.DisplayName + "】进入" + dungeonInstanceDisplayName(targetInstanceKey) + "需要" + rule.TicketName + "x" + strconv.Itoa(rule.TicketCount) + "，队伍同步取消。", false
	}
	return "", true
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
	case session.DungeonInstanceShihuku:
		return dungeonEntryRule{
			InstanceKey:   instanceKey,
			TicketName:    "狮虎窟通行证",
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

func inactiveDungeonInstancePush(instanceKey string, mapID int) *classicTownDungeonInstancePush {
	return &classicTownDungeonInstancePush{
		Key:             instanceKey,
		DisplayName:     dungeonInstanceDisplayName(instanceKey),
		MapID:           strconv.Itoa(mapID),
		Active:          false,
		DurationSeconds: session.DungeonInstanceTTLSeconds(),
	}
}

func resolveClassicTownMapSpecialMapID(socketSession *packetSession, request classicTownMapSpecialRequest) int {
	if mapID, err := strconv.Atoi(strings.TrimSpace(request.MapID)); err == nil && mapID > 0 {
		return mapID
	}
	if mapID, err := strconv.Atoi(strings.TrimSpace(request.ID)); err == nil && mapID > 0 {
		return mapID
	}
	if socketSession != nil && socketSession.selectedRole != nil {
		return socketSession.selectedRole.MapID
	}
	return 0
}

func buildClassicTownMapSpecialPush(dungeonInstance *classicTownDungeonInstancePush) *classicTownMapSpecialPush {
	if dungeonInstance == nil || !dungeonInstance.Active || dungeonInstance.ExpiresAtUnix <= 0 {
		return &classicTownMapSpecialPush{
			LastTime:      0,
			SourceCapture: classicTownMapSpecialLastTimeSourceCapture,
		}
	}
	lastTime := dungeonInstance.ExpiresAtUnix * 1000
	return &classicTownMapSpecialPush{
		LastTime:      lastTime,
		Entries:       []string{"lastTime:" + strconv.FormatInt(lastTime, 10)},
		SourceCapture: classicTownMapSpecialLastTimeSourceCapture,
	}
}

func syncDungeonInstanceState(store *session.Store, socketSession *packetSession, mapID int) *classicTownDungeonInstancePush {
	if store == nil || socketSession == nil || socketSession.selectedRole == nil || socketSession.playerBase == nil {
		return nil
	}
	instanceKey, ok := world.DungeonInstanceKeyForMapID(mapID)
	if !ok {
		setDefeatedVisibleMonsterHandles(socketSession, keepPointCouponThiefDefeatedHandles(socketSession))
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

func keepPointCouponThiefDefeatedHandles(socketSession *packetSession) []string {
	if socketSession == nil || len(socketSession.defeatedVisibleMonsters) == 0 {
		return nil
	}
	handles := make([]string, 0, len(socketSession.defeatedVisibleMonsters))
	for handle := range socketSession.defeatedVisibleMonsters {
		if classicactivity.IsPointCouponThiefHandleAnyMap(handle) {
			handles = append(handles, handle)
		}
	}
	return handles
}

func dungeonInstanceDisplayName(key string) string {
	switch key {
	case session.DungeonInstanceShuiliandong:
		return "水帘洞"
	case session.DungeonInstanceHuangfengzhai:
		return "黄风寨"
	case session.DungeonInstanceFeixiandong:
		return "飞仙洞"
	case session.DungeonInstanceShihuku:
		return "狮虎窟"
	default:
		return "副本"
	}
}
