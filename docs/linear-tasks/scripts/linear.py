#!/usr/bin/env python3
"""Manage Linear issues as tasks, and Linear documents attached to them.

Every read and every write of Linear goes through this script. Standard library only.
"""

import argparse
import json
import os
import re
import sys
import urllib.error
import urllib.request

API_URL = "https://api.linear.app/graphql"
KEY_ENV = "LINEAR_API_KEY"
DONE = "completed"                    # the workflow state type of a completed issue
CLOSED = ("completed", "canceled")    # state types that are no longer pending
UUID_RE = re.compile(r"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")


# --- queries ---------------------------------------------------------------

ISSUE_LIST_FIELDS = """
  id identifier title url
  state { name type }
  team { id key }
  parent { identifier }
  labels(first: 50) { nodes { name } }
  inverseRelations(first: 50) { nodes { id type issue { id identifier state { type } } } }
"""

ISSUE_FIELDS = ISSUE_LIST_FIELDS + """
  description
  project { id name }
  children(first: 100) { nodes { identifier } }
  relations(first: 50) { nodes { id type relatedIssue { identifier } } }
  documents(first: 50) { nodes { id title url } }
  attachments(first: 50) { nodes { title url } }
"""

DOCUMENT_FIELDS = "id title url slugId content issue { identifier } project { id name }"

ISSUE_GET_Q = "query IssueGet($id: String!) { issue(id: $id) {" + ISSUE_FIELDS + "} }"

ISSUE_LIST_Q = """query IssueList($filter: IssueFilter, $after: String) {
  issues(filter: $filter, first: 50, after: $after) {
    nodes {""" + ISSUE_LIST_FIELDS + """}
    pageInfo { hasNextPage endCursor }
  }
}"""

TEAM_LIST_Q = "query TeamList { teams(first: 250) { nodes { id key name } } }"

TEAM_BY_KEY_Q = """query TeamByKey($key: String!) {
  teams(filter: { key: { eqIgnoreCase: $key } }) { nodes { id key name } }
}"""

TEAM_STATES_Q = """query TeamStates($id: String!) {
  team(id: $id) { states(first: 100) { nodes { id name type position } } }
}"""

PROJECT_FIND_Q = """query ProjectFind($filter: ProjectFilter) {
  projects(filter: $filter, first: 250) { nodes { id name url } }
}"""

LABEL_FIND_Q = """query LabelFind($filter: IssueLabelFilter) {
  issueLabels(filter: $filter, first: 50) { nodes { id name team { id } } }
}"""

DOCUMENT_GET_Q = "query DocumentGet($id: String!) { document(id: $id) { " + DOCUMENT_FIELDS + " } }"

ISSUE_CREATE_M = """mutation IssueCreate($input: IssueCreateInput!) {
  issueCreate(input: $input) { issue { id identifier } }
}"""

ISSUE_UPDATE_M = """mutation IssueUpdate($id: String!, $input: IssueUpdateInput!) {
  issueUpdate(id: $id, input: $input) { issue { id identifier } }
}"""

RELATION_CREATE_M = """mutation RelationCreate($input: IssueRelationCreateInput!) {
  issueRelationCreate(input: $input) { issueRelation { id } }
}"""

RELATION_DELETE_M = """mutation RelationDelete($id: String!) {
  issueRelationDelete(id: $id) { success }
}"""

LABEL_CREATE_M = """mutation LabelCreate($input: IssueLabelCreateInput!) {
  issueLabelCreate(input: $input) { issueLabel { id name team { id } } }
}"""

DOCUMENT_CREATE_M = """mutation DocumentCreate($input: DocumentCreateInput!) {
  documentCreate(input: $input) { document { """ + DOCUMENT_FIELDS + """ } }
}"""

DOCUMENT_UPDATE_M = """mutation DocumentUpdate($id: String!, $input: DocumentUpdateInput!) {
  documentUpdate(id: $id, input: $input) { document { """ + DOCUMENT_FIELDS + """ } }
}"""

