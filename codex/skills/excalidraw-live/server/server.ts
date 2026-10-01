// The session server: HTTP for the agent, WebSocket for the drawing engine. It
// is the only writer of the .excalidraw file and the holder of the lock.
//
// A session has exactly one engine — the page that owns the scene. By default
// the server launches that page itself, in headless Chrome, and nobody watches.
// `POST /open` puts a visible tab in front of it: the tab takes the engine role
// over and the headless Chrome we own is killed, without restarting the server.
//
// This file is source. The build bundles it (with `ws` inlined) to
// <cache>/<buildId>/server.mjs, and that is what a session runs.
import http from "node:http";
import fs from "node:fs";
import fsp from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import crypto from "node:crypto";
import { spawn, spawnSync } from "node:child_process";
import { fileURLToPath, pathToFileURL } from "node:url";
import { BadMark, MarkStore, validateMark } from "../lib/marks.ts";

type Mode = "headless" | "shared";
// A `ws` WebSocket. Typed loosely so this file can be imported as source — the
// lock test does — where `ws` is not installed.
type Sock = any;

// `ws` is inlined by the build, so the bundle always has it. Imported as source
// it is absent, and everything that needs it lives behind main().
const wsLib: any = await import("ws").catch(() => null);
const OPEN = 1; // WebSocket.OPEN

const HERE = path.dirname(fileURLToPath(import.meta.url));
let DIST = "";

// ---- arguments
function parseArgv(argv: string[]) {
  const get = (n: string) => { const i = argv.indexOf(n); return i >= 0 ? argv[i + 1] : undefined; };
  const mode = get("--mode");
  if (mode !== undefined && mode !== "headless" && mode !== "shared") {
    console.error(`BAD_MODE: --mode must be headless or shared (got ${mode})`);
    process.exit(11);
  }
  return { file: get("--file"), sessionDir: get("--session-dir"), mode: mode as Mode | undefined };
}
const args = parseArgv(process.argv.slice(2));
const sessionDir = path.resolve(args.sessionDir ?? fs.mkdtempSync(path.join(os.tmpdir(), "xl-")));
let mode: Mode = args.mode ?? "headless";

// ---- file. One file per session: the lock and the single `file` in
// session.json are the two halves of that rule.
const EMPTY = { type: "excalidraw", version: 2, source: "excalidraw-live", elements: [], appState: { viewBackgroundColor: "#ffffff", gridSize: null }, files: {} };
const rawFile = path.resolve(args.file ?? "drawing.excalidraw");
// `.excalidraw` is the only format. Obsidian's `.excalidraw.md` and the SVG and
// PNG carriers keep the scene somewhere this server would not find or rewrite.
if (!rawFile.endsWith(".excalidraw")) {
  console.error(`BAD_FILE: only .excalidraw files are supported (got ${path.extname(rawFile) || path.basename(rawFile)})`);
  process.exit(11);
}
if (!fs.existsSync(rawFile)) fs.writeFileSync(rawFile, JSON.stringify(EMPTY, null, 2));
const file = fs.realpathSync(rawFile);
const lockPath = file + ".lock";

// ---- state
const hash = (s: string) => crypto.createHash("sha256").update(s).digest("hex");
let scene: any = JSON.parse(fs.readFileSync(file, "utf8"));
let rev = 0;                          // bumped by the engine on every save
let engine: Sock | null = null;
let enginePid: number | null = null;  // the headless Chrome we own, if any
const tabs = new Set<Sock>();
let lastWrittenHash = hash(fs.readFileSync(file, "utf8"));
let externalChangePending = false;    // cleared when reported in an apply result

// What the user pointed at with ⌘K / ⌘⇧K, kept here rather than on the
// clipboard: the tab posts it, only the id is copied, and the agent reads the
// mark back with getMark. Memory only — marks die with the session.
const marks = new MarkStore();

const token = crypto.randomBytes(32).toString("hex");
const startedAt = new Date().toISOString();
let port = 0;

const send = (ws: Sock, m: any) => ws.readyState === OPEN && ws.send(JSON.stringify(m));
const broadcast = (m: any) => tabs.forEach((ws) => send(ws, m));
const log = (...a: any[]) => console.log(new Date().toISOString(), ...a);
const sessionUrl = () => `http://127.0.0.1:${port}/#token=${token}`;

function writeSessionJson() {
  const info = { port, token, file, mode, pid: process.pid, enginePid, startedAt };
  fs.writeFileSync(path.join(sessionDir, "session.json"), JSON.stringify(info, null, 2));
}

