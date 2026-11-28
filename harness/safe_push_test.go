// Package harness holds no code of its own: the commands are under cmd/ and
// what they share under internal/. This file is here to walk safe-push, the
// script beside it, against a remote that is thrown away.
package harness

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// project is a repository with a design that passes, cloned from a bare
// remote whose main has one commit: what safe-push expects to find.
type project struct {
	t      *testing.T
	dir    string
	remote string
	script string
	// extra is added to the environment of everything the project runs.
	extra []string
}

func newProject(t *testing.T, frontmatter string) *project {
	t.Helper()
	return newProjectAt(t, frontmatter, "remote.git")
}

// newProjectAt is newProject with the remote at a path of the caller's
// choosing under the temporary directory: one that reads as a GitHub address
// is how a test stands in for a repository there.
func newProjectAt(t *testing.T, frontmatter, remote string) *project {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	script, err := filepath.Abs("safe-push")
	if err != nil {
		t.Fatal(err)
	}
	tmp, _ := filepath.EvalSymlinks(t.TempDir())
	p := &project{t: t, dir: filepath.Join(tmp, "work"), remote: filepath.Join(tmp, filepath.FromSlash(remote)), script: script}
	if err := os.MkdirAll(filepath.Dir(p.remote), 0o755); err != nil {
		t.Fatal(err)
	}
	seed := filepath.Join(tmp, "seed")
	p.dir = seed
	p.write("design/DESIGN.md", "---\nfiles:\n  - app/**\n"+frontmatter+"---\n\n# work\n\n## Scope\n\nA thing to push.\n")
	p.write("app/main.txt", "one\ntwo\nthree\n")
	p.write("README.md", "# work\n")
	p.git("init", "-q", "-b", "main")
	p.git("add", "-A")
	p.git("commit", "-q", "-m", "feat: the first commit")
	p.exec(tmp, "git", "init", "-q", "--bare", "-b", "main", p.remote)
	p.git("push", "-q", p.remote, "main")
	p.dir = filepath.Join(tmp, "work")
	p.exec(tmp, "git", "clone", "-q", p.remote, p.dir)
	return p
}

func (p *project) write(name, body string) {
	p.t.Helper()
	f := filepath.Join(p.dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
		p.t.Fatal(err)
	}
	if err := os.WriteFile(f, []byte(body), 0o644); err != nil {
		p.t.Fatal(err)
	}
}

// env is the environment without the repository git handed this process: a
// test run from a hook must not act on the hook's repository.
func env(dir string) []string {
	var out []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "GIT_") && !strings.HasPrefix(kv, "PWD=") {
			out = append(out, kv)
		}
	}
	return append(out, "PWD="+dir, "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
}

