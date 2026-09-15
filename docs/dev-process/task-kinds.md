# Task kinds

The six kinds of task a dev process plans. The kind fixes the doc the task runs; the task body
carries everything else, so whoever runs a task reads nothing but its body. Every kind except
prototype also fixes the tag: prototype is `for-agent` or human depending on the question.

Tasks tagged `for-agent` need no person and run in the background. Untagged tasks are the user's
and run in the foreground.

- **research** — `for-agent`. One focused question whose answer a later decision needs. Output:
  the question, the answer, and the evidence it rests on, short enough to be read whole. A
  question that cannot be answered still completes, with the output saying what was missing.
- **feature definition** — human. Runs `../feature-implementation/defining-feature.md` on what the
  body asks for. Output: the feature definition.
- **implementation definition** — human. Runs
  `../feature-implementation/defining-implementation.md` on the feature definition the body points
  at. Output: the implementation spec.
- **prototype** — `for-agent` or human. Runs `../prototyping.md` on the question in the body.
  `for-agent`, without a human in the loop, when the question is whether something works, can
  work, or how it behaves in real code. Human, with a human in the loop, when it needs a person's
  judgment — how it looks, how it feels. The body names the mode and gives the task key as the
  prototype's name. Output: the question, what was tried, what it settled, and the branch as a
  pointer.
- **split tasks** — `for-agent`. Runs
  `../feature-implementation/spec-to-task-dual-agent-workflow.md` on the spec and feature
  definition the body points at, into the slice store the body describes. Output: the slice
  store, filled.
- **implement tasks** — `for-agent`. Runs
  `../feature-implementation/implementing-tasks-workflow.md` on the slice store the body
  describes. Output: the run's report — what was completed and what was reported and not
  resolved.

## Output location

A body names the task's output location. That location overrides any default output path the doc
behind the kind names — a fact stated in the prompt outranks the doc.
