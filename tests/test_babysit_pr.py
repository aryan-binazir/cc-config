from __future__ import annotations

import contextlib
import importlib.machinery
import importlib.util
import io
import json
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

SCRIPT = Path(__file__).resolve().parents[1] / "skills/personal_dev/babysit-pr/scripts/pr-wait"
HEAD = "abc1234def"


def load():
    loader = importlib.machinery.SourceFileLoader("pr_wait", str(SCRIPT))
    module = importlib.util.module_from_spec(importlib.util.spec_from_loader("pr_wait", loader))
    loader.exec_module(module)
    return module


def fake_gh(module, comments=(), threads=()):
    pr = {
        "url": "https://github.com/o/r/pull/7", "state": "OPEN", "headRefOid": HEAD,
        "baseRefName": "main", "mergeStateStatus": "CLEAN", "author": {"login": "me"},
        "baseRef": {"target": {"oid": "base1"}},
        "comments": {"nodes": list(comments)}, "reviews": {"nodes": []},
        "reviewThreads": {"nodes": list(threads)},
    }

    def gh(args, allow=(0,)):
        if args[:2] == ["api", "graphql"]:
            out = json.dumps({"data": {"viewer": {"login": "me"}, "repository": {"pullRequest": pr}}})
        elif args[:2] == ["pr", "checks"]:
            out = json.dumps([{"name": "ci", "bucket": "pass", "link": "l", "description": ""}])
        elif args[0] == "api" and "/compare/" in args[1]:
            out = "0"
        else:
            raise module.GhError("HTTP 404", False)
        return subprocess.CompletedProcess(args, 0, out, "")

    return gh


class PrWaitBotWaitTest(unittest.TestCase):
    def setUp(self) -> None:
        self.module = load()
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.module.STATE_DIR = Path(temp.name)

    def poll_twice(self, bot_wait: int, comments=(), threads=(), now: float = 1000.0) -> tuple[int | None, str]:
        watcher = self.module.Watcher("o/r", 7, "ready", 5, bot_wait)
        out = io.StringIO()
        with patch.object(self.module, "gh", fake_gh(self.module, comments, threads)), \
                patch.object(self.module.time, "time", return_value=now), \
                contextlib.redirect_stdout(out), contextlib.redirect_stderr(io.StringIO()):
            try:
                watcher.poll()
                watcher.baseline = False
                watcher.poll()
            except SystemExit as exit_:
                return exit_.code, out.getvalue()
            watcher.save()
        return None, out.getvalue()

    def test_bot_window_runs_from_first_green_across_restarts(self) -> None:
        self.assertIsNone(self.poll_twice(bot_wait=1200, now=1000.0)[0])
        self.assertIsNone(self.poll_twice(bot_wait=1200, now=2199.0)[0])
        code, out = self.poll_twice(bot_wait=1200, now=2200.0)
        self.assertEqual(code, 0)
        self.assertIn("20m bot window passed", out)

    def test_green_without_bots_waits_within_budget(self) -> None:
        code, out = self.poll_twice(bot_wait=20 * 60)
        self.assertIsNone(code, out)

    def test_green_without_bots_is_ready_after_budget(self) -> None:
        code, out = self.poll_twice(bot_wait=0)
        self.assertEqual(code, 0)
        self.assertIn("ready on abc1234", out)
        self.assertIn("0m bot window passed", out)

    def test_bot_review_on_head_keeps_window_open(self) -> None:
        summary = {"id": "s1", "url": "u", "body": f"Greptile summary for {HEAD[:7]}",
                   "author": {"login": "greptile-apps", "__typename": "Bot"}}
        code, out = self.poll_twice(bot_wait=20 * 60, comments=[summary])
        self.assertIsNone(code, out)

    def test_new_thread_in_window_exits_for_the_agent(self) -> None:
        self.poll_twice(bot_wait=30 * 60)
        thread = {"isResolved": False, "path": "a.go", "line": 3, "comments": {"nodes": [
            {"id": "t1", "url": "u", "body": "nil deref", "author": {"login": "work-bot", "__typename": "Bot"}}]}}
        code, out = self.poll_twice(bot_wait=30 * 60, threads=[thread], now=1600.0)
        self.assertEqual(code, 1)
        self.assertIn("new thread from work-bot on a.go:3", out)


if __name__ == "__main__":
    unittest.main()
