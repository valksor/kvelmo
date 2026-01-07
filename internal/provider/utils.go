package provider

// PersonNames extracts names from a Person slice.
// If a person has a name, that is used; otherwise falls back to their ID.
// Duplicate persons (by ID) are deduplicated in the result.
func PersonNames(persons []Person) []string {
	if len(persons) == 0 {
		return []string{}
	}

	// Deduplicate by ID while preserving order
	seen := make(map[string]bool, len(persons))
	var names []string
	for _, p := range persons {
		if !seen[p.ID] {
			seen[p.ID] = true
			if p.Name != "" {
				names = append(names, p.Name)
			} else {
				names = append(names, p.ID)
			}
		}
	}

	return names
}
