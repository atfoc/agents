// A mark is the only thing the user's ⌘K hands the agent, so the store has to
// mint ids that do not collide, find one again from whatever the user pasted,
// refuse nonsense, and never grow without bound.
//   node --test tests/marks.test.mjs            (Node 24.x)
//   node --test --experimental-strip-types …    (Node 22.x)
import { test } from "node:test";
import assert from "node:assert/strict";

const { BadMark, MarkStore, markSummary, newMarkId, normalizeMarkId, validateMark } =
  await import("../lib/marks.ts");

const viewport = { x: 0, y: 0, width: 1400, height: 900, zoom: 1 };
const selection = (over = {}) => ({
  kind: "selection", file: "arch.excalidraw", rev: 4, text: "@excalidraw selection …",
  elements: [{ id: "k3f9aa21", type: "rectangle", x: 1, y: 2, width: 180, height: 80, key: "db", label: "Postgres" }],
  viewport, ...over,
});
const point = (over = {}) => ({
  kind: "point", file: "arch.excalidraw", rev: 4, text: "@excalidraw point …",
  point: [120, 340], near: [], viewport, ...over,
});

test("an id is the prefix and eight hex digits, and a taken one is skipped", () => {
  assert.match(newMarkId(), /^xlm_[0-9a-f]{8}$/);
  const seen = new Set();
  for (let i = 0; i < 200; i++) seen.add(newMarkId());
  assert.ok(seen.size > 195, `${seen.size} distinct ids out of 200`);

  let asked = 0;
  const id = newMarkId((c) => { asked++; return asked < 3; });   // the first two are taken
  assert.equal(asked, 3);
  assert.match(id, /^xlm_[0-9a-f]{8}$/);
});

test("the id is found in the bare id, the prefixed id, or the whole pasted line", () => {
  assert.equal(normalizeMarkId("xlm_7f3a9c2b"), "xlm_7f3a9c2b");
  assert.equal(normalizeMarkId("7f3a9c2b"), "xlm_7f3a9c2b");
  assert.equal(normalizeMarkId("XLM_7F3A9C2B"), "xlm_7f3a9c2b");
  assert.equal(normalizeMarkId("`xlm_7f3a9c2b`,"), "xlm_7f3a9c2b");
  assert.equal(
    normalizeMarkId("@excalidraw selection xlm_7f3a9c2b — 3 elements in arch.excalidraw (rev 14)"),
    "xlm_7f3a9c2b",
  );
  assert.equal(normalizeMarkId("no id here"), null);
  assert.equal(normalizeMarkId(undefined), null);
  assert.equal(normalizeMarkId(42), null);
});

test("a stored selection comes back whole, by any spelling of its id", () => {
  const store = new MarkStore();
  const mark = store.put(validateMark(selection()));
  assert.match(mark.id, /^xlm_[0-9a-f]{8}$/);
  assert.ok(Date.parse(mark.at) > 0, mark.at);

  for (const spelling of [mark.id, mark.id.slice(4), `pasted: ${mark.id}`]) {
    const got = store.get(spelling);
    assert.equal(got?.id, mark.id);
    assert.equal(got.elements[0].key, "db");
    assert.equal(got.elements[0].label, "Postgres");
    assert.equal(got.text, "@excalidraw selection …");
  }
  assert.equal(store.get("xlm_deadbeef"), undefined);
  assert.equal(store.get("nonsense"), undefined);
});

test("a point keeps its coordinates and a selection its count", () => {
  const store = new MarkStore();
  const p = store.put(validateMark(point()));
  assert.deepEqual(p.point, [120, 340]);
  assert.deepEqual(markSummary(p), { id: p.id, kind: "point", at: p.at, file: "arch.excalidraw", rev: 4, point: [120, 340] });

  const s = store.put(validateMark(selection({ total: 12 })));
  assert.equal(markSummary(s).count, 12, "the count is what was selected, not what was listed");
});

test("only the fields the formatter reads survive, and a selection is cut at forty", () => {
  const elements = Array.from({ length: 50 }, (_, i) => ({
    id: `e${i}`, type: "rectangle", x: 0, y: 0, width: 1, height: 1,
    seed: 12345, versionNonce: 999, points: [[0, 0]],          // Excalidraw noise, not ours to keep
  }));
  const m = validateMark(selection({ elements, total: 50 }));
  assert.equal(m.elements.length, 40);
  assert.equal(m.total, 50, "the total is what the user selected, not what was kept");
  assert.deepEqual(Object.keys(m.elements[0]), ["id", "type", "x", "y", "width", "height"]);
});

test("nonsense is refused rather than stored", () => {
  assert.throws(() => validateMark({ kind: "scribble" }), (e) => e instanceof BadMark && e.code === "BAD_MARK");
  assert.throws(() => validateMark({ ...point(), point: [1] }), BadMark);
  assert.throws(() => validateMark({ ...point(), point: ["a", "b"] }), BadMark);
  assert.throws(() => validateMark({ ...selection(), file: 7 }), BadMark);
  // A viewport that is missing or broken is not worth losing the mark over.
  assert.deepEqual(validateMark({ ...selection(), viewport: undefined }).viewport,
    { x: 0, y: 0, width: 0, height: 0, zoom: 1 });
});

test("the store keeps its limit, dropping the oldest first", () => {
  const store = new MarkStore(3);
  const ids = [1, 2, 3, 4, 5].map(() => store.put(validateMark(selection())).id);
  assert.equal(store.size, 3);
  assert.equal(store.get(ids[0]), undefined, "the oldest went");
  assert.equal(store.get(ids[1]), undefined);
  assert.equal(store.get(ids[4])?.id, ids[4], "the newest stayed");
  assert.deepEqual(store.recent().map((m) => m.id), [ids[4], ids[3], ids[2]], "newest first");
  assert.equal(store.recent(2).length, 2);
});
