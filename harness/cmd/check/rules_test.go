package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"harness/internal/design"
)

const ruledHead = `---
files:
  - src/**
  - docs/**
rules:
  - id: INV-001
    files: ['src/**/*.ts']
    except: [src/billing/**]
    forbid: 'from "(\.\./)+billing'
---

# shop

- **INV-001** — Nothing outside billing reaches into billing.
`

func day(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func failures(res *Result) string { return strings.Join(res.Failures, "\n") }

func reload(t *testing.T, repo *design.Repo) *design.Repo {
	t.Helper()
	r, err := design.Load(repo.Root)
	if err != nil {
		t.Fatal(err)
	}
	r.Name = repo.Name
	return r
}

func putFile(t *testing.T, repo *design.Repo, p, s string) {
	t.Helper()
	p = filepath.Join(repo.Root, filepath.FromSlash(p))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestARuleRefusesABreachThatIsNotOnItsList(t *testing.T) {
	repo := spined(t, ruledHead, nil, map[string]string{
		"src/cart/add.ts":      "import { charge } from \"../billing/charge\"\n",
		"src/billing/total.ts": "import { rate } from \"../billing/rate\"\n",
		"src/cart/sum.ts":      "export const sum = 1\n",
	})
	res := runChecksOn(repo, day("2026-01-01"))
	if !strings.Contains(failures(res), "INV-001: src/cart/add.ts:1:") {
		t.Fatalf("the breach was not refused:\n%s", failures(res))
	}
	if strings.Contains(failures(res), "src/billing/total.ts") {
		t.Fatalf("a file the rule leaves out was read:\n%s", failures(res))
	}
	if !strings.Contains(failures(res), "-known INV-001") {
		t.Fatalf("the refusal does not say how the rule is adopted:\n%s", failures(res))
	}

	// Adopted: what is there is known, and allowed.
	if err := recordKnown(repo, nil, &strings.Builder{}); err != nil {
		t.Fatal(err)
	}
	if res := runChecksOn(repo, day("2026-01-01")); res.failed() {
		t.Fatalf("a known breach was refused:\n%s", failures(res))
	}

	// A second breach in the same file is new, though the first is known,
	// and the first is still known when lines above it move.
	putFile(t, repo, "src/cart/add.ts", "// moved\nimport { charge } from \"../billing/charge\"\nimport { refund } from \"../billing/refund\"\n")
	res = runChecksOn(reload(t, repo), day("2026-01-01"))
	if len(res.Failures) != 1 || !strings.Contains(res.Failures[0], "src/cart/add.ts:3:") {
		t.Fatalf("want the one new breach refused, got:\n%s", failures(res))
	}
}

func TestARuleOnlyWarnsBeforeItsDay(t *testing.T) {
	head := strings.Replace(ruledHead, "    forbid:", "    from: 2026-06-01\n    forbid:", 1)
	repo := spined(t, head, nil, map[string]string{"src/cart/add.ts": "import { charge } from \"../billing/charge\"\n"})
	res := runChecksOn(repo, day("2026-05-31"))
	if res.failed() {
		t.Fatalf("refused before its day:\n%s", failures(res))
	}
	if len(res.Warnings) == 0 || !strings.Contains(res.Warnings[0], "src/cart/add.ts:1:") {
		t.Fatalf("the breach was not said: %v", res.Warnings)
	}
	if res.Rules[0].From != "2026-06-01" {
		t.Fatalf("the report does not carry the day: %+v", res.Rules[0])
	}
	if res := runChecksOn(repo, day("2026-06-01")); !res.failed() {
		t.Fatal("not refused on its day")
	}
}

func TestAMendedBreachMustComeOffTheList(t *testing.T) {
	repo := spined(t, ruledHead, nil, map[string]string{"src/cart/add.ts": "import { charge } from \"../billing/charge\"\nimport { refund } from \"../billing/refund\"\n"})
	if err := recordKnown(repo, []string{"INV-001"}, &strings.Builder{}); err != nil {
		t.Fatal(err)
	}
	putFile(t, repo, "src/cart/add.ts", "import { charge } from \"../billing/charge\"\n")
	repo = reload(t, repo)
	res := runChecksOn(repo, day("2026-01-01"))
	if !strings.Contains(failures(res), "1 breach(es) on design/rules/INV-001.known are mended") {
		t.Fatalf("a mended breach left on the list was not refused:\n%s", failures(res))
	}
	if err := recordKnown(repo, nil, &strings.Builder{}); err != nil {
		t.Fatal(err)
	}
	if res := runChecksOn(repo, day("2026-01-01")); res.failed() {
		t.Fatalf("still refused once the list was brought in line:\n%s", failures(res))
	}
	b, _ := os.ReadFile(filepath.Join(repo.Root, "design/rules/INV-001.known"))
	if strings.Contains(string(b), "refund") || !strings.Contains(string(b), "charge") {
		t.Fatalf("the list is not the one breach left:\n%s", b)
	}
}

func TestRecordingNeverAddsABreachToAnExistingList(t *testing.T) {
	repo := spined(t, ruledHead, nil, map[string]string{"src/cart/add.ts": "import { charge } from \"../billing/charge\"\n"})
	if err := recordKnown(repo, nil, &strings.Builder{}); err != nil {
		t.Fatal(err)
	}
	putFile(t, repo, "src/cart/new.ts", "import { refund } from \"../billing/refund\"\n")
	repo = reload(t, repo)
	var said strings.Builder
	if err := recordKnown(repo, nil, &said); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(said.String(), "1 new breach(es) are not added") {
		t.Fatalf("recording did not say it left the new breach out: %s", said.String())
	}
	if res := runChecksOn(repo, day("2026-01-01")); !strings.Contains(failures(res), "src/cart/new.ts:1:") {
		t.Fatalf("the new breach was let through:\n%s", failures(res))
	}
}

func TestACheckBelongsToARuleTheDocumentStates(t *testing.T) {
	for name, tc := range map[string]struct{ from, to, want string }{
		"an id the document does not define": {"id: INV-001", "id: INV-009", "INV-009 is checked and not defined there"},
		"two checks in one":                  {"    forbid:", "    require: 'x'\n    forbid:", "exactly one of"},
		"a pattern that does not compile":    {`'from "(\.\./)+billing'`, `'from ('`, "missing closing"},
		"a day that is not one":              {"    forbid:", "    from: soon\n    forbid:", "is not a date"},
	} {
		repo := spined(t, strings.Replace(ruledHead, tc.from, tc.to, 1), nil, map[string]string{"src/a.ts": "x\n"})
		if res := runChecksOn(repo, day("2026-01-01")); !strings.Contains(failures(res), tc.want) {
			t.Errorf("%s: want a refusal holding %q, got:\n%s", name, tc.want, failures(res))
		}
	}
}

func TestASpinesRuleReadsOnlyItsArea(t *testing.T) {
	spine := "---\nrules:\n  - id: INV-001\n    forbid: 'console\\.log'\n---\n\n# cart\n\n- **INV-001** — The cart logs through the logger.\n"
	other := "---\n---\n\n# checkout\n"
	repo := spined(t, shopHead, map[string]string{"cart": spine, "checkout": other}, map[string]string{
		"package.json":        "{}\n",
		"src/cart/add.ts":     "console.log(1)\n",
		"src/checkout/pay.ts": "console.log(2)\n",
	})
	res := runChecksOn(repo, day("2026-01-01"))
	if !strings.Contains(failures(res), "cart/INV-001: src/cart/add.ts:1:") {
		t.Fatalf("the area's breach was not refused under the area's name:\n%s", failures(res))
	}
	if strings.Contains(failures(res), "src/checkout/pay.ts") {
		t.Fatalf("a spine's rule read another area:\n%s", failures(res))
	}
	if err := recordKnown(repo, []string{"cart/INV-001"}, &strings.Builder{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repo.Root, "design/rules/cart/INV-001.known")); err != nil {
		t.Fatalf("an area's list is kept under its name: %v", err)
	}
}

func TestNoFileMayBeWhereTheRuleSaysNone(t *testing.T) {
	head := "---\nfiles: [src/**]\nrules:\n  - id: INV-001\n    absent: ['**/*.proto']\n---\n\n# shop\n\n- **INV-001** — No interface is defined here.\n"
	repo := spined(t, head, nil, map[string]string{"src/api/cart.proto": "syntax = \"proto3\";\n", "src/api/cart.ts": "x\n"})
	res := runChecksOn(repo, day("2026-01-01"))
	if !strings.Contains(failures(res), "INV-001: src/api/cart.proto: is there") || strings.Contains(failures(res), "cart.ts") {
		t.Fatalf("want the one file refused, got:\n%s", failures(res))
	}
}

func TestADocumentOverItsSizeIsSaidAndNotRefused(t *testing.T) {
	head := "---\nfiles: [src/**]\nsize: 10\n---\n\n# shop\n" + strings.Repeat("\nA line.\n", 20)
	repo := spined(t, head, nil, map[string]string{"src/a.ts": "x\n"})
	res := runChecksOn(repo, day("2026-01-01"))
	if res.failed() {
		t.Fatalf("a long document was refused:\n%s", failures(res))
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "over the 10") {
		t.Fatalf("the size was not said: %v", res.Warnings)
	}
	if len(res.Sizes) != 1 || res.Sizes[0].Lines < 40 {
		t.Fatalf("the document was not weighed: %+v", res.Sizes)
	}
}

func TestAReplacedDocumentStillThereIsNamed(t *testing.T) {
	head := "---\nfiles: [src/**, docs/**]\nreplaces: [docs/ARCHITECTURE.md, docs/rules]\n---\n\n# shop\n"
	repo := spined(t, head, nil, map[string]string{"src/a.ts": "x\n", "docs/ARCHITECTURE.md": "old\n"})
	res := runChecksOn(repo, day("2026-01-01"))
	if res.failed() {
		t.Fatalf("refused:\n%s", failures(res))
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "docs/ARCHITECTURE.md is still there") {
		t.Fatalf("want the one document still there named, got: %v", res.Warnings)
	}
}

func TestTheReportCarriesEachRuleAndWhatItFound(t *testing.T) {
	repo := spined(t, ruledHead, nil, map[string]string{"src/cart/add.ts": "import { charge } from \"../billing/charge\"\n"})
	res := runChecksOn(repo, day("2026-01-01"))
	md := res.Markdown(repo)
	for _, want := range []string{"shop is refused", "| INV-001 | forbid | in force | 1 | 0 |", "### Refused", "src/cart/add.ts:1:", "| design/DESIGN.md |"} {
		if !strings.Contains(md, want) {
			t.Errorf("the report lacks %q:\n%s", want, md)
		}
	}
	sum := filepath.Join(t.TempDir(), "summary.md")
	t.Setenv("GITHUB_STEP_SUMMARY", sum)
	out := filepath.Join(t.TempDir(), "report.md")
	if err := writeReport(md, out); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{sum, out} {
		if b, _ := os.ReadFile(p); string(b) != md {
			t.Errorf("%s does not hold the report", p)
		}
	}
}
