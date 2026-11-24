package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"harness/internal/convention"
	"harness/internal/design"
)

// Survey is what a machine can say about a repository without a model: the
// facts a design is drafted from. It reads and decides nothing.
type Survey struct {
	Name  string
	Date  string
	Files int
	// Globs are a spine's files: that account for every file as it stands:
	// each top-level directory whole, each root file by name.
	Globs     []string
	Languages []Count
	TestFiles []string
	// Tests are the test names found in the test files, where the language
	// declares them by name.
	Tests []string
	// Gates are commands the repository appears to prove itself with, the
	// likeliest first, each with the file that suggests it.
	Gates []Gate
	// Docs are files that may already say what was decided, and why.
	Docs []string
	CI   []string
	// Subjects are the most recent commit subjects; Conventional is how many
	// of them are Conventional Commits.
	Subjects     []string
	Conventional int
	Nested       []string
	// Harness is how many of the files are the method's own harness, copied
	// in: accounted for, and not surveyed as the project's code.
	Harness int
}

// Count is a language and how many files are written in it.
type Count struct {
	Name  string
	Files int
}

// Gate is a command and what suggests it.
type Gate struct {
	Command string
	From    string
}

var languages = map[string]string{
	".go": "Go", ".py": "Python", ".ts": "TypeScript", ".tsx": "TypeScript", ".js": "JavaScript",
	".jsx": "JavaScript", ".mjs": "JavaScript", ".rs": "Rust", ".java": "Java", ".kt": "Kotlin",
	".rb": "Ruby", ".php": "PHP", ".cs": "C#", ".c": "C", ".h": "C", ".cc": "C++", ".cpp": "C++",
	".hpp": "C++", ".swift": "Swift", ".scala": "Scala", ".ex": "Elixir", ".exs": "Elixir",
	".sh": "Shell", ".sql": "SQL", ".proto": "Protocol Buffers", ".tf": "Terraform",
	".html": "HTML", ".css": "CSS", ".scss": "CSS", ".vue": "Vue", ".svelte": "Svelte",
	".yaml": "YAML", ".yml": "YAML", ".md": "Markdown", ".dart": "Dart", ".lua": "Lua",
	".zig": "Zig", ".hs": "Haskell", ".ml": "OCaml", ".clj": "Clojure", ".r": "R",
}

var (
	goTest = regexp.MustCompile(`(?m)^func (Test\w+)\(`)
	pyTest = regexp.MustCompile(`(?m)^\s*(?:async\s+)?def (test_\w+)\(`)
	rsTest = regexp.MustCompile(`(?m)#\[(?:tokio::)?test\]\s*(?:async\s+)?fn (\w+)\(`)
	jsTest = regexp.MustCompile("(?m)^\\s*(?:it|test)\\(\\s*['\"`]([^'\"`\\n]{3,120})['\"`]")
)

// docNames are files that say what a repository is and how to work in it.
var docNames = map[string]bool{
	"readme.md": true, "architecture.md": true, "contributing.md": true, "design.md": true,
	"claude.md": true, "agents.md": true, "decisions.md": true, "adr.md": true,
	".cursorrules": true, "copilot-instructions.md": true, "security.md": true,
}

// docDirs are directories whose markdown is likely to record decisions.
var docDirs = []string{"docs/", "doc/", "adr/", "adrs/", "decisions/", "rfcs/", "rfc/", "architecture/", ".cursor/rules/"}

// maxListed bounds each list a survey prints, so that a large repository's
// survey is still something a session can be handed.
const maxListed = 200

