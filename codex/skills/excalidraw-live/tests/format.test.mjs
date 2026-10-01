// The scene formatter is the one thing an agent reads a drawing through, so the
// columns have to line up exactly and the order has to be the same every time.
// Node's own test runner, because the skill folder carries no dependencies:
//   node --test tests/format.test.mjs            (Node 24.x)
//   node --test --experimental-strip-types …     (Node 22.x)
import { test } from "node:test";
import assert from "node:assert/strict";

const {
  formatElement, formatScene, formatViewport, formatSelectionBlock, formatPointBlock, formatMarkHandle, nearest,
} = await import("../lib/format.ts");

/** A plain element; every field the formatter reads has a value. */
const el = (o) => ({ id: "00000000-0000", type: "rectangle", x: 0, y: 0, width: 180, height: 80, ...o });

test("a rectangle with a key and a label prints at the fixed column widths", () => {
  const line = formatElement(el({ id: "k3f9aa21-b7c2-4e11", key: "db", label: "Postgres", x: 480, y: 100 }));
  assert.equal(line, `rectangle  key=db          id=k3f9aa21  "Postgres"        (480,100 180×80)`);
  assert.equal(line, line.trimEnd(), "no trailing whitespace survives");
});

test("an element with no key leaves the column blank and the id still starts at the same offset", () => {
  const withKey = formatElement(el({ id: "k3f9aa21-b7c2-4e11", key: "db", label: "Postgres", x: 480, y: 100 }));
  const without = formatElement(el({ id: "k3f9aa21-b7c2-4e11", label: "Postgres", x: 480, y: 100 }));
  assert.equal(without.indexOf("id="), withKey.indexOf("id="));
  assert.match(without, /^rectangle {18}id=k3f9aa21/);
});

test("an arrow bound at both ends prints the link column", () => {
  const line = formatElement(el({ id: "a1b2c3d4-ee", type: "arrow", startRef: "api", endRef: "db" }));
  assert.ok(line.endsWith("api → db"), line);
});

test("an arrow bound at one end prints · for the missing side", () => {
  const start = formatElement(el({ id: "a1b2c3d4-ee", type: "arrow", startRef: "api" }));
  const end = formatElement(el({ id: "a1b2c3d4-ee", type: "arrow", endRef: "db" }));
  assert.ok(start.endsWith("api → ·"), start);
  assert.ok(end.endsWith("· → db"), end);
});

test("a text element bound to a container is not printed on its own", () => {
  const out = formatScene([
    el({ id: "box11111", label: "Postgres" }),
    el({ id: "txt22222", type: "text", containerId: "box11111", label: "Postgres" }),
  ]);
  assert.equal(out.split("\n").length, 1);
  assert.ok(!out.includes("txt22222"), out);
});

test("a deleted element is not printed", () => {
  const out = formatScene([el({ id: "box11111" }), el({ id: "gone2222", isDeleted: true })]);
  assert.ok(!out.includes("gone2222"), out);
  assert.equal(out.split("\n").length, 1);
});

test("a frame's children are indented two spaces directly under it", () => {
  const out = formatScene([
    el({ id: "kid11111", frameId: "frm00001", label: "one" }),
    el({ id: "frm00001", type: "frame", width: 600, height: 400 }),
    el({ id: "kid22222", frameId: "frm00001", label: "two" }),
  ]).split("\n");
  assert.equal(out.length, 3);
  assert.match(out[0], /^frame /);
  assert.ok(out[1].startsWith("  rectangle"), out[1]);
  assert.ok(out[2].startsWith("  rectangle"), out[2]);
  assert.ok(out[1].includes("kid11111") && out[2].includes("kid22222"));
  assert.equal(out[1].slice(2), formatElement(el({ id: "kid11111", frameId: "frm00001", label: "one" })));
});

test("an element in both a frame and a group is printed once, under the frame", () => {
  const out = formatScene([
    el({ id: "frm00001", type: "frame", width: 600, height: 400 }),
    el({ id: "kid11111", frameId: "frm00001", groupIds: ["grp00001"] }),
    el({ id: "loose111", groupIds: ["grp00001"] }),
  ]).split("\n");
  assert.equal(out.filter((l) => l.includes("kid11111")).length, 1);
  assert.equal(out[1].trimStart(), formatElement(el({ id: "kid11111", frameId: "frm00001", groupIds: ["grp00001"] })));
  assert.ok(out.some((l) => l.startsWith("group      grp00001  (1 elements)")), out.join("\n"));
});

const viewport = { x: 320, y: 40, width: 1400, height: 900, zoom: 1 };

