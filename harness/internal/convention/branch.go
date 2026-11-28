package convention

import (
	"fmt"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"

	"harness/internal/design"
)

// DefaultBranches is the pattern a change branch's name matches in a
// repository whose design does not set one: feat/<what> or fix/<what>.
const DefaultBranches = `(feat|fix)/.+`

// BranchRule is how a change branch is named in one repository.
type BranchRule struct {
	// Pattern is the regular expression the whole of a name must match.
	Pattern string
	// Own is true when the design's frontmatter set it, with `branches:`.
	Own bool
	re  *regexp.Regexp
}

// Branches reads how a change branch is named in the repository at root: the
// pattern its design's frontmatter gives under `branches:`, or
// DefaultBranches when it gives none. A pattern that is not a regular
// expression is an error, never read as no pattern.
func Branches(root string) (BranchRule, error) {
	fm, err := design.ReadFrontmatterFile(filepath.Join(root, filepath.FromSlash(design.DocPath)))
	if err != nil {
		return BranchRule{}, fmt.Errorf("%s: %w", design.DocPath, err)
	}
	var d design.Design
	if err := yaml.Unmarshal([]byte(fm.YAML), &d); err != nil {
		return BranchRule{}, fmt.Errorf("%s: frontmatter: %w", design.DocPath, err)
	}
	return BranchesOf(d)
}

// BranchesOf is Branches for frontmatter already read.
func BranchesOf(d design.Design) (BranchRule, error) {
	rule := BranchRule{Pattern: DefaultBranches}
	if d.Branches != "" {
		rule = BranchRule{Pattern: d.Branches, Own: true}
	}
	re, err := regexp.Compile(`^(?:` + rule.Pattern + `)$`)
	if err != nil {
		return BranchRule{}, fmt.Errorf("%s: branches: %q is not a regular expression: %v", design.DocPath, rule.Pattern, err)
	}
	rule.re = re
	return rule, nil
}

// Matches reports whether name, the whole of it, is a change branch's name.
func (b BranchRule) Matches(name string) bool { return b.re != nil && b.re.MatchString(name) }

// Form says in words what a change branch is called here, for a message.
func (b BranchRule) Form() string {
	if !b.Own {
		return "feat/<what> or fix/<what>"
	}
	return fmt.Sprintf("to match %s (%s, branches:)", b.Pattern, design.DocPath)
}
