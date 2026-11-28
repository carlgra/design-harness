package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"harness/internal/design"
	"harness/internal/rules"
)

// RuleLine is what one rule came to in a run: what its check found, how much
// of that was already known, and what the change did to the list.
type RuleLine struct {
	Name  string // the id, qualified by its area
	Doc   string
	Kind  string
	From  string // the day it refuses from, when it only warns today
	Found int
	Known int // entries on the list
	Fresh []rules.Finding
	Gone  []string // on the list and no longer found
	// Moved is the number of listed breaches found under another file: the
	// same breaches, after a rename or a move.
	Moved int
	// Was is the length of the list where this change branched off, and
	// Grown the entries added to it since; Based is whether that could be
	// read at all, and Adopted whether there was no list there: the rule's
	// check, or its list, is new in this change.
	Based   bool
	Adopted bool
	Was     int
	Grown   int
	Err     error
}

// shown is how many breaches of one rule are printed before the rest are
// counted: a rule adopted without a list can find thousands.
const shown = 15

// checkRules runs the checks the design and its spines give their rules. A
// check must belong to an id its own document defines. A breach on the
// rule's list is allowed; one that is not is refused, or, before the rule's
// day, reported. An entry of the list that is no longer found is refused
// until it is taken off, so that the list only shrinks.
func checkRules(repo *design.Repo, today time.Time, res *Result) {
	list, wrong := rules.Compile(repo)
	for _, w := range wrong {
		res.fail("%s", w)
	}
	if len(list) == 0 {
		return
	}
	defs := map[string]map[string]int{design.DocPath: definitions(repo.Body)}
	for _, sp := range repo.Spines {
		if sp.Err == nil {
			defs[sp.Path] = definitions(sp.Body)
		}
	}
	found, failed := rules.Evaluate(repo, list)

	// Several checks may hold one id: they share its list and its day.
	var order []string
	groups := map[string][]*rules.Rule{}
	for _, r := range list {
		if defs[r.Doc][r.ID] == 0 {
			res.fail("%s rules: %s is checked and not defined there: a check belongs to a rule the document states", r.Doc, r.ID)
			continue
		}
		if _, ok := groups[r.Name()]; !ok {
			order = append(order, r.Name())
		}
		groups[r.Name()] = append(groups[r.Name()], r)
	}
	base := baseOf(repo.Root)
	for _, name := range order {
		g := groups[name]
		first := g[0]
		line := RuleLine{Name: name, Doc: first.Doc, Kind: first.Kind()}
		var all []rules.Finding
		for _, r := range g {
			if r.From != first.From {
				res.fail("%s rules: the checks of %s name different days (from:): a rule comes into force once", first.Doc, first.ID)
			}
			if err := failed[r]; err != nil {
				line.Err = err
				res.fail("%s: %s: %v", name, r.Run, err)
			}
			all = append(all, found[r]...)
		}
		known, _, err := rules.ReadKnown(repo.Root, first)
		if err != nil {
			res.fail("%s: %v", first.KnownPath(), err)
		}
		line.Found, line.Known = len(all), len(known)
		fresh, gone := rules.Split(all, known)
		var moved []string
		moved, line.Fresh, line.Gone = rules.Moved(fresh, gone)
		line.Moved = len(moved)
		if was, there, ok := knownAt(repo.Root, base, first.KnownPath()); ok {
			line.Based, line.Was, line.Adopted = true, len(was), !there
			if there {
				line.Grown = len(rules.Minus(known, was))
			}
		}
		warns := first.Warns(today)
		if warns {
			line.From = first.From
		}
		if line.Err == nil {
			say := res.fail
			if warns {
				say = res.warn
			}
			for i, f := range line.Fresh {
				if i == shown {
					say("%s: and %d more", name, len(line.Fresh)-shown)
					break
				}
				say("%s: %s", name, clip(f.String(), 180))
			}
			if n := len(line.Fresh); n > 0 && !warns && len(known) == 0 {
				res.fail("%s: %d breach(es) and no list of known ones. To adopt the rule with these allowed and every new one refused: go -C harness run ./cmd/check -known %s", name, n, name)
			}
			if line.Moved > 0 {
				res.fail("%s: %d known breach(es) are now in another file. They are the same breaches: follow them with go -C harness run ./cmd/check -known", name, line.Moved)
			}
			if n := len(line.Gone); n > 0 {
				res.fail("%s: %d breach(es) on %s are mended. Take them off, so that they cannot come back: go -C harness run ./cmd/check -known", name, n, first.KnownPath())
			}
		}
		if line.Grown > 0 {
			res.warn("%s: %d breach(es) were added to %s in this change. A list only shrinks: whoever merges this approves each one", name, line.Grown, first.KnownPath())
		}
		res.Rules = append(res.Rules, line)
	}
}

