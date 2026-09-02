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
├── DOCS.md            # the index for this directory — required
├── topic-a.md         # a doc
├── topic-b.md
├── sub-topic/         # a nested doc directory
│   ├── DOCS.md        # its own index — required
│   └── topic-c.md
└── scripts/           # a helper directory — no DOCS.md
    └── helper.py
```

- **`DOCS.md` marks a directory as docs.** It is the index: what can be found in that directory
  and where. It routes; it does not carry the content itself.
- **A directory with no `DOCS.md` holds no docs.** It is a helper for its parent — scripts,
  templates, assets. Never search it for information and never descend into it looking for docs.
  Use its files only when a doc points at them.
- Docs are plain markdown, one topic per file, and link to each other by relative path.

## Where roots are

- **User docs** — the `AI_DOCS` environment variable: a `:`-separated string of paths. Every path
  in it is a docs root.
- **Project docs** — under the working directory, any folder holding a `DOCS.md`. The shallowest
  such folder on a branch is a root; deeper ones are reached through their parent's index.

## How an agent searches

1. **Name what is missing.** Before reading anything, write down the specific questions the task
   cannot be completed without.
2. **Navigate by index.** Read each root's `DOCS.md` and open only the entries that answer a
   question. Descend into a subdirectory only through its own `DOCS.md`. Follow links between docs.
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
- Add its entry to the directory's `DOCS.md`.
- Push long reference material and scripts into their own files: a script is run, never read into
  context.

## Writing a `DOCS.md`

- A title, one line on what the directory covers, then the entries.
- A list of the docs in this directory, a list of the child directories that have their own
  `DOCS.md`, and a list of the helper directories — each list separate.
- No content of its own beyond that routing, and no backlinks: an index lists what is inside its
  own directory, never what elsewhere depends on it.

### Entry format

One bullet per doc or directory: the name, a dash, a **description**, then an optional **trigger**.

```
- `<name>` — <description>. <trigger>
```

**Description — required.** What the doc or directory *is*, written in the words someone looking
for it would use. Two jobs: it decides the routing when an agent reads the index, and it is what a
plain-text search matches when the index was not read. Name the subject, not the shape — "the local
folder task format and every operation with its command" beats "documentation about tasks".

**Trigger — optional.** One sentence naming the situation that should send an agent to this entry,
phrased as `Use when <situation>`:

```
- `explaining-code.md` — explaining code at the abstraction level the user asked for. Use when the
  user asks a question about the codebase.
```

Write a trigger when the description alone would not make the routing decision obvious — most
often for a doc that answers a request the user phrases in their own words rather than in the
doc's terms. Leave it off when the description already settles it. Triggers are added as gaps show
up: when an agent misses a doc it should have opened, that is the signal to give that entry a
trigger, or to sharpen the one it has.

Keep each entry to what a routing decision needs. The content lives in the doc.
