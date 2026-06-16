package canonical

import "strings"

// InstructionText returns the instruction body a harness should receive: the
// per-harness override when one is set, otherwise the global instructions.
func (b *Bundle) InstructionText(harness string) string {
	if override, ok := b.Instructions.PerHarness[harness]; ok && override != "" {
		return override
	}
	return b.Instructions.Global
}

// InstructionTextWithRules returns InstructionText for the harness with every
// canonical rule body folded in. Harnesses without a native rules directory
// (everything except claude-code/kilo) only receive rule content this way —
// via their global instructions file. Rule frontmatter is stripped so the
// folded output stays clean; bodies are joined in load order (deterministic).
func (b *Bundle) InstructionTextWithRules(harness string) string {
	base := b.InstructionText(harness)
	var sb strings.Builder
	sb.WriteString(base)
	for _, r := range b.Rules {
		body := strings.TrimSpace(stripFrontmatter(r.Body))
		if body == "" {
			continue
		}
		if sb.Len() > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(body)
	}
	return sb.String()
}

// stripFrontmatter removes a leading YAML frontmatter block ("---\n…\n---") so
// folding a rule into a plain instructions file does not leak its metadata.
func stripFrontmatter(s string) string {
	if !strings.HasPrefix(s, "---\n") {
		return s
	}
	end := strings.Index(s[4:], "\n---")
	if end == -1 {
		return s
	}
	return strings.TrimPrefix(s[4+end+len("\n---"):], "\n")
}
