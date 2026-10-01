// The page: the only place the scene changes. Runs Excalidraw, answers RPC
// calls relayed by the server, and saves after every change.
//
// The shell only: the canvas, the socket, saving, loading and the viewport
// loop. The RPC methods live in ./engine.ts, the scene engine in ./apply.ts,
// what the user sees in ./ui.tsx and the two shortcuts in ./clipboard.ts.
(window as any).EXCALIDRAW_ASSET_PATH = "/";
import React, { useEffect, useRef, useState } from "react";
import { createRoot } from "react-dom/client";
import { Excalidraw, restoreElements, getSceneVersion, CaptureUpdateAction } from "@excalidraw/excalidraw";
import "@excalidraw/excalidraw/index.css";
import { createEngine, type Change, type EngineDeps, type Mode, type Notice } from "./engine.ts";
import { RpcError, type El } from "./apply.ts";
import { Tab, type Rect } from "./ui.tsx";
import { installShortcuts } from "./clipboard.ts";
import type { MarkInput } from "../lib/marks.ts";

const token = new URLSearchParams(location.hash.slice(1)).get("token") ?? "";
const rid = () => Math.random().toString(36).slice(2, 12) + Math.random().toString(36).slice(2, 6);

/**
 * Hand a ⌘K / ⌘⇧K mark to the session server and take back its id. Never
 * throws: the shortcut falls back to a plain-text copy when there is no id.
 */
async function saveMark(mark: MarkInput): Promise<string | null> {
  try {
    const r = await fetch("/mark", {
      method: "POST",
      headers: { "content-type": "application/json", authorization: `Bearer ${token}` },
      body: JSON.stringify(mark),
    });
    const out = await r.json();
    return out?.ok ? String(out.id) : null;
  } catch (e) {
    console.error("could not store the mark:", e);
    return null;
  }
}

