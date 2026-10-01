#!/usr/bin/env python3
"""Manage Asana tasks, and markdown documents attached to them.

Every read and every write of Asana goes through this script. Standard library only.
"""

import argparse
import json
import os
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid

API_URL = "https://app.asana.com/api/1.0"
KEY_ENV = "ASANA_ACCESS_TOKEN"
MAX_RETRIES = 5                       # a rate-limited request is retried after its Retry-After
PAGE = 100                            # the largest page Asana returns
GID_RE = re.compile(r"^\d+$")
DOC_EXT = ".md"
UPLOADED = "asana"                    # the host of a file uploaded to Asana, as opposed to a link

TASK_FIELDS = "name,completed,permalink_url,parent,tags.name,dependencies,workspace"
DETAIL_FIELDS = TASK_FIELDS + ",notes,dependents,projects.name,workspace.name"
ATTACHMENT_FIELDS = "name,host,permanent_url,view_url,download_url,parent.name"


# --- helpers ---------------------------------------------------------------


def die(message):
    sys.stderr.write(message + "\n")
    sys.exit(1)


def multipart(fields):
    """Encodes fields as multipart/form-data; a (filename, bytes, type) tuple is a file."""
    boundary = uuid.uuid4().hex
    parts = []
    for name, value in fields.items():
        if isinstance(value, tuple):
            filename, content, content_type = value
            head = ('--%s\r\nContent-Disposition: form-data; name="%s"; filename="%s"\r\n'
                    'Content-Type: %s\r\n\r\n' % (boundary, name, filename, content_type))
        else:
            head = '--%s\r\nContent-Disposition: form-data; name="%s"\r\n\r\n' % (boundary, name)
            content = str(value).encode("utf-8")
        parts.append(head.encode("utf-8") + content + b"\r\n")
    parts.append(("--%s--\r\n" % boundary).encode("utf-8"))
    return b"".join(parts), "multipart/form-data; boundary=" + boundary


def api(method, path, query=None, body=None, form=None):
    """Runs one request against Asana; returns the response payload or dies."""
    key = os.environ.get(KEY_ENV, "").strip()
    if key == "":
        die(KEY_ENV + " is not set")
    url = API_URL + path + ("?" + urllib.parse.urlencode(query) if query else "")
    headers = {"Authorization": "Bearer " + key, "Accept": "application/json"}
    data = None
    if form is not None:
        data, headers["Content-Type"] = multipart(form)
    elif body is not None:
        data = json.dumps({"data": body}).encode("utf-8")
        headers["Content-Type"] = "application/json"
    for attempt in range(MAX_RETRIES + 1):
        request = urllib.request.Request(url, data=data, headers=headers, method=method)
        try:
            with urllib.request.urlopen(request, timeout=60) as response:
                raw = response.read().decode("utf-8")
            return json.loads(raw) if raw.strip() else {"data": {}}
        except urllib.error.HTTPError as err:
            if err.code == 429 and attempt < MAX_RETRIES:
                time.sleep(float(err.headers.get("Retry-After") or 1))
                continue
            raw = err.read().decode("utf-8", "replace")
            try:
                errors = json.loads(raw).get("errors") or []
            except ValueError:
                errors = []
            if not errors:
                die("Asana API HTTP " + str(err.code) + ": " + raw[:300])
            die("Asana API error: " + "; ".join(e.get("message", "unknown error") for e in errors))
        except urllib.error.URLError as err:
            die("cannot reach the Asana API: " + str(err.reason))


def download(url):
    """Fetches an attachment's content from its short-lived download URL, which takes no token."""
    try:
        with urllib.request.urlopen(url, timeout=60) as response:
            return response.read().decode("utf-8", "replace")
    except (urllib.error.HTTPError, urllib.error.URLError) as err:
        die("cannot download document content: " + str(err))


def call(method, path, query=None, body=None, form=None):
    return api(method, path, query, body, form)["data"]


def fetch_all(path, query=None):
    query = dict(query or {}, limit=PAGE)
    out = []
    while True:
        payload = api("GET", path, query)
        out.extend(payload["data"])
        if not payload.get("next_page"):
            return out
        query["offset"] = payload["next_page"]["offset"]


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


def read_stdin(what):
    text = sys.stdin.read()
    if text.strip() == "":
        die(what + " is empty; provide it on stdin")
    return text


def url_segments(ref):
    return [s for s in urllib.parse.urlparse(ref).path.split("/") if s]


def segment_after(segments, name):
    if name in segments:
        i = segments.index(name)
        if i + 1 < len(segments) and GID_RE.match(segments[i + 1]):
            return segments[i + 1]
    return None


