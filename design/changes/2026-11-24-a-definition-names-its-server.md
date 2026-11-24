# A definition names the repository that serves it, and the server names its test

## The problem

Where services talk through definitions kept in a contracts repository, a new RPC or
request field can merge, be generated into every client and be called, while no
repository serves it, or while the server accepts the request and ignores the new field.
Every repository's own checks pass throughout. The root's map recorded which repository
holds the contracts and nothing read it (issue #2).

## The plan

ADR-0010, in two halves, each checked where it is owned.

In the repository the root's map gives the kind `contracts`: every note the rules bind
carries a fenced `servers` block, a YAML list of each RPC and request field the change
adds with the repository that serves it and the ones that call it, by their paths on the
map, or `[]`. The checker scans the `.proto` files in the working tree and at the merged
state and refuses a name the first has and the second does not that no note lists
(INV-017).

In every other repository under that root: the checker reads the contracts repository's
notes and definitions at its merged state, and for each entry naming this repository as
server whose name a merged definition declares, requires an item under `serves:` in the
spine with a test that exists (INV-018). Only what is merged binds. A contracts
repository that is not checked out is a refusal.

Around it: the first note bootstrap writes in a contracts repository carries an empty
block; the nested CI workflow checks out the whole history of its repository and the
contracts repository beside it; a drafting session is told to fill `serves:` for what it
can name a test for.

A project that already has the harness is bound by none of this until its root's map
lists a repository of kind `contracts`. From then, notes in that repository dated from
its `notes: from:` need the block, and each server needs `serves:` for what is already
listed and merged.

## Alternatives rejected

- **A registry at the root of who serves what.** A third place to keep in step, changed
  in a third pull request.
- **An option in the definition naming its server.** It ties the check to one format and
  puts a deployment fact into an interface.
- **Bind a server from the note alone.** It would be refused for a field that does not
  exist yet.
- **Skip when the contracts repository is not on disk.** In CI that is every run of a
  workflow nobody extended.
- **Wait for a second format before writing the first.** The rules are the same for any
  format; only the scan differs, and protobuf is the one in use.

## Files

- `harness/cmd/check/contracts.go` and its test: new. `checks.go`: calls it.
- `harness/internal/design/design.go`: `serves:`.
- `harness/cmd/elephant/`: the first note in a contracts repository, `where` for
  bootstrap, and `reverse.md`.
- `harness/templates/harness.nested.yml`, `change.md`, `CLAUDE.nested.md`.
- `bootstrap`: the contracts checkout in a nested workflow.
- `harness/FRAMEWORK.md`, `README.md`, `design/DESIGN.md`: ADR-0010, INV-017, INV-018.
