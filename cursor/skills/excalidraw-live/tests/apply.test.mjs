// `apply` is the one call that changes a drawing, and every agent edit goes
// through it. None of these cases need a browser: the scene is a plain array
// and the only piece of Excalidraw that has to be real is text measuring, which
// tests/stubs/excalidraw.mjs makes deterministic instead.
//
//   node --test tests/apply.test.mjs                (Node 24.x)
//   node --test --experimental-strip-types …        (Node 22.x)
import { test } from "node:test";
import assert from "node:assert/strict";
import { register } from "node:module";

// app/apply.ts imports the real package, which needs a browser and a bundler.
// A resolve hook points that one bare specifier at the stub, so the module
// under test is the shipped source, unmodified.
const STUB = new URL("./stubs/excalidraw.mjs", import.meta.url).href;
register(
  "data:text/javascript," + encodeURIComponent(
    `export function resolve(spec, ctx, next) {
       if (spec === "@excalidraw/excalidraw") return { url: ${JSON.stringify(STUB)}, shortCircuit: true };
       return next(spec, ctx);
     }`),
  import.meta.url,
);

const { applyChanges, RpcError } = await import("../app/apply.ts");
const { CaptureUpdateAction } = await import("./stubs/excalidraw.mjs");

// ---- fixtures: ids are the keys, so a failing assertion reads like the scene

const base = (o) => ({
  x: 0, y: 0, width: 180, height: 80, angle: 0,
  strokeColor: "#1e1e1e", backgroundColor: "transparent", fillStyle: "solid",
  strokeWidth: 2, strokeStyle: "solid", roughness: 1, opacity: 100,
  groupIds: [], frameId: null, roundness: null, seed: 1, version: 1, versionNonce: 1,
  isDeleted: false, boundElements: null, locked: false, link: null, ...o,
});

const box = (key, o = {}) => base({ id: key, type: "rectangle", customData: { key }, ...o });

/** A container plus its bound, centred text, the way a real labelled box looks. */
const labelled = (key, text, o = {}) => {
  const c = box(key, o);
  const t = base({
    id: `${key}.text`, type: "text", text, originalText: text, fontSize: 20, fontFamily: 5,
    textAlign: "center", verticalAlign: "middle", containerId: c.id,
    width: text.length * 12, height: 25, boundElements: null,
  });
  t.x = c.x + (c.width - t.width) / 2;
  t.y = c.y + (c.height - t.height) / 2;
  c.boundElements = [{ id: t.id, type: "text" }];
  return [c, t];
};

const arrow = (key, from, to, o = {}) => base({
  id: key, type: "arrow", customData: { key }, points: [[0, 0], [100, 0]], width: 100, height: 0,
  startBinding: from ? { elementId: from, focus: 0, gap: 1 } : null,
  endBinding: to ? { elementId: to, focus: 0, gap: 1 } : null, ...o,
});

/** Two boxes with an arrow between them, all three wired the way the page wires them. */
const wired = () => {
  const a = box("a"), b = box("b", { x: 400 });
  const ar = arrow("ar", "a", "b");
  a.boundElements = [{ id: "ar", type: "arrow" }];
  b.boundElements = [{ id: "ar", type: "arrow" }];
  return [a, b, ar];
};

function fakeApi(elements) {
  const api = {
    elements: [...elements],
    updates: [],
    scrolled: null,
    appState: {
      viewBackgroundColor: "#ffffff", gridSize: null, selectedElementIds: {},
      zoom: { value: 1 }, scrollX: 0, scrollY: 0, width: 1280, height: 800, theme: "light",
    },
    getSceneElementsIncludingDeleted: () => api.elements,
    getSceneElements: () => api.elements.filter((e) => !e.isDeleted),
    getAppState: () => api.appState,
    getFiles: () => ({}),
    updateScene: (u) => {
      api.updates.push(u);
      if (u.elements) api.elements = u.elements;
      if (u.appState) Object.assign(api.appState, u.appState);
    },
    scrollToContent: (els) => { api.scrolled = els; },
  };
  return api;
}

function run(api, p) {
  const calls = { undo: [], changes: [], saves: 0 };
  const result = applyChanges(api, p, {
    pushUndo: (elements, changeId) => calls.undo.push({ elements, changeId }),
    onChange: (c) => calls.changes.push(c),
    saveNow: () => { calls.saves++; },
    rev: () => 42,
  });
  return { result, calls, api, at: (id) => api.elements.find((e) => e.id === id) };
}

