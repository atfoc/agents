---
name: manage-asana-tasks
description: Creates, lists, blocks, tags, updates and completes Asana tasks and subtasks, and creates, links, reads and replaces markdown documents attached to them, all through a bundled script. Use when you want to create, read, update or complete tasks in an Asana project or parent task, or attach a document to an Asana task.
argument-hint: [what to do, and the Asana project or parent task]
---

# Manage Asana tasks

Act on the request in `$ARGUMENTS`, or the one made earlier in the conversation: what to do, and the scope the tasks live in — an Asana project, or a parent task whose subtasks are the tasks. If there is no request, ask what to do and stop. Creating and listing tasks need a scope; if the request needs one and names neither a project nor a parent task, report that the scope is missing and stop. Commands on one task or document by its id need no scope.

## Hard rules

- Every read and every write of Asana goes through `${CLAUDE_SKILL_DIR}/scripts/asana.py`. Never reach Asana any other way, and do not read the script for information — run it.
- Never ask for the access token in the conversation, and never write it into a file or a command line.
- Never work around an error, and never fall back to calling Asana another way. Report the error and stop.

## Running the script

`SCRIPT` below stands for `python3 ${CLAUDE_SKILL_DIR}/scripts/asana.py`, written out as that literal absolute path. Use the literal path in every command you run and in any command you hand to someone else — no variables, nothing relative to a working directory. It needs Python 3 and nothing else.

- **Token** — the script reads an Asana personal access token from `ASANA_ACCESS_TOKEN`. When a command fails with `ASANA_ACCESS_TOKEN is not set`, report it and stop: the user creates a personal access token in Asana's developer console and exports it in the environment the script runs in.
- **Output** — every command prints JSON to stdout, except `body` and `doc-content`, which print raw markdown. Every id is an Asana gid, printed as a string.
- **Errors** — a failure prints one line to stderr and exits non-zero; Asana's own rejections read `Asana API error: …`. The script waits out Asana's rate limit and retries by itself, so a command can pause.
- **Failed create** — a `create` that fails after Asana accepted the task leaves the task behind. Check with a listing before creating it again.

## Referring to things

- **Task** — its gid, such as `1204567890123456`, or its Asana URL.
- **Workspace** — its name, case-insensitive, or its gid. `SCRIPT workspaces` lists every workspace.
- **Project** — its name, case-insensitive, its gid, or its Asana URL. `SCRIPT projects [--workspace <workspace>]` lists the projects that are not archived. A name is looked up among those, in every workspace unless `--workspace` narrows it; a name shared by two projects fails, so give the gid or `--workspace`.
- **Document** — its id, or its URL.

## The task

Every command that returns a task prints:

- `id` — the task's gid, assigned by Asana on create. Create tasks in dependency order, so each blocker's id exists before the task it blocks.
- `title` — one line; the task name.
- `completed` — `true` when the task is marked complete.
- `parent` — the parent task's id, or `null`.
- `blockedBy` — the ids of the tasks this one depends on through Asana's dependencies.
- `tags` — the task's Asana tags, by name.
- `url` — the task in Asana.

The body is the task description, in markdown; Asana shows it as plain text.

- **Pending** — not completed. Asana has no canceled state.
- **Startable** — pending, and every task in `blockedBy` is completed. Every blocker counts, inside the scope or not, whatever tags either task carries.
- **Tags** never affect blocking; they only narrow which tasks a listing shows. A tag with no match in the task's workspace is created there the first time it is used.
- **Subtasks** are independent of their parent for blocking: a parent's blockers do not block its subtasks, and open subtasks do not stop the parent from completing.
- **A subtask is in no project** unless created with `--project`. A project listing does not show the subtasks of its tasks; list those with `--parent`.
- A task has at most 30 blockers and blocked tasks combined; past that, blocking fails.

## Task operations

- **Create a task** — `SCRIPT create --project <project> [--workspace <workspace>] --title "<title>" [--blocked-by <id> …] [--tag <tag> …]`, body on **stdin**; prints the created task with its new id.
- **Create a subtask** — `SCRIPT create --parent <id> --title "<title>" [--blocked-by <id> …] [--tag <tag> …]`, body on **stdin**. Adding `--project <project>` also puts it in that project.
- **Make an existing task a subtask** — `SCRIPT set-parent --id <id> --parent <id>`.
- **One task in full** — `SCRIPT get --id <id>`; the task plus `workspace`, `projects` (their names), `subtasks`, `blocks` (the ids it blocks), `documents` and `links`.
- **A task's body** — `SCRIPT body --id <id>`; prints the body alone — no id, title, blockers or status — so it can be given to someone who must see only the body.
- **Listings** — each takes a scope: `--parent <id>` for that task's direct subtasks, or `--project <project> [--workspace <workspace>]` for the project's tasks. Tasks come in the order Asana shows them.
  - All tasks — `SCRIPT list <scope>`. An empty scope prints `[]`.
  - Not-completed tasks — `SCRIPT list-pending <scope>`.
  - Startable tasks — `SCRIPT list-unblocked <scope>`.
  - Only tasks carrying tags — add `--tag <tag>` to any listing, repeatable; a task is shown only when it carries every given tag.
- **Complete a task** — `SCRIPT complete --id <id>`. Fails while any blocker is not completed, although Asana itself would allow it. A task already completed is printed unchanged.
- **Block a task** — `SCRIPT block --id <id> --blocked-by <id> …`; a blocker already present is skipped. Blocking a task by itself, or in a way that creates a cycle, fails.
- **Unblock a task** — `SCRIPT unblock --id <id> --blocked-by <id> …`; fails when a given task does not block it.
- **Tag a task** — `SCRIPT tag --id <id> --tag <tag> …`; adds, never removes.
- **Replace a task's body** — `SCRIPT update --id <id> [--title "<title>"]`, the full new body on **stdin**; prints the task. It replaces, never appends: to change part of a body, fetch it with `body`, then write back the whole of it. Blockers, tags and parent are untouched.

Pass a body or document content on stdin with a quoted heredoc, such as `<<'EOF'`, so the shell does not expand anything in it.

## Documents

When a document has to be created on a task or in a project, linked to tasks, read, or replaced, read `${CLAUDE_SKILL_DIR}/references/documents.md` first and follow it.

The skill is finished when the requested Asana operations have run and their results are reported, or when an error or missing scope has been reported; stop there.
