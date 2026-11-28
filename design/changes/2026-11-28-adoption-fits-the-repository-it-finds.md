# Adoption fits the repository it finds

## The problem

Trials on existing repositories found seven places where the harness assumed a
repository shaped like its own, or spent effort before saying what was wrong.

- Bootstrap added the method's rules to `CLAUDE.md` even where that file is a link to
  `AGENTS.md` or one line importing it. An agent that reads `AGENTS.md` was told nothing
  of the design, the notes or `safe-push` (issue #43).
- Bootstrap wrote a CI workflow with `runs-on: ubuntu-latest` into a repository all of
  whose workflows run on its own runners, and did not say it had chosen (issue #44).
- `safe-push` accepted `feat/` and `fix/` branches only, so a repository whose branches
  are named by another rule could not use it (issue #45).
- `bootstrap --update` refused a `harness/` with no `FRAMEWORK.md`, which is what an
  earlier form of this harness looks like, and once past that printed the advice for a
  first adoption to a project that had a design (issue #20).
- A citation of another repository's decision written `billing ADR-0045`, without the
  slash, was refused as an id the citing design does not define, with nothing to say
  what form was wanted (issue #21).
- The contracts rules bind nothing unless a map entry has `kind: contracts`. A root
  that listed its repository of definitions under another kind was told OK while six
  planted faults passed, and a `serves:` entry naming a test that does not exist was
  among them (issue #23).
- `coldread` started its sessions on a document that was not a design, a skeleton or a
  README under `files: []`, and said so only in each verdict (issue #19).

## The plan

1. **The rules go where the instructions live.** `reverse where` prints `rules=`: the
   file that holds the repository's standing instructions. It is `AGENTS.md` where
   `CLAUDE.md` is a link to it or holds nothing but an import of it, the file that
   already holds the method's rules where one does, and `CLAUDE.md` otherwise.
   Bootstrap writes the rules there and says which file when it is not `CLAUDE.md`;
   `reverse draft` writes the project's section to the same file; `--update` replaces
   the rules in it, in place, so a link stays a link.
2. **The workflow runs where the repository's workflows run.** `reverse where` reads
   `runs-on:` from every job of the workflows already under `.github/workflows/`. Where
   all agree on one plain label, bootstrap writes that. Where they differ, or one is a
   list, an expression or a matrix, it keeps `ubuntu-latest` and prints that the runner
   is a line to check, with what it found.
3. **A design can say how its branches are named.** A new frontmatter key, `branches:`,
   holds a regular expression a change branch's whole name must match; with none it is
   `feat/` or `fix/` as before. `cmd/branch` answers for `safe-push`, as `cmd/subject`
   does for a title. Work `safe-push` must move goes to a name it can build that
   matches; where it can build none, it refuses, moves nothing and says what to type.
4. **An update knows an earlier harness, and says what an update leaves.** A
   `harness/` with `cmd/check` and a `go.mod` for the module `harness` is this harness,
   with or without `FRAMEWORK.md`. `--update` replaces it; without `--update` bootstrap
   stops and says to use it. After an update the closing text is about an update: read
   the diff, run the checker, expect it to name tests the design cites from the harness
   it had. Bootstrap lists those it can see: tests the old harness held, the new one
   does not, and the design names.
5. **A dangling citation says the form.** The slash stays the only form. The refusal
   now says that another repository's id is cited as `name/ADR-0045`, and where the
   word before the id is a repository's name on the map, names the citation it expects.
6. **The checker says when the contracts rules are off.** Where no map entry is the
   contracts repository and the signs of one are there, a `serves:` key, a `servers`
   block in a note, or `.proto` files in a mapped repository, the checker prints a
   line saying the rules are off and how to switch them on. It does not refuse. A test
   `serves:` names must exist in every repository, whatever the map says.
7. **A cold read refuses a document that is not a design, before any session.** A
   skeleton, a design that accounts for no files, one missing a section that holds what
   governs, or one whose every section is still a question is refused with what to do
   first. It is a third outcome, not a FAIL.

A project that already has the harness need do nothing until it next updates. After
`bootstrap --update`: a project whose rules were added to a `CLAUDE.md` that only
imports `AGENTS.md` keeps them there, since the file that holds them is the rules
file; to move them, cut the marked block into `AGENTS.md`. A project that wants its own
branch names adds `branches:` to its design. A `serves:` entry that names a missing
test is refused from the first run of the new checker.

## Alternatives rejected

- **Write the method's rules to both `CLAUDE.md` and `AGENTS.md`.** Two copies of one
  rule, and an agent that follows the import is handed both.
- **Decide the rules file in the shell script.** `reverse draft` needs the same answer,
  and two implementations of one decision drift.
- **Always copy the first `runs-on:` found.** Where workflows differ the first is a
  guess, and a guess written without a word is the fault being fixed.
- **Read `runs-on:` with `grep` in the script.** It cannot tell a label from a list, an
  expression or a matrix; the scripts parse nothing (ADR-0005).
- **A list of branch prefixes instead of a pattern.** It cannot say "a ticket number
  first", which is the naming rule the trial met.
- **Have `safe-push` ask for a branch name.** It runs unattended, for an agent.
- **A flag to replace whatever is in `harness/`.** It would delete a directory that is
  not a harness on one mistyped command.
- **Read `name ADR-0045` as a citation of `name`.** Two forms for one thing, and a
  sentence that happens to end in a repository's name before an id of this design would
  be read as another's. One form, and a refusal that says it.
- **Refuse when the contracts rules look as if they should be on.** The signs are
  evidence, not proof: a repository may hold `.proto` files for another purpose. A
  check that is sometimes wrong is switched off (ADR-0002); a line is not.
- **Stop a cold read whenever the checker fails.** A person may want advice mid-change,
  with one file still unaccounted for. A skeleton has no such use.

## Files

- `bootstrap`: asks `reverse where` once; writes the rules to the file it names; fits
  the workflow's runner; knows an earlier harness; closing text for an update.
- `harness/cmd/reverse/where.go`, `main.go`: the rules file and the runner, printed by
  `where`; the draft writes the project's section to the rules file.
- `harness/reverse/reverse.md`: says where the project's rules go.
- `harness/safe-push`, `harness/cmd/branch/`, `harness/internal/convention/`,
  `harness/internal/design/design.go`: the `branches:` key and the check.
- `harness/cmd/check/checks.go`, `contracts.go`: the citation's refusal; the line when
  the contracts rules are off; `serves:` tests checked everywhere.
- `harness/cmd/coldread/main.go`: the refusal before any session.
- `harness/templates/DESIGN.md`, `CLAUDE.md`, `CLAUDE.nested.md`, and `CLAUDE.md` at
  the root: the `branches:` key, and the rule on branch names.
- `design/DESIGN.md`, `harness/FRAMEWORK.md`, `README.md`: what is now true.
- Tests beside each: `bootstrap_test.go`, `where_test.go`, `safe_push_test.go`,
  `convention_test.go`, `checks_test.go`, `contracts_test.go`, `coldread/main_test.go`.
