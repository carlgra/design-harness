You are a session with no memory. You have never seen this repository, its code, or any
conversation about it. Below, after the first line of dashes, is its design document:
what it is for, how it works, what must always hold, every decision in force with its
reason and what was rejected, and which files realise it. After the second line of
dashes is a description of a change. You are handed these and nothing else.

This is the readiness test: could you build the change from the design alone?

You are about to be asked to implement that change, in one pass, with no one to ask.
Before you start, answer:

1. What would you build, in which files, and why those files? Name the shape that claims
   each.
2. Which decisions bind the change, and which invariants it must keep?
3. What does the design leave you to decide that it should have decided? List each, and
   say whether you could choose safely or would have to guess.
4. What would you have to read that you were not given?

Then judge: do you have everything you need to implement this change in one pass and
get it right, from the design alone? If yes, the design passes this test. If you would
have to guess at something that changes what you build, or read something you were not
given, it fails.

A design holds what must be governed, not every detail of the code. A choice it leaves
to whoever builds, which no rule binds and which could safely go either way, is not a
gap; a rule the change would have to keep and the design does not state is.

A change that is itself to `design/` is implemented by making the design say what the
change says; judge whether what you were given lets you see that the change is complete
and consistent with the rest.

End your reply with exactly one line, and nothing after it:

VERDICT: PASS

or

VERDICT: FAIL

---
