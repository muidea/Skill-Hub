package utils

import (
	"strings"

	"github.com/muidea/skill-hub/pkg/errors"
)

// Matcher is a compiled glob pattern used to match skill IDs.
type Matcher struct {
	all     bool
	pattern string
}

// CompileSkillIDPattern compiles a glob pattern for matching skill IDs.
//
// Supported wildcards:
//   - `*`  matches zero or more arbitrary characters (including `/`)
//   - `?`  matches exactly one arbitrary character
//   - `[abc]`  character class; `[a-z]` ranges are supported
//   - `[^abc]`  negated character class (Go convention; shell uses `[!...]`)
//   - `**`  only as a stand-alone token meaning "match every id"
//
// An empty pattern matches every id. A bare `*` is rejected as ambiguous;
// callers should use `**` to mean "match all".
func CompileSkillIDPattern(p string) (Matcher, error) {
	if p == "" || p == "**" {
		return Matcher{all: true}, nil
	}
	if p == "*" {
		return Matcher{}, errors.NewWithCodef(
			"CompileSkillIDPattern",
			errors.ErrInvalidInput,
			"pattern '%s' is ambiguous; use '**' to match every skill",
			p,
		)
	}
	if err := validatePattern(p); err != nil {
		return Matcher{}, err
	}
	return Matcher{pattern: p}, nil
}

// Match reports whether s matches the compiled pattern. `*` matches `/` so
// that path-shaped IDs (e.g. "magic-team/magic-skill") are matched naturally
// by owner-stripped patterns like "magic*".
func (m Matcher) Match(s string) bool {
	if m.all {
		return true
	}
	return matchGlob(m.pattern, s)
}

// IsLiteral reports whether the compiled pattern matches a single literal
// string (no wildcards, not the "**" all-match sentinel). Callers can use
// this to short-circuit pattern resolution: a literal pattern is a
// direct-ID lookup, so the resolved set is just {pattern}.
func (m Matcher) IsLiteral() bool {
	if m.all {
		return false
	}
	return !strings.ContainsAny(m.pattern, "*?[")
}

func validatePattern(p string) error {
	// `[` opens a character class which must terminate with `]`. Bare `[`
	// or `[a-` is a malformed pattern.
	for i := 0; i < len(p); i++ {
		if p[i] != '[' {
			continue
		}
		j := i + 1
		if j == len(p) {
			return errors.NewWithCodef(
				"CompileSkillIDPattern",
				errors.ErrInvalidInput,
				"无效的pattern %q",
				p,
			)
		}
		if p[j] == '^' {
			j++
		}
		if j == len(p) {
			return errors.NewWithCodef(
				"CompileSkillIDPattern",
				errors.ErrInvalidInput,
				"无效的pattern %q",
				p,
			)
		}
		closeIdx := strings.IndexByte(p[j:], ']')
		if closeIdx < 0 {
			return errors.NewWithCodef(
				"CompileSkillIDPattern",
				errors.ErrInvalidInput,
				"无效的pattern %q",
				p,
			)
		}
		i = j + closeIdx
	}
	return nil
}

// matchGlob is a small backtracking glob matcher that mirrors the user-facing
// semantics of Go's path.Match but allows `*` to span `/`. It is used only
// after the empty / "**" / "match all" short-circuits have been considered.
func matchGlob(pattern, s string) bool {
	return matchAt(pattern, 0, s, 0)
}

func matchAt(pat string, pi int, s string, si int) bool {
	for pi < len(pat) {
		switch pat[pi] {
		case '*':
			// Collapse runs of `*`.
			for pi < len(pat) && pat[pi] == '*' {
				pi++
			}
			if pi == len(pat) {
				return true
			}
			// Try to match the rest of the pattern starting at every
			// position in s (including past the end for zero-match).
			for si <= len(s) {
				if matchAt(pat, pi, s, si) {
					return true
				}
				if si == len(s) {
					return false
				}
				// Advance one rune to handle multi-byte UTF-8 safely.
				si += len(string(s[si]))
			}
			return false
		case '[':
			if si >= len(s) {
				return false
			}
			negate := false
			pi++
			if pi < len(pat) && pat[pi] == '^' {
				negate = true
				pi++
			}
			cls := s[si]
			matched := false
			for pi < len(pat) && pat[pi] != ']' {
				if pi+2 < len(pat) && pat[pi+1] == '-' {
					if cls >= pat[pi] && cls <= pat[pi+2] {
						matched = true
					}
					pi += 3
					continue
				}
				if pat[pi] == cls {
					matched = true
				}
				pi++
			}
			if pi >= len(pat) {
				return false
			}
			if matched == negate {
				return false
			}
			pi++
			si += len(string(s[si]))
		case '?':
			if si >= len(s) {
				return false
			}
			pi++
			si += len(string(s[si]))
		default:
			if si >= len(s) || pat[pi] != s[si] {
				return false
			}
			pi++
			si++
		}
	}
	return si == len(s)
}