// survey reads the repository as walked. subjects are recent commit subjects,
// newest first, and today the date a new spine's rules begin.
func survey(repo *design.Repo, subjects []string, today string) Survey {
	s := Survey{Name: repo.Name, Date: today, Subjects: subjects, Nested: repo.NestedRepos}
	top := map[string]bool{}
	langs := map[string]int{}
	tests := map[string]bool{}
	_, err := os.Stat(filepath.Join(repo.Root, "harness", "FRAMEWORK.md"))
	copied := err == nil
	for _, f := range repo.Files {
		s.Files++
		if i := strings.Index(f, "/"); i >= 0 {
			top[f[:i]+"/**"] = true
		} else if !design.Exempt[f] {
			top[f] = true
		}
		if copied && strings.HasPrefix(f, "harness/") {
			s.Harness++
			continue
		}
		if l, ok := languages[strings.ToLower(path.Ext(f))]; ok {
			langs[l]++
		}
		lower := strings.ToLower(f)
		if isDoc(lower) {
			s.Docs = append(s.Docs, f)
		}
		if strings.HasPrefix(f, ".github/workflows/") || f == ".gitlab-ci.yml" || strings.HasPrefix(f, ".circleci/") || f == "Jenkinsfile" || f == ".buildkite/pipeline.yml" {
			s.CI = append(s.CI, f)
		}
		if design.IsTestFile(f) {
			s.TestFiles = append(s.TestFiles, f)
			for _, name := range testNames(repo.Root, f) {
				tests[name] = true
			}
		}
	}
	// A design lives in design/, which the walk leaves out; whatever is
	// already there is the first thing a draft should read.
	if entries, err := os.ReadDir(filepath.Join(repo.Root, "design")); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				s.Docs = append(s.Docs, "design/"+e.Name())
			}
		}
	}
	for g := range top {
		s.Globs = append(s.Globs, g)
	}
	sort.Strings(s.Globs)
	for l, n := range langs {
		s.Languages = append(s.Languages, Count{l, n})
	}
	sort.Slice(s.Languages, func(i, j int) bool {
		if s.Languages[i].Files != s.Languages[j].Files {
			return s.Languages[i].Files > s.Languages[j].Files
		}
		return s.Languages[i].Name < s.Languages[j].Name
	})
	for t := range tests {
		s.Tests = append(s.Tests, t)
	}
	sort.Strings(s.Tests)
	sort.Strings(s.Docs)
	s.Gates = gates(repo)
	for _, subj := range subjects {
		if convention.Is(subj) {
			s.Conventional++
		}
	}
	return s
}

func isDoc(lower string) bool {
	if docNames[path.Base(lower)] {
		return true
	}
	if !strings.HasSuffix(lower, ".md") && !strings.HasSuffix(lower, ".mdc") && !strings.HasSuffix(lower, ".adoc") && !strings.HasSuffix(lower, ".rst") {
		return false
	}
	for _, d := range docDirs {
		if strings.HasPrefix(lower, d) || strings.Contains(lower, "/"+d) {
			return true
		}
	}
	return false
}

func testNames(root, f string) []string {
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f)))
	if err != nil {
		return nil
	}
	var re *regexp.Regexp
	switch {
	case strings.HasSuffix(f, ".go"):
		re = goTest
	case strings.HasSuffix(f, ".py"):
		re = pyTest
	case strings.HasSuffix(f, ".rs"):
		re = rsTest
	default:
		re = jsTest
	}
	var names []string
	for _, m := range re.FindAllSubmatch(b, -1) {
		names = append(names, string(m[1]))
	}
	return names
}

