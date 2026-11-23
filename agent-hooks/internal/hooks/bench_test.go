package hooks

import (
	"encoding/json"
	"testing"
)

var all = map[string]bool{
	"block-agy": true, "draft-prs": true, "pr-assignee": true, "no-admin-merge": true, "merge-head-match": true,
	"no-production": true, "commit-format": true, "no-secrets-commit": true, "destructive-git": true, "rg-replace-flag": true,
}

func benchPayload(b *testing.B, dir, command string) []byte {
	data, err := json.Marshal(map[string]any{"tool_name": "Bash", "tool_input": map[string]any{"command": command}, "cwd": dir})
	if err != nil {
		b.Fatal(err)
	}
	return data
}

func BenchmarkRunPayloadOnly(b *testing.B) {
	for name, command := range map[string]string{
		"echo":      "echo hello",
		"pipeline":  "cd src && rg -n TODO | sort | uniq -c | head -20",
		"gh-create": "gh pr create --draft --assignee @me --title 'feat(no-ticket): X' --body-file /tmp/body.md",
		"script":    "set -euo pipefail\nfor f in $(ls); do\n  if [ -f \"$f\" ]; then wc -l \"$f\"; fi\ndone\ncat <<'EOF' > out.txt\nline\nEOF\nbash -c 'echo nested'",
	} {
		b.Run(name, func(b *testing.B) {
			payload := benchPayload(b, "/nonexistent", command)
			for b.Loop() {
				if r := Run(Options{Phase: "pre", Stdin: payload, Enabled: all}); r.Exit != 0 {
					b.Fatalf("%+v", r)
				}
			}
		})
	}
}

func BenchmarkRunGitLookup(b *testing.B) {
	r := newRepo(b)
	r.git("switch", "-q", "-c", "amb/x")
	r.write("a.txt", "a")
	r.git("add", "a.txt")
	r.git("commit", "-q", "-m", "feat(no-ticket): First")
	r.write("notes.txt", "n")
	for name, command := range map[string]string{
		"commit-type": `git commit -m "feat(no-ticket): More"`,
		"add-all":     "git add -A",
		"reset-hard":  "git reset --hard",
	} {
		b.Run(name, func(b *testing.B) {
			payload := benchPayload(b, r.dir, command)
			for b.Loop() {
				Run(Options{Phase: "pre", Stdin: payload, Enabled: all})
			}
		})
	}
}
