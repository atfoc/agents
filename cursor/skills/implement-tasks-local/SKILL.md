---
name: implement-tasks-local
description: Used when we need to implement tasks stored as local files
disable-model-invocation: true
---

# Implement tasks stored as local files

1. The input is a path — the skill argument, or a path named in the conversation. If there is
   none, ask for it and stop.
2. Invoke the `local-task-format` skill by name. It supplies the task format and the operations
   for it.
3. Resolve the root: if `<path>/tasks` exists that is the root, otherwise `<path>` is the root. Do
   nothing further to check it — a wrong path surfaces as a hard error from the first operation.
4. Invoke the `implement-tasks` skill by name, giving it the local format's operations with the
   root filled in.
5. Add no rules of your own about execution, subagents or failure handling.

`local-task-format` is invoked before `implement-tasks`, so the operations are in context before
`implement-tasks` checks for them.

This skill is finished when `implement-tasks` has run to its own completion.
