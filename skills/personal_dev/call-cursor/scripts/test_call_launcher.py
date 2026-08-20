from __future__ import annotations

import os
import subprocess
import tempfile
import unittest
from pathlib import Path


LAUNCHER = Path(__file__).with_name("call.sh")


class CallLauncherTest(unittest.TestCase):
    def setUp(self) -> None:
        self.temp_dir = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp_dir.cleanup)
        self.bin_dir = Path(self.temp_dir.name) / "bin"
        self.bin_dir.mkdir()
        self.capture_path = Path(self.temp_dir.name) / "capture"
        self._write_fake("uv")
        self._write_fake("systemd-run")

    def _write_fake(self, name: str) -> None:
        path = self.bin_dir / name
        path.write_text(
            "#!/usr/bin/env bash\n"
            "printf '%s\\n' \"$0\" \"$@\" > \"$CALL_CURSOR_TEST_CAPTURE\"\n",
            encoding="utf-8",
        )
        path.chmod(0o755)

    def _run(self, *, in_t3: bool) -> list[str]:
        env = {
            **os.environ,
            "CALL_CURSOR_TEST_CAPTURE": str(self.capture_path),
            "PATH": f"{self.bin_dir}:/usr/bin",
        }
        if in_t3:
            env["T3_MCP_BEARER_TOKEN"] = "present"
        else:
            env.pop("T3_MCP_BEARER_TOKEN", None)

        subprocess.run(
            ["bash", str(LAUNCHER), "prompt"],
            check=True,
            env=env,
        )
        return self.capture_path.read_text(encoding="utf-8").splitlines()

    def test_t3_detaches_entire_uv_call(self) -> None:
        invocation = self._run(in_t3=True)

        self.assertEqual(invocation[0], str(self.bin_dir / "systemd-run"))
        self.assertIn("--user", invocation)
        self.assertIn("--pipe", invocation)
        self.assertIn("--wait", invocation)
        self.assertIn("--collect", invocation)
        self.assertIn("--quiet", invocation)
        self.assertIn("--service-type=exec", invocation)
        self.assertIn(str(self.bin_dir / "uv"), invocation)
        self.assertIn(str(LAUNCHER.with_name("call.py")), invocation)
        self.assertEqual(invocation[-1], "prompt")

    def test_non_t3_runs_uv_directly(self) -> None:
        invocation = self._run(in_t3=False)

        self.assertEqual(invocation[0], str(self.bin_dir / "uv"))
        self.assertEqual(invocation[1:3], ["run", "--script"])
        self.assertEqual(invocation[3], str(LAUNCHER.with_name("call.py")))
        self.assertEqual(invocation[4], "prompt")


if __name__ == "__main__":
    unittest.main()
