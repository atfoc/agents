#!/usr/bin/env python3
"""Autobuild v3: an orchestrator agent steers a run of Claude Code agents toward a goal.

  python3 autobuild-v3/autobuild.py run "add a dark mode toggle"   # or a path to a document
  python3 autobuild-v3/autobuild.py run docs/feature.md --tier light=claude-sonnet-5-5:low
  python3 autobuild-v3/autobuild.py ui                              # UI for the latest run
  python3 autobuild-v3/autobuild.py status

There are no fixed phases. A run is a set of tasks, and an orchestrator agent decides what they
are:

  - The orchestrator is a Claude Code agent that is given the goal and a set of MCP tools to read
    and edit the run: look at the tasks, their reports and what their agents are doing; add,
    change, cancel and retry tasks; keep notes; finish the run. It cannot change the repository.
  - A task is one job for one agent, described by a brief. It either changes the repository (it
    then works in its own git worktree and is merged into the integration branch when it is
    done) or only reports (research, review, verification). Tasks whose dependencies are done
    run in parallel. Task agents have no MCP server at all: only the orchestrator can see or
    edit the run.
  - Every task has a tier (deep, standard or light) that the orchestrator picks; the run's
    configuration maps each tier to a model and an effort (--tier, or --model / --effort for all
    three at once).
  - A new orchestrator instance is started when the tasks the last one said it was waiting for
    are done (wait_for), when a task fails, and when nothing is left running, and is asked what
    the run needs next (--wake each | idle for the fixed policies). It remembers nothing of the
    earlier instances; the run and its notes carry over. This repeats until an instance declares
    the run finished.

Everything is on disk under <repo>/.autobuild3/<run>/ (state.json, goal.md, tasks/<id>.md,
agents/<id>/ with the prompt, transcript, report and result of every agent, worktrees/). Kill
the script at any point and run the same command again: finished work is kept and interrupted
agents are resumed. The result is left on the branch autobuild3/<run>/integration; your checkout
is never touched.

Agents run unattended with --permission-mode bypassPermissions (override with
--permission-mode). Standard library only, Python 3.9+.
"""
from __future__ import annotations

import argparse
import contextlib
import fcntl
import hashlib
import json
import os
import re
import secrets
import signal
import subprocess
import sys
import threading
import time
import traceback
import uuid
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import parse_qs, urlparse

HERE = os.path.dirname(os.path.abspath(__file__))
UI_FILE = os.path.join(HERE, "ui.html")

TIERS = ("deep", "standard", "light")
ORCHESTRATOR_TIER = "deep"
MERGE_TIER = "standard"

DEFAULTS = {
    # what each tier runs on; the orchestrator picks a tier per task and never names a model
    "tiers": {"deep": {"model": "claude-opus-5-5", "effort": "high"},
              "standard": {"model": "claude-opus-5-5", "effort": "medium"},
              "light": {"model": "claude-sonnet-5-5", "effort": "medium"}},
    "max_parallel": 8,
    # when the orchestrator is started again: when the tasks it said it waits for are done
    # ("declared"), after "each" finished task, or only when "idle"
    "wake": "declared",
    "max_turns": 60,           # orchestrator turns before the run is stopped
    "max_cost": 0.0,           # USD the run may spend before it is stopped (0 = no limit)
    "agent_timeout": 3 * 3600,
    "agent_retries": 2,        # extra attempts when an agent process fails
    "setup_cmd": "",           # run once in every new task worktree (e.g. "cd web && npm ci")
    "claude_bin": "claude",
    "permission_mode": "bypassPermissions",
    "keep_worktrees": False,
}
RUN_ROOT = ".autobuild3"
BRANCH_NS = "autobuild3"
MCP_SERVER = "autobuild"
INLINE_LIMIT = 40_000          # longer goals and reports are passed by path instead of inline
REPORTS_LIMIT = 60_000         # characters of dependency reports a task is given inline
REPORTS_MANY = 3               # reports a task may be given before the orchestrator is warned
NOTES_LIMIT = 20_000           # characters of notes before the orchestrator is told to shorten them
MAX_IDLE_TURNS = 3             # turns in a row that may start with nothing running
ID_RE = re.compile(r"^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$")
KIND_RE = re.compile(r"^[a-z][a-z0-9-]{0,23}$")
RUN_IDENT = ["-c", "user.name=autobuild", "-c", "user.email=autobuild@localhost"]
ACTIVE = ("running", "merging")
# what belongs to one attempt at a task, and is put aside when the task is retried
ATTEMPT_FIELDS = ("base", "branch", "worktree", "head", "merged", "conflicts", "started_at", "ended_at", "error",
                  "summary", "outcome", "setup_done", "work_done")
# the orchestrator decides and delegates: it cannot edit files, and it cannot hide work in subagents
ORCHESTRATOR_DISALLOWED = "Edit,Write,NotebookEdit,Task,Agent,Workflow"

TASK_SCHEMA = {
    "type": "object",
    "properties": {
        "outcome": {"type": "string", "enum": ["completed", "failed"],
                    "description": "completed: you did what the brief asks. failed: you could not."},
        "summary": {"type": "string", "description": "Two or three sentences: what you did or found. Tasks that "
                                                     "build on yours may be given only this."},
        "report": {"type": "string", "description": "The full report, in markdown."},
    },
    "required": ["outcome", "summary", "report"],
}

TOOLS = [
    {"name": "get_run",
     "description": "The run as it is now: the notes, every task with its status and the summary of its result, "
                    "what is running, what happened since your turn began, and the earlier turns.",
     "inputSchema": {"type": "object", "properties": {}}},
    {"name": "get_task",
     "description": "Everything about one task: its brief, status, dependencies, the full report of its agent, "
                    "the commits it produced, its error if it failed, and its agents.",
     "inputSchema": {"type": "object", "properties": {"id": {"type": "string", "description": "Task id, e.g. T03"}},
                     "required": ["id"]}},
    {"name": "get_agent",
     "description": "What one agent has been doing: its status and the most recent things it said and did. Use it "
                    "to see the progress of a running task or to find out why one failed.",
     "inputSchema": {"type": "object", "properties": {
         "agent": {"type": "string", "description": "Agent id, e.g. T03-work"},
         "last": {"type": "integer", "description": "How many recent items to return (default 15)"}},
         "required": ["agent"]}},
    {"name": "set_notes",
     "description": "Replaces the whole of the run's notes, your memory across orchestrator instances: what the "
                    "goal requires, the definition of done and how it will be checked, facts established and by "
                    "which task, the approach and what is planned next, decisions and why, open questions and "
                    "risks. For the first version and for reorganising them; to change one part use edit_notes.",
     "inputSchema": {"type": "object", "properties": {"notes": {"type": "string", "description": "Markdown"}},
                     "required": ["notes"]}},
    {"name": "edit_notes",
     "description": "Replaces one section of the notes and leaves the rest as it is. The section is everything "
                    "under the heading you name, up to the next heading of the same or a higher level. A heading "
                    "the notes do not have is added at the end as a new section; empty text removes the section.",
     "inputSchema": {"type": "object", "properties": {
         "heading": {"type": "string", "description": "The heading of the section, without the # marks"},
         "text": {"type": "string", "description": "Markdown: the new content of the section, without its "
                                                   "heading. Empty to remove the section."}},
         "required": ["heading", "text"]}},
    {"name": "add_task",
     "description": "Adds a task and returns its id. It starts after your turn ends, once every task it depends on "
                    "is done. Its agent knows the brief, the summaries of the tasks it depends on, and the full "
                    "reports of those named in needs_report; nothing else.",
     "inputSchema": {"type": "object", "properties": {
         "title": {"type": "string", "description": "Short imperative title"},
         "brief": {"type": "string", "description": "Markdown: what to do and why, what exists that it builds on, "
                                                    "exact names and shapes of anything shared with other tasks, "
                                                    "what is out of scope, how the result is to be verified."},
         "kind": {"type": "string", "description": "A one-word label for the kind of work, e.g. research, design, "
                                                   "implement, review, verify, fix"},
         "writes": {"type": "boolean", "description": "true: the task changes the repository and its changes are "
                                                      "merged. false: it only reports; its checkout is discarded."},
         "tier": {"type": "string", "enum": list(TIERS),
                  "description": "How capable an agent the task gets. deep: its result is a decision that other "
                                 "work is built on, or nothing after it checks it. standard: work from a precise "
                                 "brief whose result a build, a test or a later task checks. light: gathering "
                                 "facts or following a recipe, cheap to do again."},
         "tier_reason": {"type": "string", "description": "One sentence: why this tier is enough for this task"},
         "depends_on": {"type": "array", "items": {"type": "string"},
                        "description": "Ids of the tasks that must be done before this one starts. Its agent is "
                                       "given the summary of each and where its report is."},
         "needs_report": {"type": "array", "items": {"type": "string"},
                          "description": "Those of depends_on whose full report the agent must read before it "
                                         "can start: they are put in front of it whole. Name as few as the task "
                                         "can work from."}},
         "required": ["title", "brief", "kind", "writes", "tier", "tier_reason"]}},
    {"name": "update_task",
     "description": "Changes a task that has not started (or one that failed or was cancelled, before you retry "
                    "it). Give only the fields to change. A running task must be cancelled first; a done task is "
                    "final.",
     "inputSchema": {"type": "object", "properties": {
         "id": {"type": "string"}, "title": {"type": "string"}, "brief": {"type": "string"},
         "kind": {"type": "string"}, "writes": {"type": "boolean"},
         "tier": {"type": "string", "enum": list(TIERS)}, "tier_reason": {"type": "string"},
         "depends_on": {"type": "array", "items": {"type": "string"}},
         "needs_report": {"type": "array", "items": {"type": "string"}}},
         "required": ["id"]}},
    {"name": "cancel_task",
     "description": "Cancels a task. A running task's agent is stopped and its work is not merged. Tasks that "
                    "depend on a cancelled task cannot start until you change or cancel them too.",
     "inputSchema": {"type": "object", "properties": {
         "id": {"type": "string"}, "reason": {"type": "string", "description": "Why, for the record"}},
         "required": ["id", "reason"]}},
    {"name": "retry_task",
     "description": "Queues a failed or cancelled task again, from a fresh checkout of the integration branch. "
                    "Change its brief first (update_task) if the brief was the problem; give a higher tier if "
                    "the task was too hard for its agent.",
     "inputSchema": {"type": "object", "properties": {
         "id": {"type": "string"}, "reason": {"type": "string", "description": "Why it should work this time"},
         "tier": {"type": "string", "enum": list(TIERS), "description": "The tier of the new attempt, if it "
                                                                        "should differ"}},
         "required": ["id", "reason"]}},
    {"name": "wait_for",
     "description": "Says which tasks you are waiting for. After this turn you are started again when they have "
                    "ended (all of them, or the first one with mode any), and in any case when a task fails or "
                    "nothing is left running. Other tasks that finish meanwhile do not start you; you are told "
                    "about them then. It holds until your next turn starts; calling it again replaces it.",
     "inputSchema": {"type": "object", "properties": {
         "tasks": {"type": "array", "items": {"type": "string"},
                   "description": "Ids of pending or running tasks whose results you need before you can decide "
                                  "anything more"},
         "mode": {"type": "string", "enum": ["all", "any"], "description": "Default all"}},
         "required": ["tasks"]}},
    {"name": "finish_run",
     "description": "Ends the run. No further orchestrator instance is started. Refused while tasks are pending or "
                    "running.",
     "inputSchema": {"type": "object", "properties": {
         "outcome": {"type": "string", "enum": ["achieved", "not_achieved"]},
         "summary": {"type": "string", "description": "Markdown for the person who started the run: what was "
                                                      "built, how it was verified, and anything they should know."}},
         "required": ["outcome", "summary"]}},
]
READ_OPS = ("get_run", "get_task", "get_agent")
# the operations that change what the run will do; a turn without one left the plan as it was
PLAN_OPS = ("add_task", "update_task", "cancel_task", "retry_task", "finish_run")

WRITE_RULES = """Rules:
- Work only inside your working directory. Other agents are working at the same time in other
  git worktrees of this repository; never touch anything outside your own.
- Do not commit, push, stash, switch branches or rewrite history.
- You run unattended and cannot ask questions. Where something is unclear, make the most
  reasonable choice and say so in your report."""

RESUME_PROMPT = """Your previous run of this job stopped before it finished. Continue where you left off and
complete the job. Everything in the original instructions still applies, including how to end."""

NOTES_GATE = """The notes are empty, so a task that changes the repository is refused. First record with set_notes
what the goal requires, what done means and how it will be checked, and the approach the facts support. If you
cannot write that yet, the run needs tasks that investigate (writes: false) before tasks that build."""


class Stop(Exception):
    """The run is shutting down: a signal arrived or something ended it."""


class AgentError(Exception):
    """An agent produced no usable result after all its attempts."""


class Cancelled(Exception):
    """The orchestrator cancelled the task this agent was working on."""


class TaskFailed(Exception):
    """A task's agent reported that it could not do the task."""


class OpError(Exception):
    """An operation of the orchestrator that the run refuses. The message goes back to the orchestrator."""


# ---------------------------------------------------------------- small helpers

def now():
    return time.time()


def read_text(path, default=""):
    try:
        with open(path, encoding="utf-8", errors="replace") as f:
            return f.read()
    except OSError:
        return default


def write_text(path, text):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    tmp = f"{path}.tmp{os.getpid()}.{threading.get_ident()}"
    with open(tmp, "w", encoding="utf-8") as f:
        f.write(text)
    os.replace(tmp, path)


def read_json(path, default=None):
    try:
        with open(path, encoding="utf-8") as f:
            return json.load(f)
    except (OSError, ValueError):
        return default


def write_json(path, data):
    write_text(path, json.dumps(data, indent=1))


