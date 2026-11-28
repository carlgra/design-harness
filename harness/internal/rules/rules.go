// Package rules runs the checks a design gives its own rules. A rule is an
// invariant or a decision a person stated; its check is a pattern no line may
// match, a pattern every file must hold, paths no file may be at, or a
// command. What a check finds is
// a breach. The breaches that were in the tree when the rule was adopted are
// kept as a list beside the design, which is allowed and only shrinks; a
// breach that is not on it is new.
//
// The checker reads the tree as it stands; the backtest reads it as it was at
// each past change. Both come through here, so that a rule tried on the
// history is the rule that will be held.
package rules

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"harness/internal/design"
)

// Dir is where the lists of known breaches are kept.
const Dir = "design/rules"

// Rule is one check, ready to run.
type Rule struct {
	design.Rule
	// Doc is the document that states the rule, and Area the area whose
	// spine that is: empty for the design itself.
	Doc  string
	Area string
	// Scope are the globs of the document: everything, for the design, and
	// the area's own for a spine. A rule reads nothing outside them.
	Scope []string

	re   *regexp.Regexp
	from time.Time
}

// Kind is which of the three checks the rule is.
func (r *Rule) Kind() string {
	switch {
	case r.Forbid != "":
		return "forbid"
	case r.Require != "":
		return "require"
	case len(r.Absent) > 0:
		return "absent"
	case r.Run != "":
		return "run"
	}
	return ""
}

// Name is the rule's id as it is cited from outside its document.
func (r *Rule) Name() string {
	if r.Area == "" {
		return r.ID
	}
	return r.Area + "/" + r.ID
}

// KnownPath is the list of the rule's known breaches, relative to the root.
func (r *Rule) KnownPath() string {
	if r.Area == "" {
		return Dir + "/" + r.ID + ".known"
	}
	return Dir + "/" + r.Area + "/" + r.ID + ".known"
}

// Warns reports whether the rule only warns on the day given: it has a day
// to refuse from, and that day has not come.
func (r *Rule) Warns(today time.Time) bool {
	return !r.from.IsZero() && today.Before(r.from)
}

// Reads reports whether the rule reads the file at this path.
func (r *Rule) Reads(f string) bool {
	if r.Scope != nil && !design.Claimed(r.Scope, f) {
		return false
	}
	if len(r.Files) > 0 && !design.Claimed(r.Files, f) {
		return false
	}
	if len(r.Absent) > 0 && !design.Claimed(r.Absent, f) {
		return false
	}
	return !design.Claimed(r.Except, f)
}

var idShape = regexp.MustCompile(`^(ADR|INV)-\d+$`)

// Compile reads every rule the design and its spines give, in their order.
// What is wrong with one is returned as a sentence; such a rule is left out.
func Compile(repo *design.Repo) ([]*Rule, []string) {
	var out []*Rule
	var wrong []string
	add := func(doc, area string, scope []string, list []design.Rule) {
		for i, d := range list {
			r := &Rule{Rule: d, Doc: doc, Area: area, Scope: scope}
			where := fmt.Sprintf("%s rules: entry %d", doc, i+1)
			if d.ID != "" {
				where = fmt.Sprintf("%s rules: %s", doc, d.ID)
			}
			if !idShape.MatchString(d.ID) {
				wrong = append(wrong, fmt.Sprintf("%s: id: %q is not an invariant's or a decision's id", where, d.ID))
				continue
			}
			n := 0
			for _, s := range []string{d.Forbid, d.Require, d.Run, strings.Join(d.Absent, "")} {
				if s != "" {
					n++
				}
			}
			if n != 1 {
				wrong = append(wrong, fmt.Sprintf("%s: a check is exactly one of forbid:, require:, absent: and run:; this has %d", where, n))
				continue
			}
			if pat := d.Forbid + d.Require; pat != "" {
				re, err := regexp.Compile(pat)
				if err != nil {
					wrong = append(wrong, fmt.Sprintf("%s: %v", where, err))
					continue
				}
				r.re = re
			}
			if d.From != "" {
				t, err := time.Parse("2006-01-02", d.From)
				if err != nil {
					wrong = append(wrong, fmt.Sprintf("%s: from: %q is not a date, YYYY-MM-DD", where, d.From))
					continue
				}
				r.from = t
			}
			out = append(out, r)
		}
	}
	add(design.DocPath, "", nil, repo.Design.Rules)
	for _, sp := range repo.Spines {
		if sp.Err == nil {
			add(sp.Path, sp.Name, sp.Files, sp.Rules)
		}
	}
	return out, wrong
}

// Finding is one breach: the file, the line where there is one, and what
// was found there.
type Finding struct {
	File string
	Line int
	Text string
}

// Key is what a breach is known by in the list: the file and what was found,
// and no line number, so that a breach stays the same breach when the lines
// above it change.
func (f Finding) Key() string {
	if f.File == "" {
		return f.Text
	}
	return f.File + ": " + f.Text
}

func (f Finding) String() string {
	switch {
	case f.File == "":
		return f.Text
	case f.Line > 0:
		return fmt.Sprintf("%s:%d: %s", f.File, f.Line, f.Text)
	}
	return f.File + ": " + f.Text
}

// textWidth is how much of a line is kept: enough to tell breaches in one
// file apart, and short enough to read in a list.
const textWidth = 160

func squeeze(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > textWidth {
		s = s[:textWidth]
	}
	return s
}

// binary reports whether content is not text: it holds a NUL near its head.
func binary(content []byte) bool {
	head := content
	if len(head) > 8000 {
		head = head[:8000]
	}
	return bytes.IndexByte(head, 0) >= 0
}

