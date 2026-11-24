You are writing down the design of a repository that already exists and has never had
one. You are in its root, and you can read it: `Read`, `Grep` and `Glob`, and nothing that
writes. Your reply is the design document, and the project's rules for an agent working in
it. A program writes your reply to disk; a person then reads every line of it and corrects
what you got wrong.

The document you are writing is the only thing a future reader will be handed. A session
with no memory will be given it, and nothing else, and asked to explain the repository and
to build a change from it. Write for that reader.

After the first line of dashes below is a survey a machine made: the files, the tests
found, the documents that may already record decisions, the commits. After the second is
the shape to fill, with its frontmatter, the spine, as the survey gives it.

## How to work

`harness/` and `design/changes/` are the method's own, already in place: the checker and
its scripts, and the note that records adopting it. Keep `harness/**` in `files:`, give it
one row under `Files`, name none of its tests, and do not spend reading on it;
`harness/FRAMEWORK.md` says what the method is if you need to know.

1. Read before you write. Start with the documents the survey lists, then the entry
   points, the build and CI configuration, the schema or migrations if there are any, and
   the tests. Then read the code of every part you are going to describe. Do not describe
   a part you have not opened.
2. Write what the code proves, not what would be tidy. If two parts disagree, say so. If a
   directory looks abandoned, say what it holds and that nothing appears to use it.
3. Be honest about what you could not see. You can see what the code does. You usually
   cannot see why, or what else was considered. So:
   - Where you state a reason or an intent that no file states, end the sentence with
     `(inferred)`.
   - Where a decision's alternatives are not recorded in any document, comment or commit
     subject you were given, write exactly `Not recorded. TODO(elephant)` under
     `Alternatives rejected`. Never invent an alternative.
   - Where something a reader needs is simply not knowable from the repository, write
     `TODO(elephant):` and the question a person must answer.
   A confident guess is the failure. A marked gap is the document working.

## A repository that is one of several

If the survey says the repository is **nested under a root**, the root's design is given
to you after the shape. What the root decided binds this repository. Do not restate it:
cite it by the root's name and the id, exactly as the survey shows, wherever this
repository's code is doing what the root requires, and record here only the decisions and
invariants that are this repository's own. Keep the paragraph in `Scope` that says where
it sits. Keep `commits:` and `notes:` as given: they came from the root's spine.

Where the survey lists a sibling of kind `contracts`, the definitions repositories talk
through live there and not here. Under `How it works`, say which of them this repository
serves and which it calls, as far as its code shows, naming each as the code names it; if
you can read the sibling at the path the survey gives, check the names against it, and if
you cannot, say that you could not.

If the survey says repositories are **nested under this one**, this is a root. Its design
is about what binds all of them, not about what any one of them does: keep the `repos:`
map as given, correcting a `kind:` only where a document says what the repository is, and
add a section `The map` after `Scope` with one row per repository: path, kind, what it is.
Do not read into the nested repositories beyond what a row needs.

## What each section holds

**The spine** (the frontmatter). Keep its comments and its `notes:` block as given.
- `files:` starts as the coarsest globs that cover everything. Narrow it where that says
  more, a glob per part, but every file that exists must still fall under one: a trailing
  `/**` is everything under a directory, otherwise `*` does not cross a `/`. Never drop a
  glob to hide files you did not understand; describe them in `Files` as not understood.
- `tests:` names the tests that walk a path a fault would travel, the ones you cite under
  `What must always hold` and `Files`. Use only names from the survey's list, spelled
  exactly. A few that matter beat a long list.
- Leave `validate:`, `commits:` and `repos:` exactly as given, commented or not. They
  are a person's to switch on or change.

**Scope.** What the repository is for, in a paragraph a stranger could repeat. What it
holds, and what it deliberately does not.

**How it works.** The parts, what each does, how they fit: what calls what, what is stored
where, what crosses each boundary, what runs when. Name real paths. This is the section
the comprehension test leans on hardest.

**Principles.** The few rules of thumb the code visibly follows, each a bold line and a
sentence, citing the decision that sets it. Keep the one the shape already has. Three true
ones beat ten plausible ones.

**What must always hold.** The invariants: what must be true for the system to be correct,
whatever changes. Look for them where code refuses things: validation, constraints in a
schema, assertions, authorisation checks, error paths, a test named for a rule, a comment
that says must, never or always, a CI step that fails the build. Each is one list item:
`- **INV-001** — the rule, as a sentence that is true. *(what proves it)*`. Number them in
order. What proves it is one of: `test: TheTestName` for a test that exists and that you
have read, which also goes in the spine's `tests:`; `the checker` for INV-001 as given;
the name of the mechanism, such as a database constraint or a CI step, with its file; or
`read at review` when nothing does. Keep INV-001 as the shape has it.

**What no machine proves yet.** One row for every invariant whose proof is
`read at review`: how it is held today, the test or check that would prove it, and what
would make writing that due. If every invariant is proved, say so in a sentence and leave
the table out.

**How a change lands.** Keep as given.

**Decisions.** One `### ADR-NNNN — title` for each material choice the repository
embodies, numbered from ADR-0001 in the order a reader needs them: the language and
framework, how it is structured into parts, where data lives and in what store, how it
talks to anything outside itself, how it is built, deployed and configured, how it
authenticates, what it deliberately leaves out. The title is the decision as a sentence
that is true. Under it: what was decided and why, stated as a constraint and not a value,
naming the file a value lives in rather than repeating it. Then
`#### Alternatives rejected`. If the repository already has decision records, bring in
every one still in force, keep its id if it has one of the form ADR-NNNN so that existing
citations still land, and leave out the ones that were superseded: this document is
current state. Cite decisions and invariants only by ids you define.

**Files.** A table: every path under `files:`, at the grain a reader needs, and what it is
for. Then the paths a fault would travel and the test that walks each, one per line, for
every name in the spine's `tests:`.

## The project's rules

The second part of your reply goes into `CLAUDE.md`, under a heading that is already
there, as the part that is this project's own. Read the existing `CLAUDE.md`, `AGENTS.md`
or other agent instructions if the survey lists any, and carry over what is still true.
Write only what an agent would get wrong here without being told:

- the commands to build, test, lint and run, exactly as the repository's own scripts and
  CI use them;
- the conventions the code visibly keeps that a newcomer would break: layout, naming,
  error handling, how a new part is added;
- what must never be done here, where a document or the code says so.

Short. No headings above `###`. Nothing that `design/DESIGN.md` already says, and nothing
about the method itself: that is already in the file.

## Your reply

Reply with exactly these two parts and nothing before, between or after them, no
commentary and no code fence around either:

=== design/DESIGN.md ===
(the whole document: the frontmatter between its two lines of `---`, then the body)
=== CLAUDE.md ===
(the project's rules)
