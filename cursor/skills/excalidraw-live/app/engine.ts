// The RPC methods the page answers. One object of plain functions over the
// live Excalidraw API, so the shell in main.tsx only has to dispatch into it.
import { exportToBlob, exportToSvg } from "@excalidraw/excalidraw";
import type { FmtElement } from "../lib/format.ts";
import { applyChanges, keyOf, matches, type El, type Ref } from "./apply.ts";

export type Mode = "headless" | "shared";

export type Change = { id: string; at: number; created: number; updated: number; deleted: number; ids: string[] };
export type Notice = { id: string; at: number; text: string; level: "info" | "done" | "question" | "error" };

export type ElementSummary = {
  id: string; key?: string; type: string; label?: string;
  bounds: { x: number; y: number; width: number; height: number };
  start?: Ref; end?: Ref; groupIds: string[]; frameId?: string | null; version: number;
};

export type EngineDeps = {
  api: any;                                        // ExcalidrawImperativeAPI
  state: { rev: number; role: string };
  mode: Mode;                                      // read on every call: a session can be promoted
  saveNow: () => void;
  onChange: (c: Omit<Change, "id" | "at">) => void;   // feeds the change markers
  onNotice: (n: Omit<Notice, "id" | "at">) => void;   // feeds the notify log
  pushUndo: (elements: El[], changeId: string | null) => void;
};

export function createEngine(deps: EngineDeps) {
  const api = () => deps.api;

  const summarize = (e: El, all: Map<string, El>): ElementSummary => {
    const bt = e.boundElements?.find((b: any) => b.type === "text");
    const label = e.type === "text" ? e.text : bt ? all.get(bt.id)?.text : undefined;
    const ref = (b: any) => { const t = b && all.get(b.elementId); return t ? (keyOf(t) ? { key: keyOf(t)! } : { id: t.id }) : undefined; };
    return { id: e.id, key: keyOf(e), type: e.type, label, bounds: { x: e.x, y: e.y, width: e.width, height: e.height }, start: ref(e.startBinding), end: ref(e.endBinding), groupIds: e.groupIds, frameId: e.frameId, version: e.version };
  };

  /** The bridge to lib/format.ts: one live element as the formatter's input. */
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

  const methods: Record<string, (p: any) => any | Promise<any>> = {
    getScene: (p) => {
      const all = api().getSceneElementsIncludingDeleted();
      const elements = p.includeDeleted ? all : all.filter((e: El) => !e.isDeleted);
      const keys: Record<string, string> = {};
      for (const e of elements) if (keyOf(e) && !e.isDeleted) keys[keyOf(e)!] = e.id;
      const as = api().getAppState();
      return { elements, appState: { viewBackgroundColor: as.viewBackgroundColor, gridSize: as.gridSize ?? null }, files: p.includeFiles ? api().getFiles() : undefined, keys, rev: deps.state.rev };
    },
    getSelection: () => {
      const all = new Map<string, El>(api().getSceneElements().map((e: El) => [e.id, e]));
      const sel = api().getAppState().selectedElementIds;
      return { elements: [...all.values()].filter((e) => sel[e.id] && !(e.type === "text" && e.containerId)).map((e) => summarize(e, all)) };
    },
    getViewport: () => {
      const a = api().getAppState(); const z = a.zoom.value;
      return { bounds: { x: -a.scrollX, y: -a.scrollY, width: a.width / z, height: a.height / z }, zoom: z, theme: a.theme };
    },
    apply: (p) => applyChanges(api(), p, {
      pushUndo: deps.pushUndo,
      onChange: deps.onChange,
      saveNow: deps.saveNow,
      rev: () => deps.state.rev,
    }),
    render: async (p) => {
      const all = api().getSceneElements();
      const els = p.refs ? all.filter((e: El) => p.refs.some((r: Ref) => matches(e, r)) || (e.containerId && p.refs.some((r: Ref) => matches(all.find((c: El) => c.id === e.containerId), r)))) : all;
      const appState = { ...api().getAppState(), exportBackground: p.background ?? true, exportWithDarkMode: !!p.dark, exportScale: p.scale ?? 2 };
      if (p.format === "svg") {
        const svg = await exportToSvg({ elements: els, appState, files: api().getFiles(), exportPadding: p.padding ?? 20 });
        return { mime: "image/svg+xml", data: svg.outerHTML, width: +svg.getAttribute("width")!, height: +svg.getAttribute("height")! };
      }
      const blob = await exportToBlob({ elements: els, appState, files: api().getFiles(), mimeType: "image/png", exportPadding: p.padding ?? 20 });
      const bytes = new Uint8Array(await blob.arrayBuffer());
      let bin = ""; for (let i = 0; i < bytes.length; i += 0x8000) bin += String.fromCharCode(...bytes.subarray(i, i + 0x8000));
      const bmp = await createImageBitmap(blob);
      return { mime: "image/png", data: btoa(bin), width: bmp.width, height: bmp.height };
    },
    focus: (p) => {
      const els = api().getSceneElements().filter((e: El) => p.refs.some((r: Ref) => matches(e, r)));
      if (p.select) api().updateScene({ appState: { selectedElementIds: Object.fromEntries(els.map((e: El) => [e.id, true])) } });
      api().scrollToContent(els, { fitToContent: p.zoom === "fit", animate: false });
      return {};
    },
    // The notice is always kept; `shown` says whether anyone could have seen it.
    // Headless has no one looking, and the server logs the text instead.
    notify: (p) => {
      const level = p.level ?? "info";
      deps.onNotice({ text: p.text, level });
      if (deps.mode !== "shared") console.log(`[${level}]`, p.text);
      return { shown: deps.mode === "shared" };
    },
  };

  return { methods, summarize, toFmt };
}
