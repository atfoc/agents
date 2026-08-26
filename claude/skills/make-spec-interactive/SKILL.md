---
name: make-spec-interactive
description: Used when user requests interactive session for creating feature definition and then implementation spec
argument-hint: [what is being built]
disable-model-invocation: true
---

# How to create implementation spec
In order to create implementation spec you will follow this process. First we need to define feature. Then we need to convert it to detailed implementation spec.
This skill produces both documents by running two skills in order, in this conversation. I stay in the loop providing feedback and instructions.

## Step 1 — Feature definition
Invoke the `make-feature-definition` skill by name with whatever the user gave. Let it run to its own completion: its own sync check, its own approval, its own file write.

## Step 2 — Implementation spec
When `feature-design.md` has been written, invoke the `make-implementation-spec` skill by name, passing the path that was just written.

## Step 3 — Stop
Stop when `implementation-spec.md` is written, and report both paths.

## Rules
- Add no questions, rules or opinions of your own to either phase.
- Never merge the two loops.
- If the user stops after the feature definition that is a complete outcome, the file is already on disk and `make-implementation-spec` can pick it up in a later session.
- Do not split into tasks and do not implement.
