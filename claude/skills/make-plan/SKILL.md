---
name: make-plan
description: Turn a spec into a concrete implementation plan. Explores first with parallel scout subagents (docs, codebase, external facts), then writes the plan itself and raises the critical open questions. Use whenever the user has a spec, feature request, or change description and wants a plan before any code is written.
---

# Make a plan

Turn a spec into a plan specific enough that someone else could implement it without re-deciding anything.

The spec is whatever the user gave you: the skill argument, a file they pointed at, or the request in the conversation. If there is no spec, ask for one and stop.

**Nothing is written to disk.** The exploration and the plan live in this conversation. Do not create plan files, task files, or notes unless the user asks for them.

## Step 1 — Decide what you need to know

Read the spec and list the questions that must be answered before the spec can become a plan. Not curiosity — blockers. Typical ones: how is the relevant part of the code structured, what already exists that this should reuse, what do the project docs mandate, what is the interface of an external thing the spec depends on.

## Step 2 — Send the scouts

Launch `scout` subagents to answer them. **Send them all in one message so they run in parallel.**

Always send:

- **A docs scout** — summarize only the documentation relevant to this spec, skipping unrelated docs, keeping exact file paths and line numbers. Skip this scout only if the project has no docs.
- **A codebase scout** — how the codebase is structured and where this spec plugs into it: the files, functions and boundaries it touches, and the conventions it must follow.

Then send **one scout per remaining question**. Use as many as the questions demand — there is no reason to squeeze several unrelated questions into one general exploration. Anything the spec references that lives outside the docs and the code (an external binary's flags, an API's shape, a file format) is its own scout.

Every scout prompt must be self-contained. A scout sees none of this conversation, so give it: the relevant part of the spec, the one question or goal it owns, and what its answer must contain (paths, line numbers, signatures, exact values). Vague scouts return vague summaries and the plan pays for it.

If a scout comes back thin or contradicts another, send another scout with a sharper question before you plan on top of it.

## Step 3 — Write the plan yourself

Write the plan in this conversation, in the main agent. **Do not delegate this step** — not to a subagent, not to `thinker`. You have the spec and every scout summary in context; that is what the plan is made of.

The plan must be:

- **Specific.** Real file paths, real function and type names, real signatures. If a scout gave you a line number, use it.
- **Decided.** Never "we could do X or Y". Pick one and give the reason in a line. An undecided plan is not a plan.
- **Ordered.** State what depends on what, so the work can be sequenced later.
- **Scoped.** Say what is explicitly out of scope, so nobody widens the work later.
- **Unit-tested only.** Testing in the plan is limited to writing new unit tests and running existing ones. Never include end-to-end or smoke testing — they take too long and provide little value here.

Structure it however the spec deserves; usually: goal, decisions, the change file by file, tests, out of scope.

## Step 4 — Ask the open questions

End with the open questions: the decisions that are **critical** to the implementation and that the spec does not settle and no scout could answer.

- Keep them few. If you can settle a question yourself from the spec, the docs or the code, settle it in the plan instead of asking.
- Never ask about something the answer to which would not change the plan.
- For each question give your recommended answer and why, so the user can confirm in a word.
- Questions only. Notes, findings and flags belong in the plan or in your reply, not in the question list.

Present the plan and the questions, then **stop**. Do not start implementing, and do not split the plan into tasks — that is `implement-plan`'s job. If the user answers the questions, fold the answers into the plan, restate the parts that changed, and stop again.
