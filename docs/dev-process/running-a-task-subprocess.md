# Running a task subprocess

Starting a task as a new, non-interactive session of the same agent — a subprocess, not a
subagent — so that the task can spawn subagents of its own and several tasks can run at once.

## The scripts

One script per agent, in `scripts/`. Use the one for the agent the current session runs in:

- `scripts/run-claude-task.sh` — under Claude Code.
- `scripts/run-cursor-task.sh` — under Cursor.

Each takes the prompt as its arguments, or on stdin when given no arguments, and starts a session
that runs the prompt to completion without asking anyone anything. Its stdout is the session's
final message — the report — and its exit code is the agent's. The scripts are run, never read for
information.

## Running one

- Run the script with the working directory set to the project root, so the session sees the
  project's docs, skills and task folders. Call the script by its absolute path.
- Put the prompt on stdin, from a file written for that run. A prompt on the command line is fine
  for a one-liner; a multi-line prompt is not.
- Run it in the background and capture stdout to a file per task, so the report survives the
  session and can be read once the subprocess exits. Never wait for one subprocess in the
  foreground when others could be started.
- Start every task that may run now at once. The scripts do not serialise anything; whatever must
  not run together has to be kept apart by whoever starts them.
- A subprocess reports back only when it exits. A non-zero exit code with no report is a problem to
  record, not a task to retry.

## What a subprocess sees

- The same project, the same docs and the same skills as the session that started it.
- Nothing of the parent session's conversation. Everything it needs has to be in the prompt or
  reachable from it.
- It can spawn subagents, and it can start subprocesses of its own through the same scripts.