def task_gid(ref):
    """Accepts a task gid or an Asana task URL; returns the gid."""
    ref = ref.strip()
    if GID_RE.match(ref):
        return ref
    if ref.startswith("http"):
        segments = url_segments(ref)
        gid = segment_after(segments, "task")
        if gid is None and len(segments) >= 3 and segments[0] == "0" and GID_RE.match(segments[2]):
            gid = segments[2]                                  # /0/<project>/<task>
        if gid:
            return gid
    die("not a task id or task URL: " + ref)


def project_gid(ref):
    """Returns the gid of a project gid or project URL, or None for a project name."""
    if GID_RE.match(ref):
        return ref
    if not ref.startswith("http"):
        return None
    segments = url_segments(ref)
    gid = segment_after(segments, "project")
    if gid is None and len(segments) >= 2 and segments[0] == "0" and GID_RE.match(segments[1]):
        gid = segments[1]                                      # /0/<project>/...
    if gid is None or gid == "0":
        die("not a project URL: " + ref)
    return gid


def document_gid(ref):
    """Accepts a document id or its permanent URL; returns the attachment gid."""
    ref = ref.strip()
    if GID_RE.match(ref):
        return ref
    asset = urllib.parse.parse_qs(urllib.parse.urlparse(ref).query).get("asset_id")
    if ref.startswith("http") and asset and GID_RE.match(asset[0]):
        return asset[0]
    die("not a document id or document URL: " + ref)


def clean_title(title):
    title = title.strip()
    if title.lower().endswith(DOC_EXT):
        title = title[:-len(DOC_EXT)].strip()
    if title == "":
        die("title is empty")
    return title


def fetch_task(ref):
    return call("GET", "/tasks/" + task_gid(ref), {"opt_fields": DETAIL_FIELDS})


def existing_task_gid(ref):
    return call("GET", "/tasks/" + task_gid(ref), {"opt_fields": "name"})["gid"]


def blocker_gids(task):
    return [d["gid"] for d in task["dependencies"]]


def completion_lookup(known=()):
    """Returns completed(gid), asking Asana only for tasks not already known."""
    cache = {t["gid"]: t["completed"] for t in known}

    def completed(gid):
        if gid not in cache:
            cache[gid] = call("GET", "/tasks/" + gid, {"opt_fields": "completed"})["completed"]
        return cache[gid]
    return completed


def tag_names(task):
    return [tag["name"] for tag in task["tags"]]


def has_tags(task, tags):
    names = {n.lower() for n in tag_names(task)}
    return all(tag.lower() in names for tag in tags)


def public(task):
    return {
        "id": task["gid"],
        "title": task["name"],
        "completed": task["completed"],
        "parent": task["parent"]["gid"] if task["parent"] else None,
        "blockedBy": blocker_gids(task),
        "tags": tag_names(task),
        "url": task["permalink_url"],
    }


def is_uploaded(attachment):
    return attachment["host"] == UPLOADED


def link_urls(attachment):
    return {u for u in (attachment.get("view_url"), attachment.get("download_url")) if u}


def attachments_of(gid):
    return fetch_all("/attachments", {"parent": gid, "opt_fields": ATTACHMENT_FIELDS})


def detail(task):
    out = public(task)
    out["workspace"] = task["workspace"]["name"]
    out["projects"] = [p["name"] for p in task["projects"]]
    out["subtasks"] = [s["gid"] for s in fetch_all("/tasks/" + task["gid"] + "/subtasks",
                                                   {"opt_fields": "name"})]
    out["blocks"] = [d["gid"] for d in task["dependents"]]
    attachments = attachments_of(task["gid"])
    out["documents"] = [{"id": a["gid"], "title": clean_title(a["name"]), "url": a["permanent_url"]}
                        for a in attachments if is_uploaded(a)]
    out["links"] = [{"title": a["name"], "url": a.get("view_url") or a.get("download_url")}
                    for a in attachments if not is_uploaded(a)]
    return out


def public_document(attachment):
    parent = attachment["parent"] or {}
    kind = parent.get("resource_type")
    return {
        "id": attachment["gid"],
        "title": clean_title(attachment["name"]),
        "url": attachment["permanent_url"],
        "task": parent["gid"] if kind == "task" else None,
        "project": parent["name"] if kind == "project" else None,
    }


def all_workspaces():
    return fetch_all("/workspaces", {"opt_fields": "name"})


