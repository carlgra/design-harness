# The commands are named for what they do

## Problem

The two commands that run model sessions were `elephant` and `goldfish`, and the README
and the method document opened with the two animals. The owner, 2026-11-26: the names
are met with no reason a reader can use, and it would be better to name the commands for
what they are and do than to confuse people. An earlier trial said the same from the
other side: a session with no memory reported `TODO(elephant)` as a term used throughout
and never defined (#17).

## Plan

`cmd/elephant` becomes `cmd/reverse` and `cmd/goldfish` becomes `cmd/coldread`, with
their directories of prompts and settings, the prefix of what they print, the heading of
the report (`## Cold read: <repository>`), the kept reply (`design/reverse-reply.md`),
the mark `TODO(design)` and the two comment marks. The prose says "the design" and "a
cold read" where it said the animals. `bootstrap --update` carries settings from the old
file names and renames the old marks in a project's `CLAUDE.md` and design, so a project
that adopted the method before this loses nothing. ADR-0012 records the names.

The notes under `design/changes/` written before today keep the old names: they say what
was true when they were written.

## Alternatives rejected

- Keep the animals and explain them: the explanation was already in the README.
- Rename in the prose only: a command line still shows the name.
- `draft` for the command: its last step is already `draft`.

## Files

- `harness/cmd/reverse/`, `harness/cmd/coldread/`, `harness/reverse/`,
  `harness/coldread/`: moved, and what they print and read renamed.
- `harness/cmd/check/checks.go`: the mark it counts.
- `harness/internal/session/`: comments.
- `harness/templates/`, `harness/FRAMEWORK.md`, `README.md`, `CLAUDE.md`, `.gitignore`:
  the names and the marks.
- `bootstrap`: the names; an update carries old settings and renames old marks.
- `design/DESIGN.md`: ADR-0012, and the names throughout.