// ---- lock
export async function acquireLock(port: number) {
  const body = JSON.stringify({ pid: process.pid, port, file, sessionDir, mode, startedAt });
  for (let attempt = 0; attempt < 2; attempt++) {
    try {
      const fh = await fsp.open(lockPath, "wx");
      await fh.writeFile(body); await fh.close();
      return;
    } catch (e: any) {
      if (e.code !== "EEXIST") throw e;
      let owner: any = {};
      try { owner = JSON.parse(await fsp.readFile(lockPath, "utf8")); } catch {}
      let alive = false;
      try { process.kill(owner.pid, 0); alive = true; } catch {}
      if (alive) {
        try {
          const r = await fetch(`http://127.0.0.1:${owner.port}/health`, { signal: AbortSignal.timeout(1500) });
          const h: any = await r.json();
          // Only a live session serving this same file owns it; anything else
          // on that port is a coincidence, and the lock is stale.
          if (h.file === file) {
            console.error(`LOCKED: ${file} is being edited by another session (pid ${owner.pid}, port ${owner.port}, mode ${h.mode ?? owner.mode ?? "unknown"})`);
            process.exit(10);
          }
        } catch {}
      }
      console.error(`stale lock (pid ${owner.pid}) replaced`);
      await fsp.rm(lockPath, { force: true });
    }
  }
  throw new Error("could not take lock");
}
export const releaseLock = () => { try { if (JSON.parse(fs.readFileSync(lockPath, "utf8")).pid === process.pid) fs.rmSync(lockPath); } catch {} };

// ---- save (atomic)
async function writeScene(s: any) {
  const out = JSON.stringify({ type: "excalidraw", version: 2, source: "excalidraw-live", elements: s.elements, appState: s.appState ?? {}, files: s.files ?? {} }, null, 2);
  lastWrittenHash = hash(out);
  const tmp = `${file}.tmp-${process.pid}`;
  await fsp.writeFile(tmp, out);
  await fsp.rename(tmp, file);
}

// ---- changes made by another program. Never reloaded silently: without the
// warning this session's next save would destroy the other program's edit.
function watchFile() {
  fs.watch(path.dirname(file), (_ev, name) => {
    if (name !== path.basename(file)) return;
    let h = ""; try { h = hash(fs.readFileSync(file, "utf8")); } catch { return; }
    if (h === lastWrittenHash) return;
    lastWrittenHash = h;
    externalChangePending = true;
    log("external change detected");
    broadcast({ t: "external-change" });
  });
}

function reloadFromDisk() {
  scene = JSON.parse(fs.readFileSync(file, "utf8"));
  lastWrittenHash = hash(fs.readFileSync(file, "utf8"));
  externalChangePending = false;
  rev++;
  if (engine) send(engine, { t: "hello", role: "engine", mode, file, scene, rev });
  for (const t of tabs) if (t !== engine) send(t, { t: "scene", rev, scene });
}

