# <What changes, as a sentence that will be true afterwards>

<!-- Save as design/changes/<YYYY-MM-DD>-<slug>.md before touching code. Short is
     right: a note is read once at review and kept as the record of why. Every
     section says something; "None." is an answer, an empty section is not. -->

## The problem

<!-- What is wrong or missing, as someone meets it. What was seen, where. -->

## The plan

<!-- What will be true afterwards, and how. Name the decisions and invariants
     this rests on by id, and any it adds, amends or reverses. -->

## Alternatives rejected

<!-- What else would have solved it, and why not. This is the part nobody can
     reconstruct later. -->

<!-- In a contracts repository only, add a section with a fenced block whose info
     string is `servers`: each RPC and request field this change adds, the
     repository that serves it and the ones that call it, by their paths on the
     root's map; or `[]` when it adds none.

         ```servers
         - name: billing.v1.BillingService.CreateInvoice
           server: billing
           callers: [ledger]
         ```
-->

## Files

<!-- Each file to add or change, and why that file. A new path must be under a
     glob in design/DESIGN.md's files:, or the note says which glob it adds. -->
