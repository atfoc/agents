#!/usr/bin/env python3
"""Autobuild: turns a description of what to build into reviewed, parallel Claude Code work.

  python3 scripts/autobuild/autobuild.py run "add a dark mode toggle"   # or a path to a document
  python3 scripts/autobuild/autobuild.py run docs/feature.md --max-parallel 6
  python3 scripts/autobuild/autobuild.py ui                              # UI for the latest run
  python3 scripts/autobuild/autobuild.py status

What a run does:
  1. split    one agent breaks the goal into tasks (tasks.json + one brief per task), kept in a
              small git repo of its own so every later change to the plan is a commit.
  2. tasks    each task is implemented in its own git worktree, branched from the integration
              branch once its dependencies are merged. Independent tasks run in parallel.
  3. final    the merged result is checked against the goal.

Every one of those (the split, each task, the final result) goes through the same loop:
review -> fix -> review -> fix -> review -> fix -> review, and the run aborts if the last review
still finds problems. The first review sees the whole thing; later reviews see only the diff of
the previous fix. Every few finished tasks a read-only monitor agent reports on how the run is
going.

Everything is on disk under <repo>/.autobuild/<run>/ (state.json, plan/, agents/<id>/ with the
prompt, transcript, report and result of every agent, worktrees/). Kill the script at any point
and run the same command again: finished steps are skipped and interrupted agents are resumed.
The result is left on the branch autobuild/<run>/integration; your checkout is never touched.

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

DEFAULTS = {
    "model": "claude-opus-5-5",
    "effort": "high",
    "max_parallel": 4,
    "max_fix_rounds": 3,       # review, then up to this many fix + review rounds
    "monitor_every": 3,        # finished tasks between monitor reports (0 = never)
    "agent_timeout": 3 * 3600,
    "agent_retries": 2,        # extra attempts when an agent process fails
    "setup_cmd": "",           # run once in every new worktree (e.g. "cd web && npm ci")
    "claude_bin": "claude",
    "permission_mode": "bypassPermissions",
    "keep_worktrees": False,
}
BLOCKING = ("blocker", "major")
SEVERITIES = ("blocker", "major", "minor")
INLINE_LIMIT = 40_000          # longer goals and diffs are passed by path instead of inline
ID_RE = re.compile(r"^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$")
TASK_ID_RE = re.compile(r"^T[0-9]{2,4}$")
PLAN_IDENT = ["-c", "user.name=autobuild", "-c", "user.email=autobuild@localhost"]

REVIEW_SCHEMA = {
    "type": "object",
    "properties": {
        "summary": {"type": "string", "description": "Two or three sentences: what was reviewed and the outcome."},
        "findings": {
            "type": "array",
            "items": {
                "type": "object",
                "properties": {
                    "severity": {"type": "string", "enum": list(SEVERITIES)},
                    "title": {"type": "string"},
                    "location": {"type": "string", "description": "file:line, task id, or section the finding is about"},
                    "details": {"type": "string", "description": "What is wrong and how you verified it."},
                    "suggestion": {"type": "string", "description": "What a correct fix looks like."},
                },
                "required": ["severity", "title", "details"],
            },
        },
        "previous_findings": {
            "type": "array",
            "description": "Later rounds only: the state of each finding from the previous review.",
            "items": {
                "type": "object",
                "properties": {
                    "id": {"type": "string"},
                    "status": {"type": "string", "enum": ["resolved", "unresolved", "dispute_accepted"]},
                    "note": {"type": "string"},
                },
                "required": ["id", "status"],
            },
        },
    },
    "required": ["summary", "findings"],
}

MONITOR_SCHEMA = {
    "type": "object",
    "properties": {
        "health": {"type": "string", "enum": ["on_track", "at_risk", "off_track"]},
        "summary": {"type": "string", "description": "Two or three sentences."},
        "report": {"type": "string", "description": "The full report, in markdown."},
    },
    "required": ["health", "summary", "report"],
}

SEVERITY_RULES = """Severity:
- blocker: left as it is, the result is wrong, incomplete or broken.
- major: a real problem that is likely to cause a defect, a failing build or test, a merge
  conflict between parallel tasks, or a departure from what was asked for.
- minor: a real but small improvement. Minor findings are recorded and do not block.

Report only problems you have verified: no style preferences, no speculation, no praise. The
review passes when there is no blocker or major finding, so an empty findings list is a valid
result. Say exactly where each problem is and what a correct fix looks like, because the agent
that fixes it sees your findings and nothing else from you."""

READ_ONLY_RULES = """Rules:
- You are read-only. Do not create, edit or delete files, and do not commit. You may run
  commands that only inspect or verify (git, builds, tests); anything they leave behind is
  discarded.
- You run unattended and cannot ask questions."""

WRITE_RULES = """Rules:
- Work only inside your working directory{extra}. Other agents are working at the same time in
  other git worktrees of this repository; never touch anything outside your own.
- Do not commit, push, stash, switch branches or rewrite history. The orchestrator commits your
  work when you finish.
