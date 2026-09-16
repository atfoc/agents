---
name: define-implementation-auto
description: Turns a description of what is to be built into a detailed implementation spec without asking the user anything, settling every open question by choosing and grounding the choice in the codebase's existing patterns. Use when you want the implementation spec written for you with no questions.
---

# Define an implementation without questions

Act on what the user gave you: the skill argument, or what was named earlier in the conversation.

- **The input** — whatever describes what is to be built. A feature definition in a file, a document produced earlier in the conversation, a prompt, or the conversation itself: any of them is enough, as long as there is enough there to build a spec from. If there is nothing, say the input is missing and stop. Never go looking for one, and never invent one.
- **The spec location** — the file path the user gave. There is no default, and it is never derived from where the input lives. If the user has not said where the spec goes, say the location is missing and stop before doing any work. Never pick a location yourself.

This is not an interactive session. Never ask the user anything about the feature or its implementation.

## 1. Read the input

Take it in whole. Then walk each part of it and work out how that part is implemented in this codebase: module splits, migrations, signatures — what is added, removed or changed.

## 2. Leave no open question

Every question that would have gone to the user is answered here, and answered by choosing. Ground the choice in the codebase first — existing patterns, conventions, neighbouring code — and where the codebase does not settle it, prefer the option that changes the least. Assume nothing and leave nothing hanging.

A conflict — within the input, or between it and the existing project — is resolved the same way. Never resolve one silently: say what was resolved, in the spec, where it applies.

## 3. Write the spec

Cover everything in the input. Never narrow its scope to make the spec easier.

Write to the level of detail an implementation spec requires:

- Signatures of functions; class definitions with fields and methods; interfaces and who implements them.
- At least pseudo code for new functions and for additions to existing ones. When a change is a mix of deletions and additions, write the full rewrite in pseudo code, then explain how to get there from the existing code.
- Anything named — a method, class, interface — is itself defined to the same level. A name from pseudo code never stands in for a definition.
- A location for every item: where new code is placed, where existing code being changed lives.
- Test cases for every unit where they make sense, and how the change is verified.

A part that cannot be specified at all — the input contradicts itself with no resolution that keeps its intent, or it depends on something that is neither in the project nor decidable — is recorded as unresolved. Specify everything else.

End the spec with one section:

- **Unresolved** — every part that could not be specified and what is missing to specify it. Write `None` when there are none.

## 4. Stop

Write the spec to the file as soon as it is complete. Ask for no approval, and never write the spec back into the conversation.

Report the path and each unresolved part. Do not split the spec into tasks and do not implement it. The skill is finished there and stops.
