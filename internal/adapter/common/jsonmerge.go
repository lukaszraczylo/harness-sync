package common

import "encoding/json"

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