- You run unattended and cannot ask questions. Where something is unclear, make the most
  reasonable choice and say so in your report."""

RESUME_PROMPT = """Your previous run of this job stopped before it finished. Continue where you left off and
complete the job. Everything in the original instructions still applies, including how to end."""


class Stop(Exception):
    """The run is shutting down: a signal arrived or another unit failed."""


class AgentError(Exception):
    """An agent produced no usable result after all its attempts."""


class UnitFailed(Exception):
    """A review loop still found problems after its last round."""


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


# ---------------------------------------------------------------- plan validation

def validate_plan(plan_dir):
    """Checks tasks.json and the briefs. Returns (tasks, errors, warnings)."""
    errors, warnings = [], []
    path = os.path.join(plan_dir, "tasks.json")
    if not os.path.exists(path):
        return [], ["tasks.json does not exist"], warnings
    try:
        with open(path, encoding="utf-8") as f:
            data = json.load(f)
    except ValueError as e:
        return [], [f"tasks.json is not valid JSON: {e}"], warnings
    raw = data.get("tasks") if isinstance(data, dict) else None
    if not isinstance(raw, list) or not raw:
        return [], ['tasks.json must be an object with a non-empty "tasks" array'], warnings

    tasks, seen = [], set()
    for i, t in enumerate(raw):
        if not isinstance(t, dict):
            errors.append(f"tasks[{i}] is not an object")
            continue
        tid = t.get("id")
        if not isinstance(tid, str) or not TASK_ID_RE.match(tid):
            errors.append(f"tasks[{i}] has id {tid!r}; ids must look like T01")
            continue
        if tid in seen:
            errors.append(f"task id {tid} is used more than once")
            continue
        seen.add(tid)
        if not isinstance(t.get("title"), str) or not t["title"].strip():
            errors.append(f"{tid} has no title")
        for key in ("depends_on", "paths"):
            if not isinstance(t.get(key, []), list) or not all(isinstance(x, str) for x in t.get(key, [])):
                errors.append(f"{tid}.{key} must be a list of strings")
                t[key] = []
        tasks.append({"id": tid, "title": str(t.get("title") or "").strip(),
                      "summary": str(t.get("summary") or "").strip(),
                      "depends_on": list(dict.fromkeys(t.get("depends_on", []))),
                      "paths": list(t.get("paths", []))})
    by_id = {t["id"]: t for t in tasks}

    for t in tasks:
        for d in t["depends_on"]:
            if d == t["id"]:
                errors.append(f"{t['id']} depends on itself")
            elif d not in by_id:
                errors.append(f"{t['id']} depends on {d}, which is not a task")
        if not read_text(os.path.join(plan_dir, "tasks", t["id"] + ".md")).strip():
            errors.append(f"{t['id']} has no brief at tasks/{t['id']}.md")
    tasks_dir = os.path.join(plan_dir, "tasks")
    if os.path.isdir(tasks_dir):
        for name in sorted(os.listdir(tasks_dir)):
            if name.endswith(".md") and name[:-3] not in by_id:
                errors.append(f"tasks/{name} belongs to no task in tasks.json")

    # cycles, and every task's full set of ancestors
    ancestors, state = {}, {}

    def visit(tid, stack):
        if state.get(tid) == 2:
            return ancestors[tid]
        if state.get(tid) == 1:
            errors.append("dependency cycle: " + " -> ".join(stack[stack.index(tid):] + [tid]))
            return set()
        state[tid] = 1
        acc = set()
        for d in by_id[tid]["depends_on"]:
            if d in by_id and d != tid:
                acc.add(d)
                acc |= visit(d, stack + [tid])
        state[tid], ancestors[tid] = 2, acc
        return acc

    for t in tasks:
        visit(t["id"], [])

    def overlap(a, b):
        a, b = a.strip("./").rstrip("/"), b.strip("./").rstrip("/")
        return bool(a) and bool(b) and (a == b or a.startswith(b + "/") or b.startswith(a + "/"))

    for i, a in enumerate(tasks):
        for b in tasks[i + 1:]:
            if a["id"] in ancestors.get(b["id"], ()) or b["id"] in ancestors.get(a["id"], ()):
                continue
            shared = sorted({p for p in a["paths"] for q in b["paths"] if overlap(p, q)})
            if shared:
                warnings.append(f"{a['id']} and {b['id']} can run in parallel but both list {', '.join(shared)}")
    return tasks, errors, warnings


# ---------------------------------------------------------------- review units

class Unit:
    """Something that goes through the review loop: the split, one task, or the final result."""

    def __init__(self, key, cwd, repo, loop, ident, target, scope, checklist, fix_note,
                 set_status, commit_label, add_dirs=(), guards=(), validate=None):
        self.key = key
        self.cwd = cwd                  # where its agents run
        self.repo = repo                # git repo whose commits hold the reviewed work
        self.loop = loop                # this unit's loop record inside the state
        self.ident = ident
        self.target = target            # markdown: the desired end state
        self.scope = scope              # fn(agent_dir) -> markdown: what the first review covers
        self.checklist = checklist
        self.fix_note = fix_note
        self.set_status = set_status
        self.commit_label = commit_label
        self.add_dirs = list(add_dirs)
        self.guards = list(guards)      # git dirs to put back after a read-only agent
        self.validate = validate or (lambda: [])


# ---------------------------------------------------------------- orchestrator

class Orchestrator:
    def __init__(self, run_dir):
        self.run_dir = run_dir
        self.state_path = os.path.join(run_dir, "state.json")
        self.state = read_json(self.state_path)
        self.cfg = self.state["config"]
        self.repo = self.state["repo"]
        self.plan_dir = os.path.join(run_dir, "plan")
        self.agents_dir = os.path.join(run_dir, "agents")
        self.int_wt = os.path.join(run_dir, "worktrees", "integration")
        self.int_branch = self.state["integration_branch"]
        self.lock = threading.RLock()
        self.cond = threading.Condition(self.lock)
        self.git_lock = threading.Lock()      # worktree bookkeeping and merges, one at a time
        self.log_lock = threading.Lock()
        self.procs_lock = threading.Lock()
        self.procs = {}
        self.stop = threading.Event()
        self.failure = None                   # (status, message) of the first thing that ended the run
        self.monitor_thread = None
        configured = git_ok(self.repo, "config", "user.name") and git_ok(self.repo, "config", "user.email")
        self.ident = [] if configured else PLAN_IDENT

    # ---- state

    @contextlib.contextmanager
    def mut(self):
        with self.lock:
            yield self.state
            self.state["updated_at"] = now()
            write_json(self.state_path, self.state)

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

    # ---- agents

    def agent_dir(self, agent_id):
        assert ID_RE.match(agent_id), agent_id
        d = os.path.join(self.agents_dir, agent_id)
        os.makedirs(d, exist_ok=True)
        return d

    def reap_stale_agents(self):
        """Agents a previous orchestrator left 'running': stop any survivor and mark them."""
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

    def run_agent(self, agent_id, role, unit, prompt, cwd, schema=None, read_only=False,
                  add_dirs=(), title=""):
        """Runs one Claude Code agent to completion and returns its meta record.

        Idempotent: a finished agent is returned from disk, an interrupted one resumes its
        session. Raises Stop when the run is shutting down and AgentError when the agent keeps
        failing.
        """
        d = self.agent_dir(agent_id)
        meta_path = os.path.join(d, "meta.json")
        meta = read_json(meta_path)
        if meta and meta.get("status") == "done":
            return meta
        self.check_stop()
        if not meta:
            meta = {"id": agent_id, "role": role, "unit": unit, "title": title, "cwd": cwd,
                    "model": self.cfg["model"], "effort": self.cfg["effort"], "read_only": read_only,
                    "created_at": now(), "attempts": [], "session_id": None, "session_live": False,
                    "cost_usd": 0.0, "num_turns": 0, "output_tokens": 0}
        meta.update(status="running", started_at=meta.get("started_at") or now(), ended_at=None, error=None)
        write_json(meta_path, meta)

        failures = 0
        while True:
            resume = bool(meta.get("session_id") and meta.get("session_live"))
            if resume:
                text = RESUME_PROMPT
            else:
                text = prompt
                if meta["attempts"] and not read_only:
                    text += ("\n\nNote: an earlier attempt at this job did not finish. Your working directory may "
                             "hold its partial work; check `git status` and build on whatever is sound.")
                meta["session_id"], meta["session_live"] = str(uuid.uuid4()), False
                write_text(os.path.join(d, "prompt.md"), prompt)
            stdin_path = os.path.join(d, "stdin.md")
            write_text(stdin_path, text)
            attempt = {"n": len(meta["attempts"]) + 1, "resume": resume, "started_at": now()}
            meta["attempts"].append(attempt)
            self.log(f"agent {agent_id}: {'resuming' if resume else 'starting'} (attempt {attempt['n']})")
            ok, error = self._exec(meta, meta_path, d, stdin_path, cwd, schema, read_only, add_dirs, resume, attempt)
            attempt["ended_at"] = now()
            if ok:
                meta.update(status="done", ended_at=now(), error=None)
                write_json(meta_path, meta)
                self.log(f"agent {agent_id}: done in {fmt_duration(meta['ended_at'] - meta['started_at'])}"
                         f" (${meta['cost_usd']:.2f})")
                return meta
            attempt["error"] = error
            if self.stop.is_set():
                meta.update(status="interrupted", ended_at=now(), error="interrupted")
                write_json(meta_path, meta)
                raise Stop()
            if resume and attempt.get("no_session"):
                meta["session_live"] = False   # nothing to resume; start over without counting it
                write_json(meta_path, meta)
                continue
            failures += 1
            self.log(f"agent {agent_id}: attempt {attempt['n']} failed: {error}")
            if failures > self.cfg["agent_retries"]:
                meta.update(status="failed", ended_at=now(), error=error)
                write_json(meta_path, meta)
                raise AgentError(f"agent {agent_id} failed: {error}")
            write_json(meta_path, meta)
            if self.stop.wait(30 * failures * failures):
                meta.update(status="interrupted", ended_at=now(), error="interrupted")
                write_json(meta_path, meta)
                raise Stop()

    def _exec(self, meta, meta_path, d, stdin_path, cwd, schema, read_only, add_dirs, resume, attempt):
        cfg = self.cfg
        cmd = [cfg["claude_bin"], "-p", "--model", cfg["model"], "--output-format", "stream-json", "--verbose",
               "--permission-mode", cfg["permission_mode"], "--strict-mcp-config",
               "--name", f"autobuild {self.state['name']} / {meta['id']}"]
        if cfg["effort"]:
            cmd += ["--effort", cfg["effort"]]
        cmd += ["--resume", meta["session_id"]] if resume else ["--session-id", meta["session_id"]]
        if schema:
            cmd += ["--json-schema", json.dumps(schema)]
        if read_only:
            cmd += ["--disallowedTools", "Edit,Write,NotebookEdit"]
        for a in add_dirs:
            cmd += ["--add-dir", a]

        result, timed_out = None, []
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
                self.procs[proc.pid] = meta["id"]
            if self.stop.is_set():      # the stop may have swept the process list just before we joined it
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
                    if e.get("type") == "system" and e.get("subtype") == "init" and not meta["session_live"]:
                        meta["session_live"] = True
                        write_json(meta_path, meta)
                    elif e.get("type") == "result":
                        result = e
                rc = proc.wait()
            finally:
                timer.cancel()
                with self.procs_lock:
                    self.procs.pop(proc.pid, None)

        if result:
            meta["cost_usd"] += result.get("total_cost_usd") or 0
            meta["num_turns"] += result.get("num_turns") or 0
            meta["output_tokens"] += (result.get("usage") or {}).get("output_tokens") or 0
        if timed_out:
            return False, f"timed out after {fmt_duration(cfg['agent_timeout'])}"
        if result is None:
            tail = read_text(os.path.join(d, "stderr.log"))[-400:].strip()
            return False, f"exited with code {rc} and no result. {tail}"
        if result.get("is_error"):
            if resume and not result.get("num_turns"):
                attempt["no_session"] = True
            return False, str(result.get("result") or result.get("subtype") or "error")[:600]
        text = result.get("result") or ""
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
        write_text(os.path.join(d, "report.md"), str(text))
        return True, None

    def agent_result(self, agent_id):
        return read_json(os.path.join(self.agents_dir, agent_id, "result.json"), {})

    def agent_report(self, agent_id):
        return read_text(os.path.join(self.agents_dir, agent_id, "report.md")).strip()

    # ---- git

    def ensure_worktree(self, path, branch, base):
        with self.git_lock, repo_lock(self.repo):
            if os.path.exists(os.path.join(path, ".git")):
                return
            git(self.repo, "worktree", "prune")
            if git_ok(self.repo, "rev-parse", "--verify", "-q", f"refs/heads/{branch}"):
                git(self.repo, "worktree", "add", "-q", path, branch)
            else:
                git(self.repo, "worktree", "add", "-q", "-b", branch, path, base)

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

    def heads(self, dirs):
        return {d: git(d, "rev-parse", "HEAD") for d in dirs}

    def restore(self, heads, who):
        """Puts git dirs back to the commits recorded before an agent that must not change them."""
        for d, head in heads.items():
            if git(d, "rev-parse", "HEAD") != head or git(d, "status", "--porcelain"):
                self.log(f"{who} left changes in {d}; discarding them")
                git(d, "reset", "-q", "--hard", head)
                git(d, "clean", "-fdq")

    def diff_section(self, repo, a, b, agent_dir, heading):
        rc, diff, _ = git_rc(repo, "diff", f"{a}..{b}")
        stat = git(repo, "diff", "--stat", f"{a}..{b}")
        path = os.path.join(agent_dir, "input.diff")
        write_text(path, diff)
        if not diff.strip():
            return f"{heading}\n\nThere are no changes between `{a[:12]}` and `{b[:12]}`."
        out = (f"{heading}\n\n- Commits: `{a[:12]}..{b[:12]}` in {repo} (`git -C {repo} diff {a[:12]}..{b[:12]}`)\n"
               f"- The full diff is saved at {path}\n\n<diffstat>\n{stat}\n</diffstat>\n")
        if len(diff) <= INLINE_LIMIT:
            return out + f"\n<diff>\n{diff}</diff>"
        return out + "\nThe diff is too large to show here: read it from the file above before you start."

    # ---- prompts

    def goal_block(self):
        path = os.path.join(self.run_dir, "goal.md")
        text = read_text(path)
        src = self.state.get("goal_source")
        note = f"(The goal was read from {src}.)\n\n" if src else ""
        if len(text) <= INLINE_LIMIT:
            return f"{note}<goal>\n{text.strip()}\n</goal>"
        return f"{note}The goal document is long. Read all of it before anything else: {path}"

    def plan_pointers(self):
        return (f"- The overall goal: {os.path.join(self.run_dir, 'goal.md')}\n"
                f"- The task list: {os.path.join(self.plan_dir, 'tasks.json')}\n"
                f"- The brief of every task: {os.path.join(self.plan_dir, 'tasks')}/<id>.md")

    def p_split(self):
        return f"""You are the planning agent of an automated build pipeline. Break the goal below into implementation tasks and write them to disk. You plan only: do not implement any part of the goal.

