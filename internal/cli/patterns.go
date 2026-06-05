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
func rejectPositionalPattern(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	return errors.NewWithCodef(
		"rejectPositionalPattern",
		errors.ErrInvalidInput,
		"positional arguments are no longer accepted; use --pattern '%s' instead",
		strings.Join(args, " "),
	)
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
