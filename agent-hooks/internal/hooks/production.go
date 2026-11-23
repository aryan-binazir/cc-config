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

func isProduction(ref string) bool {
	return strings.TrimPrefix(ref, "refs/heads/") == production
}

func touchesProduction(c *ctx, inv shell.Invocation) bool {
	if pr := parseGhPr(inv); pr != nil {
		return pr.action == "merge" && !pr.flags["--disable-auto"] && prTouchesProduction(c, inv, pr)
	}
	if api := parseGhAPI(inv); api != nil {
		return api.method != "GET" && productionRef.MatchString(api.endpoint)
	}
	if words, rest := ghWords(inv); len(words) == 2 && words[0] == "repo" && words[1] == "sync" {
		o := parseOpts(wordsOf(rest), "bs", []string{"-b", "-s"}, map[string]string{"--branch": "-b", "--source": "-s"})
		return names(o.vals["-b"], production)
	}
	g := parseGit(inv, c.opts.Home)
	if g == nil {
		return false
	}
	switch g.sub {
	case "push":
		return pushesProduction(c, g)
	case "fetch":
		o := parseOpts(g.args, "oj", []string{"--upload-pack", "--depth", "--deepen", "--shallow-since", "--shallow-exclude", "--refmap", "--recurse-submodules-default", "--server-option", "--negotiation-tip", "-o", "-j"}, nil)
		for _, r := range append(o.pos[min(1, len(o.pos)):], o.paths...) {
			if _, dst, ok := strings.Cut(strings.TrimPrefix(r.Val, "+"), ":"); ok && isProduction(dst) {
				return true
			}
		}
	case "branch":
		o := parseOpts(g.args, "", []string{"--set-upstream-to"}, map[string]string{"--delete": "-d", "--force": "-f", "--move": "-m", "--copy": "-c"})
		refs := append(append([]shell.Word{}, o.pos...), o.paths...)
		switch {
		case o.has("-d", "-D", "-f"):
			return names(refs, production)
		case o.has("-m", "-M"):
			// Renaming deletes the old name: production as either operand, or the
			// current branch when only the new name is given.
			return names(refs, production) || (len(refs) == 1 && c.currentBranch(g) == production)
		case o.has("-c", "-C"):
			return len(refs) > 0 && refs[len(refs)-1].Val == production
		}
	case "checkout", "switch":
		o := parseOpts(g.args, "BCbc", []string{"-C"}, map[string]string{"--force-create": "-C"})
		return names(o.vals["-B"], production) || names(o.vals["-C"], production)
	case "update-ref":
		o := parseOpts(g.args, "m", nil, nil)
		return len(o.pos) > 0 && isProduction(o.pos[0].Val)
	case "reset":
		o := parseOpts(g.args, "", nil, nil)
		moves := o.has("--hard", "--soft", "--keep", "--merge", "--mixed")
		for _, p := range o.pos {
			if !exists(shellPath(g.dir, p.Val, c.opts.Home)) {
				moves = true
			}
		}
		return moves && c.currentBranch(g) == production
	case "merge", "pull", "rebase":
		o := parseOpts(g.args, "sXm", []string{"--strategy", "--strategy-option", "--onto"}, nil)
		if o.has("--abort", "--continue", "--skip", "--quit") {
			return false
		}
		if g.sub == "rebase" && len(o.pos) == 2 && o.pos[1].Val == production {
			return true
		}
		return c.currentBranch(g) == production
	}
	return false
}

// prTouchesProduction looks up the PR's base and head: merging into production,
// or deleting a production head with --delete-branch.
func prTouchesProduction(c *ctx, inv shell.Invocation, pr *ghPr) bool {
	args := []string{"pr", "view"}
	args = append(args, pr.positionals...)
	for _, repo := range pr.values["--repo"] {
		args = append(args, "--repo", repo)
	}
	var env []string
	for _, name := range []string{"GH_REPO", "GH_HOST"} {
		if v, ok := inv.Assigns[name]; ok {
			env = append(env, name+"="+v)
		}
	}
	out, ok := c.gh(inv.Cwd, env, append(args, "--json", "baseRefName,headRefName", "-q", `.baseRefName + " " + .headRefName`)...)
	fields := strings.Fields(out)
	if !ok || len(fields) != 2 {
		return false
	}
	return fields[0] == production || (fields[1] == production && (pr.flags["--delete-branch"] || pr.flags["-d"]))
}

func pushesProduction(c *ctx, g *gitCmd) bool {
	o := parseOpts(g.args, "o", pushLongValues, pushAliases)
	if o.has("-n") {
		return false
	}
	if o.has("--mirror") {
		return true // mirroring can delete a remote production branch
	}
	refspecs := pushRefspecs(o)
	if o.has("--all", "--branches") && c.hasLocalProduction(g) {
		return true
	}
	if len(refspecs) == 0 {
		if c.currentBranch(g) == production {
			return true
		}
		refspecs = configuredRefspecs(c, g, o)
		if len(refspecs) == 0 {
			out, ok := c.git(g, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{push}")
			_, branch, _ := strings.Cut(strings.TrimSpace(out), "/")
			return ok && branch == production
		}
	}
	for _, spec := range refspecs {
		spec = strings.TrimPrefix(spec, "+")
		dst := spec
		if _, d, ok := strings.Cut(spec, ":"); ok && !o.has("-d") {
			dst = d
		}
		switch {
		case dst == "" || strings.Contains(dst, "*"):
			// `:` pushes every matching branch; a wildcard maps many.
			if c.hasLocalProduction(g) {
				return true
			}
		case dst == "HEAD" || dst == "@":
			if c.currentBranch(g) == production {
				return true
			}
		case isProduction(dst):
			return true
		}
	}
	return false
}

// pushRefspecs returns the refspecs after the remote operand.
func pushRefspecs(o opts) []string {
	if len(o.pos) == 0 && len(o.paths) > 0 {
		return values(o.paths[1:]) // git push -- <remote> <refspec>...
	}
	return values(append(append([]shell.Word{}, o.pos[min(1, len(o.pos)):]...), o.paths...))
}

// configuredRefspecs returns the remote's configured push refspecs, which
// git uses when the command names none.
func configuredRefspecs(c *ctx, g *gitCmd, o opts) []string {
	remote := "origin"
	if len(o.pos) > 0 {
		remote = o.pos[0].Val
	}
	out, _ := c.git(g, "config", "--get-all", "remote."+remote+".push")
	return strings.Fields(out)
}

func (c *ctx) hasLocalProduction(g *gitCmd) bool {
	_, ok := c.git(g, "rev-parse", "--verify", "-q", "refs/heads/"+production)
	return ok
}

func names(words []shell.Word, name string) bool {
	for _, w := range words {
		if w.Val == name {
			return true
		}
	}
	return false
}

func wordsOf(vals []string) []shell.Word {
	out := make([]shell.Word, len(vals))
	for i, v := range vals {
		out[i] = shell.Word{Val: v}
	}
	return out
}

func shellPath(dir, p, home string) string {
	if full := shell.Resolve(dir, p, home); full != "" {
		return full
	}
	return p
}
