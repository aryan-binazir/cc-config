package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Real-shaped copies of the configs on Ar's machine, plus the legacy block-agy wiring.
const claudeOriginal = `{
  "cleanupPeriodDays": 3650,
  "attribution": {
    "pr": ""
  },
  "model": "opus[1m]",
  "hooks": {
    "SessionStart": [
      {
        "matcher": "*",
        "hooks": [
          {
            "type": "command",
            "command": "bash '/home/ar/.claude/hooks/herdr-agent-state.sh' session",
            "timeout": 10
          }
        ]
      }
    ]
  },
  "statusLine": {
    "type": "command",
    "command": "echo <ok> && true"
  },
  "alwaysThinkingEnabled": true
}
`

const claudeLegacy = `{
  "model": "opus[1m]",
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "bash ~/.claude/hooks/block-agy.sh"
          }
        ]
      },
      {
        "matcher": "Edit",
        "hooks": [
          {
            "type": "command",
            "command": "other-hook"
          }
        ]
      }
    ]
  }
}
`

const codexOriginal = `{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "command": "bash '/home/ar/.codex/herdr-agent-state.sh' session",
            "timeout": 10,
            "type": "command"
          }
        ]
      }
    ]
  }
}`

const codexLegacy = `{
  "hooks": {
    "PreToolUse": [
      {
        "hooks": [
          {
            "command": "bash ~/.claude/hooks/block-agy.sh",
            "type": "command"
          }
        ]
      }
    ],
    "SessionStart": []
  }
}`

const agyOriginal = `{
  "lint": {
    "PostToolUse": [
      {
        "matcher": "run_command",
        "hooks": [
          {
            "command": "./lint.sh"
          }
        ]
      }
    ]
  }
}
`

const bin = "/home/ar/.local/bin/agent-hooks"

var now = time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)

