// Command agent-hooks enforces agent guardrails as PreToolUse and PostToolUse
// hooks in Claude Code, Codex, and the Antigravity CLI.
//
//	agent-hooks pre [--agy] [--only=<hook>]   read a hook payload on stdin
//	agent-hooks post [--agy] [--only=<hook>]
//	agent-hooks install | uninstall           wire into each harness's config
package main

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aryan-binazir/cc-config/agent-hooks/internal/hooks"
	"github.com/aryan-binazir/cc-config/agent-hooks/internal/install"
)

//go:embed hooks.yaml
var hooksYAML string

const (
	maxPayload = 16 << 20
	deadline   = 8 * time.Second
	usage      = "usage: agent-hooks pre|post [--agy] [--only=<hook>] | install | uninstall"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, usage)
		return 64
	}
	home, homeErr := os.UserHomeDir()
	switch args[0] {
	case "pre", "post":
		return check(args[0], args[1:], hooksYAML, home, stdin, stdout, stderr)
	case "install", "uninstall":
		if homeErr != nil {
			fmt.Fprintln(stderr, "agent-hooks: cannot find the home directory:", homeErr)
			return 1
		}
		return wire(args[0], hooksYAML, home, stdout, stderr)
	}
	fmt.Fprintln(stderr, usage)
	return 64
}

func check(phase string, flags []string, switches, home string, stdin io.Reader, stdout, stderr io.Writer) int {
	o := hooks.Options{Phase: phase, Home: home, StateDir: stateDir(home)}
	for _, f := range flags {
		switch {
		case f == "--agy":
			o.Agy = true
		case strings.HasPrefix(f, "--only="):
			o.Only = strings.TrimPrefix(f, "--only=")
		}
	}
	// Never hang the agent: past the deadline, allow the call and say why.
	time.AfterFunc(deadline, func() {
		if o.Agy && phase == "pre" {
			fmt.Fprintln(stdout, `{"decision":"allow"}`)
		}
		fmt.Fprintln(stderr, "agent-hooks: timed out; allowing the call")
		os.Exit(1)
	})
	enabled, err := parseSwitches(switches)
	if err != nil {
		// A fault, not a silent allow: Claude Code and Codex show it, and pi fails closed.
		if o.Agy && phase == "pre" {
			fmt.Fprintln(stdout, `{"decision":"allow"}`)
		}
		fmt.Fprintln(stderr, "agent-hooks: hooks.yaml:", err)
		return 1
	}
	o.Enabled = enabled
	o.Stdin, _ = io.ReadAll(io.LimitReader(stdin, maxPayload))
	res := hooks.Run(o)
	io.WriteString(stdout, res.Stdout)
	io.WriteString(stderr, res.Stderr)
	return res.Exit
}

func stateDir(home string) string {
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return filepath.Join(dir, "agent-hooks")
	}
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".local", "state", "agent-hooks")
}

func wire(action, switches, home string, stdout, stderr io.Writer) int {
	if _, err := parseSwitches(switches); err != nil && action == "install" {
		fmt.Fprintln(stderr, "agent-hooks: refusing to install: hooks.yaml:", err)
		return 1
	}
	paths := install.DefaultPaths(home)
	var changed []string
	var err error
	if action == "install" {
		bin, e := os.Executable()
		if e == nil {
			bin, e = filepath.EvalSymlinks(bin)
		}
		if e != nil {
			fmt.Fprintln(stderr, "agent-hooks: cannot find this binary:", e)
			return 1
		}
		changed, err = install.Install(paths, bin, time.Now())
	} else {
		changed, err = install.Uninstall(paths, time.Now())
	}
	for _, path := range changed {
		fmt.Fprintln(stdout, "updated", path)
	}
	if err != nil {
		fmt.Fprintln(stderr, "agent-hooks:", err)
		return 1
	}
	if len(changed) == 0 {
		fmt.Fprintln(stdout, "already up to date")
	}
	if action == "install" {
		fmt.Fprintln(stdout, "Codex runs a hook only after you trust it: open Codex, run /hooks, and approve the agent-hooks PreToolUse and PostToolUse entries.")
	}
	return 0
}

// parseSwitches reads hooks.yaml: a top-level `hooks:` mapping of hook names
// to true or false, with comments. It accepts nothing else.
func parseSwitches(text string) (map[string]bool, error) {
	on := map[string]bool{}
	inHooks := false
	for n, line := range strings.Split(text, "\n") {
		if i := strings.Index(line, "#"); i >= 0 {
			line = line[:i]
		}
		line = strings.TrimRight(line, " \r")
		switch {
		case strings.TrimSpace(line) == "":
		case line == "hooks:" && !inHooks && len(on) == 0:
			inHooks = true
		case inHooks && strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "   "):
			name, value, ok := strings.Cut(strings.TrimSpace(line), ":")
			value = strings.TrimSpace(value)
			if !ok || name == "" || (value != "true" && value != "false") {
				return nil, fmt.Errorf("line %d: want `  <hook>: true|false`", n+1)
			}
			on[name] = value == "true"
		default:
			return nil, fmt.Errorf("line %d: want a single top-level hooks: mapping", n+1)
		}
	}
	if !inHooks {
		return nil, fmt.Errorf("missing hooks: mapping")
	}
	return on, nil
}
