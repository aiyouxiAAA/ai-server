package quest

import (
	"regexp"
	"strings"
)

// Source records stay as evidence; these Classic quest answers are no longer live.
var retiredQuestAnswerPattern = regexp.MustCompile(`^(?:[0-9]+q[0-9]+|day_[0-9]+)`)

func IsRetiredAnswer(handle, text string) bool {
	return retiredQuestAnswerPattern.MatchString(handle) || strings.Contains(text, "<m/>") || strings.Contains(text, "<mn/>")
}
