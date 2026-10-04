from __future__ import annotations

import importlib.util
import os
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

import yaml

SKILLS = Path(__file__).resolve().parents[1] / "skills/personal_dev"
NAMES = (
    "call-claude", "call-codex", "call-cursor", "ask-adversary-in-block",
    "lead", "rocket", "verify-sandbox", "pr-comments",
)


class SkillConfigTest(unittest.TestCase):
    def test_local_selection_and_missing_local_fallback(self) -> None:
        for name in NAMES:
            with self.subTest(skill=name), tempfile.TemporaryDirectory() as tmp:
                directory = Path(tmp)
                filename = (
                    "call.py" if name.startswith("call-") else
                    "pr_comments.py" if name == "pr-comments" else "resolve_config.py"
                )
                spec = importlib.util.spec_from_file_location(
                    name.replace("-", "_"), SKILLS / name / "scripts" / filename,
                )
                module = importlib.util.module_from_spec(spec)
                spec.loader.exec_module(module)
                example = directory / f"{name}.example.yaml"
                local = directory / f"{name}.local.yaml"
                if name == "lead":
                    baseline = {"workers": {"audit": {"model": "example", "extra": 1}}}
                    selected = {"workers": {"audit": {"model": "local"}}}
                    resolve = lambda module=module, directory=directory: module.resolve(directory, "audit")["worker"]["config"]
                    expected = selected["workers"]["audit"]
                    fallback = baseline["workers"]["audit"]
                elif name == "rocket":
                    baseline = {"defaults": {"marker": "example", "extra": 1}}
                    selected = {"defaults": {"marker": "local"}}
                    resolve = lambda module=module, directory=directory: module.resolve_profiles(directory, None, None)["defaults"]
                    expected = selected["defaults"]
                    fallback = baseline["defaults"]
                elif name == "pr-comments":
                    baseline = {"agent": "Example", "sweep_days": 99}
                    selected = {"agent": "Local"}
                    module.SKILL_DIR = directory
                    resolve = module.load_config
                    expected = selected | {"state_dir": "_scratch/pr_reviews", "sweep_days": 7}
                    fallback = baseline | {"state_dir": "_scratch/pr_reviews"}
                else:
                    baseline = yaml.safe_load((SKILLS / name / example.name).read_text())
                    selected = baseline | {"runtime" if name == "verify-sandbox" else "model": "local"}
                    resolve = (
                        lambda module=module, directory=directory, name=name: module.resolve(directory)["config"]
                        if name == "ask-adversary-in-block" else module.resolve(directory)[0]
                    )
                    expected, fallback = selected, baseline
                    if name == "ask-adversary-in-block":
                        fallback = module.validate(dict(baseline))
                        expected = module.validate(dict(selected))
                with patch.dict(os.environ, {}, clear=True):
                    example.write_text(yaml.safe_dump(baseline))
                    self.assertEqual(resolve(), fallback)
                    local.write_text(yaml.safe_dump(selected))
                    example.write_text("broken: [")
                    self.assertEqual(resolve(), expected)
                    local.write_text("broken: [")
                    example.write_text(yaml.safe_dump(baseline))
                    with self.assertRaises(yaml.YAMLError):
                        resolve()
                    if name not in ("lead", "rocket", "pr-comments"):
                        local.write_text("{}")
                        with self.assertRaises(ValueError):
                            resolve()

    def test_sandbox_environment_overrides_selected_file(self) -> None:
        spec = importlib.util.spec_from_file_location(
            "sandbox_config", SKILLS / "verify-sandbox/scripts/resolve_config.py",
        )
        module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(module)
        with tempfile.TemporaryDirectory() as tmp, patch.dict(os.environ, {}, clear=True):
            directory = Path(tmp)
            local = directory / "verify-sandbox.local.yaml"
            local.write_text((SKILLS / "verify-sandbox/verify-sandbox.example.yaml").read_text())
            with patch.dict(os.environ, {"SBX_POSTGRES_VERSION": "19"}):
                config, files, _ = module.resolve(directory)
            self.assertEqual(config["postgres_version"], "19")
            self.assertEqual(files, [local])


if __name__ == "__main__":
    unittest.main()