ATTACHMENT_LINK_M = """mutation AttachmentLink($issueId: String!, $url: String!, $title: String) {
  attachmentLinkURL(issueId: $issueId, url: $url, title: $title) { attachment { id url } }
}"""


# --- helpers ---------------------------------------------------------------


def die(message):
    sys.stderr.write(message + "\n")
    sys.exit(1)


def send(query, variables=None):
    """Runs one GraphQL request against Linear; returns its data or dies."""
    key = os.environ.get(KEY_ENV, "").strip()
    if key == "":
        die(KEY_ENV + " is not set")
    data = json.dumps({"query": query, "variables": variables or {}}).encode("utf-8")
    request = urllib.request.Request(
        API_URL, data=data, headers={"Content-Type": "application/json", "Authorization": key})
    try:
        with urllib.request.urlopen(request, timeout=60) as response:
            payload = json.loads(response.read().decode("utf-8"))
    except urllib.error.HTTPError as err:
        raw = err.read().decode("utf-8", "replace")
        try:
            payload = json.loads(raw)
        except ValueError:
            die("Linear API HTTP " + str(err.code) + ": " + raw[:300])
    except urllib.error.URLError as err:
        die("cannot reach the Linear API: " + str(err.reason))
    if payload.get("errors"):
        messages = []
        for error in payload["errors"]:
            extra = (error.get("extensions") or {}).get("userPresentableMessage")
            messages.append(error.get("message", "unknown error") + (" (" + extra + ")" if extra else ""))
        die("Linear API error: " + "; ".join(messages))
    return payload["data"]


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


def nodes(connection):
    return connection["nodes"] if connection else []


def identifier_sort_key(identifier):
    prefix, _, number = identifier.rpartition("-")
    return (prefix, int(number)) if number.isdigit() else (prefix, 0)


def document_ref(ref):
    """Accepts a document id, slug id or URL; returns what document(id:) takes."""
    ref = ref.strip()
    if ref.startswith("http"):
        last = ref.rstrip("/").split("?")[0].split("#")[0].rsplit("/", 1)[-1]
        return last.rsplit("-", 1)[-1]
    return ref


def fetch_issue(ref):
    return send(ISSUE_GET_Q, {"id": ref.strip()})["issue"]


def fetch_issues(issue_filter):
    out, after = [], None
    while True:
        connection = send(ISSUE_LIST_Q, {"filter": issue_filter, "after": after})["issues"]
        out.extend(connection["nodes"])
        if not connection["pageInfo"]["hasNextPage"]:
            break
        after = connection["pageInfo"]["endCursor"]
    out.sort(key=lambda i: identifier_sort_key(i["identifier"]))
    return out


def blockers(issue):
    return [r["issue"] for r in nodes(issue["inverseRelations"]) if r["type"] == "blocks"]


def is_completed(issue):
    return issue["state"]["type"] == DONE


def is_pending(issue):
    return issue["state"]["type"] not in CLOSED


def is_unblocked(issue):
    return is_pending(issue) and all(b["state"]["type"] == DONE for b in blockers(issue))


def tag_names(issue):
    return [label["name"] for label in nodes(issue["labels"])]


def has_tags(issue, tags):
    names = {n.lower() for n in tag_names(issue)}
    return all(tag.lower() in names for tag in tags)


def public(issue):
    return {
        "id": issue["identifier"],
        "title": issue["title"],
        "state": issue["state"]["name"],
        "completed": is_completed(issue),
        "parent": issue["parent"]["identifier"] if issue["parent"] else None,
        "blockedBy": [b["identifier"] for b in blockers(issue)],
        "tags": tag_names(issue),
        "url": issue["url"],
    }


def detail(issue):
    out = public(issue)
    out["team"] = issue["team"]["key"]
    out["project"] = issue["project"]["name"] if issue["project"] else None
    out["subtasks"] = sorted((c["identifier"] for c in nodes(issue["children"])),
                             key=identifier_sort_key)
    out["blocks"] = [r["relatedIssue"]["identifier"]
                     for r in nodes(issue["relations"]) if r["type"] == "blocks"]
    out["documents"] = [{"id": d["id"], "title": d["title"], "url": d["url"]}
                        for d in nodes(issue["documents"])]
    out["links"] = [{"title": a["title"], "url": a["url"]} for a in nodes(issue["attachments"])]
    return out


