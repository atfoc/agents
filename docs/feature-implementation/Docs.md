# Feature implementation

The pipeline from an idea to shipped code: define the feature, specify the implementation, cut it
into tasks, implement them. Each step is its own document and each produces one artifact the next
step consumes.

Order: `defining-feature.md` → `defining-implementation.md` → `split-spec-to-tasks.md` →
`implementing-tasks.md`.

## Docs

- `defining-feature.md` — producing a feature definition: what is being built, never how. Output is
  `feature-design.md`.
- `defining-implementation.md` — turning a feature definition into a detailed implementation spec.
  Output is `implementation-spec.md`.
- `split-spec-to-tasks.md` — cutting an implementation spec into the slices that become tasks: how
  to cut, how to write each body, how to decide blockers.
- `implementing-tasks.md` — the loop that takes a store of tasks to done: fetch what is startable,
  spawn one implementer subagent per task, complete, repeat. Also the implementer's prompt and how
  a task body reaches it.

## Helper directories

- `prompt-templates/` — prompt templates for running steps of this pipeline. Paste one into a
  prompt and fill its placeholders; do not read them for information.
  - `spec-to-local-tasks.md` — split a spec into vertical slices in one subagent, then create one
    local task per slice in a second subagent.
