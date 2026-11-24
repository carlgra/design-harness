---
# The spine: the part of the design a machine reads. Everything under the
# second line of dashes is read by people, and by sessions with no memory.
#
# The code this design accounts for. Every source file must be under one of
# these; the checker refuses one that is not.
files:
  - harness/**
  - bootstrap
  - .github/**
# Tests that must exist: each walks a path a fault would travel. The checker
# proves each is there; the gate below is what runs them.
tests:
  - TestEveryFileIsAccountedForOrRefused
  - TestANamedTestMustExist
  - TestAnIdIsDefinedOnceAndEveryCitationLands
  - TestADesignNoteCarriesItsParts
  - TestADesignSaysWhatItsNotesCarry
  - TestAnUnfinishedDesignIsSaidToBeUnfinished
  - TestAWorktreeIsNotANestedRepository
  - TestASessionIsHandedItsPromptTheDesignAndNothingElse
  - TestAVerdictIsReadFromTheLastLineAndNothingElseCounts
  - TestASessionIsStartedWithExactlyTheToolsItIsGiven
  - TestTheSurveySaysWhatIsThereAndDecidesNothing
  - TestASkeletonsSpineAccountsForEveryFileAndSwitchesNothingOn
  - TestADraftIsReadAsTheDesignAndTheRules
  - TestADraftNeverReplacesADesignSomeoneWrote
  - TestTheProjectsRulesReplaceOnlyTheirOwnSection
  - TestBootstrappingAnExistingProjectLeavesItPassingAndOverwritesNothing
  - TestASubjectThatIsNotConventionalIsRefused
  - TestTheConventionBindsOnlyARepositoryWhoseDesignAsksForIt
# The repository's own gate, run from the root after the checks pass.
validate: go -C harness vet ./... && go -C harness test ./...
# A commit's subject, and so a pull request's title, is a Conventional Commit.
commits: conventional
# What a note under design/changes/ carries, from the date the rule began.
notes:
  from: 2026-11-24
  sections: [Problem, Plan, Alternatives rejected, Files]
---

# design-harness

This is the design: what this repository is for, how it works, what must always
hold, every decision in force with its reason and what was rejected, and which
files realise it. It is current state, edited in place; git is its history. A
change to the code starts with a note under `design/changes/` and ends with this
file saying what is now true.

## Scope

This repository is a method for working on a codebase with agents, and the scripts
that enforce it. The method: each repository has one design document, the code is
derived from it, a machine refuses a push in which the two disagree, and a session
with no memory is asked whether the document alone is enough to understand the
repository and to build a change.

It holds three things. `harness/` is everything a project needs to work this way: the
checker, the two model-driven commands, the push script, the hooks, the templates and
the method written out in full, `harness/FRAMEWORK.md`. `bootstrap` copies `harness/`
into a project that already exists and adds what is missing around it. `design/` is
this repository's own design, held to the same checker, so that the method is used on
itself. Beside them is the CI workflow that runs the checker here, a copy of the
template projects are given.

It holds no project's design but its own, and nothing that names a particular project.
It is not a library a project depends on: a project gets a copy and owns it (ADR-0005).

## How it works

Two names carry the method.

**The elephant** is `design/DESIGN.md`, the one thing in a repository that must not be
forgotten. It has a body, read by people and by model sessions, and a spine, its YAML
frontmatter, read by a machine. The spine says which paths hold the code (`files:`),
which tests must exist (`tests:`), the repository's own gate (`validate:`), whether a
commit's subject is a Conventional Commit (`commits:`), and what a change note must
carry (`notes:`). Six files at a repository's root need no entry in `files:`, because
they are how a repository works rather than its code: `README.md`, `CLAUDE.md`,
`AGENTS.md`, `LICENSE` (with or without `.md` or `.txt`), `.gitignore` and
`.gitattributes`. A glob ending `/**` is everything under a directory; any other is a
path in which `*` does not cross a `/`.

**The goldfish** is a model session with no memory, handed the document and nothing
else.

The parts, all under `harness/`:

- `internal/design` reads the spine and walks the repository for the files a design
  must account for. In a git repository that is git's view, tracked plus untracked and
  not ignored; it never enters `.git` or `design/`. A directory with a `.git` of its own
  is one of two things, and the walk enters neither: a nested repository, which has its
  own design, or a linked worktree, another checkout of this repository, whose files
  are this repository's already. A test file is recognised by its name: `_test.go`,
  `test_*.py` or `*_test.py`, `.test.` or `.spec.` in a JavaScript or TypeScript name,
  and the like for Rust, Java, Kotlin and Ruby. Every command starts here.
- `cmd/check`, the checker, loads the design and refuses four things: a file under no
  glob in `files:`; a name in `tests:` that appears in no test file the design accounts
  for; a decision or invariant id defined twice, or cited and defined nowhere; and a
  change note, dated from `notes: from:`, that lacks a section or leaves one empty.
  A decision is defined by a heading that opens with its id, `### ADR-0007 — …`; an
  invariant by a list item or heading that opens with its id, `- **INV-012** — …`;
  any other appearance is a citation, and one written `other-repository/ADR-0007` is
  that repository's and is not looked up. Then the checker runs `validate:`. It also
  counts the places the document still says `TODO(elephant)` outside backticks, the
  mark a draft leaves for a person, and reports the count without failing.
- `cmd/goldfish` puts two questions, comprehension and readiness, each to its own
  session. A session is the `claude` command-line tool in print mode, started with no
  tools and no persisted session, in the system's temporary directory so that it reads
  no project file. It is handed a prompt from `harness/goldfish/`, the design, and for
  readiness the change: whatever text it was given with `-change`, or the contents of
  the file named with `-change-file`, usually the change's note. The command reads
  `VERDICT: PASS` or `VERDICT: FAIL` from the last line of each reply, prints the
  replies and can write them as markdown for a pull request. It exits non-zero when a
  session fails, so that a person's own script can tell; nothing in the harness reads
  that.
- `cmd/elephant` writes down the design of a repository that has code and no design.
  `survey` lists what a machine can see. `skeleton` writes a `design/DESIGN.md` whose
  spine accounts for every file and whose body is questions. `draft` starts one session
  in the repository with the tools that read, `Read`, `Grep` and `Glob`, and no other,
  hands it `harness/elephant/reverse.md`, the survey and the shape to fill, and writes
  its reply to `design/DESIGN.md` and to the project's section of `CLAUDE.md`. `prompt`
  prints what that session would be handed. A skeleton is known by one line it carries,
  the comment `<!-- elephant:skeleton -->`; a draft does not carry it, so once a draft
  is written the document counts as someone's and a second draft needs `-force`. The
  project's section of `CLAUDE.md` is what lies between the comments
  `<!-- elephant:project:begin -->` and `<!-- elephant:project:end -->`. The method's
  own rules sit above it between `<!-- design-harness:begin -->` and
  `:end -->`, come from `templates/CLAUDE.md`, and are never written by a session.
- `internal/session` is the one place a model is started: the arguments, the settings
  file, the runner the tests replace.
- `cmd/subject` and `internal/convention` say whether a subject is a Conventional
  Commit, in a repository whose spine asks for them.
- `safe-push` is the one way a branch is pushed. It never pushes `main`, or a branch
  not named `feat/` or `fix/`: work it finds on one is moved, commits and uncommitted
  changes both, to a new branch named for the work, and that is what is pushed. It
  requires a clean tree; checks the
  branch merges cleanly; runs vet and the checker; checks the title; pushes with
  `--force-with-lease`; and opens the pull request on GitHub through `gh api`.
- `hooks/pre-push` refuses a push to `main` and runs the checker; `hooks/commit-msg`
  runs `cmd/subject`; `hooks/install` points `core.hooksPath` at them and refuses to
  switch off hooks already in use.
- `templates/` holds the shapes: the design, a change note, the adoption note,
  `CLAUDE.md`, and the CI workflow.

`bootstrap`, at the root, is POSIX shell. Given a project, new or old, it copies
`harness/` in, then writes `CLAUDE.md` or adds the method's rules to the end of one
that exists, the CI workflow, then the skeleton and the first note, then installs the
hooks, and runs the checker. Each of those is skipped when the thing is already there,
and a `harness/` directory that is not this harness stops it before anything is
written. `--update` is the one thing that replaces: it swaps a project's `harness/`
for this one, whole, keeping only the project's two settings files,
`goldfish/goldfish.yaml` and `elephant/elephant.yaml`. A change a project made to its
own copy is lost from the working tree by that and shows in `git diff`, where the
person updating decides what to keep.

## Principles

- **The design is current state; git is the history.** This document says what is
  decided now, is edited when that changes with the reason in the commit, and keeps
  each decision's id for life so that citations land (ADR-0001).
- **A machine checks little, and is always right about it.** Whatever cannot be
  decided without judgment is read by a person (ADR-0002).
- **A model proposes and advises; a person decides.** No session's output gates a
  merge, and none replaces what a person wrote unless that person forces it (ADR-0003,
  ADR-0004).
- **Adopting takes nothing away.** It overwrites no file and replaces no hook a
  project already uses. What it adds that can refuse a push is the refusal to push
  `main`, and the checker, against a spine written so that it passes; the project's
  own gate is left for a person to switch on (ADR-0004, ADR-0006).
- **What a project gets is whole.** Nothing in `harness/` is needed from outside it
  (ADR-0005).

## What must always hold

- **INV-001** — Design before code: no source file exists that the design does not
  account for. *(the checker; `test: TestEveryFileIsAccountedForOrRefused`)*
- **INV-002** — A test the design names exists, as a whole word, in a test file the
  design accounts for. *(`test: TestANamedTestMustExist`)*
- **INV-003** — A decision or invariant id is defined once in a design, and every id
  it cites as its own is defined in it; another repository's, qualified by its name,
  is not this document's to define. *(`test: TestAnIdIsDefinedOnceAndEveryCitationLands`)*
- **INV-004** — A change note dated from the spine's `notes: from:` carries every
  section the spine asks of a note, each with something under it; a note dated before
  is never reread. *(`test: TestADesignNoteCarriesItsParts`,
  `test: TestADesignSaysWhatItsNotesCarry`)*
- **INV-005** — A goldfish is handed its prompt, the design and, for readiness, the
  change, and nothing else; it is started with no tools. *(`test:
  TestASessionIsHandedItsPromptTheDesignAndNothingElse`, `test:
  TestASessionIsStartedWithExactlyTheToolsItIsGiven`; what the `claude` tool itself
  adds to a session is read at review)*
- **INV-006** — The goldfish command judges nothing: a verdict is the last line of a
  reply in the exact form, anything else is a failure, and no verdict is read by any
  other command. *(`test: TestAVerdictIsReadFromTheLastLineAndNothingElseCounts`)*
- **INV-007** — The session that drafts a design can read and cannot write. Only the
  `elephant` command writes, only `design/DESIGN.md` and the marked section of
  `CLAUDE.md`, and never over a design that is no longer a skeleton unless forced.
  *(`test: TestASessionIsStartedWithExactlyTheToolsItIsGiven`, `test:
  TestADraftNeverReplacesADesignSomeoneWrote`, `test:
  TestTheProjectsRulesReplaceOnlyTheirOwnSection`, `test:
  TestADraftIsReadAsTheDesignAndTheRules`)*
- **INV-008** — A survey and a skeleton decide nothing: the spine a skeleton writes
  accounts for every file as it stands, names no test, and leaves the gate and the
  commit convention as comments, the convention excepted when the history already
  follows it. *(`test: TestTheSurveySaysWhatIsThereAndDecidesNothing`, `test:
  TestASkeletonsSpineAccountsForEveryFileAndSwitchesNothingOn`)*
- **INV-009** — Adopting with bootstrap overwrites nothing that exists, adding to the
  end of a `CLAUDE.md` that is already there and leaving every line of it as it was,
  and commits nothing; and a project it has run on passes the checker with its own copy of the
  harness alone. With `--update`, and only then, it replaces `harness/`, and nothing
  else.
  *(`test: TestBootstrappingAnExistingProjectLeavesItPassingAndOverwritesNothing`)*
- **INV-010** — A design that still says `TODO(elephant)` is reported as unfinished on
  every run of the checker, and is not refused for it. *(`test:
  TestAnUnfinishedDesignIsSaidToBeUnfinished`)*
- **INV-011** — Nothing reaches `main` except through a pull request that passed the
  checker. *(`harness/hooks/pre-push` and `harness/safe-push` refuse the push; the
  server's branch protection is read at review)*
- **INV-012** — Nothing under `harness/` needs anything outside it, or names a
  particular project. One test looks outside: `bootstrap_test.go` looks for `bootstrap`
  beside `harness/`, runs it where it finds it, which is here, and skips where it does
  not, which is in every project's copy. *(read at review; INV-009's test proves a copy
  works alone)*
- **INV-013** — The commit convention binds only a repository whose spine asks for it.
  *(`test: TestTheConventionBindsOnlyARepositoryWhoseDesignAsksForIt`, `test:
  TestASubjectThatIsNotConventionalIsRefused`)*

## What no machine proves yet

Each rule below is held by a person reading, with what would prove it and what makes
that due. None is accepted as final.

| Gap | Today | Destination | Trigger |
|---|---|---|---|
| INV-005 | the command hands a session only the prompt, the design and for readiness the change, and runs it away from the project; the `claude` tool may still load a person's own global instructions and memory into it | a start-up mode of the tool that loads nothing and still uses the stored login, named in `internal/session`, with the test extended to pin it | the tool offering one |
| INV-011 | the hook and the push script refuse a push to `main`; protection on the server is set by hand and nothing re-reads it | the checker reads the repository's protection through `gh api` and fails if the pull request requirement or the `harness` check is off | the first project that asks for it |
| INV-012 | read at review | a test that fails on any `..` path or absolute path in a script, and on a name from a list of project names | the first slip |
| `safe-push` | exercised by use; no test walks its refusals | a shell test against a throwaway remote: `main` refused, a stray branch moved, a dirty tree refused, a conflict named | the next change to the script |

## How a change lands

A change starts with a note under `design/changes/<date>-<slug>.md`: the
problem, the plan, the alternatives rejected, and the files to change and why.
Work happens on a `feat/` or `fix/` branch, never on `main`. `harness/safe-push`
runs the checker and the repository's own gate, pushes the branch and opens the
pull request; CI runs the checker again, and `main` is protected behind it. A
person merges. The change ends with this document saying what is now true.

A change to `harness/` is a change to what every project that updates will get, so its
note says what a project that already has the harness must do, or that it need do
nothing.

## Decisions

One decision each, with its reason and what was rejected. The ids are kept so
that a citation always lands. A decision states a constraint, never a value:
where it depends on a number or a string, it says what the value must satisfy
and names the file the value lives in.

### ADR-0001 — One design document per repository, and it is current state

A repository's design is one file, `design/DESIGN.md`, holding purpose, mechanism,
invariants, decisions and the map of files. It is edited in place when something
changes, with the reason in the commit, and git is its history. A decision keeps its id
for life.

One file because the document's reader is handed all of it at once, a goldfish most of
all, and a design spread over many files is one nobody can be handed. Current state
because a reader should never have to work out which of two statements still holds.

#### Alternatives rejected

- **One file per decision, with a status and a superseded-by chain**, the usual ADR
  directory. The project this method was drawn from worked that way first. Reading the
  design meant replaying its history, an index had to be kept in step by hand, and a
  session handed the directory spent its attention on decisions no longer in force.
- **A wiki or documents outside the repository.** They cannot be changed in the same
  pull request as the code, so they drift, and a checker cannot hold a push to them.

### ADR-0002 — The checker proves a few small things, and refuses on each

The machine checks only what it can be right about every time: files accounted for,
named tests present, ids that land, notes with their parts. Each is a refusal with a
line saying what and where, and there are always two fixes: amend the design or change
the code. What the spine does not state, the checker does not check.

A check that is sometimes wrong gets switched off, and takes the trustworthy ones with
it. The value of the small check is that a FAIL is never argued with.

#### Alternatives rejected

- **Lint the prose**: require every decision to have a reason of some length, every
  invariant a proof in a fixed grammar. It measures form, is satisfied by filler, and
  refuses honest documents that are shaped differently.
- **Have a model judge the design in CI.** See ADR-0003.
- **Fail on `TODO(elephant)`.** The adoption pull request is itself pushed through the
  checker, with the marks still in it; refusing them would make the method impossible
  to adopt through its own door. They are counted and reported instead (INV-010).

### ADR-0003 — The goldfish advises; it never gates

Two sessions with no memory and no tools are handed the design and one question each.
Their verdicts are printed and written for the pull request, for the person who merges.
Nothing reads a verdict to allow or refuse anything. The command is run by hand when a
change to a design is material. The model, the most a session may spend in dollars and
the form of the verdict line are values in `harness/goldfish/goldfish.yaml`; the questions are the two prompt files
beside it.

No tools and no memory because the question is whether the document is enough, and a
session that can read the code answers a different question. Two questions because
they fail differently: a document can describe a repository well and still leave a
particular change to guesswork.

#### Alternatives rejected

- **A required status check.** The project this method was drawn from ran it as a gate
  first. A model's verdict is not repeatable: the same design passed and failed on
  consecutive runs, each run cost money, and a gate that can be re-rolled until it
  passes proves nothing while a person waits on it.
- **A session with the repository to read.** It passes designs that say too little,
  because it fills the gaps from the code.
- **One question.** Comprehension alone passes a design that explains the present and
  is silent on what a change must respect.

### ADR-0004 — A design is reverse-engineered in steps, and a person takes it on

For a repository with code and no design: a machine surveys, a model session drafts,
a person confirms. The survey reports and decides nothing. The drafting session is
given the tools that read and no others, and the command, not the session, writes its
reply. The session is told to mark what it could not find in the repository, in two
ways. A reason no file states ends `(inferred)`: a claim to be confirmed. Something it
could not know at all is written `TODO(elephant)` with the question, and alternatives
recorded nowhere are written `Not recorded. TODO(elephant)` rather than invented. A draft is written only over a skeleton; a design a
person has edited is replaced only with `-force`. The gate and the commit convention
are never switched on by a survey, the convention excepted when the history already
follows it. The model and budget are values in `harness/elephant/elephant.yaml`; what
the session is told is `harness/elephant/reverse.md`.

A wrong draft is faster to correct than a blank page is to fill, so the draft is worth
having; but a design nobody has read is a guess. So the questions, the `TODO(elephant)`
marks, are counted by the checker on every run until a person has answered each, and
the `(inferred)` claims stay in the text for that person to confirm or correct as they
read; the checker does not count those, because a confirmed inference and an
unconfirmed one look the same to a machine.

#### Alternatives rejected

- **Let the session edit the repository itself.** It would then be trusted with every
  file in order to write two, and a design could be overwritten without the command
  being able to refuse.
- **Derive the design mechanically**, from the tree and the tests alone. That produces
  the spine and a list of files; the decisions and the invariants are the part worth
  having, and they are not in the tree.
- **Require the design to be written by hand before adopting.** The repository goes
  unchecked for as long as that takes, which in practice is for good.
- **Switch on the gate the survey found.** A `validate:` that fails today refuses every
  push from the moment it is written, including the one that adopts the method.

### ADR-0005 — A project gets a copy of the harness, in Go, and owns it

`bootstrap` copies `harness/` into the project, where it is a Go module of its own
that needs nothing outside its directory and has one dependency, a YAML parser.
The commands are run with `go -C harness run`, so the checker, the hooks and the push
need the Go toolchain and git and nothing else, whatever language the project is
written in. Two things need more, and say so when it is missing: opening a pull request
needs GitHub's `gh`, and the goldfish and the elephant's draft need the `claude`
command-line tool. The scripts set `GOWORK=off` so that
a project's own Go workspace does not claim the module. Updating is running
`bootstrap --update`, which replaces the copy and keeps the project's settings.

A copy because the hook that guards every push must work offline, from a fresh clone,
at the version the project last reviewed; and because a project should be able to read
and change what refuses its pushes.

#### Alternatives rejected

- **A released binary or `go run module@version`.** Every push would depend on a
  network and on this repository staying where it is, and the version in force would
  be a string in a script rather than code in the pull request that changed it.
- **A git submodule.** Fresh clones and worktrees arrive without it, and the hook
  fails exactly when someone new starts.
- **Python or shell for the checker.** Shell cannot parse the spine safely; Python
  brings an interpreter version and a package environment into every hook. What is
  shell, `bootstrap`, `safe-push` and the three hooks, parses nothing: each only
  drives git, `gh` and the Go commands.

### ADR-0006 — Nothing reaches main except through a pull request, and there is one door

Work is pushed with `harness/safe-push`, which never pushes `main` or any branch not
named `feat/` or `fix/`, moving work found on one to a branch of its own; runs the
checker, and with it the repository's gate, as CI will; and opens the pull request. The `pre-push` hook refuses a push to `main` however it is issued and runs the
checker. A person merges. `hooks/install` refuses to replace hooks a repository already
uses, and says how to call these from them.

One door because an agent that is told "never push to main" forgets, and one that can
only push through a script that refuses does not need to remember.

#### Alternatives rejected

- **Rely on branch protection alone.** It refuses at the server, minutes later, after
  the wrong thing has been pushed; and it is a setting someone can turn off.
- **Let the hook open the pull request.** A hook that reaches a server on every push is
  a hook people disable.
- **Install the hooks over whatever is there.** `core.hooksPath` switches off every
  hook outside it without a word.

### ADR-0007 — The commit convention is asked for by the design, repository by repository

A repository whose spine says `commits: conventional` has every commit's subject, and
its pull request's title, checked as a Conventional Commit: by the `commit-msg` hook as
it is written and by `safe-push` before it leaves. One that does not say so is bound by
nothing. The types accepted are a value in `harness/internal/convention`, and include
`design` for a change to what is decided.

Opt-in because the convention pays only where a changelog or a release is generated
from subjects, and a project adopting the method should not have its history refused.

#### Alternatives rejected

- **Always on.** Adoption would then mean rewriting how a team commits before it could
  push its first design.
- **Left to a separate tool.** The hook path is already the harness's, and a second
  tool would need the same care about not replacing it.

### ADR-0008 — A change starts with a note in four parts, from a date

A change to code starts with `design/changes/<date>-<slug>.md`: the problem, the plan,
the alternatives rejected, the files. The checker requires each section of every note
dated from the spine's `notes: from:`, with something under it. A repository asks more
of its notes by listing more sections in its spine.

The note is where the alternatives are written while someone still remembers them. The
date is there because a rule binds what is written after it exists: rereading old notes
against new rules turns every new rule into an afternoon of editing history.

#### Alternatives rejected

- **Put the reasoning in the pull request.** It stays on the server, outside the
  repository, and is not there for the next session to read.
- **Require a new decision for every change.** Most changes are within the decisions in
  force; a record for each buries the ones that decide something.

## Files

| File | Purpose |
|---|---|
| `bootstrap` | Adopts the method in a project: copies `harness/`, then adds what is missing and overwrites nothing (INV-009). `--update` replaces a project's harness and keeps its settings. |
| `harness/FRAMEWORK.md` | The method, written for whoever works in a project that uses it. Its presence is how `bootstrap` knows a `harness/` is this one. |
| `harness/go.mod`, `harness/go.sum` | The harness's own module, `harness`, with one dependency (ADR-0005). |
| `harness/internal/design/` | Reads the spine, walks the repository, finds the root and the harness, matches globs, recognises test files. |
| `harness/internal/session/` | Starts a model session through the `claude` tool with exactly the tools it is given; reads a settings file. |
| `harness/internal/convention/` | Whether a subject is a Conventional Commit, and whether a repository asks for them (ADR-0007). |
| `harness/cmd/check/` | The checker (ADR-0002): `checks.go` holds files, tests and ids; `notes.go` the notes; `main.go` runs them and then `validate:`. |
| `harness/cmd/goldfish/` | The two questions (ADR-0003): composes what a session is handed, reads the verdict line, prints and writes the replies. |
| `harness/cmd/elephant/` | Reverse-engineering (ADR-0004): `survey.go` is the survey and the skeleton; `main.go` the four steps, the reply's two parts, and the writes. `bootstrap_test.go` runs the real `bootstrap`. |
| `harness/cmd/subject/` | The command the hook and the push script call for ADR-0007. |
| `harness/goldfish/` | `comprehension.md` and `readiness.md`, the two prompts; `goldfish.yaml`, the model, verdict prefix and budget. |
| `harness/elephant/` | `reverse.md`, what a drafting session is told; `elephant.yaml`, the model and budget. |
| `harness/templates/` | `DESIGN.md`, the shape of a design, with the placeholders a skeleton fills; `change.md`, a note; `adopt.md`, the first note; `CLAUDE.md`, the rules for an agent; `harness.yml`, the CI workflow. |
| `harness/safe-push` | The one door (ADR-0006). |
| `harness/hooks/` | `pre-push`, `commit-msg`, and `install`. |
| `.github/workflows/harness.yml` | This repository's CI: vet, the harness's tests, the checker. A copy of the template. |

Paths a fault would travel, and the test that walks each:

- a source file the design does not account for is refused — `TestEveryFileIsAccountedForOrRefused`
- a test the design names that does not exist is refused — `TestANamedTestMustExist`
- an id defined twice, or cited and defined nowhere, is refused — `TestAnIdIsDefinedOnceAndEveryCitationLands`
- a note without one of its parts is refused, and an old note is not reread — `TestADesignNoteCarriesItsParts`
- a section a repository adds to its notes is required from its date — `TestADesignSaysWhatItsNotesCarry`
- an unfinished design is reported and not refused — `TestAnUnfinishedDesignIsSaidToBeUnfinished`
- another checkout of a repository is not taken for its files — `TestAWorktreeIsNotANestedRepository`
- a goldfish is handed the design and the change and nothing else — `TestASessionIsHandedItsPromptTheDesignAndNothingElse`
- a reply that does not end in the verdict form is not a pass — `TestAVerdictIsReadFromTheLastLineAndNothingElseCounts`
- a session is started with the tools it was given and no other — `TestASessionIsStartedWithExactlyTheToolsItIsGiven`
- a survey invents nothing and misses no file — `TestTheSurveySaysWhatIsThereAndDecidesNothing`
- a skeleton passes the checker and switches nothing on — `TestASkeletonsSpineAccountsForEveryFileAndSwitchesNothingOn`
- a reply that is not a design is refused, not half written — `TestADraftIsReadAsTheDesignAndTheRules`
- a draft does not replace a design a person wrote — `TestADraftNeverReplacesADesignSomeoneWrote`
- a project's own `CLAUDE.md` keeps everything outside the marked section — `TestTheProjectsRulesReplaceOnlyTheirOwnSection`
- bootstrap leaves a project passing, overwrites nothing, and is safe to run twice — `TestBootstrappingAnExistingProjectLeavesItPassingAndOverwritesNothing`
- a subject that is not conventional is refused — `TestASubjectThatIsNotConventionalIsRefused`
- a repository that has not asked for the convention is not bound by it — `TestTheConventionBindsOnlyARepositoryWhoseDesignAsksForIt`
