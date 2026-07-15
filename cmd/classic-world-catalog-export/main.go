package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"ai-server/internal/world"
)

func main() {
	outputDir := flag.String("out-dir", "internal/classicdata", "directory for generated world catalog CSV files")
	flag.Parse()
	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		panic(err)
	}
	writeMapCatalog(filepath.Join(*outputDir, "classic_map_catalog.csv"))
	writeMapNPCCatalog(filepath.Join(*outputDir, "classic_map_npc_catalog.csv"))
	writeMapSceneTransportCatalog(filepath.Join(*outputDir, "classic_map_scene_transport_catalog.csv"))
	writeMapCollectionCatalog(filepath.Join(*outputDir, "classic_map_collection_catalog.csv"))
	fmt.Printf("exported maps=%d staticNpcs=%d sceneTransports=%d collections=%d\n",
		len(world.ExportClassicMapCatalogRows()),
		len(world.ExportClassicMapNPCatalogRows()),
		len(world.ExportClassicMapSceneTransportCatalogRows()),
		len(world.ExportClassicMapCollectionCatalogRows()),
	)
}

func writeMapCatalog(output string) {
	file, err := os.Create(output)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	mustWrite(writer, []string{"map_id", "name", "xml_url", "default_spawn_x", "default_spawn_y", "supports_transfer_in"})
	for _, row := range world.ExportClassicMapCatalogRows() {
		mustWrite(writer, []string{
			strconv.Itoa(row.ID),
			row.Name,
			row.XMLURL,
			strconv.Itoa(row.DefaultSpawn.X),
			strconv.Itoa(row.DefaultSpawn.Y),
			strconv.FormatBool(row.SupportsTransferIn),
		})
	}
}

func writeMapNPCCatalog(output string) {
	file, err := os.Create(output)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	mustWrite(writer, []string{"map_id", "handle", "role_id", "display_name", "source_query", "sprite_name", "width", "height", "spawn_x", "spawn_y", "state", "quest_state", "guild_name", "guild_pic", "kind"})
	for _, row := range world.ExportClassicMapNPCatalogRows() {
		mustWrite(writer, []string{
			strconv.Itoa(row.MapID),
			row.Handle,
			row.RoleID,
			row.DisplayName,
			row.SourceQuery,
			row.SpriteName,
			strconv.Itoa(row.Width),
			strconv.Itoa(row.Height),
			strconv.Itoa(row.SpawnFlash.X),
			strconv.Itoa(row.SpawnFlash.Y),
			strconv.Itoa(row.State),
			strconv.Itoa(row.QuestState),
			row.GuildName,
			row.GuildPic,
			row.Kind,
		})
	}
}

func writeMapSceneTransportCatalog(output string) {
	file, err := os.Create(output)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	mustWrite(writer, []string{"map_id", "handle", "source_query", "sprite_name", "width", "height", "spawn_x", "spawn_y", "target_map_id", "target_spawn_x", "target_spawn_y", "protocol", "answer_handle"})
	for _, row := range world.ExportClassicMapSceneTransportCatalogRows() {
		mustWrite(writer, []string{
			strconv.Itoa(row.MapID),
			row.Handle,
			row.SourceQuery,
			row.SpriteName,
			strconv.Itoa(row.Width),
			strconv.Itoa(row.Height),
			strconv.Itoa(row.SpawnFlash.X),
			strconv.Itoa(row.SpawnFlash.Y),
			strconv.Itoa(row.TargetMapID),
			strconv.Itoa(row.TargetSpawn.X),
			strconv.Itoa(row.TargetSpawn.Y),
			row.Protocol,
			row.AnswerHandle,
		})
	}
}

func writeMapCollectionCatalog(output string) {
	file, err := os.Create(output)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	mustWrite(writer, []string{"map_id", "handle", "display_name", "source_query", "sprite_name", "width", "height", "spawn_x", "spawn_y", "quest_state", "required_item_name", "reward_item_name", "quest_title", "quest_description", "protocol"})
	for _, row := range world.ExportClassicMapCollectionCatalogRows() {
		mustWrite(writer, []string{
			strconv.Itoa(row.MapID),
			row.Handle,
			row.DisplayName,
			row.SourceQuery,
			row.SpriteName,
			strconv.Itoa(row.Width),
			strconv.Itoa(row.Height),
			strconv.Itoa(row.SpawnFlash.X),
			strconv.Itoa(row.SpawnFlash.Y),
			strconv.Itoa(row.QuestState),
			row.RequiredItemName,
			row.RewardItemName,
			row.QuestTitle,
			row.QuestDescription,
			row.Protocol,
		})
	}
}

func mustWrite(writer *csv.Writer, record []string) {
	if err := writer.Write(record); err != nil {
		panic(err)
	}
}