# Goal

{self.goal_block()}

# How the tasks will be used

- Each task is implemented by its own agent. That agent gets the task's brief and the repository, can read the goal and the other briefs, and cannot ask anyone a question.
- Each task is built in its own git worktree, branched from the integration branch after every task it depends on has been reviewed and merged there. Tasks with no dependency path between them are built at the same time and merged one after another.
- Every task is reviewed against its brief, fixed if needed, and merged. Once all tasks are merged, the whole result is reviewed against the goal.

# What a good split looks like

1. Complete and correct. Built in dependency order, the tasks produce exactly what the goal asks for: nothing missing, nothing invented. Ground every brief in the code as it is today, so read the relevant parts of the repository (your working directory) before you write anything.
2. Focused. One task is one coherent, reviewable change with a clear boundary, and it leaves the project building with its tests passing once merged. Do not split so finely that a task makes no sense alone, and do not bundle unrelated work into one task.
3. Parallel wherever it is safe. Parallelism is what makes the run fast, so keep the dependency graph as wide and shallow as it can be without causing problems:
   - `depends_on` lists only the tasks whose code this task really needs in its worktree. A dependency that is not needed serialises the run for nothing.
   - Two tasks with no dependency path between them must not edit the same files and must not rely on each other's output. Where they would, restructure the split or add the dependency.
   - When several tasks need the same shared piece (a type, an interface, a schema, a route table, a config entry), put that piece in an early task they all depend on, so that they can then run side by side.
4. Explicit contracts. Whatever crosses a task boundary (names, signatures, file paths, data shapes, wire formats) is spelled out identically in the brief that provides it and in every brief that uses it. Parallel agents cannot talk to each other; the briefs are the only coordination there is.
5. Verifiable. Every task has acceptance criteria a reviewer can check, and says how to check them: the tests to add and the commands to run.

# Output

Write these files under {self.plan_dir} and change nothing anywhere else:

`tasks.json`:

```json
{{"tasks": [{{"id": "T01", "title": "short imperative title", "summary": "one or two sentences on what this task delivers", "depends_on": [], "paths": ["a/dir/", "a/file.go"]}}]}}
```

- `id`: T01, T02, ... in a sensible build order.
- `depends_on`: the ids of the tasks that must be merged before this one starts.
- `paths`: the files or directories the task is expected to create or edit. They are used to spot parallel tasks that would collide, so be accurate.

`tasks/<id>.md`, the brief of each task, with these sections:

- Context: where this task sits in the whole and what already exists that it builds on (name the files).
- What to build: the change itself, precisely enough that two different engineers would build the same thing.
- Contracts: what this task provides to other tasks and what it uses from its dependencies, with exact names and shapes.
- Files: what it creates or edits.
- Acceptance criteria: a checklist of observable results.
- Verification: the tests to add and the commands that must pass.
- Out of scope: neighbouring work that belongs to other tasks (name them), so the agent does not do it.

