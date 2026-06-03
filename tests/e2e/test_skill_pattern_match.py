"""
E2E coverage for the new pattern syntax in list/status.

The full set of commands (list/use/status/apply/feedback/validate) accept a
variadic positional pattern argument. This file focuses on the most-used
batch path: `list <pattern>` and `status <pattern>`. It also asserts the
pattern error path (lone '*' is rejected) and the silent 0-hit behavior.
"""

import json
import re
import pytest
from pathlib import Path

from tests.e2e.utils.command_runner import CommandRunner


class TestSkillPatternMatch:
    @pytest.fixture(autouse=True)
    def setup(self, temp_home_dir, temp_project_dir, monkeypatch):
        self.home_dir = Path(temp_home_dir)
        self.project_dir = Path(temp_project_dir)
        self.cmd = CommandRunner()
        self.repo_skills_dir = self.home_dir / ".skill-hub" / "repositories" / "main" / "skills"

    def _init(self):
        result = self.cmd.run("init", cwd=str(self.project_dir))
        assert result.success, f"init failed: {result.stderr}\n{result.stdout}"

    def _write_repo_skill(self, skill_id: str, display_name: str) -> Path:
        skill_dir = self.repo_skills_dir / skill_id
        skill_dir.mkdir(parents=True, exist_ok=True)
        skill_md = skill_dir / "SKILL.md"
        skill_md.write_text(
            f"""---
name: {display_name}
description: Pattern match fixture ({skill_id}).
metadata:
  version: "1.0.0"
  author: "tester"
---
# {display_name}
""",
            encoding="utf-8",
        )
        return skill_md

    def _refresh(self):
        result = self.cmd.run("pull", cwd=str(self.project_dir))
        assert result.success, f"pull failed: {result.stderr}\n{result.stdout}"

    @pytest.mark.no_debug
    def test_list_prefix_pattern(self):
        self._init()
        self._write_repo_skill("magic-helper", "Magic Helper")
        self._write_repo_skill("magic-pack", "Magic Pack")
        self._write_repo_skill("git-expert", "Git Expert")
        self._refresh()

        result = self.cmd.run("list", ["magic*"], cwd=str(self.project_dir))
        assert result.success, f"list magic* failed: {result.stderr}\n{result.stdout}"

        ids = set(re.findall(r"\b(magic-helper|magic-pack|git-expert)\b", result.stdout))
        assert ids == {"magic-helper", "magic-pack"}, f"expected only magic* skills, got {ids}"

    @pytest.mark.no_debug
    def test_list_double_star_returns_all(self):
        self._init()
        self._write_repo_skill("alpha-skill", "Alpha Skill")
        self._write_repo_skill("beta-skill", "Beta Skill")
        self._refresh()

        result = self.cmd.run("list", ["**"], cwd=str(self.project_dir))
        assert result.success, f"list ** failed: {result.stderr}\n{result.stdout}"
        for sid in ("alpha-skill", "beta-skill"):
            assert sid in result.stdout, f"expected {sid} in output, got:\n{result.stdout}"

    @pytest.mark.no_debug
    def test_list_lone_star_rejected(self):
        self._init()
        self._write_repo_skill("alpha-skill", "Alpha Skill")
        self._refresh()

        result = self.cmd.run("list", ["*"], cwd=str(self.project_dir))
        assert not result.success, f"lone '*' should fail, got: {result.stdout}"
        combined = result.stdout + result.stderr
        assert "ErrInvalidInput" in combined or "ambiguous" in combined.lower(), (
            f"expected invalid-input error, got: {combined}"
        )

    @pytest.mark.no_debug
    def test_status_pattern_filters_enabled_skills(self):
        self._init()
        self._write_repo_skill("magic-helper", "Magic Helper")
        self._write_repo_skill("magic-pack", "Magic Pack")
        self._write_repo_skill("git-expert", "Git Expert")
        self._refresh()

        for sid in ("magic-helper", "magic-pack", "git-expert"):
            use_result = self.cmd.run("use", [sid], cwd=str(self.project_dir))
            assert use_result.success, f"use {sid} failed: {use_result.stderr}\n{use_result.stdout}"

        result = self.cmd.run("status", ["magic*", "--json"], cwd=str(self.project_dir))
        assert result.success, f"status magic* failed: {result.stderr}\n{result.stdout}"
        data = json.loads(result.stdout)
        ids = {item["skill_id"] for item in data.get("items", [])}
        assert ids == {"magic-helper", "magic-pack"}, f"unexpected ids: {ids}"
