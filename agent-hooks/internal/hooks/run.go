// Package hooks decides whether an agent's tool call may run.
package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aryan-binazir/cc-config/agent-hooks/internal/shell"
)

// Options configure one hook run.
type Options struct {
	Phase    string          // "pre" or "post"
	Stdin    []byte          // the hook payload
	Enabled  map[string]bool // hook name -> on
	Only     string          // run just this hook, if set
	Agy      bool            // answer in Antigravity's JSON protocol
	StateDir string          // where overrides.log goes
	Home     string          // expands ~ in commands
}

// Result is what the process prints and exits with.
type Result struct {
	Exit   int
	Stdout string
	Stderr string
}

// hook checks one invocation and returns a message when it objects.
type hook struct {
	name     string
	phase    string
	override string // the AGENT_HOOKS_ALLOW name that lets it through; "" for none
	check    func(*ctx, shell.Invocation) string
	// whole checks the call once instead of per invocation (block-agy).
	whole func(*ctx) string
}

// Catalog lists every hook in the order they run. Cheap payload-only checks
// come before ones that may need repository state.
var catalog = []hook{
	blockAgy,
	draftPrs, prAssignee, noAdminMerge, mergeHeadMatch,
	commitFormat, noSecretsCommit, destructiveGit, noProduction,
	rgReplaceFlag,
}

// Names returns every hook name.
func Names() []string {
	names := make([]string, len(catalog))
	for i, h := range catalog {
		names[i] = h.name
	}
	return names
}

// Run evaluates the payload. A malformed payload or an internal fault allows
// the call: faults exit 1, which Claude Code and Codex report without blocking.
func Run(o Options) (res Result) {
	defer func() {
		if r := recover(); r != nil {
			res = Result{Exit: 1, Stderr: fmt.Sprintf("agent-hooks: internal error: %v", r)}
			if o.Agy {
				res = respond(o, nil)
			}
		}
	}()
	c := newCtx(decode(o.Stdin), o)
	if c.call.agy {
		o.Agy = true
	}
	var messages []string
	for _, h := range catalog {
		if h.phase != o.Phase || !o.Enabled[h.name] || (o.Only != "" && o.Only != h.name) {
			continue
		}
		if msg := c.evaluate(h); msg != "" {
			messages = append(messages, fmt.Sprintf("[agent-hooks: %s] %s", h.name, msg))
			c.lookupsOff = true
		}
	}
	res = respond(o, messages)
	if c.logErr != nil {
		res.Stderr = strings.TrimSpace(res.Stderr + "\nagent-hooks: could not log override: " + c.logErr.Error())
	}
	return res
}

func (c *ctx) evaluate(h hook) string {
	if h.whole != nil {
		return h.whole(c)
	}
	for _, inv := range c.invs {
		msg := h.check(c, inv)
		if msg == "" {
			continue
		}
		if h.override != "" && allows(inv, h.override) {
			c.logOverride(h.name, inv)
			continue
		}
		return msg
	}
	return ""
}

func allows(inv shell.Invocation, name string) bool {
	for _, v := range strings.Split(inv.Assigns["AGENT_HOOKS_ALLOW"], ",") {
		if strings.TrimSpace(v) == name {
			return true
		}
	}
	return false
}

func respond(o Options, messages []string) Result {
	text := strings.Join(messages, "\n\n")
	switch {
	case o.Phase == "pre" && o.Agy:
		decision := map[string]string{"decision": "allow"}
		if text != "" {
			decision = map[string]string{"decision": "deny", "reason": text}
		}
		return Result{Stdout: mustJSON(decision)}
	case o.Phase == "pre" && text != "":
		return Result{Exit: 2, Stderr: text}
	case o.Phase == "post" && o.Agy:
		return Result{Stdout: "{}"}
	case o.Phase == "post" && text != "":
		return Result{Stdout: mustJSON(map[string]any{
			"hookSpecificOutput": map[string]string{"hookEventName": "PostToolUse", "additionalContext": text},
		})}
	}
	return Result{}
}

func mustJSON(v any) string {
	data, _ := json.Marshal(v)
	return string(data)
}

func (c *ctx) logOverride(hookName string, inv shell.Invocation) {
	dir := c.opts.StateDir
	if dir == "" {
		return
	}
	line := mustJSON(map[string]string{
		"time":    time.Now().UTC().Format(time.RFC3339),
		"hook":    hookName,
		"cwd":     inv.Cwd,
		"command": strings.Join(c.call.commands, "\n"),
	}) + "\n"
	if err := os.MkdirAll(dir, 0o700); err != nil {
		c.logErr = err
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, "overrides.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		c.logErr = err
		return
	}
	defer f.Close()
	if _, err := f.WriteString(line); err != nil {
		c.logErr = err
	}
}