// ---- the engine: headless Chrome we own, or the user's own browser
const CHROME_CANDIDATES: Record<string, string[]> = {
  darwin: ["/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
           "/Applications/Chromium.app/Contents/MacOS/Chromium",
           "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"],
  linux:  ["google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "microsoft-edge"],
  win32:  ["C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
           "C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe",
           "C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe"],
};

function which(cmd: string): string | null {
  const r = spawnSync(process.platform === "win32" ? "where" : "which", [cmd], { encoding: "utf8" });
  const first = String(r.stdout ?? "").split(/\r?\n/).map((s) => s.trim()).find(Boolean);
  return r.status === 0 && first ? first : null;
}

export function findChrome(): string | null {
  // Used as given: a wrong $CHROME_PATH has to fail by name rather than fall
  // back to a different browser and hide the mistake.
  if (process.env.CHROME_PATH) return process.env.CHROME_PATH;
  for (const c of CHROME_CANDIDATES[process.platform] ?? []) {
    if (c.includes("/") || c.includes("\\")) { if (fs.existsSync(c)) return c; }
    else { const p = which(c); if (p) return p; }
  }
  return null;
}

// A browser that cannot be started must not take the session down with it: the
// failure arrives asynchronously, and an unhandled 'error' would be fatal.
function launch(cmd: string, argv: string[], opts: any) {
  const child = spawn(cmd, argv, opts);
  child.on("error", (e: any) => log(`could not launch ${cmd}: ${String(e?.message ?? e)}`));
  child.unref();
  return child;
}

async function startEngine(m: Mode) {
  const url = sessionUrl();
  const chrome = findChrome();
  if (m === "headless") {
    if (!chrome) {
      console.error(`NO_CHROME: install Google Chrome or set $CHROME_PATH. Tried: ${(CHROME_CANDIDATES[process.platform] ?? []).join(", ") || "nothing — unknown platform"}`);
      process.exit(12);
    }
    // detached: the headless Chrome gets its own process group, so killing
    // -pid takes its renderer children with it. The throwaway profile lives in
    // the session directory and goes when the session directory goes.
    const child = launch(chrome, ["--headless=new", "--disable-gpu", "--no-first-run",
                                  "--no-default-browser-check", "--window-size=1280,800",
                                  `--user-data-dir=${path.join(sessionDir, "chrome")}`, url],
                         { detached: true, stdio: "ignore" });
    enginePid = child.pid ?? null;
    log(`headless engine started (pid ${enginePid})`);
    writeSessionJson();
    return;
  }
  if (chrome) launch(chrome, [url], { detached: true, stdio: "ignore" });
  else {
    const opener = process.platform === "darwin" ? "open" : process.platform === "win32" ? "start" : "xdg-open";
    log(`no Chrome found; opening the default browser with ${opener}`);
    launch(opener, [url], { detached: true, stdio: "ignore", shell: process.platform === "win32" });
  }
}

function killHeadless() {
  if (enginePid === null) return;
  log(`killing the headless engine (pid ${enginePid})`);
  try {
    if (process.platform === "win32") spawnSync("taskkill", ["/pid", String(enginePid), "/T", "/F"]);
    else process.kill(-enginePid, "SIGTERM");
  } catch {}
  enginePid = null;
  writeSessionJson();
}

// ---- roles. The engine owns the scene; every other tab follows it.
async function makeEngine(ws: Sock) {
  const old = engine;
  // Hand over before switching, so nothing the outgoing engine holds is lost.
  if (old && old !== ws && old.readyState === OPEN) {
    await new Promise<void>((resolve) => {
      let done = false;
      const finish = () => { if (done) return; done = true; clearTimeout(timer); old.off("message", onMessage); resolve(); };
      const onMessage = (raw: any) => { try { if (JSON.parse(String(raw)).t === "save") finish(); } catch {} };
      const timer = setTimeout(finish, 1500);
      old.on("message", onMessage);          // the frame itself is handled by the normal path
      send(old, { t: "final-save" });
    });
  }
  engine = ws;
  send(ws, { t: "hello", role: "engine", mode, file, scene, rev });   // the live scene, not the file
  for (const t of tabs) send(t, { t: "role", role: t === ws ? "engine" : "follower" });
  failPending("ENGINE_CHANGED", "the engine changed while the call was in progress");
  // A visible tab took over from the headless Chrome we launched. Safe when the
  // old engine was itself the user's tab: enginePid is null then.
  if (old && old !== ws) killHeadless();
}

// ---- RPC relay, one at a time
type Pending = { resolve: (v: any) => void; timer: NodeJS.Timeout; ws: Sock };
const pending = new Map<string, Pending>();
let rpcSeq = 0;
let chain: Promise<any> = Promise.resolve();
function failPending(code: string, message: string) {
  for (const [id, p] of pending) { clearTimeout(p.timer); p.resolve({ ok: false, error: { code, message } }); pending.delete(id); }
}
function rpc(method: string, params: any): Promise<any> {
  const run = () => new Promise((resolve) => {
    if (!engine || engine.readyState !== OPEN) return resolve({ ok: false, error: { code: "NO_ENGINE", message: `No drawing engine is connected. Open ${sessionUrl()}` } });
    const id = `r${++rpcSeq}`;
    const ms = method === "render" ? 60000 : 15000;
    const timer = setTimeout(() => { pending.delete(id); resolve({ ok: false, error: { code: "TIMEOUT", message: `the engine did not reply in ${ms} ms` } }); }, ms);
    pending.set(id, { resolve, timer, ws: engine });
    send(engine, { t: "rpc", id, method, params });
  });
  const p = chain.then(run, run);
  chain = p;
  return p;
}

// ---- idle shutdown. A headless session nobody is watching does not outlive
// the agent that started it; a shared one never times out.
let idleTimer: NodeJS.Timeout | null = null;
function touchIdle() {
  if (idleTimer) clearTimeout(idleTimer);
  idleTimer = null;
  if (mode !== "headless") return;
  idleTimer = setTimeout(() => { log("idle for 30 minutes"); shutdown(); }, 30 * 60_000);
}

// ---- HTTP
const auth = (req: http.IncomingMessage) => req.headers.authorization === `Bearer ${token}`;
const json = (res: http.ServerResponse, code: number, body?: any) => { res.writeHead(code, { "content-type": "application/json" }); res.end(body === undefined ? "" : JSON.stringify(body)); };
const readBody = (req: http.IncomingMessage) => new Promise<any>((r, j) => { let b = ""; req.on("data", (c) => (b += c)); req.on("end", () => { try { r(b ? JSON.parse(b) : {}); } catch (e) { j(e); } }); });
const MIME: Record<string, string> = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".woff2": "font/woff2", ".ttf": "font/ttf" };

