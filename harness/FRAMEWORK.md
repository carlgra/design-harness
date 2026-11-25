# The design harness

How this repository is worked on. It travels with the harness, so that the method is in
every repository that uses it and nothing has to be looked up elsewhere.

## The idea

An agent writes most of the code now. The judgments that used to be left in the code, and
in the heads of the people who wrote it, are lost every time a session ends. So they are
written down first, in one document per repository, and the code is derived from it.
Design is the new code.

Two animals keep that honest.

**The elephant never forgets.** `design/DESIGN.md` is the repository's memory: what it is
for, how it works, what must always hold, every decision in force with its reason and
what was rejected, and which files realise it. It is current state. A machine, the
checker, holds the code to it on every push.

**The goldfish remembers nothing.** A session with no memory is handed the document, and
only the document, and asked to explain the repository and to build a change from it.
What it cannot do, the document does not say. That is the test of whether the elephant
remembered the right things: everyone who arrives later, person or agent, is a goldfish.

## What is where

```
README.md            what this is and why it exists
CLAUDE.md            how an agent works here: the rules, and the project's own
design/DESIGN.md     the design, with its spine
design/changes/      one short note per change: problem, plan, alternatives, files
harness/             the checker, goldfish, elephant, the push script, the hooks, this file
                     (in a repository nested under a root, the root's: ../harness)
<code>               the project itself
```

## The design document

One file. Its sections, in order:

| Section | What it holds |
|---|---|
| the spine | The frontmatter: the small part a machine reads. Below. |
| Scope | What the repository is for, what it holds, what it deliberately does not. |
| How it works | The parts, what each does, how they fit together. |
| Principles | The few rules of thumb every decision refers back to, each citing the decision that set it. |
| What must always hold | The invariants, each with an id and what proves it. |
| What no machine proves yet | Each invariant held only by a person reading: how it is held today, what would prove it, what makes that due. |
| How a change lands | The path from a note to `main`. |
| Decisions | Every decision in force: what, why, and the alternatives rejected. |
| Files | Every path the spine accounts for and what it is for; then each path a fault would travel and the test that walks it. |

How it is written:

- **It is current state; git is the history.** When a decision changes, the document is
  edited in place and the reason goes in the commit. Nothing in it says "previously" or
  "superseded". A reader should never have to work out which of two statements still
  holds.
- **Ids are kept for life.** A decision is `ADR-0007`, an invariant `INV-012`, and each
  keeps its number however much its text changes, so that a citation always lands. A
  retired id is never reused. Another repository's is cited by name: `platform/ADR-0029`.
- **A decision states a constraint, never a value.** Where it depends on a number or a
  string, it says what the value must satisfy and names the file the value lives in.
  Tuning the value is then a code change, and the design does not go stale.
- **A decision records what was rejected.** The reason and the alternatives are the part
  nobody can reconstruct from the code. A decision without them is a description.
- **Not everything is a decision.** A decision is recorded when something is newly
  decided, reversed or renamed. A change within the decisions in force, even a product
  choice, is a note and a pull request.
- **An invariant says what proves it.** A named test, the checker, a constraint in a
  schema, a CI step; or, honestly, `read at review`. Every one that only a person holds
  has a row under *What no machine proves yet* with a destination and a trigger. A gap
  with a destination is a plan. A gap nobody wrote down is a surprise.
- **Tests are named for the fault they catch.** The design says which paths a fault would
  travel and names the test that walks each, such as
  `TestAnEmptyCartIsRefused`. The name is in the spine, and the checker refuses the push
  if the test does not exist.

## The spine

```yaml
---
files:                 # the code this design accounts for
  - cmd/**             #   a trailing /** is everything under a directory
  - internal/**
  - go.mod             #   otherwise a path, in which * does not cross a /
tests:                 # tests that must exist, by name
  - TestAnEmptyCartIsRefused
validate: make test    # the repository's own gate, run after the checks pass (optional)
commits: conventional  # a commit's subject is a Conventional Commit (optional)
protection: required   # the server's protection of main is read before every push (optional)
notes:                 # what a note under design/changes/ carries
  from: 2026-01-31     #   notes dated before this are never reread
  sections: [Problem, Plan, Alternatives rejected, Files]
---
```

`README.md`, `CLAUDE.md`, `AGENTS.md`, `LICENSE`, `.gitignore` and `.gitattributes` at the
root need no entry, and nothing under `design/` does. What git ignores is not looked at.
A repository asks more of its notes by adding a heading to `sections:`, for instance
`What existing data changes meaning` where a change can alter what stored data means.

## What the machine checks, and what it does not

