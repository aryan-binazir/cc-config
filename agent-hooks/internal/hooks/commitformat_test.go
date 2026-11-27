package hooks

import (
	"strings"
	"testing"
)

func TestCommitFormat(t *testing.T) {
	r := newRepo(t)
	r.expect("commit-format", true,
		`git commit -m "fixed stuff"`, `git commit -m "feat: no scope"`, `git commit -m "feat(bba): lower"`,
		`git commit -am "wip"`, `git commit --message="x"`, `git commit --message "x"`, `git commit -m "Feat(no-ticket): x"`,
		`git commit -m "feat(no-ticket):x"`, `git commit -m "feature(no-ticket): x"`, `git commit -m "feat(B): x"`,
		"git commit -F - <<'EOF'\nbad subject\n\nbody\nEOF",
		"git commit -m \"$(cat <<'EOF'\nbad subject\n\nbody\nEOF\n)\"",
		`git commit -q -mbad`, `cd /tmp && git -C `+r.dir+` commit -m bad`,
		`git commit -m "bad" -m "feat(no-ticket): second paragraph"`,
	)
	r.expect("commit-format", false,
		`git commit -m "feat(no-ticket): Add x"`, `git commit -m "fix(BBA): y"`, `git commit -m "chore(AB2): y"`,
		`git commit --amend --no-edit`, `git commit --no-edit`, `git commit`, `git commit --amend`, `git commit --fixup HEAD`,
		`git commit --squash=HEAD`, `git commit -m "fixup! anything"`, `git commit -m "squash! anything"`, `git commit -m "$MSG"`,
		`git commit -F msg.txt`, `git commit -C HEAD`, `git commit -c HEAD`, `echo git commit -m bad`, `git log -m`,
		"git commit -F - <<'EOF'\nfeat(no-ticket): Good\n\nbody\nEOF",
		"git commit -m \"$(cat <<'EOF'\ndocs(no-ticket): Good\n\nbody\nEOF\n)\"",
		`git commit -am "test(no-ticket): ok"`,
	)
	if res := r.run("commit-format", `git commit -m bad`); !strings.Contains(res.Stderr, "type(PROJECT): description") {
		t.Errorf("message should show the format: %q", res.Stderr)
	}
}

func TestCommitFormatKeepsTheBranchType(t *testing.T) {
	r := newRepo(t)
	r.expect("commit-format", false, `git commit -m "fix(no-ticket): ok"`)
	r.git("switch", "-q", "-c", "amb/x")
	r.expect("commit-format", false, `git commit -m "fix(no-ticket): first commit picks the type"`)
	r.write("a.txt", "a")
	r.git("add", "a.txt")
	r.git("commit", "-q", "-m", "feat(no-ticket): First")
	r.expect("commit-format", false, `git commit -m "feat(BBA): more"`, `git commit --amend -m "fix(no-ticket): retype the only commit"`,
		`git commit --amen -m "fix(no-ticket): git reads --amen as --amend"`)
	r.expect("commit-format", true, `git commit -m "fix(no-ticket): other type"`)
	r.write("b.txt", "b")
	r.git("add", "b.txt")
	r.git("commit", "-q", "-m", "feat(no-ticket): Second")
	r.expect("commit-format", true, `git commit --amend -m "fix(no-ticket): x"`)
	res := r.run("commit-format", `git commit -m "fix(no-ticket): other type"`)
	if !strings.Contains(res.Stderr, "feat") {
		t.Errorf("message should name the branch type: %q", res.Stderr)
	}
}

func TestCommitFormatValidatesTheMessageGitUses(t *testing.T) {
	r := newRepo(t)
	r.expect("commit-format", true,
		`git commit --no-edit -m "bad"`, `git commit --amend --no-edit -m bad`,
		"git commit -F - <<EOF\nbad subject\n\n$BODY\nEOF",
		"git commit -F - <<'EOF'\n\\feat(no-ticket): x\nEOF",
	)
	r.expect("commit-format", false,
		"git commit -F - 3<<'EOF'\nbad\nEOF", "git commit -F - <<'EOF' < msg.txt\nbad\nEOF", "git commit -F - <<'EOF' 0<&3\nbad\nEOF", // stdin is not the here-document
		"git commit -F - <<EOF\nfeat(no-ticket): $X\nEOF",
		"git commit -F - <<EOF\nfeat(no-ticket): ok\n\n$BODY\nEOF",
		"git commit -Sm bad", // m is -S's key: no message is given
	)
}
