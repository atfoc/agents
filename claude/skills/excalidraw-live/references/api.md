# `excalidraw-live` — API reference

Everything a session script can call: the agent-side library (`lib/index.ts`), the scene formatter (`lib/format.ts`), the mark store (`lib/marks.ts`), the RPC methods the drawing engine answers, the error codes, the `summary()` line format and the two clipboard shortcuts.

`${CLAUDE_SKILL_DIR}` stands for this skill's folder; write it out as a literal absolute path.

---

## 1. Common types

```ts
type Mode = "headless" | "shared";
type Ref = { id: string } | { key: string };
type El = any;                                   // a full Excalidraw element
type Target = Ref | El | { label: string };      // anything a call can name an element by

type Handle = {                                  // what a create call hands back
  key?: string; id?: string; type: string;
  x: number; y: number; width: number; height: number;
};

type ApplyResult = {
  created: Record<string, string>;               // key → id; key-less items use "#<index>"
  updated: string[];                             // ids
  deleted: string[];                             // ids
  conflicts: { ref: Ref; expectedVersion: number; actualVersion: number }[];
  externalChange?: boolean;                      // another program changed the file — tell the user
  rev: number;
};

type ElementSummary = {                          // the compact form getSelection returns
  id: string; key?: string; type: string; label?: string;
  bounds: { x: number; y: number; width: number; height: number };
  start?: Ref; end?: Ref;                        // arrows: what they are bound to
  groupIds: string[]; frameId?: string | null; version: number;
};

type Mark = {                                    // what ⌘K / ⌘⇧K left with the server
  id: string;                                    // "xlm_" + 8 hex digits
  kind: "selection" | "point";
  at: string;                                    // ISO 8601, when the user pressed the key
  file: string;                                  // the drawing's name, as the block header prints it
  rev: number;                                   // the revision the mark describes
  text: string;                                  // the block of §6, ready to read
  viewport: FmtViewport;
  elements?: FmtElement[];                       // selection: at most 40, bound label text excluded
  total?: number;                                // selection: how many were selected
  point?: [number, number];                      // point: scene coordinates
  near?: FmtElement[];                           // point: at most 3, nearest first
};

type MarkSummary = {                             // one line of getMarks
  id: string; kind: "selection" | "point"; at: string; file: string; rev: number;
  count?: number;                                // selection
  point?: [number, number];                      // point
};
```

A `Target` is resolved in this order: an element with `customData.key` → `{ key }`; anything with an `id` → `{ id }`; a bare `{ key }` → itself; `{ label }` → the first element with that label. An element created earlier in the same uncommitted batch can be named by its `key`.

---

## 2. The library — `lib/index.ts`

```ts
import { connect } from "${CLAUDE_SKILL_DIR}/lib/index.ts";
```

Erasable TypeScript, no dependencies: a script runs with plain `node <script>.ts` on Node 22.18+.

### 2.1 Module exports

| Export | What it is |
|---|---|
| `connect` | opens a session and returns a loaded `Drawing` |
| `session` | reads `session.json` |
| `post` | one authenticated POST to the session server |
| `call` | one RPC call |
| `Drawing` | the class every script works through |
| `RpcFailure` | the error thrown when a call answers `ok: false` |
| types | `Mode`, `Ref`, `El`, `Target`, `Handle`, `ShapeOptions`, `FrameOptions`, `ApplyResult`, and `Mark`, `MarkInput`, `MarkKind`, `MarkSummary` re-exported from `lib/marks.ts` |

#### `connect(dir?)`

`Promise<Drawing>`. `dir` defaults to `$EXCALIDRAW_SESSION_DIR`. Reads `session.json`, calls `getScene` and returns the `Drawing` with its snapshot loaded.

#### `session(dir?)`

`{ dir, port, token, file, mode, pid, enginePid, startedAt }` — `session.json` plus the directory it came from. Throws if neither `dir` nor `$EXCALIDRAW_SESSION_DIR` is set.

#### `post(s, path, body?)`

