package main

import (
	"fmt"
	"strings"

	"harness/internal/history"
)

// Report is what a backtest found.
type Report struct {
	Repo    string
	Commits []history.Commit
	Areas   *AreaScore
	Rules   []RuleTrial
	// Trial is a rule given on the command line; Detail lists every change a
	// rule would have refused, and not the most recent few.
	// Asked is how many changes were asked for.
	Asked  int
	Trial  bool
	Detail bool
}

// few is how many of anything are listed where not everything is.
const few = 5

func pct(n, of int) string {
	if of == 0 {
		return "0%"
	}
	p := (n*100 + of/2) / of
	switch {
	case p == 0 && n > 0:
		return "under 1%"
	case p == 100 && n < of:
		return "99%"
	}
	return fmt.Sprintf("%d%%", p)
}

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

func (r Report) span() string {
	newest, oldest := r.Commits[0], r.Commits[len(r.Commits)-1]
	return fmt.Sprintf("%s, %s to %s", plural(len(r.Commits), "change", "changes"), oldest.Date, newest.Date)
}

// Text is the report as the terminal shows it.
func (r Report) Text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "backtest: %s: %s\n", r.Repo, r.span())
	if r.Asked > len(r.Commits) {
		fmt.Fprintf(&b, "backtest: %d were asked for, and that is all the history there is\n", r.Asked)
	}
	if a := r.Areas; a != nil {
		b.WriteString("\n")
		b.WriteString(a.text())
	}
	for _, t := range r.Rules {
		b.WriteString("\n")
		b.WriteString(t.text(len(r.Commits), r.Detail, r.Trial))
	}
	if r.Areas == nil && len(r.Rules) == 0 {
		b.WriteString("\nNothing to try: the design gives no rule a check (rules:).\n")
	}
	return b.String()
}

func (a *AreaScore) text() string {
	var b strings.Builder
	if a.Code == 0 {
		return "areas: none of these changes touched a file the design accounts for\n"
	}
	if a.Defined == 0 {
		b.WriteString("areas: the design defines none (spines:). Directories that changed together,\n")
		b.WriteString("which is the evidence of where one area ends and the next begins:\n")
		n := 0
		for _, p := range a.Suggest {
			if p.Together < 3 {
				break
			}
			fmt.Fprintf(&b, "  %-28s %-28s %3d changes together, %s of the rarer one's\n", p.A, p.B, p.Together, pct(p.Together, min(p.OfA, p.OfB)))
			if n++; n == 12 {
				break
			}
		}
		if n == 0 {
			b.WriteString("  no two directories changed together three times\n")
		}
		return b.String()
	}
	inside := a.By[0] + a.By[1]
	fmt.Fprintf(&b, "areas: %s of %s to code stayed inside one area or none (%d of %d)\n", pct(inside, a.Code), plural(a.Code, "change", "changes"), inside, a.Code)
	fmt.Fprintf(&b, "  one area            %4d  %4s", a.By[1], pct(a.By[1], a.Code))
	if a.Mixed > 0 {
		fmt.Fprintf(&b, "   (%d of them with files no area holds)", a.Mixed)
	}
	b.WriteString("\n")
	fmt.Fprintf(&b, "  no area             %4d  %4s   (the design's own files only)\n", a.By[0], pct(a.By[0], a.Code))
	fmt.Fprintf(&b, "  two areas           %4d  %4s\n", a.By[2], pct(a.By[2], a.Code))
	fmt.Fprintf(&b, "  three or more       %4d  %4s\n", a.By[3], pct(a.By[3], a.Code))
	b.WriteString("  each area: changes that touched it, and how many of those touched no other\n")
	for _, l := range a.Areas {
		note := ""
		switch {
		case l.Changes == 0:
			note = "   never changed here: is it an area yet?"
		case l.Changes >= 5 && l.Alone*2 < l.Changes:
			note = "   more often changed with another than alone"
		}
		fmt.Fprintf(&b, "    %-24s %4d  %4d alone (%s)%s\n", l.Name, l.Changes, l.Alone, pct(l.Alone, l.Changes), note)
	}
	if len(a.Pairs) > 0 {
		b.WriteString("  areas that changed together most\n")
		for i, p := range a.Pairs {
			if i == few {
				break
			}
			note := ""
			if p.Together >= 5 && p.Share() >= 0.5 {
				note = "   one area in practice, or a rule both keep that belongs in the design"
			}
			fmt.Fprintf(&b, "    %-24s %-24s %4d  (%s of the rarer one's changes)%s\n", p.A, p.B, p.Together, pct(p.Together, min(p.OfA, p.OfB)), note)
			for _, c := range a.Examples[[2]string{p.A, p.B}] {
				fmt.Fprintf(&b, "        %s %s %s\n", c.Date, c.Short(), clip(c.Subject, 72))
			}
		}
	}
	if len(a.Loose) > 0 {
		b.WriteString("  directories no area holds that changed most\n")
		for i, l := range a.Loose {
			if i == few {
				break
			}
			with := ""
			if len(l.With) > 0 && l.With[0].N*2 >= l.Changes && l.Changes >= 3 {
				with = fmt.Sprintf("   %s of them with %s: it may belong there", pct(l.With[0].N, l.Changes), l.With[0].Name)
			}
			fmt.Fprintf(&b, "    %-24s %4d  %4d alone%s\n", l.Dir, l.Changes, l.Alone, with)
		}
	}
	return b.String()
}