// gates looks for the command the repository already proves itself with. It
// suggests; the spine's validate: is a person's to set, because a gate that
// fails today would refuse every push from the moment it is written down.
func gates(repo *design.Repo) []Gate {
	has := map[string]bool{}
	for _, f := range repo.Files {
		has[f] = true
	}
	read := func(f string) string {
		b, _ := os.ReadFile(filepath.Join(repo.Root, f))
		return string(b)
	}
	var out []Gate
	for _, mk := range []string{"Makefile", "makefile", "GNUmakefile"} {
		if has[mk] {
			for _, target := range []string{"check", "test", "ci"} {
				if regexp.MustCompile(`(?m)^` + target + `:`).MatchString(read(mk)) {
					out = append(out, Gate{"make " + target, mk})
					break
				}
			}
			break
		}
	}
	if has["Taskfile.yml"] || has["Taskfile.yaml"] {
		f := "Taskfile.yml"
		if !has[f] {
			f = "Taskfile.yaml"
		}
		if regexp.MustCompile(`(?m)^  test:`).MatchString(read(f)) {
			out = append(out, Gate{"task test", f})
		}
	}
	if has["justfile"] && regexp.MustCompile(`(?m)^test\b.*:`).MatchString(read("justfile")) {
		out = append(out, Gate{"just test", "justfile"})
	}
	if has["go.mod"] {
		out = append(out, Gate{"go vet ./... && go test ./...", "go.mod"})
	}
	if has["package.json"] {
		var pkg struct {
			Scripts map[string]string `json:"scripts"`
		}
		if json.Unmarshal([]byte(read("package.json")), &pkg) == nil {
			if t := pkg.Scripts["test"]; t != "" && !strings.Contains(t, "no test specified") {
				run := "npm test"
				switch {
				case has["pnpm-lock.yaml"]:
					run = "pnpm test"
				case has["yarn.lock"]:
					run = "yarn test"
				case has["bun.lockb"], has["bun.lock"]:
					run = "bun run test"
				}
				out = append(out, Gate{run, "package.json"})
			}
		}
	}
	if has["Cargo.toml"] {
		out = append(out, Gate{"cargo test", "Cargo.toml"})
	}
	for _, f := range []string{"pytest.ini", "tox.ini", "pyproject.toml", "setup.cfg"} {
		if has[f] && (f == "pytest.ini" || strings.Contains(read(f), "pytest")) {
			out = append(out, Gate{"pytest", f})
			break
		}
	}
	for _, f := range []string{"pom.xml", "build.gradle", "build.gradle.kts", "Gemfile", "mix.exs"} {
		if has[f] {
			cmd := map[string]string{"pom.xml": "mvn -q verify", "build.gradle": "./gradlew check", "build.gradle.kts": "./gradlew check", "Gemfile": "bundle exec rake test", "mix.exs": "mix test"}[f]
			out = append(out, Gate{cmd, f})
		}
	}
	return out
}

// subjects are the most recent commit subjects of the repository at root,
// newest first; none when it has no history.
func recentSubjects(root string, n int) []string {
	out, err := exec.Command("git", "-C", root, "log", fmt.Sprintf("-%d", n), "--no-merges", "--format=%s").Output()
	if err != nil {
		return nil
	}
	var subjects []string
	for _, l := range strings.Split(string(out), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			subjects = append(subjects, l)
		}
	}
	return subjects
}

// AsksConventional reports whether the history already follows the
// convention closely enough that the spine may ask for it: at least twenty
// subjects, four in five of them conventional.
func (s Survey) AsksConventional() bool {
	return len(s.Subjects) >= 20 && s.Conventional*5 >= len(s.Subjects)*4
}

