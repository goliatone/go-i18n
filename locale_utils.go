package i18n

import (
	"sort"
	"strings"

	"golang.org/x/text/language"
)

func localeParentTag(locale string) string {
	if locale == "" {
		return ""
	}

	tag, err := language.Parse(locale)
	if err == nil {
		parent := tag.Parent()
		if parent == language.Und {
			return ""
		}
		value := parent.String()
		if value == "" || value == "und" {
			return ""
		}
		return value
	}

	if idx := strings.LastIndex(locale, "-"); idx > 0 {
		return locale[:idx]
	}

	return ""
}

func localeParentChain(locale string) []string {
	if locale == "" {
		return nil
	}

	var chain []string
	seen := make(map[string]struct{}, 4)

	if tag, err := language.Parse(locale); err == nil {
		for parent := tag.Parent(); parent != language.Und; parent = parent.Parent() {
			parentValue := parent.String()
			if parentValue == "" || parentValue == "und" {
				break
			}
			if _, exists := seen[parentValue]; exists {
				break
			}
			seen[parentValue] = struct{}{}
			chain = append(chain, parentValue)
		}
	}

	for current := localeParentTag(locale); current != ""; current = localeParentTag(current) {
		if _, exists := seen[current]; exists {
			continue
		}
		seen[current] = struct{}{}
		chain = append(chain, current)
	}

	return chain
}

// normalizeLocale normalizes a single locale identifier by replacing
// underscores with hyphens and trimming whitespace.
func normalizeLocale(locale string) string {
	return strings.ReplaceAll(strings.TrimSpace(locale), "_", "-")
}

func normalizeLocales(locales []string) []string {
	if len(locales) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(locales))
	result := make([]string, 0, len(locales))
	for _, locale := range locales {
		normalized := normalizeLocale(locale)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}

	sort.Strings(result)
	return result
}

// deriveLocaleParents returns all parent locales for the given locale,
// ordered from closest parent to root. This is an alias for localeParentChain
// to maintain backward compatibility.
func deriveLocaleParents(locale string) []string {
	return localeParentChain(locale)
}
