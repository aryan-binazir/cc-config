from __future__ import annotations

import json
import subprocess
import unittest
from pathlib import Path

HOOK = Path(__file__).resolve().parents[1] / "hooks/block-agy.sh"


def run(payload: dict) -> int:
    return subprocess.run(
        ["bash", str(HOOK)], input=json.dumps(payload), text=True, capture_output=True, check=False
    ).returncode


def bash(command: str) -> dict:
    return {"tool_name": "Bash", "tool_input": {"command": command}}


class BlockAgyTest(unittest.TestCase):
    def test_blocks_every_agent_launch_of_agy(self) -> None:
        for payload in (
            bash('agy -p "review this"'),
            bash('timeout 45m agy -p "$PROMPT" --model gemini-4-neon-high'),
            bash("agy --model gemini-4-neon-high --print hi"),
            bash("cd /x && antigravity --prompt=hi"),
            bash("/home/ar/.local/share/mise/installs/antigravity-cli/latest/antigravity -p hi"),
            bash('"$HOME/.local/share/mise/installs/antigravity-cli/latest/agy" models'),
            bash("acg -p hi"),
            bash("echo x | agy --input-format stream-json --output-format stream-json"),
            bash("agy acp"),
            bash("agy"),
            bash("agy --version"),
            bash("agy models"),
            bash("agy mcp list"),
            bash("FOO=1 agy update"),
            bash("out=$(agy -p hi)"),
            bash("tmux new-session -d -s gemini agy"),
            bash("tmux send-keys -t gemini 'agy' Enter"),
            bash("bash -lc 'agy --help'"),
            bash("nohup agy > /dev/null 2>&1 &"),
            bash("mise exec antigravity-cli -- agy -p hi"),
            bash("ls\nagy -p hi"),
            {"tool_name": "exec_command", "tool_input": {"cmd": "agy -p hi"}},
            {"tool_name": "shell", "tool_input": {"command": ["bash", "-lc", "agy -p hi"]}},
            {"tool_name": "write_stdin", "tool_input": {"chars": "agy\n"}},
            {"tool_name": "bash", "tool_input": {"command": "agy acp"}},
        ):
            with self.subTest(payload=payload):
                self.assertEqual(run(payload), 2)

    def test_allows_commands_that_only_mention_agy(self) -> None:
        for payload in (
            bash("rg -n agy ~/.gemini/antigravity-cli/log"),
            bash("echo agy -p hi"),
            bash("ps aux | grep agy"),
            bash("which agy && command -v agy && type agy"),
            bash('git commit -m "Block agy calls from agents"'),
            bash("mise upgrade antigravity-cli"),
            bash("strategy -p foo"),
            bash("ls ~/.gemini/antigravity-cli/log"),
            {"tool_name": "Write", "tool_input": {"file_path": "/x.md", "content": "agy -p"}},
            {"tool_name": "apply_patch", "tool_input": {"command": "agy -p"}},
            {"tool_name": "write", "tool_input": {"path": "/x.md", "content": "agy -p"}},
            {},
        ):
            with self.subTest(payload=payload):
                self.assertEqual(run(payload), 0)


if __name__ == "__main__":
    unittest.main()
