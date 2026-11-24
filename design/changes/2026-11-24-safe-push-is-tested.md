# safe-push is walked by tests, against a remote that is thrown away

## The problem

`harness/safe-push` is the one door every push goes through, and nothing tested it: its
refusals were exercised only by use (issue #6). Two parts of it were written for this
repository and had never run: reading the remote's default branch, and naming the branch
work is moved to. The second was wrong. The slug was cut at 40 characters and then
always lost its last word, so `fix(app): mend the third line` became
`fix/mend-the-third` whatever its length.

## The plan

`harness/safe_push_test.go` runs the real script, with `--no-pr`, in a clone of a bare
remote made for the test: a change branch that passes is pushed; work on `main`, on a
branch someone else named, or on a branch already merged is moved and `main` never
moves; a dirty tree, a branch that conflicts, a branch the checker refuses and, where
the design asks for them, a title that is not conventional are each refused with the
remote left as it was. The tests are named in the spine.

The slug keeps a short subject whole and cuts only a long one, at a word.

A project that already has the harness gets both when it next updates and need do
nothing else. The part that opens the pull request is not walked: it needs a server.

## Alternatives rejected

- **A shell test framework.** One more tool to install wherever the gate runs; the gate
  already runs `go test`, and a Go test can run a script.
- **Test the script's pieces by sourcing it.** It is one script top to bottom, and what
  matters is what reaches the remote.
- **Fake `gh` to walk the pull request step.** It would pin the fake. The step is two
  calls, and is exercised each time this repository is pushed.

## Files

- `harness/safe_push_test.go`: new, at the module's root beside the script.
- `harness/safe-push`: the slug.
- `design/DESIGN.md`: the tests, the files row, the gap row for the script removed.
