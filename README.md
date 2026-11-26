# design-harness

A way of working on a codebase with agents in which the design is the source and the code
is derived from it, and the scripts that enforce it.

**The design is the repository's memory.** Each repository has one design document,
`design/DESIGN.md`: what it is for, how it works, what must always hold, every decision in
force with its reason and what was rejected, and which files realise it. A checker refuses
any push in which the code and the document disagree.

**A cold read tests it.** A session with no memory is handed that document, and only that
document, and asked to explain the repository and to build a change from it. What it
cannot do, the document does not say.

Everyone who arrives later, a new colleague or tomorrow's agent session, starts with no
memory too. The method is a way of making sure the design holds what they will need.

## What a project looks like once it has adopted this

**One repository.** The design is one file, and the top of that file is the part a
machine reads:

```
your-project/
  design/
    DESIGN.md          the design of this repository: its spine, its rules, its decisions
    changes/           one short note per change: problem, plan, alternatives, files
  harness/             the checker, the scripts and the hooks, copied in
  CLAUDE.md            the method's rules for an agent, then the project's own
  src/ ...             the code, every file of it under a path the spine lists
```

`design/DESIGN.md` has two parts.

- **The spine** is its YAML frontmatter: which paths hold the code, which tests must
  exist, the command that is the repository's own gate. The checker reads it and refuses
  a push in which a source file is under no path it lists, or a test it names does not
  exist.

  ```yaml
  ---
  files:
    - src/**
    - package.json
  tests:
    - TestAnEmptyCartIsRefused
  validate: npm test
  ---
  ```

- **The body** is the rules for that repository, for people and for agents: what it is
  for, how it works, and then

  - *What must always hold*: the invariants, each with an id it keeps for life and what
    proves it.

    ```
    - **INV-004** — An empty cart is refused at checkout, never charged.
      *(test: TestAnEmptyCartIsRefused)*
    ```

  - *Decisions*: each one in force, `ADR-0007` and so on, with its reason and what was
    rejected.
  - *What no machine proves yet*: every rule that is still held only by a person
    reading, and what would prove it.

**Several repositories.** Each component is a repository of its own, with its own
`design/DESIGN.md`: its own spine and its own rules. They are checked out side by side
inside a project folder, the root, which is a repository too and holds what binds all of
them. The definitions they talk through, gRPC with protobuf, live in one repository
beside them and nowhere else.

```
project/                     the root: a repository
  design/DESIGN.md           the rules every repository below is bound by, and the map of them
  harness/                   the one harness; each repository below runs it as ../harness
  contracts/                 a repository: the definitions between components
    design/DESIGN.md         its spine and rules
    design/changes/          a note per change, naming who serves each new RPC or field
    billing/v1/billing.proto
    ledger/v1/ledger.proto
  billing/                   a repository: one component
    design/DESIGN.md         its spine and rules; cites the root's as project/ADR-0004
    design/changes/
    ...
  ledger/                    a repository: another
    design/DESIGN.md
    ...
```

