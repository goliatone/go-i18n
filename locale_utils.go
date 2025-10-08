package i18n

import "sort"

// normalizeLocales removes duplicates and empty strings from a locale list,
// then sorts the result alphabetically.
func normalizeLocales(locales []string) []string {
	if len(locales) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(locales))
	result := make([]string, 0, len(locales))
	for _, locale := range locales {
		if locale == "" {
			continue
		}
		if _, exists := seen[locale]; exists {
			continue
		}
		seen[locale] = struct{}{}
		result = append(result, locale)
	}

	sort.Strings(result)
	return result
}
