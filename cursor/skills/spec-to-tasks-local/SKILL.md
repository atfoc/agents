---
name: spec-to-tasks-local
description: Used when we need to split implementation spec into tasks stored as local files
disable-model-invocation: true
---

# Split an implementation spec into local file tasks

1. Your input is a path to an implementation spec. If there is none, ask for it and stop.
2. Invoke the `local-task-format` skill by name. It supplies the task format and its operations.
3. The task store root is `<directory of the spec>/tasks`. Do not create it and do not inspect it —
   `spec-to-tasks` checks it through the format's operations.
4. If a `feature-design.md` sits beside the spec, pass its path as the feature definition.
5. Invoke the `spec-to-tasks` skill by name, giving it the spec path, that optional feature
   definition path, and the local task format as the task format with the root resolved.
6. Add no rules of your own about how to split, what a task contains, or how to report.

`local-task-format` is invoked first so the format is in context before `spec-to-tasks` checks for
it.

You are done when `spec-to-tasks` has finished. Stop there.
