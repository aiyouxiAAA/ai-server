package main

import (
	"time"

	"ai-server/internal/classicactivity"
	"ai-server/internal/world"
)

const classicPointCouponThiefSpawnNoticeText = "[<font color='#00ccff'>点券盗贼</font>]突然出现在树海、卧佛谷、竹林地区。"

func startClassicActivityAnnouncementLoop() {
	lastAnnouncedHour := time.Now().Truncate(time.Hour)
	lastBainianBucket := classicactivity.BainianChongjingCycleBucket(time.Now())
	lastBainianPhase := classicactivity.BainianChongjingPhaseAt(time.Now())
	ticker := time.NewTicker(time.Second)
	go func() {
		for now := range ticker.C {
			currentHour := now.Truncate(time.Hour)
			if currentHour.After(lastAnnouncedHour) {
				lastAnnouncedHour = currentHour
				broadcastClassicPointCouponThiefRefresh(classicactivity.AdvancePointCouponThiefRefresh(now))
			}

			phase := classicactivity.BainianChongjingPhaseAt(now)
			bucket := classicactivity.BainianChongjingCycleBucket(now)
			if bucket != lastBainianBucket {
				// New cycle: remove any leftover live entities, then announce warning.
				lastBainianBucket = bucket
				lastBainianPhase = classicactivity.BainianChongjingPhaseWarning
				broadcastClassicBainianChongjingLiveRemove()
				broadcastClassicBainianChongjingWarningNotice()
				continue
			}
			if phase == classicactivity.BainianChongjingPhaseSpawned && lastBainianPhase != classicactivity.BainianChongjingPhaseSpawned {
				lastBainianPhase = phase
				if classicactivity.BainianChongjingIsForcedForDev(now) {
					continue
				}
				if classicactivity.BainianChongjingIsAlive(now) {
					broadcastClassicBainianChongjingSpawnNotice()
					broadcastClassicBainianChongjingLiveSpawn()
				}
				continue
			}
			lastBainianPhase = phase
		}
	}()
}

func broadcastClassicPointCouponThiefSpawnNotice() {
	worldSceneHub.broadcastChatToAll(classicPointCouponThiefSpawnAnnouncementMessage())
}

func classicPointCouponThiefSpawnAnnouncementMessage() classicTownChatMessagePush {
	return classicTownChatMessagePush{
		Channel: "system",
		Msg:     "<w>" + classicPointCouponThiefSpawnNoticeText,
	}
}

func broadcastClassicPointCouponThiefRefresh(refresh classicactivity.PointCouponThiefRefresh) {
	for _, spawn := range refresh.Previous {
		worldSceneHub.broadcastStaticRemoveHandlesToMap(spawn.MapID, []string{spawn.Handle})
	}
	broadcastClassicPointCouponThiefSpawnNotice()
	for _, spawn := range refresh.Current {
		worldSceneHub.broadcastStaticCreateRolesToMap(spawn.MapID, []world.RolePush{world.PointCouponThiefRolePush(spawn)})
	}
}

func broadcastClassicBainianChongjingWarningNotice() {
	worldSceneHub.broadcastChatToAll(classicBainianChongjingWarningAnnouncementMessage())
}

func classicBainianChongjingWarningAnnouncementMessage() classicTownChatMessagePush {
	return classicTownChatMessagePush{
		Channel: "system",
		Msg:     "<w>" + classicactivity.BainianChongjingWarningNoticeText(),
	}
}

func broadcastClassicBainianChongjingSpawnNotice() {
	worldSceneHub.broadcastChatToAll(classicBainianChongjingSpawnAnnouncementMessage())
}

func classicBainianChongjingSpawnAnnouncementMessage() classicTownChatMessagePush {
	return classicTownChatMessagePush{
		Channel: "system",
		Msg:     "<w>" + classicactivity.BainianChongjingSpawnNoticeText(),
	}
}

func classicBainianChongjingKillAnnouncementMessage(killerNames ...string) classicTownChatMessagePush {
	return classicTownChatMessagePush{
		Channel: "system",
		Msg:     "<w>" + classicactivity.BainianChongjingKillNoticeText(killerNames...),
	}
}

func broadcastClassicBainianChongjingLiveSpawn() {
	roles := world.BainianChongjingLiveRolePushes()
	if len(roles) == 0 {
		return
	}
	worldSceneHub.broadcastStaticCreateRolesToMap(classicactivity.BainianChongjingMapID, roles)
}

func broadcastClassicBainianChongjingLiveRemove() {
	handles := world.BainianChongjingLiveHandles()
	if len(handles) == 0 {
		return
	}
	worldSceneHub.broadcastStaticRemoveHandlesToMap(classicactivity.BainianChongjingMapID, handles)
}
