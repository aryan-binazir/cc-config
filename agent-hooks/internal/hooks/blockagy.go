package hooks

import (
	"path"
	"regexp"
	"strings"
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

var launchers = map[string]bool{
	"bash": true, "sh": true, "zsh": true, "dash": true, "fish": true, "eval": true, "source": true, ".": true,
	"env": true, "exec": true, "command": true, "nohup": true, "setsid": true, "timeout": true, "stdbuf": true,
	"nice": true, "time": true, "sudo": true, "doas": true, "xargs": true, "parallel": true, "watch": true,
	"unbuffer": true, "script": true, "systemd-run": true, "tmux": true, "screen": true, "ssh": true, "mise": true,
}

var assignment = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)

func isAgy(word string) bool {
	word = strings.NewReplacer(`"`, "", "'", "", "`", "").Replace(word)
	word = word[strings.LastIndexByte(word, '=')+1:]
	word = word[strings.LastIndexByte(word, '/')+1:]
	return word == "agy" || word == "antigravity" || word == "acg"
}

func splitAny(s, seps string) [][]string {
	var out [][]string
	for _, seg := range strings.FieldsFunc(s, func(r rune) bool { return strings.ContainsRune(seps, r) }) {
		out = append(out, strings.Fields(seg))
	}
	return out
}

const separators = ";&|()`\n"

func checkAgy(c *ctx) string {
	text := strings.Join(c.call.commands, "\n")
	if text == "" {
		return ""
	}
	mentionedAnywhere := false
	for _, word := range strings.Fields(strings.Map(func(r rune) rune {
		if strings.ContainsRune(separators, r) {
			return ' '
		}
		return r
	}, text)) {
		if isAgy(word) {
			mentionedAnywhere = true
			break
		}
	}
	launches := func(words []string) bool {
		if len(words) == 0 {
			return false
		}
		if isAgy(words[0]) {
			return true
		}
		first := path.Base(strings.NewReplacer(`"`, "", "'", "").Replace(words[0]))
		if !launchers[first] || first == "command" && len(words) > 1 && (words[1] == "-v" || words[1] == "-V") {
			return false
		}
		return mentionedAnywhere
	}
	if c.allParsed {
		for _, inv := range c.invs {
			if launches(inv.Values()) {
				return agyMessage
			}
		}
		return ""
	}
	for _, words := range splitAny(text, separators) {
		for len(words) > 0 && assignment.MatchString(words[0]) {
			words = words[1:]
		}
		if launches(words) {
			return agyMessage
		}
	}
	return ""
}
