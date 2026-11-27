package hooks

import (
	"cmp"
	"os"
	"slices"
	"strings"
	"unicode"

	"github.com/aryan-binazir/cc-config/agent-hooks/internal/shell"
)

type gitCmd struct {
	sub    string
	subAt  int
	args   []shell.Word
	dir    string
	env    []string
	config map[string]string
}

func (c *ctx) parseGit(inv shell.Invocation) *gitCmd {
	g := parseGit(inv, c.opts.Home)
	for hops := 0; g != nil && !gitInstalls[g.sub] && !inv.Args[g.subAt].Dynamic && hops < 4; hops++ {
		words := c.alias(g)
		if len(words) == 0 {
			break
		}
		inv.Args = slices.Concat(inv.Args[:g.subAt], words, inv.Args[g.subAt+1:])
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
		where, scope := *g, []string{"config"}
		if where.dir == "" {
			where.dir, where.env = cmp.Or(c.opts.Home, "/"), nil
			scope = append(scope, "--global")
		}
		value, _ = c.git(&where, append(scope, "--get", "alias."+g.sub)...)
		value = strings.TrimSuffix(value, "\n") // git config prints one after the value
	}
	if strings.HasPrefix(value, "!") {
		return nil
	}
	return splitAlias(value)
}

// splitAlias divides an alias into words much as git does: on whitespace, with
// single quotes, double quotes, and backslash escapes, and nothing else
// special. Unlike git, it also splits on non-ASCII spaces such as U+00A0. It
// returns nil for an unclosed quote or a trailing backslash.
func splitAlias(s string) []shell.Word {
	var words []shell.Word
	var word strings.Builder
	open, quote, escaped := false, rune(0), false
	for _, r := range s {
		switch {
		case escaped:
			word.WriteRune(r)
			open, escaped = true, false
		case r == '\\' && quote != '\'':
			escaped = true
		case quote == 0 && unicode.IsSpace(r):
			if open {
				words = append(words, shell.Word{Val: word.String()})
				word.Reset()
				open = false
			}
		case quote == 0 && (r == '\'' || r == '"'):
			open, quote = true, r
		case r == quote:
			quote = 0
		default:
			word.WriteRune(r)
			open = true
		}
	}
	if escaped || quote != 0 {
		return nil
	}
	if open {
		words = append(words, shell.Word{Val: word.String()})
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
			g.sub, g.subAt = a, i+1
			g.args = args[i+1:]
			break
		}
		name, value, hasValue := strings.Cut(a, "=")
		if !slices.Contains(gitGlobalValues, name) {
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

// optSpec is one command's option grammar: every name it lists, mapped to
// the option's canonical name, and the canonical names that take a value.
// Options it does not list are read as flags.
type optSpec struct {
	canon  map[string]string
	values map[string]bool
}

// grammar reads entries like "-m|--message=": one option's names, the first
// canonical, with a trailing = when it takes a value.
func grammar(entries string) optSpec {
	s := optSpec{canon: map[string]string{}, values: map[string]bool{}}
	for _, entry := range strings.Fields(entries) {
		names := strings.Split(strings.TrimSuffix(entry, "="), "|")
		for _, name := range names {
			if _, dup := s.canon[name]; dup {
				panic("option listed twice: " + name)
			}
			s.canon[name] = names[0]
		}
		if strings.HasSuffix(entry, "=") {
			s.values[names[0]] = true
		}
	}
	return s
}

// gitOptions holds what the hooks read of each git subcommand's grammar, from
// git 2.56's: the options a hook checks, and every option that takes a value
// before the words a hook reads. Any unambiguous prefix of a listed long name
// stands for it, as in git, so a real option that is itself a prefix of one
// (--recurse-submodules, --set-upstream) is listed too.
var gitOptions = map[string]optSpec{
	"add":      grammar("-A|--all|--no-ignore-removal -u|--update -f|--force -n|--dry-run -p|--patch -i|--interactive -e|--edit --chmod= --pathspec-from-file="),
	"branch":   grammar("-d|--delete -f|--force -m|--move -c|--copy -u|--set-upstream-to= --set-upstream"),
	"checkout": grammar("-f|--force -b= -B= --orphan= --conflict= --guess --no-guess --pathspec-from-file="),
	"clean":    grammar("-f|--force -n|--dry-run -i|--interactive -e|--exclude="),
	"commit": grammar("-m|--message= -F|--file= -C|--reuse-message= -c|--reedit-message= -t|--template= -a|--all -i|--include -o|--only -p|--patch -z|--null " +
		"--interactive --amend --dry-run --short --porcelain --long --help --fixup= --squash= --author= --date= --trailer= --cleanup= --pathspec-from-file="),
	"fetch":      grammar("-o|--server-option= -j= --upload-pack= --depth= --deepen= --shallow-since= --shallow-exclude= --refmap= --recurse-submodules-default= --recurse-submodules --negotiation-tip="),
	"merge":      grammar("-m= -s|--strategy= -X|--strategy-option= --abort --continue --quit"),
	"pull":       grammar("-s|--strategy= -X|--strategy-option="),
	"push":       grammar("-f|--force -n|--dry-run -d|--delete -u|--set-upstream -o|--push-option= --repo= --receive-pack= --exec= --mirror --all --branches --prune --force-with-lease --tags"),
	"rebase":     grammar("-s|--strategy= -X|--strategy-option= -x|--exec= -C= --onto= --empty= --trailer= --whitespace= --abort --continue --skip --quit"),
	"reset":      grammar("--hard --soft --mixed --keep --merge"),
	"restore":    grammar("-s|--source= -S|--staged -W|--worktree --conflict= --pathspec-from-file="),
	"switch":     grammar("-f|--force|--discard-changes -c|--create= -C|--force-create= --orphan= --conflict= --guess --no-guess"),
	"update-ref": grammar("-m="),
	"worktree":   grammar("-f|--force"),
}

func (g *gitCmd) opts() opts { return parseOpts(g.args, gitOptions[g.sub]) }

func parseOpts(args []shell.Word, spec optSpec) opts {
	o := opts{set: map[string]int{}, vals: map[string][]shell.Word{}}
	canon := func(name string) string {
		if c, ok := spec.canon[name]; ok {
			return c
		}
		// git accepts any unambiguous prefix of a long option.
		match := ""
		for long := range spec.canon {
			if len(name) > 3 && strings.HasPrefix(long, name) {
				if match != "" {
					return name
				}
				match = long
			}
		}
		if match != "" {
			return spec.canon[match]
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
			} else if spec.values[name] && i+1 < len(args) {
				i++
				o.vals[name] = append(o.vals[name], args[i])
			}
		case strings.HasPrefix(a, "-") && len(a) > 1:
			for j := 1; j < len(a); j++ {
				name := canon("-" + string(a[j]))
				o.set[name]++
				if !spec.values[name] {
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

func (o opts) listed(specs []shell.Word) []shell.Word {
	if o.has(pathspecList) {
		wholeTree := shell.Word{Val: ":/"}
		return append(specs, wholeTree)
	}
	return specs
}

const pathspecList = "--pathspec-from-file"

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

// gitInstalls lists commands every git 2.56 has: the ones compiled in (git
// --list-cmds=builtins) and five of the scripts installed beside them. An
// alias cannot shadow an installed command, so these need no alias lookup.
var gitInstalls = func() map[string]bool {
	installs := map[string]bool{}
	for _, name := range strings.Fields(`
add am annotate apply archive backfill bisect blame branch bugreport bundle cat-file check-attr check-ignore
check-mailmap checkout checkout-index checkout--worker check-ref-format cherry cherry-pick clean clone column
commit commit-graph commit-tree config count-objects credential credential-cache credential-cache--daemon
credential-store describe diagnose diff diff-files diff-index diff-pairs difftool diff-tree fast-export
fast-import fetch fetch-pack fmt-merge-msg for-each-ref for-each-repo format-patch format-rev fsck
fsck-objects fsmonitor--daemon gc get-tar-commit-id grep hash-object help history hook index-pack init
init-db interpret-trailers last-modified log ls-files ls-remote ls-tree mailinfo mailsplit maintenance merge
merge-base merge-file merge-index merge-ours merge-recursive merge-recursive-ours merge-recursive-theirs
merge-subtree merge-tree mktag mktree multi-pack-index mv name-rev notes pack-objects pack-redundant
pack-refs patch-id pickaxe prune prune-packed pull push range-diff read-tree rebase receive-pack reflog refs
remote remote-ext remote-fd repack replace replay repo rerere reset restore revert rev-list rev-parse rm
send-pack shortlog show show-branch show-index show-ref sparse-checkout stage stash status stripspace
submodule--helper switch symbolic-ref tag unpack-file unpack-objects update-index update-ref
update-server-info upload-archive upload-archive--writer upload-pack url-parse var verify-commit verify-pack
verify-tag version whatchanged worktree write-tree
filter-branch mergetool quiltimport request-pull submodule
`) {
		installs[name] = true
	}
	return installs
}()
