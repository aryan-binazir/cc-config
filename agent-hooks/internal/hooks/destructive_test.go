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
	r.git("switch", "-q", "main")
	forced := []string{
		// A pathspec list is not read: it could name every file, or none, which leaves checkout moving HEAD.
		"git checkout -f --pathspec-from-file=paths.txt", "git checkout -f --pathspec-from-file=- </dev/null", "git checkout -f --pathspec-fro=paths.txt",
		"git checkout --pathspec-from-file=paths.txt", "git restore --pathspec-from-file paths.txt", "git restore --pathspec-fro=paths.txt",
		"git checkout --pathspec-from-file=- <<<notes.txt", `git checkout --pathspec-from-file="$LIST"`,
		"git checkout -f", "git checkout --force", "git checkout -f other", "git checkout -f -B fresh", `git checkout -f "$BRANCH"`,
		"git switch -f other", "git switch --force other", "git switch --discard-changes other", "git switch -f -",
		"git checkout -f other --", "git checkout -f -bfresh", "git checkout -f --orphan fresh", "git checkout -f --orph fresh", "git checkout -f -",
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
	r.expect("destructive-git", true, "git checkout -f", "git switch -f other", "git checkout -f --pathspec-from-file=- </dev/null")
	r.expect("destructive-git", false, "git checkout --pathspec-from-file=paths.txt") // the index is not what a checkout from it rewrites
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

	r.git("remote", "add", "fork", filepath.Join(r.root, "remote.git"))
	r.git("fetch", "-q", "fork")
	r.expect("destructive-git", false, "git checkout -f remote-only", "git switch -f remote-only") // two remotes have it: git refuses to guess
	r.expect("destructive-git", true, "git -c checkout.defaultRemote=origin checkout -f remote-only")
	r.expect("destructive-git", true, "git -c checkout.defaultRemote=origin checkout -f --no-guess --guess remote-only")
	r.expect("destructive-git", false, // no remote is tried
		"git -c checkout.defaultRemote=origin checkout -f --no-guess remote-only", "git -c checkout.defaultRemote=origin checkout -f --no-gue remote-only")
	r.git("config", "checkout.defaultRemote", "origin")
	r.expect("destructive-git", true, "git checkout -f remote-only", "git switch -f remote-only")
}

func TestForcedSwitchKeepsADirectoryTheTargetReplacesWithAFile(t *testing.T) {
	r := newRepo(t)
	r.git("switch", "-q", "-c", "other")
	r.write("cache", "a file on other")
	r.git("add", "-f", "cache")
	r.git("commit", "-q", "-m", "chore(no-ticket): Track a file named cache")
	r.git("switch", "-q", "main")
	forced := []string{"git switch --discard-changes other", "git checkout -f other", "git reset --hard other"}
	r.expect("destructive-git", false, forced...)
	r.write(".git/info/exclude", "cache/\n")
	r.write("cache/work.txt", "ignored, and only here")
	r.expect("destructive-git", true, forced...) // git removes the directory to put the file there
}

