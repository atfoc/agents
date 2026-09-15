# Dev process

Taking a vague, large chunk of work, understanding it, then breaking it into efforts that each go
through feature definition, implementation definition, split, and implement: an orchestrator that
turns that road into tasks, a runner for the tasks agents do, and a runner for the tasks a person
does.

## Docs

- `dev-process-orchestrator.md` — one iteration of the planning loop: read the state, check the
  store, read what finished, decide and create the next tasks, report. Use when the user has a
  goal to drive to done, or when finished tasks need the next step planned.
- `running-agent-tasks.md` — one session that runs every startable `for-agent` task of a goal, one
  subagent per task, until nothing is startable. Use when an orchestrator iteration reported
  `for-agent` tasks to run.
- `running-a-task-subprocess.md` — starting a task as a new non-interactive session of the same
  agent through the script for that agent: how to call it, where the report lands, what the
  subprocess sees. Use when a task has to run as a subprocess rather than a subagent.
- `running-human-tasks.md` — listing the human tasks of a goal and running one of them in the
  foreground. Use when the user asks what tasks wait on them, or names one to work.
- `dev-process-state.md` — the goal folder and the state file: where a goal's state, task outputs
  and store live, what the state file holds, and what a store description must say.
- `task-kinds.md` — the six task kinds: who runs each, the doc each runs, and what each outputs.

## Helper directories

- `scripts/` — `run-claude-task.sh`, `run-cursor-task.sh` and `run-pi-task.sh`, one per agent, each
  starting a new non-interactive session on a prompt. Run them; do not read them for information.
