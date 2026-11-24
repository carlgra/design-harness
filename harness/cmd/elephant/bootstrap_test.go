package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Bootstrapping a project that already exists leaves it passing the checker
// with nothing but its own harness, adds only what was missing, and run a
// second time changes nothing. The script is the method's own repository's,
// beside harness/; a harness copied into a project has no bootstrap beside
// it, and there this test has nothing to walk.
func TestBootstrappingAnExistingProjectLeavesItPassingAndOverwritesNothing(t *testing.T) {
	script, err := filepath.Abs(filepath.Join("..", "..", "..", "bootstrap"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(script); err != nil {
		t.Skip("no bootstrap beside this harness: it is a copy in a project")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	root := t.TempDir()
	root, _ = filepath.EvalSymlinks(root)
	mine := "# Mine\n\nAlways run make test.\n"
	for name, body := range map[string]string{
		"README.md":                  "# shop\n",
		"CLAUDE.md":                  mine,
		"go.mod":                     "module shop\n\ngo 1.22\n",
		"Makefile":                   "test:\n\tgo test ./...\n",
		"cmd/shop/main.go":           "package main\n\nfunc main() {}\n",
		"internal/cart/cart.go":      "package cart\n",
		"internal/cart/cart_test.go": "package cart\n\nimport \"testing\"\n\nfunc TestTotalSums(t *testing.T) {}\n",
		".hidden/notes.txt":          "a directory whose name YAML would not read plainly\n",
	} {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q", "-b", "main")
	git("add", "-A")
	git("commit", "-q", "-m", "feat: a shop")

	bootstrap := func() string {
		t.Helper()
		cmd := exec.Command("sh", script, root)
		// A test run from a hook is handed the hook's repository; the script
		// must find the project's own.
		for _, kv := range os.Environ() {
			if !strings.HasPrefix(kv, "GIT_") {
				cmd.Env = append(cmd.Env, kv)
			}
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("bootstrap: %v\n%s", err, out)
		}
		return string(out)
	}
	read := func(name string) string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}

	out := bootstrap()
	if !strings.Contains(out, "harness: OK") || strings.Contains(out, "FAIL ") {
		t.Fatalf("the project does not pass its checker after bootstrap:\n%s", out)
	}
	doc := read("design/DESIGN.md")
	for _, want := range []string{SkeletonMark, "  - cmd/**\n", "  - harness/**\n", "  - .github/**\n", "  - .hidden/**\n", "# validate: make test", "# " + filepath.Base(root) + "\n"} {
		if !strings.Contains(doc, want) {
			t.Errorf("the skeleton does not hold %q:\n%s", want, doc)
		}
	}
	claude := read("CLAUDE.md")
	if !strings.HasPrefix(claude, mine) {
		t.Errorf("the project's own CLAUDE.md was not kept as it was:\n%s", claude)
	}
	for _, want := range []string{"design-harness:begin", "Design before code", ProjectBegin} {
		if !strings.Contains(claude, want) {
			t.Errorf("CLAUDE.md does not hold %q", want)
		}
	}
	if strings.Contains(claude, "{{NAME}}") || strings.Contains(claude, "# Working in") {
		t.Errorf("CLAUDE.md was given a second title or an unfilled name:\n%s", claude)
	}
	notes, _ := filepath.Glob(filepath.Join(root, "design", "changes", "*-adopt-the-design.md"))
	if len(notes) != 1 {
		t.Fatalf("expected the first note, found %v", notes)
	}
	for _, f := range []string{"harness/FRAMEWORK.md", "harness/safe-push", "harness/cmd/check/main.go", ".github/workflows/harness.yml"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(f))); err != nil {
			t.Errorf("%s was not written", f)
		}
	}
	if got := git("config", "--get", "core.hooksPath"); got != "harness/hooks" {
		t.Errorf("core.hooksPath = %q", got)
	}
	if got := git("log", "--oneline"); strings.Count(got, "\n") != 0 {
		t.Errorf("bootstrap committed something:\n%s", got)
	}

	// A person takes the design on; a second run touches none of it.
	doc = strings.Replace(doc, SkeletonMark+"\n", "", 1) + "\nWritten by someone.\n"
	os.WriteFile(filepath.Join(root, "design", "DESIGN.md"), []byte(doc), 0o644)
	out = bootstrap()
	if read("design/DESIGN.md") != doc || read("CLAUDE.md") != claude {
		t.Errorf("a second run changed what the first wrote:\n%s", out)
	}
	if again, _ := filepath.Glob(filepath.Join(root, "design", "changes", "*.md")); len(again) != 1 {
		t.Errorf("a second run wrote another note: %v", again)
	}
	if strings.Count(out, "kept") < 4 {
		t.Errorf("a second run did not say what it kept:\n%s", out)
	}
}
