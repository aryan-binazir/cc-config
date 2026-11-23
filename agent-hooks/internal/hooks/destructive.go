package hooks

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/aryan-binazir/cc-config/agent-hooks/internal/shell"
)

// Identity entries let git's unique-prefix abbreviations (--mir) resolve.
var pushAliases = map[string]string{
	"--force": "-f", "--dry-run": "-n", "--delete": "-d", "--set-upstream": "-u", "--push-option": "-o",
	"--mirror": "--mirror", "--all": "--all", "--branches": "--branches", "--prune": "--prune",
	"--force-with-lease": "--force-with-lease", "--repo": "--repo", "--tags": "--tags",
}

var resetModes = map[string]string{"--hard": "--hard", "--soft": "--soft", "--mixed": "--mixed", "--keep": "--keep", "--merge": "--merge"}
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
		refspecs := pushRefspecs(o)
		if len(refspecs) == 0 {
			refspecs = configuredRefspecs(c, g, o)
		}
		for _, refspec := range refspecs {
			// A + forces that ref even when a lease is given for another.
			if strings.HasPrefix(refspec, "+") {
				return "A +refspec force-pushes. Push without the + and use --force-with-lease, or rerun with AGENT_HOOKS_ALLOW=destructive if the user asked for it."
			}
		}
	case "clean":
		o := parseOpts(g.args, "e", []string{"-e"}, map[string]string{"--force": "-f", "--dry-run": "-n", "--interactive": "-i", "--exclude": "-e"})
		if o.has("-n", "-i") {
			return ""
		}
		if !o.has("-f") && !requireForceOff(c, g) {
			return "" // git refuses to clean without -f
		}
		// git's own dry run knows its excludes, -d, and nested repositories.
		out, ok := c.git(g, cleanDryRun(o)...)
		if first, _, _ := strings.Cut(strings.TrimSpace(out), "\n"); ok && first != "" {
			return "git clean would delete untracked files (" + strings.TrimPrefix(first, "Would remove ") + ", ...). " + discardAdvice
		}
	case "reset":
		o := parseOpts(g.args, "", nil, resetModes)
		if !o.has("--hard") {
			return ""
		}
		if c.dirty(g, nil, false) {
			return "git reset --hard would discard uncommitted changes. " + discardAdvice
		}
		if len(o.pos) > 0 && c.overwritesUntracked(g, o.pos[0].Val) {
			return "git reset --hard " + o.pos[0].Val + " would overwrite untracked files that the target tracks. " + discardAdvice
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
		// Restoring from a commit also discards staged changes.
		fromCommit := o.has("-s", "--source") || (g.sub == "checkout" && len(o.pos) > 0 && (o.dashed || !hasAll(o.pos[:1])))
		if c.dirty(g, specs, !fromCommit) {
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
func (c *ctx) dirty(g *gitCmd, specs []shell.Word, worktreeOnly bool) bool {
	entries, ok := c.status(g, withPaths([]string{"--untracked-files=no"}, specs)...)
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

// cleanDryRun builds `git clean -n` from the parsed flags only, never the
// agent's own arguments, so the lookup cannot delete anything.
func cleanDryRun(o opts) []string {
	args := []string{"clean", "-n"}
	if o.has("-d") {
		args = append(args, "-d")
	}
	if o.set["-f"] > 1 {
		args = append(args, "-f", "-f") // -ff also removes nested repositories
	}
	for _, flag := range []string{"-x", "-X"} {
		if o.has(flag) {
			args = append(args, flag)
		}
	}
	for _, e := range o.vals["-e"] {
		args = append(args, "-e", e.Val)
	}
	if specs := append(o.pos, o.paths...); len(specs) > 0 {
		args = append(append(args, "--"), values(specs)...)
	}
	return args
}

func requireForceOff(c *ctx, g *gitCmd) bool {
	if v, ok := g.config["clean.requireforce"]; ok {
		switch strings.ToLower(v) {
		case "false", "no", "off", "0":
			return true
		}
		return false
	}
	out, ok := c.git(g, "config", "--type=bool", "--get", "clean.requireForce")
	return ok && strings.TrimSpace(out) == "false"
}

// overwritesUntracked reports whether resetting to target would replace an
// untracked or ignored file, or a directory holding one, with something target
// tracks.
func (c *ctx) overwritesUntracked(g *gitCmd, target string) bool {
	entries, ok := c.status(g, "--untracked-files=all", "--ignored=matching")
	if !ok {
		return false
	}
	var local []string
	for _, e := range entries {
		if e.xy == "??" || e.xy == "!!" {
			local = append(local, e.path)
		}
	}
	if len(local) == 0 {
		return false
	}
	out, ok := c.git(g, "ls-tree", "-r", "-z", "--full-tree", "--name-only", target)
	if !ok {
		return false
	}
	tracked := map[string]bool{}
	for _, p := range strings.Split(out, "\x00") {
		tracked[p] = true
	}
	// Porcelain paths are relative to the repository root, like --full-tree.
	// A collision is a shared path, or one side's path under the other's:
	// an untracked file inside a tracked file's spot, or the reverse.
	localSet := map[string]bool{}
	for _, p := range local {
		localSet[strings.TrimSuffix(p, "/")] = true // an ignored directory is listed as dir/
		for q := strings.TrimSuffix(p, "/"); q != "." && q != ""; q = path.Dir(q) {
			if tracked[q] {
				return true
			}
		}
	}
	for t := range tracked {
		for q := t; q != "." && q != ""; q = path.Dir(q) {
			if localSet[q] {
				return true
			}
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
	if target.Dynamic {
		return homeVar.MatchString(v) // only a real expansion of $HOME
	}
	v = strings.TrimSuffix(v, "/*")
	if strings.HasPrefix(v, "~") && target.Quoted {
		v = "./" + v // a quoted ~ is a literal name, not home
	}
	if v == "" || v == "~" || v == "~/" {
		return true
	}
	full := shell.Resolve(cwd, v, c.opts.Home)
	if full == "" {
		return false
	}
	full = path.Clean(full)
	bare := exists(path.Join(full, "HEAD")) && exists(path.Join(full, "objects")) && exists(path.Join(full, "refs"))
	return full == "/" || (c.opts.Home != "" && full == path.Clean(c.opts.Home)) || exists(path.Join(full, ".git")) || bare
}
