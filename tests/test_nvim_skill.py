from __future__ import annotations

import os
import shutil
import subprocess
import tempfile
import time
import unittest
from pathlib import Path

SCRIPT = Path(__file__).resolve().parents[1] / "skills/personal_dev/nvim/scripts/open_nvim_tmux.sh"


def wait_for(predicate, timeout: float = 5.0):
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        value = predicate()
        if value:
            return value
        time.sleep(0.05)
    raise AssertionError("timed out waiting")


@unittest.skipUnless(shutil.which("tmux") and shutil.which("nvim"), "needs tmux and nvim")
class TmuxSplitTest(unittest.TestCase):
    def setUp(self) -> None:
        self.tmp = Path(tempfile.mkdtemp(prefix="nvim-skill-"))
        self.addCleanup(shutil.rmtree, self.tmp, ignore_errors=True)
        self.repo = self.tmp / "repo"
        self.repo.mkdir()
        (self.repo / "a.txt").write_text("".join(f"a{i}\n" for i in range(1, 41)))
        (self.repo / "it's b.txt").write_text("".join(f"b{i}\n" for i in range(1, 41)))
        bin_dir = self.tmp / "bin"
        bin_dir.mkdir()
        # pane_current_command reports the executable name, so a renamed sleep poses as the agent.
        os.symlink(shutil.which("sleep"), bin_dir / "pi")
        self.socket = str(self.tmp / "tmux.sock")
        self.env = {
            **{k: v for k, v in os.environ.items() if not k.startswith(("TMUX", "HERDR"))},
            "PATH": f"{bin_dir}:{os.environ['PATH']}",
            "XDG_CONFIG_HOME": str(self.tmp / "config"),
            "XDG_DATA_HOME": str(self.tmp / "data"),
            "XDG_STATE_HOME": str(self.tmp / "state"),
            "XDG_RUNTIME_DIR": str(self.tmp / "run"),
        }
        (self.tmp / "run").mkdir(mode=0o700)
        subprocess.run(
            ["tmux", "-S", self.socket, "-f", "/dev/null", "new-session", "-d", "-x", "200", "-y", "50", "pi", "600"],
            check=True, env=self.env,
        )
        self.addCleanup(subprocess.run, ["tmux", "-S", self.socket, "kill-server"], capture_output=True)
        self.agent = self.tmux("display-message", "-p", "#{pane_id}")
        server_pid = self.tmux("display-message", "-p", "#{pid}")
        self.env |= {"TMUX": f"{self.socket},{server_pid},0", "TMUX_PANE": self.agent}

    def tmux(self, *args: str) -> str:
        return subprocess.run(["tmux", "-S", self.socket, *args], check=True, capture_output=True, text=True, env=self.env).stdout.strip()

    def open(self, *args: str) -> subprocess.CompletedProcess[str]:
        return subprocess.run([str(SCRIPT), str(self.repo), *args], capture_output=True, text=True, env=self.env, cwd=self.repo)

    def panes(self) -> dict[str, dict[str, str]]:
        fields = ("pane_id", "pane_left", "pane_width", "pane_active", "@agent_nvim_socket", "@nvim_ai_pane")
        out = self.tmux("list-panes", "-t", self.agent, "-F", "\t".join(f"#{{{f}}}" for f in fields))
        return {row.split("\t")[0]: dict(zip(fields, row.split("\t"))) for row in out.splitlines()}

    def editor(self) -> dict[str, str]:
        return next(p for p in self.panes().values() if p["pane_id"] != self.agent)

    def nvim_eval(self, expr: str) -> str:
        sock = self.editor()["@agent_nvim_socket"]
        wait_for(lambda: Path(sock).exists())
        return subprocess.run(["nvim", "--server", sock, "--remote-expr", expr], check=True, capture_output=True, text=True, env=self.env).stdout.strip()

    def test_first_open_splits_nvim_left_of_agent_at_the_line(self) -> None:
        result = self.open("a.txt", "12")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout, "")
        editor, agent = self.editor(), self.panes()[self.agent]
        self.assertEqual(editor["pane_left"], "0")
        self.assertGreater(int(agent["pane_left"]), 0)
        self.assertAlmostEqual(int(editor["pane_width"]), 130, delta=2)
        self.assertEqual(editor["pane_active"], "1")
        self.assertEqual(self.nvim_eval("expand('%:p') .. ':' .. line('.')"), f"{self.repo / 'a.txt'}:12")

    def test_next_open_reuses_the_editor_and_focuses_it(self) -> None:
        self.open("a.txt")
        self.nvim_eval("1")
        self.tmux("select-pane", "-t", self.agent)
        result = self.open("it's b.txt", "30", "2")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(len(self.panes()), 2)
        self.assertEqual(self.editor()["pane_active"], "1")
        self.assertEqual(
            self.nvim_eval("expand('%:p') .. ':' .. line('.') .. ':' .. col('.')"),
            str(self.repo / "it's b.txt") + ":30:2",
        )

    def test_reopening_current_buffer_preserves_unsaved_edits_and_jumps(self) -> None:
        self.open("a.txt")
        self.nvim_eval("setline(1, 'unsaved')")
        self.tmux("select-pane", "-t", self.agent)
        result = self.open("a.txt", "25", "2")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.nvim_eval("getline(1) .. ':' .. &modified"), "unsaved:1")
        self.assertEqual(self.nvim_eval("line('.') .. ':' .. col('.')"), "25:2")
        self.assertEqual(self.editor()["pane_active"], "1")
        self.assertEqual(len(self.panes()), 2)

    def test_dot_relative_reopen_preserves_unsaved_edits(self) -> None:
        self.open("a.txt")
        self.nvim_eval("setline(1, 'unsaved')")
        result = self.open("./a.txt", "25", "2")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.nvim_eval("getline(1) .. ':' .. &modified"), "unsaved:1")
        self.assertEqual(self.nvim_eval("line('.') .. ':' .. col('.')"), "25:2")

    def test_symlinked_repo_reopen_preserves_unsaved_edits(self) -> None:
        alias = self.tmp / "alias"
        alias.symlink_to(self.repo, target_is_directory=True)
        self.repo = alias
        self.open("a.txt")
        self.nvim_eval("setline(1, 'unsaved')")
        result = self.open("a.txt", "25", "2")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.nvim_eval("getline(1) .. ':' .. &modified"), "unsaved:1")
        self.assertEqual(self.nvim_eval("line('.') .. ':' .. col('.')"), "25:2")

    def test_first_open_honors_column_and_line_defaults(self) -> None:
        (self.repo / "a.txt").write_text("    abcdef\n" * 100)
        for args, column in ((('60', '6'), '6'), (('60',), '1')):
            with self.subTest(args=args):
                self.assertEqual(self.open("a.txt", *args).returncode, 0)
                self.assertEqual(self.nvim_eval("line('.') .. ':' .. col('.')"), '60:' + column)
                self.assertLess(abs(int(self.nvim_eval("winline()")) - 24), 3)
                subprocess.run(["nvim", "--server", self.editor()["@agent_nvim_socket"],
                                "--remote-send", "<C-\\><C-N>:qa!<CR>"], check=True, env=self.env)
                wait_for(lambda: list(self.panes()) == [self.agent])

    def test_reuse_focuses_before_an_edit_error(self) -> None:
        self.open("a.txt")
        self.nvim_eval("setline(1, 'unsaved')")
        self.nvim_eval("execute('set nohidden')")
        self.tmux("select-pane", "-t", self.agent)
        result = self.open("it's b.txt")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.editor()["pane_active"], "1")

    def test_broken_out_editor_reuses_recorded_socket(self) -> None:
        self.open("a.txt")
        self.nvim_eval("1")
        editor = self.editor()
        sock = editor["@agent_nvim_socket"]
        self.tmux("break-pane", "-d", "-s", editor["pane_id"])
        self.env["XDG_RUNTIME_DIR"] = str(self.tmp / "other-run")
        result = self.open("a.txt", "20")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(len(self.tmux("list-panes", "-s").splitlines()), 2)
        self.assertTrue(Path(sock).exists())
        self.assertEqual(self.tmux("display-message", "-p", "#{pane_id}"), editor["pane_id"])
        value = subprocess.run(["nvim", "--server", sock, "--remote-expr", "line('.')"],
                               capture_output=True, text=True, check=True, env=self.env)
        self.assertEqual(value.stdout.strip(), "20")

    def test_dead_editor_is_replaced(self) -> None:
        self.open("a.txt")
        self.nvim_eval("1")
        editor = self.editor()
        self.tmux("set-option", "-p", "-t", editor["pane_id"], "remain-on-exit", "on")
        subprocess.run(["nvim", "--server", editor["@agent_nvim_socket"], "--remote-send",
                        "<C-\\><C-N>:qa!<CR>"], check=True, env=self.env)
        wait_for(lambda: self.tmux("display-message", "-p", "-t", editor["pane_id"], "#{pane_dead}") == "1")
        result = self.open("a.txt", "8")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(len(self.panes()), 3)
        self.assertNotEqual(self.tmux("display-message", "-p", "#{pane_id}"), editor["pane_id"])

    def test_hash_in_repo_path(self) -> None:
        renamed = self.repo.with_name("repo#{pane_id}")
        self.repo.rename(renamed)
        self.repo = renamed
        result = self.open("a.txt")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.nvim_eval("expand('%:p')"), str(self.repo / "a.txt"))

    def test_socket_directory_is_private(self) -> None:
        directory = self.tmp / "run/agent-nvim"
        directory.mkdir(mode=0o755)
        result = self.open("a.txt")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(directory.stat().st_mode & 0o777, 0o700)

    def test_symlink_socket_directory_is_rejected(self) -> None:
        directory = self.tmp / "run/agent-nvim"
        directory.symlink_to(self.repo, target_is_directory=True)
        result = self.open("a.txt")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("not a symlink", result.stderr)
        self.assertEqual(len(self.panes()), 1)

    def test_missing_tmux_pane_is_an_error(self) -> None:
        del self.env["TMUX_PANE"]
        result = self.open("a.txt")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("TMUX_PANE is required", result.stderr)
        self.assertEqual(result.stdout, "")

    def test_print_mode_keeps_existing_session_command(self) -> None:
        self.tmux("rename-session", "repo")
        del self.env["TMUX_PANE"]
        result = subprocess.run([str(SCRIPT), "--print", str(self.repo), "a.txt"],
                                capture_output=True, text=True, env=self.env)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout, "# opens in tmux session repo\n"
                         f'tmux new-window -t repo: -c {self.repo} \\; send-keys "nvim -- a.txt" Enter \\; switch-client -t repo\n')
        self.assertEqual(len(self.panes()), 1)

    def test_agent_pane_is_marked_for_nvim_sends(self) -> None:
        self.open("a.txt")
        self.assertEqual(self.panes()[self.agent]["@nvim_ai_pane"], "pi")

    def test_quitting_nvim_closes_only_its_pane(self) -> None:
        self.open("a.txt")
        self.nvim_eval("1")
        subprocess.run(["nvim", "--server", self.editor()["@agent_nvim_socket"], "--remote-send", "<C-\\><C-N>:qa!<CR>"], env=self.env)
        wait_for(lambda: list(self.panes()) == [self.agent])


