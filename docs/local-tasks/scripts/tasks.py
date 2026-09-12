#!/usr/bin/env python3
"""Manage tasks stored as markdown files in a flat root folder.

Every read and every write of the task store goes through this script. Standard library only.
"""

import argparse
import json
import os
import re
import sys

FIELDS = ("id", "title", "blockedBy", "tags", "completed")   # also the write order
ID_WIDTH = 3
SLUG_MAX = 50


# --- helpers ---------------------------------------------------------------


def die(message):
    sys.stderr.write(message + "\n")
    sys.exit(1)


def id_sort_key(task_id):
    if task_id.isdigit():
        return (0, int(task_id))
    return (1, task_id)


def slugify(title):
    s = title.lower()
    s = re.sub(r"[^a-z0-9]+", "-", s)
    s = s.strip("-")[:SLUG_MAX].rstrip("-")
    return s or "task"


def parse_task_file(path):
    with open(path, encoding="utf-8") as handle:
        text = handle.read()
    lines = text.split("\n")
    if not lines or lines[0].strip() != "---":
        die("malformed task file (no frontmatter): " + path)
    close = None
    for i in range(1, len(lines)):
        if lines[i].strip() == "---":
            close = i
            break
    if close is None:
        die("malformed task file (unterminated frontmatter): " + path)
    fields = {}
    for line in lines[1:close]:
        if line.strip() == "":
            continue
        if ": " not in line:
            die("malformed frontmatter line in " + path + ": " + line)
        key, raw = line.split(": ", 1)
        key = key.strip()
        if key not in FIELDS:
            die("unknown frontmatter field '" + key + "' in " + path)
        try:
            value = json.loads(raw.strip())
        except Exception:
            die("frontmatter value for '" + key + "' in " + path + " is not valid JSON: " + raw)
        fields[key] = value
    fields.setdefault("tags", [])   # stores written before tags existed have no tags line
    for key in FIELDS:
        if key not in fields:
            die("missing frontmatter field '" + key + "' in " + path)
    if not isinstance(fields["id"], str):
        die("'id' must be a string in " + path)
    if not isinstance(fields["title"], str):
        die("'title' must be a string in " + path)
    if not isinstance(fields["completed"], bool):
        die("'completed' must be a boolean in " + path)
    if not (isinstance(fields["blockedBy"], list)
            and all(isinstance(x, str) for x in fields["blockedBy"])):
        die("'blockedBy' must be a list of strings in " + path)
    if not (isinstance(fields["tags"], list)
            and all(isinstance(x, str) for x in fields["tags"])):
        die("'tags' must be a list of strings in " + path)
    body = "\n".join(lines[close + 1:]).lstrip("\n")
    return {**fields, "body": body, "path": path}


def format_task_file(task):
    out = "---\n"
    for key in FIELDS:
        out += key + ": " + json.dumps(task[key]) + "\n"
    out += "---\n\n" + task["body"].rstrip("\n") + "\n"
    return out


def write_task(task):
    with open(task["path"], "w", encoding="utf-8") as handle:
        handle.write(format_task_file(task))


def load_store(root):
    if not os.path.isdir(root):
        die("task store root does not exist: " + root)
    tasks, seen = [], {}
    for name in sorted(os.listdir(root)):
        p = os.path.join(root, name)
        if os.path.isdir(p):
            die("unexpected directory in task store: " + p)
        if not name.endswith(".md"):
            die("unexpected file in task store: " + p)
        t = parse_task_file(p)
        if t["id"] in seen:
            die("duplicate task id " + t["id"] + ": " + seen[t["id"]] + " and " + p)
        seen[t["id"]] = p
        tasks.append(t)
    ids = set(seen)
    for t in tasks:
        for b in t["blockedBy"]:
            if b not in ids:
                die("task " + t["id"] + " is blocked by unknown task " + b)
    tasks.sort(key=lambda t: id_sort_key(t["id"]))
    return tasks


def find(tasks, task_id):
    for t in tasks:
        if t["id"] == task_id:
            return t
    return None


def public(task):
    return {k: task[k] for k in FIELDS}


def next_id(tasks):
    highest = 0
    for t in tasks:
        if t["id"].isdigit():
            highest = max(highest, int(t["id"]))
    return str(highest + 1).zfill(ID_WIDTH)


def normalize_strings(values):
    if values is None:
        return []
    out = []
    for value in values:
        value = value.strip()
        if value == "":
            continue
        if value not in out:
            out.append(value)
    return out


def is_unblocked(task, by_id):
    if task["completed"]:
        return False
    return all(by_id[b]["completed"] for b in task["blockedBy"])


def has_tags(task, tags):
    return all(tag in task["tags"] for tag in tags)


def would_cycle(tasks, task_id, blockers):
    edges = {t["id"]: list(t["blockedBy"]) for t in tasks}
    edges[task_id] = list(blockers)
    stack, seen = list(blockers), set()
    while stack:
        cur = stack.pop()
        if cur == task_id:
            return True
        if cur in seen:
            continue
        seen.add(cur)
        stack.extend(edges.get(cur, []))
    return False


# --- commands --------------------------------------------------------------


def cmd_exists(args):
    print(json.dumps({"exists": os.path.isdir(args.root)}, indent=2))
    return 0


