#!/usr/bin/env python3
"""Tests for asana.py against an in-memory fake of the Asana API. Run with: python3 asana_test.py"""

import contextlib
import io
import json
import os
import re
import sys
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import asana  # noqa: E402


def compact(value):
    if isinstance(value, list):
        return [compact(v) for v in value]
    if isinstance(value, dict):
        return {k: value[k] for k in ("gid", "resource_type", "name") if k in value}
    return value


def select(value, fields):
    """Keeps only the requested opt_fields, as Asana does; gid and resource_type always come back."""
    if isinstance(value, list):
        return [select(v, fields) for v in value]
    if not isinstance(value, dict):
        return value
    out = {k: value[k] for k in ("gid", "resource_type") if k in value}
    groups = {}
    for field in fields:
        head, _, rest = field.partition(".")
        groups.setdefault(head, [])
        if rest:
            groups[head].append(rest)
    for head, rest in groups.items():
        if head not in value:
            raise AssertionError("opt_field the fake does not know: " + head)
        out[head] = select(value[head], rest) if rest else compact(value[head])
    return out


class FakeAsana:
    """Answers the requests asana.py sends, by method and path, from in-memory state."""

    ROUTES = [
        ("GET", r"/workspaces", "list_workspaces"),
        ("GET", r"/workspaces/(\d+)/projects", "list_projects"),
        ("GET", r"/workspaces/(\d+)/tags", "list_tags"),
        ("POST", r"/workspaces/(\d+)/tags", "create_tag"),
        ("GET", r"/projects/(\d+)", "get_project"),
        ("GET", r"/projects/(\d+)/tasks", "project_tasks"),
        ("POST", r"/tasks", "create_task"),
        ("GET", r"/tasks/(\d+)", "get_task"),
        ("PUT", r"/tasks/(\d+)", "update_task"),
        ("GET", r"/tasks/(\d+)/subtasks", "subtasks"),
        ("POST", r"/tasks/(\d+)/addDependencies", "add_dependencies"),
        ("POST", r"/tasks/(\d+)/removeDependencies", "remove_dependencies"),
        ("POST", r"/tasks/(\d+)/addTag", "add_tag_to_task"),
        ("POST", r"/tasks/(\d+)/setParent", "set_parent"),
        ("GET", r"/attachments", "list_attachments"),
        ("POST", r"/attachments", "create_attachment"),
        ("GET", r"/attachments/(\d+)", "get_attachment"),
        ("DELETE", r"/attachments/(\d+)", "delete_attachment"),
    ]

    def __init__(self):
        self.workspaces, self.projects, self.tags, self.tasks, self.attachments = [], [], [], [], []
        self.calls = []
        self.counter = 1000

    # --- seeding ---

    def next_gid(self):
        self.counter += 1
        return str(self.counter)

    def add_workspace(self, name):
        workspace = {"gid": self.next_gid(), "resource_type": "workspace", "name": name}
        self.workspaces.append(workspace)
        return workspace

    def add_project(self, name, workspace, archived=False):
        project = {"gid": self.next_gid(), "resource_type": "project", "name": name,
                   "workspace": workspace["gid"], "archived": archived}
        self.projects.append(project)
        return project

    def add_tag(self, name, workspace_gid):
        tag = {"gid": self.next_gid(), "resource_type": "tag", "name": name, "workspace": workspace_gid}
        self.tags.append(tag)
        return tag

    def add_link(self, task_gid, name, url):
        self.attachments.append({"gid": self.next_gid(), "name": name, "parent": task_gid,
                                 "host": "external", "url": url, "content": None})

    def set_completed(self, gid, value=True):
        self.find(self.tasks, gid, "task")["completed"] = value

    # --- lookups ---

    def fail(self, message):
        asana.die("Asana API error: " + message)

    def find(self, items, gid, kind):
        for item in items:
            if item["gid"] == gid:
                return item
        self.fail(kind + ": Unknown object: " + gid)

    def parent_object(self, gid):
        for item in self.tasks + self.projects:
            if item["gid"] == gid:
                return item
        self.fail("parent: Unknown object: " + gid)

    def render_task(self, task):
        workspace = self.find(self.workspaces, task["workspace"], "workspace")
        return {
            "gid": task["gid"], "resource_type": "task", "name": task["name"], "notes": task["notes"],
            "completed": task["completed"],
            "permalink_url": "https://app.asana.com/1/%s/task/%s" % (workspace["gid"], task["gid"]),
            "parent": self.find(self.tasks, task["parent"], "task") if task["parent"] else None,
            "tags": [self.find(self.tags, t, "tag") for t in task["tags"]],
            "dependencies": [self.find(self.tasks, d, "task") for d in task["dependencies"]],
            "dependents": [t for t in self.tasks if task["gid"] in t["dependencies"]],
            "projects": [self.find(self.projects, p, "project") for p in task["projects"]],
            "workspace": workspace,
        }

    def render_project(self, project):
        return dict(project, workspace=self.find(self.workspaces, project["workspace"], "workspace"))

    def render_attachment(self, attachment):
        gid = attachment["gid"]
        permanent = "https://app.asana.com/app/asana/-/get_asset?asset_id=" + gid
        uploaded = attachment["host"] == "asana"
        return {
            "gid": gid, "resource_type": "attachment", "name": attachment["name"],
            "host": attachment["host"], "permanent_url": permanent,
            "view_url": permanent if uploaded else attachment["url"],
            "download_url": "https://s3.fake/" + gid if uploaded else attachment["url"],
            "parent": self.parent_object(attachment["parent"]),
        }

    # --- the API ---

    def api(self, method, path, query=None, body=None, form=None):
        query = query or {}
        self.calls.append((method, path))
        for route_method, pattern, handler in self.ROUTES:
            match = re.fullmatch(pattern, path)
            if route_method == method and match:
                result = getattr(self, handler)(*match.groups(), query=query, body=body, form=form)
                break
        else:
            raise AssertionError("no fake route for " + method + " " + path)
        fields = query["opt_fields"].split(",") if "opt_fields" in query else None

        def shape(obj):
            return select(obj, fields) if fields else compact(obj)
        if isinstance(result, list):
            start, page = int(query.get("offset") or 0), 2
            more = start + page < len(result)
            return {"data": [shape(o) for o in result[start:start + page]],
                    "next_page": {"offset": str(start + page)} if more else None}
        return {"data": shape(result) if result else {}}

    def download(self, url):
        attachment = next(a for a in self.attachments if "https://s3.fake/" + a["gid"] == url)
        return attachment["content"]

    def list_workspaces(self, **_):
        return self.workspaces

    def list_projects(self, workspace, query, **_):
        return [self.render_project(p) for p in self.projects if p["workspace"] == workspace
                and not (query.get("archived") == "false" and p["archived"])]

    def list_tags(self, workspace, **_):
        return [t for t in self.tags if t["workspace"] == workspace]

    def create_tag(self, workspace, body, **_):
        return self.add_tag(body["name"], workspace)

    def get_project(self, gid, **_):
        return self.render_project(self.find(self.projects, gid, "project"))

    def project_tasks(self, gid, query, **_):
        self.find(self.projects, gid, "project")
        return [self.render_task(t) for t in self.tasks if gid in t["projects"]
                and not (query.get("completed_since") == "now" and t["completed"])]

    def create_task(self, body, **_):
        parent = self.find(self.tasks, body["parent"], "task") if body.get("parent") else None
        projects = [self.find(self.projects, p, "project") for p in body.get("projects", [])]
        if projects:
            workspace = projects[0]["workspace"]
        elif parent:
            workspace = parent["workspace"]
        else:
            self.fail("workspace: Missing input")
        tags = [self.find(self.tags, t, "tag")["gid"] for t in body.get("tags", [])]
        task = {"gid": self.next_gid(), "resource_type": "task", "name": body["name"], "notes": body.get("notes", ""),
                "completed": False, "parent": parent["gid"] if parent else None,
                "projects": [p["gid"] for p in projects], "tags": tags, "dependencies": [],
                "workspace": workspace}
        self.tasks.append(task)
        return self.render_task(task)

    def get_task(self, gid, **_):
        return self.render_task(self.find(self.tasks, gid, "task"))

    def update_task(self, gid, body, **_):
        task = self.find(self.tasks, gid, "task")
        task.update(body)
        return self.render_task(task)

    def subtasks(self, gid, **_):
        self.find(self.tasks, gid, "task")
        return [self.render_task(t) for t in self.tasks if t["parent"] == gid]

    def add_dependencies(self, gid, body, **_):
        task = self.find(self.tasks, gid, "task")
        for dependency in body["dependencies"]:
            self.find(self.tasks, dependency, "task")
            if dependency not in task["dependencies"]:
                task["dependencies"].append(dependency)
        return {}

    def remove_dependencies(self, gid, body, **_):
        task = self.find(self.tasks, gid, "task")
        task["dependencies"] = [d for d in task["dependencies"] if d not in body["dependencies"]]
        return {}

    def add_tag_to_task(self, gid, body, **_):
        task = self.find(self.tasks, gid, "task")
        self.find(self.tags, body["tag"], "tag")
        if body["tag"] not in task["tags"]:
            task["tags"].append(body["tag"])
        return {}

    def set_parent(self, gid, body, **_):
        self.find(self.tasks, body["parent"], "task")
        self.find(self.tasks, gid, "task")["parent"] = body["parent"]
        return {}

    def list_attachments(self, query, **_):
        return [self.render_attachment(a) for a in self.attachments if a["parent"] == query["parent"]]

    def create_attachment(self, form, **_):
        self.parent_object(form["parent"])
        attachment = {"gid": self.next_gid(), "parent": form["parent"]}
        if form.get("resource_subtype") == "external":
            attachment.update(name=form["name"], host="external", url=form["url"], content=None)
        else:
            filename, content, _ = form["file"]
            attachment.update(name=filename, host="asana", url=None, content=content.decode("utf-8"))
        self.attachments.append(attachment)
        return self.render_attachment(attachment)

    def get_attachment(self, gid, **_):
        return self.render_attachment(self.find(self.attachments, gid, "attachment"))

    def delete_attachment(self, gid, **_):
        self.find(self.attachments, gid, "attachment")
        self.attachments = [a for a in self.attachments if a["gid"] != gid]
        return {}


