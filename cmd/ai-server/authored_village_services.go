package main

import (
	"ai-server/internal/session"
	"ai-server/internal/world"
	"fmt"
	"strings"
)

const authoredServiceMessageID = "village-services"

func init() {
	// Reuse verified goods, prices and transaction code; only their authored owner changes.
	for _, npc := range world.AuthoredServiceNPCs() {
		if _, exists := sourceGuangqingItemShopRoutes[npc.Handle()]; exists {
			panic("duplicate shop owner: " + npc.Handle())
		}
		for _, service := range npc.Services {
			var rows, title, category string
			switch service {
			case "heal":
				continue
			case "medicine":
				rows, title, category = sourceGuangqingHealerShopRows, "药品", "医疗"
			case "weapon":
				rows, title, category = sourceGuangqingWeaponShopRows, "武器", "武器"
			case "armor":
				rows, title, category = sourceYunyinArmorShopRows, "护具", "护具"
			case "grocery":
				rows, title, category = sourceDafoGroceryShopRows, "杂货", "道具"
			default:
				panic("unregistered village shop: " + service)
			}
			sourceGuangqingItemShopRoutes[npc.Handle()] = append(sourceGuangqingItemShopRoutes[npc.Handle()], sourceItemShopRoute{handle: npc.Handle(), answerHandle: service, shopID: "item:" + npc.Handle() + ":" + service, title: npc.Name + "的" + title + "商店", vocation: category, rows: rows})
		}
	}
}

func authoredServiceError(socket *packetSession, npc world.AuthoredServiceNPC) string {
	if socket == nil || socket.selectedRole == nil || socket.playerBase == nil {
		return "请先选择角色。"
	}
	if socket.battleRuntime != nil {
		return "战斗中不能使用村庄服务。"
	}
	if socket.playerBase.MapID != npc.MapID {
		return "请回到雪栈村与" + npc.Name + "交谈。"
	}
	if socket.playerBase.RoleState == nil || socket.playerBase.RolePhysique == nil {
		return "角色状态尚未就绪，请重新登录。"
	}
	return ""
}

func authoredServiceDialogue(socket *packetSession, npc world.AuthoredServiceNPC) world.AnswerSpeakPush {
	message := world.AnswerSpeakPush{Handle: npc.Handle(), MsgHandle: authoredServiceMessageID, Msg: npc.Dialogue, Answers: []world.AnswerOption{}}
	for _, service := range npc.Services {
		label := ""
		switch service {
		case "heal":
			base := socket.playerBase
			if base.RoleState == nil || base.RolePhysique == nil {
				panic("authored healer requires authoritative role state")
			}
			cost := session.ClassicTownHealerCost(base.Level, base.RolePhysique.MaxHP-base.RoleState.HP, base.RolePhysique.MaxMP-base.RoleState.MP)
			label = "疗伤（免费）"
			if cost > 0 {
				label = fmt.Sprintf("疗伤（%d铜钱）", cost)
			}
			message.Msg += "<br/>疗伤可补满气力与精力，15级前免费，15级起按损耗收取铜钱。"
		case "medicine":
			label = "购买药品"
		case "weapon":
			label = "购买武器"
		case "armor":
			label = "购买护具"
		case "grocery":
			label = "杂货买卖"
		}
		message.Answers = append(message.Answers, world.AnswerOption{Handle: service, Msg: label})
	}
	return message
}

func buildAuthoredServiceOpenResult(socket *packetSession, handle string) (packetResult, bool) {
	npc, ok := world.FindAuthoredServiceNPC(handle)
	if !ok {
		return packetResult{}, false
	}
	if message := authoredServiceError(socket, npc); message != "" {
		return packetResult{handled: true, errorMessages: []classicTownErrorPush{{Msg: message}}}, true
	}
	dialogue := authoredServiceDialogue(socket, npc)
	return packetResult{handled: true, answerSpeak: &dialogue}, true
}

func buildAuthoredServiceAnswerResult(store *session.Store, socket *packetSession, request classicTownAnswerRequest) (packetResult, bool) {
	npc, ok := world.FindAuthoredServiceNPC(request.Handle)
	if !ok {
		return packetResult{}, false
	}
	if message := authoredServiceError(socket, npc); message != "" {
		return packetResult{handled: true, errorMessages: []classicTownErrorPush{{Msg: message}}}, true
	}
	if request.MsgHandle != authoredServiceMessageID {
		return packetResult{handled: true, errorMessages: []classicTownErrorPush{{Msg: "对话已失效，请重新与村民交谈。"}}}, true
	}
	for _, service := range npc.Services {
		if request.AnswerHandle != service {
			continue
		}
		if service != "heal" {
			return buildClassicTownItemShopResult(request)
		}
		// Reuse the existing authoritative heal result, including inventory/currency deltas.
		request.AnswerHandle = "2"
		result, handled := buildClassicTownHealerResult(store, socket, request)
		if result.answerSpeak != nil {
			dialogue := authoredServiceDialogue(socket, npc)
			dialogue.Msg = result.answerSpeak.Msg + "<br/>" + dialogue.Msg
			result.answerSpeak = &dialogue
		}
		return result, handled
	}
	return packetResult{handled: true, errorMessages: []classicTownErrorPush{{Msg: "该村民不提供此项服务。"}}}, true
}

func authoredShopPurchaseError(socket *packetSession, shopID string) string {
	if !strings.HasPrefix(shopID, "item:authored-") {
		return ""
	}
	for _, npc := range world.AuthoredServiceNPCs() {
		for _, route := range sourceGuangqingItemShopRoutes[npc.Handle()] {
			if sourceItemShopRouteID(route) == shopID {
				return authoredServiceError(socket, npc)
			}
		}
	}
	return "商品商店不存在。"
}
