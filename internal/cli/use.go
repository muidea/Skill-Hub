package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	httpapibiz "github.com/muidea/skill-hub/internal/modules/blocks/httpapi/biz"
	globalservice "github.com/muidea/skill-hub/internal/modules/kernel/global/service"
	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/spec"
	"github.com/spf13/cobra"
)

var useCmd = &cobra.Command{
	Use:   "use <pattern> [pattern...]",
	Short: "使用技能",
	Long: `将技能标记为在当前项目中使用。每个 positional 参数是一个 glob pattern（基于 Go path.Match），
匹配技能 ID 字段：
  *        匹配零或多个任意字符
  ?        匹配恰好一个任意字符
  **       匹配全部

此命令仅更新 state.json 中的状态记录，不直接修改项目文件，需要通过 apply 命令进行物理分发。

如果项目工作区里首次使用技能，也会同步在 state.json 里完成项目工作区信息刷新。

行为：
  - pattern 命中 0 个技能：静默通过，不报错
  - 命中 1 个：直接启用
  - 命中多个：交互式选择（跨仓库时）`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		global, _ := cmd.Flags().GetBool("global")
		agents, _ := cmd.Flags().GetStringArray("agent")
		// Single literal ID (no wildcards) keeps the legacy exact-ID lookup so
		// `use <id>` continues to match by ID. Patterns and multi-arg go
		// through the new ID-based resolution.
		if len(args) == 1 && !hasWildcard(args[0]) {
			return runUseByID(args[0], agents, global)
		}
		if global {
			return runUseByPatterns(args, agents, true)
		}
		return runUseByPatterns(args, agents, false)
	},
}

func init() {
	useCmd.Flags().Bool("global", false, "将技能标记为本机全局使用")
	useCmd.Flags().StringArray("agent", nil, "指定全局应用的 agent，可重复使用: codex, opencode, claude")
	_ = useCmd.RegisterFlagCompletionFunc("agent", completeAgentNames)
	useCmd.ValidArgsFunction = completeSkillIDs
}

// runUseByPatterns resolves each pattern against the cross-repo skill set
// and applies the resolved skill(s). Per pattern: 0 hits is silent, 1 hit is
// used directly, N hits trigger chooseSkillCandidate. A failure on one
// pattern does not stop the remaining patterns; the last error is returned.
func runUseByPatterns(patterns []string, agents []string, isGlobal bool) error {
	matchers, err := compilePatterns(patterns)
	if err != nil {
		return err
	}

	allMatches, err := resolveSkillsByPatterns(patterns)
	if err != nil {
		return err
	}

	var lastErr error
	processed := 0
	for i, p := range patterns {
		matcher := matchers[i]
		var pMatches []spec.SkillMetadata
		for _, s := range allMatches {
			if matcher.Match(s.ID) {
				pMatches = append(pMatches, s)
			}
		}

		if len(pMatches) == 0 {
			continue
		}

		var selected spec.SkillMetadata
		if len(pMatches) == 1 {
			selected = pMatches[0]
		} else {
			selected, err = chooseSkillCandidate(pMatches)
			if err != nil {
				lastErr = err
				continue
			}
		}

		if err := runUseOneSkill(selected, isGlobal, agents); err != nil {
			fmt.Fprintf(os.Stderr, "❌ pattern '%s' 解析的技能 '%s' 启用失败: %v\n", p, selected.ID, err)
			lastErr = err
			continue
		}
		processed++
	}

	if processed == 0 && lastErr == nil {
		// 0 hits across all patterns: silent per design
		return nil
	}
	return lastErr
}

