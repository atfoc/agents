// ⌘K and ⌘⇧K are the user's half of the conversation, and what they put on the
// clipboard is now a handle rather than a scene. These cases drive the real
// listener with a fake canvas, a fake clipboard and a fake session server, so
// the keystroke, the post and the copied line are all checked without a browser.
//
//   node --test tests/clipboard.test.mjs             (Node 24.x)
//   node --test --experimental-strip-types …         (Node 22.x)
import { test } from "node:test";
import assert from "node:assert/strict";
import { register } from "node:module";

// app/clipboard.ts reaches app/apply.ts, which imports the real package; the
// same resolve hook the apply tests use points it at the stub.
const STUB = new URL("./stubs/excalidraw.mjs", import.meta.url).href;
register(
  "data:text/javascript," + encodeURIComponent(
    `export function resolve(spec, ctx, next) {
       if (spec === "@excalidraw/excalidraw") return { url: ${JSON.stringify(STUB)}, shortCircuit: true };
       return next(spec, ctx);
     }`),
  import.meta.url,
);

// The page's globals, as much of them as the shortcuts touch.
const listeners = [];
globalThis.window = {
  addEventListener: (type, fn, capture) => listeners.push({ type, fn, capture }),
  removeEventListener: (type, fn) => {
    const i = listeners.findIndex((l) => l.type === type && l.fn === fn);
    if (i >= 0) listeners.splice(i, 1);
  },
};
let clipboard = null;
// Node has a `navigator` of its own, and it is a getter — the page's one has to
// be defined over it rather than assigned.
Object.defineProperty(globalThis, "navigator", {
  configurable: true,
  value: { clipboard: { writeText: async (t) => { clipboard = t; } } },
});

const { installShortcuts } = await import("../app/clipboard.ts");

const box = (id, key, o = {}) => ({
  id, type: "rectangle", x: 100, y: 200, width: 180, height: 80,
  customData: key ? { key } : undefined, boundElements: null, isDeleted: false, ...o,
});

/** A page: one scene, one selection, one pointer, and a recording server. */
function page(o = {}) {
  const elements = o.elements ?? [box("k3f9aa21", "db")];
  const selected = o.selected ?? { k3f9aa21: true };
  const posted = [];
  const toasts = [];
  clipboard = null;
  const deps = {
    api: () => ({
      getSceneElements: () => elements,
      getAppState: () => ({ selectedElementIds: selected, zoom: { value: 1 }, scrollX: 0, scrollY: 0, width: 1400, height: 900 }),
    }),
    file: () => "arch.excalidraw",
    rev: () => 14,
    pointer: () => o.pointer ?? [1240, 380],
    enabled: () => o.enabled ?? true,
    save: async (mark) => { posted.push(mark); return o.id === undefined ? "xlm_7f3a9c2b" : o.id; },
    toast: (t) => toasts.push(t),
    crosshair: () => {},
  };
  const off = installShortcuts(deps);
  return { posted, toasts, off, clip: () => clipboard };
}

/** One keystroke, and the microtasks the handler spawns settled. */
async function press(key = "k", mods = {}) {
  let defaulted = false, stopped = false;
  const ev = {
    key, metaKey: true, shiftKey: false, ...mods,
    preventDefault: () => { defaulted = true; },
    stopImmediatePropagation: () => { stopped = true; },
  };
  for (const l of listeners) if (l.type === "keydown") l.fn(ev);
  await new Promise((r) => setTimeout(r, 0));
  return { defaulted, stopped };
}

test("⌘K posts the selection and copies only the handle", async () => {
  const p = page();
  const { defaulted, stopped } = await press();
  assert.ok(defaulted && stopped, "Excalidraw's own ⌘K must not also run");

  assert.equal(p.posted.length, 1);
  const mark = p.posted[0];
  assert.equal(mark.kind, "selection");
  assert.equal(mark.file, "arch.excalidraw");
  assert.equal(mark.rev, 14);
  assert.equal(mark.total, 1);
  assert.equal(mark.elements[0].key, "db");
  assert.ok(mark.text.startsWith("@excalidraw selection — arch.excalidraw (rev 14, 1 selected)"), mark.text);

  assert.equal(p.clip(), "@excalidraw selection xlm_7f3a9c2b — 1 element in arch.excalidraw (rev 14)");
  assert.equal(p.clip().split("\n").length, 1, "one line, whatever was selected");
  assert.ok(p.toasts[0].includes("xlm_7f3a9c2b"), p.toasts[0]);
  p.off();
});

test("⌘⇧K posts the cursor position and copies its handle", async () => {
  const p = page();
  await press("k", { shiftKey: true });
  const mark = p.posted[0];
  assert.equal(mark.kind, "point");
  assert.deepEqual(mark.point, [1240, 380]);
  assert.ok(mark.text.startsWith("@excalidraw point — arch.excalidraw (rev 14)"), mark.text);
  assert.equal(p.clip(), "@excalidraw point xlm_7f3a9c2b — (1240,380) in arch.excalidraw (rev 14)");
  p.off();
});

test("bound label text is not copied as an element of its own", async () => {
  const p = page({
    elements: [box("k3f9aa21", "db"), box("txt22222", null, { type: "text", containerId: "k3f9aa21", text: "Postgres" })],
    selected: { k3f9aa21: true, txt22222: true },
  });
  await press();
  assert.equal(p.posted[0].elements.length, 1);
  assert.equal(p.posted[0].total, 1);
  p.off();
});

test("with nothing selected nothing is posted and nothing is copied", async () => {
  const p = page({ selected: {} });
  await press();
  assert.equal(p.posted.length, 0);
  assert.equal(p.clip(), null);
  assert.match(p.toasts[0], /Nothing selected/);
  p.off();
});

test("a session that cannot store the mark falls back to the whole block", async () => {
  const p = page({ id: null });
  await press();
  assert.equal(p.posted.length, 1, "it was offered to the server first");
  assert.ok(p.clip().startsWith("@excalidraw selection — arch.excalidraw"), p.clip());
  assert.ok(p.clip().includes("key=db"), "the fallback is the block the agent used to get");
  assert.match(p.toasts[0], /no id/);
  p.off();
});

test("a headless page, a follower tab and a plain K are all left alone", async () => {
  const off = page({ enabled: false }).off;
  const p = page({ enabled: false });
  assert.deepEqual((await press()).defaulted, false);
  assert.equal(p.posted.length, 0);
  assert.equal(p.clip(), null);
  p.off(); off();

  const q = page();
  await press("k", { metaKey: false });          // plain K stays Excalidraw's
  await press("j");                              // ⌘J is not ours either
  assert.equal(q.posted.length, 0);
  q.off();
  assert.equal(listeners.length, 0, "every listener was removed again");
});
