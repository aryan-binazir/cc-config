// Blocks agent-run commands that launch the Antigravity CLI (agy / antigravity / acg)
// in any mode. The check lives in agent-hooks (cc-config/agent-hooks), shared with
// Claude Code, Codex and agy.
import { spawnSync } from "node:child_process";
import os from "node:os";
import path from "node:path";

const bin = path.join(os.homedir(), ".local/bin/agent-hooks");

export default function (pi) {
  pi.on("tool_call", async (event, ctx) => {
    const input = event.input ?? {};
    if (input.command === undefined && input.cmd === undefined && input.chars === undefined) return;

    const run = spawnSync(bin, ["pre", "--only=block-agy"], {
      input: JSON.stringify({ tool_name: event.toolName, tool_input: input, cwd: ctx?.cwd }),
      encoding: "utf8",
      timeout: 10_000,
    });
    if (run.status === 2) return { block: true, reason: run.stderr.trim() };
    if (run.status !== 0 && /\b(agy|antigravity|acg)\b/.test(JSON.stringify(input))) {
      return { block: true, reason: `block-agy check could not run (${run.error?.message ?? `exit ${run.status}`}); not running anything that mentions agy.` };
    }
  });
}
