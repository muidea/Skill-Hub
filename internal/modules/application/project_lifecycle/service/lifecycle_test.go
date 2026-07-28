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

func newProjectLifecycle(t *testing.T) *ProjectLifecycle {
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
	return New(projectstateport.New(hub, "project_lifecycle_test"), repositoryport.NewProjectSource(hub, "project_lifecycle_test"))
}

func TestImportArchiveRefreshesIndexWhenSkillUnchanged(t *testing.T) {
	config.ResetForTest()
	defer config.ResetForTest()

	homeDir := t.TempDir()
	projectDir := filepath.Join(homeDir, "workspace")
	t.Setenv("SKILL_HUB_HOME", homeDir)
	cfg := &config.Config{
		MultiRepo: &config.MultiRepoConfig{
			Enabled:     true,
			DefaultRepo: "main",
			Repositories: map[string]config.RepositoryConfig{
				"main": {Name: "main", Enabled: true, IsArchive: true},
			},
		},
	}
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	sourceSkillDir := filepath.Join(projectDir, "agent-skills", "demo-skill")
	repoSkillDir := filepath.Join(homeDir, "repositories", "main", "skills", "demo-skill")
	writeLifecycleTestSkill(t, sourceSkillDir, "1.2.3")
	writeLifecycleTestSkill(t, repoSkillDir, "1.2.3")
	writeStaleRegistry(t, filepath.Join(homeDir, "repositories", "main", "registry.json"))

	summary, err := newProjectLifecycle(t).Import(projectDir, "agent-skills", ImportOptions{
		Archive:     true,
		ArchiveOnly: true,
		Force:       true,
	})
	if err != nil {
		t.Fatalf("Import returned error: %v", err)
	}
	if summary.Unchanged != 1 || summary.Archived != 0 || summary.Valid != 1 || summary.Registered != 0 {
		t.Fatalf("summary = %#v, want unchanged=1 archived=0 valid=1 registered=0", summary)
	}

	registry := readLifecycleRegistry(t, filepath.Join(homeDir, "repositories", "main", "registry.json"))
	if len(registry.Skills) != 1 {
		t.Fatalf("registry skill count = %d, want 1", len(registry.Skills))
	}
	if registry.Skills[0].ID != "demo-skill" || registry.Skills[0].Version != "1.2.3" {
		t.Fatalf("registry skill = %#v, want demo-skill 1.2.3", registry.Skills[0])
	}
	if _, err := os.Stat(filepath.Join(homeDir, "state.json")); !os.IsNotExist(err) {
		t.Fatalf("archive-only should not write project state, stat err=%v", err)
	}
}

func TestImportArchiveOnlyRequiresArchive(t *testing.T) {
	if _, err := newProjectLifecycle(t).Import(t.TempDir(), "agent-skills", ImportOptions{ArchiveOnly: true}); err == nil {
		t.Fatal("Import returned nil error, want --archive-only requires --archive")
	}
}

func writeLifecycleTestSkill(t *testing.T, skillDir, version string) {
	t.Helper()

	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	content := []byte(`---
name: demo-skill
description: Demo skill used by import archive tests.
metadata:
  version: ` + version + `
---

# Demo Skill
`)
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), content, 0644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
}

func writeStaleRegistry(t *testing.T, registryPath string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(registryPath), 0755); err != nil {
		t.Fatalf("mkdir registry dir: %v", err)
	}
	data := []byte(`{
  "version": "1.0.0",
  "skills": [
    {
      "id": "demo-skill",
      "name": "demo-skill",
      "version": "0.1.0",
      "author": "unknown",
      "description": "stale",
      "tags": null,
      "repository": "main",
      "repository_path": "skills/demo-skill/SKILL.md"
    }
  ]
}`)
	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		t.Fatalf("write stale registry: %v", err)
	}
}

func readLifecycleRegistry(t *testing.T, path string) spec.Registry {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read registry: %v", err)
	}
	var registry spec.Registry
	if err := json.Unmarshal(data, &registry); err != nil {
		t.Fatalf("unmarshal registry: %v", err)
	}
	return registry
}
