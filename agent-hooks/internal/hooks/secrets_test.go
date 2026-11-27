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
	r.write("docs/a.txt", "a")
	r.git("add", "-f", ".env", "notes.txt")
	r.expect("no-secrets-commit", true,
		`git commit -m "feat(no-ticket): x"`, "git commit", "git commit --amend --no-edit", "git commit --fixup=amend:HEAD",
		"git commit -i notes.txt -m x", "git commit --include notes.txt -m x",
		"git commit --include docs/ -m x", "git commit --patch docs/", "git commit --interactive", // each keeps what is already staged
		// A pathspec list is not read: it could be empty, and then git commits the index.
		"git commit --pathspec-from-file=list.txt -m x", "git commit --pathspec-from-file=- -m x <<<notes.txt", "git commit --pathspec-from-file=- -m x </dev/null",
	)
	if res := r.run("no-secrets-commit", "git commit -m x"); !strings.Contains(res.Stderr, ".env") || !strings.Contains(res.Stderr, "AGENT_HOOKS_ALLOW=secrets") {
		t.Errorf("the message should name the file and the override: %q", res.Stderr)
	}
	r.expect("no-secrets-commit", false,
		"git commit notes.txt -m x", "git commit -o notes.txt -m x", // git commits only those paths
		"git commit --only --amend --no-edit", "git commit --fixup=reword:HEAD", // neither reads the index
		"git commit --only --amen --no-edit", "git commit --fix=reword:HEAD",
		"git commit --dry-run", "git commit --short", "git commit --porcelain", "git commit --long", "git commit -z", "git commit --null", "git commit -h",
		"git commit --porc", "git commit --dry",
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
	r.expect("no-secrets-commit", true, "git commit -m x", "git commit --include README.md -m x")        // the staged copy is still committed
	r.expect("no-secrets-commit", true, "git commit --pathspec-from-file=- -m x </dev/null")             // and so it is with an empty list
	r.expect("no-secrets-commit", false, "git commit -am x", "git commit --include README.md .env -m x") // each stages the deletion instead
	r.git("rm", "-q", "--cached", ".env")
	r.write("key.pem", "k")
	r.git("add", "-N", "key.pem")
	r.expect("no-secrets-commit", false, "git commit -m x") // intent-to-add records nothing
	r.expect("no-secrets-commit", true, "git commit -am x")
	r.expect("no-secrets-commit", true,
		`git commit --pathspec-from-file="$LIST" -m x`, "printf 'key.pem' | git commit --pathspec-from-file=- -m x",
		"git commit --pathspec-from-file=list.txt -m x", "git commit --pathspec-fro=list.txt -m x", "git commit --pathspec-from-file=/dev/stdin -m x <<<key.pem",
	)

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

func TestAddFromASubdirectorySeesTrackedSecretsAboveIt(t *testing.T) {
	r := newRepo(t)
	r.write(".env", "SECRET=1")
	r.write("sub/a.txt", "a")
	r.git("add", "-f", ".env", "sub/a.txt")
	r.git("commit", "-q", "-m", "chore(no-ticket): Track them")
	sub := filepath.Join(r.dir, "sub")
	whole := []string{"git add --pathspec-from-file=list.txt", "git add -A", "git add -u", "git add :/", "git add .."}
	expect := func(wantBlock bool) {
		t.Helper()
		for _, command := range whole {
			if res := run(t, "no-secrets-commit", bashIn(sub, command)); (res.Exit == 2) != wantBlock {
				t.Errorf("%q from sub/: want block=%v, got %+v", command, wantBlock, res)
			}
		}
	}
	expect(false)
	r.write(".env", "SECRET=2")
	expect(true) // each stages the change to ../.env
	if err := os.Remove(filepath.Join(r.dir, ".env")); err != nil {
		t.Fatal(err)
	}
	expect(false)
}
