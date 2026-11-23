package hooks

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/aryan-binazir/cc-config/agent-hooks/internal/shell"
)

var pushAliases = map[string]string{
	"--force": "-f", "--dry-run": "-n", "--delete": "-d", "--set-upstream": "-u", "--push-option": "-o",
}
var pushLongValues = []string{"--repo", "--receive-pack", "--exec", "-o"}

const discardAdvice = "Commit or stash them first; if the user asked to throw them away, rerun with AGENT_HOOKS_ALLOW=destructive."

// destructiveGit blocks force pushes, rm -rf of /, home, or a repository
// root, and commands that discard uncommitted work.
var destructiveGit = hook{name: "destructive-git", phase: "pre", override: "destructive", check: func(c *ctx, inv shell.Invocation) string {
	if inv.Name() == "rm" {
		return checkRm(c, inv)
	}
	g := parseGit(inv, c.opts.Home)
	if g == nil {
		return ""
	}
	switch g.sub {
	case "push":
		o := parseOpts(g.args, "o", pushLongValues, pushAliases)
		if o.has("-n") {
			return ""
		}
		if o.has("-f", "--mirror") {
			return "Don't force-push: use --force-with-lease (without --force, which overrides it) so you cannot overwrite commits you haven't seen. If the user asked for a force push, rerun with AGENT_HOOKS_ALLOW=destructive."
		}
		if !o.has("--force-with-lease") {
			for _, refspec := range o.pos[min(1, len(o.pos)):] {
				if strings.HasPrefix(refspec.Val, "+") {
					return "A +refspec force-pushes. Use --force-with-lease instead, or rerun with AGENT_HOOKS_ALLOW=destructive if the user asked for it."
				}
			}
		}
	case "clean":
		o := parseOpts(g.args, "e", []string{"--exclude"}, map[string]string{"--force": "-f", "--dry-run": "-n", "--interactive": "-i"})
		if !o.has("-f") || o.has("-n", "-i") {
			return ""
		}
		args := []string{"--untracked-files=all"}
		if o.has("-x", "-X") {
			args = append(args, "--ignored=matching")
		}
		entries, ok := c.status(g.dir, withPaths(args, append(o.pos, o.paths...))...)
		for _, e := range entries {
			if ok && (e.xy == "??" && !o.has("-X") || e.xy == "!!" && o.has("-x", "-X")) {
				return "git clean would delete untracked files (" + e.path + ", ...). " + discardAdvice
			}
		}
	case "reset":
		if o := parseOpts(g.args, "", nil, nil); o.has("--hard") && c.dirty(g.dir, nil, false) {
			return "git reset --hard would discard uncommitted changes. " + discardAdvice
		}
	case "checkout", "restore":
		o := parseOpts(g.args, "s", []string{"--source"}, map[string]string{"--staged": "-S", "--worktree": "-W"})
		if g.sub == "restore" && o.has("-S") && !o.has("-W") {
			return ""
		}
		specs := o.paths
		if g.sub == "restore" || !o.dashed {
			specs = append(append([]shell.Word{}, o.pos...), o.paths...)
		}
		if !hasAll(specs) {
			return ""
		}
		fromCommit := o.has("-s", "--source") || (g.sub == "checkout" && o.dashed && len(o.pos) > 0)
		if c.dirty(g.dir, specs, !fromCommit) {
			return fmt.Sprintf("git %s would discard uncommitted changes. %s", g.sub, discardAdvice)
		}
	}
	return ""
}}

// hasAll reports whether the pathspecs cover the whole tree or the current directory.
func hasAll(specs []shell.Word) bool {
	for _, s := range specs {
		if v := strings.TrimSuffix(s.Val, "/"); v == "." || v == ":/" || v == ":" || v == "*" {
			return true
		}
	}
	return false
}

// dirty reports tracked changes; worktreeOnly ignores changes already staged.
func (c *ctx) dirty(dir string, specs []shell.Word, worktreeOnly bool) bool {
	entries, ok := c.status(dir, withPaths([]string{"--untracked-files=no"}, specs)...)
	if !ok {
		return false
	}
	for _, e := range entries {
		if !worktreeOnly || e.xy[1] != ' ' {
			return true
		}
	}
	return false
}

var homeVar = regexp.MustCompile(`^\$(HOME|\{HOME\})/?(\*)?$`)

func checkRm(c *ctx, inv shell.Invocation) string {
	o := parseOpts(inv.Args[1:], "", nil, map[string]string{"--recursive": "-r", "-R": "-r", "--force": "-f"})
	if !o.has("-r") || !o.has("-f") {
		return ""
	}
	for _, target := range append(o.pos, o.paths...) {
		if dangerousTarget(c, inv.Cwd, target) {
			return fmt.Sprintf("Refusing rm -rf %s: it is /, your home directory, or a git repository root. Delete specific paths instead; if the user asked for this, rerun with AGENT_HOOKS_ALLOW=destructive.", target.Val)
		}
	}
	return ""
}

func dangerousTarget(c *ctx, cwd string, target shell.Word) bool {
	v := target.Val
	if homeVar.MatchString(v) {
		return true
	}
	if target.Dynamic {
		return false
	}
	v = strings.TrimSuffix(v, "/*")
	if v == "" || v == "~" || v == "~/" {
		return true
	}
	full := shell.Resolve(cwd, v, c.opts.Home)
	if full == "" {
		return false
	}
	full = path.Clean(full)
	return full == "/" || (c.opts.Home != "" && full == path.Clean(c.opts.Home)) || exists(path.Join(full, ".git"))
}
