// Package hooks decides whether an agent's tool call may run.
package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/aryan-binazir/cc-config/agent-hooks/internal/shell"
)

type Options struct {
	Phase    string   // "pre" or "post"
	Stdin    []byte   // the hook payload
	Only     []string // run just these hooks; empty runs the whole catalog
	Agy      bool     // answer in Antigravity's JSON protocol
	StateDir string   // where overrides.log goes
	Home     string   // expands ~ in commands
}

// Result is what the process prints and exits with.
type Result struct {
	Exit   int
	Stdout string
	Stderr string
}

type hook struct {
	name      string
	phase     string
	override  string
	check     func(*ctx, shell.Invocation) string
	checkCall func(*ctx) string
}

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
	var selected []hook
	for _, h := range catalog {
		if h.phase == o.Phase && (len(o.Only) == 0 || slices.Contains(o.Only, h.name)) {
			selected = append(selected, h)
		}
	}
	if o.Phase == "pre" && slices.ContainsFunc(selected, func(h hook) bool { return h.check != nil }) && c.unreadCouldBeGuarded() {
		messages = append(messages, "[agent-hooks: unparsed] "+unparsedMessage)
		c.blocked = true
	}
	for _, h := range selected {
		if msg := c.evaluate(h); msg != "" {
			messages = append(messages, fmt.Sprintf("[agent-hooks: %s] %s", h.name, msg))
			c.blocked = true
		}
	}
	res = respond(o, messages)
	if c.logErr != nil {
		res.Stderr = strings.TrimSpace(res.Stderr + "\nagent-hooks: could not log override: " + c.logErr.Error())
	}
	return res
}

const unparsedMessage = "This command stops being valid bash partway through, and the part that cannot be read mentions git, gh, or rm, so the guardrails cannot check it. Fix the syntax, or run that part as its own command."

var guardedCommand = regexp.MustCompile("(^|[\\s;&|()<>'\"`/])(git|gh|rm)($|[\\s;&|()<>'\"`])")

// unreadCouldBeGuarded reports whether shell text the parser gave up on names
// a command the hooks guard. Typed input is exempt: it need not be shell.
func (c *ctx) unreadCouldBeGuarded() bool {
	for _, p := range c.parsed {
		if !p.typed && slices.ContainsFunc(p.unread, guardedCommand.MatchString) {
			return true
		}
	}
	return false
}

func (c *ctx) evaluate(h hook) string {
	if h.checkCall != nil {
		return h.checkCall(c)
	}
	for _, inv := range c.invs {
		c.unanswered = ""
		msg := h.check(c, inv)
		if msg == "" && c.unanswered != "" && h.override != "" {
			msg = fmt.Sprintf("Could not check this command: %s. Retry it; if it keeps failing and the user approved the command, rerun with AGENT_HOOKS_ALLOW=%s.", c.unanswered, h.override)
		}
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
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
	return strings.TrimSuffix(b.String(), "\n")
}

func (c *ctx) logOverride(hookName string, inv shell.Invocation) {
	dir := c.opts.StateDir
	if dir == "" {
		return
	}
	cwd := inv.Cwd
	if cwd == "" {
		cwd = c.call.cwd
	}
	line := mustJSON(map[string]string{
		"time":    time.Now().UTC().Format(time.RFC3339),
		"hook":    hookName,
		"cwd":     cwd,
		"command": c.call.text(),
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
