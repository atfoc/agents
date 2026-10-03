---
name: orchestrate-implementation
description: Orchestrates implementation through subagents, one job at a time in testable vertical slices, reviewing each against its brief and against the rest of the system, with at most two fix rounds per job. Use when you want a goal or specification implemented by subagents.
---

# Orchestrate implementation

Implement the goal or specification in the skill argument, or given earlier in the conversation. If none is given, ask for it and stop.

Only plan, delegate, and track progress. Subagents do all mapping, implementation, verification, reviews, fixes, and integration; never do that work yourself. If delegation is unavailable, report the blocker and stop.

Nesting stops after one level: a subagent cannot start subagents of its own. If this skill is already running inside a subagent, it cannot delegate; report that to whoever started it and stop.

Work on the branch checked out when the run starts. If that checkout has uncommitted changes, report them and stop.

## 1. Plan the jobs

If the goal touches existing code and no map of it came with the goal, delegate one first: the code, data, configuration and tests affected, the callers and consumers of whatever changes, the patterns to reuse, and the existing behaviour that must stay unchanged.

Split the goal into tracer bullets: small end-to-end vertical slices, each delivering observable behavior with acceptance criteria and runnable checks. Order jobs by dependencies and keep their contracts compatible.

Run one implementer or fixer at a time; never two subagents editing at once. Reviewers do not edit and may run in parallel.

## 2. Adapt the plan to what you learn

The split is a working plan, not a commitment. After every report, from an implementer, reviewer or fixer, note what it settled, what it revealed, and whether it invalidates a brief or the split. Then change the remaining work as the findings require: resplit, merge, reorder, add or drop jobs, and rewrite briefs not yet delegated.

- A job that passed review stays complete. If it has to change, that is a new job.
- When a finding shows a job's brief was itself wrong, withdraw the job and replace it. Withdraw only for a wrong brief, never because the implementation fell short of a sound one. Its commits stay on the branch; the replacement's brief says what to keep and what to undo.
- If a change alters what the goal will deliver, tell the user. Otherwise carry on.

Record every change to the plan with the finding that caused it.

## 3. Brief and implement

Write each job's brief just before delegating it, so it reflects what earlier jobs found. The brief states:

- scope, and what is out of scope;
- the context and the completed jobs it depends on;
- acceptance criteria and the checks that prove them;
- existing behaviour that must stay unchanged, and the contracts other jobs rely on.

Send the same brief, verbatim, to the job's implementer, reviewers and fixers.

Require the implementer to commit its work and report: the commit range, each check run with its result, every deviation from the brief and assumption made, unresolved issues, and anything it found that affects other jobs.

## 4. Review the job

After implementation, send fresh review subagents the brief, the implementer's report, the commit range and the map, if there is one. Reviewers read the repository and run checks; they must not edit. They treat the implementer's report as claims to verify, not as evidence. A review answers two questions:

- **Job: did the implementer do what the brief asked?** Before reading the diff, list from the brief what must be true. Then find the evidence for each acceptance criterion, rerun the checks, confirm the tests exercise the behaviour and that none were weakened or skipped, and report any change outside the scope.
- **Fit: is the change sound in the rest of the system?** Explore outward from the diff, independently of the implementer's report. Check the callers and consumers of everything changed, the existing behaviour that had to stay unchanged, the contracts other jobs rely on, and whether the change rebuilds something the app already has or breaks its conventions. Run the full checks, not only the job's.

Use one reviewer per question, in parallel. For a small job one reviewer may answer both.

Require `PASS` or `MUST_FIX` from each reviewer. A finding is must-fix only when an acceptance criterion is unmet, a check fails, existing behaviour is broken, or a contract with another job is broken. Each must-fix finding gives the file and line, what it breaks, and the evidence: a failing command or a path. Everything else is an optional suggestion and never starts a fix round. The review passes only when every reviewer passes.

## 5. Fix, at most twice

On `MUST_FIX`, first decide the cause. If the brief or the split was wrong, replan as section 2 says; that is not a fix round.

Otherwise send a new fix subagent the brief, the findings and the commit range, and require it to commit and report as an implementer does. Then send fresh reviewers the findings and the fixer's report along with everything section 4 lists. They confirm each finding is resolved and answer both questions again over the job's whole range, since a fix can break something new.

Allow at most two fix rounds per job: **implement → review → fix → review → fix → review**. Stop early when no must-fix findings remain. After the third review, report the remaining findings; never reset the limit by renaming, splitting or withdrawing a job whose brief was sound.

## 6. Finish

Mark a job complete only after a passing review. Block dependent jobs on unresolved findings; continue independent jobs.

When no job is left to run, treat integration as one more job under the same limit: delegate a review of the whole change, from the commit the run started on, against the goal, answering both questions and running the full checks, and send must-fix findings to a fix subagent. Do not rework jobs that exhausted their rounds; report them.

Finish when every job is reviewed or blocked and the final review is reported. Summarize completed slices, verification, changes made to the plan and why, remaining must-fix findings, and blockers; stop there.
