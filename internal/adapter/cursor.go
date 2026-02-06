package adapter

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

// CursorAdapter 实现Cursor规则的适配器
type CursorAdapter struct {
	filePath string
}

// NewCursorAdapter 创建新的Cursor适配器
func NewCursorAdapter() *CursorAdapter {
	return &CursorAdapter{
		filePath: ".cursorrules",
	}
}

// markerPattern 匹配技能标记块的正则表达式
var markerPattern = regexp.MustCompile(`(?s)# === SKILL-HUB BEGIN: (?P<id>.*?) ===\n(?P<content>.*?)\n# === SKILL-HUB END: (?P<id2>.*?) ===`)

// Apply 应用技能到.cursorrules文件
func (a *CursorAdapter) Apply(skillID string, content string, variables map[string]string) error {
	// 渲染模板
	rendered, err := a.renderTemplate(content, variables)
	if err != nil {
		return fmt.Errorf("渲染模板失败: %w", err)
	}

	// 创建标记块
	markerBlock := a.createMarkerBlock(skillID, rendered)

	// 读取现有文件内容
	existingContent, err := a.readFile()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// 替换或添加标记块
	newContent := a.replaceOrAddMarker(existingContent, skillID, markerBlock)

	// 写入文件
	return a.writeFile(newContent)
}

// Extract 从.cursorrules文件提取技能内容
func (a *CursorAdapter) Extract(skillID string) (string, error) {
	content, err := a.readFile()
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("文件不存在: %s", a.filePath)
		}
		return "", err
	}

	// 查找标记块
	matches := markerPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) >= 4 && match[1] == skillID && match[3] == skillID {
			return strings.TrimSpace(match[2]), nil
		}
	}

	return "", fmt.Errorf("未找到技能 '%s' 的标记块", skillID)
}

// Remove 从.cursorrules文件移除技能
func (a *CursorAdapter) Remove(skillID string) error {
	content, err := a.readFile()
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在，无需移除
		}
		return err
	}

	// 移除指定技能的标记块
	pattern := regexp.MustCompile(fmt.Sprintf(`(?s)# === SKILL-HUB BEGIN: %s ===\n.*?\n# === SKILL-HUB END: %s ===\n?`, regexp.QuoteMeta(skillID), regexp.QuoteMeta(skillID)))
	newContent := pattern.ReplaceAllString(content, "")

	// 如果内容为空，删除文件
	newContent = strings.TrimSpace(newContent)
	if newContent == "" {
		return os.Remove(a.filePath)
	}

	return a.writeFile(newContent)
}

// List 列出.cursorrules文件中的所有技能
func (a *CursorAdapter) List() ([]string, error) {
	content, err := a.readFile()
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var skillIDs []string
	matches := markerPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) >= 2 && match[1] == match[3] { // 确保BEGIN和END的ID匹配
			skillIDs = append(skillIDs, match[1])
		}
	}

	return skillIDs, nil
}

// Supports 检查是否支持当前环境
func (a *CursorAdapter) Supports() bool {
	// Cursor适配器总是可用
	return true
}

// renderTemplate 渲染模板内容
func (a *CursorAdapter) renderTemplate(content string, variables map[string]string) (string, error) {
	tmpl, err := template.New("skill").Parse(content)
	if err != nil {
		return "", fmt.Errorf("解析模板失败: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, variables); err != nil {
		return "", fmt.Errorf("执行模板失败: %w", err)
	}

	return buf.String(), nil
}

// createMarkerBlock 创建标记块
func (a *CursorAdapter) createMarkerBlock(skillID string, content string) string {
	return fmt.Sprintf("# === SKILL-HUB BEGIN: %s ===\n%s\n# === SKILL-HUB END: %s ===\n", skillID, content, skillID)
}

// readFile 读取文件内容
func (a *CursorAdapter) readFile() (string, error) {
	data, err := os.ReadFile(a.filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// writeFile 写入文件内容（原子操作）
func (a *CursorAdapter) writeFile(content string) error {
	// 创建备份
	if _, err := os.Stat(a.filePath); err == nil {
		backupPath := a.filePath + ".bak"
		if err := os.Rename(a.filePath, backupPath); err != nil {
			return fmt.Errorf("创建备份失败: %w", err)
		}
	}

	// 写入临时文件
	tmpPath := a.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入临时文件失败: %w", err)
	}

	// 重命名为目标文件
	if err := os.Rename(tmpPath, a.filePath); err != nil {
		return fmt.Errorf("重命名文件失败: %w", err)
	}

	return nil
}

// replaceOrAddMarker 替换或添加标记块
func (a *CursorAdapter) replaceOrAddMarker(existingContent, skillID, markerBlock string) string {
	// 尝试替换现有标记块
	pattern := regexp.MustCompile(fmt.Sprintf(`(?s)# === SKILL-HUB BEGIN: %s ===\n.*?\n# === SKILL-HUB END: %s ===`, regexp.QuoteMeta(skillID), regexp.QuoteMeta(skillID)))

	if pattern.MatchString(existingContent) {
		return pattern.ReplaceAllString(existingContent, markerBlock)
	}

	// 没有现有标记块，添加到文件末尾
	existingContent = strings.TrimSpace(existingContent)
	if existingContent == "" {
		return markerBlock
	}

	return existingContent + "\n\n" + markerBlock
}

// GetFilePath 获取适配器管理的文件路径
func (a *CursorAdapter) GetFilePath() string {
	absPath, err := filepath.Abs(a.filePath)
	if err != nil {
		return a.filePath
	}
	return absPath
}
