# A rule a person states is checked, tried against the history, and adopted with its breaches known

## Problem

A trial on a large existing repository reported six things that are one thing. Every
rule in a design was one a session read out of the code; a person could not state a rule
the code does not keep yet and have the code held to it. A rule could name no check of
its own. A rule was in force or absent, with no period in which it warns. Nothing
recorded, change by change, what a rule found. Nothing said what a rule coming into
force does about the breaches already in the tree. And a rule could not be tried against
the repository's history before it was adopted.

Beside them: no document had a size to hold to, though five real drafts came out at 744
to 982 lines where a few hundred were asked for; nothing scored a division into areas
against the way the code changes; and a repository's existing architecture documents
were brought into the design with nothing to say what was taken from each, or that the
old copy was still there.

## Plan

ADR-0015 and ADR-0016.

- A rule, an invariant or a decision, may be given a check under `rules:` in the
  frontmatter of the document that states it: `forbid:`, `require:`, `absent:` or
  `run:`, with `files:` and `except:`. The checker runs it and reports under the rule's
  id.
- The breaches in the tree when a rule is adopted are recorded by `check -known` in
  `design/rules/<id>.known`. Those are allowed, every other is refused, and the list
  only shrinks: a mended entry is refused until taken off, and recording never adds.
  This is the answer to the undecided question.
- `from:` a date makes a rule report until that day and refuse from it.
- `check -report` writes each run as Markdown, and to a GitHub job's summary.
- `backtest` reads the last changes on the default branch and says which each rule
  would have refused, and how many changes stayed inside one area. It tries a pattern
  given on the command line too, before it is written anywhere.
- The checker weighs each document and warns of one over `size:` lines, 600 by default.
- `replaces:` lists the documents a design took its rules from; the checker names each
  one still on disk. A draft fills it in and ends with what it took from each, kept in
  `design/replaces.md`.
- `reverse survey` lists the directories that change together. `reverse draft -rules`
  hands a draft the rules a person states.
- A glob may hold `**` as any path element.

Tried on a private repository of 590 files with 300 changes of history, with five rules
written from its own invariants: the replay took under six seconds, named the changes
that brought each breach in, and the area score showed two areas that changed together
in three quarters of the rarer one's changes. That run found and fixed: documents under
`design/` were being read as code in the history; a rule newly listed was reported as a
list that had grown; and a history shorter than asked for was not said.

A project that already has the harness need do nothing: every key is optional, and a
design with no `rules:` is checked as before. It gains two lines of output, the size of
its design, and a warning if that is over 600 lines.

## Alternatives rejected

- **Hold only files changed after a date.** A file touched for another reason fails for
  what it always held, and an untouched file's breach can be copied.
- **Refuse until the tree is clean.** No rule could then be adopted where rules are
  needed most.
- **Let the drafting session write the checks.** A pattern nobody tried is noise; the
  backtest is how one is tried, and a person adds it.
- **Refuse a document over its size.** A document is cut down by a person deciding what
  belongs in it, not by a push that cannot go out.
- **Replay the history by checking out every commit.** Kept for commands, behind `-run`;
  for a pattern the files a change touched say everything.

## Files

- `harness/internal/rules/`: new. A check, a breach, the list of known ones.
- `harness/internal/history/`: new. Past changes from git.
- `harness/cmd/backtest/`: new. The areas and the rules against the history.
- `harness/cmd/check/rules.go`: new. Rules, `-known`, sizes, replaced documents, the report.
- `harness/cmd/check/checks.go`, `main.go`: warnings, and the two flags.
- `harness/internal/design/design.go`: `rules:`, `size:`, `replaces:`; `**` in a glob.
- `harness/cmd/reverse/`: directories that change together, `-rules`, what a draft took.
- `harness/reverse/reverse.md`, `spine.md`: `replaces:`, the last part of a reply, no checks from a session.
- `harness/templates/DESIGN.md`, `CLAUDE.md`, `CLAUDE.nested.md`: the new keys; a breach is mended, never listed.
- `harness/FRAMEWORK.md`, `README.md`, `design/DESIGN.md`: what is now true.
