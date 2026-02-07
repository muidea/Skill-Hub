package main

import (
	"fmt"
	"os"
	"path/filepath"

	"skill-hub/pkg/validator"
)

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

	// 创建校验器
	validator := validator.NewValidator()

	// 分析每个文件
	errorCount := 0
	warningCount := 0

	for _, skillFile := range skillFiles {
		result, err := validator.ValidateFile(skillFile)
		if err != nil {
			fmt.Printf("❌ 分析失败 %s: %v\n", skillFile, err)
			continue
		}

		result.Print()

		errorCount += len(result.Errors)
		warningCount += len(result.Warnings)
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
