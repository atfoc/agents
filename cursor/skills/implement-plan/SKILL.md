---
name: implement-plan
description: Implement a plan file that is written as a list of vertical slices — read the slices, ask how the work should be split across sequential subagents, then build them in order. Use whenever the user points at a plan file and asks to implement it, execute it, build it, or work through its slices.
disable-model-invocation: true
---

# Implement a plan

The plan is whatever the user pointed at: a path in this conversation, a file they opened, or the skill argument. If there is no plan path, ask which plan file to implement and stop — do not go looking for one and do not start guessing at work.

A plan is a list of **vertical slices**. A slice is one unit of work whose operations depend on each other so tightly that splitting it would cost more than doing it in sequence. A slice is never split. Slices are always done in plan order.

## Step 1 — Read the plan and list the slices

Read the plan file end to end. Write out the slices you found as a numbered list, one line each, using the plan's own titles.

If the file is not a slice-structured plan, or you cannot tell where one slice ends and the next begins, say so, show what you did find, and stop.

## Step 2 — Ask how to split the work

Before implementing anything, ask the user how the work should be executed and wait for their answer. Never skip this and never assume a default. Put the two choices to them plainly:

1. **One subagent per slice.** Each slice gets its own subagent, run one after another.
2. **Inline — no subagents.** You implement every slice yourself, in this conversation.

List the one you recommend first, mark it `(Recommended)`, and give the reason alongside it.

Recommend inline when the plan has 3 or fewer slices, or when the slices are mostly edits to existing files rather than new packages — a single context never grows large enough for the per-subagent restart cost to pay off. Recommend one subagent per slice at 4 or more slices with substantial new code in each.

Skip the question only when the user already stated the mode in their request ("implement plan.md, each slice in its own subagent"). Honour what they said.

If you are already running as a subagent you cannot spawn further subagents — nesting stops after one level. Say so, implement everything inline, and continue.

## Step 2.5 — Split the plan into per-slice files

Subagent mode only. If the plan carries no `**Used by:**` annotations, skip this step entirely and pass the plan path to each subagent as before.

Spawn one plain general-purpose subagent with this prompt and wait for it to return. It writes files; it implements nothing.

```
Split <pathToPlan> into one self-contained file per slice under <planDir>/slices/.

Read the whole plan first.

A section runs from its heading to the next heading of any level. Sections scoped to
particular slices carry a `**Used by:** slices <list>` line under the heading; table
rows carry a `Used by` column. Unannotated sections and rows are shared. Where a
section and one of its subsections are both annotated, the subsection's list wins.

For each slice N write `slices/slice-<N>.md` containing, in plan order, every shared
section and every section or row whose `Used by` includes N. That one rule covers the
whole document, slice blocks included.

COPY EVERY LINE VERBATIM. Do not summarise, condense, reword, reformat, re-order or
otherwise improve anything you carry across. The plan's value is in its exact figures
and signatures — an implementer writes test assertions directly from them, so a
paraphrase that reads better is a defect. Prefer copying whole sections with
`sed -n '<start>,<end>p'` over retyping them.

Use judgment for exactly three things, and report each one:
  1. An annotation that is missing or wrong — put the section where it clearly belongs.
  2. A mostly-shared section containing a passage only one slice needs — carry that
     passage verbatim into that slice's file.
  3. A one-line `**Builds on:**` note at the top of each slice file naming which
     earlier slices' code it will use. This is the only prose you author.

Verify before reporting:
  - A plan with K slices has exactly K sections annotated with a single slice number
    under a `### Slice` heading. If not, say which are missing.
  - For each file, run
        comm -23 <(sort -u slices/slice-<N>.md) <(sort -u <pathToPlan>)
    Every line it prints must be one you deliberately authored under rule 3. Anything
    else is a paraphrase — fix it by copying the original.

Report each file's path and line count, every judgment call you made, and the
verification output for any file that was not clean. Implement nothing, and do not
modify the plan.
```

Check the report lists one file per slice before continuing. If the splitter reports a missing or wrong annotation, relay it to the user — that is the plan needing a fix, and it will matter the next time it is implemented.

## Step 3 — Implement

### Inline mode

Work through the slices in order. Finish a slice completely — code, wiring, and whatever the plan says verifies it (tests, build, lint) — before starting the next one. After each slice, tell the user in one or two lines what changed.

### Subagent mode

One subagent at a time. Never run two in parallel, never start the next one before the previous has returned. Spawn each one as a plain general-purpose subagent — do not pick a specialised type. If this environment gives you no way to spawn a subagent, tell the user and implement inline instead.

Use this prompt for each subagent:

```
You are implementing vertical slice <N>: <slice title>. Everything you need is in
<sliceFilePath>.

Before writing code, read <planDir>/slices/interfaces.md if it exists. It holds the
exact exported signatures earlier slices actually built — trust it over the slice
file's prose, and use it instead of opening those packages to look them up.

Read <sliceFilePath> and implement slice <N> only. Do not read the full plan file.
Finish the whole slice, including whatever it says verifies it (tests, build, lint).

When the slice verifies, append a `## Slice <N> — <packages>` section to
<planDir>/slices/interfaces.md: every exported symbol you added or changed, one line
each, signature copied verbatim from the code, no bodies and no prose. Add a trailing
comment only where error or nil behaviour is not obvious from the signature. If a
`## Slice <N>` section already exists, replace it.

Report back: what you changed, what you ran, what passed or failed, anything left
undone, and how many symbols you appended. Do not repeat the signatures in your report.
If <sliceFilePath> was missing something you needed, say so explicitly.
```

Substitute the real slice number, title and paths. If Step 2.5 was skipped because the plan carries no annotations, replace `<sliceFilePath>` with the plan path and drop the "Do not read the full plan file" line.

After each subagent returns, relay a short summary of what it did to the user — its report is not shown to them.

## Step 4 — Stop on failure

If a slice fails — the subagent reports it could not finish, or verification fails — stop. Do not start the next slice or the next subagent. Report which slice failed, what the failure was, and which slices are already done. Wait for the user.

## Step 5 — Report

When the last slice is done, report: every slice with its status, whether it ran inline or in a subagent, what changed on disk — including the generated `slices/` directory and `slices/interfaces.md` — and anything the plan asked for that you did not do. Then stop.

## Rules

- Treat the plan file as read-only. Do not rewrite it while implementing.
- `slices/` and `slices/interfaces.md` are generated working files, not the plan. The read-only rule covers the plan file only.
- Never split one slice across two subagents.
- Never reorder slices.
- Do not commit or push unless the user asks.
- Do not add work the plan does not call for. If the plan is wrong or a slice is blocked, say so and stop rather than improvising around it.