def resolve_workspace(ref):
    ref = ref.strip()
    found = [w for w in all_workspaces() if ref == w["gid"] or ref.lower() == w["name"].lower()]
    if not found:
        die("unknown workspace: " + ref)
    if len(found) > 1:
        die("workspace name is ambiguous, give its id: " + ref)
    return found[0]


def workspace_projects(workspace):
    return fetch_all("/workspaces/" + workspace["gid"] + "/projects",
                     {"archived": "false", "opt_fields": "name,workspace"})


def resolve_project(ref, workspace=None):
    ref = ref.strip()
    gid = project_gid(ref)
    if gid:
        return call("GET", "/projects/" + gid, {"opt_fields": "name,workspace"})
    spaces = [resolve_workspace(workspace)] if workspace else all_workspaces()
    found = [p for space in spaces for p in workspace_projects(space)
             if p["name"].lower() == ref.lower()]
    if not found:
        die("unknown project: " + ref)
    if len(found) > 1:
        die("project name is ambiguous, give its id or --workspace: " + ref)
    return found[0]


def resolve_tags(names, workspace_gid):
    """Returns tag gids for names, creating a workspace tag for any name that does not exist."""
    path = "/workspaces/" + workspace_gid + "/tags"
    existing = fetch_all(path, {"opt_fields": "name"})
    ids = []
    for name in names:
        match = next((t for t in existing if t["name"].lower() == name.lower()), None)
        if match is None:
            match = call("POST", path, {"opt_fields": "name"}, body={"name": name})
            existing.append(match)
        ids.append(match["gid"])
    return ids


def would_cycle(gid, blockers):
    """True when gid already blocks, directly or transitively, any of blockers."""
    stack, seen = list(blockers), set()
    while stack:
        current = stack.pop()
        if current == gid:
            return True
        if current in seen:
            continue
        seen.add(current)
        stack.extend(blocker_gids(call("GET", "/tasks/" + current, {"opt_fields": "dependencies"})))
    return False


def print_json(value):
    print(json.dumps(value, indent=2))


# --- task commands ---------------------------------------------------------


def cmd_workspaces(args):
    print_json([{"id": w["gid"], "name": w["name"]} for w in all_workspaces()])
    return 0


def cmd_projects(args):
    spaces = [resolve_workspace(args.workspace)] if args.workspace else all_workspaces()
    print_json([{"id": p["gid"], "name": p["name"], "workspace": space["name"]}
                for space in spaces for p in workspace_projects(space)])
    return 0


def cmd_create(args):
    if not args.project and not args.parent:
        die("give --project or --parent")
    if args.workspace and not args.project:
        die("--workspace only narrows --project")
    title = args.title.strip()
    if title == "":
        die("title is empty")
    body = read_stdin("task body")
    parent = fetch_task(args.parent) if args.parent else None
    project = resolve_project(args.project, args.workspace) if args.project else None
    workspace_gid = project["workspace"]["gid"] if project else parent["workspace"]["gid"]
    blockers = normalize_strings([existing_task_gid(b) for b in normalize_strings(args.blocked_by)])
    fields = {"name": title, "notes": body}
    if parent:
        fields["parent"] = parent["gid"]
    if project:
        fields["projects"] = [project["gid"]]
    tags = normalize_strings(args.tag)
    if tags:
        fields["tags"] = resolve_tags(tags, workspace_gid)
    created = call("POST", "/tasks", {"opt_fields": "name"}, body=fields)
    if blockers:
        call("POST", "/tasks/" + created["gid"] + "/addDependencies", body={"dependencies": blockers})
    print_json(public(fetch_task(created["gid"])))
    return 0


def cmd_get(args):
    print_json(detail(fetch_task(args.id)))
    return 0


def cmd_body(args):
    body = fetch_task(args.id)["notes"] or ""
    sys.stdout.write(body if body.endswith("\n") else body + "\n")
    return 0


def cmd_list(args):
    if args.parent and (args.project or args.workspace):
        die("give either --parent, or --project with an optional --workspace")
    query = {"opt_fields": TASK_FIELDS}
    if args.parent:
        path = "/tasks/" + task_gid(args.parent) + "/subtasks"
    elif args.project:
        path = "/projects/" + resolve_project(args.project, args.workspace)["gid"] + "/tasks"
        if args.filter != "all":
            query["completed_since"] = "now"                  # incomplete tasks only
    else:
        die("give --parent or --project")
    selected = fetch_all(path, query)
    if args.filter != "all":
        selected = [t for t in selected if not t["completed"]]
    if args.filter == "unblocked":
        completed = completion_lookup(selected)
        selected = [t for t in selected if all(completed(b) for b in blocker_gids(t))]
    # Blocking above is judged against every blocker, inside the scope or not; the tag filter only
    # narrows which of the resulting tasks are shown.
    tags = normalize_strings(args.tag)
    if tags:
        selected = [t for t in selected if has_tags(t, tags)]
    print_json([public(t) for t in selected])
    return 0


