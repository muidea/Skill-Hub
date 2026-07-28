package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/muidea/skill-hub/internal/config"
	projectstatebiz "github.com/muidea/skill-hub/internal/modules/blocks/project_state/biz"
	projectstateservice "github.com/muidea/skill-hub/internal/modules/blocks/project_state/service"
	repositorybiz "github.com/muidea/skill-hub/internal/modules/blocks/repository/biz"
	repositoryservice "github.com/muidea/skill-hub/internal/modules/blocks/repository/service"
	"github.com/muidea/skill-hub/internal/pkg/projectstateport"
	"github.com/muidea/skill-hub/internal/pkg/repositoryport"
	"github.com/muidea/skill-hub/pkg/spec"

	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"
)

func newProjectUse(t *testing.T) *ProjectUse {
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
	return New(projectstateport.New(hub, "project_use_test"), repositoryport.NewProjectSource(hub, "project_use_test"))
}

func TestProjectUse_EnableSkill(t *testing.T) {
	config.ResetForTest()
	defer config.ResetForTest()

	homeDir := t.TempDir()
	t.Setenv("SKILL_HUB_HOME", homeDir)

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

	repoSkillDir := filepath.Join(homeDir, "repositories", "main", "skills", "demo-skill")
	if err := os.MkdirAll(repoSkillDir, 0755); err != nil {
		t.Fatalf("mkdir repo skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoSkillDir, "SKILL.md"), []byte("---\nname: Demo Skill\nversion: 1.0.0\n---\nHello\n"), 0644); err != nil {
		t.Fatalf("write skill file: %v", err)
	}

	projectDir := filepath.Join(homeDir, "workspace", "demo")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}

	result, err := newProjectUse(t).EnableSkill(projectDir, "demo-skill", "main", map[string]string{"env": "test"})
	if err != nil {
		t.Fatalf("EnableSkill returned error: %v", err)
	}
	if result.SkillID != "demo-skill" || result.Repository != "main" {
		t.Fatalf("unexpected result: %+v", result)
	}

	statePayload, err := os.ReadFile(filepath.Join(homeDir, "state.json"))
	if err != nil {
		t.Fatalf("read state.json: %v", err)
	}

	var states map[string]spec.ProjectState
	if err := json.Unmarshal(statePayload, &states); err != nil {
		t.Fatalf("unmarshal state: %v", err)
	}
	projectState, ok := states[result.ProjectPath]
	if !ok {
		t.Fatalf("expected project state for %q", result.ProjectPath)
	}
	if projectState.Skills["demo-skill"].Version != "1.0.0" {
		t.Fatalf("expected stored version 1.0.0, got %q", projectState.Skills["demo-skill"].Version)
	}
	if projectState.Skills["demo-skill"].SourceRepository != "main" {
		t.Fatalf("expected source repository main, got %q", projectState.Skills["demo-skill"].SourceRepository)
	}
	if projectState.Skills["demo-skill"].Variables["env"] != "test" {
		t.Fatalf("expected stored variable env=test, got %+v", projectState.Skills["demo-skill"].Variables)
	}
}
