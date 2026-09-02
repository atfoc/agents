# Implementing tasks

The loop that takes a store of tasks to done: fetch what is startable, spawn one implementer per
task, mark each completed, fetch again.

Every task is implemented in its own subagent — one subagent, one task, never two tasks in one and
never a task split across two. You run the loop and the store; the subagents write the code.

Every read and every write of a task goes through the task store's own operations. Which store, and
what those operations are, comes from whoever starts the run.

## The loop

1. **Fetch the startable tasks** — the ones whose blockers are all completed. That list is
   authoritative. Never reason about blocking yourself and never inspect the store by hand.
2. **Spawn one implementer subagent per startable task** not already in flight, all of them in
   parallel, in a single message. There is no cap; the cut already decided what may run together.
3. **Record the spawned ids in an in-flight ledger.** A running task is still pending and still
   startable, so without the ledger it gets spawned again on the next completion.
4. **On a success report** — mark that task completed, drop it from the ledger, fetch the startable
   list again, and immediately spawn anything newly startable without waiting for in-flight work to
   finish.

The only two things that change what is startable are the initial fetch and a completion.

Run no verification of your own. The implementer runs the task's verification, and a reported
verification failure means the task is **not** completed.

## How the task reaches the implementer

Never paste a body into the prompt. The subagent fetches its own body, so what you hand it is a
command that prints that one task's body and nothing else.

Work the command out once, before spawning anything:

- Take the store's own operation that prints **only** a body for a given task id, and substitute the
  values that store declares — the task id, plus a root folder, a project id, whatever else it asks
  for. Change nothing else about it.
- Write it literally: absolute paths, no variables, nothing relative to a working directory. The
  subagent's shell is not yours.
- If the store only offers whole-task retrieval — a command that also prints the id, title, blockers
  or status — write one small adapter script for this run. It takes a task id as its one argument,
  calls the store's own operation, and prints the body alone; it never reads or parses the store
  itself. What you hand implementers is then the call to that adapter. The adapter is never cleaned
  up and its path is never reported.
- If no command a subagent could run reaches a body at all, stop, report that this store cannot be
  driven this way, and spawn nothing.

Per task, the only thing that changes in that command is the id.

## The implementer prompt

An implementer sees one task body and nothing else — never the spec, never another task, never a
reason why. The whole prompt is this, with `<BODY COMMAND>` replaced by the command above carrying
that task's id:

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

Add nothing about what the task contains — never summarise, quote or preview a body, never name the
implementation spec or the feature definition, never mention that other tasks exist, never explain
why this task was picked now.

## Problems

- A one-or-two-line fix may be applied inline.
- Anything larger becomes a new task, with the affected existing tasks blocked by it.
- A task whose implementer failed is blocked by its own remediation task, so it leaves the startable
  set honestly instead of being respawned in a loop. If the remediation cannot be expressed as a
  task, exclude that task for the rest of the run and report it.
- Never edit a task's body. One failure never stops the run — in-flight work continues and the loop
  keeps spawning whatever is startable.

The only writes this loop makes: create a task, block a task, complete a task, and at most a
two-line code edit.

## Stop

- **Nothing pending** — report what was done, one line per completed task, plus everything that was
  reported and not resolved.
- **Something pending, nothing startable, nothing in flight** — report the stall, naming the pending
  tasks and what blocks them, and stop.
