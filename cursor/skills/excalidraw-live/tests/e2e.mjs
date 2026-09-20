#!/usr/bin/env node
// The whole skill, end to end, from a cache that does not exist yet.
//
//   node tests/e2e.mjs
//
// This is the proof that a skill folder carrying nothing but source still
// produces a working drawing session: it builds on first use, drives a real
// headless Chrome through the CLI and the library, and checks what lands on
// disk. It needs Chrome, and on a cold cache the network.
//
// Everything it touches is its own: its own temporary directory, its own
// $EXCALIDRAW_LIVE_CACHE inside it, its own drawing file, its own session
// directory and the server's own ephemeral port. The user's cache is never
// read or written, and nothing is left behind.
//
// The session scripts are written into the session directory and run exactly
// the way an agent runs them — `EXCALIDRAW_SESSION_DIR=<dir> node
// <dir>/scripts/<name>.ts` — because that path is part of what this proves.
//
// POSIX only (macOS, Linux): the npm shim that proves the second start does
// not build is a /bin/sh script.
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const HERE = path.dirname(fileURLToPath(import.meta.url));
const SKILL = path.resolve(HERE, "..");
const XL = path.join(SKILL, "bin", "xl.mjs");
const LIB = path.join(SKILL, "lib", "index.ts");
const REPORT = "##result##";                       // how a session script hands its answer back

// ---- reporting. One line per assertion, so a run reads as the list above.
let passes = 0;
function ok(label) { console.log(`ok ${String(++passes).padStart(2)}  ${label}`); }
function check(label, cond, detail) {
  if (!cond) throw new Error(`${label}${detail ? `\n      ${detail}` : ""}`);
  ok(label);
}
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

/** Poll until `fn` returns something truthy, or give up with a readable reason. */
async function waitFor(what, fn, ms = 15000) {
  const until = Date.now() + ms;
  let last = "";
  for (;;) {
    try { const v = await fn(); if (v) return v; } catch (e) { last = String(e?.message ?? e); }
    if (Date.now() > until) throw new Error(`${what} did not happen in ${ms} ms${last ? ` — ${last}` : ""}`);
    await sleep(150);
  }
}

// ---- the environment this test builds for itself
if (process.platform === "win32") {
  console.error("tests/e2e.mjs runs on macOS and Linux only (it shims npm with a /bin/sh script)");
  process.exit(1);
}
const [major, minor] = String(process.versions.node).split(".").map(Number);
if (major < 22 || (major === 22 && minor < 18)) {
  console.error(`OLD_NODE: this test runs the session scripts with plain node; found ${process.version}, need 22.18+`);
  process.exit(1);
}

const tmp = fs.realpathSync(fs.mkdtempSync(path.join(os.tmpdir(), "xl-e2e-")));
const file = path.join(tmp, "e2e.excalidraw");
const cache = path.join(tmp, "cache");
const cacheHome = path.join(cache, "excalidraw-live");
const sessionDir = tmp;                            // the session lives in the same directory
const warmDir = path.join(tmp, "warm");            // the second start, on its own file
const warmFile = path.join(warmDir, "warm.excalidraw");
const shimDir = path.join(tmp, "shim");
const npmMarker = path.join(tmp, "npm-ran.txt");

const realNpm = String(spawnSync("which", ["npm"], { encoding: "utf8" }).stdout ?? "").trim().split("\n")[0];
if (!realNpm) { console.error("no npm on PATH — the cold build cannot run"); process.exit(1); }
fs.mkdirSync(shimDir, { recursive: true });
fs.mkdirSync(warmDir, { recursive: true });
// Every npm the build reaches for goes through here first. The cold start must
// leave a mark in the marker file and the warm one must not: that is the whole
// difference between building and reusing a build.
fs.writeFileSync(
  path.join(shimDir, "npm"),
  `#!/bin/sh\necho "npm $1" >> ${JSON.stringify(npmMarker)}\nexec ${JSON.stringify(realNpm)} "$@"\n`,
  { mode: 0o755 },
);

/** The environment every child gets: our cache, our npm, and no inherited build. */
function env(extra = {}) {
  const e = { ...process.env, EXCALIDRAW_LIVE_CACHE: cache, PATH: `${shimDir}:${process.env.PATH}`, ...extra };
  delete e.EXCALIDRAW_LIVE_DIST;                   // a build pointed at by hand would defeat the point
  return e;
}

