package main

import (
	"errors"
	"strings"

	"harness/internal/session"
	"testing"
)

// The script judges nothing itself: a session's reply is a pass
// only when it ends in the verdict form, one failure fails the run, and a
// session is handed its prompt, the design and, for readiness, the change —
// and nothing else. The model is faked; what these pin is the script.

func TestAVerdictIsReadFromTheLastLineAndNothingElseCounts(t *testing.T) {
	cases := []struct {
		reply       string
		pass, found bool
	}{
		{"Reasons.\n\nVERDICT: PASS\n", true, true},
		{"Reasons.\n\nVERDICT: FAIL", false, true},
		{"verdict: pass", false, false},                // the prefix is exact
		{"VERDICT: pass", true, true},                  // the word is not
		{"VERDICT: PASS\n\nMore after.", false, false}, // not the last line
		{"VERDICT: MAYBE", false, false},
		{"", false, false},
		{"I think it passes.", false, false},
	}
	for _, c := range cases {
		pass, found := parseVerdict(c.reply, "VERDICT:")
		if pass != c.pass || found != c.found {
			t.Errorf("%q: got pass=%v found=%v, want pass=%v found=%v", c.reply, pass, found, c.pass, c.found)
		}
	}
}

func fakeRunner(replies map[string]string, err error, seen *[]string) session.Runner {
	return func(model, prompt string) (string, error) {
		*seen = append(*seen, prompt)
		if err != nil {
			return "", err
		}
		for key, reply := range replies {
			if strings.HasPrefix(prompt, key) {
				return reply, nil
			}
		}
		return "", errors.New("no canned reply")
	}
}

var both = []test{
	{Name: "comprehension", Prompt: "COMPREHENSION PROMPT\n\n---\n"},
	{Name: "readiness", Prompt: "READINESS PROMPT\n\n---\n"},
}

func TestASessionIsHandedItsPromptTheDesignAndNothingElse(t *testing.T) {
	var seen []string
	r := fakeRunner(map[string]string{"COMPREHENSION": "ok\nVERDICT: PASS", "READINESS": "ok\nVERDICT: PASS"}, nil, &seen)
	head := "# x\n\nTHE DESIGN\n"
	change := "The change.\n"
	vs := run(both, head, change, session.Settings{Model: "m", Verdict: "VERDICT:"}, r)
	if len(seen) != 2 {
		t.Fatalf("expected two sessions, got %d", len(seen))
	}
	want := []string{
		"COMPREHENSION PROMPT\n\n---\n\n# x\n\nTHE DESIGN\n",
		"READINESS PROMPT\n\n---\n\n# x\n\nTHE DESIGN\n\n---\n\nThe change.\n",
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Errorf("session %d was handed:\n%q\nwant:\n%q", i, seen[i], want[i])
		}
	}
	for _, v := range vs {
		if !v.Pass || v.Err != nil {
			t.Errorf("%s: expected a pass, got %+v", v.Test, v)
		}
		if v.Model != "m" {
			t.Errorf("%s: the model is recorded with the verdict, got %q", v.Test, v.Model)
		}
	}
}

func TestOneFailureFailsTheRunAndIsReportedWithItsReasons(t *testing.T) {
	var seen []string
	r := fakeRunner(map[string]string{
		"COMPREHENSION": "fine\nVERDICT: PASS",
		"READINESS":     "The weight is pinned in ADR-0014 §3.\nVERDICT: FAIL",
	}, nil, &seen)
	vs := run(both, "head", "change", session.Settings{Model: "m", Verdict: "VERDICT:"}, r)
	failed := 0
	for _, v := range vs {
		if !v.Pass {
			failed++
			if v.Test != "readiness" || !strings.Contains(v.Reply, "pinned in ADR-0014") {
				t.Errorf("the failure is not the readiness session's with its reasons: %+v", v)
			}
		}
	}
	if failed != 1 {
		t.Fatalf("expected exactly one failure, got %d", failed)
	}
	md := markdown("x", vs)
	for _, want := range []string{"readiness: FAIL", "pinned in ADR-0014", "comprehension: PASS", "model m"} {
		if !strings.Contains(md, want) {
			t.Errorf("the markdown for the pull request does not say %q:\n%s", want, md)
		}
	}
}

func TestAReplyWithNoVerdictLineIsAFailureNotAPass(t *testing.T) {
	var seen []string
	r := fakeRunner(map[string]string{"COMPREHENSION": "I think this is fine."}, nil, &seen)
	vs := run(both[:1], "head", "change", session.Settings{Model: "m", Verdict: "VERDICT:"}, r)
	if vs[0].Pass || vs[0].Err == nil || !strings.Contains(vs[0].Err.Error(), "nothing else counts") {
		t.Fatalf("expected a failure for want of a verdict line, got %+v", vs[0])
	}
}

func TestASessionThatCannotRunIsAFailure(t *testing.T) {
	var seen []string
	r := fakeRunner(nil, errors.New("claude: not found"), &seen)
	vs := run(both[:1], "head", "change", session.Settings{Model: "m", Verdict: "VERDICT:"}, r)
	if vs[0].Pass || vs[0].Err == nil {
		t.Fatalf("expected a failure, got %+v", vs[0])
	}
	if !strings.Contains(headline(vs[0]), "not found") {
		t.Fatalf("the reason is not reported: %s", headline(vs[0]))
	}
}
