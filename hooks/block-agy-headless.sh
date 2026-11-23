#!/usr/bin/env bash
# PreToolUse hook for Claude Code and Codex: refuse agent-run commands that
# drive the Antigravity CLI (agy / antigravity / acg) non-interactively.
# Google's consumer terms forbid third-party agents wrapping agy; doing so
# gets the account suspended. Exit 2 + stderr blocks the call in both tools.
set -u

payload=$(cat)

tool=$(jq -r '.tool_name // ""' <<<"$payload" 2>/dev/null)
case "$tool" in
  Write | Edit | MultiEdit | NotebookEdit | apply_patch) exit 0 ;;
esac

commands=$(jq -r '
  .tool_input // {} | [.command, .cmd, .chars]
  | map(select(. != null) | if type == "array" then join(" ") else tostring end)
  | join("\n")
' <<<"$payload" 2>/dev/null) || exit 0

binary='(^|[[:space:];&|(`"'"'"'/=])(agy|antigravity|acg)([[:space:]]|$)'
headless='(^|[[:space:]])(-p|--?print|--?prompt|--?prompt-interactive|-i|--?input-format|acp)([[:space:]=]|$)'

while IFS= read -r segment; do
  if [[ $segment =~ $binary ]]; then
    rest=${segment#*"${BASH_REMATCH[0]}"}
    if [[ " $rest" =~ $headless ]]; then
      cat >&2 <<'EOF'
Blocked: agents must not run the Antigravity CLI (agy / antigravity / acg) in print, prompt, stream or ACP mode.
Google's consumer-plan terms forbid third-party agents driving agy, and it gets the account suspended.
Do not work around this with scripts, other binary paths, aliases or variables.
If Gemini is needed, tell the user: use a GEMINI_API_KEY-based tool, or skip Gemini.
EOF
      exit 2
    fi
  fi
done < <(tr ';&|\n' '\n\n\n\n' <<<"$commands")

exit 0
