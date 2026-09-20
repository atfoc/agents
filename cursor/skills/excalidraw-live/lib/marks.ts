// The mark store: what ⌘K and ⌘⇧K hand back.
//
// The shortcuts do not put a scene on the clipboard any more. The tab posts
// what the user pointed at to the session server, the server keeps it in memory
// under a short random id, and only that id is copied. The agent reads the mark
// back through `getMark`, so the chat carries a handle instead of a wall of
// element lines, and what the user meant is recorded at the moment they meant
// it rather than re-derived later from a changed scene.
//
// Marks live in the server's memory and die with the session: an id from a
// stopped session resolves to nothing, and the user simply presses ⌘K again.
//
// No imports beyond the formatter's types, no DOM, and erasable TypeScript
// only — this module is bundled into both the page and the server, and read as
// source by the tests.
import type { FmtElement, FmtViewport } from "./format.ts";

export type MarkKind = "selection" | "point";

/** What the tab posts to `POST /mark`. */
export type MarkInput = {
  kind: MarkKind;
  file: string;                       // the drawing's name, as the block header prints it
  rev: number;
  text: string;                       // the block the formatter produced, ready to read
  viewport: FmtViewport;
  elements?: FmtElement[];            // selection
  total?: number;                     // how many were selected, before the 40-element cut
  point?: [number, number];           // point
  near?: FmtElement[];                // point: what it landed beside
};

export type Mark = MarkInput & { id: string; at: string };

/** One line per mark, for listing what the user has copied this session. */
export type MarkSummary = {
  id: string; kind: MarkKind; at: string; file: string; rev: number;
  count?: number; point?: [number, number];
};

export const MARK_PREFIX = "xlm_";
export const MARK_LIMIT = 200;                  // marks kept per session, oldest dropped first
const ID_RE = new RegExp(`(?:${MARK_PREFIX})?([0-9a-f]{8})`, "i");

const hex = (n: number) => Array.from({ length: n }, () => "0123456789abcdef"[Math.floor(Math.random() * 16)]).join("");

/** A fresh id, checked against whatever already holds ids. */
export function newMarkId(taken: (id: string) => boolean = () => false): string {
  for (let i = 0; i < 100; i++) {
    const id = MARK_PREFIX + hex(8);
    if (!taken(id)) return id;
  }
  throw new Error("could not mint a free mark id");
}

/**
 * The id inside whatever the user pasted. Generous on purpose: the agent may
 * pass the bare id, the id with its prefix, or the whole line it arrived on,
 * in any case and with any punctuation around it.
 */
export function normalizeMarkId(raw: unknown): string | null {
  const m = typeof raw === "string" ? raw.match(ID_RE) : null;
  return m ? MARK_PREFIX + m[1].toLowerCase() : null;
}

export class BadMark extends Error {
  code = "BAD_MARK";
}

const str = (v: unknown, field: string, max: number): string => {
  if (typeof v !== "string") throw new BadMark(`${field} must be a string`);
  return v.slice(0, max);
};

/** Only the fields the formatter reads, so a mark cannot carry a whole scene. */
const pickElement = (e: any): FmtElement => ({
  id: String(e?.id ?? ""), type: String(e?.type ?? "unknown"),
  x: Number(e?.x) || 0, y: Number(e?.y) || 0,
  width: Number(e?.width) || 0, height: Number(e?.height) || 0,
  ...(e?.key ? { key: String(e.key) } : {}),
  ...(e?.label !== undefined && e?.label !== null ? { label: String(e.label) } : {}),
  ...(e?.containerId ? { containerId: String(e.containerId) } : {}),
  ...(e?.frameId ? { frameId: String(e.frameId) } : {}),
  ...(Array.isArray(e?.groupIds) ? { groupIds: e.groupIds.map(String) } : {}),
  ...(e?.startRef ? { startRef: String(e.startRef) } : {}),
  ...(e?.endRef ? { endRef: String(e.endRef) } : {}),
});

const pickViewport = (v: any): FmtViewport => ({
  x: Number(v?.x) || 0, y: Number(v?.y) || 0,
  width: Number(v?.width) || 0, height: Number(v?.height) || 0,
  zoom: Number(v?.zoom) || 1,
});

const list = (v: unknown, cap: number): FmtElement[] =>
  (Array.isArray(v) ? v : []).slice(0, cap).map(pickElement);

/** A posted body, cut down to a storable mark. Throws `BadMark` on nonsense. */
export function validateMark(body: any): MarkInput {
  if (body?.kind !== "selection" && body?.kind !== "point") {
    throw new BadMark(`kind must be "selection" or "point" (got ${JSON.stringify(body?.kind)})`);
  }
  const base = {
    kind: body.kind as MarkKind,
    file: str(body.file ?? "", "file", 300),
    rev: Number(body.rev) || 0,
    text: str(body.text ?? "", "text", 20000),
    viewport: pickViewport(body.viewport),
  };
  if (body.kind === "selection") {
    const elements = list(body.elements, 40);
    return { ...base, elements, total: Number(body.total) || elements.length };
  }
  const p = body.point;
  if (!Array.isArray(p) || p.length !== 2 || !p.every((n: unknown) => Number.isFinite(n))) {
    throw new BadMark("point must be [x, y]");
  }
  return { ...base, point: [Number(p[0]), Number(p[1])], near: list(body.near, 3) };
}

export function markSummary(m: Mark): MarkSummary {
  return {
    id: m.id, kind: m.kind, at: m.at, file: m.file, rev: m.rev,
    ...(m.kind === "selection" ? { count: m.total ?? m.elements?.length ?? 0 } : { point: m.point }),
  };
}

/**
 * The session's marks, newest last, capped. A Map preserves insertion order,
 * so the oldest entry is the first key and eviction is one delete.
 */
export class MarkStore {
  private marks = new Map<string, Mark>();
  private limit: number;
  constructor(limit = MARK_LIMIT) { this.limit = limit; }

  put(input: MarkInput): Mark {
    const mark: Mark = { ...input, id: newMarkId((id) => this.marks.has(id)), at: new Date().toISOString() };
    this.marks.set(mark.id, mark);
    while (this.marks.size > this.limit) this.marks.delete(this.marks.keys().next().value!);
    return mark;
  }

  get(raw: unknown): Mark | undefined {
    const id = normalizeMarkId(raw);
    return id ? this.marks.get(id) : undefined;
  }

  /** The most recent marks, newest first. */
  recent(limit = 10): MarkSummary[] {
    return [...this.marks.values()].slice(-Math.max(1, limit)).reverse().map(markSummary);
  }

  get size() { return this.marks.size; }
}
