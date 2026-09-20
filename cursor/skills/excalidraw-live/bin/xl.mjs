#!/usr/bin/env node
// The session CLI: one entry point for starting, inspecting, promoting,
// stopping and building a drawing session.
//
//   node bin/xl.mjs start  --file <path> [--session-dir <dir>] [--mode headless|shared]
//                          [--timeout 30] [--no-build]
//   node bin/xl.mjs status [--session-dir <dir>]
//   node bin/xl.mjs open   [--session-dir <dir>]
//   node bin/xl.mjs stop   [--session-dir <dir>]
//   node bin/xl.mjs build  [--force] [--with-cjk] [--offline]
//   node bin/xl.mjs check
//
// Every command that says anything says it as one JSON object on stdout, so the
// skill's instructions never have to describe process handling, polling or
// parsing prose. Failures go to stderr as "<CODE>: <message>" with exit 1.
//
// Plain ESM, no dependencies, no TypeScript syntax: this file has to parse on
// any Node >= 18 so that it can report a too-old Node itself rather than fail
// to load.
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { spawn } from "node:child_process";
import { fileURLToPath, pathToFileURL } from "node:url";

// Where a build lives is decided in exactly one place, and this is not it.
import { build, buildId, cacheRoot, resolveBuild } from "../scripts/build.mjs";

const USAGE = `usage:
  xl.mjs start  --file <path.excalidraw> [--session-dir <dir>] [--mode headless|shared] [--timeout 30] [--no-build]
  xl.mjs status [--session-dir <dir>]
  xl.mjs open   [--session-dir <dir>]
  xl.mjs stop   [--session-dir <dir>]
  xl.mjs build  [--force] [--with-cjk] [--offline]
  xl.mjs check`;

// ---- plumbing

/** Prints "<code>: <message>" on stderr and leaves with 1. */
function die(code, message) {
  console.error(`${code}: ${message}`);
  process.exit(1);
}

const out = (o) => console.log(JSON.stringify(o));

export function skillDir() {
  return path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
}

export function parseArgs(argv) {
  const flag = (n) => argv.includes(n);
  const get = (n) => {
    const i = argv.indexOf(n);
    return i >= 0 ? argv[i + 1] : undefined;
  };
  const timeout = get("--timeout");
  const mode = get("--mode");
  if (mode !== undefined && mode !== "headless" && mode !== "shared") {
    die("BAD_MODE", `--mode must be headless or shared (got ${mode})`);
  }
  return {
    cmd: argv.find((a) => !a.startsWith("-")),
    file: get("--file"),
    sessionDir: get("--session-dir"),
    mode,
    timeout: timeout === undefined ? 30 : Number(timeout),
    noBuild: flag("--no-build"),
    force: flag("--force"),
    withCjk: flag("--with-cjk"),
    offline: flag("--offline"),
  };
}

/** The session directory, from the flag or the environment — never guessed. */
function needSessionDir(a) {
  const dir = a.sessionDir ?? process.env.EXCALIDRAW_SESSION_DIR;
  if (!dir) die("NO_SESSION_DIR", "no session dir: pass --session-dir or set EXCALIDRAW_SESSION_DIR");
  return path.resolve(dir);
}

/** The session file, or a NO_SESSION throw — callers decide what absence means. */
export function readSession(dir) {
  const p = path.join(dir, "session.json");
  let raw;
  try {
    raw = fs.readFileSync(p, "utf8");
  } catch {
    const e = new Error(`no session at ${p}`);
    e.code = "NO_SESSION";
    throw e;
  }
  try {
    return JSON.parse(raw);
  } catch {
    const e = new Error(`${p} is not readable JSON`);
    e.code = "NO_SESSION";
    throw e;
  }
}

/** The health of whatever answers on that port, or null — never a throw. */
export async function health(port) {
  if (!port) return null;
  try {
    const r = await fetch(`http://127.0.0.1:${port}/health`, { signal: AbortSignal.timeout(2000) });
    if (!r.ok) return null;
    return await r.json();
  } catch {
    return null;
  }
}

/** A POST to the session, carrying its token. */
export async function post(session, p, body) {
  const r = await fetch(`http://127.0.0.1:${session.port}${p}`, {
    method: "POST",
    headers: { "content-type": "application/json", authorization: `Bearer ${session.token}` },
    body: JSON.stringify(body ?? {}),
    signal: AbortSignal.timeout(20000),
  });
  const text = await r.text();
  try {
    return JSON.parse(text || "{}");
  } catch {
    return { ok: false, error: { code: "BAD_REPLY", message: text.slice(0, 500) } };
  }
}

