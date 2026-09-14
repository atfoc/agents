# Running agent tasks

One session that runs every startable `for-agent` task of a goal, one same-agent subprocess per
task, until nothing `for-agent` is startable that this session has not already failed.

Input: `{goalName}`. Read `./.tasks/{goalName}/dev-process.md` first — `dev-process-state.md` —
and take the store description from it verbatim; every store operation uses it. No state file, or
no store description in it: say which is missing, point at `dev-process-orchestrator.md`, and
stop. Never take a store description from the user directly.

## The loop

The loop is the one in `../feature-implementation/implementing-tasks-workflow.md`, run on the
store the state file describes: fetch the startable `for-agent` tasks, spawn one same-agent
subprocess per task not already in flight, all in parallel, keep an in-flight ledger, complete a
task on its success report, fetch again. The body reaches the subprocess the same way: a command
that prints only that task's body, worked out once before spawning anything. What differs:

- A same-agent subprocess is a new top-level session, started through the script in `scripts/`
  for the agent this session runs in — `run-claude-task.sh` under Claude Code,
  `run-cursor-task.sh` under Cursor. Never spawn it as a subagent: a subagent cannot spawn
  subagents of its own, and a split task runs two, an implement task one per slice. The rules for
  a run are in `running-a-task-subprocess.md`.
- Each same-agent subprocess is told to work through the `with-docs` skill, in addition to its
  task. The skill is named; no command is.
- Only a subprocess whose report says the task is done, with its output written, completes the
  task. A subprocess that stops on a problem, exits without a report, or otherwise does not
  finish its task has **failed** it, and the task stays uncompleted. Its report is still written:
  if the subprocess did not write one, write what it reported — or that it exited without
  reporting — into that task's output location. Then move the id from the in-flight ledger to a
  failed set kept for the rest of this session. A failed task is still pending and still
  startable, so without the failed set it is spawned again on the next fetch; a task is never
  respawned by the session that failed it.
- A failed task keeps blocking everything that depends on it; nothing here works around that.
  It is retried by starting a new session on the same goal: the task is still startable, so the
  loop picks it up like any other, and its report from the last attempt is at its output
  location.
- This session never creates a task, never blocks a task, and never edits a body. Remediation is
  the orchestrator's job on the next iteration. The `Problems` section of the implementing loop
  does not apply here; the only store write this session makes is completing a task.

The store in the state file is the orchestrator's. Only an `implement tasks` same-agent subprocess
runs the implementing loop as written, and only on the slice store its body describes — never on
the orchestrator's store.

## The same-agent subprocess prompt

The whole prompt, with `<BODY COMMAND>` replaced by the command that prints that task's body:

```
Read your task by running:

    <BODY COMMAND>

That is your whole assignment. Work it through the `with-docs` skill.

- Do it fully, and write your output where the task says.
- Do no other task's work.
- Never mark anything completed, and never change the store your task came from. A store your
  task tells you to create or work is yours to write.
- On a problem the task does not cover, write what you found and what is missing to your output
  location, then stop and report it rather than resolving it.
- Report back what was done.
```

Add nothing about what the task contains — never summarise, quote or preview a body, never
mention that other tasks exist, never explain why this task was picked now.

## Stop

- **Nothing `for-agent` is startable outside the failed set and nothing is in flight** — report
  one line per completed task, with what it concluded, and one line per failed task, with where
  its report is and which tasks it blocks. Failed tasks are retried by a new session on the same
  goal; for everything else point at `dev-process-orchestrator.md` for the next iteration. Where
  a `for-agent` task is still pending behind a human task, say so: it waits on the user, through
  `running-human-tasks.md`.
