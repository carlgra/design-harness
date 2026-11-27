package main

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"harness/internal/design"
)

const whole = "# A change\n\n## The problem\n\nIt breaks.\n\n## The plan\n\nMend it.\n\n## Alternatives rejected\n\nNone.\n\n## Files\n\n- `app/main.go`: the mend.\n"

// A design note carries its four parts, each with something under it: a
// missing part and an empty one are refused, "None." is an answer, a heading
// in a code fence is not a heading, and a note dated before the rules began
// is never reread.
func TestADesignNoteCarriesItsParts(t *testing.T) {
	rules := design.Notes{From: "2026-10-24"}
	cases := []struct {
		name, file, body string
		refused          []string
	}{
		{"whole", "design/changes/2026-10-24-a.md", whole, nil},
		{"no plan", "design/changes/2026-10-24-b.md", strings.Replace(whole, "## The plan\n\nMend it.\n\n", "", 1), []string{`no section "Plan"`}},
		{"an empty part", "design/changes/2026-10-25-c.md", strings.Replace(whole, "None.", "<!-- what else was considered -->", 1), []string{`the section "Alternatives rejected" is empty`}},
		{"undated", "design/changes/plan.md", "# C\n", []string{"Problem", "Plan", "Alternatives rejected", "Files"}},
		{"written before the rules", "design/changes/2026-10-23-d.md", "# D\n\nNo parts at all.\n", nil},
		{"no article, deeper, and more in the title", "design/changes/2026-11-01-e.md", "# E\n\n### Problem\n\nx\n\n### Plan, in two steps\n\nx\n\n### Alternatives rejected\n\nx\n\n### Files to change and why\n\nx\n", nil},
		{"in a fence", "design/changes/2026-10-24-f.md", strings.Replace(whole, "## Files\n\n- `app/main.go`: the mend.\n", "```\n## Files\n\n- x\n```\n", 1), []string{`no section "Files"`}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := &Result{}
			checkNotes([]Note{NewNote(c.file, c.body)}, rules, res)
			if len(res.Failures) != len(c.refused) {
				t.Fatalf("expected %d refusal(s), got %v", len(c.refused), res.Failures)
			}
			for i, want := range c.refused {
				if !strings.Contains(res.Failures[i], want) || !strings.Contains(res.Failures[i], c.file) {
					t.Fatalf("expected %q for %s, got %q", want, c.file, res.Failures[i])
				}
			}
		})
	}
}

// A repository asks more of a note in its own design: a section it adds is
// required from the date it says, read from the frontmatter as written.
func TestADesignSaysWhatItsNotesCarry(t *testing.T) {
	var d design.Design
	frontmatter := "notes:\n  from: 2026-11-24\n  sections: [Problem, Plan, Alternatives rejected, Files, What existing data changes meaning]\n"
	if err := yaml.Unmarshal([]byte(frontmatter), &d); err != nil {
		t.Fatal(err)
	}
	if d.Notes.From != "2026-11-24" || len(d.Notes.Sections) != 5 {
		t.Fatalf("the frontmatter was read as %+v", d.Notes)
	}
	res := &Result{}
	checkNotes([]Note{
		NewNote("design/changes/2026-11-23-old.md", whole),
		NewNote("design/changes/2026-11-24-new.md", whole),
		NewNote("design/changes/2026-11-25-says.md", whole+"\n## What existing data changes meaning\n\nNothing.\n"),
	}, d.Notes, res)
	if res.Notes != 2 || len(res.Failures) != 1 || !strings.Contains(res.Failures[0], "2026-11-24-new.md") || !strings.Contains(res.Failures[0], "What existing data changes meaning") {
		t.Fatalf("held %d notes, refused %v", res.Notes, res.Failures)
	}
}
