package hooks

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/aryan-binazir/cc-config/agent-hooks/internal/shell"
)

var commitSubject = regexp.MustCompile(`^(feat|fix|chore|refactor|test|docs|build|ci|revert)\((no-ticket|[A-Z][A-Z0-9]+)\): \S`)
var commitType = regexp.MustCompile(`^(feat|fix|chore|refactor|test|docs|build|ci|revert)\(`)

var commitAliases = map[string]string{
	"--message": "-m", "--file": "-F", "--reuse-message": "-C", "--reedit-message": "-c", "--template": "-t", "--all": "-a",
	"--include": "-i", "--only": "-o", "--patch": "-p", "--null": "-z",
	"--dry-run": "--dry-run", "--short": "--short", "--porcelain": "--porcelain", "--long": "--long", "--help": "--help",
	pathspecList: pathspecList,
}
var commitLongValues = []string{"-m", "-F", "-C", "-c", "-t", "--fixup", "--squash", "--author", "--date", "--trailer", "--cleanup", pathspecList}

var commitFormat = hook{name: "commit-format", phase: "pre", override: "commit-format", check: func(c *ctx, inv shell.Invocation) string {
	if pr := parseGhPr(inv); pr != nil {
		return prTitleProblem(c, inv, pr)
	}
	g := c.parseGit(inv)
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
	return c.subjectProblem(g, "Commit messages", subject, o.has("--amend"))
}}

func prTitleProblem(c *ctx, inv shell.Invocation, pr *ghPr) string {
	titles := pr.values["--title"]
	_, otherRepo := inv.Assigns["GH_REPO"]
	otherRepo = otherRepo || os.Getenv("GH_REPO") != "" || len(pr.values["--repo"]) > 0 ||
		slices.ContainsFunc(pr.positionals, func(w shell.Word) bool { return strings.Contains(w.Val, "://") })
	if pr.action == "merge" || len(titles) == 0 || otherRepo {
		return ""
	}
	title := titles[len(titles)-1]
	if title.Val == "" || (title.Dynamic && strings.ContainsAny(title.Val, "$`")) {
		return ""
	}
	subject := strings.TrimSpace(title.Val)
	if formatProblem := c.subjectProblem(nil, "PR titles", subject, false); formatProblem != "" {
		return formatProblem
	}
	var branch *gitCmd
	forCurrentBranch := len(pr.positionals) == 0 && len(pr.values["--head"]) == 0
	if forCurrentBranch {
		branch = &gitCmd{dir: inv.Cwd}
	}
	var bases []string
	if given := pr.values["--base"]; len(given) > 0 {
		base := given[len(given)-1]
		if base.Dynamic {
			branch = nil
		}
		bases = []string{"origin/" + base.Val, base.Val}
	} else if branch != nil && pr.action == "create" {
		// Without --base, gh opens the pull request against this setting when
		// the branch has it. One that is already open keeps its base.
		if base, ok := c.git(branch, "config", "--get", "branch."+c.currentBranch(branch)+".gh-merge-base"); ok {
			bases = []string{"origin/" + strings.TrimSpace(base), strings.TrimSpace(base)}
		}
	}
	return c.subjectProblem(branch, "PR titles", subject, false, bases...)
}

func (c *ctx) subjectProblem(g *gitCmd, what, subject string, amending bool, bases ...string) string {
	if !commitSubject.MatchString(subject) {
		return fmt.Sprintf("%s must look like `type(PROJECT): description`: type is one of feat, fix, chore, refactor, test, docs, build, ci, revert; PROJECT is the Jira project key (e.g. BBA) or no-ticket. Got %q. If this repository uses another convention, rerun with AGENT_HOOKS_ALLOW=commit-format.", what, subject)
	}
	if g == nil {
		return ""
	}
	first, count := c.firstBranchCommit(g, bases...)
	if count == 0 || (amending && count == 1) {
		return ""
	}
	want := commitType.FindStringSubmatch(first)
	got := commitType.FindStringSubmatch(subject)
	if want != nil && want[1] != got[1] {
		return fmt.Sprintf("This branch's commits use type `%s` (first commit: %q). Keep the same type on every commit and the PR title; CI depends on it. If this repository uses another convention, rerun with AGENT_HOOKS_ALLOW=commit-format.", want[1], first)
	}
	return ""
}

func (c *ctx) firstBranchCommit(g *gitCmd, bases ...string) (subject string, count int) {
	if len(bases) == 0 {
		bases = []string{"origin/HEAD", "origin/main", "origin/master", "main", "master"}
	}
	for _, base := range bases {
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
