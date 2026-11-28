package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// What bootstrap asks of a repository it is about to write into. The scripts
// are shell and parse nothing, so each answer is worked out here and printed
// by the where step as a key=value line.

const (
	// ClaudeFile and AgentsFile are the two files at a repository's root an
	// agent is handed as its standing instructions.
	ClaudeFile = "CLAUDE.md"
	AgentsFile = "AGENTS.md"
	// MethodBegin opens the method's rules where bootstrap wrote them.
	MethodBegin = "<!-- design-harness:begin -->"
)

// RulesFile is the file that holds a repository's standing instructions, and
// so the one the method's rules and the project's own section are written to,
// with why when it is not CLAUDE.md. It is AGENTS.md where CLAUDE.md is a link
// to it or holds nothing but an import of it: rules added to a file that only
// points elsewhere are read by one agent and missed by every other. A file
// that already holds the method's rules is the one, so that an update finds
// them where they are. Anything else is CLAUDE.md.
func RulesFile(root string) (name, why string) {
	claude := filepath.Join(root, ClaudeFile)
	if fi, err := os.Lstat(claude); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(claude)
		if err != nil {
			return ClaudeFile, ""
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(root, target)
		}
		if filepath.Clean(target) == filepath.Join(root, AgentsFile) {
			return AgentsFile, ClaudeFile + " is a link to it"
		}
		return ClaudeFile, ""
	}
	own, _ := os.ReadFile(claude)
	if strings.Contains(string(own), MethodBegin) {
		return ClaudeFile, ""
	}
	if agents, _ := os.ReadFile(filepath.Join(root, AgentsFile)); strings.Contains(string(agents), MethodBegin) {
		return AgentsFile, "the method's rules are already there"
	}
	if onlyImports(string(own), AgentsFile) {
		return AgentsFile, ClaudeFile + " only imports it"
	}
	return ClaudeFile, ""
}

// onlyImports reports whether a document is nothing but an import of the file
// named, `@AGENTS.md` on a line of its own, under headings or none: every
// other line is blank.
func onlyImports(doc, name string) bool {
	imports := false
	for _, line := range strings.Split(doc, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "" || strings.HasPrefix(line, "#"):
		case line == "@"+name || line == "@./"+name:
			imports = true
		default:
			return false
		}
	}
	return imports
}

// WorkflowsDir is where a repository's CI workflows live, and HarnessWorkflow
// the one bootstrap writes there.
const (
	WorkflowsDir    = ".github/workflows"
	HarnessWorkflow = "harness.yml"
)

// plainLabel is a runner named by one word, ubuntu-latest or self-hosted: a
// value that can be written into another workflow as it stands.
var plainLabel = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// RunsOn reads what the jobs of the repository's own workflows run on, so
// that the workflow bootstrap writes runs where they do. label is the one
// runner every job names, when every job names the same one as a plain
// label. Where they differ, or one is not a plain label, a list, an
// expression or a matrix's value, label is empty and check says what was
// found: bootstrap then keeps its own runner and says the line is one to
// check. With no workflow to read both are empty. The harness's own workflow
// is not read: it is the one being written.
func RunsOn(root string) (label, check string) {
	labels, others := map[string]bool{}, map[string]bool{}
	entries, _ := os.ReadDir(filepath.Join(root, filepath.FromSlash(WorkflowsDir)))
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || name == HarnessWorkflow || !(strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml")) {
			continue
		}
		var w struct {
			Jobs map[string]struct {
				RunsOn yaml.Node `yaml:"runs-on"`
			} `yaml:"jobs"`
		}
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(WorkflowsDir), name))
		if err == nil {
			err = yaml.Unmarshal(b, &w)
		}
		if err != nil {
			others["a workflow that could not be read, "+name] = true
			continue
		}
		for _, job := range w.Jobs {
			switch n := job.RunsOn; {
			case n.Kind == 0:
				// A job that calls another workflow names no runner.
			case n.Kind == yaml.ScalarNode && plainLabel.MatchString(n.Value):
				labels[n.Value] = true
			case n.Kind == yaml.ScalarNode && strings.Contains(n.Value, "${{"):
				others["an expression"] = true
			case n.Kind == yaml.SequenceNode:
				others["a list of labels"] = true
			default:
				others["a value that is not one label"] = true
			}
		}
	}
	if len(labels) == 1 && len(others) == 0 {
		for l := range labels {
			return l, ""
		}
	}
	if len(labels)+len(others) == 0 {
		return "", ""
	}
	found := append(sortedKeys(labels), sortedKeys(others)...)
	return "", "this repository's workflows run on " + oxford(found)
}

func sortedKeys(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// oxford joins a list as a sentence would: a, b and c.
func oxford(items []string) string {
	if len(items) < 2 {
		return strings.Join(items, "")
	}
	return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
}
