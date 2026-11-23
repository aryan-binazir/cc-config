package hooks

import (
	"fmt"
	"os"
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
	"--patch": "-p", "--interactive": "-i", "--edit": "-e",
}

var noSecretsCommit = hook{name: "no-secrets-commit", phase: "pre", override: "secrets", check: func(c *ctx, inv shell.Invocation) string {
	g := parseGit(inv, c.opts.Home)
	if g == nil {
		return ""
	}
	var found []string
	switch g.sub {
	case "add":
		o := parseOpts(g.args, "", []string{"--chmod", "--pathspec-from-file"}, addAliases)
		if o.has("-n", "-p", "-i", "-e") {
			return ""
		}
		specs := append(o.pos, o.paths...)
		found = c.explicitSecrets(g, specs)
		if o.has("-A", "-u", "--pathspec-from-file") || broad(g.dir, specs, c.opts.Home) {
			found = append(found, c.wouldAdd(g, o, specs)...)
		}
	case "commit":
		o := parseOpts(g.args, "mFCct", commitLongValues, commitAliases)
		specs := append(o.pos, o.paths...)
		found = c.explicitSecrets(g, specs)
		if o.has("-a") || broad(g.dir, specs, c.opts.Home) {
			found = append(found, c.secretChanges(g, withPaths([]string{"--untracked-files=no"}, specs))...)
		}
	default:
		return ""
	}
	if len(found) == 0 {
		return ""
	}
	return fmt.Sprintf("Don't stage credentials: %s. Unstage or .gitignore them. If the user explicitly asked to commit them, rerun with AGENT_HOOKS_ALLOW=secrets.", strings.Join(dedupe(found), ", "))
}}

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
	for _, file := range o.vals["--pathspec-from-file"] {
		paths = append(paths, c.pathspecFile(g, file, o.has("--pathspec-file-nul"))...)
	}
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
		out, _ := c.git(g, append([]string{"diff-files", "--name-only", "-z", "--relative", "--"}, tracked...)...)
		for _, p := range strings.Split(out, "\x00") {
			if p != "" && exists(shellPath(g.dir, p, c.opts.Home)) {
				found = append(found, p)
			}
		}
	}
	return found
}

func (c *ctx) pathspecFile(g *gitCmd, file shell.Word, nul bool) []string {
	full := shell.Resolve(g.dir, file.Val, c.opts.Home)
	if file.Dynamic || file.Val == "-" || full == "" {
		return nil
	}
	data, err := os.ReadFile(full)
	if err != nil || len(data) > 1<<20 {
		return nil
	}
	sep := "\n"
	if nul {
		sep = "\x00"
	}
	var out []string
	for _, line := range strings.Split(string(data), sep) {
		if line = strings.TrimRight(line, "\r"); line != "" {
			out = append(out, line)
		}
	}
	return out
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