test("a selection over forty elements is truncated with a pointer to the scene", () => {
  const elements = Array.from({ length: 45 }, (_, i) => el({ id: `e${String(i).padStart(7, "0")}` }));
  const lines = formatSelectionBlock({ file: "diagram.excalidraw", rev: 7, elements, viewport }).split("\n");
  assert.equal(lines[0], "@excalidraw selection — diagram.excalidraw (rev 7, 45 selected)");
  assert.equal(lines.filter((l) => l.startsWith("rectangle")).length, 40);
  assert.equal(lines[41], "… and 5 more selected — read the scene for the rest");
  assert.equal(lines[42], formatViewport(viewport));
  assert.equal(lines.length, 43);
});

test("a point block lists the three nearest elements and drops the far one", () => {
  const near = [
    el({ id: "far00001", x: 0, y: 960, width: 180, height: 80 }),       // 820px left
    el({ id: "mid00001", x: 700, y: 960, width: 180, height: 80 }),     // 120px left
    el({ id: "close001", x: 940, y: 1100, width: 180, height: 80 }),    // 100px below
    el({ id: "near0001", x: 600, y: 960, width: 180, height: 80 }),     // 220px left
  ];
  const lines = formatPointBlock({ file: "diagram.excalidraw", rev: 7, point: [1000, 1000], near, viewport }).split("\n");
  assert.equal(lines[0], "@excalidraw point — diagram.excalidraw (rev 7)");
  assert.equal(lines[1], "point      (1000,1000)");
  const nears = lines.filter((l) => l.startsWith("near "));
  assert.equal(nears.length, 3);
  assert.ok(nears[0].includes("close001") && nears[1].includes("mid00001") && nears[2].includes("near0001"), nears.join("\n"));
  assert.ok(!lines.some((l) => l.includes("far00001")), lines.join("\n"));
  assert.ok(nears[0].endsWith("— 100px below"), nears[0]);
  assert.equal(lines[lines.length - 1], formatViewport(viewport));
});

test("nearest reads directions off the dominant axis and honours the radius", () => {
  const point = [1000, 1000];
  const left = el({ id: "left0001", x: 520, y: 960, width: 180, height: 80 });   // right edge 300px left
  const upLeft = el({ id: "upleft01", x: 720, y: 720, width: 180, height: 180 }); // 100px left, 100px up
  const far = el({ id: "far00001", x: 420, y: 960, width: 180, height: 80 });     // 400px left

  const all = nearest(point, [left, upLeft, far]);
  assert.deepEqual(all.map((n) => n.el.id), ["upleft01", "left0001"]);
  assert.equal(all.find((n) => n.el.id === "left0001").direction, "left");
  assert.equal(all.find((n) => n.el.id === "left0001").distance, 300);
  assert.equal(all.find((n) => n.el.id === "upleft01").direction, "above-left");
  assert.ok(!all.some((n) => n.el.id === "far00001"), "400px away is outside the default radius");

  assert.equal(nearest(point, [far], 500)[0].direction, "left");
  assert.equal(nearest(point, [left, upLeft], 300, 1).length, 1);
});

test("the viewport line", () => {
  assert.equal(formatViewport(viewport), "viewport   (320,40 1400×900) zoom 1");
  assert.equal(formatViewport({ ...viewport, zoom: 1.2345 }), "viewport   (320,40 1400×900) zoom 1.23");
});

test("the handle line names the mark, what is in it and where it came from", () => {
  assert.equal(
    formatMarkHandle({ kind: "selection", id: "xlm_7f3a9c2b", file: "arch.excalidraw", rev: 14, count: 3 }),
    "@excalidraw selection xlm_7f3a9c2b — 3 elements in arch.excalidraw (rev 14)",
  );
  assert.equal(
    formatMarkHandle({ kind: "point", id: "xlm_7f3a9c2b", file: "arch.excalidraw", rev: 14, point: [1240.4, 380.6] }),
    "@excalidraw point xlm_7f3a9c2b — (1240,381) in arch.excalidraw (rev 14)",
  );
});

test("one selected element is not pluralised, and the line stays a single line", () => {
  const line = formatMarkHandle({ kind: "selection", id: "xlm_00000001", file: "a.excalidraw", rev: 1, count: 1 });
  assert.ok(line.includes("1 element in"), line);
  assert.equal(line.split("\n").length, 1);
});

test("a handle starts with the same @excalidraw marker the blocks do", () => {
  // The skill triggers on the paste, so the first two words have to be stable.
  for (const kind of ["selection", "point"]) {
    const line = formatMarkHandle({ kind, id: "xlm_abcdef12", file: "a.excalidraw", rev: 2, count: 0, point: [0, 0] });
    assert.ok(line.startsWith(`@excalidraw ${kind} xlm_abcdef12 `), line);
  }
});