def cmd_complete(args):
    task = fetch_task(args.id)
    if task["completed"]:
        print_json(public(task))
        return 0
    completed = completion_lookup()
    for blocker in blocker_gids(task):
        if not completed(blocker):
            die("cannot complete " + task["gid"] + ": blocker " + blocker + " is not completed")
    call("PUT", "/tasks/" + task["gid"], body={"completed": True})
    print_json(public(fetch_task(task["gid"])))
    return 0


def cmd_block(args):
    task = fetch_task(args.id)
    refs = normalize_strings(args.blocked_by)
    if not refs:
        die("no --blocked-by given")
    existing = set(blocker_gids(task))
    add = []
    for ref in refs:
        blocker = existing_task_gid(ref)
        if blocker == task["gid"]:
            die("a task cannot block itself: " + ref)
        if blocker not in existing and blocker not in add:
            add.append(blocker)
    if add and would_cycle(task["gid"], add):
        die("blocking " + task["gid"] + " by " + ", ".join(add) + " would create a cycle")
    if add:
        call("POST", "/tasks/" + task["gid"] + "/addDependencies", body={"dependencies": add})
    print_json(public(fetch_task(task["gid"])))
    return 0


def cmd_unblock(args):
    task = fetch_task(args.id)
    refs = normalize_strings(args.blocked_by)
    if not refs:
        die("no --blocked-by given")
    existing = blocker_gids(task)
    remove = []
    for ref in refs:
        blocker = existing_task_gid(ref)
        if blocker not in existing:
            die(blocker + " does not block " + task["gid"])
        if blocker not in remove:
            remove.append(blocker)
    call("POST", "/tasks/" + task["gid"] + "/removeDependencies", body={"dependencies": remove})
    print_json(public(fetch_task(task["gid"])))
    return 0


def cmd_tag(args):
    task = fetch_task(args.id)
    tags = normalize_strings(args.tag)
    if not tags:
        die("no --tag given")
    present = {n.lower() for n in tag_names(task)}
    missing = [t for t in tags if t.lower() not in present]
    for tag in resolve_tags(missing, task["workspace"]["gid"]) if missing else []:
        call("POST", "/tasks/" + task["gid"] + "/addTag", body={"tag": tag})
    print_json(public(fetch_task(task["gid"])))
    return 0


def cmd_set_parent(args):
    task = existing_task_gid(args.id)
    parent = existing_task_gid(args.parent)
    if parent == task:
        die("a task cannot be its own parent: " + args.parent)
    call("POST", "/tasks/" + task + "/setParent", body={"parent": parent})
    print_json(public(fetch_task(task)))
    return 0


def cmd_update(args):
    task = existing_task_gid(args.id)
    fields = {"notes": read_stdin("task body")}
    if args.title is not None:
        if args.title.strip() == "":
            die("title is empty")
        fields["name"] = args.title.strip()
    call("PUT", "/tasks/" + task, {"opt_fields": "name"}, body=fields)
    print_json(public(fetch_task(task)))
    return 0


# --- document commands -----------------------------------------------------


def fetch_document(ref):
    document = call("GET", "/attachments/" + document_gid(ref), {"opt_fields": ATTACHMENT_FIELDS})
    if not is_uploaded(document):
        die("not a document, but a link: " + ref)
    return document


def upload_document(parent_gid, title, content):
    filename = (title + DOC_EXT).replace('"', "'").replace("\r", " ").replace("\n", " ")
    created = call("POST", "/attachments", form={
        "parent": parent_gid, "file": (filename, content.encode("utf-8"), "text/markdown")})
    return fetch_document(created["gid"])


def cmd_doc_create(args):
    if bool(args.task) == bool(args.project):
        die("give exactly one of --task or --project")
    if args.workspace and not args.project:
        die("--workspace only narrows --project")
    title = clean_title(args.title)
    content = read_stdin("document content")
    if args.task:
        parent = existing_task_gid(args.task)
    else:
        parent = resolve_project(args.project, args.workspace)["gid"]
    print_json(public_document(upload_document(parent, title, content)))
    return 0


def cmd_doc_get(args):
    print_json(public_document(fetch_document(args.doc)))
    return 0


def cmd_doc_content(args):
    document = fetch_document(args.doc)
    if not document.get("download_url"):
        die("document has no downloadable content: " + args.doc)
    content = download(document["download_url"])
    sys.stdout.write(content if content.endswith("\n") else content + "\n")
    return 0