const url = (s) => `http://127.0.0.1:${s.port}/#token=${s.token}`;
const view = (s, dir) => ({ port: s.port, file: s.file, mode: s.mode, sessionDir: dir, url: url(s) });

function tail(file, n) {
  try {
    return fs.readFileSync(file, "utf8").trimEnd().split("\n").slice(-n).join("\n");
  } catch {
    return "";
  }
}

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

/** Chrome discovery lives in the server; this reaches for that one copy. */
async function findChrome() {
  const scratch = fs.mkdtempSync(path.join(os.tmpdir(), "xl-chrome-"));
  const argv = process.argv;
  // server.ts settles its file and session directory at module scope, so the
  // probe points both at a throwaway directory and then takes them away again.
  process.argv = [process.execPath, "xl", "--file", path.join(scratch, "probe.excalidraw"), "--session-dir", scratch];
  try {
    const mod = await import(pathToFileURL(path.join(skillDir(), "server", "server.ts")).href);
    return mod.findChrome();
  } catch {
    return null;
  } finally {
    process.argv = argv;
    fs.rmSync(scratch, { recursive: true, force: true });
  }
}

function nodeAtLeast(major, minor) {
  const [ma, mi] = String(process.versions.node).split(".").map(Number);
  return ma > major || (ma === major && mi >= minor);
}

// ---- start

export async function cmdStart(a) {
  // The server and the bundle it runs need type stripping and import.meta.main.
  if (!nodeAtLeast(22, 18)) {
    die("OLD_NODE", `the session scripts need Node 22.18+; found ${process.version}`);
  }
  if (!a.file) die("NO_FILE", "pass --file <path.excalidraw>");
  const file = path.resolve(a.file);
  // `.excalidraw` is the only format, and saying so here costs nothing —
  // waiting for the server to say it would mean building first.
  if (!file.endsWith(".excalidraw")) {
    die("BAD_FILE", `only .excalidraw files are supported (got ${path.extname(file) || path.basename(file)})`);
  }
  const sessionDir = needSessionDir(a);
  fs.mkdirSync(path.join(sessionDir, "scripts"), { recursive: true });
  fs.mkdirSync(path.join(sessionDir, "renders"), { recursive: true });

  const real = () => (fs.existsSync(file) ? fs.realpathSync(file) : file);

  // start is idempotent: a live session on this same file is the answer.
  try {
    const s = readSession(sessionDir);
    const h = await health(s.port);
    if (h && h.file === real()) return out(view({ ...s, mode: h.mode ?? s.mode }, sessionDir));
  } catch {}

  const b = resolveBuild(a.withCjk);
  if (!b.present) {
    if (b.source === "env") {
      die("NO_BUILD", `$EXCALIDRAW_LIVE_DIST is ${b.dir}, which holds no build (no dist/main.js) — point it at a build directory or unset it`);
    }
    if (a.noBuild) die("NO_BUILD", `no build for ${b.id} — run: xl.mjs build`);
    console.error(`building the drawing engine (first run, ~60s) — ${b.dir}`);
    await cmdBuild({ withCjk: a.withCjk, silent: true });   // BUILD_FAILED leaves with the log tail
  }

  const logPath = path.join(sessionDir, "server.log");
  const log = fs.openSync(logPath, "a");
  const child = spawn(
    process.execPath,
    [path.join(b.dir, "server.mjs"), "--file", file, "--session-dir", sessionDir, "--mode", a.mode ?? "headless"],
    { detached: true, stdio: ["ignore", log, log] },
  );
  let exited = false;
  child.on("exit", () => (exited = true));
  child.unref();

  const timeout = Number.isFinite(a.timeout) && a.timeout > 0 ? a.timeout : 30;
  const deadline = Date.now() + timeout * 1000;
  while (Date.now() < deadline) {
    await sleep(250);
    let s = null;
    try {
      s = readSession(sessionDir);
    } catch {}
    if (s) {
      const h = await health(s.port);
      // The session file may still be the previous one, so the file it serves
      // is checked too before its engine is believed.
      if (h && h.file === real() && h.engine) return out(view({ ...s, mode: h.mode ?? s.mode }, sessionDir));
    }
    if (exited) die("SERVER_EXIT", `the session server exited — ${logPath}\n${tail(logPath, 20)}`);
  }
  die("NO_ENGINE", `the browser did not connect in ${timeout}s — see ${logPath}`);
}

// ---- the other commands

export async function cmdStatus(a) {
  const dir = needSessionDir(a);
  let s;
  try {
    s = readSession(dir);
  } catch {
    return out({ running: false });
  }
  const h = await health(s.port);
  if (!h) return out({ running: false });
  out({ running: true, ...h, port: s.port, sessionDir: dir, pid: s.pid, enginePid: s.enginePid, startedAt: s.startedAt, url: url(s) });
}

