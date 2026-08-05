package multirepo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/muidea/skill-hub/internal/config"
	"github.com/muidea/skill-hub/pkg/spec"
)

func TestManager_ListRepositories(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
	}{
		{
			name: "多仓库模式（默认配置）",
			config: &config.Config{
				MultiRepo: &config.MultiRepoConfig{
					Enabled:     true,
					DefaultRepo: "main",
					Repositories: map[string]config.RepositoryConfig{
						"main": {
							Name:        "main",
							URL:         "https://github.com/test/repo.git",
							Branch:      "main",
							Enabled:     true,
							Description: "主技能仓库",
							Type:        "user",
							IsArchive:   true,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "多仓库模式 - 启用",
			config: &config.Config{
				MultiRepo: &config.MultiRepoConfig{
					Enabled:     true,
					DefaultRepo: "main",
					Repositories: map[string]config.RepositoryConfig{
						"main": {
							Name:        "main",
							Enabled:     true,
							IsArchive:   true,
							Description: "主仓库",
							Type:        "user",
						},
						"community": {
							Name:        "community",
							Enabled:     true,
							IsArchive:   false,
							Description: "社区仓库",
							Type:        "community",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "多仓库模式 - 有禁用仓库",
			config: &config.Config{
				MultiRepo: &config.MultiRepoConfig{
					Enabled:     true,
					DefaultRepo: "main",
					Repositories: map[string]config.RepositoryConfig{
						"main": {
							Name:        "main",
							Enabled:     true,
							IsArchive:   true,
							Description: "主仓库",
							Type:        "user",
						},
						"disabled": {
							Name:        "disabled",
							Enabled:     false,
							IsArchive:   false,
							Description: "禁用仓库",
							Type:        "community",
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{
				config: tt.config,
			}

			repos, err := m.ListRepositories()
			if (err != nil) != tt.wantErr {
				t.Errorf("ListRepositories() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 验证返回的仓库数量（只支持多仓库模式）
			if tt.config.MultiRepo == nil {
				t.Errorf("多仓库配置不能为nil")
				return
			}

			// 多仓库模式应该只返回启用的仓库
			enabledCount := 0
			for _, repo := range tt.config.MultiRepo.Repositories {
				if repo.Enabled {
					enabledCount++
				}
			}
			if len(repos) != enabledCount {
				t.Errorf("多仓库模式期望 %d 个启用的仓库，实际得到 %d", enabledCount, len(repos))
			}
		})
	}
}

func TestManager_FindSkill(t *testing.T) {
	// 这是一个简化测试，实际实现需要文件系统操作
	m := &Manager{
		config: &config.Config{
			MultiRepo: &config.MultiRepoConfig{
				Enabled:     true,
				DefaultRepo: "main",
				Repositories: map[string]config.RepositoryConfig{
					"main": {
						Name:        "main",
						Enabled:     true,
						IsArchive:   true,
						Description: "主仓库",
						Type:        "user",
					},
				},
			},
		},
	}

	skills, err := m.FindSkill("test-skill")
	if err != nil {
		t.Errorf("FindSkill() 返回错误: %v", err)
	}

	// 在测试环境中，我们期望返回空数组（因为没有实际文件）
	if len(skills) != 0 {
		t.Errorf("期望空技能数组，实际得到 %d 个技能", len(skills))
	}
}

func TestFilterRepositories(t *testing.T) {
	repos := []config.RepositoryConfig{
		{Name: "main"},
		{Name: "community"},
	}

	got := filterRepositories(repos, "community")
	want := []config.RepositoryConfig{{Name: "community"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("filterRepositories() = %#v, want %#v", got, want)
	}
}

func TestListSkillsInRepository_UsesRegistryIndex(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("SKILL_HUB_HOME", tmpDir)

	repoDir := filepath.Join(tmpDir, "repositories", "main")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("创建仓库目录失败: %v", err)
	}

	registry := spec.Registry{
		Version: "1.0.0",
		Skills: []spec.SkillMetadata{
			{
				ID:      "indexed-skill",
				Name:    "Indexed Skill",
				Version: "1.2.3",
			},
		},
	}

	data, err := json.Marshal(registry)
	if err != nil {
		t.Fatalf("序列化registry失败: %v", err)
	}

	registryPath := filepath.Join(repoDir, "registry.json")
	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		t.Fatalf("写入registry失败: %v", err)
	}

	skills := listSkillsInRepository(config.RepositoryConfig{Name: "main"})
	if len(skills) != 1 {
		t.Fatalf("listSkillsInRepository() 返回 %d 个技能, want 1", len(skills))
	}
	if skills[0].Repository != "main" {
		t.Fatalf("Repository = %q, want %q", skills[0].Repository, "main")
	}
	if skills[0].RepositoryPath != filepath.Join("skills", "indexed-skill") {
		t.Fatalf("RepositoryPath = %q, want %q", skills[0].RepositoryPath, filepath.Join("skills", "indexed-skill"))
	}
}

func TestListSkillsInRepository_PrefersLocalCacheOverRepositoryRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("SKILL_HUB_HOME", tmpDir)

	repoDir := filepath.Join(tmpDir, "repositories", "main")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("创建仓库目录失败: %v", err)
	}
	writeRegistry(t, repoDir, []spec.SkillMetadata{{ID: "canonical-skill", Name: "Canonical"}})

	cachePath := filepath.Join(tmpDir, "cache", "repositories", "main", "registry.json")
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		t.Fatalf("创建缓存目录失败: %v", err)
	}
	cacheRegistry := spec.Registry{Version: "1.0.0", Skills: []spec.SkillMetadata{{ID: "cached-skill", Name: "Cached"}}}
	cacheData, err := json.Marshal(cacheRegistry)
	if err != nil {
		t.Fatalf("序列化缓存索引失败: %v", err)
	}
	if err := os.WriteFile(cachePath, cacheData, 0644); err != nil {
		t.Fatalf("写入缓存索引失败: %v", err)
	}

	skills := listSkillsInRepository(config.RepositoryConfig{Name: "main"})
	if len(skills) != 1 || skills[0].ID != "cached-skill" {
		t.Fatalf("listSkillsInRepository() = %#v, want cached registry", skills)
	}
}

func TestManager_RebuildRepositoryIndex(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("SKILL_HUB_HOME", tmpDir)

	cfg := &config.Config{
		MultiRepo: &config.MultiRepoConfig{
			Enabled:     true,
			DefaultRepo: "main",
			Repositories: map[string]config.RepositoryConfig{
				"main": {
					Name:      "main",
					Enabled:   true,
					IsArchive: true,
				},
			},
		},
	}

	repoDir := filepath.Join(tmpDir, "repositories", "main", "skills", "rebuilt-skill")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("创建技能目录失败: %v", err)
	}

	content := `---
name: Rebuilt Skill
description: rebuilt from repo
version: 1.0.0
---

# Skill
`
	if err := os.WriteFile(filepath.Join(repoDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("写入技能文件失败: %v", err)
	}
	canonicalRegistryPath := filepath.Join(tmpDir, "repositories", "main", "registry.json")
	canonicalRegistry := []byte(`{"version":"1.0.0","skills":[]}`)
	if err := os.WriteFile(canonicalRegistryPath, canonicalRegistry, 0644); err != nil {
		t.Fatalf("写入仓库索引失败: %v", err)
	}

	m := &Manager{config: cfg}
	if err := m.RebuildRepositoryIndex("main"); err != nil {
		t.Fatalf("RebuildRepositoryIndex() error = %v", err)
	}

	registryPath := filepath.Join(tmpDir, "cache", "repositories", "main", "registry.json")
	data, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatalf("读取本地索引缓存失败: %v", err)
	}

	var registry spec.Registry
	if err := json.Unmarshal(data, &registry); err != nil {
		t.Fatalf("解析仓库索引失败: %v", err)
	}

	if len(registry.Skills) != 1 || registry.Skills[0].ID != "rebuilt-skill" {
		t.Fatalf("unexpected registry contents: %#v", registry.Skills)
	}
	if data, err := os.ReadFile(canonicalRegistryPath); err != nil {
		t.Fatalf("读取仓库索引失败: %v", err)
	} else if string(data) != string(canonicalRegistry) {
		t.Fatalf("仓库内 registry.json was modified: %s", data)
	}
}

func TestManager_ArchiveToDefaultRepository(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("SKILL_HUB_HOME", tmpDir)

	cfg := &config.Config{
		MultiRepo: &config.MultiRepoConfig{
			Enabled:     true,
			DefaultRepo: "main",
			Repositories: map[string]config.RepositoryConfig{
				"main": {
					Name:      "main",
					Enabled:   true,
					IsArchive: true,
				},
			},
		},
	}

	sourceDir := filepath.Join(tmpDir, "source-skill")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("创建源技能目录失败: %v", err)
	}

	content := `---
name: Archived Skill
description: archived into default repo
version: 1.0.0
---

# Skill
`
	if err := os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("写入源技能文件失败: %v", err)
	}

	m := &Manager{config: cfg}
	if err := m.ArchiveToDefaultRepository("archived-skill", sourceDir); err != nil {
		t.Fatalf("ArchiveToDefaultRepository() error = %v", err)
	}

	targetPath := filepath.Join(tmpDir, "repositories", "main", "skills", "archived-skill", "SKILL.md")
	if _, err := os.Stat(targetPath); err != nil {
		t.Fatalf("归档技能文件不存在: %v", err)
	}

	registryPath := filepath.Join(tmpDir, "cache", "repositories", "main", "registry.json")
	data, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatalf("读取本地索引缓存失败: %v", err)
	}

	var registry spec.Registry
	if err := json.Unmarshal(data, &registry); err != nil {
		t.Fatalf("解析仓库索引失败: %v", err)
	}

	if len(registry.Skills) != 1 || registry.Skills[0].ID != "archived-skill" {
		t.Fatalf("unexpected registry contents: %#v", registry.Skills)
	}
}

func TestManager_ArchiveToDefaultRepositoryRejectsCoreInformationLoss(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("SKILL_HUB_HOME", tmpDir)

	cfg := &config.Config{MultiRepo: &config.MultiRepoConfig{
		Enabled: true, DefaultRepo: "main",
		Repositories: map[string]config.RepositoryConfig{"main": {Name: "main", Enabled: true, IsArchive: true}},
	}}
	targetDir := filepath.Join(tmpDir, "repositories", "main", "skills", "protected-skill")
	sourceDir := filepath.Join(tmpDir, "source-skill")
	for _, dir := range []string{targetDir, sourceDir} {
		if err := os.MkdirAll(filepath.Join(dir, "references"), 0755); err != nil {
			t.Fatalf("mkdir skill directory: %v", err)
		}
	}
	existing := "---\nname: Protected Skill\ndescription: complete guidance\nversion: 2.0.0\nmetadata:\n  author: skill-hub\n---\n# Protected Skill\n\n## Workflow\nfull workflow\n\n## Safety Checklist\nkeep this\n"
	candidate := "---\nname: Protected Skill\ndescription: reduced guidance\nversion: 2.1.0\n---\n# Protected Skill\n\n## Workflow\nshort workflow\n"
	if err := os.WriteFile(filepath.Join(targetDir, "SKILL.md"), []byte(existing), 0644); err != nil {
		t.Fatalf("write target skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "references", "safety.md"), []byte("safety"), 0644); err != nil {
		t.Fatalf("write target reference: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte(candidate), 0644); err != nil {
		t.Fatalf("write source skill: %v", err)
	}

	err := (&Manager{config: cfg}).ArchiveToDefaultRepository("protected-skill", sourceDir)
	if err == nil || !strings.Contains(err.Error(), "核心信息") {
		t.Fatalf("expected core information protection error, got %v", err)
	}
	got, err := os.ReadFile(filepath.Join(targetDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("read protected skill: %v", err)
	}
	if string(got) != existing {
		t.Fatalf("protected skill was overwritten: %q", got)
	}
}

func TestManager_ArchiveToDefaultRepositoryRejectsMissingVersion(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("SKILL_HUB_HOME", tmpDir)
	cfg := &config.Config{MultiRepo: &config.MultiRepoConfig{
		Enabled: true, DefaultRepo: "main",
		Repositories: map[string]config.RepositoryConfig{"main": {Name: "main", Enabled: true, IsArchive: true}},
	}}
	targetDir := filepath.Join(tmpDir, "repositories", "main", "skills", "versioned-skill")
	sourceDir := filepath.Join(tmpDir, "source-skill")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("mkdir target skill: %v", err)
	}
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("mkdir source skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "SKILL.md"), []byte("---\nname: Versioned Skill\ndescription: complete guidance\nversion: 2.5.1\n---\n# Skill\n"), 0644); err != nil {
		t.Fatalf("write target skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("---\nname: Versioned Skill\ndescription: changed guidance\n---\n# Skill\n"), 0644); err != nil {
		t.Fatalf("write source skill: %v", err)
	}

	err := (&Manager{config: cfg}).ArchiveToDefaultRepository("versioned-skill", sourceDir)
	if err == nil || !strings.Contains(err.Error(), "frontmatter.version") {
		t.Fatalf("expected missing version protection error, got %v", err)
	}
}

func TestManager_LoadSkill(t *testing.T) {
	m := &Manager{
		config: &config.Config{
			MultiRepo: &config.MultiRepoConfig{
				Enabled:     true,
				DefaultRepo: "main",
				Repositories: map[string]config.RepositoryConfig{
					"main": {
						Name:        "main",
						Enabled:     true,
						IsArchive:   true,
						Description: "主仓库",
						Type:        "user",
					},
				},
			},
		},
	}

	skill, err := m.LoadSkill("test-skill", "main")
	if err == nil {
		t.Error("LoadSkill() 应该返回错误（技能不存在）")
	}

	if skill != nil {
		t.Error("LoadSkill() 应该返回 nil（技能不存在）")
	}
}

func TestManager_CheckSkillInDefaultRepository(t *testing.T) {
	tests := []struct {
		name       string
		config     *config.Config
		skillID    string
		wantExists bool
		wantErr    bool
	}{
		{
			name: "默认仓库存在",
			config: &config.Config{
				MultiRepo: &config.MultiRepoConfig{
					Enabled:     true,
					DefaultRepo: "main",
					Repositories: map[string]config.RepositoryConfig{
						"main": {
							Name:        "main",
							Enabled:     true,
							IsArchive:   true,
							Description: "主仓库",
							Type:        "user",
						},
					},
				},
			},
			skillID:    "test-skill",
			wantExists: false, // 没有实际文件系统，所以返回false
			wantErr:    false,
		},
		{
			name: "多仓库配置但默认仓库不存在",
			config: &config.Config{
				MultiRepo: &config.MultiRepoConfig{
					Enabled:     true,
					DefaultRepo: "nonexistent",
					Repositories: map[string]config.RepositoryConfig{
						"main": {
							Name:        "main",
							Enabled:     true,
							IsArchive:   true,
							Description: "主仓库",
							Type:        "user",
						},
					},
				},
			},
			skillID:    "test-skill",
			wantExists: false,
			wantErr:    true, // 应该返回错误，因为默认仓库不存在
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{
				config: tt.config,
			}

			exists, err := m.CheckSkillInDefaultRepository(tt.skillID)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckSkillInDefaultRepository() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if exists != tt.wantExists {
				t.Errorf("CheckSkillInDefaultRepository() = %v, want %v", exists, tt.wantExists)
			}
		})
	}
}

func TestParseSkillMetadata(t *testing.T) {
	content := []byte(`---
name: test-skill
description: 测试技能
version: 1.0.0
author: test-author
tags: test,example
---
# 测试技能

这是一个测试技能。`)

	metadata, err := parseSkillMetadata(content, "main", "test-skill", "/fake/path/skills/test-skill/SKILL.md")
	if err != nil {
		t.Errorf("parseSkillMetadata() 返回错误: %v", err)
		return
	}

	if metadata == nil {
		t.Error("parseSkillMetadata() 返回 nil")
		return
	}

	// 检查基本字段
	if metadata.ID != "test-skill" {
		t.Errorf("期望技能ID 'test-skill', 实际得到 '%s'", metadata.ID)
	}

	if metadata.Repository != "main" {
		t.Errorf("期望仓库 'main', 实际得到 '%s'", metadata.Repository)
	}

	// 从技能文件内容中解析的名称
	if metadata.Name != "test-skill" {
		t.Errorf("期望技能名 'test-skill', 实际得到 '%s'", metadata.Name)
	}

	// 从技能文件内容中解析的版本
	if metadata.Version != "1.0.0" {
		t.Errorf("期望版本 '1.0.0', 实际得到 '%s'", metadata.Version)
	}

	// 从技能文件内容中解析的作者
	if metadata.Author != "test-author" {
		t.Errorf("期望作者 'test-author', 实际得到 '%s'", metadata.Author)
	}

	// 从技能文件内容中解析的描述
	expectedDesc := "测试技能"
	if metadata.Description != expectedDesc {
		t.Errorf("期望描述 '%s', 实际得到 '%s'", expectedDesc, metadata.Description)
	}

	// 从技能文件内容中解析的标签
	expectedTags := 2
	if len(metadata.Tags) != expectedTags {
		t.Errorf("期望%d个标签, 实际得到 %d", expectedTags, len(metadata.Tags))
	}

	// 检查标签内容
	expectedTag1 := "test"
	expectedTag2 := "example"
	if len(metadata.Tags) >= 1 && metadata.Tags[0] != expectedTag1 {
		t.Errorf("期望第一个标签 '%s', 实际得到 '%s'", expectedTag1, metadata.Tags[0])
	}
	if len(metadata.Tags) >= 2 && metadata.Tags[1] != expectedTag2 {
		t.Errorf("期望第二个标签 '%s', 实际得到 '%s'", expectedTag2, metadata.Tags[1])
	}

}

// writeRegistry writes a registry.json under the repository directory so that
// listSkillsInRepository picks it up via the registry-index fast path.
func writeRegistry(t *testing.T, repoDir string, skills []spec.SkillMetadata) {
	t.Helper()
	reg := spec.Registry{Version: "1.0.0", Skills: skills}
	data, err := json.Marshal(reg)
	if err != nil {
		t.Fatalf("marshal registry: %v", err)
	}
	path := filepath.Join(repoDir, "registry.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write registry: %v", err)
	}
}

func TestManager_FindSkillsByPatterns(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("SKILL_HUB_HOME", tmpDir)

	mainRepo := filepath.Join(tmpDir, "repositories", "main")
	communityRepo := filepath.Join(tmpDir, "repositories", "community")
	if err := os.MkdirAll(mainRepo, 0755); err != nil {
		t.Fatalf("mkdir main: %v", err)
	}
	if err := os.MkdirAll(communityRepo, 0755); err != nil {
		t.Fatalf("mkdir community: %v", err)
	}

	writeRegistry(t, mainRepo, []spec.SkillMetadata{
		{ID: "magic-team/magic-skill", Name: "Magic Skill", Version: "1.0.0", Repository: "main"},
		{ID: "magic-team/magic-helper", Name: "Magic Helper", Version: "1.0.0", Repository: "main"},
		{ID: "git/git-expert", Name: "Git Expert", Version: "1.0.0", Repository: "main"},
	})
	writeRegistry(t, communityRepo, []spec.SkillMetadata{
		{ID: "magic-community/magic-pack", Name: "Magic Pack", Version: "0.1.0", Repository: "community"},
		{ID: "git/git-helper", Name: "Git Helper", Version: "0.2.0", Repository: "community"},
	})

	cfg := &config.Config{
		MultiRepo: &config.MultiRepoConfig{
			Enabled:     true,
			DefaultRepo: "main",
			Repositories: map[string]config.RepositoryConfig{
				"main":      {Name: "main", Enabled: true, IsArchive: true},
				"community": {Name: "community", Enabled: true},
			},
		},
	}
	m := &Manager{config: cfg}

	names := func(skills []spec.SkillMetadata) []string {
		out := make([]string, 0, len(skills))
		for _, s := range skills {
			out = append(out, s.Name)
		}
		return out
	}

	t.Run("prefix star matches across repos", func(t *testing.T) {
		got, err := m.FindSkillsByPatterns([]string{"magic*"}, nil)
		if err != nil {
			t.Fatalf("FindSkillsByPatterns: %v", err)
		}
		// Sorted by (Repository, ID): community first, then main alphabetical by ID.
		want := []string{"Magic Pack", "Magic Helper", "Magic Skill"}
		if !reflect.DeepEqual(names(got), want) {
			t.Errorf("got %v, want %v", names(got), want)
		}
	})

	t.Run("question mark matches single char", func(t *testing.T) {
		got, err := m.FindSkillsByPatterns([]string{"magic-community/magic-?ack"}, nil)
		if err != nil {
			t.Fatalf("FindSkillsByPatterns: %v", err)
		}
		want := []string{"Magic Pack"}
		if !reflect.DeepEqual(names(got), want) {
			t.Errorf("got %v, want %v", names(got), want)
		}
	})

	t.Run("double star matches all", func(t *testing.T) {
		got, err := m.FindSkillsByPatterns([]string{"**"}, nil)
		if err != nil {
			t.Fatalf("FindSkillsByPatterns: %v", err)
		}
		if len(got) != 5 {
			t.Errorf("expected 5 skills across both repos, got %d (%v)", len(got), names(got))
		}
	})

	t.Run("no match returns empty", func(t *testing.T) {
		got, err := m.FindSkillsByPatterns([]string{"no-such-prefix*"}, nil)
		if err != nil {
			t.Fatalf("FindSkillsByPatterns: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty, got %v", names(got))
		}
	})

	t.Run("multiple patterns union their results, deduped", func(t *testing.T) {
		got, err := m.FindSkillsByPatterns([]string{"magic*", "git/git-helper"}, nil)
		if err != nil {
			t.Fatalf("FindSkillsByPatterns: %v", err)
		}
		// Sorted by (Repository, ID): community/git/* then community/magic-*,
		// then main/* alphabetical.
		want := []string{"Git Helper", "Magic Pack", "Magic Helper", "Magic Skill"}
		if !reflect.DeepEqual(names(got), want) {
			t.Errorf("got %v, want %v", names(got), want)
		}
	})

	t.Run("repo filter restricts to one repository", func(t *testing.T) {
		got, err := m.FindSkillsByPatterns([]string{"magic*"}, []string{"community"})
		if err != nil {
			t.Fatalf("FindSkillsByPatterns: %v", err)
		}
		want := []string{"Magic Pack"}
		if !reflect.DeepEqual(names(got), want) {
			t.Errorf("got %v, want %v", names(got), want)
		}
	})

	t.Run("lone star is rejected", func(t *testing.T) {
		_, err := m.FindSkillsByPatterns([]string{"*"}, nil)
		if err == nil {
			t.Fatal("expected error for lone '*', got nil")
		}
	})

	t.Run("empty pattern list is rejected", func(t *testing.T) {
		_, err := m.FindSkillsByPatterns(nil, nil)
		if err == nil {
			t.Fatal("expected error for empty pattern list, got nil")
		}
	})
}
