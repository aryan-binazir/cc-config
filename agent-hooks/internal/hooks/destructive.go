package hooks

import (
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aryan-binazir/cc-config/agent-hooks/internal/shell"
)

var pushLongNames = map[string]string{
	"--force": "-f", "--dry-run": "-n", "--delete": "-d", "--set-upstream": "-u", "--push-option": "-o",
	"--mirror": "--mirror", "--all": "--all", "--branches": "--branches", "--prune": "--prune",
	"--force-with-lease": "--force-with-lease", "--repo": "--repo", "--tags": "--tags",
}

var resetModes = map[string]string{"--hard": "--hard", "--soft": "--soft", "--mixed": "--mixed", "--keep": "--keep", "--merge": "--merge"}
var pushLongValues = []string{"--repo", "--receive-pack", "--exec", "-o"}

const discardAdvice = "Commit or stash them first; if the user asked to throw them away, rerun with AGENT_HOOKS_ALLOW=destructive."

var destructiveGit = hook{name: "destructive-git", phase: "pre", override: "destructive", check: func(c *ctx, inv shell.Invocation) string {
	if inv.Name() == "rm" {
		return checkRm(c, inv)
	}
	g := c.parseGit(inv)
	if g == nil {
		return ""
	}
	switch g.sub {
	case "push":
		o := parseOpts(g.args, "o", pushLongValues, pushLongNames)
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
			return ""
		}
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
	case "checkout", "switch":
		if moves, lost := c.forcedSwitch(g); lost {
			return fmt.Sprintf("git %s --force would discard uncommitted changes. %s", g.sub, discardAdvice)
		} else if moves || g.sub == "switch" {
			return ""
		}
		fallthrough
	case "restore":
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
		fromCommit := g.sub == "checkout" && len(o.pos) > 0 && (o.dashed || !hasAll(o.pos[:1]))
		if c.dirty(g, specs, !fromCommit) {
			return fmt.Sprintf("git %s would discard uncommitted changes. %s", g.sub, discardAdvice)
		}
	case "worktree":
		o := parseOpts(g.args, "", nil, map[string]string{"--force": "-f"})
		words := append(o.pos, o.paths...)
		if len(words) < 2 || words[0].Val != "remove" || !o.has("-f") || words[1].Dynamic {
			return ""
		}
		dir, locked := c.worktree(g, words[1].Val)
		if locked && o.set["-f"] > 1 {
			return fmt.Sprintf("git worktree remove --force --force would remove %s, which someone locked. Unlock it with git worktree unlock if it is yours; if the user asked for this, rerun with AGENT_HOOKS_ALLOW=destructive.", dir)
		}
		// Without --force git refuses exactly when this status is not empty.
		if entries, ok := c.status(&gitCmd{dir: dir}); ok && len(entries) > 0 {
			return fmt.Sprintf("git worktree remove --force would delete uncommitted work in %s (%s, ...). %s", dir, entries[0].path, discardAdvice)
		}
	case "stash":
		if len(g.args) > 0 && g.args[0].Val == "clear" && c.refExists(g, "refs/stash") {
			return "git stash clear would delete every stash in this repository, including ones from other worktrees. Drop the ones you made instead; if the user asked to clear them, rerun with AGENT_HOOKS_ALLOW=destructive."
		}
	}
	return ""
}}

var switchAliases = map[string]string{"--force": "-f", "--discard-changes": "-f", "--create": "-c", "--force-create": "-C"}

// forcedSwitch looks at checkout -f and switch -f. moves says the command
// moves HEAD, where git throws local changes away instead of refusing; a
// checkout that names paths only rewrites those. lost says the move would
// discard tracked changes, or untracked files the target tracks.
func (c *ctx) forcedSwitch(g *gitCmd) (moves, lost bool) {
	o := parseOpts(g.args, "bBcC", []string{"-b", "-B", "-c", "-C", "--orphan", "--conflict", "--pathspec-from-file"}, switchAliases)
	if !o.has("-f") {
		return false, false
	}
	pathsAllowed := g.sub == "checkout" && !o.has("-b", "-B", "--orphan")
	if pathsAllowed && (len(o.paths) > 0 || len(o.pos) > 1) {
		return false, false
	}
	target := ""
	if len(o.pos) > 0 && !o.pos[0].Dynamic {
		target = o.pos[0].Val
	}
	if target == "-" {
		target = "@{-1}"
	}
	if target != "" && !c.refExists(g, target) {
		// Not a commit: git creates the branch from the one remote that has
		// it, and failing that checkout reads the word as a path.
		refs, _ := c.git(g, "for-each-ref", "--format=%(refname)", "refs/remotes/*/"+target)
		target = strings.TrimSpace(refs)
		if target == "" || strings.Contains(target, "\n") {
			if pathsAllowed && !o.dashed {
				return false, false
			}
			target = ""
		}
	}
	return true, c.dirty(g, nil, false) || (target != "" && c.overwritesUntracked(g, target))
}

