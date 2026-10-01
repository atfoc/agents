---
name: orchestrate-implementation-plan
description: Delegates research, planning and review to subagents to produce a high-level implementation plan grounded in facts. Use when you want a reviewed plan for how to build a feature without code or low-level implementation details.
---

# Orchestrate an implementation plan

Use the skill argument or the goal and feature definition provided in the conversation. If the goal is missing, ask for it and stop.

Coordinate only: delegate research, experiments, drafting, fixes and assembly. Scope jobs, pass context, track attempts and report results; do not do the substantive work yourself or implement the feature. If delegation is unavailable, report the blocker and stop.

Use one delegation level. If already running inside a subagent, perform only the assigned job and return to the parent; do not delegate further.

## 1. Delegate scoped jobs

Choose jobs appropriate to the goal: exploration, feasibility checks, isolated experiments, or drafting part or all of the plan. Give each subagent the goal, relevant requirements and artifacts, its scope, expected output and acceptance criteria. Run independent jobs in parallel; pass accepted findings to dependent jobs. Keep shared decisions consistent across jobs.

## 2. Require a high-level plan

Tell drafting subagents to cover:

- Scope, requirements, constraints and acceptance criteria.
- Modules and components: responsibilities, relationships, ownership and integration points.
- Operations and flows: triggers, outcomes, lifecycle, relevant failure handling and access requirements.
- How to build it: phases, dependencies, prerequisites, integration and verification approach.
- References supporting factual claims: repository paths/lines, documentation URLs or recorded experiment results. Distinguish verified facts from assumptions and unresolved questions.

Exclude implementation code, pseudocode and low-level definitions such as exact function signatures, field schemas or internal algorithms. Explain enough to guide implementation while leaving those details for later.

## 3. Review each job; cap fixes

After each job, delegate review to a subagent other than its author. Supply the original scope, requirements, output and supporting sources. Require `PASS` or `MUST_FIX`, with actionable must-fix findings separated from optional suggestions. Review coverage, factual support, feasibility, consistency and the required level of detail.

Track attempts per job: **one initial production attempt plus at most two fixes**, each followed by review. On `MUST_FIX`, send a new fixing subagent the current artifact, original context and review findings, then review its revised output independently. On `PASS`, accept the job.

If the third review still reports `MUST_FIX`, stop and report the remaining findings, affected job and partial artifacts to the user. Do not reset the limit by renaming or splitting the failed job, or present it as complete.

## 4. Assemble and deliver

Delegate assembly of accepted outputs into one coherent plan, resolving gaps and conflicting decisions. Review the complete plan for cross-section consistency and goal coverage as an assembly job under the same attempt limit.

Deliver the reviewed plan at the requested output path, or in the conversation if none was given, and identify any unresolved questions. Finish after the complete plan passes review or the attempt limit requires escalation. Stop there; do not implement it.
