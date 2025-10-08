package i18n

import (
	"strings"

	"golang.org/x/text/language"
)

// localeParentTag returns the CLDR parent for the given locale.
// Falls back to simple tag truncation if parsing fails.
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

	// Fallback to manual truncation when parsing fails.
	if idx := strings.LastIndex(locale, "-"); idx > 0 {
		return locale[:idx]
	}
	return ""
}

// localeParentChain returns all parents for the locale, ordered from closest parent to root.
func localeParentChain(locale string) []string {
	if locale == "" {
		return nil
	}

	var chain []string
	seen := make(map[string]struct{}, 4)

	tag, err := language.Parse(locale)
	if err == nil {
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

	// Ensure canonical hyphen-based parents are also considered.
	for current := localeParentTag(locale); current != ""; current = localeParentTag(current) {
		if _, exists := seen[current]; exists {
			continue
		}
		seen[current] = struct{}{}
		chain = append(chain, current)
	}

	return chain
}
