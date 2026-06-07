package utils

import (
	"errors"
	"testing"

	pkgerrors "github.com/muidea/skill-hub/pkg/errors"
)

func assertInvalidInput(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var appErr *pkgerrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *pkgerrors.AppError, got %T", err)
	}
	if appErr.Code != pkgerrors.ErrInvalidInput {
		t.Errorf("expected ErrInvalidInput code, got %s", appErr.Code)
	}
}

func TestCompileSkillIDPattern_ValidPatterns(t *testing.T) {
	cases := []struct {
		name    string
		pattern string
		input   string
		want    bool
	}{
		{"empty matches all", "", "anything", true},
		{"double-star matches all", "**", "magic", true},
		{"exact match", "magic", "magic", true},
		{"exact no match", "magic", "wizard", false},
		{"prefix star", "magic*", "magic-skill", true},
		{"prefix star no match", "magic*", "wizard-skill", false},
		{"suffix star", "*-skill", "magic-skill", true},
		{"middle star", "magic*wizard", "magic-and-wizard", true},
		{"question mark single char", "magic?", "magic1", true},
		{"question mark no match length", "magic?", "magic", false},
		{"question mark no match extra", "magic?", "magic12", false},
		{"character class", "magic[ab]", "magica", true},
		{"character class miss", "magic[ab]", "magicc", false},
		{"negated class (Go uses ^)", "magic[^ab]", "magicc", true},
		{"negated class miss", "magic[^ab]", "magica", false},
		{"star crosses slash", "magic*", "magic-team/magic-skill", true},
		{"star crosses slash owner prefix", "magic-team/*", "magic-team/magic-skill", true},
		{"star still matches simple id", "magic*", "magic-skill", true},
		{"utf8 exact", "数据技能", "数据技能", true},
		{"utf8 question mark", "数据?", "数据集", true},
		{"utf8 star", "数据*", "数据处理", true},
		{"utf8 character class", "数据[处理]", "数据处", true},
		{"utf8 character range", "技能[α-ω]", "技能β", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m, err := CompileSkillIDPattern(c.pattern)
			if err != nil {
				t.Fatalf("CompileSkillIDPattern(%q) returned error: %v", c.pattern, err)
			}
			got := m.Match(c.input)
			if got != c.want {
				t.Errorf("Match(%q) with pattern %q = %v, want %v", c.input, c.pattern, got, c.want)
			}
		})
	}
}

func TestCompileSkillIDPattern_RejectsLoneStar(t *testing.T) {
	_, err := CompileSkillIDPattern("*")
	assertInvalidInput(t, err)
}

func TestCompileSkillIDPattern_RejectsMalformedBrackets(t *testing.T) {
	// Bare `]` and `[!]` are accepted as literal characters, so they are NOT
	// in the rejection list below.
	bad := []string{"[", "[a-"}
	for _, p := range bad {
		t.Run(p, func(t *testing.T) {
			_, err := CompileSkillIDPattern(p)
			assertInvalidInput(t, err)
		})
	}
}

func TestMatcher_MatchAfterCompile(t *testing.T) {
	m, err := CompileSkillIDPattern("magic*")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !m.Match("magic-skill") {
		t.Error("expected magic-skill to match magic*")
	}
	if m.Match("wizard") {
		t.Error("expected wizard to not match magic*")
	}
}

func TestMatcher_AllMatchesAll(t *testing.T) {
	for _, p := range []string{"", "**"} {
		m, err := CompileSkillIDPattern(p)
		if err != nil {
			t.Fatalf("compile %q: %v", p, err)
		}
		for _, in := range []string{"", "x", "magic-skill", "anything-goes-here"} {
			if !m.Match(in) {
				t.Errorf("all-matcher for %q should match %q", p, in)
			}
		}
	}
}
