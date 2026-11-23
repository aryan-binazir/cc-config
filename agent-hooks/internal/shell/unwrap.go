package shell

import "strings"

// wrapper describes a command that runs its operands as another command.
type wrapper struct {
	shortValues string   // short options that take a value
	longValues  []string // long options that take a value when not written --opt=value
	assigns     bool     // NAME=value operands before the command set its environment
	operand     bool     // one operand (a duration or priority) precedes the command
}

var wrappers = map[string]wrapper{
	"env":      {shortValues: "uCS", longValues: []string{"--unset", "--chdir", "--split-string"}, assigns: true},
	"sudo":     {shortValues: "ugCDprtTURh", longValues: []string{"--user", "--group", "--chdir", "--prompt", "--role", "--type", "--other-user", "--close-from", "--host", "--chroot", "--command-timeout"}, assigns: true},
	"doas":     {shortValues: "uC"},
	"nohup":    {},
	"setsid":   {},
	"nice":     {shortValues: "n", longValues: []string{"--adjustment"}},
	"timeout":  {shortValues: "sk", longValues: []string{"--signal", "--kill-after"}, operand: true},
	"stdbuf":   {shortValues: "ioe", longValues: []string{"--input", "--output", "--error"}},
	"command":  {},
	"exec":     {shortValues: "a"},
	"time":     {shortValues: "fo", longValues: []string{"--format", "--output"}},
	"xargs":    {shortValues: "IinPdLsEae", longValues: []string{"--max-args", "--max-procs", "--delimiter", "--max-lines", "--max-chars", "--eof", "--arg-file", "--replace"}},
	"unbuffer": {},
}

// unwrap returns the command a wrapper runs, with any environment it sets.
func unwrap(name string, args []Word) (rest []Word, env map[string]string, ok bool) {
	if name == "mise" {
		for i, a := range args {
			if a.Val == "--" && i > 0 && (args[0].Val == "exec" || args[0].Val == "x") {
				return args[i+1:], nil, true
			}
		}
		return nil, nil, false
	}
	spec, known := wrappers[name]
	if !known {
		return nil, nil, false
	}
	if name == "command" && len(args) > 0 && (args[0].Val == "-v" || args[0].Val == "-V") {
		return nil, nil, false
	}
	i := 0
	for i < len(args) {
		a := args[i].Val
		if a == "--" {
			i++
			break
		}
		if !strings.HasPrefix(a, "-") || a == "-" {
			break
		}
		if strings.HasPrefix(a, "--") {
			if !strings.Contains(a, "=") && contains(spec.longValues, a) {
				i++
			}
			i++
			continue
		}
		// A short cluster: the first value-taking letter consumes the rest or the next word.
		for j := 1; j < len(a); j++ {
			if strings.IndexByte(spec.shortValues, a[j]) >= 0 {
				if j == len(a)-1 {
					i++
				}
				break
			}
		}
		i++
	}
	if spec.assigns {
		for i < len(args) && isAssign(args[i].Val) {
			if env == nil {
				env = map[string]string{}
			}
			k, v, _ := strings.Cut(args[i].Val, "=")
			env[k] = v
			i++
		}
	}
	if spec.operand && i < len(args) {
		i++
	}
	if i >= len(args) {
		return nil, nil, false
	}
	return args[i:], env, true
}

func isAssign(s string) bool {
	name, _, ok := strings.Cut(s, "=")
	if !ok || name == "" {
		return false
	}
	for i, c := range name {
		if !(c == '_' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || i > 0 && c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
