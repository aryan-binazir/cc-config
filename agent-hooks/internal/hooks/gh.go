package hooks

import (
	"cmp"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/aryan-binazir/cc-config/agent-hooks/internal/shell"
)

// ghCmd is one gh command as gh reads it: the command words, as cobra finds
// them, and the flags after or between them, as pflag reads them.
type ghCmd struct {
	path string
	opts
}

// ghOptions holds what the hooks read of each gh command's flags, from gh
// 2.105's: the flags a hook checks, and the flags that take a value a hook
// would otherwise misread as an argument. Unlike git, gh accepts no
// abbreviated or --no- names.
var ghOptions = map[string]optSpec{
	"pr create": grammar("--assignee|-a= --attach= --base|-B= --body|-b= --body-file|-F= --head|-H= --label|-l= --milestone|-m= --project|-p= --recover= --reviewer|-r= --template|-T= --title|-t= --repo|-R= --draft|-d --web|-w"),
	"pr merge":  grammar("--author-email|-A= --body|-b= --body-file|-F= --match-head-commit= --subject|-t= --repo|-R= --delete-branch|-d"),
	"pr edit": grammar("--add-assignee= --add-label= --add-project= --add-reviewer= --attach= --base|-B= --body|-b= --body-file|-F= --milestone|-m= " +
		"--remove-assignee= --remove-label= --remove-project= --remove-reviewer= --title|-t= --repo|-R="),
	"api":       grammar("--method|-X= --raw-field|-f= --field|-F= --header|-H= --input= --jq|-q= --template|-t= --preview|-p= --hostname= --cache="),
	"repo sync": grammar("--branch|-b= --source|-s="),
}

// parseGh returns nil for a gh command the hooks do not read, or one that
// only shows help.
func parseGh(inv shell.Invocation) *ghCmd {
	if inv.Name() != "gh" {
		return nil
	}
	path, rest := ghPath(inv.Args[1:], true)
	spec, ok := ghOptions[path]
	if !ok {
		path, rest = ghPath(inv.Args[1:], false)
		if spec, ok = ghOptions[path]; !ok {
			return nil
		}
	}
	g := &ghCmd{path: path, opts: opts{set: map[string]int{}, vals: map[string][]shell.Word{}}}
	canon := func(name string) string { return cmp.Or(spec.canon[name], name) }
	flag := func(name, value string, hasValue bool) {
		if hasValue && !truthy(value) {
			delete(g.set, name)
		} else {
			g.set[name] = 1
		}
	}
	for i := 0; i < len(rest); i++ {
		a := rest[i].Val
		switch {
		case a == "--":
			g.pos = append(g.pos, rest[i+1:]...)
			i = len(rest)
		case strings.HasPrefix(a, "--"):
			name, value, hasValue := strings.Cut(a, "=")
			name = canon(name)
			if !spec.values[name] {
				flag(name, value, hasValue)
				continue
			}
			word := shell.Word{Val: value, Dynamic: rest[i].Dynamic}
			if !hasValue && i+1 < len(rest) {
				i++
				word = rest[i]
			}
			g.vals[name] = append(g.vals[name], word)
		case strings.HasPrefix(a, "-") && len(a) > 1:
			for j := 1; j < len(a); j++ {
				name := canon("-" + a[j:j+1])
				value, hasValue := strings.CutPrefix(a[j+1:], "=")
				if name == "-h" { // gh shows help for -h whatever follows it
					g.set[name] = 1
					break
				}
				if !spec.values[name] {
					flag(name, value, hasValue)
					if hasValue {
						break
					}
					continue
				}
				word := shell.Word{Val: value, Dynamic: rest[i].Dynamic}
				if j == len(a)-1 {
					word = shell.Word{}
					if i+1 < len(rest) {
						i++
						word = rest[i]
					}
				}
				g.vals[name] = append(g.vals[name], word)
				break
			}
		default:
			g.pos = append(g.pos, rest[i])
		}
	}
	if g.has("-h", "--help") {
		return nil
	}
	return g
}