const live = (api, type) => api.elements.filter((e) => !e.isDeleted && (!type || e.type === type));

// ---- creates

test("a create with a key reports the new id and stamps the key on the element", () => {
  const api = fakeApi([]);
  const { result, at } = run(api, { create: [{ type: "rectangle", key: "db", x: 10, y: 20, width: 180, height: 80 }] });
  const id = result.created.db;
  assert.ok(id, "created holds the key");
  assert.equal(at(id).customData.key, "db");
  assert.equal(at(id).x, 10);
});

test("a create with a key that already exists becomes an update", () => {
  const api = fakeApi([box("db")]);
  const { result, at } = run(api, { create: [{ type: "rectangle", key: "db", width: 300 }] });
  assert.deepEqual(result.created, {});
  assert.deepEqual(result.updated, ["db"]);
  assert.equal(live(api, "rectangle").length, 1, "no second element was created");
  assert.equal(at("db").width, 300);
});

test("a create with an existing key but another type is BAD_PARAMS and applies nothing", () => {
  const api = fakeApi([box("db")]);
  assert.throws(
    () => run(api, { create: [{ type: "ellipse", key: "db" }], update: [{ ref: { key: "db" }, x: 99 }] }),
    (e) => e instanceof RpcError && e.code === "BAD_PARAMS" && e.data.path === "create[0].type",
  );
  assert.equal(api.updates.length, 0, "the scene was never touched");
  assert.equal(api.elements[0].x, 0);
});

test("a create without a key is reported under its index", () => {
  const api = fakeApi([]);
  const { result } = run(api, { create: [{ type: "rectangle", x: 0, y: 0 }] });
  assert.deepEqual(Object.keys(result.created), ["#0"]);
  assert.ok(result.created["#0"]);
});

// ---- ifVersion

test("a stale ifVersion is one conflict, and the rest of the call still applies", () => {
  const api = fakeApi([box("a", { version: 5 }), box("b", { x: 400, version: 3 })]);
  const { result, at } = run(api, {
    update: [
      { ref: { key: "a" }, ifVersion: 1, x: 50 },
      { ref: { key: "b" }, ifVersion: 3, x: 70 },
    ],
  });
  assert.equal(result.conflicts.length, 1);
  assert.deepEqual(result.conflicts[0], { ref: { key: "a" }, expectedVersion: 1, actualVersion: 5 });
  assert.equal(at("a").x, 0, "the conflicting element is untouched");
  assert.equal(at("a").version, 5);
  assert.equal(at("b").x, 70);
  assert.deepEqual(result.updated, ["b"]);
});

test("ifVersion is read from the scene as it was before the call, not after an arrow bound to it", () => {
  const api = fakeApi([box("a"), box("b", { x: 400 })]);
  const { result, at } = run(api, {
    create: [{ type: "arrow", key: "ar", start: { key: "a" }, end: { key: "b" } }],
    update: [{ ref: { key: "a" }, ifVersion: 1, backgroundColor: "#eeeeee" }],
  });
  assert.deepEqual(result.conflicts, [], "binding the arrow bumped a's version, which is not a conflict");
  assert.equal(at("a").backgroundColor, "#eeeeee");
  assert.ok(at("a").version > 1);
});

test("deleting an element created by the same call never conflicts on ifVersion", () => {
  const api = fakeApi([]);
  const { result, at } = run(api, {
    create: [{ type: "rectangle", key: "tmp", x: 0, y: 0 }],
    delete: [{ ref: { key: "tmp" }, ifVersion: 7 }],
  });
  assert.deepEqual(result.conflicts, []);
  assert.deepEqual(result.deleted, [result.created.tmp]);
  assert.equal(at(result.created.tmp).isDeleted, true);
});

// ---- deletes

test("a delete without cascade keeps the arrow and nulls the binding that pointed at the element", () => {
  const api = fakeApi(wired());
  const { result, at } = run(api, { delete: [{ ref: { key: "a" } }] });
  assert.deepEqual(result.deleted, ["a"]);
  assert.equal(at("ar").isDeleted, false);
  assert.equal(at("ar").startBinding, null);
  assert.equal(at("ar").endBinding.elementId, "b");
});

