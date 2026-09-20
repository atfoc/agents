// Agent-side library (design.md §7). No dependencies, erasable-only TypeScript,
// so session scripts run with plain `node script.ts` on Node 22.18+ / 23.6+.
import fs from "node:fs";
import path from "node:path";
import { formatScene, type FmtElement } from "./format.ts";

export type Mode = "headless" | "shared";
export type Ref = { id: string } | { key: string };
export type El = any;
export type Target = Ref | El | { label: string };

/** What a create call hands back: enough to place the next element against it. */
export type Handle = {
  key?: string; id?: string; type: string;
  x: number; y: number; width: number; height: number;
};

export type ShapeOptions = {
  key?: string; at?: [number, number]; x?: number; y?: number;
  width?: number; height?: number; gap?: number;
  rightOf?: Target; leftOf?: Target; below?: Target; above?: Target;
  groupWith?: Target[];
  [style: string]: any;              // plus any style field from an update item
};
export type FrameOptions = ShapeOptions & { children?: Target[] };

export type ApplyResult = {
  created: Record<string, string>;
  updated: string[];
  deleted: string[];
  conflicts: { ref: Ref; expectedVersion: number; actualVersion: number }[];
  externalChange?: boolean;          // another program changed the file — tell the user
  rev: number;
};

export class RpcFailure extends Error {
  code: string; data: unknown;
  constructor(code: string, message: string, data?: unknown) { super(`${code}: ${message}`); this.code = code; this.data = data; }
}

export function session(dir?: string) {
  const d = dir ?? process.env.EXCALIDRAW_SESSION_DIR;
  if (!d) throw new Error("set EXCALIDRAW_SESSION_DIR or pass the session dir");
  return { dir: d, ...JSON.parse(fs.readFileSync(path.join(d, "session.json"), "utf8")) };
}

/**
 * One POST to the session server, with the bearer token. Every endpoint answers
 * `{ ok: true, ... }` or `{ ok: false, error }`, so the failure check lives here
 * and both `/rpc` and `/open` get it.
 */
export async function post(s: { port: number; token: string }, p: string, body: unknown = {}) {
  const r = await fetch(`http://127.0.0.1:${s.port}${p}`, {
    method: "POST",
    headers: { "content-type": "application/json", authorization: `Bearer ${s.token}` },
    body: JSON.stringify(body),
  });
  const out: any = await r.json();
  if (!out?.ok) {
    const e = out?.error ?? {};
    throw new RpcFailure(e.code ?? `HTTP_${r.status}`, e.message ?? `POST ${p} failed`, e.data);
  }
  return out;
}

export async function call(s: { port: number; token: string }, method: string, params: unknown = {}) {
  return (await post(s, "/rpc", { method, params })).result;
}

export async function connect(dir?: string) {
  const s = session(dir);
  const d = new Drawing(s);
  await d.refresh();
  return d;
}

export class Drawing {
  s: any;
  readonly file: string;
  readonly mode: Mode;
  elements: El[] = [];
  rev = 0;
  private pending = { create: [] as any[], update: [] as any[], delete: [] as any[] };
  constructor(s: any) { this.s = s; this.file = s.file; this.mode = s.mode; }

  async refresh() {
    const sc = await call(this.s, "getScene");
    this.elements = sc.elements; this.rev = sc.rev;
  }

  /**
   * Re-read the file from disk, discarding what is in the session. The server
   * handles `reload` itself, so this works headless too. The library never does
   * it on its own: `externalChange` is reported, and the user decides.
   */
  async reload(): Promise<void> {
    await call(this.s, "reload");
    await this.refresh();
  }

  /** Promote a headless session to shared: a visible tab, same session, same scene. */
  async open(): Promise<{ mode: Mode; url: string }> {
    const r = await post(this.s, "/open", {});
    (this as { mode: Mode }).mode = r.mode as Mode;
    return { mode: r.mode as Mode, url: r.url as string };
  }

