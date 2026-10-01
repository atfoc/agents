---
name: prototype
description: Answers a question that words cannot settle by building it in the real project on its own branch and worktree — alone when running code can settle it, or as a series of variants the user judges — and writes the answer in words to an output location. Use when you want to find out whether something works, how it behaves, or how it should look and feel, by building it rather than reasoning about it.
---

# Prototype to answer a question

Answer a question that words cannot settle by building it in the real project. The code is the
means; the answer is written in words.

For an explicit invocation, use `$prototype <the question to settle>`. Act on the question the user gave you, or the one named earlier in the conversation. If there is no
question, say it is missing, write that to the output location, and stop.

## 1. Take the inputs

- **The question**, including what settled looks like.
- **The mode.** When not given, take it from the question: **with a human in the loop** when it
  needs a person's judgment — how something looks, how it feels, which of several directions is
  right; **without a human in the loop** when running code can show the answer — whether something
  works, whether it can work, how it behaves.
- **The name** — short kebab-case. When not given, infer it from the question.
- **The output location.** Write only where the user specifies. When not given, ask for it and
  wait — never assume one or fall back to a default.

## 2. Set up

- Create a branch and a worktree, both named `{name}`. Build every attempt there and nowhere else.
- Commit every attempt on that branch as it is made, with a message saying what it is.

## 3a. Without a human in the loop

- Build in the worktree until the question has an answer.
- Settle it by running the code — a test, a script, the app — never by reading the code and
  reasoning about what it would do.
- Do not ask the user. The question and what settled looks like are the whole assignment.
- Stop when the question is answered, or when it cannot be settled from what was given — then say
  what is missing.

## 3b. With a human in the loop

Every round produces one or more variants: versions the user can judge. The user compares them by
switching inside the running prototype — never by checking out commits.

- **Build a switch first.** Before the first variant, give the prototype one way to move between
  variants while it runs, showing which one is active by number and name:
  - with a UI, an on-screen control — a small floating picker or toolbar, with a keyboard shortcut
    to cycle — that swaps the variant in place, without a restart, and keeps the user's place;
  - without a UI — a CLI, an API, a library — a flag, an argument or an environment variable that
    selects the variant, with every variant runnable from the same checkout.
- **Keep every variant in the code at once.** Add each variant alongside the earlier ones behind the
  switch, never in place of them, so the latest commit holds all of them. When a variant comes from
  an earlier one, start from that variant's files and add it as a new entry.
- Build the variants asked for. When the user asks for several — "give me 3 options" — build them
  all in the same round, each a genuinely different direction, and put them all behind the switch.
  Otherwise build one. Commit each as `variant <n>: <what it is>`, add it to the variant list in the
  output, and show it to the user: run the prototype, say how to switch, and capture it when
  that helps them judge.
- Take what the user wants next: a change, a new direction, or another look at an earlier variant.
  Build the next variant from the one they name — the latest when they name none — and show it.
- When asked, list the variants, or switch to any one of them and show it again.
- The commits stay the record; the switch is how the user compares. The switch is scaffolding for
  judging, not part of the answer.
- Repeat until the user names the variant that answers the question. Whether it is answered is the
  user's call, never yours.

## 4. Write the output

Write to the output location. With a human in the loop, rewrite it after every variant, so the
variant list can be read while the session is still going. It holds:

- the question and the mode;
- the branch and worktree name;
- what was tried — without a human, one line per attempt; with a human, the variant list: number,
  one line on what it is, its commit, and how to switch to it;
- with a human, how to run the prototype and use the switch;
- the answer — without a human, what was found and the evidence: what was run and what it showed;
  with a human, the chosen variant and what it settled, in words, so the answer is usable without
  reading the code.

## Rules

- The code is a reference. Never merge it; the answer is the words in the output.
- Every attempt stays reachable in the branch. Discard nothing because a later one was preferred.
- Never remove the branch or the worktree. Whoever asked for the prototype decides when they go.

The skill is finished when the answer is written to the output location — without a human in the
loop, once the question is settled by running code or found unanswerable; with a human, once the
user names the variant that answers it. Stop there.
