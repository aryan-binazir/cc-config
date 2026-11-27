package shell_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/aryan-binazir/cc-config/agent-hooks/internal/shell"
)

const (
	cwd  = "/w"
	home = "/home/u"
)

func parse(t *testing.T, src string) []shell.Invocation {
	t.Helper()
	invs, unread := shell.Parse(src, cwd, home)
	if len(unread) > 0 {
		t.Fatalf("%q: unexpected unread text %q", src, unread)
	}
	return invs
}

func argv(invs []shell.Invocation) [][]string {
	out := [][]string{}
	for _, inv := range invs {
		words := []string{}
		for _, w := range inv.Args {
			words = append(words, w.Val)
		}
		out = append(out, words)
	}
	return out
}

func named(invs []shell.Invocation, name string) []shell.Invocation {
	var out []shell.Invocation
	for _, inv := range invs {
		if inv.Name() == name {
			out = append(out, inv)
		}
	}
	return out
}

func TestParseReportsEveryCommandAWrapperOrNestedShellRuns(t *testing.T) {
	for _, tc := range []struct {
		src  string
		want [][]string
	}{
		{"sudo -u ar timeout --sig KILL 5 nohup target x", [][]string{
			{"sudo", "-u", "ar", "timeout", "--sig", "KILL", "5", "nohup", "target", "x"},
			{"timeout", "--sig", "KILL", "5", "nohup", "target", "x"},
			{"nohup", "target", "x"},
			{"target", "x"},
		}},
		{"command target x", [][]string{{"command", "target", "x"}, {"target", "x"}}},
		{"command -v target", [][]string{{"command", "-v", "target"}}},
		{"mise exec tool -- target x", [][]string{{"mise", "exec", "tool", "--", "target", "x"}, {"target", "x"}}},
		{"time target x", [][]string{{"time"}, {"target", "x"}}},
		{"bash -o pipefail -c 'target x'", [][]string{{"bash", "-o", "pipefail", "-c", "target x"}, {"target", "x"}}},
		{"bash -c -- 'target x'", [][]string{{"bash", "-c", "--", "target x"}, {"target", "x"}}},
		{"bash +O extglob -c 'target x'", [][]string{{"bash", "+O", "extglob", "-c", "target x"}, {"target", "x"}}},
		{`bash -c "bash -c 'target x'"`, [][]string{{"bash", "-c", "bash -c 'target x'"}, {"bash", "-c", "target x"}, {"target", "x"}}},
		{"bash --version -c 'target x'", [][]string{{"bash", "--version", "-c", "target x"}}},
		{"eval -- target x", [][]string{{"eval", "--", "target", "x"}, {"target", "x"}}},
		{"eval 'target x; other'", [][]string{{"eval", "target x; other"}, {"target", "x"}, {"other"}}},
		{"bash -s arg <<'EOF'\ntarget x\nEOF", [][]string{{"bash", "-s", "arg"}, {"target", "x"}}},
		{"bash -o pipefail <<'EOF'\ntarget x\nEOF", [][]string{{"bash", "-o", "pipefail"}, {"target", "x"}}},
		{"bash <<'EOF'; other\ntarget x\nEOF", [][]string{{"bash"}, {"target", "x"}, {"other"}}},
		{"case x in $(target x)) ;; esac", [][]string{{"target", "x"}}},
		{"out=$(echo `target x`)", [][]string{{"target", "x"}, {"echo", "`target x`"}}},
		{"diff <(target x) /dev/null", [][]string{{"target", "x"}, {"diff", "<(target x)", "/dev/null"}}},
	} {
		if got := argv(parse(t, tc.src)); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%q:\n got %q\nwant %q", tc.src, got, tc.want)
		}
	}
}

func TestParseSplitsEnvSplitStringTheWayEnvDoes(t *testing.T) {
	for _, tc := range []struct {
		src     string
		envRuns [][]string
	}{
		{`env -S 'target -m "a b"'`, [][]string{{"target", "-m", "a b"}}},
		{`env --split-string='V=1 target x'`, [][]string{{"target", "x"}}},
		{`env -S 'target\_x' y`, [][]string{{"target", "x", "y"}}},
		{`env -S '# a comment ends the string' target x`, [][]string{{"target", "x"}}},
		{`env -S '\c ignored' target x`, [][]string{{"target", "x"}}},
		{`env -S 'bash -c "target x"'`, [][]string{{"bash", "-c", "target x"}, {"target", "x"}}},
		{`env -S 'echo \q' target x`, [][]string{}}, // env rejects the string and runs nothing
	} {
		got := argv(parse(t, tc.src))
		if len(got) == 0 || got[0][0] != "env" || !reflect.DeepEqual(got[1:], tc.envRuns) {
			t.Errorf("%q:\n got %q\nwant env then %q", tc.src, got, tc.envRuns)
		}
	}
	if env := parse(t, `env -S 'V=1 target'`); len(env) != 2 || env[1].Assigns["V"] != "1" {
		t.Errorf("an assignment in the split string reaches the command: %+v", env)
	}
}

