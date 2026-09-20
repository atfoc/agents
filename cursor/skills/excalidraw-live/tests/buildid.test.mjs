// The build id is what makes a stale bundle impossible: it is recomputed on
// every start, so anything that changes the output has to change the id, and
// anything that does not must leave it alone.
import { test } from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { spawn } from "node:child_process";
import { fileURLToPath, pathToFileURL } from "node:url";

const BUILD = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../scripts/build.mjs");
const { buildId, resolveBuild, publish, sourceFiles } = await import(pathToFileURL(BUILD).href);

const tmps = [];
function tmpdir(prefix = "xl-test-") {
  const d = fs.mkdtempSync(path.join(os.tmpdir(), prefix));
  tmps.push(d);
  return d;
}
process.on("exit", () => { for (const d of tmps) fs.rmSync(d, { recursive: true, force: true }); });

/** A miniature skill tree: the files a build depends on, and some it must not. */
function fixture(overrides = {}) {
  const root = tmpdir();
  const files = {
    "scripts/build.mjs": "// the build\n",
    "scripts/build/package.json": '{ "name": "excalidraw-live-build" }\n',
    "scripts/build/package-lock.json": '{ "lockfileVersion": 3 }\n',
    "app/index.html": "<!doctype html>\n",
    "app/main.tsx": "export const main = 1;\n",
    "app/ui.tsx": "export const ui = 1;\n",
    "app/engine.ts": "export const engine = 1;\n",
    "server/server.ts": "export const server = 1;\n",
    "SKILL.md": "# excalidraw-live\n",
    "references/api.md": "# api\n",
    "tests/format.test.mjs": "// a test\n",
    ...overrides,
  };
  for (const [rel, body] of Object.entries(files)) {
    fs.mkdirSync(path.join(root, path.dirname(rel)), { recursive: true });
    fs.writeFileSync(path.join(root, rel), body);
  }
  return root;
}
const write = (root, rel, body) => {
  fs.mkdirSync(path.join(root, path.dirname(rel)), { recursive: true });
  fs.writeFileSync(path.join(root, rel), body);
};

test("the same tree hashes to the same id", () => {
  const root = fixture();
  assert.equal(buildId(false, root), buildId(false, root));
});

test("one byte changed in app/ui.tsx is a different id", () => {
  const root = fixture();
  const before = buildId(false, root);
  write(root, "app/ui.tsx", "export const ui = 2;\n");
  assert.notEqual(buildId(false, root), before);
});

test("a rename with the same contents is a different id", () => {
  const root = fixture();
  const before = buildId(false, root);
  fs.renameSync(path.join(root, "app/ui.tsx"), path.join(root, "app/panel.tsx"));
  assert.notEqual(buildId(false, root), before);
});

test("SKILL.md, references/ and tests/ do not affect the id", () => {
  const root = fixture();
  const before = buildId(false, root);
  // A port of this skill to another harness rewrites only SKILL.md, and both
  // copies have to resolve to the same cache entry.
  write(root, "SKILL.md", "# a completely different SKILL.md\n");
  write(root, "references/api.md", "# rewritten\n");
  write(root, "references/troubleshooting.md", "# new file\n");
  write(root, "tests/format.test.mjs", "// rewritten\n");
  write(root, "tests/e2e.mjs", "// new file\n");
  assert.equal(buildId(false, root), before);
});

test("--with-cjk is the same id plus the -cjk suffix", () => {
  const root = fixture();
  assert.equal(buildId(true, root), `${buildId(false, root)}-cjk`);
});

test("node_modules in the tree is ignored", () => {
  const root = fixture();
  const before = buildId(false, root);
  write(root, "node_modules/left-pad/index.js", "module.exports = 1;\n");
  write(root, "app/node_modules/ws/package.json", '{ "name": "ws" }\n');
  assert.equal(buildId(false, root), before);
  assert.ok(!sourceFiles(root).some((f) => f.includes("node_modules/")));
});

