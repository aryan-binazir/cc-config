#!/usr/bin/env bash
# PreToolUse hook for Claude Code, Codex and pi: refuse any agent-run command that
# launches the Antigravity CLI (agy / antigravity / acg), in any mode. Google's
# consumer terms forbid wrapping agy in a third-party agent (print mode, ACP, or an
# interactive session driven through tmux), and accounts get suspended for it.
# People running agy in their own terminal are unaffected.
# Exit 2 + stderr blocks the call in all three tools.
set -u

payload=$(cat)

tool=$(jq -r '.tool_name // ""' <<<"$payload" 2>/dev/null)
case "$tool" in
  Write | Edit | MultiEdit | NotebookEdit | apply_patch | write | edit) exit 0 ;;
esac

commands=$(jq -r '
  .tool_input // {} | [.command, .cmd, .chars]
  | map(select(. != null) | if type == "array" then join(" ") else tostring end)
  | join("\n")
' <<<"$payload" 2>/dev/null) || exit 0

launchers='^(bash|sh|zsh|dash|fish|eval|source|\.|env|exec|command|nohup|setsid|timeout|stdbuf|nice|time|sudo|doas|xargs|parallel|watch|unbuffer|script|systemd-run|tmux|screen|ssh|mise)$'

is_agy() {
  local word=${1//[\"\'\`]/}
  word=${word##*=}
  word=${word##*/}
  [[ $word =~ ^(agy|antigravity|acg)$ ]]
}

block() {
  cat >&2 <<'EOF'
Blocked: agents must not run the Antigravity CLI (agy / antigravity / acg) at all, in any mode.
Google's consumer-plan terms forbid third-party agents wrapping agy (print mode, ACP, or driving it in tmux), and it gets the account suspended.
Do not work around this with scripts, other binary paths, aliases or variables.
If Gemini is needed, tell the user: use a GEMINI_API_KEY-based tool, or skip Gemini.
EOF
  exit 2
}

# Any mention of agy anywhere in the call, so `echo 'agy -p x' | bash` is caught
# by the launcher check below even though the bash segment never names agy.
mentioned=false
read -ra all_words <<<"$(tr ';&|()`\n' '       ' <<<"$commands")"
for word in "${all_words[@]}"; do
  is_agy "$word" && mentioned=true && break
done

while IFS= read -r segment; do
  read -ra words <<<"$segment"
  i=0
  while ((i < ${#words[@]})) && [[ ${words[i]} =~ ^[A-Za-z_][A-Za-z0-9_]*= ]]; do ((i++)); done
  ((i < ${#words[@]})) || continue

  is_agy "${words[i]}" && block

  first=${words[i]//[\"\']/}
  first=${first##*/}
  [[ $first =~ $launchers ]] || continue
  [[ $first == command && ${words[i + 1]:-} == -[vV] ]] && continue
  $mentioned && block
done < <(tr ';&|()`\n' '\n\n\n\n\n\n\n' <<<"$commands")

exit 0
