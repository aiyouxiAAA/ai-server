package main

import "time"

const classicPointCouponThiefSpawnNoticeText = "[<font color='#00ccff'>点券盗贼</font>]突然出现在树海、卧佛谷、竹林地区。"

func startClassicActivityAnnouncementLoop() {
	lastAnnouncedHour := time.Now().Truncate(time.Hour)
	ticker := time.NewTicker(time.Second)
	go func() {
		for now := range ticker.C {
			currentHour := now.Truncate(time.Hour)
			if !currentHour.After(lastAnnouncedHour) {
				continue
			}
			lastAnnouncedHour = currentHour
			broadcastClassicPointCouponThiefSpawnNotice()
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
