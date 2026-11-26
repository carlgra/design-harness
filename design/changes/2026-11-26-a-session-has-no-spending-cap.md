# A session has no spending cap, and the goldfish runs on a lower model

## Problem

`goldfish.yaml` set `max-budget-usd: 3` and `elephant.yaml` set 15, and `internal/session`
passed the figure to the `claude` tool, which stops a session whose cost reaches it. In a
trial on 2026-11-26 an earlier form of the goldfish with the same cap, handed a design of
about 417 KB, had both sessions end `Exceeded USD budget (3)`, reported as FAIL: the
design could not be asked about at all, and the report read as a verdict on it. The same
stop waits for the elephant on a repository large enough.

## Plan

The setting goes from both files, from `session.Settings`, from the runner's signature and
from the arguments: no session is ever given a cap. The test that pins a session's
arguments now fails if a cap appears among them.

The goldfish's model moves from `claude-fable-5-2` to `claude-sonnet-5-5`. With no cap,
what a run costs is set by the model alone, and its two questions are reading questions.
The elephant keeps its model: it drafts the design, once, and the draft is what everything
after rests on.

Within the decisions in force; no new decision.

## Alternatives rejected

- **Higher caps.** Any figure is one larger design away from the same stop.
- **Keep the key, default 0.** A cap that reports as FAIL is a fault whoever sets it; if a
  cap is wanted later it comes back with an outcome of its own, "could not be asked".
- **Lower the elephant's model too.** Not tried; the one measured draft was on the
  current model.

## Files

- `harness/internal/session/session.go`, `session_test.go`: the field, the argument and
  the flag removed; the test refuses a cap.
- `harness/cmd/goldfish/main.go`, `main_test.go`, `harness/cmd/elephant/main.go`: the
  runner's new signature.
- `harness/goldfish/goldfish.yaml`, `harness/elephant/elephant.yaml`: the key removed;
  the goldfish's model.
- `bootstrap`, `harness/FRAMEWORK.md`, `design/DESIGN.md`: where the budget was named.
