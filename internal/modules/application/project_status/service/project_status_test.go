package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/muidea/skill-hub/internal/config"
	projectstatebiz "github.com/muidea/skill-hub/internal/modules/blocks/project_state/biz"
	projectstateservice "github.com/muidea/skill-hub/internal/modules/blocks/project_state/service"
	repositorybiz "github.com/muidea/skill-hub/internal/modules/blocks/repository/biz"
	repositoryservice "github.com/muidea/skill-hub/internal/modules/blocks/repository/service"
	"github.com/muidea/skill-hub/internal/pkg/projectstateport"
	"github.com/muidea/skill-hub/internal/pkg/repositoryport"
	"github.com/muidea/skill-hub/pkg/skill"
	"github.com/muidea/skill-hub/pkg/spec"

	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"
)

func newProjectStatus(t *testing.T) *ProjectStatus {
	t.Helper()
	hub := event.NewHub(8)
	background := task.NewBackgroundRoutine(2)
	projectState := projectstatebiz.New(hub, background, projectstateservice.New())
	repository := repositorybiz.New(hub, background, repositoryservice.New())
	t.Cleanup(func() {
		projectState.Teardown()
		repository.Teardown()
		hub.Terminate(context.Background())
		background.Shutdown(context.Background())
	})
	return New(projectstateport.New(hub, "project_status_test"), repositoryport.NewProjectSource(hub, "project_status_test"))
}

func TestProjectStatus_InspectSyncedSkill(t *testing.T) {
	config.ResetForTest()
	defer config.ResetForTest()

	homeDir := t.TempDir()
	t.Setenv("SKILL_HUB_HOME", homeDir)

	projectDir := filepath.Join(homeDir, "workspace", "demo")
	repoSkillDir := filepath.Join(homeDir, "repositories", "main", "skills", "demo-skill")
	localSkillDir := filepath.Join(projectDir, ".agents", "skills", "demo-skill")

	if err := os.MkdirAll(repoSkillDir, 0755); err != nil {
		t.Fatalf("mkdir repo skill dir: %v", err)
	}
	if err := os.MkdirAll(localSkillDir, 0755); err != nil {
		t.Fatalf("mkdir local skill dir: %v", err)
	}

	skillContent := []byte("---\nname: Demo Skill\nversion: 1.2.3\n---\nHello\n")
	if err := os.WriteFile(filepath.Join(repoSkillDir, "SKILL.md"), skillContent, 0644); err != nil {
		t.Fatalf("write repo skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localSkillDir, "SKILL.md"), skillContent, 0644); err != nil {
		t.Fatalf("write local skill: %v", err)
	}

	cfg := &config.Config{
		MultiRepo: &config.MultiRepoConfig{
			Enabled:     true,
			DefaultRepo: "main",
			Repositories: map[string]config.RepositoryConfig{
				"main": {
					Name:    "main",
					Enabled: true,
				},
			},
		},
	}
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	statePath := filepath.Join(homeDir, "state.json")
	stateData := map[string]spec.ProjectState{
		projectDir: {
			ProjectPath:     projectDir,
			PreferredTarget: spec.TargetOpenCode,
			Skills: map[string]spec.SkillVars{
				"demo-skill": {
					SkillID:          "demo-skill",
					Version:          "1.2.3",
					Status:           spec.SkillStatusSynced,
					SourceRepository: "main",
					Variables:        map[string]string{},
				},
			},
		},
	}
	payload, err := json.Marshal(stateData)
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	if err := os.WriteFile(statePath, payload, 0644); err != nil {
		t.Fatalf("write state: %v", err)
	}

	summary, err := newProjectStatus(t).Inspect(projectDir, "")
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}

	if summary.ProjectPath != projectDir {
		t.Fatalf("expected project path %q, got %q", projectDir, summary.ProjectPath)
	}
	if len(summary.Items) != 1 {
		t.Fatalf("expected 1 status item, got %d", len(summary.Items))
	}
	if summary.Items[0].Status != spec.SkillStatusSynced {
		t.Fatalf("expected synced status, got %q", summary.Items[0].Status)
	}
	if summary.Items[0].RepoVersion != "1.2.3" {
		t.Fatalf("expected repo version 1.2.3, got %q", summary.Items[0].RepoVersion)
	}
	if !strings.HasSuffix(summary.Items[0].RepoPath, filepath.Join("main", "skills", "demo-skill", "SKILL.md")) {
		t.Fatalf("unexpected repo path %q", summary.Items[0].RepoPath)
	}
	if summary.Items[0].SourceRepository != "main" {
		t.Fatalf("expected source repository main, got %q", summary.Items[0].SourceRepository)
	}
}

