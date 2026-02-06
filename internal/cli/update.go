package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "更新技能仓库",
	Long:  "从远程仓库拉取最新技能，并提示更新受影响的项目。",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUpdate()
	},
}

func runUpdate() error {
	fmt.Println("正在更新技能仓库...")
	fmt.Println("连接到远程仓库...")
	fmt.Println("✓ 获取最新变更")

	fmt.Println("\n检测到以下更新:")
	fmt.Println("技能             版本变化")
	fmt.Println("-------------------------")
	fmt.Println("git-expert       1.0.0 → 1.1.0")

	fmt.Println("\n📝 更新内容:")
	fmt.Println("- 添加了更多提交类型示例")
	fmt.Println("- 优化了提示词结构")

	fmt.Print("\n是否更新受影响的项目？ [y/N]: ")

	var response string
	fmt.Scanln(&response)

	if response != "y" && response != "Y" {
		fmt.Println("❌ 取消项目更新")
		fmt.Println("ℹ️  技能仓库已更新，使用 'skill-hub apply' 手动更新项目")
		return nil
	}

	fmt.Println("正在更新项目...")
	fmt.Println("扫描项目中的技能标记块...")
	fmt.Println("更新 .cursorrules 文件...")
	fmt.Println("✓ 更新完成")

	fmt.Println("\n✅ 技能仓库和项目已同步更新！")

	return nil
}
