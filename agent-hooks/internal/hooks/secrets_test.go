package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommitFlagsSecretsAlreadyInTheIndex(t *testing.T) {
	r := newRepo(t)
	r.write(".env", "SECRET=1")
	r.write("notes.txt", "n")
	r.write("only-notes.txt", "notes.txt\n")
	r.write("with-env.txt", ".env\n")
	r.git("add", "-f", ".env", "notes.txt")
	r.expect("no-secrets-commit", true,
		`git commit -m "feat(no-ticket): x"`, "git commit", "git commit --amend --no-edit", "git commit --fixup=amend:HEAD",
		"git commit -i notes.txt -m x", "git commit --include notes.txt -m x", "git commit --pathspec-from-file=with-env.txt -m x",
	)
	if res := r.run("no-secrets-commit", "git commit -m x"); !strings.Contains(res.Stderr, ".env") || !strings.Contains(res.Stderr, "AGENT_HOOKS_ALLOW=secrets") {
		t.Errorf("the message should name the file and the override: %q", res.Stderr)
	}
	r.expect("no-secrets-commit", false,
		"git commit notes.txt -m x", "git commit -o notes.txt -m x", "git commit --pathspec-from-file=only-notes.txt -m x", // git commits only those paths
		"git commit --only --amend --no-edit", "git commit --fixup=reword:HEAD", // neither reads the index
		"git commit --dry-run", "git commit --short", "git commit --porcelain",
		"AGENT_HOOKS_ALLOW=secrets git commit -m x", "git status", "git diff --cached",
	)
	r.git("config", "diff.relative", "true")
	r.write("sub/a.txt", "a")
	if res := run(t, "no-secrets-commit", bashIn(filepath.Join(r.dir, "sub"), "git commit -m x")); res.Exit != 2 {
		t.Errorf("a commit from a subdirectory still records the staged .env: %+v", res)
	}
}

func TestCommitFollowsWhatEachModeWouldRecord(t *testing.T) {
	r := newRepo(t)
	r.write(".env", "SECRET=1")
	r.git("add", "-f", ".env")
	if err := os.Remove(filepath.Join(r.dir, ".env")); err != nil {
		t.Fatal(err)
	}
	r.expect("no-secrets-commit", true, "git commit -m x")   // the staged copy is still committed
	r.expect("no-secrets-commit", false, "git commit -am x") // -a stages the deletion instead
	r.git("rm", "-q", "--cached", ".env")
	r.write("key.pem", "k")
	r.git("add", "-N", "key.pem")
	r.expect("no-secrets-commit", false, "git commit -m x") // intent-to-add records nothing
	r.expect("no-secrets-commit", true, "git commit -am x")

	fresh := t.TempDir()
	gitIn(t, fresh, "init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(fresh, "id_rsa"), []byte("k"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitIn(t, fresh, "add", "id_rsa")
	if res := run(t, "no-secrets-commit", bashIn(fresh, "git commit -m x")); res.Exit != 2 {
		t.Errorf("first commit of a repository: %+v", res)
	}
}
