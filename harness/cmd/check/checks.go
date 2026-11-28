package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"harness/internal/design"
)

// Unfinished is the mark the reverse-engineering skeleton and draft leave
// where a person has still to write or confirm something.
const Unfinished = design.Unfinished

// Result is what a run found.
type Result struct {
	Failures []string
	Claimed  int
	InSpines int // of Claimed, those an area holds
	Orphans  []string
	Notes    int
	// Warnings are what the run says and does not refuse: a rule's breach
	// before its day, a document over its size, a replaced document still
	// there.
	Warnings []string
	Rules    []RuleLine
	Sizes    []Size
	Limit    int
	Replaced []string
}

func (r *Result) warn(format string, args ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
}

func (r *Result) fail(format string, args ...any) {
	r.Failures = append(r.Failures, fmt.Sprintf(format, args...))
}

func (r *Result) failed() bool { return len(r.Failures) > 0 }

func (r *Result) report(w io.Writer, repo *design.Repo) {
	for _, f := range r.Failures {
		fmt.Fprintf(w, "FAIL  %s\n", f)
	}
	for _, f := range r.Warnings {
		fmt.Fprintf(w, "WARN  %s\n", f)
	}
	tests := len(repo.Design.Tests)
	for _, sp := range repo.Spines {
		tests += len(sp.Tests)
	}
	fmt.Fprintf(w, "harness: %s: %d files accounted for, %d not; %d tests named; %d notes held to their parts\n",
		repo.Name, r.Claimed, len(r.Orphans), tests, r.Notes)
	if len(repo.Spines) > 0 {
		fmt.Fprintf(w, "harness: %d spine(s), holding %d of those files; the rest are the design's own\n", len(repo.Spines), r.InSpines)
	}
	r.reportRules(w)
	r.reportSizes(w)
	if n := unfinished(repo.Body); n > 0 {
		fmt.Fprintf(w, "harness: %s still says %s in %d place(s): the design is not finished until a person has answered each\n", design.DocPath, Unfinished, n)
	}
	for _, sp := range repo.Spines {
		if n := unfinished(sp.Body); n > 0 {
			fmt.Fprintf(w, "harness: %s still says %s in %d place(s)\n", sp.Path, Unfinished, n)
		}
	}
	if r.failed() {
		fmt.Fprintln(w, "harness: FAILED. Two fixes: amend the design, or change the code.")
	} else {
		fmt.Fprintln(w, "harness: OK")
	}
}

// unfinished counts the marks left for a person to answer.
func unfinished(body string) int { return design.Marks(body) }

func runChecks(repo *design.Repo) *Result { return runChecksOn(repo, time.Now()) }

// runChecksOn runs the checks as of a day: a rule with a day to refuse from
// warns before it.
func runChecksOn(repo *design.Repo, today time.Time) *Result {
	res := &Result{}
	parent, err := design.FindParent(repo.Root)
	if err != nil {
		res.fail("the root this repository is nested under: %v", err)
	}
	checkSpines(repo, parent, res)
	checkFiles(repo, res)
	checkTests(repo, res)
	checkMap(repo, res)
	checkIDs(repo, parent, res)
	checkContracts(repo, LocateEnv(parent), res)
	checkRules(repo, today, res)
	checkSizes(repo, res)
	checkReplaced(repo, res)
	if notes, err := readNotes(repo.Root); err != nil {
		res.fail("%s: %v", ChangesDir, err)
	} else {
		checkNotes(notes, repo.Design.Notes, res)
	}
	return res
}

// areaName is what an area may be called: it names a file, and qualifies an
// id.
var areaName = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

