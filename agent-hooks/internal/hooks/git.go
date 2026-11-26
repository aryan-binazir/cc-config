package hooks

import (
	"cmp"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/aryan-binazir/cc-config/agent-hooks/internal/shell"
)

type gitCmd struct {
	sub    string
	at     int // index of sub in the invocation's words
	args   []shell.Word
	dir    string
	env    []string
	config map[string]string
}

// parseGit reads a git invocation, following plain aliases to the command
// they stand for. Shell aliases (alias.x = !cmd) are not followed.
func (c *ctx) parseGit(inv shell.Invocation) *gitCmd {
	g := parseGit(inv, c.opts.Home)
	for hops := 0; g != nil && !gitShips[g.sub] && !inv.Args[g.at].Dynamic && hops < 4; hops++ {
		words := c.alias(g)
		if len(words) == 0 {
			break
		}
		inv.Args = slices.Concat(inv.Args[:g.at], words, inv.Args[g.at+1:])
		g = parseGit(inv, c.opts.Home)
	}
	if g != nil && g.sub == "stage" {
		g.sub = "add"
	}
	return g
}

func (c *ctx) alias(g *gitCmd) []shell.Word {
	value, ok := g.config["alias."+strings.ToLower(g.sub)]
	if !ok {
		where := *g
		if where.dir == "" { // global aliases still apply when the directory is unknown
			where.dir, where.env = cmp.Or(c.call.cwd, c.opts.Home, "/"), nil
		}
		value, _ = c.git(&where, "config", "--get", "alias."+g.sub)
	}
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "!") {
		return nil
	}
	if invs, unread := shell.Parse(value, "", ""); len(invs) == 1 && len(unread) == 0 {
		return invs[0].Args // git splits an alias on shell-style quoting
	}
	var words []shell.Word
	for _, field := range strings.Fields(value) {
		words = append(words, shell.Word{Val: field})
	}
	return words
}

var gitGlobalValues = []string{"-C", "-c", "--git-dir", "--work-tree", "--namespace", "--config-env", "--super-prefix", "--exec-path"}

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
			g.sub, g.at = a, i+1
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
		case "-c":
			if k, v, ok := strings.Cut(word.Val, "="); ok {
				if g.config == nil {
					g.config = map[string]string{}
				}
				g.config[strings.ToLower(k)] = v
			}
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
			g.dir = ""
			return g
		}
		g.env = append(g.env, name+"="+full)
	}
	return g
}

type opts struct {
	set    map[string]int
	vals   map[string][]shell.Word
	pos    []shell.Word
	paths  []shell.Word
	dashed bool
}

func parseOpts(args []shell.Word, shortValues string, longValues []string, alias map[string]string) opts {
	o := opts{set: map[string]int{}, vals: map[string][]shell.Word{}}
	canon := func(name string) string {
		if c, ok := alias[name]; ok {
			return c
		}
		// git accepts any unambiguous prefix of a long option.
		match := ""
		for long := range alias {
			if len(name) > 3 && strings.HasPrefix(long, name) {
				if match != "" {
					return name
				}
				match = long
			}
		}
		if match != "" {
			return alias[match]
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
			if negated, ok := strings.CutPrefix(name, "--no-"); ok {
				delete(o.set, canon("--"+negated))
				delete(o.vals, canon("--"+negated))
			}
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

// gitShips lists the commands git 2.56 installs. An alias cannot shadow one,
// so these never need an alias lookup.
var gitShips = func() map[string]bool {
	ships := map[string]bool{}
	for _, name := range strings.Fields(`
add am annotate apply archimport archive backfill bisect blame branch bugreport bundle cat-file check-attr
check-ignore check-mailmap checkout checkout-index check-ref-format cherry cherry-pick citool clean clone
column commit commit-graph commit-tree config count-objects credential credential-cache credential-libsecret
credential-store cvsexportcommit cvsimport cvsserver daemon describe diagnose diff diff-files diff-index
diff-pairs difftool diff-tree fast-export fast-import fetch fetch-pack filter-branch fmt-merge-msg
for-each-ref for-each-repo format-patch format-rev fsck fsck-objects gc get-tar-commit-id grep gui
hash-object help history hook http-backend http-fetch http-push imap-send index-pack init init-db instaweb
interpret-trailers last-modified log ls-files ls-remote ls-tree mailinfo mailsplit maintenance merge
merge-base merge-file merge-index merge-octopus merge-one-file merge-ours merge-recursive
merge-recursive-ours merge-recursive-theirs merge-resolve merge-subtree mergetool merge-tree mktag mktree
multi-pack-index mv name-rev notes p4 pack-objects pack-redundant pack-refs patch-id pickaxe prune
prune-packed pull push quiltimport range-diff read-tree rebase receive-pack reflog refs remote remote-ext
remote-fd remote-ftp remote-ftps remote-http remote-https repack replace replay repo request-pull rerere
reset restore revert rev-list rev-parse rm send-email send-pack shell shortlog show show-branch show-index
show-ref sparse-checkout stage stash status stripspace submodule subtree svn switch symbolic-ref tag
unpack-file unpack-objects update-index update-ref update-server-info upload-archive upload-pack url-parse
var verify-commit verify-pack verify-tag version whatchanged worktree write-tree
`) {
		ships[name] = true
	}
	return ships
}()
