package main

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"harness/internal/design"
)

// ChangesDir is where a repository's design notes live.
const ChangesDir = "design/changes"

// Note is one design note: its path relative to the repository, the date its
// name begins with ("" when it has none), and its text.
type Note struct {
	Path string
	Date string
	Body string
}

var datePrefix = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})-`)

// NewNote names a note from its path and text.
func NewNote(p, body string) Note {
	n := Note{Path: p, Body: strings.ReplaceAll(body, "\r\n", "\n")}
	if m := datePrefix.FindStringSubmatch(path.Base(p)); m != nil {
		n.Date = m[1]
	}
	return n
}

// Bound reports whether the note is held to the rules, which bind from a
// date: a note written before a rule existed is never reread. A note whose
// name carries no date is held to them: it cannot show it is old.
func (n Note) Bound(from string) bool { return n.Date == "" || n.Date >= from }

// readNotes reads every .md directly under design/changes in a working tree.
func readNotes(root string) ([]Note, error) {
	dir := filepath.Join(root, filepath.FromSlash(ChangesDir))
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var notes []Note
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		notes = append(notes, NewNote(ChangesDir+"/"+e.Name(), string(b)))
	}
	sort.Slice(notes, func(i, j int) bool { return notes[i].Path < notes[j].Path })
	return notes, nil
}

var heading = regexp.MustCompile(`^(#{1,6})\s+(.*?)\s*#*\s*$`)

// plain is a heading's title as it is compared: lower case, without a
// leading article, so that "The problem" is the section "Problem".
func plain(title string) string {
	t := strings.ToLower(strings.TrimSpace(title))
	for _, article := range []string{"the ", "a ", "an "} {
		t = strings.TrimPrefix(t, article)
	}
	return t
}

// Section returns the text under the first heading whose title begins with
// title, case and a leading article aside, up to the next heading of the
// same or a higher level. A heading inside a code fence is not a heading.
func Section(body, title string) (string, bool) {
	lines := strings.Split(body, "\n")
	fenced := false
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			fenced = !fenced
			continue
		}
		m := heading.FindStringSubmatch(l)
		if fenced || m == nil || !strings.HasPrefix(plain(m[2]), plain(title)) {
			continue
		}
		level := len(m[1])
		var out []string
		inner := false
		for _, l2 := range lines[i+1:] {
			if strings.HasPrefix(strings.TrimSpace(l2), "```") {
				inner = !inner
			}
			if m2 := heading.FindStringSubmatch(l2); !inner && m2 != nil && len(m2[1]) <= level {
				break
			}
			out = append(out, l2)
		}
		return strings.Join(out, "\n"), true
	}
	return "", false
}

var htmlComment = regexp.MustCompile(`(?s)<!--.*?-->`)

// checkNotes: every note the rules bind carries each section the design asks
// of a note, with something under it. "None." is an answer; an empty section,
// or one holding only the template's comment, is not.
func checkNotes(notes []Note, rules design.Notes, res *Result) {
	sections := rules.Sections
	if len(sections) == 0 {
		sections = design.DefaultSections
	}
	for _, n := range notes {
		if !n.Bound(rules.From) {
			continue
		}
		res.Notes++
		for _, title := range sections {
			text, ok := Section(n.Body, title)
			if !ok {
				res.fail("%s: no section %q: a design note says it, or says there is none", n.Path, title)
				continue
			}
			if strings.TrimSpace(htmlComment.ReplaceAllString(text, "")) == "" {
				res.fail("%s: the section %q is empty: \"None.\" is an answer, silence is not", n.Path, title)
			}
		}
	}
}