// checkSpines: the areas the design defines and the spines on disk are the
// same set. Every area has a name that can qualify an id, globs, and a spine
// that reads; every document under design/spines/ is the spine of an area.
// An area called what the repository or its root is called is refused, since
// a citation qualified by that name could then mean either.
func checkSpines(repo *design.Repo, parent *design.Parent, res *Result) {
	seen := map[string]bool{}
	for _, a := range repo.Design.Spines {
		switch {
		case !areaName.MatchString(a.Name):
			res.fail("design/DESIGN.md spines: %q is not a name an area can have: lower-case letters, digits, '.', '_' and '-'", a.Name)
			continue
		case seen[a.Name]:
			res.fail("design/DESIGN.md spines: the area %s is defined twice", a.Name)
			continue
		case a.Name == repo.Name || (parent != nil && a.Name == parent.Name):
			res.fail("design/DESIGN.md spines: the area %s has a repository's name, so %s/ADR-0001 could mean either", a.Name, a.Name)
		}
		seen[a.Name] = true
		if len(a.Files) == 0 {
			res.fail("design/DESIGN.md spines: the area %s lists no files:, so its spine is the design of nothing", a.Name)
		}
	}
	for _, sp := range repo.Spines {
		if sp.Err != nil && areaName.MatchString(sp.Name) {
			res.fail("%s: the spine of the area %s: %v", sp.Path, sp.Name, sp.Err)
		}
	}
	for _, p := range repo.StraySpines {
		res.fail("%s: a spine for an area the design does not define (design/DESIGN.md spines:)", p)
	}
}

// checkFiles: every source file is under a glob the design lists, its own or
// an area's. Design before code: a file the design does not account for is a
// file nobody designed. A file two areas hold has two sets of rules and
// nobody to say which wins, and is refused.
func checkFiles(repo *design.Repo, res *Result) {
	for _, f := range repo.Files {
		if design.Exempt[f] {
			continue
		}
		areas := repo.AreasOf(f)
		if len(areas) > 1 {
			res.fail("%s: held by %d areas, %s: a file is under one spine (design/DESIGN.md spines:)", f, len(areas), strings.Join(areas, " and "))
		}
		if len(areas) > 0 {
			res.Claimed++
			res.InSpines++
			continue
		}
		if design.Claimed(repo.Design.Files, f) {
			res.Claimed++
			continue
		}
		res.Orphans = append(res.Orphans, f)
		res.fail("%s: the design accounts for no such file (design/DESIGN.md files:, or an area under spines:)", f)
	}
}

// checkTests: every test a document names exists in a test file it accounts
// for: the design's anywhere in the repository, a spine's within its area.
// The design says which paths a fault would travel and what walks each; this
// proves the walker is real.
func checkTests(repo *design.Repo, res *Result) {
	checkNamed(repo, "design/DESIGN.md", repo.Design.Tests, func(f string) bool {
		return design.Claimed(repo.Design.Files, f) || len(repo.AreasOf(f)) > 0
	}, res)
	for _, sp := range repo.Spines {
		globs := sp.Files
		checkNamed(repo, sp.Path, sp.Tests, func(f string) bool { return design.Claimed(globs, f) }, res)
	}
}

// testFiles are the test files the design accounts for, its own and every
// area's.
func testFiles(repo *design.Repo) []string {
	var tests []string
	for _, f := range repo.Files {
		if design.IsTestFile(f) && (design.Claimed(repo.Design.Files, f) || len(repo.AreasOf(f)) > 0) {
			tests = append(tests, f)
		}
	}
	return tests
}

