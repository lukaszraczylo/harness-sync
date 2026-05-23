package goose

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

func importFrom(home string) (*adapter.ImportResult, error) {
	cfgPath := filepath.Join(home, ".config", "goose", "config.yaml")
	body, err := os.ReadFile(cfgPath)
	if os.IsNotExist(err) {
		return &adapter.ImportResult{}, nil
	}
	if err != nil {
		return nil, err
	}

	var raw map[string]any
	if err := yaml.Unmarshal(body, &raw); err != nil {
		return &adapter.ImportResult{}, nil
	}

	res := &adapter.ImportResult{}

	// Reconstruct MCP servers from extensions entries with type: stdio.
	if exts, ok := raw["extensions"]; ok {
		if extMap, ok := exts.(map[string]any); ok {
			for name, v := range extMap {
				entry, ok := v.(map[string]any)
				if !ok {
					continue
				}
				t, _ := entry["type"].(string)
				if t != "stdio" {
					continue
				}
				s := canonical.MCPServer{Name: name}
				if cmd, ok := entry["cmd"].(string); ok {
					s.Command = cmd
				}
				if args, ok := entry["args"].([]any); ok {
					for _, a := range args {
						if str, ok := a.(string); ok {
							s.Args = append(s.Args, str)
						}
					}
				}
				if envRaw, ok := entry["env"].(map[string]any); ok {
					s.Env = make(map[string]string, len(envRaw))
					for k, val := range envRaw {
						if str, ok := val.(string); ok {
							s.Env[k] = str
						}
					}
				}
				res.MCP = append(res.MCP, s)
			}
		}
	}

	return res, nil
}
