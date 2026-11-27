package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"harness/internal/design"
)

func repoWith(t *testing.T, files map[string]string) *design.Repo {
	t.Helper()
	root := t.TempDir()
	for name, body := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	repo, err := design.Walk(root)
	if err != nil {
		t.Fatal(err)
	}
	return repo
}

var existing = map[string]string{
	"README.md":                  "# shop\n",
	"go.mod":                     "module shop\n",
	"Makefile":                   "build:\n\tgo build ./...\ntest:\n\tgo test ./...\n",
	"*.weird":                    "a name YAML would read as an alias\n",
	"cmd/shop/main.go":           "package main\n",
	"internal/cart/cart.go":      "package cart\n",
	"internal/cart/cart_test.go": "package cart\nfunc TestAnEmptyCartIsRefused(t *testing.T) {}\nfunc helper() {}\n",
	"web/cart.test.ts":           "it('totals a cart', () => {})\n",
	"web/app.tsx":                "export {}\n",
	"docs/adr/0001-use-go.md":    "# Use Go\n",
	"docs/guide.md":              "# Guide\n",
	".github/workflows/ci.yml":   "on: push\n",
	"design/old-notes.md":        "# Notes from before\n",
}

const template = "---\nfiles:\n{{FILES}}\ntests: []\n{{VALIDATE}}\n{{COMMITS}}\nnotes:\n  from: {{DATE}}\n  sections: {{SECTIONS}}\n{{REPOS}}\n---\n\n# {{NAME}}\n\n" + SkeletonMark + "\n\n## Scope\n\n{{PARENT}}\n\nTODO(design)\n"

// The survey reports what is there and decides nothing: every file falls
// under a glob it proposes, the tests it lists are ones that exist, the gate
// is one the repository already has, and the documents that may record
// decisions are found, design/ included.
func TestTheSurveySaysWhatIsThereAndDecidesNothing(t *testing.T) {
	repo := repoWith(t, existing)
	s := survey(repo, nil, nil, "2026-01-02")
	for _, f := range repo.Files {
		if !design.Exempt[f] && !design.Claimed(s.Globs, f) {
			t.Errorf("%s is under no glob the survey proposes: %v", f, s.Globs)
		}
	}
	if got := strings.Join(s.Tests, "|"); got != "TestAnEmptyCartIsRefused|totals a cart" {
		t.Errorf("tests found: %q", got)
	}
	if len(s.Gates) < 2 || s.Gates[0].Command != "make test" || s.Gates[1].From != "go.mod" {
		t.Errorf("gates found: %+v", s.Gates)
	}
	docs := strings.Join(s.Docs, "|")
	for _, want := range []string{"README.md", "docs/adr/0001-use-go.md", "docs/guide.md", "design/old-notes.md"} {
		if !strings.Contains(docs, want) {
			t.Errorf("the survey did not find %s among the documents: %s", want, docs)
		}
	}
	if len(s.CI) != 1 || s.Languages[0].Name != "Go" {
		t.Errorf("ci %v, languages %v", s.CI, s.Languages)
	}
	// A copy of the harness is accounted for and not taken for the project.
	withHarness := map[string]string{"harness/FRAMEWORK.md": "# The method\n", "harness/cmd/check/checks_test.go": "package main\nfunc TestOfTheHarness(t *testing.T) {}\n"}
	for k, v := range existing {
		withHarness[k] = v
	}
	h := survey(repoWith(t, withHarness), nil, nil, "2026-01-02")
	if h.Harness != 2 || !design.Claimed(h.Globs, "harness/FRAMEWORK.md") || strings.Contains(strings.Join(h.Tests, "|"), "TestOfTheHarness") || strings.Contains(strings.Join(h.Docs, "|"), "harness/") {
		t.Errorf("the harness was surveyed as the project's own: %d files, tests %v, docs %v", h.Harness, h.Tests, h.Docs)
	}
	md := s.Markdown()
	for _, want := range []string{"`internal/**`", "`TestAnEmptyCartIsRefused`", "`make test` (suggested by `Makefile`)", "nothing here is a decision"} {
		if !strings.Contains(md, want) {
			t.Errorf("the survey as handed over does not say %q", want)
		}
	}
}

