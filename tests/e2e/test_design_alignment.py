import json
import shutil
import subprocess
from pathlib import Path

import pytest

from tests.e2e.utils.command_runner import CommandRunner


class TestDesignAlignment:
    @pytest.fixture(autouse=True)
    def setup(self, temp_home_dir):
        self.home_dir = Path(temp_home_dir)
        self.project_dir = self.home_dir / "project"
        self.project_dir.mkdir(exist_ok=True)
        self.repo_skills_dir = self.home_dir / ".skill-hub" / "repositories" / "main" / "skills"
        self.project_skills_dir = self.project_dir / ".agents" / "skills"
        self.cmd = CommandRunner()

    def _init(self):
        result = self.cmd.run("init", cwd=str(self.project_dir))
        assert result.success, f"init failed: {result.stderr}\n{result.stdout}"

    def _write_repo_skill(self, skill_id: str, body: str = "repo body"):
        skill_dir = self.repo_skills_dir / skill_id
        skill_dir.mkdir(parents=True, exist_ok=True)
        (skill_dir / "SKILL.md").write_text(
            f"""---
name: {skill_id}
description: Design alignment fixture.
metadata:
  version: "1.0.0"
---
# {skill_id}

{body}
""",
            encoding="utf-8",
        )

    def _rebuild_index(self):
        result = self.cmd.run("repo", ["rebuild-index"], cwd=str(self.project_dir))
        assert result.success, f"repo rebuild-index failed: {result.stderr}\n{result.stdout}"

    def _use_and_apply(self, skill_id: str, cwd: Path | None = None):
        cwd = cwd or self.project_dir
        use_result = self.cmd.run("use", ["--pattern", skill_id], cwd=str(cwd))
        assert use_result.success, f"use failed: {use_result.stderr}\n{use_result.stdout}"
        apply_result = self.cmd.run("apply", ["--pattern", skill_id], cwd=str(cwd))
        assert apply_result.success, f"apply failed: {apply_result.stderr}\n{apply_result.stdout}"

    @pytest.mark.no_debug
    def test_feedback_literal_pattern_prompts_and_archives_without_force(self):
        skill_id = "feedback-literal-interactive"
        self._init()
        self._write_repo_skill(skill_id)
        self._rebuild_index()
        self._use_and_apply(skill_id)

        project_skill_md = self.project_skills_dir / skill_id / "SKILL.md"
        project_skill_md.write_text(
            project_skill_md.read_text(encoding="utf-8") + "\ninteractive feedback edit\n",
            encoding="utf-8",
        )

        result = self.cmd.run(
            "feedback",
            ["--pattern", skill_id],
            cwd=str(self.project_dir),
            input_text="y\n",
        )
        assert result.success, f"feedback literal failed: {result.stderr}\n{result.stdout}"
        assert "是否将这些修改更新到本地仓库" in result.stdout
        assert "反馈完成" in result.stdout

        repo_content = (self.repo_skills_dir / skill_id / "SKILL.md").read_text(encoding="utf-8")
        assert "interactive feedback edit" in repo_content

    @pytest.mark.no_debug
    def test_project_commands_from_subdirectory_use_registered_project_root(self):
        skill_id = "subdir-root-resolution"
        self._init()
        self._write_repo_skill(skill_id)
        self._rebuild_index()

        register_root = self.cmd.run("status", ["--json"], cwd=str(self.project_dir))
        assert register_root.success, f"root status should register project: {register_root.stderr}\n{register_root.stdout}"

        subdir = self.project_dir / "src" / "pkg"
        subdir.mkdir(parents=True)

        self._use_and_apply(skill_id, cwd=subdir)

        root_skill_md = self.project_skills_dir / skill_id / "SKILL.md"
        assert root_skill_md.exists(), "apply from subdirectory should write root .agents/skills"
        assert not (subdir / ".agents").exists(), "subdirectory should not become a separate project workspace"

        state_path = self.home_dir / ".skill-hub" / "state.json"
        state_data = json.loads(state_path.read_text(encoding="utf-8"))
        assert str(self.project_dir) in state_data
        assert str(subdir) not in state_data
        assert skill_id in state_data[str(self.project_dir)]["skills"]

        root_skill_md.write_text(
            root_skill_md.read_text(encoding="utf-8") + "\nsubdir feedback edit\n",
            encoding="utf-8",
        )

        feedback_result = self.cmd.run("feedback", ["--pattern", skill_id, "--force"], cwd=str(subdir))
        assert feedback_result.success, f"feedback from subdir failed: {feedback_result.stderr}\n{feedback_result.stdout}"
        assert "subdir feedback edit" in (self.repo_skills_dir / skill_id / "SKILL.md").read_text(encoding="utf-8")

        remove_result = self.cmd.run("remove", [skill_id], cwd=str(subdir), input_text="y\n")
        assert remove_result.success, f"remove from subdir failed: {remove_result.stderr}\n{remove_result.stdout}"
        assert not (self.project_skills_dir / skill_id).exists()

        state_data = json.loads(state_path.read_text(encoding="utf-8"))
        assert skill_id not in state_data[str(self.project_dir)]["skills"]


