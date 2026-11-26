package shell

import "strings"

type wrapper struct {
	shortValues    string
	longValues     []string
	assigns        bool
	leadingOperand bool
}

var wrappers = map[string]wrapper{
	"env":      {shortValues: "uCS", longValues: []string{"--unset", "--chdir", "--split-string"}, assigns: true},
	"sudo":     {shortValues: "ugCDprtTURh", longValues: []string{"--user", "--group", "--chdir", "--prompt", "--role", "--type", "--other-user", "--close-from", "--host", "--chroot", "--command-timeout"}, assigns: true},
	"doas":     {shortValues: "uC"},
	"nohup":    {},
	"setsid":   {},
	"nice":     {shortValues: "n", longValues: []string{"--adjustment"}},
	"timeout":  {shortValues: "sk", longValues: []string{"--signal", "--kill-after"}, leadingOperand: true},
	"stdbuf":   {shortValues: "ioe", longValues: []string{"--input", "--output", "--error"}},
	"command":  {},
	"exec":     {shortValues: "a"},
	"time":     {shortValues: "fo", longValues: []string{"--format", "--output"}},
	"xargs":    {shortValues: "IinPdLsEae", longValues: []string{"--max-args", "--max-procs", "--delimiter", "--max-lines", "--max-chars", "--eof", "--arg-file", "--replace"}},
	"unbuffer": {},
}

func unwrap(name string, args []Word) (rest []Word, env map[string]string, chdir *Word, ok bool) {
	if name == "mise" {
		for i, a := range args {
			if a.Val == "--" && i > 0 && (args[0].Val == "exec" || args[0].Val == "x") {
				return args[i+1:], nil, nil, true
			}
		}
		return nil, nil, nil, false
	}
	spec, known := wrappers[name]
	if !known {
		return nil, nil, nil, false
	}
	if name == "command" && len(args) > 0 && (args[0].Val == "-v" || args[0].Val == "-V") {
		return nil, nil, nil, false
	}
	i := 0
options:
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
			opt, value, hasValue := strings.Cut(a, "=")
			if !hasValue && contains(spec.longValues, a) && i+1 < len(args) {
				i++
				value = args[i].Val
			}
			if name == "env" && opt == "--chdir" {
				chdir = &Word{Val: value, Dynamic: args[i].Dynamic}
			}
			if name == "env" && opt == "--split-string" {
				args, i = resplit(value, args[i+1:]), 0
				continue
			}
			i++
			continue
		}
		for j := 1; j < len(a); j++ {
			if strings.IndexByte(spec.shortValues, a[j]) >= 0 {
				value := Word{Val: a[j+1:], Dynamic: args[i].Dynamic}
				if j == len(a)-1 && i+1 < len(args) {
					i++
					value = args[i]
				}
				if name == "env" && a[j] == 'C' {
					chdir = &value
				}
				if name == "env" && a[j] == 'S' {
					args, i = resplit(value.Val, args[i+1:]), 0
					continue options
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
	if spec.leadingOperand && i < len(args) {
		i++
	}
	if i >= len(args) {
		return nil, nil, nil, false
	}
	return args[i:], env, chdir, true
}

// resplit is env's argument list after -S: env splits the string into words,
// options included, and carries on with what follows.
func resplit(value string, rest []Word) []Word {
	words, _ := Split(value)
	return append(words, rest...)
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
