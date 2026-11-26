package hooks

import (
	"os"
	"path/filepath"
	"testing"
)

const gitHooks = "commit-format,destructive-git,no-production,no-secrets-commit"

func TestGitAliasesAreCheckedAsTheCommandTheyStandFor(t *testing.T) {
	r := newRepo(t)
	for alias, value := range map[string]string{
		"ci": "commit", "co": "checkout", "br": "branch", "pf": "push --force",
		"quiet":  "-c advice.detachedHead=false ci -q", // options first, then another alias
		"loop":   "loop",
		"status": "push --force", // git ignores an alias that shadows one of its own commands
		"sh":     "!git push --force",
	} {
		r.git("config", "alias."+alias, value)
	}
	r.write("README.md", "changed\n")
	r.expect("commit-format", true, `git ci -m "wip"`, `git quiet -m "wip"`, `git -c alias.wip=commit wip -m "wip"`, `git -C `+r.dir+` ci -m "wip"`)
	r.expect("destructive-git", true, "git co -- .", "git pf", "git pf origin x")
	r.expect("no-production", true, "git br -D production")
	r.expect("no-secrets-commit", true, "git stage .env")
	r.expect(gitHooks, false, `git ci -m "feat(no-ticket): ok"`, "git co main", "git loop", "git status", "git nosuchalias", "git sh")
}

func TestAliasesBelongToTheRepositoryTheCommandRunsIn(t *testing.T) {
	strict, loose := newRepo(t), newRepo(t)
	strict.git("config", "alias.x", "push --force")
	loose.git("config", "alias.x", "status")
	res := run(t, "destructive-git", bashIn(loose.root, "git -C "+loose.dir+" x && git -C "+strict.dir+" x"))
	if !blocked(t, res, "destructive-git") {
		t.Errorf("the second repository's alias force-pushes: %+v", res)
	}
	if res := run(t, "destructive-git", bashIn(loose.dir, "git x")); res.Exit != 0 {
		t.Errorf("the first repository's alias is harmless: %+v", res)
	}
}

func TestAGlobalAliasIsFollowedWhenTheDirectoryIsUnknown(t *testing.T) {
	r := newRepo(t)
	global := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(global, []byte("[alias]\n\tci = commit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	r.expect("commit-format", true, `cd "$WORKTREE" && git ci -m "wip"`)
}

func TestOrdinaryGitCommandsStartNoSubprocess(t *testing.T) {
	r := newRepo(t)
	marker := filepath.Join(t.TempDir(), "ran")
	fakeGit(t, "touch "+marker)
	res := runOpts(t, Options{Phase: "pre"}, bashIn(r.dir, "git status && git log -3 --oneline && git diff --stat && git show HEAD && git fetch origin && git rev-parse HEAD && git submodule status"))
	if _, err := os.Stat(marker); err == nil || res.Exit != 0 {
		t.Errorf("read-only git commands must be decided from the payload alone: %+v", res)
	}
}

func TestARepositoryAliasDoesNotFollowACommandIntoAnUnknownDirectory(t *testing.T) {
	r := newRepo(t)
	r.git("config", "alias.x", "push --force")
	r.expect("destructive-git", true, "git x")
	r.expect("destructive-git", false, `cd "$WORKTREE" && git x`)
}

func TestAliasWordsAreSplitTheWayGitSplitsThem(t *testing.T) {
	r := newRepo(t)
	r.git("config", "alias.cost", `commit -m "wip $5"`) // git passes $5 through; it is not a shell expansion
	r.git("config", "alias.good", `commit -m "feat(no-ticket): Cost $5 #1; it's done"`)
	r.git("config", "alias.open", `commit -m "unclosed`)                                                // git rejects this alias
	r.git("config", "alias.padded", `commit -m wip\ `)                                                  // an escaped space ends the message
	r.expect("commit-format", true, "git cost", "git padded", "git -c 'alias.nl=commit -m wip\\\n' nl") // an escaped newline ends that message too
	r.expect("commit-format", false, "git good", "git open")
}

func TestAnAliasNamedAfterACommandGitInstallsIsNeverUsed(t *testing.T) {
	r := newRepo(t)
	r.git("config", "alias.submodule", "push --force")
	r.git("config", "alias.status", "push --force")
	r.expect("destructive-git", false, "git submodule status", "git status")
}
