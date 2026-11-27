package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func run(t *testing.T, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "GIT_") {
			cmd.Env = append(cmd.Env, kv)
		}
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
	return string(out)
}

// A repository under a root is given no harness of its own: it uses the
// root's by relative path, its rules say to read the root's, its frontmatter takes
// the root's commit convention, its CI checks the root out around it, and it
// passes the checker run from the root's harness. The root, with the
// repository now inside it, still passes its own.
func TestARepositoryUnderARootUsesTheRootsHarnessAndRules(t *testing.T) {
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
	tmp, _ := filepath.EvalSymlinks(t.TempDir())
	root := filepath.Join(tmp, "anything")
	child := filepath.Join(root, "services", "billing")
	write := func(p, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// The root: a repository that holds what binds all of them.
	write(filepath.Join(root, "README.md"), "# project\n")
	write(filepath.Join(root, ".gitignore"), "/services/billing/\n")
	gitIn(t, root, "init", "-q", "-b", "main")
	gitIn(t, root, "remote", "add", "origin", "git@github.com:acme/project.git")
	run(t, "sh", script, root)
	rootDoc := filepath.Join(root, "design", "DESIGN.md")
	b, _ := os.ReadFile(rootDoc)
	withConvention := strings.Replace(string(b), "# commits: conventional", "commits: conventional", 1)
	// The root's map: a contracts repository, and the service about to be adopted.
	theMap := "repos:\n  - path: contracts\n    kind: contracts\n    remote: git@github.com:acme/contracts.git\n  - path: services/billing\n    kind: service\n"
	write(rootDoc, strings.Replace(withConvention, "\n---\n\n# ", "\n"+theMap+"---\n\n# ", 1))
	write(filepath.Join(root, "contracts", "README.md"), "# contracts\n")
	gitIn(t, filepath.Join(root, "contracts"), "init", "-q", "-b", "main")
	gitIn(t, filepath.Join(root, "contracts"), "add", "-A")
	gitIn(t, filepath.Join(root, "contracts"), "commit", "-q", "-m", "feat: contracts")

	// A service, checked out inside it.
	write(filepath.Join(child, "README.md"), "# billing\n")
	write(filepath.Join(child, "go.mod"), "module billing\n\ngo 1.22\n")
	write(filepath.Join(child, "main.go"), "package main\n\nfunc main() {}\n")
	gitIn(t, child, "init", "-q", "-b", "main")
	gitIn(t, child, "add", "-A")
	gitIn(t, child, "commit", "-q", "-m", "feat: billing")
	out := run(t, "sh", script, child)

	if _, err := os.Stat(filepath.Join(child, "harness")); err == nil {
		t.Fatalf("a repository under a root was given a harness of its own:\n%s", out)
	}
	for _, want := range []string{"under project: its harness, ../../harness", "harness: OK", "Put it on the map", "- path: services/billing"} {
		if !strings.Contains(out, want) {
			t.Errorf("bootstrap did not say %q:\n%s", want, out)
		}
	}
	read := func(name string) string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(child, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	claude := read("CLAUDE.md")
	for _, want := range []string{"nested under `project`, at `services/billing/`", "`../../CLAUDE.md`", "../../harness/safe-push", "`project/ADR-0004`", "go -C ../../harness run ./cmd/check"} {
		if !strings.Contains(claude, want) {
			t.Errorf("the rules do not say %q:\n%s", want, claude)
		}
	}
	doc := read("design/DESIGN.md")
	for _, want := range []string{"\ncommits: conventional\n", "nested under `project`, at `services/billing/`", "  - main.go\n"} {
		if !strings.Contains(doc, want) {
			t.Errorf("the skeleton does not hold %q:\n%s", want, doc)
		}
	}
	if strings.Contains(doc, "harness/**") {
		t.Errorf("the skeleton accounts for a harness the repository does not have:\n%s", doc)
	}
	ci := read(".github/workflows/harness.yml")
	for _, want := range []string{"repository: acme/project", "path: services/billing", "go -C ../../harness run ./cmd/check", "repository: acme/contracts", "path: contracts\n", "fetch-depth: 0"} {
		if !strings.Contains(ci, want) {
			t.Errorf("the CI workflow does not say %q:\n%s", want, ci)
		}
	}
	for _, f := range []string{"CLAUDE.md", "design/DESIGN.md", ".github/workflows/harness.yml"} {
		if regexp.MustCompile(`\{\{[A-Z_]+\}\}`).MatchString(read(f)) {
			t.Errorf("%s has a placeholder left in it", f)
		}
	}
	if got := gitIn(t, child, "config", "--get", "core.hooksPath"); got != "../../harness/hooks" {
		t.Errorf("core.hooksPath = %q", got)
	}

	// The root is none the worse for it, and is held to its map once it has one.
	if out := run(t, "go", "-C", filepath.Join(root, "harness"), "run", "./cmd/check", "-root", root, "-validate=false"); !strings.Contains(out, "harness: OK") {
		t.Errorf("the root does not pass with a repository nested in it:\n%s", out)
	}
}

// An update replaces what is the harness's and nothing that is the
// project's: harness/ and the method's rules between their marks are this
// harness's again, and the project's settings, its own lines of CLAUDE.md
// and its own section there are as they were.
func TestAnUpdateReplacesTheHarnessAndTheRulesAndNothingOfTheProjects(t *testing.T) {
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
	root, _ := filepath.EvalSymlinks(t.TempDir())
	path := func(name string) string { return filepath.Join(root, filepath.FromSlash(name)) }
	read := func(name string) string {
		t.Helper()
		b, err := os.ReadFile(path(name))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(path(name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("CLAUDE.md", "# Mine\n\nAlways run make test.\n")
	gitIn(t, root, "init", "-q", "-b", "main")
	run(t, "sh", script, root)
	fresh := read("CLAUDE.md")

	// A year passes: the rules here are an old version's, the project has
	// written its own section and set its own model, and someone has
	// changed the harness in place.
	stale := strings.Replace(fresh, "1. **Design before code.**", "1. **A RULE FROM AN OLD VERSION.**", 1)
	stale = strings.Replace(stale, "TODO(design): how to build", "Build with make. TODO(design): how to build", 1)
	if stale == fresh || !strings.Contains(stale, "Build with make.") {
		t.Fatalf("the fixture did not change the file:\n%s", fresh)
	}
	write("CLAUDE.md", stale+"\n## After\n\nThe project's last word.\n")
	write("harness/coldread/coldread.yaml", "model: the-projects-own\nverdict: \"VERDICT:\"\n")
	write("harness/FRAMEWORK.md", "# changed in place\n")

	out := run(t, "sh", script, "--update", root)
	got := read("CLAUDE.md")
	if strings.Contains(got, "A RULE FROM AN OLD VERSION") || !strings.Contains(got, "1. **Design before code.**") {
		t.Errorf("the method's rules were not replaced:\n%s", got)
	}
	for _, kept := range []string{"# Mine\n\nAlways run make test.\n", "Build with make.", "## After\n\nThe project's last word.\n"} {
		if !strings.Contains(got, kept) {
			t.Errorf("an update lost what was the project's, %q:\n%s", kept, got)
		}
	}
	if strings.Count(got, "design-harness:begin") != 1 || strings.Count(got, "design-harness:end") != 1 {
		t.Errorf("the marks are not as they were:\n%s", got)
	}
	if !strings.Contains(read("harness/coldread/coldread.yaml"), "the-projects-own") {
		t.Error("an update replaced the project's settings")
	}
	if strings.Contains(read("harness/FRAMEWORK.md"), "changed in place") {
		t.Error("an update did not replace the harness")
	}
	if !strings.Contains(out, "replaced  the method's rules in CLAUDE.md") {
		t.Errorf("an update did not say what it replaced:\n%s", out)
	}

	// With nothing stale, nothing is rewritten.
	out = run(t, "sh", script, "--update", root)
	if read("CLAUDE.md") != got || !strings.Contains(out, "the method's rules are current") {
		t.Errorf("a second update changed the rules again:\n%s", out)
	}
}
