# Working in {{NAME}}

<!-- design-harness:begin -->
This repository is one of several nested under `{{PARENT}}`, at `{{PATH}}/`. Read
`README.md`, then `design/DESIGN.md`, then `{{PARENT}}`'s own `CLAUDE.md` and
`design/DESIGN.md`, `{{UP}}/CLAUDE.md` and `{{UP}}/design/DESIGN.md`: every rule and
decision there applies here unchanged. The method is in `{{HARNESS}}/FRAMEWORK.md`.

## Rules

1. **Design before code.** A change to code starts with a short note under
   `design/changes/<date>-<slug>.md` in four parts: the problem, the plan, the alternatives
   rejected, and the files to change and why. It ends with `design/DESIGN.md` saying what
   is now true. The note is kept. `{{HARNESS}}/templates/change.md` is the shape.
2. **`design/DESIGN.md` is the design, and it is current state.** What the repository is
   for, how it works, what must always hold, every decision in force with its reason and
   what was rejected, and which files realise it. It is edited in place when a decision
   changes, with the reason in the commit; git is its history. Its frontmatter lists the
   globs the code lives under and the tests that must exist, and the checker refuses a
   source file the design does not account for or a named test that does not exist.
   Decisions and invariants keep their ids, so a citation such as `ADR-0007` always lands.
3. **What `{{PARENT}}` decided is cited, not repeated.** A decision or invariant of the
   root's is written `{{PARENT}}/ADR-0004` or `{{PARENT}}/INV-002`, and the checker
   refuses one the root's design does not define. This repository's design records only
   what is its own. A decision that would bind the other repositories too belongs in the
   root.
4. **This is its own git repository. Work in it from its own root.** The harness is
   `{{HARNESS}}`, the root's: there is no copy here, and nothing committed here points
   outside this repository except to it. If `{{PARENT}}` keeps a contracts repository, an
   interface between repositories is defined there and nowhere else: a note there lists
   each new RPC and request field in a `servers` block with the repository that serves
   it, and the repository that serves one names a test for it under `serves:` in its
   design. The checker refuses either half that is missing.
5. **Never commit on `main`, and never run `git push`.** Branch first, `feat/<what>` or
   `fix/<what>`, and push with `{{HARNESS}}/safe-push`, from here, which runs the checker
   and opens the pull request. A person merges.
6. **A new decision is recorded; a change within the decisions in force is not one.** Add
   or amend a decision in `design/DESIGN.md` only when something is newly decided, reversed
   or renamed. Tuning a value a decision already constrains is code. Only what
   must be governed goes in the design: a choice inside one function or file, which
   whoever edits it can see and change safely, stays in the code.
7. **A rule's breach is mended, never listed.** Where the checker refuses a breach of a
   rule (`rules:` in the design or a spine), change the code. The lists under
   `design/rules/` hold the breaches that were there when a rule was adopted and only
   shrink: never add a line to one, and when you mend a listed breach run
   `go -C {{HARNESS}} run ./cmd/check -known` to take it off.
8. **Anything found and not finished is an issue before the conversation ends**, in the
   repository whose scope it falls in, with what was seen and what it needs.

## Before you build

Run the checker from this directory, `go -C {{HARNESS}} run ./cmd/check`, then read
`design/DESIGN.md` for the part you are about to change. Where the design divides the code into areas (`spines:` in its
frontmatter), each area's own rules are in its spine, `design/spines/<area>.md`:
`go -C {{HARNESS}} run ./cmd/check -for <path>...` names the documents to read for the paths
you are about to change. Read those, and not the spines of other areas. If what you need is not there,
that is the first thing to fix. When a design change is material,
`go -C {{HARNESS}} run ./cmd/coldread -change "..."` asks two sessions with no memory
whether the document explains the repository and whether the change could be built from
it alone; paste what they say into the pull request. It is advice for whoever merges, not
a gate.

## Setup after clone

Work in place, inside `{{PARENT}}`, so that `{{HARNESS}}` exists. Then, once:

```
{{HARNESS}}/hooks/install
```
<!-- design-harness:end -->

## This project

<!-- design:project:begin -->
TODO(design): how to build, test and run this project, and what an agent would get wrong
here without being told. `go -C {{HARNESS}} run ./cmd/reverse draft` writes a first version.
<!-- design:project:end -->
