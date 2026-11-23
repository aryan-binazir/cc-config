package hooks

import (
	"strings"
	"testing"
)

func TestBlockAgyBlocksEveryAgentLaunch(t *testing.T) {
	payloads := []any{
		bash(`agy -p "review this"`),
		bash(`timeout 45m agy -p "$PROMPT" --model gemini-4-neon-high`),
		bash("agy --model gemini-4-neon-high --print hi"),
		bash("cd /x && antigravity --prompt=hi"),
		bash("/home/ar/.local/share/mise/installs/antigravity-cli/latest/antigravity -p hi"),
		bash(`"$HOME/.local/share/mise/installs/antigravity-cli/latest/agy" models`),
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
		bash("echo 'agy -p hi' | bash"),
		bash(`printf "agy -p hi" | sh -s`),
		bash(`eval "agy -p hi"`),
		bash(`x='agy -p hi'; bash -c "$x"`),
		bash("echo agy -p hi > /tmp/r.sh && sh /tmp/r.sh"),
		bash("nohup agy > /dev/null 2>&1 &"),
		bash("mise exec antigravity-cli -- agy -p hi"),
		bash("ls\nagy -p hi"),
		map[string]any{"tool_name": "exec_command", "tool_input": map[string]any{"cmd": "agy -p hi"}},
		map[string]any{"tool_name": "shell", "tool_input": map[string]any{"command": []string{"bash", "-lc", "agy -p hi"}}},
		map[string]any{"tool_name": "write_stdin", "tool_input": map[string]any{"chars": "agy\n"}},
		map[string]any{"tool_name": "bash", "tool_input": map[string]any{"command": "agy acp"}},
		bash(`"agy" -p hi`),
		bash(`'/usr/bin/agy' -p hi`),
		bash(`a\gy -p hi`),
		bash("x=$(echo `agy -p hi`)"),
		bash("( cd /tmp && { agy -p hi; } )"),
		bash("if true; then agy -p hi; fi"),
		bash("sudo -u ar agy -p hi"),
		bash("env -i PATH=/bin agy -p hi"),
		bash("time agy -p hi"),
		bash("bash <<'EOF'\nagy -p hi\nEOF"),
		bash("bash -c \"bash -c 'agy -p hi'\""),
		bash("cat <<EOF | sh\nagy -p hi\nEOF"),
		bash("ssh box agy -p hi"),
		bash("xargs agy < list"),
		bash("command agy -p hi"),
		bash("diff <(agy models) /dev/null"),
		bash("unterminated 'quote; agy -p hi"),
		map[string]any{"tool_name": "Bash", "tool_input": map[string]any{"command": []string{"/opt/agy/bin/agy", "-p", "hi"}}},
	}
	for _, p := range payloads {
		r := run(t, "block-agy", p)
		if r.Exit != 2 || !strings.Contains(r.Stderr, "Blocked: agents must not run the Antigravity CLI") {
			t.Errorf("want block for %v, got %+v", p, r)
		}
	}
}

func TestBlockAgyAllowsCommandsThatOnlyMentionIt(t *testing.T) {
	payloads := []any{
		bash("rg -n agy ~/.gemini/antigravity-cli/log"),
		bash("echo agy -p hi"),
		bash("ps aux | grep agy"),
		bash("which agy && command -v agy && type agy"),
		bash(`git commit -m "Block agy calls from agents"`),
		bash("mise upgrade antigravity-cli"),
		bash("strategy -p foo"),
		bash("ls ~/.gemini/antigravity-cli/log"),
		map[string]any{"tool_name": "Write", "tool_input": map[string]any{"file_path": "/x.md", "content": "agy -p"}},
		map[string]any{"tool_name": "apply_patch", "tool_input": map[string]any{"command": "agy -p"}},
		map[string]any{"tool_name": "write", "tool_input": map[string]any{"path": "/x.md", "content": "agy -p"}},
		map[string]any{},
		bash("rg agy; grep -r antigravity ."),
		bash("cat agy.md"),
	}
	for _, p := range payloads {
		if r := run(t, "block-agy", p); r.Exit != 0 || r.Stderr != "" {
			t.Errorf("want allow for %v, got %+v", p, r)
		}
	}
}

func TestBlockAgyHasNoOverride(t *testing.T) {
	if r := run(t, "block-agy", bash("AGENT_HOOKS_ALLOW=block-agy agy -p hi")); r.Exit != 2 {
		t.Fatalf("override must not apply to block-agy: %+v", r)
	}
}
