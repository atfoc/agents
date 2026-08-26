---
name: implement-tasks
description: Used when we need to implement existing tasks. Requires task operations in context, provided by the skill that invokes it
disable-model-invocation: true
---

# Implement tasks

This skill has no input of its own. The tasks come from the task format that is already in
context. This skill contains no knowledge of how tasks are stored — no files, no folders, no
frontmatter, no id schemes. Everything it does to a task, it does through the operations that
format gives it.

## Step 1 — Check you can run

The context must already give you all five of these:

- how to list the tasks that are startable now,
- how to mark a task completed,
- how an implementer fetches a task's full body, as a literal instruction you can hand over
  unchanged,
- how to create a new task,
- how to make an existing task blocked by another.

Plus any further input those operations declare (a root folder, a project id, whatever the format
asks for), with a value for each.

If any of the five is missing, or a declared input has no value, stop, report exactly which one is
missing, and spawn nothing.

## Step 2 — Fetch the startable tasks

Ask the context's operations for the tasks that are startable now. That list is authoritative.
Never reason about blocking yourself, never re-derive what is startable, never inspect the store.

## Step 3 — Spawn one subagent per startable task

Spawn one subagent for every startable task, all of them in parallel, in a single message. There
is no cap — the split already decided what may run together.

The prompt is **built, not templated**. Take the fetch-body instruction from the context,
substitute the values it declares — always the task id, plus whatever else that format needs — and
copy it into the prompt verbatim. Add nothing about what the task contains: never summarise, quote
or preview a body.

Then append the standing rules for the implementer:

- implement the task fully, including its verification,
- do no other task's work,
- never modify the task store and never mark anything completed,
- on a conflict, a gap, or anything the task does not cover, stop and report it rather than
  resolving it,
- report back what was done.

Record which task ids are in flight. A running task is still pending and still startable, so every
later fetch returns it; without that ledger the same task gets spawned again on the next
completion.

## Step 4 — On each report

On success: mark that task completed, fetch the startable list again, and immediately spawn
anything newly startable without waiting for in-flight work to finish. The only two things that
change what is startable are the initial fetch and a completion.

Run no verification of your own — the implementer runs the task's verification. A reported
verification failure means the task is **not** completed.

## Step 5 — Turn problems into tasks

A one-or-two-line fix may be applied inline. Anything larger becomes a new task, with the affected
existing tasks blocked by it.

A task whose implementer failed is blocked by its own remediation task, so it leaves the startable
set honestly instead of being respawned in a loop. If the remediation cannot be expressed as a
task, exclude that task for the rest of the run and report it.

Never edit a task's body. One failure never stops the run — in-flight work continues and the loop
keeps spawning whatever is startable.

The only writes this skill makes are: create a task, block a task, complete a task, and at most a
two-line code edit.

## Step 6 — Stop

Nothing pending → report what was done, one line per completed task, plus everything that was
reported and not resolved.

Something pending, nothing startable, nothing in flight → report the stall, naming the pending
tasks and what blocks them, and stop.
