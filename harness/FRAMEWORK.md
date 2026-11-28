# The design harness

How this repository is worked on. It travels with the harness, so that the method is in
every repository that uses it and nothing has to be looked up elsewhere.

## The idea

This is a way of writing down a codebase's design and architecture and enforcing them:
the rules it is governed by, in one format, with a machine that holds the code to them.

An agent writes most of the code now. The judgments that used to be left in the code, and
in the heads of the people who wrote it, are lost every time a session ends. So they are
written down first, in one document per repository, and the code is derived from it.
Design is the new code.

Two things keep that honest.

**The design is the repository's memory.** `design/DESIGN.md` holds what it is
for, how it works, what must always hold, every decision in force with its reason and
what was rejected, and which files realise it. It is current state. A machine, the
checker, holds the code to it on every push.

**A cold read tests it.** A session with no memory is handed the document, and
only the document, and asked to explain the repository and to build a change from it.
What it cannot do, the document does not say. That is the test of whether the design
holds the right things: everyone who arrives later, person or agent, starts with no
memory too.

## What is where

```
README.md            what this is and why it exists
CLAUDE.md            how an agent works here: the rules, and the project's own
                     (in AGENTS.md, where CLAUDE.md only points at it)
design/DESIGN.md     the design, with its frontmatter
design/spines/       in a large repository: one document per area of the code, its spine
design/changes/      one short note per change: problem, plan, alternatives, files
harness/             the checker, the cold read, the reverse command, the push script, the hooks, this file
                     (in a repository nested under a root, the root's: ../harness)
<code>               the project itself
```

## The design document

One file. Its sections, in order:

| Section | What it holds |
|---|---|
| the frontmatter | The small part a machine reads. Below. |
| Scope | What the repository is for, what it holds, what it deliberately does not. |
| How it works | The parts, what each does, how they fit together. |
| Principles | The few rules of thumb every decision refers back to, each citing the decision that set it. |
| What must always hold | The invariants, each with an id and what proves it. |
| What no machine proves yet | Each invariant held only by a person reading: how it is held today, what would prove it, what makes that due. |
| How a change lands | The path from a note to `main`. |
| Decisions | Every decision in force: what, why, and the alternatives rejected. |
| Files | Every path the frontmatter accounts for and what it is for; then each path a fault would travel and the test that walks it. |

What belongs in it:

- **Only what must be governed.** A rule or a decision belongs in a design when it has to be governed: it binds more
  than one part or more than one person's work, a change could break it without anyone
  noticing at the place it was broken, and whoever changes the code must know it
  beforehand. A choice made inside one function or one file, which whoever edits it can
  see and can change safely, is code, and stays in the code.
  A design that holds everything is read by nobody, and a rule lost among a hundred
  descriptions is not kept.
- **One place for a rule, in one format.** A project that already has architecture
  documents, standing rules or decision records brings what they govern into the design
  when it adopts the method, as decisions and invariants, and retires the old copies. The
  design does not point at a second set of rules that stays in force: two places that
  say what governs are two standards. Every project that works this way then holds its
  standards in the same shape, and a person or an agent who knows one knows where to
  look in the next.

How it is written:

- **It is current state; git is the history.** When a decision changes, the document is
  edited in place and the reason goes in the commit. Nothing in it says "previously" or
  "superseded". A reader should never have to work out which of two statements still
  holds.
- **Ids are kept for life.** A decision is `ADR-0007`, an invariant `INV-012`, and each
  keeps its number however much its text changes, so that a citation always lands. A
  retired id is never reused. Another repository's is cited by name: `platform/ADR-0029`.
  The form is the name, a slash and the id, with nothing between: `platform ADR-0029`,
  with a space, is read as this design's own id, and refused if it defines none.
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
  `TestAnEmptyCartIsRefused`. The name is in the frontmatter, and the checker refuses the push
  if the test does not exist.