def cmd_create(args):
    body = sys.stdin.read()
    if body.strip() == "":
        die("task body is empty; provide it on stdin")
    title = args.title.strip()
    if title == "":
        die("title is empty")
    blockers = normalize_strings(args.blocked_by)
    tags = normalize_strings(args.tag)
    if not os.path.isdir(args.root):
        os.makedirs(args.root)
    tasks = load_store(args.root)
    known = {t["id"] for t in tasks}
    for b in blockers:
        if b not in known:
            die("unknown blocker task: " + b)
    task = {
        "id": next_id(tasks),
        "title": title,
        "blockedBy": blockers,
        "tags": tags,
        "completed": False,
        "body": body,
    }
    task["path"] = os.path.join(args.root, task["id"] + "-" + slugify(title) + ".md")
    if os.path.exists(task["path"]):
        die("task file already exists: " + task["path"])
    write_task(task)
    print(json.dumps(public(task), indent=2))
    return 0


def cmd_complete(args):
    tasks = load_store(args.root)
    task = find(tasks, args.id) or die("unknown task: " + args.id)
    if task["completed"]:
        print(json.dumps(public(task), indent=2))
        return 0
    for b in task["blockedBy"]:
        if not find(tasks, b)["completed"]:
            die("cannot complete " + task["id"] + ": blocker " + b + " is not completed")
    task["completed"] = True
    write_task(task)
    print(json.dumps(public(task), indent=2))
    return 0


def cmd_block(args):
    tasks = load_store(args.root)
    task = find(tasks, args.id) or die("unknown task: " + args.id)
    add = normalize_strings(args.blocked_by)
    if not add:
        die("no --blocked-by given")
    known = {t["id"] for t in tasks}
    for b in add:
        if b == task["id"]:
            die("a task cannot block itself: " + b)
        if b not in known:
            die("unknown blocker task: " + b)
    merged = task["blockedBy"] + [b for b in add if b not in task["blockedBy"]]
    if would_cycle(tasks, task["id"], merged):
        die("blocking " + task["id"] + " by " + ", ".join(add) + " would create a cycle")
    task["blockedBy"] = merged
    write_task(task)
    print(json.dumps(public(task), indent=2))
    return 0


def cmd_tag(args):
    tasks = load_store(args.root)
    task = find(tasks, args.id) or die("unknown task: " + args.id)
    add = normalize_strings(args.tag)
    if not add:
        die("no --tag given")
    task["tags"] = task["tags"] + [t for t in add if t not in task["tags"]]
    write_task(task)
    print(json.dumps(public(task), indent=2))
    return 0


def cmd_list(args):
    tasks = load_store(args.root)
    by_id = {t["id"]: t for t in tasks}
    if args.filter == "pending":
        selected = [t for t in tasks if not t["completed"]]
    elif args.filter == "unblocked":
        selected = [t for t in tasks if is_unblocked(t, by_id)]
    else:
        selected = tasks
    # Blocking above is judged against every task in the store; the tag filter only narrows
    # which of the resulting tasks are shown.
    tags = normalize_strings(args.tag)
    if tags:
        selected = [t for t in selected if has_tags(t, tags)]
    print(json.dumps([public(t) for t in selected], indent=2))
    return 0


def cmd_body(args):
    tasks = load_store(args.root)
    task = find(tasks, args.id) or die("unknown task: " + args.id)
    body = task["body"]
    sys.stdout.write(body if body.endswith("\n") else body + "\n")
    return 0


# --- wiring ----------------------------------------------------------------


def build_parser():
    common = argparse.ArgumentParser(add_help=False)
    common.add_argument("--root", required=True)

    listing = argparse.ArgumentParser(add_help=False)
    listing.add_argument("--tag", action="append")

    parser = argparse.ArgumentParser(
        description="Manage tasks stored as markdown files in a flat root folder.")
    sub = parser.add_subparsers(dest="command")
    sub.required = True

    p = sub.add_parser("create", parents=[common])
    p.add_argument("--title", required=True)
    p.add_argument("--blocked-by", action="append")
    p.add_argument("--tag", action="append")
    p.set_defaults(func=cmd_create)

    p = sub.add_parser("complete", parents=[common])
    p.add_argument("--id", required=True)
    p.set_defaults(func=cmd_complete)

    p = sub.add_parser("block", parents=[common])
    p.add_argument("--id", required=True)
    p.add_argument("--blocked-by", action="append")
    p.set_defaults(func=cmd_block)

    p = sub.add_parser("tag", parents=[common])
    p.add_argument("--id", required=True)
    p.add_argument("--tag", action="append")
    p.set_defaults(func=cmd_tag)

    p = sub.add_parser("list", parents=[common, listing])
    p.set_defaults(func=cmd_list, filter="all")

    p = sub.add_parser("list-pending", parents=[common, listing])
    p.set_defaults(func=cmd_list, filter="pending")

    p = sub.add_parser("list-unblocked", parents=[common, listing])
    p.set_defaults(func=cmd_list, filter="unblocked")

    p = sub.add_parser("body", parents=[common])
    p.add_argument("--id", required=True)
    p.set_defaults(func=cmd_body)

    p = sub.add_parser("exists", parents=[common])
    p.set_defaults(func=cmd_exists)

    return parser


def main(argv=None):
    args = build_parser().parse_args(argv)
    return args.func(args)


if __name__ == "__main__":
    sys.exit(main())
