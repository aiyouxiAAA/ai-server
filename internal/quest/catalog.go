package quest

import (
	"embed"
	"encoding/csv"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

//go:embed classic_quest_catalog.csv
var catalogFS embed.FS

type Info struct {
	ID               string
	Title            string
	Level            int
	Type             string
	Description      string
	State            string
	Routes           []AnswerRoute
	QuestStateHandle string
	Requirements     []RewardItem
	Reward           Reward
	RewardEntries    []RewardEntry
}

type AnswerRoute struct {
	Handle       string
	MsgHandle    string
	AnswerHandle string
}

type Reward struct {
	Experience    int
	Items         []RewardItem
	Skills        []RewardSkill
	OptionalItems []RewardItem
}

type RewardItem struct {
	Name        string
	Count       int
	Display     string
	Description string
	SourceMeta  string
}

type RewardSkill struct {
	Name string
}

type RewardEntryPhase string

const (
	RewardEntryPhaseGrant    RewardEntryPhase = "grant"
	RewardEntryPhaseOptional RewardEntryPhase = "optional"
)

type RewardEntryKind string

const (
	RewardEntryKindExperience RewardEntryKind = "experience"
	RewardEntryKindCurrency   RewardEntryKind = "currency"
	RewardEntryKindItem       RewardEntryKind = "item"
	RewardEntryKindSkill      RewardEntryKind = "skill"
)

const (
	RewardEntrySourceQuest        = "quest"
	RewardEntryProbabilityCertain = 100
)

type RewardEntry struct {
	SourceType  string
	SourceID    string
	Phase       RewardEntryPhase
	Kind        RewardEntryKind
	Name        string
	CountMin    int
	CountMax    int
	Probability int
	ChooseGroup string
	Display     string
	Description string
	SourceMeta  string
}

func All() []Info {
	rows, err := loadCatalog()
	if err != nil {
		panic(err)
	}
	return cloneInfos(rows)
}

func AllRewardEntries() []RewardEntry {
	infos := All()
	entries := []RewardEntry{}
	for _, info := range infos {
		entries = append(entries, info.RewardEntries...)
	}
	return cloneRewardEntries(entries)
}

func FindByTitle(title string) (Info, bool) {
	normalizedTitle := strings.TrimSpace(title)
	if normalizedTitle == "" {
		return Info{}, false
	}
	for _, info := range All() {
		if info.Title == normalizedTitle {
			return info, true
		}
	}
	return Info{}, false
}

func FindByID(id string) (Info, bool) {
	normalizedID := strings.TrimSpace(id)
	if normalizedID == "" {
		return Info{}, false
	}
	for _, info := range All() {
		if info.ID == normalizedID {
			return info, true
		}
	}
	return Info{}, false
}

func loadCatalog() ([]Info, error) {
	file, err := catalogFS.Open("classic_quest_catalog.csv")
	if err != nil {
		return nil, fmt.Errorf("open classic quest catalog: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read classic quest catalog header: %w", err)
	}
	indexByHeader := map[string]int{}
	for index, header := range headers {
		indexByHeader[strings.TrimSpace(header)] = index
	}

	readField := func(record []string, name string) string {
		index, ok := indexByHeader[name]
		if !ok || index < 0 || index >= len(record) {
			return ""
		}
		return strings.TrimSpace(record[index])
	}

	infos := []Info{}
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read classic quest catalog row: %w", err)
		}
		level, _ := strconv.Atoi(readField(record, "level"))
		info := Info{
			ID:               readField(record, "quest_id"),
			Title:            readField(record, "title"),
			Level:            level,
			Type:             readField(record, "type"),
			Description:      readField(record, "description"),
			State:            readField(record, "state"),
			Routes:           parseAnswerRoutes(readField(record, "routes")),
			QuestStateHandle: readField(record, "quest_state_handle"),
		}
		info.Requirements = ParseRequirements(info.Description)
		info.Reward = ParseReward(info.Description)
		info.RewardEntries = BuildRewardEntries(RewardEntrySourceQuest, info.ID, info.Reward)
		if info.ID == "" || info.Title == "" {
			continue
		}
		infos = append(infos, info)
	}

	sort.SliceStable(infos, func(left int, right int) bool {
		if infos[left].Level != infos[right].Level {
			return infos[left].Level < infos[right].Level
		}
		return infos[left].Title < infos[right].Title
	})
	return infos, nil
}

