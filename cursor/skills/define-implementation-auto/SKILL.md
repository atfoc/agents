---
name: define-implementation-auto
description: Turns a description of what is to be built into an implementation spec — the modules that change, the exact contracts between them and the rules they must follow — without asking the user anything, settling every open question by choosing and grounding the choice in the codebase's existing patterns. Use when you want the implementation spec written for you with no questions.
---

# Define an implementation without questions

Act on what the user gave you: the skill argument, or what was named earlier in the conversation.

- **The input** — whatever describes what is to be built. A feature definition in a file, a document produced earlier in the conversation, a prompt, or the conversation itself: any of them is enough, as long as there is enough there to build a spec from. If there is nothing, say the input is missing and stop. Never go looking for one, and never invent one.
- **The spec location** — the file path the user gave. There is no default, and it is never derived from where the input lives. If the user has not said where the spec goes, say the location is missing and stop before doing any work. Never pick a location yourself.

This is not an interactive session. Never ask the user anything about the feature or its implementation.

## 1. Read the input

Take it in whole. Then walk each part of it and work out how that part lands in this codebase: which modules are added or changed, what each one is responsible for, where modules meet, and what behaviour each must have.

## 2. Leave no open question

Every question that would have gone to the user is answered here, and answered by choosing. Ground the choice in the codebase first — existing patterns, conventions, neighbouring code — and where the codebase does not settle it, prefer the option that changes the least. Assume nothing and leave nothing hanging.

A conflict — within the input, or between it and the existing project — is resolved the same way. Never resolve one silently: say what was resolved, in the spec, where it applies.

## 3. Write the spec

Cover everything in the input. Never narrow its scope to make the spec easier.

The spec says what each module does and how modules meet. How a module works inside is left to whoever builds it. It has these parts:

- **Modules.** Every module added or changed: where it lives, what it is responsible for, and what it owns — state, files, external systems. For an existing module, what changes in it.
- **Contracts.** Exact wherever modules meet: public signatures, types and their fields, HTTP endpoints with their paths and payloads, command-line syntax, file and data formats. Each has a location. Anything a contract names — a type, a field, an error — is itself defined in the contracts.
- **Rules.** The behaviour each module must have, written as rules a test can check, not as code: its states and what moves it between them, what happens when two operations overlap, what happens when a dependency fails or answers with something missing, and how outside input is treated before it reaches a process, a file path or a pattern.
- **Exact where it must match.** Anything that has to match something outside the code — an existing script, an external system, a label, a key, text the verification checks — is spelled out exactly.
- **Verification.** How the whole change is verified end to end.

Never write pseudo code or a full rewrite for what happens inside a module.

Anything read from a live system that the spec relies on — test data, an expected value, something to compare against — is saved verbatim to a file next to the spec when it is read. The spec points at that file, never at the live system.

A part that cannot be specified at all — the input contradicts itself with no resolution that keeps its intent, or it depends on something that is neither in the project nor decidable — is recorded as unresolved. Specify everything else.

End the spec with two sections:

- **Prerequisites** — everything the implementation or its verification needs from outside the repository: tools and their versions, network access, permissions, accounts. Each has a read-only probe command and what it printed when you ran it while writing the spec. Write `None` when there are none.
- **Unresolved** — every part that could not be specified and what is missing to specify it. Write `None` when there are none.

## 4. Stop

Write the spec to the file as soon as it is complete. Ask for no approval, and never write the spec back into the conversation.

Report the path, each prerequisite whose probe failed, and each unresolved part. Do not split the spec into tasks and do not implement it. The skill is finished there and stops.
