---
name: spec-to-tasks
description: Used when we need to split implementation spec into tasks. Requires a task format in context, provided by the skill that invokes it
disable-model-invocation: true
---

# Split an implementation spec into tasks

Your input is a path to an implementation spec — the argument given to this skill, a file named in
the conversation, or one an invoking skill passed in. If there is none, ask for it and stop. This
skill never stores tasks itself and never learns how they are stored.

## Step 1 — Check you can create tasks

The context must already contain a task format that gives you all of:

- how to create a task with a title, a body and a list of blocking task ids, returning that task's
  id;
- where ids come from;
- how to ask whether the task store exists;
- how to ask whether it is empty;
- any further input that format declares (a root folder, a project id).

Rules:

- If any of those is missing, or a declared input has no value, stop, report exactly which one is
  missing, and create nothing.
- Never infer a format. Never invent a store. Never fall back to writing files of your own design.

Then apply the store precondition:

- The store does not exist → proceed.
- The store exists and is empty → proceed.
- The store exists and is not empty → stop, create nothing, and report which store it is, that it
  already holds tasks, and that re-splitting requires emptying it first.

## Step 2 — Read the sources in full

Read the implementation spec end to end. If a feature definition was given, or a
`feature-design.md` sits beside the spec, read it too. Do not begin cutting before both have been
read completely.

## Step 3 — Cut into vertical slices

A slice is a collection of work that builds one functional and testable unit of the whole
implementation spec.

- A slice that needs another slice's work to be functional cuts that branch with a stub, and the
  body names the stub explicitly so the slice that later replaces it knows what it is replacing.
- Every slice carries its own verification. A slice is done when its verification passes, so there
  are no verification-only slices.
- Work that cannot be verified alone folds into the slice whose verification exercises it.
- **Never re-cut a slice to gain parallelism.** Keeping the change → verify → fix loop inside one
  implementer's context beats handing pieces around. Parallelism is only what falls out of slices
  that were already independent.

## Step 4 — Write each task body

Check each of these:

- Carry text from the implementation spec verbatim — no summarising, no rewording, no "see the
  spec".
- The union with deduplication of all task bodies reproduces the whole implementation spec.
- Duplication across tasks is expected and correct: a section that applies to four tasks is copied
  into all four.
- Feature-definition text is carried verbatim wherever a slice needs its reason or its
  user-visible behaviour.
- The split may add its own detail, and added detail only narrows or sequences what the documents
  already decided.
- The split may never add anything that contradicts, weakens or reinterprets the implementation
  spec or the feature definition. If a workable slice would require that, stop, create nothing,
  and report what conflicts with what.
- Never name the spec's or the feature definition's path in a body — the implementer gets one task
  and nothing else.

## Step 5 — Decide blockers

A task's blockers are the tasks that must be completed before it may start, for exactly two
reasons:

1. It consumes an artifact another task produces — a type, a table, a stub it replaces, a module
   it imports.
2. The two would edit the same file and so cannot run at the same time. Same file, one blocks the
   other; when the order is arbitrary, pick one and say nothing more about it.

The graph must be acyclic. A cycle means a bad cut, so merge or re-cut; a cycle that cannot be
resolved is a stop-and-report. At least one task must have no blockers.

## Step 6 — Check before creating anything

All four, before the first task is created:

- Every section of the implementation spec appears in at least one drafted body. An orphaned
  section is a stop, not a silent drop.
- No cycles.
- At least one startable task.
- Nothing contradicted.

Any failure → stop and create nothing.

## Step 7 — Create in dependency order

Because ids are assigned at creation, a task is created only after every task blocking it exists
and its id is known. The DAG guarantees such an order exists.

## Step 8 — Report and stop

Report:

- where the tasks were created and how many;
- one line per task with its id, its title and its blocker ids;
- which tasks are startable now;
- every piece of detail the split added beyond the source documents, called out so the user can
  check it.

Then stop. Never begin implementing and never invoke an execution skill, even when the user asked
for both at once.
