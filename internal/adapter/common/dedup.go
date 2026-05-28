package common

import (
	"encoding/json"
	"maps"
)

// MergeProviderMap parses the existing JSON/JSONC config, unions our provider
// entries over the user's existing `provider` map (ours win per key), then
// absorbs any duplicate providers that point at the same gateway URL. This is
// the canonical "preserve user providers + dedup gateway" routine used by the
// opencode-shaped adapters (opencode, kilo) so a user's hand-added providers
// survive an apply instead of being wholesale-replaced.
func MergeProviderMap(existing []byte, ours map[string]any, gatewayURL string) map[string]any {
	merged := map[string]any{}
	if len(existing) > 0 {
		base := map[string]any{}
		if json.Unmarshal(existing, &base) != nil {
			_ = json.Unmarshal([]byte(StripJSONComments(string(existing))), &base)
		}
		if existingProv, ok := base["provider"].(map[string]any); ok {
			maps.Copy(merged, existingProv)
		}
	}
	maps.Copy(merged, ours) // ours win over a same-keyed user entry
	return AbsorbDuplicateProviders(merged, GatewayProviderKey(gatewayURL), gatewayURL)
}

// AbsorbDuplicateProviders scans provMap for entries (other than ourKey) whose
// options.baseURL matches gatewayURL and absorbs their models into our entry.
//
// Merge strategy:
//   - Models only in the duplicate: added to our provider.
//   - Models only in our canonical entry: kept as-is.
//   - Models in both: the DUPLICATE's entry wins (preserves user-crafted display
//     names and custom limits that are often more descriptive than the profile
//     alias).
//
// Absorbed entries are deleted from the returned map. Returns provMap unchanged
// when gatewayURL is empty or our entry is absent.
func AbsorbDuplicateProviders(provMap map[string]any, ourKey, gatewayURL string) map[string]any {
	if gatewayURL == "" || provMap == nil {
		return provMap
	}
	ourEntry, ok := provMap[ourKey].(map[string]any)
	if !ok {
		return provMap
	}

	// Clone top-level so we don't mutate the caller's map.
	result := make(map[string]any, len(provMap))
	maps.Copy(result, provMap)

	ourModels, _ := ourEntry["models"].(map[string]any)
	if ourModels == nil {
		ourModels = map[string]any{}
	}

	// merged starts with our canonical models; duplicates may override per-model.
	merged := make(map[string]any, len(ourModels))
	maps.Copy(merged, ourModels)

	// updated is our entry; we also absorb the duplicate's non-models keys into
	// it (when absent) so user-crafted options/headers aren't silently lost.
	updated := make(map[string]any, len(ourEntry))
	maps.Copy(updated, ourEntry)

	for k, v := range result {
		if k == ourKey {
			continue
		}
		entry, ok := v.(map[string]any)
		if !ok {
			continue
		}
		opts, _ := entry["options"].(map[string]any)
		if opts == nil {
			continue
		}
		if opts["baseURL"] != gatewayURL {
			continue
		}
		// A non-map (e.g. array-shaped) models value can't be merged into our
		// map; keep the duplicate intact rather than silently dropping it.
		if dm, present := entry["models"]; present {
			if _, isMap := dm.(map[string]any); !isMap {
				continue
			}
		}
		// Same gateway URL with absorbable models — absorb and delete.
		if dupModels, ok := entry["models"].(map[string]any); ok {
			// Duplicate (user-crafted) wins for overlapping IDs.
			maps.Copy(merged, dupModels)
		}
		// Preserve the duplicate's other keys we don't already have.
		for ek, ev := range entry {
			if ek == "models" {
				continue
			}
			if _, exists := updated[ek]; !exists {
				updated[ek] = ev
			}
		}
		delete(result, k)
	}

	if len(merged) > 0 {
		updated["models"] = merged
	}
	result[ourKey] = updated
	return result
}
