#!/usr/bin/env python3
"""Tests for tasks.py. Run with: python3 tasks_test.py"""

import contextlib
import io
import json
import os
import shutil
import sys
import tempfile
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import tasks  # noqa: E402


ROOT = None   # the current test's task store root, set in setUp


def run(argv, stdin=""):
    """Runs tasks.main(argv) with stdin redirected; returns what it printed to stdout."""
    out = io.StringIO()
    real_stdin = sys.stdin
    sys.stdin = io.StringIO(stdin)
    try:
        with contextlib.redirect_stdout(out):
            tasks.main(argv)
    finally:
        sys.stdin = real_stdin
    return out.getvalue()


def create(title, body, blockers=(), root=None):
    argv = ["create", "--root", root or ROOT, "--title", title]
    for b in blockers:
        argv += ["--blocked-by", b]
    return json.loads(run(argv, stdin=body))


class TasksTestCase(unittest.TestCase):
    def setUp(self):
        global ROOT
        self.root = ROOT = tempfile.mkdtemp()

    def tearDown(self):
        shutil.rmtree(self.root, ignore_errors=True)

    # --- helpers ---

    def run_err(self, argv, stdin=""):
        """Runs argv expecting SystemExit; returns the stderr text."""
        err = io.StringIO()
        real_stdin = sys.stdin
        sys.stdin = io.StringIO(stdin)
        try:
            with contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(err):
                with self.assertRaises(SystemExit):
                    tasks.main(argv)
        finally:
            sys.stdin = real_stdin
        return err.getvalue()

    def listing(self, command="list", root=None):
        return json.loads(run([command, "--root", root or self.root]))

    # --- create ---

    def test_first_create_in_empty_root(self):
        task = create("First task", "do the thing\n")
        self.assertEqual(task["id"], "001")
        self.assertFalse(task["completed"])
        self.assertEqual(task["blockedBy"], [])
        self.assertTrue(os.path.isfile(os.path.join(self.root, "001-first-task.md")))

    def test_second_create(self):
        create("First task", "a\n")
        second = create("Second task", "b\n")
        self.assertEqual(second["id"], "002")

    def test_create_into_non_existent_root(self):
        root = os.path.join(self.root, "nested", "tasks")
        task = create("First task", "a\n", root=root)
        self.assertTrue(os.path.isdir(root))
        self.assertEqual(task["id"], "001")

    def test_create_with_blocker(self):
        create("First task", "a\n")
        second = create("Second task", "b\n", blockers=["001"])
        self.assertEqual(second["blockedBy"], ["001"])

    def test_create_with_repeated_blocker(self):
        create("First task", "a\n")
        create("Second task", "b\n")
        third = create("Third task", "c\n", blockers=["002", "001", "002"])
        self.assertEqual(third["blockedBy"], ["002", "001"])

    def test_create_with_unknown_blocker(self):
        self.run_err(["create", "--root", self.root, "--title", "T", "--blocked-by", "099"],
                     stdin="a\n")
        self.assertEqual(os.listdir(self.root), [])

    def test_create_with_empty_stdin(self):
        self.run_err(["create", "--root", self.root, "--title", "T"], stdin="   \n")

    def test_create_with_whitespace_only_title(self):
        self.run_err(["create", "--root", self.root, "--title", "   "], stdin="a\n")

    def test_awkward_title_round_trips(self):
        title = 'He said: "hi" [x]'
        create(title, "a\n")
        self.assertEqual(self.listing()[0]["title"], title)

    def test_body_with_dashes_round_trips(self):
        body = "intro line\n---\nafter the dashes\n"
        create("Dashed", body)
        self.assertEqual(run(["body", "--root", self.root, "--id", "001"]), body)

    # --- listing ---

    def test_list_on_empty_existing_root(self):
        self.assertEqual(self.listing(), [])

    def test_list_pending_excludes_completed(self):
        create("First task", "a\n")
        create("Second task", "b\n")
        run(["complete", "--root", self.root, "--id", "001"])
        self.assertEqual([t["id"] for t in self.listing("list-pending")], ["002"])

    def test_list_unblocked(self):
        create("First task", "a\n")
        create("Second task", "b\n", blockers=["001"])
        self.assertEqual([t["id"] for t in self.listing("list-unblocked")], ["001"])
        run(["complete", "--root", self.root, "--id", "001"])
        self.assertEqual([t["id"] for t in self.listing("list-unblocked")], ["002"])

    def test_list_output_shape(self):
        create("First task", "a\n")
        self.assertEqual(set(self.listing()[0]), {"id", "title", "blockedBy", "completed"})

    # --- complete ---

    def test_complete_with_pending_blocker(self):
        create("First task", "a\n")
        create("Second task", "b\n", blockers=["001"])
        self.run_err(["complete", "--root", self.root, "--id", "002"])

    def test_complete_twice(self):
        create("First task", "a\n")
        first = run(["complete", "--root", self.root, "--id", "001"])
        second = run(["complete", "--root", self.root, "--id", "001"])
        self.assertEqual(first, second)
        self.assertTrue(json.loads(second)["completed"])

    # --- block ---

    def test_block_adds_edge_without_duplicates(self):
        create("First task", "a\n")
        create("Second task", "b\n")
        create("Third task", "c\n", blockers=["001"])
        out = json.loads(run(
            ["block", "--root", self.root, "--id", "003", "--blocked-by", "001",
             "--blocked-by", "002"]))
        self.assertEqual(out["blockedBy"], ["001", "002"])

    def test_block_self(self):
        create("First task", "a\n")
        self.run_err(["block", "--root", self.root, "--id", "001", "--blocked-by", "001"])

    def test_block_closing_a_cycle(self):
        create("First task", "a\n")
        create("Second task", "b\n")
        run(["block", "--root", self.root, "--id", "001", "--blocked-by", "002"])
        path = os.path.join(self.root, "002-second-task.md")
        with open(path, encoding="utf-8") as handle:
            before = handle.read()
        self.run_err(["block", "--root", self.root, "--id", "002", "--blocked-by", "001"])
        with open(path, encoding="utf-8") as handle:
            self.assertEqual(handle.read(), before)

    def test_block_with_unknown_blocker(self):
        create("First task", "a\n")
        self.run_err(["block", "--root", self.root, "--id", "001", "--blocked-by", "099"])

    # --- body ---

    def test_body_is_byte_for_byte(self):
        body = "line one\n\n  indented line\nlast\n"
        create("Bodied", body)
        self.assertEqual(run(["body", "--root", self.root, "--id", "001"]), body)

    # --- exists ---

    def test_exists_on_missing_root(self):
        missing = os.path.join(self.root, "nope")
        out = run(["exists", "--root", missing])
        self.assertEqual(json.loads(out), {"exists": False})

    def test_exists_on_real_root(self):
        self.assertEqual(json.loads(run(["exists", "--root", self.root])), {"exists": True})

    # --- missing root ---

    def test_missing_root_is_fatal_for_every_command(self):
        missing = os.path.join(self.root, "nope")
        self.run_err(["list", "--root", missing])
        self.run_err(["list-pending", "--root", missing])
        self.run_err(["list-unblocked", "--root", missing])
        self.run_err(["complete", "--root", missing, "--id", "001"])
        self.run_err(["block", "--root", missing, "--id", "001", "--blocked-by", "002"])
        self.run_err(["body", "--root", missing, "--id", "001"])

    # --- corrupt stores ---

    def test_readme_without_frontmatter(self):
        create("First task", "a\n")
        readme = os.path.join(self.root, "README.md")
        with open(readme, "w", encoding="utf-8") as handle:
            handle.write("just some notes\n")
        for argv in (["list", "--root", self.root],
                     ["list-pending", "--root", self.root],
                     ["list-unblocked", "--root", self.root],
                     ["complete", "--root", self.root, "--id", "001"],
                     ["block", "--root", self.root, "--id", "001", "--blocked-by", "001"],
                     ["body", "--root", self.root, "--id", "001"]):
            self.assertIn(readme, self.run_err(argv))

    def test_stray_non_markdown_file(self):
        create("First task", "a\n")
        with open(os.path.join(self.root, "notes.txt"), "w", encoding="utf-8") as handle:
            handle.write("stray\n")
        self.run_err(["list", "--root", self.root])

    def test_duplicate_ids(self):
        create("First task", "a\n")
        original = os.path.join(self.root, "001-first-task.md")
        copy = os.path.join(self.root, "001-copy.md")
        shutil.copyfile(original, copy)
        err = self.run_err(["list", "--root", self.root])
        self.assertIn(original, err)
        self.assertIn(copy, err)

    def test_blocked_by_deleted_task(self):
        create("First task", "a\n")
        create("Second task", "b\n", blockers=["001"])
        os.remove(os.path.join(self.root, "001-first-task.md"))
        self.run_err(["list", "--root", self.root])


if __name__ == "__main__":
    unittest.main()
