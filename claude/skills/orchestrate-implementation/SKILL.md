---
name: orchestrate-implementation
description: Orchestrates implementation through subagents, one job at a time in testable vertical slices, reviewing each against its brief and against the rest of the system, with at most two fix rounds per job. Use when you want a goal or specification implemented by subagents.
argument-hint: "[goal or specification]"
---

# Orchestrate implementation

Implement the goal or specification in `$ARGUMENTS`, or given earlier in the conversation. If none is given, ask for it and stop.

Only plan, delegate, and track progress. Subagents do all mapping, implementation, verification, reviews, fixes, and integration; never do that work yourself. If delegation is unavailable, report the blocker and stop.

Work on the branch checked out when the run starts. If that checkout has uncommitted changes, report them and stop. Untracked files that hold the goal or its supporting material do not count; tell every subagent to read them and never commit them.

Keep the run's files in one run folder outside the checkout, so nothing in it can be committed, and tell the user where it is. It holds:

- the ledger: the jobs in order, each with its status and commit range; every change to the plan with the finding that caused it; and the follow-ups, each with what was decided about it;
- each job's brief;
- every subagent's report. The subagent writes it there itself and replies with the path and a short summary.

Update the ledger after every report. Pass briefs and reports on as files; never retype one. When a run is resumed, continue from the ledger.

## 1. Plan the jobs

If the goal touches existing code and no map of it came with the goal, delegate one first: the code, data, configuration and tests affected, the callers and consumers of whatever changes, the patterns to reuse, and the existing behaviour that must stay unchanged.

Split the goal into tracer bullets: small end-to-end vertical slices, each delivering observable behavior with acceptance criteria and runnable checks. Order jobs by dependencies and keep their contracts compatible.

Run one implementer or fixer at a time; never two subagents editing at once. Reviewers do not edit and may run in parallel.

## 2. Adapt the plan to what you learn

The split is a working plan, not a commitment. After every report, from an implementer, reviewer or fixer, note what it settled, what it revealed, and whether it invalidates a brief or the split. Then change the remaining work as the findings require: resplit, merge, reorder, add or drop jobs, and rewrite briefs not yet delegated.

- A job that passed review stays complete. If it has to change, that is a new job.
- When a finding shows a job's brief was itself wrong, withdraw the job and replace it. Withdraw only for a wrong brief, never because the implementation fell short of a sound one. Its commits stay on the branch; the replacement's brief says what to keep and what to undo.
- If a change alters what the goal will deliver, tell the user. Otherwise carry on.

Record every change to the plan in the ledger, with the finding that caused it.

## 3. Brief and implement

Write each job's brief to the run folder just before delegating it, so it reflects what earlier jobs found. The brief states:

- scope, and what is out of scope;
- the context and the completed jobs it depends on;
- acceptance criteria and the checks that prove them. When the job changes behaviour a user can see, one check exercises that behaviour end to end in the running app, written so a reviewer can rerun it without touching the user's own running instance or data;
- existing behaviour that must stay unchanged, and the contracts other jobs rely on.

Give the same brief file to the job's implementer, reviewers and fixers.

Require the implementer to commit its work and report: the commit range, each check run with its result, every deviation from the brief and assumption made, unresolved issues, and anything it found that affects other jobs.

Two rules hold for every implementer and fixer:

- It never breaks code in the checkout to see whether a test notices. It does that in a copy outside the checkout, and saves the script it used in the run folder.
- It finishes or stops everything it started before it replies; its reply ends its work.

## 4. Review the job

After implementation, send fresh review subagents the brief, the commit range and the map, if there is one. Reviewers read the repository and run checks; they must not edit. A review answers two questions, each with its own inputs and its own ground:

- **Job: did the implementer do what the brief asked?** Give this reviewer the implementer's report as well, as claims to verify, not as evidence. Stay inside the brief; do not trace callers or consumers. Open the report with the list, made from the brief before reading the diff, of what must be true. Then find the evidence for each acceptance criterion, rerun the checks, including the end-to-end one, confirm the tests exercise the behaviour and that none were weakened or skipped, judge each deviation the implementer disclosed, and report any change outside the scope. Where the implementer saved a script that proves its tests, rerun that script rather than writing another, and add cases only for the acceptance criteria it leaves out.
- **Fit: is the change sound in the rest of the system?** Do not give this reviewer the implementer's report. Give it instead the risks you see in this job: the interleavings, consumers and failure paths most likely to break. Explore outward from the diff. Check the callers and consumers of everything changed, the existing behaviour that had to stay unchanged, the contracts other jobs rely on, and whether the change rebuilds something the app already has or breaks its conventions. Run the full checks, not only the job's. Unchanged behaviour and contracts are this reviewer's ground alone.

Use one reviewer per question, in parallel. For a small job one reviewer may answer both; it answers the fit question first, before reading the implementer's report.

Require `PASS` or `MUST_FIX` from each reviewer, and its findings in three separate lists:

- **Must-fix:** only when an acceptance criterion is unmet, a check fails, existing behaviour is broken, or a contract with another job is broken. Each gives the file and line, what it breaks, and the evidence: a failing command or a path.
- **Test gap:** a behaviour the brief requires is correct, but no test would fail without it.
- **Optional suggestion:** everything else.

Only must-fix findings start a fix round. A check that fails for a reason the job's commits cannot have caused, because it fails the same way on the commit the job started from or lies in code those commits do not reach, is reported, not investigated, and is not must-fix. The review passes only when every reviewer passes.

Enter every test gap and optional suggestion in the ledger as a follow-up. Schedule each test gap: into the brief of a later job that touches the same code, or into the follow-up job of section 6. For each optional suggestion, schedule it the same way or record why it is dropped.

## 5. Fix, at most twice

On `MUST_FIX`, first decide the cause. If the brief or the split was wrong, replan as section 2 says; that is not a fix round.

Otherwise send a new fix subagent the brief, the findings and the commit range, and require it to commit and report as an implementer does. Then review again as section 4 says, with fresh reviewers. Both also get the findings; the job reviewer also gets the fixer's report. They confirm each finding is resolved and answer both questions again over the job's whole range, since a fix can break something new.

Allow at most two fix rounds per job: **implement → review → fix → review → fix → review**. Stop early when no must-fix findings remain. After the third review, report the remaining findings; never reset the limit by renaming, splitting or withdrawing a job whose brief was sound.

## 6. Finish

Mark a job complete only after a passing review. Block dependent jobs on unresolved findings; continue independent jobs.

When no planned job is left and the ledger still holds scheduled follow-ups, run them as one follow-up job, briefed, implemented and reviewed like any other.

Then treat integration as one more job under the same limit: delegate a review of the whole change, from the commit the run started on, against the goal, answering both questions and running the full checks, and send must-fix findings to a fix subagent. Do not rework jobs that exhausted their rounds; report them.

Finish when every job is reviewed or blocked and the final review is reported. Summarize completed slices, verification, changes made to the plan and why, remaining must-fix findings, follow-ups dropped and why, and blockers; stop there.
