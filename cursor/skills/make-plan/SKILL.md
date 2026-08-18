---
name: make-plan
description: Turn a spec into a handoff-ready implementation plan at ./.tasks/{task-name}/plan.md, cut into vertical slices. Use whenever the user hands over a spec, feature request, ticket, or change description and wants it planned out before any code is written — "plan this", "make a plan", "write a plan for X", "how would we implement this".
disable-model-invocation: true
---

# Make a plan

Produce one file: `./.tasks/{task-name}/plan.md`. It holds everything another agent needs to implement the spec without repeating your investigation, organised as vertical slices.

Write the plan. Do not implement it. Do not edit source files, do not run migrations, do not open PRs.

## Step 0 — Find the spec

The spec is whatever the user pointed at: the request in this conversation, a file or ticket they named, or the skill argument. If there is no spec, ask what should be planned and stop.

## Step 1 — Investigate before writing

Do the investigation yourself, in this context. Do not fan the work out to subagents — the reading you do is what makes the plan specific, and it is also the context you need for Step 3. This holds whether you are the main agent or already running as a subagent.

Read the actual code. The plan is worthless if it points at files that do not exist or invents function names.

Find and confirm, in the real repository:

- The entry points the spec touches — handlers, routes, commands, components, jobs.
- The call path from each entry point down to persistence.
- Existing structures to reuse: types, DTOs, error helpers, validation helpers, query builders, test fixtures, factories.
- The conventions this codebase already follows for the kind of change being planned — copy them, do not invent new ones.
- Anything in the spec that the current code makes impossible or expensive.

Record file paths with line numbers as you go. Every claim in the plan must trace back to something you read.

## Step 2 — Name the task

Infer `{task-name}` from the spec: lowercase, hyphens, 2–4 words, describing the change and not the ticket number — `add-bulk-export-endpoint`, `split-billing-service`, `fix-stale-session-cache`.

If `./.tasks/{task-name}/plan.md` already exists, read it and update it in place rather than starting over. Say so in the final report.

## Step 3 — Cut the work into vertical slices

A vertical slice is an ordered set of changes where **each step benefits heavily from the context the previous step just built**. Working through a slice top to bottom should feel like one continuous piece of work, not context-switching.

**One slice:** add the handler → change the service method signature the handler now needs → implement the logic in that method → unit-test it. Each step is decided by the one before it.

**A separate slice:** a second handler with a similar shape. It does not build on the first handler's service methods, so it starts from cold context.

**Also a separate slice:** the database work — migrations and repository/query changes — even when the same handler triggered it. It carries substantial work of its own and only loosely depends on the handler's context. Splitting it keeps one slice from exploding.

Rules for cutting:

- Order slices so a slice's dependencies land in an earlier slice. State the dependency explicitly (`depends on Slice 2`).
- If a slice's steps stop feeding each other, cut at the weak link.
- If a slice is large enough that a fresh agent would lose the thread, split it — bounded slices beat "complete" slices.
- Repetitive mechanical work that shares no reasoning (rename across 30 call sites, regenerate clients) is its own slice.
- Each slice should end in a state where the codebase builds and its unit tests pass.

## Step 4 — Plan the tests

- Unit tests only. Write them into the slice that produces the code they cover.
- Do not plan end-to-end tests, integration suites that need live services, browser automation, or manual QA steps.
- Do not plan automated tests for UI — this codebase does not test UI automatically. Say "no automated test — UI" on those steps.
- For each unit test, name what it asserts, not just "add tests".

## Step 5 — Write the file

Create `./.tasks/{task-name}/` and write `plan.md`. Write it for an agent who has not seen this conversation: no "as discussed", no references to earlier turns.

### Annotate what is slice-scoped

An implementer should be handed only the part of the plan it needs. Annotate any section whose content only some slices need with a line directly under its heading:

    **Used by:** slices 2, 5

Anything unannotated is shared — every implementer gets it. When the parts of a section have different answers, annotate the parts rather than the section; a child's annotation overrides its parent's. In a table, add a rightmost `Used by` column instead, so rows can differ.

Every `### Slice N` heading carries `**Used by:** slice N` — its own number and nothing else, even when a later slice builds on it. This is mechanical, not a judgment call: it lets the plan be filtered with one rule.

Never annotate `## Spec` or `## Key files`, or anything under them. Requirements are always shared, and Key files is the interface map — narrowing it makes an implementer open a file it could have known about.

A section no slice uses is dead weight. Delete it rather than annotating it.

```markdown
# <Task title>

## Spec
What was asked, restated completely enough to implement from. Include the user's own
constraints verbatim where wording matters.

## Out of scope
What this plan deliberately does not do.

## Investigation notes
How the relevant code works today, what the call path is, and what surprised you.
Facts a fresh agent would otherwise spend an hour re-deriving.

### <finding>
**Used by:** slices 2, 5
What you found, in full. Verbatim figures, signatures and identifiers — an implementer
writes assertions straight from these.

## Decisions already made
| Decision | Rationale | Rejected alternative | Used by |
| :-- | :-- | :-- | :-- |
Settled choices. An implementer should not reopen these.

## Key files
| Path:line | What is there | Why it matters |
| :-- | :-- | :-- |

## Reuse
Existing types, helpers, patterns, and test fixtures to use instead of writing new ones,
each with its path.

## Slices

### Slice 1 — <name>
**Used by:** slice 1
**Goal:** one sentence.
**Depends on:** nothing / Slice N.
**Why these steps are one slice:** the shared context that binds them.

**Steps**
1. `path/to/file.ext:120` — what to change, and what it must look like afterwards.
2. `path/to/other.ext` (new) — what it contains.
3. ...

**Tests**
- `path/to/file_test.ext` — asserts <behaviour>.

**Done when:** the checkable end state.

### Slice 2 — <name>
...

## Open questions
Anything genuinely unresolved, with the recommended default so implementation is not blocked.
```

Drop a section only when it is truly empty. An empty `Open questions` is a good sign; an empty `Key files` means Step 1 was skipped.

## Step 6 — Check and report

Before reporting, verify:

- Every path in the plan exists, or is explicitly marked `(new)`.
- Line numbers were read from the current files, not guessed.
- Every slice states why its steps belong together; no slice is a bag of unrelated tasks.
- Slice order respects the stated dependencies.
- Every section only some slices need carries a `**Used by:**` line or a `Used by` cell.
- Every `### Slice N` heading carries `**Used by:** slice N`.
- `## Spec` and `## Key files` carry no annotations.
- No end-to-end tests, no manual QA steps, no automated UI tests.
- No source file was modified.

Report the path to the plan and a one-line summary of each slice, then stop.
