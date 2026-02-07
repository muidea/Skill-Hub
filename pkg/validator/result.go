package validator

import (
	"fmt"
	"path/filepath"
)

// ValidationResult 表示校验结果
type ValidationResult struct {
	IsValid        bool                   // 是否通过所有校验
	Errors         []ValidationError      // 错误列表
	Warnings       []ValidationWarning    // 警告列表
	SkillName      string                 // 技能名称
	FilePath       string                 // 文件路径
	DirName        string                 // 目录名
	HasFrontmatter bool                   // 是否有frontmatter
	Frontmatter    map[string]interface{} // frontmatter内容
}

// NewValidationResult 创建新的校验结果
func NewValidationResult(filePath string) *ValidationResult {
	return &ValidationResult{
		FilePath:       filePath,
		DirName:        filepath.Base(filepath.Dir(filePath)),
		Errors:         []ValidationError{},
		Warnings:       []ValidationWarning{},
		IsValid:        true,
		HasFrontmatter: false,
		Frontmatter:    make(map[string]interface{}),
	}
}

// AddError 添加错误
func (r *ValidationResult) AddError(err ValidationError) {
	r.Errors = append(r.Errors, err)
	r.IsValid = false
}

// AddWarning 添加警告
func (r *ValidationResult) AddWarning(warn ValidationWarning) {
	r.Warnings = append(r.Warnings, warn)
}

// HasErrors 检查是否有错误
func (r *ValidationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// HasWarnings 检查是否有警告
func (r *ValidationResult) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// GetFixableErrors 获取可修复的错误
func (r *ValidationResult) GetFixableErrors() []ValidationError {
	var fixable []ValidationError
	for _, err := range r.Errors {
		if err.Fixable {
			fixable = append(fixable, err)
		}
	}
	return fixable
}

// GetFixableWarnings 获取可修复的警告
func (r *ValidationResult) GetFixableWarnings() []ValidationWarning {
	var fixable []ValidationWarning
	for _, warn := range r.Warnings {
		if warn.Fixable {
			fixable = append(fixable, warn)
		}
	}
	return fixable
}

// Summary 返回校验结果摘要
func (r *ValidationResult) Summary() string {
	if r.IsValid && !r.HasWarnings() {
		return "✅ 通过所有检查"
	}

	var summary string
	if r.HasErrors() {
		summary += fmt.Sprintf("❌ %d个错误", len(r.Errors))
	}
	if r.HasWarnings() {
		if summary != "" {
			summary += ", "
		}
		summary += fmt.Sprintf("⚠️  %d个警告", len(r.Warnings))
	}
	return summary
}

// Print 打印校验结果
func (r *ValidationResult) Print() {
	fmt.Printf("\n=== 分析: %s ===\n", filepath.Base(filepath.Dir(r.FilePath)))
	fmt.Printf("文件: %s\n", r.FilePath)
	fmt.Printf("目录名: %s\n", r.DirName)

	if len(r.Frontmatter) > 0 {
		fmt.Println("\nFrontmatter字段:")
		for key, value := range r.Frontmatter {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	if r.HasErrors() {
		fmt.Println("\n❌ 错误:")
		for _, err := range r.Errors {
			fmt.Printf("  - [%s] %s\n", err.Code, err.Message)
		}
	}

	if r.HasWarnings() {
		fmt.Println("\n⚠️  警告:")
		for _, warn := range r.Warnings {
			fmt.Printf("  - [%s] %s\n", warn.Code, warn.Message)
		}
	}

	if r.IsValid && !r.HasWarnings() {
		fmt.Println("\n✅ 通过所有检查")
	}
}

// Merge 合并多个校验结果
func (r *ValidationResult) Merge(other *ValidationResult) {
	r.Errors = append(r.Errors, other.Errors...)
	r.Warnings = append(r.Warnings, other.Warnings...)
	if !other.IsValid {
		r.IsValid = false
	}
}
