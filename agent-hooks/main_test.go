package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/aryan-binazir/cc-config/agent-hooks/internal/hooks"
)

func TestShippedSwitchesNameEveryHook(t *testing.T) {
	on, err := parseSwitches(hooksYAML)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for name, enabled := range on {
		if !enabled {
			t.Errorf("%s is off", name)
		}
		names = append(names, name)
	}
	want := hooks.Names()
	slices.Sort(names)
	slices.Sort(want)
	if !slices.Equal(names, want) {
		t.Errorf("hooks.yaml lists %v, catalog has %v", names, want)
	}
}

func TestParseSwitches(t *testing.T) {
	on, err := parseSwitches("# c\nhooks:\n  a: true\n  b: false # off\n\n")
	if err != nil || !on["a"] || on["b"] {
		t.Fatalf("%v %v", on, err)
	}
	for _, bad := range []string{"a: true", "hooks:\n  a: yes", "hooks:\n  a true", "other:\nhooks:\n  a: true", "hooks:\n\ta: true", "hooks:\n  a: true\nx: 1"} {
		if _, err := parseSwitches(bad); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
}

func BenchmarkParseSwitches(b *testing.B) {
	for b.Loop() {
		if _, err := parseSwitches(hooksYAML); err != nil {
			b.Fatal(err)
		}
	}
}

// binary builds agent-hooks once per test binary.
var builtBinary string

func binary(t testing.TB) string {
	t.Helper()
	if builtBinary != "" {
		return builtBinary
	}
	dir, err := os.MkdirTemp("", "agent-hooks-bin")
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "agent-hooks")
	if msg, err := exec.Command("go", "build", "-o", out, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, msg)
	}
	builtBinary = out
	return out
}

func TestMain(m *testing.M) {
	code := m.Run()
	if builtBinary != "" {
		os.RemoveAll(filepath.Dir(builtBinary))
	}
	os.Exit(code)
}

type proc struct {
	exit           int
	stdout, stderr string
}

