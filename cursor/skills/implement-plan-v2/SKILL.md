---
name: implement-plan-v2
description: Implement a plan file with a single worker subagent, then have a thinker subagent review the resulting diff against the plan and write its findings to a file next to the plan. If the review finds gaps, run one more worker-then-review round using those findings as the handoff. Use when the user points at a plan file and wants it built and checked, or asks for implement-plan-v2.
---

# Implement a plan (worker, then review)

Build a plan file with one `worker` subagent, then have a `thinker` subagent review the diff against the plan and write its findings to a file. At most two rounds: implement → review → (if needed) implement → review → report.

The plan is a **file path**: whatever the user pointed at — the skill argument, a path they named in the conversation, or the plan file a previous skill just wrote. Resolve it to an absolute path and confirm the file exists. If there is no plan file — or the plan only exists in the conversation and not on disk — say so and stop. This skill implements a plan file; it does not invent one and does not accept a plan pasted into chat.

**You never implement and never review yourself.** Your whole job is: capture the baseline, brief the subagents, read the review file, decide whether to run a second round, report. Do not edit project files, do not fix findings by hand, and do not run the plan's tests yourself.

If this skill is itself running as a subagent, it cannot spawn further subagents — nesting stops after one level. In that case do all three roles yourself, sequentially and in order: implement the plan, then capture the diff and review it against the plan under the same verdict and output rules, writing the findings file all the same, then address any findings and review once more. Say in your report that you ran it in-place without subagents.

## Step 1 — Capture the baseline

Before any subagent runs, in the project root:

1. `git rev-parse HEAD` — note this sha. Call it **BASE**. You will substitute the literal sha into later commands.
2. `git status --porcelain` — note every file that is already modified or untracked. This is **pre-existing work**, not part of what the worker builds.

If the project is not a git repository, say so and tell the user the review will be scoped to the files the plan names instead of a diff, then continue.

## Step 2 — Round 1: implement

Spawn **one** subagent of the custom type `worker`. Spawn that custom type by name — never a generic or built-in agent type — and do not override its model or effort; the `worker` type sets its own, and an override silently replaces the agent this skill is built around.

One worker, not a fan-out. Do not split the plan into tasks and do not launch parallel workers — that is `implement-plan`'s job, not this skill's.

Its prompt is the plan file path and nothing else you have not been told to add:

> Implement the plan at `<absolute path to plan file>`. Read the plan file first, then implement all of it — every slice, in the order the plan gives, including the tests the plan defines and the test commands it names. Report what you implemented, the test results as they actually came out, and anything you could not do.

Do not paste the plan's contents, do not summarize it, and do not add context from this conversation. The worker reads the plan itself.

Wait for it to finish.

## Step 3 — Round 1: review

Produce the diff of what changed:

1. `git add -N .` — registers untracked files as intent-to-add so new files appear in the diff. It stages no content.
2. `git diff --stat <BASE>` — read this yourself; it is the scope summary you pass on.

Do **not** run the full `git diff` yourself. The reviewer pulls the diff.

Spawn one subagent of the custom type `thinker` — by name, with no model or effort override, for the same reason as above. Its prompt contains:

- **The diff.** The exact command to get it: `git diff <BASE>` (literal sha substituted), run from the project root. Include the `--stat` output inline so the reviewer knows the scope up front, plus the list of files that were **already modified or untracked before implementation started** — those are not the worker's work and are not under review.
- **The plan file path**, absolute.
- **The job:** decide whether the plan was implemented as written. Read the plan, read the diff, and check each slice: is it there, does it do what the plan says, does it use the files, names, signatures and values the plan specified, and are the plan's tests written. Also flag anything in the diff that contradicts the plan or goes beyond it.
- **The verdict rules**, quoted below.
- **The output rules**, quoted below.

Quote these two blocks into the reviewer's prompt as its rules:

> **Returning no comments is a completely valid and expected outcome.** If the plan was implemented correctly, say exactly that and stop. Do not manufacture findings to look thorough, do not report style preferences, do not suggest improvements the plan did not ask for, and do not report anything the plan explicitly put out of scope. A clean review is a success, not a failure to find something. Only report a gap between the plan and the code.

> **Write your findings to a file** in the same directory as the plan file. Pick a name that does not already exist in that directory — check first, and include the round number and a timestamp, e.g. `review-round-1-20260817-142530.md`. Never overwrite an existing file. Start the file with:
>
> ```markdown
> # Review — round <N>
> **Verdict:** COMPLETE
> ```
>
> Use `COMPLETE` when nothing needs to change, and `INCOMPLETE` when anything does. Then the findings.
>
> **Your findings are a handoff, not a note.** The next agent to act on this file will receive *only* this file — it will not see the plan, the diff, this conversation, or you. Write every finding so it can be implemented from your text alone: the exact file path and location, what is wrong or missing, what the correct implementation is with the concrete names, signatures, structures and values it needs, the surrounding conventions and interfaces it must fit, which tests to write or fix and the command that runs them, and what done looks like. Quote the slice of the plan the finding comes from rather than referring the reader to the plan. Repetition between findings is fine; a reader guessing is not. If the verdict is `COMPLETE`, the file needs nothing beyond the verdict and a sentence on what you checked.
>
> **Return only the absolute path to the file you wrote** as your final answer.

Wait for it, then read the file at the path it returned.

## Step 4 — Round 2, only if the review found something

If the verdict is `COMPLETE`, skip to Step 5.

If it is `INCOMPLETE`, run exactly one more round:

1. Spawn one subagent of the custom type `worker` — by name, no overrides. Its input is **the review findings file path, not the plan file path**:

   > Address every finding in `<absolute path to the review file>`. Read that file first, then implement all of it, including the tests it calls for and the test commands it names. Report what you changed, the test results as they actually came out, and anything you could not do.

   Do not also hand it the plan file, and do not paste the findings into the prompt.

2. When it finishes, review again exactly as in Step 3 — a fresh `thinker`, same diff command against the same **BASE** sha, same plan file path, verdict and output rules unchanged, `round 2` in the file header so the filename and heading differ from round 1. Additionally give it the round-1 review file path, with this instruction: judge the code against the plan, and use the earlier review only to check whether its findings were actually addressed.

3. Read the file it returns. **Stop here either way.** If this second review is still `INCOMPLETE`, do not run a third round and do not fix it yourself — report it.

## Step 5 — Report

Reply in chat with:

1. The plan file path and how many rounds ran.
2. What each worker reported it built, and the test results as they actually came out — failures reported as failures.
3. The review outcome per round, with the review file path so the user can open it.
4. If the final review is `INCOMPLETE`: the outstanding findings, summarized, and the review file path that holds the detail. State plainly that they were not addressed and that the skill stopped after two rounds.

Then stop.
