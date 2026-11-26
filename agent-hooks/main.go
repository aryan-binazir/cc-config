// Command agent-hooks enforces agent guardrails as PreToolUse and PostToolUse
// hooks in Claude Code, Codex, and the Antigravity CLI.
//
//	agent-hooks pre [--agy] [--only=<hook>]   read a hook payload on stdin
//	agent-hooks post [--agy] [--only=<hook>]
//	agent-hooks install | uninstall           wire into each harness's config
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/aryan-binazir/cc-config/agent-hooks/internal/hooks"
	"github.com/aryan-binazir/cc-config/agent-hooks/internal/install"
)

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
	switch {
	case args[0] == "pre" || args[0] == "post":
		return check(args[0], args[1:], home, stdin, stdout, stderr)
	case (args[0] == "install" || args[0] == "uninstall") && len(args) == 1:
		if homeErr != nil {
			fmt.Fprintln(stderr, "agent-hooks: cannot find the home directory:", homeErr)
			return 1
		}
		return wire(args[0], home, stdout, stderr)
	}
	fmt.Fprintln(stderr, usage)
	return 64
}

func check(phase string, flags []string, home string, stdin io.Reader, stdout, stderr io.Writer) int {
	o := hooks.Options{Phase: phase, Home: home, StateDir: stateDir(home)}
	misuse := ""
	for _, f := range flags {
		name, isOnly := strings.CutPrefix(f, "--only=")
		switch {
		case f == "--agy":
			o.Agy = true
		case isOnly && len(o.Only) > 0:
			misuse = "--only takes one hook"
		case isOnly && slices.Contains(hooks.Names(), name):
			o.Only = []string{name}
		case isOnly:
			misuse = fmt.Sprintf("no hook named %q", name)
		default:
			misuse = "unknown argument " + f
		}
	}
	allowWith := func(exit int, reason string) int {
		if o.Agy && phase == "pre" {
			fmt.Fprintln(stdout, `{"decision":"allow"}`)
		}
		fmt.Fprintln(stderr, "agent-hooks:", reason)
		return exit
	}
	if misuse != "" {
		return allowWith(64, misuse+"\n"+usage)
	}
	time.AfterFunc(deadline, func() { os.Exit(allowWith(1, "timed out; allowing the call")) })
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

func wire(action, home string, stdout, stderr io.Writer) int {
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
