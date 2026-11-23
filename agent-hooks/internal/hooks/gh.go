package hooks

import (
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/aryan-binazir/cc-config/agent-hooks/internal/shell"
)

// ghPr is an effective `gh pr create|merge` invocation, following gh's option
// parsing: flags may come before the command words, boolean flags take =false,
// and short flags cluster.
type ghPr struct {
	action      string              // "create" or "merge"
	flags       map[string]bool     // boolean flags in effect
	values      map[string][]string // value flags, by long name
	positionals []string
}

var ghValueFlags = map[string][]string{
	"create": {"--assignee", "--attach", "--base", "--body", "--body-file", "--head", "--label", "--milestone", "--project", "--recover", "--reviewer", "--template", "--title", "--repo"},
	"merge":  {"--author-email", "--body", "--body-file", "--match-head-commit", "--subject", "--repo"},
}

var ghShortValues = map[string]map[byte]string{
	"create": {'a': "--assignee", 'B': "--base", 'b': "--body", 'F': "--body-file", 'H': "--head", 'l': "--label", 'm': "--milestone", 'p': "--project", 'r': "--reviewer", 'T': "--template", 't': "--title", 'R': "--repo"},
	"merge":  {'A': "--author-email", 'b': "--body", 'F': "--body-file", 't': "--subject", 'R': "--repo"},
}

// ghWords selects up to two command words after gh, skipping flags and the
// values they probably take, and returns them with the remaining arguments.
func ghWords(inv shell.Invocation) (words, rest []string) {
	if inv.Name() != "gh" {
		return nil, nil
	}
	rest = inv.Strings()
	for level := 0; level < 2; level++ {
		for i := 0; i < len(rest); i++ {
			arg := rest[i]
			if arg == "--" {
				break
			}
			if strings.HasPrefix(arg, "-") {
				if !strings.Contains(arg, "=") && ((strings.HasPrefix(arg, "--") && arg != "--help" && !(level == 0 && arg == "--version")) || len(arg) == 2) {
					i++
				}
				continue
			}
			if arg == "" {
				continue
			}
			words = append(words, arg)
			rest = append(rest[:i:i], rest[i+1:]...)
			break
		}
	}
	return words, rest
}

func parseGhPr(inv shell.Invocation) *ghPr {
	words, rest := ghWords(inv)
	if len(words) < 2 || words[0] != "pr" {
		return nil
	}
	action := words[1]
	if action == "new" {
		action = "create"
	}
	if action != "create" && action != "merge" {
		return nil
	}
	pr := &ghPr{action: action, flags: map[string]bool{}, values: map[string][]string{}}
	record := func(name, value string, hasValue bool) {
		if truthy(value) {
			hasValue = false
		}
		if hasValue {
			delete(pr.flags, name)
		} else {
			pr.flags[name] = true
		}
	}
	for i := 0; i < len(rest); i++ {
		arg := rest[i]
		switch {
		case arg == "--":
			pr.positionals = append(pr.positionals, rest[i+1:]...)
			i = len(rest)
		case strings.HasPrefix(arg, "--"):
			name, value, hasValue := strings.Cut(arg, "=")
			if slices.Contains(ghValueFlags[action], name) {
				if !hasValue && i+1 < len(rest) {
					i++
					value = rest[i]
				}
				pr.values[name] = append(pr.values[name], value)
			} else {
				record(name, value, hasValue)
			}
		case strings.HasPrefix(arg, "-") && len(arg) > 1:
			for j := 1; j < len(arg); j++ {
				flag := arg[j]
				if long, ok := ghShortValues[action][flag]; ok {
					value := strings.TrimPrefix(arg[j+1:], "=")
					if j == len(arg)-1 {
						value = ""
						if i+1 < len(rest) {
							i++
							value = rest[i]
						}
					}
					pr.values[long] = append(pr.values[long], value)
					break
				}
				value, hasValue := "", j+1 < len(arg) && arg[j+1] == '='
				if hasValue {
					value = arg[j+2:]
				}
				name := "-" + string(flag)
				switch {
				case flag == 'h':
					name = "--help"
				case action == "create" && flag == 'd':
					name = "--draft"
				case action == "create" && flag == 'w':
					name = "--web"
				}
				record(name, value, hasValue)
				if hasValue {
					break
				}
			}
		default:
			pr.positionals = append(pr.positionals, arg)
		}
	}
	if pr.flags["--help"] {
		return nil
	}
	return pr
}

// has reports whether the last value given for a flag is non-empty, as gh
// keeps only the last value of a scalar flag.
func (pr *ghPr) has(name string) bool {
	vals := pr.values[name]
	if name == "--assignee" {
		return slices.ContainsFunc(vals, func(v string) bool { return v != "" })
	}
	return len(vals) > 0 && vals[len(vals)-1] != ""
}

