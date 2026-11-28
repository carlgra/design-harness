package main

import (
	"errors"
	"harness/internal/design"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"harness/internal/session"
	"testing"
)

// The script judges nothing itself: a session's reply is a pass
// only when it ends in the verdict form, one failure fails the run, and a
// session is handed its prompt, the design and, for readiness, the change —
// and nothing else. The model is faked; what these pin is the script.

func TestAVerdictIsReadFromTheLastLineAndNothingElseCounts(t *testing.T) {
	cases := []struct {
		reply       string
		pass, found bool
	}{
		{"Reasons.\n\nVERDICT: PASS\n", true, true},
		{"Reasons.\n\nVERDICT: FAIL", false, true},
		{"verdict: pass", false, false},                // the prefix is exact
		{"VERDICT: pass", true, true},                  // the word is not
		{"VERDICT: PASS\n\nMore after.", false, false}, // not the last line
		{"VERDICT: MAYBE", false, false},
		{"", false, false},
		{"I think it passes.", false, false},
	}
	for _, c := range cases {
		pass, found := parseVerdict(c.reply, "VERDICT:")
		if pass != c.pass || found != c.found {
			t.Errorf("%q: got pass=%v found=%v, want pass=%v found=%v", c.reply, pass, found, c.pass, c.found)
		}
	}
}

func fakeRunner(replies map[string]string, err error, seen *[]string) session.Runner {
	return func(model, prompt string) (string, error) {
		*seen = append(*seen, prompt)
		if err != nil {
			return "", err
		}
		for key, reply := range replies {
			if strings.HasPrefix(prompt, key) {
				return reply, nil
			}
		}
		return "", errors.New("no canned reply")
	}
}

var both = []test{
	{Name: "comprehension", Prompt: "COMPREHENSION PROMPT\n\n---\n"},
	{Name: "readiness", Prompt: "READINESS PROMPT\n\n---\n"},
}

func TestASessionIsHandedItsPromptTheDesignAndNothingElse(t *testing.T) {
	var seen []string
	r := fakeRunner(map[string]string{"COMPREHENSION": "ok\nVERDICT: PASS", "READINESS": "ok\nVERDICT: PASS"}, nil, &seen)
	head := "# x\n\nTHE DESIGN\n"
	change := "The change.\n"
	vs := run(both, head, change, session.Settings{Model: "m", Verdict: "VERDICT:"}, r)
	if len(seen) != 2 {
		t.Fatalf("expected two sessions, got %d", len(seen))
	}
	want := []string{
		"COMPREHENSION PROMPT\n\n---\n\n# x\n\nTHE DESIGN\n",
		"READINESS PROMPT\n\n---\n\n# x\n\nTHE DESIGN\n\n---\n\nThe change.\n",
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Errorf("session %d was handed:\n%q\nwant:\n%q", i, seen[i], want[i])
		}
	}
	for _, v := range vs {
		if !v.Pass || v.Err != nil {
			t.Errorf("%s: expected a pass, got %+v", v.Test, v)
		}
		if v.Model != "m" {
			t.Errorf("%s: the model is recorded with the verdict, got %q", v.Test, v.Model)
		}
	}
}

func TestOneFailureFailsTheRunAndIsReportedWithItsReasons(t *testing.T) {
	var seen []string
	r := fakeRunner(map[string]string{
		"COMPREHENSION": "fine\nVERDICT: PASS",
		"READINESS":     "The weight is pinned in ADR-0014 §3.\nVERDICT: FAIL",
	}, nil, &seen)
	vs := run(both, "head", "change", session.Settings{Model: "m", Verdict: "VERDICT:"}, r)
	failed := 0
	for _, v := range vs {
		if !v.Pass {
			failed++
			if v.Test != "readiness" || !strings.Contains(v.Reply, "pinned in ADR-0014") {
				t.Errorf("the failure is not the readiness session's with its reasons: %+v", v)
			}
		}
	}
	if failed != 1 {
		t.Fatalf("expected exactly one failure, got %d", failed)
	}
	md := markdown("x", []string{"design/DESIGN.md"}, vs)
	for _, want := range []string{"readiness: FAIL", "pinned in ADR-0014", "comprehension: PASS", "model m"} {
		if !strings.Contains(md, want) {
			t.Errorf("the markdown for the pull request does not say %q:\n%s", want, md)
		}
	}
}