// A skeleton's frontmatter accounts for every file as it stands, so that the
// checker passes the moment a repository adopts the method; and it switches
// nothing on: the gate is a comment, and so is the convention unless the
// history already follows it.
func TestASkeletonsFrontmatterAccountsForEveryFileAndSwitchesNothingOn(t *testing.T) {
	repo := repoWith(t, existing)
	read := func(subjects []string) design.Design {
		t.Helper()
		doc := survey(repo, nil, subjects, "2026-01-02").Skeleton(template)
		fm, err := design.ParseFrontmatter(doc)
		if err != nil {
			t.Fatalf("%v\n%s", err, doc)
		}
		var d design.Design
		if err := yaml.Unmarshal([]byte(fm.YAML), &d); err != nil {
			t.Fatalf("the frontmatter does not parse: %v\n%s", err, fm.YAML)
		}
		if !strings.Contains(fm.Body, "# "+repo.Name) || !strings.Contains(fm.Body, SkeletonMark) {
			t.Fatalf("the body is not the skeleton's:\n%s", fm.Body)
		}
		return d
	}
	d := read([]string{"wip", "more"})
	for _, f := range repo.Files {
		if !design.Exempt[f] && !design.Claimed(d.Files, f) {
			t.Errorf("%s is not accounted for by the skeleton's frontmatter: %v", f, d.Files)
		}
	}
	if d.Validate != "" || d.Commits != "" || len(d.Tests) != 0 {
		t.Errorf("a skeleton switched something on: %+v", d)
	}
	if d.Notes.From != "2026-01-02" {
		t.Errorf("the rules for notes begin %q", d.Notes.From)
	}

	var history []string
	for i := 0; i < 24; i++ {
		history = append(history, "feat(cart): a thing")
	}
	if d := read(append(history, "wip")); d.Commits != "conventional" {
		t.Errorf("a history that follows the convention was not asked to keep it: %q", d.Commits)
	}
	if d := read(append(history[:10], "wip", "wip", "wip", "wip", "wip", "wip", "wip", "wip", "wip", "wip")); d.Commits != "" {
		t.Errorf("a history that half follows the convention was bound to it")
	}
}

const reply = "Here is the design.\n\n" + DesignHeader + "\n---\nfiles:\n  - cmd/**\ntests: [TestAnEmptyCartIsRefused]\n---\n\n# shop\n\n## Scope\n\nA shop. It sells things (inferred).\n\n" + RulesHeader + "\n### Commands\n\n- `make test`\n"

// A drafting session's reply is read as two parts and nothing else: the
// design, which must be a document the checker can load, and the project's
// rules. A reply that is not in the form is refused, not half written.
func TestADraftIsReadAsTheDesignAndTheRules(t *testing.T) {
	doc, rules, err := split(reply)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(doc, "---\nfiles:\n") || !strings.HasSuffix(doc, "It sells things (inferred).\n") {
		t.Errorf("the design was read as:\n%q", doc)
	}
	if rules != "### Commands\n\n- `make test`" {
		t.Errorf("the rules were read as %q", rules)
	}

	fenced := DesignHeader + "\n````markdown\n---\nfiles: [a/**]\n---\n\n# x\n\n```\ncode\n```\n````\n" + RulesHeader + "\n```\nrules\n```\n"
	doc, rules, err = split(fenced)
	if err != nil || !strings.HasPrefix(doc, "---\nfiles: [a/**]") || !strings.Contains(doc, "```\ncode\n```") || strings.Contains(doc, "````") || rules != "rules" {
		t.Errorf("a part wrapped in a fence was read as %q and %q (%v)", doc, rules, err)
	}

	for name, bad := range map[string]string{
		"no header":      "---\nfiles: [a/**]\n---\n\n# x\n",
		"no frontmatter": DesignHeader + "\n# x\n\nNo frontmatter.\n",
		"no files":       DesignHeader + "\n---\ntests: []\n---\n\n# x\n",
		"no body":        DesignHeader + "\n---\nfiles: [a/**]\n---\n",
		"not yaml":       DesignHeader + "\n---\nfiles: [a/**\n---\n\n# x\n",
	} {
		if _, _, err := split(bad); err == nil {
			t.Errorf("%s: a reply that is not a design was accepted", name)
		}
	}
}

