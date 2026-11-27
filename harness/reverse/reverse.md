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
the shape to fill, with its frontmatter as the survey gives it.

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
     subject you were given, write exactly `Not recorded. TODO(design)` under
     `Alternatives rejected`. Never invent an alternative.
   - Where something a reader needs is simply not knowable from the repository, write
     `TODO(design):` and the question a person must answer.
   A confident guess is the failure. A marked gap is the document working.

## What belongs in a design

A design is the set of rules a codebase is governed by. It is not a description of the
code, and it is not where small decisions go. Before you write a rule or a decision, put
it to this test:

A rule or a decision belongs in a design when it has to be governed: it binds more
than one part or more than one person's work, a change could break it without anyone
noticing at the place it was broken, and whoever changes the code must know it
beforehand. A choice made inside one function or one file, which whoever edits it can
see and can change safely, is code, and stays in the code.

So: a rule about how parts depend on each other, what may call what, where data lives
and who may write it, what crosses a boundary and in what form, what must never happen,
belongs. How one function is written, the name of a helper, the order of steps inside a
method, a constant's value, do not. Twelve rules that bind the whole are a better design
than a hundred true statements about the code. If you are unsure, leave it out.

The repository may already have documents that govern it: architecture documents,
standing rules for contributors or reviewers, decision records, conventions. Bring what
they govern into this document, as decisions and invariants in its format, each with its
reason where the document gives one. Do not point at them as rules that stay in force
elsewhere, and do not copy their descriptions of the code. The aim is one place where a
rule is written. Where you bring a rule in, name the document it came from in the
sentence, so that a person can retire the old copy.

## A repository that is one of several

If the survey says the repository is **nested under a root**, the root's design is given
to you after the shape. What the root decided binds this repository. Do not restate it:
cite it by the root's name and the id, exactly as the survey shows, wherever this
repository's code is doing what the root requires, and record here only the decisions and
invariants that are this repository's own. Keep the paragraph in `Scope` that says where
it sits. Keep `commits:` and `notes:` as given: they came from the root's frontmatter.

Where the survey lists a sibling of kind `contracts`, the definitions repositories talk
through live there and not here. Under `How it works`, say which of them this repository
serves and which it calls, as far as its code shows, naming each as the code names it; if
you can read the sibling at the path the survey gives, check the names against it, and if
you cannot, say that you could not. For each definition this repository serves that one
of its tests walks, add an entry to a `serves:` list in the frontmatter, `name:` the definition
in full as the contracts repository declares it and `test:` the test, and put the test in
`tests:` as well. Leave out any you cannot name a test for, and give each a row under
`What no machine proves yet`.

If the survey says repositories are **nested under this one**, this is a root. Its design
is about what binds all of them, not about what any one of them does: keep the `repos:`
map as given, correcting a `kind:` only where a document says what the repository is, and
add a section `The map` after `Scope` with one row per repository: path, kind, what it is.
Do not read into the nested repositories beyond what a row needs.

## A repository too large for one document

A design is read whole by whoever changes anything, so it must stay small enough to read.
If the survey shows a repository whose rules will not fit one document a person would
read before a change, as a rule of thumb more than a few hundred source files in several
distinct parts, divide the code into **areas**, and write this document as the design of
the whole: what the repository is for, how the areas fit together, and only the
invariants and decisions that bind every area. Each area's own rules go in a document of
its own, its spine, `design/spines/<area>.md`, which a later session drafts by reading
that area alone. Do not write the spines, and do not put an area's own rules here.

Define the areas in the frontmatter:

```yaml
spines:
  - name: checkout            # lower-case letters, digits and hyphens
    files: [src/checkout/**]  # the globs the area holds
```

An area is a part with rules of its own that someone could change without reading the
rest: a bounded part of the product, a layer, a package. Draw them where the code already
draws them, by directory. No file may fall under two areas. A file under no area must
still fall under a glob in `files:`, and is then this document's own to describe. Add a
section `The areas` after `How it works`, with one row per area: its name, its files and
what it is for. Under `Files`, give each area one row and leave the detail to its spine.

A repository small enough for one document has no `spines:` and no such section.

## What each section holds

**The frontmatter.** Keep its comments and its `notes:` block as given.
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
whatever changes, at the level of the architecture and not of one function. Look for them where code refuses things: validation, constraints in a
schema, assertions, authorisation checks, error paths, a test named for a rule, a comment
that says must, never or always, a CI step that fails the build. Each is one list item:
`- **INV-001** — the rule, as a sentence that is true. *(what proves it)*`. Number them in
order. What proves it is one of: `test: TheTestName` for a test that exists and that you
have read, which also goes in the frontmatter's `tests:`; `the checker` for INV-001 as given;
the name of the mechanism, such as a database constraint or a CI step, with its file; or
`read at review` when nothing does. Keep INV-001 as the shape has it.

**What no machine proves yet.** One row for every invariant whose proof is
`read at review`: how it is held today, the test or check that would prove it, and what
would make writing that due. If every invariant is proved, say so in a sentence and leave
the table out.

**How a change lands.** Keep as given.

**Decisions.** One `### ADR-NNNN — title` for each choice the repository embodies that
has to be governed, by the test above, numbered from ADR-0001 in the order a reader needs them: the language and
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
every name in the frontmatter's `tests:`.

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
