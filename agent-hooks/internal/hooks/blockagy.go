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

// runners start their arguments as a command somewhere the parser does not
// follow: another host, a terminal multiplexer, a later moment.
var runners = map[string]bool{
	"tmux": true, "screen": true, "ssh": true, "script": true, "systemd-run": true, "parallel": true, "watch": true,
}

// launchers put another command behind them. The parser unwraps the ones it
// reads; this list is for text that did not parse.
var launchers = map[string]bool{
	"bash": true, "sh": true, "zsh": true, "dash": true, "fish": true, "eval": true, "source": true, ".": true,
	"env": true, "exec": true, "command": true, "nohup": true, "setsid": true, "timeout": true, "stdbuf": true,
	"nice": true, "time": true, "sudo": true, "doas": true, "xargs": true, "parallel": true, "watch": true,
	"unbuffer": true, "script": true, "systemd-run": true, "tmux": true, "screen": true, "ssh": true, "mise": true,
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

// firstMention is the offset of the first word in text that names agy, or -1.
func firstMention(text string) int {
	start := -1
	for i, r := range text {
		separator := unicode.IsSpace(r) || strings.ContainsRune(separators, r)
		switch {
		case !separator && start < 0:
			start = i
		case separator && start >= 0:
			if isAgy(text[start:i]) {
				return start
			}
			start = -1
		}
	}
	if start >= 0 && isAgy(text[start:]) {
		return start
	}
	return -1
}

func checkAgy(c *ctx) string {
	first := map[*shell.Source]int{}
	names := func(r shell.Reach) bool {
		at, scanned := first[r.Source]
		if !scanned {
			at = firstMention(r.Text)
			first[r.Source] = at
		}
		return at >= 0 && at < r.End
	}
	earlier := false // a previous source of this call names agy
	for _, p := range c.parsed {
		for _, inv := range p.invs {
			if launchesAgy(inv, earlier || names(inv.Fed) || slices.ContainsFunc(inv.Fed.Outer, names)) {
				return agyMessage
			}
		}
		earlier = earlier || firstMention(p.text) >= 0
		if slices.ContainsFunc(p.unread, func(text string) bool { return unreadLaunchesAgy(text, earlier) }) {
			return agyMessage
		}
	}
	return ""
}

// launchesAgy reports whether inv runs agy, or runs something the parser
// cannot read while agy is named in text that can feed it (fed).
func launchesAgy(inv shell.Invocation, fed bool) bool {
	args := inv.Args[1:]
	switch {
	case isAgy(inv.Args[0].Val):
		return true
	case runners[inv.Name()]:
		return slices.ContainsFunc(args, func(w shell.Word) bool { return firstMention(w.Val) >= 0 || (fed && w.Dynamic) })
	}
	// "$HOME/bin/tool" names tool wherever HOME points; "$tool" names anything.
	unknown := inv.Args[0].Dynamic && !plainName.MatchString(inv.Name())
	return fed && (inv.Opaque || unknown)
}

// unreadLaunchesAgy scans text the parser gave up on: agy as a command word,
// or any launcher once agy is named anywhere in the command.
func unreadLaunchesAgy(text string, mentioned bool) bool {
	for _, segment := range strings.FieldsFunc(text, func(r rune) bool { return strings.ContainsRune(separators, r) }) {
		words := strings.Fields(segment)
		for len(words) > 0 && assignment.MatchString(words[0]) {
			words = words[1:]
		}
		if len(words) == 0 {
			continue
		}
		first := path.Base(strings.NewReplacer(`"`, "", "'", "").Replace(words[0]))
		looksUp := first == "command" && len(words) > 1 && (words[1] == "-v" || words[1] == "-V")
		if isAgy(words[0]) || (mentioned && launchers[first] && !looksUp) {
			return true
		}
	}
	return false
}