  // ---- read
  private byId(id: string) { return this.elements.find((e) => e.id === id); }
  labelOf(e: El): string | undefined {
    if (e.type === "text") return e.text;
    const bt = e.boundElements?.find((b: any) => b.type === "text");
    return bt ? this.byId(bt.id)?.text : undefined;
  }
  private top() { return this.elements.filter((e) => !e.isDeleted && !(e.type === "text" && e.containerId)); }
  findAll(q: { type?: string; label?: string | RegExp; key?: string } = {}) {
    return this.top().filter((e) =>
      (!q.type || e.type === q.type) &&
      (!q.key || e.customData?.key === q.key) &&
      (!q.label || (q.label instanceof RegExp ? q.label.test(this.labelOf(e) ?? "") : this.labelOf(e) === q.label)));
  }
  find(q: { type?: string; label?: string | RegExp; key?: string }) {
    const m = this.findAll(q)[0];
    if (!m) throw new Error(`no element matches ${JSON.stringify(q)}`);
    return m;
  }
  get(r: Ref) { return "id" in r ? this.byId(r.id) : this.top().find((e) => e.customData?.key === r.key); }
  connections(t: Target) {
    const e = this.resolve(t);
    const arrows = this.top().filter((a) => a.type === "arrow");
    return {
      incoming: arrows.filter((a) => a.endBinding?.elementId === e.id),
      outgoing: arrows.filter((a) => a.startBinding?.elementId === e.id),
    };
  }
  /** The scene as text, through the one formatter the page prints with too. */
  summary(): string {
    const name = (id?: string) => { const e = id && this.byId(id); return e ? (e.customData?.key ?? e.id.slice(0, 6)) : "·"; };
    const fmt: FmtElement[] = this.elements.map((e) => ({
      id: e.id, type: e.type, x: e.x, y: e.y, width: e.width, height: e.height,
      key: e.customData?.key, label: this.labelOf(e), isDeleted: e.isDeleted,
      containerId: e.containerId, frameId: e.frameId, groupIds: e.groupIds,
      ...(e.type === "arrow"
        ? { startRef: name(e.startBinding?.elementId), endRef: name(e.endBinding?.elementId) }
        : {}),
    }));
    return formatScene(fmt);
  }
  selection() { return call(this.s, "getSelection").then((r: any) => r.elements); }
  viewport() { return call(this.s, "getViewport"); }