def public_document(document):
    return {
        "id": document["id"],
        "title": document["title"],
        "url": document["url"],
        "issue": document["issue"]["identifier"] if document["issue"] else None,
        "project": document["project"]["name"] if document["project"] else None,
    }


def resolve_team(key):
    teams = send(TEAM_BY_KEY_Q, {"key": key.strip()})["teams"]["nodes"]
    if not teams:
        die("unknown team key: " + key)
    return teams[0]


def resolve_project(ref, team_id=None):
    ref = ref.strip()
    project_filter = {"id": {"eq": ref}} if UUID_RE.match(ref) else {"name": {"eqIgnoreCase": ref}}
    if team_id:
        project_filter["accessibleTeams"] = {"some": {"id": {"eq": team_id}}}
    found = send(PROJECT_FIND_Q, {"filter": project_filter})["projects"]["nodes"]
    if not found:
        die("unknown project: " + ref)
    if len(found) > 1:
        die("project name is ambiguous, give its id or --team: " + ref)
    return found[0]


def resolve_labels(names, team_id):
    """Returns label ids for names, creating a team label for any name that does not exist."""
    ids = []
    for name in names:
        label_filter = {
            "name": {"eqIgnoreCase": name},
            "isGroup": {"eq": False},
            "or": [{"team": {"id": {"eq": team_id}}}, {"team": {"null": True}}],
        }
        found = send(LABEL_FIND_Q, {"filter": label_filter})["issueLabels"]["nodes"]
        found.sort(key=lambda label: label["team"] is None)   # a team label wins over a workspace one
        if found:
            ids.append(found[0]["id"])
        else:
            created = send(LABEL_CREATE_M, {"input": {"name": name, "teamId": team_id}})
            ids.append(created["issueLabelCreate"]["issueLabel"]["id"])
    return ids


def done_state(team_id):
    states = send(TEAM_STATES_Q, {"id": team_id})["team"]["states"]["nodes"]
    done = sorted((s for s in states if s["type"] == DONE), key=lambda s: s["position"])
    if not done:
        die("team has no workflow state of type completed")
    return done[0]


def would_cycle(issue_id, blocker_ids):
    """True when issue_id already blocks, directly or transitively, any of blocker_ids."""
    stack, seen = list(blocker_ids), set()
    while stack:
        current = stack.pop()
        if current == issue_id:
            return True
        if current in seen:
            continue
        seen.add(current)
        stack.extend(b["id"] for b in blockers(fetch_issue(current)))
    return False


def update_issue(issue, fields):
    send(ISSUE_UPDATE_M, {"id": issue["id"], "input": fields})


def print_json(value):
    print(json.dumps(value, indent=2))


# --- task commands ---------------------------------------------------------


def cmd_teams(args):
    teams = send(TEAM_LIST_Q)["teams"]["nodes"]
    print_json([{"key": t["key"], "name": t["name"]} for t in teams])
    return 0


def cmd_projects(args):
    team = resolve_team(args.team)
    project_filter = {"accessibleTeams": {"some": {"id": {"eq": team["id"]}}}}
    found = send(PROJECT_FIND_Q, {"filter": project_filter})["projects"]["nodes"]
    print_json([{"id": p["id"], "name": p["name"], "url": p["url"]} for p in found])
    return 0


def cmd_create(args):
    if not args.team and not args.parent:
        die("give --team or --parent")
    title = args.title.strip()
    if title == "":
        die("title is empty")
    body = read_stdin("task body")
    parent = fetch_issue(args.parent) if args.parent else None
    team = resolve_team(args.team) if args.team else parent["team"]
    if args.project:
        project_id = resolve_project(args.project, team["id"])["id"]
    elif parent and parent["project"] and parent["team"]["id"] == team["id"]:
        project_id = parent["project"]["id"]
    else:
        project_id = None
    blocker_issues = [fetch_issue(b) for b in normalize_strings(args.blocked_by)]
    fields = {"teamId": team["id"], "title": title, "description": body}
    if parent:
        fields["parentId"] = parent["id"]
    if project_id:
        fields["projectId"] = project_id
    tags = normalize_strings(args.tag)
    if tags:
        fields["labelIds"] = resolve_labels(tags, team["id"])
    created = send(ISSUE_CREATE_M, {"input": fields})["issueCreate"]["issue"]
    for blocker in blocker_issues:
        send(RELATION_CREATE_M, {"input": {
            "issueId": blocker["id"], "relatedIssueId": created["id"], "type": "blocks"}})
    print_json(public(fetch_issue(created["id"])))
    return 0


