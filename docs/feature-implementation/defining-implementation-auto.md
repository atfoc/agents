# Defining an implementation without questions

Output: an implementation spec — see `implementation-spec.md` for what it contains and the level of
detail it requires. It is always written to a file, at the location the user gives. There is no
default location, and it is never derived from where the input lives. If the user has not said
where the spec goes, say the location is missing and stop before doing any work. Never pick a
location yourself.

This is not an interactive session. Never ask the user anything about the feature or its
implementation: every question is answered from the input and the codebase.

## Input

Whatever describes what is to be built. A feature definition in a file, a document produced earlier
in the conversation, a prompt, or the conversation itself — any of them is enough, as long as there
is enough there to build a spec from. If there is nothing, say the input is missing and stop. Never
go looking for one, and never invent one.

## Building it

- Walk the whole input and work out how each part is implemented in this codebase. Cover module
  splits, migrations, signatures — what is added, removed or changed.
- Leave no open question. Every question that would have gone to the user is answered here, and
  answered by choosing: ground the choice in the codebase first — existing patterns, conventions,
  neighbouring code — and where the codebase does not settle it, prefer the option that changes the
  least. Assume nothing and leave nothing hanging.
- A conflict — within the input, or between it and the existing project — is resolved the same way.
  Never resolve one silently: say what was resolved, in the spec, where it applies.
- Cover everything in the input. Never narrow its scope to make the spec easier.
- A part that cannot be specified at all — the input contradicts itself with no resolution that
  keeps its intent, or it depends on something that is neither in the project nor decidable — is
  recorded as unresolved. Specify everything else.

## Unresolved

The spec ends with one section:

- **Unresolved** — every part that could not be specified and what is missing to specify it.
  Written as `None` when there are none.

## Stop

- Write the spec to the file as soon as it is complete. Ask for no approval, and never write the
  spec back into the conversation.
- Report the path and each unresolved part, then stop. Do not split the spec into tasks and do not
  implement it — see `split-work-into-vertical-slices.md`.
