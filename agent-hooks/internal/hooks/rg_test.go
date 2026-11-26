package hooks

import (
	"encoding/json"
	"testing"
)

const rgNote = "[agent-hooks: rg-replace-flag] In rg, -r means --replace: it rewrites each match in the output and does not mean recursive. rg already searches recursively, so drop -r unless you meant to replace."

func advice(t *testing.T, o Options, payload any) string {
	t.Helper()
	o.Phase, o.Only = "post", only("rg-replace-flag")
	r := runOpts(t, o, payload)
	if r.Exit != 0 || r.Stderr != "" {
		t.Fatalf("post must not block: %+v", r)
	}
	if r.Stdout == "" {
		return ""
	}
	var out struct {
		HookSpecificOutput struct{ HookEventName, AdditionalContext string }
	}
	if err := json.Unmarshal([]byte(r.Stdout), &out); err != nil || out.HookSpecificOutput.HookEventName != "PostToolUse" {
		t.Fatalf("bad post output %q", r.Stdout)
	}
	return out.HookSpecificOutput.AdditionalContext
}

func TestRgReplaceFlagAdvice(t *testing.T) {
	for _, command := range []string{
		"rg -r foo src", "rg -rn TODO", "rg -nr TODO", "rg -nrfoo TODO", "rg --replace=x TODO", "rg --replace x TODO",
		"rg TODO -r x", `rg "-r" x TODO`, "rg -tpy -r x TODO", "cd src && rg -i -r x TODO", "/usr/bin/rg -r x TODO", "LC_ALL=C rg -r x TODO",
	} {
		if got := advice(t, Options{}, bash(command)); got != rgNote {
			t.Errorf("%q: got %q", command, got)
		}
	}
	for _, command := range []string{
		"rg TODO", "rg -n TODO src", "rg -e -r TODO", "rg -er TODO", "rg -g -r TODO", "rg --glob -r TODO", "rg --glob=-r TODO",
		"rg -- -r", "rg TODO -- -r", "echo rg -r x", "grep -r TODO .", "rgx -r x", "rg --type-add -r TODO",
	} {
		if got := advice(t, Options{}, bash(command)); got != "" {
			t.Errorf("%q: unexpected advice %q", command, got)
		}
	}
	if got := advice(t, Options{}, map[string]any{"tool_name": "Read", "tool_input": map[string]any{"file_path": "rg -r x"}}); got != "" {
		t.Errorf("non-shell tool got advice")
	}
}

func TestRgAdviceIsNotPreAndAgyGetsEmptyObject(t *testing.T) {
	if r := run(t, "rg-replace-flag", bash("rg -r x")); r.Exit != 0 || r.Stdout != "" {
		t.Errorf("rg advice ran in pre: %+v", r)
	}
	r := runOpts(t, Options{Phase: "post", Only: only("rg-replace-flag"), Agy: true}, agyPayload("run_command", map[string]any{"CommandLine": "rg -r x"}))
	if r.Stdout != "{}" || r.Exit != 0 {
		t.Errorf("agy post must answer {}: %+v", r)
	}
}
