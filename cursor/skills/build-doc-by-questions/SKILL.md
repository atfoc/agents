---
name: build-doc-by-questions
description: Builds a document together with the user by asking one question at a time, each with a recommended answer, and writes it to file only once the user approves. Use when you want to work out a spec, plan, or other document by being asked questions instead of having it written for you.
---

# Build a document by asking questions

The subject is the document the user named in the skill argument, or what was described earlier in the conversation. If there is no subject, ask what document to build and stop.

## 1. Ask

- Ask one question at a time. Attach one recommended answer to every question.
- Pick a topic, drill down until it is covered, then change topic. Never jump between topics.
- Build on what was already decided. Every question assumes the answers before it.
- Own the coverage. Treat the subject as a graph and walk every path; the user is not responsible for remembering what has not been asked.
- Raise conflicts and problems as soon as you see them.
- You may recommend stopping. Whether to stop is the user's call.

## 2. Repeat when unanswered

When an answer does not cover the question, repeat the question verbatim. Reword it only when the user asks for a clarification or a rewording.

## 3. End

1. Do not write the finished document into the conversation. Ask whether that is all and whether to write it to file.
2. Write nothing to disk until the user approves the write. Do not edit a file continuously while asking; hold the document in the conversation until then.
3. After the write, report the path.

## Rules

- Do not use an ask-question tool. Ask in plain conversation.
- Add no questions, rules or opinions this procedure does not call for.

The skill is finished once the document is written and its path reported. Stop there.