func TestAReplyWithNoVerdictLineIsAFailureNotAPass(t *testing.T) {
	var seen []string
	r := fakeRunner(map[string]string{"COMPREHENSION": "I think this is fine."}, nil, &seen)
	vs := run(both[:1], "head", "change", session.Settings{Model: "m", Verdict: "VERDICT:"}, r)
	if vs[0].Pass || vs[0].Err == nil || !strings.Contains(vs[0].Err.Error(), "nothing else counts") {
		t.Fatalf("expected a failure for want of a verdict line, got %+v", vs[0])
	}
}

func TestASessionThatCannotRunIsAFailure(t *testing.T) {
	var seen []string
	r := fakeRunner(nil, errors.New("claude: not found"), &seen)
	vs := run(both[:1], "head", "change", session.Settings{Model: "m", Verdict: "VERDICT:"}, r)
	if vs[0].Pass || vs[0].Err == nil {
		t.Fatalf("expected a failure, got %+v", vs[0])
	}
	if !strings.Contains(headline(vs[0]), "not found") {
		t.Fatalf("the reason is not reported: %s", headline(vs[0]))
	}
}

// Asked about one area, a session is handed the design and that area's
// spine, and no other area's; an area the design does not define is an
// error, not a read of the design alone.
func TestAnAreaIsReadWithTheDesignAndWithoutTheOtherAreas(t *testing.T) {
	repo := &design.Repo{Spines: []design.Spine{
		{Name: "cart", Path: "design/spines/cart.md", Doc: "---\ntests: []\n---\n# cart\n\nCART RULES\n"},
		{Name: "checkout", Path: "design/spines/checkout.md", Doc: "CHECKOUT RULES"},
	}}
	head, reading, err := handed(repo, "# shop\n\nTHE DESIGN\n", "cart")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(head, "THE DESIGN") || !strings.Contains(head, "CART RULES") || strings.Contains(head, "CHECKOUT RULES") {
		t.Fatalf("handed the wrong documents:\n%s", head)
	}
	if strings.Index(head, "THE DESIGN") > strings.Index(head, "CART RULES") {
		t.Fatal("the design is read before the spine")
	}
	if len(reading) != 2 || reading[1] != "design/spines/cart.md" {
		t.Fatalf("reading %v", reading)
	}
	if _, _, err := handed(repo, "x", "returns"); err == nil {
		t.Fatal("an area the design does not define was read")
	}
	if head, reading, _ := handed(repo, "x", ""); head != "x" || len(reading) != 1 {
		t.Fatal("with no area named, the design alone is handed over")
	}
}

const aDesign = "---\nfiles: [app/**]\n---\n\n# shop\n\n## Scope\n\nA shop.\n\n## What must always hold\n\n- **INV-001** — a cart is never negative.\n\n## Decisions\n\n### ADR-0001 — one cart a session\n\nNot recorded. TODO(design)\n\n## The files\n\n`app/` is the shop.\n"

