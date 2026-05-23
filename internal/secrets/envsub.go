// Package secrets provides ${VAR} substitution for canonical configs.
package secrets

import (
	"fmt"
	"os"
	"strings"
)

// Lookup resolves a placeholder name to a value. Returns false if missing.
type Lookup func(name string) (string, bool)

// OSEnv resolves via os.LookupEnv.
func OSEnv(name string) (string, bool) {
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