func TestParseMarksCommandsThatRunTextItWasNotShown(t *testing.T) {
	for _, tc := range []struct {
		src    string
		opaque []bool
	}{
		{"bash -c 'target x'", []bool{false, false}},
		{`bash -c "$script"`, []bool{true}},
		{"bash build.sh", []bool{true}},
		{"echo x | bash", []bool{false, true}},
		{"source env.sh", []bool{true}},
		{". ./env.sh", []bool{true}},
		{`env -S "$launcher"`, []bool{true}},
		{"env -S 'target x'", []bool{false, false}},
	} {
		var got []bool
		for _, inv := range parse(t, tc.src) {
			got = append(got, inv.Opaque)
		}
		if !slices.Equal(got, tc.opaque) {
			t.Errorf("%q: opaque %v, want %v", tc.src, got, tc.opaque)
		}
	}
}

func TestParseRunsACommandInEveryDirectoryACdCanLeave(t *testing.T) {
	for _, tc := range []struct {
		src  string
		dirs []string
	}{
		{"target", []string{"/w"}},
		{"cd a && target", []string{"/w/a"}},
		{"cd a || cd b || exit 1; target", []string{"/w/a", "/w/b"}},
		{"cd a || exit 1; target", []string{"/w/a"}},
		{"{ cd a; false; } || target", []string{"/w", "/w/a"}},
		{"cd a && cd b && false || target", []string{"/w", "/w/a", "/w/a/b"}},
		{"(cd a); target", []string{"/w"}},
		{"cd /abs/../x; target", []string{"/x"}},
		{"cd; target", []string{"/home/u"}},
		{"cd ~/p; target", []string{"/home/u/p"}},
		{`cd "$D"; target`, []string{""}},
		{"cd -; target", []string{""}},
		{"env -C sub target", []string{"/w/sub"}},
		{"env --chdir=/x target", []string{"/x"}},
		{`env -C "$D" target`, []string{""}},
		{"bash -c 'cd a; target'", []string{"/w/a"}},
	} {
		var got []string
		for _, inv := range named(parse(t, tc.src), "target") {
			got = append(got, inv.Cwd)
		}
		slices.Sort(got)
		if !slices.Equal(got, tc.dirs) {
			t.Errorf("%q: target runs in %q, want %q", tc.src, got, tc.dirs)
		}
	}
	invs, unread := shell.Parse("cd a; target", "", home)
	if unread != nil || len(invs) != 2 || invs[1].Cwd != "" {
		t.Errorf("a relative cd from an unknown directory stays unknown: %+v %q", invs, unread)
	}
}

func TestParseCarriesExportsTheWayTheShellDoes(t *testing.T) {
	type value struct {
		val     string
		present bool
	}
	for _, tc := range []struct {
		src  string
		want value
	}{
		{"V=1 target", value{"1", true}},
		{"env V=1 target", value{"1", true}},
		{"export V=1; target", value{"1", true}},
		{"export V=; target", value{"", true}},
		{"export V; V=2; target", value{"2", true}},
		{"export V=a; V+=b; target", value{"ab", true}},
		{"declare -x V=1; target", value{"1", true}},
		{"typeset -gx V=1; target", value{"1", true}},
		{"{ export V=1; }; target", value{"1", true}},
		{"if true; then export V=1; fi; target", value{"1", true}},
		{"export V=1; bash -c 'target'", value{"1", true}},
		{"bash -c 'export V=1; target'", value{"1", true}},
		{"V=1; target", value{}},
		{"declare V=1; target", value{}},
		{"target; export V=1", value{}},
		{"(export V=1); target", value{}},
		{"echo $(export V=1); target", value{}},
		{"export V=1 | true; target", value{}},
		{"export V=1 & target", value{}},
		{"f() { export V=1; }; target", value{}},
		{"bash -c 'export V=1'; target", value{}},
		{"export V=1; unset V; target", value{}},
		{"export V=1; export -n V; target", value{}},
	} {
		targets := named(parse(t, tc.src), "target")
		if len(targets) != 1 {
			t.Fatalf("%q: %d targets", tc.src, len(targets))
		}
		val, present := targets[0].Assigns["V"]
		if got := (value{val, present}); got != tc.want {
			t.Errorf("%q: V=%+v, want %+v", tc.src, got, tc.want)
		}
	}
	invs := named(parse(t, "target; export V=1; target"), "target")
	if len(invs) != 2 {
		t.Fatalf("%d targets", len(invs))
	}
	if _, before := invs[0].Assigns["V"]; before || invs[1].Assigns["V"] != "1" {
		t.Errorf("an export reaches only the commands after it: %+v", invs)
	}
}

