# Researching

Answering a question, or reaching a goal, from evidence you gathered — not from what you already
believe. Use when a single lookup cannot settle it: how something works, why something happens,
whether something is viable, which option is better.

Work in **rounds**. Each round: pose the questions standing between you and the answer, answer them
from evidence, fold what you found into the picture. Stop when another round would not change the
answer.

Do the work yourself. The answer lives in the conversation — write no report file unless asked.

## 1. Frame the topic

- **The question**, restated sharply in your own words. For a goal, state the verdict or outcome
  that would satisfy it.
- **Done means** — what a complete answer must contain: the mechanisms, values, comparisons or
  verdicts required. You check yourself against this at the end.
- **What you already know**, and separately **what you are assuming**. Every load-bearing
  assumption is a question, and it goes into round 1.

Ask the user for clarification only if the framing is ambiguous in a way that changes what you
research. Ask once, then proceed.

## 2. Pose the round's questions

A question is ready to work when all hold:

- **It is one question**, answerable on its own from evidence you can reach.
- **It is decidable** — a concrete answer exists and you would recognise it. "Tell me about X" is
  not a question.
- **You know what its answer must contain** — paths and line numbers, a signature, exact values,
  command output, a URL.
- **Its answer would change your final answer.** If not, drop it.

Write the round's questions down before answering any of them. Choosing what to ask is a separate
act from answering it; merging the two is how research turns into wandering.

Three to six questions per round. Never re-ask what an earlier round established. In round 1 on
unfamiliar territory, spend one question mapping it — what exists, where it lives, what the pieces
are.

## 3. Decide what evidence each question needs

- **The answer already exists** and needs finding, reading and condensing: code, docs, config,
  history, external material. Keep the question narrow — asked broadly, you get back something
  vague. Write down exact paths, line numbers, names, short quoted snippets. Never "somewhere in
  the codebase".
- **The answer does not exist yet** and must be produced: run it, script a probe, measure,
  reproduce the failure, test the hypothesis. Several steps, still one question.

A probe is **not a feature**: the deliverable is the finding, so no production code and no unit
tests. Keep every probe inside `.research-scratch/<slug>/` and touch nothing else. Delete it when
the research ends, or say what you left behind and why.

## 4. Run the round

Work the questions one at a time; issue independent lookups together in a single message.

- **Read for the question, not for the file.** A targeted read beats loading everything nearby.
- **Finish the round before concluding anything.** It is a barrier. Do not fold results while
  questions are open, and do not skip the rest because the first two answers looked clear.

Record each answer as you get it, in concrete terms.

## 5. Fold it in

Sort the round's answers:

- **Established** — verified, with a reference you could check again.
- **Asserted** — written down without evidence because you recalled it or it seemed obvious. Not a
  finding. If it matters, verify it.
- **Contradicted** — two pieces of evidence disagree. Never silently keep the one you prefer. Make
  the disagreement the next question and take it to primary evidence. If it is a disagreement of
  judgement rather than fact, state the real alternatives, weigh them, commit to one, give the
  reason in a line.
- **New leads** — things that change what is worth asking next.

Then restate the picture: what of "done" is covered, what is open, what the round changed about the
question itself. Distrust a thin answer — no paths, no values, no output means the question was too
broad, not that the answer is unavailable.

## 6. Another round, or stop

Stop when any holds:

- **Saturation** — new findings only confirm what you have.
- **Exhausted** — what remains cannot be established with the tools and access available. Say so;
  do not fill the gap with a guess.
- **Spent** — another round costs more than it could add. Four rounds is a lot; if you are not
  converging by then, answer with what you have and name what is unresolved.

## 7. Answer

Lead with the answer, then the findings that support it, then what is unsettled.

- **Every claim carries its support** — a file path and line, a command and its real output, a URL.
  A claim you cannot support is marked as inference.
- **State confidence where it is not high**, and what would raise it.
- **Report the unresolved parts as unresolved.** Never paper over a gap.
- **Keep out the process.** How many rounds it took and what you tried first are not findings.

Answer at the depth the topic deserves, then stop.
