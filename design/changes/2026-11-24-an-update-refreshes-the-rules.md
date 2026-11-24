# An update replaces the method's rules in CLAUDE.md as well as the harness

## The problem

`bootstrap --update` swapped a project's `harness/` for the current one and left its
`CLAUDE.md` alone. The method's rules in that file were written from a template once, at
adoption, so a rule changed in the template never reached a project that updated: its
checker was this year's and the rules its agents read were last year's. A repository
nested under a root could not be updated at all, though its rules go stale the same way
when the root's harness is replaced (issue #3).

## The plan

The rules between `<!-- design-harness:begin -->` and `<!-- design-harness:end -->` are
the harness's, and go where it goes. `--update` rewrites what lies between the two marks
from the current template, with the same placeholders filled, and touches no line
outside them: what a project wrote above, its own section between the `elephant:project`
marks, and anything after. A file that is already current is left as it is and said to
be. In a nested repository `--update` rewrites the rules alone, from the root's
templates, since there is no harness there to swap. INV-009 says what an update
replaces.

A project that already has the harness need do nothing until it next updates; then its
rules are refreshed with it. A project that edited the rules between the marks loses
that edit on update, and sees it in `git diff`.

## Alternatives rejected

- **Print the new rules and leave the file to a person.** The same step that is skipped
  today.
- **Rewrite the whole file from the template.** It would take the project's own rules
  and its own section, which are not the harness's to replace.
- **Keep the rules out of `CLAUDE.md`, in a file under `harness/` it points to.** An
  agent is handed `CLAUDE.md`; rules one hop away are rules it may not read.

## Files

- `bootstrap`: the rewrite between the marks, and `--update` in a nested repository.
- `harness/cmd/elephant/bootstrap_test.go`: the test.
- `design/DESIGN.md`: the mechanism, INV-009, the gap row removed.
