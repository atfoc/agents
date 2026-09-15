# Implementation spec

A fully detailed set of instructions for what changes, where, and how it is verified. Built from a
feature definition.

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
