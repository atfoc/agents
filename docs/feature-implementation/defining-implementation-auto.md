# Defining an implementation without questions

Output: an implementation spec — see `implementation-spec.md` for what it contains and the level of
detail it requires. It is always written to a file, at the location the user gives. There is no
default location, and it is never derived from where the feature definition lives. If the user has
not said where the spec goes, say the location is missing and stop before doing any work. Never
pick a location yourself.

This is not an interactive session. Never ask the user anything about the feature or its
implementation: every question is answered from the feature definition, the codebase and the docs.

## Input

A feature definition: its location as the user gives it, or one produced earlier in the
conversation. If neither was provided, say it is missing and stop. Never go looking for one — not
next to where the spec is written, not anywhere else. Never invent a feature definition and never
start from a bare description.

## Building it

- Walk the whole feature definition and work out how each part is implemented in this codebase.
  Cover module splits, migrations, signatures — what is added, removed or changed.
- Assume nothing. A question that would have gone to the user is answered here: from the
  codebase first — existing patterns, conventions, neighbouring code — then from the docs.
- When a question needs running code to answer — whether something works, how a library or the
  app behaves — settle it by building it, without a human in the loop: see `../prototyping.md`.
  Never settle it by reading code and reasoning about what it would do.
- When neither the codebase, the docs nor a prototype settles a question, choose. Prefer the option
  that follows the codebase's existing patterns and changes the least. Record it as a decision.
- A conflict — within the feature definition, or between it and the existing project — is
  resolved the same way and recorded as a decision. Never resolve one silently.
- Cover everything in the feature definition. Never narrow its scope to make the spec easier.
- A part that cannot be specified at all — the definition contradicts itself with no resolution
  that keeps its intent, or it depends on something that is neither in the project nor decidable —
  is recorded as unresolved. Specify everything else.

## Decisions and unresolved parts

The spec ends with two sections:

- **Decisions** — every choice made without the user: the question, what was chosen, the
  alternatives considered, and the evidence behind the choice — a file path, a doc, a prototype
  result.
- **Unresolved** — every part that could not be specified and what is missing to specify it.
  Written as `None` when there are none.

## Stop

- Write the spec to the file as soon as it is complete. Ask for no approval, and never write the
  spec back into the conversation.
- Report the path, the number of decisions, and each unresolved part, then stop. Do not split the
  spec into tasks and do not implement it — see `split-work-into-vertical-slices.md`.
