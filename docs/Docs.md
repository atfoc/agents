# Docs

Working knowledge for building software with an agent.

Open only what your task needs. Every directory listed below has its own `Docs.md`. A directory
without a `Docs.md` holds no docs — it is a helper directory (scripts, templates, assets) for its
parent, used only when a doc points at it.

## Docs

- `asking-questions.md` — how to run an interactive question loop with the user, when a document is
  built with them rather than written at them.
- `vertical-slices.md` — what a vertical slice is: its verification, its body, its blockers, and
  what holds across the whole set.
- `researching.md` — answering an open question from evidence, in rounds.
- `explaining-code.md` — explaining code at the abstraction level the user asked for.

## Directories

- `feature-implementation/` — the pipeline from an idea to shipped code: defining a feature,
  specifying the implementation, creating tasks, implementing them.
- `local-tasks/` — the local folder task format and the commands that read and write it.
- `docs/` — this collection's own format and how an agent searches it.
