# Vertical slices

A **vertical slice** is a collection of work that builds one functional and testable unit of a
whole implementation spec.

## What every slice has

- **Its own verification.** A slice is done when its verification passes. There are no
  verification-only slices, and no slice that is verified by another slice.
- **A body that is the whole assignment.** Whoever implements a slice sees that body and nothing
  else — not the implementation spec, not the feature definition, not any other slice, not a reason
  why it was cut this way.
- **A title**, one line.
- **Blockers** — the slices that must be completed before it may start.

## Stubs

A slice that needs another slice's work to be functional cuts that branch with a stub. The body
names the stub explicitly, so the slice that later replaces it knows what it is replacing.

## Blockers

A slice is blocked by another for exactly two reasons:

1. It consumes an artifact the other produces — a type, a table, a stub it replaces, a module it
   imports.
2. The two would edit the same file and so cannot run at the same time.

Nothing else is a blocker. A slice is startable when every slice blocking it is complete.

## The set as a whole

- The graph of blockers is **acyclic**, and at least one slice has no blockers.
- **The deduplicated union of all bodies is the whole implementation spec.** Every section of the
  spec appears in at least one body.
- **Duplication between slices is expected and correct.** A section that applies to four slices is
  copied into all four; a slice is never made smaller by pointing at another slice's body.
