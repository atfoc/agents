---
name: define-feature
description: Defines a feature — a high level description of what is being built, never how — by building that document with the user through questions. Use when you want to define a feature, nail down what a feature is before any implementation work, or produce a feature definition document.
argument-hint: [feature]
---

# Define a feature

The feature is what the user named: $ARGUMENTS, or what was described earlier in the conversation. If there is no description of what is being built, ask for it first and stop until you have it.

## 1. Build the document by asking questions

The feature definition is a document the user wants to work out by being asked questions instead of having it written for you. Build it that way, with the feature as the document's subject and section 2 as what the questions must cover.

The document is written to the location the user gives. There is no default location — if the user has not said where it goes, ask. Never pick a location yourself.

## 2. What the document must contain

A shared understanding of what is being built: scope, reason, direction and goal.

- Never implementation level: no class names, no signatures, no pseudo code, no file layout.
- Describe behaviour and what the user sees. The right altitude reads like: "The social element of liking and disliking user posts is being added to our platform. Like and dislike buttons with a progress bar representing the ratio of likes to dislikes sit below the post title. Clicking like flashes an animation…"
- This holds even when the feature is technical. A refactor is described by what changes for the project, not by how the code is rearranged.
- Cover how the feature impacts the existing project: what it changes, adds and removes — still functionality, not implementation.
- Surface every conflict or problem the feature creates with what already exists.

## 3. Stop

Do not produce the implementation spec, do not split the feature into tasks, and do not implement anything. That is a separate step, run in its own session from the path just written.

The skill is finished once the feature definition is written and its path reported. Stop there.
