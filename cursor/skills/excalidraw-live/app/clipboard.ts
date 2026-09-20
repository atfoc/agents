// The two shortcuts that hand something back to the agent: ⌘K marks the
// selection, ⌘⇧K marks the spot under the cursor.
//
// Neither one puts a scene on the clipboard. The page posts what the user
// pointed at to the session server, which keeps it in memory under a short
// random id and hands that id back; only a single handle line is copied, and
// the agent reads the mark itself with `getMark`. If the post fails there is no
// id to paste, so the shortcut falls back to copying the whole block — a
// shortcut that quietly does nothing is worse than a long paste.
//
// The listener is installed on `window` in the capture phase, so it answers
// before Excalidraw's own handlers — but only for the ⌘/Ctrl combinations, so
// plain `K` stays Excalidraw's shortcut.
import { formatMarkHandle, formatPointBlock, formatSelectionBlock, nearest, type FmtElement, type FmtViewport } from "../lib/format.ts";
import type { MarkInput } from "../lib/marks.ts";
import { keyOf, type El } from "./apply.ts";

export type ClipboardDeps = {
  api: () => any;                               // ExcalidrawImperativeAPI
  file: () => string;
  rev: () => number;
  pointer: () => [number, number];              // last pointer position, in scene coordinates
  enabled: () => boolean;                       // false in headless, or when this tab is not the engine
  save: (mark: MarkInput) => Promise<string | null>;   // POST /mark — the id, or null if it did not land
  toast: (text: string) => void;
  crosshair: (p: [number, number] | null) => void;
};

/**
 * One live element as the formatter's input. The same bridge engine.ts uses for
 * RPC results, so a mark reads exactly like a scene the agent printed.
 */
const toFmt = (e: El, all: Map<string, El>): FmtElement => {
  const name = (b: any) => { const t = b && all.get(b.elementId); return t ? (keyOf(t) ?? t.id.slice(0, 6)) : "·"; };
  const bt = e.boundElements?.find((b: any) => b.type === "text");
  return {
    id: e.id, type: e.type, x: e.x, y: e.y, width: e.width, height: e.height,
    key: keyOf(e), label: e.type === "text" ? e.text : bt ? all.get(bt.id)?.text : undefined,
    isDeleted: e.isDeleted, containerId: e.containerId, frameId: e.frameId, groupIds: e.groupIds,
    ...(e.type === "arrow" ? { startRef: name(e.startBinding), endRef: name(e.endBinding) } : {}),
  };
};

const viewportOf = (api: any): FmtViewport => {
  const a = api.getAppState(); const z = a.zoom.value;
  return { x: -a.scrollX, y: -a.scrollY, width: a.width / z, height: a.height / z, zoom: z };
};

export function installShortcuts(deps: ClipboardDeps): () => void {
  const write = async (text: string, ok: string) => {
    try {
      await navigator.clipboard.writeText(text);       // text/plain only, and nothing else
      deps.toast(ok);
    } catch (e) {
      // No transient activation, or the permission was refused. Nothing was
      // copied, so say so rather than letting the user paste stale text.
      console.error("clipboard write failed:", e);
      deps.toast("Could not write the clipboard");
    }
  };

  /**
   * Store the mark, then copy its handle. The post is one request to a server
   * on this machine, well inside the few seconds a click or keypress keeps the
   * clipboard open to us, so the write still counts as user-initiated.
   */
  const keep = async (mark: MarkInput, handle: (id: string) => string, ok: (id: string) => string) => {
    const id = await deps.save(mark);
    if (!id) return write(mark.text, "Copied as text — the session did not answer, so there is no id");
    return write(handle(id), ok(id));
  };

  const copySelection = () => {
    const api = deps.api();
    const all = new Map<string, El>(api.getSceneElements().map((e: El) => [e.id, e]));
    const selected = api.getAppState().selectedElementIds;
    // Bound text belongs to its container and is already printed as its label.
    const chosen = [...all.values()].filter((e) => selected[e.id] && !(e.type === "text" && e.containerId));
    if (!chosen.length) {
      deps.toast("Nothing selected — ⌘⇧K marks the cursor position instead");
      return;
    }
    const file = deps.file(), rev = deps.rev();
    const elements = chosen.slice(0, 40).map((e) => toFmt(e, all));
    const viewport = viewportOf(api);
    const text = formatSelectionBlock({ file, rev, elements, viewport, total: chosen.length });
    const n = chosen.length;
    return keep(
      { kind: "selection", file, rev, text, elements, total: n, viewport },
      (id) => formatMarkHandle({ kind: "selection", id, file, rev, count: n }),
      (id) => `Selection marked ${id} — paste it to the agent (${n} element${n === 1 ? "" : "s"})`,
    );
  };

  const copyPoint = () => {
    const api = deps.api();
    const point = deps.pointer();
    const all = new Map<string, El>(api.getSceneElements().map((e: El) => [e.id, e]));
    const visible = [...all.values()].filter((e) => !e.isDeleted && !(e.type === "text" && e.containerId)).map((e) => toFmt(e, all));
    const near = nearest(point, visible).map((n) => n.el);
    const file = deps.file(), rev = deps.rev();
    const viewport = viewportOf(api);
    const text = formatPointBlock({ file, rev, point, near, viewport });
    deps.crosshair(point);
    return keep(
      { kind: "point", file, rev, text, point, near, viewport },
      (id) => formatMarkHandle({ kind: "point", id, file, rev, point }),
      (id) => `Point marked ${id} — paste it to the agent`,
    );
  };

  const handler = (e: KeyboardEvent) => {
    if (!(e.metaKey || e.ctrlKey) || e.key.toLowerCase() !== "k") return;
    if (!deps.enabled()) return;
    e.preventDefault();
    e.stopImmediatePropagation();
    void (e.shiftKey ? copyPoint() : copySelection());
  };

  window.addEventListener("keydown", handler, true);
  return () => window.removeEventListener("keydown", handler, true);
}
