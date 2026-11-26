package hooks

import (
	"strings"
	"testing"
)

func TestCommandsBeforeASyntaxErrorAreStillChecked(t *testing.T) {
	for _, command := range []string{
		"git push --force origin x\n((",
		"echo ok\ngit push --force origin x\nif then",
		"bash -c $'git push --force origin x\\n(('",
	} {
		if r := run(t, "destructive-git", bash(command)); !blocked(t, r, "destructive-git") {
			t.Errorf("%q: bash runs the first line before the syntax error, got %+v", command, r)
		}
	}
}

func TestUnreadableTextThatCouldHoldAGuardedCommandIsBlocked(t *testing.T) {
	for _, command := range []string{
		"echo hi\n(( git push --force origin x",
		"ls 'unterminated; rm -rf /",
		`bash -c "(( gh pr merge 1 --admin"`,
		"echo $((/usr/bin/git push --force origin x); (true))",
	} {
		if r := run(t, "destructive-git", bash(command)); !blocked(t, r, "unparsed") {
			t.Errorf("%q: want an unparsed block, got %+v", command, r)
		}
	}
	expectAllowed(t, "destructive-git",
		"echo $((printf ok); (true))", "echo hi\n((", "ls 'unterminated", "echo 'digit forgot rmdir",
	)
}

func TestTypedInputIsNeverAShellSyntaxError(t *testing.T) {
	for name, payload := range map[string]any{
		"codex write_stdin": map[string]any{"tool_name": "write_stdin", "tool_input": map[string]any{"session_id": 3, "chars": "print('git rm')) # python\n"}},
		"agy send_input":    agyPayload("manage_task", map[string]any{"Action": "send_input", "TaskId": "1", "Input": "SELECT 'rm' FROM git WHERE (\n"}),
	} {
		r := runOpts(t, Options{Phase: "pre"}, payload)
		if r.Exit != 0 || strings.Contains(r.Stdout, "deny") || r.Stderr != "" {
			t.Errorf("%s: keystrokes for a running program were blocked: %+v", name, r)
		}
	}
}

func TestUnparsedBlockNeedsAHookThatReadsCommands(t *testing.T) {
	if r := run(t, "block-agy", bash("ls 'unterminated; rm -rf /")); r.Exit != 0 {
		t.Errorf("block-agy alone keeps its own fallback: %+v", r)
	}
	if r := runOpts(t, Options{Phase: "post"}, bash("ls 'unterminated; rm -rf /")); r.Exit != 0 || r.Stdout != "" {
		t.Errorf("post cannot block: %+v", r)
	}
}

func TestHereDocumentsThatShareALineWithAnotherCommandAreRead(t *testing.T) {
	for _, command := range []string{
		"bash <<'EOF'; echo after\ngit push --force origin x\nEOF",
		"echo before; bash <<'EOF'; echo after\ngit push --force origin x\nEOF",
	} {
		if res := run(t, "destructive-git", bash(command)); !blocked(t, res, "destructive-git") {
			t.Errorf("%q: the here-document is a script bash runs, got %+v", command, res)
		}
	}
	for _, command := range []string{
		"cat <<'EOF'; echo after\ngit is only mentioned here\nEOF\n((",
		"cat <<'A'; cat <<'B'\nrm -rf / is text\nA\ngh is text\nB\necho done\n((",
		"bash <<'EOF'; ((\necho hi\nEOF",
		"{ cat <<'EOF'; }; echo done\nrm is just text\nEOF\n((",
		"if true; then cat <<'EOF'; fi; echo done\ngit is just text\nEOF\n((",
		"(cat <<'EOF'); echo \"$(cat <<'INNER'\ngh is just text\nINNER\n)\"\nrm is just text\nEOF\n((",
	} {
		if res := runOpts(t, Options{Phase: "pre"}, bash(command)); res.Exit != 0 {
			t.Errorf("%q: a here-document that was read is not unread text, got %+v", command, res)
		}
	}
}
