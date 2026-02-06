package cli

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"os"
	"strings"

	"skill-hub/internal/adapter"
	"skill-hub/internal/adapter/claude"
	"skill-hub/internal/adapter/cursor"
	"skill-hub/internal/adapter/opencode"
	"skill-hub/internal/engine"
	"skill-hub/internal/state"
	"skill-hub/pkg/spec"

	"github.com/spf13/cobra"
)

var (
	removeTarget string
	forceRemove  bool
)

var removeCmd = &cobra.Command{
	Use:   "remove [skill-id]",
	Short: "从当前项目中移除技能",
	Long: `从当前项目中移除指定的技能。

移除操作会：
1. 从状态文件中删除技能记录
2. 从目标工具配置文件中物理清理技能内容
3. 如果检测到本地修改，会提示警告

使用 --target 参数指定目标工具 (cursor/claude_code/open_code/all)。
使用 --force 参数跳过安全检查。`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRemove(args[0])
	},
}

func init() {
	removeCmd.Flags().StringVar(&removeTarget, "target", "", "目标工具: cursor, claude_code, open_code, all (为空时使用状态绑定的目标)")
	removeCmd.Flags().BoolVar(&forceRemove, "force", false, "跳过安全检查，强制移除")
}

func runRemove(skillID string) error {
	fmt.Printf("正在从当前项目移除技能: %s\n", skillID)

	// 获取当前目录
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %w", err)
	}

	// 创建状态管理器
	stateMgr, err := state.NewStateManager()
	if err != nil {
		return err
	}

	// 检查技能是否在项目中启用（仅用于信息提示）
	hasSkill, err := stateMgr.ProjectHasSkill(cwd, skillID)
	if err != nil {
		return fmt.Errorf("检查技能状态失败: %w", err)
	}
	if !hasSkill {
		fmt.Printf("ℹ️  技能 %s 未在当前项目中启用，仅清理目标工具中的残留文件\n", skillID)
	}

	// 获取项目状态以确定目标
	projectState, err := stateMgr.FindProjectByPath(cwd)
	if err != nil {
		return fmt.Errorf("查找项目状态失败: %w", err)
	}

	// 确定目标工具
	resolvedTarget := removeTarget
	if resolvedTarget == "" && projectState != nil {
		resolvedTarget = spec.NormalizeTarget(projectState.PreferredTarget)
		fmt.Printf("🔍 使用状态绑定的目标: %s\n", resolvedTarget)
	}

	// 如果没有指定目标且项目未绑定目标，需要用户指定
	if resolvedTarget == "" {
		fmt.Println("❌ 当前项目未关联目标工具")
		fmt.Println("请使用 --target 参数指定目标工具:")
		fmt.Printf("  skill-hub remove %s --target cursor\n", skillID)
		fmt.Printf("  skill-hub remove %s --target claude_code\n", skillID)
		fmt.Printf("  skill-hub remove %s --target open_code\n", skillID)
		fmt.Printf("  skill-hub remove %s --target all\n", skillID)
		return nil
	}

	fmt.Printf("当前项目: %s\n", cwd)
	fmt.Printf("目标工具: %s\n", resolvedTarget)

	// 加载技能管理器
	skillManager, err := engine.NewSkillManager()
	if err != nil {
		return err
	}

	// 加载技能详情
	skill, err := skillManager.LoadSkill(skillID)
	if err != nil {
		return fmt.Errorf("加载技能失败: %w", err)
	}

	// 根据目标选择适配器
	adapters := selectAdapters(resolvedTarget, "project")
	if len(adapters) == 0 {
		return fmt.Errorf("无效的目标工具: %s，可用选项: %s, %s, %s, %s", resolvedTarget, spec.TargetCursor, spec.TargetClaudeCode, spec.TargetOpenCode, spec.TargetAll)
	}

	// 获取项目技能变量
	projectSkills, err := stateMgr.GetProjectSkills(cwd)
	if err != nil {
		return err
	}
	skillVars, skillEnabled := projectSkills[skillID]
	fmt.Printf("[DEBUG] 技能 %s 启用状态: %v\n", skillID, skillEnabled)

	// 安全检查：检测本地修改（仅当技能已启用时）
	if !forceRemove && skillEnabled {
		hasModifications, err := checkSkillModifications(adapters, skillID, skillManager, skillVars.Variables)
		if err != nil {
			fmt.Printf("⚠️  安全检查失败: %v\n", err)
			fmt.Println("使用 --force 参数跳过安全检查")
			return nil
		}

		if hasModifications {
			if !confirmRemoval(skillID) {
				fmt.Println("❌ 操作已取消")
				return nil
			}
		}
	}

	// 执行物理清理
	fmt.Println("\n=== 执行物理清理 ===")
	removedFromAdapters := []string{}

	for _, adapter := range adapters {
		adapterName := getAdapterName(adapter)

		// 检查适配器是否支持该技能
		if !adapterSupportsSkill(adapter, skill) {
			fmt.Printf("ℹ️  技能 %s 不支持 %s，跳过清理\n", skillID, adapterName)
			continue
		}

		// 检查适配器是否支持当前模式
		if !adapter.Supports() {
			fmt.Printf("ℹ️  %s 适配器不支持当前模式，跳过清理\n", adapterName)
			continue
		}

		fmt.Printf("清理 %s 适配器...\n", adapterName)
		if err := adapter.Remove(skillID); err != nil {
			fmt.Printf("❌ 从 %s 清理技能失败: %v\n", adapterName, err)
			continue
		}

		fmt.Printf("✓ 成功从 %s 清理技能\n", adapterName)
		removedFromAdapters = append(removedFromAdapters, adapterName)
	}

	if len(removedFromAdapters) == 0 {
		fmt.Println("⚠️  技能未从任何适配器清理")
		fmt.Println("可能原因:")
		fmt.Println("  1. 技能与目标工具不兼容")
		fmt.Println("  2. 适配器不支持当前模式")
		fmt.Println("  3. 技能内容不存在于配置文件中")
	} else {
		fmt.Printf("\n✅ 技能已从以下适配器清理: %s\n", strings.Join(removedFromAdapters, ", "))
	}

	// 更新状态：从项目中移除技能（仅当技能已启用时）
	if skillEnabled {
		fmt.Println("\n=== 更新状态 ===")
		fmt.Printf("[DEBUG] 准备从状态移除技能: %s\n", skillID)
		if err := stateMgr.RemoveSkillFromProject(cwd, skillID); err != nil {
			return fmt.Errorf("更新状态失败: %w", err)
		}
		fmt.Printf("✓ 成功从项目状态移除技能 %s\n", skillID)
	} else {
		fmt.Printf("[DEBUG] 技能 %s 未启用，跳过状态更新\n", skillID)
	}

	fmt.Println("\n🎉 技能移除完成")
	fmt.Println("使用 'skill-hub status' 检查当前状态")

	return nil
}

