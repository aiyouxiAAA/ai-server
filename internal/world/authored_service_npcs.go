package world

import (
	_ "embed"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
)

// Authored identities are separate from the immutable captured NPC catalog.
//
//go:embed authored_service_npcs.csv
var authoredServiceNPCData string

type AuthoredServiceNPC struct {
	MapID               int
	Key, Name, Dialogue string
	Spawn               SpawnPoint
	Services            []string
}

func (npc AuthoredServiceNPC) Handle() string { return "authored-" + npc.Key }

var authoredServiceNPCs = parseAuthoredServiceNPCs(authoredServiceNPCData)

func parseAuthoredServiceNPCs(data string) []AuthoredServiceNPC {
	rows, err := csv.NewReader(strings.NewReader(data)).ReadAll()
	if err != nil || len(rows) < 2 {
		panic(fmt.Sprintf("invalid authored service NPC catalog: %v", err))
	}
	seen := map[string]bool{}
	result := make([]AuthoredServiceNPC, 0, len(rows)-1)
	for _, row := range rows[1:] {
		if len(row) != 7 {
			panic("authored service NPC column mismatch")
		}
		number := func(index int) int {
			n, err := strconv.Atoi(row[index])
			if err != nil || n < 0 {
				panic("invalid authored service NPC coordinates/map")
			}
			return n
		}
		npc := AuthoredServiceNPC{MapID: number(0), Key: row[1], Name: row[2], Spawn: SpawnPoint{X: number(3), Y: number(4)}, Dialogue: row[5], Services: strings.Split(row[6], "|")}
		if npc.MapID == 0 || npc.Key == "" || npc.Name == "" || npc.Dialogue == "" || seen[npc.Handle()] {
			panic("duplicate or incomplete authored service NPC")
		}
		seen[npc.Handle()] = true
		services := map[string]bool{}
		for _, service := range npc.Services {
			if services[service] {
				panic("duplicate authored NPC service")
			}
			services[service] = true
			switch service {
			case "heal", "medicine", "weapon", "armor", "grocery":
			default:
				panic("unknown authored NPC service: " + service)
			}
		}
		result = append(result, npc)
	}
	return result
}

func AuthoredServiceNPCs() []AuthoredServiceNPC {
	result := append([]AuthoredServiceNPC(nil), authoredServiceNPCs...)
	for i := range result {
		result[i].Services = append([]string(nil), result[i].Services...)
	}
	return result
}

func FindAuthoredServiceNPC(handle string) (AuthoredServiceNPC, bool) {
	for _, npc := range AuthoredServiceNPCs() {
		if npc.Handle() == handle {
			return npc, true
		}
	}
	return AuthoredServiceNPC{}, false
}

func AppendAuthoredServiceNPCs(snapshot *TownBootstrapSnapshot) {
	if snapshot == nil {
		return
	}
	for _, npc := range authoredServiceNPCs {
		if snapshot.LoadMap.MapID != strconv.Itoa(npc.MapID) {
			continue
		}
		exists := false
		for _, role := range snapshot.CreateRoles {
			if role.Handle == npc.Handle() {
				exists = true
				break
			}
		}
		if exists {
			continue
		}
		snapshot.CreateRoles = append(snapshot.CreateRoles, RolePush{Handle: npc.Handle(), RoleID: "-1", DisplayName: npc.Name, Level: 1, MapID: snapshot.LoadMap.MapID, SourceQuery: "authored/" + npc.Key, Kind: "npc", SpawnFlash: npc.Spawn})
	}
}
