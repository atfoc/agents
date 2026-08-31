# Docs format

Docs are a collection of information, not a set of procedures bound to one tool. An agent given a
task searches them and builds the context that task needs. The consequences are the point:

- Docs are **linkable** — one doc points at another instead of repeating it.
- Docs are **progressively disclosed** — an index is read before its contents, so only what the
  task needs is loaded.
- Docs are **searchable** by plain text when the index does not resolve something.
- Docs are **agent independent** — nothing in them names a tool or a harness.
- Docs are **composable** — a procedure names what it needs, not where every piece of it lives.
  A task procedure never has to link platform specifics; if a task needs them, the agent finds that
  doc and uses it.
- Links run **one way**, from a doc to what it depends on. Nothing records who depends on it, so a
  doc can be reused by anything without being edited.

## Layout

```
<root>/
├── Docs.md            # the index for this directory — required
├── topic-a.md         # a doc
├── topic-b.md
├── sub-topic/         # a nested doc directory
│   ├── Docs.md        # its own index — required
│   └── topic-c.md
└── scripts/           # a helper directory — no Docs.md
    └── helper.py
```

- **`Docs.md` marks a directory as docs.** It is the index: what can be found in that directory
  and where. It routes; it does not carry the content itself.
- **A directory with no `Docs.md` holds no docs.** It is a helper for its parent — scripts,
  templates, assets. Never search it for information and never descend into it looking for docs.
  Use its files only when a doc points at them.
- Docs are plain markdown, one topic per file, and link to each other by relative path.

## Where roots are

- **User docs** — the `AI_DOCS` environment variable: a `:`-separated string of paths. Every path
  in it is a docs root.
- **Project docs** — under the working directory, any folder holding a `Docs.md`. The shallowest
  such folder on a branch is a root; deeper ones are reached through their parent's index.

## How an agent searches

1. **Name what is missing.** Before reading anything, write down the specific questions the task
   cannot be completed without.
2. **Navigate by index.** Read each root's `Docs.md` and open only the entries that answer a
   question. Descend into a subdirectory only through its own `Docs.md`. Follow links between docs.
3. **Fall back to text search.** For whatever the indexes did not answer, grep the roots
   (`grep -ril "<term>" <root>`) and read the hits.
4. **Report what is still missing.** If a question is still open, say which, and ask the user to
   supply it. Never guess a convention that the docs were supposed to provide.

## Writing a doc

- Concise and direct. Say what to do, not why, unless a rule fails without the reason.
- Make rules checkable: "never write a file until the user approves" beats "be careful with files".
- Bullet lists, not tables — they cost fewer tokens.
- One topic per file. When a topic is shared by two docs, give it its own file and link to it from
  both.
- **Link forward, never back.** A doc names what it needs — the next step, the format it depends
  on, the reference it pulls numbers from. It never names who uses it, what it is used by, or which
  step comes before it. A backlink goes stale the moment a second caller appears, and it costs the
  reader context they did not ask for. If you find yourself writing "used by", "applies to",
  "shared by" or a `Related` section, delete it.
- Add its line to the directory's `Docs.md`: the filename, then what it holds and when to open it.
- Push long reference material and scripts into their own files: a script is run, never read into
  context.

## Writing a `Docs.md`

- A title, one line on what the directory covers, then the entries.
- One bullet per doc: name, a dash, what it holds and when to open it. Enough for a routing
  decision, no more.
- A separate list for child directories that have their own `Docs.md`.
- A separate list for helper directories, saying what they are for.
- No content of its own beyond that routing, and no backlinks: an index lists what is inside its
  own directory, never what elsewhere depends on it.
