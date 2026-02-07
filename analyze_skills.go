package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type SkillAnalysis struct {
	FilePath       string
	DirName        string
	HasFrontmatter bool
	Frontmatter    map[string]interface{}
	Errors         []string
	Warnings       []string
}

func analyzeSkillFile(skillPath string) (*SkillAnalysis, error) {
	analysis := &SkillAnalysis{
		FilePath: skillPath,
		DirName:  filepath.Base(filepath.Dir(skillPath)),
		Errors:   []string{},
		Warnings: []string{},
	}

	// 读取文件内容
	content, err := ioutil.ReadFile(skillPath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	// 检查是否有frontmatter
	lines := strings.Split(string(content), "\n")
	if len(lines) < 2 || lines[0] != "---" {
		analysis.Errors = append(analysis.Errors, "缺少YAML frontmatter（必须以---开头）")
		return analysis, nil
	}

	analysis.HasFrontmatter = true

	// 提取frontmatter
	var frontmatterLines []string
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			break
		}
		frontmatterLines = append(frontmatterLines, lines[i])
	}

	if len(frontmatterLines) == 0 {
		analysis.Errors = append(analysis.Errors, "frontmatter为空")
		return analysis, nil
	}

	// 解析YAML
	frontmatterContent := strings.Join(frontmatterLines, "\n")
	var frontmatter map[string]interface{}
	if err := yaml.Unmarshal([]byte(frontmatterContent), &frontmatter); err != nil {
		analysis.Errors = append(analysis.Errors, fmt.Sprintf("解析YAML失败: %v", err))
		return analysis, nil
	}

	analysis.Frontmatter = frontmatter

	return analysis, nil
}

func validateSkillAnalysis(analysis *SkillAnalysis) {
	// 检查必需字段
	if name, ok := analysis.Frontmatter["name"].(string); ok {
		validateNameField(name, analysis.DirName, analysis)
	} else {
		analysis.Errors = append(analysis.Errors, "缺少必需字段: name")
	}

	if desc, ok := analysis.Frontmatter["description"].(string); ok {
		validateDescriptionField(desc, analysis)
	} else {
		analysis.Errors = append(analysis.Errors, "缺少必需字段: description")
	}

	// 检查可选字段
	if compat, ok := analysis.Frontmatter["compatibility"]; ok {
		validateCompatibilityField(compat, analysis)
	}

	if metadata, ok := analysis.Frontmatter["metadata"]; ok {
		validateMetadataField(metadata, analysis)
	}

	if license, ok := analysis.Frontmatter["license"]; ok {
		validateLicenseField(license, analysis)
	}

	if allowedTools, ok := analysis.Frontmatter["allowed-tools"]; ok {
		validateAllowedToolsField(allowedTools, analysis)
	}
}

func validateNameField(name, dirName string, analysis *SkillAnalysis) {
	// 检查长度
	if len(name) < 1 || len(name) > 64 {
		analysis.Errors = append(analysis.Errors, fmt.Sprintf("name长度无效: %d字符 (必须1-64字符)", len(name)))
	}

	// 检查命名规范: ^[a-z0-9]+(-[a-z0-9]+)*$
	namePattern := `^[a-z0-9]+(-[a-z0-9]+)*$`
	matched, _ := regexp.MatchString(namePattern, name)
	if !matched {
		analysis.Errors = append(analysis.Errors,
			fmt.Sprintf("name不符合规范: '%s' (必须小写字母数字，用连字符分隔)", name))
	}

	// 检查不能以连字符开头或结尾
	if strings.HasPrefix(name, "-") {
		analysis.Errors = append(analysis.Errors, "name不能以连字符开头")
	}
	if strings.HasSuffix(name, "-") {
		analysis.Errors = append(analysis.Errors, "name不能以连字符结尾")
	}

	// 检查不能有连续连字符
	if strings.Contains(name, "--") {
		analysis.Errors = append(analysis.Errors, "name不能有连续连字符")
	}

	// 检查目录名是否匹配
	if name != dirName {
		analysis.Warnings = append(analysis.Warnings,
			fmt.Sprintf("name字段('%s')与目录名('%s')不匹配", name, dirName))
	}
}

func validateDescriptionField(desc string, analysis *SkillAnalysis) {
	// 检查长度
	if len(desc) < 1 || len(desc) > 1024 {
		analysis.Errors = append(analysis.Errors,
			fmt.Sprintf("description长度无效: %d字符 (必须1-1024字符)", len(desc)))
	}

	// 检查内容质量（启发式检查）
	if len(desc) < 20 {
		analysis.Warnings = append(analysis.Warnings, "description可能太短，建议提供更详细的描述")
	}

	if strings.Count(desc, ".") < 1 {
		analysis.Warnings = append(analysis.Warnings, "description应该包含完整的句子")
	}
}

