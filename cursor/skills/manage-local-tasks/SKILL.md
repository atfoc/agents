---
name: manage-local-tasks
description: Creates, lists, blocks, tags, completes and reads tasks stored as markdown files in a local task folder, doing every read and write through the bundled tasks.py script. Use when you want to add a task, see pending or startable tasks, mark a task done, or fetch a task's body in a local task folder.
---

# Manage tasks in a local task folder

Act on the task root folder and the operation given in the skill argument, or named earlier in the conversation. If no root folder was given, report that the root folder is missing and stop. If no operation was given, ask what to do with the tasks and stop.

## Hard rules

- Every read and every write of the task folder goes through `scripts/tasks.py`. Never read, edit, create, rename or delete files in the root folder yourself, and never parse the folder any other way.
- Run the script as `python3 scripts/tasks.py`, with the skill directory written out as a literal absolute path. Pass `--root` as an absolute path as well. Use no shell variables and nothing relative to a working directory, including in any command you hand to someone else.
- Run the script. Do not read it for information.
- If a command fails, it prints one line to stderr and exits non-zero. Do not work around it or fall back to touching the folder directly. Report the error and stop.

## The format

Background only. The script enforces all of it.

- One `.md` file per task, flat in the root folder, named `<id>-<slug-of-title>.md`. Nothing else lives in that folder.
- Frontmatter is one `key: <JSON value>` line per field, in this order:
  - `id`: assigned by the script, zero-padded, never reused. The id in the frontmatter identifies a task, never its filename.
  - `title`: one line.
  - `blockedBy`: the ids that must be completed before this task may start.
  - `tags`: free-form strings saying who the task is for or what area it touches. Empty unless given. A file with no `tags` line reads as `[]`.
  - `completed`: `false` at creation.
- The body is everything after the closing `---`.
- A task is startable when every id in its `blockedBy` is completed, whatever tags either task carries. Tags never affect blocking. They only narrow which tasks a listing shows.

## Operations

`SCRIPT` stands for `python3 <absolute skill dir>/scripts/tasks.py`. Every command prints JSON unless noted.

- **Does the store exist**: `SCRIPT exists --root <root>` prints `{"exists": true|false}` and exits 0 either way.
- **All tasks**: `SCRIPT list --root <root>`. The store is empty when this prints `[]`.
- **Not-completed tasks**: `SCRIPT list-pending --root <root>`.
- **Startable tasks**: `SCRIPT list-unblocked --root <root>`.
- **Only tasks carrying a tag**: add `--tag <tag>` to any of the three listings. It is repeatable, and a task is shown only when it carries every given tag. Blocking is still judged against every task in the store, so `list-unblocked --tag for-agent` leaves out a `for-agent` task blocked by an untagged task until that task is completed.
- **Create a task**: `SCRIPT create --root <root> --title "<title>" [--blocked-by <id> …] [--tag <tag> …]`, with the body on stdin, for example through a quoted heredoc. It prints the created task with its new id and creates the root folder if it does not exist. The body must not be empty.
- **Create several dependent tasks**: the script assigns ids, so create them in dependency order and pass each new task the ids printed for its blockers.
- **Complete a task**: `SCRIPT complete --root <root> --id <id>`. This fails while any blocker is not completed. Completing an already completed task is a no-op.
- **Block an existing task**: `SCRIPT block --root <root> --id <id> --blocked-by <id> …`. It adds blockers and refuses self-blocks, unknown ids and cycles.
- **Tag an existing task**: `SCRIPT tag --root <root> --id <id> --tag <tag> …`. It only adds tags, never removes them.
- **Fetch a task's body**: `SCRIPT body --root <root> --id <id>` prints the body alone as plain text, with no id, title, blockers or status. Use it to hand a task to someone who must see only the body.

## Steps

1. Resolve the absolute path of `scripts/tasks.py` and the absolute root folder.
2. Pick the operation or operations above that match the request. For a read-only question, run the matching listing or `body` command.
3. Run the commands. After one fails, run nothing further.
4. Report what the script printed: the tasks listed, created or changed with their ids, or the exact error line.

The skill is finished when the requested operations have run and their results or error have been reported. Stop there.
