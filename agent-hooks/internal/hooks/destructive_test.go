package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestForcedCheckoutAndSwitchKeepUncommittedWork(t *testing.T) {
	r := newRepo(t)
	r.git("switch", "-q", "-c", "other")
	r.git("switch", "-q", "main") // so that - names a previous branch
	forced := []string{
		"git checkout -f", "git checkout --force", "git checkout -f other", "git checkout -f -B fresh", `git checkout -f "$BRANCH"`,
		"git switch -f other", "git switch --force other", "git switch --discard-changes other", "git switch -f -",
		"git checkout -f other --", "git checkout -f -bfresh", "git checkout -f --orphan fresh", "git checkout -f -",
		"git switch -f --force-create fresh", "git switch -f --orphan fresh",
	}
	r.expect("destructive-git", false, forced...)
	r.write("README.md", "changed\n")
	r.expect("destructive-git", true, forced...)
	if res := r.run("destructive-git", "git switch --discard-changes other"); !strings.Contains(res.Stderr, "AGENT_HOOKS_ALLOW=destructive") {
		t.Errorf("the message should name the override: %q", res.Stderr)
	}
	r.expect("destructive-git", false,
		"git checkout other", "git switch other", // without force git refuses on its own
		"git checkout -f -- notes.txt", "git checkout -f notes.txt", "git checkout -f HEAD notes.txt", "git checkout -f other -- notes.txt", // path mode touches only that path
		"git switch --force-create fresh", // resets the branch, keeps the changes
		"AGENT_HOOKS_ALLOW=destructive git checkout -f",
	)
	r.git("add", "README.md")
	r.expect("destructive-git", true, "git checkout -f", "git switch -f other")
}

func TestForcedCheckoutKeepsUntrackedFilesTheTargetTracks(t *testing.T) {
	r := newRepo(t)
	r.git("switch", "-q", "-c", "other")
	r.write("collide.txt", "tracked on other")
	r.git("add", "collide.txt")
	r.git("commit", "-q", "-m", "chore(no-ticket): Collide")
	r.git("switch", "-q", "main")
	r.git("push", "-q", "origin", "other:remote-only")
	r.write("collide.txt", "untracked work")
	r.expect("destructive-git", true,
		"git checkout -f other", "git switch -f other", "git checkout -f -B fresh other",
		"git checkout -f -bfresh other", "git checkout -f other --", "git switch -f --force-create fresh other",
		"git checkout -f -", "git switch -f -", // the previous branch is other
		"git checkout -f remote-only", "git switch -f remote-only", // git creates it from origin/remote-only
	)
	r.expect("destructive-git", false, "git checkout -f", "git checkout -f main")
}

func TestForcedWorktreeRemovalKeepsUncommittedWork(t *testing.T) {
	r := newRepo(t)
	wt := filepath.Join(r.root, "wt")
	r.git("worktree", "add", "-q", wt, "-b", "topic")
	forced := []string{
		"git worktree remove --force " + wt, "git worktree remove -f " + wt, "git worktree remove -f ../wt", "git worktree remove -f wt",
		"git worktree remove -f -- " + wt, "git worktree remove -ff " + wt,
	}
	put := func(name, content string) {
		t.Helper()
		full := filepath.Join(wt, name)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r.expect("destructive-git", false, forced...)
	put("build/out.o", "x")
	r.expect("destructive-git", false, forced...) // ignored build output: git would remove it without --force too
	put("scratch.txt", "wip")
	r.expect("destructive-git", true, forced...)
	if res := r.run("destructive-git", "git worktree remove --force "+wt); !strings.Contains(res.Stderr, "scratch.txt") {
		t.Errorf("the message should name what would be lost: %q", res.Stderr)
	}
	r.expect("destructive-git", false,
		"git worktree remove "+wt, "git worktree prune", `git worktree remove --force "$WT"`, "git worktree remove --force nowhere",
		"AGENT_HOOKS_ALLOW=destructive git worktree remove --force "+wt,
	)
	if err := os.Remove(filepath.Join(wt, "scratch.txt")); err != nil {
		t.Fatal(err)
	}
	put("README.md", "edited\n")
	r.expect("destructive-git", true, forced...)
}

func TestTwiceForcedRemovalKeepsALockedWorktree(t *testing.T) {
	r := newRepo(t)
	wt := filepath.Join(r.root, "wt")
	r.git("worktree", "add", "-q", wt, "-b", "topic")
	twice := []string{"git worktree remove -f -f " + wt, "git worktree remove --force --force wt", "git worktree remove -ff -- " + wt}
	r.expect("destructive-git", false, twice...)
	r.git("worktree", "lock", "--reason", "on a removable disk", wt)
	r.expect("destructive-git", true, twice...)
	r.expect("destructive-git", false, "git worktree remove -f "+wt) // git itself refuses a single --force here
}

func TestStashClearKeepsExistingStashes(t *testing.T) {
	r := newRepo(t)
	r.expect("destructive-git", false, "git stash clear")
	r.write("README.md", "changed\n")
	r.git("stash", "push", "-q")
	r.expect("destructive-git", true, "git stash clear")
	r.expect("destructive-git", false, "git stash drop", "git stash list", "git stash pop", "AGENT_HOOKS_ALLOW=destructive git stash clear")
}

func TestRecursiveRmOfARootBlocksWithoutForceAndThroughABareGlob(t *testing.T) {
	r := newRepo(t)
	r.expect("destructive-git", true,
		"rm -r ~", "rm -R /", "rm -r "+r.dir, "rm -rif "+r.dir, "rm -r --interactive=never "+r.dir,
		"rm -rf *", "rm -r *", "rm -rf -- * .[!.]*", "cd "+r.dir+" && rm -rf *",
		"rm -ri --interactive=n "+r.dir, "rm -ri --inter=never "+r.dir, "rm -ri --for "+r.dir, // GNU spellings that turn the prompt off
	)
	r.expect("destructive-git", false,
		"rm -ri "+r.dir, "rm -rfi "+r.dir, "rm -rI ~", "rm -r --interactive ~", "rm -rf --inter "+r.dir, "rm -rf --interactive=o ~", // rm asks first, and an agent cannot answer
		"rm -rf '*'", `rm -rf \*`, "cd "+r.dir+"/sub && rm -rf *", "rm *", "rm -f *",
	)
	if res := r.run("destructive-git", "rm -r *"); !strings.Contains(res.Stderr, "rm -r *") {
		t.Errorf("the message should quote the command: %q", res.Stderr)
	}
}