// A draft replaces a skeleton and never a design a person has taken on,
// unless it is forced to.
func TestADraftNeverReplacesADesignSomeoneWrote(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "DESIGN.md")
	if err := replaceable(p, false); err != nil {
		t.Errorf("no design yet, and a draft was refused: %v", err)
	}
	os.WriteFile(p, []byte("---\nfiles: []\n---\n\n# x\n\n"+SkeletonMark+"\n"), 0o644)
	if err := replaceable(p, false); err != nil {
		t.Errorf("a skeleton was not replaceable: %v", err)
	}
	os.WriteFile(p, []byte("---\nfiles: []\n---\n\n# x\n\nWritten by someone.\n"), 0o644)
	if err := replaceable(p, false); err == nil || !strings.Contains(err.Error(), "-force") {
		t.Errorf("a design someone wrote was replaceable: %v", err)
	}
	if err := replaceable(p, true); err != nil {
		t.Errorf("-force was refused: %v", err)
	}
}

// The project's rules go between their two marks in CLAUDE.md, and nothing
// outside the marks is touched; a file that has no marks keeps all it had.
func TestTheProjectsRulesReplaceOnlyTheirOwnSection(t *testing.T) {
	old := "# Working here\n\nThe method.\n\n## This project\n\n" + ProjectBegin + "\nTODO\n" + ProjectEnd + "\n\n## After\n\nKept.\n"
	got := project(old, "- `make test`\n")
	want := "# Working here\n\nThe method.\n\n## This project\n\n" + ProjectBegin + "\n- `make test`\n" + ProjectEnd + "\n\n## After\n\nKept.\n"
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
	if again := project(got, "- `make test`"); again != want {
		t.Errorf("writing the same rules twice changed the file:\n%q", again)
	}
	got = project("# Mine\n\nMy own rules.\n", "- x")
	if !strings.HasPrefix(got, "# Mine\n\nMy own rules.\n\n## This project\n\n"+ProjectBegin+"\n- x\n"+ProjectEnd) {
		t.Errorf("a file without marks lost something:\n%q", got)
	}
}

// What a drafting session is handed: the instructions, the survey and the
// shape to fill, without the mark that would make its own reply a skeleton.
func TestADraftingSessionIsHandedTheInstructionsTheSurveyAndTheShape(t *testing.T) {
	harness := t.TempDir()
	os.MkdirAll(filepath.Join(harness, "reverse"), 0o755)
	os.WriteFile(filepath.Join(harness, "reverse", "reverse.md"), []byte("INSTRUCTIONS\n"), 0o644)
	repo := repoWith(t, existing)
	s := survey(repo, nil, nil, "2026-01-02")
	p, err := prompt(harness, repo.Root, s, template)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(p, "INSTRUCTIONS\n\n---\n\n# Survey of ") {
		t.Errorf("the prompt opens:\n%q", p[:60])
	}
	for _, want := range []string{"`cmd/**`", "The shape to fill", "  - cmd/**\n", "from: 2026-01-02"} {
		if !strings.Contains(p, want) {
			t.Errorf("the prompt does not hold %q", want)
		}
	}
	if strings.Contains(p, SkeletonMark) {
		t.Error("the shape handed over still carries the skeleton's mark")
	}
}

