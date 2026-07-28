package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/muidea/skill-hub/internal/config"
	"github.com/muidea/skill-hub/pkg/spec"
)

func TestRunRepoRebuildIndexDefaultsToArchiveRepo(t *testing.T) {
	config.ResetForTest()
	defer config.ResetForTest()

	homeDir := t.TempDir()
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

	writeTestSkill(t, filepath.Join(homeDir, "repositories", "main", "skills", "demo-skill"), "1.2.3")

	output := captureStdout(t, func() {
		if err := runRepoRebuildIndex(nil, false); err != nil {
			t.Fatalf("runRepoRebuildIndex returned error: %v", err)
		}
	})
	if !strings.Contains(output, "索引已重建") {
		t.Fatalf("unexpected output: %q", output)
	}

	registry := readRegistry(t, filepath.Join(homeDir, "repositories", "main", "registry.json"))
	if len(registry.Skills) != 1 {
		t.Fatalf("registry skill count = %d, want 1", len(registry.Skills))
	}
	if registry.Skills[0].ID != "demo-skill" || registry.Skills[0].Version != "1.2.3" {
		t.Fatalf("registry skill = %#v, want demo-skill 1.2.3", registry.Skills[0])
	}
	rootRegistry := readRegistry(t, filepath.Join(homeDir, "registry.json"))
	if len(rootRegistry.Skills) != 1 || rootRegistry.Skills[0].ID != "demo-skill" {
		t.Fatalf("root registry = %#v, want demo-skill", rootRegistry.Skills)
	}
}

func writeTestSkill(t *testing.T, skillDir, version string) {
	t.Helper()

	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	content := []byte(`---
name: demo-skill
description: Demo skill used by repository index tests.
metadata:
  version: ` + version + `
---

# Demo Skill
`)
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), content, 0644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
}

func readRegistry(t *testing.T, path string) spec.Registry {
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
