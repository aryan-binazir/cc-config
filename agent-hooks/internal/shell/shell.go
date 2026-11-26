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
	// Assigns holds leading NAME=value pairs and variables exported earlier in
	// the same shell; wrapped and nested commands inherit them.
	Assigns map[string]string
	Args    []Word  // Args[0] is the command name
	Stdin   *string // here-document or here-string body on fd 0
	// StdinDynamic marks a Stdin that contains expansions, kept as source text.
	StdinDynamic bool
	Cwd          string // directory it runs in; "" when unknown
	Source       string // the source text of the whole command
	// Fed is the text that can feed this command: its source up to the end of
	// the pipeline it is part of. Text after that cannot reach it.
	Fed Reach
	// Opaque marks a shell, or source, that runs text Parse was not shown: a
	// script file, a pipe or terminal, or a string built by an expansion.
	Opaque bool

	start, stop int // the command's own text within its source
}

// Source is one piece of shell text Parse walked: the command it was given,
// or the string a nested shell runs.
type Source struct {
	Text string
	// Outer is what else can feed the commands of a nested shell's string:
	// the text around that shell, then its other arguments, environment,
	// and input.
	Outer []Reach
}

// Reach is the part of a source from Start up to End.
type Reach struct {
	*Source
	Start, End int
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
// unread is the text it could not parse: what follows the last complete
// statement of src, or of a shell string nested in it. Bash runs the complete
// statements before it reaches such an error, so they are still reported.
func Parse(src, cwd, home string) (invs []Invocation, unread []string) {
	w := &walker{home: home}
	w.script(frame{src: &Source{Text: src}, env: &exported{}, home: home}, cwd)
	return w.out, w.unread
}

type walker struct {
	home    string
	out     []Invocation
	unread  []string
	reached dirs
}

func (w *walker) script(f frame, cwd string) {
	// The parser hands over a statement before it has read the here-documents
	// named on that line, so the walk waits until parsing is done.
	var stmts []*syntax.Stmt
	err := syntax.NewParser(syntax.Variant(syntax.LangBash)).Stmts(strings.NewReader(f.src.Text), func(s *syntax.Stmt) bool {
		stmts = append(stmts, s)
		return true
	})
	f.stmts(w, stmts, dirs{cwd})
	if err != nil {
		read := 0
		for _, s := range stmts {
			read = max(read, stmtEnd(s))
		}
		w.unread = append(w.unread, f.src.Text[min(read, len(f.src.Text)):])
	}
}

// stmtEnd is the offset just past s, including the here-documents named
// anywhere inside it, whose bodies sit after the line that names them.
func stmtEnd(s *syntax.Stmt) int {
	end := int(s.End().Offset())
	syntax.Walk(s, func(n syntax.Node) bool {
		if r, ok := n.(*syntax.Redirect); ok && r.Hdoc != nil {
			end = max(end, int(r.Hdoc.End().Offset()))
		}
		return true
	})
	return end
}

type frame struct {
	src   *Source
	env   *exported
	depth int
	home  string
	end   int // Fed.End for the commands being walked
}

// reach is how far into the source can feed s: to the end of its pipeline or
// loop, and for a function body, which runs wherever it is called, to the end.
func (f frame) reach(s *syntax.Stmt) int {
	switch c := s.Cmd.(type) {
	case *syntax.FuncDecl:
		return len(f.src.Text)
	case *syntax.CallExpr, *syntax.WhileClause, *syntax.ForClause:
		return max(f.end, stmtEnd(s))
	case *syntax.BinaryCmd:
		piped := c.Op == syntax.Pipe || c.Op == syntax.PipeAll
		if !piped || f.end >= int(s.End().Offset()) { // a list, or a stage of a pipeline already measured
			return f.end
		}
		return max(f.end, stmtEnd(s))
	}
	if len(s.Redirs) > 0 {
		return max(f.end, stmtEnd(s))
	}
	return f.end
}

// exported is the environment a shell passes to its commands. Updates replace
// the map, so invocations already emitted keep the environment they ran with.
type exported struct{ vars map[string]string }

func (e *exported) set(name, value string) {
	e.vars = copyEnv(e.vars)
	e.vars[name] = value
}

func (e *exported) unset(name string) {
	if _, ok := e.vars[name]; ok {
		e.vars = copyEnv(e.vars)
		delete(e.vars, name)
	}
}

// subshell is a copy of f whose exports do not reach f.
func (f frame) subshell() frame {
	f.env = &exported{f.env.vars}
	return f
}

func (f frame) text(n syntax.Node) string {
	start, end := int(n.Pos().Offset()), int(n.End().Offset())
	if start < 0 || end > len(f.src.Text) || start > end {
		return ""
	}
	return f.src.Text[start:end]
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
	if s.Background || s.Coprocess {
		f = f.subshell()
	}
	f.end = f.reach(s)
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
		f.subshell().stmt(w, c.X, cwd)
		f.subshell().stmt(w, c.Y, cwd)
	case *syntax.Block:
		return f.stmts(w, c.Stmts, cwd)
	case *syntax.Subshell:
		f.subshell().stmts(w, c.Stmts, cwd)
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
		f.subshell().stmt(w, c.Body, cwd)
	case *syntax.TimeClause:
		w.out = append(w.out, Invocation{Args: []Word{{Val: "time"}}, Cwd: cwd[0], Source: f.text(s), Fed: Reach{f.src, 0, f.end}})
		return f.stmt(w, c.Stmt, cwd)
	case *syntax.CoprocClause:
		f.subshell().stmt(w, c.Stmt, cwd)
	case *syntax.DeclClause:
		f.substs(w, c, cwd)
		f.declare(c)
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
			f.subshell().stmts(w, sub.Stmts, cwd)
			return false
		case *syntax.ProcSubst:
			f.subshell().stmts(w, sub.Stmts, cwd)
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
	if len(c.Args) == 0 {
		for _, a := range c.Assigns {
			if _, ok := f.env.vars[a.Name.Value]; ok {
				f.env.set(a.Name.Value, f.assigned(a))
			}
		}
		return cwd
	}
	env := f.env.vars
	if len(c.Assigns) > 0 {
		env = copyEnv(env)
		for _, a := range c.Assigns {
			env[a.Name.Value] = f.assigned(a)
		}
	}
	if c.Args[0].Lit() == "unset" {
		for _, arg := range c.Args[1:] {
			if name := arg.Lit(); name == "-f" {
				break
			} else if name != "-v" {
				f.env.unset(name)
			}
		}
	}
	inv := Invocation{Assigns: env, Source: f.text(s), Fed: Reach{f.src, 0, f.end}, start: int(s.Pos().Offset()), stop: stmtEnd(s)}
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

// assigned is the value a NAME=value or NAME+=value assignment leaves.
func (f frame) assigned(a *syntax.Assign) string {
	value := ""
	if a.Value != nil {
		value = f.word(a.Value).Val
	}
	if a.Append {
		value = f.env.vars[a.Name.Value] + value
	}
	return value
}

// declare applies export, export -n, and declare/typeset -x.
func (f frame) declare(c *syntax.DeclClause) {
	exporting := c.Variant.Value == "export"
	unexporting := false
	for _, a := range c.Args {
		if a.Name != nil {
			continue
		}
		if flag := f.word(a.Value).Val; strings.HasPrefix(flag, "-") {
			exporting = exporting || strings.Contains(flag, "x")
			unexporting = unexporting || (c.Variant.Value == "export" && strings.Contains(flag, "n"))
		}
	}
	for _, a := range c.Args {
		switch {
		case a.Name == nil:
		case unexporting:
			f.env.unset(a.Name.Value)
		case !exporting:
		case a.Naked:
			if _, ok := f.env.vars[a.Name.Value]; !ok {
				f.env.set(a.Name.Value, "")
			}
		default:
			f.env.set(a.Name.Value, f.assigned(a))
		}
	}
}

func (f frame) emit(w *walker, inv Invocation) {
	at := len(w.out)
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
			case a == "--version" || a == "--help":
				return
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
				f.nested(w, args[i].Val, inv, args[i+1:], inv.Stdin)
			} else {
				w.out[at].Opaque = true
			}
			return
		}
		if i < len(args) && args[i].Val == "--" {
			i++
		}
		if inv.Stdin != nil && (i == len(args) || fromStdin) {
			f.nested(w, *inv.Stdin, inv, args[i:], nil)
		} else {
			w.out[at].Opaque = true
		}
	case "source", ".":
		w.out[at].Opaque = true
	case "eval":
		if len(args) > 0 && args[0].Val == "--" {
			args = args[1:]
		}
		parts := make([]string, 0, len(args))
		for _, a := range args {
			parts = append(parts, a.Val)
		}
		f.nested(w, strings.Join(parts, " "), inv, nil, inv.Stdin)
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

// nested walks src, a string parent runs as shell. Besides src itself, its
// commands can be fed by what feeds parent (the text around it, as far as
// parent's own reach goes), by params (the positional parameters), by
// parent's environment, and by input (parent's standard input when that is
// not src).
func (f frame) nested(w *walker, src string, parent Invocation, params []Word, input *string) {
	var around []string
	for _, p := range params {
		around = append(around, p.Val)
	}
	for _, value := range parent.Assigns {
		around = append(around, value)
	}
	if input != nil {
		around = append(around, *input)
	}
	passed := &Source{Text: strings.Join(around, "\n")}
	outer := append(slices.Clone(f.src.Outer),
		Reach{f.src, 0, parent.start}, Reach{f.src, parent.stop, parent.Fed.End}, Reach{passed, 0, len(passed.Text)})
	inner := &walker{home: w.home, out: w.out}
	inner.script(frame{src: &Source{Text: src, Outer: outer}, env: &exported{parent.Assigns}, depth: f.depth + 1, home: w.home}, parent.Cwd)
	w.out = inner.out
	w.unread = append(w.unread, inner.unread...)
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
