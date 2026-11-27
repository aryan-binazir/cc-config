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
	"time"

	"github.com/aryan-binazir/cc-config/agent-hooks/internal/hooks"
	"github.com/aryan-binazir/cc-config/agent-hooks/internal/install"
)

const usage = "usage: agent-hooks pre|post [--agy] [--only=<hook>] | install | uninstall"

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
		exit := hooks.Command(args[0], args[1:], home, stdin, stdout, stderr)
		if exit == 64 {
			fmt.Fprintln(stderr, usage)
		}
		return exit
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
