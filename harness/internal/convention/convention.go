// Package convention says whether a commit's subject is a Conventional
// Commit: `type(scope): description`, the scope optional, a `!` before the
// colon marking a breaking change. A changelog and a release can be generated
// from such subjects, so one that does not conform is a change the changelog
// never mentions.
//
// It binds a repository whose design says `commits: conventional` in its
// frontmatter, and nothing in one that does not.
package convention

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"harness/internal/design"
)

// Conventional is the value of `commits:` that binds a repository.
const Conventional = "conventional"

// Types are the types a subject may open with. feat and fix are a release's
// substance; design is a change to what is decided, which here is the source
// the code is derived from; the rest are the Conventional Commits types in
// common use.
var Types = []string{"feat", "fix", "design", "docs", "ci", "build", "chore", "refactor", "test", "perf", "style", "revert"}

var conventional = regexp.MustCompile(`^(` + strings.Join(Types, "|") + `)(\([a-z0-9][a-z0-9/_.-]*\))?!?: \S.*$`)

// git writes these subjects itself, or a person writes them for git to fold
// away before the branch is pushed; none of them is the subject of a change.
var gitsOwn = []string{"Merge ", `Revert "`, "fixup! ", "squash! ", "amend! "}

// Is reports whether a subject has the form, with nothing excused.
func Is(subject string) bool { return conventional.MatchString(strings.TrimSpace(subject)) }

// Refusal says why a subject is refused, or "" when it is accepted. A title
// is a pull request's: it is never one of git's own, because it is what lands
// on main when the pull request is squashed.
func Refusal(subject string, title bool) string {
	subject = strings.TrimSpace(subject)
	if !title {
		for _, p := range gitsOwn {
			if strings.HasPrefix(subject, p) {
				return ""
			}
		}
	}
	if subject == "" {
		return "the subject is empty"
	}
	if conventional.MatchString(subject) {
		return ""
	}
	return fmt.Sprintf("%q is not a Conventional Commit.\n"+
		"  A subject is  type(scope): description  with the scope optional, and ! before the colon for a breaking change:\n"+
		"      feat(billing): an invoice names its payer\n"+
		"      fix: the checker reads a worktree as its own repository\n"+
		"      feat(api/v1)!: CreateOrderRequest.customer is required\n"+
		"  The type is one of: %s.\n"+
		"  The scope is lower case: letters, digits and / _ . -",
		subject, strings.Join(Types, ", "))
}

// SubjectOf is the subject of a commit message as git hands it to a hook: its
// first line that is not a comment and not blank.
func SubjectOf(message string) string {
	for _, line := range strings.Split(message, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return line
	}
	return ""
}

// Binds reports whether the design at root asks for conventional subjects.
func Binds(root string) (bool, error) {
	fm, err := design.ReadFrontmatterFile(filepath.Join(root, filepath.FromSlash(design.DocPath)))
	if err != nil {
		return false, fmt.Errorf("%s: %w", design.DocPath, err)
	}
	var d design.Design
	if err := yaml.Unmarshal([]byte(fm.YAML), &d); err != nil {
		return false, fmt.Errorf("%s: frontmatter: %w", design.DocPath, err)
	}
	switch d.Commits {
	case "":
		return false, nil
	case Conventional:
		return true, nil
	}
	return false, fmt.Errorf("%s: commits: %q is not a convention the harness knows; the only one is %q", design.DocPath, d.Commits, Conventional)
}