// ghPath finds the command words as cobra does, skipping a value after any
// flag before them that is not known to be a boolean, and returns the other
// arguments. An expanded flag may carry its own =value, so dynamicTakesValue
// picks which reading to take of one.
func ghPath(args []shell.Word, dynamicTakesValue bool) (string, []shell.Word) {
	var words []string
	rest := args
	for level := 0; level < 2; level++ {
		if _, leaf := ghOptions[strings.Join(words, " ")]; leaf {
			break
		}
		for i := 0; i < len(rest); i++ {
			arg := rest[i].Val
			if arg == "--" {
				break
			}
			if strings.HasPrefix(arg, "-") {
				if (dynamicTakesValue || !rest[i].Dynamic) && !strings.Contains(arg, "=") && ((strings.HasPrefix(arg, "--") && arg != "--help" && !(level == 0 && arg == "--version")) || len(arg) == 2) {
					i++
				}
				continue
			}
			if arg == "" {
				continue
			}
			if strings.Contains(arg, " ") {
				return "", nil // no gh command has one
			}
			words = append(words, arg)
			rest = append(rest[:i:i], rest[i+1:]...)
			break
		}
	}
	path := strings.Join(words, " ")
	if path == "pr new" {
		path = "pr create"
	}
	return path, rest
}

// given reports whether the value gh uses for name is non-empty: the last one,
// or any one for a list flag like --assignee.
func (g *ghCmd) given(name string) bool {
	vals := g.vals[name]
	if name == "--assignee" {
		return slices.ContainsFunc(vals, func(v shell.Word) bool { return v.Val != "" })
	}
	return len(vals) > 0 && vals[len(vals)-1].Val != ""
}

// request is the method and endpoint of a gh api call: the last method given,
// else POST when it sends fields or a body, else GET.
func (g *ghCmd) request() (method, endpoint string) {
	if methods := g.vals["--method"]; len(methods) > 0 {
		method = strings.ToUpper(methods[len(methods)-1].Val)
	}
	if method == "" {
		method = "GET"
		if len(g.vals["--raw-field"])+len(g.vals["--field"])+len(g.vals["--input"]) > 0 {
			method = "POST"
		}
	}
	if len(g.pos) > 0 {
		endpoint = g.pos[0].Val
	}
	return method, path.Clean("/" + endpoint)
}

var draftPrs = hook{name: "draft-prs", phase: "pre", check: func(_ *ctx, inv shell.Invocation) string {
	pr := parseGh(inv)
	switch {
	case pr == nil || pr.path != "pr create":
		return ""
	case pr.has("--web"):
		return "gh cannot create drafts with --web. Drop --web/-w and pass --draft."
	case !pr.has("--draft"):
		return "Create PRs as drafts. Add --draft; mark ready with gh pr ready after review."
	}
	return ""
}}

var prAssignee = hook{name: "pr-assignee", phase: "pre", check: func(_ *ctx, inv shell.Invocation) string {
	if pr := parseGh(inv); pr != nil && pr.path == "pr create" && !pr.given("--assignee") {
		return "Assign PRs when you create them: add --assignee @me, or --assignee <login> for someone else."
	}
	return ""
}}

var mergeHeadMatch = hook{name: "merge-head-match", phase: "pre", check: func(_ *ctx, inv shell.Invocation) string {
	if pr := parseGh(inv); pr != nil && pr.path == "pr merge" && !pr.has("--disable-auto") && !pr.given("--match-head-commit") {
		return "Pin the merge to the commit you verified: add --match-head-commit <sha> (gh pr view <pr> --json headRefOid -q .headRefOid)."
	}
	return ""
}}

var noAdminMerge = hook{name: "no-admin-merge", phase: "pre", check: func(_ *ctx, inv shell.Invocation) string {
	cmd := parseGh(inv)
	switch {
	case cmd == nil:
	case cmd.path == "pr merge" && cmd.has("--admin"):
		return "Merge without --admin. Satisfy the repository review and check requirements instead of bypassing them."
	case cmd.path == "api":
		if method, endpoint := cmd.request(); method == "PUT" && pullMerge.MatchString(endpoint) {
			return "Merge with gh pr merge --match-head-commit <sha>, not the REST merge endpoint: a direct API merge skips gh's checks and bypasses branch protection for admins."
		}
	}
	return ""
}}

var pullMerge = regexp.MustCompile(`(^|/)pulls/[^/]+/merge/?$`)

// truthy matches the boolean spellings gh's flag parser accepts as true.
func truthy(v string) bool {
	switch v {
	case "1", "t", "T", "true", "TRUE", "True":
		return true
	}
	return false
}
