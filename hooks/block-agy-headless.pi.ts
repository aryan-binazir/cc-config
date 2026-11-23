// Blocks agent-run commands that drive the Antigravity CLI (agy / antigravity / acg)
// non-interactively. The check lives in the shared Claude Code / Codex hook script.
import { spawnSync } from "node:child_process";
import os from "node:os";
import path from "node:path";

const script = path.join(os.homedir(), ".claude/hooks/block-agy-headless.sh");

export default function (pi) {
  pi.on("tool_call", async (event) => {
    const input = event.input ?? {};
    if (input.command === undefined && input.cmd === undefined && input.chars === undefined) return;

    const run = spawnSync("bash", [script], {
      input: JSON.stringify({ tool_name: event.toolName, tool_input: input }),
      encoding: "utf8",
      timeout: 10_000,
    });
    if (run.status === 2) return { block: true, reason: run.stderr.trim() };
  });
}