def slugify(text, limit=40):
    s = re.sub(r"[^a-z0-9]+", "-", text.lower()).strip("-")
    return s[:limit].strip("-") or "run"


def clip(text, limit):
    text = str(text or "")
    return text if len(text) <= limit else text[:limit].rstrip() + " ..."


def pid_alive(pid):
    if not pid:
        return False
    try:
        os.kill(pid, 0)
    except ProcessLookupError:
        return False
    except PermissionError:
        return True
    return True


def kill_tree(pid):
    """SIGTERM the process group, SIGKILL whatever is left a few seconds later."""
    try:
        os.killpg(pid, signal.SIGTERM)
    except (ProcessLookupError, PermissionError):
        return

    def hard():
        with contextlib.suppress(ProcessLookupError, PermissionError):
            os.killpg(pid, signal.SIGKILL)

    t = threading.Timer(8, hard)
    t.daemon = True
    t.start()


def git_rc(cwd, *args):
    p = subprocess.run(["git", *args], cwd=cwd, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                       encoding="utf-8", errors="replace")
    return p.returncode, p.stdout, p.stderr


def git(cwd, *args):
    rc, out, err = git_rc(cwd, *args)
    if rc:
        raise RuntimeError(f"git {' '.join(args)} failed in {cwd}:\n{(err or out).strip()}")
    return out.strip()


def git_ok(cwd, *args):
    return git_rc(cwd, *args)[0] == 0


def in_merge(cwd):
    return os.path.exists(os.path.join(cwd, git(cwd, "rev-parse", "--git-path", "MERGE_HEAD")))


@contextlib.contextmanager
def repo_lock(repo):
    """Held around worktree bookkeeping: other runs, from any worktree, share this repository."""
    path = os.path.join(repo, git(repo, "rev-parse", "--git-common-dir"), "autobuild.lock")
    with open(path, "w") as f:
        fcntl.flock(f, fcntl.LOCK_EX)
        yield


def commit_all(cwd, message, ident):
    """Commits everything in cwd if there is anything to commit. Returns HEAD."""
    git(cwd, "add", "-A")
    if git(cwd, "status", "--porcelain"):
        rc, out, err = git_rc(cwd, *ident, "commit", "-q", "-m", message)
        if rc:  # a commit hook rejecting an intermediate commit must not wedge the run
            git(cwd, *ident, "commit", "-q", "--no-verify", "-m", message)
    return git(cwd, "rev-parse", "HEAD")


def fmt_duration(seconds):
    seconds = int(seconds or 0)
    h, rem = divmod(seconds, 3600)
    m, s = divmod(rem, 60)
    return f"{h}h{m:02d}m" if h else (f"{m}m{s:02d}s" if m else f"{s}s")


def cycle_through(tasks, tid, deps):
    """The dependency path that would lead from tid back to itself if it were given these deps."""
    todo, seen = [(d, [tid, d]) for d in deps], set()
    while todo:
        cur, path = todo.pop()
        if cur == tid:
            return path
        if cur in seen or cur not in tasks:
            continue
        seen.add(cur)
        todo += [(d, path + [d]) for d in tasks[cur]["depends_on"]]
    return None


def agent_prefix(tid, attempt):
    return tid if attempt == 1 else f"{tid}-a{attempt}"


def replace_section(notes, heading, text):
    """The notes with the section under a heading replaced, added or (empty text) removed.

    Returns (notes, what was done). A section runs from its heading to the next heading of the
    same or a higher level; headings inside code fences are not headings.
    """
    def norm(s):
        return " ".join(s.strip().strip("#").split()).lower()

    lines, want, text = notes.split("\n"), norm(heading), text.strip()
    if not want:
        raise OpError("the heading is empty")
    heads, fenced = [], False            # (line index, level) of every heading
    for i, line in enumerate(lines):
        if line.lstrip().startswith(("```", "~~~")):
            fenced = not fenced
        m = None if fenced else re.match(r"(#{1,6})\s+\S", line)
        if m:
            heads.append((i, len(m.group(1))))
    found = [(i, level) for i, level in heads if norm(lines[i]) == want]
    if len(found) > 1:
        raise OpError(f"the notes have {len(found)} sections called {heading.strip()!r}; rename them with set_notes")
    if not found:
        if not text:
            raise OpError(f"the notes have no section called {heading.strip()!r}; there is nothing to remove")
        title = heading.strip() if heading.lstrip().startswith("#") else "## " + heading.strip()
        return (notes.rstrip() + "\n\n" if notes.strip() else "") + f"{title}\n\n{text}", "added"
    start, level = found[0]
    end = next((i for i, lv in heads if i > start and lv <= level), len(lines))
    body = [lines[start], "", text, ""] if text else []
    out = "\n".join(lines[:start] + body + lines[end:])
    return re.sub(r"\n{3,}", "\n\n", out).strip(), "replaced" if text else "removed"


def tokens_of(result):
    """Token counts of a result event. modelUsage includes the agent's subagents; usage does not."""
    per_model = [m for m in (result.get("modelUsage") or {}).values() if isinstance(m, dict)]
    if per_model:
        keys = {"output_tokens": "outputTokens", "input_tokens": "inputTokens",
                "cache_read_tokens": "cacheReadInputTokens", "cache_write_tokens": "cacheCreationInputTokens"}
        return {k: sum(m.get(v) or 0 for m in per_model) for k, v in keys.items()}
    u = result.get("usage") or {}
    return {"output_tokens": u.get("output_tokens") or 0, "input_tokens": u.get("input_tokens") or 0,
            "cache_read_tokens": u.get("cache_read_input_tokens") or 0,
            "cache_write_tokens": u.get("cache_creation_input_tokens") or 0}


def context_of(event):
    """The tokens of context an agent's own request was made with (0 for anything else)."""
    if event.get("type") != "assistant" or event.get("parent_tool_use_id"):
        return 0
    u = (event.get("message") or {}).get("usage") or {}
    return sum(u.get(k) or 0 for k in ("input_tokens", "cache_read_input_tokens", "cache_creation_input_tokens"))


