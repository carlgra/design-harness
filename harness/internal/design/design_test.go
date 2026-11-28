package design

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A hook run from a linked worktree is handed an absolute GIT_DIR. Until it
// is forgotten, a nested repository answers with the outer one's origin.
func TestForgetGitEnv(t *testing.T) {
	outer := t.TempDir()
	inner := filepath.Join(outer, "inner")
	git := func(dir string, args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git(outer, "init", "-q")
	git(outer, "remote", "add", "origin", "outer-remote")
	git(outer, "init", "-q", "inner")
	git(inner, "remote", "add", "origin", "inner-remote")

	t.Setenv("GIT_DIR", filepath.Join(outer, ".git"))
	if got := git(inner, "remote", "get-url", "origin"); got != "outer-remote" {
		t.Fatalf("expected the fault this guards against, got %q", got)
	}

	ForgetGitEnv()
	if got := git(inner, "remote", "get-url", "origin"); got != "inner-remote" {
		t.Errorf("nested origin = %q, want inner-remote", got)
	}
}

// A linked worktree under the root (git worktree add, as the agents use
// under .claude/worktrees/) is another checkout of this repository: not a
// nested repository, and not source the design must account for.
func TestAWorktreeIsNotANestedRepository(t *testing.T) {
	root := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		cmd.Env = append(cmd.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q", "-b", "main")
	run("commit", "-q", "--allow-empty", "-m", "root")
	run("worktree", "add", "-q", filepath.Join(root, ".claude", "worktrees", "other"), "-b", "other")
	if !IsWorktree(filepath.Join(root, ".claude", "worktrees", "other")) {
		t.Fatal("a linked worktree was not recognised")
	}
	if IsWorktree(root) {
		t.Fatal("the root is not a worktree")
	}
	r := &Repo{Root: root}
	if err := r.walk(); err != nil {
		t.Fatal(err)
	}
	for _, n := range r.NestedRepos {
		if strings.Contains(n, "worktrees") {
			t.Fatalf("the worktree was taken for a nested repository: %v", r.NestedRepos)
		}
	}
	for _, f := range r.Files {
		if strings.HasPrefix(f, ".claude/worktrees/") {
			t.Fatalf("the worktree's files were walked: %s", f)
		}
	}
}

// The marks left for a person are counted one way: in the body, and not a
// mark written as code. The checker and the drafting command both count
// with this, so they agree.
func TestTheMarksLeftForAPersonAreCountedOneWay(t *testing.T) {
	doc := "---\n# a comment that says TODO(design)\nfiles: []\n---\n\nTODO(design): one.\n\nWritten `Not recorded. TODO(design)` it is spoken of. Not recorded. TODO(design)\n"
	if n := Marks(doc); n != 2 {
		t.Fatalf("expected 2 marks, got %d", n)
	}
	if n := Marks("no frontmatter, TODO(design) once"); n != 1 {
		t.Fatalf("a body alone is counted too: got %d", n)
	}
}

func TestAGlobReachesThroughDirectories(t *testing.T) {
	for _, tc := range []struct {
		glob, path string
		want       bool
	}{
		{"src/**", "src/a/b.ts", true},
		{"src/**", "src", true},
		{"src/**", "srcs/a.ts", false},
		{"src/**/*.ts", "src/a.ts", true},
		{"src/**/*.ts", "src/a/b/c.ts", true},
		{"src/**/*.ts", "src/a/b/c.go", false},
		{"**/*.proto", "a.proto", true},
		{"**/*.proto", "api/v1/a.proto", true},
		{"**/internal/**", "pkg/cart/internal/x.go", true},
		{"**/internal/**", "pkg/cart/x.go", false},
		{"src/*.ts", "src/a/b.ts", false},
		{"*.md", "README.md", true},
		{"*.md", "docs/README.md", false},
	} {
		if got := MatchGlob(tc.glob, tc.path); got != tc.want {
			t.Errorf("MatchGlob(%q, %q) = %v, want %v", tc.glob, tc.path, got, tc.want)
		}
	}
}
