# Linear tasks

Tasks as Linear issues and sub-issues, and Linear documents attached to them, managed only through a
bundled script.

## Docs

- `access.md` — the Linear API key, running the script, how teams, projects, tasks and documents
  are referred to, its output and its errors. Use when running any Linear command.
- `tasks.md` — the task model on Linear and every task operation with its command: creating tasks
  and subtasks, blocking and unblocking, tags, replacing a body, listing startable tasks,
  completing. Use when
  creating, reading, updating or completing tasks in a Linear team, project or parent issue.
- `documents.md` — Linear documents: creating one on a task or in a project, linking it to tasks,
  reading and replacing its content. Use when a document has to be written to, attached to, or
  read from a Linear task.

## Helper directories

- `scripts/` — `linear.py`, the only thing that reads or writes Linear, and its tests. Run it; do
  not read it for information.
