// Package secrets provides ${VAR} substitution for canonical configs.
package secrets

import (
	"fmt"
	"os"
	"strings"
)

// Lookup resolves a placeholder name to a value. Returns false if missing.
type Lookup func(name string) (string, bool)

// OSEnv resolves via os.LookupEnv. Returns (value, true) when the variable
// is set, otherwise (placeholder, true) so the literal ${NAME} reaches the
// downstream config unchanged.
//
// Rationale: many harness MCP configs reference variables that the harness
// itself injects at MCP launch time (e.g. ${CLAUDE_PROJECT} for claude-code,
// ${GITHUB_TOKEN} when the user keeps it out of their login shell). Aborting
// the apply for an unset variable would block valid configurations, so unset
// names are passed through. Real secrets the user wants resolved at apply
// time must be exported in the process environment.
func OSEnv(name string) (string, bool) {
	if v, ok := os.LookupEnv(name); ok {
		return v, true
	}
	return "${" + name + "}", true
}

// StrictOSEnv resolves via os.LookupEnv and reports false for unset
// variables, causing Substitute to return an error. Use when an empty
// resolution is unacceptable (e.g. dedicated secrets pipeline).
func StrictOSEnv(name string) (string, bool) {
	return os.LookupEnv(name)
}

// Substitute replaces ${NAME} placeholders in s using lookup. Escape with $$.
// Missing keys produce an error so secrets are never silently empty.
func Substitute(s string, lookup Lookup) (string, error) {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if i+1 < len(s) && s[i] == '$' && s[i+1] == '$' {
			b.WriteByte('$')
			i += 2
			continue
		}
		if i+1 < len(s) && s[i] == '$' && s[i+1] == '{' {
			end := strings.IndexByte(s[i+2:], '}')
			if end == -1 {
				return "", fmt.Errorf("unterminated placeholder at offset %d", i)
			}
			name := s[i+2 : i+2+end]
			val, ok := lookup(name)
			if !ok {
				return "", fmt.Errorf("missing env var %q", name)
			}
			b.WriteString(val)
			i += 2 + end + 1
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String(), nil
}