function node(argv, opts = {}) {
  const r = spawnSync(process.execPath, argv, {
    encoding: "utf8", cwd: SKILL, env: env(opts.env), timeout: opts.timeout ?? 300000,
  });
  return { code: r.status, out: String(r.stdout ?? ""), err: String(r.stderr ?? "") };
}

/** A CLI command, with its one JSON object parsed. */
function xl(argv, opts = {}) {
  const r = node([XL, ...argv], opts);
  const line = r.out.trim().split("\n").filter(Boolean).pop();
  let json = null;
  try { json = line ? JSON.parse(line) : null; } catch {}
  return { ...r, json };
}

/** A session script, written where an agent writes it and run the same way. */
function script(name, body) {
  fs.mkdirSync(path.join(sessionDir, "scripts"), { recursive: true });
  fs.writeFileSync(path.join(sessionDir, "scripts", `${name}.ts`), body);
}
function runScript(name) {
  const p = path.join(sessionDir, "scripts", `${name}.ts`);
  const r = spawnSync(process.execPath, [p], {
    encoding: "utf8", cwd: tmp, timeout: 120000,
    env: env({ EXCALIDRAW_SESSION_DIR: sessionDir }),
  });
  const out = String(r.stdout ?? ""), err = String(r.stderr ?? "");
  if (r.status !== 0) throw new Error(`${name}.ts exited ${r.status}\n${out}\n${err}`);
  const line = out.split("\n").find((l) => l.startsWith(REPORT));
  if (!line) throw new Error(`${name}.ts printed no ${REPORT} line\n${out}\n${err}`);
  return JSON.parse(line.slice(REPORT.length));
}

const readScene = () => JSON.parse(fs.readFileSync(file, "utf8"));
const liveOf = (s) => s.elements.filter((e) => !e.isDeleted);

/** Chrome processes still holding a profile inside our temporary directory. */
function ourChromes() {
  const r = spawnSync("ps", ["-A", "-o", "pid=,command="], { encoding: "utf8", maxBuffer: 64 << 20 });
  return String(r.stdout ?? "").split("\n").filter((l) => l.includes(`--user-data-dir=${tmp}`));
}

// ---- the session scripts. script1 is written once and run twice, unchanged:
// re-runnability is a promise the skill makes, so the second run is the same
// file and the same bytes.
const SCRIPT1 = `import { connect } from ${JSON.stringify(LIB)};

const d = await connect();
d.rect("Alpha", { key: "a", at: [100, 100] });
d.rect("Beta", { key: "b", rightOf: { key: "a" }, gap: 120 });
d.arrow({ key: "a" }, { key: "b" }, { key: "a-to-b" });
const res = await d.commit();
console.log(d.summary());
console.log("${REPORT}" + JSON.stringify({
  created: res.created,
  updated: res.updated,
  conflicts: res.conflicts,
  live: d.elements.length,
}));
`;

// A bound label wraps to its container's width, so a longer label grows the
// container downwards — the sentence has to be long enough to need a line the
// old height had no room for, or nothing about the box changes.
const LONGER = "A much longer label than before, long enough to need more than two lines";

const SCRIPT2 = `import { connect } from ${JSON.stringify(LIB)};

const d = await connect();
const before = d.find({ key: "a" });
const was = { x: before.x, y: before.y, width: before.width, height: before.height };
d.update(before, { label: ${JSON.stringify(LONGER)} });
const res = await d.commit();
const box = d.find({ key: "a" });
const label = d.elements.find((e) => e.containerId === box.id);
console.log(d.summary());
console.log("${REPORT}" + JSON.stringify({
  was,
  box: { x: box.x, y: box.y, width: box.width, height: box.height },
  label: label && { x: label.x, y: label.y, width: label.width, height: label.height, text: label.text },
  conflicts: res.conflicts,
}));
`;

const SCRIPT3 = `import { connect } from ${JSON.stringify(LIB)};
import path from "node:path";

const d = await connect();
const out = path.join(process.env.EXCALIDRAW_SESSION_DIR, "renders", "a.png");
const r = await d.render(out);
console.log("${REPORT}" + JSON.stringify(r));
`;

