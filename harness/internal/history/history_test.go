package history

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func TestAChangeIsACommitAgainstItsFirstParent(t *testing.T) {
	root := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false"}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	write := func(name, s string) {
		p := filepath.Join(root, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(s), 0o644)
	}
	git("init", "-q", "-b", "main")
	write("a/one.txt", "one\n")
	git("add", "-A")
	git("commit", "-q", "-m", "first")
	git("checkout", "-q", "-b", "feat/x")
	write("a/two.txt", "two\n")
	git("add", "-A")
	git("commit", "-q", "-m", "step one")
	write("b/three.txt", "three\n")
	git("mv", "a/one.txt", "a/uno.txt")
	git("add", "-A")
	git("commit", "-q", "-m", "step two")
	git("checkout", "-q", "main")
	git("merge", "-q", "--no-ff", "-m", "the pull request", "feat/x")

	commits, err := Commits(root, 10)
	if err != nil {
		t.Fatal(err)
	}
	// The merge is one change holding the branch's two commits; the commits
	// of the branch are not changes of their own.
	if len(commits) != 2 || commits[0].Subject != "the pull request" || commits[1].Subject != "first" {
		t.Fatalf("commits: %+v", commits)
	}
	want := []string{"a/one.txt", "a/uno.txt", "a/two.txt", "b/three.txt"}
	got := commits[0].Paths()
	if len(got) != len(want) {
		t.Fatalf("paths %v, want %v", got, want)
	}
	seen := map[string]bool{}
	for _, p := range got {
		seen[p] = true
	}
	for _, p := range want {
		if !seen[p] {
			t.Fatalf("paths %v lack %s", got, p)
		}
	}
	if !reflect.DeepEqual(commits[1].Paths(), []string{"a/one.txt"}) {
		t.Fatalf("the first commit is set against the empty tree: %v", commits[1].Paths())
	}

	blobs, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer blobs.Close()
	if got := string(blobs.Read(commits[0].Hash, "a/uno.txt")); got != "one\n" {
		t.Fatalf("after: %q", got)
	}
	if got := blobs.Read(commits[0].Hash, "a/one.txt"); got != nil {
		t.Fatalf("a file that is gone was read: %q", got)
	}
	if got := blobs.Read(commits[0].Hash, "a"); got != nil {
		t.Fatalf("a directory was read as a file: %q", got)
	}
	if got := string(blobs.Read(commits[0].Parent, "a/one.txt")); got != "one\n" {
		t.Fatalf("before: %q", got)
	}

	pairs, of := Together(commits, Dir, 8)
	if len(pairs) != 1 || pairs[0].A != "a" || pairs[0].B != "b" || pairs[0].Together != 1 || of["a"] != 2 {
		t.Fatalf("together: %+v %v", pairs, of)
	}
}
