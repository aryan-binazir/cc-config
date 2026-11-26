package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func lookupLimits(t *testing.T, git, gh, budget time.Duration) {
	t.Helper()
	oldGit, oldGh, oldBudget := gitTimeout, ghTimeout, lookupBudget
	gitTimeout, ghTimeout, lookupBudget = git, gh, budget
	t.Cleanup(func() { gitTimeout, ghTimeout, lookupBudget = oldGit, oldGh, oldBudget })
}

// fakeGit puts a git on PATH that runs body instead of the real one. Call it
// after the repository is set up.
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
	// A token the hook cannot read is no reason to skip the lookup.
	fakeGhScript(t, `[ "${GH_TOKEN:-none}" = none ] || [ "$GH_TOKEN" = literal ] || exit 1; echo production feature`)
	r.expect("no-production", true, `GH_TOKEN="$(cat token)" gh pr merge 5`, "GH_TOKEN=literal gh pr merge 5")
	lookupLimits(t, time.Second, 100*time.Millisecond, time.Second)
	fakeGhScript(t, "sleep 5; echo production feature")
	if res := r.run("no-production", "gh pr merge 5"); !blocked(t, res, "no-production") || !strings.Contains(res.Stderr, "timed out") {
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
	lookupLimits(t, 100*time.Millisecond, time.Second, time.Second)
	fakeGit(t, "sleep 5")
	res := r.run("destructive-git", "git reset --hard")
	if !blocked(t, res, "destructive-git") || !strings.Contains(res.Stderr, "git status timed out") || !strings.Contains(res.Stderr, "AGENT_HOOKS_ALLOW=destructive") {
		t.Errorf("want a block that names the lookup and the override, got %+v", res)
	}
	r.expect("destructive-git", false, "AGENT_HOOKS_ALLOW=destructive git reset --hard", "git status", "git log -3")
	if res := r.run("destructive-git", "git push --force"); !strings.Contains(res.Stderr, "Don't force-push") {
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
	lookupLimits(t, time.Second, time.Second, 300*time.Millisecond)
	fakeGit(t, `sleep 0.2; exec "`+real+`" "$@"`)
	res := r.run("no-production", "git push")
	if !blocked(t, res, "no-production") || !strings.Contains(res.Stderr, "ran out of time") {
		t.Errorf("slow lookups must stop inside the budget and block, got %+v", res)
	}
}