const SCRIPT5 = `import { connect } from ${JSON.stringify(LIB)};
import fs from "node:fs";

const d = await connect();
// Another program edits the drawing while the session holds it.
const raw = JSON.parse(fs.readFileSync(d.file, "utf8"));
raw.appState = { ...(raw.appState ?? {}), viewBackgroundColor: "#fff8e7" };
fs.writeFileSync(d.file, JSON.stringify(raw, null, 2));
await new Promise((r) => setTimeout(r, 500));
const res = await d.commit();
console.log("${REPORT}" + JSON.stringify(res));
`;

// ---- the run
async function main() {
  const pre = xl(["check"]);
  if (pre.code !== 0) throw new Error(`xl.mjs check says this machine cannot draw:\n${pre.err || pre.out}`);
  check("the cache this test builds into does not exist yet", !fs.existsSync(cacheHome), cacheHome);

  // 1. cold start: build, then the engine connects
  const started = xl(["start", "--file", file, "--session-dir", sessionDir, "--mode", "headless", "--timeout", "90"]);
  check("start returns a session on a cold cache", started.code === 0 && !!started.json?.port,
        `exit ${started.code}\n${started.err || started.out}`);
  const session = started.json;

  const entries = fs.readdirSync(cacheHome);
  const ids = entries.filter((n) => !n.startsWith(".tmp-"));
  check("the build landed in our own cache", ids.length === 1 && fs.existsSync(path.join(cacheHome, ids[0], "dist", "main.js")),
        `${cacheHome}: ${entries.join(", ") || "(empty)"}`);
  check("no .tmp-* directory survived the build", entries.every((n) => !n.startsWith(".tmp-")), entries.join(", "));
  check("the cold build really ran npm", fs.existsSync(npmMarker), `${npmMarker} was never written`);

  // 2. a second start reuses it
  fs.rmSync(npmMarker, { force: true });
  const t0 = Date.now();
  const warm = xl(["start", "--file", warmFile, "--session-dir", warmDir, "--mode", "headless", "--timeout", "30"], { timeout: 60000 });
  const warmMs = Date.now() - t0;
  check("a second start returns a session too", warm.code === 0 && !!warm.json?.port, `exit ${warm.code}\n${warm.err || warm.out}`);
  check("the second start spawned no npm", !fs.existsSync(npmMarker), fs.existsSync(npmMarker) ? fs.readFileSync(npmMarker, "utf8") : "");
  check("the second start returned in under 5 s", warmMs < 5000, `took ${warmMs} ms`);
  const warmStop = xl(["stop", "--session-dir", warmDir]);
  check("the second session stops again", warmStop.code === 0 && warmStop.json?.stopped === true, warmStop.err || warmStop.out);

  // 3. script 1 — two labelled boxes and an arrow
  script("script1", SCRIPT1);
  const r1 = runScript("script1");
  const created = Object.values(r1.created);
  check("script 1 created three elements", created.length === 3, JSON.stringify(r1.created));
  check("script 1 hit no conflicts", r1.conflicts.length === 0, JSON.stringify(r1.conflicts));

  const scene = await waitFor("the drawing reaching disk", () => {
    const s = readScene();
    return s.elements.length === 5 ? s : null;
  });
  check("the file on disk parses and holds 5 elements", liveOf(scene).length === 5,
        scene.elements.map((e) => e.type).join(", "));
  const boxes = scene.elements.filter((e) => e.type === "rectangle");
  const texts = scene.elements.filter((e) => e.type === "text" && e.containerId);
  const arrows = scene.elements.filter((e) => e.type === "arrow");
  check("they are 2 boxes, 2 bound texts and 1 arrow", boxes.length === 2 && texts.length === 2 && arrows.length === 1,
        scene.elements.map((e) => e.type).join(", "));

  const arrow = arrows[0];
  const pts = arrow.points ?? [];
  const finite = pts.length >= 2 && pts.every((p) => p.length === 2 && p.every((n) => Number.isFinite(n)));
  const moved = finite && (Math.abs(pts.at(-1)[0] - pts[0][0]) > 1 || Math.abs(pts.at(-1)[1] - pts[0][1]) > 1);
  check("the arrow's points are finite and non-zero (routing ran)", finite && moved, JSON.stringify(pts));
  check("the arrow is bound to both boxes", !!arrow.startBinding && !!arrow.endBinding,
        JSON.stringify({ start: arrow.startBinding, end: arrow.endBinding }));
  check("the bound label has a measured width (a real canvas measured it)", texts.every((t) => t.width > 0),
        texts.map((t) => `${t.text}=${t.width}`).join(", "));

  // 4. script 2 — a longer label
  script("script2", SCRIPT2);
  const r2 = runScript("script2");
  check("script 2 hit no conflicts", r2.conflicts.length === 0, JSON.stringify(r2.conflicts));
  check("the container grew to fit the longer label",
        r2.box.height > r2.was.height || r2.box.width > r2.was.width,
        `${r2.was.width}×${r2.was.height} → ${r2.box.width}×${r2.box.height}`);
  const dx = Math.abs((r2.label.x + r2.label.width / 2) - (r2.box.x + r2.box.width / 2));
  const dy = Math.abs((r2.label.y + r2.label.height / 2) - (r2.box.y + r2.box.height / 2));
  check("the label is centred in it", r2.label.text.replace(/\n/g, " ") === LONGER && dx <= 1.5 && dy <= 1.5,
        `off by ${dx.toFixed(2)},${dy.toFixed(2)} — ${JSON.stringify(r2.label)}`);

  // 5. script 3 — a render read back
  script("script3", SCRIPT3);
  const r3 = runScript("script3");
  const png = fs.readFileSync(path.join(sessionDir, "renders", "a.png"));
  check("the render is a PNG of more than 2 KB", png.length > 2048 &&
        png.subarray(0, 8).equals(Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a])),
        `${png.length} bytes, magic ${png.subarray(0, 4).toString("hex")}, ${r3.width}×${r3.height}`);

  // 6. script 1 again, unchanged
  const r4 = runScript("script1");
  check("re-running script 1 creates nothing new", Object.keys(r4.created).length === 0, JSON.stringify(r4.created));
  check("it updates the same three ids", [...r4.updated].sort().join() === [...created].sort().join(),
        `${JSON.stringify(r4.updated)} vs ${JSON.stringify(created)}`);
  check("it hit no conflicts and the scene still holds 5 elements", r4.conflicts.length === 0 && r4.live === 5,
        `live=${r4.live} conflicts=${JSON.stringify(r4.conflicts)}`);

  // 7. script 5 — someone else writes the file
  script("script5", SCRIPT5);
  const r5 = runScript("script5");
  check("the next commit reports externalChange", r5.externalChange === true, JSON.stringify(r5));

  // 8. stop
  const stopped = xl(["stop", "--session-dir", sessionDir]);
  check("stop exits 0 and confirms the session is gone", stopped.code === 0 && stopped.json?.stopped === true,
        `exit ${stopped.code}\n${stopped.err || stopped.out}`);
  check("the lock file is gone", !fs.existsSync(`${file}.lock`), `${file}.lock still exists`);
  const leftovers = await waitFor("our Chrome processes exiting", async () => {
    const left = ourChromes();
    return left.length ? null : [];
  }, 15000).catch(() => ourChromes());
  check("no Chrome is left holding our profile", leftovers.length === 0, leftovers.join("\n"));

  console.log(`\n${passes} assertions passed (session port ${session.port}, warm start ${warmMs} ms)`);
}

let failure = null;
try {
  await main();
} catch (e) {
  failure = e;
} finally {
  // Whatever happened, nothing of ours outlives this process.
  for (const d of [sessionDir, warmDir]) {
    try { xl(["stop", "--session-dir", d], { timeout: 20000 }); } catch {}
  }
  for (const line of ourChromes()) {
    const pid = Number(line.trim().split(/\s+/)[0]);
    if (Number.isFinite(pid)) { try { process.kill(pid, "SIGKILL"); } catch {} }
  }
  fs.rmSync(tmp, { recursive: true, force: true });
}
if (failure) {
  console.error(`\nnot ok ${passes + 1}  ${failure.message}`);
  process.exit(1);
}
