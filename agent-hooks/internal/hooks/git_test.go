package hooks

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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
	r.expect("commit-format", false, `git commit -m "fix(no-ticket): ok"`)
	r.git("switch", "-q", "-c", "amb/x")
	r.expect("commit-format", false, `git commit -m "fix(no-ticket): first commit picks the type"`)
	r.write("a.txt", "a")
	r.git("add", "a.txt")
	r.git("commit", "-q", "-m", "feat(no-ticket): First")
	r.expect("commit-format", false, `git commit -m "feat(BBA): more"`, `git commit --amend -m "fix(no-ticket): retype the only commit"`,
		`git commit --amen -m "fix(no-ticket): git reads --amen as --amend"`)
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
	res := runOpts(t, Options{Phase: "pre", Only: only("no-secrets-commit"), StateDir: state}, bashIn(r.dir, "AGENT_HOOKS_ALLOW=secrets git add .env"))
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
	for _, command := range []string{"AGENT_HOOKS_ALLOW=destructive git add .env", "AGENT_HOOKS_ALLOW=secrets true; git add .env", "AGENT_HOOKS_ALLOW=x,secretsy git add .env"} {
		if res := runOpts(t, Options{Phase: "pre", Only: only("no-secrets-commit"), StateDir: state}, bashIn(r.dir, command)); res.Exit != 2 {
			t.Errorf("%q should still block: %+v", command, res)
		}
	}
	for _, command := range []string{"AGENT_HOOKS_ALLOW=destructive,secrets git add .env", "env AGENT_HOOKS_ALLOW=secrets git add .env", "AGENT_HOOKS_ALLOW=secrets bash -c 'git add .env'"} {
		if res := runOpts(t, Options{Phase: "pre", Only: only("no-secrets-commit"), StateDir: state}, bashIn(r.dir, command)); res.Exit != 0 {
			t.Errorf("%q should be allowed: %+v", command, res)
		}
	}
}

func TestOverrideExportedEarlierInTheCommand(t *testing.T) {
	r := newRepo(t)
	r.expect("no-secrets-commit", false,
		"export AGENT_HOOKS_ALLOW=secrets; git add .env",
		"export AGENT_HOOKS_ALLOW=secrets && git add .env",
		"declare -x AGENT_HOOKS_ALLOW=secrets; git add .env",
		"typeset -gx AGENT_HOOKS_ALLOW=secrets; git add .env",
		"export AGENT_HOOKS_ALLOW; AGENT_HOOKS_ALLOW=secrets; git add .env",
		"export AGENT_HOOKS_ALLOW=destructive; AGENT_HOOKS_ALLOW+=,secrets; git add .env",
		"{ export AGENT_HOOKS_ALLOW=secrets; }; git add .env",
		"if true; then export AGENT_HOOKS_ALLOW=secrets; fi; git add .env",
		"export AGENT_HOOKS_ALLOW=secrets; bash -c 'git add .env'",
		"bash -c 'export AGENT_HOOKS_ALLOW=secrets; git add .env'",
	)
	r.expect("no-secrets-commit", true,
		"AGENT_HOOKS_ALLOW=secrets; git add .env",
		"declare AGENT_HOOKS_ALLOW=secrets; git add .env",
		"git add .env; export AGENT_HOOKS_ALLOW=secrets",
		"(export AGENT_HOOKS_ALLOW=secrets); git add .env",
		"echo $(export AGENT_HOOKS_ALLOW=secrets); git add .env",
		"export AGENT_HOOKS_ALLOW=secrets | true; git add .env",
		"export AGENT_HOOKS_ALLOW=secrets & git add .env",
		"f() { export AGENT_HOOKS_ALLOW=secrets; }; git add .env",
		"export AGENT_HOOKS_ALLOW=secrets; unset AGENT_HOOKS_ALLOW; git add .env",
		"export AGENT_HOOKS_ALLOW=secrets; export -n AGENT_HOOKS_ALLOW; git add .env",
		"bash -c 'export AGENT_HOOKS_ALLOW=secrets'; git add .env",
	)
}

