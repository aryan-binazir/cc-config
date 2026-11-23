// Package shell turns a shell command string into the simple commands it would run.
//
// It walks every statement, including pipelines, subshells, command and
// process substitutions, and the bodies of `bash -c`, `eval`, and shells fed a
// here-document. Common wrappers (sudo, env, timeout, nohup, ...) are peeled so
// the wrapped command is reported as its own invocation. Expansions are not
// performed: a word that depends on runtime state keeps its source text and is
// marked Dynamic.
package shell

import (
	"path"
	"reflect"
	"slices"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// Word is one argument as the shell would pass it, when that is knowable.
type Word struct {
	Val     string // the decoded value, or the source text when Dynamic
	Dynamic bool   // depends on parameter, arithmetic or command expansion
	Quoted  bool   // starts with a quote, so a leading ~ is literal
}

// Invocation is one simple command.
type Invocation struct {
	Assigns map[string]string // leading NAME=value pairs, inherited by wrapped and nested commands
	Args    []Word            // Args[0] is the command name
	Stdin   *string           // here-document or here-string body on fd 0
	// StdinDynamic marks a Stdin that contains expansions, kept as source text.
	StdinDynamic bool
	Cwd          string // directory it runs in; "" when unknown
	Source       string // the source text of the whole command
}

// Name is the basename of the command word.
func (inv Invocation) Name() string {
	if len(inv.Args) == 0 {
		return ""
	}
	return path.Base(inv.Args[0].Val)
}

// Values returns every word's value, the command name first.
func (inv Invocation) Values() []string {
	out := make([]string, len(inv.Args))
	for i, w := range inv.Args {
		out[i] = w.Val
	}
	return out
}

// Strings returns the argument values after the command name.
func (inv Invocation) Strings() []string {
	out := make([]string, 0, len(inv.Args))
	for _, w := range inv.Args[1:] {
		out = append(out, w.Val)
	}
	return out
}

const maxDepth = 8

// Parse returns every simple command in src, run from cwd ("" when unknown).
// ok is false when src is not valid shell.
func Parse(src, cwd, home string) (invs []Invocation, ok bool) {
	w := &walker{home: home}
	ok = w.script(src, cwd, nil, 0)
	return w.out, ok
}

type walker struct {
	home    string
	out     []Invocation
	reached dirs
}

func (w *walker) script(src, cwd string, env map[string]string, depth int) (parsed bool) {
	file, err := syntax.NewParser(syntax.Variant(syntax.LangBash)).Parse(strings.NewReader(src), "")
	if err != nil {
		return false
	}
	f := frame{src: src, env: env, depth: depth, home: w.home}
	f.stmts(w, file.Stmts, dirs{cwd})
	return true
}

type frame struct {
	src   string
	env   map[string]string
	depth int
	home  string
}

func (f frame) text(n syntax.Node) string {
	start, end := int(n.Pos().Offset()), int(n.End().Offset())
	if start < 0 || end > len(f.src) || start > end {
		return ""
	}
	return f.src[start:end]
}

type dirs []string

const maxDirs = 4

func (d dirs) union(more ...dirs) dirs {
	out := append(dirs{}, d...)
	for _, m := range more {
		for _, dir := range m {
			if len(out) < maxDirs && !slices.Contains(out, dir) {
				out = append(out, dir)
			}
		}
	}
	return out
}

func (f frame) stmts(w *walker, list []*syntax.Stmt, cwd dirs) dirs {
	for _, s := range list {
		cwd = f.stmt(w, s, cwd)
	}
	return cwd
}

func (f frame) stmt(w *walker, s *syntax.Stmt, cwd dirs) dirs {
	if s == nil {
		return cwd
	}
	for _, r := range s.Redirs {
		f.substs(w, r.Word, cwd)
		f.substs(w, r.Hdoc, cwd)
	}
	switch c := s.Cmd.(type) {
	case *syntax.CallExpr:
		return f.call(w, c, s, cwd)
	case *syntax.BinaryCmd:
		switch c.Op {
		case syntax.AndStmt:
			return f.stmt(w, c.Y, f.stmt(w, c.X, cwd))
		case syntax.OrStmt:
			mark := len(w.reached)
			afterX := f.stmt(w, c.X, cwd)
			whereXFailed := cwd
			if !isCd(c.X) {
				whereXFailed = cwd.union(w.reached[mark:], afterX)
			}
			afterY := f.stmt(w, c.Y, whereXFailed)
			if exits(c.Y) {
				return afterX
			}
			return afterX.union(afterY)
		}
		f.stmt(w, c.X, cwd)
		f.stmt(w, c.Y, cwd)
	case *syntax.Block:
		return f.stmts(w, c.Stmts, cwd)
	case *syntax.Subshell:
		f.stmts(w, c.Stmts, cwd)
	case *syntax.IfClause:
		for clause := c; clause != nil; clause = clause.Else {
			f.stmts(w, clause.Then, f.stmts(w, clause.Cond, cwd))
		}
	case *syntax.WhileClause:
		f.stmts(w, c.Do, f.stmts(w, c.Cond, cwd))
	case *syntax.ForClause:
		f.substs(w, c.Loop, cwd)
		f.stmts(w, c.Do, cwd)
	case *syntax.CaseClause:
		f.substs(w, c.Word, cwd)
		for _, item := range c.Items {
			for _, pattern := range item.Patterns {
				f.substs(w, pattern, cwd)
			}
			f.stmts(w, item.Stmts, cwd)
		}
	case *syntax.FuncDecl:
		f.stmt(w, c.Body, cwd)
	case *syntax.TimeClause:
		w.out = append(w.out, Invocation{Args: []Word{{Val: "time"}}, Cwd: cwd[0], Source: f.text(s)})
		return f.stmt(w, c.Stmt, cwd)
	case *syntax.CoprocClause:
		f.stmt(w, c.Stmt, cwd)
	case nil:
	default:
		f.substs(w, c, cwd)
	}
	return cwd
}

func (f frame) substs(w *walker, n syntax.Node, cwd dirs) {
	if n == nil || reflect.ValueOf(n).IsNil() {
		return
	}
	syntax.Walk(n, func(node syntax.Node) bool {
		switch sub := node.(type) {
		case *syntax.CmdSubst:
			f.stmts(w, sub.Stmts, cwd)
			return false
		case *syntax.ProcSubst:
			f.stmts(w, sub.Stmts, cwd)
			return false
		}
		return true
	})
}

func (f frame) call(w *walker, c *syntax.CallExpr, s *syntax.Stmt, cwd dirs) dirs {
	for _, a := range c.Assigns {
		f.substs(w, a, cwd)
	}
	for _, arg := range c.Args {
		f.substs(w, arg, cwd)
	}
	env := f.env
	if len(c.Assigns) > 0 {
		env = copyEnv(env)
		for _, a := range c.Assigns {
			if a.Name != nil && a.Value != nil {
				env[a.Name.Value] = f.word(a.Value).Val
			} else if a.Name != nil {
				env[a.Name.Value] = ""
			}
		}
	}
	if len(c.Args) == 0 {
		return cwd
	}
	inv := Invocation{Assigns: env, Source: f.text(s)}
	for _, arg := range c.Args {
		inv.Args = append(inv.Args, f.word(arg))
	}
	for _, r := range s.Redirs {
		if r.N != nil && r.N.Value != "0" {
			continue
		}
		var body Word
		switch r.Op {
		case syntax.Hdoc, syntax.DashHdoc:
			body = f.heredoc(r)
		case syntax.WordHdoc:
			body = f.word(r.Word)
			body.Val += "\n"
		case syntax.RdrIn, syntax.RdrInOut, syntax.DplIn:
			inv.Stdin, inv.StdinDynamic = nil, false
			continue
		default:
			continue
		}
		inv.Stdin, inv.StdinDynamic = &body.Val, body.Dynamic
	}
	var after dirs
	for _, dir := range cwd {
		inv.Cwd = dir
		f.emit(w, inv)
		if inv.Args[0].Val == "cd" {
			after = after.union(dirs{f.cd(inv, dir)})
		}
	}
	if after == nil {
		return cwd
	}
	w.reached = append(w.reached, after...)
	return after
}

func (f frame) emit(w *walker, inv Invocation) {
	w.out = append(w.out, inv)
	if f.depth >= maxDepth {
		return
	}
	name := inv.Name()
	args := inv.Args[1:]
	switch name {
	case "bash", "sh", "zsh", "dash", "ksh", "fish":
		command, fromStdin := false, false
		i := 0
		for ; i < len(args); i++ {
			a := args[i].Val
			if a == "--" || a == "-" || !(strings.HasPrefix(a, "-") || strings.HasPrefix(a, "+")) {
				break
			}
			switch {
			case a == "-o" || a == "+o" || a == "-O" || a == "+O":
				i++ // the option name
			case !strings.HasPrefix(a, "--") && strings.Contains(a, "c"):
				command = true
			case !strings.HasPrefix(a, "--") && strings.Contains(a, "s"):
				fromStdin = true
			}
		}
		if command {
			if i < len(args) && args[i].Val == "--" {
				i++
			}
			if i < len(args) && !args[i].Dynamic {
				f.nested(w, args[i].Val, inv)
			}
			return
		}
		if i < len(args) && args[i].Val == "--" {
			i++
		}
		if inv.Stdin != nil && (i == len(args) || fromStdin) {
			f.nested(w, *inv.Stdin, inv)
		}
	case "eval":
		if len(args) > 0 && args[0].Val == "--" {
			args = args[1:]
		}
		parts := make([]string, 0, len(args))
		for _, a := range args {
			parts = append(parts, a.Val)
		}
		f.nested(w, strings.Join(parts, " "), inv)
	default:
		if rest, env, chdir, ok := unwrap(name, args); ok && len(rest) > 0 {
			inner := inv
			inner.Args = rest
			if chdir != nil {
				inner.Cwd = ""
				if !chdir.Dynamic {
					inner.Cwd = Resolve(inv.Cwd, chdir.Val, f.home)
				}
			}
			if len(env) > 0 {
				inner.Assigns = copyEnv(inv.Assigns)
				for k, v := range env {
					inner.Assigns[k] = v
				}
			}
			sub := f
			sub.depth++
			sub.emit(w, inner)
		}
	}
}

func (f frame) nested(w *walker, src string, parent Invocation) {
	inner := &walker{home: w.home, out: w.out}
	inner.script(src, parent.Cwd, parent.Assigns, f.depth+1)
	w.out = inner.out
}

func (f frame) cd(inv Invocation, cwd string) string {
	args := inv.Strings()
	for len(args) > 0 && (args[0] == "-L" || args[0] == "-P" || args[0] == "--") {
		args = args[1:]
	}
	if len(args) == 0 {
		return f.home
	}
	if len(args) != 1 || inv.Args[len(inv.Args)-1].Dynamic {
		return ""
	}
	return Resolve(cwd, args[0], f.home)
}

func isCd(s *syntax.Stmt) bool {
	call, ok := s.Cmd.(*syntax.CallExpr)
	return ok && len(call.Args) > 0 && call.Args[0].Lit() == "cd"
}

func exits(s *syntax.Stmt) bool {
	switch c := s.Cmd.(type) {
	case *syntax.CallExpr:
		return len(c.Args) > 0 && (c.Args[0].Lit() == "exit" || c.Args[0].Lit() == "return")
	case *syntax.Block:
		return len(c.Stmts) > 0 && exits(c.Stmts[len(c.Stmts)-1])
	}
	return false
}

// Resolve joins p onto cwd, expanding a leading ~. It returns "" when the
// result cannot be known.
func Resolve(cwd, p, home string) string {
	switch {
	case p == "~":
		return home
	case strings.HasPrefix(p, "~/"):
		if home == "" {
			return ""
		}
		return path.Join(home, p[2:])
	case strings.HasPrefix(p, "/"):
		return path.Clean(p)
	case cwd == "" || p == "-" || strings.HasPrefix(p, "~"):
		return ""
	}
	return path.Join(cwd, p)
}

func copyEnv(env map[string]string) map[string]string {
	out := make(map[string]string, len(env)+1)
	for k, v := range env {
		out[k] = v
	}
	return out
}
