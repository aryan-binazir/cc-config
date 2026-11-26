package hooks

import (
	"fmt"
	"slices"
	"strings"

	"github.com/aryan-binazir/cc-config/agent-hooks/internal/shell"
)

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
	"--patch": "-p", "--interactive": "-i", "--edit": "-e", pathspecList: pathspecList,
}

var noSecretsCommit = hook{name: "no-secrets-commit", phase: "pre", override: "secrets", check: func(c *ctx, inv shell.Invocation) string {
	g := c.parseGit(inv)
	if g == nil {
		return ""
	}
	var found []string
	switch g.sub {
	case "add":
		o := parseOpts(g.args, "", []string{"--chmod", pathspecList}, addAliases)
		if o.has("-n", "-p", "-i", "-e") {
			return ""
		}
		specs := o.listed(append(o.pos, o.paths...))
		found = c.explicitSecrets(g, specs)
		if o.has("-A", "-u") || broad(g.dir, specs, c.opts.Home) {
			found = append(found, c.wouldAdd(g, o, specs)...)
		}
	case "commit":
		o := parseOpts(g.args, "mFCct", commitLongValues, commitAliases)
		if o.has("--dry-run", "--short", "--porcelain", "--long", "-z", "-h", "--help") {
			return ""
		}
		specs := o.listed(append(o.pos, o.paths...))
		found = c.explicitSecrets(g, specs)
		if o.has("-a") || broad(g.dir, specs, c.opts.Home) {
			found = append(found, c.secretChanges(g, withPaths([]string{"--untracked-files=no"}, specs))...)
		}
		if commitsTheIndex(o, specs) {
			staged := c.stagedSecrets(g)
			if len(staged) > 0 && len(specs) > 0 && o.has("-i") && !o.has(pathspecList) { // --include first stages the deletion of a named file that is gone
				gone, _ := c.git(g, append([]string{"diff-files", "--name-only", "-z", "--diff-filter=D", "--no-relative", "--"}, values(specs)...)...)
				staged = slices.DeleteFunc(staged, func(p string) bool { return slices.Contains(strings.Split(gone, "\x00"), p) })
			}
			found = append(found, staged...)
		}
	default:
		return ""
	}
	if len(found) == 0 {
		return ""
	}
	return fmt.Sprintf("Don't stage credentials: %s. Unstage or .gitignore them. If the user explicitly asked to commit them, rerun with AGENT_HOOKS_ALLOW=secrets.", strings.Join(dedupe(found), ", "))
}}

// commitsTheIndex reports whether a commit records what is already staged.
// With pathspecs git commits only those paths, unless --include, --patch, or
// --interactive adds them to the index instead, or they come from a list,
// which may be empty. -a restages every tracked file, which the status check
// covers, and --only --amend and --fixup=reword: rewrite a commit without
// reading the index.
func commitsTheIndex(o opts, specs []shell.Word) bool {
	for _, fixup := range o.vals["--fixup"] {
		if strings.HasPrefix(fixup.Val, "reword:") {
			return false
		}
	}
	if o.has("-a") || (o.has("-o") && o.has("--amend")) {
		return false
	}
	return len(specs) == 0 || o.has("-i", "-p", "--interactive", pathspecList)
}

func (c *ctx) stagedSecrets(g *gitCmd) []string {
	out, _ := c.git(g, "diff", "--cached", "--name-only", "-z", "--diff-filter=d", "--no-renames", "--no-relative", "--ita-invisible-in-index")
	var found []string
	for _, p := range strings.Split(out, "\x00") {
		if p != "" && isSecret(p) {
			found = append(found, p)
		}
	}
	return found
}

func (c *ctx) explicitSecrets(g *gitCmd, specs []shell.Word) []string {
	var found []string
	for _, s := range specs {
		if strings.ContainsAny(s.Val, "*?[") || !isSecret(s.Val) {
			continue
		}
		if full := shell.Resolve(g.dir, s.Val, c.opts.Home); full != "" && !exists(full) {
			if _, stagesDeletion := c.git(g, "ls-files", "--error-unmatch", "--", s.Val); stagesDeletion {
				continue
			}
		}
		found = append(found, s.Val)
	}
	return found
}

func (c *ctx) wouldAdd(g *gitCmd, o opts, specs []shell.Word) []string {
	paths := values(specs)
	if len(paths) == 0 {
		paths = []string{":/"} // -A and -u cover the whole tree
	}
	var found []string
	keep := func(out string) {
		for _, p := range strings.Split(out, "\x00") {
			if p != "" && isSecret(p) {
				found = append(found, p)
			}
		}
	}
	if !o.has("-u") {
		out, _ := c.git(g, append([]string{"ls-files", "-z", "--others", "--exclude-standard", "--"}, paths...)...)
		keep(out)
		if o.has("-f") {
			out, _ := c.git(g, append([]string{"ls-files", "-z", "--others", "--ignored", "--exclude-standard", "--"}, paths...)...)
			keep(out)
		}
	}
	out, _ := c.git(g, append([]string{"ls-files", "-z", "--"}, paths...)...)
	var tracked []string
	for _, p := range strings.Split(out, "\x00") {
		if p != "" && isSecret(p) {
			tracked = append(tracked, p)
		}
	}
	if len(tracked) > 0 {
		// Changed and still there: staging a secret's deletion is fine.
		out, _ := c.git(g, append([]string{"diff-files", "--name-only", "-z", "--diff-filter=d", "--no-relative", "--"}, tracked...)...)
		keep(out)
	}
	return found
}

func broad(dir string, specs []shell.Word, home string) bool {
	for _, s := range specs {
		if s.Dynamic || strings.ContainsAny(s.Val, "*?[") || strings.HasPrefix(s.Val, ":") || isDir(dir, s.Val, home) {
			return true
		}
	}
	return false
}

func (c *ctx) secretChanges(g *gitCmd, statusArgs []string) []string {
	entries, ok := c.status(g, statusArgs...)
	if !ok {
		return nil
	}
	var found []string
	for _, e := range entries {
		x, y := e.xy[0], e.xy[1]
		if y == 'D' || (x == 'D' && y == ' ') || x == '?' {
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
