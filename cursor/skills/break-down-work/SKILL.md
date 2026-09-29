---
name: break-down-work
description: Cuts one large chunk of work — an implementation spec, or any written body of work — into vertical slices, each with its own title, self-contained body, verification, blockers, tags and a call on whether it is `for-agent`, with a contracts slice first and a review slice paired with every implementation slice. Use when a spec or a big piece of work has to be broken down into individual tasks that can be picked up one at a time.
---

# Break work down into vertical slices

Split the work the user gave you: the skill argument, or the spec or chunk of work named earlier in the conversation. If no work is named, ask what to split and stop.

## What a slice is

A **vertical slice** is a collection of work that builds one functional and testable unit of the whole body of work. Every slice has:

- **Its own verification.** A slice is done when its verification passes. Its verification is automated — tests, type checks, builds — and never needs a human. The only verification-only slices are review slices and the ones described in Step 3a.
- **A body that is the whole assignment.** Whoever implements a slice sees that body and nothing else — not the implementation spec, not the feature definition, not any other slice, not a reason why it was cut this way.
- **A title**, one line.
- **Blockers** — the slices that must be completed before it may start.
- **`for-agent` or not.** A slice is `for-agent` when an agent can do all of it, verification included, with no human in the loop. Every other slice is not `for-agent`.
- **A kind tag**, where it has one: `contracts`, `implement`, `review` or `prerequisite`. Verification and environment setup slices carry none.

## Step 1 — Read the whole source

Read the whole of the work being split before cutting anything. Do not start cutting from a partial read.

Establish where the slices go. The user names the destination — a task store, a folder, a file. Never default or invent one. If the destination is not named, ask for it and stop.

## Step 2 — Cut

- **Contracts first.** When the source defines contracts where modules meet, cut one **contracts slice** that runs before any implementation slice. It turns every contract into code and wires everything in, so implementation slices build against real types and never edit shared wiring. When the source defines no contracts, there is no contracts slice.
- **Every implementation slice owns its ground.** Give it the modules and files it owns. It changes only those, plus its tests. Two slices that own the same module or file are ordered in Step 4.
- **Every implementation slice gets a review slice**, written in Step 3.
- Work that cannot be verified on its own folds into the slice whose verification exercises it.
- **Never re-cut a slice to gain parallelism.** Keeping the change → verify → fix loop inside one implementer's context beats handing pieces around. Parallelism is only what falls out of slices that were already independent.
- Where a slice depends on work that is not cut yet, stub that branch rather than widening the slice to cover it. The body names the stub, so the slice that later replaces it knows what it is replacing.

## Step 3 — Write each body

- Carry text from the source **verbatim**. No summarising, no rewording, no "see the source".
- Carry the reason for the work and its user-visible behaviour verbatim wherever a slice needs them.
- Carry the source's conventions — style, patterns, project guides it names — into every implementation and review body.
- A saved copy of live data the source relies on is carried verbatim into each body that needs it. When it is too large to carry, the contracts slice commits it to the repository and the bodies name the committed path.
- You may add your own detail, and added detail only narrows or sequences what the source already decided.
- Never add anything that contradicts, weakens or reinterprets the source. If a workable slice would require that, stop, write nothing, and report what conflicts with what.
- Never name the source's path in a body. The body has to stand alone.
- **Duplication between slices is expected and correct.** A section that applies to four slices is copied into all four; a slice is never made smaller by pointing at another slice's body.

**The contracts slice** is tagged `contracts` and is `for-agent`. Its body carries every contract of the source verbatim and has the agent write each one as code — types, signatures, fields, endpoints, command-line entry points, formats — with bodies that fail plainly as not implemented. It wires every new piece in — construction, registration, routes — and writes the fakes consumers need in their tests. Its verification is that the project builds and every existing test still passes.

**An implementation slice** is tagged `implement`, and is `for-agent` when an agent can do all of it. Its body carries:

- What it delivers, with the rules it must follow, from the source.
- The modules and files it owns. It changes nothing else except tests.
- The contracts it implements and the ones it uses, verbatim, and that they are read-only: their signatures, fields, paths, payloads and formats stay as they are. The code behind a contract it implements is its own. If the work cannot be done without changing a contract, the agent stops and reports the change it needs and why.
- That the agent writes tests for every rule it carries, runs them with the full build, and fixes what fails until everything passes. That is its verification.

