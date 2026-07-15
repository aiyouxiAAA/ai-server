package quest

import "ai-server/internal/classicdata"

type ClassicMapNPCQuestRoute struct {
	QuestID      string
	QuestTitle   string
	MapID        int
	MapName      string
	NPCHandle    string
	NPCName      string
	MsgHandle    string
	AnswerHandle string
}

func ClassicMapNPCQuestRoutes() []ClassicMapNPCQuestRoute {
	result := []ClassicMapNPCQuestRoute{}
	for _, info := range All() {
		for _, route := range info.Routes {
			for _, npc := range classicdata.FindClassicMapNPCSpawnsByHandle(route.Handle) {
				mapEntry, ok := classicdata.FindClassicMap(npc.MapID)
				if !ok {
					panic("Classic NPC route references an unknown map")
				}
				result = append(result, ClassicMapNPCQuestRoute{
					QuestID:      info.ID,
					QuestTitle:   info.Title,
					MapID:        mapEntry.ID,
					MapName:      mapEntry.Name,
					NPCHandle:    npc.Handle,
					NPCName:      npc.DisplayName,
					MsgHandle:    route.MsgHandle,
					AnswerHandle: route.AnswerHandle,
				})
			}
		}
	}
	return result
}

type WuliangMapNPCQuestRoute = ClassicMapNPCQuestRoute

func WuliangMapNPCQuestRoutes() []WuliangMapNPCQuestRoute {
	result := []WuliangMapNPCQuestRoute{}
	for _, route := range ClassicMapNPCQuestRoutes() {
		if route.MapID == 190 || route.MapID == 191 {
			result = append(result, route)
		}
	}
	return result
}
