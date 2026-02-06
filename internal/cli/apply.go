package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"skill-hub/internal/adapter"
	"skill-hub/internal/adapter/claude"
	"skill-hub/internal/engine"
	"skill-hub/internal/state"
	"skill-hub/pkg/spec"
)

var (
	dryRun bool
	target string
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "将已启用的技能应用到当前项目",
	Long: `将当前项目已启用的技能分发到目标工具配置文件。

使用 --dry-run 参数可以预览变更而不实际修改文件。
使用 --target 参数指定目标工具 (cursor/claude/all)。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runApply()
	},
}

func init() {
	applyCmd.Flags().BoolVar(&dryRun, "dry-run", false, "预览变更而不实际修改文件")
	applyCmd.Flags().StringVar(&target, "target", "all", "目标工具: cursor, claude, all")
}

func runApply() error {
	fmt.Println("正在应用技能到当前项目...")

	// 获取当前目录
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %w", err)
	}

	fmt.Printf("当前项目: %s\n", cwd)
	fmt.Printf("目标工具: %s\n", target)

	// 加载项目状态
	stateManager, err := state.NewStateManager()
	if err != nil {
		return err
	}

	skills, err := stateManager.GetProjectSkills(cwd)
	if err != nil {
		return err
	}

	if len(skills) == 0 {
		fmt.Println("ℹ️  当前项目未启用任何技能")
		fmt.Println("使用 'skill-hub use <skill-id>' 启用技能")
		return nil
	}

	// 加载技能管理器
	skillManager, err := engine.NewSkillManager()
	if err != nil {
		return err
	}

	// 根据目标选择适配器
	var adapters []adapter.Adapter

	if target == "all" || target == "cursor" {
		adapters = append(adapters, adapter.NewCursorAdapter())
	}

	if target == "all" || target == "claude" {
		adapters = append(adapters, claude.NewClaudeAdapter())
	}

	if len(adapters) == 0 {
		return fmt.Errorf("无效的目标工具: %s，可用选项: cursor, claude, all", target)
	}

	// 应用每个技能到每个适配器
	totalApplied := 0

	for _, adapter := range adapters {
		adapterName := getAdapterName(adapter)
		fmt.Printf("\n=== 处理 %s 适配器 ===\n", adapterName)

		adapterApplied := 0
		for skillID, skillVars := range skills {
			fmt.Printf("\n处理技能: %s\n", skillID)

			// 加载技能详情
			skill, err := skillManager.LoadSkill(skillID)
			if err != nil {
				fmt.Printf("⚠️  跳过技能 %s: %v\n", skillID, err)
				continue
			}

			// 检查适配器支持
			if !adapterSupportsSkill(adapter, skill) {
				fmt.Printf("ℹ️  技能 %s 不支持 %s，跳过\n", skillID, adapterName)
				continue
			}

			// 获取提示词内容
			prompt, err := skillManager.GetSkillPrompt(skillID)
			if err != nil {
				fmt.Printf("⚠️  跳过技能 %s: %v\n", skillID, err)
				continue
			}

			if dryRun {
				fmt.Printf("🔍 DRY RUN - 将应用技能 %s 到 %s\n", skillID, adapterName)
				fmt.Printf("变量: %v\n", skillVars.Variables)
				adapterApplied++
				continue
			}

			// 实际应用技能
			if err := adapter.Apply(skillID, prompt, skillVars.Variables); err != nil {
				fmt.Printf("❌ 应用技能 %s 到 %s 失败: %v\n", skillID, adapterName, err)
				continue
			}

			fmt.Printf("✓ 成功应用技能 %s 到 %s\n", skillID, adapterName)
			adapterApplied++
		}

		if adapterApplied > 0 {
			fmt.Printf("\n✅ %s: 成功应用 %d 个技能\n", adapterName, adapterApplied)
			totalApplied += adapterApplied
		} else {
			fmt.Printf("\nℹ️  %s: 没有技能被应用\n", adapterName)
		}
	}

	if dryRun {
		fmt.Printf("\n🔍 DRY RUN 完成 - 将应用 %d 个技能\n", totalApplied)
		fmt.Println("使用 'skill-hub apply' 实际应用变更")
		return nil
	}

	if totalApplied == 0 {
		fmt.Println("\nℹ️  没有技能被应用到任何工具")
		return nil
	}

	fmt.Printf("\n🎉 总计成功应用 %d 个技能\n", totalApplied)
	fmt.Println("使用 'skill-hub status' 检查技能状态")

	return nil
}

// getAdapterName 获取适配器名称
func getAdapterName(adpt adapter.Adapter) string {
	// 使用类型断言
	if _, ok := adpt.(*adapter.CursorAdapter); ok {
		return "Cursor"
	}
	if _, ok := adpt.(*claude.ClaudeAdapter); ok {
		return "Claude"
	}
	return "Unknown"
}

// adapterSupportsSkill 检查适配器是否支持该技能
func adapterSupportsSkill(adpt adapter.Adapter, skill *spec.Skill) bool {
	// 使用类型断言
	if _, ok := adpt.(*adapter.CursorAdapter); ok {
		return skill.Compatibility.Cursor
	}
	if _, ok := adpt.(*claude.ClaudeAdapter); ok {
		return skill.Compatibility.ClaudeCode
	}
	return false
}