def median(values):
    values = sorted(values)
    return values[len(values) // 2] if values else 0


def agent_activity(agent_dir, last):
    """The most recent things an agent said and did, from the tail of its transcript."""
    try:
        with open(os.path.join(agent_dir, "events.jsonl"), "rb") as f:
            f.seek(0, os.SEEK_END)
            start = max(0, f.tell() - 600_000)
            f.seek(start)
            lines = f.read().splitlines()
    except OSError:
        return []
    items = []
    for raw in lines[1 if start else 0:]:
        try:
            e = json.loads(raw)
        except ValueError:
            continue
        if e.get("parent_tool_use_id") or e.get("type") not in ("assistant", "user"):
            continue
        content = (e.get("message") or {}).get("content")
        for b in content if isinstance(content, list) else []:
            if b.get("type") == "text" and e["type"] == "assistant" and b.get("text", "").strip():
                items.append("said: " + clip(b["text"].strip(), 1200))
            elif b.get("type") == "tool_use":
                inp = b.get("input") or {}
                what = inp.get("command") or inp.get("file_path") or inp.get("pattern") or inp.get("description") or ""
                items.append(f"used {b.get('name')}: " + clip(str(what).split("\n")[0], 200))
            elif b.get("type") == "tool_result" and b.get("is_error"):
                c = b.get("content")
                text = c if isinstance(c, str) else " ".join(x.get("text", "") for x in c or [] if isinstance(x, dict))
                items.append("tool error: " + clip(text, 300))
    return items[-last:]


# ---------------------------------------------------------------- the run

class Run:
    def __init__(self, run_dir):
        self.run_dir = run_dir
        self.state_path = os.path.join(run_dir, "state.json")
        self.state = read_json(self.state_path)
        self.cfg = self.state["config"]
        self.repo = self.state["repo"]
        self.tasks_dir = os.path.join(run_dir, "tasks")
        self.agents_dir = os.path.join(run_dir, "agents")
        self.int_wt = os.path.join(run_dir, "worktrees", "integration")
        self.orch_wt = os.path.join(run_dir, "worktrees", "orchestrator")
        self.int_branch = self.state["integration_branch"]
        self.lock = threading.RLock()
        self.cond = threading.Condition(self.lock)
        self.git_lock = threading.Lock()      # worktree bookkeeping and merges, one at a time
        self.log_lock = threading.Lock()
        self.procs_lock = threading.Lock()
        self.procs = {}                       # pid -> unit (a task id, or "orchestrator")
        self.cancels = {}                     # task id -> Event set when the orchestrator cancels it
        self.tokens = {}                      # MCP bearer token -> the turn it belongs to, while that turn runs
        self.mcp_url = None
        self.stop = threading.Event()
        self.failure = None                   # (status, message) of the first thing that ended the run
        configured = git_ok(self.repo, "config", "user.name") and git_ok(self.repo, "config", "user.email")
        self.ident = [] if configured else RUN_IDENT

    # ---- state

    @contextlib.contextmanager
    def mut(self):
        with self.cond:
            yield self.state
            self.state["updated_at"] = now()
            write_json(self.state_path, self.state)
            self.cond.notify_all()

    def log(self, msg):
        line = f"{time.strftime('%H:%M:%S')} {msg}"
        with self.log_lock:
            print(line, flush=True)
            with open(os.path.join(self.run_dir, "orchestrator.log"), "a", encoding="utf-8") as f:
                f.write(f"{time.strftime('%Y-%m-%d')} {line}\n")

    def check_stop(self):
        if self.stop.is_set():
            raise Stop()

    def end_run(self, status, message):
        """Stops the whole run. The first caller decides how the run is reported."""
        with self.lock:
            if self.failure is None:
                self.failure = (status, message)
        self.stop.set()
        with self.procs_lock:
            pids = list(self.procs)
        for pid in pids:
            kill_tree(pid)
        with self.cond:
            self.cond.notify_all()

    def event(self, kind, task, text):
        """Something the next orchestrator instance is told about."""
        with self.mut() as s:
            s["events"].append({"seq": len(s["events"]) + 1, "t": now(), "type": kind, "task": task,
                                "text": clip(text, 600)})

    # ---- agents

    def agent_dir(self, agent_id):
        assert ID_RE.match(agent_id), agent_id
        d = os.path.join(self.agents_dir, agent_id)
        os.makedirs(d, exist_ok=True)
        return d

    def reap_stale_agents(self):
        """Agents a previous process left 'running': stop any survivor and mark them."""
        if not os.path.isdir(self.agents_dir):
            return
        for name in os.listdir(self.agents_dir):
            path = os.path.join(self.agents_dir, name, "meta.json")
            meta = read_json(path)
            if not meta or meta.get("status") != "running":
                continue
            pid, session = meta.get("pid"), meta.get("session_id")
            if pid_alive(pid):
                # The pid may have been reused since, by another run's agent or any other claude
                # process: only the session id on the command line says the process is this agent.
                cmd = subprocess.run(["ps", "-ww", "-o", "command=", "-p", str(pid)], stdout=subprocess.PIPE,
                                     encoding="utf-8", errors="replace").stdout
                if session and session in cmd:
                    self.log(f"stopping agent {name} left over from an earlier run (pid {pid})")
                    kill_tree(pid)
                    for _ in range(100):     # let it let go of its session before that is resumed
                        if not pid_alive(pid):
                            break
                        time.sleep(0.1)
            meta["status"] = "interrupted"
            write_json(path, meta)

    def run_agent(self, agent_id, role, unit, prompt, cwd, tier, schema=None, add_dirs=(), title="",
                  mcp_config=None, disallowed="", cancel=None):
        """Runs one Claude Code agent to completion and returns its meta record.

        Idempotent: a finished agent is returned from disk, an interrupted one resumes its
        session. The tier decides the model and effort when the agent is first started; it keeps
        them for as long as it lives, whatever the configuration says later, because a session
        cannot be resumed on another model. Only an agent given mcp_config can reach the run's
        MCP tools. Raises Stop when the run is shutting down, Cancelled when its task was
        cancelled, and AgentError when the agent keeps failing.
        """
        d = self.agent_dir(agent_id)
        meta_path = os.path.join(d, "meta.json")
        meta = read_json(meta_path)
        if meta and meta.get("status") == "done":
            return meta
        self.check_stop()
        if not meta:
            on = self.cfg["tiers"][tier]
            meta = {"id": agent_id, "role": role, "unit": unit, "title": title, "cwd": cwd, "tier": tier,
                    "model": on["model"], "effort": on["effort"], "mcp": bool(mcp_config),
                    "mcp_servers": None, "created_at": now(), "attempts": [], "session_id": None,
                    "session_live": False, "cost_usd": 0.0, "num_turns": 0, "output_tokens": 0,
                    "input_tokens": 0, "cache_read_tokens": 0, "cache_write_tokens": 0, "peak_context": 0}
        meta.update(status="running", started_at=meta.get("started_at") or now(), ended_at=None, error=None)
        write_json(meta_path, meta)

        def halted():
            """Ends the agent as interrupted or cancelled when either was asked for."""
            cancelled = cancel is not None and cancel.is_set()
            if not cancelled and not self.stop.is_set():
                return
            word = "cancelled" if cancelled else "interrupted"
            meta.update(status=word, ended_at=now(), error=word)
            write_json(meta_path, meta)
            raise Cancelled() if cancelled else Stop()

        failures = 0
        while True:
            resume = bool(meta.get("session_id") and meta.get("session_live"))
            if resume:
                text = RESUME_PROMPT
            else:
                text = prompt
                if meta["attempts"]:
                    text += ("\n\nNote: an earlier attempt at this job did not finish. Your working directory may "
                             "hold its partial work; check `git status` and build on whatever is sound.")
                meta["session_id"], meta["session_live"] = str(uuid.uuid4()), False
                write_text(os.path.join(d, "prompt.md"), prompt)
            stdin_path = os.path.join(d, "stdin.md")
            write_text(stdin_path, text)
            attempt = {"n": len(meta["attempts"]) + 1, "resume": resume, "started_at": now()}
            meta["attempts"].append(attempt)
            self.log(f"agent {agent_id}: {'resuming' if resume else 'starting'} (attempt {attempt['n']})")
            ok, error = self._exec(meta, meta_path, d, stdin_path, cwd, schema, add_dirs, resume, attempt,
                                   mcp_config, disallowed, cancel)
            attempt["ended_at"] = now()
            if ok:
                meta.update(status="done", ended_at=now(), error=None)
                write_json(meta_path, meta)
                self.log(f"agent {agent_id}: done in {fmt_duration(meta['ended_at'] - meta['started_at'])}"
                         f" (${meta['cost_usd']:.2f})")
                return meta
            attempt["error"] = error
            halted()
            if resume and attempt.get("no_session"):
                meta["session_live"] = False   # nothing to resume; start over without counting it
                write_json(meta_path, meta)
                continue
            failures += 1
            self.log(f"agent {agent_id}: attempt {attempt['n']} failed: {error}")
            if failures > self.cfg["agent_retries"] or attempt.get("fatal"):
                meta.update(status="failed", ended_at=now(), error=error)
                write_json(meta_path, meta)
                raise AgentError(f"agent {agent_id} failed: {error}")
            write_json(meta_path, meta)
            wait_until = now() + 30 * failures * failures
            while now() < wait_until:
                halted()
                time.sleep(0.5)

    def _exec(self, meta, meta_path, d, stdin_path, cwd, schema, add_dirs, resume, attempt, mcp_config,
              disallowed, cancel):
        cfg = self.cfg
        # --strict-mcp-config: an agent gets the MCP servers named here and no others, so an agent
        # that is not handed the run's server has none.
        cmd = [cfg["claude_bin"], "-p", "--model", meta["model"], "--output-format", "stream-json", "--verbose",
               "--permission-mode", cfg["permission_mode"], "--strict-mcp-config",
               "--name", f"autobuild {self.state['name']} / {meta['id']}"]
        if mcp_config:
            cmd += ["--mcp-config", mcp_config, "--allowedTools", f"mcp__{MCP_SERVER}"]
        if meta.get("effort"):
            cmd += ["--effort", meta["effort"]]
        cmd += ["--resume", meta["session_id"]] if resume else ["--session-id", meta["session_id"]]
        if schema:
            cmd += ["--json-schema", json.dumps(schema)]
        if disallowed:
            cmd += ["--disallowedTools", disallowed]
        for a in add_dirs:
            cmd += ["--add-dir", a]

        result, timed_out, leak, peak = None, [], [], 0
        with open(os.path.join(d, "events.jsonl"), "ab") as events, \
                open(os.path.join(d, "stderr.log"), "ab") as stderr, open(stdin_path, "rb") as stdin:
            marker = {"type": "orchestrator", "subtype": "attempt", "n": attempt["n"], "resume": resume,
                      "timestamp": now()}
            events.write(json.dumps(marker).encode() + b"\n")
            events.flush()
            try:
                proc = subprocess.Popen(cmd, cwd=cwd, stdin=stdin, stdout=subprocess.PIPE, stderr=stderr,
                                        start_new_session=True)
            except OSError as e:
                return False, f"could not start {cfg['claude_bin']}: {e}"
            with self.procs_lock:
                self.procs[proc.pid] = meta["unit"]
            # a stop or a cancel may have swept the process list just before we joined it
            if self.stop.is_set() or (cancel is not None and cancel.is_set()):
                kill_tree(proc.pid)
            meta["pid"] = proc.pid
            write_json(meta_path, meta)

            def on_timeout():
                timed_out.append(True)
                kill_tree(proc.pid)

            timer = threading.Timer(cfg["agent_timeout"], on_timeout)
            timer.daemon = True
            timer.start()
            try:
                for raw in proc.stdout:
                    events.write(raw)
                    events.flush()
                    try:
                        e = json.loads(raw)
                    except ValueError:
                        continue
                    if e.get("type") == "system" and e.get("subtype") == "init":
                        servers = [str(m.get("name")) for m in e.get("mcp_servers") or [] if isinstance(m, dict)]
                        meta.update(session_live=True, mcp_servers=servers)
                        write_json(meta_path, meta)
                        tools = [t for t in e.get("tools") or [] if str(t).startswith(f"mcp__{MCP_SERVER}__")]
                        if not mcp_config and (MCP_SERVER in servers or tools):
                            leak.append(True)      # must never happen; do not let it act on the run
                            kill_tree(proc.pid)
                    elif e.get("type") == "result":
                        result = e
                    else:
                        peak = max(peak, context_of(e))
                rc = proc.wait()
            finally:
                timer.cancel()
                with self.procs_lock:
                    self.procs.pop(proc.pid, None)

        meta["peak_context"] = max(meta.get("peak_context") or 0, peak)
        if result:
            meta["cost_usd"] += result.get("total_cost_usd") or 0
            meta["num_turns"] += result.get("num_turns") or 0
            for k, v in tokens_of(result).items():
                meta[k] = (meta.get(k) or 0) + v
        if leak:
            attempt["fatal"] = True
            return False, "the run's MCP server was available to an agent that must not have it; stopped it"
        if timed_out:
            return False, f"timed out after {fmt_duration(cfg['agent_timeout'])}"
        if result is None:
            tail = read_text(os.path.join(d, "stderr.log"))[-400:].strip()
            return False, f"exited with code {rc} and no result. {tail}"
        if result.get("is_error"):
            if resume and not result.get("num_turns"):
                attempt["no_session"] = True
            return False, str(result.get("result") or result.get("subtype") or "error")[:600]
        text = str(result.get("result") or "")
        if schema:
            structured = result.get("structured_output")
            if not isinstance(structured, dict):
                try:
                    structured = json.loads(text[text.index("{"): text.rindex("}") + 1])
                except ValueError:
                    return False, "finished without the required structured output"
            missing = [k for k in schema.get("required", []) if k not in structured]
            if missing:
                return False, f"structured output is missing {', '.join(missing)}"
            write_json(os.path.join(d, "result.json"), structured)
        write_text(os.path.join(d, "report.md"), text)
        return True, None

    def agent_result(self, agent_id):
        return read_json(os.path.join(self.agents_dir, agent_id, "result.json"), {})

    def agent_report(self, agent_id):
        return read_text(os.path.join(self.agents_dir, agent_id, "report.md")).strip()

    # ---- git

    def ensure_worktree(self, path, branch, base):
        """A worktree at path: on branch (created from base if new), or detached at base without one."""
        with self.git_lock, repo_lock(self.repo):
            if os.path.exists(os.path.join(path, ".git")):
                return
            git(self.repo, "worktree", "prune")
            if not branch:
                git(self.repo, "worktree", "add", "-q", "--detach", path, base)
            elif git_ok(self.repo, "rev-parse", "--verify", "-q", f"refs/heads/{branch}"):
                git(self.repo, "worktree", "add", "-q", path, branch)
            else:
                git(self.repo, "worktree", "add", "-q", "-b", branch, path, base)

    def remove_worktree(self, path):
        with self.git_lock, repo_lock(self.repo):
            git_rc(self.repo, "worktree", "remove", "--force", path)

    def run_setup(self, cwd, label):
        cmd = self.cfg["setup_cmd"]
        if not cmd:
            return
        self.log(f"{label}: running setup command")
        log_path = os.path.join(self.run_dir, "setup-logs", f"{label}.log")
        os.makedirs(os.path.dirname(log_path), exist_ok=True)
        with open(log_path, "ab") as out:
            rc = subprocess.run(cmd, shell=True, cwd=cwd, stdout=out, stderr=subprocess.STDOUT).returncode
        if rc:
            raise RuntimeError(f"setup command failed in {cwd} (exit {rc}); see {log_path}")

    def refresh_orchestrator_worktree(self):
        """The orchestrator reads the code from a checkout of its own, put at the integration head
        before every turn, so that nothing it runs there can disturb a merge."""
        self.ensure_worktree(self.orch_wt, None, self.int_branch)
        with self.git_lock:
            git(self.orch_wt, "checkout", "-q", "--detach", "--force", self.int_branch)
            git(self.orch_wt, "clean", "-fdq")

    # ---- reading the run

    def task_state(self, t):
        """A task's status, with what a pending task is waiting for."""
        tasks = self.state["tasks"]
        if t["status"] != "pending":
            return t["status"]
        if t.get("held"):
            return "pending, starts when the current turn ends"
        bad = [d for d in t["depends_on"] if tasks[d]["status"] in ("failed", "cancelled")]
        if bad:
            return "pending, blocked by " + ", ".join(f"{d} ({tasks[d]['status']})" for d in bad)
        waiting = [d for d in t["depends_on"] if tasks[d]["status"] != "done"]
        return ("pending, waiting on " + ", ".join(waiting)) if waiting else "pending, ready to start"

    def is_ready(self, tid):
        t = self.state["tasks"][tid]
        return (t["status"] == "pending" and not t.get("held")
                and all(self.state["tasks"][d]["status"] == "done" for d in t["depends_on"]))

    def goal_block(self):
        path = os.path.join(self.run_dir, "goal.md")
        text = read_text(path)
        src = self.state.get("goal_source")
        note = f"(The goal was read from {src}.)\n\n" if src else ""
        if len(text) <= INLINE_LIMIT:
            return f"{note}<goal>\n{text.strip()}\n</goal>"
        return f"{note}The goal document is long. Read all of it before anything else: {path}"

    def snapshot(self):
        """The run as text, for the orchestrator."""
        s = self.state
        tasks, order = s["tasks"], s["task_order"]
        agents = load_agents(self.run_dir)
        cost = {}
        for a in agents:
            cost[a["unit"]] = cost.get(a["unit"], 0) + (a.get("cost_usd") or 0)
        counts = {}
        for tid in order:
            counts[tasks[tid]["status"]] = counts.get(tasks[tid]["status"], 0) + 1
        head = git(self.repo, "rev-parse", "--short=12", self.int_branch)
        commits = git(self.repo, "rev-list", "--count", f"{s['base_ref']}..{self.int_branch}")
        lines = [f"Run {s['name']}, turn {len(s['turns'])} of at most {self.cfg['max_turns']}. Tasks: "
                 + (", ".join(f"{n} {k}" for k, n in counts.items()) or "none") + f". Spent ${sum(cost.values()):.2f}"
                 + f" (orchestrator ${cost.get('orchestrator', 0):.2f}).",
                 f"Integration branch {self.int_branch} at {head}, {commits} commit(s) since the run started "
                 f"from {s['base_ref'][:12]}.",
                 "", "## Notes" + (f" (last changed in turn {s['notes_turn']})" if s.get("notes_turn") else ""), "",
                 s["notes"].strip() or "(empty)", "", "## Tasks", ""]
        for tid in order:
            t = tasks[tid]
            line = (f"{tid} [{t['kind']}, {'writes' if t['writes'] else 'reports only'}, {t['tier']}] "
                    f"{self.task_state(t)}")
            if t["status"] in ACTIVE and t.get("started_at"):
                line += f" for {fmt_duration(now() - t['started_at'])} (agent {agent_prefix(tid, t['attempt'])}-work)"
            if t["attempt"] > 1:
                line += f", attempt {t['attempt']}"
            if cost.get(tid):
                line += f", ${cost[tid]:.2f}"
            line += f": {t['title']}"
            if t["depends_on"]:
                line += f" (depends on {', '.join(t['depends_on'])})"
            lines.append(line)
            if t.get("error"):
                lines.append(f"    error: {clip(t['error'], 400)}")
            elif t.get("summary"):
                lines.append(f"    result: {clip(t['summary'], 400)}")
        if not order:
            lines.append("(no tasks yet)")
        earlier = [t for t in s["turns"][:-1]]
        if earlier:
            lines += ["", "## Earlier turns", ""]
            for t in earlier[-12:]:
                changes = [o for o in t["ops"] if o["op"] not in READ_OPS and not o.get("error")]
                did = ", ".join(o["op"] + (f" {o['task']}" if o.get("task") else "") for o in changes) or "no changes"
                lines.append(f"Turn {t['n']} ({did}): {clip(t.get('summary') or '(no message)', 500)}")
        return "\n".join(lines)

    def event_lines(self, events):
        out = []
        for e in events:
            if e["type"] == "task_done":
                out.append(f"- {e['task']} finished. {e['text']}")
            elif e["type"] == "task_failed":
                out.append(f"- {e['task']} failed. {e['text']}")
            else:
                out.append(f"- {e['text']}")
        return "\n".join(out)

    def task_report(self, tid):
        """(the report of a task's latest attempt, the path it can be read at)."""
        agent_id = agent_prefix(tid, self.state["tasks"][tid]["attempt"]) + "-work"
        return (str(self.agent_result(agent_id).get("report") or "").strip(),
                os.path.join(self.agents_dir, agent_id, "report.md"))

    def dependency_results(self, t):
        """What a task's agent is told about the tasks it depends on.

        The summary of each, and the whole report of those the orchestrator named in
        needs_report: everything put in front of an agent is paid for on every step it takes.
        """
        tasks, out, budget = self.state["tasks"], [], REPORTS_LIMIT
        for d in t["depends_on"]:
            dt = tasks[d]
            part = f"## {d} [{dt['kind']}] {dt['title']}\n\n{dt.get('summary') or ''}\n"
            if dt["writes"] and dt.get("head") and dt["head"] != dt.get("base"):
                part += (f"\nIts changes are in your working directory already "
                         f"(`git diff {dt['base'][:12]}..{dt['head'][:12]}` shows them).\n")
            report, path = self.task_report(d)
            if not report:
                pass
            elif d not in t.get("needs_report", []):
                part += f"\nIts full report is at {path}. Read it only if your brief and this summary leave you short.\n"
            elif len(report) <= budget:
                budget -= len(report)
                part += f"\n<report>\n{report}\n</report>\n"
            else:
                part += f"\nYou need its full report, which is at {path}; read it.\n"
            out.append(part)
        return "\n".join(out)

    def reports_warning(self, tid):
        """Tells the orchestrator when a task is given more reading than a task should start with."""
        t = self.state["tasks"][tid]
        named = t.get("needs_report", [])
        size = sum(len(self.task_report(d)[0]) for d in named)
        if len(named) <= REPORTS_MANY and size <= REPORTS_LIMIT:
            return ""
        return (f" Note: its agent is given {len(named)} full report(s)"
                + (f", {size:,} characters so far," if size else "") + " to read before it writes a line. Name only "
                "the reports it cannot work without, and quote what it needs from the others in the brief.")

    # ---- prompts

    def p_orchestrator(self, turn):
        cfg = self.cfg
        if turn["reason"] == "start":
            why = "This is the first turn. Nothing has happened yet: there are no tasks and no notes."
        else:
            why = "Since the last orchestrator turn:\n\n" + (self.event_lines(turn["events"]) or "- nothing new")
        if turn.get("idle") and turn["reason"] != "start":
            why += ("\n\nNothing is running and nothing can start. The run only moves again if you add work, "
                    "unblock or retry a task, or finish the run.")
            streak = self.state.get("idle_streak", 0)
            if streak > 1:
                why += (f" This is turn {streak} in a row that began this way; after {MAX_IDLE_TURNS} the run is "
                        "stopped as stalled.")
        waited = turn.get("wait")
        if waited:
            asked = (f"The instance of turn {waited['turn']} asked to be started again when "
                     f"{'all' if waited['mode'] == 'all' else 'any'} of {', '.join(waited['tasks'])} had ended. ")
            early = "nothing is left running" if turn.get("idle") else "a task failed"
            why = asked + ("That has happened." if turn["reason"] == "wait" else
                           f"This instance is started before that, because {early}.") + "\n\n" + why
        ending = ("When the run is the way you want it, end with a short message for the person watching: what you "
                  "learned, what you changed in the run and why, and what you expect to happen next.")
        if cfg["wake"] == "each":
            wake = "You are started again whenever a task finishes or fails"
        elif cfg["wake"] == "idle":
            wake = "You are started again when a task fails and when nothing is left running"
        else:
            wake = ("You say when you are started again: `wait_for` names the tasks whose results you need before "
                    "you can decide anything more, and the next instance starts when they have ended. One also "
                    "starts whenever a task fails and when nothing is left running")
            ending = ("Before you end, call `wait_for` with the tasks whose results the next decision depends on, "
                      "unless nothing is pending or running. Wait for all the tasks of a stage rather than for "
                      "each one: without `wait_for` an instance is started after every task that finishes, and "
                      f"the run is stopped after {cfg['max_turns']} turns.\n\n" + ending)
        tiers = "\n".join(f"  - `{name}` ({cfg['tiers'][name]['model']}, {cfg['tiers'][name]['effort'] or 'default'} "
                          f"effort): {what}" for name, what in (
            ("deep", "the result is a decision that other work is built on (a design, a contract, a review), or "
                     "nothing after it checks it."),
            ("standard", "work from a precise brief, whose result a build, a test or a later task checks."),
            ("light", "gathering facts, taking inventory or following a recipe: cheap to do again if it is wrong.")))
        return f"""You are the orchestrator of an automated build run. Someone gave the run the goal below and left; nobody is available to answer questions. The run reaches that goal through tasks, each carried out by a separate agent, and your job is to decide what those tasks are. You do none of the work yourself: you cannot change the repository, and whatever you want investigated, built, checked or fixed has to become a task.

You are one in a series of orchestrator instances. An instance is started when something in the run changes; it looks at where the run stands, edits the run through the `{MCP_SERVER}` tools, and ends. You remember nothing of the earlier instances. What carries over is the run itself: its tasks and their reports, the notes, and the record of earlier turns. This is turn {turn['n']}.

# Goal

{self.goal_block()}

# Why you were started

{why}

# The run right now

<run>
{self.snapshot()}
</run>

`get_run` returns this again, up to date. `get_task` gives everything about one task, including its brief and its full report, and `get_agent` shows what an agent has been doing. Your working directory is a checkout of the integration branch, where the finished work of every task that changes the repository is merged; read it when you need to see the code as it stands. The files of the run are under {self.run_dir}.

# How the run works

- A task is one job for one agent. That agent starts from nothing: it gets the task's brief, the summary of each task it depends on, the full reports of those you name in `needs_report`, and a checkout of the repository. It cannot ask you or anyone else a question, and it cannot see or change the run.
- Every task has a tier, which decides how capable an agent it gets and what it costs:
{tiers}
- A task either changes the repository (`writes: true`) or only reports (`writes: false`). A writing task works in its own git worktree, branched from the integration branch at the moment it starts, and its changes are committed and merged into the integration branch when it finishes. A reporting task works in a scratch checkout that is thrown away; its report is its whole result. Investigation, design, review and verification are reporting tasks.
- A task starts by itself once every task it depends on is done, up to {cfg['max_parallel']} at a time. It then sees the work of its dependencies and of anything else merged before it started. Tasks you add or change in this turn start when your turn ends.
- A task that is done is final. To build on it, correct it or check it, add another task that depends on it. A pending task can be changed, a running one cancelled, and a failed or cancelled one changed and retried.
- {wake}. You never need to wait or poll: end your turn and the run carries on.

# How to decide

Work from evidence. The task reports and the code are the facts; the notes are what earlier instances made of them. Read the reports of the tasks that finished since the last turn before building on them, and when a task failed, find out why (`get_task`, `get_agent`) before deciding whether to retry it with a better brief, replace it, or change course.

Do not rush to a solution. A task that builds something before the problem is understood produces confident work on the wrong thing, and every later task inherits the mistake. So the first turns are for understanding: what the goal really asks for, what the repository already has, what is unknown, what "done" means here and how it will be checked. A quick look at the code is yours to take; anything deeper is a reporting task, and several can run side by side. A turn that adds only investigation tasks, or that changes nothing because the running tasks are the right ones, is a good turn. Add tasks that change the repository once the notes can state the definition of done and an approach that the findings support. The tools refuse a writing task while the notes are empty.

Plan as far as you can see and no further. You do not have to lay out the whole run now: add the tasks whose briefs you can write precisely today, and leave the rest to a later instance that will have their results in hand. When what you learn makes a pending task wrong, change it; when it makes one unnecessary, cancel it.

Write briefs for a reader who knows nothing. A brief says what to do and why, what exists that the task builds on (name the files), the exact names, paths and shapes of anything shared with other tasks, what is out of scope, and how the result is to be verified. Tasks with no dependency path between them run at the same time, which is what makes the run fast, so make the graph as wide as it safely can be: two such tasks must not edit the same files or rely on each other's output, and a piece that several tasks need belongs in an earlier task they all depend on.

Keep a task narrow: one package or one area of the code, with the files it works on named. An agent pays for everything it reads before it writes its first line, and again on every step after that, so what makes a task expensive is how much it has to take in, more than how much it has to produce. Quote in the brief the parts of earlier reports that the task needs (the contract, the names, the decision) instead of sending its agent to read them, and name a report in `needs_report` only when the task cannot be done without the whole of it. A task that needs several whole reports, or touches several areas, is two tasks.

Give each task the lowest tier that is safe for it, and say why in `tier_reason`. When you are unsure between two, take the higher. When a task fails, or a review finds real faults in its work, and the brief was not the cause, retry or redo it one tier up.

Nothing is checked unless you ask for it. A report saying that something works is a claim, not evidence. After work lands, add reporting tasks that review it against its brief and verify it against the goal (the build, the tests, the behaviour itself), and add tasks to fix what they find. Size the checking to the risk of the work.

Finish with `finish_run` only when it is true: `achieved` when verification of the merged result shows that the goal is met, `not_achieved` when it cannot be met and you can say why. Do not finish while a check you asked for is outstanding, and do not keep the run going with work the goal does not need.

# Notes

The notes are your memory: the next instance knows only what the run and the notes tell it. They should hold, briefly: what the goal requires, the definition of done and how it will be checked, the facts established so far and which task established them, the approach and what is planned next, decisions made and why, and open questions and risks. Give each of these a section of its own (a `##` heading), write the first version with `set_notes`, and after that keep them current with `edit_notes`, which replaces the one section you name and leaves the rest alone. Keep them short and true: every instance reads all of them, so replace what is out of date instead of adding to it, and do not turn them into a log.

# Ending your turn

{ending}

Rules:
- Do not create, edit or delete files, and do not commit. Change the run only through the `{MCP_SERVER}` tools.
- You run unattended and cannot ask questions. Where the goal leaves something open, make the most reasonable choice and record it in the notes."""

    def p_task(self, t):
        tid = t["id"]
        deps = self.dependency_results(t)
        if t["writes"]:
            where = """Your working directory is a git worktree made for this task, branched from the run's integration branch as it is now. When you finish, whatever you changed in it is committed and merged into that branch for you. Other tasks may be running at the same time in other worktrees and are merged the same way, so:

- Build what your brief says and stay inside its scope. Work that belongs to another task will collide with that task's agent.
- Follow the contracts in your brief exactly (names, signatures, paths, data shapes). Other tasks may be written against the same contracts right now.
- Follow the conventions of the code around you, and verify what you built: run the build and the tests your change could affect. Do not assume."""
        else:
            where = """Your working directory is a scratch checkout of the run's integration branch as it is now. This task produces a report, not changes: the checkout is thrown away when you finish, so you may build, run and experiment in it freely, but nothing you change there survives. Whatever the orchestrator should know has to be in your report."""
        brief = read_text(os.path.join(self.tasks_dir, tid + ".md")).strip()
        results = f"\n# Results of the tasks this one depends on\n\n{deps}" if deps else ""
        unmerged = " The changes of a failed task are not merged." if t["writes"] else ""
        return f"""You are one of the agents of an automated build run. The run works toward a goal through tasks: an orchestrator decides what the tasks are and reads what each one reports. You have one task. Do that task completely, and only that task.

# Your task: {tid} {t['title']}

<brief>
{brief}
</brief>
{results}
# Where this fits

For context only, you can read the overall goal at {os.path.join(self.run_dir, 'goal.md')} and the briefs of the other tasks at {self.tasks_dir}/<id>.md. Your brief is what you are asked to do; the goal is not.

{where}

# Your result

Finish by giving your structured result:

- `outcome`: `completed` when you did what the brief asks, `failed` when you could not.{unmerged}
- `summary`: two or three sentences on what you did or found. A task that builds on yours may be given only this, so put in it what such a task has to know first.
- `report`: the full account, in markdown. It is all the orchestrator and the tasks that depend on yours will see of your work, so it has to stand on its own: what you did or found, with the evidence (files and lines, the commands you ran and what they showed), what you decided and why, and what you could not do or are unsure of.

{WRITE_RULES}
- Everything you read stays with you for the rest of the task and is paid for on every step, so read what the task needs and not more. A subagent starts from nothing and has to read the code again: hand one a part only when that part is independent and you do not need what it reads yourself."""

    def p_merge(self, t, files):
        return f"""You are resolving a merge conflict in an automated build run.

Task {t['id']} ({t['title']}) was built on a branch of its own. While it was being built, other tasks were merged into the integration branch. The integration branch has just been merged into this task's worktree (your working directory) and these files conflict:

{chr(10).join('- ' + f for f in files)}

Resolve every conflict so that the result keeps the intent of both sides: this task's change and what the other tasks merged. Neither side may lose behaviour.

- The brief of this task: {os.path.join(self.tasks_dir, t['id'] + '.md')}
- The other briefs, to understand what the other side meant: {self.tasks_dir}
- `git log --merge` and `git diff` show both sides.
- When no conflict marker is left, build and run the tests the conflicting files affect.
- Do not run `git commit`, `git merge --abort` or `git reset`. Leave the resolved files in the working tree; the run concludes the merge.

Finish with a report: for each file, what conflicted and how you resolved it, and what you ran to verify it.

{WRITE_RULES}"""

    # ---- tasks

    def run_task(self, tid, cancel):
        """Carries one task through: worktree, agent, commit, merge. Each step is skipped once done."""
        t = self.state["tasks"][tid]
        prefix = agent_prefix(tid, t["attempt"])
        wt = os.path.join(self.run_dir, "worktrees", prefix)

        def check_cancel():
            if cancel.is_set():
                raise Cancelled()

        if not t.get("base"):
            with self.mut():
                t.update(base=git(self.repo, "rev-parse", self.int_branch), worktree=wt, started_at=now(),
                         branch=f"{BRANCH_NS}/{self.state['name']}/{prefix}" if t["writes"] else None)
        self.ensure_worktree(wt, t["branch"], t["base"])
        check_cancel()
        if not t.get("setup_done"):
            self.run_setup(wt, prefix)
            with self.mut():
                t["setup_done"] = True
        check_cancel()

        if not t.get("work_done"):
            agent_id = f"{prefix}-work"
            self.run_agent(agent_id, "task", tid, self.p_task(t), wt, t["tier"], schema=TASK_SCHEMA,
                           title=t["kind"], cancel=cancel)
            r = self.agent_result(agent_id)
            outcome = "failed" if r.get("outcome") == "failed" else "completed"
            write_text(os.path.join(self.agent_dir(agent_id), "report.md"),
                       f"**{outcome}**: {r.get('summary', '')}\n\n---\n\n{r.get('report', '')}")
            with self.mut():
                t.update(work_done=True, outcome=outcome, summary=str(r.get("summary") or ""))
        if t["outcome"] == "failed":
            raise TaskFailed(f"its agent reported that it could not do the task: {t['summary']}")

        if t["writes"]:
            with self.mut():
                t["status"] = "merging"
            check_cancel()     # past this point the task is no longer cancellable
            head = commit_all(wt, f"{tid}: {t['title']}", self.ident)
            with self.mut():
                t["head"] = head
            if head != t["base"]:
                self.merge_task(tid, wt)
        if not self.cfg["keep_worktrees"]:
            self.remove_worktree(wt)
        with self.mut():
            t.update(status="done", ended_at=now())
        self.event("task_done", tid, t["summary"])
        self.log(f"{tid}: done")

    def merge_task(self, tid, wt):
        """Merges a finished task into the integration branch, resolving conflicts on the task's side."""
        t = self.state["tasks"][tid]
        if t.get("merged"):
            return
        with self.git_lock:
            self.check_stop()

            def merge_into_integration():
                return git_rc(self.int_wt, *self.ident, "merge", "-q", "--no-ff", "-m",
                              f"Merge {tid}: {t['title']}", t["branch"])

            if in_merge(self.int_wt):       # an earlier attempt died in the middle of this step
                git(self.int_wt, "merge", "--abort")
            if not git_ok(self.repo, "merge-base", "--is-ancestor", t["branch"], self.int_branch):
                rc, out, err = (1, "", "") if in_merge(wt) else merge_into_integration()
                if rc:
                    # The branches conflict. Bring integration into the task's worktree, have an agent
                    # resolve it there, and merge the result, which then applies cleanly.
                    if in_merge(self.int_wt):
                        git(self.int_wt, "merge", "--abort")
                    if not in_merge(wt):
                        git_rc(wt, *self.ident, "merge", "--no-edit", self.int_branch)
                    files = git(wt, "diff", "--name-only", "--diff-filter=U").splitlines()
                    if files and not t.get("conflicts"):
                        with self.mut():
                            t["conflicts"] = files
                    if not in_merge(wt):
                        raise RuntimeError(f"merging {tid} into {self.int_branch} failed:\n{(err or out).strip()}")
                    files = t.get("conflicts", [])
                    self.log(f"{tid}: merge conflict in {len(files)} file(s), resolving")
                    self.run_agent(f"{agent_prefix(tid, t['attempt'])}-merge", "merge", tid, self.p_merge(t, files),
                                   wt, MERGE_TIER, title="resolve merge conflict")
                    left = [f for f in files if re.search(r"^(<<<<<<<|>>>>>>>) ", read_text(os.path.join(wt, f)), re.M)]
                    if left:
                        raise RuntimeError(f"conflict markers remain in {', '.join(left)}")
                    commit_all(wt, f"Merge {self.int_branch} into {tid}", self.ident)
                    rc, out, err = merge_into_integration()
                    if rc:
                        raise RuntimeError(f"merging {tid} into {self.int_branch} failed:\n{(err or out).strip()}")
            with self.mut():
                t["merged"] = git(self.repo, "rev-parse", self.int_branch)

    def discard(self, tid):
        """Puts aside what a failed or cancelled task left: its changes stay on its branch, unmerged."""
        t = self.state["tasks"][tid]
        wt = t.get("worktree")
        if not wt or not os.path.exists(os.path.join(wt, ".git")):
            return
        with contextlib.suppress(Exception):
            if in_merge(wt):
                git(wt, "merge", "--abort")
            if t.get("branch"):
                commit_all(wt, f"{tid}: unfinished work ({t['title']})", self.ident)
        if not self.cfg["keep_worktrees"]:
            self.remove_worktree(wt)

    def task_worker(self, tid, cancel):
        t = self.state["tasks"][tid]
        try:
            self.run_task(tid, cancel)
        except Stop:
            pass                 # stays as it is in the state; the next process picks it up
        except Cancelled:
            self.discard(tid)
            with self.mut():
                t.update(status="cancelled", ended_at=now())
            self.log(f"{tid}: cancelled")
        except Exception as e:
            if not isinstance(e, (AgentError, TaskFailed)):
                self.log(f"{tid}: {traceback.format_exc()}")
            self.discard(tid)
            with self.mut():
                t.update(status="failed", error=str(e), ended_at=now())
            self.event("task_failed", tid, str(e))
            self.log(f"{tid}: failed: {e}")
        finally:
            with self.cond:
                self.cond.notify_all()

    # ---- orchestrator turns

    def run_turn(self, n):
        turn = self.state["turns"][n - 1]
        agent_id = turn["agent"]
        d = self.agent_dir(agent_id)
        token = secrets.token_urlsafe(24)
        config = os.path.join(d, "mcp.json")       # exists, and its token is valid, only while the turn runs
        fd = os.open(config, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600)
        with os.fdopen(fd, "w") as f:
            json.dump({"mcpServers": {MCP_SERVER: {"type": "http", "url": self.mcp_url,
                                                   "headers": {"Authorization": f"Bearer {token}"}}}}, f)
        with self.lock:
            self.tokens[token] = n
        try:
            self.refresh_orchestrator_worktree()
            with self.lock:
                prompt = self.p_orchestrator(turn)
            self.run_agent(agent_id, "orchestrator", "orchestrator", prompt, self.orch_wt, ORCHESTRATOR_TIER,
                           add_dirs=[self.run_dir], title=f"turn {n}", mcp_config=config,
                           disallowed=ORCHESTRATOR_DISALLOWED)
        finally:
            with self.lock:
                self.tokens.pop(token, None)
            with contextlib.suppress(OSError):
                os.remove(config)
        with self.mut() as s:
            turn.update(status="done", ended_at=now(), summary=self.agent_report(agent_id))
            for t in s["tasks"].values():      # what the turn added or changed may start now
                t.pop("held", None)
        changes = [o for o in turn["ops"] if o["op"] not in READ_OPS and not o.get("error")]
        self.log(f"turn {n}: done, {len(changes)} change(s) to the run")

    def turn_worker(self, n):
        try:
            self.run_turn(n)
        except Stop:
            pass                 # the turn stays 'running' in the state and is resumed by the next process
        except Exception as e:
            if not isinstance(e, AgentError):
                self.log(traceback.format_exc())
            with self.mut() as s:
                s["turns"][n - 1].update(status="failed", ended_at=now(), error=str(e))
            self.end_run("error", f"orchestrator turn {n}: {e}")
        finally:
            with self.cond:
                self.cond.notify_all()

    def wait_met(self):
        """Whether the tasks the last turn said it waits for have ended. None when it named none."""
        wait, tasks = self.state.get("wait"), self.state["tasks"]
        if not wait:
            return None
        ended = [tasks[t]["status"] in ("done", "failed", "cancelled") for t in wait["tasks"]]
        return all(ended) if wait["mode"] == "all" else any(ended)

    def next_turn(self, busy):
        """Starts a turn record when the orchestrator is due, and returns its number.

        Called with the lock held, when no turn is running. busy: a task is running or can start.
        """
        s, cfg = self.state, self.cfg
        turns = s["turns"]
        if turns and turns[-1]["status"] == "running":
            return turns[-1]["n"]            # interrupted with the last process: resume it
        unseen = [e for e in s["events"] if e["seq"] > s["seen_seq"]]
        idle = not busy
        failed = any(e["type"] == "task_failed" for e in unseen)
        # the tasks the last turn said it waits for; without any, "declared" behaves like "each"
        met = self.wait_met() if cfg["wake"] == "declared" else None
        if not turns:
            reason = "start"
        elif met:
            reason = "wait"
        elif idle:
            reason = "idle"
        elif unseen and (failed or cfg["wake"] == "each" or (cfg["wake"] == "declared" and met is None)):
            reason = "events"
        else:
            return None
        if idle and turns and s.get("idle_streak", 0) >= MAX_IDLE_TURNS:
            self.end_run("stalled", f"the orchestrator was started {MAX_IDLE_TURNS} times in a row with nothing "
                                    "running and neither added work nor finished the run")
            return None
        if len(turns) >= cfg["max_turns"]:
            self.end_run("stalled", f"reached the limit of {cfg['max_turns']} orchestrator turns (--max-turns)")
            return None
        if cfg["max_cost"]:
            spent = sum(a.get("cost_usd") or 0 for a in load_agents(self.run_dir))
            if spent >= cfg["max_cost"]:
                self.end_run("stalled", f"spent ${spent:.2f}, over the limit of ${cfg['max_cost']:.2f} (--max-cost)")
                return None
        n = len(turns) + 1
        with self.mut():
            if idle and turns:
                s["idle_streak"] = s.get("idle_streak", 0) + 1
            turns.append({"n": n, "agent": f"turn-{n:03d}", "reason": reason, "idle": idle, "events": unseen,
                          "wait": s.get("wait"), "status": "running", "started_at": now(), "ops": []})
            s["wait"] = None             # a wait holds until the next turn starts, whatever started it
            if unseen:
                s["seen_seq"] = unseen[-1]["seq"]
        self.log(f"turn {n}: starting the orchestrator ({reason}"
                 + (": " + ", ".join(f"{e['task']} {e['type'][5:]}" for e in unseen) if unseen else "") + ")")
        return n

    def loop(self):
        """Starts tasks as they become ready and the orchestrator as it becomes due, until it finishes the run."""
        tasks = self.state["tasks"]
        workers, turn_thread = {}, None

        def start_task(tid):
            cancel = self.cancels[tid] = threading.Event()
            th = threading.Thread(target=self.task_worker, args=(tid, cancel), name=tid, daemon=True)
            workers[tid] = th
            th.start()

        with self.cond:
            while not self.stop.is_set():
                s = self.state
                for tid in [k for k, th in workers.items() if not th.is_alive()]:
                    workers.pop(tid)
                    self.cancels.pop(tid, None)
                turn_live = turn_thread is not None and turn_thread.is_alive()
                if s.get("result") and not turn_live and not workers:
                    break
                for tid in s["task_order"]:        # tasks the last process left in the middle
                    if tasks[tid]["status"] in ACTIVE and tid not in workers:
                        start_task(tid)
                ready = [tid for tid in s["task_order"] if self.is_ready(tid) and tid not in workers]
                for tid in ready[: max(0, self.cfg["max_parallel"] - len(workers))]:
                    with self.mut():
                        tasks[tid]["status"] = "running"
                        s["idle_streak"] = 0
                    self.log(f"{tid}: starting ({tasks[tid]['kind']}) {tasks[tid]['title']}")
                    start_task(tid)
                if not turn_live and not s.get("result") and not self.stop.is_set():
                    n = self.next_turn(busy=bool(workers))
                    if n:
                        turn_thread = threading.Thread(target=self.turn_worker, args=(n,), name=f"turn-{n}",
                                                       daemon=True)
                        turn_thread.start()
                self.cond.wait(1)
            while any(th.is_alive() for th in workers.values()) or (turn_thread and turn_thread.is_alive()):
                self.cond.wait(1)          # on a stop, wait for the workers to unwind
        self.check_stop()

    # ---- MCP: what the orchestrator can do to the run

    def start_mcp(self):
        server = ThreadingHTTPServer(("127.0.0.1", 0), make_mcp_handler(self))
        server.daemon_threads = True
        self.mcp_url = f"http://127.0.0.1:{server.server_address[1]}/mcp"
        threading.Thread(target=server.serve_forever, name="mcp", daemon=True).start()
        return server

    def mcp_message(self, turn_n, msg):
        """Answers one JSON-RPC message from an orchestrator's MCP client. Notifications get no answer."""
        if not isinstance(msg, dict) or "id" not in msg or "method" not in msg:
            return None
        method, params = msg["method"], msg.get("params") or {}
        if method == "initialize":
            result = {"protocolVersion": params.get("protocolVersion") or "2025-03-26",
                      "capabilities": {"tools": {}}, "serverInfo": {"name": MCP_SERVER, "version": "3"}}
        elif method == "ping":
            result = {}
        elif method == "tools/list":
            result = {"tools": self.tools()}
        elif method == "tools/call":
            text, failed = self.call_tool(turn_n, str(params.get("name")), params.get("arguments") or {})
            result = {"content": [{"type": "text", "text": text}], "isError": failed}
        else:
            return {"jsonrpc": "2.0", "id": msg["id"], "error": {"code": -32601, "message": f"unknown method {method}"}}
        return {"jsonrpc": "2.0", "id": msg["id"], "result": result}

    def tools(self):
        """The tools of this run: wait_for only where the orchestrator decides when it is started."""
        return [t for t in TOOLS if t["name"] != "wait_for" or self.cfg["wake"] == "declared"]

    def call_tool(self, turn_n, name, args):
        """Runs one operation for a turn and records it in that turn. Returns (text, failed)."""
        fn = getattr(self, "op_" + name, None) if name in [t["name"] for t in self.tools()] else None
        rec = {"op": name, "t": now()}
        try:
            if fn is None:
                raise OpError(f"there is no tool {name}")
            if not isinstance(args, dict):
                raise OpError("the arguments must be an object")
            if self.state.get("result") and name not in READ_OPS:
                raise OpError("the run is finished; nothing more can be changed")
            text, failed = fn(turn_n, args, rec), False
        except OpError as e:
            text, failed = str(e), True
        except Exception as e:
            self.log(f"turn {turn_n}: {name} raised: {traceback.format_exc()}")
            text, failed = f"internal error in {name}: {e}", True
        if failed:
            rec["error"] = clip(text, 400)
            if isinstance(args, dict) and args.get("id"):
                rec.setdefault("task", str(args["id"]))
        with self.mut() as s:
            s["turns"][turn_n - 1]["ops"].append(rec)
            if not failed and name not in READ_OPS:
                s["rev"] += 1
        if name not in READ_OPS:
            self.log(f"turn {turn_n}: {name}" + (f" {rec['task']}" if rec.get("task") else "")
                     + (f" refused: {clip(text, 200)}" if failed else ""))
        return text, failed

    def need_task(self, args):
        tid = str(args.get("id") or "")
        if tid not in self.state["tasks"]:
            known = ", ".join(self.state["task_order"]) or "there are no tasks yet"
            raise OpError(f"there is no task {tid!r} ({known})")
        return tid

    def check_definition(self, args, tid=None):
        """Validates the fields of a task given to add_task or update_task. Returns the clean values."""
        tasks, out = self.state["tasks"], {}
        if "tier" in args:
            if args["tier"] not in TIERS:
                raise OpError(f"tier must be one of {', '.join(TIERS)}")
            out["tier"] = args["tier"]
        if "tier_reason" in args:
            out["tier_reason"] = " ".join(str(args["tier_reason"]).split())
            if not out["tier_reason"]:
                raise OpError("tier_reason is empty: say in a sentence why this tier is enough for the task")
        if "title" in args:
            out["title"] = " ".join(str(args["title"]).split())
            if not out["title"]:
                raise OpError("the title is empty")
        if "brief" in args:
            out["brief"] = str(args["brief"]).strip()
            if len(out["brief"]) < 40:
                raise OpError("the brief is too short to work from: its agent knows nothing but the brief")
        if "kind" in args:
            out["kind"] = slugify(str(args["kind"]), 24)
            if not KIND_RE.match(out["kind"]):
                raise OpError("kind must be one short word, e.g. research, implement, review")
        if "writes" in args:
            if not isinstance(args["writes"], bool):
                raise OpError("writes must be true or false")
            out["writes"] = args["writes"]
        if "depends_on" in args:
            deps = args["depends_on"]
            if not isinstance(deps, list) or not all(isinstance(d, str) for d in deps):
                raise OpError("depends_on must be a list of task ids")
            deps = list(dict.fromkeys(deps))
            unknown = [d for d in deps if d not in tasks]
            if unknown:
                raise OpError(f"depends_on names tasks that do not exist: {', '.join(unknown)}")
            if tid and tid in deps:
                raise OpError(f"{tid} cannot depend on itself")
            path = cycle_through(tasks, tid, deps) if tid else None
            if path:
                raise OpError("that would make a dependency cycle: " + " -> ".join(path))
            out["depends_on"] = deps
        if "needs_report" in args:
            named = args["needs_report"]
            if not isinstance(named, list) or not all(isinstance(d, str) for d in named):
                raise OpError("needs_report must be a list of task ids")
            named = list(dict.fromkeys(named))
            deps = out["depends_on"] if "depends_on" in out else tasks[tid]["depends_on"] if tid else []
            outside = [d for d in named if d not in deps]
            if outside:
                raise OpError(f"needs_report names tasks this one does not depend on: {', '.join(outside)}. "
                              "Add them to depends_on as well.")
            out["needs_report"] = named
        return out

    def save_brief(self, t, brief):
        t["rev"] = t.get("rev", 0) + 1
        write_text(os.path.join(self.tasks_dir, "history", f"{t['id']}.r{t['rev']}.md"), brief)
        write_text(os.path.join(self.tasks_dir, t["id"] + ".md"), brief)

    def dependency_warning(self, tid):
        t, tasks = self.state["tasks"][tid], self.state["tasks"]
        bad = [d for d in t["depends_on"] if tasks[d]["status"] in ("failed", "cancelled")]
        if not bad:
            return ""
        names = ", ".join(f"{d} ({tasks[d]['status']})" for d in bad)
        return (f" Note: it depends on {names} and will not start until that is retried and done, or the "
                "dependency is removed.")

    def op_get_run(self, turn_n, args, rec):
        with self.mut() as s:
            turn = s["turns"][turn_n - 1]
            unseen = [e for e in s["events"] if e["seq"] > s["seen_seq"]]
            if unseen:                       # told now, so they do not start another turn by themselves
                turn["events"] += unseen
                s["seen_seq"] = unseen[-1]["seq"]
            text = self.snapshot()
        if unseen:
            text += "\n\n## New since you last looked\n\n" + self.event_lines(unseen)
        return text

    def op_get_task(self, turn_n, args, rec):
        tid = self.need_task(args)
        rec["task"] = tid
        with self.lock:
            s = self.state
            t = dict(s["tasks"][tid])
            dependents = [x for x in s["task_order"] if tid in s["tasks"][x]["depends_on"]]
            state = self.task_state(s["tasks"][tid])
        agents = [a for a in load_agents(self.run_dir) if a["unit"] == tid]
        out = [f"# {tid} [{t['kind']}, {'writes' if t['writes'] else 'reports only'}] {t['title']}", "",
               f"Status: {state}" + (f" (attempt {t['attempt']})" if t["attempt"] > 1 else ""),
               f"Tier: {t['tier']} ({t['tier_reason']})",
               f"Depends on: {', '.join(t['depends_on']) or 'nothing'}"
               + (f" (given the full report of {', '.join(t['needs_report'])})" if t["needs_report"] else ""),
               f"Needed by: {', '.join(dependents) or 'nothing'}",
               f"Added in turn {t['added_turn']}" + (f", changed in turn {', '.join(map(str, t['changed_turns']))}"
                                                     if t.get("changed_turns") else "")]
        if t.get("started_at"):
            out.append(f"Time: {fmt_duration((t.get('ended_at') or now()) - t['started_at'])}")
        if t.get("error"):
            out.append(f"Error: {t['error']}")
        if t.get("branch") and t.get("head"):
            out.append(f"Commits: {t['base'][:12]}..{t['head'][:12]} on branch {t['branch']}"
                       + (", merged into the integration branch" if t.get("merged") else
                          ", not merged" if t["head"] != t["base"] else ", no changes"))
            if t["head"] != t["base"]:
                out += ["", "```", git_rc(self.repo, "diff", "--stat", f"{t['base']}..{t['head']}")[1].rstrip(), "```"]
        elif t.get("branch") and t["status"] in ("failed", "cancelled"):
            out.append(f"Whatever it changed before it stopped is on branch {t['branch']}, not merged.")
        if t.get("conflicts"):
            out.append(f"Merge conflicts resolved by an agent in: {', '.join(t['conflicts'])}")
        if agents:
            out += ["", "Agents:"] + [f"- {a['id']} ({a['role']}, {a.get('model')}): {a['status']}, "
                                      f"{a.get('num_turns', 0)} turns, ${a.get('cost_usd') or 0:.2f}, context up to "
                                      f"{(a.get('peak_context') or 0) // 1000}k tokens" for a in agents]
        for p in t.get("past", []):
            out.append(f"\nEarlier attempt {p['attempt']} ({p.get('tier')}): {p['status']}. "
                       f"{clip(p.get('error') or p.get('summary'), 600)}"
                       + (f" Its unfinished work is on branch {p['branch']}." if p.get("branch") else ""))
        out += ["", "## Brief", "", read_text(os.path.join(self.tasks_dir, tid + ".md")).strip()]
        agent_id = agent_prefix(tid, t["attempt"]) + "-work"
        report = str(self.agent_result(agent_id).get("report") or "").strip()
        if t.get("work_done"):
            out += ["", f"## Result ({t.get('outcome')})", "", t.get("summary") or "", ""]
            if len(report) <= 2 * INLINE_LIMIT:
                out.append(report or "(no report)")
            else:
                out.append(f"The report is long; read it at {os.path.join(self.agents_dir, agent_id, 'report.md')}")
        elif t["status"] in ACTIVE:
            out += ["", f"It has no result yet. `get_agent` with agent {agent_id} shows what it is doing."]
        return "\n".join(out)

    def op_get_agent(self, turn_n, args, rec):
        agent_id = str(args.get("agent") or "")
        rec["agent"] = agent_id
        d = os.path.join(self.agents_dir, agent_id)
        meta = read_json(os.path.join(d, "meta.json")) if ID_RE.match(agent_id) else None
        if not meta:
            ids = ", ".join(a["id"] for a in load_agents(self.run_dir)) or "none yet"
            raise OpError(f"there is no agent {agent_id!r}. Agents: {ids}")
        try:
            last = max(1, min(60, int(args.get("last") or 15)))
        except (TypeError, ValueError):
            raise OpError("last must be a number")
        out = [f"# Agent {agent_id} ({meta['role']} of {meta['unit']}): {meta['status']}",
               f"{meta.get('num_turns', 0)} turns, ${meta.get('cost_usd') or 0:.2f}, "
               f"{fmt_duration((meta.get('ended_at') or now()) - (meta.get('started_at') or now()))}, "
               f"{len(meta['attempts'])} attempt(s)"]
        if meta.get("error"):
            out.append(f"Error: {meta['error']}")
        for a in meta["attempts"]:
            if a.get("error"):
                out.append(f"Attempt {a['n']} ended with: {clip(a['error'], 400)}")
        items = agent_activity(d, last)
        out += ["", f"## Its last {len(items)} steps", ""] + ([f"- {i}" for i in items] or ["(nothing yet)"])
        if meta["status"] == "done":
            out += ["", "## Its report", "", clip(self.agent_report(agent_id), INLINE_LIMIT)]
        return "\n".join(out)

    def op_set_notes(self, turn_n, args, rec):
        notes = str(args.get("notes") or "").strip()
        if not notes:
            raise OpError("the notes are empty")
        return self.save_notes(turn_n, notes, rec, "Notes saved")

    def op_edit_notes(self, turn_n, args, rec):
        heading = str(args.get("heading") or "")
        with self.lock:
            notes, did = replace_section(self.state["notes"], heading, str(args.get("text") or ""))
            rec["heading"] = clip(heading.strip(), 120)
            return self.save_notes(turn_n, notes, rec, f"Section {heading.strip()!r} {did}")

    def save_notes(self, turn_n, notes, rec, done):
        with self.mut() as s:
            s.update(notes=notes, notes_turn=turn_n)
        write_text(os.path.join(self.run_dir, "notes.md"), notes + "\n")
        rec["size"] = len(notes)
        over = (f" They are over {NOTES_LIMIT:,} characters, and every instance reads all of them: shorten them, "
                "keeping what the next instance needs to decide.") if len(notes) > NOTES_LIMIT else ""
        return f"{done}; the notes are {len(notes):,} characters.{over}"

    def op_add_task(self, turn_n, args, rec):
        missing = [k for k in ("title", "brief", "kind", "writes", "tier", "tier_reason") if k not in args]
        if missing:
            raise OpError(f"missing: {', '.join(missing)}")
        with self.mut() as s:
            d = self.check_definition(dict(args, depends_on=args.get("depends_on") or [],
                                           needs_report=args.get("needs_report") or []))
            if d["writes"] and not s["notes"].strip():
                raise OpError(" ".join(NOTES_GATE.split()))
            tid = f"T{len(s['task_order']) + 1:02d}"
            t = {"id": tid, "title": d["title"], "kind": d["kind"], "writes": d["writes"], "tier": d["tier"],
                 "tier_reason": d["tier_reason"], "depends_on": d["depends_on"], "needs_report": d["needs_report"],
                 "status": "pending", "attempt": 1, "added_turn": turn_n, "changed_turns": [], "held": turn_n,
                 "created_at": now()}
            self.save_brief(t, d["brief"])
            s["tasks"][tid] = t
            s["task_order"].append(tid)
            warn = self.dependency_warning(tid) + self.reports_warning(tid)
        rec.update(task=tid, title=d["title"], kind=d["kind"], writes=d["writes"], tier=d["tier"],
                   tier_reason=clip(d["tier_reason"], 300), depends_on=d["depends_on"],
                   needs_report=d["needs_report"])
        return f"Added {tid}: {d['title']}. It starts after your turn ends, once its dependencies are done.{warn}"

    def op_update_task(self, turn_n, args, rec):
        tid = self.need_task(args)
        rec["task"] = tid
        with self.mut() as s:
            t = s["tasks"][tid]
            if t["status"] in ACTIVE:
                raise OpError(f"{tid} is {t['status']}. Cancel it first if it must change, then retry it.")
            if t["status"] == "done":
                raise OpError(f"{tid} is done and final. Add a task that depends on it instead.")
            d = self.check_definition({k: v for k, v in args.items() if k != "id"}, tid)
            if not d:
                raise OpError("nothing to change: give title, brief, kind, writes, tier, tier_reason, depends_on "
                              "or needs_report")
            if d.get("writes") and not s["notes"].strip():
                raise OpError(" ".join(NOTES_GATE.split()))
            brief = d.pop("brief", None)
            if "depends_on" in d and "needs_report" not in d:      # a report of a task it no longer depends on
                kept = [x for x in t["needs_report"] if x in d["depends_on"]]
                if kept != t["needs_report"]:
                    d["needs_report"] = kept
            changed = [k for k, v in d.items() if t[k] != v]
            t.update(d)
            if brief is not None and brief != read_text(os.path.join(self.tasks_dir, tid + ".md")).strip():
                self.save_brief(t, brief)
                changed.append("brief")
            if not changed:
                raise OpError(f"{tid} already is as you describe it")
            if turn_n not in t["changed_turns"]:
                t["changed_turns"].append(turn_n)
            if t["status"] == "pending":
                t["held"] = turn_n
            warn = self.dependency_warning(tid) + self.reports_warning(tid)
            retry = "" if t["status"] == "pending" else f" It is {t['status']}: call retry_task to queue it again."
        rec.update(changed=changed, title=t["title"], tier=t["tier"], depends_on=t["depends_on"],
                   needs_report=t["needs_report"])
        return f"Updated {tid} ({', '.join(changed)}).{retry}{warn}"

    def op_cancel_task(self, turn_n, args, rec):
        tid = self.need_task(args)
        reason = str(args.get("reason") or "").strip()
        rec.update(task=tid, reason=clip(reason, 400))
        with self.cond:
            t = self.state["tasks"][tid]
            st = t["status"]
            if st == "done":
                raise OpError(f"{tid} is done; its work is already part of the run. Add a task to change it.")
            if st == "merging":
                raise OpError(f"{tid} has finished and is being merged; it can no longer be cancelled.")
            if st == "cancelled":
                raise OpError(f"{tid} is already cancelled")
            if st == "running":
                cancel = self.cancels.get(tid)
                if cancel is None:
                    raise OpError(f"{tid} is starting; try again in a moment")
                cancel.set()
                with self.procs_lock:
                    pids = [pid for pid, unit in self.procs.items() if unit == tid]
                for pid in pids:
                    kill_tree(pid)
                stopped = self.cond.wait_for(lambda: t["status"] not in ACTIVE or self.stop.is_set(), timeout=90)
                if t["status"] != "cancelled":
                    raise OpError(f"{tid} is now {t['status']}" if stopped else
                                  f"{tid} is being stopped but has not let go yet; look again with get_run")
            with self.mut():
                t.update(status="cancelled", cancel_reason=reason, cancelled_turn=turn_n,
                         ended_at=t.get("ended_at") or now())
                t.pop("held", None)
            order, tasks = self.state["task_order"], self.state["tasks"]
            stuck = [x for x in order if tid in tasks[x]["depends_on"] and tasks[x]["status"] == "pending"]
        note = f" {', '.join(stuck)} depend on it and cannot start until you change or cancel them." if stuck else ""
        return f"Cancelled {tid}.{note}"

    def op_retry_task(self, turn_n, args, rec):
        tid = self.need_task(args)
        reason = str(args.get("reason") or "").strip()
        rec.update(task=tid, reason=clip(reason, 400))
        tier = args.get("tier")
        if tier is not None and tier not in TIERS:
            raise OpError(f"tier must be one of {', '.join(TIERS)}")
        with self.mut() as s:
            t = s["tasks"][tid]
            if t["status"] not in ("failed", "cancelled"):
                raise OpError(f"{tid} is {t['status']}; only a failed or cancelled task can be retried")
            past = {k: t.pop(k) for k in ATTEMPT_FIELDS if k in t}
            past.update(attempt=t["attempt"], status=t["status"], tier=t["tier"])
            if tier and tier != t["tier"]:       # the new attempt is a new agent, so it can run on another model
                t.update(tier=tier, tier_reason=reason or t["tier_reason"])
            t.setdefault("past", []).append(past)
            for k in ("cancel_reason", "cancelled_turn"):
                t.pop(k, None)
            t.update(status="pending", attempt=t["attempt"] + 1, held=turn_n)
            if turn_n not in t["changed_turns"]:
                t["changed_turns"].append(turn_n)
            warn = self.dependency_warning(tid)
        rec.update(attempt=t["attempt"], tier=t["tier"])
        return f"{tid} is queued again as attempt {t['attempt']} ({t['tier']}), from a fresh checkout.{warn}"

    def op_wait_for(self, turn_n, args, rec):
        ids, mode = args.get("tasks"), args.get("mode") or "all"
        if not isinstance(ids, list) or not ids or not all(isinstance(x, str) for x in ids):
            raise OpError("tasks must be a list of task ids, with at least one")
        if mode not in ("all", "any"):
            raise OpError("mode must be all or any")
        ids = list(dict.fromkeys(ids))
        with self.mut() as s:
            unknown = [x for x in ids if x not in s["tasks"]]
            if unknown:
                raise OpError(f"there is no task {', '.join(unknown)} ({', '.join(s['task_order']) or 'no tasks yet'})")
            over = [f"{x} ({s['tasks'][x]['status']})" for x in ids if s["tasks"][x]["status"] not in ("pending", *ACTIVE)]
            if over:
                raise OpError(f"{', '.join(over)} already ended, so there is nothing to wait for: its result is "
                              "there to read now (get_task). Name only tasks that are pending or running.")
            s["wait"] = {"tasks": ids, "mode": mode, "turn": turn_n}
        rec.update(tasks=ids, mode=mode)
        return (f"After this turn the next instance starts when {'all' if mode == 'all' else 'any'} of "
                f"{', '.join(ids)} ended, or earlier if a task fails or nothing is left running.")

    def op_finish_run(self, turn_n, args, rec):
        outcome, summary = args.get("outcome"), str(args.get("summary") or "").strip()
        if outcome not in ("achieved", "not_achieved"):
            raise OpError("outcome must be achieved or not_achieved")
        if not summary:
            raise OpError("the summary is empty")
        with self.mut() as s:
            open_ = [tid for tid in s["task_order"] if s["tasks"][tid]["status"] in ("pending", *ACTIVE)]
            if open_:
                raise OpError(f"{', '.join(open_)} are not finished. Let them finish and decide then, or cancel "
                              "the ones the run no longer needs.")
            s["result"] = {"outcome": outcome, "summary": summary, "turn": turn_n, "at": now()}
        rec.update(outcome=outcome, summary=clip(summary, 600))
        return "The run ends when your turn does."

    # ---- run

    def run(self):
        with self.mut() as s:
            s.update(status="running", pid=os.getpid(), error=None, last_started_at=now(), idle_streak=0)
        self.reap_stale_agents()
        try:
            self.ensure_worktree(self.int_wt, self.int_branch, self.state["base_ref"])
            self.start_mcp()
            self.loop()
            result = self.state["result"]
            if not self.cfg["keep_worktrees"]:
                self.remove_worktree(self.orch_wt)
            achieved = result["outcome"] == "achieved"
            with self.mut() as s:
                s.update(status="completed" if achieved else "gave_up", ended_at=now(),
                         result_head=git(self.repo, "rev-parse", self.int_branch))
            self.log(("completed" if achieved else "ended without reaching the goal")
                     + f". The result is on branch {self.int_branch} ({self.int_wt}).")
            if achieved:
                self.log(f"to take it: git merge {self.int_branch}")
            return 0 if achieved else 2
        except Stop:
            pass
        except Exception as e:
            self.log(traceback.format_exc())
            self.end_run("error", str(e))
        status, message = self.failure or ("interrupted", "stopped by a signal")
        with self.mut() as s:
            s.update(status=status, error=message)
        self.log(f"{status}: {message}")
        self.log("rerun the same command to continue from where it stopped")
        return {"stalled": 3, "error": 1}.get(status, 130)


def make_mcp_handler(run):
    """The run's MCP server: streamable HTTP, answering every request with one JSON body."""

    class Handler(BaseHTTPRequestHandler):
        protocol_version = "HTTP/1.1"

        def log_message(self, *args):
            pass

        def reply(self, code, body=None, headers=()):
            data = b"" if body is None else json.dumps(body).encode("utf-8")
            self.send_response(code)
            if body is not None:
                self.send_header("Content-Type", "application/json")
            for k, v in headers:
                self.send_header(k, v)
            self.send_header("Content-Length", str(len(data)))
            self.end_headers()
            self.wfile.write(data)

        def do_GET(self):          # there is no server-initiated stream
            self.reply(405, headers=[("Allow", "POST")])

        do_DELETE = do_GET

        def do_POST(self):
            try:
                raw = self.rfile.read(int(self.headers.get("Content-Length") or 0))
                auth = self.headers.get("Authorization") or ""
                with run.lock:
                    turn_n = run.tokens.get(auth[7:].strip() if auth.startswith("Bearer ") else "")
                if urlparse(self.path).path != "/mcp":
                    return self.reply(404)
                if turn_n is None:      # only an orchestrator turn that is running holds a valid token
                    return self.reply(403, {"jsonrpc": "2.0", "id": None,
                                            "error": {"code": -32001, "message": "not an active orchestrator turn"}})
                try:
                    msg = json.loads(raw)
                except ValueError:
                    return self.reply(400, {"jsonrpc": "2.0", "id": None,
                                            "error": {"code": -32700, "message": "parse error"}})
                answers = [a for a in (run.mcp_message(turn_n, m) for m in (msg if isinstance(msg, list) else [msg]))
                           if a is not None]
                if not answers:
                    return self.reply(202)
                self.reply(200, answers if isinstance(msg, list) else answers[0])
            except (BrokenPipeError, ConnectionResetError):
                pass

    return Handler


# ---------------------------------------------------------------- reports

def load_agents(run_dir):
    agents_dir = os.path.join(run_dir, "agents")
    out = []
    if os.path.isdir(agents_dir):
        for name in os.listdir(agents_dir):
            meta = read_json(os.path.join(agents_dir, name, "meta.json"))
            if meta:
                out.append(meta)
    return sorted(out, key=lambda m: m.get("created_at") or 0)


def summary_text(run_dir):
    s = read_json(os.path.join(run_dir, "state.json"))
    if not s:
        return f"no run at {run_dir}"
    agents = load_agents(run_dir)
    cost = {}
    for a in agents:
        cost[a["unit"]] = cost.get(a["unit"], 0) + (a.get("cost_usd") or 0)
    alive = pid_alive(s.get("pid")) and s["status"] == "running"
    lines = [f"run {s['name']}: {s['status']}" + ("" if alive or s["status"] != "running" else " (process not running)")
             + f", {len(s['turns'])} orchestrator turn(s)",
             f"repo {s['repo']}, branch {s['integration_branch']}",
             f"agents {len(agents)}, cost ${sum(cost.values()):.2f} (orchestrator ${cost.get('orchestrator', 0):.2f})"]
    if s.get("error"):
        lines.append(f"reason: {s['error']}")
    lines += ["", f"{'task':<6}{'status':<11}{'kind':<12}{'tier':<10}{'added':<9}{'cost':>8}  title / depends on"]
    for tid in s["task_order"]:
        t = s["tasks"][tid]
        deps = f" <- {','.join(t['depends_on'])}" if t["depends_on"] else ""
        lines.append(f"{tid:<6}{t['status']:<11}{t['kind'][:11]:<12}{t['tier']:<10}{'turn ' + str(t['added_turn']):<9}"
                     f"{cost.get(tid, 0):>8.2f}  {t['title']}{deps}")
    if not s["task_order"]:
        lines.append("(no tasks yet)")
    lines += usage_lines(s, agents)
    running = [a["id"] for a in agents if a["status"] == "running"]
    if running:
        lines += ["", "running agents: " + ", ".join(running)]
    if s.get("result"):
        lines += ["", f"The orchestrator finished the run in turn {s['result']['turn']}: {s['result']['outcome']}",
                  "", s["result"]["summary"]]
    return "\n".join(lines)


def usage_lines(s, agents):
    """Where the money went: by tier and by kind of task, how large the contexts grew, and how many
    turns left the plan as it was. These are what a change to the run's policies is judged by."""
    work = [a for a in agents if a.get("role") == "task"]
    if not work:
        return []

    def table(title, key):
        groups = {}
        for a in work:
            groups.setdefault(key(a), []).append(a)
        out = ["", f"{title:<14}{'agents':>6}{'cost':>10}{'per agent':>10}{'out tok':>10}{'cache read':>12}"
                   f"{'cache write':>12}{'peak context':>14}"]
        for name, group in sorted(groups.items(), key=lambda kv: -sum(a.get("cost_usd") or 0 for a in kv[1])):
            spent = sum(a.get("cost_usd") or 0 for a in group)
            out.append(f"{str(name)[:13]:<14}{len(group):>6}{spent:>10.2f}{spent / len(group):>10.2f}"
                       f"{sum(a.get('output_tokens') or 0 for a in group) / 1e6:>9.2f}M"
                       f"{sum(a.get('cache_read_tokens') or 0 for a in group) / 1e6:>11.1f}M"
                       f"{sum(a.get('cache_write_tokens') or 0 for a in group) / 1e6:>11.2f}M"
                       f"{median(a.get('peak_context') or 0 for a in group) // 1000:>13}k")
        return out

    tasks = s["tasks"]
    lines = table("by tier", lambda a: a.get("tier") or "?")
    lines += table("by kind", lambda a: (tasks.get(a["unit"]) or {}).get("kind") or "?")
    turns = [t for t in s["turns"] if t.get("status") == "done"]
    quiet = [t for t in turns if not any(o["op"] in PLAN_OPS and not o.get("error") for o in t["ops"])]
    spent = {a["id"]: a.get("cost_usd") or 0 for a in agents if a.get("role") == "orchestrator"}
    lines += ["", f"orchestrator: {len(turns)} turn(s) for ${sum(spent.values()):.2f}; {len(quiet)} of them left the "
                  f"plan as it was (${sum(spent.get(t['agent'], 0) for t in quiet):.2f})"]
    return lines


# ---------------------------------------------------------------- UI server

def slim_event(e):
    """Cuts a stream-json event down to what the transcript view shows."""
    t = e.get("type")
    if t == "orchestrator":
        return e
    if t == "system":
        if e.get("subtype") != "init":
            return None
        return {"type": "system", "subtype": "init", "model": e.get("model"), "cwd": e.get("cwd"),
                "session_id": e.get("session_id")}
    if t == "result":
        return {k: e.get(k) for k in ("type", "subtype", "is_error", "result", "num_turns", "duration_ms",
                                      "total_cost_usd")}
    if t not in ("assistant", "user"):
        return None

    def cut(v, limit=20_000):
        if isinstance(v, str):
            return v if len(v) <= limit else v[:limit] + f"\n... [{len(v) - limit} more characters]"
        if isinstance(v, list):
            return [cut(x, limit) for x in v]
        if isinstance(v, dict):
            return {k: cut(x, limit) for k, x in v.items() if k not in ("signature", "data")}
        return v

    content = (e.get("message") or {}).get("content")
    if isinstance(content, str):
        content = [{"type": "text", "text": content}]
    return {"type": t, "content": cut(content or []), "parent": e.get("parent_tool_use_id"),
            "timestamp": e.get("timestamp")}


def make_handler(run_dir):
    agents_dir = os.path.join(run_dir, "agents")
    tasks_dir = os.path.join(run_dir, "tasks")

    class Handler(BaseHTTPRequestHandler):
        def log_message(self, *args):
            pass

        def send(self, body, ctype="application/json", code=200):
            if not isinstance(body, (bytes, str)):
                body = json.dumps(body)
            if isinstance(body, str):
                body = body.encode("utf-8")
            self.send_response(code)
            self.send_header("Content-Type", ctype + "; charset=utf-8")
            self.send_header("Content-Length", str(len(body)))
            self.send_header("Cache-Control", "no-store")
            self.end_headers()
            self.wfile.write(body)

        def do_GET(self):
            url = urlparse(self.path)
            parts = [p for p in url.path.split("/") if p]
            query = parse_qs(url.query)
            try:
                if not parts:
                    return self.send(read_text(UI_FILE, "ui.html is missing"), "text/html")
                if parts == ["api", "state"]:
                    state = read_json(os.path.join(run_dir, "state.json"), {})
                    return self.send({"state": state, "agents": load_agents(run_dir), "now": now(),
                                      "alive": pid_alive(state.get("pid")), "run_dir": run_dir})
                if parts == ["api", "plan"]:
                    # every revision of every brief, oldest first; the last one is the brief in force
                    state, briefs = read_json(os.path.join(run_dir, "state.json"), {}), {}
                    for tid, t in (state.get("tasks") or {}).items():
                        briefs[tid] = [read_text(os.path.join(tasks_dir, "history", f"{tid}.r{r}.md"))
                                       for r in range(1, (t.get("rev") or 0) + 1)]
                    return self.send({"goal": read_text(os.path.join(run_dir, "goal.md")), "briefs": briefs})
                if parts == ["api", "log"]:
                    return self.send({"log": read_text(os.path.join(run_dir, "orchestrator.log"))[-60_000:]})
                if len(parts) >= 3 and parts[:2] == ["api", "agent"] and ID_RE.match(parts[2]):
                    d = os.path.join(agents_dir, parts[2])
                    if not os.path.isdir(d):
                        return self.send({"error": "no such agent"}, code=404)
                    if len(parts) == 3:
                        return self.send({"meta": read_json(os.path.join(d, "meta.json"), {}),
                                          "prompt": read_text(os.path.join(d, "prompt.md")),
                                          "report": read_text(os.path.join(d, "report.md")),
                                          "result": read_json(os.path.join(d, "result.json")),
                                          "stderr": read_text(os.path.join(d, "stderr.log"))[-8000:]})
                    if parts[3] == "events":
                        return self.send(self.events(os.path.join(d, "events.jsonl"),
                                                     int(query.get("from", ["0"])[0])))
                self.send({"error": "not found"}, code=404)
            except (BrokenPipeError, ConnectionResetError):
                pass
            except Exception as e:
                with contextlib.suppress(Exception):
                    self.send({"error": str(e)}, code=500)

        def events(self, path, offset):
            out, limit = [], 1_500_000
            try:
                with open(path, "rb") as f:
                    f.seek(offset)
                    chunk = f.read(limit)
            except OSError:
                return {"events": [], "next": offset}
            end = chunk.rfind(b"\n") + 1      # only complete lines; the rest comes with the next poll
            for line in chunk[:end].splitlines():
                try:
                    e = slim_event(json.loads(line))
                except ValueError:
                    continue
                if e:
                    out.append(e)
            return {"events": out, "next": offset + end, "more": len(chunk) == limit}

    return Handler


def start_ui(run_dir, port, block=False):
    handler = make_handler(run_dir)
    server = None
    for p in range(port, port + 20):
        try:
            server = ThreadingHTTPServer(("127.0.0.1", p), handler)
            break
        except OSError:
            continue
    if server is None:
        print(f"UI: no free port in {port}-{port + 19}", file=sys.stderr)
        return None
    server.daemon_threads = True
    url = f"http://127.0.0.1:{server.server_address[1]}"
    if block:
        print(f"UI for {run_dir}: {url}  (Ctrl-C to stop)")
        with contextlib.suppress(KeyboardInterrupt):
            server.serve_forever()
        return url
    threading.Thread(target=server.serve_forever, name="ui", daemon=True).start()
    return url


# ---------------------------------------------------------------- command line

def repo_root(path):
    rc, out, err = git_rc(path, "rev-parse", "--show-toplevel")
    if rc:
        sys.exit(f"{path} is not inside a git repository")
    return out.strip()


def latest_run(repo):
    base = os.path.join(repo, RUN_ROOT)
    runs = [os.path.join(base, n) for n in (os.listdir(base) if os.path.isdir(base) else [])]
    runs = [r for r in runs if os.path.exists(os.path.join(r, "state.json"))]
    if not runs:
        sys.exit(f"no runs under {base}; pass --run-dir")
    return max(runs, key=lambda r: os.path.getmtime(os.path.join(r, "state.json")))


def resolve_run_dir(args):
    if args.run_dir:
        return os.path.abspath(args.run_dir)
    return latest_run(repo_root(os.path.abspath(args.repo)))


def merge_config(saved, args):
    """The configuration of a run: the defaults, what the run was started with, and this command line.

    --model and --effort set all three tiers; --tier NAME=MODEL[:EFFORT] sets one. A tier given
    without an effort runs at the model's own default effort.
    """
    cfg = {**DEFAULTS, **saved}
    tiers = {name: {**DEFAULTS["tiers"][name], **(saved.get("tiers") or {}).get(name, {})} for name in TIERS}
    for name in TIERS:
        if getattr(args, "model", None):
            tiers[name]["model"] = args.model
        if getattr(args, "effort", None) is not None:
            tiers[name]["effort"] = args.effort
    for spec in getattr(args, "tier", None) or []:
        name, sep, on = spec.partition("=")
        model, _, effort = on.partition(":")
        if name not in TIERS or not sep or not model:
            sys.exit(f"--tier {spec!r}: give NAME=MODEL or NAME=MODEL:EFFORT, where NAME is one of {', '.join(TIERS)}")
        tiers[name] = {"model": model, "effort": effort}
    cfg.update({k: getattr(args, k) for k in DEFAULTS if k != "tiers" and getattr(args, k, None) is not None})
    cfg["tiers"] = tiers
    return {k: cfg[k] for k in DEFAULTS}


def create_run(run_dir, repo, name, goal, goal_source, args):
    base = git(repo, "rev-parse", args.base or "HEAD")
    stale = git(repo, "for-each-ref", "--format=%(refname:short)", f"refs/heads/{BRANCH_NS}/{name}/")
    if stale:
        sys.exit(f"branches of an earlier run named {name} still exist ({', '.join(stale.split())}).\n"
                 "Delete them or start this run under another --name.")
    if not args.base and git(repo, "status", "--porcelain"):
        print("note: the checkout has uncommitted changes; the run starts from the last commit and will not see them")
    os.makedirs(run_dir, exist_ok=True)
    exclude = os.path.join(repo, git(repo, "rev-parse", "--git-path", "info/exclude"))
    if os.path.commonpath([run_dir, repo]) == repo and f"{RUN_ROOT}/" not in read_text(exclude).split():
        os.makedirs(os.path.dirname(exclude), exist_ok=True)
        with open(exclude, "a", encoding="utf-8") as f:
            f.write(f"\n{RUN_ROOT}/\n")
    write_text(os.path.join(run_dir, "goal.md"), goal)
    write_json(os.path.join(run_dir, "state.json"), {
        "version": 3, "name": name, "created_at": now(), "repo": repo, "base_ref": base,
        "goal_source": goal_source, "goal_hash": hashlib.sha1(goal.encode()).hexdigest(),
        "integration_branch": f"{BRANCH_NS}/{name}/integration", "config": {},
        "status": "running", "error": None, "rev": 0, "notes": "", "notes_turn": None,
        "tasks": {}, "task_order": [], "turns": [], "events": [], "seen_seq": 0, "idle_streak": 0,
        "wait": None, "result": None,
    })


def cmd_run(args):
    repo = repo_root(os.path.abspath(args.repo))
    goal = goal_source = None
    if args.goal:
        if os.path.isfile(args.goal):
            goal_source = os.path.abspath(args.goal)
            goal = read_text(goal_source)
        else:
            goal = args.goal
        if not goal.strip():
            sys.exit("the goal is empty")

    if args.run_dir:
        run_dir = os.path.abspath(args.run_dir)
        name = args.name or os.path.basename(run_dir)
    elif goal:
        digest = hashlib.sha1(goal.encode()).hexdigest()[:6]
        stem = os.path.splitext(os.path.basename(goal_source))[0] if goal_source else " ".join(goal.split()[:6])
        name = args.name or f"{slugify(stem)}-{digest}"
        run_dir = os.path.join(repo, RUN_ROOT, name)
    else:
        sys.exit("give a goal (text or a path to a document) or --run-dir of a run to continue")
    if not ID_RE.match(name):
        sys.exit(f"run name {name!r} must be letters, digits, '.', '_' or '-'")

    state_path = os.path.join(run_dir, "state.json")
    if not os.path.exists(state_path):
        if not goal:
            sys.exit(f"there is no run at {run_dir}")
        create_run(run_dir, repo, name, goal, goal_source, args)
    state = read_json(state_path)
    if state.get("version") != 3:
        sys.exit(f"{run_dir} is not an autobuild v3 run")
    if goal and hashlib.sha1(goal.encode()).hexdigest() != state["goal_hash"]:
        sys.exit(f"{run_dir} was started with a different goal; use another --name for a new run")

    lock = open(os.path.join(run_dir, "lock"), "w")
    try:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
    except OSError:
        sys.exit(f"another autobuild process is already working on {run_dir}")

    state["config"] = merge_config(state["config"], args)
    write_json(state_path, state)
    if state["status"] in ("completed", "gave_up"):
        print(summary_text(run_dir))
        return 0 if state["status"] == "completed" else 2

    run = Run(run_dir)
    stops = []

    def on_signal(signum, frame):
        if stops:
            os._exit(130)
        stops.append(signum)
        print("\nstopping agents (again to force quit)...", flush=True)
        threading.Thread(target=run.end_run, args=("interrupted", "stopped by a signal"), daemon=True).start()

    signal.signal(signal.SIGINT, on_signal)
    signal.signal(signal.SIGTERM, on_signal)
    run.log(f"run {name} in {run_dir}")
    if not args.no_ui:
        url = start_ui(run_dir, args.port)
        if url:
            run.log(f"UI: {url}")
    code = run.run()
    print()
    print(summary_text(run_dir))
    return code


def cmd_ui(args):
    start_ui(resolve_run_dir(args), args.port, block=True)
    return 0


def cmd_status(args):
    print(summary_text(resolve_run_dir(args)))
    return 0


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0],
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = ap.add_subparsers(dest="cmd", required=True)

    run = sub.add_parser("run", help="start a run, or continue one that was stopped")
    run.add_argument("goal", nargs="?", help="what to build: text, or a path to a document describing it")
    run.add_argument("--name", help="run name (default: derived from the goal, so the same goal continues its run)")
    run.add_argument("--base", help="commit or branch to start from (default: HEAD)")
    run.add_argument("--max-parallel", type=int, help=f"tasks run at once (default {DEFAULTS['max_parallel']})")
    run.add_argument("--wake", choices=["declared", "each", "idle"],
                     help="when a new orchestrator instance is started: when the tasks the last one said it waits "
                          "for have ended, after each finished task, or only when nothing is left running; a "
                          f"failed task always starts one (default {DEFAULTS['wake']})")
    run.add_argument("--max-turns", type=int,
                     help=f"orchestrator turns before the run is stopped (default {DEFAULTS['max_turns']})")
    run.add_argument("--max-cost", type=float, help="USD the run may spend before it is stopped (default: no limit)")
    tiers = ", ".join(f"{n}={v['model']}:{v['effort']}" for n, v in DEFAULTS["tiers"].items())
    run.add_argument("--tier", action="append", metavar="NAME=MODEL[:EFFORT]",
                     help=f"what a tier runs on; repeat for several (default {tiers}). The orchestrator runs at "
                          f"{ORCHESTRATOR_TIER}, merge agents at {MERGE_TIER}")
    run.add_argument("--model", help="one model for all three tiers")
    run.add_argument("--effort", help="one effort for all three tiers")
    run.add_argument("--setup-cmd", help="shell command run once in every new task worktree, e.g. 'cd web && npm ci'")
    run.add_argument("--agent-timeout", type=int, help="seconds before an agent is stopped and retried")
    run.add_argument("--agent-retries", type=int, help="extra attempts when an agent process fails")
    run.add_argument("--permission-mode", help=f"passed to claude (default {DEFAULTS['permission_mode']})")
    run.add_argument("--claude-bin", help="claude executable (default: claude)")
    run.add_argument("--keep-worktrees", action="store_true", default=None,
                     help="keep each task's worktree after the task ends")
    run.add_argument("--no-ui", action="store_true", help="do not serve the UI while running")
    run.set_defaults(fn=cmd_run)

    ui = sub.add_parser("ui", help="serve the UI for a run (works while it runs and after)")
    ui.set_defaults(fn=cmd_ui)
    status = sub.add_parser("status", help="print a summary of a run")
    status.set_defaults(fn=cmd_status)

    for p in (run, ui, status):
        p.add_argument("--repo", default=".", help="the git repository (default: the current directory)")
        p.add_argument("--run-dir", help=f"run directory (default: <repo>/{RUN_ROOT}/<name>; latest run for ui/status)")
    for p in (run, ui):
        p.add_argument("--port", type=int, default=8797, help="UI port (default 8797; the next free one is used)")

    args = ap.parse_args()
    sys.exit(args.fn(args))


if __name__ == "__main__":
    main()
