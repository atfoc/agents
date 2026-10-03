---
name: orchestrate-implementation-plan
description: Delegates research, prototypes, planning and review to subagents, choosing each next move from what the last one found, to produce a high-level implementation plan grounded in facts and fitted to the existing app. Use when you want a reviewed plan for how to build a feature without code or low-level implementation details.
argument-hint: "[goal or definition] [optional output path]"
---

# Orchestrate an implementation plan

Use `$ARGUMENTS` or the goal and feature definition provided in the conversation. If the goal is missing, ask for it and stop.

Coordinate only: delegate research, experiments, drafting, fixes and assembly. Scope jobs, pass context, track attempts and report results; do not do the substantive work yourself or implement the feature. If delegation is unavailable, report the blocker and stop.

Use one delegation level. If already running inside a subagent, perform only the assigned job and return to the parent; do not delegate further.

There is no fixed sequence. Section 1 says what must hold before delivery, section 2 how to choose the next move, section 3 the moves available. Scale the effort to the risk: a small, well-understood goal needs few moves, a hard one many rounds.

## 1. What must hold before delivery

The path is yours to choose; these conditions are not:

- No plan-critical question is open. The only exceptions are questions only the user can settle, which the plan lists together with what depends on them.
- Every claim the plan rests on is verified by a repository path and line, a documentation URL or a recorded run result. Nothing rests on an inference.
- The plan says how it fits the existing app: what it touches, what it reuses, which existing behaviour changes and which must stay unchanged.
- Every element of the plan traces to a requirement or a verified constraint. Anything that does not is removed.
- The plan passed review by subagents other than its author, within the attempt limit in section 5.

## 2. Choose the next move from the open questions

Keep one list of the open questions and assumptions the plan depends on. Start it from the goal. After every job returns, update it:

- what the job settled;
- what new unknowns it raised;
- whether it invalidated an earlier decision.

Then choose the next moves from the list. Stop groundwork when nothing plan-critical remains open; do not research what would not change the plan.

## 3. Moves

Give each subagent the brief, the relevant requirements and artifacts, its scope, expected output and acceptance criteria. Run independent jobs in parallel; pass their outputs to dependent jobs. Keep shared decisions consistent across jobs.

Mapping, research and prototypes are groundwork. Require every groundwork finding to be marked verified, with its evidence, inferred or unknown. Put inferred and unknown findings the plan depends on back on the list.

**Brief.** Reach for it when the goal is loose enough that subagents would read it differently. It yields one statement of what is being built and why, with requirements, non-goals, constraints and acceptance criteria. Give it to every later subagent. A goal that is already this precise is the brief.

**Map the existing app.** Reach for it whenever the feature touches existing code. It yields the code, data, configuration and tests affected; the callers and consumers of whatever changes; the patterns and components to reuse; and the existing behaviour that must stay unchanged.

**Research.** Reach for it when a question can be settled by reading code or documentation. It yields an answer with its evidence.

**Prototype.** Reach for it when only running code can settle a question, when findings contradict each other, or when the plan leans on an assumption that would be expensive to get wrong. Tell the subagent to invoke the `prototype` skill and follow it. The subagent cannot reach the user, so supply every input that skill would otherwise ask for or infer:

- The question, including what settled looks like.
- The mode: without a human in the loop.
- The name: short kebab-case, unique among this run's prototypes.
- The output location: an absolute file path, beside the plan's output path, or in a temporary directory when the plan is delivered in the conversation.

Tell it not to ask questions: if the question cannot be settled from these inputs, it writes what is missing to the output location and returns. Pass the output file to dependent jobs as an artifact. Questions that need a person's judgment, such as how something looks or feels, are not prototype jobs; record them as unresolved questions for the user.

**Ask the user.** Reach for it when an open question is a decision that is the user's to make, not a fact, and the shape of the plan depends on it. Ask such questions together, and keep running the jobs that do not depend on the answers. Record decisions the plan does not hinge on as unresolved questions instead.

