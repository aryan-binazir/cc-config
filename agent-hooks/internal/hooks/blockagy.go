package hooks

import (
	"path"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/aryan-binazir/cc-config/agent-hooks/internal/shell"
)

var blockAgy = hook{
	name:      "block-agy",
	phase:     "pre",
	checkCall: checkAgy,
}

const agyMessage = `Blocked: agents must not run the Antigravity CLI (agy / antigravity / acg) at all, in any mode.
Google's consumer-plan terms forbid third-party agents wrapping agy (print mode, ACP, or driving it in tmux), and it gets the account suspended.
Do not work around this with scripts, other binary paths, aliases or variables.
If Gemini is needed, tell the user: use a GEMINI_API_KEY-based tool, or skip Gemini.`

var argumentRunners = map[string]bool{
	"tmux": true, "screen": true, "ssh": true, "script": true, "systemd-run": true, "parallel": true, "watch": true,
}

var (
	assignment = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)
	plainName  = regexp.MustCompile(`^[A-Za-z0-9._+-]+$`)
)

const separators = ";&|()`\n"

func isAgy(word string) bool {
	word = strings.NewReplacer(`"`, "", "'", "", "`", "").Replace(word)
	word = word[strings.LastIndexByte(word, '=')+1:]
	word = word[strings.LastIndexByte(word, '/')+1:]
	return word == "agy" || word == "antigravity" || word == "acg"
}

func agyOffsets(text string) []int {
	var at []int
	start := -1
	for i, r := range text {
		separator := unicode.IsSpace(r) || strings.ContainsRune(separators, r)
		switch {
		case !separator && start < 0:
			start = i
		case separator && start >= 0:
			if isAgy(text[start:i]) {
				at = append(at, start)
			}
			start = -1
		}
	}
	if start >= 0 && isAgy(text[start:]) {
		at = append(at, start)
	}
	return at
}

func checkAgy(c *ctx) string {
	found := map[*shell.Source][]int{}
	names := func(r shell.Reach) bool {
		at, scanned := found[r.Source]
		if !scanned {
			at = agyOffsets(r.Text)
			found[r.Source] = at
		}
		next, _ := slices.BinarySearch(at, r.Start)
		return next < len(at) && at[next] < r.End
	}
	namedSoFar := false
	for _, p := range c.parsed {
		for _, inv := range p.invs {
			if launchesAgy(inv, namedSoFar || names(inv.Fed) || slices.ContainsFunc(inv.Fed.Outer, names)) {
				return agyMessage
			}
		}
		namedSoFar = namedSoFar || len(agyOffsets(p.text)) > 0
		if slices.ContainsFunc(p.unread, func(text string) bool { return unreadLaunchesAgy(text, namedSoFar) }) {
			return agyMessage
		}
	}
	return ""
}

func launchesAgy(inv shell.Invocation, fedNamesAgy bool) bool {
	args := inv.Args[1:]
	switch {
	case isAgy(inv.Args[0].Val):
		return true
	case argumentRunners[inv.Name()]:
		return slices.ContainsFunc(args, func(w shell.Word) bool { return len(agyOffsets(w.Val)) > 0 || (fedNamesAgy && w.Dynamic) })
	}
	nameFromExpansion := inv.Args[0].Dynamic && !plainName.MatchString(inv.Name())
	return fedNamesAgy && (inv.Opaque || nameFromExpansion)
}

func unreadLaunchesAgy(text string, agyNamed bool) bool {
	for _, segment := range strings.FieldsFunc(text, func(r rune) bool { return strings.ContainsRune(separators, r) }) {
		words := strings.Fields(segment)
		for len(words) > 0 && assignment.MatchString(words[0]) {
			words = words[1:]
		}
		if len(words) == 0 {
			continue
		}
		first := path.Base(strings.NewReplacer(`"`, "", "'", "").Replace(words[0]))
		firstArg := ""
		if len(words) > 1 {
			firstArg = words[1]
		}
		if isAgy(words[0]) || (agyNamed && (argumentRunners[first] || shell.MayRunCommand(first, firstArg))) {
			return true
		}
	}
	return false
}