func TestProjectStatus_InspectUsesSourceRepository(t *testing.T) {
	config.ResetForTest()
	defer config.ResetForTest()

	homeDir := t.TempDir()
	t.Setenv("SKILL_HUB_HOME", homeDir)

	projectDir := filepath.Join(homeDir, "workspace", "demo")
	mainSkillDir := filepath.Join(homeDir, "repositories", "main", "skills", "demo-skill")
	communitySkillDir := filepath.Join(homeDir, "repositories", "community", "skills", "demo-skill")
	localSkillDir := filepath.Join(projectDir, ".agents", "skills", "demo-skill")

	for _, dir := range []string{mainSkillDir, communitySkillDir, localSkillDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("mkdir skill dir %q: %v", dir, err)
		}
	}

	if err := os.WriteFile(filepath.Join(mainSkillDir, "SKILL.md"), []byte("---\nname: Demo Skill\nversion: 1.0.0\n---\nMAIN\n"), 0644); err != nil {
		t.Fatalf("write main skill: %v", err)
	}
	communityContent := []byte("---\nname: Demo Skill\nversion: 2.0.0\n---\nCOMMUNITY\n")
	if err := os.WriteFile(filepath.Join(communitySkillDir, "SKILL.md"), communityContent, 0644); err != nil {
		t.Fatalf("write community skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localSkillDir, "SKILL.md"), communityContent, 0644); err != nil {
		t.Fatalf("write local skill: %v", err)
	}

	cfg := &config.Config{
		MultiRepo: &config.MultiRepoConfig{
			Enabled:     true,
			DefaultRepo: "main",
			Repositories: map[string]config.RepositoryConfig{
				"main":      {Name: "main", Enabled: true},
				"community": {Name: "community", Enabled: true},
			},
		},
	}
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	statePath := filepath.Join(homeDir, "state.json")
	stateData := map[string]spec.ProjectState{
		projectDir: {
			ProjectPath:     projectDir,
			PreferredTarget: spec.TargetOpenCode,
			Skills: map[string]spec.SkillVars{
				"demo-skill": {
					SkillID:          "demo-skill",
					Version:          "2.0.0",
					Status:           spec.SkillStatusSynced,
					SourceRepository: "community",
					Variables:        map[string]string{},
				},
			},
		},
	}
	payload, err := json.Marshal(stateData)
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	if err := os.WriteFile(statePath, payload, 0644); err != nil {
		t.Fatalf("write state: %v", err)
	}

	summary, err := newProjectStatus(t).Inspect(projectDir, "")
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	if len(summary.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(summary.Items))
	}
	if summary.Scope != "project" {
		t.Fatalf("scope = %q, want project", summary.Scope)
	}
	if summary.Items[0].Status != spec.StatusSynced || summary.Items[0].LegacyStatus != "Synced" || summary.Items[0].Reason != "source_and_target_match" {
		t.Fatalf("unexpected normalized status item: %+v", summary.Items[0])
	}
	if summary.Items[0].RepoVersion != "2.0.0" {
		t.Fatalf("expected repo version 2.0.0, got %q", summary.Items[0].RepoVersion)
	}
	if !strings.HasSuffix(summary.Items[0].RepoPath, filepath.Join("community", "skills", "demo-skill", "SKILL.md")) {
		t.Fatalf("unexpected repo path %q", summary.Items[0].RepoPath)
	}
}

func TestProjectStatus_InspectModifiedAgainstOutdatedRepo(t *testing.T) {
	config.ResetForTest()
	defer config.ResetForTest()

	homeDir := t.TempDir()
	t.Setenv("SKILL_HUB_HOME", homeDir)

	projectDir := filepath.Join(homeDir, "workspace", "demo")
	repoSkillDir := filepath.Join(homeDir, "repositories", "main", "skills", "demo-skill")
	localSkillDir := filepath.Join(projectDir, ".agents", "skills", "demo-skill")
	if err := os.MkdirAll(repoSkillDir, 0755); err != nil {
		t.Fatalf("mkdir repo skill dir: %v", err)
	}
	if err := os.MkdirAll(localSkillDir, 0755); err != nil {
		t.Fatalf("mkdir local skill dir: %v", err)
	}

	appliedContent := []byte("---\nname: Demo Skill\nversion: 1.0.0\n---\napplied\n")
	localContent := []byte("---\nname: Demo Skill\nversion: 1.0.0\n---\nlocal changed\n")
	repoContent := []byte("---\nname: Demo Skill\nversion: 1.1.0\n---\nrepo newer\n")
	if err := os.WriteFile(filepath.Join(repoSkillDir, "SKILL.md"), repoContent, 0644); err != nil {
		t.Fatalf("write repo skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localSkillDir, "SKILL.md"), localContent, 0644); err != nil {
		t.Fatalf("write local skill: %v", err)
	}
	appliedDir := filepath.Join(homeDir, "applied")
	if err := os.MkdirAll(appliedDir, 0755); err != nil {
		t.Fatalf("mkdir applied dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(appliedDir, "SKILL.md"), appliedContent, 0644); err != nil {
		t.Fatalf("write applied skill: %v", err)
	}
	appliedHash, err := skill.DirectoryContentHash(appliedDir)
	if err != nil {
		t.Fatalf("hash applied skill: %v", err)
	}

	cfg := &config.Config{
		MultiRepo: &config.MultiRepoConfig{
			Enabled:     true,
			DefaultRepo: "main",
			Repositories: map[string]config.RepositoryConfig{
				"main": {Name: "main", Enabled: true},
			},
		},
	}
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	stateData := map[string]spec.ProjectState{
		projectDir: {
			ProjectPath: projectDir,
			Skills: map[string]spec.SkillVars{
				"demo-skill": {
					SkillID:          "demo-skill",
					Version:          "1.0.0",
					Status:           spec.SkillStatusSynced,
					SourceRepository: "main",
					AppliedHash:      appliedHash,
					Variables:        map[string]string{},
				},
			},
		},
	}
	payload, err := json.Marshal(stateData)
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	if err := os.WriteFile(filepath.Join(homeDir, "state.json"), payload, 0644); err != nil {
		t.Fatalf("write state: %v", err)
	}

	summary, err := newProjectStatus(t).Inspect(projectDir, "demo-skill")
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	if len(summary.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(summary.Items))
	}
	if summary.Items[0].Status != spec.SkillStatusModifiedAgainstOutdatedRepo {
		t.Fatalf("status = %q, want %q", summary.Items[0].Status, spec.SkillStatusModifiedAgainstOutdatedRepo)
	}
}
