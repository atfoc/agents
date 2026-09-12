# Local task format

Tasks live as markdown files inside one root folder. Every read and every write goes through
`scripts/tasks.py` — never through your own file reads or edits.

Whoever uses this format supplies the root folder. If you do not have one, report that the root
folder is missing and stop.

## The format

- One `.md` file per task, flat in the root folder. Nothing else lives in that folder.
- Filename is `<id>-<slug-of-title>.md`. It is never used to find a task — the id in the
  frontmatter is.
- Frontmatter is one `key: <JSON value>` line per field, in the fixed order `id`, `title`,
  `blockedBy`, `tags`, `completed`.
- The body is everything after the closing `---`.

```markdown
---
id: "003"
title: "Add cart handler and its request validation"
blockedBy: ["001", "002"]
tags: ["for-agent"]
completed: false
---

<the full task body>
```

Fields:

- `id` — assigned by the script, zero-padded, never reused.
- `title` — one line.
- `blockedBy` — the ids that must be completed before this task may start.
- `tags` — free-form strings that classify the task: who it is for, what area it touches. Empty
  unless given. A task file written before tags existed has no `tags` line and reads as `[]`.
- `completed` — `false` at creation.

Nothing hand-edits a task file. Nothing else parses the folder. A task is startable when every id
in its `blockedBy` is completed — every id, whatever tags either task carries. Tags never affect
blocking; they only narrow which tasks a listing shows.

## Operations

`SCRIPT` below stands for `python3 <this directory>/scripts/tasks.py`, written out as a literal
absolute path. Resolve it once before running anything; if you do not know this directory:

    find . ~/.claude -path '*local-tasks/scripts/tasks.py' 2>/dev/null | head -1

Every command uses that literal path, and so does any command you hand to someone else. No
variables, nothing relative to a working directory — the shell that runs it is not necessarily
yours.

- **Create a task** — `SCRIPT create --root <root> --title "<title>" [--blocked-by <id> …]
  [--tag <tag> …]`, body on **stdin**; prints the created task including its new id.
- **Where ids come from** — assigned by the script on create, so tasks must be created in
  dependency order.
- **Does the store exist** — `SCRIPT exists --root <root>` → `{"exists": true|false}`, exit 0
  either way.
- **Is the store empty** — `SCRIPT list --root <root>` prints `[]`.
- **All tasks** — `SCRIPT list --root <root>`.
- **Not-completed tasks** — `SCRIPT list-pending --root <root>`.
- **Startable tasks** — `SCRIPT list-unblocked --root <root>`.
- **Only the tasks carrying a tag** — add `--tag <tag>` to any of the three listings, repeatable;
  a task is shown only when it carries every given tag. Blocking is still judged against every
  task in the store: `list-unblocked --tag for-agent` omits a `for-agent` task blocked by an
  untagged task until that untagged task is completed.
- **Complete a task** — `SCRIPT complete --root <root> --id <id>`.
- **Block an existing task** — `SCRIPT block --root <root> --id <id> --blocked-by <id> …`.
- **Tag an existing task** — `SCRIPT tag --root <root> --id <id> --tag <tag> …`; adds, never
  removes.
- **Fetch a task's body** — `SCRIPT body --root <root> --id <id>`.

`body` prints the body alone — no id, title, blockers or status — so it can be given to someone who
must see only the body.

## Errors

Every command prints one line to stderr and exits non-zero on failure. Never work around an error.
Never fall back to reading or writing the folder directly. Report it and stop.