// A repository nested under a root takes the root's rules into its own
// frontmatter, where they can be read: the commit convention and the sections the
// root asks of a note. Its scope says where it sits and how the root is
// cited, the survey names the siblings the root lists, and a drafting session
// is handed the root's design. A repository that stands alone gets none of it.
func TestANestedRepositoryTakesItsRulesFromTheRoot(t *testing.T) {
	repo := repoWith(t, existing)
	parent := &design.Parent{
		Name: "project", Path: "services/billing", Up: "../..",
		Doc: "---\nfiles: [harness/**]\n---\n\n# project\n\n### ADR-0004 — interfaces live in contracts\n",
		Design: design.Design{
			Commits: "conventional",
			Notes:   design.Notes{Sections: []string{"Problem", "Plan", "Alternatives rejected", "Files", "What existing data changes meaning"}},
			Repos: []design.RepoEntry{
				{Path: "services/billing", Kind: "service"},
				{Path: "contracts", Kind: "contracts"},
				{Path: "services/ledger", Kind: "service", Status: "planned"},
			},
		},
	}
	s := survey(repo, parent, nil, "2026-01-02")
	doc := s.Skeleton(template)
	fm, err := design.ParseFrontmatter(doc)
	if err != nil {
		t.Fatal(err)
	}
	var d design.Design
	if err := yaml.Unmarshal([]byte(fm.YAML), &d); err != nil {
		t.Fatalf("the frontmatter does not parse: %v\n%s", err, fm.YAML)
	}
	if d.Commits != "conventional" || len(d.Notes.Sections) != 5 || d.Notes.Sections[4] != "What existing data changes meaning" {
		t.Errorf("the root's rules were not taken into the frontmatter: %+v", d)
	}
	if len(d.Repos) != 0 {
		t.Errorf("a nested repository was given a map: %+v", d.Repos)
	}
	for _, want := range []string{"nested under `project`, at `services/billing/`", "`../../harness`", "as `project/ADR-0001`"} {
		if !strings.Contains(fm.Body, want) {
			t.Errorf("the scope does not say %q:\n%s", want, fm.Body)
		}
	}
	md := s.Markdown()
	for _, want := range []string{"## Where this repository sits", "- `contracts/` (contracts)", "- `services/ledger/` (service, planned)"} {
		if !strings.Contains(md, want) {
			t.Errorf("the survey does not say %q", want)
		}
	}
	if strings.Contains(md, "- `services/billing/` (service)") {
		t.Error("the repository was listed as its own sibling")
	}
	harness := t.TempDir()
	os.MkdirAll(filepath.Join(harness, "reverse"), 0o755)
	os.WriteFile(filepath.Join(harness, "reverse", "reverse.md"), []byte("INSTRUCTIONS\n"), 0o644)
	p, err := prompt(harness, repo.Root, s, template)
	if err != nil || !strings.Contains(p, "The design of `project`, the root this repository is nested under") || !strings.Contains(p, "### ADR-0004 — interfaces live in contracts") {
		t.Errorf("a drafting session was not handed the root's design (%v)", err)
	}

	// The first note is dated the day the rules begin, so it carries every
	// section the root asks for, the one the template lacks included.
	note := s.AdoptionNote("# Adopted\n\n## The problem\n\nx\n\n## The plan\n\nFrom {{DATE}}, with `{{HARNESS}}/`.\n\n## Alternatives rejected\n\nx\n\n## Files\n\nx\n")
	if !strings.Contains(note, "From 2026-01-02, with `../../harness/`.") || !strings.Contains(note, "\n## What existing data changes meaning\n\nNothing: ") || strings.Count(note, "\n## ") != 5 {
		t.Errorf("the first note does not carry what the root asks of a note:\n%s", note)
	}

	// The contracts repository's first note says it adds no definition, in
	// the block its checker reads; a service's first note has no such block.
	if strings.Contains(note, "```servers") || s.IsContracts() || s.Contracts() == nil || s.Contracts().Path != "contracts" {
		t.Errorf("a service was taken for the contracts repository, or was not told where it is:\n%s", note)
	}
	asContracts := *parent
	asContracts.Path = "contracts"
	c := survey(repo, &asContracts, nil, "2026-01-02")
	if !c.IsContracts() || c.Contracts() != nil || !strings.Contains(c.AdoptionNote("# Adopted\n"), "```servers\n[]\n```") {
		t.Errorf("the contracts repository's first note does not carry an empty servers block")
	}

	alone := survey(repo, nil, nil, "2026-01-02")
	doc = alone.Skeleton(template)
	if strings.Contains(doc, "nested under") || strings.Contains(doc, "{{") || strings.Contains(alone.Markdown(), "Where this repository sits") {
		t.Errorf("a repository that stands alone was told it is nested, or a placeholder was left:\n%s", doc)
	}
}

// A root's skeleton carries the map: every repository found nested under it,
// with where it is cloned from and a kind read from its name.
func TestARootsSkeletonCarriesTheMap(t *testing.T) {
	repo := repoWith(t, existing)
	s := survey(repo, nil, nil, "2026-01-02")
	s.Map = []design.RepoEntry{{Path: "billing", Kind: kindOf("billing"), Remote: "git@host:x/billing.git"}, {Path: "contracts", Kind: kindOf("contracts")}}
	fm, err := design.ParseFrontmatter(s.Skeleton(template))
	if err != nil {
		t.Fatal(err)
	}
	var d design.Design
	if err := yaml.Unmarshal([]byte(fm.YAML), &d); err != nil {
		t.Fatalf("the frontmatter does not parse: %v\n%s", err, fm.YAML)
	}
	if len(d.Repos) != 2 || d.Repos[0].Path != "billing" || d.Repos[0].Kind != "service" || d.Repos[0].Remote != "git@host:x/billing.git" || d.Repos[1].Kind != "contracts" || d.Repos[1].Remote != "" {
		t.Errorf("the map was written as %+v", d.Repos)
	}
}
