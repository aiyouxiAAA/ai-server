package main

import (
	"strings"
	"testing"

	"ai-server/internal/classicactivity"
)

func TestClassicBainianChongjingAnnouncementsKeepSystemChannelOutOfText(t *testing.T) {
	cases := []struct {
		name string
		got  classicTownChatMessagePush
		want string
	}{
		{
			name: "warning",
			got:  classicBainianChongjingWarningAnnouncementMessage(),
			want: "<w>" + classicactivity.BainianChongjingWarningNoticeText(),
		},
		{
			name: "spawn",
			got:  classicBainianChongjingSpawnAnnouncementMessage(),
			want: "<w>" + classicactivity.BainianChongjingSpawnNoticeText(),
		},
		{
			name: "kill",
			got:  classicBainianChongjingKillAnnouncementMessage("桥头的樵夫"),
			want: "<w>" + classicactivity.BainianChongjingKillNoticeText("桥头的樵夫"),
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.got.Channel != "system" {
				t.Fatalf("expected system channel, got %+v", testCase.got)
			}
			if testCase.got.Msg != testCase.want || strings.Contains(testCase.got.Msg, ",50") {
				t.Fatalf("expected message %q without protocol channel field, got %+v", testCase.want, testCase.got)
			}
		})
	}
}