FAKE_HERDR = r"""#!/usr/bin/env bash
printf '%s\n' "$*" >> "$HERDR_LOG"
case "$1 $2" in
  "pane list") printf '{"result":{"panes":%s}}' "$(cat "$HERDR_PANES")" ;;
  "pane get") printf '{"result":{"pane":{"pane_id":"%s","agent":"claude"}}}' "$3" ;;
  "pane split") printf '{"result":{"pane":{"pane_id":"w1:p9"}}}' ;;
esac
"""


@unittest.skipUnless(shutil.which("jq"), "needs jq")
class HerdrSplitTest(unittest.TestCase):
    def setUp(self) -> None:
        self.tmp = Path(tempfile.mkdtemp(prefix="nvim-skill-herdr-"))
        self.addCleanup(shutil.rmtree, self.tmp, ignore_errors=True)
        (self.tmp / "a.txt").write_text("a\n" * 5)
        bin_dir = self.tmp / "bin"
        bin_dir.mkdir()
        (bin_dir / "herdr").write_text(FAKE_HERDR)
        (bin_dir / "herdr").chmod(0o755)
        self.log = self.tmp / "herdr.log"
        self.panes = self.tmp / "panes.json"
        self.panes.write_text("[]")
        self.env = {
            **{k: v for k, v in os.environ.items() if not k.startswith(("TMUX", "HERDR"))},
            "PATH": f"{bin_dir}:{os.environ['PATH']}",
            "XDG_RUNTIME_DIR": str(self.tmp / "run"),
            "HERDR_ENV": "1", "HERDR_PANE_ID": "w1:p2", "HERDR_WORKSPACE_ID": "w1",
            "HERDR_LOG": str(self.log), "HERDR_PANES": str(self.panes),
        }

    def open(self) -> list[str]:
        result = subprocess.run([str(SCRIPT), str(self.tmp), "a.txt", "3"], capture_output=True, text=True, env=self.env)
        self.assertEqual(result.returncode, 0, result.stderr)
        return self.log.read_text().splitlines()

    def test_first_open_puts_nvim_left_at_65_percent_and_marks_the_agent(self) -> None:
        calls = self.open()
        sock = f"{self.tmp}/run/agent-nvim/herdr-w1-p2.sock"
        self.assertEqual([c for c in calls if not c.startswith("pane list")], [
            f"pane split --pane w1:p2 --direction right --ratio 0.65 --cwd {self.tmp}",
            "pane swap --source-pane w1:p2 --target-pane w1:p9",
            "pane rename w1:p9 agent-nvim-w1-p2",
            f"pane run w1:p9 exec nvim --listen {sock} +call\\ cursor\\(3\\,\\ 1\\) +normal\\!\\ zz -- a.txt",
            "pane focus --pane w1:p2 --direction left",
            "pane get w1:p2",
            "pane rename w1:p2 nvim-ai-w1-p9-claude",
        ])

    @unittest.skipUnless(shutil.which("nvim"), "needs nvim")
    def test_next_open_reuses_the_labelled_editor(self) -> None:
        sock = self.tmp / "run/agent-nvim/herdr-w1-p2.sock"
        sock.parent.mkdir(parents=True)
        env = self.env | {"XDG_CONFIG_HOME": str(self.tmp / "c"), "XDG_DATA_HOME": str(self.tmp / "d"), "XDG_STATE_HOME": str(self.tmp / "s")}
        nvim = subprocess.Popen(["nvim", "--headless", "--listen", str(sock)], env=env, cwd=self.tmp)
        self.addCleanup(nvim.wait)
        self.addCleanup(nvim.kill)
        wait_for(sock.exists)
        self.panes.write_text('[{"pane_id":"w1:p9","label":"agent-nvim-w1-p2"}]')
        calls = self.open()
        self.assertFalse([c for c in calls if c.startswith(("pane split", "pane run"))])
        self.assertIn("pane focus --pane w1:p2 --direction left", calls)
        where = subprocess.run(["nvim", "--server", str(sock), "--remote-expr", "expand('%:p') .. ':' .. line('.')"], capture_output=True, text=True, env=env)
        self.assertEqual(where.stdout.strip(), f"{self.tmp / 'a.txt'}:3")


if __name__ == "__main__":
    unittest.main()