func checkNamed(repo *design.Repo, doc string, names []string, mine func(string) bool, res *Result) {
	if len(names) == 0 {
		return
	}
	var tests []string
	for _, f := range repo.Files {
		if design.IsTestFile(f) && mine(f) {
			tests = append(tests, f)
		}
	}
	if len(tests) == 0 {
		res.fail("%s names %d test(s) but accounts for no test file", doc, len(names))
		return
	}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if !definedIn(repo.Root, tests, name) {
			res.fail("%s names the test %s, which is in no test file it accounts for", doc, name)
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

// checkMap: a root's map is true. Where the design lists the repositories
// nested under it, every repository found on disk under it is one it lists.
// One listed and not on disk is not refused: sync clones it, and the root is
// checked on its own wherever it is checked out alone.
func checkMap(repo *design.Repo, res *Result) {
	if len(repo.Design.Repos) == 0 {
		return
	}
	listed := map[string]bool{}
	for _, e := range repo.Design.Repos {
		listed[strings.Trim(e.Path, "/")] = true
	}
	for _, n := range repo.NestedRepos {
		if !listed[n] {
			res.fail("%s: a repository on disk that the map does not list (design/DESIGN.md repos:)", n)
		}
	}
}

// definitions are the ids a design defines, each with how many times.
func definitions(body string) map[string]int {
	defined := map[string]int{}
	fenced := false
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			fenced = !fenced
			continue
		}
		if fenced {
			continue
		}
		if m := adrDef.FindStringSubmatch(line); m != nil {
			defined[m[1]]++
		} else if m := invDef.FindStringSubmatch(line); m != nil {
			defined[m[1]]++
		}
	}
	return defined
}

var qualifier = regexp.MustCompile(`[A-Za-z0-9._-]+$`)

// checkIDs: decisions keep their ids for life so that a citation always
// lands. Each document, the design and every spine, numbers its own: within
// it, an id defined twice lands in two places and one cited that is defined
// nowhere lands in none, and both are refused. An id another document
// defines is cited by that document's name: an area's as checkout/ADR-0003,
// the repository's own design, from a spine, as <repository>/ADR-0004, and
// the root's, from a repository nested under it, as <root>/ADR-0004. Each of
// those is looked up and refused if it is not there. An id qualified by any
// other name is another repository's to define and is not looked up. A kind
// a document defines none of is not checked there: it may cite decisions it
// has not yet brought in.
func checkIDs(repo *design.Repo, parent *design.Parent, res *Result) {
	named := map[string]map[string]int{}
	if parent != nil {
		named[parent.Name] = definitions(parent.Body)
	}
	named[repo.Name] = definitions(repo.Body)
	for _, sp := range repo.Spines {
		if sp.Err == nil {
			named[sp.Name] = definitions(sp.Body)
		}
	}
	checkDocIDs(design.DocPath, repo.Body, named, res)
	for _, sp := range repo.Spines {
		if sp.Err == nil {
			checkDocIDs(sp.Path, sp.Body, named, res)
		}
	}
}

// checkDocIDs holds one document's ids and citations. named are the
// documents a citation here can be qualified by, each with what it defines.
func checkDocIDs(doc, body string, named map[string]map[string]int, res *Result) {
	defined := definitions(body)
	type cite struct {
		id   string
		line int
	}
	var cites []cite
	seen := map[string]bool{}
	fenced := false
	for i, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			fenced = !fenced
			continue
		}
		if fenced {
			continue
		}
		for _, loc := range idCite.FindAllStringIndex(line, -1) {
			id := line[loc[0]:loc[1]]
			if loc[0] > 0 && line[loc[0]-1] == '/' {
				before := line[:loc[0]-1]
				q := qualifier.FindString(before)
				// a/b/ADR-0001 is b in the repository a: not this one's.
				if strings.HasSuffix(strings.TrimSuffix(before, q), "/") {
					continue
				}
				if defs, ok := named[q]; ok && defs[id] == 0 && !seen[q+"/"+id] {
					seen[q+"/"+id] = true
					res.fail("%s cites %s/%s (first at body line %d), which %s's design does not define", doc, q, id, i+1, q)
				}
				continue
			}
			cites = append(cites, cite{id, i + 1})
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
		res.fail("%s defines %s %d times: an id names one thing, for life", doc, id, defined[id])
	}
	for _, c := range cites {
		if defined[c.id] > 0 || !kinds[c.id[:3]] || seen[c.id] {
			continue
		}
		seen[c.id] = true
		res.fail("%s cites %s (first at body line %d), which it does not define", doc, c.id, c.line)
	}
}
