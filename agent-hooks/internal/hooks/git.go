package hooks

import (
	"os"
	"path"
	"strings"

	"github.com/aryan-binazir/cc-config/agent-hooks/internal/shell"
)

// gitCmd is a `git <subcommand>` invocation with its global options applied.
type gitCmd struct {
	sub  string
	args []shell.Word
	dir  string   // effective working directory; "" when unknown
	env  []string // GIT_DIR and friends that select the repository
}

var gitGlobalValues = []string{"-C", "-c", "--git-dir", "--work-tree", "--namespace", "--config-env", "--super-prefix", "--exec-path"}

// gitRepoEnv are the variables that point git at a different repository or index.
var gitRepoEnv = []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_COMMON_DIR"}

func parseGit(inv shell.Invocation, home string) *gitCmd {
	if inv.Name() != "git" {
		return nil
	}
	g := &gitCmd{dir: inv.Cwd}
	repo := map[string]shell.Word{}
	for _, name := range gitRepoEnv {
		if v, ok := inv.Assigns[name]; ok {
			repo[name] = shell.Word{Val: v}
		}
	}
	args := inv.Args[1:]
	for i := 0; i < len(args); i++ {
		a := args[i].Val
		if !strings.HasPrefix(a, "-") {
			g.sub = a
			g.args = args[i+1:]
			break
		}
		name, value, hasValue := strings.Cut(a, "=")
		if !contains(gitGlobalValues, name) {
			continue
		}
		word := shell.Word{Val: value, Dynamic: args[i].Dynamic}
		if !hasValue {
			if i+1 >= len(args) {
				break
			}
			i++
			word = args[i]
		}
		switch name {
		case "-C":
			g.dir = ""
			if !word.Dynamic {
				g.dir = shell.Resolve(inv.Cwd, word.Val, home)
			}
			inv.Cwd = g.dir
		case "--git-dir":
			repo["GIT_DIR"] = word
		case "--work-tree":
			repo["GIT_WORK_TREE"] = word
		}
	}
	if g.sub == "" {
		return nil
	}
	for _, name := range gitRepoEnv {
		w, ok := repo[name]
		if !ok {
			continue
		}
		full := ""
		if !w.Dynamic {
			full = shell.Resolve(g.dir, w.Val, home)
		}
		if full == "" {
			g.dir = "" // the repository is unknowable, so skip lookups
			return g
		}
		g.env = append(g.env, name+"="+full)
	}
	return g
}

// opts is a parsed argument list. Flags are keyed by their canonical spelling.
type opts struct {
	set    map[string]int
	vals   map[string][]shell.Word
	pos    []shell.Word // operands before --
	paths  []shell.Word // operands after --
	dashed bool         // a -- was present
}

// parseOpts reads args given the short options that take a value, the long
// options that take a separate value, and aliases from long to canonical names.
func parseOpts(args []shell.Word, shortValues string, longValues []string, alias map[string]string) opts {
	o := opts{set: map[string]int{}, vals: map[string][]shell.Word{}}
	canon := func(name string) string {
		if c, ok := alias[name]; ok {
			return c
		}
		return name
	}
	for i := 0; i < len(args); i++ {
		a := args[i].Val
		switch {
		case a == "--" && !o.dashed:
			o.dashed = true
			o.paths = append(o.paths, args[i+1:]...)
			return o
		case strings.HasPrefix(a, "--"):
			name, value, hasValue := strings.Cut(a, "=")
			name = canon(name)
			o.set[name]++
			if hasValue {
				o.vals[name] = append(o.vals[name], shell.Word{Val: value, Dynamic: args[i].Dynamic})
			} else if contains(longValues, name) && i+1 < len(args) {
				i++
				o.vals[name] = append(o.vals[name], args[i])
			}
		case strings.HasPrefix(a, "-") && len(a) > 1:
			for j := 1; j < len(a); j++ {
				name := canon("-" + string(a[j]))
				o.set[name]++
				if strings.IndexByte(shortValues, a[j]) < 0 {
					continue
				}
				if j < len(a)-1 {
					o.vals[name] = append(o.vals[name], shell.Word{Val: a[j+1:], Dynamic: args[i].Dynamic})
				} else if i+1 < len(args) {
					i++
					o.vals[name] = append(o.vals[name], args[i])
				}
				break
			}
		default:
			o.pos = append(o.pos, args[i])
		}
	}
	return o
}

func (o opts) has(names ...string) bool {
	for _, n := range names {
		if o.set[n] > 0 {
			return true
		}
	}
	return false
}

func values(words []shell.Word) []string {
	out := make([]string, len(words))
	for i, w := range words {
		out[i] = w.Val
	}
	return out
}

// porcelain lists `git status --porcelain=v1 -z` entries as (XY, path) pairs.
type entry struct {
	xy   string
	path string
}

func (c *ctx) status(g *gitCmd, extra ...string) ([]entry, bool) {
	out, ok := c.git(g, append([]string{"status", "--porcelain=v1", "-z"}, extra...)...)
	if !ok {
		return nil, false
	}
	var entries []entry
	records := strings.Split(out, "\x00")
	for i := 0; i < len(records); i++ {
		r := records[i]
		if len(r) < 4 {
			continue
		}
		e := entry{xy: r[:2], path: r[3:]}
		entries = append(entries, e)
		if e.xy[0] == 'R' || e.xy[0] == 'C' {
			i++ // the original path of a rename or copy follows
		}
	}
	return entries, true
}

func withPaths(args []string, paths []shell.Word) []string {
	if len(paths) == 0 {
		return args
	}
	return append(append(args, "--"), values(paths)...)
}

func (c *ctx) currentBranch(g *gitCmd) string {
	out, _ := c.git(g, "symbolic-ref", "--short", "-q", "HEAD")
	return strings.TrimSpace(out)
}

func isDir(dir, p, home string) bool {
	full := shell.Resolve(dir, p, home)
	if full == "" {
		return false
	}
	info, err := os.Stat(full)
	return err == nil && info.IsDir()
}

func exists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}

func baseName(p string) string { return path.Base(strings.TrimRight(p, "/")) }

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
