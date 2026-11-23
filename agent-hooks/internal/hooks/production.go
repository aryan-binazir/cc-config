package hooks

import (
	"regexp"
	"strings"

	"github.com/aryan-binazir/cc-config/agent-hooks/internal/shell"
)

const production = "production"

// noProduction blocks pushing to, merging into, force-updating, resetting, or
// deleting a branch named production unless the user asked for it.
var noProduction = hook{name: "no-production", phase: "pre", override: "production", check: func(c *ctx, inv shell.Invocation) string {
	if touchesProduction(c, inv) {
		return "Only push to production if the user, or an agent acting on the user's instruction, told you to. If so, rerun with AGENT_HOOKS_ALLOW=production. When delegating, pass the user's instruction along."
	}
	return ""
}}

var productionRef = regexp.MustCompile(`(^|/)git/refs/heads/production/?$`)

func touchesProduction(c *ctx, inv shell.Invocation) bool {
	if pr := parseGhPr(inv); pr != nil {
		if pr.action != "merge" || pr.flags["--disable-auto"] {
			return false
		}
		args := []string{"pr", "view"}
		args = append(args, pr.positionals...)
		for _, repo := range pr.values["--repo"] {
			args = append(args, "--repo", repo)
		}
		out, ok := c.gh(inv.Cwd, append(args, "--json", "baseRefName", "-q", ".baseRefName")...)
		return ok && strings.TrimSpace(out) == production
	}
	if api := parseGhAPI(inv); api != nil {
		return api.method != "GET" && productionRef.MatchString(api.endpoint)
	}
	g := parseGit(inv, c.opts.Home)
	if g == nil {
		return false
	}
	switch g.sub {
	case "push":
		return pushesProduction(c, g)
	case "branch":
		o := parseOpts(g.args, "", []string{"--set-upstream-to"}, map[string]string{"--delete": "-d", "--force": "-f", "--move": "-m", "--copy": "-c"})
		return o.has("-d", "-D", "-f", "-m", "-M", "-c", "-C") && names(o.pos, production)
	case "checkout", "switch":
		o := parseOpts(g.args, "BCbc", []string{"-C"}, map[string]string{"--force-create": "-C"})
		return names(o.vals["-B"], production) || names(o.vals["-C"], production)
	case "update-ref":
		o := parseOpts(g.args, "m", nil, nil)
		return len(o.pos) > 0 && strings.TrimPrefix(o.pos[0].Val, "refs/heads/") == production
	case "reset":
		o := parseOpts(g.args, "", nil, nil)
		moves := o.has("--hard", "--soft", "--keep", "--merge", "--mixed")
		for _, p := range o.pos {
			if !exists(shellPath(g.dir, p.Val, c.opts.Home)) {
				moves = true
			}
		}
		return moves && c.currentBranch(g.dir) == production
	case "merge", "pull", "rebase":
		o := parseOpts(g.args, "sXm", []string{"--strategy", "--strategy-option", "--onto"}, nil)
		if g.sub == "rebase" && len(o.pos) == 2 && o.pos[1].Val == production {
			return true
		}
		return c.currentBranch(g.dir) == production
	}
	return false
}

func pushesProduction(c *ctx, g *gitCmd) bool {
	o := parseOpts(g.args, "o", pushLongValues, pushAliases)
	if o.has("-n") {
		return false
	}
	if o.has("--all", "--mirror", "--branches") {
		_, ok := c.git(g.dir, "rev-parse", "--verify", "-q", "refs/heads/"+production)
		return ok
	}
	refspecs := append(o.pos[min(1, len(o.pos)):], o.paths...)
	if len(refspecs) == 0 {
		if c.currentBranch(g.dir) == production {
			return true
		}
		out, ok := c.git(g.dir, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{push}")
		_, branch, _ := strings.Cut(strings.TrimSpace(out), "/")
		return ok && branch == production
	}
	for _, r := range refspecs {
		spec := strings.TrimPrefix(r.Val, "+")
		dst := spec
		if _, d, ok := strings.Cut(spec, ":"); ok && !o.has("-d") {
			dst = d
		}
		if dst == "HEAD" || dst == "@" {
			dst = c.currentBranch(g.dir)
		}
		if strings.TrimPrefix(dst, "refs/heads/") == production {
			return true
		}
	}
	return false
}

func names(words []shell.Word, name string) bool {
	for _, w := range words {
		if w.Val == name {
			return true
		}
	}
	return false
}

func shellPath(dir, p, home string) string {
	if full := shell.Resolve(dir, p, home); full != "" {
		return full
	}
	return p
}
