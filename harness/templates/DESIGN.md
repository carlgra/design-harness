---
# The spine: the part of the design a machine reads. Everything under the
# second line of dashes is read by people, and by sessions with no memory.
#
# The code this design accounts for. Every source file must be under one of
# these; the checker refuses one that is not. A trailing /** is everything
# under a directory; otherwise * does not cross a /.
files:
{{FILES}}
# Tests that must exist: each walks a path a fault would travel. The checker
# refuses a name that is in no test file under files:; the gate runs them.
tests: []
# The repository's own gate, run from the root after the checks pass.
{{VALIDATE}}
# A commit's subject, and so a pull request's title, is a Conventional Commit.
{{COMMITS}}
# The server's protection of the default branch is read before every push
# (GitHub, through gh). Take the # off once main is protected.
# protection: required
# What a note under design/changes/ carries, from the date the rule began.
notes:
  from: {{DATE}}
  sections: {{SECTIONS}}
{{REPOS}}
---

# {{NAME}}

<!-- elephant:skeleton -->

This is the design: what this repository is for, how it works, what must always
hold, every decision in force with its reason and what was rejected, and which
files realise it. It is current state, edited in place; git is its history. A
change to the code starts with a note under `design/changes/` and ends with this
file saying what is now true.

## Scope

{{PARENT}}

TODO(elephant): what this repository is for, in a paragraph a stranger could
repeat. Then what it holds, and what it deliberately does not.

## How it works

TODO(elephant): the parts, what each does, and how they fit together: what
calls what, what is stored where, what crosses each boundary. Enough that
someone who has never seen the code could explain it to a colleague.

## Principles

TODO(elephant): the few rules of thumb every decision here refers back to,
each one line in bold and a sentence of why, citing the decision that set it.

- **The design is current state; git is the history.** This document says what
  is decided now, is edited when that changes with the reason in the commit,
  and keeps each decision's id for life so that citations land.

## What must always hold

Each rule has an id it keeps for life, and says what proves it: a named test,
the checker, or a person reading at review.

TODO(elephant): the invariants. For example:

- **INV-001** — Design before code: no source file exists that the design does
  not account for. *(the checker)*

## What no machine proves yet

Each rule below is held by a person reading, with what would prove it and what
makes that due. None is accepted as final.

| Gap | Today | Destination | Trigger |
|---|---|---|---|
| TODO(elephant) | how the rule is held now | the test or check that would prove it | what makes that due |

## How a change lands

A change starts with a note under `design/changes/<date>-<slug>.md`: the
problem, the plan, the alternatives rejected, and the files to change and why.
Work happens on a `feat/` or `fix/` branch, never on `main`. `harness/safe-push`
runs the checker and the repository's own gate, pushes the branch and opens the
pull request; CI runs the checker again, and `main` is protected behind it. A
person merges. The change ends with this document saying what is now true.

## Decisions

One decision each, with its reason and what was rejected. The ids are kept so
that a citation always lands. A decision states a constraint, never a value:
where it depends on a number or a string, it says what the value must satisfy
and names the file the value lives in.

### ADR-0001 — TODO(elephant): the decision, as one sentence that is true

TODO(elephant): what was decided, and why.

#### Alternatives rejected

TODO(elephant): what else was considered, and why not.

## Files

TODO(elephant): every path under `files:` and what it is for.

| File | Purpose |
|---|---|
| `path/` | what it is for |

Paths a fault would travel, and the test that walks each:

- TODO(elephant): what goes wrong — `TheTestThatWalksIt`
