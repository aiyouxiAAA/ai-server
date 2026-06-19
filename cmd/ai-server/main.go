package main

import (
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"

	"ai-server/internal/mall"
	"ai-server/internal/protocol"
	"ai-server/internal/session"
	"github.com/gorilla/websocket"
)

const (
	cmdHeartbeat          = 1000
	cmdAuthLoginRequest   = 1001
	cmdAuthLoginResponse  = 1002
	cmdRoleListRequest    = 1011
	cmdRoleListResponse   = 1012
	cmdRoleCreateRequest  = 1013
	cmdRoleCreateResponse = 1014
	cmdRoleSelectRequest  = 1015
	cmdRoleSelectResponse = 1016
	cmdRoleRemoveRequest  = 1017
	cmdRoleRemoveResponse = 1018

	cmdClassicTownLoadMapPush       = 1101
	cmdClassicTownCreatePlayerPush  = 1102
	cmdClassicTownCreateRolePush    = 1103
	cmdClassicTownRemoveRolePush    = 1104
	cmdClassicTownQuestStatePush    = 1105
	cmdClassicTownAnswerSpeakPush   = 1106
	cmdClassicTownSkillInfoPush     = 1107
	cmdClassicTownSkillCapPush      = 1108
	cmdClassicTownSkillShopPush     = 1109
	cmdClassicTownCurrencyPush      = 1110
	cmdClassicTownTargetRoleReq     = 1111
	cmdClassicTownActiveRoleReq     = 1112
	cmdClassicTownAnswerReq         = 1113
	cmdClassicTownTransferReq       = 1114
	cmdClassicTownCrossRoleReq      = 1115
	cmdClassicTownGetSkillListReq   = 1116
	cmdClassicTownBuySkillReq       = 1117
	cmdClassicTownBuySkillResult    = 1118
	cmdClassicTownGetCapacityReq    = 1119
	cmdClassicTownGetItemListReq    = 1120
	cmdClassicTownCapacityPush      = 1121
	cmdClassicTownItemInfoPush      = 1122
	cmdClassicTownItemInfoClear     = 1123
	cmdClassicTownEquipItemReq      = 1124
	cmdClassicTownFastPanelPush     = 1125
	cmdClassicTownGetFastPanelReq   = 1126
	cmdClassicTownSetFastPanelReq   = 1180
	cmdClassicTownGetQuestLogReq    = 1127
	cmdClassicTownQuestInfoPush     = 1128
	cmdClassicTownClearQuestInfo    = 1129
	cmdClassicTownRemoveQuestReq    = 1130
	cmdClassicTownRoleStatePush     = 1131
	cmdClassicTownRolePhysiquePush  = 1132
	cmdClassicTownContainerMove     = 1133
	cmdClassicTownAddPointReq       = 1134
	cmdClassicTownActiveItemReq     = 1135
	cmdClassicTownClearSkillInfo    = 1136
	cmdClassicTownRemoveSkillReq    = 1137
	cmdClassicTownChatMessagePush   = 1138
	cmdClassicTownChatSendReq       = 1139
	cmdClassicTownDestroyItemReq    = 1181
	cmdClassicTownSaleItemReq       = 1182
	cmdClassicTownMoveRoleReq       = 1184
	cmdClassicTownMoveRolePush      = 1185
	cmdClassicSocialFriendInfo      = 1140
	cmdClassicSocialClearFriend     = 1141
	cmdClassicSocialBlackListInfo   = 1142
	cmdClassicSocialClearBlackList  = 1143
	cmdClassicSocialEnemyInfo       = 1144
	cmdClassicSocialClearEnemy      = 1145
	cmdClassicSocialAddFriendReq    = 1146
	cmdClassicSocialRemoveFriend    = 1147
	cmdClassicSocialAddBlackReq     = 1148
	cmdClassicSocialRemoveBlack     = 1149
	cmdClassicGuildInfoReq          = 1150
	cmdClassicGuildInfoPush         = 1151
	cmdClassicGuildMemberPush       = 1152
	cmdClassicGuildMemberClear      = 1153
	cmdClassicGuildAuthPush         = 1154
	cmdClassicGuildNoticePush       = 1155
	cmdClassicGuildCreateReq        = 1156
	cmdClassicGuildCreateResult     = 1157
	cmdClassicGuildLeaveReq         = 1158
	cmdClassicGuildKickReq          = 1159
	cmdClassicGuildDismissReq       = 1160
	cmdClassicGuildNoticeUpdateReq  = 1161
	cmdClassicTownDungeonInstance   = 1162
	cmdClassicTeamInviteReq         = 1163
	cmdClassicTeamInvitePush        = 1164
	cmdClassicTeamInviteReplyReq    = 1165
	cmdClassicTeamLeaveReq          = 1166
	cmdClassicTeamKickReq           = 1167
	cmdClassicTeamTransferLeaderReq = 1168
	cmdClassicTeamSyncChangeMapReq  = 1169
	cmdClassicMallCategoryListReq   = 1170
	cmdClassicMallCategoryListPush  = 1171
	cmdClassicMallSearchCountReq    = 1172
	cmdClassicMallSearchCountPush   = 1173
	cmdClassicMallSearchPageReq     = 1174
	cmdClassicMallSearchPagePush    = 1175
	cmdClassicMallCurrencyPush      = 1176
	cmdClassicMallPurchaseReq       = 1177
	cmdClassicMallPurchaseResult    = 1178
	cmdClassicMallPaymentDisabled   = 1179
	cmdClassicTeamResetDungeonReq   = 1183
	cmdClassicTeamMemberPush        = 1190
	cmdClassicTeamMemberClearPush   = 1191
	cmdClassicTeamClearPush         = 1192
	cmdClassicTeamInfoPush          = 1193
	cmdClassicTeamResultPush        = 1194

	cmdClassicBattleStartReq      = 3000
	cmdClassicBattleStartPush     = 3001
	cmdClassicBattleCellInfoPush  = 3002
	cmdClassicBattleStartCommand  = 3003
	cmdClassicBattleActionReq     = 3004
	cmdClassicBattleActionPush    = 3005
	cmdClassicBattleOverPush      = 3006
	cmdClassicBattleBuffInfoPush  = 3008
	cmdClassicBattleClearBuffInfo = 3009
	cmdClassicBattleActiveItemReq = 3011
	cmdClassicBattlePlayOverReq   = 3012
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(_ *http.Request) bool {
		return true
	},
}

