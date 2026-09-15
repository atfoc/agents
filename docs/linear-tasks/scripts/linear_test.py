#!/usr/bin/env python3
"""Tests for linear.py against an in-memory fake of the Linear API. Run with: python3 linear_test.py"""

import contextlib
import io
import json
import os
import re
import sys
import unittest
import uuid

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import linear  # noqa: E402


class FakeLinear:
    """Answers the operations linear.py sends, by operation name, from in-memory state."""

    def __init__(self):
        self.teams, self.states, self.projects, self.labels = [], [], [], []
        self.issues, self.relations, self.documents, self.attachments = [], [], [], []
        self.calls = []

    # --- seeding ---

    def add_team(self, key):
        team = {"id": str(uuid.uuid4()), "key": key, "name": key + " team", "counter": 0}
        self.teams.append(team)
        for position, (name, kind) in enumerate([("Todo", "unstarted"), ("In Progress", "started"),
                                                 ("Done", "completed"), ("Shipped", "completed"),
                                                 ("Canceled", "canceled")]):
            self.states.append({"id": str(uuid.uuid4()), "name": name, "type": kind,
                                "position": position, "teamId": team["id"]})
        return team

    def add_project(self, name, team):
        project = {"id": str(uuid.uuid4()), "name": name, "url": "https://linear.app/p/" + name,
                   "teamIds": [team["id"]]}
        self.projects.append(project)
        return project

    def add_label(self, name, team=None):
        label = {"id": str(uuid.uuid4()), "name": name, "teamId": team["id"] if team else None,
                 "isGroup": False}
        self.labels.append(label)
        return label

    def set_state(self, identifier, state_name):
        issue = self.issue_by_ref(identifier)
        issue["stateId"] = next(s["id"] for s in self.states
                                if s["teamId"] == issue["teamId"] and s["name"] == state_name)

    # --- lookups ---

    def fail(self, message):
        linear.die("Linear API error: " + message)

    def issue_by_ref(self, ref):
        for issue in self.issues:
            if ref in (issue["id"], issue["identifier"]) or ref.upper() == issue["identifier"]:
                return issue
        self.fail("Entity not found: Issue")

    def team(self, team_id):
        return next(t for t in self.teams if t["id"] == team_id)

    def state(self, state_id):
        return next(s for s in self.states if s["id"] == state_id)

    def render_state(self, issue):
        s = self.state(issue["stateId"])
        return {"name": s["name"], "type": s["type"]}

    def render_issue(self, issue):
        team = self.team(issue["teamId"])
        project = next((p for p in self.projects if p["id"] == issue["projectId"]), None)
        parent = next((i for i in self.issues if i["id"] == issue["parentId"]), None)
        return {
            "id": issue["id"], "identifier": issue["identifier"], "title": issue["title"],
            "url": "https://linear.app/i/" + issue["identifier"],
            "description": issue["description"],
            "state": self.render_state(issue),
            "team": {"id": team["id"], "key": team["key"]},
            "project": {"id": project["id"], "name": project["name"]} if project else None,
            "parent": {"identifier": parent["identifier"]} if parent else None,
            "labels": {"nodes": [{"name": l["name"]} for l in self.labels
                                 if l["id"] in issue["labelIds"]]},
            "children": {"nodes": [{"identifier": i["identifier"]} for i in self.issues
                                   if i["parentId"] == issue["id"]]},
            "inverseRelations": {"nodes": [
                {"id": r["id"], "type": r["type"], "issue": self.issue_ref(r["issueId"])}
                for r in self.relations if r["relatedIssueId"] == issue["id"]]},
            "relations": {"nodes": [
                {"id": r["id"], "type": r["type"], "relatedIssue": self.issue_ref(r["relatedIssueId"])}
                for r in self.relations if r["issueId"] == issue["id"]]},
            "documents": {"nodes": [{"id": d["id"], "title": d["title"], "url": d["url"]}
                                    for d in self.documents if d["issueId"] == issue["id"]]},
            "attachments": {"nodes": [{"title": a["title"], "url": a["url"]}
                                      for a in self.attachments if a["issueId"] == issue["id"]]},
        }

    def issue_ref(self, issue_id):
        issue = self.issue_by_ref(issue_id)
        return {"id": issue["id"], "identifier": issue["identifier"],
                "state": {"type": self.state(issue["stateId"])["type"]}}

    def render_document(self, document):
        issue = next((i for i in self.issues if i["id"] == document["issueId"]), None)
        project = next((p for p in self.projects if p["id"] == document["projectId"]), None)
        return {"id": document["id"], "title": document["title"], "url": document["url"],
                "slugId": document["slugId"], "content": document["content"],
                "issue": {"identifier": issue["identifier"]} if issue else None,
                "project": {"id": project["id"], "name": project["name"]} if project else None}

    # --- the API ---

    def send(self, query, variables=None):
        variables = variables or {}
        operation = re.match(r"\s*(?:query|mutation)\s+(\w+)", query).group(1)
        self.calls.append(operation)
        return getattr(self, "op_" + operation)(variables)

    def op_IssueGet(self, v):
        return {"issue": self.render_issue(self.issue_by_ref(v["id"]))}

    def op_IssueList(self, v):
        f = v["filter"]
        selected = self.issues
        if "parent" in f:
            selected = [i for i in selected if i["parentId"] == f["parent"]["id"]["eq"]]
        if "team" in f:
            selected = [i for i in selected if i["teamId"] == f["team"]["id"]["eq"]]
        if "project" in f:
            selected = [i for i in selected if i["projectId"] == f["project"]["id"]["eq"]]
        rendered = [self.render_issue(i) for i in reversed(selected)]
        page, start = 2, int(v["after"] or 0)
        return {"issues": {"nodes": rendered[start:start + page], "pageInfo": {
            "hasNextPage": start + page < len(rendered), "endCursor": str(start + page)}}}

    def op_TeamList(self, v):
        return {"teams": {"nodes": [{"id": t["id"], "key": t["key"], "name": t["name"]}
                                    for t in self.teams]}}

    def op_TeamByKey(self, v):
        return {"teams": {"nodes": [{"id": t["id"], "key": t["key"], "name": t["name"]}
                                    for t in self.teams if t["key"].lower() == v["key"].lower()]}}

    def op_TeamStates(self, v):
        return {"team": {"states": {"nodes": [
            {k: s[k] for k in ("id", "name", "type", "position")}
            for s in reversed(self.states) if s["teamId"] == v["id"]]}}}

    def op_ProjectFind(self, v):
        f = v["filter"]
        selected = self.projects
        if "id" in f:
            selected = [p for p in selected if p["id"] == f["id"]["eq"]]
        if "name" in f:
            selected = [p for p in selected if p["name"].lower() == f["name"]["eqIgnoreCase"].lower()]
        if "accessibleTeams" in f:
            team_id = f["accessibleTeams"]["some"]["id"]["eq"]
            selected = [p for p in selected if team_id in p["teamIds"]]
        return {"projects": {"nodes": [{k: p[k] for k in ("id", "name", "url")} for p in selected]}}

    def op_LabelFind(self, v):
        f = v["filter"]
        team_id = f["or"][0]["team"]["id"]["eq"]
        return {"issueLabels": {"nodes": [
            {"id": l["id"], "name": l["name"], "team": {"id": l["teamId"]} if l["teamId"] else None}
            for l in self.labels
            if l["name"].lower() == f["name"]["eqIgnoreCase"].lower() and not l["isGroup"]
            and l["teamId"] in (team_id, None)]}}

    def op_LabelCreate(self, v):
        label = self.add_label(v["input"]["name"], self.team(v["input"]["teamId"]))
        return {"issueLabelCreate": {"issueLabel": {"id": label["id"], "name": label["name"],
                                                    "team": {"id": label["teamId"]}}}}

    def op_IssueCreate(self, v):
        data = v["input"]
        team = self.team(data["teamId"])
        team["counter"] += 1
        todo = next(s for s in self.states if s["teamId"] == team["id"] and s["type"] == "unstarted")
        issue = {"id": str(uuid.uuid4()), "identifier": team["key"] + "-" + str(team["counter"]),
                 "title": data["title"], "description": data["description"], "teamId": team["id"],
                 "stateId": todo["id"], "projectId": data.get("projectId"),
                 "parentId": data.get("parentId"), "labelIds": data.get("labelIds", [])}
        self.issues.append(issue)
        return {"issueCreate": {"issue": {"id": issue["id"], "identifier": issue["identifier"]}}}

    def op_IssueUpdate(self, v):
        issue = self.issue_by_ref(v["id"])
        data = v["input"]
        if "stateId" in data:
            issue["stateId"] = data["stateId"]
        if "parentId" in data:
            issue["parentId"] = data["parentId"]
        for field in ("title", "description"):
            if field in data:
                issue[field] = data[field]
        for label_id in data.get("addedLabelIds", []):
            if label_id not in issue["labelIds"]:
                issue["labelIds"].append(label_id)
        return {"issueUpdate": {"issue": {"id": issue["id"], "identifier": issue["identifier"]}}}

    def op_RelationCreate(self, v):
        relation = dict(v["input"], id=str(uuid.uuid4()))
        self.relations.append(relation)
        return {"issueRelationCreate": {"issueRelation": {"id": relation["id"]}}}

    def op_RelationDelete(self, v):
        self.relations = [r for r in self.relations if r["id"] != v["id"]]
        return {"issueRelationDelete": {"success": True}}

    def op_DocumentGet(self, v):
        for document in self.documents:
            if v["id"] in (document["id"], document["slugId"]):
                return {"document": self.render_document(document)}
        self.fail("Entity not found: Document")

    def op_DocumentCreate(self, v):
        data = v["input"]
        slug = uuid.uuid4().hex[:12]
        document = {"id": str(uuid.uuid4()), "slugId": slug, "title": data["title"],
                    "content": data["content"], "issueId": data.get("issueId"),
                    "projectId": data.get("projectId"),
                    "url": "https://linear.app/acme/document/" + linear.re.sub(
                        r"[^a-z0-9]+", "-", data["title"].lower()) + "-" + slug}
        self.documents.append(document)
        return {"documentCreate": {"document": self.render_document(document)}}

    def op_DocumentUpdate(self, v):
        document = next(d for d in self.documents if d["id"] == v["id"])
        document.update(v["input"])
        return {"documentUpdate": {"document": self.render_document(document)}}

    def op_AttachmentLink(self, v):
        attachment = {"id": str(uuid.uuid4()), "issueId": v["issueId"], "url": v["url"],
                      "title": v["title"]}
        self.attachments.append(attachment)
        return {"attachmentLinkURL": {"attachment": {"id": attachment["id"], "url": v["url"]}}}


