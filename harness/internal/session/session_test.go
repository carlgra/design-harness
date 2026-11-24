package session

import (
	"strings"
	"testing"
)

// A session's tools are always stated: a goldfish is started with none, by
// name, because leaving the flag out gives a session every tool; a surveying
// session is started with the ones that read and no other.
func TestASessionIsStartedWithExactlyTheToolsItIsGiven(t *testing.T) {
	none := strings.Join(Args("m", 0, nil), " ")
	if !strings.Contains(none, "--tools  --no-session-persistence") {
		t.Fatalf("a session with no tools was not started with an empty --tools: %q", none)
	}
	if strings.Contains(none, "--max-budget-usd") {
		t.Fatalf("a budget of 0 is no cap, got %q", none)
	}
	args := Args("m", 2.5, []string{"Read", "Grep", "Glob"})
	got := strings.Join(args, " ")
	for _, want := range []string{"-p", "--tools Read,Grep,Glob", "--model m", "--max-budget-usd 2.5"} {
		if !strings.Contains(got, want) {
			t.Errorf("args %q do not say %q", got, want)
		}
	}
	for _, a := range args {
		for _, writes := range []string{"Edit", "Write", "Bash"} {
			if strings.Contains(a, writes) {
				t.Errorf("a session that only reads was given %s", writes)
			}
		}
	}
}
