// The two shortcuts that hand something back to the agent: ⌘K copies the
// selection, ⌘⇧K copies the spot under the cursor. Both write one plain-text
// block, in the same format the agent reads a scene in, for the user to paste
// into the chat. The page never *reads* the clipboard.
//
// The listener is installed on `window` in the capture phase, so it answers
// before Excalidraw's own handlers — but only for the ⌘/Ctrl combinations, so
// plain `K` stays Excalidraw's shortcut.
import { formatPointBlock, formatSelectionBlock, nearest, type FmtElement, type FmtViewport } from "../lib/format.ts";
import { keyOf, type El } from "./apply.ts";

export type ClipboardDeps = {
  api: () => any;                               // ExcalidrawImperativeAPI
  file: () => string;
  rev: () => number;
  pointer: () => [number, number];              // last pointer position, in scene coordinates
  enabled: () => boolean;                       // false in headless, or when this tab is not the engine
  toast: (text: string) => void;
  crosshair: (p: [number, number] | null) => void;
};

/**
 * One live element as the formatter's input. The same bridge engine.ts uses for
 * RPC results, so a pasted block reads exactly like a scene the agent printed.
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

  const copySelection = () => {
    const api = deps.api();
    const all = new Map<string, El>(api.getSceneElements().map((e: El) => [e.id, e]));
    const selected = api.getAppState().selectedElementIds;
    // Bound text belongs to its container and is already printed as its label.
    const chosen = [...all.values()].filter((e) => selected[e.id] && !(e.type === "text" && e.containerId));
    if (!chosen.length) {
      deps.toast("Nothing selected — ⌘⇧K copies the cursor position instead");
      return;
    }
    const text = formatSelectionBlock({
      file: deps.file(), rev: deps.rev(),
      elements: chosen.slice(0, 40).map((e) => toFmt(e, all)),
      viewport: viewportOf(api), total: chosen.length,
    });
    return write(text, `Selection copied — paste it to the agent (${chosen.length} element${chosen.length === 1 ? "" : "s"})`);
  };

  const copyPoint = () => {
    const api = deps.api();
    const point = deps.pointer();
    const all = new Map<string, El>(api.getSceneElements().map((e: El) => [e.id, e]));
    const visible = [...all.values()].filter((e) => !e.isDeleted && !(e.type === "text" && e.containerId)).map((e) => toFmt(e, all));
    const text = formatPointBlock({
      file: deps.file(), rev: deps.rev(), point,
      near: nearest(point, visible).map((n) => n.el),
      viewport: viewportOf(api),
    });
    deps.crosshair(point);
    return write(text, "Point copied — paste it to the agent");
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
