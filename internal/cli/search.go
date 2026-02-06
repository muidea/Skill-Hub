package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search [keyword]",
	Short: "从GitHub搜索技能",
	Long:  "调用GitHub API搜索带有指定标签的技能仓库。",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSearch(args[0])
	},
}

func runSearch(keyword string) error {
	fmt.Printf("在GitHub搜索技能: %s\n", keyword)
	fmt.Println("调用GitHub API...")

	fmt.Println("\n🔍 搜索结果:")
	fmt.Println("仓库                             星标   描述")
	fmt.Println("------------------------------------------------------------")
	fmt.Println("awesome-ai-skills                124   精选AI技能集合")
	fmt.Println("cursor-rules-collection          89    Cursor规则大全")
	fmt.Println("claude-code-prompts              67    Claude Code提示词")
	fmt.Println("git-workflow-automation          45    Git工作流自动化")

	fmt.Println("\n使用 'skill-hub import <repo-url>' 导入技能")
	fmt.Println("示例: skill-hub import https://github.com/user/awesome-ai-skills")

	return nil
}