Finish with a short report: the shape of the dependency graph, why it is cut where it is, which tasks run in parallel, and any assumption you had to make about the goal.

{WRITE_RULES.format(extra=" and " + self.plan_dir)}
- Your working directory is a checkout of the repository for you to read. Do not change it."""

    def review_prompt(self, u, n, agent_dir):
        rounds = u.loop["rounds"]
        if n == 1:
            return f"""You are a reviewer in an automated build pipeline. Review the work described below against its desired end state and report every real problem.

# Desired end state

{u.target}

# What to review

{u.scope(agent_dir)}

# What to check

{u.checklist}

{SEVERITY_RULES}

{READ_ONLY_RULES}

Finish by giving your structured result: a short summary and the list of findings."""

        prev = rounds[n - 2]
        findings = "\n\n".join(render_finding(f) for f in prev["findings"]) or "(none)"
        diff = self.diff_section(u.repo, prev["fix_pre"], prev["fix_post"], agent_dir,
                                 "These are the changes the fix agent made, and they are all you review:")
        return f"""You are a reviewer in an automated build pipeline. This is review round {n} of the same work. The previous review found problems and a fix agent has worked on them. Your review is narrow: only the fix.

# Desired end state

{u.target}

# Findings of the previous review

{findings}

# The fix agent's report

<fix_report>
{self.agent_report(prev["fix"]) or "(the fix agent left no report)"}
</fix_report>

# What changed

{diff}

# What to do

1. For each previous blocker or major finding, decide from the diff and the current state whether it is resolved. If the fix agent disputed a finding instead of fixing it, check its reasoning against the code: accept the dispute if it is right, otherwise the finding stands. Record the outcome of each in `previous_findings`.
2. Check the changes themselves for new problems they introduce. For reference, the first review checked:

{u.checklist}

3. Report as findings only what is still wrong: unresolved previous findings (restated so they stand on their own) and new problems caused by the fix. Do not re-review parts the fix did not touch; they were covered by the earlier rounds.

{SEVERITY_RULES}

{READ_ONLY_RULES}

Finish by giving your structured result: a short summary, the findings that remain, and the state of each previous finding."""

    def fix_prompt(self, u, n):
        rnd = u.loop["rounds"][n - 1]
        must = [f for f in rnd["findings"] if f["severity"] in BLOCKING]
        minor = [f for f in rnd["findings"] if f["severity"] not in BLOCKING]
        optional = ""
        if minor:
            optional = ("\n\n# Minor findings (optional)\n\nFix these only where the fix is small and safe; "
                        "otherwise leave them.\n\n" + "\n\n".join(render_finding(f) for f in minor))
        return f"""You are a fix agent in an automated build pipeline. A review found the problems listed below. Fix them.

# Desired end state

{u.target}

# Findings to fix

{chr(10).join(render_finding(f) + chr(10) for f in must)}{optional}

# How to work

- Fix the cause of each finding, not its symptom, and change no more than the findings need. The next review looks only at your diff, so unrelated changes are both unreviewed and unwelcome.
- If you are sure a finding is wrong, do not change anything for it: explain why in your report, with evidence from the code. The next reviewer decides.
- Verify what you changed.
{u.fix_note}

Finish with a report that goes through the findings by id: what you changed for each, or why you dispute it, and how you verified the result. The next reviewer reads this report.

{WRITE_RULES.format(extra="")}"""

    def p_implement(self, t, brief):
        deps = ", ".join(t["depends_on"]) or "none"
        return f"""You are an implementation agent in an automated build pipeline. Implement the one task described below, completely and only that task.

# Your task: {t['id']} {t['title']}

<brief>
{brief.strip()}
</brief>

# Where this fits

Your task is one piece of a larger plan. For context only, you can read:
{self.plan_pointers()}

Your working directory is a git worktree made for this task. The tasks you depend on ({deps}) are already merged into it. Other tasks are being built at the same time by other agents in other worktrees, and all of them are merged together afterwards, so:

- Build what your brief says and stay inside its scope. Work that belongs to another task will collide with that task's agent.
- Follow the contracts in your brief exactly (names, signatures, paths, data shapes). Other tasks are being written against the same contracts right now.
- Follow the conventions of the code around you.

# Done means

- Every acceptance criterion in the brief holds.
- The verification the brief asks for passes, along with the build and the existing tests your change could affect. Run them; do not assume.

Your work is then reviewed against the brief. Finish with a report: what you built, which files you changed, what you ran to verify it and with what result, and anything you decided or could not do, with the reason.

{WRITE_RULES.format(extra="")}"""

    def p_merge(self, t, files):
        return f"""You are resolving a merge conflict in an automated build pipeline.

Task {t['id']} ({t['title']}) was built on a branch and has passed review. While it was being built, other tasks were merged into the integration branch. The integration branch has just been merged into this task's worktree (your working directory) and these files conflict:

{chr(10).join('- ' + f for f in files)}

Resolve every conflict so that the result keeps the intent of both sides: this task's change and what the other tasks merged. Neither side may lose behaviour.

- The brief of this task: {os.path.join(self.plan_dir, 'tasks', t['id'] + '.md')}
- The other briefs, to understand what the other side meant: {os.path.join(self.plan_dir, 'tasks')}
- `git log --merge` and `git diff` show both sides.
- When no conflict marker is left, build and run the tests the conflicting files affect.
- Do not run `git commit`, `git merge --abort` or `git reset`. Leave the resolved files in the working tree; the orchestrator concludes the merge.

Finish with a report: for each file, what conflicted and how you resolved it, and what you ran to verify it.

{WRITE_RULES.format(extra="")}"""

    def p_monitor(self, reason):
        return f"""You are the monitor of an automated build pipeline. Go through the run so far and judge how it is progressing and whether everything is on track. You are an observer: your report is stored for the person running the pipeline and changes nothing.

Why now: {reason}.

# How the pipeline works

A goal is split into tasks; each task is implemented in its own git worktree, in parallel where the dependency graph allows; the merged result gets a final review. The split, every task and the final result each go through review -> fix -> review (at most {self.cfg['max_fix_rounds']} fix rounds; if the last review still fails, the run aborts).

# Where everything is

- Run directory: {self.run_dir}
- state.json: the orchestrator's state (phase, every task's status, every review round with its findings)
- goal.md: what is being built
- plan/tasks.json and plan/tasks/<id>.md: the split (plan/ is a git repo; its log shows how the plan changed)
- agents/<agent id>/: one directory per agent, with meta.json (role, status, cost, timing), prompt.md, report.md, result.json (review findings) and events.jsonl (the full transcript; large, read selectively)
- orchestrator.log
- The integration branch `{self.int_branch}` is checked out at {self.int_wt}

# Snapshot

<snapshot>
{summary_text(self.run_dir)}
</snapshot>

# What to look at

- Progress: what is done, what is running, what is waiting, and whether anything looks stuck.
- Review loops: are they converging, or are findings repeating, growing, or bouncing between reviewer and fixer? Which units are at risk of running out of rounds?
- Quality of the split in hindsight: findings that trace back to a vague brief or a wrong contract, merge conflicts, tasks stepping on each other.
- Drift: is what is being built still what the goal asks for? Read the code on the integration branch where a report alone does not settle it.
- Agent behaviour: agents that failed, were retried, ran very long or cost far more than their peers, and why.
- Risks for the rest of the run, and what the person running it should look at or do.

Be specific: name tasks, agents and findings. Do not pad the report with what is fine.

{READ_ONLY_RULES}

