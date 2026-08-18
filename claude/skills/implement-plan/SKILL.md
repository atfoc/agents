---
name: implement-plan
description: Implement a plan file that is written as a list of vertical slices — read the slices, ask how the work should be split across sequential subagents, then build them in order. Use whenever the user points at a plan file and asks to implement it, execute it, build it, or work through its slices.
argument-hint: [path-to-plan]
disable-model-invocation: true
---

# Implement a plan

The plan is whatever the user gave you: the skill argument, a path in the conversation, or a file they pointed at earlier. If there is no plan path, ask which plan file to implement and stop — do not go looking for one and do not start guessing at work.

A plan is a list of **vertical slices**. A slice is one unit of work whose operations depend on each other so tightly that splitting it would cost more than doing it in sequence. A slice is never split. Slices are always done in plan order.

## Step 1 — Read the plan and list the slices

Read the plan file end to end. Write out the slices you found as a numbered list, one line each, using the plan's own titles.

If the file is not a slice-structured plan, or you cannot tell where one slice ends and the next begins, say so, show what you did find, and stop.

## Step 2 — Ask how to split the work

Before implementing anything, use `AskUserQuestion` to settle the execution mode. Never skip this and never assume a default.

Ask one question, header `Execution`, with these options:

- **Inline — no subagents.** You implement every slice yourself, in this conversation.
- **One subagent per slice.** Each slice gets its own subagent, run one after another.
- **Grouped slices per subagent.** Slices are bundled into groups; each group gets one subagent, run one after another.

If the user picks grouped, propose a grouping from the plan — contiguous slices only, in plan order — as a second question, and let them override it. Keep proposing until they accept a grouping.

Skip the question only when the user already stated the mode in their request ("implement plan.md, each slice in its own subagent"). Honour what they said.

If you are already running as a subagent, you cannot spawn further subagents: say so, implement everything inline, and continue.

## Step 3 — Implement

### Inline mode

Work through the slices in order. Finish a slice completely — code, wiring, and whatever the plan says verifies it (tests, build, lint) — before starting the next one. After each slice, tell the user in one or two lines what changed.

### Subagent mode

One subagent at a time. Never run two in parallel, never start the next one before the previous has returned. Use the `Agent` tool with **no** `subagent_type`.

The prompt for each subagent starts with this line, then the fixed instructions below it:

```
You are implementing vertical slice <N>: <slice title>. Here is the path to the plan: <pathToPlan>

Read the plan file yourself. Implement only slice <N> — leave every other slice alone.
Finish the whole slice, including whatever the plan says verifies it (tests, build, lint).
Report back: what you changed, what you ran, what passed or failed, and anything you left undone.
```

For a group, name the whole range in the first line — `You are implementing vertical slices <N>-<M>: <titles>. Here is the path to the plan: <pathToPlan>` — and implement them in order inside that one subagent.

Substitute the real slice number, title, and plan path. Pass the plan path exactly as it was given to you.

After each subagent returns, relay a short summary of what it did to the user — its report is not shown to them.

## Step 4 — Stop on failure

If a slice fails — the subagent reports it could not finish, or verification fails — stop. Do not start the next slice or the next subagent. Report which slice failed, what the failure was, and which slices are already done. Wait for the user.

## Step 5 — Report

When the last slice is done, report: every slice with its status, whether it ran inline or in a subagent, what changed on disk, and anything the plan asked for that you did not do. Then stop.

## Rules

- Treat the plan file as read-only. Do not rewrite it while implementing.
- Never split one slice across two subagents.
- Never reorder slices.
- Do not commit or push unless the user asks.
- Do not add work the plan does not call for. If the plan is wrong or a slice is blocked, say so and stop rather than improvising around it.
