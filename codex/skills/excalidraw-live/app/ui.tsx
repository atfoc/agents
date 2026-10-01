// What the user sees on top of the drawing in a shared session: the smallest
// amount of chrome that says what the agent is doing and lets the user hand a
// selection or a point back to it.
//
// One component and one only. The status pill is the only permanent piece;
// everything else appears when it has something to say and leaves when it does
// not. Anything anchored to the canvas is re-placed from `state.tick`, so it
// stays on its elements while the user pans and zooms.
import React, { useEffect, useState } from "react";
import type { Change, Notice } from "./engine.ts";

/** A rectangle in page (screen) pixels — where some elements are right now. */
export type Rect = { left: number; top: number; width: number; height: number };

export type TabState = {
  role: "engine" | "follower" | "ended";
  mode: "headless" | "shared";
  saved: "saved" | "saving" | "failed";
  working: boolean;
  external: boolean;
  changes: Change[];
  notices: Notice[];
  toast: string | null;
  crosshair: [number, number] | null;
  tick: number;
};

export type TabActions = {
  show: (ids: string[]) => void;          // select the elements and scroll to them
  undoLast: () => void;                   // undo the agent's last change
  dismiss: (id: string) => void;          // stop marking one change
  takeOver: () => void;                   // become the engine tab
  reload: () => void;                     // re-read the file from disk
  rectOf: (ids: string[]) => Rect | null;
  sceneToScreen: (p: [number, number]) => { left: number; top: number };
};

export type TabProps = { state: TabState; actions: TabActions };

/** One short line for a change: "added 2, changed 1". */
export const changeWords = (e: { created: number; updated: number; deleted: number }) =>
  [e.created && `added ${e.created}`, e.updated && `changed ${e.updated}`, e.deleted && `removed ${e.deleted}`].filter(Boolean).join(", ");

/** The one line the pill shows, by precedence. A follower also gets a button. */
export function pillText(s: Pick<TabState, "role" | "working" | "saved">): string {
  if (s.role === "ended") return "Session ended — changes are no longer saved";
  if (s.role === "follower") return "Follower";
  if (s.working) return "Agent working";
  if (s.saved === "saving") return "Saving…";
  if (s.saved === "failed") return "Save failed";
  return "Saved";
}

const DOT: Record<Notice["level"], string> = { info: "#6b7280", done: "#16a34a", question: "#d97706", error: "#dc2626" };
const ACCENT = "#6965db";
const FONT = "13px -apple-system, BlinkMacSystemFont, Segoe UI, Roboto, sans-serif";

const card: React.CSSProperties = {
  background: "#fff", border: "1px solid #e5e7eb", borderRadius: 10,
  boxShadow: "0 4px 16px rgba(0,0,0,.12)", color: "#111827", font: FONT,
};
const linkButton: React.CSSProperties = {
  background: "none", border: "none", padding: 0, margin: 0, font: FONT,
  color: ACCENT, cursor: "pointer", textDecoration: "underline",
};

