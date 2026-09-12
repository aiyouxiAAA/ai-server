package main

import (
	"testing"
)

func TestFreshStartSocialMailAndAuctionHaveNoCapturedPlayerData(t *testing.T) {
	store, socket := seedVillageServiceRole(t, 1, 0, 0)
	for pass := 0; pass < 3; pass++ {
		if result := buildClassicSocialGetFriendListResult(store, socket); len(result.friendInfos) != 0 {
			t.Fatal("captured friend injected")
		}
		if result := buildClassicMailListPush(); len(result.Items) != 0 {
			t.Fatal("captured mail injected")
		}
		if result := buildClassicAuctionListResult(classicAuctionListRequest{}); result.auctionList.Total != 0 || len(result.auctionList.Items) != 0 {
			t.Fatal("captured auction injected")
		}
	}
	if _, ok := resolveSocialEntryBase(store, classicSocialMutateRequest{RoleName: "恐龙抗狼1"}); ok {
		t.Fatal("nonexistent friend synthesized")
	}
	if result := buildClassicTownStoredEquipmentResult(store, classicTownOtherEquipmentRequest{Handle: "player_21424"}); result.otherEquipment.ErrorCode != "role_missing" || len(result.otherEquipment.Items) != 0 {
		t.Fatal("captured player equipment returned")
	}
	if result := buildClassicTownStoredEquipmentResult(store, classicTownOtherEquipmentRequest{RoleID: socket.selectedRole.RoleID}); result.otherEquipment.ErrorCode != "" || result.otherEquipment.RoleID != socket.selectedRole.RoleID {
		t.Fatal("real player equipment missing")
	}
	socket.currentMailHandle = "5544758100159914"
	if result := buildClassicMailContainerMoveResult(store, socket); len(result.itemInfos) != 0 {
		t.Fatal("captured attachments claimed")
	}
	if result := buildClassicMailInfoResult(socket, classicMailInfoRequest{Handle: "5544758100159914"}); result.mailInfo.Found {
		t.Fatal("captured mail found")
	}
	// Normal NPC equipment/healing and quest grants use the real store, not these removed fixtures.
	if result := store.TransactAuthoredQuest(socket.playerBase.PlayerID, socket.selectedRole.RoleID, "XZ-M001", false); !result.Changed {
		t.Fatal(result)
	}
}
