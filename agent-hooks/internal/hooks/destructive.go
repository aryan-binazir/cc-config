package hooks

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/aryan-binazir/cc-config/agent-hooks/internal/shell"
)

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
		o := g.opts()
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
		o := g.opts()
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
		o := g.opts()
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
		o := g.opts()
		if g.sub == "restore" && o.has("-S") && !o.has("-W") {
			return ""
		}
		specs := o.paths
		if g.sub == "restore" || !o.dashed {
			specs = append(append([]shell.Word{}, o.pos...), o.paths...)
		}
		specs = o.listed(specs)
		if !hasAll(specs) {
			return ""
		}
		// From a commit, checkout rewrites the index too, and so does restore --staged --worktree.
		rewritesIndex := o.has("-S") || (g.sub == "checkout" && len(o.pos) > 0 && (o.dashed || !hasAll(o.pos[:1])))
		if c.dirty(g, specs, !rewritesIndex) {
			return fmt.Sprintf("git %s would discard uncommitted changes. %s", g.sub, discardAdvice)
		}
	case "worktree":
		o := g.opts()
		words := append(o.pos, o.paths...)
		if len(words) < 2 || words[0].Val != "remove" || !o.has("-f") || words[1].Dynamic {
			return ""
		}
		dir, locked := c.worktree(g, words[1].Val)
		if locked && o.set["-f"] > 1 {
			return fmt.Sprintf("git worktree remove --force --force would remove %s, which someone locked. Unlock it with git worktree unlock if it is yours; if the user asked for this, rerun with AGENT_HOOKS_ALLOW=destructive.", dir)
		}
		// Without --force git refuses exactly when this status is not empty,
		// or the worktree has a submodule checked out.
		if entries, ok := c.status(&gitCmd{dir: dir}); ok && len(entries) > 0 {
			return fmt.Sprintf("git worktree remove --force would delete uncommitted work in %s (%s, ...). %s", dir, entries[0].path, discardAdvice)
		}
		if c.holdsSubmodules(dir) {
			return fmt.Sprintf("git worktree remove --force would delete the submodule repositories in %s, with any commits and stashes only they hold. Git refuses this without --force; if the user asked for it, rerun with AGENT_HOOKS_ALLOW=destructive.", dir)
		}
	case "stash":
		if len(g.args) > 0 && g.args[0].Val == "clear" && c.refExists(g, "refs/stash") {
			return "git stash clear would delete every stash in this repository, including ones from other worktrees. Drop the ones you made instead; if the user asked to clear them, rerun with AGENT_HOOKS_ALLOW=destructive."
		}
	}
	return ""
}}

// forcedSwitch looks at checkout -f and switch -f. moves says the command
// moves HEAD, where git throws local changes away instead of refusing; a
// checkout that names paths only rewrites those. A pathspec list may be
// empty, which moves HEAD. lost says the move would discard tracked changes,
// or untracked files the target tracks.
func (c *ctx) forcedSwitch(g *gitCmd) (moves, lost bool) {
	o := g.opts()
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
		// Not a commit: unless --no-guess says otherwise, git creates the
		// branch from the one remote that has it, or from the remote
		// checkout.defaultRemote names, and failing that checkout reads the
		// word as a path.
		var refs []string
		if !o.has("--no-guess") || o.has("--guess") {
			out, _ := c.git(g, "for-each-ref", "--format=%(refname)", "refs/remotes/*/"+target)
			refs = strings.Fields(out)
		}
		if len(refs) > 1 {
			preferred, inline := g.config["checkout.defaultremote"]
			if !inline {
				preferred, _ = c.git(g, "config", "--get", "checkout.defaultRemote")
			}
			refs = slices.DeleteFunc(refs, func(ref string) bool { return ref != "refs/remotes/"+strings.TrimSpace(preferred)+"/"+target })
		}
		switch {
		case len(refs) == 1:
			target = refs[0]
		case pathsAllowed && !o.dashed:
			return false, false
		default:
			target = ""
		}
	}
	return true, c.dirty(g, nil, false) || (target != "" && c.overwritesUntracked(g, target))
}

// holdsSubmodules mirrors the other reason git refuses to remove a worktree
// without --force: a submodule's repository sits in the worktree, or in the
// worktree's git directory, and is deleted with it.
func (c *ctx) holdsSubmodules(dir string) bool {
	wt := &gitCmd{dir: dir}
	gitDir, _ := c.git(wt, "rev-parse", "--absolute-git-dir")
	if gitDir = strings.TrimSpace(gitDir); gitDir != "" && isDir(gitDir, "modules", "") {
		return true
	}
	index, _ := c.git(wt, "ls-files", "--stage", "-z")
	for _, entry := range strings.Split(index, "\x00") {
		mode, name, _ := strings.Cut(entry, "\t")
		if strings.HasPrefix(mode, "160000 ") && exists(filepath.Join(dir, name, ".git")) {
			return true
		}
	}
	return false
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
		endOfRecord := line == ""
		if endOfRecord {
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
		for q := strings.TrimSuffix(p, "/"); q != "." && q != ""; q = path.Dir(q) {
			if tracked[q] {
				return true
			}
		}
		if strings.HasSuffix(p, "/") {
			collapsed = append(collapsed, p)
		} else {
			localSet[p] = true
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

var rmOptions = grammar("-r|-R|--recursive")

func checkRm(c *ctx, inv shell.Invocation) string {
	o := parseOpts(inv.Args[1:], rmOptions)
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
		v = "."
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
	if info, err := os.Lstat(full); err == nil && info.Mode()&os.ModeSymlink != 0 && !strings.HasSuffix(target.Val, "/") && !strings.HasSuffix(target.Val, "*") {
		return false // rm removes a link itself, not what it points to
	}
	bare := exists(path.Join(full, "HEAD")) && exists(path.Join(full, "objects")) && exists(path.Join(full, "refs"))
	return full == "/" || (c.opts.Home != "" && full == path.Clean(c.opts.Home)) || exists(path.Join(full, ".git")) || bare
}
