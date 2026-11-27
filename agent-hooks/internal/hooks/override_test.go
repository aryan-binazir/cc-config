package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoSecretsOverrideIsLogged(t *testing.T) {
	r := newRepo(t)
	state := t.TempDir()
	res := runOpts(t, Options{Phase: "pre", Only: only("no-secrets-commit"), StateDir: state}, bashIn(r.dir, "AGENT_HOOKS_ALLOW=secrets git add .env"))
	if res.Exit != 0 {
		t.Fatalf("override did not apply: %+v", res)
	}
	lines := overrideLog(t, state)
	if len(lines) != 1 || lines[0]["hook"] != "no-secrets-commit" || lines[0]["cwd"] != r.dir || lines[0]["command"] != "AGENT_HOOKS_ALLOW=secrets git add .env" || lines[0]["time"] == "" {
		t.Fatalf("bad log: %v", lines)
	}
	info, err := os.Stat(filepath.Join(state, "overrides.log"))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("log should be private: %v %v", info, err)
	}
	for _, command := range []string{"AGENT_HOOKS_ALLOW=destructive git add .env", "AGENT_HOOKS_ALLOW=secrets true; git add .env", "AGENT_HOOKS_ALLOW=x,secretsy git add .env"} {
		if res := runOpts(t, Options{Phase: "pre", Only: only("no-secrets-commit"), StateDir: state}, bashIn(r.dir, command)); res.Exit != 2 {
			t.Errorf("%q should still block: %+v", command, res)
		}
	}
	for _, command := range []string{"AGENT_HOOKS_ALLOW=destructive,secrets git add .env", "env AGENT_HOOKS_ALLOW=secrets git add .env", "AGENT_HOOKS_ALLOW=secrets bash -c 'git add .env'"} {
		if res := runOpts(t, Options{Phase: "pre", Only: only("no-secrets-commit"), StateDir: state}, bashIn(r.dir, command)); res.Exit != 0 {
			t.Errorf("%q should be allowed: %+v", command, res)
		}
	}
}

func TestOverrideExportedEarlierInTheCommand(t *testing.T) {
	r := newRepo(t)
	r.expect("no-secrets-commit", false,
		"export AGENT_HOOKS_ALLOW=secrets; git add .env",
		"export AGENT_HOOKS_ALLOW=secrets && git add .env",
		"declare -x AGENT_HOOKS_ALLOW=secrets; git add .env",
		"typeset -gx AGENT_HOOKS_ALLOW=secrets; git add .env",
		"export AGENT_HOOKS_ALLOW; AGENT_HOOKS_ALLOW=secrets; git add .env",
		"export AGENT_HOOKS_ALLOW=destructive; AGENT_HOOKS_ALLOW+=,secrets; git add .env",
		"{ export AGENT_HOOKS_ALLOW=secrets; }; git add .env",
		"if true; then export AGENT_HOOKS_ALLOW=secrets; fi; git add .env",
		"export AGENT_HOOKS_ALLOW=secrets; bash -c 'git add .env'",
		"bash -c 'export AGENT_HOOKS_ALLOW=secrets; git add .env'",
	)
	r.expect("no-secrets-commit", true,
		"AGENT_HOOKS_ALLOW=secrets; git add .env",
		"declare AGENT_HOOKS_ALLOW=secrets; git add .env",
		"git add .env; export AGENT_HOOKS_ALLOW=secrets",
		"(export AGENT_HOOKS_ALLOW=secrets); git add .env",
		"echo $(export AGENT_HOOKS_ALLOW=secrets); git add .env",
		"export AGENT_HOOKS_ALLOW=secrets | true; git add .env",
		"export AGENT_HOOKS_ALLOW=secrets & git add .env",
		"f() { export AGENT_HOOKS_ALLOW=secrets; }; git add .env",
		"export AGENT_HOOKS_ALLOW=secrets; unset AGENT_HOOKS_ALLOW; git add .env",
		"export AGENT_HOOKS_ALLOW=secrets; export -n AGENT_HOOKS_ALLOW; git add .env",
		"bash -c 'export AGENT_HOOKS_ALLOW=secrets'; git add .env",
	)
}

func TestOverrideLogSkipsNoOps(t *testing.T) {
	r := newRepo(t)
	state := t.TempDir()
	before := len(overrideLog(t, state))
	runOpts(t, Options{Phase: "pre", Only: only("no-secrets-commit"), StateDir: state}, bashIn(r.dir, "AGENT_HOOKS_ALLOW=secrets git add README.md"))
	if len(overrideLog(t, state)) != before {
		t.Error("logged an override that did not change the outcome")
	}
}

func TestDestructiveOverride(t *testing.T) {
	state := t.TempDir()
	res := runOpts(t, Options{Phase: "pre", Only: only("destructive-git"), StateDir: state}, bash("AGENT_HOOKS_ALLOW=destructive git push --force"))
	if res.Exit != 0 || len(overrideLog(t, state)) != 1 {
		t.Fatalf("override: %+v %v", res, overrideLog(t, state))
	}
}

func TestNoProductionOverride(t *testing.T) {
	r := newRepo(t)
	state := t.TempDir()
	res := runOpts(t, Options{Phase: "pre", Only: only("no-production"), StateDir: state}, bashIn(r.dir, "AGENT_HOOKS_ALLOW=production git push origin production"))
	if res.Exit != 0 || len(overrideLog(t, state)) != 1 {
		t.Fatalf("override: %+v", res)
	}
}

func TestOverrideLogFallsBackToThePayloadCwdAndKeepsShellText(t *testing.T) {
	state := t.TempDir()
	payload := bashIn("/work/dir", `cd "$X" && AGENT_HOOKS_ALLOW=destructive git push -f`)
	if res := runOpts(t, Options{Phase: "pre", Only: only("destructive-git"), StateDir: state}, payload); res.Exit != 0 {
		t.Fatalf("%+v", res)
	}
	raw, _ := os.ReadFile(filepath.Join(state, "overrides.log"))
	if !strings.Contains(string(raw), `"cwd":"/work/dir"`) || !strings.Contains(string(raw), `$X\" && AGENT`) {
		t.Errorf("log line: %s", raw)
	}
}