Finish by giving your structured result: health (on_track, at_risk or off_track), a summary of two or three sentences, and the full report in markdown."""

    # ---- the review loop

    def review_loop(self, u):
        """review -> fix -> review ... until a review passes or the rounds run out."""
        loop = u.loop
        if loop["status"] == "passed":
            return
        if loop["status"] == "failed":
            raise UnitFailed(f"{u.key} failed its review loop")
        max_reviews = self.cfg["max_fix_rounds"] + 1
        prefix = u.key if loop["attempt"] == 1 else f"{u.key}-a{loop['attempt']}"
        with self.mut():
            loop["status"] = "running"
        n = 1
        while True:
            if len(loop["rounds"]) < n:
                with self.mut():
                    loop["rounds"].append({"n": n})
            rnd = loop["rounds"][n - 1]

            if "verdict" not in rnd:
                u.set_status("reviewing")
                rid = f"{prefix}-review-{n}"
                heads = self.heads(u.guards)
                prompt = self.review_prompt(u, n, self.agent_dir(rid))
                self.run_agent(rid, "review", u.key, prompt, u.cwd, schema=REVIEW_SCHEMA, read_only=True,
                               add_dirs=u.add_dirs, title=f"review {n}")
                self.restore(heads, rid)
                result = self.agent_result(rid)
                findings = []
                for f in list(result.get("findings") or []) + u.validate():
                    sev = f.get("severity") if f.get("severity") in SEVERITIES else "major"
                    findings.append({"id": f"R{n}.{len(findings) + 1}", "severity": sev,
                                     "title": str(f.get("title") or "untitled"),
                                     "location": str(f.get("location") or ""),
                                     "details": str(f.get("details") or ""),
                                     "suggestion": str(f.get("suggestion") or ""),
                                     "source": f.get("source", "reviewer")})
                verdict = "fail" if any(f["severity"] in BLOCKING for f in findings) else "pass"
                write_text(os.path.join(self.agent_dir(rid), "report.md"),
                           render_review(rid, verdict, result, findings))
                with self.mut():
                    rnd.update(review=rid, verdict=verdict, findings=findings, summary=result.get("summary", ""))
                blocking = sum(f["severity"] in BLOCKING for f in findings)
                self.log(f"{u.key}: review {n} {verdict.upper()} ({blocking} blocking, {len(findings) - blocking} minor)")

            if rnd["verdict"] == "pass":
                with self.mut():
                    loop["status"] = "passed"
                return
            if n >= max_reviews:
                with self.mut():
                    loop["status"] = "failed"
                raise UnitFailed(f"{u.key} still has problems after {max_reviews} reviews and "
                                 f"{max_reviews - 1} fix rounds")

            if "fix_post" not in rnd:
                u.set_status("fixing")
                fid = f"{prefix}-fix-{n}"
                if "fix_pre" not in rnd:
                    with self.mut():
                        rnd.update(fix=fid, fix_pre=git(u.repo, "rev-parse", "HEAD"))
                side = self.heads([g for g in u.guards if g != u.repo])
                self.run_agent(fid, "fix", u.key, self.fix_prompt(u, n), u.cwd, add_dirs=u.add_dirs,
                               title=f"fix {n}")
                self.restore(side, fid)
                post = commit_all(u.repo, f"{u.commit_label}: fix review round {n}", u.ident)
                with self.mut():
                    rnd["fix_post"] = post
            n += 1

    # ---- phase 1: split

    def split_unit(self):
        st = self.state["split"]

        def set_status(s):
            with self.mut():
                st["status"] = s

        def scope(agent_dir):
            _, _, warnings = validate_plan(self.plan_dir)
            notes = ""
            if warnings:
                notes = ("\n\nAn automated check flagged these parallel tasks as touching the same paths. Decide for "
                         "each whether it is a real collision:\n" + "\n".join(f"- {w}" for w in warnings))
            return (f"The whole plan: read `tasks.json` and every brief in `tasks/` under {self.plan_dir}. "
                    f"Your working directory is the repository the plan is for, at the commit the work starts "
                    f"from; check the plan against it.{notes}")

        def validate():
            _, errors, _ = validate_plan(self.plan_dir)
            return [{"severity": "blocker", "title": "The plan files are not valid", "location": "tasks.json",
                     "details": e, "suggestion": "Make tasks.json and tasks/ consistent and well formed.",
                     "source": "validator"} for e in errors]

        checklist = """- Coverage: every requirement of the goal is delivered by some task, and no task builds something the goal does not ask for.