  // ---- change (collected, sent on commit)
  private ref(t: Target): Ref {
    const o = t as any;
    if (o.customData?.key) return { key: o.customData.key };
    if (o.id) return { id: o.id };
    if (o.key) return { key: o.key };
    if (typeof o.label === "string") return { id: this.find({ label: o.label }).id };
    throw new Error(`cannot refer to ${JSON.stringify(t)}`);
  }
  private resolve(t: Target) {
    const r = this.ref(t);
    const e = this.get(r) ?? this.pending.create.find((c) => "key" in r && c.key === r.key);
    if (!e) throw new Error(`unknown element ${JSON.stringify(r)}`);
    return e;
  }
  /**
   * Where a new w×h element goes. `rightOf`/`below` measure from the anchor's
   * far edge, `leftOf`/`above` from the new element's own — so `leftOf` undoes
   * `rightOf` with the same gap, and `above` undoes `below`.
   */
  private place(o: any, w: number, h: number) {
    const gap = o.gap ?? 60;
    if (o.rightOf) { const a = this.resolve(o.rightOf); return { x: a.x + (a.width ?? 180) + gap, y: a.y }; }
    if (o.leftOf) { const a = this.resolve(o.leftOf); return { x: a.x - gap - w, y: a.y }; }
    if (o.below) { const a = this.resolve(o.below); return { x: a.x, y: a.y + (a.height ?? 80) + gap }; }
    if (o.above) { const a = this.resolve(o.above); return { x: a.x, y: a.y - gap - h }; }
    const [ax, ay] = o.at ?? [];
    return { x: o.x ?? ax ?? 0, y: o.y ?? ay ?? 0 };
  }
  shape(type: "rectangle" | "ellipse" | "diamond", label: string, o: ShapeOptions = {}): Handle {
    const { key, at, x, y, width, height, gap, rightOf, leftOf, below, above, ...style } = o;
    const w = width ?? 180, h = height ?? 80;
    const item = { type, key, ...this.place(o, w, h), width: w, height: h, label: { text: label }, ...style };
    this.pending.create.push(item);
    return { key, type, x: item.x, y: item.y, width: w, height: h };
  }
  rect(label: string, o: ShapeOptions = {}) { return this.shape("rectangle", label, o); }
  ellipse(label: string, o: ShapeOptions = {}) { return this.shape("ellipse", label, o); }
  diamond(label: string, o: ShapeOptions = {}) { return this.shape("diamond", label, o); }
  /**
   * A frame, titled and with its children moved into it. Given children and no
   * geometry of its own, the frame is the box around them — a frame that does
   * not contain what it holds is a drawing bug, not a placement choice.
   */
  frame(title: string, o: FrameOptions = {}): Handle {
    const { key, at, x, y, width, height, gap, rightOf, leftOf, below, above, children, ...style } = o;
    const kids = (children ?? []).map((t) => this.ref(t));
    const positioned = at !== undefined || x !== undefined || y !== undefined || rightOf || leftOf || below || above;
    const pad = gap ?? 40;
    let box: { x: number; y: number; width: number; height: number } | undefined;
    if (children?.length && !positioned) {
      const bs = children.map((t) => this.resolve(t));
      const x0 = Math.min(...bs.map((b) => b.x)), y0 = Math.min(...bs.map((b) => b.y));
      const x1 = Math.max(...bs.map((b) => b.x + (b.width ?? 180))), y1 = Math.max(...bs.map((b) => b.y + (b.height ?? 80)));
      box = { x: x0 - pad, y: y0 - pad, width: x1 - x0 + pad * 2, height: y1 - y0 + pad * 2 };
    }
    const w = width ?? box?.width ?? 400, h = height ?? box?.height ?? 300;
    const pos = box ? { x: box.x, y: box.y } : this.place(o, w, h);
    const item = { type: "frame", key, name: title, ...pos, width: w, height: h, children: kids, ...style };
    this.pending.create.push(item);
    return { key, type: "frame", x: item.x, y: item.y, width: w, height: h };
  }
  text(t: string, o: any = {}) { const { key, ...rest } = o; this.pending.create.push({ type: "text", key, text: t, x: 0, y: 0, ...rest }); return { key } as any; }
  arrow(from: Target, to: Target, o: any = {}) {
    const { key, label, ...style } = o;
    this.pending.create.push({ type: "arrow", key, start: this.ref(from), end: this.ref(to), ...(label ? { label: { text: label } } : {}), ...style });
    return { key } as any;
  }
  update(t: Target, patch: any) {
    const r = this.ref(t); const e = this.get(r);
    this.pending.update.push({ ref: r, ...(e ? { ifVersion: e.version } : {}), ...patch });
  }
  move(t: Target, o: { by?: [number, number]; to?: [number, number] }) {
    this.update(t, o.by ? { moveBy: o.by } : { x: o.to![0], y: o.to![1] });
  }
  remove(t: Target, o: { cascade?: boolean } = {}) {
    const r = this.ref(t); const e = this.get(r);
    if (!e) return; // already gone: removing is idempotent
    this.pending.delete.push({ ref: r, ifVersion: e.version, ...o });
  }
  row(items: Target[], o: { gap?: number; at?: [number, number] } = {}) {
    let [x, y] = o.at ?? [this.resolve(items[0]).x, this.resolve(items[0]).y];
    for (const t of items) { const e = this.resolve(t); this.update(t, { x, y }); x += (e.width ?? 180) + (o.gap ?? 60); }
  }
  async commit(extra: { select?: Ref[]; focus?: boolean } = {}): Promise<ApplyResult> {
    const res = await call(this.s, "apply", { ...this.pending, ...extra });
    this.pending = { create: [], update: [], delete: [] };
    await this.refresh();
    return res as ApplyResult;
  }

  // ---- other
  focus(ts: Target[], o: { select?: boolean; zoom?: "fit" | number } = {}) { return call(this.s, "focus", { refs: ts.map((t) => this.ref(t)), ...o }); }
  notify(text: string, level = "info") { return call(this.s, "notify", { text, level }); }
  async render(file: string, o: any = {}) {
    const svg = file.endsWith(".svg");
    const r = await call(this.s, "render", { format: svg ? "svg" : "png", ...o });
    fs.mkdirSync(path.dirname(file), { recursive: true });
    fs.writeFileSync(file, svg ? r.data : Buffer.from(r.data, "base64"));
    return { file, width: r.width, height: r.height };
  }
}