func TestParseReadsHereDocumentsAndHereStringsAsStdin(t *testing.T) {
	type stdin struct {
		body    string
		present bool
		dynamic bool
	}
	read := func(inv shell.Invocation) stdin {
		if inv.Stdin == nil {
			return stdin{dynamic: inv.StdinDynamic}
		}
		return stdin{*inv.Stdin, true, inv.StdinDynamic}
	}
	for _, tc := range []struct {
		src  string
		want []stdin
	}{
		{"cat <<A; cat <<B\none\nA\ntwo\nB", []stdin{{"one\n", true, false}, {"two\n", true, false}}},
		{"cat <<'EOF'\n$X\nEOF", []stdin{{"$X\n", true, false}}},
		{"cat <<EOF\n$X\nEOF", []stdin{{"$X\n", true, true}}},
		{"cat <<<word", []stdin{{"word\n", true, false}}},
		{"cat <<'EOF' < file\nx\nEOF", []stdin{{}}},
		{"cat <<EOF < file\n$X\nEOF", []stdin{{}}},
		{"cat <<'EOF' 0<&3\nx\nEOF", []stdin{{}}},
		{"cat 3<<'EOF'\nx\nEOF", []stdin{{}}},
		{"{ cat <<'EOF'; }; echo done\nbody\nEOF", []stdin{{"body\n", true, false}, {}}},
	} {
		var got []stdin
		for _, inv := range parse(t, tc.src) {
			got = append(got, read(inv))
		}
		if !slices.Equal(got, tc.want) {
			t.Errorf("%q: stdin %+v, want %+v", tc.src, got, tc.want)
		}
	}
}

func TestParseReportsTextAfterTheLastCompleteStatementAsUnread(t *testing.T) {
	for _, tc := range []struct {
		src    string
		argv   [][]string
		unread []string
	}{
		{"target x", [][]string{{"target", "x"}}, nil},
		{"target x\n((", [][]string{{"target", "x"}}, []string{"\n(("}},
		{"echo ok\ntarget x\nif then", [][]string{{"echo", "ok"}, {"target", "x"}}, []string{"\nif then"}},
		{"ls 'unterminated; target", [][]string{}, []string{"ls 'unterminated; target"}},
		{"cat <<'EOF'; echo after\ntext\nEOF\n((", [][]string{{"cat"}, {"echo", "after"}}, []string{"\n(("}},
		{"bash -c $'target x\\n(('", [][]string{{"bash", "-c", "target x\n(("}, {"target", "x"}}, []string{"\n(("}},
	} {
		invs, unread := shell.Parse(tc.src, cwd, home)
		if got := argv(invs); !reflect.DeepEqual(got, tc.argv) || !slices.Equal(unread, tc.unread) {
			t.Errorf("%q:\n got %q unread %q\nwant %q unread %q", tc.src, got, unread, tc.argv, tc.unread)
		}
	}
}

func TestParseDecodesWordsAndMarksTheOnesItCannotKnow(t *testing.T) {
	invs := parse(t, `echo lit '~' "~" ~ \~ $HOME "$HOME" "a b" $'tab\t' "$(cat <<'EOF'
heredoc
EOF
)" $(date)`)
	want := []shell.Word{
		{Val: "echo"},
		{Val: "lit"},
		{Val: "~", Quoted: true},
		{Val: "~", Quoted: true},
		{Val: "~"},
		{Val: "~", Quoted: true},
		{Val: "$HOME", Dynamic: true, Quoted: true},
		{Val: "$HOME", Dynamic: true, Quoted: true},
		{Val: "a b", Quoted: true},
		{Val: "tab\t", Quoted: true},
		{Val: "heredoc", Quoted: true},
		{Val: "$(date)", Dynamic: true, Quoted: true},
	}
	echo := named(invs, "echo")
	if len(echo) != 1 {
		t.Fatalf("%d echo invocations", len(echo))
	}
	if got := echo[0].Args; !slices.Equal(got, want) {
		t.Errorf("words:\n got %+v\nwant %+v", got, want)
	}
}

func TestParseFeedsACommandTheTextUpToTheEndOfItsPipeline(t *testing.T) {
	fed := func(src string) string {
		targets := named(parse(t, src), "target")
		if len(targets) != 1 {
			t.Fatalf("%q: %d targets", src, len(targets))
		}
		r := targets[0].Fed
		if r.Source == nil || r.Start < 0 || r.Start > r.End || r.End > len(r.Text) {
			t.Fatalf("%q: bad reach %+v", src, r)
		}
		return r.Text[r.Start:r.End]
	}
	if got := fed("echo before | target; echo after"); got != "echo before | target;" {
		t.Errorf("target is fed %q", got)
	}
	if got := fed("f() { target; }; echo after"); got != "f() { target; }; echo after" {
		t.Errorf("a function body can be called with any later text, got %q", got)
	}
}
