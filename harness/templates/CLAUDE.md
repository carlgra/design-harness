# Working in {{NAME}}

<!-- design-harness:begin -->
Read `README.md`, then `design/DESIGN.md`. The method this repository works by is in
`harness/FRAMEWORK.md`.

## Rules

1. **Design before code.** A change to code starts with a short note under
   `design/changes/<date>-<slug>.md` in four parts: the problem, the plan, the alternatives
   rejected, and the files to change and why. It ends with `design/DESIGN.md` saying what
   is now true. The note is kept. `harness/templates/change.md` is the shape.
2. **`design/DESIGN.md` is the design, and it is current state.** What the repository is
   for, how it works, what must always hold, every decision in force with its reason and
   what was rejected, and which files realise it. It is edited in place when a decision
   changes, with the reason in the commit; git is its history. Its frontmatter lists the
   globs the code lives under and the tests that must exist, and the checker refuses a
   source file the design does not account for or a named test that does not exist.
   Decisions and invariants keep their ids, so a citation such as `ADR-0007` always lands.
3. **Never commit on `main`, and never run `git push`.** Branch first, `feat/<what>` or
   `fix/<what>`, and push with `harness/safe-push`, which runs the checker and opens the
   pull request. A person merges.
4. **A new decision is recorded; a change within the decisions in force is not one.** Add
   or amend a decision in `design/DESIGN.md` only when something is newly decided, reversed
   or renamed. Tuning a value a decision already constrains is code.
5. **Anything found and not finished is an issue before the conversation ends**, with what
   was seen and what it needs.

## Before you build

Run the checker, `go -C harness run ./cmd/check`, then read `design/DESIGN.md` for the
part you are about to change. If what you need is not there, that is the first thing to
fix. When a design change is material, `go -C harness run ./cmd/coldread -change "..."`
asks two sessions with no memory whether the document explains the repository and whether
the change could be built from it alone; paste what they say into the pull request. It is
advice for whoever merges, not a gate.

## Setup after clone

```
harness/hooks/install
```
<!-- design-harness:end -->

## This project

<!-- design:project:begin -->
TODO(design): how to build, test and run this project, and what an agent would get wrong
here without being told. `go -C harness run ./cmd/reverse draft` writes a first version.
<!-- design:project:end -->
