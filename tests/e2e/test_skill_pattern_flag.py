"""
E2E coverage for the v0.8.13 --pattern flag migration.

The pre-v0.8.12 positional pattern form (`list magic*`) had a real footgun:
the shell expands unquoted globs against the working directory before the
CLI sees them, so `skill-hub list magic*` would dispatch on the expanded
filenames rather than the pattern. v0.8.13 cuts the positional form and
moves glob patterns to a dedicated --pattern flag, which cobra parses
after the shell, sidestepping the issue entirely.

This file covers the six pattern-aware commands end-to-end:
  - list, use, apply, status, feedback, validate
and asserts both the new --pattern path and the rejection of the legacy
positional form.
"""
import json
import re
import pytest
from pathlib import Path
from tests.e2e.utils.command_runner import CommandRunner

class TestSkillPatternFlag:

    @pytest.fixture(autouse=True)
    def setup(self, temp_home_dir, temp_project_dir, monkeypatch):
        self.home_dir = Path(temp_home_dir)
        self.project_dir = Path(temp_project_dir)
        self.cmd = CommandRunner()
        self.repo_skills_dir = self.home_dir / '.skill-hub' / 'repositories' / 'main' / 'skills'

    def _init(self):
        result = self.cmd.run('init', cwd=str(self.project_dir))
        assert result.success, f'init failed: {result.stderr}\n{result.stdout}'

    def _write_repo_skill(self, skill_id: str, display_name: str) -> Path:
        skill_dir = self.repo_skills_dir / skill_id
        skill_dir.mkdir(parents=True, exist_ok=True)
        skill_md = skill_dir / 'SKILL.md'
        skill_md.write_text(f'---\nname: {display_name}\ndescription: Pattern match fixture ({skill_id}).\nmetadata:\n  version: "1.0.0"\n  author: "tester"\n---\n# {display_name}\n', encoding='utf-8')
        return skill_md

    def _refresh(self):
        result = self.cmd.run('pull', cwd=str(self.project_dir))
        assert result.success, f'pull failed: {result.stderr}\n{result.stdout}'

    @pytest.mark.no_debug
    def test_list_pattern_flag_filters(self):
        self._init()
        self._write_repo_skill('magic-helper', 'Magic Helper')
        self._write_repo_skill('magic-pack', 'Magic Pack')
        self._write_repo_skill('git-expert', 'Git Expert')
        self._refresh()
        result = self.cmd.run('list', ['--pattern', 'magic*'], cwd=str(self.project_dir))
        assert result.success, f"list --pattern 'magic*' failed: {result.stderr}\n{result.stdout}"
        ids = set(re.findall('\\b(magic-helper|magic-pack|git-expert)\\b', result.stdout))
        assert ids == {'magic-helper', 'magic-pack'}, f'expected only magic* skills, got {ids}'

    @pytest.mark.no_debug
    def test_list_multiple_pattern_flags_take_union(self):
        self._init()
        self._write_repo_skill('magic-helper', 'Magic Helper')
        self._write_repo_skill('git-expert', 'Git Expert')
        self._write_repo_skill('alpha-skill', 'Alpha Skill')
        self._refresh()
        result = self.cmd.run('list', ['--pattern', 'magic*', '--pattern', 'git-*'], cwd=str(self.project_dir))
        assert result.success, f'list with two --pattern failed: {result.stderr}\n{result.stdout}'
        ids = set(re.findall('\\b(magic-helper|git-expert|alpha-skill)\\b', result.stdout))
        assert 'magic-helper' in ids
        assert 'git-expert' in ids
        assert 'alpha-skill' not in ids, f'alpha-skill should not match, got {ids}'

    @pytest.mark.no_debug
    def test_list_double_star_via_flag_returns_all(self):
        self._init()
        self._write_repo_skill('alpha-skill', 'Alpha Skill')
        self._write_repo_skill('beta-skill', 'Beta Skill')
        self._refresh()
        result = self.cmd.run('list', ['--pattern', '**'], cwd=str(self.project_dir))
        assert result.success, f"list --pattern '**' failed: {result.stderr}\n{result.stdout}"
        for sid in ('alpha-skill', 'beta-skill'):
            assert sid in result.stdout, f'expected {sid} in output, got:\n{result.stdout}'

    @pytest.mark.no_debug
    def test_list_lone_star_via_flag_rejected(self):
        self._init()
        self._write_repo_skill('alpha-skill', 'Alpha Skill')
        self._refresh()
        result = self.cmd.run('list', ['--pattern', '*'], cwd=str(self.project_dir))
        assert not result.success, f"lone '*' should fail, got: {result.stdout}"
        combined = result.stdout + result.stderr
        assert 'ErrInvalidInput' in combined or 'ambiguous' in combined.lower(), f'expected invalid-input error, got: {combined}'

    @pytest.mark.no_debug
    def test_list_positional_pattern_rejected(self):
        self._init()
        self._write_repo_skill('magic-helper', 'Magic Helper')
        self._refresh()
        result = self.cmd.run('list', ['magic*'], cwd=str(self.project_dir))
        assert not result.success, f'positional pattern should fail, got: {result.stdout}'
        combined = result.stdout + result.stderr
        assert '--pattern' in combined, f'expected --pattern hint, got: {combined}'

    @pytest.mark.no_debug
    def test_status_pattern_flag_filters(self):
        self._init()
        self._write_repo_skill('magic-helper', 'Magic Helper')
        self._write_repo_skill('magic-pack', 'Magic Pack')
        self._write_repo_skill('git-expert', 'Git Expert')
        self._refresh()
        for sid in ('magic-helper', 'magic-pack', 'git-expert'):
            use_result = self.cmd.run('use', ['--pattern', sid], cwd=str(self.project_dir))
            assert use_result.success, f'use --pattern {sid} failed: {use_result.stderr}\n{use_result.stdout}'
        result = self.cmd.run('status', ['--pattern', 'magic*', '--json'], cwd=str(self.project_dir))
        assert result.success, f"status --pattern 'magic*' failed: {result.stderr}\n{result.stdout}"
        data = json.loads(result.stdout)
        ids = {item['skill_id'] for item in data.get('items', [])}
        assert ids == {'magic-helper', 'magic-pack'}, f'unexpected ids: {ids}'

    @pytest.mark.no_debug
    def test_use_positional_rejected(self):
        self._init()
        self._write_repo_skill('magic-helper', 'Magic Helper')
        self._refresh()
        result = self.cmd.run('use', ['magic-helper'], cwd=str(self.project_dir))
        assert not result.success, f'positional use should fail, got: {result.stdout}'
        combined = result.stdout + result.stderr
        assert '--pattern' in combined, f'expected --pattern hint, got: {combined}'

    @pytest.mark.no_debug
    def test_apply_pattern_flag_dispatches_per_skill(self):
        self._init()
        self._write_repo_skill('magic-helper', 'Magic Helper')
        self._write_repo_skill('magic-pack', 'Magic Pack')
        self._write_repo_skill('git-expert', 'Git Expert')
        self._refresh()
        for sid in ('magic-helper', 'magic-pack'):
            use_result = self.cmd.run('use', ['--pattern', sid], cwd=str(self.project_dir))
            assert use_result.success, f'use --pattern {sid} failed: {use_result.stderr}\n{use_result.stdout}'
        result = self.cmd.run('apply', ['--pattern', 'magic*'], cwd=str(self.project_dir))
        assert result.success, f"apply --pattern 'magic*' failed: {result.stderr}\n{result.stdout}"

    @pytest.mark.no_debug
    def test_feedback_requires_all_or_pattern(self):
        self._init()
        result = self.cmd.run('feedback', cwd=str(self.project_dir))
        assert not result.success, f'feedback with no flag should fail, got: {result.stdout}'
        combined = result.stdout + result.stderr
        assert '--pattern' in combined or '--all' in combined, f'expected --pattern or --all hint, got: {combined}'

    @pytest.mark.no_debug
    def test_validate_requires_all_or_pattern(self):
        self._init()
        result = self.cmd.run('validate', cwd=str(self.project_dir))
        assert not result.success, f'validate with no flag should fail, got: {result.stdout}'
        combined = result.stdout + result.stderr
        assert '--pattern' in combined or '--all' in combined, f'expected --pattern or --all hint, got: {combined}'