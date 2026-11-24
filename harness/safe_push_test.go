// Package harness holds no code of its own: the commands are under cmd/ and
// what they share under internal/. This file is here to walk safe-push, the
// script beside it, against a remote that is thrown away.
package harness

import (
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
}

func newProject(t *testing.T, spine string) *project {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	script, err := filepath.Abs("safe-push")
	if err != nil {
		t.Fatal(err)
	}
	tmp, _ := filepath.EvalSymlinks(t.TempDir())
	p := &project{t: t, dir: filepath.Join(tmp, "work"), remote: filepath.Join(tmp, "remote.git"), script: script}
	seed := filepath.Join(tmp, "seed")
	p.dir = seed
	p.write("design/DESIGN.md", "---\nfiles:\n  - app/**\n"+spine+"---\n\n# work\n\n## Scope\n\nA thing to push.\n")
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
	cmd.Env = env(dir)
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
