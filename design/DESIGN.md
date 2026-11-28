---
# The frontmatter: the part of the design a machine reads. Everything under the
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
  - TestASkeletonsFrontmatterAccountsForEveryFileAndSwitchesNothingOn
  - TestADraftIsReadAsTheDesignAndTheRules
  - TestADraftNeverReplacesADesignSomeoneWrote
  - TestTheProjectsRulesReplaceOnlyTheirOwnSection
  - TestBootstrappingAnExistingProjectLeavesItPassingAndOverwritesNothing
  - TestASubjectThatIsNotConventionalIsRefused
  - TestTheConventionBindsOnlyARepositoryWhoseDesignAsksForIt
  - TestARepositoryOnDiskIsOnTheMap
  - TestACitationOfTheRootLandsInTheRootsDesign
  - TestANestedRepositoryTakesItsRulesFromTheRoot
  - TestARootsSkeletonCarriesTheMap
  - TestARepositoryUnderARootUsesTheRootsHarnessAndRules
  - TestTheMapIsBroughtInLineWithoutMerging
  - TestAnUpdateReplacesTheHarnessAndTheRulesAndNothingOfTheProjects
  - TestAChangeBranchThatPassesIsPushed
  - TestWorkOnMainOrAStrayBranchIsMovedAndMainIsNeverPushed
  - TestABranchAlreadyMergedIsNotPushedToAgain
  - TestADirtyTreeIsRefused
  - TestABranchThatDoesNotMergeCleanlyIsRefused
  - TestABranchTheCheckerRefusesIsNotPushed
  - TestATitleThatIsNotConventionalIsRefusedWhereTheDesignAsks
  - TestTheProtoScanFindsRPCsAndRequestFields
  - TestTheServersBlockIsReadFromItsFence
  - TestARepositorysRoleIsReadFromTheRootsMap
  - TestANewDefinitionIsListedWithItsServer
  - TestAListedDefinitionIsRefusedUntilItsServerNamesATest
  - TestAPlannedEntryBindsNoServerUntilItsDefinitionMerges
  - TestAProtectionThatLacksSomethingIsNamedForIt
  - TestAProtectionThatCannotBeReadIsNotTakenForNone
  - TestSettingAddsWhatIsLackingAndChangesNothingElse
  - TestProtectionIsReadOnlyWhereTheDesignAsks
  - TestAPushIsRefusedWhileTheServersProtectionLacksSomething
  - TestADesignDividedIntoAreasPasses
  - TestAnAreaHasASpineAndASpineHasAnArea
  - TestAFileIsUnderOneSpine
  - TestASpinesTestIsInItsOwnArea
  - TestACitationAcrossDocumentsLandsInTheOneNamed
  - TestAnAreasNameCanQualifyAnId
  - TestTheDocumentsToReadAreTheDesignAndTheAreasSpine
  - TestAnAreaIsReadWithTheDesignAndWithoutTheOtherAreas
  - TestASpinesSkeletonIsWrittenOnlyForAnAreaTheDesignDefines
  - TestASpineIsDraftedFromItsAreaAndTheDesignAlone
  - TestADraftedSpineReplacesOnlyASkeleton
  - TestEveryAreaADraftDefinesIsGivenASkeleton
  - TestTheMarksLeftForAPersonAreCountedOneWay
# The repository's own gate, run from the root after the checks pass.
validate: go -C harness vet ./... && go -C harness test ./...
# A commit's subject, and so a pull request's title, is a Conventional Commit.
commits: conventional
# The server's protection of main is read before every push, and a push is
# refused while it lacks a pull request, the harness check or the refusal of
# a force push.
protection: required
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

This repository is a tool for writing down a codebase's design and architecture and
enforcing them, made for codebases that agents work on: a method, and the scripts that
enforce it. A design holds what must be governed and nothing else (ADR-0014). The method: each repository has one design document, the code is
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
It is not a library a project depends on: a project gets a copy and owns it (ADR-0005),
or, where it is one of several repositories under a root, uses the root's copy
(ADR-0009).

## How it works

Two things carry the method.

**The design** is `design/DESIGN.md`, the one thing in a repository that must not be
forgotten. It has a body, read by people and by model sessions, and YAML frontmatter,
read by a machine. The frontmatter says which paths hold the code (`files:`),
which tests must exist (`tests:`), the repository's own gate (`validate:`), whether a
commit's subject is a Conventional Commit (`commits:`), whether the server's protection
of the default branch is read before a push (`protection:`), and what a change note
must carry (`notes:`). Six files at a repository's root need no entry in `files:`, because
they are how a repository works rather than its code: `README.md`, `CLAUDE.md`,
`AGENTS.md`, `LICENSE` (with or without `.md` or `.txt`), `.gitignore` and
`.gitattributes`. A glob ending `/**` is everything under a directory; any other is a
path in which `*` does not cross a `/`.

**A spine** is the design of one area of a repository too large for one document
(ADR-0013). The design's frontmatter divides the code into areas under `spines:`, each a
`name:` and the `files:` it holds, and each area has a document,
`design/spines/<name>.md`, with the same sections as the design and the rules of that
area alone: its invariants, its decisions, its files. Its frontmatter has one key,
`tests:`, the tests that must exist in the area's own test files. The design then holds
only what binds every area. Whoever changes an area reads the design and that area's
spine, and none of the others; `check -for <path>...` prints which documents those are
for the paths given. Each document numbers its own ids. Within a document an id is
cited bare; an area's is cited from anywhere else as `checkout/ADR-0003`, and the
design's is cited from a spine as `<repository>/ADR-0004`. A repository with no
`spines:` has no areas and is one document, as before. `coldread -spine <area>` hands
its sessions the design and then that area's spine. `reverse skeleton -spine <area>`
writes a spine's skeleton for an area the design already defines, and `reverse draft
-spine <area>` has one session read that area, handed the design and a survey of the
area's files alone, and writes its reply as the spine, only over a skeleton. A draft of
the design itself that defines areas has each one's skeleton written with it.

**A cold read** is a model session with no memory, handed the document and nothing
else.

The parts, all under `harness/`:

- `internal/design` reads the frontmatter and walks the repository for the files a design
  must account for. In a git repository that is git's view, tracked plus untracked and
  not ignored; it never enters `.git` or `design/`. Directories are walked on disk,
  ignored ones included, which is how a nested repository the root's `.gitignore` hides
  is still found. A directory with a `.git` of its own
  is one of two things, and the walk enters neither: a nested repository, which has its
  own design, or a linked worktree, another checkout of this repository, whose files
  are this repository's already. `go -C harness` moves a command's process into the harness, so a command takes
  the directory it was typed in from the shell's `$PWD`, and its repository is the
  nearest directory at or above that holding a design; the shell scripts find the
  harness from their own path and the repository from git. A test file is recognised
  by its name: `_test.go`,
  `test_*.py` or `*_test.py`, `.test.` or `.spec.` in a JavaScript or TypeScript name,
  and the like for Rust, Java, Kotlin and Ruby. Every command starts here.
- `cmd/check`, the checker, loads the design and refuses, in every repository, four
  things, and under a root three more that are described further down, the map, a
  citation of the root's, and the contracts rules: a file under no
  glob in `files:`; a name in `tests:` that appears in no test file the design accounts
  for; a decision or invariant id defined twice, or cited and defined nowhere; and a
  change note, dated from `notes: from:`, that lacks a section or leaves one empty.
  A note's date is the `YYYY-MM-DD` its file name opens with, and a section is a
  heading of any level whose title begins with the section's name, case and a leading
  "The" aside, so `## The problem` is the section `Problem`. A decision is defined by a heading that opens with its id, `### ADR-0007 — …`; an
  invariant by a list item or heading that opens with its id, `- **INV-012** — …`;
  any other appearance is a citation, and one written `other-repository/ADR-0007` is
  that repository's and is not looked up. Then the checker runs `validate:`. It also
  counts the places the document still says `TODO(design)` outside backticks, the
  mark a draft leaves for a person, and reports the count without failing.