func (t RuleTrial) text(changes int, detail, trial bool) string {
	var b strings.Builder
	name := "rule " + t.Name
	if trial {
		name = "the rule tried"
	}
	fmt.Fprintf(&b, "%s (%s %s)\n", name, t.Kind, t.Check)
	if t.Skipped != "" {
		fmt.Fprintf(&b, "  not tried: %s\n", t.Skipped)
		return b.String()
	}
	if t.Kind == "forbid" || t.Kind == "require" {
		fmt.Fprintf(&b, "  reads %s in the tree as it stands\n", plural(t.Reads, "file", "files"))
		if t.Reads == 0 {
			b.WriteString("  a check that reads no file holds nothing: look at files: and except:\n")
		}
	}
	fmt.Fprintf(&b, "  would have refused %d of %s (%s): %s added, %s mended\n",
		len(t.Refused), plural(changes, "change", "changes"), pct(len(t.Refused), changes),
		plural(t.Added, "breach", "breaches"), plural(t.Mended, "breach", "breaches"))
	fmt.Fprintf(&b, "  breaches in the tree: %d before the first of these changes, %d now", t.Start(), t.Now)
	if t.Known > 0 {
		fmt.Fprintf(&b, ", %d of them on its list", t.Known)
	}
	b.WriteString("\n")
	if t.Failed > 0 {
		fmt.Fprintf(&b, "  the command could not be run at %s, which are not counted\n", plural(t.Failed, "change", "changes"))
	}
	switch {
	case (t.Reads > 0 || t.Kind == "absent") && t.Now == 0 && t.Added == 0 && t.Mended == 0:
		b.WriteString("  it found nothing, now or in any of these changes: either the code already keeps it, or it checks nothing\n")
	case len(t.Refused)*4 >= changes && changes >= 20:
		b.WriteString("  a quarter or more of the changes break it: as it stands it is how the code is written, not a rule the code keeps\n")
	case t.Now > 0 && t.Added == 0:
		b.WriteString("  nothing added in these changes: the breaches are old, and the rule can be adopted with them known\n")
	}
	list := t.Refused
	if !detail && len(list) > few {
		list = list[:few]
	}
	for _, r := range list {
		fmt.Fprintf(&b, "  %s %s %s\n", r.Commit.Date, r.Commit.Short(), clip(r.Commit.Subject, 72))
		for i, f := range r.Added {
			if !detail && i == 3 {
				fmt.Fprintf(&b, "      and %d more\n", len(r.Added)-3)
				break
			}
			fmt.Fprintf(&b, "      %s\n", clip(f.String(), 180))
		}
	}
	if n := len(t.Refused) - len(list); n > 0 {
		fmt.Fprintf(&b, "  and %d earlier; -rule %s lists every one\n", n, t.Name)
	}
	return b.String()
}

func clip(s string, n int) string {
	if len(s) > n {
		return s[:n-1] + "…"
	}
	return s
}

// Markdown is the report as a pull request or a design note carries it.
func (r Report) Markdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Backtest of the design: %s\n\n%s.\n\n", r.Repo, r.span())
	if a := r.Areas; a != nil && a.Defined > 0 && a.Code > 0 {
		inside := a.By[0] + a.By[1]
		fmt.Fprintf(&b, "### Areas\n\n%s of %d changes to code stayed inside one area or none.\n\n", pct(inside, a.Code), a.Code)
		b.WriteString("| A change touched | Changes | Share |\n|---|---|---|\n")
		for i, label := range []string{"no area", "one area", "two areas", "three or more"} {
			fmt.Fprintf(&b, "| %s | %d | %s |\n", label, a.By[i], pct(a.By[i], a.Code))
		}
		b.WriteString("\n| Area | Changes | Alone |\n|---|---|---|\n")
		for _, l := range a.Areas {
			fmt.Fprintf(&b, "| %s | %d | %d (%s) |\n", l.Name, l.Changes, l.Alone, pct(l.Alone, l.Changes))
		}
		b.WriteString("\n")
	}
	if len(r.Rules) > 0 {
		b.WriteString("### Rules\n\n| Rule | Check | Files read | Would have refused | Added | Mended | Before | Now |\n|---|---|---|---|---|---|---|---|\n")
		for _, t := range r.Rules {
			if t.Skipped != "" {
				fmt.Fprintf(&b, "| %s | %s | | not tried: %s | | | | |\n", t.Name, t.Kind, t.Skipped)
				continue
			}
			fmt.Fprintf(&b, "| %s | %s | %d | %d of %d (%s) | %d | %d | %d | %d |\n", t.Name, t.Kind, t.Reads,
				len(t.Refused), len(r.Commits), pct(len(t.Refused), len(r.Commits)), t.Added, t.Mended, t.Start(), t.Now)
		}
		b.WriteString("\n")
		for _, t := range r.Rules {
			if len(t.Refused) == 0 {
				continue
			}
			fmt.Fprintf(&b, "#### %s would have refused\n\n", t.Name)
			for i, ref := range t.Refused {
				if !r.Detail && i == few {
					fmt.Fprintf(&b, "- and %d earlier\n", len(t.Refused)-few)
					break
				}
				fmt.Fprintf(&b, "- %s `%s` %s: %s\n", ref.Commit.Date, ref.Commit.Short(), ref.Commit.Subject, plural(len(ref.Added), "breach", "breaches"))
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}
