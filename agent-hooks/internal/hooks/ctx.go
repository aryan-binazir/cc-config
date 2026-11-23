package hooks

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/aryan-binazir/cc-config/agent-hooks/internal/shell"
)

const (
	gitTimeout = time.Second
	ghTimeout  = 2 * time.Second
)

// ctx is the state shared by the hooks evaluating one call.
type ctx struct {
	opts       Options
	call       call
	invs       []shell.Invocation
	parsed     bool // every command parsed as shell
	lookupsOff bool // a hook already blocked, so skip slow lookups
	cache      map[string]lookup
	logErr     error
}

type lookup struct {
	out string
	ok  bool
}

func newCtx(c call, o Options) *ctx {
	x := &ctx{opts: o, call: c, parsed: true}
	for _, src := range c.commands {
		invs, ok := shell.Parse(src, c.cwd, o.Home)
		x.invs = append(x.invs, invs...)
		x.parsed = x.parsed && ok
	}
	return x
}

// git runs a read-only git command for g's repository, cached for the call.
// ok is false when it fails, times out, overflows, or lookups are off;
// callers then allow. No lookup adds, locks, or hashes the worktree in bulk,
// and fsmonitor stays off.
func (c *ctx) git(g *gitCmd, args ...string) (string, bool) {
	safe := []string{"-c", "core.fsmonitor=false", "-c", "core.untrackedCache=false"}
	return c.run(gitTimeout, g.dir, g.env, "git", append(safe, args...)...)
}

func (c *ctx) gh(dir string, env []string, args ...string) (string, bool) {
	return c.run(ghTimeout, dir, env, "gh", args...)
}

func (c *ctx) run(timeout time.Duration, dir string, env []string, name string, args ...string) (string, bool) {
	if c.lookupsOff || dir == "" {
		return "", false
	}
	key := dir + "\x00" + strings.Join(env, "\x00") + "\x00" + name + "\x00" + strings.Join(args, "\x00")
	if hit, ok := c.cache[key]; ok {
		return hit.out, hit.ok
	}
	cx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(cx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(append(cmd.Environ(), "GIT_OPTIONAL_LOCKS=0", "GH_PROMPT_DISABLED=1", "GIT_TERMINAL_PROMPT=0"), env...)
	cmd.WaitDelay = 100 * time.Millisecond // a child holding stdout open cannot outlive the timeout
	var out bytes.Buffer
	capped := &limited{w: &out, n: 64 << 20}
	cmd.Stdout = capped
	err := cmd.Run()
	res := lookup{out: out.String(), ok: err == nil && !capped.overflow}
	if c.cache == nil {
		c.cache = map[string]lookup{}
	}
	c.cache[key] = res
	return res.out, res.ok
}

// limited stops buffering after n bytes so a huge output cannot exhaust memory.
// An overflowing lookup counts as failed rather than as a complete answer.
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
