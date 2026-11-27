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
   boundary. Start with the documents the survey lists: those in the area, and, of those
   elsewhere in the repository, the parts that speak of this area, then the entry points and the
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
they govern into this spine, where they are this area's alone, as decisions and invariants in its format, each with its
reason where the document gives one. Do not point at them as rules that stay in force
elsewhere, and do not copy their descriptions of the code. The aim is one place where a
rule is written. Where you bring a rule in, name the document it came from in the
sentence, so that a person can retire the old copy. Bring a rule in whole or leave it
out: never send the reader to the old document for the rest of it. If an existing
document already numbers this area's decisions or invariants, keep each id exactly as it
is written. Take only what governs this area alone: what the repository's design already
holds is cited, not repeated.

Where the repository's existing documents already number its decisions or invariants,
those numbers are in its code's comments and in people's heads, and they are bare. So
give a rule that is new here a number above the highest any existing document uses, in
this document and in every spine alike, so that a bare number still means one thing.
The checker reads every id written in this document as a citation, and refuses one the
document does not define. So never write an id this document does not define, not even
to speak of it, to say where a rule came from, or to say that code comments use it. Name
the document a rule came from by its path alone. Another document's rule that is in
force is cited by that document's name, as the shape shows.
If the numbering needs a sentence, write it with no id in it, not even as the end of a
range: "the ids here are those `docs/architecture.md` gave them; a gap is a rule kept in
an area's spine, and new ones follow the highest".

The document is current state. Say nothing in it about how or when it was drafted, what
it was drafted from, when a decision was made or amended, or what is still to be written.
A numbered rule of an existing document that the repository's design did not take, and
that governs this area alone, keeps its id here.

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
document's own numbering, or as *What belongs in a design* says where the repository
already numbers its rules. Look for them where code refuses things: validation,
constraints, assertions, authorisation checks, error paths, a test named for a rule, a
comment that says must, never or always. Each is one list item:
`- **INV-001** — the rule, as a sentence that is true. *(what proves it)*`. What proves
it is `test: TheTestName` for a test that exists in this area and that you have read,
which also goes in `tests:`; the name of a mechanism with its file; or `read at review`.

**What no machine proves yet.** One row for every invariant whose proof is
`read at review`. If every invariant is proved, say so in a sentence and leave the table
out.

**Decisions.** One `### ADR-NNNN — title` for each choice this area embodies that has to
be governed, by the test above, and that the repository's design does not already record, numbered from ADR-0001, or above the repository's existing numbers where it has them, in this
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
