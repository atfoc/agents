---
name: implement-plan
description: How to implement plan
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