class TestRepoSyncAll:
    @pytest.fixture(autouse=True)
    def setup(self, temp_home_dir):
        if shutil.which("git") is None:
            pytest.skip("git CLI is required for repo sync e2e coverage")
        self.home_dir = Path(temp_home_dir)
        self.work_dir = self.home_dir / "git-work"
        self.work_dir.mkdir()
        self.project_dir = self.home_dir / "project"
        self.project_dir.mkdir()
        self.cmd = CommandRunner(timeout=60)

    def _git(self, args, cwd: Path):
        result = subprocess.run(
            ["git", *args],
            cwd=str(cwd),
            capture_output=True,
            text=True,
            encoding="utf-8",
            timeout=30,
        )
        assert result.returncode == 0, f"git {' '.join(args)} failed:\n{result.stdout}\n{result.stderr}"

    def _create_remote(self, name: str, skill_id: str) -> tuple[Path, Path]:
        bare = self.work_dir / f"{name}.git"
        work = self.work_dir / f"{name}-work"
        self._git(["init", "--bare", "--initial-branch=main", str(bare)], cwd=self.work_dir)
        self._git(["init", "--initial-branch=main", str(work)], cwd=self.work_dir)
        self._git(["config", "user.email", "tester@example.com"], cwd=work)
        self._git(["config", "user.name", "Skill Hub Tester"], cwd=work)
        self._git(["remote", "add", "origin", bare.as_uri()], cwd=work)
        self._write_work_skill(work, skill_id, "initial")
        self._git(["add", "."], cwd=work)
        self._git(["commit", "-m", f"seed {skill_id}"], cwd=work)
        self._git(["push", "-u", "origin", "main"], cwd=work)
        return bare, work

    def _write_work_skill(self, work: Path, skill_id: str, body: str):
        skill_dir = work / "skills" / skill_id
        skill_dir.mkdir(parents=True, exist_ok=True)
        (skill_dir / "SKILL.md").write_text(
            f"""---
name: {skill_id}
description: Repo sync fixture.
metadata:
  version: "1.0.0"
---
# {skill_id}

{body}
""",
            encoding="utf-8",
        )

    def _push_remote_update(self, work: Path, skill_id: str, body: str):
        self._write_work_skill(work, skill_id, body)
        self._git(["add", "."], cwd=work)
        self._git(["commit", "-m", f"update {skill_id}"], cwd=work)
        self._git(["push"], cwd=work)

    @pytest.mark.no_debug
    def test_repo_sync_all_syncs_disabled_repositories(self):
        main_bare, _ = self._create_remote("main", "main-seed")
        disabled_bare, disabled_work = self._create_remote("disabled", "disabled-seed")

        init_result = self.cmd.run("init", [main_bare.as_uri()], cwd=str(self.project_dir))
        assert init_result.success, f"init remote failed: {init_result.stderr}\n{init_result.stdout}"

        add_result = self.cmd.run("repo", ["add", "disabled", disabled_bare.as_uri(), "--type", "user"], cwd=str(self.project_dir))
        assert add_result.success, f"repo add disabled failed: {add_result.stderr}\n{add_result.stdout}"

        disable_result = self.cmd.run("repo", ["disable", "disabled"], cwd=str(self.project_dir), input_text="y\n")
        assert disable_result.success, f"repo disable failed: {disable_result.stderr}\n{disable_result.stdout}"

        self._push_remote_update(disabled_work, "disabled-new", "new content after disabled")

        sync_result = self.cmd.run("repo", ["sync", "--all", "--json"], cwd=str(self.project_dir))
        assert sync_result.success, f"repo sync --all failed: {sync_result.stderr}\n{sync_result.stdout}"
        data = json.loads(sync_result.stdout)
        disabled_item = next(item for item in data["items"] if item["name"] == "disabled")
        assert disabled_item["enabled"] is False
        assert disabled_item["status"] == "synced"
        assert data["failed"] == 0

        synced_skill = self.home_dir / ".skill-hub" / "repositories" / "disabled" / "skills" / "disabled-new" / "SKILL.md"
        assert synced_skill.exists()
        assert "new content after disabled" in synced_skill.read_text(encoding="utf-8")
