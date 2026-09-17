---
name: break-down-work
description: Cuts one large chunk of work — an implementation spec, or any written body of work — into vertical slices, each with its own title, self-contained body, verification, blockers and a call on whether it is `for-agent`. Use when a spec or a big piece of work has to be broken down into individual tasks that can be picked up one at a time.
argument-hint: [path to the work to split]
---

# Break work down into vertical slices

Split the work the user gave you: `$ARGUMENTS`, or the spec or chunk of work named earlier in the conversation. If no work is named, ask what to split and stop.

## What a slice is

A **vertical slice** is a collection of work that builds one functional and testable unit of the whole body of work. Every slice has:

- **Its own verification.** A slice is done when its verification passes. Its verification is automated — tests, type checks, builds — and never needs a human. The only verification-only slices are the ones described in Step 3a.
- **A body that is the whole assignment.** Whoever implements a slice sees that body and nothing else — not the implementation spec, not the feature definition, not any other slice, not a reason why it was cut this way.
- **A title**, one line.
- **Blockers** — the slices that must be completed before it may start.
- **`for-agent` or not.** A slice is `for-agent` when an agent can do all of it, verification included, with no human in the loop. Every other slice is not `for-agent`.

## Step 1 — Read the whole source

Read the whole of the work being split before cutting anything. Do not start cutting from a partial read.

Establish where the slices go. The user names the destination — a task store, a folder, a file. Never default or invent one. If the destination is not named, ask for it and stop.

## Step 2 — Cut

- Work that cannot be verified on its own folds into the slice whose verification exercises it.
- **Never re-cut a slice to gain parallelism.** Keeping the change → verify → fix loop inside one implementer's context beats handing pieces around. Parallelism is only what falls out of slices that were already independent.
- Where a slice depends on work that is not cut yet, stub that branch rather than widening the slice to cover it. The body names the stub, so the slice that later replaces it knows what it is replacing.

## Step 3 — Write each body

- Carry text from the source **verbatim**. No summarising, no rewording, no "see the source".
- Carry the reason for the work and its user-visible behaviour verbatim wherever a slice needs them.
- You may add your own detail, and added detail only narrows or sequences what the source already decided.
- Never add anything that contradicts, weakens or reinterprets the source. If a workable slice would require that, stop, write nothing, and report what conflicts with what.
- Never name the source's path in a body. The body has to stand alone.
- **Duplication between slices is expected and correct.** A section that applies to four slices is copied into all four; a slice is never made smaller by pointing at another slice's body.

## Step 3a — Move manual verification out of the slices

Never put manual verification — clicking through the app, eyeballing output, calling an endpoint by hand — in a slice body. Wherever the source asks for it, or a slice would need it, cut it out and handle it one of two ways.

**When an agent can simulate it**, create a separate **simulated verification slice**, blocked by the slice it verifies. It is `for-agent`. Its body is written so an agent does it end to end with no human:

- What behaviour to confirm, and what the pass and fail results look like.
- That the agent writes a throwaway script that simulates the user: driving the UI (for example with a browser automation tool) for interactive behaviour, or sending HTTP requests or calling the CLI for functional-only behaviour. The script is not committed.
- Every piece of environment setup and state the check needs — services to run, config, seed data, accounts, feature flags — and how to get there, so the agent does that prep before running the check.
- That the slice is done when the script passes, and that failures are reported with what the script saw.

**When no agent can run it**, even as a one-off — it needs physical hardware, a real third-party account only a human holds, a human judgement call — it goes into one **manual verification slice** at the end of the work. There is exactly one of these, and only if something needs it. It collects every such check across all slices, each with what to confirm, the setup it needs, and the pass result. It is **not** `for-agent`. It is blocked by every slice whose work it verifies and blocks nothing, so no other slice ever waits on a human.

The slice the check was cut from keeps its automated verification and is not blocked by either of these.

## Step 4 — Decide blockers

A slice is blocked by another for exactly three reasons:

1. It consumes an artifact the other produces — a type, a table, a stub it replaces, a module it imports.
2. The two would edit the same file and so cannot run at the same time.
3. It is a simulated or manual verification slice from Step 3a, and the other is a slice it verifies.

Nothing else is a blocker. Nothing is ever blocked by the manual verification slice. A slice is startable when every slice blocking it is complete.

Apply all three reasons to every pair of slices. Where two slices touch the same file and the order is arbitrary, pick one and say nothing more about it.

A cycle means a bad cut: merge the slices or cut them differently. A cycle that cannot be resolved is a stop-and-report.

## Step 5 — Check before writing anything

All of these, before a single slice is written down:

- Every section of the source appears in at least one body — the deduplicated union of all bodies is the whole source. An orphaned section is a stop, not a silent drop.
- No cycles in the graph of blockers.
- At least one slice with no blockers.
- Nothing contradicted.
- No slice body asks a human to verify anything, except the one manual verification slice.
- Every slice is marked `for-agent` or not, and the manual verification slice is not `for-agent`.

Any failure → write nothing and report why.

## Step 6 — Write the slices

Write every slice to the destination from Step 1, each with its title, its body, its blockers, and whether it is `for-agent`.

## Step 7 — Report

How many slices and which are not `for-agent`, every piece of detail you added beyond the source, and anything you had to stop on. Never report the bodies back — they are the slices.

The skill is finished once that report is given, and it stops there.