export function Tab({ state, actions }: TabProps): React.JSX.Element {
  const [log, setLog] = useState(false);            // the notice log is closed by default
  const [kept, setKept] = useState(false);          // "Keep this version" hides the banner
  const [noticeToast, setNoticeToast] = useState<string | null>(null);

  // A new notice always says one line, whether or not the log is open.
  const latest = state.notices[0];
  useEffect(() => {
    if (!latest) return;
    setNoticeToast(latest.text);
    const t = setTimeout(() => setNoticeToast(null), 2500);
    return () => clearTimeout(t);
  }, [latest?.id]);

  // A fresh external change re-opens the banner the user dismissed last time.
  useEffect(() => { if (state.external) setKept(false); }, [state.external]);

  const headless = state.mode === "headless";
  const pending = state.changes;
  const totals = pending.reduce(
    (a, c) => ({ created: a.created + c.created, updated: a.updated + c.updated, deleted: a.deleted + c.deleted }),
    { created: 0, updated: 0, deleted: 0 },
  );
  const allIds = pending.flatMap((c) => c.ids);
  const question = state.notices.find((n) => n.level === "question");
  const toast = state.toast ?? noticeToast;

  return (
    <>
      {/* 1. the status pill — the only permanent chrome, centred on the window
          and below Excalidraw's toolbar, never over it. */}
      <div
        onClick={() => setLog((v) => !v)}
        title="Show what the agent has said"
        style={{
          ...card, position: "fixed", top: 56, left: "50%", transform: "translateX(-50%)", zIndex: 6,
          display: "flex", alignItems: "center", gap: 8, padding: "6px 12px", borderRadius: 999,
          cursor: "pointer", whiteSpace: "nowrap", userSelect: "none",
        }}
      >
        <span style={{
          width: 7, height: 7, borderRadius: "50%", flex: "none",
          background: state.role === "ended" ? "#9ca3af" : state.saved === "failed" ? DOT.error : state.working ? ACCENT : DOT.done,
        }} />
        <span>{pillText(state)}</span>
        {state.role === "follower" && (
          <button style={linkButton} onClick={(e) => { e.stopPropagation(); actions.takeOver(); }}>Take over</button>
        )}
        {state.notices.length > 0 && <span style={{ color: "#9ca3af" }}>{state.notices.length}</span>}
      </div>

      {/* 2. change markers, one outlined box per changed element group, plus one
          bar that says what happened and offers the three things to do about it. */}
      {!headless && pending.map((c) => {
        const r = actions.rectOf(c.ids);
        if (!r) return null;
        return (
          <div key={c.id} style={{
            position: "fixed", left: r.left - 6, top: r.top - 6, width: r.width + 12, height: r.height + 12,
            border: `2px solid ${ACCENT}`, borderRadius: 6, pointerEvents: "none", zIndex: 4,
          }} />
        );
      })}
      {!headless && pending.length > 0 && (
        <div style={{
          ...card, position: "fixed", bottom: 72, left: "50%", transform: "translateX(-50%)", zIndex: 6,
          display: "flex", alignItems: "center", gap: 10, padding: "8px 14px", whiteSpace: "nowrap",
        }}>
          <span>Agent {changeWords(totals)}</span>
          <span style={{ color: "#d1d5db" }}>·</span>
          <button style={linkButton} onClick={() => actions.show(allIds)}>Show</button>
          <span style={{ color: "#d1d5db" }}>·</span>
          <button style={linkButton} onClick={() => actions.undoLast()}>Undo last</button>
          <span style={{ color: "#d1d5db" }}>·</span>
          <button style={linkButton} onClick={() => pending.forEach((c) => actions.dismiss(c.id))}>Got it</button>
        </div>
      )}

      {/* 3. the notice log — everything the agent has said, newest first. */}
      {!headless && log && (
        <div style={{
          ...card, position: "fixed", top: 96, left: "50%", transform: "translateX(-50%)", zIndex: 7,
          width: 320, maxHeight: "50vh", overflowY: "auto", padding: 10,
        }}>
          {question && (
            <div style={{ padding: "6px 8px", marginBottom: 8, borderRadius: 6, background: "#fffbeb", color: "#92400e" }}>
              <div style={{ fontWeight: 600 }}>{question.text}</div>
              <div style={{ marginTop: 2 }}>the agent is asking you something — answer in your chat</div>
            </div>
          )}
          {state.notices.length === 0 && <div style={{ color: "#9ca3af", padding: "4px 8px" }}>Nothing from the agent yet.</div>}
          {state.notices.map((n) => (
            <div key={n.id} style={{ display: "flex", gap: 8, alignItems: "flex-start", padding: "4px 8px" }}>
              <span style={{ width: 7, height: 7, borderRadius: "50%", background: DOT[n.level], marginTop: 5, flex: "none" }} />
              <span>{n.text}</span>
            </div>
          ))}
        </div>
      )}

      {/* 4. the toast — one line, bottom centre, gone in 2.5s. */}
      {!headless && toast && (
        <div style={{
          position: "fixed", bottom: 24, left: "50%", transform: "translateX(-50%)", zIndex: 8,
          background: "#111827", color: "#fff", font: FONT, padding: "8px 14px", borderRadius: 8,
          boxShadow: "0 4px 16px rgba(0,0,0,.2)", maxWidth: "80vw", whiteSpace: "nowrap",
          overflow: "hidden", textOverflow: "ellipsis",
        }}>{toast}</div>
      )}

      {/* 5. the crosshair — where the point that was copied is. DOM, never an
          element on the canvas; main.tsx clears it after 3s. */}
      {!headless && state.crosshair && (() => {
        const p = actions.sceneToScreen(state.crosshair!);
        return (
          <div style={{ position: "fixed", left: p.left - 12, top: p.top - 12, width: 24, height: 24, pointerEvents: "none", zIndex: 5 }}>
            <div style={{ position: "absolute", left: 11, top: 0, width: 2, height: 24, background: ACCENT }} />
            <div style={{ position: "absolute", top: 11, left: 0, height: 2, width: 24, background: ACCENT }} />
          </div>
        );
      })()}

      {/* 6. the external-change banner. Reloading is never silent: what is in
          this tab would be thrown away. */}
      {!headless && state.external && !kept && (
        <div style={{
          position: "fixed", top: 0, left: 0, right: 0, zIndex: 9, font: FONT,
          display: "flex", alignItems: "center", justifyContent: "center", gap: 12, flexWrap: "wrap",
          background: "#fef3c7", borderBottom: "1px solid #fcd34d", color: "#92400e", padding: "8px 14px",
        }}>
          <span>This file was changed by another program. Reloading discards what is in this tab.</span>
          <button style={{ ...linkButton, color: "#92400e", fontWeight: 600 }} onClick={() => actions.reload()}>Reload from disk</button>
          <button style={{ ...linkButton, color: "#92400e" }} onClick={() => setKept(true)}>Keep this version</button>
        </div>
      )}
    </>
  );
}