async function handle(req: http.IncomingMessage, res: http.ServerResponse) {
  const url = new URL(req.url!, "http://x");
  try {
    if (url.pathname === "/health") return json(res, 200, { ok: true, file, pid: process.pid, mode, engine: !!engine && engine.readyState === OPEN, tabs: tabs.size, rev });
    if (url.pathname === "/rpc" && req.method === "POST") {
      if (!auth(req)) return json(res, 401, { ok: false, error: { code: "UNAUTHORIZED", message: "bad token" } });
      touchIdle();
      const { method, params } = await readBody(req);
      // The mark calls are the server's own: it holds the marks, so they answer
      // without the page and keep answering after the engine has changed.
      if (method === "getMark") {
        const m = marks.get(params?.id);
        if (!m) {
          return json(res, 200, { ok: false, error: { code: "NO_MARK", message: `no mark ${JSON.stringify(params?.id ?? null)} in this session — marks live in the session's memory, so ask the user to press ⌘K again`, data: { recent: marks.recent(5) } } });
        }
        return json(res, 200, { ok: true, result: m });
      }
      if (method === "getMarks") {
        return json(res, 200, { ok: true, result: { marks: marks.recent(Number(params?.limit) || 10) } });
      }
      // reload is the server's own: it reads the file, so it works headless and
      // is the one call that does not need the page to answer.
      if (method === "reload") {
        try { reloadFromDisk(); } catch (e: any) { return json(res, 200, { ok: false, error: { code: "BAD_FILE", message: `${file} could not be read: ${String(e?.message ?? e)}` } }); }
        return json(res, 200, { ok: true, result: { rev, elements: scene?.elements?.length ?? 0 } });
      }
      const r = await rpc(method, params ?? {});
      if (method === "apply" && r.ok && externalChangePending) {
        r.result = { ...(r.result ?? {}), externalChange: true };
        externalChangePending = false;
      }
      // Nobody is watching a headless session, so a notification the page could
      // not show goes to the log rather than nowhere.
      if (method === "notify" && r.ok && r.result?.shown === false) log(`notify[${params?.level ?? "info"}] ${params?.text ?? ""}`);
      return json(res, 200, r);
    }
    // The tab's ⌘K / ⌘⇧K. The page posts what the user pointed at and copies
    // only the id it gets back, so the chat carries a handle, not a scene.
    if (url.pathname === "/mark" && req.method === "POST") {
      if (!auth(req)) return json(res, 401, { ok: false, error: { code: "UNAUTHORIZED", message: "bad token" } });
      let mark;
      try {
        mark = marks.put(validateMark(await readBody(req)));
      } catch (e: any) {
        const bad = e instanceof BadMark;
        return json(res, bad ? 400 : 500, { ok: false, error: { code: bad ? "BAD_MARK" : "INTERNAL", message: String(e?.message ?? e) } });
      }
      log(`mark ${mark.id} (${mark.kind}, rev ${mark.rev})`);
      return json(res, 200, { ok: true, id: mark.id, at: mark.at });
    }
    if (url.pathname === "/open" && req.method === "POST") {
      if (!auth(req)) return json(res, 401, { ok: false, error: { code: "UNAUTHORIZED", message: "bad token" } });
      mode = "shared";
      writeSessionJson();
      touchIdle();                       // clears the idle timer and does not re-arm it
      await startEngine("shared");
      return json(res, 200, { ok: true, mode, url: sessionUrl() });
    }
    if (url.pathname === "/shutdown" && req.method === "POST") {
      if (!auth(req)) return json(res, 401, { ok: false });
      json(res, 200, { ok: true });
      return shutdown();
    }
    // static: everything the page needs sits under <dist>, fonts included
    const p = url.pathname === "/" ? "/index.html" : url.pathname;
    const f = path.join(DIST, p);
    if (!DIST || !f.startsWith(DIST)) return json(res, 404);
    const data = await fsp.readFile(f).catch(() => null);
    if (!data) return json(res, 404);
    res.writeHead(200, { "content-type": MIME[path.extname(f)] ?? "application/octet-stream" });
    res.end(data);
  } catch (e: any) { json(res, 500, { ok: false, error: { code: "INTERNAL", message: String(e?.message ?? e) } }); }
}

