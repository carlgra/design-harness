package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"harness/internal/design"
)

// shop is a repository with a design that defines two areas, and the
// harness's templates and prompts beside it.
func shop(t *testing.T) (root, harness string) {
	t.Helper()
	root = t.TempDir()
	files := map[string]string{
		"design/DESIGN.md":              "---\nfiles: [go.mod]\nspines:\n  - name: cart\n    files: [internal/cart/**]\n  - name: checkout\n    files: [internal/checkout/**]\n---\n\n# shop\n\nTHE DESIGN OF THE SHOP\n\n### ADR-0001 — Go\n",
		"design/spines/checkout.md":     "---\ntests: []\n---\n# checkout\n\nCHECKOUT RULES\n",
		"go.mod":                        "module shop\n",
		"internal/cart/cart.go":         "package cart\n",
		"internal/cart/cart_test.go":    "package cart\nfunc TestAnEmptyCartIsRefused(t *testing.T) {}\n",
		"internal/cart/DECISIONS.md":    "# decisions\n",
		"internal/checkout/pay.go":      "package checkout\n",
		"internal/checkout/pay_test.go": "package checkout\nfunc TestPay(t *testing.T) {}\n",
	}
	for name, body := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	wd, _ := os.Getwd()
	return root, filepath.Join(wd, "..", "..")
}

func step(t *testing.T, name, root, harness, area string, force bool, reply string) error {
	t.Helper()
	repo, err := design.Walk(root)
	if err != nil {
		t.Fatal(err)
	}
	ask := func(model, prompt string) (string, error) { return reply, nil }
	return spineStep(name, root, harness, area, force, repo, "2026-01-31", ask, "m")
}

// A spine's skeleton is the first document of an area the design already
// defines: it passes as a spine, says which files the area holds, and is
// never written for an area the design does not define or over one that is
// there.
func TestASpinesSkeletonIsWrittenOnlyForAnAreaTheDesignDefines(t *testing.T) {
	root, harness := shop(t)
	if err := step(t, "skeleton", root, harness, "returns", false, ""); err == nil || !strings.Contains(err.Error(), "defines no area called returns") {
		t.Fatalf("an area the design does not define was given a spine: %v", err)
	}
	if err := step(t, "skeleton", root, harness, "checkout", false, ""); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("a spine that exists was overwritten: %v", err)
	}
	if err := step(t, "skeleton", root, harness, "cart", false, ""); err != nil {
		t.Fatal(err)
	}
	repo, err := design.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	sp := repo.Spine("cart")
	if sp == nil || sp.Err != nil {
		t.Fatalf("the skeleton does not read as a spine: %+v", sp)
	}
	for _, want := range []string{SkeletonMark, "`internal/cart/**`", "cart/ADR-0001", repo.Name + "/ADR-0001"} {
		if !strings.Contains(sp.Doc, want) {
			t.Fatalf("the skeleton lacks %q:\n%s", want, sp.Doc)
		}
	}
}

// A session drafting a spine is handed the area and the design, and nothing
// of any other area: the survey lists the area's files and tests alone, and
// no other spine is in the prompt.
func TestASpineIsDraftedFromItsAreaAndTheDesignAlone(t *testing.T) {
	root, harness := shop(t)
	repo, _ := design.Walk(root)
	full, _ := design.Load(root)
	a, err := areaOf(full.Design, "cart")
	if err != nil {
		t.Fatal(err)
	}
	s := areaSurvey(repo, a, "2026-01-31")
	if s.Files != 3 || len(s.Tests) != 1 || s.Tests[0] != "TestAnEmptyCartIsRefused" {
		t.Fatalf("the survey is not of the area alone: %d files, tests %v", s.Files, s.Tests)
	}
	template, _ := os.ReadFile(filepath.Join(harness, "templates", "spine.md"))
	p, err := spinePrompt(harness, s, "shop", string(template), "THE DESIGN OF THE SHOP")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"THE DESIGN OF THE SHOP", "internal/cart/DECISIONS.md", "TestAnEmptyCartIsRefused", SpineHeader("cart")} {
		if !strings.Contains(p, want) {
			t.Fatalf("the prompt lacks %q", want)
		}
	}
	for _, not := range []string{"CHECKOUT RULES", "TestPay", "internal/checkout/pay.go", "{{", SkeletonMark} {
		if strings.Contains(p, not) {
			t.Fatalf("the prompt carries %q, which is not this area's", not)
		}
	}
}

// A draft is written as the area's spine, only over a skeleton unless
// forced, and a reply that is not a spine is kept and not written.
func TestADraftedSpineReplacesOnlyASkeleton(t *testing.T) {
	root, harness := shop(t)
	good := "Here it is.\n" + SpineHeader("cart") + "\n---\ntests: [TestAnEmptyCartIsRefused]\n---\n\n# cart\n\nTHE CART\n"
	if err := step(t, "draft", root, harness, "cart", false, good); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(root, "design/spines/cart.md"))
	if !strings.HasPrefix(string(b), "---\n") || !strings.Contains(string(b), "THE CART") {
		t.Fatalf("the spine was not written as replied:\n%s", b)
	}
	if err := step(t, "draft", root, harness, "cart", false, good); err == nil || !strings.Contains(err.Error(), "no longer a skeleton") {
		t.Fatalf("a spine someone took on was replaced: %v", err)
	}
	if err := step(t, "draft", root, harness, "cart", true, strings.Replace(good, "THE CART", "AGAIN", 1)); err != nil {
		t.Fatal(err)
	}
	err := step(t, "draft", root, harness, "cart", true, "no header at all")
	if err == nil || !strings.Contains(err.Error(), "kept in design/reverse-reply.md") {
		t.Fatalf("a reply that is no spine was not kept: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "design/spines/cart.md")); !strings.Contains(string(b), "AGAIN") {
		t.Fatal("a reply that is no spine was written")
	}
}

// Where a drafted design divides the code into areas, each area with no
// spine is given its skeleton, and a spine already there is left alone.
func TestEveryAreaADraftDefinesIsGivenASkeleton(t *testing.T) {
	root, harness := shop(t)
	repo, _ := design.Walk(root)
	wrote, err := skeletonsForAreas(root, harness, repo, "2026-01-31")
	if err != nil {
		t.Fatal(err)
	}
	if len(wrote) != 1 || wrote[0] != "cart" {
		t.Fatalf("expected the one missing spine, cart, to be written; got %v", wrote)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "design/spines/checkout.md")); !strings.Contains(string(b), "CHECKOUT RULES") {
		t.Fatal("a spine that was there was replaced")
	}
	if b, _ := os.ReadFile(filepath.Join(root, "design/spines/cart.md")); !strings.Contains(string(b), SkeletonMark) {
		t.Fatal("the missing spine was not written as a skeleton")
	}
}