func main() {
	store, err := session.NewPersistentStore(filepath.Join("data", "ai-server.db"))
	if err != nil {
		log.Fatalf("[ai-server] initialize persistent store failed: %v", err)
	}
	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			log.Printf("[ai-server] close persistent store failed: %v", closeErr)
		}
	}()

	apiMux := http.NewServeMux()
	wsMux := http.NewServeMux()

	healthHandler := func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"ok":true}`))
	}

	apiMux.HandleFunc("/healthz", healthHandler)
	registerDevItemHandlers(apiMux, store)
	wsMux.HandleFunc("/healthz", healthHandler)
	wsMux.HandleFunc("/ws", func(writer http.ResponseWriter, request *http.Request) {
		handleWebSocket(store, writer, request)
	})

	startClassicActivityAnnouncementLoop()

	go func() {
		log.Println("[ai-server] http api listening on http://127.0.0.1:18080")
		if err := http.ListenAndServe("127.0.0.1:18080", apiMux); err != nil {
			log.Fatalf("[ai-server] http api failed: %v", err)
		}
	}()

	log.Println("[ai-server] websocket listening on ws://127.0.0.1:18443/ws")
	log.Fatal(http.ListenAndServe("127.0.0.1:18443", wsMux))
}

func handleWebSocket(store *session.Store, writer http.ResponseWriter, request *http.Request) {
	conn, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		log.Printf("[ai-server] websocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	socketWriter := &websocketWriter{
		conn:          conn,
		nextServerSeq: 1000,
	}
	socketSession := &packetSession{}
	registeredTeamRoleID := ""
	registeredSceneRoleID := ""
	registeredSceneMapID := 0
	defer func() {
		// world scene:断开连接时给原 mapId 邻居推 removeRole,避免对端留僵尸节点。
		if registeredSceneRoleID != "" {
			if oldMapID, ok := worldSceneHub.unregister(registeredSceneRoleID); ok {
				worldSceneHub.broadcastRemoveRoleToMap(oldMapID, registeredSceneRoleID, registeredSceneRoleID)
			}
		}
		if registeredTeamRoleID == "" {
			return
		}
		classicTeamHub.unregister(registeredTeamRoleID)
		classicTeamHub.broadcast(classicTeamManager.SetOffline(registeredTeamRoleID))
	}()

	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[ai-server] websocket closed: %v", err)
			return
		}
		if messageType != websocket.BinaryMessage {
			continue
		}

		packet, err := protocol.Decode(data)
		if err != nil {
			log.Printf("[ai-server] decode packet failed: %v", err)
			continue
		}

		result := handlePacketWithSession(store, packet, socketSession)
		if !result.handled {
			log.Printf("[ai-server] unsupported command: %d", packet.Cmd)
			continue
		}

		if result.responseCmd != 0 {
			if err := socketWriter.writePacket(result.responseCmd, packet.Seq, result.responsePayload); err != nil {
				log.Printf("[ai-server] write response failed: %v", err)
				return
			}
		}
		if result.townBootstrap != nil {
			socketWriter.writeClassicTownBootstrap(*result.townBootstrap)
			// world scene:传送/切图成功后,把 scene 状态从旧 mapId 迁到新 mapId。
			// sceneTransferFromMapID > 0 表示本次 townBootstrap 来自传送路径(非首次进图)。
			// 同图传送(旧新 mapId 相同)时跳过,避免邻居收到多余的 removeRole+createRole 闪烁。
			if result.sceneTransferFromMapID > 0 && registeredSceneRoleID != "" && socketSession.playerBase != nil && result.sceneTransferFromMapID != socketSession.playerBase.MapID {
				registeredSceneMapID = announceWorldSceneTransfer(socketWriter, socketSession, registeredSceneRoleID, result.sceneTransferFromMapID, result.sceneTransferSpawn)
			}
		}
		if result.teamSyncTransfer != nil {
			classicTeamHub.syncTransfer(store, *result.teamSyncTransfer)
		}
		if result.teamDungeonReset != nil {
			classicTeamHub.resetDungeonInstances(store, *result.teamDungeonReset)
		}
		if result.createPlayer != nil {
			if err := socketWriter.writePush(cmdClassicTownCreatePlayerPush, encodePayload(*result.createPlayer)); err != nil {
				log.Printf("[ai-server] write classic town createPlayer push failed: %v", err)
				return
			}
		}
		if result.rolePhysique != nil {
			if err := socketWriter.writePush(cmdClassicTownRolePhysiquePush, encodePayload(*result.rolePhysique)); err != nil {
				log.Printf("[ai-server] write classic town rolePhysique push failed: %v", err)
				return
			}
		}
		if result.roleState != nil {
			if err := socketWriter.writePush(cmdClassicTownRoleStatePush, encodePayload(*result.roleState)); err != nil {
				log.Printf("[ai-server] write classic town roleState push failed: %v", err)
				return
			}
		}
		if result.answerSpeak != nil {
			if err := socketWriter.writePush(cmdClassicTownAnswerSpeakPush, encodePayload(*result.answerSpeak)); err != nil {
				log.Printf("[ai-server] write classic town answerSpeak push failed: %v", err)
				return
			}
		}
		for _, chatMessage := range result.chatMessages {
			if err := socketWriter.writePush(cmdClassicTownChatMessagePush, encodePayload(chatMessage)); err != nil {
				log.Printf("[ai-server] write classic town chat message failed: %v", err)
				return
			}
		}
		for _, chatBroadcast := range result.chatBroadcasts {
			classicTeamHub.broadcastChat(chatBroadcast.Recipients, chatBroadcast.Message)
		}
		if len(result.teamEvents) > 0 {
			classicTeamHub.broadcast(result.teamEvents)
		}
		if result.skillCap != nil {
			if err := socketWriter.writePush(cmdClassicTownSkillCapPush, encodePayload(*result.skillCap)); err != nil {
				log.Printf("[ai-server] write classic town skillCap push failed: %v", err)
				return
			}
		}
		for _, skillInfo := range result.skillInfos {
			if err := socketWriter.writePush(cmdClassicTownSkillInfoPush, encodePayload(skillInfo)); err != nil {
				log.Printf("[ai-server] write classic town skillInfo push failed: %v", err)
				return
			}
		}
		for _, skillClear := range result.skillClears {
			if err := socketWriter.writePush(cmdClassicTownClearSkillInfo, encodePayload(skillClear)); err != nil {
				log.Printf("[ai-server] write classic town clearSkillInfo push failed: %v", err)
				return
			}
		}
		if result.skillShop != nil {
			if err := socketWriter.writePush(cmdClassicTownSkillShopPush, encodePayload(*result.skillShop)); err != nil {
				log.Printf("[ai-server] write classic town skillShop push failed: %v", err)
				return
			}
		}
		if result.currencyPush != nil {
			if err := socketWriter.writePush(cmdClassicTownCurrencyPush, encodePayload(*result.currencyPush)); err != nil {
				log.Printf("[ai-server] write classic town currency push failed: %v", err)
				return
			}
		}
		if result.fastPanel != nil {
			if err := socketWriter.writePush(cmdClassicTownFastPanelPush, encodePayload(*result.fastPanel)); err != nil {
				log.Printf("[ai-server] write classic town fastPanel push failed: %v", err)
				return
			}
		}
		if result.buySkillResult != nil {
			if err := socketWriter.writePush(cmdClassicTownBuySkillResult, encodePayload(*result.buySkillResult)); err != nil {
				log.Printf("[ai-server] write classic town buySkill result failed: %v", err)
				return
			}
		}
		if result.containerCap != nil {
			if err := socketWriter.writePush(cmdClassicTownCapacityPush, encodePayload(*result.containerCap)); err != nil {
				log.Printf("[ai-server] write classic town container capacity failed: %v", err)
				return
			}
		}
		for _, itemClear := range result.itemClears {
			if err := socketWriter.writePush(cmdClassicTownItemInfoClear, encodePayload(itemClear)); err != nil {
				log.Printf("[ai-server] write classic town item clear failed: %v", err)
				return
			}
		}
		for _, itemInfo := range result.itemInfos {
			if err := socketWriter.writePush(cmdClassicTownItemInfoPush, encodePayload(itemInfo)); err != nil {
				log.Printf("[ai-server] write classic town item info failed: %v", err)
				return
			}
		}
		for _, questInfo := range result.questInfos {
			if err := socketWriter.writePush(cmdClassicTownQuestInfoPush, encodePayload(questInfo)); err != nil {
				log.Printf("[ai-server] write classic town QuestInfo failed: %v", err)
				return
			}
		}
		for _, questClear := range result.questClears {
			if err := socketWriter.writePush(cmdClassicTownClearQuestInfo, encodePayload(questClear)); err != nil {
				log.Printf("[ai-server] write classic town ClearQuestInfo failed: %v", err)
				return
			}
		}
		for _, questState := range result.questStates {
			if err := socketWriter.writePush(cmdClassicTownQuestStatePush, encodePayload(questState)); err != nil {
				log.Printf("[ai-server] write classic town QuestState failed: %v", err)
				return
			}
		}
		if result.dungeonInstance != nil {
			if err := socketWriter.writePush(cmdClassicTownDungeonInstance, encodePayload(*result.dungeonInstance)); err != nil {
				log.Printf("[ai-server] write classic town DungeonInstance failed: %v", err)
				return
			}
		}
		for _, friend := range result.friendInfos {
			if err := socketWriter.writePush(cmdClassicSocialFriendInfo, encodePayload(friend)); err != nil {
				log.Printf("[ai-server] write classic social FriendInfo failed: %v", err)
				return
			}
		}
		for _, friendClear := range result.friendClears {
			if err := socketWriter.writePush(cmdClassicSocialClearFriend, encodePayload(friendClear)); err != nil {
				log.Printf("[ai-server] write classic social ClearFriendInfo failed: %v", err)
				return
			}
		}
		for _, black := range result.blackInfos {
			if err := socketWriter.writePush(cmdClassicSocialBlackListInfo, encodePayload(black)); err != nil {
				log.Printf("[ai-server] write classic social BlackListInfo failed: %v", err)
				return
			}
		}
		for _, blackClear := range result.blackClears {
			if err := socketWriter.writePush(cmdClassicSocialClearBlackList, encodePayload(blackClear)); err != nil {
				log.Printf("[ai-server] write classic social ClearBlackListInfo failed: %v", err)
				return
			}
		}
		for _, enemy := range result.enemyInfos {
			if err := socketWriter.writePush(cmdClassicSocialEnemyInfo, encodePayload(enemy)); err != nil {
				log.Printf("[ai-server] write classic social EnemyInfo failed: %v", err)
				return
			}
		}
		for _, enemyClear := range result.enemyClears {
			if err := socketWriter.writePush(cmdClassicSocialClearEnemy, encodePayload(enemyClear)); err != nil {
				log.Printf("[ai-server] write classic social ClearEnemyInfo failed: %v", err)
				return
			}
		}
		if result.guildInfo != nil {
			if err := socketWriter.writePush(cmdClassicGuildInfoPush, encodePayload(*result.guildInfo)); err != nil {
				log.Printf("[ai-server] write classic guild GuidInfo failed: %v", err)
				return
			}
		}
		for _, member := range result.guildMembers {
			if err := socketWriter.writePush(cmdClassicGuildMemberPush, encodePayload(member)); err != nil {
				log.Printf("[ai-server] write classic guild GuidMemberInfo failed: %v", err)
				return
			}
		}
		for _, memberClear := range result.guildMemberClears {
			if err := socketWriter.writePush(cmdClassicGuildMemberClear, encodePayload(memberClear)); err != nil {
				log.Printf("[ai-server] write classic guild ClearGuidMemberInfo failed: %v", err)
				return
			}
		}
		if result.guildAuth != nil {
			if err := socketWriter.writePush(cmdClassicGuildAuthPush, encodePayload(*result.guildAuth)); err != nil {
				log.Printf("[ai-server] write classic guild auth failed: %v", err)
				return
			}
		}
		if result.guildNotice != nil {
			if err := socketWriter.writePush(cmdClassicGuildNoticePush, encodePayload(*result.guildNotice)); err != nil {
				log.Printf("[ai-server] write classic guild notice failed: %v", err)
				return
			}
		}
		if result.guildResult != nil {
			if err := socketWriter.writePush(cmdClassicGuildCreateResult, encodePayload(*result.guildResult)); err != nil {
				log.Printf("[ai-server] write classic guild result failed: %v", err)
				return
			}
		}
		for _, category := range result.mallCategories {
			if err := socketWriter.writePush(cmdClassicMallCategoryListPush, encodePayload(category)); err != nil {
				log.Printf("[ai-server] write classic mall category failed: %v", err)
				return
			}
		}
		if result.mallSearchCount != nil {
			if err := socketWriter.writePush(cmdClassicMallSearchCountPush, encodePayload(*result.mallSearchCount)); err != nil {
				log.Printf("[ai-server] write classic mall search count failed: %v", err)
				return
			}
		}
		if result.mallSearchPage != nil {
			if err := socketWriter.writePush(cmdClassicMallSearchPagePush, encodePayload(*result.mallSearchPage)); err != nil {
				log.Printf("[ai-server] write classic mall search page failed: %v", err)
				return
			}
		}
		if result.mallCurrency != nil {
			if err := socketWriter.writePush(cmdClassicMallCurrencyPush, encodePayload(*result.mallCurrency)); err != nil {
				log.Printf("[ai-server] write classic mall currency failed: %v", err)
				return
			}
		}
		if result.mallPurchase != nil {
			cmd := uint64(cmdClassicMallPurchaseResult)
			if result.mallPurchase.ErrorCode == mall.PAYMENT_DISABLED {
				cmd = cmdClassicMallPaymentDisabled
			}
			if err := socketWriter.writePush(cmd, encodePayload(*result.mallPurchase)); err != nil {
				log.Printf("[ai-server] write classic mall purchase result failed: %v", err)
				return
			}
		}
		if result.battleStart != nil {
			if err := socketWriter.writePush(cmdClassicBattleStartPush, encodePayload(*result.battleStart)); err != nil {
				log.Printf("[ai-server] write classic battle StartBattle failed: %v", err)
				return
			}
		}
		for _, battleCell := range result.battleCells {
			if err := socketWriter.writePush(cmdClassicBattleCellInfoPush, encodePayload(battleCell)); err != nil {
				log.Printf("[ai-server] write classic battle BattleCellInfo failed: %v", err)
				return
			}
		}
		if result.battleStart != nil && result.battleCommand != nil {
			if err := socketWriter.writePush(cmdClassicBattleStartCommand, encodePayload(*result.battleCommand)); err != nil {
				log.Printf("[ai-server] write classic battle startCommand failed: %v", err)
				return
			}
		}
		for _, battleBuff := range result.battleBuffs {
			if err := socketWriter.writePush(cmdClassicBattleBuffInfoPush, encodePayload(battleBuff)); err != nil {
				log.Printf("[ai-server] write classic battle BuffInfo failed: %v", err)
				return
			}
		}
		for _, battleAction := range result.battleActions {
			if err := socketWriter.writePush(cmdClassicBattleActionPush, encodePayload(battleAction)); err != nil {
				log.Printf("[ai-server] write classic battle battleAction failed: %v", err)
				return
			}
		}
		for _, clearBuff := range result.battleClearBuffs {
			if err := socketWriter.writePush(cmdClassicBattleClearBuffInfo, encodePayload(clearBuff)); err != nil {
				log.Printf("[ai-server] write classic battle clearBuffInfo failed: %v", err)
				return
			}
		}
		if result.battleOver != nil {
			if err := socketWriter.writePush(cmdClassicBattleOverPush, encodePayload(*result.battleOver)); err != nil {
				log.Printf("[ai-server] write classic battle OverBattle failed: %v", err)
				return
			}
		}
		for _, handle := range result.removeRoleHandles {
			if err := socketWriter.writePush(cmdClassicTownRemoveRolePush, encodePayload(handle)); err != nil {
				log.Printf("[ai-server] write classic town removeRole push failed: %v", err)
				return
			}
		}
		if result.moveRole != nil && socketSession.selectedRole != nil && socketSession.playerBase != nil {
			worldSceneHub.broadcastMoveRoleToMap(socketSession.playerBase.MapID, socketSession.selectedRole.RoleID, *result.moveRole)
		}
		if result.battleStart == nil && result.battleCommand != nil {
			if err := socketWriter.writePush(cmdClassicBattleStartCommand, encodePayload(*result.battleCommand)); err != nil {
				log.Printf("[ai-server] write classic battle startCommand failed: %v", err)
				return
			}
		}
		if result.teamBattleStart != nil {
			classicTeamHub.startSharedBattle(*result.teamBattleStart)
		}
		if result.teamBattleSync != nil {
			classicTeamHub.syncSharedBattle(store, *result.teamBattleSync)
		}
		if socketSession.selectedRole != nil && socketSession.playerBase != nil && registeredTeamRoleID != socketSession.selectedRole.RoleID {
			if registeredTeamRoleID != "" {
				classicTeamHub.unregister(registeredTeamRoleID)
				classicTeamHub.broadcast(classicTeamManager.SetOffline(registeredTeamRoleID))
			}
			registeredTeamRoleID = socketSession.selectedRole.RoleID
			classicTeamHub.register(registeredTeamRoleID, socketWriter, socketSession)
			classicTeamHub.broadcast(classicTeamManager.UpsertOnline(classicTeamMemberFromSession(socketSession)))
		} else if socketSession.selectedRole != nil && socketSession.playerBase != nil {
			classicTeamHub.register(socketSession.selectedRole.RoleID, socketWriter, socketSession)
			classicTeamHub.broadcast(classicTeamManager.UpsertOnline(classicTeamMemberFromSession(socketSession)))
		}
		// world scene:只在 roleID 或 mapId 相对上次同步发生变化时才互推 createRole/removeRole,
		// 避免同角色同地图的每个后续包(NPC 交互/背包/移动/心跳)重复互推导致前端节点重复或闪烁。
		// 普通选角后 socketSession.selectedRole.RoleID 与 registeredSceneRoleID 相同、mapId 不变时,
		// syncWorldScenePresence 内部幂等返回,不产生任何 push。
		if socketSession.selectedRole != nil && socketSession.playerBase != nil {
			registeredSceneRoleID, registeredSceneMapID = syncWorldScenePresence(socketWriter, socketSession, registeredSceneRoleID, registeredSceneMapID)
		}
	}
}

func decodePayload(payload []byte, target any) bool {
	if err := json.Unmarshal(payload, target); err != nil {
		log.Printf("[ai-server] decode json payload failed: %v", err)
		return false
	}
	return true
}

func encodePayload(value any) []byte {
	payload, err := json.Marshal(value)
	if err != nil {
		log.Printf("[ai-server] encode json payload failed: %v", err)
		return []byte(`{}`)
	}
	return payload
}