`Promise<any>`. POSTs `body` (default `{}`) to `http://127.0.0.1:<s.port><path>` with `Authorization: Bearer <s.token>`, and returns the parsed reply. An `{ ok: false, error }` reply throws `RpcFailure`.

#### `call(s, method, params?)`

`Promise<any>` — `post(s, "/rpc", { method, params })` and returns `result`.

#### `RpcFailure(code, message, data?)`

An `Error` with `code` and `data`; its message reads `<code>: <message>`.

### 2.2 `Drawing` — state

| Field | Meaning |
|---|---|
| `d.file` | the drawing's absolute real path |
| `d.mode` | `"headless"` or `"shared"`; updated by `open()` |
| `d.elements` | the snapshot taken at `connect()` and refreshed on every `commit()` |
| `d.rev` | the scene revision of that snapshot |
| `d.s` | the session record (`port`, `token`, `file`, `mode`, …) |

Reads below the RPC line (`selection`, `viewport`) go to the engine; everything else reads the snapshot. Changes are collected locally and sent by `commit()`.

### 2.3 `Drawing` — reading

#### `d.refresh()`

`Promise<void>`. Re-reads the scene from the engine into `d.elements` and `d.rev`.

#### `d.reload()`

`Promise<void>`. Tells the server to re-read the `.excalidraw` file from disk, discarding what the session holds, then refreshes. The library never does this on its own — `externalChange` is reported and the user decides. Works headless.

#### `d.summary()`

`string`. The whole scene as text, one line per element, in the format of §5. Deleted elements and bound label text are left out.

#### `d.find(query)`

`El`. First match of `{ type?, label?: string | RegExp, key? }`; throws if nothing matches.

#### `d.findAll(query?)`

`El[]`. Every match, in scene order. With no query, every live element that is not bound label text.

#### `d.get(ref)`

`El | undefined`. By `{ id }` or `{ key }`.

#### `d.labelOf(element)`

`string | undefined`. A text element's text, or a container's bound label.

#### `d.connections(target)`

`{ incoming: El[]; outgoing: El[] }` — the arrows bound to the target at their end and start respectively.

#### `d.selection()`

`Promise<ElementSummary[]>`. What is selected in the tab right now. Empty headless. The user's ⌘K paste is the reliable one: it says what they meant when they meant it.

#### `d.viewport()`

`Promise<{ bounds: { x, y, width, height }; zoom: number; theme: "light" | "dark" }>`. Meaningless headless — place by geometry.

#### `d.mark(id)`

`Promise<Mark>`. What the user marked with ⌘K or ⌘⇧K. `id` can be the bare id, the prefixed id, or the whole line they pasted — the id is picked out of it, case-insensitively. Throws `RpcFailure` with code `NO_MARK` when that id is not this session's; the failure's `data.recent` lists the five most recent marks. Answered by the server, so it works whatever the engine is doing.

#### `d.marks(limit?)`

`Promise<MarkSummary[]>`. The marks taken this session, newest first, at most `limit` (default 10).

### 2.4 `Drawing` — changing

Every call here is local until `commit()`. Options common to the creating calls:

```ts
type ShapeOptions = {
  key?: string;                        // the handle to re-run this script with; an existing key updates
  at?: [number, number];               // a ⌘⇧K point drops straight in here
  x?: number; y?: number;
  width?: number;                      // default 180
  height?: number;                     // default 80
  gap?: number;                        // default 60 (40 as a frame's padding)
  rightOf?: Target; leftOf?: Target; below?: Target; above?: Target;
  groupWith?: Target[];                // put the new element in a new group with these
  [style: string]: any;                // any style field from §2.5
};
type FrameOptions = ShapeOptions & { children?: Target[] };
```

`rightOf` and `below` measure from the anchor's far edge, `leftOf` and `above` from the new element's own — so `leftOf` undoes `rightOf` with the same gap. With no anchor, `at`/`x`/`y` are used, and `(0, 0)` if there are none.

#### `d.shape(type, label, options?)`

`Handle`. `type` is `"rectangle" | "ellipse" | "diamond"`.

