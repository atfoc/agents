---
name: implement-plan
description: Implement a plan file that is written as a list of vertical slices — read the slices, ask how the work should be split across sequential subagents, then build them in order. Use whenever the user points at a plan file and asks to implement it, execute it, build it, or work through its slices.
argument-hint: [path-to-plan]
disable-model-invocation: true
---

- We need to implement a plan
- We converted spec of what needs to be built in plan
- Plan is in format of vertical slices
    - vertical slic is set of things to complete in order to build one working and verifiable part of grater plan
    - each vertical slice contains informatino what needs to be done and how should it be verified
- your job is to implement the plan one vertical slice at the time
- each vertical slice should be executed in its own subagent