// recordKnown brings each rule's list in line with the tree. A rule with no
// list is given one holding every breach its check finds: that is the rule
// adopted, with what was already there allowed. A rule with a list loses the
// entries that are mended and gains nothing: a new breach is mended, not
// listed. names picks rules; none is all of them.
func recordKnown(repo *design.Repo, names []string, w io.Writer) error {
	list, wrong := rules.Compile(repo)
	if len(wrong) > 0 {
		return fmt.Errorf("%s", strings.Join(wrong, "\n"))
	}
	want := map[string]bool{}
	for _, n := range names {
		want[n] = true
	}
	found, failed := rules.Evaluate(repo, list)
	var order []string
	groups := map[string][]*rules.Rule{}
	for _, r := range list {
		if len(want) > 0 && !want[r.Name()] {
			continue
		}
		if err := failed[r]; err != nil {
			return fmt.Errorf("%s: %s: %v", r.Name(), r.Run, err)
		}
		if _, ok := groups[r.Name()]; !ok {
			order = append(order, r.Name())
		}
		groups[r.Name()] = append(groups[r.Name()], r)
		delete(want, r.Name())
	}
	for n := range want {
		return fmt.Errorf("%s: no rule of that name has a check (rules: in the design or a spine)", n)
	}
	for _, name := range order {
		first := groups[name][0]
		var all []rules.Finding
		for _, r := range groups[name] {
			all = append(all, found[r]...)
		}
		known, exists, err := rules.ReadKnown(repo.Root, first)
		if err != nil {
			return err
		}
		if !exists {
			if err := rules.WriteKnown(repo.Root, first, rules.Keys(all)); err != nil {
				return err
			}
			if len(all) == 0 {
				fmt.Fprintf(w, "harness: %s: no breach in the tree, and so no list\n", name)
			} else {
				fmt.Fprintf(w, "harness: %s: %d breach(es) recorded in %s\n", name, len(all), first.KnownPath())
			}
			continue
		}
		fresh, gone := rules.Split(all, known)
		moved, fresh, mended := rules.Moved(fresh, gone)
		if len(gone) > 0 {
			if err := rules.WriteKnown(repo.Root, first, append(rules.Minus(known, gone), moved...)); err != nil {
				return err
			}
		}
		if len(moved) > 0 {
			fmt.Fprintf(w, "harness: %s: %d breach(es) followed to the file they moved to\n", name, len(moved))
		}
		fmt.Fprintf(w, "harness: %s: %d mended breach(es) taken off %s, %d left\n", name, len(mended), first.KnownPath(), len(known)-len(mended))
		if len(fresh) > 0 {
			fmt.Fprintf(w, "harness: %s: %d new breach(es) are not added: mend them. To adopt the rule afresh, delete the list and run this again\n", name, len(fresh))
		}
	}
	return nil
}