func TestRestoringIndexAndWorktreeKeepsStagedWork(t *testing.T) {
	r := newRepo(t)
	both := []string{
		"git restore --staged --worktree .", "git restore -SW .", "git restore -S -W :/", "git restore --source=HEAD --staged --worktree .",
		"git restore -S -W --pathspec-from-file=- <<<README.md",
	}
	r.expect("destructive-git", false, both...)
	r.write("README.md", "edited\n")
	r.git("add", "README.md")
	r.expect("destructive-git", true, both...) // the edit is in the index and the working copy, and both are rewritten
	r.expect("destructive-git", false,
		"git restore --staged .",                                                   // the working copy keeps the edit
		"git restore .", "git restore --worktree .", "git restore --source=HEAD .", // the index keeps the edit
		"git restore -SW notes.txt",
	)
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

func TestForcedWorktreeRemovalKeepsSubmoduleRepositories(t *testing.T) {
	r := newRepo(t)
	lib := filepath.Join(r.root, "lib")
	gitIn(t, r.root, "init", "-q", "-b", "main", lib)
	gitIn(t, lib, "commit", "-q", "--allow-empty", "-m", "chore(no-ticket): Init")

	// Status is clean in both worktrees, yet git removes neither without
	// --force: the submodule's own repository, stashes included, would go too.
	registered := filepath.Join(r.root, "registered")
	r.git("worktree", "add", "-q", registered, "-b", "registered")
	r.expect("destructive-git", false, "git worktree remove --force "+registered)
	gitIn(t, registered, "-c", "protocol.file.allow=always", "submodule", "add", "-q", lib, "lib")
	gitIn(t, registered, "commit", "-q", "-m", "chore(no-ticket): Add lib")

	embedded := filepath.Join(r.root, "embedded")
	r.git("worktree", "add", "-q", embedded, "-b", "embedded")
	gitIn(t, embedded, "clone", "-q", lib, "vendored")
	gitIn(t, embedded, "-c", "advice.addEmbeddedRepo=false", "add", "vendored")
	gitIn(t, embedded, "commit", "-q", "-m", "chore(no-ticket): Vendor lib")

	for _, wt := range []string{registered, embedded} {
		if status := gitIn(t, wt, "status", "--porcelain"); status != "" {
			t.Fatalf("the fixture should be clean: %q", status)
		}
		r.expect("destructive-git", true, "git worktree remove --force "+wt, "git worktree remove -f -- "+wt)
		r.expect("destructive-git", false, "git worktree remove "+wt, "AGENT_HOOKS_ALLOW=destructive git worktree remove --force "+wt)
	}
	if res := r.run("destructive-git", "git worktree remove --force "+registered); !strings.Contains(res.Stderr, "submodule") || !strings.Contains(res.Stderr, "AGENT_HOOKS_ALLOW=destructive") {
		t.Errorf("the message should say what would be lost and name the override: %q", res.Stderr)
	}
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
	link := filepath.Join(r.root, "link")
	if err := os.Symlink(r.dir, link); err != nil {
		t.Fatal(err)
	}
	r.expect("destructive-git", false, "rm -r "+link, "rm -rf "+link, "rm -rf ../link") // rm removes the link, not the repository behind it
	r.expect("destructive-git", true, "rm -rf "+link+"/*")                              // the glob reaches through it
}

func TestDestructiveGitPushAndRm(t *testing.T) {
	r := newRepo(t)
	r.expect("destructive-git", true,
		"git push --force", "git push -n --no-dry --force", "git push -f origin x", "git push -fu origin x", "git push --force --force-with-lease", "git push --force-with-lease origin +x",
		"git push origin +x", "git push origin +HEAD:x", "git push --mirror",
		"rm -rf /", "rm -fr ~", "rm -r -f $HOME", `rm -Rf "$HOME"`, "rm --recursive --force /*", "rm -rf ~/", "rm -rf ${HOME}",
		"sudo rm -rf /", "rm -rf "+r.dir, "rm -rf "+r.dir+"/", "rm -rf .", "cd "+r.dir+"/sub && rm -rf ..", "rm -rf -- ~/*",
	)
	r.expect("destructive-git", false,
		"git push --force-with-lease", "git push origin x", "git push -n -f",
		"rm -rf node_modules", "rm -rf /tmp/agent-hooks-x/build", "rm -f ~/.cache/x", "echo rm -rf /",
		`rm -rf "$HOME/.cache/foo"`, "rm -rf build", "rm -rf "+r.root+"/elsewhere", "rm -rf ./src",
	)
	bare := filepath.Join(r.root, "bare.git")
	gitIn(t, r.root, "init", "-q", "--bare", bare)
	r.expect("destructive-git", true, "rm -rf "+bare)
}

func TestDestructiveGitOnlyWhenTheTreeIsDirty(t *testing.T) {
	r := newRepo(t)
	risky := []string{"git clean -fd", "git clean -fdx", "git clean -f", "git clean -ffdx", "git reset --hard", "git reset --hard HEAD~1", "git checkout -- .", "git checkout .", "git restore ."}
	r.expect("destructive-git", false, risky...)
	r.write("README.md", "changed\n")
	r.expect("destructive-git", true, "git reset --hard", "git reset --hard HEAD", "git checkout -- .", "git checkout .", "git restore .", "git -C "+r.dir+" reset --hard", "git reset --har")
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

func TestDestructiveForceRefspecs(t *testing.T) {
	r := newRepo(t)
	r.expect("destructive-git", true, "git push --force-with-lease origin +main", "git push origin -- +main", "git push --force-with-lease=main:abc origin +main:main")
}

func TestConfiguredForcedRefspecsAreForcePushes(t *testing.T) {
	r := newRepo(t)
	r.git("config", "remote.origin.push", "+refs/heads/main:refs/heads/main")
	r.expect("destructive-git", true, "git push", "git push origin")
	r.git("config", "--unset", "remote.origin.push")
	r.git("config", "remote.backup.url", r.root+"/remote.git")
	r.git("config", "remote.backup.push", "+refs/heads/main:refs/heads/main")
	r.git("config", "branch.main.pushRemote", "backup")
	r.expect("destructive-git", true, "git push")
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
	r.expect("destructive-git", false, "git clean -fd -e scratch.txt -e nested", "git clean -f -e scratch.txt",
		"git clean -f --exclude=scratch.txt", "git clean -f --exclude scratch.txt")
	r.expect("destructive-git", true, "git clean -f", "git clean -fd nested", "git clean -f nested", "git clean -f --exclude=other.txt")
	r.git("config", "clean.requireForce", "false")
	r.expect("destructive-git", true, "git clean -d")
}

func TestCleanSparesANestedRepositoryUnlessForcedTwice(t *testing.T) {
	r := newRepo(t)
	r.write(".env", "x")
	r.write("list.txt", ".env\n")
	r.write("nested/README", "n")
	gitIn(t, filepath.Join(r.dir, "nested"), "init", "-q")
	r.expect("destructive-git", true, "git clean -ffd nested")
	r.expect("destructive-git", false, "git clean -fd nested")
}

func TestCleanNeedsNoForceWhereConfigSaysSo(t *testing.T) {
	r := newRepo(t)
	r.write(".env", "x")
	r.write("sub/a.txt", "a")
	r.expect("destructive-git", true, "git -c clean.requireForce=false clean -d")
	r.git("config", "clean.requireForce", "FALSE")
	r.expect("destructive-git", true, "git clean")
}

func TestResetHardToACommitThatTracksAnUntrackedFile(t *testing.T) {
	r := newRepo(t)
	r.git("switch", "-q", "-c", "other")
	r.write("collide.txt", "tracked on other")
	r.git("add", "collide.txt")
	r.git("commit", "-q", "-m", "chore(no-ticket): Collide")
	r.git("switch", "-q", "main")
	r.write(".env", "x")
	r.write("list.txt", ".env\n")
	r.write("collide.txt", "untracked work")
	r.expect("destructive-git", true, "git reset --hard other")
	r.expect("destructive-git", false, "git reset --hard")
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

func TestQuotedHomeIsLiteral(t *testing.T) {
	expectAllowed(t, "destructive-git", "rm -rf '~'", `rm -rf '$HOME'`, `rm -rf "~"`, `rm -rf \~`)
	expectBlocked(t, "destructive-git", "rm -rf ~", "rm -rf $HOME", `rm -rf "$HOME"`)
}