- `cmd/coldread` puts two questions, comprehension and readiness, each to its own
  session. A session is the `claude` command-line tool in print mode, started with no
  tools and no persisted session, in the system's temporary directory so that it reads
  no project file. It is handed a prompt from `harness/coldread/`, the design, and for
  readiness the change: whatever text it was given with `-change`, or the contents of
  the file named with `-change-file`, usually the change's note. The command reads
  `VERDICT: PASS` or `VERDICT: FAIL` from the last line of each reply, prints the
  replies and can write them as markdown for a pull request. It exits non-zero when a
  session fails, so that a person's own script can tell; nothing in the harness reads
  that.
- `cmd/reverse` writes down the design of a repository that has code and no design.
  `survey` lists what a machine can see. `skeleton` writes a `design/DESIGN.md` whose
  frontmatter accounts for every file and whose body is questions. `draft` starts one session
  in the repository with the tools that read, `Read`, `Grep` and `Glob`, and no other,
  hands it `harness/reverse/reverse.md`, the survey and the shape to fill, and writes
  its reply to `design/DESIGN.md` and to the project's section of `CLAUDE.md`. `prompt`
  prints what that session would be handed. A fifth step, `where`, is for scripts: it
  prints where a repository sits as `key=value` lines: `root=`, the root's name, which is
  that of its `origin` or, with none, of its directory; `path=`, this repository's path
  from the root with forward slashes, `billing` or `services/billing`; and where the
  root's map lists a contracts repository beside it, `contracts_path=`, its path from
  the root, and `contracts_remote=`, its remote as the map gives it. Under no root it
  prints nothing. `bootstrap` reads the last two to write a CI workflow. A skeleton is known by one line it carries,
  the comment `<!-- design:skeleton -->`; a draft does not carry it, so once a draft
  is written the document counts as someone's and a second draft needs `-force`. The
  project's section of `CLAUDE.md` is what lies between the comments
  `<!-- design:project:begin -->` and `<!-- design:project:end -->`. The method's
  own rules sit above it between `<!-- design-harness:begin -->` and
  `:end -->`, come from `templates/CLAUDE.md`, and are never written by a session.
- `internal/session` is the one place a model is started: the arguments, the settings
  file, the runner the tests replace.
- `cmd/subject` and `internal/convention` say whether a subject is a Conventional
  Commit, in a repository whose frontmatter asks for them.
- `cmd/protection` reads how the server protects the default branch, in a repository
  whose frontmatter says `protection: required` (ADR-0011), and passes without reading
  anything in one that does not. It needs three things of the branch: that a pull
  request is required, that the status check named `harness` is required, which is the
  name of the one job in the CI workflow, and that a force push is refused. It reads
  them from two places and either will do for each: the branch's own protection, and
  the rules of every ruleset that applies to the branch. The first is shown to an
  administrator of the repository only; the second to anyone who can read it. With
  nothing lacking it prints one line and passes. With something lacking, read as an
  administrator, it refuses, names each thing, and prints the command that adds them.
  With something lacking and the branch's own protection not shown, it says so in one
  line and passes: whoever cannot read the setting cannot change it, and is not
  refused for it. A server that cannot be asked, no `gh`, no network, no login, is a
  refusal and never a pass. `-set` adds what is lacking and changes nothing else: it
  sends back the protection it read, with the pull request requirement, the `harness`
  check and the refusal of a force push added, and where the branch had none, the
  least that will do, a pull request with no approvals asked, so that a person working
  alone can still merge. It refuses to write over a protection it could not read, or
  one that names who may dismiss a review or bypass a pull request, which it cannot
  hand back whole; it then reads the branch again and reports what the server holds.
  `-set -n` prints what would be sent and sends nothing. `-set` acts whatever the
  frontmatter says, since the setting is made before the line that has it read is switched
  on, and where the frontmatter does not ask it says how to. When anything is lacking it
  writes all three into the branch's own protection, whatever a ruleset already
  supplies; when nothing is, it sends nothing. A value of `protection:` other than
  `required` is an error from this command, with the one value named, and never read
  as off: `safe-push` then pushes nothing. GitHub means an `origin` whose address names
  the host `github.com`, over HTTPS or SSH; any other host, a GitHub Enterprise server
  included, is elsewhere.

  What the server answers is sorted so. It is asked three things in turn: the
  repository, `repos/<owner>/<name>`; the branch's protection,
  `.../branches/<branch>/protection`; and the rules that apply to the branch,
  `.../rules/branches/<branch>`. No answer at all, or any status but 200 for the
  repository, a login that has lapsed included, is a server that cannot be asked. For
  the protection: 200 is the protection; 404 whose message says the branch is "not
  protected" is an administrator being told there is none; 403, and any other 404, is
  the protection not shown; anything else cannot be asked. The rules add to what the
  protection gave and never take away, and an answer for them that is not 200 adds
  nothing and refuses nothing, since a server may have no rulesets to offer. Of the
  rules, one of type `pull_request` is a pull request required, one of type
  `non_fast_forward` is a force push refused, and one of type `required_status_checks`
  requires each check its parameters list by `context`. The server lists there only
  rules in force, not those of a ruleset being tried out, and a rule counts whoever
  its ruleset lets bypass it, as a protection counts though an administrator may.

  What `-set` sends is the whole protection, because the server replaces it whole. Of
  what was read it carries back: for status checks, whether the branch must be up to
  date, and every check, as the list that ties a check to the app that reports it,
  with a check the server gave by name alone added to that list, and `harness` added
  untied; for reviews, the number of approvals, whether a stale review is dismissed,
  whether a code owner must review and whether the last push must be approved; who
  may push, as logins and slugs, or nobody named where nobody was; and whether
  administrators are held to it, a linear history is required, deletion is allowed,
  creation is blocked, conversations must be resolved, the branch is locked, and a
  fork may sync. A force push is always sent as refused. Signed commits are a setting
  of their own on the server that this request does not carry and does not change.
  Anything else the server holds there now or comes to hold is not carried, and is the
  limit of "changes nothing else". It reads through the `gh`
  tool as whoever is logged in, on GitHub only: where the frontmatter asks and `origin` is
  elsewhere, it refuses and says to take the line out. The default branch is
  `origin/HEAD` as last fetched, else `main`. Under a root each repository asks for
  itself: `protection:` is not one of the rules a nested repository takes from the
  root's frontmatter, because it is a fact about one repository's server settings.
- `safe-push` is the one way a branch is pushed. It never pushes `main`, or a branch
  not named `feat/` or `fix/`: work it finds on one is moved, commits and uncommitted
  changes both, to a new branch, and that is what is pushed. The new branch is named
  `feat/` or `fix/` and then a slug of the first commit's subject, which is also the
  pull request's title: the subject without its Conventional Commit type, whole when it
  is short and cut at a word when it is long. It is `fix/` when the subject opens with
  `fix`, or with Fix, Correct, Repair or Revert, and `feat/` otherwise. A branch whose remote is already merged
  into the default branch is treated the same way: new work on it is moved. It requires a clean tree; checks the branch merges cleanly into
  the remote's default branch; runs `go vet` over the harness itself and then the
  checker, which runs the repository's gate; checks the title; runs `cmd/protection`,
  and pushes nothing if it refuses; pushes with `--force-with-lease`; and opens the pull request on GitHub through `gh api`.