func TestOverrideLogSkipsNoOps(t *testing.T) {
	r := newRepo(t)
	state := t.TempDir()
	before := len(overrideLog(t, state))
	runOpts(t, Options{Phase: "pre", Only: only("no-secrets-commit"), StateDir: state}, bashIn(r.dir, "AGENT_HOOKS_ALLOW=secrets git add README.md"))
	if len(overrideLog(t, state)) != before {
		t.Error("logged an override that did not change the outcome")
	}
}

func TestDestructiveGitPushAndRm(t *testing.T) {
	r := newRepo(t)
	r.expect("destructive-git", true,
		"git push --force", "git push -f origin x", "git push -fu origin x", "git push --force --force-with-lease", "git push --force-with-lease origin +x",
		"git push origin +x", "git push origin +HEAD:x", "git push --mirror",
		"rm -rf /", "rm -fr ~", "rm -r -f $HOME", `rm -Rf "$HOME"`, "rm --recursive --force /*", "rm -rf ~/", "rm -rf ${HOME}",
		"sudo rm -rf /", "rm -rf "+r.dir, "rm -rf "+r.dir+"/", "rm -rf .", "cd "+r.dir+"/sub && rm -rf ..", "rm -rf -- ~/*",
	)
	r.expect("destructive-git", false,
		"git push --force-with-lease", "git push origin x", "git push -n -f",
		"rm -rf node_modules", "rm -rf /tmp/agent-hooks-x/build", "rm -f ~/.cache/x", "echo rm -rf /",
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
	res := runOpts(t, Options{Phase: "pre", Only: only("destructive-git"), StateDir: state}, bash("AGENT_HOOKS_ALLOW=destructive git push --force"))
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
		"git checkout --force production", "git switch --force production", // --force is not --force-create
		"git switch -B production", "git checkout -C production", // each is the other's spelling, and git rejects it
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
	r.expect("no-production", false, "AGENT_HOOKS_ALLOW=production gh pr merge 5", "gh pr merge 5 --disable-auto",
		"export AGENT_HOOKS_ALLOW=production && gh pr ready 5 && gh pr merge 5 --merge --match-head-commit abc")
	t.Setenv("PATH", t.TempDir())
	r.expect("no-production", false, "gh pr merge 5")
}

func TestNoProductionOverride(t *testing.T) {
	r := newRepo(t)
	state := t.TempDir()
	res := runOpts(t, Options{Phase: "pre", Only: only("no-production"), StateDir: state}, bashIn(r.dir, "AGENT_HOOKS_ALLOW=production git push origin production"))
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

func TestOverrideLogFallsBackToThePayloadCwdAndKeepsShellText(t *testing.T) {
	state := t.TempDir()
	payload := bashIn("/work/dir", `cd "$X" && AGENT_HOOKS_ALLOW=destructive git push -f`)
	if res := runOpts(t, Options{Phase: "pre", Only: only("destructive-git"), StateDir: state}, payload); res.Exit != 0 {
		t.Fatalf("%+v", res)
	}
	raw, _ := os.ReadFile(filepath.Join(state, "overrides.log"))
	if !strings.Contains(string(raw), `"cwd":"/work/dir"`) || !strings.Contains(string(raw), `$X\" && AGENT`) {
		t.Errorf("log line: %s", raw)
	}
}

func TestNoProductionCoversEveryPushDestination(t *testing.T) {
	r := newRepo(t)
	r.git("branch", "production", "origin/production")
	r.expect("no-production", true,
		"git push origin :", "git push origin 'refs/heads/*:refs/heads/*'", "git push --mirror origin", "git push --all origin",
		"git push origin -- production", "git branch -D -- production", "git fetch origin main:production",
		"git fetch origin +refs/heads/main:refs/heads/production", "gh repo sync --branch production", "gh repo sync -b production",
	)
	r.expect("no-production", false,
		"git fetch origin production", "git fetch origin production:refs/remotes/origin/production", "gh repo sync --branch main",
		"git branch -c production backup", "git branch -C production prod-copy",
	)
	// rebase's -m is --merge and takes no value, while -x runs a command: both
	// still check out production and rebase it.
	r.expect("no-production", true, "git rebase -m main production", "git rebase -x true main production", "git rebase --exec true main production",
		"git rebase -C 3 main production", "git rebase --empty drop main production",
		"git rebase -Sx main production", "git rebase -SC main production") // -S takes only an attached key
	r.git("config", "remote.origin.push", "refs/heads/main:refs/heads/production")
	r.expect("no-production", true, "git push", "git push origin")
}

func TestNoProductionOnTheProductionBranch(t *testing.T) {
	r := newRepo(t)
	r.git("switch", "-q", "-c", "production", "--track", "origin/production")
	r.expect("no-production", true, "git branch -m renamed", "git branch -M renamed")
	r.expect("no-production", false, "git merge --abort", "git rebase --abort", "git rebase --continue", "git merge --continue", "git pull --abort 2>/dev/null; true", "git branch -c backup",
		"git merge --abo", "git rebase --cont")
	r.expect("no-production", true, "git merge --message --abo feature", "git merge -F --abo feature") // --abo is the message, or its file
}

func TestNoProductionPRLookupKeepsSelectorAndRepository(t *testing.T) {
	r := newRepo(t)
	fakeGhScript(t, `case "$GH_REPO $*" in o/r*'pr view'*) echo production feature ;; *'pr view 7'*) echo production feature ;; *'pr view 9'*) echo main production ;; *'pr view'*) echo main feature ;; *) exit 1 ;; esac`)
	r.expect("no-production", true, "GH_REPO=o/r gh pr merge 5", "gh pr merge --squash -- 7", "gh pr merge 9 --delete-branch", "gh pr merge 9 -d")
	r.expect("no-production", false, "gh pr merge 5", "gh pr merge 9", "GH_REPO=x/y gh pr merge 5")
}

func TestDestructiveForceRefspecs(t *testing.T) {
	r := newRepo(t)
	r.expect("destructive-git", true, "git push --force-with-lease origin +main", "git push origin -- +main", "git push --force-with-lease=main:abc origin +main:main")
}

func TestGitLookupsFollowRepositorySelection(t *testing.T) {
	r := newRepo(t)
	r.write(".env", "x")
	for _, command := range []string{
		"git --git-dir=" + r.dir + "/.git --work-tree=" + r.dir + " add -A",
		"git --git-dir " + r.dir + "/.git --work-tree " + r.dir + " add -A",
		"GIT_DIR=" + r.dir + "/.git GIT_WORK_TREE=" + r.dir + " git add -A",
		"env -C " + r.dir + " git add -A",
		"env --chdir=" + r.dir + " git add -A",
	} {
		if res := run(t, "no-secrets-commit", bashIn("/", command)); res.Exit != 2 {
			t.Errorf("%q: %+v", command, res)
		}
	}
}

func TestCommitFormatValidatesTheMessageGitUses(t *testing.T) {
	r := newRepo(t)
	r.expect("commit-format", true,
		`git commit --no-edit -m "bad"`, `git commit --amend --no-edit -m bad`,
		"git commit -F - <<EOF\nbad subject\n\n$BODY\nEOF",
		"git commit -F - <<'EOF'\n\\feat(no-ticket): x\nEOF",
	)
	r.expect("commit-format", false,
		"git commit -F - 3<<'EOF'\nbad\nEOF", "git commit -F - <<EOF\nfeat(no-ticket): $X\nEOF",
		"git commit -F - <<EOF\nfeat(no-ticket): ok\n\n$BODY\nEOF",
		"git commit -Sm bad", // m is -S's key: no message is given
	)
}

func TestDestructiveCheckoutFromACommitAndClean(t *testing.T) {
	r := newRepo(t)
	r.write("README.md", "staged\n")
	r.git("add", "README.md")
	r.expect("destructive-git", true, "git checkout HEAD .", "git checkout HEAD -- .")
	r.expect("destructive-git", false, "git restore --source=HEAD .")        // writes only the worktree; the index keeps the staged change
	r.expect("destructive-git", false, "git checkout --conflict merge -- .") // merge is the conflict style: this checks out from the index
	r.git("commit", "-q", "-m", "chore(no-ticket): X")
	r.write("nested/deep/file.txt", "x")
	r.write("scratch.txt", "x")
	r.expect("destructive-git", false, "git clean -fd -e scratch.txt -e nested", "git clean -f -e scratch.txt")
	r.expect("destructive-git", true, "git clean -f", "git clean -fd nested", "git clean -f nested")
	r.git("config", "clean.requireForce", "false")
	r.expect("destructive-git", true, "git clean -d")
}

func TestNoSecretsCoversForcedAndStagedFiles(t *testing.T) {
	r := newRepo(t)
	r.write("build/.env", "x")
	r.expect("no-secrets-commit", true, "git add -f build/", "git add -f -A")
	r.expect("no-secrets-commit", false, "git add build/", "git add -A")
	r.write("tracked.txt", "a")
	r.write("key.pem", "k")
	r.git("add", "tracked.txt")
	r.git("add", "key.pem")
	r.git("commit", "-q", "-m", "chore(no-ticket): Seed")
	r.write("key.pem", "rotated")
	r.git("add", "key.pem")
	r.write("tracked.txt", "b")
	r.expect("no-secrets-commit", true, "git commit -am x")
	r.git("reset", "-q")
	if err := os.Remove(filepath.Join(r.dir, "key.pem")); err != nil {
		t.Fatal(err)
	}
	r.expect("no-secrets-commit", false, "git add key.pem", "git add -A")
}

func TestLookupsDoNotRunRepositoryHelpers(t *testing.T) {
	r := newRepo(t)
	marker := filepath.Join(t.TempDir(), "ran")
	script := filepath.Join(t.TempDir(), "fsmonitor.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ntouch "+marker+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.git("config", "core.fsmonitor", script)
	r.write(".env", "x")
	r.run("no-secrets-commit,destructive-git", "git add -A && git reset --hard")
	if _, err := os.Stat(marker); err == nil {
		t.Error("a lookup ran the repository's fsmonitor hook")
	}
}

func TestLookupsNeverChangeTheRepository(t *testing.T) {
	r := newRepo(t)
	r.write(".env", "x")
	r.write("scratch.txt", "x")
	r.expect("no-secrets-commit", true, "git add -A --no-dry-run", "git add --dry-run --no-dry-run -A")
	r.expect("destructive-git", true, "git clean -fd --no-dry-run", "git clean -n --no-dry-run -fd")
	if got := r.git("status", "--porcelain"); !strings.Contains(got, "?? .env") || !strings.Contains(got, "?? scratch.txt") {
		t.Fatalf("a hook lookup changed the repository:\n%s", got)
	}
	r.expect("destructive-git", true, "git push -n --no-dry-run --force", "git push --force-with-lease --no-force-with-lease origin +main")
}

func TestConfiguredAndMappedRefspecs(t *testing.T) {
	r := newRepo(t)
	r.git("config", "remote.origin.push", "+refs/heads/main:refs/heads/main")
	r.expect("destructive-git", true, "git push", "git push origin")
	r.git("config", "--unset", "remote.origin.push")
	r.git("branch", "staging/production")
	r.expect("no-production", true, "git push origin 'refs/heads/staging/*:refs/heads/*'", "git update-ref -d -- refs/heads/production")
	r.git("branch", "production", "origin/production")
	r.expect("no-production", true, "git push --repo=origin --all")
	r.expect("no-production", false, "git push origin 'refs/tags/*:refs/tags/*'", "git push origin 'refs/heads/feature/*:refs/heads/feature/*'")
}

func TestReviewRoundTwoGitCases(t *testing.T) {
	r := newRepo(t)
	r.write(".env", "x")
	r.write("list.txt", ".env\n")
	r.expect("no-secrets-commit", true, "git add --pathspec-from-file=list.txt", "cd /nonexistent-dir || git add -A")
	r.write("nested/README", "n")
	gitIn(t, filepath.Join(r.dir, "nested"), "init", "-q")
	r.expect("destructive-git", true, "git clean -ffd nested", "git -c clean.requireForce=false clean -d")
	r.expect("destructive-git", false, "git clean -fd nested")
	r.git("switch", "-q", "-c", "other")
	r.write("collide.txt", "tracked on other")
	r.git("add", "collide.txt")
	r.git("commit", "-q", "-m", "chore(no-ticket): Collide")
	r.git("switch", "-q", "main")
	r.write("collide.txt", "untracked work")
	r.expect("destructive-git", true, "git reset --hard other")
	r.expect("destructive-git", false, "git reset --hard")
	r.expect("commit-format", false, "git commit -F - <<'EOF' < msg.txt\nbad\nEOF")
}

func TestReviewRoundTwoShellAndPayload(t *testing.T) {
	expectBlocked(t, "draft-prs", "bash -o pipefail <<'EOF'\ngh pr create\nEOF")
	expectAllowed(t, "no-admin-merge", "gh api --help=true -X PUT repos/o/r/pulls/1/merge")
	if res := run(t, "block-agy", map[string]any{"tool_name": "Mystery", "tool_input": map[string]any{"command": "agy -p hi"}}); res.Exit != 0 {
		t.Errorf("unknown tool blocked: %+v", res)
	}
}

func TestReviewRoundThreeGitCases(t *testing.T) {
	r := newRepo(t)
	marker := filepath.Join(t.TempDir(), "filtered")
	r.git("config", "filter.mark.clean", "touch "+marker+"; cat")
	r.write(".gitattributes", "*.txt filter=mark\n")
	r.write("notes.txt", "x")
	r.write(".env", "x")
	r.expect("no-secrets-commit", true, "git add -A", "git add --no-dry -A", "git add ignored.env .env notes.txt")
	if _, err := os.Stat(marker); err == nil {
		t.Error("a lookup ran the repository's clean filter")
	}
	r.git("branch", "production")
	r.git("config", "push.default", "matching")
	r.expect("no-production", true, "git push", "git push --prune origin 'refs/heads/staging/*:refs/heads/*'")
	r.git("config", "--unset", "push.default")
	r.git("config", "remote.backup.url", r.root+"/remote.git")
	r.git("config", "remote.backup.push", "+refs/heads/main:refs/heads/main")
	r.git("config", "branch.main.pushRemote", "backup")
	r.expect("destructive-git", true, "git push", "git push -n --no-dry --force")
}

func TestResetHardKeepsUntrackedWork(t *testing.T) {
	r := newRepo(t)
	r.git("switch", "-q", "-c", "other")
	r.write("dir", "file on other")
	r.write("sub/tracked.txt", "on other")
	r.write("ignored.env", "on other")
	r.git("add", "-f", "dir", "sub/tracked.txt", "ignored.env")
	r.git("commit", "-q", "-m", "chore(no-ticket): Other")
	r.git("switch", "-q", "main")
	r.write("dir/untracked.txt", "work")
	r.expect("destructive-git", true, "git reset --hard other")
	os.RemoveAll(filepath.Join(r.dir, "dir"))
	r.write("ignored.env", "local secret")
	r.expect("destructive-git", true, "git reset --hard other")
	os.Remove(filepath.Join(r.dir, "ignored.env"))
	r.write("sub/tracked.txt", "work")
	if res := run(t, "destructive-git", bashIn(filepath.Join(r.dir, "sub"), "git reset --hard other")); res.Exit != 2 {
		t.Errorf("subdirectory reset: %+v", res)
	}
}

func TestReviewRoundThreeShellCases(t *testing.T) {
	r := newRepo(t)
	r.write(".env", "x")
	r.write("sub/a.txt", "a")
	r.expect("no-secrets-commit", true, "cd sub || exit 1; git add -A ..", "cd "+r.dir+" || exit 1; git add -A", "{ cd sub; false; } || git add -A ..")
	expectBlocked(t, "draft-prs", "bash -s arg1 <<'EOF'\ngh pr create\nEOF")
	expectBlocked(t, "no-admin-merge", "gh api --help=0 -X PUT repos/o/r/pulls/1/merge", "gh api --help=true --help=false -X PUT repos/o/r/pulls/1/merge")
	r.expect("commit-format", false, "git commit -F - <<'EOF' 0<&3\nbad\nEOF")
	clean := newRepo(t)
	clean.write("scratch.txt", "x")
	clean.expect("destructive-git", false, "git clean -f --exclude=scratch.txt", "git clean -f --exclude scratch.txt")
	clean.expect("destructive-git", true, "git clean -f --exclude=other.txt")
	r.git("config", "clean.requireForce", "FALSE")
	r.expect("destructive-git", true, "git clean")
	bare := filepath.Join(r.root, "bare.git")
	gitIn(t, r.root, "init", "-q", "--bare", bare)
	r.expect("destructive-git", true, "rm -rf "+bare)
}

func TestReviewRoundFourCases(t *testing.T) {
	r := newRepo(t)
	marker := filepath.Join(t.TempDir(), "filtered")
	r.git("config", "filter.mark.clean", "touch "+marker+"; cat")
	os.MkdirAll(filepath.Join(r.dir, ".git", "info"), 0o755)
	os.WriteFile(filepath.Join(r.dir, ".git", "info", "attributes"), []byte("* filter=mark\n"), 0o644)
	r.write(".env", "x")
	r.write("notes.txt", "n")
	r.expect("no-secrets-commit", true, "git add -A")
	if _, err := os.Stat(marker); err == nil {
		t.Error("a lookup ran a filter selected by .git/info/attributes")
	}
	r.git("update-ref", "-d", "refs/remotes/origin/production")
	r.expect("no-production", true, "git push --all --prune origin", "git push --prune origin :", "git push --mir origin")
	r.git("config", "remote.origin.mirror", "true")
	r.expect("no-production", true, "git push", "git push origin")
	r.git("config", "--unset", "remote.origin.mirror")
	r.write("sub/b.txt", "b")
	r.expect("no-secrets-commit", true, "cd missing && git status || git add -A")
	r.write("README.md", "edited\n")
	r.expect("destructive-git", true, "git reset --har")
}

func TestResetHardDirectoryCollisions(t *testing.T) {
	r := newRepo(t)
	r.git("switch", "-q", "-c", "other")
	r.write("dir/file", "tracked")
	r.write("build/out.txt", "tracked under an ignored dir")
	r.git("add", "-f", "dir/file", "build/out.txt")
	r.git("commit", "-q", "-m", "chore(no-ticket): Other")
	r.git("switch", "-q", "main")
	r.write("dir", "an untracked file where other has a directory")
	r.expect("destructive-git", true, "git reset --hard other")
	os.Remove(filepath.Join(r.dir, "dir"))
	r.write("build/out.txt", "local build output")
	r.expect("destructive-git", true, "git reset --hard other")
}

func TestQuotedHomeIsLiteral(t *testing.T) {
	expectAllowed(t, "destructive-git", "rm -rf '~'", `rm -rf '$HOME'`, `rm -rf "~"`)
	expectBlocked(t, "destructive-git", "rm -rf ~", "rm -rf $HOME", `rm -rf "$HOME"`)
	expectBlocked(t, "no-admin-merge", "gh api -X PUT repos/o/r/pulls/1/merge")
	expectAllowed(t, "no-admin-merge", "gh api -h --help=false -X PUT repos/o/r/pulls/1/merge")
}

func TestLookupsWorkInSHA256Repositories(t *testing.T) {
	dir := t.TempDir()
	gitIn(t, dir, "init", "-q", "--object-format=sha256", "-b", "main")
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644)
	gitIn(t, dir, "add", "a.txt")
	gitIn(t, dir, "commit", "-q", "-m", "chore(no-ticket): Init")
	os.WriteFile(filepath.Join(dir, ".env"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("changed"), 0o644)
	for _, tc := range []struct{ hooks, command string }{{"no-secrets-commit", "git add -A"}, {"destructive-git", "git reset --hard"}} {
		if res := run(t, tc.hooks, bashIn(dir, tc.command)); res.Exit != 2 {
			t.Errorf("%s in a SHA-256 repo: %+v", tc.command, res)
		}
	}
}

func TestSecretLookupsNeitherLockNorHash(t *testing.T) {
	r := newRepo(t)
	marker := filepath.Join(t.TempDir(), "filtered")
	r.git("config", "filter.mark.clean", "touch "+marker+"; cat")
	r.write(".gitattributes", "* filter=mark\n")
	for i := 0; i < 300; i++ {
		r.write(fmt.Sprintf("bulk/f%03d.txt", i), "x")
	}
	r.write(".env", "x")
	start := time.Now()
	r.expect("no-secrets-commit", true, "git add -A", "git add .", "git add bulk .env")
	if _, err := os.Stat(marker); err == nil {
		t.Error("a secrets lookup ran a clean filter")
	}
	if _, err := os.Stat(filepath.Join(r.dir, ".git", "index.lock")); err == nil {
		t.Error("a lookup left index.lock")
	}
	if time.Since(start) > 2*time.Second {
		t.Errorf("bulk add lookups took %v", time.Since(start))
	}
	r.write("sub/x.txt", "x")
	if res := run(t, "no-secrets-commit", bashIn(filepath.Join(r.dir, "sub"), "git add -A")); res.Exit != 2 {
		t.Errorf("add -A from a subdirectory stages the whole tree: %+v", res)
	}
}

func TestDirectoryCandidates(t *testing.T) {
	r := newRepo(t)
	other := t.TempDir()
	r.write("README.md", "edited\n")
	r.expect("destructive-git", true, "cd "+r.dir+" || cd "+other+"; git reset --hard", "cd "+other+" || cd "+r.dir+"; git reset --hard")
	r.write(".env", "x")
	if res := run(t, "no-secrets-commit", bashIn(r.root, "cd work && cd missing && git status || git add -A")); res.Exit != 2 {
		t.Errorf("intermediate directory: %+v", res)
	}
	deep := strings.Repeat("cd a || cd b || ", 14) + "true"
	start := time.Now()
	run(t, "no-secrets-commit", bashIn(r.dir, deep))
	if time.Since(start) > 100*time.Millisecond {
		t.Errorf("nested fallbacks took %v", time.Since(start))
	}
}

func TestReviewRoundFiveLowCases(t *testing.T) {
	r := newRepo(t)
	r.expect("no-production", false, "git push --prune origin main", "git push --prune origin 'refs/tags/*:refs/tags/*'")
	r.expect("no-production", true, "git push --prune origin : 'refs/tags/*:refs/tags/*'")
	expectAllowed(t, "destructive-git", `rm -rf \~`)
	solo := newRepo(t)
	solo.git("remote", "rename", "origin", "backup")
	solo.git("config", "remote.backup.mirror", "true")
	solo.git("branch", "-q", "--unset-upstream")
	solo.expect("no-production", true, "git push")
}

func TestResetKeepsIgnoredSiblings(t *testing.T) {
	r := newRepo(t)
	r.git("switch", "-q", "-c", "other")
	r.write("build/out.txt", "tracked")
	r.git("add", "-f", "build/out.txt")
	r.git("commit", "-q", "-m", "chore(no-ticket): Other")
	r.git("switch", "-q", "main")
	r.write("build/cache.log", "ignored sibling")
	r.expect("destructive-git", false, "git reset --hard other")
}