- Grounding: what the briefs say about the existing code (files, functions, behaviour) is true. Check it in the repository.
- Dependencies: a task that needs another task's code depends on it, directly or through its dependencies; and no task has a dependency it does not need, since that serialises the run for nothing.
- Parallel safety: two tasks with no dependency path between them do not edit the same files and do not rely on each other's output.
- Contracts: everything that crosses a task boundary is named and shaped identically in the brief that provides it and the briefs that use it.
- Briefs: an agent with only the brief and the repository can build the task without guessing. Scope, acceptance criteria and verification are concrete, and "out of scope" keeps it off its neighbours' work.
- Sizing: each task is one coherent change that leaves the project building and its tests passing. Flag tasks that should be split for parallelism or merged because they cannot stand alone.
- Format: `tasks.json` and the briefs follow the format the pipeline needs (ids like T01, `depends_on`, `paths`, one `tasks/<id>.md` per task)."""

        return Unit(
            key="split", cwd=self.int_wt, repo=self.plan_dir, loop=st["loop"], ident=PLAN_IDENT,
            target=("A plan that breaks this goal into tasks which, built in dependency order by separate agents "
                    "and merged, produce exactly what the goal asks for, with as much safe parallelism as the "
                    f"work allows.\n\n{self.goal_block()}"),
            scope=scope, checklist=checklist,
            fix_note=(f"- The plan is in {self.plan_dir} (`tasks.json` and `tasks/<id>.md`). Edit it there and change "
                      "nothing else; your working directory is the repository, for reading.\n"
                      "- When you change a contract, a dependency or a task's scope, update every brief it touches "
                      "so the plan stays consistent."),
            set_status=set_status, commit_label="plan", add_dirs=[self.plan_dir],
            guards=[self.int_wt, self.plan_dir], validate=validate)

    def phase_split(self):
        st = self.state["split"]
        if st["status"] == "done":
            return
        with self.mut() as s:
            s["phase"] = "split"
        if not os.path.isdir(os.path.join(self.plan_dir, ".git")):
            os.makedirs(self.plan_dir, exist_ok=True)
            git(self.plan_dir, "init", "-q")
            write_text(os.path.join(self.plan_dir, "goal.md"), read_text(os.path.join(self.run_dir, "goal.md")))
            commit_all(self.plan_dir, "goal", PLAN_IDENT)
        u = self.split_unit()
        if not st.get("head"):
            u.set_status("splitting")
            heads = self.heads([self.int_wt])
            self.run_agent("split", "split", "split", self.p_split(), self.int_wt, add_dirs=[self.plan_dir],
                           title="split the goal into tasks")
            self.restore(heads, "split")
            head = commit_all(self.plan_dir, "plan: split the goal into tasks", PLAN_IDENT)
            with self.mut():
                st["head"] = head
        self.review_loop(u)
        tasks, errors, _ = validate_plan(self.plan_dir)
        if errors:
            raise RuntimeError("the reviewed plan is not valid: " + "; ".join(errors))
        with self.mut() as s:
            st["status"] = "done"
            s["task_order"] = [t["id"] for t in tasks]
            for t in tasks:
                s["tasks"].setdefault(t["id"], dict(t, status="pending",
                                                    loop={"status": "pending", "attempt": 1, "rounds": []}))
        self.log(f"split done: {len(tasks)} tasks")

    # ---- phase 2: tasks

    def phase_tasks(self):
        with self.mut() as s:
            s["phase"] = "tasks"
        tasks, order = self.state["tasks"], self.state["task_order"]
        dependents = {tid: 0 for tid in order}
        for tid in order:      # tasks that unblock the most work go first
            seen, todo = set(), list(tasks[tid]["depends_on"])
            while todo:
                d = todo.pop()
                if d not in seen:
                    seen.add(d)
                    dependents[d] += 1
                    todo.extend(tasks[d]["depends_on"])
        running = {}

        def worker(tid):
            try:
                self.run_task(tid)
            except Stop:
                pass
            except UnitFailed as e:
                self.end_run("failed", str(e))
            except Exception as e:
                if not isinstance(e, AgentError):
                    self.log(f"{tid}: {traceback.format_exc()}")
                with self.mut():
                    tasks[tid]["error"] = str(e)
                self.end_run("error", f"{tid}: {e}")
            finally:
                with self.cond:
                    running.pop(tid, None)
                    self.cond.notify_all()

        with self.cond:
            while not self.stop.is_set():
                left = [tid for tid in order if tasks[tid]["status"] != "done"]
                if not left:
                    break
                ready = [tid for tid in left if tid not in running
                         and all(tasks[d]["status"] == "done" for d in tasks[tid]["depends_on"])]
                ready.sort(key=lambda tid: (-dependents[tid], order.index(tid)))
                for tid in ready[: max(0, self.cfg["max_parallel"] - len(running))]:
                    th = threading.Thread(target=worker, args=(tid,), name=tid, daemon=True)
                    running[tid] = th
                    th.start()
                if not running and not ready:
                    raise RuntimeError("no task can start: " + ", ".join(left))
                self.cond.wait(1)
            while running:          # on a stop, wait for the workers to unwind
                self.cond.wait(1)
        self.check_stop()

    def run_task(self, tid):
        t = self.state["tasks"][tid]
        wt = os.path.join(self.run_dir, "worktrees", tid)
        branch = f"autobuild/{self.state['name']}/{tid}"

        def set_status(s):
            with self.mut():
                t["status"] = s

        if not t.get("base"):
            with self.mut():
                t.update(base=git(self.repo, "rev-parse", self.int_branch), branch=branch, worktree=wt,
                         started_at=now())
        self.ensure_worktree(wt, branch, t["base"])
        if not t.get("setup_done"):
            self.run_setup(wt, tid)
            with self.mut():
                t["setup_done"] = True

        brief = read_text(os.path.join(self.plan_dir, "tasks", tid + ".md"))
        if not t.get("impl_head"):
            set_status("implementing")
            self.run_agent(f"{tid}-implement", "implement", tid, self.p_implement(t, brief), wt,
                           title="implement")
            head = commit_all(wt, f"{tid}: {t['title']}", self.ident)
            with self.mut():
                t["impl_head"] = head

        def scope(agent_dir):
            return (self.diff_section(wt, t["base"], git(wt, "rev-parse", "HEAD"), agent_dir,
                                      f"Everything task {tid} changed. Your working directory is the task's "
                                      "worktree with these changes applied, so you can read around them, build "
                                      "and run tests.")
                    + "\n\nThe implementation agent's report:\n\n<implementation_report>\n"
                    + (self.agent_report(f"{tid}-implement") or "(none)") + "\n</implementation_report>")

        checklist = """- Completeness: every acceptance criterion in the brief holds. Check each one against the code; do not take the implementation report's word for it.