`go -C harness run ./cmd/check` refuses, with a line saying what and where:

0. *(under a root that keeps a contracts repository)* a new definition no note lists
   with its server, and a definition this repository serves with no test named for it;
   see *Several repositories under one root*;
1. a source file under no glob in `files:`: nobody designed it;
2. a name in `tests:` that is in no test file the design accounts for;
3. a decision or invariant id defined twice, or cited and defined nowhere;
4. a note, dated from `notes: from:`, that lacks one of its sections or leaves one empty.

Then it runs `validate:`. There are always two fixes: amend the design, or change the
code.

That is all. It does not judge whether the design is good, whether a decision's reason
holds, or whether a test proves what its name says. Those are read by a person at review,
with the goldfish's help. A small check that is always right is trusted; a large one that
is sometimes wrong is switched off.

## How a change lands

1. **Run the checker, then read the design** for the part you are about to change. If
   what you need is not there, that is the first thing to fix.
2. **Write the note.** `design/changes/<date>-<slug>.md`, from
   `harness/templates/change.md`: the problem, the plan, the alternatives rejected, and
   the files to change and why. Short. It is kept.
3. **Branch.** `feat/<what>` or `fix/<what>`. Never commit on `main`.
4. **Make the design say what will be true**, then make the code agree with it. A new file
   goes under a glob in `files:`. A new rule gets an invariant and the test that proves
   it, named in `tests:`.
5. **Ask the goldfish** when the change to the design is material:
   `go -C harness run ./cmd/goldfish -change-file design/changes/<the note>.md -out goldfish.md`,
   and paste `goldfish.md` into the pull request.
6. **Push through the one door**, `harness/safe-push`. It refuses `main`, runs the
   checker, pushes the branch and opens the pull request. Never type `git push`.
7. **A person merges.** CI ran the checker again; `main` is protected behind it.

The change ends with `design/DESIGN.md` saying what is now true.

## The goldfish

Two questions, each put to its own fresh session with no tools, handed the prompt and
`design/DESIGN.md` and nothing else:

- **Comprehension.** Say what the repository is for, how it works, what must hold and
  which files realise which parts, writing only what the design supports.
- **Readiness.** Given this change, say what you would build and in which files, which
  decisions and invariants bind it, and what the design leaves you to guess.

Each ends its reply with `VERDICT: PASS` or `VERDICT: FAIL`. The script reads that line
and judges nothing itself.

**It is advice, not a gate.** A FAIL is a list of what the document does not say. Most of
the time the right response is to mend the document and ask again; sometimes the session
is wrong, and the person who merges says so. The verdicts go on the pull request for that
person. A model's opinion never blocks a merge and never permits one.

The prompts are `harness/goldfish/comprehension.md` and `readiness.md`; the model and the
budget are in `harness/goldfish/goldfish.yaml`. It needs the `claude` command-line tool,
signed in.

## The elephant: a design for a repository that already has code

A repository that existed before the method has a design; it is only not written down.
`elephant` writes it down, in steps that are each less mechanical than the last:

| Step | What it does | What it decides |
|---|---|---|
| `elephant survey` | Lists what a machine can see: the files and the globs that account for them, languages, test files and names, the command the repository proves itself with, the documents that may record decisions, CI, whether the history follows a commit convention. | Nothing. |
| `elephant skeleton` | Writes `design/DESIGN.md` with a spine that accounts for every file as it stands, so the checker passes at once, and a body whose every section is a question. | Nothing: `validate:` and `commits:` are written as comments. |
| `elephant draft` | One session, able to read the repository and to write nothing, drafts the whole document and the project's own section of `CLAUDE.md`. The command writes its reply to disk, and only over a skeleton. | Nothing: it proposes. |
| a person | Reads every line. Answers each `(inferred)` and each `TODO(elephant)`. Switches on `validate:`. | Everything. |

`elephant prompt` prints what the drafting session is handed, to give to an agent of your
own instead.

The draft is told to write what the code proves. It can see what the code does; it usually
cannot see why, or what else was considered. So a reason no file states ends
`(inferred)`, a decision whose alternatives are recorded nowhere says
`Not recorded. TODO(elephant)`, and it never invents one. The checker counts the
`TODO(elephant)` marks on every run until none is left. Then ask the goldfish for
comprehension: if a session with no memory can explain the repository from the document,
the elephant remembered enough.

Adopting the method is itself a change, so it has a note: bootstrap writes
`design/changes/<date>-adopt-the-design.md`, and the rules for notes begin that day.
Nothing written before is reread.

## Several repositories under one root

A system is usually several repositories: each service in its own, checked out side by
side in a project folder, with a contracts repository beside them holding the definitions
they talk through.

