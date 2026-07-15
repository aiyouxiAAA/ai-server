package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var sourceNPCVariablePattern = regexp.MustCompile("^map([0-9]+)SourceNPCs$")

type npcRow struct {
	mapID       int
	handle      string
	roleID      string
	displayName string
	sourceQuery string
	spriteName  string
	width       int
	height      int
	spawnX      int
	spawnY      int
	state       int
	questState  int
	guildName   string
	guildPic    string
	kind        string
}

func main() {
	sourcePath := flag.String("source", "internal/world/town_bootstrap_source_data*.go", "source NPC definition file glob")
	outputPath := flag.String("out", "internal/classicdata/classic_map_npc_catalog.csv", "generated NPC catalog CSV")
	flag.Parse()

	rows, err := readNPCSourceRows(*sourcePath)
	if err != nil {
		panic(err)
	}
	if err := os.MkdirAll(filepath.Dir(*outputPath), 0o755); err != nil {
		panic(err)
	}
	file, err := os.Create(*outputPath)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	mustWrite(writer, []string{"map_id", "handle", "role_id", "display_name", "source_query", "sprite_name", "width", "height", "spawn_x", "spawn_y", "state", "quest_state", "guild_name", "guild_pic", "kind"})
	for _, row := range rows {
		mustWrite(writer, []string{
			strconv.Itoa(row.mapID),
			row.handle,
			row.roleID,
			row.displayName,
			row.sourceQuery,
			row.spriteName,
			strconv.Itoa(row.width),
			strconv.Itoa(row.height),
			strconv.Itoa(row.spawnX),
			strconv.Itoa(row.spawnY),
			strconv.Itoa(row.state),
			strconv.Itoa(row.questState),
			row.guildName,
			row.guildPic,
			row.kind,
		})
	}
	fmt.Printf("exported staticNpcs=%d\n", len(rows))
}

func readNPCSourceRows(sourcePath string) ([]npcRow, error) {
	matches, err := filepath.Glob(sourcePath)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("source NPC file glob matched nothing: %s", sourcePath)
	}
	rows := []npcRow{}
	for _, matchedPath := range matches {
		fileRows, err := readNPCSourceFile(matchedPath)
		if err != nil {
			return nil, err
		}
		rows = append(rows, fileRows...)
	}
	sort.SliceStable(rows, func(left, right int) bool {
		if rows[left].mapID != rows[right].mapID {
			return rows[left].mapID < rows[right].mapID
		}
		return false
	})
	return rows, nil
}

func readNPCSourceFile(sourcePath string) ([]npcRow, error) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, sourcePath, nil, 0)
	if err != nil {
		return nil, err
	}
	rows := []npcRow{}
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.VAR {
			continue
		}
		for _, specification := range general.Specs {
			valueSpec, ok := specification.(*ast.ValueSpec)
			if !ok || len(valueSpec.Names) != 1 || len(valueSpec.Values) != 1 {
				continue
			}
			match := sourceNPCVariablePattern.FindStringSubmatch(valueSpec.Names[0].Name)
			if len(match) != 2 {
				continue
			}
			mapID, err := strconv.Atoi(match[1])
			if err != nil {
				return nil, err
			}
			list, ok := valueSpec.Values[0].(*ast.CompositeLit)
			if !ok {
				return nil, fmt.Errorf("%s is not an NPC list", valueSpec.Names[0].Name)
			}
			for index, element := range list.Elts {
				entry, ok := element.(*ast.CompositeLit)
				if !ok {
					if isGeneratedTransportCall(element) {
						continue
					}
					return nil, fmt.Errorf("%s[%d] is not an NPC literal", valueSpec.Names[0].Name, index)
				}
				row, include, err := parseNPCRow(mapID, entry)
				if err != nil {
					return nil, fmt.Errorf("%s[%d]: %w", valueSpec.Names[0].Name, index, err)
				}
				if include {
					rows = append(rows, row)
				}
			}
		}
	}
	return rows, nil
}

func isGeneratedTransportCall(expression ast.Expr) bool {
	call, ok := expression.(*ast.CallExpr)
	if !ok {
		return false
	}
	name, ok := call.Fun.(*ast.Ident)
	return ok && strings.HasPrefix(name.Name, "buildCapturedSourceTransport")
}

func parseNPCRow(mapID int, entry *ast.CompositeLit) (npcRow, bool, error) {
	row := npcRow{mapID: mapID}
	for _, element := range entry.Elts {
		field, ok := element.(*ast.KeyValueExpr)
		if !ok {
			return npcRow{}, false, fmt.Errorf("NPC field is not keyed")
		}
		name, ok := field.Key.(*ast.Ident)
		if !ok {
			return npcRow{}, false, fmt.Errorf("NPC field key is not an identifier")
		}
		switch name.Name {
		case "Handle":
			row.handle = requireString(field.Value)
		case "RoleID":
			row.roleID = requireString(field.Value)
		case "DisplayName":
			row.displayName = requireString(field.Value)
		case "SourceQuery":
			row.sourceQuery = requireString(field.Value)
		case "SpriteName":
			row.spriteName = requireString(field.Value)
		case "Width":
			row.width = requireInt(field.Value)
		case "Height":
			row.height = requireInt(field.Value)
		case "SpawnFlash":
			row.spawnX, row.spawnY = requireSpawnPoint(field.Value)
		case "State":
			row.state = requireInt(field.Value)
		case "QuestState":
			row.questState = requireInt(field.Value)
		case "GuildName":
			row.guildName = requireString(field.Value)
		case "GuildPic":
			row.guildPic = requireString(field.Value)
		case "Kind":
			row.kind = requireString(field.Value)
		}
	}
	if row.roleID == "-3" {
		return npcRow{}, false, nil
	}
	if row.handle == "" || row.displayName == "" || row.sourceQuery == "" || row.spriteName == "" {
		return npcRow{}, false, fmt.Errorf("required static NPC fields are empty")
	}
	return row, true, nil
}

func requireString(expression ast.Expr) string {
	value, ok := expression.(*ast.BasicLit)
	if !ok || value.Kind != token.STRING {
		panic(fmt.Sprintf("expected string expression, got %T", expression))
	}
	result, err := strconv.Unquote(value.Value)
	if err != nil {
		panic(err)
	}
	return result
}

func requireInt(expression ast.Expr) int {
	value, ok := expression.(*ast.BasicLit)
	if !ok || value.Kind != token.INT {
		panic(fmt.Sprintf("expected integer expression, got %T", expression))
	}
	result, err := strconv.Atoi(value.Value)
	if err != nil {
		panic(err)
	}
	return result
}

func requireSpawnPoint(expression ast.Expr) (int, int) {
	literal, ok := expression.(*ast.CompositeLit)
	if !ok {
		panic(fmt.Sprintf("expected SpawnPoint literal, got %T", expression))
	}
	x := 0
	y := 0
	for _, element := range literal.Elts {
		field, ok := element.(*ast.KeyValueExpr)
		if !ok {
			panic("expected keyed SpawnPoint field")
		}
		name, ok := field.Key.(*ast.Ident)
		if !ok {
			panic("expected named SpawnPoint field")
		}
		switch name.Name {
		case "X":
			x = requireInt(field.Value)
		case "Y":
			y = requireInt(field.Value)
		}
	}
	return x, y
}

func mustWrite(writer *csv.Writer, record []string) {
	if err := writer.Write(record); err != nil {
		panic(err)
	}
}
