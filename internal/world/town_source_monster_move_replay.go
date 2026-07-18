package world

import "strconv"

func CapturedSourceMonsterMoveReplayForMap(mapID int) []RoleMovePush {
	steps := capturedSourceMonsterMoveReplayByMapID[mapID]
	if len(steps) == 0 {
		return nil
	}

	mapIDText := strconv.Itoa(mapID)
	replay := make([]RoleMovePush, 0, len(steps))
	for _, step := range steps {
		step.MapID = mapIDText
		replay = append(replay, step)
	}
	return replay
}

func CapturedSourceMonsterMoveReplayLoopsForMap(mapID int) bool {
	return capturedSourceMonsterMoveReplayLoopingMapIDs[mapID]
}

func CapturedSourceMonsterMoveReplayForBootstrap(snapshot TownBootstrapSnapshot) []RoleMovePush {
	mapID, err := strconv.Atoi(snapshot.LoadMap.MapID)
	if err != nil {
		return nil
	}

	visibleHandles := make(map[string]bool, len(snapshot.CreateRoles))
	for _, role := range snapshot.CreateRoles {
		if role.Kind == "monster" && role.RoleID == "-2" {
			visibleHandles[role.Handle] = true
		}
	}
	if len(visibleHandles) == 0 {
		return nil
	}

	steps := CapturedSourceMonsterMoveReplayForMap(mapID)
	if len(steps) == 0 {
		return nil
	}

	filtered := make([]RoleMovePush, 0, len(steps))
	for _, step := range steps {
		if visibleHandles[step.Handle] {
			filtered = append(filtered, step)
		}
	}
	return filtered
}

func CapturedSourceMonsterMoveReplayLoopsForBootstrap(snapshot TownBootstrapSnapshot) bool {
	mapID, err := strconv.Atoi(snapshot.LoadMap.MapID)
	if err != nil {
		return false
	}
	return CapturedSourceMonsterMoveReplayLoopsForMap(mapID)
}
