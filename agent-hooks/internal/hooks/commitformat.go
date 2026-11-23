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

var commitFormat = hook{name: "commit-format", phase: "pre", check: func(c *ctx, inv shell.Invocation) string {
	g := parseGit(inv, c.opts.Home)
	if g == nil || g.sub != "commit" {
		return ""
	}
	o := parseOpts(g.args, "mFCct", commitLongValues, commitAliases)
	if o.has("--fixup", "--squash", "-C", "-c", "--dry-run") {
		return ""
	}
	var message string
	dynamic := false
	switch {
	case len(o.vals["-m"]) > 0:
		w := o.vals["-m"][0]
		message, dynamic = w.Val, w.Dynamic
	case len(o.vals["-F"]) > 0 && o.vals["-F"][0].Val == "-" && inv.Stdin != nil:
		message, dynamic = *inv.Stdin, inv.StdinDynamic
	default:
		return ""
	}
	subject, _, _ := strings.Cut(strings.TrimLeft(message, " \t\n"), "\n")
	subject = strings.TrimRight(subject, " \t\r")
	if dynamic && strings.ContainsAny(subject, "$`") {
		return ""
	}
	for _, p := range []string{"fixup! ", "squash! ", "amend! "} {
		if strings.HasPrefix(subject, p) {
			return ""
		}
	}
	if !commitSubject.MatchString(subject) {
		return fmt.Sprintf("Commit messages must look like `type(PROJECT): description`: type is one of feat, fix, chore, refactor, test, docs, build, ci, revert; PROJECT is the Jira project key (e.g. BBA) or no-ticket. Got %q.", subject)
	}
	first, count := c.firstBranchCommit(g)
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

func (c *ctx) firstBranchCommit(g *gitCmd) (subject string, count int) {
	for _, base := range []string{"origin/HEAD", "origin/main", "origin/master", "main", "master"} {
		out, ok := c.git(g, "log", "--first-parent", "--reverse", "--format=%s", base+"..HEAD")
		if !ok {
			if c.blocked || g.dir == "" {
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
