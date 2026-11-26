package hooks

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/aryan-binazir/cc-config/agent-hooks/internal/shell"
)

// Lookup limits. Together they stay under the process deadline in main.go, so
// a slow lookup is reported instead of timing the whole hook out.
var (
	gitTimeout   = time.Second
	ghTimeout    = 2 * time.Second
	lookupBudget = 6 * time.Second
)

type ctx struct {
	opts    Options
	call    call
	parsed  []parsed
	invs    []shell.Invocation // every source's invocations
	blocked bool
	cache   map[string]lookup
	stopAt  time.Time
	// unanswered says why the current check could not learn what it asked.
	unanswered string
	logErr     error
}

type parsed struct {
	source
	invs   []shell.Invocation
	unread []string
}

type outcome int

const (
	answered outcome = iota
	// missed: the program ran and exited non-zero, which is how git says no
	// (unset config, missing ref, detached HEAD, not a repository).
	missed
	// skipped: not attempted, or it could not start. The real command would
	// fail the same way, or there is nothing to compare it against.
	skipped
	// failed: it ran out of time or output room, so there is no answer.
	failed
)

type lookup struct {
	out string
	res outcome
	why string
}

func newCtx(c call, o Options) *ctx {
	x := &ctx{opts: o, call: c, stopAt: time.Now().Add(lookupBudget)}
	for _, src := range c.sources {
		invs, unread := shell.Parse(src.text, c.cwd, o.Home)
		x.parsed = append(x.parsed, parsed{source: src, invs: invs, unread: unread})
		x.invs = append(x.invs, invs...)
	}
	return x
}

func (c *ctx) git(g *gitCmd, args ...string) (string, bool) {
	safe := []string{"-c", "core.fsmonitor=false", "-c", "core.untrackedCache=false"}
	out, res := c.run("git "+args[0], gitTimeout, g.dir, g.env, "git", append(safe, args...)...)
	return out, res == answered
}

func (c *ctx) gh(dir string, env []string, args ...string) (string, outcome) {
	return c.run("gh "+strings.Join(args[:2], " "), ghTimeout, dir, env, "gh", args...)
}

func (c *ctx) run(label string, timeout time.Duration, dir string, env []string, name string, args ...string) (string, outcome) {
	if c.blocked || dir == "" {
		return "", skipped
	}
	key := dir + "\x00" + strings.Join(env, "\x00") + "\x00" + name + "\x00" + strings.Join(args, "\x00")
	hit, ok := c.cache[key]
	if !ok {
		hit = c.exec(label, timeout, dir, env, name, args...)
		if c.cache == nil {
			c.cache = map[string]lookup{}
		}
		c.cache[key] = hit
	}
	if hit.res == failed {
		c.unanswered = hit.why
	}
	return hit.out, hit.res
}

func (c *ctx) exec(label string, timeout time.Duration, dir string, env []string, name string, args ...string) lookup {
	left := time.Until(c.stopAt)
	if left <= 0 {
		return lookup{res: failed, why: "the hook ran out of time before " + label}
	}
	cx, cancel := context.WithTimeout(context.Background(), min(timeout, left))
	defer cancel()
	cmd := exec.CommandContext(cx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(append(cmd.Environ(), "GIT_OPTIONAL_LOCKS=0", "GH_PROMPT_DISABLED=1", "GIT_TERMINAL_PROMPT=0"), env...)
	cmd.WaitDelay = 100 * time.Millisecond // a child holding stdout open cannot outlive the timeout
	var out bytes.Buffer
	capped := &limited{w: &out, n: 64 << 20}
	cmd.Stdout = capped
	if cmd.Start() != nil {
		return lookup{res: skipped}
	}
	err := cmd.Wait()
	var exit *exec.ExitError
	switch {
	case cx.Err() != nil && left < timeout:
		return lookup{res: failed, why: "the hook ran out of time during " + label}
	case cx.Err() != nil:
		return lookup{res: failed, why: fmt.Sprintf("%s timed out after %s", label, timeout)}
	case capped.overflow:
		return lookup{res: failed, why: label + " printed more than the hook can read"}
	case err == nil:
		return lookup{out: out.String(), res: answered}
	case errors.As(err, &exit):
		return lookup{res: missed}
	}
	return lookup{res: failed, why: label + " did not finish: " + err.Error()}
}

type limited struct {
	w        *bytes.Buffer
	n        int
	overflow bool
}

func (l *limited) Write(p []byte) (int, error) {
	n := len(p)
	room := l.n - l.w.Len()
	if len(p) > room {
		l.overflow = true
		p = p[:max(room, 0)]
	}
	l.w.Write(p)
	return n, nil
}
