# agent-hooks

One Go binary that enforces agent guardrails as `PreToolUse` and `PostToolUse`
hooks in Claude Code, Codex, and the Antigravity CLI (agy). It parses each
shell command with [mvdan.cc/sh](https://pkg.go.dev/mvdan.cc/sh/v3/syntax), so it
sees every command inside pipelines, subshells, `$(...)`, here-documents,
`bash -c` strings, `eval`, and wrappers like `sudo`, `env`, and `timeout`.

## Hooks

`hooks.yaml` switches each hook on or off (unlisted means off). It is compiled
into the binary, so run `make install` after editing it.

| Hook | Blocks | Override |
| --- | --- | --- |
| `block-agy` | Running agy / antigravity / acg in any mode, directly, through a launcher, or by piping agy text into a shell | none |
| `draft-prs` | `gh pr create` without `--draft`, or with `--web` | none |
| `pr-assignee` | `gh pr create` without `--assignee` / `-a` | none |
| `no-admin-merge` | `gh pr merge --admin`, and `PUT …/pulls/N/merge` through `gh api` | none |
| `merge-head-match` | `gh pr merge` without `--match-head-commit <sha>` | none |
| `no-production` | Pushing to, merging into, force-updating, resetting, or deleting a `production` branch, including `gh pr merge` into production | `production` |
| `commit-format` | `git commit` messages that don't match `type(PROJECT): description`, or whose type differs from the branch's first commit | none |
| `no-secrets-commit` | `git add` / `git commit -a` staging `.env*`, keys, certificates, `credentials*.json`, or `settings.local.json` | `secrets` |
| `destructive-git` | `git push --force`, `rm -rf` of `/`, `~`, or a repository root, and `git clean`, `reset --hard`, or `checkout -- .` that would discard uncommitted work | `destructive` |
| `rg-replace-flag` | Nothing. After `rg -r`, it adds a note that `-r` means `--replace` (Claude Code and Codex only) | n/a |

To let one command through, put the override at its front:
`AGENT_HOOKS_ALLOW=production git push origin production`. Separate several
names with commas. Each use is logged as a JSON line to
`~/.local/state/agent-hooks/overrides.log` (or under `$XDG_STATE_HOME`).

## Install

```sh
make install     # build to ~/.local/bin/agent-hooks and wire it into every harness
make uninstall   # remove the entries again
```

`agent-hooks install` edits these files in place. It keeps their other settings
and writes a `.bak-<timestamp>` copy before any change. It also removes the old
`block-agy.sh` entries.

- **Claude Code** (`~/.claude/settings.json`): `PreToolUse` and `PostToolUse`, with matcher `Bash`.
- **Codex** (`~/.codex/hooks.json`): the same entries. Codex runs a hook only after
  you trust it, and the trust is tied to the entry: after the first install, or
  after an install that changes the entry, open Codex, run `/hooks`, and approve both entries.
- **agy** (`~/.gemini/config/hooks.json`): a hook named `agent-hooks` with a
  `PreToolUse` entry matching `run_command|manage_task`. agy's `PostToolUse` can't add
  context, so it gets no post hook.

## Protocols

- **Claude Code and Codex**: the payload is `{tool_name, tool_input, cwd, …}` on stdin.
  - A block exits 2 with the reason on stderr.
  - Advice is `{"hookSpecificOutput": {"hookEventName": "PostToolUse", "additionalContext": …}}` on stdout.
  - Shell commands arrive in `tool_input.command` (a string or an argv array), `.cmd`, or `.chars`.
  - Codex 0.171 sends the shell as `Bash`, including commands nested in code mode. It never sends `write_stdin` to hooks, so input typed into a running terminal isn't checked there.
- **agy** (`pre --agy`): the payload is camelCase, `{toolCall: {name: "run_command", args: {CommandLine, Cwd}}, …}`.
  - The answer is always JSON on stdout: `{"decision": "allow"}` or `{"decision": "deny", "reason": …}`.
- A malformed or unknown payload is allowed.
- An internal fault, including an invalid embedded `hooks.yaml`, exits 1. Claude Code and Codex report that without blocking, and the pi wrapper treats it as a failure.
- Each `git` / `gh` lookup times out after 1–2 s, and a failed lookup allows the call.

## Performance

The hooks run on every tool call, so:

- Calls decided from the payload alone never start a subprocess.
- `git` and `gh` run only for git/gh commands that need repository state. Their results are cached within the call and skipped once a hook has blocked.

`make check` prints these numbers. Measured end to end on the built binary, process start included, with all ten hooks on:

| Calls | p50 | p99 |
| --- | --- | --- |
| Decided from the payload alone | 1.1 ms | 1.7 ms |
| Needing a git lookup | 5.0 ms | 10.7 ms |
| Old `block-agy.sh` (bash + jq, one hook) | 10.3 ms | 12.3 ms |

## Development

```sh
make test    # go test ./...
make check   # gofmt, go vet, tests, binary latency, benchmarks
```

Tests run each hook through `hooks.Run` with real payloads. Git hooks run
against temporary repositories cloned from a local bare remote that has a
`production` branch, and `gh` lookups use a fake `gh` on `PATH`.
