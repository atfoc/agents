---
name: research
description: Research a question or a goal properly — frame what a complete answer needs, then close the gap in rounds of parallel scout and worker subagents, one focused question each, until the evidence stops changing the answer. Use whenever the user asks something that a single lookup cannot settle: how something works, why something happens, whether something is viable, which of several options is better, or any open question or investigation.
---

# Research

Answer a question, or reach a goal, from evidence you gathered — not from what you already believe.

The topic is whatever the user gave you: the skill argument, a file they pointed at, or the request in the conversation. If there is no topic, ask for one and stop.

You work in **rounds**. Each round you pose the questions standing between you and the answer, send one subagent per question in parallel, and fold what comes back into the picture. You keep going until another round would not change the answer.

**The answer lives in this conversation.** Do not write report files unless the user asks for one.

If you are already running as a subagent you cannot fan out — do the rounds yourself, sequentially, with the same discipline.

## Step 1 — Frame the topic

Write down, in a few lines:

- **The question**, restated sharply in your own words. If the user gave a goal, state the verdict or outcome that would satisfy it.
- **Done means** — what a complete answer must contain: the mechanisms, values, comparisons or verdicts required. Everything after this closes in on that target, and you check yourself against it at the end.
- **What you already know**, and separately **what you are assuming**. Every load-bearing assumption is a question, and it goes into round 1.

Ask the user for clarification only if the framing is ambiguous in a way that would change what you research. Ask once, then proceed.

## Step 2 — Pose the round's questions

List what stands between you and "done": the things whose answers you cannot write down right now.

A question is ready to dispatch when all of these hold:

- **It is one question**, answerable on its own by one agent without asking anyone anything.
- **It is decidable.** A concrete answer exists and you would recognise it when you saw it. "Tell me about X" is not a question.
- **You know what its answer must contain** — paths and line numbers, a signature, exact values, command output, a URL.
- **Its answer would change your final answer.** If it would not, drop it.

Ask a subagent to find out something, never to decide something. Deciding is your job.

Aim for three to six questions per round. Fewer wastes the parallelism; more buries you in summaries you cannot digest. Never send two subagents the same question, and never re-ask something an earlier round already established.

In round 1, when the territory is unfamiliar, spend one question on mapping it — what exists, where it lives, what the pieces are — so the next round's questions are informed instead of guessed.

## Step 3 — Route each question to the custom type `scout` or `worker`

Every question goes to a subagent of one of these custom types. Route on what kind of evidence the answer needs:

- **custom type `scout`** — the answer already exists and needs finding, reading and condensing: code, docs, config, history, external material. Read-only. Keep its question narrow; a scout asked something broad returns something vague.
- **custom type `worker`** — the answer does not exist yet and must be produced: run the thing, script a probe, measure it, reproduce the failure, check whether the hypothesis holds. A worker's job may take several steps, but it still answers one question.

Spawn the custom type by name — never a generic or built-in agent type — and never pass a model or effort override on the spawn call. Each type pins its own model and effort; an override on the call outranks them and silently replaces the agent this skill is built around.

A worker prompt must say that **this is a probe, not a feature**: the deliverable is the finding, no production code and no unit tests. Give it one scratch directory it owns — `.research-scratch/<slug>/` — and tell it to touch nothing else in the project. Delete the scratch directory when the research ends, or tell the user what you left behind and why it is worth keeping.

## Step 4 — Run the round

Launch **every subagent of the round in a single message**, each spawned as the custom type Step 3 routed it to, so the whole round runs in parallel. Each prompt must stand alone — a subagent sees none of this conversation — and contains:

- **The topic in one line**, so it understands what its answer feeds.
- **The one question it owns**, worded as narrowly as Step 2 made it.
- **What its answer must contain**, in the concrete terms you already decided.
- **Where to look or what to try**, when you know something it would otherwise spend its budget rediscovering.
- **What is out of scope**, including the questions the round's other subagents own.

Then wait for the whole round. It is a barrier: never start folding results while some are still running, and never launch the next round early.

## Step 5 — Fold the reports into the picture

Read every report and sort what it gives you:

- **Established** — verified, with a reference you could check yourself.
- **Asserted** — stated without evidence. It is not a finding. If it matters, re-ask it.
- **Contradicted** — two reports disagree. Never silently pick the one you prefer. Send a tie-breaker whose question is the disagreement itself, aimed at primary evidence. If the disagreement is one of judgement rather than fact, a subagent of the custom type `thinker` can settle it: give it the evidence and the question. It never writes the answer.
- **New leads** — things nobody asked about that change what is worth asking next.

Then restate the picture: what of "done" is now covered, what is still open, and what the reports changed about the question itself.

Distrust a thin report. A summary with no paths, no values and no output is a sign the question was too broad, not that the answer is unavailable — sharpen it and send it again.

## Step 6 — Another round, or stop

Run another round only for questions that still meet the Step 2 bar and still matter to "done".

Stop when any of these is true:

- **Saturation** — new findings only confirm what you have, and nothing open would change the answer.
- **Exhausted** — what remains cannot be established with the tools and access available. Stop and say so in the answer; do not fill the gap with a guess.
- **Spent** — the cost of another round outstrips what it could add. Four rounds is a lot of research; if you are not converging by then, answer with what you have and name what is unresolved.

## Step 7 — Answer

Write the answer yourself, in the main agent. **Do not delegate this step** — every report is already in your context, and that is what the answer is made of.

Lead with the answer. First the direct response to the question or the verdict on the goal, then the findings that support it, then what is unsettled:

- **Every claim carries its support** — a file path and line, a command and its real output, a URL. A claim you cannot support is a claim you mark as inference and label as such.
- **State confidence where it is not high**, and say what would raise it.
- **Report the unresolved parts as unresolved**: contradictions you could not break, questions you could not answer, and parts of "done" you did not reach. Never paper over a gap to make the answer look finished.
- **Keep out the process.** Which subagent found what, how many rounds it took, and what you searched are not findings. Answer the question.

Answer at the depth the topic deserves — a settled question gets a paragraph, a real investigation gets its structure. Then stop.
