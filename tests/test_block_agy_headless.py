from __future__ import annotations

import json
import subprocess
import unittest
from pathlib import Path

HOOK = Path(__file__).resolve().parents[1] / "hooks/block-agy-headless.sh"


def run(payload: dict) -> int:
    return subprocess.run(
        ["bash", str(HOOK)], input=json.dumps(payload), text=True, capture_output=True, check=False
    ).returncode


def bash(command: str) -> dict:
    return {"tool_name": "Bash", "tool_input": {"command": command}}


class BlockAgyHeadlessTest(unittest.TestCase):
    def test_blocks_non_interactive_agy(self) -> None:
        for payload in (
            bash('agy -p "review this"'),
            bash('timeout 45m agy -p "$PROMPT" --model gemini-4-neon-high'),
            bash("agy --model gemini-4-neon-high --print hi"),
            bash("cd /x && antigravity --prompt=hi"),
            bash("/home/ar/.local/share/mise/installs/antigravity-cli/latest/antigravity -p hi"),
            bash("acg -p hi"),
            bash("echo x | agy --input-format stream-json --output-format stream-json"),
            bash("agy acp"),
            bash("agy -print hi"),
            bash("ls\nagy -p hi"),
            {"tool_name": "exec_command", "tool_input": {"cmd": "agy -p hi"}},
            {"tool_name": "shell", "tool_input": {"command": ["bash", "-lc", "agy -p hi"]}},
            {"tool_name": "write_stdin", "tool_input": {"chars": "agy -p hi\n"}},
        ):
            with self.subTest(payload=payload):
                self.assertEqual(run(payload), 2)

    def test_allows_everything_else(self) -> None:
        for payload in (
            bash("agy --help"),
            bash("agy --version"),
            bash("agy models"),
            bash("agy mcp list"),
            bash("strategy -p foo"),
            bash("git log -p && echo agy"),
            bash("ls ~/.gemini/antigravity-cli/log"),
            {"tool_name": "Write", "tool_input": {"file_path": "/x.md", "content": "never run agy -p"}},
            {"tool_name": "apply_patch", "tool_input": {"command": "*** agy -p"}},
            {},
        ):
            with self.subTest(payload=payload):
                self.assertEqual(run(payload), 0)


if __name__ == "__main__":
    unittest.main()