export async function cmdOpen(a) {
  const dir = needSessionDir(a);
  let s;
  try {
    s = readSession(dir);
  } catch (e) {
    die("NO_SESSION", `${e.message} — run: xl.mjs start --file <path.excalidraw>`);
  }
  let r;
  try {
    r = await post(s, "/open", {});
  } catch (e) {
    die("NO_SESSION", `the session on port ${s.port} did not answer: ${String(e && e.message ? e.message : e)}`);
  }
  if (!r.ok) die(r.error?.code ?? "OPEN_FAILED", r.error?.message ?? "the session refused /open");
  out({ mode: r.mode ?? "shared", url: r.url ?? url(s) });
}

export async function cmdStop(a) {
  const dir = needSessionDir(a);
  const sessionFile = path.join(dir, "session.json");
  let s;
  try {
    s = readSession(dir);
  } catch {
    return out({ stopped: false, running: false });       // nothing to stop is not a failure
  }
  let asked = false;
  try {
    asked = !!(await post(s, "/shutdown", {})).ok;
  } catch {}

  const gone = async (ms) => {
    const until = Date.now() + ms;
    while (Date.now() < until) {
      if (!(await health(s.port))) return true;
      await sleep(200);
    }
    return !(await health(s.port));
  };

  let stopped = asked ? await gone(5000) : false;
  if (!stopped && s.pid) {
    try {
      process.kill(s.pid, "SIGTERM");
      stopped = await gone(3000);
      if (!stopped) {
        process.kill(s.pid, "SIGKILL");
        stopped = await gone(2000);
      }
    } catch {
      stopped = !(await health(s.port));                  // already gone
    }
  }
  fs.rmSync(sessionFile, { force: true });
  out({ stopped: true, wasRunning: true, port: s.port, file: s.file, sessionDir: dir, confirmed: stopped });
}

export async function cmdBuild(a) {
  const r = build({ withCjk: !!a.withCjk, force: !!a.force, offline: !!a.offline });
  // start's stdout is one session object, so a build it triggered reports aside.
  if (a.silent) console.error(JSON.stringify(r));
  else out(r);
  return r;
}

export async function cmdCheck() {
  const b = resolveBuild(false);
  const chrome = await findChrome();
  const chromeOk = !!chrome && fs.existsSync(chrome);
  const present =
    fs.existsSync(path.join(b.dir, "dist/main.js")) &&
    fs.existsSync(path.join(b.dir, "dist/fonts/Excalifont")) &&
    fs.existsSync(path.join(b.dir, "server.mjs"));

  const home = path.join(cacheRoot(), "excalidraw-live");
  let entries = [];
  try {
    entries = fs.readdirSync(home).filter((n) => !n.startsWith(".tmp-"));
  } catch {}

  const o = {
    node: process.version,
    platform: `${process.platform}-${process.arch}`,
    chrome,
    chromeSource: process.env.CHROME_PATH ? "CHROME_PATH" : "search",
    skillDir: skillDir(),
    buildId: b.id ?? buildId(false),
    buildDir: b.dir,
    buildSource: b.source,
    buildPresent: present,
    cacheEntries: entries,
    // A missing build is the design, not a broken install: it is made on first
    // use. Only a missing browser makes this machine unable to draw.
    ok: chromeOk,
  };
  if (!chromeOk) {
    o.problem = chrome
      ? `$CHROME_PATH points at ${chrome}, which does not exist`
      : "no Chrome, Chromium or Edge found — install one or set $CHROME_PATH";
  }
  if (!present) o.hint = `no build yet — run: xl.mjs build (start builds it too)`;
  out(o);
  if (!o.ok) process.exit(1);
}

// ---- main

const COMMANDS = { start: cmdStart, status: cmdStatus, open: cmdOpen, stop: cmdStop, build: cmdBuild, check: cmdCheck };

async function main(argv) {
  if (argv.includes("--help") || argv.includes("-h")) {
    console.log(USAGE);
    return;
  }
  const a = parseArgs(argv);
  const run = COMMANDS[a.cmd];
  if (!run) {
    console.error(`${a.cmd ? `BAD_COMMAND: unknown command ${a.cmd}` : "BAD_COMMAND: no command"}\n${USAGE}`);
    process.exit(1);
  }
  await run(a);
}

if (process.argv[1] && import.meta.url === pathToFileURL(fs.realpathSync(process.argv[1])).href) {
  main(process.argv.slice(2)).catch((e) => die(e?.code ?? "FAILED", String(e?.message ?? e)));
}
