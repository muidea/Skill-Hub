package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/muidea/skill-hub/pkg/errors"
)

// rejectPositionalPattern is a cobra Args validator used by every
// pattern-aware command. Since the v0.8.13 flag migration, glob patterns are
// accepted only via --pattern; any positional arg is a leftover from the
// pre-flag shape and is rejected with a clear pointer to the new flag.
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
	hint := "positional arguments are no longer accepted; pass the pattern via --pattern"
	if looksLikeShellExpansion(args) {
		hint += " (your shell expanded the glob before skill-hub saw it — quote the pattern, e.g. --pattern 'magic*')"
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
// validation across list/use/status/apply/feedback/validate.
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
