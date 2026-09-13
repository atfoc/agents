# Running human tasks

One session that either lists the human tasks of a goal or runs one of them in the foreground.
Human tasks are the ones in the store without the `for-agent` tag.

Input: `{goalName}`, and optionally a task name. Read `./.tasks/{goalName}/dev-process.md` first —
`dev-process-state.md` — and take the store description from it verbatim; every store operation
uses it. No state file, or no store description in it: say which is missing, point at
`dev-process-orchestrator.md`, and stop. Never take a store description from the user directly.

## Listing

With no task name, or when asked what tasks there are:

- list the human tasks that are startable — every blocker completed;
- separately, list the human tasks still blocked, each with what blocks it;
- stop.

## Running one

With a task name:

- Fetch that task. It must be a human task whose blockers are all completed and which is not yet
  completed. A `for-agent` task is never run here, even when asked by name — say so and point at
  `running-agent-tasks.md`. A blocked task is never run — say what blocks it. Stop in either case.
- Fetch its body. The body names its kind and the doc that kind runs — `task-kinds.md` — and its
  output location. Run that doc in the foreground as the interactive session it describes, with
  the body as its input and the body's output location overriding the doc's default.
- When that session ends with its output written, mark the task completed, report the output
  path, and stop. If it ends without the output written, complete nothing; report where it
  stopped, and stop.

One task per session, never two.
