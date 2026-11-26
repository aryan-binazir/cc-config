package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPRTitlesFollowTheCommitFormat(t *testing.T) {
	r := newRepo(t)
	r.expect("commit-format", true,
		`gh pr create --draft --assignee @me --title "wip stuff" --body x`, `gh pr create -t "Fix the thing"`, "gh pr new --title=wip",
		`gh pr edit --title "wip stuff"`, `gh pr edit 23 --title "I found a bug"`, "gh pr edit --add-label bug -t wip",
		`gh pr create --title "feat(no-ticket): ok" --title wip`, // gh keeps the last title
	)
	res := r.run("commit-format", `gh pr create --title "wip stuff"`)
	for _, want := range []string{"PR titles", "type(PROJECT): description", `"wip stuff"`, "AGENT_HOOKS_ALLOW=commit-format"} {
		if !strings.Contains(res.Stderr, want) {
			t.Errorf("message should contain %q: %q", want, res.Stderr)
		}
	}
	r.expect("commit-format", false,
		`gh pr create --title "feat(no-ticket): Add x"`, `gh pr edit 23 --title "fix(BBA): y"`, "gh pr create --fill", "gh pr create",
		`gh pr create --title "$TITLE"`, `gh pr create --title "feat(no-ticket): $WHAT"`, `gh pr create --title ""`,
		`gh pr create --title wip --title "feat(no-ticket): ok"`,
		`gh pr edit 23 --title "I found a bug" -R upstream/repo`, `GH_REPO=upstream/repo gh pr create --title "Fix it"`,
		`gh pr edit https://github.com/upstream/repo/pull/5 --title "Fix it"`,
		`gh pr edit --add-label "wip stuff"`, "gh issue create --title wip", "gh pr merge 5 --subject wip",
		"AGENT_HOOKS_ALLOW=commit-format gh pr create --title wip",
	)
}

func TestPRTitleKeepsTheBranchType(t *testing.T) {
	r := newRepo(t)
	r.git("switch", "-q", "-c", "amb/x")
	r.write("a.txt", "a")
	r.git("add", "a.txt")
	r.git("commit", "-q", "-m", "feat(no-ticket): First")
	r.expect("commit-format", true, `gh pr create --title "fix(no-ticket): Other type"`, `gh pr edit --title "fix(BBA): Other type"`)
	if res := r.run("commit-format", `gh pr create --title "fix(no-ticket): Other type"`); !strings.Contains(res.Stderr, "`feat`") || !strings.Contains(res.Stderr, "AGENT_HOOKS_ALLOW=commit-format") {
		t.Errorf("message should name the branch type and the override: %q", res.Stderr)
	}
	r.expect("commit-format", true, `gh pr edit --attach shot.png --title "fix(no-ticket): Other type"`) // the file is not a PR selector
	r.git("branch", "release")
	r.expect("commit-format", false,
		`gh pr create --title "feat(BBA): Same type"`, `gh pr edit --title "feat(no-ticket): Same type"`,
		`gh pr edit 23 --title "fix(no-ticket): Some other pull request"`,
		`gh pr create --head other --title "fix(no-ticket): Some other branch"`,
		`gh pr create --base release --title "fix(no-ticket): Nothing ahead of release yet"`,
		`gh pr create --base "$BASE" --title "fix(no-ticket): Unknown base"`,
	)
	r.git("config", "branch.amb/x.gh-merge-base", "release") // where gh opens this branch's pull request
	r.expect("commit-format", false, `gh pr create --title "fix(no-ticket): Nothing ahead of release"`)
	r.expect("commit-format", true, `gh pr edit --title "fix(no-ticket): An open pull request keeps its base"`)
	r.git("config", "--unset", "branch.amb/x.gh-merge-base")
	t.Setenv("GH_REPO", "upstream/repo")
	r.expect("commit-format", false, `gh pr create --title "Fix it"`, `gh pr edit --title "fix(no-ticket): Other type"`)
}

func TestCommitFormatCanBeOverriddenWhereTheConventionDiffers(t *testing.T) {
	r := newRepo(t)
	state := t.TempDir()
	res := runOpts(t, Options{Phase: "pre", Only: only("commit-format"), StateDir: state}, bashIn(r.dir, `AGENT_HOOKS_ALLOW=commit-format git commit -m "Fix parser crash"`))
	if res.Exit != 0 || len(overrideLog(t, state)) != 1 {
		t.Errorf("override: %+v %v", res, overrideLog(t, state))
	}
	if res := r.run("commit-format", `git commit -m "Fix parser crash"`); !strings.Contains(res.Stderr, "AGENT_HOOKS_ALLOW=commit-format") {
		t.Errorf("the message should name the override: %q", res.Stderr)
	}
}

func TestAMalformedPRTitleIsRejectedFromThePayloadAlone(t *testing.T) {
	r := newRepo(t)
	marker := filepath.Join(t.TempDir(), "ran")
	fakeGit(t, "touch "+marker)
	res := r.run("commit-format", "gh pr create --title wip")
	if _, err := os.Stat(marker); err == nil || res.Exit != 2 {
		t.Errorf("a title that fails the format needs no lookup: %+v", res)
	}
}
