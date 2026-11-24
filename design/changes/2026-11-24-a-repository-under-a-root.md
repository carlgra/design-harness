# A repository under a root uses the root's harness and takes its rules from it

## The problem

The usual shape of a system is several repositories, not one: each domain service in a
repository of its own, checked out side by side inside a project folder, with a contracts
repository beside them holding the definitions they talk through. The project folder is
itself a repository, and the rules that bind every service are decided there.

`bootstrap` knew nothing of that. Run on a service inside such a folder it copied a second
harness in, wrote rules that made no mention of the root's, and drafted a design as if
the service stood alone. The root had no way to say which repositories belong under it,
or to rebuild them on a fresh machine.

## The plan

A root is a repository whose design lists the repositories nested under it, in its spine
as `repos:`, each with a path, a kind and a remote. A repository nested under a root has
no harness of its own: it uses the root's by relative path, and what the root decided
binds it (ADR-0009).

- `bootstrap`, run on a repository that sits under a directory holding both a harness and
  a design, adopts the method there as a nested repository. It copies no harness, writes
  `CLAUDE.md` from `templates/CLAUDE.nested.md` with the path to the root's harness and
  the instruction to read the root's rules first, writes a CI workflow that checks the
  root out around the repository, and installs the root's hooks by relative path. It then
  prints the entry to add to the root's `repos:`. `--standalone` adopts with a copy
  instead.
- A nested repository's skeleton takes the root's `commits:` and the sections the root
  asks of a note, written into its own spine, and says in its scope where it sits and
  how to cite the root (INV-016). They are written once, there to be read and changed,
  not looked up on every run.
- The survey says where a repository sits: its root, and the siblings the root lists
  with their kinds, a contracts repository among them. A drafting session is handed the
  root's design after the survey and is told to cite it and not restate it.
- The checker holds a root to its map: a nested repository on disk that `repos:` does
  not list is refused (INV-014). In a nested repository it follows a citation of the
  root's, such as `root/ADR-0004`, into the root's design, and refuses one that is not
  defined there (INV-015).
- `cmd/sync` makes the disk match `repos:`: clones what is missing, fetches what is
  there, never merges.

A project that already has the harness need do nothing. One that is a root gains
`repos:` when it wants the map checked.

## Alternatives rejected

- **A copy of the harness in every service.** Each copy is reviewed and updated on its
  own, and within a month the services are held to different versions of the same rule.
- **Inherit the root's spine at check time**, so a service with no `commits:` of its own
  is bound by the root's. A reader of the service's design could then not tell what binds
  it without opening another repository, and a service checked out alone would be held to
  different rules than the same service under its root.
- **Git submodules for the nested repositories.** The root would pin each service's
  commit, so every change to a service becomes a second change to the root; and fresh
  clones and worktrees arrive without them.
- **Hand the goldfish the root's design as well.** It would pass a service's design that
  leans on the root without saying what it relies on. A service's design still has to
  stand on its own for a reader handed only that; the citation says where the reason
  lives.
- **Bring over the checks that bind a contracts definition to the repository that serves
  it.** They read protobuf and assume one layout of one contracts repository. The map
  records a repository's kind so that such a check has what it needs when it is written;
  it is a gap with a destination, not part of this change.

## Files

- `harness/internal/design/design.go`: `repos:` in the spine; finding the root above a
  repository.
- `harness/cmd/check/checks.go`: the map, and citations of the root's ids; tests beside it.
- `harness/cmd/sync/main.go`: new.
- `harness/cmd/elephant/`: where a repository sits, in the survey, the skeleton and what
  a drafting session is handed; `reverse.md` says what to do with it.
- `harness/templates/CLAUDE.nested.md`, `harness/templates/harness.nested.yml`: new.
  `harness/templates/DESIGN.md`: two placeholders.
- `bootstrap`: the nested adoption, `--standalone`.
- `harness/FRAMEWORK.md`, `README.md`, `design/DESIGN.md`: ADR-0009, INV-014 to INV-016.
