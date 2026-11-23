package hooks

import (
	"strings"
	"testing"
)

// ghBoth mirrors pi's gh-pr tests, which enable draft-prs and no-admin-merge.
func ghBoth(t *testing.T, command string) Result {
	t.Helper()
	return run(t, "draft-prs,no-admin-merge", bash(command))
}

func expectBlocked(t *testing.T, hooks string, commands ...string) {
	t.Helper()
	for _, command := range commands {
		if r := run(t, hooks, bash(command)); r.Exit != 2 {
			t.Errorf("want block (%s): %q, got %+v", hooks, command, r)
		}
	}
}

func expectAllowed(t *testing.T, hooks string, commands ...string) {
	t.Helper()
	for _, command := range commands {
		if r := run(t, hooks, bash(command)); r.Exit != 0 || r.Stderr != "" {
			t.Errorf("want allow (%s): %q, got %+v", hooks, command, r)
		}
	}
}

const both = "draft-prs,no-admin-merge"

func TestDraftCreationAndUnrelatedCommandsRemainAvailable(t *testing.T) {
	expectAllowed(t, both, "gh pr create --draft", "gh pr create -d", "gh pr ready", "gh pr ready 42 --undo", "git status",
		"gh pr merge 42 --squash", "gh pr merge --auto --squash", "gh pr view --web")
}

func TestShellQuotingAndBoundariesDistinguishExecutionFromText(t *testing.T) {
	expectBlocked(t, both,
		"cd repo && gh pr create", "echo ok; gh pr merge --admin",
		`gh pr create --title "a --draft title"`, "gh pr create --title --draft",
		"gh pr create --draft; gh pr create", "gh pr create # --draft",
		"gh pr create\n# --draft", "gh pr create --body \"first\n--draft\"",
		"(gh pr create)", "x=$(gh pr merge 1 --admin)", "bash -c 'gh pr merge 1 --admin'",
		"sudo gh pr merge 1 --admin", "env GH_HOST=x gh pr create",
	)
	expectAllowed(t, both,
		`echo "gh pr merge --admin"`, "echo gh pr create", "# gh pr create",
		`gh pr create --title "semi;colon" --draft`,
		`gh pr create --title "a \"quoted\" title" --draft`,
		`gh pr merge --body "--admin"`, "gh pr merge --subject=--admin",
		"gh pr merge --body-file -", "gh pr create --draft \\\n --body-file -",
	)
}

func TestANSICAndLocaleQuotingRevealFlags(t *testing.T) {
	expectBlocked(t, both,
		"gh pr merge $'--admin'", `gh pr merge $"--admin"`, "gh pr merge --ad'min'", "gh pr merge $'--'admin",
		"gh pr merge $'--admin=true'", `gh $'pr' $"merge" 42 --admin`, "$'gh' pr merge --admin",
		`gh pr merge --body $'it\'s' --admin`, `gh pr merge --body $'x\\' --admin`,
		"gh pr create --draft $'--draft=false'", "gh pr create $'--web' --draft",
		`gh pr merge 1 $$'\' --admin #'`,
		"gh pr merge $1 --admin", `gh pr create --title "$(git log -1 --format=%s)"`,
		`gh pr merge $'--ad\x6din'`, `gh pr merge $'--ad\155in'`, `gh pr merge $'--admin'`,
		`gh pr merge $'--ad\0'min`, `$'\x67h' pr merge --admin`, `/usr/bin/$'\x67'h pr merge --admin`,
	)
	expectAllowed(t, both,
		"gh pr create $'--draft' --title t --body b", `gh pr create $"--draft"`, "gh pr create -$'d'",
		`gh pr create --draft --body $'Line 1\nLine 2\t\\ \'q\' \"x\" \? \a\b\e\E\f\r\v'`,
		"gh pr merge $'--admin=false'", "gh pr merge --admin=$'false'", "gh pr merge --body $'--admin'",
		`gh pr merge "$'\x6d'" --squash`, `gh pr merge \$'--admin'`, "gh pr merge $$'--admin'",
		`printf $'\x41\033[0m' && IFS=$'\x1f' read -r a b; gh pr merge 1 --squash`,
		"gh pr merge $PR --squash", `gh pr create --draft --title "$TITLE" --body "$(cat body.md)"`,
		`gh pr create --draft --body $'\cA'`, `gh pr create --draft --title $'\z'`,
	)
}

