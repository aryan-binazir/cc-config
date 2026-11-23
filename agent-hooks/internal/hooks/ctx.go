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

// git runs a read-only git command in dir, cached for the call. ok is false
// when it fails, times out, or lookups are off; callers then allow.
func (c *ctx) git(dir string, args ...string) (string, bool) {
	return c.run(gitTimeout, dir, "git", args...)
}

func (c *ctx) gh(dir string, args ...string) (string, bool) {
	return c.run(ghTimeout, dir, "gh", args...)
}

func (c *ctx) run(timeout time.Duration, dir, name string, args ...string) (string, bool) {
	if c.lookupsOff || dir == "" {
		return "", false
	}
	key := dir + "\x00" + name + "\x00" + strings.Join(args, "\x00")
	if hit, ok := c.cache[key]; ok {
		return hit.out, hit.ok
	}
	cx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(cx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(), "GIT_OPTIONAL_LOCKS=0", "GH_PROMPT_DISABLED=1", "GIT_TERMINAL_PROMPT=0")
	var out bytes.Buffer
	cmd.Stdout = &limited{w: &out, n: 4 << 20}
	err := cmd.Run()
	res := lookup{out: out.String(), ok: err == nil}
	if c.cache == nil {
		c.cache = map[string]lookup{}
	}
	c.cache[key] = res
	return res.out, res.ok
}

// limited stops buffering after n bytes so a huge output cannot exhaust memory.
type limited struct {
	w *bytes.Buffer
	n int
}

func (l *limited) Write(p []byte) (int, error) {
	if room := l.n - l.w.Len(); room > 0 {
		if len(p) > room {
			l.w.Write(p[:room])
		} else {
			l.w.Write(p)
		}
	}
	return len(p), nil
}
