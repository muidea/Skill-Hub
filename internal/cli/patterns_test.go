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
		{"shell-expanded magic glob", []string{"magicCas", "magicCommon", "magicEngine"}, "quote the pattern"},
		{"shell-expanded single-prefix glob", []string{"foo", "foobar"}, "quote the pattern"},
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