## The frontmatter

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
branches: '[A-Z]+-[0-9]+/.+'  # how a change branch is named, if not feat/ or fix/ (optional)
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
`branches:` is a regular expression, and a change branch's whole name must match it: the
one above accepts `SHOP-12/a-fourth-line` and refuses `feat/a-fourth-line`. `safe-push`
pushes a branch named so and no other. Work it finds on `main` or a stray branch it moves
to a name it can build that matches, and where it can build none it stops and says what
to type.

## Spines: a large repository, one area at a time

A design is read whole before a change. In a repository of a few thousand files that
works. In one of millions of lines it does not: the document is either too thin to build
from or too long to read, and an agent that reads the rules of every area to change one
has filled its context with rules that do not bind it.

Such a repository divides its code into **areas**, in the design's frontmatter, and gives
each a **spine**: a document with the design's sections and the rules of that area alone.

```yaml
# design/DESIGN.md
files:                          # what no area holds: the design's own
  - package.json
  - tools/**
spines:                         # the areas, defined here and nowhere else
  - name: checkout
    files: [src/checkout/**, src/payments/**]
  - name: catalogue
    files: [src/catalogue/**]
```

```
design/DESIGN.md              what binds every area, how the areas fit, and the list of them
design/spines/checkout.md     the rules of checkout: its invariants, decisions, files, tests
design/spines/catalogue.md    the rules of catalogue
```

- **The design says where the areas are; a spine says what holds inside one.** A spine's
  frontmatter has one key, `tests:`, and its tests must be in its own area's files.
- **A file is under one area at most.** One under none is the design's own and must be
  under its `files:`. An area with no spine, and a spine with no area, are refused.
- **You read the design and the spine of the area you are changing, and no other.**
  `go -C harness run ./cmd/check -for src/checkout/cart.ts` prints which documents those
  are. A change that crosses two areas reads both.
- **Each document numbers its own ids.** Inside `checkout.md`, `ADR-0003` is its own. From
  anywhere else it is `checkout/ADR-0003`. A spine cites the design as
  `<repository>/ADR-0004`. Each is looked up, and refused if it is not there.
- **A rule that binds more than one area belongs in the design**, and is cited from the
  spines, not copied into them.
- **A cold read is of one area:** `coldread -spine checkout` hands a session the design
  and that spine. Without `-spine` it is handed the design alone.

Adopt it area by area. Start with the design holding everything under `files:`; when an
area's rules are worth writing down, add it under `spines:`, and

```
go -C harness run ./cmd/reverse skeleton -spine checkout   # the spine, every section a question
go -C harness run ./cmd/reverse draft -spine checkout      # one session reads that area and drafts it
go -C harness run ./cmd/coldread -spine checkout -only comprehension
```

The drafting session is handed the design and a survey of that area's files, and reads
the area; it is not handed the other spines. `reverse draft` of the design itself, in a
large repository, proposes the areas and writes each one's skeleton.

A repository with no `spines:` is one document, and nothing above applies to it.

## What the machine checks, and what it does not

`go -C harness run ./cmd/check` refuses, with a line saying what and where:

0. *(under a root that keeps a contracts repository)* a new definition no note lists
   with its server, and a definition this repository serves with no test named for it;
   see *Several repositories under one root*;
1. a source file under no glob in `files:`: nobody designed it;
2. a name in `tests:` that is in no test file the design accounts for;
3. a decision or invariant id defined twice, or cited and defined nowhere;
   *(where the design defines areas)* an area with no spine or a spine with no area, a
   file under two areas, a test a spine names that is not in its area, and a citation of
   another document's id that the document does not define;
4. a note, dated from `notes: from:`, that lacks one of its sections or leaves one empty.

Then it runs `validate:`. There are always two fixes: amend the design, or change the
code.

That is all. It does not judge whether the design is good, whether a decision's reason
holds, or whether a test proves what its name says. Those are read by a person at review,
with a cold read's help. A small check that is always right is trusted; a large one that
is sometimes wrong is switched off.

## How a change lands

1. **Run the checker, then read the design** for the part you are about to change. If
   what you need is not there, that is the first thing to fix.
2. **Write the note.** `design/changes/<date>-<slug>.md`, from
   `harness/templates/change.md`: the problem, the plan, the alternatives rejected, and
   the files to change and why. Short. It is kept.
