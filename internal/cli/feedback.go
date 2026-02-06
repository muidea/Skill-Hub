package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"skill-hub/internal/adapter"
	"skill-hub/internal/engine"
	"skill-hub/internal/state"
)

var feedbackCmd = &cobra.Command{
	Use:   "feedback [skill-id]",
	Short: "将项目内的手动修改反馈回技能仓库",
	Long:  "将项目配置文件中手动修改的技能内容反向更新到本地技能仓库。",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFeedback(args[0])
	},
}

func runFeedback(skillID string) error {
	fmt.Printf("收集技能 '%s' 的反馈...\n", skillID)

	// 获取当前目录
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %w", err)
	}

	// 检查项目是否启用了该技能
	stateManager, err := state.NewStateManager()
	if err != nil {
		return err
	}

	hasSkill, err := stateManager.ProjectHasSkill(cwd, skillID)
	if err != nil {
		return err
	}

	if !hasSkill {
		return fmt.Errorf("技能 '%s' 未在当前项目启用", skillID)
	}

	// 加载技能管理器
	skillManager, err := engine.NewSkillManager()
	if err != nil {
		return err
	}

	// 检查技能是否存在
	if !skillManager.SkillExists(skillID) {
		return fmt.Errorf("技能 '%s' 不存在", skillID)
	}

	// 创建Cursor适配器
	cursorAdapter := adapter.NewCursorAdapter()

	// 从文件提取当前内容
	fileContent, err := cursorAdapter.Extract(skillID)
	if err != nil {
		return fmt.Errorf("提取技能内容失败: %w", err)
	}

	// 从仓库获取原始内容
	originalPrompt, err := skillManager.GetSkillPrompt(skillID)
	if err != nil {
		return fmt.Errorf("获取原始内容失败: %w", err)
	}

	// 获取项目变量
	skills, err := stateManager.GetProjectSkills(cwd)
	if err != nil {
		return err
	}

	skillVars, exists := skills[skillID]
	if !exists {
		return fmt.Errorf("未找到技能变量配置")
	}

	// 渲染原始内容（使用项目变量）
	renderedOriginal, err := renderTemplate(originalPrompt, skillVars.Variables)
	if err != nil {
		return fmt.Errorf("渲染原始内容失败: %w", err)
	}

	// 比较内容
	if strings.TrimSpace(fileContent) == strings.TrimSpace(renderedOriginal) {
		fmt.Println("✅ 技能内容未修改，无需反馈")
		return nil
	}

	// 显示差异
	fmt.Println("\n🔍 检测到手动修改:")
	fmt.Println("========================================")

	fileLines := strings.Split(strings.TrimSpace(fileContent), "\n")
	originalLines := strings.Split(strings.TrimSpace(renderedOriginal), "\n")

	// 简单差异显示
	maxLines := len(fileLines)
	if len(originalLines) > maxLines {
		maxLines = len(originalLines)
	}

	changesFound := false
	for i := 0; i < maxLines; i++ {
		var fileLine, originalLine string
		if i < len(fileLines) {
			fileLine = fileLines[i]
		}
		if i < len(originalLines) {
			originalLine = originalLines[i]
		}

		if fileLine != originalLine {
			if !changesFound {
				fmt.Println("行号 | 修改前                      | 修改后")
				fmt.Println("-----|---------------------------|---------------------------")
				changesFound = true
			}

			lineNum := i + 1
			fmt.Printf("%4d | %-25s | %-25s\n", lineNum,
				truncate(originalLine, 25),
				truncate(fileLine, 25))
		}
	}

	if !changesFound {
		fmt.Println("（仅空白字符差异）")
	}

	fmt.Println("========================================")

	// 确认反馈
	fmt.Print("\n是否将这些修改更新到技能仓库？ [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(response)

	if response != "y" && response != "Y" {
		fmt.Println("❌ 取消反馈操作")
		return nil
	}

	// 更新技能仓库
	fmt.Println("正在更新技能仓库...")

	// 获取技能目录
	skillsDir, err := engine.GetSkillsDir()
	if err != nil {
		return err
	}

	skillDir := fmt.Sprintf("%s/%s", skillsDir, skillID)
	promptPath := fmt.Sprintf("%s/prompt.md", skillDir)

	// 写入更新后的prompt.md
	// 注意：这里应该实现智能的变量提取，暂时直接保存文件内容
	// 在实际实现中，应该尝试从修改内容中移除项目特定变量值

	if err := os.WriteFile(promptPath, []byte(fileContent), 0644); err != nil {
		return fmt.Errorf("更新prompt.md失败: %w", err)
	}

	fmt.Println("✓ 更新 prompt.md")

	// 更新skill.yaml版本
	skill, err := skillManager.LoadSkill(skillID)
	if err != nil {
		return fmt.Errorf("加载技能失败: %w", err)
	}

	// 增加版本号
	versionParts := strings.Split(skill.Version, ".")
	if len(versionParts) == 3 {
		// 简单增加修订版本号
		// 在实际实现中应该更智能地处理版本号
		skill.Version = fmt.Sprintf("%s.%s.%d",
			versionParts[0],
			versionParts[1],
			parseInt(versionParts[2])+1)
	}

	// 保存更新后的skill.yaml
	yamlPath := fmt.Sprintf("%s/skill.yaml", skillDir)
	yamlData, err := yaml.Marshal(skill)
	if err != nil {
		return fmt.Errorf("序列化skill.yaml失败: %w", err)
	}

	if err := os.WriteFile(yamlPath, yamlData, 0644); err != nil {
		return fmt.Errorf("更新skill.yaml失败: %w", err)
	}

	fmt.Println("✓ 更新 skill.yaml")
	fmt.Printf("✓ 版本更新: %s\n", skill.Version)

	fmt.Println("\n✅ 反馈完成！")
	fmt.Println("使用 'skill-hub update' 同步到远程仓库")

	return nil
}

// truncate 截断字符串
func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}

// parseInt 解析整数，失败返回0
func parseInt(s string) int {
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}
