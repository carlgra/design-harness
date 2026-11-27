# The method says what it is for: a design holds what must be governed, in one format

## The problem

The method is a tool for documenting a codebase's design and architecture and enforcing
them. Nothing in the repository said so. The README and `harness/FRAMEWORK.md` opened
with the mechanics; the drafting prompts told a session to write what the code proves.
On a first trial in a large existing repository the draft came back with rules at the
level of one function (#34), and the repository's own architecture documents and rules
were left standing beside the new design, two of each (#42). Asked whether stating
rules for the code to be held to changed what the tool is, the owner answered that it
has always been the point, that only things that need to be governed go in a design,
and that a project's existing rules are to be subsumed into this format so that there
is no duplication and every project has the same standards.

## The plan

Record it as ADR-0014 and say it wherever a reader or a session first meets the method.

- `README.md` and `harness/FRAMEWORK.md` open with what the tool is for, the test of
  what belongs in a design, and that existing rules are brought in and the old copies
  retired.
- `harness/reverse/reverse.md` and `harness/reverse/spine.md` gain a section, *What
  belongs in a design*: the test, examples either side of it, and the instruction to
  bring in what existing documents govern, naming the source, and not to point at them.
- The templates' opening paragraphs, the method's rules in `CLAUDE.md`, and the
  readiness prompt say the same in a sentence each. The readiness prompt no longer
  counts a free choice the design leaves to the builder as a gap.
- `design/DESIGN.md`: the scope, a principle, ADR-0014, and a row under *What no machine
  proves yet*, since nothing yet holds a design to it.

The prompts were then run for real, five rounds, on a repository of about 590 files in
four parts whose earlier design, some 3,500 lines, stood in as an existing architecture
document. Each round showed a fault only a live session shows, and each is mended here:

- The spine template's example citation was read as a real one, so a fresh skeleton was
  refused. It is written so that it is not.
- A draft renumbered the repository's existing invariants to make room for the
  method's own. Existing ids are kept exactly; the method's rule takes the next free
  one, and new rules anywhere are numbered above the highest existing number, so that
  a bare number in a code comment still means one thing (#41).
- Rules of one area stayed in the design (#38). The design's draft leaves a one-area
  rule out, keeps of a wider decision only what binds several areas, and is given a
  size to aim at.
- A session drafting a spine was never shown a document outside its area, so an area's
  existing rules were lost. The survey of an area lists such documents.
- Documents wrote ids they did not define, to say where a rule came from or to explain
  their numbering, and the checker refused them. The prompts say why not to, and how to
  say it without one.
- An area is described as a part a change can stay inside, never a layer or the tests'
  tree, holding its own tests wherever they are kept (#36, #39); a design says nothing
  of whether a spine is written (#40) or of how it was drafted.
- A cold read of one area counted the other spines' absence as a gap. It is told they
  are withheld on purpose.
- The checker and `reverse draft` counted the marks left for a person differently (#46).
  Both now count with `design.Marks`.

What enforces the purpose, a check per rule, a warning period, a size for a document,
is #28 to #35 and is designed after this.

## Alternatives rejected

- **Leave it implicit and fix only the prompts.** The prompts follow the documents; a
  purpose stated nowhere is argued again at every change.
- **Say that a design may point at existing documents.** The owner ruled it out: two
  places that say what governs are two standards.
- **Wait and state the purpose with the rule engine.** The drafts being made now are
  wrong in the way this corrects, and the wording costs nothing to ship first.

## Files

- `README.md`, `harness/FRAMEWORK.md`: the opening, and *What belongs in it*.
- `harness/reverse/reverse.md`, `harness/reverse/spine.md`: the new section and the
  wording of *Decisions* and *What must always hold*.
- `harness/templates/DESIGN.md`, `spine.md`, `CLAUDE.md`, `CLAUDE.nested.md`: a sentence
  each.
- `harness/coldread/readiness.md`: what is and is not a gap.
- `design/DESIGN.md`: scope, principle, ADR-0014, the gap, and the new test.
- `harness/internal/design/design.go`, `design_test.go`: `Marks`, the one count.
- `harness/cmd/check/checks.go`, `harness/cmd/reverse/main.go`, `spine.go`: use it; the
  survey of an area lists documents elsewhere; `survey.go` carries the list.
- `harness/cmd/coldread/main.go`: what a session reading one area is told.
- `harness/cmd/reverse/spine_test.go`: the template's example, and the documents
  elsewhere.