```
project/               the root: its own repository
  design/DESIGN.md     what binds every repository, and the map of them
  harness/             the one harness
  contracts/           its own repository: every definition the services talk through
  billing/             its own repository: a service
    design/DESIGN.md   billing's design: only what is billing's own
    CLAUDE.md          "read the root's first"; the harness is ../harness
  ledger/              its own repository: another service
```

- **The root holds what binds all of them**, and nothing else: the decisions that apply
  everywhere, the harness, and the map. Its spine lists the nested repositories:

  ```yaml
  repos:
    - path: contracts
      kind: contracts
      remote: git@github.com:acme/contracts.git
    - path: billing
      kind: service
      remote: git@github.com:acme/billing.git
  ```

  The root's `.gitignore` ignores each of them. It pins no commit of any: the map says
  which exist and where they come from. `go -C harness run ./cmd/sync` clones what is
  missing, so a fresh machine is `git clone` of the root and one command. Where a map is
  given, the checker refuses a repository on disk under the root that it does not list.
- **A nested repository has a design and no harness.** Every command is run from its own
  directory with the root's harness: `go -C ../harness run ./cmd/check`,
  `../harness/safe-push`, `../harness/hooks/install`. It is its own git repository:
  branch, commit and push from its own root.
- **A decision lives in the repository whose scope it governs.** What applies everywhere
  is the root's. A nested design cites it by the root's name, `project/ADR-0004`, and
  does not repeat it; the checker follows the citation and refuses one the root does not
  define. The root never cites down.
- **An interface between repositories is defined in the contracts repository and nowhere
  else**, where the root keeps one. A service's design says which definitions it serves
  and which it calls.
- **A definition names who serves it, and the server names its test.** In the contracts
  repository, the note for a change carries a fenced `servers` block listing each RPC and
  request field it adds, the repository that serves it and the ones that call it, or
  `[]`:

  ````
  ```servers
  - name: billing.v1.CreateInvoiceRequest.currency
    server: billing
    callers: [ledger]
  ```
  ````

  The contracts repository's checker refuses a new definition no note lists. Once the
  definition has merged, the checker in `billing` refuses until its spine names a test
  for it:

  ```yaml
  serves:
    - name: billing.v1.CreateInvoiceRequest.currency
      test: TestAnInvoiceIsIssuedInItsCurrency
  ```

  Only what is merged binds, so the definition and its server can be built in either
  order: the server's test first, then the definition; or the definition as a draft
  pull request until the server is ready. Definitions are read from `.proto` files.
- **What the root's spine asks of every repository is written into each.** A nested
  repository's `commits:` and `notes:` are its own lines, taken from the root's when it
  adopted the method. Nothing is inherited at check time, so a design always says what
  binds its repository.

`bootstrap` does this by itself: run on a repository that sits under a root, it copies no
harness and sets all of the above up, then prints the entry to add to the root's map.

## Commands

```
go -C harness run ./cmd/check                    # the checks, then the repository's own gate
go -C harness run ./cmd/check -validate=false    # the checks only
go -C harness run ./cmd/goldfish -change "..."   # two fresh sessions read the design
go -C harness run ./cmd/goldfish -only comprehension
go -C harness run ./cmd/elephant survey          # what a machine can see
go -C harness run ./cmd/elephant draft           # draft the design from the code
go -C harness run ./cmd/subject "fix: ..."       # is this subject a Conventional Commit
harness/safe-push                                # check, push, open the pull request
harness/safe-push --no-pr                        # push only
harness/safe-push --tag v1.2.0                   # tag main's tip, as merged
harness/hooks/install                            # once per clone
go -C harness run ./cmd/sync                     # at a root: make the disk match repos:
```

In a repository nested under a root, the same commands with the root's harness:
`go -C ../harness run ./cmd/check`, `../harness/safe-push`.

The harness is Go, so that each command needs nothing but the Go toolchain; the project
may be in any language. It is a module of its own in `harness/` and is part of the
repository: read it, and change it through the same door as everything else.

## What a person still does

Protect `main`: require a pull request and the `harness` status check, refuse force
pushes, and squash on merge if `commits: conventional` is on. On GitHub,
`go -C harness run ./cmd/protection -set` adds the first three to whatever the branch
already has, when you run it and never otherwise; and with `protection: required` in the
design's frontmatter, `safe-push` reads them before every push and refuses while one is
missing. Switch on "Automatically delete head branches" too, so that a pull request
stacked on another is retargeted to `main` when the one below it merges. Read the design
at review. Merge. No script here does those two, and none should.
