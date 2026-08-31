# Explaining code

Build understanding of the code, then explain it at the abstraction level the user asked for. The
level is the whole difference: "how does this method work" and "how does this module work" get
different answers.

If the user did not say what to explain, ask and stop.

## Functions

Low level — dig into implementation details.

- Start with the signature: what it takes, what it returns.
- Per input: what is valid, what leads to failure.
- The return: one sentence if it is straightforward. If the result has several states, cover each
  and what data it carries. E.g. "returns an object in one of: user not found, user banned, user
  active", then each.
- **Algorithms** also get worked examples: input, output, and a step-by-step walk-through where
  complexity warrants. At least one example per case and one per edge case.

## Classes

Focus on the public API. Each public function: what it does, and any rule about how it must be
used. Not inputs and outputs one by one — what the collection of functions achieves together, and
what state it encapsulates. If the class models a state machine, explain that state machine.

## Modules (a collection of classes)

What the collection is for. A sentence or two per class: its job, what it encapsulates. Do not go
into any class's public API.

## Whole codebase

Rare. Two or three modules — repeat the module explanation for each. More than that — name the
modules, one sentence each, and ask which to dive into.

## Usage of a function

Combine how the function works with how it is used. Traverse its call stack upward; each level
unlocks more product-level meaning. Define each use case as close to the product as possible.

Not: "this fetches a locked user from the database and is used whenever a user is mutated."
Instead: "this returns a locked user from the database, used in these cases — when a user buys gem
stones, the state object is fetched locked and the stash is updated by the right amount…"

## The loop

The user zooms in and out: a function leads to where it is used, that leads to a module, that
leads to a class, that leads back to a method. Keep applying these rules to progressively disclose
complexity and lead the user through the codebase.

## Always

- Do not assume familiarity with the codebase. Spend extra sentences on anything non-obvious.
- Introduce and explain anything you reference before you reference it.
- Do not collapse multiple facts into one dense sentence. Split them.
