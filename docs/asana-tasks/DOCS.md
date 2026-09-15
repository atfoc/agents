# Asana tasks

Tasks as Asana tasks and subtasks, and markdown documents attached to them, managed only through a
bundled script.

## Docs

- `access.md` — the Asana personal access token, running the script, how workspaces, projects,
  tasks and documents are referred to, its output and its errors. Use when running any Asana
  command.
- `tasks.md` — the task model on Asana and every task operation with its command: creating tasks
  and subtasks, blocking and unblocking through dependencies, tags, replacing a body, listing
  startable tasks, completing. Use when creating, reading, updating or completing tasks in an Asana project or
  parent task.
- `documents.md` — documents as markdown files attached in Asana: creating one on a task or in a
  project, linking it to tasks, reading and replacing its content. Use when a document has to be
  written to, attached to, or read from an Asana task.

## Helper directories

- `scripts/` — `asana.py`, the only thing that reads or writes Asana, and its tests. Run it; do
  not read it for information.
