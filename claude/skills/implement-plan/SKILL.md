---
name: implement-plan
description: Execute an implementation plan. Splits it into focused subtasks grouped into ordered waves of parallel-safe work, then implements each wave with parallel worker subagents. Use when a plan exists (from make-plan, a plan file, or the conversation) and the user wants it built.
---

# Implement a plan

Take a plan and build it: split it into subtasks, group them into ordered waves, then run each wave as parallel `worker` subagents.

The plan is whatever the user pointed at: the plan already in this conversation, a file, or the skill argument. If there is no plan, say so and stop — this skill executes a plan, it does not invent one.

## Step 1 — Split the plan into subtasks and waves

Do this yourself, in the main agent. **Do not delegate the split** — not to a subagent, not to `thinker`. You hold the plan; splitting it is reading, not reasoning at a distance.

**A subtask is one unit of work that can be executed on its own.** Any bigger and it splits further; any smaller and it stops making sense on its own. Aim for a job a `worker` can finish without asking anyone anything.

**A wave is a set of subtasks that run at the same time.** Waves are ordered: wave 1 runs to completion, then wave 2, and so on. Order inside a wave is irrelevant — by construction.

Two subtasks may only share a wave if they are genuinely independent:

- **They never touch the same file.** Same file means different waves. No exceptions — two workers editing one file in parallel corrupt each other's work.
- **Neither consumes the other's output.** If T1 creates an API and T2 calls it, T1 is in an earlier wave. Same for a type and its user, a config key and its reader, a helper and its caller.
- **Neither's decisions constrain the other.** If one task's choice of shape or name determines how the other must be written, they are sequential, not parallel.

When in doubt, put them in different waves. A wasted wave costs a little time; a bad parallel pair costs a rewrite.

Number tasks uniquely across the whole plan (task 1..n), so a task id never repeats between waves.

Do not create tasks for end-to-end or smoke testing. Testing is unit tests only: new unit tests for the code a task writes, and the existing unit tests that code affects.

Show the user the wave/task breakdown — one line per task, grouped by wave — before you start.

## Step 2 — Write each task so a worker can execute it blind

A `worker` sees none of this conversation and has not read the plan. Whatever it needs must be in its prompt. Each task prompt contains:

- **The goal** — what this task must achieve, in a sentence.
- **The files it owns** — exact paths, marked create or edit. Plus an explicit "do not touch" list for the files that neighbouring tasks own.
- **What to do** — the concrete change, with the names, signatures, structures and values from the plan. Quote the relevant slice of the plan rather than referring to it.
- **The context it cannot see** — the surrounding conventions, the interfaces it must fit, what an earlier wave already built, and what a later wave will build on top of it.
- **The unit tests** it must write and the existing ones it must run.
- **Done means** — what has to be true for the task to be complete.

If two tasks need the same context, repeat it in both prompts. Repetition is cheap; a worker guessing is not.

## Step 3 — Run the waves

For each wave, in order:

1. Launch **one `worker` subagent per task, all in a single message**, so the whole wave runs in parallel.
2. Wait for every task in the wave to finish. The wave is a barrier — never start the next wave while one is still running.
3. Read the reports. Check the workers stayed inside their files, and that what they built actually matches what the next wave assumes.
4. If a task failed or came back short: fix it before moving on — re-run it as a new `worker` with a sharper prompt, or do the remaining piece yourself if it is small. Never carry a broken task into the next wave.
5. If a report invalidates the split — an unforeseen dependency, a file two tasks both need — re-plan the remaining waves before continuing, and tell the user what changed.

Then move to the next wave. Repeat until every wave is done.

## Step 4 — Report

When the last wave is done, run the project's unit tests once over the whole change and report:

- what was built, wave by wave;
- the test result, as it actually came out;
- anything a worker flagged, skipped, or decided differently from the plan;
- anything left undone, and why.

Report failures as failures. Do not paper over a wave that did not fully land.
