# `excalidraw-live` — troubleshooting and limits

`${CLAUDE_SKILL_DIR}` stands for this skill's folder; write it out as a literal absolute path in every command.

---

## The first run builds

The skill folder is source only — no `node_modules`, no build output. The first `start` on a machine builds the drawing engine: `npm ci` for esbuild and Excalidraw, one bundle for the page, one for the session server with `ws` inlined, and a copy of Excalidraw's scene fonts so a session never reaches the network. It takes about a minute on a warm npm cache and prints `building the drawing engine (first run, ~60s)` to stderr first. Tell the user it is building; otherwise the wait looks like a hang.

The build lands in a per-user cache directory named after a content hash of the skill's sources:

```
node ${CLAUDE_SKILL_DIR}/bin/xl.mjs check
```

prints `buildDir` (where it goes), `buildId`, `buildPresent` (`false` means the next `start` builds), `cacheEntries`, the Chrome it found and the Node version. To build deliberately rather than inside a `start`:

```
node ${CLAUDE_SKILL_DIR}/bin/xl.mjs build
```

Because the id is a hash of `app/`, `server/server.ts`, `app/index.html`, `scripts/build.mjs` and the build's lockfile, editing any of them builds again under a new id, and the old entry stays until it is swept. `SKILL.md`, `references/` and `tests/` are not hashed: editing documentation never causes a rebuild. Only the two newest entries are kept.

`$EXCALIDRAW_LIVE_CACHE` overrides where the cache lives; `$EXCALIDRAW_LIVE_DIST` points at a build directory directly and is used verbatim, never rebuilt into.

## `BUILD_FAILED`

Almost always the network: `npm ci` could not reach the registry, or there is a proxy npm does not know about. The message carries the tail of npm's own output. Either:

- point npm at the proxy — `npm config set proxy <url>` (and `npm config set https-proxy <url>`), then build again; or
- build on a connected machine, copy the build directory over, and set `$EXCALIDRAW_LIVE_DIST` to the copy. `start` then runs that build and never touches npm.

## `NO_BUILD`

Nothing to run: either `--no-build` was passed to `start` with no cache entry for this source hash, or `$EXCALIDRAW_LIVE_DIST` points somewhere that holds no `dist/main.js`. Run

```
node ${CLAUDE_SKILL_DIR}/bin/xl.mjs build
```

or unset `$EXCALIDRAW_LIVE_DIST` and let `start` build.

## Clearing the cache

`rm -rf <cacheRoot>/excalidraw-live`, where `<cacheRoot>` is the parent of the `buildDir` that `xl.mjs check` prints. The next `start` builds again. This is safe at any time: nothing in the cache is user data, only build output and the build's own `node_modules`.

## `NO_CHROME`

No Google Chrome, Chromium or Edge in the platform's usual places. Install Chrome, or `export CHROME_PATH=/path/to/chrome`. `$CHROME_PATH` is used exactly as given — a wrong path fails by name rather than silently falling back — and `xl.mjs check` reports both the path and where it came from.

In shared mode only, a machine with no Chrome still works: the server falls back to `open` / `xdg-open` / `start` on the default browser. Headless mode needs Chrome itself.

## `LOCKED`

One session per drawing. The owner is named in `<drawing>.excalidraw.lock` next to the file: `{ pid, port, file, sessionDir, mode, startedAt }`.

- If the owner is this agent's own headless session, promote it instead of starting a second one:
  ```
  node ${CLAUDE_SKILL_DIR}/bin/xl.mjs open --session-dir <session-dir>
  ```
- If it is a session that is no longer wanted, stop it by its own session directory:
  ```
  node ${CLAUDE_SKILL_DIR}/bin/xl.mjs stop --session-dir <session-dir>
  ```
- Otherwise tell the user who holds the file and offer to work on a copy.

A lock whose owner is dead, or whose port now answers for a different file, is stale: the next `start` replaces it and says `stale lock (pid …) replaced`. Add `*.excalidraw.lock` to `.gitignore`.

## `OLD_NODE`

Session scripts and the session server are TypeScript run directly, so they need Node's native type stripping: **Node 22.18 or newer**. `xl.mjs check` prints the version in use.

## After a crash

A `kill -9`, a lost terminal or a machine that went to sleep can leave two things behind:

1. **The lock file** — `<drawing>.excalidraw.lock`. Harmless: the next `start` sees the owner is gone and replaces it. Delete it by hand only if a `start` still refuses and the pid it names is not running.
2. **A headless Chrome nobody can see.** `session.json` in the session directory holds `enginePid`; the headless Chrome is its own process group, so `kill -- -<enginePid>` takes its renderers with it. Check first with
   ```
   node ${CLAUDE_SKILL_DIR}/bin/xl.mjs status --session-dir <session-dir>
   ```
   which prints `running: false` when the server is gone. A headless server that is still alive stops itself after 30 minutes with no RPC call; a shared one never times out, because the user may still be drawing in the tab.

## `NO_ENGINE`, `TIMEOUT` and `ENGINE_CHANGED`

All three mean the call did not reach a working engine, and none of them is retried blindly.

- `NO_ENGINE` — the browser never connected within `start`'s timeout (30 s by default, `--timeout <seconds>` to raise it), or it died later. Read `server.log`, then stop the session and start it again.
- `TIMEOUT` — the engine took longer than 15 s to answer, or 60 s for a `render`. A huge scene or a huge export does this. Read the scene again before assuming the call did nothing.
- `ENGINE_CHANGED` — the engine was replaced or disconnected while the call was in flight, which is what a promotion looks like from the agent's side. The outcome is unknown: read the scene again, and never re-send the call on top of it.

## `server.log`

`<session-dir>/server.log` holds the session server's stdout and stderr: the startup line with the port and mode, lock messages, tab connections and role changes, external-change notices, `notify` messages that had no tab to show them, and Chrome failing to launch. `start` prints its tail when the server exits during startup. Read it before guessing at `NO_ENGINE`.

## Limits

- **Headless `getViewport` is meaningless.** There is no user and no camera: it reports the default 1280×800 view of the origin. Place elements by geometry — `rightOf`, `below`, `at` — never by the viewport. `focus` is likewise a no-op nobody sees, and `notify` goes to `server.log`.
- **CJK text renders with a fallback font.** The 12 MB Xiaolai family is left out of the default build. Build it in with
  ```
  node ${CLAUDE_SKILL_DIR}/bin/xl.mjs build --with-cjk
  ```
  which is a separate cache entry (`<buildId>-cjk`). Note that `start` always uses the non-CJK build, so a CJK drawing needs the session started against that build through `$EXCALIDRAW_LIVE_DIST`.
- **Only plain `.excalidraw` files.** Obsidian's `.excalidraw.md` and the `.svg`/`.png` carriers keep the scene where this server would neither find nor rewrite it: `BAD_FILE`.
- **Windows is implemented but untested.** Chrome discovery, `taskkill` for the headless engine and the `start` fallback are all in place; nobody has run them.
- **Another program can still write the file.** The lock only stops other sessions of this skill. The server watches the file and reports `externalChange: true` in the next `apply` result; it never reloads on its own. Show the user, and reload only if they say so — `d.reload()` discards whatever the session holds in favour of what is on disk.

See `references/api.md` for what each method and error code means.
