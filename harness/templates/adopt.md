# This repository has a design, and a machine holds the code to it

## The problem

What this repository is for, how it works, what must always hold and why each choice was
made lived in the code and in the heads of the people who wrote it. Someone new, or a
session with no memory, had to rebuild it by reading everything, and nothing said when a
change broke a rule nobody had written down.

## The plan

Adopt the design-harness method. `design/DESIGN.md` becomes the one design
document, current state, edited in place. Its frontmatter accounts for every source file
and names the tests that must exist; `{{HARNESS}}/` holds the checker that refuses a push when
either is untrue, the push script and the hooks. From {{DATE}} every change starts with a
note here, in four parts.

The first design is reverse-engineered: a survey of what a machine can see, then a draft
by a session that read the repository, with everything it could not find in the code
marked `(inferred)` or `TODO(elephant)`. A person answers each mark; until then the design
is a draft and the checker says so on every run.

## Alternatives rejected

- **Write the design by hand first, adopt afterwards.** The repository would go unchecked
  for as long as that took, and a blank page is slower to correct than a wrong draft.
- **Leave the decisions where they are**, in commit messages, comments and separate
  documents. Nothing then says which of them is still in force.

## Files

- `design/DESIGN.md`: the design.
- `design/changes/`: one note per change, this one first.
- `{{HARNESS}}/`: the checker, goldfish, elephant, the push script, the hooks and the
  method itself, `{{HARNESS}}/FRAMEWORK.md`.
- `CLAUDE.md`: how an agent works here.
- `.github/workflows/harness.yml`: the checker as the status check `main` requires.
