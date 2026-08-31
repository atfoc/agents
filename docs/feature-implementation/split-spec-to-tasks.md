# Cutting an implementation spec into tasks

Going from an implementation spec to the set of slices that become tasks. What a slice is, and what
every slice must have, is `../vertical-slices.md` — this is the procedure for producing them.

Each slice becomes exactly one task: its title and its body, unchanged.

Read the implementation spec end to end, and the feature definition end to end, before cutting
anything.

## Cutting

- Work that cannot be verified on its own folds into the slice whose verification exercises it.
- **Never re-cut a slice to gain parallelism.** Keeping the change → verify → fix loop inside one
  implementer's context beats handing pieces around. Parallelism is only what falls out of slices
  that were already independent.
- Where a slice depends on work that is not cut yet, stub that branch rather than widening the
  slice to cover it.

## Writing each body

- Carry text from the implementation spec **verbatim**. No summarising, no rewording, no "see the
  spec".
- Carry feature-definition text verbatim wherever a slice needs its reason or its user-visible
  behaviour.
- You may add your own detail, and added detail only narrows or sequences what the documents
  already decided.
- Never add anything that contradicts, weakens or reinterprets the implementation spec or the
  feature definition. If a workable slice would require that, stop, write nothing, and report what
  conflicts with what.
- Never name the implementation spec's or the feature definition's path in a body. The body has to
  stand alone.

## Deciding blockers

Apply the two blocking reasons to every pair of slices. Where two slices touch the same file and
the order is arbitrary, pick one and say nothing more about it.

A cycle means a bad cut: merge the slices or cut them differently. A cycle that cannot be resolved
is a stop-and-report.

## Check before writing anything

All four, before a single slice is written down:

- Every section of the implementation spec appears in at least one body. An orphaned section is a
  stop, not a silent drop.
- No cycles.
- At least one slice with no blockers.
- Nothing contradicted.

Any failure → write nothing and report why.

## Report

How many slices, every piece of detail you added beyond the source documents, and anything you had
to stop on. Never report the bodies back — they are the slices.
