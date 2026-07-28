package cli

import (
	"context"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/muidea/skill-hub/pkg/errors"
	pkgutils "github.com/muidea/skill-hub/pkg/utils"
)

// rejectPositionalPattern is retained for commands that only accept patterns.
// Most pattern-aware commands now use validatePatternOrExactID so a single
// literal skill ID can be supplied directly as a positional argument.
//
// Returning a clear error here is intentional: cobra's default
// "unknown command" message would leave users wondering why their typed
// `list magic*` was rejected, when the real answer is "use --pattern".
//
// Shell-expansion trap: when the user types `list --pattern magic*` without
// quoting, the shell expands `magic*` to a list of cwd files BEFORE the
// CLI process starts. cobra then sees those filenames as positional args
// and the validator fires here. In that scenario echoing the args back as
// the suggested --pattern value is actively misleading (it tells the user
// to pass the expanded list, not their original glob). The message instead
// points at the quoting fix and gives a canonical example.
func rejectPositionalPattern(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	cmd.SilenceUsage = true
	command := cmd.CommandPath()
	if command == "" {
		command = cmd.Use
	}

	hint := "不再支持位置参数 pattern，请通过 --pattern 传入"
	switch {
	case looksLikeShellExpansion(args):
		hint += "；检测到多个位置参数，可能是未引用的 glob 被 shell 展开。请引用 pattern，例如：" + command + " --pattern 'magic*'"
	case len(args) == 1:
		hint += "。请改用：" + command + " --pattern '" + args[0] + "'"
	}
	return errors.NewWithCodef("rejectPositionalPattern", errors.ErrInvalidInput, "%s", hint)
}

// looksLikeShellExpansion returns true when the positional args look like
// the result of the shell expanding a single unquoted glob: multiple
// args that all share a leading prefix and contain no path separators or
// shell-special characters. This is a heuristic, not a guarantee, but it
// catches the common footgun (typing `list --pattern magic*` in a cwd that
// has matching files) and lets the error message point at the quoting fix
// instead of at the expanded filenames themselves.
func looksLikeShellExpansion(args []string) bool {
	if len(args) < 2 {
		return false
	}
	prefix := longestCommonPrefix(args)
	if len(prefix) < 2 {
		return false
	}
	for _, a := range args {
		if strings.ContainsAny(a, "/ \t\n") {
			return false
		}
	}
	return true
}

func longestCommonPrefix(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	p := ss[0]
	for _, s := range ss[1:] {
		for !strings.HasPrefix(s, p) && p != "" {
			p = p[:len(p)-1]
		}
		if p == "" {
			return ""
		}
	}
	return p
}

// readPatternFlag returns the cleaned value of the --pattern flag for a
// pattern-aware command. It is the single source of truth for the flag's
// validation across list/use/status/apply/remove/feedback/validate.
//
// Behaviour:
//   - flag not set or empty slice → returns (nil, nil) so the caller falls
//     through to its "no pattern" / "all enabled" default path
//   - flag set with an empty string element → returns ErrInvalidInput
//   - flag set with non-empty elements → returns them unchanged (no dedup,
//     no compile) so callers that want pattern compilation (compilePatterns)
//     still get the raw strings
//
// Returning nil for "not set" keeps the existing "no args" semantics for
// commands like `list` and `status` working without a separate code path.
func readPatternFlag(cmd *cobra.Command) ([]string, error) {
	raw, _ := cmd.Flags().GetStringArray("pattern")
	if len(raw) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		if p == "" {
			return nil, errors.NewWithCodef(
				"readPatternFlag",
				errors.ErrInvalidInput,
				"'--pattern' value cannot be empty",
			)
		}
		out = append(out, p)
	}
	return out, nil
}

// readPatternOrExactID accepts either repeatable --pattern values or one
// literal positional skill ID. Glob syntax remains flag-only so shell
// expansion cannot rewrite a batch request before Cobra sees it.
func readPatternOrExactID(cmd *cobra.Command, args []string) ([]string, error) {
	patterns, err := readPatternFlag(cmd)
	if err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return patterns, nil
	}
	if len(args) > 1 {
		return nil, errors.NewWithCode("readPatternOrExactID", errors.ErrInvalidInput, "只接受一个位置参数 ID；批量匹配请使用 --pattern")
	}
	if len(patterns) > 0 {
		return nil, errors.NewWithCode("readPatternOrExactID", errors.ErrInvalidInput, "<id> 与 --pattern 不能同时使用")
	}
	if strings.ContainsAny(args[0], "*?[") {
		return nil, errors.NewWithCode("readPatternOrExactID", errors.ErrInvalidInput, "位置参数仅支持精确 ID；通配匹配请使用 --pattern '<glob>'")
	}
	return []string{args[0]}, nil
}

func validatePatternOrExactID(cmd *cobra.Command, args []string) error {
	_, err := readPatternOrExactID(cmd, args)
	return err
}

func filterSkillIDsByPatterns(patterns []string, skillIDs []string) ([]string, error) {
	matchers, err := compilePatterns(patterns)
	if err != nil {
		return nil, err
	}
	if len(matchers) == 0 {
		return nil, nil
	}

	seen := make(map[string]struct{}, len(skillIDs))
	var out []string
	for _, id := range skillIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		for _, matcher := range matchers {
			if matcher.Match(id) {
				seen[id] = struct{}{}
				out = append(out, id)
				break
			}
		}
	}
	sort.Strings(out)
	return out, nil
}

func stringSetFromSlice(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func resolveProjectSkillIDsByPatterns(patterns []string) ([]string, error) {
	if client, ok := hubClientIfAvailable(); ok {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, pkgutils.GetCwdErr(err)
		}
		data, err := client.GetProjectStatus(context.Background(), cwd, "")
		if err != nil {
			return nil, errors.Wrap(err, "通过服务读取项目技能状态失败")
		}
		if data.Item == nil {
			return nil, nil
		}
		ids := make([]string, 0, len(data.Item.Items))
		for _, item := range data.Item.Items {
			ids = append(ids, item.SkillID)
		}
		return filterSkillIDsByPatterns(patterns, ids)
	}

	ctx, err := RequireInitAndWorkspace("")
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(ctx.ProjectState.Skills))
	for id := range ctx.ProjectState.Skills {
		ids = append(ids, id)
	}
	return filterSkillIDsByPatterns(patterns, ids)
}

func resolveGlobalSkillIDsByPatterns(patterns []string, agents []string) ([]string, error) {
	if client, ok := hubClientIfAvailable(); ok {
		data, err := client.GetGlobalStatus(context.Background(), "", agents)
		if err != nil {
			return nil, errors.Wrap(err, "通过服务读取全局技能状态失败")
		}
		if data.Item == nil {
			return nil, nil
		}
		ids := make([]string, 0, len(data.Item.Items))
		for _, item := range data.Item.Items {
			ids = append(ids, item.SkillID)
		}
		return filterSkillIDsByPatterns(patterns, ids)
	}

	if err := CheckInitDependency(); err != nil {
		return nil, err
	}
	summary, err := localRuntime.Global().Inspect("", agents)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(summary.Items))
	for _, item := range summary.Items {
		ids = append(ids, item.SkillID)
	}
	return filterSkillIDsByPatterns(patterns, ids)
}
