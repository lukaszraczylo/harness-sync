package common

import (
	"maps"

	"gopkg.in/yaml.v3"
)

// MergeYAMLKeys reads existing YAML body, overlays the given top-level keys
// (replacing values for listed keys, preserving every other key), and returns
// the merged YAML. If existing is empty or invalid YAML, the function returns
// just the overlay encoded fresh (with 2-space indent + trailing \n).
// An overlay value of nil deletes the key from the base.
func MergeYAMLKeys(existing []byte, overlay map[string]any) ([]byte, error) {
	base := map[string]any{}
	if len(existing) > 0 {
		if err := yaml.Unmarshal(existing, &base); err != nil {
			base = map[string]any{}
		}
	}
	for k, v := range overlay {
		if v == nil {
			delete(base, k)
			continue
		}
		base[k] = v
	}
	out, err := yaml.Marshal(base)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// UnionNestedMapYAML parses existing YAML, reads the map nested under key, and
// overlays ours on top (ours wins per entry name). Entries the user added under
// key that are not in ours are PRESERVED — the YAML analogue of UnionNestedMap,
// used for goose `extensions` so an apply never drops a user-added extension.
func UnionNestedMapYAML(existing []byte, key string, ours map[string]any) map[string]any {
	merged := map[string]any{}
	if len(existing) > 0 {
		base := map[string]any{}
		if yaml.Unmarshal(existing, &base) == nil {
			if existingNested, ok := base[key].(map[string]any); ok {
				maps.Copy(merged, existingNested)
			}
		}
	}
	maps.Copy(merged, ours)
	return merged
}
