package hooks

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
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

type repo struct {
	t    testing.TB
	dir  string
	root string
}

func gitIn(t testing.TB, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null", "-c", "user.name=T", "-c", "user.email=t@example.com"}, args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func newRepo(t testing.TB) *repo {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	root := t.TempDir()
	remote := filepath.Join(root, "remote.git")
	dir := filepath.Join(root, "work")
	gitIn(t, root, "init", "-q", "--bare", "-b", "main", remote)
	gitIn(t, root, "init", "-q", "-b", "main", dir)
	r := &repo{t: t, dir: dir, root: root}
	r.write("README.md", "hi\n")
	r.write(".gitignore", "ignored.env\nbuild/\n")
	r.git("add", ".")
	r.git("commit", "-q", "-m", "chore(no-ticket): Init")
	r.git("remote", "add", "origin", remote)
	r.git("push", "-q", "origin", "main", "main:production")
	r.git("remote", "set-head", "origin", "main")
	r.git("branch", "-q", "--set-upstream-to=origin/main")
	return r
}

func (r *repo) git(args ...string) string { return gitIn(r.t, r.dir, args...) }

func (r *repo) write(name, content string) {
	r.t.Helper()
	p := filepath.Join(r.dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		r.t.Fatal(err)
	}
}

func (r *repo) run(hooks, command string) Result {
	r.t.Helper()
	return run(r.t.(*testing.T), hooks, bashIn(r.dir, command))
}

func (r *repo) expect(hooks string, wantBlock bool, commands ...string) {
	r.t.Helper()
	for _, command := range commands {
		res := r.run(hooks, command)
		if got := res.Exit == 2; got != wantBlock {
			r.t.Errorf("%s: %q: want block=%v, got %+v", hooks, command, wantBlock, res)
		}
	}
}

func fakeGh(t *testing.T, base string) {
	t.Helper()
	fakeGhScript(t, `case "$*" in *'pr view'*baseRefName*) echo `+base+` feature ;; *) exit 1 ;; esac`)
}

func fakeGhScript(t *testing.T, body string) {
	t.Helper()
	bin := t.TempDir()
	script := "#!/bin/sh\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func overrideLog(t *testing.T, stateDir string) []map[string]string {
	t.Helper()
	f, err := os.Open(filepath.Join(stateDir, "overrides.log"))
	if err != nil {
		return nil
	}
	defer f.Close()
	var lines []map[string]string
	s := bufio.NewScanner(f)
	for s.Scan() {
		var m map[string]string
		if err := json.Unmarshal(s.Bytes(), &m); err != nil {
			t.Fatal(err)
		}
		lines = append(lines, m)
	}
	return lines
}
