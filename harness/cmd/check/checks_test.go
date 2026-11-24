package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"harness/internal/design"
)

func repoWith(t *testing.T, d design.Design, files map[string]string) *design.Repo {
	t.Helper()
	root := t.TempDir()
	var names []string
	for name, body := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
	}
	return &design.Repo{Root: root, Name: "x", Design: d, Files: names}
}

// Design before code: a file the design does not account for is refused, a
// file it does is counted, and the root's own files need no entry.
func TestEveryFileIsAccountedForOrRefused(t *testing.T) {
	repo := repoWith(t, design.Design{Files: []string{"app/**", "go.mod", "cmd/*/main.go"}}, map[string]string{
		"app/main.go": "package main", "go.mod": "module x", "stray.go": "package stray", "README.md": "#",
		"cmd/a/main.go": "package main", "cmd/a/deep/main.go": "package main",
	})
	res := runChecks(repo)
	if res.Claimed != 3 || len(res.Orphans) != 2 {
		t.Fatalf("claimed %d, orphans %v", res.Claimed, res.Orphans)
	}
	got := strings.Join(res.Failures, "\n")
	for _, want := range []string{"stray.go", "cmd/a/deep/main.go"} {
		if !strings.Contains(got, want+": the design accounts for no such file") {
			t.Fatalf("expected %s to be refused, got %v", want, res.Failures)
		}
	}
}

// A test the design names must exist in a test file the design accounts for,
// as a whole word; a name that is only a prefix of another does not count,
// and a name a runner is given as a string does.
func TestANamedTestMustExist(t *testing.T) {
	d := design.Design{Files: []string{"app/**"}, Tests: []string{"TestDecide", "TestMissing", "refuses an empty cart"}}
	repo := repoWith(t, d, map[string]string{
		"app/main.go":      "package main",
		"app/main_test.go": "package main\nfunc TestDecideTwice(t *testing.T) {}\nfunc TestDecide(t *testing.T) {}\n",
		"app/cart.test.ts": "it('refuses an empty cart', () => {})\n",
	})
	res := runChecks(repo)
	if len(res.Failures) != 1 || !strings.Contains(res.Failures[0], "TestMissing") {
		t.Fatalf("expected only TestMissing to be refused, got %v", res.Failures)
	}
}

// A design that names tests but accounts for no test file is itself the finding.
func TestNamedTestsWithNoTestFileIsRefused(t *testing.T) {
	d := design.Design{Files: []string{"app/**"}, Tests: []string{"TestX"}}
	repo := repoWith(t, d, map[string]string{"app/main.go": "package main"})
	res := runChecks(repo)
	if len(res.Failures) != 1 || !strings.Contains(res.Failures[0], "no test file") {
		t.Fatalf("got %v", res.Failures)
	}
}

// An id names one thing for life: one defined twice is refused, one cited
// and defined nowhere is refused, another repository's is its own to define,
// and a kind the document defines none of is not checked.
func TestAnIdIsDefinedOnceAndEveryCitationLands(t *testing.T) {
	cases := []struct {
		name, body string
		refused    []string
	}{
		{"lands", "## What must always hold\n\n- **INV-001** — a thing. *(ADR-0001)*\n\n### ADR-0001 — a decision\n\nSee INV-001.\n", nil},
		{"twice", "### ADR-0001 — one\n\n### ADR-0001 — another\n", []string{"defines ADR-0001 2 times"}},
		{"nowhere", "### ADR-0001 — one\n\nAs ADR-0007 says, and ADR-0007 again.\n", []string{"cites ADR-0007"}},
		{"an invariant cited and not defined", "- **INV-001** — a thing.\n\nINV-002 follows.\n", []string{"cites INV-002"}},
		{"another repository's", "### ADR-0001 — one\n\nThis follows platform/ADR-0029.\n", nil},
		{"a kind not defined here", "### ADR-0001 — one\n\nINV-009 is kept elsewhere for now.\n", nil},
		{"in a fence", "### ADR-0001 — one\n\n```\n### ADR-0001 — an example\nADR-0099\n```\n", nil},
		{"a heading for an invariant", "#### INV-003 — a thing\n\n- INV-003 again, in a list, is a second definition.\n", []string{"defines INV-003 2 times"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := &Result{}
			checkIDs(c.body, res)
			if len(res.Failures) != len(c.refused) {
				t.Fatalf("expected %d refusal(s), got %v", len(c.refused), res.Failures)
			}
			for i, want := range c.refused {
				if !strings.Contains(res.Failures[i], want) {
					t.Fatalf("expected %q, got %q", want, res.Failures[i])
				}
			}
		})
	}
}

// A design that still carries the reverse-engineering mark passes, and is
// said to be unfinished: the mark is a question for a person, not a fault,
// and the mark spoken of, as code, is not one left.
func TestAnUnfinishedDesignIsSaidToBeUnfinished(t *testing.T) {
	repo := repoWith(t, design.Design{Files: []string{"app/**"}}, map[string]string{"app/main.go": "package main"})
	repo.Body = "## Scope\n\nTODO(elephant): say what this is for.\n\n### ADR-0001 — Go (inferred) TODO(elephant)\n\nA mark is written `TODO(elephant)`, or `Not recorded. TODO(elephant)`.\n"
	res := runChecks(repo)
	if res.failed() {
		t.Fatalf("the mark is not a refusal: %v", res.Failures)
	}
	var out bytes.Buffer
	res.report(&out, repo)
	if !strings.Contains(out.String(), "still says TODO(elephant) in 2 place(s)") || !strings.Contains(out.String(), "harness: OK") {
		t.Fatalf("the report does not say the design is unfinished:\n%s", out.String())
	}
}
