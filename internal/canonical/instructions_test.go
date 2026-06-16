package canonical

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstructionText(t *testing.T) {
	t.Run("global when no override", func(t *testing.T) {
		t.Parallel()
		b := Bundle{Instructions: Instructions{Global: "G"}}
		assert.Equal(t, "G", b.InstructionText("zed"))
	})
	t.Run("per-harness override wins", func(t *testing.T) {
		t.Parallel()
		b := Bundle{Instructions: Instructions{Global: "G", PerHarness: map[string]string{"zed": "Z"}}}
		assert.Equal(t, "Z", b.InstructionText("zed"))
	})
	t.Run("empty override falls back to global", func(t *testing.T) {
		t.Parallel()
		b := Bundle{Instructions: Instructions{Global: "G", PerHarness: map[string]string{"zed": ""}}}
		assert.Equal(t, "G", b.InstructionText("zed"))
	})
	t.Run("empty everything", func(t *testing.T) {
		t.Parallel()
		b := Bundle{}
		assert.Equal(t, "", b.InstructionText("zed"))
	})
}

func TestInstructionTextWithRules(t *testing.T) {
	t.Run("folds rules after global", func(t *testing.T) {
		t.Parallel()
		b := Bundle{
			Instructions: Instructions{Global: "# Global"},
			Rules: []Rule{
				{Name: "a", Body: "# Rule A\n\nbe concise"},
				{Name: "b", Body: "# Rule B\n\nbe safe"},
			},
		}
		got := b.InstructionTextWithRules("opencode")
		assert.Equal(t, "# Global\n\n# Rule A\n\nbe concise\n\n# Rule B\n\nbe safe", got)
	})

	t.Run("strips rule frontmatter", func(t *testing.T) {
		t.Parallel()
		b := Bundle{
			Instructions: Instructions{Global: "G"},
			Rules: []Rule{
				{Name: "scoped", Body: "---\nname: scoped\npaths: \"**/*.go\"\n---\n# Go\n\ngofmt"},
			},
		}
		got := b.InstructionTextWithRules("opencode")
		assert.Equal(t, "G\n\n# Go\n\ngofmt", got)
		assert.NotContains(t, got, "paths:")
		assert.NotContains(t, got, "---")
	})

	t.Run("no rules returns plain instruction text", func(t *testing.T) {
		t.Parallel()
		b := Bundle{Instructions: Instructions{Global: "G"}}
		assert.Equal(t, "G", b.InstructionTextWithRules("opencode"))
	})

	t.Run("empty global still emits rules", func(t *testing.T) {
		t.Parallel()
		b := Bundle{Rules: []Rule{{Name: "a", Body: "rule body"}}}
		assert.Equal(t, "rule body", b.InstructionTextWithRules("opencode"))
	})

	t.Run("blank rule bodies are skipped", func(t *testing.T) {
		t.Parallel()
		b := Bundle{
			Instructions: Instructions{Global: "G"},
			Rules: []Rule{
				{Name: "empty", Body: "   \n  "},
				{Name: "real", Body: "keep"},
			},
		}
		assert.Equal(t, "G\n\nkeep", b.InstructionTextWithRules("opencode"))
	})
}

func TestStripFrontmatter(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"no frontmatter", "# Title\nbody", "# Title\nbody"},
		{"with frontmatter", "---\nname: x\n---\n# Title", "# Title"},
		{"unterminated frontmatter left intact", "---\nname: x\nbody", "---\nname: x\nbody"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, stripFrontmatter(tt.in))
		})
	}
}