func cloneInfos(infos []Info) []Info {
	result := make([]Info, len(infos))
	for index, info := range infos {
		result[index] = cloneInfo(info)
	}
	return result
}

func cloneInfo(info Info) Info {
	info.Routes = append([]AnswerRoute(nil), info.Routes...)
	info.Requirements = append([]RewardItem(nil), info.Requirements...)
	info.Reward.Items = append([]RewardItem(nil), info.Reward.Items...)
	info.Reward.Skills = append([]RewardSkill(nil), info.Reward.Skills...)
	info.Reward.OptionalItems = append([]RewardItem(nil), info.Reward.OptionalItems...)
	info.RewardEntries = cloneRewardEntries(info.RewardEntries)
	return info
}

func cloneRewardEntries(entries []RewardEntry) []RewardEntry {
	return append([]RewardEntry(nil), entries...)
}

func parseAnswerRoutes(value string) []AnswerRoute {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ";")
	routes := make([]AnswerRoute, 0, len(parts))
	for _, part := range parts {
		columns := strings.Split(part, "|")
		if len(columns) != 3 {
			continue
		}
		route := AnswerRoute{
			Handle:       strings.TrimSpace(columns[0]),
			MsgHandle:    strings.TrimSpace(columns[1]),
			AnswerHandle: strings.TrimSpace(columns[2]),
		}
		if route.Handle != "" && route.MsgHandle != "" && route.AnswerHandle != "" {
			routes = append(routes, route)
		}
	}
	return routes
}

func ParseRequirements(description string) []RewardItem {
	requirementText := sourceQuestRewardSection(description, "[x]=")
	if requirementText == "" {
		return []RewardItem{}
	}
	return parseSourceQuestRewardItems(requirementText)
}

func ParseReward(description string) Reward {
	reward := Reward{}
	grantText := sourceQuestRewardSection(description, "[g]=")
	if grantText != "" {
		reward.Experience = parseSourceQuestRewardExperience(grantText)
		reward.Items = parseSourceQuestRewardItems(grantText)
		reward.Skills = parseSourceQuestRewardSkills(grantText)
	}
	optionalText := sourceQuestRewardSection(description, "[s]=")
	if optionalText != "" {
		reward.OptionalItems = parseSourceQuestRewardItems(optionalText)
	}
	return reward
}

func BuildRewardEntries(sourceType string, sourceID string, reward Reward) []RewardEntry {
	sourceType = strings.TrimSpace(sourceType)
	sourceID = strings.TrimSpace(sourceID)
	entries := []RewardEntry{}
	if reward.Experience > 0 {
		entries = append(entries, RewardEntry{
			SourceType:  sourceType,
			SourceID:    sourceID,
			Phase:       RewardEntryPhaseGrant,
			Kind:        RewardEntryKindExperience,
			Name:        "经验",
			CountMin:    reward.Experience,
			CountMax:    reward.Experience,
			Probability: RewardEntryProbabilityCertain,
		})
	}
	for _, item := range reward.Items {
		if entry, ok := rewardEntryFromItem(sourceType, sourceID, RewardEntryPhaseGrant, "", item); ok {
			entries = append(entries, entry)
		}
	}
	for _, skill := range reward.Skills {
		name := strings.TrimSpace(skill.Name)
		if name == "" {
			continue
		}
		entries = append(entries, RewardEntry{
			SourceType:  sourceType,
			SourceID:    sourceID,
			Phase:       RewardEntryPhaseGrant,
			Kind:        RewardEntryKindSkill,
			Name:        name,
			CountMin:    1,
			CountMax:    1,
			Probability: RewardEntryProbabilityCertain,
		})
	}
	optionalGroup := ""
	if sourceID != "" {
		optionalGroup = sourceID + ":optional"
	}
	for _, item := range reward.OptionalItems {
		if entry, ok := rewardEntryFromItem(sourceType, sourceID, RewardEntryPhaseOptional, optionalGroup, item); ok {
			entries = append(entries, entry)
		}
	}
	return entries
}

