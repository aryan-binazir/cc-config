// Package install wires agent-hooks into the Claude Code, Codex, and
// Antigravity CLI hook configs, and takes it back out.
package install

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Paths are the hook config files of each harness.
type Paths struct {
	Claude string // ~/.claude/settings.json
	Codex  string // ~/.codex/hooks.json
	Agy    string // ~/.gemini/config/hooks.json
}

// DefaultPaths returns the user-level configs under home.
func DefaultPaths(home string) Paths {
	return Paths{
		Claude: filepath.Join(home, ".claude", "settings.json"),
		Codex:  filepath.Join(home, ".codex", "hooks.json"),
		Agy:    filepath.Join(home, ".gemini", "config", "hooks.json"),
	}
}

const (
	agyHookName = "agent-hooks"
	timeout     = 10
)

var (
	owned  = regexp.MustCompile(`^'?[^']*/agent-hooks'? (pre|post)( |$)`)
	legacy = regexp.MustCompile(`block-agy\.sh`)
)

// Install adds this binary's hooks to every config, replacing earlier copies
// and the legacy block-agy.sh entries. It returns the files it changed.
func Install(p Paths, bin string, now time.Time) ([]string, error) {
	command := func(args string) string { return "'" + bin + "' " + args }
	return apply(now, []edit{
		{p.Claude, func(root *object) bool {
			unwire(root)
			hooks := hooksOf(root)
			addGroup(hooks, "PreToolUse", "Bash", command("pre"))
			addGroup(hooks, "PostToolUse", "Bash", command("post"))
			return false
		}},
		{p.Codex, func(root *object) bool {
			unwire(root)
			hooks := hooksOf(root)
			addGroup(hooks, "PreToolUse", "Bash", command("pre"))
			addGroup(hooks, "PostToolUse", "Bash", command("post"))
			return false
		}},
		{p.Agy, func(root *object) bool {
			spec := newObject()
			spec.set("PreToolUse", []any{group("run_command|manage_task", command("pre --agy"))})
			root.set(agyHookName, spec)
			return false
		}},
	})
}

// Uninstall removes this binary's hooks and the legacy entries, leaving
// everything else as it was. An Antigravity hooks file left empty is deleted.
func Uninstall(p Paths, now time.Time) ([]string, error) {
	return apply(now, []edit{
		{p.Claude, func(root *object) bool { unwire(root); return false }},
		{p.Codex, func(root *object) bool { unwire(root); return false }},
		{p.Agy, func(root *object) bool {
			root.del(agyHookName)
			return len(root.keys) == 0
		}},
	})
}

// edit changes a parsed config and reports whether the file should be deleted.
type edit struct {
	path   string
	change func(*object) (remove bool)
}

type pending struct {
	path     string // the real file, after following symlinks
	existed  bool
	mode     fs.FileMode
	original []byte
	updated  string
	remove   bool
}

func apply(now time.Time, edits []edit) ([]string, error) {
	var plans []pending
	for _, e := range edits {
		plan, err := prepare(e)
		if err != nil {
			return nil, err
		}
		if plan.remove && plan.existed || !plan.remove && plan.updated != string(plan.original) {
			plans = append(plans, plan)
		}
	}
	var changed []string
	stamp := now.UTC().Format("20060102T150405Z")
	for _, plan := range plans {
		if plan.existed {
			if err := os.WriteFile(plan.path+".bak-"+stamp, plan.original, plan.mode); err != nil {
				return changed, fmt.Errorf("backing up %s: %w", plan.path, err)
			}
		}
		var err error
		if plan.remove {
			err = os.Remove(plan.path)
		} else {
			err = writeAtomic(plan.path, plan.updated, plan.mode)
		}
		if err != nil {
			return changed, fmt.Errorf("writing %s: %w", plan.path, err)
		}
		changed = append(changed, plan.path)
	}
	return changed, nil
}

func prepare(e edit) (pending, error) {
	plan := pending{path: e.path, mode: 0o644}
	if real, err := filepath.EvalSymlinks(e.path); err == nil {
		plan.path = real
	}
	data, err := os.ReadFile(plan.path)
	root := newObject()
	trailing := "\n"
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return plan, err
	default:
		plan.existed, plan.original = true, data
		if info, err := os.Stat(plan.path); err == nil {
			plan.mode = info.Mode().Perm()
		}
		if strings.TrimSpace(string(data)) != "" {
			if root, err = parseObject(data); err != nil {
				return plan, fmt.Errorf("%s is not valid JSON, so it was left alone: %w", plan.path, err)
			}
		}
		if !strings.HasSuffix(string(data), "\n") {
			trailing = ""
		}
	}
	plan.remove = e.change(root)
	plan.updated = encode(root) + trailing
	return plan, nil
}

func writeAtomic(path, content string, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".agent-hooks-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// hooksOf returns root.hooks, creating it if needed.
func hooksOf(root *object) *object {
	v, _ := root.get("hooks")
	hooks, ok := v.(*object)
	if !ok {
		hooks = newObject()
		root.set("hooks", hooks)
	}
	return hooks
}

// unwire removes owned and legacy handlers from root.hooks. Groups, events,
// and the hooks object are dropped only when that removal emptied them.
func unwire(root *object) {
	v, _ := root.get("hooks")
	hooks, ok := v.(*object)
	if ok && strip(hooks) && len(hooks.keys) == 0 {
		root.del("hooks")
	}
}

// strip removes owned and legacy handlers and reports whether it removed any.
func strip(hooks *object) bool {
	removedAny := false
	for _, event := range append([]string(nil), hooks.keys...) {
		v, _ := hooks.get(event)
		groups, ok := v.([]any)
		if !ok {
			continue
		}
		var kept []any
		removed := false
		for _, g := range groups {
			group, _ := g.(*object)
			var handlers []any
			if group != nil {
				hv, _ := group.get("hooks")
				handlers, _ = hv.([]any)
			}
			var keep []any
			for _, h := range handlers {
				if isOurs(h) {
					removed = true
				} else {
					keep = append(keep, h)
				}
			}
			switch {
			case len(keep) == len(handlers):
				kept = append(kept, g)
			case len(keep) > 0:
				group.set("hooks", keep)
				kept = append(kept, g)
			}
		}
		if !removed {
			continue
		}
		removedAny = true
		if len(kept) == 0 {
			hooks.del(event)
		} else {
			hooks.set(event, kept)
		}
	}
	return removedAny
}

func isOurs(h any) bool {
	handler, ok := h.(*object)
	if !ok {
		return false
	}
	v, _ := handler.get("command")
	cmd, _ := v.(string)
	return owned.MatchString(cmd) || legacy.MatchString(cmd)
}

func handler(command string) *object {
	h := newObject()
	h.set("type", "command")
	h.set("command", command)
	h.set("timeout", timeout)
	return h
}

func group(matcher, command string) *object {
	g := newObject()
	g.set("matcher", matcher)
	g.set("hooks", []any{handler(command)})
	return g
}

func addGroup(hooks *object, event, matcher, command string) {
	v, _ := hooks.get(event)
	groups, _ := v.([]any)
	hooks.set(event, append(groups, group(matcher, command)))
}
