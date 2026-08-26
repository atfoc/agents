---
name: local-task-format
description: Documents the local folder task format and the operations for managing tasks in it
disable-model-invocation: false
---

# Local task format

Tasks live as markdown files inside one root folder. Every read and every write of a task goes
through `scripts/tasks.py` — never through your own file reads or edits. The skill that invokes
this one must supply the root folder; if it did not, report that the root folder is missing and
stop.

## The format

- One `.md` file per task, flat in the root folder. Nothing else lives in that folder.
- The filename is `<id>-<slug-of-title>.md`. It is never used to find a task — the id in the
  frontmatter is.
- The frontmatter is one `key: <JSON value>` line per field, in the fixed order `id`, `title`,
  `blockedBy`, `completed`.
- The body is everything after the closing `---`.

```markdown
---
id: "003"
title: "Add cart handler and its request validation"
blockedBy: ["001", "002"]
completed: false
---

<the full task body>
```

Field meanings:

- `id` — assigned by the script, zero-padded, never reused.
- `title` — one line.
- `blockedBy` — the ids that must be completed before this task may start.
- `completed` — `false` at creation.

Nothing hand-edits a task file. Nothing else parses the folder. A task is startable when every id
in its `blockedBy` is completed.

## Operations

`SCRIPT` below stands for `python3 ${CLAUDE_SKILL_DIR}/scripts/tasks.py`.

| Capability | Command |
| :-- | :-- |
| Create a task | `SCRIPT create --root <root> --title "<title>" [--blocked-by <id> …]`, body on **stdin**; prints the created task including its new id |
| Where ids come from | Assigned by the script on create, so tasks must be created in dependency order |
| Does the store exist | `SCRIPT exists --root <root>` → `{"exists": true|false}`, exit 0 either way |
| Is the store empty | `SCRIPT list --root <root>` → the store is empty when it prints `[]` |
| All tasks | `SCRIPT list --root <root>` |
| Not-completed tasks | `SCRIPT list-pending --root <root>` |
| Startable tasks | `SCRIPT list-unblocked --root <root>` |
| Complete a task | `SCRIPT complete --root <root> --id <id>` |
| Block an existing task | `SCRIPT block --root <root> --id <id> --blocked-by <id> …` |
| Fetch a task's body | the literal block below |

Copy the block below verbatim into an implementer's prompt, substituting `<root>` and `<id>` and
changing nothing else:

```
Read the task body by running: python3 ${CLAUDE_SKILL_DIR}/scripts/tasks.py body --root <root> --id <id>
```

## Errors

Every command prints one line to stderr and exits non-zero on failure. Never work around an error.
Never fall back to reading or writing the folder directly. Report it and stop.