func rewardEntryFromItem(sourceType string, sourceID string, phase RewardEntryPhase, chooseGroup string, item RewardItem) (RewardEntry, bool) {
	name := strings.TrimSpace(item.Name)
	if name == "" {
		return RewardEntry{}, false
	}
	count := item.Count
	if count <= 0 {
		count = 1
	}
	kind := RewardEntryKindItem
	if IsCurrencyRewardName(name) {
		kind = RewardEntryKindCurrency
	}
	return RewardEntry{
		SourceType:  sourceType,
		SourceID:    sourceID,
		Phase:       phase,
		Kind:        kind,
		Name:        name,
		CountMin:    count,
		CountMax:    count,
		Probability: RewardEntryProbabilityCertain,
		ChooseGroup: chooseGroup,
		Display:     item.Display,
		Description: item.Description,
		SourceMeta:  item.SourceMeta,
	}, true
}

func IsCurrencyRewardName(name string) bool {
	switch strings.TrimSpace(name) {
	case "铜钱", "银元宝":
		return true
	default:
		return false
	}
}

func IsCompletableState(state string) bool {
	return strings.Contains(state, "<over>")
}

func sourceQuestRewardSection(description string, marker string) string {
	start := strings.Index(description, marker)
	if start < 0 {
		return ""
	}
	start += len(marker)
	end := len(description)
	for _, nextMarker := range []string{"[g]=", "[s]=", "[x]="} {
		if nextIndex := strings.Index(description[start:], nextMarker); nextIndex >= 0 && start+nextIndex < end {
			end = start + nextIndex
		}
	}
	return strings.TrimSpace(description[start:end])
}

var sourceQuestExperiencePattern = regexp.MustCompile(`经验\s*([0-9]+)`)

func parseSourceQuestRewardExperience(text string) int {
	matches := sourceQuestExperiencePattern.FindStringSubmatch(text)
	if len(matches) < 2 {
		return 0
	}
	experience, _ := strconv.Atoi(matches[1])
	return experience
}

func parseSourceQuestRewardItems(text string) []RewardItem {
	items := []RewardItem{}
	searchStart := 0
	for {
		start := strings.Index(text[searchStart:], "[i=")
		if start < 0 {
			break
		}
		start += searchStart
		closeTag := strings.Index(text[start:], "[/]")
		if closeTag < 0 {
			break
		}
		closeTag += start
		rawItem := text[start+3 : closeTag]
		nameSep := strings.LastIndex(rawItem, "]")
		if nameSep < 0 || nameSep+1 >= len(rawItem) {
			searchStart = closeTag + len("[/]")
			continue
		}
		sourceMeta := strings.TrimSpace(rawItem[:nameSep])
		name := strings.TrimSpace(rawItem[nameSep+1:])
		count, afterCount := parseSourceQuestRewardItemCount(text[closeTag+len("[/]"):])
		if name != "" {
			items = append(items, RewardItem{
				Name:        name,
				Count:       count,
				Display:     parseSourceQuestRewardItemDisplay(sourceMeta),
				Description: sourceMeta,
				SourceMeta:  sourceMeta,
			})
		}
		searchStart = closeTag + len("[/]") + afterCount
	}
	return items
}

func parseSourceQuestRewardItemCount(text string) (int, int) {
	index := 0
	for index < len(text) && (text[index] == ' ' || text[index] == '\t' || text[index] == '\r' || text[index] == '\n') {
		index++
	}
	if index >= len(text) || (text[index] != 'x' && text[index] != 'X') {
		return 1, index
	}
	index++
	digitStart := index
	for index < len(text) && text[index] >= '0' && text[index] <= '9' {
		index++
	}
	if digitStart == index {
		return 1, index
	}
	count, _ := strconv.Atoi(text[digitStart:index])
	if count <= 0 {
		count = 1
	}
	return count, index
}

var sourceQuestItemDisplayPattern = regexp.MustCompile(`&101@([^&\]]+)`)

func parseSourceQuestRewardItemDisplay(sourceMeta string) string {
	matches := sourceQuestItemDisplayPattern.FindStringSubmatch(sourceMeta)
	if len(matches) < 2 {
		return ""
	}
	return strings.TrimSpace(matches[1])
}

var sourceQuestSkillPattern = regexp.MustCompile(`习得【([^】]+)】技能`)

func parseSourceQuestRewardSkills(text string) []RewardSkill {
	matches := sourceQuestSkillPattern.FindAllStringSubmatch(text, -1)
	skills := make([]RewardSkill, 0, len(matches))
	seen := map[string]bool{}
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		name := strings.TrimSpace(match[1])
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		skills = append(skills, RewardSkill{Name: name})
	}
	return skills
}
