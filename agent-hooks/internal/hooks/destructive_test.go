package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestForcedCheckoutAndSwitchKeepUncommittedWork(t *testing.T) {
	r := newRepo(t)
	r.git("branch", "other")
	forced := []string{
		"git checkout -f", "git checkout --force", "git checkout -f other", "git checkout -f -B fresh", `git checkout -f "$BRANCH"`,
		"git switch -f other", "git switch --force other", "git switch --discard-changes other", "git switch -f -",
	}
	r.expect("destructive-git", false, forced...)
	r.write("README.md", "changed\n")
	r.expect("destructive-git", true, forced...)
	if res := r.run("destructive-git", "git switch --discard-changes other"); !strings.Contains(res.Stderr, "AGENT_HOOKS_ALLOW=destructive") {
		t.Errorf("the message should name the override: %q", res.Stderr)
	}
	r.expect("destructive-git", false,
		"git checkout other", "git switch other", // without force git refuses on its own
		"git checkout -f -- notes.txt", "git checkout -f notes.txt", // path mode touches only that path
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
	r.write("collide.txt", "untracked work")
	r.expect("destructive-git", true, "git checkout -f other", "git switch -f other", "git checkout -f -B fresh other")
	r.expect("destructive-git", false, "git checkout -f", "git checkout -f main")
}

func TestForcedWorktreeRemovalKeepsUncommittedWork(t *testing.T) {
	r := newRepo(t)
	wt := filepath.Join(r.root, "wt")
	r.git("worktree", "add", "-q", wt, "-b", "topic")
	forced := []string{"git worktree remove --force " + wt, "git worktree remove -f " + wt, "git worktree remove -f ../wt", "git worktree remove -f wt"}
	r.expect("destructive-git", false, forced...)
	if err := os.MkdirAll(filepath.Join(wt, "build"), 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(wt, "build", "out.o"), []byte("x"), 0o644)
	r.expect("destructive-git", false, forced...) // ignored build output: git would remove it without --force too
	os.WriteFile(filepath.Join(wt, "scratch.txt"), []byte("wip"), 0o644)
	r.expect("destructive-git", true, forced...)
	if res := r.run("destructive-git", "git worktree remove --force "+wt); !strings.Contains(res.Stderr, "scratch.txt") {
		t.Errorf("the message should name what would be lost: %q", res.Stderr)
	}
	r.expect("destructive-git", false,
		"git worktree remove "+wt, "git worktree prune", `git worktree remove --force "$WT"`, "git worktree remove --force nowhere",
		"AGENT_HOOKS_ALLOW=destructive git worktree remove --force "+wt,
	)
	os.Remove(filepath.Join(wt, "scratch.txt"))
	os.WriteFile(filepath.Join(wt, "README.md"), []byte("edited\n"), 0o644)
	r.expect("destructive-git", true, forced...)
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
	)
	r.expect("destructive-git", false,
		"rm -ri "+r.dir, "rm -rfi "+r.dir, "rm -rI ~", "rm -r --interactive ~", // rm asks first, and an agent cannot answer
		"rm -rf '*'", `rm -rf \*`, "cd "+r.dir+"/sub && rm -rf *", "rm *", "rm -f *",
	)
	if res := r.run("destructive-git", "rm -r *"); !strings.Contains(res.Stderr, "rm -r *") {
		t.Errorf("the message should quote the command: %q", res.Stderr)
	}
}
