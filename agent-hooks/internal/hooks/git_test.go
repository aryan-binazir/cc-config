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

// repo is a work tree cloned from a local bare remote that has main and production.
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

// fakeGh puts a gh on PATH that reports baseRefName for gh pr view.
func fakeGh(t *testing.T, base string) {
	t.Helper()
	bin := t.TempDir()
	script := "#!/bin/sh\ncase \"$*\" in *'pr view'*baseRefName*) echo " + base + " ;; *) exit 1 ;; esac\n"
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

func TestCommitFormat(t *testing.T) {
	r := newRepo(t)
	r.expect("commit-format", true,
		`git commit -m "fixed stuff"`, `git commit -m "feat: no scope"`, `git commit -m "feat(bba): lower"`,
		`git commit -am "wip"`, `git commit --message="x"`, `git commit --message "x"`, `git commit -m "Feat(no-ticket): x"`,
		`git commit -m "feat(no-ticket):x"`, `git commit -m "feature(no-ticket): x"`, `git commit -m "feat(B): x"`,
		"git commit -F - <<'EOF'\nbad subject\n\nbody\nEOF",
		"git commit -m \"$(cat <<'EOF'\nbad subject\n\nbody\nEOF\n)\"",
		`git commit -q -mbad`, `cd /tmp && git -C `+r.dir+` commit -m bad`,
		`git commit -m "bad" -m "feat(no-ticket): second paragraph"`,
	)
	r.expect("commit-format", false,
		`git commit -m "feat(no-ticket): Add x"`, `git commit -m "fix(BBA): y"`, `git commit -m "chore(AB2): y"`,
		`git commit --amend --no-edit`, `git commit --no-edit`, `git commit`, `git commit --amend`, `git commit --fixup HEAD`,
		`git commit --squash=HEAD`, `git commit -m "fixup! anything"`, `git commit -m "squash! anything"`, `git commit -m "$MSG"`,
		`git commit -F msg.txt`, `git commit -C HEAD`, `git commit -c HEAD`, `echo git commit -m bad`, `git log -m`,
		"git commit -F - <<'EOF'\nfeat(no-ticket): Good\n\nbody\nEOF",
		"git commit -m \"$(cat <<'EOF'\ndocs(no-ticket): Good\n\nbody\nEOF\n)\"",
		`git commit -am "test(no-ticket): ok"`,
	)
	if res := r.run("commit-format", `git commit -m bad`); !strings.Contains(res.Stderr, "type(PROJECT): description") {
		t.Errorf("message should show the format: %q", res.Stderr)
	}
}

func TestCommitFormatKeepsTheBranchType(t *testing.T) {
	r := newRepo(t)
	// On the default branch there is no branch type to match.
	r.expect("commit-format", false, `git commit -m "fix(no-ticket): ok"`)
	r.git("switch", "-q", "-c", "amb/x")
	r.expect("commit-format", false, `git commit -m "fix(no-ticket): first commit picks the type"`)
	r.write("a.txt", "a")
	r.git("add", "a.txt")
	r.git("commit", "-q", "-m", "feat(no-ticket): First")
	r.expect("commit-format", false, `git commit -m "feat(BBA): more"`, `git commit --amend -m "fix(no-ticket): retype the only commit"`)
	r.expect("commit-format", true, `git commit -m "fix(no-ticket): other type"`)
	r.write("b.txt", "b")
	r.git("add", "b.txt")
	r.git("commit", "-q", "-m", "feat(no-ticket): Second")
	r.expect("commit-format", true, `git commit --amend -m "fix(no-ticket): x"`)
	res := r.run("commit-format", `git commit -m "fix(no-ticket): other type"`)
	if !strings.Contains(res.Stderr, "feat") {
		t.Errorf("message should name the branch type: %q", res.Stderr)
	}
}

func TestNoSecretsCommit(t *testing.T) {
	r := newRepo(t)
	r.write("notes.txt", "n")
	r.expect("no-secrets-commit", false, "git add -A", "git add .", "git add notes.txt", "git add .env.example", "git add config/.env.sample", "git add -u")
	r.expect("no-secrets-commit", true,
		"git add .env", "git add config/.env.local", "git add -f key.pem", "git add id_rsa", "git add ~/.ssh/id_ed25519.pub",
		"git add credentials-prod.json", "git add cert.p12", "git add a.pfx", "git add .claude/settings.local.json", "git add server.key",
		"git add notes.txt .env", "git -C sub add .env", "bash -c 'git add .env'",
	)
	r.write(".env", "SECRET=1")
	r.write("ignored.env", "x")
	r.write("src/server.key", "k")
	r.expect("no-secrets-commit", true, "git add -A", "git add .", "git add --all", "git add src/", "git add src", "git add '*.key'", "git add :/")
	r.expect("no-secrets-commit", false, "git add notes.txt", "git add -n .", "git add -u", "git add docs/", "git add ignored.env.bak 2>/dev/null; true")
	if res := r.run("no-secrets-commit", "git add -A"); !strings.Contains(res.Stderr, ".env") || !strings.Contains(res.Stderr, "src/server.key") {
		t.Errorf("message should list the files: %q", res.Stderr)
	}
	// commit -a stages tracked changes only.
	r.expect("no-secrets-commit", false, `git commit -am "feat(no-ticket): x"`)
	r.git("add", "-f", ".env")
	r.git("commit", "-q", "-m", "chore(no-ticket): Track env")
	r.write(".env", "SECRET=2")
	r.expect("no-secrets-commit", true, `git commit -am "feat(no-ticket): x"`, `git commit -a -m x`, `git commit --all -m x`, `git commit .env -m x`)
	r.expect("no-secrets-commit", false, `git commit -m "feat(no-ticket): x"`)
}

func TestNoSecretsOverrideIsLogged(t *testing.T) {
	r := newRepo(t)
	state := t.TempDir()
	res := runOpts(t, Options{Phase: "pre", Enabled: switches("no-secrets-commit"), StateDir: state}, bashIn(r.dir, "AGENT_HOOKS_ALLOW=secrets git add .env"))
	if res.Exit != 0 {
		t.Fatalf("override did not apply: %+v", res)
	}
	lines := overrideLog(t, state)
	if len(lines) != 1 || lines[0]["hook"] != "no-secrets-commit" || lines[0]["cwd"] != r.dir || lines[0]["command"] != "AGENT_HOOKS_ALLOW=secrets git add .env" || lines[0]["time"] == "" {
		t.Fatalf("bad log: %v", lines)
	}
	info, err := os.Stat(filepath.Join(state, "overrides.log"))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("log should be private: %v %v", info, err)
	}
	// The override names one hook and covers only its own command.
	for _, command := range []string{"AGENT_HOOKS_ALLOW=destructive git add .env", "AGENT_HOOKS_ALLOW=secrets true; git add .env", "AGENT_HOOKS_ALLOW=x,secretsy git add .env"} {
		if res := runOpts(t, Options{Phase: "pre", Enabled: switches("no-secrets-commit"), StateDir: state}, bashIn(r.dir, command)); res.Exit != 2 {
			t.Errorf("%q should still block: %+v", command, res)
		}
	}
	for _, command := range []string{"AGENT_HOOKS_ALLOW=destructive,secrets git add .env", "env AGENT_HOOKS_ALLOW=secrets git add .env", "AGENT_HOOKS_ALLOW=secrets bash -c 'git add .env'"} {
		if res := runOpts(t, Options{Phase: "pre", Enabled: switches("no-secrets-commit"), StateDir: state}, bashIn(r.dir, command)); res.Exit != 0 {
			t.Errorf("%q should be allowed: %+v", command, res)
		}
	}
	// No log line when nothing was overridden.
	before := len(overrideLog(t, state))
	runOpts(t, Options{Phase: "pre", Enabled: switches("no-secrets-commit"), StateDir: state}, bashIn(r.dir, "AGENT_HOOKS_ALLOW=secrets git add README.md"))
	if len(overrideLog(t, state)) != before {
		t.Error("logged an override that did not change the outcome")
	}
}

