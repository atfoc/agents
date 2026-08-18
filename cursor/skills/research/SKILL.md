---
name: research
description: Research a question or a goal properly — frame what a complete answer needs, then close the gap in rounds of focused questions, one at a time, until the evidence stops changing the answer. Use whenever the user asks something that a single lookup cannot settle: how something works, why something happens, whether something is viable, which of several options is better, or any open question or investigation.
---

# Research

Answer a question, or reach a goal, from evidence you gathered — not from what you already believe.

The topic is whatever the user gave you: the skill argument, a file they pointed at, or the request in the conversation. If there is no topic, ask for one and stop.

You work in **rounds**. Each round you pose the questions standing between you and the answer, answer them from evidence, and fold what you found into the picture. You keep going until another round would not change the answer.

You do this work yourself. Do not delegate it.

**The answer lives in this conversation.** Do not write report files unless the user asks for one.

## Step 1 — Frame the topic

Write down, in a few lines:

- **The question**, restated sharply in your own words. If the user gave a goal, state the verdict or outcome that would satisfy it.
- **Done means** — what a complete answer must contain: the mechanisms, values, comparisons or verdicts required. Everything after this closes in on that target, and you check yourself against it at the end.
- **What you already know**, and separately **what you are assuming**. Every load-bearing assumption is a question, and it goes into round 1.

Ask the user for clarification only if the framing is ambiguous in a way that would change what you research. Ask once, then proceed.

## Step 2 — Pose the round's questions

List what stands between you and "done": the things whose answers you cannot write down right now.

A question is ready to work when all of these hold:

- **It is one question**, answerable on its own from evidence you can reach.
- **It is decidable.** A concrete answer exists and you would recognise it when you saw it. "Tell me about X" is not a question.
- **You know what its answer must contain** — paths and line numbers, a signature, exact values, command output, a URL.
- **Its answer would change your final answer.** If it would not, drop it.

Write the round's questions down before you answer any of them. Choosing what to ask is a separate act from answering it, and merging the two is how research turns into wandering.

Aim for three to six questions per round. Fewer and the round is not worth its bookkeeping; more and you lose the thread before you finish. Never re-ask something an earlier round already established.

In round 1, when the territory is unfamiliar, spend one question on mapping it — what exists, where it lives, what the pieces are — so the next round's questions are informed instead of guessed.

## Step 3 — Decide what evidence each question needs

Two kinds, and the difference changes what you actually do:

- **The answer already exists** and needs finding, reading and condensing: code, docs, config, history, external material. Keep the question narrow — asked broadly, you get back something vague. What you write down must be specific: exact paths, line numbers, names, short quoted snippets, never "somewhere in the codebase".
- **The answer does not exist yet** and must be produced: run the thing, script a probe, measure it, reproduce the failure, check whether the hypothesis holds. This may take several steps, but it is still one question.

A probe is **not a feature**: the deliverable is the finding, so no production code and no unit tests. Keep every probe inside one scratch directory — `.research-scratch/<slug>/` — and touch nothing else in the project. Delete that directory when the research ends, or tell the user what you left behind and why it is worth keeping.

## Step 4 — Run the round

Work the round's questions, one at a time, and answer each on its own terms. Where lookups are independent of each other, issue them together in a single message rather than one after another.

Two things to hold to:

- **Read for the question, not for the file.** A targeted read that answers what you asked beats loading everything nearby. Your context is the place the whole picture has to fit, and every page you did not need costs you room you will want later.
- **Finish the round before you conclude anything.** It is a barrier. Do not start folding results while questions are still open, and do not skip the rest of the round because the first two answers made the picture look clear.

Record each answer as you get it, in the concrete terms Step 2 asked for. An answer you did not write down is one you will half-remember when it matters.

## Step 5 — Fold what you found into the picture

Go through the round's answers and sort them:

- **Established** — verified, with a reference you could check again yourself.
- **Asserted** — something you wrote down without evidence, because you recalled it or it seemed obvious. It is not a finding. If it matters, go and verify it.
- **Contradicted** — two pieces of evidence disagree. Never silently keep the one you prefer. Make the disagreement itself the next question and take it to primary evidence: the source, the actual output, the running system. If the disagreement is one of judgement rather than fact, settle it in the open — state the real alternatives, weigh them against the evidence you have, commit to one, and give the reason in a line.
- **New leads** — things you were not looking for that change what is worth asking next.

Then restate the picture: what of "done" is now covered, what is still open, and what the round changed about the question itself.

Distrust a thin answer. No paths, no values, no output means the question was too broad, not that the answer is unavailable — sharpen it and ask it again.

## Step 6 — Another round, or stop

Run another round only for questions that still meet the Step 2 bar and still matter to "done".

Stop when any of these is true:

- **Saturation** — new findings only confirm what you have, and nothing open would change the answer.
- **Exhausted** — what remains cannot be established with the tools and access available. Stop and say so in the answer; do not fill the gap with a guess.
- **Spent** — the cost of another round outstrips what it could add. Four rounds is a lot of research; if you are not converging by then, answer with what you have and name what is unresolved.

## Step 7 — Answer

Lead with the answer. First the direct response to the question or the verdict on the goal, then the findings that support it, then what is unsettled:

- **Every claim carries its support** — a file path and line, a command and its real output, a URL. A claim you cannot support is a claim you mark as inference and label as such.
- **State confidence where it is not high**, and say what would raise it.
- **Report the unresolved parts as unresolved**: contradictions you could not break, questions you could not answer, and parts of "done" you did not reach. Never paper over a gap to make the answer look finished.
- **Keep out the process.** How many rounds it took, what you searched and what you tried first are not findings. Answer the question.

Answer at the depth the topic deserves — a settled question gets a paragraph, a real investigation gets its structure. Then stop.
