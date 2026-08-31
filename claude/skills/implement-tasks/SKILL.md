---
name: implement-tasks
description: Used when we need to implement existing tasks. You say where the tasks live and which skill or doc describes that storage
argument-hint: [where-the-tasks-live]
disable-model-invocation: true
---

# Implement tasks

You drive the task store yourself for the whole run, through the format the user's input names.
Implementers see one task body and nothing else — never the spec, never another task, never a
reason why.

## Step 1 — Take the storage input

Your one input is the user's own statement of where the tasks live and what describes that format
— a folder plus "use the `local-task-format` skill", a Linear project plus the skill or doc that
covers Linear, anything of that shape. Take it from the skill argument or from the conversation.

Check only that it is *present*. If it is missing, ask for it and stop. Never default to a folder,
never go looking for something task-shaped, never invent a store.

Load whatever skill or doc it names and follow it. Every read and every write of a task for the
rest of this run goes through that format's own operations — never your own file reads or edits.

Pre-validate nothing else. Do not check the location exists, do not check the format offers the
operations you are going to need, do not count its capabilities. A wrong or insufficient input
surfaces as a hard error from the first real operation; when it does, stop, report that error
exactly as it came, and never work around it.

## Step 2 — Work out how an implementer fetches a body

An implementer receives the task's body and nothing else. You never paste a body into a prompt —
you hand over a command the implementer runs itself.

From the format you loaded, take the command that prints **only** a task's body for a given task
id, substituting the values that format declares — the task id, plus a root folder, a project id,
whatever else it asks for — and changing nothing else about it.

The command you hand over must be literal: absolute paths, no variables, nothing relative to your
working directory. The implementer's shell is not yours.

If the format only offers whole-task retrieval — a command that also prints the id, title,
blockers, status or any other field — write one small adapter for this run:

    ADAPTER="$(mktemp "${TMPDIR:-/tmp}/task-body-XXXXXX.sh")"

It takes a task id as its one argument, calls the format's own retrieval, and prints the body
alone. It calls the format's operations; it never reads or parses the store itself. What you hand
implementers is then `sh "<adapter path>" <task id>`.

The adapter is never cleaned up and its path is never reported to the user.

If no command an implementer could run reaches a body at all, stop, report that this format cannot
be driven this way, and spawn nothing.

## Step 3 — Fetch the startable tasks

Ask the format's operations for the tasks that are startable now. That list is authoritative.
Never reason about blocking yourself, never re-derive what is startable, never inspect the store
by hand.

## Step 4 — Spawn one subagent per startable task

Spawn one subagent for every startable task that is not already in flight, all of them in
parallel, in a single message. There is no cap — the split already decided what may run together.

Build each prompt: take the body command from Step 2, substitute that task's id, and put it in as
the literal command. Add nothing about what the task contains — never summarise, quote or preview
a body, never name the implementation spec or the feature definition, never mention that other
tasks exist, and never explain why this task was picked now.

The prompt is exactly this, with `<BODY COMMAND>` replaced:

```
Read your task by running:

    <BODY COMMAND>

That is your whole assignment.

- Implement it fully, including its verification.
- Do no other task's work.
- Never modify the task store and never mark anything completed.
- On a conflict, a gap, or anything the task does not cover, stop and report it rather than
  resolving it.
- Report back what was done.
```

Record the ids you just spawned in the in-flight ledger. A running task is still pending and still
startable, so every later fetch returns it; without that ledger the same task gets spawned again
on the next completion.

## Step 5 — On each report

On success: mark that task completed through the format's operations, drop it from the in-flight
ledger, fetch the startable list again, and immediately spawn anything newly startable without
waiting for in-flight work to finish. The only two things that change what is startable are the
initial fetch and a completion.

Run no verification of your own — the implementer runs the task's verification. A reported
verification failure means the task is **not** completed.

## Step 6 — Turn problems into tasks

A one-or-two-line fix may be applied inline. Anything larger becomes a new task, created through
the format's operations, with the affected existing tasks blocked by it through the format's
operations.

A task whose implementer failed is blocked by its own remediation task, so it leaves the startable
set honestly instead of being respawned in a loop. If the remediation cannot be expressed as a
task, exclude that task for the rest of the run and report it.

Never edit a task's body. One failure never stops the run — in-flight work continues and the loop
keeps spawning whatever is startable.

The only writes this skill makes are: create a task, block a task, complete a task, and at most a
two-line code edit.

## Step 7 — Stop

Nothing pending → report what was done, one line per completed task, plus everything that was
reported and not resolved.

Something pending, nothing startable, nothing in flight → report the stall, naming the pending
tasks and what blocks them, and stop.
