// The lock is what makes "one session per drawing" true, and its only other
// job is to get out of the way when the session behind it is gone. Both halves
// are checked here without ever starting a server: server.ts keeps its startup
// behind `import.meta.main`, so a driver can import it and call the two lock
// functions on their own.
//
// Each case runs in its own child process, because a refused lock exits the
// process that took it — an exit code is the thing under test.
import { test } from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import http from "node:http";
import os from "node:os";
import path from "node:path";
import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";

const HERE = path.dirname(fileURLToPath(import.meta.url));
const SERVER = path.join(HERE, "../server/server.ts");

const tmps = [];
function tmpdir() {
  const d = fs.mkdtempSync(path.join(os.tmpdir(), "xl-lock-"));
  tmps.push(d);
  return d;
}
process.on("exit", () => { for (const d of tmps) fs.rmSync(d, { recursive: true, force: true }); });

/** A drawing, a session directory, and the lock path the server will use. */
function fixture() {
  const dir = tmpdir();
  const file = path.join(dir, "drawing.excalidraw");
  fs.writeFileSync(file, JSON.stringify({ type: "excalidraw", version: 2, elements: [], appState: {}, files: {} }));
  const sessionDir = path.join(dir, "session");
  fs.mkdirSync(sessionDir);
  return { dir, file: fs.realpathSync(file), sessionDir, lockPath: fs.realpathSync(file) + ".lock" };
}

/**
 * Run one lock call the way the server would: same module, same arguments,
 * nothing else started.
 */
function drive(fx, body, { mode } = {}) {
  const driver = path.join(fx.dir, `driver-${Math.random().toString(36).slice(2, 8)}.mjs`);
  fs.writeFileSync(driver, `import { acquireLock, releaseLock } from ${JSON.stringify(SERVER)};\n${body}\n`);
  const argv = [driver, "--file", fx.file, "--session-dir", fx.sessionDir, ...(mode ? ["--mode", mode] : [])];
  return new Promise((resolve) => {
    const child = spawn(process.execPath, argv, { cwd: fx.dir });
    let stdout = "", stderr = "";
    child.stdout.on("data", (c) => (stdout += c));
    child.stderr.on("data", (c) => (stderr += c));
    child.on("close", (code) => resolve({ code, stdout, stderr, pid: child.pid }));
  });
}

/** A stand-in for a session that is alive and answering. */
async function health(body) {
  const server = http.createServer((req, res) => {
    res.writeHead(200, { "content-type": "application/json" });
    res.end(JSON.stringify(req.url === "/health" ? body : {}));
  });
  await new Promise((r) => server.listen(0, "127.0.0.1", r));
  const port = server.address().port;
  return { port, close: () => new Promise((r) => server.close(r)) };
}

/** A pid that is certainly not running any more. */
async function deadPid() {
  return new Promise((resolve) => {
    const child = spawn(process.execPath, ["-e", ""]);
    child.on("close", () => resolve(child.pid));
  });
}

const writeLock = (fx, owner) => fs.writeFileSync(fx.lockPath, JSON.stringify(owner));

test("with no lock file the lock is taken and says who took it", async () => {
  const fx = fixture();
  const r = await drive(fx, `await acquireLock(4321); console.log(JSON.stringify({ pid: process.pid }));`, { mode: "shared" });
  assert.equal(r.code, 0, r.stderr);
  const child = JSON.parse(r.stdout).pid;
  const owner = JSON.parse(fs.readFileSync(fx.lockPath, "utf8"));
  assert.equal(owner.pid, child);
  assert.equal(owner.port, 4321);
  assert.equal(owner.file, fx.file);
  assert.equal(owner.sessionDir, fx.sessionDir);
  assert.equal(owner.mode, "shared");
  assert.ok(!Number.isNaN(Date.parse(owner.startedAt)), `startedAt parses, got ${owner.startedAt}`);
});

test("a live session on the same file refuses the second one by name", async () => {
  const fx = fixture();
  const h = await health({ ok: true, file: fx.file, pid: process.pid, mode: "headless" });
  try {
    writeLock(fx, { pid: process.pid, port: h.port, file: fx.file, sessionDir: fx.sessionDir, mode: "headless", startedAt: new Date().toISOString() });
    const r = await drive(fx, `await acquireLock(4321); console.log("took it");`);
    assert.equal(r.code, 10, `expected exit 10, got ${r.code}: ${r.stdout}${r.stderr}`);
    assert.match(r.stderr, /LOCKED/);
    assert.match(r.stderr, new RegExp(`pid ${process.pid}\\b`), r.stderr);
    assert.match(r.stderr, new RegExp(`port ${h.port}\\b`), r.stderr);
    assert.match(r.stderr, /mode headless/, r.stderr);
    // The refusal leaves the owner's lock exactly as it found it.
    assert.equal(JSON.parse(fs.readFileSync(fx.lockPath, "utf8")).pid, process.pid);
  } finally { await h.close(); }
});

test("a lock whose pid is dead is replaced and startup continues", async () => {
  const fx = fixture();
  const gone = await deadPid();
  writeLock(fx, { pid: gone, port: 1, file: fx.file, sessionDir: fx.sessionDir, mode: "headless", startedAt: new Date().toISOString() });
  const r = await drive(fx, `await acquireLock(4321); console.log(JSON.stringify({ pid: process.pid }));`);
  assert.equal(r.code, 0, r.stderr);
  assert.match(r.stderr, /stale lock/);
  assert.equal(JSON.parse(fs.readFileSync(fx.lockPath, "utf8")).pid, JSON.parse(r.stdout).pid);
});

test("a live pid whose session serves another file is a stale lock", async () => {
  const fx = fixture();
  // The pid is alive and the port answers — but it is editing something else,
  // so the lock is a leftover whose pid has been reused.
  const h = await health({ ok: true, file: path.join(fx.dir, "other.excalidraw"), pid: process.pid, mode: "shared" });
  try {
    writeLock(fx, { pid: process.pid, port: h.port, file: fx.file, sessionDir: fx.sessionDir, mode: "shared", startedAt: new Date().toISOString() });
    const r = await drive(fx, `await acquireLock(4321); console.log(JSON.stringify({ pid: process.pid }));`);
    assert.equal(r.code, 0, `expected the lock to be replaced, got ${r.code}: ${r.stderr}`);
    assert.match(r.stderr, /stale lock/);
    assert.equal(JSON.parse(fs.readFileSync(fx.lockPath, "utf8")).pid, JSON.parse(r.stdout).pid);
  } finally { await h.close(); }
});

test("releaseLock leaves someone else's lock alone", async () => {
  const fx = fixture();
  const owner = { pid: process.pid, port: 4321, file: fx.file, sessionDir: fx.sessionDir, mode: "headless", startedAt: new Date().toISOString() };
  writeLock(fx, owner);
  const r = await drive(fx, `releaseLock(); console.log("done");`);
  assert.equal(r.code, 0, r.stderr);
  assert.ok(fs.existsSync(fx.lockPath), "the lock file survives");
  assert.deepEqual(JSON.parse(fs.readFileSync(fx.lockPath, "utf8")), owner);
});