**Draft.** Reach for it when the questions a section depends on are closed. It yields part or all of the plan, following section 4.

**Assemble.** Reach for it when the drafts cover the goal. It yields one coherent plan, with gaps and conflicting decisions resolved.

**Review and fix.** Reach for them once the plan is assembled, following section 5.

**Backtrack.** Reach for it when a finding invalidates an earlier decision: a prototype disproves the approach, or the map shows the design conflicts with the app. Return to the move that produced the decision, redo what depends on it and have the affected sections redrafted; do not patch around it. If it changes what the goal can deliver, tell the user.

## 4. Require a high-level plan

Tell drafting subagents to cover:

- Scope, requirements, non-goals, constraints and acceptance criteria.
- Fit with the existing app: what is touched, what is reused, changes to existing behaviour and what must stay unchanged.
- Modules and components: responsibilities, relationships, ownership and integration points.
- Operations and flows: triggers, outcomes, lifecycle, relevant failure handling and access requirements.
- How to build it: phases as testable vertical slices, dependencies, prerequisites, integration, and the verification for each acceptance criterion. Migration, rollout and rollback where relevant.
- Assumptions and risks, and unresolved questions with what depends on them.
- References supporting factual claims: repository paths/lines, documentation URLs or recorded experiment results. Distinguish verified facts from assumptions and unresolved questions.

Exclude implementation code, pseudocode and low-level definitions such as exact function signatures, field schemas or internal algorithms. Explain enough to guide implementation while leaving those details for later.

## 5. Review the built plan; cap fixes

Do not review jobs that produce artifacts: mapping, research, prototypes or draft sections. Their output is data for the plan, not a deliverable. Take it as returned and pass it on; if an output is unusable or misses its scope, re-delegate the job rather than review it.

Review only the plan, once it is assembled. A review round is one or more reviewers run in parallel, none of them the plan's author. Give each the brief, the plan and the artifacts it draws on, let it read the repository, and assign it one lens:

- **Gaps.** Before reading the plan, list from the brief and the code what a complete plan must cover. Then report what the plan misses.
- **Fit.** Explore the affected areas independently of the map. Report callers, side effects, conflicting patterns and behaviour changes the plan does not account for.
- **Failure.** Report error paths, concurrency, data states, permissions and partial rollout the plan leaves unhandled, and anything else that would surprise once built.
- **Scope.** Trace every element to a requirement or a verified constraint. Report what does not trace, what rebuilds something the app already has, and detail below the required level.
- **Grounding.** Report where the plan does not follow from the artifacts, contradicts them or rests on unsupported assumptions, and where sections disagree. Check the facts in the artifacts where needed.

Every round covers gaps and fit; add the other lenses where the risk is. For a small goal one reviewer may carry several lenses. Require `PASS` or `MUST_FIX` from each reviewer, with actionable must-fix findings separated from optional suggestions. The round passes only when every reviewer passes.

Track attempts on the plan: **one initial assembly plus at most two fixes**, each followed by a review round that reruns every lens used so far, since a fix can open new gaps. On `MUST_FIX`, decide what the fix needs before delegating it; a fixing subagent does not have to come next. Where findings call for more groundwork or for backtracking, first delegate the jobs that supply it, in whatever order and number you judge right. These are artifact jobs: they are not reviewed and do not count as attempts. Then send a new fixing subagent the current plan, the brief, all artifacts and the review findings, and review its revised plan independently. On `PASS`, accept the plan.

If the third review round still reports `MUST_FIX`, stop and report the remaining findings and the partial plan to the user. Do not reset the limit by renaming or splitting the plan, or present it as complete.

## 6. Deliver

Deliver the reviewed plan at the requested output path, or in the conversation if none was given, and identify any unresolved questions. List the branches and worktrees left by prototype jobs; they are the user's to remove. Finish after the plan passes review or the attempt limit requires escalation. Stop there; do not implement it.