// Scan is what a rule that reads files finds in one file's content. A rule
// of paths no file may be at finds the file itself, whatever it holds.
func (r *Rule) Scan(path string, content []byte) []Finding {
	if len(r.Absent) > 0 {
		return []Finding{{File: path, Text: "is there, where no file may be"}}
	}
	if r.re == nil || binary(content) {
		return nil
	}
	if r.Require != "" {
		if r.re.Match(content) {
			return nil
		}
		return []Finding{{File: path, Text: "holds no line matching " + r.Require}}
	}
	if !r.re.Match(content) {
		return nil
	}
	var out []Finding
	for i, line := range bytes.Split(content, []byte("\n")) {
		if r.re.Match(line) {
			out = append(out, Finding{File: path, Line: i + 1, Text: squeeze(string(line))})
		}
	}
	return out
}

// located is the shape a tool prints a finding in: path:line: or
// path:line:column:, then what it has to say.
var located = regexp.MustCompile(`^([^\s:]+):(\d+):(?:\d+:)?\s*(.*)$`)

// RunAt runs a command rule in dir, a checkout of the repository. Each line
// it prints is a breach, whatever its exit status; a command that fails and
// prints nothing could not be run, which is an error and not a clean tree.
func (r *Rule) RunAt(dir string) ([]Finding, error) {
	cmd := exec.Command("sh", "-c", r.Run)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	var out []Finding
	for _, line := range strings.Split(stdout.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if m := located.FindStringSubmatch(line); m != nil {
			n := 0
			fmt.Sscanf(m[2], "%d", &n)
			out = append(out, Finding{File: m[1], Line: n, Text: squeeze(m[3])})
			continue
		}
		out = append(out, Finding{Text: squeeze(line)})
	}
	if err != nil && len(out) == 0 {
		said := strings.TrimSpace(stderr.String())
		if i := strings.IndexByte(said, '\n'); i >= 0 {
			said = said[:i]
		}
		if said == "" {
			said = err.Error()
		}
		return nil, errors.New("the command failed and printed no breach: " + said)
	}
	return out, nil
}

// Evaluate runs every rule against the tree as it stands. Each file is read
// once, whatever the number of rules that read it. A rule that could not be
// run is in the second map with the reason.
func Evaluate(repo *design.Repo, list []*Rule) (map[*Rule][]Finding, map[*Rule]error) {
	found := map[*Rule][]Finding{}
	failed := map[*Rule]error{}
	var patterns []*Rule
	for _, r := range list {
		found[r] = nil
		if r.Kind() == "run" {
			f, err := r.RunAt(repo.Root)
			if err != nil {
				failed[r] = err
			}
			found[r] = f
			continue
		}
		patterns = append(patterns, r)
	}
	if len(patterns) == 0 {
		return found, failed
	}
	for _, f := range repo.Files {
		var readers []*Rule
		for _, r := range patterns {
			if r.Reads(f) {
				readers = append(readers, r)
			}
		}
		if len(readers) == 0 {
			continue
		}
		content, err := os.ReadFile(filepath.Join(repo.Root, filepath.FromSlash(f)))
		if err != nil {
			continue
		}
		for _, r := range readers {
			found[r] = append(found[r], r.Scan(f, content)...)
		}
	}
	return found, failed
}

// --- the list of known breaches ---

// ReadKnown reads a rule's list: one breach to a line, as Key writes it. The
// second value is whether there is a list at all.
func ReadKnown(root string, r *Rule) ([]string, bool, error) {
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(r.KnownPath())))
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return ParseKnown(string(b)), true, nil
}

// ParseKnown reads the text of a list.
func ParseKnown(text string) []string {
	var keys []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		keys = append(keys, line)
	}
	return keys
}

// WriteKnown writes a rule's list, sorted. An empty list is removed: a rule
// with no known breach has no list.
func WriteKnown(root string, r *Rule, keys []string) error {
	path := filepath.Join(root, filepath.FromSlash(r.KnownPath()))
	if len(keys) == 0 {
		err := os.Remove(path)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	sort.Strings(keys)
	var b strings.Builder
	fmt.Fprintf(&b, "# Breaches of %s that were in the tree when its check was adopted.\n", r.Name())
	b.WriteString("# The checker allows these and refuses any other. This list only shrinks:\n")
	b.WriteString("# a breach that is mended comes off it, with `check -known`.\n")
	for _, k := range keys {
		b.WriteString(k + "\n")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// Split sets what a rule found against its list. Fresh are the breaches not
// on the list; gone are the entries of the list that were not found, which
// are breaches since mended. A breach that appears three times needs three
// entries.
func Split(found []Finding, known []string) (fresh []Finding, gone []string) {
	left := map[string]int{}
	for _, k := range known {
		left[k]++
	}
	for _, f := range found {
		if left[f.Key()] > 0 {
			left[f.Key()]--
			continue
		}
		fresh = append(fresh, f)
	}
	for _, k := range known {
		if left[k] > 0 {
			left[k]--
			gone = append(gone, k)
		}
	}
	return fresh, gone
}

// Keys are the keys of what was found, in order.
func Keys(found []Finding) []string {
	keys := make([]string, len(found))
	for i, f := range found {
		keys[i] = f.Key()
	}
	return keys
}

// Minus is the multiset a without b: what is in a more often than in b.
func Minus(a, b []string) []string {
	left := map[string]int{}
	for _, k := range b {
		left[k]++
	}
	var out []string
	for _, k := range a {
		if left[k] > 0 {
			left[k]--
			continue
		}
		out = append(out, k)
	}
	return out
}
