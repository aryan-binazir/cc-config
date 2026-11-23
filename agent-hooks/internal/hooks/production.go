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
		refs := append(append([]shell.Word{}, o.pos...), o.paths...)
		return len(refs) > 0 && isProduction(refs[0].Val)
	case "reset":
		o := parseOpts(g.args, "", nil, resetModes)
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
	if len(refspecs) == 0 {
		if c.currentBranch(g) == production {
			return true
		}
		if out, _ := c.git(g, "config", "--type=bool", "remote."+c.remoteFor(g, o)+".mirror"); strings.TrimSpace(out) == "true" {
			return true // a mirror remote mirrors (and deletes) every ref
		}
		refspecs = configuredRefspecs(c, g, o)
		if len(refspecs) == 0 {
			if out, _ := c.git(g, "config", "push.default"); strings.TrimSpace(out) == "matching" {
				refspecs = []string{":"}
			}
		}
	}
	if o.has("--all", "--branches") && c.hasLocalProduction(g) {
		return true
	}
	if o.has("--prune") && c.pruneDeletesProduction(g, o, refspecs) {
		return true
	}
	if len(refspecs) == 0 {
		out, ok := c.git(g, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{push}")
		_, branch, _ := strings.Cut(strings.TrimSpace(out), "/")
		return ok && branch == production
	}
	for _, spec := range refspecs {
		spec = strings.TrimPrefix(spec, "+")
		src, dst := spec, spec
		if s, d, ok := strings.Cut(spec, ":"); ok && !o.has("-d") {
			src, dst = s, d
		}
		switch {
		case src == "" && dst == "":
			// `:` pushes every branch that exists on both sides.
			if c.hasLocalProduction(g) {
				return true
			}
		case strings.Contains(dst, "*"):
			if from, covers := mapWildcard(src, dst); covers && c.refExists(g, from) {
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

// pushRemote returns the remote a push names, or "" when it names none.
func pushRemote(o opts) string {
	switch {
	case len(o.vals["--repo"]) > 0:
		return o.vals["--repo"][len(o.vals["--repo"])-1].Val
	case len(o.pos) > 0:
		return o.pos[0].Val
	case len(o.paths) > 0:
		return o.paths[0].Val
	}
	return ""
}

// remoteFor returns the remote a push goes to: the one named, else the current
// branch's push remote (branch.<name>.pushRemote, remote.pushDefault, or its
// upstream's remote), else origin.
func (c *ctx) remoteFor(g *gitCmd, o opts) string {
	if r := pushRemote(o); r != "" {
		return r
	}
	if branch := c.currentBranch(g); branch != "" {
		if out, _ := c.git(g, "for-each-ref", "--format=%(push:remotename)", "refs/heads/"+branch); strings.TrimSpace(out) != "" {
			return strings.TrimSpace(out)
		}
	}
	if out, _ := c.git(g, "remote"); len(strings.Fields(out)) == 1 {
		return strings.TrimSpace(out) // git pushes to a sole remote
	}
	return "origin"
}

// configuredRefspecs returns the remote's configured push refspecs, which
// git uses when the command names none.
func configuredRefspecs(c *ctx, g *gitCmd, o opts) []string {
	out, _ := c.git(g, "config", "--get-all", "remote."+c.remoteFor(g, o)+".push")
	return strings.Fields(out)
}

// mapWildcard reports whether a src:dst wildcard refspec's destination covers
// refs/heads/production, and the local ref it would push there.
func mapWildcard(src, dst string) (string, bool) {
	full := func(ref string) string {
		if strings.HasPrefix(ref, "refs/") {
			return ref
		}
		return "refs/heads/" + ref
	}
	target := "refs/heads/" + production
	prefix, suffix, _ := strings.Cut(full(dst), "*")
	if !strings.HasPrefix(target, prefix) || !strings.HasSuffix(target, suffix) || len(target) < len(prefix)+len(suffix) {
		return "", false
	}
	match := target[len(prefix) : len(target)-len(suffix)]
	return strings.Replace(full(src), "*", match, 1), true
}

// pruneDeletesProduction reports whether --prune could delete a remote
// production branch: a wildcard covers it with no local source, or a matching
// push (--all, `:`) runs with no local production branch. The remote's own
// branches are unknowable offline, so that case assumes it has one.
func (c *ctx) pruneDeletesProduction(g *gitCmd, o opts, refspecs []string) bool {
	matching := o.has("--all", "--branches")
	for _, spec := range refspecs {
		src, dst, ok := strings.Cut(strings.TrimPrefix(spec, "+"), ":")
		if !ok {
			dst = src
		}
		switch {
		case src == "" && dst == "":
			matching = true
		case strings.Contains(dst, "*"):
			if from, covers := mapWildcard(src, dst); covers && !c.refExists(g, from) {
				return true
			}
		}
	}
	return matching && !c.hasLocalProduction(g)
}

func (c *ctx) refExists(g *gitCmd, ref string) bool {
	_, ok := c.git(g, "rev-parse", "--verify", "-q", ref)
	return ok
}

func (c *ctx) hasLocalProduction(g *gitCmd) bool { return c.refExists(g, "refs/heads/"+production) }

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