test("a delete with cascade takes the arrow with it and reports it", () => {
  const api = fakeApi(wired());
  const { result, at } = run(api, { delete: [{ ref: { key: "a" }, cascade: true }] });
  assert.deepEqual(result.deleted, ["a", "ar"]);
  assert.equal(at("ar").isDeleted, true);
});

test("deleting a frame keeps its children and clears their frameId", () => {
  const api = fakeApi([
    box("f", { type: "frame", width: 600, height: 400 }),
    box("c", { frameId: "f", x: 20, y: 20 }),
  ]);
  const { result, at } = run(api, { delete: [{ ref: { key: "f" } }] });
  assert.deepEqual(result.deleted, ["f"]);
  assert.equal(at("c").isDeleted, false);
  assert.equal(at("c").frameId, null);
});

// ---- updates

test("moveBy together with x is BAD_PARAMS and applies nothing", () => {
  const api = fakeApi([box("a")]);
  assert.throws(
    () => run(api, { update: [{ ref: { key: "a" }, moveBy: [10, 10], x: 5 }] }),
    (e) => e instanceof RpcError && e.code === "BAD_PARAMS" && e.data.path === "update[0]",
  );
  assert.equal(api.updates.length, 0);
});

test("moveBy on a labelled box moves the bound text by the same delta", () => {
  const api = fakeApi(labelled("a", "Hello"));
  const before = api.elements[1];
  const { at } = run(api, { update: [{ ref: { key: "a" }, moveBy: [10, 20] }] });
  assert.equal(at("a").x, 10);
  assert.equal(at("a").y, 20);
  assert.equal(at("a.text").x, before.x + 10);
  assert.equal(at("a.text").y, before.y + 20);
});

test("a style-only update on a labelled box re-measures the label and keeps it centred", () => {
  const api = fakeApi(labelled("a", "Hello"));
  const { at } = run(api, { update: [{ ref: { key: "a" }, roughness: 2 }] });
  assert.equal(at("a").roughness, 2);
  assert.equal(at("a.text").isDeleted, true, "the old label is replaced, not edited");

  const c = at("a");
  const label = api.elements.find((e) => e.type === "text" && e.containerId === "a" && !e.isDeleted);
  assert.ok(label, "the container still has a live label");
  assert.equal(label.text, "Hello", "the text survives a style-only change");
  assert.equal(label.roughness, 2, "the style cascaded into the label");
  assert.equal(label.x + label.width / 2, c.x + c.width / 2);
  assert.equal(label.y + label.height / 2, c.y + c.height / 2);
  assert.deepEqual(c.boundElements, [{ id: label.id, type: "text" }]);
});

test("angle, locked, link and points pass through untouched", () => {
  const api = fakeApi([box("a"), arrow("ar", null, null)]);
  const { at } = run(api, {
    update: [
      { ref: { key: "a" }, angle: 0.5, locked: true, link: "https://example.com" },
      { ref: { key: "ar" }, points: [[0, 0], [40, 90]] },
    ],
  });
  assert.equal(at("a").angle, 0.5);
  assert.equal(at("a").locked, true);
  assert.equal(at("a").link, "https://example.com");
  assert.deepEqual(at("ar").points, [[0, 0], [40, 90]]);
});

// ---- the call as a whole

test("a BAD_REF anywhere fails the whole call and names where it was", () => {
  const api = fakeApi([box("a")]);
  assert.throws(
    () => run(api, { update: [{ ref: { key: "a" }, x: 10 }, { ref: { key: "nope" }, x: 20 }] }),
    (e) => e instanceof RpcError && e.code === "BAD_REF" && e.data.path === "update[1].ref",
  );
  assert.equal(api.updates.length, 0);
  assert.equal(api.elements[0].x, 0, "the first update was not applied either");
});

test("a successful call is one updateScene, captured immediately, so it is one undo step", () => {
  const api = fakeApi([box("a")]);
  const { result, calls } = run(api, { update: [{ ref: { key: "a" }, x: 60 }] });
  assert.equal(api.updates.length, 1);
  assert.equal(api.updates[0].captureUpdate, CaptureUpdateAction.IMMEDIATELY);
  assert.equal(calls.saves, 1);
  assert.equal(result.rev, 42, "the rev comes from the shell, not from apply");
  assert.equal(calls.undo.length, 1);
  assert.equal(calls.undo[0].elements[0].x, 0, "the snapshot is the scene before the call");
  assert.deepEqual(calls.changes, [{ created: 0, updated: 1, deleted: 0, ids: ["a"] }]);
});
