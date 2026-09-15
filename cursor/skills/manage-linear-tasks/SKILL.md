---
name: manage-linear-tasks
description: Creates, reads, updates, blocks, tags and completes Linear issues and sub-issues as tasks, and writes, links and reads Linear documents attached to them, through a bundled script. Use when you want to create Linear tasks or subtasks, list startable or pending Linear issues, complete or block a Linear task, or attach a document to a Linear issue.
---

# Manage Linear tasks

Act on the request in the skill argument, or the Linear work named earlier in the conversation. If there is no request, ask what to do in Linear and stop. Any request that creates or lists tasks needs a scope: a team key with an optional project, or a parent issue whose sub-issues are the tasks. If the scope is missing, report that and stop.

## Access

1. Every read and every write of Linear, issues and documents alike, goes through `scripts/linear.py`. Never reach Linear any other way — no other API calls, no MCP servers, no browser.
2. `SCRIPT` below and in `references/` stands for `python3 scripts/linear.py`, written out as that literal absolute path. Use the literal path in every command, including any command you hand to someone else: no shell variables, nothing relative to a working directory. The script needs only Python 3. Run it; do not read it for information.
3. The script reads a Linear personal API key from `LINEAR_API_KEY`. When a command fails with `LINEAR_API_KEY is not set`, report that the user must create a key in Linear under Settings → Security & access → Personal API keys and export it in the environment the script runs in, then stop. Never ask for the key in the conversation, and never write it into a file or a command line.
4. Pass bodies and document content on stdin with a quoted heredoc, such as `SCRIPT update --id ENG-1 <<'EOF'` … `EOF`. Empty stdin fails.

## Referring to things

- **Task** — its issue identifier, such as `ENG-123`; case does not matter.
- **Team** — its key, such as `ENG`. `SCRIPT teams` lists every team's key and name.
- **Project** — its name, case-insensitive, or its id. `SCRIPT projects --team <key>` lists a team's projects. A name shared by two projects fails; give the id.
- **Document** — its id or its Linear URL.

## Output and errors

- Every command prints JSON to stdout, except `body` and `doc-content`, which print raw markdown.
- On failure a command prints one line to stderr and exits non-zero; Linear's own rejections read `Linear API error: …`. Never work around an error or fall back to reaching Linear another way. Report it and stop.
- A `create` that fails after Linear accepted the issue leaves the issue behind. Check with a listing before creating it again.

## The task model

A task is a Linear issue; a subtask is a sub-issue. Its body is the issue description, in markdown. Every command that returns a task prints:

- `id` — the issue identifier, assigned by Linear on create, so create tasks in dependency order.
- `title` — one line.
- `state` — the name of the issue's workflow state.
- `completed` — `true` when that state is of Linear's completed type (Done, or any other completed state the team has).
- `parent` — the parent task's id, or `null`.
- `blockedBy` — ids of the tasks blocking this one through Linear's "blocks" relation. Related and duplicate relations are not blockers.
- `tags` — the issue's Linear labels, by name.
- `url` — the issue in Linear.

Rules:

- **Pending** — the state is neither completed nor canceled.
- **Startable** — pending, and every task in `blockedBy` is completed. Every blocker counts, inside the scope or not, whatever tags either task carries. A canceled blocker is not completed: it keeps blocking until removed with `unblock`.
- **Tags** never affect blocking; they only narrow which tasks a listing shows. A tag with no matching label is created as a label on the task's team the first time it is used.
- **Subtasks** are independent of their parent for blocking: a parent's blockers do not block its subtasks, and open subtasks do not stop the parent from completing.

## Task operations

- **Create a task** — `SCRIPT create --team <key> [--project <project>] --title "<title>" [--blocked-by <id> …] [--tag <tag> …]`, body on stdin; prints the created task with its new id.
- **Create a subtask** — `SCRIPT create --parent <id> --title "<title>" [--blocked-by <id> …] [--tag <tag> …]`, body on stdin. It takes the parent's team and project. `--team <key>` puts it in another team, without the parent's project; `--project <project>` sets the project.
- **Make an existing task a subtask** — `SCRIPT set-parent --id <id> --parent <id>`.
- **One task in full** — `SCRIPT get --id <id>`; the task plus `team`, `project`, `subtasks`, `blocks` (the ids it blocks), `documents` and `links`.
- **Fetch a task's body** — `SCRIPT body --id <id>`. It prints the body alone — no id, title, blockers or status — so it can be given to someone who must see only the body.
- **Listings** — each takes a scope: `--parent <id>` for that task's direct subtasks, or `--team <key> [--project <project>]`.
  - All tasks — `SCRIPT list <scope>`. An empty scope prints `[]`.
  - Not-completed tasks — `SCRIPT list-pending <scope>`.
  - Startable tasks — `SCRIPT list-unblocked <scope>`.
  - Add `--tag <tag>` to any listing, repeatable, to show only tasks carrying every given tag.
- **Complete a task** — `SCRIPT complete --id <id>`; moves it to the team's first completed state. Fails while any blocker is not completed. A task already completed is printed unchanged.
- **Block a task** — `SCRIPT block --id <id> --blocked-by <id> …`; a blocker already present is skipped. Blocking a task by itself, or in a way that creates a cycle, fails.
- **Unblock a task** — `SCRIPT unblock --id <id> --blocked-by <id> …`; fails when a given task does not block it.
- **Tag a task** — `SCRIPT tag --id <id> --tag <tag> …`; adds, never removes.
- **Replace a task's body** — `SCRIPT update --id <id> [--title "<title>"]`, the full new body on stdin; prints the task. It replaces, never appends: to change part of a body, fetch it with `body`, then write back the whole of it. Blockers, tags and parent are untouched.

## Documents

When the request writes, attaches, links or reads a Linear document on a task or in a project, read `references/documents.md` and follow it.

The skill is finished when the requested Linear operations have run and their results, or the error that stopped them, are reported. Stop there.