func runBinary(t testing.TB, env []string, stdin string, args ...string) proc {
	t.Helper()
	cmd := exec.Command(binary(t), args...)
	cmd.Stdin = strings.NewReader(stdin)
	cmd.Env = append(os.Environ(), env...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	p := proc{stdout: out.String(), stderr: errb.String()}
	if ee, ok := err.(*exec.ExitError); ok {
		p.exit = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return p
}

func payload(command string) string {
	data, _ := json.Marshal(map[string]any{"tool_name": "Bash", "tool_input": map[string]any{"command": command}, "cwd": "/nonexistent"})
	return string(data)
}

func TestBinaryEndToEnd(t *testing.T) {
	state := t.TempDir()
	env := []string{"XDG_STATE_HOME=" + state}
	cases := []struct {
		name     string
		stdin    string
		args     []string
		exit     int
		stdout   string
		stderrIn string
	}{
		{"claude block", payload("agy -p hi"), []string{"pre"}, 2, "", "[agent-hooks: block-agy] Blocked"},
		{"claude allow", payload("echo agy"), []string{"pre"}, 0, "", ""},
		{"codex argv", `{"tool_name":"Bash","tool_input":{"command":["bash","-lc","gh pr create"]},"cwd":"/x","turn_id":"t"}`, []string{"pre"}, 2, "", "draft-prs"},
		{"several hooks", payload("gh pr create"), []string{"pre"}, 2, "", "pr-assignee"},
		{"only", payload("gh pr create"), []string{"pre", "--only=block-agy"}, 0, "", ""},
		{"agy deny", `{"toolCall":{"name":"run_command","args":{"CommandLine":"agy"}}}`, []string{"pre", "--agy"}, 0, `"decision":"deny"`, ""},
		{"agy allow", `{"toolCall":{"name":"run_command","args":{"CommandLine":"ls"}}}`, []string{"pre", "--agy"}, 0, `{"decision":"allow"}`, ""},
		{"agy malformed", `garbage`, []string{"pre", "--agy"}, 0, `{"decision":"allow"}`, ""},
		{"post advice", payload("rg -r x TODO"), []string{"post"}, 0, `"additionalContext":"[agent-hooks: rg-replace-flag]`, ""},
		{"malformed", `{`, []string{"pre"}, 0, "", ""},
		{"empty", ``, []string{"post"}, 0, "", ""},
	}
	for _, tc := range cases {
		p := runBinary(t, env, tc.stdin, tc.args...)
		if p.exit != tc.exit || !strings.Contains(p.stdout, tc.stdout) || !strings.Contains(p.stderr, tc.stderrIn) || (tc.stdout == "" && p.stdout != "") {
			t.Errorf("%s: got %+v", tc.name, p)
		}
	}
	p := runBinary(t, env, payload("AGENT_HOOKS_ALLOW=destructive git push --force"), "pre")
	log, _ := os.ReadFile(filepath.Join(state, "agent-hooks", "overrides.log"))
	if p.exit != 0 || !strings.Contains(string(log), `"hook":"destructive-git"`) {
		t.Errorf("override: %+v log=%q", p, log)
	}
	if p := runBinary(t, env, "", "bogus"); p.exit == 0 || !strings.Contains(p.stderr, "usage") {
		t.Errorf("bogus subcommand: %+v", p)
	}
}

func TestBinaryInstallRoundTrip(t *testing.T) {
	home := t.TempDir()
	env := []string{"HOME=" + home}
	if p := runBinary(t, env, "", "install"); p.exit != 0 || !strings.Contains(p.stdout, "/hooks") {
		t.Fatalf("install: %+v", p)
	}
	settings, err := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if err != nil || !strings.Contains(string(settings), "'"+binary(t)+"' pre") {
		t.Fatalf("settings: %s %v", settings, err)
	}
	if p := runBinary(t, env, "", "uninstall"); p.exit != 0 {
		t.Fatalf("uninstall: %+v", p)
	}
	if _, err := os.Stat(filepath.Join(home, ".gemini", "config", "hooks.json")); !os.IsNotExist(err) {
		t.Errorf("agy file left: %v", err)
	}
}

func percentile(samples []time.Duration, p float64) time.Duration {
	s := slices.Clone(samples)
	slices.Sort(s)
	return s[min(len(s)-1, int(float64(len(s))*p))]
}

// TestBinaryLatency times the built binary end to end, process start included.
// It prints p50/p99 and fails only on gross regressions, since timing is noisy.
func TestBinaryLatency(t *testing.T) {
	if testing.Short() {
		t.Skip("timing")
	}
	repo := t.TempDir()
	git := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-c", "user.name=T", "-c", "user.email=t@e", "-c", "commit.gpgsign=false"}, args...)...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
	}
	git("init", "-q", "-b", "main")
	os.WriteFile(filepath.Join(repo, "a.txt"), []byte("a"), 0o644)
	git("add", ".")
	git("commit", "-q", "-m", "chore(no-ticket): Init")
	git("switch", "-q", "-c", "amb/x")
	os.WriteFile(filepath.Join(repo, "b.txt"), []byte("b"), 0o644)
	git("add", ".")
	git("commit", "-q", "-m", "feat(no-ticket): First")
	in := func(dir, command string) string {
		data, _ := json.Marshal(map[string]any{"tool_name": "Bash", "tool_input": map[string]any{"command": command}, "cwd": dir})
		return string(data)
	}
	env := []string{"XDG_STATE_HOME=" + t.TempDir()}
	classes := []struct {
		name  string
		limit time.Duration
		stdin []string
	}{
		{"payload-only", 20 * time.Millisecond, []string{
			in("/nonexistent", "echo hello"),
			in("/nonexistent", "cd src && rg -n TODO | sort | uniq -c | head -20"),
			in("/nonexistent", "gh pr create --draft --assignee @me --title t --body-file b.md"),
			in("/nonexistent", "git status && git diff --stat"),
			`{"tool_name":"Read","tool_input":{"file_path":"/x"}}`,
		}},
		{"git-lookup", 100 * time.Millisecond, []string{
			in(repo, `git commit -m "feat(no-ticket): More"`),
			in(repo, "git add -A"),
			in(repo, "git reset --hard"),
			in(repo, "git push"),
		}},
	}
	binary(t)
	for _, class := range classes {
		var samples []time.Duration
		for i := 0; i < 400; i++ {
			stdin := class.stdin[i%len(class.stdin)]
			start := time.Now()
			runBinary(t, env, stdin, "pre")
			samples = append(samples, time.Since(start))
		}
		p50, p99 := percentile(samples, 0.50), percentile(samples, 0.99)
		t.Logf("%-13s n=%d p50=%v p99=%v max=%v", class.name, len(samples), p50.Round(10*time.Microsecond), p99.Round(10*time.Microsecond), slices.Max(samples).Round(10*time.Microsecond))
		if p99 > class.limit {
			t.Errorf("%s p99 %v exceeds %v", class.name, p99, class.limit)
		}
	}
}