// baseOf is the commit this change branched from: the merge base of HEAD and
// the default branch as the remote last gave it. HARNESS_BASE names another
// ref. Empty where there is none to find, as in a clone with no history.
func baseOf(root string) string {
	git := func(args ...string) string {
		out, err := exec.Command("git", append([]string{"-C", root}, args...)...).Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(out))
	}
	refs := []string{os.Getenv("HARNESS_BASE"), git("symbolic-ref", "-q", "refs/remotes/origin/HEAD"), "refs/remotes/origin/main", "refs/remotes/origin/master"}
	for _, ref := range refs {
		if ref == "" {
			continue
		}
		if base := git("merge-base", "HEAD", ref); base != "" {
			return base
		}
	}
	return ""
}

// knownAt is a list as it was at a commit, and whether there was one. No
// commit is no answer.
func knownAt(root, commit, path string) (keys []string, there, ok bool) {
	if commit == "" {
		return nil, false, false
	}
	out, err := exec.Command("git", "-C", root, "show", commit+":"+path).Output()
	if err != nil {
		return nil, false, true
	}
	return rules.ParseKnown(string(out)), true, true
}

// Size is how much one document weighs.
type Size struct {
	Path  string
	Lines int
	Bytes int
}

// Tokens is a rough count of what the document costs a session to read:
// four bytes to a token, to the nearest hundred.
func (s Size) Tokens() int { return (s.Bytes/4 + 50) / 100 * 100 }

// sizes weighs the design and every spine.
func sizes(repo *design.Repo) []Size {
	weigh := func(path string) (Size, bool) {
		b, err := os.ReadFile(filepath.Join(repo.Root, filepath.FromSlash(path)))
		if err != nil {
			return Size{}, false
		}
		return Size{Path: path, Lines: strings.Count(string(b), "\n"), Bytes: len(b)}, true
	}
	var out []Size
	if s, ok := weigh(design.DocPath); ok {
		out = append(out, s)
	}
	for _, sp := range repo.Spines {
		if s, ok := weigh(sp.Path); ok {
			out = append(out, s)
		}
	}
	return out
}

// checkSizes says of each document what it weighs, and warns of one over the
// size the design sets. It refuses nothing: a document is cut down by a
// person, into a spine or out of the design, not by a failed push.
func checkSizes(repo *design.Repo, res *Result) {
	limit := repo.Design.Size
	if limit <= 0 {
		limit = design.DefaultSize
	}
	res.Limit = limit
	res.Sizes = sizes(repo)
	for _, s := range res.Sizes {
		if s.Lines > limit {
			res.warn("%s is %d lines, over the %d a document is to stay under (size:). What binds one area belongs in its spine; what binds nothing belongs in neither", s.Path, s.Lines, limit)
		}
	}
}

// checkReplaced names every document the design, or a spine, replaces that
// is still there. Two copies of a rule are two standards.
func checkReplaced(repo *design.Repo, res *Result) {
	seen := map[string]bool{}
	name := func(doc string, paths []string) {
		for _, p := range paths {
			p = strings.Trim(filepath.ToSlash(p), "/")
			if p == "" || seen[p] {
				continue
			}
			seen[p] = true
			if _, err := os.Lstat(filepath.Join(repo.Root, filepath.FromSlash(p))); err == nil {
				res.warn("%s is still there, and %s replaces it (replaces:). Retire it, or cut it down to what the design does not hold, and take it off the list", p, doc)
				res.Replaced = append(res.Replaced, p)
			}
		}
	}
	name(design.DocPath, repo.Design.Replaces)
	for _, sp := range repo.Spines {
		name(sp.Path, sp.Replaces)
	}
}

// reportRules prints one line to a rule.
func (r *Result) reportRules(w io.Writer) {
	for _, l := range r.Rules {
		state := "in force"
		if l.From != "" {
			state = "warns until " + l.From
		}
		fmt.Fprintf(w, "harness: rule %s (%s, %s): %d new, %d known", l.Name, l.Kind, state, len(l.Fresh), l.Known)
		switch {
		case l.Based && l.Adopted && l.Known > 0:
			fmt.Fprint(w, ", listed in this change")
		case l.Based && l.Was != l.Known:
			fmt.Fprintf(w, " (%d where this change began)", l.Was)
		}
		fmt.Fprintln(w)
	}
}