def cmd_get(args):
    print_json(detail(fetch_issue(args.id)))
    return 0


def cmd_body(args):
    body = fetch_issue(args.id)["description"] or ""
    sys.stdout.write(body if body.endswith("\n") else body + "\n")
    return 0


def cmd_list(args):
    if args.parent and (args.team or args.project):
        die("give either --parent, or --team with an optional --project")
    if args.parent:
        issue_filter = {"parent": {"id": {"eq": fetch_issue(args.parent)["id"]}}}
    elif args.team:
        issue_filter = {"team": {"id": {"eq": resolve_team(args.team)["id"]}}}
        if args.project:
            project = resolve_project(args.project, issue_filter["team"]["id"]["eq"])
            issue_filter["project"] = {"id": {"eq": project["id"]}}
    else:
        die("give --parent or --team")
    selected = fetch_issues(issue_filter)
    if args.filter == "pending":
        selected = [i for i in selected if is_pending(i)]
    elif args.filter == "unblocked":
        selected = [i for i in selected if is_unblocked(i)]
    # Blocking above is judged against every blocker, inside the scope or not; the tag filter only
    # narrows which of the resulting tasks are shown.
    tags = normalize_strings(args.tag)
    if tags:
        selected = [i for i in selected if has_tags(i, tags)]
    print_json([public(i) for i in selected])
    return 0


def cmd_complete(args):
    issue = fetch_issue(args.id)
    if is_completed(issue):
        print_json(public(issue))
        return 0
    for blocker in blockers(issue):
        if blocker["state"]["type"] != DONE:
            die("cannot complete " + issue["identifier"] + ": blocker " + blocker["identifier"]
                + " is not completed")
    update_issue(issue, {"stateId": done_state(issue["team"]["id"])["id"]})
    print_json(public(fetch_issue(issue["id"])))
    return 0


def cmd_block(args):
    issue = fetch_issue(args.id)
    refs = normalize_strings(args.blocked_by)
    if not refs:
        die("no --blocked-by given")
    existing = {b["id"] for b in blockers(issue)}
    add = []
    for ref in refs:
        blocker = fetch_issue(ref)
        if blocker["id"] == issue["id"]:
            die("a task cannot block itself: " + ref)
        if blocker["id"] not in existing and blocker["id"] not in {a["id"] for a in add}:
            add.append(blocker)
    if add and would_cycle(issue["id"], [a["id"] for a in add]):
        die("blocking " + issue["identifier"] + " by " + ", ".join(a["identifier"] for a in add)
            + " would create a cycle")
    for blocker in add:
        send(RELATION_CREATE_M, {"input": {
            "issueId": blocker["id"], "relatedIssueId": issue["id"], "type": "blocks"}})
    print_json(public(fetch_issue(issue["id"])))
    return 0


def cmd_unblock(args):
    issue = fetch_issue(args.id)
    refs = normalize_strings(args.blocked_by)
    if not refs:
        die("no --blocked-by given")
    relations = [r for r in nodes(issue["inverseRelations"]) if r["type"] == "blocks"]
    remove = []
    for ref in refs:
        blocker = fetch_issue(ref)
        match = [r for r in relations if r["issue"]["id"] == blocker["id"]]
        if not match:
            die(blocker["identifier"] + " does not block " + issue["identifier"])
        remove.extend(match)
    for relation in remove:
        send(RELATION_DELETE_M, {"id": relation["id"]})
    print_json(public(fetch_issue(issue["id"])))
    return 0


