# agent-hooks

One Go binary that enforces agent guardrails as `PreToolUse` and `PostToolUse`
hooks in Claude Code, Codex, and the Antigravity CLI (agy). It parses each
shell command with [mvdan.cc/sh](https://pkg.go.dev/mvdan.cc/sh/v3/syntax), so it
sees every command inside pipelines, subshells, `$(...)`, here-documents,
`bash -c` strings, `eval`, and wrappers like `sudo`, `env`, and `timeout`.

## Hooks

Every hook in the `catalog` in `internal/hooks/run.go` runs. To drop one,
delete its name there and run `make install`.

| Hook | Blocks | Override |
| --- | --- | --- |
| `block-agy` | Running agy / antigravity / acg in any mode: directly, through a wrapper or nested shell, as an argument to `tmux`, `ssh`, and the like, or when agy is named in text that feeds a command the hook cannot read | none |
| `draft-prs` | `gh pr create` without `--draft`, or with `--web` | none |
| `pr-assignee` | `gh pr create` without `--assignee` / `-a` | none |
| `no-admin-merge` | `gh pr merge --admin`, and `PUT …/pulls/N/merge` through `gh api` | none |
| `merge-head-match` | `gh pr merge` without `--match-head-commit <sha>` | none |
| `no-production` | Pushing to, merging into, force-updating, resetting, or deleting a `production` branch, including `gh pr merge` into production | `production` |
| `commit-format` | `git commit` messages, and `--title` on `gh pr create` / `gh pr edit`, that don't match `type(PROJECT): description`, or whose type differs from the branch's first commit | `commit-format` |
| `no-secrets-commit` | `git add` / `git commit -a` staging, or `git commit` recording from the index, `.env*`, keys, certificates, `credentials*.json`, or `settings.local.json` | `secrets` |
| `destructive-git` | `git push --force`; `rm -r` of `/`, `~`, or a repository root, including a bare `*` there; `git clean`, `reset --hard`, `checkout -- .`, `checkout -f`, `switch --discard-changes`, or `worktree remove --force` that would discard uncommitted work or a submodule's repository; a twice-forced `worktree remove` of a locked worktree; `git stash clear` while stashes exist | `destructive` |
| `rg-replace-flag` | Nothing. After `rg -r`, it adds a note that `-r` means `--replace` (Claude Code and Codex only) | n/a |

To let one command through, put the override at its front:
`AGENT_HOOKS_ALLOW=production git push origin production`. Exporting it
earlier in the same tool call also works, as it would in the shell:
`export AGENT_HOOKS_ALLOW=production && gh pr merge 5 …`. An export inside a
subshell, pipeline, or `$(...)` does not carry over. Separate several names
with commas. Each use is logged as a JSON line to
`~/.local/state/agent-hooks/overrides.log` (or under `$XDG_STATE_HOME`).

## What the hooks read

A hook sees the command before it runs and decides from that text plus, for
git and gh commands, the repository as it is at that moment.

- **Git aliases** are followed to the command they stand for, from `-c alias.x=…`
  or `git config`, up to four levels. Shell aliases (`alias.x = !cmd`) are not.
  When the working directory is unknown, only the user's global aliases apply.
- **A command that stops parsing partway** still has every complete statement
  before the error checked, as bash would still run them. If the part that
  could not be read mentions `git`, `gh`, or `rm`, the call is blocked as
  `[agent-hooks: unparsed]`. There is no override: fix the syntax. Input typed
  into a running program (`chars`, agy's `send_input`) is exempt, since it need
  not be shell.
- **A lookup with no answer** blocks instead of allowing. If `git` or `gh` times
  out, or `gh pr view` runs and cannot name the pull request's base branch, a
  hook that has an override blocks with the reason and the override to use. A
  lookup that cannot be attempted allows the call: the working directory is
  unknown (a `cd` into an expansion), or `gh` is not installed.
- **`--pathspec-from-file`** lists are not read. The list stands for the whole
  tree, and for no pathspec at all where git then does something else (a forced
  checkout moves HEAD, a commit records the index). Name the paths on the
  command line to have only those checked.
- **block-agy** counts a mention of agy only where it can reach a command the
  hook cannot read: at or before the end of that command's pipeline, or
  anywhere when the command sits in a function body. So `time grep agy notes.txt`
  and `bash build.sh && rg agy` pass, while `a=agy; $a -p hi` and
  `echo 'agy -p hi' | bash` are blocked. Inside a `bash -c` string or a
  here-document fed to a shell the same rule applies to that text, together
  with the text that can reach the shell itself and the shell's other
  arguments.

Not covered: commands started by an interpreter (`python -c`, `node -e`), by a
wrapper the parser does not unwrap (`uv run`, `npx`, `find -exec`, `xargs -I{}`),
or by a `bash -c` or `env -S` string built from an expansion (only block-agy
reads those).
The hooks guard against mistakes, not against an agent set on evading them.

## Install

```sh
make install     # build to ~/.local/bin/agent-hooks and wire it into every harness
make uninstall   # remove the entries again
```

`agent-hooks install` edits these files in place. It keeps their other settings
and writes a `.bak-<timestamp>` copy before any change. It also points any old
`block-agy.sh` entry under Claude Code's or Codex's `PreToolUse` or
`PostToolUse` at agent-hooks instead.

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
- `pre --only=<hook>` runs that one hook, as the pi wrapper does for block-agy.
- A malformed or unknown payload is allowed.
- An unknown hook name, flag, or extra argument is a usage error and exits 64. An internal fault exits 1. Claude Code and Codex report either without blocking, and the pi wrapper treats it as a failure.
- Each `git` lookup times out after 1 s and each `gh` lookup after 2 s; all lookups in one call share 6 s. See [What the hooks read](#what-the-hooks-read) for what a lookup with no answer does.
- The whole process stops after 8 s and allows the call.

## Performance

The hooks run on every tool call, so:

- Calls decided from the payload alone never start a subprocess.
- `git` and `gh` run only for git/gh commands that need repository state. Their results are cached within the call. Lookups are skipped once a hook has blocked, and, for the rest of one hook's check of one command, once a lookup has had no answer; a "Could not check" block names that first lookup.
- A git subcommand that git itself ships (`status`, `log`, `diff`, …) never needs an alias lookup. Any other costs one `git config` call per alias level it follows, except a level the command sets with `-c`.

`make check` prints the first two rows. Measured end to end on the built binary, process start included, with all ten hooks on:

| Calls | p50 | p99 |
| --- | --- | --- |
| Decided from the payload alone | 1.5 ms | 2.0 ms |
| Needing a git lookup | 6.5 ms | 8.6 ms |
| Old `block-agy.sh` (bash + jq, one hook) | 10.3 ms | 12.3 ms |

## Development

```sh
make test    # go test ./...
make check   # gofmt, go vet, tests, binary latency, benchmarks
```

`main` only dispatches: `pre` and `post` are `hooks.Command`, which parses the
flags, reads the payload, and owns the 8 s deadline; `install` and `uninstall`
are `internal/install`. Tests run each hook through the package's `decide` with
real payloads. The binary tests pin the exact bytes of usage errors, the
deadline answer, and agy's allow answers. Git hooks run against temporary
repositories cloned from a local bare remote that has a `production` branch,
and `gh` lookups use a fake `gh` on `PATH`.
