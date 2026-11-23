package hooks

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/aryan-binazir/cc-config/agent-hooks/internal/shell"
)

var commitSubject = regexp.MustCompile(`^(feat|fix|chore|refactor|test|docs|build|ci|revert)\((no-ticket|[A-Z][A-Z0-9]+)\): \S`)
var commitType = regexp.MustCompile(`^(feat|fix|chore|refactor|test|docs|build|ci|revert)\(`)

var commitAliases = map[string]string{
	"--message": "-m", "--file": "-F", "--reuse-message": "-C", "--reedit-message": "-c", "--template": "-t", "--all": "-a",
}
var commitLongValues = []string{"-m", "-F", "-C", "-c", "-t", "--fixup", "--squash", "--author", "--date", "--trailer", "--cleanup", "--pathspec-from-file"}

// commitFormat enforces `type(PROJECT): description` on git commit messages
// and one commit type per branch.
var commitFormat = hook{name: "commit-format", phase: "pre", check: func(c *ctx, inv shell.Invocation) string {
	g := parseGit(inv, c.opts.Home)
	if g == nil || g.sub != "commit" {
		return ""
	}
	o := parseOpts(g.args, "mFCct", commitLongValues, commitAliases)
	if o.has("--no-edit", "--fixup", "--squash", "-C", "-c", "--dry-run") {
		return ""
	}
	var message string
	switch {
	case len(o.vals["-m"]) > 0:
		w := o.vals["-m"][0]
		if w.Dynamic {
			return ""
		}
		message = w.Val
	case len(o.vals["-F"]) > 0 && o.vals["-F"][0].Val == "-" && inv.Stdin != nil:
		message = *inv.Stdin
	default:
		return "" // the editor writes the message
	}
	subject, _, _ := strings.Cut(strings.TrimLeft(message, " \t\n"), "\n")
	subject = strings.TrimRight(subject, " \t\r")
	for _, p := range []string{"fixup! ", "squash! ", "amend! "} {
		if strings.HasPrefix(subject, p) {
			return ""
		}
	}
	if !commitSubject.MatchString(subject) {
		return fmt.Sprintf("Commit messages must look like `type(PROJECT): description`: type is one of feat, fix, chore, refactor, test, docs, build, ci, revert; PROJECT is the Jira project key (e.g. BBA) or no-ticket. Got %q.", subject)
	}
	first, count := c.firstBranchCommit(g.dir)
	if count == 0 || (o.has("--amend") && count == 1) {
		return ""
	}
	want := commitType.FindStringSubmatch(first)
	got := commitType.FindStringSubmatch(subject)
	if want != nil && want[1] != got[1] {
		return fmt.Sprintf("This branch's commits use type `%s` (first commit: %q). Keep the same type on every commit and the PR title; CI depends on it.", want[1], first)
	}
	return ""
}}

// firstBranchCommit returns the subject of the oldest commit on HEAD's
// first-parent chain since the default branch, and how many there are.
func (c *ctx) firstBranchCommit(dir string) (string, int) {
	for _, base := range []string{"origin/HEAD", "origin/main", "origin/master", "main", "master"} {
		out, ok := c.git(dir, "log", "--first-parent", "--reverse", "--format=%s", base+"..HEAD")
		if !ok {
			if c.lookupsOff || dir == "" {
				return "", 0
			}
			continue
		}
		out = strings.TrimSpace(out)
		if out == "" {
			return "", 0
		}
		lines := strings.Split(out, "\n")
		return lines[0], len(lines)
	}
	return "", 0
}
