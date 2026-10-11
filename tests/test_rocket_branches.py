from __future__ import annotations

import json
import os
import shlex
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ROCKET = Path(__file__).resolve().parents[1] / "skills/personal_dev/rocket"


class RocketBranchTest(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        root = Path(self.temp.name)
        self.repo = root / "repo"
        remote = root / "origin.git"
        self.env = os.environ | {"GIT_CONFIG_NOSYSTEM": "1", "GIT_CONFIG_GLOBAL": os.devnull}
        self.run_command("git", "init", "--bare", "--initial-branch=main", str(remote))
        self.run_command("git", "init", "--initial-branch=main", str(self.repo))
        (self.repo / ".gitignore").write_text("_scratch/\n")
        self.git("add", ".gitignore")
        self.git("-c", "user.name=Rocket Test", "-c", "user.email=rocket@example.invalid",
                 "commit", "-m", "Initial fixture")
        self.git("remote", "add", "origin", str(remote))
        self.git("push", "origin", "main")
        fake_bin = root / "bin"
        fake_bin.mkdir()
        gh = fake_bin / "gh"
        gh.write_text("#!/bin/sh\nexit 0\n")
        gh.chmod(0o755)
        self.env["PATH"] = str(fake_bin) + os.pathsep + self.env.get("PATH", "")

    def run_command(self, *args: str, cwd: Path | None = None) -> str:
        return subprocess.run(args, cwd=cwd, env=self.env, text=True,
                              capture_output=True, check=True, timeout=15).stdout

    def git(self, *args: str) -> str:
        return self.run_command("git", *args, cwd=self.repo).strip()

    def ensure(self, *args: str) -> dict:
        result = json.loads(self.run_command(
            sys.executable, str(ROCKET / "scripts/ensure_branch.py"),
            "--repo", str(self.repo), "--ticket-key", "BBA-359", *args,
        ))
        self.assertTrue(result["ok"], result)
        return result

    def test_default_worktree_uses_amb_prefix(self) -> None:
        result = self.ensure()
        self.assertEqual(result["branch"], "amb/BBA-359")
        self.assertEqual(result["checkout_mode"], "worktree")
        self.assertEqual(self.run_command("git", "branch", "--show-current",
                                         cwd=Path(result["checkout_path"])).strip(), "amb/BBA-359")
        self.assertEqual(self.git("branch", "--show-current"), "main")

    def test_default_branch_mode_uses_amb_prefix(self) -> None:
        result = self.ensure("--checkout-mode", "branch")
        self.assertEqual(result["branch"], "amb/BBA-359")
        self.assertEqual(self.git("branch", "--show-current"), "amb/BBA-359")

    def test_explicit_prefix_is_preserved(self) -> None:
        result = self.ensure("--prefix", "custom")
        self.assertEqual(result["branch"], "custom/BBA-359")

    def test_explicit_no_ticket_branch_is_preserved(self) -> None:
        result = self.ensure("--ticket-key", "NO-TICKET-TEST-PREFIX",
                             "--branch-name", "amb/test-prefix")
        self.assertEqual(result["branch"], "amb/test-prefix")
        self.assertNotIn("NO-TICKET", result["branch"])
        skill = (ROCKET / "SKILL.md").read_text()
        self.assertIn("`amb/<task-slug>`", skill)
        self.assertNotIn("aryan-binazir/", skill)

    def test_repo_facts_suggests_the_same_default_branch(self) -> None:
        facts = json.loads(self.run_command(
            sys.executable, str(ROCKET / "scripts/repo_facts.py"),
            "--repo", str(self.repo), "--input", "BBA-359",
        ))
        self.assertTrue(facts["ok"], facts)
        self.assertEqual(facts["context"]["suggested_branch"], "amb/BBA-359")
        command = shlex.split(facts["context"]["branch_setup_command"])
        result = json.loads(self.run_command(sys.executable, *command[3:]))
        self.assertTrue(result["ok"], result)
        self.assertEqual(result["branch"], facts["context"]["suggested_branch"])


if __name__ == "__main__":
    unittest.main()
