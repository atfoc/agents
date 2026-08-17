---
name: implement-plan
description: Execute an implementation plan. Splits it into focused subtasks grouped into ordered waves of parallel-safe work, then implements each wave with parallel worker subagents. Use when a plan exists (from make-plan, a plan file, or the conversation) and the user wants it built.
---

# Implement a plan

Take a plan and build it: split it into subtasks, group them into ordered waves, then run each wave as parallel subagents of the custom type `worker`.

The plan is whatever the user pointed at: the plan already in this conversation, a file, or the skill argument. If there is no plan, say so and stop — this skill executes a plan, it does not invent one.

## Step 1 — Split the plan into subtasks and waves

Do this yourself, in the main agent. **Do not delegate the split** — not to a subagent, not to `thinker`. You hold the plan; splitting it is reading, not reasoning at a distance.

**A subtask is one unit of work that can be executed on its own.** Any bigger and it splits further; any smaller and it stops making sense on its own. Aim for a job a `worker` can finish without asking anyone anything.

**Size is a second test, independent of coherence.** Coherence says what belongs in one task; size says what fits in one worker. If a task's output would exceed roughly one large file, or you would expect it to take more than ~50 tool calls, split it — even when it is perfectly coherent. A worker that outgrows its task does not slow down, it dies on the output-token cap mid-file, and the task is then paid for twice: once for the attempt that failed and once for the retry.

**A wave is a set of subtasks that run at the same time.** Waves are ordered: wave 1 runs to completion, then wave 2, and so on. Order inside a wave is irrelevant — by construction.

Two subtasks may only share a wave if they are genuinely independent:

- **They never touch the same file.** Same file means different waves. No exceptions — two workers editing one file in parallel corrupt each other's work.
- **Neither consumes the other's output.** If T1 creates an API and T2 calls it, T1 is in an earlier wave. Same for a type and its user, a config key and its reader, a helper and its caller.
- **Neither's decisions constrain the other.** If one task's choice of shape or name determines how the other must be written, they are sequential, not parallel.

When in doubt, put them in different waves. A wasted wave costs a little time; a bad parallel pair costs a rewrite.

Number tasks uniquely across the whole plan (task 1..n), so a task id never repeats between waves.

A task's testing is the unit tests for the code it writes, plus the existing unit tests that code affects. The plan decides whether anything beyond that is in scope — do not add test scope it does not call for. If the plan does call for end-to-end or integration testing, that is a task like any other: write it up per Step 2, spelling out exactly what is under test, how to run it, and what a pass looks like, and give it to a `worker`. Never run it yourself.

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

Your job across every wave is split, brief, verify, integrate — not implement. You hold the largest context and the most expensive model in the run, so the same work costs more done by you than by a `worker`. While a wave is running, never edit a file one of its workers owns: you collide with it exactly the way two parallel workers would, and its report will describe a file you have since changed underneath it.

For each wave, in order:

1. Launch **one subagent of the custom type `worker` per task, all in a single message**, so the whole wave runs in parallel. Spawn that custom type by name — never a generic or built-in agent type — and never pass a model or effort override on the spawn call. The `worker` type pins its own model and effort; an override on the call outranks them and silently replaces the agent this skill is built around.
2. Wait for every task in the wave to finish. The wave is a barrier — never start the next wave while one is still running.
3. Read the reports. Check the workers stayed inside their files, and that what they built actually matches what the next wave assumes.
4. If a task failed or came back short, read the report and diagnose before retrying — the fix depends on why it failed:
   - **It hit an output or context limit.** The task did not fit in one worker. Split it and run the pieces. Never re-run the same task with a sharper prompt: it hits the same wall and you pay for both attempts.
   - **It misunderstood the goal.** Re-run it as a new subagent of the custom type `worker`, with a prompt that closes the gap the report exposed.
   - **It landed almost everything.** Finish the remainder yourself only if it is genuinely small; otherwise spawn a `worker` for the remainder alone.

   Never carry a broken task into the next wave.
5. If a report invalidates the split — an unforeseen dependency, a file two tasks both need — re-plan the remaining waves before continuing, and tell the user what changed.
6. Work you discover mid-run becomes a task, not something you do by hand. A gap between two tasks, a fix a worker flagged but did not own, a follow-up its output makes necessary — write it up the way Step 2 describes, give it the next unused task number, and either run it now, if nothing in the current wave touches its files, or queue it into a later wave. Tell the user you added it. Splitting is not a one-off first step; it is how you absorb what the run teaches you.

Then move to the next wave. Repeat until every wave is done.

## Step 4 — Report

When the last wave is done, run the project's unit tests once over the whole change and report:

- what was built, wave by wave;
- the test result, as it actually came out;
- anything a worker flagged, skipped, or decided differently from the plan;
- anything left undone, and why.

Report failures as failures. Do not paper over a wave that did not fully land.
