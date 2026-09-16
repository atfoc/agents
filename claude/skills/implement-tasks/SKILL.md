---
name: implement-tasks
description: Implements every for-agent task by looping over the startable tasks, spawning one implementer subagent per task in parallel, completing each on success and fetching again until nothing is left. Use when tasks already exist and you want them implemented.
argument-hint: "[tasks location]"
---

# Implement tasks

This skill acts on the tasks at the location the user provides, `$ARGUMENTS`, or the location given earlier in the conversation. If no location is given, ask for it and stop. Never default or invent one.

The tasks to implement are marked with the tag `for-agent`.

You run the loop and manage the tasks. Subagents write the code: one subagent per task, never two tasks in one, never one task across two.

## Hard rules

- Never reason about blocking yourself; the startable list decides what can run.
- Work only tasks tagged `for-agent`. Any other task is someone else's: never fetch it as work, spawn it or complete it. It still blocks.
- Run no verification yourself. A reported verification failure means the task is not completed.
- Never edit a task's body.
- The only writes you make: create a task and tag it, block a task, complete a task, and code edits of at most two lines.

## 1. Work out the body command

Before spawning anything, settle the one command that prints a single task's body and nothing else:

- Use the command that reads a task body by task id, with every value it requires filled in. Change nothing else.
- Write it literally: absolute paths, no variables, nothing relative to a working directory.
- If reading a task prints the whole task (id, title, blockers, status along with the body), write a small adapter script for this run in the scratchpad. It takes the task id as its one argument, reads the task and prints the body alone. The body command is then a call to that adapter. Do not clean it up and do not report its path.
- If no command a subagent could run reaches a body, report that the task bodies cannot be read this way, spawn nothing, and stop.

Per task, only the id in this command changes.

## 2. Run the loop

1. **Fetch the startable tasks tagged `for-agent`.** That list is authoritative. If the startable list cannot be filtered by tag, fetch it whole and keep only entries tagged `for-agent`.
2. **Spawn one implementer subagent per startable task not already in flight**, all in parallel in a single message, with no cap.
3. **Record the spawned ids in an in-flight ledger.** A running task is still startable; the ledger stops it being spawned twice.
4. **On a success report**, mark that task completed, drop it from the ledger, fetch the startable list again, and immediately spawn anything newly startable without waiting for other in-flight work.

Only the initial fetch and a completion change what is startable.

## 3. The implementer prompt

The whole prompt is exactly this, with `<BODY COMMAND>` replaced by the body command carrying that task's id:

```
Read your task by running:

    <BODY COMMAND>

That is your whole assignment.

- Implement it fully, including its verification.
- Do no other task's work.
- Never create, block, tag or complete tasks.
- On a conflict, a gap, or anything the task does not cover, stop and report it rather than
  resolving it.
- Report back what was done.
```

Add nothing else. Never paste, summarise, quote or preview a body; never name the spec or feature definition; never mention other tasks; never explain why this task was picked.

## 4. Handle problems

- A one-or-two-line fix may be applied inline.
- Anything larger becomes a new task, tagged `for-agent`, with the affected existing tasks blocked by it.
- A task whose implementer failed or reported a verification failure is blocked by its own remediation task, so it leaves the startable set instead of being respawned. If the remediation cannot be expressed as a task, exclude that task for the rest of the run and report it.
- One failure never stops the run: in-flight work continues and the loop keeps spawning whatever is startable.

## 5. Stop

- **Nothing pending tagged `for-agent`** — report one line per completed task, plus everything reported and not resolved.
- **Something pending tagged `for-agent`, nothing startable, nothing in flight** — report the stall, naming each pending task and what blocks it. Where a blocker is untagged, say it belongs to someone else and the run can be started again once they complete it.

The skill is finished when one of these stop conditions is reported; stop there.
