---
# The spine of one area: the rules of {{AREA}} alone. Which files the area
# holds is said once, in design/DESIGN.md under spines:, and not here.
#
# Tests that must exist in this area's own test files, by name: each walks a
# path a fault would travel.
tests: []
---

# {{AREA}}

<!-- design:skeleton -->

This is the spine of `{{AREA}}`, one area of `{{REPO}}`: what the area is for, how it
works, what must always hold in it, every decision in force with its reason and what
was rejected, and which files realise it. It is read after `design/DESIGN.md`, which
holds what binds every area, and without the spines of the other areas. It holds what
must be governed in this area and nothing else. It is current state, edited in place;
git is its history.

The files it holds, as `design/DESIGN.md` gives them:

{{FILES}}

Ids here are this spine's own. From anywhere else an id of this spine is cited as
`{{AREA}}/ADR-NNNN`; here the design's are cited as `{{REPO}}/ADR-NNNN`, and another
area's by that area's name.

## Scope

TODO(design): what this area is for, in a paragraph a stranger could repeat. Then
what it holds, and what belongs to another area or to the design and not here.

## How it works

TODO(design): the parts, what each does, and how they fit together; and what
crosses the boundary of the area: what it calls outside itself, and what calls it.

## What must always hold

Each rule has an id it keeps for life, and says what proves it: a named test, a
mechanism with its file, or a person reading at review.

TODO(design): the invariants of this area. For example:

- **INV-001** — TODO(design): the rule, as a sentence that is true. *(read at review)*

## What no machine proves yet

| Gap | Today | Destination | Trigger |
|---|---|---|---|
| TODO(design) | how the rule is held now | the test or check that would prove it | what makes that due |

## Decisions

One decision each, with its reason and what was rejected. A decision that binds more
than this area belongs in `design/DESIGN.md`.

### ADR-0001 — TODO(design): the decision, as one sentence that is true

TODO(design): what was decided, and why.

#### Alternatives rejected

TODO(design): what else was considered, and why not.

## Files

TODO(design): every path this area holds and what it is for.

| File | Purpose |
|---|---|
| `path/` | what it is for |

Paths a fault would travel, and the test that walks each:

- TODO(design): what goes wrong — `TheTestThatWalksIt`