func TestDestructiveGitPushAndRm(t *testing.T) {
	r := newRepo(t)
	r.expect("destructive-git", true,
		"git push --force", "git push -f origin x", "git push -fu origin x", "git push --force --force-with-lease",
		"git push origin +x", "git push origin +HEAD:x", "git push --mirror",
		"rm -rf /", "rm -fr ~", "rm -r -f $HOME", `rm -Rf "$HOME"`, "rm --recursive --force /*", "rm -rf ~/", "rm -rf ${HOME}",
		"sudo rm -rf /", "rm -rf "+r.dir, "rm -rf "+r.dir+"/", "rm -rf .", "cd "+r.dir+"/sub && rm -rf ..", "rm -rf -- ~/*",
	)
	r.expect("destructive-git", false,
		"git push --force-with-lease", "git push origin x", "git push --force-with-lease origin +x", "git push -n -f",
		"rm -rf node_modules", "rm -rf /tmp/agent-hooks-x/build", "rm -r ~", "rm -f ~/.cache/x", "echo rm -rf /",
		`rm -rf "$HOME/.cache/foo"`, "rm -rf build", "rm -rf "+r.root+"/elsewhere", "rm -rf ./src",
	)
}

func TestDestructiveGitOnlyWhenTheTreeIsDirty(t *testing.T) {
	r := newRepo(t)
	risky := []string{"git clean -fd", "git clean -fdx", "git clean -f", "git clean -ffdx", "git reset --hard", "git reset --hard HEAD~1", "git checkout -- .", "git checkout .", "git restore ."}
	r.expect("destructive-git", false, risky...)
	r.write("README.md", "changed\n")
	r.expect("destructive-git", true, "git reset --hard", "git reset --hard HEAD", "git checkout -- .", "git checkout .", "git restore .", "git -C "+r.dir+" reset --hard")
	r.expect("destructive-git", false, "git clean -fd", "git clean -n -fd", "git reset --soft HEAD~1", "git reset", "git checkout main", "git restore --staged .")
	r.git("add", "README.md")
	r.expect("destructive-git", true, "git reset --hard")
	r.expect("destructive-git", false, "git checkout -- .", "git restore .")
	r.git("commit", "-q", "-m", "chore(no-ticket): Change")
	r.write("scratch.txt", "x")
	r.expect("destructive-git", true, "git clean -fd", "git clean -f", "git clean -fdx")
	r.expect("destructive-git", false, "git clean -nfd", "git clean -fd -n", "git reset --hard", "git clean -fd docs/")
	r.git("add", "scratch.txt")
	r.git("commit", "-q", "-m", "chore(no-ticket): Scratch")
	r.write("ignored.env", "x")
	r.expect("destructive-git", false, "git clean -fd")
	r.expect("destructive-git", true, "git clean -fdx", "git clean -fX")
}

