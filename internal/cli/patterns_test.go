package cli

import (
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
