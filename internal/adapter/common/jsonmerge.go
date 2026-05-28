package common

import (
	"encoding/json"
	"maps"
)

// MergeJSONKeys reads existing JSON/JSONC body, overlays the given top-level
// keys (replacing values for listed keys, preserving every other key), and
// returns the merged JSON. If existing is empty or invalid JSON the function
// returns just the overlay encoded fresh (2-space indent + trailing \n).
// An overlay value of nil deletes the key from the base.
func MergeJSONKeys(existing []byte, overlay map[string]any) ([]byte, error) {
	base := map[string]any{}
	if len(existing) > 0 {
		// Try plain JSON first (avoids mangling URL strings that contain //).
		// Only fall back to comment-stripping for JSONC content.
		if err := json.Unmarshal(existing, &base); err != nil {
			clean := StripJSONComments(string(existing))
			if err2 := json.Unmarshal([]byte(clean), &base); err2 != nil {
				base = map[string]any{}
			}
		}
	}
	for k, v := range overlay {
		if v == nil {
			delete(base, k)
			continue
		}
		base[k] = v
	}
	out, err := json.MarshalIndent(base, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

// UnionNestedMap parses existing JSON/JSONC, reads the object nested under key,
// and overlays ours on top (ours wins per entry name). Entries the user added
// under key that are not in ours are PRESERVED. This is the map-valued analogue
// of the manual provider-union the openai-compatible adapters do, so that an
// apply never silently drops a server/entry the user added directly to the
// target tool's config (e.g. via `claude mcp add`).
func UnionNestedMap(existing []byte, key string, ours map[string]any) map[string]any {
	merged := map[string]any{}
	if len(existing) > 0 {
		base := map[string]any{}
		if json.Unmarshal(existing, &base) != nil {
			_ = json.Unmarshal([]byte(StripJSONComments(string(existing))), &base)
		}
		if existingNested, ok := base[key].(map[string]any); ok {
			maps.Copy(merged, existingNested)
		}
	}
	maps.Copy(merged, ours)
	return merged
}
