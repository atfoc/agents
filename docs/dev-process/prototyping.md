# Prototyping

Answering a question that words cannot settle by building in the real project. The question is
whether something works or can work (`for-agent`), or how it looks or feels (human).

Input: the question — including what settled looks like — and the location the output is written
to. No question: write that it is missing to the output location and stop.

## Setup

- Create a branch and a worktree, both named for the task. Every attempt is built there and
  nowhere else.

## `for-agent`

The question is whether something works or can work in real code.

- Build in the worktree until the question has an answer.
- Do not ask the user. The body's question and end result are the whole assignment.
- Stop when the answer is yes, no, or that it cannot be settled from what the body gave.

## Human

The question needs a person's judgment — how it looks, how it feels.

- Build one variant. Show it to the user in whatever form lets them judge it — run it, capture it,
  or point at the code.
- Take what the user wants changed, or the next variant they want to see. Build it. Show it.
- Repeat until the user names the variant that answers the question. Whether the question is
  answered is the user's call, never yours.

## Output

Written to the given location:

- the question;
- what was tried, one line each;
- the answer — for `for-agent`, whether it works or can work, and the evidence; for human, the
  chosen variant and what it settled in words, so the answer is usable without reading the code;
- the branch name, as a pointer.

## Rules

- The code is a reference. It is never merged; the answer is the words in the output.
- Every attempt stays reachable in the branch. Nothing is discarded because a later one was
  preferred.
- The branch and worktree are not cleaned up when the task ends. They stay until the whole process
  the task belongs to is finished.