- Correctness: the logic is right, including edge cases and error paths the brief or the surrounding code makes relevant.
- Contracts: names, signatures, paths and data shapes match the brief exactly. Other tasks are being built against them in parallel.
- Scope: nothing outside the brief was built or changed. Work belonging to other tasks, drive-by refactors and unrelated edits are findings, because they collide with parallel tasks.
- Verification: the tests the brief asks for exist and test the behaviour for real. Run the build and the relevant tests yourself.
- Fit: the change follows the conventions of the code around it and breaks nothing that existed."""

        u = Unit(
            key=tid, cwd=wt, repo=wt, loop=t["loop"], ident=self.ident,
            target=(f"Task {tid} ({t['title']}) implemented exactly as its brief describes: complete, correct, "
                    f"and nothing beyond it.\n\n<brief>\n{brief.strip()}\n</brief>\n\n"
                    f"The task is one piece of a larger plan. For context only:\n{self.plan_pointers()}"),
            scope=scope, checklist=checklist,
            fix_note="- Stay inside this task's scope. Other tasks are being built in parallel in other worktrees.",
            set_status=set_status, commit_label=tid, guards=[wt])
        self.review_loop(u)

        self.merge_task(tid, wt)
        if not self.cfg["keep_worktrees"]:
            with self.git_lock, repo_lock(self.repo):
                git_rc(self.repo, "worktree", "remove", "--force", wt)
        with self.mut():
            t.update(status="done", ended_at=now())
        done = sum(1 for x in self.state["tasks"].values() if x["status"] == "done")
        self.log(f"{tid}: done ({done}/{len(self.state['tasks'])} tasks)")
        with self.cond:
            self.cond.notify_all()
        every = self.cfg["monitor_every"]
        if every and done % every == 0 and done < len(self.state["tasks"]):
            self.spawn_monitor(f"tasks-{done}", f"{done} of {len(self.state['tasks'])} tasks are finished")

    def merge_task(self, tid, wt):
        """Merges a reviewed task into the integration branch, resolving conflicts on the task's side."""
        t = self.state["tasks"][tid]
        if t.get("merged"):
            return
        with self.git_lock:
            self.check_stop()
            with self.mut():
                t["status"] = "merging"

            def in_merge(d):
                return os.path.exists(os.path.join(d, git(d, "rev-parse", "--git-path", "MERGE_HEAD")))

            def merge_into_integration():
                return git_rc(self.int_wt, *self.ident, "merge", "-q", "--no-ff", "-m",
                              f"Merge {tid}: {t['title']}", t["branch"])

            if in_merge(self.int_wt):       # an earlier run died in the middle of this step
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
                    self.run_agent(f"{tid}-merge", "merge", tid, self.p_merge(t, files), wt,
                                   title="resolve merge conflict")
                    left = [f for f in files if re.search(r"^(<<<<<<<|>>>>>>>) ", read_text(os.path.join(wt, f)), re.M)]
                    if left:
                        raise RuntimeError(f"{tid}: conflict markers remain in {', '.join(left)}")
                    commit_all(wt, f"Merge {self.int_branch} into {tid}", self.ident)
                    rc, out, err = merge_into_integration()
                    if rc:
                        raise RuntimeError(f"merging {tid} into {self.int_branch} failed:\n{(err or out).strip()}")
            with self.mut():
                t["merged"] = git(self.repo, "rev-parse", self.int_branch)

    # ---- phase 3: final review

    def phase_final(self):
        st = self.state["final"]
        if st["status"] == "done":
            return
        with self.mut() as s:
            s["phase"] = "final"

        def set_status(s):
            with self.mut():
                st["status"] = s

        def scope(agent_dir):
            tasks = self.state["tasks"]
            listing = "\n".join(f"- {tid}: {tasks[tid]['title']}" for tid in self.state["task_order"])
            conflicts = [f"- {tid}: {', '.join(t['conflicts'])}" for tid, t in tasks.items() if t.get("conflicts")]
            notes = ""
            if conflicts:
                notes = ("\n\nThese merges had conflicts that an agent resolved, and that resolution has not been "
                         "reviewed yet. Look at these files closely:\n" + "\n".join(conflicts))
            return (self.diff_section(self.int_wt, self.state["base_ref"], git(self.int_wt, "rev-parse", "HEAD"),
                                      agent_dir,
                                      "Everything the run changed, all tasks merged. Your working directory is the "
                                      "integration branch with these changes applied.")
                    + f"\n\nThe work was split into these tasks, each already reviewed against its own brief:\n"
                    f"{listing}\n\nThe plan is here if you need it:\n{self.plan_pointers()}{notes}")

        checklist = """- The goal: every requirement in it is met by the code as it now stands. Go through the goal point by point.
- Integration: the pieces built by separate tasks work together. Follow the paths that cross task boundaries end to end: callers and callees agree, wiring is complete, nothing was left for "another task" that no task did.
- Merge damage: duplicated or lost code, and conflict resolutions that dropped one side's behaviour.
- The build and the full test suite pass. Run them.
- Nothing unrelated to the goal was changed or broken.
Each task was already reviewed against its own brief, so put your effort into what only shows up in the whole."""

        u = Unit(
            key="final", cwd=self.int_wt, repo=self.int_wt, loop=st["loop"], ident=self.ident,
            target=f"The goal below, fully and correctly implemented in this repository.\n\n{self.goal_block()}",
            scope=scope, checklist=checklist,
            fix_note="- Your working directory is the integration branch with every task merged.",
            set_status=set_status, commit_label="final", guards=[self.int_wt])
        self.review_loop(u)
        with self.mut() as s:
            st["status"] = "done"
            s["result_head"] = git(self.int_wt, "rev-parse", "HEAD")

    # ---- monitor

    def spawn_monitor(self, mark, reason, wait=False):
        """Starts a monitor agent in the background, at most one at a time and once per mark."""
        if wait and self.monitor_thread:
            self.monitor_thread.join()
        with self.mut() as s:
            if self.monitor_thread and self.monitor_thread.is_alive():
                return
            if mark not in s["monitor_marks"]:
                s["monitor_marks"].append(mark)
            elif not wait:
                return
            agent_id = f"monitor-{s['monitor_marks'].index(mark) + 1:02d}"

            def body():
                try:
                    self.run_agent(agent_id, "monitor", "monitor", self.p_monitor(reason), self.run_dir,
                                   schema=MONITOR_SCHEMA, read_only=True, add_dirs=[self.int_wt], title=reason)
                    r = self.agent_result(agent_id)
                    write_text(os.path.join(self.agent_dir(agent_id), "report.md"),
                               f"**Health: {r.get('health', '?')}**\n\n{r.get('summary', '')}\n\n---\n\n{r.get('report', '')}")
                    self.log(f"{agent_id}: {r.get('health')} - {r.get('summary', '')}")
                except Stop:
                    pass
                except Exception as e:      # the monitor only observes; it never fails the run
                    self.log(f"{agent_id} failed: {e}")

            self.monitor_thread = threading.Thread(target=body, name=agent_id, daemon=True)
            self.monitor_thread.start()
        if wait:
            self.monitor_thread.join()

    # ---- run

    def run(self):
        with self.mut() as s:
            s.update(status="running", pid=os.getpid(), error=None, last_started_at=now())
        self.reap_stale_agents()
        try:
            self.ensure_worktree(self.int_wt, self.int_branch, self.state["base_ref"])
            if not self.state.get("integration_setup"):
                self.run_setup(self.int_wt, "integration")
                with self.mut() as s:
                    s["integration_setup"] = True
            self.phase_split()
            self.phase_tasks()
            self.phase_final()
            self.spawn_monitor("end", "the run has finished; this is the closing assessment", wait=True)
            self.check_stop()
            with self.mut() as s:
                s.update(status="completed", phase="done", ended_at=now())
            self.log(f"completed. The result is on branch {self.int_branch} ({self.int_wt}).")
            self.log(f"to take it: git merge {self.int_branch}")
            return 0
        except Stop:
            pass
        except UnitFailed as e:
            self.end_run("failed", str(e))
        except Exception as e:
            if not isinstance(e, AgentError):
                self.log(traceback.format_exc())
            self.end_run("error", str(e))
        if self.monitor_thread:
            self.monitor_thread.join()
        status, message = self.failure or ("interrupted", "stopped by a signal")
        with self.mut() as s:
            for unit in [s["split"], s["final"], *s["tasks"].values()]:
                if unit["loop"]["status"] == "failed":
                    unit["status"] = "failed"
            s.update(status=status, error=message)
        self.log(f"{status}: {message}")
        if status == "failed":
            self.log("look at the review reports, then rerun with --retry-failed to give it a fresh set of rounds")
        elif status != "interrupted":
            self.log("rerun the same command to continue from where it stopped")
        return {"failed": 2, "error": 1}.get(status, 130)


# ---------------------------------------------------------------- reports

def render_finding(f):
    out = f"## {f['id']} [{f['severity']}] {f['title']}\n"
    if f.get("location"):
        out += f"Location: {f['location']}\n"
    out += f"\n{f['details']}\n"
    if f.get("suggestion"):
        out += f"\nSuggested fix: {f['suggestion']}\n"
    return out


def render_review(agent_id, verdict, result, findings):
    blocking = sum(f["severity"] in BLOCKING for f in findings)
    out = [f"# {agent_id}: {verdict.upper()}", "",
           f"{blocking} blocking, {len(findings) - blocking} minor.", "", result.get("summary", ""), ""]
    for f in findings:
        out.append(render_finding(f))
    prev = result.get("previous_findings") or []
    if prev:
        out += ["## Previous findings", ""]
        out += [f"- {p.get('id')}: {p.get('status')}" + (f" - {p['note']}" if p.get("note") else "") for p in prev]
    return "\n".join(out)


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
    by_unit = {}
    for a in agents:
        by_unit.setdefault(a["unit"], []).append(a)

    def loop_text(loop):
        parts = []
        for r in loop["rounds"]:
            if "verdict" in r:
                blocking = sum(f["severity"] in BLOCKING for f in r["findings"])
                parts.append(f"R{r['n']}:{r['verdict']}" + (f"({blocking})" if blocking else ""))
        return " ".join(parts) or "-"

    def unit_cost(key):
        return sum(a.get("cost_usd") or 0 for a in by_unit.get(key, []))

    alive = pid_alive(s.get("pid")) and s["status"] == "running"
    lines = [f"run {s['name']}: {s['status']}" + ("" if alive or s["status"] != "running" else " (process not running)")
             + f", phase {s['phase']}",
             f"repo {s['repo']}, branch {s['integration_branch']}",
             f"agents {len(agents)}, cost ${sum(a.get('cost_usd') or 0 for a in agents):.2f}"]
    if s.get("error"):
        lines.append(f"reason: {s['error']}")
    lines += ["", f"{'unit':<8}{'status':<14}{'reviews':<34}{'cost':>8}  title / depends on"]
    lines.append(f"{'split':<8}{s['split']['status']:<14}{loop_text(s['split']['loop']):<34}{unit_cost('split'):>8.2f}")
    for tid in s["task_order"]:
        t = s["tasks"][tid]
        deps = f" <- {','.join(t['depends_on'])}" if t["depends_on"] else ""
        lines.append(f"{tid:<8}{t['status']:<14}{loop_text(t['loop']):<34}{unit_cost(tid):>8.2f}  {t['title']}{deps}")
    lines.append(f"{'final':<8}{s['final']['status']:<14}{loop_text(s['final']['loop']):<34}{unit_cost('final'):>8.2f}")
    running = [a["id"] for a in agents if a["status"] == "running"]
    if running:
        lines += ["", "running agents: " + ", ".join(running)]
    return "\n".join(lines)


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

    def clip(v, limit=20_000):
        if isinstance(v, str):
            return v if len(v) <= limit else v[:limit] + f"\n... [{len(v) - limit} more characters]"
        if isinstance(v, list):
            return [clip(x, limit) for x in v]
        if isinstance(v, dict):
            return {k: clip(x, limit) for k, x in v.items() if k not in ("signature", "data")}
        return v

    content = (e.get("message") or {}).get("content")
    if isinstance(content, str):
        content = [{"type": "text", "text": content}]
    return {"type": t, "content": clip(content or []), "parent": e.get("parent_tool_use_id"),
            "timestamp": e.get("timestamp")}


