package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"harness/internal/design"
	"harness/internal/history"
	"harness/internal/rules"
)

// past is a git repository built one change at a time.
type past struct {
	t    *testing.T
	root string
}

func newPast(t *testing.T) *past {
	t.Helper()
	design.ForgetGitEnv()
	p := &past{t: t, root: t.TempDir()}
	p.git("init", "-q", "-b", "main")
	return p
}

func (p *past) git(args ...string) {
	p.t.Helper()
	cmd := exec.Command("git", append([]string{"-C", p.root, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false"}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		p.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// change commits the files given; a file given as "" is deleted.
func (p *past) change(subject string, files map[string]string) {
	p.t.Helper()
	for name, s := range files {
		path := filepath.Join(p.root, filepath.FromSlash(name))
		if s == "" {
			os.Remove(path)
			continue
		}
		os.MkdirAll(filepath.Dir(path), 0o755)
		if err := os.WriteFile(path, []byte(s), 0o644); err != nil {
			p.t.Fatal(err)
		}
	}
	p.git("add", "-A")
	p.git("commit", "-q", "-m", subject)
}

func (p *past) load() (*design.Repo, []history.Commit) {
	p.t.Helper()
	repo, err := design.Load(p.root)
	if err != nil {
		p.t.Fatal(err)
	}
	repo.Name = "shop"
	commits, err := history.Commits(p.root, 100)
	if err != nil {
		p.t.Fatal(err)
	}
	return repo, commits
}

const pastHead = `---
files: [package.json, tools/**]
spines:
  - name: cart
    files: [src/cart/**]
  - name: billing
    files: [src/billing/**]
  - name: search
    files: [src/search/**]
rules:
  - id: INV-001
    files: ['src/**/*.ts']
    except: [src/billing/**]
    forbid: 'from "(\.\./)+billing'
---

# shop

- **INV-001** — Nothing outside billing reaches into billing.
`

func shopPast(t *testing.T) *past {
	p := newPast(t)
	p.change("feat: the shop", map[string]string{
		"design/DESIGN.md":         pastHead,
		"design/spines/cart.md":    "---\n---\n# cart\n",
		"design/spines/billing.md": "---\n---\n# billing\n",
		"design/spines/search.md":  "---\n---\n# search\n",
		"package.json":             "{}\n",
		"src/cart/add.ts":          "export const add = 1\n",
		"src/billing/charge.ts":    "export const charge = 1\n",
		"src/search/find.ts":       "import { rate } from \"../billing/rate\"\n",
		"src/billing/rate.ts":      "export const rate = 1\n",
		"tools/lint.sh":            "true\n",
	})
	p.change("feat(cart): totals", map[string]string{"src/cart/total.ts": "export const total = 1\n"})
	p.change("feat(cart): charge from the cart", map[string]string{
		"src/cart/add.ts":       "// charge\nimport { charge } from \"../billing/charge\"\nexport const add = 1\n",
		"src/billing/charge.ts": "export const charge = 2\n",
	})
	p.change("refactor(cart): move a line", map[string]string{"src/cart/add.ts": "// charge\n// moved\nimport { charge } from \"../billing/charge\"\nexport const add = 1\n"})
	p.change("fix(search): no rate", map[string]string{"src/search/find.ts": "export const find = 1\n"})
	p.change("docs: the design", map[string]string{"design/DESIGN.md": pastHead + "\nMore.\n"})
	p.change("chore: tools and cart", map[string]string{"tools/lint.sh": "false\n", "src/cart/total.ts": "export const total = 2\n"})
	return p
}

func TestAChangeThatAddsABreachWouldHaveBeenRefused(t *testing.T) {
	p := shopPast(t)
	repo, commits := p.load()
	list, wrong := rules.Compile(repo)
	if len(wrong) > 0 {
		t.Fatal(wrong)
	}
	trials, err := tryRules(repo, list, commits, false)
	if err != nil {
		t.Fatal(err)
	}
	tr := trials[0]
	// The first change brought one breach in with the tree, the charge from
	// the cart brought another, moving a line brought none, and the fix to
	// search mended the first.
	if len(tr.Refused) != 2 || tr.Added != 2 || tr.Mended != 1 {
		t.Fatalf("refused %d, added %d, mended %d", len(tr.Refused), tr.Added, tr.Mended)
	}
	if got := tr.Refused[0].Commit.Subject; got != "feat(cart): charge from the cart" {
		t.Fatalf("the most recent refusal is %q", got)
	}
	if f := tr.Refused[0].Added[0]; f.File != "src/cart/add.ts" || f.Line != 2 {
		t.Fatalf("the breach is %+v", f)
	}
	if tr.Now != 1 || tr.Start() != 0 || tr.Reads != 3 {
		t.Fatalf("now %d, start %d, reads %d", tr.Now, tr.Start(), tr.Reads)
	}
	text := Report{Repo: "shop", Commits: commits, Rules: trials}.Text()
	for _, want := range []string{"would have refused 2 of 7 changes (29%)", "feat(cart): charge from the cart", "src/cart/add.ts:2:"} {
		if !strings.Contains(text, want) {
			t.Errorf("the report lacks %q:\n%s", want, text)
		}
	}
	// Nothing was changed by reading the history.
	if out, _ := exec.Command("git", "-C", p.root, "status", "--porcelain").Output(); len(out) != 0 {
		t.Fatalf("the backtest changed the tree:\n%s", out)
	}
}

func TestACommandRuleIsTriedInACheckoutOfEachChange(t *testing.T) {
	p := shopPast(t)
	repo, commits := p.load()
	repo.Design.Rules = []design.Rule{{ID: "INV-001", Run: `grep -rn 'billing/' src/cart src/search || true`}}
	list, _ := rules.Compile(repo)
	trials, err := tryRules(repo, list, commits, false)
	if err != nil {
		t.Fatal(err)
	}
	if trials[0].Skipped == "" {
		t.Fatal("a command was tried without being asked for")
	}
	trials, err = tryRules(repo, list, commits, true)
	if err != nil {
		t.Fatal(err)
	}
	tr := trials[0]
	if len(tr.Refused) != 1 || tr.Refused[0].Commit.Subject != "feat(cart): charge from the cart" || tr.Mended != 1 {
		t.Fatalf("refused %+v, mended %d", tr.Refused, tr.Mended)
	}
	if out, _ := exec.Command("git", "-C", p.root, "worktree", "list").Output(); strings.Count(string(out), "\n") != 1 {
		t.Fatalf("the checkout was left behind:\n%s", out)
	}
}

func TestAreasAreScoredByTheChangesThatStayedInside(t *testing.T) {
	p := shopPast(t)
	repo, commits := p.load()
	s := scoreAreas(repo, commits)
	// Seven changes, one to documents only. Of the six to code: the first
	// touched all three areas, the charge touched two, three stayed in one
	// (one of them with a tool no area holds).
	if s.Code != 6 || s.By != [4]int{0, 4, 1, 1} || s.Mixed != 1 {
		t.Fatalf("code %d, by %v, mixed %d", s.Code, s.By, s.Mixed)
	}
	cart := s.Areas[0]
	if cart.Name != "cart" || cart.Changes != 5 || cart.Alone != 3 {
		t.Fatalf("cart: %+v", cart)
	}
	if len(s.Pairs) == 0 || s.Pairs[0].A != "billing" || s.Pairs[0].B != "cart" || s.Pairs[0].Together != 2 {
		t.Fatalf("pairs: %+v", s.Pairs)
	}
	if len(s.Loose) == 0 || s.Loose[0].Dir != "tools" {
		t.Fatalf("loose: %+v", s.Loose)
	}
	text := s.text()
	if !strings.Contains(text, "67% of 6 changes to code stayed inside one area or none") {
		t.Errorf("the score is not said:\n%s", text)
	}

	// With no areas defined, the directories that change together are the
	// evidence of where to draw them.
	repo.Design.Spines = nil
	s = scoreAreas(repo, commits)
	if s.Defined != 0 || len(s.Suggest) == 0 {
		t.Fatalf("no suggestion: %+v", s)
	}
}
