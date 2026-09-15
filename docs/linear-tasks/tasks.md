# Linear tasks

A task is a Linear issue; a subtask is a sub-issue. Running the script, referring to teams,
projects and tasks, and handling errors are in `access.md`.

Whoever uses this doc supplies the scope the tasks live in: a team key with an optional project,
or a parent issue whose sub-issues are the tasks. If you have neither, report that the scope is
missing and stop.

## The task

Every command that returns a task prints these fields:

- `id` — the issue identifier, such as `ENG-12`. Assigned by Linear on create, so tasks must be
  created in dependency order.
- `title` — one line.
- `state` — the name of the issue's workflow state.
- `completed` — `true` when that state is of Linear's completed type (Done, or any other completed
  state the team has).
- `parent` — the parent task's id, or `null`.
- `blockedBy` — the ids of the tasks that block this one through Linear's "blocks" relation.
  Related and duplicate relations are not blockers.
- `tags` — the issue's Linear labels, by name.
- `url` — the issue in Linear.

The body is the issue description, in markdown.

Rules:

- **Pending** — the state is neither completed nor canceled.
- **Startable** — pending, and every task in `blockedBy` is completed. Every blocker counts, inside
  the scope or not, whatever tags either task carries. A canceled blocker is not completed: it
  keeps blocking until it is removed with `unblock`.
- **Tags** never affect blocking; they only narrow which tasks a listing shows. A tag with no
  matching label is created as a label on the task's team the first time it is used.
- **Subtasks** are independent of their parent for blocking: a parent's blockers do not block its
  subtasks, and open subtasks do not stop the parent from completing.

## Operations

- **Create a task** — `SCRIPT create --team <key> [--project <project>] --title "<title>"
  [--blocked-by <id> …] [--tag <tag> …]`, body on **stdin**; prints the created task with its new
  id.
- **Create a subtask** — `SCRIPT create --parent <id> --title "<title>" [--blocked-by <id> …]
  [--tag <tag> …]`, body on **stdin**. It takes the parent's team and project. `--team <key>` puts
  it in another team, without the parent's project; `--project <project>` sets the project.
- **Make an existing task a subtask** — `SCRIPT set-parent --id <id> --parent <id>`.
- **One task in full** — `SCRIPT get --id <id>`; the task plus `team`, `project`, `subtasks`,
  `blocks` (the ids it blocks), `documents` and `links`.
- **Fetch a task's body** — `SCRIPT body --id <id>`.
- **Listings** — each takes a scope: `--parent <id>` for that task's direct subtasks, or
  `--team <key> [--project <project>]`.
  - **All tasks** — `SCRIPT list <scope>`.
  - **Not-completed tasks** — `SCRIPT list-pending <scope>`.
  - **Startable tasks** — `SCRIPT list-unblocked <scope>`.
  - **Only the tasks carrying a tag** — add `--tag <tag>` to any listing, repeatable; a task is
    shown only when it carries every given tag.
  - **Is the scope empty** — `SCRIPT list <scope>` prints `[]`.
- **Complete a task** — `SCRIPT complete --id <id>`; moves it to the team's first completed state.
  Fails while any blocker is not completed. A task already completed is printed unchanged.
- **Block a task** — `SCRIPT block --id <id> --blocked-by <id> …`; a blocker already present is
  skipped. Blocking a task by itself, or in a way that creates a cycle, fails.
- **Unblock a task** — `SCRIPT unblock --id <id> --blocked-by <id> …`; fails when a given task
  does not block it.
- **Tag a task** — `SCRIPT tag --id <id> --tag <tag> …`; adds, never removes.

`body` prints the body alone — no id, title, blockers or status — so it can be given to someone who
must see only the body.

Documents written to or attached to a task are in `documents.md`.
