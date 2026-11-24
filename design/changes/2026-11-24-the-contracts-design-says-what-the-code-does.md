# The contracts design says what the code does

## The problem

A session handed `design/DESIGN.md` alone was asked whether it could build the
contracts rules from it (ADR-0010) and said no. What it would have had to choose, each
choice safe but none written down: the name of a definition in a file with no
`package`; how deep `gen/`, `vendor/` and `node_modules/` exclude; what `elephant
where` prints; the name of the secret for a private contracts repository; how a
repository the map does not list yet is held; whether a server may be its own caller;
and which contracts repository is the one when the first listed is planned.

One of them was also a fault. The code excluded any path containing `vendor/` or
`node_modules/` as text, so a definition under `vendors/v1/` was not read at all.

## The plan

A sentence in *How it works* for each, saying what the code does. The exclusion is
by whole directory name: `gen/` at the top of the repository, `vendor` and
`node_modules` at any depth. The nested CI workflow's comment suggests a name for the
contracts repository's secret, as it does for the root's. No decision changes.

## Alternatives rejected

- **Leave them to the code.** The design is what a change is built from; a reader who
  has to open the code to learn a rule has no design for it.
- **Exclude `gen/` at any depth, like the other two.** `gen` is an ordinary word for a
  directory a person writes in, a generator's own source for one; `vendor` and
  `node_modules` are named by the tools that fill them.
- **Fix a name for the secrets.** The harness never reads them; the workflow belongs
  to the project once written, and a project may already have one that fits.

## Files

- `design/DESIGN.md`: the seven sentences.
- `harness/cmd/check/contracts.go`: `isProto` excludes by directory name.
- `harness/cmd/check/contracts_test.go`: `TestTheProtoScanFindsRPCsAndRequestFields`
  walks a file with no package and the exclusions.
- `harness/templates/harness.nested.yml`: the comment names a secret for the contracts
  repository.
