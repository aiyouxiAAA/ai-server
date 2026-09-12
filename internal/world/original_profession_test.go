package world

import (
	"strings"
	"testing"
)

func TestBladeDancerReplacesTeacherProfessionBranches(t *testing.T) {
	reply := BuildAnswerReply("1000542608713897", "1", "2")
	if reply == nil || len(reply.Answers) != 2 || reply.Answers[0].Handle != "job_blade-dancer" {
		t.Fatalf("profession menu: %+v", reply)
	}
	for _, handle := range []string{"1000542608713897", "2220542612946566", "5040542617131880"} {
		reply := BuildAnswerReply(handle, "1", "1")
		if reply == nil || !strings.Contains(reply.Msg, "格挡为被动") || len(reply.Answers) != 1 {
			t.Fatalf("old skills exposed for %s: %+v", handle, reply)
		}
	}
}
