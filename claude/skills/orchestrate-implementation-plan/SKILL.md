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

- No plan-critical question is open. The only exceptions are questions only the user can settle and questions a prototype reported it could not settle; the plan lists both together with what depends on them.
- Every claim the plan rests on is verified by a repository path and line, a documentation URL or a recorded run result. Nothing rests on an inference. What an external process does at runtime is verified only by a recorded run.
- The plan says how it fits the existing app: what it touches, what it reuses, which existing behaviour changes and which must stay unchanged.
- Every element of the plan traces to a requirement or a verified constraint. Anything that does not is removed.
- The plan passed review by subagents other than its author, within the attempt limit in section 5.

## 2. Choose the next move from the open questions

Keep one list of the open questions and assumptions the plan depends on, in a file beside the plan's output path, or in a temporary directory when the plan is delivered in the conversation. Start it from the goal. Mark each entry as one of:

- settled, with its evidence;
- a user decision, with the answer once it is given;
- unknown.

An entry is plan-critical when a different answer would change a component, a flow or a phase. An entry stays unknown until evidence or the user's answer settles it; a default you picked or an assumption you recorded does not.

After every job returns, update the list:

- what the job settled;
- what new unknowns it raised;
- whether it invalidated an earlier decision.

Then choose the next moves from the list. Start the job for a question as soon as the question is on the list; do not wait for the jobs already running to return.

Do not start a draft while a plan-critical entry it depends on is unknown, unless a prototype reported that it cannot be settled. Stop groundwork when nothing plan-critical remains open; do not research what would not change the plan.

## 3. Moves

Give each subagent the brief, the relevant requirements and artifacts, its scope, expected output and acceptance criteria. Run independent jobs in parallel; pass their outputs to dependent jobs. Give a groundwork job one question, or questions settled by the same reading or the same run; split independent questions into separate jobs. Keep shared decisions consistent across jobs.

Mapping, research and prototypes are groundwork. Require every groundwork finding to be marked verified, with its evidence, inferred or unknown. Put inferred and unknown findings the plan depends on back on the list.

**Brief.** Reach for it when the goal is loose enough that subagents would read it differently. It yields one statement of what is being built and why, with requirements, non-goals, constraints and acceptance criteria. Give it to every later subagent. A goal that is already this precise is the brief.

**Map the existing app.** Reach for it whenever the feature touches existing code. It yields the code, data, configuration and tests affected; the callers and consumers of whatever changes; the patterns and components to reuse; and the existing behaviour that must stay unchanged.

**Research.** Reach for it when a question can be settled by reading code or documentation. It yields an answer with its evidence. What an external process does at runtime, such as a CLI, an API or a model, is not a research question: reading its source or binary yields an inference. Prototype it.

**Prototype.** Reach for it when only running code can settle a question, when findings contradict each other, or when the plan leans on an assumption that would be expensive to get wrong. When the feature rests on a premise about runtime behaviour, prototype that premise before the draft. Tell the subagent to invoke the `prototype` skill and follow it. The subagent cannot reach the user, so supply every input that skill would otherwise ask for or infer:

- The question, including what settled looks like.
- The mode: without a human in the loop.
- The name: short kebab-case, unique among this run's prototypes.
- The output location: an absolute file path, beside the plan's output path, or in a temporary directory when the plan is delivered in the conversation.

Tell it not to ask questions: if the question cannot be settled from these inputs, it writes what is missing to the output location and returns. Pass the output file to dependent jobs as an artifact. Questions that need a person's judgment, such as how something looks or feels, are not prototype jobs; record them as unresolved questions for the user.

**Ask the user.** Reach for it when an open question is a decision that is the user's to make, not a fact, and the shape of the plan depends on it. Ask such questions together, before the draft that depends on them, and keep running the jobs that do not depend on the answers. Do not answer them with a default and have the plan drafted to stay open to either answer. Record decisions the plan does not hinge on as unresolved questions instead.

**Draft.** Reach for it when the questions a section depends on are closed, as section 2 requires. It yields part or all of the plan, following section 4.

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
- Assumptions and risks, known limits, and unresolved questions with what depends on them.
- References supporting factual claims: the artifact and section that holds the evidence, or a repository path and line, documentation URL or recorded run result where no artifact covers the claim. Distinguish verified facts from assumptions and unresolved questions.