var draftPrs = hook{name: "draft-prs", phase: "pre", check: func(_ *ctx, inv shell.Invocation) string {
	pr := parseGhPr(inv)
	switch {
	case pr == nil || pr.action != "create":
		return ""
	case pr.flags["--web"]:
		return "gh cannot create drafts with --web. Drop --web/-w and pass --draft."
	case !pr.flags["--draft"]:
		return "Create PRs as drafts. Add --draft; mark ready with gh pr ready after review."
	}
	return ""
}}

var prAssignee = hook{name: "pr-assignee", phase: "pre", check: func(_ *ctx, inv shell.Invocation) string {
	if pr := parseGhPr(inv); pr != nil && pr.action == "create" && !pr.has("--assignee") {
		return "Assign PRs when you create them: add --assignee @me, or --assignee <login> for someone else."
	}
	return ""
}}

var mergeHeadMatch = hook{name: "merge-head-match", phase: "pre", check: func(_ *ctx, inv shell.Invocation) string {
	if pr := parseGhPr(inv); pr != nil && pr.action == "merge" && !pr.flags["--disable-auto"] && !pr.has("--match-head-commit") {
		return "Pin the merge to the commit you verified: add --match-head-commit <sha> (gh pr view <pr> --json headRefOid -q .headRefOid)."
	}
	return ""
}}

var noAdminMerge = hook{name: "no-admin-merge", phase: "pre", check: func(_ *ctx, inv shell.Invocation) string {
	if pr := parseGhPr(inv); pr != nil && pr.action == "merge" && pr.flags["--admin"] {
		return "Merge without --admin. Satisfy the repository review and check requirements instead of bypassing them."
	}
	if api := parseGhAPI(inv); api != nil && api.method == "PUT" && pullMerge.MatchString(api.endpoint) {
		return "Merge with gh pr merge --match-head-commit <sha>, not the REST merge endpoint: a direct API merge skips gh's checks and bypasses branch protection for admins."
	}
	return ""
}}

var pullMerge = regexp.MustCompile(`(^|/)pulls/[^/]+/merge/?$`)

// ghAPI is a `gh api` call.
type ghAPI struct {
	endpoint string
	method   string // upper case; GET unless fields or --input imply POST
}

var ghAPIValueShort = "XfFHqtp"
var ghAPIValueLong = []string{"--method", "--raw-field", "--field", "--header", "--input", "--jq", "--template", "--preview", "--hostname", "--cache"}

func parseGhAPI(inv shell.Invocation) *ghAPI {
	if inv.Name() != "gh" {
		return nil
	}
	args := inv.Strings()
	i := 0
	for i < len(args) && strings.HasPrefix(args[i], "-") {
		i++ // gh takes no global options with values before a command
	}
	if i >= len(args) || args[i] != "api" {
		return nil
	}
	rest := args[i+1:]
	api := &ghAPI{}
	body, help := false, false
	for i := 0; i < len(rest); i++ {
		arg := rest[i]
		name, value, hasValue := strings.Cut(arg, "=")
		switch {
		case name == "--help":
			help = !hasValue || truthy(value) // the last value wins
		case arg == "-h":
			help = true
		case strings.HasPrefix(arg, "--") && slices.Contains(ghAPIValueLong, name):
			if !hasValue && i+1 < len(rest) {
				i++
				value = rest[i]
			}
			switch name {
			case "--method":
				api.method = strings.ToUpper(value)
			case "--raw-field", "--field", "--input":
				body = true
			}
		case strings.HasPrefix(arg, "--"):
		case strings.HasPrefix(arg, "-") && len(arg) > 1:
			for j := 1; j < len(arg); j++ {
				flag := arg[j]
				if flag == 'h' {
					return nil
				}
				if strings.IndexByte(ghAPIValueShort, flag) < 0 {
					continue
				}
				value := strings.TrimPrefix(arg[j+1:], "=")
				if j == len(arg)-1 && i+1 < len(rest) {
					i++
					value = rest[i]
				}
				switch flag {
				case 'X':
					api.method = strings.ToUpper(value)
				case 'f', 'F':
					body = true
				}
				break
			}
		default:
			if api.endpoint == "" {
				api.endpoint = arg
			}
		}
	}
	if help {
		return nil
	}
	if api.method == "" {
		api.method = "GET"
		if body {
			api.method = "POST"
		}
	}
	api.endpoint = path.Clean("/" + api.endpoint)
	return api
}

// truthy matches the boolean spellings gh's flag parser accepts as true.
func truthy(v string) bool {
	switch v {
	case "1", "t", "T", "true", "TRUE", "True":
		return true
	}
	return false
}