let server: http.Server;
let wss: any;

function onConnection(ws: Sock) {
  tabs.add(ws);
  log(`tab connected (${tabs.size} tabs), it is the engine`);
  makeEngine(ws);
  ws.on("message", async (raw: any) => {
    const m = JSON.parse(String(raw));
    if (m.t === "rpc-result" || m.t === "rpc-error") {
      const p = pending.get(m.id); if (!p) return;
      clearTimeout(p.timer); pending.delete(m.id);
      p.resolve(m.t === "rpc-result" ? { ok: true, result: m.result } : { ok: false, error: m.error });
    } else if (m.t === "save") {
      if (ws !== engine) return;
      try {
        scene = m.scene; rev = m.rev; await writeScene(scene);
        send(ws, { t: "saved", rev });
        for (const t of tabs) if (t !== ws) send(t, { t: "scene", rev, scene });
      } catch (e: any) { send(ws, { t: "save-failed", rev: m.rev, message: String(e) }); }
    } else if (m.t === "take-over") {
      makeEngine(ws);
    } else if (m.t === "reload") {
      try { reloadFromDisk(); } catch (e: any) { log(`reload failed: ${String(e?.message ?? e)}`); }
    }
  });
  ws.on("close", () => {
    tabs.delete(ws);
    if (ws === engine) {
      engine = null;
      for (const [id, p] of pending) if (p.ws === ws) { clearTimeout(p.timer); pending.delete(id); p.resolve({ ok: false, error: { code: "ENGINE_CHANGED", message: "the engine disconnected" } }); }
      const next = [...tabs].pop(); if (next) makeEngine(next);
    }
    log(`tab disconnected (${tabs.size} tabs)`);
  });
}

function onUpgrade(req: http.IncomingMessage, sock: any, head: any) {
  const url = new URL(req.url!, "http://x");
  if (url.pathname !== "/ws") return sock.destroy();
  if (req.headers.origin !== `http://127.0.0.1:${port}`) { sock.write("HTTP/1.1 403 Forbidden\r\n\r\n"); return sock.destroy(); }
  wss.handleUpgrade(req, sock, head, (ws: Sock) => {
    if (url.searchParams.get("token") !== token) return ws.close(4401, "unauthorized");
    onConnection(ws);
  });
}

// ---- shutdown
let shuttingDown = false;
async function shutdown() {
  if (shuttingDown) return; shuttingDown = true;
  log("shutting down");
  if (idleTimer) { clearTimeout(idleTimer); idleTimer = null; }
  if (engine) {
    const saved = new Promise<void>((r) => { engine!.once("message", () => r()); setTimeout(r, 1500); });
    send(engine, { t: "final-save" }); await saved; await new Promise((r) => setTimeout(r, 200));
  }
  broadcast({ t: "shutdown" });
  killHeadless();
  releaseLock();
  setTimeout(() => process.exit(0), 100);
}

// The bundled server lives at <cache>/<buildId>/server.mjs, so its own location
// is the build entry: everything it serves sits in ./dist beside it and no
// hashing has to be repeated. Running this file unbundled from the skill folder
// (development only) has no such sibling, so it asks the build where one is.
async function resolveDist() {
  const beside = path.join(HERE, "dist");
  if (fs.existsSync(beside)) return beside;
  const buildMjs = pathToFileURL(path.join(HERE, "..", "scripts", "build.mjs")).href;
  const { resolveBuild } = await import(buildMjs);
  return path.join(resolveBuild().dir, "dist");
}

async function main() {
  fs.mkdirSync(path.join(sessionDir, "scripts"), { recursive: true });
  fs.mkdirSync(path.join(sessionDir, "renders"), { recursive: true });
  DIST = await resolveDist();
  watchFile();
  server = http.createServer(handle);
  wss = new wsLib.WebSocketServer({ noServer: true });
  server.on("upgrade", onUpgrade);
  process.on("SIGINT", shutdown);
  process.on("SIGTERM", shutdown);
  server.listen(0, "127.0.0.1", async () => {
    port = (server.address() as any).port;
    await acquireLock(port);
    writeSessionJson();
    log(`serving ${file} at ${sessionUrl()} (${mode})`);
    await startEngine(mode);
    touchIdle();
  });
}

if (import.meta.main) main();
