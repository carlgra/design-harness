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

// theScript is the real bootstrap, beside harness/ in the method's own
// repository; a copy of the harness in a project has none, and there a test
// that runs it has nothing to walk.
func theScript(t *testing.T) string {
	t.Helper()
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
	return script
}

// aProject is a git repository holding the files given, with nothing
// committed.
func aProject(t *testing.T, files map[string]string) string {
	t.Helper()
	root, _ := filepath.EvalSymlinks(t.TempDir())
	for name, body := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	gitIn(t, root, "init", "-q", "-b", "main")
	return root
}

func readIn(t *testing.T, root, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Where CLAUDE.md only points at AGENTS.md, by an import or as a link, the
// method's rules are added to AGENTS.md, where every agent reads them, and
// bootstrap says so; CLAUDE.md is left exactly as it was. The draft writes
// the project's section into the same file, and an update replaces the rules
// there.
func TestTheMethodsRulesGoWhereTheRepositorysInstructionsLive(t *testing.T) {
	script := theScript(t)
	const (
		pointer  = "# Agents\n\n@AGENTS.md\n"
		standing = "# Standing instructions\n\nAlways run make test.\n"
	)
	cases := []struct {
		name string
		link bool
		says string
	}{
		{"an import", false, "at the end of AGENTS.md (CLAUDE.md only imports it)"},
		{"a link", true, "at the end of AGENTS.md (CLAUDE.md is a link to it)"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := aProject(t, map[string]string{"AGENTS.md": standing, "CLAUDE.md": pointer})
			claude := filepath.Join(root, "CLAUDE.md")
			if c.link {
				if err := os.Remove(claude); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("AGENTS.md", claude); err != nil {
					t.Skip("this file system has no links")
				}
			}
			untouched := func(when string) {
				t.Helper()
				fi, err := os.Lstat(claude)
				if err != nil {
					t.Fatal(err)
				}
				if c.link && fi.Mode()&os.ModeSymlink == 0 {
					t.Fatalf("%s, CLAUDE.md is no longer a link", when)
				}
				if !c.link && readIn(t, root, "CLAUDE.md") != pointer {
					t.Fatalf("%s, CLAUDE.md was written to:\n%s", when, readIn(t, root, "CLAUDE.md"))
				}
			}

			out := run(t, "sh", script, "--no-ci", "--no-hooks", root)
			if !strings.Contains(out, c.says) {
				t.Errorf("bootstrap did not say which file it chose, %q:\n%s", c.says, out)
			}
			untouched("after adopting")
			agents := readIn(t, root, "AGENTS.md")
			if !strings.HasPrefix(agents, standing) {
				t.Errorf("the project's own AGENTS.md was not kept as it was:\n%s", agents)
			}
			for _, want := range []string{MethodBegin, "Design before code", ProjectBegin} {
				if strings.Count(agents, want) != 1 {
					t.Errorf("AGENTS.md holds %q %d times, want once", want, strings.Count(agents, want))
				}
			}

			// The draft's section for the project goes in the same file.
			file, err := writeProject(root, "- Build with make.")
			if err != nil || file != "AGENTS.md" {
				t.Fatalf("the project's section went to %q (%v)", file, err)
			}
			untouched("after a draft")
			agents = readIn(t, root, "AGENTS.md")
			if !strings.Contains(agents, "- Build with make.") || strings.Count(agents, ProjectBegin) != 1 {
				t.Errorf("the project's section is not in AGENTS.md once:\n%s", agents)
			}

			// A year passes; an update finds the rules where they are.
			stale := strings.Replace(agents, "1. **Design before code.**", "1. **A RULE FROM AN OLD VERSION.**", 1)
			if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(stale), 0o644); err != nil {
				t.Fatal(err)
			}
			out = run(t, "sh", script, "--update", "--no-ci", "--no-hooks", root)
			if !strings.Contains(out, "replaced  the method's rules in AGENTS.md") {
				t.Errorf("an update did not say where it replaced the rules:\n%s", out)
			}
			untouched("after an update")
			if got := readIn(t, root, "AGENTS.md"); got != agents {
				t.Errorf("an update did not bring AGENTS.md back to the current rules and nothing else:\n%s", got)
			}
		})
	}
}

