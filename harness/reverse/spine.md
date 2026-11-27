You are writing down the design of one area of a repository that already exists. The
repository has a design, `design/DESIGN.md`, which divides its code into areas and holds
what binds all of them. Each area has a document of its own, its spine, holding the rules
of that area alone. You are writing the spine of one area. You are in the repository's
root, and you can read it: `Read`, `Grep` and `Glob`, and nothing that writes. Your reply
is the spine. A program writes it to disk; a person then reads every line of it and
corrects what you got wrong.

The spine exists so that whoever changes this area reads the repository's design and this
document, and none of the other areas'. A session with no memory will be handed those two
and asked to explain the area and to build a change in it. Write for that reader, and keep
out of it everything that reader does not need.

After the first line of dashes below is a survey a machine made of the area: its files,
the tests found in it, its documents. After the second is the shape to fill. After the
third is the repository's design.

## How to work

1. Read the area, and only as far outside it as you must to say what crosses its
   boundary. Start with any documents the survey lists, then the entry points and the
   tests, then the code of every part you are going to describe. Do not describe a part
   you have not opened.
2. Write what the code proves, not what would be tidy. If two parts disagree, say so.
3. Do not restate the repository's design. Where this area's code is doing what the
   design requires, cite the decision or invariant by the repository's name and its id,
   exactly as the shape shows. A rule that would bind another area as well does not
   belong here: leave it out, and say at the end of `Scope`, as a `TODO(design):`, that
   it may belong in `design/DESIGN.md`.
4. Be honest about what you could not see. You can see what the code does. You usually
   cannot see why, or what else was considered. So:
   - Where you state a reason or an intent that no file states, end the sentence with
     `(inferred)`.
   - Where a decision's alternatives are not recorded in any document, comment or commit
     you were given, write exactly `Not recorded. TODO(design)` under
     `Alternatives rejected`. Never invent an alternative.
   - Where something a reader needs is not knowable from the repository, write
     `TODO(design):` and the question a person must answer.
   A confident guess is the failure. A marked gap is the document working.

## What each section holds

**The frontmatter.** Keep its comments. `tests:` names the tests that walk a path a fault
would travel, the ones you cite under `What must always hold` and `Files`. Use only names
from the survey's list, spelled exactly; they must be in this area's own test files. A few
that matter beat a long list. Which files the area holds is not said here and is not yours
to change.

**Scope.** What the area is for, in a paragraph a stranger could repeat. What it holds,
and what belongs elsewhere.

**How it works.** The parts, what each does, how they fit, naming real paths; then what
crosses the area's boundary: what it calls outside itself and what calls it, as far as the
code shows.

**What must always hold.** The invariants of this area, numbered from INV-001 in this
document's own numbering. Look for them where code refuses things: validation,
constraints, assertions, authorisation checks, error paths, a test named for a rule, a
comment that says must, never or always. Each is one list item:
`- **INV-001** — the rule, as a sentence that is true. *(what proves it)*`. What proves
it is `test: TheTestName` for a test that exists in this area and that you have read,
which also goes in `tests:`; the name of a mechanism with its file; or `read at review`.

**What no machine proves yet.** One row for every invariant whose proof is
`read at review`. If every invariant is proved, say so in a sentence and leave the table
out.

**Decisions.** One `### ADR-NNNN — title` for each material choice this area embodies and
the repository's design does not already record, numbered from ADR-0001 in this
document's own numbering. The title is the decision as a sentence that is true. Under it:
what was decided and why, stated as a constraint and not a value, naming the file a value
lives in. Then `#### Alternatives rejected`. Cite only ids that are defined: this
document's bare, the repository's design's and another area's by name.

**Files.** A table of the paths the area holds, at the grain a reader needs, and what each
is for. Then the paths a fault would travel and the test that walks each, one per line,
for every name in `tests:`.

Keep the opening paragraphs of the shape, down to `## Scope`, as given.

## Your reply

Reply with exactly this one part and nothing before or after it, no commentary and no
code fence around it:

{{HEADER}}
(the whole document: the frontmatter between its two lines of `---`, then the body)