// A document that is not a design is known for one from the document alone:
// a skeleton by the line it carries, frontmatter that accounts for no file,
// a document without the sections that hold what governs, and, asked about
// one area, a spine that is still a skeleton. A draft with marks still in it
// is a design, and so is one that only speaks of the skeleton's mark.
func TestADocumentThatIsNotADesignIsKnownWithoutAskingASession(t *testing.T) {
	files := design.Design{Files: []string{"app/**"}}
	areas := design.Design{Spines: []design.Area{{Name: "cart", Files: []string{"app/cart/**"}}}}
	skeleton := strings.Replace(aDesign, "# shop\n", "# shop\n\n"+design.SkeletonMark+"\n", 1)
	cases := []struct {
		name  string
		repo  *design.Repo
		main  string
		spine string
		why   string // "" when it is a design
		first string
	}{
		{name: "a design, with a mark left in it", repo: &design.Repo{Design: files}, main: aDesign},
		{name: "a design that speaks of the mark", repo: &design.Repo{Design: files}, main: aDesign + "\nA skeleton carries `" + design.SkeletonMark + "`.\n"},
		{name: "a design whose files are all in areas", repo: &design.Repo{Design: areas}, main: aDesign},
		{name: "a skeleton", repo: &design.Repo{Design: files}, main: skeleton,
			why: "design/DESIGN.md is still a skeleton", first: "go -C ../harness run ./cmd/reverse draft  and answer each TODO(design)"},
		{name: "no files", repo: &design.Repo{}, main: aDesign,
			why: "accounts for no files", first: "under files:"},
		{name: "a README under frontmatter", repo: &design.Repo{Design: files}, main: "---\nfiles: [app/**]\n---\n\n# shop\n\n## Install\n\nRun make.\n\n```\n## Decisions\n```\n",
			why: `has no section "What must always hold", "Decisions", "Files"`, first: "../harness/templates/DESIGN.md"},
		{name: "one section missing", repo: &design.Repo{Design: files}, main: strings.Replace(aDesign, "## Decisions", "## Choices", 1),
			why: `has no section "Decisions", so`, first: "write them"},
		{name: "an area whose spine is a skeleton", spine: "cart", main: aDesign,
			repo: &design.Repo{Design: areas, Spines: []design.Spine{{Name: "cart", Path: "design/spines/cart.md", Doc: "# cart\n\n" + design.SkeletonMark + "\n"}}},
			why:  "design/spines/cart.md is still a skeleton", first: "run ./cmd/reverse draft -spine cart"},
		{name: "an area whose spine is drafted", spine: "cart", main: aDesign,
			repo: &design.Repo{Design: areas, Spines: []design.Spine{{Name: "cart", Path: "design/spines/cart.md", Doc: "# cart\n\nCART RULES\n"}}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			why, first := notADesign(c.repo, c.main, c.spine, "../harness")
			if c.why == "" {
				if why != "" {
					t.Fatalf("a design was refused: %s", why)
				}
				return
			}
			if !strings.Contains(why, c.why) {
				t.Errorf("why is %q, want it to hold %q", why, c.why)
			}
			if !strings.Contains(first, c.first) {
				t.Errorf("what to do first is %q, want it to hold %q", first, c.first)
			}
		})
	}
}

// The command itself, run on a skeleton, starts no session: it refuses,
// says there is no design to test and what to do first, and the stand-in for
// the claude tool is never run. Run on a design, the same stand-in is run,
// which is what shows the first case would have started one.
func TestADocumentThatIsNotADesignIsRefusedBeforeAnySessionIsStarted(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh to stand in for the claude tool")
	}
	skeleton := strings.Replace(aDesign, "# shop\n", "# shop\n\n"+design.SkeletonMark+"\n", 1)
	cold := func(doc string) (out string, err error, started bool) {
		t.Helper()
		root, _ := filepath.EvalSymlinks(t.TempDir())
		marker := filepath.Join(root, "a-session-was-started")
		fake := filepath.Join(root, "claude")
		for name, body := range map[string]string{
			"design/DESIGN.md":                  doc,
			"harness/go.mod":                    "module harness\n",
			"harness/coldread/coldread.yaml":    "model: a-model\nverdict: \"VERDICT:\"\n",
			"harness/coldread/comprehension.md": "Explain it.\n",
			"claude":                            "#!/bin/sh\ncat >/dev/null\n: > '" + marker + "'\necho 'VERDICT: PASS'\n",
		} {
			p := filepath.Join(root, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(body), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		cmd := exec.Command("go", "run", ".", "-root", root, "-only", "comprehension", "-claude", fake)
		for _, kv := range os.Environ() {
			if !strings.HasPrefix(kv, "GIT_") {
				cmd.Env = append(cmd.Env, kv)
			}
		}
		cmd.Env = append(cmd.Env, "GOWORK=off")
		b, err := cmd.CombinedOutput()
		_, statErr := os.Stat(marker)
		return string(b), err, statErr == nil
	}

	out, err, started := cold(aDesign)
	if err != nil || !started || !strings.Contains(out, "comprehension: PASS") {
		t.Fatalf("the fixture does not start a session on a design (%v, started %v):\n%s", err, started, out)
	}

	out, err, started = cold(skeleton)
	if started {
		t.Fatalf("a session was started on a skeleton:\n%s", out)
	}
	if err == nil {
		t.Fatalf("a skeleton was not refused:\n%s", out)
	}
	for _, want := range []string{"no design to test, and no session was started", "design/DESIGN.md is still a skeleton", "first, draft it with  go -C harness run ./cmd/reverse draft"} {
		if !strings.Contains(out, want) {
			t.Errorf("the refusal does not say %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "comprehension: FAIL") || strings.Contains(out, "VERDICT") {
		t.Errorf("the refusal reads as a verdict:\n%s", out)
	}
}
