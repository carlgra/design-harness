# A design may divide the code into areas, each with a spine of its own

## The problem

The method keeps a repository's design in one file because its reader is handed all of
it at once (ADR-0001). The owner is adopting it in a web application of about five
million lines. There the one file is either too thin to build from or too long to read
before a change, and an agent that reads every area's rules to change one area has
filled its context with rules that do not bind what it is doing. The project this
method was drawn from once had a document per area, called a spine, and gave it up when
it went to one document per repository; this repository kept the word and used it for
the frontmatter, which is not what the owner means by it and confused him on reading
the README.

## The plan

Add ADR-0013. A design may define areas in its frontmatter, `spines:`, each a `name:`
and the `files:` it holds. Each area has a spine, `design/spines/<name>.md`, with the
design's sections and that area's rules alone; its frontmatter has `tests:` only. The
design holds what binds every area.

- The checker (INV-019, INV-020): the areas and the spines on disk are the same set; a
  file is under one area at most, and one under none must be under the design's
  `files:`; a spine's tests are in its own area's test files; each document numbers its
  own ids, and a citation across documents, `checkout/ADR-0003` or
  `<repository>/ADR-0004`, is looked up in the document named. `check -for <path>...`
  prints the documents to read for those paths.
- `coldread -spine <area>` hands a session the design and that spine, and no other
  (INV-021).
- `reverse skeleton|prompt|draft -spine <area>` does for one area what those steps do
  for a repository: the survey is of the area's files alone and the session is handed
  the design with it (INV-021, INV-022). `reverse draft` of the design is told how to
  propose areas for a repository too large for one document, and each area it defines
  is given its skeleton.
- The word "spine" no longer means the frontmatter anywhere: that is "the frontmatter".
  The test `TestASkeletonsSpineAccountsForEveryFileAndSwitchesNothingOn` is renamed
  with it.
- A repository with no `spines:` is unchanged.

## Alternatives rejected

- **One document, however large.** The cold read fails first, then people stop reading.
- **A design per directory, found by walking up from a file.** The division is written
  nowhere, and two documents can claim one file.
- **Each spine lists its own files.** The same fact in many places; an overlap is found
  only by reading all of them. The owner asked for the areas to be defined in the main
  design file.
- **One sequence of ids across the repository.** Two pull requests in different areas
  take the same number.
- **Split the repository under a root** (ADR-0009). Right where parts ship apart; wrong
  for one application that ships together.
- **Keep "spine" for the frontmatter and call an area's document something else.** The
  owner's word for an area's rules is spine, and the frontmatter already has a name.

## Files

- `harness/internal/design/design.go`: `spines:` in the frontmatter, and loading each
  area's spine.
- `harness/cmd/check/checks.go`, `main.go`, `spines_test.go`, `checks_test.go`: the
  rules above, and `-for`.
- `harness/cmd/coldread/main.go`, `main_test.go`: `-spine`.
- `harness/cmd/reverse/main.go`, `spine.go`, `spine_test.go`: the steps for one area.
- `harness/reverse/spine.md`, `harness/reverse/reverse.md`: what the sessions are told.
- `harness/templates/spine.md`, `DESIGN.md`, `CLAUDE.md`, `CLAUDE.nested.md`: the shape
  of a spine, the commented `spines:` block, and the rule to read only the area's spine.
- `README.md`, `harness/FRAMEWORK.md`, `design/DESIGN.md`: the method and this
  repository's design say what is now true; "spine" for the frontmatter is gone from
  them and from the code's comments and messages.