func TestBooleanValuesRepeatedFlagsAndShortClusters(t *testing.T) {
	expectBlocked(t, both,
		"gh pr create --draft --draft=false", "gh pr create -d=false",
		"gh pr create -ft--draft", "gh pr create --web", "gh pr create --dry-run",
		"gh pr merge --admin=true", "gh pr merge --admin=1",
	)
	expectAllowed(t, both,
		"gh pr create --draft=true", "gh pr create --draft=1", "gh pr create -fd",
		"gh pr create --draft=false -d", "gh pr create -dtTitle",
		"gh pr merge --admin=false", "gh pr merge --admin --admin=false",
		"gh pr merge -- --admin", "gh pr create --draft --body-file -",
	)
}

func TestFlagsBeforeCommandWords(t *testing.T) {
	expectBlocked(t, both,
		"gh pr --title Fix create --body Details",
		"gh --title=Fix pr create --body Details",
		"gh pr --draft=false create --title Fix --body Details",
		"gh pr --body Note merge 42 --admin --squash",
		"gh pr --admin=true merge 42 --squash",
		"gh --admin=true pr merge 42 --squash",
	)
}

func TestReorderedOptionsRetainValuesAndSelection(t *testing.T) {
	expectBlocked(t, both,
		"gh --title pr pr create", "gh pr --body merge create",
		"gh pr --body create merge 42 --admin", "gh pr --body merge merge 42 --admin",
		"gh pr --body -- merge 42 --admin", "gh pr --admin merge merge --squash",
		"gh pr -tcreate create", "gh pr -t Fix new", "gh pr -ftFix create", "gh pr -d=false create",
		"gh --draft=true pr create --draft=false", "gh pr --admin=false merge 42 --admin",
		"gh --help pr --title Fix create --help=false", "gh pr --web=false create --web --draft",
		"gh pr --title --draft create", "gh pr --body --help merge 42 --admin",
		"gh pr create -- --draft", `"gh" "pr" --title "Fix" "create"`,
	)
	expectAllowed(t, both,
		"gh pr --title Fix create --draft", "gh --draft=false pr --draft=true create",
		"gh pr --body Note merge 42 --admin=false", "gh pr --body --admin merge 42",
		"gh --admin=true pr merge 42 --admin=false", "gh pr --help=false create --help",
		"gh pr --web=true create --web=false --draft", "gh --repo pr pr --title merge create --draft",
		"gh pr -Rcreate --body merge merge 42 --admin=false", "gh pr -fd create", "gh pr -tmerge create -d",
		"gh pr --body create view", "gh pr --body merge view", "gh -- pr create", "gh pr -- create",
		"gh pr --body Note merge -- --admin", `echo "gh pr --admin=true merge"`,
	)
}

func TestRepositorySelectionAssignmentsAbsolutePathsAndAlias(t *testing.T) {
	expectBlocked(t, both, "gh -R org/repo pr create", "gh pr --repo=org/repo create",
		"GH_HOST=github.com /usr/bin/gh pr merge --admin", "gh pr new", "gh pr create --help=false")
	expectAllowed(t, both, "gh -Rorg/repo pr create -d", "gh pr create --help", "gh pr merge --admin --help")
}

func TestHelpFlagsWebAliasesAndLiteralWhitespace(t *testing.T) {
	expectBlocked(t, both,
		"gh --help=false pr merge --admin", "gh pr --help=false merge --admin",
		"gh --help=false pr create", "gh pr --help=false create",
		"gh --help pr create --help=false",
		"gh pr create --title=Fix --draft --body=Details",
	)
	expectAllowed(t, both, "gh --help pr create", "gh pr -h merge --admin",
		"gh pr create -w --web=false --draft", "gh pr create --web -w=false --draft",
		"gh pr create -h", "gh pr merge --admin -h")
}