// worktree finds the worktree git would remove for arg, its path or a unique
// trailing part of one, and whether that worktree is locked.
func (c *ctx) worktree(g *gitCmd, arg string) (dir string, locked bool) {
	full := shell.Resolve(g.dir, arg, c.opts.Home)
	if real, err := filepath.EvalSymlinks(full); err == nil {
		full = real
	}
	out, _ := c.git(g, "worktree", "list", "--porcelain", "-z")
	found, trailing := -1, []int{}
	lines := strings.Split(out, "\x00")
	for i, line := range lines {
		path, ok := strings.CutPrefix(line, "worktree ")
		switch {
		case !ok:
		case path == full:
			found = i
		case strings.HasSuffix(path, "/"+strings.Trim(arg, "/")):
			trailing = append(trailing, i)
		}
	}
	if found < 0 && len(trailing) == 1 {
		found = trailing[0]
	}
	if found < 0 {
		return "", false
	}
	for _, line := range lines[found+1:] {
		if line == "" { // the end of this worktree's lines
			break
		}
		locked = locked || line == "locked" || strings.HasPrefix(line, "locked ")
	}
	return strings.TrimPrefix(lines[found], "worktree "), locked
}

func hasAll(specs []shell.Word) bool {
	for _, s := range specs {
		if v := strings.TrimSuffix(s.Val, "/"); v == "." || v == ":/" || v == ":" || v == "*" {
			return true
		}
	}
	return false
}

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
	localSet := map[string]bool{}
	var collapsed []string
	for _, p := range local {
		if strings.HasSuffix(p, "/") {
			collapsed = append(collapsed, p)
			continue
		}
		localSet[p] = true
		for q := p; q != "." && q != ""; q = path.Dir(q) {
			if tracked[q] {
				return true
			}
		}
	}
	root := ""
	if len(collapsed) > 0 {
		out, _ := c.git(g, "rev-parse", "--show-toplevel")
		root = strings.TrimSpace(out)
	}
	for t := range tracked {
		for q := t; q != "." && q != ""; q = path.Dir(q) {
			if localSet[q] {
				return true
			}
		}
		for _, dir := range collapsed {
			if strings.HasPrefix(t, dir) && (root == "" || exists(path.Join(root, t))) {
				return true
			}
		}
	}
	return false
}

var homeVar = regexp.MustCompile(`^\$(HOME|\{HOME\})/?(\*)?$`)

func checkRm(c *ctx, inv shell.Invocation) string {
	o := parseOpts(inv.Args[1:], "", nil, map[string]string{"--recursive": "-r", "-R": "-r"})
	if !o.has("-r") || rmAsksFirst(inv.Args[1:]) {
		return ""
	}
	for _, target := range append(o.pos, o.paths...) {
		if dangerousTarget(c, inv.Cwd, target) {
			return fmt.Sprintf("Refusing rm -r %s: it is /, your home directory, or a git repository root. Delete specific paths instead; if the user asked for this, rerun with AGENT_HOOKS_ALLOW=destructive.", target.Val)
		}
	}
	return ""
}

// rmAsksFirst reports whether rm would prompt before deleting: the last of
// -i, -I, -f, and --interactive decides. An agent cannot answer the prompt.
func rmAsksFirst(args []shell.Word) bool {
	asks := false
	for _, arg := range args {
		name, when, _ := strings.Cut(arg.Val, "=")
		// GNU rm accepts any unambiguous start of a long option, and of its value.
		long := func(option string) bool { return len(name) > 2 && strings.HasPrefix(option, name) }
		switch {
		case name == "--":
			return asks
		case long("--force"):
			asks = false
		case long("--interactive"):
			asks = !strings.HasPrefix(when, "n") // never, no, none
		case strings.HasPrefix(name, "-") && !strings.HasPrefix(name, "--"):
			for _, flag := range name[1:] {
				switch flag {
				case 'i', 'I':
					asks = true
				case 'f':
					asks = false
				}
			}
		}
	}
	return asks
}

func dangerousTarget(c *ctx, cwd string, target shell.Word) bool {
	v := target.Val
	if target.Dynamic {
		return homeVar.MatchString(v)
	}
	v = strings.TrimSuffix(v, "/*")
	if v == "*" && !target.Quoted {
		v = "." // an unquoted * is everything in the current directory
	}
	if strings.HasPrefix(v, "~") && target.Quoted {
		v = "./" + v
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