func resolveSkillsByPatterns(patterns []string) ([]spec.SkillMetadata, error) {
	if client, ok := hubClientIfAvailable(); ok {
		matches, err := client.FindSkillsByPatterns(context.Background(), patterns, nil)
		if err != nil {
			return nil, errors.Wrap(err, "通过服务按pattern查找技能失败")
		}
		return matches, nil
	}

	if err := CheckInitDependency(); err != nil {
		return nil, err
	}
	repoManager, err := newRepositoryManager()
	if err != nil {
		return nil, errors.Wrap(err, "创建多仓库管理器失败")
	}
	matches, err := repoManager.FindSkillsByPatterns(patterns, nil)
	if err != nil {
		return nil, errors.Wrap(err, "按pattern查找技能失败")
	}
	return matches, nil
}

// runUseOneSkill applies a single resolved skill (post-candidate-selection)
// to the project state. It branches between local and service mode and
// between project-local and global use.
func runUseOneSkill(selected spec.SkillMetadata, isGlobal bool, agents []string) error {
	if isGlobal {
		if client, ok := hubClientIfAvailable(); ok {
			return runUseGlobalOneViaService(client, selected, agents)
		}
		return runUseGlobalOne(selected, agents)
	}
	if client, ok := hubClientIfAvailable(); ok {
		return runUseProjectOneViaService(client, selected)
	}
	return runUseProjectOne(selected)
}

// runUseByID is the legacy single-ID entry point: it resolves a literal
// skill ID (no wildcards) to a SkillMetadata via the bridge's exact-ID
// lookup, then applies it. Used by the `use <id>` shortcut to preserve
// backward compatibility.
func runUseByID(id string, agents []string, isGlobal bool) error {
	if client, ok := hubClientIfAvailable(); ok {
		candidates, err := client.FindSkillCandidates(context.Background(), id)
		if err != nil {
			return errors.Wrap(err, "通过服务查找技能失败")
		}
		if len(candidates) == 0 {
			return errors.NewWithCodef("runUseByID", errors.ErrSkillNotFound, "技能 '%s' 不存在", id)
		}
		selected := candidates[0]
		if len(candidates) > 1 {
			selected, err = chooseSkillCandidate(candidates)
			if err != nil {
				return err
			}
		}
		return runUseOneSkill(selected, isGlobal, agents)
	}

	if err := CheckInitDependency(); err != nil {
		return err
	}
	repoManager, err := newRepositoryManager()
	if err != nil {
		return errors.Wrap(err, "创建多仓库管理器失败")
	}
	candidates, err := repoManager.FindSkill(id)
	if err != nil {
		return errors.Wrap(err, "查找技能失败")
	}
	if len(candidates) == 0 {
		return errors.NewWithCodef("runUseByID", errors.ErrSkillNotFound, "技能 '%s' 不存在", id)
	}
	selected := candidates[0]
	if len(candidates) > 1 {
		selected, err = chooseSkillCandidate(candidates)
		if err != nil {
			return err
		}
	}
	return runUseOneSkill(selected, isGlobal, agents)
}

func runUseProjectOne(selected spec.SkillMetadata) error {
	repoManager, err := newRepositoryManager()
	if err != nil {
		return errors.Wrap(err, "创建多仓库管理器失败")
	}

	fullSkill, err := repoManager.LoadSkill(selected.ID, selected.Repository)
	if err != nil {
		return errors.Wrap(err, "加载技能详情失败")
	}

	fmt.Printf("启用技能: %s (%s)\n", fullSkill.Name, selected.ID)
	fmt.Printf("来源仓库: %s\n", fullSkill.Repository)
	fmt.Printf("描述: %s\n", fullSkill.Description)

	if len(fullSkill.Tags) > 0 {
		fmt.Printf("标签: %s\n", strings.Join(fullSkill.Tags, ", "))
	}

	ctx, err := RequireInitAndWorkspace("")
	if err != nil {
		return err
	}

	hasSkill, err := ctx.StateManager.ProjectHasSkill(ctx.Cwd, selected.ID)
	if err != nil {
		return err
	}
	if hasSkill {
		if !confirmSkillReconfigure() {
			fmt.Println("❌ 取消操作")
			return nil
		}
	}

	variables, err := promptSkillVariables(fullSkill)
	if err != nil {
		return err
	}

	if err := ctx.StateManager.AddSkillToProjectWithSource(ctx.Cwd, selected.ID, fullSkill.Version, selected.Repository, variables); err != nil {
		return errors.Wrap(err, "保存项目状态失败")
	}

	fmt.Printf("\n✅ 技能 '%s' 已成功标记为使用！\n", selected.ID)
	fmt.Println("使用 'skill-hub apply' 将技能物理分发到当前项目")
	return nil
}