// Markdown is the survey as a session, or a person, is handed it.
func (s Survey) Markdown() string {
	b := &strings.Builder{}
	fmt.Fprintf(b, "# Survey of %s\n\nRead by a machine on %s. Facts only: nothing here is a decision.\n\n", s.Name, s.Date)
	fmt.Fprintf(b, "## Files\n\n%d files, not counting `design/`, `.git` and what git ignores.\n\n", s.Files)
	b.WriteString("Globs that account for every one of them as they stand, the coarsest possible `files:`:\n\n")
	list(b, s.Globs, "`%s`")
	if s.Harness > 0 {
		fmt.Fprintf(b, "\n%d of the files are `harness/`, the method's own harness, copied in. It is accounted for by `harness/**` and is not surveyed below: its tests and documents are not this project's.\n", s.Harness)
	}
	if len(s.Nested) > 0 {
		b.WriteString("\nNested git repositories, which the walk does not enter and this design does not account for:\n\n")
		list(b, s.Nested, "`%s/`")
	}
	b.WriteString("\n## Languages\n\n")
	if len(s.Languages) == 0 {
		b.WriteString("None recognised.\n")
	}
	for _, l := range s.Languages {
		fmt.Fprintf(b, "- %s: %d file(s)\n", l.Name, l.Files)
	}
	fmt.Fprintf(b, "\n## Tests\n\n%d test file(s), %d test name(s) found.\n", len(s.TestFiles), len(s.Tests))
	if len(s.TestFiles) > 0 {
		b.WriteString("\nTest files:\n\n")
		list(b, s.TestFiles, "`%s`")
	}
	if len(s.Tests) > 0 {
		b.WriteString("\nTest names, the only ones the spine's `tests:` may name:\n\n")
		list(b, s.Tests, "`%s`")
	}
	b.WriteString("\n## The repository's own gate\n\n")
	if len(s.Gates) == 0 {
		b.WriteString("No command found that the repository proves itself with.\n")
	}
	for _, g := range s.Gates {
		fmt.Fprintf(b, "- `%s` (suggested by `%s`)\n", g.Command, g.From)
	}
	b.WriteString("\n## Continuous integration\n\n")
	if len(s.CI) == 0 {
		b.WriteString("None found.\n")
	}
	list(b, s.CI, "`%s`")
	b.WriteString("\n## Documents that may already record what was decided\n\n")
	if len(s.Docs) == 0 {
		b.WriteString("None found.\n")
	}
	list(b, s.Docs, "`%s`")
	fmt.Fprintf(b, "\n## Commits\n\n%d of the last %d subjects are Conventional Commits.", s.Conventional, len(s.Subjects))
	if s.AsksConventional() {
		b.WriteString(" The history already follows the convention.")
	}
	b.WriteString("\n")
	if len(s.Subjects) > 0 {
		b.WriteString("\nThe most recent:\n\n")
		recent := s.Subjects
		if len(recent) > 40 {
			recent = recent[:40]
		}
		for _, subj := range recent {
			fmt.Fprintf(b, "- %s\n", subj)
		}
	}
	return b.String()
}

func list(b *strings.Builder, items []string, format string) {
	for i, it := range items {
		if i == maxListed {
			fmt.Fprintf(b, "- … and %d more\n", len(items)-maxListed)
			return
		}
		fmt.Fprintf(b, "- "+format+"\n", it)
	}
}

// Skeleton fills the design template with what the survey found: a spine that
// accounts for every file as it stands, and a body whose every section is
// still a question. The template's placeholders are {{NAME}}, {{DATE}},
// {{FILES}}, {{VALIDATE}} and {{COMMITS}}.
func (s Survey) Skeleton(template string) string {
	files := &strings.Builder{}
	for _, g := range s.Globs {
		fmt.Fprintf(files, "  - %s\n", yamlScalar(g))
	}
	if len(s.Globs) == 0 {
		files.WriteString("  []\n")
	}
	validate := "# validate: <the command this repository proves itself with; none was found>"
	if len(s.Gates) > 0 {
		validate = "# validate: " + s.Gates[0].Command
	}
	commits := "# commits: conventional"
	if s.AsksConventional() {
		commits = "commits: conventional"
	}
	return strings.NewReplacer(
		"{{NAME}}", s.Name,
		"{{DATE}}", s.Date,
		"{{FILES}}", strings.TrimRight(files.String(), "\n"),
		"{{VALIDATE}}", validate,
		"{{COMMITS}}", commits,
	).Replace(template)
}

// yamlScalar quotes a glob that YAML would read as something else: one that
// opens with a character YAML gives meaning to, a * or a . among them.
func yamlScalar(s string) string {
	if strings.ContainsAny(s[:1], "*&!|>'\"%@`{}[],#?-") || strings.Contains(s, ": ") || strings.Contains(s, " #") {
		return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
	}
	return s
}
