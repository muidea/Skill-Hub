package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	projectstatebiz "github.com/muidea/skill-hub/internal/modules/blocks/project_state/biz"
	projectstateservice "github.com/muidea/skill-hub/internal/modules/blocks/project_state/service"
	"github.com/muidea/skill-hub/internal/pkg/projectstateport"
	"github.com/muidea/skill-hub/pkg/spec"

	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"
)

func newProjectInventory(t *testing.T) *ProjectInventory {
	t.Helper()
	hub := event.NewHub(8)
	background := task.NewBackgroundRoutine(2)
	projectState := projectstatebiz.New(hub, background, projectstateservice.New())
	t.Cleanup(func() {
		projectState.Teardown()
		hub.Terminate(context.Background())
		background.Shutdown(context.Background())
	})
	return New(projectstateport.New(hub, "project_inventory_test"))
}

func TestProjectInventory_ListAndGet(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("SKILL_HUB_HOME", homeDir)

	statePath := filepath.Join(homeDir, "state.json")
	stateData := map[string]spec.ProjectState{
		"/tmp/project-b": {
			ProjectPath:     "/tmp/project-b",
			PreferredTarget: spec.TargetClaudeCode,
			Skills: map[string]spec.SkillVars{
				"skill-b": {
					SkillID:          "skill-b",
					Version:          "2.0.0",
					Status:           spec.SkillStatusModified,
					SourceRepository: "community",
					Variables:        map[string]string{},
				},
			},
		},
		"/tmp/project-a": {
			ProjectPath:     "/tmp/project-a",
			PreferredTarget: spec.TargetOpenCode,
			Skills: map[string]spec.SkillVars{
				"skill-a": {
					SkillID:          "skill-a",
					Version:          "1.0.0",
					Status:           spec.SkillStatusSynced,
					SourceRepository: "main",
					Variables:        map[string]string{},
				},
			},
		},
	}

	payload, err := json.Marshal(stateData)
	if err != nil {
		t.Fatalf("marshal state data: %v", err)
	}
	if err := os.WriteFile(statePath, payload, 0644); err != nil {
		t.Fatalf("write state.json: %v", err)
	}

	inventory := newProjectInventory(t)

	projects, err := inventory.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects returned error: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}
	if projects[0].ProjectPath != "/tmp/project-a" {
		t.Fatalf("expected sorted projects, got first path %q", projects[0].ProjectPath)
	}

	detail, err := inventory.GetProject(projects[0].ID)
	if err != nil {
		t.Fatalf("GetProject returned error: %v", err)
	}
	if detail.ProjectPath != "/tmp/project-a" {
		t.Fatalf("expected project-a detail, got %q", detail.ProjectPath)
	}

	skills, err := inventory.ListProjectSkills(projects[0].ID)
	if err != nil {
		t.Fatalf("ListProjectSkills returned error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	if skills[0].SourceRepository != "main" {
		t.Fatalf("expected source repository main, got %q", skills[0].SourceRepository)
	}
}
