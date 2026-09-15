# Prototyping

Answering a question that words cannot settle by building it in the real project. The code is the
means; the answer is written in words.

There are two modes:

- **Without a human in the loop** — the question has an answer that running code can show: whether
  something works, whether it can work, how it behaves. Built and settled alone.
- **With a human in the loop** — the question needs a person's judgment: how something looks, how
  it feels, which of several directions is right. Built as a series of variants the user reacts to
  and can move between.

## Inputs

- **The question**, including what settled looks like. No question: say it is missing, write that
  to the output location, and stop.
- **The mode.** When not given, take it from the question: with a human in the loop when it needs a
  person's judgment, without one otherwise.
- **The name** — short kebab-case. When not given, infer it from the question.
- **The output location.** When not given, `./.prototypes/{name}.md`.

## Setup

- Create a branch and a worktree, both named `{name}`. Every attempt is built there and nowhere
  else.
- Commit every attempt on that branch as it is made, with a message saying what it is.

## Without a human in the loop

- Build in the worktree until the question has an answer.
- Settle it by running the code — a test, a script, the app — never by reading the code and
  reasoning about what it would do.
- Do not ask the user. The question and what settled looks like are the whole assignment.
- Stop when the question is answered, or when it cannot be settled from what was given — then say
  what is missing.

## With a human in the loop

Every round produces a variant: one version the user can judge.

- Build one variant. Commit it as `variant <n>: <what it is>`, add it to the variant list in the
  output, and show it to the user in whatever form lets them judge it — run it, capture it, or
  point at the code.
- Take what the user wants next: a change, a new direction, or another look at an earlier variant.
  Build the next variant from the one they name — the latest when they name none — and show it.
- When asked, list the variants, or put the worktree back on any one of them and show it again.
- A variant built from an earlier one starts from that variant's files and is committed on top of
  the branch, so the branch stays linear and no variant is lost.
- When variants can sit side by side — screens, components, copy — the prototype may also carry a
  switch between them, so the user can compare without asking. The commits stay the record.
- Repeat until the user names the variant that answers the question. Whether it is answered is the
  user's call, never yours.

## Output

Written to the output location. With a human in the loop it is rewritten after every variant, so
the variant list can be read while the session is still going.

- the question and the mode;
- the branch and worktree name;
- what was tried — without a human, one line per attempt; with a human, the variant list: number,
  one line on what it is, its commit, and how to see it;
- the answer — without a human, what was found and the evidence: what was run and what it showed;
  with a human, the chosen variant and what it settled, in words, so the answer is usable without
  reading the code.

## Rules

- The code is a reference. It is never merged; the answer is the words in the output.
- Every attempt stays reachable in the branch. Nothing is discarded because a later one was
  preferred.
- Prototyping never removes its branch or worktree. Whoever asked for the prototype decides when
  they go.
