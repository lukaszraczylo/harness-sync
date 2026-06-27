package pi

import (
	"os"
	"path/filepath"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
)

func importFrom(home string) (*adapter.ImportResult, error) {
	base := filepath.Join(home, ".pi", "agent")
	res := &adapter.ImportResult{}

	if body, err := os.ReadFile(filepath.Join(base, "AGENTS.md")); err == nil {
		res.Instructions = string(body)
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	return res, nil
}
