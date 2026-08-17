---
name: make-plan-v2
description: Turn a spec into a vertical-slice implementation plan saved to ./.tasks/<task-name>/plan.md. Writes an ordered list of vertical slices — each naming the exact files, lines and code structures to change plus its own unit-test definition — and reports a high-level overview of every slice in chat. Use whenever the user has a spec, feature request, or change description and wants a plan written to disk before any code is written.
---

# Make a plan (vertical slices)

Turn a spec into a plan made of vertical slices, specific enough that someone else could implement any slice without re-deciding anything. The plan goes to a file; a high-level overview goes to the user.

The spec is whatever the user gave you: the skill argument, a file they pointed at, or the request in the conversation. If there is no spec, ask for one and stop.

Write nothing except the plan file. No task files, no notes, no code.

## Step 1 — Name the task

Infer the task name from the spec: lowercase kebab-case, two to four words, naming what the spec delivers, not how. "Add OAuth login with Google" → `oauth-login`. "Speed up the report export" → `report-export-performance`.

The plan file is `./.tasks/<task-name>/plan.md`, relative to the project root. Create the directory.

If that file already exists, do not overwrite it. Tell the user it exists and ask whether to replace it or write under a different name, and wait for the answer.

## Step 2 — Learn enough to plan

The plan below demands real file paths, real signatures, real line numbers, real test commands. You cannot write that from assumptions, so find out first: how the relevant part of the code is structured and where this spec plugs into it, what already exists that it should reuse, what the project docs mandate, the interface of anything external the spec depends on, and how this project writes and runs its unit tests.

**How you get that is your call** — read the files, search, delegate, or any mix. The bar is not the method, it is the plan: every path, name and number in it must come from something you actually looked at.

## Step 3 — Cut the spec into vertical slices

Do this yourself, in the main agent. **Do not delegate it.** You have the spec and everything you just learned in context; that is what the plan is made of.

**A vertical slice is one thin cut through every layer the spec touches, delivering one behaviour that can be demonstrated and tested on its own.** Schema, service, endpoint and UI for a single capability is a slice. "All the database changes" is not — that is a horizontal layer, and a plan made of layers cannot be verified until the last one lands.

Cut them so that:

- **Each slice delivers something observable.** State what works after it that did not work before.
- **Each slice is testable alone**, with unit tests written in that same slice.
- **The slices are ordered**, and each names the earlier slices it depends on. Prefer the slice that proves the riskiest assumption first.
- **A slice is one unit of work.** If a slice's changes would span far more than a handful of files, or read as two unrelated behaviours, split it.
- **Together they cover the whole spec** and nothing beyond it.

## Step 4 — Write the plan file

Write the plan to `./.tasks/<task-name>/plan.md`. It is a handoff document: the implementer has not read the spec and will not see this conversation, so everything they need is in the file.

Use this structure:

```markdown
# <Task title>

## Goal
<What this delivers and why, in a few lines.>

## Decisions
<Each decision that shapes the implementation, one line each, with the reason. Approaches considered and rejected go here, not in the slices.>

## Slices

### Slice 1 — <name>
**Delivers:** <the behaviour that works after this slice.>
**Depends on:** <earlier slice numbers, or "nothing".>

**Changes:**
- `path/to/file.ts:120-148` — <what changes in `functionName`, and what it becomes.>
- `path/to/new_file.ts` (new) — <what it contains: exported names, signatures, types.>

**Tests:**
- `path/to/file.test.ts` (new) — <each case: input, expected result.>
- Run: `<the project's test command for this scope>`

**Done when:** <the checkable condition.>

### Slice 2 — <name>
...

## Out of scope
<What this plan deliberately does not do, so nobody widens the work later.>

## Open questions
<Only decisions that are critical to the implementation, that the spec does not settle and that you could not answer from the code, the docs or the outside world. Each with your recommended answer and why. Omit the section if there are none.>
```

Hold every slice to this bar:

- **Specific.** Real file paths, real function and type names, real signatures. Where you know the line numbers, use them. Line numbers are the plan's job, not the implementer's.
- **Decided.** Never "we could do X or Y". Pick one; the reason belongs in Decisions.
- **Self-contained.** Name the conventions and interfaces the slice must fit, since the implementer cannot see what you saw.

### Testing rules

Every slice defines its own tests, and they are **unit tests only**.

- Unit tests for the code the slice writes, plus the existing unit tests that code affects.
- **UI is tested as UI units** — one component mounted in isolation with its inputs stubbed, using whatever this project already uses (Playwright component testing, Testing Library, Vitest + jsdom, or similar).
- **Never end-to-end tests, never smoke tests, never full-app or full-browser flows.** They take too long and provide little value here. If a slice seems to need one, the slice is cut wrong — re-cut it so its behaviour is reachable by unit tests.
- Name the exact test cases, not "add tests". State the command that runs them.

## Step 5 — Report to the user

After the file is written, reply in chat with:

1. **The plan file path on the first line**, so the user can reference it immediately.
2. **One entry per slice**, in order: the slice name, what it delivers, and — at a high level — what it changes through the code, layer by layer (which modules or boundaries it touches, and how). Keep it to a few lines each. No line numbers, no file-by-file detail — that lives in the plan file.
3. **The open questions**, if the plan has any, each with your recommended answer so the user can confirm in a word.

Then **stop**. Do not start implementing, and do not split the slices into tasks — that is `implement-plan`'s job. If the user answers the open questions, fold the answers into the plan file, restate in chat only the slices that changed, and stop again.
