package common

import "gopkg.in/yaml.v3"

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
