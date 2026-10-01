---
name: orchestrate-implementation
description: Orchestrates implementation through subagents in testable vertical slices with at most two fix rounds per job. Use when you want a goal or specification implemented by subagents.
---

# Orchestrate implementation

Invoke with `$orchestrate-implementation <goal or specification>`. Implement the goal or specification in the request, or given earlier in the conversation. If none is given, ask for it and stop.

Only plan, delegate, and track progress. Subagents do all implementation, verification, reviews, fixes, and integration; never do that work yourself. If delegation is unavailable, report the blocker and stop.

Use the available Codex subagent tools for each implement, review, fix, and integration assignment. Create each role with fresh context and give it the relevant assignment and artifacts explicitly. With `collaboration.spawn_agent`, set `fork_turns: "none"`, supply that assignment in `message`, and omit model and effort overrides. Receive and inspect completion reports before advancing the job. Do not create separate user-owned Codex chats as workers. If the runtime cannot provide fresh subagents and their reports, report the delegation blocker and stop.

1. Split the goal into tracer bullets: small end-to-end vertical slices, each delivering observable behavior with acceptance criteria and runnable checks. Order jobs by dependencies and keep their contracts compatible. Parallelize only independent jobs whose changes cannot collide.
2. Delegate each job with its scope, relevant context, dependencies, acceptance criteria, and checks. Require the implementer to report changes, verification results, and unresolved issues.
3. After implementation, send a fresh review subagent the requirements and resulting changes. Have it verify behavior, checks, and compatibility with completed slices, distinguishing must-fix defects from optional suggestions. Reviews must not edit the implementation.
4. If there are must-fix findings, send them to a new fix subagent, then request a fresh review. Allow at most two fix rounds per job: **implement → review → fix → review → fix → review**. Stop early when no must-fix findings remain. After the third review, report remaining findings; never reset the limit by renaming or splitting the failed job.
5. Mark a job complete only after a passing review. Block dependent jobs on unresolved findings; continue independent jobs. Delegate integration and final checks of the combined behavior, reporting failures without restarting exhausted cycles.

Finish when every job is reviewed or blocked and final checks are reported. Summarize completed slices, verification, remaining must-fix findings, and blockers; stop there.