def run(argv, stdin=""):
    """Runs asana.main(argv) with stdin redirected; returns what it printed to stdout."""
    out = io.StringIO()
    real_stdin = sys.stdin
    sys.stdin = io.StringIO(stdin)
    try:
        with contextlib.redirect_stdout(out):
            asana.main(argv)
    finally:
        sys.stdin = real_stdin
    return out.getvalue()


def run_json(argv, stdin=""):
    return json.loads(run(argv, stdin))


class AsanaTestCase(unittest.TestCase):
    def setUp(self):
        self.fake = FakeAsana()
        self.real_api, self.real_download = asana.api, asana.download
        asana.api, asana.download = self.fake.api, self.fake.download
        self.acme = self.fake.add_workspace("Acme")
        self.cart = self.fake.add_project("Cart", self.acme)
        self.ops = self.fake.add_project("Ops", self.acme)

    def tearDown(self):
        asana.api, asana.download = self.real_api, self.real_download

    # --- helpers ---

    def run_err(self, argv, stdin=""):
        """Runs argv expecting SystemExit; returns the stderr text."""
        err = io.StringIO()
        real_stdin = sys.stdin
        sys.stdin = io.StringIO(stdin)
        try:
            with contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(err):
                with self.assertRaises(SystemExit):
                    asana.main(argv)
        finally:
            sys.stdin = real_stdin
        return err.getvalue()

    def create(self, title, *extra, body="body"):
        return run_json(["create", "--title", title, *extra], stdin=body)["id"]

    def ids(self, command, *scope):
        return [t["id"] for t in run_json([command, *scope])]

    # --- token and transport ---

    def test_missing_token_fails_before_any_request(self):
        asana.api = self.real_api
        saved = os.environ.pop(asana.KEY_ENV, None)
        try:
            self.assertIn("ASANA_ACCESS_TOKEN is not set", self.run_err(["workspaces"]))
        finally:
            if saved is not None:
                os.environ[asana.KEY_ENV] = saved

    def test_multipart_carries_fields_and_file(self):
        data, content_type = asana.multipart({"parent": "12", "file": ("Spec.md", b"# Spec\n", "text/markdown")})
        boundary = content_type.split("boundary=")[1]
        self.assertTrue(data.endswith(("--" + boundary + "--\r\n").encode()))
        self.assertIn(b'name="parent"\r\n\r\n12\r\n', data)
        self.assertIn(b'filename="Spec.md"\r\nContent-Type: text/markdown\r\n\r\n# Spec\n\r\n', data)

    # --- workspaces and projects ---

    def test_workspaces_and_projects(self):
        other = self.fake.add_workspace("Other")
        self.fake.add_project("Old", self.acme, archived=True)
        self.fake.add_project("Web", other)
        self.assertEqual([w["name"] for w in run_json(["workspaces"])], ["Acme", "Other"])
        self.assertEqual([p["name"] for p in run_json(["projects"])], ["Cart", "Ops", "Web"])
        self.assertEqual([p["name"] for p in run_json(["projects", "--workspace", "other"])], ["Web"])
        self.assertIn("unknown workspace: Nope", self.run_err(["projects", "--workspace", "Nope"]))

    def test_project_by_name_gid_or_url(self):
        for ref in ("cart", self.cart["gid"],
                    "https://app.asana.com/0/%s/list" % self.cart["gid"],
                    "https://app.asana.com/1/%s/project/%s/list" % (self.acme["gid"], self.cart["gid"])):
            gid = self.create("T", "--project", ref)
            self.assertEqual(run_json(["get", "--id", gid])["projects"], ["Cart"])

    def test_project_name_ambiguous_across_workspaces(self):
        other = self.fake.add_workspace("Other")
        self.fake.add_project("Cart", other)
        self.assertIn("project name is ambiguous", self.run_err(
            ["create", "--title", "x", "--project", "Cart"], stdin="b"))
        gid = self.create("x", "--project", "Cart", "--workspace", "Other")
        self.assertEqual(run_json(["get", "--id", gid])["workspace"], "Other")

    # --- create ---

    def test_create_in_project(self):
        task = run_json(["create", "--title", "First", "--project", "Cart"], stdin="do it")
        self.assertFalse(task["completed"])
        self.assertIsNone(task["parent"])
        self.assertEqual((task["blockedBy"], task["tags"]), ([], []))
        self.assertIn("/task/" + task["id"], task["url"])
        detail = run_json(["get", "--id", task["id"]])
        self.assertEqual((detail["workspace"], detail["projects"]), ("Acme", ["Cart"]))

    def test_create_errors_create_nothing(self):
        self.assertIn("give --project or --parent", self.run_err(["create", "--title", "x"], stdin="b"))
        self.assertIn("title is empty", self.run_err(
            ["create", "--title", " ", "--project", "Cart"], stdin="b"))
        self.assertIn("task body is empty", self.run_err(
            ["create", "--title", "x", "--project", "Cart"], stdin="  \n"))
        self.assertIn("unknown project: Nope", self.run_err(
            ["create", "--title", "x", "--project", "Nope"], stdin="b"))
        self.assertIn("Unknown object", self.run_err(
            ["create", "--title", "x", "--project", "Cart", "--blocked-by", "999"], stdin="b"))
        self.assertIn("--workspace only narrows --project", self.run_err(
            ["create", "--title", "x", "--parent", "1", "--workspace", "Acme"], stdin="b"))
        self.assertEqual(self.fake.tasks, [])

    def test_subtask_is_in_no_project_unless_given(self):
        parent = self.create("Parent", "--project", "Cart")
        child = run_json(["create", "--title", "Child", "--parent", parent], stdin="b")
        self.assertEqual(child["parent"], parent)
        self.assertEqual(run_json(["get", "--id", child["id"]])["projects"], [])
        other = self.create("Other", "--parent", parent, "--project", "Ops")
        self.assertEqual(run_json(["get", "--id", other])["projects"], ["Ops"])
        self.assertEqual(run_json(["get", "--id", parent])["subtasks"], [child["id"], other])

    def test_create_with_blockers(self):
        a = self.create("A", "--project", "Cart")
        b = self.create("B", "--project", "Cart")
        task = run_json(["create", "--title", "C", "--project", "Cart", "--blocked-by", a,
                         "--blocked-by", b, "--blocked-by", a], stdin="b")
        self.assertEqual(task["blockedBy"], [a, b])
        self.assertEqual(run_json(["get", "--id", a])["blocks"], [task["id"]])

    def test_create_with_tags_reuses_and_creates_tags(self):
        self.fake.add_tag("for-agent", self.acme["gid"])
        task = run_json(["create", "--title", "A", "--project", "Cart", "--tag", "FOR-AGENT",
                         "--tag", "backend"], stdin="b")
        self.assertEqual(task["tags"], ["for-agent", "backend"])
        self.create("B", "--project", "Cart", "--tag", "backend")
        self.assertEqual([t["name"] for t in self.fake.tags], ["for-agent", "backend"])

    # --- read ---

    def test_body_and_task_urls(self):
        gid = self.create("A", "--project", "Cart", body="line one\nline two")
        for ref in (gid, "https://app.asana.com/0/%s/%s/f" % (self.cart["gid"], gid),
                    "https://app.asana.com/1/%s/project/%s/task/%s?focus=true"
                    % (self.acme["gid"], self.cart["gid"], gid)):
            self.assertEqual(run(["body", "--id", ref]), "line one\nline two\n")
        self.assertIn("not a task id or task URL", self.run_err(["body", "--id", "ENG-1"]))
        self.assertIn("Unknown object", self.run_err(["get", "--id", "404"]))

    # --- listings ---

    def test_list_scopes_and_pagination(self):
        outside = self.create("Outside", "--project", "Ops")
        store = self.create("Store", "--project", "Cart")
        a = self.create("A", "--parent", store)
        b = self.create("B", "--parent", store)
        c = self.create("C", "--parent", store, "--project", "Cart", "--blocked-by", outside)
        self.assertEqual(self.ids("list", "--parent", store), [a, b, c])
        self.assertEqual(self.ids("list", "--project", "Cart"), [store, c])
        self.assertEqual(self.ids("list", "--project", "Ops"), [outside])
        self.assertEqual(run_json(["list", "--parent", a]), [])

    def test_list_requires_one_scope(self):
        self.assertIn("give either --parent", self.run_err(["list", "--parent", "1", "--project", "Cart"]))
        self.assertIn("give --parent or --project", self.run_err(["list"]))

    def test_pending_and_unblocked(self):
        outside = self.create("Outside", "--project", "Ops")
        store = self.create("Store", "--project", "Cart")
        a = self.create("A", "--parent", store, "--tag", "for-agent")
        b = self.create("B", "--parent", store)
        c = self.create("C", "--parent", store, "--blocked-by", a, "--blocked-by", b, "--tag", "for-agent")
        d = self.create("D", "--parent", store, "--blocked-by", outside, "--tag", "for-agent")
        e = self.create("E", "--parent", store)
        self.fake.set_completed(e)
        scope = ("--parent", store)
        self.assertEqual(self.ids("list-pending", *scope), [a, b, c, d])
        self.assertEqual(self.ids("list-unblocked", *scope), [a, b])
        self.assertEqual(self.ids("list-unblocked", *scope, "--tag", "for-agent"), [a])

        run(["complete", "--id", a])
        self.assertEqual(self.ids("list-unblocked", *scope, "--tag", "for-agent"), [])
        run(["complete", "--id", b])
        self.assertEqual(self.ids("list-unblocked", *scope, "--tag", "for-agent"), [c])

        # a blocker outside the scope still blocks
        run(["complete", "--id", outside])
        self.assertEqual(self.ids("list-unblocked", *scope), [c, d])

    def test_project_listings_skip_completed_tasks_at_the_source(self):
        a = self.create("A", "--project", "Cart")
        b = self.create("B", "--project", "Cart", "--blocked-by", a)
        self.fake.set_completed(a)
        self.assertEqual(self.ids("list", "--project", "Cart"), [a, b])
        self.assertEqual(self.ids("list-pending", "--project", "Cart"), [b])
        self.assertEqual(self.ids("list-unblocked", "--project", "Cart"), [b])

    def test_tag_filter_requires_every_tag(self):
        a = self.create("A", "--project", "Cart", "--tag", "for-agent", "--tag", "backend")
        self.create("B", "--project", "Cart", "--tag", "for-agent")
        self.assertEqual(self.ids("list", "--project", "Cart", "--tag", "for-agent", "--tag", "Backend"), [a])

    # --- complete ---

    def test_complete(self):
        a = self.create("A", "--project", "Cart")
        self.assertTrue(run_json(["complete", "--id", a])["completed"])
        self.fake.calls.clear()
        self.assertTrue(run_json(["complete", "--id", a])["completed"])
        self.assertNotIn(("PUT", "/tasks/" + a), self.fake.calls)

    def test_complete_refuses_while_blocked(self):
        a = self.create("A", "--project", "Cart")
        b = self.create("B", "--project", "Cart", "--blocked-by", a)
        self.assertIn("cannot complete %s: blocker %s is not completed" % (b, a),
                      self.run_err(["complete", "--id", b]))

    # --- block / unblock ---

    def test_block_adds_and_skips_existing(self):
        a = self.create("A", "--project", "Cart")
        b = self.create("B", "--project", "Cart")
        c = self.create("C", "--project", "Cart", "--blocked-by", a)
        task = run_json(["block", "--id", c, "--blocked-by", a, "--blocked-by", b])
        self.assertEqual(task["blockedBy"], [a, b])

    def test_block_rejects_self_and_cycles(self):
        a = self.create("A", "--project", "Cart")
        b = self.create("B", "--project", "Cart", "--blocked-by", a)
        c = self.create("C", "--project", "Cart", "--blocked-by", b)
        self.assertIn("cannot block itself", self.run_err(["block", "--id", a, "--blocked-by", a]))
        self.assertIn("would create a cycle", self.run_err(["block", "--id", a, "--blocked-by", c]))
        self.assertIn("no --blocked-by given", self.run_err(["block", "--id", a]))
        self.assertEqual(run_json(["get", "--id", a])["blockedBy"], [])

    def test_unblock(self):
        a = self.create("A", "--project", "Cart")
        b = self.create("B", "--project", "Cart")
        c = self.create("C", "--project", "Cart", "--blocked-by", a, "--blocked-by", b)
        self.assertEqual(run_json(["unblock", "--id", c, "--blocked-by", a])["blockedBy"], [b])
        self.assertIn("%s does not block %s" % (a, c),
                      self.run_err(["unblock", "--id", c, "--blocked-by", a]))

    # --- tag / set-parent ---

    def test_tag_adds_never_removes(self):
        a = self.create("A", "--project", "Cart", "--tag", "one")
        self.fake.calls.clear()
        self.assertEqual(run_json(["tag", "--id", a, "--tag", "two", "--tag", "ONE"])["tags"], ["one", "two"])
        self.assertEqual([c for c in self.fake.calls if c[1].endswith("/addTag")], [("POST", "/tasks/" + a + "/addTag")])
        self.assertIn("no --tag given", self.run_err(["tag", "--id", a]))

    def test_set_parent(self):
        a = self.create("A", "--project", "Cart")
        b = self.create("B", "--project", "Cart")
        self.assertEqual(run_json(["set-parent", "--id", b, "--parent", a])["parent"], a)
        self.assertIn("cannot be its own parent", self.run_err(["set-parent", "--id", a, "--parent", a]))

    def test_update_replaces_body_and_optionally_title(self):
        b = self.create("B", "--project", "Cart")
        a = self.create("A", "--project", "Cart", "--tag", "for-agent", "--blocked-by", b, body="old")
        task = run_json(["update", "--id", a], stdin="new")
        self.assertEqual((task["title"], task["tags"], task["blockedBy"]), ("A", ["for-agent"], [b]))
        self.assertEqual(run(["body", "--id", a]), "new\n")
        self.assertEqual(run_json(["update", "--id", a, "--title", " A v2 "], stdin="newer")["title"], "A v2")
        self.assertEqual(run(["body", "--id", a]), "newer\n")
        self.assertIn("task body is empty", self.run_err(["update", "--id", a], stdin=" \n"))
        self.assertIn("title is empty", self.run_err(["update", "--id", a, "--title", " "], stdin="x"))
        self.assertEqual(run(["body", "--id", a]), "newer\n")

    # --- documents ---

    def test_doc_create_on_task_shows_on_task(self):
        a = self.create("A", "--project", "Cart")
        doc = run_json(["doc-create", "--title", "Spec.md", "--task", a], stdin="# Spec\n")
        self.assertEqual((doc["title"], doc["task"], doc["project"]), ("Spec", a, None))
        self.assertEqual(self.fake.attachments[0]["name"], "Spec.md")
        self.assertEqual(run_json(["get", "--id", a])["documents"],
                         [{"id": doc["id"], "title": "Spec", "url": doc["url"]}])
        self.assertEqual(run(["doc-content", "--doc", doc["url"]]), "# Spec\n")

    def test_doc_create_in_project(self):
        doc = run_json(["doc-create", "--title", "Plan", "--project", "cart", "--workspace", "Acme"], stdin="x")
        self.assertEqual((doc["task"], doc["project"]), (None, "Cart"))

    def test_doc_create_argument_errors(self):
        a = self.create("A", "--project", "Cart")
        self.assertIn("exactly one of --task or --project", self.run_err(["doc-create", "--title", "x"], stdin="c"))
        self.assertIn("exactly one of --task or --project", self.run_err(
            ["doc-create", "--title", "x", "--task", a, "--project", "Cart"], stdin="c"))
        self.assertIn("--workspace only narrows --project", self.run_err(
            ["doc-create", "--title", "x", "--task", a, "--workspace", "Acme"], stdin="c"))
        self.assertIn("document content is empty", self.run_err(
            ["doc-create", "--title", "x", "--task", a], stdin=""))
        self.assertIn("title is empty", self.run_err(["doc-create", "--title", ".md", "--task", a], stdin="c"))
        self.assertEqual(self.fake.attachments, [])

    def test_doc_get_by_id_or_url(self):
        doc = run_json(["doc-create", "--title", "Plan", "--project", "Cart"], stdin="x")
        for ref in (doc["id"], doc["url"]):
            self.assertEqual(run_json(["doc-get", "--doc", ref])["id"], doc["id"])
        self.assertIn("not a document id or document URL", self.run_err(["doc-get", "--doc", "nope"]))
        a = self.create("A", "--project", "Cart")
        self.fake.add_link(a, "Figma", "https://figma.com/x")
        link = self.fake.attachments[-1]["gid"]
        self.assertIn("not a document, but a link", self.run_err(["doc-content", "--doc", link]))

    def test_doc_update_replaces_the_file(self):
        doc = run_json(["doc-create", "--title", "Plan", "--project", "Cart"], stdin="old")
        updated = run_json(["doc-update", "--doc", doc["url"], "--title", "Plan v2"], stdin="new")
        self.assertNotEqual(updated["id"], doc["id"])
        self.assertEqual((updated["title"], updated["project"]), ("Plan v2", "Cart"))
        self.assertEqual(updated["replaced"], {"id": doc["id"], "url": doc["url"]})
        self.assertEqual(run(["doc-content", "--doc", updated["id"]]), "new\n")
        self.assertIn("Unknown object", self.run_err(["doc-get", "--doc", doc["id"]]))
        again = run_json(["doc-update", "--doc", updated["id"]], stdin="newer")
        self.assertEqual(again["title"], "Plan v2")
        self.assertEqual(len(self.fake.attachments), 1)

    def test_doc_link_to_several_tasks_once_each(self):
        a = self.create("A", "--project", "Cart")
        b = self.create("B", "--project", "Ops")
        doc = run_json(["doc-create", "--title", "Plan", "--project", "Cart"], stdin="x")
        task = run_json(["doc-link", "--doc", doc["id"], "--task", a])
        self.assertEqual(task["links"], [{"title": "Plan", "url": doc["url"]}])
        self.assertEqual(task["documents"], [])
        run(["doc-link", "--doc", doc["url"], "--task", a])
        run(["doc-link", "--doc", doc["id"], "--task", b])
        self.assertEqual(len(self.fake.attachments), 3)
        self.assertEqual(run_json(["get", "--id", b])["links"][0]["url"], doc["url"])


if __name__ == "__main__":
    unittest.main()
