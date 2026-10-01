# Asana documents

A document is a markdown file attached in Asana: a title and markdown content, stored as the file `<title>.md`. It lives on one task or in one project. `SCRIPT`, referring to tasks, projects and documents, and handling errors are as in [SKILL.md](../SKILL.md). Expand `SCRIPT` to `python3 "<skill-dir>/scripts/asana.py"`, replacing `<skill-dir>` with the absolute directory containing the loaded skill before running commands; quote values containing spaces.

## Attaching a document to a task

A document belongs to a task in one of two ways:

- **On the task** — created with `--task`. Asana shows it among the task's attachments; `get` lists it under `documents`. A document is on at most one task. Use it for a document that belongs to exactly one task: its output, its spec.
- **Linked to the task** — an existing document linked to the task as a link attachment; `get` lists it under `links`. One document can be linked to any number of tasks. Use it for a document several tasks share: create it in the project, then link it to each task.

## Operations

Every command that returns a document prints `id`, `title`, `url`, `task` (the task id it is on, or `null`) and `project` (the project name it is in, or `null`).

- **Create a document on a task** — `SCRIPT doc-create --task <id> --title "<title>"`, content on **stdin**.
- **Create a document in a project** — `SCRIPT doc-create --project <project> [--workspace <workspace>] --title "<title>"`, content on **stdin**.
- **Link a document to a task** — `SCRIPT doc-link --doc <document> --task <id>`; prints the task in full. Linking a document the task already links is skipped.
- **A task's documents** — `SCRIPT get --id <id>`: `documents` for the files on it, `links` for everything linked to it, documents and other links alike.
- **A document's details** — `SCRIPT doc-get --doc <document>`.
- **A document's content** — `SCRIPT doc-content --doc <document>`; prints the markdown alone.
- **Replace a document's content** — `SCRIPT doc-update --doc <document> [--title "<title>"]`, the full new content on **stdin**. It replaces, never appends: to add to a document, fetch its content, then write back the whole of it.

## Replacing gives a new id

Asana cannot change an attached file, so `doc-update` uploads a new file where the old one was and deletes the old one:

- The document gets a new `id` and `url`. The output adds `replaced`, the old `id` and `url`.
- Use the new id from then on.
- Links to the old document stop working. Link the new one, with `doc-link`, to every task that linked the old one.
- A `doc-update` that fails after the upload can leave both files on the task or project. Check with `get` or `doc-get` before running it again.