- `hooks/pre-push` refuses a push to `main` and runs the checker; `hooks/commit-msg`
  runs `cmd/subject`; `hooks/install` points `core.hooksPath` at them and refuses to
  switch off hooks already in use.
- `templates/` holds the shapes: the design, a change note, the adoption note,
  `CLAUDE.md`, and the CI workflow.

**Several repositories.** A system is usually several repositories: each service in
its own, checked out side by side inside a project folder, with a contracts repository
beside them holding the definitions they talk through. The project folder is itself a
repository, the root. Its design holds what binds all of them, and its frontmatter lists them
under `repos:`, the map. An entry is a `path:` relative to the root; a `kind:` in the
root's own words (service, platform), of which the harness gives meaning to one,
`contracts`, the repository that holds the definitions the others talk through; a `remote:` it is cloned from, left out for one that is local only; and
`status: planned` for one that is decided and does not exist yet:

```yaml
repos:
  - path: contracts
    kind: contracts
    remote: git@github.com:acme/contracts.git
  - path: billing
    kind: service
    remote: git@github.com:acme/billing.git
  - path: ledger
    kind: service
    status: planned
```
 A repository nested under a root
has a design of its own and no harness of its own. It runs the root's by relative
path, `../harness` one level down, and what the root decided binds it and is cited by
the root's name, `project/ADR-0004`. Every command finds its repository from the
directory it was typed in and the harness from the nearest directory above that holds
one, so the same commands work at the root and beneath it. The root is found the same
way: the nearest directory above a repository that holds both a design and a harness.
A repository's name, the root's included, is the last part of its `origin` remote
without `.git`, or its directory's name when it has no remote; that is the name a
citation of the root is qualified by. A nested repository is worked on in place, under
its root: checked out with no root above it, it has no harness to run, which is why
its CI checks the root out around it. A citation qualified by any name other than the
root's found on disk is not looked up.

- `cmd/sync` makes the disk match the map: it clones a listed repository that is
  missing, fetches one that is there, skips one marked planned, and never merges.
- The checker, at a root whose frontmatter has a map, refuses a repository found on disk
  under it that the map does not list: any directory with a `.git` of its own that is
  not a linked worktree, a vendored checkout included, which is then listed with no
  remote. Frontmatter with no `repos:`, or an empty one, is no map and nothing is held to
  it. In a nested repository it follows a citation
  qualified by the root's name into the root's design, and refuses one not defined
  there.
- The checker holds a definition to having someone who serves it, where the root's
  map lists a repository of kind `contracts` (ADR-0010). A definition is an RPC, or a
  field of a top-level message whose name ends `Request`, fields in a `oneof` included
  and those of a message nested inside it not, in a `.proto` file a person wrote: any
  that is not under `gen/` at the top of the repository, nor under a directory named
  `vendor` or `node_modules` at any depth; its name is package, service or message, and RPC or field, joined by dots:
  `billing.v1.BillingService.CreateInvoice`, `billing.v1.CreateInvoiceRequest.currency`.
  A file with no `package` line names its definitions without one:
  `BillingService.CreateInvoice`.

  In the contracts repository, every note the rules bind carries a fenced block whose
  info string is `servers`: a YAML list of the definitions the change adds, each with
  the repository that serves it and the ones that call it, by their paths on the
  root's map, or `[]` when it adds none.

  ````
  ```servers
  - name: billing.v1.CreateInvoiceRequest.currency
    server: billing
    callers: [ledger]
  ```
  ````

  The checker refuses a bound note with no block, an entry whose server or caller is
  not on the map or that names no caller, and any RPC or request field the working
  tree has that the merged state does not and no note lists. A field is new only when
  its request message already exists at the merged state: the fields of a request
  message that is itself new are never listed one by one, whether or not an RPC takes
  it yet, because the RPC that comes to take it is listed. Every entry of every note is
  validated, whatever the note's date: an entry with no `name:`, no `server:` or no
  caller is refused, and so is one whose server or caller has left the map. A note may
  have several blocks and a name may be listed in more than one note; all are read, and
  in the server each entry that names it binds. A repository the map marks planned is
  on the map, and may be named as a server or a caller. A server may be listed among
  its own callers. The merged state is the remote's
  default branch as last fetched: `origin/HEAD`, else `origin/main`, else
  `origin/master`. In the contracts repository, where there is a remote and none of
  those is present, the checker refuses and says how to set one, since reading a branch
  against itself would pass everything; a repository with no remote at all has merged
  nothing anywhere and is read against its own `HEAD`. A server reading the contracts
  repository beside it falls back to that checkout's `HEAD`, which in CI is the default
  branch and on a machine may bind it to an entry not yet merged, never to fewer.

  In every other repository under that root, the checker reads the contracts
  repository's notes and definitions at its merged state, and for each entry naming
  this repository as server whose name a merged definition declares, requires an item
  under `serves:` in this repository's frontmatter with a `test:`, and requires every test
  `serves:` names to exist:

  ```yaml
  serves:
    - name: billing.v1.CreateInvoiceRequest.currency
      test: TestAnInvoiceIsIssuedInItsCurrency
  ```

  A block inside an HTML comment, as the note template shows one, or quoted inside
  a longer fence, is not a block and lists nothing. A malformed block, one never
  closed or not a list, is refused where it is read, in the contracts repository and
  in every server that reads its notes, and never taken for an empty one; whether an
  entry's server and callers are on the map is checked in the contracts repository
  only, so one repository never goes red for the wording of another's note. Every note of the contracts repository is read for
  entries, whatever its date; only the requirement that a note has a block follows
  `notes: from:`. An entry in a note whose definition has not merged binds nothing: it
  is a design.
  An entry on an unmerged branch of the contracts repository binds nothing. A contracts
  repository that is not checked out is a refusal, not a skip, in a repository that
  must read it. At a root, and under a root with no contracts repository, none of this
  applies; a contracts repository the map marks `status: planned` is no contracts
  repository yet, and where a map lists more than one the first that is not planned is
  the one. A repository under the root that the map does not list yet is held as a
  server: it must be able to read the contracts repository, and no entry binds it until
  the map lists it, since an entry naming a server off the map is refused in the
  contracts repository.

  The root's map is read at check time for this and for nothing else: to find which
  repository is which. No rule is taken from the root's frontmatter when a nested repository
  is checked (INV-016).
- `cmd/reverse`, in a nested repository, says in the survey where it sits and which
  siblings the root lists; writes a skeleton whose frontmatter takes the root's `commits:`
  and the sections the root asks of a note, and whose scope says how the root is cited;
  and hands a drafting session the root's design after the shape, to cite and not
  restate. It is not handed the contracts repository's entries: it is told where the
  sibling is and reads it with the tools it has, and says so if it cannot. The first
  note it writes carries every section that frontmatter asks for, since
  the note is dated the day the rules begin. At a root it writes the map into the
  skeleton from the repositories it
  finds, with a kind read from each one's name: `contracts` for one called
  `contracts`, `protos`, `api`, `schemas` or the like, `service` for any other, there
  for a person to correct.

