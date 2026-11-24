package main

import (
	"testing"

	"harness/internal/design"
)

// The map is brought in line without merging anything: a repository that is
// missing is cloned, one on disk is fetched, one planned is skipped, and one
// with no remote is left as it is or reported missing.
func TestTheMapIsBroughtInLineWithoutMerging(t *testing.T) {
	cases := []struct {
		name   string
		entry  design.RepoEntry
		onDisk bool
		want   string
	}{
		{"missing", design.RepoEntry{Path: "billing", Remote: "git@host:x/billing.git"}, false, "clone"},
		{"on disk", design.RepoEntry{Path: "billing", Remote: "git@host:x/billing.git"}, true, "fetch"},
		{"planned", design.RepoEntry{Path: "ledger", Remote: "git@host:x/ledger.git", Status: "planned"}, false, "skip"},
		{"local only", design.RepoEntry{Path: "scratch"}, true, "local-only"},
		{"nowhere", design.RepoEntry{Path: "scratch"}, false, "missing"},
	}
	for _, c := range cases {
		if got, _ := plan(c.entry, c.onDisk); got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}
}
