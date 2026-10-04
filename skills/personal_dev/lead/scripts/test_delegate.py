from __future__ import annotations

import json
import os
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

SCRIPTS = Path(__file__).parent


class DelegateCompletionTest(unittest.TestCase):
    def setUp(self) -> None:
        self.temp_dir = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp_dir.cleanup)
        self.root = Path(self.temp_dir.name)
        scripts = self.root / "lead" / "scripts"
        scripts.mkdir(parents=True)
        for name in ("delegate.py", "resolve_config.py"):
            shutil.copy2(SCRIPTS / name, scripts / name)
        self.delegate = scripts / "delegate.py"
        (scripts.parent / "lead.example.yaml").write_text(
            "workers:\n"
            "  implement: {runner: codex}\n"
            "  explore: {runner: codex, kind: explore}\n",
            encoding="utf-8",
        )
        self.cwd = self.root / "repo"
        self.cwd.mkdir()
        self.bin_dir = self.root / "bin"
        self.bin_dir.mkdir()
        runner = self.bin_dir / "codex"
        runner.write_text(
            f"#!{sys.executable}\n"
            "import json, os, re, sys, time\n"
            "from pathlib import Path\n"
            "case = json.loads(os.environ['DELEGATE_TEST_CASE'])\n"
            "Path(os.environ['DELEGATE_TEST_ARGS']).write_text(json.dumps(sys.argv[1:]))\n"
            "if case['file'] is not None:\n"
            "    if '-o' in sys.argv:\n"
            "        path = Path(sys.argv[sys.argv.index('-o') + 1])\n"
            "    else:\n"
            "        path = Path(re.search(r'Progress file \\(required\\): (\\S+)', sys.argv[-1])[1])\n"
            "    path.write_text(case['file'])\n"
            "print(case['stdout'], flush=True)\n"
            "time.sleep(case['sleep'])\n"
            "sys.exit(case['exit'])\n",
            encoding="utf-8",
        )
        runner.chmod(0o755)

    def run_delegate(
        self, *, summary: str | None = None, stdout: str = "",
        kind: str = "implement", exit_code: int = 0, timeout: bool = False,
    ) -> tuple[subprocess.CompletedProcess[str], dict]:
        env = {
            **os.environ,
            "PATH": f"{self.bin_dir}{os.pathsep}{os.environ['PATH']}",
            "DELEGATE_TEST_ARGS": str(self.root / "args.json"),
            "DELEGATE_TEST_CASE": json.dumps({
                "file": summary, "stdout": stdout, "exit": exit_code,
                "sleep": 10 if timeout else 0,
            }),
        }
        env.pop("DELEGATE_WORKER", None)
        proc = subprocess.run(
            [sys.executable, str(self.delegate), "--worker", kind,
             "--cwd", str(self.cwd), "--prompt", "Do the assigned work.",
             "--timeout-ms", "100" if timeout else "5000"],
            env=env, capture_output=True, text=True, timeout=15, check=False,
        )
        self.assertIn(proc.returncode, (0, 1), proc.stderr)
        return proc, json.loads(proc.stdout)

    def test_empty_success_is_not_completed(self) -> None:
        proc, result = self.run_delegate()
        self.assertEqual(proc.returncode, 0)
        self.assertTrue(result["ok"])
        self.assertEqual(result["exit_code"], 0)
        self.assertFalse(result["completed"])

    def test_progress_success_is_not_completed(self) -> None:
        _, result = self.run_delegate(summary="## SUMMARY\nTests remain.\nStatus: incomplete\n")
        self.assertTrue(result["ok"])
        self.assertFalse(result["completed"])

    def test_complete_summary_is_completed(self) -> None:
        _, result = self.run_delegate(summary="## SUMMARY\nVerified.\nStatus: complete\n")
        self.assertTrue(result["completed"])
        self.assertEqual(result["summary_source"], "file")

    def test_missing_or_malformed_status_is_not_completed(self) -> None:
        for summary in (
            "## SUMMARY\nAll done.",
            "## SUMMARY\nStatus: completed",
            "## SUMMARY\nStatus: COMPLETE",
            "## SUMMARY\nStatus: complete because tests passed",
            "Status: complete",
            "## SUMMARY\nStatus: complete\nStill checking.",
        ):
            with self.subTest(summary=summary):
                _, result = self.run_delegate(summary=summary)
                self.assertFalse(result["completed"])

    def test_last_summary_controls_completion(self) -> None:
        _, result = self.run_delegate(
            summary="## SUMMARY\nStatus: complete\n## SUMMARY\nStatus: incomplete\n",
        )
        self.assertFalse(result["completed"])

    def test_status_is_parsed_before_display_truncation(self) -> None:
        _, result = self.run_delegate(summary="## SUMMARY\n" + "x" * 1600 + "\nStatus: complete\n")
        self.assertTrue(result["completed"])
        self.assertEqual(len(result["summary"]), 1500)
        self.assertNotIn("Status:", result["summary"])

    def test_nonzero_exit_overrides_complete_status(self) -> None:
        proc, result = self.run_delegate(summary="## SUMMARY\nStatus: complete\n", exit_code=2)
        self.assertEqual(proc.returncode, 1)
        self.assertFalse(result["ok"])
        self.assertEqual(result["exit_code"], 2)
        self.assertFalse(result["completed"])

    def test_timeout_overrides_complete_status(self) -> None:
        proc, result = self.run_delegate(summary="## SUMMARY\nStatus: complete\n", timeout=True)
        self.assertEqual(proc.returncode, 1)
        self.assertTrue(result["timed_out"])
        self.assertIsNone(result["exit_code"])
        self.assertFalse(result["completed"])

    def test_stdout_fallback_can_complete(self) -> None:
        _, result = self.run_delegate(stdout="## SUMMARY\nVerified.\nStatus: complete\n")
        self.assertTrue(result["completed"])
        self.assertEqual(result["summary_source"], "stdout")

    def test_progress_file_takes_precedence_over_stdout(self) -> None:
        _, result = self.run_delegate(
            summary="## SUMMARY\nStatus: incomplete\n",
            stdout="## SUMMARY\nStatus: complete\n",
        )
        self.assertFalse(result["completed"])

    def test_explorer_completion_preserves_read_only_and_findings(self) -> None:
        findings = "Evidence at module.py:10.\n## SUMMARY\nAnswer verified.\nStatus: complete\n"
        for use_stdout in (False, True):
            with self.subTest(use_stdout=use_stdout):
                _, result = self.run_delegate(
                    kind="explore", summary=None if use_stdout else findings,
                    stdout=findings if use_stdout else "",
                )
                self.assertTrue(result["completed"])
                self.assertTrue(result["read_only"])
                self.assertEqual(Path(result["summary_file"]).read_text(), findings)
                args = json.loads((self.root / "args.json").read_text())
                self.assertEqual(args[args.index("--sandbox") + 1], "read-only")
                self.assertEqual(args[args.index("--ask-for-approval") + 1], "never")

    def test_explorer_requires_explicit_completion(self) -> None:
        for findings in ("", "Evidence found.", "## SUMMARY\nStatus: incomplete\n"):
            with self.subTest(findings=findings):
                proc, result = self.run_delegate(kind="explore", stdout=findings)
                self.assertFalse(result["completed"])
                self.assertEqual(result["ok"], bool(findings))
                self.assertEqual(proc.returncode, 0 if findings else 1)


if __name__ == "__main__":
    unittest.main()
