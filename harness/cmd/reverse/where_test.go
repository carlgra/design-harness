package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The file the rules go in is the one that holds the repository's standing
// instructions: AGENTS.md where CLAUDE.md is nothing but an import of it or a
// link to it, the file that already holds the method's rules where one does,
// and CLAUDE.md in every other case, one that imports AGENTS.md among rules
// of its own included.
func TestTheRulesFileIsTheOneThatHoldsTheInstructions(t *testing.T) {
	const method = "\n" + MethodBegin + "\nrules\n<!-- design-harness:end -->\n"
	cases := []struct {
		name           string
		claude, agents string // "" is no such file
		link           string // what CLAUDE.md links to, when it is a link
		want           string
	}{
		{"neither file", "", "", "", ClaudeFile},
		{"rules of its own", "# Mine\n\nAlways run make test.\n", "", "", ClaudeFile},
		{"rules of its own, and an AGENTS.md beside it", "# Mine\n\nAlways run make test.\n", "# Theirs\n", "", ClaudeFile},
		{"an import and nothing else", "@AGENTS.md\n", "# Theirs\n", "", AgentsFile},
		{"a heading and an import", "# CLAUDE.md\n\n@./AGENTS.md\n", "# Theirs\n", "", AgentsFile},
		{"an import among rules of its own", "@AGENTS.md\n\nPlan before editing.\n", "# Theirs\n", "", ClaudeFile},
		{"an import in a sentence", "See @AGENTS.md for the rules.\n", "# Theirs\n", "", ClaudeFile},
		{"a heading and no import", "# CLAUDE.md\n", "# Theirs\n", "", ClaudeFile},
		{"the method's rules already under an import", "@AGENTS.md\n" + method, "# Theirs\n", "", ClaudeFile},
		{"the method's rules already in AGENTS.md", "# Mine\n\nPlan before editing.\n", "# Theirs\n" + method, "", AgentsFile},
		{"a link to AGENTS.md", "", "# Theirs\n", "AGENTS.md", AgentsFile},
		{"a link to another file", "", "# Theirs\n", "docs/rules.md", ClaudeFile},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			write := func(name, body string) {
				t.Helper()
				p := filepath.Join(root, filepath.FromSlash(name))
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if c.agents != "" {
				write(AgentsFile, c.agents)
			}
			switch {
			case c.link != "":
				write("docs/rules.md", "# Elsewhere\n")
				if err := os.Symlink(c.link, filepath.Join(root, ClaudeFile)); err != nil {
					t.Skip("this file system has no links")
				}
			case c.claude != "":
				write(ClaudeFile, c.claude)
			}
			got, why := RulesFile(root)
			if got != c.want {
				t.Fatalf("the rules go in %s, want %s", got, c.want)
			}
			if (got == AgentsFile) != (why != "") {
				t.Errorf("AGENTS.md is chosen with a reason and CLAUDE.md with none; got %q for %s", why, got)
			}
		})
	}
}

// The runner is read from the workflows that are there: one plain label that
// every job agrees on is the answer; jobs that differ, a list, an expression
// or a workflow that does not parse are no answer, and what was found is
// said; no workflow at all is neither. The harness's own workflow and a job
// that only calls another workflow are not read.
func TestTheRunnerIsReadFromTheWorkflowsThatAreThere(t *testing.T) {
	job := func(runsOn string) string {
		return "name: ci\non: [push]\njobs:\n  build:\n    runs-on: " + runsOn + "\n    steps:\n      - run: make\n"
	}
	cases := []struct {
		name      string
		workflows map[string]string
		label     string
		check     string // a part of what is said, "" when nothing is
	}{
		{"no workflows", nil, "", ""},
		{"only the harness's own", map[string]string{"harness.yml": job("self-hosted")}, "", ""},
		{"one runner", map[string]string{"ci.yml": job("self-hosted"), "harness.yml": job("ubuntu-latest")}, "self-hosted", ""},
		{"every job agrees", map[string]string{"ci.yml": job("build-pool"), "release.yaml": job("'build-pool'") + "  ship:\n    runs-on: build-pool\n"}, "build-pool", ""},
		{"the default", map[string]string{"ci.yml": job("ubuntu-latest")}, "ubuntu-latest", ""},
		{"two runners", map[string]string{"ci.yml": job("self-hosted"), "mac.yml": job("macos-14")}, "", "macos-14 and self-hosted"},
		{"a list of labels", map[string]string{"ci.yml": job("[self-hosted, linux, x64]")}, "", "a list of labels"},
		{"a matrix", map[string]string{"ci.yml": job("${{ matrix.os }}")}, "", "an expression"},
		{"a label and an expression", map[string]string{"ci.yml": job("self-hosted"), "os.yml": job("${{ matrix.os }}")}, "", "self-hosted and an expression"},
		{"a group", map[string]string{"ci.yml": job("\n      group: big")}, "", "a value that is not one label"},
		{"a workflow that does not parse", map[string]string{"ci.yml": "jobs: [\n"}, "", "could not be read, ci.yml"},
		{"a job that calls a workflow", map[string]string{"ci.yml": job("self-hosted") + "  call:\n    uses: ./.github/workflows/other.yml\n"}, "self-hosted", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			dir := filepath.Join(root, filepath.FromSlash(WorkflowsDir))
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			for name, body := range c.workflows {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			label, check := RunsOn(root)
			if label != c.label {
				t.Errorf("the runner is %q, want %q", label, c.label)
			}
			if (c.check == "") != (check == "") || !strings.Contains(check, c.check) {
				t.Errorf("what is said is %q, want it to hold %q", check, c.check)
			}
		})
	}
}
