package hooks

import (
	"encoding/json"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

type call struct {
	agy      bool
	commands []string
	cwd      string
}

// shellTools are the tool names whose input runs in, or types into, a shell:
// Claude Code's Bash, Codex's Bash/exec_command/shell/write_stdin, pi's bash.
var shellTools = map[string]bool{
	"Bash": true, "bash": true, "shell": true, "local_shell": true, "exec_command": true, "write_stdin": true,
}

type payload struct {
	ToolName  string                     `json:"tool_name"`
	ToolInput map[string]json.RawMessage `json:"tool_input"`
	Cwd       string                     `json:"cwd"`
	ToolCall  *struct {
		Name string                     `json:"name"`
		Args map[string]json.RawMessage `json:"args"`
	} `json:"toolCall"`
}

func decode(data []byte) call {
	var p payload
	if json.Unmarshal(data, &p) != nil {
		return call{}
	}
	if p.ToolCall != nil {
		return decodeAgy(p.ToolCall.Name, p.ToolCall.Args)
	}
	c := call{cwd: p.Cwd}
	if !shellTools[p.ToolName] {
		return c
	}
	for _, key := range []string{"command", "cmd", "chars"} {
		if src, ok := commandText(p.ToolInput[key]); ok {
			c.commands = append(c.commands, src)
		}
	}
	return c
}

func decodeAgy(tool string, args map[string]json.RawMessage) call {
	c := call{agy: true}
	switch tool {
	case "run_command":
		c.commands = appendString(c.commands, args["CommandLine"])
		var cwd string
		if json.Unmarshal(args["Cwd"], &cwd) == nil {
			c.cwd = cwd
		}
	case "manage_task":
		var action string
		if json.Unmarshal(args["Action"], &action) == nil && action == "send_input" {
			c.commands = appendString(c.commands, args["Input"])
		}
	}
	return c
}

func appendString(list []string, raw json.RawMessage) []string {
	var s string
	if json.Unmarshal(raw, &s) == nil && s != "" {
		return append(list, s)
	}
	return list
}

func commandText(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s, s != ""
	}
	var argv []string
	if json.Unmarshal(raw, &argv) != nil || len(argv) == 0 {
		return "", false
	}
	quoted := make([]string, len(argv))
	for i, arg := range argv {
		q, err := syntax.Quote(arg, syntax.LangBash)
		if err != nil {
			q = "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'"
		}
		quoted[i] = q
	}
	return strings.Join(quoted, " "), true
}
