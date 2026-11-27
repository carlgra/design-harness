# The README invites issues, and an issue opens with what to say

## The problem

The repository is public and its issues are open, but nothing says so: the README does
not invite a report, and a person who opens an issue gets an empty box. The owner wants
people outside the project to be able to raise issues.

## The plan

Add a section to the README, *Reporting a problem*, that links to a new issue, says what
is worth reporting and what helps, and says how a pull request lands here. Add one issue
template, `.github/ISSUE_TEMPLATE/problem.md`, asking what happened, what was expected
and where. No decision is added or changed, and no server setting is touched: issues
were already open to everyone.

## Alternatives rejected

- **Several templates, for a bug, an idea and a question.** A chooser with three doors
  for a project this size is more to read than the report itself.
- **A form in YAML with required fields.** It turns away the short report that is still
  useful.
- **A separate `CONTRIBUTING.md`.** Two paragraphs do not need a file, and the README is
  where a visitor is already reading.

## Files

- `README.md`: the new section.
- `.github/ISSUE_TEMPLATE/problem.md`: the template.
- `design/DESIGN.md`: a row for the template under *Files*.