**A review slice** is tagged `review` and is `for-agent`. There is one for each implementation slice. Its body carries the same deliverable, rules and contracts as that implementation slice, verbatim, and has the agent:

1. Before reading the implementation, write test cases from the rules and contracts: normal use, two operations overlapping, a dependency failing or answering with something missing, hostile input, and limits — each wherever the rule makes it meaningful.
2. Then read the implementation and its tests, and add cases for what they leave unreached: code paths no test runs, overlaps that come from how it was built, outside input that reaches a process, a file path or a pattern.
3. Write the cases as tests next to the existing ones, and run them with the full build.
4. For every rule, break the implementation on purpose, confirm at least one test fails, then undo the break. A rule no test catches is not covered yet.
5. Leave no production code changed, and never weaken or delete the implementation's own tests.
6. Judge only against the carried text. Something it asks for that is missing is a failure. Something the text itself seems to miss is a finding to report, not work to add.

A review passes when every test passes and every rule is caught by a test. It reports either a pass, with the tests it added and the rule each covers, or a failure, with each failing test, what it expects and what it saw. Failing tests stay committed.

## Step 3a — Move manual verification out of the slices

Never put manual verification — clicking through the app, eyeballing output, calling an endpoint by hand — in a slice body. Wherever the source asks for it, or a slice would need it, cut it out and handle it one of two ways.

**When an agent can simulate it**, create a separate **simulated verification slice**, blocked by the review slice of each slice it verifies. It is `for-agent`. Its body is written so an agent does it end to end with no human:

- What behaviour to confirm, and what the pass, fail and inconclusive results look like.
- That the agent writes a throwaway script that simulates the user: driving the UI (for example with a browser automation tool) for interactive behaviour, or sending HTTP requests or calling the CLI for functional-only behaviour. The script is not committed. The script and its full output are kept outside the repository, and the report gives their path.
- Every piece of environment setup and state the check needs — services to run, config, seed data, accounts, feature flags — and how to reach the environment it runs against, which Step 3b decides.
- **Presence before absence.** A check that something was deleted, cleaned up or rolled back first confirms, in the same run, that it existed. If it never existed, the run is inconclusive: repeat it so that it does. An inconclusive run is never reported as a pass.
- **A person's pace.** The script acts at the pace a person would. A check that passes only after slowing input, adding waits or retrying has failed, and the report gives both runs.
- **Report what you see.** Any wrong behaviour seen along the way is reported as a finding, even outside what the check covers. It is put down to the script only after it has been reproduced by hand, without the script.
- **Exact names.** Names are matched exactly — a label selector, `grep -w` — never as a substring: `grep app-1` also matches `app-10`.
- That the slice is done when the script passes, and that failures are reported with what the script saw.

**When no agent can run it**, even as a one-off — it needs physical hardware, a real third-party account only a human holds, a human judgement call — it goes into one **manual verification slice** at the end of the work. There is exactly one of these, and only if something needs it. It collects every such check across all slices, each with what to confirm, the setup it needs, and the pass result. It is **not** `for-agent`. It is blocked by the review slice of every slice whose work it verifies and blocks nothing, so no other slice ever waits on a human.

The slice the check was cut from keeps its automated verification and is not blocked by either of these.

## Step 3b — Give every simulated verification its own environment

Simulated verification slices run against a live environment, and by default they all run against the *same* one — one server, one port, one database, one config, one data directory, one set of seed data. Two of them running at the same time set that environment up, mutate it and tear it down underneath each other, and neither result can be trusted. So no simulated verification slice ever shares an environment with another.

First look at how the work's environment is actually started — the server entry point, the database name or connection string, the ports, the config and migration tooling — and decide which of the two below applies.

**When the environment can be stood up more than once**, create an **environment setup slice** for each simulated verification slice. It stands up an environment used by nothing else — its own database or schema, its own server on its own port, its own config, its own data directory, its own seed data — and tears it down when the check is finished. It is `for-agent`. Its body names exactly what it stands up, how the check addresses it (ports, connection strings, environment variables, file paths), and how it is torn down. Its verification is that the environment comes up, can be reached, and can be torn down.