// selectAdapters 根据目标选择适配器
func selectAdapters(target string, mode string) []adapter.Adapter {
	var adapters []adapter.Adapter

	if target == spec.TargetAll || target == spec.TargetCursor {
		cursorAdapter := cursor.NewCursorAdapter()
		if mode == "global" {
			cursorAdapter = cursorAdapter.WithGlobalMode()
		} else {
			cursorAdapter = cursorAdapter.WithProjectMode()
		}
		adapters = append(adapters, cursorAdapter)
	}

	if target == spec.TargetAll || target == spec.TargetClaudeCode {
		claudeAdapter := claude.NewClaudeAdapter()
		if mode == "global" {
			claudeAdapter = claudeAdapter.WithGlobalMode()
		} else {
			claudeAdapter = claudeAdapter.WithProjectMode()
		}
		adapters = append(adapters, claudeAdapter)
	}

	if target == spec.TargetAll || target == spec.TargetOpenCode {
		opencodeAdapter := opencode.NewOpenCodeAdapter()
		if mode == "global" {
			opencodeAdapter = opencodeAdapter.WithGlobalMode()
		} else {
			opencodeAdapter = opencodeAdapter.WithProjectMode()
		}
		adapters = append(adapters, opencodeAdapter)
	}

	return adapters
}

// checkSkillModifications 检查技能是否有本地修改
func checkSkillModifications(adapters []adapter.Adapter, skillID string, skillManager *engine.SkillManager, variables map[string]string) (bool, error) {
	fmt.Println("\n=== 安全检查 ===")

	// 获取原始技能内容
	originalPrompt, err := skillManager.GetSkillPrompt(skillID)
	if err != nil {
		return false, fmt.Errorf("获取技能原始内容失败: %w", err)
	}

	// 渲染原始内容（使用项目变量）
	renderedOriginal, err := renderTemplateForRemove(originalPrompt, variables)
	if err != nil {
		return false, fmt.Errorf("渲染技能内容失败: %w", err)
	}

	originalHash := sha256.Sum256([]byte(strings.TrimSpace(renderedOriginal)))

	hasModifications := false

	for _, adapter := range adapters {
		adapterName := getAdapterName(adapter)

		// 检查适配器是否支持
		if !adapter.Supports() {
			continue
		}

		// 从适配器提取当前内容
		currentContent, err := adapter.Extract(skillID)
		if err != nil {
			// 如果提取失败（可能技能不存在于该适配器），跳过
			continue
		}

		if currentContent == "" {
			// 技能内容不存在于该适配器
			continue
		}

		// 计算当前内容的哈希
		currentHash := sha256.Sum256([]byte(strings.TrimSpace(currentContent)))

		// 比较哈希
		if currentHash != originalHash {
			fmt.Printf("⚠️  检测到 %s 适配器中的技能 %s 有本地修改\n", adapterName, skillID)
			hasModifications = true
		} else {
			fmt.Printf("✓ %s 适配器中的技能 %s 与原始内容一致\n", adapterName, skillID)
		}
	}

	return hasModifications, nil
}

// confirmRemoval 确认是否继续移除（当有本地修改时）
func confirmRemoval(skillID string) bool {
	fmt.Printf("\n⚠️  警告: 技能 %s 有本地修改，移除将丢失这些改动\n", skillID)
	fmt.Print("是否继续移除？(y/n): ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	return input == "y" || input == "yes"
}

// renderTemplateForRemove 渲染模板内容（用于remove命令）
func renderTemplateForRemove(content string, variables map[string]string) (string, error) {
	// 简单替换变量
	result := content
	for key, value := range variables {
		placeholder := "{{." + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result, nil
}