def run(argv, stdin=""):
    """Runs linear.main(argv) with stdin redirected; returns what it printed to stdout."""
    out = io.StringIO()
    real_stdin = sys.stdin
    sys.stdin = io.StringIO(stdin)
    try:
        with contextlib.redirect_stdout(out):
            linear.main(argv)
    finally:
        sys.stdin = real_stdin
    return out.getvalue()


def run_json(argv, stdin=""):
    return json.loads(run(argv, stdin))


class LinearTestCase(unittest.TestCase):
    def setUp(self):
        self.fake = FakeLinear()
        self.real_send = linear.send
        linear.send = self.fake.send
        self.eng = self.fake.add_team("ENG")
        self.ops = self.fake.add_team("OPS")
        self.cart = self.fake.add_project("Cart", self.eng)

    def tearDown(self):
        linear.send = self.real_send

    # --- helpers ---

    def run_err(self, argv, stdin=""):
        """Runs argv expecting SystemExit; returns the stderr text."""
        err = io.StringIO()
        real_stdin = sys.stdin
        sys.stdin = io.StringIO(stdin)
        try:
            with contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(err):
                with self.assertRaises(SystemExit):
                    linear.main(argv)
        finally:
            sys.stdin = real_stdin
        return err.getvalue()

    def create(self, title, *extra, body="body"):
        return run_json(["create", "--title", title, *extra], stdin=body)

    def ids(self, command, *scope):
        return [t["id"] for t in run_json([command, *scope])]

    # --- api key ---

    def test_missing_api_key_fails_before_any_request(self):
        linear.send = self.real_send
        saved = os.environ.pop(linear.KEY_ENV, None)
        try:
            self.assertIn("LINEAR_API_KEY is not set", self.run_err(["teams"]))
        finally:
            if saved is not None:
                os.environ[linear.KEY_ENV] = saved

    # --- teams and projects ---

    def test_teams_and_projects(self):
        self.assertEqual([t["key"] for t in run_json(["teams"])], ["ENG", "OPS"])
        self.assertEqual([p["name"] for p in run_json(["projects", "--team", "eng"])], ["Cart"])
        self.assertEqual(run_json(["projects", "--team", "OPS"]), [])

    # --- create ---

    def test_create_in_team(self):
        task = self.create("First", "--team", "ENG", body="do it")
        self.assertEqual(task["id"], "ENG-1")
        self.assertEqual(task["state"], "Todo")
        self.assertFalse(task["completed"])
        self.assertIsNone(task["parent"])
        self.assertEqual(task["blockedBy"], [])
        self.assertEqual(task["tags"], [])
        self.assertIsNone(run_json(["get", "--id", "ENG-1"])["project"])

    def test_create_in_project_by_name(self):
        self.create("First", "--team", "ENG", "--project", "cart")
        self.assertEqual(run_json(["get", "--id", "ENG-1"])["project"], "Cart")

    def test_create_unknown_team_or_project(self):
        self.assertIn("unknown team key: NOPE", self.run_err(
            ["create", "--title", "x", "--team", "NOPE"], stdin="b"))
        self.assertIn("unknown project: Nope", self.run_err(
            ["create", "--title", "x", "--team", "ENG", "--project", "Nope"], stdin="b"))
        self.assertIn("unknown project: Cart", self.run_err(
            ["create", "--title", "x", "--team", "OPS", "--project", "Cart"], stdin="b"))

    def test_create_requires_scope_title_and_body(self):
        self.assertIn("give --team or --parent", self.run_err(["create", "--title", "x"], stdin="b"))
        self.assertIn("title is empty", self.run_err(
            ["create", "--title", " ", "--team", "ENG"], stdin="b"))
        self.assertIn("task body is empty", self.run_err(
            ["create", "--title", "x", "--team", "ENG"], stdin="  \n"))
        self.assertEqual(self.fake.issues, [])

    def test_subtask_inherits_team_and_project(self):
        self.create("Parent", "--team", "ENG", "--project", "Cart")
        child = self.create("Child", "--parent", "ENG-1")
        self.assertEqual(child["id"], "ENG-2")
        self.assertEqual(child["parent"], "ENG-1")
        detail = run_json(["get", "--id", "ENG-2"])
        self.assertEqual((detail["team"], detail["project"]), ("ENG", "Cart"))
        self.assertEqual(run_json(["get", "--id", "ENG-1"])["subtasks"], ["ENG-2"])

    def test_subtask_in_another_team(self):
        self.create("Parent", "--team", "ENG", "--project", "Cart")
        child = self.create("Child", "--parent", "ENG-1", "--team", "OPS")
        self.assertEqual((child["id"], child["parent"]), ("OPS-1", "ENG-1"))
        self.assertIsNone(run_json(["get", "--id", "OPS-1"])["project"])

    def test_create_with_blockers(self):
        self.create("A", "--team", "ENG")
        self.create("B", "--team", "ENG")
        task = self.create("C", "--team", "ENG", "--blocked-by", "ENG-1", "--blocked-by", "ENG-2",
                           "--blocked-by", "ENG-1")
        self.assertEqual(sorted(task["blockedBy"]), ["ENG-1", "ENG-2"])
        self.assertEqual(run_json(["get", "--id", "ENG-1"])["blocks"], ["ENG-3"])

    def test_create_with_unknown_blocker_creates_nothing(self):
        self.assertIn("Entity not found", self.run_err(
            ["create", "--title", "x", "--team", "ENG", "--blocked-by", "ENG-9"], stdin="b"))
        self.assertEqual(self.fake.issues, [])

    def test_create_with_tags_reuses_and_creates_labels(self):
        self.fake.add_label("for-agent")          # workspace label
        task = self.create("A", "--team", "ENG", "--tag", "FOR-AGENT", "--tag", "backend")
        self.assertEqual(task["tags"], ["for-agent", "backend"])
        self.assertEqual(len(self.fake.labels), 2)
        self.assertEqual(self.fake.labels[1]["teamId"], self.eng["id"])
        self.create("B", "--team", "ENG", "--tag", "backend")
        self.assertEqual(len(self.fake.labels), 2)

    def test_team_label_wins_over_workspace_label(self):
        self.fake.add_label("x")
        team_label = self.fake.add_label("x", self.eng)
        self.create("A", "--team", "ENG", "--tag", "x")
        self.assertEqual(self.fake.issues[0]["labelIds"], [team_label["id"]])

    # --- read ---

    def test_body_prints_description_only(self):
        self.create("A", "--team", "ENG", body="line one\nline two")
        self.assertEqual(run(["body", "--id", "eng-1"]), "line one\nline two\n")

    def test_get_unknown_issue(self):
        self.assertIn("Entity not found", self.run_err(["get", "--id", "ENG-404"]))

    # --- listings ---

    def test_list_scopes_are_sorted_and_paginated(self):
        self.create("Outside", "--team", "ENG")
        self.create("Store", "--team", "ENG", "--project", "Cart")
        self.create("A", "--parent", "ENG-2")
        self.create("B", "--parent", "ENG-2")
        self.create("C", "--parent", "ENG-2", "--blocked-by", "ENG-1")
        self.assertEqual(self.ids("list", "--parent", "ENG-2"), ["ENG-3", "ENG-4", "ENG-5"])
        self.assertEqual(self.ids("list", "--team", "ENG"), ["ENG-1", "ENG-2", "ENG-3", "ENG-4", "ENG-5"])
        self.assertEqual(self.ids("list", "--team", "ENG", "--project", "Cart"),
                         ["ENG-2", "ENG-3", "ENG-4", "ENG-5"])
        self.assertEqual(self.ids("list", "--team", "OPS"), [])

    def test_list_requires_one_scope(self):
        self.create("A", "--team", "ENG")
        self.assertIn("give either --parent", self.run_err(["list", "--parent", "ENG-1", "--team", "ENG"]))
        self.assertIn("give --parent or --team", self.run_err(["list"]))

    def test_pending_and_unblocked(self):
        self.create("Outside", "--team", "OPS")                     # OPS-1
        self.create("Store", "--team", "ENG")                       # ENG-1
        self.create("A", "--parent", "ENG-1", "--tag", "for-agent")  # ENG-2
        self.create("B", "--parent", "ENG-1")                       # ENG-3
        self.create("C", "--parent", "ENG-1", "--blocked-by", "ENG-2", "--blocked-by", "ENG-3",
                    "--tag", "for-agent")                           # ENG-4
        self.create("D", "--parent", "ENG-1", "--blocked-by", "OPS-1", "--tag", "for-agent")  # ENG-5
        self.create("E", "--parent", "ENG-1")                       # ENG-6
        self.fake.set_state("ENG-6", "Canceled")
        scope = ("--parent", "ENG-1")
        self.assertEqual(self.ids("list-pending", *scope), ["ENG-2", "ENG-3", "ENG-4", "ENG-5"])
        self.assertEqual(self.ids("list-unblocked", *scope), ["ENG-2", "ENG-3"])
        self.assertEqual(self.ids("list-unblocked", *scope, "--tag", "for-agent"), ["ENG-2"])

        run(["complete", "--id", "ENG-2"])
        self.assertEqual(self.ids("list-unblocked", *scope, "--tag", "for-agent"), [])
        run(["complete", "--id", "ENG-3"])
        self.assertEqual(self.ids("list-unblocked", *scope, "--tag", "for-agent"), ["ENG-4"])

        # a blocker outside the scope still blocks; a canceled blocker never unblocks
        run(["complete", "--id", "OPS-1"])
        self.assertEqual(self.ids("list-unblocked", *scope), ["ENG-4", "ENG-5"])
        run(["block", "--id", "ENG-4", "--blocked-by", "ENG-6"])
        self.assertEqual(self.ids("list-unblocked", *scope), ["ENG-5"])

    def test_tag_filter_requires_every_tag(self):
        self.create("A", "--team", "ENG", "--tag", "for-agent", "--tag", "backend")
        self.create("B", "--team", "ENG", "--tag", "for-agent")
        self.assertEqual(self.ids("list", "--team", "ENG", "--tag", "for-agent", "--tag", "Backend"),
                         ["ENG-1"])

    # --- complete ---

    def test_complete_moves_to_first_completed_state(self):
        self.create("A", "--team", "ENG")
        task = run_json(["complete", "--id", "ENG-1"])
        self.assertTrue(task["completed"])
        self.assertEqual(task["state"], "Done")

    def test_complete_is_idempotent(self):
        self.create("A", "--team", "ENG")
        self.fake.set_state("ENG-1", "Shipped")
        self.fake.calls.clear()
        self.assertEqual(run_json(["complete", "--id", "ENG-1"])["state"], "Shipped")
        self.assertNotIn("IssueUpdate", self.fake.calls)

    def test_complete_refuses_while_blocked(self):
        self.create("A", "--team", "ENG")
        self.create("B", "--team", "ENG", "--blocked-by", "ENG-1")
        self.assertIn("cannot complete ENG-2: blocker ENG-1 is not completed",
                      self.run_err(["complete", "--id", "ENG-2"]))

    # --- block / unblock ---

    def test_block_adds_and_skips_existing(self):
        self.create("A", "--team", "ENG")
        self.create("B", "--team", "ENG")
        self.create("C", "--team", "ENG", "--blocked-by", "ENG-1")
        task = run_json(["block", "--id", "ENG-3", "--blocked-by", "ENG-1", "--blocked-by", "ENG-2"])
        self.assertEqual(sorted(task["blockedBy"]), ["ENG-1", "ENG-2"])
        self.assertEqual(len(self.fake.relations), 2)

    def test_block_rejects_self_and_cycles(self):
        self.create("A", "--team", "ENG")
        self.create("B", "--team", "ENG", "--blocked-by", "ENG-1")
        self.create("C", "--team", "ENG", "--blocked-by", "ENG-2")
        self.assertIn("cannot block itself", self.run_err(["block", "--id", "ENG-1", "--blocked-by", "ENG-1"]))
        self.assertIn("would create a cycle", self.run_err(["block", "--id", "ENG-1", "--blocked-by", "ENG-3"]))
        self.assertIn("no --blocked-by given", self.run_err(["block", "--id", "ENG-1"]))
        self.assertEqual(len(self.fake.relations), 2)

    def test_unblock(self):
        self.create("A", "--team", "ENG")
        self.create("B", "--team", "ENG")
        self.create("C", "--team", "ENG", "--blocked-by", "ENG-1", "--blocked-by", "ENG-2")
        task = run_json(["unblock", "--id", "ENG-3", "--blocked-by", "ENG-1"])
        self.assertEqual(task["blockedBy"], ["ENG-2"])
        self.assertIn("ENG-1 does not block ENG-3",
                      self.run_err(["unblock", "--id", "ENG-3", "--blocked-by", "ENG-1"]))

    def test_related_relations_are_not_blockers(self):
        self.create("A", "--team", "ENG")
        self.create("B", "--team", "ENG")
        self.fake.relations.append({"id": "r", "issueId": self.fake.issues[0]["id"],
                                    "relatedIssueId": self.fake.issues[1]["id"], "type": "related"})
        self.assertEqual(run_json(["get", "--id", "ENG-2"])["blockedBy"], [])
        self.assertEqual(self.ids("list-unblocked", "--team", "ENG"), ["ENG-1", "ENG-2"])

    # --- tag / set-parent ---

    def test_tag_adds_never_removes(self):
        self.create("A", "--team", "ENG", "--tag", "one")
        task = run_json(["tag", "--id", "ENG-1", "--tag", "two", "--tag", "one"])
        self.assertEqual(task["tags"], ["one", "two"])
        self.assertIn("no --tag given", self.run_err(["tag", "--id", "ENG-1"]))

    def test_set_parent(self):
        self.create("A", "--team", "ENG")
        self.create("B", "--team", "ENG")
        self.assertEqual(run_json(["set-parent", "--id", "ENG-2", "--parent", "ENG-1"])["parent"], "ENG-1")
        self.assertIn("cannot be its own parent",
                      self.run_err(["set-parent", "--id", "ENG-1", "--parent", "ENG-1"]))

    def test_update_replaces_body_and_optionally_title(self):
        self.create("A", "--team", "ENG", "--tag", "for-agent", body="old")
        run(["block", "--id", "ENG-1", "--blocked-by", self.create("B", "--team", "ENG")["id"]])
        task = run_json(["update", "--id", "eng-1"], stdin="new")
        self.assertEqual((task["title"], task["tags"], task["blockedBy"]), ("A", ["for-agent"], ["ENG-2"]))
        self.assertEqual(run(["body", "--id", "ENG-1"]), "new\n")
        self.assertEqual(run_json(["update", "--id", "ENG-1", "--title", " A v2 "], stdin="newer")["title"],
                         "A v2")
        self.assertEqual(run(["body", "--id", "ENG-1"]), "newer\n")
        self.assertIn("task body is empty", self.run_err(["update", "--id", "ENG-1"], stdin=" \n"))
        self.assertIn("title is empty", self.run_err(["update", "--id", "ENG-1", "--title", " "], stdin="x"))
        self.assertEqual(run(["body", "--id", "ENG-1"]), "newer\n")

    # --- documents ---

    def test_doc_create_on_issue_shows_on_issue(self):
        self.create("A", "--team", "ENG")
        doc = run_json(["doc-create", "--title", "Spec", "--issue", "ENG-1"], stdin="# Spec\n")
        self.assertEqual((doc["title"], doc["issue"], doc["project"]), ("Spec", "ENG-1", None))
        self.assertEqual(run_json(["get", "--id", "ENG-1"])["documents"],
                         [{"id": doc["id"], "title": "Spec", "url": doc["url"]}])
        self.assertEqual(run(["doc-content", "--doc", doc["id"]]), "# Spec\n")

    def test_doc_create_in_project(self):
        doc = run_json(["doc-create", "--title", "Plan", "--project", "Cart", "--team", "ENG"], stdin="x")
        self.assertEqual((doc["issue"], doc["project"]), (None, "Cart"))
        doc = run_json(["doc-create", "--title", "Plan 2", "--project", "cart"], stdin="x")
        self.assertEqual(doc["project"], "Cart")

    def test_doc_create_argument_errors(self):
        self.create("A", "--team", "ENG")
        self.assertIn("exactly one of --issue or --project",
                      self.run_err(["doc-create", "--title", "x"], stdin="c"))
        self.assertIn("exactly one of --issue or --project", self.run_err(
            ["doc-create", "--title", "x", "--issue", "ENG-1", "--project", "Cart"], stdin="c"))
        self.assertIn("--team only narrows --project", self.run_err(
            ["doc-create", "--title", "x", "--issue", "ENG-1", "--team", "ENG"], stdin="c"))
        self.assertIn("document content is empty", self.run_err(
            ["doc-create", "--title", "x", "--issue", "ENG-1"], stdin=""))
        self.assertEqual(self.fake.documents, [])

    def test_doc_get_by_id_slug_or_url(self):
        doc = run_json(["doc-create", "--title", "Plan", "--project", "Cart"], stdin="x")
        slug = self.fake.documents[0]["slugId"]
        for ref in (doc["id"], slug, doc["url"], doc["url"] + "?tab=x"):
            self.assertEqual(run_json(["doc-get", "--doc", ref])["id"], doc["id"])
        self.assertIn("Entity not found", self.run_err(["doc-get", "--doc", "nope"]))

    def test_doc_update(self):
        doc = run_json(["doc-create", "--title", "Plan", "--project", "Cart"], stdin="old")
        updated = run_json(["doc-update", "--doc", doc["url"], "--title", "Plan v2"], stdin="new")
        self.assertEqual(updated["title"], "Plan v2")
        self.assertEqual(run(["doc-content", "--doc", doc["id"]]), "new\n")
        run(["doc-update", "--doc", doc["id"]], stdin="newer")
        self.assertEqual(run_json(["doc-get", "--doc", doc["id"]])["title"], "Plan v2")

    def test_doc_link_to_several_issues_once_each(self):
        self.create("A", "--team", "ENG")
        self.create("B", "--team", "OPS")
        doc = run_json(["doc-create", "--title", "Plan", "--project", "Cart"], stdin="x")
        issue = run_json(["doc-link", "--doc", doc["id"], "--issue", "ENG-1"])
        self.assertEqual(issue["links"], [{"title": "Plan", "url": doc["url"]}])
        run(["doc-link", "--doc", doc["url"], "--issue", "ENG-1"])
        run(["doc-link", "--doc", doc["id"], "--issue", "OPS-1"])
        self.assertEqual(len(self.fake.attachments), 2)
        self.assertEqual(run_json(["get", "--id", "OPS-1"])["links"][0]["url"], doc["url"])


if __name__ == "__main__":
    unittest.main()
