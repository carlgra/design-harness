# design-harness

A way of working on a codebase with agents in which the design is the source and the code
is derived from it, and the scripts that enforce it.

**The elephant never forgets.** Each repository has one design document,
`design/DESIGN.md`: what it is for, how it works, what must always hold, every decision in
force with its reason and what was rejected, and which files realise it. A checker refuses
any push in which the code and the document disagree.

**The goldfish remembers nothing.** A session with no memory is handed that document, and
only that document, and asked to explain the repository and to build a change from it.
What it cannot do, the document does not say.

Everyone who arrives later, a new colleague or tomorrow's agent session, is a goldfish.
The method is a way of making sure the elephant remembered what they will need.

## What is here

```
bootstrap            adopt the method in a project that already exists
harness/             everything that is copied into a project
  FRAMEWORK.md       the method, in full: read this first
  cmd/check          the checker: files accounted for, named tests exist, ids land, notes have their parts
  cmd/goldfish       two sessions with no memory read the design: comprehension, readiness
  cmd/elephant       reverse-engineer the design of existing code: survey, skeleton, draft
  cmd/subject        is a commit's subject a Conventional Commit (optional, per repository)
  safe-push          the one way a branch is pushed: check, push, open the pull request
  hooks/             pre-push and commit-msg, and their installer
  templates/         the design, a change note, CLAUDE.md, the CI workflow
  goldfish/          the two prompts and their settings
  elephant/          the reverse-engineering prompt and its settings
design/              this repository's own design, held to the same checker
```

## Adopt it in an existing project

You need git and the [Go toolchain](https://go.dev/dl/) (1.22 or later), whatever language
the project is written in. The goldfish and the elephant's draft also need the
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
go -C harness run ./cmd/elephant draft               # 1. draft the design from the code
$EDITOR design/DESIGN.md                             # 2. read all of it; answer every (inferred) and TODO(elephant)
go -C harness run ./cmd/goldfish -only comprehension # 3. can a stranger explain the repository from it?
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
./cmd/elephant prompt` prints the same instructions and survey for you to hand over.

Last, protect `main` on the server: require a pull request and the `harness` status check.

## Use it in a new project

The same command. An empty repository gets a skeleton with nothing to account for yet;
write the design first, then the code it describes.

## Day to day

```
go -C harness run ./cmd/check                 # does the code still agree with the design
cp harness/templates/change.md design/changes/$(date +%F)-what-changes.md
                                              # every change starts with a note: problem, plan, alternatives, files
go -C harness run ./cmd/goldfish -change-file design/changes/<the note>.md -out goldfish.md
                                              # when the design change is material: advice for whoever merges
harness/safe-push                             # check, push, open the pull request; a person merges
```

`harness/FRAMEWORK.md` is the whole method: how the document is written, what the machine
checks and deliberately does not, how a change lands, and why the goldfish advises and
never gates.

## Keeping a project's harness current

```
design-harness/bootstrap --update /path/to/your/project
```

replaces the project's `harness/` with this one and keeps its settings. The harness is
part of the project once copied: change it there if you need to, through the same door as
everything else.

## What it does not do

- It does not judge whether a design is good. The checker proves four small things and is
  always right about them; everything else is read by a person.
- It does not let a model gate a merge, or admit anything. The goldfish advises, the
  elephant proposes, a person decides.
- It does not open pull requests anywhere but GitHub; elsewhere, `harness/safe-push
  --no-pr` pushes and you open it yourself.
- The headless sessions are Claude's command-line tool. The prompts are plain files
  beside the settings, and `elephant prompt` exists so that any agent can do the drafting.

## Licence

MIT. See `LICENSE`.
