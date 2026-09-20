// A stand-in for @excalidraw/excalidraw, so app/apply.ts can be unit-tested in
// Node. The skill folder carries no node_modules and the real package needs a
// browser; what apply.ts actually uses of it is three things, and only the text
// measuring has to be plausible — it just has to be deterministic.
//
// tests/apply.test.mjs redirects the bare specifier here with a resolve hook.

let n = 0;
const nid = () => `gen${String(++n).padStart(4, "0")}`;

/** Reset ids between tests so expectations can be written against them. */
export function __resetIds() { n = 0; }

const PAD = 5;
const measure = (text, fontSize = 20) => ({
  width: Math.max(1, [...String(text)].length * fontSize * 0.6),
  height: fontSize * 1.25,
});

export const CaptureUpdateAction = { IMMEDIATELY: "IMMEDIATELY", NEVER: "NEVER", EVENTUALLY: "EVENTUALLY" };

/** The real one returns a clone with `version` bumped; that is all apply.ts relies on. */
export function newElementWith(element, updates) {
  return { ...element, ...updates, version: (element.version ?? 1) + 1, versionNonce: (element.versionNonce ?? 1) + 1 };
}

const isRef = (v) => !!v && typeof v === "object" && v.id !== undefined && v.type === undefined;

export function convertToExcalidrawElements(skeletons, opts = {}) {
  const regen = opts.regenerateIds !== false;
  const out = [];
  const map = new Map();
  const bindings = [];

  const convertOne = (s) => {
    const { label, start, end, children, ...rest } = s;
    const el = {
      x: 0, y: 0, width: 100, height: 100, angle: 0,
      strokeColor: "#1e1e1e", backgroundColor: "transparent", fillStyle: "solid",
      strokeWidth: 2, strokeStyle: "solid", roughness: 1, opacity: 100,
      groupIds: [], frameId: null, roundness: null, seed: 1, version: 1, versionNonce: 1,
      isDeleted: false, boundElements: null, updated: 1, link: null, locked: false,
      ...rest,
      id: regen ? nid() : (s.id ?? nid()),
    };
    if (el.type === "text") {
      el.fontSize = el.fontSize ?? 20;
      el.fontFamily = el.fontFamily ?? 5;
      el.textAlign = el.textAlign ?? "left";
      el.verticalAlign = el.verticalAlign ?? "top";
      el.containerId = el.containerId ?? null;
      el.originalText = el.text;
      Object.assign(el, measure(el.text ?? "", el.fontSize));
    }
    if ((el.type === "arrow" || el.type === "line") && Array.isArray(el.points)) {
      el.width = Math.max(...el.points.map((q) => Math.abs(q[0])));
      el.height = Math.max(...el.points.map((q) => Math.abs(q[1])));
    }
    out.push(el); map.set(el.id, el);

    if (label) {
      const fontSize = label.fontSize ?? 20;
      const t = {
        type: "text", id: nid(), text: label.text, originalText: label.text, fontSize,
        fontFamily: label.fontFamily ?? 5,
        strokeColor: label.strokeColor ?? el.strokeColor,
        opacity: el.opacity, roughness: el.roughness,
        textAlign: label.textAlign ?? "center", verticalAlign: label.verticalAlign ?? "middle",
        containerId: el.id, angle: 0, groupIds: [], frameId: el.frameId ?? null,
        version: 1, versionNonce: 1, isDeleted: false, boundElements: null,
        locked: false, link: null, x: 0, y: 0, ...measure(label.text, fontSize),
      };
      if (el.type === "arrow" || el.type === "line") {
        const last = (el.points ?? [[0, 0], [0, 0]]).at(-1);
        t.x = el.x + last[0] / 2 - t.width / 2;
        t.y = el.y + last[1] / 2 - t.height / 2;
      } else {
        // a label the container cannot hold grows the container, then centres in it
        el.width = Math.max(el.width, t.width + PAD * 2);
        el.height = Math.max(el.height, t.height + PAD * 2);
        t.x = el.x + (el.width - t.width) / 2;
        t.y = el.y + (el.height - t.height) / 2;
      }
      el.boundElements = [...(el.boundElements ?? []), { id: t.id, type: "text" }];
      out.push(t); map.set(t.id, t);
    }
    return el;
  };

  for (const s of skeletons) {
    const sides = {};
    for (const side of ["start", "end"]) {
      const v = s[side];
      if (!v) continue;
      sides[side] = isRef(v) ? v.id : convertOne(v).id;
    }
    const el = convertOne(s);
    if (sides.start || sides.end) bindings.push({ el, ...sides });
  }

  for (const b of bindings) {
    for (const side of ["start", "end"]) {
      const target = map.get(b[side]);
      if (!target) continue;
      b.el[`${side}Binding`] = { elementId: target.id, focus: 0, gap: 1 };
      target.boundElements = [...(target.boundElements ?? []), { id: b.el.id, type: "arrow" }];
    }
  }
  return out;
}
