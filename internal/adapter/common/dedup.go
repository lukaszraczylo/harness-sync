package common

import "maps"

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
		// Same gateway URL — absorb and delete.
		if dupModels, ok := entry["models"].(map[string]any); ok {
			// Duplicate (user-crafted) wins for overlapping IDs.
			maps.Copy(merged, dupModels)
		}
		delete(result, k)
	}

	updated := make(map[string]any, len(ourEntry))
	maps.Copy(updated, ourEntry)
	if len(merged) > 0 {
		updated["models"] = merged
	}
	result[ourKey] = updated
	return result
}