func TestGhMessages(t *testing.T) {
	cases := map[string]string{
		"gh pr merge 42 --admin":                "[agent-hooks: no-admin-merge] Merge without --admin. Satisfy the repository review and check requirements instead of bypassing them.",
		`gh pr create --title "Fix" --body "D"`: "[agent-hooks: draft-prs] Create PRs as drafts. Add --draft; mark ready with gh pr ready after review.",
		"gh pr create --web":                    "[agent-hooks: draft-prs] gh cannot create drafts with --web. Drop --web/-w and pass --draft.",
		"gh pr create -w --draft":               "[agent-hooks: draft-prs] gh cannot create drafts with --web. Drop --web/-w and pass --draft.",
	}
	for command, want := range cases {
		if r := ghBoth(t, command); r.Stderr != want {
			t.Errorf("%q: got %q", command, r.Stderr)
		}
	}
}

func TestEachGhHookSwitchesIndependently(t *testing.T) {
	expectAllowed(t, "no-admin-merge", "gh pr create")
	expectBlocked(t, "no-admin-merge", "gh pr merge --admin")
	expectBlocked(t, "draft-prs", "gh pr create")
	expectAllowed(t, "draft-prs", "gh pr merge --admin")
	r := run(t, both, map[string]any{"tool_name": "Read", "tool_input": map[string]any{"file_path": "gh pr create"}})
	if r.Exit != 0 {
		t.Errorf("non-shell tool blocked: %+v", r)
	}
}

func TestAdminMergeThroughTheAPI(t *testing.T) {
	expectBlocked(t, "no-admin-merge",
		"gh api -X PUT repos/o/r/pulls/42/merge",
		"gh api --method=PUT /repos/{owner}/{repo}/pulls/42/merge -f merge_method=squash",
		"gh api --paginate --method put repos/o/r/pulls/7/merge/",
	)
	expectAllowed(t, "no-admin-merge",
		"gh api repos/o/r/pulls/42/merge",
		"gh api repos/o/r/pulls/42",
		"gh api -X PUT repos/o/r/pulls/42/requested_reviewers",
		"echo gh api -X PUT repos/o/r/pulls/42/merge",
	)
	if r := run(t, "no-admin-merge", bash("gh api -X PUT repos/o/r/pulls/42/merge")); !strings.Contains(r.Stderr, "gh pr merge") {
		t.Errorf("message should point at gh pr merge: %q", r.Stderr)
	}
}

func TestPRAssignee(t *testing.T) {
	expectBlocked(t, "pr-assignee",
		"gh pr create --draft", "gh pr create --draft --assignee", `gh pr create --draft --assignee ""`,
		"gh pr create --draft --title -a", "gh pr new -d",
	)
	expectAllowed(t, "pr-assignee",
		"gh pr create --draft --assignee @me", "gh pr create -d -a @me", "gh pr create -da@me",
		"gh pr create --assignee=aryan-binazir --draft", "gh pr create -d --assignee \"$ME\"",
		"gh pr ready", "gh pr merge 1", "gh pr create --help",
	)
}

func TestMergeHeadMatch(t *testing.T) {
	expectBlocked(t, "merge-head-match",
		"gh pr merge 42 --squash", "gh pr merge --auto --squash", "gh pr merge 42 --match-head-commit",
		`gh pr merge 42 --match-head-commit ""`, "cd x && gh pr merge",
	)
	expectAllowed(t, "merge-head-match",
		"gh pr merge 42 --squash --match-head-commit abc123", "gh pr merge --match-head-commit=abc123",
		`gh pr merge 42 --match-head-commit "$SHA"`, "gh pr merge 42 --disable-auto",
		"gh pr ready 42", "gh pr create --draft", "gh pr merge --help",
	)
}
