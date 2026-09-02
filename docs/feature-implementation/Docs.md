# Feature implementation

The pipeline from an idea to shipped code: define the feature, specify the implementation, cut it
into tasks, implement them. Each step is its own document and each produces one artifact the next
step consumes.

Order: `defining-feature.md` → `defining-implementation.md` → `split-work-into-vertical-slices.md`
→ `implementing-tasks-workflow.md`.

## Docs

- `defining-feature.md` — producing a feature definition: what is being built, never how. Output is
  `feature-design.md`.
- `defining-implementation.md` — turning a feature definition into a detailed implementation spec.
  Output is `implementation-spec.md`.
- `split-work-into-vertical-slices.md` — cutting a large chunk of work into the slices that become
  tasks: how to cut, how to write each body, how to decide blockers.
- `spec-to-task-dual-agent-workflow.md` — running the split and the task creation as two agents,
  one cutting and one transcribing. The task store is whatever the user requested — it is never
  defaulted or invented; open this when the spec is to become tasks in a store the user named.
- `implementing-tasks-workflow.md` — the loop that takes a store of tasks to done: fetch what is
  startable, spawn one implementer subagent per task, complete, repeat. Also the implementer's
  prompt and how a task body reaches it.
