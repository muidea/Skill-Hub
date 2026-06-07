package utils

import (
	"strings"
	"unicode/utf8"

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
	if si > len(s) {
		return false
	}
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
				_, size := utf8.DecodeRuneInString(s[si:])
				si += size
			}
			return false
		case '[':
			if si >= len(s) {
				return false
			}
			r, size := utf8.DecodeRuneInString(s[si:])
			negate := false
			pi++
			if pi < len(pat) && pat[pi] == '^' {
				negate = true
				pi++
			}
			matched := false
			for pi < len(pat) && pat[pi] != ']' {
				start, startSize := utf8.DecodeRuneInString(pat[pi:])
				if next := pi + startSize; next < len(pat) && pat[next] == '-' {
					endIdx := next + 1
					if endIdx < len(pat) && pat[endIdx] != ']' {
						end, endSize := utf8.DecodeRuneInString(pat[endIdx:])
						if r >= start && r <= end {
							matched = true
						}
						pi = endIdx + endSize
						continue
					}
				}
				if start == r {
					matched = true
				}
				pi += startSize
			}
			if pi >= len(pat) {
				return false
			}
			if matched == negate {
				return false
			}
			pi++
			si += size
		case '?':
			if si >= len(s) {
				return false
			}
			_, size := utf8.DecodeRuneInString(s[si:])
			pi++
			si += size
		default:
			if si >= len(s) {
				return false
			}
			r, size := utf8.DecodeRuneInString(s[si:])
			pr, pSize := utf8.DecodeRuneInString(pat[pi:])
			if pr != r {
				return false
			}
			pi += pSize
			si += size
		}
	}
	return si == len(s)
}