func TestDestructiveOverride(t *testing.T) {
	state := t.TempDir()
	res := runOpts(t, Options{Phase: "pre", Enabled: switches("destructive-git"), StateDir: state}, bash("AGENT_HOOKS_ALLOW=destructive git push --force"))
	if res.Exit != 0 || len(overrideLog(t, state)) != 1 {
		t.Fatalf("override: %+v %v", res, overrideLog(t, state))
	}
}

const productionMessage = "[agent-hooks: no-production] Only push to production if the user, or an agent acting on the user's instruction, told you to. If so, rerun with AGENT_HOOKS_ALLOW=production. When delegating, pass the user's instruction along."

func TestNoProduction(t *testing.T) {
	r := newRepo(t)
	fakeGh(t, "main")
	r.expect("no-production", true,
		"git push origin production", "git push origin HEAD:production", "git push origin +production", "git push origin :production",
		"git push origin x:refs/heads/production", "git push --delete origin production", "git push origin -d production",
		"git push origin main production", "git push -u origin HEAD:refs/heads/production",
		"git branch -D production", "git branch -d production", "git branch -f production HEAD", "git branch -m production",
		"git branch -M main production", "git checkout -B production", "git switch -C production", "git switch --force-create production",
		"git update-ref refs/heads/production HEAD", "git update-ref -d refs/heads/production",
		"gh api -X DELETE repos/o/r/git/refs/heads/production", "gh api -X PATCH repos/o/r/git/refs/heads/production -f sha=x",
		"cd /tmp && git -C "+r.dir+" push origin production",
	)
	r.expect("no-production", false,
		"git push origin main", "git push origin production-fix", "git push origin HEAD:production-2", "git fetch origin production",
		"git checkout production", "git log production", "git branch production-2", "git push", "git push origin HEAD",
		"git merge origin/production", "git reset --hard origin/main", "gh pr merge 5 --match-head-commit abc",
		"gh api repos/o/r/git/refs/heads/production", "echo git push origin production", "git diff main production",
	)
	if res := r.run("no-production", "git push origin production"); res.Stderr != productionMessage {
		t.Errorf("message: %q", res.Stderr)
	}
}

func TestNoProductionUsesTheCurrentBranch(t *testing.T) {
	r := newRepo(t)
	r.git("switch", "-q", "-c", "production", "--track", "origin/production")
	r.git("config", "push.default", "upstream")
	r.expect("no-production", true, "git push", "git push origin HEAD", "git push origin", "git reset --hard origin/main", "git merge feature", "git pull", "git rebase main")
	r.expect("no-production", false, "git status", "git reset", "git fetch", "git log")
	r.git("switch", "-q", "-c", "deploy", "--track", "origin/production")
	r.expect("no-production", true, "git push")
}

func TestNoProductionPRBase(t *testing.T) {
	r := newRepo(t)
	fakeGh(t, "production")
	r.expect("no-production", true, "gh pr merge 5 --match-head-commit abc", "gh pr merge --squash")
	if res := run(t, "no-production", bash("gh pr merge 5")); res.Exit != 0 {
		t.Errorf("unknown cwd should skip the lookup: %+v", res)
	}
	r.expect("no-production", false, "AGENT_HOOKS_ALLOW=production gh pr merge 5", "gh pr merge 5 --disable-auto")
	t.Setenv("PATH", t.TempDir())
	r.expect("no-production", false, "gh pr merge 5")
}

func TestNoProductionOverride(t *testing.T) {
	r := newRepo(t)
	state := t.TempDir()
	res := runOpts(t, Options{Phase: "pre", Enabled: switches("no-production"), StateDir: state}, bashIn(r.dir, "AGENT_HOOKS_ALLOW=production git push origin production"))
	if res.Exit != 0 || len(overrideLog(t, state)) != 1 {
		t.Fatalf("override: %+v", res)
	}
}

func TestLookupsAreSkippedOnceAHookBlocks(t *testing.T) {
	r := newRepo(t)
	fakeGh(t, "production")
	res := r.run("merge-head-match,no-production", "gh pr merge 5")
	if res.Exit != 2 || strings.Contains(res.Stderr, "no-production") {
		t.Fatalf("expected only merge-head-match: %+v", res)
	}
}