func runUseProjectOneViaService(client serviceUseClient, selected spec.SkillMetadata) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	fullSkill, err := client.GetSkillDetail(context.Background(), selected.ID, selected.Repository)
	if err != nil {
		return errors.Wrap(err, "通过服务加载技能详情失败")
	}

	fmt.Printf("启用技能: %s (%s)\n", fullSkill.Name, selected.ID)
	fmt.Printf("来源仓库: %s\n", fullSkill.Repository)
	fmt.Printf("描述: %s\n", fullSkill.Description)
	if len(fullSkill.Tags) > 0 {
		fmt.Printf("标签: %s\n", strings.Join(fullSkill.Tags, ", "))
	}

	if projectStatus, err := client.GetProjectStatus(context.Background(), cwd, selected.ID); err == nil && projectStatus.Item != nil && len(projectStatus.Item.Items) > 0 {
		if !confirmSkillReconfigure() {
			fmt.Println("❌ 取消操作")
			return nil
		}
	}

	variables, err := promptSkillVariables(fullSkill)
	if err != nil {
		return err
	}

	_, err = client.UseSkill(context.Background(), httpapibiz.UseSkillRequest{
		ProjectPath: cwd,
		SkillID:     selected.ID,
		Repository:  selected.Repository,
		Variables:   variables,
	})
	if err != nil {
		return errors.Wrap(err, "通过服务启用技能失败")
	}

	fmt.Printf("\n✅ 技能 '%s' 已成功标记为使用！\n", selected.ID)
	fmt.Println("使用 'skill-hub apply' 将技能物理分发到当前项目")
	return nil
}

func runUseGlobalOne(selected spec.SkillMetadata, agents []string) error {
	repoManager, err := newRepositoryManager()
	if err != nil {
		return errors.Wrap(err, "创建多仓库管理器失败")
	}

	fullSkill, err := repoManager.LoadSkill(selected.ID, selected.Repository)
	if err != nil {
		return errors.Wrap(err, "加载技能详情失败")
	}

	fmt.Printf("全局启用技能: %s (%s)\n", fullSkill.Name, selected.ID)
	fmt.Printf("来源仓库: %s\n", fullSkill.Repository)
	fmt.Printf("描述: %s\n", fullSkill.Description)
	if len(fullSkill.Tags) > 0 {
		fmt.Printf("标签: %s\n", strings.Join(fullSkill.Tags, ", "))
	}

	variables, err := promptSkillVariables(fullSkill)
	if err != nil {
		return err
	}

	result, err := globalservice.New().EnableSkill(selected.ID, selected.Repository, agents, variables)
	if err != nil {
		return errors.Wrap(err, "保存全局技能状态失败")
	}

	fmt.Printf("\n✅ 技能 '%s' 已成功标记为本机全局使用！\n", result.SkillID)
	fmt.Printf("目标 agent: %s\n", strings.Join(result.Agents, ", "))
	fmt.Println("使用 'skill-hub apply --global' 刷新本机 agent 全局 skills 目录")
	return nil
}

