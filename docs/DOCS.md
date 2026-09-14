# Docs

Working knowledge for building software with an agent.

Open only what your task needs. Every directory listed below has its own `DOCS.md`. A directory
without a `DOCS.md` holds no docs — it is a helper directory (scripts, templates, assets) for its
parent, used only when a doc points at it.

## Docs

- `asking-questions.md` — running an interactive question loop with the user, one question at a
  time, when a document is built with them rather than written at them. Use when a document needs
  the user's input to be written.
- `vertical-slices.md` — what a vertical slice is: its verification, its body, its blockers, and
  what holds across the whole set of slices.
- `explaining-code.md` — explaining code at the abstraction level the user asked for. Use when the
  user asks a question about the codebase.

## Directories

- `feature-implementation/` — the pipeline from an idea to shipped code: defining a feature,
  specifying the implementation, cutting it into tasks, implementing them. Use when building a
  feature end to end, or when doing any single step of that pipeline.
- `dev-process/` — taking a vague, large chunk of work, understanding it, then breaking it into
  efforts that fit feature definition, implementation definition, split, and implement: an
  orchestrator that plans that road as tasks, and the runners for the agent tasks and the human
  tasks it creates. Use when the user has a goal to drive to done, when finished tasks need the
  next step planned, or when running the tasks a planning iteration created.
- `local-tasks/` — the local folder task format and the commands that read and write it. Use when
  reading, creating or updating tasks in a local task folder.
- `docs/` — this collection's own format and how an agent searches it. Use when writing a new doc,
  adding an index entry, or setting up a docs collection.