func setup(t *testing.T, claude, codex, agy string) Paths {
	t.Helper()
	dir := t.TempDir()
	p := Paths{
		Claude: filepath.Join(dir, "claude", "settings.json"),
		Codex:  filepath.Join(dir, "codex", "hooks.json"),
		Agy:    filepath.Join(dir, "gemini", "config", "hooks.json"),
	}
	for path, content := range map[string]string{p.Claude: claude, p.Codex: codex, p.Agy: agy} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if content != "" {
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	return p
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func decode(t *testing.T, text string) map[string]any {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal([]byte(text), &v); err != nil {
		t.Fatalf("%v\n%s", err, text)
	}
	return v
}

func handlers(t *testing.T, cfg map[string]any, event string) []map[string]any {
	t.Helper()
	hooks, _ := cfg["hooks"].(map[string]any)
	groups, _ := hooks[event].([]any)
	var out []map[string]any
	for _, g := range groups {
		group := g.(map[string]any)
		for _, h := range group["hooks"].([]any) {
			handler := h.(map[string]any)
			handler["matcher"] = group["matcher"]
			out = append(out, handler)
		}
	}
	return out
}

func TestInstallWiresEachHarness(t *testing.T) {
	p := setup(t, claudeOriginal, codexOriginal, "")
	changed, err := Install(p, bin, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(changed) != 3 {
		t.Fatalf("changed: %v", changed)
	}
	claude := decode(t, read(t, p.Claude))
	for event, arg := range map[string]string{"PreToolUse": "pre", "PostToolUse": "post"} {
		hs := handlers(t, claude, event)
		if len(hs) != 1 || hs[0]["command"] != "'"+bin+"' "+arg || hs[0]["matcher"] != "Bash" || hs[0]["type"] != "command" {
			t.Errorf("claude %s: %v", event, hs)
		}
	}
	codex := decode(t, read(t, p.Codex))
	if hs := handlers(t, codex, "PreToolUse"); len(hs) != 1 || hs[0]["command"] != "'"+bin+"' pre" || hs[0]["matcher"] != "Bash" {
		t.Errorf("codex pre: %v", hs)
	}
	if hs := handlers(t, codex, "PostToolUse"); len(hs) != 1 || hs[0]["command"] != "'"+bin+"' post" {
		t.Errorf("codex post: %v", hs)
	}
	agy := decode(t, read(t, p.Agy))
	want := map[string]any{"agent-hooks": map[string]any{"PreToolUse": []any{map[string]any{
		"matcher": "run_command|manage_task",
		"hooks":   []any{map[string]any{"type": "command", "command": "'" + bin + "' pre --agy", "timeout": float64(10)}},
	}}}}
	if !reflect.DeepEqual(agy, want) {
		t.Errorf("agy: %v", agy)
	}
	// Unrelated settings survive, in their original order and spelling.
	got := read(t, p.Claude)
	for _, s := range []string{`"statusLine"`, `"echo <ok> && true"`, `herdr-agent-state.sh`, `"alwaysThinkingEnabled": true`} {
		if !strings.Contains(got, s) {
			t.Errorf("lost %s", s)
		}
	}
	if strings.Index(got, `"cleanupPeriodDays"`) > strings.Index(got, `"model"`) || !strings.HasSuffix(got, "}\n") {
		t.Errorf("key order or trailing newline changed:\n%s", got)
	}
	if strings.HasSuffix(read(t, p.Codex), "\n") {
		t.Error("codex file gained a trailing newline")
	}
	if b := read(t, p.Claude+".bak-20261123T120000Z"); b != claudeOriginal {
		t.Errorf("backup: %q", b)
	}
	if info, _ := os.Stat(p.Claude); info.Mode().Perm() != 0o600 {
		t.Errorf("mode changed: %v", info.Mode())
	}
}

func TestInstallIsIdempotent(t *testing.T) {
	p := setup(t, claudeOriginal, codexOriginal, agyOriginal)
	if _, err := Install(p, bin, now); err != nil {
		t.Fatal(err)
	}
	first := []string{read(t, p.Claude), read(t, p.Codex), read(t, p.Agy)}
	changed, err := Install(p, bin, now.Add(time.Hour))
	if err != nil || len(changed) != 0 {
		t.Fatalf("second install changed %v (%v)", changed, err)
	}
	if again := []string{read(t, p.Claude), read(t, p.Codex), read(t, p.Agy)}; !reflect.DeepEqual(first, again) {
		t.Error("second install rewrote files")
	}
	if _, err := os.Stat(p.Claude + ".bak-20261123T130000Z"); err == nil {
		t.Error("backup written without a change")
	}
	// A moved binary replaces the old entry instead of adding a second one.
	if _, err := Install(p, "/opt/bin/agent-hooks", now); err != nil {
		t.Fatal(err)
	}
	if hs := handlers(t, decode(t, read(t, p.Claude)), "PreToolUse"); len(hs) != 1 || hs[0]["command"] != "'/opt/bin/agent-hooks' pre" {
		t.Errorf("moved binary: %v", hs)
	}
}

func TestInstallRemovesLegacyBlockAgyAndKeepsOtherHooks(t *testing.T) {
	p := setup(t, claudeLegacy, codexLegacy, agyOriginal)
	if _, err := Install(p, bin, now); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{p.Claude, p.Codex} {
		if text := read(t, path); strings.Contains(text, "block-agy.sh") {
			t.Errorf("legacy entry left in %s:\n%s", path, text)
		}
	}
	claude := decode(t, read(t, p.Claude))
	pre := handlers(t, claude, "PreToolUse")
	if len(pre) != 2 || pre[0]["command"] != "'"+bin+"' pre" || pre[1]["command"] != "other-hook" {
		t.Errorf("claude pre: %v", pre)
	}
	codex := decode(t, read(t, p.Codex))
	if hs := handlers(t, codex, "PreToolUse"); len(hs) != 1 {
		t.Errorf("codex pre: %v", hs)
	}
	if _, ok := codex["hooks"].(map[string]any)["SessionStart"]; !ok {
		t.Error("an unrelated empty event was dropped")
	}
	agy := decode(t, read(t, p.Agy))
	if _, ok := agy["lint"]; !ok {
		t.Error("lost the other agy hook")
	}
}

func TestUninstallRestoresTheOriginal(t *testing.T) {
	p := setup(t, claudeOriginal, codexOriginal, agyOriginal)
	if _, err := Install(p, bin, now); err != nil {
		t.Fatal(err)
	}
	if _, err := Uninstall(p, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]string{p.Claude: claudeOriginal, p.Codex: codexOriginal, p.Agy: agyOriginal} {
		if got := read(t, path); got != want {
			t.Errorf("%s not restored:\n%s", path, got)
		}
	}
	changed, err := Uninstall(p, now.Add(2*time.Minute))
	if err != nil || len(changed) != 0 {
		t.Errorf("second uninstall changed %v (%v)", changed, err)
	}
}

func TestUninstallEmptiesAnAgyFileItCreated(t *testing.T) {
	p := setup(t, "", "", "")
	if _, err := Install(p, bin, now); err != nil {
		t.Fatal(err)
	}
	if _, err := Uninstall(p, now); err != nil {
		t.Fatal(err)
	}
	if got := read(t, p.Agy); got != "{}\n" {
		t.Errorf("agy hooks file: %q", got)
	}
	if got := read(t, p.Claude); got != "{}\n" {
		t.Errorf("claude: %q", got)
	}
}

func TestInstallWritesThroughSymlinks(t *testing.T) {
	p := setup(t, "", codexOriginal, "")
	target := filepath.Join(t.TempDir(), "real-settings.json")
	if err := os.WriteFile(target, []byte(claudeOriginal), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, p.Claude); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(p, bin, now); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Lstat(p.Claude); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("symlink replaced by a file")
	}
	if !strings.Contains(read(t, target), "agent-hooks") {
		t.Error("target not updated")
	}
}

func TestMalformedConfigIsLeftAlone(t *testing.T) {
	p := setup(t, "{not json", codexOriginal, "")
	if _, err := Install(p, bin, now); err == nil {
		t.Fatal("want an error")
	}
	if got := read(t, p.Claude); got != "{not json" {
		t.Errorf("malformed file was modified: %q", got)
	}
}

func TestBackupsNeverOverwriteEachOther(t *testing.T) {
	p := setup(t, claudeOriginal, codexOriginal, "")
	if _, err := Install(p, bin, now); err != nil {
		t.Fatal(err)
	}
	if _, err := Uninstall(p, now); err != nil {
		t.Fatal(err)
	}
	if got := read(t, p.Claude+".bak-20261123T120000Z"); got != claudeOriginal {
		t.Errorf("original backup overwritten: %q", got)
	}
	if _, err := os.Stat(p.Claude + ".bak-20261123T120000Z-2"); err != nil {
		t.Errorf("second backup missing: %v", err)
	}
}

func TestOnlyOwnedAndLegacyEntriesAreTouched(t *testing.T) {
	mention := strings.Replace(claudeOriginal, `"bash '/home/ar/.claude/hooks/herdr-agent-state.sh' session"`, `"echo see block-agy.sh docs"`, 1)
	p := setup(t, mention, codexOriginal, `{"agent-hooks": {"PreToolUse": [{"matcher": "", "hooks": [{"command": "./mine.sh"}]}]}}`)
	if _, err := Install(p, bin, now); err == nil || !strings.Contains(err.Error(), "agent-hooks") {
		t.Fatalf("want a refusal to replace someone else's agent-hooks entry, got %v", err)
	}
	p = setup(t, mention, codexOriginal, "")
	if _, err := Install(p, bin, now); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(read(t, p.Claude), "echo see block-agy.sh docs") {
		t.Error("removed a command that only mentions block-agy.sh")
	}
}

func TestTrailingGarbageIsRejected(t *testing.T) {
	p := setup(t, `{}x`, codexOriginal, "")
	if _, err := Install(p, bin, now); err == nil {
		t.Fatal("accepted trailing garbage")
	}
}

func TestReinstallKeepsGroupPositions(t *testing.T) {
	codex := `{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "'/old/agent-hooks' pre",
            "timeout": 10
          }
        ]
      },
      {
        "hooks": [
          {
            "command": "custom",
            "type": "command"
          }
        ]
      }
    ]
  }
}`
	p := setup(t, "", codex, "")
	if _, err := Install(p, bin, now); err != nil {
		t.Fatal(err)
	}
	pre := handlers(t, decode(t, read(t, p.Codex)), "PreToolUse")
	if len(pre) != 2 || pre[0]["command"] != "'"+bin+"' pre" || pre[1]["command"] != "custom" {
		t.Errorf("groups moved: %v", pre)
	}
}

func TestUninstallLeavesUntouchedHomesAlone(t *testing.T) {
	odd := "{\"hooks\":{\"SessionStart\":[]},\"x\":1}"
	p := setup(t, "", odd, "{}")
	changed, err := Uninstall(p, now)
	if err != nil || len(changed) != 0 {
		t.Fatalf("changed %v (%v)", changed, err)
	}
	if _, err := os.Stat(p.Claude); !os.IsNotExist(err) {
		t.Error("created a Claude config")
	}
	if read(t, p.Codex) != odd || read(t, p.Agy) != "{}" {
		t.Error("rewrote configs without our entries")
	}
}

func TestBinaryPathIsQuotedAndNamed(t *testing.T) {
	p := setup(t, "", "", "")
	odd := "/home/o'brien/my bin/agent-hooks"
	if _, err := Install(p, odd, now); err != nil {
		t.Fatal(err)
	}
	if hs := handlers(t, decode(t, read(t, p.Claude)), "PreToolUse"); hs[0]["command"] != `'/home/o'\''brien/my bin/agent-hooks' pre` {
		t.Errorf("quoting: %v", hs[0]["command"])
	}
	if _, err := Uninstall(p, now); err != nil || strings.Contains(read(t, p.Claude), "agent-hooks") {
		t.Errorf("uninstall missed the quoted entry: %v", err)
	}
	if _, err := Install(p, "/usr/local/bin/ah", now); err == nil {
		t.Error("accepted a binary not named agent-hooks, which uninstall could not find")
	}
}

func TestInstallKeepsOtherGroupsAndHandlers(t *testing.T) {
	codex := `{
  "hooks": {
    "PreToolUse": [
      {
        "hooks": [
          {
            "command": "bash ~/.claude/hooks/block-agy.sh",
            "type": "command"
          }
        ]
      },
      {
        "matcher": "Write",
        "hooks": [
          {
            "command": "custom",
            "type": "command"
          },
          {
            "command": "'/x/agent-hooks' pre",
            "type": "command"
          }
        ]
      },
      {
        "hooks": [
          {
            "command": "'/x/agent-hooks' pre --only=block-agy",
            "type": "command"
          }
        ]
      }
    ]
  }
}`
	p := setup(t, "", codex, "")
	if _, err := Install(p, bin, now); err != nil {
		t.Fatal(err)
	}
	cfg := decode(t, read(t, p.Codex))
	groups := cfg["hooks"].(map[string]any)["PreToolUse"].([]any)
	if len(groups) != 3 {
		t.Fatalf("groups: %v", groups)
	}
	first := groups[0].(map[string]any)
	if first["matcher"] != "Bash" || first["hooks"].([]any)[0].(map[string]any)["command"] != "'"+bin+"' pre" {
		t.Errorf("the legacy group should become ours in place: %v", first)
	}
	second := groups[1].(map[string]any)
	if hs := second["hooks"].([]any); second["matcher"] != "Write" || len(hs) != 2 || hs[0].(map[string]any)["command"] != "custom" || hs[1].(map[string]any)["command"] != "'"+bin+"' pre" {
		t.Errorf("shared group should keep its matcher and handler positions: %v", second)
	}
	if third := groups[2].(map[string]any); third["hooks"].([]any)[0].(map[string]any)["command"] != "'/x/agent-hooks' pre --only=block-agy" {
		t.Errorf("custom --only handler touched: %v", third)
	}
}

func TestUninstallKeepsAPreexistingEmptyAgyFile(t *testing.T) {
	p := setup(t, "", "", "{}")
	if _, err := Install(p, bin, now); err != nil {
		t.Fatal(err)
	}
	if _, err := Uninstall(p, now); err != nil {
		t.Fatal(err)
	}
	if got := read(t, p.Agy); got != "{}" {
		t.Errorf("agy file: %q", got)
	}
}

func TestInstallNeverShiftsCodexIndices(t *testing.T) {
	codex := `{"hooks": {"PreToolUse": [
  {"matcher": "Write", "hooks": [{"command": "'/x/agent-hooks' pre", "type": "command"}, {"command": "custom", "type": "command"}]},
  {"matcher": "Bash", "hooks": [{"command": "'/y/agent-hooks' pre", "type": "command"}]},
  {"hooks": [{"command": "later", "type": "command"}]}
], "PostToolUse": [{"hooks": [{"command": "bash ~/.claude/hooks/block-agy.sh"}]}]}}`
	p := setup(t, "", codex, "")
	if _, err := Install(p, bin, now); err != nil {
		t.Fatal(err)
	}
	cfg := decode(t, read(t, p.Codex))
	groups := cfg["hooks"].(map[string]any)["PreToolUse"].([]any)
	commands := func(i int) []any { return groups[i].(map[string]any)["hooks"].([]any) }
	if len(groups) != 3 || commands(0)[1].(map[string]any)["command"] != "custom" || commands(2)[0].(map[string]any)["command"] != "later" {
		t.Errorf("indices shifted: %v", groups)
	}
	if groups[0].(map[string]any)["matcher"] != "Write" {
		t.Errorf("shared matcher changed: %v", groups[0])
	}
	if strings.Contains(read(t, p.Codex), "block-agy.sh") || strings.Contains(read(t, p.Codex), "/x/agent-hooks") || strings.Contains(read(t, p.Codex), "/y/agent-hooks") {
		t.Errorf("legacy or stale paths left:\n%s", read(t, p.Codex))
	}
}

func TestInstallLeavesOtherEventsAndSharedBashGroups(t *testing.T) {
	codex := `{"hooks": {"SessionStart": [{"hooks": [{"command": "bash ~/.claude/hooks/block-agy.sh"}, {"command": "custom"}]}],
"PreToolUse": [{"matcher": "Bash", "hooks": [{"command": "'/x/agent-hooks' pre"}, {"command": "mine"}]}]}}`
	p := setup(t, "", codex, "")
	if _, err := Install(p, bin, now); err != nil {
		t.Fatal(err)
	}
	cfg := decode(t, read(t, p.Codex))
	start := cfg["hooks"].(map[string]any)["SessionStart"].([]any)[0].(map[string]any)["hooks"].([]any)
	if len(start) != 2 {
		t.Errorf("SessionStart handlers moved: %v", start)
	}
	if pre := cfg["hooks"].(map[string]any)["PreToolUse"].([]any); len(pre) != 1 {
		t.Errorf("appended a second Bash group: %v", pre)
	}
}
