# The method lives in a repository of its own, and can be adopted by any project

## The problem

The method, one design document per repository with a checker that holds the code to it
and sessions with no memory that test it, existed only inside the project it grew up in.
Its scripts carried that project's rules beside the general ones, and its instructions
were spread across that project's own documents. Another project could not take it
without taking that one apart, and a project that already had code had no way to start:
the method assumes a design, and an existing codebase has never written one down.

## The plan

A repository that holds only the method: `harness/`, everything a project needs, written
out so that it refers to nothing outside itself; `bootstrap`, which copies it into a
project and adds what is missing; and this repository's own design, held to the same
checker.

Carried over as they were: the walk, the two checks on files and named tests, the
goldfish and its two prompts, the push script, the hooks, the commit convention.

Left behind, as that project's own: the checks that bind interface definitions to the
repositories that serve them, the manifest of nested repositories and the command that
clones them, and the release command.

New here:

- `cmd/elephant`: a survey, a skeleton, and a draft by a session that reads and cannot
  write, for a repository that has code and no design (ADR-0004).
- The checker holds ids to being defined once and citations to landing (INV-003), and
  holds a note to the sections the spine asks for, from a date the spine gives
  (INV-004), in place of one fixed section and a date in the code.
- The goldfish runs away from the project's directory, and comprehension can be asked
  alone without a change to describe.
- `hooks/install` refuses to switch off hooks a repository already uses (ADR-0006).
- `safe-push` judges against the remote's default branch rather than assuming its name.

## Alternatives rejected

- **Publish the original harness as it was.** It would have published one project's rules
  as if they were the method, and would not have run anywhere else.
- **A template repository to start new projects from.** It does nothing for a project that
  already exists, which is most of them, and a copy made by a script can be made again
  when the harness changes.
- **Keep the nested-repository manifest.** One harness can still be shared by repositories
  nested under it, because a command finds the harness in the nearest directory above;
  the manifest that cloned them was that project's arrangement, not part of the method.

## Files

- `harness/**`: the checker, goldfish, elephant, the convention, the push script, the
  hooks, the templates and `FRAMEWORK.md`.
- `bootstrap`: adoption.
- `design/DESIGN.md`: this repository's design, with decisions ADR-0001 to ADR-0008 and
  invariants INV-001 to INV-013.
- `README.md`, `CLAUDE.md`, `LICENSE`, `.github/workflows/harness.yml`, `.gitignore`.
