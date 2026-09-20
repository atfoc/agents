#!/usr/bin/env node
// Builds the skill's runtime bundle into a per-user cache, once, on first use.
//
// The skill folder is source only — no node_modules, no build output. Everything
// the browser and the session server need is produced here into
//   <cacheRoot>/excalidraw-live/<buildId>/
// and every run after the first starts from that directory.
//
//   node scripts/build.mjs [--with-cjk] [--force] [--offline] [--watch]
//
// The build id is a content hash of the sources, so an edit to any of them
// rebuilds automatically and a stale artefact cannot be served.
import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

export const SKILL_DIR = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");

/** Where cache entries live: $EXCALIDRAW_LIVE_CACHE, else the platform cache dir. */
export function cacheRoot() {
  const env = process.env.EXCALIDRAW_LIVE_CACHE;
  if (env) return path.resolve(env);
  if (process.env.XDG_CACHE_HOME) return path.resolve(process.env.XDG_CACHE_HOME);
  if (process.platform === "darwin") return path.join(os.homedir(), "Library", "Caches");
  if (process.platform === "win32") return process.env.LOCALAPPDATA || path.join(os.homedir(), "AppData", "Local");
  return path.join(os.homedir(), ".cache");
}

/**
 * The files the build output depends on, relative to the skill root, sorted.
 * `lib/` is in here because both bundles reach into it — the page through the
 * shortcuts, the server through the mark store — so an edit there has to change
 * the id like any other source. SKILL.md, references/ and tests/ are
 * deliberately absent: a port of this skill to another harness rewrites only
 * SKILL.md and must resolve to the same entry. node_modules is never walked.
 */
export function sourceFiles(root = SKILL_DIR) {
  const list = ["scripts/build/package-lock.json", "scripts/build.mjs", "app/index.html", "server/server.ts"];
  for (const dir of ["app", "lib"]) {
    const full = path.join(root, dir);
    if (!fs.existsSync(full)) continue;
    for (const name of fs.readdirSync(full)) {
      if (/\.(ts|tsx)$/.test(name) && fs.statSync(path.join(full, name)).isFile()) list.push(`${dir}/${name}`);
    }
  }
  return [...new Set(list)].filter((rel) => fs.existsSync(path.join(root, rel))).sort();
}

/** Per-file sha256, recorded in .stamp.json so a surprising rebuild can be explained. */
export function sourceHashes(root = SKILL_DIR) {
  const out = {};
  for (const rel of sourceFiles(root)) out[rel] = createHash("sha256").update(fs.readFileSync(path.join(root, rel))).digest("hex");
  return out;
}

/** Ten small files, one hash — cheap enough to run on every start. */
export function buildId(withCjk = false, root = SKILL_DIR) {
  const h = createHash("sha256");
  for (const rel of sourceFiles(root)) {
    h.update(rel); h.update("\0");                                  // the path is hashed too, so a rename is a new id
    h.update(fs.readFileSync(path.join(root, rel))); h.update("\0");
  }
  return h.digest("hex").slice(0, 12) + (withCjk ? "-cjk" : "");
}

/** The one place that knows where a build lives. */
export function resolveBuild(withCjk = false, root = SKILL_DIR) {
  const env = process.env.EXCALIDRAW_LIVE_DIST;
  // Used verbatim and never rebuilt into: a wrong path must fail by name rather
  // than fall back to the cache and hide the mistake.
  if (env) return { dir: env, id: null, source: "env", present: fs.existsSync(path.join(env, "dist/main.js")) };
  const id = buildId(withCjk, root);
  const dir = path.join(cacheRoot(), "excalidraw-live", id);
  return { dir, id, source: "cache", present: fs.existsSync(path.join(dir, "dist/main.js")) };
}

/**
 * Publish a finished temp directory as the cache entry for an id. One
 * filesystem, so the rename is atomic; two sessions racing both succeed,
 * because the loser's EEXIST means the winner's entry is already complete.
 */
export function publish(tmp, dest) {
  fs.mkdirSync(path.dirname(dest), { recursive: true });
  try {
    fs.renameSync(tmp, dest);
    return true;
  } catch (e) {
    if (e.code !== "EEXIST" && e.code !== "ENOTEMPTY") throw e;
    fs.rmSync(tmp, { recursive: true, force: true });
    return false;
  }
}

const alive = (pid) => { try { process.kill(pid, 0); return true; } catch (e) { return e.code === "EPERM"; } };

/** Drop abandoned temp directories and all but the two most recent entries. */
export function sweep(root, keep = 2) {
  let names = [];
  try { names = fs.readdirSync(root); } catch { return; }
  const entries = [];
  for (const name of names) {
    const full = path.join(root, name);
    if (name.startsWith(".tmp-")) {
      const pid = Number(name.split("-")[1]);
      // A live build owns its temp directory; only abandoned ones are swept.
      if (!Number.isFinite(pid) || !alive(pid)) fs.rmSync(full, { recursive: true, force: true });
      continue;
    }
    try { entries.push({ full, mtime: fs.statSync(full).mtimeMs }); } catch {}
  }
  entries.sort((a, b) => b.mtime - a.mtime);
  for (const e of entries.slice(keep)) fs.rmSync(e.full, { recursive: true, force: true });
}

function fail(message, log = "") {
  const tail = log.trim().split("\n").slice(-25).join("\n");
  console.error(`BUILD_FAILED: ${message}`);
  if (tail) console.error(tail);
  console.error("If npm cannot reach the registry, set a proxy with `npm config set proxy <url>`, or build on a connected machine and point $EXCALIDRAW_LIVE_DIST at the copied build directory.");
  process.exit(1);
}

