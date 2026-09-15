# Asana tasks

A task is an Asana task; a subtask is an Asana subtask. Running the script, referring to
workspaces, projects and tasks, and handling errors are in `access.md`.

Whoever uses this doc supplies the scope the tasks live in: a project, or a parent task whose
subtasks are the tasks. If you have neither, report that the scope is missing and stop.

## The task

Every command that returns a task prints these fields:

- `id` — the task's gid. Assigned by Asana on create, so tasks must be created in dependency order.
- `title` — one line; the task name.
- `completed` — `true` when the task is marked complete.
- `parent` — the parent task's id, or `null`.
- `blockedBy` — the ids of the tasks this one depends on through Asana's dependencies.
- `tags` — the task's Asana tags, by name.
- `url` — the task in Asana.

The body is the task description, in markdown. Asana shows it as plain text.

Rules:

- **Pending** — not completed. Asana has no canceled state.
- **Startable** — pending, and every task in `blockedBy` is completed. Every blocker counts, inside
  the scope or not, whatever tags either task carries.
- **Tags** never affect blocking; they only narrow which tasks a listing shows. A tag with no
  matching tag in the task's workspace is created there the first time it is used.
- **Subtasks** are independent of their parent for blocking: a parent's blockers do not block its
  subtasks, and open subtasks do not stop the parent from completing.
- **A subtask is in no project** unless it is created with `--project`. A project listing does not
  show the subtasks of its tasks; list those with `--parent`.
- Asana allows a task at most 30 blockers and blocked tasks combined; past that, blocking fails.

## Operations

- **Create a task** — `SCRIPT create --project <project> [--workspace <workspace>] --title
  "<title>" [--blocked-by <id> …] [--tag <tag> …]`, body on **stdin**; prints the created task with
  its new id.
- **Create a subtask** — `SCRIPT create --parent <id> --title "<title>" [--blocked-by <id> …]
  [--tag <tag> …]`, body on **stdin**. `--project <project>` also puts it in that project.
- **Make an existing task a subtask** — `SCRIPT set-parent --id <id> --parent <id>`.
- **One task in full** — `SCRIPT get --id <id>`; the task plus `workspace`, `projects` (their
  names), `subtasks`, `blocks` (the ids it blocks), `documents` and `links`.
- **Fetch a task's body** — `SCRIPT body --id <id>`.
- **Listings** — each takes a scope: `--parent <id>` for that task's direct subtasks, or
  `--project <project> [--workspace <workspace>]` for the project's tasks. Tasks come in the order
  Asana shows them.
  - **All tasks** — `SCRIPT list <scope>`.
  - **Not-completed tasks** — `SCRIPT list-pending <scope>`.
  - **Startable tasks** — `SCRIPT list-unblocked <scope>`.
  - **Only the tasks carrying a tag** — add `--tag <tag>` to any listing, repeatable; a task is
    shown only when it carries every given tag.
  - **Is the scope empty** — `SCRIPT list <scope>` prints `[]`.
- **Complete a task** — `SCRIPT complete --id <id>`; marks it complete. Fails while any blocker is
  not completed, although Asana itself would allow it. A task already completed is printed
  unchanged.
- **Block a task** — `SCRIPT block --id <id> --blocked-by <id> …`; a blocker already present is
  skipped. Blocking a task by itself, or in a way that creates a cycle, fails.
- **Unblock a task** — `SCRIPT unblock --id <id> --blocked-by <id> …`; fails when a given task
  does not block it.
- **Tag a task** — `SCRIPT tag --id <id> --tag <tag> …`; adds, never removes.
- **Replace a task's body** — `SCRIPT update --id <id> [--title "<title>"]`, the full new body on
  **stdin**; prints the task. It replaces, never appends: to change part of a body, fetch it with
  `body`, then write back the whole of it. Blockers, tags and parent are untouched.

`body` prints the body alone — no id, title, blockers or status — so it can be given to someone who
must see only the body.

Documents written to or attached to a task are in `documents.md`.
