# Linear documents

A document is a Linear document: a title and markdown content. It lives on one issue or in one project. `SCRIPT`, referring to tasks, projects and documents, and handling errors are defined in `SKILL.md`.

## Attaching a document to a task

A document belongs to a task in one of two ways:

- **On the task** — created with `--issue`. Linear shows it on the issue; `get` lists it under `documents`. A document is on at most one task. Use it for a document that belongs to exactly one task: its output, its spec.
- **Linked to the task** — any existing document linked to the issue as a link attachment; `get` lists it under `links`. One document can be linked to any number of tasks. Use it for a document several tasks share: create it in the project, then link it to each task.

## Operations

Every command that returns a document prints `id`, `title`, `url`, `issue` (the task id it is on, or `null`) and `project` (the project name it is in, or `null`).

- **Create a document on a task** — `SCRIPT doc-create --issue <id> --title "<title>"`, content on stdin.
- **Create a document in a project** — `SCRIPT doc-create --project <project> [--team <key>] --title "<title>"`, content on stdin. `--team` narrows a project name that more than one team can see. Give exactly one of `--issue` or `--project`.
- **Link a document to a task** — `SCRIPT doc-link --doc <document> --issue <id>`; prints the task in full. Linking a document the task already links is skipped.
- **A task's documents** — `SCRIPT get --id <id>`: `documents` for the ones on it, `links` for everything linked to it, documents and other links alike.
- **A document's details** — `SCRIPT doc-get --doc <document>`.
- **Fetch a document's content** — `SCRIPT doc-content --doc <document>`; prints the markdown alone.
- **Replace a document's content** — `SCRIPT doc-update --doc <document> [--title "<title>"]`, the full new content on stdin. It replaces, never appends: to add to a document, fetch its content, then write back the whole of it.

## Markdown that Linear rewrites

Linear renumbers numbered lists when it saves document content, so a list written `1.`, `5.`, `6.`, `8.` is stored as `1.`, `2.`, `3.`, `4.` and references such as "step 5" stop matching. Before sending content with `doc-create` or `doc-update`, rewrite each numbered list item `N. text` as a bullet with the number in bold, `- **N.** text`. Keep continuation lines indented under the item. Leave numbers inside code blocks alone.
