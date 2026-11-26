package shell

import (
	"strconv"
	"strings"
	"unicode"

	"mvdan.cc/sh/v3/syntax"
)

func (f frame) word(w *syntax.Word) Word {
	if w == nil {
		return Word{}
	}
	var b strings.Builder
	dynamic := false
	for _, part := range w.Parts {
		if !f.part(&b, part, false) {
			dynamic = true
		}
	}
	quoted := false
	if len(w.Parts) > 0 {
		lit, ok := w.Parts[0].(*syntax.Lit)
		quoted = !ok || strings.HasPrefix(lit.Value, `\`)
	}
	return Word{Val: b.String(), Dynamic: dynamic, Quoted: quoted}
}

func (f frame) part(b *strings.Builder, part syntax.WordPart, quoted bool) bool {
	switch p := part.(type) {
	case *syntax.Lit:
		b.WriteString(unescape(p.Value, quoted))
	case *syntax.SglQuoted:
		if p.Dollar {
			b.WriteString(ansiC(p.Value))
		} else {
			b.WriteString(p.Value)
		}
	case *syntax.DblQuoted:
		literal := true
		for _, inner := range p.Parts {
			if !f.part(b, inner, true) {
				literal = false
			}
		}
		return literal
	case *syntax.CmdSubst:
		if body, ok := catHeredoc(f, p); ok {
			b.WriteString(body)
			return true
		}
		b.WriteString(f.text(p))
		return false
	case *syntax.ExtGlob:
		b.WriteString(f.text(p))
	default:
		b.WriteString(f.text(part))
		return false
	}
	return true
}

func catHeredoc(f frame, c *syntax.CmdSubst) (string, bool) {
	if len(c.Stmts) != 1 {
		return "", false
	}
	s := c.Stmts[0]
	call, ok := s.Cmd.(*syntax.CallExpr)
	if !ok || len(call.Assigns) > 0 || len(call.Args) != 1 || call.Args[0].Lit() != "cat" || len(s.Redirs) != 1 {
		return "", false
	}
	r := s.Redirs[0]
	if r.Op != syntax.Hdoc && r.Op != syntax.DashHdoc {
		return "", false
	}
	body := f.heredoc(r)
	if body.Dynamic {
		return "", false
	}
	return strings.TrimRight(body.Val, "\n"), true
}

// A quoted here-document delimiter keeps the body literal; otherwise
// backslashes and expansions work as inside double quotes.
func (f frame) heredoc(r *syntax.Redirect) Word {
	if r.Hdoc == nil {
		return Word{}
	}
	quoted := false
	if r.Word != nil {
		for _, part := range r.Word.Parts {
			if lit, ok := part.(*syntax.Lit); !ok || strings.Contains(lit.Value, `\`) {
				quoted = true
			}
		}
	}
	var b strings.Builder
	dynamic := false
	for _, part := range r.Hdoc.Parts {
		if lit, ok := part.(*syntax.Lit); ok && quoted {
			b.WriteString(lit.Value)
		} else if !f.part(&b, part, true) {
			dynamic = true
		}
	}
	return Word{Val: b.String(), Dynamic: dynamic}
}

func unescape(s string, quoted bool) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '\\' || i+1 >= len(s) {
			b.WriteByte(c)
			continue
		}
		next := s[i+1]
		switch {
		case next == '\n':
			i++
		case !quoted || strings.IndexByte("$`\"\\", next) >= 0:
			b.WriteByte(next)
			i++
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

func ansiC(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '\\' || i+1 >= len(s) {
			b.WriteByte(c)
			continue
		}
		i++
		switch e := s[i]; e {
		case 'a':
			b.WriteByte('\a')
		case 'b':
			b.WriteByte('\b')
		case 'e', 'E':
			b.WriteByte(0x1b)
		case 'f':
			b.WriteByte('\f')
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 't':
			b.WriteByte('\t')
		case 'v':
			b.WriteByte('\v')
		case '\\', '\'', '"', '?':
			b.WriteByte(e)
		case 'c':
			if i+1 < len(s) {
				i++
				b.WriteByte(s[i] & 0x1f)
			}
		case 'x', 'u', 'U':
			max := map[byte]int{'x': 2, 'u': 4, 'U': 8}[e]
			j := i + 1
			for j < len(s) && j-i-1 < max && isHex(s[j]) {
				j++
			}
			if j == i+1 {
				b.WriteByte('\\')
				b.WriteByte(e)
				continue
			}
			n, _ := strconv.ParseUint(s[i+1:j], 16, 32)
			i = j - 1
			if n == 0 {
				return b.String()
			}
			if e == 'x' {
				b.WriteByte(byte(n))
			} else {
				b.WriteRune(rune(n))
			}
		case '0', '1', '2', '3', '4', '5', '6', '7':
			j := i
			for j < len(s) && j-i < 3 && s[j] >= '0' && s[j] <= '7' {
				j++
			}
			n, _ := strconv.ParseUint(s[i:j], 8, 16)
			i = j - 1
			if n == 0 {
				return b.String()
			}
			b.WriteByte(byte(n))
		default:
			b.WriteByte('\\')
			b.WriteByte(e)
		}
	}
	return b.String()
}

func isHex(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}

// Split divides s into words the way git splits an alias: on whitespace, with
// single quotes, double quotes, and backslash escapes, and nothing else
// special. It reports false for an unclosed quote or a trailing backslash.
func Split(s string) ([]Word, bool) { return split(s, false) }

// split also reads the string of env -S, where \_ separates words (inside
// double quotes it is a space) and \c ends the string.
func split(s string, env bool) ([]Word, bool) {
	var words []Word
	var word strings.Builder
	open, quote, escaped := false, rune(0), false
	end := func() {
		if open {
			words = append(words, Word{Val: word.String()})
		}
		word.Reset()
		open = false
	}
	for _, r := range s {
		switch {
		case escaped && env && quote == 0 && r == 'c':
			end()
			return words, true
		case escaped && env && quote == 0 && r == '_':
			end()
			escaped = false
		case escaped:
			if env && r == '_' {
				r = ' '
			}
			word.WriteRune(r)
			open, escaped = true, false
		case r == '\\' && quote != '\'':
			escaped = true
		case quote == 0 && unicode.IsSpace(r):
			end()
		case quote == 0 && (r == '\'' || r == '"'):
			open, quote = true, r
		case r == quote:
			quote = 0
		default:
			word.WriteRune(r)
			open = true
		}
	}
	if escaped || quote != 0 {
		return nil, false
	}
	end()
	return words, true
}
