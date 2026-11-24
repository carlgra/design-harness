package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"harness/internal/design"
)

// Unfinished is the mark the reverse-engineering skeleton and draft leave
// where a person has still to write or confirm something.
const Unfinished = "TODO(elephant)"

// Result is what a run found.
type Result struct {
	Failures []string
	Claimed  int
	Orphans  []string
	Notes    int
}

func (r *Result) fail(format string, args ...any) {
	r.Failures = append(r.Failures, fmt.Sprintf(format, args...))
}

func (r *Result) failed() bool { return len(r.Failures) > 0 }

func (r *Result) report(w io.Writer, repo *design.Repo) {
	for _, f := range r.Failures {
		fmt.Fprintf(w, "FAIL  %s\n", f)
	}
	fmt.Fprintf(w, "harness: %s: %d files accounted for, %d not; %d tests named; %d notes held to their parts\n",
		repo.Name, r.Claimed, len(r.Orphans), len(repo.Design.Tests), r.Notes)
	if n := unfinished(repo.Body); n > 0 {
		fmt.Fprintf(w, "harness: %s still says %s in %d place(s): the design is not finished until a person has answered each\n", design.DocPath, Unfinished, n)
	}
	if r.failed() {
		fmt.Fprintln(w, "harness: FAILED. Two fixes: amend the design, or change the code.")
	} else {
		fmt.Fprintln(w, "harness: OK")
	}
}

var inlineCode = regexp.MustCompile("`[^`\\n]*`")

// unfinished counts the marks left for a person to answer. A mark written as
// code, between backticks, is the mark being spoken of and not one left.
func unfinished(body string) int {
	return strings.Count(inlineCode.ReplaceAllString(body, ""), Unfinished)
}

func runChecks(repo *design.Repo) *Result {
	res := &Result{}
	checkFiles(repo, res)
	checkTests(repo, res)
	checkIDs(repo.Body, res)
	if notes, err := readNotes(repo.Root); err != nil {
		res.fail("%s: %v", ChangesDir, err)
	} else {
		checkNotes(notes, repo.Design.Notes, res)
	}
	return res
}

// checkFiles: every source file is under a glob the design lists. Design
// before code: a file the design does not account for is a file nobody
// designed.
func checkFiles(repo *design.Repo, res *Result) {
	for _, f := range repo.Files {
		if design.Exempt[f] {
			continue
		}
		if design.Claimed(repo.Design.Files, f) {
			res.Claimed++
			continue
		}
		res.Orphans = append(res.Orphans, f)
		res.fail("%s: the design accounts for no such file (design/DESIGN.md files:)", f)
	}
}

// checkTests: every test the design names exists in a test file the design
// accounts for. The design says which paths a fault would travel and what
// walks each; this proves the walker is real.
func checkTests(repo *design.Repo, res *Result) {
	if len(repo.Design.Tests) == 0 {
		return
	}
	var tests []string
	for _, f := range repo.Files {
		if design.IsTestFile(f) && design.Claimed(repo.Design.Files, f) {
			tests = append(tests, f)
		}
	}
	if len(tests) == 0 {
		res.fail("design/DESIGN.md names %d test(s) but accounts for no test file", len(repo.Design.Tests))
		return
	}
	for _, name := range repo.Design.Tests {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if !definedIn(repo.Root, tests, name) {
			res.fail("design/DESIGN.md names the test %s, which is in no test file it accounts for", name)
		}
	}
}

// definedIn reports whether name appears as a whole word in any of the files:
// a Go func, a Python def, or a name a runner is given as a string.
func definedIn(root string, files []string, name string) bool {
	word, err := regexp.Compile(`(^|\W)` + regexp.QuoteMeta(name) + `($|\W)`)
	if err != nil {
		return false
	}
	for _, f := range files {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f)))
		if err != nil {
			continue
		}
		if word.Match(b) {
			return true
		}
	}
	return false
}

// --- ids ---

var (
	// A decision is defined by a heading that opens with its id.
	adrDef = regexp.MustCompile(`^#{1,6}\s+(ADR-\d+)\b`)
	// An invariant is defined by a list item or a heading that opens with
	// its id, bold or not.
	invDef = regexp.MustCompile(`^\s*(?:[-*]|#{1,6})\s+\*{0,2}(INV-\d+)\b`)
	idCite = regexp.MustCompile(`\b(ADR|INV)-\d+\b`)
)

// checkIDs: decisions keep their ids for life so that a citation always
// lands. An id defined twice lands in two places, and one cited that is
// defined nowhere lands in none; both are refused. An id qualified by another
// repository's name, such as other/ADR-0004, is that repository's to define.
// A kind the document defines none of is not checked: a repository may cite
// decisions it has not yet brought in.
func checkIDs(body string, res *Result) {
	defined := map[string]int{}
	type cite struct {
		id   string
		line int
	}
	var cites []cite
	fenced := false
	for i, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			fenced = !fenced
			continue
		}
		if fenced {
			continue
		}
		def := ""
		if m := adrDef.FindStringSubmatch(line); m != nil {
			def = m[1]
		} else if m := invDef.FindStringSubmatch(line); m != nil {
			def = m[1]
		}
		if def != "" {
			defined[def]++
		}
		for _, loc := range idCite.FindAllStringIndex(line, -1) {
			if loc[0] > 0 && line[loc[0]-1] == '/' {
				continue
			}
			cites = append(cites, cite{line[loc[0]:loc[1]], i + 1})
		}
	}
	kinds := map[string]bool{}
	var ids []string
	for id, n := range defined {
		kinds[id[:3]] = true
		if n > 1 {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		res.fail("design/DESIGN.md defines %s %d times: an id names one thing, for life", id, defined[id])
	}
	seen := map[string]bool{}
	for _, c := range cites {
		if defined[c.id] > 0 || !kinds[c.id[:3]] || seen[c.id] {
			continue
		}
		seen[c.id] = true
		res.fail("design/DESIGN.md cites %s (first at body line %d), which it does not define", c.id, c.line)
	}
}
