# Prototyping

Answering a question of how something works or looks that words cannot settle: grow several
variants inside the real project and loop with the user until one of them answers it.

Input: the question, and the location the output is written to. No question: ask for it and stop.

## Setup

- Create a branch and a worktree, both named for the task. Every variant is built there and
  nowhere else.

## The loop

- Build one variant. Show it to the user in whatever form lets them judge it — run it, capture it,
  or point at the code.
- Take what the user wants changed, or the next variant they want to see. Build it. Show it.
- Repeat until the user names the variant that answers the question. Whether the question is
  answered is the user's call, never yours.
- Every variant stays reachable in the branch. Nothing is discarded because a later one was
  preferred.

## Output

Written to the given location:

- the question;
- the variants tried, one line each;
- the chosen variant;
- what it settled, in words, so the answer is usable without reading the code;
- the branch name, as a pointer.

## Rules

- The code is a reference. It is never merged; the answer is the words in the output.
- The branch and worktree are not cleaned up when the task ends. They stay until the whole process
  the task belongs to is finished.