The simulated verification slice is then blocked by its own setup slice and by no other setup slice, and its body says it runs against that environment and no other. With one environment per check, every setup slice can start at once and every simulated verification slice follows in parallel.

Do this even when there is only one simulated verification slice. A check in its own environment is cleaner, and state left behind by a simulated user is easier to throw away with the environment than to clean out of a shared one.

**When the environment cannot be stood up more than once** — a hardcoded port or database name, shared global state, migration or seeding tooling that only ever targets one place — and making it parameterisable is outside the work being split, then write no setup slices. Instead chain the simulated verification slices: each one is blocked by the previous, so exactly one ever runs. Each body says the environment is shared, and that the check leaves it in the state it found it.

## Step 3c — Turn missing prerequisites into slices

Collect every prerequisite: the source's Prerequisites section, and anything else a slice needs from outside the repository — tools and their versions, network access, permissions, accounts, live data. Run the read-only probe of each one now, yourself, with the same command tool the agents will use.

Each prerequisite whose probe fails, or that cannot be probed without changing something, becomes a **prerequisite slice**. It is tagged `prerequisite` and is not `for-agent`. Its body says what is missing, what the probe printed, and what would fix it. Its verification is the probe passing in a new shell, not only in the terminal where the fix was made. It blocks only the slices that need it.

Installing or upgrading a tool, changing PATH or shell config, and granting access or permissions are always prerequisite slices. Never ask an agent to do them in a `for-agent` body.

## Step 4 — Decide blockers

A slice is blocked by another for exactly these reasons:

1. **Contracts.** An implementation slice is blocked by the contracts slice.
2. **Real behaviour.** A slice that needs what an implementation slice builds, beyond its contract, is blocked by that slice's review slice — never by the implementation slice itself — so it only builds on reviewed work.
3. **Same ground.** Two implementation slices that own the same module or file cannot run at the same time: the later one is blocked by the earlier one's review slice.
4. **Review.** A review slice is blocked by the implementation slice it reviews.
5. **Verification.** A simulated or manual verification slice is blocked by the review slice of every slice it verifies.
6. **Environment.** A simulated verification slice is blocked by its environment setup slice from Step 3b — or, where environments could not be duplicated, by the simulated verification slice chained before it.
7. **Prerequisites.** A slice is blocked by every prerequisite slice it needs.

Nothing else is a blocker. Nothing is ever blocked by the manual verification slice. A slice is startable when every slice blocking it is complete.

Apply all seven reasons to every pair of slices. Where two implementation slices own the same ground and the order is arbitrary, pick one and say nothing more about it.

A cycle means a bad cut: merge the slices or cut them differently. A cycle that cannot be resolved is a stop-and-report.

## Step 5 — Check before writing anything

All of these, before a single slice is written down:

- Every section of the source appears in at least one body — the deduplicated union of all bodies is the whole source. An orphaned section is a stop, not a silent drop.
- No cycles in the graph of blockers.
- At least one slice with no blockers.
- Nothing contradicted.
- No slice body asks a human to verify anything, except the one manual verification slice and the prerequisite slices.
- When the source defines contracts, there is exactly one contracts slice, and every implementation slice is blocked by it.
- Every implementation slice has exactly one review slice, blocked by it and carrying the same deliverable, rules and contracts verbatim.
- No two implementation slices that own the same module or file can run at the same time.
- Every simulated verification body gives pass, fail and inconclusive results, and every check in it that something is gone first confirms that it existed.
- Every simulated verification slice either has its own environment setup slice blocking it, or is chained behind another simulated verification slice. No two of them can run at the same time against the same environment.
- No environment setup slice is shared by two simulated verification slices.
- Every failing probe has a prerequisite slice, and no `for-agent` body asks an agent to install a tool, change PATH or shell config, or grant access.
- Every slice is marked `for-agent` or not and carries its kind tag. The manual verification slice and the prerequisite slices are not `for-agent`.

Any failure → write nothing and report why.

## Step 6 — Write the slices

Write every slice to the destination from Step 1, each with its title, its body, its blockers, its tags, and whether it is `for-agent`.

## Step 7 — Report

How many slices of each kind and which are not `for-agent` — the prerequisite slices first, each with what its probe printed — every piece of detail you added beyond the source, and anything you had to stop on. Never report the bodies back — they are the slices.

The skill is finished once that report is given, and it stops there.
