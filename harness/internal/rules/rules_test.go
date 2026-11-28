package rules

import (
	"reflect"
	"testing"

	"harness/internal/design"
)

func compiled(t *testing.T, d design.Rule) *Rule {
	t.Helper()
	list, wrong := Compile(&design.Repo{Design: design.Design{Rules: []design.Rule{d}}})
	if len(wrong) > 0 {
		t.Fatal(wrong)
	}
	return list[0]
}

func TestABreachIsKnownByItsFileAndTextNotItsLine(t *testing.T) {
	r := compiled(t, design.Rule{ID: "INV-001", Forbid: `TODO`})
	before := r.Scan("a.go", []byte("x\n\tTODO   one\nTODO two\n"))
	after := r.Scan("a.go", []byte("x\ny\nz\nTODO one\nTODO two\nTODO two\n"))
	if before[0].Key() != "a.go: TODO one" || before[0].Line != 2 {
		t.Fatalf("key %q line %d", before[0].Key(), before[0].Line)
	}
	fresh, gone := Split(after, Keys(before))
	if len(fresh) != 1 || fresh[0].Line != 6 || len(gone) != 0 {
		t.Fatalf("a breach that moved is the same breach and a second copy is new: fresh %v gone %v", fresh, gone)
	}
	fresh, gone = Split(before[:1], Keys(before))
	if len(fresh) != 0 || !reflect.DeepEqual(gone, []string{"a.go: TODO two"}) {
		t.Fatalf("fresh %v gone %v", fresh, gone)
	}
	if got := r.Scan("a.bin", []byte("TODO\x00")); got != nil {
		t.Fatalf("a file that is not text was read: %v", got)
	}
	req := compiled(t, design.Rule{ID: "INV-001", Require: `^// Copyright`})
	if got := req.Scan("a.go", []byte("package a\n")); len(got) != 1 || got[0].File != "a.go" {
		t.Fatalf("a file without the line is a breach: %v", got)
	}
	if got := req.Scan("a.go", []byte("// Copyright\n")); got != nil {
		t.Fatalf("a file with the line is none: %v", got)
	}
}

func TestACommandsLinesAreItsBreaches(t *testing.T) {
	dir := t.TempDir()
	r := compiled(t, design.Rule{ID: "ADR-0001", Run: `printf 'src/a.ts:12:3: reaches into billing\nloose words\n\n'; exit 1`})
	got, err := r.RunAt(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []Finding{{File: "src/a.ts", Line: 12, Text: "reaches into billing"}, {Text: "loose words"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
	if got[0].Key() != "src/a.ts: reaches into billing" {
		t.Fatalf("a command's breach is known without its line: %q", got[0].Key())
	}
	clean := compiled(t, design.Rule{ID: "ADR-0001", Run: `true`})
	if got, err := clean.RunAt(dir); err != nil || len(got) != 0 {
		t.Fatalf("a command that prints nothing and succeeds found nothing: %v %v", got, err)
	}
	broken := compiled(t, design.Rule{ID: "ADR-0001", Run: `echo no such tool >&2; exit 127`})
	if _, err := broken.RunAt(dir); err == nil {
		t.Fatal("a command that failed and printed no breach was taken for a clean tree")
	}
}