`bootstrap`, at this repository's root, is POSIX shell. Given a project, new or old, it copies
`harness/` in, then writes `CLAUDE.md` or adds the method's rules to the end of one
that exists, the CI workflow, then the skeleton and the first note, then installs the
hooks, and runs the checker. Each of those is skipped when the thing is already there,
and a `harness/` directory that is not this harness stops it before anything is
written. `--update` is the one thing that replaces, and it replaces what is the
harness's: it swaps a project's `harness/` for this one, whole, keeping only the
project's two settings files, `coldread/coldread.yaml` and `reverse/reverse.yaml`;
and it rewrites the method's rules in `CLAUDE.md`, the lines between their two marks,
from the current template, leaving every line outside the marks as it was. In a
repository nested under a root there is no harness to swap, so `--update` there
rewrites the rules alone, from the root's templates. A change a project made to its
own copy is lost from the working tree by that and shows in `git diff`, where the
person updating decides what to keep.

Run on a repository under a root, `bootstrap` adopts it as nested: it copies no
harness; writes `CLAUDE.md` from `templates/CLAUDE.nested.md`, which names the path to
the root's harness and says to read the root's rules first; writes a CI workflow from
`templates/harness.nested.yml` that checks the root out and this repository inside it,
if the root has a GitHub origin to check out; installs the root's hooks by relative
path; and prints the entry for the root's map and the line for its `.gitignore`, which
are the root's files and a person's to change. `--standalone` gives such a repository
a copy of its own instead; it is still a repository under the root: it belongs on the
map, its skeleton takes the root's rules the same way, and its citations of the root
are followed. The first note bootstrap writes in a contracts repository carries an
empty `servers` block; the repository is known for one by its kind on the root's map,
or, adopted before the map lists it, by its name. A repository adopted when merged
contracts already name it as a server is refused by the checker from its first run,
for each definition it has no test named for: bootstrap prints the refusals and
finishes, and they are the first thing its design has to answer. That is the one case
in which a project bootstrap has run on does not pass (INV-009). The nested CI workflow checks out this repository with its whole
history, so that what a branch adds is read against what is merged, and the contracts
repository beside it where the root's map lists one with a GitHub remote; where it
lists one with no such remote bootstrap says the workflow cannot check it out, and the
checker refuses in CI until a person adds the step. A private contracts repository
needs a secret of its own, set by hand like the root's. The harness fixes no name for
either: the workflow's comment suggests `ROOT_READ_TOKEN` or `ROOT_DEPLOY_KEY` for the
root and `CONTRACTS_READ_TOKEN` or `CONTRACTS_DEPLOY_KEY` for the contracts
repository, and a person names the one they set on its checkout step. It checks
out the root's default branch, by the
owner and name read from the root's `origin` when bootstrap ran; the secret a private
root needs is named in the workflow's comment and set by a person. A root is bootstrapped first, then each repository in it.

## Principles

- **A design governs; it does not describe.** Only what must be governed goes in one,
  each project's rules are written once, and every project holds them in the same
  format (ADR-0014).
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
  `main`, and the checker, against frontmatter written so that it passes; the project's
  own gate is left for a person to switch on (ADR-0004, ADR-0006).
- **What a project gets is whole.** Nothing in `harness/` is needed from outside it
  (ADR-0005).

## What must always hold

- **INV-001** — Design before code: no source file exists that the design does not
  account for. *(the checker; `test: TestEveryFileIsAccountedForOrRefused`)*
- **INV-002** — A test the design names exists, as a whole word, in a test file the
  design accounts for. *(`test: TestANamedTestMustExist`)*
- **INV-003** — A decision or invariant id is defined once in a document, the design or
  a spine, and every id it cites as its own is defined in it; another repository's,
  qualified by its name, is not this document's to define.
  *(`test: TestAnIdIsDefinedOnceAndEveryCitationLands`)*
- **INV-004** — A change note dated from the frontmatter's `notes: from:` carries every
  section the frontmatter asks of a note, each with something under it; a note dated before
  is never reread. *(`test: TestADesignNoteCarriesItsParts`,
  `test: TestADesignSaysWhatItsNotesCarry`)*
- **INV-005** — A cold-read session is handed its prompt, the design and, for readiness, the
  change, and nothing else; it is started with no tools. *(`test:
  TestASessionIsHandedItsPromptTheDesignAndNothingElse`, `test:
  TestASessionIsStartedWithExactlyTheToolsItIsGiven`; what the `claude` tool itself
  adds to a session is read at review)*
- **INV-006** — The `coldread` command judges nothing: a verdict is the last line of a
  reply in the exact form, anything else is a failure, and no verdict is read by any
  other command. *(`test: TestAVerdictIsReadFromTheLastLineAndNothingElseCounts`)*
- **INV-007** — The session that drafts a design can read and cannot write. Only the
  `reverse` command writes, only `design/DESIGN.md` and the marked section of
  `CLAUDE.md`, and never over a design that is no longer a skeleton unless forced.
  *(`test: TestASessionIsStartedWithExactlyTheToolsItIsGiven`, `test:
  TestADraftNeverReplacesADesignSomeoneWrote`, `test:
  TestTheProjectsRulesReplaceOnlyTheirOwnSection`, `test:
  TestADraftIsReadAsTheDesignAndTheRules`)*
- **INV-008** — A survey and a skeleton decide nothing: the frontmatter a skeleton writes
  accounts for every file as it stands, names no test, asks of a note the four default
  sections from the day it is written, or under a root the sections the root asks
  (INV-016), and leaves the gate and the commit convention as
  comments. The convention is written switched on in two cases only: the history
  already follows it, or the repository is nested under a root whose frontmatter asks for it,
  which is the root's decision and not the skeleton's (INV-016). *(`test: TestTheSurveySaysWhatIsThereAndDecidesNothing`, `test:
  TestASkeletonsFrontmatterAccountsForEveryFileAndSwitchesNothingOn`)*
- **INV-009** — Adopting with bootstrap overwrites nothing that exists, adding to the
  end of a `CLAUDE.md` that is already there and leaving every line of it as it was,
  and commits nothing; and a project it has run on passes the checker with its own copy of the
  harness alone, or, adopted under a root, with the root's and no copy (INV-016); the
  one exception is a repository merged contracts already name as a server, which is
  refused until it names its tests (INV-018). With `--update`, and only then, it replaces what is the harness's:
  `harness/`, less the project's two settings files, and the method's rules between
  their marks in `CLAUDE.md`; and nothing else.
  *(`test: TestBootstrappingAnExistingProjectLeavesItPassingAndOverwritesNothing`,
  `test: TestAnUpdateReplacesTheHarnessAndTheRulesAndNothingOfTheProjects`)*
- **INV-010** — A design that still says `TODO(design)` is reported as unfinished on
  every run of the checker, and is not refused for it. *(`test:
  TestAnUnfinishedDesignIsSaidToBeUnfinished`)*
- **INV-011** — Nothing reaches `main` except through a pull request that passed the
  checker. *(`harness/hooks/pre-push` and `harness/safe-push` refuse the push, `test:
  TestWorkOnMainOrAStrayBranchIsMovedAndMainIsNeverPushed`, `test:
  TestABranchTheCheckerRefusesIsNotPushed`; where the frontmatter asks, the server's
  protection is read before every push, `test:
  TestAPushIsRefusedWhileTheServersProtectionLacksSomething`)*
- **INV-012** — Nothing under `harness/` needs anything outside it, or names a
  particular project. One test looks outside: `bootstrap_test.go` looks for `bootstrap`
  beside `harness/`, runs it where it finds it, which is here, and skips where it does
  not, which is in every project's copy. *(read at review; INV-009's test proves a copy
  works alone)*
- **INV-013** — The commit convention binds only a repository whose frontmatter asks for it.
  *(`test: TestTheConventionBindsOnlyARepositoryWhoseDesignAsksForIt`, `test:
  TestASubjectThatIsNotConventionalIsRefused`)*