def make_handler(run_dir):
    agents_dir = os.path.join(run_dir, "agents")
    plan_dir = os.path.join(run_dir, "plan")

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
                    briefs = {}
                    tasks_dir = os.path.join(plan_dir, "tasks")
                    if os.path.isdir(tasks_dir):
                        for name in sorted(os.listdir(tasks_dir)):
                            if name.endswith(".md"):
                                briefs[name[:-3]] = read_text(os.path.join(tasks_dir, name))
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
                                          "diff": read_text(os.path.join(d, "input.diff")),
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
    base = os.path.join(repo, ".autobuild")
    runs = [os.path.join(base, n) for n in (os.listdir(base) if os.path.isdir(base) else [])]
    runs = [r for r in runs if os.path.exists(os.path.join(r, "state.json"))]
    if not runs:
        sys.exit(f"no runs under {base}; pass --run-dir")
    return max(runs, key=lambda r: os.path.getmtime(os.path.join(r, "state.json")))


def resolve_run_dir(args):
    if args.run_dir:
        return os.path.abspath(args.run_dir)
    return latest_run(repo_root(os.path.abspath(args.repo)))


def create_run(run_dir, repo, name, goal, goal_source, args):
    base = git(repo, "rev-parse", args.base or "HEAD")
    stale = git(repo, "for-each-ref", "--format=%(refname:short)", f"refs/heads/autobuild/{name}/")
    if stale:
        sys.exit(f"branches of an earlier run named {name} still exist ({', '.join(stale.split())}).\n"
                 "Delete them or start this run under another --name.")
    if not args.base and git(repo, "status", "--porcelain"):
        print("note: the checkout has uncommitted changes; the run starts from the last commit and will not see them")
    os.makedirs(run_dir, exist_ok=True)
    exclude = os.path.join(repo, git(repo, "rev-parse", "--git-path", "info/exclude"))
    if os.path.commonpath([run_dir, repo]) == repo and ".autobuild/" not in read_text(exclude).split():
        os.makedirs(os.path.dirname(exclude), exist_ok=True)
        with open(exclude, "a", encoding="utf-8") as f:
            f.write("\n.autobuild/\n")
    write_text(os.path.join(run_dir, "goal.md"), goal)
    loop = lambda: {"status": "pending", "attempt": 1, "rounds": []}  # noqa: E731
    write_json(os.path.join(run_dir, "state.json"), {
        "version": 1, "name": name, "created_at": now(), "repo": repo, "base_ref": base,
        "goal_source": goal_source, "goal_hash": hashlib.sha1(goal.encode()).hexdigest(),
        "integration_branch": f"autobuild/{name}/integration", "config": dict(DEFAULTS),
        "status": "running", "phase": "split", "error": None,
        "split": {"status": "pending", "loop": loop()}, "tasks": {}, "task_order": [],
        "final": {"status": "pending", "loop": loop()}, "monitor_marks": [],
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
        run_dir = os.path.join(repo, ".autobuild", name)
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
    if goal and hashlib.sha1(goal.encode()).hexdigest() != state["goal_hash"]:
        sys.exit(f"{run_dir} was started with a different goal; use another --name for a new run")

    lock = open(os.path.join(run_dir, "lock"), "w")
    try:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
    except OSError:
        sys.exit(f"another autobuild process is already working on {run_dir}")

    overrides = {k: getattr(args, k) for k in DEFAULTS if getattr(args, k, None) is not None}
    state["config"] = {**DEFAULTS, **state["config"], **overrides}
    failed = [u for u in [state["split"], state["final"], *state["tasks"].values()] if u["loop"]["status"] == "failed"]
    if failed and args.retry_failed:
        for u in failed:     # a fresh set of rounds, starting again from a full review
            loop = u["loop"]
            loop.setdefault("history", []).append({"attempt": loop["attempt"], "rounds": loop["rounds"]})
            loop.update(status="pending", attempt=loop["attempt"] + 1, rounds=[])
            u["status"] = "pending" if "depends_on" in u else "reviewing"
    write_json(state_path, state)
    if state["status"] == "completed":
        print(summary_text(run_dir))
        return 0
    if failed and not args.retry_failed:
        print(summary_text(run_dir))
        print("\nThis run failed its review loop. Rerun with --retry-failed to try again with a fresh set of rounds.")
        return 2

    orch = Orchestrator(run_dir)
    stops = []

    def on_signal(signum, frame):
        if stops:
            os._exit(130)
        stops.append(signum)
        print("\nstopping agents (again to force quit)...", flush=True)
        threading.Thread(target=orch.end_run, args=("interrupted", "stopped by a signal"), daemon=True).start()

    signal.signal(signal.SIGINT, on_signal)
    signal.signal(signal.SIGTERM, on_signal)
    orch.log(f"run {name} in {run_dir}")
    if not args.no_ui:
        url = start_ui(run_dir, args.port)
        if url:
            orch.log(f"UI: {url}")
    code = orch.run()
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
    run.add_argument("--retry-failed", action="store_true",
                     help="give units that ran out of review rounds a fresh set")
    run.add_argument("--max-parallel", type=int, help=f"tasks built at once (default {DEFAULTS['max_parallel']})")
    run.add_argument("--max-fix-rounds", type=int,
                     help=f"fix rounds before a unit fails (default {DEFAULTS['max_fix_rounds']})")
    run.add_argument("--monitor-every", type=int,
                     help=f"finished tasks between monitor reports, 0 for none (default {DEFAULTS['monitor_every']})")
    run.add_argument("--model", help=f"default {DEFAULTS['model']}")
    run.add_argument("--effort", help=f"default {DEFAULTS['effort']}")
    run.add_argument("--setup-cmd", help="shell command run once in every new worktree, e.g. 'cd web && npm ci'")
    run.add_argument("--agent-timeout", type=int, help="seconds before an agent is stopped and retried")
    run.add_argument("--agent-retries", type=int, help="extra attempts when an agent process fails")
    run.add_argument("--permission-mode", help=f"passed to claude (default {DEFAULTS['permission_mode']})")
    run.add_argument("--claude-bin", help="claude executable (default: claude)")
    run.add_argument("--keep-worktrees", action="store_true", default=None,
                     help="keep each task's worktree after it is merged")
    run.add_argument("--no-ui", action="store_true", help="do not serve the UI while running")
    run.set_defaults(fn=cmd_run)

    ui = sub.add_parser("ui", help="serve the UI for a run (works while it runs and after)")
    ui.set_defaults(fn=cmd_ui)
    status = sub.add_parser("status", help="print a summary of a run")
    status.set_defaults(fn=cmd_status)

    for p in (run, ui, status):
        p.add_argument("--repo", default=".", help="the git repository (default: the current directory)")
        p.add_argument("--run-dir", help="run directory (default: <repo>/.autobuild/<name>; latest run for ui/status)")
    for p in (run, ui):
        p.add_argument("--port", type=int, default=8787, help="UI port (default 8787; the next free one is used)")

    args = ap.parse_args()
    sys.exit(args.fn(args))


if __name__ == "__main__":
    main()
