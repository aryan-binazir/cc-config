package hooks

import (
	"encoding/json"
	"strings"
	"testing"
)

func agyPayload(tool string, args map[string]any) map[string]any {
	return map[string]any{
		"toolCall":       map[string]any{"name": tool, "args": args},
		"stepIdx":        19,
		"conversationId": "c",
		"workspacePaths": []string{"/nonexistent"},
		"modelName":      "auto",
	}
}

func TestPayloadShapesFromEachHarness(t *testing.T) {
	blockedShapes := map[string]any{
		"claude-code Bash":   bash("agy -p hi"),
		"codex Bash":         map[string]any{"tool_name": "Bash", "hook_event_name": "PreToolUse", "turn_id": "t", "tool_input": map[string]any{"command": "agy -p hi"}, "cwd": "/x"},
		"codex exec_command": map[string]any{"tool_name": "exec_command", "tool_input": map[string]any{"cmd": "agy -p hi"}},
		"codex shell argv":   map[string]any{"tool_name": "shell", "tool_input": map[string]any{"command": []string{"agy", "-p", "hi"}}},
		"codex write_stdin":  map[string]any{"tool_name": "write_stdin", "tool_input": map[string]any{"session_id": 3, "chars": "agy -p hi\n"}},
	}
	for harness, p := range blockedShapes {
		if r := run(t, "block-agy", p); r.Exit != 2 {
			t.Errorf("%s: want block for %v, got %+v", harness, p, r)
		}
	}
}

func TestAgyProtocolAnswersWithJSONDecisions(t *testing.T) {
	cases := []struct {
		payload  any
		decision string
	}{
		{agyPayload("run_command", map[string]any{"CommandLine": "agy -p hi", "Cwd": "/x"}), "deny"},
		{agyPayload("manage_task", map[string]any{"Action": "send_input", "TaskId": "1", "Input": "agy -p hi\n"}), "deny"},
		{agyPayload("run_command", map[string]any{"CommandLine": "echo agy"}), "allow"},
		{agyPayload("manage_task", map[string]any{"Action": "kill", "Input": "agy -p hi"}), "allow"},
		{agyPayload("view_file", map[string]any{"AbsolutePath": "/agy"}), "allow"},
	}
	for _, tc := range cases {
		r := runOpts(t, Options{Phase: "pre", Only: only("block-agy"), Agy: true}, tc.payload)
		var out struct{ Decision, Reason string }
		if r.Exit != 0 || json.Unmarshal([]byte(r.Stdout), &out) != nil || out.Decision != tc.decision {
			t.Errorf("%v: want decision %q, got %+v", tc.payload, tc.decision, r)
		}
		if tc.decision == "deny" && !strings.Contains(out.Reason, "[agent-hooks: block-agy]") {
			t.Errorf("deny without reason: %+v", r)
		}
	}
}

func TestAgyPayloadIsRecognizedWithoutTheFlag(t *testing.T) {
	r := run(t, "block-agy", agyPayload("run_command", map[string]any{"CommandLine": "agy"}))
	if r.Exit != 0 || !strings.Contains(r.Stdout, `"deny"`) {
		t.Fatalf("got %+v", r)
	}
}

func TestMalformedPayloadsAreAllowed(t *testing.T) {
	for _, raw := range []string{"", "not json", "[]", "null", `{"tool_name": 5}`, `{"tool_input": "agy"}`, `{"tool_input": {"command": 7}}`, `{"toolCall": null}`, `{"toolCall": {"name": "run_command", "args": {"CommandLine": 3}}}`, strings.Repeat("{", 100000)} {
		if r := run(t, "block-agy", raw); r.Exit != 0 || r.Stderr != "" || (r.Stdout != "" && r.Stdout != `{"decision":"allow"}`) {
			t.Errorf("%.40q: got %+v", raw, r)
		}
		r := runOpts(t, Options{Phase: "pre", Only: only("block-agy"), Agy: true}, raw)
		if r.Exit != 0 || r.Stdout != `{"decision":"allow"}` {
			t.Errorf("agy %.40q: got %+v", raw, r)
		}
	}
}

func TestOnlySelectsHooksWithinTheirPhase(t *testing.T) {
	if r := run(t, "draft-prs", bash("agy")); r.Exit != 0 {
		t.Errorf("a hook outside --only ran: %+v", r)
	}
	if r := runOpts(t, Options{Phase: "pre"}, bash("agy && gh pr create")); r.Exit != 2 || !strings.Contains(r.Stderr, "block-agy") || !strings.Contains(r.Stderr, "draft-prs") {
		t.Errorf("without --only every hook runs: %+v", r)
	}
	if r := runOpts(t, Options{Phase: "post", Only: only("block-agy")}, bash("agy")); r.Exit != 0 || r.Stdout != "" {
		t.Errorf("pre hook ran in post: %+v", r)
	}
}
