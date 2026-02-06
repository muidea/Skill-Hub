package engine

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"skill-hub/internal/config"
	"skill-hub/pkg/spec"
)

// SkillManager 管理技能加载和操作
type SkillManager struct {
	skillsDir string
}

// NewSkillManager 创建新的技能管理器
func NewSkillManager() (*SkillManager, error) {
	skillsDir, err := config.GetSkillsDir()
	if err != nil {
		return nil, err
	}
	return &SkillManager{skillsDir: skillsDir}, nil
}

// LoadSkill 加载指定ID的技能
func (m *SkillManager) LoadSkill(skillID string) (*spec.Skill, error) {
	skillDir := filepath.Join(m.skillsDir, skillID)

	// 检查技能目录是否存在
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("技能 '%s' 不存在", skillID)
	}

	// 加载skill.yaml
	yamlPath := filepath.Join(skillDir, "skill.yaml")
	yamlData, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("读取skill.yaml失败: %w", err)
	}

	var skill spec.Skill
	if err := yaml.Unmarshal(yamlData, &skill); err != nil {
		return nil, fmt.Errorf("解析skill.yaml失败: %w", err)
	}

	// 验证必需字段
	if skill.ID == "" {
		return nil, fmt.Errorf("skill.yaml缺少id字段")
	}
	if skill.Name == "" {
		return nil, fmt.Errorf("skill.yaml缺少name字段")
	}
	if skill.Version == "" {
		return nil, fmt.Errorf("skill.yaml缺少version字段")
	}

	// 确保ID与目录名一致
	if skill.ID != skillID {
		return nil, fmt.Errorf("技能ID不匹配: 目录名为%s, skill.yaml中为%s", skillID, skill.ID)
	}

	return &skill, nil
}

// LoadAllSkills 加载所有技能
func (m *SkillManager) LoadAllSkills() ([]*spec.Skill, error) {
	entries, err := os.ReadDir(m.skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*spec.Skill{}, nil
		}
		return nil, fmt.Errorf("读取技能目录失败: %w", err)
	}

	var skills []*spec.Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillID := entry.Name()
		skill, err := m.LoadSkill(skillID)
		if err != nil {
			fmt.Printf("警告: 跳过技能 %s: %v\n", skillID, err)
			continue
		}

		skills = append(skills, skill)
	}

	return skills, nil
}

// GetSkillPrompt 获取技能的提示词内容
func (m *SkillManager) GetSkillPrompt(skillID string) (string, error) {
	skillDir := filepath.Join(m.skillsDir, skillID)
	promptPath := filepath.Join(skillDir, "prompt.md")

	// 检查prompt.md文件是否存在
	if _, err := os.Stat(promptPath); os.IsNotExist(err) {
		return "", fmt.Errorf("技能 '%s' 缺少prompt.md文件", skillID)
	}

	promptData, err := os.ReadFile(promptPath)
	if err != nil {
		return "", fmt.Errorf("读取prompt.md失败: %w", err)
	}

	return string(promptData), nil
}

// SkillExists 检查技能是否存在
func (m *SkillManager) SkillExists(skillID string) bool {
	skillDir := filepath.Join(m.skillsDir, skillID)

	// 检查目录是否存在
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return false
	}

	// 检查skill.yaml是否存在
	yamlPath := filepath.Join(skillDir, "skill.yaml")
	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		return false
	}

	// 检查prompt.md是否存在
	promptPath := filepath.Join(skillDir, "prompt.md")
	if _, err := os.Stat(promptPath); os.IsNotExist(err) {
		return false
	}

	return true
}

// GetSkillsDir 获取技能目录路径（包级函数）
func GetSkillsDir() (string, error) {
	manager, err := NewSkillManager()
	if err != nil {
		return "", err
	}
	return manager.skillsDir, nil
}
