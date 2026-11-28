package convention

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A subject that is not a Conventional Commit is refused, with the form it
// should take; one that is passes, breaking mark and scope included. git's
// own subjects pass on a commit and never on a title.
func TestASubjectThatIsNotConventionalIsRefused(t *testing.T) {
	cases := []struct {
		name, subject string
		title         bool
		refused       bool
	}{
		{"type and description", "fix: the checker reads a worktree as its own repository", false, false},
		{"scope", "feat(billing): an invoice names its payer", false, false},
		{"scope with a path", "feat(api/v1): CreateOrderRequest.customer", false, false},
		{"breaking", "feat(api/v1)!: customer is required", false, false},
		{"breaking without scope", "feat!: the queue is keyed by claim", false, false},
		{"design", "design: nothing admitted is ever deleted", false, false},
		{"the old form, an area and no type", "billing: an invoice names its payer", false, true},
		{"a sentence", "The check runs once, locally", false, true},
		{"no space after the colon", "fix:the checker", false, true},
		{"no description", "fix: ", false, true},
		{"an upper case type", "Fix: the checker", false, true},
		{"an upper case scope", "fix(Memory): the checker", false, true},
		{"an empty scope", "fix(): the checker", false, true},
		{"empty", "", false, true},
		{"a merge, on a commit", "Merge branch 'main' into feat/x", false, false},
		{"a revert, on a commit", `Revert "feat: a thing"`, false, false},
		{"a fixup, on a commit", "fixup! feat: a thing", false, false},
		{"a merge, as a title", "Merge branch 'main' into feat/x", true, true},
		{"a revert, as a title", `Revert "feat: a thing"`, true, true},
		{"a title", "feat(harness): a thing", true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			why := Refusal(c.subject, c.title)
			if c.refused && why == "" {
				t.Fatalf("%q was accepted", c.subject)
			}
			if !c.refused && why != "" {
				t.Fatalf("%q was refused: %s", c.subject, why)
			}
			if c.refused && c.subject != "" && !strings.Contains(why, "type(scope): description") {
				t.Fatalf("the refusal does not say what a subject is: %s", why)
			}
		})
	}
}

// The subject is the first line of the message that is not a comment, as git
// hands the message to the hook.
func TestTheSubjectIsTheMessagesFirstLine(t *testing.T) {
	if got := SubjectOf("\n# Please enter the commit message\nfeat: a thing\n\nBody.\n"); got != "feat: a thing" {
		t.Fatalf("got %q", got)
	}
	if got := SubjectOf("# only comments\n"); got != "" {
		t.Fatalf("got %q", got)
	}
}

// The convention binds a repository whose design asks for it and no other:
// one that has not adopted it yet commits as it did, and a value the harness
// does not know is an error, not a silent pass.
func TestTheConventionBindsOnlyARepositoryWhoseDesignAsksForIt(t *testing.T) {
	cases := []struct {
		name, frontmatter string
		binds, fails      bool
	}{
		{"asks", "files: [a/**]\ncommits: conventional\n", true, false},
		{"silent", "files: [a/**]\n", false, false},
		{"unknown", "files: [a/**]\ncommits: gitmoji\n", false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, "design"), 0o755); err != nil {
				t.Fatal(err)
			}
			doc := "---\n" + c.frontmatter + "---\n\n# x\n"
			if err := os.WriteFile(filepath.Join(root, "design", "DESIGN.md"), []byte(doc), 0o644); err != nil {
				t.Fatal(err)
			}
			binds, err := Binds(root)
			if (err != nil) != c.fails {
				t.Fatalf("err %v, expected failure %v", err, c.fails)
			}
			if binds != c.binds {
				t.Fatalf("binds %v, expected %v", binds, c.binds)
			}
		})
	}
}

// A change branch is feat/<what> or fix/<what> unless the design says
// otherwise, and then it is a name the whole of which matches the design's
// pattern, and feat/ is one only if the pattern says so. A pattern that does
// not compile is an error, not a repository with no rule.
func TestAChangeBranchIsNamedAsTheDesignAsksOrFeatOrFix(t *testing.T) {
	cases := []struct {
		name, frontmatter string
		own, fails        bool
		yes, no           []string
	}{
		{"silent", "files: [a/**]\n", false, false,
			[]string{"feat/a-thing", "fix/a/b"}, []string{"main", "feat/", "feat", "chore/a-thing", "claude/session-1", "x/feat/a-thing"}},
		{"a ticket first", "files: [a/**]\nbranches: '[A-Z]+-[0-9]+/.+'\n", true, false,
			[]string{"SHOP-12/a-thing"}, []string{"feat/a-thing", "SHOP-12", "my/SHOP-12/a-thing"}},
		{"the whole name, with alternatives", "files: [a/**]\nbranches: 'feature/.+|bugfix/.+'\n", true, false,
			[]string{"feature/a", "bugfix/b"}, []string{"feature/", "a-bugfix/b", "feat/a"}},
		{"not a pattern", "files: [a/**]\nbranches: '(feat'\n", false, true, nil, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, "design"), 0o755); err != nil {
				t.Fatal(err)
			}
			doc := "---\n" + c.frontmatter + "---\n\n# x\n"
			if err := os.WriteFile(filepath.Join(root, "design", "DESIGN.md"), []byte(doc), 0o644); err != nil {
				t.Fatal(err)
			}
			rule, err := Branches(root)
			if (err != nil) != c.fails {
				t.Fatalf("err %v, expected failure %v", err, c.fails)
			}
			if c.fails {
				if rule.Matches("feat/a-thing") {
					t.Fatal("a pattern that did not compile still passed a name")
				}
				return
			}
			if rule.Own != c.own {
				t.Errorf("own %v, expected %v", rule.Own, c.own)
			}
			if c.own && !strings.Contains(rule.Form(), rule.Pattern) {
				t.Errorf("the form does not give the pattern: %s", rule.Form())
			}
			for _, name := range c.yes {
				if !rule.Matches(name) {
					t.Errorf("%s was refused", name)
				}
			}
			for _, name := range c.no {
				if rule.Matches(name) {
					t.Errorf("%s was accepted", name)
				}
			}
		})
	}
}