- **INV-014** — A root's map is true of what is on disk: where a design lists the
  repositories nested under it, every repository found under it is one it lists. One
  listed and absent is not refused; `sync` clones it. *(`test:
  TestARepositoryOnDiskIsOnTheMap`, `test: TestTheMapIsBroughtInLineWithoutMerging`)*
- **INV-015** — In a repository nested under a root, a citation qualified by the root's
  name is defined in the root's design. *(`test:
  TestACitationOfTheRootLandsInTheRootsDesign`)*
- **INV-016** — A repository adopted under a root has no harness of its own and points
  at the root's by relative path, unless it was adopted with `--standalone`; either
  way the rules it takes from the root's frontmatter, the commit convention and the sections
  of a note, are written into its own, and no rule is read from the root's frontmatter when
  it is checked. What is read from the root then is its map, to find which repository
  is which, and its design, to follow a citation. *(`test:
  TestARepositoryUnderARootUsesTheRootsHarnessAndRules`, `test:
  TestANestedRepositoryTakesItsRulesFromTheRoot`, `test:
  TestARootsSkeletonCarriesTheMap`)*

- **INV-017** — In a contracts repository, every RPC and every field added to an
  existing request message is listed in a change note with the repository that serves
  it and at least one that calls it, each on the root's map; and every note the rules
  bind says what it adds, or that it adds nothing. *(`test:
  TestANewDefinitionIsListedWithItsServer`, `test:
  TestTheServersBlockIsReadFromItsFence`, `test:
  TestTheProtoScanFindsRPCsAndRequestFields`)*
- **INV-018** — A repository the contracts repository names as serving a definition
  names, under `serves:`, a test that exists for it. A name binds from the moment a
  merged definition declares it, and not before. A contracts repository that cannot be
  read is a refusal. *(`test: TestAListedDefinitionIsRefusedUntilItsServerNamesATest`,
  `test: TestAPlannedEntryBindsNoServerUntilItsDefinitionMerges`, `test:
  TestARepositorysRoleIsReadFromTheRootsMap`)*
- **INV-019** — The areas a design defines and the spines on disk are the same set: every
  area has a spine that reads, and every document under `design/spines/` is the spine of
  an area. A source file is under one area at most; a file under none is the design's
  own, under its `files:`, or it is refused. *(`test:
  TestAnAreaHasASpineAndASpineHasAnArea`, `test: TestAFileIsUnderOneSpine`, `test:
  TestADesignDividedIntoAreasPasses`, `test: TestAnAreasNameCanQualifyAnId`)*
- **INV-020** — A test a spine names exists in a test file its own area holds. An id
  cited across documents of one repository, an area's by the area's name or the design's
  by the repository's, is defined in the document named. *(`test:
  TestASpinesTestIsInItsOwnArea`, `test: TestACitationAcrossDocumentsLandsInTheOneNamed`)*
- **INV-021** — No session and no reader is handed another area's spine: a cold read of
  one area is handed the design and that area's spine; a session drafting a spine is
  handed the design and a survey of that area's files alone; and `check -for` names the
  design and the spines of the areas the paths fall in, and no other. *(`test:
  TestAnAreaIsReadWithTheDesignAndWithoutTheOtherAreas`, `test:
  TestASpineIsDraftedFromItsAreaAndTheDesignAlone`, `test:
  TestTheDocumentsToReadAreTheDesignAndTheAreasSpine`)*
- **INV-022** — A spine is drafted only for an area the design defines, and a draft
  replaces only a skeleton unless forced; a reply that is not a spine is kept and not
  written. *(`test: TestASpinesSkeletonIsWrittenOnlyForAnAreaTheDesignDefines`, `test:
  TestADraftedSpineReplacesOnlyASkeleton`, `test:
  TestEveryAreaADraftDefinesIsGivenASkeleton`)*

## What no machine proves yet

Each rule below is held by a person reading, with what would prove it and what makes
that due. None is accepted as final.

| Gap | Today | Destination | Trigger |
|---|---|---|---|
| A design holds only what must be governed (ADR-0014) | The prompts and the documents say so; a person reads a draft for it | A size each document is held to and reported at, and a cold read asked whether each rule earns its place | The first design that a cold read passes and a person finds too long to read |
| INV-005 | the command hands a session only the prompt, the design and for readiness the change, and runs it away from the project; the `claude` tool may still load a person's own global instructions and memory into it | a start-up mode of the tool that loads nothing and still uses the stored login, named in `internal/session`, with the test extended to pin it | the tool offering one |
| INV-011 | the server's protection is read at a push, by a repository that asks, as an administrator; switched off and merged past in the server's own pages, nothing says so until the next administrator pushes, and a repository none of whose administrators push is never read | a scheduled CI job that reads it with a token an administrator stored, and fails | the first project that asks for it |
| INV-011 | `-set` carries back the settings of a protection that are listed in *How it works*; one the server adds later is dropped by it without a word | `-set` refuses a protection holding a key it does not know | the first one lost |
| INV-011 | a ruleset's rules are read from the shape the server documents; no ruleset has been read from a live repository by a test | a test against a repository kept for it | the first project protected by a ruleset alone |
| INV-012 | read at review | a test that fails on any `..` path or absolute path in a script, and on a name from a list of project names | the first slip |
| INV-017 | definitions are read from protobuf only; a contracts repository in another format, OpenAPI or JSON Schema, has its notes held to the block and nothing scanned | a scan per format, behind the same two rules | the first project whose contracts are not protobuf |
| INV-017 | a contracts repository whose default branch is neither `main` nor `master`, checked out without `origin/HEAD`, as a CI checkout is, is refused until a step sets it | the workflow template sets it after checkout | the first such repository |
| INV-018 | the test a server names is proved to exist, not to exercise the definition | none a machine can give: it is read at review | not due |
| INV-016 | a nested repository's CI needs a secret that can read a private root, set by hand | none: it is a setting on the server | not due |

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
invariants, decisions and the map of files; a repository too large for that divides
into areas, each with a document of its own under this one (ADR-0013). It is edited in place when something
changes, with the reason in the commit, and git is its history. A decision keeps its id
for life.

One file because the document's reader is handed all of it at once, a cold read most of
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
the code. What the frontmatter does not state, the checker does not check.

A check that is sometimes wrong gets switched off, and takes the trustworthy ones with
it. The value of the small check is that a FAIL is never argued with.

#### Alternatives rejected

- **Lint the prose**: require every decision to have a reason of some length, every
  invariant a proof in a fixed grammar. It measures form, is satisfied by filler, and
  refuses honest documents that are shaped differently.
- **Have a model judge the design in CI.** See ADR-0003.
- **Fail on `TODO(design)`.** The adoption pull request is itself pushed through the
  checker, with the marks still in it; refusing them would make the method impossible
  to adopt through its own door. They are counted and reported instead (INV-010).

### ADR-0003 — A cold read advises; it never gates

Two sessions with no memory and no tools are handed the design and one question each.
Their verdicts are printed and written for the pull request, for the person who merges.
Nothing reads a verdict to allow or refuse anything. The command is run by hand when a
change to a design is material. The model, the most a session may spend in dollars and
the form of the verdict line are values in `harness/coldread/coldread.yaml`; the questions are the two prompt files
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
- **Hand a nested repository's cold read the root's design too.** It would pass a
  design that leans on the root without saying what it relies on. A session is handed
  the one design, whichever repository it is (ADR-0009).
- **One question.** Comprehension alone passes a design that explains the present and
  is silent on what a change must respect.

### ADR-0004 — A design is reverse-engineered in steps, and a person takes it on

For a repository with code and no design: a machine surveys, a model session drafts,
a person confirms. The survey reports and decides nothing. The drafting session is
given the tools that read and no others, and the command, not the session, writes its
reply. The session is told to mark what it could not find in the repository, in two
ways. A reason no file states ends `(inferred)`: a claim to be confirmed. Something it
could not know at all is written `TODO(design)` with the question, and alternatives
recorded nowhere are written `Not recorded. TODO(design)` rather than invented. A draft is written only over a skeleton; a design a
person has edited is replaced only with `-force`. The gate and the commit convention
are never switched on by a survey, the convention excepted when the history already
follows it or, under a root, when the root's frontmatter asks for it (INV-008). The model is a value in `harness/reverse/reverse.yaml`, and no session is given a spending cap, since one stopped short of the end has answered nothing; what
the session is told is `harness/reverse/reverse.md`.

A wrong draft is faster to correct than a blank page is to fill, so the draft is worth
having; but a design nobody has read is a guess. So the questions, the `TODO(design)`
marks, are counted by the checker on every run until a person has answered each, and
the `(inferred)` claims stay in the text for that person to confirm or correct as they
read; the checker does not count those, because a confirmed inference and an
unconfirmed one look the same to a machine.

#### Alternatives rejected

- **Let the session edit the repository itself.** It would then be trusted with every
  file in order to write two, and a design could be overwritten without the command
  being able to refuse.
- **Derive the design mechanically**, from the tree and the tests alone. That produces
  the frontmatter and a list of files; the decisions and the invariants are the part worth
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
written in. Three things need more, and say so when it is missing: opening a pull request
needs GitHub's `gh`, so does reading the server's protection where the frontmatter asks for
it (ADR-0011), and the cold read and the drafting session need the `claude`
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
- **Python or shell for the checker.** Shell cannot parse the frontmatter safely; Python
  brings an interpreter version and a package environment into every hook. What is
  shell, `bootstrap`, `safe-push`, the two hooks and their installer, parses nothing: each only
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

A repository whose frontmatter says `commits: conventional` has every commit's subject, and
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

### ADR-0009 — Several repositories share one harness at a root, and take their rules from it

A root is a repository whose design lists the repositories nested under it. A nested
repository has its own design, its own history and its own remote, and no harness: it
runs the root's by relative path. What the root decided binds every repository under it
and is cited there by the root's name and the id; a nested design records only what is
its own. A rule the root's frontmatter sets for all of them, the commit convention and the
sections of a note, is written into a nested repository's own frontmatter when it adopts the
method, and from then on is read from there. The root pins no commit of any nested
repository: its map says which exist and where they come from, and `sync` clones them.

One harness because a rule that binds several repositories must be one piece of code,
reviewed once. Written into each frontmatter, and not looked up, because a design must say
what binds its repository to a reader who has only that design, and because what a
repository is held to should not change when the root's frontmatter is edited for another
repository's sake: a rule reaches a nested repository through a change to its own
design.

#### Alternatives rejected

- **A copy of the harness in every repository.** Each is updated when someone
  remembers, and the services drift onto different versions of the same check. A
  repository that needs to stand alone can still have one, with `--standalone`.
- **Inherit the root's frontmatter at check time.** What binds a repository would then not be
  in its design, and would change when it is checked out without its root.
- **Git submodules.** The root would record every nested commit, turning each change to
  a service into a second change to the root, and a fresh clone or worktree arrives
  without them.
- **One repository for everything.** A real choice for some projects, and then none of
  this applies: one design, one harness. It is not forced on a project whose services
  already have their own histories, owners and release cadences.
- **Have bootstrap edit the root's design and `.gitignore`** to add the new repository.
  They are the root's, changed through the root's own pull request; bootstrap prints
  what to add.

### ADR-0008 — A change starts with a note in four parts, from a date

A change to code starts with `design/changes/<date>-<slug>.md`: the problem, the plan,
the alternatives rejected, the files. The checker requires each section of every note
dated from the frontmatter's `notes: from:`, with something under it. A repository asks more
of its notes by listing more sections in its frontmatter.

The note is where the alternatives are written while someone still remembers them. The
date is there because a rule binds what is written after it exists: rereading old notes
against new rules turns every new rule into an afternoon of editing history.

#### Alternatives rejected

- **Put the reasoning in the pull request.** It stays on the server, outside the
  repository, and is not there for the next session to read.
- **Require a new decision for every change.** Most changes are within the decisions in
  force; a record for each buries the ones that decide something.

### ADR-0010 — A definition names the repository that serves it, and the server names its test

Where a root keeps a contracts repository, a new RPC or request field is not finished
when its definition merges. The note that adds it names the repository that serves it
and the ones that call it, and from the moment the definition is merged the serving
repository's checker refuses until its design names a test for it. The two halves are
checked where each is owned: the listing in the contracts repository, the test in the
server.

The fault this exists for is a definition that merges, is generated into every
client, is called, and is served by nobody, or is accepted by a server that ignores
the new field. Each repository's own checks pass throughout; only the pair of them is
wrong. A request field is held to this and a response field is not, because a server
that ignores a request field changes what a caller asked for, while a response field
nobody fills is visible to the first caller that reads it.

What is merged binds, and nothing else: a name in a note with no definition is a
design, and a branch of the contracts repository is a proposal. So a server is never
held to something that may not land, and a definition and its server can be built in
either order.

#### Alternatives rejected

- **A registry of who serves what, kept at the root.** A third place to keep in step
  with the definition and the server, changed in a third pull request.
- **An annotation in the definition itself**, an option naming the server. It ties the
  check to one format and puts a deployment fact into an interface.
- **Bind the server from the note alone**, before the definition merges. A server would
  be refused for a field that does not exist yet and might never.
- **Skip the check when the contracts repository is not on disk.** In CI that is every
  run of a workflow someone forgot to extend, and a check that is silently skipped is
  believed.
- **Require a test that calls the definition by name.** Nothing can tell that from a
  test that merely mentions it; the name of the test is proved, and what it walks is
  read at review.

### ADR-0011 — The server's protection is read at the door, where the design asks

A repository whose frontmatter says `protection: required` has the server's protection of
its default branch read by `safe-push` before every push, and the push refused while a
pull request, the `harness` check or the refusal of a force push is missing. One that
does not say so has nothing read. The command that reads it can also add what is
lacking, when a person asks it to and never otherwise.

Everything else the method enforces runs on a person's machine or in CI, and what
makes CI count is one setting on the server, made by hand at adoption and never looked
at again. A repository can lose it, or never have had the check added to it, and every
local refusal still passes. Reading it at the push puts the question where an answer is
already being waited for, with a login that is already there.

It is read at the push and not by the checker, because the checker runs in CI with a
token that is not shown a branch's protection, and on every commit's hook, where a
call to a server has no place. It is opt-in because it needs GitHub, the `gh` tool
and, to be of use, an administrator among those who push.

#### Alternatives rejected

- **A check in `cmd/check`.** CI could not run it without a stored administrator's
  token, and every local run of the checker would wait on a server.
- **Refuse whoever cannot read the protection.** Every contributor who is not an
  administrator would be locked out by a setting they can neither see nor change.
- **Set the protection from `bootstrap`.** Adoption would change a setting on a server
  without being asked; `-set` is a command a person types.
- **Always on.** A project on another server, or with no `gh`, could not push.
- **Replace the protection with a known-good one.** A team's approvals, restrictions
  and other checks would be wiped by a command that said it was adding one.


### ADR-0012 — The commands are named for what they do

The command that drafts a design from existing code is `cmd/reverse`, and the one that
puts a design to sessions with no memory is `cmd/coldread`; their prompts and settings are
in `harness/reverse/` and `harness/coldread/`. A place a draft could not answer is marked
`TODO(design)`, and the marks the commands leave in a project's files are
`<!-- design:skeleton -->` and `<!-- design:project:begin -->` / `:end -->`. They were
`elephant` and `goldfish` until 2026-11-26. `bootstrap --update` carries a project's
settings across from the old names and renames the old marks in its `CLAUDE.md` and
design. A name says what a thing does, so that a reader needs no story to follow a
command line; the owner found the two animals confused a reader who met them cold.

#### Alternatives rejected

- **Keep the animals and explain them in the README.** A name that needs its
  explanation read first is a cost paid by every reader, and the explanation was there
  and did not land.
- **Keep the commands' names and drop the animals from the prose only.** The README
  cannot show a command line without the name, so the question comes back there.
- **`draft` for the first command.** Its last step is `draft`, and `draft draft` reads
  as a mistake; `reverse` is what the README already called the work.

### ADR-0013 — A design may divide the code into areas, each with a spine

A design may name areas of the code in its frontmatter, under `spines:`, each with the
globs it holds. Each area then has a spine, `design/spines/<name>.md`: a document with
the design's sections and the rules of that area alone. The areas are defined in the
design and nowhere else, so that the division is one decision read in one place; a
spine says nothing about which files are its own. A file is under one area at most, and
the areas and the spines on disk are the same set (INV-019). Each document numbers its
own ids, and a citation across documents names the other (INV-020). Whoever works in an
area, person or session, is handed the design and that area's spine and no other
(INV-021).

The reason is the reader. ADR-0001 keeps a design in one file because its reader is
handed all of it at once. In a repository of millions of lines that file is either too
thin to build from or too long to read before a change, and an agent that reads the
rules of forty areas to change one has spent its attention on thirty-nine that do not
bind it. An area's rules are read by whoever changes that area. Ids are each document's
own because one sequence across a large repository is taken by two pull requests at
once, and because an area's spine can then be written, read and renumbered by the
people who own the area. The word is the one the project this method was drawn from
used for such a document; until 2026-11-27 this repository used it for a design's
frontmatter, which is now called the frontmatter.

Nothing changes for a repository with no `spines:`.

#### Alternatives rejected

- **One document, however large.** The cold read fails first, and then people stop
  reading before they change things, which is the failure the method exists to prevent.
- **A design per directory, found by walking up from a file**, as `CLAUDE.md` files are.
  The division is then nowhere written down, two documents can both claim a file with
  no rule to say which wins, and a reader cannot see the areas without walking the tree.
- **Each spine lists its own files.** The same fact in as many places as there are
  areas: an overlap is found only by reading all of them, and moving a boundary is two
  edits that can disagree.
- **One sequence of ids across the repository.** Every citation would be bare, but two
  pull requests in different areas take the same number, and a renumbering in one area
  is a change to the whole.
- **Split the repository into several under a root** (ADR-0009). That is the right
  answer where the parts are built and released apart. A single application whose parts
  ship together is one repository, and its history, build and reviews should not be cut
  up to make its design readable.
- **Hand a cold read every spine.** It is the whole design, but it is the size the
  areas exist to avoid, and it is not what a person changing one area reads.

### ADR-0014 — A design holds what must be governed, and nothing else

The method is a tool for documenting a codebase's design and architecture and enforcing
them. A rule or a decision belongs in a design when it has to be governed: it binds
more than one part or more than one person's work, a change could break it without
anyone noticing at the place it was broken, and whoever changes the code must know it
beforehand. A choice inside one function or one file, which whoever edits it can see
and change safely, is code and stays there. A project that already has architecture
documents, standing rules or decision records brings what they govern into the design
when it adopts the method and retires the old copies; the design does not point at a
second set that stays in force. Every project that adopts the method so holds its
standards once, in one format.

This was the purpose from the start and was written nowhere, and what was not written
was not kept: the README described the mechanics, the drafting prompts said to write
what the code proves, and a first draft on a large repository came back with rules at
the level of one function, beside architecture documents that still said the same
things another way. A document that holds everything is read by nobody, and a rule
among a hundred descriptions is not enforced. One format across projects is what lets a
person, an agent or a check move from one to the next.

What says so: the opening of `README.md` and of `harness/FRAMEWORK.md`, the two drafting
prompts, the templates' opening paragraphs and the rules in a project's `CLAUDE.md`.
No machine holds a design to it yet.

#### Alternatives rejected

- **A design describes the whole repository, so that a session can build from it
  alone.** It was never the aim, and it does not scale: the document grows with the
  code and goes stale with every edit. A cold read asks whether what governs a change
  is there, not whether every detail is.
- **Leave a project's existing documents in force and cite them.** Less work on the day
  of adoption, and two standards from then on, with nothing to say which wins and
  nothing a checker can hold.
- **Let each project choose its own format for its rules.** Then nothing learned in one
  carries to the next, and no check can be written once for all of them.

## Files

| File | Purpose |
|---|---|
| `bootstrap` | Adopts the method in a project: copies `harness/`, then adds what is missing and overwrites nothing (INV-009). `--update` replaces a project's harness and keeps its settings. |
| `harness/FRAMEWORK.md` | The method, written for whoever works in a project that uses it. Its presence is how `bootstrap` knows a `harness/` is this one. |
| `harness/go.mod`, `harness/go.sum` | The harness's own module, `harness`, with one dependency (ADR-0005). |
| `harness/internal/design/` | Reads the frontmatter and the spine of each area it defines (ADR-0013), walks the repository, finds the root and the harness, matches globs, recognises test files. |
| `harness/internal/session/` | Starts a model session through the `claude` tool with exactly the tools it is given; reads a settings file. |
| `harness/internal/convention/` | Whether a subject is a Conventional Commit, and whether a repository asks for them (ADR-0007). |
| `harness/cmd/check/` | The checker (ADR-0002): `checks.go` holds files, tests, ids and the map; `notes.go` the notes; `contracts.go` a repository's role from the root's map, the `servers` block, the scan of `.proto` files and the two rules of ADR-0010; `main.go` runs them and then `validate:`. `checks.go` also holds the areas and their spines (ADR-0013), and `main.go` the `-for` that names the documents to read for a path. |
| `harness/cmd/coldread/` | The two questions (ADR-0003): composes what a session is handed, reads the verdict line, prints and writes the replies. With `-spine`, what is handed is the design and then one area's spine. |
| `harness/cmd/reverse/` | Reverse-engineering (ADR-0004): `survey.go` is the survey, the skeleton and the first note, with its empty `servers` block in a contracts repository; `main.go` the five steps, the reply's two parts, and the writes. `bootstrap_test.go` runs the real `bootstrap`. `spine.go` is the same steps for one area with `-spine`: the survey of the area alone, the skeleton, the prompt and the draft of its spine, and the skeletons a drafted design's areas are given. |
| `harness/cmd/subject/` | The command the hook and the push script call for ADR-0007. |
| `harness/cmd/protection/` | Reads the server's protection of the default branch, and on request adds what it lacks (ADR-0011): `protection.go` is what is read, what is lacking and what is sent; `main.go` the frontmatter's `protection:`, the `gh` tool and what is printed. |
| `harness/cmd/sync/` | Makes the disk match a root's map (ADR-0009). |
| `harness/coldread/` | `comprehension.md` and `readiness.md`, the two prompts; `coldread.yaml`, the model and the verdict prefix. |
| `harness/reverse/` | `reverse.md`, what a drafting session is told, which under a root includes citing the root and, beside a contracts repository, filling `serves:` for each definition it serves that it can name a test for; `reverse.yaml`, the model and budget. `spine.md` is what a session drafting one area's spine is told. |
| `harness/templates/` | `DESIGN.md`, the shape of a design, with the placeholders a skeleton fills, `{{NAME}}`, `{{DATE}}`, `{{FILES}}`, `{{VALIDATE}}`, `{{COMMITS}}`, `{{SECTIONS}}`, `{{REPOS}}` and `{{PARENT}}`; `change.md`, a note; `adopt.md`, the first note; `CLAUDE.md`, the rules for an agent; `harness.yml`, the CI workflow; `CLAUDE.nested.md` and `harness.nested.yml`, the same two for a repository under a root (ADR-0009). For ADR-0010, `change.md` carries a comment, for contracts repositories only, showing the `servers` block; `CLAUDE.nested.md` states both halves of the rule; and `harness.nested.yml` has one line, `{{CONTRACTS_CHECKOUT}}`, that bootstrap replaces with a checkout step for the contracts repository or removes. `spine.md` is the shape of a spine, with `{{AREA}}`, `{{REPO}}` and `{{FILES}}`. |
| `harness/safe-push` | The one door (ADR-0006). |
| `harness/safe_push_test.go` | Runs the real `safe-push`, without a pull request, against a bare remote it throws away: one test for each thing the script must do or refuse. The test of ADR-0011 puts a stand-in for `gh` on the path and the remote at a path that reads as a GitHub address. The only Go file at the module's root. |
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
- a cold-read session is handed the design and the change and nothing else — `TestASessionIsHandedItsPromptTheDesignAndNothingElse`
- a reply that does not end in the verdict form is not a pass — `TestAVerdictIsReadFromTheLastLineAndNothingElseCounts`
- a session is started with the tools it was given and no other — `TestASessionIsStartedWithExactlyTheToolsItIsGiven`
- a survey invents nothing and misses no file — `TestTheSurveySaysWhatIsThereAndDecidesNothing`
- a skeleton passes the checker and switches nothing on — `TestASkeletonsFrontmatterAccountsForEveryFileAndSwitchesNothingOn`
- a reply that is not a design is refused, not half written — `TestADraftIsReadAsTheDesignAndTheRules`
- a draft does not replace a design a person wrote — `TestADraftNeverReplacesADesignSomeoneWrote`
- a project's own `CLAUDE.md` keeps everything outside the marked section — `TestTheProjectsRulesReplaceOnlyTheirOwnSection`
- bootstrap leaves a project passing, overwrites nothing, and is safe to run twice — `TestBootstrappingAnExistingProjectLeavesItPassingAndOverwritesNothing`
- a subject that is not conventional is refused — `TestASubjectThatIsNotConventionalIsRefused`
- a repository that has not asked for the convention is not bound by it — `TestTheConventionBindsOnlyARepositoryWhoseDesignAsksForIt`
- a repository on disk under a root that the map does not list is refused — `TestARepositoryOnDiskIsOnTheMap`
- a citation of the root's that the root does not define is refused — `TestACitationOfTheRootLandsInTheRootsDesign`
- a nested repository's skeleton carries the root's rules, and one with no root above it carries none — `TestANestedRepositoryTakesItsRulesFromTheRoot`
- a root's skeleton lists every repository found under it — `TestARootsSkeletonCarriesTheMap`
- a repository under a root is given no second harness, and passes with the root's — `TestARepositoryUnderARootUsesTheRootsHarnessAndRules`
- a listed repository is cloned or fetched and never merged — `TestTheMapIsBroughtInLineWithoutMerging`
- every RPC and request field of a definition is found, and nothing else is taken for one — `TestTheProtoScanFindsRPCsAndRequestFields`
- a servers block is read from its fence, and a malformed one is refused — `TestTheServersBlockIsReadFromItsFence`
- a repository is held to the contracts rules exactly when the root's map gives it a part in them — `TestARepositorysRoleIsReadFromTheRootsMap`
- a new definition no note lists with a server on the map is refused — `TestANewDefinitionIsListedWithItsServer`
- a merged definition whose server names no test is refused there, and so is a contracts repository that cannot be read — `TestAListedDefinitionIsRefusedUntilItsServerNamesATest`
- a definition that has not merged binds no server — `TestAPlannedEntryBindsNoServerUntilItsDefinitionMerges`
- a change branch that passes is pushed, and main does not move — `TestAChangeBranchThatPassesIsPushed`
- work on main or a stray branch is moved to a change branch before it is pushed — `TestWorkOnMainOrAStrayBranchIsMovedAndMainIsNeverPushed`
- new work on a branch whose pull request has landed is moved to a branch of its own — `TestABranchAlreadyMergedIsNotPushedToAgain`
- a tree with uncommitted work is refused — `TestADirtyTreeIsRefused`
- a branch that conflicts with main is refused, with the file named — `TestABranchThatDoesNotMergeCleanlyIsRefused`
- a branch the checker refuses is not pushed — `TestABranchTheCheckerRefusesIsNotPushed`
- a title that is not conventional is refused where the design asks for them — `TestATitleThatIsNotConventionalIsRefusedWhereTheDesignAsks`
- a branch's protection that lacks a pull request, the check or the refusal of a force push is named for each, whichever of the two places holds them — `TestAProtectionThatLacksSomethingIsNamedForIt`
- a protection the server does not show, or a server that cannot be asked, is not read as a branch with none — `TestAProtectionThatCannotBeReadIsNotTakenForNone`
- setting sends back what was there with what was lacking added, and writes over nothing it could not read whole — `TestSettingAddsWhatIsLackingAndChangesNothingElse`
- a repository that has not asked has nothing read — `TestProtectionIsReadOnlyWhereTheDesignAsks`
- a push is refused while the server's protection lacks something, where the design asks — `TestAPushIsRefusedWhileTheServersProtectionLacksSomething`
- an update replaces the harness and the method's rules, and nothing that is the project's — `TestAnUpdateReplacesTheHarnessAndTheRulesAndNothingOfTheProjects`
- a design divided into areas passes, with each file counted once — `TestADesignDividedIntoAreasPasses`
- an area with no spine, or a spine with no area, is refused — `TestAnAreaHasASpineAndASpineHasAnArea`
- a file two areas hold is refused, and so is one nothing holds — `TestAFileIsUnderOneSpine`
- a test a spine names that is only in another area is refused — `TestASpinesTestIsInItsOwnArea`
- a citation of another document's id that the document does not define is refused — `TestACitationAcrossDocumentsLandsInTheOneNamed`
- an area whose name could be read as a repository's, or is no name, or holds nothing, is refused — `TestAnAreasNameCanQualifyAnId`
- the documents named for a path are the design and that path's spine, and no other — `TestTheDocumentsToReadAreTheDesignAndTheAreasSpine`
- a cold read of one area is handed the design and that spine, never another — `TestAnAreaIsReadWithTheDesignAndWithoutTheOtherAreas`
- a skeleton is not written for an area the design does not define, or over a spine — `TestASpinesSkeletonIsWrittenOnlyForAnAreaTheDesignDefines`
- a session drafting a spine is handed nothing of another area — `TestASpineIsDraftedFromItsAreaAndTheDesignAlone`
- a drafted spine does not replace one a person took on, and a reply that is no spine is kept — `TestADraftedSpineReplacesOnlyASkeleton`
- every area a drafted design defines has a spine at once, and one that was there is kept — `TestEveryAreaADraftDefinesIsGivenASkeleton`
- the checker and the drafting command give one number for the marks left in a document — `TestTheMarksLeftForAPersonAreCountedOneWay`