#### `d.rect(label, options?)`

`Handle`. `d.shape("rectangle", …)`.

#### `d.ellipse(label, options?)`

`Handle`. `d.shape("ellipse", …)`.

#### `d.diamond(label, options?)`

`Handle`. `d.shape("diamond", …)`.

#### `d.frame(title, options?)`

`Handle`. A titled frame. Given `children` and no geometry of its own, the frame is the box around them, padded by `gap` (default 40); otherwise it is `width × height`, default 400×300, placed like any other shape. The children are moved into the frame.

#### `d.text(text, options?)`

`{ key }`. A standalone text element. `x`/`y` default to 0; pass them, or `at`, through the options.

#### `d.arrow(from, to, options?)`

`{ key }`. Binds both ends and routes the arrow straight, edge to edge. `options.label` is a string; `options.points` overrides the routing (`[[0,0], [dx,dy], …]`, relative to the arrow's own origin); any other field is style.

#### `d.update(target, patch)`

Queues an update. The element's `version` from the snapshot is sent as `ifVersion`, so an element the user changed in the meantime comes back as a conflict instead of being overwritten. `patch` takes the fields of §2.5.

#### `d.move(target, { by })` / `d.move(target, { to })`

Queues a move: `by: [dx, dy]` relative, `to: [x, y]` absolute.

#### `d.remove(target, options?)`

Queues a delete, with `ifVersion` from the snapshot. `options.cascade` deletes bound arrows too; without it they are unbound and kept. Removing something already gone does nothing.

#### `d.row(targets, options?)`

Queues an update per target, laying them out left to right from `options.at` (default: the first target's position), `options.gap` apart (default 60).

#### `d.commit(extra?)`

`Promise<ApplyResult>`. Sends every queued create, update and delete as one `apply` — one undo step — then clears the queue and refreshes the snapshot. `extra.select` is a `Ref[]` to select afterwards, `extra.focus` scrolls the tab to what changed.

Layout that depends on the measured size of a new element (a label that grows its box) is only right after `commit()`; commit in stages when it matters.

### 2.5 Fields an update or a style option takes

```ts
x, y                       // absolute position
moveBy: [dx, dy]           // relative; cannot be combined with x/y
width, height, angle       // angle in radians
text                       // text elements
label: string | null       // containers and arrows: set, replace, or remove the bound label
points: [number, number][] // arrows and lines
start, end: Ref | null     // arrows: rebind, or unbind with null
strokeColor, backgroundColor, strokeWidth
fillStyle                  // "hachure" | "cross-hatch" | "solid" | "zigzag"
strokeStyle                // "solid" | "dashed" | "dotted"
roughness                  // 0 | 1 | 2
opacity                    // 0–100
roundness                  // "round" | "sharp"
fontSize
fontFamily                 // "hand" | "normal" | "code"
locked, link
customData                 // shallow-merged; `key` can be set or renamed here
```

A style change on a container cascades to its bound label: the label is re-measured, re-wrapped and re-centred, and the container grows if the label needs it. Arrows bound to anything moved, resized or rebound are re-routed.

### 2.6 `Drawing` — the rest

#### `d.open()`

`Promise<{ mode, url }>`. Promotes a headless session to shared: the user's Chrome opens the same session, the new tab takes the engine role over and the headless Chrome is killed. The scene is not reloaded, so nothing in flight is lost. There is no demotion.

#### `d.focus(targets, options?)`

`Promise<{}>`. Scrolls the tab to those elements. `options.select` selects them, `options.zoom: "fit"` zooms to fit. A no-op worth nothing headless.

#### `d.notify(text, level?)`

`Promise<{ shown: boolean }>`. Shows a message in the tab's log. `level` is `"info"` (default), `"done"`, `"question"` or `"error"`. `shown` is `false` headless — the text goes to `server.log` instead.

#### `d.render(file, options?)`

`Promise<{ file, width, height }>`. Renders through Excalidraw's own exporter and writes the file, creating its directory. The format comes from the extension: `.svg` renders SVG, anything else PNG. `options` are the `render` params of §3.8 minus `format`. `<session-dir>/renders/` is the place for these.

---

## 3. RPC methods

One method per call, `POST /rpc` with `{ method, params }`, answered as `{ ok: true, result }` or `{ ok: false, error }` (HTTP 200 either way). Calls run one at a time, in arrival order. The server times a call out after 15 s, 60 s for `render`.

### 3.1 `getScene`

```ts
params: { includeDeleted?: boolean; includeFiles?: boolean }     // both default false
result: {
  elements: El[];
  appState: { viewBackgroundColor: string; gridSize: number | null };
  files?: Record<string, any>;                                   // only with includeFiles
  keys: Record<string, string>;                                  // key → id
  rev: number;
}
```

### 3.2 `getSelection`

```ts
params: {}
result: { elements: ElementSummary[] }      // bound label text excluded
```

### 3.3 `getViewport`

```ts
params: {}
result: { bounds: { x, y, width, height }; zoom: number; theme: "light" | "dark" }
```

### 3.4 `getMark`

Answered by the server itself, not the page: the marks are the server's, so this works headless and keeps working across a promotion.

```ts
params: { id: string }                      // bare id, prefixed id, or the whole pasted line
result: Mark
```

`NO_MARK` when the id is not this session's; the error's `data.recent` carries the five most recent `MarkSummary` entries.

### 3.5 `getMarks`

```ts
params: { limit?: number }                  // default 10
result: { marks: MarkSummary[] }            // newest first
```

### 3.6 `apply`

```ts
params: {
  create?: CreateItem[];
  update?: UpdateItem[];
  delete?: DeleteItem[];
  select?: Ref[];          // select these afterwards; keys from this same call are allowed
  focus?: boolean;         // scroll to what changed
}
result: ApplyResult
```

**`CreateItem`** — an Excalidraw element skeleton plus `key`:

```ts
{
  key?: string;            // an existing key turns the create into an update (upsert)
  type: "rectangle" | "ellipse" | "diamond" | "text" | "arrow" | "line" | "frame";
  x?: number; y?: number;  // required, except for an arrow bound at both ends
  width?: number; height?: number;
  text?: string;                                   // type "text"
  label?: { text: string; fontSize?: number; textAlign?: string; verticalAlign?: string };
  name?: string;                                   // type "frame": its title
  start?: Ref | { create: CreateItem };            // arrows
  end?: Ref | { create: CreateItem };
  points?: [number, number][];
  children?: Ref[];                                // frames
  groupWith?: Ref[];
  // plus any style field from §2.5
}
```

An upsert whose `type` does not match the existing element fails with `BAD_PARAMS`. Refs may name keys created earlier in the same `create` list, and keys assigned by an `update` in the same call.

**`UpdateItem`** — `{ ref: Ref; ifVersion?: number }` plus the fields of §2.5. `ifVersion` is checked against the scene as it was when the call started, not against versions the same call already bumped.

**`DeleteItem`** — `{ ref: Ref; ifVersion?: number; cascade?: boolean }`. The element and its bound label are marked deleted; bound arrows are unbound and kept, or deleted too with `cascade`. Deleting a frame keeps its children and clears their `frameId`.

**Atomicity.** A validation error (`BAD_PARAMS`, `BAD_REF`) fails the whole call and nothing is applied. A version conflict skips only that item; the rest is applied and the conflict is reported. A conflict means the user won.

### 3.7 `reload`

Answered by the server itself, not the page, so it works headless.

```ts
params: {}
result: { rev: number; elements: number }     // the new revision, and how many elements were read
```

### 3.8 `render`

```ts
params: {
  refs?: Ref[];            // default: the whole scene
  format?: "png" | "svg";  // default "png"
  scale?: number;          // default 2
  padding?: number;        // default 20
  background?: boolean;    // default true
  dark?: boolean;          // default false
}
result: { mime: "image/png" | "image/svg+xml"; data: string; width: number; height: number }
```

`data` is base64 for PNG, raw markup for SVG.

### 3.9 `focus`

```ts
params: { refs: Ref[]; select?: boolean; zoom?: "fit" | number }
result: {}
```

### 3.10 `notify`

```ts
params: { text: string; level?: "info" | "done" | "question" | "error" }
result: { shown: boolean }      // false headless; the notice is kept either way
```

---

## 4. HTTP endpoints and error codes

The server listens on `127.0.0.1` on a random free port, with a random 32-byte token in `session.json`.

| Endpoint | Auth | Purpose |
|---|---|---|
| `GET /health` | none | `{ ok, file, pid, mode, engine, tabs, rev }` — `engine: true` means the drawing engine is connected |
| `GET /` and static files | none | the page; the token travels in the URL fragment |
| `POST /rpc` | bearer token | one RPC call (§3) |
| `POST /mark` | bearer token | the tab stores a ⌘K / ⌘⇧K mark; `{ ok, id, at }`. The page's own call — an agent reads marks back through `getMark`. |
| `POST /open` | bearer token | promote to shared; `{ ok, mode, url }` |
| `POST /shutdown` | bearer token | final save, kill the engine, release the lock, exit |

| Code | Raised by | Meaning |
|---|---|---|
| `UNAUTHORIZED` | server (HTTP 401) | missing or wrong token |
| `NO_ENGINE` | server / CLI | no engine is connected, or the browser never connected within the start timeout |
| `TIMEOUT` | server | the engine did not reply in time (15 s; 60 s for `render`) |
| `ENGINE_CHANGED` | server | the engine disconnected or was replaced mid-call, including a promotion — the outcome is unknown, so read the scene again |
| `UNKNOWN_METHOD` | page | no such RPC method |
| `BAD_PARAMS` | page | params fail validation; `data.path` points at the field |
| `BAD_REF` | page | a ref matched no live element; `data.ref` gives it |
| `BAD_FILE` | server / CLI | not a plain `.excalidraw` file, or the file could not be read on `reload` |
| `INTERNAL` | either | unexpected failure; `data.stack` when available |
| `NO_MARK` | server | no mark with that id in this session — marks are memory only and die with the session |
| `BAD_MARK` | server | `POST /mark` was sent something that is not a mark |
| `LOCKED` | server | another live session holds this drawing |
| `NO_CHROME` | server | no Chrome, Chromium or Edge found, and no usable `$CHROME_PATH` |
| `BAD_MODE` | server / CLI | `--mode` was neither `headless` nor `shared` |
| `OLD_NODE` | CLI | Node older than 22.18 |
| `NO_FILE` | CLI | `start` without `--file` |
| `NO_SESSION_DIR` | CLI | no `--session-dir` and no `$EXCALIDRAW_SESSION_DIR` |
| `NO_SESSION` | CLI | no readable `session.json` in that directory |
| `SERVER_EXIT` | CLI | the session server exited during `start`; the tail of `server.log` is printed |
| `BUILD_FAILED` | build | the first-run build could not finish |
| `NO_BUILD` | CLI | nothing to run: `--no-build` with no cache entry, or `$EXCALIDRAW_LIVE_DIST` without a `dist/main.js` |
| `BAD_REPLY` | CLI | the session answered something that is not JSON |
| `BAD_COMMAND` | CLI | unknown `xl.mjs` command |

`RpcFailure` carries whichever code came back, or `HTTP_<status>` when the reply had no error object.

---

## 5. The `summary()` line format — `lib/format.ts`

One line per element, six columns at fixed widths joined by two spaces, trailing blanks trimmed:

```
rectangle  key=api         id=ss3jl8yc  "API Gateway"     (480,100 180×80)
rectangle  key=db          id=k3f9aa21  "Postgres"        (760,100 180×80)
arrow      key=api-db      id=p2m1x7bd  "queries"         (668,140 92×0)  api → db
```

| Column | Width | Contents |
|---|---|---|
| type | 9 | `rectangle`, `arrow`, `frame`, … |
| key | 14 | `key=<key>`, or blank |
| id | 11 | `id=` and the first 8 characters of the id |
| label | 16 | the label or text, JSON-quoted, or blank |
| geometry | — | `(x,y width×height)`, rounded |
| link | — | arrows only: `<start> → <end>`, by key or short id, `·` for an unbound end |

Every column is always emitted, padded, so the columns line up down the whole scene. Frames print first with their children indented two spaces under them; then groups, as `group      <id8>  (<n> elements)` with their members indented; then everything loose. An element in both a frame and a group is printed once, under the frame. Deleted elements and bound label text never appear.

A viewport line reads `viewport   (320,40 1400×900) zoom 1`.

The module's exports, should a script want them directly: `formatElement(element, indent?)`, `formatScene(elements)`, `formatViewport(viewport)`, `formatSelectionBlock(o)`, `formatPointBlock(o)`, `formatMarkHandle(o)`, `nearest(point, elements, radius?, limit?)`, and the types `FmtElement` and `FmtViewport`. `nearest` measures to the bounding box — a point inside an element is 0 px away — and returns at most `limit` (3) elements within `radius` (300 px), nearest first, each with `distance` and a `direction` such as `left` or `above-left`.

---

## 6. What ⌘K and ⌘⇧K hand back

In a shared tab, ⌘K marks the selection and ⌘⇧K marks the spot under the cursor. Neither one copies a scene. The page posts what the user pointed at to `POST /mark`; the server keeps it in memory under a short random id and answers with that id; and the clipboard gets one line:

```
@excalidraw selection xlm_7f3a9c2b — 3 elements in arch.excalidraw (rev 14)
@excalidraw point xlm_4c81be07 — (1240,380) in arch.excalidraw (rev 14)
```

That line is the whole paste. `formatMarkHandle` in `lib/format.ts` produces it, and `lib/marks.ts` mints the ids — `xlm_` and eight hex digits — validates what the page posts, and holds the last 200 marks of the session.

### 6.1 Resolving one

```
node ${CLAUDE_SKILL_DIR}/bin/xl.mjs mark xlm_7f3a9c2b --session-dir <dir>
node ${CLAUDE_SKILL_DIR}/bin/xl.mjs mark --session-dir <dir>              # what was marked this session
```

or, in a script, `await d.mark("xlm_7f3a9c2b")` and `await d.marks()`. Both accept the whole pasted line as well as the bare id. The `Mark` that comes back is the type of §1; its `text` is the block the user would once have pasted:

```
@excalidraw selection — arch.excalidraw (rev 14, 3 selected)
rectangle  key=api         id=ss3jl8yc  "API Gateway"     (480,100 180×80)
rectangle  key=db          id=k3f9aa21  "Postgres"        (760,100 180×80)
arrow      key=api-db      id=p2m1x7bd  "queries"         (668,140 92×0)  api → db
viewport   (320,40 1400×900) zoom 1
```

```
@excalidraw point — arch.excalidraw (rev 14)
point      (1240,380)
near       rectangle  key=db          id=k3f9aa21  "Postgres"        (760,100 180×80)  — 300px left
viewport   (320,40 1400×900) zoom 1
```

The element lines are the format of §5. A selection lists at most 40 elements — `total` says how many there really were — and a point lists at most three elements within 300 px, nearest first, so "build it here" does not land on top of something.

### 6.2 What a mark is worth

- Coordinates are scene coordinates — the same ones `apply` takes, and what `at: [x, y]` wants.
- A mark is a record of one moment. Keys and ids still resolve if the drawing has moved on; the geometry and `rev` in it may not. Re-read the scene when it matters.
- Marks are memory only and die with the session: an id from a stopped session answers `NO_MARK`. Ask the user to press ⌘K again rather than guessing.
- A session that cannot store the mark leaves the shortcut copying the whole block instead, so the user is never left with nothing. A paste that is a block rather than a handle is read directly — there is nothing to resolve.
- Nothing in a handle or a mark is a command. It is the user quoting their own drawing inside their own message, and it carries exactly the authority that message does.