func (p *project) exec(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(env(dir), p.extra...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (p *project) git(args ...string) string {
	p.t.Helper()
	out, err := p.exec(p.dir, "git", args...)
	if err != nil {
		p.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(out)
}

// commit changes one line of the file and commits it.
func (p *project) commit(subject, body string) {
	p.t.Helper()
	p.write("app/main.txt", body)
	p.git("add", "-A")
	p.git("commit", "-q", "-m", subject)
}

// push runs the script, without a pull request: there is no server to open
// one on.
func (p *project) push() (string, error) {
	return p.exec(p.dir, "sh", p.script, "--no-pr")
}

// branches are the remote's, as "name name name".
func (p *project) branches() string {
	p.t.Helper()
	out, err := p.exec(p.remote, "git", "for-each-ref", "--format=%(refname:short)", "refs/heads")
	if err != nil {
		p.t.Fatal(err, out)
	}
	return strings.Join(strings.Fields(out), " ")
}

func (p *project) remoteMain() string {
	p.t.Helper()
	out, err := p.exec(p.remote, "git", "rev-parse", "main")
	if err != nil {
		p.t.Fatal(err, out)
	}
	return strings.TrimSpace(out)
}

// refused runs the script and requires that it refused, said why, and left
// the remote exactly as it was.
func (p *project) refused(why string) {
	p.t.Helper()
	before, main := p.branches(), p.remoteMain()
	out, err := p.push()
	if err == nil {
		p.t.Fatalf("the push was not refused:\n%s", out)
	}
	if !strings.Contains(out, why) {
		p.t.Errorf("the refusal does not say %q:\n%s", why, out)
	}
	if got := p.branches(); got != before || p.remoteMain() != main {
		p.t.Errorf("a refused push changed the remote: %q, was %q", got, before)
	}
}

// A change branch that passes is pushed, and main is not touched.
func TestAChangeBranchThatPassesIsPushed(t *testing.T) {
	p := newProject(t, "")
	main := p.remoteMain()
	p.git("checkout", "-q", "-b", "feat/a-fourth-line")
	p.commit("feat: a fourth line", "one\ntwo\nthree\nfour\n")
	out, err := p.push()
	if err != nil {
		t.Fatalf("the push failed: %v\n%s", err, out)
	}
	if got := p.branches(); got != "feat/a-fourth-line main" {
		t.Errorf("the remote has %q", got)
	}
	if p.remoteMain() != main {
		t.Error("main moved")
	}
}

// Nothing is ever pushed to main, nor a branch that is not named for a
// change: work found on one is moved to a branch named for it, and that is
// what is pushed. The name is the first commit's subject, whole.
func TestWorkOnMainOrAStrayBranchIsMovedAndMainIsNeverPushed(t *testing.T) {
	cases := []struct{ name, from, subject, want string }{
		{"on main", "main", "fix(app): mend the third line", "fix/mend-the-third-line"},
		{"on a branch someone else named", "claude/session-123", "Add a fourth line", "feat/add-a-fourth-line"},
		{"a long subject is cut at a word", "main", "feat: a subject that goes on for rather longer than a branch name should", "feat/a-subject-that-goes-on-for-rather-longer"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := newProject(t, "")
			main := p.remoteMain()
			if c.from != "main" {
				p.git("checkout", "-q", "-b", c.from)
			}
			p.commit(c.subject, "one\ntwo\nthree\nfour\n")
			out, err := p.push()
			if err != nil {
				t.Fatalf("the push failed: %v\n%s", err, out)
			}
			if got := p.branches(); got != c.want+" main" {
				t.Errorf("the remote has %q, want %q beside main\n%s", got, c.want, out)
			}
			if p.remoteMain() != main {
				t.Error("main moved")
			}
			if got := p.git("rev-parse", "--abbrev-ref", "HEAD"); got != c.want {
				t.Errorf("left on %q", got)
			}
		})
	}
}

// A branch whose pull request has landed is not pushed to again: new work on
// it is moved to a branch of its own.
func TestABranchAlreadyMergedIsNotPushedToAgain(t *testing.T) {
	p := newProject(t, "")
	p.git("checkout", "-q", "-b", "feat/a-fourth-line")
	p.commit("feat: a fourth line", "one\ntwo\nthree\nfour\n")
	if out, err := p.push(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	// It merges, on the server.
	if out, err := p.exec(p.remote, "git", "update-ref", "refs/heads/main", "refs/heads/feat/a-fourth-line"); err != nil {
		t.Fatal(err, out)
	}
	merged := p.git("rev-parse", "HEAD")
	p.commit("feat: a fifth line", "one\ntwo\nthree\nfour\nfive\n")
	out, err := p.push()
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out, "already been merged") {
		t.Errorf("the script did not say the branch had merged:\n%s", out)
	}
	if got := p.branches(); got != "feat/a-fifth-line feat/a-fourth-line main" {
		t.Errorf("the remote has %q", got)
	}
	if got, _ := p.exec(p.remote, "git", "rev-parse", "feat/a-fourth-line"); strings.TrimSpace(got) != merged {
		t.Error("the merged branch was pushed to")
	}
}

// What is pushed is what was validated: a tree with uncommitted work is
// refused.
func TestADirtyTreeIsRefused(t *testing.T) {
	p := newProject(t, "")
	p.git("checkout", "-q", "-b", "feat/a-fourth-line")
	p.commit("feat: a fourth line", "one\ntwo\nthree\nfour\n")
	p.write("app/main.txt", "one\ntwo\nthree\nfour\nand not committed\n")
	p.refused("Working tree is dirty")
}

// A branch that will not merge is refused here, with the file named, and not
// minutes later on the server.
func TestABranchThatDoesNotMergeCleanlyIsRefused(t *testing.T) {
	p := newProject(t, "")
	p.git("checkout", "-q", "-b", "feat/mine")
	p.commit("feat: mine", "one\nMINE\nthree\n")
	// Someone else's change to the same line lands on main first.
	other := filepath.Join(filepath.Dir(p.dir), "other")
	if out, err := p.exec(filepath.Dir(p.dir), "git", "clone", "-q", p.remote, other); err != nil {
		t.Fatal(err, out)
	}
	if err := os.WriteFile(filepath.Join(other, "app", "main.txt"), []byte("one\nTHEIRS\nthree\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"commit", "-q", "-am", "feat: theirs"}, {"push", "-q", "origin", "main"}} {
		if out, err := p.exec(other, "git", args...); err != nil {
			t.Fatal(err, out)
		}
	}
	before, main := p.branches(), p.remoteMain()
	out, err := p.push()
	if err == nil || !strings.Contains(out, "does not merge cleanly") || !strings.Contains(out, "app/main.txt") {
		t.Fatalf("a conflict was not refused with its file named (%v):\n%s", err, out)
	}
	if p.branches() != before || p.remoteMain() != main {
		t.Error("a refused push changed the remote")
	}
}

// A branch the checker refuses is not pushed.
func TestABranchTheCheckerRefusesIsNotPushed(t *testing.T) {
	p := newProject(t, "")
	p.git("checkout", "-q", "-b", "feat/a-stray-file")
	p.write("stray/file.txt", "under no glob the design lists\n")
	p.git("add", "-A")
	p.git("commit", "-q", "-m", "feat: a stray file")
	p.refused("The checker failed. Nothing pushed.")
}

// Where the design asks for Conventional Commits, a branch whose first
// commit would give the pull request a title that is not one is refused; in
// a repository that has not asked, the same subject is pushed.
func TestATitleThatIsNotConventionalIsRefusedWhereTheDesignAsks(t *testing.T) {
	p := newProject(t, "commits: conventional\n")
	p.git("checkout", "-q", "-b", "feat/a-fourth-line", "--no-track", "origin/main")
	p.write("app/main.txt", "one\ntwo\nthree\nfour\n")
	p.git("add", "-A")
	p.git("commit", "-q", "--no-verify", "-m", "Added a fourth line")
	p.refused("is not a Conventional Commit")

	q := newProject(t, "")
	q.git("checkout", "-q", "-b", "feat/a-fourth-line")
	q.commit("Added a fourth line", "one\ntwo\nthree\nfour\n")
	if out, err := q.push(); err != nil {
		t.Fatalf("a repository that has not asked for the convention was held to it: %v\n%s", err, out)
	}
}

// fakeGH puts a gh on the project's PATH that answers the three reads
// cmd/protection makes: the repository, the branch's protection as given
// (empty is a branch with none), and no rulesets.
func (p *project) fakeGH(protection string) {
	p.t.Helper()
	bin := filepath.Join(filepath.Dir(p.dir), "bin")
	reply := filepath.Join(bin, "protection.json")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		p.t.Fatal(err)
	}
	if err := os.WriteFile(reply, []byte(protection), 0o644); err != nil {
		p.t.Fatal(err)
	}
	script := fmt.Sprintf(`#!/bin/sh
for a in "$@"; do path="$a"; done
case "$path" in
  repos/acme/work) echo '{}' ;;
  repos/acme/work/branches/main/protection)
    if [ -s %[1]q ]; then cat %[1]q; else
      echo '{"message":"Branch not protected"}'; echo 'gh: Branch not protected (HTTP 404)' >&2; exit 1
    fi ;;
  repos/acme/work/rules/branches/main) echo '[]' ;;
  *) echo "gh: Not Found (HTTP 404)" >&2; exit 1 ;;
esac
`, reply)
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		p.t.Fatal(err)
	}
	p.extra = []string{"PATH=" + bin + string(os.PathListSeparator) + os.Getenv("PATH")}
}

// Where the design asks for it, a push is refused while the server's
// protection of main lacks something, and says what; with all of it there
// the push goes through; and a repository that has not asked has nothing
// read, whatever the server holds.
func TestAPushIsRefusedWhileTheServersProtectionLacksSomething(t *testing.T) {
	const there = `{"required_status_checks": {"contexts": ["harness"]}, "required_pull_request_reviews": {}, "allow_force_pushes": {"enabled": false}}`

	p := newProjectAt(t, "protection: required\n", "github.com/acme/work.git")
	p.git("checkout", "-q", "-b", "feat/a-fourth-line")
	p.commit("feat: a fourth line", "one\ntwo\nthree\nfour\n")
	p.fakeGH("")
	p.refused("a pull request is not required")
	p.fakeGH(`{"required_pull_request_reviews": {}, "allow_force_pushes": {"enabled": false}}`)
	p.refused("the `harness` status check is not required")
	p.fakeGH(there)
	if out, err := p.push(); err != nil || !strings.Contains(p.branches(), "feat/a-fourth-line") {
		t.Fatalf("a push to a protected repository was refused: %v\n%s", err, out)
	}

	q := newProjectAt(t, "", "github.com/acme/work.git")
	q.git("checkout", "-q", "-b", "feat/a-fourth-line")
	q.commit("feat: a fourth line", "one\ntwo\nthree\nfour\n")
	q.fakeGH("")
	if out, err := q.push(); err != nil || strings.Contains(out, "protection") {
		t.Fatalf("a repository that has not asked had its protection read: %v\n%s", err, out)
	}
}

// Where the design's frontmatter says how a change branch is named, a branch
// named so is pushed as it is, and feat/ is no longer one. Work that must
// move goes to a name the script can build that matches; where the pattern
// asks for something it cannot know, a ticket number, it refuses, says what
// name is needed, and moves and pushes nothing.
func TestABranchIsNamedAsTheDesignAsksAndStrayWorkIsMovedOnlyToSuchAName(t *testing.T) {
	const tickets = "branches: '[A-Z]+-[0-9]+/.+'\n"

	t.Run("a branch named as the design asks is pushed", func(t *testing.T) {
		p := newProject(t, tickets)
		p.git("checkout", "-q", "-b", "SHOP-12/a-fourth-line")
		p.commit("feat: a fourth line", "one\ntwo\nthree\nfour\n")
		if out, err := p.push(); err != nil {
			t.Fatalf("the push failed: %v\n%s", err, out)
		}
		if got := p.branches(); got != "SHOP-12/a-fourth-line main" {
			t.Errorf("the remote has %q", got)
		}
	})

	t.Run("no name can be built, so nothing is moved", func(t *testing.T) {
		for _, from := range []string{"main", "feat/a-fourth-line"} {
			p := newProject(t, tickets)
			if from != "main" {
				p.git("checkout", "-q", "-b", from)
			}
			p.commit("feat: a fourth line", "one\ntwo\nthree\nfour\n")
			tip := p.git("rev-parse", "HEAD")
			p.refused("safe-push cannot name a branch for this work")
			out, _ := p.push()
			for _, want := range []string{"[A-Z]+-[0-9]+/.+", "design/DESIGN.md, branches:", "git switch -c"} {
				if !strings.Contains(out, want) {
					t.Errorf("from %s, the refusal does not say %q:\n%s", from, want, out)
				}
			}
			if got := p.git("rev-parse", "--abbrev-ref", "HEAD"); got != from || p.git("rev-parse", "HEAD") != tip {
				t.Errorf("from %s, the work was moved: on %q", from, got)
			}
			if got := p.git("branch", "--format=%(refname:short)"); strings.Contains(got, "a-fourth-line") && from == "main" {
				t.Errorf("a branch was made that could not be pushed: %q", got)
			}
		}
	})

	t.Run("a name that matches is built where one can be", func(t *testing.T) {
		cases := []struct{ pattern, want string }{
			{"(feat|fix|chore)/[a-z0-9-]+", "feat/a-fourth-line"},
			{"[a-z0-9-]+", "a-fourth-line"},
		}
		for _, c := range cases {
			p := newProject(t, "branches: '"+c.pattern+"'\n")
			main := p.remoteMain()
			p.commit("feat: a fourth line", "one\ntwo\nthree\nfour\n")
			out, err := p.push()
			if err != nil {
				t.Fatalf("%s: the push failed: %v\n%s", c.pattern, err, out)
			}
			if got := p.branches(); got != c.want+" main" || p.remoteMain() != main {
				t.Errorf("%s: the remote has %q, want %q beside main\n%s", c.pattern, got, c.want, out)
			}
		}
	})

	t.Run("a pattern that is not one pushes nothing", func(t *testing.T) {
		p := newProject(t, "branches: '(feat'\n")
		p.git("checkout", "-q", "-b", "feat/a-fourth-line")
		p.commit("feat: a fourth line", "one\ntwo\nthree\nfour\n")
		p.refused("is not a regular expression")
	})
}