function run(cmd, argv, opts = {}) {
  const r = spawnSync(cmd, argv, { encoding: "utf8", ...opts });
  const out = `${r.stdout ?? ""}${r.stderr ?? ""}`;
  if (r.error) return { code: 1, out: `${out}\n${r.error.message}` };
  return { code: r.status ?? 1, out };
}

export function build({ withCjk = false, force = false, offline = false, watch = false, root = SKILL_DIR } = {}) {
  const started = Date.now();
  const id = buildId(withCjk, root);
  const home = path.join(cacheRoot(), "excalidraw-live");
  const dest = path.join(home, id);

  if (watch) return watchPage(root, dest, id);
  if (fs.existsSync(path.join(dest, "dist/main.js")) && !force) return { buildId: id, dir: dest, built: false };

  const tmp = path.join(home, `.tmp-${process.pid}-${Math.random().toString(36).slice(2, 10)}`);
  fs.rmSync(tmp, { recursive: true, force: true });
  fs.mkdirSync(tmp, { recursive: true });

  try {
    for (const f of ["package.json", "package-lock.json"]) fs.copyFileSync(path.join(root, "scripts/build", f), path.join(tmp, f));

    const npm = run(process.platform === "win32" ? "npm.cmd" : "npm", ["ci", "--prefer-offline", ...(offline ? ["--offline"] : [])], { cwd: tmp });
    if (npm.code !== 0) fail("npm ci could not install the build's dependencies", npm.out);

    const modules = path.join(tmp, "node_modules");
    const esbuild = path.join(modules, ".bin", process.platform === "win32" ? "esbuild.cmd" : "esbuild");
    // The sources stay in the skill folder while the dependencies live in the
    // cache, so esbuild is told where to resolve bare imports from. Its CLI
    // reads node paths from the environment.
    const env = { ...process.env, NODE_PATH: modules };

    // 1. the page
    const page = run(esbuild, [
      path.join(root, "app/main.tsx"), "--bundle", "--format=esm", `--outdir=${path.join(tmp, "dist")}`,
      "--conditions=production", `--define:process.env.NODE_ENV="production"`, `--define:process.env.IS_PREACT="false"`,
      "--minify", "--loader:.woff2=file", "--jsx=automatic", "--log-level=warning",
    ], { cwd: tmp, env });
    if (page.code !== 0) fail("esbuild could not bundle the page", page.out);

    // 2. the server, with ws inlined. ws is CommonJS, so the ESM bundle needs a
    // real `require` for the node builtins it reaches for; the banner supplies
    // one. Its two optional native helpers are asked for inside a try/catch, so
    // leaving them external lets them fail harmlessly at runtime.
    const srv = run(esbuild, [
      path.join(root, "server/server.ts"), "--bundle", "--platform=node", "--target=node22", "--format=esm",
      `--outfile=${path.join(tmp, "server.mjs")}`, "--external:bufferutil", "--external:utf-8-validate",
      `--banner:js=import{createRequire as __cr}from"node:module";var require=__cr(import.meta.url);`,
      "--log-level=warning",
    ], { cwd: tmp, env });
    if (srv.code !== 0) fail("esbuild could not bundle the server", srv.out);

    // 3. the scene fonts, so a session never reaches the network
    const fonts = path.join(modules, "@excalidraw/excalidraw/dist/prod/fonts");
    fs.cpSync(fonts, path.join(tmp, "dist/fonts"), { recursive: true });
    if (!withCjk) fs.rmSync(path.join(tmp, "dist/fonts/Xiaolai"), { recursive: true, force: true });

    // 4. the page's own entry point
    fs.copyFileSync(path.join(root, "app/index.html"), path.join(tmp, "dist/index.html"));

    fs.writeFileSync(path.join(tmp, ".stamp.json"), JSON.stringify({ buildId: id, builtAt: new Date().toISOString(), node: process.version, cjk: withCjk, sources: sourceHashes(root) }, null, 2));

    // A cheap guard that the sources are loadable.
    const guard = path.join(root, "tests/format.test.mjs");
    const t = run(process.execPath, [guard], { cwd: root });
    if (t.code !== 0) fail("tests/format.test.mjs failed against these sources", t.out);

    if (force) fs.rmSync(dest, { recursive: true, force: true });
    publish(tmp, dest);
  } finally {
    fs.rmSync(tmp, { recursive: true, force: true });
  }

  sweep(home);
  return { buildId: id, dir: dest, built: true, ms: Date.now() - started };
}

/** Development only: rebuild the page in place against a live shared session. */
function watchPage(root, dest, id) {
  if (!fs.existsSync(path.join(dest, "node_modules"))) fail(`no build to watch at ${dest} — run the build once without --watch first`);
  const modules = path.join(dest, "node_modules");
  const esbuild = path.join(modules, ".bin", process.platform === "win32" ? "esbuild.cmd" : "esbuild");
  console.error(JSON.stringify({ buildId: id, dir: dest, watching: true }));
  const r = spawnSync(esbuild, [
    path.join(root, "app/main.tsx"), "--bundle", "--format=esm", `--outdir=${path.join(dest, "dist")}`,
    "--conditions=production", `--define:process.env.NODE_ENV="production"`, `--define:process.env.IS_PREACT="false"`,
    "--minify", "--loader:.woff2=file", "--jsx=automatic", "--log-level=warning", "--watch",
  ], { stdio: "inherit", env: { ...process.env, NODE_PATH: modules } });
  process.exit(r.status ?? 0);
}

if (process.argv[1] && import.meta.url === pathToFileURL(fs.realpathSync(process.argv[1])).href) {
  const argv = process.argv.slice(2);
  const out = build({
    withCjk: argv.includes("--with-cjk"),
    force: argv.includes("--force"),
    offline: argv.includes("--offline"),
    watch: argv.includes("--watch"),
  });
  console.log(JSON.stringify(out));
}