func runUseGlobalOneViaService(client serviceUseClient, selected spec.SkillMetadata, agents []string) error {
	fullSkill, err := client.GetSkillDetail(context.Background(), selected.ID, selected.Repository)
	if err != nil {
		return errors.Wrap(err, "通过服务加载技能详情失败")
	}

	fmt.Printf("全局启用技能: %s (%s)\n", fullSkill.Name, selected.ID)
	fmt.Printf("来源仓库: %s\n", fullSkill.Repository)
	fmt.Printf("描述: %s\n", fullSkill.Description)
	if len(fullSkill.Tags) > 0 {
		fmt.Printf("标签: %s\n", strings.Join(fullSkill.Tags, ", "))
	}

	variables, err := promptSkillVariables(fullSkill)
	if err != nil {
		return err
	}

	resp, err := client.UseGlobalSkill(context.Background(), httpapibiz.UseGlobalSkillRequest{
		SkillID:    selected.ID,
		Repository: selected.Repository,
		Agents:     agents,
		Variables:  variables,
	})
	if err != nil {
		return errors.Wrap(err, "通过服务启用全局技能失败")
	}

	skillID := selected.ID
	enabledAgents := agents
	if resp.Item != nil {
		skillID = resp.Item.SkillID
		enabledAgents = resp.Item.Agents
	}

	fmt.Printf("\n✅ 技能 '%s' 已成功标记为本机全局使用！\n", skillID)
	fmt.Printf("目标 agent: %s\n", strings.Join(enabledAgents, ", "))
	fmt.Println("使用 'skill-hub apply --global' 刷新本机 agent 全局 skills 目录")
	return nil
}

// compilePatterns is defined in list.go (same package).
var _ = compilePatterns

type serviceUseClient interface {
	FindSkillCandidates(ctx context.Context, skillID string) ([]spec.SkillMetadata, error)
	FindSkillsByPatterns(ctx context.Context, patterns, repoNames []string) ([]spec.SkillMetadata, error)
	GetSkillDetail(ctx context.Context, skillID, repoName string) (*spec.Skill, error)
	GetProjectStatus(ctx context.Context, projectPath, skillID string) (*httpapibiz.ProjectStatusData, error)
	UseSkill(ctx context.Context, req httpapibiz.UseSkillRequest) (*httpapibiz.UseSkillData, error)
	UseGlobalSkill(ctx context.Context, req httpapibiz.UseGlobalSkillRequest) (*httpapibiz.UseGlobalSkillData, error)
}

func chooseSkillCandidate(skills []spec.SkillMetadata) (spec.SkillMetadata, error) {
	if len(skills) == 1 {
		return skills[0], nil
	}

	fmt.Printf("发现 %d 个同名技能，请选择要使用的技能:\n", len(skills))
	for i, skill := range skills {
		fmt.Printf("  %d. [%s] %s - %s\n", i+1, skill.Repository, skill.Name, skill.Description)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("请选择 (输入编号): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	var choice int
	if _, err := fmt.Sscanf(input, "%d", &choice); err != nil || choice < 1 || choice > len(skills) {
		return spec.SkillMetadata{}, errors.NewWithCode("chooseSkillCandidate", errors.ErrInvalidInput, "无效的选择")
	}

	return skills[choice-1], nil
}

func promptSkillVariables(fullSkill *spec.Skill) (map[string]string, error) {
	variables := make(map[string]string)
	if len(fullSkill.Variables) == 0 {
		fmt.Println("\n该技能没有可配置的变量")
		return variables, nil
	}

	fmt.Println("\n请设置技能变量 (按Enter使用默认值):")
	reader := bufio.NewReader(os.Stdin)
	for _, variable := range fullSkill.Variables {
		defaultValue := variable.Default
		fmt.Printf("%s [%s]: ", variable.Name, defaultValue)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			variables[variable.Name] = defaultValue
		} else {
			variables[variable.Name] = input
		}
	}
	return variables, nil
}

func confirmSkillReconfigure() bool {
	fmt.Println("⚠️  该技能已在当前项目启用")
	fmt.Print("是否重新配置变量？ [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(response)
	return response == "y" || response == "Y"
}
