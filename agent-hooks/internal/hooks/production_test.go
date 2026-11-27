package hooks

import (
	"testing"
)

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
	r.expect("no-production", true, "git push --branches origin")
	r.expect("no-production", false, "git push --all --no-branches origin", "git push --branches --no-all origin") // one option, two names
	r.git("config", "remote.origin.push", "refs/heads/main:refs/heads/production")
	r.expect("no-production", true, "git push", "git push origin")
}

func TestNoProductionFollowsGlobRefspecs(t *testing.T) {
	r := newRepo(t)
	r.git("branch", "staging/production")
	r.expect("no-production", true, "git push origin 'refs/heads/staging/*:refs/heads/*'", "git update-ref -d -- refs/heads/production")
	r.git("branch", "production", "origin/production")
	r.expect("no-production", true, "git push --repo=origin --all")
	r.expect("no-production", false, "git push origin 'refs/tags/*:refs/tags/*'", "git push origin 'refs/heads/feature/*:refs/heads/feature/*'")
}

func TestNoProductionUnderPushDefaultMatching(t *testing.T) {
	r := newRepo(t)
	r.git("branch", "production")
	r.git("config", "push.default", "matching")
	r.expect("no-production", true, "git push", "git push --prune origin 'refs/heads/staging/*:refs/heads/*'")
}

func TestNoProductionCatchesPrunesAndMirrorsWithoutATrackingRef(t *testing.T) {
	r := newRepo(t)
	r.git("update-ref", "-d", "refs/remotes/origin/production")
	r.expect("no-production", true, "git push --all --prune origin", "git push --prune origin :", "git push --mir origin")
	r.git("config", "remote.origin.mirror", "true")
	r.expect("no-production", true, "git push", "git push origin")
}

func TestNoProductionAllowsAPruneThatSparesProduction(t *testing.T) {
	r := newRepo(t)
	r.expect("no-production", false, "git push --prune origin main", "git push --prune origin 'refs/tags/*:refs/tags/*'")
	r.expect("no-production", true, "git push --prune origin : 'refs/tags/*:refs/tags/*'")
}

func TestNoProductionCatchesAMirrorConfiguredOnTheOnlyRemote(t *testing.T) {
	solo := newRepo(t)
	solo.git("remote", "rename", "origin", "backup")
	solo.git("config", "remote.backup.mirror", "true")
	solo.git("branch", "-q", "--unset-upstream")
	solo.expect("no-production", true, "git push")
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
