package hooks

import (
	"strings"

	"github.com/aryan-binazir/cc-config/agent-hooks/internal/shell"
)

// ripgrep 15's options that take a value.
const rgShortValues = "efEmjgdtTABCMr"

var rgLongValues = []string{
	"--regexp", "--file", "--encoding", "--max-count", "--threads", "--glob", "--max-depth", "--type", "--type-not",
	"--after-context", "--before-context", "--context", "--max-columns", "--pre", "--pre-glob", "--dfa-size-limit",
	"--engine", "--regex-size-limit", "--iglob", "--ignore-file", "--max-filesize", "--type-add", "--type-clear", "--color",
	"--colors", "--context-separator", "--field-context-separator", "--field-match-separator", "--hostname-bin",
	"--hyperlink-format", "--path-separator", "--sort", "--sortr", "--generate",
}

// rgReplaceFlag adds a note after rg runs with -r, which agents often mistake for recursive.
var rgReplaceFlag = hook{name: "rg-replace-flag", phase: "post", check: func(_ *ctx, inv shell.Invocation) string {
	if inv.Name() != "rg" || !rgReplaces(inv.Strings()) {
		return ""
	}
	return "In rg, -r means --replace: it rewrites each match in the output and does not mean recursive. rg already searches recursively, so drop -r unless you meant to replace."
}}

func rgReplaces(args []string) bool {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--":
			return false
		case strings.HasPrefix(arg, "--"):
			name, _, hasValue := strings.Cut(arg, "=")
			if name == "--replace" {
				return true
			}
			if contains(rgLongValues, name) && !hasValue {
				i++
			}
		case strings.HasPrefix(arg, "-") && len(arg) > 1:
			for j := 1; j < len(arg); j++ {
				if arg[j] == 'r' {
					return true
				}
				if strings.IndexByte(rgShortValues, arg[j]) >= 0 {
					if j == len(arg)-1 {
						i++
					}
					break
				}
			}
		}
	}
	return false
}