// The workflow bootstrap writes runs where the repository's own workflows
// run: on the one runner they all name, and where they name two, on the
// template's, with a line that says the runner is one to check and what was
// found. A repository with no workflow is told nothing.
func TestTheWorkflowRunsWhereTheRepositorysWorkflowsRun(t *testing.T) {
	script := theScript(t)
	job := func(runsOn string) string {
		return "name: ci\non: [push]\njobs:\n  build:\n    runs-on: " + runsOn + "\n    steps:\n      - run: make\n"
	}
	const written = ".github/workflows/harness.yml"

	root := aProject(t, map[string]string{".github/workflows/ci.yml": job("self-hosted"), ".github/workflows/release.yml": job("self-hosted")})
	out := run(t, "sh", script, "--no-hooks", root)
	if ci := readIn(t, root, written); !strings.Contains(ci, "    runs-on: self-hosted\n") || strings.Contains(ci, "ubuntu-latest") {
		t.Errorf("the workflow does not run where the repository's do:\n%s", ci)
	}
	if !strings.Contains(out, "runs-on: self-hosted in that workflow") || !strings.Contains(out, "harness: OK") {
		t.Errorf("bootstrap did not say which runner it wrote:\n%s", out)
	}

	root = aProject(t, map[string]string{".github/workflows/ci.yml": job("self-hosted"), ".github/workflows/mac.yml": job("macos-14")})
	out = run(t, "sh", script, "--no-hooks", root)
	if ci := readIn(t, root, written); !strings.Contains(ci, "    runs-on: ubuntu-latest\n") {
		t.Errorf("with two runners to choose from the workflow was not left on the template's:\n%s", ci)
	}
	if !strings.Contains(out, "is a line to check: this repository's workflows run on macos-14 and self-hosted") {
		t.Errorf("bootstrap did not say the runner is a line to check:\n%s", out)
	}

	root = aProject(t, map[string]string{"README.md": "# shop\n"})
	out = run(t, "sh", script, "--no-hooks", root)
	if ci := readIn(t, root, written); !strings.Contains(ci, "    runs-on: ubuntu-latest\n") || strings.Contains(out, "runs-on") {
		t.Errorf("a repository with no workflow was told about a runner:\n%s", out)
	}
}

// An update replaces an earlier form of this harness, one with the checker
// and the push script and no FRAMEWORK.md, carrying its settings across from
// their old names; says what an update leaves to do, and not what a first
// adoption does; and names each test the design cites that went with the old
// harness. Without --update the earlier harness stops bootstrap, which says
// to use it. A harness/ that is not a harness is never replaced.
func TestAnUpdateReplacesAnEarlierHarnessAndSaysWhatAnUpdateLeavesToDo(t *testing.T) {
	script := theScript(t)
	refused := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("sh", append([]string{script}, args...)...)
		for _, kv := range os.Environ() {
			if !strings.HasPrefix(kv, "GIT_") {
				cmd.Env = append(cmd.Env, kv)
			}
		}
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatalf("bootstrap %v was not refused:\n%s", args, out)
		}
		return string(out)
	}

	root := aProject(t, map[string]string{"README.md": "# shop\n"})
	run(t, "sh", script, "--no-ci", "--no-hooks", root)
	write := func(name, body string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// The name of a test only the earlier harness held, put together here so
	// that this file, which the update copies in, does not hold it.
	const lost = "TestOnlyThe" + "EarlierHarnessHeld"
	// The harness as it once was: no FRAMEWORK.md, its cold-read settings
	// under the old name, and a test this harness does not hold, which the
	// project's design names.
	for _, gone := range []string{"harness/FRAMEWORK.md", "harness/coldread"} {
		if err := os.RemoveAll(filepath.Join(root, filepath.FromSlash(gone))); err != nil {
			t.Fatal(err)
		}
	}
	write("harness/goldfish/goldfish.yaml", "model: the-projects-own\nverdict: \"VERDICT:\"\n")
	write("harness/cmd/check/earlier_test.go", "package main\n\nimport \"testing\"\n\nfunc "+lost+"(t *testing.T) {}\n")
	doc := readIn(t, root, "design/DESIGN.md")
	named := strings.Replace(strings.Replace(doc, "tests: []", "tests:\n  - "+lost, 1), SkeletonMark+"\n", "", 1)
	if named == doc {
		t.Fatalf("the fixture did not name the test:\n%s", doc)
	}
	write("design/DESIGN.md", named)

	if out := refused(root); !strings.Contains(out, "an earlier form of this harness") || !strings.Contains(out, "--update") {
		t.Errorf("an earlier harness met without --update was not sent to it:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(root, "harness", "FRAMEWORK.md")); err == nil {
		t.Fatal("the earlier harness was replaced without --update")
	}

	out := run(t, "sh", script, "--update", "--no-ci", "--no-hooks", root)
	if _, err := os.Stat(filepath.Join(root, "harness", "FRAMEWORK.md")); err != nil {
		t.Fatalf("an update did not replace the earlier harness:\n%s", out)
	}
	if !strings.Contains(readIn(t, root, "harness/coldread/coldread.yaml"), "the-projects-own") {
		t.Error("the earlier harness's settings were not carried across")
	}
	if readIn(t, root, "design/DESIGN.md") != named {
		t.Error("an update changed the project's design")
	}
	for _, not := range []string{"Draft the design from the code", "feat/adopt-the-design", "cmd/reverse draft"} {
		if strings.Contains(out, not) {
			t.Errorf("an update printed the advice for a first adoption, %q:\n%s", not, out)
		}
	}
	for _, want := range []string{
		"An update leaves three things to do",
		"git diff",
		"run ./cmd/check",
		"design/DESIGN.md names the test " + lost, // the checker's own refusal
		"this harness does not hold them:\n\n         " + lost + "\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("an update did not say %q:\n%s", want, out)
		}
	}

	// A directory of the project's own that happens to be called harness.
	other := aProject(t, map[string]string{"harness/notes.txt": "the project's own\n"})
	for _, args := range [][]string{{other}, {"--update", other}} {
		if out := refused(args...); !strings.Contains(out, "is not this harness") {
			t.Errorf("bootstrap %v did not say what it found:\n%s", args, out)
		}
	}
	if readIn(t, other, "harness/notes.txt") != "the project's own\n" {
		t.Error("a directory that is not a harness was written to")
	}
}
