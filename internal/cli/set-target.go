package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"skill-hub/internal/state"
	"skill-hub/pkg/spec"
)

var setTargetCmd = &cobra.Command{
	Use:   "set-target [cursor|claude_code|open_code]",
	Short: "设置当前项目的首选目标",
	Long: `设置当前项目的首选目标（Cursor、Claude Code 或 OpenCode）。

此命令会更新项目状态，使后续的 apply、feedback 等命令自动使用指定的目标适配器。

示例:
  skill-hub set-target cursor      # 设置为 Cursor
  skill-hub set-target claude_code # 设置为 Claude Code
  skill-hub set-target open_code   # 设置为 OpenCode
  skill-hub set-target ""          # 清除目标设置
  
注意: 也接受简写形式 claude 和 opencode`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSetTarget(args[0])
	},
}

func init() {
	rootCmd.AddCommand(setTargetCmd)
}

func runSetTarget(target string) error {
	// 获取当前目录
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %w", err)
	}

	// 验证目标值（先规范化）
	normalizedTarget := spec.NormalizeTarget(target)
	if normalizedTarget != spec.TargetCursor && normalizedTarget != spec.TargetClaudeCode && normalizedTarget != spec.TargetOpenCode && normalizedTarget != "" {
		return fmt.Errorf("无效的目标值: %s，可用选项: %s, %s, %s (也接受简写 claude 和 opencode)", target, spec.TargetCursor, spec.TargetClaudeCode, spec.TargetOpenCode)
	}

	// 创建状态管理器
	stateManager, err := state.NewStateManager()
	if err != nil {
		return err
	}

	// 设置首选目标（使用规范化后的值）
	if err := stateManager.SetPreferredTarget(cwd, normalizedTarget); err != nil {
		return fmt.Errorf("设置首选目标失败: %w", err)
	}

	// 显示结果
	if normalizedTarget == "" {
		fmt.Printf("✅ 已清除项目 '%s' 的首选目标\n", filepath.Base(cwd))
	} else {
		fmt.Printf("✅ 已将项目 '%s' 的首选目标设置为: %s\n", filepath.Base(cwd), normalizedTarget)
		fmt.Println("下次执行 'skill-hub apply' 时将自动使用此目标")
	}

	return nil
}
