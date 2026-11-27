package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func (r *repo) runWithin(l limits, hook, command string) Result {
	r.t.Helper()
	return runOpts(r.t.(*testing.T), Options{Phase: "pre", Only: only(hook), limits: l}, bashIn(r.dir, command))
}

func fakeGit(t *testing.T, body string) {
	t.Helper()
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestProductionMergeIsBlockedWhenThePRBaseCannotBeRead(t *testing.T) {
	r := newRepo(t)
	for name, gh := range map[string]string{"gh fails": "exit 1", "gh answers nonsense": "echo"} {
		fakeGhScript(t, gh)
		res := r.run("no-production", "gh pr merge 5 --squash --match-head-commit abc")
		if !blocked(t, res, "no-production") || !strings.Contains(res.Stderr, "gh pr view") || !strings.Contains(res.Stderr, "AGENT_HOOKS_ALLOW=production") {
			t.Errorf("%s: want a block that names the lookup and the override, got %+v", name, res)
		}
	}
	state := t.TempDir()
	res := runOpts(t, Options{Phase: "pre", Only: only("no-production"), StateDir: state}, bashIn(r.dir, "AGENT_HOOKS_ALLOW=production gh pr merge 5"))
	if res.Exit != 0 || len(overrideLog(t, state)) != 1 {
		t.Errorf("the override must still apply and be logged: %+v", res)
	}
	t.Setenv("GH_TOKEN", "ambient")
	fakeGhScript(t, `[ "$GH_TOKEN" = literal ] && echo production feature || echo main feature`)
	r.expect("no-production", true, "GH_TOKEN=literal gh pr merge 5")
	fakeGhScript(t, `[ "$GH_TOKEN" = ambient ] && echo production feature || echo main feature`)
	r.expect("no-production", true, `GH_TOKEN="$(cat token)" gh pr merge 5`)
	// gh takes the last --repo, and --repo over GH_REPO.
	fakeGhScript(t, `case "$*" in *"--repo owner/repo"*) echo production feature ;; *) echo main feature ;; esac`)
	r.expect("no-production", true, `gh pr merge 5 --repo "$R" --repo owner/repo`, `GH_REPO="$R" gh pr merge 5 --repo owner/repo`)
	fakeGhScript(t, "sleep 5; echo production feature")
	if res := r.runWithin(limits{time.Second, 100 * time.Millisecond, time.Second}, "no-production", "gh pr merge 5"); !blocked(t, res, "no-production") || !strings.Contains(res.Stderr, "timed out") {
		t.Errorf("slow gh: %+v", res)
	}
}

func TestAPRLookupThatCannotBeMadeAllowsTheMerge(t *testing.T) {
	r := newRepo(t)
	fakeGhScript(t, "exit 1")
	r.expect("no-production", false,
		`gh pr merge "$PR" --squash`, `gh pr merge 5 --repo "$REPO"`, `GH_REPO="$REPO" gh pr merge 5`, "gh pr merge 5 --disable-auto",
	)
	if res := run(t, "no-production", bash("gh pr merge 5")); res.Exit != 0 {
		t.Errorf("unknown directory: %+v", res)
	}
}

func TestAGitLookupWithNoAnswerBlocksHooksThatHaveAnOverride(t *testing.T) {
	r := newRepo(t)
	r.write("README.md", "changed\n")
	fakeGit(t, "sleep 5")
	fast := limits{100 * time.Millisecond, time.Second, time.Second}
	res := r.runWithin(fast, "destructive-git", "git reset --hard")
	if !blocked(t, res, "destructive-git") || !strings.Contains(res.Stderr, "git status timed out") || !strings.Contains(res.Stderr, "AGENT_HOOKS_ALLOW=destructive") {
		t.Errorf("want a block that names the lookup and the override, got %+v", res)
	}
	for _, command := range []string{"AGENT_HOOKS_ALLOW=destructive git reset --hard", "git status", "git log -3"} {
		if res := r.runWithin(fast, "destructive-git", command); res.Exit != 0 {
			t.Errorf("%q: want allowed, got %+v", command, res)
		}
	}
	if res := r.runWithin(fast, "destructive-git", "git push --force"); !strings.Contains(res.Stderr, "Don't force-push") {
		t.Errorf("a decision that needs no lookup keeps its own message: %+v", res)
	}
}

func TestAGitLookupThatAnswersNoIsNotAFailure(t *testing.T) {
	r := newRepo(t)
	r.git("switch", "-q", "--detach")
	r.expect("no-production", false, "git push", "git merge feature")
	if res := run(t, "destructive-git,no-secrets-commit", bashIn(r.root, "git reset --hard && git add -A")); res.Exit != 0 {
		t.Errorf("outside a repository git cannot do harm: %+v", res)
	}
}

func TestLookupsShareOneBudget(t *testing.T) {
	r := newRepo(t)
	real, err := filepath.Abs(mustLookPath(t, "git"))
	if err != nil {
		t.Fatal(err)
	}
	fakeGit(t, `sleep 0.2; exec "`+real+`" "$@"`)
	res := r.runWithin(limits{time.Second, time.Second, 300 * time.Millisecond}, "no-production", "git push")
	if !blocked(t, res, "no-production") || !strings.Contains(res.Stderr, "ran out of time") {
		t.Errorf("slow lookups must stop inside the budget and block, got %+v", res)
	}
}

func TestALookupKilledByASignalHasNoAnswer(t *testing.T) {
	r := newRepo(t)
	fakeGit(t, "kill -TERM $$")
	if res := r.run("destructive-git", "git reset --hard"); !blocked(t, res, "destructive-git") || !strings.Contains(res.Stderr, "AGENT_HOOKS_ALLOW=destructive") {
		t.Errorf("want a block that names the override, got %+v", res)
	}
}

func TestAHookStopsLookingUpAfterOneLookupHasNoAnswer(t *testing.T) {
	for _, tc := range []struct{ hook, command, lookup, override string }{
		{"commit-format", "git commit -m 'feat(AB): add a thing'", "git log", "commit-format"},
		{"no-production", "git push", "git symbolic-ref", "production"},
		{"destructive-git", "git push", "git symbolic-ref", "destructive"},
	} {
		t.Run(tc.hook, func(t *testing.T) {
			r := newRepo(t)
			launches := filepath.Join(t.TempDir(), "launches")
			fakeGit(t, `echo "$*" >> "`+launches+`"; kill -TERM $$`)
			res := r.run(tc.hook, tc.command)
			if !blocked(t, res, tc.hook) || !strings.Contains(res.Stderr, tc.lookup+" did not finish") || !strings.Contains(res.Stderr, "AGENT_HOOKS_ALLOW="+tc.override) {
				t.Errorf("%q: want a block that names the first lookup and the override, got %+v", tc.command, res)
			}
			if log, _ := os.ReadFile(launches); strings.Count(string(log), "\n") != 1 {
				t.Errorf("%q: want one git launch, got:\n%s", tc.command, log)
			}
		})
	}
}

func TestALookupWithNoAnswerStopsOnlyItsOwnInvocation(t *testing.T) {
	r := newRepo(t)
	r.write("untracked.txt", "new\n")
	real, err := filepath.Abs(mustLookPath(t, "git"))
	if err != nil {
		t.Fatal(err)
	}
	first := filepath.Join(t.TempDir(), "first")
	fakeGit(t, `[ -e "`+first+`" ] || { : > "`+first+`"; kill -TERM $$; }; exec "`+real+`" "$@"`)
	res := r.run("destructive-git", "AGENT_HOOKS_ALLOW=destructive git reset --hard && git clean -f")
	if !blocked(t, res, "destructive-git") || !strings.Contains(res.Stderr, "git clean would delete untracked files") {
		t.Errorf("the next command's lookups must still run, got %+v", res)
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

func TestSecretLookupsRunNoCleanFilterFromGitattributes(t *testing.T) {
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
}

func TestSecretLookupsRunNoCleanFilterFromInfoAttributes(t *testing.T) {
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

func TestLookupsRunInEveryCandidateDirectory(t *testing.T) {
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
