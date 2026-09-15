# Defining a feature

Output: a feature definition — a high level description of **what** is being built, never how.
It is always written to a file, at the location the user gives. There is no default location: if
the user has not said where it goes, ask. Never pick a location yourself.

Run it as an interactive session — see `../asking-questions.md` for the loop, the repeat rule, and
the write-approval rule.

## What a feature definition is

Shared understanding of what is being built. Not implementation level: no class names, no
signatures, no pseudo code. Scope, reason, direction and goal.

Example of the right altitude:

> The social element of liking and disliking user posts is being added to our platform. Like and
> dislike buttons with a progress bar representing the ratio of likes to dislikes sit below the
> post title. Clicking like flashes an animation…

This holds even when the feature is technical in nature — a refactor is still described by what
changes for the project, not by how the code is rearranged.

## Building it

- If the user gave no description of what is being built, ask that first.
- Once there is something to work with, run the question loop until the whole feature is covered.
- Besides what is being built, cover how the feature impacts the existing project: what it
  changes, adds or removes. Still functionality, not implementation.
- Surface every conflict or problem the feature creates with what already exists.

## Stop

After the feature definition is written, report the path and stop. Producing the implementation
spec is a separate step — `defining-implementation.md`. Do not split into tasks and do not
implement.

When both documents are wanted in one session, run this one to its own completion first — its own
approval, its own file write — then start `defining-implementation.md` with the path just written.
Never merge the two loops. Stopping after the feature definition is a complete outcome: the file
is on disk and the spec can be built from it in a later session.
