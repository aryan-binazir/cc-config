package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestUnknownHooksFlagsAndArgumentsAreUsageErrors(t *testing.T) {
	env := []string{"HOME=" + t.TempDir(), "XDG_STATE_HOME=" + t.TempDir()}
	for _, args := range [][]string{
		{"pre", "--only=block-agi"}, {"pre", "--only="}, {"pre", "--bogus"}, {"post", "extra"},
		{"install", "extra"}, {"uninstall", "--force"}, {"pre", "--only=block-agy", "--only=draft-prs"},
	} {
		p := runBinary(t, env, payload("agy -p hi"), args...)
		if p.exit != 64 || !strings.Contains(p.stderr, "usage") || p.stdout != "" {
			t.Errorf("%v: got %+v", args, p)
		}
	}
	const usage = "usage: agent-hooks pre|post [--agy] [--only=<hook>] | install | uninstall\n"
	agyLs := `{"toolCall":{"name":"run_command","args":{"CommandLine":"ls"}}}`
	for _, tc := range []struct {
		args           []string
		stdout, stderr string
	}{
		{[]string{"pre", "--bogus", "--agy"}, `{"decision":"allow"}` + "\n", "agent-hooks: unknown argument --bogus\n" + usage},
		{[]string{"pre", "--agy", "--only=nope"}, `{"decision":"allow"}` + "\n", `agent-hooks: no hook named "nope"` + "\n" + usage},
		{[]string{"pre", "--only=block-agy", "--agy", "--only=nope"}, `{"decision":"allow"}` + "\n", "agent-hooks: --only takes one hook\n" + usage},
		{[]string{"pre", "--only=nope", "--only=block-agy"}, "", `agent-hooks: no hook named "nope"` + "\n" + usage},
		{[]string{"post", "--agy", "--bogus"}, "", "agent-hooks: unknown argument --bogus\n" + usage},
		{[]string{"bogus"}, "", usage},
		{nil, "", usage},
	} {
		if p := runBinary(t, env, agyLs, tc.args...); p.exit != 64 || p.stdout != tc.stdout || p.stderr != tc.stderr {
			t.Errorf("%v: got %+v", tc.args, p)
		}
	}
	closed := exec.Command(binary(t), "pre", "--bogus")
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		t.Fatal(err)
	}
	syscall.Close(fds[0])
	w := os.NewFile(uintptr(fds[1]), "stdout")
	var errb bytes.Buffer
	closed.Stdout, closed.Stderr = w, &errb
	err = closed.Run()
	w.Close()
	if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 64 || errb.String() != "agent-hooks: unknown argument --bogus\n"+usage {
		t.Errorf("a usage error with stdout closed: %v %q", err, errb.String())
	}
	if p := runBinary(t, env, payload("agy -p hi"), "post", "--only=block-agy"); p.exit != 0 || p.stdout != "" {
		t.Errorf("a pre hook named in post runs nothing: %+v", p)
	}
}

func TestAStuckCallIsAllowedAfterEightSeconds(t *testing.T) {
	if testing.Short() {
		t.Skip("timing")
	}
	bin := binary(t)
	const timedOut = "agent-hooks: timed out; allowing the call\n"
	for _, tc := range []struct{ args, stdout string }{
		{"pre", ""}, {"pre --agy", `{"decision":"allow"}` + "\n"}, {"post --agy", ""},
	} {
		t.Run(tc.args, func(t *testing.T) {
			t.Parallel()
			outer, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(outer, bin, strings.Fields(tc.args)...)
			stdin, err := cmd.StdinPipe()
			if err != nil {
				t.Fatal(err)
			}
			defer stdin.Close()
			var out, errb bytes.Buffer
			cmd.Stdout, cmd.Stderr = &out, &errb
			start := time.Now()
			err = cmd.Run()
			elapsed := time.Since(start)
			if outer.Err() != nil {
				t.Fatalf("still running after %v", elapsed)
			}
			exit := -1
			if ee, ok := err.(*exec.ExitError); ok {
				exit = ee.ExitCode()
			}
			if exit != 1 || out.String() != tc.stdout || errb.String() != timedOut || elapsed < 8*time.Second || elapsed > 20*time.Second {
				t.Errorf("exit=%d stdout=%q stderr=%q after %v", exit, out.String(), errb.String(), elapsed)
			}
		})
	}
}

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
	agyLs := `{"toolCall":{"name":"run_command","args":{"CommandLine":"ls"}}}`
	if p := runBinary(t, env, agyLs, "pre", "--agy"); p != (proc{stdout: `{"decision":"allow"}`}) {
		t.Errorf("an agy answer ends without a newline: %+v", p)
	}
	if p := runBinary(t, env, agyLs, "post", "--agy"); p != (proc{stdout: "{}"}) {
		t.Errorf("agy post: %+v", p)
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
	if agy, _ := os.ReadFile(filepath.Join(home, ".gemini", "config", "hooks.json")); strings.Contains(string(agy), "agent-hooks") {
		t.Errorf("agy entry left: %s", agy)
	}
}

func percentile(samples []time.Duration, p float64) time.Duration {
	s := slices.Clone(samples)
	slices.Sort(s)
	return s[min(len(s)-1, int(float64(len(s))*p))]
}

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
			in("/nonexistent", "gh pr create --draft --assignee @me --fill --body-file b.md"),
			in("/nonexistent", "git status && git diff --stat"),
			`{"tool_name":"Read","tool_input":{"file_path":"/x"}}`,
		}},
		{"git-lookup", 100 * time.Millisecond, []string{
			in(repo, `git commit -m "feat(no-ticket): More"`),
			in(repo, "git add -A"),
			in(repo, "git reset --hard"),
			in(repo, "git push"),
			in(repo, "gh pr create --draft --assignee @me --title 'feat(no-ticket): More' --body-file b.md"),
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