func validateCompatibilityField(compat interface{}, analysis *SkillAnalysis) {
	// 根据规范，compatibility应该是字符串，最大500字符
	switch v := compat.(type) {
	case string:
		if len(v) > 500 {
			analysis.Errors = append(analysis.Errors,
				fmt.Sprintf("compatibility太长: %d字符 (最大500字符)", len(v)))
		}
	case map[string]interface{}:
		// 当前实现使用对象格式，但规范要求字符串
		analysis.Warnings = append(analysis.Warnings,
			"compatibility应该是字符串格式，而不是对象（当前实现可能不符合规范）")
	default:
		analysis.Warnings = append(analysis.Warnings,
			fmt.Sprintf("compatibility字段类型未知: %T", v))
	}
}

func validateMetadataField(metadata interface{}, analysis *SkillAnalysis) {
	// metadata应该是键值对
	switch v := metadata.(type) {
	case map[string]interface{}:
		// 检查键值类型
		for key, value := range v {
			switch val := value.(type) {
			case string:
				// 字符串值，符合规范
			default:
				analysis.Warnings = append(analysis.Warnings,
					fmt.Sprintf("metadata.%s的值类型可能不符合规范: %T", key, val))
			}
		}
	default:
		analysis.Warnings = append(analysis.Warnings,
			fmt.Sprintf("metadata字段类型可能不符合规范: %T", v))
	}
}

func validateLicenseField(license interface{}, analysis *SkillAnalysis) {
	// license应该是字符串
	switch v := license.(type) {
	case string:
		if len(v) > 200 {
			analysis.Warnings = append(analysis.Warnings,
				"license字段建议保持简短")
		}
	default:
		analysis.Warnings = append(analysis.Warnings,
			fmt.Sprintf("license字段类型可能不符合规范: %T", v))
	}
}

func validateAllowedToolsField(allowedTools interface{}, analysis *SkillAnalysis) {
	// allowed-tools应该是字符串（空格分隔的列表）
	switch v := allowedTools.(type) {
	case string:
		// 符合规范
	default:
		analysis.Warnings = append(analysis.Warnings,
			fmt.Sprintf("allowed-tools字段类型可能不符合规范: %T", v))
	}
}

func printAnalysis(analysis *SkillAnalysis) {
	fmt.Printf("\n=== 分析: %s ===\n", filepath.Base(filepath.Dir(analysis.FilePath)))
	fmt.Printf("文件: %s\n", analysis.FilePath)
	fmt.Printf("目录名: %s\n", analysis.DirName)

	if len(analysis.Frontmatter) > 0 {
		fmt.Println("\nFrontmatter字段:")
		for key, value := range analysis.Frontmatter {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	if len(analysis.Errors) > 0 {
		fmt.Println("\n❌ 错误:")
		for _, err := range analysis.Errors {
			fmt.Printf("  - %s\n", err)
		}
	}

	if len(analysis.Warnings) > 0 {
		fmt.Println("\n⚠️  警告:")
		for _, warn := range analysis.Warnings {
			fmt.Printf("  - %s\n", warn)
		}
	}

	if len(analysis.Errors) == 0 && len(analysis.Warnings) == 0 {
		fmt.Println("\n✅ 通过所有检查")
	}
}

func main() {
	fmt.Println("开始分析Skill Hub中的技能文件...")

	// 查找所有SKILL.md文件
	skillFiles := []string{}

	// 检查.agents/skills目录
	agentsDir := ".agents/skills"
	if _, err := os.Stat(agentsDir); err == nil {
		filepath.Walk(agentsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() && info.Name() == "SKILL.md" {
				skillFiles = append(skillFiles, path)
			}
			return nil
		})
	}

	// 检查examples目录
	examplesDir := "examples"
	if _, err := os.Stat(examplesDir); err == nil {
		filepath.Walk(examplesDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() && info.Name() == "SKILL.md" {
				skillFiles = append(skillFiles, path)
			}
			return nil
		})
	}

	if len(skillFiles) == 0 {
		fmt.Println("未找到SKILL.md文件")
		return
	}

	fmt.Printf("找到 %d 个技能文件:\n", len(skillFiles))

	// 分析每个文件
	errorCount := 0
	warningCount := 0

	for _, skillFile := range skillFiles {
		analysis, err := analyzeSkillFile(skillFile)
		if err != nil {
			fmt.Printf("❌ 分析失败 %s: %v\n", skillFile, err)
			continue
		}

		if analysis.HasFrontmatter {
			validateSkillAnalysis(analysis)
		}

		printAnalysis(analysis)

		errorCount += len(analysis.Errors)
		warningCount += len(analysis.Warnings)
	}

	// 总结
	fmt.Printf("\n=== 分析总结 ===\n")
	fmt.Printf("分析文件数: %d\n", len(skillFiles))
	fmt.Printf("总错误数: %d\n", errorCount)
	fmt.Printf("总警告数: %d\n", warningCount)

	if errorCount > 0 {
		fmt.Println("\n❌ 发现规范不符合项，需要修复")
		os.Exit(1)
	} else if warningCount > 0 {
		fmt.Println("\n⚠️  发现警告项，建议检查")
	} else {
		fmt.Println("\n✅ 所有技能文件符合规范")
	}
}