test("$EXCALIDRAW_LIVE_DIST is used verbatim and nothing is hashed", () => {
  const dist = tmpdir();
  fs.mkdirSync(path.join(dist, "dist"), { recursive: true });
  fs.writeFileSync(path.join(dist, "dist/main.js"), "// bundle\n");
  const saved = process.env.EXCALIDRAW_LIVE_DIST;
  process.env.EXCALIDRAW_LIVE_DIST = dist;
  try {
    // The root does not exist, so any hashing at all would have to fail or lie.
    const r = resolveBuild(false, path.join(dist, "no-such-skill"));
    assert.equal(r.source, "env");
    assert.equal(r.dir, dist);
    assert.equal(r.id, null);
    assert.equal(r.present, true);
  } finally {
    if (saved === undefined) delete process.env.EXCALIDRAW_LIVE_DIST; else process.env.EXCALIDRAW_LIVE_DIST = saved;
  }
});

test("an entry with dist/ but no dist/main.js is not present", () => {
  const root = fixture();
  const cache = tmpdir();
  const saved = process.env.EXCALIDRAW_LIVE_CACHE;
  process.env.EXCALIDRAW_LIVE_CACHE = cache;
  try {
    const r = resolveBuild(false, root);
    assert.equal(r.source, "cache");
    assert.equal(r.id, buildId(false, root));
    assert.equal(r.dir, path.join(cache, "excalidraw-live", r.id));
    assert.equal(r.present, false);
    fs.mkdirSync(path.join(r.dir, "dist"), { recursive: true });
    assert.equal(resolveBuild(false, root).present, false, "a half-populated directory must not read as a build");
    fs.writeFileSync(path.join(r.dir, "dist/main.js"), "// bundle\n");
    assert.equal(resolveBuild(false, root).present, true);
  } finally {
    if (saved === undefined) delete process.env.EXCALIDRAW_LIVE_CACHE; else process.env.EXCALIDRAW_LIVE_CACHE = saved;
  }
});

test("two concurrent publishes of one id both succeed and leave one entry", async () => {
  const home = path.join(tmpdir(), "excalidraw-live");
  fs.mkdirSync(home, { recursive: true });
  const dest = path.join(home, "abc123abc123");
  const startAt = Date.now() + 500;
  // Each side materialises its own temp directory and renames it onto the same
  // id at the same moment: the loser's EEXIST must be a success, not a failure.
  const script = `
    import fs from "node:fs";
    import path from "node:path";
    import { publish } from ${JSON.stringify(pathToFileURL(BUILD).href)};
    const { XL_HOME: home, XL_DEST: dest, XL_START: startAt } = process.env;
    const tmp = path.join(home, \`.tmp-\${process.pid}-\${Math.random().toString(36).slice(2, 10)}\`);
    fs.mkdirSync(path.join(tmp, "dist"), { recursive: true });
    fs.writeFileSync(path.join(tmp, "dist/main.js"), "// bundle " + process.pid);
    while (Date.now() < Number(startAt)) {}
    publish(tmp, dest);
  `;
  const kids = await Promise.all([0, 1].map(() => new Promise((resolve) => {
    const child = spawn(process.execPath, ["--input-type=module", "-e", script], {
      encoding: "utf8",
      env: { ...process.env, XL_HOME: home, XL_DEST: dest, XL_START: String(startAt) },
    });
    let out = "";
    child.stdout.on("data", (c) => (out += c));
    child.stderr.on("data", (c) => (out += c));
    child.on("close", (code) => resolve({ code, out }));
  })));
  for (const k of kids) assert.equal(k.code, 0, k.out);

  assert.ok(fs.existsSync(path.join(dest, "dist/main.js")), "the entry was published");
  const left = fs.readdirSync(home);
  assert.deepEqual(left, ["abc123abc123"], `only the entry survives, found ${left.join(", ")}`);
  assert.ok(!left.some((n) => n.startsWith(".tmp-")), "no temp directory survives");
});
