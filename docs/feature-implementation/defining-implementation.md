# Defining an implementation

Output: an implementation spec — a fully detailed set of instructions for what changes, where, and
how it is verified. It is always written to a file, at the location the user gives. There is no
default location, and it is never derived from where the feature definition lives: if the user has
not said where the spec goes, ask. Never pick a location yourself.

Run it as an interactive session — see `../asking-questions.md`.

## Input

A feature definition: its location as the user gives it, or one produced earlier in the
conversation. If neither was provided, ask the user whether a feature definition exists and where it
is. Never go looking for one — not next to where the spec is written, not anywhere else. Never
invent a feature definition and never start from a bare description.

## Required level of detail

- Signatures of functions; class definitions with fields and methods; interfaces and who
  implements them.
- At least pseudo code for new functions and for additions to existing ones. When a change is a
  mix of deletions and additions, write the full rewrite in pseudo code, then explain how to get
  there from the existing code.
- Anything named — a method, class, interface — is itself defined to the same level. A name from
  pseudo code never stands in for a definition.
- A location for every item: where new code is placed, where existing code being changed lives.
- Test cases for every unit where they make sense, and how the change is verified.

## Building it

Walk the whole feature definition and discuss how each part is implemented in this codebase. Cover
module splits, migrations, signatures — what is added, removed or changed. Assume nothing; the
point is shared understanding of the whole implementation.

## Stop

After the implementation spec is written, report the path and stop. Do not split the spec into
tasks and do not implement it — see `split-work-into-vertical-slices.md`.
