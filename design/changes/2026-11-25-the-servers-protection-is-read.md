# The server's protection is read before a push

## The problem

Nothing reaches `main` except through a pull request that passed the checker
(INV-011). On a person's machine the hook and `safe-push` hold that. On the server it
is held by one setting, the branch's protection, made by hand at adoption, and nothing
in the harness ever looks at it. This repository showed both ways it goes wrong in one
day: it was public with `main` unprotected, and when the protection was added by hand
the `harness` check was not on it, so a pull request with a failing check could have
merged. Every local refusal passed throughout.

## The plan

A spine key, `protection: required`, and a command, `cmd/protection`, that `safe-push`
runs before it pushes (ADR-0011). It needs three things of the default branch: a pull
request required, the `harness` status check required, a force push refused. It reads
them from the branch's own protection and from the rulesets that apply to it, through
`gh` as whoever is logged in.

- Something lacking, read as an administrator: the push is refused, each thing named,
  and the command that adds them printed.
- The protection not shown to whoever asked, which is anyone but an administrator: one
  line saying so, and the push goes on.
- The server cannot be asked: refused.
- The spine does not ask: nothing is read, nothing is printed.

`cmd/protection -set` adds what is lacking to the protection that is there and sends
everything else back as it was; `-n` prints what it would send. It is never run by
another command.

This repository's own spine asks for it.

A project that already has the harness need do nothing: after `bootstrap --update`
the key is there to switch on, and off until someone does.

## Alternatives rejected

- **A check in `cmd/check`**, as the gap table had it. CI's token is not shown a
  branch's protection, and the checker runs on hooks where a call to a server has no
  place.
- **Refuse whoever cannot read the protection.** It would lock out every contributor
  who is not an administrator.
- **Set it from `bootstrap`.** A setting on a server changed by a command that was
  asked to adopt a method.
- **Replace the protection with a known-good one.** It would wipe a team's approvals,
  restrictions and other checks.

## Files

- `harness/cmd/protection/protection.go`, `main.go`, `protection_test.go`: the command.
- `harness/internal/design/design.go`: `protection:` in the spine.
- `harness/safe-push`: runs it before the push.
- `harness/safe_push_test.go`: the push refused and allowed, with a stand-in for `gh`.
- `harness/templates/DESIGN.md`: the key, commented out.
- `harness/FRAMEWORK.md`, `README.md`: how to set it and switch it on.
- `design/DESIGN.md`: ADR-0011, INV-011, the gap table, the spine.