def cmd_tag(args):
    issue = fetch_issue(args.id)
    tags = normalize_strings(args.tag)
    if not tags:
        die("no --tag given")
    update_issue(issue, {"addedLabelIds": resolve_labels(tags, issue["team"]["id"])})
    print_json(public(fetch_issue(issue["id"])))
    return 0


def cmd_set_parent(args):
    issue = fetch_issue(args.id)
    parent = fetch_issue(args.parent)
    if parent["id"] == issue["id"]:
        die("a task cannot be its own parent: " + args.parent)
    update_issue(issue, {"parentId": parent["id"]})
    print_json(public(fetch_issue(issue["id"])))
    return 0


def cmd_update(args):
    issue = fetch_issue(args.id)
    fields = {"description": read_stdin("task body")}
    if args.title is not None:
        if args.title.strip() == "":
            die("title is empty")
        fields["title"] = args.title.strip()
    update_issue(issue, fields)
    print_json(public(fetch_issue(issue["id"])))
    return 0


# --- document commands -----------------------------------------------------


def fetch_document(ref):
    return send(DOCUMENT_GET_Q, {"id": document_ref(ref)})["document"]


def cmd_doc_create(args):
    if bool(args.issue) == bool(args.project):
        die("give exactly one of --issue or --project")
    if args.team and not args.project:
        die("--team only narrows --project")
    title = args.title.strip()
    if title == "":
        die("title is empty")
    content = read_stdin("document content")
    fields = {"title": title, "content": content}
    if args.issue:
        fields["issueId"] = fetch_issue(args.issue)["id"]
    else:
        team_id = resolve_team(args.team)["id"] if args.team else None
        fields["projectId"] = resolve_project(args.project, team_id)["id"]
    created = send(DOCUMENT_CREATE_M, {"input": fields})["documentCreate"]["document"]
    print_json(public_document(created))
    return 0


def cmd_doc_get(args):
    print_json(public_document(fetch_document(args.doc)))
    return 0


def cmd_doc_content(args):
    content = fetch_document(args.doc)["content"] or ""
    sys.stdout.write(content if content.endswith("\n") else content + "\n")
    return 0


def cmd_doc_update(args):
    document = fetch_document(args.doc)
    fields = {"content": read_stdin("document content")}
    if args.title is not None:
        if args.title.strip() == "":
            die("title is empty")
        fields["title"] = args.title.strip()
    updated = send(DOCUMENT_UPDATE_M, {"id": document["id"], "input": fields})
    print_json(public_document(updated["documentUpdate"]["document"]))
    return 0


def cmd_doc_link(args):
    document = fetch_document(args.doc)
    issue = fetch_issue(args.issue)
    if document["url"] not in {a["url"] for a in nodes(issue["attachments"])}:
        send(ATTACHMENT_LINK_M, {"issueId": issue["id"], "url": document["url"],
                                 "title": document["title"]})
    print_json(detail(fetch_issue(issue["id"])))
    return 0


# --- wiring ----------------------------------------------------------------


def build_parser():
    scope = argparse.ArgumentParser(add_help=False)
    scope.add_argument("--parent")
    scope.add_argument("--team")
    scope.add_argument("--project")
    scope.add_argument("--tag", action="append")

    parser = argparse.ArgumentParser(
        description="Manage Linear issues as tasks, and Linear documents attached to them.")
    sub = parser.add_subparsers(dest="command")
    sub.required = True

    p = sub.add_parser("teams")
    p.set_defaults(func=cmd_teams)

    p = sub.add_parser("projects")
    p.add_argument("--team", required=True)
    p.set_defaults(func=cmd_projects)

    p = sub.add_parser("create")
    p.add_argument("--title", required=True)
    p.add_argument("--team")
    p.add_argument("--project")
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
    p.add_argument("--issue")
    p.add_argument("--project")
    p.add_argument("--team")
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
    p.add_argument("--issue", required=True)
    p.set_defaults(func=cmd_doc_link)

    return parser


def main(argv=None):
    args = build_parser().parse_args(argv)
    return args.func(args)


if __name__ == "__main__":
    sys.exit(main())