A rule that binds one component is in that component's design. A rule that binds all of
them is in the root's, and is cited from the others, not copied. An interface between
two components is a definition in `contracts`, and the checker holds each one to having
a component that serves it and a test there that proves it. How to set this up is under
[Several repositories](#several-repositories-services-beside-a-contracts-repository)
below.

## What is here

```
bootstrap            adopt the method in a project that already exists
harness/             everything that is copied into a project
  FRAMEWORK.md       the method, in full: read this first
  cmd/check          the checker: files accounted for, named tests exist, ids land, notes have their parts
  cmd/coldread       two sessions with no memory read the design: comprehension, readiness
  cmd/reverse       reverse-engineer the design of existing code: survey, skeleton, draft
  cmd/subject        is a commit's subject a Conventional Commit (optional, per repository)
  cmd/sync           at a root: clone and fetch the repositories its design lists
  safe-push          the one way a branch is pushed: check, push, open the pull request
  hooks/             pre-push and commit-msg, and their installer
  templates/         the design, a change note, CLAUDE.md, the CI workflow
  coldread/          the two prompts and their settings
  reverse/           the reverse-engineering prompt and its settings
design/              this repository's own design, held to the same checker
```

## Adopt it in an existing project

You need git and the [Go toolchain](https://go.dev/dl/) (1.22 or later), whatever language
the project is written in. The cold read and the drafting session also need the
[`claude` command-line tool](https://claude.com/claude-code), signed in. Pull requests are
opened with `gh`.

```
git clone <this repository> design-harness
design-harness/bootstrap /path/to/your/project
```

Bootstrap copies `harness/` into the project and adds only what is missing: a
`design/DESIGN.md` whose frontmatter accounts for every file as it stands and whose every
section is still a question, the first change note, the method's rules in `CLAUDE.md`
(after whatever is already there), a CI workflow, and the git hooks. It overwrites nothing
and commits nothing. When it finishes, the checker passes.

Then, in the project:

```
go -C harness run ./cmd/reverse draft               # 1. draft the design from the code
$EDITOR design/DESIGN.md                             # 2. read all of it; answer every (inferred) and TODO(design)
go -C harness run ./cmd/coldread -only comprehension # 3. can a stranger explain the repository from it?
git checkout -b feat/adopt-the-design                # 4. commit, and push through the one door
git add -A && git commit -m "design: this repository has a design, and a machine holds the code to it"
harness/safe-push
```

Step 1 is one model session that can read the repository and write nothing. It is told to
write what the code proves: a reason it had to guess ends `(inferred)`, and where the
alternatives to a decision are recorded nowhere it says so rather than inventing them.
Step 2 is the one that matters. A draft becomes the design when a person has read it and
answered each mark, and the checker counts the marks on every run until none is left.

To use an agent of your own instead of the headless session, `go -C harness run
./cmd/reverse prompt` prints the same instructions and survey for you to hand over.

Last, protect `main` on the server: require a pull request and the `harness` status check.
On GitHub, as an administrator of the repository:

```
go -C harness run ./cmd/protection -set -n   # what it would send
go -C harness run ./cmd/protection -set      # add what is lacking, change nothing else
```

Then put `protection: required` in the frontmatter of `design/DESIGN.md`, and every
`safe-push` reads the setting again and refuses while it lacks something.

## Several repositories: services beside a contracts repository

The usual shape is a project folder that is itself a repository, the root, with each
domain service checked out inside it as a repository of its own, and a contracts
repository beside them holding the definitions they talk through.

```
design-harness/bootstrap ~/code/project             # the root first: it gets the harness
design-harness/bootstrap ~/code/project/contracts   # then each repository inside it
design-harness/bootstrap ~/code/project/billing
```

Bootstrap sees that `billing` sits under a root and adopts it as nested. It copies no
second harness: `billing` runs the root's as `../harness`. Its `CLAUDE.md` says to read
the root's rules first, its design skeleton takes the root's commit convention and note
sections and says how the root's decisions are cited (`project/ADR-0004`), and its CI
checks the root out around it. The drafting session for `billing` is handed the root's
design, and told that the definitions live in `contracts`. Bootstrap then prints the
entry to add to the root's map, the `repos:` list in its design:

```yaml
repos:
  - path: contracts
    kind: contracts
    remote: git@github.com:acme/contracts.git
  - path: billing
    kind: service
    remote: git@github.com:acme/billing.git
```

With the map in place the root's checker refuses a repository on disk that is not on it,
a nested repository's checker refuses a citation the root's design does not define, and
a definition is held to having someone who serves it. In `contracts`, a new RPC or
request field is refused until the change's note lists it in a `servers` block with the
repository that serves it and the ones that call it. Once it has merged, that
repository's checker refuses until its design names, under `serves:`, a test for it.
Also,
`go -C harness run ./cmd/sync` rebuilds the whole tree on a fresh machine. The root pins
no commits and uses no submodules. `--standalone` gives a repository under a root a
harness of its own instead.

## Use it in a new project

The same command. An empty repository gets a skeleton with nothing to account for yet;
write the design first, then the code it describes.

## Day to day

```
go -C harness run ./cmd/check                 # does the code still agree with the design
cp harness/templates/change.md design/changes/$(date +%F)-what-changes.md
                                              # every change starts with a note: problem, plan, alternatives, files
go -C harness run ./cmd/coldread -change-file design/changes/<the note>.md -out coldread.md
                                              # when the design change is material: advice for whoever merges
harness/safe-push                             # check, push, open the pull request; a person merges
```

`harness/FRAMEWORK.md` is the whole method: how the document is written, what the machine
checks and deliberately does not, how a change lands, and why a cold read advises and
never gates.

## Keeping a project's harness current

```
design-harness/bootstrap --update /path/to/your/project
```

replaces the project's `harness/` with this one and keeps its settings, and rewrites the
method's rules in its `CLAUDE.md`, between their two marks, leaving every other line of
that file alone. In a repository nested under a root it refreshes the rules only; the
harness is updated at the root. The harness is
part of the project once copied: change it there if you need to, through the same door as
everything else.

## What it does not do

- It does not judge whether a design is good. The checker proves four small things and is
  always right about them; everything else is read by a person.
- It does not let a model gate a merge, or admit anything. A cold read advises, a
  draft proposes, a person decides.
- It reads definitions from protobuf only. A contracts repository in another format has
  its notes held to the `servers` block and nothing scanned.
- It does not open pull requests anywhere but GitHub; elsewhere, `harness/safe-push
  --no-pr` pushes and you open it yourself.
- The headless sessions are Claude's command-line tool. The prompts are plain files
  beside the settings, and `reverse prompt` exists so that any agent can do the drafting.

## Licence

MIT. See `LICENSE`.
