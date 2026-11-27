package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"harness/internal/design"
)

// spined is a repository on disk with a design, areas and their spines,
// loaded as the checker loads it.
func spined(t *testing.T, head string, spines, files map[string]string) *design.Repo {
	t.Helper()
	root := t.TempDir()
	put := func(p, s string) {
		p = filepath.Join(root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	put("design/DESIGN.md", head)
	for name, s := range spines {
		put("design/spines/"+name+".md", s)
	}
	for p, s := range files {
		put(p, s)
	}
	repo, err := design.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	repo.Name = "shop"
	return repo
}

const shopHead = `---
files:
  - package.json
spines:
  - name: cart
    files: [src/cart/**]
  - name: checkout
    files: [src/checkout/**]
---

# shop

### ADR-0001 — one design, an area each

The cart is held to cart/INV-001.
`

const cartSpine = `---
tests: [TestAnEmptyCartIsRefused]
---

# cart

- **INV-001** — An empty cart is refused. *(test: TestAnEmptyCartIsRefused)*

### ADR-0001 — carts are kept in the session

Under shop/ADR-0001. Checkout reads a cart as checkout/ADR-0001 says.
`

const checkoutSpine = `---
tests: []
---

# checkout

### ADR-0001 — a cart is read once
`

func shopFiles() map[string]string {
	return map[string]string{
		"package.json":          "{}",
		"src/cart/cart.go":      "package cart",
		"src/cart/cart_test.go": "package cart\nfunc TestAnEmptyCartIsRefused(t *testing.T) {}\n",
		"src/checkout/pay.go":   "package checkout",
	}
}

func refusals(t *testing.T, repo *design.Repo, want ...string) {
	t.Helper()
	res := runChecks(repo)
	if len(res.Failures) != len(want) {
		t.Fatalf("expected %d refusal(s), got %v", len(want), res.Failures)
	}
	for i, w := range want {
		if !strings.Contains(res.Failures[i], w) {
			t.Fatalf("expected %q, got %q", w, res.Failures[i])
		}
	}
}

// A design may divide the code into areas, each with a spine of its own: a
// file an area holds is accounted for, each document numbers its own ids,
// and a citation across documents is by the other's name.
func TestADesignDividedIntoAreasPasses(t *testing.T) {
	repo := spined(t, shopHead, map[string]string{"cart": cartSpine, "checkout": checkoutSpine}, shopFiles())
	refusals(t, repo)
	if res := runChecks(repo); res.Claimed != 4 || res.InSpines != 3 {
		t.Fatalf("expected 4 files accounted for, 3 of them in spines; got %d and %d", res.Claimed, res.InSpines)
	}
}

// The areas the design defines and the spines on disk are the same set.
func TestAnAreaHasASpineAndASpineHasAnArea(t *testing.T) {
	repo := spined(t, shopHead, map[string]string{"cart": cartSpine, "returns": checkoutSpine}, shopFiles())
	refusals(t, repo,
		"design/spines/checkout.md: the spine of the area checkout: no such file",
		"design/spines/returns.md: a spine for an area the design does not define",
	)
}

// A file is under one spine: one that two areas hold is refused, and one no
// area and no glob of the design's holds is refused as before.
func TestAFileIsUnderOneSpine(t *testing.T) {
	head := strings.Replace(shopHead, "files: [src/checkout/**]", "files: [src/checkout/**, src/cart/cart.go]", 1)
	files := shopFiles()
	files["src/returns/back.go"] = "package returns"
	repo := spined(t, head, map[string]string{"cart": cartSpine, "checkout": checkoutSpine}, files)
	refusals(t, repo,
		"src/cart/cart.go: held by 2 areas, cart and checkout",
		"src/returns/back.go: the design accounts for no such file",
	)
}

// A spine's tests are its area's: a test it names that exists only in
// another area's files is refused.
func TestASpinesTestIsInItsOwnArea(t *testing.T) {
	spine := strings.Replace(checkoutSpine, "tests: []", "tests: [TestAnEmptyCartIsRefused]", 1)
	files := shopFiles()
	files["src/checkout/pay_test.go"] = "package checkout\nfunc TestPay(t *testing.T) {}\n"
	repo := spined(t, shopHead, map[string]string{"cart": cartSpine, "checkout": spine}, files)
	refusals(t, repo, "design/spines/checkout.md names the test TestAnEmptyCartIsRefused")
}

// Each document numbers its own ids, and a citation of another document's is
// looked up there: an area's, the repository's own, each refused when it is
// not defined, while a third repository's is not looked up.
func TestACitationAcrossDocumentsLandsInTheOneNamed(t *testing.T) {
	spine := cartSpine + "\nAlso cart/INV-001, shop/ADR-0009, checkout/INV-004 and elsewhere/ADR-0500, and other/cart/ADR-0077.\n\nADR-0002 is not defined here.\n"
	repo := spined(t, shopHead, map[string]string{"cart": spine, "checkout": checkoutSpine}, shopFiles())
	refusals(t, repo,
		"design/spines/cart.md cites shop/ADR-0009",
		"design/spines/cart.md cites checkout/INV-004",
		"design/spines/cart.md cites ADR-0002",
	)
}

// An area's name qualifies an id and names a file: one that could be read
// as a repository's, is given twice, or holds no files is refused.
func TestAnAreasNameCanQualifyAnId(t *testing.T) {
	head := "---\nfiles: [package.json]\nspines:\n  - name: shop\n    files: [src/**]\n  - name: Cart Rules\n    files: [a/**]\n  - name: empty\n---\n\n# shop\n"
	repo := spined(t, head, map[string]string{"shop": checkoutSpine, "empty": checkoutSpine}, map[string]string{"package.json": "{}"})
	refusals(t, repo,
		"the area shop has a repository's name",
		`"Cart Rules" is not a name an area can have`,
		"the area empty lists no files:",
	)
}

// Before a change, the documents to read are the design and the spine of
// each area the paths fall in, and no other.
func TestTheDocumentsToReadAreTheDesignAndTheAreasSpine(t *testing.T) {
	repo := spined(t, shopHead, map[string]string{"cart": cartSpine, "checkout": checkoutSpine}, shopFiles())
	cases := map[string][]string{
		"src/cart/cart.go": {"design/DESIGN.md", "design/spines/cart.md"},
		"src/checkout":     {"design/DESIGN.md", "design/spines/checkout.md"},
		"src":              {"design/DESIGN.md", "design/spines/cart.md", "design/spines/checkout.md"},
		"package.json":     {"design/DESIGN.md"},
	}
	for p, want := range cases {
		if got := documentsFor(repo, []string{p}); !reflect.DeepEqual(got, want) {
			t.Fatalf("%s: expected %v, got %v", p, want, got)
		}
	}
}