func thousands(n int) string {
	s := fmt.Sprint(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

// Markdown is the run as a report a pull request can carry: each rule with
// what it found, what the change mended and what it added, then what the
// checker refused and what it only said.
func (r *Result) Markdown(repo *design.Repo) string {
	var b strings.Builder
	verdict := "passes"
	if r.failed() {
		verdict = "is refused"
	}
	fmt.Fprintf(&b, "## The design's checker: %s %s\n\n", repo.Name, verdict)
	fmt.Fprintf(&b, "%d files accounted for, %d not; %d notes held to their parts.\n\n", r.Claimed, len(r.Orphans), r.Notes)
	if len(r.Rules) > 0 {
		b.WriteString("| Rule | Check | State | New | Known | Mended here | Added to the list here |\n|---|---|---|---|---|---|---|\n")
		for _, l := range r.Rules {
			state := "in force"
			if l.From != "" {
				state = "warns until " + l.From
			}
			mended, added := "", ""
			switch {
			case l.Based && l.Adopted && l.Known > 0:
				added = "the list is new here"
			case l.Based:
				mended, added = fmt.Sprint(l.Was+l.Grown-l.Known), fmt.Sprint(l.Grown)
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %d | %d | %s | %s |\n", l.Name, l.Kind, state, len(l.Fresh), l.Known, mended, added)
		}
		b.WriteString("\n")
	}
	if len(r.Sizes) > 0 {
		b.WriteString("| Document | Lines | Tokens, about |\n|---|---|---|\n")
		for _, s := range r.Sizes {
			over := ""
			if s.Lines > r.Limit {
				over = fmt.Sprintf(" (over %d)", r.Limit)
			}
			fmt.Fprintf(&b, "| %s | %d%s | %s |\n", s.Path, s.Lines, over, thousands(s.Tokens()))
		}
		b.WriteString("\n")
	}
	section := func(title string, lines []string) {
		if len(lines) == 0 {
			return
		}
		fmt.Fprintf(&b, "### %s\n\n", title)
		for _, l := range lines {
			fmt.Fprintf(&b, "- %s\n", strings.ReplaceAll(l, "|", "\\|"))
		}
		b.WriteString("\n")
	}
	section("Refused", r.Failures)
	section("Said, and not refused", r.Warnings)
	return b.String()
}

// writeReport writes the report to path, and appends it where a GitHub
// workflow shows a job's summary.
func writeReport(md, path string) error {
	if path != "" {
		if err := os.WriteFile(path, []byte(md), 0o644); err != nil {
			return err
		}
	}
	if sum := os.Getenv("GITHUB_STEP_SUMMARY"); sum != "" {
		f, err := os.OpenFile(sum, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.WriteString(md)
		return err
	}
	return nil
}

func clip(s string, n int) string {
	if len(s) > n {
		return s[:n-1] + "…"
	}
	return s
}

// reportSizes says what there is to read: the design, and where there are
// spines, the largest, since a change reads the design and one of them.
func (r *Result) reportSizes(w io.Writer) {
	if len(r.Sizes) == 0 {
		return
	}
	d := r.Sizes[0]
	fmt.Fprintf(w, "harness: %s is %d lines, about %s tokens", d.Path, d.Lines, thousands(d.Tokens()))
	if len(r.Sizes) > 1 {
		big := r.Sizes[1]
		for _, s := range r.Sizes[2:] {
			if s.Lines > big.Lines {
				big = s
			}
		}
		fmt.Fprintf(w, "; the largest spine, %s, is %d lines, about %s", big.Path, big.Lines, thousands(big.Tokens()))
	}
	fmt.Fprintln(w)
}
