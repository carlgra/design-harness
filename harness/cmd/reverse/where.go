package main

import (
	"os"
	"path/filepath"
	"strings"
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