def cmd_doc_update(args):
    old = fetch_document(args.doc)
    title = clean_title(args.title) if args.title is not None else clean_title(old["name"])
    content = read_stdin("document content")
    new = upload_document(old["parent"]["gid"], title, content)
    call("DELETE", "/attachments/" + old["gid"])
    out = public_document(new)
    out["replaced"] = {"id": old["gid"], "url": old["permanent_url"]}
    print_json(out)
    return 0


def cmd_doc_link(args):
    document = fetch_document(args.doc)
    task = existing_task_gid(args.task)
    linked = set()
    for attachment in attachments_of(task):
        if not is_uploaded(attachment):
            linked |= link_urls(attachment)
    if document["permanent_url"] not in linked:
        call("POST", "/attachments", form={
            "parent": task, "resource_subtype": "external",
            "name": clean_title(document["name"]), "url": document["permanent_url"]})
    print_json(detail(fetch_task(task)))
    return 0


# --- wiring ----------------------------------------------------------------


def build_parser():
    scope = argparse.ArgumentParser(add_help=False)
    scope.add_argument("--parent")
    scope.add_argument("--project")
    scope.add_argument("--workspace")
    scope.add_argument("--tag", action="append")

    parser = argparse.ArgumentParser(
        description="Manage Asana tasks, and markdown documents attached to them.")
    sub = parser.add_subparsers(dest="command")
    sub.required = True

    p = sub.add_parser("workspaces")
    p.set_defaults(func=cmd_workspaces)

    p = sub.add_parser("projects")
    p.add_argument("--workspace")
    p.set_defaults(func=cmd_projects)

    p = sub.add_parser("create")
    p.add_argument("--title", required=True)
    p.add_argument("--project")
    p.add_argument("--workspace")
    p.add_argument("--parent")
    p.add_argument("--blocked-by", action="append")
    p.add_argument("--tag", action="append")
    p.set_defaults(func=cmd_create)

    p = sub.add_parser("get")
    p.add_argument("--id", required=True)
    p.set_defaults(func=cmd_get)

    p = sub.add_parser("body")
    p.add_argument("--id", required=True)
    p.set_defaults(func=cmd_body)

    p = sub.add_parser("list", parents=[scope])
    p.set_defaults(func=cmd_list, filter="all")

    p = sub.add_parser("list-pending", parents=[scope])
    p.set_defaults(func=cmd_list, filter="pending")

    p = sub.add_parser("list-unblocked", parents=[scope])
    p.set_defaults(func=cmd_list, filter="unblocked")

    p = sub.add_parser("complete")
    p.add_argument("--id", required=True)
    p.set_defaults(func=cmd_complete)

    p = sub.add_parser("block")
    p.add_argument("--id", required=True)
    p.add_argument("--blocked-by", action="append")
    p.set_defaults(func=cmd_block)

    p = sub.add_parser("unblock")
    p.add_argument("--id", required=True)
    p.add_argument("--blocked-by", action="append")
    p.set_defaults(func=cmd_unblock)

    p = sub.add_parser("tag")
    p.add_argument("--id", required=True)
    p.add_argument("--tag", action="append")
    p.set_defaults(func=cmd_tag)

    p = sub.add_parser("set-parent")
    p.add_argument("--id", required=True)
    p.add_argument("--parent", required=True)
    p.set_defaults(func=cmd_set_parent)

    p = sub.add_parser("update")
    p.add_argument("--id", required=True)
    p.add_argument("--title")
    p.set_defaults(func=cmd_update)

    p = sub.add_parser("doc-create")
    p.add_argument("--title", required=True)
    p.add_argument("--task")
    p.add_argument("--project")
    p.add_argument("--workspace")
    p.set_defaults(func=cmd_doc_create)

    p = sub.add_parser("doc-get")
    p.add_argument("--doc", required=True)
    p.set_defaults(func=cmd_doc_get)

    p = sub.add_parser("doc-content")
    p.add_argument("--doc", required=True)
    p.set_defaults(func=cmd_doc_content)

    p = sub.add_parser("doc-update")
    p.add_argument("--doc", required=True)
    p.add_argument("--title")
    p.set_defaults(func=cmd_doc_update)

    p = sub.add_parser("doc-link")
    p.add_argument("--doc", required=True)
    p.add_argument("--task", required=True)
    p.set_defaults(func=cmd_doc_link)

    return parser


def main(argv=None):
    args = build_parser().parse_args(argv)
    return args.func(args)


if __name__ == "__main__":
    sys.exit(main())
