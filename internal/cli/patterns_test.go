package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func newTestCmdWithPatternFlag(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().StringArray("pattern", nil, "test pattern flag")
	return cmd
}

func TestReadPatternFlag_NotSetReturnsNil(t *testing.T) {
	cmd := newTestCmdWithPatternFlag(t)
	got, err := readPatternFlag(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil slice, got %#v", got)
	}
}

func TestReadPatternFlag_PassesValuesThrough(t *testing.T) {
	cmd := newTestCmdWithPatternFlag(t)
	if err := cmd.Flags().Set("pattern", "magic*"); err != nil {
		t.Fatalf("set pattern: %v", err)
	}
	if err := cmd.Flags().Set("pattern", "git-*"); err != nil {
		t.Fatalf("set pattern: %v", err)
	}
	got, err := readPatternFlag(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"magic*", "git-*"}
	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("got[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func TestReadPatternFlag_RejectsEmptyElement(t *testing.T) {
	cmd := newTestCmdWithPatternFlag(t)
	if err := cmd.Flags().Set("pattern", "magic*"); err != nil {
		t.Fatalf("set pattern: %v", err)
	}
	// Setting the same flag to an empty string appends "" to the slice
	// (cobra's StringArray behavior). The helper must reject it.
	if err := cmd.Flags().Set("pattern", ""); err != nil {
		t.Fatalf("set pattern: %v", err)
	}
	got, err := readPatternFlag(cmd)
	if err == nil {
		t.Fatalf("expected error for empty element, got slice %#v", got)
	}
	if got != nil {
		t.Errorf("expected nil slice on error, got %#v", got)
	}
}

func TestReadPatternOrExactID(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		patterns  []string
		want      []string
		wantError bool
	}{
		{name: "no selector", want: nil},
		{name: "exact positional id", args: []string{"demo-skill"}, want: []string{"demo-skill"}},
		{name: "pattern flag", patterns: []string{"demo-*"}, want: []string{"demo-*"}},
		{name: "positional glob rejected", args: []string{"demo-*"}, wantError: true},
		{name: "multiple positional ids rejected", args: []string{"one", "two"}, wantError: true},
		{name: "id and pattern rejected", args: []string{"demo"}, patterns: []string{"demo-*"}, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newTestCmdWithPatternFlag(t)
			for _, pattern := range tt.patterns {
				if err := cmd.Flags().Set("pattern", pattern); err != nil {
					t.Fatalf("set pattern: %v", err)
				}
			}
			got, err := readPatternOrExactID(cmd, tt.args)
			if (err != nil) != tt.wantError {
				t.Fatalf("readPatternOrExactID() error = %v, wantError %v", err, tt.wantError)
			}
			if tt.wantError {
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("readPatternOrExactID() = %#v, want %#v", got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("readPatternOrExactID()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestRejectPositionalPattern(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cases := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"no args", nil, false},
		{"empty args", []string{}, false},
		{"single arg", []string{"magic*"}, true},
		{"multiple args", []string{"a", "b"}, true},
		{"literal arg", []string{"foo"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := rejectPositionalPattern(cmd, c.args)
			if c.wantErr && err == nil {
				t.Errorf("expected error for args=%v, got nil", c.args)
			}
			if !c.wantErr && err != nil {
				t.Errorf("expected no error for args=%v, got %v", c.args, err)
			}
		})
	}
}

func TestRejectPositionalPattern_ShellExpansionHint(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cases := []struct {
		name        string
		args        []string
		wantHintSub string
	}{
		// The shell-expansion footgun: typing `list --pattern magic*` in a cwd
		// with matching files. Args are the shell-expanded filenames.
		{"shell-expanded magic glob", []string{"magicCas", "magicCommon", "magicEngine"}, "请引用 pattern"},
		{"shell-expanded single-prefix glob", []string{"foo", "foobar"}, "请引用 pattern"},
		// Non-shell cases: args don't share a 2-char prefix, or contain
		// path separators. Don't claim shell expansion in the message.
		{"unrelated multiple args", []string{"a", "b"}, "--pattern"},
		{"path args not shell expansion", []string{"./foo", "./bar"}, "--pattern"},
		{"single literal", []string{"foo"}, "--pattern"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := rejectPositionalPattern(cmd, c.args)
			if err == nil {
				t.Fatalf("expected error for args=%v", c.args)
			}
			if !strings.Contains(err.Error(), c.wantHintSub) {
				t.Errorf("error %q must contain %q", err.Error(), c.wantHintSub)
			}
			// The misleading "use --pattern 'magicCas magicCommon...'" form
			// must never appear — we don't echo the shell-expanded args back
			// as a pattern suggestion.
			if strings.Contains(err.Error(), "use --pattern '") {
				t.Errorf("error %q must not echo args as a suggested pattern", err.Error())
			}
		})
	}
}

func TestLongestCommonPrefix(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want string
	}{
		{"empty", nil, ""},
		{"single", []string{"abc"}, "abc"},
		{"common", []string{"magicCas", "magicCommon", "magicEngine"}, "magic"},
		{"divergent", []string{"foo", "bar"}, ""},
		{"nested", []string{"foo/bar", "foo/baz"}, "foo/ba"},
		{"empty element", []string{"", "foo"}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := longestCommonPrefix(c.in)
			if got != c.want {
				t.Errorf("longestCommonPrefix(%v) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestLooksLikeShellExpansion(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want bool
	}{
		{"empty", nil, false},
		{"single", []string{"magic*"}, false},
		{"two shared prefix", []string{"magicFoo", "magicBar"}, true},
		{"three shared prefix", []string{"magicCas", "magicCommon", "magicEngine"}, true},
		{"unrelated", []string{"foo", "bar"}, false},
		{"path separator", []string{"./foo", "./bar"}, false},
		{"space in arg", []string{"foo bar", "foo baz"}, false},
		{"single char prefix", []string{"a1", "a2"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := looksLikeShellExpansion(c.args)
			if got != c.want {
				t.Errorf("looksLikeShellExpansion(%v) = %v, want %v", c.args, got, c.want)
			}
		})
	}
}

func TestRejectPositionalPatternSilencesUsage(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	err := rejectPositionalPattern(cmd, []string{"magic*"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !cmd.SilenceUsage {
		t.Fatalf("expected positional pattern errors to silence cobra usage")
	}
	if !strings.Contains(err.Error(), "test --pattern 'magic*'") {
		t.Fatalf("expected replacement command in error, got %q", err.Error())
	}
}

func TestFilterSkillIDsByPatterns(t *testing.T) {
	ids := []string{"demo-skill", "magic-pack", "magic-pack", "magic-community/magic-tool", "other"}
	got, err := filterSkillIDsByPatterns([]string{"magic*"}, ids)
	if err != nil {
		t.Fatalf("filterSkillIDsByPatterns: %v", err)
	}
	want := []string{"magic-community/magic-tool", "magic-pack"}
	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q (all got: %v)", i, got[i], want[i], got)
		}
	}
}

func TestFilterSkillIDsByPatternsRejectsBareStar(t *testing.T) {
	_, err := filterSkillIDsByPatterns([]string{"*"}, []string{"demo-skill"})
	if err == nil {
		t.Fatalf("expected error for bare star")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("unexpected error: %v", err)
	}
}