3. **Branch.** `feat/<what>` or `fix/<what>`, or as the frontmatter's `branches:` says.
   Never commit on `main`.
4. **Make the design say what will be true**, then make the code agree with it. A new file
   goes under a glob in `files:`. A new rule gets an invariant and the test that proves
   it, named in `tests:`.
5. **Ask for a cold read** when the change to the design is material:
   `go -C harness run ./cmd/coldread -change-file design/changes/<the note>.md -out coldread.md`,
   and paste `coldread.md` into the pull request.
6. **Push through the one door**, `harness/safe-push`. It refuses `main`, runs the
   checker, pushes the branch and opens the pull request. Never type `git push`.
7. **A person merges.** CI ran the checker again; `main` is protected behind it.

The change ends with `design/DESIGN.md` saying what is now true.

## The cold read

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

The prompts are `harness/coldread/comprehension.md` and `readiness.md`; the model is
in `harness/coldread/coldread.yaml`, and a session has no spending cap. It needs the `claude` command-line tool,
signed in.

## Reverse: a design for a repository that already has code

A repository that existed before the method has a design; it is only not written down.
`cmd/reverse` writes it down, in steps that are each less mechanical than the last:

| Step | What it does | What it decides |
|---|---|---|
| `reverse survey` | Lists what a machine can see: the files and the globs that account for them, languages, test files and names, the command the repository proves itself with, the documents that may record decisions, CI, whether the history follows a commit convention. | Nothing. |
| `reverse skeleton` | Writes `design/DESIGN.md` with frontmatter that accounts for every file as it stands, so the checker passes at once, and a body whose every section is a question. | Nothing: `validate:` and `commits:` are written as comments. |
| `reverse draft` | One session, able to read the repository and to write nothing, drafts the whole document and the project's own section of `CLAUDE.md`, or of `AGENTS.md` where `CLAUDE.md` only points at it. The command writes its reply to disk, and only over a skeleton. | Nothing: it proposes. |
| a person | Reads every line. Answers each `(inferred)` and each `TODO(design)`. Switches on `validate:`. | Everything. |

`reverse prompt` prints what the drafting session is handed, to give to an agent of your
own instead.

The draft is told to write what the code proves. It can see what the code does; it usually
cannot see why, or what else was considered. So a reason no file states ends
`(inferred)`, a decision whose alternatives are recorded nowhere says
`Not recorded. TODO(design)`, and it never invents one. The checker counts the
`TODO(design)` marks on every run until none is left. Then ask a cold read for
comprehension: if a session with no memory can explain the repository from the document,
the design holds enough.

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
  everywhere, the harness, and the map. Its frontmatter lists the nested repositories:

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
  definition has merged, the checker in `billing` refuses until its frontmatter names a test
  for it:

  ```yaml
  serves:
    - name: billing.v1.CreateInvoiceRequest.currency
      test: TestAnInvoiceIsIssuedInItsCurrency
  ```

  Only what is merged binds, so the definition and its server can be built in either
  order: the server's test first, then the definition; or the definition as a draft
  pull request until the server is ready. Definitions are read from `.proto` files.
- **What the root's frontmatter asks of every repository is written into each.** A nested
  repository's `commits:` and `notes:` are its own lines, taken from the root's when it
  adopted the method. Nothing is inherited at check time, so a design always says what
  binds its repository.

`bootstrap` does this by itself: run on a repository that sits under a root, it copies no
harness and sets all of the above up, then prints the entry to add to the root's map.

## Commands

```
go -C harness run ./cmd/check                    # the checks, then the repository's own gate
go -C harness run ./cmd/check -validate=false    # the checks only
go -C harness run ./cmd/coldread -change "..."   # two fresh sessions read the design
go -C harness run ./cmd/coldread -only comprehension
go -C harness run ./cmd/reverse survey          # what a machine can see
go -C harness run ./cmd/reverse draft           # draft the design from the code
go -C harness run ./cmd/subject "fix: ..."       # is this subject a Conventional Commit
go -C harness run ./cmd/branch feat/a-thing      # is this name a change branch's here
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
