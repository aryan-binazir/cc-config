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
	binaryName  = "agent-hooks"
	agyHookName = "agent-hooks"
	timeout     = 10
)

var (
	// owned matches the commands Install writes: a single-quoted path ending in
	// /agent-hooks, then pre or post.
	owned = regexp.MustCompile(`^'(?:[^']|'\\'')*/agent-hooks' (pre|post)( |$)`)
	// legacy matches the hook commands that ran block-agy.sh.
	legacy = regexp.MustCompile(`^((ba)?sh\s+)?['"]?[^'"\s]*/block-agy\.sh['"]?$`)
)

// shellQuote single-quotes s for the sh -c every harness runs hooks through.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// Install adds this binary's hooks to every config, updating earlier copies in
// place and removing the legacy block-agy.sh entries. It returns the files it
// changed.
func Install(p Paths, bin string, now time.Time) ([]string, error) {
	if filepath.Base(bin) != binaryName {
		return nil, fmt.Errorf("the binary must be named %s so uninstall can find its entries, not %s", binaryName, filepath.Base(bin))
	}
	command := func(args string) string { return shellQuote(bin) + " " + args }
	wire := func(root *object) (bool, bool, error) {
		before := encode(root)
		hooks := hooksOf(root)
		removeLegacy(hooks)
		upsert(hooks, "PreToolUse", "Bash", command("pre"))
		upsert(hooks, "PostToolUse", "Bash", command("post"))
		return encode(root) != before, false, nil
	}
	return apply(now, []edit{
		{p.Claude, true, wire},
		{p.Codex, true, wire},
		{p.Agy, true, func(root *object) (bool, bool, error) {
			if v, ok := root.get(agyHookName); ok && !ownsAgyHook(v) {
				return false, false, errors.New("an agy hook named agent-hooks exists and is not ours; rename or remove it first")
			}
			before := encode(root)
			spec := newObject()
			spec.set("PreToolUse", []any{group("run_command|manage_task", command("pre --agy"))})
			root.set(agyHookName, spec)
			return encode(root) != before, false, nil
		}},
	})
}

// Uninstall removes this binary's hooks and the legacy entries, leaving
// everything else as it was. It never creates a config, and it deletes an
// Antigravity hooks file only when removing our hook emptied it.
func Uninstall(p Paths, now time.Time) ([]string, error) {
	unwire := func(root *object) (bool, bool, error) {
		v, _ := root.get("hooks")
		hooks, ok := v.(*object)
		if !ok || !strip(hooks) {
			return false, false, nil
		}
		if len(hooks.keys) == 0 {
			root.del("hooks")
		}
		return true, false, nil
	}
	return apply(now, []edit{
		{p.Claude, false, unwire},
		{p.Codex, false, unwire},
		{p.Agy, false, func(root *object) (bool, bool, error) {
			v, ok := root.get(agyHookName)
			if !ok || !ownsAgyHook(v) {
				return false, false, nil
			}
			root.del(agyHookName)
			return true, len(root.keys) == 0, nil
		}},
	})
}

// edit changes a parsed config and reports whether it changed anything and
// whether the file should be deleted.
type edit struct {
	path   string
	create bool // write the file when it does not exist yet
	change func(*object) (changed, remove bool, err error)
}

type pending struct {
	path     string // the real file, after following symlinks
	link     bool   // the configured path is a symlink
	existed  bool
	mode     fs.FileMode
	original []byte
	updated  string
	remove   bool
}