function App() {
  const [api, setApi] = useState<any>(null);
  const [role, setRole] = useState<"engine" | "follower" | "ended">("follower");
  const [mode, setMode] = useState<Mode>("shared");
  const [saved, setSaved] = useState<"saved" | "saving" | "failed">("saved");
  const [working, setWorking] = useState(false);   // the agent made an RPC call in the last few seconds
  const [external, setExternal] = useState(false);
  const [tick, setTick] = useState(0);          // bumped when the canvas is panned or zoomed, so anchored UI follows
  const [changes, setChanges] = useState<Change[]>([]);
  const [notices, setNotices] = useState<Notice[]>([]);
  const [toast, setToast] = useState<string | null>(null);
  const [crosshair, setCrosshair] = useState<[number, number] | null>(null);
  // Excalidraw's undo only answers real keystrokes, so an "Undo" button in the
  // agent UI needs the page to keep its own scene snapshot per agent change.
  const undoStack = useRef<{ changeId: string | null; elements: El[] }[]>([]);
  const ws = useRef<WebSocket | null>(null);
  // The cursor in scene coordinates, for ⌘⇧K. A ref: it changes on every mouse
  // move and nothing on the page renders from it.
  const pointer = useRef<[number, number]>([0, 0]);
  const workTimer = useRef<any>(0);
  const st = useRef({ rev: 0, lastVersion: -1, loading: false, saveTimer: 0 as any, role: "follower", file: "" });
  const deps = useRef<EngineDeps | null>(null);
  const engine = useRef<ReturnType<typeof createEngine> | null>(null);

  // ---- saving
  const doSave = () => {
    if (!api || st.current.role !== "engine") return;
    const elements = api.getSceneElementsIncludingDeleted();
    const v = getSceneVersion(elements);
    if (v === st.current.lastVersion) return;
    st.current.lastVersion = v;
    const rev = ++st.current.rev;
    const as = api.getAppState();
    ws.current?.send(JSON.stringify({ t: "save", rev, scene: { elements, appState: { viewBackgroundColor: as.viewBackgroundColor, gridSize: as.gridSize ?? null }, files: api.getFiles() } }));
    setSaved("saving");
  };
  const saveSoon = () => { clearTimeout(st.current.saveTimer); st.current.saveTimer = setTimeout(doSave, 300); };
  const saveNow = () => { clearTimeout(st.current.saveTimer); doSave(); };

  const loadScene = (scene: any) => {
    st.current.loading = true;
    const elements = restoreElements(scene?.elements ?? [], null, { repairBindings: true });
    api.updateScene({ elements, appState: { viewBackgroundColor: scene?.appState?.viewBackgroundColor ?? "#ffffff" }, captureUpdate: CaptureUpdateAction.NEVER });
    st.current.lastVersion = getSceneVersion(api.getSceneElementsIncludingDeleted());
    st.current.loading = false;
  };

  // ---- the engine: built once the canvas exists, then fed by the socket
  useEffect(() => {
    if (!api) return;
    deps.current = {
      api,
      state: st.current,
      mode,
      saveNow,
      onChange: (c) => setChanges((xs) => [...xs, { id: rid(), at: Date.now(), ...c }]),
      onNotice: (n) => setNotices((xs) => [{ id: rid(), at: Date.now(), ...n }, ...xs]),
      pushUndo: (elements, changeId) => { undoStack.current = [...undoStack.current.slice(-19), { changeId, elements }]; },
    };
    engine.current = createEngine(deps.current);
  }, [api]);

  // ---- WebSocket
  useEffect(() => {
    if (!api) return;
    const sock = new WebSocket(`ws://${location.host}/ws?token=${token}`);
    ws.current = sock;
    sock.onmessage = async (ev) => {
      const m = JSON.parse(ev.data);
      if (m.t === "hello") {
        st.current.rev = m.rev; st.current.role = m.role; setRole(m.role);
        if (m.file) st.current.file = String(m.file).split(/[\\/]/).pop()!;   // the name, not the path: it is a header the user pastes
        if (m.mode) { setMode(m.mode); if (deps.current) deps.current.mode = m.mode; }
        loadScene(m.scene);
      }
      else if (m.t === "role") { st.current.role = m.role; setRole(m.role); }
      else if (m.t === "scene" && st.current.role === "follower") { st.current.rev = m.rev; loadScene(m.scene); }
      else if (m.t === "saved") setSaved("saved");
      else if (m.t === "save-failed") { setSaved("failed"); console.error("save failed:", m.message); }
      else if (m.t === "external-change") setExternal(true);
      else if (m.t === "final-save") { st.current.lastVersion = -1; saveNow(); }
      else if (m.t === "shutdown") { st.current.role = "ended"; setRole("ended"); }
      else if (m.t === "rpc") {
        // "Agent working" is simply: a call arrived and none has since gone quiet.
        setWorking(true);
        clearTimeout(workTimer.current);
        workTimer.current = setTimeout(() => setWorking(false), 2500);
        try {
          const fn = engine.current?.methods[m.method];
          if (!fn) throw new RpcError("UNKNOWN_METHOD", `unknown method ${m.method}`);
          sock.send(JSON.stringify({ t: "rpc-result", id: m.id, result: await fn(m.params ?? {}) }));
        } catch (e: any) {
          sock.send(JSON.stringify({ t: "rpc-error", id: m.id, error: { code: e.code ?? "INTERNAL", message: e.message, data: { ...(e.data ?? {}), stack: e.code ? undefined : e.stack } } }));
        }
      }
    };
    sock.onclose = () => { if (st.current.role !== "ended") setRole("ended"); };
    return () => sock.close();
  }, [api]);

  // Anything anchored to an element on the canvas has to move when the canvas
  // moves. One rAF loop, one state bump per actual viewport change.
  useEffect(() => {
    if (!api) return;
    let raf = 0, last = "";
    const loop = () => {
      const a = api.getAppState();
      const sig = [Math.round(a.scrollX), Math.round(a.scrollY), a.zoom.value, a.offsetLeft, a.offsetTop, a.width, a.height].join(",");
      if (sig !== last) { last = sig; setTick((t) => t + 1); }
      raf = requestAnimationFrame(loop);
    };
    raf = requestAnimationFrame(loop);
    return () => cancelAnimationFrame(raf);
  }, [api]);

  // ---- ⌘K / ⌘⇧K. One listener for the life of the canvas: everything it needs
  // is read through a getter, so a promotion or a new revision needs no rewiring.
  useEffect(() => {
    if (!api) return;
    return installShortcuts({
      api: () => api,
      file: () => st.current.file,
      rev: () => st.current.rev,
      pointer: () => pointer.current,
      enabled: () => mode === "shared" && st.current.role === "engine",
      save: saveMark,
      toast: (text) => setToast(text),
      crosshair: (p) => setCrosshair(p),
    });
  }, [api, mode]);

  // Both of these say something once and then stop saying it.
  useEffect(() => { if (!toast) return; const t = setTimeout(() => setToast(null), 2500); return () => clearTimeout(t); }, [toast]);
  useEffect(() => { if (!crosshair) return; const t = setTimeout(() => setCrosshair(null), 3000); return () => clearTimeout(t); }, [crosshair]);

  // ---- what the tab UI can do. Everything here needs the live API, so it is
  // built here rather than inside the component.
  const rectOf = (ids: string[]): Rect | null => {
    if (!api || !ids.length) return null;
    const els = api.getSceneElements().filter((e: El) => ids.includes(e.id) || (e.containerId && ids.includes(e.containerId)));
    if (!els.length) return null;
    const a = api.getAppState(), z = a.zoom.value;
    let x0 = Infinity, y0 = Infinity, x1 = -Infinity, y1 = -Infinity;
    for (const e of els) { x0 = Math.min(x0, e.x); y0 = Math.min(y0, e.y); x1 = Math.max(x1, e.x + e.width); y1 = Math.max(y1, e.y + e.height); }
    return { left: (x0 + a.scrollX) * z + a.offsetLeft, top: (y0 + a.scrollY) * z + a.offsetTop, width: (x1 - x0) * z, height: (y1 - y0) * z };
  };

  const actions = {
    show: (ids: string[]) => {
      const els = api.getSceneElements().filter((e: El) => ids.includes(e.id));
      if (!els.length) return;
      api.updateScene({ appState: { selectedElementIds: Object.fromEntries(els.map((e: El) => [e.id, true])) } });
      api.scrollToContent(els, { fitToContent: false, animate: true });
    },
    undoLast: () => {
      const last = undoStack.current.pop();
      if (!last) return;
      api.updateScene({ elements: last.elements, captureUpdate: CaptureUpdateAction.IMMEDIATELY });
      saveNow();
      setChanges((xs) => xs.slice(0, -1));
    },
    dismiss: (id: string) => setChanges((xs) => xs.filter((c) => c.id !== id)),
    takeOver: () => ws.current?.send(JSON.stringify({ t: "take-over" })),
    reload: () => { ws.current?.send(JSON.stringify({ t: "reload" })); setExternal(false); },
    rectOf,
    sceneToScreen: (p: [number, number]) => {
      const a = api.getAppState(), z = a.zoom.value;
      return { left: (p[0] + a.scrollX) * z + a.offsetLeft, top: (p[1] + a.scrollY) * z + a.offsetTop };
    },
  };

  (window as any).__xl = { api, role: () => st.current.role, mode, saved, working, external, tick, changes, notices, toast, crosshair, undoStack, actions };

  return (
    <div style={{ position: "fixed", inset: 0 }}>
      <Excalidraw
        excalidrawAPI={setApi}
        viewModeEnabled={role !== "engine"}
        onChange={() => { if (!st.current.loading && st.current.role === "engine") saveSoon(); }}
        onPointerUpdate={(p: any) => { pointer.current = [p.pointer.x, p.pointer.y]; }}
      />
      {api && <Tab state={{ role, mode, saved, working, external, changes, notices, toast, crosshair, tick }} actions={actions} />}
    </div>
  );
}

createRoot(document.getElementById("root")!).render(<App />);
