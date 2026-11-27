package hooks

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

func bash(command string) map[string]any {
	return map[string]any{"tool_name": "Bash", "tool_input": map[string]any{"command": command}, "cwd": "/nonexistent"}
}

func bashIn(dir, command string) map[string]any {
	return map[string]any{"tool_name": "Bash", "tool_input": map[string]any{"command": command}, "cwd": dir}
}

func encode(t *testing.T, payload any) []byte {
	t.Helper()
	if raw, ok := payload.(string); ok {
		return []byte(raw)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func run(t *testing.T, names string, payload any) Result {
	t.Helper()
	return runOpts(t, Options{Phase: "pre", Only: only(names)}, payload)
}

func runOpts(t *testing.T, o Options, payload any) Result {
	t.Helper()
	o.Stdin = encode(t, payload)
	if o.StateDir == "" {
		o.StateDir = t.TempDir()
	}
	return decide(o)
}

func only(names string) []string { return strings.Split(names, ",") }

func blocked(t *testing.T, r Result, hook string) bool {
	t.Helper()
	return r.Exit == 2 && strings.Contains(r.Stderr, "[agent-hooks: "+hook+"]")
}

func mustLookPath(t *testing.T, name string) string {
	t.Helper()
	p, err := exec.LookPath(name)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