func apply(now time.Time, edits []edit) ([]string, error) {
	var plans []pending
	for _, e := range edits {
		plan, changed, err := prepare(e)
		if err != nil {
			return nil, err
		}
		if changed {
			plans = append(plans, plan)
		}
	}
	var changed []string
	stamp := now.UTC().Format("20060102T150405Z")
	for _, plan := range plans {
		if plan.existed {
			if err := backup(plan, stamp); err != nil {
				return changed, err
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

// backup writes the original next to the file under a name no earlier backup has.
func backup(plan pending, stamp string) error {
	name := plan.path + ".bak-" + stamp
	for n := 2; ; n++ {
		f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, plan.mode)
		if errors.Is(err, fs.ErrExist) {
			name = fmt.Sprintf("%s.bak-%s-%d", plan.path, stamp, n)
			continue
		}
		if err != nil {
			return fmt.Errorf("backing up %s: %w", plan.path, err)
		}
		_, err = f.Write(plan.original)
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		return err
	}
}

func prepare(e edit) (pending, bool, error) {
	plan := pending{path: e.path, mode: 0o644}
	if info, err := os.Lstat(e.path); err == nil && info.Mode()&fs.ModeSymlink != 0 {
		plan.link = true
		if real, err := filepath.EvalSymlinks(e.path); err == nil {
			plan.path = real
		}
	}
	data, err := os.ReadFile(plan.path)
	root := newObject()
	trailing := "\n"
	switch {
	case errors.Is(err, fs.ErrNotExist):
		if !e.create {
			return plan, false, nil
		}
	case err != nil:
		return plan, false, err
	default:
		plan.existed, plan.original = true, data
		if info, err := os.Stat(plan.path); err == nil {
			plan.mode = info.Mode().Perm()
		}
		if strings.TrimSpace(string(data)) != "" {
			if root, err = parseObject(data); err != nil {
				return plan, false, fmt.Errorf("%s is not valid JSON, so it was left alone: %w", plan.path, err)
			}
		}
		if !strings.HasSuffix(string(data), "\n") {
			trailing = ""
		}
	}
	changed, remove, err := e.change(root)
	if err != nil {
		return plan, false, fmt.Errorf("%s: %w", plan.path, err)
	}
	if !changed {
		return plan, false, nil
	}
	// A symlinked config keeps its link: empty it rather than deleting the target.
	plan.remove = remove && !plan.link
	plan.updated = encode(root) + trailing
	return plan, true, nil
}

func ownsAgyHook(v any) bool {
	spec, ok := v.(*object)
	if !ok {
		return false
	}
	found := false
	for _, event := range spec.keys {
		groups, _ := spec.vals[event].([]any)
		for _, g := range groups {
			group, _ := g.(*object)
			if group == nil {
				return false
			}
			hv, _ := group.get("hooks")
			handlers, _ := hv.([]any)
			for _, h := range handlers {
				if !isOwned(h) {
					return false
				}
				found = true
			}
		}
	}
	return found
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

// upsert points the first owned handler of event at command, in place so
// Codex's index-keyed trust for other groups is untouched, and removes any
// other owned handlers. With none, it appends a new group.
func upsert(hooks *object, event, matcher, command string) {
	v, _ := hooks.get(event)
	groups, _ := v.([]any)
	placed := false
	var kept []any
	for _, g := range groups {
		group, _ := g.(*object)
		var handlers []any
		if group != nil {
			hv, _ := group.get("hooks")
			handlers, _ = hv.([]any)
		}
		var keep []any
		for _, h := range handlers {
			switch {
			case !isOwned(h):
				keep = append(keep, h)
			case !placed:
				h.(*object).set("command", command)
				keep = append(keep, h)
				group.set("matcher", matcher)
				placed = true
			}
		}
		if len(handlers) > 0 && len(keep) == 0 {
			continue
		}
		if len(keep) != len(handlers) {
			group.set("hooks", keep)
		}
		kept = append(kept, g)
	}
	if !placed {
		kept = append(kept, group(matcher, command))
	}
	hooks.set(event, kept)
}

// removeLegacy drops block-agy.sh handlers, and groups and events they emptied.
func removeLegacy(hooks *object) { filter(hooks, isLegacy) }

// strip removes owned and legacy handlers and reports whether it removed any.
func strip(hooks *object) bool {
	return filter(hooks, func(h any) bool { return isOwned(h) || isLegacy(h) })
}

// filter removes handlers matching drop. Groups and events are deleted only
// when that removal emptied them. It reports whether it removed anything.
func filter(hooks *object, drop func(any) bool) bool {
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
				if drop(h) {
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

func commandOf(h any) string {
	handler, ok := h.(*object)
	if !ok {
		return ""
	}
	v, _ := handler.get("command")
	cmd, _ := v.(string)
	return cmd
}

func isOwned(h any) bool  { return owned.MatchString(commandOf(h)) }
func isLegacy(h any) bool { return legacy.MatchString(commandOf(h)) }

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
