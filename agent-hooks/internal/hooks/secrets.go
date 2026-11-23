package hooks

import (
	"fmt"
	"strings"

	"github.com/aryan-binazir/cc-config/agent-hooks/internal/shell"
)

// isSecret reports whether a file name looks like credentials.
func isSecret(p string) bool {
	b := baseName(p)
	switch {
	case b == ".env.example" || b == ".env.sample":
		return false
	case strings.HasPrefix(b, ".env"), strings.HasPrefix(b, "id_rsa"), strings.HasPrefix(b, "id_ed25519"), b == "settings.local.json":
		return true
	case strings.HasPrefix(b, "credentials") && strings.HasSuffix(b, ".json"):
		return true
	}
	for _, ext := range []string{".pem", ".key", ".p12", ".pfx"} {
		if strings.HasSuffix(b, ext) {
			return true
		}
	}
	return false
}

var addAliases = map[string]string{
	"--all": "-A", "--no-ignore-removal": "-A", "--update": "-u", "--force": "-f", "--dry-run": "-n",
	"--patch": "-p", "--interactive": "-i", "--edit": "-e",
}

// noSecretsCommit blocks staging files that look like credentials.
var noSecretsCommit = hook{name: "no-secrets-commit", phase: "pre", override: "secrets", check: func(c *ctx, inv shell.Invocation) string {
	g := parseGit(inv, c.opts.Home)
	if g == nil {
		return ""
	}
	var found []string
	switch g.sub {
	case "add":
		o := parseOpts(g.args, "", []string{"--chmod", "--pathspec-from-file"}, addAliases)
		if o.has("-n", "-p", "-i", "-e", "--pathspec-from-file") {
			return ""
		}
		specs := append(o.pos, o.paths...)
		found = c.explicitSecrets(g, specs)
		if o.has("-A", "-u") || broad(g.dir, specs, c.opts.Home) {
			found = append(found, c.wouldAdd(g)...)
		}
	case "commit":
		o := parseOpts(g.args, "mFCct", commitLongValues, commitAliases)
		specs := append(o.pos, o.paths...)
		found = c.explicitSecrets(g, specs)
		if o.has("-a") || broad(g.dir, specs, c.opts.Home) {
			found = append(found, c.secretChanges(g, withPaths([]string{"--untracked-files=no"}, specs), true)...)
		}
	default:
		return ""
	}
	if len(found) == 0 {
		return ""
	}
	return fmt.Sprintf("Don't stage credentials: %s. Unstage or .gitignore them. If the user explicitly asked to commit them, rerun with AGENT_HOOKS_ALLOW=secrets.", strings.Join(dedupe(found), ", "))
}}

// explicitSecrets lists named paths that look like credentials, except
// tracked files gone from disk: staging those records a deletion.
func (c *ctx) explicitSecrets(g *gitCmd, specs []shell.Word) []string {
	var found []string
	for _, s := range specs {
		if strings.ContainsAny(s.Val, "*?[") || !isSecret(s.Val) {
			continue
		}
		if full := shell.Resolve(g.dir, s.Val, c.opts.Home); full != "" && !exists(full) {
			if _, tracked := c.git(g, "ls-files", "--error-unmatch", "--", s.Val); tracked {
				continue
			}
		}
		found = append(found, s.Val)
	}
	return found
}

// wouldAdd asks `git add --dry-run` with the same arguments which files it
// would stage, so ignore rules, -f, -u, and pathspec magic match git exactly.
func (c *ctx) wouldAdd(g *gitCmd) []string {
	out, ok := c.git(g, append([]string{"add", "--dry-run"}, values(g.args)...)...)
	if !ok {
		return nil
	}
	var found []string
	for _, line := range strings.Split(out, "\n") {
		if p, ok := strings.CutPrefix(line, "add '"); ok && isSecret(strings.TrimSuffix(p, "'")) {
			found = append(found, strings.TrimSuffix(p, "'"))
		}
	}
	return found
}

// broad reports whether any pathspec can match files it does not name.
func broad(dir string, specs []shell.Word, home string) bool {
	for _, s := range specs {
		if s.Dynamic || strings.ContainsAny(s.Val, "*?[") || strings.HasPrefix(s.Val, ":") || isDir(dir, s.Val, home) {
			return true
		}
	}
	return false
}

// secretChanges lists changed, non-deleted files that look like credentials.
// tracked limits it to what `git commit -a` commits: staged and modified
// tracked files.
func (c *ctx) secretChanges(g *gitCmd, statusArgs []string, tracked bool) []string {
	entries, ok := c.status(g, statusArgs...)
	if !ok {
		return nil
	}
	var found []string
	for _, e := range entries {
		x, y := e.xy[0], e.xy[1]
		if y == 'D' || (x == 'D' && y == ' ') || (tracked && x == '?') {
			continue
		}
		if isSecret(e.path) {
			found = append(found, e.path)
		}
	}
	return found
}

func dedupe(list []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range list {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
