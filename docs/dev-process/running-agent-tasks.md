# Running agent tasks

One session that runs every startable `for-agent` task of a goal, one same-agent subprocess per
task, until nothing `for-agent` is startable.

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

- Spawn each same-agent subprocess so that it can spawn same-agent subprocesses of its own. A split
  task runs two subprocesses; an implement task runs one per slice.
- Each same-agent subprocess is told to work through the `with-docs` skill, in addition to its
  task. The skill is named; no command is.
- A same-agent subprocess that reports a problem still ends as a completed task. Its output must
  record the problem: if the subprocess did not write it, write the subprocess's report into that
  task's output location, then complete the task.
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

- **Nothing `for-agent` is startable and nothing is in flight** — report one line per completed
  task, with what it concluded or what problem it recorded, and point at
  `dev-process-orchestrator.md` for the next iteration. Where a `for-agent` task is still pending
  behind a human task, say so: it waits on the user, through `running-human-tasks.md`.