Exclude implementation code, pseudocode and low-level definitions such as exact function signatures, field schemas or internal algorithms. Explain enough to guide implementation while leaving those details for later.

Keep the plan short enough to read in one sitting. State a fact once and cite its evidence; do not repeat in the plan the evidence an artifact already holds. Describe the decided design: an unresolved question gets its default and what depends on it, not a second design.

## 5. Review the built plan; cap fixes

Do not review jobs that produce artifacts: mapping, research, prototypes or draft sections. Their output is data for the plan, not a deliverable. Take it as returned and pass it on; if an output is unusable or misses its scope, re-delegate the job rather than review it.

Review only the plan, once it is assembled. A review round is one or more reviewers run in parallel, none of them the plan's author. Give each the brief, the plan and the artifacts it draws on, and assign it one lens. Reviewers read the repository and may confirm a finding by running a scratch experiment outside it; they do not edit the repository or the plan. The lenses:

- **Gaps.** Before reading the plan, list from the brief and the code what a complete plan must cover. Then report what the plan misses.
- **Fit.** Explore the affected areas independently of the map. Report callers, side effects, conflicting patterns and behaviour changes the plan does not account for.
- **Failure.** Report error paths, concurrency, data states, permissions and partial rollout the plan leaves unhandled, and anything else that would surprise once built.
- **Scope.** Trace every element to a requirement or a verified constraint. Report what does not trace, what rebuilds something the app already has, detail below the required level, and length that comes from repeating evidence or designing for undecided alternatives.
- **Grounding.** Report where the plan does not follow from the artifacts, contradicts them or rests on unsupported assumptions, and where sections disagree. Check the facts in the artifacts where needed.

The first round reviews the whole plan. It covers gaps and fit; add the other lenses where the risk is. For a small goal one reviewer may carry several lenses.

Require `PASS` or `MUST_FIX` from each reviewer. A finding is must-fix only when the plan, built as written, would fail a requirement or an acceptance criterion, break behaviour it says stays unchanged, rest on a claim the code or an artifact contradicts, or break a condition of section 1. Each must-fix finding gives the scenario, what it breaks and its evidence, and says which part of it, if any, is inferred. An existing defect or a failure path that touches no requirement or acceptance criterion is reported as a limit for the plan to state. That and everything else is an optional suggestion and never starts a fix round. The round passes only when every reviewer passes.

Track attempts on the plan: **one initial assembly plus at most three fixes**, each followed by a review round.

On `MUST_FIX`, give every finding a disposition before delegating anything. Merge the findings reviewers share, then mark each:

- fix: the plan changes;
- limit: the plan states it as a known limit or a risk, with its consequence;
- groundwork: a fact is missing, so research or a prototype comes first;
- user: the decision is the user's;
- rejected, with the evidence against it.

Prefer a stated limit to a new mechanism where the finding is a rare failure path or an existing defect.

A fixing subagent does not have to come next. Where findings call for groundwork or for backtracking, first delegate the jobs that supply it, in whatever order and number you judge right. These are artifact jobs: they are not reviewed and do not count as attempts.

Then copy the current plan to a numbered backup and send a new fixing subagent the plan, the brief, all artifacts, and the findings with their dispositions. Require it to:

- edit the plan in place, not rewrite it;
- make the smallest change that carries out each disposition;
- write a change log: per finding, what changed and where;
- name in the change log every design change, which is a fix that adds a component, a state or a change to another module.

Review the revised plan independently. A round after a fix is narrower than the first. Give its reviewers the earlier findings with their dispositions and the change log. They confirm each finding is resolved, then review the changed sections and what those touch, since a fix can open new gaps, and trace the callers and side effects of every design change the change log names. Rerun fit, and failure if it was used, as separate reviewers. One more reviewer carries gaps and any scope or grounding lens used so far; it does not re-check the evidence of unchanged claims. On `PASS`, accept the plan.

If the fourth review round still reports `MUST_FIX`, stop and report the remaining findings and the partial plan to the user. Do not reset the limit by renaming or splitting the plan, or present it as complete.

## 6. Deliver

Deliver the reviewed plan at the requested output path, or in the conversation if none was given, and identify any unresolved questions. List the branches and worktrees left by prototype jobs; they are the user's to remove. Finish after the plan passes review or the attempt limit requires escalation. Stop there; do not implement it.
