// Package glob matches slash-separated paths against patterns where "**"
// matches any number of path segments and every other segment follows
// path.Match.
package glob

import (
	"path"
	"strings"
)

// Match reports whether name matches pattern. Both are slash-separated and
// relative. A malformed segment never matches.
func Match(pattern, name string) bool {
	if pattern == "**" {
		return true
	}
	return matchSegments(strings.Split(pattern, "/"), strings.Split(name, "/"))
}

// MatchAny reports whether name matches at least one of patterns.
func MatchAny(patterns []string, name string) bool {
	for _, p := range patterns {
		if Match(p, name) {
			return true
		}
	}
	return false
}

// matchSegments matches pattern segments against name segments, letting "**"
// absorb zero or more name segments.
func matchSegments(pat, name []string) bool {
	for len(pat) > 0 {
		if pat[0] == "**" {
			rest := pat[1:]
			for len(rest) > 0 && rest[0] == "**" {
				rest = rest[1:]
			}
			if len(rest) == 0 {
				return true
			}
			for i := range len(name) + 1 {
				if matchSegments(rest, name[i:]) {
					return true
				}
			}
			return false
		}
		if len(name) == 0 {
			return false
		}
		ok, err := path.Match(pat[0], name[0])
		if err != nil || !ok {
			return false
		}
		pat, name = pat[1:], name[1:]
	}
	return len(name) == 0
}
